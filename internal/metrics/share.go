package metrics

import (
	"math"
	"sort"
)

// Intent is what the routing configuration says an upstream should carry.
//
// It mirrors catalog.GatewayUpstream just enough for this comparison:
// Upstream is the rendered erpc.yaml id, and Local is
// catalog.GatewayUpstream.Local — whether the operator runs it.
//
// Local is a preference relative to the rest of ITS OWN NETWORK, which is why
// Shares below reads the whole []Intent rather than each flag alone: a
// non-local upstream is rendered into the 0.2-score fallback tier (catalog's
// gatewayConfigTemplate, "scoreMultipliers: - overall: 0.2") only when that
// network also has a local upstream. On a network with none, every upstream
// goes in at full weight and intent spreads evenly across all of them — see
// the default arm of the switch in Shares.
//
// This package does not import catalog to build it — a caller assembles
// []Intent from whatever GatewayConfig it already has, keeping this package's
// only dependency on the shape of that decision, not on catalog itself.
type Intent struct {
	Upstream string
	Local    bool
}

// divergenceThreshold is how far Actual may drift from Intended before
// Shares calls it Diverged. 20 percentage points is coarse on purpose: eRPC
// picks a single best-scoring upstream per request rather than
// weighted-round-robining across the preferred tier (see the note on
// Shares below), so even a perfectly healthy gateway will show a lumpy,
// bursty split across a short observation window — one upstream having a
// slightly better recent latency sample can send it 100% of traffic for a
// while by design. A tight threshold would flag that as broken every time
// the score naturally shifts; 0.20 is loose enough to survive that churn
// while still catching an upstream that is structurally carrying the wrong
// share (e.g. the "local" node down and 100% of traffic silently going to
// a fallback).
const divergenceThreshold = 0.20

// Share is one upstream's actual vs. intended share of a network's
// attributed successes (erpc_network_successful_request_total, summed —
// see NetworkTraffic).
type Share struct {
	Upstream  string
	Succeeded float64
	// Actual is Succeeded / (sum of Succeeded across every upstream on this
	// network), in [0, 1]. 0 when the network has carried no traffic at all
	// (see the zero-total case below).
	Actual float64
	// Intended is this upstream's target share, per the rule documented on
	// Shares, in [0, 1].
	Intended float64
	// Diverged is true when |Actual - Intended| exceeds divergenceThreshold.
	Diverged bool
}

// Shares compares nt's attributed successes against intents and reports,
// per upstream, whether it is carrying the share the routing config
// intended.
//
// # The intent rule
//
// Intended is 1/N spread evenly across the N upstreams marked Local — the
// preferred tier every erpc.yaml gives a plain (unscored) routing slot.
// Fallback upstreams intend 0: anything a fallback carries is, by
// definition, traffic the preferred tier could not or did not serve, i.e.
// failover, and failover happening is not itself divergence — a fallback
// sitting at 0% is the config working, not an anomaly to flag.
//
// If a network has NO local upstream at all — every upstream in intents is
// a fallback — there is nothing to prefer, so intent spreads evenly across
// all of them instead of collapsing to all-zero (which would make every
// last upstream "diverge" from an intent of 0 the moment it serves
// anything, in a config that has no other option).
//
// eRPC does not weighted-round-robin across same-tier upstreams; its
// selection policy picks the single best-scoring one per request (see
// catalog's routing.scoreMultipliers comment). The 0.2 score multiplier
// fallbacks get is therefore a priority signal, not a traffic-share target,
// and must NOT be normalised into one (e.g. "fallbacks should get 0.2 /
// (0.2 + 1*N) of traffic") — that would present an invented number as if it
// were the config's actual intent.
//
// # Edge cases
//
// A network with zero total successes (every upstream's Succeeded is 0)
// gets Actual = 0 for everyone and Diverged = false for everyone: a gateway
// nobody has called yet is not misrouting, it is merely unused, and
// dividing 0/0 into a share is not a meaningful signal either way.
//
// An upstream named in intents but absent from nt.Upstreams (never appears
// in erpc_network_successful_request_total) is still included, with
// Succeeded and Actual both 0 — it exists in the config and has carried
// nothing, which is itself worth reporting when Intended says it should be
// carrying some.
//
// An upstream present in nt.Upstreams but absent from intents — removed
// from the routing config while eRPC's cumulative counters (which reset
// only on restart) still remember it — is included with Intended = 0: it
// has no claim on any traffic under the current config, so whatever it is
// still carrying is exactly as divergent as a fallback overshooting its
// zero intent.
func Shares(nt NetworkTraffic, intents []Intent) []Share {
	succeededByUpstream := make(map[string]float64, len(nt.Upstreams))
	var total float64
	for _, u := range nt.Upstreams {
		succeededByUpstream[u.Upstream] = u.Succeeded
		total += u.Succeeded
	}

	var localCount int
	inIntents := make(map[string]bool, len(intents))
	localSet := make(map[string]bool, len(intents))
	for _, in := range intents {
		inIntents[in.Upstream] = true
		if in.Local {
			localSet[in.Upstream] = true
			localCount++
		}
	}

	// The set of upstreams to report is the union of "named in the config"
	// and "has counters" — either alone would silently drop one of the two
	// edge cases above.
	seen := make(map[string]bool, len(intents)+len(nt.Upstreams))
	var names []string
	for _, in := range intents {
		if !seen[in.Upstream] {
			seen[in.Upstream] = true
			names = append(names, in.Upstream)
		}
	}
	for _, u := range nt.Upstreams {
		if !seen[u.Upstream] {
			seen[u.Upstream] = true
			names = append(names, u.Upstream)
		}
	}

	shares := make([]Share, 0, len(names))
	for _, name := range names {
		succeeded := succeededByUpstream[name]

		var actual float64
		if total > 0 {
			actual = succeeded / total
		}

		var intended float64
		switch {
		case !inIntents[name]:
			intended = 0
		case localCount > 0 && localSet[name]:
			intended = 1 / float64(localCount)
		case localCount > 0:
			intended = 0
		default:
			// No local upstream anywhere in this network's intents: spread
			// evenly across whatever fallbacks exist instead of every one
			// of them "diverging" from an intent of 0.
			intended = 1 / float64(len(intents))
		}

		diverged := total > 0 && math.Abs(actual-intended) > divergenceThreshold

		shares = append(shares, Share{
			Upstream:  name,
			Succeeded: succeeded,
			Actual:    actual,
			Intended:  intended,
			Diverged:  diverged,
		})
	}

	sort.Slice(shares, func(i, j int) bool { return shares[i].Upstream < shares[j].Upstream })
	return shares
}
