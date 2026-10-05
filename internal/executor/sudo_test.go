//go:build unix

package executor

import (
	"context"
	"io"
	"io/fs"
	"strings"
	"testing"
)

type recordExec struct {
	cmds   []string
	stdins []string
}

func (r *recordExec) Run(_ context.Context, cmd string, o *RunOpts) (Result, error) {
	r.cmds = append(r.cmds, cmd)
	if o != nil && o.Stdin != nil {
		b, _ := io.ReadAll(o.Stdin)
		r.stdins = append(r.stdins, string(b))
	}
	return Result{}, nil
}
func (r *recordExec) WriteFile(context.Context, string, []byte, fs.FileMode) error { return nil }
func (r *recordExec) ReadFile(context.Context, string) ([]byte, error)             { return nil, nil }
func (r *recordExec) Close() error                                                 { return nil }

func TestSudoWrapsCommandsAndKeepsStdin(t *testing.T) {
	inner := &recordExec{}
	s := Sudo(inner)
	_, _ = s.Run(context.Background(), "echo 'hi'", nil)
	if inner.cmds[0] != `sudo -n sh -c 'echo '"'"'hi'"'"''` {
		t.Fatalf("cmd = %s", inner.cmds[0])
	}
	_ = s.WriteFile(context.Background(), "/etc/x", []byte("secret"), 0o600)
	if !strings.HasPrefix(inner.cmds[1], "sudo -n sh -c ") || strings.Contains(inner.cmds[1], "secret") || inner.stdins[0] != "secret" {
		t.Fatalf("WriteFile via sudo: cmd %s stdin %q", inner.cmds[1], inner.stdins)
	}
}
