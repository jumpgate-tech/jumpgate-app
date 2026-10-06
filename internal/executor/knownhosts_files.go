package executor

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
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

// OpenSSHKnownHosts lists the OpenSSH known_hosts sources strict checking
// consults besides jumpgate's confirmed store: the user's file, and the
// system-wide one when it exists and only trusted accounts can write it or
// its directory (M-10). Anyone who could write the system file could vouch
// for a host key, so an untrusted one is skipped with a warning rather than
// failing the dial: the confirmed store still applies.
//
// A path checked now can be swapped before it is read, so the system file is
// read once, from the very file that was checked, and its bytes are kept in
// memory. The entry returned for it is not a path but a token for those bytes
// (see memKnownHosts); the host-key code writes them to a private temp file
// only while it builds a checker, and removes that file before returning.
func OpenSSHKnownHosts(home string) []string {
	files := []string{filepath.Join(home, ".ssh", "known_hosts")}
	f := openTrustedSystemKnownHosts()
	if f == nil {
		return files
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, 16<<20))
	if err != nil {
		log.Printf("jumpgate: ignoring %s: %v", systemKnownHosts, err)
		return files
	}
	return append(files, registerKnownHostsBytes(data))
}

// OpenSSHKnownHostsPaths is the same set as OpenSSHKnownHosts, decided by the
// same trust check, as paths for an ssh child process (the TUI and CLI shell),
// which cannot read the in-memory snapshot. The child re-opens the system
// file by name, so unlike Strict it reads whatever is there then; the check
// it passed means only a trusted account could have changed it since.
func OpenSSHKnownHostsPaths(home string) []string {
	files := []string{filepath.Join(home, ".ssh", "known_hosts")}
	f := openTrustedSystemKnownHosts()
	if f == nil {
		return files
	}
	f.Close()
	return append(files, systemKnownHosts)
}

// openTrustedSystemKnownHosts opens the system-wide file when it exists and
// passes the trusted-writable check, or returns nil (with a warning when it
// exists but is not trusted).
func openTrustedSystemKnownHosts() *os.File {
	if _, err := os.Stat(systemKnownHosts); err != nil {
		return nil
	}
	f, err := openTrustedKnownHosts(systemKnownHosts)
	if err != nil {
		log.Printf("jumpgate: ignoring %s: %v", systemKnownHosts, err)
		return nil
	}
	return f
}

// memTokenPrefix starts a known_hosts entry that stands for in-memory bytes.
// It begins with a NUL, which no file path can.
const memTokenPrefix = "\x00jumpgate-known-hosts:"

var (
	memMu         sync.Mutex
	memKnownHosts = map[string][]byte{} // token -> trusted bytes
)

// registerKnownHostsBytes keeps data and returns its token. Equal content
// shares a token, so the registry only grows with distinct file contents.
func registerKnownHostsBytes(data []byte) string {
	sum := sha256.Sum256(data)
	tok := memTokenPrefix + hex.EncodeToString(sum[:])
	memMu.Lock()
	memKnownHosts[tok] = data
	memMu.Unlock()
	return tok
}

// memKnownHostsBytes returns the bytes behind a token.
func memKnownHostsBytes(entry string) ([]byte, bool) {
	if !strings.HasPrefix(entry, memTokenPrefix) {
		return nil, false
	}
	memMu.Lock()
	defer memMu.Unlock()
	b, ok := memKnownHosts[entry]
	return b, ok
}
