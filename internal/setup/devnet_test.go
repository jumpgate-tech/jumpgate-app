package setup

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/valve-tech/valve-node-app/internal/catalog"
	"github.com/valve-tech/valve-node-app/internal/executor"
	"github.com/valve-tech/valve-node-app/internal/ops"
)

// No real docker and no network anywhere in here: every fact about the engine,
// the container and the chain is scripted. What was verified by hand — that
// ghcr.io/paradigmxyz/reth:latest is multi-arch, that --dev seals blocks with
// no consensus client, that the chain answers 0x539 — is encoded in
// catalog/devnet.go and asserted there.

// devnetChainIDAnswer / devnetHeightAnswer are what a healthy devnet returns:
// chain 1337, a few blocks in.
const (
	devnetChainIDAnswer = `{"jsonrpc":"2.0","id":1,"result":"0x539"}`
	devnetHeightAnswer  = `{"jsonrpc":"2.0","id":1,"result":"0x7"}`
)

// testDevnet is a devnet on non-default host ports, so that a test asserting a
// port mapping cannot pass by coincidence with the fixed container ports.
func testDevnet() catalog.DevnetConfig {
	return catalog.DevnetConfig{HTTPPort: 18545, WSPort: 18546}
}

// devnetReady is a target with a working, VM-backed engine, free ports and a
// devnet already answering on the requested settings.
func devnetReady(d catalog.DevnetConfig) *fakeExecutor {
	cmd, err := json.Marshal(catalog.DevnetCommand(d))
	if err != nil {
		panic(err)
	}
	return newFakeExecutor().
		script("command -v docker", executor.Result{Stdout: "/usr/local/bin/docker\n"}).
		script("docker --version", executor.Result{Stdout: "Docker version 27.4.0, build bde2b89\n"}).
		script("docker info --format", executor.Result{Stdout: "27.4.0|linux|aarch64|docker-desktop|Docker Desktop\n"}).
		script("uname -m", executor.Result{Stdout: "arm64\n"}).
		script("eth_chainId", executor.Result{Stdout: devnetChainIDAnswer}).
		script("eth_blockNumber", executor.Result{Stdout: devnetHeightAnswer}).
		script("{{json .Config.Cmd}}", executor.Result{Stdout: string(cmd) + "\n"})
}

func mustPlanDevnet(t *testing.T, d catalog.DevnetConfig) []Step {
	t.Helper()
	steps, err := PlanDevnet(d)
	if err != nil {
		t.Fatalf("PlanDevnet: %v", err)
	}
	return steps
}

// shrinkDevnetWait keeps the ready-poll out of real time.
func shrinkDevnetWait(t *testing.T) {
	t.Helper()
	oldTimeout, oldInterval := devnetReadyTimeout, devnetPollInterval
	devnetReadyTimeout = 5 * time.Millisecond
	devnetPollInterval = time.Millisecond
	t.Cleanup(func() { devnetReadyTimeout, devnetPollInterval = oldTimeout, oldInterval })
}

// liveDocker wraps the shared fake with the one behaviour a static script
// cannot express: creating a container CHANGES what `docker inspect` afterwards
// reports about it. Without that, a plan run twice with different settings
// could never be observed converging — the drift the second run is supposed to
// fix would still be there once it had fixed it.
type liveDocker struct{ *fakeExecutor }

func (l liveDocker) Run(ctx context.Context, cmd string, opts *executor.RunOpts) (executor.Result, error) {
	res, err := l.fakeExecutor.Run(ctx, cmd, opts)
	if strings.HasPrefix(cmd, "docker 'run'") {
		argv := unquoteArgv(cmd)
		for i, a := range argv {
			if a != catalog.DefaultDevnetImage {
				continue
			}
			// Everything after the image ref is what docker records as the
			// container's command, which is what .Config.Cmd reports back.
			body, mErr := json.Marshal(argv[i+1:])
			if mErr != nil {
				panic(mErr)
			}
			l.script("{{json .Config.Cmd}}", executor.Result{Stdout: string(body) + "\n"})
			break
		}
	}
	return res, err
}

// unquoteArgv reverses ops.DockerRun's per-argument sh quoting. Adequate
// because no argument this package renders for `docker run` contains a space.
func unquoteArgv(cmd string) []string {
	fields := strings.Fields(cmd)
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		out = append(out, strings.Trim(f, "'"))
	}
	return out
}

// lastCallWithPrefix returns the last recorded command starting with prefix.
func lastCallWithPrefix(e *fakeExecutor, prefix string) string {
	var out string
	for _, cmd := range e.callLog() {
		if strings.HasPrefix(cmd, prefix) {
			out = cmd
		}
	}
	return out
}

// ---- plan shape ----

func TestPlanDevnet_HasOnlyTheTwoStepsADevnetNeeds(t *testing.T) {
	steps := mustPlanDevnet(t, testDevnet())

	var ids []string
	for _, s := range steps {
		ids = append(ids, s.ID)
		if s.Title == "" {
			t.Fatalf("step %q has no title", s.ID)
		}
	}
	if got := strings.Join(ids, ","); got != "preflight,run" {
		t.Fatalf("plan is %q, want preflight,run", got)
	}
	// A devnet has no config file, no dataset, no checkpoint, no snapshot, no
	// service account, no JWT and no beacon client to shake hands with — the
	// entire reason it is worth modelling separately.
	for _, unwanted := range []string{"config", "account", "toolchain", "install-exec", "install-beacon", "snapshot", "wire", "handshake"} {
		if slices.Contains(ids, unwanted) {
			t.Fatalf("devnet plan must not contain a %q step", unwanted)
		}
	}
}

// An unusable devnet must fail at plan time, not partway through RunAll with
// a container already removed.
func TestPlanDevnet_ValidatesTheConfigUpFront(t *testing.T) {
	cases := map[string]catalog.DevnetConfig{
		"chain reth cannot serve": {ChainID: 1},
		"unparsable block time":   {BlockTime: "every so often"},
		"one port for two things": {HTTPPort: 9000, WSPort: 9000},
	}
	for name, d := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := PlanDevnet(d); err == nil {
				t.Fatal("want a plan-time error")
			}
		})
	}
}

// ---- preflight ----

func TestDevnetPreflight_DockerAbsenceStaysRecognizable(t *testing.T) {
	e := newFakeExecutor().script("command -v docker", executor.Result{ExitCode: 1})

	err := stepByID(t, mustPlanDevnet(t, testDevnet()), "preflight").Verify(context.Background(), e, &State{})
	if err == nil {
		t.Fatal("want an error when docker is absent")
	}
	// The UI branches on this to offer an install prompt.
	if !errors.Is(err, ops.ErrDockerAbsent) {
		t.Fatalf("want ErrDockerAbsent through the wrap, got %v", err)
	}
}

func TestDevnetPreflight_DaemonDownIsAClearFailure(t *testing.T) {
	e := newFakeExecutor().
		script("command -v docker", executor.Result{Stdout: "/usr/local/bin/docker\n"}).
		script("docker info --format", executor.Result{
			ExitCode: 1,
			Stderr:   "Cannot connect to the Docker daemon at unix:///var/run/docker.sock.\n",
		})

	err := stepByID(t, mustPlanDevnet(t, testDevnet()), "preflight").Verify(context.Background(), e, &State{})
	if err == nil || !strings.Contains(err.Error(), "Cannot connect to the Docker daemon") {
		t.Fatalf("want the engine's own words surfaced, got %v", err)
	}
}

func TestDevnetPreflight_WindowsContainerModeIsItsOwnFailure(t *testing.T) {
	e := newFakeExecutor().
		script("command -v docker", executor.Result{Stdout: "docker\n"}).
		script("docker info --format", executor.Result{Stdout: "27.4.0|windows|x86_64|DESKTOP-ABC|Microsoft Windows Server 2022\n"})

	err := stepByID(t, mustPlanDevnet(t, testDevnet()), "preflight").Verify(context.Background(), e, &State{})
	if err == nil || !strings.Contains(err.Error(), "Linux containers") {
		t.Fatalf("want a switch-to-Linux-containers error, got %v", err)
	}
}

// The whole point of a container-hosted devnet: an unprivileged macOS desktop
// qualifies. Requiring Linux or root here would put the local devnet out of
// reach of the machine it exists for.
func TestDevnetPreflight_DoesNotRequireLinuxOrRoot(t *testing.T) {
	e := devnetReady(testDevnet()).
		script("uname", executor.Result{Stdout: "Darwin\n"}).
		script("id -u", executor.Result{Stdout: "501\n"})

	if err := stepByID(t, mustPlanDevnet(t, testDevnet()), "preflight").Verify(context.Background(), e, &State{}); err != nil {
		t.Fatalf("a macOS desktop with docker must pass devnet preflight: %v", err)
	}
}

// 8546 is every EVM client's default WebSocket port, so on a machine that
// already runs a node it is at least as likely to be taken as 8545 — and
// docker binds both mappings at creation.
func TestDevnetPreflight_ChecksBothPublishedPorts(t *testing.T) {
	for _, busy := range []int{18545, 18546} {
		e := devnetReady(testDevnet()).script(strings.ReplaceAll(listenerProbe, "%d", strconv.Itoa(busy)), executor.Result{
			Stdout: "tcp   LISTEN 0  4096   127.0.0.1:" + strconv.Itoa(busy) + "   0.0.0.0:*\n",
		})

		err := stepByID(t, mustPlanDevnet(t, testDevnet()), "preflight").Verify(context.Background(), e, &State{})
		if err == nil {
			t.Fatalf("want an error when port %d is taken", busy)
		}
		if !strings.Contains(err.Error(), strconv.Itoa(busy)) {
			t.Fatalf("want port %d named in the error, got %v", busy, err)
		}
	}
}

// A false "busy" terminally blocks a preflight that has no Run to fix it,
// while docker itself fails loudly on a real collision.
func TestDevnetPreflight_NoListenerToolReadsAsFree(t *testing.T) {
	e := devnetReady(testDevnet()).script("grep -Ei", executor.Result{ExitCode: 1})

	if err := stepByID(t, mustPlanDevnet(t, testDevnet()), "preflight").Verify(context.Background(), e, &State{}); err != nil {
		t.Fatalf("want preflight to pass when nothing could be probed: %v", err)
	}
}

// Re-running the plan against a live devnet (a settings change, a resume) must
// not trip over our own listener.
func TestDevnetPreflight_ExemptsOurOwnRunningDevnet(t *testing.T) {
	e := devnetReady(testDevnet()).
		script("{{.State.Running}}", executor.Result{Stdout: "true\n"}).
		script("grep -Ei", executor.Result{Stdout: "tcp LISTEN 0 4096 127.0.0.1:18545 0.0.0.0:*\n"})

	if err := stepByID(t, mustPlanDevnet(t, testDevnet()), "preflight").Verify(context.Background(), e, &State{}); err != nil {
		t.Fatalf("our own running container must not fail preflight: %v", err)
	}
}

// ---- run ----

func TestDevnetRun_RemovesStaleThenRunsTheVerifiedInvocation(t *testing.T) {
	shrinkDevnetWait(t)
	e := devnetReady(testDevnet())

	if err := stepByID(t, mustPlanDevnet(t, testDevnet()), "run").Run(context.Background(), e, &State{}); err != nil {
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
		// Ports, image and command are fixed at creation, so applying a
		// changed setting is necessarily remove + run.
		t.Fatalf("remove (%d) must precede run (%d)", idxRM, idxRun)
	}

	run := e.callLog()[idxRun]
	// The Rosetta bug: without an explicit --platform on RUN, docker looks up
	// the wrong variant of the manifest.
	if !strings.Contains(run, "'--platform' 'linux/arm64'") {
		t.Fatalf("want --platform from the host's own arch reading, got %q", run)
	}
	if !strings.Contains(run, "'-p' '127.0.0.1:18545:8545'") || !strings.Contains(run, "'-p' '127.0.0.1:18546:8546'") {
		t.Fatalf("want both host ports published onto the fixed container ports, got %q", run)
	}
	if !strings.Contains(run, "'"+catalog.DefaultDevnetImage+"'") {
		t.Fatalf("want the upstream multi-arch reth image pulled, got %q", run)
	}
	if !strings.Contains(run, "'--dev'") {
		t.Fatalf("without --dev this is an ordinary reth with nothing to sync from: %q", run)
	}
}

// The gateway image is built on the target because upstream publishes no
// usable one; reth's is published multi-arch, so building it would cost
// minutes to produce the same binary.
func TestDevnetRun_PullsRatherThanBuilds(t *testing.T) {
	shrinkDevnetWait(t)
	e := devnetReady(testDevnet())

	if err := stepByID(t, mustPlanDevnet(t, testDevnet()), "run").Run(context.Background(), e, &State{}); err != nil {
		t.Fatalf("run: %v", err)
	}
	for _, cmd := range e.callLog() {
		if strings.HasPrefix(cmd, "docker 'build'") {
			t.Fatalf("a devnet must not build an image: %q", cmd)
		}
	}
}

func TestDevnetRun_HonoursTheEnginesArchOnAPlainLinuxEngine(t *testing.T) {
	shrinkDevnetWait(t)
	e := devnetReady(testDevnet()).
		script("docker info --format", executor.Result{Stdout: "27.3.1|linux|x86_64|node-01|Ubuntu 24.04.1 LTS\n"}).
		script("uname -m", executor.Result{Stdout: "x86_64\n"})

	if err := stepByID(t, mustPlanDevnet(t, testDevnet()), "run").Run(context.Background(), e, &State{}); err != nil {
		t.Fatalf("run: %v", err)
	}
	if run := lastCallWithPrefix(e, "docker 'run'"); !strings.Contains(run, "'--platform' 'linux/amd64'") {
		t.Fatalf("want the engine's own arch honoured on a non-VM engine: %q", run)
	}
}

func TestDevnetRun_FailureSurfacesTheEnginesMessage(t *testing.T) {
	shrinkDevnetWait(t)
	e := devnetReady(testDevnet()).script("docker 'run'", executor.Result{
		ExitCode: 125,
		Stderr:   "docker: Error response from daemon: driver failed programming external connectivity: port is already allocated.\n",
	})

	err := stepByID(t, mustPlanDevnet(t, testDevnet()), "run").Run(context.Background(), e, &State{})
	if err == nil {
		t.Fatal("want an error when docker run fails")
	}
	if !strings.Contains(err.Error(), "port is already allocated") {
		t.Fatalf("want the engine's message surfaced, got %v", err)
	}
}

func TestDevnetRun_VerifyProbesTheDevnetsOwnPort(t *testing.T) {
	e := devnetReady(testDevnet())

	if err := stepByID(t, mustPlanDevnet(t, testDevnet()), "run").Verify(context.Background(), e, &State{}); err != nil {
		t.Fatalf("verify: %v", err)
	}
	if probe := lastCallWithPrefix(e, "curl"); !strings.Contains(probe, "'http://127.0.0.1:18545'") {
		t.Fatalf("want the devnet's host port probed, got %q", probe)
	}
}

// Something else on 8545 — an already-running node, a devnet from another
// project — answers eth_chainId perfectly happily; only the id tells them
// apart.
func TestDevnetRun_VerifyRejectsAnotherChainOnThePort(t *testing.T) {
	e := devnetReady(testDevnet()).script("eth_chainId", executor.Result{Stdout: `{"jsonrpc":"2.0","id":1,"result":"0x1"}`})

	err := stepByID(t, mustPlanDevnet(t, testDevnet()), "run").Verify(context.Background(), e, &State{})
	if err == nil {
		t.Fatal("want an error when the port serves another chain")
	}
	if !strings.Contains(err.Error(), "serving chain 1") {
		t.Fatalf("want the mismatch spelled out, got %v", err)
	}
}

// reth answers RPC before it has sealed anything, so a chain-id check alone
// would report a devnet ready while the sealer was silently doing nothing —
// and producing blocks is the entire service a devnet provides.
func TestDevnetRun_VerifyRequiresASealedBlock(t *testing.T) {
	e := devnetReady(testDevnet()).script("eth_blockNumber", executor.Result{Stdout: `{"jsonrpc":"2.0","id":1,"result":"0x0"}`})

	err := stepByID(t, mustPlanDevnet(t, testDevnet()), "run").Verify(context.Background(), e, &State{})
	if err == nil {
		t.Fatal("want an error while the chain is still at block 0")
	}
	if !strings.Contains(err.Error(), "block 0") {
		t.Fatalf("want a not-sealing-yet error, got %v", err)
	}
}

func TestDevnetRun_VerifyFailsOnAnEmptyOrErrorAnswer(t *testing.T) {
	cases := map[string]string{
		"nothing listening": "",
		"rpc error":         `{"jsonrpc":"2.0","id":1,"error":{"code":-32601,"message":"method not found"}}`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			e := devnetReady(testDevnet()).script("eth_chainId", executor.Result{Stdout: body})
			if err := stepByID(t, mustPlanDevnet(t, testDevnet()), "run").Verify(context.Background(), e, &State{}); err == nil {
				t.Fatal("want an error")
			}
		})
	}
}

func TestDevnetRun_TimesOutWhenTheChainNeverAnswers(t *testing.T) {
	shrinkDevnetWait(t)
	e := devnetReady(testDevnet()).script("eth_chainId", executor.Result{Stdout: ""})

	err := stepByID(t, mustPlanDevnet(t, testDevnet()), "run").Run(context.Background(), e, &State{})
	if err == nil {
		t.Fatal("want a timeout error")
	}
	if !strings.Contains(err.Error(), "did not answer") {
		t.Fatalf("want a ready-timeout error, got %v", err)
	}
}

// ---- end to end through the engine ----

func TestPlanDevnet_RunAllHappyPath(t *testing.T) {
	shrinkDevnetWait(t)
	e := devnetReady(testDevnet())
	events := make(chan Event, 64)
	st := &State{Events: events}

	if err := RunAll(context.Background(), e, mustPlanDevnet(t, testDevnet()), st); err != nil {
		t.Fatalf("RunAll: %v", err)
	}
	close(events)

	done := map[string]bool{}
	for ev := range events {
		if ev.Done {
			done[ev.StepID] = true
		}
	}
	for _, id := range []string{"preflight", "run"} {
		if !done[id] {
			t.Fatalf("step %q never reported done", id)
		}
	}
}

// A satisfied devnet must not be re-created: doing so would throw away the
// operator's chain state on every re-run.
func TestPlanDevnet_RunAllIsIdempotent(t *testing.T) {
	shrinkDevnetWait(t)
	e := devnetReady(testDevnet())
	steps := mustPlanDevnet(t, testDevnet())

	if err := RunAll(context.Background(), e, steps, &State{}); err != nil {
		t.Fatalf("first RunAll: %v", err)
	}
	before := len(e.callLog())
	if err := RunAll(context.Background(), e, steps, &State{}); err != nil {
		t.Fatalf("second RunAll: %v", err)
	}
	for _, cmd := range e.callLog()[before:] {
		if strings.HasPrefix(cmd, "docker 'run'") || strings.HasPrefix(cmd, "docker 'rm'") {
			t.Fatalf("a satisfied devnet must not be re-created: %q", cmd)
		}
	}
}

// A devnet has no config file, so a settings change is visible only in the
// container's command line. Without reading that back, a block-time change
// would leave the old container serving while setup reported success — the
// same trap gatewayPlan's configPending exists to avoid.
func TestPlanDevnet_ASettingsChangeReCreatesALiveDevnet(t *testing.T) {
	shrinkDevnetWait(t)
	e := liveDocker{devnetReady(testDevnet())}
	if err := RunAll(context.Background(), e, mustPlanDevnet(t, testDevnet()), &State{}); err != nil {
		t.Fatalf("first RunAll: %v", err)
	}
	before := len(e.callLog())

	changed := testDevnet()
	changed.BlockTime = "5s"
	if err := RunAll(context.Background(), e, mustPlanDevnet(t, changed), &State{}); err != nil {
		t.Fatalf("second RunAll: %v", err)
	}

	run := ""
	for _, cmd := range e.callLog()[before:] {
		if strings.HasPrefix(cmd, "docker 'run'") {
			run = cmd
		}
	}
	if run == "" {
		t.Fatalf("want the devnet re-created onto the new settings: %#v", e.callLog()[before:])
	}
	if !strings.Contains(run, "'--dev.block-time' '5s'") {
		t.Fatalf("want the new block time in the new container: %q", run)
	}
}

// A devnet answers eth_chainId identically whichever block time it was started
// with, so "it responds" alone must not be enough to skip the re-create.
func TestDevnetRun_VerifyDetectsSettingsItDidNotAskFor(t *testing.T) {
	running := testDevnet()
	running.BlockTime = "12s"
	e := devnetReady(running) // the container on the target, started earlier

	// The plan wants the default block time; the live container has 12s.
	err := stepByID(t, mustPlanDevnet(t, testDevnet()), "run").Verify(context.Background(), e, &State{})
	if err == nil {
		t.Fatal("want verify to fail against a devnet running other settings")
	}
	if !strings.Contains(err.Error(), "12s") || !strings.Contains(err.Error(), catalog.DefaultDevnetBlockTime) {
		t.Fatalf("want both the running and the requested settings in the error, got %v", err)
	}
}

// An unreadable inspect must NOT be read as drift: failing closed there would
// re-create the container (and destroy its chain) on every single verify.
func TestDevnetRun_VerifyDoesNotClaimDriftItCannotSee(t *testing.T) {
	cases := map[string]executor.Result{
		"no such container": {ExitCode: 1, Stderr: "Error: No such object: valve-node-app-devnet\n"},
		"empty answer":      {Stdout: "\n"},
		"not json":          {Stdout: "<nil>\n"},
		"null command":      {Stdout: "null\n"},
	}
	for name, res := range cases {
		t.Run(name, func(t *testing.T) {
			e := devnetReady(testDevnet()).script("{{json .Config.Cmd}}", res)
			if err := stepByID(t, mustPlanDevnet(t, testDevnet()), "run").Verify(context.Background(), e, &State{}); err != nil {
				t.Fatalf("want the chain-id probe to be the only gate here: %v", err)
			}
		})
	}
}
