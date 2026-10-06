package server

import (
	"testing"
	"time"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/monitor"
	"github.com/valve-tech/jumpgate/internal/ops"
)

func TestNodeStatusView(t *testing.T) {
	at := time.Unix(1000, 0).UTC()
	for name, c := range map[string]struct {
		snap    monitor.Snapshot
		overall api.SyncState
		exec    api.SyncState
		lag     *uint64
	}{
		"both stopped":   {monitor.Snapshot{At: at}, api.SyncStopped, api.SyncStopped, nil},
		"exec syncing":   {monitor.Snapshot{At: at, ExecActive: true, BeaconActive: true, ExecSyncing: true, ExecHead: 10, BeaconSlot: 5, RefHead: 100}, api.SyncSyncing, api.SyncSyncing, ptr(uint64(90))},
		"beacon behind":  {monitor.Snapshot{At: at, ExecActive: true, BeaconActive: true, ExecHead: 10, BeaconSlot: 5, BeaconDistance: 3}, api.SyncSyncing, api.SyncSynced, nil},
		"synced":         {monitor.Snapshot{At: at, ExecActive: true, BeaconActive: true, ExecHead: 100, BeaconSlot: 5, RefHead: 99}, api.SyncSynced, api.SyncSynced, ptr(uint64(0))},
		"no data yet":    {monitor.Snapshot{At: at, ExecActive: true, BeaconActive: true}, api.SyncNoData, api.SyncNoData, nil},
		"only exec down": {monitor.Snapshot{At: at, BeaconActive: true, BeaconSlot: 5}, api.SyncStopped, api.SyncStopped, nil},
	} {
		v := nodeStatusView(c.snap)
		if v.Overall != c.overall || v.Exec.State != c.exec || !v.At.Equal(at) {
			t.Errorf("%s: overall %s exec %s", name, v.Overall, v.Exec.State)
		}
		if (v.HeadLag == nil) != (c.lag == nil) || (v.HeadLag != nil && *v.HeadLag != *c.lag) {
			t.Errorf("%s: head lag %v, want %v", name, v.HeadLag, c.lag)
		}
	}
	if v := nodeStatusView(monitor.Snapshot{DiskKnown: false, DiskUsedPct: 0}); v.DiskUsedPct != nil {
		t.Error("an unknown disk reading must be absent, never 0%")
	}
	if v := nodeStatusView(monitor.Snapshot{DiskKnown: true, DiskUsedPct: 42.5}); v.DiskUsedPct == nil || *v.DiskUsedPct != 42.5 {
		t.Error("a known disk reading is lost")
	}
}

func ptr[T any](v T) *T { return &v }

func TestDiskFit(t *testing.T) {
	const tb = 1_000_000_000_000
	for name, c := range map[string]struct {
		du   ops.DU
		want api.FitVerdict
	}{
		"room to spare": {ops.DU{ExecBytes: 1 * tb, DiskFreeBytes: 2 * tb, ExpectedExecBytes: 2 * tb, ExpectedBeaconBytes: 0}, api.FitOK},
		"tight":         {ops.DU{ExecBytes: 1 * tb, DiskFreeBytes: 1.05 * tb, ExpectedExecBytes: 2 * tb}, api.FitTight},
		"short":         {ops.DU{ExecBytes: 1 * tb, DiskFreeBytes: 0.5 * tb, ExpectedExecBytes: 2 * tb}, api.FitShort},
		"no estimate":   {ops.DU{ExecBytes: 1 * tb, DiskFreeBytes: 0.5 * tb}, api.FitUnknown},
	} {
		if got := diskView(time.Now(), c.du, nil); got.Fit != c.want || got.ExpectedLabel != "estimate" {
			t.Errorf("%s: fit %s label %q", name, got.Fit, got.ExpectedLabel)
		}
	}
}

func TestDiskTrendNeedsSixHours(t *testing.T) {
	t0 := time.Unix(0, 0)
	short := []api.DiskSample{{At: t0, UsedBytes: 100, FreeBytes: 1000}, {At: t0.Add(5 * time.Hour), UsedBytes: 200, FreeBytes: 900}}
	if g, d := diskTrend(short); g != nil || d != nil {
		t.Fatal("a trend from five hours of samples")
	}
	day := []api.DiskSample{{At: t0, UsedBytes: 100, FreeBytes: 1000}, {At: t0.Add(12 * time.Hour), UsedBytes: 150, FreeBytes: 950}, {At: t0.Add(24 * time.Hour), UsedBytes: 200, FreeBytes: 900}}
	g, d := diskTrend(day)
	if g == nil || *g != 100 || d == nil || *d != 9 {
		t.Fatalf("growth %v days %v", g, d)
	}
	shrinking := []api.DiskSample{{At: t0, UsedBytes: 200, FreeBytes: 900}, {At: t0.Add(24 * time.Hour), UsedBytes: 100, FreeBytes: 1000}}
	if g, d := diskTrend(shrinking); g == nil || *g != -100 || d != nil {
		t.Fatalf("shrinking: growth %v days %v (no days-to-full when shrinking)", g, d)
	}
}

func TestFirewallSummary(t *testing.T) {
	items := []ops.CheckItem{{Status: "pass"}, {Status: "warn"}, {Status: "pass"}}
	if s := firewallSummary(items); s.Grade != "warn" || s.Pass != 2 || s.Warn != 1 {
		t.Fatalf("%+v", s)
	}
	if s := firewallSummary(append(items, ops.CheckItem{Status: "fail"})); s.Grade != "fail" {
		t.Fatalf("%+v", s)
	}
	if s := firewallSummary([]ops.CheckItem{{Status: "unknown"}}); s.Grade != "unknown" {
		t.Fatalf("%+v", s)
	}
	if s := firewallSummary([]ops.CheckItem{{Status: "pass"}}); s.Grade != "ok" {
		t.Fatalf("%+v", s)
	}
}
