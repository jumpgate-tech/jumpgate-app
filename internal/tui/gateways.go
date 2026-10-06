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

// gatewaysScreen lists the eRPC gateways read-only (spec D28): their
// operations are not on the agent path yet.
type gatewaysScreen struct {
	gws    []api.GatewaySummary
	has    bool
	err    error
	offset int // clamped against the content when drawn
}

func (s *gatewaysScreen) capturing() bool { return false }
func (s *gatewaysScreen) keys() []key.Binding {
	return []key.Binding{navKeys.Up, navKeys.Down, gatewayKeys.Reload}
}

func (s *gatewaysScreen) update(a *App, msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case enterMsg:
		return loadGateways(a)
	case gatewaysMsg:
		if msg.err != nil {
			s.err = msg.err
			break
		}
		s.gws, s.has, s.err = msg.gws, true, nil
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, navKeys.Up):
			s.offset = max(s.offset-1, 0)
		case key.Matches(msg, navKeys.Down):
			s.offset++
		case key.Matches(msg, gatewayKeys.Reload):
			return loadGateways(a)
		}
	}
	return nil
}

func (s *gatewaysScreen) view(a *App, w, h int) string {
	head := truncate(" "+a.th.Title.Render("GATEWAYS")+a.th.Dim.Render(" (read-only; change them in the web app)"), w, a.gl.Ellipsis)
	switch {
	case s.err != nil && !s.has:
		return head + "\n\n " + a.th.Bad.Render("unavailable: ") + a.errText(s.err)
	case !s.has:
		return head + "\n\n " + a.th.Dim.Render("loading"+a.gl.Ellipsis)
	case len(s.gws) == 0:
		return head + "\n\n " + a.th.Dim.Render("no gateways")
	}
	var lines []string
	for _, g := range s.gws {
		lines = append(lines, gatewayLines(a, g)...)
		lines = append(lines, "")
	}
	return head + "\n\n" + strings.Join(window(lines, &s.offset, h-2, w, a.gl.Ellipsis), "\n")
}
