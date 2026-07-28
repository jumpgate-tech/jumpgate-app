package ops

// The diagnostics ladder and the docker image helpers. The ladder's whole
// design is that it STOPS at the first hard failure, because every rung below
// a dead service would report a second failure caused by the first — and an
// operator handed six red rows cannot tell which one to act on.

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/valve-tech/valve-node-app/internal/executor"
)

// ---------------------------------------------------------------------
// the ladder stops at the first failure
// ---------------------------------------------------------------------

// A dead service makes every later probe fail for the same reason. Running
// them anyway produces a wall of red whose actual cause is the first row, so
// the ladder stops — and the stopping is the feature.
func TestNetworkDiagnostics_StopsAtADeadService(t *testing.T) {
	e := newFakeExecutor().script("systemctl is-active", executor.Result{ExitCode: 3, Stdout: "inactive\ninactive\n"})

	items, err := NetworkDiagnostics(context.Background(), e, diagWire(), DiagnoseOpts{})
	if err != nil {
		t.Fatalf("NetworkDiagnostics: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("got %d rows, want only the services one: %+v", len(items), items)
	}
	if items[0].Status != "fail" {
		t.Errorf("the services row is %q, want fail", items[0].Status)
	}
	// Nothing below it was even probed.
	for _, c := range e.callLog() {
		if strings.Contains(c, "ss -ltn") {
			t.Errorf("a later probe ran below a failed rung: %q", c)
		}
	}
}

// An unknown chain has no port expectations to check against, so there is no
// honest diagnosis to give — better an error than a ladder of "unknown".
func TestNetworkDiagnostics_RefusesAChainItDoesNotKnow(t *testing.T) {
	w := diagWire()
	w.ChainID = 999999

	if _, err := NetworkDiagnostics(context.Background(), newFakeExecutor(), w, DiagnoseOpts{}); err == nil {
		t.Fatal("diagnostics ran for a chain the catalog does not know")
	}
}

func TestNetworkDiagnostics_RefusesABeaconClientItDoesNotKnow(t *testing.T) {
	w := diagWire()
	w.BeaconID = "not-a-client"

	if _, err := NetworkDiagnostics(context.Background(), newFakeExecutor(), w, DiagnoseOpts{}); err == nil {
		t.Fatal("diagnostics ran for a beacon client with no known p2p ports")
	}
}

// A transport failure is not a diagnosis. Reporting it as one would tell the
// operator their p2p ports are closed when in fact the connection dropped.
func TestNetworkDiagnostics_ATransportFailureIsNotADiagnosis(t *testing.T) {
	// A healthy machine all the way down to the port scan, which is where the
	// connection then drops. Every rung above it must pass, or the ladder
	// stops early and this test would be asserting on the wrong failure.
	e := newFakeExecutor().
		script("systemctl is-active", executor.Result{ExitCode: 0, Stdout: "active\nactive\n"}).
		script("eth_chainId", executor.Result{ExitCode: 0, Stdout: `{"jsonrpc":"2.0","id":1,"result":"0x171"}`}).
		script("/eth/v1/node/version", executor.Result{ExitCode: 0, Stdout: "200"}).
		script("eth_syncing", executor.Result{ExitCode: 0, Stdout: `{"jsonrpc":"2.0","id":1,"result":false}`}).
		script("/eth/v1/node/syncing", executor.Result{ExitCode: 0, Stdout: `{"data":{"is_syncing":false,"sync_distance":"0"}}`}).
		errOn("ss -ltn", errors.New("ssh: connection lost"))

	_, err := NetworkDiagnostics(context.Background(), e, diagWire(), DiagnoseOpts{})
	if err == nil {
		t.Fatal("a dropped connection was reported as a set of findings")
	}
	if !strings.Contains(err.Error(), "connection lost") {
		t.Errorf("error %q loses the transport failure", err)
	}
}

// ---------------------------------------------------------------------
// the sync row
// ---------------------------------------------------------------------

// "Syncing" is not a failure — a node mid-initial-sync is working exactly as
// intended — so the row reports it without raising an alarm, and says how far
// behind the beacon is, which is the number that tells an operator whether to
// wait an hour or a day.
func TestSyncItem_ReportsDistanceWithoutCallingItAFailure(t *testing.T) {
	e := newFakeExecutor().
		script("eth_syncing", executor.Result{ExitCode: 0, Stdout: `{"jsonrpc":"2.0","id":1,"result":{"currentBlock":"0x1","highestBlock":"0x99"}}`}).
		script("/eth/v1/node/syncing", executor.Result{ExitCode: 0, Stdout: `{"data":{"is_syncing":true,"sync_distance":"4321"}}`})

	got := syncItem(context.Background(), e, diagWire())
	if got.Status == "fail" {
		t.Errorf("a syncing node was reported as a failure: %+v", got)
	}
	if !strings.Contains(got.Detail, "4321") {
		t.Errorf("detail %q does not say how far behind the beacon is", got.Detail)
	}
}

func TestSyncItem_BothInSyncIsAPass(t *testing.T) {
	e := newFakeExecutor().
		script("eth_syncing", executor.Result{ExitCode: 0, Stdout: `{"jsonrpc":"2.0","id":1,"result":false}`}).
		script("/eth/v1/node/syncing", executor.Result{ExitCode: 0, Stdout: `{"data":{"is_syncing":false,"sync_distance":"0"}}`})

	if got := syncItem(context.Background(), e, diagWire()); got.Status != "pass" {
		t.Errorf("status %q, want pass: %+v", got.Status, got)
	}
}

// A client that did not answer is "unknown", never "pass". Reporting an
// unanswered probe as in-sync is the exact shape of bug this repo keeps
// finding: a success reported for a check that never ran.
func TestSyncItem_AnUnansweredProbeIsUnknownNotPass(t *testing.T) {
	for name, e := range map[string]*fakeExecutor{
		"exec did not answer": newFakeExecutor().
			script("eth_syncing", executor.Result{ExitCode: 7}).
			script("/eth/v1/node/syncing", executor.Result{ExitCode: 0, Stdout: `{"data":{"is_syncing":false,"sync_distance":"0"}}`}),
		"beacon did not answer": newFakeExecutor().
			script("eth_syncing", executor.Result{ExitCode: 0, Stdout: `{"jsonrpc":"2.0","id":1,"result":false}`}).
			script("/eth/v1/node/syncing", executor.Result{ExitCode: 7}),
		"exec answered garbage": newFakeExecutor().
			script("eth_syncing", executor.Result{ExitCode: 0, Stdout: `<html>502 Bad Gateway</html>`}).
			script("/eth/v1/node/syncing", executor.Result{ExitCode: 0, Stdout: `{"data":{"is_syncing":false,"sync_distance":"0"}}`}),
		"neither answered": newFakeExecutor(),
	} {
		t.Run(name, func(t *testing.T) {
			got := syncItem(context.Background(), e, diagWire())
			if got.Status != "unknown" {
				t.Errorf("status %q, want unknown: %+v", got.Status, got)
			}
		})
	}
}

// ---------------------------------------------------------------------
// asUint — the beacon's sync_distance
// ---------------------------------------------------------------------

// The beacon API returns sync_distance as a JSON string, but some versions and
// proxies send a number. Both have to work, and anything else must be refused
// rather than silently becoming 0 — which would read as "fully synced".
func TestAsUint_AcceptsBothWireShapesAndRefusesTheRest(t *testing.T) {
	for name, tc := range map[string]struct {
		in   any
		want uint64
		ok   bool
	}{
		"string digits":  {"4321", 4321, true},
		"json number":    {float64(4321), 4321, true},
		"zero string":    {"0", 0, true},
		"zero number":    {float64(0), 0, true},
		"negative":       {float64(-1), 0, false},
		"not a number":   {"soon", 0, false},
		"empty string":   {"", 0, false},
		"boolean":        {true, 0, false},
		"nil":            {nil, 0, false},
		"nested object":  {map[string]any{}, 0, false},
		"float with dot": {"12.5", 0, false},
	} {
		t.Run(name, func(t *testing.T) {
			got, ok := asUint(tc.in)
			if ok != tc.ok {
				t.Fatalf("asUint(%#v) ok = %v, want %v", tc.in, ok, tc.ok)
			}
			if got != tc.want {
				t.Errorf("asUint(%#v) = %d, want %d", tc.in, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------
// image helpers
// ---------------------------------------------------------------------

// A build that failed must not be reported as a success — the next step would
// run a container against an image that is not there, and docker's own error
// ("no such image") is a long way from "the build failed, here is why".
func TestBuildImage_AFailedBuildCarriesTheReason(t *testing.T) {
	e := newFakeExecutor().script("docker", executor.Result{
		ExitCode: 1,
		Stderr:   "ERROR: failed to solve: process \"/bin/sh -c cargo build\" did not complete successfully\n",
	})

	_, err := BuildImage(context.Background(), e, "build", "-t", "valve-node-app/erpc:x", ".")
	if err == nil {
		t.Fatal("a failed build reported success")
	}
	if !strings.Contains(err.Error(), "failed to solve") {
		t.Errorf("error %q does not carry the builder's own words", err)
	}
}

// Build arguments are quoted individually. A tag or path with a space in it
// otherwise becomes two arguments, and docker builds something other than what
// was asked for.
func TestBuildImage_QuotesEveryArgument(t *testing.T) {
	e := newFakeExecutor()

	if _, err := BuildImage(context.Background(), e, "build", "-t", "my image:latest", "/src/my dir"); err != nil {
		t.Fatalf("BuildImage: %v", err)
	}
	var cmd string
	for _, c := range e.callLog() {
		if strings.HasPrefix(c, "docker") {
			cmd = c
		}
	}
	for _, want := range []string{"'my image:latest'", "'/src/my dir'"} {
		if !strings.Contains(cmd, want) {
			t.Errorf("command %q does not carry %q as one quoted argument", cmd, want)
		}
	}
}

func TestBuildImage_ATransportFailureIsReported(t *testing.T) {
	e := newFakeExecutor().errOn("docker", errors.New("ssh: connection lost"))

	if _, err := BuildImage(context.Background(), e, "build", "."); err == nil {
		t.Fatal("a dropped connection during a build reported success")
	}
}

// ImageExists is the gate that decides whether to build at all. An engine that
// answered nothing must read as "not there" rather than "there", or the build
// is skipped and the run fails on a missing image.
func TestImageExists_OnlyTrueWhenTheEngineNamesAnID(t *testing.T) {
	for name, tc := range map[string]struct {
		res  executor.Result
		want bool
	}{
		"an id":              {executor.Result{ExitCode: 0, Stdout: "sha256:abc\n"}, true},
		"no such image":      {executor.Result{ExitCode: 1, Stderr: "Error: No such image"}, false},
		"exit zero, no id":   {executor.Result{ExitCode: 0, Stdout: "\n"}, false},
		"exit zero, blanks":  {executor.Result{ExitCode: 0, Stdout: "   \n"}, false},
		"nonzero with an id": {executor.Result{ExitCode: 1, Stdout: "sha256:abc\n"}, false},
	} {
		t.Run(name, func(t *testing.T) {
			e := newFakeExecutor().script("docker image inspect", tc.res)
			got, err := ImageExists(context.Background(), e, "valve-node-app/erpc:x")
			if err != nil {
				t.Fatalf("ImageExists: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestImageExists_ATransportFailureIsNotAbsence(t *testing.T) {
	e := newFakeExecutor().errOn("docker image inspect", errors.New("ssh: connection lost"))

	if _, err := ImageExists(context.Background(), e, "x"); err == nil {
		t.Fatal("a dropped connection was reported as 'the image is not there'")
	}
}
