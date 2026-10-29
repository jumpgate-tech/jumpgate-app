package tui

import (
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/valve-tech/jumpgate/internal/api"
)

var gatewayKeys = struct{ Reload key.Binding }{
	Reload: key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "reload")),
}

// gatewaysScreen lists the eRPC gateways (spec D28); m turns the selected
// one's modules on or off. The rest is changed in the web app for now.
type gatewaysScreen struct {
	gws    []api.GatewaySummary
	has    bool
	err    error
	cursor int // the selected gateway
	offset int // clamped against the content when drawn
}

func (s *gatewaysScreen) capturing() bool { return false }
func (s *gatewaysScreen) keys() []key.Binding {
	return []key.Binding{navKeys.Up, navKeys.Down, gwModKey, gatewayKeys.Reload}
}

func (s *gatewaysScreen) update(a *App, msg tea.Msg) tea.Cmd {
	if cmd, ok := s.gwModulesUpdate(a, msg); ok {
		return cmd
	}
	switch msg := msg.(type) {
	case enterMsg:
		return loadGateways(a)
	case gatewaysMsg:
		if msg.err != nil {
			s.err = msg.err
			break
		}
		s.gws, s.has, s.err = msg.gws, true, nil
		s.cursor = min(s.cursor, max(len(s.gws)-1, 0))
	case tea.KeyPressMsg:
		if a.active != scrGateways || a.detail != nil {
			return nil
		}
		switch {
		case key.Matches(msg, navKeys.Up):
			s.cursor = max(s.cursor-1, 0)
		case key.Matches(msg, navKeys.Down):
			s.cursor = min(s.cursor+1, max(len(s.gws)-1, 0))
		case key.Matches(msg, gwModKey):
			if s.cursor < len(s.gws) {
				m, cmd := newGwModules(a, s.gws[s.cursor])
				a.modal = m
				return cmd
			}
		case key.Matches(msg, gatewayKeys.Reload):
			return loadGateways(a)
		}
	}
	return nil
}

func (s *gatewaysScreen) view(a *App, w, h int) string {
	head := truncate(a.th.Dim.Render(" m modules; the rest in the web app"), w, a.gl.Ellipsis) // the title says Gateways
	switch {
	case s.err != nil && !s.has:
		return head + "\n\n " + a.th.Bad.Render("unavailable: ") + a.errText(s.err)
	case !s.has:
		return head + "\n\n " + a.th.Dim.Render("loading"+a.gl.Ellipsis)
	case len(s.gws) == 0:
		return head + "\n\n " + a.th.Dim.Render("no gateways")
	}
	var lines []string
	for i, g := range s.gws {
		gl := gatewayLines(a, g)
		if i == s.cursor {
			s.offset = len(lines)
			gl[0] = a.gl.Sel + gl[0][1:]
		}
		lines = append(lines, gl...)
		lines = append(lines, "")
	}
	return head + "\n\n" + strings.Join(window(lines, &s.offset, h-2, w, a.gl.Ellipsis), "\n")
}
