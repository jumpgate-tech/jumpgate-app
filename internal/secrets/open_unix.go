//go:build unix

package secrets

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

// openNoFollow opens path with flag, refusing a symlink at the final path
// element (O_NOFOLLOW) and never blocking on a FIFO.
func openNoFollow(path string, flag int) (*os.File, error) {
	f, err := os.OpenFile(path, flag|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if errors.Is(err, syscall.ELOOP) {
		return nil, fmt.Errorf("%s is a link, not a secret file", path)
	}
	return f, err
}
