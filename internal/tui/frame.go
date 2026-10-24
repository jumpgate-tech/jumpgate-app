package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/valve-tech/jumpgate/internal/apiclient"
)

// The dashboard frame (spec §1): header, views, boxed main, activity drawer
// and the hints line.

// border is the box-drawing set for a glyph set: rounded unicode, or +-|.
type border struct{ TL, TR, BL, BR, H, V string }

func (a *App) border() border {
	if a.gl.ASCII {
		return border{"+", "+", "+", "+", "-", "|"}
	}
	b := lipgloss.RoundedBorder()
	return border{b.TopLeft, b.TopRight, b.BottomLeft, b.BottomRight, b.Top, b.Left}
}

// boxed draws body in a w×h box with title in the top border. The focused
// box's border is drawn in the info colour; colour is never the only sign
// of focus (the title says which region it is).
func (a *App) boxed(title string, focused bool, body string, w, h int) string {
	if w < 2 || h < 2 {
		return fit(body, max(w, 0), max(h, 0), a.gl.Ellipsis)
	}
	b := a.border()
	st := a.th.Dim
	if focused {
		st = a.th.Info
	}
	head := ""
	if title != "" {
		head = " " + truncate(title, max(w-6, 1), a.gl.Ellipsis) + " "
	}
	fill := max(w-3-lipgloss.Width(head), 0)
	lines := []string{st.Render(b.TL+b.H) + head + st.Render(strings.Repeat(b.H, fill)+b.TR)}
	for _, l := range strings.Split(fit(body, w-2, h-2, a.gl.Ellipsis), "\n") {
		lines = append(lines, st.Render(b.V)+l+st.Render(b.V))
	}
	lines = append(lines, st.Render(b.BL+strings.Repeat(b.H, w-2)+b.BR))
	return strings.Join(lines, "\n")
}

// drawerOpen is the drawer's resolved state at this window size.
func (a *App) drawerOpen() bool {
	return layoutFor(a.w, a.h, drawerFromPref(a.prefs.Drawer)).DrawerOpen
}

// toggleDrawer stores the opposite of what shows now, explicitly.
func (a *App) toggleDrawer() tea.Cmd {
	p := a.prefs
	p.Drawer = drawerOpen.pref()
	if a.drawerOpen() {
		p.Drawer = drawerMin.pref()
	}
	return savePrefs(a, p)
}

// jobsRunning is the fleet-wide running-job count, or n/a until some box
// has reported its jobs (unknown is not 0).
func (a *App) jobsRunning() string {
	n, known := 0, false
	for _, r := range a.fleet.Rows {
		if r.Jobs != nil {
			n, known = n+r.Jobs.Running, true
		}
	}
	if !known {
		return "jobs n/a"
	}
	return fmt.Sprintf("jobs %d running", n)
}

// drawerLine is the minimized drawer: counts (or the flash), then hints.
// The drawer never opens itself; a pending decision is highlighted here.
func (a *App) drawerLine(w int) string {
	left := a.flash
	if left == "" {
		parts := []string{a.jobsRunning()}
		if n := a.needCount(); n > 0 {
			parts = append(parts, a.th.Warn.Render(fmt.Sprintf("[! %d needs decision]", n)))
		}
		parts = append(parts, "! alerts n/a")
		left = strings.Join(parts, "  ")
	}
	return truncate(" "+left+" | "+a.hints(), w, a.gl.Ellipsis)
}

// hints are the keys worth showing on one line, the drawer's first.
func (a *App) hints() string {
	hs := []string{". drawer", "? help", ": command", "1-6 views"}
	if a.detail != nil {
		hs = append(hs, "esc back")
	} else {
		hs = append(hs, "q quit")
	}
	return strings.Join(hs, "  ")
}

// statusLine is the selected box and its link, the server connection and
// the age of the fleet data.
func (a *App) statusLine() string {
	box := "no box selected"
	if a.sel != "" {
		box = sanitizeLine(a.sel)
		if row, ok := a.fleet.Row(a.sel); ok {
			box += " " + a.gl.Sep + " " + linkWord(row)
		}
	}
	server := "server " + a.conn.String()
	if a.conn == apiclient.Retrying && a.connErr != nil {
		server += " (" + a.errText(a.connErr) + ")"
	}
	s := box + " " + a.gl.Sep + " " + server
	if a.fleetHas {
		s += " " + a.gl.Sep + " updated " + formatAge(a.now().Sub(a.fleet.At)) + " ago"
	}
	return s
}

// activity is the open drawer's lines: a line per box with jobs, then the
// flash or the status line.
func (a *App) activity() []string {
	var out []string
	for _, r := range a.fleet.Rows {
		if r.Jobs == nil {
			continue
		}
		l := fmt.Sprintf("jobs  %s  %d running", sanitizeLine(r.ID), r.Jobs.Running)
		if r.Jobs.NeedsDecision > 0 {
			l += " " + a.gl.Sep + " " + a.th.Warn.Render(fmt.Sprintf("%d needs decision", r.Jobs.NeedsDecision))
		}
		out = append(out, l)
	}
	last := a.statusLine()
	if a.flash != "" {
		last = a.flash
	}
	return append(out, last)
}

// drawerView is the open drawer in r: boxed with the title Activity, or one
// unboxed line when compact (the flash, else the first job, else status).
func (a *App) drawerView(r rect, compact bool) string {
	lines := a.activity()
	if compact {
		l := lines[0]
		if a.flash != "" {
			l = a.flash
		}
		return truncate(" "+l, r.W, a.gl.Ellipsis)
	}
	room := max(r.H-2, 1)
	if len(lines) > room { // keep the last (status) line
		lines = append(lines[:room-1:room-1], lines[len(lines)-1])
	}
	for i := range lines {
		lines[i] = " " + lines[i]
	}
	return a.boxed("Activity", false, strings.Join(lines, "\n"), r.W, r.H)
}
