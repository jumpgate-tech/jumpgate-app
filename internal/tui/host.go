package tui

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/apiclient"
)

var hostKeys = struct{ Measure, Sidebar key.Binding }{
	Measure: key.NewBinding(key.WithKeys("m"), key.WithHelp("m", "measure disk now (Storage)")),
	Sidebar: key.NewBinding(key.WithKeys("["), key.WithHelp("[", "host list")),
}

// statusStaleAfter: three missed 5 s readings.
const statusStaleAfter = 15 * time.Second

var tabTitles = map[string]string{
	"overview": "Overview", "storage": "Storage", "endpoints": "Endpoints",
	"security": "Security", "logs": "Logs", "services": "Services",
}

type (
	statusMsg struct {
		id  string
		gen uint64 // the open that asked: a reopened box drops the old one's replies
		u   apiclient.Update[api.NodeStatus]
		ch  <-chan apiclient.Update[api.NodeStatus]
	}
	diskMsg struct {
		id  string
		gen uint64
		d   api.DiskView
		err error
	}
	endpointsMsg struct {
		id  string
		gen uint64
		e   api.Endpoints
		err error
	}
	firewallMsg struct {
		id    string
		gen   uint64
		items []api.CheckItem
		err   error
	}
	// gatewaysMsg is not per box: the list is the same for every screen that
	// asks, and broadcast hands it to the host and the Gateways screen alike.
	gatewaysMsg struct {
		gws []api.GatewaySummary
		err error
	}
)

// hostScreen is one box's detail. Its streams live in ctx, cancelled when
// the person leaves the box.
type hostScreen struct {
	id     string
	visit  // gen is the open's number (App.hostGen); ctx holds the streams
	tab    int
	loaded map[string]bool

	status     api.NodeStatus
	statusHas  bool
	statusConn apiclient.ConnState
	statusErr  error

	disk      api.DiskView
	diskHas   bool
	diskErr   error
	measuring bool

	eps        api.Endpoints
	epsHas     bool
	epsErr     error
	gws        []api.GatewaySummary
	fw         []api.CheckItem
	fwHas      bool
	fwErr      error
	logs       []api.LogHit // sanitized on arrival, at most logKeep
	logsConn   apiclient.ConnState
	logsErr    error
	logsNote   string
	logFilter  textinput.Model
	logEditing bool
	logMin     int  // 0 all, 1 warn and above, 2 error and above
	follow     bool // keep the newest line in view
	logBack    int  // lines scrolled back from the newest

	svcCursor int
	svcLast   string

	scroll int // first visible line of the Endpoints or Security tab; clamped when drawn
}

// openHost shows a box's detail, closing any open one.
func (a *App) openHost(id string) tea.Cmd {
	a.closeDetail()
	a.hostGen++
	h := &hostScreen{id: id, loaded: map[string]bool{}, follow: true, logFilter: newLogFilter()}
	h.open(a.hostGen, a.ctx)
	a.detail, a.sel = h, id
	return tea.Batch(h.watchStatus(a.be.WatchStatus(h.ctx, id)), h.load(a, h.tabName(a)))
}

// closeDetail leaves the open box and stops its streams.
func (a *App) closeDetail() {
	if h, ok := a.detail.(*hostScreen); ok {
		h.cancel()
		switch m := a.modal.(type) { // a menu or confirmation for this box goes with it
		case *actionsModal:
			if m.h == h {
				a.modal = nil
			}
		case *confirmModal:
			a.modal = nil
		}
	}
	a.detail = nil
}

// watchStatus waits for the next status update; the handler asks again. It
// ends when the box is left, whether or not the backend closes the channel.
func (h *hostScreen) watchStatus(ch <-chan apiclient.Update[api.NodeStatus]) tea.Cmd {
	id, gen, ctx := h.id, h.gen, h.ctx
	return func() tea.Msg {
		select {
		case <-ctx.Done():
			return nil
		case u, ok := <-ch:
			if !ok {
				return nil
			}
			return statusMsg{id: id, gen: gen, u: u, ch: ch}
		}
	}
}

func (h *hostScreen) tabName(a *App) string {
	tabs := a.prefs.HostTabs
	if len(tabs) == 0 {
		return "overview"
	}
	return tabs[((h.tab%len(tabs))+len(tabs))%len(tabs)]
}

// load fetches what a tab shows, once; later tasks add their tabs here.
func (h *hostScreen) load(a *App, tab string) tea.Cmd {
	if h.loaded[tab] {
		return nil
	}
	h.loaded[tab] = true
	id, gen, ctx, be := h.id, h.gen, h.ctx, a.be // captured: the command goroutine reads no App field
	switch tab {
	case "storage":
		return func() tea.Msg {
			d, err := be.Disk(ctx, id)
			return diskMsg{id: id, gen: gen, d: d, err: err}
		}
	case "endpoints":
		return tea.Batch(func() tea.Msg {
			e, err := be.Endpoints(ctx, id)
			return endpointsMsg{id: id, gen: gen, e: e, err: err}
		}, loadGateways(a))
	case "logs":
		return h.watchLogs(be.WatchLogs(ctx, id, logBacklog))
	case "security":
		return func() tea.Msg {
			items, err := be.Firewall(ctx, id)
			return firewallMsg{id: id, gen: gen, items: items, err: err}
		}
	}
	return nil
}

func (h *hostScreen) capturing() bool { return h.logEditing }

func (h *hostScreen) keys() []key.Binding {
	return []key.Binding{navKeys.Left, navKeys.Right, globalKeys.Back, hostKeys.Measure, hostKeys.Sidebar, svcKeys.Shell, svcKeys.Actions}
}

func (h *hostScreen) update(a *App, msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case statusMsg:
		if msg.id != h.id || !h.current(msg.gen) {
			return nil
		}
		h.statusConn = msg.u.State
		switch {
		case msg.u.Has:
			h.status, h.statusHas, h.statusErr = msg.u.Value, true, nil
		case msg.u.Err != nil:
			h.statusErr = msg.u.Err
		case msg.u.State == apiclient.Live:
			h.statusErr = nil
		}
		return h.watchStatus(msg.ch)
	case diskMsg:
		if msg.id != h.id || !h.current(msg.gen) {
			return nil
		}
		h.measuring = false
		if msg.err != nil {
			h.diskErr = msg.err
		} else {
			h.disk, h.diskHas, h.diskErr = msg.d, true, nil
		}
		return nil
	case endpointsMsg:
		if msg.id != h.id || !h.current(msg.gen) {
			return nil
		}
		if msg.err != nil {
			h.epsErr = msg.err
		} else {
			h.eps, h.epsHas, h.epsErr = msg.e, true, nil
		}
		return nil
	case firewallMsg:
		if msg.id != h.id || !h.current(msg.gen) {
			return nil
		}
		if msg.err != nil {
			h.fwErr = msg.err
		} else {
			h.fw, h.fwHas, h.fwErr = msg.items, true, nil
		}
		return nil
	case logsMsg:
		if msg.id != h.id || !h.current(msg.gen) {
			return nil
		}
		h.applyLogs(msg.u)
		return h.watchLogs(msg.ch)
	case disclosureMsg:
		if msg.id != h.id || !h.current(msg.gen) {
			return nil
		}
		if m, ok := a.modal.(*explainModal); ok && m.h == h {
			m.gotDisclosure(msg)
		}
		return nil
	case explainMsg:
		if msg.id != h.id || !h.current(msg.gen) {
			return nil
		}
		if m, ok := a.modal.(*explainModal); ok && m.h == h {
			m.gotExplain(msg)
		}
		return nil
	case gatewaysMsg:
		if msg.err == nil {
			h.gws = msg.gws
		}
		return nil
	case serviceMsg:
		if msg.id != h.id || !h.current(msg.gen) {
			return nil
		}
		svc, id := sanitizeLine(msg.svc), sanitizeLine(msg.id)
		if msg.err != nil {
			h.svcLast = fmt.Sprintf("%s %s on %s failed: %s", sanitizeLine(msg.action), svc, id, a.errText(msg.err))
		} else {
			state := "inactive"
			if msg.res.Active {
				state = "active"
			}
			h.svcLast = fmt.Sprintf("%s %s on %s: %s", sanitizeLine(msg.action), svc, id, state)
		}
		a.flash = h.svcLast
		return nil
	case sshCmdMsg:
		if msg.id != h.id || !h.current(msg.gen) {
			return nil
		}
		return h.runShell(a, msg)
	case shellDoneMsg:
		if msg.id != h.id || !h.current(msg.gen) {
			return nil
		}
		if msg.err != nil {
			a.flash = "the ssh session ended: " + sanitizeLine(msg.err.Error())
		} else {
			a.flash = "back from " + sanitizeLine(msg.id)
		}
		return nil
	case tea.KeyPressMsg:
		if cmd, ok := h.tabKey(a, msg); ok {
			return cmd
		}
		switch {
		case key.Matches(msg, svcKeys.Shell):
			return h.shell(a)
		case key.Matches(msg, svcKeys.Actions):
			a.modal = newActions(a, h)
			return nil
		case key.Matches(msg, globalKeys.Back):
			a.closeDetail()
		case key.Matches(msg, navKeys.Left):
			h.tab--
			h.scroll = 0
			return h.load(a, h.tabName(a))
		case key.Matches(msg, navKeys.Right):
			h.tab++
			h.scroll = 0
			return h.load(a, h.tabName(a))
		case key.Matches(msg, hostKeys.Sidebar):
			a.sidebar = !a.sidebar
		}
	}
	return h.tabUpdate(a, msg)
}

// tabKey gives the visible tab first refusal of a key.
func (h *hostScreen) tabKey(a *App, k tea.KeyPressMsg) (tea.Cmd, bool) {
	switch h.tabName(a) {
	case "endpoints", "security":
		switch {
		case key.Matches(k, navKeys.Up):
			h.scroll = max(h.scroll-1, 0)
			return nil, true
		case key.Matches(k, navKeys.Down):
			h.scroll++ // clamped against the content when drawn
			return nil, true
		}
	case "logs":
		return h.logsKey(a, k)
	case "storage":
		if key.Matches(k, hostKeys.Measure) {
			return h.measure(a), true
		}
	case "services":
		return h.servicesKey(a, k)
	}
	return nil, false
}

// measure takes a disk reading now, unless one is under way.
func (h *hostScreen) measure(a *App) tea.Cmd {
	if h.measuring {
		return nil // already measuring; the server joins it anyway
	}
	h.measuring = true
	id, gen, ctx, be := h.id, h.gen, h.ctx, a.be
	return func() tea.Msg {
		d, err := be.MeasureDisk(ctx, id)
		return diskMsg{id: id, gen: gen, d: d, err: err}
	}
}

// tabUpdate hands other messages to the tabs that keep their own state
// (later tasks: logs, services).
func (h *hostScreen) tabUpdate(a *App, msg tea.Msg) tea.Cmd {
	if p, ok := msg.(tea.PasteMsg); ok && h.logEditing {
		// A paste is text, never escape sequences or line breaks.
		h.logFilter, _ = h.logFilter.Update(tea.PasteMsg{Content: sanitizeLine(p.Content)})
	}
	return nil
}

func (h *hostScreen) view(a *App, w, hgt int) string {
	head := " " + a.th.Title.Render(sanitizeLine(h.id))
	if row, ok := a.fleet.Row(h.id); ok {
		head += " " + a.gl.Sep + " " + linkWord(row)
		if row.Network != "" {
			head += " " + a.gl.Sep + " " + sanitizeLine(row.Network)
		}
	}
	var tabs []string
	cur := h.tabName(a)
	for _, t := range a.prefs.HostTabs {
		title, ok := tabTitles[t]
		if !ok {
			title = sanitizeLine(t) // a tab name from the server's prefs
		}
		if t == cur {
			title = a.th.Sel.Render("[" + title + "]")
		}
		tabs = append(tabs, title)
	}
	top := head + "\n " + strings.Join(tabs, "  ") + "\n"
	return top + h.body(a, w, hgt-2)
}

// body is the visible tab; later tasks add their cases.
func (h *hostScreen) body(a *App, w, hgt int) string {
	switch h.tabName(a) {
	case "overview":
		return h.viewOverview(a, w)
	case "storage":
		return h.viewStorage(a, w)
	case "endpoints":
		return h.viewEndpoints(a, w, hgt)
	case "security":
		return h.viewSecurity(a, w, hgt)
	case "logs":
		return h.viewLogs(a, w, hgt)
	case "services":
		return h.viewServices(a, w)
	}
	title, ok := tabTitles[h.tabName(a)]
	if !ok {
		title = sanitizeLine(h.tabName(a))
	}
	return a.th.Dim.Render(" " + title)
}

// statusErrText is a status stream's error in one line. A ninth window on a
// box is refused by the server; say so rather than showing a bare 429.
func (a *App) statusErrText(err error) string {
	var e *api.Error
	if errors.As(err, &e) && e.Code == api.CodeTooManyStreams {
		return "too many windows are watching this box; retrying slowly"
	}
	return a.errText(err)
}

// sidebarView lists the fleet beside an open host (spec D10).
func (a *App) sidebarView(h int) string {
	var lines []string
	for _, r := range a.fleet.Rows {
		name := truncate(sanitizeLine(r.ID), 16, a.gl.Ellipsis)
		if r.ID == a.sel {
			name = a.th.Sel.Render(a.gl.Sel + name)
		} else {
			name = " " + name
		}
		lines = append(lines, name)
	}
	return lipgloss.NewStyle().Width(18).Render(fit(strings.Join(lines, "\n"), 18, h, a.gl.Ellipsis))
}
