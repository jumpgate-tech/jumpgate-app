package tui

// drawerState is the activity drawer's saved choice (UIPrefs.Drawer).
type drawerState int

const (
	drawerAuto drawerState = iota // minimized when the window is short
	drawerOpen
	drawerMin
)

// Layout breakpoints.
const (
	leftMinWidth     = 100 // at or above, a left sidebar replaces the header's view tabs
	leftWidth        = 22  // narrower (down to leftMinW) when main needs the room
	leftMinW         = 18
	drawerTallRows   = 34  // auto opens, and an open drawer is boxed, at or above
	drawerMaxLines   = 7   // boxed: 6 bars + "+N more"
	drawerCompactMax = 4   // compact: 3 bars + "+N more"
	kpiBoxedMinRows  = 30  // at or above, the KPI strip is boxed tiles
	kpiBoxedRows     = 4   // 2 borders + 2 lines
	rightMinWidth    = 140 // at or above, the activity log is a right sidebar; below, ] overlays it
	rightMinW        = 28
	rightMaxW        = 40
	mainMinW         = 50 // main never gets narrower than this
	mainFleetW       = 82 // boxed main that holds the default fleet columns (80 cells)
)

// drawerFromPref reads a stored pref; anything unknown is auto.
func drawerFromPref(s string) drawerState {
	switch s {
	case "open":
		return drawerOpen
	case "min":
		return drawerMin
	}
	return drawerAuto
}

// pref is the stored form of d.
func (d drawerState) pref() string {
	switch d {
	case drawerOpen:
		return "open"
	case drawerMin:
		return "min"
	}
	return "auto"
}

func (d drawerState) String() string { return d.pref() }

// rect is a screen region in cells.
type rect struct{ X, Y, W, H int }

// layout is the dashboard's regions for one window size.
type layout struct {
	Header        rect // always 1 row at Y=0; below 100 columns it carries the view tabs
	KPI           rect // at Y=1: 1 row when h < 30, boxed tiles (4 rows) at h >= 30
	Left          rect // views list + tree, W=22 (18-22) when w >= 100, else zero
	Main          rect
	Right         rect // the activity log, 20% of w (28-40) when w >= 140, else zero
	Drawer        rect // activity rows; H=0 when minimized
	Hints         rect // 1 row, last line (minimized: the status line)
	DrawerOpen    bool // resolved: auto -> h >= 34
	DrawerCompact bool // open but h < 34: up to 4 activity lines, no box
}

// layoutFor splits a w x h window (at least 80x24) into regions; lines is
// how many activity lines an open drawer has to show. The skew banner is not
// a region: render takes one row from Main for it.
func layoutFor(w, h int, d drawerState, lines int) layout {
	var l layout
	l.Header = rect{0, 0, w, 1}
	l.KPI = rect{0, 1, w, 1}
	if h >= kpiBoxedMinRows {
		l.KPI.H = kpiBoxedRows
	}
	y := 1 + l.KPI.H
	switch d {
	case drawerOpen:
		l.DrawerOpen = true
	case drawerAuto:
		l.DrawerOpen = h >= drawerTallRows
	}
	drawerH := 0
	if l.DrawerOpen {
		if h >= drawerTallRows {
			drawerH = 2 + min(max(lines, 1), drawerMaxLines)
		} else {
			l.DrawerCompact = true
			drawerH = min(max(lines, 1), drawerCompactMax)
		}
	}
	l.Hints = rect{0, h - 1, w, 1}
	bodyH := h - 1 - drawerH - y
	if drawerH > 0 {
		l.Drawer = rect{0, y + bodyH, w, drawerH}
	}
	leftW := 0
	if w >= leftMinWidth {
		leftW = min(max(w-mainFleetW, leftMinW), leftWidth)
		l.Left = rect{0, y, leftW, bodyH}
	}
	rightW := 0
	if w >= rightMinWidth {
		rightW = min(max(w/5, rightMinW), rightMaxW)
		if w-leftW-rightW < mainFleetW { // the region the rule added folds
			rightW = 0
		}
	}
	if rightW > 0 {
		l.Right = rect{w - rightW, y, rightW, bodyH}
	}
	l.Main = rect{leftW, y, w - leftW - rightW, bodyH}
	return l
}
