// internal/daemon/detach_unix.go
//go:build unix

package daemon

import (
	"os/exec"
	"syscall"
)

// detach starts the server in its own session, so closing the terminal that
// launched it does not deliver SIGHUP to it.
func detach(cmd *exec.Cmd) { cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true} }
