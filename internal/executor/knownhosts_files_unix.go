//go:build !windows

package executor

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

// checkKnownHostsFile accepts path only when it and its directory belong to
// root or this user and nobody else can write them.
func checkKnownHostsFile(path string) error {
	for _, p := range []string{path, filepath.Dir(path)} {
		fi, err := os.Stat(p)
		if err != nil {
			return err
		}
		st, ok := fi.Sys().(*syscall.Stat_t)
		if !ok {
			return fmt.Errorf("cannot read the owner of %s", p)
		}
		if err := trustedUnixFile(p, fi.Mode(), st.Uid, uint32(os.Geteuid())); err != nil {
			return err
		}
	}
	return nil
}

// trustedUnixFile is the decision itself: owner root or euid, and no group or
// world write bit.
func trustedUnixFile(p string, mode os.FileMode, uid, euid uint32) error {
	if uid != 0 && uid != euid {
		return fmt.Errorf("%s is owned by uid %d, not root or you", p, uid)
	}
	if mode.Perm()&0o022 != 0 {
		return fmt.Errorf("%s is writable by others (%v)", p, mode.Perm())
	}
	return nil
}
