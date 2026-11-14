package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/apiclient"
	"github.com/valve-tech/jumpgate/internal/tui/tuitest"
)

// Before any reading the strip is one phrase, not a tile of n/a for each.
func TestStripBeforeAnyReading(t *testing.T) {
	for _, c := range []struct {
		w, h   int
		glyphs string
		want   string
	}{
		{80, 24, "unicode", "connecting to the server… no readings yet"},
		{80, 24, "ascii", "connecting to the server... no readings yet"},
		{120, 40, "unicode", "connecting to the server… no readings yet"},
	} {
		fr := tuitest.Frame(newTestApp(t, tuitest.NewFake(), c.w, c.h, c.glyphs))
		if !strings.Contains(fr, c.want) || strings.Contains(strings.Split(fr, "\n")[1], "n/a") {
			t.Errorf("%dx%d %s:\n%s", c.w, c.h, c.glyphs, fr)
		}
		if c.glyphs == "ascii" {
			requireASCII(t, fr)
		}
	}
}

// Once live, the strip names each reading and says used, never free.
func TestStripWhenLive(t *testing.T) {
	a := dashApp(t, tuitest.NewFake(), 80, 24, "unicode", "auto")
	line := ansi.Strip(strings.Split(tuitest.Frame(a), "\n")[1])
	for _, want := range []string{"1/3 up", "synced", "lag 2", "peers", "disk 61% used", "! 1 fail 2 warn", " · "} {
		if !strings.Contains(line, want) {
			t.Errorf("80 cols: strip lacks %q: %q", want, line)
		}
	}
	if strings.Contains(line, "free") || strings.Contains(line, "n/a n/a") || strings.Contains(line, "gw") {
		t.Errorf("80 cols: %q", line)
	}
	// Wider, the readings are spelled out.
	wide := dashApp(t, tuitest.NewFake(), 100, 24, "unicode", "auto")
	line = ansi.Strip(strings.Split(tuitest.Frame(wide), "\n")[1])
	for _, want := range []string{"lag 2 blk", " el / ", " cl", "disk max 61% used"} {
		if !strings.Contains(line, want) {
			t.Errorf("100 cols: strip lacks %q: %q", want, line)
		}
	}
	if ansi.StringWidth(line) > 100 {
		t.Errorf("strip is %d wide", ansi.StringWidth(line))
	}
}

// The tab bar has three tiers: names, short names, numbers.
func TestTabTiers(t *testing.T) {
	a := dashApp(t, tuitest.NewFake(), 80, 24, "ascii", "auto")
	for _, c := range []struct {
		room int
		tier int
		want string
	}{
		{60, tabsNames, "[Fleet] Jobs Signers Gateways VPN Vault Server Settings"},
		{45, tabsShort, "[Fleet] Jobs Sign GW VPN Vault Srv Set"},
		{20, tabsNumbers, "[Fleet] 2 3 4 5 6 7 8"},
	} {
		got, tier := a.tabs(c.room)
		if got = ansi.Strip(got); got != c.want || tier != c.tier {
			t.Errorf("room %d: %q tier %d, want %q tier %d", c.room, got, tier, c.want, c.tier)
		}
	}
	// A locked vault and an unfinished setup crowd the names out of 80
	// columns: the short names show instead of numbers, with the age dropped
	// only if it must.
	m, _ := tuitest.Send(a, fleetMsg{u: apiclient.Update[api.Fleet]{Value: lockedFleet(), Has: true, State: apiclient.Live}})
	m, _ = tuitest.Send(m, guideMsg{v: api.VaultView{State: "locked"}, ctl: api.ControllerView{State: "ok"}})
	h := ansi.Strip(m.(*App).header(80))
	if ansi.StringWidth(h) > 80 || !strings.Contains(h, "[Fleet] Jobs Sign GW VPN Vault Srv Set") || !strings.Contains(h, "LOCKED") {
		t.Fatalf("header %q", h)
	}
	// A click on a short name opens its view.
	app := m.(*App)
	app.prefs = app.prefs.WithMouse(true)
	x, y := at(t, m, "Srv")
	if got := click(t, m, x, y); got.(*App).active != scrServer {
		t.Fatalf("clicking Srv: active %v", got.(*App).active)
	}
}
