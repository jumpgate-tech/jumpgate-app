package tui_test

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/exp/teatest/v2"

	"github.com/valve-tech/jumpgate/internal/apiclient"
	"github.com/valve-tech/jumpgate/internal/catalog"
	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/daemon"
	"github.com/valve-tech/jumpgate/internal/executor"
	"github.com/valve-tech/jumpgate/internal/server"
	"github.com/valve-tech/jumpgate/internal/testutil"
	"github.com/valve-tech/jumpgate/internal/tui"
	"github.com/valve-tech/jumpgate/internal/tui/tuitest"
)

// quietExec answers every command with success and no output: the node
// reads as stopped, which is all this test needs from it.
type quietExec struct{}

func (quietExec) Run(context.Context, string, *executor.RunOpts) (executor.Result, error) {
	return executor.Result{}, nil
}
func (quietExec) WriteFile(context.Context, string, []byte, fs.FileMode) error { return nil }
func (quietExec) ReadFile(context.Context, string) ([]byte, error)             { return nil, nil }
func (quietExec) Close() error                                                 { return nil }

// The whole path: the TUI's client, the real server's routes and fleet
// poller, and the real program loop, on every OS CI runs.
func TestTUIAgainstARealServer(t *testing.T) {
	home := testutil.Home(t) // HOME and USERPROFILE: never the real ~/.jumpgate
	if _, err := config.Update(func(c *config.Config) error {
		c.Targets = []config.Target{{ID: "box", Mode: "ssh", SSH: &executor.SSHConfig{Host: "10.0.0.5", User: "root"},
			Wire: &catalog.WireConfig{ChainID: 369, ExecID: "reth", BeaconID: "lighthouse-pulse", DataDir: "/var/lib/jumpgate-node/369"}}}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	token := server.NewSessionToken()
	srv := server.New(server.Config{Token: token, UI: fstest.MapFS{},
		NewExecutor: func(config.Target) (executor.Executor, error) { return quietExec{}, nil }})
	sock := filepath.Join(home, "s.sock")
	ctx, cancel := context.WithCancel(context.Background())
	served := make(chan error, 1)
	go func() { served <- srv.ServeUnix(ctx, sock) }()
	t.Cleanup(func() {
		cancel()
		select {
		case <-served:
		case <-time.After(15 * time.Second):
			t.Error("the server did not stop")
		}
	})
	for deadline := time.Now().Add(5 * time.Second); ; time.Sleep(10 * time.Millisecond) {
		if _, err := os.Stat(sock); err == nil {
			break
		}
		select {
		case err := <-served:
			t.Skipf("no local socket here: %v", err)
		default:
		}
		if time.Now().After(deadline) {
			t.Fatal("the server never listened")
		}
	}

	a := tui.New(tui.Options{Backend: apiclient.New(daemon.Info{Socket: sock, Token: token}), GOOS: "linux", Getenv: func(string) string { return "" }})
	tm := teatest.NewTestModel(t, a, teatest.WithInitialTermSize(80, 24), teatest.WithProgramOptions(tea.WithColorProfile(colorprofile.Ascii)))
	wait := func(s string) {
		teatest.WaitFor(t, tm.Output(), func(b []byte) bool { return strings.Contains(string(b), s) }, teatest.WithDuration(10*time.Second))
	}
	// Wait on whole new lines: the renderer sends only what changed, so a
	// word inside an updated line may not arrive in one piece.
	wait("1 boxes") // the fleet footer, once /api/fleet/stream delivers the polled row
	tm.Send(tuitest.Key("2"))
	wait("10.0.0.5") // Hosts, from /api/targets
	tm.Send(tuitest.Key("q"))
	tm.WaitFinished(t, teatest.WithFinalTimeout(5*time.Second))
}
