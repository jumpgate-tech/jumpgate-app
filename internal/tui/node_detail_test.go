package tui

import (
	"slices"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/exp/teatest/v2"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/apiclient"
	"github.com/valve-tech/jumpgate/internal/catalog"
	"github.com/valve-tech/jumpgate/internal/tui/tuitest"
)

// Phase 5 (U5): node detail, current values and the delta from the previous
// reading; no sparklines or trend graphs.

// nodeFake is box-a on chain 369 with reth and every module on: the message
// board, the EL bootnode, the health watch and Checkpointz.
func nodeFake() *tuitest.Fake {
	f := tuitest.NewFake()
	f.TargetsV = []api.TargetView{{
		ID:   "box-a",
		Wire: &catalog.WireConfig{ChainID: 369, ExecID: "reth", Modules: []catalog.NodeModule{{ID: "msgboard"}}},
		Modules: []catalog.BoxModule{
			{ID: "el-bootnode", ChainID: 369}, {ID: "watch"}, {ID: "checkpointz", ChainID: 369},
		},
	}}
	ok := []api.ModuleApplies{{ChainID: 369, Client: "reth", OK: true}}
	box := []api.ModuleApplies{{ChainID: 369, OK: true}}
	f.CatalogV = []api.CatalogModule{
		{ID: "msgboard", Name: "Message board", Placement: "node", Applies: ok},
		{ID: "el-bootnode", Name: "EL bootnode", Placement: "box", Applies: box},
		{ID: "watch", Name: "Health watch", Placement: "box", Applies: []api.ModuleApplies{{ChainID: 0, OK: true}}},
		{ID: "checkpointz", Name: "Checkpointz", Placement: "box", Applies: box},
	}
	f.ModulesV = map[string]api.ModulesView{"box-a": {Modules: []api.ModuleStatus{
		{ID: "msgboard", Served: true, Accepting: true, Count: 1234},
		{ID: "el-bootnode", Active: true},
		{ID: "watch", Active: true},
		{ID: "checkpointz", Error: "unit failed"},
	}}}
	return f
}

func u64(v uint64) *uint64   { return &v }
func f64(v float64) *float64 { return &v }

// readingBefore and readingNow are two successive status readings of box-a.
func readingBefore() api.NodeStatus {
	return api.NodeStatus{At: testNow.Add(-8 * time.Second), Overall: api.SyncSyncing, HeadLag: u64(40), RefHead: 21_345_718, DiskUsedPct: f64(60),
		Exec:   api.ClientStatus{Active: true, Syncing: true, Head: 21_345_600, Peers: 40, State: api.SyncSyncing},
		Beacon: api.ClientStatus{Active: true, Head: 9_876_500, Distance: 3, Peers: 70, State: api.SyncSynced}}
}

func readingNow() api.NodeStatus {
	return api.NodeStatus{At: testNow.Add(-3 * time.Second), Overall: api.SyncSynced, HeadLag: u64(2), RefHead: 21_345_680, DiskUsedPct: f64(61),
		Exec:   api.ClientStatus{Active: true, Head: 21_345_678, Peers: 48, State: api.SyncSynced},
		Beacon: api.ClientStatus{Active: true, Head: 9_876_543, Peers: 72, State: api.SyncSynced}}
}

// nodeApp opens box-a's node from the tree (tab, expand, down to it, enter)
// with the readings queued on the box's status stream.
func nodeApp(t *testing.T, f *tuitest.Fake, w, h int, node string, rs ...api.NodeStatus) tea.Model {
	t.Helper()
	for _, r := range rs {
		f.Status("box-a") <- apiclient.Update[api.NodeStatus]{Value: r, Has: true, State: apiclient.Live}
	}
	keys := []string{"tab", "space", "j"}
	if node == "beacon" {
		keys = append(keys, "j")
	}
	return press(t, dashApp(t, f, w, h, "ascii", "auto"), append(keys, "enter")...)
}

func TestNodeDetailOpensFromTree(t *testing.T) {
	m := nodeApp(t, nodeFake(), 120, 40, "exec", readingBefore(), readingNow())
	h, ok := m.(*App).detail.(*hostScreen)
	if !ok || h.id != "box-a" || h.node != "exec" {
		t.Fatalf("detail %T %+v", m.(*App).detail, h)
	}
	fr := tuitest.Frame(m)
	for _, want := range []string{"box-a / exec", "EXECUTION", "Overview"} {
		if want == "Overview" {
			if strings.Contains(fr, "[Overview]") {
				t.Errorf("node detail shows the box tabs:\n%s", fr)
			}
			continue
		}
		if !strings.Contains(fr, want) {
			t.Errorf("no %q:\n%s", want, fr)
		}
	}
}

func TestNodeDetailDeltas(t *testing.T) {
	fr := tuitest.Frame(nodeApp(t, nodeFake(), 120, 40, "exec", readingBefore(), readingNow()))
	for _, want := range []string{
		"head lag  2 blk (was 40)",
		"peers     48 (was 40)",
		"head      21,345,678 (was 21,345,600)",
		"state     synced (was syncing)",
		"ref head  21,345,680 (was 21,345,718)",
		"disk      61% used (was 60%)",
	} {
		if !strings.Contains(fr, want) {
			t.Errorf("no %q:\n%s", want, fr)
		}
	}
	for _, bad := range []string{"sparkline", "trend"} {
		if strings.Contains(fr, bad) {
			t.Errorf("frame says %q", bad)
		}
	}
}

func TestNodeDetailBeacon(t *testing.T) {
	fr := tuitest.Frame(nodeApp(t, nodeFake(), 120, 40, "beacon", readingBefore(), readingNow()))
	for _, want := range []string{"box-a / beacon", "BEACON", "slot      9,876,543 (was 9,876,500)", "distance  0 (was 3)", "peers     72 (was 70)"} {
		if !strings.Contains(fr, want) {
			t.Errorf("no %q:\n%s", want, fr)
		}
	}
}

// One reading: no "was"; an unchanged value says no "was" either.
func TestNodeDetailFirstReadingHasNoDelta(t *testing.T) {
	fr := tuitest.Frame(nodeApp(t, nodeFake(), 120, 40, "exec", readingNow()))
	if strings.Contains(fr, "(was") || !strings.Contains(fr, "head lag  2 blk") {
		t.Fatalf("first reading:\n%s", fr)
	}
	same := readingNow()
	same.At = testNow.Add(-2 * time.Second)
	fr = tuitest.Frame(nodeApp(t, nodeFake(), 120, 40, "exec", readingNow(), same))
	if strings.Contains(fr, "(was") {
		t.Fatalf("unchanged values show a delta:\n%s", fr)
	}
}

// Missing data is n/a, never 0; a delta from n/a says so.
func TestNodeDetailMissingIsNA(t *testing.T) {
	gone := readingNow()
	gone.At = testNow.Add(-time.Second)
	gone.HeadLag, gone.RefHead, gone.DiskUsedPct = nil, 0, nil
	gone.Exec = api.ClientStatus{State: api.SyncUnavailable}
	fr := tuitest.Frame(nodeApp(t, nodeFake(), 120, 40, "exec", readingNow(), gone))
	for _, want := range []string{"head lag  n/a (was 2)", "head      n/a (was 21,345,678)", "peers     n/a (was 48)", "ref head  n/a", "disk      n/a"} {
		if !strings.Contains(fr, want) {
			t.Errorf("no %q:\n%s", want, fr)
		}
	}
	for _, bad := range []string{"0 blk", "head      0", "peers     0", " 0% used"} {
		if strings.Contains(fr, bad) {
			t.Errorf("unknown shown as 0 (%q):\n%s", bad, fr)
		}
	}
	none := tuitest.Frame(nodeApp(t, nodeFake(), 120, 40, "exec"))
	if !strings.Contains(none, "waiting for the first reading") || strings.Contains(none, "head lag  0") {
		t.Fatalf("no reading:\n%s", none)
	}
}

// An old reading marks every value ~.
func TestNodeDetailStale(t *testing.T) {
	old := readingNow()
	old.At = testNow.Add(-time.Minute)
	fr := tuitest.Frame(nodeApp(t, nodeFake(), 120, 40, "exec", old))
	for _, want := range []string{"head lag  2 blk~", "peers     48~", "stale: 1m old"} {
		if !strings.Contains(fr, want) {
			t.Errorf("no %q:\n%s", want, fr)
		}
	}
}

func TestNodeDetailModules(t *testing.T) {
	fr := tuitest.Frame(nodeApp(t, nodeFake(), 120, 40, "exec", readingNow()))
	for _, want := range []string{"MODULES", "Message board", "EL bootnode", "Health watch", "Checkpointz", "Running", "unit failed"} {
		if !strings.Contains(fr, want) {
			t.Errorf("no %q:\n%s", want, fr)
		}
	}
	f := nodeFake()
	f.ModulesV["box-a"] = api.ModulesView{GateStale: true, Modules: f.ModulesV["box-a"].Modules}
	if fr := tuitest.Frame(nodeApp(t, f, 120, 40, "exec", readingNow())); !strings.Contains(fr, gateStaleText) {
		t.Fatalf("a stale gate is not said:\n%s", fr)
	}
}

// Esc goes back to the box's tabs, then closes the box.
func TestNodeDetailEscBack(t *testing.T) {
	m := nodeApp(t, nodeFake(), 120, 40, "exec", readingNow())
	m = press(t, m, "tab", "tab") // focus back on main
	if m.(*App).focus != focusMain {
		t.Fatalf("focus %v", m.(*App).focus)
	}
	m = press(t, m, "esc")
	h, ok := m.(*App).detail.(*hostScreen)
	if !ok || h.node != "" || !strings.Contains(tuitest.Frame(m), "[Overview]") {
		t.Fatalf("esc: %T\n%s", m.(*App).detail, tuitest.Frame(m))
	}
	if m = press(t, m, "esc"); m.(*App).detail != nil {
		t.Fatal("second esc kept the box open")
	}
}

// Node detail asks the backend nothing the box detail does not.
func TestNodeDetailNoNewPolling(t *testing.T) {
	names := func(calls []string) []string {
		var out []string
		for _, c := range calls {
			n, _, _ := strings.Cut(c, " ")
			if !slices.Contains(out, n) {
				out = append(out, n)
			}
		}
		slices.Sort(out)
		return out
	}
	fb := nodeFake()
	press(t, dashApp(t, fb, 120, 40, "ascii", "auto"), "tab", "space", "enter")
	fn := nodeFake()
	nodeApp(t, fn, 120, 40, "exec", readingNow())
	if b, n := names(fb.Calls), names(fn.Calls); !slices.Equal(b, n) {
		t.Fatalf("box detail asks %v, node detail %v", b, n)
	}
}

func TestNodeDetailGoldens(t *testing.T) {
	for _, c := range []struct {
		name string
		w, h int
	}{
		{"dash_node_detail_80x24", 80, 24},
		{"dash_node_detail_120x40", 120, 40},
		{"dash_node_detail_160x48", 160, 48},
	} {
		t.Run(c.name, func(t *testing.T) {
			var m tea.Model
			if c.w < leftMinWidth {
				f := nodeFake()
				f.Status("box-a") <- apiclient.Update[api.NodeStatus]{Value: readingBefore(), Has: true, State: apiclient.Live}
				f.Status("box-a") <- apiclient.Update[api.NodeStatus]{Value: readingNow(), Has: true, State: apiclient.Live}
				m = press(t, dashApp(t, f, c.w, c.h, "ascii", "auto"), "[", "space", "j", "enter")
			} else {
				m = nodeApp(t, nodeFake(), c.w, c.h, "exec", readingBefore(), readingNow())
			}
			fr := tuitest.Frame(m)
			requireASCII(t, fr)
			if n := len(strings.Split(fr, "\n")); n != c.h {
				t.Errorf("%d lines", n)
			}
			if !strings.Contains(fr, "(was 40)") {
				t.Errorf("no delta:\n%s", fr)
			}
			tuitest.Golden(t, c.name, fr)
		})
	}
}

// The real program loop: a new reading moves the current value and the
// previous one becomes the "was".
func TestNodeDetailProgram(t *testing.T) {
	f := nodeFake()
	f.FleetCh <- apiclient.Update[api.Fleet]{Value: dashFleet(), Has: true, State: apiclient.Live}
	f.Status("box-a") <- apiclient.Update[api.NodeStatus]{Value: readingBefore(), Has: true, State: apiclient.Live}
	a := New(Options{Backend: f, GOOS: "linux", Getenv: func(string) string { return "" }, Now: func() time.Time { return testNow }})
	tm := teatest.NewTestModel(t, a, teatest.WithInitialTermSize(120, 40), teatest.WithProgramOptions(tea.WithColorProfile(colorprofile.ASCII)))
	teatest.WaitFor(t, tm.Output(), func(b []byte) bool { return strings.Contains(string(b), "box-a") }, teatest.WithDuration(3*time.Second))
	tm.Send(tuitest.Key("tab"))
	tm.Send(tuitest.Key("space"))
	teatest.WaitFor(t, tm.Output(), func(b []byte) bool { return strings.Contains(string(b), "exec") }, teatest.WithDuration(3*time.Second))
	tm.Send(tuitest.Key("j"))
	tm.Send(tuitest.Key("enter"))
	teatest.WaitFor(t, tm.Output(), func(b []byte) bool { return strings.Contains(string(b), "head lag  40 blk") }, teatest.WithDuration(3*time.Second))
	f.Status("box-a") <- apiclient.Update[api.NodeStatus]{Value: readingNow(), Has: true, State: apiclient.Live}
	teatest.WaitFor(t, tm.Output(), func(b []byte) bool { return strings.Contains(string(b), "2 blk (was 40)") }, teatest.WithDuration(3*time.Second))
	tm.Send(tuitest.Key("ctrl+c"))
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}
