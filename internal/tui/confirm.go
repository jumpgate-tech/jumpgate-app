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
// title, detail and want usually quote a target or service name, so they are
// sanitized once here. want is compared in its sanitized form too: that is
// the text the person sees and can type; run keeps the raw id it acts on.
type confirmModal struct {
	title, detail, want string
	in                  textinput.Model
	miss                string
	run                 func() tea.Cmd
}

func newConfirm(title, detail, want string, run func() tea.Cmd) *confirmModal {
	in := textinput.New()
	in.Prompt = "> "
	in.CharLimit = 128
	in.Focus()
	return &confirmModal{title: sanitize(title), detail: sanitize(detail), want: sanitize(want), in: in, run: run}
}

func (m *confirmModal) key(a *App, k tea.KeyPressMsg) (bool, tea.Cmd) {
	switch k.String() {
	case "esc":
		a.flash = "cancelled; nothing was changed"
		return true, nil
	case "enter":
		if strings.TrimSpace(m.in.Value()) != m.want {
			m.miss = fmt.Sprintf("that is not %q; type it exactly, or esc to cancel", m.want)
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
	s := a.th.Bad.Render(" "+m.title) + "\n\n " + m.detail + "\n\n" +
		fmt.Sprintf(" Type %s to confirm, esc to cancel:\n ", a.th.Key.Render(m.want)) + m.in.View()
	if m.miss != "" {
		s += "\n\n " + a.th.Warn.Render(m.miss)
	}
	return s
}
