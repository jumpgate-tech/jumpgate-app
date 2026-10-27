//go:build !linux

package vault

// DisableCoreDumps does nothing outside Linux.
func DisableCoreDumps() error { return nil }
