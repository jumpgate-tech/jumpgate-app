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
	leftMinWidth    = 100 // at or above, a left sidebar replaces the header's view tabs
	leftWidth       = 24
	drawerTallRows  = 34 // auto opens, and an open drawer is boxed, at or above
	drawerBoxedRows = 5  // 2 borders + 3 activity lines
	kpiBoxedMinRows = 30 // at or above, the KPI strip is boxed tiles
	kpiBoxedRows    = 4  // 2 borders + 2 lines
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
	Left          rect // views list + host list, W=24 when w >= 100, else zero
	Main          rect
	Drawer        rect // activity rows; H=0 when minimized
	Hints         rect // 1 row, last line (minimized: the status line)
	DrawerOpen    bool // resolved: auto -> h >= 34
	DrawerCompact bool // open but h < 34: 1 activity line, no box
}

// layoutFor splits a w x h window (at least 80x24) into regions. The skew
// banner is not a region: render takes one row from Main for it.
func layoutFor(w, h int, d drawerState) layout {
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
			drawerH = drawerBoxedRows
		} else {
			l.DrawerCompact = true
			drawerH = 1
		}
	}
	l.Hints = rect{0, h - 1, w, 1}
	bodyH := h - 1 - drawerH - y
	if drawerH > 0 {
		l.Drawer = rect{0, y + bodyH, w, drawerH}
	}
	leftW := 0
	if w >= leftMinWidth {
		leftW = leftWidth
		l.Left = rect{0, y, leftW, bodyH}
	}
	l.Main = rect{leftW, y, w - leftW, bodyH}
	return l
}
