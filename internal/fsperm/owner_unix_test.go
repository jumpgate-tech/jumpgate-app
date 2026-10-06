//go:build unix

package fsperm

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

// P31 on unix: a secret belongs to this euid, or the process is root.
func TestCheckOwnerModeRequiresTheOwner(t *testing.T) {
	cases := []struct {
		uid  uint32
		euid int
		mode os.FileMode
		ok   bool
	}{
		{uid: 1000, euid: 1000, mode: 0o600, ok: true},
		{uid: 1001, euid: 1000, mode: 0o600, ok: false},
		{uid: 0, euid: 1000, mode: 0o600, ok: false},
		{uid: 1000, euid: 0, mode: 0o600, ok: true}, // root checks a user's file
		{uid: 1000, euid: 1000, mode: 0o640, ok: false},
		{uid: 1000, euid: 1000, mode: os.ModeDir | 0o700, ok: true},
	}
	for _, c := range cases {
		err := checkOwnerMode("f", c.mode, c.uid, c.euid)
		if c.ok != (err == nil) || (err != nil && !errors.Is(err, ErrNotPrivate)) {
			t.Errorf("checkOwnerMode(mode %o, uid %d, euid %d) = %v, want ok=%v", c.mode, c.uid, c.euid, err, c.ok)
		}
	}
}

// The real files: as a normal user, a root-owned path is refused for its
// owner; as root (the AS_ROOT=1 Linux leg), a 0600 file chowned to another
// uid is accepted.
func TestCheckPrivateChecksTheRealOwner(t *testing.T) {
	if os.Geteuid() != 0 {
		var st syscall.Stat_t
		if err := syscall.Stat("/", &st); err != nil || st.Uid == uint32(os.Geteuid()) {
			t.Skip("/ is not owned by another user here")
		}
		err := CheckPrivate("/")
		if !errors.Is(err, ErrNotPrivate) || !strings.Contains(err.Error(), "belongs to uid") {
			t.Fatalf("CheckPrivate(/) as uid %d = %v, want a refusal for its owner", os.Geteuid(), err)
		}
		return
	}
	p := filepath.Join(t.TempDir(), "k")
	if err := os.WriteFile(p, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chown(p, 1000, 1000); err != nil {
		t.Fatal(err)
	}
	if err := CheckPrivate(p); err != nil {
		t.Fatalf("CheckPrivate as root on a 0600 file owned by uid 1000 = %v, want nil", err)
	}
}
