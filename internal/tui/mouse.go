package tui

import (
	"regexp"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

// Mouse: a click maps onto the text each region drew last (a.shown). Panes
// are plain text, so a "[k] label" button is found in the line itself. Mouse
// input arrives only when the Mouse display setting is on.

// doubleClick is how close two clicks on one cell must be to open.
const doubleClick = 400 * time.Millisecond

// modalFlush drops keys that arrive this soon after a modal opens: keys typed
// ahead of the modal must not answer it. Tests set it to 0.
var modalFlush = 150 * time.Millisecond

// pane is one region as last drawn: its title, where its body starts on
// screen, and its body lines without colour.
type pane struct {
	title string
	x, y  int
	lines []string
}

// tileHit is one KPI tile's columns, for clicks on the strip.
type tileHit struct {
	x0, x1 int
	title  string
}

// keep keeps what a region drew, for the mouse and the tests.
func (a *App) keep(name, title string, x, y int, body string) {
	if a.shown == nil {
		a.shown = map[string]pane{}
	}
	lines := strings.Split(ansi.Strip(body), "\n")
	a.shown[name] = pane{title: title, x: x, y: y, lines: lines}
}

// keepBoxed keeps a boxed region's body: the box less its border.
func (a *App) keepBoxed(name, title string, r rect, drawn string) {
	ls := strings.Split(drawn, "\n")
	var body []string
	for i := 1; i < len(ls)-1; i++ {
		body = append(body, ansi.Cut(ls[i], 1, r.W-1))
	}
	a.keep(name, title, r.X+1, r.Y+1, strings.Join(body, "\n"))
}

var buttonRE = regexp.MustCompile(`\[([^\]\n]+)\]`)

// buttonKey turns a button's label ("u", "esc", "enter") into a key press.
func buttonKey(label string) (tea.KeyPressMsg, bool) {
	f := strings.Fields(label)
	if len(f) == 0 {
		return tea.KeyPressMsg{}, false
	}
	w := f[0]
	switch strings.ToLower(w) {
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEscape}, true
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}, true
	case "tab":
		return tea.KeyPressMsg{Code: tea.KeyTab}, true
	case "space":
		return tea.KeyPressMsg{Code: ' ', Text: " "}, true
	}
	r := []rune(w)
	if len(r) != 1 {
		return tea.KeyPressMsg{}, false // "1-5", "Overview", "####|...": not a key
	}
	return tea.KeyPressMsg{Code: r[0], Text: w}, true
}

// buttonAt is the key of the "[k] label" button under column col (in
// cells). A button reaches from its "[" to the next button or a run of 3+
// spaces.
func buttonAt(line string, col int) (tea.KeyPressMsg, bool) {
	s := line
	ms := buttonRE.FindAllStringSubmatchIndex(s, -1)
	cell := func(byteOff int) int { return ansi.StringWidth(s[:byteOff]) }
	for i, m := range ms {
		start, end := m[0], m[1]
		next := len(s)
		if i+1 < len(ms) {
			next = ms[i+1][0]
		}
		stop := next
		if gap := strings.Index(s[end:next], "   "); gap >= 0 {
			stop = end + gap
		}
		if col >= cell(start) && col < cell(stop) {
			return buttonKey(s[m[2]:m[3]])
		}
	}
	return tea.KeyPressMsg{}, false
}

// selLine is the line the screen marks as selected: its mark (gl.Sel) is
// the first thing on it, after at most two spaces.
func (a *App) selLine(lines []string) int {
	for i, l := range lines {
		t := strings.TrimLeft(l, " ")
		if len(l)-len(t) <= 2 && strings.HasPrefix(t, a.gl.Sel) {
			return i
		}
	}
	return -1
}

// inList: row is in the same run of list lines as the selected line sel.
func inList(lines []string, sel, row int) bool {
	isRow := func(i int) bool {
		return i >= 0 && i < len(lines) && strings.TrimSpace(lines[i]) != ""
	}
	lo, hi := min(sel, row), max(sel, row)
	for i := lo; i <= hi; i++ {
		if !isRow(i) {
			return false
		}
	}
	return sel >= 0
}

func in(x, y int, r rect) bool { return x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H }

// press runs a key as if typed.
func (a *App) press(k tea.KeyPressMsg) tea.Cmd {
	a.flash = ""
	return a.key(k)
}

// click handles a left click: it focuses the region under it and acts.
func (a *App) click(m tea.MouseClickMsg) tea.Cmd {
	if !a.prefs.Mouse || m.Button != tea.MouseLeft || a.w < minW || a.h < minH {
		return nil
	}
	now := a.clock()
	double := a.lastClick.x == m.X && a.lastClick.y == m.Y && now.Sub(a.lastClick.at) <= doubleClick
	a.lastClick.x, a.lastClick.y, a.lastClick.at = m.X, m.Y, now
	if double {
		a.lastClick.at = time.Time{} // a third click starts over
	}
	l := a.layout()
	if a.modal != nil {
		return a.clickModal(m.X, m.Y)
	}
	switch {
	case m.Y == 0:
		return a.clickHeader(m.X)
	case in(m.X, m.Y, l.KPI):
		return a.clickKPI(m.X)
	case m.Y == a.h-1:
		return a.clickLast(m.X, l)
	case l.DrawerOpen && in(m.X, m.Y, l.Drawer):
		return a.toggleDrawer()
	case a.treeOver && in(m.X, m.Y, rect{l.Main.X, l.Main.Y, min(overlayWidth, l.Main.W), l.Main.H}):
		a.focus = focusTree
		return a.clickTree(m.Y-l.Main.Y-1, double)
	case a.logOver && in(m.X, m.Y, rect{l.Main.X + l.Main.W - min(overlayWidth, l.Main.W), l.Main.Y, min(overlayWidth, l.Main.W), l.Main.H}):
		a.focus = focusLog
		return nil
	case l.Left.W > 0 && in(m.X, m.Y, l.Left):
		a.focus = focusTree
		row := m.Y - l.Left.Y - 1
		if row >= 0 && row < len(screenNames) {
			return a.openScreen(screenID(row))
		}
		return a.clickTree(row-len(screenNames)-1, double)
	case l.Right.W > 0 && in(m.X, m.Y, l.Right):
		a.focus = focusLog
		return nil
	case in(m.X, m.Y, l.Main):
		a.focus = focusMain
		return a.clickMain(m.X, m.Y, double)
	}
	return nil
}

// clickModal: a modal with a text field (the typed confirm, the passphrase)
// takes no clicks, so a click can never confirm or submit it. Other modals
// take a click on a [k] button as the key, once the flush window has passed.
func (a *App) clickModal(x, y int) tea.Cmd {
	if _, typed := a.modal.(pasteTaker); typed || a.flushing() {
		return nil
	}
	p, ok := a.shown["main"]
	if !ok || y < p.y || y-p.y >= len(p.lines) {
		return nil
	}
	if k, ok := buttonAt(p.lines[y-p.y], x-p.x); ok {
		return a.press(k)
	}
	return nil
}

// clickHeader: a view tab (below 100 columns) opens its view, LOCKED opens
// the unlock prompt, the alert count opens the fleet.
func (a *App) clickHeader(x int) tea.Cmd {
	p, ok := a.shown["header"]
	if !ok || len(p.lines) == 0 {
		return nil
	}
	line := p.lines[0]
	w := wordAt(line, x)
	for i, n := range screenNames {
		if w == n || w == "["+n+"]" {
			return a.openScreen(screenID(i))
		}
	}
	switch {
	case strings.HasPrefix(w, "LOCKED"):
		a.modal = newUnlock()
	case w == "alerts":
		return a.openScreen(scrFleet)
	}
	return nil
}

// clickKPI opens the view behind a tile: Gateways, else the fleet.
func (a *App) clickKPI(x int) tea.Cmd {
	for _, t := range a.kpiHits {
		if x >= t.x0 && x < t.x1 {
			if t.title == "Gateways" {
				return a.openScreen(scrGateways)
			}
			return a.openScreen(scrFleet)
		}
	}
	return nil
}

// clickLast is the bottom line: the minimized drawer's counts toggle the
// drawer (its decision count opens Jobs); a [k] hint presses k.
func (a *App) clickLast(x int, l layout) tea.Cmd {
	p, ok := a.shown["hints"]
	if !ok || len(p.lines) == 0 {
		return nil
	}
	line := p.lines[0]
	if !l.DrawerOpen {
		if i := strings.Index(line, " | "); i >= 0 && x < ansi.StringWidth(line[:i]) {
			left := line[:i]
			if d := strings.Index(left, "needs decision]"); d >= 0 && x >= ansi.StringWidth(left[:strings.LastIndex(left[:d], "[")]) &&
				x < ansi.StringWidth(left[:d+len("needs decision]")]) {
				return a.openScreen(scrJobs)
			}
			return a.toggleDrawer()
		}
	}
	if k, ok := buttonAt(line, x); ok {
		return a.press(k)
	}
	return nil
}

// clickTree selects tree line i (from the top of the drawn tree); a double
// click opens it.
func (a *App) clickTree(i int, double bool) tea.Cmd {
	ns := a.treeNodes()
	j := a.treeFirst + i
	if i < 0 || j >= len(ns) {
		return nil
	}
	a.treeSelect(ns, j)
	if double {
		cmd, _ := a.treeKey(tea.KeyPressMsg{Code: tea.KeyEnter})
		return cmd
	}
	return nil
}

// clickMain: a [k] button presses k; on a list, one click selects the row
// and a double click opens it. A screen with a text field takes no clicks.
func (a *App) clickMain(x, y int, double bool) tea.Cmd {
	cur := a.current()
	if cur.capturing() {
		return nil
	}
	if fs, ok := cur.(*fleetScreen); ok {
		if row, ok := fs.clickRow(a, x, y); ok {
			fs.cursor = row
			fs.clamp(a)
			if double {
				return a.press(tea.KeyPressMsg{Code: tea.KeyEnter})
			}
			return nil
		}
	}
	p, ok := a.shown["main"]
	if !ok {
		return nil
	}
	row, col := y-p.y, x-p.x
	if row < 0 || row >= len(p.lines) {
		return nil
	}
	if k, ok := buttonAt(p.lines[row], col); ok {
		return a.press(k)
	}
	if _, fleet := cur.(*fleetScreen); fleet && !cur.(*fleetScreen).hostMode {
		return nil // the table's rows are clickRow's
	}
	sel := a.selLine(p.lines)
	if sel < 0 || !inList(p.lines, sel, row) {
		return nil
	}
	step := tea.KeyPressMsg{Code: tea.KeyDown}
	if row < sel {
		step = tea.KeyPressMsg{Code: tea.KeyUp}
	}
	var cmds []tea.Cmd
	for i := 0; i < abs(row-sel); i++ {
		cmds = append(cmds, cur.update(a, step))
	}
	if double {
		cmds = append(cmds, a.press(tea.KeyPressMsg{Code: tea.KeyEnter}))
	}
	return tea.Batch(cmds...)
}

// wheel scrolls the region under the pointer.
func (a *App) wheel(m tea.MouseWheelMsg) tea.Cmd {
	if !a.prefs.Mouse || a.modal != nil || a.w < minW || a.h < minH {
		return nil
	}
	up := m.Button == tea.MouseWheelUp
	if !up && m.Button != tea.MouseWheelDown {
		return nil
	}
	step := tea.KeyPressMsg{Code: tea.KeyDown}
	if up {
		step = tea.KeyPressMsg{Code: tea.KeyUp}
	}
	l := a.layout()
	switch {
	case (l.Right.W > 0 && in(m.X, m.Y, l.Right)) || (a.logOver && in(m.X, m.Y, l.Main) && m.X >= l.Main.X+l.Main.W-min(overlayWidth, l.Main.W)):
		if up {
			a.logOff = max(a.logOff-1, 0)
		} else {
			a.logOff++
		}
	case (l.Left.W > 0 && in(m.X, m.Y, l.Left)) || (a.treeOver && in(m.X, m.Y, l.Main) && m.X < l.Main.X+min(overlayWidth, l.Main.W)):
		cmd, _ := a.treeKey(step)
		return cmd
	case in(m.X, m.Y, l.Main):
		if cur := a.current(); !cur.capturing() {
			return cur.update(a, step)
		}
	}
	return nil
}

// flushing: a modal opened too recently to take keys.
func (a *App) flushing() bool {
	return a.modal != nil && a.clock().Sub(a.modalAt) < modalFlush
}

// wordAt is the space-separated word under column x.
func wordAt(line string, x int) string {
	col := 0
	for _, w := range strings.Split(line, " ") {
		n := ansi.StringWidth(w)
		if x >= col && x < col+n {
			return w
		}
		col += n + 1
	}
	return ""
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
