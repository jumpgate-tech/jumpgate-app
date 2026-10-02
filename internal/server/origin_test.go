package server

import (
	"net/http"
	"testing"
)

// The session cookie is SameSite=Strict, but "site" ignores the port: a page on
// http://127.0.0.1:3000 (any local dev server) is same-site with this UI on
// :8799, so the browser attaches the cookie to its cross-port POSTs. A changing
// request authorised by the cookie must therefore also come from this origin.

func cookieReq(t *testing.T, method, url, token string, hdr map[string]string) *http.Response {
	t.Helper()
	req, _ := http.NewRequest(method, url, nil)
	req.AddCookie(&http.Cookie{Name: cookieName, Value: token})
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	res.Body.Close()
	return res
}

func TestCookieWritesRefuseAnotherOrigin(t *testing.T) {
	ts, token := testServer(t)
	url := ts.URL + "/api/health"

	for name, hdr := range map[string]map[string]string{
		"other port":       {"Origin": "http://127.0.0.1:3000"},
		"other host":       {"Origin": "https://evil.example"},
		"null origin":      {"Origin": "null"},
		"fetch cross-site": {"Sec-Fetch-Site": "cross-site"},
		"fetch same-site":  {"Sec-Fetch-Site": "same-site"},
	} {
		if res := cookieReq(t, http.MethodPost, url, token, hdr); res.StatusCode != http.StatusForbidden {
			t.Errorf("%s: POST = %d, want 403", name, res.StatusCode)
		}
	}
}

func TestCookieWritesFromThisOriginStillWork(t *testing.T) {
	ts, token := testServer(t)
	url := ts.URL + "/api/health"

	for name, hdr := range map[string]map[string]string{
		"same origin":        {"Origin": ts.URL},
		"fetch same-origin":  {"Sec-Fetch-Site": "same-origin", "Origin": ts.URL},
		"no browser headers": nil,
	} {
		if res := cookieReq(t, http.MethodPost, url, token, hdr); res.StatusCode == http.StatusForbidden {
			t.Errorf("%s: POST was refused as cross-origin", name)
		}
	}
}

// A cross-origin page cannot attach an Authorization header without a CORS
// preflight this server never grants, so bearer callers (the TUI, scripts) are
// not subject to the origin rule.
func TestBearerWritesAreNotOriginChecked(t *testing.T) {
	ts, token := testServer(t)
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/health", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Origin", "http://127.0.0.1:3000")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode == http.StatusForbidden {
		t.Error("a bearer request was refused for its Origin")
	}
}

// Reads stay open: a cross-origin page cannot read the response anyway.
func TestCookieReadsAreNotOriginChecked(t *testing.T) {
	ts, token := testServer(t)
	res := cookieReq(t, http.MethodGet, ts.URL+"/api/health", token, map[string]string{"Origin": "http://127.0.0.1:3000"})
	if res.StatusCode != http.StatusOK {
		t.Errorf("GET = %d, want 200", res.StatusCode)
	}
}

// Both listeners bound how long a client may take to send its headers, so a
// slow-header client cannot hold sockets open indefinitely.
func TestServersBoundHeaderReads(t *testing.T) {
	srv := newHTTPServer("127.0.0.1:0", http.NotFoundHandler())
	if srv.ReadHeaderTimeout <= 0 {
		t.Fatal("ReadHeaderTimeout is unset")
	}
	if srv.IdleTimeout <= 0 {
		t.Error("IdleTimeout is unset")
	}
	// SSE and WebSocket responses are long-lived, so a whole-response write
	// deadline would cut them off.
	if srv.WriteTimeout != 0 {
		t.Errorf("WriteTimeout = %v; it would cut off streaming responses", srv.WriteTimeout)
	}
}
