package executor

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

// ErrNoPOSIXShell reports that this machine cannot run local mode because it
// has no POSIX shell. Callers can test for it with errors.Is.
var ErrNoPOSIXShell = errors.New("local mode requires a POSIX shell")

// localShellError returns a non-nil error when goos cannot run local mode.
//
// Every command string in this codebase is POSIX shell — pipelines, `&&`,
// heredocs, `systemctl`, `journalctl` — written for the Linux node being
// provisioned. Handing those to `cmd /C` or PowerShell would not run them, it
// would fail in a hundred different confusing ways halfway through a setup,
// so there is deliberately no Windows shell port here. A Windows control
// plane's supported path is an SSH target pointing at a Linux host; local mode
// is only ever "the control plane IS the node", which a Windows box never is.
//
// Taking goos as an argument rather than reading runtime.GOOS inline keeps
// this decision unit-testable from any host OS.
func localShellError(goos string) error {
	if goos == "windows" {
		return fmt.Errorf("%w, which %s does not provide: add the node as an SSH target pointing at a Linux host instead of using local mode", ErrNoPOSIXShell, goos)
	}
	return nil
}

// LocalAvailable returns nil if local mode works on this machine, or an error
// wrapping ErrNoPOSIXShell if it does not. Call it at target-construction time
// so an unsupported control plane fails immediately with an actionable
// message, rather than midway through a setup run.
func LocalAvailable() error {
	return localShellError(runtime.GOOS)
}

// local runs commands and touches files on the machine it executes on.
type local struct {
	// unsupported is non-nil when this machine cannot run local mode (see
	// localShellError). Every method returns it rather than pretending: a
	// half-working local executor that writes systemd units into C:\var\lib
	// but cannot run a single command is worse than a clear refusal.
	unsupported error
}

// NewLocal returns an Executor that runs commands on the local machine. On a
// host with no POSIX shell every call on the returned Executor fails with
// ErrNoPOSIXShell; prefer checking LocalAvailable at construction time so the
// failure surfaces before a target is persisted.
func NewLocal() Executor {
	return &local{unsupported: LocalAvailable()}
}

func (l *local) Run(ctx context.Context, cmd string, opts *RunOpts) (Result, error) {
	if l.unsupported != nil {
		return Result{}, l.unsupported
	}
	c := exec.CommandContext(ctx, "sh", "-c", cmd)

	// See proc_unix.go / proc_windows.go: on unix this runs cmd in its own
	// process group so ctx cancellation can kill every descendant it spawned,
	// not just the direct `sh` PID.
	setupProcAttrs(c)
	// Backstop: if killing the process group somehow doesn't unblock our
	// stdout read within this window (e.g. a doubly-detached daemon in a
	// different process group), exec forcibly closes the stdout pipe so Run
	// can still return instead of hanging forever.
	c.WaitDelay = 2 * time.Second

	var stdoutBuf, stderrBuf bytes.Buffer
	c.Stderr = &stderrBuf

	stdoutPipe, err := c.StdoutPipe()
	if err != nil {
		return Result{}, err
	}

	if err := c.Start(); err != nil {
		return Result{}, err
	}

	var streamFn StreamFunc
	if opts != nil {
		streamFn = opts.Stream
	}
	w := &lineStreamer{buf: &stdoutBuf, fn: streamFn}

	copyErrCh := make(chan error, 1)
	go func() {
		_, err := io.Copy(w, stdoutPipe)
		copyErrCh <- err
	}()

	// It is incorrect to call Wait before all reads from the StdoutPipe have
	// completed, so wait for the copy goroutine first.
	copyErr := <-copyErrCh
	w.Flush()
	waitErr := c.Wait()

	if ctx.Err() != nil {
		return Result{}, ctx.Err()
	}
	if copyErr != nil {
		return Result{}, copyErr
	}

	result := Result{
		Stdout: stdoutBuf.String(),
		Stderr: stderrBuf.String(),
	}

	if waitErr != nil {
		exitErr, ok := waitErr.(*exec.ExitError)
		if !ok {
			return Result{}, waitErr
		}
		result.ExitCode = exitErr.ExitCode()
		return result, nil
	}

	return result, nil
}

// WriteFile writes to the machine this process runs on, so path is a LOCAL
// path and filepath (host separator) is the correct choice here — the opposite
// of the SSH executor, whose paths are always POSIX. See remotepath.go.
func (l *local) WriteFile(_ context.Context, path string, content []byte, mode fs.FileMode) error {
	if l.unsupported != nil {
		return l.unsupported
	}
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	if err := os.WriteFile(path, content, mode); err != nil {
		return err
	}
	// os.WriteFile only applies mode on create; an existing file keeps its
	// old permissions, and umask can still narrow them even then. Chmod
	// unconditionally so mode is authoritative in both cases.
	return os.Chmod(path, mode)
}

// ReadFile reads from the machine this process runs on; path is a LOCAL path.
func (l *local) ReadFile(_ context.Context, path string) ([]byte, error) {
	if l.unsupported != nil {
		return nil, l.unsupported
	}
	return os.ReadFile(path)
}

func (l *local) Close() error {
	return nil
}
