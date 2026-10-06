//go:build !notui

package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/valve-tech/jumpgate/internal/apiclient"
	"github.com/valve-tech/jumpgate/internal/tui"
)

// tuiBuilt: controller builds carry the TUI; agent builds use -tags notui.
const tuiBuilt = true

// Seams for tests: how the TUI reaches the server, and the program itself.
var (
	connectTUI = func(ctx context.Context, exe string) (*apiclient.Client, error) {
		return apiclient.Connect(ctx, apiclient.Options{Start: true, Exe: exe})
	}
	runTUIProgram = tui.Run
)

// tuiOptions wires the TUI to this process: the server client, this binary
// for foreground children, the browser opener and the restart.
func tuiOptions(c *apiclient.Client, exe, host string) tui.Options {
	cur := c
	return tui.Options{
		Backend:  c,
		Self:     exe,
		Hostname: host,
		Location: time.Local,
		OpenWeb:  func(ctx context.Context) error { return openWebApp(ctx, io.Discard) },
		Restart: func(ctx context.Context) (tui.Backend, error) {
			nc, err := apiclient.Restart(ctx, cur)
			if err != nil {
				return nil, err
			}
			cur = nc // a second restart stops the server the first one started
			return nc, nil
		},
	}
}

func runTUI(ctx context.Context, in io.Reader, out io.Writer) int {
	exe, _ := os.Executable()
	c, err := connectTUI(ctx, exe)
	if err != nil {
		fmt.Fprintf(out, "jumpgate: %v\n(run `jumpgate home` for the plain screen)\n", err)
		pauseIfStandalone(in, out)
		return 1
	}
	host, _ := os.Hostname()
	host, _, _ = strings.Cut(strings.ToLower(host), ".")
	if host == "" {
		host = "local"
	}
	if err := runTUIProgram(ctx, in, out, tuiOptions(c, exe, host)); err != nil {
		fmt.Fprintf(out, "jumpgate: %v\n", err)
		pauseIfStandalone(in, out)
		return 1
	}
	return 0
}
