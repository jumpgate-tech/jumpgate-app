//go:build unix

package daemon

import (
	"os/exec"
	"syscall"
)

// startDetached starts the server in its own session, so closing the
// terminal that launched it does not deliver SIGHUP to it. A server that must
// also outlive an SSH logout under systemd-logind's KillUserProcesses=yes
// needs `loginctl enable-linger` (see README).
func startDetached(cmd *exec.Cmd) (*exec.Cmd, error) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	return cmd, cmd.Start()
}
