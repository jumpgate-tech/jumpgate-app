package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/valve-tech/jumpgate/internal/tui/tuitest"
)

// A command reads the backend it was made with: a restart that swaps a.be
// before the command runs neither races with it nor sends the old screen's
// request to the new server.
func TestCommandsUseTheBackendTheyWereMadeWith(t *testing.T) {
	for name, c := range map[string]struct {
		call string
		make func(a *App) tea.Cmd
	}{
		"hosts":    {"Targets", loadTargets},
		"gateways": {"Gateways", loadGateways},
		"prefs":    {"Prefs", func(a *App) tea.Cmd { return a.fetchPrefs() }},
		"refetch":  {"Prefs", func(a *App) tea.Cmd { return a.refetchPrefs() }},
		"refresh": {"Fleet", func(a *App) tea.Cmd {
			a.active = scrFleet
			return a.screens[scrFleet].update(a, tuitest.Key("r"))
		}},
		"remove": {"RemoveTarget", func(a *App) tea.Cmd {
			confirmRemove(a, "box-a")
			return a.modal.(*confirmModal).run()
		}},
	} {
		t.Run(name, func(t *testing.T) {
			old, next := tuitest.NewFake(), tuitest.NewFake()
			a := newTestApp(t, old, 80, 24, "unicode")
			cmd := c.make(a)
			if cmd == nil {
				t.Fatal("no command")
			}
			a.Update(restartedMsg{be: next})
			cmd()
			if !old.Called(c.call) || next.Called(c.call) {
				t.Fatalf("old %v, new %v", old.Calls, next.Calls)
			}
		})
	}
}
