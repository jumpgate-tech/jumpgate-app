package setup

// Every command the gateway plan issues, broken one at a time.
//
// The assertion is always the same and it is deliberately strict: the original
// cause must survive to the top via errors.Is. A step that catches a dropped
// connection and returns its own tidy error reads identically to a step that
// succeeded, once the caller only looks at "did this fail" — and this repo's
// worst bugs have all been things that reported success while broken.
//
// Two commands are exempt, both by documented design, and both get their own
// test below rather than being quietly left out of the table.

import (
	"context"
	"errors"
	"io/fs"
	"strings"
	"testing"
	"time"

	"github.com/valve-tech/valve-node-app/internal/catalog"
	"github.com/valve-tech/valve-node-app/internal/executor"
)

// runPlan runs every step in order and returns the first error, which is what
// a real setup run does. Breaking one command anywhere in the plan therefore
// surfaces wherever that command happens to be issued.
//
// st is shared across every step, as it is under RunAll: steps read the wire
// config from it, and some write findings a later step reads back.
func runPlan(t *testing.T, steps []Step, e executor.Executor, st *State) error {
	t.Helper()
	for _, s := range steps {
		if s.Run != nil {
			if err := s.Run(context.Background(), e, st); err != nil {
				return err
			}
		}
		if s.Verify != nil {
			if err := s.Verify(context.Background(), e, st); err != nil {
				return err
			}
		}
	}
	return nil
}

// collectLines drains a closed event channel down to its progress lines.
func collectLines(events <-chan Event) []string {
	var out []string
	for ev := range events {
		if ev.Line != "" {
			out = append(out, ev.Line)
		}
	}
	return out
}

// shrinkCaddyRootWait keeps the internal-CA export's retry loop out of real
// time, the same way shrinkGatewayWait does for the readiness poll.
func shrinkCaddyRootWait(t *testing.T) {
	t.Helper()
	oldTimeout, oldInterval := caddyRootTimeout, caddyRootPollInterval
	caddyRootTimeout = 5 * time.Millisecond
	caddyRootPollInterval = time.Millisecond
	t.Cleanup(func() { caddyRootTimeout, caddyRootPollInterval = oldTimeout, oldInterval })
}

func TestGatewayPlan_DockerCommandFailuresAllSurface(t *testing.T) {
	shrinkGatewayWait(t)
	shrinkCaddyRootWait(t)
	boom := errors.New("ssh: connection lost")

	// Each entry is a substring of exactly one command in the docker plan.
	breaks := []struct {
		name    string
		cmd     string
		wantSay string
	}{
		{name: "probing for docker", cmd: "command -v docker", wantSay: "docker"},
		{name: "reading docker info", cmd: "docker info --format", wantSay: "docker"},
		{name: "probing the HTTPS port", cmd: "[:.]8443", wantSay: "8443"},
		{name: "probing the metrics port", cmd: "[:.]4001", wantSay: "4001"},
		{name: "resolving $HOME", cmd: `printf '%s\n' "$HOME"`, wantSay: "home"},
		{name: "inspecting the image", cmd: "docker image inspect", wantSay: "image"},
		{name: "building the image", cmd: "docker 'build'", wantSay: "build"},
		{name: "inspecting the network", cmd: "docker 'network' 'inspect'", wantSay: "network"},
		{name: "removing the old eRPC container", cmd: "docker 'rm' '-f' 'valve-node-app-erpc'", wantSay: "run"},
		{name: "running eRPC", cmd: "'--name' 'valve-node-app-erpc'", wantSay: "run"},
		{name: "removing the old Caddy container", cmd: "docker 'rm' '-f' 'valve-node-app-caddy'", wantSay: "run"},
		{name: "running Caddy", cmd: "'--name' 'valve-node-app-caddy'", wantSay: "run"},
		{name: "asking the gateway for its chain id", cmd: "eth_chainId", wantSay: "gateway"},
	}

	for _, tc := range breaks {
		t.Run(tc.name, func(t *testing.T) {
			e := caddyReady()
			e.errOn(tc.cmd, boom)

			err := runPlan(t, mustPlanGateway(t, frontedGateway(), BackendDocker), e, &State{})
			if err == nil {
				t.Fatalf("breaking %q provisioned a gateway successfully", tc.cmd)
			}
			if !errors.Is(err, boom) {
				t.Errorf("the transport cause did not reach the top: %v", err)
			}
			if !strings.Contains(strings.ToLower(err.Error()), tc.wantSay) {
				t.Errorf("error does not say what it was doing (want %q): %v", tc.wantSay, err)
			}
		})
	}
}

func TestGatewayPlan_SystemdCommandFailuresAllSurface(t *testing.T) {
	shrinkGatewayWait(t)
	boom := errors.New("ssh: connection lost")

	breaks := []struct {
		name    string
		cmd     string
		wantSay string
	}{
		{name: "reading the OS", cmd: "uname", wantSay: "uname"},
		{name: "reading the uid", cmd: "id -u", wantSay: "id -u"},
		{name: "probing the RPC port", cmd: "[:.]4100", wantSay: "4100"},
		{name: "grouping the config file", cmd: "chgrp", wantSay: "config"},
		{name: "driving systemctl", cmd: "systemctl daemon-reload", wantSay: "systemctl"},
		{name: "asking the gateway for its chain id", cmd: "eth_chainId", wantSay: "gateway"},
	}

	for _, tc := range breaks {
		t.Run(tc.name, func(t *testing.T) {
			e := systemdReady()
			e.errOn(tc.cmd, boom)

			err := runPlan(t, mustPlanGateway(t, testGateway(), BackendSystemd), e, &State{})
			if err == nil {
				t.Fatalf("breaking %q provisioned a gateway successfully", tc.cmd)
			}
			if !errors.Is(err, boom) {
				t.Errorf("the transport cause did not reach the top: %v", err)
			}
			if !strings.Contains(strings.ToLower(err.Error()), tc.wantSay) {
				t.Errorf("error does not say what it was doing (want %q): %v", tc.wantSay, err)
			}
		})
	}
}

// ---------------------------------------------------------------------
// unwritable files
// ---------------------------------------------------------------------

// readOnlyTarget fails WriteFile for any path containing `deny`, which is what
// a full disk or a directory the user cannot write looks like from here.
type readOnlyTarget struct {
	*fakeExecutor
	deny string
	err  error
}

func (r readOnlyTarget) WriteFile(ctx context.Context, path string, content []byte, mode fs.FileMode) error {
	if strings.Contains(path, r.deny) {
		return r.err
	}
	return r.fakeExecutor.WriteFile(ctx, path, content, mode)
}

func TestGatewayPlan_UnwritableFilesSurface(t *testing.T) {
	shrinkGatewayWait(t)
	shrinkCaddyRootWait(t)
	full := errors.New("write /dev/sda1: no space left on device")

	tests := []struct {
		name    string
		deny    string
		backend string
		gw      func() catalog.GatewayConfig
		base    func() *fakeExecutor
		wantSay string
	}{
		{
			name: "erpc.yaml", deny: "erpc.yaml", backend: BackendDocker,
			gw: frontedGateway, base: caddyReady, wantSay: "erpc.yaml",
		},
		{
			name: "the Caddyfile", deny: "Caddyfile", backend: BackendDocker,
			gw: frontedGateway, base: caddyReady, wantSay: "caddyfile",
		},
		{
			name: "the systemd unit", deny: ".service", backend: BackendSystemd,
			gw: testGateway, base: systemdReady, wantSay: "service",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e := readOnlyTarget{fakeExecutor: tc.base(), deny: tc.deny, err: full}

			err := runPlan(t, mustPlanGateway(t, tc.gw(), tc.backend), e, &State{})
			if err == nil {
				t.Fatalf("a gateway whose %s could not be written provisioned successfully", tc.name)
			}
			if !errors.Is(err, full) {
				t.Errorf("the write failure did not reach the top: %v", err)
			}
			if !strings.Contains(strings.ToLower(err.Error()), tc.wantSay) {
				t.Errorf("error does not name the file (want %q): %v", tc.wantSay, err)
			}
		})
	}
}

// ---------------------------------------------------------------------
// the two documented exceptions
// ---------------------------------------------------------------------

// gatewayHoldsPort swallows its failure ON PURPOSE: any failure reading
// "is our own gateway already on this port" is reported as "no", and the
// caller then runs the port check, which produces a precise, evidence-carrying
// error rather than this function inventing one.
func TestGatewayPlan_AnUnreadableContainerStateFallsThroughToThePortCheck(t *testing.T) {
	shrinkGatewayWait(t)
	shrinkCaddyRootWait(t)

	e := caddyReady()
	e.errOn("docker 'inspect' '-f' '{{.State.Running}}'", errors.New("no such container"))

	if err := runPlan(t, mustPlanGateway(t, frontedGateway(), BackendDocker), e, &State{}); err != nil {
		t.Fatalf("an unreadable container state failed the plan instead of falling through: %v", err)
	}
	// It must actually have fallen through to the port probe, not skipped it.
	var probed bool
	for _, c := range e.callLog() {
		if strings.Contains(c, "[:.]8443") {
			probed = true
		}
	}
	if !probed {
		t.Error("the port check never ran, so nothing checked the port at all")
	}
}

// Two readings are best-effort by design, and a provisioning run must survive
// losing either. The architecture probe is one of TWO readings of the same
// fact — `docker info` is the other — so a target without a usable `uname`
// still deserves the engine's answer rather than a failed run. The version
// banner is informational; only the PRESENCE probe decides whether this
// target can host containers at all.
func TestGatewayPlan_BestEffortProbesDoNotFailTheRun(t *testing.T) {
	shrinkGatewayWait(t)
	shrinkCaddyRootWait(t)

	for _, tc := range []struct{ name, cmd string }{
		{"the architecture probe", "command -p uname -m"},
		{"the version banner", "docker --version"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := caddyReady()
			e.errOn(tc.cmd, errors.New("command not found"))

			if err := runPlan(t, mustPlanGateway(t, frontedGateway(), BackendDocker), e, &State{}); err != nil {
				t.Fatalf("losing %s failed the whole run: %v", tc.name, err)
			}
		})
	}
}

// Losing the arch probe must still produce a --platform flag. Omitting it does
// not hand the choice to the engine's manifest resolution — it hands it to
// DOCKER_DEFAULT_PLATFORM, which is how QEMU-emulated containers that report
// State: running and answer nothing get created.
func TestGatewayPlan_APlatformIsStillChosenWithoutUname(t *testing.T) {
	shrinkGatewayWait(t)
	shrinkCaddyRootWait(t)

	e := caddyReady()
	e.errOn("command -p uname -m", errors.New("command not found"))

	if err := runPlan(t, mustPlanGateway(t, frontedGateway(), BackendDocker), e, &State{}); err != nil {
		t.Fatalf("runPlan: %v", err)
	}

	run := lastCallWithPrefix(e, "docker 'run'")
	if run == "" {
		t.Fatal("no container was run")
	}
	if !strings.Contains(run, "'--platform'") {
		t.Errorf("docker run carries no --platform, so DOCKER_DEFAULT_PLATFORM decides: %s", run)
	}
}

// exportRoot reports failure as a LINE, never as an error. The gateway serves
// HTTPS either way, and refusing to finish provisioning over a file that only
// silences a browser warning would be the worse trade.
func TestGatewayRun_AnUnreadableCARootWarnsWithoutFailingTheRun(t *testing.T) {
	shrinkGatewayWait(t)
	shrinkCaddyRootWait(t)

	e := dockerReady() // no `docker exec ... cat` script, so the root never appears
	steps := mustPlanGateway(t, frontedGateway(), BackendDocker)

	events := make(chan Event, 256)
	st := &State{Events: events}

	if err := stepByID(t, steps, "config").Run(context.Background(), e, st); err != nil {
		t.Fatalf("config: %v", err)
	}
	if err := stepByID(t, steps, "run").Run(context.Background(), e, st); err != nil {
		t.Fatalf("run failed over a CA root that is only needed to silence a browser warning: %v", err)
	}
	close(events)

	lines := collectLines(events)
	var warned bool
	for _, l := range lines {
		if strings.Contains(l, "internal CA root could not be read") {
			warned = true
		}
	}
	if !warned {
		t.Fatalf("the root was not exported and nothing said so; lines were %#v", lines)
	}
}

// The success side of the same path: the root lands on the host, and the line
// tells the operator what to do with it.
func TestGatewayRun_AnExportedCARootSaysWhereItWent(t *testing.T) {
	shrinkGatewayWait(t)
	shrinkCaddyRootWait(t)

	e := caddyReady()
	steps := mustPlanGateway(t, frontedGateway(), BackendDocker)

	events := make(chan Event, 256)
	st := &State{Events: events}

	if err := stepByID(t, steps, "config").Run(context.Background(), e, st); err != nil {
		t.Fatalf("config: %v", err)
	}
	if err := stepByID(t, steps, "run").Run(context.Background(), e, st); err != nil {
		t.Fatalf("run: %v", err)
	}
	close(events)

	lines := collectLines(events)
	var told bool
	for _, l := range lines {
		if strings.Contains(l, "caddy-root.crt") && strings.Contains(l, "trust store") {
			told = true
		}
	}
	if !told {
		t.Fatalf("the root was exported without telling the operator to install it; lines were %#v", lines)
	}
}
