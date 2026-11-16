package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/valve-tech/jumpgate/docs/guide"
)

// The full user guide ("the manual" here, so the names stay clear of guide.go,
// the first-run setup checklist). F opens it from the ? key list, and :help
// opens it from the palette; H is not used, since two views bind it (host
// keys, this machine). On a page the keys below scroll and turn it. The
// guide's own TUI page names these, and a test holds it to them.
var manualKeys = struct {
	Open, PageUp, PageDown, Top, Bottom, Topics key.Binding
}{
	Open:     key.NewBinding(key.WithKeys("F"), key.WithHelp("F", "the full guide (from ?)")),
	PageUp:   key.NewBinding(key.WithKeys("pgup", "b"), key.WithHelp("pgup/b", "page up")),
	PageDown: key.NewBinding(key.WithKeys("pgdown", "space"), key.WithHelp("pgdn/space", "page down")),
	Top:      key.NewBinding(key.WithKeys("home", "g"), key.WithHelp("g", "top")),
	Bottom:   key.NewBinding(key.WithKeys("end", "G"), key.WithHelp("G", "bottom")),
	Topics:   key.NewBinding(key.WithKeys("esc", "backspace"), key.WithHelp("esc", "back to the topics")),
}

// manualSource is the guide the TUI shows.
var manualSource = guide.Embedded()

// manualModal is the user guide (docs/guide): a list of topics, and each
// topic as a page that scrolls. It reads only the embedded files.
type manualModal struct {
	topics []guide.Topic
	cur    int // the topic selected in the list, or shown on the page
	open   bool
	offset int
	room   int // the page's text rows at the last draw, for paging

	// The rendered page, kept until the topic, width or glyphs change.
	cacheKey string
	lines    []string
}

// newManual opens the guide on the topic list, or on topic's page when one
// is named. ok is false for a name that is not a topic.
func newManual(topic string) (g *manualModal, ok bool) {
	g = &manualModal{topics: manualSource.Topics()}
	if topic == "" {
		return g, true
	}
	for i, t := range g.topics {
		if t.Name == topic {
			g.cur, g.open = i, true
			return g, true
		}
	}
	return nil, false
}

// manualTopicNames is the topics as :help takes them.
func manualTopicNames() string {
	var names []string
	for _, t := range manualSource.Topics() {
		names = append(names, t.Name)
	}
	return strings.Join(names, " ")
}

func (g *manualModal) key(_ *App, k tea.KeyPressMsg) (bool, tea.Cmd) {
	if k.String() == "q" {
		return true, nil
	}
	if !g.open {
		switch {
		case key.Matches(k, globalKeys.Back):
			return true, nil
		case key.Matches(k, navKeys.Up):
			g.cur = max(g.cur-1, 0)
		case key.Matches(k, navKeys.Down):
			g.cur = min(g.cur+1, len(g.topics)-1)
		case key.Matches(k, navKeys.Enter), key.Matches(k, navKeys.Right):
			g.open, g.offset = true, 0
		}
		return false, nil
	}
	page := max(g.room-1, 1)
	switch {
	case key.Matches(k, manualKeys.Topics):
		g.open = false
	case key.Matches(k, navKeys.Up):
		g.offset--
	case key.Matches(k, navKeys.Down):
		g.offset++
	case key.Matches(k, manualKeys.PageUp):
		g.offset -= page
	case key.Matches(k, manualKeys.PageDown):
		g.offset += page
	case key.Matches(k, manualKeys.Top):
		g.offset = 0
	case key.Matches(k, manualKeys.Bottom):
		g.offset = len(g.lines) // clamped when drawn
	case key.Matches(k, navKeys.Left) && g.cur > 0:
		g.cur, g.offset = g.cur-1, 0
	case key.Matches(k, navKeys.Right) && g.cur < len(g.topics)-1:
		g.cur, g.offset = g.cur+1, 0
	}
	g.offset = max(g.offset, 0) // and clamped below the end when drawn
	return false, nil
}

func (g *manualModal) view(a *App, w, h int) string {
	if !g.open {
		return g.listView(a)
	}
	t := g.topics[g.cur]
	if ck := fmt.Sprint(t.Name, w, a.gl.ASCII); ck != g.cacheKey {
		// The head names the topic, so its own "# " title is not drawn again.
		body := t.Body
		if first, rest, ok := strings.Cut(body, "\n"); ok && strings.HasPrefix(first, "# ") {
			body = rest
		}
		g.cacheKey, g.lines = ck, renderMarkdown(body, w-2, a.th, a.gl)
	}
	head := a.th.Title.Render(" Guide "+a.gl.Sep+" "+sanitizeLine(t.Title)) +
		a.th.Dim.Render(fmt.Sprintf("  (%d of %d)", g.cur+1, len(g.topics)))
	g.room = max(h-2, 1) // the head and the footer
	g.offset = min(g.offset, max(len(g.lines)-g.room, 0))
	end := min(g.offset+g.room, len(g.lines))
	body := make([]string, 0, g.room)
	for _, l := range g.lines[g.offset:end] {
		body = append(body, " "+l)
	}
	for len(body) < g.room {
		body = append(body, "")
	}
	where := "all"
	if len(g.lines) > g.room {
		where = fmt.Sprintf("%d%%", end*100/len(g.lines))
	}
	foot := a.th.Dim.Render(fmt.Sprintf(" %s  %s scroll  [space] page  [h] prev  [l] next  [esc] topics  [q] close",
		where, a.gl.UpDown))
	return head + "\n" + strings.Join(body, "\n") + "\n" + foot
}

func (g *manualModal) listView(a *App) string {
	lines := []string{
		a.th.Title.Render(" Guide") + a.th.Dim.Render("  (click a topic or [enter] to read, [esc] close)"),
		"",
	}
	for i, t := range g.topics {
		mark := "  "
		row := fmt.Sprintf("%-20s", sanitizeLine(t.Title))
		if i == g.cur {
			mark = a.gl.Sel + " "
			row = a.th.Sel.Render(row)
		}
		lines = append(lines, "  "+mark+row+a.th.Dim.Render("  :help "+t.Name))
	}
	lines = append(lines, "", a.th.Dim.Render("  ? or :keys lists the keys for the screen you are on"))
	return strings.Join(lines, "\n")
}

// click on a topic opens its page; on a page, the footer's buttons are keys.
func (g *manualModal) click(a *App, row, _ int, _ bool) (done, handled bool, cmd tea.Cmd) {
	if g.open || row < manualFirstRow || row-manualFirstRow >= len(g.topics) {
		return false, false, nil
	}
	g.cur, g.open, g.offset = row-manualFirstRow, true, 0
	return false, true, nil
}

// manualFirstRow is the topic list's first topic: the head and a blank line
// come first.
const manualFirstRow = 2
