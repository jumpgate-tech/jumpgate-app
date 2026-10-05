//go:build windows

package fsperm

import (
	"net"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/windows"
)

// M-5: an indexer or antivirus holding the target open makes MoveFileEx fail
// with a sharing violation for a moment. Rename must wait it out.
func TestRenameRetriesASharingViolation(t *testing.T) {
	dir := t.TempDir()
	a, b := filepath.Join(dir, "a"), filepath.Join(dir, "b")
	if err := os.WriteFile(a, []byte("a"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b, []byte("b"), 0o600); err != nil {
		t.Fatal(err)
	}
	calls := 0
	old := moveFile
	moveFile = func(from, to string) error {
		calls++
		if calls < 3 {
			return &os.LinkError{Op: "rename", Old: from, New: to, Err: windows.ERROR_SHARING_VIOLATION}
		}
		return old(from, to)
	}
	t.Cleanup(func() { moveFile = old })
	if err := Rename(a, b); err != nil {
		t.Fatalf("Rename = %v after %d calls", err, calls)
	}
	if calls != 3 {
		t.Fatalf("moveFile called %d times, want 3", calls)
	}
	if got, err := os.ReadFile(b); err != nil || string(got) != "a" {
		t.Fatalf("b = %q, %v; want a", got, err)
	}
}

// The server socket is an AF_UNIX reparse point; MakePrivate must set its
// DACL without following it (D2).
func TestMakePrivateOnAUnixSocket(t *testing.T) {
	dir, err := os.MkdirTemp("", "jg")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	sock := filepath.Join(dir, "s.sock")
	ln, err := netListenUnix(sock)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	if err := MakePrivate(sock); err != nil {
		t.Fatalf("MakePrivate(socket): %v", err)
	}
	if err := CheckPrivate(sock); err != nil {
		t.Fatalf("CheckPrivate(socket): %v", err)
	}
}

// netListenUnix is kept in the test so production code has no net import.
func netListenUnix(path string) (interface{ Close() error }, error) {
	return net.Listen("unix", path)
}
