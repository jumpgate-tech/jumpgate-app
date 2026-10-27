//go:build unix

package vault

import (
	"io/fs"
	"syscall"
	"testing"
	"time"
)

type fakeInfo struct{ st *syscall.Stat_t }

func (fakeInfo) Name() string       { return "vault" }
func (fakeInfo) Size() int64        { return 0 }
func (fakeInfo) Mode() fs.FileMode  { return 0o600 }
func (fakeInfo) ModTime() time.Time { return time.Time{} }
func (fakeInfo) IsDir() bool        { return false }
func (f fakeInfo) Sys() any         { return f.st }

func TestRootWriteKeepsTheOwner(t *testing.T) {
	uid, gid, chown := keepOwner(fakeInfo{&syscall.Stat_t{Uid: 990, Gid: 990}}, 0)
	if uid != 990 || gid != 990 || !chown {
		t.Fatalf("root: (%d, %d, %v)", uid, gid, chown)
	}
	if _, _, chown := keepOwner(fakeInfo{&syscall.Stat_t{Uid: 501, Gid: 20}}, 501); chown {
		t.Fatal("a user's own write chowns")
	}
	if _, _, chown := keepOwner(nil, 0); chown {
		t.Fatal("chown with no previous file")
	}
}
