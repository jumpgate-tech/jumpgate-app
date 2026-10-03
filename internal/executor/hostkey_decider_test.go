package executor

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

func newHostKey(t *testing.T) ssh.PublicKey {
	t.Helper()
	pub, _, _ := ed25519.GenerateKey(rand.Reader)
	k, _ := ssh.NewPublicKey(pub)
	return k
}

var testRemoteAddr = &net.TCPAddr{IP: net.IPv4(203, 0, 113, 7), Port: 22}

func TestStrictRefusesAnUnknownHostWithItsFingerprint(t *testing.T) {
	cb := Strict(filepath.Join(t.TempDir(), "known_hosts"))
	key := newHostKey(t)
	err := cb("203.0.113.7:22", testRemoteAddr, key)
	var unknown *UnknownHostError
	if !errors.As(err, &unknown) || !errors.Is(err, ErrUnknownHost) {
		t.Fatalf("err = %v, want *UnknownHostError", err)
	}
	if unknown.Fingerprint != Fingerprint(key) {
		t.Errorf("fingerprint = %q, want %q", unknown.Fingerprint, Fingerprint(key))
	}
}

func TestStrictAcceptsARecordedKeyAndRefusesAChangedOne(t *testing.T) {
	file := filepath.Join(t.TempDir(), "known_hosts")
	key := newHostKey(t)
	if err := RecordHostKey(file, "203.0.113.7:22", key); err != nil {
		t.Fatal(err)
	}
	cb := Strict(file)
	if err := cb("203.0.113.7:22", testRemoteAddr, key); err != nil {
		t.Fatalf("recorded key refused: %v", err)
	}
	if err := cb("203.0.113.7:22", testRemoteAddr, newHostKey(t)); err == nil || errors.Is(err, ErrUnknownHost) {
		t.Fatalf("changed key: err = %v, want a mismatch error", err)
	}
}

// An operator who already ssh'd to the box has verified it; jumpgate trusts
// their OpenSSH known_hosts rather than asking again.
func TestStrictTrustsOpenSSHKnownHosts(t *testing.T) {
	dir := t.TempDir()
	openssh := filepath.Join(dir, "ssh_known_hosts")
	key := newHostKey(t)
	line := knownhosts.Line([]string{knownhosts.Normalize("203.0.113.7:22")}, key)
	if err := os.WriteFile(openssh, []byte(line+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cb := Strict(filepath.Join(dir, "jg_known_hosts"), openssh)
	if err := cb("203.0.113.7:22", testRemoteAddr, key); err != nil {
		t.Fatalf("key in OpenSSH known_hosts refused: %v", err)
	}
	if err := cb("203.0.113.7:22", testRemoteAddr, newHostKey(t)); err == nil || errors.Is(err, ErrUnknownHost) {
		t.Fatalf("mismatch against OpenSSH known_hosts: err = %v, want a mismatch error", err)
	}
}

func TestCaptureHostKeyNeedsNoCredentials(t *testing.T) {
	d, _ := startTestSSHD(t)
	key, err := CaptureHostKey(context.Background(), SSHConfig{Host: d.host, Port: d.port, User: "nobody"})
	if err != nil {
		t.Fatalf("CaptureHostKey: %v", err)
	}
	if string(key.Marshal()) != string(d.hostKey.Marshal()) {
		t.Fatal("captured the wrong key")
	}
}

// The forwarded channel's SetDeadline is unsupported, so capturing through a
// jump host to a target that stalls must be bounded by closing the connection.
func TestCaptureHostKeyIsBoundedThroughAJumpHost(t *testing.T) {
	shortHandshakeTimeout(t)
	jump, keyPath := startTestSSHDWithForwarding(t)
	silent := listenAndStall(t, "tcp", "127.0.0.1:0").Addr().(*net.TCPAddr)

	start := time.Now()
	_, err := CaptureHostKey(context.Background(), SSHConfig{
		Host: "127.0.0.1", Port: silent.Port, User: "x",
		Jump: &SSHConfig{Host: jump.host, Port: jump.port, User: "x", KeyPath: keyPath,
			HostKey: ssh.FixedHostKey(jump.hostKey)},
	})
	if err == nil {
		t.Fatal("capture from a silent target succeeded")
	}
	if time.Since(start) > 3*time.Second {
		t.Fatalf("took %v; the capture was not bounded through the jump host", time.Since(start))
	}
}

// TOFU records whatever answered first; nobody confirmed that key, so Strict
// reading the same file would launder it into a trusted one.
func TestStrictIgnoresKeysTOFURecorded(t *testing.T) {
	dir := t.TempDir()
	tofuFile := filepath.Join(dir, "known_hosts")
	key := newHostKey(t)
	if err := tofuHostKeyCallback(tofuFile)("203.0.113.7:22", testRemoteAddr, key); err != nil {
		t.Fatal(err)
	}
	cb := Strict(filepath.Join(dir, "confirmed_hosts"))
	if err := cb("203.0.113.7:22", testRemoteAddr, key); !errors.Is(err, ErrUnknownHost) {
		t.Fatalf("err = %v, want ErrUnknownHost for a TOFU-only key", err)
	}
}

func TestStrictDialRefusesAnUnconfirmedJumpHostAndWritesNothing(t *testing.T) {
	dir := t.TempDir()
	confirmed := filepath.Join(dir, "confirmed_hosts")
	jump, keyPath := startTestSSHDWithForwarding(t)
	_, err := DialSSH(context.Background(), SSHConfig{
		Host: "127.0.0.1", Port: 1, User: "x", KeyPath: keyPath,
		HostKey: Strict(confirmed),
		Jump:    &SSHConfig{Host: jump.host, Port: jump.port, User: "x", KeyPath: keyPath, HostKeyFile: filepath.Join(dir, "known_hosts")},
	})
	var unknown *UnknownHostError
	if !errors.As(err, &unknown) {
		t.Fatalf("err = %v, want *UnknownHostError for the jump host", err)
	}
	if want := net.JoinHostPort(jump.host, strconv.Itoa(jump.port)); unknown.Host != want {
		t.Errorf("unknown host = %q, want the jump host %q", unknown.Host, want)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 0 {
		t.Fatalf("files written: %v", entries)
	}
}

func TestCaptureHostKeyThroughAJumpNeedsAConfirmedJump(t *testing.T) {
	jump, keyPath := startTestSSHDWithForwarding(t)
	_, err := CaptureHostKey(context.Background(), SSHConfig{
		Host: "127.0.0.1", Port: 1, User: "x",
		Jump: &SSHConfig{Host: jump.host, Port: jump.port, User: "x", KeyPath: keyPath},
	})
	if err == nil || !strings.Contains(err.Error(), "confirm the jump host") {
		t.Fatalf("err = %v, want a demand to confirm the jump host first", err)
	}
}

func writeLines(t *testing.T, path string, lines ...string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestStrictRefusesWhenOpenSSHHasADifferentKeyEvenIfJumpgateMatches(t *testing.T) {
	dir := t.TempDir()
	confirmed, openssh := filepath.Join(dir, "confirmed_hosts"), filepath.Join(dir, "ssh_known_hosts")
	k1, k2 := newHostKey(t), newHostKey(t)
	if err := RecordHostKey(confirmed, "203.0.113.7:22", k1); err != nil {
		t.Fatal(err)
	}
	writeLines(t, openssh, knownhosts.Line([]string{knownhosts.Normalize("203.0.113.7:22")}, k2))
	err := Strict(confirmed, openssh)("203.0.113.7:22", testRemoteAddr, k1)
	if err == nil || errors.Is(err, ErrUnknownHost) {
		t.Fatalf("err = %v, want a mismatch error", err)
	}
}

func TestStrictRefusesAKeyOpenSSHRevokedEvenIfJumpgateMatches(t *testing.T) {
	dir := t.TempDir()
	confirmed, openssh := filepath.Join(dir, "confirmed_hosts"), filepath.Join(dir, "ssh_known_hosts")
	k1 := newHostKey(t)
	if err := RecordHostKey(confirmed, "203.0.113.7:22", k1); err != nil {
		t.Fatal(err)
	}
	writeLines(t, openssh, "@revoked * "+strings.TrimSpace(string(ssh.MarshalAuthorizedKey(k1))))
	err := Strict(confirmed, openssh)("203.0.113.7:22", testRemoteAddr, k1)
	if err == nil || errors.Is(err, ErrUnknownHost) {
		t.Fatalf("err = %v, want a revoked error", err)
	}
}

func TestStrictDoesNotPanicOnANilRemote(t *testing.T) {
	dir := t.TempDir()
	openssh := filepath.Join(dir, "ssh_known_hosts")
	writeLines(t, openssh, knownhosts.Line([]string{"198.51.100.1"}, newHostKey(t)))
	cb := Strict(filepath.Join(dir, "confirmed_hosts"), openssh)
	for _, host := range []string{"203.0.113.7:22", "example.test:2222", "[2001:db8::1]:22"} {
		if err := cb(host, nil, newHostKey(t)); !errors.Is(err, ErrUnknownHost) {
			t.Errorf("%s: err = %v, want ErrUnknownHost", host, err)
		}
	}
}

// A host only covered by a @cert-authority line is not a mismatch: nobody has
// pinned a host key for it, so the confirm flow must be able to start.
func TestStrictTreatsCertAuthorityOnlyEntriesAsUnknown(t *testing.T) {
	dir := t.TempDir()
	openssh := filepath.Join(dir, "ssh_known_hosts")
	writeLines(t, openssh, "@cert-authority 203.0.113.7 "+strings.TrimSpace(string(ssh.MarshalAuthorizedKey(newHostKey(t)))))
	err := Strict(filepath.Join(dir, "confirmed_hosts"), openssh)("203.0.113.7:22", testRemoteAddr, newHostKey(t))
	if !errors.Is(err, ErrUnknownHost) {
		t.Fatalf("err = %v, want ErrUnknownHost", err)
	}
}
