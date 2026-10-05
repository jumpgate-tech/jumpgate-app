//go:build windows

package main

import "os"

// readPrivateFile reads a credential file. Windows has no mode bits to check
// here; the ACL check arrives with the platform-support branch.
func readPrivateFile(path string) ([]byte, error) {
	// TODO(platform T2 merge): fsperm.CheckPrivateFile
	return os.ReadFile(path)
}
