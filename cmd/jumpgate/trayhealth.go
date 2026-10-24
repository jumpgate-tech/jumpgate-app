package main

// Overall health signal for the menubar status dot. Kept out of the `tray`
// build tag (no CGo, no webview) so the reduction is plain, testable Go; the
// darwin status item (statusitem_darwin.go) and the poller (tray.go) consume it.

import (
	"fmt"

	"github.com/valve-tech/jumpgate/internal/api"
)

// healthKind is the menubar dot's state, worst-first when it matters.
type healthKind int

const (
	healthOff  healthKind = iota // grey: nothing set up, all stopped, or unreachable
	healthOK                     // green: at least one gateway running, none blocked
	healthWarn                   // amber: a health-watch alert, or a box that stopped answering
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

// overallHealth folds every gateway and the fleet's warnings (open
// health-watch alerts plus boxes that stopped answering) into one dot color.
// A blocked gateway (engine down, machine unreachable) is the loud failure,
// so it wins over everything. Then any warning turns the dot amber. Otherwise
// any running gateway means we're serving (green); with none running — all
// stopped or nothing configured — the dot is idle grey.
//
// docs:covers watch-alerts
func overallHealth(gws []gwHealth, alerts int) healthKind {
	running := false
	for _, g := range gws {
		if g.Blocked != "" {
			return healthDown
		}
		if g.Status.State == "running" {
			running = true
		}
	}
	if alerts > 0 {
		return healthWarn
	}
	if running {
		return healthOK
	}
	return healthOff
}

// fleetSummary is what the dot takes from GET /api/fleet: the open alerts
// (and the first box with any), and the boxes that stopped answering.
type fleetSummary struct {
	Alerts, AlertBoxes int
	Box                string // the first box with an alert
	Unreachable        int
	Down               string // the first box that stopped answering
}

// warnings is what turns the dot amber.
func (s fleetSummary) warnings() int { return s.Alerts + s.Unreachable }

// summarizeFleet counts the fleet's alerts and the boxes that were checked
// and did not answer (a box that is not set up is not one).
func summarizeFleet(f api.Fleet) fleetSummary {
	var s fleetSummary
	for _, r := range f.Rows {
		if r.AlertCount > 0 {
			s.Alerts += r.AlertCount
			s.AlertBoxes++
			if s.Box == "" {
				s.Box = r.ID
			}
		}
		if !r.Reachable && r.CheckedAt != nil && (r.Error == nil || r.Error.Code != api.CodeTargetNotSetUp) {
			s.Unreachable++
			if s.Down == "" {
				s.Down = r.ID
			}
		}
	}
	return s
}

// trayHealth is one repaint of the status item: the dot and its hover text.
type trayHealth struct {
	Kind healthKind
	Tip  string
}

// healthTip is healthTooltip with the amber dot's reason:
// "Jumpgate — 2 alerts on box-1", or the box that stopped answering.
func healthTip(k healthKind, s fleetSummary) string {
	if k != healthWarn {
		return healthTooltip(k)
	}
	switch {
	case s.Alerts > 0:
		noun := "alerts"
		if s.Alerts == 1 {
			noun = "alert"
		}
		if s.AlertBoxes > 1 {
			return fmt.Sprintf("Jumpgate — %d %s on %d boxes", s.Alerts, noun, s.AlertBoxes)
		}
		return fmt.Sprintf("Jumpgate — %d %s on %s", s.Alerts, noun, s.Box)
	case s.Unreachable == 1:
		return "Jumpgate — " + s.Down + " is not answering"
	case s.Unreachable > 1:
		return fmt.Sprintf("Jumpgate — %d boxes are not answering", s.Unreachable)
	}
	return healthTooltip(k)
}

// healthTooltip is the status item's hover text, the single source for the
// macOS menubar item (statusitem_darwin.go) and the Windows notification icon
// (statusitem_windows.go).
func healthTooltip(k healthKind) string {
	switch k {
	case healthOK:
		return "Jumpgate — serving"
	case healthWarn:
		return "Jumpgate — degraded"
	case healthDown:
		return "Jumpgate — a gateway is unavailable"
	}
	return "Jumpgate — idle"
}

// trayAction is what a tray menu choice does. The menu ids are fixed here so
// the mapping is tested without a tray build.
type trayAction int

const (
	trayNone trayAction = iota
	trayOpen
	trayQuit
)

func trayActionFor(menuID int) trayAction {
	switch menuID {
	case 1:
		return trayOpen
	case 2:
		return trayQuit
	}
	return trayNone
}
