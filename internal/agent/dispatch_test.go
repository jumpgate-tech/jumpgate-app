package agent

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/valve-tech/jumpgate/internal/catalog"
	"github.com/valve-tech/jumpgate/internal/executor"
	"github.com/valve-tech/jumpgate/internal/intent"
	"github.com/valve-tech/jumpgate/internal/monitor"
	"github.com/valve-tech/jumpgate/internal/testutil"
)

// writeNode replaces the rig's node.json with raw.
func (r *rig) writeNode(t *testing.T, raw string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(r.dir, "node.json"), []byte(raw), 0o600); err != nil {
		t.Fatal(err)
	}
}

// node.json's paths reach root commands. One the agent would not act on (a
// system tree as the data dir, names it does not know, a file that does not
// parse) refuses every node kind before anything runs, and agent.info says
// the box is set up wrongly rather than not set up.
func TestANodeConfigTheAgentWillNotActOnRefusesEveryNodeKind(t *testing.T) {
	testutil.RequireUnix(t)
	w := catalog.WireConfig{ChainID: 369, ExecID: "reth", BeaconID: "lighthouse", DataDir: "/var/lib/valve-node-app/369", JWTPath: "/var/lib/valve-node-app/369/jwt.hex"}
	bad := func(edit func(*catalog.WireConfig)) string {
		c := w
		edit(&c)
		b, _ := json.Marshal(c)
		return string(b)
	}
	for name, raw := range map[string]string{
		"system data dir": bad(func(c *catalog.WireConfig) { c.DataDir = "/etc" }),
		"unknown names":   bad(func(c *catalog.WireConfig) { c.Names = "someone-else" }),
		"not json":        "{",
	} {
		t.Run(name, func(t *testing.T) {
			r := newRig(t, true)
			r.writeNode(t, raw)
			for _, k := range []string{intent.KindStatusRead, intent.KindDiskRead, intent.KindLogsRead,
				intent.KindServiceAction} {
				rc, res := r.send(t, k, intent.ServiceActionPayload{Service: "exec", Action: "stop"}, nil)
				if got := rejection(t, rc, res).Code; got != intent.ReasonValidation {
					t.Errorf("%s: code %s, want validation", k, got)
				}
			}
			if len(r.exec.cmds) != 0 {
				t.Fatalf("ran commands on a refused config: %v", r.exec.cmds)
			}
			_, res := r.send(t, intent.KindAgentInfo, struct{}{}, nil)
			var info intent.AgentInfo
			if json.Unmarshal(res, &info) != nil || info.SetUp || info.SetUpError == nil || info.SetUpError.Code != intent.ReasonValidation {
				t.Fatalf("agent.info %s", res)
			}
		})
	}
}

// A payload that does not decode into the kind's shape is refused as
// invalid, and nothing runs.
func TestAPayloadOfTheWrongShapeIsRefused(t *testing.T) {
	testutil.RequireUnix(t)
	r := newRig(t, true)
	for _, k := range []string{intent.KindEndpointsRead, intent.KindFirewallRead, intent.KindLogsRead, intent.KindLogsSince, intent.KindServiceAction} {
		rc, res := r.send(t, k, json.RawMessage(`[1,2]`), nil)
		if got := rejection(t, rc, res).Code; got != intent.ReasonInvalidPayload {
			t.Errorf("%s: code %s, want invalid_payload", k, got)
		}
	}
	if len(r.exec.cmds) != 0 {
		t.Fatalf("ran %v", r.exec.cmds)
	}
}

// logs.read without a count reads the default number of lines per unit.
func TestLogsReadDefaultsN(t *testing.T) {
	testutil.RequireUnix(t)
	r := newRig(t, true)
	r.exec.res = executor.Result{Stdout: "a line\n\n"}
	rc, res := r.send(t, intent.KindLogsRead, intent.LogsReadPayload{}, nil)
	if rc.Status != intent.StatusOK {
		t.Fatalf("status %d: %s", rc.Status, res)
	}
	for _, c := range r.exec.cmds {
		if !strings.Contains(c, " -n "+itoa(intent.LogsDefaultN)+" ") {
			t.Fatalf("command %q does not read the default count", c)
		}
	}
	if strings.Count(string(res), `"a line"`) != len(r.exec.cmds) {
		t.Fatalf("result %s: want one hit per unit, blank lines dropped", res)
	}
}

func itoa(n int) string { b, _ := json.Marshal(n); return string(b) }

// failExec fails every command, as a box whose shell cannot run does.
type failExec struct{ fakeExec }

func (f *failExec) Run(_ context.Context, cmd string, _ *executor.RunOpts) (executor.Result, error) {
	f.cmds = append(f.cmds, cmd)
	return executor.Result{}, errors.New("exec: cannot run")
}

// A read the box cannot perform is a signed failure receipt naming the
// error, never an OK with an empty result a controller would believe.
// (endpoints.read is a reachability probe: a failure there reads as
// unreachable, which is its answer.)
func TestAReadThatCannotRunIsASignedFailure(t *testing.T) {
	testutil.RequireUnix(t)
	r := newRig(t, true)
	fe := &failExec{}
	r.a.cfg.Exec = fe
	for k, p := range map[string]any{
		intent.KindDiskRead:     struct{}{},
		intent.KindFirewallRead: FirewallReadPayload{},
		intent.KindLogsRead:     intent.LogsReadPayload{},
		intent.KindLogsSince:    intent.LogsSincePayload{},
	} {
		rc, res := r.send(t, k, p, nil)
		var f intent.Failure
		if rc.Status != intent.StatusFailed || json.Unmarshal(res, &f) != nil || !strings.Contains(f.Message, "cannot run") {
			t.Errorf("%s: status %d, result %s", k, rc.Status, res)
		}
	}
}

// disk.read answers with the data dir's usage and the free space under it.
func TestDiskReadReportsUsageAndFreeSpace(t *testing.T) {
	testutil.RequireUnix(t)
	r := newRig(t, true)
	r.exec.res = executor.Result{Stdout: "4096\n"} // du and df both read one number
	rc, res := r.send(t, intent.KindDiskRead, struct{}{}, nil)
	if rc.Status != intent.StatusOK {
		t.Fatalf("status %d: %s", rc.Status, res)
	}
	var d DiskResult
	if err := json.Unmarshal(res, &d); err != nil || d.FreeBytes != 4096 {
		t.Fatalf("result %s: %v", res, err)
	}
	ran := strings.Join(r.exec.cmds, "\n")
	if !strings.Contains(ran, "/var/lib/valve-node-app/369") {
		t.Fatalf("commands %v", r.exec.cmds)
	}
}

// modules.read needs no node.json: a bootnode-only box still reports its
// modules through its gate.
func TestModulesReadNeedsNoNode(t *testing.T) {
	testutil.RequireUnix(t)
	r := newRig(t, false)
	r.exec.res = executor.Result{}
	rc, res := r.send(t, intent.KindModulesRead, struct{}{}, nil)
	if rc.Status != intent.StatusOK {
		t.Fatalf("status %d: %s", rc.Status, res)
	}
	var got []monitor.ModuleStatus
	if err := json.Unmarshal(res, &got); err != nil || got == nil || len(got) != 0 {
		t.Fatalf("result %s: %v", res, err)
	}
	if len(r.exec.cmds) == 0 || !strings.Contains(r.exec.cmds[0], "/var/lib/jumpgate/modules/") {
		t.Fatalf("commands %v", r.exec.cmds)
	}
}
