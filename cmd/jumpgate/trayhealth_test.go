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
