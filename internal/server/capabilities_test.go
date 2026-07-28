package server

// Every test in this file is in-process: any RPC endpoint a test needs is an
// httptest.Server this test started itself (or, for the "must never be
// dialed" cases, a Transport that fails the test if it is ever asked to dial
// anything at all). Nothing here opens a socket to the real internet — see
// the internal/capabilities package's own tests for the same discipline and
// the same reason: a probe test that can reach the real internet is a test
// that can flake on someone else's outage.

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/valve-tech/valve-node-app/internal/capabilities"
	"github.com/valve-tech/valve-node-app/internal/catalog"
	"github.com/valve-tech/valve-node-app/internal/config"
	"github.com/valve-tech/valve-node-app/internal/executor"
	"github.com/valve-tech/valve-node-app/internal/setup"
)

// ---------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------

// chainIDCounter counts how many times a capCountingServer answered
// eth_chainId — exactly once per capabilities.Probe run — which is what lets
// a test assert on ProbeRepeat's repeat count directly instead of guessing
// from wall-clock timing.
type chainIDCounter struct {
	mu  sync.Mutex
	hit int
}

func (c *chainIDCounter) inc() {
	c.mu.Lock()
	c.hit++
	c.mu.Unlock()
}

func (c *chainIDCounter) calls() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.hit
}

// capCountingServer answers JSON-RPC (single calls and the 2-item batch
// probe sends) for chainID, counting eth_chainId calls on counter. Every
// other method answers -32601, which is a normal, fast "unsupported" verdict
// rather than a hang.
func capCountingServer(t *testing.T, counter *chainIDCounter, chainID int) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "read", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		trimmed := strings.TrimSpace(string(raw))
		if strings.HasPrefix(trimmed, "[") {
			// The batch probe: answer both items so it does not stall.
			io.WriteString(w, `[{"jsonrpc":"2.0","id":1,"result":"0x1"},{"jsonrpc":"2.0","id":2,"result":"0x1"}]`)
			return
		}
		var req struct {
			Method string `json:"method"`
		}
		_ = json.Unmarshal([]byte(trimmed), &req)
		if req.Method == "eth_chainId" {
			counter.inc()
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":1,"result":%q}`, fmt.Sprintf("0x%x", chainID))
			return
		}
		io.WriteString(w, `{"jsonrpc":"2.0","id":1,"error":{"code":-32601,"message":"method not found"}}`)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// downServer answers every request with a 500, which capabilities.Prober
// treats as "no well-formed reply" (deliberately — a 5xx is not evidence
// about the node behind it) and therefore unreachable. It is a faster and
// more deterministic way to get "genuinely unreachable" than racing a closed
// listener's connection-refused behaviour.
func downServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "down", http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// fastTestProber is a Prober tuned for these tests: short timeouts (nothing
// here should ever legitimately take long) and WebSocket probing off, since
// none of these tests are about the WS transport and it would otherwise add
// a dozen more sockets per endpoint for no assertion made on them.
func fastTestProber() *capabilities.Prober {
	p := capabilities.NewProber()
	p.ProbeTimeout = 3 * time.Second
	p.ProbeWS = false
	return p
}

// hostPort splits an httptest.Server's URL into a dialable host and port,
// for building a config.Target that points a managed upstream at it.
func hostPort(t *testing.T, rawURL string) (string, int) {
	t.Helper()
	u, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("parse %q: %v", rawURL, err)
	}
	port, err := strconv.Atoi(u.Port())
	if err != nil {
		t.Fatalf("port of %q: %v", rawURL, err)
	}
	return u.Hostname(), port
}

// poisonedTransport fails the test the moment anything tries to dial through
// it, for asserting that an "unprobeable" endpoint is never even attempted.
type poisonedTransport struct{ t *testing.T }

func (p poisonedTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	p.t.Fatalf("HTTP request made to %s — an unprobeable endpoint must never be dialed", r.URL)
	return nil, fmt.Errorf("unreachable in test")
}

// ---------------------------------------------------------------------
// probeAddressFor — the crux
// ---------------------------------------------------------------------

// A same-host container gateway (Backend "docker") resolves a devnet
// upstream, for eRPC's own purposes, to the container-name form
// (ws://valve-node-app-devnet:...) — see
// TestGateways_SameMachineContainersTalkByContainerName. probeAddressFor must
// NOT hand that name to the local prober: this process is not a container on
// the engine's network however "same host" the gateway and devnet are to
// each other, and the name resolves nowhere else.
func TestCapabilities_ProbeAddressFor_DevnetPublishedNotContainer(t *testing.T) {
	cfg := config.Config{Targets: []config.Target{
		{ID: "local", Mode: "local", Devnet: &catalog.DevnetConfig{HTTPPort: 8600, WSPort: 8601}},
	}}
	gw := config.Gateway{ID: "default", Placement: config.GatewayPlacement{TargetID: "local", Backend: "docker"}}
	u := catalog.GatewayUpstream{ID: "devnet", Kind: catalog.UpstreamManagedDevnet, TargetID: "local"}

	probeURL, unprobeable := probeAddressFor(cfg, gw, u)
	if unprobeable != "" {
		t.Fatalf("unprobeable = %q, want a probeable devnet", unprobeable)
	}
	if want := "ws://127.0.0.1:8601"; probeURL != want {
		t.Fatalf("probeURL = %q, want %q — the devnet's PUBLISHED ws endpoint", probeURL, want)
	}
	if strings.Contains(probeURL, "valve-node-app-devnet") {
		t.Fatalf("probeURL %q names the container — that resolves only inside the docker engine's own network, never on this process's host", probeURL)
	}
}

// The HTTP and WS ports differ by default (8600 vs 8601): a probeAddressFor
// that derived ws:// from the HTTP endpoint via DeriveWSURL, instead of using
// WSEndpoint directly, would point at the wrong port and report a false
// "WebSocket unsupported". This pins the port, not just the scheme.
func TestCapabilities_ProbeAddressFor_DevnetKeepsWSPortNotHTTPPort(t *testing.T) {
	cfg := config.Config{Targets: []config.Target{
		{ID: "local", Mode: "local", Devnet: &catalog.DevnetConfig{HTTPPort: 9111, WSPort: 9222}},
	}}
	gw := config.Gateway{ID: "default", Placement: config.GatewayPlacement{TargetID: "local", Backend: "docker"}}
	u := catalog.GatewayUpstream{Kind: catalog.UpstreamManagedDevnet, TargetID: "local"}

	probeURL, unprobeable := probeAddressFor(cfg, gw, u)
	if unprobeable != "" {
		t.Fatalf("unprobeable = %q", unprobeable)
	}
	if want := "ws://127.0.0.1:9222"; probeURL != want {
		t.Fatalf("probeURL = %q, want %q (the WS port, not the HTTP one)", probeURL, want)
	}
}

// An upstream on an SSH-reached machine, bound to loopback, is a node this
// process cannot dial: 127.0.0.1 there is that machine's own loopback, not a
// hop this process can take. This holds even when the gateway itself is
// placed on that very machine (so reachableAcrossMachines, which asks a
// different question — can eRPC reach it — is satisfied): the two functions
// answer from two different vantage points, and probeAddressFor's is always
// "here", never "the gateway's machine".
func TestCapabilities_ProbeAddressFor_SSHLoopbackIsUnprobeable(t *testing.T) {
	cfg := config.Config{Targets: []config.Target{
		{ID: "boxa", Mode: "ssh", SSH: &executor.SSHConfig{Host: "10.0.0.5"},
			Wire: &catalog.WireConfig{ChainID: 369, ExecID: "reth"}}, // RPCBindAddr unset → loopback
	}}
	gw := config.Gateway{ID: "default", Placement: config.GatewayPlacement{TargetID: "boxa", Backend: "docker"}}
	u := catalog.GatewayUpstream{ID: "node", Kind: catalog.UpstreamManagedNode, TargetID: "boxa"}

	probeURL, unprobeable := probeAddressFor(cfg, gw, u)
	if probeURL != "" {
		t.Fatalf("probeURL = %q, want empty", probeURL)
	}
	for _, want := range []string{"boxa", "loopback", "reachable from there"} {
		if !strings.Contains(unprobeable, want) {
			t.Errorf("unprobeable %q missing %q", unprobeable, want)
		}
	}
}

// The other half of the same rule: an SSH target with a ROUTABLE address is
// probed exactly like eRPC would probe it — same host, same port.
func TestCapabilities_ProbeAddressFor_SSHRoutableIsProbedNormally(t *testing.T) {
	cfg := config.Config{Targets: []config.Target{
		{ID: "boxa", Mode: "ssh", SSH: &executor.SSHConfig{Host: "10.0.0.5"},
			Wire: &catalog.WireConfig{ChainID: 369, ExecID: "reth", RPCBindAddr: "100.64.0.7"}},
	}}
	gw := config.Gateway{ID: "default", Placement: config.GatewayPlacement{TargetID: "here", Backend: "docker"}}
	u := catalog.GatewayUpstream{Kind: catalog.UpstreamManagedNode, TargetID: "boxa"}

	probeURL, unprobeable := probeAddressFor(cfg, gw, u)
	if unprobeable != "" {
		t.Fatalf("unprobeable = %q, want a probeable routable node", unprobeable)
	}
	if want := "http://100.64.0.7:8545"; probeURL != want {
		t.Fatalf("probeURL = %q, want %q", probeURL, want)
	}
}

// A LOCAL target's loopback is OUR loopback: Mode "local" names the machine
// this very process runs on, so a managed node there is probed exactly where
// eRPC would probe it, loopback and all.
func TestCapabilities_ProbeAddressFor_LocalLoopbackIsProbedNormally(t *testing.T) {
	cfg := config.Config{Targets: []config.Target{
		{ID: "here", Mode: "local", Wire: &catalog.WireConfig{ChainID: 369, ExecID: "reth"}},
	}}
	gw := config.Gateway{ID: "default", Placement: config.GatewayPlacement{TargetID: "here", Backend: "docker"}}
	u := catalog.GatewayUpstream{Kind: catalog.UpstreamManagedNode, TargetID: "here"}

	probeURL, unprobeable := probeAddressFor(cfg, gw, u)
	if unprobeable != "" {
		t.Fatalf("unprobeable = %q, want a probeable local node", unprobeable)
	}
	if want := "http://127.0.0.1:8545"; probeURL != want {
		t.Fatalf("probeURL = %q, want %q", probeURL, want)
	}
}

// A dead reference (a machine no longer registered) is unprobeable from
// anywhere, and says so rather than returning an empty URL with no reason.
func TestCapabilities_ProbeAddressFor_DeadReferenceIsUnprobeable(t *testing.T) {
	cfg := config.Config{}
	gw := config.Gateway{ID: "default", Placement: config.GatewayPlacement{TargetID: "here", Backend: "docker"}}
	u := catalog.GatewayUpstream{Kind: catalog.UpstreamManagedNode, TargetID: "gone"}

	probeURL, unprobeable := probeAddressFor(cfg, gw, u)
	if probeURL != "" {
		t.Fatalf("probeURL = %q, want empty", probeURL)
	}
	if !strings.Contains(unprobeable, "gone") {
		t.Errorf("unprobeable %q must name the missing machine", unprobeable)
	}
}

// An external endpoint carries no target of ours to reason about, so it is
// always probed as given, loopback or not.
func TestCapabilities_ProbeAddressFor_ExternalIsAlwaysProbed(t *testing.T) {
	cfg := config.Config{}
	gw := config.Gateway{ID: "default"}
	u := catalog.GatewayUpstream{Kind: catalog.UpstreamExternal, Endpoint: "https://rpc.example.com"}

	probeURL, unprobeable := probeAddressFor(cfg, gw, u)
	if unprobeable != "" {
		t.Fatalf("unprobeable = %q", unprobeable)
	}
	if probeURL != "https://rpc.example.com" {
		t.Fatalf("probeURL = %q, want the operator's own endpoint unchanged", probeURL)
	}
}

// ---------------------------------------------------------------------
// unprobeable endpoints are never dialed
// ---------------------------------------------------------------------

func TestCapabilities_UnprobeableEndpointIsNeverDialed(t *testing.T) {
	cfg := config.Config{Targets: []config.Target{
		{ID: "boxa", Mode: "ssh", SSH: &executor.SSHConfig{Host: "10.0.0.5"},
			Wire: &catalog.WireConfig{ChainID: 369, ExecID: "reth"}},
	}}
	gw := config.Gateway{
		ID:        "default",
		Placement: config.GatewayPlacement{TargetID: "boxa", Backend: "docker"},
		Config: catalog.GatewayConfig{Networks: []catalog.GatewayNetwork{{ChainID: 369, Upstreams: []catalog.GatewayUpstream{
			{ID: "node", Kind: catalog.UpstreamManagedNode, TargetID: "boxa"},
		}}}},
	}

	prober := fastTestProber()
	prober.HTTPClient = &http.Client{Transport: poisonedTransport{t}}

	res := probeGatewayCapabilities(context.Background(), prober, cfg, gw)
	if len(res.Endpoints) != 1 {
		t.Fatalf("got %d endpoints, want 1", len(res.Endpoints))
	}
	ep := res.Endpoints[0]
	if ep.Unprobeable == "" {
		t.Fatal("Unprobeable is empty, want the loopback-on-SSH reason")
	}
	if ep.Reachable {
		t.Error("an unprobeable endpoint must not read as reachable")
	}
	if ep.ProbedURL != "" {
		t.Errorf("ProbedURL = %q, want empty — nothing was dialed", ep.ProbedURL)
	}
	if len(ep.Capabilities) != len(capabilities.Keys()) {
		t.Fatalf("got %d capability rows, want one per capabilities.Keys() (%d)", len(ep.Capabilities), len(capabilities.Keys()))
	}
	for _, c := range ep.Capabilities {
		if c.Status != string(capabilities.StatusInconclusive) {
			t.Errorf("capability %q status = %q, want inconclusive", c.Key, c.Status)
		}
		if !strings.Contains(c.Detail, "loopback") {
			t.Errorf("capability %q detail %q should carry the same reason as the row", c.Key, c.Detail)
		}
	}
}

// ---------------------------------------------------------------------
// probe repetition
// ---------------------------------------------------------------------

func TestCapabilities_ExternalProbedThriceManagedOnce(t *testing.T) {
	extCounter := &chainIDCounter{}
	extSrv := capCountingServer(t, extCounter, 369)

	mgdCounter := &chainIDCounter{}
	mgdSrv := capCountingServer(t, mgdCounter, 369)
	mgdHost, mgdPort := hostPort(t, mgdSrv.URL)

	cfg := config.Config{Targets: []config.Target{
		{ID: "local", Mode: "local", Wire: &catalog.WireConfig{ChainID: 369, ExecID: "reth", RPCBindAddr: mgdHost, ExecHTTPPort: mgdPort}},
	}}
	gw := config.Gateway{
		ID:        "default",
		Placement: config.GatewayPlacement{TargetID: "local", Backend: "docker"},
		Config: catalog.GatewayConfig{Networks: []catalog.GatewayNetwork{{ChainID: 369, Upstreams: []catalog.GatewayUpstream{
			{ID: "ext", Kind: catalog.UpstreamExternal, Endpoint: extSrv.URL},
			{ID: "node", Kind: catalog.UpstreamManagedNode, TargetID: "local"},
		}}}},
	}

	res := probeGatewayCapabilities(context.Background(), fastTestProber(), cfg, gw)
	if len(res.Endpoints) != 2 {
		t.Fatalf("got %d endpoints, want 2", len(res.Endpoints))
	}

	if got := extCounter.calls(); got != 3 {
		t.Errorf("external endpoint: got %d eth_chainId calls, want 3 — a public endpoint may load-balance across backends that disagree", got)
	}
	if got := mgdCounter.calls(); got != 1 {
		t.Errorf("managed endpoint: got %d eth_chainId calls, want 1 — a single node cannot disagree with itself", got)
	}
}

// ---------------------------------------------------------------------
// every key present, even unreachable
// ---------------------------------------------------------------------

func TestCapabilities_UnreachableStillCarriesEveryKey(t *testing.T) {
	srv := downServer(t)

	gw := config.Gateway{
		ID:        "default",
		Placement: config.GatewayPlacement{TargetID: "local", Backend: "docker"},
		Config: catalog.GatewayConfig{Networks: []catalog.GatewayNetwork{{ChainID: 369, Upstreams: []catalog.GatewayUpstream{
			{ID: "dead", Kind: catalog.UpstreamExternal, Endpoint: srv.URL},
		}}}},
	}

	res := probeGatewayCapabilities(context.Background(), fastTestProber(), config.Config{}, gw)
	if len(res.Endpoints) != 1 {
		t.Fatalf("got %d endpoints, want 1", len(res.Endpoints))
	}
	ep := res.Endpoints[0]
	if ep.Reachable {
		t.Error("a 500-answering endpoint must not read as reachable")
	}
	if ep.Unprobeable != "" {
		t.Errorf("Unprobeable = %q, want empty — this endpoint WAS dialed, it just did not answer usefully", ep.Unprobeable)
	}

	want := capabilities.Keys()
	if len(ep.Capabilities) != len(want) {
		t.Fatalf("got %d capability rows, want %d (one per capabilities.Keys())", len(ep.Capabilities), len(want))
	}
	seen := make(map[string]bool, len(ep.Capabilities))
	for _, c := range ep.Capabilities {
		seen[c.Key] = true
		if c.Status != string(capabilities.StatusInconclusive) {
			t.Errorf("capability %q status = %q, want inconclusive for an unreachable endpoint", c.Key, c.Status)
		}
	}
	for _, k := range want {
		if !seen[k] {
			t.Errorf("capabilities.Keys() key %q missing from the response entirely — a short array would look identical to a key never asked about", k)
		}
	}
}

// ---------------------------------------------------------------------
// caching / TTL / refresh
// ---------------------------------------------------------------------

func TestCapabilities_CachedWithinTTLRefreshForcesAProbe(t *testing.T) {
	counter := &chainIDCounter{}
	srv := capCountingServer(t, counter, 369)

	a := gatewayServer(t)
	addGateway(t, a, "default", "local", catalog.GatewayConfig{
		Port: 4100,
		Networks: []catalog.GatewayNetwork{{ChainID: 369, Upstreams: []catalog.GatewayUpstream{
			{ID: "ext", Kind: catalog.UpstreamExternal, Endpoint: srv.URL},
		}}},
	})

	first := decode[capabilitiesResponse](t, a.do(t, "GET", "/api/gateways/default/capabilities", nil))
	afterFirst := counter.calls()
	if afterFirst != 3 {
		t.Fatalf("first request: got %d eth_chainId calls, want 3 (external repeat)", afterFirst)
	}

	second := decode[capabilitiesResponse](t, a.do(t, "GET", "/api/gateways/default/capabilities", nil))
	afterSecond := counter.calls()
	if afterSecond != afterFirst {
		t.Fatalf("a second request inside the TTL re-probed: %d -> %d eth_chainId calls", afterFirst, afterSecond)
	}
	if !second.At.Equal(first.At) {
		t.Errorf("a cached response must carry the ORIGINAL probe time, got %v want %v", second.At, first.At)
	}

	third := a.do(t, "GET", "/api/gateways/default/capabilities?refresh=1", nil)
	_ = decode[capabilitiesResponse](t, third)
	afterThird := counter.calls()
	if afterThird <= afterSecond {
		t.Fatalf("?refresh=1 must force a fresh probe: eth_chainId calls stayed at %d", afterThird)
	}
}

// ---------------------------------------------------------------------
// 404
// ---------------------------------------------------------------------

func TestCapabilities_UnknownGatewayIs404(t *testing.T) {
	a := gatewayServer(t)
	res := a.do(t, "GET", "/api/gateways/nope/capabilities", nil)
	defer res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("got %d, want 404", res.StatusCode)
	}
}

// ---------------------------------------------------------------------
// upstream id derivation matches the traffic route's
// ---------------------------------------------------------------------

func TestCapabilities_UpstreamIDsMatchIntentsFor(t *testing.T) {
	deadA := downServer(t)
	deadB := downServer(t)

	cfg := config.Config{Targets: []config.Target{{ID: "local", Mode: "local"}}}
	gw := config.Gateway{
		ID:        "default",
		Placement: config.GatewayPlacement{TargetID: "local", Backend: "docker"},
		Config: catalog.GatewayConfig{Networks: []catalog.GatewayNetwork{{ChainID: 369, Upstreams: []catalog.GatewayUpstream{
			{Kind: catalog.UpstreamExternal, Endpoint: deadA.URL},
			{Kind: catalog.UpstreamExternal, Endpoint: deadB.URL, Local: true},
		}}}},
	}

	resolved, _ := resolveGateway(cfg, gw)
	if len(resolved.Networks) != 1 {
		t.Fatalf("got %d resolved networks, want 1", len(resolved.Networks))
	}
	wantIntents := setup.IntentsFor(resolved.Networks[0])

	res := probeGatewayCapabilities(context.Background(), fastTestProber(), cfg, gw)
	if len(res.Endpoints) != len(wantIntents) {
		t.Fatalf("got %d capability endpoints, want %d (setup.IntentsFor)", len(res.Endpoints), len(wantIntents))
	}
	for i, ep := range res.Endpoints {
		if ep.Upstream != wantIntents[i].Upstream {
			t.Errorf("endpoint %d id = %q, want %q — the traffic route derives its id the same way, and the UI joins on this string", i, ep.Upstream, wantIntents[i].Upstream)
		}
	}
}
