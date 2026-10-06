package tui

import (
	"context"
	"errors"
	"os/exec"
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/tui/tuitest"
)

func signersApp(t *testing.T, f *tuitest.Fake) *App {
	t.Helper()
	f.TargetsV = sampleTargets()
	if f.ControllerV.State == "" {
		f.ControllerV = api.ControllerView{Recorded: "0xc0ffee00000000000000000000000000000000aa", Address: "0xc0ffee00000000000000000000000000000000aa", Store: "keychain", State: "ok"}
	}
	return press(t, newTestApp(t, f, 80, 24, "unicode"), "3").(*App)
}

func TestSignersShowsTheKeyAndBoxes(t *testing.T) {
	fr := tuitest.Frame(signersApp(t, tuitest.NewFake()))
	for _, want := range []string{"THIS CONTROLLER", "0xc0ff…00aa", "keychain", "ok", "box-a", "agent 0x1234…5678", "routine", "approval tier"} {
		if !strings.Contains(fr, want) {
			t.Errorf("signers lack %q:\n%s", want, fr)
		}
	}
	if strings.Contains(fr, "box-b") {
		t.Errorf("an unpaired box is listed as signed for:\n%s", fr)
	}
	tuitest.Golden(t, "signers", fr)
}

func TestTestSigningShowsTheVerifiedAnswer(t *testing.T) {
	f := tuitest.NewFake()
	f.CheckV = api.AgentCheck{Agent: "0x1234567890abcdef1234567890abcdef12345678", Version: "v0.9.0", ChainID: 369, SetUp: true, ElapsedMs: 120}
	m := press(t, signersApp(t, f), "t")
	if !f.Called("CheckAgent box-a") {
		t.Fatalf("calls %v", f.Calls)
	}
	if fr := tuitest.Frame(m); !strings.Contains(fr, "signed and verified in 120ms") || !strings.Contains(fr, "v0.9.0") {
		t.Fatalf("frame:\n%s", fr)
	}
}

func TestMismatchIsAnError(t *testing.T) {
	f := tuitest.NewFake()
	f.ControllerV = api.ControllerView{Recorded: "0xaaaa", Address: "0xbbbb", Store: "1password", State: "mismatch", Reason: "the key store holds a different key"}
	if fr := tuitest.Frame(signersApp(t, f)); !strings.Contains(fr, "MISMATCH") || !strings.Contains(fr, "different key") {
		t.Fatalf("frame:\n%s", fr)
	}
}

// Every controller state says what it is and what to do about it.
func TestEveryControllerStateHasAHint(t *testing.T) {
	for _, c := range []struct {
		v    api.ControllerView
		want []string
	}{
		{api.ControllerView{State: "missing"}, []string{"no key yet", "jumpgate keys init"}},
		{api.ControllerView{State: "unopened", Store: "1password", Recorded: "0xc0ffee00000000000000000000000000000000aa", Reason: "the vault is locked"}, []string{"not loaded", "vault is locked", "R"}},
		{api.ControllerView{State: "unrecorded", Address: "0xc0ffee00000000000000000000000000000000aa", Store: "keychain"}, []string{"not recorded", "jumpgate keys init"}},
		{api.ControllerView{State: "mismatch", Address: "0xbbbb", Recorded: "0xaaaa", Store: "keychain"}, []string{"MISMATCH"}},
		{api.ControllerView{State: "ok", Address: "0xc0ffee00000000000000000000000000000000aa", Store: "keychain"}, []string{"ok"}},
	} {
		f := tuitest.NewFake()
		f.ControllerV = c.v
		a := signersApp(t, f)
		a.o.Restart = func(context.Context) (Backend, error) { return f, nil }
		fr := tuitest.Frame(a)
		for _, w := range c.want {
			if !strings.Contains(fr, w) {
				t.Errorf("state %s lacks %q:\n%s", c.v.State, w, fr)
			}
		}
	}
}

func TestControllerErrorIsShown(t *testing.T) {
	f := tuitest.NewFake()
	f.ControllerV = api.ControllerView{State: "ok"}
	f.Err = map[string]error{"Controller": errors.New("server unreachable")}
	if fr := tuitest.Frame(signersApp(t, f)); !strings.Contains(fr, "unavailable") || !strings.Contains(fr, "server unreachable") {
		t.Fatalf("frame:\n%s", fr)
	}
}

func TestCreateKeyRunsKeysInitInTheForeground(t *testing.T) {
	f := tuitest.NewFake()
	f.ControllerV = api.ControllerView{State: "missing"}
	a := signersApp(t, f)
	a.o.Self = "/usr/local/bin/jumpgate"
	var got []string
	a.o.Command = func(name string, args ...string) *exec.Cmd {
		got = append([]string{name}, args...)
		return exec.Command("true")
	}
	press(t, a, "K")
	if !slices.Equal(got, []string{"/usr/local/bin/jumpgate", "keys", "init"}) {
		t.Fatalf("ran %v", got)
	}
}

func TestCreateKeyIsRefusedWhenAKeyIsRecorded(t *testing.T) {
	a := signersApp(t, tuitest.NewFake())
	a.o.Self = "/usr/local/bin/jumpgate"
	ran := false
	a.o.Command = func(string, ...string) *exec.Cmd { ran = true; return exec.Command("true") }
	m := press(t, a, "K")
	if ran || !strings.Contains(tuitest.Frame(m), "never replaces") {
		t.Fatalf("ran=%v frame:\n%s", ran, tuitest.Frame(m))
	}
}

func TestRestartFromSigners(t *testing.T) {
	f := tuitest.NewFake()
	a := signersApp(t, f)
	next := tuitest.NewFake()
	a.o.Restart = func(context.Context) (Backend, error) { return next, nil }
	m := press(t, a, "R")
	if m.(*App).be != Backend(next) || !strings.Contains(tuitest.Frame(m), "the server restarted") {
		t.Fatalf("frame:\n%s", tuitest.Frame(m))
	}
}

func TestRestartRefusalIsShown(t *testing.T) {
	a := signersApp(t, tuitest.NewFake())
	a.o.Restart = func(context.Context) (Backend, error) {
		return nil, errors.New("the server on this port is not this client's")
	}
	if fr := tuitest.Frame(press(t, a, "R")); !strings.Contains(fr, "restart failed") || !strings.Contains(fr, "not this client's") {
		t.Fatalf("frame:\n%s", fr)
	}
}

func TestRestartIsHiddenWithoutAWayToDoIt(t *testing.T) {
	a := signersApp(t, tuitest.NewFake())
	if fr := tuitest.Frame(a); strings.Contains(fr, "R restart") {
		t.Fatalf("R offered with no Restart:\n%s", fr)
	}
	for _, k := range a.screens[scrSigners].keys() {
		if k.Help().Key == "R" {
			t.Fatal("R in the key list")
		}
	}
	a.o.Restart = func(context.Context) (Backend, error) { return nil, nil }
	if fr := tuitest.Frame(a); !strings.Contains(fr, "R restart") {
		t.Fatalf("R not offered:\n%s", fr)
	}
}

func TestSignersSanitizesEverything(t *testing.T) {
	const evil = "\x1b]52;c;aGk=\x07\x1b[2J"
	f := tuitest.NewFake()
	f.ControllerV = api.ControllerView{Recorded: "0x" + evil + "aaaaaaaaaaaaaaaaaaaa", Address: "0x" + evil + "bbbbbbbbbbbbbbbbbbbb", Store: "kc" + evil, State: "wat" + evil, Reason: "why" + evil}
	f.CheckV = api.AgentCheck{Version: "v" + evil, ElapsedMs: 1}
	f.TargetsV = []api.TargetView{{ID: "box" + evil, Agent: &api.AgentView{Address: "0x" + evil + "cccccccccccccccccccc", Transport: "ssh" + evil}}}
	plain := func(f *tuitest.Fake) *App {
		a := newTestApp(t, f, 80, 24, "unicode")
		a.th = plainTheme() // every ESC left in a frame came from data
		return a
	}
	a := press(t, plain(f), "3").(*App)
	noEscapes(t, "controller", a)
	m := press(t, a, "t")
	noEscapes(t, "check", m)
	f2 := tuitest.NewFake()
	f2.TargetsV, f2.ControllerV = f.TargetsV, f.ControllerV
	f2.Err = map[string]error{"CheckAgent": errors.New("boom" + evil), "Controller": errors.New("ctl" + evil)}
	b := press(t, plain(f2), "3", "t")
	noEscapes(t, "errors", b)
}

func signersScr(m interface{ View() tea.View }) *signersScreen {
	return m.(*App).screens[scrSigners].(*signersScreen)
}

// A reply from before the screen was left, or before a reload, is dropped.
func TestStaleSignerRepliesAreDropped(t *testing.T) {
	a := signersApp(t, tuitest.NewFake())
	s := signersScr(a)
	old := s.gen
	m, _ := tuitest.Send(a, tuitest.Key("r")) // reload: a new generation
	if s.gen == old {
		t.Fatal("reload did not start a new generation")
	}
	m, _ = tuitest.Send(m, checkMsg{gen: old, id: "box-a", c: api.AgentCheck{Version: "STALE", ElapsedMs: 1}})
	m, _ = tuitest.Send(m, controllerMsg{gen: old, v: api.ControllerView{State: "mismatch", Store: "STALE"}})
	if fr := tuitest.Frame(m); strings.Contains(fr, "STALE") {
		t.Fatalf("a stale reply was shown after a reload:\n%s", fr)
	}
	// Leave and come back: results from the first visit do not return.
	gen := s.gen
	m = press(t, m, "t")
	m = press(t, m, "1", "3")
	if s.gen == gen {
		t.Fatal("leaving and returning did not start a new generation")
	}
	m, _ = tuitest.Send(m, checkMsg{gen: gen, id: "box-a", c: api.AgentCheck{Version: "STALE", ElapsedMs: 1}})
	if fr := tuitest.Frame(m); strings.Contains(fr, "STALE") || strings.Contains(fr, "signed and verified") {
		t.Fatalf("a reply from the earlier visit was shown:\n%s", fr)
	}
}

// Commands started on the screen run on a context that ends when it is left.
func TestSignersContextEndsWhenLeft(t *testing.T) {
	a := signersApp(t, tuitest.NewFake())
	ctx := signersScr(a).ctx
	if ctx == nil || ctx.Err() != nil {
		t.Fatalf("no live context on the screen: %v", ctx)
	}
	press(t, a, "1")
	if ctx.Err() == nil {
		t.Fatal("the context outlived the screen")
	}
}
