package tui

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/apiclient"
)

type screenID int

const (
	scrFleet screenID = iota
	scrHosts
	scrSigners
	scrGateways
	scrJobs
	scrSettings
)

var screenNames = [...]string{"Fleet", "Hosts", "Signers", "Gateways", "Jobs", "Settings"}

// screen is one view in the main area.
type screen interface {
	update(a *App, msg tea.Msg) tea.Cmd
	view(a *App, w, h int) string
	keys() []key.Binding
	capturing() bool // a text field has the keyboard: global keys stay off
}

// modal takes the keyboard until it is done (help, confirmation, palette).
type modal interface {
	key(a *App, k tea.KeyPressMsg) (done bool, cmd tea.Cmd)
	view(a *App, w, h int) string
}

// pasteTaker is a modal with a text field. Terminals deliver a paste as one
// message rather than key presses, so without this a pasted word would never
// reach the field.
type pasteTaker interface {
	paste(a *App, p tea.PasteMsg) tea.Cmd
}

// Messages shared by every screen.
type (
	enterMsg struct{} // the screen just became visible
	flashMsg string   // a one-line notice for the status bar
	errMsg   struct {
		what string
		err  error
	}
	tickMsg  time.Time
	prefsMsg struct {
		p   api.UIPrefs
		err error
	}
	fleetMsg struct {
		u  apiclient.Update[api.Fleet]
		ch <-chan apiclient.Update[api.Fleet]
	}
)

const (
	minW = 80
	minH = 24
)

// App is the root model.
type App struct {
	o      Options
	be     Backend
	ctx    context.Context
	cancel context.CancelFunc
	now    func() time.Time

	w, h  int
	th    Theme
	gl    Glyphs
	prefs api.UIPrefs

	fleet    api.Fleet
	fleetHas bool
	conn     apiclient.ConnState
	connErr  error

	active  screenID
	screens [6]screen
	detail  screen // the open host, if any (Task 15)
	modal   modal
	flash   string
	skew    string
	sel     string // the box the status bar names
}

// New builds the root model.
func New(o Options) *App {
	if o.Command == nil {
		o.Command = exec.Command
	}
	if o.Getenv == nil {
		o.Getenv = os.Getenv
	}
	if o.GOOS == "" {
		o.GOOS = runtime.GOOS
	}
	if o.Now == nil {
		o.Now = time.Now
	}
	ctx, cancel := context.WithCancel(context.Background())
	a := &App{o: o, be: o.Backend, ctx: ctx, cancel: cancel, now: o.Now, th: NewTheme(), prefs: api.DefaultPrefs(), conn: apiclient.Connecting}
	a.gl = DetectGlyphs(a.prefs.Glyphs, o.GOOS, o.Getenv)
	for i := range a.screens {
		a.screens[i] = comingScreen{name: screenNames[i]}
	}
	a.screens[scrJobs] = jobsScreen{}
	if server, mine, differs := a.be.Skew(); differs {
		server, mine = sanitizeLine(server), sanitizeLine(mine)
		// Offer R only where it works; otherwise the banner still says why
		// some requests may fail.
		a.skew = fmt.Sprintf("server %s, this jumpgate %s: the versions differ", server, mine)
		if o.Restart != nil {
			a.skew = fmt.Sprintf("server %s, this jumpgate %s: press R to restart the server", server, mine)
		}
	}
	return a
}

// Init starts the fleet watch, the clock and a prefs fetch. The fetch does
// not block the first frame: the TUI shows the defaults until prefsMsg lands.
func (a *App) Init() tea.Cmd {
	return tea.Batch(a.watchFleet(a.be.WatchFleet(a.ctx)), tick(), a.fetchPrefs())
}

func (a *App) fetchPrefs() tea.Cmd {
	return a.do("prefs", func(ctx context.Context) tea.Msg {
		p, err := a.be.Prefs(ctx)
		return prefsMsg{p: p, err: err}
	})
}

func tick() tea.Cmd { return tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg(t) }) }

// watchFleet waits for the next fleet update; the handler asks again.
func (a *App) watchFleet(ch <-chan apiclient.Update[api.Fleet]) tea.Cmd {
	return func() tea.Msg {
		u, ok := <-ch
		if !ok {
			return nil
		}
		return fleetMsg{u: u, ch: ch}
	}
}

// do runs f off the update loop with the app's context as it is now: a
// restart that replaces a.ctx cancels this one, so work started for the old
// server stops, and the command goroutine never reads a field the update
// loop writes.
func (a *App) do(what string, f func(ctx context.Context) tea.Msg) tea.Cmd {
	ctx := a.ctx
	return func() tea.Msg { return f(ctx) }
}

// errText is an error in one line, with the server's hint when it sent one,
// sanitized: error text quotes servers, boxes and their output.
func (a *App) errText(err error) string {
	var e *api.Error
	if errors.As(err, &e) {
		if e.Hint != "" {
			return sanitizeLine(e.Message) + " " + a.gl.Dash + " " + sanitizeLine(e.Hint)
		}
		return sanitizeLine(e.Message)
	}
	return sanitizeLine(err.Error())
}

func (a *App) setPrefs(p api.UIPrefs) {
	a.prefs = p.WithDefaults()
	a.gl = DetectGlyphs(a.prefs.Glyphs, a.o.GOOS, a.o.Getenv)
}

// current is the screen with the keyboard: the open host, or the active one.
func (a *App) current() screen {
	if a.detail != nil {
		return a.detail
	}
	return a.screens[a.active]
}

// openScreen shows a top-level screen and tells it so.
func (a *App) openScreen(id screenID) tea.Cmd {
	a.detail, a.active = nil, id
	return a.screens[id].update(a, enterMsg{})
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.w, a.h = msg.Width, msg.Height
		return a, nil
	case tickMsg:
		return a, tick()
	case fleetMsg:
		a.conn, a.connErr = msg.u.State, msg.u.Err
		if msg.u.Has {
			a.fleet, a.fleetHas = msg.u.Value, true
		}
		return a, tea.Batch(a.watchFleet(msg.ch), a.broadcast(msg))
	case prefsMsg:
		if msg.err != nil {
			a.flash = "could not read display settings: " + a.errText(msg.err)
		} else {
			a.setPrefs(msg.p)
		}
		return a, a.broadcast(msg)
	case flashMsg:
		a.flash = sanitizeLine(string(msg))
		return a, nil
	case errMsg:
		a.flash = sanitizeLine(msg.what) + ": " + a.errText(msg.err)
		return a, nil
	case restartedMsg:
		if msg.err != nil {
			a.flash = "restart failed: " + a.errText(msg.err)
			return a, nil
		}
		a.cancel()
		a.ctx, a.cancel = context.WithCancel(context.Background())
		a.be, a.skew, a.flash = msg.be, "", "the server restarted"
		return a, tea.Batch(a.watchFleet(a.be.WatchFleet(a.ctx)), a.openScreen(a.active))
	case tea.KeyPressMsg:
		a.flash = ""
		return a, a.key(msg)
	case tea.PasteMsg:
		return a, a.paste(msg)
	}
	return a, a.broadcast(msg)
}

// broadcast hands a message to every screen and the open host: each takes
// only its own (host messages carry the host id), so a reply that lands after
// the person moved on is dropped by the screen it was for.
func (a *App) broadcast(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, 0, len(a.screens)+1)
	for _, s := range a.screens {
		cmds = append(cmds, s.update(a, msg))
	}
	if a.detail != nil {
		cmds = append(cmds, a.detail.update(a, msg))
	}
	return tea.Batch(cmds...)
}

func (a *App) key(k tea.KeyPressMsg) tea.Cmd {
	if key.Matches(k, globalKeys.ForceQuit) {
		a.cancel()
		return tea.Quit
	}
	if a.modal != nil {
		done, cmd := a.modal.key(a, k)
		if done {
			a.modal = nil
		}
		return cmd
	}
	cur := a.current()
	if cur.capturing() {
		return cur.update(a, k)
	}
	switch {
	case key.Matches(k, globalKeys.Help):
		a.modal = newHelp(a)
		return nil
	case key.Matches(k, globalKeys.Screens):
		return a.openScreen(screenID(k.Code - '1'))
	case key.Matches(k, globalKeys.Quit) && a.detail == nil:
		a.cancel()
		return tea.Quit
	case key.Matches(k, globalKeys.Restart) && a.canRestart():
		return a.restart()
	}
	return cur.update(a, k)
}

// paste hands a paste to whatever has a text field: the modal, or a screen
// that is capturing the keyboard. Anywhere else it is dropped, so pasted
// text never fires single-key commands.
func (a *App) paste(p tea.PasteMsg) tea.Cmd {
	if a.modal != nil {
		if pt, ok := a.modal.(pasteTaker); ok {
			return pt.paste(a, p)
		}
		return nil
	}
	if cur := a.current(); cur.capturing() {
		return cur.update(a, p)
	}
	return nil
}

type restartedMsg struct {
	be  Backend
	err error
}

// canRestart: R restarts only a server whose version differs, and only when
// cmd/jumpgate gave the TUI a way to do it.
func (a *App) canRestart() bool { return a.skew != "" && a.o.Restart != nil }

// restart replaces the server on the person's request (spec D30).
func (a *App) restart() tea.Cmd {
	a.flash = "restarting the server" + a.gl.Ellipsis
	ctx, restart := a.ctx, a.o.Restart // captured, as in do
	return func() tea.Msg {
		be, err := restart(ctx)
		return restartedMsg{be: be, err: err}
	}
}

func (a *App) View() tea.View {
	v := tea.NewView(a.render())
	v.AltScreen = true
	if a.prefs.Mouse {
		v.MouseMode = tea.MouseModeCellMotion
	}
	return v
}

func (a *App) render() string {
	if a.w <= 0 || a.h <= 0 {
		return ""
	}
	if a.w < minW || a.h < minH {
		return a.tooSmall()
	}
	bodyH := a.h - 2
	var body string
	if a.skew != "" {
		body = a.th.Warn.Render(truncate(a.skew, a.w, a.gl.Ellipsis)) + "\n"
		bodyH--
	}
	if a.modal != nil {
		body += fit(a.modal.view(a, a.w, bodyH), a.w, bodyH, a.gl.Ellipsis)
	} else {
		body += fit(a.current().view(a, a.w, bodyH), a.w, bodyH, a.gl.Ellipsis)
	}
	return lipgloss.JoinVertical(lipgloss.Left, a.topBar(), body, a.statusBar())
}

// tooSmall is the whole frame below 80×24 (spec "Layout"): the size message,
// wrapped at words to the window and cut to its height, so even a tiny
// window shows as much of it as fits without the terminal wrapping it. The
// state underneath is kept and comes back on the next resize.
func (a *App) tooSmall() string {
	msg := fmt.Sprintf("terminal too small (%dx%d); jumpgate needs %dx%d", a.w, a.h, minW, minH)
	return fit(ansi.Wrap(msg, a.w, ""), a.w, a.h, "")
}

// topBar names the screens; the active one is bracketed, so it reads without
// colour too.
func (a *App) topBar() string {
	parts := []string{a.th.Title.Render("jumpgate")}
	for i, n := range screenNames {
		label := fmt.Sprintf("%d %s", i+1, n)
		if screenID(i) == a.active && a.detail == nil {
			label = a.th.Sel.Render("[" + label + "]")
		}
		parts = append(parts, label)
	}
	left := " " + strings.Join(parts, "  ")
	right := "? help"
	gap := a.w - lipgloss.Width(left) - lipgloss.Width(right)
	return truncate(left+strings.Repeat(" ", max(gap, 1))+right, a.w, a.gl.Ellipsis)
}

// statusBar: the selected box and its link, the server connection, the jobs
// flag (always – until sub-project 2), and the age of the fleet data.
func (a *App) statusBar() string {
	if a.flash != "" {
		return truncate(" "+a.flash, a.w, a.gl.Ellipsis)
	}
	box := "no box selected"
	if a.sel != "" {
		box = sanitizeLine(a.sel)
		if row, ok := a.fleet.Row(a.sel); ok {
			box += " " + a.gl.Sep + " " + linkWord(row)
		}
	}
	server := "server " + a.conn.String()
	if a.conn == apiclient.Retrying && a.connErr != nil {
		server += " (" + a.errText(a.connErr) + ")"
	}
	left := fmt.Sprintf(" %s %s %s %s jobs %s %s", box, a.gl.Sep, server, a.gl.Sep, a.gl.Flag, a.gl.Dash)
	right := ""
	if a.fleetHas {
		right = formatAge(a.now().Sub(a.fleet.At)) + " ago "
	}
	gap := a.w - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		return truncate(left, a.w, a.gl.Ellipsis)
	}
	return left + strings.Repeat(" ", gap) + right
}

// linkWord is a row's link in one word: HOSTKEY when its host key is the
// problem, down when it answered once and does not now.
func linkWord(r api.FleetRow) string {
	word := map[api.Link]string{api.LinkAgent: "agent", api.LinkAgentLocal: "local", api.LinkSSHOnly: "ssh", api.LinkLocalOnly: "local*"}[r.Link]
	if r.Error != nil && (r.Error.Code == api.CodeHostKey || r.Error.Code == api.CodeUnknownHost) {
		return "HOSTKEY"
	}
	if !r.Reachable && r.LastSeen != nil {
		return "down"
	}
	return word
}

// comingScreen holds a screen's place until its task replaces it.
type comingScreen struct{ name string }

func (c comingScreen) update(*App, tea.Msg) tea.Cmd { return nil }
func (c comingScreen) view(a *App, w, h int) string { return a.th.Dim.Render(" " + c.name) }
func (c comingScreen) keys() []key.Binding          { return nil }
func (c comingScreen) capturing() bool              { return false }
