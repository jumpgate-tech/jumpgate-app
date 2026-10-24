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

// dashFleetBefore is the reading before dashFleet, so the activity log has
// something to say: box-b was up, box-a's exec was syncing, with no
// firewall reading and no job waiting.
func dashFleetBefore() api.Fleet {
	f := sampleFleet()
	f.Rows[1].Reachable = true
	f.Rows[0].Firewall = nil
	f.Rows[0].Status.Exec.State = api.SyncSyncing
	f.Rows[0].Jobs = &api.JobsSummary{Running: 1}
	return f
}

// dashApp is an App at w×h with the drawer pref set and the dashboard fleet
// delivered.
func dashApp(t *testing.T, f *tuitest.Fake, w, h int, glyphs, drawer string) *App {
	t.Helper()
	a := newTestApp(t, f, w, h, glyphs)
	m, _ := tuitest.Send(a, prefsMsg{p: api.UIPrefs{Glyphs: glyphs, ShowEstimates: true, Drawer: drawer}},
		fleetMsg{u: apiclient.Update[api.Fleet]{Value: dashFleetBefore(), Has: true, State: apiclient.Live}},
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
	if last := lastLine(fr); !strings.Contains(last, "jobs 1 running  [! 1 needs decision]  ! 3 alerts | ") {
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
	{"dash_140x40", 140, 40, "ascii", "auto"},
	{"dash_160x48", 160, 48, "ascii", "auto"},
}

// degradedFleet has every section missing, stale or failed: nothing in it
// may read as 0.
func degradedFleet() api.Fleet {
	old := testNow.Add(-9 * time.Minute)
	checked := testNow.Add(-3 * time.Second)
	return api.Fleet{At: testNow.Add(-5 * time.Second), IntervalSeconds: 15, Rows: []api.FleetRow{
		{ID: "box-a", Link: api.LinkAgent, Reachable: true, LastSeen: &checked, CheckedAt: &checked, Network: "PulseChain",
			StatusError: &api.ProbeError{Error: api.Error{Code: "rpc_timeout", Message: "rpc timed out"}},
			DiskError:   &api.ProbeError{Error: api.Error{Code: "disk_probe", Message: "du failed"}},
			Stale:       api.Stale{Status: true, Disk: true, Firewall: true}},
		{ID: "box-b", Link: api.LinkSSHOnly, LastSeen: &old, CheckedAt: &checked, Stale: api.Stale{Status: true},
			Error: &api.Error{Code: api.CodeUnreachable, Message: "dial tcp: connection refused"}},
		{ID: "box-c", Link: api.LinkAgent},
	}}
}

func TestDegradedGolden(t *testing.T) {
	a := newTestApp(t, tuitest.NewFake(), 120, 40, "ascii")
	m, _ := tuitest.Send(a, fleetMsg{u: apiclient.Update[api.Fleet]{Value: degradedFleet(), Has: true, State: apiclient.Live}})
	fr := tuitest.Frame(m)
	requireASCII(t, fr)
	kpi := strings.Join(strings.Split(fr, "\n")[1:5], "\n")
	for _, title := range []string{"Boxes", "Synced", "Head lag", "Peers", "Disk", "Gateways"} {
		if !strings.Contains(kpi, title) {
			t.Errorf("no %s tile:\n%s", title, kpi)
		}
	}
	if strings.Count(kpi, "n/a")+strings.Count(kpi, "FAIL") < 6 || bareZero.MatchString(kpi) {
		t.Errorf("an unknown reads as 0 or a tile has no n/a:\n%s", kpi)
	}
	tuitest.Golden(t, "dash_degraded_120x40", fr)
}

func TestHeaderAlertCount(t *testing.T) {
	for _, w := range []int{80, 120} {
		a := dashApp(t, tuitest.NewFake(), w, 40, "ascii", "auto")
		if h := ansi.Strip(a.header(w)); !strings.Contains(h, "! 3 alerts") || ansi.StringWidth(h) > w {
			t.Errorf("%d: header %q", w, h)
		}
	}
	if h := ansi.Strip(newTestApp(t, tuitest.NewFake(), 80, 24, "ascii").header(80)); strings.Contains(h, "alerts") || !strings.Contains(h, "connecting") {
		t.Errorf("before the fleet: %q", h)
	}
}

func TestDrawerLineAlertCount(t *testing.T) {
	if last := lastLine(tuitest.Frame(dashApp(t, tuitest.NewFake(), 80, 24, "ascii", "auto"))); !strings.Contains(last, "! 3 alerts") {
		t.Fatalf("last line %q", last)
	}
}

func TestGatewaysFeed(t *testing.T) {
	f := tuitest.NewFake()
	f.GatewaysV = []api.GatewaySummary{gw("rpc-main", "running")}
	a := newTestApp(t, f, 120, 40, "ascii")
	tuitest.Run(a.Init())
	if !f.Called("Gateways") {
		t.Fatal("Init does not read the gateways")
	}
	m, _ := tuitest.Send(a, gatewaysMsg{gws: f.GatewaysV})
	if a := m.(*App); !a.gwsHas || len(a.gws) != 1 {
		t.Fatalf("gws %v %+v", a.gwsHas, a.gws)
	}
	m, _ = tuitest.Send(m, gatewaysMsg{err: errors.New("boom")})
	if a := m.(*App); !a.gwsHas || len(a.gws) != 1 {
		t.Fatalf("an error dropped the list: %v %+v", a.gwsHas, a.gws)
	}
	if _, cmds := tuitest.Send(m, gwTickMsg{}); len(cmds) == 0 {
		t.Fatal("the 30 s tick does not reload")
	}
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
		for _, extra := range []string{"", "host", "skew", "[", "]"} {
			f := tuitest.NewFake()
			if extra == "skew" {
				f.ServerVersion = "v0.8.0"
			}
			var m tea.Model = dashApp(t, f, c.w, c.h, c.glyphs, c.pref)
			switch extra {
			case "host":
				m = press(t, m, "enter")
			case "[", "]":
				m = press(t, m, extra)
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
		if h := strings.TrimRight(ansi.Strip(m.(*App).header(120)), " "); !strings.HasSuffix(h, c.want) {
			t.Errorf("%v %v: header %q, want suffix %q", c.age, c.state, h, c.want)
		}
	}
}

// Below 100 columns the views are a tab row; at 100 and up a Views box,
// with the host list under it once a host is open.
func TestViewsSidebarAndTabs(t *testing.T) {
	fr := tuitest.Frame(dashApp(t, tuitest.NewFake(), 80, 24, "ascii", "auto"))
	if l := strings.Split(fr, "\n")[0]; !strings.HasPrefix(l, " jumpgate  [Fleet] Jobs Signers Gateways Settings") || !strings.HasSuffix(strings.TrimRight(l, " "), "5s ago") {
		t.Fatalf("header with tabs %q", l)
	}
	m := press(t, dashApp(t, tuitest.NewFake(), 120, 40, "ascii", "auto"), "enter")
	fr = tuitest.Frame(m)
	if !strings.Contains(fr, "+- Views") || !strings.Contains(fr, "+= > box-a") || !strings.Contains(fr, "|>+ box-a") {
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

// Below 100 columns the view tabs share the header line, so the body keeps
// the row a separate tab row would cost.
func TestNarrowHeaderFoldsTabs(t *testing.T) {
	for _, w := range []int{80, 90, 99} {
		a := dashApp(t, tuitest.NewFake(), w, 24, "ascii", "auto")
		h := ansi.Strip(a.header(w))
		if ansi.StringWidth(h) > w || !strings.Contains(h, "[Fleet] Jobs") {
			t.Errorf("%d: header %q", w, h)
		}
		if l := strings.Split(tuitest.Frame(a), "\n")[1]; strings.Contains(l, "Signers") {
			t.Errorf("%d: a tab row is back: %q", w, l)
		}
	}
	a := newTestApp(t, tuitest.NewFake(), 80, 24, "ascii")
	m, _ := tuitest.Send(a, fleetMsg{u: apiclient.Update[api.Fleet]{Value: sampleFleet(), Has: true, State: apiclient.Retrying, Err: errors.New("dial: refused")}})
	if h := ansi.Strip(m.(*App).header(80)); ansi.StringWidth(h) > 80 || !strings.Contains(h, "stale") {
		t.Errorf("retrying: header %q", h)
	}
}

func TestNarrowFramesKeepTheirLastRows(t *testing.T) {
	m, _ := tuitest.Send(newTestApp(t, tuitest.NewFake(), 80, 24, "ascii"), tuitest.Key("2"), tuitest.Key("?"))
	if fr := tuitest.Frame(m); !strings.Contains(fr, "r            reload") || strings.Contains(fr, "more (") {
		t.Errorf("help at 80x24 cut short:\n%s", fr)
	}
	if fr := tuitest.Frame(settingsApp(t, tuitest.NewFake())); !strings.Contains(fr, "[x] Security") {
		t.Errorf("settings at 80x24 lost its last row:\n%s", fr)
	}
}

// The fleet table keeps every default column, DISK and FW too, beside the
// sidebars at each dashboard size.
func TestFleetFrameKeepsDefaultColumns(t *testing.T) {
	for _, c := range []struct{ w, h int }{{100, 30}, {120, 40}, {140, 40}, {160, 48}} {
		fr := tuitest.Frame(dashApp(t, tuitest.NewFake(), c.w, c.h, "ascii", "auto"))
		var head, row string
		for _, l := range strings.Split(fr, "\n") {
			if strings.Contains(l, " HOST ") {
				head = l
			}
			if strings.Contains(l, ">box-a ") {
				row = l
			}
		}
		if !strings.Contains(head, "DISK") || !strings.Contains(head, "FW") || !strings.Contains(row, "warn") {
			t.Errorf("%dx%d: head %q row %q\n%s", c.w, c.h, head, row, fr)
		}
	}
}
