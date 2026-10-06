package tui

import (
	"errors"
	"runtime"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/apiclient"
	"github.com/valve-tech/jumpgate/internal/tui/tuitest"
)

func syncedStatus() api.NodeStatus {
	lag := uint64(0)
	pct := 61.0
	return api.NodeStatus{
		At:      testNow.Add(-3 * time.Second),
		Exec:    api.ClientStatus{Active: true, Head: 21345678, Peers: 48, State: api.SyncSynced},
		Beacon:  api.ClientStatus{Active: true, Head: 9876543, Peers: 72, State: api.SyncSynced},
		RefHead: 21345678, HeadLag: &lag, Overall: api.SyncSynced, DiskUsedPct: &pct,
	}
}

func sampleDisk() api.DiskView {
	g, d := int64(12_300_000_000), 160.0
	return api.DiskView{
		At: testNow.Add(-4 * time.Minute), ExecBytes: 1_200_000_000_000, BeaconBytes: 100_000_000_000, FreeBytes: 2_000_000_000_000,
		ExpectedExecBytes: 2_000_000_000_000, ExpectedBeaconBytes: 250_000_000_000, ExpectedLabel: "estimate", Fit: api.FitOK,
		History: []api.DiskSample{{UsedBytes: 1}, {UsedBytes: 2}, {UsedBytes: 4}, {UsedBytes: 8}}, GrowthPerDay: &g, DaysToFull: &d,
	}
}

// openTestHost opens box with its status stream already holding one reading.
func openTestHost(t *testing.T, f *tuitest.Fake, glyphs string) *App {
	t.Helper()
	f.Status("box") <- apiclient.Update[api.NodeStatus]{Value: syncedStatus(), Has: true, State: apiclient.Live}
	a := newTestApp(t, f, 80, 24, glyphs)
	return tuitest.Settle(a, a.openHost("box")).(*App)
}

func TestOverviewShowsLiveStatus(t *testing.T) {
	a := openTestHost(t, tuitest.NewFake(), "unicode")
	fr := tuitest.Frame(a)
	for _, want := range []string{"Overview", "Storage", "synced", "21,345,678", "9,876,543", "48", "72", "0 blocks"} {
		if !strings.Contains(fr, want) {
			t.Errorf("overview lacks %q:\n%s", want, fr)
		}
	}
	tuitest.Golden(t, "host_overview", fr)
}

// Honest data: before the first reading the overview says so; it never
// shows a head of 0.
func TestOverviewWithoutAReadingSaysSo(t *testing.T) {
	a := newTestApp(t, tuitest.NewFake(), 80, 24, "unicode")
	m := tuitest.Settle(a, a.openHost("box"))
	fr := tuitest.Frame(m)
	if !strings.Contains(fr, "waiting for the first reading") || strings.Contains(fr, "head    0") {
		t.Fatalf("frame:\n%s", fr)
	}
}

func TestOverviewMarksAStaleReading(t *testing.T) {
	f := tuitest.NewFake()
	st := syncedStatus()
	st.At = testNow.Add(-2 * time.Minute)
	f.Status("box") <- apiclient.Update[api.NodeStatus]{Value: st, Has: true, State: apiclient.Retrying}
	a := newTestApp(t, f, 80, 24, "unicode")
	m := tuitest.Settle(a, a.openHost("box"))
	if fr := tuitest.Frame(m); !strings.Contains(fr, "stale: 2m old") {
		t.Fatalf("frame:\n%s", fr)
	}
}

func TestStorageLoadsAndMeasures(t *testing.T) {
	f := tuitest.NewFake()
	f.DiskV["box"] = sampleDisk()
	a := openTestHost(t, f, "unicode")
	m, cmds := tuitest.Send(a, tuitest.Key("right"))
	m = tuitest.Settle(m, cmds...)
	if !f.Called("Disk box") {
		t.Fatal("the storage tab did not load the disk view")
	}
	fr := tuitest.Frame(m)
	for _, want := range []string{"1.20 TB", "2.00 TB", "(estimate)", "room for the estimate with 10% headroom", "+12.3 GB/day", "160 days"} {
		if !strings.Contains(fr, want) {
			t.Errorf("storage lacks %q:\n%s", want, fr)
		}
	}
	tuitest.Golden(t, "host_storage", fr)
	m, cmds = tuitest.Send(m, tuitest.Key("m"))
	m = tuitest.Settle(m, cmds...)
	if !f.Called("MeasureDisk box") {
		t.Fatal("m did not measure")
	}
}

func TestEscClosesTheHostAndStopsItsStreams(t *testing.T) {
	a := openTestHost(t, tuitest.NewFake(), "unicode")
	h := a.detail.(*hostScreen)
	m, _ := tuitest.Send(a, tuitest.Key("esc"))
	if m.(*App).detail != nil || h.ctx.Err() == nil {
		t.Fatal("esc must close the host and cancel its streams")
	}
}

func TestSidebarAppearsAtWideWidths(t *testing.T) {
	f := tuitest.NewFake()
	f.FleetCh <- apiclient.Update[api.Fleet]{Has: true, State: apiclient.Live, Value: api.Fleet{At: testNow, Rows: []api.FleetRow{{ID: "box"}, {ID: "other"}}}}
	a := openTestHost(t, f, "unicode")
	m := tuitest.Settle(a, a.Init())
	m, _ = tuitest.Send(m, tea.WindowSizeMsg{Width: 120, Height: 30})
	if fr := tuitest.Frame(m); !strings.Contains(fr, "other") {
		t.Fatalf("no sidebar at 120 columns:\n%s", fr)
	}
}

// Leaving a box ends its stream readers: closing the detail and opening
// another box both cancel the screen's ctx, and nothing stays behind.
func TestLeavingAHostEndsItsStreamReaders(t *testing.T) {
	f := tuitest.NewFake()
	a := newTestApp(t, f, 80, 24, "unicode")
	cmd := a.openHost("box")
	h := a.detail.(*hostScreen)
	read := make(chan tea.Msg, 1)
	go func() { read <- h.watchStatus(f.Status("box"))() }() // blocks: the stream is silent
	time.Sleep(20 * time.Millisecond)
	_ = cmd
	a.openHost("other") // switching hosts
	select {
	case msg := <-read:
		if msg != nil {
			t.Fatalf("a reader of the old box returned %v", msg)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the old box's stream reader is still blocked after switching hosts")
	}
	if h.ctx.Err() == nil {
		t.Fatal("switching hosts did not cancel the old box's context")
	}
	h2 := a.detail.(*hostScreen)
	a.closeDetail()
	if h2.ctx.Err() == nil {
		t.Fatal("closeDetail did not cancel the context")
	}
	waitNoRelay(t)
}

// waitNoRelay waits for every tuitest relay goroutine (the fake's stream
// forwarders, which end when their ctx does) to be gone.
func waitNoRelay(t *testing.T) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for {
		buf := make([]byte, 1<<20)
		buf = buf[:runtime.Stack(buf, true)]
		if !strings.Contains(string(buf), "tuitest.relay") {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("a stream goroutine outlived its screen:\n%s", buf)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// The ninth window on a box is refused (429 too_many_streams); the overview
// says so as an error state, without a reading and with the old one kept.
func TestOverviewShowsTooManyStreams(t *testing.T) {
	tooMany := &api.Error{Message: "too many status streams on this box", Code: api.CodeTooManyStreams, Status: 429}
	f := tuitest.NewFake()
	f.Status("box") <- apiclient.Update[api.NodeStatus]{State: apiclient.Retrying, Err: tooMany}
	a := newTestApp(t, f, 80, 24, "unicode")
	m := tuitest.Settle(a, a.openHost("box"))
	if fr := tuitest.Frame(m); !strings.Contains(fr, "status unavailable: too many windows are watching this box") {
		t.Fatalf("no reading:\n%s", fr)
	}
	// With a reading already shown, the error is a note and the reading stays.
	// (Messages go in directly: the settled watch above may still hold the channel.)
	m, _ = tuitest.Send(m,
		statusMsg{id: "box", u: apiclient.Update[api.NodeStatus]{Value: syncedStatus(), Has: true, State: apiclient.Live}},
		statusMsg{id: "box", u: apiclient.Update[api.NodeStatus]{State: apiclient.Retrying, Err: tooMany}})
	fr := tuitest.Frame(m)
	if !strings.Contains(fr, "21,345,678") || !strings.Contains(fr, "latest reading failed: too many windows") {
		t.Fatalf("with a reading:\n%s", fr)
	}
}

// A reading goes stale as the clock runs even if no update arrives.
func TestOverviewGoesStaleWithTheClock(t *testing.T) {
	a := openTestHost(t, tuitest.NewFake(), "unicode")
	if strings.Contains(tuitest.Frame(a), "stale") {
		t.Fatal("a 3s old reading is stale")
	}
	later := testNow.Add(time.Minute)
	a.now = func() time.Time { return later }
	if fr := tuitest.Frame(a); !strings.Contains(fr, "stale: 1m old") {
		t.Fatalf("frame:\n%s", fr)
	}
}

// An inactive client shows n/a, never a head of 0.
func TestOverviewInactiveClientIsNotZero(t *testing.T) {
	f := tuitest.NewFake()
	st := syncedStatus()
	st.Beacon = api.ClientStatus{State: api.SyncStopped}
	f.Status("box") <- apiclient.Update[api.NodeStatus]{Value: st, Has: true, State: apiclient.Live}
	a := newTestApp(t, f, 80, 24, "unicode")
	m := tuitest.Settle(a, a.openHost("box"))
	if fr := tuitest.Frame(m); !strings.Contains(fr, "slot      n/a") {
		t.Fatalf("frame:\n%s", fr)
	}
}

// Server, box and config text on the host screens cannot escape: no ESC from
// the data, and the frame keeps its height.
func TestHostScreensCannotBeEscaped(t *testing.T) {
	id := "box" + hostile
	f := tuitest.NewFake()
	st := syncedStatus()
	st.Overall = api.SyncState("synced" + hostile)
	st.Exec.State = api.SyncState("x" + hostile)
	f.Status(id) <- apiclient.Update[api.NodeStatus]{Value: st, Has: true, State: apiclient.Retrying, Err: errors.New("dial: " + hostile)}
	d := sampleDisk()
	d.ExpectedLabel = "estimate" + hostile
	d.Fit = api.FitVerdict("odd" + hostile)
	f.DiskV[id] = d
	f.Err["MeasureDisk"] = &api.Error{Message: "failed " + hostile, Hint: hostile}
	f.FleetCh <- apiclient.Update[api.Fleet]{Has: true, State: apiclient.Live, Value: api.Fleet{At: testNow, Rows: []api.FleetRow{{ID: id, Network: "net" + hostile}, {ID: "o" + hostile}}}}
	for _, w := range []int{80, 120} {
		a := newTestApp(t, f, w, 24, "unicode")
		a.th = plainTheme()
		a.setPrefs(api.UIPrefs{Glyphs: "unicode", ShowEstimates: true, HostTabs: append(api.HostTabs, "tab"+hostile)})
		m := tuitest.Settle(a, a.Init())
		m = tuitest.Settle(m, a.openHost(id))
		requireClean(t, "overview", m.View().Content)
		m, cmds := tuitest.Send(m, tuitest.Key("right"))
		m = tuitest.Settle(m, cmds...)
		requireClean(t, "storage", m.View().Content)
		m, cmds = tuitest.Send(m, tuitest.Key("m"))
		m = tuitest.Settle(m, cmds...)
		requireClean(t, "storage after a failed measure", m.View().Content)
		a.detail.(*hostScreen).tab = 6 // the hostile tab name
		requireClean(t, "unknown tab", m.View().Content)
	}
}
