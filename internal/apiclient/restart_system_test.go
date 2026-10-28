package apiclient

import (
	"context"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/valve-tech/jumpgate/internal/daemon"
)

// The jumpgate service is restarted by systemd, not by this client: Restart
// asks it to exit and waits for systemd to bring up a new one.
func TestRestartTheServiceWaitsForANewPID(t *testing.T) {
	var mu sync.Mutex
	var posted bool
	c := serve(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		if r.Method == http.MethodPost && r.URL.Path == "/api/restart" {
			posted = true
			w.WriteHeader(http.StatusAccepted)
			return
		}
		http.NotFound(w, r)
	})
	c.info.PID, c.info.System, c.info.Socket = 10, true, "/run/jumpgate-ctl/server.sock"
	oldF, oldS, oldW := daemonFind, daemonStop, restartWait
	t.Cleanup(func() { daemonFind, daemonStop, restartWait = oldF, oldS, oldW })
	restartWait = 5 * time.Second
	calls := 0
	daemonFind = func(context.Context) (daemon.Info, bool, error) {
		calls++
		pid := 10
		if calls > 2 {
			pid = 11
		}
		return daemon.Info{PID: pid, System: true, Socket: "/run/jumpgate-ctl/server.sock"}, true, nil
	}
	daemonStop = func(context.Context) error { t.Fatal("daemon.Stop on the service"); return nil }
	nc, err := Restart(context.Background(), c)
	if err != nil {
		t.Fatal(err)
	}
	if nc.Info().PID != 11 || !nc.Info().System {
		t.Fatalf("new client %+v", nc.Info())
	}
	mu.Lock()
	defer mu.Unlock()
	if !posted {
		t.Fatal("no POST /api/restart")
	}
}
