package config

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/valve-tech/jumpgate/internal/fsperm"
	"github.com/valve-tech/jumpgate/internal/testutil"
)

// P29: a secret that is never rewritten (an old transport key, files moved
// from ~/.valve-node-app, a known_hosts others can append to) is tightened
// at startup, along with the directories and a key file kept outside them.
func TestTightenStateRestrictsEveryExistingSecret(t *testing.T) {
	home := testutil.Home(t)
	dir := filepath.Join(home, ".jumpgate")
	outsideKey := filepath.Join(home, "elsewhere", "controller.key")
	files := []string{
		filepath.Join(dir, "config.json"),
		filepath.Join(dir, "run", "server.json"),
		filepath.Join(dir, "confirmed_hosts"),
		filepath.Join(dir, "known_hosts"),
		filepath.Join(dir, "ssh", "jumpgate_ed25519"),
		filepath.Join(dir, "keys", "controller.key"),
		outsideKey,
	}
	for _, f := range files {
		if err := os.MkdirAll(filepath.Dir(f), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		testutil.Loosen(t, f)
	}
	dirs := []string{dir, filepath.Join(dir, "run"), filepath.Join(dir, "ssh"), filepath.Join(dir, "keys")}
	for _, d := range dirs {
		testutil.Loosen(t, d)
	}

	tightened, warnings, err := TightenState(outsideKey)
	if err != nil || len(warnings) != 0 {
		t.Fatalf("TightenState = %v, warnings %v", err, warnings)
	}
	for _, p := range append(files, dirs...) {
		if err := fsperm.CheckPrivate(p); err != nil {
			t.Errorf("not tightened: %v", err)
		}
	}
	for _, f := range files {
		if !slices.Contains(tightened, f) {
			t.Errorf("%s was tightened but not reported (reported: %v)", f, tightened)
		}
	}

	// A second start finds nothing to do and reports nothing.
	if again, warnings, err := TightenState(outsideKey); len(again) != 0 || len(warnings) != 0 || err != nil {
		t.Fatalf("second run = %v, %v, %v; want nothing", again, warnings, err)
	}
}

// Missing files are skipped, not errors: a fresh install has none of them.
func TestTightenStateSkipsMissingFiles(t *testing.T) {
	home := testutil.Home(t)
	tightened, warnings, err := TightenState(filepath.Join(home, "no", "such.key"))
	if len(tightened) != 0 || len(warnings) != 0 || err != nil {
		t.Fatalf("TightenState on an empty home = %v, %v, %v; want nothing", tightened, warnings, err)
	}
	if err := fsperm.CheckPrivate(filepath.Join(home, ".jumpgate")); err != nil {
		t.Fatalf("the state directory was not created private: %v", err)
	}
}

// A signing key that cannot be made private stops startup, naming the file;
// any other file that cannot be is only a warning. A symlink stands in for
// "cannot be made private": MakePrivate refuses to follow one. On Windows a
// link has its own DACL, which may well be private, so this is unix-only.
func TestTightenStateFailsOnlyForASigningKeyItCannotRestrict(t *testing.T) {
	testutil.RequireUnix(t)
	home := testutil.Home(t)
	dir := filepath.Join(home, ".jumpgate")
	target := filepath.Join(home, "target")
	if err := os.WriteFile(target, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "ssh"), 0o700); err != nil {
		t.Fatal(err)
	}
	known := filepath.Join(dir, "known_hosts")
	if err := os.Symlink(target, known); err != nil {
		t.Fatal(err)
	}
	_, warnings, err := TightenState("")
	if err != nil || len(warnings) != 1 || !strings.Contains(warnings[0].Error(), known) {
		t.Fatalf("a symlinked known_hosts: err %v, warnings %v; want one warning naming it", err, warnings)
	}

	for _, key := range []string{filepath.Join(home, "controller.key"), filepath.Join(dir, "ssh", "jumpgate_ed25519")} {
		if err := os.Symlink(target, key); err != nil {
			t.Fatal(err)
		}
		keyFile := ""
		if filepath.Base(key) == "controller.key" {
			keyFile = key
		}
		_, _, err := TightenState(keyFile)
		if err == nil || !strings.Contains(err.Error(), key) || !strings.Contains(err.Error(), "signing key") {
			t.Fatalf("TightenState with %s symlinked = %v, want an error naming the key", key, err)
		}
		os.Remove(key)
	}
}
