package executor

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io/fs"
	"strings"
)

type sudoExec struct{ inner Executor }

// Sudo runs every command as root through `sudo -n`, for pairing a box whose
// login user is not root. -n makes a password prompt an immediate failure
// instead of a hang: bootstrap needs passwordless sudo, and says so.
//
// Cancelling a command run this way cannot kill root's process group from a
// non-root login; bootstrap steps are idempotent, so a retry converges.
func Sudo(e Executor) Executor { return sudoExec{inner: e} }

func (s sudoExec) Run(ctx context.Context, cmd string, opts *RunOpts) (Result, error) {
	return s.inner.Run(ctx, "sudo -n sh -c "+shQuote(cmd), opts)
}

func (s sudoExec) WriteFile(ctx context.Context, path string, content []byte, mode fs.FileMode) error {
	res, err := s.Run(ctx, writeFileCmd(path, mode), &RunOpts{Stdin: bytes.NewReader(content)})
	if err != nil {
		return err
	}
	if res.ExitCode != 0 {
		return fmt.Errorf("write %s as root: exit %d: %s", path, res.ExitCode, res.Stderr)
	}
	return nil
}

func (s sudoExec) ReadFile(ctx context.Context, path string) ([]byte, error) {
	res, err := s.Run(ctx, readFileCmd(path), nil)
	if err != nil {
		return nil, err
	}
	if res.ExitCode != 0 {
		return nil, fmt.Errorf("read %s as root: exit %d: %s", path, res.ExitCode, res.Stderr)
	}
	return base64.StdEncoding.DecodeString(strings.ReplaceAll(strings.TrimSpace(res.Stdout), "\n", ""))
}

func (s sudoExec) Close() error { return s.inner.Close() }
