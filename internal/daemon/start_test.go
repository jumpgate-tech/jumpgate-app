package daemon

import (
	"context"
	"errors"
	"os/exec"
	"slices"
	"testing"

	"github.com/valve-tech/jumpgate/internal/testutil"
)

// I-7: the auto-started server must not hold the CLI's working directory
// open (on Windows that blocks deleting or renaming the folder, often the
// unzipped download). It runs from the run dir.
func TestEnsureRunningStartsTheServerInTheRunDir(t *testing.T) {
	testutil.Home(t)
	var got *exec.Cmd
	old := startServer
	startServer = func(cmd *exec.Cmd) (*exec.Cmd, error) { got = cmd; return nil, errors.New("test: not starting") }
	t.Cleanup(func() { startServer = old })

	if _, err := EnsureRunning(context.Background(), "/path/to/jumpgate"); err == nil {
		t.Fatal("EnsureRunning succeeded with a start that failed")
	}
	dir, _ := RunDir()
	if got == nil || got.Dir != dir {
		t.Fatalf("server cmd.Dir = %v, want %q", got, dir)
	}
	if !slices.Equal(got.Args[1:], []string{"serve", "--no-open"}) {
		t.Fatalf("server args %v", got.Args)
	}
}
