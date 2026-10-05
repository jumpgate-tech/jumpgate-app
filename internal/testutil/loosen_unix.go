//go:build unix

package testutil

import (
	"os"
	"testing"
)

// Loosen makes path readable by other users, for tests that must see a
// refusal.
func Loosen(t testing.TB, path string) {
	t.Helper()
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	mode := os.FileMode(0o644)
	if fi.IsDir() {
		mode = 0o755
	}
	if err := os.Chmod(path, mode); err != nil {
		t.Fatal(err)
	}
}
