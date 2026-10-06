package api

import (
	"fmt"
	"slices"
)

// Bounds on the free-form parts of UIPrefs.
const (
	maxPrefChains = 64
	maxChainID    = 1<<31 - 1
)

// ColumnSpec is one fleet column a front end may show.
type ColumnSpec struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Width   int    `json:"width"`
	Default bool   `json:"default"`
}

// FleetColumns is every fleet column, in display order. The defaults fit 80
// columns; "host" is always shown.
var FleetColumns = []ColumnSpec{
	{"host", "HOST", 12, true},
	{"link", "LINK", 10, true},
	{"network", "NET", 9, true},
	{"sync", "SYNC", 9, true},
	{"lag", "LAG", 6, true},
	{"peers", "PEERS", 7, true},
	{"disk", "DISK", 15, true},
	{"firewall", "FW", 4, true},
	{"el", "EL", 9, false},
	{"cl", "CL", 9, false},
	{"jobs", "JOBS", 5, false},
	{"age", "AGE", 5, false},
	{"agent", "AGENT", 12, false},
}

// HostTabs is every host-detail tab, in display order.
var HostTabs = []string{"overview", "storage", "endpoints", "security", "logs", "services"}

// RefreshChoices are the fleet refresh intervals a person may pick, seconds.
var RefreshChoices = []int{5, 15, 30, 60}

// Column finds a column by id.
func Column(id string) (ColumnSpec, bool) {
	for _, c := range FleetColumns {
		if c.ID == id {
			return c, true
		}
	}
	return ColumnSpec{}, false
}

// UIPrefs are the display choices the server stores for every front end
// (spec D20). Chains empty means every chain.
type UIPrefs struct {
	FleetColumns   []string `json:"fleetColumns"`
	HostTabs       []string `json:"hostTabs"`
	Chains         []int    `json:"chains"`
	RefreshSeconds int      `json:"refreshSeconds"`
	Units          string   `json:"units"` // "GB" | "GiB"
	Compact        bool     `json:"compact"`
	ShowEstimates  bool     `json:"showEstimates"`
	Bell           bool     `json:"bell"`
	Mouse          bool     `json:"mouse"`
	Glyphs         string   `json:"glyphs"` // "auto" | "unicode" | "ascii"
}

// DefaultPrefs is what a new controller shows.
func DefaultPrefs() UIPrefs {
	var cols []string
	for _, c := range FleetColumns {
		if c.Default {
			cols = append(cols, c.ID)
		}
	}
	return UIPrefs{
		FleetColumns: cols, HostTabs: slices.Clone(HostTabs), Chains: []int{},
		RefreshSeconds: 15, Units: "GB", ShowEstimates: true, Glyphs: "auto",
	}
}

// WithDefaults fills every zero-valued field from DefaultPrefs, so a stored
// block from an older version gains new fields without losing choices.
func (p UIPrefs) WithDefaults() UIPrefs {
	d := DefaultPrefs()
	if p.FleetColumns == nil {
		p.FleetColumns = d.FleetColumns
	}
	if p.HostTabs == nil {
		p.HostTabs = d.HostTabs
	}
	if p.Chains == nil {
		p.Chains = d.Chains
	}
	if p.RefreshSeconds == 0 {
		p.RefreshSeconds = d.RefreshSeconds
	}
	if p.Units == "" {
		p.Units = d.Units
	}
	if p.Glyphs == "" {
		p.Glyphs = d.Glyphs
	}
	return p
}

// Validate refuses a choice no front end can show.
func (p UIPrefs) Validate() error {
	seen := map[string]bool{}
	for _, id := range p.FleetColumns {
		if _, ok := Column(id); !ok {
			return fmt.Errorf("unknown fleet column %q", id)
		}
		if seen[id] {
			return fmt.Errorf("fleet column %q twice", id)
		}
		seen[id] = true
	}
	if !seen["host"] {
		return fmt.Errorf("the host column is always shown")
	}
	if len(p.HostTabs) == 0 {
		return fmt.Errorf("at least one host tab")
	}
	tabs := map[string]bool{}
	for _, tab := range p.HostTabs {
		if !slices.Contains(HostTabs, tab) {
			return fmt.Errorf("unknown host tab %q", tab)
		}
		if tabs[tab] {
			return fmt.Errorf("host tab %q twice", tab)
		}
		tabs[tab] = true
	}
	if len(p.Chains) > maxPrefChains {
		return fmt.Errorf("at most %d chains", maxPrefChains)
	}
	for _, c := range p.Chains {
		if c <= 0 || c > maxChainID {
			return fmt.Errorf("chain id %d", c)
		}
	}
	if !slices.Contains(RefreshChoices, p.RefreshSeconds) {
		return fmt.Errorf("refresh %ds is not one of %v", p.RefreshSeconds, RefreshChoices)
	}
	if p.Units != "GB" && p.Units != "GiB" {
		return fmt.Errorf("units %q (want GB or GiB)", p.Units)
	}
	if p.Glyphs != "auto" && p.Glyphs != "unicode" && p.Glyphs != "ascii" {
		return fmt.Errorf("glyphs %q (want auto, unicode or ascii)", p.Glyphs)
	}
	return nil
}
