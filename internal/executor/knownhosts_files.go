package executor

import (
	"os"
	"path/filepath"
	"runtime"
)

// systemKnownHosts is OpenSSH's system-wide known_hosts on this OS; a var so
// tests can point it elsewhere.
var systemKnownHosts = defaultSystemKnownHosts()

func defaultSystemKnownHosts() string {
	if runtime.GOOS == "windows" {
		pd := os.Getenv("ProgramData")
		if pd == "" {
			pd = `C:\ProgramData`
		}
		return filepath.Join(pd, "ssh", "ssh_known_hosts")
	}
	return "/etc/ssh/ssh_known_hosts"
}

// OpenSSHKnownHosts lists the OpenSSH known_hosts files strict checking
// consults besides jumpgate's confirmed store: the user's, and the
// system-wide one when it exists (M-10).
func OpenSSHKnownHosts(home string) []string {
	files := []string{filepath.Join(home, ".ssh", "known_hosts")}
	if _, err := os.Stat(systemKnownHosts); err == nil {
		files = append(files, systemKnownHosts)
	}
	return files
}
