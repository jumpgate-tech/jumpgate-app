package tui

import (
	"slices"

	"charm.land/bubbles/v2/key"
)

// KeyDoc is one key binding as documentation shows it.
type KeyDoc struct {
	Where string   // the screen or view it works on
	Label string   // the key as the ? list shows it, such as "↑/k"
	Help  string   // what it does
	Keys  []string // every key it matches, as Bubble Tea names them
}

// keymaps is every keymap struct and where it works; a new keymap belongs
// here too, or the guide's key check will not know its keys.
var keymaps = []struct {
	where string
	km    any
}{
	{"everywhere", globalKeys}, {"moving", navKeys}, {"Fleet", fleetKeys}, {"Hosts", hostsKeys},
	{"an open box", hostKeys}, {"a box's Services", svcKeys}, {"a box's Logs", logKeys}, {"a box's other tabs", boxKeys},
	{"a box's modules", modKeys}, {"Signers", signerKeys}, {"Jobs", jobKeys}, {"Gateways", gatewayKeys}, {"Gateways (modules)", struct{ Modules key.Binding }{gwModKey}},
	{"VPN", vpnKeys}, {"Vault", vaultKeys}, {"Server", serverKeys}, {"the guide", manualKeys},
}

// Keymap is every key binding the TUI has, from the same bindings the
// screens match on, for the guide's tests and for generating docs.
func Keymap() []KeyDoc {
	var out []KeyDoc
	add := func(where string, b key.Binding) {
		out = append(out, KeyDoc{Where: where, Label: b.Help().Key, Help: b.Help().Desc, Keys: slices.Clone(b.Keys())})
	}
	for _, m := range keymaps {
		for _, b := range keyList(m.km) {
			add(m.where, b)
		}
	}
	nav := map[key.Help]bool{}
	for _, b := range keyList(navKeys) {
		nav[b.Help()] = true
	}
	for _, b := range (&settingsScreen{}).keys() {
		if !nav[b.Help()] {
			add("Settings", b)
		}
	}
	return out
}

// Commands is every command the : palette takes, as its list shows them:
// NAME is a box, TOPIC a guide topic, a|b one of the words, [x] optional.
func Commands() []string { return slices.Clone(paletteCommands) }
