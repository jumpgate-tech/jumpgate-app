package tui

import (
	"errors"
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
	a := dashApp(t, tuitest.NewFake(), 140, 40, "ascii", "auto")
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
	a := dashApp(t, tuitest.NewFake(), 140, 40, "ascii", "auto")
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
	if w := press(t, dashApp(t, tuitest.NewFake(), 120, 40, "ascii", "auto"), "]").(*App); !w.logOver || w.focus != focusLog || !strings.Contains(tuitest.Frame(w), "Log: box-a") {
		t.Fatalf("] at 120 opens no overlay: %v focus %v", w.logOver, w.focus)
	}
	a := dashApp(t, tuitest.NewFake(), 80, 24, "ascii", "auto")
	m := press(t, a, "]")
	if fr := tuitest.Frame(m); !strings.Contains(fr, "Log: box-a") || m.(*App).focus != focusLog {
		t.Fatalf("] at 80x24 (focus %v):\n%s", m.(*App).focus, fr)
	}
	m = press(t, m, "esc")
	if fr := tuitest.Frame(m); strings.Contains(fr, "Log:") || m.(*App).focus != focusMain {
		t.Fatalf("esc left the overlay:\n%s", fr)
	}
	if w := press(t, dashApp(t, tuitest.NewFake(), 140, 40, "ascii", "auto"), "]").(*App); w.logOver || w.focus != focusLog {
		t.Fatalf("] at 140: overlay %v focus %v", w.logOver, w.focus)
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

// stepInbox is the sample inbox with box-b's running job at step (0-based).
func stepInbox(step int) api.JobsInbox {
	in := sampleInbox()
	in.Jobs[1].Step = step
	return in
}

// The real program loop: the fake's job advances a step and its bar moves.
func TestDrawerJobBarAdvances(t *testing.T) {
	f := tuitest.NewFake()
	f.FleetCh <- apiclient.Update[api.Fleet]{Value: dashFleet(), Has: true, State: apiclient.Live}
	f.JobsCh <- apiclient.Update[api.JobsInbox]{Value: stepInbox(3), Has: true, State: apiclient.Live}
	a := New(Options{Backend: f, GOOS: "linux", Getenv: func(string) string { return "" }, Now: func() time.Time { return testNow }})
	tm := teatest.NewTestModel(t, a, teatest.WithInitialTermSize(120, 40), teatest.WithProgramOptions(tea.WithColorProfile(colorprofile.ASCII)))
	teatest.WaitFor(t, tm.Output(), func(b []byte) bool { return strings.Contains(string(b), "Fleet") }, teatest.WithDuration(3*time.Second))
	tm.Send(tuitest.Key("2"))
	before := "job  box-b      migrate-names  step 4/8     ["
	teatest.WaitFor(t, tm.Output(), func(b []byte) bool { return strings.Contains(string(b), before) }, teatest.WithDuration(3*time.Second))
	f.JobsCh <- apiclient.Update[api.JobsInbox]{Value: stepInbox(4), Has: true, State: apiclient.Live}
	deadline := time.Now().Add(3 * time.Second)
	for len(f.JobsCh) > 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	time.Sleep(200 * time.Millisecond) // the relay hands it on; the program draws it
	tm.Send(tuitest.Key("q"))
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
	fr := tuitest.Frame(tm.FinalModel(t))
	line := ""
	for _, l := range strings.Split(fr, "\n") {
		if strings.Contains(l, "job  box-b") {
			line = l
		}
	}
	if !strings.Contains(line, "step 5/8     [") || strings.Count(line, unicodeGlyphs.Full) != 10 {
		t.Fatalf("the bar did not move (4/8 had 8 cells): %q\n%s", line, fr)
	}
}

// jobsUpdate is the inbox stream's next message for the Jobs view now.
func jobsUpdate(a *App, u apiclient.Update[api.JobsInbox]) jobsMsg {
	return jobsMsg{gen: a.screens[scrJobs].(*jobsScreen).gen, u: u}
}

// A dropped stream marks its bars stale; the next live reading clears it.
func TestDrawerBarsStaleThenRecover(t *testing.T) {
	fl := jobsDashFleet()
	fl.Rows[1].Stale = api.Stale{}
	a := jobsDashApp(t, tuitest.NewFake(), 120, 40, "auto")
	m, _ := tuitest.Send(a, fleetMsg{u: apiclient.Update[api.Fleet]{Value: fl, Has: true, State: apiclient.Live}})
	jobLine, syncLine := "step 4/8     [########............]", "21.1M/21.3M  [####################] 99.1%"
	check := func(name string, m tea.Model, jobStale, syncStale bool) {
		t.Helper()
		fr := tuitest.Frame(m)
		requireASCII(t, fr)
		for _, c := range []struct {
			line  string
			stale bool
		}{{jobLine, jobStale}, {syncLine, syncStale}} {
			found := false
			for _, l := range strings.Split(fr, "\n") {
				if strings.Contains(l, c.line) {
					found = true
					if strings.Contains(l, "~ stale") != c.stale {
						t.Errorf("%s: %q stale %v", name, l, !c.stale)
					}
				}
			}
			if !found {
				t.Errorf("%s: no %q:\n%s", name, c.line, fr)
			}
		}
	}
	check("live", m, false, false)
	m, _ = tuitest.Send(m, jobsUpdate(m.(*App), apiclient.Update[api.JobsInbox]{State: apiclient.Retrying, Err: errors.New("stream dropped")}))
	check("jobs dropped", m, true, false)
	m, _ = tuitest.Send(m, jobsUpdate(m.(*App), apiclient.Update[api.JobsInbox]{Value: sampleInbox(), Has: true, State: apiclient.Live}))
	check("jobs back", m, false, false)
	m, _ = tuitest.Send(m, fleetMsg{u: apiclient.Update[api.Fleet]{State: apiclient.Retrying, Err: errors.New("dial: refused")}})
	check("fleet dropped", m, false, true)
	m, _ = tuitest.Send(m, fleetMsg{u: apiclient.Update[api.Fleet]{Value: fl, Has: true, State: apiclient.Live}})
	check("fleet back", m, false, false)
}
