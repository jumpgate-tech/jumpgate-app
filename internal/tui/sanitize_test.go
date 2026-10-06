package tui

import (
	"errors"
	"strings"
	"testing"
	"time"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/apiclient"
	"github.com/valve-tech/jumpgate/internal/tui/tuitest"
)

func TestSanitize(t *testing.T) {
	for name, c := range map[string]struct{ in, want string }{
		"plain":                {"box-a synced", "box-a synced"},
		"tab and newline kept": {"a\tb\nc", "a\tb\nc"},
		"cursor moves":         {"a\x1b[2Ab\x1b[10;5Hc\x1b[2Jd", "abcd"},
		"sgr colour":           {"\x1b[31mred\x1b[0m", "red"},
		"window title (BEL)":   {"x\x1b]0;title\x07y", "xy"},
		"window title (ST)":    {"x\x1b]2;title\x1b\\y", "xy"},
		"clipboard write":      {"x\x1b]52;c;cm0gLXJmIH4=\x07y", "xy"},
		"osc 8 link":           {"\x1b]8;;https://evil.example/\x1b\\click\x1b]8;;\x1b\\", "click"},
		"c1 csi raw byte":      {"a\x9b2Jb", "ab"},
		"c1 csi rune":          {"a\u009b2Jb", "ab"},
		"c1 osc rune":          {"a\u009d0;t\u009cb", "ab"},
		"other c1":             {"a\u0085b\u0080c", "abc"},
		"dcs":                  {"a\x1bPq#0;2;0;0;0\x1b\\b", "ab"},
		"apc pm sos":           {"a\x1b_x\x1b\\b\x1b^y\x1b\\c\x1bXz\x1b\\d", "abcd"},
		"two-byte escape":      {"a\x1bcb\x1b7c\x1b(Bd", "abcd"},
		"unterminated osc":     {"a\x1b]0;forever", "a"},
		"trailing esc":         {"a\x1b", "a"},
		"bare cr":              {"safe\rEVIL", "safeEVIL"},
		"bel bs del nul":       {"a\x07b\x08c\x7fd\x00e", "abcde"},
		"bidi overrides":       {"a\u202eb\u202ac\u202bd\u202ce\u202df", "abcdef"},
		"bidi isolates":        {"a\u2066b\u2067c\u2068d\u2069e", "abcde"},
		"invalid utf-8":        {"a\xffb\xc3", "a\ufffdb\ufffd"},
		"unicode kept":         {"⚑ █ é 日本", "⚑ █ é 日本"},
	} {
		if got := sanitize(c.in); got != c.want {
			t.Errorf("%s: sanitize(%q) = %q, want %q", name, c.in, got, c.want)
		}
	}
}

// hostile carries every attack in one string.
const hostile = "\x1b[2J\x1b[H\x1b]0;pwned\x07\x1b]52;c;cm0gLXJmIH4=\x07\x1b]8;;https://evil.example/\x1b\\link\x1b]8;;\x1b\\\x9b2J\u202e\rX"

// plainTheme renders without any escape, so every ESC left in a frame came
// from data rather than the TUI's own styling.
func plainTheme() Theme {
	s := lipgloss.NewStyle
	return Theme{OK: s(), Warn: s(), Bad: s(), Info: s(), Dim: s(), Title: s(), Sel: s(), Key: s()}
}

func requireClean(t *testing.T, what, content string) {
	t.Helper()
	for _, bad := range []string{"\x1b", "\x9b", "\u009b", "\r", "\x07", "\u202e"} {
		if strings.Contains(content, bad) {
			t.Fatalf("%s: frame contains %q:\n%q", what, bad, content)
		}
	}
	if !strings.Contains(content, "link") {
		t.Fatalf("%s: the visible text was lost:\n%q", what, content)
	}
}

// External text (target names, server versions, errors, log lines) reaches
// the screen only through sanitize: none of it can move the cursor, retitle
// the window, write the clipboard or plant a link.
func TestExternalTextCannotEscape(t *testing.T) {
	clean := newTestApp(t, tuitest.NewFake(), 80, 24, "unicode")
	clean.th = plainTheme()
	if c := clean.View().Content; strings.Contains(c, "\x1b") {
		t.Fatalf("plain theme still styles: %q", c)
	}

	f := tuitest.NewFake()
	f.ServerVersion = "v1" + hostile
	a := newTestApp(t, f, 80, 24, "unicode")
	a.th = plainTheme()

	box := "box" + hostile
	a.sel = box
	m, _ := tuitest.Send(a, fleetMsg{u: apiclient.Update[api.Fleet]{
		State: apiclient.Retrying, Err: errors.New("dial: " + hostile),
		Value: api.Fleet{At: testNow.Add(-time.Second), Rows: []api.FleetRow{{ID: box, Link: api.LinkAgent, Reachable: true}}}, Has: true,
	}})
	requireClean(t, "skew, target and stream error", m.View().Content)

	m, _ = tuitest.Send(m, errMsg{what: "stop " + box, err: &api.Error{Message: "failed: " + hostile, Hint: "hint " + hostile}})
	requireClean(t, "api error", m.View().Content)

	m, _ = tuitest.Send(m, flashMsg("log: "+hostile))
	requireClean(t, "flash (log line)", m.View().Content)

	a = m.(*App)
	a.modal = newConfirm("Remove "+box, "last log line: "+hostile, box, func() tea.Cmd { return nil })
	// The field's own prompt styling is the TUI's, not data: strip it too.
	cm := a.modal.(*confirmModal)
	cm.in.Blur()
	cm.in.SetStyles(textinput.Styles{})
	requireClean(t, "confirmation", a.View().Content)
}
