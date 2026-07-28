package server

// GET /api/gateways/{gid}/traffic — who is actually carrying the load.
//
// This is a separate route rather than another field on the gateway view, and
// the reason is failure isolation. Reading it runs a command on the gateway's
// machine; folding that into the list handler would put a per-gateway round
// trip (and a per-gateway way to hang) behind the one screen an operator opens
// precisely when a gateway is misbehaving. The list stays cheap and always
// answers; the traffic numbers arrive separately and a gateway whose counters
// cannot be read loses its share bars, not its card.
//
// Nothing here is cached. The counters are cumulative, the scrape is one curl,
// and the screen decides its own cadence — a cache would add a second notion of
// "now" for numbers whose whole job is to be current.

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/valve-tech/valve-node-app/internal/catalog"
	"github.com/valve-tech/valve-node-app/internal/metrics"
	"github.com/valve-tech/valve-node-app/internal/setup"
)

// trafficTimeout bounds the whole request, including the executor round trip to
// a possibly-unreachable SSH target.
const trafficTimeout = 10 * time.Second

// upstreamShareView is one endpoint's row in the share column.
//
// Both numbers are on the wire, not just the share, because a percentage with
// no count behind it cannot be sanity-checked: "100%" over four requests and
// "100%" over four million are the same bar and very different facts, and the
// UI needs to be able to say which it is looking at.
type upstreamShareView struct {
	Upstream  string  `json:"upstream"`
	Succeeded float64 `json:"succeeded"`
	// Actual and Intended are fractions in 0..1.
	Actual   float64 `json:"actual"`
	Intended float64 `json:"intended"`
	Diverged bool    `json:"diverged"`
	// Unconfigured marks an upstream eRPC is still counting that this gateway's
	// configuration no longer lists. It happens between editing a config and
	// re-creating the container, and it is worth showing rather than dropping:
	// an endpoint still taking traffic after you removed it is exactly the
	// moment someone needs to be told the change has not been applied yet.
	Unconfigured bool `json:"unconfigured,omitempty"`
}

// networkTrafficView is one chain's totals.
type networkTrafficView struct {
	ChainID int `json:"chainId"`
	// Received is what clients asked this chain for; Attributed is what was
	// answered by some upstream. The gap is failure, and reporting both is what
	// keeps a chain that is failing every call from looking identical to one
	// nobody has called.
	Received   float64 `json:"received"`
	Attributed float64 `json:"attributed"`
	// Unattributed is what this chain answered without calling any endpoint —
	// eRPC's cache. Reported separately so the received/attributed gap means
	// only one thing: requests that failed. Folding these into Attributed
	// would credit an endpoint with work it did not do; dropping them would
	// manufacture a failure count out of successful requests.
	Unattributed float64             `json:"unattributed"`
	Upstreams    []upstreamShareView `json:"upstreams"`
}

type trafficResponse struct {
	// Enabled is false when the operator turned the counters off. The UI needs
	// to tell that apart from "no traffic yet", which looks identical in the
	// numbers and is fixed by a completely different action.
	Enabled bool `json:"enabled"`
	// At is when this reading was taken; Since is when the gateway process
	// started, which is what these cumulative counters are counted from. A
	// share with no window on it invites being read as "right now".
	At       time.Time            `json:"at"`
	Since    time.Time            `json:"since"`
	Networks []networkTrafficView `json:"networks"`
	Error    string               `json:"error,omitempty"`
}

// handleGatewayTraffic scrapes one gateway and folds its counters against what
// the routing configuration intended.
//
// A failed scrape answers 200 with an Error rather than an HTTP error status,
// deliberately: the gateway itself may be perfectly healthy and merely
// unreadable (metrics off, container just restarted, a port collision), and the
// screen should render the gateway with an explained blank share column instead
// of an error banner implying the gateway is down.
func (s *Server) handleGatewayTraffic(w http.ResponseWriter, r *http.Request) {
	cfg, gw, ok := s.gateway(w, r)
	if !ok {
		return
	}
	resolved, _ := resolveGateway(cfg, gw)

	res := trafficResponse{Enabled: resolved.MetricsEnabled(), At: time.Now(), Networks: []networkTrafficView{}}

	ex, _, ok := s.gatewayExecutor(w, cfg, gw)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), trafficTimeout)
	defer cancel()

	traffic, err := setup.ReadGatewayTraffic(ctx, ex, resolved)
	if err != nil {
		if errors.Is(err, setup.ErrMetricsOff) {
			// Not an error to show in red. It is a setting, and the response
			// already says Enabled is false, which is the whole message.
			writeJSON(w, http.StatusOK, res)
			return
		}
		res.Error = err.Error()
		writeJSON(w, http.StatusOK, res)
		return
	}

	res.Since = traffic.Since
	res.Networks = trafficViews(resolved, traffic)
	writeJSON(w, http.StatusOK, res)
}

// trafficViews joins the measured counters to the configured intent, chain by
// chain.
//
// The join is over the CONFIGURED networks rather than the measured ones, so a
// chain that has never been called still gets a row of zeroes instead of
// vanishing. A gateway nobody has used yet should look like a gateway nobody
// has used yet, not like a gateway with no chains.
func trafficViews(g catalog.GatewayConfig, t metrics.Traffic) []networkTrafficView {
	measured := make(map[int]metrics.NetworkTraffic, len(t.Networks))
	for _, n := range t.Networks {
		measured[n.ChainID] = n
	}

	out := make([]networkTrafficView, 0, len(g.Networks))
	for _, n := range g.Networks {
		nt := measured[n.ChainID]
		intents := setup.IntentsFor(n)
		configured := make(map[string]bool, len(intents))
		for _, i := range intents {
			configured[i.Upstream] = true
		}

		nv := networkTrafficView{ChainID: n.ChainID, Received: nt.Received, Unattributed: nt.Unattributed}
		for _, sh := range metrics.Shares(nt, intents) {
			nv.Attributed += sh.Succeeded
			nv.Upstreams = append(nv.Upstreams, upstreamShareView{
				Upstream:     sh.Upstream,
				Succeeded:    sh.Succeeded,
				Actual:       sh.Actual,
				Intended:     sh.Intended,
				Diverged:     sh.Diverged,
				Unconfigured: !configured[sh.Upstream],
			})
		}
		out = append(out, nv)
	}
	return out
}
