package setup

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/valve-tech/jumpgate/internal/executor"
	"github.com/valve-tech/jumpgate/internal/executor/argvfake"
)

// Local paths follow this machine's rules (C:\Users\… on Windows); remote
// ones stay POSIX.
func TestPathsFollowTheMachineActedOn(t *testing.T) {
	f := argvfake.New()
	home, err := homeOn(context.Background(), f)
	if err != nil || home != f.Home {
		t.Fatalf("homeOn = %q, %v", home, err)
	}
	if got := joinOn(f, home, ".valve-node-app", "erpc.yaml"); got != filepath.Join(f.Home, ".valve-node-app", "erpc.yaml") {
		t.Fatalf("joinOn(local) = %q", got)
	}
	remote := newFakeExecutor().script(`printf '%s\n' "$HOME"`, executor.Result{Stdout: "/home/o\n"})
	if home, err := homeOn(context.Background(), remote); err != nil || home != "/home/o" {
		t.Fatalf("homeOn(remote) = %q, %v", home, err)
	}
	if got := joinOn(remote, "/home/o", ".valve-node-app", "erpc.yaml"); got != "/home/o/.valve-node-app/erpc.yaml" {
		t.Fatalf("joinOn(remote) = %q", got)
	}
}
