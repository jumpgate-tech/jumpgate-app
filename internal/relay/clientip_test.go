package relay

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// The IP allow-list is only as good as the address it checks. X-Forwarded-For
// is a header the caller writes, so the relay may read it only when the hop that
// handed it over is a proxy the operator trusts, and then only the part that
// proxy wrote.

func ipHandler(t *testing.T, rec KeyRecord, trusted []string, got *capturedRequest) *Handler {
	t.Helper()
	h, err := NewHandler(Config{
		Auth:           staticAuth{rec: rec},
		ProjectID:      "main",
		ERPC:           stubUpstream(t, got),
		TrustedProxies: trusted,
	})
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}
	return h
}

// postFrom sends a call from remote, with the given X-Forwarded-For values.
func postFrom(h http.Handler, remote string, xff ...string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/rpc/jg_k/evm/369", strings.NewReader(blockNumber))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = remote
	for _, v := range xff {
		req.Header.Add("X-Forwarded-For", v)
	}
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	return res
}

func allowOnly(ip string) KeyRecord {
	rec := enabledKey()
	rec.IPAllow = []string{ip}
	return rec
}

// THE bypass the review found: any caller could name an allowed address in
// X-Forwarded-For and walk through the allow-list.
func TestSpoofedForwardedForDoesNotBypassTheAllowList(t *testing.T) {
	var got capturedRequest
	h := ipHandler(t, allowOnly("203.0.113.7"), nil, &got)

	res := postFrom(h, "198.51.100.9:5555", "203.0.113.7")
	if res.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403: X-Forwarded-For from an untrusted peer was believed", res.Code)
	}
	if got.hits != 0 {
		t.Errorf("upstream hits = %d, want 0", got.hits)
	}
}

// Nor can it dodge a deny entry by naming some other address.
func TestSpoofedForwardedForDoesNotDodgeTheDenyList(t *testing.T) {
	rec := enabledKey()
	rec.IPDeny = []string{"198.51.100.9"}
	h := ipHandler(t, rec, nil, &capturedRequest{})

	if res := postFrom(h, "198.51.100.9:5555", "203.0.113.7"); res.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403: a denied peer escaped by naming another address", res.Code)
	}
}

// With no trusted proxies configured, X-Forwarded-For is never read, even from
// loopback. Trust is something the operator grants, not a default.
func TestNoTrustedProxiesMeansTheHeaderIsIgnored(t *testing.T) {
	h := ipHandler(t, allowOnly("203.0.113.7"), nil, &capturedRequest{})

	if res := postFrom(h, "127.0.0.1:5555", "203.0.113.7"); res.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", res.Code)
	}
}

// Behind a trusted proxy the header is the only place the caller's address
// survives, so it must be read.
func TestTrustedProxyForwardedForIsHonoured(t *testing.T) {
	var got capturedRequest
	h := ipHandler(t, allowOnly("203.0.113.7"), []string{"127.0.0.1"}, &got)

	if res := postFrom(h, "127.0.0.1:5555", "203.0.113.7"); res.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", res.Code)
	}
}

// A trusted proxy APPENDS to whatever the caller sent. The left-most entry is
// therefore the caller's own words, and the right-most entry that is not a
// trusted proxy is the first address anyone the operator trusts actually saw.
func TestTheRightMostUntrustedEntryIsTheCaller(t *testing.T) {
	trusted := []string{"127.0.0.1", "10.0.0.0/8"}
	h := ipHandler(t, allowOnly("203.0.113.7"), trusted, &capturedRequest{})

	// The caller claims 203.0.113.7; the edge proxy saw 198.51.100.9 and an
	// inner proxy at 10.0.0.2 passed it on.
	res := postFrom(h, "127.0.0.1:5555", "203.0.113.7, 198.51.100.9, 10.0.0.2")
	if res.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403: the left-most, caller-written entry was believed", res.Code)
	}

	// The same chain split across repeated headers reads the same way.
	res = postFrom(h, "127.0.0.1:5555", "203.0.113.7", "198.51.100.9, 10.0.0.2")
	if res.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403 with the chain split across headers", res.Code)
	}

	h = ipHandler(t, allowOnly("198.51.100.9"), trusted, &capturedRequest{})
	if res := postFrom(h, "127.0.0.1:5555", "203.0.113.7, 198.51.100.9, 10.0.0.2"); res.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 for the address the edge proxy saw", res.Code)
	}
}

func TestNewHandlerRejectsAMalformedTrustedProxy(t *testing.T) {
	_, err := NewHandler(Config{
		Auth:           staticAuth{rec: enabledKey()},
		ProjectID:      "main",
		ERPC:           stubUpstream(t, &capturedRequest{}),
		TrustedProxies: []string{"not-an-address"},
	})
	if err == nil {
		t.Fatal("err = nil, want a refusal: a typo here would silently trust nothing or everything")
	}
}

// --- startup wiring ------------------------------------------------------------

func buildOn(t *testing.T, bind string, trusted []string) *Handler {
	t.Helper()
	h, _, err := Build(BuildOptions{
		RelayBind:      bind,
		BillingSocket:  "/run/jumpgate/billing.sock",
		RelayToken:     "relay-token",
		ERPCURL:        "http://127.0.0.1:4000",
		TrustedProxies: trusted,
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	return h.(*Handler)
}

func forwardedFrom(remote, xff string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/rpc/jg_k/evm/369", nil)
	req.RemoteAddr = remote
	req.Header.Set("X-Forwarded-For", xff)
	return req
}

// A relay bound to loopback is reachable only from this host, where the local
// Caddy is what fronts it, so loopback is trusted by default.
func TestBuildTrustsLoopbackWhenBoundToLoopback(t *testing.T) {
	for _, bind := range []string{"127.0.0.1:8790", "[::1]:8790", "localhost:8790"} {
		h := buildOn(t, bind, nil)
		if got := h.clientIP(forwardedFrom("127.0.0.1:5555", "203.0.113.7")); got != "203.0.113.7" {
			t.Errorf("bind %s: client = %q, want the address Caddy forwarded", bind, got)
		}
		if got := h.clientIP(forwardedFrom("[::1]:5555", "203.0.113.7")); got != "203.0.113.7" {
			t.Errorf("bind %s: client over ::1 = %q, want the address Caddy forwarded", bind, got)
		}
	}
}

// Bound anywhere else, the relay cannot tell its proxy from a caller, so it
// trusts nobody until the operator says otherwise.
func TestBuildTrustsNobodyWhenBoundElsewhere(t *testing.T) {
	h := buildOn(t, "10.0.0.5:8790", nil)
	if got := h.clientIP(forwardedFrom("127.0.0.1:5555", "203.0.113.7")); got != "127.0.0.1" {
		t.Errorf("client = %q, want the peer address: nothing is trusted on a non-loopback bind", got)
	}
}

func TestBuildUsesExplicitTrustedProxies(t *testing.T) {
	h := buildOn(t, "127.0.0.1:8790", []string{"172.17.0.0/16"})
	if got := h.clientIP(forwardedFrom("172.17.0.2:5555", "203.0.113.7")); got != "203.0.113.7" {
		t.Errorf("client = %q, want the address the configured proxy forwarded", got)
	}
	if got := h.clientIP(forwardedFrom("127.0.0.1:5555", "203.0.113.7")); got != "127.0.0.1" {
		t.Errorf("client = %q: an explicit list must replace the loopback default, not add to it", got)
	}
}
