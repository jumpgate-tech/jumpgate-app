package tui

import (
	"fmt"
	"strings"
	"testing"
	"time"

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
	f.CatalogV[0].Public = []string{"read", "post"}
	gw := api.GatewaySummary{ID: "gw", Tier: "public"}
	gw.Config.Networks = append(gw.Config.Networks, struct{ Upstreams []api.GatewayUpstreamRef }{
		Upstreams: []api.GatewayUpstreamRef{{Kind: "managed-node", TargetID: "box"}},
	})
	f.GatewaysV = []api.GatewaySummary{gw}
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

// A running bootnode shows its enode and the probe's note under the cursor,
// and c copies the enode to the terminal clipboard (OSC 52).
func TestModulesModalCopiesBootnodeIdentity(t *testing.T) {
	enode := "enode://" + strings.Repeat("ab12", 32) + "@203.0.113.7:30303"
	note := "Set the box's public IP (ExtIP) on the bootnode."
	f := modulesFake()
	f.TargetsV[0].Modules = []catalog.BoxModule{{ID: "el-bootnode", ChainID: 369}}
	f.ModulesV["box"] = api.ModulesView{Modules: []api.ModuleStatus{{ID: "el-bootnode", Active: true, Identity: enode, Note: note}}}
	m := press(t, openTestHost(t, f, "unicode"), "m", "down")
	fr := tuitest.Frame(m)
	for _, want := range []string{"enode://ab12ab12", note, "c to copy"} {
		if !strings.Contains(fr, want) {
			t.Errorf("modal lacks %q:\n%s", want, fr)
		}
	}
	mm := m.(*App).modal.(*modulesModal)
	_, cmd := mm.key(m.(*App), tuitest.Key("c"))
	if cmd == nil {
		t.Fatal("c sent no clipboard command")
	}
	if got, want := fmt.Sprint(cmd()), fmt.Sprint(tea.SetClipboard(enode)()); got != want {
		t.Fatalf("clipboard msg %q, want %q", got, want)
	}
	if !strings.Contains(mm.note, "Copied the enode") {
		t.Fatalf("note %q", mm.note)
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
	f.Calls = nil
	// Moving to post shows the D11 warning first and sends nothing yet.
	m = press(t, m, "p")
	if f.Called("SetModule") {
		t.Fatal("post was sent before the warning was seen")
	}
	if fr := tuitest.Frame(m); !strings.Contains(fr, "Anyone can post through your node.\n") || !strings.Contains(fr, "Posts cost the sender proof of work, not you.") {
		t.Fatalf("no posting warning:\n%s", fr)
	}
	m = press(t, m, "esc")
	if f.Called("SetModule") || m.(*App).modal == nil {
		t.Fatal("esc on the warning sent post or closed the modal")
	}
	m = press(t, m, "p", "enter")
	if !f.Called("SetModule box msgboard") || f.LastSetModule.Public != "post" {
		t.Fatalf("calls %v, req %+v", f.Calls, f.LastSetModule)
	}
}

// The Public choice needs a public-tier gateway routing to the node, and a
// module whose catalog entry offers choices.
func TestModulesModalPublicNeedsPublicGatewayAndOptions(t *testing.T) {
	for name, edit := range map[string]func(f *tuitest.Fake){
		"private gateway": func(f *tuitest.Fake) { f.GatewaysV[0].Tier = "private" },
		"other machine":   func(f *tuitest.Fake) { f.GatewaysV[0].Config.Networks[0].Upstreams[0].TargetID = "other" },
		"no options":      func(f *tuitest.Fake) { f.CatalogV[0].Public = nil },
	} {
		t.Run(name, func(t *testing.T) {
			f := modulesFake()
			edit(f)
			m := press(t, openTestHost(t, f, "unicode"), "m", "p")
			if f.Called("SetModule") || strings.Contains(tuitest.Frame(m), "p to change") {
				t.Fatalf("p offered:\n%s", tuitest.Frame(m))
			}
		})
	}
}

// A module that is on stays listed, and can be turned off, even when the
// catalog would now hide it.
func TestModulesOnButHiddenCanBeTurnedOff(t *testing.T) {
	f := modulesFake()
	f.TargetsV[0].Modules = []catalog.BoxModule{{ID: "watch"}}
	f.SetModuleV = api.SetModuleResult{Status: "started"}
	a := openTestHost(t, f, "unicode")
	if fr := tuitest.Frame(a); !strings.Contains(fr, "Health watch") {
		t.Fatalf("an on module was hidden:\n%s", fr)
	}
	press(t, a, "m", "down", "down", "down", "enter")
	if !f.Called("SetModule box watch") || f.LastSetModule.Enabled {
		t.Fatalf("calls %v, req %+v", f.Calls, f.LastSetModule)
	}
}

// The status line's precedence, case for case with modulesModel.test.ts.
func TestModuleStatusParityWithWeb(t *testing.T) {
	msgboard := api.CatalogModule{ID: "msgboard", Placement: "node"}
	cpz := api.CatalogModule{ID: "checkpointz", Placement: "box"}
	served := &api.ModuleStatus{ID: "msgboard", Served: true, Accepting: true, Count: 1234}
	st := func(edit func(s *api.ModuleStatus)) *api.ModuleStatus {
		s := *served
		edit(&s)
		return &s
	}
	synced := api.NodeStatus{Exec: api.ClientStatus{Active: true, Peers: 5, State: api.SyncSynced}}
	cases := []struct {
		name      string
		m         api.CatalogModule
		on        bool
		s         *api.ModuleStatus
		node      *api.NodeStatus
		restarted time.Duration // ago; 0 none
		stale     bool
		want      string
	}{
		{"node stopped", msgboard, true, st(func(s *api.ModuleStatus) { s.Served = false }), &api.NodeStatus{}, 0, false, "Node stopped"},
		{"starting grace", msgboard, true, st(func(s *api.ModuleStatus) { s.Served = false }), nil, 30 * time.Second, false, "Starting"},
		{"error after grace", msgboard, true, st(func(s *api.ModuleStatus) { s.Served = false; s.Error = "msgboard is not served" }), nil, 90 * time.Second, false, "msgboard is not served"},
		{"not served", msgboard, true, st(func(s *api.ModuleStatus) { s.Served = false }), nil, 0, false, "Not served yet"},
		{"sync before peers", msgboard, true, served, &api.NodeStatus{Exec: api.ClientStatus{Active: true, State: api.SyncSyncing}}, 0, false, "Waiting for sync"},
		{"no peers", msgboard, true, st(func(s *api.ModuleStatus) { s.Accepting = false }), &api.NodeStatus{Exec: api.ClientStatus{Active: true, State: api.SyncSynced}}, 0, false, "No peers"},
		{"not accepting", msgboard, true, st(func(s *api.ModuleStatus) { s.Accepting = false }), nil, 0, false, "Starting"},
		{"accepting", msgboard, true, served, nil, 0, false, "Accepting · 1,234 msgs"},
		{"served, not set", msgboard, false, served, &synced, 0, false, "Served but not set by jumpgate"},
		{"off, not served", msgboard, false, st(func(s *api.ModuleStatus) { s.Served = false }), &synced, 0, false, ""},
		{"off, node stopped", msgboard, false, st(func(s *api.ModuleStatus) { s.Served = false }), &api.NodeStatus{}, 0, false, ""},
		{"stale gate", msgboard, true, served, nil, 0, true, "Status needs a newer gate on this box"},
		{"box identity", cpz, true, &api.ModuleStatus{ID: "checkpointz", Active: true, Identity: "enode://x"}, nil, 0, false, "Running · enode://x"},
		{"box running", cpz, true, &api.ModuleStatus{ID: "checkpointz", Active: true}, nil, 0, false, "Running"},
		{"box stopped", cpz, true, &api.ModuleStatus{ID: "checkpointz"}, nil, 0, false, "Stopped"},
		{"box error", cpz, true, &api.ModuleStatus{ID: "checkpointz", Error: "unit failed"}, nil, 0, false, "unit failed"},
		{"box no reading", cpz, true, nil, nil, 0, false, "Stopped"},
		{"box off", cpz, false, nil, nil, 0, false, ""},
	}
	a := newTestApp(t, tuitest.NewFake(), 80, 24, "unicode")
	for _, c := range cases {
		h := &hostScreen{id: "box"}
		if c.s != nil {
			h.modView.Modules = []api.ModuleStatus{*c.s}
		}
		h.modView.GateStale = c.stale
		if c.node != nil {
			h.status, h.statusHas = *c.node, true
		}
		if c.restarted > 0 {
			h.modRestarted = testNow.Add(-c.restarted)
		}
		state := "off"
		if c.on {
			state = "on"
		}
		if got := h.moduleStatus(a, modCard{m: c.m, state: state}); got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
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
