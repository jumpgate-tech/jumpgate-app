package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

// helpModal lists the global keys and the current screen's, generated from
// the same bindings the screens match on.
type helpModal struct {
	lines  []string
	offset int
}

func newHelp(a *App) *helpModal {
	row := func(b key.Binding) string {
		return fmt.Sprintf("  %-12s %s", keyLabel(b.Help().Key, a.gl), b.Help().Desc)
	}
	lines := []string{"keys everywhere"}
	listed := map[key.Help]bool{}
	for _, b := range append(keyList(globalKeys), keyList(navKeys)...) {
		if a.offers(b) {
			lines = append(lines, row(b))
			listed[b.Help()] = true
		}
	}
	// A screen's own keys, less the ones already listed above.
	var here []string
	for _, b := range a.current().keys() {
		if !listed[b.Help()] {
			here = append(here, row(b))
		}
	}
	if len(here) > 0 {
		lines = append(append(lines, "", "keys here"), here...)
	}
	return &helpModal{lines: lines}
}

// offers reports whether b can act now. Restart acts only on a skewed server
// when cmd/jumpgate gave a way to restart it (spec D30). The vault keys always
// act but are listed only when relevant; every other global key always can.
func (a *App) offers(b key.Binding) bool {
	switch b.Help() {
	case globalKeys.Restart.Help():
		return a.canRestart()
	case globalKeys.Unlock.Help():
		// The vault keys are listed when they are the news, so the help
		// still fits 80x24; both keys work at any time.
		return a.fleetHas && a.fleet.Locked
	case globalKeys.Lock.Help():
		return a.vaultOpen && !(a.fleetHas && a.fleet.Locked)
	}
	return b.Enabled()
}

func (h *helpModal) key(_ *App, k tea.KeyPressMsg) (bool, tea.Cmd) {
	switch k.String() {
	case "esc", "?", "q":
		return true, nil
	case "up", "k":
		h.offset = max(h.offset-1, 0)
	case "down", "j":
		h.offset++ // clamped against the height when drawn
	}
	return false, nil
}

// view drops its blank lines, then puts the first heading on the title's
// line, before it scrolls: at 80x24 the jobs screen's keys only just fit
// once the spacing goes, and a help that hides a key is
// worse than a cramped one. When even that is too tall the keys scroll, and
// a last line says there is more.
func (h *helpModal) view(a *App, w, height int) string {
	head := a.th.Title.Render(" Help") + a.th.Dim.Render("  (esc to close)")
	lines := h.lines
	if len(lines)+2 > height {
		lines = lines[:0:0]
		for _, l := range h.lines {
			if l != "" {
				lines = append(lines, l)
			}
		}
		// Still too tall: the first heading shares the title's line.
		if len(lines)+1 > height && len(lines) > 0 && lines[0] == "keys everywhere" {
			head = a.th.Title.Render(" Help") + ": keys everywhere" + a.th.Dim.Render("  (esc to close)")
			lines = lines[1:]
		}
	} else {
		head += "\n"
	}
	room := height - 1 - strings.Count(head, "\n")
	if len(lines) <= room || room < 2 {
		return head + "\n" + strings.Join(lines, "\n")
	}
	room-- // the "more" line
	h.offset = min(h.offset, len(lines)-room)
	more := a.th.Dim.Render(fmt.Sprintf("  %s more (%s)", a.gl.Ellipsis, a.gl.UpDown))
	return head + "\n" + strings.Join(lines[h.offset:h.offset+room], "\n") + "\n" + more
}
