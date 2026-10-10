package server

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/executor"
)

// faultyWGHost is a wgHostFake that can stop answering (every command fails,
// as an unreachable box does) or keep a peer it was told to remove (a
// `wg set … remove` that did not take).
type faultyWGHost struct {
	*wgHostFake
	mu          sync.Mutex
	down, stuck bool
}

func (f *faultyWGHost) set(down, stuck bool) {
	f.mu.Lock()
	f.down, f.stuck = down, stuck
	f.mu.Unlock()
}

func (f *faultyWGHost) Run(ctx context.Context, cmd string, o *executor.RunOpts) (executor.Result, error) {
	f.mu.Lock()
	down, stuck := f.down, f.stuck
	f.mu.Unlock()
	switch {
	case down:
		return executor.Result{}, errors.New("ssh: connect: no route to host")
	case stuck && strings.Contains(cmd, "wg set") && strings.Contains(cmd, " remove"):
		return executor.Result{}, nil
	}
	return f.wgHostFake.Run(ctx, cmd, o)
}

func newFaultyVPNServer(t *testing.T) (*apiTestServer, *faultyWGHost) {
	t.Helper()
	host := &faultyWGHost{wgHostFake: newWGHost()}
	a := newAPITestServerWithExecutor(t, func(config.Target) (executor.Executor, error) { return host, nil })
	return a, host
}

func (a *apiTestServer) status(t *testing.T, method, path string, body any) int {
	t.Helper()
	res := a.do(t, method, path, body)
	res.Body.Close()
	return res.StatusCode
}

// A revoke the host did not carry out is a 502, and the device stays in the
// record: showing it revoked while it can still connect would be a lie about
// who can reach the network.
func TestVPNServerRevokeThatDidNotTakeKeepsTheDevice(t *testing.T) {
	a, host := newFaultyVPNServer(t)
	provision(t, a, map[string]any{"id": "home", "endpointHost": "h.example.com"})
	enr := decodeJSON[vpnEnrollResponse](t, a.do(t, "POST", "/api/vpn-servers/home/peers", map[string]any{"name": "laptop"}))
	host.set(false, true)
	if code := a.status(t, "POST", "/api/vpn-servers/home/peers/remove", map[string]any{"publicKey": enr.PublicKey}); code != http.StatusBadGateway {
		t.Fatalf("revoke that did not take: %d, want 502", code)
	}
	sv := decodeJSON[vpnServerView](t, a.do(t, "GET", "/api/vpn-servers/home", nil))
	if len(sv.Peers) != 1 || sv.Peers[0].PublicKey != enr.PublicKey {
		t.Fatalf("the record dropped a device the host still lets in: %+v", sv.Peers)
	}
}

// A revoke must name the key; without one nothing reaches the host.
func TestVPNServerRevokeNeedsAKey(t *testing.T) {
	a, host := newFaultyVPNServer(t)
	provision(t, a, map[string]any{"id": "home"})
	for _, body := range []any{map[string]any{"publicKey": "  "}, "not an object"} {
		if code := a.status(t, "POST", "/api/vpn-servers/home/peers/remove", body); code != http.StatusBadRequest {
			t.Errorf("revoke %v: %d, want 400", body, code)
		}
	}
	if host.called(" remove") {
		t.Fatal("a refused revoke reached the host")
	}
}

// A wipe whose host teardown fails keeps the record (the server may still be
// up), and ?force=true forgets it anyway when the host is gone for good.
func TestVPNServerWipeKeepsTheRecordUnlessForced(t *testing.T) {
	a, host := newFaultyVPNServer(t)
	provision(t, a, map[string]any{"id": "home"})
	host.set(true, false)
	if code := a.status(t, "DELETE", "/api/vpn-servers/home", nil); code != http.StatusBadGateway {
		t.Fatalf("wipe of an unreachable host: %d, want 502", code)
	}
	if n := len(decodeJSON[[]vpnServerView](t, a.do(t, "GET", "/api/vpn-servers", nil))); n != 1 {
		t.Fatalf("the record was forgotten although the host may still run it (%d records)", n)
	}
	if code := a.status(t, "DELETE", "/api/vpn-servers/home?force=true", nil); code != http.StatusNoContent {
		t.Fatalf("forced wipe: %d, want 204", code)
	}
	if n := len(decodeJSON[[]vpnServerView](t, a.do(t, "GET", "/api/vpn-servers", nil))); n != 0 {
		t.Fatalf("a forced wipe kept the record (%d records)", n)
	}
}

// A server whose machine is no longer registered cannot be torn down; the
// record is kept unless forced.
func TestVPNServerWipeOfAnUnregisteredMachine(t *testing.T) {
	a, _ := newFaultyVPNServer(t)
	provision(t, a, map[string]any{"id": "home"})
	if _, err := a.srv.updateConfig(func(c *config.Config) error {
		c.VPNServers[0].TargetID = "gone"
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	res := a.do(t, "DELETE", "/api/vpn-servers/home", nil)
	e := decodeJSON[map[string]any](t, res)
	if res.StatusCode != http.StatusBadGateway || !strings.Contains(e["error"].(string), `"gone" is not registered`) {
		t.Fatalf("wipe: %d %v", res.StatusCode, e)
	}
	if code := a.status(t, "DELETE", "/api/vpn-servers/home?force=true", nil); code != http.StatusNoContent {
		t.Fatalf("forced wipe: %d", code)
	}
}

// status reports the live interface and its peers; an unreachable host is a
// 502, not a "down" reading.
func TestVPNServerStatusReadsTheHost(t *testing.T) {
	a, host := newFaultyVPNServer(t)
	provision(t, a, map[string]any{"id": "home", "endpointHost": "h.example.com"})
	a.do(t, "POST", "/api/vpn-servers/home/peers", map[string]any{"name": "laptop"}).Body.Close()
	st := decodeJSON[vpnStatusView](t, a.do(t, "GET", "/api/vpn-servers/home/status", nil))
	if st.ID != "home" || !st.Up || st.Peers != 1 {
		t.Fatalf("status %+v", st)
	}
	host.set(true, false)
	for _, rt := range []struct{ method, path string }{
		{"GET", "/api/vpn-servers/home/status"},
		{"POST", "/api/vpn-servers/home/down"},
		{"POST", "/api/vpn-servers/home/up"},
	} {
		if code := a.status(t, rt.method, rt.path, nil); code != http.StatusBadGateway {
			t.Errorf("%s %s on an unreachable host: %d, want 502", rt.method, rt.path, code)
		}
	}
}

// Every per-server route answers 404 for a server that was never provisioned,
// before any host is touched.
func TestVPNServerRoutesRefuseAnUnknownServer(t *testing.T) {
	a, host := newFaultyVPNServer(t)
	for _, rt := range []struct{ method, path string }{
		{"GET", "/api/vpn-servers/nope"},
		{"DELETE", "/api/vpn-servers/nope"},
		{"GET", "/api/vpn-servers/nope/status"},
		{"POST", "/api/vpn-servers/nope/down"},
		{"POST", "/api/vpn-servers/nope/up"},
		{"POST", "/api/vpn-servers/nope/peers"},
		{"POST", "/api/vpn-servers/nope/peers/remove"},
	} {
		if code := a.status(t, rt.method, rt.path, map[string]any{"name": "x", "publicKey": "k"}); code != http.StatusNotFound {
			t.Errorf("%s %s: %d, want 404", rt.method, rt.path, code)
		}
	}
	if len(host.calls) != 0 {
		t.Fatalf("an unknown server reached the host: %v", host.calls)
	}
}

// A full subnet is a conflict, and nothing reaches the host.
func TestVPNServerEnrollIntoAFullSubnet(t *testing.T) {
	a, host := newFaultyVPNServer(t)
	provision(t, a, map[string]any{"id": "home", "endpointHost": "h.example.com", "address": "10.9.0.1/30"})
	if code := a.status(t, "POST", "/api/vpn-servers/home/peers", map[string]any{"name": "one"}); code != http.StatusCreated {
		t.Fatalf("first device: %d", code)
	}
	host.mu.Lock()
	before := len(host.calls)
	host.mu.Unlock()
	if code := a.status(t, "POST", "/api/vpn-servers/home/peers", map[string]any{"name": "two"}); code != http.StatusConflict {
		t.Fatalf("enroll into a full /30: %d, want 409", code)
	}
	host.mu.Lock()
	after := len(host.calls)
	host.mu.Unlock()
	if after != before {
		t.Fatalf("a refused enroll reached the host: %v", host.calls[before:])
	}
}

// What a device routes through the tunnel: the server's subnet by default,
// everything only when asked for a full tunnel, or exactly the list given.
func TestVPNServerEnrollRoutesOnlyWhatWasAskedFor(t *testing.T) {
	a, _ := newFaultyVPNServer(t)
	provision(t, a, map[string]any{"id": "home", "endpointHost": "h.example.com"})
	for name, c := range map[string]struct {
		body map[string]any
		want string
	}{
		"default":     {map[string]any{"name": "a"}, "AllowedIPs = 10.9.0.0/24"},
		"full tunnel": {map[string]any{"name": "b", "fullTunnel": true}, "AllowedIPs = 0.0.0.0/0"},
		"explicit":    {map[string]any{"name": "c", "fullTunnel": true, "allowedIps": []string{"192.168.50.0/24"}}, "AllowedIPs = 192.168.50.0/24"},
	} {
		enr := decodeJSON[vpnEnrollResponse](t, a.do(t, "POST", "/api/vpn-servers/home/peers", c.body))
		if !strings.Contains(enr.Config, c.want) {
			t.Errorf("%s: config lacks %q:\n%s", name, c.want, enr.Config)
		}
	}
	for _, body := range []any{map[string]any{"name": "  "}, "not an object"} {
		if code := a.status(t, "POST", "/api/vpn-servers/home/peers", body); code != http.StatusBadRequest {
			t.Errorf("enroll %v: %d, want 400", body, code)
		}
	}
}
