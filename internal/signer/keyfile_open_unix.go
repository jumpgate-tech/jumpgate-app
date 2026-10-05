//go:build unix

package signer

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

// openKeyFile opens path read-only. O_NOFOLLOW refuses a symlink at the final
// path element, and O_NONBLOCK keeps opening a FIFO from hanging until the
// regular-file check rejects it.
func openKeyFile(path string) (*os.File, error) {
	f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if errors.Is(err, syscall.ELOOP) {
		return nil, fmt.Errorf("signer: %s is a link, not a key file", path)
	}
	if err != nil {
		return nil, fmt.Errorf("signer: %w", err)
	}
	return f, nil
}
