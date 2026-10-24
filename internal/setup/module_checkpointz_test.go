package setup

import (
	"context"
	"strings"
	"testing"

	"github.com/valve-tech/jumpgate/internal/catalog"
)

// Checkpointz installs its own pinned service binary, checked with --help
// (v0.32.0 has no --version), and Plan never takes it as a node client.
func TestCheckpointzPlan(t *testing.T) {
	b := catalog.BoxModule{ID: "checkpointz", ChainID: 1, Upstreams: []string{"http://127.0.0.1:5052"}}
	steps, err := PlanBoxModule(b, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	var install Step
	for _, s := range steps {
		if s.ID == "install" {
			install = s
		}
	}
	e := newFakeExecutor()
	_ = install.Verify(context.Background(), e, &State{})
	want := "test -x '/usr/local/bin/checkpointz' && '/usr/local/bin/checkpointz' --help"
	found := false
	for _, c := range e.callLog() {
		found = found || strings.Contains(c, want)
	}
	if !found {
		t.Errorf("install verify ran %v, want %q", e.callLog(), want)
	}

	w := testWire()
	w.ExecID = "checkpointz"
	if _, err := Plan(w); err == nil || !strings.Contains(err.Error(), "not a known execution client") {
		t.Errorf("Plan with checkpointz as exec: %v", err)
	}
	w = testWire()
	w.BeaconID = "checkpointz"
	if _, err := Plan(w); err == nil || !strings.Contains(err.Error(), "not a known beacon client") {
		t.Errorf("Plan with checkpointz as beacon: %v", err)
	}
}
