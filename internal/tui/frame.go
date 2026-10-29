package tui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/valve-tech/jumpgate/internal/api"
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
	l := a.layout()
	a.mainRect = l.Main
	a.shown = map[string]pane{}
	head, kpi := a.header(a.w), a.kpiView(l.KPI)
	a.keep("header", "", 0, 0, head)
	a.keep("kpi", "", 0, l.KPI.Y, kpi)
	parts := []string{head, kpi}
	body := a.mainView(l.Main)
	switch {
	case a.logOver:
		ow := min(overlayWidth, l.Main.W)
		lv := a.logView(ow, l.Main.H)
		a.keepBoxed("log", a.logTitle(), rect{l.Main.X + l.Main.W - ow, l.Main.Y, ow, l.Main.H}, lv)
		body = overlay(body, lv, l.Main.W-ow, l.Main.W)
	case a.treeOver:
		ow := min(overlayWidth, l.Main.W)
		tv := a.boxed("Tree", a.focus == focusTree, a.treeView(overlayWidth-2, l.Main.H-2), ow, l.Main.H)
		a.keepBoxed("nav", "Tree", rect{l.Main.X, l.Main.Y, ow, l.Main.H}, tv)
		body = overlay(body, tv, 0, l.Main.W)
	}
	cols := []string{}
	if l.Left.W > 0 {
		nav := a.viewsList(l.Left)
		a.keepBoxed("nav", "Views", l.Left, nav)
		cols = append(cols, nav)
	}
	cols = append(cols, body)
	if l.Right.W > 0 {
		lv := a.logView(l.Right.W, l.Right.H)
		a.keepBoxed("log", a.logTitle(), l.Right, lv)
		cols = append(cols, lv)
	}
	body = lipgloss.JoinHorizontal(lipgloss.Top, cols...)
	parts = append(parts, body)
	if l.Invite.H > 0 {
		inv := a.inviteLine(a.w)
		a.keep("invite", "", 0, l.Invite.Y, inv)
		parts = append(parts, inv)
	}
	last := a.drawerLine(a.w)
	if l.DrawerOpen {
		dv := a.drawerView(l.Drawer, l.DrawerCompact)
		if l.DrawerCompact {
			a.keep("drawer", "Activity", 0, l.Drawer.Y, dv)
		} else {
			a.keepBoxed("drawer", "Activity", l.Drawer, dv)
		}
		parts = append(parts, dv)
		last = truncate(" "+a.hints(), a.w, a.gl.Ellipsis)
	}
	a.keep("hints", "", 0, a.h-1, last)
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
	case narrow:
		if a.conn == apiclient.Retrying {
			left += server + "  "
		}
		left += a.tabs(w - lipgloss.Width(left) - lipgloss.Width(a.headerRight(narrow)) - 2)
	default:
		left += server
		if a.srvVer != "" {
			left += " " + a.gl.Sep + " " + a.srvVer
		}
	}
	right := a.headerRight(narrow)
	left = truncate(left, max(w-lipgloss.Width(right)-1, 1), a.gl.Ellipsis)
	return left + strings.Repeat(" ", max(w-lipgloss.Width(left)-lipgloss.Width(right), 1)) + right
}

// headerRight is the header's right side: the alerts, the setup segment
// and the reading's age.
func (a *App) headerRight(narrow bool) string {
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
	if seg := a.setupSegment(); seg != "" {
		right = a.th.Warn.Render(seg) + " " + a.gl.Sep + " " + right
	}
	if a.fleetHas {
		right = a.alertCount() + " " + a.gl.Sep + " " + right
		if a.fleet.Locked {
			right = a.th.Bad.Render("LOCKED: jobs wait for unlock") + " " + a.gl.Sep + " " + right
		}
	}
	return right + " "
}

// tabs is the views in one line, for the header below 100 columns; the
// active one is bracketed, so it reads without colour too.
// Where the names do not fit in room cells, the views other than the
// active one show as their number keys.
func (a *App) tabs(room int) string {
	var full, short []string
	for i, n := range screenNames {
		k := fmt.Sprint(i + 1)
		if screenID(i) == a.navActive() {
			n = a.th.Sel.Render("[" + n + "]")
			k = n
		}
		full, short = append(full, n), append(short, k)
	}
	if s := strings.Join(full, " "); lipgloss.Width(s) <= room {
		return s
	}
	return strings.Join(short, " ")
}

// viewsList is the left sidebar: the views with their number keys, then
// the fleet tree.
func (a *App) viewsList(r rect) string {
	var lines []string
	for i, n := range screenNames {
		nw := max(r.W-2-6, 1) // "x " + name + " [n]"
		l := fmt.Sprintf("  %-*s [%d]", nw, n, i+1)
		if screenID(i) == a.navActive() {
			l = a.th.Sel.Render(fmt.Sprintf("%s %-*s [%d]", a.gl.Sel, nw, n, i+1))
		}
		lines = append(lines, l)
	}
	lines = append(lines, a.th.Dim.Render(strings.Repeat(a.border().H, r.W-2)), a.treeView(r.W-2, max(r.H-3-len(lines), 1)))
	return a.boxed("Views", a.focus == focusTree, strings.Join(lines, "\n"), r.W, r.H)
}

// navActive is the view the nav marks: the open box belongs to the Fleet.
func (a *App) navActive() screenID {
	if a.detail != nil {
		return scrFleet
	}
	return a.active
}

// crumbs is where main is: the view, then the open box and its tab (or
// client). The main pane's title joins them.
func (a *App) crumbs() []string {
	hs, ok := a.detail.(*hostScreen)
	if !ok {
		return []string{screenNames[a.active]}
	}
	c := []string{screenNames[scrFleet], sanitizeLine(hs.id)}
	if hs.node != "" {
		return append(c, hs.node)
	}
	t, ok := tabTitles[hs.tabName(a)]
	if !ok {
		t = sanitizeLine(hs.tabName(a))
	}
	return append(c, t)
}

// breadcrumb is the crumbs joined: "Fleet › box › Logs" (">" in ASCII).
func (a *App) breadcrumb() string {
	sep := " › "
	if a.gl.ASCII {
		sep = " > "
	}
	return strings.Join(a.crumbs(), sep)
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
	title := a.breadcrumb()
	crumb := ""
	if !boxedMain && a.detail != nil {
		// Unboxed main has no title border: the breadcrumb takes a line.
		crumb = a.th.Title.Render(truncate(" "+title, r.W, a.gl.Ellipsis)) + "\n"
		ih--
	}
	top := r.Y + strings.Count(banner, "\n")
	var content string
	switch {
	case a.modal != nil:
		content = a.modal.view(a, iw, ih)
	default:
		content = a.current().view(a, iw, ih)
	}
	if !boxedMain {
		body := fit(content, r.W, ih, a.gl.Ellipsis)
		a.keep("main", title, r.X, top+strings.Count(crumb, "\n"), body)
		return banner + crumb + body
	}
	box := a.boxed(title, a.focus == focusMain, content, r.W, h)
	a.keepBoxed("main", title, rect{r.X, top, r.W, h}, box)
	return banner + box
}

// drawerOpen is the drawer's resolved state at this window size.
func (a *App) drawerOpen() bool {
	return a.layout().DrawerOpen
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
		// The alert count is in the header already: every fact once.
		parts := []string{a.jobsRunning()}
		if n := a.needCount(); n > 0 {
			parts = append(parts, a.th.Warn.Render(fmt.Sprintf("[! %d needs decision]", n)))
		}
		left = strings.Join(parts, "  ")
	}
	return truncate(" "+left+" | "+a.hints(), w, a.gl.Ellipsis)
}

// hints are the focused pane's keys on one line, each key a button the
// mouse can press; help (?) lists the rest.
func (a *App) hints() string {
	var hs []string
	switch {
	case a.modal != nil:
	case a.focus == focusTree:
		hs = []string{a.gl.UpDown + " select", "[space] expand", "[enter] open"}
	case a.focus == focusLog:
		hs = []string{a.gl.UpDown + " scroll"}
	default:
		hs = a.paneHints(a.current().keys())
	}
	return strings.Join(append(hs, "[?] help"), "  ")
}

// paneHints is a screen's keys as hints: the arrows folded into one each.
func (a *App) paneHints(bs []key.Binding) []string {
	var hs []string
	for _, b := range bs {
		if !b.Enabled() || (a.inviteOn && a.invite != "" && b.Help().Key == "c") {
			continue // c copies the invite while it shows
		}
		switch b.Help() {
		case navKeys.Up.Help(), navKeys.Left.Help():
			continue
		case navKeys.Down.Help():
			hs = append(hs, a.gl.UpDown+" select")
			continue
		case navKeys.Right.Help():
			hs = append(hs, "left/right tab")
			continue
		}
		hs = append(hs, "["+keyLabel(b.Help().Key, a.gl)+"] "+b.Help().Desc)
	}
	return hs
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

// barInput is what the drawer's bars read: the fleet stream, and the jobs
// inbox while the Jobs view watches it. On every other view the job bars
// come from GET /api/jobs/running, the server's cache read after each fleet
// reading (watching the inbox from here would have the server ask every box).
func (a *App) barInput() barInput {
	in := barInput{Fleet: a.fleet, FleetStale: a.conn != apiclient.Live}
	if !a.fleetHas {
		in.Fleet = api.Fleet{}
	}
	if js, ok := a.screens[scrJobs].(*jobsScreen); ok && js.has && a.active == scrJobs {
		in.Jobs, in.JobsHas, in.JobsStale = js.in.Jobs, true, js.conn != apiclient.Live
	} else if a.runningHas {
		in.Jobs, in.JobsHas, in.JobsStale = runningViews(a.running), true, a.runningErr != nil || in.FleetStale
	}
	return in
}

// runningViews is the running jobs in the inbox's shape, as much of it as
// drawerBars reads: target, kind, state, step, and the step count.
func runningViews(r api.RunningJobs) []api.JobView {
	out := make([]api.JobView, 0, len(r.Jobs))
	for _, j := range r.Jobs {
		out = append(out, api.JobView{Target: j.Target, ID: j.ID, Kind: j.Kind, State: j.State, Step: j.Step,
			Steps: make([]api.JobStepView, max(j.Total, 0))})
	}
	return out
}

// barCap is how many bars the drawer shows: 3 compact, 6 boxed.
func (a *App) barCap() int {
	if a.h >= drawerTallRows {
		return 6
	}
	return 3
}

// drawerLines is the open drawer's lines: its bars past the cap folded
// into "+N more", or the one line that says there are none.
func (a *App) drawerLines() int {
	bs, more := capBars(drawerBars(a.barInput()), a.barCap())
	if more > 0 {
		return len(bs) + 1
	}
	return max(len(bs), 1)
}

// barLine is one bar, w cells: what, where, how far, then its marks.
func (a *App) barLine(b progressBar, w int) string {
	word := map[barKind]string{barDecision: "job", barJob: "job", barSync: "sync", barDisk: "disk"}[b.Kind]
	what := b.What
	if b.Kind == barDisk {
		what = ""
	}
	cells := 20
	if w < 100 {
		cells = 10
	}
	l := fmt.Sprintf("%-4s %-10s %-14s %-12s [%s]", word, truncate(b.Box, 10, a.gl.Ellipsis), truncate(what, 14, a.gl.Ellipsis),
		truncate(b.Note, 12, a.gl.Ellipsis), bar(b.Done, 0, b.Total, cells, a.gl))
	if b.Kind == barSync {
		l += fmt.Sprintf(" %.1f%%", float64(b.Done)*100/float64(b.Total))
	}
	if b.Kind == barDecision {
		l += " " + a.th.Warn.Render(a.gl.Flag+" needs decision")
	}
	if b.Stale {
		l += " " + a.th.Warn.Render("~ stale")
	}
	return truncate(l, w, a.gl.Ellipsis)
}

// activity is the open drawer's lines, w cells each: the bars, "+N more"
// past the cap, or a line that says there are none.
func (a *App) activity(w int) []string {
	bs, more := capBars(drawerBars(a.barInput()), a.barCap())
	var out []string
	for _, b := range bs {
		out = append(out, a.barLine(b, w))
	}
	if more > 0 {
		out = append(out, a.th.Dim.Render(fmt.Sprintf("+%d more", more)))
	}
	if len(out) == 0 {
		l := "no progress to show " + a.gl.Sep + " " + a.jobsRunning()
		if n := a.needCount(); n > 0 {
			l += "  " + a.th.Warn.Render(fmt.Sprintf("[%s %d needs decision]", a.gl.Flag, n))
		}
		if a.active != scrJobs && a.jobsRunning() != "jobs n/a" && a.jobsRunning() != "jobs 0 running" {
			l += "  " + a.th.Dim.Render("(2 Jobs: steps)")
		}
		out = append(out, truncate(l, w, a.gl.Ellipsis))
	}
	return out
}

// drawerView is the open drawer in r: boxed with the title Activity, or
// unboxed lines when compact. The flash, when there is one, takes the first
// line.
func (a *App) drawerView(r rect, compact bool) string {
	iw := r.W - 1
	if !compact {
		iw = r.W - 3
	}
	lines := a.activity(iw)
	if a.flash != "" {
		lines = append([]string{truncate(a.flash, iw, a.gl.Ellipsis)}, lines...)
	}
	room := r.H
	if !compact {
		room = max(r.H-2, 1)
	}
	if len(lines) > room {
		lines = lines[:room]
	}
	for i := range lines {
		lines[i] = " " + lines[i]
	}
	if compact {
		return fit(strings.Join(lines, "\n"), r.W, r.H, a.gl.Ellipsis)
	}
	return a.boxed("Activity", false, strings.Join(lines, "\n"), r.W, r.H)
}
