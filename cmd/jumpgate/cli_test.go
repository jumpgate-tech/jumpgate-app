package main

import (
	"bufio"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"fmt"
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
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/valve-tech/jumpgate/internal/agent"
	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/apiclient"
	"github.com/valve-tech/jumpgate/internal/apiclient/apiclienttest"
	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/daemon"
	"github.com/valve-tech/jumpgate/internal/executor"
	"github.com/valve-tech/jumpgate/internal/intent"
	"github.com/valve-tech/jumpgate/internal/server"
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
		"root@:22":                 {"", "", 0, false}, // empty host
		"root@[]":                  {"", "", 0, false},
		"root@[]:22":               {"", "", 0, false},
		"root@[::1":                {"", "", 0, false}, // unbalanced brackets
		"root@host]":               {"", "", 0, false},
		"root@[::1]]":              {"", "", 0, false},
		"root@[::1]":               {"root", "::1", 0, true},
	}
	for in, want := range cases {
		user, host, port, err := parseSSHTarget(in)
		if (err == nil) != want.ok || (want.ok && (user != want.user || host != want.host || port != want.port)) {
			t.Errorf("parseSSHTarget(%q) = %q %q %d %v", in, user, host, port, err)
		}
	}
}

// Rejection remedies are the server's now; the CLI's fallback is the same
// package, so an older server's bare rejection still gets one.
func TestPrintReplyFallsBackToTheSharedHint(t *testing.T) {
	var out, errw strings.Builder
	code := printReply(&out, &errw, "box", api.IntentReply{Status: intent.StatusRejected, Rejection: &intent.Rejection{Code: intent.ReasonClockSkew, Message: "skew"}})
	if code != 1 || !strings.Contains(errw.String(), api.RejectionHint(intent.ReasonClockSkew)) {
		t.Fatalf("exit %d, stderr %q", code, errw.String())
	}
	errw.Reset()
	printReply(&out, &errw, "box", api.IntentReply{Status: intent.StatusRejected, Hint: "server says", Rejection: &intent.Rejection{Code: intent.ReasonBusy}})
	if !strings.Contains(errw.String(), "server says") {
		t.Fatalf("server hint not shown: %q", errw.String())
	}
}

// Every server code the CLI used to carry its own remedy for still gets one
// when the server sent no hint: the registry's. controller_key_mismatch is
// among them (Ruling T2c).
func TestServerErrorFallsBackToTheRegistryHint(t *testing.T) {
	for _, code := range []api.Code{
		api.CodeUnreachable, api.CodeBadReceipt, api.CodeAgentHTTP, api.CodeUnknownHost, api.CodeHostKey,
		api.CodeNoControllerKey, api.CodeNotPaired, api.CodeControllerKeyMismatch,
	} {
		hint := api.HintFor(code)
		if hint == "" {
			t.Errorf("no registry hint for %s", code)
			continue
		}
		var w strings.Builder
		reportServerError(&w, "box", apiError{Status: 502, Code: code, Message: "boom"})
		if !strings.Contains(w.String(), "-> "+hint) {
			t.Errorf("%s: output %q lacks the registry hint", code, w.String())
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

// A server code this CLI has never heard of is a plain failure (exit 1):
// status 2 is usage only.
func TestServerErrorExit(t *testing.T) {
	cases := []struct {
		code api.Code
		want int
	}{
		{"unreachable", 3}, {"bad_receipt", 4}, {"agent_http", 1}, {"unknown_host", 4}, {"host_key", 4},
		{"no_controller_key", 2}, {"not_paired", 2}, {"something_new", 1}, {"", 1},
	}
	for _, c := range cases {
		var stderr strings.Builder
		got := reportServerError(&stderr, "pair", apiError{Status: 502, Code: c.code, Message: "boom", Hint: "do this"})
		if got != c.want {
			t.Errorf("code %q: exit %d, want %d", c.code, got, c.want)
		}
		if !strings.Contains(stderr.String(), "boom") {
			t.Errorf("code %q: error text missing from %q", c.code, stderr.String())
		}
		if c.code != "" && !strings.Contains(stderr.String(), "do this") {
			t.Errorf("code %q: hint missing from %q", c.code, stderr.String())
		}
		if security := c.want == 4; security != strings.Contains(stderr.String(), "SECURITY") {
			t.Errorf("code %q: SECURITY prefix = %v, want %v: %q", c.code, !security, security, stderr.String())
		}
	}
}

// agent init is idempotent: the box keeps its identity across re-pairing.
func TestAgentInitKeepsItsIdentity(t *testing.T) {
	state, conf := t.TempDir(), t.TempDir()
	var out1, out2 strings.Builder
	if err := agentInit(&out1, state, conf, false); err != nil {
		t.Fatal(err)
	}
	if err := agentInit(&out2, state, conf, false); err != nil {
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

// M6: a box that has its key but lost its replay record must not get a
// fresh, empty one silently on re-pair: that reopens a replay window the
// running agent deliberately fails closed on (replay_state). Only an explicit
// --reset-replay starts one.
func TestAgentInitRefusesToRecreateALostReplayRecord(t *testing.T) {
	state, conf := t.TempDir(), t.TempDir()
	var first strings.Builder
	if err := agentInit(&first, state, conf, false); err != nil {
		t.Fatal(err)
	}
	replay := filepath.Join(state, "replay.json")
	if err := os.Remove(replay); err != nil {
		t.Fatal(err)
	}
	err := agentInit(io.Discard, state, conf, false)
	if err == nil || !strings.Contains(err.Error(), "--reset-replay") {
		t.Fatalf("err = %v, want a refusal naming --reset-replay", err)
	}
	if _, err := os.Stat(replay); !os.IsNotExist(err) {
		t.Fatalf("replay.json was recreated: %v", err)
	}
	var again strings.Builder
	if err := agentInit(&again, state, conf, true); err != nil {
		t.Fatalf("with --reset-replay: %v", err)
	}
	if _, err := os.Stat(replay); err != nil {
		t.Fatalf("replay.json not created: %v", err)
	}
	if again.String() != first.String() {
		t.Fatalf("identity changed: %q vs %q", again.String(), first.String())
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
	if err := agentInit(io.Discard, state, conf, false); err == nil {
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
	if err := agentInit(io.Discard, state, conf, false); err == nil {
		t.Fatal("agent init accepted a group-readable key")
	}

	// A dangling symlink is not "missing": it must not be replaced either.
	link := filepath.Join(t.TempDir(), "agent.key")
	if err := os.Symlink(filepath.Join(t.TempDir(), "gone"), link); err != nil {
		t.Fatal(err)
	}
	if err := agentInit(io.Discard, filepath.Dir(link), conf, false); err == nil {
		t.Fatal("agent init replaced a dangling symlink")
	}
	if _, err := os.Lstat(link); err != nil {
		t.Fatalf("symlink gone: %v", err)
	}
}

func TestAgentEnrollIsIdempotent(t *testing.T) {
	conf := t.TempDir()
	for i := 0; i < 2; i++ {
		if err := agentEnroll(conf, "", "0x00000000000000000000000000000000000000cc", "routine", "laptop", 501); err != nil {
			t.Fatal(err)
		}
	}
	p, err := agent.LoadPolicy(filepath.Join(conf, "policy.json"))
	if err != nil || len(p.Signers) != 1 || len(p.LocalUIDs) != 1 {
		t.Fatalf("policy = %+v, %v", p, err)
	}
}

// R23: enrolling a local uid opens the live socket to it at once. Bootstrap
// also restarts the agent, but a hand-run enroll must not leave the uid
// enrolled yet refused by the kernel at 0660.
func TestAgentEnrollLocalUIDOpensTheSocket(t *testing.T) {
	conf := t.TempDir()
	d, err := os.MkdirTemp("/tmp", "jgs")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(d) })
	sock := filepath.Join(d, "agent.sock")
	ln, err := agent.Listen(sock, -1, 0o660)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	if err := agentEnroll(conf, sock, "0x00000000000000000000000000000000000000cc", "routine", "x", -1); err != nil {
		t.Fatal(err)
	}
	if fi, _ := os.Stat(sock); fi.Mode().Perm() != 0o660 {
		t.Fatalf("controller-only enroll: socket mode %o, want 660", fi.Mode().Perm())
	}
	if err := agentEnroll(conf, sock, "0x00000000000000000000000000000000000000cc", "routine", "x", 1000); err != nil {
		t.Fatal(err)
	}
	if fi, _ := os.Stat(sock); fi.Mode().Perm() != 0o666 {
		t.Fatalf("local-uid enroll: socket mode %o, want 666", fi.Mode().Perm())
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
		if err := agentEnroll(conf, "", a, "routine", "x", -1); err != nil {
			t.Fatal(err)
		}
	}
	p, err := agent.LoadPolicy(filepath.Join(conf, "policy.json"))
	if err != nil || len(p.Signers) != 2 || len(p.LocalUIDs) != 0 {
		t.Fatalf("policy = %+v, %v", p, err)
	}
	if err := agentEnroll(conf, "", "0xnope", "routine", "x", -1); err == nil {
		t.Error("accepted a bad address")
	}
	if err := agentEnroll(conf, "", steps[0], "admin", "x", -1); err == nil {
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
	hk := cliServer(t)
	host, port, key := startHostKeySSHD(t)
	cfg := executor.SSHConfig{Host: host, Port: port, User: "root"}
	confirmed, err := config.ConfirmedHostsFile()
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	// Anything but an exact "yes" trusts nothing.
	// A bare "yes" with no newline (EOF) is not consent either.
	for _, answer := range []string{"no\n", "y\n", "\n", "", "yes"} {
		var out strings.Builder
		if code := confirmHostKeys(ctx, hk, sshViewOf(cfg), bufio.NewReader(strings.NewReader(answer)), &out, io.Discard); code == 0 {
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
	if code := confirmHostKeys(ctx, hk, sshViewOf(cfg), bufio.NewReader(strings.NewReader("yes\n")), io.Discard, io.Discard); code != 0 {
		t.Fatalf("exit %d after yes", code)
	}
	b, err := os.ReadFile(confirmed)
	if err != nil || !strings.Contains(string(b), net.JoinHostPort(host, strconv.Itoa(port))) {
		t.Fatalf("confirmed_hosts = %q, %v", b, err)
	}

	// Now known: no prompt (empty input would fail if one were needed).
	if code := confirmHostKeys(ctx, hk, sshViewOf(cfg), bufio.NewReader(strings.NewReader("")), io.Discard, io.Discard); code != 0 {
		t.Fatalf("a confirmed host prompted again: exit %d", code)
	}
}

func TestConfirmHostKeysMismatchIsFatal(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	hk := cliServer(t)
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
	code := confirmHostKeys(context.Background(), hk, sshViewOf(cfg), bufio.NewReader(strings.NewReader("yes\n")), io.Discard, io.Discard)
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
	t.Setenv("SSH_AUTH_SOCK", "") // the dial through the jump must not reach the real ssh-agent
	hk := cliServer(t)
	jh, jp, _ := startHostKeySSHD(t)
	cfg := executor.SSHConfig{
		Host: "10.255.255.1", Port: 22, User: "root",
		Jump: &executor.SSHConfig{Host: jh, Port: jp, User: "ops"},
	}
	// The target is unreachable through this fake jump (it speaks no
	// direct-tcpip), but the jump must have been shown and confirmed first, and
	// the failure must be "unreachable" rather than a TOFU acceptance.
	var out strings.Builder
	code := confirmHostKeys(context.Background(), hk, sshViewOf(cfg), bufio.NewReader(strings.NewReader("yes\n")), &out, io.Discard)
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
}

// fakeHostKeys scripts the server's probe answers and records every
// confirmation the CLI sends.
type fakeHostKeys struct {
	probes   []api.HostKeyProbe // answered in order; the last one repeats
	probeErr error
	info     daemon.Info
	confirms []string
}

func (f *fakeHostKeys) ProbeHostKeys(context.Context, api.SSHView) (api.HostKeyProbe, error) {
	if f.probeErr != nil {
		return api.HostKeyProbe{}, f.probeErr
	}
	p := f.probes[0]
	if len(f.probes) > 1 {
		f.probes = f.probes[1:]
	}
	return p, nil
}

func (f *fakeHostKeys) ConfirmHostKey(_ context.Context, probeID, fingerprint string) error {
	f.confirms = append(f.confirms, probeID+" "+fingerprint)
	return nil
}

func (f *fakeHostKeys) Info() daemon.Info { return f.info }

func unknownHop(id, fp string) api.HostKeyProbe {
	return api.HostKeyProbe{Hops: []api.HostKeyHop{{HostPort: "h:22", KeyType: "ssh-ed25519", Fingerprint: fp, State: api.HostKeyUnknown, ProbeID: id}}}
}

// The confirmation names the probe and the fingerprint the operator was
// shown, nothing else.
func TestConfirmHostKeysConfirmsExactlyTheShownProbe(t *testing.T) {
	f := &fakeHostKeys{probes: []api.HostKeyProbe{
		unknownHop("p1", "SHA256:abc"),
		{Hops: []api.HostKeyHop{{HostPort: "h:22", State: api.HostKeyConfirmed}}, AllConfirmed: true},
	}}
	var out strings.Builder
	if code := confirmHostKeys(context.Background(), f, api.SSHView{Host: "h", User: "u"}, bufio.NewReader(strings.NewReader("yes\n")), &out, io.Discard); code != 0 {
		t.Fatalf("exit %d", code)
	}
	if len(f.confirms) != 1 || f.confirms[0] != "p1 SHA256:abc" || !strings.Contains(out.String(), "SHA256:abc") {
		t.Fatalf("confirms %q, shown %q", f.confirms, out.String())
	}
}

// A mismatch is never offered for trust, whatever stdin says, and a server
// that never settles (or one that has not probed every hop) trusts nothing.
func TestConfirmHostKeysRefusals(t *testing.T) {
	for name, c := range map[string]struct {
		probes []api.HostKeyProbe
		input  string
		want   int
	}{
		"mismatch":         {[]api.HostKeyProbe{{Hops: []api.HostKeyHop{{HostPort: "h:22", State: api.HostKeyMismatch, Error: "host key mismatch"}}}}, "yes\n", 4},
		"unreachable":      {[]api.HostKeyProbe{{Hops: []api.HostKeyHop{{HostPort: "h:22", State: api.HostKeyUnreachable, Error: "refused"}}}}, "yes\n", 3},
		"never settles":    {[]api.HostKeyProbe{unknownHop("p1", "SHA256:abc")}, strings.Repeat("yes\n", 10), 1},
		"unknown state":    {[]api.HostKeyProbe{{Hops: []api.HostKeyHop{{HostPort: "h:22", State: "new-state", ProbeID: "p1"}}}}, "yes\n", 1},
		"incomplete probe": {[]api.HostKeyProbe{{Hops: []api.HostKeyHop{}}}, "", 1},
	} {
		f := &fakeHostKeys{probes: c.probes}
		code := confirmHostKeys(context.Background(), f, api.SSHView{Host: "h", User: "u"}, bufio.NewReader(strings.NewReader(c.input)), io.Discard, io.Discard)
		if code != c.want {
			t.Errorf("%s: exit %d, want %d", name, code, c.want)
		}
		if name != "never settles" && len(f.confirms) != 0 {
			t.Errorf("%s: confirmed %q", name, f.confirms)
		}
		if name == "never settles" && len(f.confirms) != 2 {
			t.Errorf("never settles: %d confirmations for one hop, want 2 rounds", len(f.confirms))
		}
	}
}

// An older server has no probe route: its 404 is reported as the version
// skew it is, not as a missing box.
func TestConfirmHostKeysOnAnOlderServer(t *testing.T) {
	f := &fakeHostKeys{probeErr: &api.Error{Status: http.StatusNotFound, Message: "404 page not found", Code: api.CodeNotFound}, info: daemon.Info{Version: "0.0.1-old"}}
	var errw strings.Builder
	if code := confirmHostKeys(context.Background(), f, api.SSHView{Host: "h", User: "u"}, bufio.NewReader(strings.NewReader("yes\n")), io.Discard, &errw); code != 1 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(errw.String(), "does not know this request") {
		t.Fatalf("stderr %q", errw.String())
	}
}

// hosts add starts the server first and confirms through it, before the
// target is recorded; the target carries the address and no host-key policy.
func TestHostsAddConfirmsThroughTheServerFirst(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	old := stdinIsTerminal
	t.Cleanup(func() { stdinIsTerminal = old })
	stdinIsTerminal = func() bool { return true }
	var paths []string
	var targetBody string
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		switch r.URL.Path {
		case "/api/hostkeys/probe":
			fmt.Fprint(w, `{"hops":[{"hostPort":"127.0.0.1:2200","state":"confirmed"}],"allConfirmed":true}`)
		case "/api/targets":
			b, _ := io.ReadAll(r.Body)
			targetBody = string(b)
			w.WriteHeader(http.StatusNoContent)
		case "/api/targets/box/pair":
			w.Header().Set("Content-Type", "text/event-stream")
			fmt.Fprint(w, "data: {\"done\":true,\"agent\":\"0xabc\"}\n\n")
		default:
			http.NotFound(w, r)
		}
	})
	_, stderr := captureStdio(t)
	if code := hostsAdd([]string{"box", "--ssh", "root@127.0.0.1:2200"}); code != 0 {
		t.Fatalf("exit %d, stderr %q", code, stderr())
	}
	if strings.Join(paths, ",") != "/api/hostkeys/probe,/api/targets,/api/targets/box/pair" {
		t.Fatalf("requests %v", paths)
	}
	if !strings.Contains(targetBody, `"Host":"127.0.0.1"`) || !strings.Contains(targetBody, `"Port":2200`) || !strings.Contains(targetBody, `"HostKeyFile":""`) {
		t.Fatalf("target body %s", targetBody)
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

// ---- non-interactive consent, dispatch, exit classes ----

func TestHostsAddRefusesNonTerminalStdin(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("SSH_AUTH_SOCK", "")
	old := stdinIsTerminal
	defer func() { stdinIsTerminal = old }()
	stdinIsTerminal = func() bool { return false }

	// Port 1 on loopback is closed: any capture attempt would exit 3. The
	// refusal comes first, so the exit is 1 and nothing is written.
	code := hostsAdd([]string{"box", "--ssh", "root@127.0.0.1:1"})
	if code != 1 {
		t.Fatalf("exit %d, want 1 (refused before any capture)", code)
	}
	confirmed, _ := config.ConfirmedHostsFile()
	if _, err := os.Stat(confirmed); err == nil {
		t.Fatal("confirmed_hosts was written")
	}
}

func TestExitClasses(t *testing.T) {
	if code := hostsAdd([]string{"box"}); code != 2 {
		t.Errorf("missing --ssh/--local: exit %d, want 2", code)
	}
	if code := hostsAdd([]string{"box", "--ssh", "root@[::1"}); code != 2 {
		t.Errorf("bad --ssh: exit %d, want 2", code)
	}
	if code := cmdKeys(nil); code != 2 {
		t.Errorf("keys without a subcommand: exit %d, want 2", code)
	}
	// Runtime failures are 1: no controller key to show, a controller that
	// already exists, an unreadable config.
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	cliServer(t)
	if code := cmdKeys([]string{"show"}); code != 1 {
		t.Errorf("keys show with no key: exit %d, want 1", code)
	}
	if code := cmdKeys([]string{"init", "--store", "file"}); code != 0 {
		t.Fatalf("init exit %d", code)
	}
	if code := cmdKeys([]string{"init", "--store", "file"}); code != 1 {
		t.Errorf("second keys init: exit %d, want 1", code)
	}
	if err := os.WriteFile(filepath.Join(home, ".jumpgate", "config.json"), []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if code := cmdHosts([]string{"list"}); code != 1 {
		t.Errorf("hosts list with a broken config: exit %d, want 1", code)
	}
	if code := failed("x"); code != 1 {
		t.Errorf("failed() = %d", code)
	}
	if code := usage("x"); code != 2 {
		t.Errorf("usage() = %d", code)
	}
}

func TestDispatch(t *testing.T) {
	var stderr strings.Builder
	code, handled := dispatch([]string{"jumpgate", "statsu", "box1"}, &stderr)
	if !handled || code != 2 {
		t.Fatalf("typo: code %d handled %v, want 2 true", code, handled)
	}
	if !strings.Contains(stderr.String(), `unknown command "statsu"`) || !strings.Contains(stderr.String(), "status") {
		t.Errorf("message = %q", stderr.String())
	}
	for _, args := range [][]string{{"jumpgate"}, {"jumpgate", "--bind", "127.0.0.1:1"}, {"jumpgate", "-version"}} {
		if _, handled := dispatch(args, io.Discard); handled {
			t.Errorf("%v was taken by dispatch; it belongs to runApp", args)
		}
	}
	// A real subcommand is routed (usage error from the handler, not unknown-command).
	if code, handled := dispatch([]string{"jumpgate", "keys"}, io.Discard); !handled || code != 2 {
		t.Errorf("keys: %d %v", code, handled)
	}
}

type endless byte

func (e endless) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = byte(e)
	}
	return len(p), nil
}

// An endless stdin, or one long line ending in "yes", is "no", and the read
// stops at the bound instead of growing.
func TestConfirmHostKeysBoundsTheAnswer(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	hk := cliServer(t)
	host, port, _ := startHostKeySSHD(t)
	cfg := executor.SSHConfig{Host: host, Port: port, User: "root"}
	confirmed, _ := config.ConfirmedHostsFile()
	inputs := map[string]io.Reader{
		"endless y":      endless('y'),
		"endless NUL":    endless(0),
		"long line, yes": strings.NewReader(strings.Repeat("a", 2*maxConsentLine) + "yes\n"),
	}
	for name, r := range inputs {
		done := make(chan int, 1)
		go func() {
			done <- confirmHostKeys(context.Background(), hk, sshViewOf(cfg), newConsentReader(r), io.Discard, io.Discard)
		}()
		select {
		case code := <-done:
			if code == 0 {
				t.Errorf("%s: trusted the key", name)
			}
		case <-time.After(10 * time.Second):
			t.Fatalf("%s: the prompt never gave up", name)
		}
		if _, err := os.Stat(confirmed); err == nil {
			t.Fatalf("%s: confirmed_hosts was written", name)
		}
	}
}

// ---- legacy directory migration ----

// I1/R24: every controller subcommand migrates ~/.valve-node-app before it
// runs, because most of them create ~/.jumpgate first (the config lock, the
// run directory). The on-box agent subcommands never touch controller state
// and must not fail on it.
func TestMigrateOnStartup(t *testing.T) {
	setup := func(t *testing.T) string {
		home := t.TempDir()
		t.Setenv("HOME", home)
		legacy := filepath.Join(home, ".valve-node-app")
		if err := os.MkdirAll(legacy, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(legacy, "config.json"), []byte(`{}`), 0o600); err != nil {
			t.Fatal(err)
		}
		// What `jumpgate status` leaves behind before serve starts.
		if err := os.MkdirAll(filepath.Join(home, ".jumpgate", "run"), 0o700); err != nil {
			t.Fatal(err)
		}
		return home
	}
	for _, args := range [][]string{{"jumpgate"}, {"jumpgate", "--bind", "x"}, {"jumpgate", "status", "box"}, {"jumpgate", "keys", "init"}, {"jumpgate", "serve"}} {
		home := setup(t)
		var errOut strings.Builder
		if err := migrateOnStartup(args, &errOut); err != nil {
			t.Fatalf("%v: %v", args, err)
		}
		if _, err := os.Stat(filepath.Join(home, ".jumpgate", "config.json")); err != nil {
			t.Errorf("%v: config not migrated: %v", args, err)
		}
		if !strings.Contains(errOut.String(), "moved ~/.valve-node-app") {
			t.Errorf("%v: no notice, got %q", args, errOut.String())
		}
	}
	home := setup(t)
	if err := migrateOnStartup([]string{"jumpgate", "agent", "run"}, io.Discard); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(home, ".jumpgate", "config.json")); !os.IsNotExist(err) {
		t.Errorf("agent run migrated controller state: %v", err)
	}
}

// M3: re-running hosts add on an existing name re-pairs the stored address,
// so a different --ssh (or --jump, or --local) is refused up front rather
// than confirming keys for one address and pairing another.
func TestHostsAddRefusesToRepointAnExistingTarget(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	_, err := config.Update(func(c *config.Config) error {
		c.Targets = append(c.Targets,
			config.Target{ID: "box", Mode: "ssh", SSH: &executor.SSHConfig{Host: "203.0.113.7", User: "root"}},
			config.Target{ID: "here", Mode: "local"})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	refused := []struct {
		name      string
		local     bool
		ssh, jump string
	}{
		{"box", false, "root@203.0.113.8", ""},
		{"box", false, "ops@203.0.113.7", ""},
		{"box", false, "root@203.0.113.7:2222", ""},
		{"box", false, "root@203.0.113.7", "ops@198.51.100.1"},
		{"box", true, "", ""},
		{"here", false, "root@203.0.113.7", ""},
	}
	for _, c := range refused {
		err := checkExistingTarget(c.name, c.local, c.ssh, c.jump)
		if err == nil || !strings.Contains(err.Error(), "already exists") || !strings.Contains(err.Error(), "remove") {
			t.Errorf("%+v: err = %v, want a refusal saying how to remove and re-add", c, err)
		}
	}
	for _, c := range []struct {
		name      string
		local     bool
		ssh, jump string
	}{
		{"box", false, "root@203.0.113.7", ""},
		{"box", false, "root@203.0.113.7:22", ""},
		{"here", true, "", ""},
		{"new", false, "root@203.0.113.9", ""},
	} {
		if err := checkExistingTarget(c.name, c.local, c.ssh, c.jump); err != nil {
			t.Errorf("%+v: %v, want allowed", c, err)
		}
	}

	// The refusal comes before any host-key prompt or capture.
	if code := hostsAdd([]string{"box", "--ssh", "root@127.0.0.1:1"}); code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	confirmed, _ := config.ConfirmedHostsFile()
	if _, err := os.Stat(confirmed); err == nil {
		t.Fatal("confirmed_hosts was written")
	}
}

// cliServer points the CLI at an in-process server over the current HOME, so
// a test never starts a detached `serve` (which would be this test binary).
func cliServer(t *testing.T) *apiclient.Client {
	t.Helper()
	token := server.NewSessionToken()
	ts := httptest.NewServer(server.New(server.Config{Token: token, UI: fstest.MapFS{}}).Handler())
	t.Cleanup(ts.Close)
	c := apiclienttest.NewHTTP(t, ts.URL, token)
	old := connect
	connect = func(context.Context) (*apiclient.Client, error) { return c, nil }
	t.Cleanup(func() { connect = old })
	return c
}

func TestHostsListComesFromTheServer(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	if _, err := config.Update(func(c *config.Config) error {
		c.Targets = []config.Target{{ID: "box", Mode: "ssh", SSH: &executor.SSHConfig{Host: "h", User: "u"}, Agent: &config.AgentPairing{Address: "0xabc", Transport: "ssh"}}}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	cliServer(t)
	var out strings.Builder
	if code := hostsList(context.Background(), &out); code != 0 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(out.String(), "box") || !strings.Contains(out.String(), "agent 0xabc") {
		t.Fatalf("output %q", out.String())
	}
}
