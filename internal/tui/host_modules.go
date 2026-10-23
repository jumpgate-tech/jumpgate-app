package tui

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/catalog"
)

// The modules a box runs, mirrored from the web's modulesModel.ts. Whether a
// module applies is read from /api/catalog's Applies rows, never derived
// here: ok false with no reason hides it, a reason disables it and is shown
// verbatim.

const (
	gateStaleText = "Status needs a newer gate on this box"
	postWarning   = "Anyone can post through your node. Posts cost the sender proof of work, not you."
	// startingGrace is how long after a restart a module not served yet
	// reads "Starting" (spec D8).
	startingGrace = time.Minute
)

// publicOptions is the Public values p cycles for c: "" (local only) plus
// what the catalog says the module offers, and only for a node module that
// is on behind a public-tier gateway (the server's tier, never re-derived).
// Empty offers no choice.
func (h *hostScreen) publicOptions(c modCard) []string {
	if c.m.Placement != "node" || c.state != "on" || len(c.m.Public) == 0 {
		return nil
	}
	behindPublic := false
	for _, g := range h.modGws {
		behindPublic = behindPublic || (g.Tier == "public" && g.RoutesTo(h.id))
	}
	if !behindPublic {
		return nil
	}
	out := []string{""}
	for _, p := range publicChoices[1:] {
		if slices.Contains(c.m.Public, p.value) {
			out = append(out, p.value)
		}
	}
	return out
}

// publicChoices are a node module's Public values in the order p cycles them.
var publicChoices = []struct{ value, label string }{
	{"", "local only"}, {"read", "public reads"}, {"post", "public reads and posting"},
}

func publicLabel(v string) string {
	for _, c := range publicChoices {
		if c.value == v {
			return c.label
		}
	}
	return sanitizeLine(v)
}

// boxFields is which settings a box module asks for when it is turned on,
// with their defaults: the web's BOX_FIELDS and BOX_FIELD_DEFAULTS.
var boxFields = map[string]struct {
	port  int
	extIP bool
}{
	"el-bootnode": {port: 30301, extIP: true},
	"cl-bootnode": {port: 9200, extIP: true},
	"checkpointz": {port: 5555},
}

type (
	modulesMsg struct {
		id     string
		gen    uint64
		cat    []api.CatalogModule
		gws    []api.GatewaySummary
		target *api.TargetView
		view   api.ModulesView
		err    error // the catalog or targets failed; a status failure only empties view
	}
	modSetMsg struct {
		id, module string
		gen        uint64
		restarts   bool
		res        api.SetModuleResult
		err        error
	}
	setupEvMsg struct {
		id     string
		gen    uint64
		ev     api.SetupEvent
		ch     <-chan api.SetupEvent
		closed bool
		err    error
	}
)

// modCard is one module as the host shows it.
type modCard struct {
	m      api.CatalogModule
	state  string // "disabled" | "off" | "on"
	reason string
	status string
	public string             // node modules: the stored Public
	box    *catalog.BoxModule // box modules: the record, when on
}

// loadModules fetches the catalog, the box's record and its module statuses.
func (h *hostScreen) loadModules(a *App) tea.Cmd {
	id, gen, ctx, be := h.id, h.gen, h.ctx, a.be
	return func() tea.Msg {
		msg := modulesMsg{id: id, gen: gen}
		cat, err := be.Catalog(ctx)
		if err != nil {
			msg.err = err
			return msg
		}
		ts, err := be.Targets(ctx)
		if err != nil {
			msg.err = err
			return msg
		}
		msg.cat = cat
		for i := range ts {
			if ts[i].ID == id {
				msg.target = &ts[i]
			}
		}
		if gws, err := be.Gateways(ctx); err == nil {
			msg.gws = gws // none known: no Public choice
		}
		if v, err := be.ListModules(ctx, id); err == nil {
			msg.view = v
		}
		return msg
	}
}

func appliesRow(m api.CatalogModule, chainID int, client string) (api.ModuleApplies, bool) {
	for _, r := range m.Applies {
		if r.ChainID == chainID && r.Client == client {
			return r, true
		}
	}
	if m.Placement == "box" {
		for _, r := range m.Applies {
			if r.ChainID == 0 && r.Client == "" {
				return r, true
			}
		}
	}
	return api.ModuleApplies{}, false
}

// moduleCards is every node and box module that is not hidden on this box,
// node modules first.
func (h *hostScreen) moduleCards(a *App) []modCard {
	if h.modTarget == nil {
		return nil
	}
	wire := h.modTarget.Wire
	chain, client := 0, ""
	if wire != nil {
		chain, client = wire.ChainID, wire.ExecID
	}
	var out []modCard
	for _, place := range []string{"node", "box"} {
		if place == "node" && wire == nil {
			continue
		}
		for _, m := range h.modCat {
			if m.Placement != place {
				continue
			}
			// A module that is on is always listed, whatever its Applies
			// row says now, so it can be turned off; only an off module is
			// hidden or disabled by the row.
			c := modCard{m: m}
			if place == "node" {
				for _, nm := range wire.Modules {
					if nm.ID == m.ID {
						c.state, c.public = "on", nm.Public
					}
				}
			} else {
				for i, bm := range h.modTarget.Modules {
					if bm.ID == m.ID {
						c.state, c.box = "on", &h.modTarget.Modules[i]
					}
				}
			}
			row, ok := appliesRow(m, chain, map[string]string{"node": client}[place])
			switch {
			case c.state == "on":
				c.reason = row.Reason
			case ok && row.OK:
				c.state = "off"
			case ok && row.Reason != "":
				c.state, c.reason = "disabled", row.Reason
			default:
				continue // hidden
			}
			if c.state != "disabled" {
				c.status = h.moduleStatus(a, c)
			}
			out = append(out, c)
		}
	}
	return out
}

// moduleStatus is the one status line (spec D8 order for node modules; box
// modules report their unit). Off and not observed is "".
func (h *hostScreen) moduleStatus(a *App, c modCard) string {
	if h.modView.GateStale {
		return gateStaleText
	}
	var s *api.ModuleStatus
	for i := range h.modView.Modules {
		if h.modView.Modules[i].ID == c.m.ID {
			s = &h.modView.Modules[i]
		}
	}
	on := c.state == "on"
	sep := " " + a.gl.Sep + " "
	if c.m.Placement == "box" {
		switch {
		case !on && s == nil:
			return ""
		case s != nil && s.Error != "":
			return s.Error
		case s != nil && s.Active && s.Identity != "":
			return "Running" + sep + s.Identity
		case s != nil && s.Active:
			return "Running"
		}
		return "Stopped"
	}
	if on && h.statusHas && !h.status.Exec.Active {
		return "Node stopped"
	}
	if !on {
		if s != nil && s.Served {
			return "Served but not set by jumpgate"
		}
		return ""
	}
	if s == nil || !s.Served {
		if !h.modRestarted.IsZero() && a.now().Sub(h.modRestarted) < startingGrace {
			return "Starting"
		}
		if s != nil && s.Error != "" {
			return s.Error
		}
		return "Not served yet"
	}
	switch {
	case h.statusHas && h.status.Exec.State == api.SyncSyncing:
		return "Waiting for sync"
	case h.statusHas && h.status.Exec.Peers == 0:
		return "No peers"
	case !s.Accepting:
		return "Starting"
	}
	return "Accepting" + sep + formatCount(s.Count) + " msgs"
}

// viewModules is the Overview's Modules block: one line per module that is
// not hidden, or nothing when none applies.
func (h *hostScreen) viewModules(a *App, w int) string {
	cards := h.moduleCards(a)
	if len(cards) == 0 {
		return ""
	}
	pad := 0
	for _, c := range cards {
		pad = max(pad, len([]rune(sanitizeLine(c.m.Name))))
	}
	sep := " " + a.gl.Sep + " "
	lines := []string{"", "", " " + a.th.Title.Render("MODULES")}
	for _, c := range cards {
		name := sanitizeLine(c.m.Name)
		name += strings.Repeat(" ", pad-len([]rune(name))+2)
		lines = append(lines, truncate(" "+name+h.moduleLine(a, c, sep), w, a.gl.Ellipsis))
	}
	return strings.Join(lines, "\n")
}

// moduleLine is a card's state, status and, for a node module that is on,
// its Public choice.
func (h *hostScreen) moduleLine(a *App, c modCard, sep string) string {
	if c.state == "disabled" {
		return a.th.Dim.Render("unavailable: " + sanitizeLine(c.reason))
	}
	s := c.state
	if c.state == "on" {
		s = a.th.OK.Render(s)
	}
	if c.status != "" {
		s += sep + sanitizeLine(c.status)
	}
	if c.state == "on" && c.m.Placement == "node" {
		s += sep + publicLabel(c.public)
	}
	return s
}

// modulesModal is `m`: every module for the open box, with its toggle. A
// node module restarts the exec client, so it takes the typed word restart;
// a box module asks for its fields. Either then follows the setup run.
type modulesModal struct {
	h      *hostScreen
	cursor int
	pub    map[string]string // Public values sent this session, ahead of the reload

	askPost string // the module whose move to public posting awaits enter

	confirm *confirmModal // the typed restart, while asking

	fieldsFor *modCard // the box module being turned on, while asking
	fields    []textinput.Model
	labels    []string
	focus     int

	busy     bool
	note     string
	errText  string
	running  string // the module whose setup run is followed
	runLines []string
	runDone  bool
	runErr   string
}

// newModulesModal opens the modules list for h.
//
// docs:covers msgboard-enable
// docs:covers box-modules
func newModulesModal(a *App, h *hostScreen) *modulesModal {
	return &modulesModal{h: h, pub: map[string]string{}}
}

var modKeys = struct{ Public key.Binding }{
	Public: key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "change who can reach a node module")),
}

func (m *modulesModal) cards(a *App) []modCard {
	cs := m.h.moduleCards(a)
	for i := range cs {
		if p, ok := m.pub[cs[i].m.ID]; ok && cs[i].state == "on" {
			cs[i].public = p
		}
	}
	return cs
}

func (m *modulesModal) restartLine() string {
	client := "the execution client"
	if w := m.h.modTarget; w != nil && w.Wire != nil {
		client = w.Wire.ExecID
	}
	return fmt.Sprintf("This restarts %s on %s. The beacon keeps running.", client, m.h.id)
}

func (m *modulesModal) key(a *App, k tea.KeyPressMsg) (bool, tea.Cmd) {
	if m.confirm != nil {
		done, cmd := m.confirm.key(a, k)
		if done {
			m.confirm = nil
		}
		return false, cmd
	}
	if m.fieldsFor != nil {
		return false, m.fieldsKey(a, k)
	}
	if m.askPost != "" {
		switch k.String() {
		case "esc":
			m.askPost = ""
		case "enter":
			id := m.askPost
			m.askPost = ""
			m.pub[id] = "post"
			return false, m.submit(a, id, api.SetModule{Enabled: true, Public: "post"}, false)
		}
		return false, nil
	}
	cards := m.cards(a)
	switch {
	case k.String() == "esc":
		return true, nil
	case key.Matches(k, navKeys.Up):
		m.cursor = max(m.cursor-1, 0)
	case key.Matches(k, navKeys.Down):
		m.cursor = min(m.cursor+1, max(len(cards)-1, 0))
	case key.Matches(k, navKeys.Enter):
		if m.cursor < len(cards) && !m.busy {
			return false, m.toggle(a, cards[m.cursor])
		}
	case key.Matches(k, modKeys.Public):
		if m.cursor < len(cards) && !m.busy {
			c := cards[m.cursor]
			if opts := m.h.publicOptions(c); len(opts) > 0 {
				next := opts[0]
				for i, p := range opts {
					if p == c.public {
						next = opts[(i+1)%len(opts)]
					}
				}
				if next == "post" { // the D11 warning comes before the request
					m.askPost = c.m.ID
					return false, nil
				}
				m.pub[c.m.ID] = next
				return false, m.submit(a, c.m.ID, api.SetModule{Enabled: true, Public: next}, false)
			}
		}
	}
	return false, nil
}

// toggle turns c on or off. Off is always allowed; a disabled module sends
// nothing and says why.
func (m *modulesModal) toggle(a *App, c modCard) tea.Cmd {
	m.note, m.errText = "", ""
	switch {
	case c.state == "disabled":
		m.note = sanitizeLine(c.reason)
		return nil
	case c.m.Placement == "node":
		req := api.SetModule{Enabled: c.state != "on", Public: ""}
		verb := "Turn on "
		if !req.Enabled {
			verb = "Turn off "
		}
		cf := newConfirm(verb+c.m.Name, m.restartLine(), "restart", func() tea.Cmd { return m.submit(a, c.m.ID, req, true) })
		cf.prompt = "Type %s to go ahead, esc to cancel:"
		m.confirm = cf
		return nil
	case c.state == "on":
		chain := 0
		if c.box != nil {
			chain = c.box.ChainID
		}
		return m.submit(a, c.m.ID, api.SetModule{Enabled: false, ChainID: chain}, false)
	}
	m.openFields(c)
	return nil
}

func (m *modulesModal) openFields(c modCard) {
	m.fieldsFor, m.fields, m.labels, m.focus = &c, nil, nil, 0
	add := func(label, value string) {
		in := textinput.New()
		in.Prompt = ""
		in.CharLimit = 64
		in.KeyMap.Paste.SetEnabled(false)
		in.SetValue(value)
		m.fields, m.labels = append(m.fields, in), append(m.labels, label)
	}
	if f, ok := boxFields[c.m.ID]; ok {
		add("port", strconv.Itoa(f.port))
		if f.extIP {
			add("public IP", "")
		}
	}
	m.focusField()
}

func (m *modulesModal) focusField() {
	for i := range m.fields {
		if i == m.focus {
			m.fields[i].Focus()
		} else {
			m.fields[i].Blur()
		}
	}
}

func (m *modulesModal) fieldsKey(a *App, k tea.KeyPressMsg) tea.Cmd {
	switch k.String() {
	case "esc":
		m.fieldsFor = nil
		return nil
	case "tab", "down":
		if len(m.fields) > 0 {
			m.focus = (m.focus + 1) % len(m.fields)
			m.focusField()
		}
		return nil
	case "shift+tab", "up":
		if len(m.fields) > 0 {
			m.focus = (m.focus + len(m.fields) - 1) % len(m.fields)
			m.focusField()
		}
		return nil
	case "enter":
		req := api.SetModule{Enabled: true}
		if w := m.h.modTarget; w != nil && w.Wire != nil {
			req.ChainID = w.Wire.ChainID
		}
		for i, l := range m.labels {
			v := strings.TrimSpace(m.fields[i].Value())
			switch l {
			case "port":
				p, err := strconv.Atoi(v)
				if err != nil || p < 1 || p > 65535 {
					m.errText = "the port must be a number from 1 to 65535"
					return nil
				}
				req.Port = p
			case "public IP":
				req.ExtIP = v
			}
		}
		id := m.fieldsFor.m.ID
		m.fieldsFor = nil
		return m.submit(a, id, req, false)
	}
	if len(m.fields) == 0 {
		return nil
	}
	var cmd tea.Cmd
	m.fields[m.focus], cmd = m.fields[m.focus].Update(k)
	return cmd
}

func (m *modulesModal) paste(a *App, p tea.PasteMsg) tea.Cmd {
	switch {
	case m.confirm != nil:
		return m.confirm.paste(a, p)
	case m.fieldsFor != nil && len(m.fields) > 0:
		var cmd tea.Cmd
		m.fields[m.focus], cmd = m.fields[m.focus].Update(tea.PasteMsg{Content: sanitizeLine(p.Content)})
		return cmd
	}
	return nil
}

func (m *modulesModal) submit(a *App, module string, req api.SetModule, restarts bool) tea.Cmd {
	m.busy, m.note, m.errText = true, "", ""
	h := m.h
	id, gen, ctx, be := h.id, h.gen, h.ctx, a.be
	return func() tea.Msg {
		res, err := be.SetModule(ctx, id, module, req)
		return modSetMsg{id: id, gen: gen, module: module, restarts: restarts, res: res, err: err}
	}
}

// gotSet takes the server's answer to a change.
func (m *modulesModal) gotSet(a *App, msg modSetMsg) tea.Cmd {
	m.busy = false
	h := m.h
	var e *api.Error
	switch {
	case msg.err != nil && errors.As(msg.err, &e) && e.Status == 502:
		m.note = "Saved. A gateway that routes to this box did not re-provision; provision it from Gateways."
	case msg.err != nil:
		delete(m.pub, msg.module)
		m.errText = a.errText(msg.err)
		return nil
	case msg.res.Status == "started":
		if msg.restarts {
			h.modRestarted = a.now()
		}
		m.running, m.runLines, m.runDone, m.runErr = msg.module, nil, false, ""
		id, gen, ctx, be := h.id, h.gen, h.ctx, a.be
		return func() tea.Msg {
			ch, err := be.WatchSetup(ctx, id)
			if err != nil {
				return setupEvMsg{id: id, gen: gen, closed: true, err: err}
			}
			return h.nextSetupEv(ch)()
		}
	case msg.res.Status == "saved-gateways-failed":
		var parts []string
		for _, f := range msg.res.Failed {
			parts = append(parts, sanitizeLine(f.ID)+" ("+sanitizeLine(f.Error)+")")
		}
		m.note = "Saved. These gateways did not re-provision: " + strings.Join(parts, ", ")
	default:
		m.note = "Saved."
	}
	return h.reloadModules(a)
}

func (h *hostScreen) nextSetupEv(ch <-chan api.SetupEvent) tea.Cmd {
	id, gen, ctx := h.id, h.gen, h.ctx
	return func() tea.Msg {
		select {
		case <-ctx.Done():
			return nil
		case ev, ok := <-ch:
			if !ok {
				return setupEvMsg{id: id, gen: gen, closed: true}
			}
			return setupEvMsg{id: id, gen: gen, ev: ev, ch: ch}
		}
	}
}

// gotSetupEv follows the run: the last few lines, then done or the error.
func (m *modulesModal) gotSetupEv(a *App, msg setupEvMsg) tea.Cmd {
	h := m.h
	if msg.err != nil {
		m.runErr = a.errText(msg.err)
	}
	if msg.ev.Line != "" {
		m.runLines = append(m.runLines, sanitizeLine(msg.ev.Line))
		if len(m.runLines) > 5 {
			m.runLines = m.runLines[len(m.runLines)-5:]
		}
	}
	if msg.ev.Err != "" {
		m.runErr = sanitizeLine(msg.ev.Err)
	}
	end := msg.closed || msg.ev.Err != "" || (msg.ev.Done && msg.ev.StepID == "")
	if msg.ev.Done && msg.ev.StepID == "" {
		m.runDone = true
	}
	if end {
		return h.reloadModules(a)
	}
	return h.nextSetupEv(msg.ch)
}

// reloadModules fetches the box's modules again after a change.
func (h *hostScreen) reloadModules(a *App) tea.Cmd { return h.loadModules(a) }

func (m *modulesModal) view(a *App, w, _ int) string {
	if m.confirm != nil {
		return m.confirm.view(a, w, 0)
	}
	if c := m.fieldsFor; c != nil {
		out := fmt.Sprintf(" %s\n\n", a.th.Title.Render("Turn on "+sanitizeLine(c.m.Name)+" on "+sanitizeLine(m.h.id)))
		for i, l := range m.labels {
			mark := " "
			if i == m.focus {
				mark = a.gl.Sel
			}
			out += fmt.Sprintf(" %s %-10s %s\n", mark, l, m.fields[i].View())
		}
		if m.errText != "" {
			out += "\n " + a.th.Warn.Render(m.errText) + "\n"
		}
		return out + "\n " + a.th.Dim.Render("tab to move, enter to turn on, esc to go back")
	}
	out := fmt.Sprintf(" %s %s\n\n", a.th.Title.Render("MODULES"), a.th.Dim.Render("for "+sanitizeLine(m.h.id)+" (enter to turn on or off, esc to close)"))
	cards := m.cards(a)
	if len(cards) == 0 {
		out += " " + a.th.Dim.Render("No modules apply to this box yet.") + "\n"
	}
	pad := 0
	for _, c := range cards {
		pad = max(pad, len([]rune(sanitizeLine(c.m.Name))))
	}
	sep := " " + a.gl.Sep + " "
	for i, c := range cards {
		mark := " "
		if i == m.cursor {
			mark = a.gl.Sel
		}
		name := sanitizeLine(c.m.Name)
		name += strings.Repeat(" ", pad-len([]rune(name))+2)
		out += truncate(fmt.Sprintf(" %s %s%s", mark, name, m.h.moduleLine(a, c, sep)), w, a.gl.Ellipsis) + "\n"
		if i == m.cursor && len(m.h.publicOptions(c)) > 0 {
			asking := m.askPost == c.m.ID
			line := "reachable: " + publicLabel(c.public) + " (p to change)"
			if asking {
				line = "make it reachable: " + publicLabel("post") + "? enter to go ahead, esc to keep " + publicLabel(c.public)
			}
			out += "     " + a.th.Dim.Render(line) + "\n"
			if c.public == "post" || asking {
				// The D11 copy, wrapped at its sentence: a modal is narrower
				// than the screen.
				for _, s := range strings.SplitAfter(postWarning, ". ") {
					out += "     " + a.th.Warn.Render(strings.TrimSpace(s)) + "\n"
				}
			}
		}
	}
	if m.busy {
		out += "\n " + a.th.Dim.Render("sending"+a.gl.Ellipsis)
	}
	if m.note != "" {
		out += "\n " + m.note
	}
	if m.errText != "" {
		out += "\n " + a.th.Bad.Render(m.errText)
	}
	if m.running != "" {
		head := "Applying " + sanitizeLine(m.running) + a.gl.Ellipsis
		switch {
		case m.runErr != "":
			head = a.th.Bad.Render(m.runErr)
		case m.runDone:
			head = "Done."
		}
		out += "\n\n " + head
		for _, l := range m.runLines {
			out += "\n   " + a.th.Dim.Render(truncate(l, max(w-4, 10), a.gl.Ellipsis))
		}
	}
	return out
}
