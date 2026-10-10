// internal/bootstrap/e2e_jobs_test.go
//go:build e2e

package bootstrap

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/valve-tech/jumpgate/internal/agentclient"
	"github.com/valve-tech/jumpgate/internal/catalog"
	"github.com/valve-tech/jumpgate/internal/executor"
	"github.com/valve-tech/jumpgate/internal/intent"
	"github.com/valve-tech/jumpgate/internal/joblog"
	"github.com/valve-tech/jumpgate/internal/jobs"
	"github.com/valve-tech/jumpgate/internal/signer"
)

// e2eJobsDir is the agent's job store on the box (its StateDirectory/jobs).
const e2eJobsDir = "/var/lib/jumpgate/jobs"

// e2eLegacyWire is the node a box set up before sub-project 2 runs: the
// legacy names (Names ""), its data under the legacy root.
var e2eLegacyWire = catalog.WireConfig{ChainID: 369, ExecID: "reth", BeaconID: "lighthouse-pulse",
	DataDir: catalog.LegacyNames.DataRoot + "/369", JWTPath: catalog.LegacyNames.DataRoot + "/369/jwt.hex"}

// seedLegacyNode makes the container look like a box set up before
// sub-project 2: the legacy account, its data root, its two units (enabled,
// as setup leaves them, but not running: there is no client binary here),
// node.json, the snapshot and owner markers setup writes, and one process
// of the account that ignores SIGTERM.
func seedLegacyNode(ctx context.Context, t *testing.T, ex executor.Executor) {
	t.Helper()
	w := e2eLegacyWire
	legacy := catalog.LegacyNames
	execUnit, beaconUnit, err := catalog.RenderUnits(w)
	if err != nil {
		t.Fatal(err)
	}
	sh := func(cmd string) {
		t.Helper()
		if r, err := ex.Run(ctx, cmd, nil); err != nil || r.ExitCode != 0 {
			t.Fatalf("%s: %+v %v", cmd, r, err)
		}
	}
	sh("useradd --system --user-group --home-dir " + legacy.DataRoot + " --no-create-home --shell /usr/sbin/nologin " + legacy.User)
	sh("mkdir -p " + w.DataDir + "/db && head -c 32 /dev/urandom | od -An -tx1 | tr -d ' \\n' > " + w.JWTPath)
	sh("chown -R " + legacy.User + ":" + legacy.Group + " " + legacy.DataRoot)
	for path, body := range map[string]string{
		catalog.UnitPath(legacy.ExecUnit()):   execUnit,
		catalog.UnitPath(legacy.BeaconUnit()): beaconUnit,
	} {
		if err := ex.WriteFile(ctx, path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	b, err := json.Marshal(w)
	if err != nil {
		t.Fatal(err)
	}
	if err := ex.WriteFile(ctx, NodePath, b, 0o600); err != nil {
		t.Fatal(err)
	}
	// Setup's markers: the restore populated the legacy data dir, and the
	// chown -R finished for the legacy account. The job must move both.
	sh(catalog.SnapshotMarkerCommand(w))
	sh(catalog.OwnerMarkerCommand(w))
	// Enabled, so preflight records enablement the job must carry over to
	// the new units (verify holds it to that).
	sh("systemctl daemon-reload && systemctl enable " + strings.Join(legacy.Units(), " "))
	sh(`systemd-run --unit=e2e-busy --uid=` + legacy.User + ` --quiet /bin/sh -c "trap '' TERM; while :; do sleep 1; done"`)
}

// TestE2EMigrateNamesSurvivesAnAgentRestart drives node.migrate-names on real
// systemd the way a controller does (Task 12's driver is not built yet, so
// the test sends job.read, job.lease, job.advance and job.decide itself).
// While stop-units waits out its SIGTERM grace, the agent is stopped; the
// step's unit, PID 1's child, must outlive it and finish while the agent is
// down, and the restarted agent must fold that finished step exactly once.
func TestE2EMigrateNamesSurvivesAnAgentRestart(t *testing.T) {
	port := e2ePort(t)
	root := executor.SSHConfig{Host: "127.0.0.1", Port: port, User: "root", KeyPath: e2eEnv(t, "JUMPGATE_E2E_ROOT_KEY"), HostKey: ssh.InsecureIgnoreHostKey()}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	ex, err := executor.NewSSHContext(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	defer ex.Close()
	controller, _ := signer.GenerateKey()
	agentAddr, err := Run(ctx, Options{Exec: ex, Controller: controller.Address(), ControllerLabel: "e2e-jobs",
		TransportKey: e2eEnv(t, "JUMPGATE_E2E_TRANSPORT_PUB"), AgentBinary: e2eAgents(t)})
	if err != nil {
		t.Fatal(err)
	}
	waitAgentSocket(ctx, t, ex)
	seedLegacyNode(ctx, t, ex)

	tunnel := agentclient.Target{Agent: agentAddr, SSH: executor.SSHConfig{Host: "127.0.0.1", Port: port, User: "jumpgate",
		KeyPath: e2eEnv(t, "JUMPGATE_E2E_TRANSPORT_KEY"), HostKey: ssh.InsecureIgnoreHostKey()}}
	seqs := agentclient.NewMemorySeqStore()
	do := func(kind string, payload any, out any) {
		t.Helper()
		for attempt := 0; ; attempt++ {
			c, err := agentclient.Dial(ctx, tunnel, controller, seqs)
			if err == nil {
				var res agentclient.Response
				res, err = c.Do(ctx, kind, payload)
				c.Close()
				if err == nil {
					if res.Status != intent.StatusOK {
						t.Fatalf("%s: status %d %s %+v", kind, res.Status, res.Result, res.Rejection)
					}
					if out != nil {
						if err := json.Unmarshal(res.Result, out); err != nil {
							t.Fatalf("%s: %v", kind, err)
						}
					}
					return
				}
			}
			// The agent restarts below: a refused dial is expected for a few seconds.
			if !errors.Is(err, agentclient.ErrUnreachable) || attempt > 30 {
				t.Fatalf("%s: %v", kind, err)
			}
			time.Sleep(time.Second)
		}
	}
	run := func(cmd string) string {
		t.Helper()
		r, err := ex.Run(ctx, cmd, nil)
		if err != nil || r.ExitCode != 0 {
			t.Fatalf("%s: %+v %v", cmd, r, err)
		}
		return strings.TrimSpace(r.Stdout)
	}
	unitState := func(unit string) string {
		r, _ := ex.Run(ctx, "systemctl is-active "+unit+".service", nil)
		return strings.TrimSpace(r.Stdout)
	}

	var rec intent.JobRecord
	do(intent.KindNodeMigrateNames, intent.MigrateNamesPayload{}, &rec)
	t.Logf("job %s started", rec.ID)
	restarted := false
	for !intent.JobTerminal(rec.State) {
		var res intent.JobReadResult
		do(intent.KindJobRead, intent.JobReadPayload{ID: rec.ID}, &res)
		rec = res.Job
		switch rec.State {
		case intent.JobWaiting:
			t.Logf("advance to %s attempt %d", rec.Steps[rec.Step].ID, rec.Attempt)
			do(intent.KindJobAdvance, intent.JobAdvancePayload{ID: rec.ID, Step: rec.Step, Attempt: rec.Attempt}, &rec)
		case intent.JobNeedsDecision:
			// The busy process ignores SIGTERM: stop-units asks first whether
			// to signal it, then, once SIGTERM failed, whether to kill it.
			q := rec.Question
			var opt string
			switch {
			case q.Key == "busy-user" && q.Step == "stop-units" && rec.Attempt == 0:
				opt = "stop-processes"
			case q.Key == "busy-user" && q.Step == "stop-units" && rec.Attempt == 1:
				opt = "kill-processes"
			default:
				t.Fatalf("unexpected question at attempt %d: %+v", rec.Attempt, q)
			}
			t.Logf("decide %s: %s", q.ID, opt)
			do(intent.KindJobDecide, intent.JobDecidePayload{ID: rec.ID, Decision: q.ID, Option: opt}, &rec)
		case intent.JobRunning:
			// stop-units attempt 1 waits 30 s for a process that ignores
			// SIGTERM. Stop the agent inside that window and keep it down
			// until the step's unit has finished, so the outcome is there
			// for the restarted agent to fold, and only for it.
			if rec.Unit == jobs.UnitName(rec.ID, "stop-units", 1) && !restarted {
				foldAcrossRestart(ctx, t, ex, rec, run, unitState)
				restarted = true
				waitAgentSocket(ctx, t, ex)
				// The controller comes back: retaking its own lease is a
				// renewal, and the job must have moved on from running.
				do(intent.KindJobLease, intent.JobLeasePayload{ID: rec.ID, Op: intent.LeaseTake}, &rec)
				do(intent.KindJobRead, intent.JobReadPayload{ID: rec.ID}, &res)
				if res.Job.State != intent.JobNeedsDecision || res.Job.Question == nil || res.Job.Question.Step != "stop-units" {
					t.Fatalf("after the restart the job is %s (question %+v), want the step folded into its next question", res.Job.State, res.Job.Question)
				}
				// A second restart finds the outcome already folded and must
				// not fold it again.
				run("systemctl restart jumpgate-agent")
				waitAgentSocket(ctx, t, ex)
				continue
			}
			time.Sleep(time.Second)
		}
	}
	if rec.State != intent.JobSucceeded || !restarted {
		t.Fatalf("job ended %s (restarted %v): %s", rec.State, restarted, rec.Error)
	}

	// The box runs under the new names.
	for _, c := range []struct {
		cmd    string
		wantOK bool
	}{
		{"id -u jumpgate-node", true}, {"id -u valve-node-app", false},
		{"getent group jumpgate-node", true}, {"getent group valve-node-app", false},
		{"test -d /var/lib/jumpgate-node/369/db", true}, {"test -e /var/lib/valve-node-app", false},
		{"test \"$(stat -c %U /var/lib/jumpgate-node/369)\" = jumpgate-node", true},
		{"test -f /etc/systemd/system/jumpgate-node-exec.service", true},
		{"test -f /etc/systemd/system/jumpgate-node-beacon.service", true},
		{"test -e /etc/systemd/system/valve-node-app-exec.service", false},
		{"test -e /etc/systemd/system/valve-node-app-beacon.service", false},
		{"systemctl is-enabled jumpgate-node-exec.service", true},
		{"systemctl is-enabled jumpgate-node-beacon.service", true},
		{"grep -qx User=jumpgate-node /etc/systemd/system/jumpgate-node-exec.service", true},
		{"grep -q /var/lib/jumpgate-node/369 /etc/systemd/system/jumpgate-node-exec.service", true},
		{"systemctl is-active e2e-busy.service", false},
	} {
		r, err := ex.Run(ctx, c.cmd, nil)
		if err != nil || (r.ExitCode == 0) != c.wantOK {
			t.Errorf("%s: exit %d %v, want ok=%v", c.cmd, r.ExitCode, err, c.wantOK)
		}
	}
	// names.json records the move, node.json names the new account and dir,
	// and both markers name the new values.
	var mk jobs.NamesMarker
	if b, err := ex.ReadFile(ctx, "/etc/jumpgate/names.json"); err != nil || json.Unmarshal(b, &mk) != nil {
		t.Errorf("names.json: %q %v", b, err)
	}
	if want := (jobs.NamesMarker{Names: catalog.NamesCurrent, DataRootFrom: catalog.LegacyNames.DataRoot, DataRootTo: catalog.CurrentNames.DataRoot}); mk != want {
		t.Errorf("names.json %+v, want %+v", mk, want)
	}
	var nw catalog.WireConfig
	if b, err := ex.ReadFile(ctx, NodePath); err != nil || json.Unmarshal(b, &nw) != nil {
		t.Errorf("node.json: %q %v", b, err)
	}
	if nw.Names != catalog.NamesCurrent || nw.DataDir != "/var/lib/jumpgate-node/369" || nw.JWTPath != "/var/lib/jumpgate-node/369/jwt.hex" {
		t.Errorf("node.json %+v", nw)
	}
	if got := run("cat " + catalog.SnapshotMarkerPath(nw)); got != "/var/lib/jumpgate-node/369" {
		t.Errorf("snapshot marker names %q", got)
	}
	if got := run("cat " + catalog.OwnerMarkerPath(nw)); got != catalog.CurrentNames.User {
		t.Errorf("owner marker names %q", got)
	}
	var info intent.AgentInfo
	do(intent.KindAgentInfo, struct{}{}, &info)
	if info.Names != catalog.NamesCurrent || info.DataDir != "/var/lib/jumpgate-node/369" {
		t.Errorf("agent.info %+v", info)
	}

	// The whole history verifies, and the restarts left exactly one outcome
	// for the step that finished while the agent was down.
	var res intent.JobReadResult
	do(intent.KindJobRead, intent.JobReadPayload{ID: rec.ID}, &res)
	if res.More || uint64(len(res.Entries)) != res.Job.LogLen {
		t.Fatalf("read %d entries (more %v) of a %d-entry log", len(res.Entries), res.More, res.Job.LogLen)
	}
	var es []joblog.Entry
	outcomes := 0
	for _, j := range res.Entries {
		e, err := joblog.FromJSON(j)
		if err != nil {
			t.Fatal(err)
		}
		es = append(es, e)
		if e.Kind == joblog.KindStepFinished || e.Kind == joblog.KindStepLost {
			var b joblog.StepBody
			if err := json.Unmarshal(e.Body, &b); err != nil {
				t.Fatal(err)
			}
			if b.Step == "stop-units" && b.Attempt == 1 {
				outcomes++
				if e.Kind != joblog.KindStepFinished || !b.OK {
					t.Errorf("stop-units attempt 1 logged as %s %+v, want finished ok", e.Kind, b)
				}
			}
		}
	}
	head, err := joblog.Verify(agentAddr, rec.ID, es, 0, [32]byte{})
	if err != nil {
		t.Fatal(err)
	}
	if got := "0x" + hex.EncodeToString(head[:]); got != res.Job.LogHead {
		t.Fatalf("the log's head is %s, the manifest says %s", got, res.Job.LogHead)
	}
	if err := joblog.VerifyIntents(agentAddr, rec.ID, es); err != nil {
		t.Fatal(err)
	}
	if outcomes != 1 {
		t.Fatalf("stop-units attempt 1 has %d outcomes in the log, want 1", outcomes)
	}
	t.Logf("log: %d entries, verified", len(es))

	// The steps' output is in the journal, and job.output reads it.
	var out intent.JobOutputResult
	do(intent.KindJobOutput, intent.JobOutputPayload{ID: rec.ID, N: 500}, &out)
	if len(out.Lines) == 0 || out.Cursor == "" {
		t.Fatalf("job.output %+v", out)
	}
	sawStop := false
	for _, l := range out.Lines {
		sawStop = sawStop || (strings.HasPrefix(l.Unit, jobs.UnitName(rec.ID, "stop-units", 1)) && strings.Contains(l.Line, "pkill -TERM"))
	}
	if !sawStop {
		t.Errorf("job.output has no SIGTERM line from stop-units attempt 1 (%d lines)", len(out.Lines))
	}
}

// foldAcrossRestart stops the agent while rec's step runs, waits for the
// step's unit to finish on its own, checks that it left its result and that
// nothing folded it yet, and starts the agent again.
func foldAcrossRestart(ctx context.Context, t *testing.T, ex executor.Executor, rec intent.JobRecord, run func(string) string, unitState func(string) string) {
	t.Helper()
	run("systemctl stop jumpgate-agent")
	t.Logf("agent stopped while %s runs", rec.Unit)
	if st := unitState(rec.Unit); st != "active" {
		t.Fatalf("the step's unit did not survive the agent stopping: %q", st)
	}
	deadline := time.Now().Add(2 * time.Minute)
	for unitState(rec.Unit) == "active" {
		if time.Now().After(deadline) {
			t.Fatalf("%s never finished", rec.Unit)
		}
		time.Sleep(time.Second)
	}
	result, err := jobs.ResultPath(e2eJobsDir, rec.ID, "stop-units", 1)
	if err != nil {
		t.Fatal(err)
	}
	var sr jobs.StepResult
	if b, err := ex.ReadFile(ctx, result); err != nil || json.Unmarshal(b, &sr) != nil {
		t.Fatalf("%s finished without its result file: %q %v", rec.Unit, b, err)
	}
	if !sr.OK || sr.Ask == nil || sr.Ask.Key != "busy-user" {
		t.Fatalf("stop-units attempt 1 result %+v, want the after-SIGTERM question", sr)
	}
	var man jobs.Manifest
	if b, err := ex.ReadFile(ctx, e2eJobsDir+"/"+rec.ID+".json"); err != nil || json.Unmarshal(b, &man) != nil {
		t.Fatalf("manifest: %q %v", b, err)
	}
	if man.State != intent.JobRunning || man.Unit != rec.Unit {
		t.Fatalf("with the agent down the manifest is %s on %q; nothing should have folded the step", man.State, man.Unit)
	}
	t.Logf("%s finished while the agent was down; starting the agent", rec.Unit)
	run("systemctl start jumpgate-agent")
}
