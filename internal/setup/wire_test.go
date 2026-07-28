package setup

// wire's failure paths. Every one of them is a way for the wizard to declare a
// machine set up while leaving it unable to run a node — a JWT that was never
// written, a data tree still owned by root, units that were rewritten but never
// restarted. The happy paths are covered in steps_test.go; what is here is
// everything that can go wrong on the way.

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/valve-tech/valve-node-app/internal/catalog"
	"github.com/valve-tech/valve-node-app/internal/executor"
)

// wiredExecutor is a target where every wire command succeeds and the JWT is
// already present, which is the shortest path through Run.
func wiredExecutor() *fakeExecutor {
	return newFakeExecutor().script("test -f", executor.Result{ExitCode: 0})
}

// ---------------------------------------------------------------------
// the JWT
// ---------------------------------------------------------------------

// The JWT is what lets the execution and beacon clients talk to each other. A
// write that failed and was not reported produces two services that start
// cleanly and never handshake — which surfaces days later as "my node is not
// syncing", with nothing in the wizard to suggest why.
func TestWire_AFailedJWTWriteIsReported(t *testing.T) {
	e := newFakeExecutor().
		script("test -f", executor.Result{ExitCode: 1}). // no JWT yet
		script("openssl rand", executor.Result{ExitCode: 1, Stderr: "openssl: command not found\n"})

	err := wireStep().Run(context.Background(), e, &State{Wire: testWire()})
	if err == nil {
		t.Fatal("a failed JWT write reported success")
	}
	if !strings.Contains(err.Error(), "openssl: command not found") {
		t.Errorf("error %q does not carry what the target said", err)
	}
}

// ---------------------------------------------------------------------
// ownership — the de-root migration
// ---------------------------------------------------------------------

// The chown is what migrates a pre-de-root install. If it fails and the run
// continues, the units are rewritten and restarted against a tree the service
// account cannot write, so both services come up and immediately die — and the
// wizard has already said the step passed.
func TestWire_AFailedChownStopsTheRun(t *testing.T) {
	e := wiredExecutor().script("chown -R", executor.Result{
		ExitCode: 1,
		Stderr:   "chown: invalid user: 'valve-node'\n",
	})

	err := wireStep().Run(context.Background(), e, &State{Wire: testWire()})
	if err == nil {
		t.Fatal("a failed chown reported success")
	}
	if !strings.Contains(err.Error(), "invalid user") {
		t.Errorf("error %q does not carry the reason", err)
	}
	// And the units must not have been enabled against an unwritable tree.
	if ran(e, "systemctl enable") {
		t.Error("the services were enabled after the chown failed")
	}
}

// The tree is re-owned on EVERY run, not only when the JWT was just written.
// The migration case is precisely the one where the JWT already exists.
func TestWire_TheTreeIsReownedEvenWhenTheJWTAlreadyExists(t *testing.T) {
	e := wiredExecutor()

	if err := wireStep().Run(context.Background(), e, &State{Wire: testWire()}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !ran(e, "chown -R", catalog.ServiceUser, "/mnt/reth") {
		t.Errorf("the data tree was not re-owned; ran: %q", e.callLog())
	}
}

// A fresh install must NOT get the extra restart: `enable --now` already
// starts a newly-written unit, and restarting it again is a second needless
// bounce during setup.
func TestWire_AFreshInstallIsNotRestartedTwice(t *testing.T) {
	e := wiredExecutor() // no unit files on the target at all

	if err := wireStep().Run(context.Background(), e, &State{Wire: testWire()}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if ran(e, "systemctl restart") {
		t.Errorf("a fresh install was restarted on top of enable --now; ran: %q", e.callLog())
	}
}

func TestWire_AFailedEnableIsReported(t *testing.T) {
	e := wiredExecutor().script("systemctl daemon-reload", executor.Result{
		ExitCode: 1,
		Stderr:   "Failed to enable unit: Unit file is masked\n",
	})

	err := wireStep().Run(context.Background(), e, &State{Wire: testWire()})
	if err == nil {
		t.Fatal("a failed enable reported success")
	}
	if !strings.Contains(err.Error(), "masked") {
		t.Errorf("error %q does not carry systemd's reason", err)
	}
}

// A transport failure is not a systemd verdict, and must not be reported as
// one — an SSH drop mid-setup is a different problem with a different fix.
func TestWire_ATransportFailureIsNotReportedAsASystemdFailure(t *testing.T) {
	e := wiredExecutor().errOn("chown -R", errors.New("ssh: connection lost"))

	err := wireStep().Run(context.Background(), e, &State{Wire: testWire()})
	if err == nil {
		t.Fatal("a dropped connection reported success")
	}
	if !strings.Contains(err.Error(), "connection lost") {
		t.Errorf("error %q loses the transport failure", err)
	}
}

// ---------------------------------------------------------------------
// verify
// ---------------------------------------------------------------------

// Verify is the gate that decides whether Run has to happen at all, so each of
// its checks has to be able to fail on its own.
//
// Ownership has its own test in steps_test.go
// (TestWire_VerifyFailsWhenDataDirNotOwnedByServiceUser) and is deliberately
// not repeated here; what this adds is the CONTROL — a fully wired target that
// verifies clean — without which none of the failure rows prove anything, since
// a Verify that always failed would satisfy every one of them.
func TestWire_VerifyFailsOnEachConditionSeparately(t *testing.T) {
	renderedExec, renderedBeacon, err := catalog.RenderUnits(testWire())
	if err != nil {
		t.Fatalf("RenderUnits: %v", err)
	}
	// A fully wired target, which each case then breaks in exactly one way.
	wired := func() *fakeExecutor {
		e := newFakeExecutor().
			script("test -f", executor.Result{ExitCode: 0}).
			script("stat -c", executor.Result{Stdout: catalog.ServiceUser + "\n"}).
			script("systemctl is-enabled", executor.Result{ExitCode: 0, Stdout: "enabled\nenabled\n"})
		e.files[execUnitPath] = []byte(renderedExec)
		e.files[beaconUnitPath] = []byte(renderedBeacon)
		return e
	}

	// The control: unbroken, it must pass. Without this the cases below
	// would pass even if Verify always failed.
	if err := wireStep().Verify(context.Background(), wired(), &State{Wire: testWire()}); err != nil {
		t.Fatalf("a fully wired target did not verify: %v", err)
	}

	for name, tc := range map[string]struct {
		break_ func(*fakeExecutor)
		want   string
	}{
		"a missing file": {
			func(e *fakeExecutor) { e.script("test -f", executor.Result{ExitCode: 1}) },
			"not all present",
		},
		"units not enabled": {
			func(e *fakeExecutor) {
				e.script("systemctl is-enabled", executor.Result{ExitCode: 1, Stdout: "disabled\n"})
			},
			"not enabled",
		},
		"a stale unit from an earlier config": {
			func(e *fakeExecutor) { e.files[execUnitPath] = []byte("[Unit]\nDescription=stale\n") },
			"",
		},
	} {
		t.Run(name, func(t *testing.T) {
			e := wired()
			tc.break_(e)
			err := wireStep().Verify(context.Background(), e, &State{Wire: testWire()})
			if err == nil {
				t.Fatalf("verify passed on a target broken by: %s", name)
			}
			if tc.want != "" && !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error %q does not say %q", err, tc.want)
			}
		})
	}
}

// The stale-unit case deserves its own assertion: existence plus enabled is
// not enough, because the units on the target could be leftovers from a run
// with a different chain. Byte-comparing against the CURRENT render is what
// turns a config change into "not verified" instead of "setup complete" while
// the old units keep running.
func TestWire_VerifyComparesTheUnitsAgainstTheCurrentConfig(t *testing.T) {
	otherWire := testWire()
	otherWire.ChainID = 943
	otherExec, otherBeacon, err := catalog.RenderUnits(otherWire)
	if err != nil {
		t.Fatalf("RenderUnits: %v", err)
	}

	e := newFakeExecutor().
		script("test -f", executor.Result{ExitCode: 0}).
		script("stat -c", executor.Result{Stdout: catalog.ServiceUser + "\n"}).
		script("systemctl is-enabled", executor.Result{ExitCode: 0, Stdout: "enabled\nenabled\n"})
	// The target is wired — for the WRONG chain.
	e.files[execUnitPath] = []byte(otherExec)
	e.files[beaconUnitPath] = []byte(otherBeacon)

	if err := wireStep().Verify(context.Background(), e, &State{Wire: testWire()}); err == nil {
		t.Fatal("a target wired for another chain verified as set up for this one")
	}
}
