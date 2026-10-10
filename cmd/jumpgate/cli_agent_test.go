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

	"github.com/valve-tech/jumpgate/internal/agent"
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

// reset-replay wants --yes; with it, a broken replay record is kept beside
// the new one as evidence, never deleted, and the agent can open the fresh,
// empty record.
func TestAgentResetReplayKeepsTheBrokenRecord(t *testing.T) {
	testutil.RequireUnix(t) // agent-side code, Linux-only
	stderr := captureStderr(t)
	state := t.TempDir()
	path := filepath.Join(state, "replay.json")
	if err := os.WriteFile(path, []byte("{broken"), 0o600); err != nil {
		t.Fatal(err)
	}
	args := []string{"reset-replay", "--state-dir", state}
	if code := cmdAgent(args); code != exitCode("usage") || !strings.Contains(stderr(), "--yes") {
		t.Fatalf("without --yes: exit %d, stderr %q", code, stderr())
	}
	if b, _ := os.ReadFile(path); string(b) != "{broken" {
		t.Fatalf("the record changed without --yes: %q", b)
	}
	if code := cmdAgent(append(args, "--yes")); code != 0 {
		t.Fatalf("exit %d, stderr %q", code, stderr())
	}
	if _, err := agent.OpenReplay(path); err != nil {
		t.Fatalf("the new record does not open: %v", err)
	}
	kept, _ := filepath.Glob(path + ".broken-*")
	if len(kept) != 1 {
		t.Fatalf("broken records kept: %v", kept)
	}
	if b, _ := os.ReadFile(kept[0]); string(b) != "{broken" {
		t.Fatalf("the kept record holds %q", b)
	}
}

// With no record at all, reset-replay just starts one.
func TestAgentResetReplayWithoutARecordStartsOne(t *testing.T) {
	testutil.RequireUnix(t)
	state := t.TempDir()
	if err := agentResetReplay(state); err != nil {
		t.Fatal(err)
	}
	if _, err := agent.OpenReplay(filepath.Join(state, "replay.json")); err != nil {
		t.Fatalf("the record does not open: %v", err)
	}
}

// The agent's key comes from systemd's credentials directory when systemd
// hands one in, and from the state directory otherwise.
func TestAgentKeyPathPrefersSystemdCredentials(t *testing.T) {
	t.Setenv("CREDENTIALS_DIRECTORY", "")
	if got := agentKeyPath("/var/lib/jumpgate"); got != filepath.Join("/var/lib/jumpgate", "agent.key") {
		t.Fatalf("outside systemd: %q", got)
	}
	t.Setenv("CREDENTIALS_DIRECTORY", "/run/credentials/jumpgate-agent.service")
	if got := agentKeyPath("/var/lib/jumpgate"); got != filepath.Join("/run/credentials/jumpgate-agent.service", "agent.key") {
		t.Fatalf("under systemd: %q", got)
	}
}

// agent run with no key never listens: the box has no identity to answer
// with.
func TestAgentRunWithoutAKeyFails(t *testing.T) {
	testutil.RequireUnix(t)
	t.Setenv("CREDENTIALS_DIRECTORY", "")
	stderr := captureStderr(t)
	if code := cmdAgent([]string{"run", "--state-dir", t.TempDir(), "--config-dir", t.TempDir()}); code != 1 || !strings.Contains(stderr(), "agent.key") {
		t.Fatalf("exit %d, stderr %q", code, stderr())
	}
}

// Unknown subcommands and job-abandon without a well-formed id are usage
// errors, and an agent on Windows is refused outright.
func TestAgentRefusesWhatItCannotRun(t *testing.T) {
	captureStderr(t)
	for _, args := range [][]string{{}, {"bogus"}, {"job-abandon", "nope", "--yes"}, {"job-abandon", "0123456789abcdef", "extra"}, {"init", "--no-such-flag"}} {
		if code := cmdAgent(args); code != exitCode("usage") {
			t.Errorf("agent %v: exit %d", args, code)
		}
	}
	old := hostGOOS
	hostGOOS = "windows"
	t.Cleanup(func() { hostGOOS = old })
	if code := cmdAgent([]string{"run"}); code != exitCode("failed") {
		t.Fatalf("agent run on Windows: exit %d", code)
	}
}
