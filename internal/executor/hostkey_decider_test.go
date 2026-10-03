package executor

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"net"
	"os"
	"path/filepath"
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
