package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/apiclient"
)

// The KPI strip (spec §1, §3): seven tiles over the fleet. Each value is a
// number, n/a (no data), ~number (an input is stale) or FAIL <code> (no
// value and the section errored); unknown is never 0, and aggregates name
// what they leave out.

// tile is one KPI: two rows for the boxed strip, one cell for the compact one.
type tile struct {
	Title string
	Lines [2]string
	Short string
	Level int // 0 plain, 1 warn, 2 bad: colour only, the text says it too
}

const (
	gwTileWidth     = 27 // fits "n/a no request counts yet" in a box
	alertsTileWidth = 18
)

const noRequests = "n/a no request counts yet" // U1: no request counts until phase 6

// kpis is the seven tiles, in order. Pure.
func kpis(in alertInput, al []alert, alertsKnown bool) []tile {
	rows := in.Fleet.Rows
	if !in.FleetHas {
		rows = nil
	}
	n := len(rows)
	na := func(k int) string {
		if k > 0 {
			return fmt.Sprintf("%d n/a", k)
		}
		return ""
	}
	join := func(parts ...string) string {
		var out []string
		for _, p := range parts {
			if p != "" {
				out = append(out, p)
			}
		}
		return strings.Join(out, " ")
	}
	tilde := func(stale bool) string {
		if stale || in.Retrying {
			return "~"
		}
		return ""
	}
	// failCode is the first errored section's code, for a tile with no value.
	failCode := func(get func(api.FleetRow) *api.ProbeError) string {
		for _, r := range rows {
			if e := get(r); e != nil {
				return strings.TrimSpace("FAIL " + sanitizeLine(string(e.Code)))
			}
		}
		return ""
	}
	unknown := func(title, short string, k int, fail string) tile {
		if fail != "" {
			return tile{Title: title, Short: short + " " + fail, Lines: [2]string{fail, na(k)}, Level: 2}
		}
		return tile{Title: title, Short: short + " n/a", Lines: [2]string{"n/a", na(k)}}
	}
	statusErr := func(r api.FleetRow) *api.ProbeError { return r.StatusError }

	var ts []tile

	// Boxes: up, down, and the ones not read yet.
	up, down := 0, 0
	for _, r := range rows {
		switch {
		case r.Reachable:
			up++
		case r.LastSeen != nil || r.CheckedAt != nil:
			down++
		}
	}
	if !in.FleetHas || up+down == 0 {
		t := unknown("Boxes", "boxes", n-up-down, "")
		if n > 0 {
			t.Lines[1] += fmt.Sprintf(" of %d", n)
		}
		ts = append(ts, t)
	} else {
		t := tile{Title: "Boxes", Short: fmt.Sprintf("boxes %s%d/%d up", tilde(false), up, n),
			Lines: [2]string{fmt.Sprintf("%s%d up %d down", tilde(false), up, down), join(na(n-up-down), fmt.Sprintf("of %d", n))}}
		if down > 0 {
			t.Level = 2
		}
		ts = append(ts, t)
	}

	// Synced: nodes synced of all boxes, then syncing and unread.
	withStatus, synced, syncing, stale := 0, 0, 0, false
	for _, r := range rows {
		if r.Status == nil {
			continue
		}
		withStatus++
		stale = stale || r.Stale.Status
		switch r.Status.Overall {
		case api.SyncSynced:
			synced++
		case api.SyncSyncing:
			syncing++
		}
	}
	if withStatus == 0 {
		ts = append(ts, unknown("Synced", "synced", n, failCode(statusErr)))
	} else {
		v := fmt.Sprintf("%s%d/%d", tilde(stale), synced, n)
		sy := ""
		if syncing > 0 {
			sy = fmt.Sprintf("%d sync", syncing)
		}
		ts = append(ts, tile{Title: "Synced", Short: "synced " + v, Lines: [2]string{v + " synced", join(sy, na(n-withStatus))}})
	}

	// Head lag: the worst known lag.
	withLag, lagMax, stale := 0, uint64(0), false
	for _, r := range rows {
		if r.Status == nil || r.Status.HeadLag == nil {
			continue
		}
		withLag++
		stale = stale || r.Stale.Status
		lagMax = max(lagMax, *r.Status.HeadLag)
	}
	if withLag == 0 {
		ts = append(ts, unknown("Head lag", "lag", n, failCode(statusErr)))
	} else {
		v := fmt.Sprintf("%s%d", tilde(stale), lagMax)
		ts = append(ts, tile{Title: "Head lag", Short: "lag " + v, Lines: [2]string{"max " + v + " blk", na(n - withLag)}})
	}

	// Peers: exec and beacon sums over the boxes with a reading.
	exec, beacon, stale := 0, 0, false
	for _, r := range rows {
		if r.Status != nil {
			exec, beacon = exec+r.Status.Exec.Peers, beacon+r.Status.Beacon.Peers
			stale = stale || r.Stale.Status
		}
	}
	if withStatus == 0 {
		ts = append(ts, unknown("Peers", "peers", n, failCode(statusErr)))
	} else {
		t := tilde(stale)
		ts = append(ts, tile{Title: "Peers", Short: fmt.Sprintf("peers %s%d/%d", t, exec, beacon),
			Lines: [2]string{fmt.Sprintf("%s%d exec", t, exec), join(fmt.Sprintf("%s%d bcn", t, beacon), na(n-withStatus))}})
	}

	// Disk: the least free space, and the box with it.
	worst, worstPct, withDisk, stale := "", -1.0, 0, false
	for _, r := range rows {
		if r.Status == nil || r.Status.DiskUsedPct == nil {
			continue
		}
		withDisk++
		stale = stale || r.Stale.Status || r.Stale.Disk
		if *r.Status.DiskUsedPct > worstPct {
			worst, worstPct = r.ID, *r.Status.DiskUsedPct
		}
	}
	if withDisk == 0 {
		ts = append(ts, unknown("Disk", "disk", n, failCode(func(r api.FleetRow) *api.ProbeError { return r.DiskError })))
	} else {
		free := fmt.Sprintf("%s%.0f%%", tilde(stale), 100-worstPct)
		ts = append(ts, tile{Title: "Disk", Short: "disk " + free,
			Lines: [2]string{"min " + free + " free", "on " + sanitizeLine(worst)}})
	}

	// Gateways: live and failing; request counts have no API yet (U1).
	if !in.GatewaysHas {
		ts = append(ts, tile{Title: "Gateways", Short: "gw n/a", Lines: [2]string{"n/a", noRequests}})
	} else {
		live, failing := 0, 0
		for _, g := range in.Gateways {
			switch g.Status.State {
			case "running":
				live++
			case "":
			default:
				failing++
			}
		}
		t := tile{Title: "Gateways", Short: fmt.Sprintf("gw %d/%d live", live, len(in.Gateways)),
			Lines: [2]string{fmt.Sprintf("%d live %d failing", live, failing), noRequests}}
		if failing > 0 {
			t.Level = 2
		}
		ts = append(ts, t)
	}

	// Alerts: counts by severity and the first one.
	if !alertsKnown {
		ts = append(ts, tile{Title: "Alerts", Short: "! alerts n/a", Lines: [2]string{"n/a", ""}})
	} else {
		fails := 0
		for _, x := range al {
			if x.Sev == sevFail {
				fails++
			}
		}
		t := tile{Title: "Alerts", Short: "! 0 alerts", Lines: [2]string{"0 alerts", ""}}
		if len(al) > 0 {
			c := fmt.Sprintf("%d FAIL %d warn", fails, len(al)-fails)
			t = tile{Title: "Alerts", Short: "! " + c, Lines: [2]string{c, al[0].Box + " " + al[0].Text}, Level: 1}
			if fails > 0 {
				t.Level = 2
			}
		}
		ts = append(ts, t)
	}
	return ts
}

// alertInput is what the App holds now, for the alerts and the tiles.
func (a *App) alertInput() alertInput {
	return alertInput{Fleet: a.fleet, FleetHas: a.fleetHas, Gateways: a.gws, GatewaysHas: a.gwsHas,
		Retrying: a.fleetHas && a.conn == apiclient.Retrying, Now: a.now()}
}

// currentAlerts asks the alert source.
func (a *App) currentAlerts() ([]alert, bool) {
	src := a.alertSrc
	if src == nil {
		src = derivedAlerts{}
	}
	return src.alerts(a.alertInput())
}

// alertCount is "! N alerts", or "! alerts n/a" before there is anything to
// judge; coloured by the worst alert.
func (a *App) alertCount() string {
	al, known := a.currentAlerts()
	if !known {
		return "! alerts n/a"
	}
	s := fmt.Sprintf("! %d alerts", len(al))
	switch {
	case len(al) > 0 && al[0].Sev == sevFail:
		return a.th.Bad.Render(s)
	case len(al) > 0:
		return a.th.Warn.Render(s)
	}
	return s
}

func (a *App) tileStyle(t tile, s string) string {
	switch t.Level {
	case 2:
		return a.th.Bad.Render(s)
	case 1:
		return a.th.Warn.Render(s)
	}
	return s
}

// kpiView draws the strip in r: one line of cells, or (4 rows) the boxed
// tiles side by side, the Gateways tile wide enough for its hint.
func (a *App) kpiView(r rect) string {
	al, known := a.currentAlerts()
	ts := kpis(a.alertInput(), al, known)
	if r.H < 4 {
		var plain, styled []string
		for _, t := range ts {
			plain = append(plain, t.Short)
			styled = append(styled, a.tileStyle(t, t.Short))
		}
		// Narrower windows close up the separators before anything is cut.
		for _, sep := range []string{" | ", "  ", " "} {
			if lipgloss.Width(" "+strings.Join(plain, sep)) <= r.W || sep == " " {
				return truncate(" "+strings.Join(styled, sep), r.W, a.gl.Ellipsis)
			}
		}
	}
	// Gateways is wide enough for its hint and Alerts for its counts (both
	// smaller in a narrow window); the other five share the rest.
	fixed := map[string]int{"Gateways": min(gwTileWidth, r.W/4), "Alerts": min(alertsTileWidth, r.W/6)}
	rest := r.W - fixed["Gateways"] - fixed["Alerts"]
	others := len(ts) - len(fixed)
	var boxes []string
	k := 0
	for _, t := range ts {
		w, ok := fixed[t.Title]
		if !ok {
			w = rest / others
			if k < rest%others {
				w++
			}
			k++
		}
		body := a.tileStyle(t, truncate(t.Lines[0], w-2, a.gl.Ellipsis)) + "\n" + truncate(t.Lines[1], w-2, a.gl.Ellipsis)
		boxes = append(boxes, a.boxed(t.Title, false, body, w, 4))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, boxes...)
}
