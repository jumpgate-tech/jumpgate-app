package tui

import (
	"context"
	"errors"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
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
)

// hostScreen is one box's detail. Its streams live in ctx, cancelled when
// the person leaves the box.
type hostScreen struct {
	id     string
	gen    uint64
	tab    int
	ctx    context.Context
	cancel context.CancelFunc
	loaded map[string]bool

	status     api.NodeStatus
	statusHas  bool
	statusConn apiclient.ConnState
	statusErr  error

	disk      api.DiskView
	diskHas   bool
	diskErr   error
	measuring bool
}

// openHost shows a box's detail, closing any open one.
func (a *App) openHost(id string) tea.Cmd {
	a.closeDetail()
	ctx, cancel := context.WithCancel(a.ctx)
	a.hostGen++
	h := &hostScreen{id: id, gen: a.hostGen, ctx: ctx, cancel: cancel, loaded: map[string]bool{}}
	a.detail, a.sel = h, id
	return tea.Batch(h.watchStatus(a.be.WatchStatus(ctx, id)), h.load(a, h.tabName(a)))
}

// closeDetail leaves the open box and stops its streams.
func (a *App) closeDetail() {
	if h, ok := a.detail.(*hostScreen); ok {
		h.cancel()
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
	}
	return nil
}

func (h *hostScreen) capturing() bool { return false }

func (h *hostScreen) keys() []key.Binding {
	return []key.Binding{navKeys.Left, navKeys.Right, globalKeys.Back, hostKeys.Measure, hostKeys.Sidebar}
}

func (h *hostScreen) update(a *App, msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case statusMsg:
		if msg.id != h.id || msg.gen != h.gen {
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
		if msg.id != h.id || msg.gen != h.gen {
			return nil
		}
		h.measuring = false
		if msg.err != nil {
			h.diskErr = msg.err
		} else {
			h.disk, h.diskHas, h.diskErr = msg.d, true, nil
		}
		return nil
	case tea.KeyPressMsg:
		if cmd, ok := h.tabKey(a, msg); ok {
			return cmd
		}
		switch {
		case key.Matches(msg, globalKeys.Back):
			a.closeDetail()
		case key.Matches(msg, navKeys.Left):
			h.tab--
			return h.load(a, h.tabName(a))
		case key.Matches(msg, navKeys.Right):
			h.tab++
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
	case "storage":
		if key.Matches(k, hostKeys.Measure) {
			if h.measuring {
				return nil, true // already measuring; the server joins it anyway
			}
			h.measuring = true
			id, gen, ctx, be := h.id, h.gen, h.ctx, a.be
			return func() tea.Msg {
				d, err := be.MeasureDisk(ctx, id)
				return diskMsg{id: id, gen: gen, d: d, err: err}
			}, true
		}
	}
	return nil, false
}

// tabUpdate hands other messages to the tabs that keep their own state
// (later tasks: logs, services).
func (h *hostScreen) tabUpdate(a *App, msg tea.Msg) tea.Cmd { return nil }

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
