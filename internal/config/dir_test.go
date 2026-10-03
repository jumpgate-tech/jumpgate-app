package config

import (
	"os"
	"path/filepath"
	"testing"
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
