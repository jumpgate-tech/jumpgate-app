package server

import (
	"encoding/json"
	"math"
	"net/http"
	"strings"
	"testing"

	"github.com/valve-tech/valve-node-app/internal/catalog"
	"github.com/valve-tech/valve-node-app/internal/executor"
)

// The happy path against the real captured dump: the route is mounted, the
// scrape runs, both folds land, and the wire carries the JSON names the
// TypeScript client reads.
func TestHandleGatewayAnalytics_HappyPathCarriesBothFolds(t *testing.T) {
	e := readyExecutor().script("curl -s --max-time", executor.Result{Stdout: erpcFixture(t)})
	a := trafficAPIServer(t, e)
	addGateway(t, a, "default", "local", trafficGatewayConfig())

	res := a.do(t, "GET", "/api/gateways/default/analytics", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("got %d, want 200", res.StatusCode)
	}
	var raw map[string]any
	if err := json.NewDecoder(res.Body).Decode(&raw); err != nil {
		t.Fatalf("decode: %v", err)
	}
	res.Body.Close()

	for _, field := range []string{"enabled", "at", "since", "networks", "endpoints"} {
		if _, ok := raw[field]; !ok {
			t.Errorf("response missing %q: %v", field, raw)
		}
	}

	networks, _ := raw["networks"].([]any)
	if len(networks) != 1 {
		t.Fatalf("networks: %v, want the one configured chain", networks)
	}
	n, _ := networks[0].(map[string]any)
	for _, field := range []string{"chainId", "name", "received", "answered", "failed", "methods", "endpoints"} {
		if _, ok := n[field]; !ok {
			t.Errorf("network row missing %q: %v", field, n)
		}
	}
	// Five requests received, five answered, none failed — the fixture's real
	// client traffic, not the poller's hundreds.
	if n["received"] != float64(5) || n["answered"] != float64(5) || n["failed"] != float64(0) {
		t.Errorf("volumes: got received=%v answered=%v failed=%v, want 5/5/0", n["received"], n["answered"], n["failed"])
	}

	methods, _ := n["methods"].([]any)
	if len(methods) != 1 {
		t.Fatalf("methods: %v, want only eth_blockNumber", methods)
	}
	m, _ := methods[0].(map[string]any)
	if m["method"] != "eth_blockNumber" || m["count"] != float64(5) {
		t.Errorf("method row: %v", m)
	}
	if mean, ok := m["mean"].(float64); !ok || math.Abs(mean-0.834064999/5) > 1e-9 {
		t.Errorf("mean: %v", m["mean"])
	}

	// The last bucket bound is the string "+Inf". Sent as a float it would be
	// unmarshallable and take the whole response down with it.
	buckets, _ := m["buckets"].([]any)
	if len(buckets) != 5 {
		t.Fatalf("buckets: %v", buckets)
	}
	last, _ := buckets[4].(map[string]any)
	if last["le"] != "+Inf" {
		t.Errorf("final bucket bound: got %v, want the string \"+Inf\"", last["le"])
	}

	endpoints, _ := raw["endpoints"].([]any)
	if len(endpoints) == 0 {
		t.Fatal("endpoints: none — the gateway's own view of its upstreams is half this page")
	}
	var devnet map[string]any
	for _, x := range endpoints {
		row, _ := x.(map[string]any)
		if row["upstream"] == "devnet" {
			devnet = row
		}
	}
	if devnet == nil {
		t.Fatalf("no devnet endpoint row: %v", endpoints)
	}
	// The devnet is not in this gateway's configuration at all, and it is
	// still reported — with 56 transport failures no client ever saw, which is
	// exactly the kind of thing this page exists to surface.
	if devnet["configured"] != false {
		t.Errorf("configured: got %v, want false", devnet["configured"])
	}
	errs, _ := devnet["errors"].([]any)
	if len(errs) != 1 {
		t.Fatalf("errors: %v", errs)
	}
	er, _ := errs[0].(map[string]any)
	if er["class"] != "ErrEndpointTransportFailure" || er["count"] != float64(56) || er["severity"] != "critical" {
		t.Errorf("error row: %v", er)
	}
}

// A method nobody has called must not claim to be infinitely fast. null is
// renderable as "—"; 0 renders as "0ms", which is a statement about speed.
func TestHandleGatewayAnalytics_MeanOfNothingIsNull(t *testing.T) {
	dump := `# HELP process_start_time_seconds Start time.
process_start_time_seconds 1.7851988e+09
erpc_network_request_duration_seconds_count{category="eth_call",network="evm:369",project="main",upstream="public-369-1"} 0
`
	e := readyExecutor().script("curl -s --max-time", executor.Result{Stdout: dump})
	a := trafficAPIServer(t, e)
	addGateway(t, a, "default", "local", trafficGatewayConfig())

	var body struct {
		Networks []struct {
			Methods []struct {
				Method string   `json:"method"`
				Mean   *float64 `json:"mean"`
			} `json:"methods"`
		} `json:"networks"`
	}
	res := a.do(t, "GET", "/api/gateways/default/analytics", nil)
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	res.Body.Close()

	if len(body.Networks) != 1 || len(body.Networks[0].Methods) != 1 {
		t.Fatalf("body: %+v", body)
	}
	if got := body.Networks[0].Methods[0].Mean; got != nil {
		t.Errorf("mean: got %v, want null for a method that has answered nothing", *got)
	}
}

// A chain nobody has called still gets a row. A gateway nobody has used should
// look like a gateway nobody has used, not like a gateway with no chains.
func TestHandleGatewayAnalytics_ConfiguredChainWithNoTrafficStillHasARow(t *testing.T) {
	dump := "process_start_time_seconds 1.7851988e+09\n"
	e := readyExecutor().script("curl -s --max-time", executor.Result{Stdout: dump})
	a := trafficAPIServer(t, e)
	addGateway(t, a, "default", "local", trafficGatewayConfig())

	var body analyticsResponse
	res := a.do(t, "GET", "/api/gateways/default/analytics", nil)
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	res.Body.Close()

	if len(body.Networks) != 1 {
		t.Fatalf("networks: %+v, want the configured chain with zeroes", body.Networks)
	}
	if body.Networks[0].ChainID != 369 || body.Networks[0].Name == "" {
		t.Errorf("row: %+v — the chain must be named, not just numbered", body.Networks[0])
	}
	if body.Error != "" {
		t.Errorf("a dump with no counters in it is not an error: %q", body.Error)
	}
}

// Metrics off is a setting: 200, Enabled false, no error string, and above all
// no scrape.
func TestHandleGatewayAnalytics_MetricsOffIsNotAnError(t *testing.T) {
	e := readyExecutor()
	a := trafficAPIServer(t, e)
	g := trafficGatewayConfig()
	g.MetricsOff = true
	addGateway(t, a, "default", "local", g)

	res := a.do(t, "GET", "/api/gateways/default/analytics", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("got %d, want 200", res.StatusCode)
	}
	var body analyticsResponse
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	res.Body.Close()

	if body.Enabled || body.Error != "" {
		t.Errorf("got enabled=%v error=%q, want false and empty", body.Enabled, body.Error)
	}
	for _, cmd := range callsOf(e) {
		if strings.Contains(cmd, "curl") {
			t.Fatalf("a gateway with its counters off must never be scraped: %q", cmd)
		}
	}
}

// An unreadable gateway is not a down gateway. 200 with the reason, so the
// screen can explain itself instead of implying the gateway has failed.
func TestHandleGatewayAnalytics_ScrapeFailureIs200WithTheReason(t *testing.T) {
	e := readyExecutor().script("curl -s --max-time", executor.Result{
		ExitCode: 7,
		Stderr:   "Failed to connect to 127.0.0.1 port 4001: Connection refused",
	})
	a := trafficAPIServer(t, e)
	addGateway(t, a, "default", "local", trafficGatewayConfig())

	res := a.do(t, "GET", "/api/gateways/default/analytics", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("got %d, want 200", res.StatusCode)
	}
	var body analyticsResponse
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	res.Body.Close()

	if body.Error == "" {
		t.Fatal("want the reason on the response")
	}
	if !body.Enabled {
		t.Error("Enabled reports the operator's setting, not the scrape outcome")
	}
	if len(body.Networks) != 0 || len(body.Endpoints) != 0 {
		t.Errorf("a failed scrape has nothing to report: %+v %+v", body.Networks, body.Endpoints)
	}
}

func TestHandleGatewayAnalytics_UnknownGatewayIs404(t *testing.T) {
	a := trafficAPIServer(t, readyExecutor())
	res := a.do(t, "GET", "/api/gateways/nope/analytics", nil)
	body := decode[errorDetail](t, res)
	if res.StatusCode != http.StatusNotFound || body.Code != codeGatewayNotFound {
		t.Fatalf("got %d/%q, want 404/%s", res.StatusCode, body.Code, codeGatewayNotFound)
	}
}

// Failure is received minus everything that came back — including what the
// gateway answered from its own cache, which succeeded without any endpoint.
// Counting cache hits as failures would invent an outage out of the fastest
// requests the gateway serves.
func TestHandleGatewayAnalytics_CacheHitsAreNotFailures(t *testing.T) {
	dump := `process_start_time_seconds 1.7851988e+09
erpc_network_request_received_total{network="evm:369",project="main"} 10
erpc_network_successful_request_total{network="evm:369",project="main",upstream="public-369-1"} 7
erpc_network_successful_request_total{network="evm:369",project="main",upstream="n/a"} 3
`
	e := readyExecutor().script("curl -s --max-time", executor.Result{Stdout: dump})
	a := trafficAPIServer(t, e)
	addGateway(t, a, "default", "local", trafficGatewayConfig())

	var body analyticsResponse
	res := a.do(t, "GET", "/api/gateways/default/analytics", nil)
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	res.Body.Close()

	n := body.Networks[0]
	if n.Answered != 7 || n.Unattributed != 3 || n.Failed != 0 {
		t.Errorf("got answered=%v unattributed=%v failed=%v, want 7/3/0", n.Answered, n.Unattributed, n.Failed)
	}
}

// The counters are read in one scrape but are not written atomically, so a
// success can land a microsecond after the received counter it belongs to. A
// negative failure count is a number that cannot be true.
func TestHandleGatewayAnalytics_FailedNeverGoesNegative(t *testing.T) {
	dump := `process_start_time_seconds 1.7851988e+09
erpc_network_request_received_total{network="evm:369",project="main"} 4
erpc_network_successful_request_total{network="evm:369",project="main",upstream="public-369-1"} 5
`
	e := readyExecutor().script("curl -s --max-time", executor.Result{Stdout: dump})
	a := trafficAPIServer(t, e)
	addGateway(t, a, "default", "local", trafficGatewayConfig())

	var body analyticsResponse
	res := a.do(t, "GET", "/api/gateways/default/analytics", nil)
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	res.Body.Close()

	if got := body.Networks[0].Failed; got != 0 {
		t.Errorf("failed: got %v, want 0 — a skewed read must not produce a count that cannot exist", got)
	}
}

// An endpoint the configuration still lists reports as configured; the join is
// over the same generated ids the renderer writes into erpc.yaml, so an
// unnamed upstream is not misfiled as a stranger.
func TestHandleGatewayAnalytics_GeneratedUpstreamIDIsRecognisedAsConfigured(t *testing.T) {
	// The SAME endpoint the configured upstream below carries, so the generated
	// id matches what the renderer would write for it.
	id := catalog.GeneratedUpstreamID(369, "https://rpc.pulsechain.com", true, 1)
	dump := "process_start_time_seconds 1.7851988e+09\n" +
		`erpc_upstream_request_total{network="evm:369",project="main",upstream="` + id + `"} 3` + "\n"

	e := readyExecutor().script("curl -s --max-time", executor.Result{Stdout: dump})
	a := trafficAPIServer(t, e)
	addGateway(t, a, "default", "local", catalog.GatewayConfig{
		Port: 4100,
		Networks: []catalog.GatewayNetwork{{ChainID: 369, Upstreams: []catalog.GatewayUpstream{
			{Endpoint: "https://rpc.pulsechain.com", Local: true},
		}}},
	})

	var body analyticsResponse
	res := a.do(t, "GET", "/api/gateways/default/analytics", nil)
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	res.Body.Close()

	if len(body.Endpoints) != 1 {
		t.Fatalf("endpoints: %+v", body.Endpoints)
	}
	if !body.Endpoints[0].Configured {
		t.Errorf("%q must read as configured: it is the id the renderer generates for an unnamed upstream", body.Endpoints[0].Upstream)
	}
}
