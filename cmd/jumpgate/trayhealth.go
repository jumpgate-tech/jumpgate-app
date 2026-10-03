package main

// Overall health signal for the menubar status dot. Kept out of the `tray`
// build tag (no CGo, no webview) so the reduction is plain, testable Go; the
// darwin status item (statusitem_darwin.go) and the poller (tray.go) consume it.

// healthKind is the menubar dot's state, worst-first when it matters.
type healthKind int

const (
	healthOff  healthKind = iota // grey: nothing set up, all stopped, or unreachable
	healthOK                     // green: at least one gateway running, none blocked
	healthWarn                   // amber: reserved for a future degraded/slow signal
	healthDown                   // red: a gateway is blocked/unavailable
)

// Minimal projection of GET /api/gateways — only the fields the dot needs.
type gwHealth struct {
	// Status is ops.ContainerStatus verbatim on the wire (untagged → PascalCase),
	// so its running flag is "State", not "state".
	Status  struct{ State string } `json:"status"`
	Blocked string                 `json:"blocked"`
}

type gwListResp struct {
	Gateways []gwHealth `json:"gateways"`
}

// overallHealth folds every gateway into one dot color. A blocked gateway
// (engine down, machine unreachable) is the loud failure, so it wins over
// everything. Otherwise any running gateway means we're serving (green); with
// none running — all stopped or nothing configured — the dot is idle grey.
// healthWarn is deliberately not emitted yet: it needs a per-network slow-rate
// read (the traffic endpoint) the dot does not fetch.
func overallHealth(gws []gwHealth) healthKind {
	if len(gws) == 0 {
		return healthOff
	}
	running := false
	for _, g := range gws {
		if g.Blocked != "" {
			return healthDown
		}
		if g.Status.State == "running" {
			running = true
		}
	}
	if running {
		return healthOK
	}
	return healthOff
}
