package tui

import (
	tea "charm.land/bubbletea/v2"
)

// newActions is `x` on an open box: every action for it in one list, each the
// same command its own key runs (and so the same typed confirmations). The
// service actions have no single key: their keys act on the service selected
// on the Services tab.
func newActions(_ *App, h *hostScreen) *menuModal {
	m := &menuModal{h: h, title: "ACTIONS"}
	if h != nil {
		m.title = "ACTIONS for " + sanitizeLine(h.id)
	}
	keyOf := map[string]string{"restart": "e", "stop": "t", "start": "s"}
	for _, svc := range services {
		for _, act := range []string{"restart", "stop", "start"} {
			svc, act := svc, act
			m.all = append(m.all, menuItem{label: act + " " + svc, hint: "Services tab: " + keyOf[act],
				run: func(a *App) tea.Cmd { return h.serviceAction(a, svc, act) }})
		}
	}
	m.all = append(m.all,
		menuItem{label: "measure disk", hint: "Storage tab: " + hostKeys.Measure.Help().Key, run: func(a *App) tea.Cmd {
			h.tab = indexOf(a.prefs.HostTabs, "storage")
			return tea.Batch(h.load(a, "storage"), h.measure(a))
		}},
		menuItem{label: "open an SSH shell", key: svcKeys.Shell.Help().Key, run: func(a *App) tea.Cmd { return h.shell(a) }},
		menuItem{label: "modules", key: hostKeys.Modules.Help().Key, run: func(a *App) tea.Cmd {
			a.modal = newModulesModal(a, h)
			return nil
		}},
	)
	return m
}

func indexOf(xs []string, x string) int {
	for i, v := range xs {
		if v == x {
			return i
		}
	}
	return 0
}
