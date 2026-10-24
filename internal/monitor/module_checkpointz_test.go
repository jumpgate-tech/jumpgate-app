package monitor

import (
	"context"
	"strings"
	"testing"

	"github.com/valve-tech/jumpgate/internal/catalog"
	"github.com/valve-tech/jumpgate/internal/executor"
)

const checkpointzStatus = `{"upstreams":{"beacon-1":{"name":"beacon-1","healthy":true,"finality":{"finalized":{"epoch":"412345","root":"0xab"}}}},` +
	`"finality":{"finalized":{"epoch":"412345","root":"0xab"},"current_justified":{"epoch":"412346","root":"0xcd"}},` +
	`"version":{"short":"v0.32.0"},"operating_mode":"full"}`

func TestCheckpointzProbe(t *testing.T) {
	ex := newFakeExecutor().
		script("curl", executor.Result{Stdout: checkpointzStatus}).
		script("is-active", executor.Result{Stdout: "active\n"})
	st := ProbeBoxModule(context.Background(), ex, catalog.BoxModule{ID: "checkpointz", ChainID: 1})
	if !st.Served || st.HeadBlock != 412345 || !st.Active || st.Error != "" || st.Note != "" {
		t.Fatalf("status %+v", st)
	}
	want := "curl -fsS --max-time 5 'http://127.0.0.1:5555/checkpointz/v1/status'"
	found := false
	for _, c := range ex.calls {
		found = found || c == want
	}
	if !found {
		t.Fatalf("calls %v, want %q", ex.calls, want)
	}

	// A private bind is probed there.
	ex = newFakeExecutor().script("curl", executor.Result{Stdout: checkpointzStatus})
	ProbeBoxModule(context.Background(), ex, catalog.BoxModule{ID: "checkpointz", ChainID: 1, BindAddr: "10.0.0.5", Port: 5600})
	if !strings.Contains(strings.Join(ex.calls, "\n"), "'http://10.0.0.5:5600/checkpointz/v1/status'") {
		t.Fatalf("calls %v", ex.calls)
	}
}

func TestCheckpointzProbeNoFinality(t *testing.T) {
	ex := newFakeExecutor().script("curl", executor.Result{Stdout: `{"upstreams":{"beacon-1":{"healthy":false}},"finality":null}`})
	st := ProbeBoxModule(context.Background(), ex, catalog.BoxModule{ID: "checkpointz", ChainID: 1})
	if st.Served || st.Note == "" || st.Error != "" {
		t.Fatalf("status %+v", st)
	}
	ex = newFakeExecutor().script("curl", executor.Result{ExitCode: 7})
	st = ProbeBoxModule(context.Background(), ex, catalog.BoxModule{ID: "checkpointz", ChainID: 1})
	if st.Served || st.Error == "" {
		t.Fatalf("status %+v", st)
	}
}
