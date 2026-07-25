package ops

// Docker backend for the eRPC gateway.
//
// WHY this exists at all: everything else valve-node-app provisions is a
// chain client — gigabytes of chain data, a dedicated service account, a
// hardened systemd unit, a Linux-only toolchain. The eRPC gateway is none
// of that. It is one static Go binary, one YAML file, and one TCP port,
// with no persistent state. That is exactly the shape that containerizes
// cleanly, which is what lets the gateway run on macOS and Windows hosts
// that internal/setup's preflight (internal/setup/steps.go, the uname !=
// "Linux" rejection) can never accept.
//
// Everything here goes through executor.Executor like the rest of the
// package, so a docker-hosted gateway works identically on the local
// machine and on an SSH target — the caller picks the executor, this file
// never knows which it got.

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"runtime"
	"strconv"
	"strings"

	"github.com/valve-tech/valve-node-app/internal/catalog"
	"github.com/valve-tech/valve-node-app/internal/executor"
)

// ---------------------------------------------------------------------
// capability probe
// ---------------------------------------------------------------------

// ErrDockerAbsent is the sentinel every "there is no docker CLI on this
// target" failure unwraps to, so callers can branch on it with errors.Is
// without string-matching. Absence is a first-class, expected outcome —
// the whole point of probing is to offer the operator a docker path when
// one exists and a clear install prompt when it doesn't — so it must be
// distinguishable from a genuine transport/daemon fault.
var ErrDockerAbsent = errors.New("ops: docker CLI not found on the target")

// DockerAbsentError is the typed error ProbeDocker returns when the docker
// CLI is not on the target's PATH. It carries the probe that established
// absence (for diagnostics) and an install hint the UI can show verbatim.
// It unwraps to ErrDockerAbsent, so both errors.Is(err, ErrDockerAbsent)
// and errors.As(err, &*DockerAbsentError) work.
type DockerAbsentError struct {
	// Probe is the shell command whose non-zero exit established absence.
	Probe string
	// ExitCode is that command's exit status.
	ExitCode int
	// Hint is operator-facing install guidance. It deliberately covers all
	// three host families because this probe runs before anything has
	// established which OS the target is — that is the very thing docker
	// support exists to stop caring about.
	Hint string
}

func (e *DockerAbsentError) Error() string {
	return fmt.Sprintf("%s (%s exited %d) — %s", ErrDockerAbsent.Error(), e.Probe, e.ExitCode, e.Hint)
}

// Unwrap makes errors.Is(err, ErrDockerAbsent) succeed for this type and
// for anything that wraps it with %w.
func (e *DockerAbsentError) Unwrap() error { return ErrDockerAbsent }

const dockerInstallHint = "install Docker on the target: Docker Desktop (macOS/Windows), " +
	"OrbStack or colima (macOS), or the docker.io / docker-ce package (Linux)"

// Known docker "flavors" — which engine is actually behind the docker CLI.
// This matters for real behavioral reasons, not cosmetics:
//   - Desktop/OrbStack/colima run the daemon inside a VM, so bind mounts
//     only work for host paths the VM shares, and the host's loopback is
//     NOT the container's loopback.
//   - podman's docker-compatible CLI accepts the same `docker run` flags
//     used here but differs on rootless port binding (<1024) and on
//     `--restart unless-stopped` semantics without a running user service.
//
// The values are stable identifiers safe to serialize to the UI.
const (
	FlavorDockerEngine   = "docker-engine"
	FlavorDockerDesktop  = "docker-desktop"
	FlavorColima         = "colima"
	FlavorOrbStack       = "orbstack"
	FlavorRancherDesktop = "rancher-desktop"
	FlavorPodman         = "podman"
	FlavorUnknown        = "unknown"
)

// DockerInfo is one point-in-time reading of a target's docker capability.
//
// Present and DaemonReachable are deliberately separate: "docker is
// installed but the daemon isn't running" is the single most common state
// on a fresh macOS/Windows box (Docker Desktop installed, never launched),
// and it needs a completely different operator prompt than "docker isn't
// installed". Neither is an error — both are readings, so ProbeDocker only
// returns an error for absence (typed) or a transport failure.
type DockerInfo struct {
	// Present is true when a `docker` executable is on the target's PATH.
	Present bool
	// DaemonReachable is true when `docker info` answered, i.e. the CLI
	// found and successfully talked to an engine.
	DaemonReachable bool

	// ClientBanner is `docker --version` verbatim, kept because it is the
	// clearest podman-shim tell ("podman version 5.4.0") and is useful
	// context in any bug report.
	ClientBanner string

	// ServerVersion, OSType and Architecture describe the ENGINE, not the
	// host running the CLI. On Docker Desktop for macOS the host is darwin
	// but OSType is "linux" (the VM) — which is precisely why containers
	// are a cross-platform answer, and why the raw host uname is the wrong
	// thing to gate on.
	ServerVersion string
	OSType        string // "linux" | "windows"
	Architecture  string // e.g. "x86_64", "aarch64"

	// HostName is `docker info`'s .Name (the engine's own host name, e.g.
	// "colima", "orbstack", "docker-desktop") and OperatingSystem is its
	// .OperatingSystem. Both feed flavor detection and are surfaced as-is.
	HostName        string
	OperatingSystem string

	// Flavor is one of the Flavor* constants.
	Flavor string

	// DaemonError is the engine's own complaint when DaemonReachable is
	// false (e.g. "Cannot connect to the Docker daemon at
	// unix:///var/run/docker.sock"). Empty when the daemon answered.
	DaemonError string
}

// WindowsContainers reports whether the engine is in Windows-container
// mode, where none of the Linux images valve-node-app ships can run. It is
// a distinct failure from "no docker" and deserves its own message: the fix
// is "switch to Linux containers", not "install docker".
func (d DockerInfo) WindowsContainers() bool {
	return strings.EqualFold(d.OSType, "windows")
}

// dockerVersionProbe / dockerInfoProbe are the exact commands ProbeDocker
// runs, named so tests and diagnostics can reference them.
//
// The info format string keeps the two free-text fields (.Name and
// .OperatingSystem) LAST so that SplitN with a fixed field count can't be
// derailed by a separator character appearing inside them.
const (
	dockerPresenceProbe = "command -v docker"
	dockerVersionProbe  = "docker --version"
	dockerInfoProbe     = "docker info --format '{{.ServerVersion}}|{{.OSType}}|{{.Architecture}}|{{.Name}}|{{.OperatingSystem}}'"
)

// ProbeDocker reports whether the target can host containers, and what kind
// of engine it has. It runs entirely over e, so it answers the same
// question for the local machine and for an SSH target.
//
// It returns *DockerAbsentError (unwrapping to ErrDockerAbsent) when there
// is no docker CLI at all. Every other reading — daemon down, Windows
// container mode, an unrecognized flavor — comes back as a populated
// DockerInfo with a nil error, because those are states the caller must
// present to the operator rather than failures of the probe itself. A
// non-nil error that is NOT ErrDockerAbsent always means the Executor
// itself failed (SSH dropped, context canceled), never a docker verdict.
func ProbeDocker(ctx context.Context, e executor.Executor) (DockerInfo, error) {
	var info DockerInfo

	res, err := e.Run(ctx, dockerPresenceProbe, nil)
	if err != nil {
		return info, fmt.Errorf("ops: docker probe: %s: %w", dockerPresenceProbe, err)
	}
	if res.ExitCode != 0 {
		return info, &DockerAbsentError{Probe: dockerPresenceProbe, ExitCode: res.ExitCode, Hint: dockerInstallHint}
	}
	info.Present = true

	// The client banner is best-effort context, never a gate: a CLI that
	// refuses --version but still talks to a daemon is odd, not fatal.
	if res, err := e.Run(ctx, dockerVersionProbe, nil); err == nil && res.ExitCode == 0 {
		info.ClientBanner = strings.TrimSpace(res.Stdout)
	}

	res, err = e.Run(ctx, dockerInfoProbe, nil)
	if err != nil {
		return info, fmt.Errorf("ops: docker probe: %s: %w", dockerInfoProbe, err)
	}
	if res.ExitCode != 0 {
		// The CLI exists but could not reach an engine. Report it as a
		// reading with the engine's own words attached — "Docker Desktop
		// isn't started" is an operator action, not a bug.
		info.DaemonError = firstNonEmptyLine(res.Stderr, res.Stdout)
		info.Flavor = detectDockerFlavor(info.ClientBanner, "", "")
		return info, nil
	}

	info.DaemonReachable = true
	info.ServerVersion, info.OSType, info.Architecture, info.HostName, info.OperatingSystem = parseDockerInfo(res.Stdout)
	info.Flavor = detectDockerFlavor(info.ClientBanner, info.HostName, info.OperatingSystem)
	return info, nil
}

// parseDockerInfo splits dockerInfoProbe's pipe-delimited line. Missing
// trailing fields come back empty rather than erroring: a future/older
// docker that can't render one of these templates still yields a usable
// partial reading, and every consumer treats these as descriptive.
func parseDockerInfo(stdout string) (serverVersion, osType, arch, name, operatingSystem string) {
	line := firstNonEmptyLine(stdout)
	fields := strings.SplitN(line, "|", 5)
	get := func(i int) string {
		if i < len(fields) {
			return strings.TrimSpace(fields[i])
		}
		return ""
	}
	return get(0), get(1), get(2), get(3), get(4)
}

// detectDockerFlavor classifies the engine from the CLI banner plus
// `docker info`'s .Name / .OperatingSystem. Pure, so the (inevitably
// heuristic) matching is directly testable against real-world strings.
//
// Order matters: podman is checked first because a podman install can
// present a `docker` CLI shim whose banner says "podman version …" while
// its info fields look like an ordinary Linux engine; Docker Desktop is
// checked before the generic engine because its .OperatingSystem is
// literally "Docker Desktop".
func detectDockerFlavor(banner, hostName, operatingSystem string) string {
	hay := strings.ToLower(banner + " " + hostName + " " + operatingSystem)
	switch {
	case strings.Contains(hay, "podman"):
		return FlavorPodman
	case strings.Contains(hay, "orbstack"):
		return FlavorOrbStack
	case strings.Contains(hay, "rancher"):
		return FlavorRancherDesktop
	case strings.Contains(hay, "colima"):
		return FlavorColima
	case strings.Contains(hay, "docker desktop"), strings.Contains(hay, "docker-desktop"):
		return FlavorDockerDesktop
	case strings.Contains(hay, "docker version"), hostName != "":
		return FlavorDockerEngine
	default:
		return FlavorUnknown
	}
}

// VMBacked reports whether this flavor runs the engine inside a VM rather
// than on the host kernel. Callers need it for the two things a VM changes:
// a bind-mount source path must be inside a directory the VM shares with
// the host, and the host's own services are not on the container's
// loopback (see ERPCContainerWire).
func (d DockerInfo) VMBacked() bool {
	switch d.Flavor {
	case FlavorDockerDesktop, FlavorColima, FlavorOrbStack, FlavorRancherDesktop:
		return true
	default:
		return false
	}
}

// ---------------------------------------------------------------------
// image platform
// ---------------------------------------------------------------------

// unameArchProbe asks the TARGET HOST's own shell what CPU it is on. It is
// the second opinion EnginePlatform needs; see there for why one reading is
// not enough.
//
// `command -p` is load-bearing, not decoration: it runs uname from the
// system's DEFAULT PATH, so a shim earlier in the operator's PATH cannot
// answer for it. MEASURED failure this prevents — a macOS host with Homebrew
// `coreutils` installed has an x86_64 GNU uname ahead of /usr/bin/uname, and
// that binary (itself translated by Rosetta) reports "x86_64" on an Apple
// Silicon machine. EnginePlatform then takes the host's word over the
// engine's (the VM-backed rule below), resolves linux/amd64, and `docker run
// --platform linux/amd64` of a locally built arm64 image does not fail with a
// platform error — it reports "Unable to find image locally" and tries to
// PULL, so the operator is shown a registry authentication failure for an
// image sitting on their own disk.
const unameArchProbe = "command -p uname -m"

// PlatformForArch maps a CPU architecture name onto a docker `--platform`
// value. It accepts every spelling this codebase meets: `docker info`'s and
// `uname -m`'s (x86_64, aarch64, armv7l) and Go's runtime.GOARCH (amd64,
// arm64).
//
// An empty or unrecognized arch yields "", meaning "emit no --platform and
// let the engine choose". That is deliberate: a WRONG --platform is strictly
// worse than none, because it turns an engine's correct multi-arch manifest
// selection into a hard "no matching manifest" failure.
func PlatformForArch(arch string) string {
	switch strings.ToLower(strings.TrimSpace(arch)) {
	case "x86_64", "x86-64", "amd64":
		return "linux/amd64"
	case "aarch64", "arm64", "arm64v8":
		return "linux/arm64"
	case "armv7l", "armhf", "arm":
		return "linux/arm/v7"
	default:
		return ""
	}
}

// DefaultPlatform is the --platform ERPCRunArgs emits when a run spec names
// none: the architecture valve-node-app ITSELF was built for.
//
// That is the right default because the container backend exists for the
// operator's own desktop (see this file's header), where the app and the
// engine are on the same machine — and unlike anything reported by the
// docker CLI, this reading cannot be distorted by binary translation.
func DefaultPlatform() string { return PlatformForArch(runtime.GOARCH) }

// EnginePlatform resolves the --platform value to run images as on this
// target. It exists because `docker info`'s architecture cannot be trusted
// on its own, which is a real, hand-observed failure and not a theoretical
// one:
//
// On an Apple Silicon Mac whose docker CLI is an x86_64 binary running under
// Rosetta, the arch reading comes back "x86_64". `docker run` of a locally
// built arm64 image then reports "Unable to find image locally" and goes off
// to PULL a nonexistent amd64 variant — the image is right there, it is just
// being looked up under the wrong platform. Passing --platform on BUILD is
// not enough; it is required on RUN.
//
// The host's own shell is not translated, so `uname -m` there reports the
// real CPU. When the two readings disagree AND the engine is VM-backed, the
// host wins: a desktop VM engine runs on the same silicon as its host, so a
// disagreement there means the CLI is misreporting. When the engine is NOT
// VM-backed the engine's reading wins — a plain engine reached over SSH or
// DOCKER_HOST genuinely can be a different architecture from the machine
// running this probe, and there the host reading is the misleading one.
//
// The one case this rule gets wrong is a deliberately cross-architecture
// desktop VM (`colima start --arch x86_64` on an arm64 Mac). Set
// ERPCRunSpec.Platform explicitly for that; an explicit value always wins.
func EnginePlatform(ctx context.Context, e executor.Executor, info DockerInfo) string {
	engine := PlatformForArch(info.Architecture)

	res, err := e.Run(ctx, unameArchProbe, nil)
	if err != nil || res.ExitCode != 0 {
		// Best effort by design: a target without `uname` still deserves the
		// engine's own reading rather than a failed provisioning run.
		return engine
	}
	host := PlatformForArch(firstNonEmptyLine(res.Stdout))
	if host == "" {
		return engine
	}
	if engine == "" || (host != engine && info.VMBacked()) {
		return host
	}
	return engine
}

// ---------------------------------------------------------------------
// eRPC container spec (pure rendering)
// ---------------------------------------------------------------------

const (
	// ERPCContainerName is the stable name the gateway container always
	// carries. Stability is the whole mechanism: it is how exists/running/
	// stop/remove find the container across app restarts, and how a re-run
	// of provisioning is idempotent instead of spawning a second gateway.
	ERPCContainerName = "valve-node-app-erpc"

	// ERPCSourceRepo and ERPCSourceRef pin the gateway's source. The image is
	// BUILT ON THE TARGET from this ref rather than pulled, for three
	// reasons: upstream eRPC has no WebSocket support at all, so a published
	// upstream image is the wrong binary for us; valve-tech/erpc publishes no
	// image (its CI builds and tests but never pushes); and a local build is
	// native-arch by construction, which sidesteps the missing-arm64-image
	// problem entirely instead of needing it solved.
	//
	// The ref is a full commit SHA, not the branch name. valve-ws is a moving
	// feature branch based on an open upstream PR, so building from the
	// branch head would silently change an operator's gateway between runs.
	// This SHA is the exact commit whose WebSocket support was verified end
	// to end (eth_subscribe newHeads through a real container).
	ERPCSourceRepo = "https://github.com/valve-tech/erpc.git"
	ERPCSourceRef  = "e909aacb462120e39db8d9a285dff4cde596425f"

	// erpcImageRepo is the local tag the built image carries. It is tagged by
	// source SHA so a rebuild is skippable when the image already exists, and
	// so bumping ERPCSourceRef produces a distinct image rather than silently
	// replacing one that containers may still reference.
	erpcImageRepo = "valve-node-app/erpc"

	// DefaultERPCImage is the pulled fallback for operators who would rather
	// not build. It is UPSTREAM eRPC and therefore has NO WebSocket support —
	// eth_subscribe against a gateway running this image fails with
	// ErrNoWsUpstreamAvailable. Only correct when the gateway is used purely
	// for request/response RPC.
	DefaultERPCImage = "ghcr.io/erpc/erpc:0.1.1"

	// erpcContainerConfigPath is where the host's erpc.yaml is mounted
	// inside the container. It matches the upstream image's own default
	// config location, so the explicit --config flag below is belt and
	// braces rather than the only thing making this work.
	erpcContainerConfigPath = "/erpc.yaml"

	// ERPCContainerPort is the port eRPC listens on INSIDE the container.
	// It is fixed at eRPC's own default (catalog's defaultERPCPort) and is
	// deliberately not configurable: the container's port namespace is
	// private, so there is nothing to avoid colliding with, and the
	// operator's choice of port is expressed on the host side of the -p
	// mapping instead.
	ERPCContainerPort = 4000

	// DockerHostAlias is the DNS name that resolves to the container's host
	// from inside a container. It is built in on Docker Desktop, OrbStack,
	// colima and Rancher Desktop; on plain Linux engines it only exists
	// when the container was started with the --add-host host-gateway
	// mapping ERPCRunSpec.AddHostGateway emits.
	DockerHostAlias = "host.docker.internal"
)

// ERPCRunSpec is everything the pure arg renderer needs. Zero values
// resolve to the same defaults catalog.WireConfig's ERPC* accessors use, so
// a zero spec is still a valid, loopback-bound gateway.
type ERPCRunSpec struct {
	// Image is the container image ref ("" → DefaultERPCImage).
	Image string
	// ContainerName is the container's --name ("" → ERPCContainerName).
	ContainerName string
	// BindAddr is the HOST address the published port binds to
	// ("" → 127.0.0.1, matching catalog.WireConfig.ERPCBind). This is the
	// only thing controlling who can reach the gateway; the in-container
	// listener is always wide open on the container's private namespace.
	BindAddr string
	// HostPort is the host-side port (0 → ERPCContainerPort).
	HostPort int
	// Platform is the image platform to run as, e.g. "linux/arm64"
	// ("" → DefaultPlatform()). It must be set on RUN, not only on BUILD:
	// docker resolves an image ref by (ref, platform), so a CLI that
	// misreports its architecture looks up the wrong variant of a perfectly
	// present local image and falls through to a pull. See EnginePlatform
	// for how to derive this for a target the app is not running on.
	Platform string
	// HostConfigPath is the absolute path to erpc.yaml ON THE HOST. It is
	// bind-mounted read-only: the gateway never writes its config, and a
	// read-only mount means a compromised gateway cannot rewrite its own
	// upstream list.
	HostConfigPath string
	// AddHostGateway emits --add-host host.docker.internal:host-gateway.
	// Required on plain Linux engines whenever the config's local-node
	// upstream is addressed via DockerHostAlias; harmless (and ignored) on
	// VM-backed engines that already provide the alias.
	AddHostGateway bool
}

// ERPCRunArgs renders the argv for `docker run` — WITHOUT the leading
// "docker" — for the eRPC gateway.
//
// It is deliberately pure (no Executor, no error): the argument list is the
// part with all the fiddly, easy-to-regress detail (bind-address forms,
// IPv6 bracketing, mount flags, flag order), so it is worth being able to
// assert on directly in a test without a fake process anywhere. DockerRun
// below is the thin impure shell around it.
//
// The rendered container is detached (-d) and --restart unless-stopped, so
// the gateway survives a host reboot and a daemon restart but stays down
// when an operator deliberately stops it — the container-world equivalent
// of the systemd `Restart=always` + explicit `systemctl stop` contract the
// exec/beacon units already have.
func ERPCRunArgs(spec ERPCRunSpec) []string {
	name := spec.ContainerName
	if name == "" {
		name = ERPCContainerName
	}
	image := spec.Image
	if image == "" {
		// The built-from-source image, not the upstream pull: upstream eRPC
		// cannot do WebSocket, and a gateway that silently drops
		// eth_subscribe is worse than one that fails to start.
		image = ERPCImageTag()
	}
	hostPort := spec.HostPort
	if hostPort == 0 {
		hostPort = ERPCContainerPort
	}
	bind := spec.BindAddr
	if bind == "" {
		bind = "127.0.0.1"
	}
	platform := spec.Platform
	if platform == "" {
		platform = DefaultPlatform()
	}

	args := []string{
		"run", "-d",
		"--name", name,
		"--restart", "unless-stopped",
	}
	// Emitted only when a platform is actually known: PlatformForArch
	// returns "" for an architecture nobody recognized, and there the
	// engine's own manifest selection beats a guess.
	if platform != "" {
		args = append(args, "--platform", platform)
	}
	if spec.AddHostGateway {
		args = append(args, "--add-host", DockerHostAlias+":host-gateway")
	}
	args = append(args,
		"-p", publishSpec(bind, hostPort, ERPCContainerPort),
		"-v", spec.HostConfigPath+":"+erpcContainerConfigPath+":ro",
		image,
	)
	// NOTHING is appended after the image ref, and that is a correction, not
	// an omission. This used to end `image --config /erpc.yaml`, which reads
	// as belt-and-braces and is in fact fatal: anything after the image ref
	// REPLACES the image's CMD, and the gateway image built from
	// ERPCBuildContext carries no ENTRYPOINT — its CMD *is* ["/erpc-server"].
	// Overriding it left docker trying to exec a flag, and the container died
	// at creation with:
	//
	//   exec: "--config": executable file not found in $PATH
	//
	// (measured against the image built from e909aacb). The config path is
	// already the one the image looks in by default — that is why
	// erpcContainerConfigPath is what it is — so the mount alone is what makes
	// this work, on this image and on the upstream one, whichever of them
	// declares its binary as an ENTRYPOINT and whichever as a CMD.
	return args
}

// publishSpec renders one -p value: "<bind>:<hostPort>:<containerPort>".
// An IPv6 bind address must be bracketed or docker parses its colons as
// the field separators and rejects the whole mapping — the one piece of
// this string that is not obvious, and the reason it is a named function
// with its own tests.
func publishSpec(bind string, hostPort, containerPort int) string {
	if strings.Contains(bind, ":") && !strings.HasPrefix(bind, "[") {
		bind = "[" + bind + "]"
	}
	return bind + ":" + strconv.Itoa(hostPort) + ":" + strconv.Itoa(containerPort)
}

// ERPCRunSpecFor derives a run spec from the same catalog.WireConfig the
// systemd path uses, so a docker-hosted gateway and a unit-hosted gateway
// are driven by ONE config object and cannot drift apart. addHostGateway
// should be true for a non-VM-backed engine (see DockerInfo.VMBacked).
func ERPCRunSpecFor(w catalog.WireConfig, addHostGateway bool) ERPCRunSpec {
	return ERPCRunSpec{
		BindAddr:       w.ERPCBind(),
		HostPort:       w.ERPCHTTP(),
		HostConfigPath: w.ERPCConfigPath(),
		AddHostGateway: addHostGateway,
	}
}

// ERPCContainerWire returns a COPY of w describing the gateway as it must
// be configured from inside the container, for feeding to
// catalog.RenderERPCConfig. Two rewrites, both mandatory, both silent
// killers if skipped:
//
//  1. The listener must bind 0.0.0.0 inside the container. The container
//     has its own network namespace, so a 127.0.0.1 listener there is
//     reachable only from inside the container and docker's published port
//     forwards into a black hole. Host-side exposure is enforced by the -p
//     bind address instead (ERPCRunSpec.BindAddr) — which is strictly
//     safer, since it is the host kernel refusing the connection rather
//     than the app.
//  2. The local-node upstream must not be loopback. "127.0.0.1:8545" inside
//     the container is the CONTAINER's loopback, not the node's. When the
//     node's RPC is loopback-bound, the endpoint is rewritten to hostAlias
//     (normally DockerHostAlias). When the operator has bound the node's
//     RPC to a routable address (e.g. a Tailscale IP), that address already
//     works from inside the container and is left untouched.
//
// hostAlias == "" disables rewrite 2 (useful when the caller knows the
// gateway and node share a network some other way).
func ERPCContainerWire(w catalog.WireConfig, hostAlias string) catalog.WireConfig {
	w.ERPCBindAddr = "0.0.0.0"
	w.ERPCPort = ERPCContainerPort
	if hostAlias != "" && isLoopbackAddr(w.RPCBind()) {
		w.RPCBindAddr = hostAlias
	}
	return w
}

// GatewayContainerConfig is ERPCContainerWire's equivalent for a multi-chain
// gateway (catalog.GatewayConfig): it returns a COPY of g describing the
// gateway as it must be configured from INSIDE the container. The two
// rewrites, and why each is mandatory, are documented on ERPCContainerWire —
// they apply identically here, just across every configured chain rather
// than one.
//
// The copy is deep, unlike ERPCContainerWire's. A GatewayConfig carries
// slices, so a plain value copy would share the caller's upstream backing
// array and the endpoint rewrite below would reach back and mutate the
// operator's own config — the kind of aliasing bug that only shows up once
// the same GatewayConfig is rendered twice (once for the container, once for
// the UI) and the second render is already container-flavored.
//
// Every loopback endpoint is rewritten, not just the ones flagged Local: an
// endpoint on 127.0.0.1 is unreachable from inside the container regardless
// of which tier the operator filed it under.
func GatewayContainerConfig(g catalog.GatewayConfig, hostAlias string) catalog.GatewayConfig {
	out := g
	out.BindAddr = "0.0.0.0"
	out.Port = ERPCContainerPort

	out.Networks = make([]catalog.GatewayNetwork, len(g.Networks))
	for i, n := range g.Networks {
		ups := make([]catalog.GatewayUpstream, len(n.Upstreams))
		copy(ups, n.Upstreams)
		if hostAlias != "" {
			for j := range ups {
				ups[j].Endpoint = rewriteLoopbackHost(ups[j].Endpoint, hostAlias)
			}
		}
		out.Networks[i] = catalog.GatewayNetwork{ChainID: n.ChainID, Upstreams: ups}
	}
	return out
}

// rewriteLoopbackHost swaps a loopback host in endpoint for hostAlias,
// preserving scheme, port and path. An endpoint that does not parse, or that
// is not loopback, is returned untouched — being conservative here matters
// because a mangled upstream URL is a gateway that silently serves nothing.
func rewriteLoopbackHost(endpoint, hostAlias string) string {
	u, err := url.Parse(strings.TrimSpace(endpoint))
	if err != nil || u.Host == "" || !isLoopbackAddr(u.Hostname()) {
		return endpoint
	}
	if port := u.Port(); port != "" {
		u.Host = net.JoinHostPort(hostAlias, port)
	} else {
		u.Host = hostAlias
	}
	return u.String()
}

// isLoopbackAddr reports whether addr is one of the loopback spellings a
// WireConfig can carry. Kept as a plain string comparison (not net.ParseIP)
// because RPCBindAddr may legitimately be a host name, and anything that is
// not literally loopback should be left alone.
func isLoopbackAddr(addr string) bool {
	switch strings.Trim(addr, "[]") {
	case "127.0.0.1", "::1", "localhost":
		return true
	default:
		return false
	}
}

// ---------------------------------------------------------------------
// container lifecycle helpers
// ---------------------------------------------------------------------

// DockerRun executes `docker <args...>` over e, shell-quoting every
// argument. Quoting each element individually is what keeps ERPCRunArgs
// able to stay a plain []string: the Executor contract is a single `sh -c`
// string (internal/executor/executor.go), so something has to do the
// joining, and doing it here means a path with a space or a quote in it
// cannot break out of its argument.
func DockerRun(ctx context.Context, e executor.Executor, args ...string) (executor.Result, error) {
	quoted := make([]string, 0, len(args)+1)
	quoted = append(quoted, "docker")
	for _, a := range args {
		quoted = append(quoted, shQuote(a))
	}
	cmd := strings.Join(quoted, " ")
	res, err := e.Run(ctx, cmd, nil)
	if err != nil {
		return res, fmt.Errorf("ops: %s: %w", cmd, err)
	}
	return res, nil
}

// ContainerExists reports whether a container with this name exists in any
// state (running, exited, created). The `^name$` anchors matter: docker's
// --filter name= is a substring REGEX, so an unanchored filter would report
// "valve-node-app-erpc" as existing when only "valve-node-app-erpc-old" is
// present.
func ContainerExists(ctx context.Context, e executor.Executor, name string) (bool, error) {
	res, err := DockerRun(ctx, e, "ps", "-a", "--filter", "name=^"+name+"$", "--format", "{{.Names}}")
	if err != nil {
		return false, err
	}
	if res.ExitCode != 0 {
		return false, fmt.Errorf("ops: docker ps failed (exit %d): %s", res.ExitCode, strings.TrimSpace(res.Stderr))
	}
	for _, line := range strings.Split(res.Stdout, "\n") {
		if strings.TrimSpace(line) == name {
			return true, nil
		}
	}
	return false, nil
}

// ContainerRunning reports whether the named container is running right
// now. A missing container is NOT an error — `docker inspect` exits
// non-zero for "no such object", which reads as "not running", exactly as
// ops.isActive treats a non-zero `systemctl is-active`. Only a transport
// failure is returned as an error.
func ContainerRunning(ctx context.Context, e executor.Executor, name string) (bool, error) {
	res, err := DockerRun(ctx, e, "inspect", "-f", "{{.State.Running}}", name)
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(res.Stdout) == "true", nil
}

// StopContainer stops the named container. A container that does not exist
// or is already stopped is success, not failure: this is used on the
// toggle-off and re-provision paths, where the goal state is "not running"
// and both of those already satisfy it.
func StopContainer(ctx context.Context, e executor.Executor, name string) error {
	res, err := DockerRun(ctx, e, "stop", name)
	if err != nil {
		return err
	}
	if res.ExitCode != 0 && !isNoSuchContainer(res.Stderr) {
		return fmt.Errorf("ops: docker stop %s failed (exit %d): %s", name, res.ExitCode, strings.TrimSpace(res.Stderr))
	}
	return nil
}

// RemoveContainer force-removes the named container, stopping it first if
// needed (-f). Removal is how a config/image change is applied — a
// container's -p mapping, mounts and image are all fixed at creation, so
// "reconfigure" always means remove + run, never a restart. As with
// StopContainer, an absent container is success.
func RemoveContainer(ctx context.Context, e executor.Executor, name string) error {
	res, err := DockerRun(ctx, e, "rm", "-f", name)
	if err != nil {
		return err
	}
	if res.ExitCode != 0 && !isNoSuchContainer(res.Stderr) {
		return fmt.Errorf("ops: docker rm -f %s failed (exit %d): %s", name, res.ExitCode, strings.TrimSpace(res.Stderr))
	}
	return nil
}

// isNoSuchContainer recognizes the engine's "container doesn't exist"
// complaint across docker ("No such container: x") and podman ("no such
// container"), which is the one non-zero exit stop/remove must treat as
// success.
func isNoSuchContainer(stderr string) bool {
	return strings.Contains(strings.ToLower(stderr), "no such container")
}

// firstNonEmptyLine returns the first non-blank trimmed line across the
// given strings, in order. Engine errors arrive on stderr for docker and
// (for some podman builds) on stdout, so both are offered here.
func firstNonEmptyLine(streams ...string) string {
	for _, s := range streams {
		for _, line := range strings.Split(s, "\n") {
			if trimmed := strings.TrimSpace(line); trimmed != "" {
				return trimmed
			}
		}
	}
	return ""
}

// ---------------------------------------------------------------------
// building the gateway image on the target
// ---------------------------------------------------------------------
//
// Publishing multi-arch images for the valve forks would be a nicer setup
// experience, but it is not a prerequisite and treating it as one was a
// mistake. Building on the target removes the dependency entirely and is
// native-arch by construction, so the absence of arm64 images stops being a
// problem to solve rather than one to work around. It is also what this
// codebase already does everywhere else: every catalog.Client installs by
// source build.
//
// Docker builds straight from a git ref, so there is no clone step to manage:
// measured at ~83s cold for the gateway and ~1s when the layers are cached.

// ERPCImageTag is the local tag the gateway image is built as, derived from
// the pinned source SHA so that bumping ERPCSourceRef yields a distinct image
// and an unchanged ref makes the build skippable.
func ERPCImageTag() string {
	ref := ERPCSourceRef
	if len(ref) > 8 {
		ref = ref[:8]
	}
	return erpcImageRepo + ":" + ref
}

// ERPCBuildContext is the git build context docker resolves the source from.
// A full commit SHA is a valid ref here, which is what makes the build
// reproducible against a moving feature branch.
func ERPCBuildContext() string {
	return ERPCSourceRepo + "#" + ERPCSourceRef
}

// ImageBuildSpec is everything the pure build-arg renderer needs.
type ImageBuildSpec struct {
	// Tag is the image tag to produce ("" → ERPCImageTag()).
	Tag string
	// Context is a git URL with a #ref, or a directory on the target
	// ("" → ERPCBuildContext()).
	Context string
	// Platform is the target platform, e.g. "linux/arm64" ("" →
	// DefaultPlatform()). Omitted entirely when no architecture is
	// recognizable, for the same reason ERPCRunArgs omits it: a wrong
	// --platform turns correct manifest selection into a hard failure.
	Platform string
}

// ImageBuildArgs renders the docker build argv. Pure, so the argv is testable
// without a daemon.
func ImageBuildArgs(spec ImageBuildSpec) []string {
	tag := spec.Tag
	if tag == "" {
		tag = ERPCImageTag()
	}
	buildCtx := spec.Context
	if buildCtx == "" {
		buildCtx = ERPCBuildContext()
	}
	platform := spec.Platform
	if platform == "" {
		platform = DefaultPlatform()
	}

	args := []string{"build"}
	if platform != "" {
		args = append(args, "--platform", platform)
	}
	// The context goes last: docker treats the first non-flag argument as
	// the context, so an option appended after it would be read as a second
	// context and rejected.
	return append(args, "-t", tag, buildCtx)
}

// ImageExists reports whether tag is already present on the target, so a
// re-run of provisioning skips a rebuild rather than repeating it. `docker
// image inspect` is used rather than `docker images -q` because it
// distinguishes "absent" (non-zero exit) from "daemon unreachable" (an error
// on the command itself).
func ImageExists(ctx context.Context, e executor.Executor, tag string) (bool, error) {
	res, err := e.Run(ctx, "docker image inspect "+shQuote(tag)+" --format '{{.Id}}'", nil)
	if err != nil {
		return false, fmt.Errorf("ops: docker image inspect %s: %w", tag, err)
	}
	return res.ExitCode == 0 && strings.TrimSpace(res.Stdout) != "", nil
}

// BuildImage runs a docker build on the target. Each argv element is quoted at
// the sh -c boundary, matching DockerRun.
func BuildImage(ctx context.Context, e executor.Executor, args ...string) (executor.Result, error) {
	quoted := make([]string, 0, len(args)+1)
	quoted = append(quoted, "docker")
	for _, a := range args {
		quoted = append(quoted, shQuote(a))
	}
	cmd := strings.Join(quoted, " ")
	res, err := e.Run(ctx, cmd, nil)
	if err != nil {
		return res, fmt.Errorf("ops: docker build: %w", err)
	}
	if res.ExitCode != 0 {
		return res, fmt.Errorf("ops: docker build failed (exit %d): %s",
			res.ExitCode, firstNonEmptyLine(res.Stderr, res.Stdout))
	}
	return res, nil
}
