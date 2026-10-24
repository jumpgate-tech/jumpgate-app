package monitor

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/valve-tech/jumpgate/internal/catalog"
	"github.com/valve-tech/jumpgate/internal/executor"
)

type catExec struct {
	executor.Executor
	cmd, out string
}

func (c *catExec) Run(_ context.Context, cmd string, _ *executor.RunOpts) (executor.Result, error) {
	c.cmd = cmd
	return executor.Result{Stdout: c.out}, nil
}

func TestWatchProbeStale(t *testing.T) {
	now := time.Date(2026, 10, 23, 12, 0, 0, 0, time.UTC)
	watchNow = func() time.Time { return now }
	defer func() { watchNow = time.Now }()
	b := catalog.BoxModule{ID: "watch"}
	state := func(at time.Time) string {
		ts := at.Format(time.RFC3339)
		return fmt.Sprintf(`{"at":%q,"open":[{"key":"exec-down","message":"m","since":%q}],"recent":[]}`, ts, ts)
	}
	ex := &catExec{out: state(now.Add(-10*time.Minute - 20*time.Second))}
	st := boxProbes["watch"](context.Background(), ex, b)
	if st.Error != "The health watch has not written a reading for 10 minutes." || st.Served {
		t.Errorf("stale: %+v", st)
	}
	if ex.cmd != "cat '/var/lib/jumpgate-node/modules/watch/state.json'" {
		t.Errorf("cmd = %s", ex.cmd)
	}
	ex.out = state(now.Add(-time.Minute))
	st = boxProbes["watch"](context.Background(), ex, b)
	if !st.Served || st.Error != "" || len(st.Alerts) != 1 || st.Alerts[0].Key != "exec-down" {
		t.Errorf("fresh: %+v", st)
	}
}
