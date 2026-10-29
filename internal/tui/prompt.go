package tui

import (
	"crypto/subtle"
	"encoding/base64"
	"strings"
	"unicode/utf8"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

// askModal asks for one line of text (a file path, a name, a number). submit
// answers a miss to keep the field open, or a command once it is done.
type askModal struct {
	title, detail string
	in            textinput.Model
	miss          string
	submit        func(a *App, v string) (miss string, cmd tea.Cmd)
}

func newAsk(title, detail, value string, submit func(a *App, v string) (string, tea.Cmd)) *askModal {
	in := textinput.New()
	in.Prompt = "> "
	in.CharLimit = 512
	in.KeyMap.Paste.SetEnabled(false) // ctrl+v would run pbpaste; a bracketed paste fills the field
	in.SetValue(value)
	in.Focus()
	return &askModal{title: sanitizeLine(title), detail: sanitize(detail), in: in, submit: submit}
}

func (m *askModal) key(a *App, k tea.KeyPressMsg) (bool, tea.Cmd) {
	switch k.String() {
	case "esc":
		a.flash = "cancelled; nothing was changed"
		return true, nil
	case "enter":
		before := a.modal
		miss, cmd := m.submit(a, strings.TrimSpace(m.in.Value()))
		if miss != "" {
			m.miss = miss
			return false, cmd
		}
		// submit may have opened the next modal (a confirmation, a masked
		// field): that one stays.
		return a.modal == before, cmd
	}
	var cmd tea.Cmd
	m.in, cmd = m.in.Update(k)
	return false, cmd
}

func (m *askModal) paste(_ *App, p tea.PasteMsg) tea.Cmd {
	var cmd tea.Cmd
	m.in, cmd = m.in.Update(p)
	return cmd
}

func (m *askModal) view(a *App, _, _ int) string {
	s := a.th.Title.Render(" "+m.title) + "\n\n"
	if m.detail != "" {
		s += " " + m.detail + "\n\n"
	}
	s += " " + m.in.View()
	if m.miss != "" {
		s += "\n\n " + a.th.Warn.Render(m.miss)
	}
	return s + "\n\n " + a.th.Dim.Render("enter to go on, esc to cancel")
}

// secretModal is a masked field for a secret: a passphrase or a token. As in
// unlockModal the bytes live in buffers allocated once and cleared after
// every send and on close; the field shows one mask glyph per character.
// With twice, the secret is typed a second time and must match. submit gets
// its own copy, which the command it returns must clear.
type secretModal struct {
	title, detail, label string
	twice                bool
	buf, first           []byte
	second               bool // typing the repeat
	lines                bool // a paste keeps its line breaks (a WireGuard config)
	miss                 string
	submit               func(a *App, secret []byte) tea.Cmd
}

func newSecret(title, detail, label string, twice bool, submit func(a *App, secret []byte) tea.Cmd) *secretModal {
	return &secretModal{title: sanitizeLine(title), detail: sanitize(detail), label: label, twice: twice,
		buf: make([]byte, 0, maxUnlockInput), first: make([]byte, 0, maxUnlockInput), submit: submit}
}

// multiline makes m take a pasted file: line breaks are kept and it may be
// up to 16 KiB.
func (m *secretModal) multiline() *secretModal {
	m.wipe()
	m.lines = true
	m.buf, m.first = make([]byte, 0, 16<<10), make([]byte, 0, 16<<10)
	return m
}

func (m *secretModal) wipe() {
	clear(m.buf[:cap(m.buf)])
	clear(m.first[:cap(m.first)])
	m.buf, m.first, m.second = m.buf[:0], m.first[:0], false
}

func (m *secretModal) add(s string) {
	if len(m.buf)+len(s) > cap(m.buf) {
		m.miss = "too long"
		return
	}
	m.buf = append(m.buf, s...)
}

func (m *secretModal) key(a *App, k tea.KeyPressMsg) (bool, tea.Cmd) {
	switch k.String() {
	case "esc":
		m.wipe()
		a.flash = "cancelled; nothing was changed"
		return true, nil
	case "enter":
		if len(m.buf) == 0 {
			m.miss = "type it, or esc to cancel"
			return false, nil
		}
		if m.twice && !m.second {
			m.first = append(m.first, m.buf...)
			clear(m.buf)
			m.buf, m.second, m.miss = m.buf[:0], true, ""
			return false, nil
		}
		if m.twice && subtle.ConstantTimeCompare(m.first, m.buf) != 1 {
			m.wipe()
			m.miss = "the two did not match; type it again"
			return false, nil
		}
		v := make([]byte, len(m.buf))
		copy(v, m.buf)
		m.wipe()
		return true, m.submit(a, v)
	case "backspace":
		if len(m.buf) > 0 {
			_, n := utf8.DecodeLastRune(m.buf)
			clear(m.buf[len(m.buf)-n:])
			m.buf = m.buf[:len(m.buf)-n]
		}
		return false, nil
	case "ctrl+u":
		clear(m.buf)
		m.buf = m.buf[:0]
		return false, nil
	}
	if k.Text != "" && k.Mod&^tea.ModShift == 0 {
		m.add(k.Text)
	}
	return false, nil
}

func (m *secretModal) paste(_ *App, p tea.PasteMsg) tea.Cmd {
	if m.lines {
		m.add(strings.ReplaceAll(p.Content, "\r", ""))
		return nil
	}
	m.add(strings.NewReplacer("\r", "", "\n", "").Replace(p.Content))
	return nil
}

func (m *secretModal) view(a *App, _, _ int) string {
	s := a.th.Title.Render(" "+m.title) + "\n\n"
	if m.detail != "" {
		s += " " + m.detail + "\n\n"
	}
	label := m.label
	if m.second {
		label = "again"
	}
	s += " " + label + ": " + strings.Repeat(a.gl.Mask, utf8.RuneCount(m.buf))
	if m.miss != "" {
		s += "\n\n " + a.th.Warn.Render(m.miss)
	}
	return s + "\n\n " + a.th.Dim.Render("enter to send, esc to cancel")
}

// secretBody is {<pre>"field":"<base64 of v>"} in one buffer the caller
// clears; pre is plain JSON members ending in a comma, or "".
func secretBody(pre, field string, v []byte) []byte {
	head := `{` + pre + `"` + field + `":"`
	const tail = `"}`
	b := make([]byte, len(head)+base64.StdEncoding.EncodedLen(len(v))+len(tail))
	n := copy(b, head)
	base64.StdEncoding.Encode(b[n:], v)
	copy(b[len(b)-len(tail):], tail)
	return b
}

// checkPath is why p cannot be passed to the jumpgate CLI as a file, or "".
// It becomes one argv entry: never a flag, never what the screen cannot show.
func checkPath(p string) string {
	switch {
	case p == "":
		return "type a file path, or esc to cancel"
	case strings.HasPrefix(p, "-"):
		return "a path cannot start with -; write ./" + p
	case sanitizeLine(p) != p:
		return "the path has characters the screen cannot show"
	}
	return ""
}

// pauseScript runs "$0" "$@" and waits for enter, so what the command
// printed (a recovery code, an error) stays until the person has read it.
// The command's exit status is kept.
const pauseScript = `"$0" "$@"; s=$?; printf '\npress enter to return to the dashboard '; read _; exit $s`

// runSelfPause is runSelf that waits for enter before the dashboard comes
// back. The argv is still fixed: the script is a constant and the command
// and its args are its positional parameters, never parsed as shell.
func (a *App) runSelfPause(args ...string) tea.Cmd {
	if a.o.Self == "" || a.o.GOOS == "windows" {
		return a.runSelf(args...)
	}
	argv := append([]string{"-c", pauseScript, a.o.Self}, args...)
	return tea.ExecProcess(a.o.Command("/bin/sh", argv...), func(err error) tea.Msg { return selfDoneMsg{args: args, err: err} })
}
