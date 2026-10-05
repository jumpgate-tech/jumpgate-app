package main

import "testing"

func TestOverallHealth(t *testing.T) {
	run := gwHealth{Status: struct{ State string }{State: "running"}}
	stopped := gwHealth{Status: struct{ State string }{State: "exited"}}
	blocked := gwHealth{Blocked: "docker is not reachable"}

	cases := []struct {
		name string
		gws  []gwHealth
		want healthKind
	}{
		{"none configured", nil, healthOff},
		{"one running", []gwHealth{run}, healthOK},
		{"all stopped", []gwHealth{stopped, stopped}, healthOff},
		{"one blocked wins over running", []gwHealth{run, blocked}, healthDown},
		{"blocked alone", []gwHealth{blocked}, healthDown},
		{"mixed running and stopped is serving", []gwHealth{stopped, run}, healthOK},
	}
	for _, tc := range cases {
		if got := overallHealth(tc.gws); got != tc.want {
			t.Errorf("%s: overallHealth = %d, want %d", tc.name, got, tc.want)
		}
	}
}

func TestHealthTooltip(t *testing.T) {
	for k, want := range map[healthKind]string{
		healthOff: "Jumpgate — idle", healthOK: "Jumpgate — serving",
		healthWarn: "Jumpgate — degraded", healthDown: "Jumpgate — a gateway is unavailable",
	} {
		if got := healthTooltip(k); got != want {
			t.Errorf("healthTooltip(%d) = %q, want %q", k, got, want)
		}
	}
}

func TestTrayActionFor(t *testing.T) {
	if trayActionFor(1) != trayOpen || trayActionFor(2) != trayQuit || trayActionFor(0) != trayNone || trayActionFor(99) != trayNone {
		t.Fatal("tray menu ids map wrongly")
	}
}
