package tui

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/tui/tuitest"
)

// x opens a filterable list of what the screen can do; each entry shows its
// key, and enter, the key itself or a click runs it.
func TestActionsMenuOnAScreen(t *testing.T) {
	f := tuitest.NewFake()
	a, _ := mouseApp(t, f, 80, 24)
	m := press(t, a, "x")
	menu, ok := m.(*App).modal.(*menuModal)
	if !ok {
		t.Fatalf("x opened %T", m.(*App).modal)
	}
	fr := tuitest.Frame(m)
	for _, want := range []string{"[a]", "add a box", "[r]", "refresh now", "[D]", "[esc] close"} {
		if !strings.Contains(fr, want) {
			t.Errorf("menu lacks %q:\n%s", want, fr)
		}
	}
	// Filtering narrows the list; esc clears the filter before it closes.
	m = press(t, m, "/", "r", "e", "-", "p")
	if vis := menu.visible(); len(vis) != 1 || !strings.Contains(vis[0].label, "re-pair") {
		t.Fatalf("filter re-p: %+v", vis)
	}
	m = press(t, m, "esc")
	if m.(*App).modal == nil || len(menu.visible()) != len(menu.all) {
		t.Fatal("esc did not clear the filter first")
	}
	// A click on an entry runs it as its key does.
	x, y := at(t, m, "refresh now")
	m = click(t, m, x, y)
	if m.(*App).modal != nil || !f.Called("Fleet") {
		t.Fatalf("clicking refresh: modal %T calls %v", m.(*App).modal, f.Calls)
	}
	// The menu's entry for a destructive action still asks for the name.
	m = press(t, m, "x", "D")
	if _, ok := m.(*App).modal.(*confirmModal); !ok {
		t.Fatalf("remove from the menu: %T", m.(*App).modal)
	}
}

// Every screen has a menu except Settings, whose rows are the menu.
func TestActionsMenuHasEntriesOnEveryScreen(t *testing.T) {
	for _, k := range []string{"1", "2", "3", "4", "5", "6", "7"} {
		m := press(t, newTestApp(t, tuitest.NewFake(), 80, 24, "unicode"), k, "x")
		menu, ok := m.(*App).modal.(*menuModal)
		if !ok || len(menu.all) == 0 {
			t.Errorf("%s: x opened %T", k, m.(*App).modal)
		}
	}
}

// An error stays through key presses until esc, says "error:", is written
// to the activity log, and goes by itself after a while.
func TestErrorsStayUntilDismissedAndAreLogged(t *testing.T) {
	a := newTestApp(t, tuitest.NewFake(), 80, 24, "unicode")
	m, _ := tuitest.Send(a, errMsg{what: "refresh", err: errors.New("the server said no")})
	fr := tuitest.Frame(m)
	if !strings.Contains(fr, "error: refresh: the server said no") {
		t.Fatalf("no error line:\n%s", fr)
	}
	if n := len(strings.Split(fr, "\n")); n != 24 {
		t.Fatalf("%d lines", n)
	}
	m = press(t, m, "j", "k")
	if !strings.Contains(tuitest.Frame(m), "error: refresh") {
		t.Fatal("a key press cleared the error")
	}
	ev := m.(*App).log.ev
	if len(ev) == 0 || ev[len(ev)-1].Sev != logWarn || !strings.Contains(ev[len(ev)-1].Text, "error: refresh: the server said no") {
		t.Fatalf("not in the activity log: %+v", ev)
	}
	m = press(t, m, "esc")
	if strings.Contains(tuitest.Frame(m), "error:") || m.(*App).modal != nil {
		t.Fatal("esc did not dismiss the error")
	}
	// A long error wraps to two lines and still fits.
	long := strings.Repeat("word ", 30)
	m, _ = tuitest.Send(m, errMsg{what: "refresh", err: errors.New(long)})
	fr = tuitest.Frame(m)
	if n := len(strings.Split(fr, "\n")); n != 24 || strings.Count(fr, "error:") != 1 {
		t.Fatalf("long error:\n%s", fr)
	}
	for _, l := range strings.Split(fr, "\n") {
		if ansi.StringWidth(l) > 80 {
			t.Fatalf("a line is wider than 80: %q", l)
		}
	}
	// It expires on the clock, and a click dismisses it too.
	app := m.(*App)
	now := testNow
	app.now = func() time.Time { return now }
	app.flashAt = now.Add(-errorFor - time.Second)
	m, _ = tuitest.Send(m, tickMsg(now))
	if strings.Contains(tuitest.Frame(m), "error:") {
		t.Fatal("the error never expires")
	}
}

// A notice is logged as well, and goes on the next key.
func TestNoticesAreLoggedAndGoOnTheNextKey(t *testing.T) {
	a := newTestApp(t, tuitest.NewFake(), 80, 24, "unicode")
	m, _ := tuitest.Send(a, flashMsg("copied the line"))
	if l := lastLine(tuitest.Frame(m)); !strings.Contains(l, "copied the line") || !strings.HasSuffix(strings.TrimSpace(l), "[?] more") {
		t.Fatalf("bottom line %q", l)
	}
	ev := m.(*App).log.ev
	if len(ev) == 0 || ev[len(ev)-1].Text != "copied the line" || ev[len(ev)-1].Sev != logInfo {
		t.Fatalf("log %+v", ev)
	}
	m = press(t, m, "j")
	if strings.Contains(tuitest.Frame(m), "copied the line") {
		t.Fatal("the notice stayed after a key")
	}
}

// The footer fits and always ends with a clickable "[?] more", on every
// screen and at every width; the "[?] more" button opens help.
func TestFooterEndsWithMore(t *testing.T) {
	for _, w := range []int{80, 100, 120} {
		for _, k := range []string{"1", "2", "3", "4", "5", "6", "7", "8"} {
			a, _ := mouseApp(t, tuitest.NewFake(), w, 24)
			m := press(t, a, k)
			l := lastLine(tuitest.Frame(m))
			if ansi.StringWidth(l) > w || !strings.HasSuffix(strings.TrimSpace(l), "[?] more") {
				t.Errorf("%d screen %s: footer %q", w, k, l)
			}
		}
	}
	a, _ := mouseApp(t, tuitest.NewFake(), 80, 24)
	x, y := at(t, a, "[?] more")
	if m := click(t, a, x+1, y); m.(*App).modal == nil {
		t.Fatal("clicking [?] more did not open help")
	}
}

// Fleet's footer leads with the primary actions.
func TestFleetFooterLeadsWithPrimaryActions(t *testing.T) {
	m := press(t, newTestApp(t, tuitest.NewFake(), 80, 24, "unicode"))
	want := "[enter] open  [a] add  [p] pair  [/] filter  [s] sort  [?] more"
	if l := lastLine(tuitest.Frame(m)); !strings.Contains(l, want) {
		t.Fatalf("footer %q lacks %q", l, want)
	}
}

// g and i are in the ? list whatever the setup state.
func TestHelpAlwaysListsGuideAndInvite(t *testing.T) {
	m := press(t, newTestApp(t, tuitest.NewFake(), 80, 24, "unicode"), "?")
	fr := tuitest.Frame(m)
	for _, want := range []string{"setup guide", "invite a box"} {
		if !strings.Contains(fr, want) {
			t.Errorf("help lacks %q:\n%s", want, fr)
		}
	}
}

// Taking a VPN tunnel down asks first, and a click on the question's n
// cancels; nothing is sent.
func TestVPNDownAsksFirst(t *testing.T) {
	f := vpnFake()
	f.Routes["POST /api/vpns/office/down"] = map[string]any{}
	m := press(t, vpnApp(t, f), "t")
	if _, ok := m.(*App).modal.(*yesNoModal); !ok || f.Called("Do POST /api/vpns/office/down") {
		t.Fatalf("down ran without a question: %T", m.(*App).modal)
	}
	if m = press(t, m, "n"); f.Called("Do POST /api/vpns/office/down") || m.(*App).modal != nil {
		t.Fatalf("n did not cancel: %v", f.Calls)
	}
	m = press(t, m, "t", "y")
	if !f.Called("Do POST /api/vpns/office/down") {
		t.Fatalf("y did not take it down: %v", f.Calls)
	}
}

var _ = api.UIPrefs{}
