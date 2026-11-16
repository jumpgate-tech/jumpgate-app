package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

// menuItem is one action in an actions menu: its words, the key that runs it
// directly (shown as a button; "" when the key depends on where you are), and
// what it does.
type menuItem struct {
	label string
	key   string
	hint  string
	run   func(a *App) tea.Cmd
}

// menuModal is `x`: the actions for the selected item in one filterable
// list. Each entry shows its direct key; enter or a click runs it as that
// key would, so a typed confirmation still follows where the key has one.
type menuModal struct {
	h         *hostScreen // the open box it is for; nil for a screen's menu
	title     string
	all       []menuItem
	cursor    int
	first     int // the first entry drawn, for clicks
	query     string
	filtering bool
}

// menuFirstRow is the drawn row of the first entry: title, blank.
const menuFirstRow = 2

func (m *menuModal) visible() []menuItem {
	if m.query == "" {
		return m.all
	}
	var out []menuItem
	q := strings.ToLower(m.query)
	for _, it := range m.all {
		if strings.Contains(strings.ToLower(it.label+" "+it.hint), q) {
			out = append(out, it)
		}
	}
	return out
}

// runItem closes the menu and runs the entry; a confirmation the entry opens
// takes the menu's place.
func (m *menuModal) runItem(a *App, i int) (bool, tea.Cmd) {
	vis := m.visible()
	if i < 0 || i >= len(vis) {
		return false, nil
	}
	a.modal = nil
	cmd := vis[i].run(a)
	return a.modal == nil, cmd
}

func (m *menuModal) key(a *App, k tea.KeyPressMsg) (bool, tea.Cmd) {
	vis := m.visible()
	switch {
	case k.String() == "esc":
		if m.filtering || m.query != "" {
			m.filtering, m.query, m.cursor = false, "", 0
			return false, nil
		}
		return true, nil
	case key.Matches(k, navKeys.Up) && !(m.filtering && k.String() == "k"):
		m.cursor = max(m.cursor-1, 0)
		return false, nil
	case key.Matches(k, navKeys.Down) && !(m.filtering && k.String() == "j"):
		m.cursor = min(m.cursor+1, max(len(vis)-1, 0))
		return false, nil
	case k.String() == "enter":
		return m.runItem(a, m.cursor)
	}
	if m.filtering {
		switch {
		case k.String() == "backspace":
			if r := []rune(m.query); len(r) > 0 {
				m.query = string(r[:len(r)-1])
			}
		case k.Text != "":
			m.query += sanitizeLine(k.Text)
		}
		m.cursor = 0
		return false, nil
	}
	if k.String() == "/" {
		m.filtering = true
		return false, nil
	}
	// A direct key runs its entry, as it would outside the menu.
	for i, it := range vis {
		if it.key != "" && it.key == k.String() {
			return m.runItem(a, i)
		}
	}
	return false, nil
}

func (m *menuModal) view(a *App, w, h int) string {
	vis := m.visible()
	m.cursor = min(m.cursor, max(len(vis)-1, 0))
	head := " " + a.th.Title.Render(m.title) + a.th.Dim.Render("  [enter] run  [/] filter  [esc] close")
	if m.filtering || m.query != "" {
		head = " " + a.th.Title.Render(m.title) + "  filter: " + sanitizeLine(m.query)
		if m.filtering {
			head += "_"
		}
		head += a.th.Dim.Render("  [esc] clear")
	}
	lines := []string{truncate(head, w, a.gl.Ellipsis), ""}
	room := max(h-menuFirstRow, 1)
	m.first = 0
	if m.cursor >= room {
		m.first = m.cursor - room + 1
	}
	for i := m.first; i < len(vis) && i < m.first+room; i++ {
		it := vis[i]
		mark := " "
		if i == m.cursor {
			mark = a.gl.Sel
		}
		keyCol := "         "
		if it.key != "" {
			keyCol = pad("["+keyLabel(it.key, a.gl)+"]", 9)
		}
		l := fmt.Sprintf(" %s %s %s", mark, keyCol, it.label)
		if it.hint != "" {
			l += a.th.Dim.Render("  " + it.hint)
		}
		if i == m.cursor {
			l = a.th.Sel.Render(ansi.Strip(l))
		}
		lines = append(lines, truncate(l, w, a.gl.Ellipsis))
	}
	if len(vis) == 0 {
		lines = append(lines, " "+a.th.Dim.Render("no action matches"))
	}
	return strings.Join(lines, "\n")
}

// click runs the entry on the row clicked.
func (m *menuModal) click(a *App, row, _ int, _ bool) (done, handled bool, cmd tea.Cmd) {
	if row < menuFirstRow || m.filtering {
		return false, false, nil
	}
	vis := m.visible()
	i := m.first + row - menuFirstRow
	if i >= len(vis) {
		return false, true, nil
	}
	m.cursor = i
	done, cmd = m.runItem(a, i)
	return done, true, cmd
}

// pressFor is the key press a binding's first key stands for.
func pressFor(b key.Binding) (tea.KeyPressMsg, bool) {
	ks := b.Keys()
	if len(ks) == 0 {
		return tea.KeyPressMsg{}, false
	}
	switch ks[0] {
	case "up":
		return tea.KeyPressMsg{Code: tea.KeyUp}, true
	case "down":
		return tea.KeyPressMsg{Code: tea.KeyDown}, true
	case "space":
		return tea.KeyPressMsg{Code: ' ', Text: " "}, true
	}
	return buttonKey(ks[0])
}

// menuSkip are the keys that move rather than act: they are not entries.
func menuSkip(b key.Binding) bool {
	switch b.Help() {
	case navKeys.Up.Help(), navKeys.Down.Help(), navKeys.Left.Help(), navKeys.Right.Help(), globalKeys.Back.Help():
		return true
	}
	return false
}

// screenMenu is the actions menu of the screen with the keyboard: its own
// keys that can act now, each running as its key does.
func (a *App) screenMenu() *menuModal {
	cur := a.current()
	m := &menuModal{title: "ACTIONS for " + screenNames[a.active]}
	if a.detail == nil && a.active == scrFleet {
		if r, ok := a.fleet.Row(a.sel); ok {
			m.title = "ACTIONS for " + sanitizeLine(r.ID)
		}
	}
	seen := map[string]bool{}
	for _, b := range cur.keys() {
		if !b.Enabled() || menuSkip(b) || seen[b.Help().Key] {
			continue
		}
		k, ok := pressFor(b)
		if !ok {
			continue
		}
		seen[b.Help().Key] = true
		h := b.Help()
		m.all = append(m.all, menuItem{label: menuWords(h.Desc), key: h.Key, run: func(a *App) tea.Cmd { return a.press(k) }})
	}
	// Rare actions a screen offers with no key of their own.
	if x, ok := cur.(interface{ menuExtras(a *App) []menuItem }); ok {
		m.all = append(m.all, x.menuExtras(a)...)
	}
	return m
}

// menuWords is a binding's description as an entry: its words up to the
// first parenthesis, so "(Services)" is left to the screen that names it.
func menuWords(desc string) string {
	if i := strings.Index(desc, " ("); i > 0 {
		return desc[:i]
	}
	return desc
}

// yesNoModal asks one plain question for an action that is disruptive but
// can be undone: y does it, anything else cancels. The destructive ones ask
// for a typed name instead (confirm.go).
type yesNoModal struct {
	title, detail, yes string
	run                func() tea.Cmd
}

func newYesNo(title, detail, yes string, run func() tea.Cmd) *yesNoModal {
	return &yesNoModal{title: sanitizeLine(title), detail: sanitize(detail), yes: yes, run: run}
}

func (m *yesNoModal) key(a *App, k tea.KeyPressMsg) (bool, tea.Cmd) {
	switch k.String() {
	case "y":
		return true, m.run()
	case "n", "esc":
		a.flash = "cancelled; nothing was changed"
		return true, nil
	}
	return false, nil
}

func (m *yesNoModal) view(a *App, w, _ int) string {
	return a.th.Warn.Render(" "+m.title) + "\n\n " + m.detail + "\n\n " + "[y] " + m.yes + "  [n] cancel  [esc] cancel"
}

// requestLock locks the vault, after a y/n when that would pause running
// jobs.
func (a *App) requestLock() tea.Cmd {
	if n, known := a.runningJobs(); known && n > 0 {
		word := "jobs are"
		if n == 1 {
			word = "job is"
		}
		a.modal = newYesNo("Lock the vault?", fmt.Sprintf("%d %s running; jobs wait until you unlock the vault again.", n, word), "lock it", a.lockVault)
		return nil
	}
	return a.lockVault()
}

// runningJobs is the fleet-wide count of running jobs, and whether any box
// has reported its jobs.
func (a *App) runningJobs() (n int, known bool) {
	for _, r := range a.fleet.Rows {
		if r.Jobs != nil {
			n, known = n+r.Jobs.Running, true
		}
	}
	if !known && a.runningHas {
		return len(a.running.Jobs), true
	}
	return n, known
}
