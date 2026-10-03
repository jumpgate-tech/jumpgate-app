//go:build unix

package signer

import "syscall"

// openNoFollow refuses a symlink at the final path element, and O_NONBLOCK
// keeps opening a FIFO from hanging until the regular-file check rejects it.
const openNoFollow = syscall.O_NOFOLLOW | syscall.O_NONBLOCK
