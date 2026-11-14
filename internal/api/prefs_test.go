// internal/api/prefs_test.go
package api

import (
	"encoding/json"
	"strings"
	"testing"
)

// An older stored block has no drawer: it reads as auto, and auto is valid.
func TestPrefsDrawerDefault(t *testing.T) {
	if got := (UIPrefs{}).WithDefaults().Drawer; got != "auto" {
		t.Fatalf("drawer default %q, want auto", got)
	}
	if got := DefaultPrefs().Drawer; got != "auto" {
		t.Fatalf("DefaultPrefs drawer %q, want auto", got)
	}
	if err := (UIPrefs{}).WithDefaults().Validate(); err != nil {
		t.Fatalf("an older block with defaults filled: %v", err)
	}
}

func TestPrefsDrawerValidate(t *testing.T) {
	for _, d := range []string{"auto", "open", "min"} {
		p := DefaultPrefs()
		p.Drawer = d
		if err := p.Validate(); err != nil {
			t.Errorf("drawer %q: %v", d, err)
		}
	}
	p := DefaultPrefs()
	p.Drawer = "x"
	if err := p.Validate(); err == nil || !strings.Contains(err.Error(), `drawer "x"`) {
		t.Fatalf("drawer x: %v", err)
	}
}

// The default fleet columns fit an 80-column terminal: a one-cell selection
// mark, then the columns with one space between them (spec D10).
func TestDefaultColumnsFit80(t *testing.T) {
	width, n := 0, 0
	for _, id := range DefaultPrefs().FleetColumns {
		c, ok := Column(id)
		if !ok {
			t.Fatalf("default column %q is not registered", id)
		}
		width += c.Width
		n++
	}
	if 1+width+n-1 > 80 {
		t.Fatalf("default columns need %d columns", 1+width+n-1)
	}
}

func TestPrefsValidate(t *testing.T) {
	if err := DefaultPrefs().Validate(); err != nil {
		t.Fatalf("defaults invalid: %v", err)
	}
	for name, mutate := range map[string]func(*UIPrefs){
		"unknown column":   func(p *UIPrefs) { p.FleetColumns = append(p.FleetColumns, "nope") },
		"duplicate column": func(p *UIPrefs) { p.FleetColumns = append(p.FleetColumns, "disk") },
		"no host column":   func(p *UIPrefs) { p.FleetColumns = []string{"disk"} },
		"unknown tab":      func(p *UIPrefs) { p.HostTabs = []string{"overview", "nope"} },
		"no tabs":          func(p *UIPrefs) { p.HostTabs = []string{} },
		"odd refresh":      func(p *UIPrefs) { p.RefreshSeconds = 7 },
		"units":            func(p *UIPrefs) { p.Units = "TB" },
		"glyphs":           func(p *UIPrefs) { p.Glyphs = "emoji" },
		"chain":            func(p *UIPrefs) { p.Chains = []int{-1} },
		"duplicate chain":  func(p *UIPrefs) { p.Chains = []int{369, 1, 369} },
	} {
		p := DefaultPrefs()
		mutate(&p)
		if p.Validate() == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

func TestWithDefaultsFillsZeroValues(t *testing.T) {
	p := UIPrefs{}.WithMouse(true).WithDefaults()
	if !p.MouseOn() || len(p.FleetColumns) == 0 || p.RefreshSeconds != 15 || p.Units != "GB" || p.Glyphs != "auto" || len(p.HostTabs) != len(HostTabs) {
		t.Fatalf("%+v", p)
	}
}

// The mouse is on unless it was turned off: a block with no mouse field (a
// new install, an old block) is on, and a stored false stays off.
func TestMouseDefaultsOnButStoredFalseStaysOff(t *testing.T) {
	if !DefaultPrefs().MouseOn() {
		t.Fatal("default prefs: mouse is off")
	}
	for body, want := range map[string]bool{`{}`: true, `{"units":"GiB"}`: true, `{"mouse":true}`: true, `{"mouse":false}`: false} {
		var p UIPrefs
		if err := json.Unmarshal([]byte(body), &p); err != nil {
			t.Fatal(err)
		}
		if got := p.WithDefaults().MouseOn(); got != want {
			t.Errorf("%s: mouse on = %v, want %v", body, got, want)
		}
	}
	// A false survives a round trip through the server's save.
	raw, _ := json.Marshal(DefaultPrefs().WithMouse(false))
	var back UIPrefs
	if err := json.Unmarshal(raw, &back); err != nil || back.WithDefaults().MouseOn() {
		t.Fatalf("false did not survive: %s %v", raw, err)
	}
}
