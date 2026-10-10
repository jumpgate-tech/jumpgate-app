package jobs

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
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
func (refusingRunner) Active(context.Context, string) (bool, error)    { return false, nil }
func (refusingRunner) AnyActive(context.Context, string) (bool, error) { return false, nil }

// QuestionTier lets the agent demand an approval signature before admission.
func TestQuestionTierIsTheOpenQuestions(t *testing.T) {
	g := newRig(t)
	rec, _ := g.start(t, g.a, `{"FailAt":"one"}`)
	rec = g.advance(t, g.a, rec.ID)
	if got, err := g.m.QuestionTier(rec.ID, rec.Question.ID); err != nil || got != intent.TierRoutine {
		t.Fatalf("tier %q %v", got, err)
	}
	if got, err := g.m.QuestionTier(rec.ID, "nope"); err != nil || got != intent.TierRoutine {
		t.Fatalf("an unknown question's tier %q %v; Decide refuses it later", got, err)
	}
	if got, err := g.m.QuestionTier("0123456789abcdef", "nope"); err != nil || got != intent.TierRoutine {
		t.Fatalf("no such job: %q %v; Decide refuses it later", got, err)
	}
}

// M2: a manifest the early lookup cannot read is an error, never routine.
func TestQuestionTierFailsClosed(t *testing.T) {
	g := newRig(t)
	rec, _ := g.start(t, g.a, `{"AskAt":"one"}`)
	rec = g.advance(t, g.a, rec.ID)
	path := filepath.Join(g.m.store.Dir(), rec.ID+".json")
	if err := os.WriteFile(path, []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got, err := g.m.QuestionTier(rec.ID, rec.Question.ID); err == nil {
		t.Fatalf("a damaged manifest read as tier %q", got)
	}
}

// M1: the early lookup never writes, even where a Load would save a repair.
func TestQuestionTierNeverWrites(t *testing.T) {
	g := newRig(t)
	rec, _ := g.start(t, g.a, `{"AskAt":"one"}`)
	rec = g.advance(t, g.a, rec.ID)
	path := filepath.Join(g.m.store.Dir(), rec.ID+".json")
	b, _ := os.ReadFile(path)
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	// One entry behind, chained onto the last entry's prev: the crash Load
	// repairs, and saves.
	last, _, err := g.m.store.ReadLog(rec.ID, rec.LogLen-1, 1)
	if err != nil || len(last) != 1 {
		t.Fatal(err)
	}
	m["logLen"] = float64(rec.LogLen - 1)
	m["logHead"] = "0x" + hex.EncodeToString(last[0].Prev[:])
	stale, _ := json.Marshal(m)
	if err := os.WriteFile(path, stale, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := g.m.QuestionTier(rec.ID, rec.Question.ID); err != nil {
		t.Fatal(err)
	}
	if after, _ := os.ReadFile(path); string(after) != string(stale) {
		t.Fatal("QuestionTier rewrote the manifest")
	}
	if _, err := g.m.store.Load(rec.ID); err != nil {
		t.Fatalf("the stale manifest was not the repairable kind: %v", err)
	}
	if after, _ := os.ReadFile(path); string(after) == string(stale) {
		t.Fatal("Load did not repair it, so the test proves nothing")
	}
}

// Q5, where it is answered: Decide checks the tier of the very question it
// answers, so a question that opened after the agent admitted the decision
// is still guarded, and stays open when refused.
func TestDecideChecksTheTierOfTheQuestionItAnswers(t *testing.T) {
	g := newRig(t)
	rec, _ := g.start(t, g.a, `{"AskAt":"one","AskTier":"approval"}`)
	rec = g.advance(t, g.a, rec.ID)
	if rec.Question == nil || rec.Question.Tier != intent.TierApproval {
		t.Fatalf("job %+v", rec)
	}
	before := rec.LogLen
	for name, check := range map[string]TierCheck{"routine only": routineOnly, "no check": nil} {
		_, err := g.decideAs(t, g.a, rec.ID, rec.Question.ID, "go", check)
		var r *Refusal
		if !errors.As(err, &r) || r.Code != intent.ReasonUnauthorizedKind {
			t.Fatalf("%s: %v", name, err)
		}
	}
	after, _ := g.read(t, rec.ID)
	if after.State != intent.JobNeedsDecision || after.Question == nil || after.Question.ID != rec.Question.ID || after.LogLen != before {
		t.Fatalf("a refused decision changed the job: %+v", after)
	}
	var asked string
	if _, err := g.decideAs(t, g.a, rec.ID, rec.Question.ID, "go", func(tier string) error { asked = tier; return nil }); err != nil || asked != intent.TierApproval {
		t.Fatalf("with the tier covered: %v (checked %q)", err, asked)
	}
}

// failManifestSave makes the n-th manifest save from now fail, once: a crash
// or a full disk between an entry and the manifest that records it.
func failManifestSave(t *testing.T, n int) {
	t.Helper()
	orig, calls := writeManifestFile, 0
	writeManifestFile = func(p string, b []byte) error {
		if calls++; calls == n {
			return errors.New("injected manifest save failure")
		}
		return orig(p, b)
	}
	t.Cleanup(func() { writeManifestFile = orig })
}

// failLogSync makes the n-th log fsync from now fail, once, so that append
// is cut back: a change stopped between its entries.
func failLogSync(t *testing.T, n int) {
	t.Helper()
	orig, calls := syncFile, 0
	syncFile = func(f *os.File) error {
		if calls++; calls == n {
			return errors.New("injected fsync failure")
		}
		return orig(f)
	}
	t.Cleanup(func() { syncFile = orig })
}

func (g *rig) kinds(t *testing.T, id string) []string {
	t.Helper()
	_, res := g.read(t, id)
	var out []string
	for _, e := range res.Entries {
		out = append(out, e.Kind)
	}
	return out
}

func count(kinds []string, kind string) int {
	n := 0
	for _, k := range kinds {
		if k == kind {
			n++
		}
	}
	return n
}

// mustFailThenReconcile runs the reconcile the injected failure breaks, then
// two clean ones: the first repairs, the second must change nothing.
func (g *rig) mustFailThenReconcile(t *testing.T, id string) intent.JobRecord {
	t.Helper()
	if err := g.m.Reconcile(context.Background()); err == nil {
		t.Fatal("the injected failure did not reach Reconcile")
	}
	if err := g.m.Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	rec, _ := g.read(t, id)
	if err := g.m.Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	if again, _ := g.read(t, id); again.LogLen != rec.LogLen || again.State != rec.State {
		t.Fatalf("a clean reconcile moved a repaired job: %+v", again)
	}
	return rec
}

// Review Focus 2: fold's step.finished reached the log but its manifest save
// failed; the next reconcile must not fold the same result again.
func TestAFoldWhoseManifestSaveFailedIsNotFoldedAgain(t *testing.T) {
	g := newRig(t)
	rec, _ := g.start(t, g.a, `{}`)
	if _, err := g.advanceAt(t, g.a, rec.ID, 0, 0); err != nil {
		t.Fatal(err)
	}
	failManifestSave(t, 1)
	rec = g.mustFailThenReconcile(t, rec.ID)
	ks := g.kinds(t, rec.ID)
	if count(ks, joblog.KindStepFinished) != 1 || rec.State != intent.JobWaiting || rec.Step != 1 || rec.Unit != "" {
		t.Fatalf("log %v, job %+v", ks, rec)
	}
	if rec = g.advance(t, g.a, rec.ID); rec.State != intent.JobSucceeded || compact(t, rec.Result) != `{"did":"two"}` {
		t.Fatalf("the repaired job could not go on: %+v", rec)
	}
}

func TestALostStepWhoseManifestSaveFailedIsAskedOnce(t *testing.T) {
	g := newRig(t)
	g.r.hold = true
	rec, _ := g.start(t, g.a, `{}`)
	if _, err := g.advanceAt(t, g.a, rec.ID, 0, 0); err != nil {
		t.Fatal(err)
	}
	g.r.lose(UnitName(rec.ID, "one", 0))
	failManifestSave(t, 1)
	rec = g.mustFailThenReconcile(t, rec.ID)
	ks := g.kinds(t, rec.ID)
	if count(ks, joblog.KindStepLost) != 1 || count(ks, joblog.KindDecisionAsked) != 1 || ks[len(ks)-1] != joblog.KindDecisionAsked {
		t.Fatalf("log %v", ks)
	}
	if rec.State != intent.JobNeedsDecision || rec.Question == nil || rec.Question.ID != "one-0-lost" {
		t.Fatalf("job %+v", rec)
	}
}

func TestAStepQuestionWhoseManifestSaveFailedIsAskedOnce(t *testing.T) {
	g := newRig(t)
	rec, _ := g.start(t, g.a, `{"AskAt":"one"}`)
	if _, err := g.advanceAt(t, g.a, rec.ID, 0, 0); err != nil {
		t.Fatal(err)
	}
	failManifestSave(t, 1)
	rec = g.mustFailThenReconcile(t, rec.ID)
	ks := g.kinds(t, rec.ID)
	if count(ks, joblog.KindStepFinished) != 1 || count(ks, joblog.KindDecisionAsked) != 1 || ks[len(ks)-1] != joblog.KindDecisionAsked {
		t.Fatalf("log %v", ks)
	}
	if rec.State != intent.JobNeedsDecision || rec.Question == nil || rec.Question.Key != "pick" {
		t.Fatalf("job %+v", rec)
	}
}

// A failed start whose step.finished was logged but not saved folds as the
// failure it logged, not as a lost step.
func TestAFailedStartWhoseManifestSaveFailedAsksOnce(t *testing.T) {
	g := newRig(t)
	g.r.fail = errors.New("Unit already exists")
	rec, _ := g.start(t, g.a, `{}`)
	failManifestSave(t, 3) // intent, step.started, then step.finished's save
	if _, err := g.advanceAt(t, g.a, rec.ID, 0, 0); err == nil {
		t.Fatal("the injected failure did not reach Advance")
	}
	if err := g.m.Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	rec, _ = g.read(t, rec.ID)
	ks := g.kinds(t, rec.ID)
	if count(ks, joblog.KindStepFinished) != 1 || count(ks, joblog.KindStepLost) != 0 || ks[len(ks)-1] != joblog.KindDecisionAsked {
		t.Fatalf("log %v", ks)
	}
	if rec.Question == nil || rec.Question.ID != "one-0-failed" {
		t.Fatalf("job %+v", rec)
	}
}

// The trailing decision.asked of a failed step never reached the log.
func TestReconcileRestoresADroppedQuestion(t *testing.T) {
	g := newRig(t)
	rec, _ := g.start(t, g.a, `{"FailAt":"one"}`)
	if _, err := g.advanceAt(t, g.a, rec.ID, 0, 0); err != nil {
		t.Fatal(err)
	}
	failLogSync(t, 2) // step.finished, then decision.asked
	if err := g.m.Reconcile(context.Background()); err == nil {
		t.Fatal("the injected failure did not reach Reconcile")
	}
	if ks := g.kinds(t, rec.ID); ks[len(ks)-1] != joblog.KindStepFinished {
		t.Fatalf("the failure did not land between the entries: %v", ks)
	}
	if err := g.m.Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	rec, _ = g.read(t, rec.ID)
	ks := g.kinds(t, rec.ID)
	if count(ks, joblog.KindStepFinished) != 1 || count(ks, joblog.KindDecisionAsked) != 1 || ks[len(ks)-1] != joblog.KindDecisionAsked || rec.State != intent.JobNeedsDecision {
		t.Fatalf("log %v, job %+v", ks, rec)
	}
}

// An operation closes the gap before it writes, so its own entries cannot
// hide it: the decision lands after the restored question.
func TestADecisionRestoresADroppedQuestionFirst(t *testing.T) {
	g := newRig(t)
	rec, _ := g.start(t, g.a, `{"FailAt":"one"}`)
	if _, err := g.advanceAt(t, g.a, rec.ID, 0, 0); err != nil {
		t.Fatal(err)
	}
	failLogSync(t, 2)
	_ = g.m.Reconcile(context.Background())
	rec, _ = g.read(t, rec.ID)
	if _, err := g.decide(t, g.a, rec.ID, rec.Question.ID, intent.OptRetry); err != nil {
		t.Fatal(err)
	}
	ks := g.kinds(t, rec.ID)
	want := []string{joblog.KindStepFinished, joblog.KindDecisionAsked, joblog.KindIntent, joblog.KindDecisionAnswered}
	if got := ks[len(ks)-4:]; !equal(got, want) {
		t.Fatalf("log ends %v, want %v", got, want)
	}
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// The job.finished after a job's last step never reached the log.
func TestReconcileRestoresADroppedFinish(t *testing.T) {
	g := newRig(t)
	rec, _ := g.start(t, g.a, `{}`)
	g.advance(t, g.a, rec.ID)
	if _, err := g.advanceAt(t, g.a, rec.ID, 1, 0); err != nil {
		t.Fatal(err)
	}
	failLogSync(t, 2) // step.finished, then job.finished
	rec = g.mustFailThenReconcile(t, rec.ID)
	ks := g.kinds(t, rec.ID)
	if count(ks, joblog.KindFinished) != 1 || ks[len(ks)-1] != joblog.KindFinished || rec.State != intent.JobSucceeded {
		t.Fatalf("log %v, job %+v", ks, rec)
	}
}

// The job.finished after an abort never reached the log.
func TestReconcileRestoresADroppedAbortFinish(t *testing.T) {
	g := newRig(t)
	rec, _ := g.start(t, g.a, `{"FailAt":"one"}`)
	rec = g.advance(t, g.a, rec.ID)
	failLogSync(t, 3) // intent, decision.answered, then job.finished
	if _, err := g.decide(t, g.a, rec.ID, rec.Question.ID, intent.OptAbort); err == nil {
		t.Fatal("the injected failure did not reach Decide")
	}
	if err := g.m.Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	rec, _ = g.read(t, rec.ID)
	ks := g.kinds(t, rec.ID)
	if count(ks, joblog.KindFinished) != 1 || ks[len(ks)-1] != joblog.KindFinished || rec.State != intent.JobAborted {
		t.Fatalf("log %v, job %+v", ks, rec)
	}
}

// decision.answered reached the log but its manifest save failed: the answer
// the log records is the one that takes effect, once.
func TestAnAnswerWhoseManifestSaveFailedTakesEffectOnce(t *testing.T) {
	g := newRig(t)
	rec, _ := g.start(t, g.a, `{"FailAt":"one"}`)
	rec = g.advance(t, g.a, rec.ID)
	q := rec.Question.ID
	failManifestSave(t, 2) // intent, then decision.answered's save
	if _, err := g.decide(t, g.a, rec.ID, q, intent.OptRetry); err == nil {
		t.Fatal("the injected failure did not reach Decide")
	}
	if err := g.m.Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	rec, _ = g.read(t, rec.ID)
	if rec.State != intent.JobWaiting || rec.Attempt != 1 || rec.Question != nil {
		t.Fatalf("job %+v", rec)
	}
	if _, err := g.decide(t, g.a, rec.ID, q, intent.OptRetry); refusalCode(err) != intent.ReasonJobState {
		t.Fatalf("answering again = %v", err)
	}
	if ks := g.kinds(t, rec.ID); count(ks, joblog.KindDecisionAnswered) != 1 {
		t.Fatalf("log %v", ks)
	}
}

// The lease.taken after an adoption never reached the log.
func TestReconcileRestoresADroppedLeaseTake(t *testing.T) {
	g := newRig(t)
	rec, _ := g.start(t, g.a, `{}`)
	g.now = g.now.Add(LeaseTTL + time.Second)
	failLogSync(t, 2) // intent, then lease.taken
	if _, err := g.lease(t, g.b, rec.ID, intent.LeaseTake); err == nil {
		t.Fatal("the injected failure did not reach Lease")
	}
	if err := g.m.Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	_, res := g.read(t, rec.ID)
	last := res.Entries[len(res.Entries)-1]
	var b joblog.LeaseBody
	if last.Kind != joblog.KindLeaseTaken || json.Unmarshal(last.Body, &b) != nil ||
		b.Owner != g.b.Address().Hex() || b.Previous != g.a.Address().Hex() {
		t.Fatalf("last entry %s %s", last.Kind, last.Body)
	}
	if err := g.m.Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	if ks := g.kinds(t, rec.ID); count(ks, joblog.KindLeaseTaken) != 1 {
		t.Fatalf("log %v", ks)
	}
}

// The lease.released after a release never reached the log.
func TestReconcileRestoresADroppedLeaseRelease(t *testing.T) {
	g := newRig(t)
	rec, _ := g.start(t, g.a, `{}`)
	failLogSync(t, 2) // intent, then lease.released
	if _, err := g.lease(t, g.a, rec.ID, intent.LeaseRelease); err == nil {
		t.Fatal("the injected failure did not reach Lease")
	}
	if err := g.m.Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	_, res := g.read(t, rec.ID)
	last := res.Entries[len(res.Entries)-1]
	var b joblog.LeaseBody
	if last.Kind != joblog.KindLeaseReleased || json.Unmarshal(last.Body, &b) != nil || b.Owner != g.a.Address().Hex() {
		t.Fatalf("last entry %s %s", last.Kind, last.Body)
	}
}

// A lease take whose own intent entry was never saved changed no holder:
// nothing is restored.
func TestAnUnsavedLeaseTakeIsNotRestored(t *testing.T) {
	g := newRig(t)
	rec, _ := g.start(t, g.a, `{}`)
	g.now = g.now.Add(LeaseTTL + time.Second)
	failManifestSave(t, 1) // the intent entry's save
	if _, err := g.lease(t, g.b, rec.ID, intent.LeaseTake); err == nil {
		t.Fatal("the injected failure did not reach Lease")
	}
	if err := g.m.Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	got, _ := g.read(t, rec.ID)
	if ks := g.kinds(t, rec.ID); got.Lease.Owner != g.a.Address().Hex() || count(ks, joblog.KindLeaseTaken) != 0 {
		t.Fatalf("lease %+v, log %v", got.Lease, ks)
	}
}

// unsavedFold leaves the job as an interrupted fold does: step one's
// step.finished is in the log, and the manifest still says running.
func (g *rig) unsavedFold(t *testing.T) intent.JobRecord {
	t.Helper()
	rec, _ := g.start(t, g.a, `{}`)
	if _, err := g.advanceAt(t, g.a, rec.ID, 0, 0); err != nil {
		t.Fatal(err)
	}
	failManifestSave(t, 1)
	if err := g.m.Reconcile(context.Background()); err == nil {
		t.Fatal("the injected failure did not reach Reconcile")
	}
	if ks := g.kinds(t, rec.ID); ks[len(ks)-1] != joblog.KindStepFinished {
		t.Fatalf("log %v", ks)
	}
	if rec, _ = g.read(t, rec.ID); rec.State != intent.JobRunning {
		t.Fatalf("the fold was saved after all: %+v", rec)
	}
	return rec
}

// A lease change on a job with an unsaved fold settles the fold first; its
// own entries cannot bury the outcome for a second fold.
func TestALeaseCannotBuryAnUnsavedFold(t *testing.T) {
	for _, tc := range []struct {
		name string
		op   func(t *testing.T, g *rig, id string) (intent.JobRecord, error)
	}{
		{"adoption", func(t *testing.T, g *rig, id string) (intent.JobRecord, error) {
			g.now = g.now.Add(LeaseTTL + time.Second)
			return g.lease(t, g.b, id, intent.LeaseTake)
		}},
		{"release", func(t *testing.T, g *rig, id string) (intent.JobRecord, error) {
			return g.lease(t, g.a, id, intent.LeaseRelease)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := newRig(t)
			rec := g.unsavedFold(t)
			got, err := tc.op(t, g, rec.ID)
			if err != nil || got.State != intent.JobWaiting || got.Step != 1 {
				t.Fatalf("lease op: %+v, %v", got, err)
			}
			if err := g.m.Reconcile(context.Background()); err != nil {
				t.Fatal(err)
			}
			rec, _ = g.read(t, rec.ID)
			ks := g.kinds(t, rec.ID)
			if count(ks, joblog.KindStepFinished) != 1 || rec.State != intent.JobWaiting || rec.Step != 1 || rec.Unit != "" {
				t.Fatalf("log %v, job %+v", ks, rec)
			}
		})
	}
}

// An agent that does not know the job's plan cannot settle an unsaved
// outcome, so it refuses an adoption rather than bury the outcome; an agent
// that knows the plan settles it once.
func TestALeaseOnAnUnknownPlanCannotBuryAnUnsavedFold(t *testing.T) {
	g := newRig(t)
	rec := g.unsavedFold(t)
	p, _ := PlanFor("test.two/1")
	Unregister(p.Version) // a downgraded agent
	g.now = g.now.Add(LeaseTTL + time.Second)
	if _, err := g.lease(t, g.b, rec.ID, intent.LeaseTake); refusalCode(err) != intent.ReasonUnknownPlan ||
		!strings.Contains(err.Error(), "run an agent that knows plan test.two/1") {
		t.Fatalf("adoption over an unsaved outcome = %v", err)
	}
	if after, _ := g.read(t, rec.ID); after.LogLen != rec.LogLen || after.Lease.Owner != g.a.Address().Hex() {
		t.Fatalf("the refused lease wrote: %+v", after)
	}
	Register(p) // the plan is back: an agent that knows it
	if err := g.m.Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	rec, _ = g.read(t, rec.ID)
	if ks := g.kinds(t, rec.ID); count(ks, joblog.KindStepFinished) != 1 || rec.State != intent.JobWaiting || rec.Step != 1 {
		t.Fatalf("log %v, job %+v", ks, rec)
	}
}

// With no unsaved outcome, a lease on an unknown-plan job still works.
func TestALeaseOnAnUnknownPlanStillWorks(t *testing.T) {
	g := newRig(t)
	rec, _ := g.start(t, g.a, `{}`)
	Unregister("test.two/1")
	g.now = g.now.Add(LeaseTTL + time.Second)
	got, err := g.lease(t, g.b, rec.ID, intent.LeaseTake)
	if err != nil || got.Lease.Owner != g.b.Address().Hex() {
		t.Fatalf("adoption of an unknown-plan job: %+v, %v", got.Lease, err)
	}
}

// An unsaved successful outcome whose result file is gone cannot be rebuilt:
// every writer fails on the job and writes nothing past it.
func TestAnUnsavedOutcomeWithoutItsResultFailsClosed(t *testing.T) {
	g := newRig(t)
	rec := g.unsavedFold(t)
	p, err := ResultPath(g.dir, rec.ID, "one", 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(p); err != nil {
		t.Fatal(err)
	}
	if _, err := g.lease(t, g.a, rec.ID, intent.LeaseRelease); err == nil {
		t.Fatal("a release appended past an outcome it could not settle")
	}
	g.now = g.now.Add(LeaseTTL + time.Second)
	if _, err := g.lease(t, g.b, rec.ID, intent.LeaseTake); err == nil {
		t.Fatal("an adoption appended past an outcome it could not settle")
	}
	if err := g.m.Reconcile(context.Background()); err == nil {
		t.Fatal("reconcile passed over an outcome it could not settle")
	}
	if after, _ := g.read(t, rec.ID); after.LogLen != rec.LogLen || after.State != intent.JobRunning {
		t.Fatalf("the job moved: %+v", after)
	}
}

// I2: the controller gives up on an advance after systemd accepted the unit.
// The step runs, so the job must stay running with that unit, for Reconcile
// to fold, and must not ask whether to retry: a retry would start a second
// root process beside the first.
func TestACancelledAdvanceWhoseUnitRunsDoesNotAsk(t *testing.T) {
	g := newRig(t)
	g.r.hold = true
	rec, _ := g.start(t, g.a, `{}`)
	ctx, cancel := context.WithCancel(context.Background())
	var startCtxErr error
	g.r.onStart = func(sctx context.Context) {
		cancel() // the request goes away while systemd-run runs
		startCtxErr = sctx.Err()
	}
	g.r.acceptThenFail = errors.New("signal: killed")
	_, err := g.m.Advance(ctx, rec.ID, 0, 0, g.a.Address(),
		g.sign(t, g.a, intent.KindJobAdvance, intent.JobAdvancePayload{ID: rec.ID, Step: 0, Attempt: 0}))
	if err == nil {
		t.Fatal("an advance whose start failed reported success")
	}
	if startCtxErr != nil {
		t.Fatalf("the unit's start was cut short with the request: %v", startCtxErr)
	}
	got, _ := g.read(t, rec.ID)
	if got.State != intent.JobRunning || got.Unit != UnitName(rec.ID, "one", 0) || got.Question != nil {
		t.Fatalf("job after a cancelled start of a running unit: %+v", got)
	}
	if ks := g.kinds(t, rec.ID); count(ks, joblog.KindDecisionAsked) != 0 {
		t.Fatalf("a question was asked about a running step: %v", ks)
	}
	// The unit ends; Reconcile folds it once, as any other.
	g.r.finish(context.Background(), UnitName(rec.ID, "one", 0))
	if err := g.m.Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got, _ = g.read(t, rec.ID); got.State != intent.JobWaiting || got.Step != 1 {
		t.Fatalf("job after its unit ended: %+v", got)
	}
}

// A unit of the job still running (a start that failed after systemd took
// it, or any unit left over) blocks the next advance: two of the job's steps
// must never run at once.
func TestAdvanceRefusesWhileAUnitOfTheJobRuns(t *testing.T) {
	g := newRig(t)
	g.r.hold = true
	rec, _ := g.start(t, g.a, `{}`)
	g.r.mu.Lock()
	g.r.active[UnitName(rec.ID, "one", 7)] = true
	g.r.mu.Unlock()
	if _, err := g.advanceAt(t, g.a, rec.ID, 0, 0); refusalCode(err) != intent.ReasonJobState {
		t.Fatalf("advance beside a running unit = %v", err)
	}
	if len(g.r.started) != 0 {
		t.Fatalf("units started: %v", g.r.started)
	}
}
