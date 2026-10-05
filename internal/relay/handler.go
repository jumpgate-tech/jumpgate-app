package relay

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/valve-tech/jumpgate/internal/wsrpc"
)

// maxBodyBytes caps a request body the relay must read to apply method policy.
// A JSON-RPC batch is small; anything larger is either a mistake or an attempt
// to make the relay hold megabytes per connection.
const maxBodyBytes = 4 << 20

// Config wires the relay.
type Config struct {
	// Auth resolves a raw key. In production this is a KeyCache wrapping a
	// BillingClient.
	Auth Authenticator
	// ProjectID is eRPC's project segment. It is the first segment of the
	// forwarded path.
	ProjectID string
	// ERPC is the base URL of the keyless eRPC, reached over loopback.
	ERPC *url.URL
	// Beacon resolves a chain's beacon client. It reports false when the chain
	// has no consensus layer, which the relay answers as a definite 501 rather
	// than a dead 502.
	Beacon func(chainID int) (*url.URL, bool)
	// Caller performs one JSON-RPC call over HTTP on behalf of a terminated
	// WebSocket session. Nil disables WebSocket support.
	Caller RPCCaller
	// Streams opens a synthesised subscription. Nil disables WebSocket support.
	Streams Streams
	// Credits meters spend. Nil turns metering OFF and serves everything, which
	// is how a gateway behaves before an operator switches billing on. Turning
	// it on must be a deliberate act rather than a side effect of some other
	// setting, so there is no default.
	Credits *CreditLease
	// Price is the cost in credits of one call. Nil charges one credit per call.
	Price func(method string, chainID int) int64
	// Health backs the rollup. Nil still answers, so a gateway that never wired
	// one keeps a working endpoint rather than a crashing one.
	Health *HealthProbe
	// Limiter holds each key to its plan's rate limits. Nil builds one on the
	// real clock: the upstream is keyless, so a relay without a limiter would
	// let any key send as fast as it liked, and that must not be the quiet
	// result of a missing field.
	Limiter *RateLimiter
	// TrustedProxies lists the peers, as addresses or CIDR blocks, whose
	// X-Forwarded-For the relay believes. Empty means it believes nobody and
	// checks the IP policy against the TCP peer. The header is written by
	// whoever sends the request, so reading it from an untrusted peer would let
	// any caller name an allowed address and walk through the allow-list.
	TrustedProxies []string
}

// Handler serves the public data plane.
//
// It never joins the control plane's mux. The control plane's session token
// authorises full control of the operator's servers, and a customer holds a
// key rather than that token, so the two live on separate listeners and the
// boundary is a fact of the wiring rather than a rule an edit can forget.
type Handler struct {
	cfg     Config
	trusted []*net.IPNet
}

// NewHandler builds the data-plane handler.
func NewHandler(cfg Config) (*Handler, error) {
	if cfg.Auth == nil {
		return nil, errors.New("relay: no authenticator configured")
	}
	if cfg.ERPC == nil {
		return nil, errors.New("relay: no eRPC upstream configured")
	}
	if cfg.ProjectID == "" {
		return nil, errors.New("relay: no project id configured")
	}
	if cfg.Limiter == nil {
		cfg.Limiter = NewRateLimiter(RateLimitOptions{})
	}
	trusted, err := parseTrustedProxies(cfg.TrustedProxies)
	if err != nil {
		return nil, err
	}
	return &Handler{cfg: cfg, trusted: trusted}, nil
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	route, err := ParseRoute(r.URL.Path)
	if err != nil {
		writeRouteError(w, err)
		return
	}

	// The bare liveness level is unkeyed by design, so it answers before the
	// key store is ever consulted. It stays shallow: a detailed unkeyed answer
	// would tell any scanner which chains the operator runs and when one lags.
	if route.Category == CategoryHealth && route.Depth == HealthLiveness {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		return
	}

	rec, err := h.cfg.Auth.Authenticate(r.Context(), route.Key)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	// A record that says disabled is the same refusal as a disabled verdict.
	// The store may report either, and the relay must not forward for a revoked
	// key just because the answer arrived as a record.
	if !rec.Enabled {
		writeError(w, http.StatusForbidden, "key is disabled")
		return
	}

	if err := CheckOrigin(rec, r.Header.Get("Origin")); err != nil {
		writeError(w, http.StatusForbidden, "origin not allowed for this key")
		return
	}
	if err := CheckIP(rec, h.clientIP(r)); err != nil {
		writeError(w, http.StatusForbidden, "address not allowed for this key")
		return
	}

	switch route.Category {
	case CategoryHealth:
		h.serveHealth(w, r, route)
	case CategoryBeacon:
		h.serveBeacon(w, r, route, rec)
	default:
		h.serveRPC(w, r, route, rec)
	}
}

// serveRPC applies the method policy, strips the key, and forwards to eRPC.
func (h *Handler) serveRPC(w http.ResponseWriter, r *http.Request, route Route, rec KeyRecord) {
	if err := CheckNetwork(rec, route.ChainID); err != nil {
		writeError(w, http.StatusForbidden, "chain not allowed for this key")
		return
	}

	// A WebSocket is TERMINATED here, never proxied. Proxying the upgrade would
	// put subscription management back on eRPC and bring the gzip-on-upgrade
	// hazard back with it, because there would be an upgrade on that hop again.
	// Terminating also makes upstream WebSocket support irrelevant: everything
	// past this point is plain HTTP.
	if isWebSocketUpgrade(r) {
		h.serveWebSocket(w, r, route, rec)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, maxBodyBytes))
	if err != nil {
		writeError(w, http.StatusBadRequest, "could not read the request body")
		return
	}

	methods, err := ParseMethods(body)
	if err != nil {
		// A body the relay cannot read is refused, never forwarded. Forwarding
		// it would mean an upstream serves a call the policy never checked.
		writeError(w, http.StatusBadRequest, "malformed JSON-RPC request")
		return
	}
	if err := CheckMethods(rec, methods); err != nil {
		writeError(w, http.StatusForbidden, "method not allowed for this key")
		return
	}

	// Throttle before charging, so a call the plan does not allow costs the
	// customer nothing. A batch counts as its calls, the same as charge bills it.
	if !h.admit(w, rec, len(methods)) {
		return
	}

	// Charge BEFORE forwarding, so a customer who cannot pay costs the operator
	// no upstream call at all.
	if err := h.charge(r.Context(), rec, route.ChainID, methods); err != nil {
		writeChargeError(w, err)
		return
	}

	target := *h.cfg.ERPC
	target.Path = route.UpstreamPath(h.cfg.ProjectID)
	h.forward(w, r, &target, body)
}

// charge meters one request against the key's funding account.
//
// A batch costs the sum of its calls. Charging a ten-call batch as one request
// would be a discount nobody designed, and it would make slice C's per-request
// usage rows disagree with the ledger.
func (h *Handler) charge(ctx context.Context, rec KeyRecord, chainID int, methods []string) error {
	if h.cfg.Credits == nil {
		return nil
	}
	// credit_exempt is the ONLY way a key gets free service. Keeping it the only
	// way is what makes the unbound-key case below safe to refuse.
	if rec.CreditExempt {
		return nil
	}
	// A key bound to no funding account cannot be charged. With metering on it
	// is refused rather than served free: otherwise every unbound key would be
	// a hole straight through billing, and credit_exempt would mean nothing.
	if rec.AccountAddress == "" {
		return ErrInsufficientCredits
	}

	var cost int64
	for _, method := range methods {
		cost += h.priceOf(method, chainID)
	}
	err := h.cfg.Credits.Spend(ctx, rec.AccountAddress, cost)
	if errors.Is(err, ErrNoAccount) {
		// Name the account so the operator knows what to provision. It is an
		// address, never the key.
		return fmt.Errorf("account %s: %w", rec.AccountAddress, err)
	}
	return err
}

// admit applies the key's rate limits to n calls, and answers 429 when they do
// not fit. Retry-After is whole seconds, rounded up, so a client that honours
// it never retries early.
func (h *Handler) admit(w http.ResponseWriter, rec KeyRecord, n int) bool {
	wait, ok := h.cfg.Limiter.Allow(rec, n)
	if ok {
		return true
	}
	w.Header().Set("Retry-After", strconv.FormatInt(retryAfterSeconds(wait), 10))
	writeError(w, http.StatusTooManyRequests, "rate limit exceeded for this key")
	return false
}

// retryAfterSeconds rounds a wait up to whole seconds, never below one.
func retryAfterSeconds(wait time.Duration) int64 {
	secs := int64((wait + time.Second - 1) / time.Second)
	if secs < 1 {
		secs = 1
	}
	return secs
}

// priceOf resolves one call's cost. Without a price book every call costs one
// credit, which keeps the arithmetic honest rather than free.
func (h *Handler) priceOf(method string, chainID int) int64 {
	if h.cfg.Price == nil {
		return 1
	}
	if price := h.cfg.Price(method, chainID); price > 0 {
		return price
	}
	return 1
}

// writeChargeError separates "cannot pay" from "cannot tell". Reporting a ledger
// outage as a payment problem would send a funded customer to buy credits they
// already own.
//
// A key bound to an account the ledger has never seen is a third case: the
// operator must provision the account. It is a 403 rather than a 503, so the
// operator chases an unbound account instead of a broken socket, and rather
// than a 402, because paying cannot fix it until the account exists.
func writeChargeError(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrInsufficientCredits) {
		writeError(w, http.StatusPaymentRequired, "account is out of credits")
		return
	}
	if errors.Is(err, ErrNoAccount) {
		logNoAccount(err)
		writeError(w, http.StatusForbidden, msgNotProvisioned)
		return
	}
	writeError(w, http.StatusServiceUnavailable, "the credit ledger did not answer")
}

// msgNotProvisioned is the fixed customer-facing answer for ErrNoAccount.
const msgNotProvisioned = "account not provisioned"

// logNoAccount tells the operator which account to provision. The error names
// the account (an address, never the key).
func logNoAccount(err error) {
	log.Printf("relay: refused a call: %v; the account is not provisioned in the billing store", err)
}

// serveBeacon proxies to the beacon client. Beacon is a different protocol on a
// different backend: eRPC understands /evm/ and nothing else.
func (h *Handler) serveBeacon(w http.ResponseWriter, r *http.Request, route Route, rec KeyRecord) {
	if err := CheckNetwork(rec, route.ChainID); err != nil {
		writeError(w, http.StatusForbidden, "chain not allowed for this key")
		return
	}
	if h.cfg.Beacon == nil {
		writeError(w, http.StatusNotImplemented, "this gateway serves no beacon API")
		return
	}
	if err := checkBeaconRequest(r.Method, route.Rest); err != nil {
		writeBeaconPolicyError(w, err)
		return
	}
	base, ok := h.cfg.Beacon(route.ChainID)
	if !ok || base == nil {
		// A chain with no consensus layer is a definite answer, not a dead
		// upstream. The catalog knows which chains have one.
		writeError(w, http.StatusNotImplemented, "this chain has no consensus layer")
		return
	}

	// Beacon calls draw on the same limits as JSON-RPC. A plan's rate is a
	// rate of calls to the operator's nodes, whichever API they arrive on.
	if !h.admit(w, rec, 1) {
		return
	}

	// Charge after the chain and policy checks, so a refused call costs the
	// customer nothing, and before forwarding, so an unpaid one costs the
	// operator nothing.
	if err := h.charge(r.Context(), rec, route.ChainID, []string{beaconMethod}); err != nil {
		writeChargeError(w, err)
		return
	}

	target := *base
	target.Path = route.UpstreamPath(h.cfg.ProjectID)
	h.forward(w, r, &target, nil)
}

// forward rewrites the request onto target and proxies it. body is the already
// read payload when the relay had to read one, and nil when it did not.
//
// The key is removed here and nowhere else. It cannot survive in the path,
// because the path is rebuilt from the parse result rather than edited. It
// cannot survive in a header, because Authorization is dropped: a caller may
// send the key there as well, and eRPC needs no credential from the relay.
func (h *Handler) forward(w http.ResponseWriter, r *http.Request, target *url.URL, body []byte) {
	proxy := &httputil.ReverseProxy{
		ErrorHandler: h.upstreamError,
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.Out.URL.Scheme = target.Scheme
			pr.Out.URL.Host = target.Host
			pr.Out.URL.Path = target.Path
			// The relay's grammar carries no query string of its own, so the
			// caller's is passed through untouched for the upstream to read.
			pr.Out.Host = target.Host
			pr.Out.Header.Del("Authorization")
			pr.Out.Header.Del("Proxy-Authorization")
			pr.SetXForwarded()
		},
	}
	if body != nil {
		r = r.Clone(r.Context())
		r.Body = io.NopCloser(bytes.NewReader(body))
		r.ContentLength = int64(len(body))
	}
	proxy.ServeHTTP(w, r)
}

// isWebSocketUpgrade reports whether the caller asked to upgrade. Connection is
// a comma-separated list of tokens, so it is scanned rather than compared.
func isWebSocketUpgrade(r *http.Request) bool {
	if !strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
		return false
	}
	for _, token := range strings.Split(r.Header.Get("Connection"), ",") {
		if strings.EqualFold(strings.TrimSpace(token), "upgrade") {
			return true
		}
	}
	return false
}

// serveWebSocket takes over the connection and runs one session on it.
//
// Authentication and policy already ran, so no unauthenticated caller ever gets
// a connection. Everything after the handshake is checked again per frame,
// because the handshake itself carries no method.
func (h *Handler) serveWebSocket(w http.ResponseWriter, r *http.Request, route Route, rec KeyRecord) {
	if h.cfg.Caller == nil || h.cfg.Streams == nil {
		writeError(w, http.StatusNotImplemented, "this gateway serves no WebSocket")
		return
	}

	conn, err := wsrpc.Accept(w, r, nil)
	if err != nil {
		// Accept has already answered with a status when it refused, and it
		// does not hijack on that path.
		return
	}
	defer conn.Close()

	cfg := WSConfig{
		Conn:    conn,
		Record:  rec,
		ChainID: route.ChainID,
		Caller:  h.cfg.Caller,
		Streams: h.cfg.Streams,
		Reauth:  h.reauthEvery(route.Key, wsReauthInterval),
		// Every frame draws on the same limits as an HTTP call. Without this a
		// WebSocket would be the way around every per-second limit.
		Admit: func() bool {
			_, ok := h.cfg.Limiter.Allow(rec, 1)
			return ok
		},
	}
	// The same meter as HTTP. Without it every call after the upgrade, and every
	// notification, would be served free.
	if h.cfg.Credits != nil {
		cfg.Charge = func(ctx context.Context, method string) error {
			return h.charge(ctx, rec, route.ChainID, []string{method})
		}
	}
	NewWSSession(cfg).Run(r.Context())
}

// wsReauthInterval bounds how long a revoked key keeps a live socket. The key
// cache already answers repeat lookups from memory, so this costs little.
const wsReauthInterval = 30 * time.Second

// reauthEvery re-checks a key at most once per interval.
//
// Only a definite refusal ends a session. A key store that cannot answer leaves
// it running, because the handshake already authenticated this key and dropping
// every funded customer's socket over a store blip would be the worse failure.
func (h *Handler) reauthEvery(key string, interval time.Duration) func(context.Context) error {
	var (
		mu   sync.Mutex
		last = time.Now()
	)
	return func(ctx context.Context) error {
		mu.Lock()
		defer mu.Unlock()
		if time.Since(last) < interval {
			return nil
		}
		rec, err := h.cfg.Auth.Authenticate(ctx, key)
		switch {
		case errors.Is(err, ErrUnavailable):
			return nil
		case err != nil:
			return err
		case !rec.Enabled:
			return ErrDisabledKey
		}
		last = time.Now()
		return nil
	}
}

// serveHealth answers the rollup over the category x arch x chain matrix. Each
// path level pins one more dimension, and the answer carries real state — a
// rollup that always said "ok" would be worse than none, because a monitor would
// trust it.
func (h *Handler) serveHealth(w http.ResponseWriter, r *http.Request, route Route) {
	out := map[string]any{"status": "ok"}
	if h.cfg.Health == nil {
		writeJSON(w, http.StatusOK, out)
		return
	}

	switch route.Depth {
	case HealthCell:
		out["arch"] = route.Arch
		out["chain_id"] = route.ChainID
		for k, v := range h.cfg.Health.Cell(r.Context(), route.ChainID) {
			out[k] = v
		}
	case HealthSelector:
		// Level three accepts an arch OR a category. A category selector reports
		// only the chains that actually have it, so /health/<key>/beacon does not
		// list chains with no consensus layer.
		if route.Arch != "" {
			out["arch"] = route.Arch
		} else {
			out["category"] = route.Sel
		}
		out["chains"] = h.cfg.Health.Rollup(r.Context(), route.Sel)
	case HealthAll:
		out["chains"] = h.cfg.Health.Rollup(r.Context(), "")
	}
	writeJSON(w, http.StatusOK, out)
}

// upstreamError turns a dead upstream into a plain 502. The default handler
// would log the full request URL, which carries the key.
func (h *Handler) upstreamError(w http.ResponseWriter, _ *http.Request, _ error) {
	writeError(w, http.StatusBadGateway, "upstream unavailable")
}

// writeRouteError maps a parse failure to a status. The three cases stay
// distinct on purpose: an unknown category is not addressable, a bad shape is
// the caller's mistake, and an unserved arch is a caller who has the grammar
// right and the timing wrong.
func writeRouteError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		writeError(w, http.StatusNotFound, "no such route")
	case errors.Is(err, ErrUnsupportedArch):
		writeError(w, http.StatusNotImplemented, "this arch is not served yet")
	default:
		writeError(w, http.StatusBadRequest, "malformed path")
	}
}

// writeAuthError maps a store answer to a status. An outage is a 503 and never
// a 401: reporting it as an unknown key would teach a customer to rotate a
// perfectly good credential.
func writeAuthError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrUnknownKey):
		writeError(w, http.StatusUnauthorized, "unknown key")
	case errors.Is(err, ErrDisabledKey):
		writeError(w, http.StatusForbidden, "key is disabled")
	default:
		writeError(w, http.StatusServiceUnavailable, "key store unavailable")
	}
}

// writeError answers with a fixed message. The message is never built from the
// request, so an error body cannot echo the key back into the caller's logs.
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

// clientIP reads the caller's address for the IP policy.
//
// X-Forwarded-For is believed only when the TCP peer is a trusted proxy. A
// proxy appends the address it saw to whatever the caller sent, so the list is
// read from the right, skipping hops that are themselves trusted proxies. The
// first other entry is the address the outermost trusted proxy actually saw.
// Everything to its left was written by the caller and proves nothing.
func (h *Handler) clientIP(r *http.Request) string {
	peer := r.RemoteAddr
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		peer = host
	}
	if !h.trusts(peer) {
		return peer
	}

	// Repeated headers are one list, in order, as RFC 7230 section 3.2.2 says.
	var hops []string
	for _, value := range r.Header.Values("X-Forwarded-For") {
		for _, hop := range strings.Split(value, ",") {
			if hop = strings.TrimSpace(hop); hop != "" {
				hops = append(hops, hop)
			}
		}
	}
	for i := len(hops) - 1; i >= 0; i-- {
		// An entry that is not an address is returned rather than skipped.
		// CheckIP refuses an unreadable address whenever a policy exists, which
		// is the safe answer to a list nobody can vouch for.
		if !h.trusts(hops[i]) {
			return hops[i]
		}
	}
	// Every hop is a trusted proxy, so the call began inside the operator's own
	// network. The left-most hop is as close to its origin as the list goes.
	if len(hops) > 0 {
		return hops[0]
	}
	return peer
}

// trusts reports whether addr is a configured trusted proxy.
func (h *Handler) trusts(addr string) bool {
	ip := net.ParseIP(addr)
	if ip == nil {
		return false
	}
	for _, network := range h.trusted {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

// parseTrustedProxies reads addresses and CIDR blocks. A malformed entry fails
// startup rather than being dropped: a typo in a trust list should not quietly
// change whose header the relay believes.
func parseTrustedProxies(entries []string) ([]*net.IPNet, error) {
	out := make([]*net.IPNet, 0, len(entries))
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if strings.Contains(entry, "/") {
			_, network, err := net.ParseCIDR(entry)
			if err != nil {
				return nil, fmt.Errorf("relay: trusted proxy %q: %w", entry, err)
			}
			out = append(out, network)
			continue
		}
		ip := net.ParseIP(entry)
		if ip == nil {
			return nil, fmt.Errorf("relay: trusted proxy %q is not an address or CIDR block", entry)
		}
		bits := 8 * net.IPv6len
		if v4 := ip.To4(); v4 != nil {
			ip, bits = v4, 8*net.IPv4len
		}
		out = append(out, &net.IPNet{IP: ip, Mask: net.CIDRMask(bits, bits)})
	}
	return out, nil
}
