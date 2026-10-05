package executor

import (
	"errors"
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

// An untrusted system file must not vouch for hosts: it is skipped, the user's
// file stays.
func TestOpenSSHKnownHostsSkipsAnUntrustedSystemFile(t *testing.T) {
	home := t.TempDir()
	sys := filepath.Join(t.TempDir(), "ssh_known_hosts")
	if err := os.WriteFile(sys, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	oldSys, oldCheck := systemKnownHosts, knownHostsTrusted
	systemKnownHosts = sys
	t.Cleanup(func() { systemKnownHosts, knownHostsTrusted = oldSys, oldCheck })

	knownHostsTrusted = func(string) error { return errors.New("writable by Everyone") }
	user := filepath.Join(home, ".ssh", "known_hosts")
	if got := OpenSSHKnownHosts(home); !slices.Equal(got, []string{user}) {
		t.Fatalf("untrusted file kept: %v", got)
	}
	knownHostsTrusted = func(string) error { return nil }
	if got := OpenSSHKnownHosts(home); !slices.Equal(got, []string{user, sys}) {
		t.Fatalf("trusted file dropped: %v", got)
	}
}

// The real check: a file this user owns passes while only they can write it,
// and fails once everyone can.
func TestCheckKnownHostsFileRefusesAWorldWritableFile(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "ssh_known_hosts")
	if err := os.WriteFile(f, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := checkKnownHostsFile(f); err != nil {
		t.Fatalf("owner-writable file refused: %v", err)
	}
	makeWorldWritable(t, f)
	if err := checkKnownHostsFile(f); err == nil {
		t.Fatal("a world-writable file was trusted")
	}
}
