package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/valve-tech/jumpgate/internal/catalog"
	"github.com/valve-tech/jumpgate/internal/executor"
)

var errInjected = errors.New("injected: the command could not run")

// faultExec is a scripted box whose n-th operation (a command, a file write
// or a file read) cannot run, as when the SSH session or the shell dies
// mid-step. It records every operation that came after the failure.
type faultExec struct {
	*scriptExec
	failAt, n int
	after     []string
}

func (f *faultExec) op(what string) error {
	f.n++
	switch {
	case f.n == f.failAt:
		return errInjected
	case f.failAt > 0 && f.n > f.failAt:
		f.after = append(f.after, what)
	}
	return nil
}

func (f *faultExec) Run(ctx context.Context, cmd string, o *executor.RunOpts) (executor.Result, error) {
	if err := f.op(cmd); err != nil {
		return executor.Result{}, err
	}
	return f.scriptExec.Run(ctx, cmd, o)
}

func (f *faultExec) WriteFile(ctx context.Context, path string, b []byte, m os.FileMode) error {
	if err := f.op("write " + path); err != nil {
		return err
	}
	return f.scriptExec.WriteFile(ctx, path, b, m)
}

func (f *faultExec) ReadFile(ctx context.Context, path string) ([]byte, error) {
	if err := f.op("read " + path); err != nil {
		return nil, err
	}
	return f.scriptExec.ReadFile(ctx, path)
}

// faultCase is a step on a box where it succeeds: box builds the script
// afresh for every run, and results are the earlier steps' records.
type faultCase struct {
	step    string
	box     func() *scriptExec
	wire    catalog.WireConfig
	results map[string]json.RawMessage
}

func faultCases() map[string]faultCase {
	pre := preflightResult(true, true)
	return map[string]faultCase{
		"preflight on a legacy box": {step: "preflight", box: legacyBox, wire: legacyWire()},
		"preflight finishing a half-migrated box": {step: "preflight", wire: migratedWire(), box: func() *scriptExec {
			return migratedBox().on("test -e '/etc/systemd/system/valve-node-app-exec.service'", ok0)
		}},
		"stop-units": {step: "stop-units", wire: legacyWire(), box: func() *scriptExec { return legacyBox().on("pgrep -a -u", fail1) }},
		"rename-account": {step: "rename-account", wire: legacyWire(), box: func() *scriptExec {
			return legacyBox().on("id -u jumpgate-node", fail1, idOK("998")).on("getent group jumpgate-node", fail1, stdout("jumpgate-node:x:998:\n"))
		}},
		"move-data": {step: "move-data", wire: legacyWire(), box: func() *scriptExec {
			return newScript().on("test -d '/var/lib/valve-node-app'", ok0).on("test -e '/var/lib/jumpgate-node'", fail1).on("findmnt -rn -o TARGET", stdout("/\n"))
		}},
		"rewrite-units moving the snapshot marker": {step: "rewrite-units", wire: legacyWire(),
			results: map[string]json.RawMessage{"move-data": movedResult(), "preflight": pre},
			box: func() *scriptExec {
				return newScript().on("cat '"+catalog.SnapshotMarkerPath(legacyWire())+"'", stdout("/var/lib/valve-node-app/369\n"))
			}},
		"ownership walking the tree": {step: "ownership", wire: migratedWire(), box: func() *scriptExec {
			return newScript().on("stat -c %U", stdout("UNKNOWN\n")).on("cat '"+catalog.OwnerMarkerPath(migratedWire())+"'", stdout("valve-node-app\n"))
		}},
		"start-units": {step: "start-units", wire: migratedWire(),
			results: map[string]json.RawMessage{"preflight": json.RawMessage(`{"wasActive":{"exec":true,"beacon":true}}`)},
			box: func() *scriptExec {
				return newScript().on("systemctl is-active jumpgate-node-", stdout("active\n"))
			}},
		"verify": {step: "verify", wire: migratedWire(), box: migratedBox, results: map[string]json.RawMessage{"preflight": pre}},
	}
}

func (c faultCase) run(t *testing.T, failAt int) (*faultExec, Outcome, error) {
	t.Helper()
	ex := &faultExec{scriptExec: c.box(), failAt: failAt}
	env := mnEnv(t, ex.scriptExec, c.wire)
	env.Exec = ex
	for k, v := range c.results {
		env.Results[k] = v
	}
	o, err := step(t, c.step).Run(context.Background(), env)
	return ex, o, err
}

// Every step of node.migrate-names runs as root. If any one command, file
// write or read cannot run, the step must fail with that error and do
// nothing more: carrying on would act on a box it could not inspect (a chown
// after a failed stat, a start after a failed rewrite), and a step that
// reported success would never be retried.
func TestEveryMigrateNamesStepStopsAtAnOperationThatCannotRun(t *testing.T) {
	for name, c := range faultCases() {
		t.Run(name, func(t *testing.T) {
			ex, o, err := c.run(t, 0)
			if err != nil || o.Ask != nil {
				t.Fatalf("the happy path does not succeed: %+v, %v", o, err)
			}
			total := ex.n
			if total == 0 {
				t.Fatal("the step ran nothing")
			}
			for i := 1; i <= total; i++ {
				ex, o, err := c.run(t, i)
				what := fmt.Sprintf("operation %d of %d", i, total)
				if !errors.Is(err, errInjected) {
					t.Errorf("%s failed: step returned %+v, %v, want the injected error", what, o, err)
				}
				if len(ex.after) > 0 {
					t.Errorf("%s failed: the step went on to %q", what, ex.after)
				}
			}
		})
	}
}

// Preflight refuses a box whose accounts do not match what node.json says,
// before a single change: there is no account to rename, or node.json was
// switched while the legacy account still runs the node.
func TestPreflightRefusesAccountsThatDoNotMatchNodeJSON(t *testing.T) {
	cases := map[string]struct {
		ex   *scriptExec
		w    catalog.WireConfig
		want string
	}{
		"no account":                     {legacyBox().on("id -u valve-node-app", fail1), legacyWire(), "set the node up first"},
		"node.json ahead of the account": {legacyBox(), migratedWire(), "run setup to repair the box"},
	}
	for name, c := range cases {
		_, err := step(t, "preflight").Run(context.Background(), mnEnv(t, c.ex, c.w))
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: preflight = %v, want %q", name, err, c.want)
		}
		noCommand(t, c.ex, "systemctl stop", "usermod", "groupmod", "mv ", "chown")
	}
}

// The steps that restore what preflight saw cannot guess it: without its
// record they refuse rather than start or enable units on a default.
func TestStepsThatRestoreRefuseWithoutThePreflightRecord(t *testing.T) {
	for _, id := range []string{"rewrite-units", "start-units", "verify"} {
		ex := migratedBox()
		env := mnEnv(t, ex, migratedWire())
		env.Results["move-data"] = movedResult()
		if _, err := step(t, id).Run(context.Background(), env); err == nil || !strings.Contains(err.Error(), "preflight left no result") {
			t.Errorf("%s without preflight's record: %v", id, err)
		}
		noCommand(t, ex, "systemctl start", "systemctl enable")
	}
}
