package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/valve-tech/jumpgate/internal/agentclient"
	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/catalog"
	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/intent"
	"github.com/valve-tech/jumpgate/internal/monitor"
	"github.com/valve-tech/jumpgate/internal/ops"
)

// Poller cadence (spec D12). Variables so tests can shorten them;
// fleetStatusEvery is the fallback when no refresh preference is readable. The
// per-box status sources run at agentStatusInterval (spec D31), which is
// never longer than the shortest refresh choice, so they do not follow the
// preference.
var (
	fleetStatusEvery   = 15 * time.Second
	fleetDiskEvery     = 5 * time.Minute
	fleetFirewallEvery = 10 * time.Minute
	fleetIdleStop      = 2 * time.Minute
	fleetProbeTimeout  = 20 * time.Second
	// fleetStopWait bounds how long server shutdown waits for the poller.
	fleetStopWait = 3 * time.Second
	// fleetPublishDebounce gathers the boxes that finish close together into
	// one stream frame.
	fleetPublishDebounce = 250 * time.Millisecond
)

const (
	// fleetWorkers bounds how many boxes are probed at once, so fifty boxes
	// never mean fifty SSH sessions or intents in flight.
	fleetWorkers    = 4
	diskHistorySize = 288 // a day at the 5-minute disk cadence
)

// fleetProber is how the poller reads one box. The server's own node
// operations implement it; tests use a fake.
type fleetProber interface {
	status(ctx context.Context, cfg config.Config, t config.Target) (monitor.Snapshot, error)
	disk(ctx context.Context, cfg config.Config, t config.Target) (ops.DU, error)
	firewall(ctx context.Context, cfg config.Config, t config.Target) ([]ops.CheckItem, error)
	chainID(ctx context.Context, cfg config.Config, t config.Target) (int, error)
}

// serverProber reads a box through the node operations: signed intents for
// a paired box (never root SSH, even when its agent is down), the server's
// legacy executor (legacySSHConfig's host-key policy) for an SSH-only one.
type serverProber struct{ s *Server }

// status is nodeStatus, except for an SSH-only box: nodeStatus reads that
// box's background monitor, which getMonitor starts for good (it would keep
// probing after nobody watches, against D12) and which reads a dropped
// connection as a stopped node. The fleet polls such a box once instead,
// after checking that the connection answers at all. The node status route
// still reads the monitor, so it can show "stopped" for a dead connection;
// that is left for a later fix in nodeStatus.
//
// A connection that fails the check is dropped from the registry, so the
// next poll dials the box again and a box that comes back recovers without a
// server restart. A check that ran out of time drops nothing: a slow box is
// not a dead connection.
func (p serverProber) status(ctx context.Context, cfg config.Config, t config.Target) (monitor.Snapshot, error) {
	if t.Agent != nil || t.Wire == nil {
		snap, _, err := p.s.nodeStatus(ctx, cfg, t)
		return snap, err
	}
	ex, err := p.s.legacyExec(t)
	if err != nil {
		return monitor.Snapshot{}, err
	}
	if _, err := ex.Run(ctx, "true", nil); err != nil {
		if ctx.Err() == nil {
			p.s.reg.evictExecutor(t.ID, ex)
		}
		return monitor.Snapshot{}, &dialError{fmt.Errorf("%w: %v", errConnectionLost, err)}
	}
	refRPC := ""
	if cfg.RefRPCBase != "" {
		refRPC = refRPCURL(cfg.RefRPCBase, t.Wire.ChainID)
	}
	return monitor.New(monitor.Config{Exec: ex, Wire: *t.Wire, RefRPC: refRPC}).PollOnce(ctx), nil
}

func (p serverProber) disk(ctx context.Context, cfg config.Config, t config.Target) (ops.DU, error) {
	du, _, err := p.s.nodeDisk(ctx, cfg, t)
	return du, err
}

func (p serverProber) firewall(ctx context.Context, cfg config.Config, t config.Target) ([]ops.CheckItem, error) {
	items, _, err := p.s.nodeFirewall(ctx, cfg, t)
	return items, err
}

// chainID is the controller's wire when it has one; a paired box set up by
// other means tells its own chain in agent.info.
func (p serverProber) chainID(ctx context.Context, cfg config.Config, t config.Target) (int, error) {
	if t.Wire != nil {
		return t.Wire.ChainID, nil
	}
	if t.Agent == nil {
		return 0, nil
	}
	raw, err := p.s.agentResult(ctx, cfg, t, intent.KindAgentInfo, struct{}{})
	if err != nil {
		return 0, err
	}
	var info intent.AgentInfo
	if err := json.Unmarshal(raw, &info); err != nil {
		return 0, err
	}
	return info.ChainID, nil
}

// fleetState is one box's row and what the poller remembers about it.
type fleetState struct {
	row                          api.FleetRow
	statusAt, diskAt, firewallAt time.Time // last good readings, by the poller's clock
	diskTried, firewallTried     time.Time // last attempts
	chainTried                   time.Time
	agentChainID                 int // the chain agent.info reported; 0 = not known
	history                      []api.DiskSample
	lastDU                       ops.DU // the reading the newest history sample came from
	// busy is set while a probe of this box runs, including one the poller
	// gave up waiting for, so a box that hangs is never asked twice at once.
	busy bool
}

// fleetReading is what one box's probe found, applied to its row in one go.
type fleetReading struct {
	snap      monitor.Snapshot
	statusErr error

	chainTried bool
	chainID    int

	diskTried bool
	du        ops.DU
	diskErr   error

	fwTried bool
	items   []ops.CheckItem
	fwErr   error
}

// fleetPoller keeps the fleet's rows fresh while anyone is watching: a
// stream subscriber, or a GET within the last fleetIdleStop.
type fleetPoller struct {
	s     *Server
	probe fleetProber
	now   func() time.Time
	// every is the status interval for a config: the operator's refresh
	// preference. cur is the value of the last round.
	every func(cfg config.Config) time.Duration
	// reloadCh (capacity 1) tells the loop the preference changed, so it
	// re-reads the interval and restarts its wait instead of finishing the
	// old one.
	reloadCh chan struct{}

	// ctx ends with stop; the loop and every probe run under it, and wg
	// counts them so stop can wait for all of them.
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	mu         sync.Mutex
	cur        time.Duration
	order      []string
	rows       map[string]*fleetState
	subs       map[chan api.Fleet]struct{}
	running    bool
	stopped    bool
	lastWanted time.Time
	// pubTimer is the pending debounced publish, nil when none is pending.
	pubTimer *time.Timer
	// measures are the measure-now probes in flight, one per box.
	measures map[string]*measureCall
	// statuses are the shared status sources, one per watched box.
	statuses map[string]*statusSource

	// active counts loops that have not decided to stop, and maxActive its
	// high-water mark, for the test that no two loops ever run at once.
	active, maxActive int
	// Test hooks: testIdleStopped runs after a loop decides to stop and
	// before it returns; testLoopDone as it returns.
	testIdleStopped, testLoopDone func()
}

func (p *fleetPoller) maxActiveLoops() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.maxActive
}

func newFleetPoller(s *Server) *fleetPoller {
	ctx, cancel := context.WithCancel(context.Background())
	return &fleetPoller{
		s: s, probe: serverProber{s}, now: time.Now,
		every: func(cfg config.Config) time.Duration {
			if len(cfg.UI) == 0 {
				return fleetStatusEvery // nothing chosen yet
			}
			return time.Duration(loadPrefs(cfg).RefreshSeconds) * time.Second
		},
		reloadCh: make(chan struct{}, 1),
		ctx:      ctx, cancel: cancel,
		rows: map[string]*fleetState{}, measures: map[string]*measureCall{}, statuses: map[string]*statusSource{}, subs: map[chan api.Fleet]struct{}{},
	}
}

// interval is the status interval of the last round (the default before
// the first one). Callers hold p.mu.
func (p *fleetPoller) interval() time.Duration {
	if p.cur > 0 {
		return p.cur
	}
	return fleetStatusEvery
}

// goLocked runs fn in a goroutine that stop waits for, unless the poller has
// been stopped. Callers hold p.mu, which orders every Add before stop's Wait.
func (p *fleetPoller) goLocked(fn func()) bool {
	if p.stopped {
		return false
	}
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		fn()
	}()
	return true
}

// want records interest and starts the poller if it is idle.
func (p *fleetPoller) want() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.lastWanted = time.Now()
	if p.running {
		return
	}
	p.running = p.goLocked(p.loop)
	if p.running {
		p.active++
		p.maxActive = max(p.maxActive, p.active)
	}
}

func (p *fleetPoller) isRunning() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.running
}

// stop ends the poller for good and waits for the loop and every probe it
// started, however long a probe that ignores its context takes. Tests use
// it; server shutdown uses stopWithin.
func (p *fleetPoller) stop() { p.stopWithin(0) }

// stopWithin ends the poller for good and waits at most d (no limit when d
// is 0) for its goroutines; it reports whether they all finished. The loop
// and the round end as soon as the context does. Only a probe stuck where
// its context cannot reach it (an SSH NewSession, a target's intent lock)
// can outlast the wait, and once the poller is stopped it changes nothing
// in the poller when it returns.
func (p *fleetPoller) stopWithin(d time.Duration) bool {
	p.mu.Lock()
	p.stopped = true
	if p.pubTimer != nil && p.pubTimer.Stop() {
		p.pubTimer = nil
		p.wg.Done() // the publish it would have run never will
	}
	p.mu.Unlock()
	p.cancel()
	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()
	if d <= 0 {
		<-done
		return true
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-done:
		return true
	case <-timer.C:
		return false
	}
}

// loop polls a round, sleeps one interval, and ends once nobody has watched
// for fleetIdleStop. Lifecycle time is the real clock, not p.now, which only
// dates readings. running is cleared in the same critical section that
// decides to stop, never later: a want() that arrives after the decision
// starts a new loop, and this one's exit must not then mark that one idle.
func (p *fleetPoller) loop() {
	if p.testLoopDone != nil {
		defer p.testLoopDone()
	}
	for {
		p.round(p.ctx)
		if !p.sleep() {
			p.mu.Lock()
			p.running = false
			p.active--
			p.mu.Unlock()
			return
		}
		p.mu.Lock()
		idle := len(p.subs) == 0 && time.Since(p.lastWanted) > fleetIdleStop
		if idle {
			p.running = false
			p.active--
		}
		p.mu.Unlock()
		if idle {
			if p.testIdleStopped != nil {
				p.testIdleStopped()
			}
			return
		}
	}
}

// sleep waits one interval and reports false when the poller was stopped. A
// preference change (reload) ends the wait early only to re-read the
// interval: the old timer is stopped before the new one starts, so changing
// the preference never leaves a ticker behind.
func (p *fleetPoller) sleep() bool {
	for {
		p.mu.Lock()
		every := p.interval()
		p.mu.Unlock()
		timer := time.NewTimer(every)
		select {
		case <-p.ctx.Done():
			timer.Stop()
			return false
		case <-p.reloadCh:
			timer.Stop()
			if cfg, err := p.s.loadConfig(); err == nil {
				p.mu.Lock()
				p.cur = p.every(cfg)
				p.mu.Unlock()
			}
		case <-timer.C:
			return true
		}
	}
}

// reload tells the loop (if one is running) that the preference changed.
func (p *fleetPoller) reload() {
	select {
	case p.reloadCh <- struct{}{}:
	default:
	}
}

// seed brings the rows in line with cfg: one per target in config order,
// with what the config itself says (link, agent, a wired box's chain). A row
// that has never been polled has no values, no error and no CheckedAt, which
// front ends show as loading.
func (p *fleetPoller) seed(cfg config.Config) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.seedLocked(cfg)
}

func (p *fleetPoller) seedLocked(cfg config.Config) {
	p.order = p.order[:0]
	keep := map[string]bool{}
	for _, t := range cfg.Targets {
		p.order = append(p.order, t.ID)
		keep[t.ID] = true
		st := p.rows[t.ID]
		if st == nil {
			st = &fleetState{}
			p.rows[t.ID] = st
		}
		v := targetView(t)
		st.row.ID, st.row.Link, st.row.ThisMachine = t.ID, v.Link, v.ThisMachine
		st.row.Agent = ""
		if t.Agent != nil {
			st.row.Agent = t.Agent.Address
		}
		chain := 0
		switch {
		case t.Wire != nil:
			chain = t.Wire.ChainID
		case t.Agent != nil:
			chain = st.agentChainID
		}
		st.row.ChainID, st.row.Network = chain, ""
		if n, ok := catalog.NetworkByChainID(chain); ok && chain != 0 {
			st.row.Network = n.Name
		}
	}
	for id := range p.rows {
		if !keep[id] {
			delete(p.rows, id)
		}
	}
}

// round probes every box once (status always; disk and firewall when due),
// at most fleetWorkers at a time and each within fleetProbeTimeout, and
// publishes the result.
func (p *fleetPoller) round(ctx context.Context) {
	cfg, err := p.s.loadConfig()
	if err != nil {
		return // the rows keep their last values, which age into stale
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	defer context.AfterFunc(p.ctx, cancel)()

	p.mu.Lock()
	p.cur = p.every(cfg)
	timeout := fleetProbeTimeout
	p.seedLocked(cfg)
	p.mu.Unlock()

	sem := make(chan struct{}, fleetWorkers)
	var wg sync.WaitGroup
fanOut:
	for _, t := range cfg.Targets {
		select {
		case sem <- struct{}{}:
		case <-ctx.Done():
			break fanOut
		}
		wg.Add(1)
		go func(t config.Target) {
			defer wg.Done()
			defer func() { <-sem }()
			p.pollOne(ctx, cfg, t, timeout)
		}(t)
	}
	wg.Wait()
	p.mu.Lock()
	p.notifyLocked()
	p.mu.Unlock()
}

// pollOne probes one box and applies what it found. The probe runs in its
// own goroutine so the worker can give up at the deadline even when the
// probe cannot (an intent waiting on the target's intent lock, an SSH dial
// that ignores its context): the box then reads unreachable, and is not
// probed again until that probe returns.
func (p *fleetPoller) pollOne(ctx context.Context, cfg config.Config, t config.Target, timeout time.Duration) {
	now := p.now()
	p.mu.Lock()
	st := p.rows[t.ID]
	if st == nil || st.busy {
		p.mu.Unlock()
		return
	}
	if t.Agent == nil && t.Wire == nil {
		e := api.Error{Message: "this box is not set up and not paired", Code: api.CodeTargetNotSetUp, Hint: api.HintFor(api.CodeTargetNotSetUp)}
		st.setStatusError(e, now)
		st.row.Reachable, st.row.CheckedAt = false, &now
		p.notifyLocked()
		p.mu.Unlock()
		return
	}
	needChain := t.Wire == nil && st.agentChainID == 0 && now.Sub(st.chainTried) >= fleetDiskEvery
	diskDue := now.Sub(st.diskTried) >= fleetDiskEvery
	fwDue := now.Sub(st.firewallTried) >= fleetFirewallEvery

	probeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	done := make(chan fleetReading, 1)
	st.busy = p.goLocked(func() {
		r := p.read(probeCtx, cfg, t, needChain, diskDue, fwDue)
		done <- r
		p.mu.Lock()
		if !p.stopped {
			st.busy = false
		}
		p.mu.Unlock()
	})
	started := st.busy
	p.mu.Unlock()
	if !started {
		return
	}

	select {
	case r := <-done:
		p.apply(st, now, r)
	case <-probeCtx.Done():
		if ctx.Err() != nil {
			return // the round was cancelled: nothing was learned about the box
		}
		_, e := apiErrorFor(fmt.Errorf("%w: the box did not answer within %s", agentclient.ErrUnreachable, timeout))
		p.mu.Lock()
		st.setStatusError(e, now)
		st.row.Reachable, st.row.CheckedAt = false, &now
		p.notifyLocked()
		p.mu.Unlock()
	}
}

// read runs one box's probes: status first, and the rest only when the box
// answered it.
func (p *fleetPoller) read(ctx context.Context, cfg config.Config, t config.Target, needChain, diskDue, fwDue bool) fleetReading {
	var r fleetReading
	r.snap, r.statusErr = p.probe.status(ctx, cfg, t)
	if r.statusErr != nil {
		return r
	}
	if needChain {
		r.chainTried = true
		r.chainID, _ = p.probe.chainID(ctx, cfg, t)
	}
	if diskDue {
		r.diskTried = true
		r.du, r.diskErr = p.probe.disk(ctx, cfg, t)
	}
	if fwDue {
		r.fwTried = true
		r.items, r.fwErr = p.probe.firewall(ctx, cfg, t)
	}
	return r
}

// boxAnswered says an error is still an answer from the box itself (a signed
// rejection or failure), so the box was reachable even though the probe
// failed. Every other error means the poller did not hear from it.
func boxAnswered(c api.Code) bool {
	return c == api.CodeRejected || c == api.CodeAgentFailed
}

// apply records a probe's findings. A failure never clears a value: the
// last reading stays, with its own age, and goes stale on its own clock.
func (p *fleetPoller) apply(st *fleetState, now time.Time, r fleetReading) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.stopped {
		return
	}
	defer p.notifyLocked()
	st.row.CheckedAt = &now
	if r.statusErr != nil {
		_, e := apiErrorFor(r.statusErr)
		st.setStatusError(e, now)
		st.row.Reachable = boxAnswered(e.Code)
		if st.row.Reachable {
			st.row.LastSeen = &now
		}
		return
	}
	view := nodeStatusView(r.snap)
	if view.At.IsZero() {
		view.At = now
	}
	st.row.Status, st.row.Error, st.row.StatusError, st.row.Reachable, st.row.LastSeen = &view, nil, nil, true, &now
	st.statusAt = now
	if r.chainTried {
		st.chainTried = now
		if r.chainID != 0 {
			st.agentChainID = r.chainID
			st.row.ChainID = r.chainID
			if n, ok := catalog.NetworkByChainID(r.chainID); ok {
				st.row.Network = n.Name
			}
		}
	}
	if r.diskTried {
		st.diskTried = now
		if r.diskErr == nil {
			p.recordDiskLocked(st, now, r.du)
			st.row.DiskError = nil
		} else {
			_, e := apiErrorFor(r.diskErr)
			st.row.DiskError = &api.ProbeError{Error: e, At: now}
		}
	}
	if r.fwTried {
		st.firewallTried = now
		if r.fwErr == nil {
			sum := firewallSummary(r.items)
			st.row.Firewall, st.firewallAt = &sum, now
			at := now
			st.row.FirewallAt, st.row.FirewallError = &at, nil
		} else {
			_, e := apiErrorFor(r.fwErr)
			st.row.FirewallError = &api.ProbeError{Error: e, At: now}
		}
	}
}

// setStatusError records a failed status probe: the row's error, which says
// why the box could not be read, is the status probe's.
func (st *fleetState) setStatusError(e api.Error, at time.Time) {
	st.row.Error = &e
	st.row.StatusError = &api.ProbeError{Error: e, At: at}
}

// recordDisk adds a reading to the box's history and refreshes its view.
// The disk measure route (Task 10) calls it too.
func (p *fleetPoller) recordDisk(id string, at time.Time, du ops.DU) {
	p.mu.Lock()
	defer p.mu.Unlock()
	st := p.rows[id]
	if st == nil {
		st = &fleetState{row: api.FleetRow{ID: id}}
		p.rows[id] = st
	}
	p.recordDiskLocked(st, at, du)
}

func (p *fleetPoller) recordDiskLocked(st *fleetState, at time.Time, du ops.DU) {
	st.history = append(st.history, api.DiskSample{At: at, UsedBytes: du.ExecBytes + du.BeaconBytes, FreeBytes: du.DiskFreeBytes})
	if over := len(st.history) - diskHistorySize; over > 0 {
		st.history = append(st.history[:0], st.history[over:]...)
	}
	view := diskView(at, du, st.history)
	view.History = nil // rows stay light; /api/fleet/{id}/disk carries it
	st.row.Disk, st.diskAt, st.lastDU = &view, at, du
}

func (p *fleetPoller) history(id string) []api.DiskSample {
	p.mu.Lock()
	defer p.mu.Unlock()
	if st := p.rows[id]; st != nil {
		return append([]api.DiskSample(nil), st.history...)
	}
	return nil
}

// snapshot is the fleet as of now, in config order, with stale flags. It
// never waits for a probe: a box being probed shows what is known so far.
func (p *fleetPoller) snapshot() api.Fleet {
	p.mu.Lock()
	defer p.mu.Unlock()
	now := p.now()
	every := p.interval()
	f := api.Fleet{At: now, IntervalSeconds: int(every / time.Second), Rows: []api.FleetRow{}}
	for _, id := range p.order {
		st := p.rows[id]
		if st == nil {
			continue
		}
		row := st.row
		row.Stale = api.Stale{
			Status:   row.Status != nil && now.Sub(st.statusAt) > 3*every,
			Disk:     row.Disk != nil && now.Sub(st.diskAt) > 3*fleetDiskEvery,
			Firewall: row.Firewall != nil && now.Sub(st.firewallAt) > 3*fleetFirewallEvery,
		}
		f.Rows = append(f.Rows, row)
	}
	return f
}

// notifyLocked schedules a publish fleetPublishDebounce from now, unless one
// is already pending, so boxes reach the stream as they finish while those
// that finish together share a frame. Callers hold p.mu.
func (p *fleetPoller) notifyLocked() {
	if p.stopped || p.pubTimer != nil || len(p.subs) == 0 {
		return
	}
	p.wg.Add(1)
	p.pubTimer = time.AfterFunc(fleetPublishDebounce, func() {
		defer p.wg.Done()
		p.mu.Lock()
		p.pubTimer = nil
		p.mu.Unlock()
		p.publish()
	})
}

// publish sends the fleet to every subscriber, replacing any snapshot it has
// not read yet: only the latest matters. Only publish sends, under p.mu, so
// the send after the drain never blocks.
func (p *fleetPoller) publish() {
	f := p.snapshot()
	p.mu.Lock()
	defer p.mu.Unlock()
	for ch := range p.subs {
		select {
		case <-ch:
		default:
		}
		ch <- f
	}
}

func (p *fleetPoller) subscribe() (<-chan api.Fleet, func()) {
	ch := make(chan api.Fleet, 1)
	p.mu.Lock()
	p.subs[ch] = struct{}{}
	p.mu.Unlock()
	p.want()
	return ch, func() {
		p.mu.Lock()
		delete(p.subs, ch)
		p.lastWanted = time.Now()
		p.mu.Unlock()
	}
}

func (s *Server) handleFleet(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.loadConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.fleet.seed(cfg)
	s.fleet.want()
	writeJSON(w, http.StatusOK, s.fleet.snapshot())
}

// handleFleetStream sends the fleet on connect and after every poll round,
// and keeps the poller running while it is open.
func (s *Server) handleFleetStream(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.loadConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.fleet.seed(cfg)
	conn, ok := startSSE(w)
	if !ok {
		return
	}
	defer conn.Close()
	ch, unsub := s.fleet.subscribe()
	defer unsub()
	conn.Send(s.fleet.snapshot())
	for {
		select {
		case <-r.Context().Done():
			return
		case <-s.fleet.ctx.Done():
			return // the poller was stopped: nothing more will come
		case <-conn.Pings():
			conn.Ping()
		case f := <-ch:
			conn.Send(f)
		}
	}
}

// measureCall is one measure-now probe in flight; every caller that arrives
// while it runs waits on it.
type measureCall struct {
	done    chan struct{}
	waiters int // guarded by the poller's mu
	du      ops.DU
	err     error
}

// measureWaiters is how many callers wait on id's measure in flight.
func (p *fleetPoller) measureWaiters(id string) int {
	p.mu.Lock()
	defer p.mu.Unlock()
	if c := p.measures[id]; c != nil {
		return c.waiters
	}
	return 0
}

// measure takes a disk reading of the box now, through the same prober (so
// the same executor lease or signed intent) as the poll, and records it. A
// measure that arrives while another for the box runs joins it: one probe,
// one history sample, the same answer for both. The probe runs under the
// poller's context and fleetProbeTimeout, not the first caller's, so a caller
// that gives up never fails the others.
func (p *fleetPoller) measure(ctx context.Context, cfg config.Config, t config.Target) error {
	p.mu.Lock()
	c := p.measures[t.ID]
	if c == nil {
		c = &measureCall{done: make(chan struct{})}
		p.measures[t.ID] = c
		timeout := fleetProbeTimeout
		started := p.goLocked(func() {
			defer close(c.done)
			pctx, cancel := context.WithTimeout(p.ctx, timeout)
			defer cancel()
			defer func() {
				p.mu.Lock()
				delete(p.measures, t.ID)
				p.mu.Unlock()
			}()
			du, err := p.probe.disk(pctx, cfg, t)
			if err != nil && errors.Is(pctx.Err(), context.DeadlineExceeded) {
				err = fmt.Errorf("%w: the box did not answer within %s", agentclient.ErrUnreachable, timeout)
			}
			if err == nil {
				p.recordDisk(t.ID, p.now(), du)
			}
			c.du, c.err = du, err
		})
		if !started {
			delete(p.measures, t.ID)
			p.mu.Unlock()
			return &dialError{fmt.Errorf("%w: jumpgate is shutting down", agentclient.ErrUnreachable)}
		}
	}
	c.waiters++
	p.mu.Unlock()
	select {
	case <-c.done:
		return c.err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// diskViewFor is a box's last disk reading with its full history.
func (p *fleetPoller) diskViewFor(id string) (api.DiskView, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	st := p.rows[id]
	if st == nil || len(st.history) == 0 {
		return api.DiskView{}, false
	}
	return diskView(st.diskAt, st.lastDU, append([]api.DiskSample(nil), st.history...)), true
}

func viaOf(t config.Target) via {
	if t.Agent != nil {
		return viaAgent
	}
	return viaSSH
}

// handleFleetRow is one box's fleet row. The rows are seeded from the config
// first, so a box the poller has not reached yet answers with every section
// unavailable.
func (s *Server) handleFleetRow(w http.ResponseWriter, r *http.Request) {
	cfg, t, ok := s.nodeTarget(w, r.PathValue("id"))
	if !ok {
		return
	}
	s.fleet.seed(cfg)
	s.fleet.want()
	row, found := s.fleet.snapshot().Row(t.ID)
	if !found {
		v := targetView(t)
		row = api.FleetRow{ID: t.ID, Link: v.Link, ThisMachine: v.ThisMachine}
	}
	writeJSON(w, http.StatusOK, row)
}

// measureDisk takes a disk reading now and answers with the updated view.
func (s *Server) measureDisk(w http.ResponseWriter, r *http.Request, cfg config.Config, t config.Target) {
	setVia(w, viaOf(t))
	if err := s.fleet.measure(r.Context(), cfg, t); err != nil {
		writeNodeError(w, err)
		return
	}
	v, _ := s.fleet.diskViewFor(t.ID)
	writeJSON(w, http.StatusOK, v)
}

// handleFleetDisk is the last reading with its history; with none yet it
// measures, so the first look at a box is never empty.
func (s *Server) handleFleetDisk(w http.ResponseWriter, r *http.Request) {
	cfg, t, ok := s.nodeTarget(w, r.PathValue("id"))
	if !ok {
		return
	}
	if v, found := s.fleet.diskViewFor(t.ID); found {
		setVia(w, viaOf(t))
		writeJSON(w, http.StatusOK, v)
		return
	}
	s.measureDisk(w, r, cfg, t)
}

func (s *Server) handleFleetMeasure(w http.ResponseWriter, r *http.Request) {
	cfg, t, ok := s.nodeTarget(w, r.PathValue("id"))
	if !ok {
		return
	}
	s.measureDisk(w, r, cfg, t)
}

// Status streams share one source per box (spec D31): however many windows
// watch a box, it is probed once per agentStatusInterval.
const maxStatusStreams = 8

// statusIdleStop is how long a source outlives its last subscriber, so a
// window that reconnects at once does not restart it. A variable for tests.
var statusIdleStop = 3 * time.Second

var errTooManyStreams = errors.New("too many status streams on this box")

// statusFrame is one probe's outcome: a status, or why there is none.
type statusFrame struct {
	status *api.NodeStatus
	err    *api.Error
}

// statusSource is the one prober of a box's status stream. Its fields are
// guarded by the poller's mu.
type statusSource struct {
	id        string
	cancel    context.CancelFunc
	subs      map[chan statusFrame]struct{}
	latest    *statusFrame
	idle      *time.Timer
	closed    bool
	probeBusy bool
}

// subscribeStatus joins t's status source, starting it on the first
// subscriber. The channel holds the latest frame only (a slow reader skips
// frames, never slows the probe) and is closed when the source ends: idle,
// fleet shutdown, target removal or executor eviction. It returns
// errTooManyStreams past maxStatusStreams subscribers.
func (p *fleetPoller) subscribeStatus(t config.Target) (<-chan statusFrame, func(), error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.stopped {
		return nil, nil, &dialError{fmt.Errorf("%w: jumpgate is shutting down", agentclient.ErrUnreachable)}
	}
	src := p.statuses[t.ID]
	if src != nil && len(src.subs) >= maxStatusStreams {
		return nil, nil, errTooManyStreams
	}
	if src == nil {
		ctx, cancel := context.WithCancel(p.ctx)
		src = &statusSource{id: t.ID, cancel: cancel, subs: map[chan statusFrame]struct{}{}}
		if !p.goLocked(func() { p.runStatus(ctx, src, agentStatusInterval) }) {
			cancel()
			return nil, nil, &dialError{fmt.Errorf("%w: jumpgate is shutting down", agentclient.ErrUnreachable)}
		}
		p.statuses[t.ID] = src
	}
	if src.idle != nil {
		src.idle.Stop()
		src.idle = nil
	}
	ch := make(chan statusFrame, 1)
	if src.latest != nil {
		ch <- *src.latest
	}
	src.subs[ch] = struct{}{}
	return ch, func() { p.unsubscribeStatus(src, ch) }, nil
}

func (p *fleetPoller) unsubscribeStatus(src *statusSource, ch chan statusFrame) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(src.subs, ch)
	if len(src.subs) > 0 || src.closed || src.idle != nil {
		return
	}
	src.idle = time.AfterFunc(statusIdleStop, func() {
		p.mu.Lock()
		defer p.mu.Unlock()
		if len(src.subs) == 0 && !src.closed {
			p.retireStatusLocked(src)
		}
	})
}

// retireStatus ends id's status source (if any): its subscribers' streams
// end, and a viewer that reconnects gets a new source.
func (p *fleetPoller) retireStatus(id string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if src := p.statuses[id]; src != nil {
		p.retireStatusLocked(src)
	}
}

func (p *fleetPoller) retireStatusLocked(src *statusSource) {
	if src.closed {
		return
	}
	src.closed = true
	if src.idle != nil {
		src.idle.Stop()
		src.idle = nil
	}
	if p.statuses[src.id] == src {
		delete(p.statuses, src.id)
	}
	for ch := range src.subs {
		close(ch)
	}
	src.subs = map[chan statusFrame]struct{}{}
	src.cancel()
}

func (p *fleetPoller) statusSubscribers(id string) int {
	p.mu.Lock()
	defer p.mu.Unlock()
	if src := p.statuses[id]; src != nil {
		return len(src.subs)
	}
	return 0
}

func (p *fleetPoller) statusSources() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.statuses)
}

// runStatus probes the box once per interval until ctx ends, and fans each
// answer out. A probe still running from an earlier tick (one that ignores
// its context) is not stacked on.
func (p *fleetPoller) runStatus(ctx context.Context, src *statusSource, every time.Duration) {
	defer func() {
		p.mu.Lock()
		p.retireStatusLocked(src)
		p.mu.Unlock()
	}()
	tick := time.NewTicker(every)
	defer tick.Stop()
	for {
		if !p.probeStatus(ctx, src) {
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
		}
	}
}

// probeStatus runs one probe and publishes it; false means the source should
// end (the target is gone).
func (p *fleetPoller) probeStatus(ctx context.Context, src *statusSource) bool {
	cfg, err := p.s.loadConfig()
	if err != nil {
		return ctx.Err() == nil // keep the last frame; try again next tick
	}
	t, ok := findTarget(cfg, src.id)
	if !ok {
		return false
	}
	p.mu.Lock()
	if src.probeBusy {
		p.mu.Unlock()
		return true
	}
	src.probeBusy = true
	timeout := fleetProbeTimeout
	pctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	done := make(chan statusFrame, 1)
	started := p.goLocked(func() {
		snap, err := p.probe.status(pctx, cfg, t)
		var f statusFrame
		if err != nil {
			_, e := apiErrorFor(err)
			f.err = &e
		} else {
			v := nodeStatusView(snap)
			f.status = &v
		}
		done <- f
		p.mu.Lock()
		src.probeBusy = false
		p.mu.Unlock()
	})
	if !started {
		src.probeBusy = false
	}
	p.mu.Unlock()
	if !started {
		return false
	}
	var f statusFrame
	select {
	case f = <-done:
	case <-pctx.Done():
		if ctx.Err() != nil {
			return false
		}
		_, e := apiErrorFor(fmt.Errorf("%w: the box did not answer within %s", agentclient.ErrUnreachable, timeout))
		f.err = &e
	}
	if ctx.Err() != nil {
		return false
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if src.closed {
		return false
	}
	src.latest = &f
	for ch := range src.subs {
		select {
		case <-ch:
		default:
		}
		ch <- f
	}
	return true
}

// handleFleetStatusStream sends one box's NodeStatus every
// agentStatusInterval (spec D31), for paired and SSH-only boxes alike, from
// the box's shared status source.
func (s *Server) handleFleetStatusStream(w http.ResponseWriter, r *http.Request) {
	_, t, ok := s.nodeTarget(w, r.PathValue("id"))
	if !ok {
		return
	}
	setVia(w, viaOf(t))
	ch, unsub, err := s.fleet.subscribeStatus(t)
	if errors.Is(err, errTooManyStreams) {
		writeAPIError(w, http.StatusTooManyRequests, api.Error{Message: err.Error(), Code: api.CodeTooManyStreams})
		return
	}
	if err != nil {
		writeNodeError(w, err)
		return
	}
	defer unsub()
	conn, ok := startSSE(w)
	if !ok {
		return
	}
	defer conn.Close()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-s.fleet.ctx.Done():
			return
		case <-conn.Pings():
			conn.Ping()
		case f, open := <-ch:
			if !open {
				return // the source ended: the client reconnects to a new one
			}
			if f.err != nil {
				conn.SendNamed("error", f.err)
			} else {
				conn.Send(f.status)
			}
		}
	}
}
