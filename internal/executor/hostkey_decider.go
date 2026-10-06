package executor

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// ErrUnknownHost marks a host whose key nobody has confirmed yet.
var ErrUnknownHost = errors.New("unknown SSH host")

// ErrHostKeyMismatch marks a host whose presented key contradicts one on
// record (or is revoked): a possible man-in-the-middle, or a rebuilt host.
// Callers report it as a security failure, never as an outage.
var ErrHostKeyMismatch = errors.New("SSH host key mismatch")

// mismatchError carries a mismatch's own message and wraps both the sentinel
// and any underlying cause.
type mismatchError struct {
	msg   string
	cause error
}

func (e *mismatchError) Error() string { return e.msg }

func (e *mismatchError) Unwrap() []error {
	if e.cause == nil {
		return []error{ErrHostKeyMismatch}
	}
	return []error{ErrHostKeyMismatch, e.cause}
}

func mismatchf(cause error, format string, a ...any) error {
	return &mismatchError{msg: fmt.Sprintf(format, a...), cause: cause}
}

// UnknownHostError carries what an operator needs to confirm a host: its
// address and key fingerprint.
type UnknownHostError struct {
	Host        string
	Fingerprint string
	Key         ssh.PublicKey
}

func (e *UnknownHostError) Error() string {
	return fmt.Sprintf("%s presents an unconfirmed host key %s", e.Host, e.Fingerprint)
}

func (e *UnknownHostError) Unwrap() error { return ErrUnknownHost }

// Fingerprint is OpenSSH's SHA256 form, the one `ssh-keygen -lf` prints, so
// an operator can compare it with the console of the box.
func Fingerprint(key ssh.PublicKey) string { return ssh.FingerprintSHA256(key) }

// Strict accepts a host only if its key was confirmed by a person: recorded in
// confirmedFile (written only by RecordHostKey after an explicit confirmation,
// never by trust-on-first-use) or listed in one of the operator's OpenSSH
// known_hosts files. It never records anything itself: an unknown host is an
// *UnknownHostError the caller must put in front of a person.
//
// The OpenSSH files are always consulted, even when confirmedFile matches: a
// different key or a @revoked entry there is a hard error.
func Strict(confirmedFile string, opensshFiles ...string) ssh.HostKeyCallback {
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		if _, ok := remote.(*net.TCPAddr); !ok || remote == nil {
			remote = remoteFromHostname(hostname)
		}
		var present []string
		for _, f := range opensshFiles {
			if _, err := os.Stat(f); err == nil {
				present = append(present, f)
			}
		}
		opensshOK := false
		if len(present) > 0 {
			cb, err := knownhosts.New(present...)
			if err != nil {
				return fmt.Errorf("read known_hosts: %w", err)
			}
			err = cb(hostname, remote, key)
			var keyErr *knownhosts.KeyError
			var revoked *knownhosts.RevokedError
			switch {
			case err == nil:
				opensshOK = true
			case errors.As(err, &revoked):
				return mismatchf(err, "host key for %s is revoked in your OpenSSH known_hosts: %v", hostname, err)
			case errors.As(err, &keyErr) && len(keyErr.Want) > 0 && !onlyCertAuthorities(keyErr.Want):
				return mismatchf(err, "host key mismatch for %s against your OpenSSH known_hosts: %v", hostname, err)
			case errors.As(err, &keyErr):
				// Unknown there (or only a CA covers it): not confirmed.
			default:
				return err
			}
		}
		known, err := lookupHostKey(confirmedFile, hostname)
		if err != nil {
			return err
		}
		if known != nil {
			if !bytes.Equal(known.Marshal(), key.Marshal()) {
				return mismatchf(nil, "host key mismatch for %s: presented %s does not match %s on record in %s (possible man-in-the-middle, or the host was rebuilt)",
					hostname, Fingerprint(key), Fingerprint(known), confirmedFile)
			}
			return nil
		}
		if opensshOK {
			return nil
		}
		return &UnknownHostError{Host: hostname, Fingerprint: Fingerprint(key), Key: key}
	}
}

// KnownHostKeyAlgorithms answers, for a host:port, the host-key algorithms of
// every key a person confirmed (confirmedFile) or listed in the OpenSSH
// known_hosts files, so the handshake asks for a type on record. Without it
// x/crypto asks for ECDSA first, and a box known only by its ed25519 key
// would look like a mismatch. RSA keys expand to rsa-sha2-512 and
// rsa-sha2-256. A host known nowhere, or covered only by a @cert-authority
// line, gets nil: the defaults.
func KnownHostKeyAlgorithms(confirmedFile string, opensshFiles ...string) func(hostport string) []string {
	return func(hostport string) []string {
		var keys []ssh.PublicKey
		if k, err := lookupHostKey(confirmedFile, hostport); err == nil && k != nil {
			keys = append(keys, k)
		}
		keys = append(keys, opensshHostKeys(hostport, opensshFiles)...)
		var algos []string
		seen := map[string]bool{}
		add := func(a string) {
			if !seen[a] {
				seen[a] = true
				algos = append(algos, a)
			}
		}
		for _, k := range keys {
			if k.Type() == ssh.KeyAlgoRSA {
				add(ssh.KeyAlgoRSASHA512)
				add(ssh.KeyAlgoRSASHA256)
				continue
			}
			add(k.Type())
		}
		return algos
	}
}

// opensshHostKeys lists the plain host keys the OpenSSH files hold for
// hostport. knownhosts has no lookup, so it is asked to check a throwaway key:
// the resulting KeyError lists every key on record for the host.
func opensshHostKeys(hostport string, files []string) []ssh.PublicKey {
	var present []string
	for _, f := range files {
		if _, err := os.Stat(f); err == nil {
			present = append(present, f)
		}
	}
	if len(present) == 0 {
		return nil
	}
	cb, err := knownhosts.New(present...)
	if err != nil {
		return nil
	}
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil
	}
	probe, err := ssh.NewPublicKey(pub)
	if err != nil {
		return nil
	}
	var keyErr *knownhosts.KeyError
	if !errors.As(cb(hostport, remoteFromHostname(hostport), probe), &keyErr) {
		return nil
	}
	var keys []ssh.PublicKey
	for _, w := range keyErr.Want {
		if !onlyCertAuthorities([]knownhosts.KnownKey{w}) {
			keys = append(keys, w.Key)
		}
	}
	return keys
}

// hostKeyAlgorithms is cfg's algorithm list for addr, or nil for the
// defaults.
func hostKeyAlgorithms(cfg SSHConfig, addr string) []string {
	if cfg.HostKeyAlgorithms == nil {
		return nil
	}
	return cfg.HostKeyAlgorithms(addr)
}

// noCommonHostKey turns a failed negotiation under a restricted algorithm
// list into a mismatch: the host no longer offers any key type on record.
func noCommonHostKey(err error, addr string, offered []string) error {
	var neg *ssh.AlgorithmNegotiationError
	if len(offered) > 0 && errors.As(err, &neg) && neg.What == "host key" {
		return mismatchf(err, "host key mismatch for %s: it offers none of the host key types on record (%s; it offers %s); possible man-in-the-middle, or the host was rebuilt",
			addr, strings.Join(offered, ", "), strings.Join(neg.RequestedAlgorithms, ", "))
	}
	return err
}

// remoteFromHostname builds the *net.TCPAddr knownhosts insists on when the
// caller passed none (or something else).
func remoteFromHostname(hostname string) net.Addr {
	host, portStr, err := net.SplitHostPort(hostname)
	if err != nil {
		host, portStr = hostname, "22"
	}
	port, _ := strconv.Atoi(portStr)
	a := &net.TCPAddr{Port: port}
	if ip := net.ParseIP(host); ip != nil {
		a.IP = ip
	}
	return a
}

// onlyCertAuthorities reports whether every entry that "wants" a different key
// is a @cert-authority line. knownhosts does not expose the marker, so the
// source line is re-read; if it cannot be read the entry counts as a pinned
// key, the safe direction (a mismatch refuses).
func onlyCertAuthorities(want []knownhosts.KnownKey) bool {
	for _, k := range want {
		data, err := os.ReadFile(k.Filename)
		if err != nil {
			return false
		}
		lines := strings.Split(string(data), "\n")
		if k.Line < 1 || k.Line > len(lines) {
			return false
		}
		f := strings.Fields(lines[k.Line-1])
		if len(f) == 0 || f[0] != "@cert-authority" {
			return false
		}
	}
	return true
}

// errCaptured stops the handshake once the key is in hand.
var errCaptured = errors.New("host key captured")

// CaptureHostKey connects far enough to read the host's key and then hangs up,
// before any authentication. It is how the CLI shows a fingerprint to confirm.
func CaptureHostKey(ctx context.Context, cfg SSHConfig) (ssh.PublicKey, error) {
	port := cfg.Port
	if port == 0 {
		port = 22
	}
	addr := net.JoinHostPort(cfg.Host, strconv.Itoa(port))

	// One deadline covers connect and handshake; the caller's context can
	// only shorten it. Same bound as DialSSH, for the same reason.
	deadline := time.Now().Add(handshakeTimeout)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	dialCtx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()

	var conn net.Conn
	if cfg.Jump != nil {
		// Never authenticate to a jump host nobody confirmed, and never
		// fall back to trust-on-first-use for it.
		if cfg.Jump.HostKey == nil {
			return nil, fmt.Errorf("jump host %s has no host-key policy: confirm the jump host first", cfg.Jump.Host)
		}
		jump, err := DialSSH(dialCtx, *cfg.Jump)
		if err != nil {
			return nil, fmt.Errorf("jump host %s: %w", cfg.Jump.Host, err)
		}
		through, err := jump.DialContext(dialCtx, "tcp", addr)
		if err != nil {
			jump.Close()
			return nil, fmt.Errorf("via jump host %s: %w", cfg.Jump.Host, err)
		}
		conn = &jumpConn{Conn: through, jump: jump}
	} else {
		var d net.Dialer
		var err error
		conn, err = d.DialContext(dialCtx, "tcp", addr)
		if err != nil {
			return nil, err
		}
	}
	defer conn.Close()

	// Watchdog: closing conn is the only way to abort a stalled handshake,
	// since SetDeadline is unsupported on a channel through a jump host.
	stop := context.AfterFunc(dialCtx, func() { conn.Close() })
	defer stop()

	var got ssh.PublicKey
	algos := hostKeyAlgorithms(cfg, addr)
	_, _, _, err := ssh.NewClientConn(conn, addr, &ssh.ClientConfig{
		User:              cfg.User,
		HostKeyAlgorithms: algos,
		HostKeyCallback: func(_ string, _ net.Addr, key ssh.PublicKey) error {
			got = key
			return errCaptured
		},
	})
	if got == nil {
		if cerr := dialCtx.Err(); cerr != nil {
			return nil, fmt.Errorf("no host key from %s: %w", addr, cerr)
		}
		if merr := noCommonHostKey(err, addr, algos); merr != err {
			return nil, merr
		}
		return nil, fmt.Errorf("no host key from %s: %w", addr, err)
	}
	return got, nil
}

// OpenSSHHostKeys lists the plain (non-@cert-authority) keys the OpenSSH
// known_hosts files hold for hostport, the ones Strict would accept there.
func OpenSSHHostKeys(hostport string, opensshFiles ...string) []ssh.PublicKey {
	return opensshHostKeys(hostport, opensshFiles)
}
