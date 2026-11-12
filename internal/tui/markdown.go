package tui

import (
	"strings"
	"unicode"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// renderMarkdown draws the guide's markdown as lines at most w cells wide.
// It knows only what the guide uses (docs/guide): headings, paragraphs, flat
// lists, fenced code, inline code, bold, italic and links. Anything else
// shows as the text it is. Style is never the only signal: headings stand
// apart by spacing, code by its indent, so the page reads under NO_COLOR.
//
// The source is embedded and ours, but it goes through sanitize like any
// other text, and under the ascii glyphs through asciiText, so a stray
// character in a guide file cannot break the frame.
func renderMarkdown(src string, w int, th Theme, gl Glyphs) []string {
	src = sanitize(src)
	if gl.ASCII {
		src = asciiText(src)
	}
	w = max(w, 20)
	var out []string
	blank := func() {
		if len(out) > 0 && out[len(out)-1] != "" {
			out = append(out, "")
		}
	}
	wrap := func(text string, first, rest string) {
		for i, l := range strings.Split(ansi.Wrap(text, w-lipgloss.Width(first), ""), "\n") {
			if i == 0 {
				out = append(out, first+l)
			} else {
				out = append(out, rest+l)
			}
		}
	}
	var para []string // the paragraph or list item being gathered
	var bullet string // "" for a paragraph, else the item's marker
	flush := func() {
		if len(para) == 0 {
			return
		}
		text := inline(strings.Join(para, " "), th)
		if bullet == "" {
			wrap(text, "", "")
		} else {
			wrap(text, " "+bullet+" ", strings.Repeat(" ", lipgloss.Width(bullet)+2))
		}
		para, bullet = nil, ""
	}
	lines := strings.Split(strings.ReplaceAll(src, "\t", "    "), "\n")
	for i := 0; i < len(lines); i++ {
		l := strings.TrimRight(lines[i], " ")
		trim := strings.TrimSpace(l)
		switch {
		case strings.HasPrefix(trim, "```"):
			flush()
			blank()
			for i++; i < len(lines) && !strings.HasPrefix(strings.TrimSpace(lines[i]), "```"); i++ {
				code := strings.TrimRight(lines[i], " ")
				for _, c := range strings.Split(ansi.Hardwrap(code, w-4, false), "\n") {
					out = append(out, "    "+th.Info.Render(c))
				}
			}
			blank()
		case trim == "":
			flush()
			blank()
		case strings.HasPrefix(trim, "#"):
			flush()
			level := len(trim) - len(strings.TrimLeft(trim, "#"))
			text := strings.TrimSpace(trim[level:])
			if level == 1 {
				text = strings.ToUpper(text)
			}
			blank()
			wrap(th.Title.Render(text), "", "")
			out = append(out, "")
		case isItem(trim):
			flush()
			marker, text, _ := strings.Cut(trim, " ")
			bullet = gl.Bullet
			if marker != "-" && marker != "*" {
				bullet = marker // a numbered item keeps its number
			}
			para = []string{text}
		default:
			para = append(para, trim) // a paragraph line, or an item's continuation
		}
	}
	flush()
	for len(out) > 0 && out[len(out)-1] == "" {
		out = out[:len(out)-1]
	}
	return out
}

// isItem: a list item, "- x", "* x" or "1. x".
func isItem(s string) bool {
	if strings.HasPrefix(s, "- ") || strings.HasPrefix(s, "* ") {
		return true
	}
	n := strings.IndexFunc(s, func(r rune) bool { return !unicode.IsDigit(r) })
	return n > 0 && strings.HasPrefix(s[n:], ". ")
}

// inline styles `code`, **bold**, *italic* or _italic_, and [text](url). A
// link to another guide page names the command that opens it, since a
// terminal cannot follow the link; any other shows its address.
func inline(s string, th Theme) string {
	italic := lipgloss.NewStyle().Italic(true)
	var b strings.Builder
	for i := 0; i < len(s); {
		rest := s[i:]
		// span is the text between open and end at the start of rest, and
		// the bytes the whole span takes.
		span := func(open, end string) (string, int) {
			if !strings.HasPrefix(rest, open) {
				return "", 0
			}
			j := strings.Index(rest[len(open):], end)
			if j <= 0 {
				return "", 0
			}
			return rest[len(open) : len(open)+j], len(open) + j + len(end)
		}
		wordStart := i == 0 || !isWord(rune(s[i-1]))
		if t, n := span("`", "`"); n > 0 {
			b.WriteString(th.Info.Render(t))
			i += n
		} else if t, n := span("**", "**"); n > 0 {
			b.WriteString(th.Title.Render(t))
			i += n
		} else if t, n := span("*", "*"); n > 0 && wordStart {
			b.WriteString(italic.Render(t))
			i += n
		} else if t, n := span("_", "_"); n > 0 && wordStart {
			b.WriteString(italic.Render(t))
			i += n
		} else if text, url, ok := link(rest); ok {
			i += len(text) + len(url) + 4
			b.WriteString(text)
			switch {
			case !strings.Contains(url, ":") && strings.HasSuffix(url, ".md"):
				b.WriteString(th.Dim.Render(" (:help " + strings.TrimSuffix(url, ".md") + ")"))
			case url != text:
				b.WriteString(th.Dim.Render(" (" + url + ")"))
			}
		} else {
			b.WriteByte(s[i])
			i++
		}
	}
	return b.String()
}

// link parses "[text](url)" at the start of s.
func link(s string) (text, url string, ok bool) {
	if !strings.HasPrefix(s, "[") {
		return "", "", false
	}
	text, after, ok := strings.Cut(s[1:], "](")
	if !ok || strings.Contains(text, "]") {
		return "", "", false
	}
	url, _, ok = strings.Cut(after, ")")
	return text, url, ok && url != ""
}

func isWord(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) }

// asciiPunct spells the guide's typographic characters in ASCII.
var asciiPunct = strings.NewReplacer("‘", "'", "’", "'", "“", `"`, "”", `"`, "–", "-", "—", "-", "…", "...",
	"→", "->", "←", "<-", "↑", "up", "↓", "down", "•", "*", "×", "x", " ", " ")

// asciiText is s with every rune printable ASCII: the known punctuation
// spelled out, anything else a "?".
func asciiText(s string) string {
	s = asciiPunct.Replace(s)
	return strings.Map(func(r rune) rune {
		if r == '\n' || (r >= 0x20 && r <= 0x7e) {
			return r
		}
		return '?'
	}, s)
}
