package ops

// One lifecycle surface for every docker-backed service valve-node-app runs:
// start / stop / restart / wipe / status, over executor.Executor, local or
// SSH alike.
//
// WHY this exists as a surface rather than as four more helpers in docker.go:
//
//  1. There is now more than one docker-backed service (the eRPC gateway and
//     the local devnet), and they are not independent — one sits in front of
//     the other. A per-service pile of stop/rm calls cannot express that, so
//     the relationship ends up living in whichever caller happens to
//     remember it. See WipeService for the measured, silent, wrong behaviour
//     that produced when it was forgotten.
//
//  2. The systemd side of the app already has this shape — ServiceAction
//     ("start"|"stop"|"restart", read the state back afterwards) and
//     ClearService (stop, delete exactly this service's own data, bring it
//     back). The vocabulary here is deliberately the same, so a caller
//     driving a unit-hosted node and a container-hosted gateway is using one
//     API with two backends rather than two APIs.
//
// Rendering and execution stay separate, as in docker.go: every command this
// file issues comes from a pure ...Args func that a test can assert on
// without a daemon anywhere.

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/valve-tech/valve-node-app/internal/executor"
)

// ---------------------------------------------------------------------
// states
// ---------------------------------------------------------------------

// The three states a docker-backed service can be in, plus the honest fourth
// one for "the engine did not answer, so we do not know".
//
// not-created and created-but-stopped are kept apart because they need
// completely different operator prompts: a stopped container is one `start`
// away, while an absent one has never been provisioned (or was wiped) and
// needs the setup plan run. Collapsing them into "not running" is exactly
// the reading that makes a UI offer a Start button that cannot work.
//
// The values are stable identifiers, safe to serialize to the UI.
const (
	StateNotCreated = "not-created"
	StateStopped    = "created-but-stopped"
	StateRunning    = "running"
	StateUnknown    = "unknown"
)

// ContainerStatus is one point-in-time reading of a docker-backed service.
type ContainerStatus struct {
	// ID and ContainerName echo the DockerService this reading is of, so a
	// status can travel to a UI on its own.
	ID            string
	ContainerName string

	// State is one of the State* constants.
	State string

	// Image is the image reference the container was actually created from
	// (docker inspect's .Config.Image) — NOT the image the app would build
	// or run today. The difference is the whole point: after bumping
	// ERPCSourceRef, a running gateway still carries the OLD tag here, which
	// is how "this container predates your config change" becomes visible
	// instead of being guessed at.
	Image string
	// ImageID is the resolved image digest (.Image). It disambiguates the
	// case Image cannot: a mutable tag rebuilt in place.
	ImageID string

	// ExitCode is the container's last exit status, meaningful when State is
	// StateStopped (137 = SIGKILL/OOM, 0 = a deliberate stop).
	ExitCode int

	// Detail carries the engine's own words when State is StateUnknown.
	Detail string
}

// Running reports whether this service is up right now.
func (s ContainerStatus) Running() bool { return s.State == StateRunning }

// Exists reports whether the container exists in any state. It is the
// question "can I start this, or must it be provisioned first?".
func (s ContainerStatus) Exists() bool {
	return s.State == StateRunning || s.State == StateStopped
}

// ---------------------------------------------------------------------
// service descriptors
// ---------------------------------------------------------------------

// CreateFunc provisions a service's container from scratch. It is what
// `docker run` needs and this package cannot supply: an image, a platform,
// a published port and a config path, all of which are decided by the setup
// plan (internal/setup/gateway.go's runDocker, and the devnet equivalent).
//
// A service with a nil Create can be started, stopped, restarted and wiped,
// but never created — attempting to start an absent one yields
// ErrServiceNotCreated, which says "run setup", rather than relaying
// docker's "No such container".
type CreateFunc func(ctx context.Context, e executor.Executor) error

// DockerService declares one docker-backed service's lifecycle identity:
// what its container is called, what data it OWNS, and what sits in front of
// it. It is a plain value, constructed by whoever owns the service, so this
// package needs no registry and no init-time registration.
type DockerService struct {
	// ID is a short stable identifier used in errors, reports and API paths
	// ("erpc", "devnet"). It is the container-world equivalent of ops.go's
	// svc ("exec"|"beacon").
	ID string

	// ContainerName is the container's --name. Stability is the entire
	// mechanism by which any of this works: it is how status/stop/remove
	// find the container across app restarts (see ERPCContainerName).
	ContainerName string

	// Volumes are the DOCKER NAMED VOLUMES this service owns, and WipeService
	// deletes every one of them. Two deliberate constraints:
	//
	//   - Only named volumes belong here. Validate refuses anything that is
	//     not a legal docker volume name, so a host path can never appear:
	//     `docker volume rm /var/lib/x` would not delete that directory, but
	//     a descriptor written that way means its author BELIEVED it would,
	//     and that belief is how a wipe ends up quietly not wiping (or, with
	//     a different implementation, quietly wiping far too much).
	//
	//   - Bind mounts are never listed and are never touched. The gateway's
	//     erpc.yaml is a read-only bind mount the operator owns; wiping the
	//     gateway must not delete it. Data that lives only in the container's
	//     own filesystem needs no entry either — removing the container
	//     removes it, and `docker rm -v` takes any ANONYMOUS volumes with it.
	Volumes []string

	// FrontedBy declares the services that sit IN FRONT of this one — a
	// gateway proxying it, typically. WipeService restarts every one of them
	// after a wipe. This is a declared dependency rather than something the
	// caller passes at the call site precisely because callers forget; see
	// WipeService for what forgetting looks like in production.
	//
	// It is followed transitively (a front's own fronts are restarted too)
	// and is cycle-safe.
	FrontedBy []DockerService

	// Create optionally provisions the container; see CreateFunc.
	Create CreateFunc
}

// ERPCService is the lifecycle descriptor for the eRPC gateway container.
//
// It owns NO volumes: the gateway is stateless by design (docker.go's
// header) — its config is a read-only bind mount and it keeps nothing else
// on disk — so wiping it removes a container and nothing more. It has no
// Create either, because creating the gateway container needs the rendered
// erpc.yaml path, the resolved platform and the built image, all of which
// internal/setup's gateway plan owns.
//
// Nothing is declared in FrontedBy: the gateway is the thing that fronts
// others, not the thing fronted. A service the gateway proxies names it in
// its own FrontedBy — e.g. DockerService{ID: "devnet", ..., FrontedBy:
// []DockerService{ops.ERPCService()}} — which is what makes a devnet wipe
// bounce the gateway.
func ERPCService() DockerService {
	return DockerService{
		ID:            "erpc",
		ContainerName: ERPCContainerName,
	}
}

// Validate rejects a descriptor that could make a lifecycle call do
// something other than what it says. It is called at the top of every
// operation, so a bad descriptor fails BEFORE the first destructive command
// rather than midway through one.
func (s DockerService) Validate() error {
	return validateService(s, map[string]bool{})
}

func validateService(s DockerService, seen map[string]bool) error {
	if strings.TrimSpace(s.ID) == "" {
		return fmt.Errorf("ops: docker service: ID is empty")
	}
	if strings.TrimSpace(s.ContainerName) == "" {
		return fmt.Errorf("ops: docker service %q: ContainerName is empty (every lifecycle call addresses the container by name)", s.ID)
	}
	for _, v := range s.Volumes {
		if !validVolumeName(v) {
			return fmt.Errorf("ops: docker service %q: %q is not a docker named volume — a wipe deletes every volume listed here, so a path or an odd string is refused rather than guessed at", s.ID, v)
		}
	}
	// Cycles are tolerated, not rejected: the cascade tracks what it has
	// visited, so a loop is harmless, and refusing one would only make an
	// odd-but-workable topology unusable.
	if seen[s.ContainerName] {
		return nil
	}
	seen[s.ContainerName] = true
	for _, f := range s.FrontedBy {
		if err := validateService(f, seen); err != nil {
			return fmt.Errorf("ops: docker service %q: fronting service: %w", s.ID, err)
		}
	}
	return nil
}

// validVolumeName reports whether v is a legal docker volume name
// ([a-zA-Z0-9][a-zA-Z0-9_.-]*). Deliberately structural rather than a
// denylist: a name matching this cannot contain a slash, so it cannot be
// mistaken for — or silently stand in for — a host path.
func validVolumeName(v string) bool {
	if v == "" || len(v) > 255 {
		return false
	}
	for i, r := range v {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		case (r == '_' || r == '.' || r == '-') && i > 0:
		default:
			return false
		}
	}
	return true
}

// ---------------------------------------------------------------------
// errors
// ---------------------------------------------------------------------

// ErrDockerUnreachable is the sentinel every "the docker CLI is there but no
// engine answered" failure unwraps to. It is separate from ErrDockerAbsent
// for the same reason DockerInfo keeps Present and DaemonReachable apart:
// "install docker" and "start Docker Desktop" are different operator
// actions, and a status call that conflated them would send half of all
// desktop users down the wrong path.
var ErrDockerUnreachable = errors.New("ops: no docker engine answered on the target")

// DockerUnreachableError is the typed form, carrying the engine's own
// complaint so the UI can show it verbatim.
type DockerUnreachableError struct {
	// Probe is the command whose non-zero exit established unreachability.
	Probe string
	// ExitCode is that command's exit status.
	ExitCode int
	// Detail is the engine's own words, e.g. "Cannot connect to the Docker
	// daemon at unix:///var/run/docker.sock".
	Detail string
	// Hint is operator-facing guidance, safe to show as-is.
	Hint string
}

func (e *DockerUnreachableError) Error() string {
	return fmt.Sprintf("%s (%s exited %d): %s — %s", ErrDockerUnreachable.Error(), e.Probe, e.ExitCode, e.Detail, e.Hint)
}

func (e *DockerUnreachableError) Unwrap() error { return ErrDockerUnreachable }

const dockerDaemonHint = "start the engine and retry: Docker Desktop / OrbStack / colima on a desktop, " +
	"or `systemctl start docker` on Linux"

// ErrServiceNotCreated is the sentinel for "this service's container does
// not exist, and this package cannot create it". It is a distinct outcome
// from a failed start: the fix is to run provisioning, not to retry.
var ErrServiceNotCreated = errors.New("ops: the service's container does not exist on the target")

// ServiceNotCreatedError is the typed form.
type ServiceNotCreatedError struct {
	ID            string
	ContainerName string
	// Action is what was attempted ("start"|"restart").
	Action string
}

func (e *ServiceNotCreatedError) Error() string {
	return fmt.Sprintf("ops: cannot %s %s: container %q does not exist — run setup for this service first (creating it needs an image, a platform and a config path this package does not have)",
		e.Action, e.ID, e.ContainerName)
}

func (e *ServiceNotCreatedError) Unwrap() error { return ErrServiceNotCreated }

// ---------------------------------------------------------------------
// pure argv rendering
// ---------------------------------------------------------------------

// containerInspectFormat is the one-line reading ServiceStatus parses. The
// fields are fixed-shape (no free text) and pipe-separated, so parsing
// cannot be derailed by an image name containing a separator — .Config.Image
// is the only field an operator influences and it cannot contain a pipe.
const containerInspectFormat = "{{.State.Running}}|{{.State.ExitCode}}|{{.Config.Image}}|{{.Image}}"

// ContainerInspectArgs renders the argv for the status probe (without the
// leading "docker"). One inspect answers state, exit code and image at once:
// three round trips over SSH for what the engine already has in one record
// would be three chances for the readings to disagree with each other.
func ContainerInspectArgs(name string) []string {
	return []string{"inspect", "-f", containerInspectFormat, name}
}

// ContainerActionArgs renders `start`/`stop`/`restart` for a container.
//
// restart is a single engine call rather than stop-then-start on purpose:
// it is atomic from the engine's point of view and it keeps the window in
// which a gateway sees a dead upstream as short as the engine can make it.
func ContainerActionArgs(action, name string) []string {
	return []string{action, name}
}

// ContainerRemoveArgs renders the wipe's container removal.
//
// -f stops it first (a wipe's goal state does not care whether it was
// running). -v removes the container's ANONYMOUS volumes — the ones docker
// created implicitly for VOLUME directives in the image, which nothing else
// will ever reference again and which would otherwise accumulate as
// invisible disk. It does NOT remove named volumes; those are removed
// explicitly, by name, from DockerService.Volumes, which is what keeps the
// blast radius of a wipe equal to what the descriptor declares.
func ContainerRemoveArgs(name string) []string {
	return []string{"rm", "-f", "-v", name}
}

// VolumeRemoveArgs renders the removal of ONE named volume. One call per
// volume, never `docker volume prune`: prune deletes every unused volume on
// the target, including volumes belonging to things valve-node-app did not
// create.
func VolumeRemoveArgs(name string) []string {
	return []string{"volume", "rm", name}
}

// parseInspectStatus splits containerInspectFormat's line. Missing trailing
// fields come back zero-valued rather than erroring, matching
// parseDockerInfo: a docker that cannot render one of these templates should
// still yield a usable partial reading.
func parseInspectStatus(stdout string) (running bool, exitCode int, image, imageID string) {
	fields := strings.SplitN(firstNonEmptyLine(stdout), "|", 4)
	get := func(i int) string {
		if i < len(fields) {
			return strings.TrimSpace(fields[i])
		}
		return ""
	}
	exitCode, _ = strconv.Atoi(get(1))
	return get(0) == "true", exitCode, get(2), get(3)
}

// The four kinds of non-zero docker exit this file must tell apart. Each one
// gets a different answer: a reading, an install prompt, a start-the-engine
// prompt, or a genuine error.
const (
	dockerFailureAbsentContainer   = "no-such-container"
	dockerFailureDaemonUnreachable = "daemon-unreachable"
	dockerFailureCLIMissing        = "cli-missing"
	dockerFailureOther             = "other"
)

// classifyDockerFailure reads a failed docker invocation's exit code and
// output. Pure, because the matching is (inevitably) heuristic string work
// against several engines' phrasings and is worth testing against the real
// strings directly.
//
// Order matters: absence is checked first because "No such container" can
// appear alongside other noise, and the CLI-missing check keys on the shell's
// 127 as well as its wording, since the message differs per shell
// ("command not found" in bash, "not found" in dash/ash).
func classifyDockerFailure(exitCode int, stderr, stdout string) string {
	hay := strings.ToLower(stderr + "\n" + stdout)
	switch {
	case strings.Contains(hay, "no such container"),
		strings.Contains(hay, "no such object"):
		return dockerFailureAbsentContainer
	case strings.Contains(hay, "command not found"),
		strings.Contains(hay, "executable file not found"),
		strings.Contains(hay, "docker: not found"),
		exitCode == 127:
		return dockerFailureCLIMissing
	case strings.Contains(hay, "cannot connect to the docker daemon"),
		strings.Contains(hay, "is the docker daemon running"),
		strings.Contains(hay, "error during connect"),
		strings.Contains(hay, "cannot connect to podman"):
		return dockerFailureDaemonUnreachable
	default:
		return dockerFailureOther
	}
}

// isNoSuchVolume recognizes the engine's "that volume isn't there"
// complaint, which is the one non-zero exit a wipe treats as success —
// wiping something already gone is the goal state, not a failure.
func isNoSuchVolume(streams ...string) bool {
	hay := strings.ToLower(strings.Join(streams, "\n"))
	return strings.Contains(hay, "no such volume") || strings.Contains(hay, "volume not found")
}

// ---------------------------------------------------------------------
// status
// ---------------------------------------------------------------------

// ServiceStatus reads s's current state, the image actually in use, and (for
// a stopped container) why it stopped.
//
// An ABSENT container is a reading, not an error — the same call an operator
// uses to decide whether to provision must be able to say "nothing here".
// An unreachable engine IS an error (errors.Is(err, ErrDockerUnreachable)),
// because the honest answer is "unknown", and a caller that treated unknown
// as not-created would go on to provision on top of a service that may well
// be running. The returned status is still populated in that case
// (StateUnknown plus the engine's words in Detail) for display.
func ServiceStatus(ctx context.Context, e executor.Executor, s DockerService) (ContainerStatus, error) {
	st := ContainerStatus{ID: s.ID, ContainerName: s.ContainerName, State: StateUnknown}
	if err := s.Validate(); err != nil {
		return st, err
	}

	args := ContainerInspectArgs(s.ContainerName)
	res, err := DockerRun(ctx, e, args...)
	if err != nil {
		// A transport failure (SSH dropped, context canceled) is never a
		// docker verdict; DockerRun has already wrapped it with the command.
		return st, err
	}
	if res.ExitCode != 0 {
		probe := "docker " + strings.Join(args, " ")
		detail := firstNonEmptyLine(res.Stderr, res.Stdout)
		switch classifyDockerFailure(res.ExitCode, res.Stderr, res.Stdout) {
		case dockerFailureAbsentContainer:
			st.State = StateNotCreated
			return st, nil
		case dockerFailureCLIMissing:
			return st, &DockerAbsentError{Probe: probe, ExitCode: res.ExitCode, Hint: dockerInstallHint}
		case dockerFailureDaemonUnreachable:
			st.Detail = detail
			return st, &DockerUnreachableError{Probe: probe, ExitCode: res.ExitCode, Detail: detail, Hint: dockerDaemonHint}
		default:
			st.Detail = detail
			return st, fmt.Errorf("ops: %s failed (exit %d): %s", probe, res.ExitCode, detail)
		}
	}

	running, code, image, imageID := parseInspectStatus(res.Stdout)
	st.ExitCode, st.Image, st.ImageID = code, image, imageID
	if running {
		st.State = StateRunning
	} else {
		st.State = StateStopped
	}
	return st, nil
}

// ---------------------------------------------------------------------
// published ports
// ---------------------------------------------------------------------

// portsInspectFormat renders one line per published container port as
// "<containerPort>=<hostIp>:<hostPort>". Rendered by the engine rather than
// parsed out of `{{json .NetworkSettings.Ports}}` because the JSON shape
// (a map of "8545/tcp" to a list of bindings, null when unpublished) needs
// three levels of unmarshalling to answer a one-line question.
const portsInspectFormat = `{{range $p, $bindings := .NetworkSettings.Ports}}{{range $bindings}}{{$p}}={{.HostIp}}:{{.HostPort}}{{println}}{{end}}{{end}}`

// ContainerPortsArgs renders the argv for the published-port probe.
func ContainerPortsArgs(name string) []string {
	return []string{"inspect", "-f", portsInspectFormat, name}
}

// PortBinding is one published port's host side: the address docker bound it
// to and the port it bound there.
type PortBinding struct {
	HostIP   string
	HostPort int
}

// PublishedPorts reports where a container's ports are ACTUALLY published on
// the host, keyed by the container-side port.
//
// It exists because a service's configuration and a running container are two
// different facts, and only one of them is what a caller can dial. A
// container's -p mapping is fixed at creation, so a config edited since then
// describes a devnet that does not exist yet — and showing an operator a URL
// derived from it is showing them a port nothing is listening on. Reading the
// mapping back off the container is the only way to print a URL that is true.
//
// An absent container yields an empty map and no error, matching
// ServiceStatus's treatment of absence as a reading. Ports with no host
// binding (EXPOSEd but not published) simply do not appear.
func PublishedPorts(ctx context.Context, e executor.Executor, name string) (map[int]PortBinding, error) {
	out := map[int]PortBinding{}
	res, err := DockerRun(ctx, e, ContainerPortsArgs(name)...)
	if err != nil {
		return out, err
	}
	if res.ExitCode != 0 {
		if classifyDockerFailure(res.ExitCode, res.Stderr, res.Stdout) == dockerFailureAbsentContainer {
			return out, nil
		}
		return out, fmt.Errorf("ops: docker inspect (ports) %s failed (exit %d): %s",
			name, res.ExitCode, firstNonEmptyLine(res.Stderr, res.Stdout))
	}
	return parsePublishedPorts(res.Stdout), nil
}

// parsePublishedPorts reads portsInspectFormat's output. Pure, so the (fiddly)
// splitting is testable without a daemon.
//
// A container port published on several host addresses (docker emits one line
// per binding, e.g. an IPv4 and an IPv6 one) keeps the FIRST binding: they are
// the same port on the same host, and a caller needs one URL, not a list.
//
// The host address is split off at the LAST colon, not the first: docker
// writes a wildcard IPv6 binding as "::" and splitting that at the first colon
// would yield an empty address and a port of ":4000".
func parsePublishedPorts(stdout string) map[int]PortBinding {
	out := map[int]PortBinding{}
	for _, line := range strings.Split(stdout, "\n") {
		key, value, ok := strings.Cut(strings.TrimSpace(line), "=")
		if !ok {
			continue
		}
		// "8545/tcp" → 8545. A udp mapping is kept too; the caller asks by
		// number and this app publishes no port on both protocols.
		port, err := strconv.Atoi(strings.SplitN(key, "/", 2)[0])
		if err != nil {
			continue
		}
		cut := strings.LastIndex(value, ":")
		if cut < 0 {
			continue
		}
		hostPort, err := strconv.Atoi(strings.TrimSpace(value[cut+1:]))
		if err != nil {
			continue
		}
		if _, seen := out[port]; seen {
			continue
		}
		out[port] = PortBinding{HostIP: strings.TrimSpace(value[:cut]), HostPort: hostPort}
	}
	return out
}

// ---------------------------------------------------------------------
// start / stop / restart
// ---------------------------------------------------------------------

// ContainerAction runs "start", "stop" or "restart" against s's container
// and returns the state read back AFTERWARDS — the container-world twin of
// ServiceAction, which runs `systemctl <action>` and then reads
// `systemctl is-active`. An unrecognized action is rejected before anything
// touches the target, exactly as ServiceAction rejects one.
//
// The rules for a container that does not exist:
//
//   - stop is a no-op success. The goal state is "not running", and an
//     absent container already satisfies it — the same tolerance
//     StopContainer has, and the reason re-running a toggle-off is safe.
//   - start and restart create it via s.Create when the descriptor has one,
//     and otherwise fail with ErrServiceNotCreated. Relaying docker's "No
//     such container" here would tell the operator what happened but not
//     what to do about it.
//
// A restart of a STOPPED container starts it, which is docker's own
// behaviour and matches `systemctl restart` on an inactive unit.
func ContainerAction(ctx context.Context, e executor.Executor, s DockerService, action string) (ContainerStatus, error) {
	st := ContainerStatus{ID: s.ID, ContainerName: s.ContainerName, State: StateUnknown}
	if err := s.Validate(); err != nil {
		return st, err
	}
	switch action {
	case "start", "stop", "restart":
	default:
		return st, fmt.Errorf("ops: invalid action %q (want \"start\", \"stop\", or \"restart\")", action)
	}

	st, err := ServiceStatus(ctx, e, s)
	if err != nil {
		return st, err
	}

	if st.State == StateNotCreated {
		if action == "stop" {
			return st, nil
		}
		if s.Create == nil {
			return st, &ServiceNotCreatedError{ID: s.ID, ContainerName: s.ContainerName, Action: action}
		}
		if err := s.Create(ctx, e); err != nil {
			return st, fmt.Errorf("ops: %s: create container %s: %w", s.ID, s.ContainerName, err)
		}
		return ServiceStatus(ctx, e, s)
	}

	res, err := DockerRun(ctx, e, ContainerActionArgs(action, s.ContainerName)...)
	if err != nil {
		return st, err
	}
	if res.ExitCode != 0 {
		return st, fmt.Errorf("ops: docker %s %s failed (exit %d): %s",
			action, s.ContainerName, res.ExitCode, firstNonEmptyLine(res.Stderr, res.Stdout))
	}
	return ServiceStatus(ctx, e, s)
}

// ---------------------------------------------------------------------
// wipe
// ---------------------------------------------------------------------

// WipeReport is an exact account of what a wipe did. It exists because a
// wipe is the one irreversible operation here: the operator (and the UI)
// should be able to see precisely what was deleted, not infer it.
type WipeReport struct {
	ID            string
	ContainerName string

	// ContainerRemoved is false when there was no container to remove.
	ContainerRemoved bool
	// VolumesRemoved and VolumesAbsent partition DockerService.Volumes into
	// the ones this call deleted and the ones that were already gone.
	VolumesRemoved []string
	VolumesAbsent  []string
	// Recreated is true when the descriptor's Create hook rebuilt the
	// container afterwards.
	Recreated bool

	// Cascaded lists the IDs of fronting services restarted because of this
	// wipe; CascadeSkipped lists the ones that were not running and so had
	// no stale state to clear.
	Cascaded       []string
	CascadeSkipped []string
}

// WipeService removes s's container and every named volume s declares, so
// the next start is genuinely clean — then restarts everything declared in
// s.FrontedBy.
//
// WHAT IT DELETES, exactly:
//   - the container named s.ContainerName (`docker rm -f -v`), and with it
//     anything that lived only in the container's own filesystem plus any
//     ANONYMOUS volumes docker attached to it;
//   - each named volume in s.Volumes, one explicit `docker volume rm` each.
//
// WHAT IT NEVER TOUCHES: bind mounts (the operator's own files on the host —
// erpc.yaml among them), images, networks, and any volume not named in the
// descriptor. There is no `volume prune` and no `rm -rf` anywhere in this
// path, because a wipe that removes more than the operator expected is the
// worst bug this file could have.
//
// WHY IT CASCADES A RESTART — measured by hand, and the reason this function
// takes responsibility for it instead of documenting it as a caller duty:
//
// After wiping a devnet and letting it come back from genesis, the chain was
// at block 0x4 while the eRPC gateway in front of it still advertised 0x2c.
// eRPC keeps a monotonic highest-seen-block guard per network and refuses to
// let a head move backwards, so a chain reset is invisible to it — it does
// not observe an error, it observes an upstream that has stopped making
// progress. It recovers only once the fresh chain's height overtakes the old
// high-water mark: about 40s on a 2s-block devnet, and effectively never for
// a real chain resyncing from genesis.
//
// That is SILENTLY WRONG, not slow and not an error: the gateway advertises
// a head the chain does not have, so a caller that asks for that block gets
// nothing. The guard lives in process memory only, so restarting the gateway
// clears it — which is why the cascade is a restart and why it must happen
// after the fresh service exists, not before.
//
// A plain restart of s does NOT cascade, deliberately: restarting a service
// leaves its data (and therefore its height) where it was, so no head ever
// moves backwards. Only a wipe resets a chain.
//
// The cascade is skipped entirely when nothing was actually removed and
// nothing was recreated — wiping an already-absent service changes no head,
// so bouncing a healthy gateway would be pure noise.
func WipeService(ctx context.Context, e executor.Executor, s DockerService) (WipeReport, error) {
	rep := WipeReport{ID: s.ID, ContainerName: s.ContainerName}
	if err := s.Validate(); err != nil {
		return rep, err
	}

	// Read the state first. On a target whose engine is unreachable this
	// fails here, before anything destructive has been attempted — the
	// alternative, discovering it on the `docker rm`, leaves the operator
	// unsure whether a wipe half-happened.
	st, err := ServiceStatus(ctx, e, s)
	if err != nil {
		return rep, fmt.Errorf("ops: wipe %s: %w", s.ID, err)
	}

	if st.Exists() {
		args := ContainerRemoveArgs(s.ContainerName)
		res, err := DockerRun(ctx, e, args...)
		if err != nil {
			return rep, fmt.Errorf("ops: wipe %s: %w", s.ID, err)
		}
		// A container that vanished between the status read and this call is
		// success: the goal state is "gone".
		if res.ExitCode != 0 && classifyDockerFailure(res.ExitCode, res.Stderr, res.Stdout) != dockerFailureAbsentContainer {
			return rep, fmt.Errorf("ops: wipe %s: docker %s failed (exit %d): %s",
				s.ID, strings.Join(args, " "), res.ExitCode, firstNonEmptyLine(res.Stderr, res.Stdout))
		}
		rep.ContainerRemoved = true
	}

	// Named volumes come after the container: docker refuses to remove a
	// volume that a container still references, so the reverse order would
	// fail on exactly the services that have data worth wiping.
	for _, v := range s.Volumes {
		res, err := DockerRun(ctx, e, VolumeRemoveArgs(v)...)
		if err != nil {
			return rep, fmt.Errorf("ops: wipe %s: %w", s.ID, err)
		}
		switch {
		case res.ExitCode == 0:
			rep.VolumesRemoved = append(rep.VolumesRemoved, v)
		case isNoSuchVolume(res.Stderr, res.Stdout):
			rep.VolumesAbsent = append(rep.VolumesAbsent, v)
		default:
			return rep, fmt.Errorf("ops: wipe %s: docker volume rm %s failed (exit %d): %s",
				s.ID, v, res.ExitCode, firstNonEmptyLine(res.Stderr, res.Stdout))
		}
	}

	// Bring the service back when the descriptor knows how, mirroring
	// ClearService, whose contract is likewise "stop, delete this service's
	// own data, run it again" rather than leaving the operator with a hole.
	if s.Create != nil {
		if err := s.Create(ctx, e); err != nil {
			return rep, fmt.Errorf("ops: wipe %s: recreate container %s: %w", s.ID, s.ContainerName, err)
		}
		rep.Recreated = true
	}

	if !rep.ContainerRemoved && len(rep.VolumesRemoved) == 0 && !rep.Recreated {
		return rep, nil
	}
	return rep, cascadeRestart(ctx, e, s, &rep)
}

// cascadeRestart restarts every service in front of s. See WipeService for
// why this is mandatory rather than advisory.
//
// Only RUNNING fronts are restarted. A stopped one holds no in-memory
// high-water mark, so there is nothing stale to clear, and starting it would
// override an operator's deliberate stop. A front that does not exist at all
// is skipped for the same reason.
//
// Every failure is collected rather than returned on the first one: the
// wipe has already happened, so the useful outcome is "these fronts are now
// serving a head that no longer exists" — a complete list, not the first
// name alphabetically.
func cascadeRestart(ctx context.Context, e executor.Executor, s DockerService, rep *WipeReport) error {
	var failures []error
	for _, front := range frontsOf(s) {
		st, err := ServiceStatus(ctx, e, front)
		if err != nil {
			failures = append(failures, fmt.Errorf("%s: %w", front.ID, err))
			continue
		}
		if st.State != StateRunning {
			rep.CascadeSkipped = append(rep.CascadeSkipped, front.ID)
			continue
		}
		if _, err := ContainerAction(ctx, e, front, "restart"); err != nil {
			failures = append(failures, fmt.Errorf("%s: %w", front.ID, err))
			continue
		}
		rep.Cascaded = append(rep.Cascaded, front.ID)
	}
	if len(failures) > 0 {
		return fmt.Errorf("ops: wipe %s: the wipe succeeded but a service in front of it could not be restarted, "+
			"so it is now serving a STALE head — a head the freshly wiped chain does not have, which it will not "+
			"correct on its own until the new chain outgrows the old one. Restart it by hand: %w",
			s.ID, errors.Join(failures...))
	}
	return nil
}

// frontsOf flattens s.FrontedBy breadth-first (nearest front first), deduped
// by container name and excluding s itself, so a declared cycle terminates
// instead of restarting forever.
func frontsOf(s DockerService) []DockerService {
	seen := map[string]bool{s.ContainerName: true}
	var out []DockerService
	queue := append([]DockerService(nil), s.FrontedBy...)
	for len(queue) > 0 {
		front := queue[0]
		queue = queue[1:]
		if seen[front.ContainerName] {
			continue
		}
		seen[front.ContainerName] = true
		out = append(out, front)
		queue = append(queue, front.FrontedBy...)
	}
	return out
}
