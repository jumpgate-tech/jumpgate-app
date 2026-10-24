package tui

import (
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/valve-tech/jumpgate/internal/api"
)

// Focus regions, the tree and log overlays, and the activity log's feed
// (spec §4, U6).

type focusRegion int

const (
	focusMain focusRegion = iota
	focusTree
	focusLog
)

func (f focusRegion) String() string { return [...]string{"main", "tree", "log"}[f] }

// overlayWidth is how wide the tree and log are when drawn over main.
const overlayWidth = 34

func (a *App) layout() layout {
	return layoutFor(a.w, a.h, drawerFromPref(a.prefs.Drawer), a.drawerLines())
}

// regions are the focusable regions showing now, in tab order.
func (a *App) regions() []focusRegion {
	l := a.layout()
	rs := []focusRegion{focusMain}
	if l.Left.W > 0 || a.treeOver {
		rs = append(rs, focusTree)
	}
	if l.Right.W > 0 || a.logOver {
		rs = append(rs, focusLog)
	}
	return rs
}

// fitFocus drops overlays the window now has room for and moves the focus
// off a region that no longer shows.
func (a *App) fitFocus() {
	l := a.layout()
	if l.Left.W > 0 {
		a.treeOver = false
	}
	if l.Right.W > 0 {
		a.logOver = false
	}
	for _, r := range a.regions() {
		if r == a.focus {
			return
		}
	}
	a.focus = focusMain
}

// regionKey handles the keys of the focus regions; ok is false for a key
// they leave to the screen.
func (a *App) regionKey(k tea.KeyPressMsg) (tea.Cmd, bool) {
	l := a.layout()
	switch {
	case key.Matches(k, globalKeys.Focus):
		rs := a.regions()
		i := 0
		for j, r := range rs {
			if r == a.focus {
				i = j
			}
		}
		step := 1
		if k.String() == "shift+tab" {
			step = len(rs) - 1
		}
		a.focus = rs[(i+step)%len(rs)]
		return nil, true
	case k.String() == "[":
		if l.Left.W == 0 {
			a.treeOver = !a.treeOver
			a.logOver = false
		}
		a.focus = focusTree
		if l.Left.W == 0 && !a.treeOver {
			a.focus = focusMain
		}
		return nil, true
	case k.String() == "]":
		if l.Right.W == 0 {
			a.logOver = !a.logOver
			a.treeOver = false
		}
		a.focus = focusLog
		if l.Right.W == 0 && !a.logOver {
			a.focus = focusMain
		}
		return nil, true
	case k.String() == "esc" && (a.treeOver || a.logOver):
		a.treeOver, a.logOver, a.focus = false, false, focusMain
		return nil, true
	}
	switch a.focus {
	case focusTree:
		return a.treeKey(k)
	case focusLog:
		switch {
		case key.Matches(k, navKeys.Up):
			a.logOff = max(a.logOff-1, 0)
			return nil, true
		case key.Matches(k, navKeys.Down):
			a.logOff++ // clamped when drawn
			return nil, true
		}
	}
	return nil, false
}

// treeKey moves the tree (and with it the table), expands, and opens.
func (a *App) treeKey(k tea.KeyPressMsg) (tea.Cmd, bool) {
	ns := a.treeNodes()
	i := a.treeSelected(ns)
	switch {
	case key.Matches(k, navKeys.Up):
		a.treeSelect(ns, max(i-1, 0))
	case key.Matches(k, navKeys.Down):
		a.treeSelect(ns, min(i+1, len(ns)-1))
	case k.String() == "space":
		if n := ns[i]; n.Box != "" {
			a.treeSelect(ns, i)
			return a.toggleExpand(n.Box), true
		}
	case key.Matches(k, navKeys.Enter):
		n := ns[i]
		if n.Box == "" {
			return a.openScreen(scrFleet), true
		}
		a.treeOver, a.focus = false, focusMain
		cmd := a.openHost(n.Box)
		a.selNode = n.Node
		return cmd, true
	default:
		return nil, false
	}
	return nil, true
}

// treeSelect makes line i the selection, and the table's.
func (a *App) treeSelect(ns []treeNode, i int) {
	if i < 0 || i >= len(ns) {
		return
	}
	a.treeCur = i
	n := ns[i]
	a.logOff = 0
	if n.Kind == "fleet" {
		a.selAll, a.selNode = true, ""
		return
	}
	a.selAll, a.sel, a.selNode = false, n.Box, n.Node
	if fs, ok := a.screens[scrFleet].(*fleetScreen); ok {
		fs.selectID(a, n.Box)
	}
}

// snapshot is what the App holds now, for the log.
func (a *App) snapshot() snapshot {
	s := snapshot{Fleet: a.fleet, FleetHas: a.fleetHas, Gateways: a.gws, GatewaysHas: a.gwsHas, Now: a.now(), Modules: map[string]api.ModulesView{}}
	if fs, ok := a.screens[scrFleet].(*fleetScreen); ok {
		s.Targets, s.TargetsHas = fs.hosts.ts, fs.hosts.has
	}
	if js, ok := a.screens[scrJobs].(*jobsScreen); ok && js.has {
		s.Jobs, s.JobsHas = js.in.Jobs, true
	}
	for box, m := range a.mods {
		if a.expanded[box] && m.has {
			s.Modules[box] = m.view
		}
	}
	return s
}

// record logs what changed since the last snapshot.
func (a *App) record() {
	next := a.snapshot()
	a.log.add(diffEvents(a.prev, next)...)
	a.prev = next
}

// logScope is the box and node the log shows: the fleet off the Fleet view
// and box detail, or when the tree's fleet line is selected.
func (a *App) logScope() (box, node string) {
	if a.selAll || (a.detail == nil && a.active != scrFleet) || a.sel == "" {
		return "", ""
	}
	return a.sel, a.selNode
}

func (a *App) sevGlyph(s logSev) string {
	switch s {
	case logOK:
		return a.th.OK.Render(a.gl.OK)
	case logWarn:
		return a.th.Warn.Render(a.gl.Warn)
	case logFail:
		return a.th.Bad.Render(a.gl.Fail)
	}
	return a.th.Dim.Render(a.gl.Info)
}

// logView is the activity log, boxed, w×h: newest first, each line its
// severity glyph, its age, the box (fleet-wide) and the text.
func (a *App) logView(w, h int) string {
	box, node := a.logScope()
	title := "Log: fleet"
	if box != "" {
		title = "Log: " + sanitizeLine(box)
		if node != "" {
			title += " " + node
		}
	}
	evs := a.log.scoped(box, node)
	room := max(h-2, 1)
	a.logOff = min(a.logOff, max(len(evs)-room, 0))
	var lines []string
	if len(evs) == 0 {
		lines = append(lines, a.th.Dim.Render(" no activity yet"))
	}
	for _, e := range evs[a.logOff:] {
		l := " " + a.sevGlyph(e.Sev) + " " + a.th.Dim.Render(pad(formatAge(a.now().Sub(e.At)), 3)) + " "
		if box == "" && e.Box != "" {
			l += e.Box + " "
		}
		lines = append(lines, truncate(l+e.Text, w-2, a.gl.Ellipsis))
	}
	return a.boxed(title, a.focus == focusLog, strings.Join(lines, "\n"), w, h)
}

// overlay draws panel over base from column x; base lines are padded to w.
func overlay(base, panel string, x, w int) string {
	bl, pl := strings.Split(base, "\n"), strings.Split(panel, "\n")
	for i := range bl {
		if i >= len(pl) {
			break
		}
		line := pad(bl[i], w)
		pw := ansi.StringWidth(pl[i])
		bl[i] = ansi.Cut(line, 0, x) + pl[i] + ansi.Cut(line, x+pw, w)
	}
	return strings.Join(bl, "\n")
}
