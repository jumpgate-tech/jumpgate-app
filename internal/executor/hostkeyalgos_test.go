package executor

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"net"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	gliderssh "github.com/gliderlabs/ssh"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

func ecdsaHostSigner(t *testing.T) ssh.Signer {
	t.Helper()
	k, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	s, err := ssh.NewSignerFromKey(k)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// I4: a box offering ECDSA and ed25519, known to the operator's OpenSSH only
// by its ed25519 key (ssh-keyscan -t ed25519, an older OpenSSH). x/crypto
// prefers ECDSA, so without HostKeyAlgorithms Strict saw the ECDSA key and
// called it a mismatch. Both the dial and the capture must ask for the type
// on record.
func TestStrictNegotiatesTheKnownHostKeyType(t *testing.T) {
	d, keyPath := startTestSSHDWith(t, func(s gliderssh.Session) { _ = s.Exit(0) }, func(srv *gliderssh.Server) {
		srv.AddHostKey(ecdsaHostSigner(t))
	})
	dir := t.TempDir()
	openssh, confirmed := filepath.Join(dir, "known_hosts"), filepath.Join(dir, "confirmed_hosts")
	hp := net.JoinHostPort(d.host, strconv.Itoa(d.port))
	writeLines(t, openssh, knownhosts.Line([]string{knownhosts.Normalize(hp)}, d.hostKey))
	cfg := SSHConfig{
		Host: d.host, Port: d.port, User: "x", KeyPath: keyPath,
		HostKey:           Strict(confirmed, openssh),
		HostKeyAlgorithms: KnownHostKeyAlgorithms(confirmed, openssh),
	}

	c, err := DialSSH(context.Background(), cfg)
	if err != nil {
		t.Fatalf("DialSSH: %v", err)
	}
	c.Close()

	key, err := CaptureHostKey(context.Background(), cfg)
	if err != nil {
		t.Fatalf("CaptureHostKey: %v", err)
	}
	if key.Type() != ssh.KeyAlgoED25519 {
		t.Fatalf("captured a %s key, want the ed25519 key on record", key.Type())
	}
	if err := cfg.HostKey(hp, nil, key); err != nil {
		t.Fatalf("Strict refused the captured key: %v", err)
	}
}

// A box that no longer offers any key type on record cannot negotiate; that
// is a changed host key (exit 4), never "unreachable".
func TestStrictDialWithNoCommonHostKeyTypeIsAMismatch(t *testing.T) {
	d, keyPath := startTestSSHD(t) // ed25519 only
	dir := t.TempDir()
	openssh, confirmed := filepath.Join(dir, "known_hosts"), filepath.Join(dir, "confirmed_hosts")
	hp := net.JoinHostPort(d.host, strconv.Itoa(d.port))
	writeLines(t, openssh, knownhosts.Line([]string{knownhosts.Normalize(hp)}, ecdsaHostSigner(t).PublicKey()))
	_, err := DialSSH(context.Background(), SSHConfig{
		Host: d.host, Port: d.port, User: "x", KeyPath: keyPath,
		HostKey: Strict(confirmed, openssh), HostKeyAlgorithms: KnownHostKeyAlgorithms(confirmed, openssh),
	})
	if !errors.Is(err, ErrHostKeyMismatch) {
		t.Fatalf("err = %v, want ErrHostKeyMismatch", err)
	}
}

func TestKnownHostKeyAlgorithms(t *testing.T) {
	dir := t.TempDir()
	confirmed, openssh := filepath.Join(dir, "confirmed_hosts"), filepath.Join(dir, "known_hosts")
	rk, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	rsaKey, _ := ssh.NewPublicKey(&rk.PublicKey)
	if err := RecordHostKey(confirmed, "203.0.113.7:22", rsaKey); err != nil {
		t.Fatal(err)
	}
	if err := RecordHostKey(confirmed, "203.0.113.8:2222", newHostKey(t)); err != nil {
		t.Fatal(err)
	}
	writeLines(t, openssh,
		knownhosts.Line([]string{knownhosts.Normalize("203.0.113.8:2222")}, ecdsaHostSigner(t).PublicKey()),
		knownhosts.Line([]string{knownhosts.Normalize("203.0.113.8:2222")}, newHostKey(t)),
		"@cert-authority 203.0.113.9 "+strings.TrimSpace(string(ssh.MarshalAuthorizedKey(newHostKey(t)))),
	)
	algos := KnownHostKeyAlgorithms(confirmed, openssh, filepath.Join(dir, "absent"))

	cases := map[string][]string{
		"203.0.113.7:22":   {ssh.KeyAlgoRSASHA512, ssh.KeyAlgoRSASHA256},
		"203.0.113.8:2222": {ssh.KeyAlgoED25519, ssh.KeyAlgoECDSA256},
		"203.0.113.9:22":   nil, // only a CA covers it: keep the defaults
		"198.51.100.1:22":  nil, // unknown everywhere: keep the defaults
	}
	for hp, want := range cases {
		if got := algos(hp); !reflect.DeepEqual(got, want) {
			t.Errorf("%s: %v, want %v", hp, got, want)
		}
	}
}

// I5: every way a presented key contradicts the record carries one sentinel,
// so callers can report a security failure rather than an outage.
func TestHostKeyMismatchesWrapTheSentinel(t *testing.T) {
	dir := t.TempDir()
	confirmed, openssh := filepath.Join(dir, "confirmed_hosts"), filepath.Join(dir, "known_hosts")
	k1, k2 := newHostKey(t), newHostKey(t)
	if err := RecordHostKey(confirmed, "203.0.113.7:22", k1); err != nil {
		t.Fatal(err)
	}
	if err := Strict(confirmed)("203.0.113.7:22", testRemoteAddr, k2); !errors.Is(err, ErrHostKeyMismatch) {
		t.Errorf("confirmed-store mismatch: %v", err)
	}
	writeLines(t, openssh, knownhosts.Line([]string{knownhosts.Normalize("203.0.113.7:22")}, k2))
	if err := Strict(confirmed, openssh)("203.0.113.7:22", testRemoteAddr, k1); !errors.Is(err, ErrHostKeyMismatch) {
		t.Errorf("OpenSSH mismatch: %v", err)
	}
	writeLines(t, openssh, "@revoked * "+strings.TrimSpace(string(ssh.MarshalAuthorizedKey(k1))))
	if err := Strict(confirmed, openssh)("203.0.113.7:22", testRemoteAddr, k1); !errors.Is(err, ErrHostKeyMismatch) {
		t.Errorf("revoked: %v", err)
	}
	tofu := filepath.Join(dir, "tofu")
	if err := TOFUHostKeyCallback(tofu)("203.0.113.7:22", testRemoteAddr, k1); err != nil {
		t.Fatal(err)
	}
	if err := TOFUHostKeyCallback(tofu)("203.0.113.7:22", testRemoteAddr, k2); !errors.Is(err, ErrHostKeyMismatch) {
		t.Errorf("TOFU mismatch: %v", err)
	}
	if err := Strict(confirmed)("198.51.100.1:22", testRemoteAddr, k1); errors.Is(err, ErrHostKeyMismatch) {
		t.Errorf("an unknown host is not a mismatch: %v", err)
	}
}

// The dial keeps the typed errors: callers map them to exit codes.
func TestDialSSHKeepsTheHostKeyErrorTypes(t *testing.T) {
	d, keyPath := startTestSSHD(t)
	dir := t.TempDir()
	_, err := DialSSH(context.Background(), SSHConfig{Host: d.host, Port: d.port, User: "x", KeyPath: keyPath,
		HostKey: Strict(filepath.Join(dir, "confirmed_hosts"))})
	var unknown *UnknownHostError
	if !errors.As(err, &unknown) {
		t.Fatalf("unknown host: err = %v, want *UnknownHostError", err)
	}
}
