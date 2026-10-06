package tui

import (
	"context"
	"strings"
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

// A second R while a restart is under way starts nothing: two restarts at
// once would race over which server is current.
func TestASecondRestartWhileOneIsUnderWayIsANoOp(t *testing.T) {
	for _, where := range []string{"global", "signers"} {
		t.Run(where, func(t *testing.T) {
			f := tuitest.NewFake()
			var a *App
			if where == "signers" {
				a = signersApp(t, f)
			} else {
				a = newTestApp(t, f, 80, 24, "unicode")
			}
			calls := 0
			next := tuitest.NewFake()
			a.o.Restart = func(context.Context) (Backend, error) { calls++; return next, nil }
			if where == "global" {
				a.skew = "server v0.8.0, this jumpgate v-test: press R to restart the server"
			}
			m, first := tuitest.Send(a, tuitest.Key("R"))
			if len(first) == 0 || first[0] == nil {
				t.Fatal("R started no restart")
			}
			m, second := tuitest.Send(m, tuitest.Key("R"))
			for _, c := range second {
				if c != nil {
					c()
				}
			}
			if !strings.Contains(m.(*App).flash, "already restarting") {
				t.Fatalf("flash %q", m.(*App).flash)
			}
			m = tuitest.Settle(m, first...)
			if calls != 1 || m.(*App).be != Backend(next) {
				t.Fatalf("restart ran %d times", calls)
			}
			// Once it is done, R works again.
			m.(*App).skew = "skew"
			if _, again := tuitest.Send(m, tuitest.Key("R")); len(again) == 0 || again[0] == nil {
				t.Fatal("R did nothing after the restart finished")
			}
		})
	}
}
