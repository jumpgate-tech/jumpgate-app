package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/valve-tech/jumpgate/internal/executor"
	"github.com/valve-tech/jumpgate/internal/intent"
	"github.com/valve-tech/jumpgate/internal/jobs"
	"github.com/valve-tech/jumpgate/internal/jobs/jobstest"
	"github.com/valve-tech/jumpgate/internal/signer"
	"github.com/valve-tech/jumpgate/internal/testutil"
)

// startStuckJob gives the box in state a job that has not finished, as the
// agent would have started it, and returns the open manager and the job.
func startStuckJob(t *testing.T, state, conf string) (*jobs.Manager, jobs.ManagerConfig, string) {
	t.Helper()
	jobstest.RegisterPlan(t)
	if err := agentInit(io.Discard, state, conf, false); err != nil {
		t.Fatal(err)
	}
	k, err := signer.LoadKeyFile(filepath.Join(state, "agent.key"))
	if err != nil {
		t.Fatal(err)
	}
	cfg := jobs.ManagerConfig{Dir: filepath.Join(state, "jobs"), Key: k, Runner: &jobstest.Runner{},
		StepArgv: func(a jobs.StepArgs) []string { return jobs.StepArgv("/jumpgate", state, conf, a) }}
	m, err := jobs.NewManager(cfg)
	if err != nil {
		t.Fatal(err)
	}
	ctl, _ := signer.GenerateKey()
	nonce, _ := intent.NewNonce()
	now := time.Now()
	in := intent.Intent{Agent: k.Address(), Controller: ctl.Address(), Nonce: nonce, IssuedAt: uint64(now.Unix()),
		Expiry: uint64(now.Unix() + 600), Kind: jobstest.TestKind, PayloadHash: intent.Hash([]byte(`{}`))}
	rec, err := m.Start(context.Background(), jobstest.TestKind, []byte(`{}`), ctl.Address(), intent.Envelope{Intent: in.JSON(), Payload: []byte(`{}`)})
	if err != nil {
		t.Fatal(err)
	}
	return m, cfg, rec.ID
}

// I4: job-abandon is the local way out for a job no signed request can move.
// It wants --yes, refuses while the agent holds the job store (saying to
// stop it), and then ends the job failed under the agent's own key.
func TestAgentJobAbandon(t *testing.T) {
	testutil.RequireUnix(t) // agent-side code, Linux-only
	stderr := captureStderr(t)
	state, conf := t.TempDir(), t.TempDir()
	m, cfg, id := startStuckJob(t, state, conf)
	args := []string{"job-abandon", id, "--state-dir", state, "--config-dir", conf}

	if code := cmdAgent(args); code != exitCode("usage") || !strings.Contains(stderr(), "--yes") {
		t.Fatalf("job-abandon without --yes: exit %d, stderr %q", code, stderr())
	}
	args = append(args, "--yes")
	if code := cmdAgent(args); code != 1 || !strings.Contains(stderr(), "stop the agent") {
		t.Fatalf("job-abandon beside a running agent: exit %d, stderr %q", code, stderr())
	}
	m.Close()
	if code := cmdAgent(args); code != 0 {
		t.Fatalf("job-abandon: exit %d, stderr %q", code, stderr())
	}
	m, err := jobs.NewManager(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	res, err := m.Read(id, 0)
	if err != nil {
		t.Fatal(err)
	}
	if res.Job.State != intent.JobFailed || !strings.Contains(res.Job.Error, jobs.AbandonedNote) {
		t.Fatalf("job after job-abandon: %+v", res.Job)
	}
	last := res.Entries[len(res.Entries)-1]
	var fin struct{ State string }
	if last.Kind != "job.finished" || json.Unmarshal(last.Body, &fin) != nil || fin.State != intent.JobFailed {
		t.Fatalf("last entry %+v", last)
	}
}

// M3: a job store that will not open for any reason but another agent
// holding it must not stop the agent: status and logs keep working. The
// failure is logged and returned as unavailable, never as "no jobs", so the
// agent can refuse service.action while a job may sit in the store.
func TestAJobStoreThatWillNotOpenRunsWithoutJobs(t *testing.T) {
	testutil.RequireUnix(t) // agent-side code, Linux-only
	state, conf := t.TempDir(), t.TempDir()
	if err := os.WriteFile(filepath.Join(state, "jobs"), []byte("not a directory"), 0o600); err != nil {
		t.Fatal(err)
	}
	k, _ := signer.GenerateKey()
	var log strings.Builder
	m, unavailable, err := agentJobManager(&log, state, conf, k, executor.NewLocal())
	if err != nil || m != nil || unavailable == nil {
		t.Fatalf("an unopenable store: %v, %v, %v", m, unavailable, err)
	}
	if !strings.Contains(log.String(), "without durable jobs") {
		t.Fatalf("log %q", log.String())
	}
	// Another agent holding the store still stops this one.
	state = t.TempDir()
	held, err := openJobManager(state, conf, k, executor.NewLocal())
	if err != nil {
		t.Fatal(err)
	}
	defer held.Close()
	if _, _, err := agentJobManager(io.Discard, state, conf, k, executor.NewLocal()); !errors.Is(err, jobs.ErrStoreInUse) {
		t.Fatalf("a held store: %v", err)
	}
}
