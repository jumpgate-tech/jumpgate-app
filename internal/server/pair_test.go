package server

import (
	"bufio"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"encoding/pem"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"testing/fstest"

	gliderssh "github.com/gliderlabs/ssh"
	"golang.org/x/crypto/ssh"

	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/executor"
	"github.com/valve-tech/jumpgate/internal/signer"
)

type pairTestSSHD struct {
	host    string
	port    int
	hostKey ssh.PublicKey
	keyPath string // a client key the server accepts
}

// startPairTestSSHD is a cut-down copy of executor's startTestSSHD. Every
// session runs handler; a nil handler answers every command with "FreeBSD",
// so a pairing that gets that far fails its Linux preflight without anything
// running on this machine.
func startPairTestSSHD(t *testing.T, handler gliderssh.Handler) pairTestSSHD {
	t.Helper()
	if handler == nil {
		handler = func(s gliderssh.Session) {
			_, _ = io.WriteString(s, "FreeBSD\n")
			_ = s.Exit(0)
		}
	}
	_, hostPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	hostSigner, err := ssh.NewSignerFromKey(hostPriv)
	if err != nil {
		t.Fatal(err)
	}
	_, clientPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	block, err := ssh.MarshalPrivateKey(clientPriv, "")
	if err != nil {
		t.Fatal(err)
	}
	keyPath := filepath.Join(t.TempDir(), "id_ed25519")
	if err := os.WriteFile(keyPath, pem.EncodeToMemory(block), 0o600); err != nil {
		t.Fatal(err)
	}

	srv := &gliderssh.Server{
		PublicKeyHandler: func(gliderssh.Context, gliderssh.PublicKey) bool { return true },
		Handler:          handler,
	}
	srv.AddHostKey(hostSigner)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Close() })
	return pairTestSSHD{host: "127.0.0.1", port: ln.Addr().(*net.TCPAddr).Port, hostKey: hostSigner.PublicKey(), keyPath: keyPath}
}

// shortHome is a temporary HOME short enough for unix socket paths.
func shortHome(t *testing.T) string {
	t.Helper()
	home, err := os.MkdirTemp("/tmp", "jgp")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(home) })
	return home
}

// pairServer saves an ssh target for d and starts a server with a controller
// key.
func pairServer(t *testing.T, d pairTestSSHD) (*httptest.Server, string) {
	t.Helper()
	t.Setenv("HOME", shortHome(t))
	_, err := config.Update(func(c *config.Config) error {
		c.Targets = append(c.Targets, config.Target{ID: "box", Mode: "ssh",
			SSH: &executor.SSHConfig{Host: d.host, Port: d.port, User: "root", KeyPath: d.keyPath}})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	ctrl, _ := signer.GenerateKey()
	token := NewSessionToken()
	ts := httptest.NewServer(New(Config{Token: token, UI: fstest.MapFS{}, Signer: ctrl}).Handler())
	t.Cleanup(ts.Close)
	return ts, token
}

func postPair(t *testing.T, ts *httptest.Server, token, body string) *http.Response {
	t.Helper()
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/targets/box/pair", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { res.Body.Close() })
	return res
}

func (d pairTestSSHD) hostname() string { return net.JoinHostPort(d.host, strconv.Itoa(d.port)) }

// A host whose key nobody confirmed is refused before anything runs on it,
// with the fingerprint the CLI shows the operator.
func TestPairRefusesAnUnconfirmedHost(t *testing.T) {
	var ran atomic.Bool
	d := startPairTestSSHD(t, func(s gliderssh.Session) { ran.Store(true); _ = s.Exit(0) })
	ts, token := pairServer(t, d)

	// Trust-on-first-use having seen the key is not a confirmation.
	if err := executor.RecordHostKey(filepath.Join(os.Getenv("HOME"), ".jumpgate", "known_hosts"), d.hostname(), d.hostKey); err != nil {
		t.Fatal(err)
	}

	res := postPair(t, ts, token, `{}`)
	if res.StatusCode != http.StatusConflict {
		t.Fatalf("status %d, want 409", res.StatusCode)
	}
	line, _ := bufio.NewReader(res.Body).ReadString('\n')
	if !strings.Contains(line, "unknown_host") || !strings.Contains(line, "SHA256:") {
		t.Fatalf("body %q", line)
	}
	var out map[string]string
	_ = json.Unmarshal([]byte(line), &out)
	if out["fingerprint"] != executor.Fingerprint(d.hostKey) || out["host"] != d.hostname() {
		t.Fatalf("fingerprint/host %v, want %s at %s", out, executor.Fingerprint(d.hostKey), d.hostname())
	}
	if ran.Load() {
		t.Fatal("a command ran on an unconfirmed host")
	}
	c, _ := config.Load()
	if c.Targets[0].Agent != nil {
		t.Fatal("an unconfirmed host was recorded as paired")
	}
}

// A confirmed host gets a stream: each bootstrap step as it runs, then the
// failing step's error. Nothing is recorded as paired.
func TestPairStreamsStepsAndReportsTheFailingStep(t *testing.T) {
	d := startPairTestSSHD(t, nil)
	ts, token := pairServer(t, d)
	confirmed, _ := config.ConfirmedHostsFile()
	if err := executor.RecordHostKey(confirmed, d.hostname(), d.hostKey); err != nil {
		t.Fatal(err)
	}

	res := postPair(t, ts, token, `{"sudo":false}`)
	if res.StatusCode != http.StatusOK || !strings.HasPrefix(res.Header.Get("Content-Type"), "text/event-stream") {
		t.Fatalf("status %d %q", res.StatusCode, res.Header.Get("Content-Type"))
	}
	var events []map[string]any
	sc := bufio.NewScanner(res.Body)
	for sc.Scan() {
		if data, ok := strings.CutPrefix(sc.Text(), "data: "); ok {
			var ev map[string]any
			if err := json.Unmarshal([]byte(data), &ev); err != nil {
				t.Fatalf("bad event %q: %v", data, err)
			}
			events = append(events, ev)
		}
	}
	if len(events) < 2 {
		t.Fatalf("events %v", events)
	}
	if events[0]["step"] != "preflight" || events[0]["line"] != "start" {
		t.Errorf("first event %v, want the preflight step starting", events[0])
	}
	last := events[len(events)-1]
	if last["step"] != "preflight" || last["code"] != "step_failed" || !strings.Contains(last["err"].(string), "FreeBSD") {
		t.Errorf("last event %v, want preflight's failure", last)
	}
	c, _ := config.Load()
	if c.Targets[0].Agent != nil {
		t.Fatal("a failed pairing was recorded")
	}
	// The tunnel key exists now, 0600, for the next attempt.
	if fi, err := os.Stat(transportKeyPath()); err != nil || fi.Mode().Perm() != 0o600 {
		t.Fatalf("transport key: %v %v", fi, err)
	}
}

func TestPairRefusesAKeylessServer(t *testing.T) {
	d := startPairTestSSHD(t, nil)
	_, token := pairServer(t, d)

	keyless := httptest.NewServer(New(Config{Token: token, UI: fstest.MapFS{}}).Handler())
	defer keyless.Close()
	if res := postPair(t, keyless, token, `{}`); res.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("keyless: %d", res.StatusCode)
	} else {
		var out map[string]any
		_ = json.NewDecoder(res.Body).Decode(&out)
		if out["code"] != "no_controller_key" {
			t.Fatalf("keyless: %v", out)
		}
	}
}

// Pairing takes the same per-target slot a wipe, reset or clear does.
func TestPairTakesTheTargetSlot(t *testing.T) {
	d := startPairTestSSHD(t, nil)
	t.Setenv("HOME", shortHome(t))
	_, _ = config.Update(func(c *config.Config) error {
		c.Targets = append(c.Targets, config.Target{ID: "box", Mode: "ssh",
			SSH: &executor.SSHConfig{Host: d.host, Port: d.port, User: "root", KeyPath: d.keyPath}})
		return nil
	})
	ctrl, _ := signer.GenerateKey()
	token := NewSessionToken()
	s := New(Config{Token: token, UI: fstest.MapFS{}, Signer: ctrl})
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()

	release, ok := s.claimTargetOp(httptest.NewRecorder(), "box")
	if !ok {
		t.Fatal("could not claim the slot")
	}
	if res := postPair(t, ts, token, `{}`); res.StatusCode != http.StatusConflict {
		t.Fatalf("busy target: %d, want 409", res.StatusCode)
	}
	release()
}

func TestEnsureTransportKeyIsStable(t *testing.T) {
	t.Setenv("HOME", shortHome(t))
	a, err := ensureTransportKey()
	if err != nil {
		t.Fatal(err)
	}
	b, err := ensureTransportKey()
	if err != nil {
		t.Fatal(err)
	}
	if a != b || !strings.HasPrefix(a, "ssh-ed25519 ") || !strings.HasSuffix(a, " jumpgate-controller") {
		t.Fatalf("lines %q / %q", a, b)
	}
	if _, _, _, _, err := ssh.ParseAuthorizedKey([]byte(a)); err != nil {
		t.Fatalf("not an authorized_keys line: %v", err)
	}
}
