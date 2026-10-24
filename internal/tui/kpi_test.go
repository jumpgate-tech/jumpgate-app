package tui

import (
	"regexp"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/tui/tuitest"
)

func tileByTitle(t *testing.T, ts []tile, title string) tile {
	t.Helper()
	for _, x := range ts {
		if x.Title == title {
			return x
		}
	}
	t.Fatalf("no tile %q in %+v", title, ts)
	return tile{}
}

func TestKPIsSampleFleet(t *testing.T) {
	in := dashInput()
	in.Gateways, in.GatewaysHas = []api.GatewaySummary{gw("rpc-main", "running")}, true
	al := alerts(in)
	ts := kpis(in, al, true)
	if len(ts) != 7 {
		t.Fatalf("%d tiles", len(ts))
	}
	for _, c := range []struct{ title, short, l0, l1 string }{
		{"Boxes", "boxes 1/3 up", "1 up 1 down", "1 n/a of 3"},
		{"Synced", "synced ~1/3", "~1/3 synced", "1 sync 1 n/a"},
		{"Head lag", "lag 2", "max 2 blk", "2 n/a"},
		{"Peers", "peers ~51/81", "~51 exec", "~81 bcn 1 n/a"},
		{"Disk", "disk 39%", "min 39% free", "on box-a"},
		{"Gateways", "gw 1/1 live", "1 live 0 failing", "n/a no request counts yet"},
		{"Alerts", "! 1 FAIL 2 warn", "1 FAIL 2 warn", "box-b down 4m"},
	} {
		x := tileByTitle(t, ts, c.title)
		if x.Short != c.short || x.Lines[0] != c.l0 || x.Lines[1] != c.l1 {
			t.Errorf("%s: %q %q %q, want %q %q %q", c.title, x.Short, x.Lines[0], x.Lines[1], c.short, c.l0, c.l1)
		}
	}
	in.Gateways, in.GatewaysHas = nil, false
	x := tileByTitle(t, kpis(in, al, true), "Gateways")
	if x.Short != "gw n/a" || x.Lines[0] != "n/a" || x.Lines[1] != "n/a no request counts yet" {
		t.Errorf("no gateways reply: %+v", x)
	}
}

var bareZero = regexp.MustCompile(`(^|[^0-9.])0([^0-9.%]|$)`)

func TestKPIsUnknownNeverZero(t *testing.T) {
	in := alertInput{FleetHas: true, Now: testNow, Fleet: api.Fleet{At: testNow, IntervalSeconds: 15, Rows: []api.FleetRow{{ID: "a"}, {ID: "b"}, {ID: "c"}}}}
	al, known := derivedAlerts{}.alerts(in)
	for _, x := range kpis(in, al, known) {
		all := x.Short + " " + x.Lines[0] + " " + x.Lines[1]
		if !strings.Contains(all, "n/a") && x.Title != "Alerts" {
			t.Errorf("%s lacks n/a: %q", x.Title, all)
		}
		if x.Title != "Alerts" && bareZero.MatchString(all) {
			t.Errorf("%s shows 0 for unknown: %q", x.Title, all)
		}
	}
	for _, x := range kpis(alertInput{}, nil, false) {
		all := x.Short + " " + x.Lines[0] + " " + x.Lines[1]
		if !strings.Contains(all, "n/a") || bareZero.MatchString(all) {
			t.Errorf("before the fleet %s: %q", x.Title, all)
		}
	}
}

func TestKPIsFailFromSectionError(t *testing.T) {
	in := alertInput{FleetHas: true, Now: testNow, Fleet: api.Fleet{At: testNow, IntervalSeconds: 15, Rows: []api.FleetRow{
		{ID: "a", Reachable: true, DiskError: &api.ProbeError{Error: api.Error{Code: "disk_probe", Message: "x"}},
			StatusError: &api.ProbeError{Error: api.Error{Code: "rpc", Message: "y"}}},
	}}}
	ts := kpis(in, alerts(in), true)
	if x := tileByTitle(t, ts, "Disk"); x.Short != "disk FAIL disk_probe" {
		t.Errorf("disk %+v", x)
	}
	if x := tileByTitle(t, ts, "Synced"); x.Short != "synced FAIL rpc" {
		t.Errorf("synced %+v", x)
	}
}

func TestKPIGatewaysHint(t *testing.T) {
	if x := tileByTitle(t, kpis(dashInput(), nil, true), "Gateways"); !strings.Contains(x.Lines[1], "no request counts yet") {
		t.Fatalf("%+v", x)
	}
}

func TestKPIViewFits(t *testing.T) {
	a := dashApp(t, tuitest.NewFake(), 120, 40, "ascii", "auto")
	for w := 80; w <= 200; w += 7 {
		for _, h := range []int{1, 4} {
			v := ansi.Strip(a.kpiView(rect{0, 1, w, h}))
			lines := strings.Split(v, "\n")
			if len(lines) != h {
				t.Fatalf("%dx%d: %d lines:\n%s", w, h, len(lines), v)
			}
			for _, l := range lines {
				if ansi.StringWidth(l) > w {
					t.Fatalf("%dx%d: wider: %q", w, h, l)
				}
			}
			requireASCII(t, v)
		}
	}
}
