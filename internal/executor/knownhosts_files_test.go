package executor

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/valve-tech/jumpgate/internal/fsperm"
)

// M-10: the system-wide known_hosts counts as confirmed too, when present: its
// content is offered through a private snapshot of the checked file.
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
	if err := os.WriteFile(sys, []byte("host ssh-ed25519 AAAA\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := OpenSSHKnownHosts(home)
	if len(got) != 2 || got[0] != user {
		t.Fatalf("with a system file: %v", got)
	}
	if b, err := os.ReadFile(got[1]); err != nil || string(b) != "host ssh-ed25519 AAAA\n" {
		t.Fatalf("snapshot = %q, %v", b, err)
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
	oldSys, oldOpen := systemKnownHosts, openTrustedKnownHosts
	systemKnownHosts = sys
	t.Cleanup(func() { systemKnownHosts, openTrustedKnownHosts = oldSys, oldOpen })

	openTrustedKnownHosts = func(string) (*os.File, error) { return nil, errors.New("writable by Everyone") }
	user := filepath.Join(home, ".ssh", "known_hosts")
	if got := OpenSSHKnownHosts(home); !slices.Equal(got, []string{user}) {
		t.Fatalf("untrusted file kept: %v", got)
	}
}

// What is trusted is what was checked: if the path is swapped for another file
// after the check, the snapshot still holds the checked file's content, and
// changing the original later does not change the snapshot.
func TestOpenSSHKnownHostsSnapshotsTheCheckedFile(t *testing.T) {
	home := t.TempDir()
	dir := t.TempDir()
	sys := filepath.Join(dir, "ssh_known_hosts")
	if err := os.WriteFile(sys, []byte("checked\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	oldSys, oldOpen := systemKnownHosts, openTrustedKnownHosts
	systemKnownHosts = sys
	t.Cleanup(func() { systemKnownHosts, openTrustedKnownHosts = oldSys, oldOpen })
	openTrustedKnownHosts = func(p string) (*os.File, error) {
		f, err := fsperm.OpenTrustedWritable(p) // the real check, which opens the file
		if err != nil {
			return nil, err
		}
		// An attacker swaps the path after the check.
		other := filepath.Join(dir, "evil")
		if err := os.WriteFile(other, []byte("evil\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		_ = os.Remove(p)
		if err := os.Rename(other, p); err != nil {
			f.Close()
			t.Skipf("cannot swap a file in place here: %v", err) // Windows refuses while the checked file is open, which is the point
		}
		return f, nil
	}
	got := OpenSSHKnownHosts(home)
	if len(got) != 2 {
		t.Fatalf("got %v", got)
	}
	if b, _ := os.ReadFile(got[1]); string(b) != "checked\n" {
		t.Fatalf("snapshot = %q, want the checked file's content", b)
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
	if err := fsperm.CheckTrustedWritable(f); err != nil {
		t.Fatalf("owner-writable file refused: %v", err)
	}
	makeWorldWritable(t, f)
	if err := fsperm.CheckTrustedWritable(f); err == nil {
		t.Fatal("a world-writable file was trusted")
	}
}
