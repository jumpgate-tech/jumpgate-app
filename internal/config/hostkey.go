package config

import (
	"os"

	"golang.org/x/crypto/ssh"

	"github.com/valve-tech/jumpgate/internal/executor"
)

// strictFiles are the files Strict trusts: the confirmed-only store and the
// operator's OpenSSH known_hosts files (the user's, and the system-wide one when
// trusted).
func strictFiles() (confirmed string, known []string, err error) {
	confirmed, err = ConfirmedHostsFile()
	if err != nil {
		return "", nil, err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", nil, err
	}
	return confirmed, executor.OpenSSHKnownHosts(home), nil
}

// StrictHostKey is the one Strict host-key policy: keys a person confirmed
// (ConfirmedHostsFile, never written by trust-on-first-use) or listed in the
// operator's OpenSSH ~/.ssh/known_hosts. The CLI's pairing, the server's agent
// dials and the legacy path for boxes on record all use it, so they can never
// trust different hosts. The second result asks each host for the key types on
// record, so a host known by one type is not mistaken for a changed one.
func StrictHostKey() (ssh.HostKeyCallback, func(string) []string, error) {
	confirmed, known, err := strictFiles()
	if err != nil {
		return nil, nil, err
	}
	return executor.Strict(confirmed, known...), executor.KnownHostKeyAlgorithms(confirmed, known...), nil
}

// HostOnRecord reports whether StrictHostKey would recognise hostport (the
// host:port string DialSSH hands its callback): a key for it is in the
// confirmed store, or a plain (non-@cert-authority) line for it is in the
// operator's OpenSSH known_hosts. These are Strict's own rules, so a host on
// record here is one Strict checks rather than one it calls unknown.
func HostOnRecord(hostport string) (bool, error) {
	confirmed, known, err := strictFiles()
	if err != nil {
		return false, err
	}
	return len(executor.KnownHostKeyAlgorithms(confirmed, known...)(hostport)) > 0, nil
}
