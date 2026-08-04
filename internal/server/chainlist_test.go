package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/valve-tech/valve-node-app/internal/catalog"
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

// The full-catalogue route (GET /api/chainlist) backs the search picker. It
// enumerates the feed's id+name pairs, drops junk rows, and sorts by id.
func TestHandleChainlistAll_ReturnsSortedIdNameCatalogue(t *testing.T) {
	feed := feedStub(t, []chainlist.Chain{
		{ChainID: 369, Name: "PulseChain", RPC: []string{"https://rpc.pulsechain.com"}},
		{ChainID: 1, Name: "Ethereum Mainnet", RPC: []string{"https://eth.example"}},
		{ChainID: 0, Name: "bogus zero id", RPC: nil}, // dropped: id <= 0
		{ChainID: 8453, Name: "   ", RPC: nil},        // dropped: blank name
	})
	d := chainlist.New()
	d.FeedURL = feed.URL

	a := chainlistServer(t, d)
	res := a.do(t, "GET", "/api/chainlist", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("got %d, want 200", res.StatusCode)
	}
	body := decode[chainlistAllResponse](t, res)
	if body.Stale {
		t.Errorf("a fresh fetch must not be flagged stale")
	}
	if len(body.Chains) != 2 {
		t.Fatalf("got %d chains, want 2 (zero-id and blank-name dropped): %+v", len(body.Chains), body.Chains)
	}
	if body.Chains[0].ChainID != 1 || body.Chains[1].ChainID != 369 {
		t.Errorf("not sorted by id ascending: %+v", body.Chains)
	}
	if body.Chains[0].Name != "Ethereum Mainnet" {
		t.Errorf("name: got %q, want %q", body.Chains[0].Name, "Ethereum Mainnet")
	}
}

// A feed failure with a warm cache is not a failure: the last good catalogue
// stands in, flagged stale, because the picker also carries viem's curated set
// and an empty long-tail is worse than a slightly old one.
func TestHandleChainlistAll_FallsBackToCacheWhenFeedFails(t *testing.T) {
	feed := feedStub(t, []chainlist.Chain{{ChainID: 1, Name: "Ethereum", RPC: nil}})
	d := chainlist.New()
	d.FeedURL = feed.URL

	a := chainlistServer(t, d)
	if res := a.do(t, "GET", "/api/chainlist", nil); res.StatusCode != http.StatusOK {
		t.Fatalf("warm-up got %d, want 200", res.StatusCode)
	}

	// Break the feed and force past the cache: the cached answer must stand in.
	d.HTTPClient = &http.Client{Transport: refusingTransport{}}
	res := a.do(t, "GET", "/api/chainlist?refresh=1", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("stale fallback got %d, want 200", res.StatusCode)
	}
	body := decode[chainlistAllResponse](t, res)
	if !body.Stale {
		t.Errorf("expected stale=true when the feed is unreachable but a cache exists")
	}
	if len(body.Chains) != 1 || body.Chains[0].ChainID != 1 {
		t.Errorf("cached chains not returned: %+v", body.Chains)
	}
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

// ---------------------------------------------------------------------
// provider keys: resolved for probing, redacted for the wire
// ---------------------------------------------------------------------

// A stored key turns a provider slot from a rejection into a real candidate —
// and the key must NOT come back with it.
//
// This is the whole hazard of filling templates in: chainlist resolves
// ${INFURA_API_KEY} by splicing the key into the URL PATH, and this route
// serialises endpoint URLs straight to the browser. Sending the resolved URL
// would put the operator's key in the page, in every proxy log between here and
// it, and in any screenshot of the picker. What goes out is the placeholder
// form, which is still a precise reference to one endpoint — the client posts
// it back as-is and the save path fills it in again.
func TestHandleChainlist_AKeyResolvesTheSlotButNeverReachesTheWire(t *testing.T) {
	const (
		chainID = 943
		secret  = "sk_live_this_must_not_be_serialised"
	)

	upstream := rpcStub(t, chainID)
	// The key is a path segment, exactly as the real provider URLs put it.
	templated := upstream.URL + "/v3/${TEST_API_KEY}"

	feed := feedStub(t, []chainlist.Chain{{ChainID: chainID, Name: "PulseChain Testnet", RPC: []string{templated}}})
	d := chainlist.New()
	d.FeedURL = feed.URL
	d.ProbeWS = false
	d.ProbeTimeout = 3 * time.Second

	a := chainlistServer(t, d)
	seedProviderKey(t, "TEST_API_KEY", secret)

	res := a.do(t, "GET", "/api/chainlist/943", nil)
	defer res.Body.Close()
	raw, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if strings.Contains(string(raw), secret) {
		t.Fatalf("the API key was serialised to the browser: %s", raw)
	}

	var body chainlistResponse
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Live != 1 || len(body.Endpoints) != 1 {
		t.Fatalf("the slot must resolve and probe live, not be rejected: %+v", body)
	}
	if got := body.Endpoints[0].URL; got != templated {
		t.Errorf("url = %q, want the placeholder form %q — the client posts this back to add it", got, templated)
	}
}

// Without the key the same entry is a rejection that NAMES what is missing, so
// the operator has a next step rather than a dead end.
func TestHandleChainlist_AnUnknownSlotIsRejectedByName(t *testing.T) {
	const chainID = 943
	feed := feedStub(t, []chainlist.Chain{
		{ChainID: chainID, Name: "PulseChain Testnet", RPC: []string{"https://rpc.example.com/v3/${TEST_API_KEY}"}},
	})
	d := chainlist.New()
	d.FeedURL = feed.URL
	d.ProbeWS = false

	a := chainlistServer(t, d)

	body := decode[chainlistResponse](t, a.do(t, "GET", "/api/chainlist/943", nil))
	if len(body.Endpoints) != 1 {
		t.Fatalf("endpoints: %+v", body.Endpoints)
	}
	if body.Endpoints[0].Status != "rejected" {
		t.Errorf("a slot with no key must not be offered: %+v", body.Endpoints[0])
	}
	if !strings.Contains(body.Endpoints[0].Reason, "TEST_API_KEY") {
		t.Errorf("the reason must name the key to go get: %q", body.Endpoints[0].Reason)
	}
}

// valve's own slot resolves on a box where nothing has been set up at all,
// because VALVE_API_KEY falls back to the shared demo key — the same zero-setup
// guarantee catalog.KnownSet makes. A default that only applied once an
// operator had visited Settings would not be a default.
//
// The demo key is the one value redactKeys exempts by name, and is deliberately
// not redacted: it is a published constant shipped in this binary, not a secret,
// and redaction exists for secrets. The endpoint still round-trips — the resolved URL the
// client posts back has no slot left to fill, so the save path stores it as-is.
func TestHandleChainlist_ValveSlotResolvesWithNoSetupAtAll(t *testing.T) {
	const chainID = 369

	upstream := rpcStub(t, chainID)
	templated := upstream.URL + "/rpc/${VALVE_API_KEY}/evm/369"

	feed := feedStub(t, []chainlist.Chain{{ChainID: chainID, Name: "PulseChain", RPC: []string{templated}}})
	d := chainlist.New()
	d.FeedURL = feed.URL
	d.ProbeWS = false
	d.ProbeTimeout = 3 * time.Second

	a := chainlistServer(t, d)

	body := decode[chainlistResponse](t, a.do(t, "GET", "/api/chainlist/369", nil))
	if body.Live != 1 {
		t.Fatalf("valve's slot must resolve with no key stored: %+v", body.Endpoints)
	}
	if !strings.Contains(body.Endpoints[0].URL, catalog.DefaultValveKey) {
		t.Errorf("url = %q, want the demo key filled in", body.Endpoints[0].URL)
	}
}

// An operator's OWN valve key is long, so it is redacted like any other — the
// zero-setup default being public does not make a real key public.
func TestHandleChainlist_AnOperatorsOwnValveKeyIsRedacted(t *testing.T) {
	const (
		chainID = 369
		mine    = "vk_ThisIsAnOperatorsRealValveKey"
	)

	upstream := rpcStub(t, chainID)
	templated := upstream.URL + "/rpc/${VALVE_API_KEY}/evm/369"

	feed := feedStub(t, []chainlist.Chain{{ChainID: chainID, Name: "PulseChain", RPC: []string{templated}}})
	d := chainlist.New()
	d.FeedURL = feed.URL
	d.ProbeWS = false
	d.ProbeTimeout = 3 * time.Second

	a := chainlistServer(t, d)
	seedProviderKey(t, config.ValveKeyPlaceholder, mine)

	res := a.do(t, "GET", "/api/chainlist/369", nil)
	defer res.Body.Close()
	raw, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if strings.Contains(string(raw), mine) {
		t.Fatalf("the operator's own valve key was serialised to the browser: %s", raw)
	}
}

// The same, end to end, for a key SHORT enough that the old length gate waved
// it through: it went to the browser inside a discovery URL. Length is not what
// makes a key public.
func TestHandleChainlist_AShortOperatorKeyIsRedactedToo(t *testing.T) {
	const (
		chainID = 369
		mine    = "vk_x9"
	)

	upstream := rpcStub(t, chainID)
	templated := upstream.URL + "/rpc/${VALVE_API_KEY}/evm/369"

	feed := feedStub(t, []chainlist.Chain{{ChainID: chainID, Name: "PulseChain", RPC: []string{templated}}})
	d := chainlist.New()
	d.FeedURL = feed.URL
	d.ProbeWS = false
	d.ProbeTimeout = 3 * time.Second

	a := chainlistServer(t, d)
	seedProviderKey(t, config.ValveKeyPlaceholder, mine)

	res := a.do(t, "GET", "/api/chainlist/369", nil)
	defer res.Body.Close()
	raw, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if strings.Contains(string(raw), "/rpc/"+mine+"/") {
		t.Fatalf("a short operator key was serialised to the browser: %s", raw)
	}
}

// Redaction is a carve-out for ONE published value, not a length test. A short
// operator key is a bad key, not a public one, and it must not reach the browser
// just because it is short — that was the bug in gating on length.
func TestRedactKeys_AShortOperatorKeyIsStillRedacted(t *testing.T) {
	keys := map[string]string{
		config.ValveKeyPlaceholder: "vk_x9",
		"INFURA_API_KEY":           "sk_infura_a_real_length_key",
	}

	got := redactKeys("https://one.valve.city/rpc/vk_x9/evm/369", keys)

	if want := "https://one.valve.city/rpc/${VALVE_API_KEY}/evm/369"; got != want {
		t.Errorf("redactKeys = %q, want %q — a short key is not a public one", got, want)
	}
	// Longest-first still holds, so a long key redacts as it always did.
	if got, want := redactKeys("https://mainnet.infura.io/v3/sk_infura_a_real_length_key", keys),
		"https://mainnet.infura.io/v3/${INFURA_API_KEY}"; got != want {
		t.Errorf("redactKeys = %q, want %q", got, want)
	}
}

// The published default is the one value redaction leaves alone: it ships in
// this binary, so hiding it buys nothing, and the URL an operator sees with
// nothing configured should be the URL this process actually dials.
func TestRedactKeys_TheDefaultValveKeyIsLeftAlone(t *testing.T) {
	keys := map[string]string{config.ValveKeyPlaceholder: catalog.DefaultValveKey}

	url := "https://one.valve.city/rpc/" + catalog.DefaultValveKey + "/evm/369"
	if got := redactKeys(url, keys); got != url {
		t.Errorf("redactKeys = %q, want the published default left in place", got)
	}
}

// seedProviderKey stores a provider key the way the app itself stores one,
// through config.Load()/cfg.Save() under the test's isolated HOME.
func seedProviderKey(t *testing.T, name, value string) {
	t.Helper()
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.ProviderKeys == nil {
		cfg.ProviderKeys = map[string]string{}
	}
	cfg.ProviderKeys[name] = value
	if err := cfg.Save(); err != nil {
		t.Fatalf("save config: %v", err)
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
