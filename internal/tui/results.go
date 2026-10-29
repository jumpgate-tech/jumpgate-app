package tui

import (
	"context"
	"encoding/json"
	"strings"

	tea "charm.land/bubbletea/v2"
)

// The plainest result for a screen's one-off calls (gateways, VPN): the
// answer is drawn under the list as key: value lines (jsonLines). Every
// string is sanitized where it is drawn, whatever was stored.

// resultPane is the last answer a screen showed.
type resultPane struct {
	head  string
	lines []string
	bad   bool
	has   bool
}

// callMsg is one Do's answer for the screen to.
type callMsg struct {
	to     screenID
	head   string
	v      json.RawMessage
	err    error
	reload bool // the call changed something: read the list again
}

// call sends one request for screen to and shows its answer under head.
func (a *App) call(to screenID, head, method, path string, in any, reload bool) tea.Cmd {
	return a.do(head, func(ctx context.Context, be Backend) tea.Msg {
		var v json.RawMessage
		err := be.Do(ctx, method, path, in, &v)
		return callMsg{to: to, head: head, v: v, err: err, reload: reload}
	})
}

// take stores msg's answer; the flash says how it went.
func (r *resultPane) take(a *App, msg callMsg) {
	head := sanitizeLine(msg.head)
	if msg.err != nil {
		*r = resultPane{head: head + ": FAIL", lines: []string{a.errText(msg.err)}, bad: true, has: true}
		a.flash = head + " failed"
		return
	}
	lines := []string{"ok"}
	if len(msg.v) > 0 && string(msg.v) != "null" {
		lines = jsonLines(msg.v)
	}
	*r = resultPane{head: head, lines: lines, has: true}
	a.flash = head + ": ok"
}

// pending shows head while the answer is on its way.
func (r *resultPane) pending(a *App, head string) {
	*r = resultPane{head: sanitizeLine(head) + a.gl.Ellipsis, has: true}
}

// view is the pane's lines, sanitized here: the sink.
func (r *resultPane) view(a *App) []string {
	if !r.has {
		return nil
	}
	safe := sanitizeLine(r.head)
	head := a.th.Title.Render(safe)
	if r.bad {
		head = a.th.Bad.Render(safe)
	}
	out := []string{"", " " + head}
	for _, l := range r.lines {
		out = append(out, "   "+sanitizeLine(l))
	}
	return out
}

// onceModal shows a one-time secret the server returned (a customer API
// key, a VPN peer's config). It is never put in the flash or the activity
// log; closing the modal clears the bytes and drops them.
type onceModal struct {
	title, detail string
	secret        []byte
}

func newOnce(title, detail string, secret []byte) *onceModal {
	return &onceModal{title: sanitizeLine(title), detail: sanitize(detail), secret: secret}
}

func (m *onceModal) key(a *App, k tea.KeyPressMsg) (bool, tea.Cmd) {
	switch k.String() {
	case "esc", "enter":
		clear(m.secret)
		m.secret = nil
		a.flash = "closed; it is not shown again"
		return true, nil
	}
	return false, nil
}

func (m *onceModal) view(a *App, w, _ int) string {
	s := a.th.Title.Render(" "+m.title) + "\n\n"
	if m.detail != "" {
		s += " " + m.detail + "\n\n"
	}
	for _, l := range strings.Split(sanitize(string(m.secret)), "\n") {
		s += " " + l + "\n"
	}
	return s + "\n " + a.th.Warn.Render("shown once: copy it now") + "\n " + a.th.Dim.Render("enter or esc closes and forgets it")
}

// jsonStringBody is {<pre>"field":"<v as a JSON string>"} in one buffer the
// caller clears: v (a WireGuard config) never becomes a Go string. pre is
// plain JSON members ending in a comma, or "".
func jsonStringBody(pre, field string, v []byte) []byte {
	const hex = "0123456789abcdef"
	b := make([]byte, 0, len(pre)+len(field)+8+len(v)*6)
	b = append(b, '{')
	b = append(b, pre...)
	b = append(b, '"')
	b = append(b, field...)
	b = append(b, `":"`...)
	for _, c := range v {
		switch {
		case c == '"' || c == '\\':
			b = append(b, '\\', c)
		case c < 0x20:
			b = append(b, '\\', 'u', '0', '0', hex[c>>4], hex[c&15])
		default:
			b = append(b, c)
		}
	}
	return append(b, '"', '}')
}

// secretField is the one string field of a JSON answer as bytes. It reads
// the raw answer itself, so the caller can clear both after use.
func secretField(raw json.RawMessage, field string) []byte {
	var m map[string]json.RawMessage
	if json.Unmarshal(raw, &m) != nil {
		return nil
	}
	q := m[field]
	var s string
	if json.Unmarshal(q, &s) != nil {
		return nil
	}
	clear(q)
	return []byte(s)
}
