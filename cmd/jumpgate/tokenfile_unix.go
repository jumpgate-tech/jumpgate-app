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
	return readCheckedFile(path, fix, func(perm, uid, _ uint32) error {
		if perm&0o077 != 0 {
			return fmt.Errorf("%s is readable or writable by others (mode %04o): %s", path, perm, fix)
		}
		if euid := os.Geteuid(); euid != 0 && int(uid) != euid {
			return fmt.Errorf("%s is owned by uid %d, not you (uid %d): %s", path, uid, euid, fix)
		}
		return nil
	})
}

// readSystemdCredential reads a file systemd handed in under
// $CREDENTIALS_DIRECTORY, with the same symlink and regular-file checks as
// readPrivateFile and the ownership checkSystemdCredential accepts.
func readSystemdCredential(path string) ([]byte, error) {
	const fix = "it should be the file systemd passed in with LoadCredentialEncrypted="
	return readCheckedFile(path, fix, func(perm, uid, gid uint32) error {
		if err := checkSystemdCredential(perm, uid, gid, os.Geteuid()); err != nil {
			return fmt.Errorf("%s %v: %s", path, err, fix)
		}
		return nil
	})
}

// checkSystemdCredential accepts the two forms systemd hands a credential
// over in: up to systemd 254, a file of the service user's own with no group
// or other bits; from 255, a root:root file readable by owner and group
// only (the service user reads it through an ACL, whose mask the group bits
// show). Only the owner may write, and others may never read.
func checkSystemdCredential(perm, uid, gid uint32, euid int) error {
	if perm&0o022 != 0 {
		return fmt.Errorf("is writable by group or others (mode %04o)", perm)
	}
	if perm&0o007 != 0 {
		return fmt.Errorf("is readable by others (mode %04o)", perm)
	}
	switch {
	case uid == 0 && gid == 0:
		return nil
	case perm&0o070 != 0:
		return fmt.Errorf("is readable by group %d (mode %04o)", gid, perm)
	case euid == 0 || int(uid) == euid:
		return nil
	}
	return fmt.Errorf("is owned by uid %d, not this service (uid %d)", uid, euid)
}

// readCheckedFile opens path with O_NOFOLLOW, requires a regular file, and
// runs check on the OPENED handle's mode and owner before reading it.
func readCheckedFile(path, fix string, check func(perm, uid, gid uint32) error) ([]byte, error) {
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
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return nil, fmt.Errorf("%s: no owner information", path)
	}
	if err := check(uint32(fi.Mode().Perm()), st.Uid, st.Gid); err != nil {
		return nil, err
	}
	// The token is one short line; bound the read anyway.
	return io.ReadAll(io.LimitReader(f, 64<<10))
}
