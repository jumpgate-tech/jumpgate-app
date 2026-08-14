package server

import (
	"net/http"
	"runtime"
	"testing"

	"github.com/valve-tech/valve-node-app/internal/executor"
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
