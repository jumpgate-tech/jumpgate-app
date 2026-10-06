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
		"room to spare": {ops.DU{ExecBytes: 1 * tb, DiskFreeKnown: true, DiskFreeBytes: 2 * tb, ExpectedExecBytes: 2 * tb, ExpectedBeaconBytes: 0}, api.FitOK},
		"tight":         {ops.DU{ExecBytes: 1 * tb, DiskFreeKnown: true, DiskFreeBytes: 1.05 * tb, ExpectedExecBytes: 2 * tb}, api.FitTight},
		"short":         {ops.DU{ExecBytes: 1 * tb, DiskFreeKnown: true, DiskFreeBytes: 0.5 * tb, ExpectedExecBytes: 2 * tb}, api.FitShort},
		"no estimate":   {ops.DU{ExecBytes: 1 * tb, DiskFreeKnown: true, DiskFreeBytes: 0.5 * tb}, api.FitUnknown},
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

func TestNodeStatusViewBoundaries(t *testing.T) {
	at := time.Unix(1000, 0).UTC()
	healthy := monitor.Snapshot{At: at, ExecActive: true, ExecHead: 100, RefHead: 100}
	t.Run("beacon stopped, exec healthy", func(t *testing.T) {
		if v := nodeStatusView(healthy); v.Overall != api.SyncStopped || v.Beacon.State != api.SyncStopped {
			t.Errorf("overall %s beacon %s", v.Overall, v.Beacon.State)
		}
	})
	t.Run("beacon no data, exec healthy", func(t *testing.T) {
		s := healthy
		s.BeaconActive = true
		if v := nodeStatusView(s); v.Overall != api.SyncNoData || v.Beacon.State != api.SyncNoData {
			t.Errorf("overall %s beacon %s", v.Overall, v.Beacon.State)
		}
	})
	t.Run("beacon distance of exactly one slot is syncing", func(t *testing.T) {
		s := healthy
		s.BeaconActive, s.BeaconSlot, s.BeaconDistance = true, 5, 1
		if v := nodeStatusView(s); v.Overall != api.SyncSyncing || !v.Beacon.Syncing || v.Beacon.Distance != 1 {
			t.Errorf("overall %s beacon %+v", v.Overall, v.Beacon)
		}
	})
	t.Run("unknown reference head is no lag, not a huge lag", func(t *testing.T) {
		s := healthy
		s.RefHead = 0
		if v := nodeStatusView(s); v.HeadLag != nil {
			t.Errorf("head lag %d with no reference head", *v.HeadLag)
		}
	})
	t.Run("exec ahead of the reference is lag zero", func(t *testing.T) {
		s := healthy
		s.ExecHead, s.RefHead = 101, 100
		if v := nodeStatusView(s); v.HeadLag == nil || *v.HeadLag != 0 {
			t.Errorf("head lag %v", v.HeadLag)
		}
	})
}

func TestDiskFitBoundaries(t *testing.T) {
	// used 100 + free 100 = room 200; expected 200 sits exactly on the
	// "fits" boundary, expected 200/1.10 exactly on the headroom boundary.
	base := ops.DU{ExecBytes: 60, BeaconBytes: 40, DiskFreeBytes: 100, DiskFreeKnown: true}
	at := func(execExp, beaconExp uint64) api.FitVerdict {
		du := base
		du.ExpectedExecBytes, du.ExpectedBeaconBytes = execExp, beaconExp
		return diskView(time.Now(), du, nil).Fit
	}
	if got := at(200, 0); got != api.FitTight {
		t.Errorf("room == expected: %s, want tight", got)
	}
	if got := at(201, 0); got != api.FitShort {
		t.Errorf("room just under expected: %s, want short", got)
	}
	// room 220 == expected 200 * 1.10
	du := ops.DU{ExecBytes: 60, BeaconBytes: 40, DiskFreeBytes: 120, DiskFreeKnown: true, ExpectedExecBytes: 200}
	if got := diskView(time.Now(), du, nil).Fit; got != api.FitOK {
		t.Errorf("room == expected*margin: %s, want ok", got)
	}
	du.DiskFreeBytes = 119
	if got := diskView(time.Now(), du, nil).Fit; got != api.FitTight {
		t.Errorf("room just under expected*margin: %s, want tight", got)
	}
	// the beacon estimate counts: 150 + 150 = 300 expected, room 200 is short.
	if got := at(150, 150); got != api.FitShort {
		t.Errorf("beacon estimate ignored: %s, want short", got)
	}
	if got := at(0, 150); got != api.FitOK {
		t.Errorf("beacon-only estimate: %s, want ok", got)
	}
}

func TestDiskFitUnknownWhenFreeSpaceUnread(t *testing.T) {
	du := ops.DU{ExecBytes: 100, ExpectedExecBytes: 50} // probe failed: free 0, not known
	if got := diskView(time.Now(), du, nil); got.Fit != api.FitUnknown {
		t.Errorf("an unread disk reads %s, never ok", got.Fit)
	}
}

func TestDiskTrendBoundaries(t *testing.T) {
	t0 := time.Unix(0, 0)
	exact := []api.DiskSample{{At: t0, UsedBytes: 0, FreeBytes: 1000}, {At: t0.Add(trendMinSpan), UsedBytes: 25, FreeBytes: 975}}
	if g, d := diskTrend(exact); g == nil || *g != 100 || d == nil || *d != 9.75 {
		t.Errorf("exactly trendMinSpan: growth %v days %v", g, d)
	}
	flat := []api.DiskSample{{At: t0, UsedBytes: 100, FreeBytes: 900}, {At: t0.Add(24 * time.Hour), UsedBytes: 100, FreeBytes: 900}}
	if g, d := diskTrend(flat); g == nil || *g != 0 || d != nil {
		t.Errorf("zero growth: growth %v days %v (no days-to-full)", g, d)
	}
	if g, d := diskTrend(nil); g != nil || d != nil {
		t.Error("no history has no trend")
	}
}

func TestDiskTrendIgnoresOneOutlier(t *testing.T) {
	t0 := time.Unix(0, 0)
	var h []api.DiskSample
	for i := 0; i <= 8; i++ { // 100 bytes/day for 4 days, sampled every 12h
		h = append(h, api.DiskSample{At: t0.Add(time.Duration(i) * 12 * time.Hour), UsedBytes: uint64(1000 + 50*i), FreeBytes: 5000})
	}
	h[8].UsedBytes = 100000 // one bad reading at the end (a snapshot import)
	if g, _ := diskTrend(h); g == nil || *g != 100 {
		t.Errorf("growth %v with one outlier, want 100", orNil(g))
	}
}

func TestFirewallSummaryMixes(t *testing.T) {
	// The spec is silent on a checklist that is mostly unknown with some
	// passes; the grade is "ok" (nothing failed or warned, something passed).
	if s := firewallSummary([]ops.CheckItem{{Status: "unknown"}, {Status: "pass"}, {Status: "unknown"}}); s.Grade != "ok" || s.Unknown != 2 || s.Pass != 1 {
		t.Errorf("%+v", s)
	}
	if s := firewallSummary(nil); s.Grade != "unknown" {
		t.Errorf("empty rule set: %+v", s)
	}
	if s := firewallSummary([]ops.CheckItem{{Status: "fail"}, {Status: "warn"}}); s.Grade != "fail" || s.Fail != 1 || s.Warn != 1 {
		t.Errorf("fail outranks warn: %+v", s)
	}
}

func orNil(p *int64) any {
	if p == nil {
		return nil
	}
	return *p
}
