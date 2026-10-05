package executor

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// handshakeTimeout bounds TCP connect plus the whole SSH handshake. A host
// that accepts TCP and then stalls in key exchange would otherwise hang the
// caller forever: ssh.ClientConfig.Timeout covers only the connect. A var so
// tests can shorten it.
var handshakeTimeout = 10 * time.Second

// ErrNoHostKeyPolicy means a dial named no host-key policy. There is no
// default: a caller picks Strict (keys a person confirmed) or, on the legacy
// web-UI path only, TOFUHostKeyCallback, so a forgotten policy fails loudly
// instead of silently trusting whatever key the network presents.
var ErrNoHostKeyPolicy = errors.New("ssh: no host-key policy; pass Strict or TOFUHostKeyCallback explicitly")

// DialSSH connects to cfg, through cfg.Jump if set. cfg.HostKey is required. A
// jump host with no policy of its own is checked with cfg's.
func DialSSH(ctx context.Context, cfg SSHConfig) (*ssh.Client, error) {
	if cfg.HostKey == nil {
		return nil, fmt.Errorf("dial %s: %w", cfg.Host, ErrNoHostKeyPolicy)
	}
	port := cfg.Port
	if port == 0 {
		port = 22
	}
	addr := net.JoinHostPort(cfg.Host, strconv.Itoa(port))

	// One deadline covers the whole dial: connect, handshake and any agent
	// signing. The caller's context can only shorten it.
	deadline := time.Now().Add(handshakeTimeout)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	dialCtx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()

	auth, release, err := authMethods(cfg, deadline)
	if err != nil {
		return nil, err
	}
	defer release() // auth is used only during the handshake below
	hostKey := cfg.HostKey
	// No ClientConfig.Timeout: the connect is bounded by dialCtx and the
	// handshake by the watchdog below.
	algos := hostKeyAlgorithms(cfg, addr)
	clientCfg := &ssh.ClientConfig{User: cfg.User, Auth: auth, HostKeyCallback: hostKey, HostKeyAlgorithms: algos}

	var conn net.Conn
	if cfg.Jump != nil {
		jumpCfg := *cfg.Jump
		if jumpCfg.HostKey == nil {
			// A jump host with no policy of its own gets this dial's, never a
			// default: the callback is keyed by hostname, so it serves both hops.
			jumpCfg.HostKey = cfg.HostKey
			if jumpCfg.HostKeyAlgorithms == nil {
				jumpCfg.HostKeyAlgorithms = cfg.HostKeyAlgorithms
			}
		}
		jump, err := DialSSH(dialCtx, jumpCfg)
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
		conn, err = d.DialContext(dialCtx, "tcp", addr)
		if err != nil {
			return nil, err
		}
	}

	// Watchdog: closing conn is the only way to abort a stalled handshake.
	// SetDeadline is no use here, a channel through a jump host does not
	// support it. Closing also unblocks a wedged agent via release.
	stop := context.AfterFunc(dialCtx, func() {
		conn.Close()
		release()
	})
	c, chans, reqs, err := ssh.NewClientConn(conn, addr, clientCfg)
	if !stop() && err == nil {
		// The watchdog fired just as the handshake finished: conn is closed.
		c.Close()
		err = dialCtx.Err()
	}
	if err != nil {
		conn.Close()
		if cerr := dialCtx.Err(); cerr != nil {
			if errors.Is(cerr, context.DeadlineExceeded) && ctx.Err() == nil {
				return nil, fmt.Errorf("ssh handshake with %s timed out after %v: %w", addr, handshakeTimeout, cerr)
			}
			return nil, fmt.Errorf("ssh handshake with %s: %w", addr, cerr)
		}
		return nil, noCommonHostKey(err, addr, algos)
	}
	return ssh.NewClient(c, chans, reqs), nil
}

// jumpConn is a connection through a jump host that also closes the jump
// host's own connection with it, so closing the final client leaks nothing.
type jumpConn struct {
	net.Conn
	jump *ssh.Client
}

func (c *jumpConn) Close() error {
	err := c.Conn.Close()
	c.jump.Close()
	return err
}

// authMethods builds one publickey method offering the KeyPath key first and
// then the ssh-agent's keys. It is one method because the ssh client tries
// each method name once, so separate file and agent methods would silently
// drop the second. The key file goes first so an agent that is locked or fails
// to sign (which aborts the method) cannot block a key that works.
// Passphrase-protected key files are supported only through an agent.
func authMethods(cfg SSHConfig, deadline time.Time) (methods []ssh.AuthMethod, release func(), err error) {
	var fileSigner ssh.Signer
	var agentClient agent.ExtendedAgent
	var agentConn io.ReadWriteCloser
	release = func() {
		if agentConn != nil {
			agentConn.Close()
		}
	}
	if c := dialAgent(deadline); c != nil {
		agentConn = c
		agentClient = agent.NewClient(c)
	}
	if cfg.KeyPath != "" {
		keyBytes, err := os.ReadFile(cfg.KeyPath)
		if err != nil {
			release()
			return nil, nil, fmt.Errorf("read private key %s: %w", cfg.KeyPath, err)
		}
		signer, err := ssh.ParsePrivateKey(keyBytes)
		var missing *ssh.PassphraseMissingError
		if errors.As(err, &missing) {
			if agentClient == nil {
				release()
				return nil, nil, fmt.Errorf("private key %s is passphrase-protected; load it into ssh-agent (ssh-add %s)", cfg.KeyPath, cfg.KeyPath)
			}
		} else if err != nil {
			release()
			return nil, nil, fmt.Errorf("parse private key %s: %w", cfg.KeyPath, err)
		} else {
			fileSigner = signer
		}
	}
	if fileSigner == nil && agentClient == nil {
		release()
		return nil, nil, errors.New("no SSH credentials: set a key path or run an ssh-agent")
	}
	return []ssh.AuthMethod{ssh.PublicKeysCallback(func() ([]ssh.Signer, error) {
		var signers []ssh.Signer
		if fileSigner != nil {
			signers = append(signers, fileSigner)
		}
		if agentClient != nil {
			if s, err := agentClient.Signers(); err == nil {
				signers = append(signers, s...)
			} else if fileSigner == nil {
				return nil, err
			}
		}
		return signers, nil
	})}, release, nil
}
