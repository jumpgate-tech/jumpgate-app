package tui

// The box's plainest screens (inventory group E): containers, diagnostics,
// the monitor, the setup wizard, du, clearing a service's state and the
// generic intent. Every answer is shown as key: value lines (jsonLines); a
// destructive action needs the service's name typed, as the server checks.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"sort"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/valve-tech/jumpgate/internal/apiclient"
	"github.com/valve-tech/jumpgate/internal/intent"
)

var boxKeys = struct {
	Clear, DU, Provision, Wipe, Reset, Config, Edit, Diag, Latest, Monitor, Intent, Wizard key.Binding
}{
	Clear:     key.NewBinding(key.WithKeys("C"), key.WithHelp("C", "clear the service's state, typed confirmation (Services)")),
	DU:        key.NewBinding(key.WithKeys("u"), key.WithHelp("u", "disk usage by path (Storage)")),
	Provision: key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "provision the container (Docker)")),
	Wipe:      key.NewBinding(key.WithKeys("W"), key.WithHelp("W", "wipe the container, typed confirmation (Docker)")),
	Reset:     key.NewBinding(key.WithKeys("X"), key.WithHelp("X", "reset the devnet, typed confirmation (Docker)")),
	Config:    key.NewBinding(key.WithKeys("v"), key.WithHelp("v", "show the container's config (Docker)")),
	Edit:      key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "set the container's config as JSON (Docker)")),
	Diag:      key.NewBinding(key.WithKeys("D"), key.WithHelp("D", "run diagnostics (Tools)")),
	Latest:    key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "latest diagnostics (Tools)")),
	Monitor:   key.NewBinding(key.WithKeys("M"), key.WithHelp("M", "monitor on/off (Tools)")),
	Intent:    key.NewBinding(key.WithKeys("I"), key.WithHelp("I", "run an intent (Tools)")),
	Wizard:    key.NewBinding(key.WithKeys("w"), key.WithHelp("w", "node setup wizard (Tools)")),
}

// intentKinds are the kinds the generic route runs here (internal/intent/
// kinds.go): reads. service.action has its own keys, logs.since is the logs
// tab's, agent.upgrade is not an intent route, and job kinds are refused by
// the server.
var intentKinds = []string{
	intent.KindAgentInfo, intent.KindStatusRead, intent.KindDiskRead, intent.KindEndpointsRead,
	intent.KindFirewallRead, intent.KindLogsRead, intent.KindModulesRead,
}

// maxResultLines bounds what one answer keeps.
const maxResultLines = 400

type (
	// boxDoMsg is one Do's answer for a tab's result area.
	boxDoMsg struct {
		id, tab, head string
		gen           uint64
		v             json.RawMessage
		err           error
		then          func(h *hostScreen, a *App) tea.Cmd // after a success; nil: nothing
	}
	containersMsg struct {
		id  string
		gen uint64
		v   containersView
		err error
	}
	monitorMsg struct {
		id   string
		gen  uint64
		mgen int
		u    apiclient.Update[json.RawMessage]
		ch   <-chan apiclient.Update[json.RawMessage]
	}
)

// boxResult is what a tab shows under its own lines: the last answer.
type boxResult struct {
	head  string
	lines []string
	bad   bool
}

// containersView is GET /api/targets/{id}/containers, the fields shown.
type containersView struct {
	Docker struct {
		Present       bool   `json:"present"`
		Reachable     bool   `json:"reachable"`
		Flavor        string `json:"flavor"`
		ServerVersion string `json:"serverVersion"`
		Detail        string `json:"detail"`
		Hint          string `json:"hint"`
	} `json:"docker"`
	Services []containerView `json:"services"`
}

type containerView struct {
	ID         string `json:"id"`
	Configured bool   `json:"configured"`
	Status     struct {
		State string
		Image string
	} `json:"status"`
	Endpoints []struct {
		Label string `json:"label"`
		URL   string `json:"url"`
	} `json:"endpoints"`
	Actions        []string `json:"actions"`
	Blocked        string   `json:"blocked"`
	WipeDiscards   string   `json:"wipeDiscards"`
	RestartsOnWipe []string `json:"restartsOnWipe"`
	Warnings       []string `json:"warnings"`
	Error          string   `json:"error"`
}

func tpath(id, rest string) string { return "/api/targets/" + url.PathEscape(id) + rest }

// jsonLines is an answer as sorted key: value lines, every string sanitized
// and its URLs masked; null is n/a.
func jsonLines(raw json.RawMessage) []string {
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	var v any
	if err := d.Decode(&v); err != nil {
		return []string{"FAIL: the answer is not JSON"}
	}
	var out []string
	var walk func(prefix string, v any)
	walk = func(prefix string, v any) {
		if len(out) >= maxResultLines {
			return
		}
		switch x := v.(type) {
		case map[string]any:
			keys := make([]string, 0, len(x))
			for k := range x {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			if len(keys) == 0 {
				out = append(out, prefix+": {}")
			}
			for _, k := range keys {
				p := sanitizeLine(k)
				if prefix != "" {
					p = prefix + "." + p
				}
				walk(p, x[k])
			}
		case []any:
			if len(x) == 0 {
				out = append(out, prefix+": []")
			}
			for i, e := range x {
				walk(fmt.Sprintf("%s[%d]", prefix, i), e)
			}
		case nil:
			out = append(out, prefix+": n/a")
		case string:
			s := sanitizeLine(x)
			if strings.Contains(s, "//") || strings.Contains(s, "@") {
				s = maskText(s) // a URL or user@host; a time or a plain word is kept
			}
			out = append(out, prefix+": "+s)
		default:
			out = append(out, fmt.Sprintf("%s: %v", prefix, x))
		}
	}
	walk("", v)
	for i, l := range out {
		out[i] = strings.TrimPrefix(l, ": ")
	}
	return out
}

// boxCall sends one request and shows the answer on tab under head.
func (h *hostScreen) boxCall(a *App, tab, head, method, path string, in any, then func(*hostScreen, *App) tea.Cmd) tea.Cmd {
	id, gen, ctx, be := h.id, h.gen, h.ctx, a.be
	h.setResult(tab, boxResult{head: head + a.gl.Ellipsis})
	return func() tea.Msg {
		var v json.RawMessage
		err := be.Do(ctx, method, path, in, &v)
		return boxDoMsg{id: id, gen: gen, tab: tab, head: head, v: v, err: err, then: then}
	}
}

func (h *hostScreen) setResult(tab string, r boxResult) {
	if h.results == nil {
		h.results = map[string]boxResult{}
	}
	h.results[tab] = r
}

// boxUpdate takes this file's messages; ok is false for any other.
func (h *hostScreen) boxUpdate(a *App, msg tea.Msg) (tea.Cmd, bool) {
	switch msg := msg.(type) {
	case boxDoMsg:
		if msg.id != h.id || !h.current(msg.gen) {
			return nil, true
		}
		if msg.err != nil {
			h.setResult(msg.tab, boxResult{head: sanitizeLine(msg.head) + ": FAIL", lines: []string{a.errText(msg.err)}, bad: true})
			a.flash = sanitizeLine(msg.head) + " failed"
			return nil, true
		}
		h.setResult(msg.tab, boxResult{head: sanitizeLine(msg.head), lines: jsonLines(msg.v)})
		if msg.then != nil {
			return msg.then(h, a), true
		}
		return nil, true
	case containersMsg:
		if msg.id != h.id || !h.current(msg.gen) {
			return nil, true
		}
		if msg.err != nil {
			h.ctrErr = msg.err
		} else {
			h.ctr, h.ctrHas, h.ctrErr = msg.v, true, nil
			h.ctrCursor = min(h.ctrCursor, max(len(h.ctr.Services)-1, 0))
		}
		return nil, true
	case monitorMsg:
		if msg.id != h.id || !h.current(msg.gen) || msg.mgen != h.monGen || h.monStop == nil {
			return nil, true
		}
		h.monConn = msg.u.State
		switch {
		case msg.u.Has:
			h.monLines, h.monErr = jsonLines(msg.u.Value), nil
		case msg.u.Err != nil:
			h.monErr = msg.u.Err
		}
		return h.nextMonitor(msg.mgen, msg.ch), true
	}
	return nil, false
}

func (h *hostScreen) loadContainers(a *App) tea.Cmd {
	id, gen, ctx, be := h.id, h.gen, h.ctx, a.be
	return func() tea.Msg {
		var v containersView
		err := be.Do(ctx, http.MethodGet, tpath(id, "/containers"), nil, &v)
		return containersMsg{id: id, gen: gen, v: v, err: err}
	}
}

func reloadContainers(h *hostScreen, a *App) tea.Cmd { return h.loadContainers(a) }

// typedThen opens the typed confirmation for name; run is the request.
func typedThen(a *App, title, detail, name string, run func() tea.Cmd) {
	a.modal = newConfirm(title, detail, name, run)
}

func (h *hostScreen) containersKey(a *App, k tea.KeyPressMsg) (tea.Cmd, bool) {
	switch {
	case key.Matches(k, navKeys.Up):
		h.ctrCursor = max(h.ctrCursor-1, 0)
		return nil, true
	case key.Matches(k, navKeys.Down):
		h.ctrCursor = min(h.ctrCursor+1, max(len(h.ctr.Services)-1, 0))
		return nil, true
	}
	if h.ctrCursor >= len(h.ctr.Services) {
		return nil, false
	}
	c := h.ctr.Services[h.ctrCursor]
	svc, id := c.ID, h.id
	base := tpath(id, "/containers/"+url.PathEscape(svc))
	const tab = "containers"
	switch {
	case key.Matches(k, svcKeys.Start):
		return h.boxCall(a, tab, "start "+svc, http.MethodPost, base+"/start", nil, reloadContainers), true
	case key.Matches(k, svcKeys.Stop), key.Matches(k, svcKeys.Restart):
		action := "stop"
		if key.Matches(k, svcKeys.Restart) {
			action = "restart"
		}
		typedThen(a, fmt.Sprintf("%s %s on %s", action, svc, id), fmt.Sprintf("The %s container on %s stops serving until it is running again.", svc, id), svc,
			func() tea.Cmd {
				return h.boxCall(a, tab, action+" "+svc, http.MethodPost, base+"/"+action, nil, reloadContainers)
			})
		return nil, true
	case key.Matches(k, boxKeys.Provision):
		return h.boxCall(a, tab, "provision "+svc, http.MethodPost, base+"/provision", nil, func(h *hostScreen, a *App) tea.Cmd {
			return h.followSetup(a, tab, "provision "+svc)
		}), true
	case key.Matches(k, boxKeys.Wipe):
		detail := "Its data is deleted"
		if c.WipeDiscards != "" {
			detail += ": " + c.WipeDiscards
		}
		if len(c.RestartsOnWipe) > 0 {
			detail += ". These restart after: " + strings.Join(c.RestartsOnWipe, ", ")
		}
		typedThen(a, "wipe "+svc+" on "+id, detail+".", svc, func() tea.Cmd {
			return h.boxCall(a, tab, "wipe "+svc, http.MethodPost, base+"/wipe", map[string]string{"Confirm": svc}, reloadContainers)
		})
		return nil, true
	case key.Matches(k, boxKeys.Reset):
		typedThen(a, "reset "+svc+" on "+id, "The chain starts over: its data is deleted and the gateways in front of it restart.", svc, func() tea.Cmd {
			return h.boxCall(a, tab, "reset "+svc, http.MethodPost, base+"/reset", nil, reloadContainers)
		})
		return nil, true
	case key.Matches(k, boxKeys.Config):
		return h.boxCall(a, tab, svc+" config", http.MethodGet, base+"/config", nil, nil), true
	case key.Matches(k, boxKeys.Edit):
		a.modal = newAsk("set "+svc+" config on "+id, "The whole config as one JSON object; fields left out take their defaults.", "",
			func(a *App, v string) (string, tea.Cmd) {
				obj, miss := jsonObject(v)
				if miss != "" {
					return miss, nil
				}
				return "", h.boxCall(a, tab, "set "+svc+" config", http.MethodPut, base+"/config", obj, reloadContainers)
			})
		return nil, true
	}
	return nil, false
}

// jsonObject reads one JSON object typed into a field.
func jsonObject(v string) (map[string]any, string) {
	var obj map[string]any
	if v == "" {
		return nil, "type a JSON object, or esc to cancel"
	}
	if err := json.Unmarshal([]byte(v), &obj); err != nil || obj == nil {
		return nil, "that is not a JSON object"
	}
	return obj, ""
}

// followSetup shows the box's setup run on tab until it ends.
func (h *hostScreen) followSetup(a *App, tab, head string) tea.Cmd {
	h.runTab, h.runHead = tab, head
	h.setResult(tab, boxResult{head: sanitizeLine(head) + " running" + a.gl.Ellipsis})
	id, gen, ctx, be := h.id, h.gen, h.ctx, a.be
	return func() tea.Msg {
		ch, err := be.WatchSetup(ctx, id)
		if err != nil {
			return setupEvMsg{id: id, gen: gen, closed: true, err: err}
		}
		return h.nextSetupEv(ch)()
	}
}

// gotRunEv adds one setup event to the run's tab.
func (h *hostScreen) gotRunEv(a *App, msg setupEvMsg) tea.Cmd {
	r := h.results[h.runTab]
	if msg.ev.Line != "" {
		r.lines = append(r.lines, sanitizeLine(msg.ev.StepID)+" "+maskText(sanitizeLine(msg.ev.Line)))
		if len(r.lines) > maxResultLines {
			r.lines = r.lines[len(r.lines)-maxResultLines:]
		}
	}
	end := true
	switch {
	case msg.err != nil:
		r.head, r.bad = sanitizeLine(h.runHead)+": FAIL", true
		r.lines = append(r.lines, a.errText(msg.err))
	case msg.ev.Err != "":
		r.head, r.bad = sanitizeLine(h.runHead)+": FAIL", true
		r.lines = append(r.lines, sanitizeLine(msg.ev.Err))
	case msg.ev.Done && msg.ev.StepID == "":
		r.head = sanitizeLine(h.runHead) + ": done"
	case msg.closed:
		r.head = sanitizeLine(h.runHead) + ": the stream ended"
	default:
		end = false
	}
	tab := h.runTab
	h.setResult(tab, r)
	if !end {
		return h.nextSetupEv(msg.ch)
	}
	h.runTab = ""
	if tab == "containers" {
		return h.loadContainers(a)
	}
	return nil
}

func (h *hostScreen) toolsKey(a *App, k tea.KeyPressMsg) (tea.Cmd, bool) {
	id := h.id
	const tab = "tools"
	switch {
	case key.Matches(k, navKeys.Up):
		h.scroll = max(h.scroll-1, 0)
		return nil, true
	case key.Matches(k, navKeys.Down):
		h.scroll++
		return nil, true
	case key.Matches(k, boxKeys.Diag):
		return h.boxCall(a, tab, "diagnostics", http.MethodGet, tpath(id, "/diagnostics"), nil, nil), true
	case key.Matches(k, boxKeys.Latest):
		return h.boxCall(a, tab, "latest diagnostics", http.MethodGet, tpath(id, "/diagnostics/latest"), nil, nil), true
	case key.Matches(k, boxKeys.Monitor):
		return h.toggleMonitor(a), true
	case key.Matches(k, boxKeys.Intent):
		a.modal = newAsk("run an intent on "+id, "One of: "+strings.Join(intentKinds, ", "), "", func(a *App, v string) (string, tea.Cmd) {
			if !slices.Contains(intentKinds, v) {
				return "not one of the kinds listed", nil
			}
			return "", h.boxCall(a, tab, "intent "+v, http.MethodPost, tpath(id, "/intent/"+url.PathEscape(v)), nil, nil)
		})
		return nil, true
	case key.Matches(k, boxKeys.Wizard):
		if h.runTab != "" {
			a.flash = "a setup run is already being followed"
			return nil, true
		}
		a.modal = newAsk("set up the node on "+id, "The wire config as one JSON object (chain, clients, data dir), as the web wizard sends it.", "",
			func(a *App, v string) (string, tea.Cmd) {
				obj, miss := jsonObject(v)
				if miss != "" {
					return miss, nil
				}
				return "", h.boxCall(a, tab, "setup", http.MethodPost, tpath(id, "/setup"), obj, func(h *hostScreen, a *App) tea.Cmd {
					return h.followSetup(a, tab, "setup")
				})
			})
		return nil, true
	}
	return nil, false
}

// toggleMonitor starts or stops the monitor stream. It ends with the box.
func (h *hostScreen) toggleMonitor(a *App) tea.Cmd {
	h.monGen++
	if h.monStop != nil {
		h.monStop()
		h.monStop, h.monLines, h.monErr = nil, nil, nil
		return nil
	}
	ctx, stop := context.WithCancel(h.ctx)
	h.monStop, h.monConn = stop, apiclient.Connecting
	return h.nextMonitor(h.monGen, a.be.Watch(ctx, tpath(h.id, "/monitor/stream")))
}

func (h *hostScreen) nextMonitor(mgen int, ch <-chan apiclient.Update[json.RawMessage]) tea.Cmd {
	id, gen, ctx := h.id, h.gen, h.ctx
	return func() tea.Msg {
		select {
		case <-ctx.Done():
			return nil
		case u, ok := <-ch:
			if !ok {
				return nil
			}
			return monitorMsg{id: id, gen: gen, mgen: mgen, u: u, ch: ch}
		}
	}
}

func (h *hostScreen) viewResult(a *App, tab string) []string {
	r, ok := h.results[tab]
	if !ok {
		return nil
	}
	head := a.th.Title.Render(r.head)
	if r.bad {
		head = a.th.Bad.Render(r.head)
	}
	out := []string{"", " " + head}
	for _, l := range r.lines {
		out = append(out, "   "+l)
	}
	return out
}

func (h *hostScreen) viewContainers(a *App, w, hgt int) string {
	switch {
	case !h.ctrHas && h.ctrErr != nil:
		return "\n " + a.th.Bad.Render("containers unavailable: ") + a.errText(h.ctrErr)
	case !h.ctrHas:
		return "\n " + a.th.Dim.Render("reading the containers"+a.gl.Ellipsis)
	}
	d := h.ctr.Docker
	eng := sanitizeLine(d.Flavor)
	if eng == "" {
		eng = "docker"
	}
	if d.ServerVersion != "" {
		eng += " " + sanitizeLine(d.ServerVersion)
	}
	switch {
	case !d.Present:
		eng = a.th.Bad.Render("docker is not installed")
	case !d.Reachable:
		eng = a.th.Bad.Render(eng + " unreachable")
	default:
		eng += " reachable"
	}
	lines := []string{"", " " + eng}
	if d.Detail != "" || d.Hint != "" {
		lines = append(lines, "   "+a.th.Warn.Render(strings.TrimSpace(sanitizeLine(d.Detail)+" "+sanitizeLine(d.Hint))))
	}
	if len(h.ctr.Services) == 0 {
		lines = append(lines, " "+a.th.Dim.Render("no container services"))
	}
	for i, c := range h.ctr.Services {
		mark := " "
		if i == h.ctrCursor {
			mark = a.gl.Sel
		}
		state := sanitizeLine(c.Status.State)
		if state == "" {
			state = "n/a"
		}
		if !c.Configured {
			state += ", not configured"
		}
		lines = append(lines, a.selIf(i == h.ctrCursor, " %s %-8s %s", mark, sanitizeLine(c.ID), state))
		if c.Error != "" {
			lines = append(lines, "   "+a.th.Bad.Render(sanitizeLine(c.Error)))
		}
		var acts []string
		for _, x := range c.Actions {
			acts = append(acts, sanitizeLine(x))
		}
		if len(acts) > 0 {
			lines = append(lines, "   offers "+strings.Join(acts, " "))
		}
		if c.Blocked != "" {
			lines = append(lines, "   "+a.th.Warn.Render(sanitizeLine(c.Blocked)))
		}
		for _, e := range c.Endpoints {
			lines = append(lines, "   "+sanitizeLine(e.Label)+"  "+maskURL(sanitizeLine(e.URL)))
		}
		for _, wn := range c.Warnings {
			lines = append(lines, "   "+a.th.Warn.Render(sanitizeLine(wn)))
		}
	}
	lines = append(lines, "", " s start  t stop  r restart  p provision  W wipe  X reset  v config  e edit")
	lines = append(lines, h.viewResult(a, "containers")...)
	return strings.Join(lines, "\n")
}

func (h *hostScreen) viewTools(a *App, w, hgt int) string {
	lines := []string{"", " D run diagnostics  d latest  M monitor  I intent  w setup wizard"}
	if h.monStop != nil {
		state := "monitor " + h.monConn.String()
		if h.monConn != apiclient.Live {
			state = "monitor ~ " + h.monConn.String()
		}
		lines = append(lines, "", " "+a.th.Title.Render(state))
		if h.monErr != nil {
			lines = append(lines, "   "+a.th.Bad.Render(a.errText(h.monErr)))
		}
		if len(h.monLines) == 0 {
			lines = append(lines, "   "+a.th.Dim.Render("waiting for a reading"+a.gl.Ellipsis))
		}
		for _, l := range h.monLines {
			lines = append(lines, "   "+l)
		}
	}
	lines = append(lines, h.viewResult(a, "tools")...)
	return strings.Join(window(lines, &h.scroll, hgt, w, a.gl.Ellipsis), "\n")
}

// clearService asks for the service's name, then clears its state: it
// resyncs from nothing.
func (h *hostScreen) clearService(a *App, svc string) tea.Cmd {
	id := h.id
	typedThen(a, "clear "+svc+" state on "+id, fmt.Sprintf("The %s client's data on %s is deleted; it resyncs from scratch.", svc, id), svc, func() tea.Cmd {
		return h.boxCall(a, "services", "clear "+svc, http.MethodPost, tpath(id, "/services/"+url.PathEscape(svc)+"/clear"), map[string]string{"Confirm": svc}, nil)
	})
	return nil
}

// diskUsage reads the box's usage by path into the Storage tab.
func (h *hostScreen) diskUsage(a *App) tea.Cmd {
	return h.boxCall(a, "storage", "disk usage by path", http.MethodGet, tpath(h.id, "/du"), nil, nil)
}
