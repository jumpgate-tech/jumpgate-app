package capabilities

import (
	"bytes"
	"context"
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

	"github.com/valve-tech/valve-node-app/internal/wsrpc"
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

// ProbeRepeat probes t n times and folds the runs together, so a
// load-balanced endpoint whose members disagree reports that disagreement
// rather than whichever answer happened to come back first.
//
// n <= 1 behaves exactly like Probe: one run, no folding. Callers probing a
// managed node pass 1 — a single node cannot disagree with itself, and
// probing it three times is three times the load on it for no additional
// information. Repetition is worth paying for only when the URL might resolve
// to more than one backend, which is precisely the public-endpoint case
// StatusInconsistent exists for; see its doc comment for the measurement.
//
// Runs happen one after another, not concurrently: they are probing the same
// URL, so running them in parallel would not increase the chance of landing on
// different load balancer members (a fresh dial per run does that on its own,
// the same way a browser hitting reload repeatedly does), and it would
// multiply the concurrency semaphore's per-endpoint load for no benefit.
//
// The runs respect ctx cancellation between attempts: if the caller's context
// ends after run k, the remaining n-k runs are skipped and the fold proceeds
// on what was gathered. A ProbeRepeat that folded 2 of 5 runs because it was
// cancelled is honest about having done so; it never reports a verdict as if
// all 5 had run. The first run always happens regardless of ctx, the same way
// a single Probe always makes its attempt and lets the transport report the
// cancellation — see reason() and TestProbeCancelledContext.
func (p *Prober) ProbeRepeat(ctx context.Context, t Target, chainID, n int) Endpoint {
	if n <= 1 {
		return p.Probe(ctx, t, chainID)
	}

	runs := make([]Endpoint, 0, n)
	runs = append(runs, p.Probe(ctx, t, chainID))
	for i := 1; i < n; i++ {
		if ctx.Err() != nil {
			break
		}
		runs = append(runs, p.Probe(ctx, t, chainID))
	}
	return foldRuns(runs)
}

// foldRuns combines however many Probe runs ProbeRepeat managed to complete
// against one target into a single Endpoint.
//
// Endpoint-level reachability (Reachable, ChainOK, ChainID, ReachDetail,
// Origin) is taken from the LAST run that reached the endpoint, not folded
// capability-by-capability the way the Capabilities map is: reachability is
// one fact about one moment, and the same "first-hand and current" preference
// Merge applies between valve.city and a local probe applies here between an
// older run and a newer one from the same prober. If no run reached the
// endpoint, the final run's failure is reported, matching what a single Probe
// would have said.
func foldRuns(runs []Endpoint) Endpoint {
	last := runs[len(runs)-1]
	out := Endpoint{URL: last.URL, Label: last.Label, Source: last.Source}

	reachable := -1
	for i := len(runs) - 1; i >= 0; i-- {
		if runs[i].Reachable {
			reachable = i
			break
		}
	}
	if reachable == -1 {
		out.ReachDetail = last.ReachDetail
		out.Origin = last.Origin
		out.Capabilities = map[string]Result{}
		return out
	}

	r := runs[reachable]
	out.Reachable = true
	out.ChainOK = r.ChainOK
	out.ChainID = r.ChainID
	out.ReachDetail = r.ReachDetail
	out.Origin = r.Origin

	// A run that never reached the endpoint contributed no capability entries
	// at all (Probe returns early with an empty map — see TestProbeReachability's
	// "unreachable endpoint carried 0 capability results" assertion), so the
	// union of keys across runs is exactly the keys any reachable run answered.
	keys := make(map[string]bool)
	for _, run := range runs {
		for k := range run.Capabilities {
			keys[k] = true
		}
	}
	out.Capabilities = make(map[string]Result, len(keys))
	for key := range keys {
		out.Capabilities[key] = foldCapability(key, runs)
	}
	return out
}

// foldCapability applies the disagreement rules to one capability key across
// every run: agreement passes the verdict through unchanged, a genuine
// supported/unsupported split becomes StatusInconsistent with the counts in
// Detail, and inconclusive runs are evidence-free so they are ignored in
// favour of any run that did produce a real verdict.
func foldCapability(key string, runs []Endpoint) Result {
	var supported, unsupported, inconclusive []Result
	for _, run := range runs {
		res, ok := run.Cap(key)
		if !ok {
			continue // this run never reached the endpoint, or predates it
		}
		switch res.Status {
		case StatusSupported:
			supported = append(supported, res)
		case StatusUnsupported:
			unsupported = append(unsupported, res)
		default:
			// StatusInconclusive (or any status this build does not know
			// about yet) is absence of evidence, not evidence of absence —
			// the same principle that keeps a single probe's timeout from
			// being read as a refusal, applied across runs instead of within
			// one.
			inconclusive = append(inconclusive, res)
		}
	}

	switch {
	case len(supported) > 0 && len(unsupported) > 0:
		// The one case that is genuinely a new state: the fleet disagrees
		// with itself. Counts go in Detail because "3 of 5 runs said yes" is
		// the actionable form of this finding and a bare "inconsistent" is
		// not — see the ProbeRepeat and StatusInconsistent doc comments.
		method := supported[0].Method
		total := len(supported) + len(unsupported) + len(inconclusive)
		detail := fmt.Sprintf("%s → inconsistent across %d runs: %d supported, %d unsupported",
			method, total, len(supported), len(unsupported))
		if n := len(inconclusive); n > 0 {
			detail += fmt.Sprintf(", %d inconclusive", n)
		}
		return Result{Status: StatusInconsistent, Method: method, Detail: detail, Origin: OriginLocal}
	case len(supported) > 0:
		// Every run that produced evidence agreed. Passed through unchanged,
		// per the folding rule: agreement is not a new finding.
		return supported[0]
	case len(unsupported) > 0:
		return unsupported[0]
	case len(inconclusive) > 0:
		// Every run that answered this key was inconclusive. That is a real,
		// if unhelpful, agreement — none of them saw evidence either way —
		// and StatusInconclusive already says exactly that, unchanged.
		return inconclusive[0]
	default:
		// No run answered this key at all. Should not happen for a key any
		// reachable run produced (probeCapabilities answers every key), but
		// there is nothing here to fold, so say so plainly rather than
		// panicking on an index into an empty slice.
		return Result{Status: StatusInconclusive, Method: key, Detail: key + ": no run produced a verdict", Origin: OriginLocal}
	}
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
	case errors.Is(err, wsrpc.ErrRefused):
		return Result{Status: StatusUnsupported, Method: method,
			Detail: fmt.Sprintf("%s → %s", wsURL, unwrapRefusal(err))}
	case errors.Is(err, syscall.ECONNREFUSED):
		return Result{Status: StatusUnsupported, Method: method,
			Detail: fmt.Sprintf("%s → connection refused (nothing listening for WebSocket)", wsURL)}
	case errors.Is(err, wsrpc.ErrNoAnswer):
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
		return p.wsRoundTrip(tctx, rawURL, payload)
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
// WebSocket transport
//
// This used to be a copy of internal/chainlist's, carrying a comment saying so
// and saying it should call that one instead. Both are internal/wsrpc now. What
// remains here is the part that was ever this package's own: the vocabulary a
// refusal and a silence are reported in.
// ---------------------------------------------------------------------------

func (p *Prober) wsRoundTrip(ctx context.Context, rawURL string, payload []byte) ([]byte, error) {
	return wsrpc.RoundTrip(ctx, rawURL, payload, &wsrpc.Options{Dialer: p.dialer()})
}

// unwrapRefusal renders a sentinel-wrapped websocket error without repeating
// the sentinel's own words in the operator-facing detail.
func unwrapRefusal(err error) string {
	msg := err.Error()
	for _, sentinel := range []error{wsrpc.ErrRefused, wsrpc.ErrNoAnswer} {
		if prefix := sentinel.Error() + ": "; strings.HasPrefix(msg, prefix) {
			return strings.TrimPrefix(msg, prefix)
		}
	}
	return msg
}
