package main

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/valve-tech/jumpgate/internal/daemon"
	"github.com/valve-tech/jumpgate/internal/server"
	"github.com/valve-tech/jumpgate/internal/testutil"
)

func TestOpenerCommandPerOS(t *testing.T) {
	for goos, want := range map[string]string{
		"darwin":  "open http://x/login?code=c",
		"windows": "rundll32 url.dll,FileProtocolHandler http://x/login?code=c",
		"linux":   "xdg-open http://x/login?code=c",
	} {
		name, args := openerCommand(goos, "http://x/login?code=c")
		if got := strings.Join(append([]string{name}, args...), " "); got != want {
			t.Errorf("%s: %q, want %q", goos, got, want)
		}
	}
}

// A login link must point at loopback: the server redeems codes only from
// this machine, and a wildcard bind is not an address a browser can open.
func TestLoginURLUsesALoopbackHostForAWildcardBind(t *testing.T) {
	for addr, want := range map[string]string{
		"127.0.0.1:8799": "http://127.0.0.1:8799/login?code=c",
		"0.0.0.0:8799":   "http://127.0.0.1:8799/login?code=c",
		"[::]:8799":      "http://[::1]:8799/login?code=c",
		"[::1]:8799":     "http://[::1]:8799/login?code=c",
		"10.0.0.2:8799":  "http://10.0.0.2:8799/login?code=c",
	} {
		if got := loginURL(addr, "c"); got != want {
			t.Errorf("loginURL(%q) = %q, want %q", addr, got, want)
		}
	}
}

// startUnixServer runs a real server on a unix socket and returns what
// server.json would say about it.
func startUnixServer(t *testing.T) daemon.Info {
	t.Helper()
	token := server.NewSessionToken()
	sock := filepath.Join(testutil.ShortTempDir(t), "s.sock")
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		_ = server.New(server.Config{Token: token, UI: fstest.MapFS{}}).ServeUnix(ctx, sock)
		close(done)
	}()
	t.Cleanup(func() { cancel(); <-done })
	info := daemon.Info{Socket: sock, Token: token, HTTPAddr: "127.0.0.1:8799"}
	for i := 0; i < 100; i++ {
		if _, err := requestLoginCode(context.Background(), info); err == nil {
			return info
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("server did not come up")
	return info
}

// stubOpener replaces the opener and ensureRunning for one test and returns
// a pointer to the argv the opener was handed.
func stubOpener(t *testing.T, info daemon.Info, err error) *[]string {
	t.Helper()
	var argv []string
	oldStart, oldEnsure := startOpener, ensureRunning
	startOpener = func(name string, args ...string) error { argv = append([]string{name}, args...); return err }
	ensureRunning = func(context.Context, string) (daemon.Info, error) { return info, nil }
	t.Cleanup(func() { startOpener, ensureRunning = oldStart, oldEnsure })
	return &argv
}

// I-6: what reaches the opener's argv is a one-time login link, never the
// session token; nor is the token printed.
func TestOpenWebAppNeverPutsTheTokenOnACommandLine(t *testing.T) {
	info := startUnixServer(t)
	argv := stubOpener(t, info, nil)

	var out strings.Builder
	if err := openWebApp(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	line := strings.Join(*argv, " ")
	if strings.Contains(line, info.Token) || strings.Contains(out.String(), info.Token) {
		t.Fatalf("the session token leaked: argv %q, output %q", line, out.String())
	}
	if !strings.Contains(line, "http://127.0.0.1:8799/login?code=") {
		t.Fatalf("opener argv %q, want a login link", line)
	}
	// The browser got the link; printing it too would only hand a live
	// credential to the terminal's scrollback.
	if strings.Contains(out.String(), "code=") {
		t.Fatalf("output %q prints the link although the browser opened", out.String())
	}
}

func TestOpenWebAppPrintsTheLinkWhenNoBrowserOpens(t *testing.T) {
	info := startUnixServer(t)
	stubOpener(t, info, errors.New("exec: \"xdg-open\": executable file not found in $PATH"))
	var out strings.Builder
	if err := openWebApp(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "within 60 seconds") || !strings.Contains(out.String(), "/login?code=") {
		t.Fatalf("output %q, want the link and its lifetime", out.String())
	}
	if strings.Contains(out.String(), info.Token) {
		t.Fatalf("output %q carries the session token", out.String())
	}
}

// A second app launch finds the server already running: it opens a login
// link (or prints one with --no-open), never the token.
func TestOpenRunningServerHandsOffALoginLink(t *testing.T) {
	info := startUnixServer(t)
	argv := stubOpener(t, info, nil)

	var out strings.Builder
	openRunningServer(context.Background(), info, false, false, &out)
	if line := strings.Join(*argv, " "); !strings.Contains(line, "/login?code=") || strings.Contains(line, info.Token) {
		t.Fatalf("opener argv %q, want a login link without the token", line)
	}

	*argv = nil
	out.Reset()
	openRunningServer(context.Background(), info, false, true, &out)
	if len(*argv) != 0 {
		t.Fatalf("--no-open ran an opener: %q", *argv)
	}
	if !strings.Contains(out.String(), "/login?code=") || strings.Contains(out.String(), info.Token) {
		t.Fatalf("--no-open output %q, want a login link without the token", out.String())
	}
}

func TestCmdOpenRefusesArguments(t *testing.T) {
	if code := cmdOpen([]string{"extra"}); code != exitCode("usage") {
		t.Fatalf("cmdOpen with an argument: %d, want the usage code", code)
	}
}
