package relay

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
	"time"
)

// The beacon route for a chain is a POOL, not one upstream: every target running
// that chain with a beacon endpoint. The pool reads each member's own
// /eth/v1/node/health, which answers 200 healthy, 206 syncing, 503 down.

// beaconNode is a real HTTP server that reports whatever health the test sets.
type beaconNode struct {
	srv    *httptest.Server
	url    *url.URL
	status atomic.Int32
	hits   atomic.Int64
}

func newBeaconNode(t *testing.T, status int) *beaconNode {
	t.Helper()
	n := &beaconNode{}
	n.status.Store(int32(status))
	n.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/eth/v1/node/health" {
			w.WriteHeader(int(n.status.Load()))
			return
		}
		n.hits.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(n.srv.Close)
	u, err := url.Parse(n.srv.URL)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	n.url = u
	return n
}

func poolWith(t *testing.T, chainID int, nodes ...*beaconNode) *BeaconPool {
	t.Helper()
	urls := make([]*url.URL, 0, len(nodes))
	for _, n := range nodes {
		urls = append(urls, n.url)
	}
	p := NewBeaconPool(map[int][]*url.URL{chainID: urls})
	t.Cleanup(p.Stop)
	return p
}

// A chain with no beacon endpoint reports false, which the handler turns into a
// definite 501 rather than a dead 502.
func TestBeaconPoolReportsNoUpstreamForAnUnknownChain(t *testing.T) {
	p := NewBeaconPool(nil)
	t.Cleanup(p.Stop)

	if _, ok := p.Next(369); ok {
		t.Error("Next reported an upstream for a chain with no beacon")
	}
}

// Traffic spreads across the pool rather than piling onto one node.
func TestBeaconPoolRoundRobins(t *testing.T) {
	a := newBeaconNode(t, http.StatusOK)
	b := newBeaconNode(t, http.StatusOK)
	p := poolWith(t, 369, a, b)
	p.Probe(context.Background())

	seen := map[string]int{}
	for i := 0; i < 6; i++ {
		u, ok := p.Next(369)
		if !ok {
			t.Fatalf("call %d: no upstream", i)
		}
		seen[u.String()]++
	}
	if len(seen) != 2 {
		t.Fatalf("traffic reached %d nodes, want 2: %v", len(seen), seen)
	}
	for u, n := range seen {
		if n != 3 {
			t.Errorf("%s took %d of 6 calls, want an even 3", u, n)
		}
	}
}

// A node answering 503 is out of rotation. Sending a customer to a node that
// says it is down would be a 502 the pool could have avoided.
func TestBeaconPoolDropsADownNode(t *testing.T) {
	up := newBeaconNode(t, http.StatusOK)
	down := newBeaconNode(t, http.StatusServiceUnavailable)
	p := poolWith(t, 369, up, down)
	p.Probe(context.Background())

	for i := 0; i < 6; i++ {
		u, ok := p.Next(369)
		if !ok {
			t.Fatalf("call %d: no upstream", i)
		}
		if u.String() == down.url.String() {
			t.Fatal("a node reporting 503 is still taking traffic")
		}
	}
}

// 206 means syncing. It is degraded, not dead, and it stays in rotation —
// dropping every syncing node would empty the pool exactly when a chain is
// catching up and the operator most needs it to answer.
func TestBeaconPoolKeepsASyncingNode(t *testing.T) {
	syncing := newBeaconNode(t, http.StatusPartialContent)
	p := poolWith(t, 369, syncing)
	p.Probe(context.Background())

	if _, ok := p.Next(369); !ok {
		t.Fatal("a syncing node was dropped from rotation")
	}
}

// Every node down means no upstream, which is a 501 rather than a proxy attempt
// that is certain to fail.
func TestBeaconPoolReportsNothingWhenEveryNodeIsDown(t *testing.T) {
	a := newBeaconNode(t, http.StatusServiceUnavailable)
	b := newBeaconNode(t, http.StatusServiceUnavailable)
	p := poolWith(t, 369, a, b)
	p.Probe(context.Background())

	if _, ok := p.Next(369); ok {
		t.Error("Next reported an upstream with the whole pool down")
	}
}

// A node that recovers comes back. A pool that never re-probed would shrink to
// nothing over a long uptime and never grow again.
func TestBeaconPoolRestoresARecoveredNode(t *testing.T) {
	n := newBeaconNode(t, http.StatusServiceUnavailable)
	p := poolWith(t, 369, n)
	p.Probe(context.Background())

	if _, ok := p.Next(369); ok {
		t.Fatal("a down node is in rotation")
	}

	n.status.Store(int32(http.StatusOK))
	p.Probe(context.Background())

	if _, ok := p.Next(369); !ok {
		t.Fatal("a recovered node did not return to rotation")
	}
}

// Before the first probe the pool must still serve. A relay that answered 501
// for the first probe interval after startup would look broken on every deploy.
func TestBeaconPoolServesBeforeItHasProbed(t *testing.T) {
	n := newBeaconNode(t, http.StatusOK)
	p := poolWith(t, 369, n)

	if _, ok := p.Next(369); !ok {
		t.Fatal("the pool refused traffic before its first probe")
	}
}

// Health is reported per chain, so the rollup can say WHY a chain is degraded.
func TestBeaconPoolReportsItsHealth(t *testing.T) {
	up := newBeaconNode(t, http.StatusOK)
	syncing := newBeaconNode(t, http.StatusPartialContent)
	down := newBeaconNode(t, http.StatusServiceUnavailable)
	p := poolWith(t, 369, up, syncing, down)
	p.Probe(context.Background())

	h := p.Health(369)
	if h.Total != 3 {
		t.Errorf("Total = %d, want 3", h.Total)
	}
	if h.Usable != 2 {
		t.Errorf("Usable = %d, want 2 (healthy plus syncing)", h.Usable)
	}
	if h.Syncing != 1 {
		t.Errorf("Syncing = %d, want 1", h.Syncing)
	}
	if !h.OK {
		t.Error("OK = false with two usable nodes")
	}
}

// A probe must not hang the relay. A beacon node that accepts a connection and
// never answers would otherwise stall the probe loop forever.
func TestBeaconPoolProbeIsBounded(t *testing.T) {
	// The stall only needs to outlast the probe timeout. A much longer sleep
	// would also make httptest.Close block on the handler at teardown, so the
	// test would cost its own sleep on every run.
	stall := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(beaconProbeTimeout + time.Second)
	}))
	t.Cleanup(stall.Close)
	u, _ := url.Parse(stall.URL)

	p := NewBeaconPool(map[int][]*url.URL{369: {u}})
	t.Cleanup(p.Stop)

	done := make(chan struct{})
	go func() { p.Probe(context.Background()); close(done) }()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("Probe did not return — a stalled beacon node blocks the probe loop")
	}
}
