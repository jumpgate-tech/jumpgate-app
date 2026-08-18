package relay

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

// Health filters the category x arch x chain matrix. Each path level pins one
// more dimension, and the answer must carry real state — a rollup that always
// says "ok" is worse than none, because a monitor would trust it.

func healthHandler(t *testing.T, chains []int, beacon *BeaconPool, caller RPCCaller) *Handler {
	t.Helper()
	var got capturedRequest
	up := stubUpstream(t, &got)
	cfg := Config{
		Auth:      staticAuth{rec: enabledKey()},
		ProjectID: "main",
		ERPC:      up,
		Caller:    caller,
		Health:    NewHealthProbe(caller, beacon, chains),
	}
	if beacon != nil {
		cfg.Beacon = beacon.Next
	}
	h, err := NewHandler(cfg)
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}
	return h
}

func getJSON(t *testing.T, h http.Handler, path string) map[string]any {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("%s: status = %d (body %q)", path, res.Code, res.Body.String())
	}
	var out map[string]any
	if err := json.Unmarshal(res.Body.Bytes(), &out); err != nil {
		t.Fatalf("%s: decode %q: %v", path, res.Body.String(), err)
	}
	return out
}

// The cell level reports the chain's real head, read from the upstream rather
// than assumed.
func TestHealthCellReportsTheRealHead(t *testing.T) {
	caller := newScriptedCaller()
	for i := 0; i < 42; i++ {
		caller.advance()
	}
	h := healthHandler(t, []int{369}, nil, caller)

	out := getJSON(t, h, "/health/jg_k/evm/369")
	rpc, ok := out["rpc"].(map[string]any)
	if !ok {
		t.Fatalf("rpc = %v, want an object", out["rpc"])
	}
	if rpc["head"] != "0x2a" {
		t.Errorf("head = %v, want 0x2a (42)", rpc["head"])
	}
	if rpc["ok"] != true {
		t.Errorf("ok = %v, want true", rpc["ok"])
	}
}

// An upstream that cannot answer is reported as not ok, not omitted. A missing
// field would read to a monitor as "nothing wrong here".
func TestHealthReportsAnUnreachableChainAsNotOK(t *testing.T) {
	h := healthHandler(t, []int{369}, nil, callerFunc(func(context.Context, int, []byte) ([]byte, error) {
		return nil, context.DeadlineExceeded
	}))

	out := getJSON(t, h, "/health/jg_k/evm/369")
	rpc := out["rpc"].(map[string]any)
	if rpc["ok"] != false {
		t.Errorf("ok = %v, want false for an unreachable chain", rpc["ok"])
	}
}

// A cell carries BOTH categories, so one request answers "is chain 369 well"
// rather than two.
func TestHealthCellCarriesRPCAndBeacon(t *testing.T) {
	caller := newScriptedCaller()
	caller.advance()
	node := newBeaconNode(t, http.StatusOK)
	pool := NewBeaconPool(map[int][]*url.URL{369: {node.url}})
	t.Cleanup(pool.Stop)
	pool.Probe(context.Background())

	h := healthHandler(t, []int{369}, pool, caller)
	out := getJSON(t, h, "/health/jg_k/evm/369")

	if _, ok := out["rpc"]; !ok {
		t.Error("no rpc object in a cell response")
	}
	beacon, ok := out["beacon"].(map[string]any)
	if !ok {
		t.Fatalf("beacon = %v, want an object", out["beacon"])
	}
	if beacon["ok"] != true {
		t.Errorf("beacon ok = %v, want true", beacon["ok"])
	}
	if beacon["usable"] != float64(1) {
		t.Errorf("beacon usable = %v, want 1", beacon["usable"])
	}
}

// Pinning only the arch rolls up every chain under it.
func TestHealthArchLevelRollsUpEveryChain(t *testing.T) {
	caller := newScriptedCaller()
	caller.advance()
	h := healthHandler(t, []int{1, 369}, nil, caller)

	out := getJSON(t, h, "/health/jg_k/evm")
	chains, ok := out["chains"].(map[string]any)
	if !ok {
		t.Fatalf("chains = %v, want an object", out["chains"])
	}
	if len(chains) != 2 {
		t.Errorf("rolled up %d chains, want 2: %v", len(chains), chains)
	}
	if _, ok := chains["369"]; !ok {
		t.Errorf("chain 369 missing from the rollup: %v", chains)
	}
}

// Pinning the beacon CATEGORY reports only chains that actually have one.
func TestHealthBeaconCategoryListsOnlyChainsWithABeacon(t *testing.T) {
	caller := newScriptedCaller()
	caller.advance()
	node := newBeaconNode(t, http.StatusOK)
	pool := NewBeaconPool(map[int][]*url.URL{369: {node.url}})
	t.Cleanup(pool.Stop)
	pool.Probe(context.Background())

	h := healthHandler(t, []int{1, 369}, pool, caller)
	out := getJSON(t, h, "/health/jg_k/beacon")

	chains, ok := out["chains"].(map[string]any)
	if !ok {
		t.Fatalf("chains = %v, want an object", out["chains"])
	}
	if _, ok := chains["369"]; !ok {
		t.Errorf("chain 369 has a beacon and is missing: %v", chains)
	}
	if _, ok := chains["1"]; ok {
		t.Errorf("chain 1 has no beacon and must not appear: %v", chains)
	}
}

// The unkeyed level stays shallow no matter what the probe knows. It must never
// tell a scanner which chains the operator runs or when one is lagging.
func TestHealthLivenessStaysShallowWithRealData(t *testing.T) {
	caller := newScriptedCaller()
	caller.advance()
	h := healthHandler(t, []int{1, 369}, nil, caller)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)

	body := res.Body.String()
	for _, leak := range []string{"369", "head", "chains", "upstream"} {
		if contains(body, leak) {
			t.Errorf("unkeyed liveness leaked %q: %s", leak, body)
		}
	}
}

// With no probe configured the rollup still answers rather than crashing, so a
// gateway that never wired one keeps a working endpoint.
func TestHealthWithoutAProbeStillAnswers(t *testing.T) {
	var got capturedRequest
	up := stubUpstream(t, &got)
	h, err := NewHandler(Config{Auth: staticAuth{rec: enabledKey()}, ProjectID: "main", ERPC: up})
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/health/jg_k/evm/369", nil)
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", res.Code)
	}
}
