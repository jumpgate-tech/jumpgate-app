package server

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/valve-tech/valve-node-app/internal/catalog"
	"github.com/valve-tech/valve-node-app/internal/config"
	"github.com/valve-tech/valve-node-app/internal/executor"
	"github.com/valve-tech/valve-node-app/internal/metrics"
)

// ---------------------------------------------------------------------
// fixtures and helpers
// ---------------------------------------------------------------------

// erpcFixture is the real captured eRPC dump internal/metrics' own tests
// parse against. Reused here so the happy-path route test exercises the
// exact body a real gateway produces rather than a hand-written stand-in.
func erpcFixture(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile("../metrics/testdata/erpc.txt")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	return string(b)
}

// trafficGatewayConfig is a one-chain gateway whose single upstream id
// matches the fixture's own upstream label exactly ("public-369-1"), so a
// scrape of the fixture resolves to a fully CONFIGURED row rather than an
// unconfigured one.
func trafficGatewayConfig() catalog.GatewayConfig {
	return catalog.GatewayConfig{
		Port: 4100,
		Networks: []catalog.GatewayNetwork{
			{ChainID: 369, Upstreams: []catalog.GatewayUpstream{
				{ID: "public-369-1", Endpoint: "https://rpc.pulsechain.com", Local: true},
			}},
		},
	}
}

// trafficAPIServer is gatewayServer's shape (a docker-backed target ready for
// gateway creation) but built around a caller-supplied executor, so a test
// can script exactly what the metrics curl returns without disturbing the
// plain dockerExecutor every other gateway test relies on.
func trafficAPIServer(t *testing.T, e *scriptedExecutor) *apiTestServer {
	t.Helper()
	a := newAPITestServerWithExecutor(t, func(config.Target) (executor.Executor, error) { return e, nil })
	addTarget(t, a)
	return a
}

// readyExecutor is dockerExecutor's usual "engine present, one running
// container" script, which is what lets a gateway be created and viewed at
// all — the traffic route itself only ever adds a curl script on top.
func readyExecutor() *scriptedExecutor {
	return dockerExecutor("true|0|valve-node-app/erpc:e909aacb|sha256:abc\n", "4000/tcp=127.0.0.1:4100\n")
}

// callsOf returns every command a scriptedExecutor was asked to run, safe
// for concurrent access with the executor's own bookkeeping.
func callsOf(e *scriptedExecutor) []string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return append([]string(nil), e.calls...)
}

// ---------------------------------------------------------------------
// handleGatewayTraffic, through the real mux
// ---------------------------------------------------------------------

func TestHandleGatewayTraffic_UnknownGatewayIs404WithTheTypedCode(t *testing.T) {
	a := trafficAPIServer(t, readyExecutor())

	res := a.do(t, "GET", "/api/gateways/nope/traffic", nil)
	body := decode[errorDetail](t, res)
	if res.StatusCode != http.StatusNotFound || body.Code != codeGatewayNotFound {
		t.Fatalf("got %d/%q, want 404/%s", res.StatusCode, body.Code, codeGatewayNotFound)
	}
}

// Metrics off is a setting, not a failure: the response must be 200, with
// Enabled false and no Error string, so the UI does not render it red.
func TestHandleGatewayTraffic_MetricsOffIs200WithNoErrorString(t *testing.T) {
	e := readyExecutor()
	a := trafficAPIServer(t, e)
	g := trafficGatewayConfig()
	g.MetricsOff = true
	addGateway(t, a, "default", "local", g)

	res := a.do(t, "GET", "/api/gateways/default/traffic", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("metrics-off must answer 200, got %d", res.StatusCode)
	}
	var body trafficResponse
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	res.Body.Close()

	if body.Enabled {
		t.Error("want Enabled:false")
	}
	if body.Error != "" {
		t.Errorf("want no Error string — it is a setting, and Enabled:false already says so; an Error here would render the card red for no failure at all, got %q", body.Error)
	}

	for _, cmd := range callsOf(e) {
		if strings.Contains(cmd, "curl") {
			t.Fatalf("a disabled gateway must never be scraped, got %q", cmd)
		}
	}
}

// A scrape failure (metrics enabled, but curl fails) must still answer 200:
// the gateway itself may be perfectly healthy and merely unreadable, and a
// 5xx here would tell the screen the gateway is down when it might not be.
func TestHandleGatewayTraffic_ScrapeFailureIs200WithErrorSetAndGatewayIntact(t *testing.T) {
	e := readyExecutor().script("curl -s --max-time", executor.Result{
		ExitCode: 7,
		Stderr:   "Failed to connect to 127.0.0.1 port 4001: Connection refused",
	})
	a := trafficAPIServer(t, e)
	addGateway(t, a, "default", "local", trafficGatewayConfig())

	res := a.do(t, "GET", "/api/gateways/default/traffic", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("a scrape failure must not become an HTTP error status: got %d, want 200", res.StatusCode)
	}
	var body trafficResponse
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	res.Body.Close()

	if body.Error == "" {
		t.Fatal("want Error set when the scrape itself fails")
	}
	// The gateway is otherwise intact: its own metrics setting (on) is still
	// reported accurately, unclouded by the scrape having failed.
	if !body.Enabled {
		t.Error("Enabled must still reflect the gateway's own setting, not the scrape outcome")
	}
	if len(body.Networks) != 0 {
		t.Errorf("a failed scrape has nothing to report per-network: got %+v", body.Networks)
	}
}

// The full happy path, real fixture and all: proves the route is wired into
// the mux, the scrape runs against the operator's own metrics port, the
// fixture parses, and the wire carries the JSON field names the TypeScript
// client depends on (not the exported Go field names).
func TestHandleGatewayTraffic_HappyPathUsesTheWireFieldNames(t *testing.T) {
	e := readyExecutor().script("curl -s --max-time", executor.Result{Stdout: erpcFixture(t)})
	a := trafficAPIServer(t, e)
	addGateway(t, a, "default", "local", trafficGatewayConfig())

	res := a.do(t, "GET", "/api/gateways/default/traffic", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("got %d, want 200", res.StatusCode)
	}
	var raw map[string]any
	if err := json.NewDecoder(res.Body).Decode(&raw); err != nil {
		t.Fatalf("decode: %v", err)
	}
	res.Body.Close()

	for _, field := range []string{"enabled", "at", "since", "networks"} {
		if _, ok := raw[field]; !ok {
			t.Errorf("response missing %q: %v", field, raw)
		}
	}
	if e, ok := raw["error"]; ok {
		t.Errorf("a healthy scrape must not carry an error field at all (omitempty), got %v", e)
	}

	networks, _ := raw["networks"].([]any)
	if len(networks) != 1 {
		t.Fatalf("got %d network rows, want 1: %v", len(networks), networks)
	}
	nrow, _ := networks[0].(map[string]any)
	for _, field := range []string{"chainId", "received", "attributed", "upstreams"} {
		if _, ok := nrow[field]; !ok {
			t.Errorf("network row missing %q: %v", field, nrow)
		}
	}
	if got := nrow["chainId"]; got != float64(369) {
		t.Errorf("chainId: got %v, want 369", got)
	}

	upstreams, _ := nrow["upstreams"].([]any)
	if len(upstreams) != 1 {
		t.Fatalf("got %d upstream rows, want 1: %v", len(upstreams), upstreams)
	}
	urow, _ := upstreams[0].(map[string]any)
	for _, field := range []string{"upstream", "succeeded", "actual", "intended", "diverged"} {
		if _, ok := urow[field]; !ok {
			t.Errorf("upstream row missing %q: %v", field, urow)
		}
	}
	// This upstream's id matches the gateway's configuration exactly, so the
	// omitempty unconfigured field must not appear at all.
	if _, ok := urow["unconfigured"]; ok {
		t.Errorf("a configured upstream must not carry unconfigured (omitempty), got %v", urow)
	}
}

// ---------------------------------------------------------------------
// trafficViews
// ---------------------------------------------------------------------

// The join is over the CONFIGURED networks, not the measured ones: a chain
// nobody has ever called must still show up as a row of zeroes, and an
// upstream the scrape still remembers but the current config no longer lists
// must be included and flagged, not silently absorbed or dropped.
func TestTrafficViews_JoinsOverConfiguredNetworksAndFlagsAnUnconfiguredUpstream(t *testing.T) {
	g := catalog.GatewayConfig{Networks: []catalog.GatewayNetwork{
		{ChainID: 369, Upstreams: []catalog.GatewayUpstream{
			{ID: "local-1", Local: true},
			{ID: "fallback-1"},
		}},
		// Configured, but the scrape below never mentions it — the "gateway
		// nobody has used yet" case.
		{ChainID: 1337, Upstreams: []catalog.GatewayUpstream{
			{ID: "never-called", Local: true},
		}},
	}}

	traffic := metrics.Traffic{Networks: []metrics.NetworkTraffic{
		{ChainID: 369, Received: 10, Upstreams: []metrics.UpstreamTraffic{
			{Upstream: "local-1", Succeeded: 7},
			{Upstream: "fallback-1", Succeeded: 0},
			// Still being counted by eRPC's cumulative (reset-on-restart-only)
			// counters, but removed from the config since — the state between
			// saving an edit and re-creating the container.
			{Upstream: "retired-upstream", Succeeded: 3},
		}},
		// chain 1337 does not appear in the scrape at all.
	}}

	views := trafficViews(g, traffic)
	if len(views) != 2 {
		t.Fatalf("got %d network rows, want 2 (one per CONFIGURED chain): %+v", len(views), views)
	}

	var chain369, chain1337 *networkTrafficView
	for i := range views {
		switch views[i].ChainID {
		case 369:
			chain369 = &views[i]
		case 1337:
			chain1337 = &views[i]
		}
	}
	if chain1337 == nil {
		t.Fatal("chain 1337 is configured but is missing from the response")
	}
	if chain1337.Received != 0 || chain1337.Attributed != 0 {
		t.Fatalf("a configured chain with zero measured traffic must be a row of zeroes, not vanish: %+v", chain1337)
	}
	if len(chain1337.Upstreams) != 1 || chain1337.Upstreams[0].Unconfigured {
		t.Fatalf("the configured-but-silent upstream must appear and must not be flagged unconfigured: %+v", chain1337.Upstreams)
	}

	if chain369 == nil {
		t.Fatal("chain 369 is missing from the response")
	}
	var retired *upstreamShareView
	for i := range chain369.Upstreams {
		if chain369.Upstreams[i].Upstream == "retired-upstream" {
			retired = &chain369.Upstreams[i]
		}
	}
	if retired == nil {
		t.Fatal("an upstream present in the counters but absent from the configuration must still be included, not dropped")
	}
	if !retired.Unconfigured {
		t.Error("want Unconfigured:true — this is the signal that a config edit has not been applied to the running container yet")
	}
	for _, u := range chain369.Upstreams {
		if u.Upstream != "retired-upstream" && u.Unconfigured {
			t.Errorf("a genuinely configured upstream (%s) must not be flagged Unconfigured", u.Upstream)
		}
	}
}

// Attributed sums the per-upstream successes; Received comes straight off the
// network counter. A chain failing every single call must therefore show
// Received > 0 with Attributed == 0 — the whole reason both numbers ride the
// wire rather than just a computed percentage.
func TestTrafficViews_AttributedSumsSuccessesReceivedIsTheNetworkCounter(t *testing.T) {
	g := catalog.GatewayConfig{Networks: []catalog.GatewayNetwork{
		{ChainID: 369, Upstreams: []catalog.GatewayUpstream{{ID: "only-upstream", Local: true}}},
	}}
	// 10 clients asked for chain 369 and every one of them failed: no
	// upstream ever recorded a success, so Attributed must be 0 even though
	// Received is not — distinct from "nobody has called this chain",
	// which would be Received == 0 too.
	traffic := metrics.Traffic{Networks: []metrics.NetworkTraffic{
		{ChainID: 369, Received: 10, Upstreams: []metrics.UpstreamTraffic{
			{Upstream: "only-upstream", Succeeded: 0},
		}},
	}}

	views := trafficViews(g, traffic)
	if len(views) != 1 {
		t.Fatalf("got %d rows, want 1: %+v", len(views), views)
	}
	nv := views[0]
	if nv.Received != 10 {
		t.Fatalf("Received: got %v, want 10", nv.Received)
	}
	if nv.Attributed != 0 {
		t.Fatalf("Attributed: got %v, want 0 — every call to this chain failed, so no upstream succeeded", nv.Attributed)
	}
}
