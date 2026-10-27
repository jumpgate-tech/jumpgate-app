//go:build darwin

package vault

import "golang.org/x/sys/unix"

// denyAttach applies PT_DENY_ATTACH: from now on no debugger (or other
// process of this user) can ptrace this one, and the process is killed if a
// debugger is already attached. It cannot be undone, so running jumpgate
// under dlv or lldb stops working past the first unlock. Tests stub
// DisableCoreDumps through the callers' seams.
func denyAttach() error { return unix.PtraceDenyAttach() }
