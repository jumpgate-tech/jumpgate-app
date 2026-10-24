package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/catalog"
	"github.com/valve-tech/jumpgate/internal/tui/tuitest"
)

// treeFake has box-a on chain 369 with reth and the message board on, a
// gateway routing to box-a and one routing to box-b.
func treeFake() *tuitest.Fake {
	f := tuitest.NewFake()
	f.TargetsV = []api.TargetView{{ID: "box-a", Wire: &catalog.WireConfig{ChainID: 369, ExecID: "reth", Modules: []catalog.NodeModule{{ID: "msgboard"}}}}}
	f.CatalogV = []api.CatalogModule{{ID: "msgboard", Name: "Message board", Placement: "node", Applies: []api.ModuleApplies{{ChainID: 369, Client: "reth", OK: true}}}}
	f.ModulesV = map[string]api.ModulesView{"box-a": {Modules: []api.ModuleStatus{{ID: "msgboard", Served: true, Accepting: true}}}}
	return f
}

func routed(label, state, target string) api.GatewaySummary {
	g := gw(label, state)
	g.Config.Networks = append(g.Config.Networks, struct{ Upstreams []api.GatewayUpstreamRef }{
		Upstreams: []api.GatewayUpstreamRef{{Kind: "managed-node", TargetID: target}},
	})
	return g
}

func nodeLines(ns []treeNode) string {
	var s []string
	for _, n := range ns {
		s = append(s, strings.TrimSpace(n.Kind+" "+n.Label+" "+n.State))
	}
	return strings.Join(s, "\n")
}

func TestTreeNodesFromFleet(t *testing.T) {
	a := dashApp(t, tuitest.NewFake(), 120, 40, "ascii", "auto")
	want := "fleet fleet 3 boxes\nbox box-a synced\nbox box-b down 4m\nbox laptop* local*"
	if got := nodeLines(a.treeNodes()); got != want {
		t.Fatalf("nodes:\n%s\nwant:\n%s", got, want)
	}
}

func TestTreeExpandShowsNodesGatewaysModules(t *testing.T) {
	f := treeFake()
	a := dashApp(t, f, 120, 40, "ascii", "auto")
	a.gws, a.gwsHas = []api.GatewaySummary{routed("rpc-main", "running", "box-a"), routed("rpc-eth", "running", "box-b")}, true
	m, _ := tuitest.Send(a, tuitest.Run(a.toggleExpand("box-a"))...)
	got := nodeLines(m.(*App).treeNodes())
	for _, want := range []string{"exec exec synced 48p", "beacon beacon synced 72p", "gw gw rpc-main live", "mod mod msgboard on"} {
		if !strings.Contains(got, want) {
			t.Errorf("no %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, "rpc-eth") {
		t.Errorf("a gateway routing elsewhere is under box-a:\n%s", got)
	}
}

func TestTreeModulesOnlyWhenExpanded(t *testing.T) {
	f := treeFake()
	a := dashApp(t, f, 120, 40, "ascii", "auto")
	tuitest.Run(a.Init())
	tuitest.Run(a.treeRefresh())
	if f.Called("ListModules") {
		t.Fatalf("modules read with nothing expanded: %v", f.Calls)
	}
	if got := nodeLines(a.treeNodes()); strings.Contains(got, "loading") {
		t.Fatalf("collapsed box shows modules:\n%s", got)
	}
	cmd := a.toggleExpand("box-a")
	if got := nodeLines(a.treeNodes()); !strings.Contains(got, "modules loading~") {
		t.Errorf("no loading line:\n%s", got)
	}
	tuitest.Send(a, tuitest.Run(cmd)...)
	f.Calls = nil
	_, cmds := tuitest.Send(a, treeTickMsg{})
	if len(cmds) == 0 {
		t.Fatal("the tick does not come again")
	}
	tuitest.Run(tea.Batch(cmds...))
	if !f.Called("ListModules box-a") || f.Called("ListModules box-b") {
		t.Fatalf("calls %v", f.Calls)
	}
	// Collapsing stops the refresh.
	a.toggleExpand("box-a")
	f.Calls = nil
	tuitest.Run(a.treeRefresh())
	if f.Called("ListModules") {
		t.Fatalf("a collapsed box is read: %v", f.Calls)
	}
}

func TestTreeGateStale(t *testing.T) {
	f := treeFake()
	f.ModulesV["box-a"] = api.ModulesView{GateStale: true}
	a := dashApp(t, f, 120, 40, "ascii", "auto")
	m, _ := tuitest.Send(a, tuitest.Run(a.toggleExpand("box-a"))...)
	if got := nodeLines(m.(*App).treeNodes()); !strings.Contains(got, "mod mod msgboard gate stale") {
		t.Fatalf("nodes:\n%s", got)
	}
}

func TestTreeViewASCII(t *testing.T) {
	f := treeFake()
	fl := dashFleet()
	fl.Rows[0].ID = "a-very-long-box-name-indeed"
	a := dashApp(t, f, 120, 40, "ascii", "auto")
	a.fleet = fl
	a.toggleExpand("box-b")
	v := ansi.Strip(a.treeView(22, 12))
	requireASCII(t, v)
	for i, l := range strings.Split(v, "\n") {
		if w := ansi.StringWidth(l); w > 22 {
			t.Errorf("line %d is %d wide: %q", i, w, l)
		}
	}
	if !strings.Contains(v, "a-very-long~") || !strings.Contains(v, "- box-b") || !strings.Contains(v, "+ laptop*") {
		t.Fatalf("tree:\n%s", v)
	}
	if fmt.Sprint(strings.Count(v, "\n")+1) != "12" {
		t.Fatalf("tree is not 12 lines:\n%s", v)
	}
}
