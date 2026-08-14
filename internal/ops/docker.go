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
	"encoding/json"
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
// An empty or unrecognized arch yields "", meaning "this reading told us
// nothing" — NOT "omit the flag". Every caller must fall through to another
// reading rather than emitting nothing; see EnginePlatform and
// resolveRunPlatform for why omission is no longer an option.
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

// DefaultPlatform is the --platform every run/build renderer falls back to
// when nothing better is known: the architecture valve-node-app ITSELF was
// built for.
//
// That is the right last resort because the container backend exists for the
// operator's own desktop (see this file's header), where the app and the
// engine are on the same machine — and unlike anything reported by the docker
// CLI, this reading cannot be distorted by binary translation.
//
// It never returns "". An architecture PlatformForArch does not recognize is
// still rendered as "linux/<GOARCH>", because Go's GOARCH names and docker's
// platform arch names agree for every architecture either of them supports.
// See resolveRunPlatform for why an empty answer is not acceptable here.
func DefaultPlatform() string {
	if p := PlatformForArch(runtime.GOARCH); p != "" {
		return p
	}
	return "linux/" + runtime.GOARCH
}

// resolveRunPlatform is the single place that answers "what --platform does
// this docker command get", and it ALWAYS answers.
//
// WHY it can never return "": an omitted --platform does not mean "let the
// engine choose from the manifest". It means "let DOCKER_DEFAULT_PLATFORM
// choose", and that environment variable is invisible at the point of use.
//
// MEASURED, on an arm64 Mac with DOCKER_DEFAULT_PLATFORM=linux/amd64 exported
// into the app's environment: a devnet reset through the app created an amd64
// reth container. The engine reported State=running and the container answered
// no RPC at all — QEMU user-mode emulation of reth is slow enough that the
// readiness probe timed out, and every status read afterwards said "running".
// That is the worst failure shape in this codebase: not an error, a lie. An
// explicit --platform on both `docker run` and `docker build` is what removes
// the variable from the decision entirely.
func resolveRunPlatform(explicit string) string {
	if p := strings.TrimSpace(explicit); p != "" {
		return p
	}
	return DefaultPlatform()
}

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
//
// It NEVER returns "". It used to, when neither reading was recognizable, and
// callers took that as "omit --platform" — which does not hand the choice to
// the engine's manifest resolution, it hands it to DOCKER_DEFAULT_PLATFORM.
// The fallback order is therefore engine reading, then host reading, then
// DefaultPlatform(); see resolveRunPlatform for the measured failure.
func EnginePlatform(ctx context.Context, e executor.Executor, info DockerInfo) string {
	engine := PlatformForArch(info.Architecture)

	host := ""
	// Best effort by design: a target without `uname` still deserves the
	// engine's own reading rather than a failed provisioning run.
	if res, err := e.Run(ctx, unameArchProbe, nil); err == nil && res.ExitCode == 0 {
		host = PlatformForArch(firstNonEmptyLine(res.Stdout))
	}

	switch {
	case engine == "" && host == "":
		return DefaultPlatform()
	case engine == "":
		return host
	case host == "":
		return engine
	case host != engine && info.VMBacked():
		return host
	default:
		return engine
	}
}

// ---------------------------------------------------------------------
// emulation detection
// ---------------------------------------------------------------------

// containerPlatformFormat asks a CONTAINER what platform it actually resolved
// to. This is the primary reading and the only one that is reliable on a
// modern engine.
//
// MEASURED, and it is why this is not simply an image inspect: on Docker 29
// with the containerd image store, `docker inspect -f '{{.Image}}'` of a
// container created with --platform linux/amd64 returns the MANIFEST LIST's
// digest, not the amd64 manifest's. Inspecting that id reports the HOST's
// architecture — arm64 — for a container demonstrably running amd64 code (its
// own logs said "This is the expected behaviour if you are running under
// QEMU", and `docker exec … uname -m` said x86_64). An image-only check would
// therefore have reported every emulated container as native, which is exactly
// the false negative this whole feature exists to prevent.
//
// .ImageManifestDescriptor carries the RESOLVED platform and is correct.
const containerPlatformFormat = `{{json .ImageManifestDescriptor}}`

// ContainerPlatformArgs renders the argv (without the leading "docker") for
// that reading. On an engine too old to have the field, docker fails the
// template with a non-zero exit, which is the caller's signal to fall back to
// ImagePlatformArgs — where the classic image store makes the image id
// platform-specific and the old reading is correct.
func ContainerPlatformArgs(name string) []string {
	return []string{"inspect", "-f", containerPlatformFormat, name}
}

// imageManifestDescriptor is the subset of the OCI descriptor that answers
// "what platform is this really".
type imageManifestDescriptor struct {
	Platform *struct {
		OS           string `json:"os"`
		Architecture string `json:"architecture"`
		Variant      string `json:"variant"`
	} `json:"platform"`
}

// parseContainerPlatform reads containerPlatformFormat's output into an
// "os/arch" string, or "" when the engine had nothing to say. Pure, so the
// several shapes docker can emit here (a descriptor, a JSON null, an empty
// line) are testable without a daemon.
func parseContainerPlatform(stdout string) string {
	line := firstNonEmptyLine(stdout)
	if line == "" || line == "null" {
		return ""
	}
	var d imageManifestDescriptor
	if err := json.Unmarshal([]byte(line), &d); err != nil || d.Platform == nil {
		return ""
	}
	if d.Platform.OS == "" || d.Platform.Architecture == "" {
		return ""
	}
	return d.Platform.OS + "/" + d.Platform.Architecture
}

// imagePlatformFormat renders an image's own platform as docker spells it.
// The variant is only appended when the image declares one, so an arm64 image
// reads "linux/arm64" rather than "linux/arm64/", matching what --platform
// takes.
const imagePlatformFormat = `{{.Os}}/{{.Architecture}}{{if .Variant}}/{{.Variant}}{{end}}`

// ImagePlatformArgs renders the argv (without the leading "docker") that asks
// what platform an IMAGE is. It is the FALLBACK reading — correct on the
// classic image store, where an image id names one platform's manifest, and
// misleading on the containerd store; see containerPlatformFormat.
func ImagePlatformArgs(ref string) []string {
	return []string{"image", "inspect", "-f", imagePlatformFormat, ref}
}

// enginePlatformProbe asks the ENGINE what it runs natively. `docker version`
// rather than `docker info` on purpose: it answers the same question and is
// materially cheaper, and this runs on every status read of a live container.
const enginePlatformProbe = "docker version --format '{{.Server.Os}}/{{.Server.Arch}}'"

// EmulatedPlatform reports whether an image platform is being run on an engine
// of a different architecture — i.e. through QEMU rather than natively.
//
// The comparison is on ARCHITECTURE ONLY, and deliberately ignores the ARM
// variant: an arm64 engine runs a linux/arm64/v8 image natively, and treating
// the variant as a mismatch would flag every ordinary arm64 container. An
// unknown reading on either side yields false — an unproven suspicion of
// emulation is not worth showing an operator a warning about.
func EmulatedPlatform(imagePlatform, engPlatform string) bool {
	img, eng := platformArch(imagePlatform), platformArch(engPlatform)
	if img == "" || eng == "" {
		return false
	}
	return img != eng
}

// platformArch pulls the architecture out of an "os/arch[/variant]" string,
// normalizing the spellings PlatformForArch accepts so that "x86_64" from one
// source and "amd64" from another are not read as two architectures.
func platformArch(platform string) string {
	parts := strings.Split(strings.TrimSpace(platform), "/")
	if len(parts) < 2 {
		return ""
	}
	if p := PlatformForArch(parts[1]); p != "" {
		// Reduce back to the arch half so linux/arm/v7 and armv7l agree.
		return strings.TrimPrefix(p, "linux/")
	}
	return strings.ToLower(parts[1])
}

// ---------------------------------------------------------------------
// eRPC container spec (pure rendering)
// ---------------------------------------------------------------------

const (
	// ERPCContainerName is the name the DEFAULT gateway's container carries,
	// and the prefix every other gateway's name is built from. Stability is
	// the whole mechanism: it is how exists/running/stop/remove find the
	// container across app restarts, and how a re-run of provisioning is
	// idempotent instead of spawning a second gateway.
	//
	// It is no longer THE gateway container name — see ERPCContainerNameFor.
	// A gateway is a fleet-wide layer and there can be several, so anything
	// that addresses "the" gateway container by this constant is assuming a
	// single gateway and is wrong.
	ERPCContainerName = "valve-node-app-erpc"

	// DefaultGatewayID is the gateway id whose container keeps the historical
	// bare name. It matches config.DefaultGatewayID (ops cannot import config)
	// and exists so an operator whose gateway predates multi-gateway support
	// keeps the container they already have running instead of acquiring a
	// second one beside it.
	DefaultGatewayID = "default"

	// ERPCSourceRepo and ERPCSourceRef pin the gateway's source. The image is
	// BUILT ON THE TARGET from this ref rather than pulled, for three
	// reasons: upstream eRPC has no WebSocket support at all, so a published
	// upstream image is the wrong binary for us; valve-tech/erpc publishes no
	// image (its CI builds and tests but never pushes); and a local build is
	// native-arch by construction, which sidesteps the missing-arm64-image
	// problem entirely instead of needing it solved.
	//
	// The ref is a full commit SHA, not the branch name. The branch it sits on
	// is a moving feature branch based on an open upstream PR, so building from
	// the branch head would silently change an operator's gateway between runs.
	// This SHA is the exact commit whose WebSocket support was verified end
	// to end (eth_subscribe newHeads through a real container).
	//
	// Now on fix/ws-upgrade-behind-gzip, which is valve-ws plus two fixes the
	// gateway needs and one gofmt pass. Both were found by putting real traffic
	// through this stack, and both fail silently rather than loudly:
	//
	//   - A WebSocket upgrade answered HTTP 500 whenever the client sent
	//     Accept-Encoding: gzip. Caddy sends it on every proxied request, so
	//     eth_subscribe through our own HTTPS front door could never work. The
	//     gzip wrapper does not implement http.Hijacker and gorilla needs one.
	//   - A multi-chain BATCH posted to /<project> answered every entry from
	//     one entry's network, because the batch goroutines shared the resolved
	//     architecture/chainId. A data race, so which chain won was scheduling.
	//
	// Bumping this is what actually ships them: the gateway builds from this
	// SHA, so leaving it behind keeps building the unfixed source and reports
	// success either way. See ERPCImageTag — the tag carries the SHA, so a
	// bump yields a distinct image instead of silently reusing the old one.
	ERPCSourceRepo = "https://github.com/valve-tech/erpc.git"
	ERPCSourceRef  = "a7a53ec21a7922c4c6d8582e3466331b1a7cc622"

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

	// ERPCContainerMetricsPort is the port eRPC serves its Prometheus counters
	// on INSIDE the container, fixed at eRPC's own default for the same reason
	// ERPCContainerPort is: the container's port namespace is private, so there
	// is nothing to avoid colliding with, and the operator's choice is expressed
	// on the host side of the -p mapping.
	ERPCContainerMetricsPort = 4001

	// DockerHostAlias is the DNS name that resolves to the container's host
	// from inside a container. It is built in on Docker Desktop, OrbStack,
	// colima and Rancher Desktop; on plain Linux engines it only exists
	// when the container was started with the --add-host host-gateway
	// mapping ERPCRunSpec.AddHostGateway emits.
	DockerHostAlias = "host.docker.internal"
)

// ERPCContainerNameFor is the container name for ONE gateway, derived from
// its id so that N gateways can coexist on one machine.
//
// The default id (and the empty one, which only a caller that has not been
// told about ids can produce) maps to the bare historical name. Everything
// else gets "valve-node-app-erpc-<id>". Two gateways can therefore never
// collide on a name, which is the concrete thing that made the old
// single-constant model unable to host more than one: `docker run --name`
// fails on a duplicate, so a second gateway did not merely misbehave, it
// could not be created.
//
// The id is sanitized rather than trusted: it lands in a docker --name, and
// docker's name grammar is [a-zA-Z0-9][a-zA-Z0-9_.-]*. Anything outside it
// becomes '-', so a hostile or merely sloppy id cannot smuggle a flag or a
// shell metacharacter into a command line. The API validates ids on the way
// in as well; this is the second lock on the same door, at the point where
// the value is actually used.
func ERPCContainerNameFor(gatewayID string) string {
	id := strings.TrimSpace(gatewayID)
	if id == "" || id == DefaultGatewayID {
		return ERPCContainerName
	}
	return ERPCContainerName + "-" + sanitizeNameSegment(id)
}

// sanitizeNameSegment maps an arbitrary string onto docker's name grammar.
func sanitizeNameSegment(s string) string {
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
	//
	// It is no longer the DEFAULT way to reach a same-host service — the
	// private network is (see Network) — but it does not go away: a node
	// installed by systemd on the same box is not a container and has no
	// container name, and neither does anything else the operator points an
	// upstream at.
	AddHostGateway bool

	// Network is the docker network to join ("" → none). Joining one is what
	// lets this container address its same-host neighbours BY CONTAINER NAME,
	// which in turn is what lets those neighbours publish no host port at all.
	Network string

	// NoPublish suppresses the RPC -p mapping entirely. It is set when something
	// else is the front door — a Caddy container terminating TLS — and eRPC is
	// reached only over Network. A gateway with no published port and no front
	// is unreachable, so this is never set on its own.
	//
	// It does NOT suppress MetricsPort; see there for why.
	NoPublish bool

	// MetricsPort is the HOST port the gateway's Prometheus counters are
	// published on (0 → not published at all, which is what a gateway with
	// metrics turned off gets).
	//
	// Two things about it differ deliberately from the RPC port, and both are
	// the reason it is a separate field rather than a second use of BindAddr:
	//
	//   - It is ALWAYS pinned to 127.0.0.1, whatever BindAddr says. BindAddr
	//     exists so an operator can expose the gateway on a Tailscale IP or a
	//     LAN address, which is a thing they choose for a front door. A request
	//     counter is not a front door, and nothing outside the machine has any
	//     business reading it, so the widening knob simply does not apply here.
	//
	//   - It is published even when NoPublish is set. A fronted gateway
	//     publishes nothing for RPC on purpose, because a plaintext,
	//     unauthenticated RPC port alongside the HTTPS one would be a second way
	//     in that the operator did not ask for. A read-only counter endpoint on
	//     loopback is a far smaller door, and without it the recommended
	//     configuration — HTTPS on — would be the one configuration that could
	//     never show where its traffic is going.
	MetricsPort int

	// LoopbackRPCPort is the HOST port eRPC's RPC endpoint (ERPCContainerPort)
	// is published on. Like MetricsPort it is ALWAYS pinned to 127.0.0.1 and is
	// published even when NoPublish is set, and for a closely related reason.
	//
	// A fronted gateway sets NoPublish so its only network-facing door is
	// Caddy's HTTPS port. But a wallet on the SAME machine then has to trust the
	// internal-CA certificate before it can connect, and trusting that CA needs
	// sudo or a GUI prompt — a step that fails outright when the app runs
	// detached (measured: SecTrustSettingsSetTrustSettings refuses with "no user
	// interaction was possible"). A plaintext RPC port on 127.0.0.1 removes the
	// step entirely: the port is unreachable from any other machine — loopback
	// is not a network door — wallets accept http://127.0.0.1 as a secure
	// context, and the operator gets a paste-into-your-wallet URL with nothing
	// to install.
	//
	// 0 publishes nothing, which is what an UNFRONTED gateway passes: its RPC
	// port is already on the host through the ordinary mapping above, so a
	// second loopback mapping to the same container port would only collide.
	// The caller therefore sets this ONLY for a fronted gateway — exactly where
	// the ordinary mapping is suppressed.
	LoopbackRPCPort int
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

	args := []string{
		"run", "-d",
		"--name", name,
		"--restart", "unless-stopped",
		// ALWAYS emitted, never conditionally: an absent --platform is not
		// "let the engine decide", it is "let DOCKER_DEFAULT_PLATFORM decide".
		// See resolveRunPlatform.
		"--platform", resolveRunPlatform(spec.Platform),
	}
	if spec.AddHostGateway {
		args = append(args, "--add-host", DockerHostAlias+":host-gateway")
	}
	if net := strings.TrimSpace(spec.Network); net != "" {
		// --network-alias is not needed: docker's embedded DNS resolves a
		// container's --name on a user-defined network already, and the name is
		// what every rendered upstream uses.
		args = append(args, "--network", net)
	}
	if !spec.NoPublish {
		args = append(args, "-p", publishSpec(bind, hostPort, ERPCContainerPort))
	}
	// Loopback literally, not `bind`. See ERPCRunSpec.MetricsPort.
	if spec.MetricsPort > 0 {
		args = append(args, "-p", publishSpec("127.0.0.1", spec.MetricsPort, ERPCContainerMetricsPort))
	}
	// The plaintext wallet door for the same machine, loopback literally, and
	// published even for a fronted gateway. See ERPCRunSpec.LoopbackRPCPort.
	if spec.LoopbackRPCPort > 0 {
		args = append(args, "-p", publishSpec("127.0.0.1", spec.LoopbackRPCPort, ERPCContainerPort))
	}
	args = append(args,
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
	// The metrics listener is widened for exactly the same reason the RPC one
	// is, and only that reason: a container's port namespace is private, so a
	// listener bound to the container's loopback is unreachable even from the
	// -p mapping pointed at it. What decides who can actually read the counters
	// is the HOST side of that mapping, which ERPCRunSpec.MetricsPort pins to
	// 127.0.0.1 unconditionally.
	out.MetricsBindAddr = "0.0.0.0"
	out.MetricsPort = ERPCContainerMetricsPort

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
// the private network
// ---------------------------------------------------------------------

// NetworkName is the user-defined bridge network every container this app
// places on ONE target joins: the devnet, each gateway's eRPC, and each
// gateway's TLS front.
//
// It is a single constant rather than a name derived from the target id, and
// that already IS "one network per placement target": a target is a docker
// ENGINE, and each engine has its own namespace of networks. Deriving the name
// from the id would produce the same one network per engine under a longer
// name, and would break the moment a target was renamed.
//
// WHY a user-defined network at all, rather than the default bridge: docker
// runs an embedded DNS server on user-defined networks and NOT on the default
// bridge, so a container's --name resolves to its address only here. That name
// resolution is the whole mechanism — it is what lets an upstream be written as
// "valve-node-app-devnet:8545" instead of a published host port reached back
// through host.docker.internal, and therefore what lets the devnet and eRPC
// publish nothing at all.
const NetworkName = "valve-node-app"

// NetworkInspectArgs / NetworkCreateArgs render the two halves of an
// idempotent network creation, kept pure and separate so the argv is testable
// without a daemon, matching every other renderer here.
func NetworkInspectArgs(name string) []string { return []string{"network", "inspect", name} }
func NetworkCreateArgs(name string) []string  { return []string{"network", "create", name} }

// EnsureNetwork creates the network if it is not already there, and is safe to
// call on every provision.
//
// Inspect-then-create rather than create-and-ignore-the-error because the
// common path (the network already exists) should not produce an error line in
// a provisioning log the operator is reading. The "already exists" case is
// still tolerated on the create, because two provisions racing on one target is
// possible and losing that race is not a failure.
func EnsureNetwork(ctx context.Context, e executor.Executor, name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("ops: network: name is empty")
	}
	res, err := DockerRun(ctx, e, NetworkInspectArgs(name)...)
	if err != nil {
		return err
	}
	if res.ExitCode == 0 {
		return nil
	}

	res, err = DockerRun(ctx, e, NetworkCreateArgs(name)...)
	if err != nil {
		return err
	}
	if res.ExitCode != 0 && !isNetworkExists(res.Stderr, res.Stdout) {
		return fmt.Errorf("ops: docker network create %s failed (exit %d): %s",
			name, res.ExitCode, firstNonEmptyLine(res.Stderr, res.Stdout))
	}
	return nil
}

// isNetworkExists recognizes the engine's "you already made that" complaint,
// which is the one non-zero exit an idempotent create must treat as success.
func isNetworkExists(streams ...string) bool {
	hay := strings.ToLower(strings.Join(streams, "\n"))
	return strings.Contains(hay, "already exists")
}

// ---------------------------------------------------------------------
// the TLS front (Caddy)
// ---------------------------------------------------------------------

const (
	// CaddyContainerName is the DEFAULT gateway's TLS front, and the prefix
	// every other gateway's front is named from — exactly the arrangement
	// ERPCContainerName documents, and for the same reason: one gateway may
	// have a TLS front while another does not, and two containers cannot share
	// a name.
	CaddyContainerName = "valve-node-app-caddy"

	// caddyContainerConfigPath is where the rendered Caddyfile is mounted. It
	// is the path the official image's own CMD already points at, so the mount
	// alone is what makes this work — no command override, which is precisely
	// the mistake ERPCRunArgs documents at length.
	caddyContainerConfigPath = "/etc/caddy/Caddyfile"

	// CaddyHTTPSPort is the port Caddy listens on INSIDE the container. Fixed,
	// like ERPCContainerPort: the container's port namespace is private, and
	// the operator's choice of port lives on the host side of the -p mapping.
	CaddyHTTPSPort = 443
)

// CaddyContainerNameFor is the TLS front's container name for ONE gateway,
// derived from the gateway id exactly as ERPCContainerNameFor is.
func CaddyContainerNameFor(gatewayID string) string {
	id := strings.TrimSpace(gatewayID)
	if id == "" || id == DefaultGatewayID {
		return CaddyContainerName
	}
	return CaddyContainerName + "-" + sanitizeNameSegment(id)
}

// CaddyRunSpec is everything the pure Caddy arg renderer needs.
type CaddyRunSpec struct {
	// Image is the container image ref ("" → catalog's caddy default).
	Image string
	// ContainerName is the container's --name ("" → CaddyContainerName).
	ContainerName string
	// BindAddr is the HOST address the published TLS port binds to ("" →
	// 0.0.0.0). Unlike eRPC's, this one defaults WIDE, and that is the point of
	// the whole feature: an https:// page cannot call an http:// endpoint, and
	// a TLS front bound to loopback would serve only the machine it runs on —
	// which is the one machine that never needed it.
	BindAddr string
	// HostPort is the host-side TLS port (0 → CaddyHTTPSPort).
	HostPort int
	// Platform is the image platform (""→ DefaultPlatform()), always emitted.
	Platform string
	// HostConfigPath is the absolute path to the rendered Caddyfile ON THE
	// HOST, bind-mounted read-only.
	HostConfigPath string
	// Network is the docker network to join. It is effectively mandatory: the
	// upstream in the rendered Caddyfile is a CONTAINER NAME, which resolves
	// only on a user-defined network.
	Network string
	// DataVolume is the named volume mounted at DataPath ("" →
	// catalog.CaddyDataVolume). It is not optional — Caddy's internal CA lives
	// there, and without persistence it is regenerated on every container
	// recreate, invalidating every trust-store install of the old root. See
	// catalog.CaddyDataVolume.
	DataVolume string
	// CertFile and KeyFile are host paths mounted read-only when the cert
	// source is "files". Empty otherwise.
	CertFile string
	KeyFile  string
}

// CaddyRunArgs renders the argv for `docker run` — WITHOUT the leading
// "docker" — for a gateway's TLS front. Pure, like every other renderer here.
//
// --restart unless-stopped, matching eRPC: a TLS front is infrastructure, and
// an https endpoint that silently stops existing after a reboot is worse than
// one that was never offered.
func CaddyRunArgs(spec CaddyRunSpec) []string {
	name := spec.ContainerName
	if name == "" {
		name = CaddyContainerName
	}
	image := spec.Image
	if image == "" {
		image = catalog.DefaultCaddyImage
	}
	bind := spec.BindAddr
	if bind == "" {
		bind = "0.0.0.0"
	}
	hostPort := spec.HostPort
	if hostPort == 0 {
		hostPort = CaddyHTTPSPort
	}
	volume := spec.DataVolume
	if volume == "" {
		volume = catalog.CaddyDataVolume
	}

	args := []string{
		"run", "-d",
		"--name", name,
		"--restart", "unless-stopped",
		"--platform", resolveRunPlatform(spec.Platform),
	}
	if net := strings.TrimSpace(spec.Network); net != "" {
		args = append(args, "--network", net)
	}
	args = append(args,
		"-p", publishSpec(bind, hostPort, CaddyHTTPSPort),
		"-v", spec.HostConfigPath+":"+caddyContainerConfigPath+":ro",
		// The data volume is unconditional. See catalog.CaddyDataVolume: a
		// regenerated internal CA breaks HTTPS for every device that trusted
		// the old root, and the operator has no way to know why.
		"-v", volume+":"+catalog.CaddyDataPath,
	)
	// Cert and key are mounted at the SAME paths inside the container as on
	// the host, so the Caddyfile can name one path that is true on both sides.
	if spec.CertFile != "" && spec.KeyFile != "" {
		args = append(args,
			"-v", spec.CertFile+":"+spec.CertFile+":ro",
			"-v", spec.KeyFile+":"+spec.KeyFile+":ro",
		)
	}
	return append(args, image)
}

// CaddyServiceFor is the lifecycle descriptor for ONE gateway's TLS front.
//
// It DOES declare a volume — catalog.CaddyDataVolume — and that is exactly why
// WipeService must not be pointed at this descriptor casually: wiping it
// destroys the internal CA, and every browser and trust store that installed
// the old root then rejects the gateway with no indication of why. The gateway
// wipe path deliberately uses CaddyServiceKeepingCA instead; this constructor
// exists for the one case that genuinely means "throw the CA away too".
func CaddyServiceFor(gatewayID string) DockerService {
	s := CaddyServiceKeepingCA(gatewayID)
	s.Volumes = []string{catalog.CaddyDataVolume}
	return s
}

// CaddyServiceKeepingCA is the same front WITHOUT its data volume declared, so
// a wipe removes the container and leaves the certificate authority alone.
//
// This is the default the gateway lifecycle uses. Wiping chain data is
// routine; invalidating every trust-store install on every device the operator
// owns is not, and the two must not be reachable by the same button.
func CaddyServiceKeepingCA(gatewayID string) DockerService {
	id := strings.TrimSpace(gatewayID)
	if id == "" {
		id = DefaultGatewayID
	}
	return DockerService{
		ID:            "caddy:" + id,
		ContainerName: CaddyContainerNameFor(id),
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
	// DefaultPlatform()). It is ALWAYS emitted, for the same reason
	// ERPCRunArgs always emits it: an omitted --platform on a build inherits
	// DOCKER_DEFAULT_PLATFORM, which on an arm64 machine with that variable
	// set to linux/amd64 produces an emulated image that then runs — slowly,
	// and reporting itself healthy — for the rest of its life.
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
	args := []string{"build", "--platform", resolveRunPlatform(spec.Platform)}
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
