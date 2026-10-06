package tui

import (
	"reflect"

	"charm.land/bubbles/v2/key"
)

// globalKeys work on every screen unless a text field has the keyboard. The
// help screen lists every field here (help.go), so a key added here is
// documented by adding it.
var globalKeys = struct {
	Screens, Help, Palette, Back, Quit, ForceQuit, Restart key.Binding
}{
	Screens:   key.NewBinding(key.WithKeys("1", "2", "3", "4", "5", "6"), key.WithHelp("1-6", "screens")),
	Help:      key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
	Palette:   key.NewBinding(key.WithKeys(":"), key.WithHelp(":", "command")),
	Back:      key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
	Quit:      key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
	ForceQuit: key.NewBinding(key.WithKeys("ctrl+c"), key.WithHelp("ctrl+c", "quit now")),
	Restart:   key.NewBinding(key.WithKeys("R"), key.WithHelp("R", "restart the server")),
}

// navKeys move a cursor or switch tabs.
var navKeys = struct {
	Up, Down, Left, Right, Enter, Filter key.Binding
}{
	Up:     key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
	Down:   key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
	Left:   key.NewBinding(key.WithKeys("left", "h", "shift+tab"), key.WithHelp("←/h", "previous tab")),
	Right:  key.NewBinding(key.WithKeys("right", "l", "tab"), key.WithHelp("→/l", "next tab")),
	Enter:  key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "open")),
	Filter: key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
}

// keyList is every key.Binding field of a keymap struct, in declaration
// order. Help is built from it rather than from a second list of the same
// keys, which would drift.
func keyList(keymap any) []key.Binding {
	v := reflect.ValueOf(keymap)
	var out []key.Binding
	for i := 0; i < v.NumField(); i++ {
		if b, ok := v.Field(i).Interface().(key.Binding); ok {
			out = append(out, b)
		}
	}
	return out
}
