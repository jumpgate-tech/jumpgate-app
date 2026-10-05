//go:build !windows

package executor

import (
	"io"
	"net"
	"os"
	"time"
)

// dialAgent connects to the ssh-agent named by SSH_AUTH_SOCK, or returns nil
// when none is configured or reachable: the agent is optional.
func dialAgent(deadline time.Time) io.ReadWriteCloser {
	sock := os.Getenv("SSH_AUTH_SOCK")
	if sock == "" {
		return nil
	}
	c, err := (&net.Dialer{Deadline: deadline}).Dial("unix", sock)
	if err != nil {
		return nil
	}
	_ = c.SetDeadline(deadline) // a wedged agent cannot outlast the dial
	return c
}
