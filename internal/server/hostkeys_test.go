package server

import (
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/fstest"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/executor"
)

// testSSHD answers the SSH handshake with a fresh host key and nothing else.
func testSSHD(t *testing.T) (string, int, ssh.PublicKey) {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := ssh.NewSignerFromKey(priv)
	if err != nil {
		t.Fatal(err)
	}
	sc := &ssh.ServerConfig{NoClientAuth: true}
	sc.AddHostKey(signer)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func() {
				defer c.Close()
				if conn, _, _, err := ssh.NewServerConn(c, sc); err == nil {
					conn.Close()
				}
			}()
		}
	}()
	a := ln.Addr().(*net.TCPAddr)
	return a.IP.String(), a.Port, signer.PublicKey()
}

// otherKey is a host key no test server presents.
func otherKey(t *testing.T) ssh.PublicKey {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	s, err := ssh.NewSignerFromKey(priv)
	if err != nil {
		t.Fatal(err)
	}
	return s.PublicKey()
}

func postJSON(t *testing.T, url, token string, in, out any) *http.Response {
	t.Helper()
	b, _ := json.Marshal(in)
	req, _ := http.NewRequest(http.MethodPost, url, strings.NewReader(string(b)))
	req.Header.Set("Authorization", "Bearer "+token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if out != nil {
		_ = json.NewDecoder(res.Body).Decode(out)
	}
	return res
}

func TestProbeThenConfirmRecordsExactlyTheProbedKey(t *testing.T) {
	ts, token := contractServer(t)
	host, port, key := testSSHD(t)
	addr := api.HostKeyProbeRequest{SSH: api.SSHView{Host: host, Port: port, User: "root"}}

	var p api.HostKeyProbe
	postJSON(t, ts.URL+"/api/hostkeys/probe", token, addr, &p)
	hop := p.Pending()
	if p.AllConfirmed || hop == nil || hop.State != api.HostKeyUnknown || hop.ProbeID == "" || hop.Fingerprint != executor.Fingerprint(key) {
		t.Fatalf("probe %+v", p)
	}
	if !strings.HasPrefix(hop.Fingerprint, "SHA256:") || hop.KeyType != ssh.KeyAlgoED25519 {
		t.Fatalf("fingerprint %q type %q: want OpenSSH's SHA256 form", hop.Fingerprint, hop.KeyType)
	}

	var e api.Error
	if res := postJSON(t, ts.URL+"/api/hostkeys/confirm", token, api.HostKeyConfirm{ProbeID: hop.ProbeID, Fingerprint: "SHA256:wrong"}, &e); res.StatusCode != http.StatusConflict || e.Code != api.CodeFingerprintMismatch {
		t.Fatalf("wrong fingerprint: %d %+v", res.StatusCode, e)
	}
	confirmed, _ := config.ConfirmedHostsFile()
	if _, err := os.Stat(confirmed); err == nil {
		t.Fatal("a mismatched confirmation wrote the confirmed store")
	}
	// A wrong guess spends the probe: the right fingerprint is not accepted
	// on a second try against the same id.
	if res := postJSON(t, ts.URL+"/api/hostkeys/confirm", token, api.HostKeyConfirm{ProbeID: hop.ProbeID, Fingerprint: hop.Fingerprint}, &e); res.StatusCode != http.StatusGone || e.Code != api.CodeProbeExpired {
		t.Fatalf("a probe id survived a wrong fingerprint: %d %+v", res.StatusCode, e)
	}

	var again api.HostKeyProbe
	postJSON(t, ts.URL+"/api/hostkeys/probe", token, addr, &again)
	hop = again.Pending()
	if res := postJSON(t, ts.URL+"/api/hostkeys/confirm", token, api.HostKeyConfirm{ProbeID: hop.ProbeID, Fingerprint: hop.Fingerprint}, nil); res.StatusCode != http.StatusNoContent {
		t.Fatalf("confirm: %d", res.StatusCode)
	}
	if res := postJSON(t, ts.URL+"/api/hostkeys/confirm", token, api.HostKeyConfirm{ProbeID: hop.ProbeID, Fingerprint: hop.Fingerprint}, &e); res.StatusCode != http.StatusGone || e.Code != api.CodeProbeExpired {
		t.Fatalf("a probe id worked twice: %d %+v", res.StatusCode, e)
	}
	b, _ := os.ReadFile(confirmed)
	if !strings.Contains(string(b), net.JoinHostPort(host, strconv.Itoa(port))) {
		t.Fatalf("confirmed_hosts = %q", b)
	}
	var after api.HostKeyProbe
	postJSON(t, ts.URL+"/api/hostkeys/probe", token, addr, &after)
	if !after.AllConfirmed || after.Hops[0].State != api.HostKeyConfirmed || after.Hops[0].ProbeID != "" {
		t.Fatalf("after confirm: %+v", after)
	}
	// The TOFU store is never written by a probe or a confirmation.
	if _, err := os.Stat(filepath.Join(filepath.Dir(confirmed), "known_hosts")); err == nil {
		t.Fatal("known_hosts (TOFU) was written")
	}
}

func TestProbeIDsExpire(t *testing.T) {
	old := hostKeyProbeTTL
	hostKeyProbeTTL = time.Millisecond
	t.Cleanup(func() { hostKeyProbeTTL = old })
	ts, token := contractServer(t)
	host, port, _ := testSSHD(t)
	var p api.HostKeyProbe
	postJSON(t, ts.URL+"/api/hostkeys/probe", token, api.HostKeyProbeRequest{SSH: api.SSHView{Host: host, Port: port, User: "root"}}, &p)
	if p.Pending() == nil {
		t.Fatalf("probe %+v", p)
	}
	time.Sleep(10 * time.Millisecond)
	var e api.Error
	if res := postJSON(t, ts.URL+"/api/hostkeys/confirm", token, api.HostKeyConfirm{ProbeID: p.Pending().ProbeID, Fingerprint: p.Pending().Fingerprint}, &e); res.StatusCode != http.StatusGone || e.Code != api.CodeProbeExpired {
		t.Fatalf("expired probe: %d %+v", res.StatusCode, e)
	}
	confirmed, _ := config.ConfirmedHostsFile()
	if _, err := os.Stat(confirmed); err == nil {
		t.Fatal("an expired probe wrote the confirmed store")
	}
}

// The target is only reached through a confirmed jump host, so a probe stops
// at an unconfirmed jump.
func TestProbeStopsAtAnUnconfirmedJump(t *testing.T) {
	ts, token := contractServer(t)
	jh, jp, _ := testSSHD(t)
	var p api.HostKeyProbe
	postJSON(t, ts.URL+"/api/hostkeys/probe", token, api.HostKeyProbeRequest{SSH: api.SSHView{Host: "10.255.255.1", Port: 22, User: "root", Jump: &api.SSHView{Host: jh, Port: jp, User: "ops"}}}, &p)
	if len(p.Hops) != 1 || p.Hops[0].HostPort != net.JoinHostPort(jh, strconv.Itoa(jp)) || p.Hops[0].State != api.HostKeyUnknown || p.AllConfirmed {
		t.Fatalf("probe %+v", p)
	}
}

// Once the jump host is confirmed, the target is probed through it. This fake
// jump speaks no direct-tcpip, so the target is unreachable: reported as such,
// never as a key to trust.
func TestProbeReachesTheTargetOnlyThroughAConfirmedJump(t *testing.T) {
	t.Setenv("SSH_AUTH_SOCK", "") // the dial through the jump must not reach the real ssh-agent
	ts, token := contractServer(t)
	jh, jp, jkey := testSSHD(t)
	confirmed, _ := config.ConfirmedHostsFile()
	if err := executor.RecordHostKey(confirmed, net.JoinHostPort(jh, strconv.Itoa(jp)), jkey); err != nil {
		t.Fatal(err)
	}
	var p api.HostKeyProbe
	postJSON(t, ts.URL+"/api/hostkeys/probe", token, api.HostKeyProbeRequest{SSH: api.SSHView{Host: "10.255.255.1", Port: 22, User: "root", Jump: &api.SSHView{Host: jh, Port: jp, User: "ops"}}}, &p)
	if len(p.Hops) != 2 || p.Hops[0].State != api.HostKeyConfirmed || p.Hops[1].State != api.HostKeyUnreachable || p.Hops[1].ProbeID != "" || p.AllConfirmed {
		t.Fatalf("probe %+v", p)
	}
}

func TestProbeReportsAMismatchAndOffersNoProbeID(t *testing.T) {
	ts, token := contractServer(t)
	host, port, _ := testSSHD(t)
	confirmed, _ := config.ConfirmedHostsFile()
	if err := executor.RecordHostKey(confirmed, net.JoinHostPort(host, strconv.Itoa(port)), otherKey(t)); err != nil {
		t.Fatal(err)
	}
	var p api.HostKeyProbe
	postJSON(t, ts.URL+"/api/hostkeys/probe", token, api.HostKeyProbeRequest{SSH: api.SSHView{Host: host, Port: port, User: "root"}}, &p)
	if h := p.Pending(); h == nil || h.State != api.HostKeyMismatch || h.ProbeID != "" || h.Error == "" {
		t.Fatalf("probe %+v", p)
	}
}

// A different key recorded for the host between the probe and the
// confirmation (another window, the CLI) is a mismatch at confirm time: the
// held key is never appended beside it and the store is left as it was.
func TestConfirmNeverOverwritesADifferentKeyOnRecord(t *testing.T) {
	ts, token := contractServer(t)
	host, port, _ := testSSHD(t)
	hp := net.JoinHostPort(host, strconv.Itoa(port))
	var p api.HostKeyProbe
	postJSON(t, ts.URL+"/api/hostkeys/probe", token, api.HostKeyProbeRequest{SSH: api.SSHView{Host: host, Port: port, User: "root"}}, &p)
	hop := p.Pending()
	if hop == nil || hop.ProbeID == "" {
		t.Fatalf("probe %+v", p)
	}
	confirmed, _ := config.ConfirmedHostsFile()
	if err := executor.RecordHostKey(confirmed, hp, otherKey(t)); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(confirmed)

	var e api.Error
	if res := postJSON(t, ts.URL+"/api/hostkeys/confirm", token, api.HostKeyConfirm{ProbeID: hop.ProbeID, Fingerprint: hop.Fingerprint}, &e); res.StatusCode != http.StatusConflict || e.Code != api.CodeHostKey {
		t.Fatalf("confirm over a different key: %d %+v", res.StatusCode, e)
	}
	if after, _ := os.ReadFile(confirmed); string(after) != string(before) {
		t.Fatalf("confirmed_hosts changed:\nbefore %q\nafter  %q", before, after)
	}
}

// Two probes of the same unconfirmed host, both confirmed: the key is
// recorded once.
func TestConfirmingTheSameKeyTwiceRecordsItOnce(t *testing.T) {
	ts, token := contractServer(t)
	host, port, _ := testSSHD(t)
	addr := api.HostKeyProbeRequest{SSH: api.SSHView{Host: host, Port: port, User: "root"}}
	var p1, p2 api.HostKeyProbe
	postJSON(t, ts.URL+"/api/hostkeys/probe", token, addr, &p1)
	postJSON(t, ts.URL+"/api/hostkeys/probe", token, addr, &p2)
	for _, p := range []api.HostKeyProbe{p1, p2} {
		h := p.Pending()
		if h == nil {
			t.Fatalf("probe %+v", p)
		}
		if res := postJSON(t, ts.URL+"/api/hostkeys/confirm", token, api.HostKeyConfirm{ProbeID: h.ProbeID, Fingerprint: h.Fingerprint}, nil); res.StatusCode != http.StatusNoContent {
			t.Fatalf("confirm: %d", res.StatusCode)
		}
	}
	confirmed, _ := config.ConfirmedHostsFile()
	b, _ := os.ReadFile(confirmed)
	if n := strings.Count(string(b), net.JoinHostPort(host, strconv.Itoa(port))); n != 1 {
		t.Fatalf("recorded %d times: %q", n, b)
	}
}

func TestProbeRejectsAnIncompleteAddress(t *testing.T) {
	ts, token := contractServer(t)
	for name, req := range map[string]api.HostKeyProbeRequest{
		"no host":      {SSH: api.SSHView{User: "root"}},
		"no user":      {SSH: api.SSHView{Host: "h"}},
		"jump no host": {SSH: api.SSHView{Host: "h", User: "root", Jump: &api.SSHView{User: "ops"}}},
	} {
		var e api.Error
		if res := postJSON(t, ts.URL+"/api/hostkeys/probe", token, req, &e); res.StatusCode != http.StatusBadRequest || e.Code != api.CodeBadRequest {
			t.Errorf("%s: %d %+v", name, res.StatusCode, e)
		}
	}
}

// Both routes need the session token, and a cookie-authenticated request
// from another origin is refused before the handler runs.
func TestHostKeyRoutesNeedTheTokenAndThisOrigin(t *testing.T) {
	ts, token := contractServer(t)
	for _, path := range []string{"/api/hostkeys/probe", "/api/hostkeys/confirm"} {
		res, e := do(t, ts, "", http.MethodPost, path, `{}`)
		if res.StatusCode != http.StatusUnauthorized || e.Code != api.CodeUnauthorized {
			t.Errorf("%s without a token: %d %+v", path, res.StatusCode, e)
		}
		req, _ := http.NewRequest(http.MethodPost, ts.URL+path, strings.NewReader(`{}`))
		req.AddCookie(&http.Cookie{Name: cookieName, Value: token})
		req.Header.Set("Origin", "http://evil.example")
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		r.Body.Close()
		if r.StatusCode != http.StatusForbidden {
			t.Errorf("%s cross-origin: %d", path, r.StatusCode)
		}
	}
}

// ---- fix round 1: probe bounds (M1) and forget (Ruling T6b) ----

// hostKeyServer is contractServer that also returns the *Server, for tests
// that reach into its probe store.
func hostKeyServer(t *testing.T) (*httptest.Server, string, *Server) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	token := NewSessionToken()
	s := New(Config{Token: token, UI: fstest.MapFS{}})
	ts := httptest.NewServer(s.Handler())
	t.Cleanup(ts.Close)
	return ts, token, s
}

func TestProbesAreCappedWith429(t *testing.T) {
	old := maxLiveProbes
	maxLiveProbes = 2
	t.Cleanup(func() { maxLiveProbes = old })
	ts, token := contractServer(t)
	host, port, _ := testSSHD(t)
	addr := api.HostKeyProbeRequest{SSH: api.SSHView{Host: host, Port: port, User: "root"}}
	for i := 0; i < 2; i++ {
		var p api.HostKeyProbe
		if res := postJSON(t, ts.URL+"/api/hostkeys/probe", token, addr, &p); res.StatusCode != http.StatusOK || p.Pending() == nil || p.Pending().ProbeID == "" {
			t.Fatalf("probe %d: %d %+v", i, res.StatusCode, p)
		}
	}
	var e api.Error
	if res := postJSON(t, ts.URL+"/api/hostkeys/probe", token, addr, &e); res.StatusCode != http.StatusTooManyRequests || e.Code != api.CodeTooManyProbes {
		t.Fatalf("third probe: %d %+v", res.StatusCode, e)
	}
}

// A probe waits for a free dial slot and gives up with its request: when
// every slot is taken, no connection is made.
func TestProbeDialsAreBounded(t *testing.T) {
	ts, token, s := hostKeyServer(t)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	var accepted atomic.Int32
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			accepted.Add(1)
			c.Close()
		}
	}()
	a := ln.Addr().(*net.TCPAddr)
	body := `{"ssh":{"Host":"127.0.0.1","Port":` + strconv.Itoa(a.Port) + `,"User":"root"}}`

	for i := 0; i < cap(s.probes.dials); i++ {
		s.probes.dials <- struct{}{}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, ts.URL+"/api/hostkeys/probe", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	if res, err := http.DefaultClient.Do(req); err == nil {
		res.Body.Close()
		t.Fatalf("a probe ran with every dial slot taken: %d", res.StatusCode)
	}
	if n := accepted.Load(); n != 0 {
		t.Fatalf("%d connections with every dial slot taken", n)
	}
	for i := 0; i < cap(s.probes.dials); i++ {
		<-s.probes.dials
	}
	res, _ := do(t, ts, token, http.MethodPost, "/api/hostkeys/probe", body)
	if res.StatusCode != http.StatusOK || accepted.Load() == 0 {
		t.Fatalf("after freeing the slots: %d, %d connections", res.StatusCode, accepted.Load())
	}
}

// signerKey is a fresh host key of the given kind.
func signerKey(t *testing.T, kind string) ssh.PublicKey {
	t.Helper()
	var priv any
	var err error
	switch kind {
	case "ecdsa":
		priv, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	default:
		_, priv, err = ed25519.GenerateKey(rand.Reader)
	}
	if err != nil {
		t.Fatal(err)
	}
	s, err := ssh.NewSignerFromKey(priv)
	if err != nil {
		t.Fatal(err)
	}
	return s.PublicKey()
}

func forget(t *testing.T, ts *httptest.Server, token, hp, fp string) (*http.Response, api.Error) {
	t.Helper()
	var e api.Error
	res := postJSON(t, ts.URL+"/api/hostkeys/forget", token, api.HostKeyForget{HostPort: hp, Fingerprint: fp}, &e)
	return res, e
}

func TestRecordedHostKeysListsBothStores(t *testing.T) {
	ts, token := contractServer(t)
	hp := "203.0.113.5:22"
	k1, k2 := signerKey(t, "ed25519"), signerKey(t, "ecdsa")
	confirmed, _ := config.ConfirmedHostsFile()
	if err := executor.RecordHostKey(confirmed, hp, k1); err != nil {
		t.Fatal(err)
	}
	home, _ := os.UserHomeDir()
	writeKnownHosts(t, home, "203.0.113.5 "+strings.TrimSpace(string(ssh.MarshalAuthorizedKey(k2))))
	res, err := http.NewRequest(http.MethodGet, ts.URL+"/api/hostkeys?hostPort="+hp, nil)
	if err != nil {
		t.Fatal(err)
	}
	res.Header.Set("Authorization", "Bearer "+token)
	r, err := http.DefaultClient.Do(res)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Body.Close()
	var got api.RecordedHostKeys
	_ = json.NewDecoder(r.Body).Decode(&got)
	if r.StatusCode != http.StatusOK || len(got.Keys) != 2 ||
		got.Keys[0] != (api.RecordedHostKey{Fingerprint: executor.Fingerprint(k1), KeyType: k1.Type(), Store: api.HostKeyStoreJumpgate}) ||
		got.Keys[1] != (api.RecordedHostKey{Fingerprint: executor.Fingerprint(k2), KeyType: k2.Type(), Store: api.HostKeyStoreOpenSSH}) {
		t.Fatalf("%d %+v", r.StatusCode, got)
	}
}

// Forget removes the one line whose key has the typed fingerprint; the
// host's other key types and other hosts stay.
func TestForgetRemovesOneKeyAndKeepsTheRest(t *testing.T) {
	ts, token := contractServer(t)
	hp := "203.0.113.5:22"
	k1, k2, k3 := signerKey(t, "ed25519"), signerKey(t, "ecdsa"), signerKey(t, "ed25519")
	confirmed, _ := config.ConfirmedHostsFile()
	for _, r := range []struct {
		hp  string
		key ssh.PublicKey
	}{{hp, k1}, {hp, k2}, {"198.51.100.1:22", k3}} {
		if err := executor.RecordHostKey(confirmed, r.hp, r.key); err != nil {
			t.Fatal(err)
		}
	}
	if res, e := forget(t, ts, token, hp, executor.Fingerprint(k1)); res.StatusCode != http.StatusNoContent {
		t.Fatalf("forget: %d %+v", res.StatusCode, e)
	}
	b, _ := os.ReadFile(confirmed)
	got := string(b)
	if strings.Contains(got, base64Key(k1)) || !strings.Contains(got, base64Key(k2)) || !strings.Contains(got, base64Key(k3)) {
		t.Fatalf("confirmed_hosts after forget:\n%s", got)
	}
	if runtime.GOOS != "windows" {
		if fi, err := os.Stat(confirmed); err != nil || fi.Mode().Perm() != 0o600 {
			t.Fatalf("confirmed_hosts mode %v, %v", fi.Mode(), err)
		}
	}
}

func base64Key(k ssh.PublicKey) string {
	return strings.Fields(string(ssh.MarshalAuthorizedKey(k)))[1]
}

func TestForgetRefusals(t *testing.T) {
	ts, token := contractServer(t)
	hp := "203.0.113.5:22"
	k1 := signerKey(t, "ed25519")
	confirmed, _ := config.ConfirmedHostsFile()
	if err := executor.RecordHostKey(confirmed, hp, k1); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(confirmed)

	// A fingerprint that is not the one on record removes nothing.
	if res, e := forget(t, ts, token, hp, executor.Fingerprint(signerKey(t, "ed25519"))); res.StatusCode != http.StatusConflict || e.Code != api.CodeFingerprintMismatch {
		t.Fatalf("wrong fingerprint: %d %+v", res.StatusCode, e)
	}
	// A host with nothing on record anywhere.
	if res, e := forget(t, ts, token, "198.51.100.9:22", executor.Fingerprint(k1)); res.StatusCode != http.StatusNotFound || e.Code != api.CodeNotFound {
		t.Fatalf("unknown host: %d %+v", res.StatusCode, e)
	}
	// Malformed requests.
	for _, req := range []api.HostKeyForget{{HostPort: hp}, {Fingerprint: executor.Fingerprint(k1)}, {HostPort: "no-port", Fingerprint: executor.Fingerprint(k1)}} {
		if res, e := forget(t, ts, token, req.HostPort, req.Fingerprint); res.StatusCode != http.StatusBadRequest {
			t.Errorf("%+v: %d %+v", req, res.StatusCode, e)
		}
	}
	if after, _ := os.ReadFile(confirmed); string(after) != string(before) {
		t.Fatalf("confirmed_hosts changed:\n%q\n%q", before, after)
	}
}

// A key only in the operator's OpenSSH known_hosts is not jumpgate's to
// remove: a clear error, and known_hosts is never touched.
func TestForgetLeavesOpenSSHKnownHostsAlone(t *testing.T) {
	ts, token := contractServer(t)
	k := signerKey(t, "ed25519")
	home, _ := os.UserHomeDir()
	line := "203.0.113.5 " + strings.TrimSpace(string(ssh.MarshalAuthorizedKey(k)))
	writeKnownHosts(t, home, line)
	kh, content := filepath.Join(home, ".ssh", "known_hosts"), line+"\n"
	res, e := forget(t, ts, token, "203.0.113.5:22", executor.Fingerprint(k))
	if res.StatusCode != http.StatusConflict || e.Code != api.CodeHostKeyNotOurs || !strings.Contains(e.Message, "known_hosts") {
		t.Fatalf("forget of an OpenSSH key: %d %+v", res.StatusCode, e)
	}
	if b, _ := os.ReadFile(kh); string(b) != content {
		t.Fatalf("known_hosts changed: %q", b)
	}
	confirmed, _ := config.ConfirmedHostsFile()
	if _, err := os.Stat(confirmed); err == nil {
		t.Fatal("confirmed_hosts was created")
	}
}

// A probe issued before a forget cannot be confirmed after it: forgetting a
// key drops every held probe for that host.
func TestStaleProbeIsRejectedAfterAForget(t *testing.T) {
	ts, token := contractServer(t)
	host, port, key := testSSHD(t)
	hp := net.JoinHostPort(host, strconv.Itoa(port))
	addr := api.HostKeyProbeRequest{SSH: api.SSHView{Host: host, Port: port, User: "root"}}
	var stale, fresh api.HostKeyProbe
	postJSON(t, ts.URL+"/api/hostkeys/probe", token, addr, &stale)
	postJSON(t, ts.URL+"/api/hostkeys/probe", token, addr, &fresh)
	if res := postJSON(t, ts.URL+"/api/hostkeys/confirm", token, api.HostKeyConfirm{ProbeID: fresh.Pending().ProbeID, Fingerprint: fresh.Pending().Fingerprint}, nil); res.StatusCode != http.StatusNoContent {
		t.Fatalf("confirm: %d", res.StatusCode)
	}
	if res, e := forget(t, ts, token, hp, executor.Fingerprint(key)); res.StatusCode != http.StatusNoContent {
		t.Fatalf("forget: %d %+v", res.StatusCode, e)
	}
	var e api.Error
	if res := postJSON(t, ts.URL+"/api/hostkeys/confirm", token, api.HostKeyConfirm{ProbeID: stale.Pending().ProbeID, Fingerprint: stale.Pending().Fingerprint}, &e); res.StatusCode != http.StatusGone || e.Code != api.CodeProbeExpired {
		t.Fatalf("stale probe after forget: %d %+v", res.StatusCode, e)
	}
	confirmed, _ := config.ConfirmedHostsFile()
	if b, _ := os.ReadFile(confirmed); strings.Contains(string(b), hp) {
		t.Fatalf("the stale probe re-recorded the key: %q", b)
	}
}

// A forget racing a confirmation of an older probe: whichever runs first,
// the key is not on record once both are done.
func TestForgetThenConfirmRace(t *testing.T) {
	ts, token := contractServer(t)
	host, port, key := testSSHD(t)
	hp := net.JoinHostPort(host, strconv.Itoa(port))
	addr := api.HostKeyProbeRequest{SSH: api.SSHView{Host: host, Port: port, User: "root"}}
	confirmed, _ := config.ConfirmedHostsFile()
	for i := 0; i < 10; i++ {
		var older, first api.HostKeyProbe
		postJSON(t, ts.URL+"/api/hostkeys/probe", token, addr, &older)
		postJSON(t, ts.URL+"/api/hostkeys/probe", token, addr, &first)
		if older.Pending() == nil || first.Pending() == nil {
			t.Fatalf("round %d: probes %+v %+v", i, older, first)
		}
		if res := postJSON(t, ts.URL+"/api/hostkeys/confirm", token, api.HostKeyConfirm{ProbeID: first.Pending().ProbeID, Fingerprint: first.Pending().Fingerprint}, nil); res.StatusCode != http.StatusNoContent {
			t.Fatalf("round %d: confirm %d", i, res.StatusCode)
		}
		var wg sync.WaitGroup
		var forgetStatus, confirmStatus int
		wg.Add(2)
		go func() {
			defer wg.Done()
			res, _ := forget(t, ts, token, hp, executor.Fingerprint(key))
			forgetStatus = res.StatusCode
		}()
		go func() {
			defer wg.Done()
			res := postJSON(t, ts.URL+"/api/hostkeys/confirm", token, api.HostKeyConfirm{ProbeID: older.Pending().ProbeID, Fingerprint: older.Pending().Fingerprint}, nil)
			confirmStatus = res.StatusCode
		}()
		wg.Wait()
		if forgetStatus != http.StatusNoContent || (confirmStatus != http.StatusNoContent && confirmStatus != http.StatusGone) {
			t.Fatalf("round %d: forget %d, confirm %d", i, forgetStatus, confirmStatus)
		}
		if b, _ := os.ReadFile(confirmed); strings.Contains(string(b), hp) {
			t.Fatalf("round %d: key on record after forget and confirm (%d): %q", i, confirmStatus, b)
		}
	}
}

func TestForgetAndRecordedNeedTheTokenAndThisOrigin(t *testing.T) {
	ts, token := contractServer(t)
	for _, c := range []struct{ method, path string }{{http.MethodPost, "/api/hostkeys/forget"}, {http.MethodGet, "/api/hostkeys?hostPort=h:22"}} {
		if res, e := do(t, ts, "", c.method, c.path, `{}`); res.StatusCode != http.StatusUnauthorized || e.Code != api.CodeUnauthorized {
			t.Errorf("%s without a token: %d %+v", c.path, res.StatusCode, e)
		}
	}
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/hostkeys/forget", strings.NewReader(`{}`))
	req.AddCookie(&http.Cookie{Name: cookieName, Value: token})
	req.Header.Set("Origin", "http://evil.example")
	r, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	r.Body.Close()
	if r.StatusCode != http.StatusForbidden {
		t.Errorf("forget cross-origin: %d", r.StatusCode)
	}
}
