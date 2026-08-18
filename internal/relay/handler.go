package relay

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/valve-tech/valve-node-app/internal/wsrpc"
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
}

// Handler serves the public data plane.
//
// It never joins the control plane's mux. The control plane's session token
// authorises full control of the operator's servers, and a customer holds a
// key rather than that token, so the two live on separate listeners and the
// boundary is a fact of the wiring rather than a rule an edit can forget.
type Handler struct {
	cfg Config
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
	return &Handler{cfg: cfg}, nil
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
	if err := CheckIP(rec, clientIP(r)); err != nil {
		writeError(w, http.StatusForbidden, "address not allowed for this key")
		return
	}

	switch route.Category {
	case CategoryHealth:
		h.serveHealth(w, route)
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

	target := *h.cfg.ERPC
	target.Path = route.UpstreamPath(h.cfg.ProjectID)
	h.forward(w, r, &target, body)
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
	base, ok := h.cfg.Beacon(route.ChainID)
	if !ok || base == nil {
		// A chain with no consensus layer is a definite answer, not a dead
		// upstream. The catalog knows which chains have one.
		writeError(w, http.StatusNotImplemented, "this chain has no consensus layer")
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

	NewWSSession(WSConfig{
		Conn:    conn,
		Record:  rec,
		ChainID: route.ChainID,
		Caller:  h.cfg.Caller,
		Streams: h.cfg.Streams,
	}).Run(r.Context())
}

// serveHealth answers the rollup. The matrix it reports over is filled in by
// the gateway wiring; the relay only proves here that health authenticates and
// that its shape follows the pinned depth.
func (h *Handler) serveHealth(w http.ResponseWriter, route Route) {
	out := map[string]any{"status": "ok"}
	if route.Arch != "" {
		out["arch"] = route.Arch
	}
	if route.Sel != "" {
		out["selector"] = route.Sel
	}
	if route.Depth == HealthCell {
		out["chain_id"] = route.ChainID
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

// clientIP reads the caller's address. Caddy sets X-Forwarded-For and is the
// only thing that reaches the relay, so the left-most entry is the caller.
func clientIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		first, _, _ := strings.Cut(fwd, ",")
		return strings.TrimSpace(first)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
