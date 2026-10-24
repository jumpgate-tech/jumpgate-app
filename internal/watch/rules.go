// Package watch is the on-box health watch (`jumpgate agent watch`): it
// polls the box's node through the monitor probes and its configured
// endpoints, turns failures into alerts, and keeps them in a state file the
// box's gate and SSH probes read.
package watch

import (
	"fmt"
	"maps"
	"math"
	"slices"
	"time"

	"github.com/valve-tech/jumpgate/internal/catalog"
	"github.com/valve-tech/jumpgate/internal/monitor"
)

// Endpoint is one endpoint the watch probes: Kind "rpc" (POST
// eth_blockNumber) or "beacon" (GET /eth/v1/node/health).
type Endpoint = catalog.WatchEndpoint

// Alert is one alert; the same type monitor.ModuleStatus.Alerts carries.
type Alert = monitor.Alert

// openAfter is how many consecutive failing polls open an alert.
const openAfter = 2

// recentMax is how many cleared alerts State keeps.
const recentMax = 50

// State is what the watch writes to its state file.
type State struct {
	At      time.Time      `json:"at"`
	Open    []Alert        `json:"open"`
	Recent  []Alert        `json:"recent"`            // cleared, newest first, at most 50
	Strikes map[string]int `json:"strikes,omitempty"` // consecutive failing polls per key
}

// Inputs is one poll's readings.
type Inputs struct {
	Snap         *monitor.Snapshot // nil when the box has no node
	ChainID      int
	EndpointList []Endpoint       // the configured endpoints, in order
	Endpoints    map[string]error // endpoint name → its probe's error, nil = answered
}

// check is one rule's result this poll.
type check struct {
	key     string
	failing bool
	message string
}

// lagLimit is how many blocks behind the reference the node may be: one
// minute of blocks, and never under 3.
func lagLimit(chainID int) uint64 {
	slot := 12
	if n, ok := catalog.NetworkByChainID(chainID); ok && n.SlotSeconds > 0 {
		slot = n.SlotSeconds
	}
	return uint64(max(3, 60/slot))
}

func checks(in Inputs) []check {
	var out []check
	if s := in.Snap; s != nil {
		out = append(out,
			check{"exec-down", !s.ExecActive, "The execution client on this box is not running."},
			check{"beacon-down", !s.BeaconActive, "The beacon client on this box is not running."},
		)
		var lag uint64
		if s.RefHead > s.ExecHead {
			lag = s.RefHead - s.ExecHead
		}
		out = append(out,
			check{"exec-behind", s.RefHead > 0 && lag >= lagLimit(in.ChainID), fmt.Sprintf("The node is %d blocks behind the reference.", lag)},
			check{"no-peers", s.ExecActive && s.ExecPeers == 0, "The execution client has no peers."},
			check{"disk-90", s.DiskKnown && s.DiskUsedPct >= 90, fmt.Sprintf("Disk is %d%% full.", int(math.Floor(s.DiskUsedPct)))},
		)
	}
	for _, e := range in.EndpointList {
		err, probed := in.Endpoints[e.Name]
		out = append(out, check{"endpoint:" + e.Name, probed && err != nil, fmt.Sprintf("%s (%s) stopped answering.", e.Name, e.URL)})
	}
	return out
}

// Evaluate is the watch's rules, pure: an alert opens after two
// consecutive failing polls and clears on the first passing one, moving to
// Recent. An open alert whose rule no longer runs (an endpoint removed)
// clears too.
//
// docs:covers watch-run
func Evaluate(prev State, in Inputs, now time.Time) State {
	next := State{At: now, Strikes: map[string]int{}, Recent: slices.Clone(prev.Recent)}
	open := map[string]Alert{}
	for _, a := range prev.Open {
		open[a.Key] = a
	}
	seen := map[string]bool{}
	for _, c := range checks(in) {
		seen[c.key] = true
		a, isOpen := open[c.key]
		if !c.failing {
			if isOpen {
				next.clear(a, now)
			}
			continue
		}
		next.Strikes[c.key] = prev.Strikes[c.key] + 1
		switch {
		case isOpen:
			a.Message = c.message
			next.Open = append(next.Open, a)
		case next.Strikes[c.key] >= openAfter:
			next.Open = append(next.Open, Alert{Key: c.key, Message: c.message, Since: now})
		}
	}
	for _, k := range slices.Sorted(maps.Keys(open)) {
		if !seen[k] {
			next.clear(open[k], now)
		}
	}
	if len(next.Strikes) == 0 {
		next.Strikes = nil
	}
	return next
}

func (s *State) clear(a Alert, now time.Time) {
	at := now
	a.Cleared = &at
	s.Recent = append([]Alert{a}, s.Recent...)
	if len(s.Recent) > recentMax {
		s.Recent = s.Recent[:recentMax]
	}
}
