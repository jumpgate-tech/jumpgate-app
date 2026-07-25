package executor

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/ssh"
)

// Trust-on-first-use host key storage. Everything in this file touches the
// *operator's own* filesystem (the known_hosts file next to the config), so it
// uses "path/filepath" — the host-dependent separator is the correct one here.
// This is deliberately kept out of ssh.go, which builds paths for the remote
// Linux target and must never import filepath; see remotepath.go.

// tofuHostKeyCallback implements trust-on-first-use host key verification
// backed by a flat file of "host:port keytype base64key" lines.
func tofuHostKeyCallback(hostKeyFile string) ssh.HostKeyCallback {
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		known, err := lookupHostKey(hostKeyFile, hostname)
		if err != nil {
			return err
		}
		if known == nil {
			return appendHostKey(hostKeyFile, hostname, key)
		}
		if !bytes.Equal(known.Marshal(), key.Marshal()) {
			return fmt.Errorf("host key mismatch for %s: presented %s key does not match the key on record in %s (possible man-in-the-middle attack, or the host was rebuilt)", hostname, key.Type(), hostKeyFile)
		}
		return nil
	}
}

// lookupHostKey returns the recorded public key for hostname in hostKeyFile,
// or nil if hostKeyFile doesn't exist or has no entry for hostname.
func lookupHostKey(hostKeyFile, hostname string) (ssh.PublicKey, error) {
	data, err := os.ReadFile(hostKeyFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read host key file %s: %w", hostKeyFile, err)
	}

	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 3 || fields[0] != hostname {
			continue
		}
		keyBytes, err := base64.StdEncoding.DecodeString(fields[2])
		if err != nil {
			return nil, fmt.Errorf("host key file %s: malformed entry for %s: %w", hostKeyFile, hostname, err)
		}
		key, err := ssh.ParsePublicKey(keyBytes)
		if err != nil {
			return nil, fmt.Errorf("host key file %s: malformed entry for %s: %w", hostKeyFile, hostname, err)
		}
		return key, nil
	}
	return nil, nil
}

// appendHostKey records key for hostname in hostKeyFile, creating the file
// with mode 0600 if it doesn't already exist.
func appendHostKey(hostKeyFile, hostname string, key ssh.PublicKey) error {
	// Local path: filepath is correct here (see the file comment above).
	if dir := filepath.Dir(hostKeyFile); dir != "" {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return fmt.Errorf("create host key file dir: %w", err)
		}
	}

	f, err := os.OpenFile(hostKeyFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("open host key file %s: %w", hostKeyFile, err)
	}
	defer f.Close()

	line := fmt.Sprintf("%s %s %s\n", hostname, key.Type(), base64.StdEncoding.EncodeToString(key.Marshal()))
	if _, err := f.WriteString(line); err != nil {
		return fmt.Errorf("write host key file %s: %w", hostKeyFile, err)
	}
	return nil
}
