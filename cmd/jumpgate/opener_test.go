package main

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
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

// startUnixServer runs a real server on a unix socket and on loopback TCP,
// with its login files in a temp dir, and returns what server.json would say
// about it.
func startUnixServer(t *testing.T) daemon.Info {
	t.Helper()
	token := server.NewSessionToken()
	sock := filepath.Join(testutil.ShortTempDir(t), "s.sock")
	ts := httptest.NewUnstartedServer(nil)
	bind := ts.Listener.Addr().String()
	s := server.New(server.Config{Bind: bind, Token: token, UI: fstest.MapFS{}, LoginDir: filepath.Join(t.TempDir(), "login")})
	ts.Config.Handler = s.Handler()
	ts.Start()
	t.Cleanup(ts.Close)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		_ = s.ServeUnix(ctx, sock)
		close(done)
	}()
	t.Cleanup(func() { cancel(); <-done })
	info := daemon.Info{Socket: sock, Token: token, HTTPAddr: bind}
	for i := 0; i < 100; i++ {
		if _, err := requestLoginLink(context.Background(), info); err == nil {
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
	ensureRunning = func(context.Context, string, io.Writer) (daemon.Info, error) { return info, nil }
	t.Cleanup(func() { startOpener, ensureRunning = oldStart, oldEnsure })
	return &argv
}

// checkOpenedFile asserts the opener was handed only a private redirect file
// (no code, no token on argv), that the file leads to a working login link,
// and that redeeming that link removes the file.
func checkOpenedFile(t *testing.T, info daemon.Info, argv []string) {
	t.Helper()
	line := strings.Join(argv, " ")
	if len(argv) == 0 || strings.Contains(line, "code=") || strings.Contains(line, info.Token) {
		t.Fatalf("opener argv %q: want only a file path, no code and no token", line)
	}
	file := argv[len(argv)-1]
	fi, err := os.Lstat(file)
	if err != nil || !fi.Mode().IsRegular() {
		t.Fatalf("opener target %q is not a file: %v", file, err)
	}
	if runtime.GOOS != "windows" && fi.Mode().Perm() != 0o600 {
		t.Fatalf("redirect file mode %v, want 0600", fi.Mode().Perm())
	}
	data, _ := os.ReadFile(file)
	link := regexp.MustCompile(`http://[^"'<>\s]+/login\?code=[0-9a-f]{32}`).FindString(string(data))
	if !strings.HasPrefix(link, "http://"+info.HTTPAddr+"/login?code=") {
		t.Fatalf("redirect file %q: no login link for %s", data, info.HTTPAddr)
	}
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	res, err := client.Get(link)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("redeeming the file's link: %d, want 200", res.StatusCode)
	}
	if _, err := os.Lstat(file); !os.IsNotExist(err) {
		t.Fatalf("redirect file still there after redemption: %v", err)
	}
}

// Security review: what reaches the opener's argv is the path of an
// owner-only redirect file, never the login code (another local user could
// read it from /proc and redeem it first) nor the session token.
func TestOpenWebAppHandsTheOpenerOnlyAPrivateFile(t *testing.T) {
	info := startUnixServer(t)
	argv := stubOpener(t, info, nil)

	var out strings.Builder
	if err := openWebApp(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), info.Token) || strings.Contains(out.String(), "code=") {
		t.Fatalf("output %q prints a credential although the browser opened", out.String())
	}
	checkOpenedFile(t, info, *argv)
}

// The terminal home's "open the web app" goes through the same path.
func TestTerminalHomeOpenHandsTheOpenerOnlyAPrivateFile(t *testing.T) {
	info := startUnixServer(t)
	argv := stubOpener(t, info, nil)
	if err := termHome.open(context.Background(), io.Discard); err != nil {
		t.Fatal(err)
	}
	checkOpenedFile(t, info, *argv)
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

// With no redirect file (the server could not write one), the link is
// printed, never put on an opener's command line.
func TestHandOffWithoutAFilePrintsInsteadOfOpening(t *testing.T) {
	argv := stubOpener(t, daemon.Info{}, nil)
	var out strings.Builder
	handOffLogin(&out, "127.0.0.1:8799", server.LoginLink{Code: "c", URL: loginURL("127.0.0.1:8799", "c")}, true)
	if len(*argv) != 0 || !strings.Contains(out.String(), "/login?code=c") {
		t.Fatalf("argv %q, output %q: want no opener and the link printed", *argv, out.String())
	}
}

// A second app launch finds the server already running: it opens a login
// file (or prints the link with --no-open), never the token.
func TestOpenRunningServerHandsOffALoginLink(t *testing.T) {
	info := startUnixServer(t)
	argv := stubOpener(t, info, nil)

	var out strings.Builder
	openRunningServer(context.Background(), info, false, false, &out)
	checkOpenedFile(t, info, *argv)

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

func TestCmdOpenPrintPrintsWithoutOpening(t *testing.T) {
	info := startUnixServer(t)
	argv := stubOpener(t, info, nil)
	var out strings.Builder
	if err := openWebAppWith(context.Background(), &out, false); err != nil {
		t.Fatal(err)
	}
	if len(*argv) != 0 || !strings.Contains(out.String(), "/login?code=") {
		t.Fatalf("--print: argv %q, output %q", *argv, out.String())
	}
}

func TestCmdOpenRefusesArguments(t *testing.T) {
	if code := cmdOpen([]string{"extra"}); code != exitCode("usage") {
		t.Fatalf("cmdOpen with an argument: %d, want the usage code", code)
	}
}
