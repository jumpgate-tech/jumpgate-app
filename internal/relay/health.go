package relay

import (
	"context"
	"fmt"
	"strconv"
)

// HealthProbe answers the health rollup with real state.
//
// It reads the chain's head through the same HTTP caller the relay uses for
// everything else, and it reads beacon status from the pool's own probe rather
// than dialling again. A rollup that reported an assumption would be worse than
// no rollup at all, because a monitor would trust it.
type HealthProbe struct {
	caller RPCCaller
	beacon *BeaconPool
	chains []int
}

// NewHealthProbe builds the rollup source. chains is what this gateway serves.
func NewHealthProbe(caller RPCCaller, beacon *BeaconPool, chains []int) *HealthProbe {
	return &HealthProbe{caller: caller, beacon: beacon, chains: chains}
}

// Chains reports the chains this gateway serves.
func (h *HealthProbe) Chains() []int { return h.chains }

// HasBeacon reports whether a chain has any beacon upstream at all.
func (h *HealthProbe) HasBeacon(chainID int) bool {
	if h.beacon == nil {
		return false
	}
	return h.beacon.Health(chainID).Total > 0
}

// Cell reports one chain across both categories, so one request answers "is this
// chain well" rather than two.
func (h *HealthProbe) Cell(ctx context.Context, chainID int) map[string]any {
	out := map[string]any{"rpc": h.rpcHealth(ctx, chainID)}
	if h.HasBeacon(chainID) {
		out["beacon"] = h.beacon.Health(chainID)
	}
	return out
}

// Rollup reports every chain under a selector. An empty selector means every
// chain; a category selector reports only the chains that actually have it.
func (h *HealthProbe) Rollup(ctx context.Context, selector string) map[string]any {
	chains := map[string]any{}
	for _, chainID := range h.chains {
		if selector == string(CategoryBeacon) && !h.HasBeacon(chainID) {
			continue
		}
		chains[strconv.Itoa(chainID)] = h.Cell(ctx, chainID)
	}
	return chains
}

// rpcHealth reads a chain's head. An upstream that cannot answer is reported as
// not ok rather than omitted: a missing field would read to a monitor as
// "nothing wrong here".
func (h *HealthProbe) rpcHealth(ctx context.Context, chainID int) map[string]any {
	if h.caller == nil {
		return map[string]any{"ok": false}
	}
	head, err := NewRPCBlockFetcher(h.caller, chainID).HeadNumber(ctx)
	if err != nil {
		return map[string]any{"ok": false}
	}
	return map[string]any{"ok": true, "head": fmt.Sprintf("0x%x", head)}
}
