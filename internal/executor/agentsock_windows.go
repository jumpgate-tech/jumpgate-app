//go:build windows

package executor

import (
	"io"
	"net"
	"os"
	"strings"
	"time"
)

// openSSHAgentPipe is where Windows OpenSSH's agent service, and 1Password's
// agent, listen. They are named pipes, and SSH_AUTH_SOCK is normally unset.
const openSSHAgentPipe = `\\.\pipe\openssh-ssh-agent`

// dialAgent connects to the agent: SSH_AUTH_SOCK when it is set, otherwise
// the OpenSSH pipe. A \\.\pipe\ path is opened as a file, whose handle is
// the io.ReadWriter agent.NewClient needs; any other value is a unix socket
// (MSYS2, Cygwin). A pipe handle opened this way has no deadline, so a
// wedged agent is bounded only by the SSH handshake deadline around it.
func dialAgent(deadline time.Time) io.ReadWriteCloser {
	sock := os.Getenv("SSH_AUTH_SOCK")
	if sock == "" {
		sock = openSSHAgentPipe
	}
	if !strings.HasPrefix(sock, `\\.\pipe\`) {
		c, err := (&net.Dialer{Deadline: deadline}).Dial("unix", sock)
		if err != nil {
			return nil
		}
		_ = c.SetDeadline(deadline)
		return c
	}
	f, err := os.OpenFile(sock, os.O_RDWR, 0)
	if err != nil {
		return nil
	}
	return f
}
