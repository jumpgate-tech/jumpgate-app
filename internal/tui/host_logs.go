package tui

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/apiclient"
)

const (
	logBacklog = 500  // lines asked for on every (re)connect
	logKeep    = 2000 // lines kept in the tab
	// An Explain request is budgeted by its JSON-encoded size: "<", ">" and
	// "&" encode to 6 bytes and quote and backslash to 2. The server refuses
	// more than 5000 lines or 1 MB (413 too_many_lines, too_large); this
	// sends at most explainMax of the newest lines, each cut to explainLineMax
	// bytes, with the encoded total under explainBodyMax.
	explainMax     = 400
	explainLineMax = 1500
	explainBodyMax = 512 << 10
	// logLineMax bounds what is kept per line; the screen cuts to the width.
	logLineMax = 4096
	// explainTextMax bounds the answer read from the provider.
	explainTextMax = 16 << 10
	explainSentMax = 8 // sent lines listed in the modal
)

var logKeys = struct{ Level, Follow, Explain key.Binding }{
	Level:   key.NewBinding(key.WithKeys("!"), key.WithHelp("!", "level: all, warn+, error+")),
	Follow:  key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "follow the newest line")),
	Explain: key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "explain the errors (AI)")),
}

var levelNames = []string{"all levels", "warn and above", "error and above"}

type (
	logsMsg struct {
		id  string
		gen uint64
		u   apiclient.Update[[]api.LogHit]
		ch  <-chan apiclient.Update[[]api.LogHit]
	}
	// disclosureMsg is the provider disclosure, fetched before anything is sent.
	disclosureMsg struct {
		id         string
		gen        uint64
		disclosure string
		provider   string
		err        error
	}
	explainMsg struct {
		id  string
		gen uint64
		e   api.Explain
		err error
	}
)

func newLogFilter() textinput.Model {
	in := textinput.New()
	in.Prompt = "/"
	in.CharLimit = 64
	// ctrl+v would read the system clipboard; a terminal paste still arrives
	// as a PasteMsg, which the host screen sanitizes.
	in.KeyMap.Paste.SetEnabled(false)
	return in
}

// watchLogs waits for the next update; it ends when the box is left.
func (h *hostScreen) watchLogs(ch <-chan apiclient.Update[[]api.LogHit]) tea.Cmd {
	id, gen, ctx := h.id, h.gen, h.ctx
	return func() tea.Msg {
		select {
		case <-ctx.Done():
			return nil
		case u, ok := <-ch:
			if !ok {
				return nil
			}
			return logsMsg{id: id, gen: gen, u: u, ch: ch}
		}
	}
}

// cutBytes cuts s to at most n bytes on a rune boundary.
func cutBytes(s string, n int) string {
	if len(s) <= n {
		return s
	}
	for n > 0 && !utf8.RuneStart(s[n]) {
		n--
	}
	return s[:n]
}

// cleanHit is a line from the box made safe to keep and draw: logs are
// attacker-influenced. They are shown unmasked on purpose; the operator is
// reading their own box's logs.
func cleanHit(l api.LogHit) api.LogHit {
	l.Line = sanitizeLine(cutBytes(l.Line, logLineMax))
	l.Unit = sanitizeLine(cutBytes(l.Unit, 128))
	l.Severity = sanitizeLine(cutBytes(l.Severity, 16))
	l.Explain, l.LearnURL, l.Signature = "", "", ""
	return l
}

// applyLogs takes one stream update: a reset replaces the lines (every
// reconnect starts with one, so nothing is duplicated), a value appends.
func (h *hostScreen) applyLogs(u apiclient.Update[[]api.LogHit]) {
	h.logsConn = u.State
	if u.Note != "" {
		h.logsNote = sanitizeLine(cutBytes(u.Note, 512))
	}
	if u.Err != nil {
		h.logsErr = u.Err
	}
	if !u.Has {
		return
	}
	h.logsErr = nil
	in := u.Value
	if len(in) > logKeep {
		in = in[len(in)-logKeep:] // only the newest can survive anyway
	}
	if u.Reset {
		h.logs, h.logBack = h.logs[:0], 0
	}
	added, q := 0, h.query()
	for _, l := range in {
		c := cleanHit(l)
		h.logs = append(h.logs, c)
		if h.shows(c, q) {
			added++
		}
	}
	if !h.follow && !u.Reset {
		// Scrolled back or paused: the window keeps its lines, so it moves
		// away from the newest by what arrived in view.
		h.logBack += added
	}
	if over := len(h.logs) - logKeep; over > 0 {
		n := copy(h.logs, h.logs[over:])
		clear(h.logs[n:])
		h.logs = h.logs[:n]
	}
	h.logBack = min(h.logBack, len(h.visibleLogs()))
}

func severityRank(s string) int {
	switch s {
	case "warn":
		return 1
	case "error", "critical":
		return 2
	}
	return 0
}

// containsFold reports whether s holds sub, ignoring ASCII case. sub is
// already lower case. It is a literal match, never a pattern, and allocates
// nothing.
func containsFold(s, sub string) bool {
	n := len(sub)
	for i := 0; i+n <= len(s); i++ {
		j := 0
		for ; j < n; j++ {
			c := s[i+j]
			if c >= 'A' && c <= 'Z' {
				c += 'a' - 'A'
			}
			if c != sub[j] {
				break
			}
		}
		if j == n {
			return true
		}
	}
	return n == 0
}

// visibleLogs are the kept lines at or above the level that contain the
// filter text.
func (h *hostScreen) visibleLogs() []api.LogHit {
	out := make([]api.LogHit, 0, len(h.logs))
	q := h.query()
	for _, l := range h.logs {
		if h.shows(l, q) {
			out = append(out, l)
		}
	}
	return out
}

func (h *hostScreen) query() string {
	return strings.ToLower(strings.TrimSpace(h.logFilter.Value()))
}

// shows: at or above the level and holding the filter text.
func (h *hostScreen) shows(l api.LogHit, q string) bool {
	if severityRank(l.Severity) < h.logMin {
		return false
	}
	return q == "" || containsFold(l.Line, q)
}

func (h *hostScreen) logsKey(a *App, k tea.KeyPressMsg) (tea.Cmd, bool) {
	if h.logEditing {
		switch k.String() {
		case "enter":
			h.logEditing = false
			h.logFilter.Blur()
		case "esc":
			h.logEditing = false
			h.logFilter.SetValue("")
			h.logFilter.Blur()
		default:
			// Every other key is text, whatever it would do outside the
			// field. The field's own commands (cursor blink) are dropped.
			h.logFilter, _ = h.logFilter.Update(k)
		}
		return nil, true
	}
	switch {
	case key.Matches(k, navKeys.Filter):
		h.logEditing = true
		_ = h.logFilter.Focus()
	case key.Matches(k, logKeys.Level):
		h.logMin = (h.logMin + 1) % len(levelNames)
		h.logBack = 0
	case key.Matches(k, logKeys.Follow):
		h.follow, h.logBack = !h.follow, 0
	case key.Matches(k, navKeys.Up):
		h.follow = false
		h.logBack++
	case key.Matches(k, navKeys.Down):
		h.logBack = max(h.logBack-1, 0)
	case key.Matches(k, logKeys.Explain):
		return h.askExplain(a), true
	default:
		return nil, false
	}
	return nil, true
}

// explainLines are the lines Explain would send: the newest error lines in
// view, newest last. Each is cut on a rune boundary, and the oldest are
// dropped until the JSON-encoded request fits explainBodyMax.
func (h *hostScreen) explainLines() []string {
	vis := h.visibleLogs()
	var rev []string
	total := 0
	for i := len(vis) - 1; i >= 0 && len(rev) < explainMax; i-- {
		if severityRank(vis[i].Severity) != 2 {
			continue
		}
		l := cutBytes(vis[i].Line, explainLineMax)
		enc, err := json.Marshal(l)
		if err != nil {
			continue
		}
		if total+len(enc)+1 > explainBodyMax {
			break
		}
		total += len(enc) + 1
		rev = append(rev, l)
	}
	lines := make([]string, len(rev))
	for i, l := range rev {
		lines[len(rev)-1-i] = l
	}
	return lines
}

// askExplain opens the explain modal and fetches the provider disclosure; the
// person sees what will leave the machine, and confirms, before anything is
// sent.
func (h *hostScreen) askExplain(a *App) tea.Cmd {
	ctx, cancel := context.WithCancel(h.ctx)
	m := &explainModal{h: h, lines: h.explainLines(), ctx: ctx, cancel: cancel}
	a.modal = m
	id, gen, be := h.id, h.gen, a.be
	return func() tea.Msg {
		s, err := be.Settings(ctx)
		return disclosureMsg{id: id, gen: gen, disclosure: s.AIDisclosure, provider: s.AIProvider, err: err}
	}
}

// unitShort names a unit by its role: ...-exec.service is "exec".
func unitShort(unit string) string {
	u := strings.TrimSuffix(unit, ".service")
	if i := strings.LastIndex(u, "-"); i >= 0 {
		return u[i+1:]
	}
	return u
}

func (h *hostScreen) logLine(a *App, l api.LogHit) string {
	sev := "     "
	switch l.Severity {
	case "warn":
		sev = a.th.Warn.Render("WARN ")
	case "error":
		sev = a.th.Bad.Render("ERROR")
	case "critical":
		sev = a.th.Bad.Render("CRIT ")
	}
	return fmt.Sprintf(" %s %-6s %s %s", l.At.In(a.loc).Format("15:04:05"), sanitizeLine(unitShort(l.Unit)), sev, sanitizeLine(l.Line))
}

func (h *hostScreen) viewLogs(a *App, w, hgt int) string {
	vis := h.visibleLogs()
	follow := "follow off"
	if h.follow {
		follow = "follow on"
	}
	head := fmt.Sprintf(" %s %s %d lines %s %s %s %s", a.th.Title.Render("LOGS"), h.logsConn.String(), len(vis), a.gl.Sep, levelNames[h.logMin], a.gl.Sep, follow)
	if q := strings.TrimSpace(h.logFilter.Value()); q != "" {
		head += fmt.Sprintf(" %s filter %q", a.gl.Sep, sanitizeLine(q))
	}
	out := []string{head}
	if h.logsNote != "" {
		out = append(out, " "+a.th.Warn.Render(sanitizeLine(h.logsNote)))
	}
	if h.logsErr != nil {
		out = append(out, " "+a.th.Bad.Render(a.errText(h.logsErr)))
	}
	room := max(hgt-len(out)-2, 1)
	if h.follow {
		h.logBack = 0
	}
	h.logBack = min(h.logBack, max(len(vis)-room, 0))
	end := max(len(vis)-h.logBack, 0)
	start := max(end-room, 0)
	for _, l := range vis[start:end] {
		out = append(out, h.logLine(a, l))
	}
	if len(vis) == 0 {
		out = append(out, " "+a.th.Dim.Render("no lines yet"))
	}
	for len(out) < hgt-1 {
		out = append(out, "")
	}
	foot := " / filter  ! level  f follow  " + a.gl.Sep + "  " + a.gl.UpDown + " scroll  e explain"
	if h.logEditing {
		foot = " " + h.logFilter.View()
	}
	return strings.Join(append(out, a.th.Dim.Render(foot)), "\n")
}

type explainState int

const (
	explainAsking explainState = iota // disclosure shown; waiting for enter
	explainSending
	explainDone
	explainFailed
)

// explainModal first shows what would be sent and the provider disclosure,
// then the answer with exactly what the server says went out (spec A11).
type explainModal struct {
	h          *hostScreen
	lines      []string // what a send would carry; the server redacts it
	state      explainState
	disclosure string
	provider   string
	discLoaded bool
	err        error
	e          api.Explain
	ctx        context.Context
	cancel     context.CancelFunc
}

func (m *explainModal) gotDisclosure(msg disclosureMsg) {
	if msg.err != nil {
		m.err = msg.err
		return
	}
	m.disclosure, m.provider, m.discLoaded = msg.disclosure, msg.provider, true
}

func (m *explainModal) gotExplain(msg explainMsg) {
	if msg.err != nil {
		m.state, m.err = explainFailed, msg.err
		return
	}
	m.state, m.e, m.err = explainDone, msg.e, nil
}

func (m *explainModal) key(a *App, k tea.KeyPressMsg) (bool, tea.Cmd) {
	switch k.String() {
	case "esc", "q":
		m.cancel() // abandons a request in flight; its reply finds no modal
		return true, nil
	case "enter":
		switch m.state {
		case explainDone, explainFailed:
			m.cancel()
			return true, nil
		case explainAsking:
			if !m.discLoaded {
				return false, nil
			}
			m.state = explainSending
			id, gen, be, ctx, lines := m.h.id, m.h.gen, a.be, m.ctx, m.lines
			return false, func() tea.Msg {
				e, err := be.Explain(ctx, id, lines)
				return explainMsg{id: id, gen: gen, e: e, err: err}
			}
		}
	}
	return false, nil
}

// explainErrText says what went wrong in terms of what to do, then the
// server's own message and hint.
func explainErrText(a *App, err error) string {
	var e *api.Error
	if errors.As(err, &e) {
		switch {
		case e.Code == api.CodeAIUnconfigured:
			return "no AI provider is set up: " + a.errText(err)
		case e.Status == 413 || e.Code == api.CodeTooManyLines || e.Code == api.CodeTooLarge:
			return "too much to send: narrow the filter or the level and try again: " + a.errText(err)
		case e.Status == 502 || e.Code == api.CodeUpstream:
			return "the AI provider failed: " + a.errText(err)
		}
	}
	return a.errText(err)
}

func (m *explainModal) view(a *App, w, _ int) string {
	wrap := lipgloss.NewStyle().Width(max(w-2, 20))
	para := func(s string) string {
		return " " + strings.ReplaceAll(wrap.Render(s), "\n", "\n ") + "\n"
	}
	out := " " + a.th.Title.Render("EXPLAIN") + a.th.Dim.Render("  (esc to close)") + "\n\n"
	switch m.state {
	case explainAsking, explainSending:
		n := len(m.lines)
		if n == 0 {
			prov := sanitizeLine(cutBytes(m.provider, 64))
			if prov == "" {
				prov = "the AI provider"
			}
			out += para("No error lines in view; jumpgate will fetch the box's recent error lines (redacted) and send them to " + prov + ".")
		} else {
			out += para(fmt.Sprintf("%d error %s from this view would be sent to the AI provider.", n, plural(n, "line", "lines")))
		}
		switch {
		case m.err != nil:
			out += "\n" + para(a.th.Bad.Render("could not read the provider disclosure, so nothing is sent: ")+a.errText(m.err))
		case !m.discLoaded:
			out += "\n " + a.th.Dim.Render("reading the provider disclosure"+a.gl.Ellipsis) + "\n"
		default:
			d := sanitizeLine(cutBytes(m.disclosure, 2048))
			if d == "" {
				d = "the server sent no disclosure text"
			}
			out += "\n" + para(d)
			switch {
			case m.state == explainSending:
				out += "\n " + a.th.Dim.Render("asking"+a.gl.Ellipsis) + "\n"
			default:
				out += "\n " + a.th.Key.Render("enter") + " send   " + a.th.Key.Render("esc") + " cancel\n"
			}
		}
	case explainFailed:
		out += para(a.th.Bad.Render(explainErrText(a, m.err)))
		out += "\n " + a.th.Dim.Render("nothing was explained") + "\n"
	case explainDone:
		// The answer is the provider's text, steered by the box's logs: it is
		// untrusted, so it is sanitized and wrapped, never interpreted.
		text := sanitize(cutBytes(m.e.Text, explainTextMax))
		for _, p := range strings.Split(text, "\n") {
			out += para(strings.ReplaceAll(p, "\t", "    "))
		}
		n := len(m.e.SentExcerpt)
		sent := fmt.Sprintf("sent %d %s", n, plural(n, "line", "lines"))
		if m.e.Redacted {
			sent += ", addresses and peer ids removed"
		} else {
			sent += ", unredacted (local provider)"
		}
		out += "\n " + a.th.Dim.Render(sent) + "\n"
		// What the server says went out, as received: redacted there, never
		// derived here.
		for i, l := range m.e.SentExcerpt {
			if i == explainSentMax {
				out += " " + a.th.Dim.Render(fmt.Sprintf("%s %d more", a.gl.Ellipsis, n-i)) + "\n"
				break
			}
			out += " " + a.th.Dim.Render("  "+sanitizeLine(cutBytes(l, 512))) + "\n"
		}
		if m.discLoaded && m.disclosure != "" {
			out += para(a.th.Dim.Render(sanitizeLine(cutBytes(m.disclosure, 2048))))
		}
	}
	return out
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}
