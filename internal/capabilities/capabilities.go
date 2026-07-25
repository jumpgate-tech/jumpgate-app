// Package capabilities answers one question about an RPC endpoint: what can it
// actually DO?
//
// Almost no provider publishes a capability list, and the ones that do publish
// aspirations rather than facts. So capability is inferred from behaviour —
// duck typing. Send the cheapest representative call for a feature and read the
// reply: a result (or an error that proves the method ran) means the feature is
// there; a -32601 means it is not; no reply at all means we do not know. That
// third answer is a first-class outcome, not a rounding error — see Status.
//
// # Two sources, and why both are required
//
// The first source is valve.city: GET /v1/capabilities?chainId=N, the same API
// that renders the capability table on learn.valve.city/rpc. Consuming it means
// this app and the published documentation cannot drift: one prober, one method
// set, one set of classification rules. It also probes from a well-connected
// host with no CORS ceiling, so it sees public endpoints as the internet sees
// them rather than as this box's network happens to see them.
//
// The second source is the local Prober in this package. It is emphatically not
// a fallback for valve.city being down. valve.city can only probe endpoints it
// can reach, which excludes exactly the endpoints an operator cares most about:
// a managed node on a Tailscale address, a devnet on a private docker network,
// a validator behind a firewall, the node this very app just provisioned. For
// those, a local probe is the only source there will ever be. valve.city being
// unreachable is a second, lesser reason to have it.
//
// # Merging, and why local wins
//
// Merge keeps provenance per capability (Result.Origin), because "valve.city
// says this is an archive node" and "we asked it ourselves 30 seconds ago" are
// different claims and a UI must be able to say which one it is showing. When
// both sources have an opinion about the same endpoint, the local probe wins —
// but only for endpoints we could actually reach. First-hand and current beats
// second-hand and cached: valve.city caches for ten minutes and probes from
// somewhere else, whereas a local probe reflects this box, this network, this
// moment, which is the thing the operator is about to depend on. When the local
// probe could not reach an endpoint at all, its silence is evidence about our
// network rather than about the endpoint, so valve.city's answer is kept
// instead of being overwritten with our own ignorance.
//
// # One capability valve.city does not cover
//
// KeyWS — a real RFC 6455 handshake followed by eth_chainId over the resulting
// socket. It is load-bearing, not cosmetic. Measured on 2026-07-24: all four
// published wss:// endpoints for chain 943 fail the handshake outright while
// advertising WebSocket support, and eRPC infers a WebSocket upstream from the
// URL scheme alone, so an http:// upstream will happily serve eth_chainId over
// wss:// and then refuse eth_subscribe. Nothing short of opening the socket
// tells you which of those you have.
//
// Everything here is context-aware, and both the HTTP client and the valve.city
// base URL are injectable, so the tests in this package never touch the network.
package capabilities

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"sort"
	"strings"
)

// Status is the three-state verdict for one capability on one endpoint.
//
// StatusInconclusive is a real answer and must never be collapsed into one of
// the other two. A timeout is not a refusal: a rate-limited endpoint, a probe
// killed by a deadline, and a node that genuinely lacks debug_traceTransaction
// are three different situations, and only the last one justifies telling an
// operator "this node cannot do that".
type Status string

const (
	StatusSupported    Status = "supported"
	StatusUnsupported  Status = "unsupported"
	StatusInconclusive Status = "inconclusive"
)

// Origin records who formed an opinion, so the UI can attribute every cell.
type Origin string

const (
	// OriginValveCity means the verdict came from valve.city's server-side
	// probe — the same run that backs the published docs.
	OriginValveCity Origin = "valve.city"
	// OriginLocal means this process opened the socket itself.
	OriginLocal Origin = "local"
)

// Source records where the endpoint URL came from, mirroring valve.city's
// 'valve' | 'chainlist' | 'user' and adding SourceLocal for endpoints only this
// box knows about (a private node, a devnet, the node being provisioned).
type Source string

const (
	SourceValve     Source = "valve"
	SourceChainlist Source = "chainlist"
	SourceUser      Source = "user"
	SourceLocal     Source = "local"
)

// Capability keys. These are wire values shared with valve.city and with the
// learn.valve.city table; changing one is a contract change, not a rename.
const (
	KeyArchive   = "archive"
	KeyTrace     = "trace"
	KeyDebug     = "debug"
	KeyLogs      = "logs"
	KeyFilters   = "filters"
	KeyOtterscan = "otterscan"
	KeyTxpool    = "txpool"
	KeyMsgboard  = "msgboard"
	KeyBatch     = "batch"
	// KeyWS is ours alone; valve.city does not probe it. See the package doc.
	KeyWS = "ws"
)

// valveCityKeys is valve.city's method set, in its order. Kept byte-identical
// to CAPABILITY_KEYS in the monorepo's capabilities router so the two sources
// stay comparable cell for cell.
var valveCityKeys = []string{
	KeyArchive, KeyTrace, KeyDebug, KeyLogs, KeyFilters,
	KeyOtterscan, KeyTxpool, KeyMsgboard, KeyBatch,
}

// allKeys is what the local Prober produces: valve.city's set plus ws.
var allKeys = append(append([]string{}, valveCityKeys...), KeyWS)

// ValveCityKeys returns the capabilities valve.city probes, in its order.
func ValveCityKeys() []string { return append([]string(nil), valveCityKeys...) }

// Keys returns every capability this package can report, in display order.
func Keys() []string { return append([]string(nil), allKeys...) }

// labels are the column headings, matching the published table.
var labels = map[string]string{
	KeyArchive:   "Archive",
	KeyTrace:     "Trace",
	KeyDebug:     "Debug",
	KeyLogs:      "Logs",
	KeyFilters:   "Filters",
	KeyOtterscan: "Otterscan",
	KeyTxpool:    "Txpool",
	KeyMsgboard:  "MsgBoard",
	KeyBatch:     "Batch",
	KeyWS:        "WebSocket",
}

// help explains how a column is tested. It belongs to the column, not the cell,
// because it is identical for every endpoint; the per-cell explanation is
// Result.Detail. Wording is kept in step with the published table so an
// operator reading both sees the same sentences.
var help = map[string]string{
	KeyArchive:   `Historical state via eth_getBalance(addr, block 0x1). supported = returns state; unsupported = pruned/full node`,
	KeyTrace:     `trace_transaction (Parity/Erigon trace). supported = method exists; unsupported = -32601 not found`,
	KeyDebug:     `debug_traceTransaction (Geth debug). supported = method exists; unsupported = -32601 not found`,
	KeyLogs:      `eth_getLogs over a small range. supported = replies; unsupported = -32601 / rejected`,
	KeyFilters:   `eth_newBlockFilter. supported = filter id; unsupported = -32601`,
	KeyOtterscan: `ots_getApiLevel (Otterscan). supported = api level; unsupported = -32601`,
	KeyTxpool:    `txpool_status. supported = replies; unsupported = -32601`,
	KeyMsgboard:  `msgboard_status (Valve on-chain message board). supported = replies; unsupported = -32601 / absent`,
	KeyBatch:     `A 2-item JSON-RPC batch. supported = 2-element array reply; unsupported = not batched`,
	KeyWS:        `A real RFC 6455 handshake plus eth_chainId over the socket. supported = both succeed; unsupported = upgrade refused`,
}

// ReachHelp explains the reachability column.
const ReachHelp = `eth_chainId must return a well-formed JSON-RPC reply. supported = reachable + right chain; unsupported = unreachable; inconclusive = wrong chain`

// Label returns the display name for a capability key, falling back to the key
// itself so an unknown key from a newer valve.city still renders.
func Label(key string) string {
	if l, ok := labels[key]; ok {
		return l
	}
	return key
}

// Help returns the how-it-is-tested text for a capability key, or "" if unknown.
func Help(key string) string { return help[key] }

// Result is one capability verdict for one endpoint.
//
// Method and Detail exist so a UI never has to show a bare tick: Method is what
// we sent, Detail is what came back, in words. Together they let an operator
// argue with the verdict, which is the whole point of inferring capability from
// behaviour rather than from a provider's marketing page.
type Result struct {
	Status Status
	// Method is the JSON-RPC method used to decide this cell ("batch" and "ws"
	// for the two probes that are not a single method call).
	Method string
	// Detail is the observed evidence, phrased for an operator.
	Detail string
	// Origin says which source formed this opinion.
	Origin Origin
}

// Endpoint is one RPC URL and everything we learned about it.
type Endpoint struct {
	URL   string
	Label string
	// Source is where the URL came from, not who probed it; Origin is who
	// probed it.
	Source Source

	// Reachable means eth_chainId came back as a well-formed JSON-RPC reply —
	// including an error reply, which still proves something is listening and
	// speaking the protocol.
	Reachable bool
	// ChainOK is tri-state, mirroring valve.city's `boolean | null`: nil means
	// the endpoint answered but not with a usable chain id, so we decline to
	// claim it is either the right chain or the wrong one.
	ChainOK *bool
	// ChainID is the chain the endpoint reported, 0 if it never said.
	ChainID int
	// ReachDetail explains the reachability verdict in words.
	ReachDetail string
	// Origin says who established reachability.
	Origin Origin

	// Capabilities is keyed by the Key* constants. A missing key means nobody
	// formed an opinion (typically: the endpoint was unreachable, so the
	// capability probes were skipped) — which is different from a key present
	// with StatusInconclusive.
	Capabilities map[string]Result
}

// Cap returns the verdict for one capability, and whether anyone had one.
func (e Endpoint) Cap(key string) (Result, bool) {
	r, ok := e.Capabilities[key]
	return r, ok
}

// Matrix is a whole capability table for one chain.
type Matrix struct {
	ChainID int
	// CapabilityKeys is the column order to render.
	CapabilityKeys []string
	// ChainlistTotal and ChainlistProbed are valve.city's sampling note ("12 of
	// the 31 public endpoints chainlist.org lists"); both 0 when valve.city did
	// not report them, e.g. a locally-probed-only matrix.
	ChainlistTotal  int
	ChainlistProbed int
	// RemoteErr is why valve.city was not used, nil when it was. A non-nil
	// RemoteErr with a non-empty Endpoints is the normal degraded case, not a
	// failure: local probing carried the run.
	RemoteErr error
	Endpoints []Endpoint
}

// Endpoint returns the row for a URL, matched the way Merge matches (scheme,
// host and path, normalised).
func (m Matrix) Endpoint(rawURL string) (Endpoint, bool) {
	want := normalizeURL(rawURL)
	for _, ep := range m.Endpoints {
		if normalizeURL(ep.URL) == want {
			return ep, true
		}
	}
	return Endpoint{}, false
}

// Merge folds locally probed endpoints into a valve.city matrix, preserving
// per-capability provenance.
//
// The rules, in the order they matter:
//
//   - An endpoint only we know about is kept. This is the point of local
//     probing: a Tailscale address or a docker-network devnet is invisible to
//     valve.city, so its row exists only because we made it. Such rows are
//     placed first, matching the published table's habit of putting the
//     operator's own endpoint above the public ones.
//   - For an endpoint both sources saw, the local probe wins per capability —
//     it is first-hand and current, where valve.city's answer was formed
//     elsewhere and may be up to ten minutes stale.
//   - Unless the local probe could not reach it. Then our silence says
//     something about our network, not about the endpoint, and valve.city's
//     answer is kept rather than overwritten with our ignorance.
//   - Capabilities only one source has (in practice KeyWS, which valve.city
//     does not probe) are always taken from whoever has them, reachable or not:
//     an inconclusive ws verdict is still the only ws verdict on offer, and a
//     missing cell would read as "not applicable" rather than "unknown".
//
// remote may be a zero Matrix (valve.city unreachable); the result is then the
// local endpoints alone, with the column order derived from what was probed.
func Merge(remote Matrix, local []Endpoint) Matrix {
	out := Matrix{
		ChainID:         remote.ChainID,
		ChainlistTotal:  remote.ChainlistTotal,
		ChainlistProbed: remote.ChainlistProbed,
		RemoteErr:       remote.RemoteErr,
	}

	// Copy the remote rows so the caller's Matrix is never mutated in place;
	// the capability maps are shared structures and merging writes into them.
	merged := make([]Endpoint, len(remote.Endpoints))
	index := make(map[string]int, len(remote.Endpoints))
	for i, ep := range remote.Endpoints {
		merged[i] = cloneEndpoint(ep)
		index[normalizeURL(ep.URL)] = i
	}

	var localOnly []Endpoint
	for _, lep := range local {
		i, ok := index[normalizeURL(lep.URL)]
		if !ok {
			localOnly = append(localOnly, cloneEndpoint(lep))
			continue
		}
		merged[i] = mergeEndpoint(merged[i], lep)
	}

	out.Endpoints = append(localOnly, merged...)
	out.CapabilityKeys = mergeKeys(remote.CapabilityKeys, out.Endpoints)
	return out
}

// mergeEndpoint applies one local probe over one valve.city row.
func mergeEndpoint(remote, local Endpoint) Endpoint {
	ep := remote
	if local.Reachable {
		// We reached it, so our reachability facts are the current ones.
		ep.Reachable = local.Reachable
		ep.ChainOK = local.ChainOK
		ep.ChainID = local.ChainID
		ep.ReachDetail = local.ReachDetail
		ep.Origin = local.Origin
	}
	if local.Label != "" && ep.Label == "" {
		ep.Label = local.Label
	}
	if ep.Capabilities == nil {
		ep.Capabilities = make(map[string]Result, len(local.Capabilities))
	}
	for key, res := range local.Capabilities {
		if _, held := ep.Capabilities[key]; held && !local.Reachable {
			// valve.city has an opinion and we have nothing better; keep theirs.
			continue
		}
		ep.Capabilities[key] = res
	}
	return ep
}

// mergeKeys keeps valve.city's column order and appends any column only the
// local prober produced (KeyWS today), so a table never loses or reorders the
// columns an operator learned from the published docs.
func mergeKeys(remoteKeys []string, eps []Endpoint) []string {
	out := append([]string(nil), remoteKeys...)
	seen := make(map[string]bool, len(out))
	for _, k := range out {
		seen[k] = true
	}

	present := make(map[string]bool)
	for _, ep := range eps {
		for k := range ep.Capabilities {
			present[k] = true
		}
	}

	for _, k := range allKeys {
		if present[k] && !seen[k] {
			out = append(out, k)
			seen[k] = true
		}
	}
	// Anything a newer valve.city or a future probe invented, in a stable order.
	var extra []string
	for k := range present {
		if !seen[k] {
			extra = append(extra, k)
		}
	}
	sort.Strings(extra)
	return append(out, extra...)
}

func cloneEndpoint(ep Endpoint) Endpoint {
	out := ep
	if ep.ChainOK != nil {
		v := *ep.ChainOK
		out.ChainOK = &v
	}
	out.Capabilities = make(map[string]Result, len(ep.Capabilities))
	for k, v := range ep.Capabilities {
		out.Capabilities[k] = v
	}
	return out
}

// normalizeURL reduces a URL to the identity we merge on: scheme, host and
// path, lowercased, with the default port and a trailing slash dropped. The
// scheme is deliberately kept — https://node and wss://node are the same
// machine but not the same endpoint, and the table lists them as separate rows
// precisely because one can work while the other does not.
func normalizeURL(raw string) string {
	trimmed := strings.TrimSpace(raw)
	u, err := url.Parse(trimmed)
	if err != nil || u.Host == "" {
		return strings.ToLower(trimmed)
	}
	scheme := strings.ToLower(u.Scheme)
	host := strings.ToLower(u.Hostname())
	if port := u.Port(); port != "" && !isDefaultPort(scheme, port) {
		host = net.JoinHostPort(host, port)
	}
	return scheme + "://" + host + strings.TrimSuffix(u.EscapedPath(), "/")
}

func isDefaultPort(scheme, port string) bool {
	switch scheme {
	case "http", "ws":
		return port == "80"
	case "https", "wss":
		return port == "443"
	}
	return false
}

// Gatherer runs both sources and merges them. Either half may be nil, which
// disables that source — a caller probing only a private devnet has no use for
// valve.city, and a caller rendering the published table on a box with no
// outbound access to the endpoints has no use for the local prober.
type Gatherer struct {
	Client *Client
	Prober *Prober
}

// NewGatherer returns a Gatherer with both sources at their defaults.
func NewGatherer() *Gatherer {
	return &Gatherer{Client: NewClient(), Prober: NewProber()}
}

// Gather fetches valve.city's matrix for chainID, probes the local targets, and
// merges the two.
//
// It returns no error, by design. valve.city being unreachable is not a failure
// of this operation — it is one of two sources going quiet, recorded in
// Matrix.RemoteErr so a caller can say "chainlist's view is unavailable, these
// are the endpoints we checked ourselves" rather than showing nothing. A caller
// that genuinely needs the published matrix should check RemoteErr.
func (g *Gatherer) Gather(ctx context.Context, chainID int, local []Target) Matrix {
	var remote Matrix
	if g.Client != nil {
		m, err := g.Client.Matrix(ctx, chainID)
		if err != nil {
			remote = Matrix{ChainID: chainID, RemoteErr: fmt.Errorf("capabilities: valve.city unavailable: %w", err)}
		} else {
			remote = m
		}
	}

	var probed []Endpoint
	if g.Prober != nil && len(local) > 0 {
		probed = g.Prober.ProbeAll(ctx, local, chainID)
	}

	out := Merge(remote, probed)
	out.ChainID = chainID
	return out
}
