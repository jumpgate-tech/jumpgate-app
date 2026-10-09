package jobs

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/valve-tech/jumpgate/internal/intent"
	"github.com/valve-tech/jumpgate/internal/joblog"
)

func refusalCode(err error) string {
	var r *Refusal
	if errors.As(err, &r) {
		return r.Code
	}
	return ""
}

// compact is raw JSON without insignificant space: the manifest is saved
// indented, which re-indents a result it embeds.
func compact(t *testing.T, raw []byte) string {
	t.Helper()
	var b bytes.Buffer
	if err := json.Compact(&b, raw); err != nil {
		t.Fatal(err)
	}
	return b.String()
}

func TestAJobRunsStepByStepAndSucceeds(t *testing.T) {
	g := newRig(t)
	rec, err := g.start(t, g.a, `{}`)
	if err != nil || rec.State != intent.JobWaiting || rec.Lease.Owner != g.a.Address().Hex() || rec.Driver != DriverController {
		t.Fatalf("Start = %+v, %v", rec, err)
	}
	if rec = g.advance(t, g.a, rec.ID); rec.State != intent.JobWaiting || rec.Step != 1 {
		t.Fatalf("after one step: %+v", rec)
	}
	if rec = g.advance(t, g.a, rec.ID); rec.State != intent.JobSucceeded || compact(t, rec.Result) != `{"did":"two"}` {
		t.Fatalf("after two steps: %+v", rec)
	}
	_, res := g.read(t, rec.ID)
	var kinds []string
	for _, j := range res.Entries {
		kinds = append(kinds, j.Kind)
	}
	want := []string{joblog.KindCreated, joblog.KindIntent, joblog.KindStepStarted, joblog.KindStepFinished,
		joblog.KindIntent, joblog.KindStepStarted, joblog.KindStepFinished, joblog.KindFinished}
	if len(kinds) != len(want) {
		t.Fatalf("log kinds %v, want %v", kinds, want)
	}
	for i := range want {
		if kinds[i] != want[i] {
			t.Fatalf("log kinds %v, want %v", kinds, want)
		}
	}
	// The log is what a controller checks: the agent signed every entry, and
	// every recorded request is a controller's, for this agent and this job
	// (whose id the creating intent derives).
	es, _, err := g.m.store.ReadLog(rec.ID, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := joblog.Verify(g.box.Address(), rec.ID, es, 0, [32]byte{}); err != nil {
		t.Fatal(err)
	}
	if err := joblog.VerifyIntents(g.box.Address(), rec.ID, es); err != nil {
		t.Fatal(err)
	}
}

func TestStartRefusesAnEnvelopeWhoseIntentDoesNotParse(t *testing.T) {
	g := newRig(t)
	if _, err := g.m.Start(context.Background(), "test.two", []byte(`{}`), g.a.Address(), intent.Envelope{}); refusalCode(err) != intent.ReasonValidation {
		t.Fatalf("Start with no intent = %v", err)
	}
	if _, ok := g.m.Active(); ok {
		t.Fatal("a refused start left a job behind")
	}
}

// J1.
func TestOneJobAtATimePerBox(t *testing.T) {
	g := newRig(t)
	if _, err := g.start(t, g.a, `{}`); err != nil {
		t.Fatal(err)
	}
	if _, err := g.start(t, g.b, `{}`); refusalCode(err) != intent.ReasonJobActive {
		t.Fatalf("second Start = %v", err)
	}
}

// Review Focus 3.
func TestLeaseFreshRefusesAnotherController(t *testing.T) {
	g := newRig(t)
	rec, _ := g.start(t, g.a, `{}`)
	if _, err := g.advanceAt(t, g.b, rec.ID, 0, 0); refusalCode(err) != intent.ReasonLeaseHeld {
		t.Fatalf("advance by a non-holder = %v", err)
	}
	if _, err := g.lease(t, g.b, rec.ID, intent.LeaseTake); refusalCode(err) != intent.ReasonLeaseHeld {
		t.Fatalf("take of a fresh lease = %v", err)
	}
	g.now = g.now.Add(LeaseTTL + time.Second)
	got, err := g.lease(t, g.b, rec.ID, intent.LeaseTake)
	if err != nil || got.Lease.Owner != g.b.Address().Hex() {
		t.Fatalf("adopting a stale lease: %+v, %v", got.Lease, err)
	}
	// Adoption is "rerun the plan" from where the job stands.
	if got = g.advance(t, g.b, rec.ID); got.Step != 1 {
		t.Fatalf("the adopter could not drive: %+v", got)
	}
	if _, err := g.lease(t, g.a, rec.ID, intent.LeaseRenew); refusalCode(err) != intent.ReasonLeaseHeld {
		t.Fatalf("the old holder renewed: %v", err)
	}
}

// Q7: a renewal moves the heartbeat on disk and adds nothing to the log.
func TestLeaseRenewalIsSavedButNotLogged(t *testing.T) {
	g := newRig(t)
	rec, _ := g.start(t, g.a, `{}`)
	g.now = g.now.Add(30 * time.Second)
	got, err := g.lease(t, g.a, rec.ID, intent.LeaseRenew)
	if err != nil || got.Lease.HeartbeatAt != g.now.Unix() || got.LogLen != rec.LogLen {
		t.Fatalf("renew = %+v, %v (log was %d)", got, err, rec.LogLen)
	}
	if disk, _ := g.read(t, rec.ID); disk.Lease.HeartbeatAt != g.now.Unix() {
		t.Fatalf("the renewal was not saved: %+v", disk.Lease)
	}
	if _, err := g.lease(t, g.a, rec.ID, intent.LeaseRelease); err != nil {
		t.Fatal(err)
	}
	disk, res := g.read(t, rec.ID)
	if disk.Lease.Owner != "" || res.Entries[len(res.Entries)-1].Kind != joblog.KindLeaseReleased {
		t.Fatalf("after release: %+v, last %s", disk.Lease, res.Entries[len(res.Entries)-1].Kind)
	}
}

func TestAStaleAdvanceIsRefused(t *testing.T) {
	g := newRig(t)
	rec, _ := g.start(t, g.a, `{}`)
	g.advance(t, g.a, rec.ID)
	if _, err := g.advanceAt(t, g.a, rec.ID, 0, 0); refusalCode(err) != intent.ReasonJobState {
		t.Fatalf("an advance for a step already done = %v", err)
	}
}

func TestAFailedStepAsksAndRetryRunsTheNextAttempt(t *testing.T) {
	g := newRig(t)
	rec, _ := g.start(t, g.a, `{"FailAt":"one"}`)
	rec = g.advance(t, g.a, rec.ID)
	if rec.State != intent.JobNeedsDecision || rec.Question == nil || rec.Question.Tier != intent.TierRoutine || rec.Unit != "" {
		t.Fatalf("after a failure: %+v", rec)
	}
	if _, err := g.decide(t, g.a, rec.ID, "wrong-id", intent.OptRetry); refusalCode(err) != intent.ReasonJobState {
		t.Fatalf("an answer to another question = %v", err)
	}
	if _, err := g.decide(t, g.a, rec.ID, rec.Question.ID, "nope"); refusalCode(err) != intent.ReasonInvalidPayload {
		t.Fatalf("an option the question does not offer = %v", err)
	}
	rec, err := g.decide(t, g.a, rec.ID, rec.Question.ID, intent.OptRetry)
	if err != nil || rec.State != intent.JobWaiting || rec.Attempt != 1 || rec.Question != nil {
		t.Fatalf("after retry: %+v, %v", rec, err)
	}
	if rec = g.advance(t, g.a, rec.ID); rec.Step != 1 {
		t.Fatalf("the retry did not pass: %+v", rec)
	}
	if got := g.r.started; len(got) != 2 || got[1] != UnitName(rec.ID, "one", 1) {
		t.Fatalf("units %v", got)
	}
}

func TestAStepQuestionReachesTheNextAttempt(t *testing.T) {
	g := newRig(t)
	rec, _ := g.start(t, g.a, `{"AskAt":"two"}`)
	g.advance(t, g.a, rec.ID)
	rec = g.advance(t, g.a, rec.ID)
	if rec.Question == nil || rec.Question.Key != "pick" || rec.Question.Step != "two" {
		t.Fatalf("question %+v", rec.Question)
	}
	rec, _ = g.decide(t, g.a, rec.ID, rec.Question.ID, "go")
	if rec = g.advance(t, g.a, rec.ID); rec.State != intent.JobSucceeded {
		t.Fatalf("the answered step did not finish: %+v", rec)
	}
}

func TestAbortEndsTheJob(t *testing.T) {
	g := newRig(t)
	rec, _ := g.start(t, g.a, `{"FailAt":"two"}`)
	g.advance(t, g.a, rec.ID)
	rec = g.advance(t, g.a, rec.ID)
	rec, err := g.decide(t, g.a, rec.ID, rec.Question.ID, intent.OptAbort)
	if err != nil || rec.State != intent.JobAborted {
		t.Fatalf("abort: %+v, %v", rec, err)
	}
	if _, ok := g.m.Active(); ok {
		t.Fatal("an aborted job still counts as active")
	}
	_, res := g.read(t, rec.ID)
	if last := res.Entries[len(res.Entries)-1]; last.Kind != joblog.KindFinished {
		t.Fatalf("last entry %s", last.Kind)
	}
}

// Review Focus 2: the step finished while the agent was down; the next agent
// reconciles it exactly once.
func TestReconcileFoldsAResultWrittenWhileTheAgentWasDown(t *testing.T) {
	g := newRig(t)
	g.r.hold = true
	rec, _ := g.start(t, g.a, `{}`)
	if _, err := g.advanceAt(t, g.a, rec.ID, 0, 0); err != nil {
		t.Fatal(err)
	}
	// The agent stops; its store lock goes with it.
	if err := g.m.Close(); err != nil {
		t.Fatal(err)
	}
	g.r.finish(context.Background(), UnitName(rec.ID, "one", 0)) // while "the agent" is away

	// The same box key and directory: a new Manager is a restarted agent.
	m2, err := NewManager(ManagerConfig{Dir: g.dir, Key: g.box, Runner: g.r, Now: func() time.Time { return g.now }, StepArgv: g.m.cfg.StepArgv})
	if err != nil {
		t.Fatal(err)
	}
	g.m = m2 // the rig's cleanup closes it
	for i := 0; i < 2; i++ {
		if err := m2.Reconcile(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	res, _ := m2.Read(rec.ID, 0)
	finished := 0
	for _, e := range res.Entries {
		if e.Kind == joblog.KindStepFinished {
			finished++
		}
	}
	if finished != 1 || res.Job.Step != 1 || res.Job.State != intent.JobWaiting || res.Job.Unit != "" {
		t.Fatalf("step.finished x%d, job %+v", finished, res.Job)
	}
}

// One agent per box: a second Manager on a directory in use fails fast.
func TestASecondManagerOnTheSameDirectoryFails(t *testing.T) {
	g := newRig(t)
	_, err := NewManager(ManagerConfig{Dir: g.dir, Key: g.box, Runner: g.r, StepArgv: g.m.cfg.StepArgv})
	if !errors.Is(err, ErrStoreInUse) {
		t.Fatalf("second NewManager = %v, want ErrStoreInUse", err)
	}
}

// Review Focus 2 / Q9: a unit gone with no result (a reboot) is never rerun
// silently.
func TestLostStepAsksBeforeRerunning(t *testing.T) {
	g := newRig(t)
	g.r.hold = true
	rec, _ := g.start(t, g.a, `{}`)
	if _, err := g.advanceAt(t, g.a, rec.ID, 0, 0); err != nil {
		t.Fatal(err)
	}
	g.r.lose(UnitName(rec.ID, "one", 0))
	if err := g.m.Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	rec, res := g.read(t, rec.ID)
	if rec.State != intent.JobNeedsDecision || len(g.r.started) != 1 {
		t.Fatalf("a lost step: %+v, units %v", rec, g.r.started)
	}
	if last := res.Entries[len(res.Entries)-1]; last.Kind != joblog.KindDecisionAsked {
		t.Fatalf("last entry %s", last.Kind)
	}
	if prev := res.Entries[len(res.Entries)-2]; prev.Kind != joblog.KindStepLost {
		t.Fatalf("entry before the question %s", prev.Kind)
	}
	// A second pass finds nothing running and changes nothing.
	if err := g.m.Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	if again, _ := g.read(t, rec.ID); again.LogLen != rec.LogLen || len(g.r.started) != 1 {
		t.Fatalf("a second reconcile moved the job: %+v", again)
	}
}

func TestAStepThatCannotStartAsks(t *testing.T) {
	g := newRig(t)
	g.r.fail = errors.New("Unit already exists")
	rec, _ := g.start(t, g.a, `{}`)
	rec, err := g.advanceAt(t, g.a, rec.ID, 0, 0)
	if err != nil || rec.State != intent.JobNeedsDecision || rec.Unit != "" {
		t.Fatalf("advance with a failing runner: %+v, %v", rec, err)
	}
	if disk, _ := g.read(t, rec.ID); disk.State != intent.JobNeedsDecision || disk.LogLen != rec.LogLen {
		t.Fatalf("on disk: %+v", disk)
	}
}

// The agent refuses to resume a job whose plan it does not know (spec).
func TestUnknownPlanCanBeReadButNotAdvanced(t *testing.T) {
	g := newRig(t)
	rec, _ := g.start(t, g.a, `{}`)
	Unregister("test.two/1") // a downgraded agent
	if _, err := g.m.Read(rec.ID, 0); err != nil {
		t.Fatalf("read of an unknown-plan job: %v", err)
	}
	if _, err := g.advanceAt(t, g.a, rec.ID, 0, 0); refusalCode(err) != intent.ReasonUnknownPlan {
		t.Fatalf("advance = %v", err)
	}
}

// A damaged job is evidence: reconcile neither folds it nor fails on it.
func TestReconcileLeavesADamagedJobAlone(t *testing.T) {
	g := newRig(t)
	g.r.hold = true
	rec, _ := g.start(t, g.a, `{}`)
	if _, err := g.advanceAt(t, g.a, rec.ID, 0, 0); err != nil {
		t.Fatal(err)
	}
	g.r.lose(UnitName(rec.ID, "one", 0))
	logFile := filepath.Join(g.dir, rec.ID+".log")
	b, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(logFile, b[:len(b)-10], 0o600); err != nil { // a torn last line
		t.Fatal(err)
	}
	if err := g.m.Reconcile(context.Background()); err != nil {
		t.Fatalf("reconcile over a damaged job: %v", err)
	}
	if after, _ := os.ReadFile(logFile); string(after) != string(b[:len(b)-10]) {
		t.Fatal("reconcile wrote to a damaged job's log")
	}
	if got, _ := g.read(t, rec.ID); got.State != intent.JobDamaged {
		t.Fatalf("read of a damaged job: %+v", got)
	}
}

func TestStartRefusesWhenTheRunnerIsUnsupported(t *testing.T) {
	g := newRig(t)
	g.m.cfg.Runner = refusingRunner{}
	if _, err := g.start(t, g.a, `{}`); refusalCode(err) != intent.ReasonJobsUnsupported {
		t.Fatalf("Start = %v", err)
	}
}

type refusingRunner struct{}

func (refusingRunner) Supported(context.Context) error { return ErrUnsupported }
func (refusingRunner) Start(context.Context, string, []string) error {
	return ErrUnsupported
}
func (refusingRunner) Active(context.Context, string) (bool, error) { return false, nil }

// QuestionTier lets the agent demand an approval signature before admission.
func TestQuestionTierIsTheOpenQuestions(t *testing.T) {
	g := newRig(t)
	rec, _ := g.start(t, g.a, `{"FailAt":"one"}`)
	rec = g.advance(t, g.a, rec.ID)
	if got := g.m.QuestionTier(rec.ID, rec.Question.ID); got != intent.TierRoutine {
		t.Fatalf("tier %q", got)
	}
	if got := g.m.QuestionTier(rec.ID, "nope"); got != intent.TierRoutine {
		t.Fatalf("an unknown question's tier %q; Decide refuses it later", got)
	}
}
