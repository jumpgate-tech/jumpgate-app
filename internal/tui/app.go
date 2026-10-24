package tui

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/apiclient"
)

type screenID int

const (
	scrFleet screenID = iota // the table, and the box list (Hosts folded in, spec U3)
	scrJobs
	scrSigners
	scrGateways
	scrSettings
)

var screenNames = [...]string{"Fleet", "Jobs", "Signers", "Gateways", "Settings"}

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
	tickMsg   time.Time
	gwTickMsg struct{} // time to re-read the gateways
	prefsMsg  struct {
		p   api.UIPrefs
		err error
		// seq, with checked, marks the reply to a save (or to the re-read
		// after a refused one): it counts only if no later save was sent.
		seq     uint64
		checked bool
		refetch bool // this is the re-read
	}
	fleetMsg struct {
		u  apiclient.Update[api.Fleet]
		ch <-chan apiclient.Update[api.Fleet]
	}
	runningMsg struct {
		v   api.RunningJobs
		err error
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
	loc    *time.Location // the zone log times are shown in

	w, h     int
	th       Theme
	gl       Glyphs
	prefs    api.UIPrefs
	prefsSeq uint64 // counts prefs saves; see prefsMsg.seq

	fleet    api.Fleet
	fleetHas bool
	// running is the last GET /api/jobs/running (runningHas: one arrived;
	// runningErr: the latest read failed, so its bars are stale), read after
	// each fleet reading for the drawer's job bars on every view. It is the
	// server's cache: reading it asks no box. runningBusy: a read is out.
	running     api.RunningJobs
	runningHas  bool
	runningErr  error
	runningBusy bool
	// needs is the fleet-wide count of jobs that need a decision at the last
	// reading, and needsKnown says there was one: the bell compares against it.
	needs      int
	needsKnown bool
	// gws is the last gateways list (gwsHas: one arrived), for the KPI strip
	// and the alerts; read on start and every 30 s from the server's store.
	gws      []api.GatewaySummary
	gwsHas   bool
	alertSrc alertSource // nil: derivedAlerts
	// The fleet tree: which boxes are expanded, their modules readings, and
	// the tree's cursor. a.sel is the one selection the tree and the table
	// share; selNode narrows it to one client, selAll widens it to the fleet.
	expanded map[string]bool
	mods     map[string]treeMods
	treeCur  int
	selNode  string
	selAll   bool
	conn     apiclient.ConnState
	connErr  error

	active  screenID
	screens [len(screenNames)]screen
	detail  screen // the open host, if any
	hostGen uint64 // counts openHost calls; see hostScreen.gen
	modal   modal
	flash   string
	skew    string
	srvVer  string // the server's version, when Skew reported one
	sel     string // the box the status bar names
	// focus is the region with the keyboard; treeOver and logOver are the
	// tree and the log drawn over main where the window has no room for them.
	focus    focusRegion
	treeOver bool
	logOver  bool
	logOff   int      // lines the log is scrolled down
	log      actLog   // the activity log (spec U6), this session only
	prev     snapshot // what the log last compared against
	// restarting: a restart is under way. A second one would race the first
	// over which server is current, so R does nothing until it lands.
	restarting bool
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
	if o.Location == nil {
		o.Location = time.Local
	}
	ctx, cancel := context.WithCancel(context.Background())
	a := &App{o: o, loc: o.Location, be: o.Backend, ctx: ctx, cancel: cancel, now: o.Now, th: NewTheme(), prefs: api.DefaultPrefs(), conn: apiclient.Connecting}
	a.gl = DetectGlyphs(a.prefs.Glyphs, o.GOOS, o.Getenv)
	for i := range a.screens {
		a.screens[i] = comingScreen{name: screenNames[i]}
	}
	a.screens[scrFleet] = newFleetScreen()
	a.screens[scrJobs] = &jobsScreen{}
	a.screens[scrGateways] = &gatewaysScreen{}
	a.screens[scrSigners] = &signersScreen{}
	a.screens[scrSettings] = &settingsScreen{}
	if server, mine, differs := a.be.Skew(); differs {
		server, mine = sanitizeLine(server), sanitizeLine(mine)
		a.srvVer = server
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
	return tea.Batch(a.watchFleet(a.be.WatchFleet(a.ctx)), tick(), a.fetchPrefs(), loadGateways(a), gwTick(), treeTick(), loadTargets(a))
}

// gatewaysEvery is how often the KPI strip re-reads the gateways: the
// server's own store, so no box is polled for it.
const gatewaysEvery = 30 * time.Second

func gwTick() tea.Cmd { return tea.Tick(gatewaysEvery, func(time.Time) tea.Msg { return gwTickMsg{} }) }

func (a *App) fetchPrefs() tea.Cmd {
	return a.do("prefs", func(ctx context.Context, be Backend) tea.Msg {
		p, err := be.Prefs(ctx)
		return prefsMsg{p: p, err: err}
	})
}

// refetchPrefs reads the server's prefs after a refused save.
func (a *App) refetchPrefs() tea.Cmd {
	seq := a.prefsSeq
	return a.do("prefs", func(ctx context.Context, be Backend) tea.Msg {
		p, err := be.Prefs(ctx)
		return prefsMsg{p: p, err: err, seq: seq, checked: true, refetch: true}
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

// readRunning reads the running jobs once, unless a read is already out.
func (a *App) readRunning() tea.Cmd {
	if a.runningBusy {
		return nil
	}
	a.runningBusy = true
	return a.do("running", func(ctx context.Context, be Backend) tea.Msg {
		v, err := be.RunningJobs(ctx)
		return runningMsg{v: v, err: err}
	})
}

// do runs f off the update loop with the app's context and backend as they
// are now: a restart that replaces a.ctx cancels this one, so work started
// for the old server stops, and the command goroutine never reads a field
// the update loop writes (a.be among them). f must use the be it is given.
func (a *App) do(what string, f func(ctx context.Context, be Backend) tea.Msg) tea.Cmd {
	ctx, be := a.ctx, a.be
	return func() tea.Msg { return f(ctx, be) }
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
	a.closeDetail()
	a.screens[a.active].update(a, leaveMsg{}) // its in-flight commands end; any cmd it returns is dropped
	a.active = id
	return a.screens[id].update(a, enterMsg{})
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	m, cmd := a.update(msg)
	switch msg := msg.(type) {
	case fleetMsg, gatewaysMsg, targetsMsg, treeModsMsg:
		a.record()
	case jobsMsg:
		if msg.u.Has {
			a.record()
		}
	}
	return m, cmd
}

func (a *App) update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.w, a.h = msg.Width, msg.Height
		a.fitFocus()
		return a, nil
	case tickMsg:
		return a, tick()
	case gwTickMsg:
		return a, tea.Batch(loadGateways(a), gwTick())
	case treeModsMsg:
		a.gotTreeMods(msg)
		return a, nil
	case treeTickMsg:
		return a, tea.Batch(a.treeRefresh(), treeTick())
	case gatewaysMsg:
		if msg.err == nil { // an error keeps the last list
			a.gws, a.gwsHas = msg.gws, true
		}
		return a, a.broadcast(msg)
	case fleetMsg:
		a.conn, a.connErr = msg.u.State, msg.u.Err
		var ring tea.Cmd
		if msg.u.Has {
			a.fleet, a.fleetHas = msg.u.Value, true
			n := a.needCount()
			// Ring when more jobs need a person than before; never for the
			// first reading, which would ring at every start-up.
			if a.needsKnown && n > a.needs && a.prefs.Bell && a.o.Bell != nil {
				bell := a.o.Bell
				ring = func() tea.Msg { bell(); return nil }
			}
			a.needs, a.needsKnown = n, true
		}
		return a, tea.Batch(a.watchFleet(msg.ch), a.broadcast(msg), ring, a.readRunning())
	case runningMsg:
		a.runningBusy, a.runningErr = false, msg.err
		if msg.err == nil {
			a.running, a.runningHas = msg.v, true
		}
		return a, nil
	case openJobsMsg:
		a.flash = sanitizeLine(msg.flash)
		return a, a.openScreen(scrJobs)
	case prefsMsg:
		if msg.checked && msg.seq != a.prefsSeq {
			return a, nil // a newer save was sent since: this reply is about an older state
		}
		switch {
		case msg.err != nil && msg.checked:
			// Refused (or the re-read failed): say why, and ask the server what it
			// really has rather than going back to a local snapshot.
			a.flash = "could not save the display setting: " + a.errText(msg.err)
			if !msg.refetch {
				return a, a.refetchPrefs()
			}
			return a, nil
		case msg.err != nil:
			a.flash = "could not read display settings: " + a.errText(msg.err)
		default:
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
		a.restarting = false
		if msg.err != nil {
			a.flash = "restart failed: " + a.errText(msg.err)
			return a, nil
		}
		a.cancel()
		a.ctx, a.cancel = context.WithCancel(context.Background())
		a.be, a.skew, a.srvVer, a.flash = msg.be, "", "", "the server restarted"
		return a, tea.Batch(a.watchFleet(a.be.WatchFleet(a.ctx)), a.openScreen(a.active), loadGateways(a))
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
	if cmd, ok := a.regionKey(k); ok {
		return cmd
	}
	switch {
	case key.Matches(k, globalKeys.Help):
		a.modal = newHelp(a)
		return nil
	case key.Matches(k, globalKeys.Palette):
		a.modal = newPalette()
		return nil
	case key.Matches(k, globalKeys.Drawer):
		return a.toggleDrawer()
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
	if a.restarting {
		a.flash = "already restarting the server" + a.gl.Ellipsis
		return nil
	}
	a.restarting = true
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
	return a.frame()
}

// tooSmall is the whole frame below 80×24 (spec "Layout"): the size message,
// wrapped at words to the window and cut to its height, so even a tiny
// window shows as much of it as fits without the terminal wrapping it. The
// state underneath is kept and comes back on the next resize.
func (a *App) tooSmall() string {
	msg := fmt.Sprintf("terminal too small (%dx%d); jumpgate needs %dx%d", a.w, a.h, minW, minH)
	return fit(ansi.Wrap(msg, a.w, ""), a.w, a.h, "")
}

// needCount is the fleet-wide number of jobs that need a decision.
func (a *App) needCount() int {
	n := 0
	for _, r := range a.fleet.Rows {
		if r.Jobs != nil {
			n += r.Jobs.NeedsDecision
		}
	}
	return n
}

// linkWord is a row's link in one word: HOSTKEY when its host key is the
// problem, down when it answered once and does not now.
func linkWord(r api.FleetRow) string {
	word := map[api.Link]string{api.LinkAgent: "gate", api.LinkAgentLocal: "local", api.LinkSSHOnly: "ssh", api.LinkLocalOnly: "local*"}[r.Link]
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
