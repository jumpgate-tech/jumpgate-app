package server

import (
	"errors"
	"net/http"
	"runtime"
	"strings"
	"testing"

	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/executor"
	"github.com/valve-tech/jumpgate/internal/executor/argvfake"
)

// newDockerTestServer builds an API test server whose LOCAL executor is fake,
// so the Docker probes are scripted and never touch the real machine.
func newDockerTestServer(t *testing.T, local executor.Executor) *apiTestServer {
	t.Helper()
	return newAPITestServerCfg(t, nil, func(c *Config) {
		c.NewLocalExecutor = func() executor.Executor { return local }
	})
}

func TestDockerStatusRunning(t *testing.T) {
	// The blanket-success fake answers `command -v docker` and `docker info`
	// with exit 0 — present and reachable.
	a := newDockerTestServer(t, &autoSucceedExecutor{})

	got := decodeJSON[dockerStatusResponse](t, a.do(t, "GET", "/api/docker", nil))
	if !got.Present || !got.Running {
		t.Errorf("got %+v, want present+running", got)
	}
	if got.CanStart {
		t.Errorf("CanStart = true while running, want false")
	}
	if got.Hint != "" {
		t.Errorf("Hint = %q, want empty while running", got.Hint)
	}
}

func TestDockerStatusStopped(t *testing.T) {
	// docker present (command -v exits 0) but the daemon is down (docker info
	// exits non-zero).
	fake := (&scriptedExecutor{}).script("docker info", executor.Result{
		ExitCode: 1,
		Stderr:   "Cannot connect to the Docker daemon at unix:///var/run/docker.sock.",
	})
	a := newDockerTestServer(t, fake)

	got := decodeJSON[dockerStatusResponse](t, a.do(t, "GET", "/api/docker", nil))
	if !got.Present || got.Running {
		t.Errorf("got %+v, want present but not running", got)
	}
	if got.Hint == "" {
		t.Errorf("Hint is empty, want a start-Docker hint")
	}
	// CanStart is true only on macOS, where the app can open Docker itself.
	if want := runtime.GOOS == "darwin"; got.CanStart != want {
		t.Errorf("CanStart = %v, want %v on %s", got.CanStart, want, runtime.GOOS)
	}
}

func TestDockerStatusAbsent(t *testing.T) {
	// No docker CLI: `command -v docker` exits non-zero, which ProbeDocker
	// reports as absent.
	fake := (&scriptedExecutor{}).script("command -v docker", executor.Result{ExitCode: 1})
	a := newDockerTestServer(t, fake)

	got := decodeJSON[dockerStatusResponse](t, a.do(t, "GET", "/api/docker", nil))
	if got.Present || got.Running {
		t.Errorf("got %+v, want absent", got)
	}
	if got.CanStart {
		t.Errorf("CanStart = true while absent, want false")
	}
	if got.Hint == "" {
		t.Errorf("Hint is empty, want an install hint")
	}
}

func TestDockerStart(t *testing.T) {
	a := newDockerTestServer(t, &autoSucceedExecutor{})

	res := a.do(t, "POST", "/api/docker/start", nil)
	if runtime.GOOS == "darwin" {
		if res.StatusCode != http.StatusOK {
			t.Fatalf("POST /api/docker/start = %d, want 200 on darwin", res.StatusCode)
		}
		got := decodeJSON[struct {
			Started bool `json:"started"`
		}](t, res)
		if !got.Started {
			t.Errorf("started = false, want true (the fake launch succeeds)")
		}
	} else {
		if res.StatusCode != http.StatusBadRequest {
			t.Errorf("POST /api/docker/start = %d, want 400 off darwin", res.StatusCode)
		}
		res.Body.Close()
	}
}

func TestMacStartPlan(t *testing.T) {
	cases := []struct {
		ctx        string
		openCmd    string
		colima     bool // colima may be started at all
		colimaOnly bool // colima is the only thing to try
	}{
		{"colima", "", true, true},
		{"colima-dev", "", true, true},
		{"orbstack", "open -a OrbStack || open -a Docker", false, false},
		{"desktop-linux", "open -a Docker || open -a OrbStack", false, false},
		{"something-else", "open -a Docker || open -a OrbStack", false, false},
		{"default", "open -a Docker || open -a OrbStack", true, false},
		{"", "open -a Docker || open -a OrbStack", true, false},
	}
	for _, c := range cases {
		p := macStartPlan(c.ctx)
		if p.openCmd != c.openCmd || p.colima != c.colima || p.colimaOnly != c.colimaOnly {
			t.Errorf("context %q: got %+v", c.ctx, p)
		}
	}
	if !strings.Contains(colimaStartCommand, "command -v colima") {
		t.Error("colima must only be tried when it is on PATH")
	}
}

// A Windows controller can now probe its engine (it used to answer 500).
// An engine in Windows-container mode is up but unusable: not running, and
// with the hint, so the panel's "not running" path shows it and does not
// start provisioning (spec D35).
func TestDockerStatusWindowsContainers(t *testing.T) {
	f := argvfake.New().
		Script("docker --version", executor.Result{Stdout: "Docker version 27.4.0\n"}).
		Script("docker info --format", executor.Result{Stdout: "27.4.0|windows|x86_64|WIN-RUNNER|Microsoft Windows Server 2022 Datacenter\n"})
	a := newDockerTestServer(t, f)
	got := decodeJSON[dockerStatusResponse](t, a.do(t, "GET", "/api/docker", nil))
	if !got.Present || got.Running || !got.WindowsContainers || got.CanStart || !strings.Contains(got.Hint, "Linux containers") {
		t.Fatalf("got %+v, want present, not running, windowsContainers, the hint, no auto-start", got)
	}
}

// "This computer" can be added on any OS. On Windows it has no shell, and
// RequireShell is what the shell routes check (Task 13).
func TestDefaultExecutorBuildsALocalTargetEverywhere(t *testing.T) {
	ex, err := defaultNewExecutor(config.Target{ID: "me", Mode: "local"})
	if err != nil {
		t.Fatalf("local target refused on %s: %v", runtime.GOOS, err)
	}
	shellErr := executor.RequireShell(ex)
	if (runtime.GOOS == "windows") != errors.Is(shellErr, executor.ErrNoPOSIXShell) {
		t.Fatalf("RequireShell on %s = %v", runtime.GOOS, shellErr)
	}
}

// Windows has neither OrbStack nor colima; its stopped-engine hint names
// Docker Desktop alone.
func TestDockerStartHintPerOS(t *testing.T) {
	win := dockerStartHintFor("windows")
	if !strings.Contains(win, "Docker Desktop") || strings.Contains(win, "OrbStack") || strings.Contains(win, "colima") {
		t.Errorf("windows hint %q", win)
	}
	if mac := dockerStartHintFor("darwin"); !strings.Contains(mac, "colima") {
		t.Errorf("darwin hint %q", mac)
	}
}
