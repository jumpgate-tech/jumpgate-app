package agentclient

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/valve-tech/jumpgate/internal/executor"
)

// DefaultSocket is where the agent listens.
const DefaultSocket = "/run/jumpgate/agent.sock"

// transport returns an HTTP client whose every connection is to the agent
// socket — local, or through the SSH connection as a direct-streamlocal
// channel — and a closer for the SSH connection.
func transport(ctx context.Context, t Target) (*http.Client, func() error, error) {
	sock := t.Socket
	if sock == "" {
		sock = DefaultSocket
	}
	var dial func(ctx context.Context) (net.Conn, error)
	closer := func() error { return nil }

	if t.Local {
		dial = func(ctx context.Context) (net.Conn, error) {
			return (&net.Dialer{Timeout: 5 * time.Second}).DialContext(ctx, "unix", sock)
		}
	} else {
		client, err := executor.DialSSH(ctx, t.SSH)
		if err != nil {
			// A host-key failure is a security event, not an outage: keep
			// its type and leave ErrUnreachable out of it.
			if errors.Is(err, executor.ErrUnknownHost) || errors.Is(err, executor.ErrHostKeyMismatch) {
				return nil, nil, fmt.Errorf("agentclient: ssh %s: %w", t.SSH.Host, err)
			}
			return nil, nil, fmt.Errorf("%w: ssh %s: %w", ErrUnreachable, t.SSH.Host, err)
		}
		closer = client.Close
		dial = func(context.Context) (net.Conn, error) { return dialUnix(client, sock) }
	}
	hc := &http.Client{
		Timeout: 2 * time.Minute,
		Transport: &http.Transport{
			DialContext:     func(ctx context.Context, _, _ string) (net.Conn, error) { return dial(ctx) },
			MaxIdleConns:    2,
			IdleConnTimeout: 30 * time.Second,
		},
	}
	return hc, closer, nil
}

// dialUnix opens a direct-streamlocal channel. x/crypto's Client.Dial
// supports "unix" for exactly this.
func dialUnix(c *ssh.Client, path string) (net.Conn, error) { return c.Dial("unix", path) }
