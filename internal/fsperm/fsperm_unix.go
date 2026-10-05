//go:build unix

package fsperm

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

// makePrivate opens path without following a link and changes the mode of
// the open descriptor, so the object restricted is the object that was
// checked: a link swapped in after the check makes the open fail instead.
func makePrivate(path string) error {
	fi, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("fsperm: %s is a symlink; refusing to change what it points to", path)
	}
	if fi.Mode()&os.ModeSocket != 0 {
		// A socket cannot be opened, so it is changed by name. The only
		// socket restricted is the server's, in the owner-only run dir, where
		// nobody else can swap it for a link.
		return os.Chmod(path, 0o600)
	}
	// O_NONBLOCK: opening a FIFO must not hang.
	f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return err
	}
	defer f.Close()
	ofi, err := f.Stat()
	if err != nil {
		return err
	}
	return f.Chmod(privateMode(ofi.Mode()))
}

// checkPrivate looks at path itself, as Windows does: a symlink is not a
// private file whatever its target's mode.
func checkPrivate(path string) error {
	fi, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%w: %s is a symlink; jumpgate keeps secrets only in regular files and directories", ErrNotPrivate, path)
	}
	return checkInfo(path, fi)
}

func checkPrivateFile(f *os.File) error {
	fi, err := f.Stat()
	if err != nil {
		return err
	}
	return checkInfo(f.Name(), fi)
}

func checkInfo(path string, fi os.FileInfo) error {
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return fmt.Errorf("fsperm: no owner information for %s", path)
	}
	return checkOwnerMode(path, fi.Mode(), st.Uid, os.Geteuid())
}

// checkOwnerMode is the decision for a secret on unix (unix's half of ruling
// P31): it must belong to this euid, unless that is root, and nobody else
// may have any permission bit.
func checkOwnerMode(path string, m os.FileMode, uid uint32, euid int) error {
	if euid != 0 && int(uid) != euid {
		return fmt.Errorf("%w: %s belongs to uid %d, not you (uid %d); delete it and let jumpgate recreate it", ErrNotPrivate, path, uid, euid)
	}
	if m.Perm()&0o077 != 0 {
		return fmt.Errorf("%w: %s is mode %o; run chmod %o %s", ErrNotPrivate, path, m.Perm(), privateMode(m), path)
	}
	return nil
}

func privateMode(m os.FileMode) os.FileMode {
	if m.IsDir() {
		return 0o700
	}
	return 0o600
}

func createPrivate(path string, appendOnly bool) (*os.File, error) {
	flag := os.O_RDWR
	if appendOnly {
		flag = os.O_WRONLY | os.O_APPEND
	}
	// O_EXCL also refuses a link at path. The mode is set again on the
	// descriptor because a umask may have taken owner bits away.
	f, err := os.OpenFile(path, flag|os.O_CREATE|os.O_EXCL|syscall.O_NOFOLLOW, 0o600)
	if err != nil {
		return nil, err
	}
	if err := f.Chmod(0o600); err != nil {
		f.Close()
		return nil, err
	}
	return f, nil
}

func openAppendExisting(path string) (*os.File, error) {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}
	fi, err := f.Stat()
	if err == nil && !fi.Mode().IsRegular() {
		err = fmt.Errorf("fsperm: %s is not a regular file", path)
	}
	if err == nil {
		err = f.Chmod(0o600)
	}
	if err != nil {
		f.Close()
		return nil, err
	}
	return f, nil
}

func ownedByCurrentUser(path string) (bool, error) {
	fi, err := os.Lstat(path)
	if err != nil {
		return false, err
	}
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return false, fmt.Errorf("no owner information for %s", path)
	}
	return int(st.Uid) == os.Geteuid(), nil
}

// checkAncestors refuses a path one of whose parents another user could
// rename entries in: a parent writable by group or others without the sticky
// bit (/tmp, mode 1777, is fine: there only an entry's owner may move it), or
// a parent owned by someone other than this user or root, who could make it
// so. path itself is not checked; the caller makes it private.
func checkAncestors(path string) error {
	for p := filepath.Dir(path); ; p = filepath.Dir(p) {
		fi, err := os.Lstat(p)
		if err != nil {
			return err
		}
		st, ok := fi.Sys().(*syscall.Stat_t)
		if !ok {
			return fmt.Errorf("no owner information for %s", p)
		}
		if st.Uid != 0 && int(st.Uid) != os.Geteuid() {
			return fmt.Errorf("its parent %s belongs to another user, who could move it and put their own in its place", p)
		}
		if fi.Mode().Perm()&0o022 != 0 && fi.Mode()&os.ModeSticky == 0 {
			return fmt.Errorf("its parent %s is writable by other users (mode %o, no sticky bit), who could move it and put their own in its place", p, fi.Mode().Perm())
		}
		if filepath.Dir(p) == p {
			return nil
		}
	}
}

func rename(oldpath, newpath string) error { return os.Rename(oldpath, newpath) }
