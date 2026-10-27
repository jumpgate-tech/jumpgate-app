//go:build unix

package vault

import (
	"fmt"
	"os"
	"syscall"
)

// keepOwner says whose a rewritten vault must stay: when root rewrites a
// file, it hands the new one to the old one's owner (S10). A user's own
// write needs no chown.
func keepOwner(fi os.FileInfo, euid int) (uid, gid int, chown bool) {
	if fi == nil || euid != 0 {
		return 0, 0, false
	}
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, 0, false
	}
	return int(st.Uid), int(st.Gid), true
}

// keepFileOwner chowns the temp file f to the previous vault's owner before
// it is renamed into place.
func keepFileOwner(f *os.File, prev os.FileInfo) error {
	uid, gid, ok := keepOwner(prev, os.Geteuid())
	if !ok {
		return nil
	}
	if err := f.Chown(uid, gid); err != nil {
		return fmt.Errorf("vault: keep the owner of the vault: %w", err)
	}
	return nil
}
