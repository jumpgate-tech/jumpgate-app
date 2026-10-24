package tui

import (
	"strings"
	"testing"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/tui/tuitest"
)

// The health watch covers the whole box: turning it on sends no chain, and
// asks for its endpoints (name=url, comma separated).
func TestModulesModalTurnsOnWatch(t *testing.T) {
	f := modulesFake()
	for i := range f.CatalogV {
		if f.CatalogV[i].ID == "watch" {
			f.CatalogV[i].Applies = []api.ModuleApplies{{ChainID: 0, OK: true}}
		}
	}
	f.SetModuleV = api.SetModuleResult{Status: "started"}
	m := press(t, openTestHost(t, f, "unicode"), "m", "down", "down", "down", "enter")
	if fr := tuitest.Frame(m); !strings.Contains(fr, "endpoints") {
		t.Fatalf("no endpoints field:\n%s", fr)
	}
	m = typed(t, m, "gw=https://rpc.example, cl=beacon+http://10.0.0.2:5052")
	press(t, m, "enter")
	req := f.LastSetModule
	if !f.Called("SetModule box watch") || !req.Enabled || req.ChainID != 0 ||
		len(req.Upstreams) != 2 || req.Upstreams[0] != "gw=https://rpc.example" || req.Upstreams[1] != "cl=beacon+http://10.0.0.2:5052" {
		t.Fatalf("calls %v, req %+v", f.Calls, req)
	}
}
