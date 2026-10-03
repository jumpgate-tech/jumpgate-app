package executor

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// ErrUnknownHost marks a host whose key nobody has confirmed yet.
var ErrUnknownHost = errors.New("unknown SSH host")

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

// Strict accepts a host only if its key is already recorded, in jumpgate's own
// file or in one of the operator's OpenSSH known_hosts files. Unlike
// trust-on-first-use it never records anything itself: an unknown host is an
// *UnknownHostError the caller must put in front of a person.
func Strict(jumpgateFile string, opensshFiles ...string) ssh.HostKeyCallback {
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		known, err := lookupHostKey(jumpgateFile, hostname)
		if err != nil {
			return err
		}
		if known != nil {
			if !bytes.Equal(known.Marshal(), key.Marshal()) {
				return fmt.Errorf("host key mismatch for %s: presented %s does not match %s on record in %s (possible man-in-the-middle, or the host was rebuilt)",
					hostname, Fingerprint(key), Fingerprint(known), jumpgateFile)
			}
			return nil
		}
		var present []string
		for _, f := range opensshFiles {
			if _, err := os.Stat(f); err == nil {
				present = append(present, f)
			}
		}
		if len(present) > 0 {
			cb, err := knownhosts.New(present...)
			if err != nil {
				return fmt.Errorf("read known_hosts: %w", err)
			}
			err = cb(hostname, remote, key)
			var keyErr *knownhosts.KeyError
			switch {
			case err == nil:
				return nil
			case errors.As(err, &keyErr) && len(keyErr.Want) > 0:
				return fmt.Errorf("host key mismatch for %s against your OpenSSH known_hosts: %w", hostname, err)
			case errors.As(err, &keyErr):
				// Not listed there either: fall through to unknown.
			default:
				return err
			}
		}
		return &UnknownHostError{Host: hostname, Fingerprint: Fingerprint(key), Key: key}
	}
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
	_, _, _, err := ssh.NewClientConn(conn, addr, &ssh.ClientConfig{
		User: cfg.User,
		HostKeyCallback: func(_ string, _ net.Addr, key ssh.PublicKey) error {
			got = key
			return errCaptured
		},
	})
	if got == nil {
		if cerr := dialCtx.Err(); cerr != nil {
			return nil, fmt.Errorf("no host key from %s: %w", addr, cerr)
		}
		return nil, fmt.Errorf("no host key from %s: %w", addr, err)
	}
	return got, nil
}
