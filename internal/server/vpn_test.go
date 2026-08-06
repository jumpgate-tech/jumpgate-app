package server

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/valve-tech/valve-node-app/internal/config"
	"github.com/valve-tech/valve-node-app/internal/executor"
)

// A ProtonVPN-style `.conf` with a private key, for the save/redaction tests.
const testWGConf = `[Interface]
PrivateKey = QFhTdU5rZXlQcml2YXRlS2V5UHJpdmF0ZUtleVByaXY=
Address = 10.2.0.2/32
DNS = 10.2.0.1

[Peer]
PublicKey = eFRJQkE1cHVibGljS2V5cHVibGljS2V5cHVibGljS2U=
AllowedIPs = 0.0.0.0/0
Endpoint = 203.0.113.7:51820
`

// wg show dump: interface line + one peer that has handshaked.
const testWGDumpUp = "PRIVKEY\tPUBKEY\t51820\toff\n" +
	"peerpub\t(none)\t203.0.113.7:51820\t0.0.0.0/0\t1785000000\t128\t256\t25\n"

func createTestVPN(t *testing.T, a *apiTestServer, id string) vpnView {
	t.Helper()
	res := a.do(t, "POST", "/api/vpns", map[string]any{
		"id":       id,
		"provider": "proton",
		"config":   testWGConf,
	})
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("POST /api/vpns: got %d, want 201", res.StatusCode)
	}
	return decodeJSON[vpnView](t, res)
}

func TestVPNSaveCreatesAndDerivesDisplayFields(t *testing.T) {
	a := newAPITestServer(t)
	v := createTestVPN(t, a, "proton-us")

	if v.ID != "proton-us" || v.Provider != "proton" {
		t.Errorf("view = %+v, want id/provider proton-us/proton", v)
	}
	if v.Interface != defaultVPNInterface {
		t.Errorf("Interface = %q, want default %q", v.Interface, defaultVPNInterface)
	}
	if !v.Valid || !v.Configured {
		t.Errorf("view = %+v, want valid+configured", v)
	}
	if len(v.Endpoints) != 1 || v.Endpoints[0] != "203.0.113.7:51820" {
		t.Errorf("Endpoints = %v, want [203.0.113.7:51820]", v.Endpoints)
	}
	if len(v.Overlay) != 1 || v.Overlay[0] != "10.2.0.2/32" {
		t.Errorf("Overlay = %v, want [10.2.0.2/32]", v.Overlay)
	}
	if v.Peers != 1 {
		t.Errorf("Peers = %d, want 1", v.Peers)
	}
}

// The load-bearing security test: the API must NEVER echo the `.conf` back —
// it holds the interface private key. Not in a create response, not in the
// list, not anywhere.
func TestVPNResponsesNeverLeakTheKey(t *testing.T) {
	a := newAPITestServer(t)
	createTestVPN(t, a, "proton-us")

	for _, path := range []string{"/api/vpns", "/api/vpns"} {
		res := a.do(t, "GET", path, nil)
		body, _ := io.ReadAll(res.Body)
		res.Body.Close()
		s := string(body)
		if strings.Contains(s, "PrivateKey") || strings.Contains(s, "QFhTdU5rZXlQcml2YXRl") {
			t.Fatalf("GET %s leaked the WireGuard config/key:\n%s", path, s)
		}
	}
}

func TestVPNListReturnsEmptyArrayNotNull(t *testing.T) {
	a := newAPITestServer(t)
	res := a.do(t, "GET", "/api/vpns", nil)
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if strings.TrimSpace(string(body)) != "[]" {
		t.Errorf("empty list body = %q, want []", body)
	}
}

func TestVPNSaveRejectsUnusableConfig(t *testing.T) {
	a := newAPITestServer(t)
	res := a.do(t, "POST", "/api/vpns", map[string]any{
		"id":     "bad",
		"config": "[Interface]\nAddress = 10.0.0.1/32\n", // no private key, no peer
	})
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("POST bad config: got %d, want 400", res.StatusCode)
	}
}

func TestVPNSaveRequiresConfigToCreate(t *testing.T) {
	a := newAPITestServer(t)
	res := a.do(t, "POST", "/api/vpns", map[string]any{"id": "empty", "provider": "proton"})
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("create without config: got %d, want 400", res.StatusCode)
	}
}

func TestVPNSaveRejectsBadID(t *testing.T) {
	a := newAPITestServer(t)
	res := a.do(t, "POST", "/api/vpns", map[string]any{"id": "Bad ID!", "config": testWGConf})
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("bad id: got %d, want 400", res.StatusCode)
	}
}

func TestVPNSaveUnknownTargetRejected(t *testing.T) {
	a := newAPITestServer(t)
	res := a.do(t, "POST", "/api/vpns", map[string]any{
		"id":       "on-ghost",
		"config":   testWGConf,
		"targetId": "no-such-box",
	})
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("unknown target: got %d, want 400", res.StatusCode)
	}
}

// An update that omits config must not wipe the stored key — GET never returns
// it, so a client editing only metadata sends no config, and the overlay must
// stay valid (its config, and thus its key, still there).
func TestVPNUpdatePreservesStoredConfig(t *testing.T) {
	a := newAPITestServer(t)
	createTestVPN(t, a, "proton-us")

	res := a.do(t, "POST", "/api/vpns", map[string]any{
		"id":        "proton-us",
		"autostart": true,
	})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("update: got %d, want 200", res.StatusCode)
	}
	v := decodeJSON[vpnView](t, res)
	if !v.Autostart {
		t.Errorf("Autostart = false, want the update to have taken")
	}
	if !v.Valid || len(v.Endpoints) != 1 {
		t.Errorf("view after metadata-only update = %+v — the stored config was lost", v)
	}
}

func TestVPNDelete(t *testing.T) {
	a := newAPITestServer(t)
	createTestVPN(t, a, "proton-us")

	res := a.do(t, "DELETE", "/api/vpns/proton-us", nil)
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("DELETE: got %d, want 204", res.StatusCode)
	}

	list := decodeJSON[[]vpnView](t, a.do(t, "GET", "/api/vpns", nil))
	if len(list) != 0 {
		t.Errorf("after delete list = %+v, want empty", list)
	}
}

func TestVPNDeleteUnknownIs404(t *testing.T) {
	a := newAPITestServer(t)
	res := a.do(t, "DELETE", "/api/vpns/ghost", nil)
	res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("DELETE unknown: got %d, want 404", res.StatusCode)
	}
}

// Bring-up over a scripted executor: wg-quick up succeeds AND wg show reports
// the interface, so Up's verification passes and the tunnel is reported live.
func TestVPNUpReportsLiveTunnel(t *testing.T) {
	a := newAPITestServerWithExecutor(t, func(config.Target) (executor.Executor, error) {
		return (&scriptedExecutor{}).script("wg show", executor.Result{ExitCode: 0, Stdout: testWGDumpUp}), nil
	})
	createTestVPN(t, a, "proton-us")

	res := a.do(t, "POST", "/api/vpns/proton-us/up", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("up: got %d, want 200", res.StatusCode)
	}
	st := decodeJSON[vpnStatusView](t, res)
	if !st.Up || st.Peers != 1 || !st.Handshaked {
		t.Errorf("status = %+v, want up with 1 handshaked peer", st)
	}
}

// The load-bearing verify: wg-quick exits 0 but wg show cannot find the
// interface. Up must FAIL (502), not report success — this app's worst bugs
// report success over a broken state.
func TestVPNUpFailsWhenInterfaceAbsentDespiteExit0(t *testing.T) {
	a := newAPITestServerWithExecutor(t, func(config.Target) (executor.Executor, error) {
		return (&scriptedExecutor{}).
			script("wg show", executor.Result{ExitCode: 1, Stderr: "Unable to access interface: No such device"}), nil
	})
	createTestVPN(t, a, "proton-us")

	res := a.do(t, "POST", "/api/vpns/proton-us/up", nil)
	res.Body.Close()
	if res.StatusCode != http.StatusBadGateway {
		t.Fatalf("up over an interface that never came up: got %d, want 502", res.StatusCode)
	}
}

func TestVPNStatusReadsLiveInterface(t *testing.T) {
	a := newAPITestServerWithExecutor(t, func(config.Target) (executor.Executor, error) {
		return (&scriptedExecutor{}).script("wg show", executor.Result{ExitCode: 0, Stdout: testWGDumpUp}), nil
	})
	createTestVPN(t, a, "proton-us")

	st := decodeJSON[vpnStatusView](t, a.do(t, "GET", "/api/vpns/proton-us/status", nil))
	if !st.Up || st.Peers != 1 {
		t.Errorf("status = %+v, want up with 1 peer", st)
	}
	if st.Provider != "proton" {
		t.Errorf("Provider = %q, want proton", st.Provider)
	}
}

func TestVPNStatusDownWhenInterfaceAbsent(t *testing.T) {
	a := newAPITestServerWithExecutor(t, func(config.Target) (executor.Executor, error) {
		return (&scriptedExecutor{}).script("wg show", executor.Result{ExitCode: 1}), nil
	})
	createTestVPN(t, a, "proton-us")

	st := decodeJSON[vpnStatusView](t, a.do(t, "GET", "/api/vpns/proton-us/status", nil))
	if st.Up {
		t.Errorf("status = %+v, want Up=false for an absent interface", st)
	}
}

func TestVPNDownIsIdempotent(t *testing.T) {
	a := newAPITestServerWithExecutor(t, func(config.Target) (executor.Executor, error) {
		return (&scriptedExecutor{}).
			script("wg-quick down", executor.Result{ExitCode: 1, Stderr: "`jumpgate0' is not a WireGuard interface"}), nil
	})
	createTestVPN(t, a, "proton-us")

	res := a.do(t, "POST", "/api/vpns/proton-us/down", nil)
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("down a tunnel that is not up: got %d, want 204", res.StatusCode)
	}
}

func TestVPNStatusUnknownIs404(t *testing.T) {
	a := newAPITestServer(t)
	res := a.do(t, "GET", "/api/vpns/ghost/status", nil)
	res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("status unknown: got %d, want 404", res.StatusCode)
	}
}

func TestVPNRoutesRequireToken(t *testing.T) {
	a := newAPITestServer(t)
	routes := []struct{ method, path string }{
		{"GET", "/api/vpns"},
		{"POST", "/api/vpns"},
		{"DELETE", "/api/vpns/x"},
		{"GET", "/api/vpns/x/status"},
		{"POST", "/api/vpns/x/up"},
		{"POST", "/api/vpns/x/down"},
	}
	for _, rt := range routes {
		t.Run(rt.method+" "+rt.path, func(t *testing.T) {
			res := a.doNoAuth(t, rt.method, rt.path)
			res.Body.Close()
			if res.StatusCode != http.StatusUnauthorized {
				t.Fatalf("%s %s without token: got %d, want 401", rt.method, rt.path, res.StatusCode)
			}
		})
	}
}
