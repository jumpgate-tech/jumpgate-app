package server

import (
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/valve-tech/valve-node-app/internal/catalog"
	"github.com/valve-tech/valve-node-app/internal/config"
)

// rewire re-runs the node wizard on "local" with wire and waits for the run
// to finish.
func rewire(t *testing.T, a *apiTestServer, wire catalog.WireConfig) {
	t.Helper()
	res := a.do(t, "POST", "/api/targets/local/setup", wire)
	if res.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(res.Body)
		t.Fatalf("setup kickoff status = %d, want 202, body=%s", res.StatusCode, body)
	}
	res.Body.Close()
	select {
	case <-setupRunOf(a.srv, "local").done:
	case <-time.After(10 * time.Second):
		t.Fatal("the setup run against a fake executor never finished")
	}
}

// localTarget reads "local" back from the saved config, Wire and all.
func localTarget(t *testing.T, a *apiTestServer) (config.Target, string) {
	t.Helper()
	cfg, err := a.srv.loadConfig()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	tgt, ok := findTarget(cfg, "local")
	if !ok {
		t.Fatal("target local is not in the config")
	}
	return tgt, cfg.RefRPCBase
}

// A retired monitor or watcher stops publishing but never closes its
// subscribers' channels, so a stream still attached to one would stay open
// and silent forever. The stream has to end instead, which an EventSource
// answers by reconnecting, and the reconnect gets the rebuilt observer.
func TestObservers_OpenStreamsEndWhenSetupIsRerun(t *testing.T) {
	for _, path := range []string{"/api/targets/local/monitor/stream", "/api/targets/local/logs/stream"} {
		t.Run(path, func(t *testing.T) {
			a := newAPITestServer(t)
			addAndWireLocalTarget(t, a)

			res := a.do(t, "GET", path, nil)
			defer res.Body.Close()
			if res.StatusCode != http.StatusOK {
				t.Fatalf("got %d, want 200", res.StatusCode)
			}
			// The handler has subscribed once its headers are flushed.

			ended := make(chan struct{})
			go func() {
				io.Copy(io.Discard, res.Body)
				close(ended)
			}()

			rewire(t, a, catalog.WireConfig{
				ChainID: 369, ExecID: "reth", BeaconID: "lighthouse-pulse", DataDir: "/srv/elsewhere/369",
			})

			select {
			case <-ended:
			case <-time.After(5 * time.Second):
				t.Fatal("the stream is still open on the observer that setup retired")
			}
		})
	}
}

// The monitor and log watcher are built from the target's Wire the first
// time anything asks for them, then cached. Re-running setup with a new Wire
// (another data directory, another client) must retire them, or the node
// screen, log tail and auto-diagnostics keep watching the node as it was.
func TestObservers_RebuiltAfterSetupIsRerun(t *testing.T) {
	a := newAPITestServer(t)
	addAndWireLocalTarget(t, a)

	tgt, ref := localTarget(t, a)
	mon1, monRetired, err := a.srv.getMonitor(tgt, ref)
	if err != nil {
		t.Fatalf("getMonitor: %v", err)
	}
	watch1, watchRetired, err := a.srv.getWatcher(tgt)
	if err != nil {
		t.Fatalf("getWatcher: %v", err)
	}

	rewire(t, a, catalog.WireConfig{
		ChainID: 369, ExecID: "reth", BeaconID: "lighthouse-pulse", DataDir: "/srv/elsewhere/369",
	})

	// Retiring stops the old observers, and the diagnostics goroutines that
	// share their contexts, rather than only forgetting them.
	for name, retired := range map[string]<-chan struct{}{"monitor": monRetired, "watcher": watchRetired} {
		select {
		case <-retired:
		default:
			t.Errorf("the old %s is still running after setup was re-run", name)
		}
	}

	tgt, ref = localTarget(t, a)
	if tgt.Wire.DataDir != "/srv/elsewhere/369" {
		t.Fatalf("the new Wire was not saved: %+v", tgt.Wire)
	}
	mon2, _, err := a.srv.getMonitor(tgt, ref)
	if err != nil {
		t.Fatalf("getMonitor after re-setup: %v", err)
	}
	watch2, _, err := a.srv.getWatcher(tgt)
	if err != nil {
		t.Fatalf("getWatcher after re-setup: %v", err)
	}
	if mon2 == mon1 {
		t.Error("the monitor built from the old Wire is still the one served after setup was re-run")
	}
	if watch2 == watch1 {
		t.Error("the log watcher built from the old Wire is still the one served after setup was re-run")
	}
}
