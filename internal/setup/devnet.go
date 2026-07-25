package setup

// Devnet provisioning: getting a throwaway local chain up, so that the rest of
// this app — a gateway in front of a node, the dashboard's health reads, the
// whole start/stop lifecycle — can be exercised end to end on a laptop.
//
// WHY this is its own plan rather than a mode of Plan() or of PlanGateway():
// the nine-step node chain exists because a real chain client is a big,
// stateful, Linux-only install — a service account, a toolchain, a terabyte of
// dataset, a JWT, and a two-process handshake before it is worth anything. A
// devnet has literally none of that (catalog/devnet.go explains why: reth's
// --dev mode self-seals from an in-image genesis). It is shorter than even the
// gateway's three steps, because a gateway at least has a config FILE to
// write, and a devnet's entire configuration is command-line flags baked into
// the container at creation.
//
// So: two steps. Preflight (can this target host a container, and are the
// ports free) and run (create the container and confirm the chain is actually
// producing blocks). PlanGateway is the model for both — same docker probe,
// same remove-before-run, same readiness poll, same "verify with a real RPC
// call" contract — because a devnet is the same problem one size smaller, and
// two plans that do the same thing differently is how they drift.

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/valve-tech/valve-node-app/internal/catalog"
	"github.com/valve-tech/valve-node-app/internal/executor"
	"github.com/valve-tech/valve-node-app/internal/ops"
)

// devnetReadyTimeout/devnetPollInterval bound the wait for a freshly created
// devnet to be serving a live chain. Package vars, not consts, so tests can
// shrink them to avoid real sleeps — the same arrangement as
// gatewayReadyTimeout.
//
// A minute is generous against the ~10s observed between `docker run`
// returning and the first sealed block, and it is measured AFTER the pull:
// `docker run` blocks while the image comes down, so a cold first run does not
// spend this budget on the network.
var (
	devnetReadyTimeout = 60 * time.Second
	devnetPollInterval = 2 * time.Second
)

// devnetBlockNumberCall is the second half of the readiness probe. The chain-id
// half reuses gatewayChainIDCall verbatim from gateway.go — it is the same
// JSON-RPC call, and two spellings of one literal is how they end up
// disagreeing.
const devnetBlockNumberCall = `{"jsonrpc":"2.0","id":1,"method":"eth_blockNumber","params":[]}`

// PlanDevnet returns the ordered steps that put a devnet on a target:
// preflight, run.
//
// The config is validated here, at plan time, exactly as PlanGateway renders
// its config up front — an unusable devnet (a chain id reth cannot serve, a
// block time it cannot parse, two listeners on one host port) fails
// immediately instead of partway through RunAll with a container already
// removed.
func PlanDevnet(d catalog.DevnetConfig) ([]Step, error) {
	if err := d.Validate(); err != nil {
		return nil, fmt.Errorf("setup: %w", err)
	}
	p := &devnetPlan{dev: d}
	return []Step{p.preflightStep(), p.runStep()}, nil
}

// DevnetService is the ops lifecycle descriptor for d's container: the value
// ops.ServiceStatus / ContainerAction / WipeService are driven with, so a
// devnet can be started, stopped, restarted, wiped and read from the API
// without going anywhere near a setup plan.
//
// It lives HERE rather than in ops for the reason ops.ERPCService documents
// about its own missing Create hook: creating the container needs a resolved
// image platform and the rendered `docker run` argv, which this package owns
// and ops (which catalog and setup both sit on top of) cannot reach. Wiring
// Create to the run step's own runDocker — rather than re-deriving the argv —
// is what keeps "the devnet the wizard creates" and "the devnet a wipe
// re-creates" the same container, down to the flag order.
//
// frontedBy names the services that PROXY this devnet, normally the eRPC
// gateway. Passing it is what makes ops.WipeService bounce the gateway
// afterwards, which is not optional: eRPC's per-network head is monotonic, so
// a wiped chain leaves the gateway advertising a head that no longer exists
// (measured: chain back at 0x4, gateway still reporting 0x2c). See
// ops.WipeService for the full account.
//
// No Volumes are declared, and that is a fact about reth --dev rather than an
// omission: the chain lives in the container's own writable layer (see
// catalog.DevnetRunArgs, which mounts nothing), so `docker rm -f -v` takes
// every byte of it with the container.
func DevnetService(d catalog.DevnetConfig, frontedBy ...ops.DockerService) ops.DockerService {
	p := &devnetPlan{dev: d}
	return ops.DockerService{
		ID:            "devnet",
		ContainerName: d.Name(),
		FrontedBy:     frontedBy,
		Create: func(ctx context.Context, e executor.Executor) error {
			if err := d.Validate(); err != nil {
				return err
			}
			// nil State: this path has no event stream to report into. emit
			// is nil-State safe precisely so the same step body can run
			// under RunAll and under a plain lifecycle call.
			return p.runDocker(ctx, e, nil)
		},
	}
}

// devnetPlan is the state the two steps share. It exists for the same reason
// gatewayPlan does: Step's funcs are handed a *State carrying a
// catalog.WireConfig — a real node's config, which cannot describe a devnet —
// so the devnet config travels by closure instead.
//
// Unlike gatewayPlan it holds no mutable state at all. It needs no
// configPending flag because there is no config file whose rewrite could be
// missed: a devnet's settings live in the container's own command line, and
// the run step's Verify reads them back off the running container (see
// commandDrift), which detects a settings change directly rather than
// inferring one.
type devnetPlan struct {
	dev catalog.DevnetConfig
}

// ---------------------------------------------------------------------
// preflight
// ---------------------------------------------------------------------

// preflightStep, like the node and gateway plans', has nothing to fix and so
// has no Run: RunAll's Verify pre-check IS the check, and its failure is
// terminal.
func (p *devnetPlan) preflightStep() Step {
	return Step{
		ID:    "preflight",
		Title: "Preflight checks (devnet)",
		Verify: func(ctx context.Context, e executor.Executor, st *State) error {
			return p.preflight(ctx, e)
		},
	}
}

// preflight checks only what a devnet actually needs: an engine that can run
// Linux containers, and two free ports.
//
// Deliberately absent, and each absence is a fact about devnets rather than an
// oversight: no OS/root check (the container backend exists precisely so a
// macOS or Windows desktop qualifies), no disk sizing (the chain lives in the
// container's writable layer and starts empty), no service account, no JWT.
func (p *devnetPlan) preflight(ctx context.Context, e executor.Executor) error {
	info, err := ops.ProbeDocker(ctx, e)
	if err != nil {
		// Wrapped, not reworded: ops.ErrDockerAbsent must survive to the caller
		// so the UI can offer an install prompt instead of a generic failure,
		// and the typed error already carries the hint.
		return fmt.Errorf("preflight: %w", err)
	}
	if info.WindowsContainers() {
		return fmt.Errorf("preflight: this docker engine is in Windows-container mode, and the reth image is a Linux image — switch Docker to Linux containers and retry")
	}
	if !info.DaemonReachable {
		return fmt.Errorf("preflight: the docker CLI is installed but no engine answered — start Docker Desktop / OrbStack / colima (or `systemctl start docker`) and retry: %s", info.DaemonError)
	}
	return p.checkPortsFree(ctx, e)
}

// checkPortsFree fails when something other than our own devnet already holds
// the JSON-RPC or WebSocket port.
//
// Both ports are checked, not just HTTP: 8546 is the default WebSocket port
// for every EVM client, so on a machine that already runs a node it is at
// least as likely to be taken as 8545 — and docker binds both mappings at
// creation, so a collision on either is a container that never starts.
//
// The two leniencies are gateway.go's, for gateway.go's reasons: a target with
// none of ss/netstat/lsof reads as free (docker itself fails loudly and
// specifically on a real collision, whereas a false "busy" terminally blocks a
// preflight that has no Run to fix it), and our own already-running devnet is
// exempt, which is what makes re-running this plan against a live devnet work
// at all.
func (p *devnetPlan) checkPortsFree(ctx context.Context, e executor.Executor) error {
	if p.devnetHoldsPorts(ctx, e) {
		return nil
	}
	for _, port := range []int{p.dev.HTTP(), p.dev.WS()} {
		res, err := e.Run(ctx, fmt.Sprintf(listenerProbe, port), nil)
		if err != nil {
			return fmt.Errorf("preflight: probe listeners on port %d: %w", port, err)
		}
		if res.ExitCode == 0 && strings.TrimSpace(res.Stdout) != "" {
			return fmt.Errorf("preflight: port %d is already in use by something other than valve-node-app's devnet:\n%s", port, strings.TrimSpace(res.Stdout))
		}
	}
	return nil
}

// devnetHoldsPorts reports whether OUR devnet container is the thing already on
// the ports. Any failure reading that answer is reported as "no": the caller
// then runs the port probe, which produces a precise, evidence-carrying error
// rather than this function inventing one.
func (p *devnetPlan) devnetHoldsPorts(ctx context.Context, e executor.Executor) bool {
	running, err := ops.ContainerRunning(ctx, e, p.dev.Name())
	return err == nil && running
}

// ---------------------------------------------------------------------
// run
// ---------------------------------------------------------------------

func (p *devnetPlan) runStep() Step {
	return Step{
		ID:    "run",
		Title: "Start devnet container (chain " + strconv.Itoa(p.dev.ChainIDOrDefault()) + ")",
		Run: func(ctx context.Context, e executor.Executor, st *State) error {
			if err := p.runDocker(ctx, e, st); err != nil {
				return err
			}
			return p.waitReady(ctx, e)
		},
		Verify: func(ctx context.Context, e executor.Executor, st *State) error {
			if err := p.commandDrift(ctx, e); err != nil {
				return err
			}
			return p.devnetCheck(ctx, e)
		},
	}
}

func (p *devnetPlan) runDocker(ctx context.Context, e executor.Executor, st *State) error {
	info, err := ops.ProbeDocker(ctx, e)
	if err != nil {
		return fmt.Errorf("run: %w", err)
	}

	// No image build, unlike the gateway. The gateway is built on the target
	// because upstream eRPC lacks WebSocket support and the valve fork
	// publishes no image; reth's own image is published multi-arch
	// (linux/amd64 + linux/arm64, verified), so `docker run` pulling it is
	// both correct and native. Resolving the platform still matters for the
	// Rosetta case ops.EnginePlatform documents — a CLI that misreports its
	// architecture makes docker look up the wrong variant of the manifest.
	d := p.dev
	d.Platform = ops.EnginePlatform(ctx, e, info)

	// Remove before run, always. A container's published ports, image and
	// command are all fixed at creation, so applying a changed setting is
	// necessarily remove + run, never a restart — and an absent container makes
	// this a no-op rather than an error. Discarding the old chain is not
	// collateral damage here, it is the point: a devnet is meant to be thrown
	// away.
	if err := ops.RemoveContainer(ctx, e, d.Name()); err != nil {
		return fmt.Errorf("run: %w", err)
	}

	res, err := ops.DockerRun(ctx, e, catalog.DevnetRunArgs(d)...)
	if err != nil {
		return fmt.Errorf("run: %w", err)
	}
	if res.ExitCode != 0 {
		return fmt.Errorf("run: docker run failed (exit %d): %s", res.ExitCode, strings.TrimSpace(res.Stderr))
	}
	_ = emit(ctx, st, Event{StepID: "run", Line: strings.TrimSpace(res.Stdout)})
	return nil
}

// waitReady polls until the devnet is serving a live chain, because "the
// container started" and "the chain is producing blocks" are seconds apart —
// reth has to open its listeners and seal a first block — and a Verify run
// immediately after Run would otherwise fail a devnet that is merely still
// starting.
func (p *devnetPlan) waitReady(ctx context.Context, e executor.Executor) error {
	deadline := time.Now().Add(devnetReadyTimeout)
	for {
		err := p.devnetCheck(ctx, e)
		if err == nil {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("run: devnet did not answer within %s: %w", devnetReadyTimeout, err)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(devnetPollInterval):
		}
	}
}

// devnetCheck is the devnet's equivalent of the node plan's handshake: real RPC
// calls, end to end, through the port a caller will actually use.
//
// It asks two things, and both are load-bearing:
//
//   - eth_chainId, because its answer is verifiable. Something else listening
//     on 8545 — an already-running node, a previous devnet on a different
//     chain — answers this call perfectly happily, and only the id tells the
//     two apart.
//   - eth_blockNumber > 0, because a devnet that never seals a block is
//     useless in exactly the way a devnet is supposed to be useful. reth
//     answers RPC before the first block exists, so a chain-id check alone
//     would report a devnet ready while the sealer was silently misconfigured.
func (p *devnetPlan) devnetCheck(ctx context.Context, e executor.Executor) error {
	want := p.dev.ChainIDOrDefault()

	raw, err := p.rpcCall(ctx, e, gatewayChainIDCall)
	if err != nil {
		return err
	}
	got, err := parseHexChainID(raw)
	if err != nil {
		return fmt.Errorf("devnet: %s returned an unreadable eth_chainId result %q: %w", p.dev.HTTPEndpoint(), raw, err)
	}
	if got != want {
		return fmt.Errorf("devnet: %s is serving chain %d, not the devnet's chain %d — something else is on that port", p.dev.HTTPEndpoint(), got, want)
	}

	raw, err = p.rpcCall(ctx, e, devnetBlockNumberCall)
	if err != nil {
		return err
	}
	height, err := parseHexQuantity(raw)
	if err != nil {
		return fmt.Errorf("devnet: %s returned an unreadable eth_blockNumber result %q: %w", p.dev.HTTPEndpoint(), raw, err)
	}
	if height == 0 {
		return fmt.Errorf("devnet: %s is at block 0 — the chain is up but has not sealed a block yet (block time %s)", p.dev.HTTPEndpoint(), p.dev.BlockTimeOrDefault())
	}
	return nil
}

// rpcCall posts one JSON-RPC body to the devnet and returns its result field.
//
// curl over the executor rather than a Go HTTP client, matching
// gatewayCheck: the target may be an SSH host, and there the only thing that
// can reach a loopback-bound port is a process ON that host.
func (p *devnetPlan) rpcCall(ctx context.Context, e executor.Executor, body string) (string, error) {
	url := p.dev.HTTPEndpoint()
	cmd := fmt.Sprintf("curl -s -X POST -H 'Content-Type: application/json' --data %s %s", shQuote(body), shQuote(url))
	res, err := e.Run(ctx, cmd, nil)
	if err != nil {
		return "", fmt.Errorf("devnet: rpc probe: %w", err)
	}
	if res.ExitCode != 0 {
		return "", fmt.Errorf("devnet: rpc at %s failed (curl exit %d): %s", url, res.ExitCode, strings.TrimSpace(res.Stderr))
	}

	var answer struct {
		Result string `json:"result"`
		Error  *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	raw := strings.TrimSpace(res.Stdout)
	if err := json.Unmarshal([]byte(raw), &answer); err != nil {
		// Not JSON at all is the normal shape of "nothing is listening yet"
		// (empty body) — report the raw answer, which is what tells that apart
		// from an HTML error page served by something else on the port.
		return "", fmt.Errorf("devnet: %s did not answer with JSON: %q", url, raw)
	}
	if answer.Error != nil {
		return "", fmt.Errorf("devnet: %s answered with an error: %s", url, answer.Error.Message)
	}
	return answer.Result, nil
}

// parseHexQuantity reads a JSON-RPC quantity ("0x539"). gateway.go's
// parseHexChainID is that same parse under a chain-specific name; this alias
// exists so a block height is not read through a function called "chain id".
func parseHexQuantity(hex string) (int, error) { return parseHexChainID(hex) }

// commandDrift fails when a devnet is running, but running on settings other
// than the ones being asked for.
//
// WHY it has to exist: RunAll skips any step whose Verify already holds, and a
// devnet started with a two-second block time answers eth_chainId and
// eth_blockNumber exactly like one started with a five-second block time.
// Without this, changing a setting would leave the old container serving while
// setup reported success. It is this plan's counterpart to gatewayPlan's
// configPending — but where a gateway has a config file to compare, a devnet's
// settings exist only as the container's command line, so that is what gets
// compared, read back off the live container with `docker inspect`.
//
// A reading that cannot be obtained — no such container, an engine that
// declines, output that does not parse — reports NO drift rather than drift.
// That is deliberate in both directions: failing closed would re-create the
// container on every single verify, destroying both idempotence and the
// operator's chain, while the devnetCheck that follows still catches the case
// that actually matters (nothing serving the right chain on that port).
func (p *devnetPlan) commandDrift(ctx context.Context, e executor.Executor) error {
	res, err := ops.DockerRun(ctx, e, "inspect", "-f", "{{json .Config.Cmd}}", p.dev.Name())
	if err != nil || res.ExitCode != 0 {
		return nil
	}
	var running []string
	if err := json.Unmarshal([]byte(strings.TrimSpace(res.Stdout)), &running); err != nil || len(running) == 0 {
		return nil
	}
	want := catalog.DevnetCommand(p.dev)
	if slices.Equal(running, want) {
		return nil
	}
	return fmt.Errorf("run: the running devnet was started with different settings (%s), not the requested ones (%s) — it has to be re-created to pick them up",
		strings.Join(running, " "), strings.Join(want, " "))
}
