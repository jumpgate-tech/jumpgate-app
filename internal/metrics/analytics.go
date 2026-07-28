package metrics

import (
	"math"
	"sort"
	"strconv"
	"time"
)

// Analytics is the diagnosis half of a gateway's counters, to Traffic's
// detection half: not "is this endpoint carrying the share I intended" but
// "how is it doing, and why is eRPC choosing what it chooses".
//
// It is split into two fields, and the split is the most important thing in
// this file. Networks holds what CLIENTS experienced; Endpoints holds what the
// GATEWAY sees of its upstreams. They come from different metric families,
// they are counted on different code paths, and averaging one into the other
// produces a number that describes nothing.
//
// # Why the two cannot be merged
//
// Only the erpc_network_* family is client-facing. eRPC's state poller — the
// loop that keeps each upstream's latest/finalized block fresh on a timer,
// whether or not anyone has called anything — bypasses network-level request
// handling entirely, so it never touches that family. Every erpc_upstream_*
// family, on the other hand, counts the poller and the clients together with
// no label that separates them.
//
// MEASURED on this package's own testdata/erpc.txt, a real dump: five client
// requests had been made (eth_blockNumber on evm:369). In the same dump,
// erpc_upstream_request_duration_seconds records 413 eth_getBlockByNumber
// calls to public-1-1 labelled network="evm:1" — a chain that appears in the
// client-facing family not once, because no client ever called it. The poller
// wearing a real network label is what makes this trap worse than the
// erpc_upstream_request_total one that traffic.go documents: the earlier
// finding (the poller shows up as network="n/a") suggests a filter, and the
// filter does not work.
//
// So: latency and volume come from erpc_network_*, always. Errors, lag and
// selection state come from erpc_upstream_*/erpc_selection_*, are attached to
// an ENDPOINT rather than a chain, and the field names say what is in them
// (EndpointHealth.Requests, not "traffic").
type Analytics struct {
	// At is when the reading was taken — the caller's to fill in, as
	// Traffic.At is, since a Prometheus dump carries no scrape timestamp.
	At time.Time
	// Since is the gateway process's start time. Every counter here is
	// cumulative from it.
	Since time.Time

	// Networks is what clients experienced, per chain.
	Networks []NetworkAnalytics
	// Endpoints is what the gateway sees of each upstream, INCLUDING its own
	// state poller's view. Keyed by the upstream id from our own erpc.yaml.
	Endpoints []EndpointHealth
}

// NetworkAnalytics is one chain's client-facing latency, cut two ways.
//
// Both cuts are over the same requests — the same histogram, summed across
// different labels — so their counts agree and their means do not have to be
// reconciled. By method answers "which call is slow"; by endpoint answers
// "which server is slow". Those are different questions and an operator
// arrives with one or the other.
type NetworkAnalytics struct {
	Network string
	ChainID int

	Methods   []MethodLatency
	Endpoints []EndpointLatency
}

// Latency is one histogram: how many requests, how long in total, and how they
// fell across the buckets eRPC configures.
//
// Buckets are CUMULATIVE, as Prometheus defines them: each Count is every
// request that finished within LE seconds, so the last (LE +Inf) equals Count.
// They are carried raw rather than turned into a quantile on purpose. eRPC's
// buckets are 0.05, 0.5, 5, 30, +Inf — with neighbours an order of magnitude
// apart, a p95 interpolated between two of them would be a number this process
// made up. "5 of 5 answered within 500ms" is the same fact without the
// invented precision, and it is what the buckets literally say.
type Latency struct {
	Count   float64
	Sum     float64 // seconds
	Buckets []Bucket
}

// Bucket is one cumulative histogram bucket. LE is +Inf for the final one.
type Bucket struct {
	LE    float64
	Count float64
}

// Mean is the average request duration in seconds, and it is NaN when nothing
// has been counted.
//
// NaN rather than 0 because they mean opposite things: 0 is a claim about how
// fast this endpoint is, and "nobody has called it" is not a claim about speed
// at all. A renderer that prints "0ms" for an uncalled method is stating the
// best possible latency for something that has never answered anything.
func (l Latency) Mean() float64 {
	if l.Count == 0 {
		return math.NaN()
	}
	return l.Sum / l.Count
}

// MethodLatency is one JSON-RPC method's client-facing latency on one chain.
type MethodLatency struct {
	Method string
	Latency
}

// EndpointLatency is one upstream's client-facing latency on one chain — the
// requests it actually answered for clients, which on a healthy gateway is a
// small fraction of what it was asked (see EndpointHealth.Requests).
type EndpointLatency struct {
	Upstream string
	Latency
}

// EndpointHealth is what the gateway knows about one upstream, from its own
// vantage point rather than a client's.
type EndpointHealth struct {
	// Upstream is the id from our own erpc.yaml.
	Upstream string
	// ChainID is the chain this endpoint serves, taken from whichever family
	// names it with a real network; 0 when every mention was network="n/a"
	// (which happens when the only thing that has ever talked to this endpoint
	// is the poller, on a gateway whose config lists it for a chain nobody has
	// called).
	ChainID int

	// Requests is erpc_upstream_request_total summed over every label.
	//
	// It INCLUDES the state poller, and on a quiet gateway it is almost
	// entirely the state poller. The name says "requests", not "traffic",
	// because it is not client traffic and must never be rendered as if it
	// were. It is carried because "how hard is my gateway leaning on this
	// endpoint" is a real question — a poller against a rate-limited public
	// endpoint is a genuine diagnosis, and it is invisible in every
	// client-facing number.
	Requests float64

	// Errors is erpc_upstream_request_errors_total, split by class, severity
	// and method, largest first.
	Errors []ErrorClass

	// Score and Position are eRPC's own selection state: why it routes the way
	// it does. Position 0 is the endpoint currently being preferred.
	Score    float64
	Position float64
	// PrimarySwitches counts erpc_selection_primary_switch_total landing ON
	// this endpoint. A number that climbs is a gateway flapping between
	// upstreams, which is a different problem from either of them being slow.
	PrimarySwitches float64
	// ExcludedSeconds is how long the selection policy has had this endpoint
	// excluded.
	ExcludedSeconds float64

	// HeadLag and FinalizationLag are how far behind this endpoint's latest
	// and finalized blocks are, in blocks. LatestBlock is its head.
	HeadLag         float64
	FinalizationLag float64
	LatestBlock     float64
}

// ErrorClass is one error class an endpoint has returned, with how often.
type ErrorClass struct {
	Class    string
	Severity string
	Method   string
	Count    float64
}

// The metric families AnalyticsFromSamples reads. The erpc_network_* ones are
// client-facing; every erpc_upstream_*/erpc_selection_* one below is not, and
// the two sets land in different halves of Analytics.
const (
	metricNetworkDurationCount  = "erpc_network_request_duration_seconds_count"
	metricNetworkDurationSum    = "erpc_network_request_duration_seconds_sum"
	metricNetworkDurationBucket = "erpc_network_request_duration_seconds_bucket"

	metricUpstreamRequests   = "erpc_upstream_request_total"
	metricUpstreamErrors     = "erpc_upstream_request_errors_total"
	metricUpstreamHeadLag    = "erpc_upstream_block_head_lag"
	metricUpstreamFinalLag   = "erpc_upstream_finalization_lag"
	metricUpstreamLatestHead = "erpc_upstream_latest_block_number"

	metricSelectionScore    = "erpc_selection_score"
	metricSelectionPosition = "erpc_selection_position"
	metricSelectionExcluded = "erpc_selection_excluded_seconds"
	metricSelectionSwitch   = "erpc_selection_primary_switch_total"

	labelCategory = "category"
	labelLE       = "le"
	labelError    = "error"
	labelSeverity = "severity"
	labelTo       = "to"

	// rollupUpstream is the label eRPC uses on the per-NETWORK summary row of
	// an upstream-scoped gauge: erpc_upstream_latest_block_number carries
	// upstream="*" alongside the real endpoints, holding the network's own
	// head. Read literally it becomes an endpoint named "*" — a phantom row
	// under a name no configuration contains, which is the same mistake
	// upstream="n/a" would have made on the share bars (see
	// NetworkTraffic.Unattributed).
	rollupUpstream = "*"
)

// AnalyticsFromSamples turns a gateway's Prometheus samples into Analytics.
//
// project filters to samples whose "project" label equals it; pass "" to skip
// the filter, exactly as FromSamples does.
func AnalyticsFromSamples(samples []Sample, project string) Analytics {
	var a Analytics

	// Client-facing, keyed [network][category] and [network][upstream].
	byMethod := map[string]map[string]*Latency{}
	byEndpoint := map[string]map[string]*Latency{}
	networks := map[string]bool{}

	// Endpoint-facing, keyed by upstream id.
	health := map[string]*EndpointHealth{}
	errs := map[string]map[ErrorClass]float64{}

	endpoint := func(id string) *EndpointHealth {
		if health[id] == nil {
			health[id] = &EndpointHealth{Upstream: id}
		}
		return health[id]
	}
	// note records the chain an endpoint serves the first time a sample names
	// a real one. Later mentions do not overwrite it: an endpoint's own
	// network label is "n/a" on every poller row, and letting those win would
	// erase the chain a perfectly well-identified endpoint belongs to.
	note := func(e *EndpointHealth, network string) {
		if e.ChainID == 0 {
			e.ChainID = parseEVMChainID(network)
		}
	}

	for _, s := range samples {
		if s.Name == metricStartTime {
			a.Since = unixSeconds(s.Value)
			continue
		}
		if project != "" && s.Labels[labelProject] != project {
			continue
		}
		network := s.Labels[labelNetwork]
		upstream := s.Labels[labelUpstream]

		switch s.Name {
		// ---- client-facing: erpc_network_* -------------------------------
		case metricNetworkDurationCount, metricNetworkDurationSum, metricNetworkDurationBucket:
			if network == "" || network == noNetwork {
				continue
			}
			networks[network] = true
			method := s.Labels[labelCategory]

			for _, target := range []struct {
				into map[string]map[string]*Latency
				key  string
			}{{byMethod, method}, {byEndpoint, upstream}} {
				if target.key == "" {
					continue
				}
				if target.into[network] == nil {
					target.into[network] = map[string]*Latency{}
				}
				if target.into[network][target.key] == nil {
					target.into[network][target.key] = &Latency{}
				}
				addSample(target.into[network][target.key], s)
			}

		// ---- endpoint-facing: erpc_upstream_* / erpc_selection_* ---------
		case metricUpstreamRequests:
			if !isRealUpstream(upstream) {
				continue
			}
			e := endpoint(upstream)
			note(e, network)
			e.Requests += s.Value

		case metricUpstreamErrors:
			if !isRealUpstream(upstream) {
				continue
			}
			e := endpoint(upstream)
			note(e, network)
			k := ErrorClass{
				Class:    s.Labels[labelError],
				Severity: s.Labels[labelSeverity],
				Method:   s.Labels[labelCategory],
			}
			if errs[upstream] == nil {
				errs[upstream] = map[ErrorClass]float64{}
			}
			errs[upstream][k] += s.Value

		// The gauges below are the endpoint's CURRENT state, not a total, so
		// they are assigned rather than summed. A gauge appears once per
		// (network, upstream) pair and an upstream serves one chain here, so
		// the last write is the only write in practice; assigning also means a
		// stale n/a-labelled duplicate cannot double a lag reading.
		case metricUpstreamHeadLag:
			if isRealUpstream(upstream) {
				e := endpoint(upstream)
				note(e, network)
				e.HeadLag = s.Value
			}
		case metricUpstreamFinalLag:
			if isRealUpstream(upstream) {
				e := endpoint(upstream)
				note(e, network)
				e.FinalizationLag = s.Value
			}
		case metricUpstreamLatestHead:
			if isRealUpstream(upstream) {
				e := endpoint(upstream)
				note(e, network)
				e.LatestBlock = s.Value
			}
		case metricSelectionScore:
			if isRealUpstream(upstream) {
				e := endpoint(upstream)
				note(e, network)
				e.Score = s.Value
			}
		case metricSelectionPosition:
			if isRealUpstream(upstream) {
				e := endpoint(upstream)
				note(e, network)
				e.Position = s.Value
			}
		case metricSelectionExcluded:
			if isRealUpstream(upstream) {
				e := endpoint(upstream)
				note(e, network)
				e.ExcludedSeconds = s.Value
			}
		case metricSelectionSwitch:
			// This family names the endpoint in "to" (and the one being left
			// in "from"), not in "upstream".
			to := s.Labels[labelTo]
			if !isRealUpstream(to) {
				continue
			}
			e := endpoint(to)
			note(e, network)
			e.PrimarySwitches += s.Value
		}
	}

	for network := range networks {
		n := NetworkAnalytics{Network: network, ChainID: parseEVMChainID(network)}
		for method, l := range byMethod[network] {
			n.Methods = append(n.Methods, MethodLatency{Method: method, Latency: finish(l)})
		}
		sort.Slice(n.Methods, func(i, j int) bool { return n.Methods[i].Method < n.Methods[j].Method })
		for up, l := range byEndpoint[network] {
			n.Endpoints = append(n.Endpoints, EndpointLatency{Upstream: up, Latency: finish(l)})
		}
		sort.Slice(n.Endpoints, func(i, j int) bool { return n.Endpoints[i].Upstream < n.Endpoints[j].Upstream })
		a.Networks = append(a.Networks, n)
	}
	sort.Slice(a.Networks, func(i, j int) bool {
		x, y := a.Networks[i], a.Networks[j]
		if x.ChainID != y.ChainID {
			return x.ChainID < y.ChainID
		}
		return x.Network < y.Network
	})

	for id, e := range health {
		for k, v := range errs[id] {
			k.Count = v
			e.Errors = append(e.Errors, k)
		}
		// Largest class first: the biggest source of failure is what someone
		// opening this page is looking for, and an alphabetical list buries it
		// under whichever class happens to start with an early letter.
		sort.Slice(e.Errors, func(i, j int) bool {
			if e.Errors[i].Count != e.Errors[j].Count {
				return e.Errors[i].Count > e.Errors[j].Count
			}
			return e.Errors[i].Class < e.Errors[j].Class
		})
		a.Endpoints = append(a.Endpoints, *e)
	}
	sort.Slice(a.Endpoints, func(i, j int) bool { return a.Endpoints[i].Upstream < a.Endpoints[j].Upstream })

	return a
}

// isRealUpstream rejects the two label values that name something other than
// an endpoint: eRPC's per-network rollup ("*") and its no-upstream sentinel
// ("n/a"). Either one, taken literally, renders as an endpoint nobody
// configured.
func isRealUpstream(upstream string) bool {
	return upstream != "" && upstream != rollupUpstream && upstream != noNetwork
}

// addSample folds one _count/_sum/_bucket line into a histogram.
func addSample(l *Latency, s Sample) {
	switch s.Name {
	case metricNetworkDurationCount:
		l.Count += s.Value
	case metricNetworkDurationSum:
		l.Sum += s.Value
	case metricNetworkDurationBucket:
		le, err := parseBucketBound(s.Labels[labelLE])
		if err != nil {
			// A bound this process cannot read would otherwise land in the
			// series as a zero and reorder every band around it. The rest of
			// the histogram is still perfectly usable without it.
			return
		}
		l.Buckets = append(l.Buckets, Bucket{LE: le, Count: s.Value})
	}
}

// finish sorts a histogram's buckets and merges the duplicates that summing
// across labels produces: two label sets on one network (say two methods on
// one endpoint) each carry their own le="0.5" row, and both belong in the
// endpoint's single histogram.
func finish(l *Latency) Latency {
	sort.Slice(l.Buckets, func(i, j int) bool { return l.Buckets[i].LE < l.Buckets[j].LE })
	merged := l.Buckets[:0]
	for _, b := range l.Buckets {
		if n := len(merged); n > 0 && merged[n-1].LE == b.LE {
			merged[n-1].Count += b.Count
			continue
		}
		merged = append(merged, b)
	}
	l.Buckets = merged
	return *l
}

// parseBucketBound decodes an "le" label, whose final value is the literal
// "+Inf" that strconv.ParseFloat already understands.
func parseBucketBound(le string) (float64, error) {
	return strconv.ParseFloat(le, 64)
}
