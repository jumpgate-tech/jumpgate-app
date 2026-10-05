package executor

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	gliderssh "github.com/gliderlabs/ssh"
	"golang.org/x/crypto/ssh"

	"github.com/valve-tech/jumpgate/internal/testutil"
)

// testSSHD is an in-process sshd used to exercise the SSH Executor without
// touching a real remote host. Commands are executed locally via `sh -c` and
// exit codes are propagated, per the task brief.
type testSSHD struct {
	host    string
	port    int
	hostKey ssh.PublicKey
}

func startTestSSHD(t *testing.T) (testSSHD, string) {
	t.Helper()
	return startTestSSHDWith(t, func(s gliderssh.Session) {
		c := exec.CommandContext(s.Context(), "sh", "-c", s.RawCommand())
		c.Stdout = s
		c.Stderr = s.Stderr()
		c.Stdin = s
		runErr := c.Run()
		code := 0
		if runErr != nil {
			if exitErr, ok := runErr.(*exec.ExitError); ok {
				code = exitErr.ExitCode()
			} else {
				code = 1
			}
		}
		_ = s.Exit(code)
	})
}

// startTestSSHDWith starts an in-process sshd that runs handler for every
// session, and returns it with the path of a client key it accepts. configure,
// if given, may adjust the server before it starts.
func startTestSSHDWith(t *testing.T, handler gliderssh.Handler, configure ...func(*gliderssh.Server)) (testSSHD, string) {
	t.Helper()

	_, hostPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate host key: %v", err)
	}
	hostSigner, err := ssh.NewSignerFromKey(hostPriv)
	if err != nil {
		t.Fatalf("host signer: %v", err)
	}

	clientPub, clientPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate client key: %v", err)
	}
	clientKeyPath := writePrivateKey(t, clientPriv)
	_ = clientPub

	srv := &gliderssh.Server{
		PublicKeyHandler: func(ctx gliderssh.Context, key gliderssh.PublicKey) bool {
			return true
		},
		Handler: handler,
	}
	srv.AddHostKey(hostSigner)
	for _, f := range configure {
		f(srv)
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	go func() {
		_ = srv.Serve(ln)
	}()
	t.Cleanup(func() {
		_ = srv.Close()
	})

	addr := ln.Addr().(*net.TCPAddr)
	return testSSHD{
		host:    "127.0.0.1",
		port:    addr.Port,
		hostKey: hostSigner.PublicKey(),
	}, clientKeyPath
}

func writePrivateKey(t *testing.T, priv ed25519.PrivateKey) string {
	t.Helper()

	block, err := ssh.MarshalPrivateKey(priv, "")
	if err != nil {
		t.Fatalf("marshal private key: %v", err)
	}
	path := filepath.Join(t.TempDir(), "id_ed25519")
	if err := os.WriteFile(path, pem.EncodeToMemory(block), 0o600); err != nil {
		t.Fatalf("write private key: %v", err)
	}
	return path
}

// writeTempKey writes a fresh client private key and returns its path.
func writeTempKey(t *testing.T) string {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	return writePrivateKey(t, priv)
}

// startTestSSHDAcceptingOnly starts an sshd that authenticates only allowed.
func startTestSSHDAcceptingOnly(t *testing.T, allowed ssh.PublicKey) testSSHD {
	t.Helper()
	d, _ := startTestSSHDWith(t, func(s gliderssh.Session) { _ = s.Exit(0) }, func(srv *gliderssh.Server) {
		srv.PublicKeyHandler = func(_ gliderssh.Context, key gliderssh.PublicKey) bool {
			return bytes.Equal(key.Marshal(), allowed.Marshal())
		}
	})
	return d
}

// startTestSSHDWithForwarding starts an sshd that allows direct-tcpip, so it
// can serve as a jump host.
func startTestSSHDWithForwarding(t *testing.T) (testSSHD, string) {
	t.Helper()
	return startTestSSHDWith(t, func(s gliderssh.Session) { _ = s.Exit(0) }, func(srv *gliderssh.Server) {
		srv.LocalPortForwardingCallback = func(gliderssh.Context, string, uint32) bool { return true }
		srv.ChannelHandlers = map[string]gliderssh.ChannelHandler{
			"session":      gliderssh.DefaultSessionHandler,
			"direct-tcpip": gliderssh.DirectTCPIPHandler,
		}
	})
}

func newSSHConfig(t *testing.T, d testSSHD, keyPath string) SSHConfig {
	t.Helper()
	return SSHConfig{
		Host:        d.host,
		User:        "testuser",
		KeyPath:     keyPath,
		HostKeyFile: filepath.Join(t.TempDir(), "known_hosts"),
		Port:        d.port,
	}
}

func TestSSH_Run_CapturesStdout(t *testing.T) {
	testutil.RequirePOSIXShell(t) // the test sshd runs commands through sh
	d, keyPath := startTestSSHD(t)
	e, err := NewSSH(newSSHConfig(t, d, keyPath))
	if err != nil {
		t.Fatalf("NewSSH: %v", err)
	}
	t.Cleanup(func() { _ = e.Close() })

	res, err := e.Run(context.Background(), "echo hello", nil)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if res.Stdout != "hello\n" {
		t.Errorf("Stdout = %q, want %q", res.Stdout, "hello\n")
	}
	if res.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", res.ExitCode)
	}
}

func TestSSH_Run_CapturesStderrSeparately(t *testing.T) {
	testutil.RequirePOSIXShell(t) // the test sshd runs commands through sh
	d, keyPath := startTestSSHD(t)
	e, err := NewSSH(newSSHConfig(t, d, keyPath))
	if err != nil {
		t.Fatalf("NewSSH: %v", err)
	}
	t.Cleanup(func() { _ = e.Close() })

	res, err := e.Run(context.Background(), "echo out; echo err 1>&2", nil)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if res.Stdout != "out\n" {
		t.Errorf("Stdout = %q, want %q", res.Stdout, "out\n")
	}
	if res.Stderr != "err\n" {
		t.Errorf("Stderr = %q, want %q", res.Stderr, "err\n")
	}
}

func TestSSH_Run_NonZeroExitIsNotAnError(t *testing.T) {
	testutil.RequirePOSIXShell(t) // the test sshd runs commands through sh
	d, keyPath := startTestSSHD(t)
	e, err := NewSSH(newSSHConfig(t, d, keyPath))
	if err != nil {
		t.Fatalf("NewSSH: %v", err)
	}
	t.Cleanup(func() { _ = e.Close() })

	res, err := e.Run(context.Background(), "exit 3", nil)
	if err != nil {
		t.Fatalf("Run returned error for non-zero exit: %v", err)
	}
	if res.ExitCode != 3 {
		t.Errorf("ExitCode = %d, want 3", res.ExitCode)
	}
}

func TestSSH_Run_StreamReceivesLinesInOrder(t *testing.T) {
	testutil.RequirePOSIXShell(t) // the test sshd runs commands through sh
	d, keyPath := startTestSSHD(t)
	e, err := NewSSH(newSSHConfig(t, d, keyPath))
	if err != nil {
		t.Fatalf("NewSSH: %v", err)
	}
	t.Cleanup(func() { _ = e.Close() })

	var lines []string
	opts := &RunOpts{Stream: func(line string) {
		lines = append(lines, line)
	}}

	res, err := e.Run(context.Background(), `printf 'a\nb\n'`, opts)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if len(lines) != 2 || lines[0] != "a" || lines[1] != "b" {
		t.Errorf("streamed lines = %v, want [a b]", lines)
	}
	if res.Stdout != "a\nb\n" {
		t.Errorf("Stdout = %q, want %q", res.Stdout, "a\nb\n")
	}
}

func TestSSH_WriteFile_ReadFile_RoundTrips(t *testing.T) {
	testutil.RequirePOSIXShell(t) // the test sshd runs commands through sh
	d, keyPath := startTestSSHD(t)
	e, err := NewSSH(newSSHConfig(t, d, keyPath))
	if err != nil {
		t.Fatalf("NewSSH: %v", err)
	}
	t.Cleanup(func() { _ = e.Close() })

	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "file.txt")
	content := []byte("hello world\n")

	if err := e.WriteFile(context.Background(), path, content, 0o640); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	got, err := e.ReadFile(context.Background(), path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != string(content) {
		t.Errorf("ReadFile content = %q, want %q", got, content)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if info.Mode().Perm() != 0o640 {
		t.Errorf("mode = %v, want %v", info.Mode().Perm(), os.FileMode(0o640))
	}
}

// sshLargeLineCmd is a portable (no coreutils/GNU-isms) shell pipeline that
// prints exactly 2MB of 'x' characters with no trailing newline, as a
// single unbroken line — well past the historical 1MB scanner buffer cap.
const sshLargeLineCmd = "head -c 2097152 /dev/zero | tr '\\0' 'x'"

func TestSSH_Run_LargeSingleLineStdout_NoStream(t *testing.T) {
	testutil.RequirePOSIXShell(t) // the test sshd runs commands through sh
	d, keyPath := startTestSSHD(t)
	e, err := NewSSH(newSSHConfig(t, d, keyPath))
	if err != nil {
		t.Fatalf("NewSSH: %v", err)
	}
	t.Cleanup(func() { _ = e.Close() })

	const size = 2 * 1024 * 1024

	res, err := e.Run(context.Background(), sshLargeLineCmd, nil)
	if err != nil {
		t.Fatalf("Run returned error for a 2MB single line: %v", err)
	}
	if res.ExitCode != 0 {
		t.Fatalf("ExitCode = %d, want 0", res.ExitCode)
	}
	if len(res.Stdout) != size {
		t.Errorf("len(Stdout) = %d, want %d", len(res.Stdout), size)
	}
	if strings.Count(res.Stdout, "x") != size {
		t.Errorf("Stdout content is not all 'x'")
	}
}

func TestSSH_Run_LargeSingleLineStdout_WithStream(t *testing.T) {
	testutil.RequirePOSIXShell(t) // the test sshd runs commands through sh
	d, keyPath := startTestSSHD(t)
	e, err := NewSSH(newSSHConfig(t, d, keyPath))
	if err != nil {
		t.Fatalf("NewSSH: %v", err)
	}
	t.Cleanup(func() { _ = e.Close() })

	const size = 2 * 1024 * 1024
	opts := &RunOpts{Stream: func(line string) {}}

	res, err := e.Run(context.Background(), sshLargeLineCmd, opts)
	if err != nil {
		t.Fatalf("Run returned error for a 2MB single line with Stream set: %v", err)
	}
	if res.ExitCode != 0 {
		t.Fatalf("ExitCode = %d, want 0", res.ExitCode)
	}
	if len(res.Stdout) != size {
		t.Errorf("len(Stdout) = %d, want %d", len(res.Stdout), size)
	}
}

func TestSSH_TOFU_UnknownHostAppendsKey(t *testing.T) {
	d, keyPath := startTestSSHD(t)
	cfg := newSSHConfig(t, d, keyPath)

	if _, err := os.Stat(cfg.HostKeyFile); err == nil {
		t.Fatalf("HostKeyFile unexpectedly exists before first connect")
	}

	e, err := NewSSH(cfg)
	if err != nil {
		t.Fatalf("NewSSH (first connect, unknown host): %v", err)
	}
	t.Cleanup(func() { _ = e.Close() })

	if _, err := os.Stat(cfg.HostKeyFile); err != nil {
		t.Fatalf("HostKeyFile not created: %v", err)
	}
	testutil.AssertPrivate(t, cfg.HostKeyFile)

	data, err := os.ReadFile(cfg.HostKeyFile)
	if err != nil {
		t.Fatalf("read HostKeyFile: %v", err)
	}
	addr := net.JoinHostPort(d.host, strconv.Itoa(d.port))
	line := strings.TrimSpace(string(data))
	fields := strings.Fields(line)
	if len(fields) != 3 {
		t.Fatalf("HostKeyFile line = %q, want 3 whitespace-separated fields", line)
	}
	if fields[0] != addr {
		t.Errorf("HostKeyFile host field = %q, want %q", fields[0], addr)
	}
	if fields[1] != d.hostKey.Type() {
		t.Errorf("HostKeyFile keytype field = %q, want %q", fields[1], d.hostKey.Type())
	}
	gotKeyBytes, err := base64.StdEncoding.DecodeString(fields[2])
	if err != nil {
		t.Fatalf("decode base64 key field: %v", err)
	}
	if string(gotKeyBytes) != string(d.hostKey.Marshal()) {
		t.Errorf("HostKeyFile key bytes mismatch server host key")
	}

	// Second connect against the now-known host with the same key must
	// succeed without erroring and without duplicating the line.
	e2, err := NewSSH(cfg)
	if err != nil {
		t.Fatalf("NewSSH (second connect, known host): %v", err)
	}
	_ = e2.Close()

	data2, err := os.ReadFile(cfg.HostKeyFile)
	if err != nil {
		t.Fatalf("read HostKeyFile after second connect: %v", err)
	}
	if strings.Count(strings.TrimSpace(string(data2)), "\n")+1 != 1 {
		t.Errorf("HostKeyFile grew after known-host reconnect: %q", string(data2))
	}
}

func TestSSH_TOFU_MismatchedHostKeyErrors(t *testing.T) {
	d, keyPath := startTestSSHD(t)
	cfg := newSSHConfig(t, d, keyPath)

	// Pre-populate the host key file with a DIFFERENT key for this host:port.
	_, wrongPub, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate wrong key: %v", err)
	}
	wrongSigner, err := ssh.NewSignerFromKey(wrongPub)
	if err != nil {
		t.Fatalf("wrong signer: %v", err)
	}
	wrongKey := wrongSigner.PublicKey()

	addr := net.JoinHostPort(d.host, strconv.Itoa(d.port))
	line := fmt.Sprintf("%s %s %s\n", addr, wrongKey.Type(), base64.StdEncoding.EncodeToString(wrongKey.Marshal()))
	if err := os.MkdirAll(filepath.Dir(cfg.HostKeyFile), 0o700); err != nil {
		t.Fatalf("mkdir HostKeyFile parent: %v", err)
	}
	if err := os.WriteFile(cfg.HostKeyFile, []byte(line), 0o600); err != nil {
		t.Fatalf("pre-write HostKeyFile: %v", err)
	}

	_, err = NewSSH(cfg)
	if err == nil {
		t.Fatalf("NewSSH: expected error for mismatched host key, got nil")
	}
	if !strings.Contains(err.Error(), "host key") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "host key")
	}
}

// The content must never appear in the remote command line: any local user on
// the target can read every process's arguments from /proc.
func TestWriteFileCmdCarriesNoContent(t *testing.T) {
	cmd := writeFileCmd("/etc/wireguard/jumpgate0.conf", 0o600)
	for _, frag := range []string{"base64", "printf"} {
		if strings.Contains(cmd, frag) {
			t.Errorf("command still embeds content via %s: %s", frag, cmd)
		}
	}
	if !strings.Contains(cmd, "umask 077") {
		t.Errorf("temp file is not created under umask 077: %s", cmd)
	}
}

// Parent directories keep the caller's umask. Only the temp file is created
// 077, so a new directory a service must traverse is not made 0700 root.
func TestWriteFileDoesNotTightenNewParentDirs(t *testing.T) {
	cmd := writeFileCmd("/var/lib/x/y/file", 0o644)
	mk := strings.Index(cmd, "mkdir -p")
	um := strings.Index(cmd, "umask 077")
	if mk < 0 || um < 0 || mk > um {
		t.Fatalf("mkdir -p must run before umask 077 takes effect: %s", cmd)
	}
}

func TestSSH_WriteFile_LargeContentAndMode(t *testing.T) {
	testutil.RequirePOSIXShell(t) // the test sshd runs commands through sh
	d, keyPath := startTestSSHD(t)
	ex, err := NewSSH(newSSHConfig(t, d, keyPath))
	if err != nil {
		t.Fatal(err)
	}
	defer ex.Close()

	dir := t.TempDir()
	target := filepath.Join(dir, "sub", "big.bin")
	content := bytes.Repeat([]byte("0123456789abcdef"), 64*1024) // 1 MiB: far past ARG_MAX for argv
	if err := ex.WriteFile(context.Background(), target, content, 0o640); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	got, err := os.ReadFile(target)
	if err != nil || !bytes.Equal(got, content) {
		t.Fatalf("content mismatch (err %v, %d bytes)", err, len(got))
	}
	fi, _ := os.Stat(target)
	if fi.Mode().Perm() != 0o640 {
		t.Fatalf("mode = %o, want 640", fi.Mode().Perm())
	}
	entries, _ := os.ReadDir(filepath.Dir(target))
	if len(entries) != 1 {
		t.Fatalf("temp file left behind: %v", entries)
	}
}
