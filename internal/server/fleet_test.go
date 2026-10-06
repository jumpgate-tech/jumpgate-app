package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/fstest"
	"time"

	"github.com/valve-tech/jumpgate/internal/agentclient"
	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/catalog"
	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/executor"
	"github.com/valve-tech/jumpgate/internal/monitor"
	"github.com/valve-tech/jumpgate/internal/ops"
)

// fakeProber answers for every box at once. down boxes refuse the
// connection; hang boxes block until release is closed, ignoring their
// context the way a probe waiting on a target's intent lock does; delay
// slows every answer so tests can watch the fan-out.
type fakeProber struct {
	mu        sync.Mutex
	down      map[string]bool
	hang      map[string]bool
	release   chan struct{}
	delay     time.Duration
	slow      map[string]time.Duration // per-box delay, on top of delay
	diskFail  bool
	fwFail    bool
	hung      chan string // when set, receives a box's id as its hung probe starts
	diskCalls int
	// diskGate, when set, holds every disk probe until it is closed;
	// diskEntered receives one value as each probe starts.
	diskGate    chan struct{}
	diskEntered chan struct{}
	fwCalls     int
	calls       map[string]int

	inFlight, maxInFlight atomic.Int32
}

func (f *fakeProber) enter(id string) {
	n := f.inFlight.Add(1)
	for {
		m := f.maxInFlight.Load()
		if n <= m || f.maxInFlight.CompareAndSwap(m, n) {
			break
		}
	}
	f.mu.Lock()
	f.calls[id]++
	f.mu.Unlock()
}

func (f *fakeProber) status(ctx context.Context, _ config.Config, t config.Target) (monitor.Snapshot, error) {
	f.enter(t.ID)
	defer f.inFlight.Add(-1)
	f.mu.Lock()
	down, hang, delay, hung := f.down[t.ID], f.hang[t.ID], f.delay+f.slow[t.ID], f.hung
	f.mu.Unlock()
	if hang {
		if hung != nil {
			hung <- t.ID
		}
		<-f.release
		return monitor.Snapshot{}, errors.New("released")
	}
	if delay > 0 {
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return monitor.Snapshot{}, ctx.Err()
		}
	}
	if down {
		return monitor.Snapshot{}, fmt.Errorf("%w: ssh %s: connection refused", agentclient.ErrUnreachable, t.ID)
	}
	return monitor.Snapshot{At: time.Now(), ExecActive: true, BeaconActive: true, ExecHead: 100, BeaconSlot: 7, ExecPeers: 12, BeaconPeers: 40, RefHead: 100, DiskKnown: true, DiskUsedPct: 61}, nil
}

func fakeDU() ops.DU {
	return ops.DU{ExecBytes: 1e12, BeaconBytes: 1e11, DiskFreeBytes: 2e12, DiskFreeKnown: true, ExpectedExecBytes: 2e12}
}

func (f *fakeProber) diskCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.diskCalls
}

func (f *fakeProber) disk(ctx context.Context, _ config.Config, _ config.Target) (ops.DU, error) {
	f.mu.Lock()
	f.diskCalls++
	fail, gate, entered := f.diskFail, f.diskGate, f.diskEntered
	f.mu.Unlock()
	if entered != nil {
		entered <- struct{}{}
	}
	if gate != nil {
		select {
		case <-gate:
		case <-ctx.Done():
			return ops.DU{}, ctx.Err()
		}
	}
	if fail {
		return ops.DU{}, errors.New("du: cannot access")
	}
	return fakeDU(), nil
}

func (f *fakeProber) firewall(context.Context, config.Config, config.Target) ([]ops.CheckItem, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.fwCalls++
	if f.fwFail {
		return nil, errors.New("ss: not found")
	}
	return []ops.CheckItem{{Status: "pass"}, {Status: "warn"}}, nil
}

func (f *fakeProber) chainID(_ context.Context, _ config.Config, t config.Target) (int, error) {
	if t.Wire != nil {
		return t.Wire.ChainID, nil
	}
	return 369, nil
}

func newFakeProber() *fakeProber {
	return &fakeProber{down: map[string]bool{}, hang: map[string]bool{}, slow: map[string]time.Duration{}, release: make(chan struct{}), calls: map[string]int{}}
}

func (f *fakeProber) callsFor(id string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls[id]
}

// fleetHome isolates the test from the real HOME on every platform.
func fleetHome(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
}

func fleetServer(t *testing.T, targets ...config.Target) (*Server, *fakeProber, *httptest.Server, string) {
	t.Helper()
	fleetHome(t)
	if _, err := config.Update(func(c *config.Config) error { c.Targets = targets; return nil }); err != nil {
		t.Fatal(err)
	}
	token := NewSessionToken()
	s := New(Config{Token: token, UI: fstest.MapFS{}})
	fp := newFakeProber()
	s.fleet.probe = fp
	// Cleanups run last first: the HTTP server closes, then the poller stops
	// and waits for its goroutines, before any test restores the timing
	// globals it shortened (Ruling T9).
	t.Cleanup(s.fleet.stop)
	ts := httptest.NewServer(s.Handler())
	t.Cleanup(ts.Close)
	return s, fp, ts, token
}

// fleetGoroutines counts the goroutines the poller owns, for leak checks.
func fleetGoroutines() int {
	buf := make([]byte, 1<<20)
	buf = buf[:runtime.Stack(buf, true)]
	n := 0
	for _, g := range strings.Split(string(buf), "\n\n") {
		if strings.Contains(g, "server.(*fleetPoller)") {
			n++
		}
	}
	return n
}

// requireNoFleetGoroutines fails when any poller goroutine outlives stop.
func requireNoFleetGoroutines(t *testing.T) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for fleetGoroutines() > 0 {
		if time.Now().After(deadline) {
			buf := make([]byte, 1<<20)
			t.Fatalf("%d fleet goroutines leaked:\n%s", fleetGoroutines(), buf[:runtime.Stack(buf, true)])
		}
		time.Sleep(5 * time.Millisecond)
	}
}

var (
	wired  = &catalog.WireConfig{ChainID: 369, ExecID: "reth", BeaconID: "lighthouse-pulse", DataDir: "/var/lib/x/369"}
	paired = &config.AgentPairing{Address: "0xabc", Transport: "ssh"}
)

func TestFleetRowsShowEveryBox(t *testing.T) {
	s, fp, _, _ := fleetServer(t,
		config.Target{ID: "a", Mode: "ssh", SSH: &executor.SSHConfig{Host: "a", User: "u"}, Agent: paired},
		config.Target{ID: "b", Mode: "ssh", SSH: &executor.SSHConfig{Host: "b", User: "u"}, Wire: wired},
		config.Target{ID: "c", Mode: "ssh", SSH: &executor.SSHConfig{Host: "c", User: "u"}},
	)
	fp.down["b"] = true
	s.fleet.round(context.Background())
	f := s.fleet.snapshot()
	a, _ := f.Row("a")
	if a.Link != api.LinkAgent || !a.Reachable || a.Status == nil || a.Status.Overall != api.SyncSynced || a.Network == "" || a.Agent != "0xabc" {
		t.Fatalf("row a %+v", a)
	}
	if a.Disk == nil || a.Disk.Fit != api.FitOK || a.Firewall == nil || a.Firewall.Grade != "warn" || a.Jobs != nil {
		t.Fatalf("row a sections %+v %+v %+v", a.Disk, a.Firewall, a.Jobs)
	}
	if a.CheckedAt == nil || a.FirewallAt == nil || a.Disk.At.IsZero() || a.Status.At.IsZero() {
		t.Fatalf("row a values carry no age: checked %v firewall %v disk %v status %v", a.CheckedAt, a.FirewallAt, a.Disk.At, a.Status.At)
	}
	b, _ := f.Row("b")
	if b.Reachable || b.Status != nil || b.Error == nil || b.Error.Code != api.CodeUnreachable || b.ChainID != 369 {
		t.Fatalf("row b %+v", b)
	}
	c, _ := f.Row("c")
	if c.Link != api.LinkSSHOnly || c.Error == nil || c.Error.Code != api.CodeTargetNotSetUp || c.Status != nil {
		t.Fatalf("row c %+v", c)
	}
	if len(f.Rows) != 3 || f.Rows[0].ID != "a" || f.Rows[2].ID != "c" {
		t.Fatalf("rows not in config order: %+v", f.Rows)
	}
}

// A box that never answered has no values at all: every section is nil
// (unavailable), never a zero reading, and nothing is marked seen.
func TestFleetUnknownStaysUnknown(t *testing.T) {
	s, fp, _, _ := fleetServer(t, config.Target{ID: "a", Mode: "ssh", Agent: paired})
	fp.down["a"] = true
	s.fleet.round(context.Background())
	a, _ := s.fleet.snapshot().Row("a")
	if a.Status != nil || a.Disk != nil || a.Firewall != nil || a.Jobs != nil || a.LastSeen != nil || a.FirewallAt != nil {
		t.Fatalf("a box that never answered shows values: %+v", a)
	}
	if a.Reachable || a.Stale != (api.Stale{}) || a.Error == nil || a.CheckedAt == nil {
		t.Fatalf("row %+v", a)
	}
}

// Before the first poll every box is listed, as loading: no values, no
// error and no check time, never an empty fleet.
func TestFleetListsEveryBoxBeforeTheFirstPoll(t *testing.T) {
	_, _, ts, token := fleetServer(t, config.Target{ID: "a", Mode: "ssh", Agent: paired}, config.Target{ID: "b", Mode: "ssh", Wire: wired})
	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/fleet", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var f api.Fleet
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d", res.StatusCode)
	}
	if err := json.NewDecoder(res.Body).Decode(&f); err != nil {
		t.Fatal(err)
	}
	if len(f.Rows) != 2 || f.IntervalSeconds != 15 {
		t.Fatalf("fleet %+v", f)
	}
	b := f.Rows[1]
	if b.ID != "b" || b.Link != api.LinkSSHOnly || b.ChainID != 369 || b.Network == "" || b.Status != nil || b.Error != nil || b.CheckedAt != nil || b.Reachable {
		t.Fatalf("row b before the first poll %+v", b)
	}
}

// Review Focus 2: a box that stops answering keeps its last values, aged
// and marked stale, never zeroed.
func TestFleetKeepsLastValuesAndMarksThemStale(t *testing.T) {
	s, fp, _, _ := fleetServer(t, config.Target{ID: "a", Mode: "ssh", SSH: &executor.SSHConfig{Host: "a", User: "u"}, Agent: paired})
	t0 := time.Unix(1_800_000_000, 0)
	now := t0
	s.fleet.now = func() time.Time { return now }
	s.fleet.round(context.Background())
	fp.down["a"] = true
	now = t0.Add(50 * time.Second) // more than 3 × 15 s
	s.fleet.round(context.Background())
	a, _ := s.fleet.snapshot().Row("a")
	if a.Status == nil || a.Status.Exec.Head != 100 {
		t.Fatalf("last status was dropped: %+v", a.Status)
	}
	if !a.Stale.Status || a.Stale.Disk || a.Stale.Firewall || a.Reachable || a.Error == nil || a.LastSeen == nil || !a.LastSeen.Equal(t0) {
		t.Fatalf("row %+v", a)
	}
	if a.CheckedAt == nil || !a.CheckedAt.Equal(now) {
		t.Fatalf("checked %v, want %v", a.CheckedAt, now)
	}
	now = t0.Add(31 * time.Minute) // past 3 × 10 min: disk and firewall too
	a, _ = s.fleet.snapshot().Row("a")
	if !a.Stale.Disk || !a.Stale.Firewall || a.Disk == nil || a.Firewall == nil {
		t.Fatalf("old disk and firewall are not marked stale: %+v", a)
	}
}

func TestFleetPollsDiskAndFirewallLessOften(t *testing.T) {
	s, fp, _, _ := fleetServer(t, config.Target{ID: "a", Mode: "ssh", SSH: &executor.SSHConfig{Host: "a", User: "u"}, Agent: paired})
	t0 := time.Unix(1_800_000_000, 0)
	for i := 0; i < 4; i++ {
		at := t0.Add(time.Duration(i) * 15 * time.Second)
		s.fleet.now = func() time.Time { return at }
		s.fleet.round(context.Background())
	}
	if fp.diskCalls != 1 || fp.fwCalls != 1 {
		t.Fatalf("disk %d firewall %d in one minute, want 1 and 1", fp.diskCalls, fp.fwCalls)
	}
	if h := s.fleet.history("a"); len(h) != 1 || h[0].UsedBytes != 1.1e12 {
		t.Fatalf("history %+v", h)
	}
}

func TestDiskHistoryIsBounded(t *testing.T) {
	s, _, _, _ := fleetServer(t, config.Target{ID: "a", Mode: "ssh", Agent: paired})
	for i := 0; i < diskHistorySize+12; i++ {
		s.fleet.recordDisk("a", time.Unix(int64(i), 0), ops.DU{ExecBytes: uint64(i)})
	}
	h := s.fleet.history("a")
	if len(h) != diskHistorySize || h[0].UsedBytes != 12 {
		t.Fatalf("history len %d, want %d starting at 12", len(h), diskHistorySize)
	}
}

// Fifty boxes never mean fifty sessions at once.
func TestFleetFanOutIsBounded(t *testing.T) {
	var targets []config.Target
	for i := 0; i < 12; i++ {
		targets = append(targets, config.Target{ID: fmt.Sprintf("box%02d", i), Mode: "ssh", Agent: paired})
	}
	s, fp, _, _ := fleetServer(t, targets...)
	fp.delay = 20 * time.Millisecond
	s.fleet.round(context.Background())
	if m := fp.maxInFlight.Load(); m > fleetWorkers || m < 2 {
		t.Fatalf("%d probes at once, want 2..%d", m, fleetWorkers)
	}
	if f := s.fleet.snapshot(); len(f.Rows) != 12 || f.Rows[11].Status == nil {
		t.Fatalf("not every box was polled: %+v", f.Rows)
	}
}

// One box that never answers costs its row a timeout, not the fleet a round:
// the others are fresh, it reads unreachable with its last check time, and
// it is not asked again while its last probe is still running.
func TestFleetSlowBoxDoesNotStallTheRound(t *testing.T) {
	old := fleetProbeTimeout
	fleetProbeTimeout = 50 * time.Millisecond
	t.Cleanup(func() { fleetProbeTimeout = old })
	s, fp, _, _ := fleetServer(t, config.Target{ID: "slow", Mode: "ssh", Agent: paired}, config.Target{ID: "ok", Mode: "ssh", Agent: paired})
	t.Cleanup(func() { close(fp.release) }) // runs before stop: the hung probe returns
	fp.hang["slow"] = true
	done := make(chan struct{})
	go func() {
		s.fleet.round(context.Background())
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("one hung box stalled the round")
	}
	f := s.fleet.snapshot()
	slow, _ := f.Row("slow")
	if slow.Error == nil || slow.Error.Code != api.CodeUnreachable || !strings.Contains(slow.Error.Message, "did not answer") || slow.Reachable || slow.CheckedAt == nil {
		t.Fatalf("slow row %+v %+v", slow, slow.Error)
	}
	if ok, _ := f.Row("ok"); ok.Status == nil || !ok.Reachable {
		t.Fatalf("ok row %+v", ok)
	}
	s.fleet.round(context.Background())
	if n := fp.callsFor("slow"); n != 1 {
		t.Fatalf("the hung box was probed %d times, want 1 while its probe runs", n)
	}
}

func TestFleetPollerStopsWhenNobodyWatches(t *testing.T) {
	oldEvery, oldIdle := fleetStatusEvery, fleetIdleStop
	fleetStatusEvery, fleetIdleStop = 10*time.Millisecond, 50*time.Millisecond
	t.Cleanup(func() { fleetStatusEvery, fleetIdleStop = oldEvery, oldIdle })
	s, fp, _, _ := fleetServer(t, config.Target{ID: "a", Mode: "ssh", Agent: paired})
	s.fleet.want()
	deadline := time.Now().Add(2 * time.Second)
	for s.fleet.isRunning() {
		if time.Now().After(deadline) {
			t.Fatal("the poller kept running with nobody watching")
		}
		time.Sleep(10 * time.Millisecond)
	}
	requireNoFleetGoroutines(t)
	if fp.callsFor("a") == 0 {
		t.Fatal("the poller never polled while it was wanted")
	}
	n := fp.callsFor("a")
	time.Sleep(50 * time.Millisecond)
	if fp.callsFor("a") != n {
		t.Fatal("a stopped poller kept probing")
	}
}

// A stream subscriber keeps the poller running past the idle time; once the
// last one leaves, it stops and leaves no goroutine behind.
func TestFleetPollerRunsWhileAStreamWatches(t *testing.T) {
	oldEvery, oldIdle, oldPing, oldDebounce := fleetStatusEvery, fleetIdleStop, ssePingInterval, fleetPublishDebounce
	fleetStatusEvery, fleetIdleStop, ssePingInterval, fleetPublishDebounce = 10*time.Millisecond, 30*time.Millisecond, time.Hour, time.Millisecond
	t.Cleanup(func() {
		fleetStatusEvery, fleetIdleStop, ssePingInterval, fleetPublishDebounce = oldEvery, oldIdle, oldPing, oldDebounce
	})
	s, _, ts, token := fleetServer(t, config.Target{ID: "a", Mode: "ssh", Agent: paired})
	// Six frames at 10 ms per round span well over the 30 ms idle time.
	if frames, _ := readFrames(t, ts, token, "/api/fleet/stream", 6); len(frames) != 6 {
		t.Fatalf("frames %q", frames)
	}
	deadline := time.Now().Add(2 * time.Second)
	for s.fleet.isRunning() {
		if time.Now().After(deadline) {
			t.Fatal("the poller kept running after the stream closed")
		}
		time.Sleep(10 * time.Millisecond)
	}
	requireNoFleetGoroutines(t)
}

// stop ends a running poller, waits for it, and keeps it stopped.
func TestFleetPollerStopIsFinal(t *testing.T) {
	oldEvery := fleetStatusEvery
	fleetStatusEvery = 10 * time.Millisecond
	t.Cleanup(func() { fleetStatusEvery = oldEvery })
	s, _, _, _ := fleetServer(t, config.Target{ID: "a", Mode: "ssh", Agent: paired})
	s.fleet.want()
	s.fleet.stop()
	if s.fleet.isRunning() || fleetGoroutines() != 0 {
		t.Fatalf("running %v, %d goroutines after stop", s.fleet.isRunning(), fleetGoroutines())
	}
	s.fleet.want()
	if s.fleet.isRunning() {
		t.Fatal("want restarted a stopped poller")
	}
	s.fleet.stop() // idempotent
}

func TestFleetStreamSendsSnapshots(t *testing.T) {
	_, _, ts, token := fleetServer(t, config.Target{ID: "a", Mode: "ssh", Agent: paired})
	// The first frame is the fleet as of the connection (possibly before the
	// first poll); the poll round that the connection starts sends the next.
	frames, _ := readFrames(t, ts, token, "/api/fleet/stream", 2)
	if !strings.HasPrefix(frames[1], "data: {") || !strings.Contains(frames[1], `"rows":[{"id":"a"`) {
		t.Fatalf("frames %q", frames)
	}
}

// The real prober reaches a paired box only through signed intents: its
// status, disk, firewall and chain come from the agent and the legacy
// executor is never opened.
func TestFleetPairedBoxUsesIntentsNeverSSH(t *testing.T) {
	requireAgentPeer(t)
	var srv *Server
	exec := &countingExec{}
	pairedBoxWith(t, &autoSucceedExecutor{}, pairedOpts{setUp: true,
		server:   func(c *Config) { c.NewExecutor = exec.newExecutor },
		onServer: func(s *Server) { srv = s; t.Cleanup(s.fleet.stop) },
	})
	srv.fleet.round(context.Background())
	row, ok := srv.fleet.snapshot().Row("box")
	if !ok || row.Link != api.LinkAgentLocal || !row.Reachable || row.Status == nil || row.Error != nil {
		t.Fatalf("row %+v error %+v", row, row.Error)
	}
	if row.ChainID != 369 || row.Network == "" || row.Firewall == nil || row.Disk == nil {
		t.Fatalf("row %+v", row)
	}
	if n := exec.n.Load(); n != 0 {
		t.Fatalf("the legacy executor was opened %d times for a paired box", n)
	}
}

// A paired box whose agent is down is unreachable; the poller never tries
// root SSH instead.
func TestFleetPairedBoxThatIsDownNeverFallsBackToSSH(t *testing.T) {
	requireAgentPeer(t)
	var srv *Server
	exec := &countingExec{}
	pairedBoxWith(t, nopExec{}, pairedOpts{setUp: true,
		server:   func(c *Config) { c.NewExecutor = exec.newExecutor },
		target:   func(tg *config.Target) { tg.Agent.Socket = tg.Agent.Socket + ".gone"; tg.Wire = wired },
		onServer: func(s *Server) { srv = s; t.Cleanup(s.fleet.stop) },
	})
	srv.fleet.round(context.Background())
	row, _ := srv.fleet.snapshot().Row("box")
	if row.Reachable || row.Status != nil || row.Error == nil || row.Error.Code != api.CodeUnreachable {
		t.Fatalf("row %+v error %+v", row, row.Error)
	}
	if n := exec.n.Load(); n != 0 {
		t.Fatalf("the legacy executor was opened %d times for a paired box", n)
	}
}

// An SSH-only box is read with one poll over the server's own executor (the
// legacySSHConfig policy), and the fleet starts no background monitor that
// would keep probing after nobody watches.
func TestFleetSSHOnlyBoxPollsOnceOverTheLegacyExecutor(t *testing.T) {
	fleetHome(t)
	if _, err := config.Update(func(c *config.Config) error {
		c.Targets = []config.Target{{ID: "a", Mode: "ssh", SSH: &executor.SSHConfig{Host: "a", User: "u"}, Wire: wired}}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	ex := &autoSucceedExecutor{}
	opened := 0
	s := New(Config{Token: NewSessionToken(), UI: fstest.MapFS{}, NewExecutor: func(config.Target) (executor.Executor, error) {
		opened++
		return ex, nil
	}})
	t.Cleanup(s.fleet.stop)
	s.fleet.round(context.Background())
	row, _ := s.fleet.snapshot().Row("a")
	if row.Link != api.LinkSSHOnly || !row.Reachable || row.Status == nil || row.Error != nil {
		t.Fatalf("row %+v error %+v", row, row.Error)
	}
	if opened != 1 {
		t.Fatalf("executor opened %d times, want 1", opened)
	}
	entry := s.reg.get("a")
	entry.mu.Lock()
	mon := entry.mon
	entry.mu.Unlock()
	if mon != nil {
		t.Fatal("the fleet started a background monitor for an SSH-only box")
	}
}

// An SSH-only box whose executor will not open is a fleet error, not a
// zeroed "stopped" reading.
func TestFleetSSHOnlyBoxThatWillNotConnectIsAnError(t *testing.T) {
	fleetHome(t)
	if _, err := config.Update(func(c *config.Config) error {
		c.Targets = []config.Target{{ID: "a", Mode: "ssh", SSH: &executor.SSHConfig{Host: "a", User: "u"}, Wire: wired}}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	s := New(Config{Token: NewSessionToken(), UI: fstest.MapFS{}, NewExecutor: func(config.Target) (executor.Executor, error) {
		return brokenExec{}, nil
	}})
	t.Cleanup(s.fleet.stop)
	s.fleet.round(context.Background())
	row, _ := s.fleet.snapshot().Row("a")
	if row.Reachable || row.Status != nil || row.Error == nil || row.Error.Code != api.CodeUnreachable {
		t.Fatalf("row %+v error %+v", row, row.Error)
	}
}

// brokenExec is an executor whose connection has gone away.
type brokenExec struct{ nopExec }

func (brokenExec) Run(context.Context, string, *executor.RunOpts) (executor.Result, error) {
	return executor.Result{}, errors.New("new ssh session: EOF")
}

// Shutting the server down stops the poller within fleetStopWait, even when
// a probe is stuck where its context cannot reach it (an SSH NewSession).
func TestFleetStopsWhenTheServerShutsDown(t *testing.T) {
	oldWait := fleetStopWait
	fleetStopWait = 200 * time.Millisecond
	t.Cleanup(func() { fleetStopWait = oldWait })
	fleetHome(t)
	if _, err := config.Update(func(c *config.Config) error {
		c.Targets = []config.Target{{ID: "a", Mode: "ssh", Agent: paired}}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s, _, errCh := startServing(t, ctx, NewSessionToken(), nil)
	fp := newFakeProber()
	fp.hang["a"], fp.hung = true, make(chan string, 1)
	t.Cleanup(func() { close(fp.release); requireNoFleetGoroutines(t) })
	s.fleet.probe = fp
	s.fleet.want()
	<-fp.hung // mid-round, with a probe that ignores its context
	start := time.Now()
	cancel()
	select {
	case <-errCh:
	case <-time.After(2 * time.Second):
		t.Fatal("the server did not shut down")
	}
	if d := time.Since(start); d > time.Second {
		t.Fatalf("shutdown took %s, want about fleetStopWait", d)
	}
	if s.fleet.isRunning() || loopGoroutines() != 0 {
		t.Fatalf("the poller outlived shutdown: running %v, %d loops", s.fleet.isRunning(), loopGoroutines())
	}
	s.fleet.want()
	if s.fleet.isRunning() {
		t.Fatal("a request after shutdown restarted the poller")
	}
}

// loopGoroutines counts running poller loops.
func loopGoroutines() int {
	buf := make([]byte, 1<<20)
	buf = buf[:runtime.Stack(buf, true)]
	n := 0
	for _, g := range strings.Split(string(buf), "\n\n") {
		if strings.Contains(g, "server.(*fleetPoller).loop") {
			n++
		}
	}
	return n
}

// A want() that lands while a loop is deciding to stop starts exactly one
// new loop, never two: the old loop's exit must not clear the new loop's
// running flag.
func TestFleetWantRacingTheIdleStopRunsOneLoop(t *testing.T) {
	oldEvery, oldIdle := fleetStatusEvery, fleetIdleStop
	fleetStatusEvery, fleetIdleStop = 5*time.Millisecond, 10*time.Millisecond
	t.Cleanup(func() { fleetStatusEvery, fleetIdleStop = oldEvery, oldIdle })
	s, _, _, _ := fleetServer(t, config.Target{ID: "a", Mode: "ssh", Agent: paired})
	var once sync.Once
	var unsub func()
	exited := make(chan struct{})
	s.fleet.testIdleStopped = func() {
		// Between the first loop's decision to stop and its exit, a stream
		// arrives and keeps a new loop running.
		once.Do(func() { _, unsub = s.fleet.subscribe() })
	}
	s.fleet.testLoopDone = func() {
		select {
		case <-exited:
		default:
			close(exited)
		}
	}
	s.fleet.want()
	select {
	case <-exited:
	case <-time.After(2 * time.Second):
		t.Fatal("the first loop never stopped")
	}
	s.fleet.want() // the new loop is running; this must not start a third
	time.Sleep(30 * time.Millisecond)
	if m := s.fleet.maxActiveLoops(); m != 1 {
		t.Fatalf("%d loops ran at once, want 1", m)
	}
	unsub()
}

// Frames go out as boxes finish, not only when the slowest one does.
func TestFleetStreamSendsEachBoxAsItFinishes(t *testing.T) {
	old := fleetPublishDebounce
	fleetPublishDebounce = 20 * time.Millisecond
	t.Cleanup(func() { fleetPublishDebounce = old })
	s, fp, _, _ := fleetServer(t, config.Target{ID: "fast", Mode: "ssh", Agent: paired}, config.Target{ID: "slow", Mode: "ssh", Agent: paired})
	fp.slow["slow"] = 500 * time.Millisecond
	ch, unsub := s.fleet.subscribe()
	defer unsub()
	deadline := time.After(400 * time.Millisecond)
	for {
		select {
		case f := <-ch:
			fast, _ := f.Row("fast")
			slow, _ := f.Row("slow")
			if fast.Status != nil && slow.CheckedAt == nil {
				return // the fast box was sent while the slow one was still being probed
			}
		case <-deadline:
			t.Fatal("no frame carried the fast box before the slow one finished")
		}
	}
}

// Each section keeps its own error with its age: a status that succeeds does
// not clear a firewall failure, and a disk that succeeds has no error.
func TestFleetProbeErrorsAreKeptPerSection(t *testing.T) {
	s, fp, _, _ := fleetServer(t, config.Target{ID: "a", Mode: "ssh", Agent: paired})
	fp.fwFail = true
	t0 := time.Unix(1_800_000_000, 0)
	now := t0
	s.fleet.now = func() time.Time { return now }
	s.fleet.round(context.Background())
	now = t0.Add(15 * time.Second)
	s.fleet.round(context.Background()) // status only: disk and firewall are not due
	a, _ := s.fleet.snapshot().Row("a")
	if a.FirewallError == nil || !a.FirewallError.At.Equal(t0) || a.FirewallError.Message == "" {
		t.Fatalf("firewall error %+v", a.FirewallError)
	}
	if a.StatusError != nil || a.DiskError != nil || a.Error != nil || a.Disk == nil || a.Firewall != nil {
		t.Fatalf("row %+v status %+v disk %+v", a, a.StatusError, a.DiskError)
	}
	fp.down["a"] = true
	now = t0.Add(30 * time.Second)
	s.fleet.round(context.Background())
	a, _ = s.fleet.snapshot().Row("a")
	if a.StatusError == nil || !a.StatusError.At.Equal(now) || a.StatusError.Code != api.CodeUnreachable || a.Error == nil || a.FirewallError == nil {
		t.Fatalf("status error %+v firewall error %+v", a.StatusError, a.FirewallError)
	}
}

// An SSH-only box whose cached connection dropped is dialled again on the
// next poll, so it recovers without restarting the server.
func TestFleetSSHOnlyBoxRecoversAfterTheConnectionDrops(t *testing.T) {
	fleetHome(t)
	if _, err := config.Update(func(c *config.Config) error {
		c.Targets = []config.Target{{ID: "a", Mode: "ssh", SSH: &executor.SSHConfig{Host: "a", User: "u"}, Wire: wired}}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	opened := 0
	s := New(Config{Token: NewSessionToken(), UI: fstest.MapFS{}, NewExecutor: func(config.Target) (executor.Executor, error) {
		opened++
		if opened == 1 {
			return brokenExec{}, nil
		}
		return &autoSucceedExecutor{}, nil
	}})
	t.Cleanup(s.fleet.stop)
	s.fleet.round(context.Background())
	if row, _ := s.fleet.snapshot().Row("a"); row.Reachable {
		t.Fatalf("a dead connection read reachable: %+v", row)
	}
	s.fleet.round(context.Background())
	row, _ := s.fleet.snapshot().Row("a")
	if !row.Reachable || row.Status == nil || opened != 2 {
		t.Fatalf("after the connection came back: opened %d, row %+v", opened, row)
	}
}
