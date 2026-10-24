package tui

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/exp/teatest/v2"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/apiclient"
	"github.com/valve-tech/jumpgate/internal/tui/tuitest"
)

// Phase-3 flows: focus regions, the tree, the activity log and overlays.

func TestTabCyclesFocus(t *testing.T) {
	a := dashApp(t, tuitest.NewFake(), 120, 40, "ascii", "auto")
	if a.focus != focusMain {
		t.Fatalf("default focus %v", a.focus)
	}
	var m tea.Model = a
	for _, want := range []focusRegion{focusTree, focusLog, focusMain} {
		m = press(t, m, "tab")
		if got := m.(*App).focus; got != want {
			t.Fatalf("tab: focus %v, want %v", got, want)
		}
	}
	for _, want := range []focusRegion{focusLog, focusTree, focusMain} {
		m = press(t, m, "shift+tab")
		if got := m.(*App).focus; got != want {
			t.Fatalf("shift+tab: focus %v, want %v", got, want)
		}
	}
	if fr := tuitest.Frame(press(t, m, "tab")); !strings.Contains(fr, "+= > Views") {
		t.Fatalf("the focused tree is not marked:\n%s", fr)
	}
	narrow := press(t, dashApp(t, tuitest.NewFake(), 80, 24, "ascii", "auto"), "tab").(*App)
	if narrow.focus != focusMain {
		t.Fatalf("80x24 without overlays: focus %v", narrow.focus)
	}
}

func TestHostTabsStillOnArrows(t *testing.T) {
	m := press(t, dashApp(t, tuitest.NewFake(), 120, 40, "ascii", "auto"), "enter")
	h := m.(*App).detail.(*hostScreen)
	m = press(t, m, "right")
	if h.tab != 1 {
		t.Fatalf("right: tab %d", h.tab)
	}
	m = press(t, m, "tab")
	if h.tab != 1 || m.(*App).focus != focusTree {
		t.Fatalf("tab moved the host tab (%d) or not the focus (%v)", h.tab, m.(*App).focus)
	}
}

func TestTreeEnterOpensBoxAndLogFollows(t *testing.T) {
	a := dashApp(t, tuitest.NewFake(), 120, 40, "ascii", "auto")
	m := press(t, a, "tab", "k") // the tree; up to the fleet line
	if fr := tuitest.Frame(m); !m.(*App).selAll || !strings.Contains(fr, "Log: fleet") {
		t.Fatalf("fleet line: selAll %v\n%s", m.(*App).selAll, fr)
	}
	m = press(t, m, "j", "j") // box-a, box-b
	fs := m.(*App).screens[scrFleet].(*fleetScreen)
	if m.(*App).sel != "box-b" || fs.cursor != 1 {
		t.Fatalf("sel %q table cursor %d", m.(*App).sel, fs.cursor)
	}
	if fr := tuitest.Frame(m); !strings.Contains(fr, "Log: box-b") || !strings.Contains(fr, "box down") {
		t.Fatalf("the log does not follow:\n%s", fr)
	}
	m = press(t, m, "enter")
	if h, ok := m.(*App).detail.(*hostScreen); !ok || h.id != "box-b" {
		t.Fatalf("enter did not open box-b: %T", m.(*App).detail)
	}
	if fr := tuitest.Frame(m); !strings.Contains(fr, "Log: box-b") {
		t.Fatalf("detail log:\n%s", fr)
	}
}

func TestTreeSpaceExpands(t *testing.T) {
	f := treeFake()
	m := press(t, dashApp(t, f, 120, 40, "ascii", "auto"), "tab", "space")
	if !m.(*App).expanded["box-a"] || !f.Called("ListModules box-a") {
		t.Fatalf("space: expanded %v calls %v", m.(*App).expanded, f.Calls)
	}
	if fr := tuitest.Frame(m); !strings.Contains(fr, "exec") || !strings.Contains(fr, "mod msgboard") {
		t.Fatalf("no children:\n%s", fr)
	}
}

func TestResizeFoldsSidebarsKeepsSelection(t *testing.T) {
	m := press(t, dashApp(t, tuitest.NewFake(), 120, 40, "ascii", "auto"), "j", "tab")
	m, _ = tuitest.Send(m, tea.WindowSizeMsg{Width: 80, Height: 24})
	fr := tuitest.Frame(m)
	if strings.Contains(fr, "Views") || strings.Contains(fr, "Log:") || !strings.Contains(fr, ">box-b") {
		t.Fatalf("frame at 80x24:\n%s", fr)
	}
	if a := m.(*App); a.sel != "box-b" || a.focus != focusMain {
		t.Fatalf("sel %q focus %v", a.sel, a.focus)
	}
}

func TestLogOverlayToggles(t *testing.T) {
	a := dashApp(t, tuitest.NewFake(), 80, 24, "ascii", "auto")
	m := press(t, a, "]")
	if fr := tuitest.Frame(m); !strings.Contains(fr, "Log: box-a") || m.(*App).focus != focusLog {
		t.Fatalf("] at 80x24 (focus %v):\n%s", m.(*App).focus, fr)
	}
	m = press(t, m, "esc")
	if fr := tuitest.Frame(m); strings.Contains(fr, "Log:") || m.(*App).focus != focusMain {
		t.Fatalf("esc left the overlay:\n%s", fr)
	}
	if w := press(t, dashApp(t, tuitest.NewFake(), 120, 40, "ascii", "auto"), "]").(*App); w.logOver || w.focus != focusLog {
		t.Fatalf("] at 120: overlay %v focus %v", w.logOver, w.focus)
	}
}

// The real program loop: ] opens the log over main and esc closes it.
func TestLogOverlayProgram(t *testing.T) {
	f := tuitest.NewFake()
	f.FleetCh <- apiclient.Update[api.Fleet]{Value: dashFleet(), Has: true, State: apiclient.Live}
	a := New(Options{Backend: f, GOOS: "linux", Getenv: func(string) string { return "" }, Now: func() time.Time { return testNow }})
	tm := teatest.NewTestModel(t, a, teatest.WithInitialTermSize(80, 24), teatest.WithProgramOptions(tea.WithColorProfile(colorprofile.ASCII)))
	teatest.WaitFor(t, tm.Output(), func(b []byte) bool { return strings.Contains(string(b), "Fleet") }, teatest.WithDuration(3*time.Second))
	tm.Send(tuitest.Key("]"))
	teatest.WaitFor(t, tm.Output(), func(b []byte) bool { return strings.Contains(string(b), "Log: ") }, teatest.WithDuration(3*time.Second))
	tm.Send(tuitest.Key("esc"))
	tm.Send(tuitest.Key("q"))
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
	if m := tm.FinalModel(t).(*App); m.logOver {
		t.Fatal("esc did not close the overlay")
	}
}

func TestTreeOverlayToggles(t *testing.T) {
	m := press(t, dashApp(t, tuitest.NewFake(), 80, 24, "ascii", "auto"), "[")
	if fr := tuitest.Frame(m); !strings.Contains(fr, "laptop*") || !strings.Contains(fr, "Tree") || m.(*App).focus != focusTree {
		t.Fatalf("[ at 80x24:\n%s", fr)
	}
	m = press(t, m, "esc")
	if fr := tuitest.Frame(m); strings.Contains(fr, "Tree") {
		t.Fatalf("esc left the tree:\n%s", fr)
	}
}

func TestPhase3Goldens(t *testing.T) {
	for _, c := range []struct {
		name string
		w, h int
		keys []string
	}{
		{"dash_80x24_tree_overlay", 80, 24, []string{"["}},
		{"dash_80x24_ctx_overlay", 80, 24, []string{"]"}},
		{"dash_box_detail_120x40", 120, 40, []string{"enter"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			fr := tuitest.Frame(press(t, dashApp(t, tuitest.NewFake(), c.w, c.h, "ascii", "auto"), c.keys...))
			requireASCII(t, fr)
			lines := strings.Split(fr, "\n")
			if len(lines) != c.h {
				t.Errorf("%d lines", len(lines))
			}
			tuitest.Golden(t, c.name, fr)
		})
	}
}
