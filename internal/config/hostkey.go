package config

import (
	"os"
	"path/filepath"

	"golang.org/x/crypto/ssh"

	"github.com/valve-tech/jumpgate/internal/executor"
)

// StrictHostKey is the one Strict host-key policy: keys a person confirmed
// (ConfirmedHostsFile, never written by trust-on-first-use) or listed in the
// operator's OpenSSH ~/.ssh/known_hosts. The CLI's pairing, the server's agent
// dials and the legacy path for confirmed or paired boxes all use it, so they
// can never trust different hosts. The second result asks each host for the
// key types on record, so a host known by one type is not mistaken for a
// changed one.
func StrictHostKey() (ssh.HostKeyCallback, func(string) []string, error) {
	confirmed, err := ConfirmedHostsFile()
	if err != nil {
		return nil, nil, err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, nil, err
	}
	known := filepath.Join(home, ".ssh", "known_hosts")
	return executor.Strict(confirmed, known), executor.KnownHostKeyAlgorithms(confirmed, known), nil
}
