package tui

import (
	"os"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/valve-tech/jumpgate/internal/tui/tuitest"
)

// TestMain turns the modal key flush off: tests send keys microseconds
// apart, faster than any typist. TestModalFlush turns it back on.
func TestMain(m *testing.M) {
	modalFlush = 0
	os.Exit(m.Run())
}

// mouseApp is the dashboard with the mouse on, drawn once (a click follows
// a render), on a clock the test moves.
func mouseApp(t *testing.T, f *tuitest.Fake, w, h int) (*App, *time.Time) {
	t.Helper()
	a := dashApp(t, f, w, h, "ascii", "auto")
	p := a.prefs
	p.Mouse = true
	a.setPrefs(p)
	now := time.Unix(1_000_000, 0)
	a.clock = func() time.Time { return now }
	tuitest.Frame(a)
	return a, &now
}

func click(t *testing.T, m tea.Model, x, y int) tea.Model {
	t.Helper()
	m, cmds := tuitest.Send(m, tea.MouseClickMsg{X: x, Y: y, Button: tea.MouseLeft})
	m = tuitest.Settle(m, cmds...)
	tuitest.Frame(m)
	return m
}

// at finds s in the frame: its column and row.
func at(t *testing.T, m tea.Model, s string) (int, int) {
	t.Helper()
	fr := tuitest.Frame(m)
	for y, l := range strings.Split(fr, "\n") {
		if i := strings.Index(l, s); i >= 0 {
			return len([]rune(l[:i])), y
		}
	}
	t.Fatalf("no %q in:\n%s", s, fr)
	return 0, 0
}

func TestButtonAt(t *testing.T) {
	line := " [enter] open  [/] filter   [s] sort  [1-5] views  [####|...]  [esc] back"
	col := func(s string, off int) int { return strings.Index(line, s) + off }
	for _, c := range []struct {
		col  int
		want string
	}{{col("[enter]", 0), "enter"}, {col("open", 2), "enter"}, {col("[/]", 1), "/"}, {col("filter", 7), ""}, {col("[s]", 0), "s"},
		{col("[1-5]", 1), ""}, {col("[####", 2), ""}, {col("[esc]", 0), "esc"}, {col("back", 3), "esc"}, {0, ""}} {
		k, ok := buttonAt(line, c.col)
		got := ""
		if ok {
			got = k.String()
		}
		if got != c.want {
			t.Errorf("col %d: %q, want %q", c.col, got, c.want)
		}
	}
}

func TestClickNavItemOpensItsView(t *testing.T) {
	a, _ := mouseApp(t, tuitest.NewFake(), 120, 40)
	x, y := at(t, a, "Signers ")
	m := click(t, a, x, y)
	if got := m.(*App); got.active != scrSigners || got.focus != focusTree {
		t.Fatalf("active %v focus %v", got.active, got.focus)
	}
}

func TestClickTreeRowSelectsAndDoubleClickOpens(t *testing.T) {
	a, now := mouseApp(t, tuitest.NewFake(), 120, 40)
	x, y := at(t, a, "+ box-b")
	m := click(t, a, x, y)
	if m.(*App).sel != "box-b" || m.(*App).detail != nil {
		t.Fatalf("one click: sel %q detail %v", m.(*App).sel, m.(*App).detail)
	}
	*now = now.Add(time.Second) // too late to be a double click
	m = click(t, m, x, y)
	if m.(*App).detail != nil {
		t.Fatal("two slow clicks opened the box")
	}
	m = click(t, m, x, y)
	if h, ok := m.(*App).detail.(*hostScreen); !ok || h.id != "box-b" {
		t.Fatalf("double click did not open box-b: %v", m.(*App).detail)
	}
}

func TestDoubleClickFleetRowOpens(t *testing.T) {
	a, _ := mouseApp(t, tuitest.NewFake(), 120, 40)
	x, y := at(t, a, " box-b ")
	m := click(t, click(t, a, x+2, y), x+2, y)
	if h, ok := m.(*App).detail.(*hostScreen); !ok || h.id != "box-b" {
		t.Fatalf("detail %v", m.(*App).detail)
	}
	if !strings.Contains(tuitest.Frame(m), "Fleet > box-b > Overview") {
		t.Fatalf("no breadcrumb:\n%s", tuitest.Frame(m))
	}
}

func TestClickHintButtonPressesItsKey(t *testing.T) {
	a, _ := mouseApp(t, tuitest.NewFake(), 120, 40)
	x, y := at(t, a, "[/] filter")
	m := click(t, a, x+1, y)
	if !m.(*App).screens[scrFleet].(*fleetScreen).editing {
		t.Fatalf("[/] did not open the filter:\n%s", tuitest.Frame(m))
	}
}

func TestWheelScrollsThePaneUnderThePointer(t *testing.T) {
	a, _ := mouseApp(t, tuitest.NewFake(), 120, 40)
	x, y := at(t, a, " box-b ")
	m, _ := tuitest.Send(a, tea.MouseWheelMsg{X: x, Y: y, Button: tea.MouseWheelDown})
	if got := m.(*App).screens[scrFleet].(*fleetScreen).cursor; got != 1 {
		t.Fatalf("wheel down on main: cursor %d", got)
	}
	tx, ty := at(t, m, "fleet ")
	m, _ = tuitest.Send(m, tea.MouseWheelMsg{X: tx, Y: ty, Button: tea.MouseWheelDown})
	if m.(*App).sel != "laptop" {
		t.Fatalf("wheel on the tree did not move it: sel %q", m.(*App).sel)
	}
}

func TestClickDrawerTogglesIt(t *testing.T) {
	f := tuitest.NewFake()
	a, _ := mouseApp(t, f, 120, 40)
	_, y := at(t, a, "+- Activity")
	click(t, a, 10, y+1)
	if f.LastPrefs.Drawer != "min" {
		t.Fatalf("click on the open drawer saved %q", f.LastPrefs.Drawer)
	}
	g := tuitest.NewFake()
	b, _ := mouseApp(t, g, 80, 24)
	click(t, b, 3, 23)
	if g.LastPrefs.Drawer != "open" {
		t.Fatalf("click on the minimized drawer saved %q:\n%s", g.LastPrefs.Drawer, tuitest.Frame(b))
	}
	c, _ := mouseApp(t, tuitest.NewFake(), 80, 24)
	x, y := at(t, c, "needs decision]")
	if m := click(t, c, x, y); m.(*App).active != scrJobs {
		t.Fatalf("click on the decision count: active %v", m.(*App).active)
	}
}

func TestClickStripOpensItsView(t *testing.T) {
	a, _ := mouseApp(t, tuitest.NewFake(), 120, 40)
	x, _ := at(t, a, "- Gateways")
	m := click(t, a, x+3, 2)
	if m.(*App).active != scrGateways {
		t.Fatalf("Gateways tile: active %v", m.(*App).active)
	}
	m = click(t, m, 3, 2) // Boxes
	if m.(*App).active != scrFleet {
		t.Fatalf("Boxes tile: active %v", m.(*App).active)
	}
	b, _ := mouseApp(t, tuitest.NewFake(), 80, 24)
	x, y := at(t, b, "] 2 ")
	if m := click(t, b, x+2, y); m.(*App).active != scrJobs {
		t.Fatalf("header tab: active %v", m.(*App).active)
	}
}

// A click never confirms a typed confirmation, wherever it lands.
func TestClickNeverConfirms(t *testing.T) {
	a, now := mouseApp(t, tuitest.NewFake(), 120, 40)
	ran := false
	a.modal = newConfirm("Remove box-a", "this removes box-a", "box-a", func() tea.Cmd { ran = true; return nil })
	*now = now.Add(time.Second)
	fr := tuitest.Frame(a)
	var m tea.Model = a
	for y, l := range strings.Split(fr, "\n") {
		for x := range len(l) {
			m = click(t, m, x, y)
			*now = now.Add(time.Second)
		}
	}
	if ran || m.(*App).modal == nil {
		t.Fatalf("a click answered the confirmation (ran %v)", ran)
	}
}

// Keys typed ahead of a modal do not answer it.
func TestModalFlush(t *testing.T) {
	modalFlush = 150 * time.Millisecond
	defer func() { modalFlush = 0 }()
	a, now := mouseApp(t, tuitest.NewFake(), 120, 40)
	m := press(t, a, "?")
	if m.(*App).modal == nil {
		t.Fatal("? did not open help")
	}
	m = press(t, m, "esc")
	if m.(*App).modal == nil {
		t.Fatal("esc within 150 ms closed the modal")
	}
	*now = now.Add(200 * time.Millisecond)
	if m = press(t, m, "esc"); m.(*App).modal != nil {
		t.Fatal("esc after 150 ms did not close the modal")
	}
}

func TestFocusMarksAndEsc(t *testing.T) {
	a, _ := mouseApp(t, tuitest.NewFake(), 140, 40)
	if l := lastLine(tuitest.Frame(a)); !strings.Contains(l, "[enter] open") {
		t.Fatalf("main hints %q", l)
	}
	m := press(t, a, "tab") // the log
	if l := lastLine(tuitest.Frame(m)); !strings.Contains(l, "scroll") || strings.Contains(l, "open") {
		t.Fatalf("log hints %q", l)
	}
	m = press(t, m, "tab") // the nav
	if l := lastLine(tuitest.Frame(m)); !strings.Contains(l, "[space] expand") || strings.Contains(l, "scroll") {
		t.Fatalf("nav hints %q", l)
	}
	m = press(t, m, "tab", "enter") // main: open box-a
	if m.(*App).detail == nil || m.(*App).focus != focusMain {
		t.Fatalf("detail %v focus %v", m.(*App).detail, m.(*App).focus)
	}
	m = press(t, m, "esc") // closes the box first
	if m.(*App).detail != nil || m.(*App).focus != focusMain {
		t.Fatalf("esc: detail %v focus %v", m.(*App).detail, m.(*App).focus)
	}
	m = press(t, m, "esc") // then back to the nav
	if m.(*App).focus != focusTree {
		t.Fatalf("second esc: focus %v", m.(*App).focus)
	}
}

func TestSelectedRowAndNavItemAreReversed(t *testing.T) {
	a := dashApp(t, tuitest.NewFake(), 120, 40, "ascii", "auto")
	a.th = NewTheme()
	v := a.View().Content
	rev := 0
	for _, l := range strings.Split(v, "\n") {
		if strings.Contains(l, "\x1b[7m") && (strings.Contains(l, "> Fleet") || strings.Contains(l, ">box-a")) {
			rev++
		}
	}
	if rev < 2 {
		t.Fatalf("%d reversed lines among the nav item and the row", rev)
	}
}
