package ops

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/valve-tech/valve-node-app/internal/executor"
)

// ---------------------------------------------------------------------
// test doubles / helpers
// ---------------------------------------------------------------------

// dockerCmd renders the exact command string DockerRun sends to the
// Executor for these argv elements. Every script key and every command
// assertion below is built with it, so a change to the quoting rule or to a
// rendered argv shows up as a real failure rather than a silently
// never-matching substring.
func dockerCmd(args ...string) string {
	parts := make([]string, 0, len(args)+1)
	parts = append(parts, "docker")
	for _, a := range args {
		parts = append(parts, shQuote(a))
	}
	return strings.Join(parts, " ")
}

// seqExecutor answers a given command with a QUEUE of results, falling back
// to the package's shared fakeExecutor for everything else.
//
// It exists because the fake's scripts are static, and the lifecycle calls
// `docker inspect` twice around every action — once to decide what to do and
// once to read the result back (the container-world equivalent of
// ServiceAction's `systemctl is-active` read-back). Asserting that the
// read-back is what gets returned needs the second inspect to answer
// differently from the first.
type seqExecutor struct {
	*fakeExecutor
	mu  sync.Mutex
	seq []seqScript
}

type seqScript struct {
	match   string
	results []executor.Result
}

func newSeqExecutor(f *fakeExecutor) *seqExecutor { return &seqExecutor{fakeExecutor: f} }

func (s *seqExecutor) queue(match string, results ...executor.Result) *seqExecutor {
	s.seq = append(s.seq, seqScript{match: match, results: results})
	return s
}

func (s *seqExecutor) Run(ctx context.Context, cmd string, opts *executor.RunOpts) (executor.Result, error) {
	s.mu.Lock()
	for i := range s.seq {
		if !strings.Contains(cmd, s.seq[i].match) || len(s.seq[i].results) == 0 {
			continue
		}
		res := s.seq[i].results[0]
		// The last queued result sticks, so a test only has to script the
		// transitions it cares about.
		if len(s.seq[i].results) > 1 {
			s.seq[i].results = s.seq[i].results[1:]
		}
		s.mu.Unlock()

		s.fakeExecutor.mu.Lock()
		s.fakeExecutor.calls = append(s.fakeExecutor.calls, cmd)
		s.fakeExecutor.mu.Unlock()
		return res, nil
	}
	s.mu.Unlock()
	return s.fakeExecutor.Run(ctx, cmd, opts)
}

// Canned `docker inspect` readings.
func inspectRunning(image string) executor.Result {
	return executor.Result{ExitCode: 0, Stdout: "true|0|" + image + "|sha256:deadbeef\n"}
}

func inspectStopped(image string, code int) executor.Result {
	return executor.Result{ExitCode: 0, Stdout: "false|" + itoa(code) + "|" + image + "|sha256:deadbeef\n"}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	if neg {
		return "-" + string(b)
	}
	return string(b)
}

// The engine's real words for the three failure modes that must stay
// distinguishable, copied verbatim from docker/podman.
var (
	inspectAbsent = executor.Result{
		ExitCode: 1,
		Stderr:   "Error: No such object: valve-devnet\n",
	}
	inspectDaemonDown = executor.Result{
		ExitCode: 1,
		Stderr:   "Cannot connect to the Docker daemon at unix:///var/run/docker.sock. Is the docker daemon running?\n",
	}
	inspectCLIMissing = executor.Result{
		ExitCode: 127,
		Stderr:   "sh: 1: docker: not found\n",
	}
)

// testDevnet is the shape the local-devnet service has: one container, one
// named volume holding its chain data, fronted by the eRPC gateway.
func testDevnet() DockerService {
	return DockerService{
		ID:            "devnet",
		ContainerName: "valve-devnet",
		Volumes:       []string{"valve-devnet-data"},
		FrontedBy:     []DockerService{ERPCService()},
	}
}

func ranIndex(f *fakeExecutor, cmd string) int {
	for i, c := range f.callLog() {
		if c == cmd {
			return i
		}
	}
	return -1
}

func ran(f *fakeExecutor, cmd string) bool { return ranIndex(f, cmd) >= 0 }

func ranContaining(f *fakeExecutor, substr string) bool {
	for _, c := range f.callLog() {
		if strings.Contains(c, substr) {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------
// pure argv rendering
// ---------------------------------------------------------------------

func TestContainerActionArgs(t *testing.T) {
	for _, tc := range []struct {
		action string
		want   []string
	}{
		{"start", []string{"start", "valve-devnet"}},
		{"stop", []string{"stop", "valve-devnet"}},
		{"restart", []string{"restart", "valve-devnet"}},
	} {
		if got := ContainerActionArgs(tc.action, "valve-devnet"); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("ContainerActionArgs(%q): got %#v, want %#v", tc.action, got, tc.want)
		}
	}
}

func TestContainerInspectArgs_AsksForStateAndImage(t *testing.T) {
	args := ContainerInspectArgs("valve-devnet")
	if args[0] != "inspect" || args[len(args)-1] != "valve-devnet" {
		t.Fatalf("ContainerInspectArgs: got %#v", args)
	}
	format := valueAfter(t, args, "-f")
	for _, want := range []string{".State.Running", ".Config.Image"} {
		if !strings.Contains(format, want) {
			t.Errorf("inspect format %q must ask for %s — status is not complete without the image tag actually in use", format, want)
		}
	}
}

// -v removes the container's ANONYMOUS volumes. It is deliberately not the
// mechanism for named volumes (docker ignores those here); those are removed
// explicitly, one `docker volume rm` per declared name.
func TestContainerRemoveArgs_ForcesAndTakesAnonymousVolumes(t *testing.T) {
	want := []string{"rm", "-f", "-v", "valve-devnet"}
	if got := ContainerRemoveArgs("valve-devnet"); !reflect.DeepEqual(got, want) {
		t.Fatalf("ContainerRemoveArgs: got %#v, want %#v", got, want)
	}
}

func TestVolumeRemoveArgs(t *testing.T) {
	want := []string{"volume", "rm", "valve-devnet-data"}
	if got := VolumeRemoveArgs("valve-devnet-data"); !reflect.DeepEqual(got, want) {
		t.Fatalf("VolumeRemoveArgs: got %#v, want %#v", got, want)
	}
}

// The argv stays unquoted; quoting happens once, at the sh -c boundary, the
// same contract ERPCRunArgs/DockerRun already have.
func TestLifecycleArgs_QuotingHappensAtTheExecutorBoundary(t *testing.T) {
	args := ContainerActionArgs("restart", "valve devnet")
	if args[1] != "valve devnet" {
		t.Fatalf("argv must carry the raw name: %#v", args)
	}

	f := newFakeExecutor()
	svc := DockerService{ID: "x", ContainerName: "valve devnet"}
	s := newSeqExecutor(f).queue("'inspect'", inspectRunning("img:1"))
	if _, err := ContainerAction(context.Background(), s, svc, "restart"); err != nil {
		t.Fatalf("ContainerAction: %v", err)
	}
	if !ran(f, dockerCmd("restart", "valve devnet")) {
		t.Fatalf("want a quoted restart command, got %#v", f.callLog())
	}
}

func TestParseInspectStatus(t *testing.T) {
	running, code, image, imageID := parseInspectStatus("true|0|valve-node-app/erpc:e909aacb|sha256:abc\n")
	if !running || code != 0 || image != "valve-node-app/erpc:e909aacb" || imageID != "sha256:abc" {
		t.Fatalf("got running=%v code=%d image=%q id=%q", running, code, image, imageID)
	}

	running, code, image, _ = parseInspectStatus("false|137|ghcr.io/erpc/erpc:0.1.1|sha256:abc")
	if running || code != 137 || image != "ghcr.io/erpc/erpc:0.1.1" {
		t.Fatalf("stopped reading: running=%v code=%d image=%q", running, code, image)
	}

	// A future/older docker that can't render one of these fields must still
	// yield a usable partial reading rather than an error, exactly as
	// parseDockerInfo does.
	running, _, image, _ = parseInspectStatus("true|0")
	if !running || image != "" {
		t.Fatalf("partial reading: running=%v image=%q", running, image)
	}
}

func TestClassifyDockerFailure(t *testing.T) {
	for _, tc := range []struct {
		name     string
		exitCode int
		stderr   string
		want     string
	}{
		{"absent (inspect)", 1, "Error: No such object: valve-devnet", dockerFailureAbsentContainer},
		{"absent (daemon)", 1, "Error response from daemon: No such container: valve-devnet", dockerFailureAbsentContainer},
		{"absent (podman)", 125, "Error: no such container valve-devnet", dockerFailureAbsentContainer},
		{"daemon down", 1, "Cannot connect to the Docker daemon at unix:///var/run/docker.sock. Is the docker daemon running?", dockerFailureDaemonUnreachable},
		{"daemon down (windows)", 1, `error during connect: Get "http://...": open //./pipe/docker_engine: The system cannot find the file specified.`, dockerFailureDaemonUnreachable},
		{"podman socket down", 125, "Cannot connect to Podman socket", dockerFailureDaemonUnreachable},
		{"cli missing", 127, "sh: 1: docker: not found", dockerFailureCLIMissing},
		{"cli missing (bash)", 127, "bash: docker: command not found", dockerFailureCLIMissing},
		{"anything else", 1, "unexpected weirdness", dockerFailureOther},
	} {
		if got := classifyDockerFailure(tc.exitCode, tc.stderr, ""); got != tc.want {
			t.Errorf("%s: got %q, want %q", tc.name, got, tc.want)
		}
	}
}

// ---------------------------------------------------------------------
// service descriptors / validation
// ---------------------------------------------------------------------

// The gateway is stateless: its only persistent thing is erpc.yaml, a
// read-only BIND MOUNT owned by setup. If someone ever lists it as a volume
// here, wiping the gateway would delete the operator's config — exactly the
// "wipe removed more than expected" bug this test exists to prevent.
func TestERPCService_OwnsNoData(t *testing.T) {
	s := ERPCService()
	if s.ContainerName != ERPCContainerName {
		t.Errorf("ContainerName: got %q, want %q", s.ContainerName, ERPCContainerName)
	}
	if len(s.Volumes) != 0 {
		t.Errorf("the gateway owns no named volumes; wiping it must delete nothing but the container, got %#v", s.Volumes)
	}
	if err := s.Validate(); err != nil {
		t.Errorf("Validate: %v", err)
	}
}

func TestValidate_RejectsAPathWhereAVolumeNameBelongs(t *testing.T) {
	// The whole point: `docker volume rm /var/lib/valve-node-app` is not a
	// path delete, but a descriptor written that way means the author
	// BELIEVED a wipe would delete that directory. Refuse it loudly rather
	// than silently doing something else.
	for _, bad := range []string{"/var/lib/valve-node-app", "../escape", "", "-flagish", "a name with spaces"} {
		s := DockerService{ID: "devnet", ContainerName: "valve-devnet", Volumes: []string{bad}}
		if err := s.Validate(); err == nil {
			t.Errorf("Validate(%q): want an error", bad)
		}
	}
	for _, ok := range []string{"valve-devnet-data", "valve_devnet.data", "a"} {
		s := DockerService{ID: "devnet", ContainerName: "valve-devnet", Volumes: []string{ok}}
		if err := s.Validate(); err != nil {
			t.Errorf("Validate(%q): %v", ok, err)
		}
	}
}

func TestValidate_RequiresIDAndContainerName(t *testing.T) {
	if err := (DockerService{ContainerName: "x"}).Validate(); err == nil {
		t.Error("want an error for an empty ID")
	}
	if err := (DockerService{ID: "x"}).Validate(); err == nil {
		t.Error("want an error for an empty ContainerName")
	}
}

// A wipe must not execute anything at all when the descriptor is refused —
// the refusal has to come before the first destructive command, not midway.
func TestWipeService_InvalidDescriptorRunsNothing(t *testing.T) {
	f := newFakeExecutor()
	s := DockerService{ID: "devnet", ContainerName: "valve-devnet", Volumes: []string{"/var/lib/devnet"}}
	if _, err := WipeService(context.Background(), f, s); err == nil {
		t.Fatal("want an error")
	}
	if len(f.callLog()) != 0 {
		t.Fatalf("nothing may be executed for a refused descriptor, ran: %#v", f.callLog())
	}
}

// ---------------------------------------------------------------------
// status: not-created / stopped / running, and absent vs unreachable
// ---------------------------------------------------------------------

func TestServiceStatus_Running(t *testing.T) {
	f := newFakeExecutor()
	f.script(dockerCmd(ContainerInspectArgs(ERPCContainerName)...), inspectRunning("valve-node-app/erpc:e909aacb"))

	st, err := ServiceStatus(context.Background(), f, ERPCService())
	if err != nil {
		t.Fatalf("ServiceStatus: %v", err)
	}
	if st.State != StateRunning || !st.Running() || !st.Exists() {
		t.Fatalf("state: got %q", st.State)
	}
	if st.Image != "valve-node-app/erpc:e909aacb" {
		t.Fatalf("image: got %q — status must report the tag actually in use, not the tag we would build", st.Image)
	}
}

func TestServiceStatus_CreatedButStopped(t *testing.T) {
	f := newFakeExecutor()
	f.script(dockerCmd(ContainerInspectArgs("valve-devnet")...), inspectStopped("valve/devnet:1", 137))

	st, err := ServiceStatus(context.Background(), f, testDevnet())
	if err != nil {
		t.Fatalf("ServiceStatus: %v", err)
	}
	if st.State != StateStopped {
		t.Fatalf("state: got %q, want %q", st.State, StateStopped)
	}
	if st.Exists() != true || st.Running() {
		t.Fatalf("a stopped container exists but is not running: %+v", st)
	}
	if st.ExitCode != 137 {
		t.Fatalf("exit code: got %d, want 137 (the operator's clue for why it stopped)", st.ExitCode)
	}
}

// Absence is a READING, not a failure — the same call that reports it is the
// one an operator uses to decide whether to provision.
func TestServiceStatus_NotCreatedIsAReadingNotAnError(t *testing.T) {
	f := newFakeExecutor()
	f.script(dockerCmd(ContainerInspectArgs("valve-devnet")...), inspectAbsent)

	st, err := ServiceStatus(context.Background(), f, testDevnet())
	if err != nil {
		t.Fatalf("an absent container is not an error: %v", err)
	}
	if st.State != StateNotCreated {
		t.Fatalf("state: got %q, want %q", st.State, StateNotCreated)
	}
	if st.Exists() {
		t.Fatal("a not-created service must not read as existing")
	}
}

// "no container" and "no engine" need different operator prompts (install/
// provision vs start the engine), so they must be distinguishable without
// string-matching — the precedent DockerAbsentError sets.
func TestServiceStatus_DaemonUnreachableIsNotAbsence(t *testing.T) {
	f := newFakeExecutor()
	f.script(dockerCmd(ContainerInspectArgs("valve-devnet")...), inspectDaemonDown)

	st, err := ServiceStatus(context.Background(), f, testDevnet())
	if err == nil {
		t.Fatal("want an error: the state of the service is genuinely unknown")
	}
	if !errors.Is(err, ErrDockerUnreachable) {
		t.Fatalf("want ErrDockerUnreachable, got %v", err)
	}
	if errors.Is(err, ErrDockerAbsent) {
		t.Fatal("an unreachable daemon must not read as a missing docker CLI — the fix is different")
	}
	if st.State != StateUnknown {
		t.Fatalf("state: got %q, want %q", st.State, StateUnknown)
	}
	var ue *DockerUnreachableError
	if !errors.As(err, &ue) {
		t.Fatal("want a typed *DockerUnreachableError carrying the engine's own words")
	}
	if !strings.Contains(ue.Detail, "Cannot connect to the Docker daemon") {
		t.Fatalf("detail must carry the engine's complaint verbatim, got %q", ue.Detail)
	}
	if ue.Hint == "" {
		t.Fatal("want an operator-facing hint")
	}
}

func TestServiceStatus_MissingCLIUnwrapsToErrDockerAbsent(t *testing.T) {
	f := newFakeExecutor()
	f.script(dockerCmd(ContainerInspectArgs("valve-devnet")...), inspectCLIMissing)

	_, err := ServiceStatus(context.Background(), f, testDevnet())
	if !errors.Is(err, ErrDockerAbsent) {
		t.Fatalf("want ErrDockerAbsent (so the UI can offer the install hint), got %v", err)
	}
}

func TestServiceStatus_TransportErrorIsNeitherVerdict(t *testing.T) {
	boom := errors.New("ssh: connection lost")
	f := newFakeExecutor()
	f.errOn("'inspect'", boom)

	_, err := ServiceStatus(context.Background(), f, testDevnet())
	if !errors.Is(err, boom) {
		t.Fatalf("want the transport error wrapped, got %v", err)
	}
	if errors.Is(err, ErrDockerAbsent) || errors.Is(err, ErrDockerUnreachable) {
		t.Fatal("a transport failure is not a docker verdict")
	}
}

// ---------------------------------------------------------------------
// transitions
// ---------------------------------------------------------------------

func TestContainerAction_RejectsUnknownAction(t *testing.T) {
	f := newFakeExecutor()
	if _, err := ContainerAction(context.Background(), f, testDevnet(), "obliterate"); err == nil {
		t.Fatal("want an error")
	}
	if len(f.callLog()) != 0 {
		t.Fatalf("an invalid action must be rejected before anything runs, ran: %#v", f.callLog())
	}
}

func TestContainerAction_StartStoppedContainer(t *testing.T) {
	f := newFakeExecutor()
	// Stopped before the action, running after: the returned status is the
	// read-back, not the pre-read.
	s := newSeqExecutor(f).queue("'inspect'", inspectStopped("valve/devnet:1", 0), inspectRunning("valve/devnet:1"))

	st, err := ContainerAction(context.Background(), s, testDevnet(), "start")
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if !ran(f, dockerCmd("start", "valve-devnet")) {
		t.Fatalf("want a docker start, ran: %#v", f.callLog())
	}
	if st.State != StateRunning {
		t.Fatalf("returned status must be read back after the action: got %q", st.State)
	}
}

func TestContainerAction_RestartUsesRestartNotStopStart(t *testing.T) {
	f := newFakeExecutor()
	s := newSeqExecutor(f).queue("'inspect'", inspectRunning("valve/devnet:1"))

	if _, err := ContainerAction(context.Background(), s, testDevnet(), "restart"); err != nil {
		t.Fatalf("restart: %v", err)
	}
	if !ran(f, dockerCmd("restart", "valve-devnet")) {
		t.Fatalf("want a docker restart, ran: %#v", f.callLog())
	}
	if ran(f, dockerCmd("stop", "valve-devnet")) {
		t.Fatal("a restart must not be decomposed into stop+start — that widens the window the gateway sees a dead upstream")
	}
}

// Idempotency: the goal state of "stop" is "not running", and an absent
// container already satisfies it. Mirrors StopContainer's tolerance and
// ops.isActive treating a non-zero is-active as a reading.
func TestContainerAction_StopNotCreatedIsANoOp(t *testing.T) {
	f := newFakeExecutor()
	f.script(dockerCmd(ContainerInspectArgs("valve-devnet")...), inspectAbsent)

	st, err := ContainerAction(context.Background(), f, testDevnet(), "stop")
	if err != nil {
		t.Fatalf("stopping a service that does not exist is success, got %v", err)
	}
	if st.State != StateNotCreated {
		t.Fatalf("state: got %q", st.State)
	}
	if ranContaining(f, "docker stop") {
		t.Fatalf("nothing to stop — no stop command should be issued, ran: %#v", f.callLog())
	}
}

func TestContainerAction_StopAlreadyStoppedIsSuccess(t *testing.T) {
	f := newFakeExecutor()
	f.script(dockerCmd(ContainerInspectArgs("valve-devnet")...), inspectStopped("valve/devnet:1", 0))

	st, err := ContainerAction(context.Background(), f, testDevnet(), "stop")
	if err != nil {
		t.Fatalf("stopping a stopped service is success, got %v", err)
	}
	if st.State != StateStopped {
		t.Fatalf("state: got %q", st.State)
	}
}

// Without a Create hook this package cannot conjure a container: creation
// needs a config path, a platform and an image, all of which live in the
// setup plan. Say so precisely instead of running `docker start` and
// relaying "No such container".
func TestContainerAction_StartNotCreatedWithoutCreateHook(t *testing.T) {
	f := newFakeExecutor()
	f.script(dockerCmd(ContainerInspectArgs(ERPCContainerName)...), inspectAbsent)

	_, err := ContainerAction(context.Background(), f, ERPCService(), "start")
	if !errors.Is(err, ErrServiceNotCreated) {
		t.Fatalf("want ErrServiceNotCreated, got %v", err)
	}
	if ranContaining(f, "docker start") {
		t.Fatalf("must not issue a start for a container that does not exist, ran: %#v", f.callLog())
	}
}

func TestContainerAction_StartNotCreatedUsesTheCreateHook(t *testing.T) {
	f := newFakeExecutor()
	s := newSeqExecutor(f).queue("'inspect'", inspectAbsent, inspectRunning("valve/devnet:1"))

	created := 0
	svc := testDevnet()
	svc.Create = func(context.Context, executor.Executor) error { created++; return nil }

	st, err := ContainerAction(context.Background(), s, svc, "start")
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if created != 1 {
		t.Fatalf("Create called %d times, want 1", created)
	}
	if st.State != StateRunning {
		t.Fatalf("state after create: got %q", st.State)
	}
}

func TestContainerAction_EngineFailureIsReported(t *testing.T) {
	f := newFakeExecutor()
	f.script(dockerCmd(ContainerInspectArgs("valve-devnet")...), inspectRunning("valve/devnet:1"))
	f.script(dockerCmd("restart", "valve-devnet"), executor.Result{ExitCode: 1, Stderr: "Error response from daemon: cannot restart\n"})

	if _, err := ContainerAction(context.Background(), f, testDevnet(), "restart"); err == nil {
		t.Fatal("want an error")
	} else if !strings.Contains(err.Error(), "cannot restart") {
		t.Fatalf("error must carry the engine's own words, got %v", err)
	}
}

// ---------------------------------------------------------------------
// wipe
// ---------------------------------------------------------------------

func TestWipeService_RemovesTheContainerAndOnlyTheDeclaredVolumes(t *testing.T) {
	f := newFakeExecutor()
	f.script(dockerCmd(ContainerInspectArgs("valve-devnet")...), inspectRunning("valve/devnet:1"))
	f.script(dockerCmd(ContainerInspectArgs(ERPCContainerName)...), inspectAbsent)

	rep, err := WipeService(context.Background(), f, testDevnet())
	if err != nil {
		t.Fatalf("wipe: %v", err)
	}
	if !rep.ContainerRemoved {
		t.Error("want ContainerRemoved")
	}
	if !reflect.DeepEqual(rep.VolumesRemoved, []string{"valve-devnet-data"}) {
		t.Errorf("VolumesRemoved: got %#v", rep.VolumesRemoved)
	}
	if !ran(f, dockerCmd(ContainerRemoveArgs("valve-devnet")...)) {
		t.Errorf("want the container removed, ran: %#v", f.callLog())
	}
	if !ran(f, dockerCmd(VolumeRemoveArgs("valve-devnet-data")...)) {
		t.Errorf("want the declared volume removed, ran: %#v", f.callLog())
	}

	// The conservatism guarantee: a wipe touches the named container and the
	// named volumes, and nothing else. Any of these appearing would mean it
	// deleted something the operator did not ask it to.
	for _, forbidden := range []string{"volume prune", "system prune", "rmi", "image rm", "rm -rf", "docker volume rm 'erpc"} {
		if ranContaining(f, forbidden) {
			t.Errorf("wipe must never run %q, ran: %#v", forbidden, f.callLog())
		}
	}
}

// The reason this whole file exists. See WipeService's doc comment for the
// measured numbers: chain at 0x4, gateway still advertising 0x2c.
func TestWipeService_CascadesARestartToTheGatewayInFront(t *testing.T) {
	f := newFakeExecutor()
	f.script(dockerCmd(ContainerInspectArgs("valve-devnet")...), inspectRunning("valve/devnet:1"))
	f.script(dockerCmd(ContainerInspectArgs(ERPCContainerName)...), inspectRunning("valve-node-app/erpc:e909aacb"))

	rep, err := WipeService(context.Background(), f, testDevnet())
	if err != nil {
		t.Fatalf("wipe: %v", err)
	}
	if !reflect.DeepEqual(rep.Cascaded, []string{"erpc"}) {
		t.Fatalf("Cascaded: got %#v, want the gateway", rep.Cascaded)
	}

	restart := ranIndex(f, dockerCmd("restart", ERPCContainerName))
	if restart < 0 {
		t.Fatalf("the gateway in front of a wiped chain MUST be restarted — eRPC's monotonic highest-block guard makes a chain reset invisible to it, so it keeps serving a head the chain does not have. ran: %#v", f.callLog())
	}
	rm := ranIndex(f, dockerCmd(ContainerRemoveArgs("valve-devnet")...))
	if restart < rm {
		t.Fatalf("the cascade restart must come after the wipe (%d) not before (%d), or the gateway re-learns the old head", rm, restart)
	}
}

// A stopped gateway holds no in-memory high-water mark, so there is nothing
// stale to clear — starting it would change the operator's intent.
func TestWipeService_CascadeSkipsAGatewayThatIsNotRunning(t *testing.T) {
	f := newFakeExecutor()
	f.script(dockerCmd(ContainerInspectArgs("valve-devnet")...), inspectRunning("valve/devnet:1"))
	f.script(dockerCmd(ContainerInspectArgs(ERPCContainerName)...), inspectStopped("valve-node-app/erpc:e909aacb", 0))

	rep, err := WipeService(context.Background(), f, testDevnet())
	if err != nil {
		t.Fatalf("wipe: %v", err)
	}
	if len(rep.Cascaded) != 0 {
		t.Errorf("Cascaded: got %#v, want none", rep.Cascaded)
	}
	if !reflect.DeepEqual(rep.CascadeSkipped, []string{"erpc"}) {
		t.Errorf("CascadeSkipped: got %#v", rep.CascadeSkipped)
	}
	if ranContaining(f, "docker start '"+ERPCContainerName) {
		t.Error("a stopped gateway must be left stopped")
	}
}

// A cascade failure is the silent-staleness bug happening, so it has to be
// loud — and the report must still say what did happen.
func TestWipeService_CascadeFailureIsLoud(t *testing.T) {
	f := newFakeExecutor()
	f.script(dockerCmd(ContainerInspectArgs("valve-devnet")...), inspectRunning("valve/devnet:1"))
	f.script(dockerCmd(ContainerInspectArgs(ERPCContainerName)...), inspectRunning("erpc:1"))
	f.script(dockerCmd("restart", ERPCContainerName), executor.Result{ExitCode: 1, Stderr: "Error response from daemon: nope\n"})

	rep, err := WipeService(context.Background(), f, testDevnet())
	if err == nil {
		t.Fatal("want an error when the fronting gateway could not be restarted")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "stale") {
		t.Errorf("the error must explain the consequence (a stale head), got %v", err)
	}
	if !rep.ContainerRemoved || len(rep.VolumesRemoved) != 1 {
		t.Errorf("the report must still describe the wipe that did happen: %+v", rep)
	}
}

// Wiping something that is not there changes nothing, so there is no stale
// head to clear and no reason to bounce a healthy gateway.
func TestWipeService_AbsentServiceIsIdempotentAndDoesNotCascade(t *testing.T) {
	f := newFakeExecutor()
	f.script(dockerCmd(ContainerInspectArgs("valve-devnet")...), inspectAbsent)
	f.script(dockerCmd(ContainerInspectArgs(ERPCContainerName)...), inspectRunning("erpc:1"))
	f.script(dockerCmd(VolumeRemoveArgs("valve-devnet-data")...), executor.Result{
		ExitCode: 1,
		Stderr:   "Error: No such volume: valve-devnet-data\n",
	})

	rep, err := WipeService(context.Background(), f, testDevnet())
	if err != nil {
		t.Fatalf("wiping an absent service is success, got %v", err)
	}
	if rep.ContainerRemoved {
		t.Error("nothing was removed")
	}
	if !reflect.DeepEqual(rep.VolumesAbsent, []string{"valve-devnet-data"}) {
		t.Errorf("VolumesAbsent: got %#v", rep.VolumesAbsent)
	}
	if ranContaining(f, "docker rm") {
		t.Errorf("no container to remove — no rm should be issued, ran: %#v", f.callLog())
	}
	if len(rep.Cascaded) != 0 {
		t.Errorf("nothing changed, so nothing needs restarting: %#v", rep.Cascaded)
	}
}

// A wipe on a target whose engine cannot be reached must remove nothing: it
// cannot even establish what it would be removing.
func TestWipeService_UnreachableDaemonRemovesNothing(t *testing.T) {
	f := newFakeExecutor()
	f.script(dockerCmd(ContainerInspectArgs("valve-devnet")...), inspectDaemonDown)

	if _, err := WipeService(context.Background(), f, testDevnet()); !errors.Is(err, ErrDockerUnreachable) {
		t.Fatalf("want ErrDockerUnreachable, got %v", err)
	}
	if ranContaining(f, "docker rm") || ranContaining(f, "volume rm") {
		t.Fatalf("nothing may be removed when the engine is unreachable, ran: %#v", f.callLog())
	}
}

func TestWipeService_RecreatesThenCascades(t *testing.T) {
	f := newFakeExecutor()
	f.script(dockerCmd(ContainerInspectArgs("valve-devnet")...), inspectRunning("valve/devnet:1"))
	f.script(dockerCmd(ContainerInspectArgs(ERPCContainerName)...), inspectRunning("erpc:1"))

	creates := 0
	svc := testDevnet()
	svc.Create = func(context.Context, executor.Executor) error {
		creates++
		return nil
	}

	rep, err := WipeService(context.Background(), f, svc)
	if err != nil {
		t.Fatalf("wipe: %v", err)
	}
	if !rep.Recreated {
		t.Error("want Recreated")
	}
	if creates != 1 {
		t.Fatalf("Create called %d times, want 1", creates)
	}
	// The gateway must be restarted AFTER the fresh chain exists, so its
	// first read of the upstream is of the new chain.
	wiped := ranIndex(f, dockerCmd(VolumeRemoveArgs("valve-devnet-data")...))
	restart := ranIndex(f, dockerCmd("restart", ERPCContainerName))
	if restart < wiped {
		t.Fatalf("restart (%d) must follow the wipe/recreate (%d)", restart, wiped)
	}
}

// A gateway in front of a gateway is not a shape this app builds today, but
// the guard is per-process memory either way — so the cascade follows the
// declared chain rather than stopping at the first hop.
func TestWipeService_CascadeIsTransitiveAndCycleSafe(t *testing.T) {
	edge := DockerService{ID: "edge", ContainerName: "valve-edge",
		// Deliberately cyclic: edge points back at the service being wiped.
		// A cascade that did not track what it had already visited would
		// loop here forever.
		FrontedBy: []DockerService{{ID: "devnet", ContainerName: "valve-devnet"}}}
	inner := DockerService{ID: "erpc", ContainerName: ERPCContainerName, FrontedBy: []DockerService{edge}}
	devnet := DockerService{ID: "devnet", ContainerName: "valve-devnet", FrontedBy: []DockerService{inner}}

	f := newFakeExecutor()
	f.script(dockerCmd(ContainerInspectArgs("valve-devnet")...), inspectRunning("valve/devnet:1"))
	f.script(dockerCmd(ContainerInspectArgs(ERPCContainerName)...), inspectRunning("erpc:1"))
	f.script(dockerCmd(ContainerInspectArgs("valve-edge")...), inspectRunning("edge:1"))

	rep, err := WipeService(context.Background(), f, devnet)
	if err != nil {
		t.Fatalf("wipe: %v", err)
	}
	if !reflect.DeepEqual(rep.Cascaded, []string{"erpc", "edge"}) {
		t.Fatalf("Cascaded: got %#v, want both hops nearest-first", rep.Cascaded)
	}
}
