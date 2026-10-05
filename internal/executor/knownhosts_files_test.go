package executor

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// M-10: the system-wide known_hosts counts as confirmed too, when present.
func TestOpenSSHKnownHostsIncludesTheSystemFileWhenPresent(t *testing.T) {
	home := t.TempDir()
	sys := filepath.Join(t.TempDir(), "ssh_known_hosts")
	old := systemKnownHosts
	systemKnownHosts = sys
	t.Cleanup(func() { systemKnownHosts = old })

	user := filepath.Join(home, ".ssh", "known_hosts")
	if got := OpenSSHKnownHosts(home); !slices.Equal(got, []string{user}) {
		t.Fatalf("without a system file: %v", got)
	}
	if err := os.WriteFile(sys, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if got := OpenSSHKnownHosts(home); !slices.Equal(got, []string{user, sys}) {
		t.Fatalf("with a system file: %v", got)
	}
}
