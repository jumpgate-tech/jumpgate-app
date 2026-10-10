// internal/server/jobsdrive_test.go
package server

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"testing/fstest"
	"time"

	"github.com/valve-tech/jumpgate/internal/agent"
	"github.com/valve-tech/jumpgate/internal/agentclient"
	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/catalog"
	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/eip712"
	"github.com/valve-tech/jumpgate/internal/intent"
	"github.com/valve-tech/jumpgate/internal/jobs/jobstest"
	"github.com/valve-tech/jumpgate/internal/logwatch"
	"github.com/valve-tech/jumpgate/internal/signer"
)

// waitJob polls the inbox until the job reaches state or the test times out.
func waitJob(t *testing.T, b *boxRig, id, state string) api.JobView {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		var in api.JobsInbox
		b.do(t, "GET", "/api/jobs", nil, &in)
		for _, j := range in.Jobs {
			if j.ID == id && j.State == state {
				return j
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("job %s never reached %s", id, state)
	return api.JobView{}
}

// drivers is how many drivers the hub runs.
func (b *boxRig) drivers() int {
	b.srv.jobs.mu.Lock()
	defer b.srv.jobs.mu.Unlock()
	return len(b.srv.jobs.driving)
}

// raiseQuestionTier rewrites the open question of job id, in the agent's own
// manifest, to need an approval-tier signature.
func raiseQuestionTier(t *testing.T, b *boxRig, id string) {
	t.Helper()
	path := filepath.Join(b.home, "jobs", id+".json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var man map[string]any
	if err := json.Unmarshal(raw, &man); err != nil {
		t.Fatal(err)
	}
	q, ok := man["question"].(map[string]any)
	if !ok {
		t.Fatalf("job %s has no open question", id)
	}
	q["tier"] = intent.TierApproval
	raw, _ = json.Marshal(man)
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
}

// ageLease moves job id's lease heartbeat, in the agent's own manifest, past
// the TTL by the box's clock (moving the clock again would expire intents).
func ageLease(t *testing.T, b *boxRig, id string) {
	t.Helper()
	path := filepath.Join(b.home, "jobs", id+".json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var man map[string]any
	if err := json.Unmarshal(raw, &man); err != nil {
		t.Fatal(err)
	}
	man["lease"].(map[string]any)["heartbeatAt"] = b.clock().Add(-91 * time.Second).Unix()
	raw, _ = json.Marshal(man)
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
}

// takeLeaseAsAnotherController enrolls a second controller key on the box,
// lets the lease go stale by the box's clock, and takes it over the agent
// directly, past this server.
func takeLeaseAsAnotherController(t *testing.T, b *boxRig, id string) {
	t.Helper()
	other, err := signer.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	policy := filepath.Join(b.home, "policy.json")
	p, err := agent.LoadPolicy(policy)
	if err != nil {
		t.Fatal(err)
	}
	p.AddSigner(agent.SignerEntry{Address: other.Address().Hex(), Tier: agent.TierRoutine, Label: "other"})
	if err := p.Save(policy); err != nil {
		t.Fatal(err)
	}
	b.advance(91 * time.Second)
	cfg, _ := b.srv.loadConfig()
	tg, _ := findTarget(cfg, "box")
	addr, err := eip712.ParseAddress(tg.Agent.Address)
	if err != nil {
		t.Fatal(err)
	}
	c, err := agentclient.Dial(context.Background(), agentclient.Target{Agent: addr, Socket: tg.Agent.Socket, Local: true}, other, agentclient.NewMemorySeqStore())
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	res, err := c.Do(context.Background(), intent.KindJobLease, intent.JobLeasePayload{ID: id, Op: intent.LeaseTake})
	if err != nil || res.Rejection != nil {
		t.Fatalf("take: %v %+v", err, res.Rejection)
	}
}

func TestAStartedJobIsDrivenToTheEnd(t *testing.T) {
	requireAgentPeer(t)
	b := newBoxRig(t)
	var v api.JobView
	if code := b.do(t, "POST", "/api/jobs/box", api.JobStart{Kind: jobstest.TestKind, Inputs: json.RawMessage(`{}`)}, &v); code != 200 {
		t.Fatalf("start = %d", code)
	}
	if got := waitJob(t, b, v.ID, intent.JobSucceeded); !got.Lease.Mine {
		t.Fatalf("finished job %+v", got)
	}
}

func TestADecisionPointWaitsForAPersonThenContinues(t *testing.T) {
	requireAgentPeer(t)
	b := newBoxRig(t)
	var v api.JobView
	b.do(t, "POST", "/api/jobs/box", api.JobStart{Kind: jobstest.TestKind, Inputs: json.RawMessage(`{"AskAt":"two"}`)}, &v)
	asked := waitJob(t, b, v.ID, intent.JobNeedsDecision)
	if asked.Question == nil || !asked.Question.Signable {
		t.Fatalf("question %+v", asked.Question)
	}
	var after api.JobView
	if code := b.do(t, "POST", "/api/jobs/box/"+v.ID+"/decide", api.JobDecision{Decision: asked.Question.ID, Option: "go"}, &after); code != 200 {
		t.Fatalf("decide = %d", code)
	}
	waitJob(t, b, v.ID, intent.JobSucceeded)
}

// Q5: an approval-tier question is refused here, before anything is signed.
func TestApprovalQuestionsAreNotSignedYet(t *testing.T) {
	requireAgentPeer(t)
	b := newBoxRig(t)
	var v api.JobView
	b.do(t, "POST", "/api/jobs/box", api.JobStart{Kind: jobstest.TestKind, Inputs: json.RawMessage(`{"AskAt":"one"}`)}, &v)
	asked := waitJob(t, b, v.ID, intent.JobNeedsDecision)
	raiseQuestionTier(t, b, v.ID) // rewrites question.tier to approval in the agent's manifest
	b.srv.jobs.refresh(context.Background(), true)
	logLen := b.readOnBox(t, v.ID).LogLen
	var e api.Error
	if code := b.do(t, "POST", "/api/jobs/box/"+v.ID+"/decide", api.JobDecision{Decision: asked.Question.ID, Option: "go"}, &e); code != 409 || e.Code != api.CodeApprovalUnavailable {
		t.Fatalf("decide = %d %+v", code, e)
	}
	// Nothing reached the job: a decide the box took would be logged, and
	// reads and lease renewals are not (Q7).
	if got := b.readOnBox(t, v.ID); got.State != intent.JobNeedsDecision || got.LogLen != logLen {
		t.Fatalf("after the refusal: %s, log %d -> %d", got.State, logLen, got.LogLen)
	}
}

// Review Focus 3: two controllers, one job.
func TestSecondControllerWatchesThenAdoptsAStaleJob(t *testing.T) {
	requireAgentPeer(t)
	a := newBoxRig(t)
	var v api.JobView
	a.do(t, "POST", "/api/jobs/box", api.JobStart{Kind: jobstest.TestKind, Inputs: json.RawMessage(`{"AskAt":"two"}`)}, &v)
	asked := waitJob(t, a, v.ID, intent.JobNeedsDecision)

	// Controller B: its own key, enrolled on the box, its own server.
	bKey, _ := signer.GenerateKey()
	p, _ := agent.LoadPolicy(filepath.Join(a.home, "policy.json"))
	p.AddSigner(agent.SignerEntry{Address: bKey.Address().Hex(), Tier: agent.TierRoutine, Label: "b"})
	_ = p.Save(filepath.Join(a.home, "policy.json"))
	bSrv := New(Config{Token: a.token, UI: fstest.MapFS{}, Signer: bKey})
	t.Cleanup(func() { bSrv.fleet.stop(); bSrv.jobs.stopWithin(time.Second) })
	bTS := httptest.NewServer(bSrv.Handler())
	t.Cleanup(bTS.Close)
	b := &boxRig{ts: bTS, token: a.token, srv: bSrv, home: a.home, target: "box"}

	var in api.JobsInbox
	b.do(t, "GET", "/api/jobs", nil, &in)
	if len(in.Jobs) != 1 || in.Jobs[0].Lease.Mine || !in.Jobs[0].Lease.Fresh {
		t.Fatalf("B's view %+v", in.Jobs)
	}
	if n := b.drivers(); n != 0 {
		t.Fatalf("B drives a job A holds (%d drivers)", n)
	}
	var e api.Error
	if code := b.do(t, "POST", "/api/jobs/box/"+v.ID+"/decide", api.JobDecision{Decision: asked.Question.ID, Option: "go"}, &e); code != 409 || e.Code != api.CodeLeaseHeld {
		t.Fatalf("B decided a job A holds: %d %+v", code, e)
	}
	if code := b.do(t, "POST", "/api/jobs/box/"+v.ID+"/adopt", nil, &e); code != 409 || e.Code != api.CodeLeaseHeld {
		t.Fatalf("B adopted a fresh lease: %d %+v", code, e)
	}

	a.srv.jobs.stopWithin(time.Second) // A goes away
	a.advance(91 * time.Second)        // and stays silent past the TTL, by the box's clock
	// Q4: a stale lease is not taken by B's driver on its own; only adopt does.
	b.do(t, "GET", "/api/jobs", nil, &in)
	if len(in.Jobs) != 1 || in.Jobs[0].Lease.Mine || in.Jobs[0].Lease.Fresh || b.drivers() != 0 {
		t.Fatalf("B's view of a stale lease %+v (%d drivers)", in.Jobs, b.drivers())
	}
	var got api.JobView
	if code := b.do(t, "POST", "/api/jobs/box/"+v.ID+"/adopt", nil, &got); code != 200 || !got.Lease.Mine {
		t.Fatalf("adopt of a stale lease: %d %+v", code, got)
	}
	if code := b.do(t, "POST", "/api/jobs/box/"+v.ID+"/decide", api.JobDecision{Decision: asked.Question.ID, Option: "go"}, &got); code != 200 {
		t.Fatalf("B's decision after adopting: %d", code)
	}
	waitJob(t, b, v.ID, intent.JobSucceeded)
}

// After B adopts, A's driver (still running, its server never stopped)
// stops on the lease B now holds instead of fighting for it.
func TestTheFirstControllersDriverYieldsToAnAdoption(t *testing.T) {
	requireAgentPeer(t)
	// A renews only when an advance has not for jobsRenewEvery; the last
	// advance (to step two) precedes the question, so with an hour here A
	// sends no renewal before B adopts, however slow the machine.
	prevRenew := jobsRenewEvery
	jobsRenewEvery = time.Hour
	t.Cleanup(func() { jobsRenewEvery = prevRenew })
	a := newBoxRig(t)
	var v api.JobView
	a.do(t, "POST", "/api/jobs/box", api.JobStart{Kind: jobstest.TestKind, Inputs: json.RawMessage(`{"AskAt":"two"}`)}, &v)
	waitJob(t, a, v.ID, intent.JobNeedsDecision)
	if a.drivers() != 1 {
		t.Fatalf("A runs %d drivers", a.drivers())
	}
	bKey, _ := signer.GenerateKey()
	p, _ := agent.LoadPolicy(filepath.Join(a.home, "policy.json"))
	p.AddSigner(agent.SignerEntry{Address: bKey.Address().Hex(), Tier: agent.TierRoutine, Label: "b"})
	_ = p.Save(filepath.Join(a.home, "policy.json"))
	bSrv := New(Config{Token: a.token, UI: fstest.MapFS{}, Signer: bKey})
	t.Cleanup(func() { bSrv.fleet.stop(); bSrv.jobs.stopWithin(time.Second) })
	bTS := httptest.NewServer(bSrv.Handler())
	t.Cleanup(bTS.Close)
	b := &boxRig{ts: bTS, token: a.token, srv: bSrv, home: a.home, target: "box"}

	a.advance(91 * time.Second) // A sends no renewal (jobsRenewEvery above)
	var got api.JobView
	if code := b.do(t, "POST", "/api/jobs/box/"+v.ID+"/adopt", nil, &got); code != 200 || !got.Lease.Mine {
		t.Fatalf("adopt: %d %+v", code, got)
	}
	deadline := time.Now().Add(5 * time.Second)
	for a.drivers() != 0 {
		if time.Now().After(deadline) {
			t.Fatal("A's driver kept running after B adopted the job")
		}
		time.Sleep(10 * time.Millisecond)
	}
	var in api.JobsInbox
	a.do(t, "GET", "/api/jobs", nil, &in)
	if len(in.Jobs) != 1 || in.Jobs[0].Lease.Mine || a.drivers() != 0 {
		t.Fatalf("A's view after the adoption %+v (%d drivers)", in.Jobs, a.drivers())
	}
}

// readOnBox reads a job straight from the agent, past the hub: nothing is
// listed, so nothing resumes as a side effect (GET /api/jobs would).
func (b *boxRig) readOnBox(t *testing.T, id string) intent.JobRecord {
	t.Helper()
	cfg, _ := b.srv.loadConfig()
	tg, _ := findTarget(cfg, "box")
	res, err := decodeResult[intent.JobReadResult](b.srv.agentResult(context.Background(), cfg, tg, intent.KindJobRead, intent.JobReadPayload{ID: id}))
	if err != nil {
		t.Fatal(err)
	}
	return res.Job
}

// startJobUnlisted creates a job through the agent without letting the hub
// see it, as a server that died just after starting it would leave it.
func (b *boxRig) startJobUnlisted(t *testing.T, inputs string) intent.JobRecord {
	t.Helper()
	var rec intent.JobRecord
	if err := json.Unmarshal(b.send(t, jobstest.TestKind, json.RawMessage(inputs)), &rec); err != nil {
		t.Fatal(err)
	}
	return rec
}

// Q4: after a restart a controller drives its own jobs again, by itself.
// Ruling P4: the test makes no HTTP request, because GET /api/jobs resumes
// jobs on its own; only sweep can have started this driver.
func TestARestartedServerResumesItsOwnJobs(t *testing.T) {
	requireAgentPeer(t)
	b := newBoxRig(t)
	rec := b.startJobUnlisted(t, `{}`) // created, never driven: the old server "died" at once
	time.Sleep(5 * jobsDriveEvery)
	n := b.drivers()
	if got := b.readOnBox(t, rec.ID); n != 0 || got.State != intent.JobWaiting || got.Step != 0 {
		t.Fatalf("before the sweep: %d drivers, job %s at step %d", n, got.State, got.Step)
	}
	b.srv.jobs.sweep(context.Background())
	deadline := time.Now().Add(10 * time.Second)
	for b.readOnBox(t, rec.ID).State != intent.JobSucceeded {
		if time.Now().After(deadline) {
			t.Fatal("the sweep did not resume the job")
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// The serve paths sweep on their own as they start (Q4), before any request.
func TestServingStartsTheSweep(t *testing.T) {
	requireAgentPeer(t)
	b := newBoxRig(t)
	rec := b.startJobUnlisted(t, `{}`)
	ctx, cancel := context.WithCancel(context.Background())
	sock := filepath.Join(shortTempDir(t, "jgs"), "s.sock")
	done := make(chan error, 1)
	go func() { done <- b.srv.ServeUnix(ctx, sock) }()
	defer func() { cancel(); <-done }()
	deadline := time.Now().Add(10 * time.Second)
	for b.readOnBox(t, rec.ID).State != intent.JobSucceeded {
		if time.Now().After(deadline) {
			t.Fatal("serving did not resume the job")
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// Ruling P7: the start route is not a generic intent proxy. A kind outside
// jobStartKinds is refused before anything is signed or sent.
func TestStartRefusesKindsThatAreNotJobs(t *testing.T) {
	requireAgentPeer(t)
	b := newBoxRig(t)
	before, _ := config.Load()
	for _, kind := range []string{intent.KindServiceAction, intent.KindJobDecide, "test.unlisted"} {
		var e api.Error
		in := api.JobStart{Kind: kind, Inputs: json.RawMessage(`{"service":"exec","action":"stop"}`)}
		if code := b.do(t, "POST", "/api/jobs/box", in, &e); code != 400 || e.Code != api.CodeJobKindNotAllowed {
			t.Fatalf("start of %s = %d %+v", kind, code, e)
		}
	}
	after, _ := config.Load()
	if after.Targets[0].Agent.NextSeq != before.Targets[0].Agent.NextSeq {
		t.Fatalf("an intent was signed and sent (next seq %d -> %d)", before.Targets[0].Agent.NextSeq, after.Targets[0].Agent.NextSeq)
	}
}

// The production allowlist is node.migrate-names alone.
func TestTheOnlyJobKindStartedIsMigrateNames(t *testing.T) {
	if len(jobStartKinds) != 1 || !jobStartKinds[intent.KindNodeMigrateNames] {
		t.Fatalf("jobStartKinds = %v", jobStartKinds)
	}
}

func TestADriverStopsWhenItsLeaseIsTaken(t *testing.T) {
	requireAgentPeer(t)
	b := newBoxRig(t)
	rec := b.startJob(t, `{"AskAt":"one"}`)
	takeLeaseAsAnotherController(t, b, rec.ID) // enrolls a key, ages the lease, takes it over the agent directly
	b.srv.jobs.drive("box", rec.ID)
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if b.drivers() == 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("the driver kept running after another controller took the lease")
}

// A controller that sees a job it does not hold never drives it, not even
// once its lease is stale: taking a lease over is an operator's adopt (Q4).
func TestAStaleLeaseOfAnotherControllerIsNotResumed(t *testing.T) {
	requireAgentPeer(t)
	b := newBoxRig(t)
	rec := b.startJob(t, `{"AskAt":"one"}`)
	takeLeaseAsAnotherController(t, b, rec.ID)
	deadline := time.Now().Add(5 * time.Second)
	for b.drivers() != 0 {
		if time.Now().After(deadline) {
			t.Fatal("the driver kept running after another controller took the lease")
		}
		time.Sleep(10 * time.Millisecond)
	}
	ageLease(t, b, rec.ID) // now the other controller's lease is stale too
	b.srv.jobs.refresh(context.Background(), true)
	time.Sleep(5 * jobsDriveEvery)
	if n := b.drivers(); n != 0 {
		t.Fatalf("%d drivers for a job another controller holds", n)
	}
	var in api.JobsInbox
	b.do(t, "GET", "/api/jobs", nil, &in)
	if len(in.Jobs) != 1 || in.Jobs[0].State == intent.JobDamaged || in.Jobs[0].Lease.Fresh || in.Jobs[0].Lease.Mine || b.drivers() != 0 {
		t.Fatalf("the inbox did not show a stale lease left alone: %+v", in.Jobs)
	}
	if got := b.readOnBox(t, rec.ID); b.srv.jobs.mine(got) {
		t.Fatalf("this controller took the lease: %+v", got.Lease)
	}
}

// legacyBox makes the box and this controller's record of it legacy: a
// node.json under the legacy names, and a stored wire to match.
func legacyBox(t *testing.T, b *boxRig) {
	t.Helper()
	legacy := catalog.WireConfig{ChainID: 369, ExecID: "reth", BeaconID: "lighthouse-pulse", DataDir: "/var/lib/valve-node-app/369", JWTPath: "/var/lib/valve-node-app/369/jwt.hex"}
	raw, _ := json.Marshal(legacy)
	if err := os.WriteFile(filepath.Join(b.home, "node.json"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := config.Update(func(c *config.Config) error {
		w := legacy
		c.Targets[0].Wire = &w
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

// migrated writes the names marker node.migrate-names leaves on the box,
// data root moved, which agent.info applies over node.json.
func migrated(t *testing.T, b *boxRig) {
	t.Helper()
	raw, _ := json.Marshal(map[string]string{"names": catalog.NamesCurrent,
		"dataRootFrom": catalog.LegacyNames.DataRoot, "dataRootTo": catalog.CurrentNames.DataRoot})
	if err := os.WriteFile(filepath.Join(b.home, "names.json"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
}

var migrateSucceeded = intent.JobRecord{ID: "00000000000000aa", Kind: intent.KindNodeMigrateNames, State: intent.JobSucceeded,
	Result: json.RawMessage(`{"names":"jumpgate-node","dataDir":"/var/lib/jumpgate-node/369","jwtPath":"/var/lib/jumpgate-node/369/jwt.hex"}`)}

// The new names, data dir and JWT path come from the box's agent.info, and
// are saved in the config.
func TestMigrateNamesSuccessUpdatesTheTargetsWire(t *testing.T) {
	requireAgentPeer(t)
	b := newBoxRig(t)
	legacyBox(t, b)
	migrated(t, b)
	cfg, _ := config.Load()
	b.srv.jobs.finished(cfg.Targets[0], migrateSucceeded)
	cfg, _ = config.Load()
	w := cfg.Targets[0].Wire
	if w.Names != catalog.NamesCurrent || w.DataDir != "/var/lib/jumpgate-node/369" || w.JWTPath != "/var/lib/jumpgate-node/369/jwt.hex" {
		t.Fatalf("wire after migrate %+v", w)
	}
	if w.ChainID != 369 || w.ExecID != "reth" || w.BeaconID != "lighthouse-pulse" {
		t.Fatalf("the rest of the wire changed: %+v", w)
	}
	if units := logUnitsFor(cfg.Targets[0]); units[0] != catalog.CurrentNames.Units()[0] {
		t.Fatalf("log units after migrate %v", units)
	}
}

// The box is the source: a job result that disagrees with agent.info does
// not reach the config.
func TestMigrateNamesWireComesFromTheBoxNotTheResult(t *testing.T) {
	requireAgentPeer(t)
	b := newBoxRig(t)
	legacyBox(t, b)
	raw, _ := json.Marshal(map[string]string{"names": catalog.NamesCurrent}) // names only: the data stayed put
	_ = os.WriteFile(filepath.Join(b.home, "names.json"), raw, 0o600)
	cfg, _ := config.Load()
	b.srv.jobs.finished(cfg.Targets[0], migrateSucceeded)
	cfg, _ = config.Load()
	if w := cfg.Targets[0].Wire; w.Names != catalog.NamesCurrent || w.DataDir != "/var/lib/valve-node-app/369" || w.JWTPath != "/var/lib/valve-node-app/369/jwt.hex" {
		t.Fatalf("wire %+v", w)
	}
}

// Only a succeeded node.migrate-names changes the record of the box, and a
// box that cannot answer leaves the job to settle on a later listing.
func TestOnlyASucceededMigrationChangesTheWire(t *testing.T) {
	requireAgentPeer(t)
	b := newBoxRig(t)
	legacyBox(t, b)
	migrated(t, b)
	cfg, _ := config.Load()
	for _, rec := range []intent.JobRecord{
		{ID: "00000000000000a1", Kind: intent.KindNodeMigrateNames, State: intent.JobFailed},
		{ID: "00000000000000a2", Kind: intent.KindNodeMigrateNames, State: intent.JobAborted},
		{ID: "00000000000000a3", Kind: jobstest.TestKind, State: intent.JobSucceeded},
	} {
		b.srv.jobs.finished(cfg.Targets[0], rec)
	}
	if c, _ := config.Load(); c.Targets[0].Wire.Names != "" {
		t.Fatalf("wire changed by a job that did not migrate: %+v", c.Targets[0].Wire)
	}
	// An unreadable names marker: agent.info refuses, nothing is saved, and
	// the job is not marked settled.
	_ = os.WriteFile(filepath.Join(b.home, "names.json"), []byte(`{"names":"nonsense"}`), 0o600)
	b.srv.jobs.finished(cfg.Targets[0], migrateSucceeded)
	if c, _ := config.Load(); c.Targets[0].Wire.Names != "" {
		t.Fatalf("wire changed from a box that refused: %+v", c.Targets[0].Wire)
	}
	migrated(t, b)
	b.srv.jobs.finished(cfg.Targets[0], migrateSucceeded)
	if c, _ := config.Load(); c.Targets[0].Wire.Names != catalog.NamesCurrent {
		t.Fatalf("the retry did not settle the job: %+v", c.Targets[0].Wire)
	}
}

// The log watcher (and the monitor) were built from the legacy names; once
// the wire changes they are retired, so the next viewer tails the new units.
func TestMigrateNamesRetiresTheLogWatcher(t *testing.T) {
	requireAgentPeer(t)
	b := newBoxRig(t)
	legacyBox(t, b)
	migrated(t, b)
	var watchStopped, monStopped, followerStopped bool
	e := b.srv.reg.get("box")
	e.mu.Lock()
	e.watch = logwatch.New(nopExec{}, catalog.LegacyNames.Units())
	e.watchStop = func() { watchStopped = true }
	e.monStop = func() { monStopped = true }
	// A paired box's logs come through its follower, not the watcher.
	e.logs = newLogFollower(func() { followerStopped = true })
	e.mu.Unlock()
	cfg, _ := config.Load()
	b.srv.jobs.finished(cfg.Targets[0], migrateSucceeded)
	e.mu.Lock()
	defer e.mu.Unlock()
	if !watchStopped || e.watch != nil {
		t.Fatalf("the legacy log watcher is still running (stopped %v)", watchStopped)
	}
	if !monStopped || e.monStop != nil || !followerStopped || e.logs != nil {
		t.Fatalf("monitor stopped %v, log follower stopped %v", monStopped, followerStopped)
	}
}

// A migration that finished while no driver ran (this controller was down,
// or another controller drove it) is settled by the next listing.
func TestAListingSettlesAMigrationThatFinishedUnwatched(t *testing.T) {
	requireAgentPeer(t)
	b := newBoxRig(t)
	legacyBox(t, b)
	migrated(t, b)
	b.srv.jobs.afterList(config.Target{ID: "box"}, []intent.JobRecord{migrateSucceeded})
	deadline := time.Now().Add(5 * time.Second)
	for {
		c, _ := config.Load()
		if c.Targets[0].Wire.Names == catalog.NamesCurrent {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("wire %+v", c.Targets[0].Wire)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// A server that holds no job keeps no driver after it stops (stopWithin).
func TestStopEndsEveryDriver(t *testing.T) {
	requireAgentPeer(t)
	b := newBoxRig(t)
	var v api.JobView
	b.do(t, "POST", "/api/jobs/box", api.JobStart{Kind: jobstest.TestKind, Inputs: json.RawMessage(`{"AskAt":"one"}`)}, &v)
	waitJob(t, b, v.ID, intent.JobNeedsDecision)
	if b.drivers() != 1 {
		t.Fatalf("%d drivers", b.drivers())
	}
	if !b.srv.jobs.stopWithin(time.Second) {
		t.Fatal("drivers outlived stopWithin")
	}
	if b.drivers() != 0 {
		t.Fatalf("%d drivers after stop", b.drivers())
	}
	b.srv.jobs.drive("box", v.ID) // a stopped hub starts none
	if b.drivers() != 0 {
		t.Fatal("a stopped hub started a driver")
	}
}

// I1: a box that refuses the driver (here this controller's key was revoked:
// unauthorized_kind) ends it, rather than being sent a newly signed intent
// every few seconds for as long as the process runs.
func TestADriverStopsWhenTheBoxRefusesIt(t *testing.T) {
	requireAgentPeer(t)
	b := newBoxRig(t)
	var v api.JobView
	b.do(t, "POST", "/api/jobs/box", api.JobStart{Kind: jobstest.TestKind, Inputs: json.RawMessage(`{"AskAt":"one"}`)}, &v)
	waitJob(t, b, v.ID, intent.JobNeedsDecision)
	if b.drivers() != 1 {
		t.Fatalf("%d drivers", b.drivers())
	}
	policy := filepath.Join(b.home, "policy.json")
	p, err := agent.LoadPolicy(policy)
	if err != nil {
		t.Fatal(err)
	}
	me := b.srv.cfg.Signer.Address().Hex()
	var kept []agent.SignerEntry
	for _, s := range p.Signers {
		if s.Address != me {
			kept = append(kept, s)
		}
	}
	p.Signers = kept
	if err := p.Save(policy); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for b.drivers() != 0 {
		if time.Now().After(deadline) {
			t.Fatal("the driver kept running against a box that refuses it")
		}
		time.Sleep(10 * time.Millisecond)
	}
	before, _ := config.Load()
	time.Sleep(10 * jobsDriveEvery)
	if after, _ := config.Load(); after.Targets[0].Agent.NextSeq != before.Targets[0].Agent.NextSeq || b.drivers() != 0 {
		t.Fatalf("intents were still signed after the refusal (next seq %d -> %d)", before.Targets[0].Agent.NextSeq, after.Targets[0].Agent.NextSeq)
	}
}

// I1: a box lists what it likes; an id that is not a job id is never driven,
// even under this controller's lease.
func TestAnInvalidJobIDIsNeverDriven(t *testing.T) {
	requireAgentPeer(t)
	b := newBoxRig(t)
	for _, id := range []string{"not-an-id", "../../etc", "", "0123456789ABCDEF", "0123456789abcdef0"} {
		b.srv.jobs.drive("box", id)
		rec := intent.JobRecord{ID: id, State: intent.JobWaiting, Lease: intent.JobLease{Owner: b.srv.cfg.Signer.Address().Hex()}}
		b.srv.jobs.afterList(config.Target{ID: "box"}, []intent.JobRecord{rec})
	}
	if n := b.drivers(); n != 0 {
		t.Fatalf("%d drivers for ids that are not job ids", n)
	}
}

// I1: a box that cannot be reached is asked again with backoff, not every
// jobsDriveEvery.
func TestADriverBacksOffWhileTheBoxIsUnreachable(t *testing.T) {
	requireAgentPeer(t)
	b := newBoxRig(t)
	var reads atomic.Int32
	read := b.srv.jobs.readJob // set before any driver runs
	b.srv.jobs.readJob = func(ctx context.Context, cfg config.Config, tg config.Target, p intent.JobReadPayload) (intent.JobReadResult, error) {
		reads.Add(1)
		return read(ctx, cfg, tg, p)
	}
	var v api.JobView
	b.do(t, "POST", "/api/jobs/box", api.JobStart{Kind: jobstest.TestKind, Inputs: json.RawMessage(`{"AskAt":"one"}`)}, &v)
	waitJob(t, b, v.ID, intent.JobNeedsDecision)
	if _, err := config.Update(func(c *config.Config) error {
		c.Targets[0].Agent.Socket = filepath.Join(b.home, "gone.sock")
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	time.Sleep(5 * jobsDriveEvery) // the driver's first failures
	reads.Store(0)
	window := 30 * jobsDriveEvery
	time.Sleep(window)
	// Without backoff: about 30 reads. With it, the wait has passed 4x
	// jobsDriveEvery by now and doubles from there: at most a few.
	if n := reads.Load(); n > 6 || b.drivers() != 1 {
		t.Fatalf("%d reads of an unreachable box in %v (%d drivers)", n, window, b.drivers())
	}
}

// M1: a box now set up for another chain is not the node the stored wire
// describes; the wire is left alone and the job settled.
func TestAMigrationOnAnotherChainLeavesTheWire(t *testing.T) {
	requireAgentPeer(t)
	b := newBoxRig(t)
	legacyBox(t, b)
	migrated(t, b)
	if _, err := config.Update(func(c *config.Config) error { c.Targets[0].Wire.ChainID = 943; return nil }); err != nil {
		t.Fatal(err)
	}
	cfg, _ := config.Load()
	b.srv.jobs.finished(cfg.Targets[0], migrateSucceeded)
	if c, _ := config.Load(); c.Targets[0].Wire.Names != "" || c.Targets[0].Wire.DataDir != "/var/lib/valve-node-app/369" {
		t.Fatalf("wire took another chain's names: %+v", c.Targets[0].Wire)
	}
	if !b.srv.jobs.settled[jobKey{"box", migrateSucceeded.ID}] {
		t.Fatal("the job is not settled: every listing would ask the box again")
	}
}

// M2: a box that reports no node set up settles the job unchanged, rather
// than costing an agent.info on every listing.
func TestAMigrationOnABoxNoLongerSetUpIsSettled(t *testing.T) {
	requireAgentPeer(t)
	b := newBoxRig(t)
	legacyBox(t, b)
	if err := os.Remove(filepath.Join(b.home, "node.json")); err != nil {
		t.Fatal(err)
	}
	cfg, _ := config.Load()
	b.srv.jobs.finished(cfg.Targets[0], migrateSucceeded)
	if c, _ := config.Load(); c.Targets[0].Wire.Names != "" {
		t.Fatalf("wire %+v", c.Targets[0].Wire)
	}
	if !b.srv.jobs.settled[jobKey{"box", migrateSucceeded.ID}] {
		t.Fatal("the job is not settled: every listing would ask the box again")
	}
}
