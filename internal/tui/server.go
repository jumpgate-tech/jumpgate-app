package tui

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/valve-tech/jumpgate/internal/apiclient"
)

// The Server view (inventory A, L): this server's version and the update
// check, the options it was built with, and starting, stopping and opening
// it; on Linux, installing and setting up the jumpgate service. The installs
// re-run this jumpgate in the foreground (runSelf), so sudo and the vault
// passphrase are asked on the real terminal and never pass through the TUI.

var serverKeys = struct{ Check, Start, Stop, Open, Install, Setup, SelfOnly, Reload key.Binding }{
	Reload:   key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "reload")),
	Check:    key.NewBinding(key.WithKeys("u"), key.WithHelp("u", "check for an update now")),
	Start:    key.NewBinding(key.WithKeys("ctrl+s"), key.WithHelp("ctrl+s", "start the server")),
	Stop:     key.NewBinding(key.WithKeys("ctrl+t"), key.WithHelp("ctrl+t", "stop the server, typed confirmation")),
	Open:     key.NewBinding(key.WithKeys("o"), key.WithHelp("o", "open the web app in the browser")),
	Install:  key.NewBinding(key.WithKeys("I"), key.WithHelp("I", "install the systemd service (sudo)")),
	Setup:    key.NewBinding(key.WithKeys("S"), key.WithHelp("S", "set up the service: vault key, unlock, pair this machine (sudo)")),
	SelfOnly: key.NewBinding(key.WithKeys("A"), key.WithHelp("A", "set up the service, self-only (sudo)")),
}

// updateView is GET /api/update.
type updateView struct {
	Current         string `json:"current"`
	Latest          string `json:"latest"`
	UpdateAvailable bool   `json:"updateAvailable"`
	ReleaseURL      string `json:"releaseUrl"`
	NotifyEnabled   bool   `json:"notifyEnabled"`
	CheckError      string `json:"checkError,omitempty"`
}

type (
	updateMsg struct {
		gen uint64
		v   updateView
		err error
	}
	stoppedMsg struct{ err error }
	startedMsg struct {
		be  Backend
		err error
	}
	serverWebMsg struct{ err error }
	// selfDoneMsg is a re-run jumpgate's exit.
	selfDoneMsg struct {
		args []string
		err  error
	}
)

type serverScreen struct {
	// What acts, synced from the App on every update and view: keys()
	// has no App.
	canStart, canOpen, linux bool
	live, system             bool // a server answers; it is the jumpgate service

	upd     updateView
	updHas  bool
	updErr  error
	updBusy bool
	visit
}

func (s *serverScreen) capturing() bool { return false }

// keys are the keys that act now, the next step first: start only while no
// server answers, stop only while one does, open only when cmd/jumpgate can,
// the install only until the service is the one running, the setup on Linux.
func (s *serverScreen) keys() []key.Binding {
	var ks []key.Binding
	if s.canStart {
		ks = append(ks, serverKeys.Start)
	}
	ks = append(ks, serverKeys.Reload, serverKeys.Check)
	if s.canOpen {
		ks = append(ks, serverKeys.Open)
	}
	if s.live {
		ks = append(ks, serverKeys.Stop)
	}
	if s.linux {
		if !s.system {
			ks = append(ks, serverKeys.Install)
		}
		ks = append(ks, serverKeys.Setup, serverKeys.SelfOnly)
	}
	return ks
}

func (s *serverScreen) sync(a *App) {
	s.canStart = a.o.Start != nil && a.conn != apiclient.Live
	s.canOpen = a.o.OpenWeb != nil
	s.linux = a.o.GOOS == "linux"
	s.live = a.conn == apiclient.Live
	s.system = false
	if a.o.Server != nil {
		s.system = a.o.Server().System
	}
}

func (s *serverScreen) loadUpdate(a *App, refresh bool) tea.Cmd {
	if s.ctx == nil {
		return nil
	}
	s.updBusy = true
	gen, ctx := s.gen, s.ctx
	path := "/api/update"
	if refresh {
		path += "?refresh=1"
	}
	return a.do("update", func(_ context.Context, be Backend) tea.Msg {
		var v updateView
		err := be.Do(ctx, http.MethodGet, path, nil, &v)
		return updateMsg{gen: gen, v: v, err: err}
	})
}

func (s *serverScreen) update(a *App, msg tea.Msg) tea.Cmd {
	s.sync(a)
	switch msg := msg.(type) {
	case enterMsg:
		s.begin(a.ctx)
		return s.loadUpdate(a, false)
	case leaveMsg:
		s.invalidate()
	case updateMsg:
		if !s.current(msg.gen) {
			return nil
		}
		s.updBusy, s.updErr = false, msg.err
		if msg.err == nil {
			s.upd, s.updHas = msg.v, true
		}
	case tea.KeyPressMsg:
		return s.key(a, msg)
	}
	return nil
}

func (s *serverScreen) key(a *App, k tea.KeyPressMsg) tea.Cmd {
	linux := a.o.GOOS == "linux"
	switch {
	case key.Matches(k, serverKeys.Reload):
		return s.loadUpdate(a, false)
	case key.Matches(k, serverKeys.Check):
		return s.loadUpdate(a, true)
	case key.Matches(k, serverKeys.Start):
		return a.startServer()
	case key.Matches(k, serverKeys.Stop):
		a.modal = newConfirm("Stop the jumpgate server",
			"Running jobs pause until it starts again; the vault locks.", "stop", func() tea.Cmd { return a.stopServer() })
	case key.Matches(k, serverKeys.Open) && a.o.OpenWeb != nil:
		open, ctx := a.o.OpenWeb, a.ctx
		a.flash = "opening the web app" + a.gl.Ellipsis
		return func() tea.Msg { return serverWebMsg{err: open(ctx)} }
	case key.Matches(k, serverKeys.Install) && linux:
		return a.runSelf("setup", "install-service")
	case key.Matches(k, serverKeys.Setup) && linux:
		return a.runSelf("setup", "server")
	case key.Matches(k, serverKeys.SelfOnly) && linux:
		return a.runSelf("setup", "server", "--self-only")
	}
	return nil
}

// runSelf re-runs this jumpgate with fixed args in the foreground: the
// terminal is its own until it exits, so sudo, a passphrase prompt or a
// printed one-time secret never pass through the TUI.
func (a *App) runSelf(args ...string) tea.Cmd {
	if a.o.Self == "" {
		a.flash = "run `jumpgate " + strings.Join(args, " ") + "` in a terminal"
		return nil
	}
	return tea.ExecProcess(a.o.Command(a.o.Self, args...), func(err error) tea.Msg { return selfDoneMsg{args: args, err: err} })
}

// selfDone says how the re-run ended and re-reads what it may have changed.
func (a *App) selfDone(m selfDoneMsg) tea.Cmd {
	what := "jumpgate " + strings.Join(m.args, " ")
	var ee *exec.ExitError
	switch {
	case m.err == nil:
		a.flash = what + ": done"
	case errors.As(m.err, &ee):
		a.flash = fmt.Sprintf("%s: exit %d", what, ee.ExitCode())
	default:
		a.setErr(what + ": " + a.errText(m.err))
	}
	return loadGuide(a)
}

// startServer starts a server when none answers and switches to it.
func (a *App) startServer() tea.Cmd {
	if a.o.Start == nil || a.conn == apiclient.Live {
		return nil
	}
	if a.restarting {
		a.flash = "already starting the server" + a.gl.Ellipsis
		return nil
	}
	a.restarting = true
	a.flash = "starting the server" + a.gl.Ellipsis
	ctx, start := a.ctx, a.o.Start
	return func() tea.Msg {
		be, err := start(ctx)
		return startedMsg{be: be, err: err}
	}
}

// stopServer asks the server to exit (POST /api/shutdown).
func (a *App) stopServer() tea.Cmd {
	return a.do("stop", func(ctx context.Context, be Backend) tea.Msg {
		return stoppedMsg{err: be.Do(ctx, http.MethodPost, "/api/shutdown", nil, nil)}
	})
}

func (s *serverScreen) view(a *App, w, h int) string {
	s.sync(a)
	row := func(label, v string) string { return truncate(fmt.Sprintf(" %-10s %s", label, v), w, a.gl.Ellipsis) }
	var lines []string
	_, mine, differs := a.be.Skew()
	ver := "this dashboard " + sanitizeLine(mine)
	if !differs {
		if server, _, _ := a.be.Skew(); server != "" {
			ver = sanitizeLine(server) + ", the same as this dashboard"
		}
	}
	lines = append(lines, row("version", ver))
	upd := a.th.Dim.Render("n/a")
	switch {
	case s.updErr != nil:
		upd = a.th.Bad.Render("FAIL") + " " + a.errText(s.updErr)
	case s.updBusy && !s.updHas:
		upd = a.th.Dim.Render("checking" + a.gl.Ellipsis)
	case s.updHas && s.upd.CheckError != "":
		upd = a.th.Warn.Render("could not check: ") + sanitizeLine(s.upd.CheckError)
	case s.updHas && s.upd.UpdateAvailable:
		upd = a.th.Warn.Render(sanitizeLine(s.upd.Latest)+" is out") + " " + maskURL(sanitizeLine(s.upd.ReleaseURL))
	case s.updHas && s.upd.Latest != "":
		upd = a.th.OK.Render("up to date") + " (latest " + sanitizeLine(s.upd.Latest) + ")"
	}
	if s.updBusy && s.updHas {
		upd += a.th.Dim.Render(" ~ checking" + a.gl.Ellipsis)
	}
	lines = append(lines, row("update", upd))

	var info *ServerInfo
	if a.o.Server != nil {
		i := a.o.Server()
		info = &i
	}
	if info == nil {
		lines = append(lines, row("process", a.th.Dim.Render("n/a")))
	} else {
		kind := "your own server"
		if info.System {
			kind = "the jumpgate service"
		}
		p := fmt.Sprintf("%s, pid %d", kind, info.PID)
		if !info.StartedAt.IsZero() {
			p += ", up " + formatAge(a.now().Sub(info.StartedAt))
		}
		lines = append(lines, row("process", p))
		lines = append(lines, row("web app", maskURL(sanitizeLine(info.HTTPAddr))))
		if sh := info.Shape; sh != nil {
			relay := "off"
			if sh.RelayBind != "" {
				relay = "on " + maskURL(sanitizeLine(sh.RelayBind))
			}
			lines = append(lines, row("relay", relay+", meter "+onOffWord(sh.Meter)+", key admin "+onOffWord(sh.KeyAdmin)))
		}
		for _, wl := range shapeWarnings(*info) {
			lines = append(lines, " "+a.th.Warn.Render(a.gl.Warn)+" "+truncate(wl, max(w-3, 1), a.gl.Ellipsis))
		}
	}

	lines = append(lines, "")
	// What can be done now, the next step first; the rest is under x.
	var prim, svc []key.Binding
	for _, b := range s.keys() {
		switch b.Help() {
		case serverKeys.Install.Help(), serverKeys.Setup.Help(), serverKeys.SelfOnly.Help():
			svc = append(svc, b)
		default:
			prim = append(prim, b)
		}
	}
	if s.canStart {
		// Nothing answers: starting it is the only next step.
		lines = append(lines, " "+a.th.Warn.Render("No server answers.")+"  "+a.buttons(prim[:1]))
	} else {
		lines = append(lines, " "+a.buttons(prim))
	}
	if a.o.GOOS == "linux" {
		if s.system {
			lines = append(lines, "", " "+a.th.Dim.Render("The jumpgate service is the server running now. [x] more: set up the service."))
		} else {
			lines = append(lines, "", " "+a.th.Title.Render("JUMPGATE SERVICE")+a.th.Dim.Render(" (asks for sudo on the terminal)"))
			for _, b := range svc {
				lines = append(lines, " ["+keyLabel(b.Help().Key, a.gl)+"] "+menuWords(b.Help().Desc))
			}
		}
	} else {
		lines = append(lines, "", a.th.Dim.Render(" The systemd service and its setup are Linux only."))
	}
	return strings.Join(lines, "\n")
}

// shapeWarnings are what is worth a word about how the server was started.
func shapeWarnings(i ServerInfo) []string {
	var out []string
	if host, _, err := net.SplitHostPort(i.HTTPAddr); err == nil && !loopback(host) {
		out = append(out, "the web app listens on "+maskURL(sanitizeLine(i.HTTPAddr))+", beyond this machine")
	}
	if i.Shape == nil {
		if !i.System {
			out = append(out, "the server does not report its options (it is older than this jumpgate)")
		}
		return out
	}
	if host, _, err := net.SplitHostPort(i.Shape.RelayBind); err == nil && !loopback(host) {
		out = append(out, "the metered RPC relay listens on "+maskURL(sanitizeLine(i.Shape.RelayBind))+", beyond this machine")
	}
	if i.Shape.Meter && i.Shape.BillingSocket == "" {
		out = append(out, "metering is on with no billing socket: counts are not reported")
	}
	return out
}

func loopback(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func onOffWord(b bool) string {
	if b {
		return "on"
	}
	return "off"
}
