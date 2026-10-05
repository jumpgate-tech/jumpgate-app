// Package fsperm makes files and directories readable and writable only by
// the user who owns them, on every OS the controller runs on. On unix that is
// 0600/0700. On Windows a mode does nothing (os.Chmod only flips the
// read-only bit), so the same promise is kept with a protected DACL that
// grants the current user and SYSTEM, and nobody else.
//
// Everything the controller keeps secret goes through here: the controller
// key file, config.json (provider API keys, VPN keys), server.json and the
// server socket (the session token), the transport SSH key and the confirmed
// host keys.
package fsperm

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// ErrNotPrivate means someone other than the owner can read or change a file.
var ErrNotPrivate = errors.New("fsperm: other users can read or change this")

// MkdirPrivate creates dir and any missing parents, then restricts dir itself
// to its owner. An existing dir is tightened too: an older release or a loose
// umask may have left ~/.jumpgate open, and existing is not the same as safe.
// Parents are created with the default mode and left alone.
func MkdirPrivate(dir string) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	return MakePrivate(dir)
}

// MakePrivate restricts an existing file, directory or socket to its owner.
// It never follows a symlink.
func MakePrivate(path string) error { return makePrivate(path) }

// CheckPrivate returns an error wrapping ErrNotPrivate, naming who has
// access, when anyone but the owner can read or change path.
func CheckPrivate(path string) error { return checkPrivate(path) }

// CheckPrivateFile is CheckPrivate on an open file, so the file checked is
// the file read.
func CheckPrivateFile(f *os.File) error { return checkPrivateFile(f) }

// WriteFilePrivate replaces path with data as an owner-only file. The data
// goes to a restricted temp file in the same directory, is synced, then
// renamed over path, so a reader sees the old file or the new one, never a
// half-written or briefly readable one.
func WriteFilePrivate(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	ok := false
	defer func() {
		if !ok {
			tmp.Close()
			os.Remove(name)
		}
	}()
	// Restrict before writing: the temp file is empty until this succeeds.
	if err := MakePrivate(name); err != nil {
		return fmt.Errorf("fsperm: restrict %s: %w", name, err)
	}
	if _, err := tmp.Write(data); err != nil {
		return err
	}
	if err := tmp.Sync(); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := Rename(name, path); err != nil {
		return err
	}
	ok = true
	return nil
}

// Rename is os.Rename. On Windows it is retried for up to about 500 ms while
// another process (an indexer, antivirus, a concurrent reader) holds the
// target open, which makes MoveFileEx fail with a sharing violation.
func Rename(oldpath, newpath string) error { return rename(oldpath, newpath) }
