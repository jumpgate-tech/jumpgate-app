package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/valve-tech/jumpgate/internal/daemon"
	"github.com/valve-tech/jumpgate/internal/server"
)

// startOpener runs a URL opener without waiting for it, and reaps it in the
// background so it never lingers as a zombie (M-7). A seam for tests.
var startOpener = func(name string, args ...string) error {
	c := exec.Command(name, args...)
	if err := c.Start(); err != nil {
		return err
	}
	go func() { _ = c.Wait() }()
	return nil
}

// openerCommand is the program and arguments that open url on goos.
func openerCommand(goos, url string) (string, []string) {
	switch goos {
	case "darwin":
		return "open", []string{url}
	case "windows":
		return "rundll32", []string{"url.dll,FileProtocolHandler", url}
	}
	return "xdg-open", []string{url}
}

// openBrowser opens target, the path of a login link's owner-only redirect
// file, in the default browser. Never a URL carrying a login code or the
// session token: an opener's command line is readable by every local user,
// who could redeem the code before the browser does (I-6, D4).
func openBrowser(target string) error {
	name, args := openerCommand(hostGOOS, target)
	return startOpener(name, args...)
}

// loginURL is the one-time login link for a server listening on addr.
func loginURL(addr, code string) string { return server.LoginURL(addr, code) }

// loginDir is where the server writes login-link redirect files,
// ~/.jumpgate/run/login, or "" (codes are then only printed) when the run
// directory is unavailable.
func loginDir() string {
	dir, err := daemon.RunDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "login")
}

// requestLoginLink asks a running server, over its socket, for a login code
// and the redirect file that carries it.
func requestLoginLink(ctx context.Context, info daemon.Info) (server.LoginLink, error) {
	var link server.LoginLink
	res, err := info.Client().Do(mustRequest(ctx, info, "/api/login-code", nil))
	if err != nil {
		return link, fmt.Errorf("ask the server for a login link: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return link, fmt.Errorf("ask the server for a login link: %s", res.Status)
	}
	if err := json.NewDecoder(io.LimitReader(res.Body, 4096)).Decode(&link); err != nil || link.Code == "" {
		return link, fmt.Errorf("ask the server for a login link: bad answer")
	}
	link.URL = loginURL(info.HTTPAddr, link.Code)
	return link, nil
}

// handOff is how handOffLogin delivers a login link.
type handOff int

const (
	// handOffOpen opens the browser on the link's redirect file, falling
	// back to printing the link (to a terminal only) when it cannot.
	handOffOpen handOff = iota
	// handOffNoOpen (--no-open) prints the link, to a terminal only.
	handOffNoOpen
	// handOffPrint (`jumpgate open --print`) prints the link wherever
	// stdout goes: the user asked for it.
	handOffPrint
)

// serveHandOff is the hand-off for an app launch with or without --no-open.
func serveHandOff(noOpen bool) handOff {
	if noOpen {
		return handOffNoOpen
	}
	return handOffOpen
}

// outIsTerminal reports whether w is a terminal. A variable so tests can
// stand in for one.
var outIsTerminal = func(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	fi, err := f.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}

// handOffLogin opens link's redirect file in the browser, or prints the
// link with its lifetime: always for handOffPrint, otherwise (no file, no
// browser opens, --no-open) only when out is a terminal. A service manager
// may capture a non-terminal stdout into a log other local users can read,
// and the link is a credential until redeemed, so there the user is told
// to run `jumpgate open --print` instead. A link the browser received is
// never printed.
func handOffLogin(out io.Writer, addr string, link server.LoginLink, mode handOff) {
	if mode == handOffOpen && link.File != "" {
		err := openBrowser(link.File)
		if err == nil {
			fmt.Fprintf(out, "opened the jumpgate web app (http://%s/) in your browser\n", addr)
			return
		}
		fmt.Fprintf(out, "could not open a browser (%v)\n", err)
	}
	if mode != handOffPrint && !outIsTerminal(out) {
		fmt.Fprintf(out, "to sign in to the jumpgate web app (http://%s/), run `jumpgate open --print` from a terminal for a one-time link\n", addr)
		return
	}
	// 60 seconds is the server's loginCodeTTL.
	fmt.Fprintf(out, "open this one-time link within 60 seconds (`jumpgate open` makes a new one):\n  %s\n", link.URL)
}

// openWebApp starts the server if needed and opens the web app in the
// browser with a fresh login link. With no browser to open, it prints the
// link and how long it works. The terminal home calls it through
// termHome.open.
func openWebApp(ctx context.Context, out io.Writer) error { return openWebAppWith(ctx, out, true) }

// openWebAppWith is openWebApp; open=false prints the link instead of
// opening a browser (`jumpgate open --print`).
func openWebAppWith(ctx context.Context, out io.Writer, open bool) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	info, err := ensureRunning(ctx, exe, os.Stderr)
	if err != nil {
		return err
	}
	link, err := requestLoginLink(ctx, info)
	if err != nil {
		return err
	}
	mode := handOffOpen
	if !open {
		mode = handOffPrint
	}
	handOffLogin(out, info.HTTPAddr, link, mode)
	return nil
}
