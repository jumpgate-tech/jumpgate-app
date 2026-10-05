//go:build unix

package fsperm_test

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/valve-tech/jumpgate/internal/fsperm"
)

// P28: a symlink to a directory someone else owns is refused, not tightened
// (and not used).
func TestMkdirPrivateRefusesASymlinkToAnotherUsersDir(t *testing.T) {
	other := "/"
	if fi, err := os.Stat(other); err != nil {
		t.Fatal(err)
	} else if st, ok := fi.Sys().(*syscall.Stat_t); !ok || int(st.Uid) == os.Geteuid() {
		t.Skip("running as the owner of /; no directory owned by another user to point at")
	}
	link := filepath.Join(t.TempDir(), ".jumpgate")
	if err := os.Symlink(other, link); err != nil {
		t.Fatal(err)
	}
	err := fsperm.MkdirPrivate(link)
	if err == nil || !strings.Contains(err.Error(), "another user owns") {
		t.Fatalf("MkdirPrivate(symlink to /) = %v, want a refusal naming another user", err)
	}
}

// symlinkInto makes base/shared (with the given mode) holding the user's own
// base/shared/real, and a link base/.jumpgate to it.
func symlinkInto(t *testing.T, sharedMode os.FileMode) (link, real string) {
	t.Helper()
	base := t.TempDir()
	shared := filepath.Join(base, "shared")
	if err := os.Mkdir(shared, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(shared, sharedMode); err != nil { // Chmod: the umask does not apply
		t.Fatal(err)
	}
	real = filepath.Join(shared, "real")
	if err := os.Mkdir(real, 0o755); err != nil {
		t.Fatal(err)
	}
	link = filepath.Join(base, ".jumpgate")
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	return link, real
}

// Another user could rename an owned directory out of a world-writable
// parent and put their own in its place, so it is refused.
func TestMkdirPrivateRefusesASymlinkIntoAWorldWritableParent(t *testing.T) {
	link, _ := symlinkInto(t, 0o777)
	err := fsperm.MkdirPrivate(link)
	if err == nil || !strings.Contains(err.Error(), "writable by other users") {
		t.Fatalf("MkdirPrivate(link into a 0777 parent) = %v, want a refusal", err)
	}
}

// With the sticky bit (as on /tmp) only an entry's owner may move it.
func TestMkdirPrivateAllowsASymlinkIntoAStickyParent(t *testing.T) {
	link, real := symlinkInto(t, 0o777|os.ModeSticky)
	if err := fsperm.MkdirPrivate(link); err != nil {
		t.Fatalf("MkdirPrivate(link into a 1777 parent): %v", err)
	}
	if err := fsperm.CheckPrivate(real); err != nil {
		t.Fatalf("the real directory was not tightened: %v", err)
	}
}
