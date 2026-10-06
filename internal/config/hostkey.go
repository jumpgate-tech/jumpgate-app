package config

import (
	"os"
	"path/filepath"

	"golang.org/x/crypto/ssh"

	"github.com/valve-tech/jumpgate/internal/executor"
)

// strictFiles are the files Strict trusts: the confirmed-only store and the
// operator's OpenSSH known_hosts.
func strictFiles() (confirmed, known string, err error) {
	confirmed, err = ConfirmedHostsFile()
	if err != nil {
		return "", "", err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", "", err
	}
	return confirmed, filepath.Join(home, ".ssh", "known_hosts"), nil
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
	return executor.Strict(confirmed, known), executor.KnownHostKeyAlgorithms(confirmed, known), nil
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
	return len(executor.KnownHostKeyAlgorithms(confirmed, known)(hostport)) > 0, nil
}

// RecordedHostKeys lists the keys Strict trusts for hostport, by store: the
// confirmed store, which jumpgate writes and can forget from, and the
// operator's OpenSSH known_hosts, which jumpgate only ever reads.
func RecordedHostKeys(hostport string) (confirmed, openssh []ssh.PublicKey, err error) {
	confirmedFile, known, err := strictFiles()
	if err != nil {
		return nil, nil, err
	}
	if confirmed, err = executor.HostKeysOnRecord(confirmedFile, hostport); err != nil {
		return nil, nil, err
	}
	return confirmed, executor.OpenSSHHostKeys(hostport, known), nil
}
