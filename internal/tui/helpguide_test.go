package tui

import (
	"regexp"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/charmbracelet/x/ansi"

	"github.com/valve-tech/jumpgate/internal/tui/tuitest"
)

// ? then F opens the guide on its topic list, which names every guide file.
func TestManualListsEveryTopic(t *testing.T) {
	m := press(t, newTestApp(t, tuitest.NewFake(), 80, 24, "unicode"), "2", "?", "F")
	if _, ok := m.(*App).modal.(*manualModal); !ok {
		t.Fatalf("modal %#v", m.(*App).modal)
	}
	fr := tuitest.Frame(m)
	for _, tp := range manualSource.Topics() {
		if !strings.Contains(fr, tp.Title) || !strings.Contains(fr, ":help "+tp.Name) {
			t.Errorf("the list lacks %s:\n%s", tp.Name, fr)
		}
	}
	tuitest.Golden(t, "manual_80x24", fr)

	m = press(t, m, "j", "enter")
	if g := m.(*App).modal.(*manualModal); !g.open || g.topics[g.cur].Name != "getting-started" {
		t.Fatalf("enter opened %#v", g)
	}
	m = press(t, m, "esc")
	if g := m.(*App).modal.(*manualModal); g.open {
		t.Fatal("esc on a page did not go back to the topics")
	}
	if m = press(t, m, "esc"); m.(*App).modal != nil {
		t.Fatal("esc on the topics did not close the guide")
	}
}

// The ? key list says how to reach the guide.
func TestHelpNamesTheManual(t *testing.T) {
	m := press(t, newTestApp(t, tuitest.NewFake(), 80, 24, "unicode"), "?")
	if fr := tuitest.Frame(m); !strings.Contains(fr, "F for the full guide") {
		t.Fatalf("frame:\n%s", fr)
	}
}

func TestHelpCommandOpensATopic(t *testing.T) {
	m := typed(t, press(t, newTestApp(t, tuitest.NewFake(), 80, 24, "unicode"), ":"), "help jobs")
	m = press(t, m, "enter")
	g, ok := m.(*App).modal.(*manualModal)
	if !ok || !g.open || g.topics[g.cur].Name != "jobs" {
		t.Fatalf("modal %#v", m.(*App).modal)
	}
	tuitest.Golden(t, "manual_jobs_80x24", tuitest.Frame(m))

	m = typed(t, press(t, newTestApp(t, tuitest.NewFake(), 80, 24, "unicode"), ":"), "help nope")
	m = press(t, m, "enter")
	if fr := tuitest.Frame(m); !strings.Contains(fr, `no guide topic "nope"`) || !strings.Contains(fr, "security") {
		t.Fatalf("frame:\n%s", fr)
	}
}

// Every page, scrolled from top to bottom at 80x24 under both glyph sets,
// fills the frame without a line wider than it, and the renderer never
// leaves fit a line to cut. Under ascii every frame is pure ASCII.
func TestManualPagesFit80x24(t *testing.T) {
	for _, glyphs := range []string{"unicode", "ascii"} {
		for i, tp := range manualSource.Topics() {
			a := newTestApp(t, tuitest.NewFake(), 80, 24, glyphs)
			for _, l := range renderMarkdown(tp.Body, 78, a.th, a.gl) {
				if w := ansi.StringWidth(l); w > 78 {
					t.Errorf("%s %s: a %d-cell line: %q", glyphs, tp.Name, w, ansi.Strip(l))
				}
			}
			m := typed(t, press(t, a, ":"), "help "+tp.Name)
			m = press(t, m, "enter")
			for n := 0; ; n++ {
				fr := tuitest.Frame(m)
				lines := strings.Split(fr, "\n")
				if len(lines) != 24 {
					t.Fatalf("%s %s: %d lines", glyphs, tp.Name, len(lines))
				}
				for _, l := range lines {
					if ansi.StringWidth(l) > 80 {
						t.Errorf("%s %s: line wider than 80: %q", glyphs, tp.Name, l)
					}
				}
				if glyphs == "ascii" {
					requireASCII(t, fr)
				}
				g := m.(*App).modal.(*manualModal)
				if g.cur != i {
					t.Fatalf("%s: on page %d", tp.Name, g.cur)
				}
				if g.offset+g.room >= len(g.lines) {
					break
				}
				if n > 50 {
					t.Fatalf("%s: space did not reach the end (offset %d of %d)", tp.Name, g.offset, len(g.lines))
				}
				m = press(t, m, "space")
			}
		}
	}
}

// Left and right turn the pages; G and g go to the end and back.
func TestManualPageKeys(t *testing.T) {
	m := typed(t, press(t, newTestApp(t, tuitest.NewFake(), 80, 24, "unicode"), ":"), "help install")
	m = press(t, m, "enter")
	tuitest.Frame(m) // a page is laid out when drawn
	m = press(t, m, "G")
	tuitest.Frame(m)
	g := m.(*App).modal.(*manualModal)
	if g.offset == 0 || g.offset+g.room != len(g.lines) {
		t.Fatalf("G: offset %d room %d lines %d", g.offset, g.room, len(g.lines))
	}
	if !strings.Contains(tuitest.Frame(m), "100%") {
		t.Fatalf("the footer does not say the end:\n%s", tuitest.Frame(m))
	}
	m = press(t, m, "g", "l")
	if g.offset != 0 || g.topics[g.cur].Name != "getting-started" {
		t.Fatalf("g, l: offset %d topic %s", g.offset, g.topics[g.cur].Name)
	}
	m = press(t, m, "h", "h")
	if g.topics[g.cur].Name != "install" {
		t.Fatalf("h past the first page: %s", g.topics[g.cur].Name)
	}
	if press(t, m, "q").(*App).modal != nil {
		t.Fatal("q did not close the guide")
	}
}

// The renderer draws what the guide uses: headings, lists, code, emphasis
// and links, and a link to another page names the command that opens it.
func TestRenderMarkdown(t *testing.T) {
	a := newTestApp(t, tuitest.NewFake(), 80, 24, "unicode")
	src := "# Title\n\nSome *soft* and **hard** text with `code`,\na [site](https://x.example) and [jobs](jobs.md).\n\n" +
		"- one\n  continued\n- two\n\n1. first\n\n```\nverbatim  *not styled*\n```\n"
	got := strings.Join(renderMarkdown(src, 60, a.th, a.gl), "\n")
	want := "TITLE\n\nSome soft and hard text with code, a site\n(https://x.example) and jobs (:help jobs).\n\n" +
		" • one continued\n • two\n\n 1. first\n\n    verbatim  *not styled*"
	if s := ansi.Strip(got); s != want {
		t.Fatalf("got\n%s\nwant\n%s", s, want)
	}
	// snake_case and a lone star stay as they are.
	if s := ansi.Strip(inline("a_b_c 2 * 3", a.th)); s != "a_b_c 2 * 3" {
		t.Fatalf("inline %q", s)
	}
}

func TestASCIITextIsASCII(t *testing.T) {
	s := asciiText("a – b — “c” … → • é 世")
	if s != `a - b - "c" ... -> * ? ?` {
		t.Fatalf("%q", s)
	}
	for _, r := range s {
		if r >= utf8.RuneSelf {
			t.Fatalf("%q", s)
		}
	}
}

// boundKeys is every key some binding matches, on any screen or modal.
func boundKeys() map[string]bool {
	out := map[string]bool{}
	for _, d := range Keymap() {
		for _, k := range d.Keys {
			out[k] = true
		}
	}
	return out
}

// Keymap reaches every screen's keys: one from each keymap, and Settings'.
func TestKeymapHasEveryScreen(t *testing.T) {
	where := map[string]bool{}
	for _, d := range Keymap() {
		where[d.Where] = true
		if d.Label == "" || d.Help == "" || len(d.Keys) == 0 {
			t.Errorf("%+v", d)
		}
	}
	if len(where) != len(keymaps)+1 {
		t.Fatalf("places %v", where)
	}
}

// expand is a command form as Commands writes it, made into the lines that
// type it: NAME and TOPIC filled in, each a|b word taken both ways, and an
// [optional] part both left out and given.
func expand(form string) []string {
	lines := []string{""}
	for _, w := range strings.Fields(form) {
		var next []string
		inner, optional := strings.CutPrefix(w, "[")
		w = strings.TrimSuffix(inner, "]")
		for _, l := range lines {
			if optional {
				next = append(next, l)
			}
			for _, alt := range strings.Split(w, "|") {
				alt = strings.NewReplacer("NAME", "box-a", "TOPIC", "jobs").Replace(alt)
				next = append(next, strings.TrimSpace(l+" "+alt))
			}
		}
		lines = next
	}
	return lines
}

// runsInPalette: the line goes through the real palette and does not come
// back as an error.
func runsInPalette(t *testing.T, line string) {
	t.Helper()
	a := newTestApp(t, tuitest.NewFake(), 80, 24, "unicode")
	stubShell(a)
	m := press(t, typed(t, press(t, a, ":"), line), "enter")
	if p, ok := m.(*App).modal.(*paletteModal); ok {
		t.Errorf("%q: %s", line, p.err)
	}
}

// Every form Commands lists runs, so a generated list of commands is true.
func TestEveryListedCommandRuns(t *testing.T) {
	for _, form := range Commands() {
		for _, line := range expand(form) {
			runsInPalette(t, line)
		}
	}
	if got := expand("start|stop exec NAME [TOPIC]"); len(got) != 4 || got[0] != "start exec box-a" || got[3] != "stop exec box-a jobs" {
		t.Fatalf("expand: %q", got)
	}
}

// namedKeys are the key names the guide may write in backticks.
var namedKeys = regexp.MustCompile(`^(enter|esc|tab|space|backspace|pgup|pgdown|home|end|up|down|left|right|ctrl\+[a-z])$`)

// guideTokens splits the TUI page into the backticked keys it names and the
// commands it names: everything backticked under "## Commands", and
// anywhere a backticked ":word" (":help [TOPIC]"). A lone character or a
// key name is a key wherever it is.
func guideTokens(t *testing.T) (keys, commands []string) {
	t.Helper()
	var body string
	for _, tp := range manualSource.Topics() {
		if tp.Name == "tui" {
			body = tp.Body
		}
	}
	if body == "" {
		t.Fatal("no tui topic")
	}
	tick := regexp.MustCompile("`([^`]+)`")
	inCommands := false
	for _, l := range strings.Split(body, "\n") {
		if strings.HasPrefix(l, "#") {
			inCommands = strings.TrimSpace(strings.TrimLeft(l, "#")) == "Commands"
			continue
		}
		for _, m := range tick.FindAllStringSubmatch(l, -1) {
			tok := m[1]
			switch {
			case utf8.RuneCountInString(tok) == 1 || namedKeys.MatchString(tok):
				keys = append(keys, tok)
			case inCommands:
				commands = append(commands, tok)
			case strings.HasPrefix(tok, ":"):
				commands = append(commands, tok[1:])
			}
		}
	}
	if len(keys) < 10 || len(commands) < 10 {
		t.Fatalf("found too few: keys %q commands %q", keys, commands)
	}
	return keys, commands
}

// The TUI page names only keys that are bound somewhere.
func TestGuideKeysAreBound(t *testing.T) {
	bound := boundKeys()
	keys, _ := guideTokens(t)
	for _, k := range keys {
		if !bound[k] {
			t.Errorf("the guide names %q, which no binding matches", k)
		}
	}
}

// The TUI page names only commands the palette runs.
func TestGuideCommandsRun(t *testing.T) {
	_, commands := guideTokens(t)
	for _, c := range commands {
		for _, line := range expand(c) {
			runsInPalette(t, line)
		}
	}
}

// The guide's command is in the palette's own list of commands.
func TestPaletteListsHelpTopic(t *testing.T) {
	m := press(t, newTestApp(t, tuitest.NewFake(), 80, 24, "unicode"), ":")
	if fr := tuitest.Frame(m); !strings.Contains(fr, "help [TOPIC]") || !strings.Contains(fr, "quit") {
		t.Fatalf("frame:\n%s", fr)
	}
}
