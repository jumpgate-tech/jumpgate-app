package setup

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/valve-tech/jumpgate/internal/catalog"
	"github.com/valve-tech/jumpgate/internal/executor"
)

// ---- wire: restart only what changed ----

// wiredTarget is a fully wired box: JWT, both units on disk with the given
// content, ownership marker in place, units enabled.
func wiredTarget(execUnit, beaconUnit string) *fakeExecutor {
	e := newFakeExecutor().
		script("test -f", executor.Result{ExitCode: 0}).
		script("stat -c %U", executor.Result{Stdout: catalog.ServiceUser + "\n"}).
		script("cat '"+catalog.OwnerMarkerPath(testWire())+"'", executor.Result{Stdout: catalog.ServiceUser + "\n"}).
		script("systemctl is-enabled", executor.Result{Stdout: "enabled\nenabled\n"})
	e.files[catalog.UnitPath(catalog.LegacyNames.ExecUnit())] = []byte(execUnit)
	e.files[catalog.UnitPath(catalog.LegacyNames.BeaconUnit())] = []byte(beaconUnit)
	return e
}

func TestWireRestartsOnlyTheChangedUnit(t *testing.T) {
	w := testWire()
	_, wantBeacon, err := catalog.RenderUnits(w)
	if err != nil {
		t.Fatal(err)
	}
	e := wiredTarget("an older exec unit", wantBeacon)
	if err := wireStep().Run(context.Background(), e, &State{Wire: w}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	execU, beaconU := catalog.LegacyNames.ExecUnit(), catalog.LegacyNames.BeaconUnit()
	if !ran(e, "systemctl restart "+execU) {
		t.Errorf("the changed exec unit was not restarted; ran %q", e.callLog())
	}
	for _, c := range e.callLog() {
		if (strings.Contains(c, "systemctl stop") || strings.Contains(c, "systemctl restart")) && strings.Contains(c, beaconU) {
			t.Errorf("the unchanged beacon was stopped or restarted: %q", c)
		}
		if c == "WriteFile "+catalog.UnitPath(beaconU) {
			t.Errorf("the unchanged beacon unit was rewritten")
		}
	}
}

func TestWireNoChangeTouchesNothing(t *testing.T) {
	w := testWire()
	wantExec, wantBeacon, err := catalog.RenderUnits(w)
	if err != nil {
		t.Fatal(err)
	}
	e := wiredTarget(wantExec, wantBeacon)
	if err := wireStep().Run(context.Background(), e, &State{Wire: w}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	for _, c := range e.callLog() {
		if strings.Contains(c, "systemctl stop") || strings.Contains(c, "chown") ||
			strings.Contains(c, "systemctl restart") || strings.HasPrefix(c, "WriteFile ") {
			t.Errorf("an unchanged box was touched: %q", c)
		}
	}
}

// ---- install: the binary probe (D9) ----

var boardHelp = moduleHelp{Flag: "--msgboard.block-range", Support: "message board"}

const d9Text = "reth at /usr/local/bin/reth has no message board support (`reth node --help` lists no `--msgboard.*` flags), even after a rebuild from valve-tech/reth main."

func rethClient(t *testing.T) catalog.Client {
	t.Helper()
	c, ok := catalog.ClientByID("reth")
	if !ok {
		t.Fatal("catalog missing reth")
	}
	return c
}

func TestInstallVerifyFailsWithoutHelpFlag(t *testing.T) {
	step := installStep("install-exec", "Install", rethClient(t), boardHelp)
	st := &State{Wire: testWire()}

	without := newFakeExecutor().script("node --help", executor.Result{Stdout: "Usage: reth node [OPTIONS]\n  --http.api <API>\n"})
	if err := step.Verify(context.Background(), without, st); err == nil {
		t.Fatal("Verify passed on a reth whose --help lacks --msgboard.block-range")
	}

	with := newFakeExecutor().script("node --help", executor.Result{Stdout: "  --msgboard.block-range <N>\n"})
	if err := step.Verify(context.Background(), with, st); err != nil {
		t.Fatalf("Verify with the flag: %v", err)
	}

	// No module on: the help is never read.
	plain := installStep("install-exec", "Install", rethClient(t))
	e := newFakeExecutor()
	if err := plain.Verify(context.Background(), e, st); err != nil {
		t.Fatalf("Verify without modules: %v", err)
	}
	if ran(e, "--help") {
		t.Error("a box with no module on read reth --help")
	}
}

func TestRebuildStillMissingNeverWritesUnit(t *testing.T) {
	e := newFakeExecutor().script("node --help", executor.Result{Stdout: "Usage: reth node\n"})
	steps := []Step{installStep("install-exec", "Install", rethClient(t), boardHelp), wireStep()}
	err := RunAll(context.Background(), e, steps, &State{Wire: testWire(), Events: make(chan Event, 100)})
	if err == nil {
		t.Fatal("RunAll passed with a reth that still lacks msgboard")
	}
	// The step's own error is the D9 text; RunAll wraps it with the step id.
	if !strings.HasSuffix(err.Error(), ": "+d9Text) {
		t.Errorf("error = %q, want the D9 text %q", err, d9Text)
	}
	if got := installStep("install-exec", "Install", rethClient(t), boardHelp).Run(context.Background(), e, &State{Wire: testWire()}); got == nil || got.Error() != d9Text {
		t.Errorf("Run error = %v, want exactly %q", got, d9Text)
	}
	for _, c := range e.callLog() {
		if c == "WriteFile "+catalog.UnitPath(catalog.LegacyNames.ExecUnit()) {
			t.Fatal("the exec unit was written for a binary that rejects it")
		}
	}
}

// ---- the modules step (D9 live probe) ----

func withNodeModules(t *testing.T, mods ...catalog.Module) {
	t.Helper()
	old := moduleByID
	moduleByID = func(id string) (catalog.Module, bool) {
		for _, m := range mods {
			if m.ID == id {
				return m, true
			}
		}
		return catalog.Module{}, false
	}
	t.Cleanup(func() { moduleByID = old })
}

func shortHandshake(t *testing.T) {
	t.Helper()
	oldT, oldP := handshakeTimeout, handshakePollInterval
	handshakeTimeout, handshakePollInterval = 50*time.Millisecond, time.Millisecond
	t.Cleanup(func() { handshakeTimeout, handshakePollInterval = oldT, oldP })
}

func TestModulesStepMapsErrors(t *testing.T) {
	withNodeModules(t, catalog.Module{ID: "msgboard", Name: "Message board", Placement: catalog.PlaceNode,
		Probe: catalog.ModuleProbe{HelpContains: "--msgboard.block-range", StatusMethod: "msgboard_status"}})
	shortHandshake(t)
	w := testWire()
	w.Modules = []catalog.NodeModule{{ID: "msgboard"}}
	notFound := executor.Result{Stdout: `{"jsonrpc":"2.0","id":1,"error":{"code":-32601,"message":"the method msgboard_status does not exist"}}`}
	const want = "reth is running but not serving msgboard. The unit's --http.api may have been edited."

	e := newFakeExecutor().script("msgboard_status", notFound)
	err := modulesStep().Run(context.Background(), e, &State{Wire: w})
	if err == nil || err.Error() != want {
		t.Fatalf("err = %v, want %q", err, want)
	}

	line := "Oct 23 10:00:00 box reth[1]: Error: Unknown RPC module: 'msgboard'"
	e = newFakeExecutor().script("msgboard_status", notFound).
		script("journalctl", executor.Result{Stdout: "Oct 23 09:59:59 box reth[1]: starting\n" + line + "\n"})
	err = modulesStep().Run(context.Background(), e, &State{Wire: w})
	if err == nil || err.Error() != want+"\n"+line {
		t.Fatalf("err = %v, want the journal line appended", err)
	}

	ok := newFakeExecutor().script("msgboard_status", executor.Result{Stdout: `{"jsonrpc":"2.0","id":1,"result":{"enabled":true}}`})
	if err := modulesStep().Verify(context.Background(), ok, &State{Wire: w}); err != nil {
		t.Fatalf("Verify on a serving node: %v", err)
	}

	// Without node modules the step is a no-op that runs nothing.
	none := newFakeExecutor()
	if err := modulesStep().Verify(context.Background(), none, &State{Wire: testWire()}); err != nil || len(none.callLog()) != 0 {
		t.Fatalf("no-module Verify: err %v, calls %q", err, none.callLog())
	}
}

// ---- PlanBoxModule ----

func withBoxModule(t *testing.T, m catalog.Module, u catalog.BoxUnit) {
	t.Helper()
	withNodeModules(t, m)
	oldR, oldV, oldP := renderBoxModule, validateBoxModules, probeBoxModule
	renderBoxModule = func(catalog.BoxModule) (catalog.BoxUnit, error) { return u, nil }
	validateBoxModules = func([]catalog.BoxModule, *catalog.WireConfig) error { return nil }
	// monitor.ProbeBoxModule checks the real registry, which has no box
	// modules yet; stand in its systemd reading.
	probeBoxModule = func(ctx context.Context, e executor.Executor, b catalog.BoxModule) (bool, error) {
		res, err := e.Run(ctx, "systemctl is-active '"+b.UnitName()+"'", nil)
		return err == nil && strings.TrimSpace(res.Stdout) == "active", err
	}
	t.Cleanup(func() { renderBoxModule, validateBoxModules, probeBoxModule = oldR, oldV, oldP })
}

func watchModule(t *testing.T) catalog.BoxModule {
	withBoxModule(t, catalog.Module{ID: "watch", Name: "Health watch", Placement: catalog.PlaceBox},
		catalog.BoxUnit{
			Unit: "[Unit]\nDescription=watch\n",
			Files: map[string]string{
				"/var/lib/jumpgate/modules/watch/module.json": `{"id":"watch"}`,
				"/var/lib/jumpgate/modules/watch/watch.yaml":  "endpoints: []\n",
			},
		})
	return catalog.BoxModule{ID: "watch"}
}

func ids(steps []Step) []string {
	var out []string
	for _, s := range steps {
		out = append(out, s.ID)
	}
	return out
}

func TestPlanBoxModuleSteps(t *testing.T) {
	b := watchModule(t)
	on, err := PlanBoxModule(b, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := strings.Join(ids(on), ","), "preflight,account,toolchain,install,write-files,unit,start,probe"; got != want {
		t.Errorf("enable ids = %s, want %s", got, want)
	}
	off, err := PlanBoxModule(b, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(ids(off), ","); got != "stop,remove" {
		t.Errorf("disable ids = %s, want stop,remove", got)
	}
}

func TestPlanBoxModuleNeedsGateForWatch(t *testing.T) {
	b := watchModule(t)
	steps, err := PlanBoxModule(b, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	e := newFakeExecutor().script("test -x "+gateBinary, executor.Result{ExitCode: 1})
	var install Step
	for _, s := range steps {
		if s.ID == "install" {
			install = s
		}
	}
	err = install.Verify(context.Background(), e, &State{})
	const want = "The health watch runs the box's gate. Pair this box first."
	if err == nil || err.Error() != want {
		t.Fatalf("err = %v, want %q", err, want)
	}
}

// Root must never chown, chmod or write through a path the service account
// could have swapped for a symlink: files are staged in a root-only dir
// under a random name and installed with `install -o -g -m`, and nothing
// uses a fixed ".tmp" name beside the destination.
func TestWriteFilesNeverFollowsServiceOwnedPaths(t *testing.T) {
	b := watchModule(t)
	e := boxTarget()
	steps, err := PlanBoxModule(b, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	var wf Step
	for _, s := range steps {
		if s.ID == "write-files" {
			wf = s
		}
	}
	if err := wf.Run(context.Background(), e, &State{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if err := wf.Verify(context.Background(), e, &State{}); err != nil {
		t.Fatalf("Verify after Run: %v", err)
	}
	data := b.DataDir()
	n := catalog.CurrentNames
	for _, c := range e.callLog() {
		if strings.Contains(c, ".tmp") {
			t.Errorf("fixed .tmp path used: %q", c)
		}
		// Nothing privileged runs inside the service-owned dir: root only
		// creates the leaf and hands it over.
		if strings.Contains(c, data+"/") {
			t.Errorf("root operated inside the service-owned dir: %q", c)
		}
		if strings.Contains(c, "WriteFile ") && !strings.HasPrefix(c, "WriteFile "+stagingDir+"/") {
			t.Errorf("root wrote outside the staging dir: %q", c)
		}
	}
	if !ran(e, "mkdir -p '"+data+"' && chown -h "+n.User+":"+n.Group+" '"+data+"'") {
		t.Errorf("data dir not created and handed over; ran %q", e.callLog())
	}
	if !ran(e, "install -d -o root -g root -m 0755 '"+catalog.BoxModulesConfigRoot+"' '"+b.ConfigDir()+"'") {
		t.Errorf("config dir not root-owned 0755; ran %q", e.callLog())
	}
	// Owner and mode are set on the staged file, then it is moved into the
	// root-owned config dir in one rename.
	for f, own := range map[string]string{
		b.SidecarPath():               "chown root:root '%s' && chmod 0644 '%s' && mv -T '%s' '" + b.SidecarPath() + "'",
		b.ConfigDir() + "/watch.yaml": "chown root:" + n.Group + " '%s' && chmod 0640 '%s' && mv -T '%s' '" + b.ConfigDir() + "/watch.yaml'",
	} {
		found := false
		for _, c := range e.callLog() {
			if i := strings.Index(c, "chown root:"); i >= 0 && strings.Contains(c, "'"+f+"'") {
				staged := strings.SplitN(c[i:], "'", 3)[1]
				if strings.HasPrefix(staged, stagingDir+"/") && strings.Contains(c, fmt.Sprintf(own, staged, staged, staged)) {
					found = true
				}
			}
		}
		if !found {
			t.Errorf("%s not staged, owned and moved in; ran %q", f, e.callLog())
		}
	}
}

// A record that is not a registered box module plans nothing, even to turn
// it off, so its id never reaches a command.
func TestPlanBoxModuleRefusesUnknownIDEvenToDisable(t *testing.T) {
	watchModule(t)
	for _, enable := range []bool{true, false} {
		if _, err := PlanBoxModule(catalog.BoxModule{ID: "x; touch /tmp/pwned"}, nil, enable); err == nil {
			t.Errorf("enable=%v: planned a hostile id", enable)
		}
	}
}

func boxTarget() *fakeExecutor {
	return newFakeExecutor().
		script("uname", executor.Result{Stdout: "Linux\n"}).
		script("id -u", executor.Result{Stdout: "0\n"}).
		script("systemctl is-active", executor.Result{Stdout: "active\n"})
}

func TestPlanBoxModuleIdempotent(t *testing.T) {
	b := watchModule(t)
	shortHandshake(t)
	e := boxTarget()
	steps, err := PlanBoxModule(b, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	if err := RunAll(context.Background(), e, steps, &State{Events: make(chan Event, 100)}); err != nil {
		t.Fatalf("first RunAll: %v", err)
	}
	if got := string(e.files[catalog.UnitPath(b.UnitName())]); got != "[Unit]\nDescription=watch\n" {
		t.Fatalf("unit not written: %q", got)
	}

	steps, _ = PlanBoxModule(b, nil, true)
	for i := range steps {
		id := steps[i].ID
		if steps[i].Run != nil {
			steps[i].Run = func(context.Context, executor.Executor, *State) error {
				t.Errorf("second run called Run on %s", id)
				return nil
			}
		}
	}
	if err := RunAll(context.Background(), e, steps, &State{Events: make(chan Event, 100)}); err != nil {
		t.Fatalf("second RunAll: %v", err)
	}
}
