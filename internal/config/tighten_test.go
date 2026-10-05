package config

import (
	"os"
	"path/filepath"
	"slices"
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

	tightened, errs := TightenState(outsideKey)
	if len(errs) != 0 {
		t.Fatalf("TightenState errors: %v", errs)
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
	if again, errs := TightenState(outsideKey); len(again) != 0 || len(errs) != 0 {
		t.Fatalf("second run = %v, %v; want nothing", again, errs)
	}
}

// Missing files are skipped, not errors: a fresh install has none of them.
func TestTightenStateSkipsMissingFiles(t *testing.T) {
	home := testutil.Home(t)
	tightened, errs := TightenState(filepath.Join(home, "no", "such.key"))
	if len(tightened) != 0 || len(errs) != 0 {
		t.Fatalf("TightenState on an empty home = %v, %v; want nothing", tightened, errs)
	}
	if err := fsperm.CheckPrivate(filepath.Join(home, ".jumpgate")); err != nil {
		t.Fatalf("the state directory was not created private: %v", err)
	}
}
