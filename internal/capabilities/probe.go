package capabilities

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha1"
	"crypto/tls"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

// Defaults for a zero-configuration Prober.
//
// The probe timeout matches valve.city's PROBE_TIMEOUT_MS so the two sources
// agree on what "slow enough to be inconclusive" means; a cell that says
// inconclusive here and supported there would be indistinguishable from a real
// capability difference. It is applied per probe rather than per endpoint,
// which is stricter than valve.city's single controller: one method that
// black-holes costs one timeout, not the endpoint's whole row.
const (
	defaultProbeTimeout = 10 * time.Second
	defaultConcurrency  = 16
	// maxReplyBytes caps a JSON-RPC reply. eth_getLogs over two blocks is
	// normally empty, but a misconfigured or hostile endpoint must not be able
	// to balloon memory on a small node.
	maxReplyBytes = 8 << 20
	// maxFrameBytes caps a single WebSocket frame. The only thing we ever read
	// over a socket is an eth_chainId answer, which is a few dozen bytes.
	maxFrameBytes = 1 << 20
)

// Probe arguments. The addresses are deliberately the zero address and the zero
// hash: we are testing whether a method EXISTS, not asking a node to do real
// work. A bogus tx hash makes trace_transaction answer "not found" in
// microseconds instead of tracing a real transaction, and the answer is just as
// conclusive — a node that does not implement the method says -32601 whatever
// you pass it.
const (
	zeroAddr = "0x0000000000000000000000000000000000000000"
	zeroHash = "0x0000000000000000000000000000000000000000000000000000000000000000"
)

// Target is an endpoint to probe.
type Target struct {
	URL string
	// Label is the display name; the host is used when empty.
	Label string
	// Source is provenance for the URL itself; SourceLocal when empty.
	Source Source
}

// Prober probes endpoints from this process.
//
// It exists because valve.city cannot see the endpoints that matter most to an
// operator — see the package doc. Everything is injectable so the tests never
// touch the network.
type Prober struct {
	// HTTPClient is used for every http(s) probe.
	HTTPClient *http.Client
	// ProbeTimeout bounds each individual probe, independent of the caller's
	// context. The clock starts when the probe starts, not when it is queued,
	// so a busy run does not manufacture timeouts.
	ProbeTimeout time.Duration
	// Concurrency caps in-flight round trips across the whole run. A chain like
	// Ethereum lists ~18 endpoints and each carries eleven probes; without a
	// cap that is 200 sockets opened at once from a box that may be a small VM.
	Concurrency int
	// ProbeWS enables the WebSocket probe (default true from NewProber). When
	// false, KeyWS is reported inconclusive rather than being omitted, because
	// "we chose not to look" is not the same as "not applicable".
	ProbeWS bool
	// Dialer dials the raw connections used for WebSocket probes.
	Dialer *net.Dialer

	semOnce sync.Once
	sem     chan struct{}
}

// NewProber returns a Prober with the package defaults.
func NewProber() *Prober {
	return &Prober{
		HTTPClient:   &http.Client{Timeout: 30 * time.Second},
		ProbeTimeout: defaultProbeTimeout,
		Concurrency:  defaultConcurrency,
		ProbeWS:      true,
	}
}

func (p *Prober) httpClient() *http.Client {
	if p.HTTPClient != nil {
		return p.HTTPClient
	}
	return http.DefaultClient
}

func (p *Prober) probeTimeout() time.Duration {
	if p.ProbeTimeout > 0 {
		return p.ProbeTimeout
	}
	return defaultProbeTimeout
}

func (p *Prober) concurrency() int {
	if p.Concurrency > 0 {
		return p.Concurrency
	}
	return defaultConcurrency
}

func (p *Prober) dialer() *net.Dialer {
	if p.Dialer != nil {
		return p.Dialer
	}
	return &net.Dialer{}
}

// acquire takes a slot in the global concurrency semaphore, or gives up if the
// caller's context ends first.
func (p *Prober) acquire(ctx context.Context) (func(), error) {
	p.semOnce.Do(func() { p.sem = make(chan struct{}, p.concurrency()) })
	select {
	case p.sem <- struct{}{}:
		return func() { <-p.sem }, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// ProbeAll probes every target concurrently and returns the endpoints in target
// order. Order is preserved rather than being whatever finished first, because
// the caller's ordering is meaningful (the operator's own node comes first) and
// a stable order keeps a rendered table from reshuffling between runs.
func (p *Prober) ProbeAll(ctx context.Context, targets []Target, chainID int) []Endpoint {
	out := make([]Endpoint, len(targets))
	var wg sync.WaitGroup
	for i := range targets {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			out[i] = p.Probe(ctx, targets[i], chainID)
		}(i)
	}
	wg.Wait()
	return out
}

// Probe establishes reachability and then every capability for one endpoint.
//
// Reachability comes first and gates the rest: capability verdicts from an
// endpoint that never answered would all be inconclusive, which is eleven cells
// of noise where one honest "unreachable" belongs. An endpoint serving the
// wrong chain is still probed — it is running, and knowing what it can do is
// how an operator works out what they actually pointed at.
func (p *Prober) Probe(ctx context.Context, t Target, chainID int) Endpoint {
	ep := Endpoint{
		URL:          t.URL,
		Label:        t.Label,
		Source:       t.Source,
		Origin:       OriginLocal,
		Capabilities: map[string]Result{},
	}
	if ep.Label == "" {
		ep.Label = labelFor(t.URL)
	}
	if ep.Source == "" {
		ep.Source = SourceLocal
	}

	msg, err := p.call(ctx, t.URL, "eth_chainId", nil)
	if err != nil {
		ep.ReachDetail = fmt.Sprintf("eth_chainId → no well-formed JSON-RPC reply (%s): unreachable", reason(ctx, err))
		return ep
	}

	// A JSON-RPC error reply still proves something is listening and speaking
	// the protocol, so it counts as reachable — with the chain left unclaimed.
	ep.Reachable = true
	switch {
	case !msg.hasResult():
		ep.ReachDetail = "eth_chainId → error reply (reachable, but chain unconfirmed)"
	default:
		got, perr := quantity(msg.Result)
		if perr != nil {
			ep.ReachDetail = fmt.Sprintf("eth_chainId → uninterpretable result %s (reachable, but chain unconfirmed)", strings.TrimSpace(string(msg.Result)))
			break
		}
		ok := got == chainID
		ep.ChainID = got
		ep.ChainOK = &ok
		if ok {
			ep.ReachDetail = fmt.Sprintf("eth_chainId → 0x%x (matches)", chainID)
		} else {
			ep.ReachDetail = fmt.Sprintf("eth_chainId → chain %d, expected %d (wrong chain)", got, chainID)
		}
	}

	p.probeCapabilities(ctx, &ep, chainID)
	return ep
}

// probeCapabilities runs every capability probe for one endpoint concurrently.
// They are independent single calls, so serialising them would multiply the
// worst case by eleven for no benefit; the global semaphore still bounds the
// real socket count.
func (p *Prober) probeCapabilities(ctx context.Context, ep *Endpoint, chainID int) {
	var mu sync.Mutex
	var wg sync.WaitGroup

	record := func(key string, res Result) {
		res.Origin = OriginLocal
		mu.Lock()
		ep.Capabilities[key] = res
		mu.Unlock()
	}

	for i := range capSpecs {
		wg.Add(1)
		go func(spec capSpec) {
			defer wg.Done()
			msg, err := p.call(ctx, ep.URL, spec.method, spec.params)
			record(spec.key, spec.classify(ctx, spec.method, msg, err))
		}(capSpecs[i])
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		record(KeyBatch, p.probeBatch(ctx, ep.URL))
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		record(KeyWS, p.probeWS(ctx, ep.URL, chainID))
	}()

	wg.Wait()
}

// ---------------------------------------------------------------------------
// The method set — kept identical to valve.city's, so the two sources are
// comparable cell for cell rather than merely similar.
// ---------------------------------------------------------------------------

type capSpec struct {
	key      string
	method   string
	params   []any
	classify func(ctx context.Context, method string, msg *rpcMessage, err error) Result
}

var capSpecs = []capSpec{
	{key: KeyArchive, method: "eth_getBalance", params: []any{zeroAddr, "0x1"}, classify: classifyArchive},
	{key: KeyTrace, method: "trace_transaction", params: []any{zeroHash}, classify: classifyPresence},
	{key: KeyDebug, method: "debug_traceTransaction", params: []any{zeroHash}, classify: classifyPresence},
	{key: KeyLogs, method: "eth_getLogs", params: []any{map[string]string{"fromBlock": "0x0", "toBlock": "0x1"}}, classify: classifyPresence},
	{key: KeyFilters, method: "eth_newBlockFilter", params: nil, classify: classifyPresence},
	{key: KeyOtterscan, method: "ots_getApiLevel", params: nil, classify: classifyPresence},
	{key: KeyTxpool, method: "txpool_status", params: nil, classify: classifyPresence},
	{key: KeyMsgboard, method: "msgboard_status", params: nil, classify: classifyPresence},
}

// missingMethodRE matches the ways a node says "I do not implement that". The
// code -32601 is the standard answer and is checked first, but a depressing
// number of gateways return -32000 or even 200-with-prose, so the message is
// matched too. Kept in step with MISSING_METHOD_RE in the monorepo prober.
var missingMethodRE = regexp.MustCompile(`(?i)method not found|not supported|not available|does not exist|unknown method|unsupported method|not implemented|no such method`)

// prunedStateRE matches the ways a node says "I had that state once". A pruned
// or full node answers a historical eth_getBalance with this, NOT with -32601 —
// the method is implemented, it simply cannot serve block 1 any more, which is
// exactly the distinction the archive column exists to draw.
var prunedStateRE = regexp.MustCompile(`(?i)missing trie node|state (is )?not available|no historical|not found.*state|pruned|state at block|does not have state|stateroot`)

// classifyPresence is the default rule: the method EXISTS unless the node says
// it does not.
//
// The counter-intuitive branch is the middle one. An error like "invalid
// params" or "transaction not found" is treated as SUPPORTED, because the node
// had to route the call to a handler and run it in order to complain about the
// arguments. Only a routing-level refusal proves absence.
func classifyPresence(ctx context.Context, method string, msg *rpcMessage, err error) Result {
	if err != nil {
		return Result{Status: StatusInconclusive, Method: method,
			Detail: fmt.Sprintf("%s: no reply (%s)", method, reason(ctx, err))}
	}
	if msg.Error != nil {
		if isMissingMethod(msg.Error) {
			return Result{Status: StatusUnsupported, Method: method,
				Detail: strings.TrimSpace(fmt.Sprintf("%s → %d %s", method, msg.Error.Code, msg.Error.Message))}
		}
		return Result{Status: StatusSupported, Method: method,
			Detail: strings.TrimSpace(fmt.Sprintf("%s → error %d %s (method exists)", method, msg.Error.Code, msg.Error.Message))}
	}
	return Result{Status: StatusSupported, Method: method, Detail: method + " → returned a result"}
}

// classifyArchive decides the archive column, where method presence is not the
// question — every node implements eth_getBalance. The question is whether it
// still holds state for block 1.
func classifyArchive(ctx context.Context, method string, msg *rpcMessage, err error) Result {
	if err != nil {
		return Result{Status: StatusInconclusive, Method: method,
			Detail: fmt.Sprintf("%s: no reply (%s)", method, reason(ctx, err))}
	}
	if msg.hasResult() {
		return Result{Status: StatusSupported, Method: method,
			Detail: method + `(…, "0x1") → returned historical state`}
	}
	if isMissingMethod(msg.Error) {
		return Result{Status: StatusUnsupported, Method: method, Detail: method + " → method not found"}
	}
	if msg.Error != nil && prunedStateRE.MatchString(msg.Error.Message) {
		return Result{Status: StatusUnsupported, Method: method,
			Detail: fmt.Sprintf("%s → %q (pruned / full node, no archive)", method, msg.Error.Message)}
	}
	// An error we cannot read is not evidence of pruning. Say so.
	return Result{Status: StatusInconclusive, Method: method,
		Detail: fmt.Sprintf("%s → %s", method, errMessageOr(msg.Error, "unclear"))}
}

func isMissingMethod(e *rpcError) bool {
	if e == nil {
		return false
	}
	return e.Code == -32601 || missingMethodRE.MatchString(e.Message)
}

func errMessageOr(e *rpcError, fallback string) string {
	if e == nil || e.Message == "" {
		return fallback
	}
	return e.Message
}

// probeBatch sends a two-item JSON-RPC batch. Batching is not a method, so it
// cannot be asked about — the only way to know is to send one and count the
// replies. A gateway that quietly answers only the first item, or answers with
// a single object, is not batching however cheerfully it accepts the request.
func (p *Prober) probeBatch(ctx context.Context, rawURL string) Result {
	const method = "batch"
	body := []byte(`[{"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1},` +
		`{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":2}]`)

	raw, err := p.do(ctx, rawURL, body)
	if err != nil {
		return Result{Status: StatusInconclusive, Method: method,
			Detail: fmt.Sprintf("batch → no reply (%s)", reason(ctx, err))}
	}
	var items []json.RawMessage
	if err := json.Unmarshal(raw, &items); err != nil {
		return Result{Status: StatusUnsupported, Method: method,
			Detail: "sent a 2-item batch → reply was not a 2-element array"}
	}
	if len(items) != 2 {
		return Result{Status: StatusUnsupported, Method: method,
			Detail: fmt.Sprintf("sent a 2-item batch → reply was a %d-element array", len(items))}
	}
	return Result{Status: StatusSupported, Method: method,
		Detail: "sent a 2-item batch → got a 2-element array reply"}
}

// probeWS is the capability valve.city does not cover, and the one an operator
// is most likely to be lied to about. See the package doc for the measurements
// that motivate it: published wss:// endpoints that fail the handshake outright,
// and eRPC inferring WebSocket support from a URL scheme.
//
// Classification is deliberately asymmetric about what counts as a refusal:
//
//   - A server that answers the Upgrade with anything other than 101, or with a
//     bad Sec-WebSocket-Accept digest, has refused. Unsupported.
//   - A connection refused means nothing is listening for WebSocket at that
//     address. Unsupported.
//   - A timeout, a cancellation, a TLS failure or a DNS failure tells us about
//     the path, not the endpoint. Inconclusive.
//   - A successful handshake that then produces no usable eth_chainId is
//     inconclusive, and the detail says the handshake worked — that is a
//     genuinely different failure from an upgrade being refused, and the
//     operator needs to see which one they have.
func (p *Prober) probeWS(ctx context.Context, rawURL string, chainID int) Result {
	const method = "ws"
	if !p.ProbeWS {
		return Result{Status: StatusInconclusive, Method: method, Detail: "WebSocket probing disabled"}
	}

	wsURL, ok := DeriveWSURL(rawURL)
	if !ok {
		return Result{Status: StatusInconclusive, Method: method,
			Detail: fmt.Sprintf("%q has no WebSocket form", rawURL)}
	}

	raw, err := p.do(ctx, wsURL, []byte(chainIDRequest))
	switch {
	case errors.Is(err, errWSRefused):
		return Result{Status: StatusUnsupported, Method: method,
			Detail: fmt.Sprintf("%s → %s", wsURL, unwrapRefusal(err))}
	case errors.Is(err, syscall.ECONNREFUSED):
		return Result{Status: StatusUnsupported, Method: method,
			Detail: fmt.Sprintf("%s → connection refused (nothing listening for WebSocket)", wsURL)}
	case errors.Is(err, errWSNoAnswer):
		return Result{Status: StatusInconclusive, Method: method,
			Detail: fmt.Sprintf("%s → handshake succeeded but eth_chainId over the socket did not answer (%s)", wsURL, unwrapRefusal(err))}
	case err != nil:
		return Result{Status: StatusInconclusive, Method: method,
			Detail: fmt.Sprintf("%s → %s", wsURL, reason(ctx, err))}
	}

	var msg rpcMessage
	if err := json.Unmarshal(raw, &msg); err != nil || !msg.hasResult() {
		return Result{Status: StatusInconclusive, Method: method,
			Detail: fmt.Sprintf("%s → handshake succeeded but eth_chainId over the socket returned no usable result", wsURL)}
	}
	got, perr := quantity(msg.Result)
	if perr != nil {
		return Result{Status: StatusInconclusive, Method: method,
			Detail: fmt.Sprintf("%s → handshake succeeded but eth_chainId over the socket was uninterpretable", wsURL)}
	}
	// The transport works, which is what this column asks. A chain mismatch is
	// reported in the detail rather than downgrading the verdict; the
	// reachability column is where wrong-chain belongs.
	if got != chainID {
		return Result{Status: StatusSupported, Method: method,
			Detail: fmt.Sprintf("%s → handshake OK, eth_chainId over WS returned chain %d (expected %d)", wsURL, got, chainID)}
	}
	return Result{Status: StatusSupported, Method: method,
		Detail: fmt.Sprintf("%s → handshake OK, eth_chainId over WS returned 0x%x", wsURL, got)}
}

// DeriveWSURL returns the WebSocket URL to probe for an endpoint, and whether
// one exists.
//
// A ws:// or wss:// endpoint is probed as given. An http(s) endpoint is probed
// at the same host and path with the scheme swapped, which is the convention
// every provider we have measured follows. The derived URL is always named in
// the result detail, so an operator whose provider puts WebSocket somewhere
// else can see that we asked the wrong door rather than being told a flat no.
func DeriveWSURL(raw string) (string, bool) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" {
		return "", false
	}
	switch strings.ToLower(u.Scheme) {
	case "ws", "wss":
		return raw, true
	case "http":
		u.Scheme = "ws"
	case "https":
		u.Scheme = "wss"
	default:
		return "", false
	}
	return u.String(), true
}

// labelFor names a row when the caller did not. The host is what an operator
// recognises; the full URL is already on the row.
func labelFor(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" {
		return raw
	}
	return u.Host
}

// ---------------------------------------------------------------------------
// JSON-RPC plumbing
// ---------------------------------------------------------------------------

// chainIDRequest is the cheapest call that still proves identity: no arguments,
// served by every EVM node regardless of pruning or archive mode.
const chainIDRequest = `{"jsonrpc":"2.0","id":1,"method":"eth_chainId","params":[]}`

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type rpcMessage struct {
	Result json.RawMessage `json:"result"`
	Error  *rpcError       `json:"error"`
}

// hasResult reports whether the reply carried a usable result. A present-but-
// null result is not one: eth_getBalance answering null is a node declining to
// serve the state, not a node handing over a balance.
func (m *rpcMessage) hasResult() bool {
	if m == nil {
		return false
	}
	trimmed := bytes.TrimSpace(m.Result)
	return len(trimmed) > 0 && !bytes.Equal(trimmed, []byte("null"))
}

// call sends one JSON-RPC request and decodes the envelope. A non-nil error
// means no well-formed reply arrived — the caller turns that into inconclusive,
// never into unsupported.
func (p *Prober) call(ctx context.Context, rawURL, method string, params []any) (*rpcMessage, error) {
	if params == nil {
		params = []any{}
	}
	body, err := json.Marshal(struct {
		JSONRPC string `json:"jsonrpc"`
		Method  string `json:"method"`
		Params  []any  `json:"params"`
		ID      int    `json:"id"`
	}{"2.0", method, params, 1})
	if err != nil {
		return nil, fmt.Errorf("build %s request: %w", method, err)
	}

	raw, err := p.do(ctx, rawURL, body)
	if err != nil {
		return nil, err
	}
	var msg rpcMessage
	if err := json.Unmarshal(raw, &msg); err != nil {
		return nil, fmt.Errorf("malformed JSON-RPC response: %w", err)
	}
	return &msg, nil
}

// do performs one round trip over whichever transport the URL names.
//
// The semaphore is taken before the deadline is set, so time spent queueing
// behind other probes is not charged to this probe. Getting that backwards
// turns a busy run into a wall of spurious "inconclusive", which is exactly the
// failure mode the three-state design exists to avoid.
func (p *Prober) do(ctx context.Context, rawURL string, payload []byte) ([]byte, error) {
	release, err := p.acquire(ctx)
	if err != nil {
		return nil, err
	}
	defer release()

	tctx, cancel := context.WithTimeout(ctx, p.probeTimeout())
	defer cancel()

	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || u.Host == "" {
		return nil, fmt.Errorf("bad URL %q", rawURL)
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https":
		return p.httpRoundTrip(tctx, rawURL, payload)
	case "ws", "wss":
		return p.wsRoundTrip(tctx, u, payload)
	}
	return nil, fmt.Errorf("unsupported URL scheme %q", u.Scheme)
}

func (p *Prober) httpRoundTrip(ctx context.Context, rawURL string, payload []byte) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, rawURL, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("bad URL: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := p.httpClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Deliberately not read as a refusal. A 429 or a 502 says the gateway
		// would not carry the question today, which is no evidence at all about
		// what the node behind it can do.
		return nil, fmt.Errorf("HTTP %s", resp.Status)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxReplyBytes))
	if err != nil {
		return nil, err
	}
	return body, nil
}

// quantity reads a JSON-RPC quantity. It is normally hex ("0x171"), but a
// handful of endpoints answer in decimal, so base-0 detection is used rather
// than assuming a prefix.
func quantity(raw json.RawMessage) (int, error) {
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return 0, fmt.Errorf("not a JSON-RPC quantity: %w", err)
	}
	n, err := strconv.ParseUint(strings.TrimSpace(s), 0, 64)
	if err != nil {
		return 0, fmt.Errorf("uninterpretable quantity %q", s)
	}
	return int(n), nil
}

// reason turns a transport error into an operator-facing phrase, calling a
// timeout a timeout instead of leaking "context deadline exceeded" into a
// tooltip.
func reason(ctx context.Context, err error) string {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, context.DeadlineExceeded), errors.Is(ctx.Err(), context.DeadlineExceeded):
		return "timed out"
	case errors.Is(err, context.Canceled), errors.Is(ctx.Err(), context.Canceled):
		return "cancelled"
	}
	// url.Error's prefix ("Post \"https://…\": ") repeats the URL the caller
	// already has; unwrap one layer for a tighter message.
	var uerr *url.Error
	if errors.As(err, &uerr) && uerr.Err != nil {
		return uerr.Err.Error()
	}
	return err.Error()
}

// ---------------------------------------------------------------------------
// WebSocket transport — a minimal RFC 6455 client on the standard library only.
//
// This mirrors internal/chainlist's probeWS deliberately and reluctantly. That
// implementation is the right one and this should call it, but its probe
// helpers are unexported and this package is not allowed to widen them; see the
// note returned with this work. Scope is one request and one response: open,
// handshake, send one masked text frame, read one message, close. No
// compression, no subprotocols, no continuation frames on send.
// ---------------------------------------------------------------------------

// wsGUID is the RFC 6455 §1.3 magic value the server mixes into the
// Sec-WebSocket-Accept digest.
const wsGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

// errWSRefused marks a definitive refusal of the upgrade — the server answered
// and said no. It is the difference between "this endpoint does not do
// WebSocket" and "we could not tell", so it is a sentinel rather than a string
// the caller has to pattern-match.
var errWSRefused = errors.New("websocket upgrade refused")

// errWSNoAnswer marks a handshake that succeeded followed by a socket that did
// not produce a message. Also load-bearing: an eRPC upstream that infers
// WebSocket from a URL scheme upgrades happily and then goes quiet.
var errWSNoAnswer = errors.New("websocket answered nothing")

func (p *Prober) wsRoundTrip(ctx context.Context, u *url.URL, payload []byte) ([]byte, error) {
	secure := strings.EqualFold(u.Scheme, "wss")

	addr := u.Host
	if u.Port() == "" {
		if secure {
			addr = net.JoinHostPort(u.Hostname(), "443")
		} else {
			addr = net.JoinHostPort(u.Hostname(), "80")
		}
	}

	conn, err := p.dialer().DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	// The context governs the whole exchange: a deadline goes onto the socket,
	// and a cancellation slams the connection shut so a blocked read unblocks.
	if dl, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(dl)
	}
	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
			_ = conn.Close()
		case <-done:
		}
	}()

	if secure {
		tconn := tls.Client(conn, &tls.Config{ServerName: u.Hostname()})
		if err := tconn.HandshakeContext(ctx); err != nil {
			return nil, err
		}
		conn = tconn
	}

	br := bufio.NewReader(conn)
	if err := wsHandshake(conn, br, u); err != nil {
		return nil, err
	}
	if err := wsWriteFrame(conn, payload); err != nil {
		return nil, fmt.Errorf("%w: %v", errWSNoAnswer, err)
	}
	msg, err := wsReadMessage(br)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", errWSNoAnswer, err)
	}
	return msg, nil
}

// wsHandshake performs the opening HTTP Upgrade and verifies the accept digest.
// Without the digest check we would happily "talk WebSocket" to a plain HTTP
// server that echoed a 101 by accident — which is precisely the confusion this
// probe exists to detect.
func wsHandshake(conn net.Conn, br *bufio.Reader, u *url.URL) error {
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return fmt.Errorf("websocket: nonce: %w", err)
	}
	key := base64.StdEncoding.EncodeToString(nonce[:])

	path := u.RequestURI()
	if path == "" {
		path = "/"
	}
	var req bytes.Buffer
	fmt.Fprintf(&req, "GET %s HTTP/1.1\r\n", path)
	fmt.Fprintf(&req, "Host: %s\r\n", u.Host)
	req.WriteString("Upgrade: websocket\r\n")
	req.WriteString("Connection: Upgrade\r\n")
	fmt.Fprintf(&req, "Sec-WebSocket-Key: %s\r\n", key)
	req.WriteString("Sec-WebSocket-Version: 13\r\n\r\n")
	if _, err := conn.Write(req.Bytes()); err != nil {
		return err
	}

	// http.ReadResponse understands the 101 and leaves the frame stream
	// untouched in br, which is exactly the split we need.
	resp, err := http.ReadResponse(br, &http.Request{Method: http.MethodGet})
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusSwitchingProtocols {
		return fmt.Errorf("%w: HTTP %s", errWSRefused, resp.Status)
	}
	if !strings.EqualFold(resp.Header.Get("Upgrade"), "websocket") {
		return fmt.Errorf("%w: 101 without Upgrade: websocket", errWSRefused)
	}
	sum := sha1.Sum([]byte(key + wsGUID))
	if want := base64.StdEncoding.EncodeToString(sum[:]); resp.Header.Get("Sec-WebSocket-Accept") != want {
		return fmt.Errorf("%w: bad Sec-WebSocket-Accept digest", errWSRefused)
	}
	return nil
}

// unwrapRefusal renders a sentinel-wrapped websocket error without repeating
// the sentinel's own words in the operator-facing detail.
func unwrapRefusal(err error) string {
	msg := err.Error()
	for _, sentinel := range []error{errWSRefused, errWSNoAnswer} {
		if prefix := sentinel.Error() + ": "; strings.HasPrefix(msg, prefix) {
			return strings.TrimPrefix(msg, prefix)
		}
	}
	return msg
}

// wsWriteFrame writes payload as a single masked text frame. Clients must mask
// (RFC 6455 §5.3); servers reject unmasked client frames.
func wsWriteFrame(w io.Writer, payload []byte) error {
	var mask [4]byte
	if _, err := rand.Read(mask[:]); err != nil {
		return fmt.Errorf("websocket: mask: %w", err)
	}

	var buf bytes.Buffer
	buf.WriteByte(0x81) // FIN | opcode 1 (text)
	switch n := len(payload); {
	case n < 126:
		buf.WriteByte(0x80 | byte(n))
	case n <= 0xFFFF:
		buf.WriteByte(0x80 | 126)
		_ = binary.Write(&buf, binary.BigEndian, uint16(n))
	default:
		buf.WriteByte(0x80 | 127)
		_ = binary.Write(&buf, binary.BigEndian, uint64(n))
	}
	buf.Write(mask[:])
	for i, b := range payload {
		buf.WriteByte(b ^ mask[i%4])
	}
	_, err := w.Write(buf.Bytes())
	return err
}

// wsReadMessage reads frames until a complete data message arrives, skipping
// the control frames a server may interleave (a ping before the answer is
// normal on endpoints with an idle timer). Fragmented messages are reassembled.
func wsReadMessage(br *bufio.Reader) ([]byte, error) {
	var msg []byte
	for {
		fin, opcode, payload, err := wsReadFrame(br)
		if err != nil {
			return nil, err
		}
		switch opcode {
		case 0x0, 0x1, 0x2: // continuation, text, binary
			msg = append(msg, payload...)
			if fin {
				return msg, nil
			}
		case 0x8: // close
			return nil, errors.New("closed before answering")
		case 0x9, 0xA: // ping, pong — nothing to do for a single round trip
		default:
			return nil, fmt.Errorf("unexpected opcode %#x", opcode)
		}
	}
}

// wsReadFrame reads one frame header and its payload, unmasking if the peer
// masked it (servers must not, but be liberal).
func wsReadFrame(br *bufio.Reader) (fin bool, opcode byte, payload []byte, err error) {
	var hdr [2]byte
	if _, err = io.ReadFull(br, hdr[:]); err != nil {
		return false, 0, nil, err
	}
	fin = hdr[0]&0x80 != 0
	opcode = hdr[0] & 0x0F
	masked := hdr[1]&0x80 != 0

	length := uint64(hdr[1] & 0x7F)
	switch length {
	case 126:
		var ext [2]byte
		if _, err = io.ReadFull(br, ext[:]); err != nil {
			return false, 0, nil, err
		}
		length = uint64(binary.BigEndian.Uint16(ext[:]))
	case 127:
		var ext [8]byte
		if _, err = io.ReadFull(br, ext[:]); err != nil {
			return false, 0, nil, err
		}
		length = binary.BigEndian.Uint64(ext[:])
	}
	if length > maxFrameBytes {
		// An eth_chainId answer is tiny; a peer claiming otherwise is either
		// broken or hostile, and we will not allocate for it.
		return false, 0, nil, fmt.Errorf("oversized frame (%d bytes)", length)
	}

	var mask [4]byte
	if masked {
		if _, err = io.ReadFull(br, mask[:]); err != nil {
			return false, 0, nil, err
		}
	}
	payload = make([]byte, length)
	if _, err = io.ReadFull(br, payload); err != nil {
		return false, 0, nil, err
	}
	if masked {
		for i := range payload {
			payload[i] ^= mask[i%4]
		}
	}
	return fin, opcode, payload, nil
}
