package tui

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/apiclient"
	"github.com/valve-tech/jumpgate/internal/daemon"
	"github.com/valve-tech/jumpgate/internal/tui/tuitest"
)

func serverApp(t *testing.T, f *tuitest.Fake, goos string) *App {
	t.Helper()
	if _, ok := f.Routes["GET /api/update"]; !ok {
		f.Routes["GET /api/update"] = updateView{Current: "v0.6.0", Latest: "v0.6.0"}
	}
	a := dashApp(t, f, 120, 40, "ascii", "auto")
	a.o.GOOS = goos
	a.o.Server = func() ServerInfo {
		return ServerInfo{PID: 4242, HTTPAddr: "127.0.0.1:7788", StartedAt: testNow.Add(-2 * time.Hour), Shape: &daemon.Shape{}}
	}
	a.o.OpenWeb = func(context.Context) error { return nil }
	return press(t, a, "6").(*App)
}

func TestServerView(t *testing.T) {
	f := tuitest.NewFake()
	f.Routes["GET /api/update"] = updateView{Current: "v0.6.0", Latest: "v0.7.0", UpdateAvailable: true, ReleaseURL: "https://github.com/valve-tech/jumpgate/releases/tag/v0.7.0"}
	m := serverApp(t, f, "linux")
	fr := tuitest.Frame(m)
	for _, want := range []string{"Server", "v0.7.0 is out", "your own server, pid 4242, up 2h", "127.0.0.1:7788",
		"[x] stop the server", "[o] open in the browser", "[I] install the systemd service"} {
		if !strings.Contains(fr, want) {
			t.Errorf("no %q:\n%s", want, fr)
		}
	}
	if strings.Contains(fr, "[s] start the server") {
		t.Errorf("start offered while the server answers:\n%s", fr)
	}
	tuitest.Golden(t, "server_linux_120x40", fr)
	checkOnce(t, "server", m)

	_ = press(t, m, "u")
	if !f.Called("Do GET /api/update?refresh=1") {
		t.Fatalf("u did not refresh: %v", f.Calls)
	}
}

func TestServerViewOffLinux(t *testing.T) {
	f := tuitest.NewFake()
	m := serverApp(t, f, "darwin")
	got := stubShell(m)
	m.o.Self = "/usr/local/bin/jumpgate"
	fr := tuitest.Frame(press(t, m, "I", "S", "A"))
	if !strings.Contains(fr, "Linux only") || len(*got) != 0 {
		t.Fatalf("installs off Linux: ran %v\n%s", *got, fr)
	}
}

func TestServerUpdateFailAndCheckError(t *testing.T) {
	f := tuitest.NewFake()
	f.Routes["GET /api/update"] = errors.New("offline")
	if fr := tuitest.Frame(serverApp(t, f, "linux")); !strings.Contains(fr, "update     FAIL offline") {
		t.Fatalf("no FAIL:\n%s", fr)
	}
	f = tuitest.NewFake()
	f.Routes["GET /api/update"] = updateView{Current: "v0.6.0", CheckError: "rate limited\x1b[2J"}
	fr := tuitest.Frame(serverApp(t, f, "linux"))
	if !strings.Contains(fr, "could not check: rate limited") || strings.Contains(fr, "\x1b[2J") {
		t.Fatalf("check error:\n%s", fr)
	}
}

// The installs re-run this jumpgate with fixed argv in the foreground, then
// flash the exit and re-read the vault.
func TestServerInstallsRunSelf(t *testing.T) {
	f := tuitest.NewFake()
	a := serverApp(t, f, "linux")
	a.o.Self = "/usr/local/bin/jumpgate"
	var ran [][]string
	a.o.Command = func(name string, args ...string) *exec.Cmd {
		ran = append(ran, append([]string{name}, args...))
		return exec.Command("true")
	}
	m := press(t, a, "I", "S", "A")
	want := []string{"/usr/local/bin/jumpgate setup install-service", "/usr/local/bin/jumpgate setup server", "/usr/local/bin/jumpgate setup server --self-only"}
	if len(ran) != 3 {
		t.Fatalf("ran %v", ran)
	}
	for i, r := range ran {
		if strings.Join(r, " ") != want[i] {
			t.Errorf("ran %v, want %s", r, want[i])
		}
	}
	m, cmds := tuitest.Send(m, selfDoneMsg{args: []string{"setup", "server"}, err: exitErr(t, 3)})
	if fl := m.(*App).flash; fl != "jumpgate setup server: exit 3" {
		t.Errorf("flash %q", fl)
	}
	f.Calls = nil
	tuitest.Settle(m, cmds...)
	if !f.Called("Vault") {
		t.Errorf("the vault was not re-read: %v", f.Calls)
	}
}

func exitErr(t *testing.T, code int) error {
	t.Helper()
	err := exec.Command("sh", "-c", "exit 3").Run()
	var ee *exec.ExitError
	if !errors.As(err, &ee) || ee.ExitCode() != code {
		t.Fatalf("no exit %d: %v", code, err)
	}
	return err
}

func TestRunSelfWithoutAnExecutable(t *testing.T) {
	a := serverApp(t, tuitest.NewFake(), "linux")
	a.o.Self = ""
	m := press(t, a, "I")
	if fl := m.(*App).flash; !strings.Contains(fl, "run `jumpgate setup install-service` in a terminal") {
		t.Fatalf("flash %q", fl)
	}
}

func TestServerStopIsTypedConfirm(t *testing.T) {
	f := tuitest.NewFake()
	f.Routes["POST /api/shutdown"] = nil
	m := press(t, serverApp(t, f, "linux"), "x")
	if _, ok := m.(*App).modal.(*confirmModal); !ok {
		t.Fatal("x did not ask for a typed confirmation")
	}
	m = press(t, m, "y", "enter")
	if f.Called("Do POST /api/shutdown") {
		t.Fatal("y stopped the server")
	}
	m = typed(t, m, "stop") // the refusal emptied the field
	m = press(t, m, "enter")
	if !f.Called("Do POST /api/shutdown") || !strings.Contains(m.(*App).flash, "the server is stopping") {
		t.Fatalf("stop: %v flash %q", f.Calls, m.(*App).flash)
	}
}

func TestServerOpenUsesTheOpener(t *testing.T) {
	a := serverApp(t, tuitest.NewFake(), "linux")
	opened := 0
	a.o.OpenWeb = func(context.Context) error { opened++; return nil }
	m := press(t, a, "o")
	if opened != 1 || m.(*App).flash != "the web app is open in your browser" {
		t.Fatalf("opened %d flash %q", opened, m.(*App).flash)
	}
}

func TestServerStartWhenNoneAnswers(t *testing.T) {
	f := tuitest.NewFake()
	a := serverApp(t, f, "linux")
	nf := tuitest.NewFake()
	a.o.Start = func(context.Context) (Backend, error) { return nf, nil }
	m, _ := tuitest.Send(a, fleetMsg{u: apiclient.Update[api.Fleet]{State: apiclient.Retrying, Err: errors.New("dial: refused")}})
	if fr := tuitest.Frame(m); !strings.Contains(fr, "[s] start the server") {
		t.Fatalf("no start offered:\n%s", fr)
	}
	m = press(t, m, "s")
	if m.(*App).be != nf || m.(*App).flash != "the server started" {
		t.Fatalf("start: flash %q", m.(*App).flash)
	}
}

func TestShapeWarnings(t *testing.T) {
	for _, c := range []struct {
		i    ServerInfo
		want string
	}{
		{ServerInfo{HTTPAddr: "0.0.0.0:7788", Shape: &daemon.Shape{}}, "the web app listens on 0.0.0.0:7788, beyond this machine"},
		{ServerInfo{HTTPAddr: "127.0.0.1:7788"}, "does not report its options"},
		{ServerInfo{HTTPAddr: "127.0.0.1:7788", Shape: &daemon.Shape{RelayBind: "0.0.0.0:8545"}}, "relay listens on 0.0.0.0:8545"},
		{ServerInfo{HTTPAddr: "127.0.0.1:7788", Shape: &daemon.Shape{Meter: true}}, "no billing socket"},
		{ServerInfo{HTTPAddr: "[::1]:7788", Shape: &daemon.Shape{RelayBind: "localhost:8545"}}, ""},
	} {
		got := strings.Join(shapeWarnings(c.i), "; ")
		if (c.want == "" && got != "") || !strings.Contains(got, c.want) {
			t.Errorf("%+v: %q, want %q", c.i, got, c.want)
		}
	}
}

func TestFakeDoRoutes(t *testing.T) {
	f := tuitest.NewFake()
	f.Routes["GET /api/x"] = map[string]int{"n": 3}
	f.Routes["PUT /api/y"] = func(in any) (any, error) { return in, nil }
	var out struct{ N int }
	if err := f.Do(context.Background(), "GET", "/api/x", nil, &out); err != nil || out.N != 3 {
		t.Fatalf("%v %+v", err, out)
	}
	var echo map[string]string
	if err := f.Do(context.Background(), "PUT", "/api/y", map[string]string{"a": "b"}, &echo); err != nil || echo["a"] != "b" {
		t.Fatalf("%v %v", err, echo)
	}
	if err := f.Do(context.Background(), "GET", "/api/none", nil, nil); err == nil {
		t.Fatal("a missing route answered")
	}
	if !f.Called("Do PUT /api/y") {
		t.Fatalf("calls %v", f.Calls)
	}
}
