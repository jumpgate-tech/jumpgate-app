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
	"io/fs"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ErrNotPrivate means someone other than the owner can read or change a file.
var ErrNotPrivate = errors.New("fsperm: other users can read or change this")

// MkdirPrivate creates dir and any missing parents, then restricts dir itself
// to its owner. An existing dir is tightened too: an older release or a loose
// umask may have left ~/.jumpgate open, and existing is not the same as safe.
// Parents are created with the default mode and left alone.
//
// dir may be a symlink to a directory (a ~/.jumpgate kept on another disk,
// say). The real directory is then tightened, but only if the current user
// owns it; one that belongs to someone else is refused, since its owner could
// read everything jumpgate puts there. Secret files themselves are never
// followed through a link (MakePrivate refuses one).
func MkdirPrivate(dir string) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	fi, err := os.Lstat(dir)
	if err != nil {
		return err
	}
	if fi.Mode()&os.ModeSymlink == 0 {
		return MakePrivate(dir)
	}
	real, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return err
	}
	mine, err := ownedByCurrentUser(real)
	if err != nil {
		return fmt.Errorf("fsperm: find the owner of %s: %w", real, err)
	}
	if !mine {
		return fmt.Errorf("fsperm: %s is a symlink to %s, which another user owns; jumpgate keeps secrets only in a directory you own", dir, real)
	}
	// Owning the directory is not enough if another user can move it: they
	// could rename it away and put one of their own at the same path.
	if err := checkAncestors(real); err != nil {
		return fmt.Errorf("fsperm: %s is a symlink to %s: %w", dir, real, err)
	}
	return MakePrivate(real)
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

// CreatePrivate creates path as a new owner-only file, open for reading and
// writing. It fails with an error matching fs.ErrExist when anything, a link
// included, is already at path. The file is private from the moment it
// exists: on Windows it is created with the owner-only descriptor instead of
// first inheriting its directory's DACL, so no other user can open a handle
// to it before it is restricted and keep that handle once a secret is in it.
func CreatePrivate(path string) (*os.File, error) { return createPrivate(path, false) }

// CreateTempPrivate is os.CreateTemp for an owner-only file: a new file in
// dir (os.TempDir() when empty) whose name is pattern with its last "*"
// replaced by a random string, created the way CreatePrivate creates one.
func CreateTempPrivate(dir, pattern string) (*os.File, error) {
	if dir == "" {
		dir = os.TempDir()
	}
	prefix, suffix := pattern, ""
	if i := strings.LastIndex(pattern, "*"); i >= 0 {
		prefix, suffix = pattern[:i], pattern[i+1:]
	}
	for try := 0; try < 10000; try++ {
		f, err := createPrivate(filepath.Join(dir, prefix+strconv.FormatUint(uint64(rand.Uint32()), 10)+suffix), false)
		if errors.Is(err, fs.ErrExist) {
			continue
		}
		return f, err
	}
	return nil, &os.PathError{Op: "createtemp", Path: filepath.Join(dir, pattern), Err: fs.ErrExist}
}

// OpenAppendPrivate opens path for appending, creating it as CreatePrivate
// does when it is missing. An existing file is restricted to its owner before
// it is returned. A link at path is refused.
func OpenAppendPrivate(path string) (*os.File, error) {
	f, err := createPrivate(path, true)
	if !errors.Is(err, fs.ErrExist) {
		return f, err
	}
	return openAppendExisting(path)
}

// WriteFilePrivate replaces path with data as an owner-only file. The data
// goes to a temp file in the same directory that is private from creation
// (CreateTempPrivate), is synced, then renamed over path, so a reader sees the
// old file or the new one, never a half-written or briefly readable one.
func WriteFilePrivate(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := CreateTempPrivate(dir, "."+filepath.Base(path)+".tmp-*")
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

// OpenTrustedWritable opens path for reading, provided the file, which other
// people may read (a system-wide known_hosts, say), can be changed or replaced
// only by this user, administrators and the system: it and its directory are
// owned and writable only by them, and nobody untrusted can move either
// through a directory above. Unlike CheckPrivate it does not mind who can read
// path.
//
// Checking a path and then opening it by name would let the path be swapped in
// between, so the check and the open are one operation: a symlink is resolved
// once, the real file is opened without following a link or reparse point, and
// the permissions are read from that open file. Read from the returned file,
// never from path again. Its Name is the resolved path that was checked (on
// Windows, where the open handle really is, without the \\?\ prefix for a
// drive path), so a caller that must pass a path on can pass that one.
func OpenTrustedWritable(path string) (*os.File, error) { return openTrustedWritable(path) }

// CheckTrustedWritable is OpenTrustedWritable for a caller that needs only the
// verdict (a test, or a file it will not read).
func CheckTrustedWritable(path string) error {
	f, err := openTrustedWritable(path)
	if err != nil {
		return err
	}
	return f.Close()
}

// trustedCheckHook runs after the checks and before the file is returned. A
// seam for tests, which swap the path here to show the file read is the one
// checked.
var trustedCheckHook func()
