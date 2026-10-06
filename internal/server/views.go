package server

import (
	"time"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/catalog"
	"github.com/valve-tech/jumpgate/internal/monitor"
	"github.com/valve-tech/jumpgate/internal/ops"
)

// trendMinSpan is the least history a growth estimate is drawn from: less
// than this and one compaction or one snapshot import dominates the line.
const trendMinSpan = 6 * time.Hour

func clientState(active, syncing bool, head uint64) api.SyncState {
	switch {
	case !active:
		return api.SyncStopped
	case syncing:
		return api.SyncSyncing
	case head == 0:
		return api.SyncNoData
	}
	return api.SyncSynced
}

// nodeStatusView turns a status reading into the verdicts both front ends
// show. The rules are the web dashboard's: a node with a stopped client is
// stopped; either client catching up is syncing; synced needs both.
func nodeStatusView(s monitor.Snapshot) api.NodeStatus {
	v := api.NodeStatus{
		At:      s.At,
		Exec:    api.ClientStatus{Active: s.ExecActive, Syncing: s.ExecSyncing, Head: s.ExecHead, Peers: s.ExecPeers},
		Beacon:  api.ClientStatus{Active: s.BeaconActive, Syncing: s.BeaconDistance > 0, Head: s.BeaconSlot, Distance: s.BeaconDistance, Peers: s.BeaconPeers},
		RefHead: s.RefHead,
	}
	v.Exec.State = clientState(v.Exec.Active, v.Exec.Syncing, v.Exec.Head)
	v.Beacon.State = clientState(v.Beacon.Active, v.Beacon.Syncing, v.Beacon.Head)
	switch {
	case v.Exec.State == api.SyncStopped || v.Beacon.State == api.SyncStopped:
		v.Overall = api.SyncStopped
	case v.Exec.State == api.SyncSyncing || v.Beacon.State == api.SyncSyncing:
		v.Overall = api.SyncSyncing
	case v.Exec.State == api.SyncNoData || v.Beacon.State == api.SyncNoData:
		v.Overall = api.SyncNoData
	default:
		v.Overall = api.SyncSynced
	}
	if s.RefHead > 0 && s.ExecHead > 0 {
		lag := uint64(0)
		if s.RefHead > s.ExecHead {
			lag = s.RefHead - s.ExecHead
		}
		v.HeadLag = &lag
	}
	if s.DiskKnown {
		pct := s.DiskUsedPct
		v.DiskUsedPct = &pct
	}
	return v
}

// diskView turns a disk reading into the fit verdict and trend.
func diskView(at time.Time, du ops.DU, history []api.DiskSample) api.DiskView {
	v := api.DiskView{
		At: at, ExecBytes: du.ExecBytes, BeaconBytes: du.BeaconBytes, FreeBytes: du.DiskFreeBytes,
		ExpectedExecBytes: du.ExpectedExecBytes, ExpectedBeaconBytes: du.ExpectedBeaconBytes,
		ExpectedLabel: "estimate", SyncLabel: du.SyncLabel, History: history,
	}
	expected := float64(du.ExpectedExecBytes + du.ExpectedBeaconBytes)
	room := float64(du.ExecBytes + du.BeaconBytes + du.DiskFreeBytes) // what the clients can grow into
	switch {
	case expected == 0:
		v.Fit = api.FitUnknown
	case room >= expected*catalog.FitMargin:
		v.Fit = api.FitOK
	case room >= expected:
		v.Fit = api.FitTight
	default:
		v.Fit = api.FitShort
	}
	v.GrowthPerDay, v.DaysToFull = diskTrend(history)
	return v
}

// diskTrend estimates growth per day from the first and last samples, and
// days until the free space is gone at that rate. Nil when the history spans
// less than trendMinSpan; no days-to-full when usage is not growing.
func diskTrend(history []api.DiskSample) (*int64, *float64) {
	if len(history) < 2 {
		return nil, nil
	}
	first, last := history[0], history[len(history)-1]
	span := last.At.Sub(first.At)
	if span < trendMinSpan {
		return nil, nil
	}
	days := span.Hours() / 24
	growth := int64((float64(last.UsedBytes) - float64(first.UsedBytes)) / days)
	if growth <= 0 {
		return &growth, nil
	}
	left := float64(last.FreeBytes) / float64(growth)
	return &growth, &left
}

// firewallSummary counts a checklist and grades it.
func firewallSummary(items []ops.CheckItem) api.FirewallSummary {
	var s api.FirewallSummary
	for _, it := range items {
		switch it.Status {
		case "pass":
			s.Pass++
		case "warn":
			s.Warn++
		case "fail":
			s.Fail++
		default:
			s.Unknown++
		}
	}
	switch {
	case s.Fail > 0:
		s.Grade = "fail"
	case s.Warn > 0:
		s.Grade = "warn"
	case s.Pass == 0:
		s.Grade = "unknown"
	default:
		s.Grade = "ok"
	}
	return s
}
