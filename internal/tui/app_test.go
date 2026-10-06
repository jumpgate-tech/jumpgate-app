package tui

import (
	"bytes"
	"context"
	"io"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/exp/teatest/v2"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/apiclient"
	"github.com/valve-tech/jumpgate/internal/tui/tuitest"
)

var _ Backend = (*tuitest.Fake)(nil)

var testNow = time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)

// newTestApp is an App over a fake at w×h with glyphs fixed, so frames are
// the same on every OS and terminal.
func newTestApp(t *testing.T, f *tuitest.Fake, w, h int, glyphs string) *App {
	t.Helper()
	a := New(Options{Backend: f, GOOS: "linux", Getenv: func(string) string { return "" }, Hostname: "laptop", Now: func() time.Time { return testNow }})
	t.Cleanup(a.cancel)
	m, _ := tuitest.Send(a, tea.WindowSizeMsg{Width: w, Height: h}, prefsMsg{p: api.UIPrefs{Glyphs: glyphs, ShowEstimates: true}})
	return m.(*App)
}

func TestShellFrame(t *testing.T) {
	a := newTestApp(t, tuitest.NewFake(), 80, 24, "unicode")
	m, _ := tuitest.Send(a, tuitest.Key("5"))
	frame := tuitest.Frame(m)
	if n := strings.Count(frame, "\n") + 1; n != 24 {
		t.Fatalf("frame has %d lines, want 24", n)
	}
	tuitest.Golden(t, "shell_jobs_80x24", frame)
}

// Review Focus 5: an ascii frame contains nothing but ASCII.
func TestASCIIFrameHasOnlyASCII(t *testing.T) {
	a := newTestApp(t, tuitest.NewFake(), 80, 24, "ascii")
	m, _ := tuitest.Send(a, tuitest.Key("5"))
	frame := tuitest.Frame(m)
	requireASCII(t, frame)
	tuitest.Golden(t, "shell_jobs_ascii", frame)
}

func requireASCII(t *testing.T, frame string) {
	t.Helper()
	for i, r := range frame {
		if r > 0x7e || (r < 0x20 && r != '\n') {
			t.Fatalf("non-ASCII %q at %d in:\n%s", r, i, frame)
		}
	}
}

// Review Focus 3: too small says so, keeps state, and comes back.
func TestTooSmallTerminalKeepsState(t *testing.T) {
	a := newTestApp(t, tuitest.NewFake(), 80, 24, "unicode")
	m, _ := tuitest.Send(a, tuitest.Key("5"), tea.WindowSizeMsg{Width: 60, Height: 20})
	if f := tuitest.Frame(m); !strings.Contains(f, "terminal too small (60x20); jumpgate needs 80x24") {
		t.Fatalf("frame:\n%s", f)
	}
	m, _ = tuitest.Send(m, tea.WindowSizeMsg{Width: 80, Height: 24})
	if m.(*App).active != scrJobs || !strings.Contains(tuitest.Frame(m), "Durable jobs") {
		t.Fatalf("state lost:\n%s", tuitest.Frame(m))
	}
}

// The size message itself fits the window it is shown in, however small:
// it wraps at word boundaries and is cut to the window's height, so the
// terminal never scrolls or wraps it.
func TestTooSmallFrameFitsTheWindow(t *testing.T) {
	for _, sz := range [][2]int{{79, 24}, {80, 23}, {30, 10}, {12, 3}, {5, 1}, {1, 1}} {
		w, h := sz[0], sz[1]
		a := newTestApp(t, tuitest.NewFake(), w, h, "unicode")
		fr := tuitest.Frame(a)
		lines := strings.Split(fr, "\n")
		if len(lines) > h {
			t.Errorf("%dx%d: %d lines:\n%s", w, h, len(lines), fr)
		}
		for _, l := range lines {
			if ansi.StringWidth(l) > w {
				t.Errorf("%dx%d: line %q is wider than the window", w, h, l)
			}
		}
		if w >= 30 && !strings.Contains(strings.Join(strings.Fields(fr), " "), "terminal too small") {
			t.Errorf("%dx%d: no size message:\n%s", w, h, fr)
		}
	}
}

func TestSkewBannerShowsBothVersions(t *testing.T) {
	f := tuitest.NewFake()
	f.ServerVersion = "v0.8.0"
	a := newTestApp(t, f, 80, 24, "unicode")
	fr := tuitest.Frame(a)
	if !strings.Contains(fr, "server v0.8.0, this jumpgate v-test") {
		t.Fatalf("frame:\n%s", fr)
	}
	if n := strings.Count(fr, "\n") + 1; n != 24 {
		t.Fatalf("frame with a banner has %d lines, want 24", n)
	}
}

func TestStatusBarShowsTheConnection(t *testing.T) {
	a := newTestApp(t, tuitest.NewFake(), 80, 24, "unicode")
	m, _ := tuitest.Send(a, fleetMsg{u: apiclient.Update[api.Fleet]{State: apiclient.Retrying, Err: context.DeadlineExceeded}})
	if fr := tuitest.Frame(m); !strings.Contains(fr, "server retrying") {
		t.Fatalf("frame:\n%s", fr)
	}
}

// The fleet's age comes from the injected clock, never the wall clock.
func TestStatusBarShowsTheDataAge(t *testing.T) {
	a := newTestApp(t, tuitest.NewFake(), 80, 24, "unicode")
	fl := api.Fleet{At: testNow.Add(-5 * time.Second)}
	m, _ := tuitest.Send(a, fleetMsg{u: apiclient.Update[api.Fleet]{State: apiclient.Live, Value: fl, Has: true}})
	lines := strings.Split(tuitest.Frame(m), "\n")
	if last := lines[len(lines)-1]; !strings.Contains(last, "server live") || !strings.HasSuffix(last, "5s ago") {
		t.Fatalf("status bar %q", last)
	}
}

func TestHelpListsGlobalKeys(t *testing.T) {
	a := newTestApp(t, tuitest.NewFake(), 80, 24, "unicode")
	// From the jobs screen, which has no keys of its own, so this frame does
	// not change as later tasks give screens their keys.
	m, _ := tuitest.Send(a, tuitest.Key("5"), tuitest.Key("?"))
	fr := tuitest.Frame(m)
	for _, want := range []string{"help", "quit", "screens", "command"} {
		if !strings.Contains(fr, want) {
			t.Errorf("help lacks %q:\n%s", want, fr)
		}
	}
	m, _ = tuitest.Send(m, tuitest.Key("esc"))
	if m.(*App).modal != nil {
		t.Fatal("esc did not close help")
	}
	tuitest.Golden(t, "help_80x24", fr)
}

// bindings is every key.Binding field of a keymap struct, found by
// reflection so that a binding added to the keymap and left out of the help
// fails here.
func bindings(keymap any) map[string]key.Binding {
	out := map[string]key.Binding{}
	v := reflect.ValueOf(keymap)
	for i := 0; i < v.NumField(); i++ {
		if b, ok := v.Field(i).Interface().(key.Binding); ok {
			out[v.Type().Field(i).Name] = b
		}
	}
	return out
}

// The help screen is generated from the keymaps: every global and
// navigation binding shows with its own help text. Restart shows only when
// it can act (a skewed server and a way to restart it).
func TestHelpIsGeneratedFromTheKeymaps(t *testing.T) {
	f := tuitest.NewFake()
	f.ServerVersion = "v0.8.0"
	a := New(Options{Backend: f, GOOS: "linux", Getenv: func(string) string { return "" },
		Restart: func(context.Context) (Backend, error) { return f, nil }})
	t.Cleanup(a.cancel)
	m, _ := tuitest.Send(a, tea.WindowSizeMsg{Width: 80, Height: 24}, prefsMsg{p: api.UIPrefs{Glyphs: "unicode"}}, tuitest.Key("5"), tuitest.Key("?"))
	fr := tuitest.Frame(m)
	all := bindings(globalKeys)
	for n, b := range bindings(navKeys) {
		all[n] = b
	}
	for name, b := range all {
		h := b.Help()
		if !regexp.MustCompile(`(?m)^\s*` + regexp.QuoteMeta(h.Key) + `\s+` + regexp.QuoteMeta(h.Desc) + `$`).MatchString(fr) {
			t.Errorf("help lacks %s (%q %q):\n%s", name, h.Key, h.Desc, fr)
		}
	}

	if !strings.Contains(fr, "press R to restart the server") {
		t.Errorf("the banner does not offer R:\n%s", fr)
	}

	// Without a way to restart, R does nothing, so neither help nor the
	// banner offers it.
	m, _ = tuitest.Send(newTestApp(t, f, 80, 24, "unicode"), tuitest.Key("5"), tuitest.Key("?"))
	fr = tuitest.Frame(m)
	if regexp.MustCompile(`(?m)^\s*R\s`).MatchString(fr) || strings.Contains(fr, "press R") {
		t.Errorf("R is offered with no way to restart:\n%s", fr)
	}
}

// Help under ascii glyphs is ASCII too: the arrows in key labels become
// words.
func TestHelpASCII(t *testing.T) {
	a := newTestApp(t, tuitest.NewFake(), 80, 24, "ascii")
	m, _ := tuitest.Send(a, tuitest.Key("5"), tuitest.Key("?"))
	fr := tuitest.Frame(m)
	requireASCII(t, fr)
	if !strings.Contains(fr, "up/k") {
		t.Fatalf("frame:\n%s", fr)
	}
}

func TestGlyphDetection(t *testing.T) {
	env := func(kv map[string]string) func(string) string { return func(k string) string { return kv[k] } }
	for name, c := range map[string]struct {
		pref, goos string
		env        map[string]string
		ascii      bool
	}{
		"mac auto":             {"auto", "darwin", nil, false},
		"legacy windows":       {"auto", "windows", nil, true},
		"windows terminal":     {"auto", "windows", map[string]string{"WT_SESSION": "x"}, false},
		"linux console":        {"auto", "linux", map[string]string{"TERM": "linux"}, true},
		"forced ascii":         {"auto", "linux", map[string]string{"JUMPGATE_ASCII": "1"}, true},
		"unicode wins on pref": {"unicode", "windows", nil, false},
		"ascii pref":           {"ascii", "darwin", nil, true},
	} {
		if g := DetectGlyphs(c.pref, c.goos, env(c.env)); g.ASCII != c.ascii {
			t.Errorf("%s: ascii %v", name, g.ASCII)
		}
	}
}

// Spec D9: mouse capture stays off unless the prefs turn it on, so native
// text selection works for copying fingerprints and commands.
func TestMouseOffByDefault(t *testing.T) {
	a := newTestApp(t, tuitest.NewFake(), 80, 24, "unicode")
	if mm := a.View().MouseMode; mm != tea.MouseModeNone {
		t.Fatalf("mouse mode %v by default", mm)
	}
	m, _ := tuitest.Send(a, prefsMsg{p: api.UIPrefs{Mouse: true}})
	if mm := m.View().MouseMode; mm != tea.MouseModeCellMotion {
		t.Fatalf("mouse mode %v with the pref on", mm)
	}
}

// The real program loop: q quits from a top-level screen.
func TestQuitWithQ(t *testing.T) {
	f := tuitest.NewFake()
	a := New(Options{Backend: f, GOOS: "linux", Getenv: func(string) string { return "" }})
	tm := teatest.NewTestModel(t, a, teatest.WithInitialTermSize(80, 24), teatest.WithProgramOptions(tea.WithColorProfile(colorprofile.ASCII)))
	teatest.WaitFor(t, tm.Output(), func(b []byte) bool { return strings.Contains(string(b), "Fleet") }, teatest.WithDuration(3*time.Second))
	tm.Send(tuitest.Key("q"))
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

// colourSGR matches an SGR sequence that sets a foreground or background
// colour (30–37, 39–47, 49, 90–97, 100–107, or 38;… / 48;…).
var colourSGR = regexp.MustCompile(`\x1b\[(?:[0-9;]*;)?(?:3[0-9]|4[0-9]|9[0-7]|10[0-7])(?:;[0-9;]*)?m`)

// programOutput runs the TUI's real program loop with env as its
// environment until the skew banner (drawn in a colour) is on screen, quits,
// and returns everything the program wrote.
func programOutput(t *testing.T, env []string) string {
	t.Helper()
	f := tuitest.NewFake()
	f.ServerVersion = "v0.8.0"
	a := New(Options{Backend: f, GOOS: "linux", Getenv: func(string) string { return "" }})
	tm := teatest.NewTestModel(t, a, teatest.WithInitialTermSize(80, 24), teatest.WithProgramOptions(tea.WithEnvironment(env)))
	var seen bytes.Buffer
	teatest.WaitFor(t, io.TeeReader(tm.Output(), &seen), func(b []byte) bool { return strings.Contains(string(b), "this jumpgate v-test") }, teatest.WithDuration(3*time.Second))
	tm.Send(tuitest.Key("ctrl+c"))
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
	rest, _ := io.ReadAll(tm.FinalOutput(t))
	return seen.String() + string(rest)
}

// Spec "Colour, glyphs and terminals": the TUI draws in colour on a colour
// terminal, and Bubble Tea's profile detection removes every colour under
// NO_COLOR or TERM=dumb. TTY_FORCE makes colorprofile treat the test's
// buffer as a terminal, so each case differs from the colour one only by
// the variable under test.
func TestColourFollowsTheEnvironment(t *testing.T) {
	if out := programOutput(t, []string{"TTY_FORCE=1", "TERM=xterm-256color"}); !colourSGR.MatchString(out) {
		t.Fatalf("no colour on a colour terminal: %q", out)
	}
	if out := programOutput(t, []string{"TTY_FORCE=1", "TERM=xterm-256color", "NO_COLOR=1"}); colourSGR.MatchString(out) {
		t.Fatalf("colour under NO_COLOR: %q", colourSGR.FindString(out))
	}
	if out := programOutput(t, []string{"TTY_FORCE=1", "TERM=dumb"}); colourSGR.MatchString(out) {
		t.Fatalf("colour under TERM=dumb: %q", colourSGR.FindString(out))
	}
}

type ctxKey struct{}

// A command keeps the context it was created with: work started before a
// server restart is cancelled with the old server and never runs on the new
// one, and reading a.ctx from the command goroutine would race the restart.
func TestCommandsKeepTheirContext(t *testing.T) {
	f := tuitest.NewFake()
	f.ServerVersion = "v0.8.0"
	var restartCtx context.Context
	a := New(Options{Backend: f, GOOS: "linux", Getenv: func(string) string { return "" },
		Restart: func(ctx context.Context) (Backend, error) { restartCtx = ctx; return f, nil }})
	t.Cleanup(func() { a.cancel() })
	a.ctx = context.WithValue(a.ctx, ctxKey{}, "old")

	var got context.Context
	work := a.do("work", func(ctx context.Context) tea.Msg { got = ctx; return nil })
	restart := a.restart()
	tuitest.Send(a, restartedMsg{be: f}) // replaces a.ctx
	work()
	restart()
	if got.Value(ctxKey{}) != "old" || got.Err() == nil {
		t.Fatalf("do ran with the new context (value %v, err %v)", got.Value(ctxKey{}), got.Err())
	}
	if restartCtx.Value(ctxKey{}) != "old" {
		t.Fatal("restart ran with the new context")
	}
}

// Init fetches the display prefs without blocking, and the answer applies.
func TestInitFetchesPrefs(t *testing.T) {
	f := tuitest.NewFake()
	f.PrefsV = api.UIPrefs{Mouse: true, Units: "GiB"}
	a := New(Options{Backend: f, GOOS: "linux", Getenv: func(string) string { return "" }, Now: func() time.Time { return testNow }})
	t.Cleanup(a.cancel)
	var m tea.Model = a
	for _, msg := range tuitest.Run(a.Init()) {
		m, _ = m.Update(msg)
	}
	if !f.Called("Prefs") {
		t.Fatal("Init did not fetch prefs")
	}
	if got := m.(*App).prefs; !got.Mouse || got.Units != "GiB" || got.RefreshSeconds != 15 {
		t.Fatalf("prefs %+v", got)
	}
}
