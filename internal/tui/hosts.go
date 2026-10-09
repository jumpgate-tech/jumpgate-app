package tui

import (
	"context"
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/valve-tech/jumpgate/internal/api"
)

var hostsKeys = struct{ Add, Pair, Local, Remove, Reload key.Binding }{
	Add:    key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "add a box")),
	Pair:   key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "pair or re-pair")),
	Local:  key.NewBinding(key.WithKeys("L"), key.WithHelp("L", "pair this machine")),
	Remove: key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "remove, typed confirmation")),
	Reload: key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "reload")),
}

type (
	targetsMsg struct {
		ts  []api.TargetView
		err error
	}
	localPairDoneMsg struct{ err error }
	removedMsg       struct {
		id  string
		err error
	}
)

func loadTargets(a *App) tea.Cmd {
	return a.do("hosts", func(ctx context.Context, be Backend) tea.Msg {
		ts, err := be.Targets(ctx)
		return targetsMsg{ts: ts, err: err}
	})
}

// hostsScreen lists every box and adds, pairs and removes them (spec screen 3).
type hostsScreen struct {
	ts     []api.TargetView
	has    bool
	err    error
	cursor int
	flow   *addFlow
	visit  // counts flows: a reply for an earlier flow is dropped
}

func (s *hostsScreen) capturing() bool { return s.flow != nil }

func (s *hostsScreen) keys() []key.Binding {
	return []key.Binding{navKeys.Up, navKeys.Down, navKeys.Enter, hostsKeys.Add, hostsKeys.Pair, hostsKeys.Local, hostsKeys.Remove, hostsKeys.Reload}
}

// startFlow opens a new add or pair conversation, ending any earlier one.
func (s *hostsScreen) startFlow(a *App, f *addFlow) {
	s.endFlow()
	s.visit.begin(a.ctx)
	f.visit = s.visit
	s.flow = f
}

// endFlow stops the flow's streams and drops it.
func (s *hostsScreen) endFlow() {
	if s.flow != nil {
		s.flow.cancel()
		s.flow = nil
	}
}

func (s *hostsScreen) update(a *App, msg tea.Msg) tea.Cmd {
	if s.flow != nil {
		switch msg.(type) {
		case leaveMsg:
			// Another screen took over (or a restart re-opens this one): the
			// flow ends now, not on return, and the person is told what it
			// left behind in the words esc uses. A host key on screen is
			// cancelled with nothing trusted.
			if note := s.flow.leaveNote(a); note != "" {
				if a.flash != "" {
					note = a.flash + "; " + note
				}
				a.flash = note
			}
			s.endFlow()
			return nil
		case enterMsg:
			s.endFlow() // nothing should be left by leaveMsg; start clean regardless
		default:
			cmd, done := s.flow.update(a, msg)
			if done {
				s.endFlow()
				return tea.Batch(cmd, loadTargets(a))
			}
			// While the flow is open it owns the keyboard: a key it did not
			// use must not reach the list behind it.
			switch msg.(type) {
			case tea.KeyPressMsg, tea.PasteMsg:
				return cmd
			}
			if cmd != nil {
				return cmd
			}
		}
	}
	switch msg := msg.(type) {
	case enterMsg:
		return loadTargets(a)
	case targetsMsg:
		s.ts, s.has, s.err = msg.ts, msg.err == nil, msg.err
		s.cursor = min(s.cursor, max(len(s.ts)-1, 0))
	case localPairDoneMsg:
		if msg.err != nil {
			a.flash = "pairing this machine did not finish: " + a.errText(msg.err)
		} else {
			a.flash = "back from pairing this machine"
		}
		return loadTargets(a)
	case removedMsg:
		id := sanitizeLine(msg.id)
		if msg.err != nil {
			a.flash = "remove " + id + ": " + a.errText(msg.err)
		} else {
			a.flash = "forgot " + id + " on this controller; the box still lists this controller until it is revoked there"
		}
		return loadTargets(a)
	case tea.KeyPressMsg:
		if a.active != scrHosts || a.detail != nil {
			return nil
		}
		return s.key(a, msg)
	}
	return nil
}

func (s *hostsScreen) key(a *App, k tea.KeyPressMsg) tea.Cmd {
	var sel *api.TargetView
	if s.cursor < len(s.ts) {
		sel = &s.ts[s.cursor]
	}
	switch {
	case key.Matches(k, navKeys.Up):
		s.cursor = max(s.cursor-1, 0)
	case key.Matches(k, navKeys.Down):
		s.cursor = min(s.cursor+1, max(len(s.ts)-1, 0))
	case key.Matches(k, navKeys.Enter) && sel != nil:
		return a.openHost(sel.ID)
	case key.Matches(k, hostsKeys.Reload):
		return loadTargets(a)
	case key.Matches(k, hostsKeys.Add):
		f := newAddFlow()
		s.startFlow(a, f)
		return f.focusCmd()
	case key.Matches(k, hostsKeys.Local):
		return s.pairThisMachine(a, localName(a.o.Hostname))
	case key.Matches(k, hostsKeys.Pair) && sel != nil:
		if sel.Mode == "local" {
			if !nameRE.MatchString(sel.ID) {
				a.flash = "this name cannot be passed to the jumpgate CLI from here; run `jumpgate hosts add NAME --local`"
				return nil
			}
			return s.pairThisMachine(a, sel.ID)
		}
		if sel.SSH == nil {
			a.flash = "this box has no SSH address to pair over"
			return nil
		}
		f := newRepairFlow(sel.ID, *sel.SSH)
		s.startFlow(a, f)
		return f.probe(a)
	case key.Matches(k, hostsKeys.Remove) && sel != nil:
		confirmRemove(a, sel.ID)
	}
	return nil
}

// confirmRemove asks for the box's name before forgetting it. The key and
// the palette both come through here.
func confirmRemove(a *App, id string) {
	a.modal = newConfirm("remove "+id,
		"This controller forgets "+id+" and its pairing. The box keeps running, and keeps this controller enrolled until it is revoked there.",
		id, func() tea.Cmd {
			return a.do("remove", func(ctx context.Context, be Backend) tea.Msg {
				return removedMsg{id: id, err: be.RemoveTarget(ctx, id)}
			})
		})
}

// localName is the box name for this machine: the host name cut down to what
// a target name may be, or "local". It becomes a CLI argument, so it is never
// passed on as found.
func localName(host string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(host) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-' || r == '.' || r == '_':
			if b.Len() > 0 {
				b.WriteByte('-')
			}
		}
	}
	name := strings.Trim(b.String(), "-")
	if len(name) > 32 {
		name = strings.TrimRight(name[:32], "-")
	}
	if !nameRE.MatchString(name) {
		return "local"
	}
	return name
}

// pairThisMachine hands the terminal to `jumpgate hosts add NAME --local`
// (spec D17): pairing this machine as a non-root user needs sudo to prompt,
// and only a foreground process can. The agent is Linux-only. The command is
// argv, never a shell line, and name is already a checked target name.
func (s *hostsScreen) pairThisMachine(a *App, name string) tea.Cmd {
	if a.o.GOOS != "linux" {
		a.flash = "pairing this machine needs Linux: the jumpgate agent runs only on Linux. Pair a Linux box with a."
		return nil
	}
	if a.o.Self == "" {
		a.flash = "this jumpgate cannot find its own executable to pair this machine; run `jumpgate hosts add NAME --local`"
		return nil
	}
	return tea.ExecProcess(a.o.Command(a.o.Self, "hosts", "add", name, "--local"), func(err error) tea.Msg { return localPairDoneMsg{err: err} })
}

func describe(t api.TargetView) string {
	switch {
	case t.ThisMachine:
		return "this machine"
	case t.SSH != nil:
		return t.SSH.Address()
	}
	return "?"
}

func (s *hostsScreen) view(a *App, w, h int) string {
	if s.flow != nil {
		return s.flow.view(a, w, h)
	}
	head := " " + a.th.Title.Render("HOSTS")
	switch {
	case s.err != nil:
		return head + "\n\n " + a.th.Bad.Render("unavailable: ") + a.errText(s.err)
	case !s.has:
		return head + "\n\n " + a.th.Dim.Render("loading"+a.gl.Ellipsis)
	}
	lines := []string{head, ""}
	for i, t := range s.ts {
		mark := " "
		if i == s.cursor {
			mark = a.gl.Sel
		}
		pairing := a.th.Warn.Render("not paired")
		if t.Agent != nil {
			pairing = a.th.OK.Render("agent " + shortAddr(sanitizeLine(t.Agent.Address), a.gl)) // the date is on Signers
		}
		id, addr := sanitizeLine(t.ID), sanitizeLine(describe(t))
		lines = append(lines, fmt.Sprintf("%s %-14s %-38s %s", mark, truncate(id, 14, a.gl.Ellipsis), truncate(addr, 38, a.gl.Ellipsis), pairing))
	}
	if len(s.ts) == 0 {
		lines = append(lines, " "+a.th.Dim.Render("no boxes yet: press a to add one"))
	}
	for len(lines) < h-1 {
		lines = append(lines, "")
	}
	foot := " a add  p pair  L pair this machine  d remove  enter open"
	return strings.Join(append(lines, a.th.Dim.Render(foot)), "\n")
}
