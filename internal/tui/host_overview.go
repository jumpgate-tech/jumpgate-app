package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/apiclient"
)

// viewOverview is the live status: both clients side by side, every value
// with its age, "unavailable" rather than 0 when there is no reading.
func (h *hostScreen) viewOverview(a *App, w int) string {
	if !h.statusHas {
		msg := "waiting for the first reading" + a.gl.Ellipsis
		if h.statusErr != nil {
			msg = "status unavailable: " + a.statusErrText(h.statusErr)
		}
		return "\n " + a.th.Dim.Render(msg)
	}
	s := h.status
	age := a.now().Sub(s.At)
	stale := age > statusStaleAfter || h.statusConn != apiclient.Live
	val := func(v string) string {
		if stale {
			return a.th.Dim.Render(v)
		}
		return v
	}
	word := func(st api.SyncState) string {
		return val(a.th.state(st).Render(sanitizeLine(strings.ReplaceAll(string(st), "_", " "))))
	}
	// A client that is not running has no head or peers to show: a 0 would
	// read as a reading.
	num := func(c api.ClientStatus, v string) string {
		if !c.Active {
			return val("n/a")
		}
		return val(v)
	}
	lag := "n/a (no reference)"
	if s.HeadLag != nil {
		lag = formatCount(*s.HeadLag) + " blocks"
	}
	col := lipgloss.NewStyle().Width(min(38, w/2))
	exec := strings.Join([]string{
		a.th.Title.Render("EXECUTION"),
		"state   " + word(s.Exec.State),
		"head    " + num(s.Exec, formatCount(s.Exec.Head)),
		"peers   " + num(s.Exec, fmt.Sprint(s.Exec.Peers)),
		"lag     " + val(lag),
	}, "\n")
	beacon := strings.Join([]string{
		a.th.Title.Render("BEACON"),
		"state     " + word(s.Beacon.State),
		"slot      " + num(s.Beacon, formatCount(s.Beacon.Head)),
		"peers     " + num(s.Beacon, fmt.Sprint(s.Beacon.Peers)),
		"distance  " + num(s.Beacon, formatCount(s.Beacon.Distance)),
	}, "\n")
	out := "\n" + lipgloss.JoinHorizontal(lipgloss.Top, col.Render(" "+strings.ReplaceAll(exec, "\n", "\n ")), col.Render(beacon)) + "\n\n"
	note := fmt.Sprintf(" overall %s %s read %s ago", word(s.Overall), a.gl.Sep, formatAge(age))
	if s.DiskUsedPct != nil {
		note += fmt.Sprintf(" %s disk %.0f%% used", a.gl.Sep, *s.DiskUsedPct)
	}
	if stale {
		note += " " + a.th.Warn.Render(fmt.Sprintf("(stale: %s old)", formatAge(age)))
	}
	if h.statusErr != nil {
		note += "\n " + a.th.Warn.Render("latest reading failed: "+a.statusErrText(h.statusErr))
	}
	return out + note
}
