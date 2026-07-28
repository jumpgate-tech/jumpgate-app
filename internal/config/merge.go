package config

import (
	"fmt"

	"github.com/valve-tech/valve-node-app/internal/catalog"
	"github.com/valve-tech/valve-node-app/internal/ops"
)

// OrphanedContainer is a container valve-node-app stopped managing but did NOT
// stop. It is persisted rather than derived: migrate() runs in memory and is
// only written back by the next Save, so a notice computed at load would
// disappear on the first save while the container kept serving.
type OrphanedContainer struct {
	// ContainerName is the container left running on the target.
	ContainerName string `json:"containerName"`
	// TargetID is the machine it is running on.
	TargetID string `json:"targetId"`
	// MergedInto is the gateway that absorbed its networks.
	MergedInto string `json:"mergedInto"`
}

// mergeGatewaysPerTarget folds every gateway sharing a Placement.TargetID into
// one. A gateway NAMES the machine it runs on, so two gateways on one device
// mean two managed eRPC containers: overlapping chains, two state pollers
// against the same node, and a reverse proxy that can only front one of them.
//
// The containers the merge leaves behind are returned, never stopped. This app
// does not stop a container it did not just start — see handleGatewayDelete,
// which says exactly that and leaves the container alone.
func mergeGatewaysPerTarget(gws []Gateway) ([]Gateway, []OrphanedContainer) {
	// Group by target, preserving first-seen order so the result is stable.
	order := make([]string, 0, len(gws))
	groups := make(map[string][]int, len(gws))
	for i, g := range gws {
		t := g.Placement.TargetID
		if _, seen := groups[t]; !seen {
			order = append(order, t)
		}
		groups[t] = append(groups[t], i)
	}

	var out []Gateway
	var orphans []OrphanedContainer

	for _, t := range order {
		idx := groups[t]
		if len(idx) == 1 {
			out = append(out, gws[idx[0]])
			continue
		}

		keep := survivorIndex(idx, gws)
		survivor := gws[keep]
		for _, i := range idx {
			if i == keep {
				continue
			}
			mergeConfigInto(&survivor.Config, gws[i].Config)
			orphans = append(orphans, OrphanedContainer{
				ContainerName: ops.ERPCContainerNameFor(gws[i].ID),
				TargetID:      t,
				MergedInto:    survivor.ID,
			})
		}
		out = append(out, survivor)
	}

	return out, orphans
}

// survivorIndex picks which gateway absorbs the others: the fronted one first,
// so the merge keeps the secure door and the stable hostname; then the one
// named DefaultGatewayID; then the earliest, so the choice is deterministic
// rather than map-order.
func survivorIndex(idx []int, gws []Gateway) int {
	for _, i := range idx {
		if gws[i].Config.Fronted() {
			return i
		}
	}
	for _, i := range idx {
		if gws[i].ID == DefaultGatewayID {
			return i
		}
	}
	return idx[0]
}

// upstreamKey identifies an upstream by what actually distinguishes it. Keying
// on endpoint alone is wrong: a managed upstream carries no endpoint of its own
// (the address is derived at render time), so every managed-devnet row on a
// target looks like {managed-devnet, <target>, ""} and two of them would both
// survive as "different".
func upstreamKey(u catalog.GatewayUpstream) [3]string {
	return [3]string{u.KindOrDefault(), u.TargetID, u.Endpoint}
}

// mergeConfigInto unions src's networks into dst. dst keeps its own ProjectID,
// BindAddr, Port, TLS and metrics settings: a merge changes WHICH chains a
// gateway serves, never the door it serves them on.
func mergeConfigInto(dst *catalog.GatewayConfig, src catalog.GatewayConfig) {
	seenUp := make(map[[3]string]bool)
	takenID := make(map[string]bool)
	for _, n := range dst.Networks {
		for _, u := range n.Upstreams {
			seenUp[upstreamKey(u)] = true
			takenID[u.ID] = true
		}
	}

	netAt := make(map[int]int, len(dst.Networks))
	for i, n := range dst.Networks {
		netAt[n.ChainID] = i
	}

	for _, sn := range src.Networks {
		at, ok := netAt[sn.ChainID]
		if !ok {
			dst.Networks = append(dst.Networks, catalog.GatewayNetwork{ChainID: sn.ChainID})
			at = len(dst.Networks) - 1
			netAt[sn.ChainID] = at
		}
		for _, u := range sn.Upstreams {
			if seenUp[upstreamKey(u)] {
				continue
			}
			seenUp[upstreamKey(u)] = true
			u.ID = uniqueUpstreamID(u.ID, takenID)
			takenID[u.ID] = true
			dst.Networks[at].Upstreams = append(dst.Networks[at].Upstreams, u)
		}
	}
}

// uniqueUpstreamID suffixes a colliding id rather than dropping the upstream:
// two gateways can legitimately carry different endpoints under the same
// generated name, and losing one silently would be a merge that reports success
// while removing an endpoint.
func uniqueUpstreamID(id string, taken map[string]bool) string {
	if id == "" || !taken[id] {
		return id
	}
	for n := 2; ; n++ {
		cand := fmt.Sprintf("%s-%d", id, n)
		if !taken[cand] {
			return cand
		}
	}
}
