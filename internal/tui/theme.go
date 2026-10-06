package tui

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/valve-tech/jumpgate/internal/api"
)

// Theme is the TUI's styles. Only the 16 ANSI colours are used (spec D7), so
// the person's terminal theme decides how they look; Bubble Tea drops them
// under NO_COLOR or a dumb terminal. Colour is never the only signal: every
// styled state is also a word or a glyph.
type Theme struct {
	OK, Warn, Bad, Info, Dim, Title, Sel, Key lipgloss.Style
}

func NewTheme() Theme {
	fg := func(c string) lipgloss.Style { return lipgloss.NewStyle().Foreground(lipgloss.Color(c)) }
	return Theme{
		OK: fg("2"), Warn: fg("3"), Bad: fg("1").Bold(true), Info: fg("6"), Dim: fg("8"),
		Title: lipgloss.NewStyle().Bold(true), Sel: lipgloss.NewStyle().Reverse(true), Key: fg("6").Bold(true),
	}
}

// state is the style for a sync word.
func (t Theme) state(s api.SyncState) lipgloss.Style {
	switch s {
	case api.SyncSynced:
		return t.OK
	case api.SyncSyncing, api.SyncNoData:
		return t.Warn
	case api.SyncStopped:
		return t.Bad
	}
	return t.Dim
}

// Glyphs are the drawing characters. ASCII is the set for terminals whose
// fonts lack block elements (spec D8).
type Glyphs struct {
	Flag, Full, Empty, Tick, Ellipsis, Sel, Sep, Dash string
	Spark                                             []string
	ASCII                                             bool
}

var (
	unicodeGlyphs = Glyphs{Flag: "⚑", Full: "█", Empty: "░", Tick: "│", Ellipsis: "…", Sel: "▸", Sep: "·", Dash: "–", Spark: strings.Split("▁▂▃▄▅▆▇█", "")}
	asciiGlyphs   = Glyphs{Flag: "!", Full: "#", Empty: ".", Tick: "|", Ellipsis: "~", Sel: ">", Sep: "|", Dash: "-", Spark: []string{"_", ".", "-", "=", "#"}, ASCII: true}
)

// DetectGlyphs picks the glyph set: the preference when it names one, else
// ascii on a Windows console that is not Windows Terminal, on the Linux
// virtual console, or when JUMPGATE_ASCII=1.
func DetectGlyphs(pref, goos string, getenv func(string) string) Glyphs {
	switch pref {
	case "unicode":
		return unicodeGlyphs
	case "ascii":
		return asciiGlyphs
	}
	if getenv("JUMPGATE_ASCII") == "1" || getenv("TERM") == "linux" || (goos == "windows" && getenv("WT_SESSION") == "") {
		return asciiGlyphs
	}
	return unicodeGlyphs
}

// asciiKeys spells out the arrows that key labels use, for the ascii set.
var asciiKeys = strings.NewReplacer("↑", "up", "↓", "down", "←", "left", "→", "right")

// keyLabel is a binding's key label drawable with g.
func keyLabel(s string, g Glyphs) string {
	if g.ASCII {
		return asciiKeys.Replace(s)
	}
	return s
}
