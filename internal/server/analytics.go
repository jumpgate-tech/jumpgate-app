package server

// GET /api/gateways/{gid}/analytics — how is it doing, and why.
//
// The Control Surface answers detection: what can this endpoint do, and is it
// carrying the share you intended. This answers the question you ask NEXT,
// once that screen has told you something is off. The two are separate routes
// for the same reason traffic is separate from the gateway list — a per-gateway
// round trip must not be able to hang the screen that lists them — and because
// they are opened at different moments and at different cadences.
//
// One scrape, folded twice. The dump carries the counters both views need, so
// this route reads it once (setup.ReadGatewaySamples) and hands the samples to
// metrics.FromSamples and metrics.AnalyticsFromSamples in turn. A page that
// wants volume alongside latency therefore costs one curl, not two.
//
// Nothing is cached, exactly as traffic.go is not: these are cumulative
// counters read from a live process, and a cache would introduce a second
// notion of "now" for numbers whose entire job is to be current.

import (
	"context"
	"errors"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/valve-tech/valve-node-app/internal/catalog"
	"github.com/valve-tech/valve-node-app/internal/metrics"
	"github.com/valve-tech/valve-node-app/internal/setup"
)

// bucketView is one cumulative histogram bucket.
//
// le is a STRING, and that is not a stylistic choice: the final bucket's bound
// is +Inf, which encoding/json refuses to marshal outright ("unsupported
// value: +Inf"), taking the whole response down with it. Prometheus itself
// spells the label "+Inf", so sending it verbatim is both encodable and the
// spelling anyone who has read the raw dump already recognises.
type bucketView struct {
	LE    string  `json:"le"`
	Count float64 `json:"count"`
}

// latencyView is one histogram on the wire.
//
// mean is a POINTER so it can be null. A mean of zero and "nothing has been
// counted" are different facts, and 0 is the worse lie of the two: it claims
// the fastest possible response time for a method that has never answered
// anything. null renders as "—"; 0 renders as "0ms".
type latencyView struct {
	Count   float64      `json:"count"`
	Mean    *float64     `json:"mean"`
	Buckets []bucketView `json:"buckets"`
}

type methodLatencyView struct {
	Method string `json:"method"`
	latencyView
}

type endpointLatencyView struct {
	Upstream string `json:"upstream"`
	latencyView
}

// networkAnalyticsView is one chain's client-facing picture: how much was
// asked of it, how much came back, and how long it took.
type networkAnalyticsView struct {
	ChainID int    `json:"chainId"`
	Name    string `json:"name"`

	// Received, Answered and Failed come from the traffic fold rather than the
	// latency one. Failed is the arithmetic the screen would otherwise have to
	// do for itself — received minus everything that came back, whether from an
	// endpoint or from the gateway's own cache — and doing it here means one
	// definition of "failed" instead of one per caller.
	Received     float64 `json:"received"`
	Answered     float64 `json:"answered"`
	Unattributed float64 `json:"unattributed"`
	Failed       float64 `json:"failed"`

	Methods   []methodLatencyView   `json:"methods"`
	Endpoints []endpointLatencyView `json:"endpoints"`
}

// errorClassView is one class of error one endpoint has returned.
type errorClassView struct {
	Class    string  `json:"class"`
	Severity string  `json:"severity"`
	Method   string  `json:"method"`
	Count    float64 `json:"count"`
}

// endpointHealthView is what the GATEWAY sees of one endpoint — not what
// clients saw. Every field here counts eRPC's own state poller alongside
// client traffic, because eRPC publishes no label that separates them, and the
// screen says so where it renders them.
type endpointHealthView struct {
	Upstream string `json:"upstream"`
	ChainID  int    `json:"chainId"`
	// Configured is false for an endpoint eRPC is still reporting on that this
	// gateway's saved configuration no longer lists — the same state the share
	// column calls "not in config", surfaced here for the same reason.
	Configured bool `json:"configured"`

	// Requests includes the state poller and is usually mostly the state
	// poller. Named for what it holds.
	Requests float64          `json:"requests"`
	Errors   []errorClassView `json:"errors"`

	Score           float64 `json:"score"`
	Position        float64 `json:"position"`
	PrimarySwitches float64 `json:"primarySwitches"`
	ExcludedSeconds float64 `json:"excludedSeconds"`

	HeadLag         float64 `json:"headLag"`
	FinalizationLag float64 `json:"finalizationLag"`
	LatestBlock     float64 `json:"latestBlock"`
}

type analyticsResponse struct {
	// Enabled is the operator's counter setting, reported whatever happened to
	// the scrape — "turned off" and "could not be read" are different problems
	// with different fixes.
	Enabled bool      `json:"enabled"`
	At      time.Time `json:"at"`
	Since   time.Time `json:"since"`

	Networks  []networkAnalyticsView `json:"networks"`
	Endpoints []endpointHealthView   `json:"endpoints"`
	Error     string                 `json:"error,omitempty"`
}

// handleGatewayAnalytics scrapes one gateway and returns both folds of the
// reading.
//
// A failed scrape answers 200 with Error set, exactly as traffic does: the
// gateway may be perfectly healthy and merely unreadable, and an HTTP error
// status would tell the screen it is down.
func (s *Server) handleGatewayAnalytics(w http.ResponseWriter, r *http.Request) {
	cfg, gw, ok := s.gateway(w, r)
	if !ok {
		return
	}
	resolved, _ := resolveGateway(cfg, gw)

	res := analyticsResponse{
		Enabled:   resolved.MetricsEnabled(),
		At:        time.Now(),
		Networks:  []networkAnalyticsView{},
		Endpoints: []endpointHealthView{},
	}

	ex, _, ok := s.gatewayExecutor(w, cfg, gw)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), trafficTimeout)
	defer cancel()

	samples, err := setup.ReadGatewaySamples(ctx, ex, resolved)
	if err != nil {
		if !errors.Is(err, setup.ErrMetricsOff) {
			// Metrics being off is a setting, and Enabled:false already says
			// so; an error string here would paint the screen red over a
			// checkbox.
			res.Error = err.Error()
		}
		writeJSON(w, http.StatusOK, res)
		return
	}

	project := resolved.ProjectIDOrDefault()
	traffic := metrics.FromSamples(samples, project)
	analytics := metrics.AnalyticsFromSamples(samples, project)

	res.Since = analytics.Since
	res.Networks = networkAnalyticsViews(resolved, traffic, analytics)
	res.Endpoints = endpointHealthViews(resolved, analytics)
	writeJSON(w, http.StatusOK, res)
}

// networkAnalyticsViews joins the two folds chain by chain, over the CONFIGURED
// networks — so a chain nobody has called still gets a row of zeroes rather
// than vanishing, which is the same rule trafficViews follows and for the same
// reason: a gateway nobody has used should look like one, not like a gateway
// with no chains.
func networkAnalyticsViews(g catalog.GatewayConfig, t metrics.Traffic, a metrics.Analytics) []networkAnalyticsView {
	traffic := make(map[int]metrics.NetworkTraffic, len(t.Networks))
	for _, n := range t.Networks {
		traffic[n.ChainID] = n
	}
	latency := make(map[int]metrics.NetworkAnalytics, len(a.Networks))
	for _, n := range a.Networks {
		latency[n.ChainID] = n
	}

	out := make([]networkAnalyticsView, 0, len(g.Networks))
	for _, n := range g.Networks {
		nt, la := traffic[n.ChainID], latency[n.ChainID]

		var answered float64
		for _, u := range nt.Upstreams {
			answered += u.Succeeded
		}

		nv := networkAnalyticsView{
			ChainID:      n.ChainID,
			Name:         chainName(n.ChainID),
			Received:     nt.Received,
			Answered:     answered,
			Unattributed: nt.Unattributed,
			// Clamped at zero. The counters are read in one scrape but are not
			// written atomically, so a request counted as received a
			// microsecond before its success lands can make the sum exceed it
			// by one — and a negative failure count is a number that cannot be
			// true, which costs more trust than the rounding it exposes.
			Failed:    math.Max(0, nt.Received-answered-nt.Unattributed),
			Methods:   []methodLatencyView{},
			Endpoints: []endpointLatencyView{},
		}
		for _, m := range la.Methods {
			nv.Methods = append(nv.Methods, methodLatencyView{Method: m.Method, latencyView: latencyOf(m.Latency)})
		}
		for _, e := range la.Endpoints {
			nv.Endpoints = append(nv.Endpoints, endpointLatencyView{Upstream: e.Upstream, latencyView: latencyOf(e.Latency)})
		}
		out = append(out, nv)
	}
	return out
}

// endpointHealthViews renders every endpoint the gateway has an opinion about,
// including ones the saved configuration no longer lists — see Configured.
func endpointHealthViews(g catalog.GatewayConfig, a metrics.Analytics) []endpointHealthView {
	configured := map[string]bool{}
	for _, n := range g.Networks {
		for _, i := range setup.IntentsFor(n) {
			configured[i.Upstream] = true
		}
	}

	out := make([]endpointHealthView, 0, len(a.Endpoints))
	for _, e := range a.Endpoints {
		ev := endpointHealthView{
			Upstream:        e.Upstream,
			ChainID:         e.ChainID,
			Configured:      configured[e.Upstream],
			Requests:        e.Requests,
			Errors:          []errorClassView{},
			Score:           e.Score,
			Position:        e.Position,
			PrimarySwitches: e.PrimarySwitches,
			ExcludedSeconds: e.ExcludedSeconds,
			HeadLag:         e.HeadLag,
			FinalizationLag: e.FinalizationLag,
			LatestBlock:     e.LatestBlock,
		}
		for _, c := range e.Errors {
			ev.Errors = append(ev.Errors, errorClassView{Class: c.Class, Severity: c.Severity, Method: c.Method, Count: c.Count})
		}
		out = append(out, ev)
	}
	return out
}

// latencyOf puts one histogram on the wire, dropping the NaN mean of a
// never-called method to null rather than marshalling a value JSON cannot
// carry.
func latencyOf(l metrics.Latency) latencyView {
	v := latencyView{Count: l.Count, Buckets: []bucketView{}}
	if m := l.Mean(); !math.IsNaN(m) && !math.IsInf(m, 0) {
		v.Mean = &m
	}
	for _, b := range l.Buckets {
		v.Buckets = append(v.Buckets, bucketView{LE: formatBound(b.LE), Count: b.Count})
	}
	return v
}

// formatBound spells a bucket bound as a string. strconv.FormatFloat already
// renders +Inf as the literal "+Inf" — the same spelling Prometheus uses in
// the `le` label this came from — so no special case is needed here, and one
// written anyway would be a branch no test could ever observe.
//
// The string is not cosmetic: encoding/json refuses +Inf outright ("json:
// unsupported value: +Inf"), which would fail the marshal and take the entire
// response down over one histogram's last bucket.
func formatBound(le float64) string {
	return strconv.FormatFloat(le, 'g', -1, 64)
}
