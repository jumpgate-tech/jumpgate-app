//go:build !notui

package main

import (
	"context"
	"io"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/valve-tech/jumpgate/internal/apiclient"
	"github.com/valve-tech/jumpgate/internal/daemon"
	"github.com/valve-tech/jumpgate/internal/tui"
)

func TestPlainWanted(t *testing.T) {
	for env, want := range map[string]bool{"": false, "JUMPGATE_PLAIN=1": true, "TERM=dumb": true, "TERM=xterm-256color": false} {
		k, v, _ := strings.Cut(env, "=")
		got := plainWanted(func(name string) string {
			if name == k {
				return v
			}
			return ""
		})
		if got != want {
			t.Errorf("%q: plainWanted = %v", env, got)
		}
	}
}

// JUMPGATE_PLAIN keeps the line-based home, for scripts and screen readers.
func TestTerminalHomeFallsBackToThePlainScreen(t *testing.T) {
	fakeHome(t, true)
	t.Setenv("JUMPGATE_PLAIN", "1")
	var out strings.Builder
	if code := runTerminalHome(context.Background(), strings.NewReader("q\n"), &out); code != 0 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(out.String(), "server:") {
		t.Fatalf("not the plain home:\n%s", out.String())
	}
}

func TestTerminalHomeRunsTheTUIWithTheRightOptions(t *testing.T) {
	t.Setenv("JUMPGATE_PLAIN", "")
	t.Setenv("TERM", "xterm-256color")
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())
	oldConnect, oldRun := connectTUI, runTUIProgram
	t.Cleanup(func() { connectTUI, runTUIProgram = oldConnect, oldRun })
	connectTUI = func(context.Context, string) (*apiclient.Client, error) {
		return apiclient.New(daemon.Info{HTTPAddr: "127.0.0.1:1", Token: "tok"}), nil
	}
	var got tui.Options
	runTUIProgram = func(_ context.Context, _ io.Reader, _ io.Writer, o tui.Options) error { got = o; return nil }
	if code := runTerminalHome(context.Background(), strings.NewReader(""), io.Discard); code != 0 {
		t.Fatalf("exit %d", code)
	}
	if got.Backend == nil || got.Self == "" || got.OpenWeb == nil || got.Restart == nil || got.Hostname == "" {
		t.Fatalf("options %+v", got)
	}
	if got.Location != time.Local {
		t.Fatalf("location %v", got.Location)
	}
}

// The agent runs as root on every box; its build links no terminal UI.
func TestAgentBuildLinksNoTUI(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("no go toolchain on PATH")
	}
	out, err := exec.Command("go", "list", "-tags", "notui", "-deps", ".").CombinedOutput()
	if err != nil {
		t.Fatalf("go list: %v\n%s", err, out)
	}
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, "charm.land/") || strings.Contains(strings.ToLower(line), "charmbracelet") || strings.Contains(line, "/internal/tui") {
			t.Fatalf("the notui build links %s", line)
		}
	}
}
