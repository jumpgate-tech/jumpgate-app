package setup

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/valve-tech/valve-node-app/internal/catalog"
	"github.com/valve-tech/valve-node-app/internal/executor"
	"github.com/valve-tech/valve-node-app/internal/ops"
)

// testGateway is a two-chain gateway with a local node in front of a
// fallback — the shape that exercises both the container rewrite and the
// multi-chain routing check.
func testGateway() catalog.GatewayConfig {
	return catalog.GatewayConfig{
		Port: 4100,
		Networks: []catalog.GatewayNetwork{
			{ChainID: 369, Upstreams: []catalog.GatewayUpstream{
				{ID: "local-node", Endpoint: "http://127.0.0.1:8545", Local: true},
				{ID: "fallback-1", Endpoint: "https://rpc.example.com"},
			}},
			{ChainID: 943, Upstreams: []catalog.GatewayUpstream{
				{ID: "testnet-1", Endpoint: "https://rpc.v4.testnet.example.com"},
			}},
		},
	}
}

// chainIDAnswer is what a healthy gateway returns for the first chain (369).
const chainIDAnswer = `{"jsonrpc":"2.0","id":1,"result":"0x171"}`

// dockerReady is a target with a working, VM-backed engine, a resolvable
// $HOME, a free port and a gateway that answers.
func dockerReady() *fakeExecutor {
	return newFakeExecutor().
		script("command -v docker", executor.Result{Stdout: "/usr/local/bin/docker\n"}).
		script("docker --version", executor.Result{Stdout: "Docker version 27.4.0, build bde2b89\n"}).
		script("docker info --format", executor.Result{Stdout: "27.4.0|linux|aarch64|docker-desktop|Docker Desktop\n"}).
		script("uname -m", executor.Result{Stdout: "arm64\n"}).
		script(`printf '%s\n' "$HOME"`, executor.Result{Stdout: "/Users/dev\n"}).
		script("eth_chainId", executor.Result{Stdout: chainIDAnswer})
}

// systemdReady is a Linux root target with a free port and a live gateway.
func systemdReady() *fakeExecutor {
	return newFakeExecutor().
		script("uname", executor.Result{Stdout: "Linux\n"}).
		script("id -u", executor.Result{Stdout: "0\n"}).
		script("eth_chainId", executor.Result{Stdout: chainIDAnswer})
}

// testGatewayID is the default gateway's id, whose container, unit and
// config file keep the historical (unsuffixed) names — which is what lets
// every name assertion in this file stay literal.
const testGatewayID = ops.DefaultGatewayID

func mustPlanGateway(t *testing.T, g catalog.GatewayConfig, backend string) []Step {
	t.Helper()
	steps, err := PlanGateway(testGatewayID, g, backend)
	if err != nil {
		t.Fatalf("PlanGateway: %v", err)
	}
	return steps
}

func stepByID(t *testing.T, steps []Step, id string) Step {
	t.Helper()
	for _, s := range steps {
		if s.ID == id {
			return s
		}
	}
	t.Fatalf("no step %q in plan", id)
	return Step{}
}

// shrinkGatewayWait keeps the ready-poll out of real time.
func shrinkGatewayWait(t *testing.T) {
	t.Helper()
	oldTimeout, oldInterval := gatewayReadyTimeout, gatewayPollInterval
	gatewayReadyTimeout = 5 * time.Millisecond
	gatewayPollInterval = time.Millisecond
	t.Cleanup(func() { gatewayReadyTimeout, gatewayPollInterval = oldTimeout, oldInterval })
}

// ---- plan shape ----

func TestPlanGateway_HasOnlyTheThreeStepsAGatewayNeeds(t *testing.T) {
	for _, backend := range []string{BackendDocker, BackendSystemd} {
		steps := mustPlanGateway(t, testGateway(), backend)

		var ids []string
		for _, s := range steps {
			ids = append(ids, s.ID)
			if s.Title == "" {
				t.Fatalf("%s: step %q has no title", backend, s.ID)
			}
		}
		if got := strings.Join(ids, ","); got != "preflight,config,run" {
			t.Fatalf("%s: plan is %q, want preflight,config,run", backend, got)
		}
		// A gateway has no dataset, no account of its own, nothing to build
		// and no peer to shake hands with.
		for _, unwanted := range []string{"account", "toolchain", "install-exec", "install-beacon", "snapshot", "wire", "start", "handshake"} {
			for _, id := range ids {
				if id == unwanted {
					t.Fatalf("%s: gateway plan must not contain a %q step", backend, unwanted)
				}
			}
		}
	}
}

// The whole reason a gateway carries an id: two gateways on ONE machine must
// not contend for a name or a file. A shared container name simply cannot be
// created twice (docker run --name fails), and a shared erpc.yaml is worse —
// each provision would silently rewrite the other's chains, and the second
// gateway would come up serving the first one's config.
func TestPlanGateway_SecondGatewayGetsItsOwnNamesAndFile(t *testing.T) {
	e := dockerReady()
	steps, err := PlanGateway("edge", testGateway(), BackendDocker)
	if err != nil {
		t.Fatalf("PlanGateway: %v", err)
	}
	if err := stepByID(t, steps, "config").Run(context.Background(), e, &State{}); err != nil {
		t.Fatalf("config: %v", err)
	}
	if _, err := e.ReadFile(context.Background(), "/Users/dev/.valve-node-app/erpc-edge.yaml"); err != nil {
		t.Fatalf("a non-default gateway must write its OWN erpc.yaml: %v", err)
	}
	if _, err := e.ReadFile(context.Background(), "/Users/dev/.valve-node-app/erpc.yaml"); err == nil {
		t.Fatal("a non-default gateway wrote the DEFAULT gateway's erpc.yaml — the two would overwrite each other on every provision")
	}

	if got := ops.ERPCContainerNameFor("edge"); got != ops.ERPCContainerName+"-edge" {
		t.Errorf("container name: got %q, want %q", got, ops.ERPCContainerName+"-edge")
	}
	if got := erpcUnitNameFor("edge"); got != "valve-node-app-erpc-edge.service" {
		t.Errorf("unit name: got %q", got)
	}
	// The default keeps the historical names, which is what stops an upgrade
	// from orphaning a gateway container that is already running.
	if got := ops.ERPCContainerNameFor(ops.DefaultGatewayID); got != ops.ERPCContainerName {
		t.Errorf("default container name: got %q, want %q", got, ops.ERPCContainerName)
	}
	if got := erpcUnitNameFor(ops.DefaultGatewayID); got != erpcUnitName {
		t.Errorf("default unit name: got %q, want %q", got, erpcUnitName)
	}
}

func TestPlanGateway_RejectsAnUnknownBackend(t *testing.T) {
	_, err := PlanGateway(testGatewayID, testGateway(), "kubernetes")
	if err == nil {
		t.Fatal("want an error for an unknown backend")
	}
	if !strings.Contains(err.Error(), "kubernetes") {
		t.Fatalf("want the offending backend named, got %v", err)
	}
}

// An unusable config must fail at plan time, not partway through RunAll with
// a container already removed.
func TestPlanGateway_ValidatesTheConfigUpFront(t *testing.T) {
	cases := map[string]catalog.GatewayConfig{
		"no networks":      {},
		"no upstreams":     {Networks: []catalog.GatewayNetwork{{ChainID: 369}}},
		"bad chain id":     {Networks: []catalog.GatewayNetwork{{ChainID: 0, Upstreams: []catalog.GatewayUpstream{{Endpoint: "http://x:1"}}}}},
		"schemeless upstr": {Networks: []catalog.GatewayNetwork{{ChainID: 369, Upstreams: []catalog.GatewayUpstream{{Endpoint: "rpc.example.com"}}}}},
	}
	for name, g := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := PlanGateway(testGatewayID, g, BackendDocker); err == nil {
				t.Fatal("want a plan-time error")
			}
		})
	}
}

// ---- preflight ----

func TestGatewayPreflight_DockerAbsenceStaysRecognizable(t *testing.T) {
	e := newFakeExecutor().script("command -v docker", executor.Result{ExitCode: 1})
	step := stepByID(t, mustPlanGateway(t, testGateway(), BackendDocker), "preflight")

	err := step.Verify(context.Background(), e, &State{})
	if err == nil {
		t.Fatal("want an error when docker is absent")
	}
	// The UI branches on this to offer an install prompt.
	if !errors.Is(err, ops.ErrDockerAbsent) {
		t.Fatalf("want ErrDockerAbsent through the wrap, got %v", err)
	}
}

func TestGatewayPreflight_DaemonDownIsAClearFailure(t *testing.T) {
	e := newFakeExecutor().
		script("command -v docker", executor.Result{Stdout: "/usr/local/bin/docker\n"}).
		script("docker info --format", executor.Result{
			ExitCode: 1,
			Stderr:   "Cannot connect to the Docker daemon at unix:///var/run/docker.sock.\n",
		})
	step := stepByID(t, mustPlanGateway(t, testGateway(), BackendDocker), "preflight")

	err := step.Verify(context.Background(), e, &State{})
	if err == nil {
		t.Fatal("want an error when no engine answers")
	}
	if !strings.Contains(err.Error(), "Cannot connect to the Docker daemon") {
		t.Fatalf("want the engine's own words surfaced, got %v", err)
	}
}

func TestGatewayPreflight_WindowsContainerModeIsItsOwnFailure(t *testing.T) {
	e := newFakeExecutor().
		script("command -v docker", executor.Result{Stdout: "docker\n"}).
		script("docker info --format", executor.Result{Stdout: "27.4.0|windows|x86_64|DESKTOP-ABC|Microsoft Windows Server 2022\n"})
	step := stepByID(t, mustPlanGateway(t, testGateway(), BackendDocker), "preflight")

	err := step.Verify(context.Background(), e, &State{})
	if err == nil || !strings.Contains(err.Error(), "Linux containers") {
		t.Fatalf("want a switch-to-Linux-containers error, got %v", err)
	}
}

func TestGatewayPreflight_DockerBackendDoesNotRequireLinuxOrRoot(t *testing.T) {
	// The point of the container backend: a macOS desktop, no root.
	e := dockerReady().
		script("uname", executor.Result{Stdout: "Darwin\n"}).
		script("id -u", executor.Result{Stdout: "501\n"})
	step := stepByID(t, mustPlanGateway(t, testGateway(), BackendDocker), "preflight")

	if err := step.Verify(context.Background(), e, &State{}); err != nil {
		t.Fatalf("a macOS desktop with docker must pass gateway preflight: %v", err)
	}
}

func TestGatewayPreflight_SystemdRequiresLinuxAndRoot(t *testing.T) {
	step := stepByID(t, mustPlanGateway(t, testGateway(), BackendSystemd), "preflight")

	notLinux := newFakeExecutor().script("uname", executor.Result{Stdout: "Darwin\n"})
	err := step.Verify(context.Background(), notLinux, &State{})
	if err == nil || !strings.Contains(err.Error(), "Darwin") {
		t.Fatalf("want a Linux-only error naming the OS, got %v", err)
	}
	if !strings.Contains(err.Error(), BackendDocker) {
		t.Fatalf("want the docker backend offered as the way out, got %v", err)
	}

	notRoot := newFakeExecutor().
		script("uname", executor.Result{Stdout: "Linux\n"}).
		script("id -u", executor.Result{Stdout: "1000\n"})
	if err := step.Verify(context.Background(), notRoot, &State{}); err == nil || !strings.Contains(err.Error(), "root") {
		t.Fatalf("want a root-required error, got %v", err)
	}
}

// The eRPC HTTP port (and the metrics port) now RECLAIM: a foreign listener is
// noted but no longer aborts a preflight that has no Run to fix it. Per the
// direction doc's "Related decision, taken separately", docker/systemd fail
// loudly and specifically on a REAL collision, so a false "busy" was the worse
// outcome. Our own gateway is still exempt (see ExemptsOurOwnRunningGateway);
// this covers the FOREIGN listener case that used to fail here.
func TestGatewayPreflight_ForeignListenerOnERPCPortNoLongerFails(t *testing.T) {
	e := dockerReady().script("grep -Ei", executor.Result{
		Stdout: "tcp   LISTEN 0  4096   127.0.0.1:4100   0.0.0.0:*\n",
	})
	step := stepByID(t, mustPlanGateway(t, testGateway(), BackendDocker), "preflight")

	if err := step.Verify(context.Background(), e, &State{}); err != nil {
		t.Fatalf("a foreign listener on the eRPC port must no longer abort preflight: %v", err)
	}
}

// A foreign listener on the eRPC/metrics port is not silently swallowed either:
// it is announced onto the event stream so the operator knows a port is being
// reclaimed and why the loud runtime failure (if any) is expected.
func TestGatewayPreflight_ForeignListenerOnERPCPortIsAnnounced(t *testing.T) {
	e := dockerReady().script("grep -Ei", executor.Result{
		Stdout: "tcp   LISTEN 0  4096   127.0.0.1:4100   0.0.0.0:*\n",
	})
	events := make(chan Event, 16)
	st := &State{Events: events}
	step := stepByID(t, mustPlanGateway(t, testGateway(), BackendDocker), "preflight")

	if err := step.Verify(context.Background(), e, st); err != nil {
		t.Fatalf("reclaim must not fail preflight: %v", err)
	}
	close(events)
	var announced bool
	for _, line := range collectLines(events) {
		if strings.Contains(line, "reclaim") && strings.Contains(line, "127.0.0.1:4100") {
			announced = true
		}
	}
	if !announced {
		t.Fatal("want a non-fatal notice naming the reclaimed listener")
	}
}

// The HTTPS front port is the deliberate, still-unresolved exception the
// direction doc carves out: reclaiming it would take down someone else's TLS
// front and the failure is silent from its owner's side. So a foreign listener
// on 8443 STILL refuses, with the port and the evidence in the error.
func TestGatewayPreflight_ForeignListenerOnHTTPSFrontStillRefuses(t *testing.T) {
	e := caddyReady().script("grep -Ei", executor.Result{
		Stdout: "tcp   LISTEN 0  4096   0.0.0.0:8443   0.0.0.0:*\n",
	})
	step := stepByID(t, mustPlanGateway(t, frontedGateway(), BackendDocker), "preflight")

	err := step.Verify(context.Background(), e, &State{})
	if err == nil {
		t.Fatal("want a refusal when a foreign listener holds the HTTPS front port")
	}
	if !strings.Contains(err.Error(), "8443") || !strings.Contains(err.Error(), "0.0.0.0:8443") {
		t.Fatalf("want the HTTPS port and the evidence in the error, got %v", err)
	}
}

// The Public (acme) tier adds a :80 probe: HTTP-01 answers the ACME challenge
// on :80, and a busy :80 makes issuance fail SILENTLY. So a foreign listener on
// :80 must fail preflight, like the HTTPS front (reclaim=false). The probe is
// scripted by the port in the grep pattern so only the :80 check sees it.
func TestGatewayPreflight_ACMERefusesBusyPortEighty(t *testing.T) {
	e := caddyReady().script("[:.]80(", executor.Result{
		Stdout: "tcp   LISTEN 0  4096   0.0.0.0:80   0.0.0.0:*\n",
	})
	step := stepByID(t, mustPlanGateway(t, acmeFrontedGateway(), BackendDocker), "preflight")

	err := step.Verify(context.Background(), e, &State{})
	if err == nil {
		t.Fatal("a busy :80 must fail acme preflight — HTTP-01 needs it and fails silently otherwise")
	}
	if !strings.Contains(err.Error(), "80") {
		t.Fatalf("want the :80 port named in the error, got %v", err)
	}
}

// A non-acme front never binds :80, so a busy :80 must NOT fail its preflight.
func TestGatewayPreflight_NonACMEIgnoresBusyPortEighty(t *testing.T) {
	e := caddyReady().script("[:.]80(", executor.Result{
		Stdout: "tcp   LISTEN 0  4096   0.0.0.0:80   0.0.0.0:*\n",
	})
	step := stepByID(t, mustPlanGateway(t, frontedGateway(), BackendDocker), "preflight")

	if err := step.Verify(context.Background(), e, &State{}); err != nil {
		t.Fatalf("an internal-CA front does not use :80, so a busy :80 must not fail preflight: %v", err)
	}
}

// The port probe must survive a target that has none of ss/netstat/lsof: a
// false "busy" terminally blocks a preflight that has no Run to fix it,
// while docker itself fails loudly on a real collision.
func TestGatewayPreflight_NoListenerToolReadsAsFree(t *testing.T) {
	e := dockerReady().script("grep -Ei", executor.Result{ExitCode: 1})
	step := stepByID(t, mustPlanGateway(t, testGateway(), BackendDocker), "preflight")

	if err := step.Verify(context.Background(), e, &State{}); err != nil {
		t.Fatalf("want preflight to pass when nothing could be probed: %v", err)
	}
}

// Re-running the plan against a live gateway (config change, resume) must
// not trip over our own listener.
func TestGatewayPreflight_ExemptsOurOwnRunningGateway(t *testing.T) {
	docker := dockerReady().
		script("docker 'inspect'", executor.Result{Stdout: "true\n"}).
		script("grep -Ei", executor.Result{Stdout: "tcp LISTEN 0 4096 127.0.0.1:4100 0.0.0.0:*\n"})
	if err := stepByID(t, mustPlanGateway(t, testGateway(), BackendDocker), "preflight").Verify(context.Background(), docker, &State{}); err != nil {
		t.Fatalf("our own running container must not fail preflight: %v", err)
	}

	systemd := systemdReady().
		script("systemctl is-active valve-node-app-erpc.service", executor.Result{Stdout: "active\n"}).
		script("grep -Ei", executor.Result{Stdout: "tcp LISTEN 0 4096 127.0.0.1:4100 0.0.0.0:*\n"})
	if err := stepByID(t, mustPlanGateway(t, testGateway(), BackendSystemd), "preflight").Verify(context.Background(), systemd, &State{}); err != nil {
		t.Fatalf("our own active unit must not fail preflight: %v", err)
	}
}

// ---- config ----

func TestGatewayConfig_DockerWritesTheContainersViewUnderHome(t *testing.T) {
	e := dockerReady()
	step := stepByID(t, mustPlanGateway(t, testGateway(), BackendDocker), "config")

	if err := step.Run(context.Background(), e, &State{}); err != nil {
		t.Fatalf("config Run: %v", err)
	}
	const want = "/Users/dev/.valve-node-app/erpc.yaml"
	got, err := e.ReadFile(context.Background(), want)
	if err != nil {
		t.Fatalf("want erpc.yaml at %s (a path the engine's VM can bind-mount): %v", want, err)
	}
	yaml := string(got)
	if !strings.Contains(yaml, `httpHostV4: "0.0.0.0"`) {
		t.Fatalf("the container's listener must be wide (a loopback listener is unreachable through -p):\n%s", yaml)
	}
	if !strings.Contains(yaml, "httpPortV4: 4000") {
		t.Fatalf("want the fixed in-container port, not the host port:\n%s", yaml)
	}
	if !strings.Contains(yaml, "http://host.docker.internal:8545") {
		t.Fatalf("the local upstream must be reachable from inside the container:\n%s", yaml)
	}
}

func TestGatewayConfig_SystemdWritesTheHostsViewAndGroupReadsIt(t *testing.T) {
	e := systemdReady()
	step := stepByID(t, mustPlanGateway(t, testGateway(), BackendSystemd), "config")

	if err := step.Run(context.Background(), e, &State{}); err != nil {
		t.Fatalf("config Run: %v", err)
	}
	const want = "/var/lib/valve-node-app/erpc.yaml"
	got, err := e.ReadFile(context.Background(), want)
	if err != nil {
		t.Fatalf("want erpc.yaml at %s (the path the unit's --config points at): %v", want, err)
	}
	yaml := string(got)
	if !strings.Contains(yaml, `httpHostV4: "127.0.0.1"`) || !strings.Contains(yaml, "httpPortV4: 4100") {
		t.Fatalf("systemd must keep the operator's own bind/port:\n%s", yaml)
	}
	if strings.Contains(yaml, "host.docker.internal") {
		t.Fatalf("the container rewrite must not leak into a systemd config:\n%s", yaml)
	}
	// Upstream URLs routinely carry provider API keys, so the file is not
	// world-readable — which means the service account has to be given it.
	var sawChgrp bool
	for _, cmd := range e.callLog() {
		if strings.HasPrefix(cmd, "chgrp "+catalog.ServiceGroup) {
			sawChgrp = true
		}
	}
	if !sawChgrp {
		t.Fatalf("want a chgrp to %s so the unprivileged unit can read its config: %#v", catalog.ServiceGroup, e.callLog())
	}
}

func TestGatewayConfig_VerifyFailsBeforeItIsWritten(t *testing.T) {
	step := stepByID(t, mustPlanGateway(t, testGateway(), BackendDocker), "config")

	if err := step.Verify(context.Background(), dockerReady(), &State{}); err == nil {
		t.Fatal("want verify to fail before the config exists")
	}
}

func TestGatewayConfig_VerifyRoundTrips(t *testing.T) {
	e := dockerReady()
	step := stepByID(t, mustPlanGateway(t, testGateway(), BackendDocker), "config")

	if err := step.Run(context.Background(), e, &State{}); err != nil {
		t.Fatalf("config Run: %v", err)
	}
	if err := step.Verify(context.Background(), e, &State{}); err != nil {
		t.Fatalf("verify must pass on what Run just wrote: %v", err)
	}
}

// A stale config from an earlier run with different chains must read as
// unverified, so Run rewrites it and the gateway is restarted onto it.
func TestGatewayConfig_VerifyDetectsDrift(t *testing.T) {
	e := dockerReady()
	if err := stepByID(t, mustPlanGateway(t, testGateway(), BackendDocker), "config").Run(context.Background(), e, &State{}); err != nil {
		t.Fatalf("config Run: %v", err)
	}

	changed := testGateway()
	changed.Networks[0].Upstreams[1].Endpoint = "https://other-provider.example.com"
	err := stepByID(t, mustPlanGateway(t, changed, BackendDocker), "config").Verify(context.Background(), e, &State{})
	if err == nil {
		t.Fatal("want verify to fail against a config written for different upstreams")
	}
	if !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("want a drift-shaped error, got %v", err)
	}
}

// ---- run ----

func TestGatewayRun_DockerRemovesStaleThenRunsWithPlatform(t *testing.T) {
	shrinkGatewayWait(t)
	e := dockerReady()
	step := stepByID(t, mustPlanGateway(t, testGateway(), BackendDocker), "run")

	if err := step.Run(context.Background(), e, &State{}); err != nil {
		t.Fatalf("run: %v", err)
	}

	idxRM, idxRun := -1, -1
	for i, cmd := range e.callLog() {
		switch {
		case strings.HasPrefix(cmd, "docker 'rm' '-f'"):
			idxRM = i
		case strings.HasPrefix(cmd, "docker 'run'"):
			idxRun = i
		}
	}
	if idxRM < 0 || idxRun < 0 {
		t.Fatalf("want a forced remove and a run: %#v", e.callLog())
	}
	if idxRM > idxRun {
		// A container's ports, mounts and image are fixed at creation, so a
		// config change is necessarily remove + run.
		t.Fatalf("remove (%d) must precede run (%d)", idxRM, idxRun)
	}

	run := e.callLog()[idxRun]
	// The Rosetta bug: without an explicit --platform on RUN, docker looks
	// up the wrong variant of a present local image and tries to pull.
	if !strings.Contains(run, "'--platform' 'linux/arm64'") {
		t.Fatalf("want --platform linux/arm64 from the host's own arch reading, got %q", run)
	}
	if !strings.Contains(run, "'-p' '127.0.0.1:4100:4000'") {
		t.Fatalf("want the operator's host port published onto the fixed container port, got %q", run)
	}
	if !strings.Contains(run, "'/Users/dev/.valve-node-app/erpc.yaml:/erpc.yaml:ro'") {
		t.Fatalf("want the resolved config path mounted read-only, got %q", run)
	}
	// A VM-backed engine already provides the host alias.
	if strings.Contains(run, "--add-host") {
		t.Fatalf("want no --add-host on a VM-backed engine, got %q", run)
	}
}

func TestGatewayRun_DockerAddsHostGatewayOnAPlainLinuxEngine(t *testing.T) {
	shrinkGatewayWait(t)
	e := dockerReady().
		script("docker info --format", executor.Result{Stdout: "27.3.1|linux|x86_64|node-01|Ubuntu 24.04.1 LTS\n"}).
		script("uname -m", executor.Result{Stdout: "x86_64\n"})
	step := stepByID(t, mustPlanGateway(t, testGateway(), BackendDocker), "run")

	if err := step.Run(context.Background(), e, &State{}); err != nil {
		t.Fatalf("run: %v", err)
	}
	var run string
	for _, cmd := range e.callLog() {
		if strings.HasPrefix(cmd, "docker 'run'") {
			run = cmd
		}
	}
	if !strings.Contains(run, "'--add-host' 'host.docker.internal:host-gateway'") {
		t.Fatalf("a plain Linux engine has no host alias unless we map it: %q", run)
	}
	if !strings.Contains(run, "'--platform' 'linux/amd64'") {
		t.Fatalf("want the engine's own arch honoured on a non-VM engine: %q", run)
	}
}

func TestGatewayRun_DockerFailureSurfacesTheEnginesMessage(t *testing.T) {
	shrinkGatewayWait(t)
	e := dockerReady().script("docker 'run'", executor.Result{
		ExitCode: 125,
		Stderr:   "docker: Error response from daemon: driver failed programming external connectivity: port is already allocated.\n",
	})
	step := stepByID(t, mustPlanGateway(t, testGateway(), BackendDocker), "run")

	err := step.Run(context.Background(), e, &State{})
	if err == nil {
		t.Fatal("want an error when docker run fails")
	}
	if !strings.Contains(err.Error(), "port is already allocated") {
		t.Fatalf("want the engine's message surfaced, got %v", err)
	}
}

func TestGatewayRun_SystemdWritesTheUnitAndRestarts(t *testing.T) {
	shrinkGatewayWait(t)
	e := systemdReady()
	step := stepByID(t, mustPlanGateway(t, testGateway(), BackendSystemd), "run")

	if err := step.Run(context.Background(), e, &State{}); err != nil {
		t.Fatalf("run: %v", err)
	}
	unit, err := e.ReadFile(context.Background(), "/etc/systemd/system/valve-node-app-erpc.service")
	if err != nil {
		t.Fatalf("want the unit written: %v", err)
	}
	for _, want := range []string{
		"User=" + catalog.ServiceUser,
		"ExecStart=erpc --config /var/lib/valve-node-app/erpc.yaml",
		"ReadWritePaths=/var/lib/valve-node-app",
	} {
		if !strings.Contains(string(unit), want) {
			t.Fatalf("want %q in the rendered unit:\n%s", want, unit)
		}
	}
	// The restart is what applies a changed config — `enable --now` alone
	// leaves an already-running unit on its old config.
	var sawRestart bool
	for _, cmd := range e.callLog() {
		if strings.Contains(cmd, "systemctl restart valve-node-app-erpc.service") {
			sawRestart = true
		}
	}
	if !sawRestart {
		t.Fatalf("want an explicit restart: %#v", e.callLog())
	}
}

func TestGatewayRun_VerifyProbesTheFirstChainsPathOnTheGatewaysOwnPort(t *testing.T) {
	e := dockerReady()
	step := stepByID(t, mustPlanGateway(t, testGateway(), BackendDocker), "run")

	if err := step.Verify(context.Background(), e, &State{}); err != nil {
		t.Fatalf("verify: %v", err)
	}
	var probe string
	for _, cmd := range e.callLog() {
		if strings.Contains(cmd, "eth_chainId") {
			probe = cmd
		}
	}
	// The URL callers actually use: /<project>/evm/<chainId> on the
	// gateway's own port. WebSocket rides the same path, so proving this
	// path resolves proves both.
	if !strings.Contains(probe, "'http://127.0.0.1:4100/main/evm/369'") {
		t.Fatalf("want the first chain's gateway path probed, got %q", probe)
	}
}

func TestGatewayRun_VerifyProbesLoopbackForAWildcardBind(t *testing.T) {
	g := testGateway()
	g.BindAddr = "0.0.0.0"
	e := newFakeExecutor().script("eth_chainId", executor.Result{Stdout: chainIDAnswer})

	if err := stepByID(t, mustPlanGateway(t, g, BackendDocker), "run").Verify(context.Background(), e, &State{}); err != nil {
		t.Fatalf("verify: %v", err)
	}
	if probe := e.callLog()[0]; !strings.Contains(probe, "http://127.0.0.1:4100/") {
		t.Fatalf("0.0.0.0 is not a connectable destination; want loopback probed, got %q", probe)
	}
}

func TestGatewayRun_VerifyRejectsAMisroutedChain(t *testing.T) {
	// 0x1 is mainnet: the path says 369, so this gateway is misconfigured.
	e := dockerReady().script("eth_chainId", executor.Result{Stdout: `{"jsonrpc":"2.0","id":1,"result":"0x1"}`})

	err := stepByID(t, mustPlanGateway(t, testGateway(), BackendDocker), "run").Verify(context.Background(), e, &State{})
	if err == nil {
		t.Fatal("want an error when the gateway answers for another chain")
	}
	if !strings.Contains(err.Error(), "serving chain 1") {
		t.Fatalf("want the mismatch spelled out, got %v", err)
	}
}

func TestGatewayRun_VerifyFailsOnAnEmptyOrErrorAnswer(t *testing.T) {
	cases := map[string]string{
		"nothing listening": "",
		"rpc error":         `{"jsonrpc":"2.0","id":1,"error":{"code":-32603,"message":"no upstream available"}}`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			e := dockerReady().script("eth_chainId", executor.Result{Stdout: body})
			if err := stepByID(t, mustPlanGateway(t, testGateway(), BackendDocker), "run").Verify(context.Background(), e, &State{}); err == nil {
				t.Fatal("want an error")
			}
		})
	}
}

func TestGatewayRun_TimesOutWhenTheGatewayNeverAnswers(t *testing.T) {
	shrinkGatewayWait(t)
	e := dockerReady().script("eth_chainId", executor.Result{Stdout: ""})

	err := stepByID(t, mustPlanGateway(t, testGateway(), BackendDocker), "run").Run(context.Background(), e, &State{})
	if err == nil {
		t.Fatal("want a timeout error")
	}
	if !strings.Contains(err.Error(), "did not answer") {
		t.Fatalf("want a ready-timeout error, got %v", err)
	}
}

// ---- end to end through the engine ----

func TestPlanGateway_RunAllDockerHappyPath(t *testing.T) {
	shrinkGatewayWait(t)
	e := dockerReady()
	events := make(chan Event, 64)
	st := &State{Events: events}

	if err := RunAll(context.Background(), e, mustPlanGateway(t, testGateway(), BackendDocker), st); err != nil {
		t.Fatalf("RunAll: %v", err)
	}
	close(events)

	done := map[string]bool{}
	for ev := range events {
		if ev.Done {
			done[ev.StepID] = true
		}
	}
	for _, id := range []string{"preflight", "config", "run"} {
		if !done[id] {
			t.Fatalf("step %q never reported done", id)
		}
	}
}

// A live gateway answers eth_chainId identically whichever config it is
// serving, so "it responds" alone must NOT be enough to skip the restart
// after an upstream change — otherwise setup reports success while the old
// config keeps serving.
func TestPlanGateway_ConfigChangeRestartsAnAlreadyLiveGateway(t *testing.T) {
	shrinkGatewayWait(t)
	e := dockerReady()
	if err := RunAll(context.Background(), e, mustPlanGateway(t, testGateway(), BackendDocker), &State{}); err != nil {
		t.Fatalf("first RunAll: %v", err)
	}
	before := len(e.callLog())

	changed := testGateway()
	changed.Networks[0].Upstreams[1].Endpoint = "https://other-provider.example.com"
	if err := RunAll(context.Background(), e, mustPlanGateway(t, changed, BackendDocker), &State{}); err != nil {
		t.Fatalf("second RunAll: %v", err)
	}

	var rewrote, restarted bool
	for _, cmd := range e.callLog()[before:] {
		if strings.HasPrefix(cmd, "WriteFile") {
			rewrote = true
		}
		if strings.HasPrefix(cmd, "docker 'run'") {
			restarted = true
		}
	}
	if !rewrote {
		t.Fatal("want the changed config rewritten")
	}
	if !restarted {
		t.Fatalf("want the gateway re-created onto the new config: %#v", e.callLog()[before:])
	}
}

// Second run against the now-live gateway: every step's Verify already
// holds, so nothing is touched. This is the idempotence the engine's
// pre-check contract buys, and the reason the run step's Verify is a real
// RPC call rather than a container-exists check.
func TestPlanGateway_RunAllIsIdempotent(t *testing.T) {
	shrinkGatewayWait(t)
	e := dockerReady()
	steps := mustPlanGateway(t, testGateway(), BackendDocker)

	if err := RunAll(context.Background(), e, steps, &State{}); err != nil {
		t.Fatalf("first RunAll: %v", err)
	}
	before := len(e.callLog())
	if err := RunAll(context.Background(), e, steps, &State{}); err != nil {
		t.Fatalf("second RunAll: %v", err)
	}
	for _, cmd := range e.callLog()[before:] {
		if strings.HasPrefix(cmd, "docker 'run'") || strings.HasPrefix(cmd, "docker 'rm'") {
			t.Fatalf("a satisfied gateway must not be re-created: %q", cmd)
		}
		if strings.HasPrefix(cmd, "WriteFile") {
			t.Fatalf("a satisfied config must not be rewritten: %q", cmd)
		}
	}
}

// The gateway image is built on the target rather than pulled: upstream eRPC
// has no WebSocket support and the valve fork publishes no image, so a pull
// would fetch either the wrong binary or nothing at all.

func TestRunDocker_BuildsTheImageWhenAbsent(t *testing.T) {
	e := dockerReady().
		script("docker image inspect", executor.Result{ExitCode: 1}).
		script("docker 'build'", executor.Result{Stdout: "built\n"}).
		script("docker 'run'", executor.Result{Stdout: "deadbeef\n"})

	steps := mustPlanGateway(t, testGateway(), BackendDocker)
	run := stepByID(t, steps, "run")
	if err := run.Run(context.Background(), e, &State{}); err != nil {
		t.Fatalf("run: %v", err)
	}

	build := ""
	for _, c := range e.callLog() {
		if strings.Contains(c, "docker 'build'") {
			build = c
		}
	}
	if build == "" {
		t.Fatalf("no docker build ran: %v", e.callLog())
	}
	if !strings.Contains(build, ops.ERPCSourceRef) {
		t.Errorf("build must pin the source SHA, got: %s", build)
	}
	if strings.Contains(build, "#valve-ws") {
		t.Errorf("build must not track a moving branch head, got: %s", build)
	}
	if !strings.Contains(build, ops.ERPCImageTag()) {
		t.Errorf("build must tag %q, got: %s", ops.ERPCImageTag(), build)
	}
}

func TestRunDocker_SkipsTheBuildWhenTheImageIsPresent(t *testing.T) {
	e := dockerReady().
		script("docker image inspect", executor.Result{Stdout: "sha256:abc\n"}).
		script("docker 'run'", executor.Result{Stdout: "deadbeef\n"})

	steps := mustPlanGateway(t, testGateway(), BackendDocker)
	run := stepByID(t, steps, "run")
	if err := run.Run(context.Background(), e, &State{}); err != nil {
		t.Fatalf("run: %v", err)
	}
	// A rebuild on every provisioning run would turn a seconds-long re-run
	// into a minutes-long one.
	for _, cmd := range e.callLog() {
		if strings.Contains(cmd, "docker 'build'") {
			t.Fatalf("image was present; no build should have run, got: %s", cmd)
		}
	}
}
