package termsafe

import (
	"strings"
	"testing"
)

func TestTextAndLine(t *testing.T) {
	if got := Text("a\x1b]52;c;QQ==\x07b\x1b[2Jc\r\x00\td\n"); got != "abc\td\n" {
		t.Errorf("Text = %q", got)
	}
	if got := Line("a\tb\nc\r"); got != "a b c" {
		t.Errorf("Line = %q", got)
	}
}

// Every way a terminal can be told to do something other than print: each
// case is a sequence a hostile box or server could send. The want column is
// what must reach the screen; anything more is an injection.
func TestTextRemovesEverySequenceATerminalWouldAct(t *testing.T) {
	cases := []struct{ name, in, want string }{
		{"OSC ended by BEL", "a\x1b]0;evil title\x07b", "ab"},
		{"OSC ended by ESC backslash", "a\x1b]8;;https://evil\x1b\\link\x1b]8;;\x1b\\b", "alinkb"},
		{"OSC ended by the ST rune", "a\x1b]0;t\u009cb", "ab"},
		{"OSC 52 clipboard write without an end swallows the rest", "a\x1b]52;c;c2VjcmV0", "a"},
		{"a 0x9C byte inside a multibyte rune does not end an OSC", "\x1b]0;\u011cpayload\x07b", "b"},
		{"CSI with parameters", "a\x1b[1;31mb\x1b[2Jc", "abc"},
		{"CSI cut off at the end", "a\x1b[12;3", "a"},
		{"ESC alone at the end", "a\x1b", "a"},
		{"two-byte escape with an intermediate", "a\x1b(Bb", "ab"},
		{"two-byte escape without one", "a\x1b7b\x1bc", "ab"},
		{"DCS", "a\x1bPq#0;2;0;0;0\x1b\\b", "ab"},
		{"SOS", "a\x1bXhidden\x1b\\b", "ab"},
		{"PM", "a\x1b^hidden\x07b", "ab"},
		{"APC", "a\x1b_hidden\x1b\\b", "ab"},
		{"C1 CSI as a rune", "a\u009b2Jb", "ab"},
		{"C1 OSC as a rune", "a\u009d0;title\u0007b", "ab"},
		{"C1 DCS as a rune", "a\u0090payload\u009cb", "ab"},
		{"C1 SOS, PM and APC as runes", "a\u0098x\u009cb\u009ey\u009cc\u009fz\u009cd", "abcd"},
		{"a C1 that opens nothing is dropped alone", "a\u0085b\u0084c", "abc"},
		{"C1 CSI as a raw byte", "a\x9b2Jb", "ab"},
		{"C1 OSC as a raw byte", "a\x9d0;title\x07b", "ab"},
		{"a raw C1 byte that opens nothing", "a\x85b", "ab"},
		{"bidi overrides, isolates and marks", "a\u202eb\u2066c\u2069d\u200fe\u200ef\u061cg", "abcdefg"},
		{"zero-width and BOM", "a\u200bb\ufeffc\u00add", "abcd"},
		{"C0 controls and DEL, keeping tab and newline", "a\rb\x00c\x08d\x7fe\tf\ng", "abcde\tf\ng"},
		{"invalid UTF-8 outside C1 becomes U+FFFD", "a\xffb", "a\ufffdb"},
		{"printable non-ASCII is kept", "héllo 世界 ✓", "héllo 世界 ✓"},
	}
	for _, c := range cases {
		if got := Text(c.in); got != c.want {
			t.Errorf("%s: Text(%q) = %q, want %q", c.name, c.in, got, c.want)
		}
	}
}

// Clean ASCII is returned as is, the same string, so the fast path cannot
// alter what it was handed.
func TestTextReturnsCleanASCIIUnchanged(t *testing.T) {
	for _, s := range []string{"", "plain", "tab\tand\nnewline", "~!@#$%^&*()"} {
		if got := Text(s); got != s {
			t.Errorf("Text(%q) = %q", s, got)
		}
	}
}

// Line must not let a hostile string add rows, even one that hides its
// newline behind an escape that Text removes.
func TestLineKeepsHostileTextOnOneRow(t *testing.T) {
	if got := Line("a\x1b[1Ab\nc\rd"); got != "ab cd" {
		t.Errorf("Line = %q", got)
	}
	for _, s := range []string{"x\ny", "x\ty", "x\x1b]0;t\x07\ny"} {
		if got := Line(s); strings.ContainsAny(got, "\n\t\r") {
			t.Errorf("Line(%q) = %q, keeps a line break", s, got)
		}
	}
}
