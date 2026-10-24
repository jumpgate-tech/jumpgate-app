package tui

import (
	"errors"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/apiclient"
	"github.com/valve-tech/jumpgate/internal/tui/tuitest"
)

// dashRunning is the sample inbox's unfinished jobs as GET /api/jobs/running
// has them: box-a asking at step 2 of 3, box-b running step 4 of 8.
func dashRunning() api.RunningJobs {
	return api.RunningJobs{At: testNow, Jobs: []api.RunningJob{
		{Target: "box-a", ID: "0123456789abcdef", Kind: "node.migrate-names", State: "needs-decision", Step: 1, Total: 3, NeedsDecision: true},
		{Target: "box-b", ID: "fedcba9876543210", Kind: "node.migrate-names", State: "running", Step: 3, Total: 8},
	}}
}

// A fleet reading asks for the running jobs (the cache route), never the
// inbox stream, which would have the server list every box.
func TestFleetReadingAsksForRunningJobs(t *testing.T) {
	f := tuitest.NewFake()
	f.RunningV = dashRunning()
	a := newTestApp(t, f, 120, 40, "ascii")
	_, cmds := tuitest.Send(a, fleetMsg{u: apiclient.Update[api.Fleet]{Value: dashFleet(), Has: true, State: apiclient.Live}})
	var got []any
	for _, c := range cmds {
		for _, m := range tuitest.Run(c) {
			got = append(got, m)
		}
	}
	found := false
	for _, m := range got {
		if r, ok := m.(runningMsg); ok && r.err == nil && len(r.v.Jobs) == 2 {
			found = true
		}
	}
	if !found || !f.Called("RunningJobs") {
		t.Fatalf("no running jobs read: %v %#v", f.Calls, got)
	}
	if f.Called("WatchJobs") || f.Called("Jobs") {
		t.Fatalf("the fleet view watched the inbox: %v", f.Calls)
	}
}

// Off the Jobs view the drawer draws job bars from the running jobs; a
// failed read keeps them, marked stale, and the next one clears the mark.
func TestDrawerJobBarsOnEveryView(t *testing.T) {
	a := dashApp(t, tuitest.NewFake(), 120, 40, "ascii", "open")
	if a.active == scrJobs {
		t.Fatal("on the Jobs view")
	}
	v := ansi.Strip(a.drawerView(a.layout().Drawer, false))
	for _, want := range []string{"job  box-a      migrate-names  step 2/3     [#######.............] ! needs decision",
		"job  box-b      migrate-names  step 4/8     [########............]"} {
		if !strings.Contains(v, want) {
			t.Errorf("no %q:\n%s", want, v)
		}
	}
	m, _ := tuitest.Send(a, runningMsg{err: errors.New("refused")})
	v = ansi.Strip(m.(*App).drawerView(a.layout().Drawer, false))
	if !strings.Contains(v, "step 4/8     [########............] ~ stale") {
		t.Errorf("a failed read is not stale:\n%s", v)
	}
	m, _ = tuitest.Send(m, runningMsg{v: dashRunning()})
	if v = ansi.Strip(m.(*App).drawerView(a.layout().Drawer, false)); strings.Contains(v, "stale") {
		t.Errorf("still stale:\n%s", v)
	}
}
