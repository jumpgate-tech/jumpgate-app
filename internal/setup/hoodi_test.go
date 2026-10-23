package setup

import (
	"context"
	"testing"

	"github.com/valve-tech/jumpgate/internal/catalog"
	"github.com/valve-tech/jumpgate/internal/executor"
)

func TestPreflightSkipsDiskFloorWithoutGuidance(t *testing.T) {
	e := rootUIDScript(newFakeExecutor()).
		script("uname", executor.Result{Stdout: "Linux\n"}).
		script("cat /proc/version", executor.Result{Stdout: "Linux version 6.8.0-ubuntu\n"}).
		script("hostname", executor.Result{Stdout: "box-a\n"}).
		script("df -B1 --output=avail", executor.Result{Stdout: "Avail\n1\n"}).
		script("ss -ltn", executor.Result{Stdout: "State  Recv-Q Send-Q Local Address:Port\n"})
	w := catalog.WireConfig{ChainID: 560048, ExecID: "reth", BeaconID: "lighthouse", DataDir: "/mnt/reth"}
	events := make(chan Event, 16)
	if err := preflightStep().Verify(context.Background(), e, &State{Wire: w, Events: events}); err != nil {
		t.Fatalf("preflight without size guidance must pass, got %v", err)
	}
	close(events)
	want := "No disk size guidance for Hoodi yet; check the disk has room for a full node."
	var lines []string
	for ev := range events {
		if ev.Line == want {
			return
		}
		lines = append(lines, ev.Line)
	}
	t.Errorf("want event line %q, got %q", want, lines)
}
