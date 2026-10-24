package tui

import (
	"cmp"
	"fmt"
	"slices"
	"time"

	"github.com/valve-tech/jumpgate/internal/api"
)

// Alerts, v1 (spec §3, U2): derived by the TUI from what it already holds.
// When the health-watch module provides a feed, an alertSource reading it
// replaces derivedAlerts and keeps it as the fallback.

type alertSev int

const (
	sevWarn alertSev = iota
	sevFail
)

func (s alertSev) String() string {
	if s == sevFail {
		return "FAIL"
	}
	return "warn"
}

// alert is one problem: Box is the box or gateway it is about; Text is
// sanitized and its free text masked, ready to draw.
type alert struct {
	Sev       alertSev
	Box, Text string
}

// alertInput is what the App holds; the Has flags say a reading exists.
type alertInput struct {
	Fleet       api.Fleet
	FleetHas    bool
	Gateways    []api.GatewaySummary
	GatewaysHas bool
	Retrying    bool // the fleet stream is reconnecting: every value is stale
	Now         time.Time
}

// alertSource is where the alerts come from. known is false when there is
// nothing to judge yet, so the count shows n/a rather than 0.
type alertSource interface {
	alerts(in alertInput) (list []alert, known bool)
}

// derivedAlerts is the v1 source: alerts() over the fleet and gateways.
type derivedAlerts struct{}

func (derivedAlerts) alerts(in alertInput) ([]alert, bool) {
	if !in.FleetHas {
		return nil, false
	}
	return alerts(in), true
}

// Thresholds for the alerts read from a status (spec U6).
const (
	diskWarnPct = 85
	diskFailPct = 95
	headLagWarn = 64 // blocks behind the public head
)

func alertText(s string) string { return maskText(sanitizeLine(s)) }

// alerts is the derived list, FAIL first, then by box and text. Pure: the
// same input gives the same list.
func alerts(in alertInput) []alert {
	var out []alert
	add := func(sev alertSev, box, text string) {
		out = append(out, alert{sev, sanitizeLine(box), text})
	}
	every := time.Duration(in.Fleet.IntervalSeconds) * time.Second
	for _, r := range in.Fleet.Rows {
		if !r.Reachable && r.LastSeen != nil {
			// Its error and stale sections are why it is down: one alert.
			add(sevFail, r.ID, "down "+formatAge(in.Now.Sub(*r.LastSeen)))
			continue
		}
		if r.Error != nil && r.Error.Code != api.CodeTargetNotSetUp {
			text := alertText(r.Error.Message)
			if r.Error.Code != "" {
				text = sanitizeLine(string(r.Error.Code)) + ": " + text
			}
			add(sevFail, r.ID, text)
		}
		for _, pe := range []struct {
			name string
			e    *api.ProbeError
		}{{"status", r.StatusError}, {"disk", r.DiskError}, {"firewall", r.FirewallError}} {
			if pe.e != nil {
				add(sevFail, r.ID, pe.name+": "+alertText(pe.e.Message))
			}
		}
		statusOld := r.Status != nil && !r.Status.At.IsZero() && every > 0 && in.Now.Sub(r.Status.At) > 2*every
		if r.Stale.Status || statusOld {
			add(sevWarn, r.ID, "status stale")
		}
		if r.Stale.Disk {
			add(sevWarn, r.ID, "disk stale")
		}
		if r.Stale.Firewall {
			add(sevWarn, r.ID, "firewall stale")
		}
		if r.Firewall != nil {
			switch r.Firewall.Grade {
			case "fail":
				add(sevFail, r.ID, "firewall fail")
			case "warn":
				add(sevWarn, r.ID, "firewall warn")
			}
		}
		if r.Disk != nil {
			switch r.Disk.Fit {
			case api.FitShort:
				add(sevFail, r.ID, "disk short")
			case api.FitTight:
				add(sevWarn, r.ID, "disk tight")
			}
		}
		if st := r.Status; st != nil {
			if p := st.DiskUsedPct; p != nil && *p >= diskWarnPct {
				sev := sevWarn
				if *p >= diskFailPct {
					sev = sevFail
				}
				add(sev, r.ID, fmt.Sprintf("disk %.0f%% used", *p))
			}
			if st.Exec.State != "" && st.Exec.State != api.SyncUnavailable && st.Exec.Peers == 0 {
				add(sevWarn, r.ID, "exec peers 0")
			}
			if st.HeadLag != nil && *st.HeadLag > headLagWarn {
				add(sevWarn, r.ID, fmt.Sprintf("head lag %d blk", *st.HeadLag))
			}
		}
		if r.Jobs != nil && r.Jobs.NeedsDecision > 0 {
			add(sevWarn, r.ID, fmt.Sprintf("%d needs decision", r.Jobs.NeedsDecision))
		}
	}
	if in.GatewaysHas {
		for _, g := range in.Gateways {
			name := g.Label
			if name == "" {
				name = g.ID
			}
			if st := g.Status.State; st != "" && st != "running" {
				add(sevFail, name, "gateway "+alertText(st))
			}
			for _, w := range g.Warnings {
				add(sevWarn, name, alertText(w))
			}
		}
	}
	slices.SortStableFunc(out, func(a, b alert) int {
		return cmp.Or(cmp.Compare(b.Sev, a.Sev), cmp.Compare(a.Box, b.Box), cmp.Compare(a.Text, b.Text))
	})
	return out
}
