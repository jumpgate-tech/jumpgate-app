package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/catalog"
	"github.com/valve-tech/jumpgate/internal/tui/tuitest"
)

const checkpointzReason = "Checkpointz needs a beacon on this box."

// modulesFake is box on chain 369 with reth: the message board on and
// accepting, the EL bootnode off, Checkpointz disabled with a reason, and the
// watch hidden (ok false, no reason).
func modulesFake() *tuitest.Fake {
	f := tuitest.NewFake()
	f.TargetsV = []api.TargetView{{ID: "box", Wire: &catalog.WireConfig{ChainID: 369, ExecID: "reth", Modules: []catalog.NodeModule{{ID: "msgboard"}}}}}
	f.CatalogV = []api.CatalogModule{
		{ID: "msgboard", Name: "Message board", Placement: "node", Applies: []api.ModuleApplies{
			{ChainID: 369, Client: "reth", OK: true},
			{ChainID: 369, Client: "go-pulse", Reason: "The message board needs reth (valve fork). This node runs go-pulse."},
		}},
		{ID: "el-bootnode", Name: "EL bootnode", Placement: "box", Applies: []api.ModuleApplies{{ChainID: 369, OK: true}}},
		{ID: "checkpointz", Name: "Checkpointz", Placement: "box", Applies: []api.ModuleApplies{{ChainID: 369, Reason: checkpointzReason}}},
		{ID: "watch", Name: "Health watch", Placement: "box", Applies: []api.ModuleApplies{{ChainID: 0}}},
		{ID: "private-send", Name: "Private send", Placement: "gateway", Applies: []api.ModuleApplies{{ChainID: 1, OK: true}}},
	}
	f.ModulesV = map[string]api.ModulesView{"box": {Modules: []api.ModuleStatus{{ID: "msgboard", Served: true, Accepting: true, Count: 1234, Size: 3_100_000}}}}
	return f
}

func TestOverviewShowsModules(t *testing.T) {
	a := openTestHost(t, modulesFake(), "unicode")
	fr := tuitest.Frame(a)
	for _, want := range []string{
		"Message board  on · Accepting · 1,234 msgs · local only",
		"EL bootnode", "off",
		"Checkpointz", checkpointzReason,
	} {
		if !strings.Contains(fr, want) {
			t.Errorf("overview lacks %q:\n%s", want, fr)
		}
	}
	for _, not := range []string{"Health watch", "Private send"} {
		if strings.Contains(fr, not) {
			t.Errorf("overview shows %q:\n%s", not, fr)
		}
	}
	tuitest.Golden(t, "host_overview_modules", fr)
}

func TestOverviewWithoutModulesHasNoBlock(t *testing.T) {
	a := openTestHost(t, tuitest.NewFake(), "unicode")
	if fr := tuitest.Frame(a); strings.Contains(fr, "MODULES") {
		t.Fatalf("frame:\n%s", fr)
	}
}

func TestModulesModalFrame(t *testing.T) {
	m := press(t, openTestHost(t, modulesFake(), "unicode"), "m")
	if _, ok := m.(*App).modal.(*modulesModal); !ok {
		t.Fatalf("m opened %T", m.(*App).modal)
	}
	tuitest.Golden(t, "host_modules_modal", tuitest.Frame(m))
}

// Turning a node module on restarts reth: only the typed word sends it.
func TestModulesModalNeedsTypedRestart(t *testing.T) {
	f := modulesFake()
	f.TargetsV[0].Wire.Modules = nil
	f.SetModuleV = api.SetModuleResult{Status: "started"}
	m := press(t, openTestHost(t, f, "unicode"), "m", "enter")
	fr := tuitest.Frame(m)
	if !strings.Contains(fr, "This restarts reth on box. The beacon keeps running.") {
		t.Fatalf("no restart line:\n%s", fr)
	}
	for _, attempt := range []string{"Restart", "restar", "yes", ""} {
		m = typed(t, m, attempt)
		m = press(t, m, "enter")
		if f.Called("SetModule") {
			t.Fatalf("%q sent a request", attempt)
		}
	}
	m = typed(t, m, "restart")
	m = press(t, m, "enter")
	if !f.Called("SetModule box msgboard") || !f.LastSetModule.Enabled {
		t.Fatalf("calls %v, req %+v", f.Calls, f.LastSetModule)
	}
	_ = m
}

// Turning a module off is always allowed, and a box module off runs at once.
func TestModulesModalTurnsABoxModuleOff(t *testing.T) {
	f := modulesFake()
	f.TargetsV[0].Modules = []catalog.BoxModule{{ID: "el-bootnode", ChainID: 369}}
	f.SetModuleV = api.SetModuleResult{Status: "started"}
	f.SetupV = []api.SetupEvent{{StepID: "modules", Line: "stopping jumpgate-mod-el-bootnode-369"}, {Done: true}}
	m := press(t, openTestHost(t, f, "unicode"), "m", "down", "enter")
	if !f.Called("SetModule box el-bootnode") || f.LastSetModule.Enabled || f.LastSetModule.ChainID != 369 {
		t.Fatalf("calls %v, req %+v", f.Calls, f.LastSetModule)
	}
	if fr := tuitest.Frame(m); !strings.Contains(fr, "stopping jumpgate-mod-el-bootnode-369") || !strings.Contains(fr, "Done.") {
		t.Fatalf("no progress:\n%s", fr)
	}
}

// A box module asks for its port and public IP, defaulted from the web's
// table, and sends what was typed.
func TestModulesModalBoxFields(t *testing.T) {
	f := modulesFake()
	f.SetModuleV = api.SetModuleResult{Status: "started"}
	m := press(t, openTestHost(t, f, "unicode"), "m", "down", "enter")
	fr := tuitest.Frame(m)
	if !strings.Contains(fr, "30301") || !strings.Contains(fr, "public IP") {
		t.Fatalf("no fields:\n%s", fr)
	}
	m = press(t, m, "tab")
	m = typed(t, m, "203.0.113.9")
	m = press(t, m, "enter")
	req := f.LastSetModule
	if !f.Called("SetModule box el-bootnode") || !req.Enabled || req.Port != 30301 || req.ExtIP != "203.0.113.9" || req.ChainID != 369 {
		t.Fatalf("calls %v, req %+v", f.Calls, req)
	}
	_ = m
}

// A disabled module never sends a request; its reason is shown.
func TestModulesModalDisabledSendsNothing(t *testing.T) {
	f := modulesFake()
	m := press(t, openTestHost(t, f, "unicode"), "m", "down", "down", "enter")
	if f.Called("SetModule") {
		t.Fatal("a disabled module was sent")
	}
	if fr := tuitest.Frame(m); !strings.Contains(fr, checkpointzReason) {
		t.Fatalf("frame:\n%s", fr)
	}
}

// The public choice of an on node module is its second line; changing it
// saves without a restart.
func TestModulesModalPublicChoice(t *testing.T) {
	f := modulesFake()
	f.SetModuleV = api.SetModuleResult{Status: "saved"}
	m := press(t, openTestHost(t, f, "unicode"), "m", "p")
	if !f.Called("SetModule box msgboard") || f.LastSetModule.Public != "read" || !f.LastSetModule.Enabled {
		t.Fatalf("calls %v, req %+v", f.Calls, f.LastSetModule)
	}
	m = press(t, m, "p")
	if fr := tuitest.Frame(m); !strings.Contains(fr, "Anyone can post through your node.\n") || !strings.Contains(fr, "Posts cost the sender proof of work, not you.") {
		t.Fatalf("no posting warning:\n%s", fr)
	}
}

func TestActionsMenuOpensModules(t *testing.T) {
	m := press(t, openTestHost(t, modulesFake(), "unicode"), "x", "j", "j", "j", "j", "j", "j", "j", "j", "enter")
	if _, ok := m.(*App).modal.(*modulesModal); !ok {
		t.Fatalf("modal %T", m.(*App).modal)
	}
}

func TestModulesStaleGate(t *testing.T) {
	f := modulesFake()
	f.ModulesV["box"] = api.ModulesView{Modules: []api.ModuleStatus{}, GateStale: true, Message: "Status needs a newer gate on this box"}
	a := openTestHost(t, f, "unicode")
	if fr := tuitest.Frame(a); !strings.Contains(fr, "Status needs a newer gate on this box") {
		t.Fatalf("frame:\n%s", fr)
	}
}

func TestModulesASCII(t *testing.T) {
	for _, keys := range [][]string{nil, {"m"}} {
		m := tea.Model(openTestHost(t, modulesFake(), "ascii"))
		m = press(t, m, keys...)
		requireASCII(t, tuitest.Frame(m))
	}
}
