package main

import (
	"bufio"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/valve-tech/jumpgate/internal/agent"
	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/executor"
	"github.com/valve-tech/jumpgate/internal/intent"
)

func TestParseSSHTarget(t *testing.T) {
	cases := map[string]struct {
		user, host string
		port       int
		ok         bool
	}{
		"root@203.0.113.7":         {"root", "203.0.113.7", 0, true},
		"ops@box.example.com:2222": {"ops", "box.example.com", 2222, true},
		"root@[2001:db8::1]:22":    {"root", "2001:db8::1", 22, true},
		"203.0.113.7":              {"", "", 0, false}, // user is required
		"root@":                    {"", "", 0, false},
		"root@host:notaport":       {"", "", 0, false},
	}
	for in, want := range cases {
		user, host, port, err := parseSSHTarget(in)
		if (err == nil) != want.ok || (want.ok && (user != want.user || host != want.host || port != want.port)) {
			t.Errorf("parseSSHTarget(%q) = %q %q %d %v", in, user, host, port, err)
		}
	}
}

// Every rejection code the agent can send has a one-line remedy; a new code
// without one fails here instead of reaching an operator as a bare string.
// The server's own error codes carry a remedy too, for when its hint is empty.
func TestEveryReasonHasARemedy(t *testing.T) {
	for _, code := range []string{
		intent.ReasonWrongAgent, intent.ReasonExpired, intent.ReasonClockSkew, intent.ReasonBadSignature,
		intent.ReasonUnauthorizedKind, intent.ReasonUnknownKind, intent.ReasonStaleSeq, intent.ReasonReplayedNonce,
		intent.ReasonBusy, intent.ReasonInvalidPayload, intent.ReasonValidation, intent.ReasonNotSetUp, intent.ReasonReplayState,
		"unreachable", "bad_receipt", "agent_http", "unknown_host", "no_controller_key", "not_paired",
	} {
		if remedies[code] == "" {
			t.Errorf("no remedy for %s", code)
		}
	}
}

func TestExitCodes(t *testing.T) {
	cases := map[string]int{
		"ok": 0, "refused": 1, "failed": 1, "usage": 2, "unreachable": 3, "bad_receipt": 4, "host_key": 4,
		// the codes the server's error JSON carries
		"agent_http": 1, "unknown_host": 4, "no_controller_key": 2, "not_paired": 2,
	}
	for outcome, want := range cases {
		if got := exitCode(outcome); got != want {
			t.Errorf("exitCode(%s) = %d, want %d", outcome, got, want)
		}
	}
}

// A server code this CLI has never heard of is a usage-class failure (exit 2),
// while a missing code is a plain failure.
func TestServerErrorExit(t *testing.T) {
	cases := []struct {
		code string
		want int
	}{
		{"unreachable", 3}, {"bad_receipt", 4}, {"agent_http", 1}, {"unknown_host", 4},
		{"no_controller_key", 2}, {"not_paired", 2}, {"something_new", 2}, {"", 1},
	}
	for _, c := range cases {
		var stderr strings.Builder
		got := reportServerError(&stderr, "pair", apiError{Status: 502, Code: c.code, Error: "boom", Hint: "do this"})
		if got != c.want {
			t.Errorf("code %q: exit %d, want %d", c.code, got, c.want)
		}
		if !strings.Contains(stderr.String(), "boom") {
			t.Errorf("code %q: error text missing from %q", c.code, stderr.String())
		}
		if c.code != "" && !strings.Contains(stderr.String(), "do this") {
			t.Errorf("code %q: hint missing from %q", c.code, stderr.String())
		}
	}
}

// agent init is idempotent: the box keeps its identity across re-pairing.
func TestAgentInitKeepsItsIdentity(t *testing.T) {
	state, conf := t.TempDir(), t.TempDir()
	var out1, out2 strings.Builder
	if err := agentInit(&out1, state, conf); err != nil {
		t.Fatal(err)
	}
	if err := agentInit(&out2, state, conf); err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(out1.String()) != strings.TrimSpace(out2.String()) {
		t.Fatalf("identity changed: %q vs %q", out1.String(), out2.String())
	}
	if _, err := os.Stat(filepath.Join(state, "replay.json")); err != nil {
		t.Fatal("agent init did not create replay.json")
	}
	if fi, _ := os.Stat(conf); fi.Mode().Perm() != 0o700 {
		t.Errorf("config dir mode %o, want 700", fi.Mode().Perm())
	}
}

// A key file that exists but cannot be used is an error, never a reason to
// make a new identity: only "does not exist" generates.
func TestAgentInitNeverReplacesAnUnreadableKey(t *testing.T) {
	state, conf := t.TempDir(), t.TempDir()
	keyPath := filepath.Join(state, "agent.key")
	if err := os.WriteFile(keyPath, []byte("not hex\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := agentInit(io.Discard, state, conf); err == nil {
		t.Fatal("agent init accepted a damaged key file")
	}
	if b, _ := os.ReadFile(keyPath); string(b) != "not hex\n" {
		t.Fatalf("key file was rewritten: %q", b)
	}

	// A group-readable key is refused the same way, and left alone.
	if err := os.WriteFile(keyPath, []byte(strings.Repeat("ab", 32)+"\n"), 0o640); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(keyPath, 0o640); err != nil {
		t.Fatal(err)
	}
	if err := agentInit(io.Discard, state, conf); err == nil {
		t.Fatal("agent init accepted a group-readable key")
	}

	// A dangling symlink is not "missing": it must not be replaced either.
	link := filepath.Join(t.TempDir(), "agent.key")
	if err := os.Symlink(filepath.Join(t.TempDir(), "gone"), link); err != nil {
		t.Fatal(err)
	}
	if err := agentInit(io.Discard, filepath.Dir(link), conf); err == nil {
		t.Fatal("agent init replaced a dangling symlink")
	}
	if _, err := os.Lstat(link); err != nil {
		t.Fatalf("symlink gone: %v", err)
	}
}

func TestAgentEnrollIsIdempotent(t *testing.T) {
	conf := t.TempDir()
	for i := 0; i < 2; i++ {
		if err := agentEnroll(conf, "0x00000000000000000000000000000000000000cc", "routine", "laptop", 501); err != nil {
			t.Fatal(err)
		}
	}
	p, err := agent.LoadPolicy(filepath.Join(conf, "policy.json"))
	if err != nil || len(p.Signers) != 1 || len(p.LocalUIDs) != 1 {
		t.Fatalf("policy = %+v, %v", p, err)
	}
}

// Enrolling a second controller appends; it never drops the first, and the
// same address in another case is the same controller.
func TestAgentEnrollAppends(t *testing.T) {
	conf := t.TempDir()
	steps := []string{
		"0x00000000000000000000000000000000000000cc",
		"0x00000000000000000000000000000000000000dd",
		"0x00000000000000000000000000000000000000CC",
	}
	for _, a := range steps {
		if err := agentEnroll(conf, a, "routine", "x", -1); err != nil {
			t.Fatal(err)
		}
	}
	p, err := agent.LoadPolicy(filepath.Join(conf, "policy.json"))
	if err != nil || len(p.Signers) != 2 || len(p.LocalUIDs) != 0 {
		t.Fatalf("policy = %+v, %v", p, err)
	}
	if err := agentEnroll(conf, "0xnope", "routine", "x", -1); err == nil {
		t.Error("accepted a bad address")
	}
	if err := agentEnroll(conf, steps[0], "admin", "x", -1); err == nil {
		t.Error("accepted an unknown tier")
	}
}

// ---- host-key confirmation ----

// startHostKeySSHD answers the SSH handshake with a fresh host key and nothing
// else, which is all CaptureHostKey needs. It returns the host:port and key.
func startHostKeySSHD(t *testing.T) (host string, port int, key ssh.PublicKey) {
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
	addr := ln.Addr().(*net.TCPAddr)
	return addr.IP.String(), addr.Port, signer.PublicKey()
}

func TestConfirmHostKeys(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	host, port, key := startHostKeySSHD(t)
	cfg := executor.SSHConfig{Host: host, Port: port, User: "root"}
	confirmed, err := config.ConfirmedHostsFile()
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	// Anything but an exact "yes" trusts nothing.
	for _, answer := range []string{"no\n", "y\n", "\n", ""} {
		var out strings.Builder
		if code := confirmHostKeys(ctx, cfg, bufio.NewReader(strings.NewReader(answer)), &out); code == 0 {
			t.Fatalf("answer %q trusted the key", answer)
		}
		if _, err := os.Stat(confirmed); err == nil {
			t.Fatalf("answer %q wrote the confirmed-hosts file", answer)
		}
		if !strings.Contains(out.String(), executor.Fingerprint(key)) {
			t.Fatalf("the fingerprint was not shown: %q", out.String())
		}
	}
	// The TOFU store is never touched by the CLI.
	if _, err := os.Stat(filepath.Join(home, ".jumpgate", "known_hosts")); err == nil {
		t.Fatal("known_hosts (TOFU) was written")
	}

	// "yes" records it in the confirmed store, under the host:port DialSSH uses.
	if code := confirmHostKeys(ctx, cfg, bufio.NewReader(strings.NewReader("yes\n")), io.Discard); code != 0 {
		t.Fatalf("exit %d after yes", code)
	}
	b, err := os.ReadFile(confirmed)
	if err != nil || !strings.Contains(string(b), net.JoinHostPort(host, strconv.Itoa(port))) {
		t.Fatalf("confirmed_hosts = %q, %v", b, err)
	}

	// Now known: no prompt (empty input would fail if one were needed).
	if code := confirmHostKeys(ctx, cfg, bufio.NewReader(strings.NewReader("")), io.Discard); code != 0 {
		t.Fatalf("a confirmed host prompted again: exit %d", code)
	}
}

func TestConfirmHostKeysMismatchIsFatal(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	host, port, _ := startHostKeySSHD(t)
	cfg := executor.SSHConfig{Host: host, Port: port, User: "root"}

	// Pin a different key for this host:port, as if the box had been replaced.
	_, other, _ := ed25519.GenerateKey(rand.Reader)
	otherSigner, _ := ssh.NewSignerFromKey(other)
	confirmed, _ := config.ConfirmedHostsFile()
	if err := executor.RecordHostKey(confirmed, net.JoinHostPort(host, strconv.Itoa(port)), otherSigner.PublicKey()); err != nil {
		t.Fatal(err)
	}
	// "yes" on stdin must not matter: a mismatch is never offered for trust.
	code := confirmHostKeys(context.Background(), cfg, bufio.NewReader(strings.NewReader("yes\n")), io.Discard)
	if code != 4 {
		t.Fatalf("exit %d, want 4", code)
	}
}

// The jump host is confirmed first, with no Jump set on its probe, and only
// then is the target reached through it under the strict callback.
func TestConfirmHostKeysJumpFirstAndTargetUsesStrict(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	jh, jp, _ := startHostKeySSHD(t)
	cfg := executor.SSHConfig{
		Host: "10.255.255.1", Port: 22, User: "root",
		Jump: &executor.SSHConfig{Host: jh, Port: jp, User: "ops"},
	}
	// The target is unreachable through this fake jump (it speaks no
	// direct-tcpip), but the jump must have been shown and confirmed first, and
	// the failure must be "unreachable" rather than a TOFU acceptance.
	var out strings.Builder
	code := confirmHostKeys(context.Background(), cfg, bufio.NewReader(strings.NewReader("yes\n")), &out)
	if code != 3 {
		t.Fatalf("exit %d, want 3 (target unreachable through the jump)", code)
	}
	confirmed, _ := config.ConfirmedHostsFile()
	b, _ := os.ReadFile(confirmed)
	if !strings.Contains(string(b), net.JoinHostPort(jh, strconv.Itoa(jp))) {
		t.Fatalf("the jump host was not recorded: %q", b)
	}
	if strings.Contains(string(b), "10.255.255.1") {
		t.Fatalf("the unreachable target was recorded: %q", b)
	}
	if cfg.Jump.HostKey != nil {
		t.Error("confirmHostKeys mutated the caller's config")
	}
}

func TestSSHConfigFromAppliesStrictEverywhere(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	cfg, err := sshConfigFrom("root@box.example:2200", "", "ops@jump.example")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.HostKey == nil || cfg.Jump == nil || cfg.Jump.HostKey == nil {
		t.Fatalf("a dial config lacks the strict host-key callback: %+v", cfg)
	}
}

// ---- keys init ----

func TestKeysInitRefusesAnExistingController(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	if code := cmdKeys([]string{"init", "--store", "file"}); code != 0 {
		t.Fatalf("first init exit %d", code)
	}
	c, err := config.Load()
	if err != nil || c.Controller == nil {
		t.Fatalf("config = %+v, %v", c, err)
	}
	before, _ := os.ReadFile(c.Controller.KeyRef)
	if code := cmdKeys([]string{"init", "--store", "file"}); code == 0 {
		t.Fatal("second init succeeded")
	}
	if code := cmdKeys([]string{"init", "--store", "file", "--ref", filepath.Join(home, "other.key")}); code == 0 {
		t.Fatal("init with a new ref replaced the controller")
	}
	if after, _ := os.ReadFile(c.Controller.KeyRef); string(after) != string(before) {
		t.Fatal("the controller key changed")
	}
	if _, err := os.Stat(filepath.Join(home, "other.key")); err == nil {
		t.Fatal("a second key was created")
	}
}

// A key already stored at the chosen place (no controller in config, e.g.
// config was lost) is reported, not replaced.
func TestKeysInitReportsAnExistingKey(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	ref := filepath.Join(home, "k.key")
	orig := strings.Repeat("ab", 32) + "\n"
	if err := os.WriteFile(ref, []byte(orig), 0o600); err != nil {
		t.Fatal(err)
	}
	if code := cmdKeys([]string{"init", "--store", "file", "--ref", ref}); code == 0 {
		t.Fatal("init over an existing key succeeded")
	}
	if b, _ := os.ReadFile(ref); string(b) != orig {
		t.Fatal("existing key was changed")
	}
	if c, _ := config.Load(); c.Controller != nil {
		t.Fatal("a controller was recorded for a key init did not create")
	}
}

// ---- serve ----

func TestWaitForSocket(t *testing.T) {
	sock := filepath.Join(t.TempDir(), "s.sock")
	go func() {
		time.Sleep(150 * time.Millisecond)
		ln, err := net.Listen("unix", sock)
		if err == nil {
			t.Cleanup(func() { ln.Close() })
		}
	}()
	if err := waitForSocket(context.Background(), sock, 2*time.Second); err != nil {
		t.Fatal(err)
	}
	if err := waitForSocket(context.Background(), filepath.Join(t.TempDir(), "never.sock"), 200*time.Millisecond); err == nil {
		t.Fatal("a socket nobody listens on counted as ready")
	}
}

// serveBoth must not return while either listener is still finishing its
// critical operations, even when the other already failed.
func TestServeBothWaitsForBothListeners(t *testing.T) {
	ctx, stop := context.WithCancel(context.Background())
	defer stop()
	var unixDone atomic.Bool
	tcp := func(context.Context) error { return errors.New("bind failed") }
	unix := func(ctx context.Context) error {
		<-ctx.Done()
		time.Sleep(200 * time.Millisecond) // a clear that is still finishing
		unixDone.Store(true)
		return nil
	}
	err := serveBoth(ctx, stop, tcp, unix, func() error { return nil }, func() error { return nil })
	if err == nil || !strings.Contains(err.Error(), "bind failed") {
		t.Fatalf("err = %v", err)
	}
	if !unixDone.Load() {
		t.Fatal("serveBoth returned before the unix listener finished")
	}
}

func TestServeBothDoesNotPublishUntilReady(t *testing.T) {
	ctx, stop := context.WithCancel(context.Background())
	defer stop()
	var published atomic.Bool
	block := func(ctx context.Context) error { <-ctx.Done(); return nil }
	err := serveBoth(ctx, stop, block, block,
		func() error { return errors.New("socket never came up") },
		func() error { published.Store(true); return nil })
	if err == nil || published.Load() {
		t.Fatalf("err = %v, published = %v", err, published.Load())
	}
}
