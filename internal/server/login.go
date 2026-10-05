package server

import (
	"crypto/subtle"
	"io"
	"net"
	"net/http"
	"sync"
	"time"
)

// loginCodeTTL bounds how long a one-time login link works (spec D4).
const loginCodeTTL = 60 * time.Second

// maxLoginCodes caps the codes outstanding at once. Minting needs the session
// token, so this guards against a runaway caller, not an attacker; past the
// cap the oldest code is retired, so a fresh `jumpgate open` always works.
const maxLoginCodes = 16

// loginCodes are one-time, short-lived stand-ins for the session token in
// URLs handed to a browser opener. An opener's command line (xdg-open, open,
// rundll32, and the browser it starts) is readable by every local user
// through /proc or ps, so the long-lived token never goes there (I-6).
//
// The codes live in a short slice, oldest first, rather than a map: redeeming
// compares the candidate against every entry in constant time, so neither a
// map's hashing nor an early exit says anything about how close a guess was.
type loginCodes struct {
	mu   sync.Mutex
	list []loginCode
}

type loginCode struct {
	code    string
	expires time.Time
}

// NewLoginCode mints a code that signs one browser in, once, within
// loginCodeTTL. It is 128 bits from crypto/rand, hex-encoded.
func (s *Server) NewLoginCode() string {
	code := NewSessionToken()
	now := s.clock()
	s.codes.mu.Lock()
	defer s.codes.mu.Unlock()
	live := s.codes.list[:0]
	for _, c := range s.codes.list {
		if now.Before(c.expires) {
			live = append(live, c)
		}
	}
	if len(live) >= maxLoginCodes {
		live = append(live[:0], live[len(live)-maxLoginCodes+1:]...)
	}
	s.codes.list = append(live, loginCode{code: code, expires: now.Add(loginCodeTTL)})
	return code
}

// redeemLoginCode consumes code. It is true only for a code that exists and
// has not expired; either way a matching code is gone afterwards. The lock
// makes the check-and-delete atomic, so of any number of concurrent
// redemptions of one code exactly one succeeds.
func (s *Server) redeemLoginCode(code string) bool {
	if code == "" {
		return false
	}
	now := s.clock()
	s.codes.mu.Lock()
	defer s.codes.mu.Unlock()
	found := -1
	for i, c := range s.codes.list {
		if subtle.ConstantTimeCompare([]byte(c.code), []byte(code)) == 1 {
			found = i
		}
	}
	if found < 0 {
		return false
	}
	ok := now.Before(s.codes.list[found].expires)
	s.codes.list = append(s.codes.list[:found], s.codes.list[found+1:]...)
	return ok
}

// handleLogin exchanges a login code for the session cookie, then redirects
// to the app root so the code leaves the address bar. Nothing here logs: the
// code is a credential for its lifetime.
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	// The URL is a credential until redeemed: keep it out of caches and out
	// of any Referer the next page sends.
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	// A login link is for a browser on this machine. Refusing other peers
	// before redeeming means a server bound beyond loopback neither accepts
	// a code from the network nor lets the network burn one. The unix socket
	// has no peer address and no browser, so it is refused too.
	if !loopbackPeer(r.RemoteAddr) {
		http.Error(w, "jumpgate login links work only from this computer", http.StatusForbidden)
		return
	}
	if !s.redeemLoginCode(r.URL.Query().Get("code")) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, "This jumpgate login link has expired or was already used.\nRun `jumpgate open` for a new one.\n")
		return
	}
	http.SetCookie(w, sessionCookie(s.cfg.Token))
	http.Redirect(w, r, "/", http.StatusFound)
}

// loopbackPeer reports whether a request's remote address is a loopback IP.
func loopbackPeer(remoteAddr string) bool {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return false
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// sessionCookie carries the session token once a browser has signed in.
// HttpOnly keeps it from page scripts; SameSite=Strict keeps other sites'
// pages from riding it (authMiddleware adds an Origin check for the
// same-site-other-port case). It is not Secure: the server speaks plain HTTP
// on loopback, and browsers do not send a Secure cookie to http://127.0.0.1
// (Safari, and others for non-"localhost" hosts), so Secure would lock the
// user out. Loopback traffic never crosses a network a Secure flag protects.
func sessionCookie(token string) *http.Cookie {
	return &http.Cookie{Name: cookieName, Value: token, Path: "/", HttpOnly: true, SameSite: http.SameSiteStrictMode}
}
