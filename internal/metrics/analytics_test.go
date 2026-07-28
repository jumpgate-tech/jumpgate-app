package metrics

import (
	"math"
	"os"
	"testing"
)

func realSamples(t *testing.T) []Sample {
	t.Helper()
	f, err := os.Open("testdata/erpc.txt")
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	defer f.Close()
	samples, err := ParseText(f)
	if err != nil {
		t.Fatalf("ParseText: %v", err)
	}
	return samples
}

// The whole point of the client-facing half: it must read the five real
// requests and nothing else. The same fixture carries 413 poller calls to
// public-1-1 on evm:1 in erpc_upstream_request_duration_seconds — a chain no
// client ever called — so a latency panel built on the wrong family would
// report a busy gateway with a 47ms mean instead of a quiet one.
func TestAnalyticsFromSamples_ClientFacingLatencyIsTheNetworkFamily(t *testing.T) {
	a := AnalyticsFromSamples(realSamples(t), "main")

	if len(a.Networks) != 1 {
		t.Fatalf("got %d networks, want only the one clients called: %+v", len(a.Networks), a.Networks)
	}
	n := a.Networks[0]
	if n.ChainID != 369 {
		t.Fatalf("ChainID: got %d, want 369", n.ChainID)
	}

	if len(n.Methods) != 1 || n.Methods[0].Method != "eth_blockNumber" {
		t.Fatalf("methods: %+v, want only eth_blockNumber", n.Methods)
	}
	m := n.Methods[0]
	if m.Count != 5 {
		t.Errorf("count: got %v, want 5 — the poller's hundreds of calls are not client traffic", m.Count)
	}
	// sum 0.834064999 over 5 requests.
	if got, want := m.Mean(), 0.834064999/5; math.Abs(got-want) > 1e-9 {
		t.Errorf("mean: got %v, want %v", got, want)
	}

	// Four finite buckets and +Inf, cumulative, exactly as the dump carries
	// them. Nothing here is interpolated into a quantile: with le=0.05 and
	// le=0.5 as neighbours, a p95 would be a number invented between them.
	wantBuckets := []Bucket{{LE: 0.05, Count: 0}, {LE: 0.5, Count: 5}, {LE: 5, Count: 5}, {LE: 30, Count: 5}, {LE: math.Inf(1), Count: 5}}
	if len(m.Buckets) != len(wantBuckets) {
		t.Fatalf("buckets: got %+v, want %+v", m.Buckets, wantBuckets)
	}
	for i, w := range wantBuckets {
		if m.Buckets[i] != w {
			t.Errorf("bucket %d: got %+v, want %+v", i, m.Buckets[i], w)
		}
	}

	// The same five requests, cut by endpoint instead of by method.
	if len(n.Endpoints) != 1 || n.Endpoints[0].Upstream != "public-369-1" || n.Endpoints[0].Count != 5 {
		t.Fatalf("per-endpoint latency: %+v, want 5 requests against public-369-1", n.Endpoints)
	}
}

// FOUND BY RUNNING IT, against a live gateway rather than this fixture: the
// client-facing duration histogram carries upstream="n/a" rows for requests
// eRPC answered from its own cache, exactly as the successful-request counter
// does. Cut by endpoint, those rows become an endpoint called "n/a" — a
// phantom server, carrying real traffic, under a name no configuration
// contains. That is the same bug NetworkTraffic.Unattributed exists to
// prevent, in a family one over.
//
// They are not dropped either: they were 12 of 22 requests on the gateway this
// was found on, and the fastest ones at that (28ms against the endpoint's
// 150ms). Dropping them would hide the cache entirely and make the endpoint
// latency look like the whole story.
func TestAnalyticsFromSamples_CacheHitsAreNotAnEndpoint(t *testing.T) {
	samples := []Sample{
		{Name: metricNetworkDurationCount, Labels: map[string]string{"project": "main", "network": "evm:369", "upstream": "public-369-1", "category": "eth_blockNumber"}, Value: 10},
		{Name: metricNetworkDurationSum, Labels: map[string]string{"project": "main", "network": "evm:369", "upstream": "public-369-1", "category": "eth_blockNumber"}, Value: 1.5},
		{Name: metricNetworkDurationCount, Labels: map[string]string{"project": "main", "network": "evm:369", "upstream": noNetwork, "category": "eth_chainId"}, Value: 12},
		{Name: metricNetworkDurationSum, Labels: map[string]string{"project": "main", "network": "evm:369", "upstream": noNetwork, "category": "eth_chainId"}, Value: 0.336},
	}
	a := AnalyticsFromSamples(samples, "main")
	n := a.Networks[0]

	for _, e := range n.Endpoints {
		if !isRealUpstream(e.Upstream) {
			t.Errorf("%q is in the endpoint cut: it is a request answered with no endpoint at all, and rendering it as one invents a server", e.Upstream)
		}
	}
	if len(n.Endpoints) != 1 || n.Endpoints[0].Upstream != "public-369-1" || n.Endpoints[0].Count != 10 {
		t.Fatalf("endpoints: %+v, want only the real one", n.Endpoints)
	}
	// Kept, named for what they are.
	if n.Cached.Count != 12 {
		t.Errorf("cached: got %v, want the 12 requests answered without calling anything", n.Cached.Count)
	}
	if got, want := n.Cached.Mean(), 0.336/12; math.Abs(got-want) > 1e-12 {
		t.Errorf("cached mean: got %v, want %v", got, want)
	}
	// And still counted once, under their method, on the by-method cut —
	// where "which call is slow" is the question and a cache hit is a
	// perfectly real answer to it.
	if len(n.Methods) != 2 {
		t.Errorf("methods: %+v, want both — a cached method is still a method clients called", n.Methods)
	}
}

// ALSO FOUND BY RUNNING IT, one poll after the cache rows: a request that
// ends in an error is labelled upstream="<error>" on the same histogram. Cut
// by endpoint it becomes a server called "<error>" — and on the gateway this
// was found on it was rendering above the real endpoint, three requests at
// 291ms.
//
// It is not dropped either, because how long a failure takes is a real
// diagnosis: failing fast and timing out after thirty seconds are different
// problems with different causes, and the counts alone cannot tell them apart.
func TestAnalyticsFromSamples_FailedRequestsAreNotAnEndpoint(t *testing.T) {
	samples := []Sample{
		{Name: metricNetworkDurationCount, Labels: map[string]string{"project": "main", "network": "evm:369", "upstream": "public-369-1", "category": "eth_blockNumber"}, Value: 22},
		{Name: metricNetworkDurationCount, Labels: map[string]string{"project": "main", "network": "evm:369", "upstream": errorUpstream, "category": "eth_getLogs"}, Value: 2},
		{Name: metricNetworkDurationSum, Labels: map[string]string{"project": "main", "network": "evm:369", "upstream": errorUpstream, "category": "eth_getLogs"}, Value: 0.7},
		// A sentinel this code has never seen. It must not become an endpoint
		// either — the shape of the value is the tell, not the exact spelling.
		{Name: metricNetworkDurationCount, Labels: map[string]string{"project": "main", "network": "evm:369", "upstream": "<something-new>", "category": "eth_call"}, Value: 4},
	}
	a := AnalyticsFromSamples(samples, "main")
	n := a.Networks[0]

	if len(n.Endpoints) != 1 || n.Endpoints[0].Upstream != "public-369-1" {
		t.Fatalf("endpoints: %+v, want only the real one", n.Endpoints)
	}
	if n.FailedLatency.Count != 2 {
		t.Errorf("failed latency: got %v requests, want 2", n.FailedLatency.Count)
	}
	if got, want := n.FailedLatency.Mean(), 0.35; math.Abs(got-want) > 1e-9 {
		t.Errorf("failed mean: got %v, want %v", got, want)
	}
	// The unknown sentinel is counted under its method — a client did call
	// eth_call — and nowhere else. Guessing what it means would be worse than
	// leaving it out of a cut it cannot be filed into.
	if len(n.Methods) != 3 {
		t.Errorf("methods: %+v, want all three", n.Methods)
	}
}

// The 56 transport failures against the devnet were every one of them the
// gateway's own state poller — no client saw a single one, and the row carries
// network="n/a" so there is no chain to file it under. It is still the most
// useful line in the dump: an endpoint that cannot be reached at all, found
// before a client finds it. Errors attach to the ENDPOINT for exactly that
// reason.
func TestAnalyticsFromSamples_EndpointErrorsSurviveHavingNoNetwork(t *testing.T) {
	a := AnalyticsFromSamples(realSamples(t), "main")

	e := endpointByID(t, a, "devnet")
	if len(e.Errors) != 1 {
		t.Fatalf("errors: %+v, want the one transport failure class", e.Errors)
	}
	err0 := e.Errors[0]
	if err0.Class != "ErrEndpointTransportFailure" || err0.Severity != "critical" || err0.Count != 56 {
		t.Errorf("error row: got %+v, want 56 critical ErrEndpointTransportFailure", err0)
	}
	if err0.Method != "eth_chainId" {
		t.Errorf("method: got %q, want eth_chainId", err0.Method)
	}

	// FOUND BY RUNNING IT: this endpoint has no selection state at all — eRPC
	// never scored it, because it could never be reached. Position 0 means
	// "currently preferred" and the zero value of a float is also 0, so on a
	// live gateway an unreachable devnet rendered as the preferred endpoint.
	// The absence of an opinion must not be the best possible verdict.
	if e.Scored {
		t.Error("an endpoint eRPC has never scored must not read as scored — position 0 would then mean 'preferred'")
	}
}

// Selection is the answer to the question the share bar raises: not "is this
// endpoint carrying the wrong share" but "why". eRPC publishes its own score
// and position per upstream, and they are the reason it routes the way it does.
func TestAnalyticsFromSamples_SelectionStateIsCarried(t *testing.T) {
	a := AnalyticsFromSamples(realSamples(t), "main")

	e := endpointByID(t, a, "public-369-1")
	if math.Abs(e.Score-0.04708422735343804) > 1e-12 {
		t.Errorf("score: got %v", e.Score)
	}
	if e.Position != 0 {
		t.Errorf("position: got %v, want 0", e.Position)
	}
	if e.PrimarySwitches != 1 {
		t.Errorf("primary switches: got %v, want 1", e.PrimarySwitches)
	}
	if !e.Scored {
		t.Error("an endpoint eRPC has published a score for must read as scored")
	}
	if e.ChainID != 369 {
		t.Errorf("ChainID: got %d, want 369 — an endpoint's chain comes from whichever family names it", e.ChainID)
	}
}

// An endpoint's request count is NOT client traffic and must never be
// presented as if it were: this number includes the state poller, which is
// most of it. It is carried because "how much is the gateway itself asking of
// this endpoint" is a real question — a poller hammering a rate-limited public
// endpoint is a genuine diagnosis — but the field says what it holds.
func TestAnalyticsFromSamples_EndpointRequestsIncludeThePoller(t *testing.T) {
	a := AnalyticsFromSamples(realSamples(t), "main")

	e := endpointByID(t, a, "public-1-1")
	// 1 eth_chainId + 412 eth_getBlockByNumber + 2 + 3 eth_syncing + 1 = 419.
	if e.Requests != 419 {
		t.Errorf("requests: got %v, want 419 (every erpc_upstream_request_total row for this endpoint)", e.Requests)
	}
	// And that endpoint appears in NO client-facing network, because no
	// client ever called chain 1.
	for _, n := range a.Networks {
		for _, ep := range n.Endpoints {
			if ep.Upstream == "public-1-1" {
				t.Errorf("public-1-1 has client-facing latency on chain %d, but no client ever called it", n.ChainID)
			}
		}
	}
}

// Head lag is how far behind an endpoint's latest block is. It is a gauge, and
// a gauge of zero is a real reading — which is why the endpoint has to exist in
// the output even when every number on it is zero.
func TestAnalyticsFromSamples_LagGauges(t *testing.T) {
	a := AnalyticsFromSamples(realSamples(t), "main")
	e := endpointByID(t, a, "public-1-1")
	if e.HeadLag != 0 {
		t.Errorf("head lag: got %v, want 0", e.HeadLag)
	}
	if e.LatestBlock != 25628270 {
		t.Errorf("latest block: got %v, want 25628270", e.LatestBlock)
	}
}

// erpc_upstream_latest_block_number carries a per-network ROLLUP row spelled
// upstream="*" alongside the real ones. Read literally it becomes an endpoint
// named "*" holding the network's own head — a phantom row under a name no
// configuration contains, which is precisely the mistake upstream="n/a" would
// have made on the share bars.
func TestAnalyticsFromSamples_RollupUpstreamIsNotAnEndpoint(t *testing.T) {
	a := AnalyticsFromSamples(realSamples(t), "main")
	for _, e := range a.Endpoints {
		if e.Upstream == "*" || e.Upstream == noNetwork {
			t.Errorf("endpoint %q is a rollup label, not an endpoint: %+v", e.Upstream, e)
		}
	}
}

// The project filter is the same one FromSamples applies, for the same reason:
// one gateway serves one project here, and a sample from another project would
// be another gateway's traffic entirely.
func TestAnalyticsFromSamples_ProjectFilter(t *testing.T) {
	a := AnalyticsFromSamples(realSamples(t), "not-this-one")
	if len(a.Networks) != 0 {
		t.Errorf("networks: %+v, want none for a project this gateway does not serve", a.Networks)
	}
	if len(a.Endpoints) != 0 {
		t.Errorf("endpoints: %+v, want none", a.Endpoints)
	}
}

// Ordering has to be deterministic or the page reshuffles under a poll.
func TestAnalyticsFromSamples_DeterministicOrdering(t *testing.T) {
	samples := []Sample{
		{Name: metricUpstreamErrors, Labels: map[string]string{"project": "main", "upstream": "zeta", "network": "evm:1", "error": "ErrB", "severity": "warn", "category": "eth_call"}, Value: 1},
		{Name: metricUpstreamErrors, Labels: map[string]string{"project": "main", "upstream": "zeta", "network": "evm:1", "error": "ErrA", "severity": "warn", "category": "eth_call"}, Value: 9},
		{Name: metricUpstreamErrors, Labels: map[string]string{"project": "main", "upstream": "alpha", "network": "evm:1", "error": "ErrA", "severity": "warn", "category": "eth_call"}, Value: 2},
		{Name: metricNetworkDurationCount, Labels: map[string]string{"project": "main", "network": "evm:2", "upstream": "u", "category": "b_method"}, Value: 1},
		{Name: metricNetworkDurationCount, Labels: map[string]string{"project": "main", "network": "evm:2", "upstream": "u", "category": "a_method"}, Value: 1},
		{Name: metricNetworkDurationCount, Labels: map[string]string{"project": "main", "network": "evm:1", "upstream": "u", "category": "a_method"}, Value: 1},
	}
	a := AnalyticsFromSamples(samples, "main")

	if len(a.Networks) != 2 || a.Networks[0].ChainID != 1 || a.Networks[1].ChainID != 2 {
		t.Fatalf("networks out of order: %+v", a.Networks)
	}
	if got := a.Networks[1].Methods; len(got) != 2 || got[0].Method != "a_method" || got[1].Method != "b_method" {
		t.Errorf("methods out of order: %+v", got)
	}
	// Two endpoints, not three: "u" is named only by the client-facing
	// histogram, and an endpoint's health row is built from what the GATEWAY
	// saw of it. Nothing in this reading is the gateway's view of "u".
	if len(a.Endpoints) != 2 {
		t.Fatalf("endpoints: %+v", a.Endpoints)
	}
	if a.Endpoints[0].Upstream != "alpha" {
		t.Errorf("endpoints out of order: %+v", a.Endpoints)
	}
	// Errors sort by count DESCENDING — the biggest error class is the one
	// worth reading first, and an alphabetical list buries it.
	zeta := endpointByID(t, a, "zeta")
	if len(zeta.Errors) != 2 || zeta.Errors[0].Class != "ErrA" {
		t.Errorf("errors must lead with the largest class: %+v", zeta.Errors)
	}
}

// A network label of "n/a" is not a chain. It is the poller's own bucket, and
// on the client-facing side there is nothing there to show.
func TestAnalyticsFromSamples_DropsNoNetworkFromTheClientSide(t *testing.T) {
	samples := []Sample{
		{Name: metricNetworkDurationCount, Labels: map[string]string{"project": "main", "network": noNetwork, "upstream": "u", "category": "eth_chainId"}, Value: 56},
		{Name: metricNetworkDurationSum, Labels: map[string]string{"project": "main", "network": noNetwork, "upstream": "u", "category": "eth_chainId"}, Value: 1},
	}
	a := AnalyticsFromSamples(samples, "main")
	if len(a.Networks) != 0 {
		t.Errorf("networks: %+v, want none — n/a is not a chain", a.Networks)
	}
}

// A bucket whose le does not parse would otherwise land in the series as a
// zero and reorder every band around it. Dropping it keeps the rest of a
// perfectly good histogram usable.
func TestAnalyticsFromSamples_UnparseableBucketBoundIsDropped(t *testing.T) {
	samples := []Sample{
		{Name: metricNetworkDurationCount, Labels: map[string]string{"project": "main", "network": "evm:1", "upstream": "u", "category": "eth_call"}, Value: 2},
		{Name: metricNetworkDurationBucket, Labels: map[string]string{"project": "main", "network": "evm:1", "upstream": "u", "category": "eth_call", "le": "0.5"}, Value: 1},
		{Name: metricNetworkDurationBucket, Labels: map[string]string{"project": "main", "network": "evm:1", "upstream": "u", "category": "eth_call", "le": "wat"}, Value: 2},
	}
	a := AnalyticsFromSamples(samples, "main")
	m := a.Networks[0].Methods[0]
	if len(m.Buckets) != 1 || m.Buckets[0].LE != 0.5 {
		t.Errorf("buckets: %+v, want only the parseable one", m.Buckets)
	}
}

// Mean over no requests is not zero — zero milliseconds is a claim about speed,
// and "nobody has called this" is not.
func TestLatency_MeanOfNothingIsNaN(t *testing.T) {
	var l Latency
	if !math.IsNaN(l.Mean()) {
		t.Errorf("mean of no requests: got %v, want NaN so it cannot be rendered as 0ms", l.Mean())
	}
}

func endpointByID(t *testing.T, a Analytics, id string) EndpointHealth {
	t.Helper()
	for _, e := range a.Endpoints {
		if e.Upstream == id {
			return e
		}
	}
	t.Fatalf("no endpoint %q in %+v", id, a.Endpoints)
	return EndpointHealth{}
}
