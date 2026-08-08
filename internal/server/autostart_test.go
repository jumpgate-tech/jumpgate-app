package server

import (
	"context"
	"net/http"
	"testing"

	"github.com/valve-tech/valve-node-app/internal/config"
	"github.com/valve-tech/valve-node-app/internal/executor"
)

// markAutostart flips an existing overlay's autostart flag on via the save API.
func markAutostart(t *testing.T, a *apiTestServer, id string) {
	t.Helper()
	res := a.do(t, "POST", "/api/vpns", map[string]any{"id": id, "autostart": true})
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("mark %q autostart: got %d, want 200", id, res.StatusCode)
	}
}

// Autostart brings up ONLY the overlays marked for it, and leaves the rest
// alone — the operator starts those by hand.
func TestAutostartBringsUpOnlyMarkedOverlays(t *testing.T) {
	a, host := newVPNServerTestServer(t) // wgHostFake for every target
	createTestVPN(t, a, "auto-one")
	createTestVPN(t, a, "manual-two")
	markAutostart(t, a, "auto-one")

	results := a.srv.AutostartOverlays(context.Background())

	if len(results) != 1 {
		t.Fatalf("results = %+v, want exactly one (only auto-one is marked)", results)
	}
	if results[0].ID != "auto-one" || results[0].Err != nil {
		t.Fatalf("result = %+v, want auto-one up with no error", results[0])
	}
	// And it was really brought up on the host, not merely iterated.
	if !host.called("wg-quick up") {
		t.Errorf("autostart did not bring the interface up; calls=%v", host.calls)
	}
}

// A stored overlay with no autostart flag at all: nothing is attempted.
func TestAutostartNoneMarkedIsNoop(t *testing.T) {
	a, host := newVPNServerTestServer(t)
	createTestVPN(t, a, "manual-only")

	results := a.srv.AutostartOverlays(context.Background())

	if len(results) != 0 {
		t.Fatalf("results = %+v, want none attempted", results)
	}
	if host.called("wg-quick up") {
		t.Errorf("brought up an overlay that was not marked autostart; calls=%v", host.calls)
	}
}

// One overlay's bring-up failing must be REPORTED (with the error) and must not
// stop the others — autostart is best-effort and independent. Here the host
// exits 0 on wg-quick up but `wg show` never finds the interface, so Up's
// verification fails (the app's worst bugs report success over a broken state).
func TestAutostartReportsFailureWithoutBlocking(t *testing.T) {
	a := newAPITestServerWithExecutor(t, func(config.Target) (executor.Executor, error) {
		return (&scriptedExecutor{}).
			script("wg show", executor.Result{ExitCode: 1, Stderr: "Unable to access interface: No such device"}), nil
	})
	createTestVPN(t, a, "flaky")
	markAutostart(t, a, "flaky")

	results := a.srv.AutostartOverlays(context.Background())

	if len(results) != 1 || results[0].ID != "flaky" {
		t.Fatalf("results = %+v, want flaky reported", results)
	}
	if results[0].Err == nil {
		t.Fatal("Up verification failed on the host, but autostart reported success")
	}
}
