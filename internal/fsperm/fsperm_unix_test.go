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
