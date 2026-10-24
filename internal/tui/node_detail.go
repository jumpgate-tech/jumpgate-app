package tui

import (
	"fmt"
	"strings"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/apiclient"
)

// Node detail (spec U5): one client of the open box, its current values and
// the delta from the previous reading ("head lag 2 blk (was 40)"). No
// sparklines or trend graphs. It reads only the box's status stream and the
// modules the Overview tab already loads: no new polling.

// nodeRow is one value of a reading: its label and its text, "n/a" when
// the reading has none.
type nodeRow struct{ label, val, was string }

// nodeValues is a reading's rows for one client, in display order. was is
// the same value without its unit, as the delta shows it.
func nodeValues(s api.NodeStatus, node string) [][3]string {
	var c api.ClientStatus
	if node == "beacon" {
		c = s.Beacon
	} else {
		c = s.Exec
	}
	na := "n/a"
	state := na
	if c.State != "" && c.State != api.SyncUnavailable {
		state = syncWord(c.State)
	}
	// A client that is not running has no head or peers: 0 would read as a reading.
	num := func(v uint64) string {
		if !c.Active {
			return na
		}
		return formatCount(v)
	}
	peers := na
	if c.Active {
		peers = fmt.Sprint(c.Peers)
	}
	disk, diskWas := na, na
	if s.DiskUsedPct != nil {
		diskWas = fmt.Sprintf("%.0f%%", *s.DiskUsedPct)
		disk = diskWas + " used"
	}
	rows := [][3]string{{"state", state, state}}
	if node == "beacon" {
		rows = append(rows,
			[3]string{"slot", num(c.Head), num(c.Head)},
			[3]string{"peers", peers, peers},
			[3]string{"distance", num(c.Distance), num(c.Distance)})
	} else {
		lag, lagWas := na, na
		if s.HeadLag != nil {
			lagWas = formatCount(*s.HeadLag)
			lag = lagWas + " blk"
		}
		ref := na
		if s.RefHead != 0 {
			ref = formatCount(s.RefHead)
		}
		rows = append(rows,
			[3]string{"head", num(c.Head), num(c.Head)},
			[3]string{"peers", peers, peers},
			[3]string{"head lag", lag, lagWas},
			[3]string{"ref head", ref, ref})
	}
	overall := na
	if s.Overall != "" && s.Overall != api.SyncUnavailable {
		overall = syncWord(s.Overall)
	}
	return append(rows, [3]string{"overall", overall, overall}, [3]string{"disk", disk, diskWas})
}

// nodeRows pairs the current reading with the previous one: was is set
// only when there is a previous reading and the value changed.
func (h *hostScreen) nodeRows() []nodeRow {
	cur := nodeValues(h.status, h.node)
	var prev [][3]string
	if h.prevHas {
		prev = nodeValues(h.prev, h.node)
	}
	out := make([]nodeRow, len(cur))
	for i, v := range cur {
		out[i] = nodeRow{label: v[0], val: v[1]}
		if prev != nil && prev[i][2] != v[2] {
			out[i].was = prev[i][2]
		}
	}
	return out
}

// viewNode draws the node detail.
func (h *hostScreen) viewNode(a *App, w int) string {
	title := "EXECUTION"
	if h.node == "beacon" {
		title = "BEACON"
	}
	lines := []string{" " + a.th.Dim.Render("node detail "+a.gl.Sep+" was = the previous reading "+a.gl.Sep+" esc box tabs"), "", " " + a.th.Title.Render(title)}
	if !h.statusHas {
		msg := "waiting for the first reading" + a.gl.Ellipsis
		if h.statusErr != nil {
			msg = "status unavailable: " + a.statusErrText(h.statusErr)
		}
		lines = append(lines, " "+a.th.Dim.Render(msg))
		return strings.Join(lines, "\n") + h.viewModules(a, w)
	}
	age := a.now().Sub(h.status.At)
	stale := age > statusStaleAfter || h.statusConn != apiclient.Live
	for _, r := range h.nodeRows() {
		v := sanitizeLine(r.val)
		if stale {
			v = a.th.Dim.Render(v + "~")
		}
		line := fmt.Sprintf(" %-9s %s", r.label, v)
		if r.was != "" {
			line += " " + a.th.Dim.Render("(was "+sanitizeLine(r.was)+")")
		}
		lines = append(lines, truncate(line, w, a.gl.Ellipsis))
	}
	note := " read " + formatAge(age) + " ago"
	if h.prevHas {
		note += " " + a.gl.Sep + " previous " + formatAge(a.now().Sub(h.prev.At)) + " ago"
	}
	if stale {
		note += " " + a.th.Warn.Render(fmt.Sprintf("(stale: %s old)", formatAge(age)))
	}
	lines = append(lines, "", note)
	if h.statusErr != nil {
		lines = append(lines, " "+a.th.Warn.Render("latest reading failed: "+a.statusErrText(h.statusErr)))
	}
	return strings.Join(lines, "\n") + h.viewModules(a, w)
}
