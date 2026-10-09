// internal/server/jobs_test.go
package server

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	"github.com/valve-tech/jumpgate/internal/agent"
	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/eip712"
	"github.com/valve-tech/jumpgate/internal/intent"
	"github.com/valve-tech/jumpgate/internal/joblog"
	"github.com/valve-tech/jumpgate/internal/jobs"
	"github.com/valve-tech/jumpgate/internal/jobs/jobstest"
	"github.com/valve-tech/jumpgate/internal/signer"
)

// boxRig is a paired box whose agent runs jobs in-process, on a clock the
// test moves. It holds the server, its token, the agent's home, key and clock.
type boxRig struct {
	ts     *httptest.Server
	token  string
	srv    *Server
	home   string
	target string
	key    *signer.Key
	mu     sync.Mutex
	now    time.Time
}

func (b *boxRig) clock() time.Time        { b.mu.Lock(); defer b.mu.Unlock(); return b.now }
func (b *boxRig) advance(d time.Duration) { b.mu.Lock(); b.now = b.now.Add(d); b.mu.Unlock() }

func newBoxRig(t *testing.T) *boxRig { return newBoxRigNamed(t, "box") }

// newBoxRigNamed is newBoxRig with the paired target saved as name.
func newBoxRigNamed(t *testing.T, name string) *boxRig {
	t.Helper()
	jobstest.RegisterPlan(t)
	b := &boxRig{now: time.Now(), target: name}
	prevDrive, prevList := jobsDriveEvery, jobsListEvery
	jobsDriveEvery, jobsListEvery = 20*time.Millisecond, 0
	t.Cleanup(func() { jobsDriveEvery, jobsListEvery = prevDrive, prevList })
	ts, token := pairedBoxWith(t, nopExec{}, pairedOpts{
		setUp:  true,
		target: func(tg *config.Target) { tg.ID = name },
		agent: func(home string, c *agent.Config) {
			b.home, b.key = home, c.Key
			dir := filepath.Join(home, "jobs")
			run := &jobstest.Runner{}
			run.Run = jobstest.InProcess(dir, jobs.Env{Exec: c.Exec, NodePath: c.NodePath, Out: io.Discard})
			m, err := jobs.NewManager(jobs.ManagerConfig{Dir: dir, Key: c.Key, Runner: run, Now: b.clock,
				StepArgv: func(a jobs.StepArgs) []string { return jobs.StepArgv("/jumpgate", home, home, a) }})
			if err != nil {
				t.Fatal(err)
			}
			c.Jobs, c.Now = m, b.clock
			p, _ := agent.LoadPolicy(c.PolicyPath)
			p.Kinds = map[string][]agent.Tier{jobstest.TestKind: {agent.TierRoutine}}
			_ = p.Save(c.PolicyPath)
		},
		onServer: func(s *Server) { b.srv = s },
	})
	b.ts, b.token = ts, token
	// GET /api/fleet starts the poller; nothing of it may outlive the test.
	t.Cleanup(func() { b.srv.fleet.stop(); b.srv.jobs.stopWithin(time.Second) })
	return b
}

func (b *boxRig) get(t *testing.T, path string) (int, []byte) {
	t.Helper()
	req, _ := http.NewRequest(http.MethodGet, b.ts.URL+path, nil)
	req.Header.Set("Authorization", "Bearer "+b.token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	return res.StatusCode, raw
}

func (b *boxRig) do(t *testing.T, path string, out any) int {
	t.Helper()
	code, raw := b.get(t, path)
	if out != nil {
		_ = json.Unmarshal(raw, out)
	}
	return code
}

// send sends one intent straight through the agent (Task 12 adds the routes
// that start and change jobs).
func (b *boxRig) send(t *testing.T, kind string, payload any) json.RawMessage {
	t.Helper()
	cfg, _ := b.srv.loadConfig()
	tg, _ := findTarget(cfg, b.target)
	raw, err := b.srv.agentResult(context.Background(), cfg, tg, kind, payload)
	if err != nil {
		t.Fatalf("%s: %v", kind, err)
	}
	return raw
}

// startJob creates a job from a real signed intent, so its id is that
// intent's (joblog.JobIDFor).
func (b *boxRig) startJob(t *testing.T, inputs string) intent.JobRecord {
	t.Helper()
	var rec intent.JobRecord
	if err := json.Unmarshal(b.send(t, jobstest.TestKind, json.RawMessage(inputs)), &rec); err != nil {
		t.Fatal(err)
	}
	return rec
}

func TestInboxListsEveryPairedBoxsJobs(t *testing.T) {
	requireAgentPeer(t)
	b := newBoxRig(t)
	rec := b.startJob(t, `{}`)
	var in api.JobsInbox
	if code := b.do(t, "/api/jobs", &in); code != 200 {
		t.Fatalf("GET /api/jobs = %d", code)
	}
	if len(in.Jobs) != 1 || in.Jobs[0].ID != rec.ID || in.Jobs[0].Target != "box" || !in.Jobs[0].Lease.Mine || !in.Jobs[0].Lease.Fresh {
		t.Fatalf("inbox %+v", in)
	}
	if len(in.Jobs[0].Steps) != 2 || in.Jobs[0].Steps[0].State != "waiting" {
		t.Fatalf("steps %+v", in.Jobs[0].Steps)
	}
}

// The server owns the inbox's order: needs-decision first, then unfinished,
// then finished, newest first within each, across boxes.
func TestInboxPutsDecisionsFirst(t *testing.T) {
	srv := New(Config{Token: NewSessionToken(), UI: fstest.MapFS{}})
	t.Cleanup(func() { srv.jobs.stopWithin(time.Second) })
	at := func(s int64) int64 { return 1_700_000_000 + s }
	srv.jobs.boxes["a"] = &boxJobs{jobs: []intent.JobRecord{
		{ID: "a-done-new", State: intent.JobSucceeded, CreatedAt: at(50)},
		{ID: "a-run-old", State: intent.JobRunning, CreatedAt: at(10)},
	}}
	srv.jobs.boxes["b"] = &boxJobs{jobs: []intent.JobRecord{
		{ID: "b-ask-oldest", State: intent.JobNeedsDecision, CreatedAt: at(1)},
		{ID: "b-wait-new", State: intent.JobWaiting, CreatedAt: at(40)},
		{ID: "b-failed", State: intent.JobFailed, CreatedAt: at(30)},
	}}
	var got []string
	for _, j := range srv.jobs.inbox().Jobs {
		got = append(got, j.ID)
	}
	want := "b-ask-oldest b-wait-new a-run-old a-done-new b-failed"
	if strings.Join(got, " ") != want {
		t.Fatalf("order %v, want %s", got, want)
	}
}

func TestJobDetailVerifiesTheLog(t *testing.T) {
	requireAgentPeer(t)
	b := newBoxRig(t)
	rec := b.startJob(t, `{}`)
	var d api.JobDetail
	if code := b.do(t, "/api/jobs/box/"+rec.ID, &d); code != 200 || len(d.Log) != 1 || d.Log[0].Kind != joblog.KindCreated {
		t.Fatalf("detail %d %+v", code, d)
	}
	// A second read verifies only what is new: the cache holds the prefix.
	b.send(t, intent.KindJobLease, intent.JobLeasePayload{ID: rec.ID, Op: intent.LeaseRelease})
	var after api.JobDetail
	if code := b.do(t, "/api/jobs/box/"+rec.ID, &after); code != 200 || len(after.Log) != 3 || after.Job.Lease.Owner != "" {
		t.Fatalf("after release %d %+v", code, after)
	}
}

func TestJobDetailOfAJobThatIsNotThere(t *testing.T) {
	requireAgentPeer(t)
	b := newBoxRig(t)
	for _, id := range []string{"0123456789abcdef", "not-an-id"} {
		var e api.Error
		if code := b.do(t, "/api/jobs/box/"+id, &e); code != http.StatusNotFound || e.Code != api.CodeJobNotFound {
			t.Fatalf("%s: %d %+v", id, code, e)
		}
	}
	var e api.Error
	if code := b.do(t, "/api/jobs/nobox/0123456789abcdef", &e); code != http.StatusNotFound || e.Code != api.CodeTargetNotFound {
		t.Fatalf("unknown box: %d %+v", code, e)
	}
}

// Review Focus 5: an edited entry on the box is refused, not shown.
func TestJobDetailRefusesABrokenLog(t *testing.T) {
	requireAgentPeer(t)
	b := newBoxRig(t)
	rec := b.startJob(t, `{}`)
	logPath := filepath.Join(b.home, "jobs", rec.ID+".log")
	raw, _ := os.ReadFile(logPath)
	var j intent.JobLogEntryJSON
	_ = json.Unmarshal([]byte(strings.TrimSpace(string(raw))), &j)
	j.Body = []byte(`{"outcome":"forged"}`)
	forged, _ := json.Marshal(j)
	_ = os.WriteFile(logPath, append(forged, '\n'), 0o600)

	code, body := b.get(t, "/api/jobs/box/"+rec.ID)
	var e api.Error
	_ = json.Unmarshal(body, &e)
	// The agent reports the job damaged and still hands over the log; the
	// controller's own check refuses it.
	if code != http.StatusBadGateway || e.Code != api.CodeJobLogInvalid || strings.Contains(string(body), `"log"`) {
		t.Fatalf("a forged log answered %d %s", code, body)
	}
}

// liar is an agent that answers job.read with the real answer, changed by
// lie: the box (or anything between) misreporting its own log.
func (b *boxRig) liar(lie func(res *intent.JobReadResult)) {
	real := b.srv.jobs.readJob
	b.srv.jobs.readJob = func(ctx context.Context, cfg config.Config, t config.Target, p intent.JobReadPayload) (intent.JobReadResult, error) {
		res, err := real(ctx, cfg, t, p)
		if err == nil {
			lie(&res)
		}
		return res, err
	}
}

// otherIntent is a genuine controller-signed request that is not this job's:
// a lease take for job, addressed to agentAddr.
func (b *boxRig) otherIntent(t *testing.T, agentAddr eip712.Address, job string) intent.Envelope {
	t.Helper()
	body := []byte(`{"id":"` + job + `","op":"take"}`)
	n, err := intent.NewNonce()
	if err != nil {
		t.Fatal(err)
	}
	now := uint64(b.clock().Unix())
	ctl := b.srv.cfg.Signer
	in := intent.Intent{Agent: agentAddr, Controller: ctl.Address(), Seq: 999, Nonce: n, IssuedAt: now, Expiry: now + 120,
		Kind: intent.KindJobLease, PayloadHash: intent.Hash(body)}
	sig, err := ctl.SignTypedData(context.Background(), in.TypedData())
	if err != nil {
		t.Fatal(err)
	}
	return intent.Envelope{Intent: in.JSON(), Payload: body, Sigs: []string{sig.Hex()}}
}

// appendSigned extends res's log with one intent entry the agent's own key
// signs, carrying env, and makes the record describe the longer log: only
// VerifyIntents can catch it.
func (b *boxRig) appendSigned(t *testing.T, res *intent.JobReadResult, env intent.Envelope) {
	t.Helper()
	es := make([]joblog.Entry, len(res.Entries))
	for i, j := range res.Entries {
		es[i], _ = joblog.FromJSON(j)
	}
	head, err := joblog.Verify(b.key.Address(), res.Job.ID, es, 0, [32]byte{})
	if err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(joblog.IntentBody{Envelope: env, Outcome: "lease take"})
	e, d, err := joblog.Sign(context.Background(), b.key, res.Job.ID, uint64(len(es)), head, joblog.KindIntent, body, b.clock())
	if err != nil {
		t.Fatal(err)
	}
	res.Entries = append(res.Entries, joblog.ToJSON(e))
	res.Job.LogLen, res.Job.LogHead = uint64(len(res.Entries)), "0x"+hex.EncodeToString(d[:])
}

// J11: whatever verifies as a prefix, a log that is not the whole log the
// job's record describes, or that carries a request not for this job and
// agent, is job_log_invalid, and the detail shows none of it.
func TestJobDetailRefusesALogTheRecordDoesNotDescribe(t *testing.T) {
	requireAgentPeer(t)
	b := newBoxRig(t)
	rec := b.startJob(t, `{}`)
	// A three-entry log: created, the release intent, lease.released.
	b.send(t, intent.KindJobLease, intent.JobLeasePayload{ID: rec.ID, Op: intent.LeaseRelease})
	real := b.srv.jobs.readJob
	stranger, _ := signer.GenerateKey()

	for _, c := range []struct {
		name string
		lie  func(*intent.JobReadResult)
		want string // what the refusal says, so each case is caught by its own check
	}{
		{"tail dropped, count kept", func(r *intent.JobReadResult) { r.Entries = r.Entries[:len(r.Entries)-1] }, "says its log has 3"},
		{"tail dropped, count lowered", func(r *intent.JobReadResult) {
			r.Entries = r.Entries[:len(r.Entries)-1]
			r.Job.LogLen--
		}, "says its log has 2"},
		{"head mismatch", func(r *intent.JobReadResult) { r.Job.LogHead = "0x" + strings.Repeat("ab", 32) }, "abab"},
		{"another job's log", func(r *intent.JobReadResult) { r.Job.ID = "fedcba9876543210" }, "answered with job"},
		{"an intent for another job", func(r *intent.JobReadResult) {
			b.appendSigned(t, r, b.otherIntent(t, b.key.Address(), "fedcba9876543210"))
		}, "is not for job"},
		{"an intent for another agent", func(r *intent.JobReadResult) {
			b.appendSigned(t, r, b.otherIntent(t, stranger.Address(), rec.ID))
		}, "addressed to agent"},
	} {
		t.Run(c.name, func(t *testing.T) {
			b.srv.jobs.readJob = real
			b.liar(c.lie)
			code, body := b.get(t, "/api/jobs/box/"+rec.ID)
			var e api.Error
			_ = json.Unmarshal(body, &e)
			if code != http.StatusBadGateway || e.Code != api.CodeJobLogInvalid || strings.Contains(string(body), `"log"`) || !strings.Contains(e.Message, c.want) {
				t.Fatalf("answered %d %s", code, body)
			}
			b.srv.jobs.mu.Lock()
			cached := b.srv.jobs.logs[jobKey{"box", rec.ID}]
			b.srv.jobs.mu.Unlock()
			if cached != nil {
				t.Fatal("a log that did not verify is still cached")
			}
		})
	}

	// The honest box again, paged one entry at a time: the whole log is read
	// before it is judged, and it verifies.
	b.srv.jobs.readJob = func(ctx context.Context, cfg config.Config, tg config.Target, p intent.JobReadPayload) (intent.JobReadResult, error) {
		res, err := real(ctx, cfg, tg, p)
		if err == nil && len(res.Entries) > 1 {
			res.Entries, res.More = res.Entries[:1], true
		}
		return res, err
	}
	var d api.JobDetail
	if code := b.do(t, "/api/jobs/box/"+rec.ID, &d); code != 200 || len(d.Log) != 3 {
		t.Fatalf("paged honest read %d %+v", code, d)
	}
}

func TestFleetRowsCarryJobCounts(t *testing.T) {
	requireAgentPeer(t)
	b := newBoxRig(t)
	b.startJob(t, `{}`)
	b.srv.jobs.refresh(context.Background(), true)
	s := b.srv.jobs.summary("box")
	if s == nil || s.Running != 1 || s.NeedsDecision != 0 {
		t.Fatalf("summary %+v", s)
	}
	var f api.Fleet
	b.do(t, "/api/fleet", &f)
	if row, ok := f.Row("box"); !ok || row.Jobs == nil || row.Jobs.Running != 1 {
		t.Fatalf("fleet row jobs %+v", row.Jobs)
	}
}

// A box whose agent runs without a job store has no summary: "unavailable",
// never 0.
func TestFleetRowOfABoxWithoutJobsHasNoCount(t *testing.T) {
	requireAgentPeer(t)
	var srv *Server
	pairedBoxWith(t, nopExec{}, pairedOpts{setUp: true, onServer: func(s *Server) { srv = s }})
	t.Cleanup(func() { srv.jobs.stopWithin(time.Second) })
	srv.jobs.refresh(context.Background(), true)
	if s := srv.jobs.summary("box"); s != nil {
		t.Fatalf("summary %+v", s)
	}
	if in := srv.jobs.inbox(); len(in.Jobs) != 0 || len(in.Errors) != 0 {
		t.Fatalf("inbox %+v", in)
	}
}

func TestJobsStreamSendsTheInbox(t *testing.T) {
	requireAgentPeer(t)
	b := newBoxRig(t)
	b.startJob(t, `{}`)
	b.srv.jobs.refresh(context.Background(), true)
	frames, _ := readFrames(t, b.ts, b.token, "/api/jobs/stream", 1)
	var in api.JobsInbox
	if err := json.Unmarshal([]byte(strings.TrimPrefix(frames[0], "data: ")), &in); err != nil || len(in.Jobs) != 1 {
		t.Fatalf("frame %q: %v", frames[0], err)
	}
}

// Route precedence: a box named "stream" keeps its job routes, and the inbox
// stream is never read as a box.
func TestJobRoutesAndATargetNamedStream(t *testing.T) {
	requireAgentPeer(t)
	b := newBoxRigNamed(t, "stream")
	rec := b.startJob(t, `{}`)
	var d api.JobDetail
	if code := b.do(t, "/api/jobs/stream/"+rec.ID, &d); code != 200 || d.Job.Target != "stream" || len(d.Log) != 1 {
		t.Fatalf("the box named stream: %d %+v", code, d)
	}
	b.srv.jobs.refresh(context.Background(), true)
	frames, h := readFrames(t, b.ts, b.token, "/api/jobs/stream", 1)
	var in api.JobsInbox
	if !strings.HasPrefix(h.Get("Content-Type"), "text/event-stream") ||
		json.Unmarshal([]byte(strings.TrimPrefix(frames[0], "data: ")), &in) != nil || len(in.Jobs) != 1 {
		t.Fatalf("the inbox stream: %v %q", h, frames)
	}
}

func TestJobOutputStreamOpens(t *testing.T) {
	requireAgentPeer(t)
	b := newBoxRig(t)
	rec := b.startJob(t, `{}`)
	if code, _ := b.get(t, "/api/jobs/box/not-an-id/output/stream"); code != http.StatusNotFound {
		t.Fatalf("a bad id = %d", code)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, b.ts.URL+"/api/jobs/box/"+rec.ID+"/output/stream", nil)
	req.Header.Set("Authorization", "Bearer "+b.token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 || !strings.HasPrefix(res.Header.Get("Content-Type"), "text/event-stream") {
		t.Fatalf("output stream %d %v", res.StatusCode, res.Header)
	}
}
