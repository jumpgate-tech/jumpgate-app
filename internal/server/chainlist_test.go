package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/valve-tech/valve-node-app/internal/chainlist"
	"github.com/valve-tech/valve-node-app/internal/config"
	"github.com/valve-tech/valve-node-app/internal/executor"
)

// These tests exist because this route had 19% coverage and could not have
// more: it fetched a 1.1 MB feed off the public internet and then opened a
// connection to every endpoint that feed listed. A test of that is a test that
// fails on a plane and passes when a third party's CDN happens to be up.
//
// The seam is Config.NewChainlist. The discoverer itself was ALREADY
// injectable (FeedURL, HTTPClient, ProbeTimeout); the handler simply reached
// past all of it to chainlist.New(), which is the sort of thing that only
// shows up when someone tries to write the test.

// feedStub serves a chains.json body, standing in for chainid.network.
func feedStub(t *testing.T, chains []chainlist.Chain) *httptest.Server {
	t.Helper()
	body, err := json.Marshal(chains)
	if err != nil {
		t.Fatalf("marshal feed: %v", err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// rpcStub answers eth_chainId with chainID, which is the only question the
// prober asks.
func rpcStub(t *testing.T, chainID int) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string `json:"method"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.Method != "eth_chainId" {
			http.Error(w, "unexpected method "+req.Method, http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"jsonrpc":"2.0","id":1,"result":"0x%x"}`, chainID)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func chainlistServer(t *testing.T, d *chainlist.Discoverer) *apiTestServer {
	t.Helper()
	return newAPITestServerCfg(t,
		func(config.Target) (executor.Executor, error) { return dockerExecutor("", ""), nil },
		func(c *Config) { c.NewChainlist = func() *chainlist.Discoverer { return d } },
	)
}

// refusingTransport fails every request, standing in for a machine with no
// route to anywhere.
type refusingTransport struct{}

func (refusingTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("no route to host (test transport)")
}

// The whole value of this route is that it probes: the feed is advertising,
// and roughly a third of what it lists for a popular chain is dead or serving
// a different chain than it claims. A live one, a wrong-chain one and a dead
// one must come back distinguishable, each carrying its reason.
func TestHandleChainlist_ProbedResultsCarryTheirReason(t *testing.T) {
	const chainID = 369

	live := rpcStub(t, chainID)
	wrongChain := rpcStub(t, 1)
	dead := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	t.Cleanup(dead.Close)

	feed := feedStub(t, []chainlist.Chain{
		{ChainID: chainID, Name: "PulseChain", RPC: []string{live.URL, wrongChain.URL, dead.URL}},
	})

	d := chainlist.New()
	d.FeedURL = feed.URL
	d.ProbeTimeout = 3 * time.Second
	d.ProbeWS = false

	a := chainlistServer(t, d)
	res := a.do(t, "GET", "/api/chainlist/369", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("got %d, want 200", res.StatusCode)
	}
	body := decode[chainlistResponse](t, res)

	if body.ChainID != chainID {
		t.Errorf("chainId: got %d", body.ChainID)
	}
	if body.Source != "feed" {
		t.Errorf("source: got %q, want feed", body.Source)
	}
	if body.Live != 1 {
		t.Errorf("live: got %d, want 1 — only one of the three actually serves this chain", body.Live)
	}
	if len(body.Endpoints) != 3 {
		t.Fatalf("endpoints: %+v, want all three, rejected ones included", body.Endpoints)
	}

	byURL := map[string]chainlistEndpoint{}
	for _, e := range body.Endpoints {
		byURL[e.URL] = e
	}
	if got := byURL[live.URL]; got.Status != "live" {
		t.Errorf("the live endpoint: %+v", got)
	}
	// The useful detail on a rejection is what the endpoint really is: "that
	// one is actually chain 1" beats "rejected".
	if got := byURL[wrongChain.URL]; got.Status == "live" || got.ChainID != 1 {
		t.Errorf("the wrong-chain endpoint must be rejected and say which chain it really serves: %+v", got)
	}
	if got := byURL[dead.URL]; got.Status == "live" || got.Reason == "" {
		t.Errorf("the dead endpoint must carry a reason: %+v", got)
	}
}

// A HEALTHY feed that simply does not list the chain takes the same path as an
// unreachable one, which is chainlist's documented intent: from the operator's
// seat "chainlist doesn't list your chain" and "chainlist is down" have the
// same remedy. With no vendored list either, that is genuinely nothing to
// offer — 502, naming the chain.
func TestHandleChainlist_ChainMissingFromAHealthyFeedIs502(t *testing.T) {
	feed := feedStub(t, []chainlist.Chain{{ChainID: 1, Name: "Ethereum", RPC: []string{"https://unused.example"}}})
	d := chainlist.New()
	d.FeedURL = feed.URL
	d.ProbeWS = false

	a := chainlistServer(t, d)
	res := a.do(t, "GET", "/api/chainlist/999999", nil)
	if res.StatusCode != http.StatusBadGateway {
		t.Fatalf("got %d, want 502", res.StatusCode)
	}
	body := decode[errorDetail](t, res)
	if !strings.Contains(body.Error, "999999") {
		t.Errorf("the message must name the chain nobody could offer anything for: %q", body.Error)
	}
}

// The feed being down is upstream of this app, so it is a 502 rather than a
// 500 — but ONLY when there is no vendored snapshot to fall back on, which is
// what makes it "genuinely nothing to offer" rather than "degraded".
func TestHandleChainlist_NoFeedAndNoSnapshotIs502(t *testing.T) {
	down := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "feed is down", http.StatusServiceUnavailable)
	}))
	t.Cleanup(down.Close)

	d := chainlist.New()
	d.FeedURL = down.URL
	d.ProbeWS = false

	a := chainlistServer(t, d)
	res := a.do(t, "GET", "/api/chainlist/424242", nil)
	if res.StatusCode != http.StatusBadGateway {
		t.Fatalf("got %d, want 502 — the failure is upstream of this app, not in the request", res.StatusCode)
	}
}

// A chain valve ships a vendored snapshot for still answers when the feed is
// unreachable, and says which it is — a snapshot is not a failure, but the
// operator should know they are looking at one, so the fetch error travels
// with it instead of being swallowed.
func TestHandleChainlist_FallsBackToTheSnapshotAndSaysSo(t *testing.T) {
	// Every HTTP call fails, feed and probes alike. That is deliberate: the
	// vendored list for chain 369 holds REAL public endpoints, so a client
	// that could reach the internet would have this test probing
	// rpc.pulsechain.com — a test that passes or fails on someone else's
	// uptime, which is the thing this whole seam exists to avoid.
	d := chainlist.New()
	d.HTTPClient = &http.Client{Transport: refusingTransport{}}
	d.ProbeWS = false
	d.ProbeTimeout = 500 * time.Millisecond

	a := chainlistServer(t, d)
	res := a.do(t, "GET", "/api/chainlist/369", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("got %d, want 200 — there is a vendored list for this chain", res.StatusCode)
	}
	body := decode[chainlistResponse](t, res)
	if body.Source != "vendored" {
		t.Errorf("source: got %q, want vendored", body.Source)
	}
	if body.FetchError == "" {
		t.Error("the fetch error must travel with the fallback, or the operator cannot tell a snapshot from a fresh read")
	}
}

func TestHandleChainlist_RejectsSomethingThatIsNotAChainID(t *testing.T) {
	a := chainlistServer(t, chainlist.New())
	for _, bad := range []string{"abc", "0", "-1"} {
		res := a.do(t, "GET", "/api/chainlist/"+bad, nil)
		res.Body.Close()
		if res.StatusCode != http.StatusBadRequest {
			t.Errorf("chain id %q: got %d, want 400", bad, res.StatusCode)
		}
	}
}

// Latency is the only ordering signal an operator has between several live
// endpoints, so it has to survive onto the wire in milliseconds.
func TestHandleChainlist_LatencyReachesTheWire(t *testing.T) {
	const chainID = 943
	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(15 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"jsonrpc":"2.0","id":1,"result":"0x%x"}`, chainID)
	}))
	t.Cleanup(slow.Close)

	feed := feedStub(t, []chainlist.Chain{{ChainID: chainID, Name: "PulseChain Testnet", RPC: []string{slow.URL}}})
	d := chainlist.New()
	d.FeedURL = feed.URL
	d.ProbeWS = false
	d.ProbeTimeout = 3 * time.Second

	a := chainlistServer(t, d)
	body := decode[chainlistResponse](t, a.do(t, "GET", "/api/chainlist/943", nil))
	if len(body.Endpoints) != 1 {
		t.Fatalf("endpoints: %+v", body.Endpoints)
	}
	if body.Endpoints[0].LatencyMS < 10 {
		t.Errorf("latencyMs: got %d, want at least the 15ms the stub sleeps", body.Endpoints[0].LatencyMS)
	}
	if !strings.HasPrefix(body.Endpoints[0].Kind, "http") {
		t.Errorf("kind: got %q", body.Endpoints[0].Kind)
	}
}
