package ops

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/valve-tech/valve-node-app/internal/catalog"
	"github.com/valve-tech/valve-node-app/internal/executor"
)

// ---- ERPCRunArgs (pure) ----

func TestERPCRunArgs_Defaults(t *testing.T) {
	got := ERPCRunArgs(ERPCRunSpec{HostConfigPath: "/var/lib/valve-node-app/369/erpc.yaml"})

	want := []string{
		"run", "-d",
		"--name", "valve-node-app-erpc",
		"--restart", "unless-stopped",
		"-p", "127.0.0.1:4000:4000",
		"-v", "/var/lib/valve-node-app/369/erpc.yaml:/erpc.yaml:ro",
		"ghcr.io/erpc/erpc:0.1.1",
		"--config", "/erpc.yaml",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ERPCRunArgs mismatch\n got: %#v\nwant: %#v", got, want)
	}
}

func TestERPCRunArgs_NoAddHostByDefault(t *testing.T) {
	got := strings.Join(ERPCRunArgs(ERPCRunSpec{HostConfigPath: "/tmp/erpc.yaml"}), " ")
	if strings.Contains(got, "--add-host") {
		t.Fatalf("want no --add-host unless requested, got %q", got)
	}
}

func TestERPCRunArgs_AddHostGateway(t *testing.T) {
	got := ERPCRunArgs(ERPCRunSpec{HostConfigPath: "/tmp/erpc.yaml", AddHostGateway: true})

	// Must appear before the image ref, i.e. still in docker's own flag
	// section — anything after the image is passed to the container.
	idxAddHost, idxImage := -1, -1
	for i, a := range got {
		switch a {
		case "--add-host":
			idxAddHost = i
		case DefaultERPCImage:
			idxImage = i
		}
	}
	if idxAddHost < 0 {
		t.Fatalf("want --add-host in %#v", got)
	}
	if idxAddHost > idxImage {
		t.Fatalf("--add-host (%d) must precede the image ref (%d): %#v", idxAddHost, idxImage, got)
	}
	if got[idxAddHost+1] != "host.docker.internal:host-gateway" {
		t.Fatalf("want host-gateway mapping, got %q", got[idxAddHost+1])
	}
}

func TestERPCRunArgs_BindAddressVariants(t *testing.T) {
	cases := []struct {
		name string
		bind string
		port int
		want string
	}{
		{"empty defaults to loopback", "", 0, "127.0.0.1:4000:4000"},
		{"explicit loopback", "127.0.0.1", 0, "127.0.0.1:4000:4000"},
		{"all interfaces", "0.0.0.0", 0, "0.0.0.0:4000:4000"},
		{"tailscale address", "100.101.102.103", 0, "100.101.102.103:4000:4000"},
		{"lan address custom port", "192.168.1.50", 8080, "192.168.1.50:8080:4000"},
		{"privileged host port", "127.0.0.1", 80, "127.0.0.1:80:4000"},
		{"ipv6 loopback gets bracketed", "::1", 0, "[::1]:4000:4000"},
		{"ipv6 all interfaces gets bracketed", "::", 4100, "[::]:4100:4000"},
		{"already-bracketed ipv6 not double bracketed", "[fd00::1]", 4000, "[fd00::1]:4000:4000"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			args := ERPCRunArgs(ERPCRunSpec{BindAddr: tc.bind, HostPort: tc.port, HostConfigPath: "/tmp/erpc.yaml"})
			got := valueAfter(t, args, "-p")
			if got != tc.want {
				t.Fatalf("publish spec: got %q, want %q", got, tc.want)
			}
		})
	}
}

// The container-side port is fixed regardless of the host port the operator
// picked — the container has its own port namespace, so there is nothing
// there to collide with.
func TestERPCRunArgs_ContainerPortAlwaysFixed(t *testing.T) {
	args := ERPCRunArgs(ERPCRunSpec{HostPort: 9999, HostConfigPath: "/tmp/erpc.yaml"})
	if got := valueAfter(t, args, "-p"); !strings.HasSuffix(got, ":4000") {
		t.Fatalf("want container port 4000, got %q", got)
	}
}

func TestERPCRunArgs_CustomImageAndName(t *testing.T) {
	args := ERPCRunArgs(ERPCRunSpec{
		Image:          "ghcr.io/erpc/erpc:0.2.0",
		ContainerName:  "my-gateway",
		HostConfigPath: "/tmp/erpc.yaml",
	})
	if got := valueAfter(t, args, "--name"); got != "my-gateway" {
		t.Fatalf("--name: got %q", got)
	}
	if !containsArg(args, "ghcr.io/erpc/erpc:0.2.0") {
		t.Fatalf("want custom image ref in %#v", args)
	}
	if containsArg(args, DefaultERPCImage) {
		t.Fatalf("default image must not appear when one was given: %#v", args)
	}
}

func TestERPCRunArgs_MountIsReadOnly(t *testing.T) {
	args := ERPCRunArgs(ERPCRunSpec{HostConfigPath: "/etc/valve/erpc.yaml"})
	got := valueAfter(t, args, "-v")
	if got != "/etc/valve/erpc.yaml:/erpc.yaml:ro" {
		t.Fatalf("mount: got %q", got)
	}
}

// Paths with spaces/quotes must survive rendering INTACT — the pure
// renderer produces argv elements, so it must not pre-quote anything;
// quoting is DockerRun's job at the sh -c boundary.
func TestERPCRunArgs_PathWithSpacesIsNotPreQuoted(t *testing.T) {
	const p = `/Users/Some One/Library/Application Support/valve/erpc.yaml`
	args := ERPCRunArgs(ERPCRunSpec{HostConfigPath: p})
	got := valueAfter(t, args, "-v")
	if got != p+":/erpc.yaml:ro" {
		t.Fatalf("mount arg: got %q, want %q", got, p+":/erpc.yaml:ro")
	}
	if strings.Contains(got, `'`) || strings.Contains(got, `\`) {
		t.Fatalf("pure renderer must not shell-quote: %q", got)
	}
}

func TestERPCRunArgs_QuotingHappensAtTheExecutorBoundary(t *testing.T) {
	e := newFakeExecutor()
	args := ERPCRunArgs(ERPCRunSpec{HostConfigPath: `/data/it's here/erpc.yaml`})
	if _, err := DockerRun(context.Background(), e, args...); err != nil {
		t.Fatalf("DockerRun: %v", err)
	}
	cmd := e.callLog()[0]
	if !strings.HasPrefix(cmd, "docker 'run' '-d'") {
		t.Fatalf("want every arg single-quoted, got %q", cmd)
	}
	// The embedded apostrophe must be escaped, not terminating the quote.
	if !strings.Contains(cmd, `'\''`) {
		t.Fatalf("want escaped apostrophe in %q", cmd)
	}
}

// ---- ERPCRunSpecFor / ERPCContainerWire ----

func TestERPCRunSpecFor_UsesWireConfigAccessors(t *testing.T) {
	w := testWire()
	w.ERPCPort = 4100
	w.ERPCBindAddr = "100.64.0.7"

	spec := ERPCRunSpecFor(w, true)
	if spec.BindAddr != "100.64.0.7" || spec.HostPort != 4100 {
		t.Fatalf("spec bind/port: %+v", spec)
	}
	if spec.HostConfigPath != w.ERPCConfigPath() {
		t.Fatalf("config path: got %q, want %q", spec.HostConfigPath, w.ERPCConfigPath())
	}
	if !spec.AddHostGateway {
		t.Fatal("want AddHostGateway propagated")
	}
}

func TestERPCRunSpecFor_ZeroValuesResolveToDefaults(t *testing.T) {
	spec := ERPCRunSpecFor(testWire(), false)
	if spec.BindAddr != "127.0.0.1" || spec.HostPort != 4000 {
		t.Fatalf("want catalog defaults, got %+v", spec)
	}
}

func TestERPCContainerWire_ListensWideInsideContainer(t *testing.T) {
	w := testWire()
	w.ERPCBindAddr = "127.0.0.1"
	w.ERPCPort = 4100

	got := ERPCContainerWire(w, DockerHostAlias)
	if got.ERPCBind() != "0.0.0.0" {
		t.Fatalf("in-container bind: got %q, want 0.0.0.0 (a loopback listener is unreachable through -p)", got.ERPCBind())
	}
	if got.ERPCHTTP() != 4000 {
		t.Fatalf("in-container port: got %d, want 4000", got.ERPCHTTP())
	}
}

func TestERPCContainerWire_RewritesLoopbackUpstreamToHostAlias(t *testing.T) {
	for _, addr := range []string{"", "127.0.0.1", "::1", "localhost"} {
		w := testWire()
		w.RPCBindAddr = addr

		got := ERPCContainerWire(w, DockerHostAlias)
		if got.RPCBind() != DockerHostAlias {
			t.Fatalf("RPCBindAddr %q: got %q, want %q", addr, got.RPCBind(), DockerHostAlias)
		}
	}
}

func TestERPCContainerWire_LeavesRoutableUpstreamAlone(t *testing.T) {
	w := testWire()
	w.RPCBindAddr = "100.64.0.7" // e.g. a Tailscale IP — reachable from the container as-is

	if got := ERPCContainerWire(w, DockerHostAlias); got.RPCBind() != "100.64.0.7" {
		t.Fatalf("want routable address untouched, got %q", got.RPCBind())
	}
}

func TestERPCContainerWire_EmptyAliasDisablesUpstreamRewrite(t *testing.T) {
	w := testWire()
	if got := ERPCContainerWire(w, ""); got.RPCBind() != "127.0.0.1" {
		t.Fatalf("want no rewrite with empty alias, got %q", got.RPCBind())
	}
}

func TestERPCContainerWire_DoesNotMutateCaller(t *testing.T) {
	w := testWire()
	w.ERPCBindAddr = "127.0.0.1"
	_ = ERPCContainerWire(w, DockerHostAlias)

	if w.ERPCBindAddr != "127.0.0.1" || w.RPCBindAddr != "" {
		t.Fatalf("caller's WireConfig was mutated: %+v", w)
	}
}

// The in-container config must actually render with the rewritten values —
// this is the join between the docker spec and catalog's renderer.
func TestERPCContainerWire_RendersUsableContainerConfig(t *testing.T) {
	yaml, err := catalog.RenderERPCConfig(ERPCContainerWire(testWire(), DockerHostAlias))
	if err != nil {
		t.Fatalf("RenderERPCConfig: %v", err)
	}
	if !strings.Contains(yaml, `httpHostV4: "0.0.0.0"`) {
		t.Fatalf("want a wide in-container listener:\n%s", yaml)
	}
	if !strings.Contains(yaml, "httpPortV4: 4000") {
		t.Fatalf("want in-container port 4000:\n%s", yaml)
	}
	if !strings.Contains(yaml, "http://host.docker.internal:8545") {
		t.Fatalf("want the local-node upstream reachable from inside the container:\n%s", yaml)
	}
}

// ---- ProbeDocker ----

func TestProbeDocker_AbsentReturnsTypedError(t *testing.T) {
	e := newFakeExecutor().script("command -v docker", executor.Result{ExitCode: 1})

	info, err := ProbeDocker(context.Background(), e)
	if err == nil {
		t.Fatal("want an error when docker is absent")
	}
	if !errors.Is(err, ErrDockerAbsent) {
		t.Fatalf("want errors.Is(err, ErrDockerAbsent), got %v", err)
	}
	var absent *DockerAbsentError
	if !errors.As(err, &absent) {
		t.Fatalf("want errors.As(*DockerAbsentError), got %T", err)
	}
	if absent.Hint == "" {
		t.Fatal("want an install hint on the typed error")
	}
	if info.Present {
		t.Fatal("want Present=false")
	}
}

// A wrapped absence must still be recognizable — this is what lets a
// caller several layers up branch on "offer the install prompt".
func TestProbeDocker_AbsentSurvivesWrapping(t *testing.T) {
	e := newFakeExecutor().script("command -v docker", executor.Result{ExitCode: 127})
	_, err := ProbeDocker(context.Background(), e)
	if wrapped := fmt.Errorf("gateway: %w", err); !errors.Is(wrapped, ErrDockerAbsent) {
		t.Fatalf("want ErrDockerAbsent through a %%w wrap, got %v", wrapped)
	}
}

func TestProbeDocker_TransportErrorIsNotAbsence(t *testing.T) {
	e := newFakeExecutor().errOn("command -v docker", errors.New("ssh: connection lost"))

	_, err := ProbeDocker(context.Background(), e)
	if err == nil {
		t.Fatal("want an error")
	}
	if errors.Is(err, ErrDockerAbsent) {
		t.Fatal("a transport failure must NOT read as docker being absent")
	}
	if !strings.Contains(err.Error(), "ssh: connection lost") {
		t.Fatalf("want the transport error wrapped in, got %v", err)
	}
}

func TestProbeDocker_DaemonDownIsAReadingNotAnError(t *testing.T) {
	e := newFakeExecutor().
		script("command -v docker", executor.Result{Stdout: "/usr/local/bin/docker\n"}).
		script("docker --version", executor.Result{Stdout: "Docker version 27.3.1, build ce12230\n"}).
		script("docker info --format", executor.Result{
			ExitCode: 1,
			Stderr:   "Cannot connect to the Docker daemon at unix:///var/run/docker.sock. Is the docker daemon running?\n",
		})

	info, err := ProbeDocker(context.Background(), e)
	if err != nil {
		t.Fatalf("a down daemon is a reading, not an error: %v", err)
	}
	if !info.Present {
		t.Fatal("want Present=true")
	}
	if info.DaemonReachable {
		t.Fatal("want DaemonReachable=false")
	}
	if !strings.Contains(info.DaemonError, "Cannot connect to the Docker daemon") {
		t.Fatalf("want the engine's own words in DaemonError, got %q", info.DaemonError)
	}
}

func TestProbeDocker_LinuxEngine(t *testing.T) {
	e := newFakeExecutor().
		script("command -v docker", executor.Result{Stdout: "/usr/bin/docker\n"}).
		script("docker --version", executor.Result{Stdout: "Docker version 27.3.1, build ce12230\n"}).
		script("docker info --format", executor.Result{Stdout: "27.3.1|linux|x86_64|node-01|Ubuntu 24.04.1 LTS\n"})

	info, err := ProbeDocker(context.Background(), e)
	if err != nil {
		t.Fatalf("ProbeDocker: %v", err)
	}
	if !info.Present || !info.DaemonReachable {
		t.Fatalf("want present+reachable, got %+v", info)
	}
	if info.ServerVersion != "27.3.1" || info.OSType != "linux" || info.Architecture != "x86_64" {
		t.Fatalf("engine fields: %+v", info)
	}
	if info.HostName != "node-01" || info.OperatingSystem != "Ubuntu 24.04.1 LTS" {
		t.Fatalf("host fields: %+v", info)
	}
	if info.Flavor != FlavorDockerEngine {
		t.Fatalf("flavor: got %q, want %q", info.Flavor, FlavorDockerEngine)
	}
	if info.VMBacked() {
		t.Fatal("a plain Linux engine is not VM-backed")
	}
	if info.WindowsContainers() {
		t.Fatal("want Linux containers")
	}
}

// A macOS host reporting OSType "linux" is the whole point: the raw host
// uname internal/setup preflights on would reject this box, but the engine
// behind it runs Linux containers perfectly well.
func TestProbeDocker_DockerDesktopOnMacReportsLinuxEngine(t *testing.T) {
	e := newFakeExecutor().
		script("command -v docker", executor.Result{Stdout: "/usr/local/bin/docker\n"}).
		script("docker --version", executor.Result{Stdout: "Docker version 27.4.0, build bde2b89\n"}).
		script("docker info --format", executor.Result{Stdout: "27.4.0|linux|aarch64|docker-desktop|Docker Desktop\n"})

	info, err := ProbeDocker(context.Background(), e)
	if err != nil {
		t.Fatalf("ProbeDocker: %v", err)
	}
	if info.OSType != "linux" {
		t.Fatalf("want a linux engine on a darwin host, got %q", info.OSType)
	}
	if info.Flavor != FlavorDockerDesktop {
		t.Fatalf("flavor: got %q", info.Flavor)
	}
	if !info.VMBacked() {
		t.Fatal("Docker Desktop is VM-backed")
	}
}

func TestProbeDocker_WindowsContainerMode(t *testing.T) {
	e := newFakeExecutor().
		script("command -v docker", executor.Result{Stdout: "/c/Program Files/Docker/Docker/resources/bin/docker\n"}).
		script("docker --version", executor.Result{Stdout: "Docker version 27.4.0, build bde2b89\n"}).
		script("docker info --format", executor.Result{Stdout: "27.4.0|windows|x86_64|DESKTOP-ABC|Microsoft Windows Server 2022\n"})

	info, err := ProbeDocker(context.Background(), e)
	if err != nil {
		t.Fatalf("ProbeDocker: %v", err)
	}
	if !info.WindowsContainers() {
		t.Fatalf("want Windows-container mode detected, got OSType %q", info.OSType)
	}
}

func TestProbeDocker_PartialInfoOutputDoesNotBreakTheReading(t *testing.T) {
	e := newFakeExecutor().
		script("command -v docker", executor.Result{Stdout: "/usr/bin/docker\n"}).
		script("docker --version", executor.Result{Stdout: "Docker version 20.10.7\n"}).
		script("docker info --format", executor.Result{Stdout: "20.10.7|linux\n"})

	info, err := ProbeDocker(context.Background(), e)
	if err != nil {
		t.Fatalf("ProbeDocker: %v", err)
	}
	if info.ServerVersion != "20.10.7" || info.OSType != "linux" {
		t.Fatalf("want the fields that were present, got %+v", info)
	}
	if info.Architecture != "" || info.HostName != "" || info.OperatingSystem != "" {
		t.Fatalf("want missing fields empty, got %+v", info)
	}
}

func TestProbeDocker_RunsOnlyReadOnlyCommands(t *testing.T) {
	e := newFakeExecutor().
		script("command -v docker", executor.Result{Stdout: "/usr/bin/docker\n"}).
		script("docker info --format", executor.Result{Stdout: "27.3.1|linux|x86_64|n|Ubuntu\n"})

	if _, err := ProbeDocker(context.Background(), e); err != nil {
		t.Fatalf("ProbeDocker: %v", err)
	}
	for _, cmd := range e.callLog() {
		for _, mutating := range []string{"docker run", "docker rm", "docker stop", "docker pull", "docker start"} {
			if strings.Contains(cmd, mutating) {
				t.Fatalf("probe must never mutate: %q", cmd)
			}
		}
	}
}

// ---- detectDockerFlavor (pure) ----

func TestDetectDockerFlavor(t *testing.T) {
	cases := []struct {
		name                 string
		banner, host, osName string
		want                 string
	}{
		{"linux engine", "Docker version 27.3.1, build ce12230", "node-01", "Ubuntu 24.04.1 LTS", FlavorDockerEngine},
		{"docker desktop", "Docker version 27.4.0", "docker-desktop", "Docker Desktop", FlavorDockerDesktop},
		{"colima", "Docker version 27.3.1", "colima", "Alpine Linux v3.20", FlavorColima},
		{"orbstack", "Docker version 27.3.1", "orbstack", "OrbStack", FlavorOrbStack},
		{"rancher desktop", "Docker version 26.1.0", "lima-rancher-desktop", "Alpine Linux v3.19", FlavorRancherDesktop},
		{"podman via banner", "podman version 5.4.0", "myhost", "Fedora Linux 41", FlavorPodman},
		{"podman via engine name", "Docker version 5.4.0", "podman-machine-default", "Fedora CoreOS", FlavorPodman},
		{"nothing known", "", "", "", FlavorUnknown},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := detectDockerFlavor(tc.banner, tc.host, tc.osName); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

// ---- lifecycle helpers ----

func TestContainerExists_AnchorsTheNameFilter(t *testing.T) {
	e := newFakeExecutor().script("docker 'ps'", executor.Result{Stdout: "valve-node-app-erpc\n"})

	ok, err := ContainerExists(context.Background(), e, ERPCContainerName)
	if err != nil {
		t.Fatalf("ContainerExists: %v", err)
	}
	if !ok {
		t.Fatal("want exists=true")
	}
	cmd := e.callLog()[0]
	if !strings.Contains(cmd, `'name=^valve-node-app-erpc$'`) {
		t.Fatalf("want an anchored name filter (docker's filter is a substring regex), got %q", cmd)
	}
}

func TestContainerExists_SubstringMatchIsNotAMatch(t *testing.T) {
	// Defence in depth: even if the engine returned a near-miss name, the
	// exact-line comparison must reject it.
	e := newFakeExecutor().script("docker 'ps'", executor.Result{Stdout: "valve-node-app-erpc-old\n"})

	ok, err := ContainerExists(context.Background(), e, ERPCContainerName)
	if err != nil {
		t.Fatalf("ContainerExists: %v", err)
	}
	if ok {
		t.Fatal("want exists=false for a near-miss name")
	}
}

func TestContainerExists_EmptyOutputMeansNo(t *testing.T) {
	e := newFakeExecutor().script("docker 'ps'", executor.Result{Stdout: "\n"})

	ok, err := ContainerExists(context.Background(), e, ERPCContainerName)
	if err != nil {
		t.Fatalf("ContainerExists: %v", err)
	}
	if ok {
		t.Fatal("want exists=false")
	}
}

func TestContainerRunning_True(t *testing.T) {
	e := newFakeExecutor().script("docker 'inspect'", executor.Result{Stdout: "true\n"})

	running, err := ContainerRunning(context.Background(), e, ERPCContainerName)
	if err != nil {
		t.Fatalf("ContainerRunning: %v", err)
	}
	if !running {
		t.Fatal("want running=true")
	}
}

func TestContainerRunning_MissingContainerIsNotAnError(t *testing.T) {
	e := newFakeExecutor().script("docker 'inspect'", executor.Result{
		ExitCode: 1,
		Stderr:   "Error: No such object: valve-node-app-erpc\n",
	})

	running, err := ContainerRunning(context.Background(), e, ERPCContainerName)
	if err != nil {
		t.Fatalf("a missing container is a reading, not an error: %v", err)
	}
	if running {
		t.Fatal("want running=false")
	}
}

func TestStopContainer_AbsentContainerIsSuccess(t *testing.T) {
	e := newFakeExecutor().script("docker 'stop'", executor.Result{
		ExitCode: 1,
		Stderr:   "Error response from daemon: No such container: valve-node-app-erpc\n",
	})

	if err := StopContainer(context.Background(), e, ERPCContainerName); err != nil {
		t.Fatalf("stopping an absent container is already the goal state: %v", err)
	}
}

func TestStopContainer_RealFailureIsAnError(t *testing.T) {
	e := newFakeExecutor().script("docker 'stop'", executor.Result{
		ExitCode: 1,
		Stderr:   "permission denied while trying to connect to the Docker daemon socket\n",
	})

	err := StopContainer(context.Background(), e, ERPCContainerName)
	if err == nil {
		t.Fatal("want an error for a genuine stop failure")
	}
	if !strings.Contains(err.Error(), "permission denied") {
		t.Fatalf("want the engine's message surfaced, got %v", err)
	}
}

func TestRemoveContainer_ForcesAndTolerAtesAbsence(t *testing.T) {
	e := newFakeExecutor().script("docker 'rm'", executor.Result{
		ExitCode: 1,
		Stderr:   "Error: No such container: valve-node-app-erpc\n",
	})

	if err := RemoveContainer(context.Background(), e, ERPCContainerName); err != nil {
		t.Fatalf("RemoveContainer: %v", err)
	}
	if cmd := e.callLog()[0]; !strings.Contains(cmd, "'rm' '-f'") {
		t.Fatalf("want a forced remove, got %q", cmd)
	}
}

func TestDockerRun_TransportErrorIsWrapped(t *testing.T) {
	e := newFakeExecutor().errOn("docker 'ps'", errors.New("ssh: session closed"))

	_, err := ContainerExists(context.Background(), e, ERPCContainerName)
	if err == nil {
		t.Fatal("want an error")
	}
	if !strings.Contains(err.Error(), "ssh: session closed") {
		t.Fatalf("want the transport error wrapped, got %v", err)
	}
}

// ---- helpers ----

// valueAfter returns the argument immediately following flag.
func valueAfter(t *testing.T, args []string, flag string) string {
	t.Helper()
	for i, a := range args {
		if a == flag && i+1 < len(args) {
			return args[i+1]
		}
	}
	t.Fatalf("flag %q not found (or has no value) in %#v", flag, args)
	return ""
}

func containsArg(args []string, want string) bool {
	for _, a := range args {
		if a == want {
			return true
		}
	}
	return false
}
