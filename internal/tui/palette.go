package tui

import (
	"context"
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/valve-tech/jumpgate/internal/catalog"
)

// paletteModal is `:`: a typed command. It only ever reaches what the keys
// reach, by the same functions: stop, restart and remove open the same typed
// confirmation as their keys, the shell asks the server for its command and
// screens it, and nothing here calls the backend or runs a program itself
// (spec D21). Text typed here is matched literally against fixed command
// words and known box names; it is never compiled as a pattern.
type paletteModal struct {
	in   textinput.Model
	err  string
	hint string // the names tab could not choose between
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
	"fleet": scrFleet, "signers": scrSigners, "gateways": scrGateways, "vpn": scrVPN, "jobs": scrJobs, "vault": scrVault, "server": scrServer, "settings": scrSettings,
}

// paletteCommands is what run takes, as the palette lists it (see Commands):
// NAME is a box, TOPIC a guide topic, a|b one of the words, [x] optional.
var paletteCommands = []string{"fleet", "hosts", "signers", "gateways", "vpn", "jobs", "vault", "server", "settings", "host NAME", "logs NAME",
	"measure NAME", "ssh NAME", "start|stop|restart exec|beacon NAME", "remove NAME", "migrate-names NAME", "add", "guide", "invite", "unlock",
	"help [TOPIC]", "keys", "quit"}

var paletteHelp = strings.Join(paletteCommands, " | ")

func (m *paletteModal) key(a *App, k tea.KeyPressMsg) (bool, tea.Cmd) {
	switch k.String() {
	case "esc":
		return true, nil
	case "tab":
		m.complete(a)
		return false, nil
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
	if f[0] == "hosts" && len(f) == 1 { // the box table mode of Fleet
		cmd := a.openScreen(scrFleet)
		fs := a.screens[scrFleet].(*fleetScreen)
		fs.hostMode = true
		return false, cmd
	}
	boxCmd := func(name string) bool { return len(f) == 2 && f[0] == name }
	if len(f) == 2 && (f[0] == "host" || f[0] == "logs" || f[0] == "measure" || f[0] == "ssh" || f[0] == "remove" || f[0] == "migrate-names") && !known(a, f[1]) {
		return bad("no box named %q", truncate(sanitizeLine(f[1]), 32, a.gl.Ellipsis))
	}
	switch {
	case f[0] == "keys" && len(f) == 1: // what ? shows
		a.modal = newHelp(a)
		return false, nil
	case f[0] == "help" && len(f) <= 2: // the guide: its topics, or one page
		topic := ""
		if len(f) == 2 {
			topic = f[1]
		}
		g, ok := newManual(topic)
		if !ok {
			return bad("no guide topic %q; the topics are %s", truncate(sanitizeLine(topic), 32, a.gl.Ellipsis), manualTopicNames())
		}
		a.modal = g
		return false, nil
	case f[0] == "guide" && len(f) == 1: // the setup checklist, as g
		a.modal = newGuide(a)
		return false, loadGuide(a)
	case f[0] == "invite" && len(f) == 1: // the line to run on a box, as i
		return false, a.toggleInvite()
	case f[0] == "unlock" && len(f) == 1: // the passphrase prompt, as U
		a.modal = newUnlock()
		return false, nil
	case f[0] == "quit" && len(f) == 1:
		a.cancel()
		return false, tea.Quit
	case f[0] == "add" && len(f) == 1:
		cmd := a.openScreen(scrFleet)
		fl := newAddFlow()
		a.screens[scrFleet].(*fleetScreen).hosts.startFlow(a, fl)
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
	case boxCmd("migrate-names"):
		id := f[1]
		a.modal = newConfirm("Rename the node on "+id,
			"Starts the node.migrate-names job: it stops the node services, renames the "+catalog.LegacyNames.Prefix+
				" account, units and data root to "+catalog.CurrentNames.Prefix+", and starts the services again. It asks you before anything it cannot undo.",
			id, func() tea.Cmd {
				return a.do("migrate-names", func(ctx context.Context, be Backend) tea.Msg {
					v, err := be.MigrateNames(ctx, id)
					if err != nil {
						return errMsg{what: "migrate-names " + id, err: err}
					}
					return openJobsMsg{flash: "job " + v.ID + " started on " + id}
				})
			})
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
	out := " " + a.th.Title.Render("COMMAND") + a.th.Dim.Render("  (enter to run, tab to complete, esc to close)") + "\n\n " + m.in.View() + "\n\n" + a.th.Dim.Render(wrapIndent(paletteHelp, w, " "))
	if m.hint != "" {
		out += "\n\n" + a.th.Dim.Render(wrapIndent("matches: "+m.hint, w, " "))
	}
	if m.err != "" {
		out += "\n\n" + a.th.Warn.Render(wrapIndent(m.err, w, " "))
	}
	return out
}

// wrapIndent wraps s at words to w cells, each line led by indent: the list
// of commands is longer than a line, and a cut one would hide the last.
func wrapIndent(s string, w int, indent string) string {
	return indent + strings.ReplaceAll(ansi.Wrap(s, w-lipgloss.Width(indent)-1, ""), "\n", "\n"+indent)
}

// paletteWords are the first words tab completes: every command's own word.
var paletteWords = []string{"fleet", "hosts", "signers", "gateways", "vpn", "jobs", "vault", "server", "settings", "host", "logs", "measure",
	"ssh", "start", "stop", "restart", "remove", "migrate-names", "add", "guide", "invite", "unlock", "help", "keys", "quit"}

// complete is tab: it finishes the word being typed from the command words,
// the services, a guide topic or a box name, whichever the line is at. One
// match fills the word in; several fill in what they share and are listed.
func (m *paletteModal) complete(a *App) {
	m.err, m.hint = "", ""
	v := m.in.Value()
	words := strings.Fields(v)
	trailing := strings.HasSuffix(v, " ")
	n := len(words)
	if trailing {
		n++ // the word being started is empty
	}
	prefix := ""
	if !trailing && len(words) > 0 {
		prefix = words[len(words)-1]
	}
	var boxes []string
	for _, r := range a.fleet.Rows {
		boxes = append(boxes, sanitizeLine(r.ID))
	}
	var cands []string
	switch {
	case n == 1:
		cands = paletteWords
	case words[0] == "help" && n == 2:
		for _, t := range manualSource.Topics() {
			cands = append(cands, t.Name)
		}
	case contains([]string{"host", "logs", "measure", "ssh", "remove", "migrate-names"}, words[0]) && n == 2:
		cands = boxes
	case contains([]string{"start", "stop", "restart"}, words[0]) && n == 2:
		cands = services
	case contains([]string{"start", "stop", "restart"}, words[0]) && n == 3:
		cands = boxes
	default:
		return
	}
	var match []string
	for _, c := range cands {
		if strings.HasPrefix(c, prefix) {
			match = append(match, c)
		}
	}
	if len(match) == 0 {
		return
	}
	head := strings.TrimSuffix(v, prefix)
	common := match[0]
	for _, c := range match[1:] {
		for !strings.HasPrefix(c, common) {
			common = common[:len(common)-1]
		}
	}
	if len(match) == 1 {
		common += " "
	} else {
		m.hint = strings.Join(match, "  ")
	}
	m.in.SetValue(head + common)
	m.in.CursorEnd()
}
