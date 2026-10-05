package server

import (
	"bytes"
	"encoding/binary"
	"log"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	"github.com/valve-tech/jumpgate/internal/fsperm"
)

func loginFileServer(t *testing.T) (*Server, *httptest.Server, string) {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "login")
	ts := httptest.NewUnstartedServer(nil)
	s := New(Config{Bind: ts.Listener.Addr().String(), Token: NewSessionToken(), UI: fstest.MapFS{"index.html": {Data: []byte("ui")}}, LoginDir: dir})
	ts.Config.Handler = s.Handler()
	ts.Start()
	t.Cleanup(ts.Close)
	return s, ts, dir
}

var linkInFile = regexp.MustCompile(`http://[^"'<>\s]+/login\?code=[0-9a-f]{32}`)

// The security review's race: a code on an opener's argv is readable by every
// local user, who can redeem it before the browser does. The code therefore
// travels only inside an owner-only file whose path is all the opener sees.
func TestLoginLinkFileIsPrivateAndCarriesTheCode(t *testing.T) {
	s, _, dir := loginFileServer(t)
	link, err := s.NewLoginLink()
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Dir(link.File) != dir || strings.Contains(link.File, link.Code) || !strings.HasSuffix(link.File, ".html") {
		t.Fatalf("file %q: want an .html file in %s whose name is not the code", link.File, dir)
	}
	fi, err := os.Lstat(link.File)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && fi.Mode().Perm() != 0o600 {
		t.Fatalf("file mode %v, want 0600", fi.Mode().Perm())
	}
	// On Windows this is the owner-only DACL, set from creation.
	if err := fsperm.CheckPrivate(link.File); err != nil {
		t.Fatalf("redirect file is not private: %v", err)
	}
	if err := fsperm.CheckPrivate(dir); err != nil {
		t.Fatalf("login dir is not private: %v", err)
	}
	data, _ := os.ReadFile(link.File)
	if got := linkInFile.FindString(string(data)); got != link.URL || !strings.Contains(link.URL, link.Code) {
		t.Fatalf("file holds %q, want %q", got, link.URL)
	}
	if !strings.Contains(string(data), `http-equiv="refresh"`) || !strings.Contains(string(data), "location.replace") {
		t.Fatalf("file %q: want a meta refresh and a location.replace", data)
	}
}

func TestLoginLinkFileIsRemovedOnRedemption(t *testing.T) {
	s, ts, _ := loginFileServer(t)
	link, err := s.NewLoginLink()
	if err != nil {
		t.Fatal(err)
	}
	if res, _ := login(t, ts, link.Code); res.StatusCode != http.StatusFound {
		t.Fatalf("redeem: %d", res.StatusCode)
	}
	if _, err := os.Lstat(link.File); !os.IsNotExist(err) {
		t.Fatalf("file still there after redemption: %v", err)
	}
}

func TestLoginLinkFileIsRemovedOnExpiry(t *testing.T) {
	s, _, _ := loginFileServer(t)
	now := time.Now()
	var mu sync.Mutex
	s.now = func() time.Time { mu.Lock(); defer mu.Unlock(); return now }
	old, err := s.NewLoginLink()
	if err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	now = now.Add(loginCodeTTL + time.Second)
	mu.Unlock()
	s.pruneLoginCodes()
	if _, err := os.Lstat(old.File); !os.IsNotExist(err) {
		t.Fatalf("expired code's file still there: %v", err)
	}
}

// A server that died with links outstanding leaves files behind; the next
// one sweeps them, and nothing else in the directory.
func TestNewServerSweepsStaleLoginFiles(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "login")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	stale := filepath.Join(dir, "open-deadbeef.html")
	other := filepath.Join(dir, "keep.txt")
	for _, p := range []string{stale, other} {
		if err := os.WriteFile(p, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	New(Config{Token: NewSessionToken(), LoginDir: dir})
	if _, err := os.Lstat(stale); !os.IsNotExist(err) {
		t.Fatalf("stale login file survived startup: %v", err)
	}
	if _, err := os.Lstat(other); err != nil {
		t.Fatalf("sweep removed an unrelated file: %v", err)
	}
}

// Without an HTTP address there is no URL to put in a file; minting a link
// fails rather than writing a broken one.
func TestLoginLinkNeedsABind(t *testing.T) {
	s := New(Config{Token: NewSessionToken(), LoginDir: t.TempDir()})
	if _, err := s.NewLoginLink(); err == nil {
		t.Fatal("NewLoginLink with no Bind: want an error")
	}
}

// The sweep removes only regular files: a link named like a redirect file is
// left alone and its target untouched, and a symlinked login dir is not
// entered at all.
func TestSweepLoginFilesRefusesLinks(t *testing.T) {
	base := t.TempDir()
	dir := filepath.Join(base, "login")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	victim := filepath.Join(base, "victim.html")
	if err := os.WriteFile(victim, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "open-link.html")
	if err := os.Symlink(victim, link); err != nil {
		t.Skipf("no symlinks here: %v", err)
	}
	sweepLoginFiles(dir)
	if _, err := os.Lstat(link); err != nil {
		t.Fatalf("sweep removed a link: %v", err)
	}
	if _, err := os.Lstat(victim); err != nil {
		t.Fatalf("sweep followed a link: %v", err)
	}

	// A symlinked login dir: its open-*.html files belong to wherever it
	// points, so the sweep stays out.
	other := filepath.Join(base, "elsewhere")
	if err := os.MkdirAll(other, 0o700); err != nil {
		t.Fatal(err)
	}
	inOther := filepath.Join(other, "open-deadbeef.html")
	if err := os.WriteFile(inOther, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	linkedDir := filepath.Join(base, "linked")
	if err := os.Symlink(other, linkedDir); err != nil {
		t.Fatal(err)
	}
	sweepLoginFiles(linkedDir)
	if _, err := os.Lstat(inOther); err != nil {
		t.Fatalf("sweep entered a symlinked dir: %v", err)
	}
}

// The mint API returns the file to open; the CLI hands only its path to the
// opener.
func TestLoginCodeMintReturnsTheFile(t *testing.T) {
	s, ts, dir := loginFileServer(t)
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/login-code", nil)
	req.Header.Set("Authorization", "Bearer "+s.cfg.Token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var body struct{ Code, File string }
	if err := jsonDecode(res, &body); err != nil || filepath.Dir(body.File) != dir || len(body.Code) != 32 {
		t.Fatalf("mint: %v %+v", err, body)
	}
}

// A second presentation of a code that was already redeemed means two parties
// held it: the user's browser lost a race. That is logged as a warning,
// without the code.
func TestReusedLoginCodeLogsARaceWarning(t *testing.T) {
	var buf bytes.Buffer
	old := log.Writer()
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(old) })

	s, ts, _ := loginServer(t)
	code := s.NewLoginCode()
	login(t, ts, code)
	if strings.Contains(buf.String(), "WARNING") {
		t.Fatalf("first redemption warned: %q", buf.String())
	}
	login(t, ts, code)
	if !strings.Contains(buf.String(), "WARNING") || strings.Contains(buf.String(), code) {
		t.Fatalf("log %q: want a WARNING without the code", buf.String())
	}
}

// Linux defence in depth: a loopback peer owned by another user is refused,
// and the refusal does not burn the code, so the user's browser still wins.
func TestLoginRefusesAPeerOfAnotherUser(t *testing.T) {
	s, ts, _ := loginServer(t)
	s.selfUID = 1000
	s.peerUID = func(*http.Request) (int, bool) { return 1001, true }
	code := s.NewLoginCode()
	if res, _ := login(t, ts, code); res.StatusCode != http.StatusForbidden {
		t.Fatalf("other user's peer: %d, want 403", res.StatusCode)
	}
	for _, uid := range []int{1000, 0} {
		s.peerUID = func(*http.Request) (int, bool) { return uid, true }
		c := s.NewLoginCode()
		if res, _ := login(t, ts, c); res.StatusCode != http.StatusFound {
			t.Fatalf("peer uid %d: %d, want 302", uid, res.StatusCode)
		}
	}
	s.peerUID = func(*http.Request) (int, bool) { return 0, false }
	if res, _ := login(t, ts, code); res.StatusCode != http.StatusFound {
		t.Fatalf("unknown peer uid (best effort) with the unburned code: %d, want 302", res.StatusCode)
	}
}

func TestSocketUIDFromProcNetTCP(t *testing.T) {
	if binary.NativeEndian.Uint16([]byte{1, 0}) != 1 {
		t.Skip("fixture is in little-endian /proc format")
	}
	tcp := []byte(`  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode
   0: 0100007F:2257 00000000:0000 0A 00000000:00000000 00:00000000 00000000  1000        0 1 1 0000000000000000 100 0 0 10 0
   1: 0100007F:D431 0100007F:2257 01 00000000:00000000 00:00000000 00000000  1001        0 2 1 0000000000000000 20 4 30 10 -1
   2: 0100007F:2257 0100007F:D431 01 00000000:00000000 00:00000000 00000000  1000        0 3 1 0000000000000000 20 4 30 10 -1
`)
	peer := netip.MustParseAddrPort("127.0.0.1:54321")
	server := netip.MustParseAddrPort("127.0.0.1:8791")
	if uid, ok := socketUID(tcp, false, peer, server); !ok || uid != 1001 {
		t.Fatalf("socketUID = %d %v, want 1001 (the client's row, not the server's)", uid, ok)
	}
	if _, ok := socketUID(tcp, false, netip.MustParseAddrPort("127.0.0.1:1"), server); ok {
		t.Fatal("found a row for a connection that is not there")
	}
	tcp6 := []byte(`  sl  local_address                         remote_address                        st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode
   0: 0000000000000000FFFF00000100007F:D431 0000000000000000FFFF00000100007F:2257 01 00000000:00000000 00:00000000 00000000  1002        0 4 1
`)
	if uid, ok := socketUID(tcp6, true, peer, server); !ok || uid != 1002 {
		t.Fatalf("v4-mapped tcp6 row: %d %v, want 1002", uid, ok)
	}
}
