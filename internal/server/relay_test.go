package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// The relay is a SEPARATE listener with a separate handler. The control plane's
// session token authorises full control of the operator's servers; a customer
// holds a key. Keeping the two on one mux would put that boundary one forgotten
// route away from collapsing, so these tests assert the separation itself
// rather than any behaviour built on top of it.

// stubRelay stands in for the data-plane handler. It records that it was reached.
type stubRelay struct{ hits int }

func (s *stubRelay) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.hits++
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"relay":true}`))
}

// The data plane must not serve the control plane's API. A customer key would
// otherwise reach routes that provision and wipe the operator's servers.
func TestRelayHandlerServesNoControlPlaneRoutes(t *testing.T) {
	relay := &stubRelay{}
	s := New(Config{Bind: "127.0.0.1:0", Token: "session-token", Relay: relay})

	handler := s.RelayHandler()
	if handler == nil {
		t.Fatal("RelayHandler() = nil, want the data-plane handler")
	}

	for _, path := range []string{
		"/api/health",
		"/api/gateways",
		"/api/gateways/g1/wipe",
		"/api/targets",
		"/",
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Authorization", "Bearer session-token")
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)

		// The relay answers every path it is given, because ParseRoute decides
		// what is addressable. What must NOT happen is the control plane's own
		// handler running here.
		if strings.Contains(res.Body.String(), `"gateways"`) || strings.Contains(res.Body.String(), `"ok":true`) {
			t.Errorf("%s: the data plane served a control-plane response: %q", path, res.Body.String())
		}
	}
}

// The data plane authenticates by key, so it must not accept the session token
// as a credential. A leaked relay URL plus a session token must not become
// control of the box.
func TestRelayHandlerIgnoresTheSessionToken(t *testing.T) {
	relay := &stubRelay{}
	s := New(Config{Bind: "127.0.0.1:0", Token: "session-token", Relay: relay})

	req := httptest.NewRequest(http.MethodPost, "/rpc/jg_k/evm/1", nil)
	req.Header.Set("Authorization", "Bearer session-token")
	res := httptest.NewRecorder()
	s.RelayHandler().ServeHTTP(res, req)

	if relay.hits != 1 {
		t.Fatalf("relay hits = %d, want 1 — the data plane did not reach the relay", relay.hits)
	}
}

// The control plane must not serve the relay's grammar. Its authMiddleware
// wraps everything, so a customer key gets a 401 rather than a proxied call.
func TestControlPlaneDoesNotServeTheRelayGrammar(t *testing.T) {
	relay := &stubRelay{}
	s := New(Config{Bind: "127.0.0.1:0", Token: "session-token", Relay: relay})

	req := httptest.NewRequest(http.MethodPost, "/rpc/jg_customerkey/evm/1", nil)
	res := httptest.NewRecorder()
	s.Handler().ServeHTTP(res, req)

	if relay.hits != 0 {
		t.Errorf("the control plane reached the relay %d times, want 0", relay.hits)
	}
	if res.Code == http.StatusOK {
		t.Errorf("status = 200, want the control plane to refuse an unauthenticated call")
	}
}

// A customer key must never authorise a control-plane route.
func TestControlPlaneRefusesACustomerKey(t *testing.T) {
	s := New(Config{Bind: "127.0.0.1:0", Token: "session-token"})

	req := httptest.NewRequest(http.MethodGet, "/api/gateways", nil)
	req.Header.Set("Authorization", "Bearer jg_customerkey")
	res := httptest.NewRecorder()
	s.Handler().ServeHTTP(res, req)

	if res.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", res.Code)
	}
}

// With no relay configured the server still runs. A gateway that sells no keys
// needs no data plane, and the absence must not be a nil dereference.
func TestRelayHandlerIsNilWhenNoRelayConfigured(t *testing.T) {
	s := New(Config{Bind: "127.0.0.1:0", Token: "session-token"})
	if s.RelayHandler() != nil {
		t.Error("RelayHandler() is non-nil with no relay configured")
	}
}
