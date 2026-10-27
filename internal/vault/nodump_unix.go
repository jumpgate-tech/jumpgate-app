//go:build unix && !linux

package vault

import (
	"fmt"

	"golang.org/x/sys/unix"
)

// DisableCoreDumps sets RLIMIT_CORE to zero, soft and hard, so a crash
// leaves no core file holding the data key and the limit cannot be raised
// again by this process. It then applies denyAttach (darwin: PT_DENY_ATTACH).
//
// There is no mlock: the data key and entry values can still be paged out
// to swap. Encrypt swap if that matters on this machine.
func DisableCoreDumps() error {
	if err := unix.Setrlimit(unix.RLIMIT_CORE, &unix.Rlimit{Cur: 0, Max: 0}); err != nil {
		return fmt.Errorf("vault: disable core dumps: %w", err)
	}
	if err := denyAttach(); err != nil {
		return fmt.Errorf("vault: deny debugger attach: %w", err)
	}
	return nil
}
