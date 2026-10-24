package tui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/valve-tech/jumpgate/internal/api"
)

// snap is a snapshot of one fleet row (changed by mod) at testNow, with
// every other kind present and empty.
func snap(mod func(*api.FleetRow)) snapshot {
	seen := testNow.Add(-10 * time.Second)
	lag, pct := uint64(2), 40.0
	r := api.FleetRow{ID: "b", Link: api.LinkAgent, Reachable: true, LastSeen: &seen,
		Status: &api.NodeStatus{At: testNow.Add(-5 * time.Second), Overall: api.SyncSynced, HeadLag: &lag, DiskUsedPct: &pct,
			Exec: api.ClientStatus{State: api.SyncSynced, Head: 100, Peers: 9}, Beacon: api.ClientStatus{State: api.SyncSynced, Head: 50, Peers: 9}}}
	if mod != nil {
		mod(&r)
	}
	return snapshot{Fleet: api.Fleet{At: testNow, IntervalSeconds: 15, Rows: []api.FleetRow{r}}, FleetHas: true,
		JobsHas: true, GatewaysHas: true, TargetsHas: true, Modules: map[string]api.ModulesView{}, Now: testNow}
}

func job(id, state string, step, steps int) api.JobView {
	return api.JobView{Target: "b", ID: id, Kind: "node.migrate-names", State: state, Step: step, Steps: make([]api.JobStepView, steps)}
}

func TestDiffEventsKinds(t *testing.T) {
	type ev struct {
		sev       logSev
		box, node string
		text      string
	}
	later := func(s snapshot) snapshot { s.Now = s.Now.Add(15 * time.Second); return s }
	for _, c := range []struct {
		name       string
		prev, next snapshot
		want       []ev
	}{
		{"gate down", snap(nil), snap(func(r *api.FleetRow) { r.Reachable = false }), []ev{{logFail, "b", "", "gate down"}}},
		{"gate up", snap(func(r *api.FleetRow) { r.Reachable = false }), snap(nil), []ev{{logOK, "b", "", "gate up"}}},
		{"box down over ssh", snap(func(r *api.FleetRow) { r.Link = api.LinkSSHOnly }), snap(func(r *api.FleetRow) { r.Link = api.LinkSSHOnly; r.Reachable = false }), []ev{{logFail, "b", "", "box down"}}},
		{"sync change", snap(func(r *api.FleetRow) { r.Status.Exec.State = api.SyncSyncing }), snap(nil), []ev{{logOK, "b", "exec", "exec synced (was syncing)"}}},
		{"beacon falls behind", snap(nil), snap(func(r *api.FleetRow) { r.Status.Beacon.State = api.SyncSyncing }), []ev{{logWarn, "b", "beacon", "beacon syncing (was synced)"}}},
		{"stalled", snap(func(r *api.FleetRow) { r.Status.Exec.State = api.SyncSyncing }),
			later(snap(func(r *api.FleetRow) {
				r.Status.Exec.State = api.SyncSyncing
				r.Status.At = testNow.Add(10 * time.Second)
			})),
			[]ev{{logWarn, "b", "exec", "exec stalled at 100"}}},
		{"box added", func() snapshot { s := snap(nil); s.Fleet.Rows = nil; return s }(), snap(nil), []ev{{logInfo, "b", "", "box added"}}},
		{"box removed", snap(nil), func() snapshot { s := snap(nil); s.Fleet.Rows = nil; return s }(), []ev{{logInfo, "b", "", "box removed"}}},
		{"paired", func() snapshot { s := snap(nil); s.Targets = []api.TargetView{{ID: "b"}}; return s }(),
			func() snapshot {
				s := snap(nil)
				s.Targets = []api.TargetView{{ID: "b", Agent: &api.AgentView{Address: "0xabc"}}}
				return s
			}(), []ev{{logOK, "b", "", "gate paired: signs for this controller"}}},
		{"job started", snap(nil), func() snapshot { s := snap(nil); s.Jobs = []api.JobView{job("j1", "running", 0, 3)}; return s }(),
			[]ev{{logInfo, "b", "", "job node.migrate-names started"}}},
		{"job step", func() snapshot { s := snap(nil); s.Jobs = []api.JobView{job("j1", "running", 0, 3)}; return s }(),
			func() snapshot { s := snap(nil); s.Jobs = []api.JobView{job("j1", "running", 1, 3)}; return s }(),
			[]ev{{logInfo, "b", "", "job node.migrate-names step 2/3"}}},
		{"job needs decision", func() snapshot { s := snap(nil); s.Jobs = []api.JobView{job("j1", "running", 1, 3)}; return s }(),
			func() snapshot { s := snap(nil); s.Jobs = []api.JobView{job("j1", "needs-decision", 1, 3)}; return s }(),
			[]ev{{logWarn, "b", "", "job node.migrate-names needs decision"}}},
		{"job answered", func() snapshot { s := snap(nil); s.Jobs = []api.JobView{job("j1", "needs-decision", 1, 3)}; return s }(),
			func() snapshot { s := snap(nil); s.Jobs = []api.JobView{job("j1", "running", 1, 3)}; return s }(),
			[]ev{{logOK, "b", "", "job node.migrate-names decision answered"}}},
		{"decision from the fleet count", func() snapshot {
			s := snap(func(r *api.FleetRow) { r.Jobs = &api.JobsSummary{Running: 1} })
			s.JobsHas = false
			return s
		}(),
			func() snapshot {
				s := snap(func(r *api.FleetRow) { r.Jobs = &api.JobsSummary{Running: 1, NeedsDecision: 1} })
				s.JobsHas = false
				return s
			}(), []ev{{logWarn, "b", "", "1 needs decision"}}},
		{"alert raised", snap(nil), snap(func(r *api.FleetRow) { r.Firewall = &api.FirewallSummary{Grade: "warn"} }), []ev{{logWarn, "b", "", "firewall warn"}}},
		{"alert cleared", snap(func(r *api.FleetRow) { r.Firewall = &api.FirewallSummary{Grade: "warn"} }), snap(nil), []ev{{logOK, "b", "", "cleared: firewall warn"}}},
		{"disk 85", snap(nil), snap(func(r *api.FleetRow) { p := 86.0; r.Status.DiskUsedPct = &p }), []ev{{logWarn, "b", "", "disk 86% used"}}},
		{"disk 95", snap(func(r *api.FleetRow) { p := 86.0; r.Status.DiskUsedPct = &p }), snap(func(r *api.FleetRow) { p := 96.0; r.Status.DiskUsedPct = &p }), []ev{{logFail, "b", "", "disk 96% used"}}},
		{"peers 0", snap(nil), snap(func(r *api.FleetRow) { r.Status.Exec.Peers = 0 }), []ev{{logWarn, "b", "", "exec peers 0"}}},
		{"head lag", snap(nil), snap(func(r *api.FleetRow) { l := uint64(65); r.Status.HeadLag = &l }), []ev{{logWarn, "b", "", "head lag 65 blk"}}},
		{"gateway state", func() snapshot { s := snap(nil); s.Gateways = []api.GatewaySummary{gw("rpc", "running")}; return s }(),
			func() snapshot { s := snap(nil); s.Gateways = []api.GatewaySummary{gw("rpc", "stopped")}; return s }(),
			[]ev{{logFail, "rpc", "", "gateway stopped"}}},
		{"module served", func() snapshot {
			s := snap(nil)
			s.Modules["b"] = api.ModulesView{Modules: []api.ModuleStatus{{ID: "msgboard"}}}
			return s
		}(), func() snapshot {
			s := snap(nil)
			s.Modules["b"] = api.ModulesView{Modules: []api.ModuleStatus{{ID: "msgboard", Served: true, Active: true}}}
			return s
		}(), []ev{{logOK, "b", "", "module msgboard on"}}},
		{"module error", func() snapshot {
			s := snap(nil)
			s.Modules["b"] = api.ModulesView{Modules: []api.ModuleStatus{{ID: "msgboard", Served: true}}}
			return s
		}(), func() snapshot {
			s := snap(nil)
			s.Modules["b"] = api.ModulesView{Modules: []api.ModuleStatus{{ID: "msgboard", Served: true, Error: "db locked"}}}
			return s
		}(), []ev{{logFail, "b", "", "module msgboard: db locked"}}},
		{"gate stale", func() snapshot { s := snap(nil); s.Modules["b"] = api.ModulesView{}; return s }(),
			func() snapshot { s := snap(nil); s.Modules["b"] = api.ModulesView{GateStale: true}; return s }(),
			[]ev{{logWarn, "b", "", "gate stale: module status needs a newer gate"}}},
	} {
		got := diffEvents(c.prev, c.next)
		var gotEv []ev
		for _, e := range got {
			gotEv = append(gotEv, ev{e.Sev, e.Box, e.Node, e.Text})
			if !e.At.Equal(c.next.Now) {
				t.Errorf("%s: At %v, want %v", c.name, e.At, c.next.Now)
			}
		}
		if fmt.Sprint(gotEv) != fmt.Sprint(c.want) {
			t.Errorf("%s:\n got %v\nwant %v", c.name, gotEv, c.want)
		}
	}
}

func TestDiffEventsFirstReadingSilent(t *testing.T) {
	next := snap(func(r *api.FleetRow) { r.Reachable = false; r.Jobs = &api.JobsSummary{NeedsDecision: 1} })
	next.Jobs = []api.JobView{job("j1", "needs-decision", 1, 3)}
	next.Gateways = []api.GatewaySummary{gw("rpc", "stopped")}
	next.Targets = []api.TargetView{{ID: "b", Agent: &api.AgentView{}}}
	next.Modules["b"] = api.ModulesView{GateStale: true}
	if evs := diffEvents(snapshot{Modules: map[string]api.ModulesView{}}, next); len(evs) != 0 {
		t.Fatalf("first reading logged %+v", evs)
	}
}

func TestDiffEventsAlertAgeIsStable(t *testing.T) {
	prev := snapshot{Fleet: dashFleet(), FleetHas: true, Now: testNow}
	next := prev
	next.Now = testNow.Add(time.Minute) // box-b: down 4m -> down 5m
	if evs := diffEvents(prev, next); len(evs) != 0 {
		t.Fatalf("an older age logged %+v", evs)
	}
}

func TestDiffEventsMasked(t *testing.T) {
	prev, next := snap(nil), snap(nil)
	prev.Gateways = []api.GatewaySummary{gw("rpc", "running")}
	next.Gateways = []api.GatewaySummary{gw("rpc", "bad https://u:p@h.example/key123")}
	for _, e := range diffEvents(prev, next) {
		if strings.Contains(e.Text, "u:p") || strings.Contains(e.Text, "key123") {
			t.Errorf("unmasked: %q", e.Text)
		}
	}
}

func TestActLogRingCap200(t *testing.T) {
	var l actLog
	for i := range 250 {
		l.add(logEvent{Text: fmt.Sprint(i)})
	}
	all := l.scoped("", "")
	if len(all) != 200 || all[0].Text != "249" || all[199].Text != "50" {
		t.Fatalf("kept %d, newest %q oldest %q", len(all), all[0].Text, all[len(all)-1].Text)
	}
}

func TestActLogScope(t *testing.T) {
	var l actLog
	l.add(logEvent{Box: "box-a", Node: "exec", Text: "1"}, logEvent{Box: "box-a", Text: "2"}, logEvent{Box: "box-b", Text: "3"})
	texts := func(evs []logEvent) string {
		var s []string
		for _, e := range evs {
			s = append(s, e.Text)
		}
		return strings.Join(s, ",")
	}
	for _, c := range []struct{ box, node, want string }{{"", "", "3,2,1"}, {"box-a", "", "2,1"}, {"box-a", "exec", "1"}, {"box-c", "", ""}} {
		if got := texts(l.scoped(c.box, c.node)); got != c.want {
			t.Errorf("scoped(%q,%q) = %q, want %q", c.box, c.node, got, c.want)
		}
	}
}

func TestAlertsDiskPeersLag(t *testing.T) {
	for _, c := range []struct {
		name string
		mod  func(*api.FleetRow)
		sev  alertSev
		text string // "" no alert
	}{
		{"disk 84", func(r *api.FleetRow) { p := 84.9; r.Status.DiskUsedPct = &p }, 0, ""},
		{"disk 85", func(r *api.FleetRow) { p := 85.0; r.Status.DiskUsedPct = &p }, sevWarn, "disk 85% used"},
		{"disk 95", func(r *api.FleetRow) { p := 95.0; r.Status.DiskUsedPct = &p }, sevFail, "disk 95% used"},
		{"peers 0", func(r *api.FleetRow) { r.Status.Exec.Peers = 0 }, sevWarn, "exec peers 0"},
		{"peers 0 unread", func(r *api.FleetRow) { r.Status.Exec = api.ClientStatus{State: api.SyncUnavailable} }, 0, ""},
		{"lag 64", func(r *api.FleetRow) { l := uint64(64); r.Status.HeadLag = &l }, 0, ""},
		{"lag 65", func(r *api.FleetRow) { l := uint64(65); r.Status.HeadLag = &l }, sevWarn, "head lag 65 blk"},
	} {
		s := snap(c.mod)
		got := alerts(alertInput{Fleet: s.Fleet, FleetHas: true, Now: testNow})
		switch {
		case c.text == "" && len(got) != 0:
			t.Errorf("%s: %+v, want none", c.name, got)
		case c.text != "" && (len(got) != 1 || got[0].Sev != c.sev || got[0].Text != c.text):
			t.Errorf("%s: %+v, want one %v %q", c.name, got, c.sev, c.text)
		}
	}
}
