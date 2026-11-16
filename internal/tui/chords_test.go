package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/valve-tech/jumpgate/internal/tui/tuitest"
)

// changing reports whether a backend call changes something on a box, the
// server or the vault.
func changing(call string) bool {
	switch {
	case strings.HasPrefix(call, "ServiceAction"), strings.HasPrefix(call, "Lock"), strings.HasPrefix(call, "MeasureDisk"),
		strings.HasPrefix(call, "RemoveTarget"), strings.HasPrefix(call, "SetModule"), strings.HasPrefix(call, "SaveSettings"),
		strings.HasPrefix(call, "Decide"), strings.HasPrefix(call, "Adopt"), strings.HasPrefix(call, "MigrateNames"):
		return true
	case strings.HasPrefix(call, "Do POST"), strings.HasPrefix(call, "Do PUT"), strings.HasPrefix(call, "Do DELETE"):
		return true
	}
	return false
}

// Plain keys only look and move: whichever one is pressed on whichever screen
// or box tab, nothing is changed. (A key may open a flow with its own steps,
// which changes nothing by itself.)
func TestPlainKeysNeverChangeAnything(t *testing.T) {
	plain := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ+-!"
	type setup struct {
		name string
		make func(f *tuitest.Fake) *App
	}
	var setups []setup
	for _, k := range []string{"1", "2", "3", "4", "5", "6", "7"} {
		k := k
		setups = append(setups, setup{"screen " + k, func(f *tuitest.Fake) *App {
			f.VaultV.State = "unlocked"
			return press(t, dashApp(t, f, 100, 30, "unicode", "auto"), k).(*App)
		}})
	}
	for ti := 0; ti < 8; ti++ {
		ti := ti
		setups = append(setups, setup{"box tab " + string(rune('0'+ti)), func(f *tuitest.Fake) *App {
			return tab(t, openTestHost(t, f, "unicode"), ti)
		}})
	}
	for _, s := range setups {
		f := boxFake()
		f.GatewaysV = nil
		a := s.make(f)
		f.Calls = nil
		var m tea.Model = a
		for _, r := range plain {
			func() {
				defer func() {
					if e := recover(); e != nil {
						t.Errorf("%s, key %q: %v", s.name, string(r), e)
					}
				}()
				m = press(t, m, string(r))
			}()
			for _, c := range f.Calls {
				if changing(c) {
					t.Errorf("%s: the plain key %q made the call %q", s.name, string(r), c)
				}
			}
			f.Calls = nil
			if app := m.(*App); app.modal != nil || app.current().capturing() {
				m = press(t, m, "esc") // out of any flow or menu the key opened
			}
		}
	}
}

// A chord changes something, and the footer shows it as a button.
func TestChordsAreInTheFooterAndAreASCIISafe(t *testing.T) {
	for _, glyphs := range []string{"unicode", "ascii"} {
		h := openTestHost(t, tuitest.NewFake(), glyphs)
		h = tab(t, h, 5)
		fr := tuitest.Frame(h)
		last := lastLine(fr)
		for _, want := range []string{"[ctrl+s] start", "[ctrl+t] stop", "[ctrl+r] restart", "[?] more"} {
			if !strings.Contains(last, want) {
				t.Errorf("%s: footer %q lacks %q", glyphs, last, want)
			}
		}
		if glyphs == "ascii" {
			requireASCII(t, fr)
		}
	}
}

// r reloads on every screen and tab, as a browser's reload does.
func TestReloadEverywhere(t *testing.T) {
	for _, c := range []struct{ screen, want string }{
		{"1", "Fleet"}, {"2", "WatchJobs"}, {"3", "Controller"}, {"4", "Gateways"}, {"5", "Do GET /api/vpns"},
		{"6", "Vault"}, {"7", "Do GET /api/update"}, {"8", "Settings"},
	} {
		f := vpnFake()
		a := newTestApp(t, f, 100, 30, "unicode")
		m := press(t, a, c.screen)
		f.Calls = nil
		press(t, m, "r")
		if !f.Called(c.want) {
			t.Errorf("screen %s: r made %v, want %s", c.screen, f.Calls, c.want)
		}
	}
	// On an open box: the tab's data is read again; the live tabs say so.
	f := boxFake()
	h := tab(t, openTestHost(t, f, "unicode"), 1) // Storage
	f.Calls = nil
	press(t, h, "r")
	if !f.Called("Disk box") {
		t.Errorf("Storage: r made %v", f.Calls)
	}
	if f.Called("MeasureDisk box") {
		t.Error("Storage: r measured; that is ctrl+p")
	}
	g := press(t, tab(t, openTestHost(t, boxFake(), "unicode"), 5), "r")
	if !strings.Contains(g.(*App).flash, "live") {
		t.Errorf("Services: r flashed %q", g.(*App).flash)
	}
}

// The help list has two groups, and the change group is the Ctrl chords.
func TestHelpGroupsLookAndChange(t *testing.T) {
	m := press(t, servicesHost(t, tuitest.NewFake()), "?")
	fr := tuitest.Frame(m)
	i, j := strings.Index(fr, "look (plain keys)"), strings.Index(fr, "change (Ctrl chords)")
	if i < 0 || j < 0 {
		t.Fatalf("help lacks its groups:\n%s", fr)
	}
	for _, want := range []string{"ctrl+s", "ctrl+t", "ctrl+r"} {
		if k := strings.Index(fr, want); k < j {
			t.Errorf("%s is not in the change group:\n%s", want, fr)
		}
	}
	for _, plainKey := range []string{"  r            reload", "  S            SSH shell"} {
		if k := strings.Index(fr, plainKey); k < 0 || k > j {
			t.Errorf("%q is not in the look group:\n%s", plainKey, fr)
		}
	}
}
