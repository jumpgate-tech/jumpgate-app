package tui

import (
	"errors"
	"strings"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"

	"github.com/valve-tech/jumpgate/internal/api"
)

// maxUnlockInput bounds the passphrase field, as the CLI bounds a line.
const maxUnlockInput = 1024

// unlockModal asks for the vault passphrase and sends it to the server. The
// typed bytes live in buf, which is allocated once and never grown, so no
// uncleared copy is left behind; it is cleared after every send and when the
// modal closes. The field shows one mask glyph per character, never the text.
type unlockModal struct {
	buf     []byte
	sending bool
	miss    string
	// custodyLine is the once-per-unlock note on who holds the key; the
	// server-install plan fills it. Empty here.
	custodyLine string
}

// unlockDoneMsg is the server's answer to an unlock.
type unlockDoneMsg struct {
	v   api.VaultView
	err error
}

// lockDoneMsg is the server's answer to a lock.
type lockDoneMsg struct {
	v   api.VaultView
	err error
}

func newUnlock() *unlockModal { return &unlockModal{buf: make([]byte, 0, maxUnlockInput)} }

// wipe clears the field.
func (m *unlockModal) wipe() {
	clear(m.buf[:cap(m.buf)])
	m.buf = m.buf[:0]
}

// add appends typed or pasted text if it fits, without reallocating.
func (m *unlockModal) add(s string) {
	if len(m.buf)+len(s) > cap(m.buf) {
		m.miss = "passphrase too long"
		return
	}
	m.buf = append(m.buf, s...)
}

func (m *unlockModal) key(a *App, k tea.KeyPressMsg) (bool, tea.Cmd) {
	if m.sending {
		if k.String() == "esc" {
			m.wipe()
			return true, nil
		}
		return false, nil
	}
	switch k.String() {
	case "esc":
		m.wipe()
		a.flash = "the vault stays locked"
		return true, nil
	case "enter":
		if len(m.buf) == 0 {
			m.miss = "type the vault passphrase, or esc to cancel"
			return false, nil
		}
		pass := make([]byte, len(m.buf))
		copy(pass, m.buf)
		m.wipe()
		m.sending, m.miss = true, ""
		be, ctx := a.be, a.ctx
		return false, func() tea.Msg {
			v, err := be.Unlock(ctx, pass) // clears pass
			clear(pass)
			return unlockDoneMsg{v: v, err: err}
		}
	case "backspace":
		if len(m.buf) > 0 {
			_, n := utf8.DecodeLastRune(m.buf)
			clear(m.buf[len(m.buf)-n:])
			m.buf = m.buf[:len(m.buf)-n]
		}
		return false, nil
	case "ctrl+u":
		m.wipe()
		return false, nil
	}
	if k.Text != "" && k.Mod&^tea.ModShift == 0 {
		m.add(k.Text)
	}
	return false, nil
}

// paste adds a bracketed paste to the field; a trailing newline does not
// submit, and line breaks are dropped.
func (m *unlockModal) paste(_ *App, p tea.PasteMsg) tea.Cmd {
	if !m.sending {
		m.add(strings.NewReplacer("\r", "", "\n", "").Replace(p.Content))
	}
	return nil
}

func (m *unlockModal) view(a *App, _, _ int) string {
	s := a.th.Title.Render(" Unlock the vault") + "\n\n" +
		" The server holds no key while the vault is locked; jobs wait for unlock.\n\n" +
		" passphrase: " + strings.Repeat(a.gl.Mask, utf8.RuneCount(m.buf))
	if m.sending {
		s += "\n\n " + a.th.Dim.Render("unlocking"+a.gl.Ellipsis)
	}
	if m.miss != "" {
		s += "\n\n " + a.th.Warn.Render(m.miss)
	}
	if m.custodyLine != "" {
		s += "\n\n " + sanitizeLine(m.custodyLine)
	}
	return s + "\n\n " + a.th.Dim.Render("enter to unlock, esc to cancel")
}

// unlockDone handles the server's answer: a success closes the modal, a
// refusal keeps it open with the field empty.
func (a *App) unlockDone(msg unlockDoneMsg) {
	if msg.err == nil {
		a.vaultOpen = true
	}
	m, ok := a.modal.(*unlockModal)
	if !ok {
		if msg.err == nil {
			a.flash = "the vault is unlocked"
		}
		return
	}
	m.sending = false
	m.wipe()
	if msg.err == nil {
		a.modal = nil
		a.flash = "the vault is unlocked"
		return
	}
	var e *api.Error
	if errors.As(msg.err, &e) && e.Code == api.CodeWrongPassphrase {
		m.miss = "wrong passphrase; try again"
		return
	}
	m.miss = a.errText(msg.err)
}

// lockVault asks the server to drop the vault key.
func (a *App) lockVault() tea.Cmd {
	be, ctx := a.be, a.ctx
	return func() tea.Msg {
		v, err := be.Lock(ctx)
		return lockDoneMsg{v: v, err: err}
	}
}
