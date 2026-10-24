package tui

import (
	"cmp"
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/valve-tech/jumpgate/internal/api"
)

// The activity drawer's progress bars (spec U7): only things with a known
// end. A snapshot download and a fleet rollout get bars once the API carries
// their done/total; today it carries neither.

// barKind is a bar's group, in the order the drawer shows them.
type barKind int

const (
	barDecision barKind = iota // a job waiting on a person
	barJob                     // a running job
	barSync                    // exec or beacon catching up
	barDisk                    // a disk at or above diskBarPct
)

// diskBarPct is the fill at which a disk gets a capacity bar (the warn alert).
const diskBarPct = 85

// progressBar is one bar. Total is never 0: nothing without a denominator
// gets a bar.
type progressBar struct {
	Kind        barKind
	Box, What   string // sanitized
	Done, Total uint64
	Note        string
	Stale       bool
}

// barInput is the data the drawer reads: the fleet stream, and the jobs
// inbox while the Jobs view watches it.
type barInput struct {
	Fleet      api.Fleet
	FleetStale bool
	Jobs       []api.JobView
	JobsHas    bool
	JobsStale  bool
}

// drawerBars is every bar, in the U7 order: needs decision, running jobs,
// syncing clients, full disks (fullest first).
func drawerBars(in barInput) []progressBar {
	var dec, run, syn, disk []progressBar
	if in.JobsHas {
		for _, j := range in.Jobs {
			n := len(j.Steps)
			if n == 0 || (j.State != "needs-decision" && j.State != "running") {
				continue
			}
			b := progressBar{Kind: barJob, Box: sanitizeLine(j.Target), What: sanitizeLine(strings.TrimPrefix(j.Kind, "node.")),
				Done: uint64(min(max(j.Step, 0), n)), Total: uint64(n), Note: fmt.Sprintf("step %d/%d", min(j.Step+1, n), n), Stale: in.JobsStale}
			if j.State == "needs-decision" {
				b.Kind = barDecision
				dec = append(dec, b)
			} else {
				run = append(run, b)
			}
		}
	}
	for _, r := range in.Fleet.Rows {
		box := sanitizeLine(r.ID)
		if st := r.Status; st != nil {
			stale := in.FleetStale || r.Stale.Status
			if st.Exec.State == api.SyncSyncing && st.RefHead > 0 {
				syn = append(syn, progressBar{Kind: barSync, Box: box, What: "exec", Done: min(st.Exec.Head, st.RefHead), Total: st.RefHead,
					Note: shortCount(st.Exec.Head) + "/" + shortCount(st.RefHead), Stale: stale})
			}
			if st.Beacon.State == api.SyncSyncing && st.Beacon.Distance > 0 {
				total := st.Beacon.Head + st.Beacon.Distance
				syn = append(syn, progressBar{Kind: barSync, Box: box, What: "beacon", Done: st.Beacon.Head, Total: total,
					Note: shortCount(st.Beacon.Head) + "/" + shortCount(total), Stale: stale})
			}
		}
		if f := diskFrac(r); f >= 0 {
			pct := uint64(math.Round(f * 100))
			if pct >= diskBarPct {
				disk = append(disk, progressBar{Kind: barDisk, Box: box, What: "disk", Done: min(pct, 100), Total: 100,
					Note: fmt.Sprintf("%d%% used", pct), Stale: in.FleetStale || r.Stale.Status || r.Stale.Disk})
			}
		}
	}
	slices.SortStableFunc(disk, func(x, y progressBar) int { return cmp.Compare(y.Done, x.Done) })
	return slices.Concat(dec, run, syn, disk)
}

// capBars keeps the first n bars and counts the rest, for "+N more".
func capBars(bs []progressBar, n int) ([]progressBar, int) {
	if len(bs) <= n {
		return bs, 0
	}
	return bs[:n], len(bs) - n
}
