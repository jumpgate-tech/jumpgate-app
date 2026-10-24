package tui

import (
	"testing"

	"github.com/valve-tech/jumpgate/internal/api"
)

func barRow(id string) api.FleetRow {
	return api.FleetRow{ID: id, Link: api.LinkAgent, Reachable: true, Status: &api.NodeStatus{Overall: api.SyncSynced,
		Exec: api.ClientStatus{State: api.SyncSynced}, Beacon: api.ClientStatus{State: api.SyncSynced}}}
}

func pctRow(id string, pct float64) api.FleetRow {
	r := barRow(id)
	r.Status.DiskUsedPct = &pct
	return r
}

func jobFor(target, state string, step, n int) api.JobView {
	return api.JobView{Target: target, ID: target + state, Kind: "node.migrate-names", State: state, Step: step, Steps: make([]api.JobStepView, n)}
}

func kinds(bs []progressBar) []barKind {
	var out []barKind
	for _, b := range bs {
		out = append(out, b.Kind)
	}
	return out
}

func TestDrawerBarsOrder(t *testing.T) {
	sync := barRow("box-b")
	sync.Status.Exec = api.ClientStatus{State: api.SyncSyncing, Head: 50}
	sync.Status.RefHead = 100
	in := barInput{
		Fleet: api.Fleet{Rows: []api.FleetRow{pctRow("box-c", 91), sync, pctRow("box-d", 96)}},
		Jobs:  []api.JobView{jobFor("box-a", "running", 1, 4), jobFor("box-a", "needs-decision", 1, 3), jobFor("box-a", "done", 3, 3)}, JobsHas: true,
	}
	bs := drawerBars(in)
	want := []barKind{barDecision, barJob, barSync, barDisk, barDisk}
	if got := kinds(bs); len(got) != len(want) {
		t.Fatalf("kinds %v, want %v", got, want)
	}
	for i := range want {
		if bs[i].Kind != want[i] {
			t.Fatalf("kinds %v, want %v", kinds(bs), want)
		}
	}
	if bs[0].Note != "step 2/3" || bs[0].What != "migrate-names" || bs[0].Done != 1 || bs[0].Total != 3 {
		t.Errorf("decision bar %+v", bs[0])
	}
	if bs[3].Box != "box-d" || bs[4].Box != "box-c" {
		t.Errorf("disk not fullest first: %+v %+v", bs[3], bs[4])
	}
}

func TestDrawerBarsNoDenominator(t *testing.T) {
	exec := barRow("box-a")
	exec.Status.Exec = api.ClientStatus{State: api.SyncSyncing, Head: 50} // RefHead 0
	beacon := barRow("box-b")
	beacon.Status.Beacon = api.ClientStatus{State: api.SyncSyncing, Head: 900} // Distance 0
	unknown := barRow("box-c")
	unknown.Status = nil
	in := barInput{Fleet: api.Fleet{Rows: []api.FleetRow{exec, beacon, unknown}}, Jobs: []api.JobView{jobFor("box-a", "running", 0, 0)}, JobsHas: true}
	if bs := drawerBars(in); len(bs) != 0 {
		t.Fatalf("bars without a denominator: %+v", bs)
	}
}

func TestDrawerBarsDiskThreshold(t *testing.T) {
	if bs := drawerBars(barInput{Fleet: api.Fleet{Rows: []api.FleetRow{pctRow("box-a", 84)}}}); len(bs) != 0 {
		t.Fatalf("84%%: %+v", bs)
	}
	bs := drawerBars(barInput{Fleet: api.Fleet{Rows: []api.FleetRow{pctRow("box-a", 85)}}})
	if len(bs) != 1 || bs[0].Note != "85% used" || bs[0].Done != 85 || bs[0].Total != 100 {
		t.Fatalf("85%%: %+v", bs)
	}
}

func TestDrawerBarsSync(t *testing.T) {
	r := barRow("box-a")
	r.Status.Exec = api.ClientStatus{State: api.SyncSyncing, Head: 21_100_000}
	r.Status.RefHead = 21_300_000
	r.Status.Beacon = api.ClientStatus{State: api.SyncSyncing, Head: 900, Distance: 100}
	bs := drawerBars(barInput{Fleet: api.Fleet{Rows: []api.FleetRow{r}}})
	if len(bs) != 2 || bs[0].What != "exec" || bs[0].Note != "21.1M/21.3M" || bs[1].What != "beacon" || bs[1].Done != 900 || bs[1].Total != 1000 {
		t.Fatalf("%+v", bs)
	}
}

func TestDrawerBarsStale(t *testing.T) {
	r := barRow("box-a")
	r.Status.Exec = api.ClientStatus{State: api.SyncSyncing, Head: 1}
	r.Status.RefHead = 2
	stale := r
	stale.Stale = api.Stale{Status: true}
	jobs := []api.JobView{jobFor("box-a", "running", 1, 2)}
	for name, c := range map[string]struct {
		in   barInput
		want bool
	}{
		"fresh":      {barInput{Fleet: api.Fleet{Rows: []api.FleetRow{r}}, Jobs: jobs, JobsHas: true}, false},
		"section":    {barInput{Fleet: api.Fleet{Rows: []api.FleetRow{stale}}}, true},
		"fleet":      {barInput{Fleet: api.Fleet{Rows: []api.FleetRow{r}}, FleetStale: true}, true},
		"jobsStream": {barInput{Jobs: jobs, JobsHas: true, JobsStale: true}, true},
	} {
		bs := drawerBars(c.in)
		if len(bs) == 0 {
			t.Fatalf("%s: no bars", name)
		}
		for _, b := range bs {
			if b.Stale != c.want {
				t.Errorf("%s: %+v stale %v, want %v", name, b, b.Stale, c.want)
			}
		}
	}
}

func TestCapBars(t *testing.T) {
	bs := make([]progressBar, 8)
	for _, c := range []struct{ n, cap, shown, more int }{{8, 3, 3, 5}, {8, 6, 6, 2}, {2, 6, 2, 0}, {3, 3, 3, 0}} {
		got, more := capBars(bs[:c.n], c.cap)
		if len(got) != c.shown || more != c.more {
			t.Errorf("%d cap %d: %d shown, %d more", c.n, c.cap, len(got), more)
		}
	}
}
