//go:build linux

package main

import (
	"fmt"

	"golang.org/x/sys/unix"
)

// disableDumpable keeps the service from core dumps and from ptrace by
// same-uid processes: its memory holds the unlocked vault key.
func disableDumpable() error {
	if err := unix.Prctl(unix.PR_SET_DUMPABLE, 0, 0, 0, 0); err != nil {
		return fmt.Errorf("prctl(PR_SET_DUMPABLE): %w", err)
	}
	return nil
}
