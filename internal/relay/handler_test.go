package relay

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// The handler is where the slice earns its name. Every test here drives a real
// http.Handler against a real stub upstream and asserts on what the upstream
// actually received, because the security property is about the forwarded
// request and not about the relay's intentions.

// capturedRequest is what the stub upstream saw.
type capturedRequest struct {
	path    string
	query   string
	headers http.Header
	body    string
	hits    int
}

// stubUpstream is a real HTTP server that records one request and answers with
// a fixed body.
func stubUpstream(t *testing.T, got *capturedRequest) *url.URL {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		got.path = r.URL.Path
		got.query = r.URL.RawQuery
		got.headers = r.Header.Clone()
		got.body = string(b)
		got.hits++
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"jsonrpc":"2.0","id":1,"result":"0x1"}`)
	}))
	t.Cleanup(srv.Close)
	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatalf("parse stub url: %v", err)
	}
	return u
}

// staticAuth answers with one record for any key it knows.
type staticAuth struct {
	rec KeyRecord
	err error
}

func (s staticAuth) Authenticate(context.Context, string) (KeyRecord, error) {
	return s.rec, s.err
}

func enabledKey() KeyRecord { return KeyRecord{ID: "k1", Enabled: true, AllowTrace: true} }

func newTestHandler(t *testing.T, auth Authenticator, erpc *url.URL, beacon *url.URL) *Handler {
	t.Helper()
	cfg := Config{
		Auth:      auth,
		ProjectID: "main",
		ERPC:      erpc,
	}
	if beacon != nil {
		cfg.Beacon = func(int) (*url.URL, bool) { return beacon, true }
	}
	h, err := NewHandler(cfg)
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}
	return h
}

func post(t *testing.T, h http.Handler, path, body string, hdr http.Header) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	for k, vs := range hdr {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

const blockNumber = `{"jsonrpc":"2.0","id":1,"method":"eth_blockNumber","params":[]}`

// THE load-bearing test. The key must not survive into the forwarded request in
// any form — not the path, not a query parameter, not a header. Everything else
// in this slice is in service of this assertion.
func TestHandlerStripsTheKeyEverywhere(t *testing.T) {
	var got capturedRequest
	up := stubUpstream(t, &got)
	h := newTestHandler(t, staticAuth{rec: enabledKey()}, up, nil)

	hdr := http.Header{}
	hdr.Set("Authorization", "Bearer jg_secretkey")
	res := post(t, h, "/rpc/jg_secretkey/evm/369", blockNumber, hdr)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %q)", res.Code, res.Body.String())
	}
	if got.hits != 1 {
		t.Fatalf("upstream hits = %d, want 1", got.hits)
	}
	if got.path != "/main/evm/369" {
		t.Errorf("upstream path = %q, want /main/evm/369", got.path)
	}
	if strings.Contains(got.path, "jg_secretkey") {
		t.Errorf("path %q still carries the key", got.path)
	}
	if strings.Contains(got.query, "jg_secretkey") {
		t.Errorf("query %q still carries the key", got.query)
	}
	for name, values := range got.headers {
		for _, v := range values {
			if strings.Contains(v, "jg_secretkey") {
				t.Errorf("header %s: %q still carries the key", name, v)
			}
		}
	}
}

// A caller's body reaches the upstream unchanged. The relay reads the method to
// apply policy; it must not rewrite the call.
func TestHandlerForwardsTheBodyIntact(t *testing.T) {
	var got capturedRequest
	up := stubUpstream(t, &got)
	h := newTestHandler(t, staticAuth{rec: enabledKey()}, up, nil)

	batch := `[{"jsonrpc":"2.0","id":1,"method":"eth_blockNumber"},{"jsonrpc":"2.0","id":2,"method":"eth_chainId"}]`
	if res := post(t, h, "/rpc/jg_k/evm/369", batch, nil); res.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", res.Code)
	}
	if got.body != batch {
		t.Errorf("upstream body = %q, want it unchanged", got.body)
	}
}

// gzip must not break the relay. eRPC once answered 500 on an upgrade whenever
// a caller sent Accept-Encoding: gzip, and the relay is now a second proxy that
// could reintroduce the condition. So gzip is a case, not a footnote.
func TestHandlerAcceptsGzipRequests(t *testing.T) {
	var got capturedRequest
	up := stubUpstream(t, &got)
	h := newTestHandler(t, staticAuth{rec: enabledKey()}, up, nil)

	hdr := http.Header{}
	hdr.Set("Accept-Encoding", "gzip")
	if res := post(t, h, "/rpc/jg_k/evm/369", blockNumber, hdr); res.Code != http.StatusOK {
		t.Fatalf("status = %d with gzip, want 200", res.Code)
	}
	if got.hits != 1 {
		t.Errorf("upstream hits = %d, want 1", got.hits)
	}
}

func TestHandlerStatusMapping(t *testing.T) {
	tests := []struct {
		name string
		auth Authenticator
		path string
		body string
		want int
	}{
		{"unknown key", staticAuth{err: ErrUnknownKey}, "/rpc/jg_k/evm/369", blockNumber, http.StatusUnauthorized},
		{"disabled key", staticAuth{err: ErrDisabledKey}, "/rpc/jg_k/evm/369", blockNumber, http.StatusForbidden},
		{"store down", staticAuth{err: ErrUnavailable}, "/rpc/jg_k/evm/369", blockNumber, http.StatusServiceUnavailable},
		{"unknown category", staticAuth{rec: enabledKey()}, "/nope/jg_k/evm/369", blockNumber, http.StatusNotFound},
		{"bad shape", staticAuth{rec: enabledKey()}, "/rpc/jg_k/evm", blockNumber, http.StatusBadRequest},
		{"unsupported arch", staticAuth{rec: enabledKey()}, "/rpc/jg_k/svm/1", blockNumber, http.StatusNotImplemented},
		{"garbage body", staticAuth{rec: enabledKey()}, "/rpc/jg_k/evm/369", `not json`, http.StatusBadRequest},
	}
	for _, tt := range tests {
		var got capturedRequest
		up := stubUpstream(t, &got)
		h := newTestHandler(t, tt.auth, up, nil)

		res := post(t, h, tt.path, tt.body, nil)
		if res.Code != tt.want {
			t.Errorf("%s: status = %d, want %d (body %q)", tt.name, res.Code, tt.want, res.Body.String())
		}
		if got.hits != 0 {
			t.Errorf("%s: a refused request still reached the upstream", tt.name)
		}
	}
}

// A key record that says disabled is the same refusal as a disabled verdict.
// The relay must not forward for a revoked key just because the store answered
// with a record rather than an error.
func TestHandlerRefusesADisabledRecord(t *testing.T) {
	var got capturedRequest
	up := stubUpstream(t, &got)
	h := newTestHandler(t, staticAuth{rec: KeyRecord{ID: "k1", Enabled: false}}, up, nil)

	if res := post(t, h, "/rpc/jg_k/evm/369", blockNumber, nil); res.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", res.Code)
	}
	if got.hits != 0 {
		t.Error("a disabled key still reached the upstream")
	}
}

// A denied method never reaches an upstream, so it never costs a credit and
// never loads a node.
func TestHandlerRefusesADeniedMethod(t *testing.T) {
	var got capturedRequest
	up := stubUpstream(t, &got)
	rec := KeyRecord{ID: "k1", Enabled: true, MethodBlock: []string{"eth_sendRawTransaction"}}
	h := newTestHandler(t, staticAuth{rec: rec}, up, nil)

	body := `{"jsonrpc":"2.0","id":1,"method":"eth_sendRawTransaction","params":["0x0"]}`
	if res := post(t, h, "/rpc/jg_k/evm/369", body, nil); res.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", res.Code)
	}
	if got.hits != 0 {
		t.Error("a denied method still reached the upstream")
	}
}

// One denied entry refuses the whole batch, and none of it reaches the upstream.
func TestHandlerRefusesTheWholeBatchOnOneDenial(t *testing.T) {
	var got capturedRequest
	up := stubUpstream(t, &got)
	rec := KeyRecord{ID: "k1", Enabled: true, MethodBlock: []string{"eth_sendRawTransaction"}}
	h := newTestHandler(t, staticAuth{rec: rec}, up, nil)

	body := `[{"jsonrpc":"2.0","id":1,"method":"eth_call"},{"jsonrpc":"2.0","id":2,"method":"eth_sendRawTransaction"}]`
	if res := post(t, h, "/rpc/jg_k/evm/369", body, nil); res.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", res.Code)
	}
	if got.hits != 0 {
		t.Error("a partly denied batch still reached the upstream")
	}
}

// Beacon is a different protocol on a different backend. The relay strips the
// whole prefix so the beacon client sees its own native tree.
func TestHandlerRewritesBeaconToTheNativeTree(t *testing.T) {
	var erpcGot, beaconGot capturedRequest
	erpc := stubUpstream(t, &erpcGot)
	beacon := stubUpstream(t, &beaconGot)
	h := newTestHandler(t, staticAuth{rec: enabledKey()}, erpc, beacon)

	req := httptest.NewRequest(http.MethodGet, "/beacon/jg_secretkey/evm/369/eth/v1/beacon/genesis", nil)
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %q)", res.Code, res.Body.String())
	}
	if beaconGot.path != "/eth/v1/beacon/genesis" {
		t.Errorf("beacon path = %q, want /eth/v1/beacon/genesis", beaconGot.path)
	}
	if strings.Contains(beaconGot.path, "jg_secretkey") {
		t.Errorf("beacon path %q still carries the key", beaconGot.path)
	}
	if erpcGot.hits != 0 {
		t.Error("a beacon call reached eRPC")
	}
}

// A chain with no consensus layer answers a definite 501, never a dead 502. The
// catalog knows which chains have a beacon, so the relay can say so plainly.
func TestHandlerBeaconWithoutAnUpstreamIs501(t *testing.T) {
	var got capturedRequest
	erpc := stubUpstream(t, &got)
	h := newTestHandler(t, staticAuth{rec: enabledKey()}, erpc, nil)

	req := httptest.NewRequest(http.MethodGet, "/beacon/jg_k/evm/369/eth/v1/beacon/genesis", nil)
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)

	if res.Code != http.StatusNotImplemented {
		t.Fatalf("status = %d, want 501", res.Code)
	}
}

// The bare liveness level is unkeyed by design, so it must answer without ever
// consulting the key store.
func TestHandlerLivenessNeedsNoKey(t *testing.T) {
	var got capturedRequest
	up := stubUpstream(t, &got)
	// An authenticator that fails every call proves liveness never asks it.
	h := newTestHandler(t, staticAuth{err: ErrUnavailable}, up, nil)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", res.Code)
	}
}

// The unkeyed level must stay shallow. A detailed answer would tell any scanner
// which chains the operator runs and when one is lagging.
func TestHandlerLivenessLeaksNoDetail(t *testing.T) {
	var got capturedRequest
	up := stubUpstream(t, &got)
	h := newTestHandler(t, staticAuth{rec: enabledKey()}, up, nil)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)

	body := res.Body.String()
	for _, leak := range []string{"chain", "upstream", "head", "369", "syncing"} {
		if strings.Contains(strings.ToLower(body), leak) {
			t.Errorf("unkeyed liveness body %q leaks %q", body, leak)
		}
	}
}

// A keyed health request still authenticates. Health is credit-exempt, not
// public.
func TestHandlerKeyedHealthAuthenticates(t *testing.T) {
	var got capturedRequest
	up := stubUpstream(t, &got)
	h := newTestHandler(t, staticAuth{err: ErrUnknownKey}, up, nil)

	req := httptest.NewRequest(http.MethodGet, "/health/jg_bad/evm/369", nil)
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)

	if res.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", res.Code)
	}
}

// A key scoped to other chains cannot reach this one, and the refusal happens
// before the forward.
func TestHandlerEnforcesNetworkScope(t *testing.T) {
	var got capturedRequest
	up := stubUpstream(t, &got)
	rec := KeyRecord{ID: "k1", Enabled: true, Networks: []string{"1"}}
	h := newTestHandler(t, staticAuth{rec: rec}, up, nil)

	if res := post(t, h, "/rpc/jg_k/evm/369", blockNumber, nil); res.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", res.Code)
	}
	if got.hits != 0 {
		t.Error("an out-of-scope chain still reached the upstream")
	}
}

// A browser key is public by construction, so the origin is what actually
// bounds who may spend it.
func TestHandlerEnforcesOrigin(t *testing.T) {
	var got capturedRequest
	up := stubUpstream(t, &got)
	rec := KeyRecord{ID: "k1", Enabled: true, Origins: []string{"https://app.example"}}
	h := newTestHandler(t, staticAuth{rec: rec}, up, nil)

	hdr := http.Header{}
	hdr.Set("Origin", "https://evil.example")
	if res := post(t, h, "/rpc/jg_k/evm/369", blockNumber, hdr); res.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", res.Code)
	}
	if got.hits != 0 {
		t.Error("a denied origin still reached the upstream")
	}
}

// An error body must never quote the key back. A relay that echoed the
// credential into a 401 would put it in the caller's logs as well as its own.
func TestHandlerErrorsNeverEchoTheKey(t *testing.T) {
	var got capturedRequest
	up := stubUpstream(t, &got)
	h := newTestHandler(t, staticAuth{err: ErrUnknownKey}, up, nil)

	res := post(t, h, "/rpc/jg_verysecret/evm/369", blockNumber, nil)
	if strings.Contains(res.Body.String(), "jg_verysecret") {
		t.Errorf("error body %q echoes the key", res.Body.String())
	}
}
