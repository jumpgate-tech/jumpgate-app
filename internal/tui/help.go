package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

// helpModal lists the global keys and the current screen's, generated from
// the same bindings the screens match on. F opens the full guide from here.
type helpModal struct {
	lines  []string
	offset int
}

// The help row for the keys that move: ${arrows} and h j k l.
const (
	movingLabel = "arrows/hjkl"
	movingHelp  = "move; left and right switch tabs"
)

// isChange: a key that changes something on a box, the server or the vault
// is a Ctrl chord; a plain key only looks and moves. ctrl+c quits and is
// not a change.
func isChange(b key.Binding) bool {
	ks := b.Keys()
	return len(ks) > 0 && strings.HasPrefix(ks[0], "ctrl+") && b.Help() != globalKeys.ForceQuit.Help()
}

// helpGroups are rows that stand for several keys, so the list fits 80x24.
func helpGroups() []struct {
	label, desc string
	members     []key.Binding
} {
	return []struct {
		label, desc string
		members     []key.Binding
	}{
		{movingLabel, movingHelp, []key.Binding{navKeys.Up, navKeys.Down, navKeys.Left, navKeys.Right}},
		{"[ ] .", "tree, activity log, activity drawer", []key.Binding{globalKeys.Panels, globalKeys.Drawer}},
		{"q ctrl+c", "quit; ctrl+c quits at once", []key.Binding{globalKeys.Quit, globalKeys.ForceQuit}},
		{"g i", "setup guide; invite a box (one line to run on it)", []key.Binding{globalKeys.Guide, globalKeys.Invite}},
		{"? :", "help; take a command", []key.Binding{globalKeys.Help, globalKeys.Palette}},
		{"enter /", "open; filter", []key.Binding{navKeys.Enter, navKeys.Filter}},
	}
}

// newHelp lists the keys in two groups: "look" (plain keys, which only look
// and move) and "change" (Ctrl chords, which change a box, the server or the
// vault). It is generated from the bindings the screens match on.
func newHelp(a *App) *helpModal {
	row := func(label, desc string) string { return fmt.Sprintf("  %-12s %s", keyLabel(label, a.gl), desc) }
	var look, change []string
	listed := map[key.Help]bool{}
	emitted := map[int]bool{}
	for _, b := range append(keyList(globalKeys), keyList(navKeys)...) {
		if !a.offers(b) {
			continue
		}
		listed[b.Help()] = true
		grouped := false
		for i, g := range helpGroups() {
			for _, m := range g.members {
				if m.Help() == b.Help() {
					grouped = true
					if !emitted[i] {
						emitted[i] = true
						look = append(look, row(g.label, g.desc))
					}
				}
			}
		}
		switch {
		case grouped:
		case isChange(b):
			change = append(change, row(b.Help().Key, b.Help().Desc))
		default:
			look = append(look, row(b.Help().Key, b.Help().Desc))
		}
	}
	// A screen's own keys, less the ones already listed above.
	for _, b := range a.current().keys() {
		if listed[b.Help()] {
			continue
		}
		listed[b.Help()] = true
		if isChange(b) {
			change = append(change, row(b.Help().Key, b.Help().Desc))
		} else {
			look = append(look, row(b.Help().Key, b.Help().Desc))
		}
	}
	lines := append([]string{"look (plain keys)"}, look...)
	if len(change) > 0 {
		lines = append(lines, "change (Ctrl chords)")
		lines = append(lines, change...)
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
	// The guide and invite keys are always listed: they are how a new person
	// finds the way in, and they work at any time.
	case globalKeys.Guide.Help(), globalKeys.Invite.Help():
		return true
	case globalKeys.Copy.Help():
		return a.inviteOn && a.invite != ""
	case globalKeys.Lock.Help():
		return a.vaultOpen && !(a.fleetHas && a.fleet.Locked)
	}
	return b.Enabled()
}

func (h *helpModal) key(a *App, k tea.KeyPressMsg) (bool, tea.Cmd) {
	if key.Matches(k, manualKeys.Open) {
		a.modal, _ = newManual("")
		return false, nil
	}
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
	const close = "  ([F] the full guide  [esc] close)  ~ means stale"
	head := a.th.Title.Render(" Help") + a.th.Dim.Render(close)
	lines := h.lines
	if len(lines)+2 > height {
		lines = lines[:0:0]
		for _, l := range h.lines {
			if l != "" {
				lines = append(lines, l)
			}
		}
		// Still too tall: the first heading shares the title's line.
		if len(lines)+1 > height && len(lines) > 0 && strings.HasPrefix(lines[0], "look") {
			head = a.th.Title.Render(" Help") + ": " + lines[0] + a.th.Dim.Render(close)
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
