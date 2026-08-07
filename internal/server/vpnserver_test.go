package server

import (
	"context"
	"io"
	"io/fs"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/valve-tech/valve-node-app/internal/config"
	"github.com/valve-tech/valve-node-app/internal/executor"
)

// wgHostFake is a stateful WireGuard host: it remembers the peers added via
// `wg set` and reflects them back in `wg show <iface> dump`, so the engine's
// verify-by-running (which checks the dump lists the peer's key) passes with the
// RANDOM key each enrollment generates — a static script cannot predict that.
type wgHostFake struct {
	mu         sync.Mutex
	peers      map[string]bool
	serverPub  string
	listenPort string
	calls      []string
}

func newWGHost() *wgHostFake {
	return &wgHostFake{
		peers:      map[string]bool{},
		serverPub:  "c2VydmVyUHVibGljS2V5c2VydmVyUHVibGljS2V5PQ==",
		listenPort: "51820",
	}
}

func (f *wgHostFake) Run(_ context.Context, cmd string, _ *executor.RunOpts) (executor.Result, error) {
	f.mu.Lock()
	f.calls = append(f.calls, cmd)
	f.mu.Unlock()
	switch {
	case strings.Contains(cmd, "id -u"):
		return executor.Result{Stdout: "0\n"}, nil
	case strings.Contains(cmd, "command -v wg"):
		return executor.Result{ExitCode: 0}, nil // wireguard-tools present
	case strings.Contains(cmd, "wg pubkey"):
		return executor.Result{Stdout: f.serverPub + "\n"}, nil
	case strings.Contains(cmd, "wg show") && strings.Contains(cmd, "listen-port"):
		return executor.Result{Stdout: f.listenPort + "\n"}, nil
	case strings.Contains(cmd, "wg show") && strings.Contains(cmd, "dump"):
		return executor.Result{Stdout: f.dump()}, nil
	case strings.Contains(cmd, "wg set") && strings.Contains(cmd, " remove"):
		f.setPeer(peerArg(cmd), false)
		return executor.Result{ExitCode: 0}, nil
	case strings.Contains(cmd, "wg set"):
		f.setPeer(peerArg(cmd), true)
		return executor.Result{ExitCode: 0}, nil
	default:
		return executor.Result{ExitCode: 0}, nil // wg genkey, wg-quick up/down/save, printf, etc.
	}
}

func (f *wgHostFake) dump() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	b := "PRIVKEY\t" + f.serverPub + "\t" + f.listenPort + "\toff\n"
	for pub := range f.peers {
		b += pub + "\t(none)\t203.0.113.7:51820\t10.9.0.2/32\t0\t0\t0\t0\n"
	}
	return b
}

func (f *wgHostFake) setPeer(pub string, present bool) {
	if pub == "" {
		return
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if present {
		f.peers[pub] = true
	} else {
		delete(f.peers, pub)
	}
}

func (f *wgHostFake) WriteFile(context.Context, string, []byte, fs.FileMode) error { return nil }
func (f *wgHostFake) ReadFile(context.Context, string) ([]byte, error)             { return nil, nil }
func (f *wgHostFake) Close() error                                                 { return nil }

// peerArg extracts the public key from `wg set '<iface>' peer '<pub>' ...`.
func peerArg(cmd string) string {
	const marker = "peer '"
	i := strings.Index(cmd, marker)
	if i < 0 {
		return ""
	}
	rest := cmd[i+len(marker):]
	j := strings.Index(rest, "'")
	if j < 0 {
		return ""
	}
	return rest[:j]
}

func newVPNServerTestServer(t *testing.T) (*apiTestServer, *wgHostFake) {
	t.Helper()
	host := newWGHost()
	a := newAPITestServerWithExecutor(t, func(config.Target) (executor.Executor, error) {
		return host, nil
	})
	return a, host
}

func provision(t *testing.T, a *apiTestServer, body map[string]any) vpnServerProvisionResponse {
	t.Helper()
	res := a.do(t, "POST", "/api/vpn-servers", body)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("provision: got %d, want 201", res.StatusCode)
	}
	return decodeJSON[vpnServerProvisionResponse](t, res)
}

func TestVPNServerProvision(t *testing.T) {
	a, _ := newVPNServerTestServer(t)
	resp := provision(t, a, map[string]any{"id": "home"})

	if resp.Server.ID != "home" || resp.Server.PublicKey == "" {
		t.Errorf("server = %+v, want id home + a public key", resp.Server)
	}
	if !strings.Contains(resp.FirewallHint, "51820/udp") {
		t.Errorf("FirewallHint = %q, want it to name 51820/udp", resp.FirewallHint)
	}
	// Local target, no endpoint host given → the app cannot know a reachable
	// endpoint, and says so rather than inventing one.
	if resp.EndpointConfigured {
		t.Errorf("EndpointConfigured = true for a local server with no endpoint host")
	}

	list := decodeJSON[[]vpnServerView](t, a.do(t, "GET", "/api/vpn-servers", nil))
	if len(list) != 1 || list[0].ID != "home" {
		t.Errorf("list = %+v, want the home server", list)
	}
}

func TestVPNServerProvisionWithEndpoint(t *testing.T) {
	a, _ := newVPNServerTestServer(t)
	resp := provision(t, a, map[string]any{"id": "home", "endpointHost": "vpn.example.com"})
	if !resp.EndpointConfigured || resp.Server.Endpoint != "vpn.example.com:51820" {
		t.Errorf("server endpoint = %q (configured=%v), want vpn.example.com:51820", resp.Server.Endpoint, resp.EndpointConfigured)
	}
}

func TestVPNServerProvisionFailsWhenInterfaceNeverComesUp(t *testing.T) {
	// A host where wg-quick "succeeds" but wg show never reports the interface.
	a := newAPITestServerWithExecutor(t, func(config.Target) (executor.Executor, error) {
		return (&scriptedExecutor{}).
			script("id -u", executor.Result{Stdout: "0\n"}).
			script("wg pubkey", executor.Result{Stdout: "c2VydmVyUHVibGljS2V5c2VydmVyUHVibGljS2V5PQ==\n"}).
			script("dump", executor.Result{ExitCode: 1, Stderr: "No such device"}), nil
	})
	res := a.do(t, "POST", "/api/vpn-servers", map[string]any{"id": "home"})
	res.Body.Close()
	if res.StatusCode != http.StatusBadGateway {
		t.Fatalf("provision over a broken host: got %d, want 502", res.StatusCode)
	}
}

func TestVPNServerEnrollReturnsClientConfigOnce(t *testing.T) {
	a, _ := newVPNServerTestServer(t)
	provision(t, a, map[string]any{"id": "home", "endpointHost": "vpn.example.com"})

	res := a.do(t, "POST", "/api/vpn-servers/home/peers", map[string]any{"name": "laptop"})
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("enroll: got %d, want 201", res.StatusCode)
	}
	enr := decodeJSON[vpnEnrollResponse](t, res)
	if enr.AllowedIP != "10.9.0.2/32" {
		t.Errorf("allocated IP = %q, want 10.9.0.2/32 (first after the server's .1)", enr.AllowedIP)
	}
	// The config is the one-time delivery: it MUST carry the device private key
	// and point at the server.
	for _, want := range []string{"[Interface]", "PrivateKey", "[Peer]", "vpn.example.com:51820"} {
		if !strings.Contains(enr.Config, want) {
			t.Errorf("client config missing %q:\n%s", want, enr.Config)
		}
	}

	// The stored record and every read-back must carry only the PUBLIC half.
	sv := decodeJSON[vpnServerView](t, a.do(t, "GET", "/api/vpn-servers/home", nil))
	if len(sv.Peers) != 1 || sv.Peers[0].Name != "laptop" || sv.Peers[0].PublicKey == "" {
		t.Fatalf("server peers = %+v, want one named laptop with a public key", sv.Peers)
	}
	body, _ := io.ReadAll(a.do(t, "GET", "/api/vpn-servers", nil).Body)
	if strings.Contains(string(body), "PrivateKey") || strings.Contains(string(body), enr.Config) {
		t.Errorf("the server list leaked a private config:\n%s", body)
	}
}

func TestVPNServerEnrollAllocatesSequentialIPs(t *testing.T) {
	a, _ := newVPNServerTestServer(t)
	provision(t, a, map[string]any{"id": "home", "endpointHost": "h.example.com"})

	first := decodeJSON[vpnEnrollResponse](t, a.do(t, "POST", "/api/vpn-servers/home/peers", map[string]any{"name": "laptop"}))
	second := decodeJSON[vpnEnrollResponse](t, a.do(t, "POST", "/api/vpn-servers/home/peers", map[string]any{"name": "phone"}))
	if first.AllowedIP != "10.9.0.2/32" || second.AllowedIP != "10.9.0.3/32" {
		t.Errorf("allocated %q then %q, want .2 then .3", first.AllowedIP, second.AllowedIP)
	}
}

func TestVPNServerEnrollNeedsEndpoint(t *testing.T) {
	a, _ := newVPNServerTestServer(t)
	provision(t, a, map[string]any{"id": "home"}) // local, no endpoint

	res := a.do(t, "POST", "/api/vpn-servers/home/peers", map[string]any{"name": "laptop"})
	res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("enroll without an endpoint: got %d, want 400", res.StatusCode)
	}
	// Providing one at enroll time works.
	ok := a.do(t, "POST", "/api/vpn-servers/home/peers", map[string]any{"name": "laptop", "endpointHost": "vpn.example.com"})
	if ok.StatusCode != http.StatusCreated {
		t.Fatalf("enroll with an endpoint override: got %d, want 201", ok.StatusCode)
	}
	ok.Body.Close()
}

func TestVPNServerRevoke(t *testing.T) {
	a, host := newVPNServerTestServer(t)
	provision(t, a, map[string]any{"id": "home", "endpointHost": "h.example.com"})
	enr := decodeJSON[vpnEnrollResponse](t, a.do(t, "POST", "/api/vpn-servers/home/peers", map[string]any{"name": "laptop"}))

	if len(host.peers) != 1 {
		t.Fatalf("host has %d peers after enroll, want 1", len(host.peers))
	}
	res := a.do(t, "POST", "/api/vpn-servers/home/peers/remove", map[string]any{"publicKey": enr.PublicKey})
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("revoke: got %d, want 204", res.StatusCode)
	}
	if len(host.peers) != 0 {
		t.Errorf("host still has the peer after revoke: %v", host.peers)
	}
	sv := decodeJSON[vpnServerView](t, a.do(t, "GET", "/api/vpn-servers/home", nil))
	if len(sv.Peers) != 0 {
		t.Errorf("record still lists the peer after revoke: %+v", sv.Peers)
	}
}

func TestVPNServerDelete(t *testing.T) {
	a, _ := newVPNServerTestServer(t)
	provision(t, a, map[string]any{"id": "home"})
	res := a.do(t, "DELETE", "/api/vpn-servers/home", nil)
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("delete: got %d, want 204", res.StatusCode)
	}
	list := decodeJSON[[]vpnServerView](t, a.do(t, "GET", "/api/vpn-servers", nil))
	if len(list) != 0 {
		t.Errorf("list after delete = %+v, want empty", list)
	}
}

func TestVPNServerRoutesRequireToken(t *testing.T) {
	a, _ := newVPNServerTestServer(t)
	routes := []struct{ method, path string }{
		{"GET", "/api/vpn-servers"},
		{"POST", "/api/vpn-servers"},
		{"GET", "/api/vpn-servers/x"},
		{"DELETE", "/api/vpn-servers/x"},
		{"GET", "/api/vpn-servers/x/status"},
		{"POST", "/api/vpn-servers/x/peers"},
		{"POST", "/api/vpn-servers/x/peers/remove"},
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
