//go:build !notui

package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
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
	// cur is the client in use: Start and Restart replace it off the TUI's
	// update loop, and Server reads it on it.
	var mu sync.Mutex
	cur := c
	set := func(nc *apiclient.Client) {
		mu.Lock()
		defer mu.Unlock()
		cur = nc
	}
	get := func() *apiclient.Client {
		mu.Lock()
		defer mu.Unlock()
		return cur
	}
	return tui.Options{
		Backend:       c,
		Self:          exe,
		Hostname:      host,
		Bell:          func() { fmt.Fprint(os.Stderr, "\a") },
		Location:      time.Local,
		OpenWeb:       func(ctx context.Context) error { return openWebApp(ctx, io.Discard) },
		SystemService: c.Info().System,
		Start: func(ctx context.Context) (tui.Backend, error) {
			nc, err := apiclient.Connect(ctx, apiclient.Options{Start: true, Exe: exe})
			if err != nil {
				return nil, err
			}
			set(nc)
			return nc, nil
		},
		Server: func() tui.ServerInfo {
			i := get().Info() // the token stays here
			return tui.ServerInfo{PID: i.PID, HTTPAddr: i.HTTPAddr, StartedAt: i.StartedAt, System: i.System, Shape: i.Shape}
		},
		Restart: func(ctx context.Context) (tui.Backend, error) {
			nc, err := apiclient.Restart(ctx, get())
			if err != nil {
				return nil, err
			}
			set(nc) // a second restart stops the server the first one started
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
