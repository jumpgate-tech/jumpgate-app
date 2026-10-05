package executor

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/valve-tech/jumpgate/internal/fsperm"
)

// systemKnownHosts is OpenSSH's system-wide known_hosts on this OS; a var so
// tests can point it elsewhere.
var systemKnownHosts = defaultSystemKnownHosts()

// openTrustedKnownHosts opens a known_hosts file only if it passes the
// trusted-writable check, as one operation. A var so tests can inject the
// verdict.
var openTrustedKnownHosts = fsperm.OpenTrustedWritable

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
//
// The host-key code reads files by name, and a name checked now can be
// swapped before it is read. So the trusted file is read once, from the very
// file that was checked, and what it held is returned as a private snapshot
// that only this user can change.
func OpenSSHKnownHosts(home string) []string {
	files := []string{filepath.Join(home, ".ssh", "known_hosts")}
	if _, err := os.Stat(systemKnownHosts); err != nil {
		return files
	}
	snap, err := trustedSnapshot(systemKnownHosts)
	if err != nil {
		log.Printf("jumpgate: ignoring %s: %v", systemKnownHosts, err)
		return files
	}
	return append(files, snap)
}

var (
	snapshotOnce sync.Once
	snapshotDir  string
	snapshotErr  error
)

// trustedSnapshot returns a private copy of path's content, taken from the
// file the trust check passed. Equal content shares one snapshot file.
func trustedSnapshot(path string) (string, error) {
	f, err := openTrustedKnownHosts(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, 16<<20))
	if err != nil {
		return "", err
	}
	snapshotOnce.Do(func() {
		snapshotDir, snapshotErr = os.MkdirTemp("", "jumpgate-known-hosts-")
		if snapshotErr == nil {
			snapshotErr = fsperm.MakePrivate(snapshotDir)
		}
	})
	if snapshotErr != nil {
		return "", snapshotErr
	}
	sum := sha256.Sum256(data)
	snap := filepath.Join(snapshotDir, hex.EncodeToString(sum[:8])+".known_hosts")
	if _, err := os.Stat(snap); err != nil {
		if err := fsperm.WriteFilePrivate(snap, data); err != nil {
			return "", err
		}
	}
	return snap, nil
}
