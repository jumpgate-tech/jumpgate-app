package setup

import (
	"context"
	"strings"
	"testing"

	"github.com/valve-tech/valve-node-app/internal/catalog"
	"github.com/valve-tech/valve-node-app/internal/executor"
)

func snapshotWire() catalog.WireConfig {
	w := testWire()
	w.ExecSnapshot = true
	w.SnapshotKey = "vk_abcd1234"
	return w
}

// ran reports whether any command the fake was given contains every fragment.
// Requiring all of them together is what stops "reth download ran" passing
// when it ran against the wrong datadir.
func ran(e *fakeExecutor, fragments ...string) bool {
	for _, cmd := range e.callLog() {
		all := true
		for _, f := range fragments {
			if !strings.Contains(cmd, f) {
				all = false
				break
			}
		}
		if all {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------
// the step is a no-op unless it was asked for
// ---------------------------------------------------------------------

// Snapshot restore downloads hundreds of gigabytes. A step that ran without
// being asked would do that to an operator who chose to sync from genesis, so
// the opt-out has to be the absence of a flag, not a guess.
func TestSnapshot_DoesNothingAtAllWhenNotOptedIn(t *testing.T) {
	e := newFakeExecutor()
	w := testWire() // ExecSnapshot false
	step := snapshotStep()

	if err := step.Run(context.Background(), e, &State{Wire: w}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if err := step.Verify(context.Background(), e, &State{Wire: w}); err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if calls := e.callLog(); len(calls) != 0 {
		t.Fatalf("the step ran %d commands on a target that did not ask for a snapshot: %q", len(calls), calls)
	}
}

// ---------------------------------------------------------------------
// what it actually runs
// ---------------------------------------------------------------------

// The datadir has to exist before `reth download` writes into it, and the
// download has to carry the operator's key. Both are asserted on the command
// the target would really receive.
func TestSnapshot_CreatesTheDataDirThenDownloadsIntoIt(t *testing.T) {
	e := newFakeExecutor()
	w := snapshotWire()

	if err := snapshotStep().Run(context.Background(), e, &State{Wire: w}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if !ran(e, "mkdir -p", w.DataDir) {
		t.Errorf("the data dir was never created; ran: %q", e.callLog())
	}
	if !ran(e, "reth download", "--datadir", w.DataDir, w.SnapshotKey) {
		t.Errorf("the download command is not what the target would receive; ran: %q", e.callLog())
	}
	// Order matters: downloading into a directory that does not exist yet is
	// the failure this sequencing prevents.
	mkdirAt, downloadAt := -1, -1
	for i, cmd := range e.callLog() {
		if strings.Contains(cmd, "mkdir -p") && mkdirAt < 0 {
			mkdirAt = i
		}
		if strings.Contains(cmd, "reth download") && downloadAt < 0 {
			downloadAt = i
		}
	}
	if mkdirAt < 0 || downloadAt < 0 || mkdirAt > downloadAt {
		t.Errorf("mkdir at %d, download at %d — the directory must be made first", mkdirAt, downloadAt)
	}
}

// A datadir with a space or a quote in it is a path an operator can genuinely
// type, and it reaches `sh -c`. It must arrive as one argument.
func TestSnapshot_QuotesADataDirThatWouldOtherwiseSplit(t *testing.T) {
	e := newFakeExecutor()
	w := snapshotWire()
	w.DataDir = "/mnt/my data/reth"

	if err := snapshotStep().Run(context.Background(), e, &State{Wire: w}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !ran(e, "mkdir -p '/mnt/my data/reth'") {
		t.Errorf("the data dir was not quoted, so mkdir would make two directories; ran: %q", e.callLog())
	}
}

// ---------------------------------------------------------------------
// failure carries the evidence
// ---------------------------------------------------------------------

// A download that failed hours in must say what the target said. "snapshot
// failed" sends the operator to the logs; the exit code and stderr are what
// they would have gone looking for.
func TestSnapshot_FailureQuotesWhatTheTargetSaid(t *testing.T) {
	e := newFakeExecutor().script("reth download", executor.Result{
		ExitCode: 1,
		Stderr:   "error: manifest 404 (is your key right?)\n",
	})

	err := snapshotStep().Run(context.Background(), e, &State{Wire: snapshotWire()})
	if err == nil {
		t.Fatal("a failed download reported success")
	}
	for _, want := range []string{"exit 1", "manifest 404"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not carry %q", err, want)
		}
	}
}

func TestSnapshot_FailureToMakeTheDataDirStopsBeforeDownloading(t *testing.T) {
	e := newFakeExecutor().script("mkdir -p", executor.Result{
		ExitCode: 1,
		Stderr:   "mkdir: /mnt/reth: Read-only file system\n",
	})

	err := snapshotStep().Run(context.Background(), e, &State{Wire: snapshotWire()})
	if err == nil {
		t.Fatal("an unwritable data dir reported success")
	}
	if !strings.Contains(err.Error(), "Read-only file system") {
		t.Errorf("error %q does not say why the directory could not be made", err)
	}
	if ran(e, "reth download") {
		t.Error("the download ran anyway, into a directory that could not be created")
	}
}

// The UI should never offer snapshot restore for a client that cannot do it,
// but the step refuses anyway rather than running a reth command against
// another client's datadir — which would be a reth database written over an
// erigon one.
func TestSnapshot_RefusesAClientThatCannotRestore(t *testing.T) {
	e := newFakeExecutor()
	w := snapshotWire()
	w.ExecID = "go-pulse"

	err := snapshotStep().Run(context.Background(), e, &State{Wire: w})
	if err == nil {
		t.Fatal("snapshot restore ran for a client that does not support it")
	}
	if ran(e, "reth download") || ran(e, "mkdir") {
		t.Errorf("a refused restore still touched the target: %q", e.callLog())
	}
}

// A chain with no reth --chain name has no snapshot. The step must fail
// BEFORE it makes the directory look prepared.
func TestSnapshot_RefusesAChainWithNoSnapshot(t *testing.T) {
	e := newFakeExecutor()
	w := snapshotWire()
	w.ChainID = catalog.DevnetChainID

	err := snapshotStep().Run(context.Background(), e, &State{Wire: w})
	if err == nil {
		t.Fatal("a chain with no snapshot reported success")
	}
	if ran(e, "reth download") {
		t.Error("a download ran for a chain that has no snapshot")
	}
}

// ---------------------------------------------------------------------
// verify
// ---------------------------------------------------------------------

// Verify is what stops the plan advancing to wire with an empty datadir — the
// node would then start and sync from genesis, which is exactly what the
// operator paid a download to avoid, while every step reported success.
func TestSnapshot_VerifyFailsUntilTheDataDirIsPopulated(t *testing.T) {
	e := newFakeExecutor().script("test -d", executor.Result{ExitCode: 1})

	err := snapshotStep().Verify(context.Background(), e, &State{Wire: snapshotWire()})
	if err == nil {
		t.Fatal("verify passed on a datadir with no db/ in it")
	}
	if !strings.Contains(err.Error(), "db/") {
		t.Errorf("error %q does not name the marker it looked for", err)
	}
}

func TestSnapshot_VerifyPassesOnceTheDataDirHasADatabase(t *testing.T) {
	e := newFakeExecutor().script("test -d", executor.Result{ExitCode: 0})

	if err := snapshotStep().Verify(context.Background(), e, &State{Wire: snapshotWire()}); err != nil {
		t.Fatalf("verify: %v", err)
	}
	if !ran(e, "test -d", "/mnt/reth/db") {
		t.Errorf("verify looked in the wrong place: %q", e.callLog())
	}
}

// ---------------------------------------------------------------------
// start
// ---------------------------------------------------------------------

// wire's `enable --now` can leave a unit inactive — a unit that failed fast on
// first boot needs a clean restart once the datadir and JWT are in place. This
// step is what makes the plan converge instead of ending on a machine where
// systemd is enabled and nothing is running.
func TestStart_StartsBothUnits(t *testing.T) {
	e := newFakeExecutor()

	if err := startStep().Run(context.Background(), e, &State{Wire: testWire()}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !ran(e, "systemctl start", execUnitName, beaconUnitName) {
		t.Errorf("both units must be started in one call; ran: %q", e.callLog())
	}
}

func TestStart_FailureQuotesSystemd(t *testing.T) {
	e := newFakeExecutor().script("systemctl start", executor.Result{
		ExitCode: 1,
		Stderr:   "Job for valve-exec.service failed\n",
	})

	err := startStep().Run(context.Background(), e, &State{Wire: testWire()})
	if err == nil {
		t.Fatal("a failed start reported success")
	}
	if !strings.Contains(err.Error(), "Job for valve-exec.service failed") {
		t.Errorf("error %q does not carry what systemd said", err)
	}
}

// The verify reads is-active for BOTH units and must fail unless both say
// active. One active and one dead is a half-running node that reports itself
// set up — the exact shape of failure this repo has been bitten by.
func TestStart_VerifyRequiresBothUnitsActive(t *testing.T) {
	for name, stdout := range map[string]string{
		"exec active, beacon dead":       "active\ninactive\n",
		"exec dead, beacon active":       "inactive\nactive\n",
		"both failed":                    "failed\nfailed\n",
		"one activating, not yet active": "active\nactivating\n",
	} {
		t.Run(name, func(t *testing.T) {
			// Exit 0 with a non-active line is the case a bare exit-code check
			// misses: systemctl reports per-unit state in stdout.
			e := newFakeExecutor().script("systemctl is-active", executor.Result{Stdout: stdout, ExitCode: 0})

			err := startStep().Verify(context.Background(), e, &State{Wire: testWire()})
			if err == nil {
				t.Fatalf("verify passed with is-active reporting %q", stdout)
			}
			if !strings.Contains(err.Error(), "not both active") {
				t.Errorf("error %q does not say which condition failed", err)
			}
		})
	}
}

func TestStart_VerifyPassesWhenBothAreActive(t *testing.T) {
	e := newFakeExecutor().script("systemctl is-active", executor.Result{Stdout: "active\nactive\n", ExitCode: 0})

	if err := startStep().Verify(context.Background(), e, &State{Wire: testWire()}); err != nil {
		t.Fatalf("verify: %v", err)
	}
}

// A non-zero exit from is-active means at least one unit is not active, and
// the message must carry what systemd printed rather than the exit code alone.
func TestStart_VerifyReportsWhatSystemdSaidOnNonZeroExit(t *testing.T) {
	e := newFakeExecutor().script("systemctl is-active", executor.Result{Stdout: "inactive\ninactive\n", ExitCode: 3})

	err := startStep().Verify(context.Background(), e, &State{Wire: testWire()})
	if err == nil {
		t.Fatal("verify passed on a non-zero is-active")
	}
	if !strings.Contains(err.Error(), "inactive") {
		t.Errorf("error %q does not carry systemd's own words", err)
	}
}
