package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/valve-tech/jumpgate/internal/buildinfo"
	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/daemon"
)

// homeDeps is what the terminal home reads and does; a seam for tests.
type homeDeps struct {
	find func(context.Context) (daemon.Info, bool, error)
	load func() (config.Config, error)
	open func(context.Context, io.Writer) error
}

// termHome's open indirects through openWebApp at call time, so Task 8 can
// replace that variable's file without touching this one.
var termHome = homeDeps{
	find: daemon.Find,
	load: config.Load,
	open: func(ctx context.Context, w io.Writer) error { return openWebApp(ctx, w) },
}

// stdoutIsTerminal reports whether stdout is a character device. A variable
// so tests can stand in for a terminal.
var stdoutIsTerminal = func() bool {
	fi, err := os.Stdout.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}

// isTerminalLaunch reports whether this is a bare `jumpgate` in a terminal,
// which opens the terminal home (spec D9). The macOS .app, the Windows GUI
// exe and a service have no terminal and still run the app.
func isTerminalLaunch(args []string) bool {
	return len(args) == 1 && stdinIsTerminal() && stdoutIsTerminal() && !inAppBundle()
}

// runTerminalHome is what a person sees when they start `jumpgate` with no
// arguments in a terminal, and what every launcher runs: the Windows console
// exe, Jumpgate Terminal.app, the Linux .desktop entry. It must not depend on
// how it was started.
//
// TUI SEAM: the TUI (sub-project 3) replaces this function's body and keeps
// its signature; main, the launchers and CI stay as they are. Until then it
// is a line-based menu (spec D8) that loops until q or end of input, so a
// double-clicked window does not flash and close.
func runTerminalHome(ctx context.Context, in io.Reader, out io.Writer) int {
	r := bufio.NewReader(in)
	printOverview(ctx, out)
	for {
		fmt.Fprint(out, "\n[o] open the web app   [s] status   [h] commands   [q] quit\n> ")
		line, err := r.ReadString('\n')
		choice := strings.ToLower(strings.TrimSpace(line))
		switch choice {
		case "o":
			if oerr := termHome.open(ctx, out); oerr != nil {
				fmt.Fprintf(out, "could not open the web app: %v\n", oerr)
			}
		case "s":
			printOverview(ctx, out)
		case "h", "?":
			printHelp(out)
		case "q", "quit", "exit":
			return 0
		case "":
		default:
			fmt.Fprintf(out, "unknown choice %q\n", choice)
		}
		if err != nil {
			fmt.Fprintln(out)
			return 0 // end of input
		}
	}
}

// printOverview is the home's status block: what is running, which key signs,
// and how many machines there are. Lines stay under 80 columns.
func printOverview(ctx context.Context, out io.Writer) {
	fmt.Fprintf(out, "jumpgate %s\n", buildinfo.Version())
	switch info, ok, err := termHome.find(ctx); {
	case err != nil:
		fmt.Fprintf(out, "  server:   unknown (%v)\n", err)
	case ok:
		fmt.Fprintf(out, "  server:   running, pid %d, http://%s/\n", info.PID, info.HTTPAddr)
	default:
		fmt.Fprintln(out, "  server:   not running (it starts when a command needs it)")
	}
	c, err := termHome.load()
	if err != nil {
		fmt.Fprintf(out, "  config:   %v\n", err)
		return
	}
	if c.Controller == nil {
		fmt.Fprintln(out, "  key:      none yet; run `jumpgate keys init`")
	} else {
		fmt.Fprintf(out, "  key:      %s (%s)\n", c.Controller.Address, c.Controller.KeyStore)
	}
	paired := 0
	for _, t := range c.Targets {
		if t.Agent != nil {
			paired++
		}
	}
	fmt.Fprintf(out, "  machines: %d (%d paired)\n", len(c.Targets), paired)
	if len(c.Targets) == 0 {
		fmt.Fprintln(out, "            add one: jumpgate hosts add NAME --ssh USER@HOST")
	}
}

// pauseIfStandalone waits for Enter when this process owns its window, so an
// error printed before exit can be read.
func pauseIfStandalone(in io.Reader, out io.Writer) {
	if !launchedStandalone() {
		return
	}
	fmt.Fprint(out, "press Enter to close this window")
	_, _ = bufio.NewReader(in).ReadString('\n')
}
