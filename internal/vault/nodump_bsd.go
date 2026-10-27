//go:build unix && !linux && !darwin

package vault

// denyAttach does nothing here: on the BSDs and other unix, another process
// of the same user may still ptrace this one (FreeBSD's procctl
// PROC_TRACE_CTL is not used). Only the core limit applies.
func denyAttach() error { return nil }
