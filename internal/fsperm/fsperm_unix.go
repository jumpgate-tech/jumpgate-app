//go:build unix

package fsperm

import (
	"fmt"
	"os"
)

func makePrivate(path string) error {
	fi, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("fsperm: %s is a symlink; refusing to change what it points to", path)
	}
	mode := os.FileMode(0o600)
	if fi.IsDir() {
		mode = 0o700
	}
	return os.Chmod(path, mode)
}

func checkPrivate(path string) error {
	fi, err := os.Stat(path)
	if err != nil {
		return err
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
