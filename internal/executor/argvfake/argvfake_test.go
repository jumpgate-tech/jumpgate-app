package argvfake

import (
	"context"
	"errors"
	"testing"

	"github.com/valve-tech/jumpgate/internal/executor"
)

func TestFakeRefusesTheShellAndAnswersArgvByLongestPrefix(t *testing.T) {
	f := New().Script("docker", executor.Result{Stdout: "short"}).Script("docker info", executor.Result{Stdout: "long"})
	if res, _ := f.RunArgv(context.Background(), []string{"docker", "info", "--format", "x"}, nil); res.Stdout != "long" {
		t.Fatalf("got %q, want the longest prefix's answer", res.Stdout)
	}
	if _, err := f.Run(context.Background(), "uname", nil); !errors.Is(err, executor.ErrNoPOSIXShell) || len(f.ShellCalls()) != 1 {
		t.Fatalf("Run: %v, calls %v", err, f.ShellCalls())
	}
}
