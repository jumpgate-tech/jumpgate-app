package jobs

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/valve-tech/jumpgate/internal/catalog"
	"github.com/valve-tech/jumpgate/internal/executor"
	"github.com/valve-tech/jumpgate/internal/intent"
)

func legacyWire() catalog.WireConfig {
	return catalog.WireConfig{ChainID: 369, ExecID: "reth", BeaconID: "lighthouse-pulse",
		DataDir: "/var/lib/valve-node-app/369", JWTPath: "/var/lib/valve-node-app/369/jwt.hex"}
}

func migratedWire() catalog.WireConfig {
	w := legacyWire()
	w.Names, w.DataDir, w.JWTPath = catalog.NamesCurrent, "/var/lib/jumpgate-node/369", "/var/lib/jumpgate-node/369/jwt.hex"
	return w
}

func mnEnv(t *testing.T, ex *scriptExec, w catalog.WireConfig) Env {
	t.Helper()
	dir := t.TempDir()
	b, err := json.Marshal(w)
	if err != nil {
		t.Fatal(err)
	}
	node := filepath.Join(dir, "node.json")
	if err := os.WriteFile(node, b, 0o600); err != nil {
		t.Fatal(err)
	}
	return Env{Exec: ex, NodePath: node, NamesPath: filepath.Join(dir, "names.json"), Out: io.Discard, Results: map[string]json.RawMessage{}}
}

func step(t *testing.T, id string) StepDef {
	t.Helper()
	p, ok := PlanFor(MigrateNamesVersion)
	if !ok {
		t.Fatal("node.migrate-names is not registered")
	}
	i := p.index(id)
	if i < 0 {
		t.Fatalf("no step %q", id)
	}
	return p.Steps[i]
}

func readWire(t *testing.T, env Env) catalog.WireConfig {
	t.Helper()
	b, err := os.ReadFile(env.NodePath)
	if err != nil {
		t.Fatal(err)
	}
	var w catalog.WireConfig
	if err := json.Unmarshal(b, &w); err != nil {
		t.Fatal(err)
	}
	return w
}

func noCommand(t *testing.T, ex *scriptExec, substrs ...string) {
	t.Helper()
	for _, c := range ex.calls {
		for _, s := range substrs {
			if strings.Contains(c, s) {
				t.Fatalf("ran %q", c)
			}
		}
	}
}

func TestThePlanIsRegisteredInOrder(t *testing.T) {
	p, ok := PlanForKind(intent.KindNodeMigrateNames)
	if !ok || p.Version != MigrateNamesVersion {
		t.Fatalf("plan %+v", p)
	}
	var ids []string
	for _, s := range p.Steps {
		ids = append(ids, s.ID)
		if s.AbortNote == "" {
			t.Errorf("step %s does not say what an abort leaves (Q8)", s.ID)
		}
	}
	if strings.Join(ids, ",") != "preflight,stop-units,rename-account,move-data,rewrite-units,ownership,start-units,verify" {
		t.Fatalf("steps %v", ids)
	}
}

func TestThePlanTakesNoInputs(t *testing.T) {
	p, _ := PlanFor(MigrateNamesVersion)
	for _, in := range []string{`{}`, `null`} {
		if err := p.Validate(json.RawMessage(in)); err != nil {
			t.Errorf("Validate(%s) = %v", in, err)
		}
	}
	for _, in := range []string{`{"dataDir":"/"}`, `[]`} {
		if err := p.Validate(json.RawMessage(in)); err == nil {
			t.Errorf("Validate(%s) accepted inputs the plan never reads", in)
		}
	}
}

func TestPreflightRecordsWhatRuns(t *testing.T) {
	ex := legacyBox().on("systemctl is-active valve-node-app-exec.service", stdout("active\n")).
		on("systemctl is-active valve-node-app-beacon.service", executor.Result{Stdout: "inactive\n", ExitCode: 3})
	o, err := step(t, "preflight").Run(context.Background(), mnEnv(t, ex, legacyWire()))
	if err != nil {
		t.Fatal(err)
	}
	var pre migratePreflight
	if err := json.Unmarshal(o.Result, &pre); err != nil {
		t.Fatal(err)
	}
	if !pre.WasActive["exec"] || pre.WasActive["beacon"] || pre.AlreadyDone {
		t.Fatalf("preflight %+v", pre)
	}
}

// Q3: a gateway unit running as the legacy account would break; refuse
// before changing anything.
func TestPreflightRefusesOtherUnitsOfTheLegacyUser(t *testing.T) {
	ex := legacyBox().on("grep -l -x", stdout("/etc/systemd/system/valve-node-app-exec.service\n/etc/systemd/system/valve-node-app-erpc.service\n"))
	_, err := step(t, "preflight").Run(context.Background(), mnEnv(t, ex, legacyWire()))
	if err == nil || !strings.Contains(err.Error(), "valve-node-app-erpc.service") || strings.Contains(err.Error(), "valve-node-app-exec.service") {
		t.Fatalf("preflight = %v", err)
	}
	noCommand(t, ex, "systemctl stop", "usermod")
}

// Moving the data root breaks any other unit that names a path in it, even
// one that runs as another account.
func TestPreflightRefusesUnitsThatUseTheLegacyDataRoot(t *testing.T) {
	ex := legacyBox().on("grep -l -F", stdout("/etc/systemd/system/valve-node-app-beacon.service\n/etc/systemd/system/backup.service\n"))
	_, err := step(t, "preflight").Run(context.Background(), mnEnv(t, ex, legacyWire()))
	if err == nil || !strings.Contains(err.Error(), "backup.service") || strings.Contains(err.Error(), "beacon.service") {
		t.Fatalf("preflight = %v", err)
	}
	// A custom data dir does not move, so nothing else can be broken by it.
	w := legacyWire()
	w.DataDir, w.JWTPath = "/data/pulse", "/data/pulse/jwt.hex"
	ex = legacyBox().on("grep -l -F", stdout("/etc/systemd/system/backup.service\n"))
	if _, err := step(t, "preflight").Run(context.Background(), mnEnv(t, ex, w)); err != nil {
		t.Fatalf("custom data dir: %v", err)
	}
}

func TestPreflightRefusesBothAccounts(t *testing.T) {
	ex := legacyBox().on("id -u jumpgate-node", idOK("997"))
	if _, err := step(t, "preflight").Run(context.Background(), mnEnv(t, ex, legacyWire())); err == nil || !strings.Contains(err.Error(), "both") {
		t.Fatalf("preflight = %v", err)
	}
}

func TestPreflightRefusesBothDataRoots(t *testing.T) {
	ex := legacyBox().on("test -e '/var/lib/jumpgate-node'", ok0)
	_, err := step(t, "preflight").Run(context.Background(), mnEnv(t, ex, legacyWire()))
	if err == nil || !strings.Contains(err.Error(), "/var/lib/jumpgate-node") {
		t.Fatalf("preflight = %v", err)
	}
	noCommand(t, ex, "systemctl stop")
}

// A data dir that overlaps jumpgate's own trees is refused before any
// command runs: every later step would chown, move or render it.
func TestPreflightRefusesADataDirItMustNotTouch(t *testing.T) {
	for _, dir := range []string{"/var/lib/jumpgate/369", "/var/lib", "relative/369", "/var/lib/valve-node-app/369\nExecStartPre=/bin/sh"} {
		w := legacyWire()
		w.DataDir, w.JWTPath = dir, ""
		ex := legacyBox()
		if _, err := step(t, "preflight").Run(context.Background(), mnEnv(t, ex, w)); err == nil {
			t.Errorf("data dir %q accepted", dir)
		}
		if len(ex.calls) != 0 {
			t.Errorf("data dir %q: ran %v", dir, ex.calls)
		}
	}
	w := legacyWire()
	w.JWTPath = "/etc/jumpgate/agent.key"
	if _, err := step(t, "preflight").Run(context.Background(), mnEnv(t, legacyBox(), w)); err == nil {
		t.Error("a jwt path inside /etc/jumpgate was accepted")
	}
}

func TestAMigratedBoxIsANoOp(t *testing.T) {
	ex := migratedBox()
	env := mnEnv(t, ex, migratedWire())
	o, err := step(t, "preflight").Run(context.Background(), env)
	if err != nil {
		t.Fatal(err)
	}
	env.Results["preflight"] = o.Result
	for _, id := range []string{"stop-units", "rename-account", "move-data", "rewrite-units", "ownership", "start-units"} {
		if _, err := step(t, id).Run(context.Background(), env); err != nil {
			t.Fatalf("%s: %v", id, err)
		}
	}
	noCommand(t, ex, "systemctl stop", "usermod", "groupmod", "mv ", "chown", "systemctl start", "mktemp")
}

// A job stopped after the rename (aborted, or its controller gone) leaves the
// box half done; the next job finishes it rather than calling it done.
func TestPreflightFinishesAHalfMigratedBox(t *testing.T) {
	cases := map[string]struct {
		ex *scriptExec
		w  catalog.WireConfig
	}{
		"renamed, units not rewritten":       {migratedBox().on("getent group valve-node-app", stdout("valve-node-app:x:998:\n")).on("getent group jumpgate-node", fail1).on("test -e '/var/lib/jumpgate-node'", fail1), legacyWire()},
		"legacy group left":                  {migratedBox().on("getent group valve-node-app", stdout("valve-node-app:x:998:\n")).on("getent group jumpgate-node", fail1), migratedWire()},
		"legacy unit left":                   {migratedBox().on("test -e '/etc/systemd/system/valve-node-app-exec.service'", ok0), migratedWire()},
		"owner marker names the old account": {migratedBox().on("cat '"+catalog.OwnerMarkerPath(migratedWire())+"'", stdout("valve-node-app\n")), migratedWire()},
		"snapshot marker names the old dir":  {migratedBox().on("cat '"+catalog.SnapshotMarkerPath(migratedWire())+"'", stdout("/var/lib/valve-node-app/369\n")), migratedWire()},
	}
	for name, c := range cases {
		o, err := step(t, "preflight").Run(context.Background(), mnEnv(t, c.ex, c.w))
		var pre migratePreflight
		_ = json.Unmarshal(o.Result, &pre)
		if err != nil || pre.AlreadyDone {
			t.Errorf("%s: preflight %+v, %v", name, pre, err)
		}
	}
}

// Review Focus (also pinned): a process ignoring SIGTERM is killed only on
// the operator's say-so.
func TestStopUnitsAsksBeforeKillingProcesses(t *testing.T) {
	ex := legacyBox().on("pgrep -a -u valve-node-app", stdout("4242 reth node\n"))
	env := mnEnv(t, ex, legacyWire())
	o, err := step(t, "stop-units").Run(context.Background(), env)
	if err != nil || o.Ask == nil || o.Ask.Key != "busy-user" || !strings.Contains(o.Ask.Prompt, "4242") {
		t.Fatalf("stop-units = %+v, %v", o.Ask, err)
	}
	if ex.ran("pkill") {
		t.Fatal("processes were signalled before anyone decided")
	}
	ex.on("pgrep -a -u valve-node-app", stdout("4242 reth node\n"), fail1)
	env.Answer = &Answer{Key: "busy-user", Option: "stop-processes"}
	prev := stopGrace
	stopGrace = 0 // no real wait in tests
	t.Cleanup(func() { stopGrace = prev })
	if o, err := step(t, "stop-units").Run(context.Background(), env); err != nil || o.Ask != nil {
		t.Fatalf("after stop-processes: %+v, %v", o.Ask, err)
	}
	if !ex.ran("pkill -TERM -u valve-node-app") || ex.ran("pkill -KILL") {
		t.Fatalf("calls %v", ex.calls)
	}
}

// Consenting to SIGTERM is not consenting to SIGKILL: a process that outlives
// SIGTERM is listed again, and killed only when someone chooses that.
func TestStopUnitsAsksAgainBeforeSIGKILL(t *testing.T) {
	prev := stopGrace
	stopGrace = 0
	t.Cleanup(func() { stopGrace = prev })
	ex := legacyBox().on("pgrep -a -u valve-node-app", stdout("4242 reth node\n"))
	env := mnEnv(t, ex, legacyWire())
	env.Answer = &Answer{Key: "busy-user", Option: "stop-processes"}
	o, err := step(t, "stop-units").Run(context.Background(), env)
	if err != nil || o.Ask == nil || o.Ask.Key != "busy-user" || !strings.Contains(o.Ask.Prompt, "SIGTERM") {
		t.Fatalf("stop-units = %+v, %v", o.Ask, err)
	}
	var kill bool
	for _, opt := range o.Ask.Options {
		kill = kill || opt.ID == "kill-processes"
	}
	if !kill || ex.ran("pkill -KILL") {
		t.Fatalf("options %+v, calls %v", o.Ask.Options, ex.calls)
	}
	ex.on("pgrep -a -u valve-node-app", stdout("4242 reth node\n"), fail1)
	env.Answer = &Answer{Key: "busy-user", Option: "kill-processes"}
	if o, err := step(t, "stop-units").Run(context.Background(), env); err != nil || o.Ask != nil {
		t.Fatalf("after kill-processes: %+v, %v", o.Ask, err)
	}
	if !ex.ran("pkill -KILL -u valve-node-app") {
		t.Fatalf("calls %v", ex.calls)
	}
}

func TestStopUnitsStopsEveryUnitAndChecksThem(t *testing.T) {
	ex := legacyBox().on("pgrep -a -u", fail1)
	if _, err := step(t, "stop-units").Run(context.Background(), mnEnv(t, ex, legacyWire())); err != nil {
		t.Fatal(err)
	}
	for _, u := range append(catalog.LegacyNames.Units(), catalog.CurrentNames.Units()...) {
		if !ex.ran("systemctl stop " + u) {
			t.Errorf("%s was not stopped: %v", u, ex.calls)
		}
	}
	ex = legacyBox().on("pgrep -a -u", fail1).on("systemctl is-active valve-node-app-exec.service", stdout("deactivating\n"))
	if _, err := step(t, "stop-units").Run(context.Background(), mnEnv(t, ex, legacyWire())); err == nil || !strings.Contains(err.Error(), "deactivating") {
		t.Fatalf("a unit still stopping passed: %v", err)
	}
}

// P3: the new account is absent before the rename and present after it:
// rename-account probes it on both sides of the usermod.
func TestRenameAccountRenamesUserThenGroup(t *testing.T) {
	ex := legacyBox().on("id -u jumpgate-node", fail1, idOK("998")).on("getent group jumpgate-node", fail1, stdout("jumpgate-node:x:998:\n"))
	if _, err := step(t, "rename-account").Run(context.Background(), mnEnv(t, ex, legacyWire())); err != nil {
		t.Fatal(err)
	}
	pre, u, g := ex.index("id -u jumpgate-node"), ex.index("usermod -l jumpgate-node -d /var/lib/jumpgate-node valve-node-app"), ex.index("groupmod -n jumpgate-node valve-node-app")
	if pre < 0 || u < pre || g < u {
		t.Fatalf("id at %d, usermod at %d, groupmod at %d: %v", pre, u, g, ex.calls)
	}
}

func TestRenameAccountRefusesAnAccountThatAppeared(t *testing.T) {
	for name, ex := range map[string]*scriptExec{
		"user":  legacyBox().on("id -u jumpgate-node", idOK("997")),
		"group": legacyBox().on("getent group jumpgate-node", stdout("jumpgate-node:x:997:\n")),
	} {
		if _, err := step(t, "rename-account").Run(context.Background(), mnEnv(t, ex, legacyWire())); err == nil {
			t.Errorf("%s: renamed onto an existing %s", name, name)
		}
		noCommand(t, ex, "usermod", "groupmod")
	}
}

func TestRenameAccountFinishesAHalfDoneRename(t *testing.T) {
	// The user was renamed by an earlier attempt; the group was not.
	ex := newScript().on("id -u valve-node-app", fail1).on("id -u jumpgate-node", idOK("998")).
		on("getent group valve-node-app", stdout("valve-node-app:x:998:\n")).on("getent group jumpgate-node", fail1, stdout("jumpgate-node:x:998:\n"))
	if _, err := step(t, "rename-account").Run(context.Background(), mnEnv(t, ex, legacyWire())); err != nil {
		t.Fatal(err)
	}
	if ex.ran("usermod") || !ex.ran("groupmod -n jumpgate-node valve-node-app") {
		t.Fatalf("calls %v", ex.calls)
	}
}

func TestRenameAccountNeedsTheNewGroup(t *testing.T) {
	// A legacy account whose primary group is named otherwise: the units'
	// Group= line would name a group that does not exist.
	ex := legacyBox().on("id -u jumpgate-node", fail1, idOK("998")).on("getent group valve-node-app", fail1)
	if _, err := step(t, "rename-account").Run(context.Background(), mnEnv(t, ex, legacyWire())); err == nil || !strings.Contains(err.Error(), "group") {
		t.Fatalf("rename-account = %v", err)
	}
}

// Review Focus 1: a mount at or under the old root would be left behind by
// fstab at the old path after a reboot.
func TestMoveDataNeverMovesAMountedTree(t *testing.T) {
	for _, mount := range []string{"/var/lib/valve-node-app", "/var/lib/valve-node-app/369/db"} {
		ex := newScript().
			on("test -d '/var/lib/valve-node-app'", ok0).on("test -e '/var/lib/jumpgate-node'", fail1).
			on("findmnt -rn -o TARGET", stdout("/\n/var/lib/valve-node-app2\n"+mount+"\n"))
		env := mnEnv(t, ex, legacyWire())
		o, err := step(t, "move-data").Run(context.Background(), env)
		if err != nil || o.Ask == nil || o.Ask.Key != "mounted-data" || !strings.Contains(o.Ask.Prompt, mount) || strings.Contains(o.Ask.Prompt, "app2") {
			t.Fatalf("move-data = %+v, %v", o, err)
		}
		if ex.ran("mv ") {
			t.Fatal("a mounted tree was moved")
		}
		env.Answer = &Answer{Key: "mounted-data", Option: "keep-path"}
		o, err = step(t, "move-data").Run(context.Background(), env)
		var mv migrateMove
		_ = json.Unmarshal(o.Result, &mv)
		if err != nil || mv.Moved || mv.DataDir != "/var/lib/valve-node-app/369" || ex.ran("mv ") {
			t.Fatalf("keep-path: %+v, %v", mv, err)
		}
		// "Check again" with the mount still there asks again.
		env.Answer = &Answer{Key: "mounted-data", Option: intent.OptRetry}
		if o, err := step(t, "move-data").Run(context.Background(), env); err != nil || o.Ask == nil || ex.ran("mv ") {
			t.Fatalf("retry: %+v, %v", o, err)
		}
	}
}

func TestMoveDataMovesTheRootAndIsIdempotent(t *testing.T) {
	ex := newScript().on("test -d '/var/lib/valve-node-app'", ok0).on("test -e '/var/lib/jumpgate-node'", fail1).
		on("findmnt -rn -o TARGET", stdout("/\n"))
	env := mnEnv(t, ex, legacyWire())
	o, err := step(t, "move-data").Run(context.Background(), env)
	var mv migrateMove
	_ = json.Unmarshal(o.Result, &mv)
	if err != nil || !mv.Moved || mv.DataDir != "/var/lib/jumpgate-node/369" || mv.JWTPath != "/var/lib/jumpgate-node/369/jwt.hex" {
		t.Fatalf("move-data = %+v, %v", mv, err)
	}
	if !ex.ran("mv -T '/var/lib/valve-node-app' '/var/lib/jumpgate-node'") {
		t.Fatalf("calls %v", ex.calls)
	}
	// Already moved: the old root is gone, the new one is there.
	again := newScript().on("test -d '/var/lib/valve-node-app'", fail1).on("test -e '/var/lib/jumpgate-node'", ok0)
	env.Exec = again
	o, err = step(t, "move-data").Run(context.Background(), env)
	mv = migrateMove{}
	_ = json.Unmarshal(o.Result, &mv)
	if err != nil || again.ran("mv ") || !mv.Moved || mv.DataDir != "/var/lib/jumpgate-node/369" {
		t.Fatalf("second run: %s, %v", o.Result, err)
	}
}

func TestMoveDataRefusesBothRoots(t *testing.T) {
	ex := newScript().on("test -d '/var/lib/valve-node-app'", ok0).on("test -e '/var/lib/jumpgate-node'", ok0)
	if _, err := step(t, "move-data").Run(context.Background(), mnEnv(t, ex, legacyWire())); err == nil || ex.ran("mv ") {
		t.Fatalf("move-data = %v, calls %v", err, ex.calls)
	}
}

func TestMoveDataLeavesACustomDataDirAlone(t *testing.T) {
	w := legacyWire()
	w.DataDir, w.JWTPath = "/data/pulse", "/data/pulse/jwt.hex"
	ex := newScript()
	o, err := step(t, "move-data").Run(context.Background(), mnEnv(t, ex, w))
	var mv migrateMove
	_ = json.Unmarshal(o.Result, &mv)
	if err != nil || mv.Moved || mv.DataDir != "/data/pulse" || len(ex.calls) != 0 {
		t.Fatalf("move-data = %+v, %v, calls %v", mv, err, ex.calls)
	}
}

func movedResult() json.RawMessage {
	return json.RawMessage(`{"moved":true,"dataDir":"/var/lib/jumpgate-node/369","jwtPath":"/var/lib/jumpgate-node/369/jwt.hex"}`)
}

func TestRewriteUnitsSwitchesNodeJSONAndTheMarker(t *testing.T) {
	ex := newScript().on("test -e '"+catalog.SnapshotMarkerPath(legacyWire())+"'", fail1)
	env := mnEnv(t, ex, legacyWire())
	env.Results["move-data"] = movedResult()
	env.Results["preflight"] = preflightResult(true, true)
	if _, err := step(t, "rewrite-units").Run(context.Background(), env); err != nil {
		t.Fatal(err)
	}
	unit := string(ex.files["/etc/systemd/system/jumpgate-node-exec.service"])
	if !strings.Contains(unit, "User=jumpgate-node\n") || !strings.Contains(unit, "ReadWritePaths=/var/lib/jumpgate-node/369\n") {
		t.Fatalf("unit:\n%s", unit)
	}
	if !ex.ran("rm -f '/etc/systemd/system/valve-node-app-exec.service' '/etc/systemd/system/valve-node-app-beacon.service'") ||
		!ex.ran("systemctl enable jumpgate-node-exec.service jumpgate-node-beacon.service") {
		t.Fatalf("calls %v", ex.calls)
	}
	if w := readWire(t, env); w.Names != catalog.NamesCurrent || w.DataDir != "/var/lib/jumpgate-node/369" || w.JWTPath != "/var/lib/jumpgate-node/369/jwt.hex" {
		t.Fatalf("node.json %+v", w)
	}
	mk, ok, err := ReadNamesMarker(env.NamesPath)
	if err != nil || !ok || mk.Names != catalog.NamesCurrent || mk.DataRootFrom != "/var/lib/valve-node-app" || mk.DataRootTo != "/var/lib/jumpgate-node" {
		t.Fatalf("marker %+v %v %v", mk, ok, err)
	}
	// No snapshot marker: none is invented.
	if ex.ran("mktemp") {
		t.Fatalf("a marker was written: %v", ex.calls)
	}
	// A second run (after a lost attempt) finds node.json already switched
	// and leaves it so.
	if _, err := step(t, "rewrite-units").Run(context.Background(), env); err != nil {
		t.Fatal(err)
	}
	if w := readWire(t, env); w.Names != catalog.NamesCurrent || w.DataDir != "/var/lib/jumpgate-node/369" {
		t.Fatalf("node.json after a rerun %+v", w)
	}
}

// Delta 2: the snapshot marker names the data dir it vouches for. When the
// data moves, the marker must follow it, or setup's next Verify refuses the
// box ("the snapshot marker names <old>, not <new>").
func TestRewriteUnitsMovesTheSnapshotMarkerWithTheData(t *testing.T) {
	marker := "cat '" + catalog.SnapshotMarkerPath(legacyWire()) + "'"
	ex := newScript().on(marker, stdout("/var/lib/valve-node-app/369\n"))
	env := mnEnv(t, ex, legacyWire())
	env.Results["move-data"] = movedResult()
	env.Results["preflight"] = preflightResult(true, true)
	if _, err := step(t, "rewrite-units").Run(context.Background(), env); err != nil {
		t.Fatal(err)
	}
	want := catalog.SnapshotMarkerCommand(migratedWire())
	if !ex.ran(want) || !strings.Contains(want, "'/var/lib/jumpgate-node/369' >") {
		t.Fatalf("the snapshot marker was not moved: %v", ex.calls)
	}
	// Already moved (a rerun), or vouching for some other dir: left alone.
	for _, named := range []string{"/var/lib/jumpgate-node/369\n", "/mnt/old-disk/reth\n"} {
		ex := newScript().on(marker, stdout(named))
		env := mnEnv(t, ex, legacyWire())
		env.Results["move-data"] = movedResult()
		env.Results["preflight"] = preflightResult(true, true)
		if _, err := step(t, "rewrite-units").Run(context.Background(), env); err != nil {
			t.Fatal(err)
		}
		if ex.ran("mktemp") {
			t.Errorf("marker naming %q was rewritten", strings.TrimSpace(named))
		}
	}
}

// Kept in place (a mount, or a custom dir), the data dir does not change, so
// the marker already names the final dir.
func TestRewriteUnitsKeepsTheSnapshotMarkerWhenTheDataStays(t *testing.T) {
	ex := newScript().on("cat '"+catalog.SnapshotMarkerPath(legacyWire())+"'", stdout("/var/lib/valve-node-app/369\n"))
	env := mnEnv(t, ex, legacyWire())
	env.Results["move-data"] = json.RawMessage(`{"moved":false,"dataDir":"/var/lib/valve-node-app/369","jwtPath":"/var/lib/valve-node-app/369/jwt.hex"}`)
	env.Results["preflight"] = preflightResult(true, true)
	if _, err := step(t, "rewrite-units").Run(context.Background(), env); err != nil {
		t.Fatal(err)
	}
	if ex.ran("mktemp") {
		t.Fatalf("calls %v", ex.calls)
	}
	w := readWire(t, env)
	if w.Names != catalog.NamesCurrent || w.DataDir != "/var/lib/valve-node-app/369" {
		t.Fatalf("node.json %+v", w)
	}
	unit := string(ex.files["/etc/systemd/system/jumpgate-node-beacon.service"])
	if !strings.Contains(unit, "User=jumpgate-node\n") || !strings.Contains(unit, "ReadWritePaths=/var/lib/valve-node-app/369\n") {
		t.Fatalf("unit:\n%s", unit)
	}
	mk, ok, _ := ReadNamesMarker(env.NamesPath)
	if !ok || mk.DataRootFrom != "" {
		t.Fatalf("marker %+v", mk)
	}
}

func TestRewriteUnitsRefusesAMoveThatDoesNotMatchNodeJSON(t *testing.T) {
	ex := newScript()
	env := mnEnv(t, ex, legacyWire())
	env.Results["move-data"] = json.RawMessage(`{"moved":true,"dataDir":"/var/lib/jumpgate/369","jwtPath":"/var/lib/jumpgate/369/jwt.hex"}`)
	env.Results["preflight"] = preflightResult(true, true)
	if _, err := step(t, "rewrite-units").Run(context.Background(), env); err == nil || len(ex.files) != 0 {
		t.Fatalf("rewrite-units = %v, files %v", err, ex.files)
	}
}

// Spec: chown only if ownership is by name rather than uid. usermod keeps the
// uid, so a normal box needs no walk at all.
func TestOwnershipChownsOnlyWhenTheNameDoesNotFollowTheUID(t *testing.T) {
	w := migratedWire()
	ex := newScript().on("stat -c %U", stdout("jumpgate-node\n"))
	if _, err := step(t, "ownership").Run(context.Background(), mnEnv(t, ex, w)); err != nil || ex.ran("chown") || ex.ran("find ") {
		t.Fatalf("calls %v, %v", ex.calls, err)
	}
	ex = newScript().on("stat -c %U", stdout("UNKNOWN\n"))
	if _, err := step(t, "ownership").Run(context.Background(), mnEnv(t, ex, w)); err != nil ||
		!ex.ran("chown -R jumpgate-node:jumpgate-node '/var/lib/jumpgate-node/369' '/var/lib/jumpgate-node/369/jwt.hex'") || !ex.ran(catalog.OwnerMarkerCommand(w)) {
		t.Fatalf("calls %v, %v", ex.calls, err)
	}
	if ex.index(catalog.OwnerMarkerCommand(w)) < ex.index("chown -R") {
		t.Fatal("the owner marker was written before the walk it vouches for")
	}
}

// Delta 2: the owner marker vouched that chown -R finished for the legacy
// account; usermod kept its uid, so it now holds for the new name, and must
// say so or setup's next Verify walks the whole tree again.
func TestOwnershipRenamesTheOwnerMarker(t *testing.T) {
	w := migratedWire()
	cat := "cat '" + catalog.OwnerMarkerPath(w) + "'"
	ex := newScript().on("stat -c %U", stdout("jumpgate-node\n")).on(cat, stdout("valve-node-app\n"))
	if _, err := step(t, "ownership").Run(context.Background(), mnEnv(t, ex, w)); err != nil || !ex.ran(catalog.OwnerMarkerCommand(w)) {
		t.Fatalf("calls %v, %v", ex.calls, err)
	}
	if !strings.Contains(catalog.OwnerMarkerCommand(w), "'jumpgate-node' >") {
		t.Fatalf("owner marker command %s", catalog.OwnerMarkerCommand(w))
	}
	// Already the new name, absent, or naming something else: nothing to
	// vouch for, so nothing is written.
	for name, ex := range map[string]*scriptExec{
		"current": newScript().on("stat -c %U", stdout("jumpgate-node\n")).on(cat, stdout("jumpgate-node\n")),
		"absent":  newScript().on("stat -c %U", stdout("jumpgate-node\n")).on("test -e '"+catalog.OwnerMarkerPath(w)+"'", fail1),
		"other":   newScript().on("stat -c %U", stdout("jumpgate-node\n")).on(cat, stdout("root\n")),
	} {
		if _, err := step(t, "ownership").Run(context.Background(), mnEnv(t, ex, w)); err != nil || ex.ran("mktemp") {
			t.Errorf("%s: calls %v, %v", name, ex.calls, err)
		}
	}
}

func TestOwnershipNeedsTheSwitchedNodeJSON(t *testing.T) {
	ex := newScript().on("stat -c %U", stdout("UNKNOWN\n"))
	if _, err := step(t, "ownership").Run(context.Background(), mnEnv(t, ex, legacyWire())); err == nil || ex.ran("chown") {
		t.Fatalf("ownership on a legacy node.json: %v, %v", err, ex.calls)
	}
}

func TestStartUnitsRestoresOnlyWhatRan(t *testing.T) {
	w := legacyWire()
	w.Names = catalog.NamesCurrent
	ex := newScript().on("systemctl is-active jumpgate-node-exec.service", stdout("active\n"))
	env := mnEnv(t, ex, w)
	env.Results["preflight"] = json.RawMessage(`{"wasActive":{"exec":true,"beacon":false}}`)
	if _, err := step(t, "start-units").Run(context.Background(), env); err != nil {
		t.Fatal(err)
	}
	if !ex.ran("systemctl start jumpgate-node-exec.service") || ex.ran("systemctl start jumpgate-node-beacon.service") {
		t.Fatalf("calls %v", ex.calls)
	}
}

func TestVerifyReportsTheResult(t *testing.T) {
	ex := migratedBox()
	env := mnEnv(t, ex, migratedWire())
	env.Results["preflight"] = preflightResult(true, true)
	o, err := step(t, "verify").Run(context.Background(), env)
	if err != nil {
		t.Fatal(err)
	}
	var r MigrateNamesResult
	if err := json.Unmarshal(o.Result, &r); err != nil || r != (MigrateNamesResult{Names: catalog.NamesCurrent, DataDir: "/var/lib/jumpgate-node/369", JWTPath: "/var/lib/jumpgate-node/369/jwt.hex"}) {
		t.Fatalf("result %+v, %v", r, err)
	}
}

func TestVerifyNamesWhatIsLeft(t *testing.T) {
	ex := migratedBox().on("getent group valve-node-app", stdout("valve-node-app:x:998:\n")).
		on("cat '"+catalog.SnapshotMarkerPath(migratedWire())+"'", stdout("/var/lib/valve-node-app/369\n")).
		on("systemctl is-enabled jumpgate-node-beacon.service", fail1)
	env := mnEnv(t, ex, migratedWire())
	env.Results["preflight"] = preflightResult(true, true)
	_, err := step(t, "verify").Run(context.Background(), env)
	for _, want := range []string{"group valve-node-app", "snapshot marker", "jumpgate-node-beacon.service is not enabled"} {
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("verify = %v, want %q", err, want)
		}
	}
}

func TestNamesMarkerAppliesToAStaleNodeJSON(t *testing.T) {
	w := legacyWire()
	NamesMarker{Names: catalog.NamesCurrent, DataRootFrom: "/var/lib/valve-node-app", DataRootTo: "/var/lib/jumpgate-node"}.Apply(&w)
	if w.Names != catalog.NamesCurrent || w.DataDir != "/var/lib/jumpgate-node/369" || w.JWTPath != "/var/lib/jumpgate-node/369/jwt.hex" {
		t.Fatalf("applied %+v", w)
	}
}

// Steps read node.json through the marker too (J10): a re-pair that wrote the
// controller's stale wire after a migration must not restart it.
func TestStepsReadNodeJSONThroughTheNamesMarker(t *testing.T) {
	ex := migratedBox()
	env := mnEnv(t, ex, legacyWire())
	if err := writeNamesMarker(env.NamesPath, NamesMarker{Names: catalog.NamesCurrent, DataRootFrom: "/var/lib/valve-node-app", DataRootTo: "/var/lib/jumpgate-node"}); err != nil {
		t.Fatal(err)
	}
	o, err := step(t, "preflight").Run(context.Background(), env)
	var pre migratePreflight
	_ = json.Unmarshal(o.Result, &pre)
	if err != nil || !pre.AlreadyDone {
		t.Fatalf("preflight %+v, %v", pre, err)
	}
}

func TestReadNamesMarkerRefusesWhatItCannotApply(t *testing.T) {
	dir := t.TempDir()
	if _, ok, err := ReadNamesMarker(filepath.Join(dir, "none.json")); ok || err != nil {
		t.Fatalf("missing marker: %v %v", ok, err)
	}
	for _, body := range []string{
		`{"names":"valve-node-app"}`,
		`{"names":"jumpgate-node","dataRootFrom":"/var/lib/valve-node-app","dataRootTo":"/var/lib/jumpgate"}`,
		`{"names":"jumpgate-node","dataRootFrom":"/","dataRootTo":"/var/lib/jumpgate-node"}`,
		`{"names":"jumpgate-node","dataRootTo":"/var/lib/jumpgate-node"}`,
		`not json`,
	} {
		p := filepath.Join(dir, "names.json")
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, ok, err := ReadNamesMarker(p); ok || err == nil {
			t.Errorf("accepted %s", body)
		}
	}
}

func preflightResult(execEnabled, beaconEnabled bool) json.RawMessage {
	b, _ := json.Marshal(migratePreflight{WasActive: map[string]bool{},
		WasEnabled: map[string]bool{"exec": execEnabled, "beacon": beaconEnabled}})
	return b
}

func rootMove() NamesMarker {
	return NamesMarker{Names: catalog.NamesCurrent, DataRootFrom: catalog.LegacyNames.DataRoot, DataRootTo: catalog.CurrentNames.DataRoot}
}

// I1: names.json is what keeps a stale re-pair from pointing the node back at
// the old root. A later job must carry the move it records, never drop it.
func TestARecordedRootMoveSurvivesMoveDataAndRewriteUnits(t *testing.T) {
	ex := newScript()
	env := mnEnv(t, ex, legacyWire()) // a stale re-pair: node.json at the old root
	if err := writeNamesMarker(env.NamesPath, rootMove()); err != nil {
		t.Fatal(err)
	}
	o, err := step(t, "move-data").Run(context.Background(), env)
	var mv migrateMove
	_ = json.Unmarshal(o.Result, &mv)
	if err != nil || !mv.Moved || mv.DataDir != "/var/lib/jumpgate-node/369" || ex.ran("mv ") {
		t.Fatalf("move-data = %+v, %v, calls %v", mv, err, ex.calls)
	}
	env.Results["move-data"] = o.Result
	env.Results["preflight"] = preflightResult(true, true)
	if _, err := step(t, "rewrite-units").Run(context.Background(), env); err != nil {
		t.Fatal(err)
	}
	if mk, ok, err := ReadNamesMarker(env.NamesPath); err != nil || !ok || mk != rootMove() {
		t.Fatalf("names.json %+v %v %v", mk, ok, err)
	}
}

// I1, end to end: job A moves the root and is aborted at ownership (its owner
// marker still names the old account); job B finishes the migration and
// names.json still records the move.
func TestASecondJobAfterAnAbortKeepsTheRootMove(t *testing.T) {
	env := mnEnv(t, nil, legacyWire())
	runJob := func(steps []string, exFor func(id string) *scriptExec) {
		t.Helper()
		env.Results, env.Answer = map[string]json.RawMessage{}, nil
		for _, id := range steps {
			env.Exec = exFor(id)
			o, err := step(t, id).Run(context.Background(), env)
			if err != nil || o.Ask != nil {
				t.Fatalf("%s: %+v, %v", id, o.Ask, err)
			}
			env.Results[id] = o.Result
		}
	}
	runJob([]string{"preflight", "stop-units", "rename-account", "move-data", "rewrite-units"}, func(id string) *scriptExec {
		switch id {
		case "stop-units":
			return legacyBox().on("pgrep -a -u", fail1)
		case "rename-account":
			return legacyBox().on("id -u jumpgate-node", fail1, idOK("998")).on("getent group jumpgate-node", fail1, stdout("jumpgate-node:x:998:\n"))
		case "move-data":
			return legacyBox().on("test -d '/var/lib/valve-node-app'", ok0).on("findmnt -rn -o TARGET", stdout("/\n"))
		}
		return legacyBox()
	})
	if mk, _, _ := ReadNamesMarker(env.NamesPath); mk != rootMove() {
		t.Fatalf("job A names.json %+v", mk)
	}
	// Job B: the box is renamed and moved; only the owner marker is left.
	ownerCat := "cat '" + catalog.OwnerMarkerPath(migratedWire()) + "'"
	box := func() *scriptExec {
		return migratedBox().on(ownerCat, stdout("valve-node-app\n")).on("pgrep -a -u", fail1).
			on("stat -c %U", stdout("jumpgate-node\n")).on("test -d '/var/lib/valve-node-app'", fail1).on("test -e '/var/lib/jumpgate-node'", ok0)
	}
	var owner *scriptExec
	runJob([]string{"preflight", "stop-units", "rename-account", "move-data", "rewrite-units", "ownership"}, func(id string) *scriptExec {
		ex := box()
		if id == "ownership" {
			owner = ex
		}
		return ex
	})
	if !owner.ran(catalog.OwnerMarkerCommand(migratedWire())) {
		t.Fatalf("job B did not finish the owner marker: %v", owner.calls)
	}
	if mk, ok, err := ReadNamesMarker(env.NamesPath); err != nil || !ok || mk != rootMove() {
		t.Fatalf("job B names.json %+v %v %v", mk, ok, err)
	}
	if w := readWire(t, env); w.DataDir != "/var/lib/jumpgate-node/369" || w.Names != catalog.NamesCurrent {
		t.Fatalf("node.json %+v", w)
	}
}

// M1: with no group to rename and none to use, the account is left alone.
func TestNoGroupMeansNothingIsRenamed(t *testing.T) {
	ex := legacyBox().on("getent group valve-node-app", fail1)
	if _, err := step(t, "preflight").Run(context.Background(), mnEnv(t, ex, legacyWire())); err == nil || !strings.Contains(err.Error(), "group") {
		t.Fatalf("preflight = %v", err)
	}
	ex = legacyBox().on("getent group valve-node-app", fail1)
	if _, err := step(t, "rename-account").Run(context.Background(), mnEnv(t, ex, legacyWire())); err == nil || !strings.Contains(err.Error(), "group") {
		t.Fatalf("rename-account = %v", err)
	}
	noCommand(t, ex, "usermod", "groupmod")
}

// M2: a reboot between stop-units and rename-account starts the legacy units
// again. That is a question, answered by stopping them again, not a failure.
func TestRenameAccountAsksWhenTheNodeRunsAgain(t *testing.T) {
	ex := legacyBox().on("id -u jumpgate-node", fail1, fail1, idOK("998")).on("getent group jumpgate-node", fail1, fail1, stdout("jumpgate-node:x:998:\n")).
		on("systemctl is-active valve-node-app-exec.service", stdout("active\n"), stdout("inactive\n")).
		on("pgrep -a -u valve-node-app", stdout("4242 reth node\n"), fail1)
	env := mnEnv(t, ex, legacyWire())
	o, err := step(t, "rename-account").Run(context.Background(), env)
	if err != nil || o.Ask == nil || o.Ask.Key != "busy-user" || !strings.Contains(o.Ask.Prompt, "valve-node-app-exec.service") {
		t.Fatalf("rename-account = %+v, %v", o.Ask, err)
	}
	if ex.ran("usermod") || ex.ran("systemctl stop") {
		t.Fatalf("calls %v", ex.calls)
	}
	env.Answer = &Answer{Key: "busy-user", Option: intent.OptRetry}
	if o, err := step(t, "rename-account").Run(context.Background(), env); err != nil || o.Ask != nil {
		t.Fatalf("after retry: %+v, %v", o.Ask, err)
	}
	if s, u := ex.index("systemctl stop valve-node-app-exec.service"), ex.index("usermod"); s < 0 || u < s {
		t.Fatalf("stop at %d, usermod at %d: %v", s, u, ex.calls)
	}
}

// M3: an operator who disabled a unit keeps it disabled under its new name.
func TestADisabledUnitStaysDisabled(t *testing.T) {
	ex := legacyBox().on("systemctl is-enabled valve-node-app-beacon.service", fail1)
	o, err := step(t, "preflight").Run(context.Background(), mnEnv(t, ex, legacyWire()))
	var pre migratePreflight
	_ = json.Unmarshal(o.Result, &pre)
	if err != nil || !pre.WasEnabled["exec"] || pre.WasEnabled["beacon"] {
		t.Fatalf("preflight %+v, %v", pre, err)
	}
	ex = newScript()
	env := mnEnv(t, ex, legacyWire())
	env.Results["move-data"] = movedResult()
	env.Results["preflight"] = o.Result
	if _, err := step(t, "rewrite-units").Run(context.Background(), env); err != nil {
		t.Fatal(err)
	}
	if !ex.ran("systemctl enable jumpgate-node-exec.service") || ex.ran("enable jumpgate-node-beacon.service") || ex.ran("enable jumpgate-node-exec.service jumpgate-node-beacon") {
		t.Fatalf("calls %v", ex.calls)
	}
	// verify holds the box to the same set: beacon enabled is a problem.
	ex = migratedBox()
	env = mnEnv(t, ex, migratedWire())
	env.Results["preflight"] = o.Result
	if _, err := step(t, "verify").Run(context.Background(), env); err == nil || !strings.Contains(err.Error(), "jumpgate-node-beacon.service is enabled") {
		t.Fatalf("verify = %v", err)
	}
	ex.on("systemctl is-enabled jumpgate-node-beacon.service", fail1)
	if _, err := step(t, "verify").Run(context.Background(), env); err != nil {
		t.Fatalf("verify = %v", err)
	}
}
