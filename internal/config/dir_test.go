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
	"github.com/valve-tech/jumpgate/internal/testutil"
)

func TestDirIsJumpgate(t *testing.T) {
	home := testutil.Home(t)
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
	home := testutil.Home(t)
	legacy := filepath.Join(home, ".valve-node-app")
	if err := os.MkdirAll(legacy, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacy, "config.json"), []byte(`{"targets":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}

	moved, _, err := MigrateLegacyDir()
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

	moved, _, err = MigrateLegacyDir()
	if err != nil || moved {
		t.Fatalf("second MigrateLegacyDir() = %v, %v; want false, nil", moved, err)
	}
}

// Both directories existing means someone already started fresh; merging
// two configs silently would lose one of them, so it is an error to resolve
// by hand.
func TestMigrateLegacyDirRefusesWhenBothExist(t *testing.T) {
	home := testutil.Home(t)
	for _, d := range []string{".valve-node-app", ".jumpgate"} {
		if err := os.MkdirAll(filepath.Join(home, d), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(home, d, "config.json"), []byte(`{}`), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if _, _, err := MigrateLegacyDir(); err == nil {
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
	home := testutil.Home(t)
	legacy := filepath.Join(home, ".valve-node-app")
	body := `{"targets":[{"id":"box","mode":"ssh","ssh":{"Host":"h","User":"root",` +
		`"KeyPath":"` + filepath.Join(legacy, "keys", "id") + `",` +
		`"HostKeyFile":"` + filepath.Join(legacy, "known_hosts") + `"}},` +
		`{"id":"other","mode":"ssh","ssh":{"Host":"o","User":"root","KeyPath":"/keys/elsewhere","HostKeyFile":"/var/lib/valve-node-app/known_hosts"}}]}`
	writeConfigJSON(t, legacy, body)

	if _, _, err := MigrateLegacyDir(); err != nil {
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
	home := testutil.Home(t)
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
	if _, _, err := MigrateLegacyDir(); err != nil {
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

func TestConfirmedHostsFileLivesInTheConfigDirAndIsNotTheTOFUFile(t *testing.T) {
	home := testutil.Home(t)
	got, err := ConfirmedHostsFile()
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(home, ".jumpgate", "confirmed_hosts"); got != want {
		t.Fatalf("ConfirmedHostsFile = %q, want %q", got, want)
	}
	if filepath.Base(got) == "known_hosts" {
		t.Fatal("must not share the trust-on-first-use file")
	}
}

// legacyInstall writes a pre-rename install: a config, a pinned known_hosts
// and a key under keys/.
func legacyInstall(t *testing.T, home string) string {
	t.Helper()
	legacy := filepath.Join(home, ".valve-node-app")
	writeConfigJSON(t, legacy, `{"targets":[]}`)
	if err := os.WriteFile(filepath.Join(legacy, "known_hosts"), []byte("pinned\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(legacy, "keys"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacy, "keys", "id"), []byte("legacy key"), 0o600); err != nil {
		t.Fatal(err)
	}
	return legacy
}

func mustRead(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}

// I1: any CLI command creates ~/.jumpgate/config.json.lock before the
// migration runs. That jumpgate-made artefact must not block the move.
func TestMigrateLegacyDirMergesIntoADirHoldingOnlyTheLock(t *testing.T) {
	home := testutil.Home(t)
	legacy := legacyInstall(t, home)
	cur := filepath.Join(home, ".jumpgate")
	if err := os.MkdirAll(cur, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cur, "config.json.lock"), nil, 0o600); err != nil {
		t.Fatal(err)
	}

	moved, kept, err := MigrateLegacyDir()
	if err != nil || !moved || len(kept) != 0 {
		t.Fatalf("MigrateLegacyDir() = %v, %v, %v; want true, none kept, nil", moved, kept, err)
	}
	if got := mustRead(t, filepath.Join(cur, "config.json")); got != `{"targets":[]}` {
		t.Errorf("config.json = %q", got)
	}
	if got := mustRead(t, filepath.Join(cur, "known_hosts")); got != "pinned\n" {
		t.Errorf("known_hosts = %q", got)
	}
	if got := mustRead(t, filepath.Join(cur, "keys", "id")); got != "legacy key" {
		t.Errorf("keys/id = %q", got)
	}
	if _, err := os.Stat(filepath.Join(cur, "config.json.lock")); err != nil {
		t.Errorf("the lock file was lost: %v", err)
	}
	if _, err := os.Stat(filepath.Join(legacy, "MOVED")); err != nil {
		t.Errorf("no pointer file: %v", err)
	}
	if _, err := os.Stat(filepath.Join(legacy, "config.json")); !os.IsNotExist(err) {
		t.Errorf("legacy config.json still present: %v", err)
	}
	if moved, _, err := MigrateLegacyDir(); err != nil || moved {
		t.Fatalf("second run = %v, %v; want false, nil", moved, err)
	}
}

// I1: `jumpgate status` creates ~/.jumpgate/run/ (and may leave server files
// in it) before spawning serve.
func TestMigrateLegacyDirMergesIntoADirHoldingOnlyRun(t *testing.T) {
	home := testutil.Home(t)
	legacyInstall(t, home)
	cur := filepath.Join(home, ".jumpgate")
	if err := os.MkdirAll(filepath.Join(cur, "run"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cur, "run", "server.lock"), nil, 0o600); err != nil {
		t.Fatal(err)
	}

	moved, kept, err := MigrateLegacyDir()
	if err != nil || !moved || len(kept) != 0 {
		t.Fatalf("MigrateLegacyDir() = %v, %v, %v; want true, none kept, nil", moved, kept, err)
	}
	if _, err := os.Stat(filepath.Join(cur, "config.json")); err != nil {
		t.Errorf("config.json not moved: %v", err)
	}
	if _, err := os.Stat(filepath.Join(cur, "run", "server.lock")); err != nil {
		t.Errorf("run/server.lock lost: %v", err)
	}
}

// R24: a merge never overwrites a jumpgate-created file. Both directories
// hold keys/; the files inside merge, and a clash leaves both copies, reports
// the legacy one and keeps the stored path pointing at it, so a pinned host
// key is never swapped for another file.
func TestMigrateLegacyDirMergeKeepsBothCopiesOnACollision(t *testing.T) {
	home := testutil.Home(t)
	legacy := filepath.Join(home, ".valve-node-app")
	writeConfigJSON(t, legacy, `{"targets":[{"id":"box","mode":"ssh","ssh":{"Host":"h","User":"root","KeyPath":"/k","HostKeyFile":"`+
		filepath.Join(legacy, "known_hosts")+`"}}]}`)
	if err := os.WriteFile(filepath.Join(legacy, "known_hosts"), []byte("pinned\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(legacy, "keys"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacy, "keys", "id"), []byte("legacy key"), 0o600); err != nil {
		t.Fatal(err)
	}
	cur := filepath.Join(home, ".jumpgate")
	if err := os.MkdirAll(filepath.Join(cur, "keys"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cur, "keys", "controller.key"), []byte("new"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cur, "known_hosts"), []byte("other\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	moved, kept, err := MigrateLegacyDir()
	if err != nil || !moved {
		t.Fatalf("MigrateLegacyDir() = %v, %v; want true, nil", moved, err)
	}
	if len(kept) != 1 || kept[0] != filepath.Join(legacy, "known_hosts") {
		t.Fatalf("kept = %v, want the legacy known_hosts only", kept)
	}
	if got := mustRead(t, filepath.Join(cur, "known_hosts")); got != "other\n" {
		t.Errorf("current known_hosts overwritten: %q", got)
	}
	if got := mustRead(t, filepath.Join(legacy, "known_hosts")); got != "pinned\n" {
		t.Errorf("legacy known_hosts lost: %q", got)
	}
	if got := mustRead(t, filepath.Join(cur, "keys", "id")); got != "legacy key" {
		t.Errorf("keys/id = %q", got)
	}
	if got := mustRead(t, filepath.Join(cur, "keys", "controller.key")); got != "new" {
		t.Errorf("keys/controller.key = %q", got)
	}
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if got, want := c.Targets[0].SSH.HostKeyFile, filepath.Join(legacy, "known_hosts"); got != want {
		t.Errorf("HostKeyFile = %q, want the kept legacy file %q", got, want)
	}
}
