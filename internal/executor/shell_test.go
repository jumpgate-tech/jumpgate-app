package executor

import (
	"context"
	"errors"
	"io/fs"
	"testing"
)

type plainExec struct{}

func (plainExec) Run(context.Context, string, *RunOpts) (Result, error)        { return Result{}, nil }
func (plainExec) WriteFile(context.Context, string, []byte, fs.FileMode) error { return nil }
func (plainExec) ReadFile(context.Context, string) ([]byte, error)             { return nil, nil }
func (plainExec) Close() error                                                 { return nil }

// Only a local executor on a machine with no POSIX shell refuses; every other
// executor (SSH, a working local one, a test fake) can run shell commands.
func TestRequireShell(t *testing.T) {
	if err := RequireShell(plainExec{}); err != nil {
		t.Fatalf("an executor without ShellError: %v, want nil", err)
	}
	if err := RequireShell(&local{}); err != nil {
		t.Fatalf("a local executor with a shell: %v, want nil", err)
	}
	if err := RequireShell(&local{unsupported: localShellError("windows")}); !errors.Is(err, ErrNoPOSIXShell) {
		t.Fatalf("a shell-less local executor: %v, want ErrNoPOSIXShell", err)
	}
}
