package executor

import (
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/valve-tech/jumpgate/internal/fsperm"
)

// systemKnownHosts is OpenSSH's system-wide known_hosts on this OS; a var so
// tests can point it elsewhere.
var systemKnownHosts = defaultSystemKnownHosts()

// knownHostsTrusted decides whether a known_hosts file may vouch for a host.
// A var so tests can inject the verdict.
var knownHostsTrusted = fsperm.CheckTrustedWritable

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
// system-wide one when it exists and only trusted accounts can write it or
// its directory (M-10). Anyone who could write the system file could vouch
// for a host key, so an untrusted one is skipped with a warning rather than
// failing the dial: the confirmed store still applies.
func OpenSSHKnownHosts(home string) []string {
	files := []string{filepath.Join(home, ".ssh", "known_hosts")}
	if _, err := os.Stat(systemKnownHosts); err == nil {
		if err := knownHostsTrusted(systemKnownHosts); err != nil {
			log.Printf("jumpgate: ignoring %s: %v", systemKnownHosts, err)
		} else {
			files = append(files, systemKnownHosts)
		}
	}
	return files
}
