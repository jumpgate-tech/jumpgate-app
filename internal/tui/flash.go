package tui

import (
	"strings"
	"time"

	"github.com/charmbracelet/x/ansi"
)

// Feedback. A notice (a.flash) is one line that shares the bottom line with
// the hints and goes on the next key press. An error is different: it needs
// to be read, so it gets its own one or two lines above the bottom line and
// stays until esc, a click on it, or errorFor has passed. Both are written to
// the activity log, so one that was missed can be found there.

// errorFor is how long an error stays; infoFor, a notice nobody pressed a key
// after.
const (
	errorFor = 10 * time.Second
	infoFor  = 15 * time.Second
)

// setErr shows text as an error: the line starts "error:" and stays.
func (a *App) setErr(text string) {
	a.flash = sanitizeLine(text)
	a.errFlash = a.flash
	a.flashSeq++
}

// noteFlash runs after every update: a notice or error that was just set
// starts its clock and goes into the activity log.
func (a *App) noteFlash(prev string, prevSeq uint64) {
	if a.flash == "" {
		a.flashErr = false
		return
	}
	if a.flash == prev && a.flashSeq == prevSeq {
		return
	}
	a.flashErr = a.flash == a.errFlash
	a.flashAt = a.now()
	ev := logEvent{At: a.now(), Sev: logInfo, Text: sanitizeLine(a.flash)}
	if a.flashErr {
		ev.Sev, ev.Text = logWarn, "error: "+ev.Text
	}
	if h, ok := a.detail.(*hostScreen); ok {
		ev.Box = sanitizeLine(h.id)
	}
	a.log.add(ev)
}

// expireFlash drops a notice or an error that has been up long enough.
func (a *App) expireFlash() {
	if a.flash == "" {
		return
	}
	ttl := infoFor
	if a.flashErr {
		ttl = errorFor
	}
	if a.now().Sub(a.flashAt) > ttl {
		a.flash, a.flashErr = "", false
	}
}

// errorText is the error line, "error: " first, as drawn in w cells: at most
// two lines, the last cut with an ellipsis.
func (a *App) errorRows(w int) []string {
	if !a.flashErr || a.flash == "" || w < 20 {
		return nil
	}
	text := "error: " + a.flash
	lines := strings.Split(ansi.Wrap(text, w-2, ""), "\n")
	if len(lines) > 2 {
		lines = []string{lines[0], truncate(lines[1]+" "+strings.Join(lines[2:], " "), w-2, a.gl.Ellipsis)}
	}
	last := len(lines) - 1
	if tail := "  [esc] dismiss"; ansi.StringWidth(lines[last])+len(tail) <= w-2 {
		lines[last] += tail
	}
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = " " + a.th.Warn.Render(l)
	}
	return out
}
