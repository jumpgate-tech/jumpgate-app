package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/valve-tech/jumpgate/internal/api"
)

var signerKeys = struct{ Test, Create, Restart, Reload key.Binding }{
	Test:    key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "test signing on the selected box")),
	Create:  key.NewBinding(key.WithKeys("K"), key.WithHelp("K", "create this controller's key")),
	Restart: key.NewBinding(key.WithKeys("R"), key.WithHelp("R", "restart the server (loads a new key)")),
	Reload:  key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "reload")),
}

// Replies carry the screen's gen: one that started before the screen was
// left, re-entered or reloaded is dropped.
type (
	controllerMsg struct {
		gen uint64
		v   api.ControllerView
		err error
	}
	checkMsg struct {
		gen uint64
		id  string
		c   api.AgentCheck
		err error
	}
	keysInitDoneMsg struct{ err error }
	// leaveMsg tells the screen that was showing that another one took over.
	leaveMsg struct{}
)

// signersScreen is which key signs, for which boxes, and a signed round trip
// to prove it (spec screen 4). Only the routine tier exists yet (spec D32).
// It never holds key material: only what ControllerView exposes.
type signersScreen struct {
	ctl    api.ControllerView
	ctlHas bool
	ctlErr error
	ts     []api.TargetView
	cursor int
	checks map[string]string

	canRestart bool // Options.Restart is set; synced from the App on every update and view

	visit // gen counts loads (see controllerMsg); ctx ends commands still running for this visit
}

func (s *signersScreen) capturing() bool { return false }

func (s *signersScreen) keys() []key.Binding {
	ks := []key.Binding{navKeys.Up, navKeys.Down, signerKeys.Test, signerKeys.Create}
	if s.canRestart {
		ks = append(ks, signerKeys.Restart)
	}
	return append(ks, signerKeys.Reload)
}

// begin starts a visit or a reload: earlier commands are cancelled and their
// replies, if any still arrive, are dropped.
func (s *signersScreen) begin(a *App) {
	s.visit.begin(a.ctx)
	s.checks = nil
}

func (s *signersScreen) load(a *App) tea.Cmd {
	s.begin(a)
	gen, ctx := s.gen, s.ctx
	be := a.be
	return tea.Batch(loadTargets(a), func() tea.Msg {
		v, err := be.Controller(ctx)
		return controllerMsg{gen: gen, v: v, err: err}
	})
}

// paired are the boxes this controller signs for.
func (s *signersScreen) paired() []api.TargetView {
	var out []api.TargetView
	for _, t := range s.ts {
		if t.Agent != nil {
			out = append(out, t)
		}
	}
	return out
}

// canCreate: a key may be made only when there is none. In every other state
// a key exists (maybe at a non-default ref), and keys init would make a
// second one that the paired boxes refuse.
func (s *signersScreen) canCreate() bool { return s.ctlHas && s.ctl.State == "missing" }

func (s *signersScreen) update(a *App, msg tea.Msg) tea.Cmd {
	s.canRestart = a.o.Restart != nil
	switch msg := msg.(type) {
	case enterMsg:
		return s.load(a)
	case leaveMsg:
		s.invalidate() // replies still in flight are for a visit that is over
	case controllerMsg:
		if !s.current(msg.gen) {
			return nil
		}
		s.ctl, s.ctlHas, s.ctlErr = msg.v, msg.err == nil, msg.err
	case targetsMsg:
		if msg.err == nil {
			s.ts = msg.ts
			s.cursor = min(s.cursor, max(len(s.paired())-1, 0))
		}
	case checkMsg:
		if !s.current(msg.gen) {
			return nil
		}
		if msg.err != nil {
			s.checks[msg.id] = a.th.Bad.Render("failed: ") + a.errText(msg.err)
		} else {
			s.checks[msg.id] = a.th.OK.Render(fmt.Sprintf("signed and verified in %dms", msg.c.ElapsedMs)) + fmt.Sprintf(", agent %s", sanitizeLine(msg.c.Version))
		}
	case keysInitDoneMsg:
		if msg.err != nil {
			a.flash = "keys init did not finish: " + a.errText(msg.err)
		} else {
			a.flash = "key created; press R to restart the server so it loads the key"
			if a.o.Restart == nil {
				a.flash = "key created; restart the server (jumpgate stop) so it loads the key"
			}
		}
		if a.active == scrSigners {
			return s.load(a)
		}
	case tea.KeyPressMsg:
		if a.active != scrSigners || a.detail != nil {
			return nil
		}
		boxes := s.paired()
		switch {
		case key.Matches(msg, navKeys.Up):
			s.cursor = max(s.cursor-1, 0)
		case key.Matches(msg, navKeys.Down):
			s.cursor = min(s.cursor+1, max(len(boxes)-1, 0))
		case key.Matches(msg, signerKeys.Reload):
			return s.load(a)
		case key.Matches(msg, signerKeys.Test) && s.cursor < len(boxes):
			id := boxes[s.cursor].ID
			if s.ctx == nil {
				return nil
			}
			if s.checks == nil {
				s.checks = map[string]string{}
			}
			s.checks[id] = a.th.Dim.Render("testing" + a.gl.Ellipsis)
			gen, ctx, be := s.gen, s.ctx, a.be
			return func() tea.Msg {
				c, err := be.CheckAgent(ctx, id)
				return checkMsg{gen: gen, id: id, c: c, err: err}
			}
		case key.Matches(msg, signerKeys.Create):
			if !s.canCreate() {
				a.flash = "this controller already has a key; jumpgate never replaces one"
				if !s.ctlHas {
					a.flash = "the controller's key state is not known yet"
				} else if s.ctl.State == "unrecorded" {
					a.flash = "a key exists but is not recorded; keys init would make a different key. Restore config.json from a backup"
				}
				return nil
			}
			if a.o.Self == "" {
				a.flash = "run `jumpgate keys init` in a terminal"
				return nil
			}
			// The key store may prompt (keychain, 1Password): only a
			// foreground process can show that (spec D17). The argv is fixed.
			return tea.ExecProcess(a.o.Command(a.o.Self, "keys", "init"), func(err error) tea.Msg { return keysInitDoneMsg{err: err} })
		case key.Matches(msg, signerKeys.Restart) && a.o.Restart != nil:
			return a.restart()
		}
	}
	return nil
}

func (s *signersScreen) view(a *App, w, h int) string {
	s.canRestart = a.o.Restart != nil
	lines := []string{" " + a.th.Title.Render("THIS CONTROLLER")}
	switch {
	case s.ctlErr != nil:
		lines = append(lines, " "+a.th.Bad.Render("unavailable: ")+a.errText(s.ctlErr))
	case !s.ctlHas:
		lines = append(lines, " "+a.th.Dim.Render("loading"+a.gl.Ellipsis))
	case s.ctl.State == "missing":
		lines = append(lines, " "+a.th.Warn.Render("no key yet")+": press K to create one (jumpgate keys init)")
	default:
		addr := sanitizeLine(s.ctl.Address)
		if addr == "" {
			addr = sanitizeLine(s.ctl.Recorded)
		}
		var state, hint string
		switch s.ctl.State {
		case "ok":
			state = a.th.OK.Render("ok")
		case "mismatch":
			state = a.th.Bad.Render("MISMATCH")
			hint = "the key store holds a key other than the recorded one (" + shortAddr(sanitizeLine(s.ctl.Recorded), a.gl) + ")"
		case "unopened":
			state = a.th.Warn.Render("not loaded")
			hint = "the key exists but the server has not opened it"
			if a.o.Restart != nil {
				hint += "; unlock the store, then press R"
			}
		case "unrecorded":
			state = a.th.Warn.Render("not recorded")
			hint = "A signing key is loaded but config.json has no controller record. Restore config.json from a backup. Running keys init would create a different key that your paired boxes won't accept."
		default:
			state = a.th.Warn.Render(sanitizeLine(s.ctl.State))
		}
		lines = append(lines, fmt.Sprintf(" address  %s   store %s   %s", shortAddr(addr, a.gl), sanitizeLine(s.ctl.Store), state))
		if r := sanitizeLine(s.ctl.Reason); r != "" {
			lines = append(lines, " "+a.th.Warn.Render(r))
		}
		if hint != "" {
			for _, l := range strings.Split(ansi.Wrap(hint, max(w-2, 20), ""), "\n") {
				lines = append(lines, " "+a.th.Dim.Render(l))
			}
		}
	}
	lines = append(lines, "", " "+a.th.Title.Render("BOXES THIS KEY SIGNS FOR"))
	boxes := s.paired()
	if len(boxes) == 0 {
		lines = append(lines, " "+a.th.Dim.Render("none paired yet (Hosts, p)"))
	}
	for i, t := range boxes {
		mark := " "
		if i == s.cursor {
			mark = a.gl.Sel
		}
		lines = append(lines, fmt.Sprintf("%s %-12s agent %s  %-5s  paired %s  routine", mark, truncate(sanitizeLine(t.ID), 12, a.gl.Ellipsis),
			shortAddr(sanitizeLine(t.Agent.Address), a.gl), truncate(sanitizeLine(t.Agent.Transport), 5, a.gl.Ellipsis), t.Agent.PairedAt.In(a.loc).Format("2006-01-02")))
		if c, ok := s.checks[t.ID]; ok {
			lines = append(lines, "    "+c)
		}
	}
	lines = append(lines, "", " "+a.th.Dim.Render("Approval keys (browser and hardware wallets) arrive with the approval tier."))
	for len(lines) < h-1 {
		lines = append(lines, "")
	}
	bar := " t test signing  K create a key  "
	if a.o.Restart != nil {
		bar += "R restart the server  "
	}
	return strings.Join(append(lines, a.th.Dim.Render(bar+"r reload")), "\n")
}
