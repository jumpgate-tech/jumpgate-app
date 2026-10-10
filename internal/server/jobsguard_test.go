// internal/server/jobsguard_test.go
package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/catalog"
	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/intent"
	"github.com/valve-tech/jumpgate/internal/jobs/jobstest"
)

var setupBody = map[string]any{"chainId": 369, "execId": "reth", "beaconId": "lighthouse-pulse"}

// boxWire gives this controller a wire for the rig's box, as setup leaves,
// so the service routes that need one reach their job check.
func boxWire(t *testing.T) {
	t.Helper()
	if _, err := config.Update(func(c *config.Config) error {
		c.Targets[0].Wire = &catalog.WireConfig{ChainID: 369, ExecID: "reth", BeaconID: "lighthouse-pulse",
			Names: catalog.NamesCurrent, DataDir: "/var/lib/jumpgate-node/369", JWTPath: "/var/lib/jumpgate-node/369/jwt.hex"}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

// setupRunning reports whether the rig's box has a setup run in flight.
func (b *boxRig) setupRunning() bool {
	e := b.srv.reg.get(b.target)
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.setup != nil && e.setup.isRunning()
}

// wantJobActive checks a refusal is J1's, naming the job.
func wantJobActive(t *testing.T, what string, code int, raw []byte, id string) {
	t.Helper()
	var e api.Error
	_ = json.Unmarshal(raw, &e)
	if code != http.StatusConflict || e.Code != api.CodeJobActive || !strings.Contains(e.Message, id) ||
		!strings.Contains(e.Message, "owns the node services until it ends") {
		t.Fatalf("%s during a job: %d %s", what, code, raw)
	}
}

// C1: J1 is the agent's rule for service.action, but setup and service
// clear run over SSH and never ask the agent. A setup during
// node.migrate-names (after rename-account, before rewrite-units) would
// recreate the legacy account under a new uid, chown the data to it and
// start the legacy units. Both are refused while any job on the box has not
// finished; here, one waiting on a person's answer.
func TestSetupAndClearAreRefusedWhileAJobWaits(t *testing.T) {
	requireAgentPeer(t)
	b := newBoxRig(t)
	boxWire(t)
	var v api.JobView
	if code := b.do(t, "POST", "/api/jobs/box", api.JobStart{Kind: jobstest.TestKind, Inputs: json.RawMessage(`{"AskAt":"two"}`)}, &v); code != 200 {
		t.Fatalf("start = %d", code)
	}
	asked := waitJob(t, b, v.ID, intent.JobNeedsDecision)

	code, raw := b.req(t, "POST", "/api/targets/box/setup", setupBody)
	wantJobActive(t, "setup", code, raw, v.ID)
	if b.setupRunning() {
		t.Fatal("a refused setup left a run in flight")
	}
	code, raw = b.req(t, "POST", "/api/targets/box/services/exec/clear", map[string]string{"Confirm": "exec"})
	wantJobActive(t, "clear", code, raw, v.ID)

	// Once the job ends, setup is the box's again.
	var after api.JobView
	if code := b.do(t, "POST", "/api/jobs/box/"+v.ID+"/decide", api.JobDecision{Decision: asked.Question.ID, Option: intent.OptAbort}, &after); code != 200 {
		t.Fatalf("abort = %d", code)
	}
	waitJob(t, b, v.ID, intent.JobAborted)
	if code, raw := b.req(t, "POST", "/api/targets/box/setup", setupBody); code == http.StatusConflict && strings.Contains(string(raw), string(api.CodeJobActive)) {
		t.Fatalf("setup after the job ended: %d %s", code, raw)
	}
}

// A paired box whose jobs cannot be listed cannot prove no job runs, so
// setup and clear fail closed.
func TestSetupAndClearFailClosedWhenTheJobsCannotBeListed(t *testing.T) {
	requireAgentPeer(t)
	b := newBoxRig(t, withoutDriver)
	boxWire(t)
	b.srv.jobs.listJobs = func(context.Context, config.Config, config.Target) (intent.JobListResult, error) {
		return intent.JobListResult{}, &agentFailed{"the job store is unreadable"}
	}
	for path, body := range map[string]any{"/api/targets/box/setup": setupBody, "/api/targets/box/services/exec/clear": map[string]string{"Confirm": "exec"}} {
		code, raw := b.req(t, "POST", path, body)
		if code < 400 || !strings.Contains(string(raw), "cannot tell whether a job runs") {
			t.Errorf("%s with no job listing: %d %s", path, code, raw)
		}
	}
	if b.setupRunning() {
		t.Fatal("a refused setup left a run in flight")
	}
}

// An agent that runs no jobs (an older one, or a box without systemd) has
// none running: setup goes on.
func TestSetupGoesOnForABoxThatRunsNoJobs(t *testing.T) {
	requireAgentPeer(t)
	b := newBoxRig(t, withoutDriver)
	b.srv.jobs.listJobs = func(context.Context, config.Config, config.Target) (intent.JobListResult, error) {
		return intent.JobListResult{}, &agentRejected{intent.Rejection{Code: intent.ReasonJobsUnsupported, Message: "no job store"}}
	}
	cfg, _ := b.srv.loadConfig()
	tg, _ := findTarget(cfg, b.target)
	if _, busy, err := b.srv.jobs.activeJob(context.Background(), cfg, tg); err != nil || busy {
		t.Fatalf("a box without jobs: busy %v, %v", busy, err)
	}
}

// The other direction: a job is not started while setup runs on the box,
// nor while a clear does.
func TestAJobIsNotStartedWhileSetupRuns(t *testing.T) {
	requireAgentPeer(t)
	b := newBoxRig(t, withoutDriver)
	claimed, ok := b.srv.claimSetupRun(httptest.NewRecorder(), b.target)
	if !ok {
		t.Fatal("could not claim the setup slot")
	}
	code, raw := b.req(t, "POST", "/api/jobs/box", api.JobStart{Kind: jobstest.TestKind, Inputs: json.RawMessage(`{}`)})
	if code != http.StatusConflict || !strings.Contains(string(raw), "setup is running") {
		t.Fatalf("start during setup: %d %s", code, raw)
	}
	b.srv.releaseSetupRun(b.target, claimed)

	release, ok := b.srv.claimTargetOp(httptest.NewRecorder(), b.target)
	if !ok {
		t.Fatal("could not claim the target")
	}
	code, raw = b.req(t, "POST", "/api/jobs/box", api.JobStart{Kind: jobstest.TestKind, Inputs: json.RawMessage(`{}`)})
	if code != http.StatusConflict {
		t.Fatalf("start during a clear: %d %s", code, raw)
	}
	release()
	if code, raw := b.req(t, "POST", "/api/jobs/box", api.JobStart{Kind: jobstest.TestKind, Inputs: json.RawMessage(`{}`)}); code != 200 {
		t.Fatalf("start once the target is free: %d %s", code, raw)
	}
}

// A job starting holds the target: setup and clear wait for its start to
// return, then see the job in the listing.
func TestSetupWaitsForAJobThatIsStarting(t *testing.T) {
	requireAgentPeer(t)
	b := newBoxRig(t, withoutDriver)
	release, ok := b.srv.claimJobStart(httptest.NewRecorder(), b.target)
	if !ok {
		t.Fatal("could not claim a job start")
	}
	defer release()
	if _, ok := b.srv.claimSetupRun(httptest.NewRecorder(), b.target); ok {
		t.Fatal("setup claimed a target whose job is starting")
	}
	if _, ok := b.srv.claimTargetOp(httptest.NewRecorder(), b.target); ok {
		t.Fatal("a clear claimed a target whose job is starting")
	}
	if _, ok := b.srv.claimJobStart(httptest.NewRecorder(), b.target); ok {
		t.Fatal("two job starts at once")
	}
}
