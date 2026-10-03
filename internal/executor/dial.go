package executor

import (
	"context"
	"errors"
	"fmt"
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

// DialSSH connects to cfg, through cfg.Jump if set.
func DialSSH(ctx context.Context, cfg SSHConfig) (*ssh.Client, error) {
	port := cfg.Port
	if port == 0 {
		port = 22
	}
	addr := net.JoinHostPort(cfg.Host, strconv.Itoa(port))

	auth, release, err := authMethods(cfg)
	if err != nil {
		return nil, err
	}
	defer release() // auth is used only during the handshake below
	hostKey := cfg.HostKey
	if hostKey == nil {
		hostKey = tofuHostKeyCallback(cfg.HostKeyFile)
	}
	clientCfg := &ssh.ClientConfig{User: cfg.User, Auth: auth, HostKeyCallback: hostKey, Timeout: handshakeTimeout}

	var conn net.Conn
	if cfg.Jump != nil {
		jump, err := DialSSH(ctx, *cfg.Jump)
		if err != nil {
			return nil, fmt.Errorf("jump host %s: %w", cfg.Jump.Host, err)
		}
		through, err := jump.DialContext(ctx, "tcp", addr)
		if err != nil {
			jump.Close()
			return nil, fmt.Errorf("via jump host %s: %w", cfg.Jump.Host, err)
		}
		conn = &jumpConn{Conn: through, jump: jump}
	} else {
		d := net.Dialer{Timeout: handshakeTimeout}
		conn, err = d.DialContext(ctx, "tcp", addr)
		if err != nil {
			return nil, err
		}
	}

	deadline := time.Now().Add(handshakeTimeout)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	_ = conn.SetDeadline(deadline)
	c, chans, reqs, err := ssh.NewClientConn(conn, addr, clientCfg)
	if err != nil {
		conn.Close()
		return nil, err
	}
	_ = conn.SetDeadline(time.Time{})
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

// authMethods offers KeyPath first, then ssh-agent keys. The key file goes
// first because an agent that is locked or refuses to sign (a failed sign
// aborts the whole publickey method) must not block a key that works.
// Passphrase-protected key files are supported only through an agent.
// authMethods builds one publickey method offering the KeyPath key first and
// then the ssh-agent's keys. It is one method because the ssh client tries
// each method name once, so separate file and agent methods would silently
// drop the second. The key file goes first so an agent that is locked or fails
// to sign (which aborts the method) cannot block a key that works.
// Passphrase-protected key files are supported only through an agent.
func authMethods(cfg SSHConfig) (methods []ssh.AuthMethod, release func(), err error) {
	var fileSigner ssh.Signer
	var agentClient agent.ExtendedAgent
	var agentConn net.Conn
	release = func() {
		if agentConn != nil {
			agentConn.Close()
		}
	}
	if sock := os.Getenv("SSH_AUTH_SOCK"); sock != "" {
		if c, err := net.Dial("unix", sock); err == nil {
			agentConn = c
			agentClient = agent.NewClient(c)
		}
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
