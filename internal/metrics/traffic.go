package metrics

import (
	"sort"
	"strconv"
	"strings"
	"time"
)

// Traffic is one gateway's counters at one moment.
type Traffic struct {
	// At is when this reading was taken. FromSamples never sets it — the
	// value has no home in a Prometheus dump, since eRPC's counters carry
	// no per-sample scrape timestamp (see the ParseText doc comment) — so
	// it is the caller's to fill in from the moment it fetched the dump.
	At time.Time
	// Since is the gateway process's start time, decoded from
	// process_start_time_seconds. It answers "how much of this counter's
	// life have we actually been counting", which matters because these
	// are cumulative counters that reset to zero on every eRPC restart: a
	// reading taken two minutes after a restart and one taken two weeks in
	// are not directly comparable without knowing which situation you're in.
	Since time.Time

	Networks []NetworkTraffic
}

// NetworkTraffic is the client-facing traffic for one network eRPC serves.
type NetworkTraffic struct {
	// Network is eRPC's own label value, e.g. "evm:369".
	Network string
	// ChainID is parsed out of Network when it has the "evm:<n>" shape eRPC
	// uses for every EVM network; 0 otherwise (a non-EVM architecture, or a
	// value that does not parse — either way, 0 is not a valid chain id, so
	// it cannot be confused with a real one).
	ChainID int
	// Received sums erpc_network_request_received_total across every other
	// label eRPC attaches (category, finality, agent_name, ...) for this
	// network.
	Received float64

	// Unattributed is the requests this network answered WITHOUT calling any
	// upstream — eRPC's own response cache being the ordinary cause. They are
	// successes, so they have to be accounted for somewhere, but they belong
	// to no endpoint and must never be rendered as one.
	//
	// MEASURED against a live gateway: 12 client calls produced 10 samples
	// labelled with a real upstream and 2 labelled upstream="n/a", the latter
	// carrying attempt="0" and vendor="n/a" where a genuine dial carries
	// attempt="1" and a vendor name.
	Unattributed float64

	Upstreams []UpstreamTraffic
}

// UpstreamTraffic is one upstream's contribution to a network's traffic.
type UpstreamTraffic struct {
	// Upstream is the upstream id from our own erpc.yaml (catalog.GatewayUpstream.ID).
	Upstream string
	// Succeeded sums erpc_network_successful_request_total across every
	// other label, for this (network, upstream) pair.
	Succeeded float64
}

// The three eRPC metric families FromSamples reads. Every other family in a
// gateway's dump (Go runtime stats, promhttp's own scrape counters, the
// selection-policy and duration histograms) describes something other than
// "who actually carried client traffic" and is ignored.
const (
	metricSuccessful = "erpc_network_successful_request_total"
	metricReceived   = "erpc_network_request_received_total"
	metricStartTime  = "process_start_time_seconds"

	labelProject  = "project"
	labelNetwork  = "network"
	labelUpstream = "upstream"

	// noNetwork is eRPC's label value for a request that never resolved to
	// a served network — devnet health checks, malformed paths, the
	// process's own eth_chainId self-tests. It is poller/no-network
	// traffic, not a chain anyone dialed, so networks with this label are
	// dropped entirely rather than surfaced as a network named "n/a".
	noNetwork = "n/a"
)

// FromSamples turns a gateway's Prometheus samples into Traffic.
//
// project filters to samples whose "project" label equals it; pass "" to
// skip the filter (useful for a single-project gateway, which is every
// gateway this app renders — see catalog.GatewayConfig.ProjectIDOrDefault).
//
// # Why erpc_upstream_request_total is not read here
//
// It is tempting to reach for erpc_upstream_request_total (or
// erpc_upstream_attempt_outcome_total) instead of the erpc_network_* family,
// because "requests to an upstream" sounds like exactly what a traffic share
// wants. It is the wrong source, and the mistake is not cosmetic:
// erpc_upstream_request_total also counts eRPC's own internal state
// poller — the loop that keeps latest/finalized block numbers fresh by
// hitting every upstream on a timer, independent of whether any client has
// asked for anything. In this package's own testdata/erpc.txt, the
// state poller alone accounts for hundreds of eth_getBlockByNumber calls
// against upstream="public-1-1" and upstream="public-369-1" (see
// erpc_upstream_attempt_outcome_total, network="n/a" — the poller's calls
// are not even attributed to the network they poll), while the genuine,
// client-facing traffic to public-369-1 that same dump records is 5 calls
// to eth_blockNumber. Reading the upstream-scoped family would report a
// gateway that has served exactly one real client request as if it were
// carrying a heavy, evenly-split load — a share bar that looks confident
// and is wrong.
//
// The erpc_network_* family does not have this problem: it is populated on
// the client-facing request path only, which the state poller bypasses
// entirely (it talks to upstreams directly, never through network-level
// request handling). That makes it the only honest source in the dump for
// "who is carrying the load a client actually generated".
//
// Networks with the label value "n/a" are dropped: see noNetwork.
func FromSamples(samples []Sample, project string) Traffic {
	received := make(map[string]float64)            // network -> sum
	unattributed := make(map[string]float64)        // network -> sum, answered with no upstream
	succeeded := make(map[[2]string]float64)        // [network, upstream] -> sum
	upstreamsOf := make(map[string]map[string]bool) // network -> set of upstream ids
	networks := make(map[string]bool)
	var since time.Time

	for _, s := range samples {
		switch s.Name {
		case metricStartTime:
			since = unixSeconds(s.Value)

		case metricReceived:
			network := s.Labels[labelNetwork]
			if network == "" || network == noNetwork {
				continue
			}
			if project != "" && s.Labels[labelProject] != project {
				continue
			}
			networks[network] = true
			received[network] += s.Value

		case metricSuccessful:
			network := s.Labels[labelNetwork]
			if network == "" || network == noNetwork {
				continue
			}
			if project != "" && s.Labels[labelProject] != project {
				continue
			}
			upstream := s.Labels[labelUpstream]
			if upstream == "" {
				// A successful-request sample with no upstream label would
				// be un-attributable to anyone; eRPC always sets it on this
				// family, but skip rather than invent an "unknown" bucket
				// if a future eRPC version ever omits it.
				continue
			}
			if upstream == noNetwork {
				// upstream="n/a" is a request eRPC answered WITHOUT calling
				// any upstream — its companion labels are attempt="0" and
				// vendor="n/a", where a real dial carries attempt="1" and a
				// vendor. MEASURED against a live gateway: 12 client calls
				// produced 10 rows against the upstream and 2 of these.
				//
				// It has to be counted, and it must NOT become an upstream.
				// Made into one it renders as a phantom endpoint carrying
				// 17% of the traffic, under a name no configuration contains
				// — which is exactly what this screen exists to make
				// impossible. Dropped entirely it opens a gap between
				// received and attributed that reads as failed requests,
				// which is equally untrue: these requests succeeded.
				networks[network] = true
				unattributed[network] += s.Value
				continue
			}
			networks[network] = true
			succeeded[[2]string{network, upstream}] += s.Value
			if upstreamsOf[network] == nil {
				upstreamsOf[network] = make(map[string]bool)
			}
			upstreamsOf[network][upstream] = true
		}
	}

	t := Traffic{Since: since}
	for network := range networks {
		nt := NetworkTraffic{
			Network:      network,
			ChainID:      parseEVMChainID(network),
			Received:     received[network],
			Unattributed: unattributed[network],
		}
		for upstream := range upstreamsOf[network] {
			nt.Upstreams = append(nt.Upstreams, UpstreamTraffic{
				Upstream:  upstream,
				Succeeded: succeeded[[2]string{network, upstream}],
			})
		}
		sort.Slice(nt.Upstreams, func(i, j int) bool {
			return nt.Upstreams[i].Upstream < nt.Upstreams[j].Upstream
		})
		t.Networks = append(t.Networks, nt)
	}
	sort.Slice(t.Networks, func(i, j int) bool {
		a, b := t.Networks[i], t.Networks[j]
		if a.ChainID != b.ChainID {
			return a.ChainID < b.ChainID
		}
		return a.Network < b.Network
	})

	return t
}

// parseEVMChainID extracts <n> from a network label shaped "evm:<n>",
// returning 0 for anything else (a non-EVM architecture, or a value that
// does not parse as a positive integer).
func parseEVMChainID(network string) int {
	const prefix = "evm:"
	if !strings.HasPrefix(network, prefix) {
		return 0
	}
	id, err := strconv.Atoi(network[len(prefix):])
	if err != nil || id <= 0 {
		return 0
	}
	return id
}

// unixSeconds decodes a Prometheus gauge holding a Unix timestamp (seconds
// since the epoch, as a float — process_start_time_seconds carries
// sub-second precision, e.g. 1.78519881873e+09) into a time.Time.
func unixSeconds(v float64) time.Time {
	sec := int64(v)
	nsec := int64((v - float64(sec)) * float64(time.Second))
	return time.Unix(sec, nsec).UTC()
}
