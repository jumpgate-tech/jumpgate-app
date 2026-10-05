//go:build unix

package fsperm

import (
	"fmt"
	"os"
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
	return checkMode(path, fi.Mode())
}

func checkPrivateFile(f *os.File) error {
	fi, err := f.Stat()
	if err != nil {
		return err
	}
	return checkMode(f.Name(), fi.Mode())
}

func checkMode(path string, m os.FileMode) error {
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

func rename(oldpath, newpath string) error { return os.Rename(oldpath, newpath) }
