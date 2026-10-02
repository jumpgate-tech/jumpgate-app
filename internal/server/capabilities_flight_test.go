package server

import (
	"context"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/valve-tech/valve-node-app/internal/catalog"
	"github.com/valve-tech/valve-node-app/internal/config"
)

// capGateway is the smallest gateway the capabilities route will probe. The
// probe itself is replaced in these tests, so the upstream is never dialed.
func capGateway(t *testing.T) *apiTestServer {
	t.Helper()
	a := gatewayServer(t)
	addGateway(t, a, "default", "local", catalog.GatewayConfig{
		Port: 4100,
		Networks: []catalog.GatewayNetwork{{ChainID: 369, Upstreams: []catalog.GatewayUpstream{
			{ID: "ext", Kind: catalog.UpstreamExternal, Endpoint: "https://rpc.invalid"},
		}}},
	})
	return a
}

// A probe cut short by its deadline reports every upstream as unreachable.
// Caching that would show the operator a table of dead upstreams for ten
// minutes, when the only thing that failed was the clock.
func TestCapabilities_TimedOutProbeIsNotCached(t *testing.T) {
	a := capGateway(t)
	var calls atomic.Int32
	a.srv.capTimeout = 10 * time.Millisecond
	a.srv.capProbe = func(ctx context.Context, _ config.Config, _ config.Gateway) capabilitiesResponse {
		calls.Add(1)
		<-ctx.Done()
		return capabilitiesResponse{At: time.Now()}
	}

	for i := 0; i < 2; i++ {
		res := a.do(t, "GET", "/api/gateways/default/capabilities", nil)
		res.Body.Close()
		if res.StatusCode != http.StatusOK {
			t.Fatalf("request %d: got %d, want 200", i, res.StatusCode)
		}
	}
	if _, hit := a.srv.cachedCapabilities("default"); hit {
		t.Error("a probe that ran out of time was cached")
	}
	if got := calls.Load(); got != 2 {
		t.Errorf("got %d probes, want 2: the second request must not be served the timed-out result", got)
	}
}

// Every screen load that misses the cache used to start its own probe, so a
// few tabs opened together multiplied the load against every upstream. Callers
// that arrive while a probe for the same gateway is running must wait for it
// and share its answer.
func TestCapabilities_ConcurrentRequestsShareOneProbe(t *testing.T) {
	a := capGateway(t)
	var calls atomic.Int32
	started := make(chan struct{})
	release := make(chan struct{})
	a.srv.capProbe = func(ctx context.Context, _ config.Config, _ config.Gateway) capabilitiesResponse {
		if calls.Add(1) == 1 {
			close(started)
		}
		<-release
		return capabilitiesResponse{At: time.Now()}
	}

	const n = 3
	results := make(chan capabilitiesResponse, n)
	get := func() {
		results <- decode[capabilitiesResponse](t, a.do(t, "GET", "/api/gateways/default/capabilities", nil))
	}
	go get()
	<-started
	for i := 1; i < n; i++ {
		go get()
	}

	// Wait until the later callers are parked on the running probe, or have
	// started probes of their own, which is the bug.
	deadline := time.Now().Add(5 * time.Second)
	for a.srv.capWaiters("default") < n-1 && calls.Load() == 1 {
		if time.Now().After(deadline) {
			t.Fatal("later callers neither joined the running probe nor started their own")
		}
		time.Sleep(time.Millisecond)
	}
	close(release)

	var first time.Time
	for i := 0; i < n; i++ {
		res := <-results
		if i == 0 {
			first = res.At
		} else if !res.At.Equal(first) {
			t.Errorf("caller %d got a different probe (%v) from the first (%v)", i, res.At, first)
		}
	}
	if got := calls.Load(); got != 1 {
		t.Errorf("got %d probes for %d concurrent requests, want 1", got, n)
	}
}
