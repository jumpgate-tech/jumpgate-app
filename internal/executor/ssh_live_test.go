package executor

// The SSH executor against a real SSH server.
//
// In-process, but a genuine RFC 4253 server from x/crypto/ssh — a real
// handshake, a real host key the TOFU callback records, real session channels
// and real exit statuses. Asserting on the command strings alone has already
// missed things in this repo: a perfectly-shaped argv that the far side does
// something else with looks identical from the near side.

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"encoding/pem"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
)

// reply is what the fake server does with one command.
type reply struct {
	stdout   string
	stderr   string
	exitCode uint32
}

// sshServer is a one-host SSH server that answers exec requests from a
// substring-keyed script, and records every command it was asked to run.
type sshServer struct {
	t       *testing.T
	ln      net.Listener
	cfg     *ssh.ServerConfig
	scripts map[string]reply

	mu    sync.Mutex
	calls []string
}

// newSSHServer starts a server on loopback and returns it with the path to a
// client private key it will accept.
func newSSHServer(t *testing.T) (*sshServer, string) {
	t.Helper()

	// The host key the client will record on first use.
	_, hostPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate host key: %v", err)
	}
	hostSigner, err := ssh.NewSignerFromKey(hostPriv)
	if err != nil {
		t.Fatalf("host signer: %v", err)
	}

	// The client key, written out in OpenSSH format for NewSSH to read.
	_, clientPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate client key: %v", err)
	}
	block, err := ssh.MarshalPrivateKey(clientPriv, "")
	if err != nil {
		t.Fatalf("marshal client key: %v", err)
	}
	keyPath := filepath.Join(t.TempDir(), "id_ed25519")
	if err := os.WriteFile(keyPath, pem.EncodeToMemory(block), 0o600); err != nil {
		t.Fatalf("write client key: %v", err)
	}

	cfg := &ssh.ServerConfig{
		// Any key is accepted: this is testing the executor, not the
		// server's authorization.
		PublicKeyCallback: func(ssh.ConnMetadata, ssh.PublicKey) (*ssh.Permissions, error) {
			return &ssh.Permissions{}, nil
		},
	}
	cfg.AddHostKey(hostSigner)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	s := &sshServer{t: t, ln: ln, cfg: cfg, scripts: map[string]reply{}}
	go s.serve()
	t.Cleanup(func() { ln.Close() })

	return s, keyPath
}

func (s *sshServer) script(substr string, r reply) *sshServer {
	s.scripts[substr] = r
	return s
}

func (s *sshServer) addr() string { return s.ln.Addr().String() }

func (s *sshServer) callLog() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.calls...)
}

// replyFor picks the longest matching script, so a specific command beats a
// general one — the same rule the fake executors elsewhere use.
func (s *sshServer) replyFor(cmd string) reply {
	best, found := "", reply{}
	for k, v := range s.scripts {
		if strings.Contains(cmd, k) && len(k) > len(best) {
			best, found = k, v
		}
	}
	return found
}

func (s *sshServer) serve() {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			return // listener closed by cleanup
		}
		go s.handleConn(conn)
	}
}

func (s *sshServer) handleConn(conn net.Conn) {
	defer conn.Close()

	sc, chans, reqs, err := ssh.NewServerConn(conn, s.cfg)
	if err != nil {
		return
	}
	defer sc.Close()
	go ssh.DiscardRequests(reqs)

	for nc := range chans {
		if nc.ChannelType() != "session" {
			_ = nc.Reject(ssh.UnknownChannelType, "only sessions here")
			continue
		}
		ch, chReqs, err := nc.Accept()
		if err != nil {
			return
		}
		go s.handleSession(ch, chReqs)
	}
}

func (s *sshServer) handleSession(ch ssh.Channel, reqs <-chan *ssh.Request) {
	defer ch.Close()

	for req := range reqs {
		if req.Type != "exec" {
			_ = req.Reply(false, nil)
			continue
		}
		// An exec payload is a single string, length-prefixed.
		var payload struct{ Command string }
		if err := ssh.Unmarshal(req.Payload, &payload); err != nil {
			_ = req.Reply(false, nil)
			return
		}
		_ = req.Reply(true, nil)

		s.mu.Lock()
		s.calls = append(s.calls, payload.Command)
		s.mu.Unlock()

		r := s.replyFor(payload.Command)
		_, _ = io.WriteString(ch, r.stdout)
		_, _ = io.WriteString(ch.Stderr(), r.stderr)

		status := make([]byte, 4)
		binary.BigEndian.PutUint32(status, r.exitCode)
		_, _ = ch.SendRequest("exit-status", false, status)
		return
	}
}

// dial connects the real executor to the fake server.
func (s *sshServer) dial(t *testing.T, keyPath string) Executor {
	t.Helper()
	host, portStr, err := net.SplitHostPort(s.addr())
	if err != nil {
		t.Fatal(err)
	}
	port := 0
	if _, err := fmtSscan(portStr, &port); err != nil {
		t.Fatal(err)
	}

	e, err := NewSSH(SSHConfig{
		Host:        host,
		Port:        port,
		User:        "root",
		KeyPath:     keyPath,
		HostKeyFile: filepath.Join(t.TempDir(), "known_hosts"),
	})
	if err != nil {
		t.Fatalf("NewSSH: %v", err)
	}
	t.Cleanup(func() { e.Close() })
	return e
}

func TestSSHRun_CarriesStdoutStderrAndExitCode(t *testing.T) {
	srv, key := newSSHServer(t)
	srv.script("uname", reply{stdout: "Linux\n"}).
		script("false", reply{stderr: "it failed\n", exitCode: 7})
	e := srv.dial(t, key)

	res, err := e.Run(context.Background(), "uname -a", nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Stdout != "Linux\n" {
		t.Errorf("Stdout = %q, want %q", res.Stdout, "Linux\n")
	}
	if res.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", res.ExitCode)
	}

	// A non-zero exit is a RESULT, not a transport error: the command ran
	// and said no, which every caller here distinguishes from "the link
	// broke".
	res, err = e.Run(context.Background(), "false", nil)
	if err != nil {
		t.Fatalf("a non-zero exit came back as an error: %v", err)
	}
	if res.ExitCode != 7 {
		t.Errorf("ExitCode = %d, want 7", res.ExitCode)
	}
	if !strings.Contains(res.Stderr, "it failed") {
		t.Errorf("Stderr = %q, want the command's own message", res.Stderr)
	}
}

// Streamed output arrives line by line, which is what drives the setup log.
func TestSSHRun_StreamsLines(t *testing.T) {
	srv, key := newSSHServer(t)
	srv.script("build", reply{stdout: "step one\nstep two\nno trailing newline"})
	e := srv.dial(t, key)

	var lines []string
	res, err := e.Run(context.Background(), "build everything", &RunOpts{
		Stream: func(l string) { lines = append(lines, l) },
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	want := []string{"step one", "step two", "no trailing newline"}
	if len(lines) != len(want) {
		t.Fatalf("streamed %#v, want %#v", lines, want)
	}
	for i := range want {
		if lines[i] != want[i] {
			t.Errorf("line %d = %q, want %q", i, lines[i], want[i])
		}
	}
	// Stdout is captured byte-exact alongside the streaming, with no
	// fabricated trailing newline.
	if !strings.HasSuffix(res.Stdout, "no trailing newline") {
		t.Errorf("Stdout = %q, want it byte-exact", res.Stdout)
	}
}

// A canceled context stops the command rather than blocking on a session that
// will never finish.
func TestSSHRun_ACanceledContextStopsTheCommand(t *testing.T) {
	srv, key := newSSHServer(t)
	e := srv.dial(t, key)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := e.Run(ctx, "sleep 60", nil); err == nil {
		t.Fatal("a canceled context still ran to completion")
	}
}

// WriteFile round-trips through the real base64 pipeline: the executor
// encodes, and the remote side is asked to decode into place.
func TestSSHWriteFile_ShipsAnEncodedPayloadAndChecksTheExit(t *testing.T) {
	srv, key := newSSHServer(t)
	e := srv.dial(t, key)

	content := []byte("networks:\n  - chainId: 369\n")
	if err := e.WriteFile(context.Background(), "/var/lib/valve-node-app/erpc.yaml", content, 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	var cmd string
	for _, c := range srv.callLog() {
		if strings.Contains(c, "base64 -d") {
			cmd = c
		}
	}
	if cmd == "" {
		t.Fatal("no write command reached the server")
	}
	// The payload travels encoded, so YAML's newlines and quotes cannot
	// break out of the shell command.
	if strings.Contains(cmd, "chainId") {
		t.Errorf("the payload was inlined rather than encoded: %s", cmd)
	}
	if !strings.Contains(cmd, base64.StdEncoding.EncodeToString(content)) {
		t.Errorf("the encoded payload is not in the command: %s", cmd)
	}
	if !strings.Contains(cmd, "chmod 600") {
		t.Errorf("the mode was not applied: %s", cmd)
	}
}

// A failed remote write is an error naming the path — silently succeeding
// here leaves a gateway pointing at a config that was never written.
func TestSSHWriteFile_AFailedRemoteWriteIsAnError(t *testing.T) {
	srv, key := newSSHServer(t)
	srv.script("base64 -d", reply{stderr: "No space left on device", exitCode: 1})
	e := srv.dial(t, key)

	err := e.WriteFile(context.Background(), "/var/lib/valve-node-app/erpc.yaml", []byte("x"), 0o600)
	if err == nil {
		t.Fatal("a failed remote write reported success")
	}
	if !strings.Contains(err.Error(), "erpc.yaml") {
		t.Errorf("the error does not name the file: %v", err)
	}
	if !strings.Contains(err.Error(), "No space left") {
		t.Errorf("the remote explanation was dropped: %v", err)
	}
}

func TestSSHReadFile_DecodesWhatTheTargetSent(t *testing.T) {
	srv, key := newSSHServer(t)
	want := []byte("-----BEGIN CERTIFICATE-----\nMIIB\n-----END CERTIFICATE-----\n")
	// Real `base64` wraps at 76 columns, so the decoder has to cope with
	// embedded newlines rather than assuming one long line.
	encoded := base64.StdEncoding.EncodeToString(want)
	wrapped := ""
	for len(encoded) > 0 {
		n := min(76, len(encoded))
		wrapped += encoded[:n] + "\n"
		encoded = encoded[n:]
	}
	srv.script("base64 <", reply{stdout: wrapped})
	e := srv.dial(t, key)

	got, err := e.ReadFile(context.Background(), "/var/lib/valve-node-app/caddy-root.crt")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != string(want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestSSHReadFile_ReportsAMissingRemoteFile(t *testing.T) {
	srv, key := newSSHServer(t)
	srv.script("base64 <", reply{stderr: "No such file or directory", exitCode: 1})
	e := srv.dial(t, key)

	_, err := e.ReadFile(context.Background(), "/nope")
	if err == nil {
		t.Fatal("reading a missing remote file reported success")
	}
	if !strings.Contains(err.Error(), "/nope") {
		t.Errorf("the error does not name the file: %v", err)
	}
}

// Output that is not base64 at all must fail rather than decode to garbage —
// a login banner on stdout is the usual cause.
func TestSSHReadFile_RefusesUndecodableOutput(t *testing.T) {
	srv, key := newSSHServer(t)
	srv.script("base64 <", reply{stdout: "Welcome to Ubuntu!\n!!!not base64!!!\n"})
	e := srv.dial(t, key)

	if _, err := e.ReadFile(context.Background(), "/etc/hostname"); err == nil {
		t.Fatal("undecodable output was accepted as file content")
	} else if !strings.Contains(err.Error(), "decode") {
		t.Errorf("the error does not say what went wrong: %v", err)
	}
}

// Commands run after Close must fail rather than panic on a dead client.
func TestSSHClose_EndsTheSession(t *testing.T) {
	srv, key := newSSHServer(t)
	e := srv.dial(t, key)

	if _, err := e.Run(context.Background(), "uname", nil); err != nil {
		t.Fatalf("Run before Close: %v", err)
	}
	if err := e.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if _, err := e.Run(context.Background(), "uname", nil); err == nil {
		t.Error("a command ran on a closed connection")
	}
}

// ---------------------------------------------------------------------
// NewSSH's own refusals
// ---------------------------------------------------------------------

func TestNewSSH_RefusesAKeyItCannotUse(t *testing.T) {
	dir := t.TempDir()
	bad := filepath.Join(dir, "not-a-key")
	if err := os.WriteFile(bad, []byte("this is not a private key"), 0o600); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		keyPath string
		wantSay string
	}{
		{name: "missing", keyPath: filepath.Join(dir, "absent"), wantSay: "read private key"},
		{name: "unparseable", keyPath: bad, wantSay: "parse private key"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewSSH(SSHConfig{
				Host: "127.0.0.1", Port: 1, User: "root",
				KeyPath:     tc.keyPath,
				HostKeyFile: filepath.Join(dir, "known_hosts"),
			})
			if err == nil {
				t.Fatal("an unusable key was accepted")
			}
			if !strings.Contains(err.Error(), tc.wantSay) {
				t.Errorf("error does not say what failed (want %q): %v", tc.wantSay, err)
			}
			// The path is named, so the operator knows which file to fix.
			if !strings.Contains(err.Error(), tc.keyPath) {
				t.Errorf("error does not name the key file: %v", err)
			}
		})
	}
}

// Port 0 means "the default", not "port zero". Dialling port 0 is not a
// connection at all, so any outcome that proves a real TCP peer was reached
// proves the substitution happened.
//
// Which outcome depends on the machine, and the test has to accept both: with
// nothing on :22 the dial is refused and the error carries the address, while
// on a machine running sshd the connection succeeds and fails later at
// authentication. Only the ":0" case is a failure of the code under test.
func TestNewSSH_DefaultsToPort22(t *testing.T) {
	_, key := newSSHServer(t)

	_, err := NewSSH(SSHConfig{
		Host: "127.0.0.1", Port: 0, User: "root",
		KeyPath:     key,
		HostKeyFile: filepath.Join(t.TempDir(), "known_hosts"),
	})
	if err == nil {
		// An sshd that accepted this key: it was reached on 22, since 0 is
		// not dialable.
		return
	}

	msg := err.Error()
	if strings.Contains(msg, ":0") {
		t.Fatalf("port 0 was used literally: %v", err)
	}
	reachedAPeer := strings.Contains(msg, ":22") || strings.Contains(msg, "handshake")
	if !reachedAPeer {
		t.Errorf("nothing shows port 0 became 22: %v", err)
	}
}

// A host that refuses the connection is an error, not a client that fails
// later on every command.
func TestNewSSH_ReportsAnUnreachableHost(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().(*net.TCPAddr)
	ln.Close() // nothing is listening there now

	_, keyPath := newSSHServer(t)

	done := make(chan error, 1)
	go func() {
		_, err := NewSSH(SSHConfig{
			Host: "127.0.0.1", Port: addr.Port, User: "root",
			KeyPath:     keyPath,
			HostKeyFile: filepath.Join(t.TempDir(), "known_hosts"),
		})
		done <- err
	}()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("dialing a closed port returned a working client")
		}
	case <-time.After(15 * time.Second):
		t.Fatal("NewSSH hung on an unreachable host instead of timing out")
	}
}

// fmtSscan is a tiny wrapper so this file does not import fmt just for one
// port parse.
func fmtSscan(s string, out *int) (int, error) {
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, errors.New("not a port: " + s)
		}
		n = n*10 + int(r-'0')
	}
	*out = n
	return 1, nil
}
