package server

// GET /api/gateways/{gid}/capabilities — what each upstream can actually DO.
//
// This is a separate route rather than a field folded into the gateway list
// or the traffic response, for the same failure-isolation reason traffic.go
// gives: probing capabilities means opening real sockets and sending real
// JSON-RPC calls to every upstream a gateway fronts, roughly a dozen calls
// each (see internal/capabilities). Folding that into a screen that has to
// answer instantly would put "the whole list waits on the slowest, or
// least reachable, upstream" behind the one screen an operator opens to see
// whether a gateway is basically fine.
//
// Unlike traffic.go, this route IS cached (capabilitiesTTL, ten minutes).
// Traffic is a single cheap curl against a metrics port the gateway itself
// already serves; a capability probe is this process personally dialing N
// upstreams with ~a dozen calls apiece, several of which (archive, trace,
// debug) are exactly the calls an operator would NOT want run on a poll
// cadence against a production node. Caching is what lets the RPC screen
// show a capability table on every load without turning every load into a
// load-generation run. ?refresh=1 bypasses the cache when the operator
// deliberately wants a fresh read — after fixing an endpoint, say.
//
// A probe failure never fails the whole request: the response is always 200,
// with the reason recorded on the affected endpoint. A gateway can be
// perfectly healthy with one unprobeable or unreachable upstream, and an
// HTTP error here would make the whole capability table vanish over that one
// row — see handleGatewayTraffic and handleGatewayTLSVerify for the same
// rule applied to their own reads.

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/valve-tech/valve-node-app/internal/capabilities"
	"github.com/valve-tech/valve-node-app/internal/catalog"
	"github.com/valve-tech/valve-node-app/internal/config"
	"github.com/valve-tech/valve-node-app/internal/setup"
)

// capabilitiesTimeout bounds the whole request. It is generous relative to
// tlsVerifyTimeout because this handler is not probing one front, it is
// probing every endpoint under every chain the gateway serves — each of
// which pays roughly a dozen JSON-RPC round trips (capacitated by the
// Prober's own Concurrency, not run serially) — so a gateway with several
// chains genuinely needs more wall clock than a single HTTPS check does.
const capabilitiesTimeout = 60 * time.Second

// capabilitiesTTL is how long a cached capability probe is served before the
// next request re-probes. Ten minutes matches the order of magnitude
// valve.city itself caches its own published matrix for (see
// internal/capabilities' package doc), so a capability table here is never
// staler, by more than a coincidence of timing, than the one an operator
// could look up on learn.valve.city.
const capabilitiesTTL = 10 * time.Minute

// Probe repetition counts. See probeGatewayCapabilities for where each is
// used and capabilities.ProbeRepeat's doc for why the distinction exists at
// all: repetition exists to catch a load-balanced public endpoint disagreeing
// with itself, which is a property ONLY an endpoint backed by more than one
// process can have.
const (
	// externalProbeRepeat: a public endpoint's hostname commonly fronts
	// several backend processes behind a load balancer, and those backends
	// can and do disagree about what they support — see
	// capabilities.StatusInconsistent's doc comment for the measurement
	// (msgboard_status on rpc.pulsechain.com, 5 supported vs 10 × -32601
	// across 15 probes to one DNS name). Three repeats is enough to catch
	// that disagreement without tripling the run's wall clock for nothing.
	externalProbeRepeat = 3
	// managedProbeRepeat: a managed node or devnet is ONE process this app
	// itself placed. It cannot disagree with itself, so a second or third
	// probe would only be three times the load on a box we are responsible
	// for, for zero additional information.
	managedProbeRepeat = 1
)

// ---------------------------------------------------------------------
// wire shapes
// ---------------------------------------------------------------------

// capabilityView is one capability verdict for one endpoint — a cell in the
// table.
type capabilityView struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	// Status is one of capabilities.Status* ("supported", "unsupported",
	// "inconclusive", "inconsistent"), kept as the package's own vocabulary
	// on the wire rather than collapsed to a boolean — see that type's doc
	// for why inconclusive and inconsistent are both first-class answers,
	// not rounding error.
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`
	Method string `json:"method,omitempty"`
}

// endpointCapabilitiesView is one upstream's whole row.
type endpointCapabilitiesView struct {
	// Upstream is the SAME id the traffic route reports for this upstream —
	// see probeGatewayCapabilities for how that is guaranteed. The UI joins
	// the two responses on this string, so a divergent derivation here would
	// silently produce two disjoint sets of rows instead of one table.
	Upstream string `json:"upstream"`
	ChainID  int    `json:"chainId"`

	// ProbedURL is the address THIS PROCESS dialed, which is frequently not
	// the address eRPC itself dials — see probeAddressFor. Empty when
	// Unprobeable is set, because then nothing was dialed at all.
	ProbedURL string `json:"probedUrl,omitempty"`

	Reachable   bool   `json:"reachable"`
	ReachDetail string `json:"reachDetail,omitempty"`

	// Unprobeable is set, and Reachable left false, when this process could
	// not even ATTEMPT to reach the endpoint — as opposed to attempting and
	// failing. The distinction matters on the screen: "unreachable" invites
	// "is the node down?", where this row's real answer is "you cannot ask
	// from here; ask from the machine that hosts it."
	Unprobeable string `json:"unprobeable,omitempty"`

	// Capabilities always carries one entry per capabilities.Keys(), even
	// when the endpoint was unreachable or unprobeable — see
	// capabilityViewsFor and unprobedCapabilityViews. A short array would
	// make "never asked" indistinguishable from "asked and got nothing".
	Capabilities []capabilityView `json:"capabilities"`
}

type capabilitiesResponse struct {
	At        time.Time                  `json:"at"`
	Endpoints []endpointCapabilitiesView `json:"endpoints"`
}

// ---------------------------------------------------------------------
// GET /api/gateways/{gid}/capabilities
// ---------------------------------------------------------------------

func (s *Server) handleGatewayCapabilities(w http.ResponseWriter, r *http.Request) {
	cfg, gw, ok := s.gateway(w, r)
	if !ok {
		return
	}

	if r.URL.Query().Get("refresh") != "1" {
		if cached, hit := s.cachedCapabilities(gw.ID); hit {
			writeJSON(w, http.StatusOK, cached)
			return
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), capabilitiesTimeout)
	defer cancel()

	res := probeGatewayCapabilities(ctx, capabilities.NewProber(), cfg, gw)
	s.storeCapabilities(gw.ID, res)
	writeJSON(w, http.StatusOK, res)
}

// cachedCapabilities / storeCapabilities keep the last capability probe per
// gateway id, guarded by capMu — mirroring storeTLSVerification /
// lastTLSVerification, with the one difference that this cache DOES expire
// (capabilitiesTTL). See the field comment on Server.capChecks for why.
func (s *Server) cachedCapabilities(gid string) (capabilitiesResponse, bool) {
	s.capMu.Lock()
	defer s.capMu.Unlock()
	res, ok := s.capChecks[gid]
	if !ok || time.Since(res.At) > capabilitiesTTL {
		return capabilitiesResponse{}, false
	}
	return res, true
}

func (s *Server) storeCapabilities(gid string, res capabilitiesResponse) {
	s.capMu.Lock()
	defer s.capMu.Unlock()
	if s.capChecks == nil {
		s.capChecks = map[string]capabilitiesResponse{}
	}
	s.capChecks[gid] = res
}

// ---------------------------------------------------------------------
// assembly
// ---------------------------------------------------------------------

// probeGatewayCapabilities probes every upstream of every chain the gateway
// serves and assembles the response. It takes the Prober as a parameter,
// rather than constructing one internally, purely so tests can pass one
// configured for a fast, network-free run (a short ProbeTimeout, ProbeWS
// off) without the handler needing any injection seam of its own.
//
// It walks resolveGateway's RESOLVED networks, not gw.Config.Networks
// directly, and that choice is what keeps this route's upstream ids
// identical to the traffic route's: resolveGateway is the same function
// handleGatewayTraffic feeds to setup.IntentsFor, dropped upstreams and all
// (a dead reference, or one eRPC itself cannot reach — see
// reachableAcrossMachines). An upstream that never made it into the running
// eRPC config carries no traffic counters to join against, so giving it a
// capabilities row here would only produce a row the UI can never pair with
// anything.
func probeGatewayCapabilities(ctx context.Context, prober *capabilities.Prober, cfg config.Config, gw config.Gateway) capabilitiesResponse {
	resolved, _ := resolveGateway(cfg, gw)

	type job struct {
		id, probeURL, unprobeable string
		chainID, repeat           int
	}
	var jobs []job
	for _, n := range resolved.Networks {
		// setup.IntentsFor is the SAME function trafficViews calls, over the
		// SAME resolved network, so intents[i].Upstream is byte-identical to
		// the id the traffic route reports for n.Upstreams[i] — see
		// TestCapabilities_UpstreamIDsMatchIntentsFor.
		intents := setup.IntentsFor(n)
		for i, u := range n.Upstreams {
			probeURL, unprobeable := probeAddressFor(cfg, gw, u)
			repeat := managedProbeRepeat
			if u.KindOrDefault() == catalog.UpstreamExternal {
				repeat = externalProbeRepeat
			}
			jobs = append(jobs, job{
				id:          intents[i].Upstream,
				probeURL:    probeURL,
				unprobeable: unprobeable,
				chainID:     n.ChainID,
				repeat:      repeat,
			})
		}
	}

	// Probed concurrently: the Prober already caps in-flight round trips via
	// its own Concurrency, so serialising the endpoints here would only add a
	// second, redundant limit — one that scales with chain count instead of
	// with real socket pressure.
	views := make([]endpointCapabilitiesView, len(jobs))
	var wg sync.WaitGroup
	for i, j := range jobs {
		wg.Add(1)
		go func(i int, j job) {
			defer wg.Done()
			views[i] = probeOneUpstream(ctx, prober, j.id, j.chainID, j.probeURL, j.unprobeable, j.repeat)
		}(i, j)
	}
	wg.Wait()

	return capabilitiesResponse{At: time.Now(), Endpoints: views}
}

// probeOneUpstream runs (or skips) one endpoint's probe and renders its row.
func probeOneUpstream(ctx context.Context, prober *capabilities.Prober, upstream string, chainID int, probeURL, unprobeable string, repeat int) endpointCapabilitiesView {
	v := endpointCapabilitiesView{Upstream: upstream, ChainID: chainID}
	if unprobeable != "" {
		// Never dialed, deliberately — see probeAddressFor and the
		// Unprobeable field doc.
		v.Unprobeable = unprobeable
		v.Capabilities = unprobedCapabilityViews("not probed: " + unprobeable)
		return v
	}

	v.ProbedURL = probeURL
	ep := prober.ProbeRepeat(ctx, capabilities.Target{URL: probeURL}, chainID, repeat)
	v.Reachable = ep.Reachable
	v.ReachDetail = ep.ReachDetail
	v.Capabilities = capabilityViewsFor(ep)
	return v
}

// capabilityViewsFor renders one probed Endpoint's full capability row,
// emitting every key capabilities.Keys() knows about — including the ones
// the endpoint's Capabilities map has no entry for, which happens whenever
// it was unreachable (Probe skips the per-capability calls entirely once
// eth_chainId fails, see capabilities.Probe). A short array here would make
// "we never asked" and "we asked and it said no" the same shape on the wire.
func capabilityViewsFor(ep capabilities.Endpoint) []capabilityView {
	keys := capabilities.Keys()
	out := make([]capabilityView, 0, len(keys))
	for _, key := range keys {
		if res, ok := ep.Cap(key); ok {
			out = append(out, capabilityView{
				Key: key, Label: capabilities.Label(key),
				Status: string(res.Status), Detail: res.Detail, Method: res.Method,
			})
			continue
		}
		detail := "no verdict"
		if !ep.Reachable {
			detail = "endpoint was unreachable, so this capability was never probed"
		}
		out = append(out, capabilityView{
			Key: key, Label: capabilities.Label(key),
			Status: string(capabilities.StatusInconclusive), Detail: detail,
		})
	}
	return out
}

// unprobedCapabilityViews is capabilityViewsFor's counterpart for an endpoint
// this process never dialed at all — see the Unprobeable field doc. Every
// key still gets a row, all inconclusive, all carrying the SAME reason the
// endpoint itself carries, so a cell and its row tell the same story.
func unprobedCapabilityViews(reason string) []capabilityView {
	keys := capabilities.Keys()
	out := make([]capabilityView, 0, len(keys))
	for _, key := range keys {
		out = append(out, capabilityView{
			Key: key, Label: capabilities.Label(key),
			Status: string(capabilities.StatusInconclusive), Detail: reason,
		})
	}
	return out
}

// ---------------------------------------------------------------------
// probeAddressFor — the crux: the dial address is not the probe address
// ---------------------------------------------------------------------

// probeAddressFor decides the URL THIS PROCESS can dial to probe one
// upstream, and returns a non-empty unprobeable reason instead whenever no
// such URL exists.
//
// It is deliberately a SEPARATE derivation from resolveUpstream, not a
// wrapper around it, because the two functions answer different questions
// from different vantage points. resolveUpstream answers "what URL does
// eRPC, running on the GATEWAY's placement machine, dial?" — and that
// machine may be exactly where the upstream lives, in which case an address
// that machine's own loopback interface serves is perfectly fine for eRPC to
// use. probeAddressFor answers "what URL can THIS PROCESS — the
// valve-node-app instance answering this very HTTP request — dial?", and
// this process is not necessarily anywhere near the gateway's machine at
// all: it is wherever the operator is running valve-node-app, commonly their
// own laptop, reaching fleet boxes over SSH. Two concrete consequences:
//
//   - A managed devnet is ALWAYS probed at its PUBLISHED address
//     (catalog.DevnetConfig.WSEndpoint), never at the container-name form
//     resolveUpstream hands to a same-host docker gateway
//     (ws://valve-node-app-devnet:8546). That name resolves inside the
//     docker engine's own embedded DNS and NOWHERE else — least of all on
//     this process's host, which is not a container on that network however
//     "same host" the gateway and devnet are to EACH OTHER. The scheme stays
//     ws:// regardless: eRPC infers WebSocket capability from the upstream
//     scheme alone, a ws upstream serves ordinary request/response calls
//     too, and — measured on this devnet's own defaults — the HTTP and WS
//     ports differ (8600 vs 8601), so deriving a ws:// URL from the HTTP
//     endpoint via capabilities.DeriveWSURL would silently probe the WRONG
//     PORT and report a false "WebSocket unsupported".
//
//   - A managed upstream (node or devnet) whose TARGET is reached over SSH,
//     and whose resolved address is loopback, is unprobeable from here even
//     when it is perfectly fine for eRPC to use. This is the situation
//     reachableAcrossMachines does NOT catch: that check compares the
//     upstream's target against the GATEWAY's placement, and is satisfied
//     the moment they are the same machine — correctly, because eRPC really
//     is running right there. But this process is running somewhere else
//     again, over yet another SSH hop (or none, if the target and this
//     process happen to coincide — see below), and 127.0.0.1 on a machine
//     reached over SSH is that machine's own loopback, not a hop this
//     process can take. A LOCAL target (Mode == "local") is the one case
//     this does not apply to: "local" names the machine this very process
//     is running on, so its loopback is our own.
//
// A dead reference (a target no longer registered, a devnet or node that was
// removed) is reported unprobeable too, with the same wording resolveUpstream
// itself would use for the equivalent gateway-side failure — there being
// nothing else useful to say about an upstream nothing can dial from
// anywhere.
func probeAddressFor(cfg config.Config, gw config.Gateway, u catalog.GatewayUpstream) (probeURL, unprobeable string) {
	switch u.KindOrDefault() {
	case catalog.UpstreamExternal:
		e := strings.TrimSpace(u.Endpoint)
		if e == "" {
			return "", "this upstream has no endpoint configured"
		}
		// An external endpoint names no target of ours, so there is no
		// second vantage point to reason about: whatever address the
		// operator gave is the one we probe, exactly as eRPC would.
		return e, ""

	case catalog.UpstreamManagedDevnet:
		t, ok := findTarget(cfg, u.TargetID)
		if !ok {
			return "", fmt.Sprintf("machine %q is no longer registered", u.TargetID)
		}
		if t.Devnet == nil {
			return "", fmt.Sprintf("machine %q has no devnet configured any more", u.TargetID)
		}
		d := resolvedDevnet(t.Devnet)
		return probeAddressAcrossSSH(t, d.WSEndpoint())

	case catalog.UpstreamManagedNode:
		t, ok := findTarget(cfg, u.TargetID)
		if !ok {
			return "", fmt.Sprintf("machine %q is no longer registered", u.TargetID)
		}
		if t.Wire == nil {
			return "", fmt.Sprintf("machine %q has no node set up any more", u.TargetID)
		}
		e := fmt.Sprintf("http://%s:%d", endpointHost(t.Wire.RPCBind()), t.Wire.ExecHTTP())
		return probeAddressAcrossSSH(t, e)

	default:
		return "", fmt.Sprintf("unknown upstream kind %q", u.Kind)
	}
}

// probeAddressAcrossSSH applies the SSH-loopback rule that is the second
// half of probeAddressFor's doc comment: candidate is fine to probe from
// here UNLESS t is reached over SSH and candidate names a loopback address,
// in which case that address belongs to t's own network namespace and this
// process — reaching t over yet another hop — cannot dial it.
func probeAddressAcrossSSH(t config.Target, candidate string) (probeURL, unprobeable string) {
	if t.Mode != "ssh" {
		return candidate, ""
	}
	u, err := url.Parse(candidate)
	if err != nil || !isLoopbackHost(u.Hostname()) {
		return candidate, ""
	}
	return "", fmt.Sprintf("this endpoint is on machine %q and bound to loopback, so it is only reachable from there", t.ID)
}
