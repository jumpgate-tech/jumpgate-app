package tui

import (
	"context"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/apiclient"
)

// The setup guide: the first-run checklist, derived from the live state
// only. Each step says whether it is done, why it matters in one sentence,
// and what does it; enter runs that existing action (a view, and its key).

// guideFacts are what the steps read.
type guideFacts struct {
	connected bool
	vault     *api.VaultView      // nil: not read yet
	ctl       *api.ControllerView // nil: not read yet
	fleet     *api.Fleet          // nil: no reading yet
	mods      []api.ModulesView   // the modules read so far (the tree's)
	modsKnown bool
	gateways  int // -1: not read yet
	goos      string
}

// guideStep is one line of the guide.
type guideStep struct {
	id, title string
	done      bool
	optional  bool
	why, how  string
	// screen is the view enter opens (-1: none), and press the keys it
	// presses there: existing actions only.
	screen screenID
	press  []string
}

const noScreen screenID = -1

// guideSteps is the checklist for f: 6 required steps, then the optional ones.
func guideSteps(f guideFacts) []guideStep {
	var steps []guideStep
	add := func(s guideStep) { steps = append(steps, s) }

	srv := guideStep{id: "server", title: "Server running", done: f.connected, screen: scrServer,
		why: "The dashboard, the CLI and every gate talk to this one server."}
	srv.how = "Running."
	if !f.connected {
		srv.how, srv.press = "Server: press s to start it.", []string{"s"}
	}
	add(srv)

	state := ""
	if f.vault != nil {
		state = f.vault.State
	}
	v := guideStep{id: "vault", title: "Vault created and unlocked", done: state == "unlocked", screen: noScreen,
		why: "It holds the controller key and the tokens, encrypted with your passphrase."}
	switch state {
	case "unlocked":
		v.how = "Unlocked."
	case "locked":
		v.how, v.press = "Press U and type the passphrase.", []string{"U"}
	case "none":
		v.how = "Create it in a terminal: jumpgate vault init."
	case "unusable":
		v.how = "Its file cannot be used; restore a backup: jumpgate vault restore FILE."
	default:
		v.how = "Not read yet: connect to the server first."
	}
	add(v)

	pass := 0
	if f.vault != nil {
		for _, s := range f.vault.Slots {
			if s == "passphrase" {
				pass++
			}
		}
	}
	r := guideStep{id: "recovery", title: "Recovery code added", done: pass >= 2, screen: noScreen,
		why: "If the passphrase is lost, the recovery code is the only way back into the vault."}
	switch {
	case r.done:
		r.how = "Added."
	case f.vault == nil || state == "none":
		r.how = "Create the vault first."
	default:
		r.how = "In a terminal: jumpgate vault add-recovery; the code prints once."
	}
	add(r)

	k := guideStep{id: "key", title: "Controller key present", screen: scrSigners,
		why: "It signs every command; gates run only what this key signed."}
	if c := f.ctl; c != nil {
		k.done = c.State == "ok" || (c.State == "locked" && c.Recorded != "")
		switch {
		case k.done:
			k.how = "Present."
		case c.State == "missing":
			k.how, k.press = "Signers: press K to create it.", []string{"K"}
		default:
			k.how = "Signers: the key is " + sanitizeLine(c.State) + "; see the reason there."
		}
	} else {
		k.how = "Not read yet: connect to the server first."
	}
	add(k)

	var rows []api.FleetRow
	if f.fleet != nil {
		rows = f.fleet.Rows
	}
	b := guideStep{id: "box", title: "First box added", done: len(rows) > 0, screen: scrFleet,
		why: "A box is a machine that runs your node and its gate."}
	b.how = "Added."
	if !b.done {
		b.how, b.press = "Fleet, hosts view: press a to add one over SSH, or i for a line to run on the box.", []string{"v", "a"}
	}
	add(b)

	pairedN, healthy := 0, false
	for _, row := range rows {
		if row.Link == api.LinkAgent || row.Link == api.LinkAgentLocal || row.Agent != "" {
			pairedN++
			if row.Reachable && row.Error == nil && row.AlertCount == 0 {
				healthy = true
			}
		}
	}
	p := guideStep{id: "paired", title: "Box paired and healthy", done: healthy, screen: scrFleet,
		why: "The gate must answer, signed, before jumpgate can run anything on the box."}
	switch {
	case healthy:
		p.how = "Paired and healthy."
	case len(rows) == 0:
		p.how = "Add a box first."
	case pairedN == 0:
		p.how, p.press = "Fleet, hosts view: select the box and press p to pair it.", []string{"v"}
	default:
		p.how = "Fleet: see what is wrong with the box."
	}
	add(p)

	sealed := false
	if f.vault != nil {
		for _, s := range f.vault.Slots {
			sealed = sealed || s == "systemd-creds" || s == "keystore"
		}
	}
	s := guideStep{id: "seal", title: "Vault sealed for auto-unlock", done: sealed, optional: true, screen: noScreen,
		why: "The service can then unlock the vault on boot without you."}
	switch {
	case sealed:
		s.how = "Sealed."
	case f.goos == "linux":
		s.how = "In a terminal: jumpgate vault seal."
	default:
		s.how = "On the Linux service: jumpgate vault seal."
	}
	add(s)

	add(guideStep{id: "second", title: "A second box", done: len(rows) >= 2, optional: true, screen: scrFleet,
		why: "Two boxes keep a node up while one is down or upgrading.", how: map[bool]string{true: "Added.", false: "Fleet, hosts view: press a."}[len(rows) >= 2]})

	on := false
	for _, m := range f.mods {
		for _, x := range m.Modules {
			on = on || x.Active
		}
	}
	m := guideStep{id: "module", title: "A module turned on", done: on, optional: true, screen: scrFleet,
		why: "Modules add services to a box, such as a bootnode or a message board."}
	switch {
	case on:
		m.how = "On."
	case f.modsKnown:
		m.how = "Open a box, its Modules tab, and press space on a module."
	default:
		m.how = "Not checked yet: expand a box in the tree."
	}
	add(m)

	g := guideStep{id: "gateway", title: "A gateway", done: f.gateways > 0, optional: true, screen: scrGateways,
		why: "A gateway serves your node's RPC to apps and customers."}
	switch {
	case g.done:
		g.how = "Set up."
	case f.gateways < 0:
		g.how = "Not read yet."
	default:
		g.how = "Gateways: add one in the web app (jumpgate open)."
	}
	add(g)
	return steps
}

// guideProgress counts the required steps.
func guideProgress(steps []guideStep) (done, total int) {
	for _, s := range steps {
		if !s.optional {
			total++
			if s.done {
				done++
			}
		}
	}
	return done, total
}

// nextStep is the first required step not done, then the first optional
// one; -1 when all are done.
func nextStep(steps []guideStep) int {
	for _, opt := range []bool{false, true} {
		for i, s := range steps {
			if s.optional == opt && !s.done {
				return i
			}
		}
	}
	return -1
}

// guideMsg is the vault and controller as the guide reads them.
type guideMsg struct {
	v      api.VaultView
	vErr   error
	ctl    api.ControllerView
	ctlErr error
}

// loadGuide reads the vault and the controller for the guide.
func loadGuide(a *App) tea.Cmd {
	return a.do("guide", func(ctx context.Context, be Backend) tea.Msg {
		v, vErr := be.Vault(ctx)
		c, cErr := be.Controller(ctx)
		return guideMsg{v: v, vErr: vErr, ctl: c, ctlErr: cErr}
	})
}

// gotGuide keeps what was read; a failed read leaves the fact unknown.
func (a *App) gotGuide(m guideMsg) {
	if m.vErr == nil {
		a.vault, a.vaultHas = m.v, true
	}
	if m.ctlErr == nil {
		a.ctl, a.ctlHas = m.ctl, true
	}
	a.guideKnown = a.guideKnown || m.vErr == nil || m.ctlErr == nil
	a.maybeGuide()
}

// facts are the guide's facts from the App as it is now.
func (a *App) facts() guideFacts {
	f := guideFacts{connected: a.conn == apiclient.Live, goos: a.o.GOOS, gateways: -1}
	if a.vaultHas {
		v := a.vault
		f.vault = &v
	}
	if a.ctlHas {
		c := a.ctl
		f.ctl = &c
	}
	if a.fleetHas {
		fl := a.fleet
		f.fleet = &fl
	}
	for _, m := range a.mods {
		if m.has {
			f.mods, f.modsKnown = append(f.mods, m.view), true
		}
	}
	if a.gwsHas {
		f.gateways = len(a.gws)
	}
	return f
}

// setupSegment is the status strip's "setup N/6", until every required
// step is done; "" before the guide's facts are read.
func (a *App) setupSegment() string {
	if !a.guideKnown {
		return ""
	}
	done, total := guideProgress(guideSteps(a.facts()))
	if done >= total {
		return ""
	}
	return fmt.Sprintf("setup %d/%d", done, total)
}

// maybeGuide opens the guide by itself once per session, on a first run:
// no box yet and the vault not unlocked.
func (a *App) maybeGuide() {
	if a.guideShown || !a.vaultHas || !a.fleetHas || a.modal != nil || len(a.fleet.Rows) > 0 || a.vault.State == "unlocked" || a.vault.State == "" {
		return
	}
	a.guideShown = true
	a.modal = newGuide(a)
}

// guideModal is the checklist; enter runs the selected step's action.
type guideModal struct {
	cursor int
	moved  bool // the person picked a step; until then the cursor follows the next one
}

func newGuide(a *App) *guideModal {
	return &guideModal{cursor: max(nextStep(guideSteps(a.facts())), 0)}
}

func (g *guideModal) key(a *App, k tea.KeyPressMsg) (bool, tea.Cmd) {
	steps := guideSteps(a.facts())
	switch k.String() {
	case "esc", "g", "q":
		return true, nil
	case "up", "k":
		g.cursor, g.moved = max(g.cursor-1, 0), true
	case "down", "j":
		g.cursor, g.moved = min(g.cursor+1, len(steps)-1), true
	case "enter":
		s := steps[min(g.cursor, len(steps)-1)]
		a.modal = nil
		return true, a.runStep(s)
	}
	return false, nil
}

// stepKeysMsg presses a guide step's keys once its view has loaded.
type stepKeysMsg []string

// runStep opens the step's view and, after the view's first reading, presses
// the step's keys there.
func (a *App) runStep(s guideStep) tea.Cmd {
	var open tea.Cmd
	if s.screen != noScreen {
		open = a.openScreen(s.screen)
		a.focus = focusMain
	}
	if s.done || len(s.press) == 0 {
		return open
	}
	keys := stepKeysMsg(s.press)
	return tea.Sequence(open, func() tea.Msg { return keys })
}

// stepKeys presses the keys; v is skipped when the hosts view is open
// already (it would close it).
func (a *App) stepKeys(keys stepKeysMsg) tea.Cmd {
	var cmds []tea.Cmd
	for _, k := range keys {
		if fs, ok := a.screens[scrFleet].(*fleetScreen); ok && k == "v" && fs.hostMode && a.active == scrFleet {
			continue
		}
		cmds = append(cmds, a.key(keyPress(k)))
	}
	return tea.Batch(cmds...)
}

// keyPress is the key press of one key name.
func keyPress(k string) tea.KeyPressMsg {
	r := []rune(k)
	return tea.KeyPressMsg{Code: r[0], Text: k}
}

func (g *guideModal) view(a *App, w, h int) string {
	steps := guideSteps(a.facts())
	next := nextStep(steps)
	if !g.moved && next >= 0 {
		g.cursor = next
	}
	g.cursor = min(g.cursor, len(steps)-1)
	var lines []string
	n := 0
	for i, s := range steps {
		if s.optional && (i == 0 || !steps[i-1].optional) {
			lines = append(lines, "", "  optional:")
		}
		n++
		mark := a.th.Warn.Render("todo")
		if s.done {
			mark = a.th.OK.Render("done")
		}
		l := fmt.Sprintf("%s %d. %s", mark, n, s.title)
		if i == next {
			l += a.th.Dim.Render("   " + a.gl.Sep + " next")
		}
		if i == g.cursor {
			l = a.gl.Sel + " " + l
		} else {
			l = "  " + l
		}
		lines = append(lines, truncate(l, w, a.gl.Ellipsis))
	}
	s := steps[g.cursor]
	lines = append(lines, "", truncate("  why: "+s.why, w, a.gl.Ellipsis), truncate("  do:  "+s.how, w, a.gl.Ellipsis), "")
	foot := "  [esc] close"
	if s.screen != noScreen || len(s.press) > 0 {
		foot = "  [enter] go there  [esc] close"
	}
	lines = append(lines, a.th.Dim.Render(foot))
	return strings.Join(lines, "\n")
}
