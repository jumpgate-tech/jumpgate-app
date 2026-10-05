package server

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/executor"
)

// gatedWGHost holds every `wg set … allowed-ips` (an AddPeer) until n of them
// are in flight at once, so concurrent enrolls are guaranteed to overlap on the
// host. It gives up waiting after a while so a serialised implementation still
// finishes.
type gatedWGHost struct {
	*wgHostFake
	n       int
	mu      sync.Mutex
	arrived int
	release chan struct{}
	fail    bool // fail every AddPeer instead
}

func (g *gatedWGHost) Run(ctx context.Context, cmd string, o *executor.RunOpts) (executor.Result, error) {
	if strings.Contains(cmd, "wg set") && !strings.Contains(cmd, " remove") {
		if g.fail {
			return executor.Result{ExitCode: 1, Stderr: "wg: no such device"}, nil
		}
		g.mu.Lock()
		g.arrived++
		if g.arrived == g.n {
			close(g.release)
		}
		g.mu.Unlock()
		select {
		case <-g.release:
		case <-time.After(2 * time.Second):
		}
	}
	return g.wgHostFake.Run(ctx, cmd, o)
}

func newGatedVPNServer(t *testing.T, n int) (*apiTestServer, *gatedWGHost) {
	t.Helper()
	host := &gatedWGHost{wgHostFake: newWGHost(), n: n, release: make(chan struct{})}
	a := newAPITestServerWithExecutor(t, func(config.Target) (executor.Executor, error) { return host, nil })
	return a, host
}

// D4: two devices enrolled at the same moment must get different overlay
// addresses. Allocating from a snapshot taken outside the config lock gave them
// the same one, and WireGuard then silently moved it to the second device.
func TestVPNServerConcurrentEnrollsGetDistinctIPs(t *testing.T) {
	const n = 4
	a, _ := newGatedVPNServer(t, n)
	provision(t, a, map[string]any{"id": "home", "endpointHost": "vpn.example.com"})

	var wg sync.WaitGroup
	ips := make([]string, n)
	codes := make([]int, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			res := a.do(t, "POST", "/api/vpn-servers/home/peers", map[string]any{"name": fmt.Sprintf("dev%d", i)})
			codes[i] = res.StatusCode
			if res.StatusCode == http.StatusCreated {
				ips[i] = decodeJSON[vpnEnrollResponse](t, res).AllowedIP
			} else {
				res.Body.Close()
			}
		}(i)
	}
	wg.Wait()

	seen := map[string]int{}
	for i, ip := range ips {
		if codes[i] != http.StatusCreated {
			t.Fatalf("enroll %d: status %d", i, codes[i])
		}
		if j, dup := seen[ip]; dup {
			t.Fatalf("enrolls %d and %d both got %s", j, i, ip)
		}
		seen[ip] = i
	}
	c, _ := config.Load()
	sv, _ := c.FindVPNServer("home")
	if len(sv.Peers) != n {
		t.Fatalf("recorded %d peers, want %d", len(sv.Peers), n)
	}
}

// The address is reserved before the host call; when the host refuses the
// peer, the reservation is rolled back so the address is free again.
func TestVPNServerEnrollRollsBackTheReservationOnHostFailure(t *testing.T) {
	a, host := newGatedVPNServer(t, 1)
	provision(t, a, map[string]any{"id": "home", "endpointHost": "vpn.example.com"})
	host.fail = true
	res := a.do(t, "POST", "/api/vpn-servers/home/peers", map[string]any{"name": "laptop"})
	res.Body.Close()
	if res.StatusCode != http.StatusBadGateway {
		t.Fatalf("status %d, want 502", res.StatusCode)
	}
	c, _ := config.Load()
	if sv, _ := c.FindVPNServer("home"); len(sv.Peers) != 0 {
		t.Fatalf("a failed enroll left a peer record: %+v", sv.Peers)
	}
	host.fail = false
	enr := decodeJSON[vpnEnrollResponse](t, a.do(t, "POST", "/api/vpn-servers/home/peers", map[string]any{"name": "laptop"}))
	if enr.AllowedIP != "10.9.0.2/32" {
		t.Fatalf("after rollback the first address is %s, want 10.9.0.2/32", enr.AllowedIP)
	}
}

// D4: a re-provision that moves a server with peers (another machine, another
// interface, another subnet) would strand every device config already handed
// out while the UI still listed them. It is refused with 409 before the host is
// touched.
func TestVPNServerReprovisionWithPeersRefusesAMove(t *testing.T) {
	cases := []struct {
		name string
		body map[string]any
	}{
		{"target", map[string]any{"id": "home", "targetId": "boxa"}},
		{"interface", map[string]any{"id": "home", "interface": "jumpgate1"}},
		{"subnet", map[string]any{"id": "home", "address": "10.20.0.1/24"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a, host, _ := newRemoteVPNServerTestServer(t)
			provision(t, a, map[string]any{"id": "home", "endpointHost": "vpn.example.com"})
			if res := a.do(t, "POST", "/api/vpn-servers/home/peers", map[string]any{"name": "laptop"}); res.StatusCode != http.StatusCreated {
				t.Fatalf("enroll: %d", res.StatusCode)
			}
			host.mu.Lock()
			before := len(host.calls)
			host.mu.Unlock()

			res := a.do(t, "POST", "/api/vpn-servers", tc.body)
			body, _ := io.ReadAll(res.Body)
			res.Body.Close()
			if res.StatusCode != http.StatusConflict {
				t.Fatalf("got %d, want 409: %s", res.StatusCode, body)
			}
			if !strings.Contains(string(body), "wipe or migrate") {
				t.Errorf("refusal does not say what to do: %s", body)
			}
			host.mu.Lock()
			after := len(host.calls)
			host.mu.Unlock()
			if after != before {
				t.Errorf("ran %d host commands before refusing", after-before)
			}
			c, _ := config.Load()
			sv, _ := c.FindVPNServer("home")
			if sv.TargetID != "" || sv.Interface != "jumpgate0" || sv.Address != "10.9.0.1/24" || len(sv.Peers) != 1 {
				t.Errorf("the record changed: %+v", sv)
			}
		})
	}
}

// Re-provisioning in place (same machine, interface and subnet) still works
// with peers, and so does a move with no peers to strand.
func TestVPNServerReprovisionInPlaceWithPeersIsFine(t *testing.T) {
	a, _, _ := newRemoteVPNServerTestServer(t)
	provision(t, a, map[string]any{"id": "home", "endpointHost": "vpn.example.com"})
	provision(t, a, map[string]any{"id": "work", "interface": "jumpgate1"})
	if res := a.do(t, "POST", "/api/vpn-servers/home/peers", map[string]any{"name": "laptop"}); res.StatusCode != http.StatusCreated {
		t.Fatalf("enroll: %d", res.StatusCode)
	}
	res := a.do(t, "POST", "/api/vpn-servers", map[string]any{"id": "home", "address": "10.9.0.1/24", "listenPort": 51820})
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("in-place re-provision: %d", res.StatusCode)
	}
	res = a.do(t, "POST", "/api/vpn-servers", map[string]any{"id": "work", "targetId": "boxa"})
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("moving a server with no peers: %d", res.StatusCode)
	}
}
