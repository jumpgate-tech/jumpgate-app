//go:build windows

package daemon

import (
	"errors"
	"os/exec"
	"syscall"

	"golang.org/x/sys/windows"
)

const (
	detachedProcess        = 0x00000008
	createNewProcessGroup  = 0x00000200
	createBreakawayFromJob = 0x01000000
)

// startProc is a seam for the retry test.
var startProc = func(cmd *exec.Cmd) error { return cmd.Start() }

// startDetached starts the server with no console, in its own process group,
// and outside the caller's job object (I-7). An OpenSSH session, Windows
// Terminal, VS Code and CI all run the CLI inside a job that kills every
// member when it closes, which would take the server with it. A job that
// forbids breakaway refuses the start with ERROR_ACCESS_DENIED; the server
// then starts inside it, which is still better than not at all.
func startDetached(cmd *exec.Cmd) (*exec.Cmd, error) {
	base := uint32(detachedProcess | createNewProcessGroup)
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: base | createBreakawayFromJob}
	err := startProc(cmd)
	if err == nil || !errors.Is(err, windows.ERROR_ACCESS_DENIED) {
		return cmd, err
	}
	// An exec.Cmd cannot be started twice, so the retry is a fresh one.
	retry := exec.Command(cmd.Path, cmd.Args[1:]...)
	retry.Args = cmd.Args
	retry.Dir, retry.Env, retry.Stdout, retry.Stderr = cmd.Dir, cmd.Env, cmd.Stdout, cmd.Stderr
	retry.SysProcAttr = &syscall.SysProcAttr{CreationFlags: base}
	return retry, startProc(retry)
}
