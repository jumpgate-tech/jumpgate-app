package tui

import (
	"strings"
	"testing"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/tui/tuitest"
)

// The Vault screen offers what the vault's state allows, the next step first;
// the rest is under x.
func TestVaultOffersOnlyWhatFitsItsState(t *testing.T) {
	for _, c := range []struct {
		state       string
		want, never []string
	}{
		{"none", []string{"No vault yet.  [I] create the vault", "[ctrl+r] restore a backup"}, []string{"[ctrl+k]", "[u] unlock", "[p] put", "[ctrl+w] back up", "[m] "}},
		{"locked", []string{"The vault is locked.  [u] unlock"}, []string{"[ctrl+k]", "[I]", "[p] put"}},
		{"unusable", []string{"The vault file cannot be used.  [ctrl+r] restore a backup"}, []string{"[ctrl+k]", "[u]", "[I]"}},
		{"unlocked", []string{"[ctrl+k] lock", "[p] put a secret", "[ctrl+w] back up", "[ctrl+n] recovery code", "[x] more: "}, []string{"[u] unlock", "[I]", "The vault is locked", "KDF"}},
	} {
		f := tuitest.NewFake()
		f.VaultV = api.VaultView{State: c.state, Slots: []string{"passphrase"}}
		a := dashApp(t, f, 100, 24, "ascii", "auto")
		a.o.GOOS = "linux"
		m := press(t, a, "6")
		fr := tuitest.Frame(m)
		for _, w := range c.want {
			if !strings.Contains(fr, w) {
				t.Errorf("%s: no %q:\n%s", c.state, w, fr)
			}
		}
		for _, w := range c.never {
			if strings.Contains(fr, w) {
				t.Errorf("%s: shows %q:\n%s", c.state, w, fr)
			}
		}
		// The menu lists the same, and only that.
		menu := press(t, m, "x").(*App).modal.(*menuModal)
		var keys []string
		for _, it := range menu.all {
			keys = append(keys, it.key)
		}
		got := strings.Join(keys, " ")
		switch c.state {
		case "none":
			if got != "I ctrl+r r" {
				t.Errorf("none: menu %q", got)
			}
		case "locked":
			if got != "u t ctrl+r r" {
				t.Errorf("locked: menu %q", got)
			}
		case "unusable":
			if got != "ctrl+r r" {
				t.Errorf("unusable: menu %q", got)
			}
		case "unlocked":
			// The two rare actions are in the menu with no key.
			if !strings.Contains(strings.Join(menuLabels(menu), "|"), "harder to crack") || !strings.Contains(strings.Join(menuLabels(menu), "|"), "host key") {
				t.Errorf("unlocked: the menu lacks the key-less actions: %v", menuLabels(menu))
			}
			if !strings.Contains(got, "ctrl+k") || !strings.Contains(got, "ctrl+s") || strings.Contains(got, "u ") {
				t.Errorf("unlocked: menu %q", got)
			}
		}
	}
}

// The Server screen offers stop only while a server answers, and the install
// only until the service is the one running.
func TestServerOffersOnlyWhatFitsItsState(t *testing.T) {
	f := tuitest.NewFake()
	a := serverApp(t, f, "linux")
	menu := press(t, a, "x").(*App).modal.(*menuModal)
	var keys string
	for _, it := range menu.all {
		keys += it.key
	}
	if !strings.Contains(keys, "ctrl+t") || !strings.Contains(keys, "I") || strings.Contains(keys, "ctrl+s") {
		t.Errorf("a live own-server: menu %q", keys)
	}
	a.o.Server = func() ServerInfo { return ServerInfo{PID: 1, System: true, HTTPAddr: "127.0.0.1:7788"} }
	m := press(t, a, "esc", "1", "7")
	menu = press(t, m, "x").(*App).modal.(*menuModal)
	keys = ""
	for _, it := range menu.all {
		keys += it.key
	}
	if strings.Contains(keys, "I") {
		t.Errorf("the install is offered to the running service: %q", keys)
	}
}

func menuLabels(m *menuModal) []string {
	var out []string
	for _, it := range m.all {
		out = append(out, it.label)
	}
	return out
}
