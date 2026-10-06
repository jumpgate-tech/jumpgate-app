// internal/tui/fleet_test.go
package tui

import (
	"regexp"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/apiclient"
	"github.com/valve-tech/jumpgate/internal/tui/tuitest"
)

func sampleFleet() api.Fleet {
	seen, old := testNow.Add(-20*time.Second), testNow.Add(-4*time.Minute)
	lag, pct := uint64(2), 61.0
	return api.Fleet{At: testNow.Add(-5 * time.Second), IntervalSeconds: 15, Rows: []api.FleetRow{
		{
			ID: "box-a", Link: api.LinkAgent, Reachable: true, LastSeen: &seen, ChainID: 369, Network: "PulseChain",
			Agent: "0x1234567890abcdef1234567890abcdef12345678",
			Status: &api.NodeStatus{Overall: api.SyncSynced, HeadLag: &lag, DiskUsedPct: &pct,
				Exec: api.ClientStatus{State: api.SyncSynced, Peers: 48}, Beacon: api.ClientStatus{State: api.SyncSynced, Peers: 72}},
			Disk:     &api.DiskView{ExecBytes: 1_200_000_000_000, BeaconBytes: 100_000_000_000, FreeBytes: 2_000_000_000_000, ExpectedExecBytes: 2_000_000_000_000, Fit: api.FitOK},
			Firewall: &api.FirewallSummary{Pass: 5, Warn: 1, Grade: "warn"},
		},
		{
			ID: "box-b", Link: api.LinkSSHOnly, LastSeen: &old, ChainID: 1, Network: "Ethereum", Stale: api.Stale{Status: true},
			Error:  &api.Error{Message: "dial tcp: connection refused", Code: api.CodeUnreachable},
			Status: &api.NodeStatus{Overall: api.SyncSyncing, Exec: api.ClientStatus{State: api.SyncSyncing, Peers: 3}, Beacon: api.ClientStatus{State: api.SyncSynced, Peers: 9}},
		},
		{ID: "laptop", Link: api.LinkLocalOnly, ThisMachine: true, Error: &api.Error{Message: "not set up", Code: api.CodeTargetNotSetUp}},
	}}
}

// fleetApp is an App on the Fleet screen with the sample fleet delivered.
func fleetApp(t *testing.T, f *tuitest.Fake, glyphs string, state apiclient.ConnState) *App {
	t.Helper()
	a := newTestApp(t, f, 80, 24, glyphs)
	p := f.PrefsV
	p.Glyphs = glyphs
	m, _ := tuitest.Send(a, prefsMsg{p: p}, fleetMsg{u: apiclient.Update[api.Fleet]{Value: sampleFleet(), Has: true, State: state}})
	return m.(*App)
}

func TestFleetFrame(t *testing.T) {
	a := fleetApp(t, tuitest.NewFake(), "unicode", apiclient.Live)
	fr := tuitest.Frame(a)
	for _, want := range []string{"HOST", "LINK", "DISK", "box-a", "agent", "synced", "48/72", "61%", "warn", "box-b", "down 4m", "laptop*", "3 boxes"} {
		if !strings.Contains(fr, want) {
			t.Errorf("fleet lacks %q:\n%s", want, fr)
		}
	}
	tuitest.Golden(t, "fleet_80x24", fr)
	tuitest.Golden(t, "fleet_ascii", tuitest.Frame(fleetApp(t, tuitest.NewFake(), "ascii", apiclient.Live)))
}

// Review Focus 2: a box with no reading shows n/a, never 0 or 0%.
func TestFleetShowsUnavailableNeverZero(t *testing.T) {
	a := fleetApp(t, tuitest.NewFake(), "unicode", apiclient.Live)
	for _, line := range strings.Split(tuitest.Frame(a), "\n") {
		if strings.Contains(line, "laptop") {
			if strings.Count(line, "n/a") < 4 || strings.Contains(line, " 0%") || strings.Contains(line, "0/0") {
				t.Fatalf("laptop row %q", line)
			}
			return
		}
	}
	t.Fatal("no laptop row")
}

// Review Focus 1: while the stream retries, every row is marked stale.
func TestFleetGreysRowsWhileTheStreamRetries(t *testing.T) {
	a := fleetApp(t, tuitest.NewFake(), "unicode", apiclient.Retrying)
	fr := tuitest.Frame(a)
	if !strings.Contains(fr, "synced~") || !strings.Contains(fr, "server retrying") {
		t.Fatalf("frame:\n%s", fr)
	}
}

func TestFleetFilterAndSort(t *testing.T) {
	a := fleetApp(t, tuitest.NewFake(), "unicode", apiclient.Live)
	m, _ := tuitest.Send(a, tuitest.Key("/"))
	m, _ = tuitest.Send(m, tuitest.Type("ether")...)
	m, _ = tuitest.Send(m, tuitest.Key("enter"))
	rows, _ := m.(*App).screens[scrFleet].(*fleetScreen).rows(m.(*App))
	if len(rows) != 1 || rows[0].ID != "box-b" {
		t.Fatalf("filtered rows %+v", rows)
	}
	m, _ = tuitest.Send(m, tuitest.Key("/"), tuitest.Key("esc"), tuitest.Key("s"))
	rows, _ = m.(*App).screens[scrFleet].(*fleetScreen).rows(m.(*App))
	if rows[0].ID != "laptop" || rows[2].ID != "box-a" {
		t.Fatalf("sort by sync (worst first): %v %v %v", rows[0].ID, rows[1].ID, rows[2].ID)
	}
}

func TestFleetChainFilterHidesRows(t *testing.T) {
	f := tuitest.NewFake()
	f.PrefsV.Chains = []int{369}
	a := fleetApp(t, f, "unicode", apiclient.Live)
	fr := tuitest.Frame(a)
	if strings.Contains(fr, "box-b") || !strings.Contains(fr, "1 hidden by the chain filter") || !strings.Contains(fr, "laptop") {
		t.Fatalf("frame:\n%s", fr)
	}
}

func TestFleetEnterOpensTheHost(t *testing.T) {
	a := fleetApp(t, tuitest.NewFake(), "unicode", apiclient.Live)
	m, _ := tuitest.Send(a, tuitest.Key("j"), tuitest.Key("enter"))
	h, ok := m.(*App).detail.(*hostScreen)
	if !ok || h.id != "box-b" {
		t.Fatalf("detail %#v", m.(*App).detail)
	}
}

func TestFleetColumnsFollowThePreferences(t *testing.T) {
	f := tuitest.NewFake()
	f.PrefsV.FleetColumns = []string{"host", "jobs", "agent"}
	fr := tuitest.Frame(fleetApp(t, f, "unicode", apiclient.Live))
	if !strings.Contains(fr, "JOBS") || !strings.Contains(fr, "AGENT") || strings.Contains(fr, "DISK") || !strings.Contains(fr, "0x1234…5678") {
		t.Fatalf("frame:\n%s", fr)
	}
}

// Every column at 80 wide: the extra ones are cut with a marker, nothing
// wraps.
func TestFleetCutsColumnsThatDoNotFit(t *testing.T) {
	f := tuitest.NewFake()
	for _, c := range api.FleetColumns {
		if !c.Default {
			f.PrefsV.FleetColumns = append(f.PrefsV.FleetColumns, c.ID)
		}
	}
	fr := tuitest.Frame(fleetApp(t, f, "unicode", apiclient.Live))
	for _, line := range strings.Split(fr, "\n") {
		if lipgloss.Width(line) > 80 {
			t.Fatalf("line wider than 80: %q", line)
		}
	}
	if !strings.Contains(fr, "›") {
		t.Fatalf("no cut marker:\n%s", fr)
	}
}

func TestFleetMouseClickSelects(t *testing.T) {
	f := tuitest.NewFake()
	f.PrefsV.Mouse = true
	a := fleetApp(t, f, "unicode", apiclient.Live)
	m, _ := tuitest.Send(a, tea.MouseClickMsg{X: 3, Y: 4, Button: tea.MouseLeft})
	if c := m.(*App).screens[scrFleet].(*fleetScreen).cursor; c != 2 {
		t.Fatalf("cursor %d after clicking the third row", c)
	}
}

// The filter field must not read the clipboard on ctrl+v; a bracketed paste
// still lands in it.
func TestFleetFilterIgnoresCtrlVButTakesAPaste(t *testing.T) {
	a := fleetApp(t, tuitest.NewFake(), "unicode", apiclient.Live)
	m, _ := tuitest.Send(a, tuitest.Key("/"))
	m, cmds := tuitest.Send(m, tuitest.Key("ctrl+v"))
	if len(cmds) != 0 {
		t.Fatalf("ctrl+v returned %d commands", len(cmds))
	}
	m, _ = tuitest.Send(m, tea.PasteMsg{Content: "ether\x1b]52;c;ZXZpbA==\x07\n"})
	fs := m.(*App).screens[scrFleet].(*fleetScreen)
	if fs.filter.Value() != "ether" {
		t.Fatalf("filter %q after a paste", fs.filter.Value())
	}
	if rows, _ := fs.rows(m.(*App)); len(rows) != 1 {
		t.Fatalf("%d rows after pasting a filter", len(rows))
	}
}

var sgr = regexp.MustCompile("\x1b\\[[0-9;:]*m")

// Hostile data: ids, networks, agent addresses and states from a box or a
// compromised server must not reach the terminal as control sequences, add
// rows or push a row past the window.
func TestFleetHostileRows(t *testing.T) {
	hostile := []string{
		"\x1b[2J\x1b[Hpwn", "\x1b]8;;http://evil\x07link\x1b]8;;\x07", "\x1b]52;c;ZXZpbA==\x07clip",
		"line1\r\nline2\nline3", "\u202eevil\u202c", "日本語のとても長いホスト名ですよ", "😀😀😀😀😀😀😀😀😀😀😀😀😀😀", "x\x9b31my",
	}
	f := tuitest.NewFake()
	f.PrefsV.FleetColumns = []string{"host", "link", "network", "sync", "el", "agent", "jobs", "age"}
	var fl api.Fleet
	fl.At = testNow
	seen := testNow.Add(-time.Minute)
	for _, h := range hostile {
		fl.Rows = append(fl.Rows, api.FleetRow{
			ID: h, Network: h, Agent: h, Link: api.LinkAgent, LastSeen: &seen,
			Status: &api.NodeStatus{Overall: api.SyncState(h), Exec: api.ClientStatus{State: api.SyncState(h)}},
			Error:  &api.Error{Message: h},
		})
	}
	a := newTestApp(t, f, 80, 24, "unicode")
	base := strings.Count(a.View().Content, "\n")
	p := f.PrefsV
	p.Glyphs = "unicode"
	m, _ := tuitest.Send(a, prefsMsg{p: p}, fleetMsg{u: apiclient.Update[api.Fleet]{Value: fl, Has: true, State: apiclient.Live}})
	for _, step := range []tea.Msg{nil, tuitest.Key("j"), tuitest.Key("s")} {
		if step != nil {
			m, _ = tuitest.Send(m, step)
		}
		raw := m.View().Content
		if strings.Contains(sgr.ReplaceAllString(raw, ""), "\x1b") || strings.ContainsAny(raw, "\r\u202e\u009b") {
			t.Fatalf("a control sequence from the data reached the frame:\n%q", raw)
		}
		lines := strings.Split(raw, "\n")
		if len(lines) != base+1 || len(lines) != 24 {
			t.Fatalf("frame has %d lines, want 24", len(lines))
		}
		for _, l := range lines {
			if w := ansi.StringWidth(l); w > 80 {
				t.Fatalf("line is %d wide: %q", w, l)
			}
		}
		if fr := tuitest.Frame(m); strings.Contains(fr, "52;c") || strings.Contains(fr, "evil") && strings.Contains(fr, "http://") {
			t.Fatalf("payload text of a sequence survived:\n%s", fr)
		}
	}
}

// A box that has a status but a client with no reading is n/a, not "0/0".
func TestFleetUnavailableClientsAreNotZero(t *testing.T) {
	fl := api.Fleet{At: testNow, Rows: []api.FleetRow{{ID: "b", Link: api.LinkAgent, Reachable: true, Status: &api.NodeStatus{
		Overall: api.SyncUnavailable, Exec: api.ClientStatus{State: api.SyncUnavailable}, Beacon: api.ClientStatus{State: api.SyncUnavailable}}}}}
	a := newTestApp(t, tuitest.NewFake(), 80, 24, "unicode")
	m, _ := tuitest.Send(a, fleetMsg{u: apiclient.Update[api.Fleet]{Value: fl, Has: true, State: apiclient.Live}})
	line := ""
	for _, l := range strings.Split(tuitest.Frame(m), "\n") {
		if strings.HasPrefix(l, "▸b ") {
			line = l
		}
	}
	if strings.Contains(line, "0/0") || strings.Contains(line, "synced") || strings.Count(line, "n/a") < 3 {
		t.Fatalf("row %q", line)
	}
}

func TestSyncRankPutsUnavailableSecondAndStaleMarksLagPeersAge(t *testing.T) {
	row := func(st api.SyncState) api.FleetRow { return api.FleetRow{Status: &api.NodeStatus{Overall: st}} }
	want := map[api.SyncState]int{api.SyncStopped: 0, api.SyncUnavailable: 1, "": 1, api.SyncSyncing: 2, api.SyncNoData: 3, api.SyncSynced: 4}
	for st, rank := range want {
		if got := syncRank(row(st)); got != rank {
			t.Errorf("syncRank(%q) = %d, want %d", st, got, rank)
		}
	}
	if syncRank(api.FleetRow{}) != 1 {
		t.Error("a row with no status ranks second")
	}
}
