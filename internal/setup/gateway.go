package setup

// Gateway provisioning: getting ONE eRPC instance up, fronting however many
// chains the operator configured.
//
// WHY this is a separate plan rather than a mode of Plan(): the nine-step
// node chain exists because a chain client is a big, stateful, Linux-only
// install — it needs its own service account, a toolchain to build with,
// gigabytes of dataset, and a two-process handshake before it is worth
// anything. A gateway is a config file and a listener. It has no dataset, no
// account of its own, nothing to build, and no peer to shake hands with, so
// six of those nine steps have nothing to do. Expressing it as a short,
// honest three-step plan is what lets a gateway run on a macOS or Windows
// desktop (via the docker backend) that Plan's preflight can never accept.
//
// The steps are: preflight (can this target host a gateway, and is the port
// free), config (render erpc.yaml and put it on the target), run (start it
// and confirm it actually answers RPC).

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/valve-tech/valve-node-app/internal/catalog"
	"github.com/valve-tech/valve-node-app/internal/executor"
	"github.com/valve-tech/valve-node-app/internal/ops"
)

// Backends a gateway can be hosted on.
//
// Docker is the cross-platform one and the reason the container path exists
// at all (internal/ops/docker.go): it is the only way to run a gateway on
// the operator's own macOS/Windows machine. Systemd is for a Linux target
// that is already hosting node services — the gateway then joins them as
// another hardened, de-rooted unit.
const (
	BackendDocker  = "docker"
	BackendSystemd = "systemd"
)

// Unit naming for the systemd backend, matching execUnitName/beaconUnitName.
//
// erpcUnitName is the DEFAULT gateway's unit; every other gateway gets
// "valve-node-app-erpc-<id>.service". The same reasoning as
// ops.ERPCContainerNameFor: several gateways can be placed on one machine,
// and two units cannot share a name.
const (
	erpcUnitName = "valve-node-app-erpc.service"
	erpcUnitDir  = "/etc/systemd/system/"
	erpcUnitPath = erpcUnitDir + erpcUnitName
)

// erpcUnitNameFor derives a gateway's unit name from its id, mapping the
// default id back to the historical name so an existing unit is restarted
// rather than shadowed by a second one.
func erpcUnitNameFor(gatewayID string) string {
	id := strings.TrimSpace(gatewayID)
	if id == "" || id == ops.DefaultGatewayID {
		return erpcUnitName
	}
	return "valve-node-app-erpc-" + sanitizeGatewayID(id) + ".service"
}

// sanitizeGatewayID keeps a gateway id safe to embed in a unit name, a file
// name and a shell command. It mirrors ops.ERPCContainerNameFor's own
// sanitization; the id is validated at the API boundary too, and this is the
// lock at the point of use.
func sanitizeGatewayID(s string) string {
	out := make([]rune, 0, len(s))
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9',
			r == '_', r == '.', r == '-':
			out = append(out, r)
		default:
			out = append(out, '-')
		}
	}
	return string(out)
}

// gatewayDataDir is the systemd backend's data root. A gateway has no
// dataset, so this exists only to give the unit a ReadWritePaths= and to
// hold erpc.yaml. It is deliberately NOT chain-scoped (unlike a node's
// /var/lib/valve-node-app/<chainId>): one gateway serves every chain.
const gatewayDataDir = "/var/lib/valve-node-app"

// gatewayHomeDir is the per-user directory the docker backend keeps
// erpc.yaml in, under the target's $HOME.
const gatewayHomeDir = ".valve-node-app"

// gatewayReadyTimeout/gatewayPollInterval bound the wait for a freshly
// started gateway to answer RPC. Package vars, not consts, so tests can
// shrink them to avoid real sleeps — same arrangement as handshakeTimeout.
var (
	gatewayReadyTimeout = 60 * time.Second
	gatewayPollInterval = 2 * time.Second

	// caddyRootTimeout/caddyRootPollInterval bound the wait for Caddy to have
	// written its internal CA. `docker run` returns before Caddy has finished
	// starting, so the first read legitimately finds nothing; a few seconds is
	// generous against the observed sub-second startup.
	caddyRootTimeout      = 15 * time.Second
	caddyRootPollInterval = 500 * time.Millisecond
)

// gatewayChainIDCall is the probe body the run step's check posts.
const gatewayChainIDCall = `{"jsonrpc":"2.0","id":1,"method":"eth_chainId","params":[]}`

// PlanGateway returns the ordered steps that put a gateway on a target:
// preflight, config, run.
//
// The config is validated here, at plan time, by rendering it — exactly as
// Plan calls RenderUnits up front — so an unusable gateway (no networks, a
// bad chain id, an endpoint with no scheme) fails immediately instead of
// partway through RunAll with a container already removed.
//
// Note for the systemd backend: the unit runs as catalog.ServiceUser, and
// this plan does not create that account or install the erpc binary. Both
// are the node plan's job. A systemd gateway is therefore an addition to a
// box valve-node-app already provisioned; a gateway on a bare machine is
// what the docker backend is for.
// gatewayID identifies WHICH gateway is being provisioned: its container
// name, unit name and config path are all derived from it, so two gateways
// on one machine never contend for the same name or file.
func PlanGateway(gatewayID string, g catalog.GatewayConfig, backend string) ([]Step, error) {
	switch backend {
	case BackendDocker, BackendSystemd:
	default:
		return nil, fmt.Errorf("setup: unknown gateway backend %q (want %q or %q)", backend, BackendDocker, BackendSystemd)
	}
	if _, err := catalog.RenderGatewayConfig(g); err != nil {
		return nil, fmt.Errorf("setup: %w", err)
	}
	// A TLS front is a CONTAINER. A systemd gateway is not, and standing a
	// docker container in front of a unit-hosted process would mean the
	// gateway's two halves had different lifecycles, different failure modes
	// and different places to look — for a target that, being Linux with root,
	// already has better tools for this than we would be adding.
	if g.Fronted() && backend != BackendDocker {
		return nil, fmt.Errorf("setup: HTTPS for a gateway is provided by a Caddy container, so it needs the %q backend — this gateway is on %q. Put a reverse proxy in front of it yourself, or move it to the container backend.", BackendDocker, backend)
	}

	p := &gatewayPlan{id: gatewayID, gw: g, backend: backend}
	return []Step{p.preflightStep(), p.configStep(), p.runStep()}, nil
}

// GatewayService is the ops lifecycle descriptor for the docker-hosted eRPC
// gateway: ops.ERPCService() plus the Create hook that package deliberately
// leaves nil, because creating the container needs the rendered erpc.yaml
// path, the resolved platform and the built image — all of which this plan
// owns.
//
// Create writes the config and then runs the container, in that order and
// through the plan's own step bodies rather than a second copy of them. Both
// halves are necessary: a gateway container started against a missing
// erpc.yaml comes up and then serves nothing, which reads as "the gateway is
// running" everywhere a container state is displayed.
//
// The systemd backend is deliberately not offered here. A unit-hosted gateway
// is not a container at all, so its lifecycle belongs to ops.ServiceAction
// (systemctl), not to this file's docker verbs.
func GatewayService(gatewayID string, g catalog.GatewayConfig) ops.DockerService {
	p := &gatewayPlan{id: gatewayID, gw: g, backend: BackendDocker}
	s := ops.ERPCServiceFor(gatewayID)
	s.Create = func(ctx context.Context, e executor.Executor) error {
		// Rendering up front, exactly as PlanGateway does, so an unusable
		// config (no networks, an endpoint with no scheme) fails before the
		// image is built or a container is created.
		if _, err := catalog.RenderGatewayConfig(g); err != nil {
			return fmt.Errorf("gateway: %w", err)
		}
		// nil State: this path has no event stream to report into. emit and
		// streamOpts are both nil-State safe precisely so one step body can
		// run under RunAll and under a plain lifecycle call. runDocker
		// resolves the platform and builds the image itself when needed.
		if err := p.configStep().Run(ctx, e, nil); err != nil {
			return err
		}
		return p.runDocker(ctx, e, nil)
	}
	return s
}

// gatewayPlan is the state the three steps share. It exists because Step's
// funcs are handed a *State carrying a catalog.WireConfig — a node's config,
// which cannot describe a gateway — so the gateway config travels by closure
// instead, and this is what those closures close over.
type gatewayPlan struct {
	// id is the gateway's stable id. Everything nameable — container, unit,
	// config file — is derived from it, so a second gateway on the same
	// machine cannot overwrite the first one's file or fight it for a name.
	id      string
	gw      catalog.GatewayConfig
	backend string

	// mu guards the two fields below. RunAll is sequential, so the lock is
	// not contended; it is here so a caller driving steps concurrently
	// cannot race on them.
	mu sync.Mutex
	// dockerConfigPath memoizes the path resolved lazily on the target (see
	// configPath) so every step that needs it agrees on one answer.
	dockerConfigPath string
	// tls memoizes the resolved TLS front for the duration of ONE plan run.
	//
	// Memoizing is not an optimization, it is a correctness requirement: the
	// config step's Verify compares the Caddyfile it would render against the
	// one on disk, and resolveTLSFront reads a certificate whose validity is a
	// function of the clock. Re-resolving between the write and the check could
	// have a certificate expire mid-run, rendering a different file and looping
	// forever on a step that had already succeeded.
	tls       *tlsFront
	tlsSolved bool
	// configPending is set by the config step whenever it (re)writes
	// erpc.yaml and cleared by the run step once the gateway has actually
	// been restarted onto it.
	//
	// WHY it has to exist: RunAll skips any step whose Verify already holds,
	// and a gateway serving the PREVIOUS config answers eth_chainId exactly
	// like one serving the new config. Without this, editing an upstream
	// would rewrite the file and then skip the restart entirely — setup
	// would report success while the old config kept serving. It is the
	// cross-step form of the change detection wireStep does within one step.
	configPending bool
}

func (p *gatewayPlan) markConfigPending() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.configPending = true
}

func (p *gatewayPlan) clearConfigPending() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.configPending = false
}

func (p *gatewayPlan) configIsPending() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.configPending
}

// ---------------------------------------------------------------------
// paths
// ---------------------------------------------------------------------

// configPath resolves where erpc.yaml lives ON THE TARGET.
//
// The systemd answer is fixed and is derived from the same WireConfig the
// unit is rendered from, so the file's location and the unit's --config flag
// cannot drift apart.
//
// The docker answer is under the target's $HOME, and that is not arbitrary:
// on a macOS/Windows desktop — the whole reason the docker backend exists —
// /var/lib is not writable without root, and worse, it is not shared into
// the engine's VM (Docker Desktop shares /Users, /tmp and /private, not
// /var/lib). A bind mount whose host path the VM cannot see does not fail;
// it silently mounts an empty directory, and the gateway starts with no
// config at all. $HOME is inside a shared path on every desktop engine.
func (p *gatewayPlan) configPath(ctx context.Context, e executor.Executor) (string, error) {
	if p.backend == BackendSystemd {
		return path.Join(gatewayDataDir, p.configFileName()), nil
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	if p.dockerConfigPath != "" {
		return p.dockerConfigPath, nil
	}
	res, err := e.Run(ctx, `printf '%s\n' "$HOME"`, nil)
	if err != nil {
		return "", fmt.Errorf("gateway: resolve $HOME on the target: %w", err)
	}
	home := strings.TrimSpace(res.Stdout)
	if res.ExitCode != 0 || home == "" {
		return "", fmt.Errorf("gateway: could not resolve $HOME on the target (exit %d): the docker backend keeps erpc.yaml there because it must be a path the engine can bind-mount", res.ExitCode)
	}
	p.dockerConfigPath = path.Join(home, gatewayHomeDir, p.configFileName())
	return p.dockerConfigPath, nil
}

// caddyfilePath / rootCAPath are the TLS front's two files on the target,
// kept beside erpc.yaml so one directory holds everything a gateway owns.
//
// rootCAPath is the file an operator installs in their trust store. It has to
// be on the HOST, not left inside the container, or "install this root" means
// "learn docker cp first".
func (p *gatewayPlan) caddyfilePath(ctx context.Context, e executor.Executor) (string, error) {
	return p.siblingPath(ctx, e, p.tlsFileName("Caddyfile", ""))
}

func (p *gatewayPlan) rootCAPath(ctx context.Context, e executor.Executor) (string, error) {
	return p.siblingPath(ctx, e, p.tlsFileName("caddy-root", ".crt"))
}

// siblingPath puts name in the same directory erpc.yaml lives in, whichever
// backend resolved it, so the two can never end up on different filesystems —
// which on a desktop engine is the difference between a bind mount the VM can
// see and one it silently mounts as an empty directory.
func (p *gatewayPlan) siblingPath(ctx context.Context, e executor.Executor, name string) (string, error) {
	cfg, err := p.configPath(ctx, e)
	if err != nil {
		return "", err
	}
	return path.Join(path.Dir(cfg), name), nil
}

// tlsFileName scopes a TLS file to this gateway, exactly as configFileName
// scopes erpc.yaml: two gateways on one machine must not share a Caddyfile,
// because each provision would overwrite the other's hostname.
func (p *gatewayPlan) tlsFileName(base, ext string) string {
	id := strings.TrimSpace(p.id)
	if id == "" || id == ops.DefaultGatewayID {
		return base + ext
	}
	return base + "-" + sanitizeGatewayID(id) + ext
}

// configFileName is this gateway's erpc.yaml, per gateway id. The default id
// keeps the bare "erpc.yaml" so an existing install's file (and the
// bind-mount path baked into its running container) is the one that gets
// rewritten. Anything else is "erpc-<id>.yaml" — two gateways sharing one
// file would each rewrite the other's config on every provision, and the
// second one's container would be started against the first one's chains.
func (p *gatewayPlan) configFileName() string {
	id := strings.TrimSpace(p.id)
	if id == "" || id == ops.DefaultGatewayID {
		return "erpc.yaml"
	}
	return "erpc-" + sanitizeGatewayID(id) + ".yaml"
}

// unitWire is the minimal catalog.WireConfig catalog.RenderERPCUnit needs.
// It reads exactly three things from it — DataDir (for ReadWritePaths= and
// the --config path) and the eRPC bind/port (to decide whether the unit
// needs CAP_NET_BIND_SERVICE for a privileged port) — and none of the
// node-shaped fields, which is why a gateway can borrow the node's unit
// template without pretending to be a node.
func (p *gatewayPlan) unitWire() catalog.WireConfig {
	return catalog.WireConfig{
		DataDir:      gatewayDataDir,
		ERPCBindAddr: p.gw.Bind(),
		ERPCPort:     p.gw.HTTP(),
	}
}

// ---------------------------------------------------------------------
// preflight
// ---------------------------------------------------------------------

// preflightStep, like the node plan's, has nothing to fix and so has no Run:
// RunAll's Verify pre-check IS the check, and its failure is terminal.
func (p *gatewayPlan) preflightStep() Step {
	return Step{
		ID:    "preflight",
		Title: "Preflight checks (" + p.backend + " gateway)",
		Verify: func(ctx context.Context, e executor.Executor, st *State) error {
			return p.preflight(ctx, e)
		},
	}
}

func (p *gatewayPlan) preflight(ctx context.Context, e executor.Executor) error {
	switch p.backend {
	case BackendDocker:
		info, err := ops.ProbeDocker(ctx, e)
		if err != nil {
			// Wrapped, not reworded: ops.ErrDockerAbsent must survive to the
			// caller so the UI can offer an install prompt instead of a
			// generic failure, and the typed error already carries the hint.
			return fmt.Errorf("preflight: %w", err)
		}
		if info.WindowsContainers() {
			return fmt.Errorf("preflight: this docker engine is in Windows-container mode, and the eRPC image is a Linux image — switch Docker to Linux containers and retry")
		}
		if !info.DaemonReachable {
			return fmt.Errorf("preflight: the docker CLI is installed but no engine answered — start Docker Desktop / OrbStack / colima (or `systemctl start docker`) and retry: %s", info.DaemonError)
		}
	case BackendSystemd:
		if err := requireLinuxRoot(ctx, e); err != nil {
			return err
		}
	}
	return p.checkPortFree(ctx, e)
}

// requireLinuxRoot performs the two host expectations a systemd install has.
//
// steps.go's preflightCheck opens with the same two probes, but a gateway
// cannot call it: the rest of that function sizes a chain dataset via
// catalog.ExpectedBytes and scans the exec/beacon/engine ports, and neither
// applies here — a gateway stores nothing, and it fronts chains this app has
// no node support for at all, so ExpectedBytes would reject the config
// outright. The overlap is these two commands, deliberately left duplicated
// rather than refactoring a function two plans depend on.
func requireLinuxRoot(ctx context.Context, e executor.Executor) error {
	res, err := e.Run(ctx, "uname", nil)
	if err != nil {
		return fmt.Errorf("preflight: uname: %w", err)
	}
	if osName := strings.TrimSpace(res.Stdout); osName != "Linux" {
		return fmt.Errorf("preflight: a systemd gateway targets a Linux host, but this one reports %q — run the gateway on the %q backend instead, or point valve-node-app at a Linux target over SSH.", osName, BackendDocker)
	}

	res, err = e.Run(ctx, "id -u", nil)
	if err != nil {
		return fmt.Errorf("preflight: id -u: %w", err)
	}
	if uid := strings.TrimSpace(res.Stdout); uid != "0" {
		return fmt.Errorf("preflight: installing a systemd unit needs root on the target (SSH as root, or run valve-node-app as root in local mode); id -u reported %q.", uid)
	}
	return nil
}

// listenerProbe lists the target's listening TCP sockets using whichever
// tool it actually has. `ss` is Linux-only and `netstat -an` is what a macOS
// desktop has; `lsof` is last because without root it only sees the invoking
// user's own sockets, which would make a busy port look free. Each is tried
// in turn — a missing binary exits non-zero and falls through.
//
// LISTEN filtering matters: `netstat -an` prints established connections
// too, and a connection whose remote port happens to match would otherwise
// read as a busy local port and terminally fail a preflight that has no Run
// to fix it.
const listenerProbe = `{ ss -ltn 2>/dev/null || netstat -an 2>/dev/null || lsof -nP -iTCP -sTCP:LISTEN 2>/dev/null; } | grep -Ei '[:.]%d([^0-9]|$)' | grep -i listen`

// checkPortFree fails when something other than our own gateway is already
// listening on the port the gateway is about to publish.
//
// Two deliberate leniencies. A target with none of the three tools yields no
// output and reads as free: docker and systemd both fail loudly and
// specifically on a port collision anyway ("port is already allocated"), so
// the cost of guessing wrong that way is a worse error message, whereas a
// false "busy" blocks setup outright. And our own already-running gateway is
// exempt, which is what makes re-running this plan against a live gateway
// (a config change, a resume) work at all.
func (p *gatewayPlan) checkPortFree(ctx context.Context, e executor.Executor) error {
	if p.gatewayHoldsPort(ctx, e) {
		return nil
	}
	// A fronted gateway publishes no host port for eRPC at all — Caddy is the
	// only front door — so the port to check is the TLS one. Checking the eRPC
	// port there would fail a perfectly good setup whenever something unrelated
	// happened to hold 4000, for a port nothing is going to bind.
	port := p.gw.HTTP()
	what := "gateway"
	if p.fronted() {
		port = p.gw.TLS.HTTPS()
		what = "gateway's HTTPS front"
	}

	res, err := e.Run(ctx, fmt.Sprintf(listenerProbe, port), nil)
	if err != nil {
		return fmt.Errorf("preflight: probe listeners on port %d: %w", port, err)
	}
	if res.ExitCode == 0 && strings.TrimSpace(res.Stdout) != "" {
		return fmt.Errorf("preflight: port %d is already in use by something other than valve-node-app's %s:\n%s", port, what, strings.TrimSpace(res.Stdout))
	}
	return nil
}

// gatewayHoldsPort reports whether OUR gateway is the thing already on the
// port. Any failure reading that answer is reported as "no": the caller then
// runs the port check, which produces a precise, evidence-carrying error
// rather than this function inventing one.
func (p *gatewayPlan) gatewayHoldsPort(ctx context.Context, e executor.Executor) bool {
	switch p.backend {
	case BackendDocker:
		running, err := ops.ContainerRunning(ctx, e, p.containerName())
		return err == nil && running
	case BackendSystemd:
		res, err := e.Run(ctx, "systemctl is-active "+p.unitName(), nil)
		return err == nil && strings.TrimSpace(res.Stdout) == "active"
	}
	return false
}

// containerName / caddyName / unitName are this gateway's names on the target.
func (p *gatewayPlan) containerName() string { return ops.ERPCContainerNameFor(p.id) }
func (p *gatewayPlan) caddyName() string     { return ops.CaddyContainerNameFor(p.id) }
func (p *gatewayPlan) unitName() string      { return erpcUnitNameFor(p.id) }

// front resolves this gateway's TLS front once per plan run, or returns nil
// when there is none. See gatewayPlan.tls for why the answer is memoized.
//
// The upstream is the eRPC CONTAINER NAME and its IN-CONTAINER port, never a
// published host port: both containers are on ops.NetworkName, where docker's
// embedded DNS resolves the name, and that is precisely what allows eRPC to
// publish nothing at all.
func (p *gatewayPlan) front(ctx context.Context, e executor.Executor) (*tlsFront, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.tlsSolved {
		return p.tls, nil
	}
	f, err := resolveTLSFront(ctx, e, p.gw.TLS, p.containerName(), ops.ERPCContainerPort)
	if err != nil {
		return nil, err
	}
	p.tls, p.tlsSolved = f, true
	return f, nil
}

// fronted reports whether this gateway HAS a TLS front, without resolving one.
// Cheap and I/O-free, for the many places that only need the shape of the plan.
func (p *gatewayPlan) fronted() bool {
	return p.backend == BackendDocker && p.gw.Fronted()
}

// ---------------------------------------------------------------------
// config
// ---------------------------------------------------------------------

func (p *gatewayPlan) configStep() Step {
	return Step{
		ID:    "config",
		Title: "Write gateway config (erpc.yaml)",
		Run: func(ctx context.Context, e executor.Executor, st *State) error {
			cfg, err := p.renderConfig()
			if err != nil {
				return err
			}
			dest, err := p.configPath(ctx, e)
			if err != nil {
				return err
			}
			// 0640 on the systemd backend, then a chgrp so the unprivileged
			// service user can still read it: upstream endpoints routinely
			// carry provider API keys, so this file is a secret in the same
			// sense the JWT is. The container backend keeps 0644 — the file
			// is bind-mounted read-only into an image whose eRPC process is
			// not the host user, so narrowing it would only lock the gateway
			// out of its own config.
			mode := writeModeGatewayConfig(p.backend)
			if err := e.WriteFile(ctx, dest, []byte(cfg), mode); err != nil {
				return fmt.Errorf("config: write %s: %w", dest, err)
			}
			if p.backend == BackendSystemd {
				res, err := e.Run(ctx, fmt.Sprintf("chgrp %s %s", catalog.ServiceGroup, shQuote(dest)), streamOpts(ctx, st, "config"))
				if err != nil {
					return fmt.Errorf("config: chgrp %s: %w", dest, err)
				}
				if res.ExitCode != 0 {
					return fmt.Errorf("config: could not give group %s read access to %s (exit %d): %s — the gateway unit runs as that account, which the node setup creates",
						catalog.ServiceGroup, dest, res.ExitCode, strings.TrimSpace(res.Stderr))
				}
			}
			if err := p.writeCaddyfile(ctx, e, st); err != nil {
				return err
			}
			p.markConfigPending()
			return nil
		},
		Verify: func(ctx context.Context, e executor.Executor, st *State) error {
			want, err := p.renderConfig()
			if err != nil {
				return err
			}
			dest, err := p.configPath(ctx, e)
			if err != nil {
				return err
			}
			if err := p.verifyCaddyfile(ctx, e); err != nil {
				return err
			}
			got, err := e.ReadFile(ctx, dest)
			if err != nil {
				return fmt.Errorf("config: %s is not written yet: %w", dest, err)
			}
			// Byte comparison, for the same reason wireStep does it: a file
			// that merely EXISTS may be a leftover from a previous run with
			// different chains or upstreams. Treating that as unverified is
			// what drives Run to rewrite it and the run step to restart the
			// gateway, instead of reporting success while the old config
			// keeps serving.
			if string(got) != want {
				return fmt.Errorf("config: %s does not match the desired gateway config (chains or upstreams changed since it was written)", dest)
			}
			return nil
		},
	}
}

// writeCaddyfile renders and writes the TLS front's configuration, and reports
// an automatic certificate fallback into the event stream when one happened.
//
// The fallback line is emitted at WRITE time and not only at read time because
// this is the moment the operator is watching: a provisioning run that quietly
// swapped their certificate for a self-signed one, and only said so on a screen
// they might visit later, would be the silent behaviour the fallback exists to
// avoid — just relocated.
func (p *gatewayPlan) writeCaddyfile(ctx context.Context, e executor.Executor, st *State) error {
	if !p.fronted() {
		return nil
	}
	front, err := p.front(ctx, e)
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	rendered, err := catalog.RenderCaddyfile(front.Caddy)
	if err != nil {
		return fmt.Errorf("config: render Caddyfile: %w", err)
	}
	dest, err := p.caddyfilePath(ctx, e)
	if err != nil {
		return err
	}
	// 0644, matching the container backend's erpc.yaml and for the same
	// reason: it is bind-mounted read-only into an image whose process is not
	// the host user, so narrowing it only locks Caddy out of its own config.
	if err := e.WriteFile(ctx, dest, []byte(rendered), 0644); err != nil {
		return fmt.Errorf("config: write %s: %w", dest, err)
	}
	if front.Fallback != "" {
		_ = emit(ctx, st, Event{StepID: "config", Line: "certificate fallback: " + front.Fallback})
	}
	return nil
}

// verifyCaddyfile is the Caddyfile's half of the config step's byte
// comparison. A file that merely exists may be from a run with a different
// hostname or a different certificate decision, and treating that as verified
// is what would leave the old TLS front serving while setup reported success.
func (p *gatewayPlan) verifyCaddyfile(ctx context.Context, e executor.Executor) error {
	if !p.fronted() {
		return nil
	}
	front, err := p.front(ctx, e)
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	want, err := catalog.RenderCaddyfile(front.Caddy)
	if err != nil {
		return fmt.Errorf("config: render Caddyfile: %w", err)
	}
	dest, err := p.caddyfilePath(ctx, e)
	if err != nil {
		return err
	}
	got, err := e.ReadFile(ctx, dest)
	if err != nil {
		return fmt.Errorf("config: %s is not written yet: %w", dest, err)
	}
	if string(got) != want {
		return fmt.Errorf("config: %s does not match the desired TLS configuration (the hostname, the certificate source, or which certificate is usable has changed since it was written)", dest)
	}
	return nil
}

// writeModeGatewayConfig is split out only so the two modes are stated once.
func writeModeGatewayConfig(backend string) fs.FileMode {
	if backend == BackendSystemd {
		return 0640
	}
	return 0644
}

// renderConfig produces the erpc.yaml for this backend.
//
// The docker backend renders the CONTAINER's view of the gateway (wide
// in-container listener, loopback upstreams re-pointed at the host alias)
// rather than the operator's view — see ops.GatewayContainerConfig for why
// each rewrite is mandatory. Both the write and the verify go through here,
// so a container config can never be verified against a host config.
func (p *gatewayPlan) renderConfig() (string, error) {
	g := p.gw
	if p.backend == BackendDocker {
		g = ops.GatewayContainerConfig(g, ops.DockerHostAlias)
	}
	cfg, err := catalog.RenderGatewayConfig(g)
	if err != nil {
		return "", fmt.Errorf("config: render gateway config: %w", err)
	}
	return cfg, nil
}

// ---------------------------------------------------------------------
// run
// ---------------------------------------------------------------------

func (p *gatewayPlan) runStep() Step {
	title := "Start gateway container"
	if p.backend == BackendSystemd {
		title = "Start gateway service (" + p.unitName() + ")"
	}
	return Step{
		ID:    "run",
		Title: title,
		Run: func(ctx context.Context, e executor.Executor, st *State) error {
			var err error
			switch p.backend {
			case BackendDocker:
				err = p.runDocker(ctx, e, st)
			case BackendSystemd:
				err = p.runSystemd(ctx, e, st)
			}
			if err != nil {
				return err
			}
			if err := p.waitReady(ctx, e); err != nil {
				return err
			}
			p.clearConfigPending()
			return nil
		},
		Verify: func(ctx context.Context, e executor.Executor, st *State) error {
			if p.configIsPending() {
				return fmt.Errorf("run: the gateway has not been (re)started onto the config just written")
			}
			return p.gatewayCheck(ctx, e)
		},
	}
}

// ensureImage builds the gateway image on the target unless it is already
// present. Presence is checked rather than always rebuilding because docker's
// own layer cache still costs a network round trip to resolve the git ref, and
// a re-run of provisioning should be quick.
//
// A build failure is terminal: continuing would call `docker run` on an image
// that does not exist, and docker would then try to PULL it — reporting a
// registry error for a build problem, which is a thoroughly misleading thing
// to show an operator.
func (p *gatewayPlan) ensureImage(ctx context.Context, e executor.Executor, st *State, platform string) error {
	tag := ops.ERPCImageTag()
	present, err := ops.ImageExists(ctx, e, tag)
	if err != nil {
		return fmt.Errorf("run: %w", err)
	}
	if present {
		_ = emit(ctx, st, Event{StepID: "run", Line: "image " + tag + " already present, skipping build"})
		return nil
	}

	_ = emit(ctx, st, Event{StepID: "run", Line: "building " + tag + " from " + ops.ERPCBuildContext() + " (first run: a few minutes)"})
	if _, err := ops.BuildImage(ctx, e, ops.ImageBuildArgs(ops.ImageBuildSpec{
		Tag:      tag,
		Platform: platform,
	})...); err != nil {
		return fmt.Errorf("run: %w", err)
	}
	return nil
}

func (p *gatewayPlan) runDocker(ctx context.Context, e executor.Executor, st *State) error {
	dest, err := p.configPath(ctx, e)
	if err != nil {
		return err
	}
	info, err := ops.ProbeDocker(ctx, e)
	if err != nil {
		return fmt.Errorf("run: %w", err)
	}
	platform := ops.EnginePlatform(ctx, e, info)

	// Build the image on the target when it isn't already there.
	//
	// The gateway image is built rather than pulled: upstream eRPC has no
	// WebSocket support, so its published image is the wrong binary, and the
	// valve fork publishes none. Building here also makes the image
	// native-arch by construction, which is what removes the missing-arm64
	// problem rather than working around it.
	//
	// Tagged by source SHA, so this is a one-time cost per pinned ref —
	// roughly 80s cold, seconds once docker's layer cache is warm.
	if err := p.ensureImage(ctx, e, st, platform); err != nil {
		return err
	}

	// The private network, before anything joins it. Idempotent, so this is a
	// no-op on every run after the first.
	if err := ops.EnsureNetwork(ctx, e, ops.NetworkName); err != nil {
		return fmt.Errorf("run: %w", err)
	}

	// Remove before run, always. A container's published port, mounts and
	// image are all fixed at creation, so applying a changed config is
	// necessarily remove + run, never a restart — and an absent container
	// makes this a no-op rather than an error.
	if err := ops.RemoveContainer(ctx, e, p.containerName()); err != nil {
		return fmt.Errorf("run: %w", err)
	}
	// A gateway that USED to be fronted and no longer is must lose its front
	// in the same breath, or the old Caddy keeps serving HTTPS from a Caddyfile
	// pointing at an eRPC that has just moved back to a published host port.
	// Removing an absent container is a no-op, so this is unconditional in the
	// unfronted direction only.
	if !p.fronted() {
		if err := ops.RemoveContainer(ctx, e, p.caddyName()); err != nil {
			return fmt.Errorf("run: %w", err)
		}
	}

	args := ops.ERPCRunArgs(ops.ERPCRunSpec{
		ContainerName:  p.containerName(),
		BindAddr:       p.gw.Bind(),
		HostPort:       p.gw.HTTP(),
		HostConfigPath: dest,
		// A VM-backed engine already provides host.docker.internal; a plain
		// Linux engine only has it when the container is started with this
		// mapping. It is no longer how a same-host CONTAINER is reached — the
		// network below is — but it remains the only way to reach a
		// systemd-installed node on the same box, which is not a container and
		// has no name on any network.
		AddHostGateway: !info.VMBacked(),
		Platform:       platform,
		Network:        ops.NetworkName,
		// A fronted gateway publishes NOTHING. Caddy is the only front door,
		// and it reaches eRPC by container name over the network above, so a
		// published eRPC port would be a second, plaintext, unauthenticated way
		// in that the operator did not ask for and would not see.
		NoPublish: p.fronted(),
	})
	res, err := ops.DockerRun(ctx, e, args...)
	if err != nil {
		return fmt.Errorf("run: %w", err)
	}
	if res.ExitCode != 0 {
		return fmt.Errorf("run: docker run failed (exit %d): %s", res.ExitCode, strings.TrimSpace(res.Stderr))
	}
	_ = emit(ctx, st, Event{StepID: "run", Line: strings.TrimSpace(res.Stdout)})

	return p.runCaddy(ctx, e, st, platform)
}

// runCaddy starts (or replaces) the gateway's TLS front and exports the root
// of its internal CA.
//
// It runs AFTER eRPC, not before, and the order is not cosmetic: Caddy resolves
// its upstream lazily per request, but starting it first would open an HTTPS
// port that answers 502 to everything, and the readiness poll below would then
// be measuring the wrong thing.
func (p *gatewayPlan) runCaddy(ctx context.Context, e executor.Executor, st *State, platform string) error {
	if !p.fronted() {
		// Already removed alongside the eRPC container; see runDocker.
		return nil
	}

	front, err := p.front(ctx, e)
	if err != nil {
		return fmt.Errorf("run: %w", err)
	}
	caddyfile, err := p.caddyfilePath(ctx, e)
	if err != nil {
		return err
	}
	if err := ops.RemoveContainer(ctx, e, p.caddyName()); err != nil {
		return fmt.Errorf("run: %w", err)
	}

	args := ops.CaddyRunArgs(ops.CaddyRunSpec{
		Image:          front.Caddy.ImageRef,
		ContainerName:  p.caddyName(),
		BindAddr:       p.gw.TLS.Bind(),
		HostPort:       p.gw.TLS.HTTPS(),
		Platform:       platform,
		HostConfigPath: caddyfile,
		Network:        ops.NetworkName,
		CertFile:       front.Caddy.CertFile,
		KeyFile:        front.Caddy.KeyFile,
	})
	res, err := ops.DockerRun(ctx, e, args...)
	if err != nil {
		return fmt.Errorf("run: %w", err)
	}
	if res.ExitCode != 0 {
		return fmt.Errorf("run: docker run (TLS front) failed (exit %d): %s", res.ExitCode, strings.TrimSpace(res.Stderr))
	}
	_ = emit(ctx, st, Event{StepID: "run", Line: strings.TrimSpace(res.Stdout)})

	if front.Caddy.CertSourceOrDefault() == catalog.CertInternal {
		p.exportRoot(ctx, e, st)
	}
	return nil
}

// exportRoot copies the internal CA's root out to the host, retrying briefly
// because Caddy writes it during startup and `docker run` returns before that.
//
// Failure is reported as a line, never as an error: the gateway serves HTTPS
// either way, and refusing to finish provisioning over a file that is only
// needed to make the browser warning go away would be a worse trade than the
// one this whole feature is built on.
func (p *gatewayPlan) exportRoot(ctx context.Context, e executor.Executor, st *State) {
	dest, err := p.rootCAPath(ctx, e)
	if err != nil {
		_ = emit(ctx, st, Event{StepID: "run", Line: "could not resolve where to put the internal CA root: " + err.Error()})
		return
	}
	deadline := time.Now().Add(caddyRootTimeout)
	for {
		got, err := exportRootCA(ctx, e, p.caddyName(), dest)
		if got {
			_ = emit(ctx, st, Event{StepID: "run", Line: "internal CA root written to " + dest +
				" — install it in your trust store (and on any other device that will call this gateway) to stop the browser warning"})
			return
		}
		if time.Now().After(deadline) {
			detail := ""
			if err != nil {
				detail = ": " + err.Error()
			}
			_ = emit(ctx, st, Event{StepID: "run", Line: "the TLS front's internal CA root could not be read yet" + detail +
				" — HTTPS still works, but this app cannot verify the chain and your browser will warn until you install the root by hand"})
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(caddyRootPollInterval):
		}
	}
}

func (p *gatewayPlan) runSystemd(ctx context.Context, e executor.Executor, st *State) error {
	cfgPath, err := p.configPath(ctx, e)
	if err != nil {
		return err
	}
	unit, err := catalog.RenderERPCUnitAt(p.unitWire(), cfgPath)
	if err != nil {
		return fmt.Errorf("run: render %s: %w", p.unitName(), err)
	}
	unitPath := erpcUnitDir + p.unitName()
	if err := e.WriteFile(ctx, unitPath, []byte(unit), 0644); err != nil {
		return fmt.Errorf("run: write %s: %w", unitPath, err)
	}

	// The unconditional restart is what applies a config change. wireStep
	// goes to the trouble of detecting a content change first because
	// restarting a chain client costs a resync-shaped outage; restarting a
	// stateless gateway costs a few hundred milliseconds, so buying that
	// distinction with extra machinery would be a bad trade.
	cmd := fmt.Sprintf("systemctl daemon-reload && systemctl enable --now %[1]s && systemctl restart %[1]s", p.unitName())
	res, err := e.Run(ctx, cmd, streamOpts(ctx, st, "run"))
	if err != nil {
		return fmt.Errorf("run: systemctl: %w", err)
	}
	if res.ExitCode != 0 {
		return fmt.Errorf("run: systemctl daemon-reload/enable/restart %s failed (exit %d): %s", p.unitName(), res.ExitCode, strings.TrimSpace(res.Stderr))
	}
	return nil
}

// waitReady polls until the gateway answers, because "the container started"
// and "the gateway serves RPC" are seconds apart — eRPC has to read its
// config and open its listener — and a Verify run immediately after Run
// would otherwise fail a gateway that is merely still starting.
func (p *gatewayPlan) waitReady(ctx context.Context, e executor.Executor) error {
	deadline := time.Now().Add(gatewayReadyTimeout)
	for {
		err := p.gatewayCheck(ctx, e)
		if err == nil {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("run: gateway did not answer within %s: %w", gatewayReadyTimeout, err)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(gatewayPollInterval):
		}
	}
}

// gatewayCheck is the gateway's equivalent of the node plan's handshake: one
// real RPC call, end to end, through the gateway's own port and the URL path
// callers will actually use (/<project>/evm/<chainId>).
//
// It targets the FIRST configured chain. Checking every chain would make the
// step's success depend on every upstream the operator listed being healthy
// right now, which is the opposite of what a gateway is for — a gateway with
// one dead fallback is still a working gateway. One chain proves what this
// step is responsible for: the process is up, the config parsed, the routing
// path resolves, and an upstream can be reached.
//
// eth_chainId specifically, because its answer is verifiable: a gateway that
// happily returns some OTHER chain's id has misrouted, and that is a
// misconfiguration worth failing on rather than a transient upstream fault.
func (p *gatewayPlan) gatewayCheck(ctx context.Context, e executor.Executor) error {
	chainID := p.gw.Networks[0].ChainID
	url, cmd, err := p.probeCommand(ctx, e, chainID)
	if err != nil {
		return err
	}
	res, err := e.Run(ctx, cmd, nil)
	if err != nil {
		return fmt.Errorf("gateway: eth_chainId probe: %w", err)
	}
	if res.ExitCode != 0 {
		return fmt.Errorf("gateway: eth_chainId at %s failed (curl exit %d): %s", url, res.ExitCode, strings.TrimSpace(res.Stderr))
	}

	var body struct {
		Result string `json:"result"`
		Error  *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	raw := strings.TrimSpace(res.Stdout)
	if err := json.Unmarshal([]byte(raw), &body); err != nil {
		// Not JSON at all is the normal shape of "nothing is listening yet"
		// (empty body) — report the raw answer, which is what the operator
		// needs to tell that apart from an HTML error page from something
		// else on the port.
		return fmt.Errorf("gateway: %s did not answer eth_chainId with JSON: %q", url, raw)
	}
	if body.Error != nil {
		return fmt.Errorf("gateway: %s answered eth_chainId with an error: %s", url, body.Error.Message)
	}
	got, err := parseHexChainID(body.Result)
	if err != nil {
		return fmt.Errorf("gateway: %s returned an unreadable eth_chainId result %q: %w", url, body.Result, err)
	}
	if got != chainID {
		return fmt.Errorf("gateway: %s is serving chain %d, but the config routes that path to chain %d", url, got, chainID)
	}
	return nil
}

// probeCommand builds the readiness probe: the URL it targets and the curl
// invocation that hits it.
//
// For an unfronted gateway that is the plain http endpoint on its published
// port, unchanged. For a fronted one the whole point is to prove the thing the
// operator will actually use, so the probe goes through HTTPS — which means
// three deliberate choices:
//
//   - --resolve pins the hostname to the published bind address ON THE TARGET
//     rather than trusting DNS. The probe runs on the target (curl over the
//     executor), Caddy publishes there, and pinning is what stops a gateway
//     that works perfectly from failing setup because the operator has not
//     pointed a name at their machine yet.
//   - --cacert names the internal CA's exported root, so the CHAIN IS ACTUALLY
//     VERIFIED. Passing -k here would have made the probe unable to distinguish
//     a working front from one serving a certificate for the wrong name — the
//     specific failure this feature introduces.
//   - a bare curl is appended with || as a second attempt when a CA file is not
//     available, which is the CertFiles case: a publicly-trusted certificate
//     (tailscale, localhost.direct) verifies against the system store and needs
//     no CA file, while a self-signed one verifies against its own certificate
//     passed as the CA. Trying the specific one first and the system store
//     second covers both without asking the operator which they have.
func (p *gatewayPlan) probeCommand(ctx context.Context, e executor.Executor, chainID int) (string, string, error) {
	body := shQuote(gatewayChainIDCall)
	const curlBase = "curl -s --max-time 10 -X POST -H 'Content-Type: application/json' --data "

	if !p.fronted() {
		url := fmt.Sprintf("http://%s:%d%s", probeHost(p.gw.Bind()), p.gw.HTTP(), p.gw.PathFor(chainID))
		return url, curlBase + body + " " + shQuote(url), nil
	}

	front, err := p.front(ctx, e)
	if err != nil {
		return "", "", err
	}
	tls := p.gw.TLS
	url := front.Caddy.URL() + p.gw.PathFor(chainID)
	resolve := fmt.Sprintf("--resolve %s", shQuote(fmt.Sprintf("%s:%d:%s", tls.Hostname, tls.HTTPS(), probeHost(tls.Bind()))))

	ca := front.Caddy.CertFile
	if front.Caddy.CertSourceOrDefault() == catalog.CertInternal {
		root, err := p.rootCAPath(ctx, e)
		if err != nil {
			return "", "", err
		}
		ca = root
	}

	attempt := func(extra string) string {
		return curlBase + body + " " + resolve + " " + extra + " " + shQuote(url)
	}
	cmd := attempt("--cacert " + shQuote(ca))
	if ca != "" {
		// Fall through to the system trust store when the CA file is absent or
		// is a leaf rather than an authority.
		cmd += " || " + attempt("")
	}
	return url, cmd, nil
}

// probeHost turns the gateway's bind address into something connectable. A
// wildcard bind names every interface but is not itself a destination on
// every platform (macOS refuses a connect to 0.0.0.0), so loopback is
// probed instead — a wildcard listener is on loopback too. IPv6 literals are
// bracketed because this value goes straight into a URL.
func probeHost(bind string) string {
	host := strings.Trim(bind, "[]")
	switch host {
	case "", "0.0.0.0":
		return "127.0.0.1"
	case "::", "::0":
		return "[::1]"
	}
	if strings.Contains(host, ":") {
		return "[" + host + "]"
	}
	return host
}

// parseHexChainID reads a JSON-RPC quantity ("0x171") as a chain id.
func parseHexChainID(hex string) (int, error) {
	n, err := strconv.ParseUint(strings.TrimPrefix(strings.TrimPrefix(strings.TrimSpace(hex), "0x"), "0X"), 16, 64)
	if err != nil {
		return 0, err
	}
	return int(n), nil
}
