//go:build unix

package signer

import (
	"os"
	"os/exec"
	"syscall"
)

// isolateTool runs secret-tool in its own process group, so the timeout kills
// anything it started (D-Bus autolaunch can spawn helpers) and not just
// secret-tool itself. Only secret-tool: `op` may ask for a password on the
// terminal, which a process outside the terminal's foreground group cannot
// read (it would stop on SIGTTIN instead of prompting).
func isolateTool(c *exec.Cmd, name string) {
	if name != "secret-tool" {
		return
	}
	c.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	c.Cancel = func() error {
		if c.Process == nil {
			return os.ErrProcessDone
		}
		return syscall.Kill(-c.Process.Pid, syscall.SIGKILL)
	}
}
