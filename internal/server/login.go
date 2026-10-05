package server

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/valve-tech/jumpgate/internal/fsperm"
)

// loginCodeTTL bounds how long a one-time login link works (spec D4).
const loginCodeTTL = 60 * time.Second

// maxLoginCodes caps the codes outstanding at once. Minting needs the session
// token, so this guards against a runaway caller, not an attacker; past the
// cap the oldest code is retired, so a fresh `jumpgate open` always works.
const maxLoginCodes = 16

// loginCodes are one-time, short-lived stand-ins for the session token, used
// to sign a browser in (I-6, D4).
//
// A code must never reach a process's command line either: every local user
// reads argv through /proc or ps, every local user is a loopback peer, and
// one who redeems the code before the browser does gets the session cookie.
// So the code goes into an owner-only redirect file (see NewLoginLink) and
// the browser opener is handed only that file's path, the way Jupyter does.
//
// The codes live in a short slice, oldest first, rather than a map: redeeming
// compares the candidate against every entry in constant time, so neither a
// map's hashing nor an early exit says anything about how close a guess was.
// spent remembers redeemed codes until they would have expired, so a second
// presentation, the sign of a lost race, can be reported.
type loginCodes struct {
	mu    sync.Mutex
	list  []loginCode
	spent []loginCode
}

type loginCode struct {
	code    string
	expires time.Time
	// file is the redirect file that carries the code, removed with it; ""
	// for a code minted without one.
	file string
}

// LoginLink is a minted login code, its URL, and the owner-only redirect
// file that leads a browser to that URL. File is "" when the server has no
// LoginDir or could not write the file; the caller must then not open a
// browser on the URL (its argv would carry the code), only show it.
type LoginLink struct {
	Code string `json:"code"`
	URL  string `json:"-"`
	File string `json:"file,omitempty"`
}

// NewLoginCode mints a code that signs one browser in, once, within
// loginCodeTTL. It is 128 bits from crypto/rand, hex-encoded.
func (s *Server) NewLoginCode() string { return s.addLoginCode("") }

func (s *Server) addLoginCode(file string) string {
	code := NewSessionToken()
	now := s.clock()
	s.codes.mu.Lock()
	defer s.codes.mu.Unlock()
	s.pruneLocked(now)
	if n := len(s.codes.list); n >= maxLoginCodes {
		for _, c := range s.codes.list[:n-maxLoginCodes+1] {
			removeLoginFile(c.file)
		}
		s.codes.list = append(s.codes.list[:0], s.codes.list[n-maxLoginCodes+1:]...)
	}
	s.codes.list = append(s.codes.list, loginCode{code: code, expires: now.Add(loginCodeTTL), file: file})
	return code
}

// NewLoginLink mints a login code and writes the owner-only redirect file
// that carries it. The file is removed when the code is redeemed, expires or
// is retired, and stale ones are swept when a server starts.
func (s *Server) NewLoginLink() (LoginLink, error) {
	if s.cfg.LoginDir == "" {
		code := s.NewLoginCode()
		return LoginLink{Code: code, URL: LoginURL(s.cfg.Bind, code)}, nil
	}
	if s.cfg.Bind == "" {
		return LoginLink{}, errors.New("login link file: the server has no HTTP address")
	}
	file, f, err := createLoginFile(s.cfg.LoginDir)
	if err != nil {
		return LoginLink{}, fmt.Errorf("login link file: %w", err)
	}
	code := s.addLoginCode(file)
	link := LoginLink{Code: code, URL: LoginURL(s.cfg.Bind, code), File: file}
	_, werr := io.WriteString(f, redirectPage(link.URL))
	if cerr := f.Close(); werr == nil {
		werr = cerr
	}
	if werr != nil {
		s.redeemLoginCode(code) // retire the code with its unusable file
		return LoginLink{}, fmt.Errorf("login link file: %w", werr)
	}
	// Remove the file promptly once the code is dead, not just on the next
	// mint or redemption.
	time.AfterFunc(loginCodeTTL+time.Second, s.pruneLoginCodes)
	return link, nil
}

// LoginURL is the one-time login link for a server listening on addr. Codes
// are redeemed only from loopback, so a wildcard bind becomes its loopback
// address, the one a local browser can reach.
func LoginURL(addr, code string) string {
	if host, port, err := net.SplitHostPort(addr); err == nil {
		switch host {
		case "", "0.0.0.0":
			addr = net.JoinHostPort("127.0.0.1", port)
		case "::":
			addr = net.JoinHostPort("::1", port)
		}
	}
	return "http://" + addr + "/login?code=" + code
}

// redirectPage sends a browser that opens it on to url at once. The meta
// refresh covers a browser with scripts off; no-referrer keeps the page's
// own URL (a file path, or /login with its code) out of the next request.
func redirectPage(url string) string {
	js, _ := json.Marshal(url) // escapes <, > and & as \u00XX
	h := html.EscapeString(url)
	return `<!doctype html>
<html><head><meta charset="utf-8"><meta name="referrer" content="no-referrer">
<meta http-equiv="refresh" content="0;url=` + h + `">
<title>jumpgate</title>
<script>location.replace(` + string(js) + `)</script>
</head><body><a href="` + h + `">Open jumpgate</a></body></html>
`
}

// Redirect files are named open-<random>.html; the random part is not the code.
const loginFilePrefix, loginFileSuffix = "open-", ".html"

// createLoginFile makes dir owner-only (fsperm.MkdirPrivate) and creates a
// fresh redirect file in it with fsperm.CreatePrivate: private from the
// moment it exists (0600 on unix, the owner-only DACL on Windows) and never
// created through a link.
func createLoginFile(dir string) (string, *os.File, error) {
	if err := fsperm.MkdirPrivate(dir); err != nil {
		return "", nil, err
	}
	for try := 0; try < 10; try++ {
		path := filepath.Join(dir, loginFilePrefix+NewSessionToken()[:16]+loginFileSuffix)
		f, err := fsperm.CreatePrivate(path)
		if errors.Is(err, fs.ErrExist) {
			continue
		}
		return path, f, err
	}
	return "", nil, fmt.Errorf("no free file name in %s", dir)
}

// sweepLoginFiles removes redirect files a previous server left behind. Only
// one server runs per user, so none of them belongs to a live code. It
// follows no link: a symlinked dir is not entered, and only regular files
// are removed, so a link named like a redirect file (and its target) is left
// alone.
func sweepLoginFiles(dir string) {
	if fi, err := os.Lstat(dir); err != nil || !fi.IsDir() {
		return
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		name := e.Name()
		if !e.Type().IsRegular() || !strings.HasPrefix(name, loginFilePrefix) || !strings.HasSuffix(name, loginFileSuffix) {
			continue
		}
		_ = os.Remove(filepath.Join(dir, name))
	}
}

func removeLoginFile(path string) {
	if path != "" {
		_ = os.Remove(path)
	}
}

// pruneLoginCodes drops expired codes and their files.
func (s *Server) pruneLoginCodes() {
	now := s.clock()
	s.codes.mu.Lock()
	defer s.codes.mu.Unlock()
	s.pruneLocked(now)
}

func (s *Server) pruneLocked(now time.Time) {
	live := s.codes.list[:0]
	for _, c := range s.codes.list {
		if now.Before(c.expires) {
			live = append(live, c)
		} else {
			removeLoginFile(c.file)
		}
	}
	s.codes.list = live
	spent := s.codes.spent[:0]
	for _, c := range s.codes.spent {
		if now.Before(c.expires) {
			spent = append(spent, c)
		}
	}
	s.codes.spent = spent
}

// redeemLoginCode consumes code. It is true only for a code that exists and
// has not expired; either way a matching code and its file are gone
// afterwards. The lock makes the check-and-delete atomic, so of any number of
// concurrent redemptions of one code exactly one succeeds.
func (s *Server) redeemLoginCode(code string) bool {
	ok, _ := s.redeem(code)
	return ok
}

// redeem is redeemLoginCode that also reports whether code was one already
// redeemed (and not yet expired).
func (s *Server) redeem(code string) (ok, reused bool) {
	if code == "" {
		return false, false
	}
	now := s.clock()
	s.codes.mu.Lock()
	defer s.codes.mu.Unlock()
	found := matchCode(s.codes.list, code)
	if found < 0 {
		i := matchCode(s.codes.spent, code)
		return false, i >= 0 && now.Before(s.codes.spent[i].expires)
	}
	c := s.codes.list[found]
	s.codes.list = append(s.codes.list[:found], s.codes.list[found+1:]...)
	removeLoginFile(c.file)
	if !now.Before(c.expires) {
		return false, false
	}
	s.codes.spent = append(s.codes.spent, loginCode{code: c.code, expires: c.expires})
	if n := len(s.codes.spent); n > maxLoginCodes {
		s.codes.spent = append(s.codes.spent[:0], s.codes.spent[n-maxLoginCodes:]...)
	}
	return true, false
}

// matchCode is the index of code in list, compared against every entry in
// constant time, or -1.
func matchCode(list []loginCode, code string) int {
	found := -1
	for i, c := range list {
		if subtle.ConstantTimeCompare([]byte(c.code), []byte(code)) == 1 {
			found = i
		}
	}
	return found
}

// handleLogin exchanges a login code for the session cookie, then sends the
// browser on to the app root so the code leaves the address bar. The code is never
// logged: it is a credential for its lifetime.
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
	// Every local user is a loopback peer. Where the peer's owner can be
	// read (Linux), a different user's connection is refused before the
	// code is touched, so it cannot burn the code either. root is let in: it
	// can read the session token anyway. Readable socket tables that do not
	// list the client are refused too (fail closed: another user can provoke
	// a torn read); only tables that cannot be read at all, or that list no
	// connections whatsoever (WSL1, gVisor), let the login through, with a
	// warning.
	if s.peerUID != nil {
		switch uid, v := s.peerUID(r); v {
		case peerFound:
			if uid != s.selfUID && uid != 0 {
				log.Printf("jumpgate: WARNING: refused a login link from another local user (uid %d)", uid)
				http.Error(w, "this jumpgate login link belongs to another user", http.StatusForbidden)
				return
			}
		case peerMissing:
			log.Printf("jumpgate: WARNING: refused a login link whose client socket /proc/net/tcp does not list (from %s); run `jumpgate open` again", r.RemoteAddr)
			http.Error(w, "jumpgate could not confirm this login link comes from you; run `jumpgate open` again", http.StatusForbidden)
			return
		case peerNotReported:
			log.Printf("jumpgate: WARNING: /proc/net/tcp and tcp6 list no connections at all (as under WSL1 or gVisor), so the owner of a login link's connection is unknown; allowing it, and on this system the login link relies on its owner-only file alone")
		default:
			log.Printf("jumpgate: WARNING: /proc/net/tcp cannot be read, so the owner of a login link's connection is unknown; allowing it, and on this system the login link relies on its owner-only file alone")
		}
	}
	ok, reused := s.redeem(r.URL.Query().Get("code"))
	if !ok {
		if reused {
			log.Printf("jumpgate: WARNING: a login link was used twice (from %s); if you did not open it twice, another local program may have signed in before your browser, so restart jumpgate to replace the session token", r.RemoteAddr)
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, "This jumpgate login link has expired or was already used.\nRun `jumpgate open` for a new one.\n")
		return
	}
	http.SetCookie(w, sessionCookie(s.cfg.Token))
	// A page that moves on to / itself, not a 302. The sign-in started on
	// the file:// redirect page, and Chromium treats a redirect chain from
	// there as a cross-site navigation, withholding the SameSite=Strict
	// cookie on / (the user lands on 401). A navigation this page starts
	// comes from this origin, so the cookie goes with it. Either way the
	// code leaves the address bar.
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, redirectPage("/"))
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
// on loopback, and not every browser treats http://127.0.0.1 as a secure
// origin (Safari does not), so a Secure cookie would be dropped there and
// lock the user out. Loopback traffic never crosses a network a Secure flag
// would protect.
func sessionCookie(token string) *http.Cookie {
	return &http.Cookie{Name: cookieName, Value: token, Path: "/", HttpOnly: true, SameSite: http.SameSiteStrictMode}
}
