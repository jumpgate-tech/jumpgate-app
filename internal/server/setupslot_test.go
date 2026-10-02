package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/valve-tech/valve-node-app/internal/catalog"
	"github.com/valve-tech/valve-node-app/internal/config"
	"github.com/valve-tech/valve-node-app/internal/executor"
)

// holdSetupSlot claims id's setup slot exactly as a setup or provision run
// does, and leaves it running until the returned release is called. Claiming
// directly, rather than starting a real run and catching it mid-step, keeps
// these tests free of timing.
func holdSetupSlot(t *testing.T, s *Server, id string) (release func()) {
	t.Helper()
	c, ok := s.claimSetupRun(httptest.NewRecorder(), id)
	if !ok {
		t.Fatalf("could not claim the setup slot on %q", id)
	}
	return func() { s.releaseSetupRun(id, c) }
}

// expectSlotRefusal asserts a destructive route answered 409 because a setup
// run holds the target, and says so.
func expectSlotRefusal(t *testing.T, res *http.Response) {
	t.Helper()
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusConflict {
		t.Fatalf("got %d while a setup run holds the target, want 409; body=%s", res.StatusCode, body)
	}
	if !strings.Contains(string(body), "setup") {
		t.Errorf("the refusal does not say a setup run is the reason: %s", body)
	}
}

func expectStatus(t *testing.T, res *http.Response, want int) {
	t.Helper()
	defer res.Body.Close()
	if res.StatusCode != want {
		body, _ := io.ReadAll(res.Body)
		t.Fatalf("got %d once the slot was free, want %d; body=%s", res.StatusCode, want, body)
	}
}

// A wipe or reset interleaved with a running provision would remove the
// container the provision is creating, or create one against a volume the
// wipe is deleting. Both share the one executor and the one machine, so they
// must take turns exactly as two setup runs do.
func TestSetupSlot_DevnetWipeAndResetWaitForARunningSetup(t *testing.T) {
	f := newFleet()
	a := newAPITestServerWithExecutor(t, f.factory)
	addTarget(t, a)
	putConfig(t, a, svcDevnet, catalog.DevnetConfig{HTTPPort: 8600, WSPort: 8601})

	release := holdSetupSlot(t, a.srv, "local")
	before := len(f.commands(t, "local"))
	expectSlotRefusal(t, a.do(t, "POST", "/api/targets/local/containers/devnet/wipe", map[string]string{"Confirm": "devnet"}))
	expectSlotRefusal(t, a.do(t, "POST", "/api/targets/local/containers/devnet/reset", nil))
	if after := f.commands(t, "local"); len(after) != before {
		t.Errorf("a refused wipe or reset still ran commands on the machine: %q", after[before:])
	}

	release()
	expectStatus(t, a.do(t, "POST", "/api/targets/local/containers/devnet/wipe", map[string]string{"Confirm": "devnet"}), http.StatusOK)
	expectStatus(t, a.do(t, "POST", "/api/targets/local/containers/devnet/reset", nil), http.StatusOK)
}

// A gateway's work happens on its placement machine, so that is the slot its
// wipe has to respect — the same one handleGatewayProvision claims.
func TestSetupSlot_GatewayWipeWaitsForARunningSetupOnItsMachine(t *testing.T) {
	a := gatewayServer(t)
	addGateway(t, a, "default", "local", catalog.GatewayConfig{
		Port: 4100,
		Networks: []catalog.GatewayNetwork{{ChainID: 369, Upstreams: []catalog.GatewayUpstream{
			{ID: "public", Endpoint: "https://rpc.pulsechain.com"},
		}}},
	})

	release := holdSetupSlot(t, a.srv, "local")
	expectSlotRefusal(t, a.do(t, "POST", "/api/gateways/default/wipe", map[string]string{"Confirm": "default"}))

	release()
	res := a.do(t, "POST", "/api/gateways/default/wipe", map[string]string{"Confirm": "default"})
	defer res.Body.Close()
	if res.StatusCode == http.StatusConflict {
		body, _ := io.ReadAll(res.Body)
		t.Fatalf("still refused once the slot was free: %s", body)
	}
}

// The turn-taking runs both ways: a setup or provision that starts while a
// wipe is running has the same interleaving problem from the other side.
func TestSetupSlot_SetupWaitsForARunningWipe(t *testing.T) {
	s := New(Config{Token: NewSessionToken()})

	release, ok := s.claimTargetOp(httptest.NewRecorder(), "box")
	if !ok {
		t.Fatal("could not claim a fresh target for a wipe")
	}
	w := httptest.NewRecorder()
	if _, ok := s.claimSetupRun(w, "box"); ok {
		t.Fatal("a setup run was claimed while a wipe was running")
	}
	if w.Code != http.StatusConflict {
		t.Errorf("got %d, want 409", w.Code)
	}
	if _, ok := s.claimTargetOp(httptest.NewRecorder(), "box"); ok {
		t.Error("a second destructive operation was claimed while the first was running")
	}

	release()
	c, ok := s.claimSetupRun(httptest.NewRecorder(), "box")
	if !ok {
		t.Fatal("setup was still refused after the wipe released the target")
	}
	s.releaseSetupRun("box", c)
}

// Clear-and-resync deletes a node's data directory. Running it while the
// wizard is still installing that node would leave whichever finished last
// to decide what is on disk.
func TestSetupSlot_ServiceClearWaitsForARunningSetup(t *testing.T) {
	a := newAPITestServerWithExecutor(t, func(config.Target) (executor.Executor, error) {
		return &autoSucceedExecutor{}, nil
	})
	addAndWireLocalTarget(t, a)

	release := holdSetupSlot(t, a.srv, "local")
	expectSlotRefusal(t, a.do(t, "POST", "/api/targets/local/services/exec/clear", map[string]any{"Confirm": "exec"}))

	release()
	expectStatus(t, a.do(t, "POST", "/api/targets/local/services/exec/clear", map[string]any{"Confirm": "exec"}), http.StatusOK)
}
