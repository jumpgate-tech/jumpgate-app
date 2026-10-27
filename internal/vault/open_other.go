//go:build !unix && !windows

package vault

import (
	"fmt"
	"os"
)

func openNoFollow(path string) (*os.File, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("vault: %w", err)
	}
	return f, nil
}

// lockFile has no advisory lock on these systems; it only refuses to be
// pointed elsewhere by an existing link.
func lockFile(path string) (func(), error) {
	if fi, err := os.Lstat(path); err == nil && !fi.Mode().IsRegular() {
		return nil, fmt.Errorf("%w: %s is not a regular file", ErrUnsafeFile, path)
	}
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return nil, fmt.Errorf("vault: lock: %w", err)
	}
	return func() { f.Close() }, nil
}

func checkOwner(string, os.FileInfo) error { return nil }
