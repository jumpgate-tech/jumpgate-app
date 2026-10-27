//go:build !unix

package vault

// DisableCoreDumps does nothing on Windows (and other non-unix systems) and
// returns nil, so unlock still works there. This is a known gap, not a
// protection: Windows has no per-process switch that stops a crash dump.
// Windows Error Reporting may write a dump holding the data key and entry
// values (for example when LocalDumps is configured), and any process with
// PROCESS_VM_READ on this one, such as one of the same user, can read its
// memory. SetErrorMode only hides the fault dialog and is not used. There is
// no VirtualLock either, so secrets can also reach the page file.
func DisableCoreDumps() error { return nil }
