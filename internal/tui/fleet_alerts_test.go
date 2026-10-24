package tui

import (
	"strings"
	"testing"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/apiclient"
	"github.com/valve-tech/jumpgate/internal/tui/tuitest"
)

// A box whose health watch has open alerts shows their count by its name.
func TestFleetRowShowsAlertCount(t *testing.T) {
	f := sampleFleet()
	f.Rows[0].AlertCount, f.Rows[0].AlertMessage = 2, "Disk is 93% full."
	a := newTestApp(t, tuitest.NewFake(), 80, 24, "unicode")
	p := tuitest.NewFake().PrefsV
	p.Glyphs = "unicode"
	m, _ := tuitest.Send(a, prefsMsg{p: p}, fleetMsg{u: apiclient.Update[api.Fleet]{Value: f, Has: true, State: apiclient.Live}})
	fr := tuitest.Frame(m.(*App))
	found := false
	for _, line := range strings.Split(fr, "\n") {
		if strings.Contains(line, "box-a !2") {
			found = true
		}
		if strings.Contains(line, "box-b !") {
			t.Errorf("box-b shows an alert: %q", line)
		}
	}
	if !found {
		t.Errorf("box-a has no alert count:\n%s", fr)
	}
	tuitest.Golden(t, "fleet_alert", fr)
}
