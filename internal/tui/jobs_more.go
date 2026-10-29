package tui

// Jobs gaps (inventory G): the open job's live output and starting a job;
// gateway modules (F) on the Gateways screen.

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/apiclient"
	"github.com/valve-tech/jumpgate/internal/intent"
)

const (
	maxJobOutput   = 200 // lines kept
	shownJobOutput = 6   // lines shown under the open job
)

// jobStartKinds are the kinds POST /api/jobs/{target} starts (the server's
// allowlist).
var jobStartKinds = []string{intent.KindNodeMigrateNames}

type jobOutputMsg struct {
	gen  uint64
	ogen int
	u    apiclient.Update[[]api.JobOutputLine]
	ch   <-chan apiclient.Update[[]api.JobOutputLine]
}

// watchOutput follows j's journal until the job is closed or the inbox left.
func (s *jobsScreen) watchOutput(a *App, j api.JobView) tea.Cmd {
	s.stopOutput()
	ctx, stop := context.WithCancel(s.ctx)
	s.outStop, s.outConn = stop, apiclient.Connecting
	return s.nextOutput(s.outGen, a.be.WatchJobOutput(ctx, j.Target, j.ID))
}

func (s *jobsScreen) stopOutput() {
	s.outGen++
	if s.outStop != nil {
		s.outStop()
		s.outStop = nil
	}
	s.out = nil
}

func (s *jobsScreen) nextOutput(ogen int, ch <-chan apiclient.Update[[]api.JobOutputLine]) tea.Cmd {
	gen := s.gen
	return func() tea.Msg {
		u, ok := <-ch
		if !ok {
			return nil
		}
		return jobOutputMsg{gen: gen, ogen: ogen, u: u, ch: ch}
	}
}

func (s *jobsScreen) gotOutput(msg jobOutputMsg) tea.Cmd {
	if !s.current(msg.gen) || msg.ogen != s.outGen || s.outStop == nil {
		return nil
	}
	s.outConn = msg.u.State
	if msg.u.Reset {
		s.out = nil
	}
	for _, l := range msg.u.Value {
		s.out = append(s.out, sanitizeLine(l.Unit)+" "+maskText(sanitizeLine(l.Line)))
	}
	if len(s.out) > maxJobOutput {
		s.out = s.out[len(s.out)-maxJobOutput:]
	}
	return s.nextOutput(msg.ogen, msg.ch)
}

func (s *jobsScreen) viewOutput(a *App, w int, ind string) string {
	if s.outStop == nil {
		return ""
	}
	head := "output"
	if s.outConn != apiclient.Live {
		head += " ~ " + s.outConn.String()
	}
	var b strings.Builder
	b.WriteString(a.th.Dim.Render(ind+head) + "\n")
	if len(s.out) == 0 {
		b.WriteString(a.th.Dim.Render(ind+"none yet") + "\n")
	}
	for _, l := range s.out[max(len(s.out)-shownJobOutput, 0):] {
		b.WriteString(truncate(ind+l, w, a.gl.Ellipsis) + "\n")
	}
	return b.String()
}

// startJob asks for the box and the kind, then the box's name typed: a job
// changes the box.
func (s *jobsScreen) startJob(a *App) {
	def := ""
	if j, ok := s.selected(); ok {
		def = j.Target + " "
	}
	a.modal = newAsk("start a job", "BOX KIND; kinds: "+strings.Join(jobStartKinds, ", "), def, func(a *App, v string) (string, tea.Cmd) {
		f := strings.Fields(v)
		if len(f) != 2 {
			return "type the box and the kind, e.g. box-a " + jobStartKinds[0], nil
		}
		id, kind := f[0], f[1]
		if !slices.Contains(jobStartKinds, kind) {
			return "not a job kind: " + strings.Join(jobStartKinds, ", "), nil
		}
		if !known(a, id) {
			return "no box named " + sanitizeLine(id), nil
		}
		a.modal = newConfirm("start "+kind+" on "+id, "The job runs on the box step by step and asks you before anything it cannot undo.", id, func() tea.Cmd {
			return a.do("start job", func(ctx context.Context, be Backend) tea.Msg {
				var jv api.JobView
				if err := be.Do(ctx, http.MethodPost, "/api/jobs/"+url.PathEscape(id), api.JobStart{Kind: kind}, &jv); err != nil {
					return errMsg{what: "start " + kind + " on " + id, err: err}
				}
				return openJobsMsg{flash: sanitizeLine("job " + jv.ID + " started on " + id)}
			})
		})
		return "", nil
	})
}

// --- gateway modules ---

var gwModKey = key.NewBinding(key.WithKeys("m"), key.WithHelp("m", "modules of the selected gateway"))

type gwModuleMsg struct {
	gw, module string
	on         bool
	err        error
}

type gwCatalogMsg struct {
	cat []api.CatalogModule
	err error
}

// gwModulesModal lists the catalog's gateway modules for one gateway; a
// number turns that one on or off.
type gwModulesModal struct {
	g    api.GatewaySummary
	cat  []api.CatalogModule // gateway modules only
	has  bool
	err  error
	busy bool
}

func newGwModules(a *App, g api.GatewaySummary) (*gwModulesModal, tea.Cmd) {
	return &gwModulesModal{g: g}, a.do("catalog", func(ctx context.Context, be Backend) tea.Msg {
		cat, err := be.Catalog(ctx)
		return gwCatalogMsg{cat: cat, err: err}
	})
}

func (m *gwModulesModal) on(id string) bool {
	return slices.ContainsFunc(m.g.Config.Modules, func(r api.GatewayModuleRef) bool { return r.ID == id })
}

func (m *gwModulesModal) key(a *App, k tea.KeyPressMsg) (bool, tea.Cmd) {
	if k.String() == "esc" {
		return true, nil
	}
	for i, c := range m.cat {
		if k.String() != fmt.Sprint(i+1) || m.busy || i >= 9 {
			continue
		}
		gid, mod, want := m.g.ID, c.ID, !m.on(c.ID)
		m.busy = true
		return false, a.do("gateway module", func(ctx context.Context, be Backend) tea.Msg {
			err := be.Do(ctx, http.MethodPut, "/api/gateways/"+url.PathEscape(gid)+"/modules/"+url.PathEscape(mod), map[string]bool{"enabled": want}, nil)
			return gwModuleMsg{gw: gid, module: mod, on: want, err: err}
		})
	}
	return false, nil
}

func (m *gwModulesModal) view(a *App, w, _ int) string {
	var b strings.Builder
	label := m.g.Label
	if label == "" {
		label = m.g.ID
	}
	b.WriteString(a.th.Title.Render(" modules of "+sanitizeLine(label)) + "\n\n")
	switch {
	case m.err != nil:
		b.WriteString(" " + a.th.Bad.Render("catalog unavailable: ") + a.errText(m.err) + "\n")
	case !m.has:
		b.WriteString(" " + a.th.Dim.Render("reading the catalog"+a.gl.Ellipsis) + "\n")
	case len(m.cat) == 0:
		b.WriteString(" " + a.th.Dim.Render("the catalog has no gateway modules") + "\n")
	}
	for i, c := range m.cat {
		mark := "[ ]"
		if m.on(c.ID) {
			mark = "[x]"
		}
		name := c.Name
		if name == "" {
			name = c.ID
		}
		b.WriteString(truncate(fmt.Sprintf(" %d %s %s", i+1, mark, sanitizeLine(name)), w, a.gl.Ellipsis) + "\n")
	}
	return b.String() + "\n " + a.th.Dim.Render("a number turns it on or off, esc closes")
}

// gwModulesUpdate takes the modal's replies; ok is false for any other.
func (s *gatewaysScreen) gwModulesUpdate(a *App, msg tea.Msg) (tea.Cmd, bool) {
	switch msg := msg.(type) {
	case gwCatalogMsg:
		if m, ok := a.modal.(*gwModulesModal); ok {
			m.has, m.err = msg.err == nil, msg.err
			m.cat = nil
			for _, c := range msg.cat {
				if c.Placement == "gateway" {
					m.cat = append(m.cat, c)
				}
			}
		}
		return nil, true
	case gwModuleMsg:
		if _, ok := a.modal.(*gwModulesModal); ok {
			a.modal = nil
		}
		if msg.err != nil {
			a.flash = sanitizeLine(msg.module) + " on " + sanitizeLine(msg.gw) + ": " + a.errText(msg.err)
			return nil, true
		}
		word := "off"
		if msg.on {
			word = "on"
		}
		a.flash = sanitizeLine(msg.module) + " " + word + " for " + sanitizeLine(msg.gw)
		return loadGateways(a), true
	}
	return nil, false
}
