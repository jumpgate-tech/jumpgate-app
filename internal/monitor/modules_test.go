package monitor

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/valve-tech/jumpgate/internal/catalog"
	"github.com/valve-tech/jumpgate/internal/executor"
)

// withNodeModules stands in a module registry for one test.
func withNodeModules(t *testing.T, ms []catalog.Module) {
	t.Helper()
	old := nodeModules
	nodeModules = func() []catalog.Module { return ms }
	t.Cleanup(func() { nodeModules = old })
}

func fakeBoard() catalog.Module {
	return catalog.Module{
		ID: "msgboard", Placement: catalog.PlaceNode, Chains: []int{369}, Clients: []string{"reth"},
		Probe: catalog.ModuleProbe{StatusMethod: "msgboard_status"},
	}
}

// Only a module that applies to the wire is probed: a go-pulse node costs no
// extra call, a reth node on 369 costs one, to the module's status method.
func TestProbeModulesOnlyApplicable(t *testing.T) {
	withNodeModules(t, []catalog.Module{fakeBoard()})
	old, had := nodeDecoders["msgboard"]
	nodeDecoders["msgboard"] = func(body string) (ModuleStatus, error) {
		return ModuleStatus{Served: true, Count: 7}, nil
	}
	t.Cleanup(func() {
		if had {
			nodeDecoders["msgboard"] = old
		} else {
			delete(nodeDecoders, "msgboard")
		}
	})

	ex := newFakeExecutor()
	got := probeModules(context.Background(), ex, catalog.WireConfig{ChainID: 369, ExecID: "go-pulse"})
	if len(got) != 0 || len(ex.calls) != 0 {
		t.Fatalf("go-pulse: %v, calls %v", got, ex.calls)
	}

	ex = newFakeExecutor().script("msgboard_status", executor.Result{Stdout: `{"result":{}}`})
	got = probeModules(context.Background(), ex, catalog.WireConfig{ChainID: 369, ExecID: "reth"})
	if len(ex.calls) != 1 || !strings.Contains(ex.calls[0], `"method":"msgboard_status"`) {
		t.Fatalf("calls %v", ex.calls)
	}
	if len(got) != 1 || got[0].ID != "msgboard" || !got[0].Served || got[0].Count != 7 {
		t.Fatalf("statuses %+v", got)
	}
}

// A node that does not know the method is not serving the module; that is a
// reading, not an error.
func TestProbeModulesMethodNotFound(t *testing.T) {
	withNodeModules(t, []catalog.Module{fakeBoard()})
	ex := newFakeExecutor().script("msgboard_status", executor.Result{
		Stdout: `{"jsonrpc":"2.0","id":1,"error":{"code":-32601,"message":"the method msgboard_status does not exist"}}`,
	})
	got := probeModules(context.Background(), ex, catalog.WireConfig{ChainID: 369, ExecID: "reth"})
	if len(got) != 1 || got[0].Served || got[0].Error != "" {
		t.Fatalf("statuses %+v", got)
	}
}

// The box's sidecars say which modules it runs; is-active says which are up.
func TestProbeBoxModulesReadsSidecars(t *testing.T) {
	a := catalog.BoxModule{ID: "el-bootnode", ChainID: 369}
	b := catalog.BoxModule{ID: "watch"}
	ja, _ := json.Marshal(a)
	jb, _ := json.Marshal(b)
	acceptBoxModules(t, "el-bootnode", "watch")
	ex := newFakeExecutor().
		script("ls ", executor.Result{Stdout: a.SidecarPath() + "\n" + b.SidecarPath() + "\n"}).
		script("cat '"+a.SidecarPath()+"'", executor.Result{Stdout: string(ja)}).
		script("cat '"+b.SidecarPath()+"'", executor.Result{Stdout: string(jb)}).
		script("is-active '"+a.UnitName()+"'", executor.Result{Stdout: "active\n"}).
		script("is-active '"+b.UnitName()+"'", executor.Result{Stdout: "inactive\n", ExitCode: 3})
	got := ProbeBoxModules(context.Background(), ex)
	if len(got) != 2 {
		t.Fatalf("statuses %+v (calls %v)", got, ex.calls)
	}
	if got[0].ID != "el-bootnode" || got[0].Instance != "el-bootnode-369" || !got[0].Active {
		t.Errorf("first %+v", got[0])
	}
	if got[1].ID != "watch" || got[1].Instance != "watch" || got[1].Active {
		t.Errorf("second %+v", got[1])
	}
}

// acceptBoxModules makes ids the only valid box modules for one test (the
// catalog registers none yet).
func acceptBoxModules(t *testing.T, ids ...string) {
	t.Helper()
	old := validateBoxModule
	validateBoxModule = func(b catalog.BoxModule) error {
		for _, id := range ids {
			if b.ID == id {
				return nil
			}
		}
		return errors.New(b.ID + " is not a known module")
	}
	t.Cleanup(func() { validateBoxModule = old })
}

// A sidecar is data the box hands back. One naming an unknown module, or
// sitting in another module's dir, is reported as an error and nothing is
// built from it.
func TestProbeBoxModulesRejectsHostileSidecar(t *testing.T) {
	acceptBoxModules(t, "watch")
	hostile := `{"id":"x; touch /tmp/pwned"}`
	moved := catalog.BoxModule{ID: "watch"}
	jm, _ := json.Marshal(moved)
	const p1 = "/var/lib/jumpgate/modules/evil/module.json"
	const p2 = "/var/lib/jumpgate/modules/other/module.json"
	ex := newFakeExecutor().
		script("ls ", executor.Result{Stdout: p1 + "\n" + p2 + "\n"}).
		script("cat '"+p1+"'", executor.Result{Stdout: hostile}).
		script("cat '"+p2+"'", executor.Result{Stdout: string(jm)})
	got := ProbeBoxModules(context.Background(), ex)
	if len(got) != 2 || got[0].Error == "" || got[1].Error == "" || got[0].Active || got[1].Active {
		t.Fatalf("statuses %+v", got)
	}
	for _, c := range ex.calls {
		if strings.Contains(c, "pwned") || strings.Contains(c, "systemctl") {
			t.Errorf("a hostile sidecar reached a command: %q", c)
		}
	}
}

// ProbeBoxModule validates its record too, and quotes the unit it asks for.
func TestProbeBoxModuleValidatesAndQuotes(t *testing.T) {
	acceptBoxModules(t, "watch")
	ex := newFakeExecutor()
	if st := ProbeBoxModule(context.Background(), ex, catalog.BoxModule{ID: "x; touch /tmp/pwned"}); st.Error == "" || len(ex.calls) != 0 {
		t.Fatalf("status %+v, calls %v", st, ex.calls)
	}
	ProbeBoxModule(context.Background(), ex, catalog.BoxModule{ID: "watch"})
	if len(ex.calls) != 1 || ex.calls[0] != "systemctl is-active 'jumpgate-mod-watch.service'" {
		t.Fatalf("calls %v", ex.calls)
	}
}

// A box with no modules directory has no modules, and that is not an error.
func TestProbeBoxModulesNoDir(t *testing.T) {
	ex := newFakeExecutor().script("ls ", executor.Result{ExitCode: 2, Stderr: "No such file"})
	if got := ProbeBoxModules(context.Background(), ex); got != nil {
		t.Fatalf("got %+v", got)
	}
	ex = newFakeExecutor().errOn("ls ", errors.New("ssh: gone"))
	if got := ProbeBoxModules(context.Background(), ex); got != nil {
		t.Fatalf("transport error: got %+v", got)
	}
}

func TestSnapshotOmitsModulesWhenEmpty(t *testing.T) {
	b, err := json.Marshal(Snapshot{})
	if err != nil || strings.Contains(string(b), `"modules"`) {
		t.Fatalf("%s %v", b, err)
	}
}
