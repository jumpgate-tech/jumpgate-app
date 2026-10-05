//go:build !windows

package executor

import (
	"io"
	"net"
	"os"
	"syscall"
	"time"
)

// dialAgent connects to the ssh-agent named by SSH_AUTH_SOCK, or returns nil
// when none is configured or reachable: the agent is optional.
func dialAgent(deadline time.Time) io.ReadWriteCloser {
	sock := os.Getenv("SSH_AUTH_SOCK")
	if sock == "" {
		return nil
	}
	// A socket another user owns could be a squatter collecting sign requests.
	if fi, err := os.Stat(sock); err != nil {
		return nil
	} else if st, ok := fi.Sys().(*syscall.Stat_t); ok && !socketOwnerOK(st.Uid, uint32(os.Geteuid())) {
		return nil
	}
	c, err := (&net.Dialer{Deadline: deadline}).Dial("unix", sock)
	if err != nil {
		return nil
	}
	_ = c.SetDeadline(deadline) // a wedged agent cannot outlast the dial
	return c
}

// socketOwnerOK accepts a socket owned by this user or root, and any socket
// when running as root (sudo keeps the invoking user's SSH_AUTH_SOCK).
func socketOwnerOK(uid, euid uint32) bool {
	return euid == 0 || uid == 0 || uid == euid
}
