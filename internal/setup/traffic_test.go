package setup

import (
	"context"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/valve-tech/valve-node-app/internal/catalog"
	"github.com/valve-tech/valve-node-app/internal/executor"
	"github.com/valve-tech/valve-node-app/internal/metrics"
)

// trafficGateway is a one-chain gateway with an OPERATOR metrics port (9101)
// that is deliberately different from both eRPC's default (4001) and the
// fixed in-container port ops.GatewayContainerConfig rewrites to. Any test
// here that scraped the wrong one of those would otherwise pass by accident.
func trafficGateway() catalog.GatewayConfig {
	return catalog.GatewayConfig{
		Port:        4100,
		MetricsPort: 9101,
		Networks: []catalog.GatewayNetwork{
			{ChainID: 369, Upstreams: []catalog.GatewayUpstream{
				{ID: "local-node", Endpoint: "http://127.0.0.1:8545", Local: true},
				{ID: "fallback-1", Endpoint: "https://rpc.example.com"},
			}},
		},
	}
}

// erpcFixture is the captured real eRPC dump internal/metrics' own tests
// parse against (TestFromSamples_RealFixture). Reusing it here means the
// "parses into networks" case below exercises the exact same body a real
// gateway produced, rather than a hand-written approximation of one.
func erpcFixture(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile("../metrics/testdata/erpc.txt")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	return string(b)
}

// callsContainingCurl finds the curl invocation among a fake executor's call
// log, or "" if none ran.
func callsContainingCurl(calls []string) string {
	for _, c := range calls {
		if strings.Contains(c, "curl") {
			return c
		}
	}
	return ""
}

// ---------------------------------------------------------------------
// ReadGatewayTraffic
// ---------------------------------------------------------------------

func TestReadGatewayTraffic_MetricsOffCostsNoRoundTrip(t *testing.T) {
	g := trafficGateway()
	g.MetricsOff = true
	e := newFakeExecutor()

	_, err := ReadGatewayTraffic(context.Background(), e, g)
	if !errors.Is(err, ErrMetricsOff) {
		t.Fatalf("got %v, want it to wrap ErrMetricsOff", err)
	}
	// The whole point of checking the setting before doing anything: a
	// disabled gateway must cost no round trip, not merely fail one silently.
	if calls := e.callLog(); len(calls) != 0 {
		t.Fatalf("a disabled gateway must never be dialed, but the executor saw: %#v", calls)
	}
}

func TestReadGatewayTraffic_ScrapesTheOperatorsMetricsPortOnLoopback(t *testing.T) {
	g := trafficGateway()
	e := newFakeExecutor().script("curl -s --max-time", executor.Result{Stdout: erpcFixture(t)})

	if _, err := ReadGatewayTraffic(context.Background(), e, g); err != nil {
		t.Fatalf("ReadGatewayTraffic: %v", err)
	}

	curl := callsContainingCurl(e.callLog())
	if curl == "" {
		t.Fatalf("no curl ran: %#v", e.callLog())
	}
	// This is the bug the code's own comment warns about: scraping
	// ops.GatewayContainerConfig's rewritten (fixed, in-container) port would
	// dial a port the host has nothing listening on. The OPERATOR's port
	// (9101, from MetricsPort) is the only one this process can reach.
	if !strings.Contains(curl, "127.0.0.1:9101") {
		t.Fatalf("want the operator's own metrics port (9101) scraped on loopback, got %q", curl)
	}
	if strings.Contains(curl, ":4001") {
		t.Fatalf("want the operator's port, not eRPC's bare default (4001): %q", curl)
	}
}

func TestReadGatewayTraffic_NonZeroExitNamesTheURLAndCarriesStderr(t *testing.T) {
	g := trafficGateway()
	e := newFakeExecutor().script("curl -s --max-time", executor.Result{
		ExitCode: 7,
		Stderr:   "Failed to connect to 127.0.0.1 port 9101: Connection refused",
	})

	_, err := ReadGatewayTraffic(context.Background(), e, g)
	if err == nil {
		t.Fatal("want an error when curl exits non-zero")
	}
	if !strings.Contains(err.Error(), "127.0.0.1:9101") {
		t.Fatalf("want the scraped URL named in the error, got %v", err)
	}
	if !strings.Contains(err.Error(), "Connection refused") {
		t.Fatalf("want curl's own stderr surfaced, got %v", err)
	}
}

// A zero exit with an empty body is what a listener that accepted the
// connection and served nothing looks like — a different fact from "nobody
// has called this gateway", and the message must say so rather than handing
// an empty string to the parser and reporting a healthy-looking zero reading.
func TestReadGatewayTraffic_ZeroExitEmptyBodyIsAnErrorNotAnEmptyReading(t *testing.T) {
	g := trafficGateway()
	e := newFakeExecutor().script("curl -s --max-time", executor.Result{ExitCode: 0, Stdout: ""})

	_, err := ReadGatewayTraffic(context.Background(), e, g)
	if err == nil {
		t.Fatal("want an error, not a silent empty Traffic{} — that would render as a healthy gateway nobody has called")
	}
	if !strings.Contains(err.Error(), "listening") || !strings.Contains(err.Error(), "not serving Prometheus") {
		t.Fatalf("want the message to distinguish 'something is listening but not serving Prometheus' from an actually-empty reading, got %v", err)
	}
}

func TestReadGatewayTraffic_NonPrometheusBodySurfacesTheParseError(t *testing.T) {
	g := trafficGateway()
	e := newFakeExecutor().script("curl -s --max-time", executor.Result{Stdout: "<html>not metrics</html>\n"})

	_, err := ReadGatewayTraffic(context.Background(), e, g)
	if err == nil {
		t.Fatal("want an error when the body is not Prometheus text")
	}
	if !strings.Contains(err.Error(), "did not answer with Prometheus text") {
		t.Fatalf("want the parse failure surfaced with context, got %v", err)
	}
}

// The realistic case: the real captured dump parses and produces the one
// network it genuinely carries client-facing traffic for.
func TestReadGatewayTraffic_RealFixtureParsesIntoNetworks(t *testing.T) {
	g := trafficGateway() // ProjectID "" -> "main", which is what the fixture tags everything with.
	e := newFakeExecutor().script("curl -s --max-time", executor.Result{Stdout: erpcFixture(t)})

	traffic, err := ReadGatewayTraffic(context.Background(), e, g)
	if err != nil {
		t.Fatalf("ReadGatewayTraffic: %v", err)
	}
	if len(traffic.Networks) != 1 {
		t.Fatalf("got %d networks, want 1: %+v", len(traffic.Networks), traffic.Networks)
	}
	nt := traffic.Networks[0]
	if nt.ChainID != 369 || nt.Received != 5 {
		t.Fatalf("got chain %d received %v, want chain 369 received 5: %+v", nt.ChainID, nt.Received, nt)
	}
	if len(nt.Upstreams) != 1 || nt.Upstreams[0].Upstream != "public-369-1" || nt.Upstreams[0].Succeeded != 5 {
		t.Fatalf("got %+v, want a single upstream public-369-1 with 5 succeeded", nt.Upstreams)
	}
}

// The project filter is not optional plumbing: without it, one gateway would
// report another project's traffic as its own. Scrape the same fixture with a
// gateway whose ProjectID the fixture never mentions, and prove nothing comes
// back rather than everything.
func TestReadGatewayTraffic_FiltersByTheGatewaysOwnProject(t *testing.T) {
	g := trafficGateway()
	g.ProjectID = "some-other-project"
	e := newFakeExecutor().script("curl -s --max-time", executor.Result{Stdout: erpcFixture(t)})

	traffic, err := ReadGatewayTraffic(context.Background(), e, g)
	if err != nil {
		t.Fatalf("ReadGatewayTraffic: %v", err)
	}
	if len(traffic.Networks) != 0 {
		t.Fatalf("got %d networks for a project the fixture does not contain, want 0 (the fixture's every sample is project=main): %+v", len(traffic.Networks), traffic.Networks)
	}
}

// ---------------------------------------------------------------------
// IntentsFor
// ---------------------------------------------------------------------

func TestIntentsFor_ExplicitUpstreamIDsPassThrough(t *testing.T) {
	n := catalog.GatewayNetwork{ChainID: 369, Upstreams: []catalog.GatewayUpstream{
		{ID: "my-node", Local: true},
		{ID: "my-fallback"},
	}}

	got := IntentsFor(n)
	want := []metrics.Intent{
		{Upstream: "my-node", Local: true},
		{Upstream: "my-fallback", Local: false},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

// An upstream with no ID must get EXACTLY catalog.GeneratedUpstreamID's
// output, because that generated string is the very id RenderGatewayConfig
// writes into erpc.yaml — and therefore the id eRPC labels its counters with.
// A mismatch here would not error; it would silently attribute every request
// to an upstream nobody can see.
func TestIntentsFor_EmptyIDGetsExactlyTheGeneratedID(t *testing.T) {
	n := catalog.GatewayNetwork{ChainID: 369, Upstreams: []catalog.GatewayUpstream{
		{Local: true, Endpoint: "http://host.docker.internal:8545"}, // position 1, local
		{Endpoint: "https://eth.drpc.org"},                          // position 2, public
	}}

	got := IntentsFor(n)
	if len(got) != 2 {
		t.Fatalf("got %d intents, want 2", len(got))
	}

	wantLocal := catalog.GeneratedUpstreamID(369, "http://host.docker.internal:8545", true, 1)
	if got[0].Upstream != wantLocal {
		t.Fatalf("local upstream id: got %q, want %q (catalog.GeneratedUpstreamID's own output)", got[0].Upstream, wantLocal)
	}
	if got[0].Upstream != "369-local-http-1" {
		t.Fatalf("local upstream id: got %q, want the literal id 369-local-http-1", got[0].Upstream)
	}

	wantPublic := catalog.GeneratedUpstreamID(369, "https://eth.drpc.org", false, 2)
	if got[1].Upstream != wantPublic {
		t.Fatalf("public upstream id: got %q, want %q", got[1].Upstream, wantPublic)
	}
	if got[1].Upstream != "369-drpc-http-2" {
		t.Fatalf("public upstream id: got %q, want the literal id 369-drpc-http-2", got[1].Upstream)
	}
}

// Local must be carried through faithfully for a mix of local and fallback
// upstreams — Shares uses it downstream to decide which tier an upstream
// belongs to, so a flipped bit here would misclassify a fallback as preferred
// or vice versa.
func TestIntentsFor_CarriesLocalThroughForAMixOfBoth(t *testing.T) {
	n := catalog.GatewayNetwork{ChainID: 943, Upstreams: []catalog.GatewayUpstream{
		{ID: "a", Local: true},
		{ID: "b", Local: false},
		{ID: "c", Local: true},
	}}

	got := IntentsFor(n)
	want := []bool{true, false, true}
	for i, wantLocal := range want {
		if got[i].Local != wantLocal {
			t.Errorf("upstream %d (%s): Local = %v, want %v", i, got[i].Upstream, got[i].Local, wantLocal)
		}
	}
}
