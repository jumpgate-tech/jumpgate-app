package server

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/fstest"
	"time"
)

func loginServer(t *testing.T) (*Server, *httptest.Server, string) {
	t.Helper()
	token := NewSessionToken()
	s := New(Config{Token: token, UI: fstest.MapFS{"index.html": {Data: []byte("ui")}}})
	ts := httptest.NewServer(s.Handler())
	t.Cleanup(ts.Close)
	return s, ts, token
}

func noRedirect() *http.Client {
	return &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
}

// login visits /login with code and returns the response with its body read.
func login(t *testing.T, ts *httptest.Server, code string) (*http.Response, string) {
	t.Helper()
	res, err := noRedirect().Get(ts.URL + "/login?code=" + code)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	return res, string(body)
}

// Review Focus 2: the first visit signs in; a second use of the same link is
// refused with a page naming `jumpgate open`; the cookie from the first stays
// valid.
func TestLoginCodeIsSingleUse(t *testing.T) {
	s, ts, token := loginServer(t)
	code := s.NewLoginCode()
	if code == token || len(code) != 32 {
		t.Fatalf("code %q: want 32 hex chars distinct from the token", code)
	}
	// A 200 page that navigates to / on its own, not a 302: the sign-in
	// starts from the file:// redirect page, so Chromium treats a redirect
	// chain from it as cross-site and withholds the SameSite=Strict cookie
	// on /. A navigation the /login page itself starts is same-origin.
	res, page := login(t, ts, code)
	if res.StatusCode != http.StatusOK || res.Header.Get("Location") != "" {
		t.Fatalf("first use: %d Location %q, want 200 and no Location", res.StatusCode, res.Header.Get("Location"))
	}
	for _, want := range []string{`http-equiv="refresh" content="0;url=/"`, `location.replace("/")`, `<meta name="referrer" content="no-referrer">`} {
		if !strings.Contains(page, want) {
			t.Fatalf("success page %q: missing %s", page, want)
		}
	}
	if strings.Contains(page, code) || strings.Contains(page, token) {
		t.Fatalf("success page %q carries a credential", page)
	}
	var cookie *http.Cookie
	for _, c := range res.Cookies() {
		if c.Name == cookieName {
			cookie = c
		}
	}
	if cookie == nil || cookie.Value != token || !cookie.HttpOnly ||
		cookie.SameSite != http.SameSiteStrictMode || cookie.Path != "/" || cookie.Secure {
		t.Fatalf("session cookie %+v", cookie)
	}
	if res.Header.Get("Referrer-Policy") != "no-referrer" || res.Header.Get("Cache-Control") != "no-store" {
		t.Fatalf("login response headers %v: want no-referrer and no-store", res.Header)
	}

	res, body := login(t, ts, code)
	if res.StatusCode != http.StatusUnauthorized || !strings.Contains(body, "jumpgate open") {
		t.Fatalf("second use: %d %q", res.StatusCode, body)
	}
	if strings.Contains(body, code) {
		t.Fatalf("the refusal page echoes the code: %q", body)
	}

	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/health", nil)
	req.AddCookie(cookie)
	res, err := http.DefaultClient.Do(req)
	if err != nil || res.StatusCode != http.StatusOK {
		t.Fatalf("cookie after a refused reuse: %v %v", res, err)
	}
	res.Body.Close()
}

// Many tabs racing one link (or an attacker racing the user): exactly one
// redemption wins.
func TestLoginCodeRedeemsOnceUnderConcurrency(t *testing.T) {
	s, ts, _ := loginServer(t)
	code := s.NewLoginCode()
	const n = 32
	var wins atomic.Int32
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			res, err := noRedirect().Get(ts.URL + "/login?code=" + code)
			if err != nil {
				t.Error(err)
				return
			}
			res.Body.Close()
			if res.StatusCode == http.StatusOK {
				wins.Add(1)
			}
		}()
	}
	close(start)
	wg.Wait()
	if got := wins.Load(); got != 1 {
		t.Fatalf("%d of %d concurrent redemptions succeeded, want exactly 1", got, n)
	}
}

func TestLoginCodeExpires(t *testing.T) {
	s, ts, _ := loginServer(t)
	now := time.Now()
	var mu sync.Mutex
	s.now = func() time.Time { mu.Lock(); defer mu.Unlock(); return now }
	code := s.NewLoginCode()
	mu.Lock()
	now = now.Add(loginCodeTTL + time.Second)
	mu.Unlock()
	if res, _ := login(t, ts, code); res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expired code: %d, want 401", res.StatusCode)
	}
}

func TestUnknownLoginCodeIsRefused(t *testing.T) {
	_, ts, _ := loginServer(t)
	for _, q := range []string{"", "?code=", "?code=00000000000000000000000000000000"} {
		res, err := noRedirect().Get(ts.URL + "/login" + q)
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		if res.StatusCode != http.StatusUnauthorized {
			t.Fatalf("/login%s: %d, want 401", q, res.StatusCode)
		}
	}
}

// A login link works only from this machine. A request from another address
// (a server bound beyond loopback) is refused and does not burn the code.
func TestLoginCodeIsLoopbackOnly(t *testing.T) {
	s, _, _ := loginServer(t)
	// The synthetic requests carry no LocalAddr, which the Linux peer check
	// (rightly) refuses; this test is about the loopback gate alone.
	s.peerUID = nil
	code := s.NewLoginCode()
	for _, peer := range []string{"192.0.2.10:5000", "@", ""} {
		req := httptest.NewRequest(http.MethodGet, "/login?code="+code, nil)
		req.RemoteAddr = peer
		rec := httptest.NewRecorder()
		s.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden || len(rec.Result().Cookies()) != 0 {
			t.Fatalf("peer %q: %d %v, want 403 and no cookie", peer, rec.Code, rec.Result().Cookies())
		}
	}
	for _, peer := range []string{"127.0.0.1:5000", "[::1]:5000"} {
		c := s.NewLoginCode()
		req := httptest.NewRequest(http.MethodGet, "/login?code="+c, nil)
		req.RemoteAddr = peer
		rec := httptest.NewRecorder()
		s.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("loopback peer %q: %d, want 200", peer, rec.Code)
		}
	}
	if !s.redeemLoginCode(code) {
		t.Fatal("a refused non-loopback request burned the code")
	}
}

// Outstanding codes are capped: minting past the cap retires the oldest, so
// a caller holding the token cannot grow the table without bound.
func TestLoginCodesAreCapped(t *testing.T) {
	s, _, _ := loginServer(t)
	first := s.NewLoginCode()
	var last string
	for i := 0; i < maxLoginCodes; i++ {
		last = s.NewLoginCode()
	}
	s.codes.mu.Lock()
	n := len(s.codes.list)
	s.codes.mu.Unlock()
	if n != maxLoginCodes {
		t.Fatalf("%d outstanding codes, want the cap %d", n, maxLoginCodes)
	}
	if s.redeemLoginCode(first) {
		t.Fatal("the oldest code survived past the cap")
	}
	if !s.redeemLoginCode(last) {
		t.Fatal("the newest code was refused")
	}
}

// The code is a credential for its lifetime: nothing on the login path may
// write it to the log.
func TestLoginCodeIsNeverLogged(t *testing.T) {
	var buf bytes.Buffer
	old := log.Writer()
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(old) })

	s, ts, _ := loginServer(t)
	code := s.NewLoginCode()
	login(t, ts, code)
	login(t, ts, code)
	if strings.Contains(buf.String(), code) {
		t.Fatalf("the login code reached the log: %q", buf.String())
	}
}

// Another process (a second app launch, `jumpgate open`) mints codes over the
// authenticated API; nobody else can.
func TestLoginCodeMintNeedsTheToken(t *testing.T) {
	_, ts, token := loginServer(t)
	res, err := http.Post(ts.URL+"/api/login-code", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauthenticated mint: %d", res.StatusCode)
	}
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/login-code", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	res, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	var body struct{ Code string }
	_ = json.NewDecoder(res.Body).Decode(&body)
	res.Body.Close()
	if res.StatusCode != http.StatusOK || len(body.Code) != 32 {
		t.Fatalf("mint: %d %+v", res.StatusCode, body)
	}
	if res, _ := login(t, ts, body.Code); res.StatusCode != http.StatusOK {
		t.Fatalf("minted code: %d, want 200", res.StatusCode)
	}
}
