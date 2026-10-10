// internal/server/migratedbox_test.go
package server

import (
	"context"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/catalog"
	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/executor"
)

// markerExec is autoSucceedExecutor whose names-marker probe answers as a
// box with (exit 0) or without (exit 1) /etc/jumpgate/names.json.
type markerExec struct {
	autoSucceedExecutor
	marker bool
	mu2    sync.Mutex
	probes int
}

func (m *markerExec) Run(ctx context.Context, cmd string, o *executor.RunOpts) (executor.Result, error) {
	if cmd == "test -e '/etc/jumpgate/names.json'" {
		m.mu2.Lock()
		m.probes++
		m.mu2.Unlock()
		if m.marker {
			return executor.Result{}, nil
		}
		return executor.Result{ExitCode: 1}, nil
	}
	return m.autoSucceedExecutor.Run(ctx, cmd, o)
}
func (m *markerExec) WriteFile(context.Context, string, []byte, os.FileMode) error { return nil }
func (m *markerExec) ReadFile(context.Context, string) ([]byte, error)             { return nil, nil }
func (m *markerExec) Close() error                                                 { return nil }

// I5: a controller whose stored wire predates a migration it did not see
// would set the box up under the legacy names again, undoing it. The box's
// names marker says it was migrated; setup that would render legacy names
// refuses such a box, SSH-only targets included.
func TestSetupRefusesLegacyNamesOnAMigratedBox(t *testing.T) {
	for _, marker := range []bool{true, false} {
		ex := &markerExec{marker: marker}
		a := newAPITestServerWithExecutor(t, func(config.Target) (executor.Executor, error) { return ex, nil })
		res := a.do(t, "POST", "/api/targets", config.Target{ID: "local", Mode: "local"})
		res.Body.Close()
		if _, err := config.Update(func(c *config.Config) error {
			c.Targets[0].Wire = &catalog.WireConfig{ChainID: 369, ExecID: "reth", BeaconID: "lighthouse-pulse",
				DataDir: "/var/lib/valve-node-app/369", JWTPath: "/var/lib/valve-node-app/369/jwt.hex"}
			return nil
		}); err != nil {
			t.Fatal(err)
		}
		res = a.do(t, "POST", "/api/targets/local/setup", catalog.WireConfig{ChainID: 369, ExecID: "reth", BeaconID: "lighthouse-pulse"})
		body, _ := io.ReadAll(res.Body)
		res.Body.Close()
		if ex.probes != 1 {
			t.Fatalf("marker %v: %d probes of the names marker", marker, ex.probes)
		}
		if !marker {
			if res.StatusCode != http.StatusAccepted {
				t.Fatalf("an unmigrated legacy box: %d %s", res.StatusCode, body)
			}
			continue
		}
		if res.StatusCode != http.StatusConflict || !strings.Contains(string(body), string(api.CodeConflict)) ||
			!strings.Contains(string(body), "names.json") || !strings.Contains(string(body), "remove the host") {
			t.Fatalf("a migrated box set up under legacy names: %d %s", res.StatusCode, body)
		}
	}
}

// The names-marker probe runs only once setup holds the target: probed
// earlier, a migrate job ending between the probe and the claim could let a
// legacy-names setup through. A refusal by the probe releases the claim, so
// the next attempt is told about the marker again, not that setup is running.
func TestSetupProbesTheNamesMarkerOnlyOnceItHoldsTheTarget(t *testing.T) {
	ex := &markerExec{marker: true}
	a := newAPITestServerWithExecutor(t, func(config.Target) (executor.Executor, error) { return ex, nil })
	res := a.do(t, "POST", "/api/targets", config.Target{ID: "local", Mode: "local"})
	res.Body.Close()
	if _, err := config.Update(func(c *config.Config) error {
		c.Targets[0].Wire = &catalog.WireConfig{ChainID: 369, ExecID: "reth", BeaconID: "lighthouse-pulse",
			DataDir: "/var/lib/valve-node-app/369", JWTPath: "/var/lib/valve-node-app/369/jwt.hex"}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	wire := catalog.WireConfig{ChainID: 369, ExecID: "reth", BeaconID: "lighthouse-pulse"}

	entry := a.srv.reg.get("local")
	entry.mu.Lock()
	entry.opBusy = true
	entry.mu.Unlock()
	res = a.do(t, "POST", "/api/targets/local/setup", wire)
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != http.StatusConflict || !strings.Contains(string(body), "a wipe, reset or clear is running") || ex.probes != 0 {
		t.Fatalf("while another op holds the target: %d %s, %d probes", res.StatusCode, body, ex.probes)
	}
	entry.mu.Lock()
	entry.opBusy = false
	entry.mu.Unlock()

	for i := 1; i <= 2; i++ {
		res = a.do(t, "POST", "/api/targets/local/setup", wire)
		body, _ = io.ReadAll(res.Body)
		res.Body.Close()
		if res.StatusCode != http.StatusConflict || !strings.Contains(string(body), "names.json") || ex.probes != i {
			t.Fatalf("attempt %d on a migrated box: %d %s, %d probes", i, res.StatusCode, body, ex.probes)
		}
	}
}
