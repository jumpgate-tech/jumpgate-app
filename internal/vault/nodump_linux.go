//go:build linux

package vault

import (
	"fmt"

	"golang.org/x/sys/unix"
)

// DisableCoreDumps marks the process not dumpable, so a crash leaves no core
// file holding the data key and no other process of this user can ptrace it.
func DisableCoreDumps() error {
	if err := unix.Prctl(unix.PR_SET_DUMPABLE, 0, 0, 0, 0); err != nil {
		return fmt.Errorf("vault: disable core dumps: %w", err)
	}
	return nil
}
