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
			return RecordHostKey(hostKeyFile, hostname, key)
		}
		if !bytes.Equal(known.Marshal(), key.Marshal()) {
			return mismatchf(nil, "host key mismatch for %s: presented %s key does not match the key on record in %s (possible man-in-the-middle attack, or the host was rebuilt)", hostname, key.Type(), hostKeyFile)
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

// RecordHostKey records key for hostname in hostKeyFile, creating the file
// with mode 0600 if it doesn't already exist. Trust-on-first-use uses it for
// HostKeyFile; the CLI also calls it, after an explicit human confirmation,
// on the confirmed-hosts file Strict reads. Those two files must stay apart.
func RecordHostKey(hostKeyFile, hostname string, key ssh.PublicKey) error {
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

// TOFUHostKeyCallback is the trust-on-first-use policy: an unknown host's key
// is appended to hostKeyFile (created 0600), and a known host presenting a
// different key is rejected. It is never a default; only the legacy web-UI
// path asks for it, for boxes nobody confirmed or paired, until sub-project 6
// retires that path.
func TOFUHostKeyCallback(hostKeyFile string) ssh.HostKeyCallback {
	return tofuHostKeyCallback(hostKeyFile)
}

// ErrHostKeyNotRecorded means hostKeyFile has no line for the host whose key
// has the given fingerprint.
var ErrHostKeyNotRecorded = errors.New("no such host key on record")

// HostKeysOnRecord lists every key hostKeyFile holds for hostname, in file
// order. A missing file holds none.
func HostKeysOnRecord(hostKeyFile, hostname string) ([]ssh.PublicKey, error) {
	data, err := os.ReadFile(hostKeyFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read host key file %s: %w", hostKeyFile, err)
	}
	var keys []ssh.PublicKey
	for _, line := range strings.Split(string(data), "\n") {
		if key, ok := parseHostKeyLine(line, hostname); ok {
			keys = append(keys, key)
		}
	}
	return keys, nil
}

// parseHostKeyLine is the key of a "host:port keytype base64key" line for
// hostname; ok is false for any other line, or one that does not parse.
func parseHostKeyLine(line, hostname string) (ssh.PublicKey, bool) {
	fields := strings.Fields(line)
	if len(fields) != 3 || fields[0] != hostname {
		return nil, false
	}
	b, err := base64.StdEncoding.DecodeString(fields[2])
	if err != nil {
		return nil, false
	}
	key, err := ssh.ParsePublicKey(b)
	if err != nil {
		return nil, false
	}
	return key, true
}

// ForgetHostKey removes from hostKeyFile the one line for hostname whose key
// has fingerprint (OpenSSH SHA256 form), leaving every other line, including
// the host's keys of other types, byte for byte. The file is replaced
// atomically: a new file, owner-only like RecordHostKey's, renamed over the
// old one, so a reader sees the old store or the new one and never a partial
// write. No such line is ErrHostKeyNotRecorded and changes nothing. Callers
// serialise ForgetHostKey with RecordHostKey on the same file.
func ForgetHostKey(hostKeyFile, hostname, fingerprint string) error {
	data, err := os.ReadFile(hostKeyFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ErrHostKeyNotRecorded
		}
		return fmt.Errorf("read host key file %s: %w", hostKeyFile, err)
	}
	lines := strings.SplitAfter(string(data), "\n")
	removed := -1
	for i, line := range lines {
		if key, ok := parseHostKeyLine(line, hostname); ok && Fingerprint(key) == fingerprint {
			removed = i
			break
		}
	}
	if removed < 0 {
		return ErrHostKeyNotRecorded
	}
	out := strings.Join(append(lines[:removed:removed], lines[removed+1:]...), "")

	// Local path: filepath is correct here (see the file comment above).
	tmp, err := os.CreateTemp(filepath.Dir(hostKeyFile), "."+filepath.Base(hostKeyFile)+"-*")
	if err != nil {
		return fmt.Errorf("rewrite host key file %s: %w", hostKeyFile, err)
	}
	defer os.Remove(tmp.Name()) // a no-op once renamed
	// CreateTemp already makes it 0600 on Unix; say so explicitly, as
	// RecordHostKey does. (On Windows only the read-only bit exists.)
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return fmt.Errorf("rewrite host key file %s: %w", hostKeyFile, err)
	}
	if _, err := tmp.WriteString(out); err != nil {
		tmp.Close()
		return fmt.Errorf("rewrite host key file %s: %w", hostKeyFile, err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("rewrite host key file %s: %w", hostKeyFile, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("rewrite host key file %s: %w", hostKeyFile, err)
	}
	if err := os.Rename(tmp.Name(), hostKeyFile); err != nil {
		return fmt.Errorf("rewrite host key file %s: %w", hostKeyFile, err)
	}
	return nil
}
