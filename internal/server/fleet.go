package server

import (
	"context"
	"encoding/json"
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

// Poller cadence (spec D12). Variables so tests can shorten them; Task 12
// lets the operator's refresh preference override fleetStatusEvery. The
// poller reads them under its lock, once per round.
var (
	fleetStatusEvery   = 15 * time.Second
	fleetDiskEvery     = 5 * time.Minute
	fleetFirewallEvery = 10 * time.Minute
	fleetIdleStop      = 2 * time.Minute
	fleetProbeTimeout  = 20 * time.Second
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
// after checking that the connection answers at all.
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
		return monitor.Snapshot{}, fmt.Errorf("%w: %v", agentclient.ErrUnreachable, err)
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
	// every is the status interval for a config; Task 12 makes it the
	// operator's refresh preference. cur is the value of the last round.
	every func(cfg config.Config) time.Duration

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
}

func newFleetPoller(s *Server) *fleetPoller {
	ctx, cancel := context.WithCancel(context.Background())
	return &fleetPoller{
		s: s, probe: serverProber{s}, now: time.Now,
		every: func(config.Config) time.Duration { return fleetStatusEvery },
		ctx:   ctx, cancel: cancel,
		rows: map[string]*fleetState{}, subs: map[chan api.Fleet]struct{}{},
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
}

func (p *fleetPoller) isRunning() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.running
}

// stop ends the poller for good and waits for the loop and every probe it
// started. Probes see their context end; one that ignores it is waited for.
func (p *fleetPoller) stop() {
	p.mu.Lock()
	p.stopped = true
	p.mu.Unlock()
	p.cancel()
	p.wg.Wait()
}

// loop polls a round, sleeps one interval, and ends once nobody has watched
// for fleetIdleStop. Lifecycle time is the real clock, not p.now, which only
// dates readings.
func (p *fleetPoller) loop() {
	defer func() {
		p.mu.Lock()
		p.running = false
		p.mu.Unlock()
	}()
	for {
		p.round(p.ctx)
		p.mu.Lock()
		every := p.interval()
		p.mu.Unlock()
		timer := time.NewTimer(every)
		select {
		case <-p.ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
		p.mu.Lock()
		idle := len(p.subs) == 0 && time.Since(p.lastWanted) > fleetIdleStop
		if idle {
			// Cleared here, under the same lock as the check, so a want()
			// that arrives now starts a new loop instead of trusting this one.
			p.running = false
		}
		p.mu.Unlock()
		if idle {
			return
		}
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
	p.publish()
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
		st.row.Error, st.row.Reachable, st.row.CheckedAt = &e, false, &now
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
		st.busy = false
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
		st.row.Error, st.row.Reachable, st.row.CheckedAt = &e, false, &now
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
	st.row.CheckedAt = &now
	if r.statusErr != nil {
		_, e := apiErrorFor(r.statusErr)
		st.row.Error, st.row.Reachable = &e, boxAnswered(e.Code)
		if st.row.Reachable {
			st.row.LastSeen = &now
		}
		return
	}
	view := nodeStatusView(r.snap)
	if view.At.IsZero() {
		view.At = now
	}
	st.row.Status, st.row.Error, st.row.Reachable, st.row.LastSeen = &view, nil, true, &now
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
		} else {
			_, e := apiErrorFor(r.diskErr)
			st.row.Error = &e
		}
	}
	if r.fwTried {
		st.firewallTried = now
		if r.fwErr == nil {
			sum := firewallSummary(r.items)
			st.row.Firewall, st.firewallAt = &sum, now
			at := now
			st.row.FirewallAt = &at
		} else {
			_, e := apiErrorFor(r.fwErr)
			st.row.Error = &e
		}
	}
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
	st.row.Disk, st.diskAt = &view, at
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
