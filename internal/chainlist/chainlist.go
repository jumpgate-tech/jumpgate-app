// Package chainlist discovers live public RPC upstreams for a chain, for use
// as eRPC fallback upstreams.
//
// catalog ships one hand-picked default fallback per chain
// (catalog.DefaultUpstreams). That list is deliberately conservative and goes
// stale the moment a provider retires an endpoint. This package is the
// dynamic counterpart: it reads the canonical ethereum-lists/chains feed —
// https://chainid.network/chains.json, the same data chainlist.org renders —
// and returns only the endpoints that are answering *right now* for the chain
// the operator is provisioning.
//
// Three properties drive the design:
//
//   - The feed is advertising, not truth. Roughly a third of the URLs it
//     lists for a popular chain are dead, rate-limited, or serving a
//     different chain than claimed. Nothing goes into an erpc.yaml without
//     an eth_chainId round trip proving otherwise.
//   - Some entries carry a slot rather than a key. Provider entries use shell-
//     style placeholders for an API key (https://mainnet.infura.io/v3/${INFURA_API_KEY});
//     chain 1 has two. A slot is RESOLVED from the operator's stored provider
//     keys and then probed like anything else, because an endpoint the
//     operator has paid for is the best upstream on offer and the probe is the
//     only way to know it works. A slot with no key stored is not probed — it
//     could only 401 — and comes back rejected, naming the key to go and get.
//   - The operator may be offline. valve-node-app runs on freshly imaged
//     boxes and air-gapped racks. When the feed is unreachable, a vendored
//     snapshot (vendored.go) stands in, so upstream discovery degrades to a
//     known-good list instead of an empty one.
//
// Everything is context-aware and both the HTTP client and the feed URL are
// injectable, so the tests in this package never touch the network.
//
// WebSocket endpoints are probed too, through internal/wsrpc. WebSocket
// upstreams matter to us — eRPC uses them for subscriptions — and an endpoint
// that advertises wss:// without being able to speak it is common enough to be
// worth the round trip. Callers that want to skip WebSocket probing entirely
// (say, a network that blocks Upgrade) can set Discoverer.ProbeWS to false;
// those endpoints then come back as StatusUnprobed rather than being dropped
// or wrongly marked live.
package chainlist

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/valve-tech/valve-node-app/internal/wsrpc"
)

// FeedURL is the canonical ethereum-lists/chains feed. ~1.1 MB, ~2660 chains
// as of the vendored snapshot date.
const FeedURL = "https://chainid.network/chains.json"

// Defaults for a zero-configuration Discoverer. The probe timeout is
// deliberately short: a public endpoint that cannot answer the cheapest call
// in the JSON-RPC surface within five seconds is not a useful fallback, and
// discovery runs interactively inside the setup wizard where the operator is
// watching a spinner.
const (
	defaultProbeTimeout = 5 * time.Second
	defaultConcurrency  = 16
	// maxFeedBytes caps the feed read so a hostile or misconfigured mirror
	// cannot balloon memory on a small node. The real feed is ~1.1 MB.
	maxFeedBytes = 32 << 20
	// maxProbeBytes caps an HTTP probe response body. An eth_chainId answer is
	// a few dozen bytes. The WebSocket side is capped separately, by
	// wsrpc.DefaultMaxMessageBytes — this number used to serve both, and being
	// two orders of magnitude apart from the other two copies of that reader is
	// what made the duplication worth ending.
	maxProbeBytes = 64 << 10
)

// Kind is an endpoint's transport, derived from its URL scheme.
type Kind string

const (
	KindHTTP Kind = "http" // http:// or https://
	KindWS   Kind = "ws"   // ws:// or wss://
)

// Status is the outcome of considering one feed entry as an upstream.
type Status string

const (
	// StatusPending is a candidate that has passed filtering but has not
	// been probed yet. Candidates returns these; Discover never does.
	StatusPending Status = "pending"
	// StatusLive means the endpoint answered eth_chainId with the requested
	// chain id.
	StatusLive Status = "live"
	// StatusUnprobed means the endpoint survived filtering but was not
	// probed — today only WebSocket endpoints when ProbeWS is false. Such an
	// endpoint is still usable (it is in URLs()); it simply carries no
	// liveness evidence.
	StatusUnprobed Status = "unprobed"
	// StatusRejected means the entry was filtered out or failed its probe.
	// Reason says why, in words meant for an operator.
	StatusRejected Status = "rejected"
)

// Source records where a Result's endpoint list came from, so a caller can
// tell the operator "these are live from chainlist" versus "the feed was
// unreachable, these are the ones we shipped with".
type Source string

const (
	SourceFeed     Source = "feed"
	SourceVendored Source = "vendored"
)

// Endpoint is one RPC URL and what we learned about it.
type Endpoint struct {
	URL  string
	Kind Kind

	Status Status
	// ChainID is the chain id the endpoint reported, 0 if it never answered.
	// It is kept even on rejection so the caller can say "that one is
	// actually chain 1" rather than a bare "rejected".
	ChainID int
	// Latency is the probe round-trip time, 0 if unprobed.
	Latency time.Duration
	// Reason is the operator-facing explanation for a rejection, empty
	// otherwise.
	Reason string
}

// Result is the outcome of a discovery run.
type Result struct {
	ChainID int
	Source  Source
	// FetchErr is why the live feed was not used; non-nil exactly when
	// Source is SourceVendored.
	FetchErr error
	// Endpoints holds every candidate — live, unprobed and rejected alike —
	// in feed order, so a UI can render the whole picture and explain each
	// exclusion.
	Endpoints []Endpoint
}

// Live returns the endpoints that proved they serve the requested chain.
func (r Result) Live() []Endpoint { return r.filter(StatusLive) }

// Unprobed returns the endpoints that were kept without liveness evidence.
func (r Result) Unprobed() []Endpoint { return r.filter(StatusUnprobed) }

// Rejected returns the endpoints that were filtered out or failed their
// probe, each carrying a Reason.
func (r Result) Rejected() []Endpoint { return r.filter(StatusRejected) }

func (r Result) filter(s Status) []Endpoint {
	var out []Endpoint
	for _, ep := range r.Endpoints {
		if ep.Status == s {
			out = append(out, ep)
		}
	}
	return out
}

// URLs returns the usable upstream URLs in feed order — the live ones plus
// any unprobed ones. This is the list a caller hands to
// catalog.WireConfig.ERPCUpstreams.
func (r Result) URLs() []string {
	var out []string
	for _, ep := range r.Endpoints {
		if ep.Status == StatusLive || ep.Status == StatusUnprobed {
			out = append(out, ep.URL)
		}
	}
	return out
}

// Chain is the slice of a chains.json entry we care about. The feed carries
// far more per chain (native currency, explorers, parent chains, icons); we
// decode only these three fields so a schema addition upstream cannot break
// us.
type Chain struct {
	ChainID int      `json:"chainId"`
	Name    string   `json:"name"`
	RPC     []string `json:"rpc"`
}

// Discoverer fetches and probes RPC endpoints. The zero value is not usable;
// call New and override fields as needed.
type Discoverer struct {
	// HTTPClient is used for both the feed fetch and the HTTP(S) probes, so
	// a test can serve everything in-process with a stub RoundTripper.
	HTTPClient *http.Client
	// FeedURL is the chains.json source; defaults to FeedURL.
	FeedURL string
	// ProbeTimeout bounds each individual probe, independent of the caller's
	// context. One dead endpoint must not hold up the batch.
	ProbeTimeout time.Duration
	// Concurrency caps in-flight probes. A chain like Ethereum lists ~18
	// endpoints and we want them checked in roughly one timeout's worth of
	// wall clock, without opening an unbounded number of sockets.
	Concurrency int
	// ProbeWS enables the stdlib WebSocket prober (default true). When
	// false, ws:// and wss:// endpoints are returned as StatusUnprobed.
	ProbeWS bool
	// Dialer dials the raw connections used for WebSocket probes. Defaults
	// to a plain net.Dialer.
	Dialer *net.Dialer
	// Keys holds provider API keys by placeholder name (e.g.
	// "INFURA_API_KEY"), used to resolve templated feed URLs into real
	// endpoints before probing. Nil means no templated URL can be filled in,
	// so every provider slot is rejected — the same behaviour as before this
	// field existed.
	Keys map[string]string
}

// New returns a Discoverer with the package defaults.
func New() *Discoverer {
	return &Discoverer{
		HTTPClient:   &http.Client{Timeout: 30 * time.Second},
		FeedURL:      FeedURL,
		ProbeTimeout: defaultProbeTimeout,
		Concurrency:  defaultConcurrency,
		ProbeWS:      true,
	}
}

func (d *Discoverer) httpClient() *http.Client {
	if d.HTTPClient != nil {
		return d.HTTPClient
	}
	return http.DefaultClient
}

func (d *Discoverer) feedURL() string {
	if d.FeedURL != "" {
		return d.FeedURL
	}
	return FeedURL
}

func (d *Discoverer) probeTimeout() time.Duration {
	if d.ProbeTimeout > 0 {
		return d.ProbeTimeout
	}
	return defaultProbeTimeout
}

func (d *Discoverer) concurrency() int {
	if d.Concurrency > 0 {
		return d.Concurrency
	}
	return defaultConcurrency
}

func (d *Discoverer) dialer() *net.Dialer {
	if d.Dialer != nil {
		return d.Dialer
	}
	return &net.Dialer{}
}

// Fetch downloads and decodes the whole feed. Callers normally want Discover;
// Fetch is exported for the rare caller that wants to enumerate chains (e.g.
// "which chain ids does chainlist know about?").
func (d *Discoverer) Fetch(ctx context.Context) ([]Chain, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, d.feedURL(), nil)
	if err != nil {
		return nil, fmt.Errorf("chainlist: build feed request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := d.httpClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("chainlist: fetch %s: %w", d.feedURL(), err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("chainlist: fetch %s: unexpected status %s", d.feedURL(), resp.Status)
	}

	var chains []Chain
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxFeedBytes)).Decode(&chains); err != nil {
		return nil, fmt.Errorf("chainlist: decode feed: %w", err)
	}
	return chains, nil
}

// Discover returns every RPC endpoint the feed lists for chainID, each
// annotated with whether it is usable and why not if it isn't.
//
// A feed failure is not a discovery failure: if a vendored snapshot exists
// for the chain it is used instead, and the reason is preserved in
// Result.FetchErr so the caller can warn ("chainlist unreachable, using the
// built-in list"). Only a failure with no fallback is returned as an error.
func (d *Discoverer) Discover(ctx context.Context, chainID int) (Result, error) {
	if err := ctx.Err(); err != nil {
		return Result{}, fmt.Errorf("chainlist: discover chain %d: %w", chainID, err)
	}

	res := Result{ChainID: chainID, Source: SourceFeed}

	rpcs, err := d.rpcsForChain(ctx, chainID)
	if err != nil {
		// A cancelled caller does not want the vendored consolation prize;
		// it wants to stop.
		if ctxErr := ctx.Err(); ctxErr != nil {
			return Result{}, fmt.Errorf("chainlist: discover chain %d: %w", chainID, ctxErr)
		}
		vendored, ok := Vendored(chainID)
		if !ok {
			return Result{}, fmt.Errorf("chainlist: no upstreams for chain %d and no vendored fallback: %w", chainID, err)
		}
		res.Source = SourceVendored
		res.FetchErr = err
		rpcs = vendored
	}

	res.Endpoints = d.probeAll(ctx, chainID, Candidates(rpcs, d.Keys))
	return res, nil
}

// rpcsForChain fetches the feed and pulls out one chain's rpc list. A chain
// missing from an otherwise healthy feed is reported as an error so it takes
// the same vendored-fallback path as an unreachable feed — from the
// operator's seat "chainlist doesn't list your chain" and "chainlist is down"
// have the same remedy.
func (d *Discoverer) rpcsForChain(ctx context.Context, chainID int) ([]string, error) {
	chains, err := d.Fetch(ctx)
	if err != nil {
		return nil, err
	}
	for _, c := range chains {
		if c.ChainID == chainID {
			return c.RPC, nil
		}
	}
	return nil, fmt.Errorf("chainlist: chain id %d not present in %s", chainID, d.feedURL())
}

// Candidates classifies raw feed URLs, returning one Endpoint per distinct
// input in input order. Entries that cannot be used are returned as
// StatusRejected with a Reason rather than dropped, so a UI can show the
// operator why an endpoint they expected to see is missing. Everything else
// comes back StatusPending, ready to probe.
//
// keys is provider API keys by placeholder name, used to resolve templated
// slots like ${INFURA_API_KEY} into real endpoints before probing. A nil or
// incomplete map is fine — anything left unresolved is rejected, naming the
// placeholder it needs.
//
// Exported because the filtering rules are useful on their own — e.g. to sanity
// check a hand-written upstream list — and because they are what the tests
// pin down.
func Candidates(rpcs []string, keys map[string]string) []Endpoint {
	out := make([]Endpoint, 0, len(rpcs))
	seen := make(map[string]bool, len(rpcs))

	for _, raw := range rpcs {
		if seen[raw] {
			continue
		}
		seen[raw] = true

		ep := Endpoint{URL: raw}
		switch {
		case isTemplated(raw):
			// A provider slot. If a key for it is known, fill it in and let
			// it be probed like anything else — resolving is not evidence,
			// only answering is. Otherwise reject as before, but name the
			// key: the old message said an account was required without
			// saying which, which is the difference between a dead end and
			// a next step.
			name := PlaceholderName(raw)
			if resolved, ok := Resolve(raw, keys); ok {
				kind := kindOf(resolved)
				if kind == "" {
					ep.Status = StatusRejected
					ep.Reason = "unsupported URL scheme (want http, https, ws or wss)"
					break
				}
				ep.URL = resolved
				ep.Kind = kind
				ep.Status = StatusPending
				break
			}
			ep.Kind = kindOf(raw)
			ep.Status = StatusRejected
			if name != "" {
				ep.Reason = "needs " + name + " — add it in Settings"
			} else {
				ep.Reason = "API-key template (contains ${...}); requires a provider account"
			}
		default:
			kind := kindOf(raw)
			if kind == "" {
				ep.Status = StatusRejected
				ep.Reason = "unsupported URL scheme (want http, https, ws or wss)"
				break
			}
			ep.Kind = kind
			ep.Status = StatusPending
		}
		out = append(out, ep)
	}
	return out
}

// isTemplated reports whether raw carries a shell-style placeholder. Matching
// on "${" is what the feed actually uses and is deliberately broader than a
// scan for known key names — a new provider's placeholder must be caught the
// day it lands, without a code change here.
func isTemplated(raw string) bool { return strings.Contains(raw, "${") }

// kindOf classifies a URL by scheme, returning "" for anything that is not an
// RPC transport we can speak (the feed also carries ipc:// paths and the odd
// malformed entry).
func kindOf(raw string) Kind {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return ""
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https":
		return KindHTTP
	case "ws", "wss":
		return KindWS
	}
	return ""
}

// probeAll probes every pending endpoint concurrently, bounded by
// Concurrency, and returns the list with each entry resolved. Order is
// preserved — results are written back into their own slot rather than
// appended as they land — because the feed's order is meaningful (the
// maintainers put the chain's own endpoint first) and a stable order keeps
// the rendered erpc.yaml diff-friendly.
func (d *Discoverer) probeAll(ctx context.Context, chainID int, eps []Endpoint) []Endpoint {
	sem := make(chan struct{}, d.concurrency())
	var wg sync.WaitGroup

	for i := range eps {
		if eps[i].Status != StatusPending {
			continue
		}
		if eps[i].Kind == KindWS && !d.ProbeWS {
			eps[i].Status = StatusUnprobed
			eps[i].Reason = ""
			continue
		}

		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			// Each probe gets its own deadline so one black-holed endpoint
			// costs one timeout, not the whole batch.
			pctx, cancel := context.WithTimeout(ctx, d.probeTimeout())
			defer cancel()

			start := time.Now()
			got, err := d.probe(pctx, eps[i])
			eps[i].Latency = time.Since(start)

			switch {
			case err != nil:
				eps[i].Status = StatusRejected
				eps[i].Reason = err.Error()
			case got != chainID:
				eps[i].ChainID = got
				eps[i].Status = StatusRejected
				eps[i].Reason = fmt.Sprintf("serves chain id %d, wanted %d", got, chainID)
			default:
				eps[i].ChainID = got
				eps[i].Status = StatusLive
			}
		}(i)
	}

	wg.Wait()
	return eps
}

// probe asks one endpoint for its chain id over its own transport.
func (d *Discoverer) probe(ctx context.Context, ep Endpoint) (int, error) {
	if ep.Kind == KindWS {
		return d.probeWS(ctx, ep.URL)
	}
	return d.probeHTTP(ctx, ep.URL)
}

// chainIDRequest is the probe payload: eth_chainId is the cheapest call that
// still proves identity, needs no arguments, and is served by every EVM node
// regardless of pruning or archive mode.
const chainIDRequest = `{"jsonrpc":"2.0","id":1,"method":"eth_chainId","params":[]}`

// rpcResponse is the sliver of a JSON-RPC response we read back.
type rpcResponse struct {
	Result string `json:"result"`
	Error  *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func (d *Discoverer) probeHTTP(ctx context.Context, rawURL string) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, rawURL, strings.NewReader(chainIDRequest))
	if err != nil {
		return 0, fmt.Errorf("bad URL: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := d.httpClient().Do(req)
	if err != nil {
		return 0, probeErr(ctx, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("HTTP %s", resp.Status)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxProbeBytes))
	if err != nil {
		return 0, probeErr(ctx, err)
	}
	return parseChainID(body)
}

// probeErr turns a transport error into an operator-facing reason, calling a
// timeout a timeout instead of leaking the "context deadline exceeded"
// plumbing into the UI.
func probeErr(ctx context.Context, err error) error {
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return errors.New("timed out")
	}
	if errors.Is(err, context.Canceled) || errors.Is(ctx.Err(), context.Canceled) {
		return errors.New("cancelled")
	}
	// url.Error's prefix ("Post \"https://…\": ") repeats the URL the caller
	// already has; unwrap one layer for a tighter message.
	var uerr *url.Error
	if errors.As(err, &uerr) && uerr.Err != nil {
		return uerr.Err
	}
	return err
}

// parseChainID pulls the chain id out of an eth_chainId response. The result
// is a JSON-RPC quantity ("0x171"), but a handful of endpoints answer in
// decimal, so ParseUint's base-0 detection is used rather than assuming hex.
func parseChainID(body []byte) (int, error) {
	var resp rpcResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return 0, fmt.Errorf("malformed JSON-RPC response: %w", err)
	}
	if resp.Error != nil {
		return 0, fmt.Errorf("JSON-RPC error %d: %s", resp.Error.Code, resp.Error.Message)
	}
	if resp.Result == "" {
		return 0, errors.New("empty eth_chainId result")
	}
	n, err := strconv.ParseUint(resp.Result, 0, 64)
	if err != nil {
		return 0, fmt.Errorf("uninterpretable eth_chainId result %q", resp.Result)
	}
	return int(n), nil
}

// ---------------------------------------------------------------------------
// WebSocket probing
//
// The RFC 6455 client used to live here, in a copy it shared with
// internal/capabilities and internal/setup. It is internal/wsrpc now.
// ---------------------------------------------------------------------------

// probeWS asks eth_chainId over a WebSocket. A refusal and a silence are both
// just errors to a discoverer that only wants to know whether the endpoint
// works — wsrpc distinguishes them for callers that report the difference.
func (d *Discoverer) probeWS(ctx context.Context, rawURL string) (int, error) {
	payload, err := wsrpc.RoundTrip(ctx, rawURL, []byte(chainIDRequest), &wsrpc.Options{Dialer: d.dialer()})
	if err != nil {
		return 0, probeErr(ctx, err)
	}
	return parseChainID(payload)
}
