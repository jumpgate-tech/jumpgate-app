// cmd/jumpgate/home_terminal_test.go
package main

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/daemon"
)

func fakeHome(t *testing.T, running bool) *int {
	t.Helper()
	opened := 0
	old := termHome
	termHome = homeDeps{
		find: func(context.Context) (daemon.Info, bool, error) {
			return daemon.Info{PID: 42, HTTPAddr: "127.0.0.1:8799"}, running, nil
		},
		load: func() (config.Config, error) {
			return config.Config{
				Controller: &config.Controller{KeyStore: "file", Address: "0xabc"},
				Targets:    []config.Target{{ID: "a", Agent: &config.AgentPairing{Address: "0x1"}}, {ID: "b"}},
			}, nil
		},
		open: func(context.Context, io.Writer) error { opened++; return nil },
	}
	t.Cleanup(func() { termHome = old })
	return &opened
}

func TestTerminalHomeShowsTheOverviewAndHelp(t *testing.T) {
	fakeHome(t, true)
	var out strings.Builder
	if code := runTerminalHome(context.Background(), strings.NewReader("h\nq\n"), &out); code != 0 {
		t.Fatalf("exit %d", code)
	}
	for _, want := range []string{"running, pid 42", "0xabc (file)", "2 (1 paired)", "[o] open the web app", "commands", "hosts add"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("output lacks %q:\n%s", want, out.String())
		}
	}
}

func TestTerminalHomeOpensTheWebApp(t *testing.T) {
	opened := fakeHome(t, false)
	var out strings.Builder
	runTerminalHome(context.Background(), strings.NewReader("o\nq\n"), &out)
	if *opened != 1 {
		t.Fatalf("open called %d times, want 1", *opened)
	}
	if !strings.Contains(out.String(), "not running") {
		t.Errorf("overview should say the server is not running:\n%s", out.String())
	}
}

// Review Focus 5: input already at its end (a script, a closed terminal)
// prints the overview once and returns; it never spins.
func TestTerminalHomeEndsOnEOF(t *testing.T) {
	opened := fakeHome(t, true)
	var out strings.Builder
	if code := runTerminalHome(context.Background(), strings.NewReader(""), &out); code != 0 {
		t.Fatalf("exit %d", code)
	}
	if n := strings.Count(out.String(), "server:"); n != 1 {
		t.Fatalf("overview printed %d times, want 1", n)
	}
	if *opened != 0 {
		t.Fatal("opened the web app on EOF")
	}
}

func TestTerminalHomeStaysOpenUntilQuit(t *testing.T) {
	fakeHome(t, true)
	var out strings.Builder
	runTerminalHome(context.Background(), strings.NewReader("\n\nx\ns\nq\nh\n"), &out)
	if strings.Count(out.String(), "server:") != 2 {
		t.Fatalf("want the overview twice (start and s), got:\n%s", out.String())
	}
	if strings.Contains(out.String(), "commands (run as") {
		t.Fatal("kept reading after q")
	}
	if !strings.Contains(out.String(), `unknown choice "x"`) {
		t.Fatal("an unknown choice was not reported")
	}
}

func TestIsTerminalLaunch(t *testing.T) {
	oldIn, oldOut := stdinIsTerminal, stdoutIsTerminal
	t.Cleanup(func() { stdinIsTerminal, stdoutIsTerminal = oldIn, oldOut })
	for _, c := range []struct {
		args    []string
		in, out bool
		want    bool
	}{
		{[]string{"jumpgate"}, true, true, true},
		{[]string{"jumpgate", "--bind", "x"}, true, true, false},
		{[]string{"jumpgate"}, false, true, false},
		{[]string{"jumpgate"}, true, false, false},
	} {
		stdinIsTerminal = func() bool { return c.in }
		stdoutIsTerminal = func() bool { return c.out }
		if got := isTerminalLaunch(c.args); got != c.want {
			t.Errorf("isTerminalLaunch(%v) in=%v out=%v = %v, want %v", c.args, c.in, c.out, got, c.want)
		}
	}
}

// A double-clicked window is 80 columns; nothing the home prints may wrap.
func TestTerminalHomeFitsEightyColumns(t *testing.T) {
	fakeHome(t, true)
	var out strings.Builder
	runTerminalHome(context.Background(), strings.NewReader("h\nq\n"), &out)
	for _, l := range strings.Split(out.String(), "\n") {
		if len(l) > 80 {
			t.Errorf("line over 80 columns (%d): %q", len(l), l)
		}
	}
}

func TestPauseIfStandaloneWaitsForEnter(t *testing.T) {
	t.Setenv("JUMPGATE_LAUNCHER", "1")
	var out strings.Builder
	pauseIfStandalone(strings.NewReader("\n"), &out)
	if !strings.Contains(out.String(), "press Enter") {
		t.Fatalf("no prompt: %q", out.String())
	}
	t.Setenv("JUMPGATE_LAUNCHER", "")
	out.Reset()
	pauseIfStandalone(strings.NewReader(""), &out)
	if out.Len() != 0 {
		t.Fatalf("prompted though not standalone: %q", out.String())
	}
}

func TestTerminalHomeTruncatesLongErrors(t *testing.T) {
	long := errors.New(strings.Repeat("/very/long/path", 20))
	old := termHome
	termHome = homeDeps{
		find: func(context.Context) (daemon.Info, bool, error) { return daemon.Info{}, false, long },
		load: func() (config.Config, error) { return config.Config{}, long },
		open: func(context.Context, io.Writer) error { return nil },
	}
	t.Cleanup(func() { termHome = old })
	var out strings.Builder
	runTerminalHome(context.Background(), strings.NewReader(""), &out)
	for _, l := range strings.Split(out.String(), "\n") {
		if len(l) > 80 {
			t.Errorf("line over 80 columns (%d): %q", len(l), l)
		}
	}
	if !strings.Contains(out.String(), "...") {
		t.Fatal("long error was not truncated visibly")
	}
}
