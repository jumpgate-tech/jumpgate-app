package tui

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/apiclient"
	"github.com/valve-tech/jumpgate/internal/tui/tuitest"
)

func sampleInbox() api.JobsInbox {
	hb := testNow.Add(-5 * time.Second)
	old := testNow.Add(-10 * time.Minute)
	return api.JobsInbox{At: testNow, Jobs: []api.JobView{
		{Target: "box-a", ID: "0123456789abcdef", Kind: "node.migrate-names", Title: "Rename the node's account, units and data to jumpgate-node",
			State: "needs-decision", Step: 1, Steps: []api.JobStepView{{ID: "preflight", State: "done"}, {ID: "stop-units", State: "asking"}, {ID: "rename-account", State: "pending"}},
			Question: &api.JobQuestion{ID: "stop-units-0-busy-user", Prompt: "1 process(es) still run as valve-node-app", Tier: "routine", Signable: true,
				Options: []api.JobOption{{ID: "retry", Label: "Check again"}, {ID: "stop-processes", Label: "Stop them"}, {ID: "abort", Label: "Stop the migration here"}}},
			Lease: api.JobLease{Owner: "0xc0ffee00000000000000000000000000000000aa", Label: "this controller", HeartbeatAt: &hb, Fresh: true, Mine: true}, CreatedAt: old},
		{Target: "box-b", ID: "fedcba9876543210", Kind: "node.migrate-names", Title: "Rename the node's account, units and data to jumpgate-node",
			State: "running", Step: 3, Steps: make([]api.JobStepView, 8),
			Lease: api.JobLease{Owner: "0xbbbb000000000000000000000000000000000bbb", HeartbeatAt: &old, Fresh: false}, CreatedAt: old},
	}}
}

func jobsApp(t *testing.T, f *tuitest.Fake, glyphs string) *App {
	t.Helper()
	f.JobsCh <- apiclient.Update[api.JobsInbox]{Value: sampleInbox(), Has: true, State: apiclient.Live}
	return press(t, newTestApp(t, f, 80, 24, glyphs), "5").(*App)
}

func TestInboxShowsWhatNeedsYouFirst(t *testing.T) {
	fr := tuitest.Frame(jobsApp(t, tuitest.NewFake(), "unicode"))
	for _, want := range []string{"Jobs inbox", "1 needs you", "box-a", "⚑", "step 2/3", "1 process(es) still run", "1 Check again", "2 Stop them", "box-b", "running", "silent 10m", "a: adopt"} {
		if !strings.Contains(fr, want) {
			t.Errorf("inbox lacks %q:\n%s", want, fr)
		}
	}
	if strings.Index(fr, "box-a") > strings.Index(fr, "box-b") {
		t.Errorf("the job that needs you is not first:\n%s", fr)
	}
	tuitest.Golden(t, "jobs_80x24", fr)
	ascii := tuitest.Frame(jobsApp(t, tuitest.NewFake(), "ascii"))
	requireASCII(t, ascii)
	tuitest.Golden(t, "jobs_ascii", ascii)
}

// Destructive choices need typed confirmation (umbrella spec); the request is
// sent only for the exact option id.
func TestDecidingNeedsTheOptionTypedExactly(t *testing.T) {
	f := tuitest.NewFake()
	m := jobsApp(t, f, "unicode")
	m = press(t, m, "d", "2").(*App)
	m = typed(t, m, "stop-process").(*App)
	m = press(t, m, "enter").(*App)
	if f.Called("Decide") {
		t.Fatal("a near miss was sent")
	}
	m = typed(t, m, "stop-processes").(*App)
	press(t, m, "enter")
	if !f.Called("Decide box-a 0123456789abcdef stop-units-0-busy-user stop-processes") {
		t.Fatalf("calls %v", f.Calls)
	}
}

// Q5: an approval-tier question is shown, never answered from here.
func TestApprovalQuestionsSayWhyTheyCannotBeAnswered(t *testing.T) {
	f := tuitest.NewFake()
	in := sampleInbox()
	in.Jobs[0].Question.Tier, in.Jobs[0].Question.Signable = "approval", false
	f.JobsCh <- apiclient.Update[api.JobsInbox]{Value: in, Has: true, State: apiclient.Live}
	m := press(t, newTestApp(t, f, 80, 24, "unicode"), "5", "d")
	if fr := tuitest.Frame(m); !strings.Contains(fr, "approval") || f.Called("Decide") {
		t.Fatalf("frame:\n%s\ncalls %v", fr, f.Calls)
	}
}

// Review Focus 3, front end: a job another controller holds freshly is
// watched; d says who holds it instead of asking.
func TestAJobHeldElsewhereIsWatchedOnly(t *testing.T) {
	f := tuitest.NewFake()
	in := sampleInbox()
	in.Jobs[0].Lease = api.JobLease{Owner: "0xbbbb000000000000000000000000000000000bbb", Fresh: true}
	f.JobsCh <- apiclient.Update[api.JobsInbox]{Value: in, Has: true, State: apiclient.Live}
	m := press(t, newTestApp(t, f, 80, 24, "unicode"), "5", "d").(*App)
	if f.Called("Decide") || !strings.Contains(m.flash, "holds this job") {
		t.Fatalf("flash %q calls %v", m.flash, f.Calls)
	}
}

func TestAdoptOnlyAStaleJob(t *testing.T) {
	f := tuitest.NewFake()
	m := press(t, jobsApp(t, f, "unicode"), "a").(*App) // box-a is ours: nothing to adopt
	if f.Called("Adopt") {
		t.Fatal("adopted a job we hold")
	}
	m = press(t, m, "down", "a").(*App)
	m = typed(t, m, "box-b").(*App)
	press(t, m, "enter")
	if !f.Called("Adopt box-b fedcba9876543210") {
		t.Fatalf("calls %v", f.Calls)
	}
}

func TestEnterShowsTheVerifiedLog(t *testing.T) {
	f := tuitest.NewFake()
	f.JobV["box-a/0123456789abcdef"] = api.JobDetail{Log: []api.JobLogLine{{Index: 0, Kind: "job.created", At: testNow, Text: "node.migrate-names by 0xc0ffee: created"}}}
	m := press(t, jobsApp(t, f, "unicode"), "enter")
	if fr := tuitest.Frame(m); !strings.Contains(fr, "node.migrate-names by 0xc0ffee: created") {
		t.Fatalf("frame:\n%s", fr)
	}
}

// Everything a job shows comes from the box, so none of it may carry a
// control sequence onto the screen.
func TestInboxStripsControlSequencesFromJobText(t *testing.T) {
	f := tuitest.NewFake()
	in := sampleInbox()
	in.Jobs[0].Title = "evil\x1b]52;c;AAAA\x07 title"
	in.Jobs[0].State = "needs-decision\x1b[2J"
	in.Jobs[0].Question.Prompt = "ask\x1b[31m red"
	in.Jobs[0].Question.Options[0].Label = "ok\x1b]8;;http://x\x07"
	f.JobsCh <- apiclient.Update[api.JobsInbox]{Value: in, Has: true, State: apiclient.Live}
	fr := tuitest.Frame(press(t, newTestApp(t, f, 80, 24, "unicode"), "5"))
	if strings.Contains(fr, "\x1b") {
		t.Fatalf("an escape reached the frame:\n%q", fr)
	}
}

// reopen presses r, which restarts the stream, then delivers u on the new one.
// Pushing after the restart matters: the old reader would otherwise take it.
func reopen(t *testing.T, f *tuitest.Fake, m *App, u apiclient.Update[api.JobsInbox]) *App {
	t.Helper()
	next, cmds := tuitest.Send(m, tuitest.Key("r"))
	f.JobsCh <- u
	return tuitest.Settle(next, cmds...).(*App)
}

func TestInboxSaysWhenTheStreamDropped(t *testing.T) {
	f := tuitest.NewFake()
	m := jobsApp(t, f, "unicode")
	if strings.Contains(tuitest.Frame(m), "out of date") {
		t.Fatal("a live stream was called stale")
	}
	m = reopen(t, f, m, apiclient.Update[api.JobsInbox]{State: apiclient.Retrying, Err: errors.New("connection refused")})
	fr := tuitest.Frame(m)
	for _, want := range []string{"jobs stream: connection refused", "box-a"} {
		if !strings.Contains(fr, want) {
			t.Errorf("frame lacks %q:\n%s", want, fr)
		}
	}
	m = reopen(t, f, m, apiclient.Update[api.JobsInbox]{State: apiclient.Connecting})
	if fr = tuitest.Frame(m); !strings.Contains(fr, "out of date") || strings.Contains(fr, "connection refused") {
		t.Errorf("a stream that is not live shows its rows as live:\n%s", fr)
	}
}

// slowBackend holds Decide until released and records whether its context
// was cancelled by then.
type slowBackend struct {
	*tuitest.Fake
	started, release chan struct{}
	ctxErr           chan error
}

func (b slowBackend) Decide(ctx context.Context, target, id string, d api.JobDecision) (api.JobView, error) {
	close(b.started)
	<-b.release
	b.ctxErr <- ctx.Err()
	return b.Fake.Decide(ctx, target, id, d)
}

func TestADecisionOutlivesTheScreen(t *testing.T) {
	f := tuitest.NewFake()
	b := slowBackend{Fake: f, started: make(chan struct{}), release: make(chan struct{}), ctxErr: make(chan error, 1)}
	f.JobViewV = api.JobView{State: "running"}
	f.JobsCh <- apiclient.Update[api.JobsInbox]{Value: sampleInbox(), Has: true, State: apiclient.Live}
	a := New(Options{Backend: b, GOOS: "linux", Getenv: func(string) string { return "" }, Now: func() time.Time { return testNow }, Location: time.UTC})
	t.Cleanup(func() { a.cancel() })
	var m tea.Model = a
	m, _ = tuitest.Send(m, tea.WindowSizeMsg{Width: 80, Height: 24}, prefsMsg{p: api.UIPrefs{Glyphs: "unicode"}})
	m = press(t, m, "5")
	m = press(t, m, "d", "1")
	m = typed(t, m, "retry")
	m, cmds := tuitest.Send(m, tuitest.Key("enter"))
	done := make(chan tea.Msg, 1)
	go func() { done <- tuitest.Run(tea.Batch(cmds...))[0] }()
	<-b.started
	m = press(t, m, "1") // leave the inbox while the decision is in flight
	close(b.release)
	if err := <-b.ctxErr; err != nil {
		t.Fatalf("leaving the screen cancelled the decision: %v", err)
	}
	msg := <-done
	m, _ = tuitest.Send(m, msg)
	if fl := m.(*App).flash; !strings.Contains(fl, "decision sent") {
		t.Fatalf("flash %q", fl)
	}
}

// watchRecorder remembers each channel WatchJobs handed out.
type watchRecorder struct {
	*tuitest.Fake
	mu  sync.Mutex
	chs []<-chan apiclient.Update[api.JobsInbox]
}

func (w *watchRecorder) WatchJobs(ctx context.Context) <-chan apiclient.Update[api.JobsInbox] {
	ch := w.Fake.WatchJobs(ctx)
	w.mu.Lock()
	w.chs = append(w.chs, ch)
	w.mu.Unlock()
	return ch
}

func TestReloadRestartsTheStreamAndStopsTheOldReader(t *testing.T) {
	w := &watchRecorder{Fake: tuitest.NewFake()}
	a := New(Options{Backend: w, GOOS: "linux", Getenv: func(string) string { return "" }, Now: func() time.Time { return testNow }, Location: time.UTC})
	t.Cleanup(func() { a.cancel() })
	var m tea.Model = a
	m, _ = tuitest.Send(m, tea.WindowSizeMsg{Width: 80, Height: 24})
	m = press(t, m, "5", "r")
	w.mu.Lock()
	defer w.mu.Unlock()
	if len(w.chs) != 2 {
		t.Fatalf("WatchJobs ran %d times, want 2", len(w.chs))
	}
	select {
	case _, ok := <-w.chs[0]:
		if ok {
			t.Fatal("the old stream delivered after reload")
		}
	case <-time.After(time.Second):
		t.Fatal("the old stream was not stopped by reload")
	}
}

// The option list shows the box's question and each option by number, all
// cleaned of escapes; esc closes it and sends nothing.
func TestTheChoiceListShowsTheQuestionAndEscSendsNothing(t *testing.T) {
	f := tuitest.NewFake()
	in := sampleInbox()
	in.Jobs[0].Question.Prompt = "1 process\x1b[2J still runs\nas valve-node-app"
	in.Jobs[0].Question.Options[1].Detail = "SIGTERM\x1b]52;c;AAAA\x07 first"
	f.JobsCh <- apiclient.Update[api.JobsInbox]{Value: in, Has: true, State: apiclient.Live}
	m := press(t, newTestApp(t, f, 80, 24, "unicode"), "5", "d").(*App)
	fr := tuitest.Frame(m)
	for _, want := range []string{"box-a asks", "1 process still runs", "as valve-node-app", "1 Check again", "2 Stop them", "SIGTERM first", "3 Stop the migration here", "press a number, or esc"} {
		if !strings.Contains(fr, want) {
			t.Errorf("the choice list lacks %q:\n%s", want, fr)
		}
	}
	if strings.Contains(fr, "\x1b]") || strings.Contains(fr, "\x1b[2J") {
		t.Fatalf("an escape from the box reached the frame:\n%q", fr)
	}
	m = press(t, m, "9").(*App) // no such option: the list stays
	if m.modal == nil {
		t.Fatal("a number with no option closed the list")
	}
	m = press(t, m, "esc").(*App)
	if m.modal != nil || f.Called("Decide") {
		t.Fatalf("esc left modal %T, calls %v", m.modal, f.Calls)
	}
}

// d explains why it cannot ask instead of sending a request the box would
// refuse: no open question, or nobody holds the lease.
func TestDecideSaysWhyItCannotAsk(t *testing.T) {
	f := tuitest.NewFake()
	m := press(t, jobsApp(t, f, "unicode"), "down", "d").(*App) // box-b has no question
	if !strings.Contains(m.flash, "not waiting on a decision") || m.modal != nil {
		t.Fatalf("flash %q modal %T", m.flash, m.modal)
	}
	f = tuitest.NewFake()
	in := sampleInbox()
	in.Jobs[0].Lease = api.JobLease{} // released: nobody drives it
	f.JobsCh <- apiclient.Update[api.JobsInbox]{Value: in, Has: true, State: apiclient.Live}
	m = press(t, newTestApp(t, f, 80, 24, "unicode"), "5").(*App)
	if fr := tuitest.Frame(m); !strings.Contains(fr, "nobody drives it") {
		t.Fatalf("frame:\n%s", fr)
	}
	m = press(t, m, "d").(*App)
	if !strings.Contains(m.flash, "adopt it first") || m.modal != nil || f.Called("Decide") {
		t.Fatalf("flash %q modal %T calls %v", m.flash, m.modal, f.Calls)
	}
}

// A job whose controller is alive cannot be adopted from here; the inbox
// says it is watched and a says why not.
func TestAdoptRefusesALiveController(t *testing.T) {
	f := tuitest.NewFake()
	in := sampleInbox()
	in.Jobs[0].Lease = api.JobLease{Owner: "0xbbbb000000000000000000000000000000000bbb", Fresh: true}
	f.JobsCh <- apiclient.Update[api.JobsInbox]{Value: in, Has: true, State: apiclient.Live}
	m := press(t, newTestApp(t, f, 80, 24, "unicode"), "5").(*App)
	if fr := tuitest.Frame(m); !strings.Contains(fr, "watching 0xbbbb") {
		t.Fatalf("frame:\n%s", fr)
	}
	m = press(t, m, "a").(*App)
	if !strings.Contains(m.flash, "still alive") || m.modal != nil || f.Called("Adopt") {
		t.Fatalf("flash %q modal %T calls %v", m.flash, m.modal, f.Calls)
	}
}

// A refused decision is flashed with the server's reason, cleaned; the
// person sees it was not taken.
func TestARefusedDecisionIsFlashed(t *testing.T) {
	f := tuitest.NewFake()
	f.Err = map[string]error{"Decide": &api.Error{Message: "the job moved on\x1b[2J", Code: api.CodeJobState}}
	m := jobsApp(t, f, "unicode")
	m = press(t, m, "d", "1").(*App)
	m = typed(t, m, "retry").(*App)
	m = press(t, m, "enter").(*App)
	if !f.Called("Decide") || !strings.HasPrefix(m.flash, "decision: ") || !strings.Contains(m.flash, "the job moved on") || strings.Contains(m.flash, "\x1b") {
		t.Fatalf("flash %q calls %v", m.flash, f.Calls)
	}
}

// A log that could not be read says why under the job, rather than showing
// an empty history as if it were verified.
func TestALogThatCannotBeReadSaysWhy(t *testing.T) {
	f := tuitest.NewFake()
	f.Err = map[string]error{"Job": &api.Error{Message: "the log does not verify", Code: api.CodeJobLogInvalid}}
	fr := tuitest.Frame(press(t, jobsApp(t, f, "unicode"), "enter"))
	if !strings.Contains(fr, "the log does not verify") {
		t.Fatalf("frame:\n%s", fr)
	}
}

// An empty inbox says how to start a job, and a box that did not answer is
// named with its error.
func TestAnEmptyInboxSaysHowToStartAndNamesSilentBoxes(t *testing.T) {
	f := tuitest.NewFake()
	f.JobsCh <- apiclient.Update[api.JobsInbox]{Value: api.JobsInbox{Errors: []api.JobsError{{Target: "box-c", Error: api.Error{Message: "could not reach the agent"}}}}, Has: true, State: apiclient.Live}
	m := press(t, newTestApp(t, f, 80, 24, "unicode"), "5", "d", "a").(*App)
	fr := tuitest.Frame(m)
	if !strings.Contains(fr, "No durable jobs on any paired box") || !strings.Contains(fr, "box-c: could not reach the agent") {
		t.Fatalf("frame:\n%s", fr)
	}
	if f.Called("Decide") || f.Called("Adopt") {
		t.Fatalf("an empty inbox sent %v", f.Calls)
	}
}
