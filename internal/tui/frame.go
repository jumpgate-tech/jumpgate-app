package tui

import (
	"fmt"
	"strings"
	"time"

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
// box's border is drawn in the info colour, and colour is never the only
// sign: its title starts with ">", and in ASCII its top border is "=".
func (a *App) boxed(title string, focused bool, body string, w, h int) string {
	if w < 2 || h < 2 {
		return fit(body, max(w, 0), max(h, 0), a.gl.Ellipsis)
	}
	b := a.border()
	st := a.th.Dim
	if focused {
		st = a.th.Info
	}
	top := b.H
	if focused {
		title = strings.TrimSpace(a.gl.Sel + " " + title)
		if a.gl.ASCII {
			top = "="
		}
	}
	head := ""
	if title != "" {
		head = " " + truncate(title, max(w-6, 1), a.gl.Ellipsis) + " "
	}
	fill := max(w-3-lipgloss.Width(head), 0)
	lines := []string{st.Render(b.TL+top) + head + st.Render(strings.Repeat(top, fill)+b.TR)}
	for _, l := range strings.Split(fit(body, w-2, h-2, a.gl.Ellipsis), "\n") {
		lines = append(lines, st.Render(b.V)+l+st.Render(b.V))
	}
	lines = append(lines, st.Render(b.BL+strings.Repeat(b.H, w-2)+b.BR))
	return strings.Join(lines, "\n")
}

// frame is the whole dashboard at a window of at least 80×24.
func (a *App) frame() string {
	l := layoutFor(a.w, a.h, drawerFromPref(a.prefs.Drawer))
	parts := []string{a.header(a.w), a.kpiView(l.KPI)}
	body := a.mainView(l.Main)
	switch {
	case a.logOver:
		body = overlay(body, a.logView(min(overlayWidth, l.Main.W), l.Main.H), l.Main.W-min(overlayWidth, l.Main.W), l.Main.W)
	case a.treeOver:
		body = overlay(body, a.boxed("Tree", a.focus == focusTree, a.treeView(overlayWidth-2, l.Main.H-2), min(overlayWidth, l.Main.W), l.Main.H), 0, l.Main.W)
	}
	cols := []string{}
	if l.Left.W > 0 {
		cols = append(cols, a.viewsList(l.Left))
	}
	cols = append(cols, body)
	if l.Right.W > 0 {
		cols = append(cols, a.logView(l.Right.W, l.Right.H))
	}
	body = lipgloss.JoinHorizontal(lipgloss.Top, cols...)
	parts = append(parts, body)
	last := a.drawerLine(a.w)
	if l.DrawerOpen {
		parts = append(parts, a.drawerView(l.Drawer, l.DrawerCompact))
		last = truncate(" "+a.hints(), a.w, a.gl.Ellipsis)
	}
	return lipgloss.JoinVertical(lipgloss.Left, append(parts, last)...)
}

// header is the top line: the server link and version on the left, the age
// of the fleet data on the right. Below 100 columns the view tabs take the
// server's place, so the body keeps the row a tab row would cost; a
// retrying link still shows first, and "connecting" shows on the right.
func (a *App) header(w int) string {
	narrow := w < leftMinWidth
	server := "server " + a.conn.String()
	if a.conn == apiclient.Retrying && a.connErr != nil {
		server += " (" + a.errText(a.connErr) + ")"
	}
	left := " " + a.th.Title.Render("jumpgate") + "  "
	switch {
	case narrow && a.conn == apiclient.Retrying:
		left += server + "  " + a.tabs()
	case narrow:
		left += a.tabs()
	default:
		left += server
		if a.srvVer != "" {
			left += " " + a.gl.Sep + " " + a.srvVer
		}
	}
	right := a.th.Dim.Render("connecting")
	if a.fleetHas {
		age := a.now().Sub(a.fleet.At)
		every := time.Duration(a.fleet.IntervalSeconds) * time.Second
		if every <= 0 {
			every = statusStaleAfter
		}
		text := "updated " + formatAge(age) + " ago"
		if narrow {
			text = formatAge(age) + " ago"
		}
		switch {
		case age > 4*every || a.conn == apiclient.Retrying:
			right = a.th.Bad.Render("stale, " + text)
		case age > 2*every:
			right = a.th.Warn.Render(text + " ~")
		default:
			right = text
		}
	}
	if a.fleetHas {
		right = a.alertCount() + " " + a.gl.Sep + " " + right
	}
	right += " "
	left = truncate(left, max(w-lipgloss.Width(right)-1, 1), a.gl.Ellipsis)
	return left + strings.Repeat(" ", max(w-lipgloss.Width(left)-lipgloss.Width(right), 1)) + right
}

// tabs is the views in one line, for the header below 100 columns; the
// active one is bracketed, so it reads without colour too.
func (a *App) tabs() string {
	var parts []string
	for i, n := range screenNames {
		if screenID(i) == a.active && a.detail == nil {
			n = a.th.Sel.Render("[" + n + "]")
		}
		parts = append(parts, n)
	}
	return strings.Join(parts, " ")
}

// viewsList is the left sidebar: the views with their number keys, then
// the fleet tree.
func (a *App) viewsList(r rect) string {
	var lines []string
	for i, n := range screenNames {
		l := fmt.Sprintf("  %-14s [%d]", n, i+1)
		if screenID(i) == a.active && a.detail == nil {
			l = a.th.Sel.Render(fmt.Sprintf("%s %-14s [%d]", a.gl.Sel, n, i+1))
		}
		lines = append(lines, l)
	}
	lines = append(lines, a.th.Dim.Render(strings.Repeat(a.border().H, r.W-2)), a.treeView(r.W-2, max(r.H-3-len(lines), 1)))
	return a.boxed("Views", a.focus == focusTree, strings.Join(lines, "\n"), r.W, r.H)
}

// mainView is the boxed main region: the skew banner above it, then the
// screen (or the modal over it) inside.
func (a *App) mainView(r rect) string {
	h := r.H
	banner := ""
	if a.skew != "" {
		banner = a.th.Warn.Render(pad(truncate(" "+a.skew, r.W, a.gl.Ellipsis), r.W)) + "\n"
		h--
	}
	// Below 100 columns main is unboxed, as in the spec's 80x24 frame: the
	// table keeps the columns it had before the dashboard.
	boxedMain := a.w >= leftMinWidth
	iw, ih := r.W, h
	if boxedMain {
		iw, ih = r.W-2, h-2
	}
	title := screenNames[a.active]
	if hs, ok := a.detail.(*hostScreen); ok {
		title = sanitizeLine(hs.id)
	}
	var content string
	switch {
	case a.modal != nil:
		content = a.modal.view(a, iw, ih)
	default:
		content = a.current().view(a, iw, ih)
	}
	if !boxedMain {
		return banner + fit(content, r.W, h, a.gl.Ellipsis)
	}
	return banner + a.boxed(title, a.focus == focusMain, content, r.W, h)
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
		parts = append(parts, a.alertCount())
		left = strings.Join(parts, "  ")
	}
	return truncate(" "+left+" | "+a.hints(), w, a.gl.Ellipsis)
}

// hints are the keys worth showing on one line, the drawer's first.
func (a *App) hints() string {
	hs := []string{". drawer", "? help", ": command", "1-5 views", "tab focus"}
	if a.w < rightMinWidth {
		hs[4] = "[ ] panels"
	}
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
