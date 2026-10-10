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
	"github.com/valve-tech/jumpgate/internal/executor"
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

// newBoxRig is a paired box whose agent runs jobs in-process. server adjusts
// the server's Config (withoutDriver, for one). The test kind may be started
// through POST /api/jobs/{target} for as long as the test runs.
func newBoxRig(t *testing.T, server ...func(*Config)) *boxRig {
	return newBoxRigNamed(t, "box", server...)
}

// withoutDriver keeps the hub listing and verifying but never driving, so a
// test can look at a job that stays where it was put.
func withoutDriver(c *Config) { c.noJobsDriver = true }

// newBoxRigNamed is newBoxRig with the paired target saved as name.
func newBoxRigNamed(t *testing.T, name string, server ...func(*Config)) *boxRig {
	t.Helper()
	return newBoxRigExec(t, name, nopExec{}, server...)
}

// newBoxRigExec is newBoxRigNamed with the box's commands answered by ex.
func newBoxRigExec(t *testing.T, name string, ex executor.Executor, server ...func(*Config)) *boxRig {
	t.Helper()
	jobstest.RegisterPlan(t)
	b := &boxRig{now: time.Now(), target: name}
	prevDrive, prevList, prevKinds := jobsDriveEvery, jobsListEvery, jobStartKinds
	jobsDriveEvery, jobsListEvery = 20*time.Millisecond, 0
	jobStartKinds = map[string]bool{intent.KindNodeMigrateNames: true, jobstest.TestKind: true}
	t.Cleanup(func() { jobsDriveEvery, jobsListEvery, jobStartKinds = prevDrive, prevList, prevKinds })
	ts, token := pairedBoxWith(t, ex, pairedOpts{
		setUp:  true,
		target: func(tg *config.Target) { tg.ID = name },
		agent: func(home string, c *agent.Config) {
			b.home, b.key = home, c.Key
			// The names marker node.migrate-names writes, which agent.info
			// applies (absent until a test writes one).
			c.NamesPath = filepath.Join(home, "names.json")
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
		server: func(c *Config) {
			for _, f := range server {
				f(c)
			}
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
	return b.req(t, http.MethodGet, path, nil)
}

// req sends one request with the session token; body, when not nil, is
// sent as JSON.
func (b *boxRig) req(t *testing.T, method, path string, body any) (int, []byte) {
	t.Helper()
	var rd io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		rd = strings.NewReader(string(raw))
	}
	req, _ := http.NewRequest(method, b.ts.URL+path, rd)
	req.Header.Set("Authorization", "Bearer "+b.token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	return res.StatusCode, raw
}

func (b *boxRig) do(t *testing.T, method, path string, body, out any) int {
	t.Helper()
	code, raw := b.req(t, method, path, body)
	if out != nil {
		_ = json.Unmarshal(raw, out)
	}
	return code
}

// send sends one intent straight through the agent (past the job routes).
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
	// Listed, as a job is once the hub has seen it: only a listed job's log
	// is cached.
	b.srv.jobs.refresh(context.Background(), true)
	return rec
}

func TestInboxListsEveryPairedBoxsJobs(t *testing.T) {
	requireAgentPeer(t)
	b := newBoxRig(t, withoutDriver)
	rec := b.startJob(t, `{}`)
	var in api.JobsInbox
	if code := b.do(t, "GET", "/api/jobs", nil, &in); code != 200 {
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
	b := newBoxRig(t, withoutDriver)
	rec := b.startJob(t, `{}`)
	var d api.JobDetail
	if code := b.do(t, "GET", "/api/jobs/box/"+rec.ID, nil, &d); code != 200 || len(d.Log) != 1 || d.Log[0].Kind != joblog.KindCreated {
		t.Fatalf("detail %d %+v", code, d)
	}
	// A second read verifies only what is new: the cache holds the prefix.
	b.send(t, intent.KindJobLease, intent.JobLeasePayload{ID: rec.ID, Op: intent.LeaseRelease})
	var after api.JobDetail
	if code := b.do(t, "GET", "/api/jobs/box/"+rec.ID, nil, &after); code != 200 || len(after.Log) != 3 || after.Job.Lease.Owner != "" {
		t.Fatalf("after release %d %+v", code, after)
	}
}

func TestJobDetailOfAJobThatIsNotThere(t *testing.T) {
	requireAgentPeer(t)
	b := newBoxRig(t, withoutDriver)
	for _, id := range []string{"0123456789abcdef", "not-an-id"} {
		var e api.Error
		if code := b.do(t, "GET", "/api/jobs/box/"+id, nil, &e); code != http.StatusNotFound || e.Code != api.CodeJobNotFound || e.Hint == "" {
			t.Fatalf("%s: %d %+v", id, code, e)
		}
	}
	var e api.Error
	if code := b.do(t, "GET", "/api/jobs/nobox/0123456789abcdef", nil, &e); code != http.StatusNotFound || e.Code != api.CodeTargetNotFound {
		t.Fatalf("unknown box: %d %+v", code, e)
	}
}

// Review Focus 5: an edited entry on the box is refused, not shown.
func TestJobDetailRefusesABrokenLog(t *testing.T) {
	requireAgentPeer(t)
	b := newBoxRig(t, withoutDriver)
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
	body, _ := json.Marshal(joblog.IntentBody{Envelope: env, Outcome: "lease take"})
	b.appendEntry(t, res, joblog.KindIntent, body)
}

// appendEntry extends res's log with one entry of kind over body, signed by
// the agent's own key and chained onto the log, and makes the record
// describe the longer log.
func (b *boxRig) appendEntry(t *testing.T, res *intent.JobReadResult, kind string, body []byte) {
	t.Helper()
	es := make([]joblog.Entry, len(res.Entries))
	for i, j := range res.Entries {
		es[i], _ = joblog.FromJSON(j)
	}
	head, err := joblog.Verify(b.key.Address(), res.Job.ID, es, 0, [32]byte{})
	if err != nil {
		t.Fatal(err)
	}
	e, d, err := joblog.Sign(context.Background(), b.key, res.Job.ID, uint64(len(es)), head, kind, body, b.clock())
	if err != nil {
		t.Fatal(err)
	}
	res.Entries = append(res.Entries, joblog.ToJSON(e))
	res.Job.LogLen, res.Job.LogHead = uint64(len(res.Entries)), "0x"+hex.EncodeToString(d[:])
}

// detailRefused asks for job id's detail and requires job_log_invalid with
// no log in the body, a message containing want, and nothing cached.
func (b *boxRig) detailRefused(t *testing.T, id, want string) {
	t.Helper()
	code, body := b.get(t, "/api/jobs/"+b.target+"/"+id)
	var e api.Error
	_ = json.Unmarshal(body, &e)
	if code != http.StatusBadGateway || e.Code != api.CodeJobLogInvalid || strings.Contains(string(body), `"log"`) || !strings.Contains(e.Message, want) {
		t.Fatalf("answered %d %s, want job_log_invalid saying %q", code, body, want)
	}
	if b.cached(id) != nil {
		t.Fatal("a log that did not verify is cached")
	}
}

func (b *boxRig) cached(id string) *verifiedLog {
	b.srv.jobs.mu.Lock()
	defer b.srv.jobs.mu.Unlock()
	return b.srv.jobs.logs[jobKey{b.target, id}]
}

// fullRead is the box's honest answer for job id, every entry in one page.
func (b *boxRig) fullRead(t *testing.T, id string) intent.JobReadResult {
	t.Helper()
	var res intent.JobReadResult
	if err := json.Unmarshal(b.send(t, intent.KindJobRead, intent.JobReadPayload{ID: id}), &res); err != nil {
		t.Fatal(err)
	}
	return res
}

// serve makes the box answer job.read from res, paged as a real one would,
// without asking it.
func (b *boxRig) serve(res intent.JobReadResult) {
	b.srv.jobs.readJob = func(_ context.Context, _ config.Config, _ config.Target, p intent.JobReadPayload) (intent.JobReadResult, error) {
		out := res
		out.Entries = nil
		if p.From < uint64(len(res.Entries)) {
			out.Entries = res.Entries[p.From:]
		}
		return out, nil
	}
}

// J11: whatever verifies as a prefix, a log that is not the whole log the
// job's record describes, or that carries a request not for this job and
// agent, is job_log_invalid, and the detail shows none of it.
func TestJobDetailRefusesALogTheRecordDoesNotDescribe(t *testing.T) {
	requireAgentPeer(t)
	b := newBoxRig(t, withoutDriver)
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
	if code := b.do(t, "GET", "/api/jobs/box/"+rec.ID, nil, &d); code != 200 || len(d.Log) != 3 {
		t.Fatalf("paged honest read %d %+v", code, d)
	}
}

func TestFleetRowsCarryJobCounts(t *testing.T) {
	requireAgentPeer(t)
	b := newBoxRig(t, withoutDriver)
	b.startJob(t, `{}`)
	b.srv.jobs.refresh(context.Background(), true)
	s := b.srv.jobs.summary("box")
	if s == nil || s.Running != 1 || s.NeedsDecision != 0 {
		t.Fatalf("summary %+v", s)
	}
	var f api.Fleet
	b.do(t, "GET", "/api/fleet", nil, &f)
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
	b := newBoxRig(t, withoutDriver)
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
	b := newBoxRigNamed(t, "stream", withoutDriver)
	rec := b.startJob(t, `{}`)
	var d api.JobDetail
	if code := b.do(t, "GET", "/api/jobs/stream/"+rec.ID, nil, &d); code != 200 || d.Job.Target != "stream" || len(d.Log) != 1 {
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
	b := newBoxRig(t, withoutDriver)
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

// I1: a box that holds its agent key can sign an endless valid chain. The
// read stops at the length the job first claimed, and at an absolute
// ceiling, quickly, and caches nothing.
func TestJobDetailRefusesAnEndlessLog(t *testing.T) {
	requireAgentPeer(t)
	b := newBoxRig(t, withoutDriver)
	rec := b.startJob(t, `{}`)
	honest := b.fullRead(t, rec.ID)

	endless := func(claim uint64) (pages *int) {
		pages = new(int)
		res := honest
		res.Entries = append([]intent.JobLogEntryJSON(nil), honest.Entries...)
		b.srv.jobs.readJob = func(_ context.Context, _ config.Config, _ config.Target, p intent.JobReadPayload) (intent.JobReadResult, error) {
			*pages++
			for uint64(len(res.Entries)) <= p.From {
				b.appendEntry(t, &res, joblog.KindLeaseReleased, []byte(`{}`))
			}
			out := res
			out.Entries, out.More = res.Entries[p.From:p.From+1], true
			out.Job.LogLen = claim
			return out, nil
		}
		return pages
	}

	start := time.Now()
	pages := endless(honest.Job.LogLen)
	b.detailRefused(t, rec.ID, "more than the 1 log entries")
	if *pages > 2 || time.Since(start) > 5*time.Second {
		t.Fatalf("read %d pages in %v", *pages, time.Since(start))
	}

	prev := jobLogMaxEntries
	jobLogMaxEntries = 5
	t.Cleanup(func() { jobLogMaxEntries = prev })
	pages = endless(1 << 40)
	b.detailRefused(t, rec.ID, "more than the 5 this server reads")
	if *pages != 1 {
		t.Fatalf("read %d pages of a log longer than the ceiling", *pages)
	}

	// Within the claimed count, the byte ceiling still holds.
	prevBytes := jobLogMaxBytes
	jobLogMaxBytes = 64
	t.Cleanup(func() { jobLogMaxBytes = prevBytes })
	b.serve(honest)
	b.detailRefused(t, rec.ID, "larger than the 64 bytes")
}

// I2: a cached log is the agent key's that verified it. After the box is
// re-paired to another key, a box answering "nothing new" with a matching
// record does not get the old log shown as its own: the read starts over
// and the old entries fail against the new key.
func TestJobLogCacheIsBoundToTheAgentKey(t *testing.T) {
	requireAgentPeer(t)
	b := newBoxRig(t, withoutDriver)
	rec := b.startJob(t, `{}`)
	honest := b.fullRead(t, rec.ID)
	var d api.JobDetail
	if code := b.do(t, "GET", "/api/jobs/box/"+rec.ID, nil, &d); code != 200 || b.cached(rec.ID) == nil {
		t.Fatalf("first read %d, cached %v", code, b.cached(rec.ID) != nil)
	}
	stranger, _ := signer.GenerateKey()
	if _, err := config.Update(func(c *config.Config) error {
		c.Targets[0].Agent.Address = stranger.Address().Hex()
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	b.serve(honest)
	b.detailRefused(t, rec.ID, "not signed by agent "+stranger.Address().Hex())
}

// M1: a log starts with its one created entry.
func TestJobDetailRefusesAMisshapenLog(t *testing.T) {
	requireAgentPeer(t)
	b := newBoxRig(t, withoutDriver)
	rec := b.startJob(t, `{}`)
	honest := b.fullRead(t, rec.ID)

	// A second created entry, carrying the job's own creating intent, so
	// every signature and binding holds and only the shape is wrong.
	twice := honest
	twice.Entries = append([]intent.JobLogEntryJSON(nil), honest.Entries...)
	b.appendEntry(t, &twice, joblog.KindCreated, honest.Entries[0].Body)
	b.serve(twice)
	b.detailRefused(t, rec.ID, "entry 1 is \"job.created\"")

	// No created entry at all: a chain the agent key signed from scratch.
	headless := honest
	headless.Entries = nil
	b.appendEntry(t, &headless, joblog.KindLeaseReleased, []byte(`{}`))
	b.serve(headless)
	b.detailRefused(t, rec.ID, "entry 0 is \"lease.released\"")

	empty := honest
	empty.Entries, empty.Job.LogLen, empty.Job.LogHead = nil, 0, ""
	b.serve(empty)
	b.detailRefused(t, rec.ID, "log is empty")
}

// M4: page two from another valid chain the same agent key signed for the
// same job is refused: each page must chain onto the last.
func TestJobDetailRefusesASplicedLog(t *testing.T) {
	requireAgentPeer(t)
	b := newBoxRig(t, withoutDriver)
	rec := b.startJob(t, `{}`)
	b.send(t, intent.KindJobLease, intent.JobLeasePayload{ID: rec.ID, Op: intent.LeaseRelease})
	honest := b.fullRead(t, rec.ID)

	// Another chain for the job: the same created entry's body re-signed at
	// another time, so its digest differs, then two entries on top of it.
	other := honest
	other.Entries = nil
	b.advance(time.Minute)
	b.appendEntry(t, &other, joblog.KindCreated, honest.Entries[0].Body)
	b.appendEntry(t, &other, joblog.KindIntent, honest.Entries[1].Body)
	b.appendEntry(t, &other, joblog.KindLeaseReleased, honest.Entries[2].Body)

	spliced := honest
	spliced.Job.LogLen, spliced.Job.LogHead = other.Job.LogLen, other.Job.LogHead
	b.srv.jobs.readJob = func(_ context.Context, _ config.Config, _ config.Target, p intent.JobReadPayload) (intent.JobReadResult, error) {
		out := spliced
		if p.From == 0 {
			out.Entries, out.More = honest.Entries[:1], true // page one: the real chain
		} else {
			out.Entries = other.Entries[p.From:] // page two: the other chain
		}
		return out, nil
	}
	b.detailRefused(t, rec.ID, "does not follow")
}

// M2: the cache holds only what the boxes still list.
func TestJobLogCacheFollowsTheListing(t *testing.T) {
	requireAgentPeer(t)
	b := newBoxRig(t, withoutDriver)
	rec := b.startJob(t, `{}`)
	var d api.JobDetail
	if code := b.do(t, "GET", "/api/jobs/box/"+rec.ID, nil, &d); code != 200 || b.cached(rec.ID) == nil {
		t.Fatalf("first read %d", code)
	}
	// The box prunes the job (J6): the next listing drops its log.
	for _, f := range []string{rec.ID + ".json", rec.ID + ".log"} {
		if err := os.Remove(filepath.Join(b.home, "jobs", f)); err != nil {
			t.Fatal(err)
		}
	}
	b.srv.jobs.refresh(context.Background(), true)
	if b.cached(rec.ID) != nil {
		t.Fatal("a pruned job's log is still cached")
	}

	// A removed box takes its logs with it.
	rec2 := b.startJob(t, `{}`)
	if code := b.do(t, "GET", "/api/jobs/box/"+rec2.ID, nil, &d); code != 200 || b.cached(rec2.ID) == nil {
		t.Fatalf("second job read %d", code)
	}
	if _, err := config.Update(func(c *config.Config) error { c.Targets = nil; return nil }); err != nil {
		t.Fatal(err)
	}
	b.srv.jobs.refresh(context.Background(), true)
	if b.cached(rec2.ID) != nil {
		t.Fatal("a removed box's log is still cached")
	}
}

// I1: the byte ceiling covers the whole log, across GETs. A box that grows
// its log one large entry per GET, each read under the ceiling, is refused
// once the log as a whole passes it, and nothing past it is cached.
func TestJobLogByteCapCoversTheWholeLog(t *testing.T) {
	requireAgentPeer(t)
	b := newBoxRig(t, withoutDriver)
	rec := b.startJob(t, `{}`)
	res := b.fullRead(t, rec.ID)
	prev := jobLogMaxBytes
	jobLogMaxBytes = 8000
	t.Cleanup(func() { jobLogMaxBytes = prev })
	pad := []byte(`{"pad":"` + strings.Repeat("x", 1500) + `"}`)

	served := 0
	for i := 0; i < 10; i++ {
		b.appendEntry(t, &res, joblog.KindLeaseReleased, pad)
		b.serve(res)
		code, body := b.get(t, "/api/jobs/box/"+rec.ID)
		if c := b.cached(rec.ID); c != nil && c.size > jobLogMaxBytes {
			t.Fatalf("cached %d bytes, over the %d ceiling", c.size, jobLogMaxBytes)
		}
		if code == http.StatusOK {
			served++
			if b.cached(rec.ID) == nil {
				t.Fatal("a verified log within the ceiling was not cached")
			}
			continue
		}
		var e api.Error
		_ = json.Unmarshal(body, &e)
		if code != http.StatusBadGateway || e.Code != api.CodeJobLogInvalid || !strings.Contains(e.Message, "larger than the 8000 bytes") || b.cached(rec.ID) != nil {
			t.Fatalf("GET %d answered %d %s", i, code, body)
		}
		if served < 2 {
			t.Fatalf("refused after only %d GETs: each GET's own read is under the ceiling", served)
		}
		return
	}
	t.Fatalf("a log grown one entry per GET was served %d times and never refused", served)
}

// The cache as a whole stays within jobLogCacheMaxBytes: the least recently
// read log is evicted for a new one, and a log bigger than the budget is
// not cached. A read that began before its target's logs were dropped does
// not put its log back.
func TestJobLogCacheBudgetAndGenerations(t *testing.T) {
	srv := New(Config{Token: NewSessionToken(), UI: fstest.MapFS{}})
	t.Cleanup(func() { srv.jobs.stopWithin(time.Second) })
	h := srv.jobs
	prev := jobLogCacheMaxBytes
	jobLogCacheMaxBytes = 100
	t.Cleanup(func() { jobLogCacheMaxBytes = prev })
	agentAddr := eip712.Address{1}
	k := func(id string) jobKey { return jobKey{"box", id} }
	listed := &boxJobs{}
	for _, id := range []string{"a", "b", "c", "d", "huge"} {
		listed.jobs = append(listed.jobs, intent.JobRecord{ID: id})
	}
	h.boxes["box"] = listed
	store := func(id string, size int) bool {
		_, gen := h.cachedLog(k(id), agentAddr)
		return h.storeLog(k(id), verifiedLog{agent: agentAddr, size: size}, gen)
	}
	if !store("a", 40) || !store("b", 40) {
		t.Fatal("logs within the budget were not cached")
	}
	h.cachedLog(k("a"), agentAddr) // a is now the more recently read
	if !store("c", 40) {
		t.Fatal("c was not cached")
	}
	if h.logs[k("b")] != nil || h.logs[k("a")] == nil || h.logs[k("c")] == nil || h.logBytes != 80 {
		t.Fatalf("after eviction: %v, %d bytes", h.logs, h.logBytes)
	}
	if store("huge", 101) || h.logs[k("huge")] != nil || h.logBytes != 80 {
		t.Fatalf("a log over the whole budget was cached: %d bytes", h.logBytes)
	}

	// A read in flight while a listing prunes its job does not re-cache it.
	_, gen := h.cachedLog(k("d"), agentAddr)
	h.dropLogs("box", map[string]bool{"a": true})
	if h.storeLog(k("d"), verifiedLog{agent: agentAddr, size: 10}, gen) || h.logs[k("d")] != nil {
		t.Fatal("a log read before its target's drop was cached after it")
	}
	if h.logs[k("c")] != nil || h.logs[k("a")] == nil || h.logBytes != 40 {
		t.Fatalf("after the drop: %v, %d bytes", h.logs, h.logBytes)
	}
}

// B3: a first read in flight, with nothing cached for its box, does not
// cache a job a listing pruned meanwhile, nor one whose box went. The
// generation and the listing each refuse it on their own.
func TestJobLogIsCachedOnlyWhileListed(t *testing.T) {
	srv := New(Config{Token: NewSessionToken(), UI: fstest.MapFS{}})
	t.Cleanup(func() { srv.jobs.stopWithin(time.Second) })
	h := srv.jobs
	agentAddr := eip712.Address{1}
	x := jobKey{"box", "x"}
	vl := verifiedLog{agent: agentAddr, size: 10}
	list := func(ids ...string) {
		box := &boxJobs{}
		for _, id := range ids {
			box.jobs = append(box.jobs, intent.JobRecord{ID: id})
		}
		h.mu.Lock()
		h.boxes["box"] = box
		h.mu.Unlock()
	}

	// Pruned by a listing during the read: dropLogs bumps the generation
	// though nothing was cached, and the listing no longer has x.
	list("x")
	_, gen := h.cachedLog(x, agentAddr)
	list()
	h.dropLogs("box", map[string]bool{})
	if h.storeLog(x, vl, gen) || len(h.logs) != 0 {
		t.Fatal("a job pruned during its first read was cached")
	}

	// The listing check alone: same generation, but x is not listed.
	_, gen = h.cachedLog(x, agentAddr)
	if h.storeLog(x, vl, gen) || len(h.logs) != 0 {
		t.Fatal("a job the listing does not have was cached")
	}

	// The generation check alone: x still listed (a stale listing), but its
	// logs were dropped since the read began.
	list("x")
	_, gen = h.cachedLog(x, agentAddr)
	h.dropLogs("box", map[string]bool{"x": true})
	if h.storeLog(x, vl, gen) || len(h.logs) != 0 {
		t.Fatal("a log read before its target's drop was cached")
	}

	// The box removed during the read.
	_, gen = h.cachedLog(x, agentAddr)
	h.mu.Lock()
	delete(h.boxes, "box")
	h.mu.Unlock()
	if h.storeLog(x, vl, gen) || len(h.logs) != 0 {
		t.Fatal("a removed box's job was cached")
	}

	// Listed and undisturbed: cached.
	list("x")
	_, gen = h.cachedLog(x, agentAddr)
	if !h.storeLog(x, vl, gen) || h.logs[x] == nil {
		t.Fatal("a listed job's verified log was not cached")
	}
}

// B3 through the listing itself: a box that disappears from the config
// bumps its generation in list's prune, so a read that began before it
// does not cache.
func TestRemovedBoxBumpsItsGeneration(t *testing.T) {
	requireAgentPeer(t)
	b := newBoxRig(t, withoutDriver)
	rec := b.startJob(t, `{}`)
	_, gen := b.srv.jobs.cachedLog(jobKey{"box", rec.ID}, b.key.Address())
	if _, err := config.Update(func(c *config.Config) error { c.Targets = nil; return nil }); err != nil {
		t.Fatal(err)
	}
	b.srv.jobs.refresh(context.Background(), true)
	if b.srv.jobs.storeLog(jobKey{"box", rec.ID}, verifiedLog{agent: b.key.Address(), size: 10}, gen) || b.cached(rec.ID) != nil {
		t.Fatal("a removed box's job was cached")
	}
}
