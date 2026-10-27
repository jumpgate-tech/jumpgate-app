//go:build unix

package vault

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

// openNoFollow opens path read-only. O_NOFOLLOW refuses a symlink at the
// final element, and O_NONBLOCK keeps a FIFO from hanging the open until the
// regular-file check rejects it.
func openNoFollow(path string) (*os.File, error) {
	f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if errors.Is(err, syscall.ELOOP) {
		return nil, fmt.Errorf("%w: %s is a link, not a vault file", ErrUnsafeFile, path)
	}
	if err != nil {
		return nil, fmt.Errorf("vault: %w", err)
	}
	return f, nil
}

// checkOwner refuses a vault another user owns. Root may open any owner's
// vault, so `sudo jumpgate vault seal` works on a service's vault (S10).
func checkOwner(path string, fi os.FileInfo) error {
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return fmt.Errorf("%w: no owner information for %s", ErrUnsafeFile, path)
	}
	return ownerProblem(path, st.Uid, os.Geteuid())
}

func ownerProblem(path string, uid uint32, euid int) error {
	if euid != 0 && int(uid) != euid {
		return fmt.Errorf("%w: %s belongs to uid %d, not you (uid %d)", ErrUnsafeFile, path, uid, euid)
	}
	return nil
}
