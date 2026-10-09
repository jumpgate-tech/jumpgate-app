package tui

import (
	"context"
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/apiclient"
)

var jobKeys = struct{ Decide, Adopt, Reload key.Binding }{
	Decide: key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "answer the selected job's question")),
	Adopt:  key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "adopt a job whose controller went silent")),
	Reload: key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "reload")),
}

// openJobsMsg opens the inbox with a notice, once a job has been started.
type openJobsMsg struct{ flash string }

type (
	jobsMsg struct {
		gen uint64
		u   apiclient.Update[api.JobsInbox]
		ch  <-chan apiclient.Update[api.JobsInbox]
	}
	jobDetailMsg struct {
		gen uint64
		key string
		d   api.JobDetail
		err error
	}
	jobActionMsg struct {
		what string
		v    api.JobView
		err  error
	}
)

// jobsScreen is the fleet-wide inbox (umbrella screen 3): jobs that need a
// decision first, then running and waiting ones, then finished ones. It
// watches /api/jobs/stream while it is visible.
type jobsScreen struct {
	in     api.JobsInbox
	has    bool
	conn   apiclient.ConnState
	err    error
	cursor int
	open   string // "target/id" whose log is shown
	detail *api.JobDetail
	detErr error
	visit
}

func (s *jobsScreen) capturing() bool { return false }

func (s *jobsScreen) keys() []key.Binding {
	return []key.Binding{navKeys.Up, navKeys.Down, navKeys.Enter, jobKeys.Decide, jobKeys.Adopt, jobKeys.Reload}
}

func (s *jobsScreen) selected() (api.JobView, bool) {
	if s.cursor < 0 || s.cursor >= len(s.in.Jobs) {
		return api.JobView{}, false
	}
	return s.in.Jobs[s.cursor], true
}

func (s *jobsScreen) watch(a *App) tea.Cmd {
	s.visit.begin(a.ctx)
	ch := a.be.WatchJobs(s.ctx)
	return s.next(s.gen, ch)
}

func (s *jobsScreen) next(gen uint64, ch <-chan apiclient.Update[api.JobsInbox]) tea.Cmd {
	return func() tea.Msg {
		u, ok := <-ch
		if !ok {
			return nil
		}
		return jobsMsg{gen: gen, u: u, ch: ch}
	}
}

func (s *jobsScreen) loadDetail(a *App, j api.JobView) tea.Cmd {
	gen, ctx, be, k := s.gen, s.ctx, a.be, j.Target+"/"+j.ID
	return func() tea.Msg {
		d, err := be.Job(ctx, j.Target, j.ID)
		return jobDetailMsg{gen: gen, key: k, d: d, err: err}
	}
}

// act runs a mutation on the app's context, not the screen's: a decision is
// signed and sent once, and leaving the inbox must not cancel it half way and
// lose the answer. The result is flashed wherever the person has gone.
func (s *jobsScreen) act(a *App, what string, f func(ctx context.Context, be Backend) (api.JobView, error)) tea.Cmd {
	ctx, be := a.ctx, a.be
	return func() tea.Msg {
		v, err := f(ctx, be)
		return jobActionMsg{what: what, v: v, err: err}
	}
}

func (s *jobsScreen) update(a *App, msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case enterMsg:
		return s.watch(a)
	case leaveMsg:
		s.visit.invalidate()
		return nil
	case jobsMsg:
		if !s.current(msg.gen) {
			return nil
		}
		s.conn, s.err = msg.u.State, msg.u.Err
		if msg.u.Has {
			s.in, s.has = msg.u.Value, true
			s.cursor = min(s.cursor, max(len(s.in.Jobs)-1, 0))
		}
		return s.next(msg.gen, msg.ch)
	case jobDetailMsg:
		if s.current(msg.gen) && msg.key == s.open {
			s.detail, s.detErr = &msg.d, msg.err
		}
		return nil
	case jobActionMsg:
		if msg.err != nil {
			a.flash = sanitizeLine(msg.what) + ": " + a.errText(msg.err)
			return nil
		}
		a.flash = sanitizeLine(msg.what + " sent; the box answered " + msg.v.State)
		return nil
	case tea.KeyPressMsg:
		if a.active != scrJobs || a.detail != nil {
			return nil
		}
		switch {
		case key.Matches(msg, navKeys.Up):
			s.cursor = max(s.cursor-1, 0)
		case key.Matches(msg, navKeys.Down):
			s.cursor = min(s.cursor+1, max(len(s.in.Jobs)-1, 0))
		case key.Matches(msg, navKeys.Enter):
			j, ok := s.selected()
			if !ok {
				return nil
			}
			if k := j.Target + "/" + j.ID; s.open != k {
				s.open, s.detail, s.detErr = k, nil, nil
				return s.loadDetail(a, j)
			}
			s.open = ""
		case key.Matches(msg, jobKeys.Decide):
			return s.decide(a)
		case key.Matches(msg, jobKeys.Adopt):
			return s.adopt(a)
		case key.Matches(msg, jobKeys.Reload):
			return s.watch(a)
		}
	}
	return nil
}

// decide opens the option list for the selected job's question. Who may
// answer is the box's to enforce (the lease, the question's tier); the TUI
// says so first rather than sending a request it knows will be refused.
func (s *jobsScreen) decide(a *App) tea.Cmd {
	j, ok := s.selected()
	switch {
	case !ok || j.Question == nil:
		a.flash = "this job is not waiting on a decision"
	case !j.Question.Signable:
		a.flash = "this decision needs an " + sanitizeLine(j.Question.Tier) + "-tier signature (a hardware or browser wallet); jumpgate cannot make one yet"
	case !j.Lease.Mine && j.Lease.Fresh:
		a.flash = "controller " + shortAddr(sanitizeLine(j.Lease.Owner), a.gl) + " holds this job; you are watching it"
	case !j.Lease.Mine:
		a.flash = "nobody drives this job now; adopt it first (a)"
	default:
		a.modal = &choiceModal{job: j, screen: s}
	}
	return nil
}

func (s *jobsScreen) adopt(a *App) tea.Cmd {
	j, ok := s.selected()
	switch {
	case !ok || j.Lease.Mine:
		a.flash = "this controller already drives that job"
	case j.Lease.Fresh:
		a.flash = "its controller is still alive; you can adopt the job once its heartbeat goes stale"
	default:
		a.modal = newConfirm("Adopt job "+j.ID+" on "+j.Target,
			"This controller takes over driving the job. The controller that started it stops when it next checks in.", j.Target,
			func() tea.Cmd {
				return s.act(a, "adopt", func(ctx context.Context, be Backend) (api.JobView, error) { return be.Adopt(ctx, j.Target, j.ID) })
			})
	}
	return nil
}

// choiceModal lists a question's options by number; a number opens the typed
// confirmation for that option.
type choiceModal struct {
	job    api.JobView
	screen *jobsScreen
}

func (m *choiceModal) key(a *App, k tea.KeyPressMsg) (bool, tea.Cmd) {
	if k.String() == "esc" {
		return true, nil
	}
	q := m.job.Question
	for i, o := range q.Options {
		if k.String() == fmt.Sprint(i+1) {
			o, j := o, m.job
			a.modal = newConfirm("Answer: "+o.Label, q.Prompt+"\n\n"+o.Detail, o.ID, func() tea.Cmd {
				return m.screen.act(a, "decision", func(ctx context.Context, be Backend) (api.JobView, error) {
					return be.Decide(ctx, j.Target, j.ID, api.JobDecision{Decision: q.ID, Option: o.ID})
				})
			})
			return false, nil
		}
	}
	return false, nil
}

func (m *choiceModal) view(a *App, w, h int) string {
	q := m.job.Question
	var b strings.Builder
	b.WriteString(a.th.Title.Render(" "+sanitizeLine(m.job.Target)+" asks") + "\n\n")
	for _, line := range strings.Split(sanitize(q.Prompt), "\n") {
		b.WriteString(" " + truncate(line, w-2, a.gl.Ellipsis) + "\n")
	}
	b.WriteString("\n")
	for i, o := range q.Options {
		b.WriteString(fmt.Sprintf(" %d %s", i+1, sanitizeLine(o.Label)))
		if o.Detail != "" {
			b.WriteString(a.th.Dim.Render(" " + a.gl.Sep + " " + sanitizeLine(o.Detail)))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n" + a.th.Dim.Render(" press a number, or esc"))
	return b.String()
}

func (s *jobsScreen) view(a *App, w, h int) string {
	var b strings.Builder
	need := s.in.NeedsDecision()
	head := " Jobs inbox"
	if s.has {
		head += fmt.Sprintf(" %s %d needs you", a.gl.Sep, need)
	}
	b.WriteString(a.th.Title.Render(head) + "\n")
	switch {
	case s.err != nil:
		b.WriteString(a.th.Bad.Render(truncate(" jobs stream: "+a.errText(s.err), w, a.gl.Ellipsis)) + "\n")
	case s.conn != apiclient.Live:
		b.WriteString(a.th.Dim.Render(" server "+s.conn.String()+"; what follows may be out of date") + "\n")
	}
	if !s.has {
		b.WriteString(a.th.Dim.Render(" loading"+a.gl.Ellipsis) + "\n")
		return b.String()
	}
	if len(s.in.Jobs) == 0 {
		b.WriteString("\n No durable jobs on any paired box. Start one with :migrate-names NAME.\n")
	}
	for i, j := range s.in.Jobs {
		mark := "  "
		if i == s.cursor {
			mark = a.gl.Sel + " "
		}
		flag := " "
		if j.State == "needs-decision" {
			flag = a.th.Warn.Render(a.gl.Flag)
		}
		line := fmt.Sprintf("%s%s %-8s %-12s step %d/%d %s", mark, flag, truncate(sanitizeLine(j.Target), 8, a.gl.Ellipsis),
			sanitizeLine(j.State), min(j.Step+1, len(j.Steps)), len(j.Steps), s.leaseWord(a, j))
		b.WriteString(truncate(line, w, a.gl.Ellipsis) + "\n")
		if i == s.cursor {
			b.WriteString(s.selectedBody(a, j, w))
		}
	}
	for _, e := range s.in.Errors {
		b.WriteString(a.th.Bad.Render(truncate(" "+sanitizeLine(e.Target)+": "+sanitizeLine(e.Error.Message), w, a.gl.Ellipsis)) + "\n")
	}
	return fit(b.String(), w, h, a.gl.Ellipsis)
}

func (s *jobsScreen) leaseWord(a *App, j api.JobView) string {
	switch {
	case j.Lease.Mine:
		return "driven here"
	case j.Lease.Owner == "":
		return "nobody drives it"
	case j.Lease.Fresh:
		return "watching " + shortAddr(sanitizeLine(j.Lease.Owner), a.gl)
	case j.Lease.HeartbeatAt != nil:
		return "silent " + formatAge(a.now().Sub(*j.Lease.HeartbeatAt)) + " " + a.gl.Sep + " a: adopt"
	}
	return "a: adopt"
}

func (s *jobsScreen) selectedBody(a *App, j api.JobView, w int) string {
	var b strings.Builder
	ind := "      "
	b.WriteString(a.th.Dim.Render(truncate(ind+sanitizeLine(j.Title), w, a.gl.Ellipsis)) + "\n")
	if q := j.Question; q != nil {
		for _, line := range strings.Split(sanitize(q.Prompt), "\n") {
			b.WriteString(truncate(ind+line, w, a.gl.Ellipsis) + "\n")
		}
		for i, o := range q.Options {
			b.WriteString(truncate(fmt.Sprintf("%s%d %s", ind, i+1, sanitizeLine(o.Label)), w, a.gl.Ellipsis) + "\n")
		}
		if !q.Signable {
			b.WriteString(a.th.Warn.Render(ind+"needs an "+sanitizeLine(q.Tier)+"-tier signature") + "\n")
		}
	}
	if s.open == j.Target+"/"+j.ID {
		switch {
		case s.detErr != nil:
			b.WriteString(a.th.Bad.Render(truncate(ind+a.errText(s.detErr), w, a.gl.Ellipsis)) + "\n")
		case s.detail == nil:
			b.WriteString(a.th.Dim.Render(ind+"reading the signed log"+a.gl.Ellipsis) + "\n")
		default:
			for _, l := range s.detail.Log {
				b.WriteString(truncate(fmt.Sprintf("%s%3d %s", ind, l.Index, sanitizeLine(l.Text)), w, a.gl.Ellipsis) + "\n")
			}
		}
	}
	return b.String()
}
