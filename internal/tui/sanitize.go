package tui

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// sanitize makes text from outside the TUI safe to draw: target ids and
// names, hostnames, server versions and responses, error messages, log lines
// and AI output. A box, its logs or a hostile server could otherwise send
// escape sequences that move the cursor, overwrite the screen, retitle the
// window, write the clipboard (OSC 52) or plant a hyperlink (OSC 8).
//
// It is the one choke point: every external string passes through it before
// it is placed in a frame, and the TUI's own styling is applied around the
// result, so Lip Gloss and Bubble Tea sequences stay intact. The server keeps
// raw data; cleaning it is the front end's job.
//
// Removed: C0 controls except \t and \n (so \r cannot overwrite a line),
// DEL, C1 controls (U+0080–U+009F, as runes or as raw bytes), ESC sequences
// (CSI, OSC, DCS, SOS, PM, APC and two-byte escapes, with their bodies; an
// unterminated string sequence takes the rest of the text with it), and the
// bidi overrides, isolates and marks (U+202A–U+202E, U+2066–U+2069, LRM,
// RLM, ALM) that make text read differently from what it is. Other invalid UTF-8 becomes U+FFFD.
func sanitize(s string) string {
	if isClean(s) {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && size == 1 {
			// A lone byte 0x80–0x9F is an 8-bit C1 control to a terminal
			// that reads them; anything else invalid is shown as U+FFFD.
			if c := s[i]; c >= 0x80 && c <= 0x9f {
				i = skipC1(s, i+1, rune(c))
				continue
			}
			b.WriteRune(utf8.RuneError)
			i++
			continue
		}
		switch {
		case r == 0x1b:
			i = skipEscape(s, i+1)
		case r >= 0x80 && r <= 0x9f:
			i = skipC1(s, i+size, r)
		case r == '\t' || r == '\n':
			b.WriteRune(r)
			i += size
		case r < 0x20 || r == 0x7f, isBidiControl(r):
			i += size
		default:
			b.WriteString(s[i : i+size])
			i += size
		}
	}
	return b.String()
}

// isClean is the fast path: printable ASCII, \t and \n only.
func isClean(s string) bool {
	for i := 0; i < len(s); i++ {
		if c := s[i]; c >= 0x7f || (c < 0x20 && c != '\t' && c != '\n') {
			return false
		}
	}
	return true
}

// isBidiControl is every invisible format character (category Cf): the bidi
// overrides, isolates and marks, and also zero-width space, joiners, word
// joiner, BOM and soft hyphen, which hide text from a reader and from
// matching. Removing them also removes ZWJ from emoji sequences.
func isBidiControl(r rune) bool { return unicode.Is(unicode.Cf, r) }

// skipEscape skips the sequence after an ESC at s[i-1] and returns the index
// just past it.
func skipEscape(s string, i int) int {
	if i >= len(s) {
		return i
	}
	switch s[i] {
	case '[':
		return skipCSI(s, i+1)
	case ']', 'P', 'X', '^', '_': // OSC, DCS, SOS, PM, APC
		return skipString(s, i+1)
	}
	// A two-byte escape, possibly with intermediates (ESC ( B).
	for i < len(s) && s[i] >= 0x20 && s[i] <= 0x2f {
		i++
	}
	if i < len(s) && s[i] >= 0x30 && s[i] <= 0x7e {
		i++
	}
	return i
}

// skipC1 skips a C1 control r whose encoding ends at s[i-1], with the body
// of the sequence it introduces.
func skipC1(s string, i int, r rune) int {
	switch r {
	case 0x9b: // CSI
		return skipCSI(s, i)
	case 0x9d, 0x90, 0x98, 0x9e, 0x9f: // OSC, DCS, SOS, PM, APC
		return skipString(s, i)
	}
	return i
}

// skipCSI skips parameter and intermediate bytes and the final byte.
func skipCSI(s string, i int) int {
	for i < len(s) && s[i] >= 0x20 && s[i] <= 0x3f {
		i++
	}
	if i < len(s) && s[i] >= 0x40 && s[i] <= 0x7e {
		i++
	}
	return i
}

// skipString skips a string sequence's body up to and including its end:
// BEL, ESC \, or the rune ST (U+009C). The body is read rune by rune, so the
// byte 0x9C inside another character (U+011C is C4 9C) is not mistaken for
// ST. Without an end it runs to the end of s, so a cut-off sequence cannot
// leak its payload.
func skipString(s string, i int) int {
	for i < len(s) {
		r, size := utf8.DecodeRuneInString(s[i:])
		switch {
		case r == 0x07, r == 0x9c:
			return i + size
		case r == 0x1b && i+1 < len(s) && s[i+1] == '\\':
			return i + 2
		}
		i += size
	}
	return i
}

// lineBreaks are the characters sanitize keeps that would break a one-line
// field across rows.
var lineBreaks = strings.NewReplacer("\n", " ", "\t", " ", "\r", " ")

// sanitizeLine is sanitize for a one-line field (the status bar, a flash, the
// skew banner, a box name): line breaks and tabs become spaces, so hostile
// text cannot add rows to the frame.
func sanitizeLine(s string) string { return lineBreaks.Replace(sanitize(s)) }
