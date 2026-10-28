//go:build windows

package daemon

// fileOwner: the jumpgate service is Linux only, so on Windows no
// system.json is trusted.
func fileOwner(string) (int, bool) { return -1, false }
