package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

// actionsModal is `x`: every action for the open box in one list, each the
// same command its own key runs (and so the same typed confirmations).
type actionsModal struct {
	h      *hostScreen
	items  []string
	cursor int
}

func newActions(_ *App, h *hostScreen) *actionsModal {
	return &actionsModal{h: h, items: []string{
		"restart exec", "stop exec", "start exec",
		"restart beacon", "stop beacon", "start beacon",
		"measure disk", "open an SSH shell",
	}}
}

func (m *actionsModal) key(a *App, k tea.KeyPressMsg) (bool, tea.Cmd) {
	switch {
	case k.String() == "esc":
		return true, nil
	case key.Matches(k, navKeys.Up):
		m.cursor = max(m.cursor-1, 0)
	case key.Matches(k, navKeys.Down):
		m.cursor = min(m.cursor+1, len(m.items)-1)
	case key.Matches(k, navKeys.Enter):
		item := m.items[m.cursor]
		a.modal = nil // a confirmation opened by the action replaces this menu
		switch item {
		case "measure disk":
			m.h.tab = indexOf(a.prefs.HostTabs, "storage")
			return true, tea.Batch(m.h.load(a, "storage"), m.h.measure(a))
		case "open an SSH shell":
			return true, m.h.shell(a)
		}
		f := strings.Fields(item) // action, service
		cmd := m.h.serviceAction(a, f[1], f[0])
		return a.modal == nil, cmd // done unless a confirmation took over
	}
	return false, nil
}

func (m *actionsModal) view(a *App, w, _ int) string {
	out := fmt.Sprintf(" %s %s\n\n", a.th.Title.Render("ACTIONS"), a.th.Dim.Render("for "+sanitizeLine(m.h.id)+" (enter to run, esc to close)"))
	for i, it := range m.items {
		mark := " "
		if i == m.cursor {
			mark = a.gl.Sel
		}
		out += fmt.Sprintf(" %s %s\n", mark, it)
	}
	return out
}

func indexOf(xs []string, x string) int {
	for i, v := range xs {
		if v == x {
			return i
		}
	}
	return 0
}
