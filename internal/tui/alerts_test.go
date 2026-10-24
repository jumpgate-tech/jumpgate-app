package tui

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/valve-tech/jumpgate/internal/api"
)

func dashInput() alertInput {
	return alertInput{Fleet: dashFleet(), FleetHas: true, Now: testNow}
}

func TestAlertsSampleFleet(t *testing.T) {
	got := alerts(dashInput())
	want := []alert{
		{sevFail, "box-b", "down 4m"},
		{sevWarn, "box-a", "1 needs decision"},
		{sevWarn, "box-a", "firewall warn"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("alerts\n got %+v\nwant %+v", got, want)
	}
}

func TestAlertsEachRule(t *testing.T) {
	seen := testNow.Add(-10 * time.Second)
	up := func(mod func(*api.FleetRow)) api.Fleet {
		r := api.FleetRow{ID: "b", Link: api.LinkAgent, Reachable: true, LastSeen: &seen}
		mod(&r)
		return api.Fleet{At: testNow, IntervalSeconds: 15, Rows: []api.FleetRow{r}}
	}
	for _, c := range []struct {
		name string
		fl   api.Fleet
		gws  []api.GatewaySummary
		sev  alertSev
		text string
	}{
		{"down", up(func(r *api.FleetRow) { r.Reachable = false }), nil, sevFail, "down 10s"},
		{"error", up(func(r *api.FleetRow) { r.Error = &api.Error{Code: "boom", Message: "it broke"} }), nil, sevFail, "boom: it broke"},
		{"status error", up(func(r *api.FleetRow) { r.StatusError = &api.ProbeError{Error: api.Error{Message: "rpc timeout"}} }), nil, sevFail, "status: rpc timeout"},
		{"disk error", up(func(r *api.FleetRow) { r.DiskError = &api.ProbeError{Error: api.Error{Message: "du failed"}} }), nil, sevFail, "disk: du failed"},
		{"firewall error", up(func(r *api.FleetRow) { r.FirewallError = &api.ProbeError{Error: api.Error{Message: "nft"}} }), nil, sevFail, "firewall: nft"},
		{"disk stale", up(func(r *api.FleetRow) { r.Stale.Disk = true }), nil, sevWarn, "disk stale"},
		{"firewall fail", up(func(r *api.FleetRow) { r.Firewall = &api.FirewallSummary{Fail: 1, Grade: "fail"} }), nil, sevFail, "firewall fail"},
		{"firewall warn", up(func(r *api.FleetRow) { r.Firewall = &api.FirewallSummary{Warn: 1, Grade: "warn"} }), nil, sevWarn, "firewall warn"},
		{"disk short", up(func(r *api.FleetRow) { r.Disk = &api.DiskView{Fit: api.FitShort} }), nil, sevFail, "disk short"},
		{"disk tight", up(func(r *api.FleetRow) { r.Disk = &api.DiskView{Fit: api.FitTight} }), nil, sevWarn, "disk tight"},
		{"decision", up(func(r *api.FleetRow) { r.Jobs = &api.JobsSummary{Running: 1, NeedsDecision: 2} }), nil, sevWarn, "2 needs decision"},
		{"gateway warning", up(func(*api.FleetRow) {}), []api.GatewaySummary{gw("rpc-main", "running", "upstream slow")}, sevWarn, "upstream slow"},
		{"gateway state", up(func(*api.FleetRow) {}), []api.GatewaySummary{gw("rpc-main", "stopped")}, sevFail, "gateway stopped"},
	} {
		got := alerts(alertInput{Fleet: c.fl, FleetHas: true, Gateways: c.gws, GatewaysHas: c.gws != nil, Now: testNow})
		if len(got) != 1 || got[0].Sev != c.sev || got[0].Text != c.text {
			t.Errorf("%s: %+v, want one %v %q", c.name, got, c.sev, c.text)
		}
	}
}

func gw(label, state string, warnings ...string) api.GatewaySummary {
	var g api.GatewaySummary
	g.ID, g.Label, g.Status.State, g.Warnings = label, label, state, warnings
	return g
}

func TestAlertsStaleTwoIntervals(t *testing.T) {
	for _, c := range []struct {
		age  time.Duration
		want int
	}{{31 * time.Second, 1}, {29 * time.Second, 0}} {
		in := alertInput{FleetHas: true, Now: testNow, Fleet: api.Fleet{At: testNow, IntervalSeconds: 15, Rows: []api.FleetRow{
			{ID: "b", Reachable: true, Status: &api.NodeStatus{At: testNow.Add(-c.age), Overall: api.SyncSynced}},
		}}}
		got := alerts(in)
		if len(got) != c.want || (c.want == 1 && got[0].Text != "status stale") {
			t.Errorf("age %v: %+v", c.age, got)
		}
	}
}

func TestAlertsGatewayWarningMasked(t *testing.T) {
	in := alertInput{FleetHas: true, Now: testNow, GatewaysHas: true,
		Gateways: []api.GatewaySummary{gw("g", "running", "upstream https://u:p@h.example/key123 failed\x1b[31m")}}
	got := alerts(in)
	if len(got) != 1 || strings.Contains(got[0].Text, "u:p") || strings.Contains(got[0].Text, "key123") || strings.Contains(got[0].Text, "\x1b") {
		t.Fatalf("%+v", got)
	}
}

func TestAlertsUnknownWithoutFleet(t *testing.T) {
	if _, known := (derivedAlerts{}).alerts(alertInput{}); known {
		t.Fatal("alerts known before any fleet")
	}
	if l, known := (derivedAlerts{}).alerts(dashInput()); !known || len(l) != 3 {
		t.Fatalf("%v %+v", known, l)
	}
}

func TestAlertsDeterministic(t *testing.T) {
	in := dashInput()
	a := alerts(in)
	rows := in.Fleet.Rows
	rows[0], rows[2] = rows[2], rows[0]
	if b := alerts(in); !reflect.DeepEqual(a, b) {
		t.Fatalf("%+v != %+v", a, b)
	}
}
