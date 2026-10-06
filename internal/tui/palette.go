package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

// paletteModal is `:`: a typed command. It only ever reaches what the keys
// reach, by the same functions: stop, restart and remove open the same typed
// confirmation as their keys, the shell asks the server for its command and
// screens it, and nothing here calls the backend or runs a program itself
// (spec D21). Text typed here is matched literally against fixed command
// words and known box names; it is never compiled as a pattern.
type paletteModal struct {
	in  textinput.Model
	err string
}

func newPalette() *paletteModal {
	in := textinput.New()
	in.Prompt = ":"
	in.CharLimit = 128
	in.KeyMap.Paste.SetEnabled(false) // terminal paste still works
	in.Focus()
	return &paletteModal{in: in}
}

var paletteScreens = map[string]screenID{
	"fleet": scrFleet, "hosts": scrHosts, "signers": scrSigners, "gateways": scrGateways, "jobs": scrJobs, "settings": scrSettings,
}

const paletteHelp = "fleet hosts signers gateways jobs settings | host NAME | logs NAME | measure NAME | ssh NAME | start|stop|restart exec|beacon NAME | remove NAME | add | help | quit"

func (m *paletteModal) key(a *App, k tea.KeyPressMsg) (bool, tea.Cmd) {
	switch k.String() {
	case "esc":
		return true, nil
	case "enter":
		a.modal = nil // a confirmation or the help the command opens replaces the palette
		stay, cmd := m.run(a)
		if stay {
			a.modal = m
			return false, cmd
		}
		return a.modal == nil, cmd
	}
	var cmd tea.Cmd
	m.in, cmd = m.in.Update(k)
	return false, cmd
}

// paste puts a bracketed paste into the field as one clean line.
func (m *paletteModal) paste(_ *App, p tea.PasteMsg) tea.Cmd {
	var cmd tea.Cmd
	m.in, cmd = m.in.Update(tea.PasteMsg{Content: sanitizeLine(p.Content)})
	return cmd
}

// known: a box the palette may act on. Until the fleet has loaded any name
// is let through (the server refuses an unknown one); after, only a listed one.
func known(a *App, id string) bool {
	if !a.fleetHas {
		return true
	}
	_, ok := a.fleet.Row(id)
	return ok
}

// openOn opens a box on a tab, as the tab keys do. It reports false, with the
// reason in a.flash, when the person has hidden that tab.
func openOn(a *App, id, tab string) (*hostScreen, tea.Cmd, bool) {
	if !contains(a.prefs.HostTabs, tab) {
		a.flash = fmt.Sprintf("the %s tab is hidden; show it in Settings", tab)
		return nil, nil, false
	}
	cmd := a.openHost(id)
	h := a.detail.(*hostScreen)
	h.tab = indexOf(a.prefs.HostTabs, tab)
	return h, tea.Batch(cmd, h.load(a, tab)), true
}

func contains(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}

// run executes the typed line. stay is true when the palette should remain
// open (to show an error).
func (m *paletteModal) run(a *App) (stay bool, cmd tea.Cmd) {
	f := strings.Fields(m.in.Value())
	if len(f) == 0 {
		return false, nil
	}
	bad := func(format string, args ...any) (bool, tea.Cmd) {
		m.err = fmt.Sprintf(format, args...)
		m.in.SetValue("")
		return true, nil
	}
	if id, ok := paletteScreens[f[0]]; ok && len(f) == 1 {
		return false, a.openScreen(id)
	}
	boxCmd := func(name string) bool { return len(f) == 2 && f[0] == name }
	if len(f) == 2 && (f[0] == "host" || f[0] == "logs" || f[0] == "measure" || f[0] == "ssh" || f[0] == "remove") && !known(a, f[1]) {
		return bad("no box named %q", truncate(sanitizeLine(f[1]), 32, a.gl.Ellipsis))
	}
	switch {
	case f[0] == "help" && len(f) == 1:
		a.modal = newHelp(a)
		return false, nil
	case f[0] == "quit" && len(f) == 1:
		a.cancel()
		return false, tea.Quit
	case f[0] == "add" && len(f) == 1:
		cmd := a.openScreen(scrHosts)
		fl := newAddFlow()
		a.screens[scrHosts].(*hostsScreen).startFlow(a, fl)
		return false, tea.Batch(cmd, fl.focusCmd())
	case boxCmd("host"):
		return false, a.openHost(f[1])
	case boxCmd("logs"):
		_, cmd, _ := openOn(a, f[1], "logs")
		return false, cmd
	case boxCmd("measure"):
		h, cmd, ok := openOn(a, f[1], "storage")
		if !ok {
			return false, nil
		}
		return false, tea.Batch(cmd, h.measure(a))
	case boxCmd("ssh"):
		cmd := a.openHost(f[1])
		return false, tea.Batch(cmd, a.detail.(*hostScreen).shell(a))
	case boxCmd("remove"):
		confirmRemove(a, f[1])
		return false, nil
	case len(f) == 3 && (f[0] == "start" || f[0] == "stop" || f[0] == "restart") && (f[1] == "exec" || f[1] == "beacon"):
		if !known(a, f[2]) {
			return bad("no box named %q", truncate(sanitizeLine(f[2]), 32, a.gl.Ellipsis))
		}
		cmd := a.openHost(f[2])
		scmd := a.detail.(*hostScreen).serviceAction(a, f[1], f[0])
		return false, tea.Batch(cmd, scmd)
	}
	return bad("unknown command %q", truncate(sanitizeLine(m.in.Value()), 40, a.gl.Ellipsis))
}

func (m *paletteModal) view(a *App, w, _ int) string {
	out := " " + a.th.Title.Render("COMMAND") + a.th.Dim.Render("  (enter to run, esc to close)") + "\n\n " + m.in.View() + "\n\n " + a.th.Dim.Render(paletteHelp)
	if m.err != "" {
		out += "\n\n " + a.th.Warn.Render(m.err)
	}
	return out
}
