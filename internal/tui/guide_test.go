package tui

import (
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/apiclient"
	"github.com/valve-tech/jumpgate/internal/tui/tuitest"
)

func gVault(state string, slots ...string) *api.VaultView {
	return &api.VaultView{State: state, Slots: slots}
}
func gCtl(state, recorded string) *api.ControllerView {
	return &api.ControllerView{State: state, Recorded: recorded}
}

func healthyFleet() *api.Fleet {
	return &api.Fleet{Rows: []api.FleetRow{{ID: "box-a", Link: api.LinkAgent, Reachable: true}}}
}

func gFacts(mod func(*guideFacts)) guideFacts {
	f := guideFacts{connected: true, vault: gVault("unlocked", "passphrase", "passphrase"), ctl: gCtl("ok", "0xabc"),
		fleet: healthyFleet(), gateways: -1, goos: "linux"}
	if mod != nil {
		mod(&f)
	}
	return f
}

func stepOf(f guideFacts, id string) guideStep {
	for _, s := range guideSteps(f) {
		if s.id == id {
			return s
		}
	}
	panic(id)
}

func doneMap(f guideFacts) map[string]bool {
	m := map[string]bool{}
	for _, s := range guideSteps(f) {
		m[s.id] = s.done
	}
	return m
}

func TestGuideFreshInstall(t *testing.T) {
	f := gFacts(func(f *guideFacts) {
		f.vault, f.ctl, f.fleet = gVault("none"), gCtl("missing", ""), &api.Fleet{}
	})
	steps := guideSteps(f)
	if d, tot := guideProgress(steps); d != 1 || tot != 6 {
		t.Fatalf("progress %d/%d", d, tot)
	}
	if n := nextStep(steps); steps[n].id != "vault" {
		t.Fatalf("next %s", steps[n].id)
	}
	for id, want := range map[string]bool{"server": true, "vault": false, "recovery": false, "key": false, "box": false, "paired": false, "seal": false, "second": false, "module": false, "gateway": false} {
		if doneMap(f)[id] != want {
			t.Errorf("%s done = %v", id, !want)
		}
	}
	if s := stepOf(f, "recovery"); !strings.Contains(s.how, "Create the vault first") {
		t.Errorf("recovery: %q", s.how)
	}
	if s := stepOf(f, "key"); s.screen != scrSigners || strings.Join(s.press, "") != "K" {
		t.Errorf("key: %+v", s)
	}
	if s := stepOf(f, "box"); s.screen != scrFleet || strings.Join(s.press, "") != "va" {
		t.Errorf("box: %+v", s)
	}
}

func TestGuideSteps(t *testing.T) {
	if s := stepOf(gFacts(func(f *guideFacts) { f.connected = false }), "server"); s.done {
		t.Error("server done while not connected")
	}
	if s := stepOf(gFacts(func(f *guideFacts) { f.vault = gVault("locked", "passphrase") }), "vault"); s.done || strings.Join(s.press, "") != "U" {
		t.Errorf("locked vault: %+v", s)
	}
	if s := stepOf(gFacts(func(f *guideFacts) { f.vault = gVault("unlocked", "passphrase") }), "recovery"); s.done {
		t.Error("one passphrase slot counted as a recovery code")
	}
	if !stepOf(gFacts(func(f *guideFacts) { f.ctl = gCtl("locked", "0xabc") }), "key").done {
		t.Error("a locked, recorded key is present")
	}
	if stepOf(gFacts(func(f *guideFacts) { f.ctl = gCtl("locked", "") }), "key").done {
		t.Error("a locked, unrecorded key counted")
	}
	if stepOf(gFacts(func(f *guideFacts) { f.ctl = nil }), "key").done {
		t.Error("an unread key counted")
	}
	unpaired := &api.Fleet{Rows: []api.FleetRow{{ID: "b", Link: api.LinkSSHOnly, Reachable: true}}}
	if d := doneMap(gFacts(func(f *guideFacts) { f.fleet = unpaired })); !d["box"] || d["paired"] {
		t.Errorf("unpaired: %v", d)
	}
	sick := &api.Fleet{Rows: []api.FleetRow{{ID: "b", Link: api.LinkAgent, Reachable: true, AlertCount: 1}}}
	if doneMap(gFacts(func(f *guideFacts) { f.fleet = sick }))["paired"] {
		t.Error("a box with an alert counted as healthy")
	}
	if d, tot := guideProgress(guideSteps(gFacts(nil))); d != tot {
		t.Errorf("all done: %d/%d", d, tot)
	}
	opt := gFacts(func(f *guideFacts) {
		f.vault = gVault("unlocked", "passphrase", "passphrase", "systemd-creds")
		f.mods, f.modsKnown = []api.ModulesView{{Modules: []api.ModuleStatus{{ID: "watch", Active: true}}}}, true
		f.gateways = 1
		f.fleet.Rows = append(f.fleet.Rows, api.FleetRow{ID: "box-b"})
	})
	for _, id := range []string{"seal", "second", "module", "gateway"} {
		if !stepOf(opt, id).done {
			t.Errorf("%s not done", id)
		}
	}
	if nextStep(guideSteps(opt)) != -1 {
		t.Error("a next step when all are done")
	}
}

// guideApp is a dashboard with the guide's facts read.
func guideApp(t *testing.T, f *tuitest.Fake, rows []api.FleetRow, v api.VaultView, c api.ControllerView) *App {
	t.Helper()
	f.VaultV, f.ControllerV = v, c
	a := newTestApp(t, f, 120, 40, "ascii")
	m, _ := tuitest.Send(a, fleetMsg{u: apiclient.Update[api.Fleet]{Value: api.Fleet{At: testNow, IntervalSeconds: 30, Rows: rows}, Has: true, State: apiclient.Live}},
		guideMsg{v: v, ctl: c})
	return m.(*App)
}

func TestGuideOnTheStripAndFirstRun(t *testing.T) {
	f := tuitest.NewFake()
	a := guideApp(t, f, nil, api.VaultView{State: "none"}, api.ControllerView{State: "missing"})
	if _, ok := a.modal.(*guideModal); !ok {
		t.Fatalf("a first run did not open the guide:\n%s", tuitest.Frame(a))
	}
	fr := tuitest.Frame(a)
	if !strings.Contains(strings.Split(fr, "\n")[0], "setup 1/6") {
		t.Fatalf("no setup 1/6 on the strip:\n%s", fr)
	}
	tuitest.Golden(t, "guide_first_run_120x40", fr)
	m := press(t, a, "esc")
	if m.(*App).modal != nil {
		t.Fatal("esc left the guide open")
	}
	// Once per session: a new reading does not open it again.
	m, _ = tuitest.Send(m, guideMsg{v: api.VaultView{State: "none"}, ctl: api.ControllerView{State: "missing"}})
	if m.(*App).modal != nil {
		t.Fatal("the guide opened a second time")
	}
	m = press(t, m, "g")
	if _, ok := m.(*App).modal.(*guideModal); !ok || !f.Called("Vault") {
		t.Fatalf("g did not open the guide and re-read the vault: %v", f.Calls)
	}
}

func TestGuideStaysShutWithABoxOrAnUnlockedVault(t *testing.T) {
	a := guideApp(t, tuitest.NewFake(), []api.FleetRow{{ID: "b"}}, api.VaultView{State: "locked"}, api.ControllerView{})
	if a.modal != nil {
		t.Fatal("opened with a box")
	}
	a = guideApp(t, tuitest.NewFake(), nil, api.VaultView{State: "unlocked"}, api.ControllerView{})
	if a.modal != nil {
		t.Fatal("opened with the vault unlocked")
	}
	all := guideApp(t, tuitest.NewFake(), []api.FleetRow{{ID: "b", Link: api.LinkAgent, Reachable: true}},
		api.VaultView{State: "unlocked", Slots: []string{"passphrase", "passphrase"}}, api.ControllerView{State: "ok"})
	if strings.Contains(tuitest.Frame(all), "setup ") {
		t.Fatalf("setup shown at 6/6:\n%s", tuitest.Frame(all))
	}
}

func TestGuideEnterRunsTheStepsAction(t *testing.T) {
	f := tuitest.NewFake()
	a := guideApp(t, f, []api.FleetRow{{ID: "b", Link: api.LinkAgent, Reachable: true}},
		api.VaultView{State: "unlocked", Slots: []string{"passphrase", "passphrase"}}, api.ControllerView{State: "missing"})
	a.o.Self = "/usr/local/bin/jumpgate"
	got := stubShell(a)
	m := press(t, a, "g")
	if fr := tuitest.Frame(m); !strings.Contains(fr, "> todo 4. Controller key present") {
		t.Fatalf("the cursor is not on the next step:\n%s", fr)
	}
	m = press(t, m, "enter")
	if m.(*App).active != scrSigners || strings.Join(*got, " ") != "/usr/local/bin/jumpgate keys init" {
		t.Fatalf("enter: view %v ran %v", m.(*App).active, *got)
	}
}

func TestGuideClickOnTheStrip(t *testing.T) {
	a := guideApp(t, tuitest.NewFake(), []api.FleetRow{{ID: "b"}}, api.VaultView{State: "locked"}, api.ControllerView{})
	a.prefs.Mouse = true
	tuitest.Frame(a)
	line := a.shown["header"].lines[0]
	x := strings.Index(line, "setup")
	if x < 0 {
		t.Fatalf("no setup on %q", line)
	}
	m, cmds := tuitest.Send(a, tea.MouseClickMsg{X: x + 1, Y: 0, Button: tea.MouseLeft})
	m = tuitest.Settle(m, cmds...)
	if _, ok := m.(*App).modal.(*guideModal); !ok {
		t.Fatal("a click on setup did not open the guide")
	}
}

const testKey = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9GKJl jumpgate-tunnel"

func TestInviteCommand(t *testing.T) {
	info := api.PairingInfo{Controller: "0x1234567890abcdef1234567890abcdef12345678", TransportKey: testKey}
	cmd, fail := inviteCommand("v0.6.0", info)
	want := "curl -fsSL https://jumpgate.app/install.sh | sh -s -- --agent --version v0.6.0 --controller 0x1234567890abcdef1234567890abcdef12345678 --transport-key '" + testKey + "'"
	if cmd != want || fail != "" {
		t.Fatalf("got %q %q", cmd, fail)
	}
	for _, c := range []struct {
		v    string
		info api.PairingInfo
		fail string
	}{
		{"dev", info, "development build"},
		{"v0.6.0", api.PairingInfo{Controller: info.Controller, TransportKey: testKey, Scope: "self"}, "self-only"},
		{"v0.6.0", api.PairingInfo{Controller: "0xzz", TransportKey: testKey}, "controller address"},
		{"v0.6.0", api.PairingInfo{Controller: info.Controller, TransportKey: "a\nb"}, "transport key"},
		{"v0.6.0", api.PairingInfo{Controller: info.Controller, TransportKey: "a\x1b[2Jb"}, "transport key"},
	} {
		if cmd, fail := inviteCommand(c.v, c.info); cmd != "" || !strings.Contains(fail, c.fail) {
			t.Errorf("%v: %q %q", c, cmd, fail)
		}
	}
	if cmd, _ := inviteCommand("v0.6.0", api.PairingInfo{Controller: info.Controller, TransportKey: "k'ey"}); !strings.HasSuffix(cmd, `'k'"'"'ey'`) {
		t.Errorf("quote: %q", cmd)
	}
}

func TestInviteLineAndCopy(t *testing.T) {
	f := tuitest.NewFake()
	f.ServerVersion = "v0.6.0"
	f.PairingV = api.PairingInfo{Controller: "0x1234567890abcdef1234567890abcdef12345678", TransportKey: testKey}
	a := dashApp(t, f, 120, 40, "ascii", "auto")
	a.o.Getenv = envOf(map[string]string{"PATH": "/nonexistent"})
	m := press(t, a, "i")
	fr := tuitest.Frame(m)
	if !strings.Contains(fr, " [c] copy  [i] hide  on the box, as root: curl -fsSL https://jumpgate.app/install.sh") {
		t.Fatalf("no invite line:\n%s", fr)
	}
	if n := strings.Count(fr, "\n") + 1; n != 40 {
		t.Fatalf("frame has %d lines", n)
	}
	tuitest.Golden(t, "invite_120x40", fr)
	checkOnce(t, "invite", m)

	// c copies: OSC 52 (a SetClipboard command), and the flash says so.
	_, cmds := tuitest.Send(m, tuitest.Key("c"))
	var osc, flash bool
	for _, c := range cmds {
		for _, msg := range tuitest.Run(c) {
			switch v := msg.(type) {
			case flashMsg:
				flash = strings.Contains(string(v), "invite copied (terminal clipboard, OSC 52)")
			default:
				osc = osc || strings.Contains(strings.ToLower(fmt.Sprintf("%T", msg)), "clipboard")
			}
		}
	}
	if !osc || !flash {
		t.Fatalf("copy: osc %v flash %v", osc, flash)
	}
	m = press(t, m, "i")
	if strings.Contains(tuitest.Frame(m), "[c] copy") {
		t.Fatal("i did not hide the invite")
	}
}

func TestInviteFeedsTheLocalCopyProgram(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix modes: the fake pbcopy is found by its exec bit")
	}
	f := tuitest.NewFake()
	f.ServerVersion = "v0.6.0"
	f.PairingV = api.PairingInfo{Controller: "0x1234567890abcdef1234567890abcdef12345678", TransportKey: testKey}
	a := dashApp(t, f, 120, 40, "ascii", "auto")
	d := t.TempDir()
	pb := fakeExe(t, d, "pbcopy", "#!/bin/sh\ncat >/dev/null\n")
	a.o.GOOS, a.o.Getenv = "darwin", envOf(map[string]string{"PATH": d})
	var ran []string
	a.o.Command = func(name string, args ...string) *exec.Cmd {
		ran = append([]string{name}, args...)
		return exec.Command("true")
	}
	m := press(t, a, "i")
	_ = press(t, m, "c")
	if len(ran) != 1 || ran[0] != pb {
		t.Fatalf("ran %v", ran)
	}
}

func TestInviteFailShows(t *testing.T) {
	f := tuitest.NewFake()
	f.Err["PairingInfo"] = errors.New("vault locked")
	m := press(t, dashApp(t, f, 120, 40, "ascii", "auto"), "i")
	if fr := tuitest.Frame(m); !strings.Contains(fr, "invite: FAIL pairing info: vault locked") {
		t.Fatalf("no FAIL:\n%s", fr)
	}
	f2 := tuitest.NewFake() // ServerVersion "": a development build
	m = press(t, dashApp(t, f2, 120, 40, "ascii", "auto"), "i")
	if fr := tuitest.Frame(m); !strings.Contains(fr, "development build") {
		t.Fatalf("no dev-build FAIL:\n%s", fr)
	}
}

func TestHelpListsTheGuideKeysWhenRelevant(t *testing.T) {
	a := guideApp(t, tuitest.NewFake(), []api.FleetRow{{ID: "b"}}, api.VaultView{State: "locked"}, api.ControllerView{})
	fr := tuitest.Frame(press(t, a, "?"))
	for _, want := range []string{"setup guide", "invite a box"} {
		if !strings.Contains(fr, want) {
			t.Errorf("help lacks %q while setup is incomplete:\n%s", want, fr)
		}
	}
	if strings.Contains(fr, "copy the invite") {
		t.Errorf("help lists c with no invite shown:\n%s", fr)
	}
}

func TestInviteLineClicks(t *testing.T) {
	f := tuitest.NewFake()
	f.ServerVersion = "v0.6.0"
	f.PairingV = api.PairingInfo{Controller: "0x1234567890abcdef1234567890abcdef12345678", TransportKey: testKey}
	a := dashApp(t, f, 120, 40, "ascii", "auto")
	a.prefs.Mouse = true
	m := press(t, a, "i")
	fr := tuitest.Frame(m)
	if strings.Contains(fr, "choose columns") {
		t.Errorf("the hints offer c for columns while c copies the invite:\n%s", fr)
	}
	p := m.(*App).shown["invite"]
	x := strings.Index(p.lines[0], "[i] hide")
	m, cmds := tuitest.Send(m, tea.MouseClickMsg{X: x + 1, Y: p.y, Button: tea.MouseLeft})
	m = tuitest.Settle(m, cmds...)
	if m.(*App).inviteOn {
		t.Fatal("a click on [i] hide left the invite shown")
	}
}
