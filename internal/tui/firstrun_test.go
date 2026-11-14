package tui

import (
	"strings"
	"testing"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/apiclient"
	"github.com/valve-tech/jumpgate/internal/tui/tuitest"
)

// A fleet with no box says what to do, and each way is a button.
func TestFleetEmptyState(t *testing.T) {
	mk := func() *App {
		a := guideApp(t, tuitest.NewFake(), nil, api.VaultView{State: "unlocked", Slots: []string{"passphrase"}}, api.ControllerView{State: "ok"})
		a.modal = nil
		return a
	}
	fr := tuitest.Frame(mk())
	for _, want := range []string{"No boxes yet.", "[a] add a box over SSH", "[i] one line to run on the box", "[g] setup guide"} {
		if !strings.Contains(fr, want) {
			t.Errorf("empty Fleet lacks %q:\n%s", want, fr)
		}
	}
	tuitest.Golden(t, "fleet_empty_120x40", fr)
	// Each button works by click.
	a := mk()
	x, y := at(t, a, "[a] add")
	m := click(t, a, x+2, y)
	if fs := m.(*App).screens[scrFleet].(*fleetScreen); fs.hosts.flow == nil {
		t.Fatalf("clicking [a] did not start the add flow:\n%s", tuitest.Frame(m))
	}
	a = mk()
	x, y = at(t, a, "[i] one line")
	if m = click(t, a, x+2, y); !m.(*App).inviteOn {
		t.Fatal("clicking [i] did not show the invite line")
	}
	a = mk()
	x, y = at(t, a, "[g] setup")
	if m = click(t, a, x+2, y); func() bool { _, ok := m.(*App).modal.(*guideModal); return !ok }() {
		t.Fatal("clicking [g] did not open the guide")
	}
	// At 80x24 too, and in ASCII.
	b := newTestApp(t, tuitest.NewFake(), 80, 24, "ascii")
	b2, _ := tuitest.Send(b, fleetMsg{u: apiclientLive(api.Fleet{At: testNow, IntervalSeconds: 15})})
	fr = tuitest.Frame(b2)
	if !strings.Contains(fr, "No boxes yet.") || !strings.Contains(fr, "[g] setup guide") {
		t.Errorf("80x24:\n%s", fr)
	}
	requireASCII(t, fr)
}

// The setup checklist names real keys and screens: nothing sends a person to
// a tab or a web page that is not there.
func TestGuideStepsNameRealThings(t *testing.T) {
	steps := guideSteps(gFacts(func(f *guideFacts) {
		f.vault, f.ctl, f.fleet = gVault("none"), gCtl("missing", ""), &api.Fleet{}
		f.modsKnown, f.gateways = true, 0
	}))
	for _, s := range steps {
		for _, bad := range []string{"Modules tab", "web app", "hosts view"} {
			if strings.Contains(s.how, bad) {
				t.Errorf("%s: %q", s.id, s.how)
			}
		}
	}
	if s := stepOf(gFacts(func(f *guideFacts) { f.modsKnown = true }), "module"); !strings.Contains(s.how, "press m") {
		t.Errorf("module: %q", s.how)
	}
	if s := stepOf(gFacts(func(f *guideFacts) { f.gateways = 0 }), "gateway"); !strings.Contains(s.how, "press ctrl+n") || strings.Join(s.press, "") != "ctrl+n" {
		t.Errorf("gateway: %+v", s)
	}
	if s := stepOf(gFacts(func(f *guideFacts) { f.vault = gVault("unusable") }), "vault"); !strings.Contains(s.how, "press ctrl+r") {
		t.Errorf("vault: %q", s.how)
	}
}

// Why and do wrap under their labels instead of being cut at 80 columns.
func TestGuideWrapsWhyAndDo(t *testing.T) {
	a := newTestApp(t, tuitest.NewFake(), 80, 24, "ascii")
	m, _ := tuitest.Send(a, fleetMsg{u: apiclientLive(api.Fleet{At: testNow, IntervalSeconds: 15})},
		guideMsg{v: api.VaultView{State: "unusable"}, ctl: api.ControllerView{State: "ok"}})
	if _, ok := m.(*App).modal.(*guideModal); !ok { // a first run opens it by itself
		m = press(t, m, "g")
	}
	fr := tuitest.Frame(m)
	flat := strings.Join(strings.Fields(fr), " ")
	for _, want := range []string{"It holds the controller key and the tokens, encrypted with your passphrase.", "Its file cannot be used. Vault: press ctrl+r to restore a backup."} {
		if !strings.Contains(flat, want) {
			t.Errorf("guide lacks the whole sentence %q:\n%s", want, fr)
		}
	}
	if n := len(strings.Split(fr, "\n")); n != 24 {
		t.Fatalf("%d lines", n)
	}
	requireASCII(t, fr)
}

// A click on a step goes there, as enter does.
func TestGuideStepClickGoesThere(t *testing.T) {
	f := tuitest.NewFake()
	a := guideApp(t, f, nil, api.VaultView{State: "none"}, api.ControllerView{State: "missing"})
	x, y := at(t, a, "First box added")
	m := click(t, a, x+2, y)
	app := m.(*App)
	if app.modal != nil || app.active != scrFleet {
		t.Fatalf("modal %T active %v", app.modal, app.active)
	}
	if fs := app.screens[scrFleet].(*fleetScreen); fs.hosts.flow == nil {
		t.Fatalf("the step did not start its action:\n%s", tuitest.Frame(m))
	}
}

func apiclientLive(f api.Fleet) apiclient.Update[api.Fleet] {
	return apiclient.Update[api.Fleet]{Value: f, Has: true, State: apiclient.Live}
}
