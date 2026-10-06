package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/apiclient"
	"github.com/valve-tech/jumpgate/internal/tui/tuitest"
)

func TestPaletteOpensAHostOnItsLogs(t *testing.T) {
	m := press(t, newTestApp(t, tuitest.NewFake(), 80, 24, "unicode"), ":")
	m = typed(t, m, "logs box-a")
	m = press(t, m, "enter")
	h, ok := m.(*App).detail.(*hostScreen)
	if !ok || h.id != "box-a" || h.tabName(m.(*App)) != "logs" {
		t.Fatalf("detail %#v", m.(*App).detail)
	}
}

func TestPaletteRestartAsksForTheName(t *testing.T) {
	f := tuitest.NewFake()
	m := press(t, newTestApp(t, f, 80, 24, "unicode"), ":")
	m = typed(t, m, "restart beacon box-a")
	m = press(t, m, "enter")
	c, ok := m.(*App).modal.(*confirmModal)
	if !ok || c.want != "beacon" || f.Called("ServiceAction") {
		t.Fatalf("modal %#v calls %v", m.(*App).modal, f.Calls)
	}
}

func TestPaletteDestructiveCommandsNeedTheTypedWord(t *testing.T) {
	cases := []struct{ line, want, call string }{
		{"restart exec box-a", "exec", "ServiceAction box-a exec restart"},
		{"stop beacon box-a", "beacon", "ServiceAction box-a beacon stop"},
		{"remove box-a", "box-a", "RemoveTarget box-a"},
	}
	for _, c := range cases {
		t.Run(c.line, func(t *testing.T) {
			f := tuitest.NewFake()
			m := press(t, newTestApp(t, f, 80, 24, "unicode"), ":")
			m = typed(t, m, c.line)
			m = press(t, m, "enter")
			cm, ok := m.(*App).modal.(*confirmModal)
			if !ok || cm.want != c.want {
				t.Fatalf("modal %#v", m.(*App).modal)
			}
			m = press(t, m, "enter") // nothing typed
			m = typed(t, m, "wrong")
			m = press(t, m, "enter")
			if f.Called(strings.Fields(c.call)[0]) {
				t.Fatalf("ran without the word: %v", f.Calls)
			}
			m = press(t, m, "esc")
			if f.Called(strings.Fields(c.call)[0]) {
				t.Fatalf("ran after esc: %v", f.Calls)
			}
			// And with the word it runs, exactly as the key's path does.
			m = press(t, newTestApp(t, f, 80, 24, "unicode"), ":")
			m = typed(t, m, c.line)
			m = press(t, m, "enter")
			m = typed(t, m, c.want)
			m = press(t, m, "enter")
			if !f.Called(c.call) {
				t.Fatalf("calls %v, want %q", f.Calls, c.call)
			}
		})
	}
}

func TestPaletteStartNeedsNoConfirmationLikeItsKey(t *testing.T) {
	f := tuitest.NewFake()
	m := press(t, newTestApp(t, f, 80, 24, "unicode"), ":")
	m = typed(t, m, "start exec box-a")
	press(t, m, "enter")
	if !f.Called("ServiceAction box-a exec start") {
		t.Fatalf("calls %v", f.Calls)
	}
}

func TestPaletteShellIsScreened(t *testing.T) {
	f := tuitest.NewFake()
	f.SSHV["box-a"] = api.SSHCommand{Argv: []string{"ssh", "-oProxyCommand=evil", "x"}}
	a := newTestApp(t, f, 80, 24, "unicode")
	got := stubShell(a)
	m := press(t, a, ":")
	m = typed(t, m, "ssh box-a")
	m = press(t, m, "enter")
	if len(*got) != 0 {
		t.Fatalf("ran %v", *got)
	}
	if fr := tuitest.Frame(m); !strings.Contains(fr, "refusing") {
		t.Fatalf("frame:\n%s", fr)
	}
}

func TestPaletteIsLiteralNotARegexp(t *testing.T) {
	a := newTestApp(t, tuitest.NewFake(), 80, 24, "unicode")
	m, _ := tuitest.Send(a, fleetMsgFor(t))
	for _, line := range []string{"logs box-.", "logs .*", "remove (", "re.*t exec box-a", "jo+bs"} {
		mm := press(t, m, ":")
		mm = typed(t, mm, line)
		mm = press(t, mm, "enter")
		if mm.(*App).detail != nil {
			t.Fatalf("%q opened a host", line)
		}
		if _, ok := mm.(*App).modal.(*confirmModal); ok {
			t.Fatalf("%q opened a confirmation", line)
		}
	}
}

func TestPaletteUnknownBoxIsRefusedOnceTheFleetIsKnown(t *testing.T) {
	a := newTestApp(t, tuitest.NewFake(), 80, 24, "unicode")
	m, _ := tuitest.Send(a, fleetMsgFor(t))
	m = press(t, m, ":")
	m = typed(t, m, "remove nope")
	m = press(t, m, "enter")
	if _, ok := m.(*App).modal.(*confirmModal); ok || !strings.Contains(tuitest.Frame(m), `no box named "nope"`) {
		t.Fatalf("frame:\n%s", tuitest.Frame(m))
	}
}

func TestPaletteTakesGlobalKeysAsTextAndPaste(t *testing.T) {
	m := press(t, newTestApp(t, tuitest.NewFake(), 80, 24, "unicode"), ":")
	m = typed(t, m, "q1?:")
	p := m.(*App).modal.(*paletteModal)
	if p.in.Value() != "q1?:" || p.in.KeyMap.Paste.Enabled() {
		t.Fatalf("value %q paste %v", p.in.Value(), p.in.KeyMap.Paste.Enabled())
	}
	m, _ = tuitest.Send(m, tea.PasteMsg{Content: "a\x1b[2Jb\nc"})
	if v := m.(*App).modal.(*paletteModal).in.Value(); strings.ContainsAny(v, "\x1b\n") {
		t.Fatalf("paste not cleaned: %q", v)
	}
}

func TestPaletteUnknownCommand(t *testing.T) {
	m := press(t, newTestApp(t, tuitest.NewFake(), 80, 24, "unicode"), ":")
	m = typed(t, m, "frobnicate")
	m = press(t, m, "enter")
	if fr := tuitest.Frame(m); !strings.Contains(fr, `unknown command "frobnicate"`) {
		t.Fatalf("frame:\n%s", fr)
	}
}

func TestPaletteSwitchesScreens(t *testing.T) {
	m := press(t, newTestApp(t, tuitest.NewFake(), 80, 24, "unicode"), ":")
	m = typed(t, m, "jobs")
	m = press(t, m, "enter")
	if m.(*App).active != scrJobs || m.(*App).modal != nil {
		t.Fatalf("active %v", m.(*App).active)
	}
}

func TestPaletteEscCloses(t *testing.T) {
	m := press(t, newTestApp(t, tuitest.NewFake(), 80, 24, "unicode"), ":", "esc")
	if m.(*App).modal != nil {
		t.Fatal("still open")
	}
}

func fleetMsgFor(t *testing.T) tea.Msg {
	t.Helper()
	return fleetMsg{u: apiclient.Update[api.Fleet]{Value: sampleFleet(), Has: true, State: apiclient.Live}}
}
