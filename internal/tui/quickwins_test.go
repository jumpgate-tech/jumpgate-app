package tui

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/apiclient"
	"github.com/valve-tech/jumpgate/internal/tui/tuitest"
)

// The inbox puts what needs a decision first whatever order the server sent,
// and keeps the cursor on the job it was on.
func TestJobsNeedsDecisionFirstAndTheCursorStays(t *testing.T) {
	f := tuitest.NewFake()
	in := sampleInbox()
	in.Jobs[0], in.Jobs[1] = in.Jobs[1], in.Jobs[0] // the running job first
	f.JobsCh <- apiclient.Update[api.JobsInbox]{Value: in, Has: true, State: apiclient.Live}
	m := press(t, newTestApp(t, f, 80, 24, "unicode"), "2")
	js := m.(*App).screens[scrJobs].(*jobsScreen)
	if js.in.Jobs[0].State != "needs-decision" {
		t.Fatalf("order: %s first", js.in.Jobs[0].State)
	}
	m = press(t, m, "j") // select box-b
	if j, _ := js.selected(); j.Target != "box-b" {
		t.Fatalf("selected %s", j.Target)
	}
	// A new reading in the server's order: the cursor stays on box-b.
	m, _ = tuitest.Send(m, jobsMsg{gen: js.gen, u: apiclient.Update[api.JobsInbox]{Value: in, Has: true, State: apiclient.Live}})
	if j, _ := js.selected(); j.Target != "box-b" {
		t.Fatalf("after a new reading selected %s", j.Target)
	}
	_ = m
}

func TestJobsEmptyState(t *testing.T) {
	f := tuitest.NewFake()
	f.JobsCh <- apiclient.Update[api.JobsInbox]{Value: api.JobsInbox{At: testNow}, Has: true, State: apiclient.Live}
	m := press(t, newTestApp(t, f, 80, 24, "unicode"), "2")
	fr := tuitest.Frame(m)
	if !strings.Contains(fr, "No jobs.  [n] start one") {
		t.Fatalf("frame:\n%s", fr)
	}
	// The button works by click: n starts one.
	a := m.(*App)
	x, y := at(t, a, "[n] start one")
	if got := click(t, a, x+1, y); got.(*App).modal == nil {
		t.Fatalf("clicking [n] opened nothing:\n%s", tuitest.Frame(got))
	}
}

// A job's lease says whose controller is silent and offers the key as a button.
func TestJobsSilentControllerWords(t *testing.T) {
	fr := tuitest.Frame(jobsApp(t, tuitest.NewFake(), "ascii"))
	if !strings.Contains(fr, "controller silent 10m | [a] take over") {
		t.Fatalf("frame:\n%s", fr)
	}
}

// guide, invite and unlock are palette commands, and tab completes a command
// or a box name.
func TestPaletteGuideInviteUnlockAndTab(t *testing.T) {
	for line, want := range map[string]func(*App) bool{
		"guide":  func(a *App) bool { _, ok := a.modal.(*guideModal); return ok },
		"unlock": func(a *App) bool { _, ok := a.modal.(*unlockModal); return ok },
		"invite": func(a *App) bool { return a.inviteOn },
	} {
		a := newTestApp(t, tuitest.NewFake(), 80, 24, "unicode")
		m := press(t, typed(t, press(t, a, ":"), line), "enter")
		if !want(m.(*App)) {
			t.Errorf("%q: modal %T inviteOn %v", line, m.(*App).modal, m.(*App).inviteOn)
		}
	}
	// A unique prefix completes, with a space after it.
	a := newTestApp(t, tuitest.NewFake(), 80, 24, "unicode")
	m := press(t, typed(t, press(t, a, ":"), "unl"), "tab")
	if v := m.(*App).modal.(*paletteModal).in.Value(); v != "unlock " {
		t.Fatalf("unl + tab = %q", v)
	}
	// A box name completes after a command that takes one, and an ambiguous
	// prefix fills in what the names share and lists them.
	m = press(t, typed(t, press(t, sendFleet(t, newTestApp(t, tuitest.NewFake(), 80, 24, "unicode"), sampleFleet()), ":"), "logs lap"), "tab")
	if v := m.(*App).modal.(*paletteModal).in.Value(); v != "logs laptop " {
		t.Fatalf("logs lap + tab = %q", v)
	}
	m = press(t, typed(t, press(t, sendFleet(t, newTestApp(t, tuitest.NewFake(), 80, 24, "unicode"), sampleFleet()), ":"), "host box"), "tab")
	p := m.(*App).modal.(*paletteModal)
	if p.in.Value() != "host box-" || !strings.Contains(p.hint, "box-a") || !strings.Contains(p.hint, "box-b") {
		t.Fatalf("host box + tab = %q hint %q", p.in.Value(), p.hint)
	}
	if fr := tuitest.Frame(m); !strings.Contains(fr, "matches: box-a  box-b") {
		t.Fatalf("frame:\n%s", fr)
	}
	// Services complete too: "restart e" is "restart exec".
	m = press(t, typed(t, press(t, sendFleet(t, newTestApp(t, tuitest.NewFake(), 80, 24, "unicode"), sampleFleet()), ":"), "restart e"), "tab")
	if v := m.(*App).modal.(*paletteModal).in.Value(); v != "restart exec " {
		t.Fatalf("restart e + tab = %q", v)
	}
}

// Diagnostics read as one line per check, for the server's real report and
// for the plainer name/ok shape.
func TestDiagnosticsLines(t *testing.T) {
	real := json.RawMessage(`{"at":"2026-10-29T10:00:00Z","trigger":"manual","items":[
		{"ID":"rpc","Title":"RPC answers","Status":"pass","Detail":""},
		{"ID":"peers","Title":"Peers","Status":"fail","Detail":"0 peers","Fix":"open port 30303"}]}`)
	lines, ok := diagLines(real)
	got := strings.Join(lines, "\n")
	if !ok || !strings.Contains(got, "RPC answers  ok") || !strings.Contains(got, "Peers  FAIL  0 peers") || !strings.Contains(got, "fix: open port 30303") || strings.Contains(got, "items[") {
		t.Fatalf("%v\n%s", ok, got)
	}
	lines, ok = diagLines(json.RawMessage(`{"checks":[{"name":"rpc","ok":true},{"name":"disk","ok":false}]}`))
	if !ok || strings.Join(lines, "|") != "rpc  ok|disk  FAIL" {
		t.Fatalf("%v %q", ok, lines)
	}
	if _, ok := diagLines(json.RawMessage(`{"other":1}`)); ok {
		t.Fatal("an answer with no checks was taken for a report")
	}
	if _, ok := diagLines(json.RawMessage(`null`)); ok {
		t.Fatal("null was taken for a report")
	}
}

// Before the fleet has the box, the open box says so instead of "link n/a".
func TestBoxHeadExplainsAnUnreadLink(t *testing.T) {
	a := newTestApp(t, tuitest.NewFake(), 80, 24, "unicode")
	m, _ := tuitest.Send(a, tuitest.Key("enter")) // no fleet rows yet: nothing opens
	_ = m
	h := &hostScreen{id: "ghost", loaded: map[string]bool{}}
	if out := ansi.Strip(h.view(a, 78, 10)); strings.Contains(out, "n/a") || !strings.Contains(out, "link: not read yet") {
		t.Fatalf("head:\n%s", out)
	}
}

// The breadcrumb names the open box, not a generic word.
func TestBreadcrumbUsesTheBoxName(t *testing.T) {
	a := dashApp(t, tuitest.NewFake(), 120, 40, "ascii", "auto")
	m := press(t, a, "enter")
	if fr := tuitest.Frame(m); !strings.Contains(fr, "Fleet > box-a > Overview") {
		t.Fatalf("frame:\n%s", fr)
	}
}
