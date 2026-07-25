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
// eRPC container spec (pure rendering)
// ---------------------------------------------------------------------

const (
	// ERPCContainerName is the stable name the gateway container always
	// carries. Stability is the whole mechanism: it is how exists/running/
	// stop/remove find the container across app restarts, and how a re-run
	// of provisioning is idempotent instead of spawning a second gateway.
	ERPCContainerName = "valve-node-app-erpc"

	// DefaultERPCImage is the pinned gateway image. Pinned, not :latest, for
	// the same reason catalog.Client.PinVersion exists — an operator's
	// gateway must not silently change version under them on a restart.
	DefaultERPCImage = "ghcr.io/erpc/erpc:0.1.1"

	// erpcContainerConfigPath is where the host's erpc.yaml is mounted
	// inside the container. It matches the upstream image's own default
	// config location, so the explicit --config flag below is belt and
	// braces rather than the only thing making this work.
	erpcContainerConfigPath = "/erpc.yaml"

	// erpcContainerPort is the port eRPC listens on INSIDE the container.
	// It is fixed at eRPC's own default (catalog's defaultERPCPort) and is
	// deliberately not configurable: the container's port namespace is
	// private, so there is nothing to avoid colliding with, and the
	// operator's choice of port is expressed on the host side of the -p
	// mapping instead.
	erpcContainerPort = 4000

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
	// HostPort is the host-side port (0 → erpcContainerPort).
	HostPort int
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
		image = DefaultERPCImage
	}
	hostPort := spec.HostPort
	if hostPort == 0 {
		hostPort = erpcContainerPort
	}
	bind := spec.BindAddr
	if bind == "" {
		bind = "127.0.0.1"
	}

	args := []string{
		"run", "-d",
		"--name", name,
		"--restart", "unless-stopped",
	}
	if spec.AddHostGateway {
		args = append(args, "--add-host", DockerHostAlias+":host-gateway")
	}
	args = append(args,
		"-p", publishSpec(bind, hostPort, erpcContainerPort),
		"-v", spec.HostConfigPath+":"+erpcContainerConfigPath+":ro",
		image,
		"--config", erpcContainerConfigPath,
	)
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
	w.ERPCPort = erpcContainerPort
	if hostAlias != "" && isLoopbackAddr(w.RPCBind()) {
		w.RPCBindAddr = hostAlias
	}
	return w
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
