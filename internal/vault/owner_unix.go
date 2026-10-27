//go:build unix

package vault

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

// writeOwnerProblem refuses a write by root to a vault another user owns:
// root never creates files in that user's directory (a planted link there
// could aim them anywhere). It is told to run as the owner instead.
func writeOwnerProblem(path string, uid uint32, euid int) error {
	if euid == 0 && uid != 0 {
		return fmt.Errorf("%w: %s belongs to uid %d; run this as that user", ErrUnsafeFile, path, uid)
	}
	return nil
}

func checkWriteOwner(path string, fi os.FileInfo) error {
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return fmt.Errorf("%w: no owner information for %s", ErrUnsafeFile, path)
	}
	return writeOwnerProblem(path, st.Uid, os.Geteuid())
}

// lockFile takes an exclusive flock on path, opened without following a
// link and checked to be a regular file of this user.
func lockFile(path string) (func(), error) {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0o600)
	if errors.Is(err, syscall.ELOOP) {
		return nil, fmt.Errorf("%w: %s is a link", ErrUnsafeFile, path)
	}
	if err != nil {
		return nil, fmt.Errorf("vault: lock: %w", err)
	}
	fi, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("vault: lock: %w", err)
	}
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !fi.Mode().IsRegular() || !ok || int(st.Uid) != os.Geteuid() {
		f.Close()
		return nil, fmt.Errorf("%w: %s is not a regular file of yours", ErrUnsafeFile, path)
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		f.Close()
		return nil, fmt.Errorf("vault: lock: %w", err)
	}
	return func() { syscall.Flock(int(f.Fd()), syscall.LOCK_UN); f.Close() }, nil
}
