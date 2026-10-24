package tui

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/apiclient"
	"github.com/valve-tech/jumpgate/internal/tui/tuitest"
)

// dashFleet is the sample fleet with box-a running a job that needs a
// decision.
func dashFleet() api.Fleet {
	f := sampleFleet()
	f.Rows[0].Jobs = &api.JobsSummary{Running: 1, NeedsDecision: 1}
	return f
}

// dashApp is an App at w×h with the drawer pref set and the dashboard fleet
// delivered.
func dashApp(t *testing.T, f *tuitest.Fake, w, h int, glyphs, drawer string) *App {
	t.Helper()
	a := newTestApp(t, f, w, h, glyphs)
	m, _ := tuitest.Send(a, prefsMsg{p: api.UIPrefs{Glyphs: glyphs, ShowEstimates: true, Drawer: drawer}},
		fleetMsg{u: apiclient.Update[api.Fleet]{Value: dashFleet(), Has: true, State: apiclient.Live}})
	return m.(*App)
}

func lastLine(fr string) string {
	lines := strings.Split(fr, "\n")
	return lines[len(lines)-1]
}

func TestDrawerToggleSaves(t *testing.T) {
	f := tuitest.NewFake()
	a := dashApp(t, f, 120, 40, "ascii", "auto")
	if !a.drawerOpen() || !strings.Contains(tuitest.Frame(a), "+- Activity") {
		t.Fatalf("auto at 120x40 is not open:\n%s", tuitest.Frame(a))
	}
	m := press(t, a, ".")
	if fr := tuitest.Frame(m); strings.Contains(fr, "+- Activity") || f.LastPrefs.Drawer != "min" {
		t.Fatalf("after . saved %q:\n%s", f.LastPrefs.Drawer, fr)
	}
	m = press(t, m, ".")
	if fr := tuitest.Frame(m); !strings.Contains(fr, "+- Activity") || f.LastPrefs.Drawer != "open" {
		t.Fatalf("after . . saved %q:\n%s", f.LastPrefs.Drawer, fr)
	}
}

func TestDrawerAutoMinimizedAt80x24(t *testing.T) {
	a := dashApp(t, tuitest.NewFake(), 80, 24, "ascii", "auto")
	last := lastLine(tuitest.Frame(a))
	if !strings.HasPrefix(last, " jobs") || !strings.Contains(last, ". drawer") {
		t.Fatalf("last line %q", last)
	}
	if a.drawerOpen() {
		t.Fatal("auto at 80x24 is open")
	}
}

func TestDrawerLineFailsClosed(t *testing.T) {
	a := newTestApp(t, tuitest.NewFake(), 80, 24, "ascii")
	if l := a.drawerLine(80); !strings.HasPrefix(l, " jobs n/a") || !strings.Contains(l, "! alerts n/a") || strings.Contains(l, "needs decision") {
		t.Fatalf("before any jobs reading: %q", l)
	}
	m, _ := tuitest.Send(a, fleetMsg{u: apiclient.Update[api.Fleet]{Value: fleetWithJobs(0, 0), Has: true}})
	if l := m.(*App).drawerLine(80); !strings.HasPrefix(l, " jobs 0 running") || strings.Contains(l, "needs decision") {
		t.Fatalf("no jobs: %q", l)
	}
}

func TestDrawerHighlightsDecisionNoOpen(t *testing.T) {
	a := dashApp(t, tuitest.NewFake(), 80, 24, "ascii", "auto")
	fr := tuitest.Frame(a)
	if last := lastLine(fr); !strings.Contains(last, "jobs 1 running  [! 1 needs decision]  ! alerts n/a | ") {
		t.Fatalf("last line %q", last)
	}
	if strings.Contains(fr, "Activity") || a.drawerOpen() {
		t.Fatalf("the drawer opened itself:\n%s", fr)
	}
}

// Review Focus 2: in a text field . is a dot.
func TestDrawerKeyIgnoredWhileCapturing(t *testing.T) {
	f := tuitest.NewFake()
	m := press(t, dashApp(t, f, 80, 24, "ascii", "auto"), "/")
	m = typed(t, m, "box.a")
	if f.Called("SavePrefs") {
		t.Fatal(". in the filter saved the drawer")
	}
	if v := m.(*App).screens[scrFleet].(*fleetScreen).filter.Value(); v != "box.a" {
		t.Fatalf("filter %q", v)
	}
}

// Review Focus 3: a refused save says so, and the server's state comes back.
func TestDrawerSaveRefused(t *testing.T) {
	f := tuitest.NewFake()
	f.Err["SavePrefs"] = errors.New("disk full")
	a := dashApp(t, f, 120, 40, "ascii", "auto")
	m := press(t, a, ".").(*App)
	if !strings.Contains(m.flash, "could not save the display setting") || !strings.Contains(tuitest.Frame(m), "could not save the display setting") {
		t.Fatalf("flash %q:\n%s", m.flash, tuitest.Frame(m))
	}
	if !f.Called("Prefs") || m.prefs.Drawer != "auto" || !m.drawerOpen() {
		t.Fatalf("state not re-read: drawer %q open %v", m.prefs.Drawer, m.drawerOpen())
	}
}

// The open drawer has a line per box with jobs, then the status line.
func TestDrawerViewLines(t *testing.T) {
	a := dashApp(t, tuitest.NewFake(), 120, 40, "ascii", "open")
	a.sel = "box-a"
	v := ansi.Strip(a.drawerView(rect{0, 0, 120, 5}, false))
	for _, want := range []string{"+- Activity", "jobs  box-a  1 running | 1 needs decision", "box-a | gate | server live | updated 5s ago"} {
		if !strings.Contains(v, want) {
			t.Errorf("drawer lacks %q:\n%s", want, v)
		}
	}
	c := ansi.Strip(a.drawerView(rect{0, 0, 80, 1}, true))
	if strings.Contains(c, "Activity") || !strings.Contains(c, "jobs  box-a") || strings.Count(c, "\n") != 0 {
		t.Fatalf("compact %q", c)
	}
}

var dashCases = []struct {
	name         string
	w, h         int
	glyphs, pref string
}{
	{"dash_120x40", 120, 40, "ascii", "auto"},
	{"dash_120x40_drawer_min", 120, 40, "ascii", "min"},
	{"dash_100x30", 100, 30, "ascii", "auto"},
	{"dash_80x24", 80, 24, "ascii", "auto"},
	{"dash_80x24_drawer_open", 80, 24, "ascii", "open"},
	{"dash_unicode_120x40", 120, 40, "unicode", "auto"},
}

func TestDashboardGoldens(t *testing.T) {
	for _, c := range dashCases {
		t.Run(c.name, func(t *testing.T) {
			fr := tuitest.Frame(dashApp(t, tuitest.NewFake(), c.w, c.h, c.glyphs, c.pref))
			if c.glyphs == "ascii" {
				requireASCII(t, fr)
			}
			tuitest.Golden(t, c.name, fr)
		})
	}
}

// Every golden size: exactly h lines, none wider than w; also with a host
// open and with the skew banner.
func TestFrameLinesFitWindow(t *testing.T) {
	for _, c := range dashCases {
		for _, extra := range []string{"", "host", "skew"} {
			f := tuitest.NewFake()
			if extra == "skew" {
				f.ServerVersion = "v0.8.0"
			}
			var m tea.Model = dashApp(t, f, c.w, c.h, c.glyphs, c.pref)
			if extra == "host" {
				m = press(t, m, "enter")
			}
			fr := tuitest.Frame(m)
			lines := strings.Split(fr, "\n")
			if len(lines) != c.h {
				t.Errorf("%s %s: %d lines:\n%s", c.name, extra, len(lines), fr)
			}
			for _, l := range lines {
				if ansi.StringWidth(l) > c.w {
					t.Errorf("%s %s: line wider than %d: %q", c.name, extra, c.w, l)
				}
			}
		}
	}
}

// Review Focus 5: help draws inside main, with the header and drawer around it.
func TestModalInsideFrame(t *testing.T) {
	m := press(t, dashApp(t, tuitest.NewFake(), 120, 40, "ascii", "auto"), "?")
	fr := tuitest.Frame(m)
	lines := strings.Split(fr, "\n")
	if !strings.HasPrefix(lines[0], " jumpgate  server live") || !strings.Contains(fr, "+- Activity") || !strings.Contains(fr, "keys everywhere") || !strings.Contains(fr, "+- Views") {
		t.Fatalf("frame:\n%s", fr)
	}
	if m.(*App).modal == nil {
		t.Fatal("help is not open")
	}
}

func TestHeaderAge(t *testing.T) {
	a := newTestApp(t, tuitest.NewFake(), 80, 24, "ascii")
	if h := a.header(80); !strings.Contains(h, "connecting") {
		t.Fatalf("before the fleet: %q", h)
	}
	for _, c := range []struct {
		age   time.Duration
		state apiclient.ConnState
		want  string
	}{
		{5 * time.Second, apiclient.Live, "updated 5s ago"},
		{40 * time.Second, apiclient.Live, "updated 40s ago ~"},
		{70 * time.Second, apiclient.Live, "stale, updated 1m ago"},
		{5 * time.Second, apiclient.Retrying, "stale, updated 5s ago"},
	} {
		fl := sampleFleet()
		fl.At = testNow.Add(-c.age)
		m, _ := tuitest.Send(a, fleetMsg{u: apiclient.Update[api.Fleet]{Value: fl, Has: true, State: c.state}})
		if h := strings.TrimRight(ansi.Strip(m.(*App).header(80)), " "); !strings.HasSuffix(h, c.want) {
			t.Errorf("%v %v: header %q, want suffix %q", c.age, c.state, h, c.want)
		}
	}
}

// Below 100 columns the views are a tab row; at 100 and up a Views box,
// with the host list under it once a host is open.
func TestViewsSidebarAndTabs(t *testing.T) {
	fr := tuitest.Frame(dashApp(t, tuitest.NewFake(), 80, 24, "ascii", "auto"))
	if l := strings.Split(fr, "\n")[1]; !strings.Contains(l, "[Fleet] Hosts Signers Gateways Jobs Settings") {
		t.Fatalf("tab row %q", l)
	}
	m := press(t, dashApp(t, tuitest.NewFake(), 120, 40, "ascii", "auto"), "enter")
	fr = tuitest.Frame(m)
	if !strings.Contains(fr, "+- Views") || !strings.Contains(fr, "+- box-a") || !strings.Contains(fr, "|>box-a") {
		t.Fatalf("frame:\n%s", fr)
	}
}

func TestStatusLineShowsTheConnection(t *testing.T) {
	a := newTestApp(t, tuitest.NewFake(), 80, 24, "unicode")
	m, _ := tuitest.Send(a, fleetMsg{u: apiclient.Update[api.Fleet]{State: apiclient.Retrying, Err: errors.New("dial: refused")}})
	if s := m.(*App).statusLine(); !strings.Contains(s, "server retrying (dial: refused)") {
		t.Fatalf("status %q", s)
	}
}
