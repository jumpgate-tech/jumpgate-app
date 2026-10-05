//go:build !windows

package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"syscall"
)

// readPrivateFile reads a credential file only if it is private to this user.
// It opens path with O_NOFOLLOW, so a symlink is refused rather than followed,
// then checks the OPENED handle (fstat, not a second lookup by name, which a
// swap could race): it must be a regular file, with no group or other
// permission bits, owned by this user (root may read any owner's file).
func readPrivateFile(path string) ([]byte, error) {
	fix := fmt.Sprintf("make it a regular file owned by you and run `chmod 600 %s`", path)
	fd, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_CLOEXEC|syscall.O_NONBLOCK, 0)
	if err != nil {
		if errors.Is(err, syscall.ELOOP) {
			return nil, fmt.Errorf("%s is a symlink; a token file must be the file itself: %s", path, fix)
		}
		return nil, &os.PathError{Op: "open", Path: path, Err: err}
	}
	f := os.NewFile(uintptr(fd), path)
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !fi.Mode().IsRegular() {
		return nil, fmt.Errorf("%s is not a regular file (%s): %s", path, fi.Mode().Type(), fix)
	}
	if perm := fi.Mode().Perm(); perm&0o077 != 0 {
		return nil, fmt.Errorf("%s is readable or writable by others (mode %04o): %s", path, perm, fix)
	}
	if st, ok := fi.Sys().(*syscall.Stat_t); ok {
		if euid := os.Geteuid(); euid != 0 && int(st.Uid) != euid {
			return nil, fmt.Errorf("%s is owned by uid %d, not you (uid %d): %s", path, st.Uid, euid, fix)
		}
	}
	// The token is one short line; bound the read anyway.
	return io.ReadAll(io.LimitReader(f, 64<<10))
}
