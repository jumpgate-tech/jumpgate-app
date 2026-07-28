package server

import (
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/valve-tech/valve-node-app/internal/catalog"
	"github.com/valve-tech/valve-node-app/internal/config"
	"github.com/valve-tech/valve-node-app/internal/executor"
)

// ---------------------------------------------------------------------
// a fleet of fakes, one per machine
// ---------------------------------------------------------------------

// fleet hands every target its OWN scriptedExecutor and keeps them, so a test
// can ask what a specific MACHINE was told to do.
//
// Per-machine rather than one shared fake is the whole point here: the
// behaviour under test is that a command lands on the box the gateway is
// actually placed on, and a single shared executor cannot tell "restarted the
// right container on the wrong machine" from "restarted it on the right one".
type fleet struct {
	mu    sync.Mutex
	execs map[string]*scriptedExecutor
	// state maps a target id to the `docker inspect` state line its executor
	// reports; anything absent gets a running container.
	state map[string]string
}

func newFleet() *fleet {
	return &fleet{execs: map[string]*scriptedExecutor{}, state: map[string]string{}}
}

func (f *fleet) factory(t config.Target) (executor.Executor, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if e, ok := f.execs[t.ID]; ok {
		return e, nil
	}
	st, ok := f.state[t.ID]
	if !ok {
		st = "true|0|img|sha256:abc\n"
	}
	e := fleetExecutor(st)
	f.execs[t.ID] = e
	return e, nil
}

// fleetExecutor is dockerExecutor plus the target's $HOME, which the docker
// backend probes to place erpc.yaml somewhere the engine can bind-mount. A
// machine whose $HOME does not answer fails every gateway create, so a fake
// that leaves it empty can only test failure paths.
func fleetExecutor(state string) *scriptedExecutor {
	return dockerExecutor(state, "").script(`"$HOME"`, executor.Result{Stdout: "/home/ops\n"})
}

// commands returns every command this machine's executor was asked to run.
func (f *fleet) commands(t *testing.T, targetID string) []string {
	t.Helper()
	f.mu.Lock()
	e, ok := f.execs[targetID]
	f.mu.Unlock()
	if !ok {
		t.Fatalf("no executor was ever created for machine %q", targetID)
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	return append([]string(nil), e.calls...)
}

// ran reports whether this machine was asked to run a command containing all
// of the given fragments — the docker verb and the container name together, so
// "restart" on some OTHER container does not count as a match.
func (f *fleet) ran(t *testing.T, targetID string, fragments ...string) bool {
	t.Helper()
	for _, cmd := range f.commands(t, targetID) {
		all := true
		for _, frag := range fragments {
			if !strings.Contains(cmd, frag) {
				all = false
				break
			}
		}
		if all {
			return true
		}
	}
	return false
}

func addSSHTarget(t *testing.T, a *apiTestServer, id string) {
	t.Helper()
	res := a.do(t, "POST", "/api/targets", map[string]any{
		"id":   id,
		"mode": "ssh",
		"ssh":  map[string]string{"host": id + ".example", "user": "ops", "keyPath": "/dev/null"},
	})
	defer res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("POST /api/targets %s: got %d, want 201", id, res.StatusCode)
	}
}

// devnetFronting builds the config for a gateway that fronts the devnet on
// targetID, which is what makes it eligible for the reset cascade.
func devnetFronting(port int, targetID string) catalog.GatewayConfig {
	return catalog.GatewayConfig{
		Port: port,
		Networks: []catalog.GatewayNetwork{{
			ChainID: catalog.DevnetChainID,
			Upstreams: []catalog.GatewayUpstream{
				{ID: "devnet", Kind: catalog.UpstreamManagedDevnet, TargetID: targetID},
			},
		}},
	}
}

// ---------------------------------------------------------------------
// reset is the devnet's, and only the devnet's
// ---------------------------------------------------------------------

// A one-click reset on a real chain node would be a one-click discard of
// hundreds of gigabytes and days of resync. The route refuses it by NAME
// rather than by any property of the request, so no combination of body or
// config can reach it.
func TestReset_RefusedForAnythingButTheDevnet(t *testing.T) {
	f := newFleet()
	a := newAPITestServerWithExecutor(t, f.factory)
	addTarget(t, a)

	for _, svc := range []string{"erpc", "gateway", "reth", "node", ""} {
		res := a.do(t, "POST", "/api/targets/local/containers/"+svc+"/reset", nil)
		body := decode[errorDetail](t, res)
		if res.StatusCode != http.StatusNotFound {
			t.Errorf("reset %q: got %d, want 404", svc, res.StatusCode)
			continue
		}
		// The message has to point at the real path for a node, or the
		// operator concludes the feature is missing and looks for another way.
		if svc != "" && !strings.Contains(body.Error, "clear-and-resync") {
			t.Errorf("reset %q: message does not name the guarded path a node actually has: %q", svc, body.Error)
		}
	}
}

// Reset deliberately has NO typed confirmation — that is the one thing
// separating it from wipe. A devnet's whole value is being cheap to throw
// away, and making the routine case type its own name is friction pretending
// to be safety.
func TestReset_NeedsNoTypedConfirmation(t *testing.T) {
	f := newFleet()
	a := newAPITestServerWithExecutor(t, f.factory)
	addTarget(t, a)
	putConfig(t, a, svcDevnet, catalog.DevnetConfig{HTTPPort: 8600, WSPort: 8601})

	res := a.do(t, "POST", "/api/targets/local/containers/devnet/reset", nil)
	body := decode[wipeResponse](t, res)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("got %d (%s), want 200 with no confirmation at all", res.StatusCode, body.Error)
	}
	if !f.ran(t, "local", "'rm'", "'"+catalog.DevnetContainerName+"'") {
		t.Errorf("the devnet container was never removed; commands: %q", f.commands(t, "local"))
	}
}

// ---------------------------------------------------------------------
// the cross-machine cascade — the stale-head bug, one box over
// ---------------------------------------------------------------------

// The bug this exists to prevent, measured on real containers: a reset chain
// restarts at a low block while eRPC keeps advertising the old head, because
// eRPC's per-network head is monotonic and never observes a reset. The gateway
// then reports itself perfectly healthy while answering with a head the chain
// no longer has.
//
// ops' own cascade holds ONE executor, so it can only reach fronts on the
// devnet's own machine. A gateway on another box is invisible to it — and is
// exactly the case where the operator has no reason to suspect anything, since
// nothing on the machine they reset looks wrong.
func TestReset_RestartsAGatewayFrontingItFromAnotherMachine(t *testing.T) {
	f := newFleet()
	a := newAPITestServerWithExecutor(t, f.factory)
	addTarget(t, a)
	addSSHTarget(t, a, "boxa")
	putConfig(t, a, svcDevnet, catalog.DevnetConfig{HTTPPort: 8600, WSPort: 8601})

	addGateway(t, a, "edge", "boxa", devnetFronting(4200, "local"))

	res := a.do(t, "POST", "/api/targets/local/containers/devnet/reset", nil)
	body := decode[wipeResponse](t, res)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("got %d (%s), want 200", res.StatusCode, body.Error)
	}

	// The command went to boxa, naming boxa's gateway container.
	if !f.ran(t, "boxa", "'restart'", "'valve-node-app-erpc-edge'") {
		t.Errorf("the remote gateway was never restarted; boxa ran: %q", f.commands(t, "boxa"))
	}
	// And it is REPORTED, in the same list as the local cascade: an operator
	// who is not told a gateway was bounced cannot tell this apart from the
	// bug it prevents.
	if !contains(body.Report.Cascaded, "erpc:edge") {
		t.Errorf("cascaded: got %q, want the remote gateway named", body.Report.Cascaded)
	}
}

// A gateway that is not running has no stale head to clear, so restarting it
// would silently START something the operator had deliberately stopped. It is
// reported as skipped instead — same list, different half.
func TestReset_DoesNotStartAStoppedRemoteGateway(t *testing.T) {
	f := newFleet()
	f.state["boxa"] = "false|0|img|sha256:abc\n"
	a := newAPITestServerWithExecutor(t, f.factory)
	addTarget(t, a)
	addSSHTarget(t, a, "boxa")
	putConfig(t, a, svcDevnet, catalog.DevnetConfig{HTTPPort: 8600, WSPort: 8601})

	addGateway(t, a, "edge", "boxa", devnetFronting(4200, "local"))

	body := decode[wipeResponse](t, a.do(t, "POST", "/api/targets/local/containers/devnet/reset", nil))

	if f.ran(t, "boxa", "'restart'", "'valve-node-app-erpc-edge'") {
		t.Error("a stopped gateway was restarted — that starts something the operator had stopped")
	}
	if !contains(body.Report.CascadeSkipped, "erpc:edge") {
		t.Errorf("cascadeSkipped: got %q, want the stopped gateway named", body.Report.CascadeSkipped)
	}
	if contains(body.Report.Cascaded, "erpc:edge") {
		t.Error("a stopped gateway must not be reported as restarted")
	}
}

// The cascade is keyed on what a gateway actually FRONTS. A gateway on another
// machine serving a different chain has nothing stale about it, and bouncing it
// would be dropping in-flight calls on an unrelated endpoint.
func TestReset_LeavesAGatewayThatDoesNotFrontThisDevnetAlone(t *testing.T) {
	f := newFleet()
	a := newAPITestServerWithExecutor(t, f.factory)
	addTarget(t, a)
	addSSHTarget(t, a, "boxa")
	putConfig(t, a, svcDevnet, catalog.DevnetConfig{HTTPPort: 8600, WSPort: 8601})

	addGateway(t, a, "edge", "boxa", catalog.GatewayConfig{
		Port: 4200,
		Networks: []catalog.GatewayNetwork{{ChainID: 369, Upstreams: []catalog.GatewayUpstream{
			{ID: "public", Endpoint: "https://rpc.pulsechain.com"},
		}}},
	})

	body := decode[wipeResponse](t, a.do(t, "POST", "/api/targets/local/containers/devnet/reset", nil))

	if f.ran(t, "boxa", "'restart'", "'valve-node-app-erpc-edge'") {
		t.Error("a gateway serving an unrelated chain was restarted")
	}
	if contains(body.Report.Cascaded, "erpc:edge") || contains(body.Report.CascadeSkipped, "erpc:edge") {
		t.Errorf("an unrelated gateway appears in the cascade report: %+v", body.Report)
	}
}

// A devnet referenced by its URL rather than by a managed reference is still
// fronted by that gateway — an operator who typed the address gets the same
// protection as one who picked it from the list.
func TestReset_CascadesToAGatewayThatNamedTheDevnetByURL(t *testing.T) {
	f := newFleet()
	a := newAPITestServerWithExecutor(t, f.factory)
	addTarget(t, a)
	addSSHTarget(t, a, "boxa")
	putConfig(t, a, svcDevnet, catalog.DevnetConfig{HTTPPort: 8600, WSPort: 8601})

	d := catalog.DevnetConfig{HTTPPort: 8600, WSPort: 8601}
	d.ChainID = catalog.DevnetChainID
	addGateway(t, a, "edge", "boxa", catalog.GatewayConfig{
		Port: 4200,
		Networks: []catalog.GatewayNetwork{{ChainID: catalog.DevnetChainID, Upstreams: []catalog.GatewayUpstream{
			{ID: "typed", Endpoint: d.HTTPEndpoint()},
		}}},
	})

	body := decode[wipeResponse](t, a.do(t, "POST", "/api/targets/local/containers/devnet/reset", nil))

	if !f.ran(t, "boxa", "'restart'", "'valve-node-app-erpc-edge'") {
		t.Errorf("a gateway pointing at the devnet's URL was not restarted; boxa ran: %q", f.commands(t, "boxa"))
	}
	if !contains(body.Report.Cascaded, "erpc:edge") {
		t.Errorf("cascaded: got %q", body.Report.Cascaded)
	}
}

// A gateway placed on a machine that has since been removed cannot be reached,
// and the reset has ALREADY happened by then. Reporting 502 with the reason is
// the only honest answer: the chain is reset and something out there is still
// serving its old head.
func TestReset_ReportsAGatewayItCouldNotReach(t *testing.T) {
	f := newFleet()
	a := newAPITestServerWithExecutor(t, f.factory)
	addTarget(t, a)
	addSSHTarget(t, a, "boxa")
	putConfig(t, a, svcDevnet, catalog.DevnetConfig{HTTPPort: 8600, WSPort: 8601})
	addGateway(t, a, "edge", "boxa", devnetFronting(4200, "local"))

	// The machine goes; the gateway record stays, which is what a delete does.
	res := a.do(t, "DELETE", "/api/targets/boxa", nil)
	res.Body.Close()

	res = a.do(t, "POST", "/api/targets/local/containers/devnet/reset", nil)
	body := decode[wipeResponse](t, res)
	if res.StatusCode != http.StatusBadGateway {
		t.Fatalf("got %d, want 502 — the reset happened but a front could not be cleared", res.StatusCode)
	}
	// The devnet itself was still reset: reporting the cascade failure must
	// not imply nothing happened.
	if !body.Report.ContainerRemoved {
		t.Error("the devnet container was not removed, so the 502 is describing the wrong failure")
	}
	for _, want := range []string{"edge", "restart it by hand"} {
		if !strings.Contains(body.Error, want) {
			t.Errorf("error %q does not mention %q", body.Error, want)
		}
	}
}

func contains(haystack []string, needle string) bool {
	for _, v := range haystack {
		if v == needle {
			return true
		}
	}
	return false
}
