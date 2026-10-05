package server

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
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
