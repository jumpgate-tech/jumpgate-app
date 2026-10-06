// internal/api/prefs_test.go
package api

import "testing"

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
	} {
		p := DefaultPrefs()
		mutate(&p)
		if p.Validate() == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

func TestWithDefaultsFillsZeroValues(t *testing.T) {
	p := UIPrefs{Mouse: true}.WithDefaults()
	if !p.Mouse || len(p.FleetColumns) == 0 || p.RefreshSeconds != 15 || p.Units != "GB" || p.Glyphs != "auto" || len(p.HostTabs) != len(HostTabs) {
		t.Fatalf("%+v", p)
	}
}
