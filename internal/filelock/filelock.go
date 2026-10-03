// Package filelock takes advisory locks on files, so two processes (the web
// app and the TUI's server, or two servers) never interleave a
// read-modify-write of the same state.
package filelock

import (
	"errors"
	"os"
)

// ErrLocked reports that TryLock found the lock held.
var ErrLocked = errors.New("filelock: already locked")

// Handle is a held lock. The lock lives as long as the open file.
type Handle struct{ f *os.File }

// Lock blocks until it holds the lock at path, creating the file 0600.
func Lock(path string, exclusive bool) (*Handle, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	if err := lockFile(f, exclusive, true); err != nil {
		f.Close()
		return nil, err
	}
	return &Handle{f: f}, nil
}

// TryLock takes an exclusive lock without waiting, or returns ErrLocked.
func TryLock(path string) (*Handle, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	if err := lockFile(f, true, false); err != nil {
		f.Close()
		return nil, err
	}
	return &Handle{f: f}, nil
}

// File exposes the locked file, for a holder that writes its pid into it.
func (h *Handle) File() *os.File { return h.f }

// Unlock releases the lock.
func (h *Handle) Unlock() error {
	if err := unlockFile(h.f); err != nil {
		h.f.Close()
		return err
	}
	return h.f.Close()
}
