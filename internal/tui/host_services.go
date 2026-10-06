package tui

import (
	"fmt"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/valve-tech/jumpgate/internal/api"
)

var services = []string{"exec", "beacon"}

var svcKeys = struct{ Start, Stop, Restart, Shell, Actions key.Binding }{
	Start:   key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "start (Services)")),
	Stop:    key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "stop, typed confirmation (Services)")),
	Restart: key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "restart, typed confirmation (Services)")),
	Shell:   key.NewBinding(key.WithKeys("S"), key.WithHelp("S", "SSH shell on this box")),
	Actions: key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "actions for this box")),
}

// Replies carry the open's gen as well as the box id, so a late reply from an
// earlier open of the same box is dropped.
type (
	serviceMsg struct {
		id, svc, action string
		gen             uint64
		res             api.ServiceResult
		err             error
	}
	sshCmdMsg struct {
		id  string
		gen uint64
		c   api.SSHCommand
		err error
	}
	shellDoneMsg struct {
		id  string
		gen uint64
		err error
	}
)

// serviceAction starts a service at once; stop and restart interrupt a
// running node, so they always need the service's name typed (spec D21).
// Every key path that can stop or restart goes through here.
func (h *hostScreen) serviceAction(a *App, svc, action string) tea.Cmd {
	id, gen, ctx, be := h.id, h.gen, h.ctx, a.be
	run := func() tea.Cmd {
		a.flash = fmt.Sprintf("%s %s on %s%s", action, sanitizeLine(svc), sanitizeLine(id), a.gl.Ellipsis)
		return func() tea.Msg {
			res, err := be.ServiceAction(ctx, id, svc, action)
			return serviceMsg{id: id, gen: gen, svc: svc, action: action, res: res, err: err}
		}
	}
	if action == "start" {
		return run()
	}
	a.modal = newConfirm(
		fmt.Sprintf("%s %s on %s", action, svc, id),
		fmt.Sprintf("The %s client on %s stops serving until it is running again.", svc, id),
		svc, run)
	return nil
}

// shell asks the server for the ssh command (it knows the address, key and
// the confirmed host keys) and runs it in the foreground.
func (h *hostScreen) shell(a *App) tea.Cmd {
	id, gen, ctx, be := h.id, h.gen, h.ctx, a.be
	a.flash = "preparing the ssh command" + a.gl.Ellipsis
	return func() tea.Msg {
		c, err := be.SSHCommand(ctx, id)
		return sshCmdMsg{id: id, gen: gen, c: c, err: err}
	}
}

// runShell screens the server's argv and hands the terminal to the child.
// Display is never run (and not shown): only Argv, directly, never a shell.
// The program is this machine's own ssh by name, whatever path the server
// wrote.
func (h *hostScreen) runShell(a *App, msg sshCmdMsg) tea.Cmd {
	if msg.err != nil {
		a.flash = "ssh: " + a.errText(msg.err)
		return nil
	}
	if err := api.CheckSSHArgv(msg.c.Argv); err != nil {
		a.flash = "refusing the server's ssh command: " + sanitizeLine(err.Error())
		return nil
	}
	name := "ssh"
	if a.o.GOOS == "windows" {
		name = "ssh.exe"
	}
	id, gen := msg.id, msg.gen
	cmd := a.o.Command(name, msg.c.Argv[1:]...)
	a.flash = ""
	return tea.ExecProcess(cmd, func(err error) tea.Msg { return shellDoneMsg{id: id, gen: gen, err: err} })
}

func (h *hostScreen) servicesKey(a *App, k tea.KeyPressMsg) (tea.Cmd, bool) {
	svc := services[h.svcCursor]
	switch {
	case key.Matches(k, navKeys.Up):
		h.svcCursor = max(h.svcCursor-1, 0)
	case key.Matches(k, navKeys.Down):
		h.svcCursor = min(h.svcCursor+1, len(services)-1)
	case key.Matches(k, svcKeys.Start):
		return h.serviceAction(a, svc, "start"), true
	case key.Matches(k, svcKeys.Stop):
		return h.serviceAction(a, svc, "stop"), true
	case key.Matches(k, svcKeys.Restart):
		return h.serviceAction(a, svc, "restart"), true
	default:
		return nil, false
	}
	return nil, true
}

func (h *hostScreen) viewServices(a *App, w int) string {
	out := "\n " + a.th.Title.Render("SERVICES") + "\n"
	for i, svc := range services {
		state := a.th.Dim.Render("unknown")
		if h.statusHas {
			cs := h.status.Exec
			if svc == "beacon" {
				cs = h.status.Beacon
			}
			state = a.th.Bad.Render("inactive")
			if cs.Active {
				state = a.th.OK.Render("active  ")
			}
		}
		mark, hint := " ", ""
		if i == h.svcCursor {
			mark, hint = a.gl.Sel, "   s start  t stop  r restart"
		}
		out += fmt.Sprintf(" %s %-7s %s%s\n", mark, sanitizeLine(svc), state, hint)
	}
	out += "\n S  open an SSH shell on this box (system ssh, confirmed host keys only)\n"
	out += " x  every action for this box\n"
	if h.svcLast != "" {
		out += "\n last: " + h.svcLast + "\n"
	}
	return out
}
