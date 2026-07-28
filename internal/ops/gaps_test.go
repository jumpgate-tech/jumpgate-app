package ops

// The remaining edges: name sanitization, the platform readings that must
// never come back empty, the idempotent network create, and the two error
// types whose whole value is the sentence they print.

import (
	"context"
	"errors"
	"runtime"
	"strings"
	"testing"

	"github.com/valve-tech/valve-node-app/internal/executor"
)

// ---------------------------------------------------------------------
// names
// ---------------------------------------------------------------------

// The id lands in a docker --name, so anything outside docker's grammar
// becomes '-'. This is the second lock on a door the API also guards: a
// sloppy or hostile id must not be able to smuggle a flag or a shell
// metacharacter onto a command line.
func TestERPCContainerNameFor_SanitizesWhatReachesTheCommandLine(t *testing.T) {
	// The default id keeps the historical unsuffixed name, which is what
	// lets an existing install's container still be found.
	if got := ERPCContainerNameFor(DefaultGatewayID); got != ERPCContainerName {
		t.Errorf("got %q, want the unsuffixed %q", got, ERPCContainerName)
	}
	if got := ERPCContainerNameFor(""); got != ERPCContainerName {
		t.Errorf("empty id = %q, want the unsuffixed name", got)
	}
	if got := ERPCContainerNameFor("   "); got != ERPCContainerName {
		t.Errorf("whitespace id = %q, want the unsuffixed name", got)
	}

	hostile := []string{
		"--privileged",
		"a b; rm -rf /",
		"$(whoami)",
		"a/b",
		"'quoted'",
	}
	for _, id := range hostile {
		t.Run(id, func(t *testing.T) {
			got := ERPCContainerNameFor(id)
			for _, bad := range []string{" ", ";", "$", "(", ")", "'", "\"", "/", "`", "|", "&"} {
				if strings.Contains(got, bad) {
					t.Errorf("%q survived into the container name %q", bad, got)
				}
			}
			if got == ERPCContainerName {
				t.Errorf("a non-default id collapsed onto the default name: %q", got)
			}
		})
	}

	// Distinct ids stay distinct, or two gateways share one container.
	if ERPCContainerNameFor("alpha") == ERPCContainerNameFor("beta") {
		t.Error("two gateway ids produced one container name")
	}
}

// ---------------------------------------------------------------------
// platform
// ---------------------------------------------------------------------

// DefaultPlatform must NEVER answer "". An empty --platform does not hand the
// choice to the engine's manifest resolution — it hands it to
// DOCKER_DEFAULT_PLATFORM, which is how QEMU-emulated containers that report
// State: running and answer nothing get created.
func TestDefaultPlatform_IsDerivedFromGOARCH(t *testing.T) {
	got := DefaultPlatform()
	if !strings.HasPrefix(got, "linux/") {
		t.Errorf("got %q, want a linux/* platform", got)
	}
	// Go's GOARCH names and docker's platform arch names agree for every
	// architecture either supports, so an unrecognized arch still renders.
	if got != PlatformForArch(runtime.GOARCH) && got != "linux/"+runtime.GOARCH {
		t.Errorf("got %q, want it derived from GOARCH %q", got, runtime.GOARCH)
	}
}

// The engine emits several shapes here, and only a complete descriptor counts.
// Guessing from a partial one is how a container gets run as the wrong arch.
func TestParseContainerPlatform_OnlyACompleteDescriptorCounts(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "a full descriptor", in: `{"platform":{"os":"linux","architecture":"arm64"}}`, want: "linux/arm64"},
		{name: "amd64", in: `{"platform":{"os":"linux","architecture":"amd64"}}`, want: "linux/amd64"},
		{name: "leading blank lines", in: "\n\n" + `{"platform":{"os":"linux","architecture":"arm64"}}`, want: "linux/arm64"},
		{name: "a JSON null", in: "null", want: ""},
		{name: "empty output", in: "", want: ""},
		{name: "only whitespace", in: "  \n\t\n", want: ""},
		{name: "not JSON", in: "Error: No such object", want: ""},
		{name: "no platform member", in: `{"digest":"sha256:abc"}`, want: ""},
		// A half-filled descriptor is worse than none: "linux/" or "/arm64"
		// would be passed to --platform verbatim.
		{name: "os without arch", in: `{"platform":{"os":"linux"}}`, want: ""},
		{name: "arch without os", in: `{"platform":{"architecture":"arm64"}}`, want: ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := parseContainerPlatform(tc.in); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------
// EnsureNetwork
// ---------------------------------------------------------------------

func TestEnsureNetwork_RefusesAnEmptyName(t *testing.T) {
	err := EnsureNetwork(context.Background(), newFakeExecutor(), "   ")
	if err == nil {
		t.Fatal("an empty network name was accepted")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("error does not say what is wrong: %v", err)
	}
}

// An existing network is left alone — no create is attempted at all.
func TestEnsureNetwork_AnExistingNetworkIsNotRecreated(t *testing.T) {
	e := newFakeExecutor().script("'network' 'inspect'", executor.Result{ExitCode: 0})

	if err := EnsureNetwork(context.Background(), e, NetworkName); err != nil {
		t.Fatalf("EnsureNetwork: %v", err)
	}
	for _, c := range e.callLog() {
		if strings.Contains(c, "'network' 'create'") {
			t.Errorf("created a network that already existed: %s", c)
		}
	}
}

// The race this exists for: two runs create the same network at once, and the
// loser's non-zero exit says "already exists". Treating that as a failure
// would make concurrent provisioning flaky for no reason.
func TestEnsureNetwork_ALostCreateRaceIsSuccess(t *testing.T) {
	e := newFakeExecutor().
		script("'network' 'inspect'", executor.Result{ExitCode: 1, Stderr: "Error: No such network"}).
		script("'network' 'create'", executor.Result{
			ExitCode: 1,
			Stderr:   `Error response from daemon: network with name valve-node-app already exists`,
		})

	if err := EnsureNetwork(context.Background(), e, NetworkName); err != nil {
		t.Fatalf("a lost create race was reported as a failure: %v", err)
	}
}

// Any OTHER non-zero create is a real failure and must not be swallowed —
// every container that follows would fail to attach.
func TestEnsureNetwork_ARealCreateFailureIsReported(t *testing.T) {
	e := newFakeExecutor().
		script("'network' 'inspect'", executor.Result{ExitCode: 1}).
		script("'network' 'create'", executor.Result{
			ExitCode: 1,
			Stderr:   "Error response from daemon: could not find an available, non-overlapping IPv4 address pool",
		})

	err := EnsureNetwork(context.Background(), e, NetworkName)
	if err == nil {
		t.Fatal("a failed network create reported success")
	}
	if !strings.Contains(err.Error(), "address pool") {
		t.Errorf("the engine's own explanation was dropped: %v", err)
	}
}

func TestEnsureNetwork_ATransportFailureSurfaces(t *testing.T) {
	boom := errors.New("ssh: connection lost")

	for _, at := range []string{"'network' 'inspect'", "'network' 'create'"} {
		t.Run(at, func(t *testing.T) {
			e := newFakeExecutor().script("'network' 'inspect'", executor.Result{ExitCode: 1})
			e.errOn(at, boom)

			err := EnsureNetwork(context.Background(), e, NetworkName)
			if err == nil {
				t.Fatalf("a broken link at %s reported success", at)
			}
			if !errors.Is(err, boom) {
				t.Errorf("the transport cause did not survive: %v", err)
			}
		})
	}
}

// ---------------------------------------------------------------------
// the error types
// ---------------------------------------------------------------------

// ServiceNotCreatedError's whole value is the sentence: "start" failed
// because there is nothing to start, and the fix is setup, not a retry.
func TestServiceNotCreatedError_SaysWhatToDoInstead(t *testing.T) {
	err := &ServiceNotCreatedError{ID: "erpc:default", ContainerName: "valve-node-app-erpc", Action: "start"}

	msg := err.Error()
	for _, want := range []string{"start", "erpc:default", "valve-node-app-erpc", "setup"} {
		if !strings.Contains(msg, want) {
			t.Errorf("the message does not mention %q: %s", want, msg)
		}
	}
	// It unwraps to the sentinel, so callers can branch on the KIND of
	// failure without matching on the sentence.
	if !errors.Is(err, ErrServiceNotCreated) {
		t.Error("does not unwrap to ErrServiceNotCreated")
	}
}

// ---------------------------------------------------------------------
// FreeBytesAt
// ---------------------------------------------------------------------

// FreeBytesAt walks up to the nearest EXISTING ancestor first, because the
// path may not exist yet — a data location the operator is still considering
// in the setup wizard, before anything is created.
func TestFreeBytesAt_WalksUpToAnExistingAncestor(t *testing.T) {
	e := newFakeExecutor().script("df -B1 --output=avail", executor.Result{Stdout: "Avail\n4000000000000\n"})

	got, err := FreeBytesAt(context.Background(), e, "/mnt/does/not/exist/yet")
	if err != nil {
		t.Fatalf("FreeBytesAt: %v", err)
	}
	if got != 4000000000000 {
		t.Errorf("got %d, want 4000000000000", got)
	}

	// The walk is in the command itself, not a pre-check, so it happens on
	// the TARGET rather than against the control plane's own filesystem.
	var probed string
	for _, c := range e.callLog() {
		if strings.Contains(c, "df -B1") {
			probed = c
		}
	}
	if !strings.Contains(probed, "while [ ! -d") {
		t.Errorf("the ancestor walk is not in the command: %s", probed)
	}
	if !strings.Contains(probed, "/mnt/does/not/exist/yet") {
		t.Errorf("the command does not name the path asked about: %s", probed)
	}
}

func TestFreeBytesAt_SurfacesAFailedProbe(t *testing.T) {
	boom := errors.New("ssh: connection lost")
	e := newFakeExecutor().errOn("df -B1", boom)

	if _, err := FreeBytesAt(context.Background(), e, "/mnt/reth"); !errors.Is(err, boom) {
		t.Fatalf("got %v, want it to wrap the transport failure", err)
	}
}
