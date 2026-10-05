package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/valve-tech/jumpgate/internal/daemon"
)

// TASK 8 MUST DELETE THIS FILE. The default below puts the session token in
// the URL it hands to the browser opener, so the token lands in a process's
// argv, which violates D4/D26. Task 8's openWebApp mints a one-time login
// code instead.
//
// openWebApp is the terminal home's "open the web app". This is the interim
// implementation: it does what main does for an already-running server (print
// the login URL and hand it to the browser), starting the background server
// first when there is none. Task 8 replaces this whole file with its
// one-time-login-code version under the same name and signature; the home
// calls it through termHome.open and needs no change.
var openWebApp = func(ctx context.Context, out io.Writer) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	info, err := daemon.EnsureRunning(ctx, exe)
	if err != nil {
		return err
	}
	url := appURL(info)
	fmt.Fprintln(out, url)
	openBrowser(url)
	return nil
}

// cmdOpen is `jumpgate open`: a thin command over the openWebApp seam, so
// Task 8 swaps the implementation and not the command.
func cmdOpen([]string) int {
	if err := openWebApp(context.Background(), os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "jumpgate:", err)
		return 1
	}
	return 0
}
