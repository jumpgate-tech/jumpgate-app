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
	}
	// The default platform is this machine's, so the expectation has to be
	// built the same way rather than hard-coded — the test must pass on an
	// arm64 laptop and an amd64 CI runner alike.
	if p := DefaultPlatform(); p != "" {
		want = append(want, "--platform", p)
	}
	want = append(want,
		"-p", "127.0.0.1:4000:4000",
		"-v", "/var/lib/valve-node-app/369/erpc.yaml:/erpc.yaml:ro",
		ERPCImageTag(),
	)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ERPCRunArgs mismatch\n got: %#v\nwant: %#v", got, want)
	}
}

// The regression this guards is a container that cannot start at all: the
// image ref must be the LAST argument, because anything after it replaces the
// image's CMD — and the gateway image declares its binary as a CMD, not an
// ENTRYPOINT. A trailing `--config /erpc.yaml` (which this function used to
// emit) made docker try to exec the flag itself:
// `exec: "--config": executable file not found in $PATH`.
func TestERPCRunArgs_ImageRefIsLast(t *testing.T) {
	for _, spec := range []ERPCRunSpec{
		{HostConfigPath: "/tmp/erpc.yaml"},
		{HostConfigPath: "/tmp/erpc.yaml", Platform: "linux/arm64", AddHostGateway: true, Image: "ghcr.io/erpc/erpc:0.1.1"},
	} {
		args := ERPCRunArgs(spec)
		image := spec.Image
		if image == "" {
			image = ERPCImageTag()
		}
		if last := args[len(args)-1]; last != image {
			t.Fatalf("ERPCRunArgs(%+v) ends with %q, want the image ref %q — arguments after the image replace the image's CMD", spec, last, image)
		}
	}
}

// The regression this guards: without --platform on RUN, a docker CLI that
// misreports its architecture (Rosetta on Apple Silicon) looks up the wrong
// variant of a present local image and falls through to a pull.
func TestERPCRunArgs_EmitsPlatformBeforeTheImage(t *testing.T) {
	args := ERPCRunArgs(ERPCRunSpec{HostConfigPath: "/tmp/erpc.yaml", Platform: "linux/arm64"})

	idxPlatform, idxImage := -1, -1
	for i, a := range args {
		switch a {
		case "--platform":
			idxPlatform = i
		case ERPCImageTag():
			idxImage = i
		}
	}
	if idxPlatform < 0 {
		t.Fatalf("want --platform in %#v", args)
	}
	if idxPlatform > idxImage {
		t.Fatalf("--platform (%d) must precede the image ref (%d) — anything after the image goes to the container: %#v", idxPlatform, idxImage, args)
	}
	if got := valueAfter(t, args, "--platform"); got != "linux/arm64" {
		t.Fatalf("--platform: got %q, want an explicit spec value to win", got)
	}
}

func TestERPCRunArgs_PlatformDefaultsToThisMachine(t *testing.T) {
	args := ERPCRunArgs(ERPCRunSpec{HostConfigPath: "/tmp/erpc.yaml"})

	want := DefaultPlatform()
	if want == "" {
		t.Skip("unrecognized GOARCH: no default platform to assert")
	}
	if got := valueAfter(t, args, "--platform"); got != want {
		t.Fatalf("--platform: got %q, want %q", got, want)
	}
}

func TestPlatformForArch(t *testing.T) {
	cases := map[string]string{
		"x86_64":  "linux/amd64", // docker info / uname -m
		"amd64":   "linux/amd64", // runtime.GOARCH
		"aarch64": "linux/arm64", // docker info / uname -m
		"arm64":   "linux/arm64", // runtime.GOARCH
		"ARM64":   "linux/arm64",
		"armv7l":  "linux/arm/v7",
		// "" means "this reading told us nothing", NOT "omit the flag" — see
		// EnginePlatform and resolveRunPlatform, which fall through to the next
		// reading rather than emitting no --platform.
		"":        "",
		"riscv64": "",
	}
	for arch, want := range cases {
		if got := PlatformForArch(arch); got != want {
			t.Fatalf("PlatformForArch(%q): got %q, want %q", arch, got, want)
		}
	}
}

// The Rosetta case, verbatim: a VM-backed desktop engine whose CLI claims
// x86_64 on a machine whose own shell says arm64. The VM runs on the host's
// silicon, so the host reading is the true one.
func TestEnginePlatform_HostWinsOnAVMBackedEngineThatDisagrees(t *testing.T) {
	e := newFakeExecutor().script("uname -m", executor.Result{Stdout: "arm64\n"})
	info := DockerInfo{Architecture: "x86_64", Flavor: FlavorDockerDesktop}

	if got := EnginePlatform(context.Background(), e, info); got != "linux/arm64" {
		t.Fatalf("got %q, want linux/arm64 (the CLI's x86_64 reading is a translation artifact)", got)
	}
}

// A plain engine can genuinely be a different architecture from the machine
// probing it (SSH, DOCKER_HOST), so there the engine's reading must stand.
func TestEnginePlatform_EngineWinsWhenNotVMBacked(t *testing.T) {
	e := newFakeExecutor().script("uname -m", executor.Result{Stdout: "arm64\n"})
	info := DockerInfo{Architecture: "x86_64", Flavor: FlavorDockerEngine}

	if got := EnginePlatform(context.Background(), e, info); got != "linux/amd64" {
		t.Fatalf("got %q, want linux/amd64", got)
	}
}

func TestEnginePlatform_AgreementNeedsNoAdjudication(t *testing.T) {
	e := newFakeExecutor().script("uname -m", executor.Result{Stdout: "x86_64\n"})
	info := DockerInfo{Architecture: "x86_64", Flavor: FlavorDockerDesktop}

	if got := EnginePlatform(context.Background(), e, info); got != "linux/amd64" {
		t.Fatalf("got %q, want linux/amd64", got)
	}
}

func TestEnginePlatform_FallsBackToEachReadingWhenTheOtherIsUnusable(t *testing.T) {
	ctx := context.Background()

	// No usable uname (missing binary / transport failure) → engine's word.
	noUname := newFakeExecutor().script("uname -m", executor.Result{ExitCode: 127})
	if got := EnginePlatform(ctx, noUname, DockerInfo{Architecture: "aarch64", Flavor: FlavorDockerDesktop}); got != "linux/arm64" {
		t.Fatalf("got %q, want the engine's reading when uname is unavailable", got)
	}

	// No engine reading (daemon down, partial info) → the host's word, even
	// on a flavor whose engine reading would normally win.
	noArch := newFakeExecutor().script("uname -m", executor.Result{Stdout: "x86_64\n"})
	if got := EnginePlatform(ctx, noArch, DockerInfo{Flavor: FlavorDockerEngine}); got != "linux/amd64" {
		t.Fatalf("got %q, want the host's reading when the engine reported none", got)
	}

	// Neither readable → this app's OWN architecture, never "".
	//
	// Returning "" used to mean "omit --platform", which reads as deferring to
	// the image manifest and in fact defers to DOCKER_DEFAULT_PLATFORM. On an
	// arm64 machine with that set to linux/amd64 it produced an emulated
	// container that reported itself running and answered nothing.
	blind := newFakeExecutor().script("uname -m", executor.Result{Stdout: "sparc64\n"})
	if got := EnginePlatform(ctx, blind, DockerInfo{Architecture: "sparc64"}); got != DefaultPlatform() {
		t.Fatalf("got %q, want the fallback %q when nothing recognizable was reported", got, DefaultPlatform())
	}
}

func TestDefaultPlatform_IsNeverEmpty(t *testing.T) {
	if DefaultPlatform() == "" {
		t.Fatal("an empty default is an omitted --platform, which is the bug")
	}
}

// Emulation is an ARCHITECTURE mismatch. The ARM variant is deliberately not
// part of it: an arm64 engine runs a linux/arm64/v8 image natively, and
// treating that as emulation would flag every ordinary arm64 container.
func TestEmulatedPlatform(t *testing.T) {
	tests := []struct {
		name       string
		image, eng string
		want       bool
	}{
		{"the measured bug: amd64 image on an arm64 engine", "linux/amd64", "linux/arm64", true},
		{"native arm64", "linux/arm64", "linux/arm64", false},
		{"variant is not a mismatch", "linux/arm64/v8", "linux/arm64", false},
		{"spellings are normalized", "linux/x86_64", "linux/amd64", false},
		{"unknown image reading accuses nobody", "", "linux/arm64", false},
		{"unknown engine reading accuses nobody", "linux/amd64", "", false},
		{"a malformed reading accuses nobody", "linux", "linux/arm64", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := EmulatedPlatform(tc.image, tc.eng); got != tc.want {
				t.Fatalf("EmulatedPlatform(%q, %q) = %v, want %v", tc.image, tc.eng, got, tc.want)
			}
		})
	}
}

// An emulated container is RUNNING — the engine really has it running, and
// hiding that would remove the stop/restart the operator needs. What must not
// happen is it reading as healthy, which is what the warning is for.
func TestContainerStatus_EmulatedIsRunningButNotSilent(t *testing.T) {
	st := ContainerStatus{State: StateRunning, Platform: "linux/amd64", EnginePlatform: "linux/arm64", Emulated: true}
	if !st.Running() {
		t.Error("an emulated container is still running")
	}
	w := st.EmulationWarning()
	for _, want := range []string{"linux/amd64", "linux/arm64", "DOCKER_DEFAULT_PLATFORM"} {
		if !strings.Contains(w, want) {
			t.Errorf("warning must name %q: %s", want, w)
		}
	}
	if (ContainerStatus{State: StateRunning}).EmulationWarning() != "" {
		t.Error("a native container must not be warned about")
	}
}

// ---- GatewayContainerConfig ----

func testGateway() catalog.GatewayConfig {
	return catalog.GatewayConfig{
		BindAddr: "127.0.0.1",
		Port:     4100,
		Networks: []catalog.GatewayNetwork{
			{ChainID: 369, Upstreams: []catalog.GatewayUpstream{
				{ID: "local-node", Endpoint: "http://127.0.0.1:8545", Local: true},
				{ID: "fallback-1", Endpoint: "https://rpc.example.com"},
			}},
			{ChainID: 1, Upstreams: []catalog.GatewayUpstream{
				{ID: "mainnet-1", Endpoint: "wss://mainnet.example.com/ws"},
			}},
		},
	}
}

func TestGatewayContainerConfig_ListensWideOnTheContainerPort(t *testing.T) {
	got := GatewayContainerConfig(testGateway(), DockerHostAlias)

	if got.Bind() != "0.0.0.0" {
		t.Fatalf("in-container bind: got %q, want 0.0.0.0", got.Bind())
	}
	if got.HTTP() != 4000 {
		t.Fatalf("in-container port: got %d, want 4000 (the host port lives on the -p mapping)", got.HTTP())
	}
}

func TestGatewayContainerConfig_RewritesLoopbackUpstreamsOnly(t *testing.T) {
	g := testGateway()
	g.Networks[0].Upstreams = append(g.Networks[0].Upstreams,
		catalog.GatewayUpstream{Endpoint: "ws://localhost:8546"},
		catalog.GatewayUpstream{Endpoint: "http://[::1]:8545/path"},
		catalog.GatewayUpstream{Endpoint: "http://100.64.0.7:8545"},
	)
	got := GatewayContainerConfig(g, DockerHostAlias)

	want := []string{
		"http://" + DockerHostAlias + ":8545",
		"https://rpc.example.com",
		"ws://" + DockerHostAlias + ":8546",
		"http://" + DockerHostAlias + ":8545/path",
		"http://100.64.0.7:8545",
	}
	for i, w := range want {
		if got := got.Networks[0].Upstreams[i].Endpoint; got != w {
			t.Fatalf("upstream %d: got %q, want %q", i, got, w)
		}
	}
}

func TestGatewayContainerConfig_EmptyAliasDisablesUpstreamRewrite(t *testing.T) {
	got := GatewayContainerConfig(testGateway(), "")
	if ep := got.Networks[0].Upstreams[0].Endpoint; ep != "http://127.0.0.1:8545" {
		t.Fatalf("got %q, want the endpoint untouched", ep)
	}
}

// The deep copy is the point: a shallow one would rewrite the operator's own
// upstream slice through the shared backing array.
func TestGatewayContainerConfig_DoesNotMutateCaller(t *testing.T) {
	g := testGateway()
	_ = GatewayContainerConfig(g, DockerHostAlias)

	if ep := g.Networks[0].Upstreams[0].Endpoint; ep != "http://127.0.0.1:8545" {
		t.Fatalf("caller's config was mutated: upstream endpoint is now %q", ep)
	}
	if g.BindAddr != "127.0.0.1" || g.Port != 4100 {
		t.Fatalf("caller's config was mutated: %+v", g)
	}
}

func TestGatewayContainerConfig_RendersUsableContainerConfig(t *testing.T) {
	yaml, err := catalog.RenderGatewayConfig(GatewayContainerConfig(testGateway(), DockerHostAlias))
	if err != nil {
		t.Fatalf("RenderGatewayConfig: %v", err)
	}
	for _, want := range []string{
		`httpHostV4: "0.0.0.0"`,
		"httpPortV4: 4000",
		`endpoint: "http://host.docker.internal:8545"`,
		"chainId: 1",
	} {
		if !strings.Contains(yaml, want) {
			t.Fatalf("want %q in:\n%s", want, yaml)
		}
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
		case ERPCImageTag():
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

func TestERPCImageTag_DerivedFromPinnedSHA(t *testing.T) {
	tag := ERPCImageTag()
	if !strings.HasPrefix(tag, erpcImageRepo+":") {
		t.Fatalf("tag %q should be under %q", tag, erpcImageRepo)
	}
	// Tagging by source SHA is what makes a rebuild skippable and what makes
	// bumping the ref produce a distinct image rather than replacing one.
	if !strings.HasPrefix(ERPCSourceRef, strings.TrimPrefix(tag, erpcImageRepo+":")) {
		t.Errorf("tag %q is not derived from ERPCSourceRef %q", tag, ERPCSourceRef)
	}
	if strings.Contains(tag, "latest") {
		t.Errorf("gateway image must not be :latest, got %q", tag)
	}
}

func TestERPCBuildContext_PinsFullSHA(t *testing.T) {
	got := ERPCBuildContext()
	if got != ERPCSourceRepo+"#"+ERPCSourceRef {
		t.Fatalf("build context = %q", got)
	}
	// valve-ws is a moving feature branch off an open upstream PR; building
	// from the branch name would change an operator's gateway between runs.
	if strings.HasSuffix(got, "#valve-ws") {
		t.Error("build context must pin a commit, not the branch head")
	}
	if len(ERPCSourceRef) != 40 {
		t.Errorf("ERPCSourceRef should be a full 40-char SHA, got %d chars", len(ERPCSourceRef))
	}
}

func TestImageBuildArgs(t *testing.T) {
	tests := []struct {
		name string
		spec ImageBuildSpec
		want []string
	}{
		{
			name: "defaults",
			spec: ImageBuildSpec{Platform: "linux/arm64"},
			want: []string{"build", "--platform", "linux/arm64", "-t", ERPCImageTag(), ERPCBuildContext()},
		},
		{
			name: "explicit tag and context",
			spec: ImageBuildSpec{Tag: "x:1", Context: "/src", Platform: "linux/amd64"},
			want: []string{"build", "--platform", "linux/amd64", "-t", "x:1", "/src"},
		},
	}
	// The "unrecognized arch omits --platform entirely" branch is governed by
	// PlatformForArch returning "", which TestPlatformForArch already pins;
	// a wrong --platform is worse than none, since it converts correct
	// manifest selection into a hard failure.
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ImageBuildArgs(tc.spec)
			if len(got) != len(tc.want) {
				t.Fatalf("args = %v, want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("args = %v, want %v", got, tc.want)
				}
			}
			// docker reads the first non-flag argument as the context, so it
			// must be last or a trailing option would be read as a second one.
			if got[len(got)-1] != tc.spec.Context && tc.spec.Context != "" {
				t.Errorf("context must be the final argument, got %v", got)
			}
		})
	}
}

// The regression this guards was measured on a macOS host with Homebrew
// `coreutils` installed: an x86_64 GNU uname ahead of /usr/bin/uname in PATH
// reports "x86_64" on Apple Silicon, EnginePlatform believes the host over a
// VM-backed engine, and `docker run --platform linux/amd64` of a locally
// built arm64 image goes off to PULL — surfacing as "pull access denied" for
// an image already on disk. `command -p` runs uname from the system's default
// PATH, where no shim can answer for it.
func TestUnameArchProbe_IgnoresPATHShims(t *testing.T) {
	if !strings.HasPrefix(unameArchProbe, "command -p ") {
		t.Fatalf("unameArchProbe is %q — it must resolve uname from the default PATH, or a shim can misreport the host architecture", unameArchProbe)
	}
}

// ---------------------------------------------------------------------
// the private network and the TLS front
// ---------------------------------------------------------------------

func TestERPCRunArgs_JoinsTheNetworkAndCanPublishNothing(t *testing.T) {
	fronted := ERPCRunArgs(ERPCRunSpec{
		ContainerName:  "valve-node-app-erpc",
		HostConfigPath: "/home/o/.valve-node-app/erpc.yaml",
		Network:        NetworkName,
		NoPublish:      true,
	})
	joined := strings.Join(fronted, " ")
	if !strings.Contains(joined, "--network "+NetworkName) {
		t.Errorf("want the private network: %v", fronted)
	}
	if strings.Contains(joined, "-p ") {
		// Caddy is the only front door; a published eRPC port would be a
		// second, plaintext, unauthenticated way in that nobody asked for.
		t.Errorf("a fronted gateway publishes nothing: %v", fronted)
	}
	// The config mount survives — it is how the gateway is configured at all.
	if !strings.Contains(joined, "/home/o/.valve-node-app/erpc.yaml:/erpc.yaml:ro") {
		t.Errorf("want the read-only config mount: %v", fronted)
	}

	plain := ERPCRunArgs(ERPCRunSpec{HostPort: 4100, Network: NetworkName})
	if !strings.Contains(strings.Join(plain, " "), "-p 127.0.0.1:4100:4000") {
		t.Errorf("an unfronted gateway still publishes: %v", plain)
	}
}

// --platform is emitted on EVERY run and build, never conditionally. See
// resolveRunPlatform for the measured failure an omitted flag produced.
func TestRunAndBuildArgs_AlwaysCarryAPlatform(t *testing.T) {
	if v := valueAfter(t, ERPCRunArgs(ERPCRunSpec{}), "--platform"); v != DefaultPlatform() {
		t.Errorf("run platform = %q, want %q", v, DefaultPlatform())
	}
	if v := valueAfter(t, ImageBuildArgs(ImageBuildSpec{}), "--platform"); v != DefaultPlatform() {
		t.Errorf("build platform = %q, want %q", v, DefaultPlatform())
	}
	if v := valueAfter(t, CaddyRunArgs(CaddyRunSpec{}), "--platform"); v != DefaultPlatform() {
		t.Errorf("caddy platform = %q, want %q", v, DefaultPlatform())
	}
}

func TestCaddyRunArgs(t *testing.T) {
	args := CaddyRunArgs(CaddyRunSpec{
		ContainerName:  "valve-node-app-caddy",
		HostConfigPath: "/home/o/.valve-node-app/Caddyfile",
		Network:        NetworkName,
		HostPort:       8443,
	})
	joined := strings.Join(args, " ")

	for _, want := range []string{
		"--network " + NetworkName,
		// The default bind is WIDE — a TLS front on loopback serves only the
		// machine that never needed it.
		"-p 0.0.0.0:8443:443",
		"/home/o/.valve-node-app/Caddyfile:/etc/caddy/Caddyfile:ro",
		// NOT optional: Caddy's internal CA lives in /data, and without a
		// persistent volume it is regenerated on every recreate — measured, a
		// different root fingerprint and every issued certificate invalidated.
		catalog.CaddyDataVolume + ":" + catalog.CaddyDataPath,
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("missing %q: %v", want, args)
		}
	}
	// Nothing may follow the image ref: anything there REPLACES the image's
	// CMD, which is what makes the config mount alone work.
	if args[len(args)-1] != catalog.DefaultCaddyImage {
		t.Errorf("the image ref must be last, got %v", args)
	}
}

func TestCaddyRunArgs_MountsCertFilesAtTheSamePathBothSides(t *testing.T) {
	args := CaddyRunArgs(CaddyRunSpec{
		CertFile: "/var/lib/valve-node-app/tls/cert.pem",
		KeyFile:  "/var/lib/valve-node-app/tls/key.pem",
	})
	joined := strings.Join(args, " ")
	// One path true on both sides, so the Caddyfile can name it once.
	for _, want := range []string{
		"/var/lib/valve-node-app/tls/cert.pem:/var/lib/valve-node-app/tls/cert.pem:ro",
		"/var/lib/valve-node-app/tls/key.pem:/var/lib/valve-node-app/tls/key.pem:ro",
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("missing %q: %v", want, args)
		}
	}
	// With no cert files there are exactly two mounts: the Caddyfile and the
	// data volume. Counting them is the check, because the Caddyfile mount is
	// itself read-only and a bare ":ro" search would always match.
	if n := strings.Count(strings.Join(CaddyRunArgs(CaddyRunSpec{}), " "), "-v "); n != 2 {
		t.Errorf("want only the Caddyfile and data-volume mounts, got %d: %v", n, CaddyRunArgs(CaddyRunSpec{}))
	}
}

// A wipe of the TLS front must NOT take the certificate authority with it by
// default: wiping chain data is routine, invalidating every trust-store
// install on every device the operator owns is not.
func TestCaddyService_WipeKeepsTheCAUnlessAskedOtherwise(t *testing.T) {
	if v := CaddyServiceKeepingCA("default").Volumes; len(v) != 0 {
		t.Errorf("the default front declares no volumes, got %v", v)
	}
	if v := CaddyServiceFor("default").Volumes; len(v) != 1 || v[0] != catalog.CaddyDataVolume {
		t.Errorf("the explicit-CA form declares the data volume, got %v", v)
	}
	if got := CaddyContainerNameFor("default"); got != CaddyContainerName {
		t.Errorf("the default gateway keeps the bare name, got %q", got)
	}
	if got := CaddyContainerNameFor("edge"); got != CaddyContainerName+"-edge" {
		t.Errorf("got %q", got)
	}
}

func TestEnsureNetwork_IsIdempotent(t *testing.T) {
	ctx := context.Background()

	// Already there: inspect succeeds and nothing is created.
	existing := newFakeExecutor()
	if err := EnsureNetwork(ctx, existing, NetworkName); err != nil {
		t.Fatalf("EnsureNetwork: %v", err)
	}
	for _, c := range existing.callLog() {
		if strings.Contains(c, "'create'") {
			t.Fatalf("must not create a network that already exists: %v", existing.callLog())
		}
	}

	// Absent: inspect fails, create runs.
	absent := newFakeExecutor().script("'network' 'inspect'", executor.Result{ExitCode: 1, Stderr: "Error: No such network"})
	if err := EnsureNetwork(ctx, absent, NetworkName); err != nil {
		t.Fatalf("EnsureNetwork: %v", err)
	}
	var created bool
	for _, c := range absent.callLog() {
		created = created || strings.Contains(c, "'network' 'create'")
	}
	if !created {
		t.Fatalf("want a create: %v", absent.callLog())
	}

	// Lost a race with another provision: "already exists" is the goal state.
	raced := newFakeExecutor().
		script("'network' 'inspect'", executor.Result{ExitCode: 1}).
		script("'network' 'create'", executor.Result{ExitCode: 1, Stderr: "Error response from daemon: network with name valve-node-app already exists"})
	if err := EnsureNetwork(ctx, raced, NetworkName); err != nil {
		t.Fatalf("losing the creation race is not a failure: %v", err)
	}
}
