package config

import (
	"crypto/ed25519"
	"crypto/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"

	"github.com/valve-tech/jumpgate/internal/executor"
)

// RecordedHostKeys reports each store separately, from the same files
// StrictHostKey reads.
func TestRecordedHostKeysSeparatesTheStores(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	key := func() ssh.PublicKey {
		_, priv, _ := ed25519.GenerateKey(rand.Reader)
		s, _ := ssh.NewSignerFromKey(priv)
		return s.PublicKey()
	}
	mine, theirs := key(), key()
	confirmed, err := ConfirmedHostsFile()
	if err != nil {
		t.Fatal(err)
	}
	if err := executor.RecordHostKey(confirmed, "203.0.113.5:2200", mine); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(home, ".ssh"), 0o700); err != nil {
		t.Fatal(err)
	}
	line := "[203.0.113.5]:2200 " + strings.TrimSpace(string(ssh.MarshalAuthorizedKey(theirs))) + "\n"
	if err := os.WriteFile(filepath.Join(home, ".ssh", "known_hosts"), []byte(line), 0o600); err != nil {
		t.Fatal(err)
	}
	c, o, err := RecordedHostKeys("203.0.113.5:2200")
	if err != nil || len(c) != 1 || len(o) != 1 || executor.Fingerprint(c[0]) != executor.Fingerprint(mine) || executor.Fingerprint(o[0]) != executor.Fingerprint(theirs) {
		t.Fatalf("confirmed %v, openssh %v, %v", c, o, err)
	}
	if c, o, err := RecordedHostKeys("198.51.100.1:22"); err != nil || len(c)+len(o) != 0 {
		t.Fatalf("unknown host: %v %v %v", c, o, err)
	}
}
