package config

import (
	"crypto/ed25519"
	"crypto/rand"
	"net"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/crypto/ssh"

	"github.com/valve-tech/jumpgate/internal/executor"
)

func TestDirIsJumpgate(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	got, err := Dir()
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(home, ".jumpgate"); got != want {
		t.Fatalf("Dir() = %q, want %q", got, want)
	}
}

// An existing install keeps its targets, keys and known_hosts: the old
// directory is moved once, and a pointer file is left so a reader of the old
// path learns where it went.
func TestMigrateLegacyDirMovesOnce(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	legacy := filepath.Join(home, ".valve-node-app")
	if err := os.MkdirAll(legacy, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacy, "config.json"), []byte(`{"targets":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}

	moved, err := MigrateLegacyDir()
	if err != nil || !moved {
		t.Fatalf("MigrateLegacyDir() = %v, %v; want true, nil", moved, err)
	}
	if _, err := os.Stat(filepath.Join(home, ".jumpgate", "config.json")); err != nil {
		t.Fatalf("config not moved: %v", err)
	}
	pointer, err := os.ReadFile(filepath.Join(legacy, "MOVED"))
	if err != nil || len(pointer) == 0 {
		t.Fatalf("no pointer file left behind: %v", err)
	}

	moved, err = MigrateLegacyDir()
	if err != nil || moved {
		t.Fatalf("second MigrateLegacyDir() = %v, %v; want false, nil", moved, err)
	}
}

// Both directories existing means someone already started fresh; merging
// two configs silently would lose one of them, so it is an error to resolve
// by hand.
func TestMigrateLegacyDirRefusesWhenBothExist(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	for _, d := range []string{".valve-node-app", ".jumpgate"} {
		if err := os.MkdirAll(filepath.Join(home, d), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(home, d, "config.json"), []byte(`{}`), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := MigrateLegacyDir(); err == nil {
		t.Fatal("want an error when both directories hold a config")
	}
}

func writeConfigJSON(t *testing.T, dir, body string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

// Stored control-plane paths are absolute, so moving the directory would leave
// every one of them pointing at nothing. They must follow the move; anything
// outside the legacy directory (a user-chosen key elsewhere) must not change.
func TestLoadRepointsPathsInsideTheLegacyDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	legacy := filepath.Join(home, ".valve-node-app")
	body := `{"targets":[{"id":"box","mode":"ssh","ssh":{"Host":"h","User":"root",` +
		`"KeyPath":"` + filepath.Join(legacy, "keys", "id") + `",` +
		`"HostKeyFile":"` + filepath.Join(legacy, "known_hosts") + `"}},` +
		`{"id":"other","mode":"ssh","ssh":{"Host":"o","User":"root","KeyPath":"/keys/elsewhere","HostKeyFile":"/var/lib/valve-node-app/known_hosts"}}]}`
	writeConfigJSON(t, legacy, body)

	if _, err := MigrateLegacyDir(); err != nil {
		t.Fatal(err)
	}
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	cur := filepath.Join(home, ".jumpgate")
	if got, want := c.Targets[0].SSH.HostKeyFile, filepath.Join(cur, "known_hosts"); got != want {
		t.Errorf("HostKeyFile = %q, want %q", got, want)
	}
	if got, want := c.Targets[0].SSH.KeyPath, filepath.Join(cur, "keys", "id"); got != want {
		t.Errorf("KeyPath = %q, want %q", got, want)
	}
	if got := c.Targets[1].SSH.KeyPath; got != "/keys/elsewhere" {
		t.Errorf("outside path changed: %q", got)
	}
	if got := c.Targets[1].SSH.HostKeyFile; got != "/var/lib/valve-node-app/known_hosts" {
		t.Errorf("on-box-looking path changed: %q", got)
	}
}

// The property the repointing exists for: a host key pinned before the move is
// still enforced after it, rather than the file being recreated empty and the
// next key trusted on first use.
func TestPinnedHostKeySurvivesMigration(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	legacy := filepath.Join(home, ".valve-node-app")
	writeConfigJSON(t, legacy, `{"targets":[{"id":"box","mode":"ssh","ssh":{"Host":"h","User":"root","KeyPath":"/k","HostKeyFile":"`+
		filepath.Join(legacy, "known_hosts")+`"}}]}`)

	newKey := func() ssh.PublicKey {
		pub, _, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		k, err := ssh.NewPublicKey(pub)
		if err != nil {
			t.Fatal(err)
		}
		return k
	}
	pinned, other := newKey(), newKey()
	addr := &net.TCPAddr{IP: net.ParseIP("203.0.113.9"), Port: 22}

	// Pin the host through the legacy file, exactly as a first connection did.
	if err := executor.TOFUHostKeyCallback(filepath.Join(legacy, "known_hosts"))("h:22", addr, pinned); err != nil {
		t.Fatal(err)
	}
	if _, err := MigrateLegacyDir(); err != nil {
		t.Fatal(err)
	}
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	cb := executor.TOFUHostKeyCallback(c.Targets[0].SSH.HostKeyFile)
	if err := cb("h:22", addr, other); err == nil {
		t.Fatal("a different key for a pinned host was accepted after migration")
	}
	if err := cb("h:22", addr, pinned); err != nil {
		t.Fatalf("the pinned key was refused after migration: %v", err)
	}
}
