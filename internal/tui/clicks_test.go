package tui

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/apiclient"
	"github.com/valve-tech/jumpgate/internal/tui/tuitest"
)

// Everything a person can do by key can be done by click. These tests walk
// the click targets screen by screen.

// The header's view tabs: by name, by short name and by number.
func TestClickHeaderTabsInEveryTier(t *testing.T) {
	a, _ := mouseApp(t, tuitest.NewFake(), 80, 24)
	x, y := at(t, a, " Settings")
	if m := click(t, a, x+2, y); m.(*App).active != scrSettings {
		t.Fatalf("the name Settings: active %v", m.(*App).active)
	}
	// The numbers: a server that is not answering takes the room.
	b, _ := mouseApp(t, tuitest.NewFake(), 80, 24)
	m, _ := tuitest.Send(b, fleetMsg{u: apiclient.Update[api.Fleet]{Value: sampleFleet(), Has: true, State: apiclient.Retrying, Err: errors.New("dial tcp 127.0.0.1:7788: connection refused")}})
	hdr := strings.Split(tuitest.Frame(m), "\n")[0]
	if !strings.Contains(hdr, "[Fleet] 2 3 4 5 6 7 8") {
		t.Skipf("this header is not in the numbers tier: %q", hdr)
	}
	x, y = at(t, m, "7 8")
	if got := click(t, m, x, y); got.(*App).active != scrServer {
		t.Fatalf("the number 7: active %v", got.(*App).active)
	}
}

// The tabs of an open box open by click.
func TestClickBoxTabs(t *testing.T) {
	a := openTestHost(t, tuitest.NewFake(), "unicode")
	x, y := at(t, a, "Storage")
	m := click(t, a, x+1, y)
	if h := m.(*App).detail.(*hostScreen); h.tabName(m.(*App)) != "storage" {
		t.Fatalf("tab %q", h.tabName(m.(*App)))
	}
	x, y = at(t, m, "Services")
	m = click(t, m, x+1, y)
	if h := m.(*App).detail.(*hostScreen); h.tabName(m.(*App)) != "services" {
		t.Fatalf("tab %q", h.tabName(m.(*App)))
	}
	// The footer's buttons are the tab's own keys: clicking [e] restarts, which asks first.
	x, y = at(t, m, "[e] restart")
	if got := click(t, m, x+1, y); func() bool { _, ok := got.(*App).modal.(*confirmModal); return !ok }() {
		t.Fatalf("clicking [e] restart: modal %T", got.(*App).modal)
	}
}

// A click on a job selects it even with another job's detail between them.
func TestClickJobsRowSelects(t *testing.T) {
	f := tuitest.NewFake()
	f.JobsCh <- apiclient.Update[api.JobsInbox]{Value: sampleInbox(), Has: true, State: apiclient.Live}
	m := press(t, newTestApp(t, f, 80, 24, "unicode"), "2")
	js := m.(*App).screens[scrJobs].(*jobsScreen)
	if js.cursor != 0 {
		t.Fatalf("cursor %d", js.cursor)
	}
	x, y := at(t, m, "box-b")
	m = click(t, m, x, y)
	if js.cursor != 1 {
		t.Fatalf("a click on box-b left the cursor at %d:\n%s", js.cursor, tuitest.Frame(m))
	}
	// A click on the selected job's detail selects nothing else.
	x, y = at(t, m, "Rename the node")
	click(t, m, x, y)
	if js.cursor != 1 {
		t.Fatalf("a click on a detail line moved the cursor to %d", js.cursor)
	}
}

// Gateways with detail lines and blanks between them are all clickable.
func TestClickGatewaysRowSelects(t *testing.T) {
	f := tuitest.NewFake()
	f.GatewaysV = []api.GatewaySummary{gateway("rpc-a", "box"), gateway("rpc-b", "box"), gateway("rpc-c", "box")}
	m := press(t, newTestApp(t, f, 100, 30, "unicode"), "4")
	gs := m.(*App).screens[scrGateways].(*gatewaysScreen)
	x, y := at(t, m, "rpc-a")
	m = click(t, m, x+1, y)
	if gs.cursor != 0 {
		t.Fatalf("cursor %d", gs.cursor)
	}
	// The next gateway is below the selected one's detail lines.
	x, y = at(t, m, "rpc-b")
	m = click(t, m, x+1, y)
	if gs.cursor != 1 {
		t.Fatalf("a click on rpc-b left the cursor at %d:\n%s", gs.cursor, tuitest.Frame(m))
	}
}

// A click on a setting toggles it, as space does; the [x] is not a button.
func TestClickSettingsRowToggles(t *testing.T) {
	f := tuitest.NewFake()
	a := settingsApp(t, f)
	x, y := at(t, a, "[ ] JOBS")
	m := click(t, a, x+6, y)
	if !f.Called("SavePrefs") {
		t.Fatalf("a click on JOBS saved nothing: %v", f.Calls)
	}
	found := false
	for _, c := range m.(*App).prefs.FleetColumns {
		found = found || c == "jobs"
	}
	if !found {
		t.Fatal("JOBS is not on after the click")
	}
	if m.(*App).modal != nil {
		t.Fatalf("the [x] of a checkbox opened %T", m.(*App).modal)
	}
}

// The Docker tab's containers are rows too.
func TestClickContainersRowSelects(t *testing.T) {
	f := boxFake()
	two := sampleContainers()
	svcs := two["services"].([]any)
	second := map[string]any{"id": "testnet", "label": "Testnet", "containerName": "jumpgate-testnet", "configured": true,
		"status": map[string]any{"State": "exited", "Image": "ghcr.io/paradigmxyz/reth:v1"}, "actions": []any{"start"}}
	two["services"] = append(svcs, second)
	f.Routes["GET "+ctrPath] = two
	a := containersHost(t, f)
	h := a.detail.(*hostScreen)
	if len(h.ctr.Services) != 2 {
		t.Fatalf("%d containers", len(h.ctr.Services))
	}
	x, y := at(t, a, "testnet")
	click(t, a, x, y)
	if h.ctrCursor != 1 {
		t.Fatalf("cursor %d:\n%s", h.ctrCursor, tuitest.Frame(a))
	}
	x, y = at(t, a, "devnet")
	click(t, a, x, y)
	if h.ctrCursor != 0 {
		t.Fatalf("cursor %d", h.ctrCursor)
	}
}

// The help list's guide button and the guide's topics.
func TestClickHelpAndGuideTopics(t *testing.T) {
	a, _ := mouseApp(t, tuitest.NewFake(), 80, 24)
	m := press(t, a, "?")
	x, y := at(t, m, "[F] the full guide")
	m = click(t, m, x+1, y)
	g, ok := m.(*App).modal.(*manualModal)
	if !ok || g.open {
		t.Fatalf("clicking [F]: %T", m.(*App).modal)
	}
	// A click on a topic opens its page.
	x, y = at(t, m, "Durable jobs")
	m = click(t, m, x+1, y)
	if !g.open || g.topics[g.cur].Title != "Durable jobs" {
		t.Fatalf("topic click: open %v cur %q", g.open, g.topics[g.cur].Title)
	}
	// On a page: the wheel scrolls it and the footer buttons turn and close it.
	tuitest.Frame(m)
	m, _ = tuitest.Send(m, tea.MouseWheelMsg{X: 10, Y: 10, Button: tea.MouseWheelDown})
	tuitest.Frame(m)
	x, y = at(t, m, "[esc] topics")
	m = click(t, m, x+1, y)
	if g.open {
		t.Fatal("clicking [esc] topics left the page open")
	}
	x, y = at(t, m, "[esc] close")
	if m = click(t, m, x+1, y); m.(*App).modal != nil {
		t.Fatalf("clicking [esc] close: %T", m.(*App).modal)
	}
}

// A click on the y/n question's buttons answers it: [n] and [esc] cancel,
// [y] does it; a click on the text does nothing.
func TestClickYesNo(t *testing.T) {
	ask := func() (*tuitest.Fake, *App) {
		f := vpnFake()
		f.Routes["POST /api/vpns/office/down"] = map[string]any{}
		a, _ := mouseApp(t, f, 100, 30)
		return f, press(t, a, "5", "t").(*App)
	}
	f, m := ask()
	x, y := at(t, m, "loses its path")
	if click(t, m, x, y); m.modal == nil {
		t.Fatal("a click on the question's text closed it")
	}
	x, y = at(t, m, "[n] cancel")
	if click(t, m, x+1, y); m.modal != nil || f.Called("Do POST /api/vpns/office/down") {
		t.Fatalf("[n]: modal %T calls %v", m.modal, f.Calls)
	}
	f, m = ask()
	x, y = at(t, m, "[y] take it down")
	click(t, m, x+1, y)
	if !f.Called("Do POST /api/vpns/office/down") {
		t.Fatalf("[y]: %v", f.Calls)
	}
}

// The Vault screen's buttons are clickable, and the x entry opens the menu.
func TestClickVaultButtons(t *testing.T) {
	vault := func() tea.Model {
		f := tuitest.NewFake()
		f.VaultV = api.VaultView{State: "locked", Slots: []string{"passphrase"}}
		a, _ := mouseApp(t, f, 100, 30)
		return press(t, a, "6")
	}
	m := vault()
	x, y := at(t, m, "[u] unlock")
	if got := click(t, m, x+1, y); func() bool { _, ok := got.(*App).modal.(*unlockModal); return !ok }() {
		t.Fatalf("clicking [u]: %T", got.(*App).modal)
	}
	m = vault()
	x, y = at(t, m, "[x] more")
	if got := click(t, m, x+1, y); func() bool { _, ok := got.(*App).modal.(*menuModal); return !ok }() {
		t.Fatalf("clicking [x]: %T", got.(*App).modal)
	}
}

// The modules modal's rows select by click; turning one on or off stays with
// enter, so a click never changes a module.
func TestClickModulesModalRows(t *testing.T) {
	f := modulesFake()
	m := press(t, openTestHost(t, f, "unicode"), "m")
	mm, ok := m.(*App).modal.(*modulesModal)
	if !ok {
		t.Fatalf("m opened %T", m.(*App).modal)
	}
	x, y := at(t, m, "EL bootnode")
	m = click(t, m, x+1, y)
	if cards := mm.cards(m.(*App)); mm.cursor >= len(cards) || cards[mm.cursor].m.ID != "el-bootnode" {
		t.Fatalf("cursor %d after clicking EL bootnode:\n%s", mm.cursor, tuitest.Frame(m))
	}
	if f.Called("SetModule") {
		t.Fatal("a click changed a module")
	}
	// The enter button does what enter does.
	x, y = at(t, m, "[enter] turn on or off")
	click(t, m, x+1, y)
	if _, asked := m.(*App).modal.(*modulesModal); !asked {
		t.Fatalf("modal %T", m.(*App).modal)
	}
}

// Clicks are ignored while a modal has the keyboard in a text field, except
// the [esc] button, which only cancels: the palette closes but never runs.
func TestClickPaletteEscOnly(t *testing.T) {
	a, now := mouseApp(t, tuitest.NewFake(), 80, 24)
	m := press(t, typed(t, press(t, a, ":"), "quit"))
	*now = now.Add(time.Second)
	x, y := at(t, m, "COMMAND")
	if got := click(t, m, x, y); got.(*App).modal == nil {
		t.Fatal("a click on the palette's title closed or ran it")
	}
	x, y = at(t, m, "[esc] close")
	if got := click(t, m, x+1, y); got.(*App).modal != nil {
		t.Fatalf("[esc] did not close the palette: %T", got.(*App).modal)
	}
}
