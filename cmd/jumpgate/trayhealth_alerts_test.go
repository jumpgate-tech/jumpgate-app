package main

import (
	"testing"
	"time"

	"github.com/valve-tech/jumpgate/internal/api"
)

func TestOverallHealthWarnsOnAlerts(t *testing.T) {
	run := gwHealth{Status: struct{ State string }{State: "running"}}
	blocked := gwHealth{Blocked: "docker is not reachable"}
	if got := overallHealth([]gwHealth{run}, 1); got != healthWarn {
		t.Errorf("running + 1 alert = %d, want warn", got)
	}
	if got := overallHealth(nil, 2); got != healthWarn {
		t.Errorf("no gateways + alerts = %d, want warn", got)
	}
	if got := overallHealth([]gwHealth{run, blocked}, 3); got != healthDown {
		t.Errorf("blocked + alerts = %d, want down", got)
	}
	if got := overallHealth([]gwHealth{run}, 0); got != healthOK {
		t.Errorf("running, no alerts = %d, want ok", got)
	}
}

func TestSummarizeFleetAndTip(t *testing.T) {
	now := time.Now()
	notSetUp := &api.Error{Code: api.CodeTargetNotSetUp}
	f := api.Fleet{Rows: []api.FleetRow{
		{ID: "box-1", Reachable: true, CheckedAt: &now, AlertCount: 2},
		{ID: "box-2", Reachable: true, CheckedAt: &now},
		{ID: "fresh", Reachable: false}, // never checked: not counted
		{ID: "bare", Reachable: false, CheckedAt: &now, Error: notSetUp},
	}}
	s := summarizeFleet(f)
	if s.Alerts != 2 || s.Box != "box-1" || s.AlertBoxes != 1 || s.Unreachable != 0 {
		t.Fatalf("summary = %+v", s)
	}
	if got := healthTip(healthWarn, s); got != "Jumpgate — 2 alerts on box-1" {
		t.Errorf("tip = %q", got)
	}
	if got := healthTip(healthWarn, fleetSummary{Alerts: 1, Box: "box-1", AlertBoxes: 1}); got != "Jumpgate — 1 alert on box-1" {
		t.Errorf("singular tip = %q", got)
	}
	if got := healthTip(healthWarn, fleetSummary{Alerts: 3, Box: "a", AlertBoxes: 2}); got != "Jumpgate — 3 alerts on 2 boxes" {
		t.Errorf("multi-box tip = %q", got)
	}

	f.Rows = append(f.Rows, api.FleetRow{ID: "gone", CheckedAt: &now, Error: &api.Error{Code: api.CodeUnreachable}})
	s = summarizeFleet(f)
	if s.Unreachable != 1 || s.Down != "gone" || s.warnings() != 3 {
		t.Fatalf("unreachable summary = %+v", s)
	}
	if got := healthTip(healthWarn, fleetSummary{Unreachable: 1, Down: "gone"}); got != "Jumpgate — gone is not answering" {
		t.Errorf("unreachable tip = %q", got)
	}
	if got := healthTip(healthWarn, fleetSummary{Unreachable: 2, Down: "gone"}); got != "Jumpgate — 2 boxes are not answering" {
		t.Errorf("unreachable tip = %q", got)
	}
	if got := healthTip(healthDown, s); got != healthTooltip(healthDown) {
		t.Errorf("down tip = %q", got)
	}
	if got := healthTip(healthOK, fleetSummary{}); got != "Jumpgate — serving" {
		t.Errorf("ok tip = %q", got)
	}
}
