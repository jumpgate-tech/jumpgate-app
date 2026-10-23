//go:build e2e

package server

import (
	"context"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/valve-tech/jumpgate/internal/catalog"
	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/executor"
	"github.com/valve-tech/jumpgate/internal/setup"
)

// Driven by scripts/e2e-agent.sh, which starts the systemd+sshd container
// (with stub reth and lighthouse-pulse binaries) and exports
// JUMPGATE_E2E_PORT and JUMPGATE_E2E_ROOT_KEY.
func e2eRoot(t *testing.T) executor.SSHConfig {
	t.Helper()
	port, err := strconv.Atoi(os.Getenv("JUMPGATE_E2E_PORT"))
	key := os.Getenv("JUMPGATE_E2E_ROOT_KEY")
	if err != nil || key == "" {
		t.Skip("JUMPGATE_E2E_PORT/JUMPGATE_E2E_ROOT_KEY not set; run scripts/e2e-agent.sh")
	}
	return executor.SSHConfig{Host: "127.0.0.1", Port: port, User: "root", KeyPath: key, HostKey: ssh.InsecureIgnoreHostKey()}
}

// e2eStubSteps are the plan steps a stub client can pass. Preflight's disk
// floor, the toolchain, the snapshot, the handshake and the modules live
// probe all need a real node, which the container does not run.
var e2eStubSteps = []string{"account", "install-exec", "install-beacon", "wire", "start"}

func e2ePlan(w catalog.WireConfig) ([]setup.Step, error) {
	steps, err := setup.Plan(w)
	if err != nil {
		return nil, err
	}
	return slices.DeleteFunc(steps, func(s setup.Step) bool { return !slices.Contains(e2eStubSteps, s.ID) }), nil
}

// TestE2EModuleToggle turns msgboard on for a 943 reth node over real SSH and
// systemd: the exec unit gains the namespace and restarts, the beacon keeps
// running, and a second identical PUT restarts nothing.
func TestE2EModuleToggle(t *testing.T) {
	root := e2eRoot(t)
	old := planWire
	planWire = e2ePlan
	t.Cleanup(func() { planWire = old })

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	ex, err := executor.NewSSHContext(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	defer ex.Close()

	wire := catalog.WireConfig{ChainID: 943, ExecID: "reth", BeaconID: "lighthouse-pulse", Names: catalog.NamesCurrent,
		DataDir: "/var/lib/jumpgate-node/943", JWTPath: "/var/lib/jumpgate-node/943/jwt.hex"}
	n := wire.NodeNames()
	steps, err := e2ePlan(wire)
	if err != nil {
		t.Fatal(err)
	}
	if err := setup.RunAll(ctx, ex, steps, &setup.State{Wire: wire}); err != nil {
		t.Fatalf("first setup: %v", err)
	}

	show := func(unit string) string {
		t.Helper()
		r, err := ex.Run(ctx, "systemctl show -p ActiveEnterTimestampMonotonic --value "+unit, nil)
		if err != nil || r.ExitCode != 0 {
			t.Fatalf("systemctl show %s: %+v %v", unit, r, err)
		}
		return strings.TrimSpace(r.Stdout)
	}
	execBefore, beaconBefore := show(n.ExecUnit()), show(n.BeaconUnit())
	if execBefore == "0" || beaconBefore == "0" {
		t.Fatalf("units not active after the first setup: exec %s beacon %s", execBefore, beaconBefore)
	}

	a := newAPITestServerWithExecutor(t, func(config.Target) (executor.Executor, error) {
		return executor.NewSSHContext(context.Background(), root)
	})
	moduleTarget(t, a, func(c *config.Config) { w := wire; targetByID(c, "box").Wire = &w })

	res := a.do(t, "PUT", "/api/targets/box/modules/msgboard", map[string]any{"enabled": true})
	if res.StatusCode != http.StatusAccepted {
		t.Fatalf("PUT = %d, want 202", res.StatusCode)
	}
	res.Body.Close()
	waitSetupIdleFor(t, a, "box", 3*time.Minute)

	unit, err := ex.ReadFile(ctx, catalog.UnitPath(n.ExecUnit()))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(unit), "--http.api eth,net,web3,msgboard") {
		t.Fatalf("exec unit lacks the msgboard api list:\n%s", unit)
	}
	execAfter := show(n.ExecUnit())
	if execAfter == execBefore {
		t.Fatal("the exec unit was not restarted")
	}
	if got := show(n.BeaconUnit()); got != beaconBefore {
		t.Fatalf("the beacon restarted: %s -> %s", beaconBefore, got)
	}

	res = a.do(t, "PUT", "/api/targets/box/modules/msgboard", map[string]any{"enabled": true})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("second PUT = %d, want 200", res.StatusCode)
	}
	res.Body.Close()
	if got := show(n.ExecUnit()); got != execAfter {
		t.Fatalf("the second PUT restarted the exec unit: %s -> %s", execAfter, got)
	}
	if got := show(n.BeaconUnit()); got != beaconBefore {
		t.Fatalf("the second PUT restarted the beacon: %s -> %s", beaconBefore, got)
	}
}

// waitSetupIdleFor waits for the target's setup run to end and fails on its error.
func waitSetupIdleFor(t *testing.T, a *apiTestServer, id string, d time.Duration) {
	t.Helper()
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		e := a.srv.reg.get(id)
		e.mu.Lock()
		run := e.setup
		e.mu.Unlock()
		if run != nil && !run.isRunning() {
			run.mu.Lock()
			err, evs := run.err, slices.Clone(run.events)
			run.mu.Unlock()
			if err != nil {
				t.Fatalf("setup run: %v; events %+v", err, evs)
			}
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatal("setup did not finish")
}
