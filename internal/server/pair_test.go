package server

import (
	"bufio"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"io"
	"io/fs"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/fstest"
	"time"

	gliderssh "github.com/gliderlabs/ssh"
	"golang.org/x/crypto/ssh"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/executor"
	"github.com/valve-tech/jumpgate/internal/fsperm"
	"github.com/valve-tech/jumpgate/internal/signer"
	"github.com/valve-tech/jumpgate/internal/testutil"
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

// pairServer saves an ssh target for d and starts a server with a controller
// key.
func pairServer(t *testing.T, d pairTestSSHD) (*httptest.Server, string) {
	t.Helper()
	testutil.Home(t)
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
	if last["step"] != "preflight" || last["code"] != "step_failed" || !strings.Contains(last["error"].(string), "FreeBSD") {
		t.Errorf("last event %v, want preflight's failure", last)
	}
	c, _ := config.Load()
	if c.Targets[0].Agent != nil {
		t.Fatal("a failed pairing was recorded")
	}
	// The tunnel key exists now, 0600, for the next attempt.
	testutil.AssertPrivate(t, transportKeyPath())
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
	testutil.Home(t)
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
	testutil.Home(t)
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

// A dangling symlink at the key path is an error, never a reason to generate
// a key (and never a loop).
func TestEnsureTransportKeyRefusesADanglingSymlink(t *testing.T) {
	testutil.RequireUnix(t) // creating a symlink needs a privilege on Windows
	testutil.Home(t)
	path := transportKeyPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(os.Getenv("HOME"), "dotfiles", "missing"), path); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { _, err := ensureTransportKey(); done <- err }()
	select {
	case err := <-done:
		if err == nil || !strings.Contains(err.Error(), path) {
			t.Fatalf("err = %v, want an error naming %s", err, path)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("ensureTransportKey did not return within 5s")
	}
	if fi, err := os.Lstat(path); err != nil || fi.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("the symlink was replaced: %v %v", fi, err)
	}
	if _, err := os.Stat(filepath.Join(os.Getenv("HOME"), "dotfiles", "missing")); !os.IsNotExist(err) {
		t.Fatalf("a key was written through the symlink: %v", err)
	}
}

// Concurrent first pairings all get the same key, and none reads a
// half-written file.
func TestEnsureTransportKeyConcurrentFirstUseAgrees(t *testing.T) {
	testutil.Home(t)
	const n = 16
	lines := make([]string, n)
	errs := make([]error, n)
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			lines[i], errs[i] = ensureTransportKey()
		}(i)
	}
	close(start)
	wg.Wait()
	for i := 0; i < n; i++ {
		if errs[i] != nil {
			t.Fatalf("call %d: %v", i, errs[i])
		}
		if lines[i] != lines[0] {
			t.Fatalf("call %d returned a different key:\n%s\n%s", i, lines[i], lines[0])
		}
	}
	testutil.AssertPrivate(t, filepath.Dir(transportKeyPath()))
	entries, _ := os.ReadDir(filepath.Dir(transportKeyPath()))
	if len(entries) != 1 {
		t.Fatalf("temp files left behind: %v", entries)
	}
}

// An existing key is returned as is, never replaced.
func TestEnsureTransportKeyKeepsAnExistingKey(t *testing.T) {
	testutil.Home(t)
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	block, _ := ssh.MarshalPrivateKey(priv, "")
	path := transportKeyPath()
	_ = os.MkdirAll(filepath.Dir(path), 0o700)
	want := pem.EncodeToMemory(block)
	if err := os.WriteFile(path, want, 0o600); err != nil {
		t.Fatal(err)
	}
	pub, _ := ssh.NewPublicKey(priv.Public())
	line, err := ensureTransportKey()
	if err != nil {
		t.Fatal(err)
	}
	if line != strings.TrimSpace(string(ssh.MarshalAuthorizedKey(pub)))+" jumpgate-controller" {
		t.Fatalf("line %q is not the existing key", line)
	}
	if got, _ := os.ReadFile(path); string(got) != string(want) {
		t.Fatal("the key file was rewritten")
	}
}

// localPairServer saves a local target and starts a server whose OS, euid
// and local executor the test chooses.
func localPairServer(t *testing.T, goos string, euid int, local executor.Executor) (*httptest.Server, string) {
	t.Helper()
	testutil.Home(t)
	if _, err := config.Update(func(c *config.Config) error {
		c.Targets = append(c.Targets, config.Target{ID: "box", Mode: "local"})
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	ctrl, _ := signer.GenerateKey()
	token := NewSessionToken()
	s := New(Config{Token: token, UI: fstest.MapFS{}, Signer: ctrl,
		NewLocalExecutor: func() executor.Executor { return local },
		Geteuid:          func() int { return euid }})
	s.goos = goos
	ts := httptest.NewServer(s.Handler())
	t.Cleanup(ts.Close)
	return ts, token
}

// pairRecordingExec answers every command with "Darwin", so bootstrap stops at
// its Linux preflight, and records what it was asked to run.
type pairRecordingExec struct {
	mu   sync.Mutex
	cmds []string
}

func (r *pairRecordingExec) Run(_ context.Context, cmd string, _ *executor.RunOpts) (executor.Result, error) {
	r.mu.Lock()
	r.cmds = append(r.cmds, cmd)
	r.mu.Unlock()
	return executor.Result{Stdout: "Darwin\n"}, nil
}
func (r *pairRecordingExec) WriteFile(context.Context, string, []byte, fs.FileMode) error { return nil }
func (r *pairRecordingExec) ReadFile(context.Context, string) ([]byte, error)             { return nil, nil }
func (r *pairRecordingExec) Close() error                                                 { return nil }

func (r *pairRecordingExec) commands() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.cmds...)
}

func readCode(t *testing.T, res *http.Response) (int, string, string) {
	t.Helper()
	var e struct{ Code, Hint string }
	_ = json.NewDecoder(res.Body).Decode(&e)
	return res.StatusCode, e.Code, e.Hint
}

func TestPairLocalIsRefusedOffLinux(t *testing.T) {
	rec := &pairRecordingExec{}
	ts, token := localPairServer(t, "darwin", 0, rec)
	status, code, _ := readCode(t, postPair(t, ts, token, `{}`))
	// 409, as on every other route: the request is well formed, this
	// machine cannot honour it.
	if status != http.StatusConflict || code != string(api.CodeLocalUnsupported) {
		t.Fatalf("got %d %q, want 409 local_unsupported", status, code)
	}
	if cmds := rec.commands(); len(cmds) != 0 {
		t.Fatalf("ran commands: %v", cmds)
	}
}

// Off Linux the refusal is the same whatever else the request carries.
func TestPairLocalOffLinuxRefusesBeforeReadingInstalled(t *testing.T) {
	ts, token := localPairServer(t, "windows", 1000, &pairRecordingExec{})
	status, code, _ := readCode(t, postPair(t, ts, token, `{"installed":"me"}`))
	// 409, as on every other route: the request is well formed, this
	// machine cannot honour it.
	if status != http.StatusConflict || code != string(api.CodeLocalUnsupported) {
		t.Fatalf("got %d %q, want 409 local_unsupported", status, code)
	}
}

// D5: the detached server has no terminal, so it never tries sudo for a
// non-root local pairing; it says how to do it instead.
func TestPairLocalNonRootNeedsTheTerminal(t *testing.T) {
	rec := &pairRecordingExec{}
	ts, token := localPairServer(t, "linux", 1000, rec)
	status, code, hint := readCode(t, postPair(t, ts, token, `{}`))
	if status != http.StatusConflict || code != "local_needs_terminal" || !strings.Contains(hint, "jumpgate hosts add") {
		t.Fatalf("got %d %q %q, want 409 local_needs_terminal with a hint", status, code, hint)
	}
	if cmds := rec.commands(); len(cmds) != 0 {
		t.Fatalf("ran commands: %v", cmds)
	}
}

// B-4: as root, pairing runs the steps directly. Stock Debian with a root
// password has no sudo at all.
func TestPairLocalAsRootDoesNotUseSudo(t *testing.T) {
	rec := &pairRecordingExec{}
	ts, token := localPairServer(t, "linux", 0, rec)
	res := postPair(t, ts, token, `{}`)
	_, _ = io.ReadAll(res.Body)
	cmds := rec.commands()
	if len(cmds) == 0 {
		t.Fatal("no command ran")
	}
	for _, c := range cmds {
		if strings.Contains(c, "sudo") {
			t.Fatalf("root pairing used sudo: %q", c)
		}
	}
}

// D18: the CLI ran the privileged steps in the foreground; the server runs
// nothing on the machine and goes straight to verifying the agent it names.
func TestPairLocalInstalledRunsNothingAndVerifies(t *testing.T) {
	rec := &pairRecordingExec{}
	ts, token := localPairServer(t, "linux", 1000, rec)
	res := postPair(t, ts, token, `{"installed":"0x0000000000000000000000000000000000000001"}`)
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d: %s", res.StatusCode, body)
	}
	// No agent listens in the test, so verification is where it stops.
	if !strings.Contains(string(body), "done in the foreground") || !strings.Contains(string(body), `"step":"verify","error"`) {
		t.Fatalf("stream:\n%s", body)
	}
	if cmds := rec.commands(); len(cmds) != 0 {
		t.Fatalf("ran commands: %v", cmds)
	}
}

func TestPairInstalledMustBeAnAddress(t *testing.T) {
	ts, token := localPairServer(t, "linux", 1000, &pairRecordingExec{})
	if res := postPair(t, ts, token, `{"installed":"me"}`); res.StatusCode != http.StatusBadRequest {
		t.Fatalf("status %d, want 400", res.StatusCode)
	}
}

func TestPairInstalledOnlyAppliesToLocalTargets(t *testing.T) {
	d := startPairTestSSHD(t, nil)
	ts, token := pairServer(t, d)
	res := postPair(t, ts, token, `{"installed":"0x0000000000000000000000000000000000000001"}`)
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("status %d, want 400", res.StatusCode)
	}
}

// linuxBoxExec passes bootstrap's preflight as an amd64 systemd box and
// fails every later command, so a pairing gets as far as choosing the agent
// binary and stops at the upload.
type linuxBoxExec struct{ pairRecordingExec }

func (b *linuxBoxExec) Run(ctx context.Context, cmd string, o *executor.RunOpts) (executor.Result, error) {
	_, _ = b.pairRecordingExec.Run(ctx, cmd, o)
	switch {
	case cmd == "uname -s":
		return executor.Result{Stdout: "Linux\n"}, nil
	case cmd == "uname -m":
		return executor.Result{Stdout: "x86_64\n"}, nil
	case strings.HasPrefix(cmd, "command -v systemctl"):
		return executor.Result{}, nil
	}
	return executor.Result{ExitCode: 1, Stderr: "refused by the test box"}, nil
}

// D14 / P19: the pairing stream says where the agent binary came from, so a
// stale developer override is visible to the person pairing. (The e2e
// harness drives bootstrap directly, so it is asserted here, on the stream.)
func TestPairStreamNamesTheAgentSource(t *testing.T) {
	ts, token := localPairServer(t, "linux", 0, &linuxBoxExec{})
	dir := filepath.Join(os.Getenv("HOME"), ".jumpgate", "agents")
	if err := fsperm.MkdirPrivate(dir); err != nil {
		t.Fatal(err)
	}
	body := []byte("dev agent")
	sum := sha256.Sum256(body)
	if err := fsperm.WriteFilePrivate(filepath.Join(dir, "jumpgate-linux-amd64"), body); err != nil {
		t.Fatal(err)
	}
	if err := fsperm.WriteFilePrivate(filepath.Join(dir, "SHA256SUMS"), []byte(hex.EncodeToString(sum[:])+"  jumpgate-linux-amd64\n")); err != nil {
		t.Fatal(err)
	}

	res := postPair(t, ts, token, `{}`)
	defer res.Body.Close()
	stream, _ := io.ReadAll(res.Body)
	want := `"line":"agent binary for linux/amd64: dev override ~/.jumpgate/agents"`
	if !strings.Contains(string(stream), want) {
		t.Fatalf("stream does not name the agent source (%s):\n%s", want, stream)
	}
}
