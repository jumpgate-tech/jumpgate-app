package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"

	"github.com/valve-tech/jumpgate/internal/daemon"
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

// openBrowser opens url in the default browser. url must be a one-time login
// link, never one carrying the session token: an opener's command line is
// readable by every local user (I-6, D4).
func openBrowser(url string) error {
	name, args := openerCommand(hostGOOS, url)
	return startOpener(name, args...)
}

// loginURL is the one-time login link for a server listening on addr. The
// server redeems codes only from this machine, so a wildcard bind becomes
// its loopback address, the one a local browser can reach.
func loginURL(addr, code string) string {
	if host, port, err := net.SplitHostPort(addr); err == nil {
		switch host {
		case "", "0.0.0.0":
			addr = net.JoinHostPort("127.0.0.1", port)
		case "::":
			addr = net.JoinHostPort("::1", port)
		}
	}
	return "http://" + addr + "/login?code=" + code
}

// requestLoginCode asks a running server, over its socket, for a login code.
func requestLoginCode(ctx context.Context, info daemon.Info) (string, error) {
	res, err := info.Client().Do(mustRequest(ctx, info, "/api/login-code", nil))
	if err != nil {
		return "", fmt.Errorf("ask the server for a login link: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ask the server for a login link: %s", res.Status)
	}
	var body struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(io.LimitReader(res.Body, 4096)).Decode(&body); err != nil || body.Code == "" {
		return "", fmt.Errorf("ask the server for a login link: bad answer")
	}
	return body.Code, nil
}

// handOffLogin opens the login link for code in the browser, or, when open
// is false or no browser opens, prints it with its lifetime. A link the
// browser received is not printed: it stays a credential until redeemed.
func handOffLogin(out io.Writer, addr, code string, open bool) {
	link := loginURL(addr, code)
	if open {
		err := openBrowser(link)
		if err == nil {
			fmt.Fprintf(out, "opened the jumpgate web app (http://%s/) in your browser\n", addr)
			return
		}
		fmt.Fprintf(out, "could not open a browser (%v)\n", err)
	}
	// 60 seconds is the server's loginCodeTTL.
	fmt.Fprintf(out, "open this one-time link within 60 seconds (`jumpgate open` makes a new one):\n  %s\n", link)
}

// openWebApp starts the server if needed and opens the web app in the
// browser with a fresh login link. With no browser to open, it prints the
// link and how long it works. The terminal home calls it through
// termHome.open.
func openWebApp(ctx context.Context, out io.Writer) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	info, err := ensureRunning(ctx, exe)
	if err != nil {
		return err
	}
	code, err := requestLoginCode(ctx, info)
	if err != nil {
		return err
	}
	handOffLogin(out, info.HTTPAddr, code, true)
	return nil
}
