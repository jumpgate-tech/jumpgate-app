//go:build unix

package daemon

import (
	"os"
	"syscall"
)

// fileOwner is the uid that owns path itself (never a link's target).
func fileOwner(path string) (int, bool) {
	fi, err := os.Lstat(path)
	if err != nil || !fi.Mode().IsRegular() {
		return -1, false
	}
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return -1, false
	}
	return int(st.Uid), true
}
