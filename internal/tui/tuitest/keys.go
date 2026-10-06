// Package tuitest drives internal/tui's models in tests: key messages by
// name, a synchronous command runner, golden frames, and a fake backend. It
// must not import internal/tui (tui's tests import it).
package tuitest

import (
	"time"

	tea "charm.land/bubbletea/v2"
)

// Key is the key press a person makes for a name: "enter", "esc", "tab",
// "shift+tab", "up", "down", "left", "right", "backspace", "space",
// "ctrl+c", "ctrl+v", or one printable character.
func Key(s string) tea.KeyPressMsg {
	switch s {
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEscape}
	case "tab":
		return tea.KeyPressMsg{Code: tea.KeyTab}
	case "shift+tab":
		return tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift}
	case "up":
		return tea.KeyPressMsg{Code: tea.KeyUp}
	case "down":
		return tea.KeyPressMsg{Code: tea.KeyDown}
	case "left":
		return tea.KeyPressMsg{Code: tea.KeyLeft}
	case "right":
		return tea.KeyPressMsg{Code: tea.KeyRight}
	case "backspace":
		return tea.KeyPressMsg{Code: tea.KeyBackspace}
	case "space":
		return tea.KeyPressMsg{Code: ' ', Text: " "}
	case "ctrl+c":
		return tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}
	case "ctrl+v":
		return tea.KeyPressMsg{Code: 'v', Mod: tea.ModCtrl}
	}
	r := []rune(s)
	if len(r) != 1 {
		panic("tuitest: unknown key " + s)
	}
	return tea.KeyPressMsg{Code: r[0], Text: s}
}

// Keys is Key for several names.
func Keys(names ...string) []tea.Msg {
	out := make([]tea.Msg, len(names))
	for i, n := range names {
		out[i] = Key(n)
	}
	return out
}

// Type is one key press per character of s.
func Type(s string) []tea.Msg {
	var out []tea.Msg
	for _, r := range s {
		out = append(out, Key(string(r)))
	}
	return out
}

// Send runs Update for each message in order and returns the commands.
func Send(m tea.Model, msgs ...tea.Msg) (tea.Model, []tea.Cmd) {
	var cmds []tea.Cmd
	for _, msg := range msgs {
		var cmd tea.Cmd
		m, cmd = m.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	return m, cmds
}

// cmdWait bounds one command. A command still blocked after it (a stream
// with nothing to say, a one-second tick) is dropped.
const cmdWait = 150 * time.Millisecond

// Run executes cmd, expanding batches, and returns the messages that arrived
// within cmdWait each.
func Run(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	ch := make(chan tea.Msg, 1)
	go func() { ch <- cmd() }()
	select {
	case msg := <-ch:
		if batch, ok := msg.(tea.BatchMsg); ok {
			var out []tea.Msg
			for _, c := range batch {
				out = append(out, Run(c)...)
			}
			return out
		}
		if msg == nil {
			return nil
		}
		return []tea.Msg{msg}
	case <-time.After(cmdWait):
		return nil
	}
}

// Settle runs cmds, feeds their messages back into m, and repeats with the
// commands that produces, up to eight rounds.
func Settle(m tea.Model, cmds ...tea.Cmd) tea.Model {
	for round := 0; round < 8 && len(cmds) > 0; round++ {
		var next []tea.Cmd
		for _, c := range cmds {
			var produced []tea.Cmd
			m, produced = Send(m, Run(c)...)
			next = append(next, produced...)
		}
		cmds = next
	}
	return m
}
