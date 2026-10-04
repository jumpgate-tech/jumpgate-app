package daemon

import (
	"fmt"
	"os"
	"testing"
)

// TestMain points HOME at a throwaway directory for the whole package, so no
// test can read the developer's ~/.jumpgate or write into it (config lock,
// run directory, keys), even one that forgets its own t.Setenv("HOME", ...).
func TestMain(m *testing.M) {
	home, err := os.MkdirTemp("", "jumpgate-test-home-")
	if err != nil {
		fmt.Fprintln(os.Stderr, "test home:", err)
		os.Exit(1)
	}
	os.Setenv("HOME", home)
	os.Setenv("USERPROFILE", home) // os.UserHomeDir on Windows
	code := m.Run()
	os.RemoveAll(home)
	os.Exit(code)
}
