package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

// helpModal lists the global keys and the current screen's, generated from
// the same bindings the screens match on.
type helpModal struct{ lines []string }

func newHelp(a *App) *helpModal {
	row := func(b key.Binding) string {
		return fmt.Sprintf("  %-12s %s", keyLabel(b.Help().Key, a.gl), b.Help().Desc)
	}
	lines := []string{"keys everywhere"}
	for _, b := range append(keyList(globalKeys), keyList(navKeys)...) {
		if a.offers(b) {
			lines = append(lines, row(b))
		}
	}
	if ks := a.current().keys(); len(ks) > 0 {
		lines = append(lines, "", "keys here")
		for _, b := range ks {
			lines = append(lines, row(b))
		}
	}
	return &helpModal{lines: lines}
}

// offers reports whether b can act now. Restart acts only on a skewed server
// when cmd/jumpgate gave a way to restart it (spec D30); every other global
// key always can.
func (a *App) offers(b key.Binding) bool {
	if b.Help() == globalKeys.Restart.Help() {
		return a.canRestart()
	}
	return b.Enabled()
}

func (h *helpModal) key(_ *App, k tea.KeyPressMsg) (bool, tea.Cmd) {
	return k.String() == "esc" || k.String() == "?" || k.String() == "q", nil
}

func (h *helpModal) view(a *App, w, _ int) string {
	return a.th.Title.Render(" Help") + a.th.Dim.Render("  (esc to close)") + "\n\n" + strings.Join(h.lines, "\n")
}
