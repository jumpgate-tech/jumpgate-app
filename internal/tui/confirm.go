package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

// confirmModal is the typed confirmation for destructive actions (spec D21):
// the person types want exactly (surrounding spaces aside) and presses
// enter. Anything else re-asks; esc cancels. It guards against mistakes; the
// approval tier, when it exists, guards authority.
//
// title and detail usually quote a target or service name, so they are
// sanitized here. want is not: it must be exactly what the screen shows, so a
// want that sanitize would change, or that is blank, makes a refusal that
// never runs the action. Otherwise a name that sanitizes to nothing would
// confirm on a bare enter, and two names that sanitize alike would accept
// the same word.
type confirmModal struct {
	title, detail, want string
	prompt              string // "Type %s to ...", for a caller with its own words; "" is the default
	refused             string // non-empty: this confirmation can never run
	in                  textinput.Model
	miss                string
	run                 func() tea.Cmd
}

func newConfirm(title, detail, want string, run func() tea.Cmd) *confirmModal {
	in := textinput.New()
	in.Prompt = "> "
	in.CharLimit = 128
	// ctrl+v would read the system clipboard by running pbpaste, xclip or
	// wl-paste; the terminal's bracketed paste fills the field without that.
	in.KeyMap.Paste.SetEnabled(false)
	in.Focus()
	m := &confirmModal{title: sanitizeLine(title), detail: sanitize(detail), want: want, in: in, run: run}
	if strings.TrimSpace(want) == "" || sanitizeLine(want) != want {
		m.want = sanitizeLine(want)
		m.refused = "This name cannot be confirmed here: it is blank, or has characters\n the screen cannot show. Use the jumpgate CLI."
	}
	return m
}

func (m *confirmModal) key(a *App, k tea.KeyPressMsg) (bool, tea.Cmd) {
	switch k.String() {
	case "esc":
		a.flash = "cancelled; nothing was changed"
		return true, nil
	case "enter":
		typed := strings.TrimSpace(m.in.Value())
		if m.refused != "" || typed == "" || typed != m.want {
			if m.refused == "" {
				m.miss = fmt.Sprintf("that is not %q; type it exactly, or esc to cancel", m.want)
			}
			m.in.SetValue("")
			return false, nil
		}
		return true, m.run()
	}
	var cmd tea.Cmd
	m.in, cmd = m.in.Update(k)
	return false, cmd
}

// paste puts a bracketed paste into the field. The field turns its newlines
// into spaces, so a copied line never confirms by itself: enter still has to
// be pressed, and the trimmed text still has to match.
func (m *confirmModal) paste(_ *App, p tea.PasteMsg) tea.Cmd {
	var cmd tea.Cmd
	m.in, cmd = m.in.Update(p)
	return cmd
}

func (m *confirmModal) view(a *App, w, _ int) string {
	s := a.th.Bad.Render(" "+m.title) + "\n\n " + m.detail + "\n\n"
	if m.refused != "" {
		return s + " " + a.th.Warn.Render(m.refused) + "\n\n " + a.th.Dim.Render("esc to close")
	}
	prompt := m.prompt
	if prompt == "" {
		prompt = "Type %s to confirm, esc to cancel:"
	}
	s += fmt.Sprintf(" "+prompt+"\n ", a.th.Key.Render(m.want)) + m.in.View()
	if m.miss != "" {
		s += "\n\n " + a.th.Warn.Render(m.miss)
	}
	return s
}
