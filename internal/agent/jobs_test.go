package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/valve-tech/jumpgate/internal/catalog"
	"github.com/valve-tech/jumpgate/internal/eip712"
	"github.com/valve-tech/jumpgate/internal/executor"
	"github.com/valve-tech/jumpgate/internal/intent"
	"github.com/valve-tech/jumpgate/internal/joblog"
	"github.com/valve-tech/jumpgate/internal/jobs"
	"github.com/valve-tech/jumpgate/internal/jobs/jobstest"
	"github.com/valve-tech/jumpgate/internal/signer"
)

// jobsRig is newRig with a job manager whose units run in-process.
func jobsRig(t *testing.T) (*rig, *jobstest.Runner) {
	t.Helper()
	jobstest.RegisterPlan(t)
	r := newRig(t, true)
	dir := filepath.Join(r.dir, "jobs")
	run := &jobstest.Runner{}
	run.Run = jobstest.InProcess(dir, jobs.Env{Exec: r.exec, NodePath: filepath.Join(r.dir, "node.json"), Out: io.Discard})
	m, err := jobs.NewManager(jobs.ManagerConfig{Dir: dir, Key: r.agentKey, Runner: run, Now: func() time.Time { return r.now },
		StepArgv: func(a jobs.StepArgs) []string { return jobs.StepArgv("/jumpgate", r.dir, r.dir, a) }})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { run.Wait(); _ = m.Close() })
	r.a.cfg.Jobs = m
	// test.steps has no default requirement: the policy must name it.
	p, _ := LoadPolicy(filepath.Join(r.dir, "policy.json"))
	p.Kinds = map[string][]Tier{jobstest.TestKind: {TierRoutine}}
	_ = p.Save(filepath.Join(r.dir, "policy.json"))
	return r, run
}

// ok checks a receipt is StatusOK and decodes its result. It returns a
// function so it can take send's two results directly: ok[T](t)(r.send(…)).
func ok[T any](t *testing.T) func(intent.Receipt, []byte) T {
	return func(rc intent.Receipt, res []byte) T {
		t.Helper()
		if rc.Status != intent.StatusOK {
			t.Fatalf("status %d: %s", rc.Status, res)
		}
		var v T
		if err := json.Unmarshal(res, &v); err != nil {
			t.Fatal(err)
		}
		return v
	}
}

func TestAJobRoundTripsThroughSignedIntents(t *testing.T) {
	r, run := jobsRig(t)
	rec := ok[intent.JobRecord](t)(r.send(t, jobstest.TestKind, map[string]string{}, nil))
	for i := 0; i < 2; i++ {
		rec = ok[intent.JobRecord](t)(r.send(t, intent.KindJobAdvance, intent.JobAdvancePayload{ID: rec.ID, Step: rec.Step, Attempt: rec.Attempt}, nil))
		run.Wait()
		if err := r.a.cfg.Jobs.Reconcile(context.Background()); err != nil {
			t.Fatal(err)
		}
		rec = ok[intent.JobReadResult](t)(r.send(t, intent.KindJobRead, intent.JobReadPayload{ID: rec.ID}, nil)).Job
	}
	res := ok[intent.JobReadResult](t)(r.send(t, intent.KindJobRead, intent.JobReadPayload{ID: rec.ID}, nil))
	if res.Job.State != intent.JobSucceeded {
		t.Fatalf("job %+v", res.Job)
	}
	var es []joblog.Entry
	for _, j := range res.Entries {
		e, err := joblog.FromJSON(j)
		if err != nil {
			t.Fatal(err)
		}
		es = append(es, e)
	}
	if _, err := joblog.Verify(r.agentKey.Address(), rec.ID, es, 0, [32]byte{}); err != nil {
		t.Fatal(err)
	}
	if err := joblog.VerifyIntents(r.agentKey.Address(), rec.ID, es); err != nil {
		t.Fatalf("the log's intents do not verify as the controller's: %v", err)
	}
	list := ok[intent.JobListResult](t)(r.send(t, intent.KindJobList, struct{}{}, nil))
	if len(list.Jobs) != 1 || list.AgentTime != r.now.Unix() {
		t.Fatalf("list %+v", list)
	}
}

// J1: the node services belong to the job while it runs.
func TestServiceActionWaitsForARunningJob(t *testing.T) {
	r, _ := jobsRig(t)
	_ = ok[intent.JobRecord](t)(r.send(t, jobstest.TestKind, map[string]string{}, nil))
	rc, res := r.send(t, intent.KindServiceAction, intent.ServiceActionPayload{Service: "exec", Action: "restart"}, nil)
	if got := rejection(t, rc, res).Code; got != intent.ReasonJobActive {
		t.Fatalf("service.action during a job: %s", got)
	}
	if len(r.exec.cmds) != 0 {
		t.Fatalf("service.action ran commands during a job: %v", r.exec.cmds)
	}
}

// J9: no job manager (or no systemd) is a signed, explained refusal.
func TestJobsWithoutAManagerAreUnsupported(t *testing.T) {
	r := newRig(t, true)
	rc, res := r.send(t, intent.KindJobList, struct{}{}, nil)
	if got := rejection(t, rc, res).Code; got != intent.ReasonJobsUnsupported {
		t.Fatalf("job.list without a manager: %s", got)
	}
}

func TestJobsOnAnUnsupportedBoxSayWhy(t *testing.T) {
	r, run := jobsRig(t)
	run.Unsupported = jobs.ErrUnsupported
	rc, res := r.send(t, jobstest.TestKind, map[string]string{}, nil)
	if got := rejection(t, rc, res); got.Code != intent.ReasonJobsUnsupported {
		t.Fatalf("start on an unsupported box: %+v", got)
	}
}

// Q5: a question of the approval tier needs an approval signature on the
// decide intent, besides the controller's.
func TestAnApprovalQuestionNeedsAnApprovalSignature(t *testing.T) {
	r, run := jobsRig(t)
	rec := ok[intent.JobRecord](t)(r.send(t, jobstest.TestKind, map[string]string{"AskAt": "one"}, nil))
	_ = ok[intent.JobRecord](t)(r.send(t, intent.KindJobAdvance, intent.JobAdvancePayload{ID: rec.ID}, nil))
	run.Wait()
	_ = r.a.cfg.Jobs.Reconcile(context.Background())
	rec = ok[intent.JobReadResult](t)(r.send(t, intent.KindJobRead, intent.JobReadPayload{ID: rec.ID}, nil)).Job
	setQuestionTier(t, r, rec.ID, intent.TierApproval) // the test plan asks routine; raise it on disk

	rc, res := r.send(t, intent.KindJobDecide, intent.JobDecidePayload{ID: rec.ID, Decision: rec.Question.ID, Option: "go"}, nil)
	if got := rejection(t, rc, res).Code; got != intent.ReasonUnauthorizedKind {
		t.Fatalf("an approval question answered with a routine signature: %s", got)
	}
	// Decide refused it, after admission: there is no earlier tier check,
	// so the refused decision used its seq.
	if got := r.a.replay.LastSeq(r.controller.Address()); got != r.seq {
		t.Fatalf("last seq %d after the refused decision sent as %d: refused before admission", got, r.seq)
	}
	wallet, _ := signer.GenerateKey()
	enroll(t, r, wallet, TierApproval)
	rc, res = r.sendSignedBy(t, intent.KindJobDecide, intent.JobDecidePayload{ID: rec.ID, Decision: rec.Question.ID, Option: "go"}, wallet)
	if rc.Status != intent.StatusOK {
		t.Fatalf("with an approval signature: %s", res)
	}
}

// C1: a decision sent after the asking step finished but before its fold.
// At admission there is no question yet; dispatchJob's reconcile then opens
// the approval question, and Decide must
// still refuse it, leaving the question open. Its id is predictable
// (<step>-<attempt>-<key>), so this is a race a routine key could win.
func TestARoutineKeyCannotAnswerAnApprovalQuestionThatOpensLate(t *testing.T) {
	r, run := jobsRig(t)
	rec := ok[intent.JobRecord](t)(r.send(t, jobstest.TestKind, map[string]string{"AskAt": "one", "AskTier": intent.TierApproval}, nil))
	_ = ok[intent.JobRecord](t)(r.send(t, intent.KindJobAdvance, intent.JobAdvancePayload{ID: rec.ID}, nil))
	run.Wait() // the step has finished; nothing has folded it
	if man, _ := jobs.LoadManifest(filepath.Join(r.dir, "jobs"), rec.ID); man.Question != nil {
		t.Fatalf("the question opened before the decision was sent: %+v", man.Question)
	}
	rc, res := r.send(t, intent.KindJobDecide, intent.JobDecidePayload{ID: rec.ID, Decision: "one-0-pick", Option: "go"}, nil)
	if got := rejection(t, rc, res).Code; got != intent.ReasonUnauthorizedKind {
		t.Fatalf("a routine decision on a late approval question: %s", got)
	}
	job := ok[intent.JobReadResult](t)(r.send(t, intent.KindJobRead, intent.JobReadPayload{ID: rec.ID}, nil)).Job
	if job.State != intent.JobNeedsDecision || job.Question == nil || job.Question.ID != "one-0-pick" || job.Question.Tier != intent.TierApproval {
		t.Fatalf("the question did not stay open: %+v", job)
	}
}

// A question tier this agent has no signers for is never satisfied: an
// unknown tier fails closed rather than falling back to routine.
func TestAQuestionOfAnUnknownTierCannotBeDecided(t *testing.T) {
	r, run := jobsRig(t)
	rec := ok[intent.JobRecord](t)(r.send(t, jobstest.TestKind, map[string]string{"AskAt": "one"}, nil))
	_ = ok[intent.JobRecord](t)(r.send(t, intent.KindJobAdvance, intent.JobAdvancePayload{ID: rec.ID}, nil))
	run.Wait()
	_ = r.a.cfg.Jobs.Reconcile(context.Background())
	rec = ok[intent.JobReadResult](t)(r.send(t, intent.KindJobRead, intent.JobReadPayload{ID: rec.ID}, nil)).Job
	setQuestionTier(t, r, rec.ID, "owner")
	rc, res := r.send(t, intent.KindJobDecide, intent.JobDecidePayload{ID: rec.ID, Decision: rec.Question.ID, Option: "go"}, nil)
	if got := rejection(t, rc, res).Code; got != intent.ReasonUnauthorizedKind {
		t.Fatalf("a question of an unknown tier: %s", got)
	}
}

// J10: a re-pair that rewrote node.json with the legacy names cannot undo
// a migration, for any kind that reads the node config.
func TestTheNamesMarkerWinsOverAStaleNodeJSON(t *testing.T) {
	r := newRig(t, true) // node.json names the legacy root
	r.a.cfg.NamesPath = filepath.Join(r.dir, "names.json")
	b, _ := json.Marshal(jobs.NamesMarker{Names: catalog.NamesCurrent, DataRootFrom: "/var/lib/valve-node-app", DataRootTo: "/var/lib/jumpgate-node"})
	_ = os.WriteFile(r.a.cfg.NamesPath, b, 0o600)
	info := ok[intent.AgentInfo](t)(r.send(t, intent.KindAgentInfo, struct{}{}, nil))
	if info.Names != catalog.NamesCurrent || info.DataDir != "/var/lib/jumpgate-node/369" || info.JWTPath != "/var/lib/jumpgate-node/369/jwt.hex" {
		t.Fatalf("agent.info %+v", info)
	}
	_, _ = r.send(t, intent.KindServiceAction, intent.ServiceActionPayload{Service: "exec", Action: "restart"}, nil)
	if last := r.exec.cmds[len(r.exec.cmds)-1]; !strings.Contains(last, "jumpgate-node-exec.service") {
		t.Fatalf("service.action ran %q", last)
	}
	r.exec.res = executor.Result{}
	_ = ok[LogsSinceResult](t)(r.send(t, intent.KindLogsSince, intent.LogsSincePayload{}, nil))
	if last := r.exec.cmds[len(r.exec.cmds)-1]; !strings.Contains(last, "jumpgate-node-exec.service") || strings.Contains(last, "valve-node-app") {
		t.Fatalf("logs.since ran %q", last)
	}
	n := len(r.exec.cmds)
	_, _ = r.send(t, intent.KindDiskRead, struct{}{}, nil) // the fake's output is no df; only the commands matter
	if disk := strings.Join(r.exec.cmds[n:], "\n"); !strings.Contains(disk, "/var/lib/jumpgate-node/369") {
		t.Fatalf("disk.read ran %q", disk)
	}
	for _, c := range r.exec.cmds {
		if strings.Contains(c, "/var/lib/valve-node-app") {
			t.Fatalf("a command still names the legacy data root: %q", c)
		}
	}
}

// I2: agent.info tells a broken node config from no node at all, so a
// controller never takes a box it cannot read for a new one.
func TestAgentInfoReportsAConfigItRefuses(t *testing.T) {
	r := newRig(t, false)
	if info := ok[intent.AgentInfo](t)(r.send(t, intent.KindAgentInfo, struct{}{}, nil)); info.SetUp || info.SetUpError != nil {
		t.Fatalf("no node.json: %+v", info)
	}
	if err := os.WriteFile(filepath.Join(r.dir, "node.json"), []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	info := ok[intent.AgentInfo](t)(r.send(t, intent.KindAgentInfo, struct{}{}, nil))
	if info.SetUp || info.SetUpError == nil || info.SetUpError.Code != intent.ReasonValidation {
		t.Fatalf("a broken node.json: %+v", info)
	}
	r = newRig(t, true)
	r.a.cfg.NamesPath = filepath.Join(r.dir, "names.json")
	_ = os.WriteFile(r.a.cfg.NamesPath, []byte(`{"names":"other"}`), 0o600)
	info = ok[intent.AgentInfo](t)(r.send(t, intent.KindAgentInfo, struct{}{}, nil))
	if info.SetUp || info.SetUpError == nil || !strings.Contains(info.SetUpError.Message, "names.json") {
		t.Fatalf("a foreign names.json: %+v", info)
	}
}

// A marker this agent could not have written is refused, never applied: it
// would move every path the agent acts on as root.
func TestAForeignNamesMarkerIsRefused(t *testing.T) {
	r := newRig(t, true)
	r.a.cfg.NamesPath = filepath.Join(r.dir, "names.json")
	b, _ := json.Marshal(jobs.NamesMarker{Names: catalog.NamesCurrent, DataRootFrom: "/var/lib/valve-node-app", DataRootTo: "/etc"})
	_ = os.WriteFile(r.a.cfg.NamesPath, b, 0o600)
	rc, res := r.send(t, intent.KindServiceAction, intent.ServiceActionPayload{Service: "exec", Action: "restart"}, nil)
	if got := rejection(t, rc, res); got.Code != intent.ReasonValidation || !strings.Contains(got.Message, "names.json") {
		t.Fatalf("a foreign marker: %+v", got)
	}
	if len(r.exec.cmds) != 0 {
		t.Fatalf("ran commands under a foreign marker: %v", r.exec.cmds)
	}
}

func TestJobOutputReadsTheJobsUnitsOnly(t *testing.T) {
	r, _ := jobsRig(t)
	rec := ok[intent.JobRecord](t)(r.send(t, jobstest.TestKind, map[string]string{}, nil))
	r.exec.res = executor.Result{Stdout: `{"__CURSOR":"s=1","__REALTIME_TIMESTAMP":"1800000000000000","_SYSTEMD_UNIT":"jumpgate-job-` + rec.ID + `-one-0.service","MESSAGE":"running one"}` + "\n"}
	out := ok[intent.JobOutputResult](t)(r.send(t, intent.KindJobOutput, intent.JobOutputPayload{ID: rec.ID, N: 50}, nil))
	if len(out.Lines) != 1 || out.Lines[0].Line != "running one" || out.Cursor != "s=1" {
		t.Fatalf("output %+v", out)
	}
	cmd := r.exec.cmds[len(r.exec.cmds)-1]
	if !strings.Contains(cmd, "-u 'jumpgate-job-"+rec.ID+"-*'") || !strings.Contains(cmd, "-o json") {
		t.Fatalf("command %q", cmd)
	}
	n := len(r.exec.cmds)
	for _, bad := range []intent.JobOutputPayload{{ID: "../../etc"}, {ID: rec.ID, Cursor: "x'; rm -rf /"}} {
		rc, res := r.send(t, intent.KindJobOutput, bad, nil)
		if got := rejection(t, rc, res).Code; got != intent.ReasonInvalidPayload {
			t.Fatalf("payload %+v: %s", bad, got)
		}
	}
	if len(r.exec.cmds) != n {
		t.Fatalf("a refused job.output ran %v", r.exec.cmds[n:])
	}
}

// D-J8 through the shared read (ruling P9): a job's output line is cut to
// logsSinceMaxLine exactly as a logs.since line is.
func TestJobOutputIsBoundedLikeLogsSince(t *testing.T) {
	r, _ := jobsRig(t)
	rec := ok[intent.JobRecord](t)(r.send(t, jobstest.TestKind, map[string]string{}, nil))
	r.exec.res = executor.Result{Stdout: `{"__CURSOR":"s=1","__REALTIME_TIMESTAMP":"1800000000000000","_SYSTEMD_UNIT":"jumpgate-job-` + rec.ID + `-one-0.service","MESSAGE":"` + strings.Repeat("x", 2*logsSinceMaxLine) + `"}` + "\n"}
	out := ok[intent.JobOutputResult](t)(r.send(t, intent.KindJobOutput, intent.JobOutputPayload{ID: rec.ID}, nil))
	if len(out.Lines) != 1 || len(out.Lines[0].Line) != logsSinceMaxLine {
		t.Fatalf("output %d lines, want 1 of %d bytes", len(out.Lines), logsSinceMaxLine)
	}
}

func TestAgentInfoListsPlanVersions(t *testing.T) {
	r, _ := jobsRig(t)
	info := ok[intent.AgentInfo](t)(r.send(t, intent.KindAgentInfo, struct{}{}, nil))
	found := false
	for _, v := range info.PlanVersions {
		found = found || v == jobs.MigrateNamesVersion
	}
	if !found {
		t.Fatalf("plan versions %v", info.PlanVersions)
	}
}

// Every job intent is refused before it reaches the manager when its id is
// not a job id: only a checked id can reach a path or a command line.
func TestJobIntentsRefuseABadID(t *testing.T) {
	r, _ := jobsRig(t)
	for kind, pl := range map[string]any{
		intent.KindJobRead:    intent.JobReadPayload{ID: "../x"},
		intent.KindJobLease:   intent.JobLeasePayload{ID: "ABCDEF0123456789", Op: intent.LeaseTake},
		intent.KindJobAdvance: intent.JobAdvancePayload{ID: "0123"},
		intent.KindJobDecide:  intent.JobDecidePayload{ID: "", Decision: "x", Option: "go"},
	} {
		rc, res := r.send(t, kind, pl, nil)
		if got := rejection(t, rc, res).Code; got != intent.ReasonInvalidPayload {
			t.Errorf("%s with a bad id: %s", kind, got)
		}
	}
}

// startEnvelope is a signed envelope of kind over body, which mutate may
// alter before it is signed.
func startEnvelope(t *testing.T, r *rig, kind string, body []byte, mutate func(*intent.Intent)) (intent.Intent, intent.Envelope) {
	t.Helper()
	r.seq++
	n, _ := intent.NewNonce()
	i := intent.Intent{Agent: r.agentKey.Address(), Controller: r.controller.Address(), Seq: r.seq, Nonce: n,
		IssuedAt: uint64(r.now.Unix()), Expiry: uint64(r.now.Unix() + 120), Kind: kind, PayloadHash: intent.Hash(body)}
	if mutate != nil {
		mutate(&i)
	}
	sig, _ := r.controller.SignTypedData(context.Background(), i.TypedData())
	return i, intent.Envelope{Intent: i.JSON(), Payload: body, Sigs: []string{sig.Hex()}}
}

// The start-job admission checks hold at the dispatcher itself, not only in
// Handle: a job's first log entry is the envelope that started it, so it must
// be signed by its controller, for this box, of the kind started, over the
// inputs passed. Each check is shown failing alone.
func TestStartingAJobChecksTheEnvelopeItLogs(t *testing.T) {
	r, _ := jobsRig(t)
	body := []byte(`{}`)
	ctl := []eip712.Address{r.controller.Address()}
	stranger, _ := signer.GenerateKey()
	pol, err := LoadPolicy(filepath.Join(r.dir, "policy.json"))
	if err != nil {
		t.Fatal(err)
	}
	_, good := startEnvelope(t, r, jobstest.TestKind, body, nil)
	_, elsewhere := startEnvelope(t, r, jobstest.TestKind, body, func(i *intent.Intent) { i.Agent = eip712.Address{1} })
	if rej := r.a.checkStart(jobstest.TestKind, body, good, ctl); rej != nil {
		t.Fatalf("a good start: %+v", rej)
	}
	for _, c := range []struct {
		name     string
		kind     string
		inputs   []byte
		env      intent.Envelope
		signers  []eip712.Address
		wantCode string
	}{
		{"controller is not a signer", jobstest.TestKind, body, good, []eip712.Address{stranger.Address()}, intent.ReasonUnauthorizedKind},
		{"addressed to another box", jobstest.TestKind, body, elsewhere, ctl, intent.ReasonWrongAgent},
		{"another kind than the job's", intent.KindNodeMigrateNames, body, good, ctl, intent.ReasonInvalidPayload},
		{"inputs that are not the signed payload", jobstest.TestKind, []byte(`{"FailAt":"one"}`), good, ctl, intent.ReasonInvalidPayload},
	} {
		if rej := r.a.checkStart(c.kind, c.inputs, c.env, c.signers); rej == nil || rej.Code != c.wantCode {
			t.Errorf("%s: %+v, want %s", c.name, rej, c.wantCode)
		}
	}

	// dispatchJob runs the checks, and also refuses an envelope that is not
	// the intent Handle admitted.
	i, env := startEnvelope(t, r, jobstest.TestKind, body, nil)
	if _, _, rej := r.a.dispatchJob(context.Background(), i, good, ctl, pol); rej == nil || rej.Code != intent.ReasonInvalidPayload {
		t.Errorf("another intent's envelope: %+v", rej)
	}
	if _, _, rej := r.a.dispatchJob(context.Background(), i, env, []eip712.Address{stranger.Address()}, pol); rej == nil || rej.Code != intent.ReasonUnauthorizedKind {
		t.Errorf("an unsigned controller: %+v", rej)
	}
	iElsewhere, envElsewhere := startEnvelope(t, r, jobstest.TestKind, body, func(i *intent.Intent) { i.Agent = eip712.Address{1} })
	if _, _, rej := r.a.dispatchJob(context.Background(), iElsewhere, envElsewhere, ctl, pol); rej == nil || rej.Code != intent.ReasonWrongAgent {
		t.Errorf("a start for another box: %+v", rej)
	}
	if list, err := r.a.cfg.Jobs.List(); err != nil || len(list.Jobs) != 0 {
		t.Fatalf("a refused start created a job: %+v %v", list, err)
	}

	// End to end, Handle refuses the same mismatches before admission.
	for _, mutate := range []func(*intent.Intent){
		func(i *intent.Intent) { i.Agent = eip712.Address{1} },
		func(i *intent.Intent) { i.PayloadHash = intent.Hash([]byte(`{"FailAt":"one"}`)) },
		func(i *intent.Intent) { i.Controller = stranger.Address() },
	} {
		rc, res := r.send(t, jobstest.TestKind, struct{}{}, mutate)
		_ = rejection(t, rc, res)
	}
	if list, _ := r.a.cfg.Jobs.List(); len(list.Jobs) != 0 {
		t.Fatalf("a refused start created a job: %+v", list)
	}
}

// syncBuffer is a log sink RunJobs writes from its own goroutine.
type syncBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

// D-J2: the loop logs a failed pass and keeps going; the next pass folds
// what the failed one could not.
func TestRunJobsLogsAFailedPassAndKeepsGoing(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root reads a directory whatever its mode")
	}
	r, run := jobsRig(t)
	log := &syncBuffer{}
	r.a.cfg.Log = log
	rec := ok[intent.JobRecord](t)(r.send(t, jobstest.TestKind, map[string]string{}, nil))
	_ = ok[intent.JobRecord](t)(r.send(t, intent.KindJobAdvance, intent.JobAdvancePayload{ID: rec.ID}, nil))
	run.Wait()
	dir := filepath.Join(r.dir, "jobs")
	if err := os.Chmod(dir, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })

	old := ReconcileEvery
	ReconcileEvery = 10 * time.Millisecond
	t.Cleanup(func() { ReconcileEvery = old })
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { r.a.RunJobs(ctx); close(done) }()
	t.Cleanup(func() { cancel(); <-done })

	waitFor(t, func() bool { return strings.Contains(log.String(), "reconcile") })
	if err := os.Chmod(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	waitFor(t, func() bool {
		man, err := jobs.LoadManifest(dir, rec.ID)
		return err == nil && man.Step == 1 && man.State == intent.JobWaiting
	})
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatal("timed out")
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// I4: once a local operator abandons the box's job, service.action runs
// again.
func TestAnAbandonedJobFreesTheNodeServices(t *testing.T) {
	r, run := jobsRig(t)
	rec := ok[intent.JobRecord](t)(r.send(t, jobstest.TestKind, map[string]string{}, nil))
	rc, res := r.send(t, intent.KindServiceAction, intent.ServiceActionPayload{Service: "exec", Action: "restart"}, nil)
	if got := rejection(t, rc, res).Code; got != intent.ReasonJobActive {
		t.Fatalf("service.action during a job: %s", got)
	}
	dir := filepath.Join(r.dir, "jobs")
	cfg := jobs.ManagerConfig{Dir: dir, Key: r.agentKey, Runner: run, Now: func() time.Time { return r.now }}
	_ = r.a.cfg.Jobs.Close() // the agent stops
	if _, err := jobs.Abandon(context.Background(), cfg, rec.ID); err != nil {
		t.Fatal(err)
	}
	m, err := jobs.NewManager(cfg) // and starts again
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = m.Close() })
	r.a.cfg.Jobs = m
	rc, res = r.send(t, intent.KindServiceAction, intent.ServiceActionPayload{Service: "exec", Action: "restart"}, nil)
	if rc.Status != intent.StatusOK {
		t.Fatalf("service.action after the abandon: %d %s", rc.Status, res)
	}
}

// unavailableJobs is a rig whose job store did not open, for why.
func unavailableJobs(t *testing.T, why string) *rig {
	t.Helper()
	r := newRig(t, true)
	r.a.cfg.JobsDir = filepath.Join(r.dir, "jobs")
	r.a.cfg.JobsUnavailable = errors.New(why)
	return r
}

// M3, fail closed: a job store that did not open may still hold a job that
// owns the node services. While it has anything in it, service.action is
// refused (J1) and says why; job kinds are refused with the reason.
func TestAnUnopenedJobStoreWithJobsRefusesServiceActions(t *testing.T) {
	r := unavailableJobs(t, "permission denied")
	if err := os.MkdirAll(r.a.cfg.JobsDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(r.a.cfg.JobsDir, "0123456789abcdef.json"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	rc, res := r.send(t, intent.KindServiceAction, intent.ServiceActionPayload{Service: "exec", Action: "restart"}, nil)
	got := rejection(t, rc, res)
	if got.Code != intent.ReasonJobActive || !strings.Contains(got.Message, "job store") || !strings.Contains(got.Message, "permission denied") {
		t.Fatalf("service.action with an unopened job store: %+v", got)
	}
	if len(r.exec.cmds) != 0 {
		t.Fatalf("service.action ran commands: %v", r.exec.cmds)
	}
	rc, res = r.send(t, intent.KindJobRead, intent.JobReadPayload{ID: "0123456789abcdef"}, nil)
	if got := rejection(t, rc, res); got.Code != intent.ReasonJobsUnsupported || !strings.Contains(got.Message, "permission denied") {
		t.Fatalf("job.read with an unopened job store: %+v", got)
	}
	// job.list is how a controller decides whether setup or a clear may
	// run (C1); it must not read as "this box runs no jobs".
	rc, res = r.send(t, intent.KindJobList, struct{}{}, nil)
	if rc.Status != intent.StatusFailed || !strings.Contains(string(res), "permission denied") {
		t.Fatalf("job.list with an unopened job store: %d %s", rc.Status, res)
	}
}

// With no jobs directory at all there can be no job, so the node services
// are the operator's.
func TestAnUnopenedJobStoreWithoutAJobsDirAllowsServiceActions(t *testing.T) {
	r := unavailableJobs(t, "read-only file system")
	rc, res := r.send(t, intent.KindServiceAction, intent.ServiceActionPayload{Service: "exec", Action: "restart"}, nil)
	if rc.Status != intent.StatusOK {
		t.Fatalf("service.action with no jobs dir: %d %s", rc.Status, res)
	}
	rc, res = r.send(t, intent.KindJobList, struct{}{}, nil)
	if got := rejection(t, rc, res); got.Code != intent.ReasonJobsUnsupported || !strings.Contains(got.Message, "read-only file system") {
		t.Fatalf("job.list with no jobs dir: %+v", got)
	}
}
