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
	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/intent"
	"github.com/valve-tech/jumpgate/internal/logwatch"
)

// Follower timing (variables so tests can shorten them). A follower copies
// them, and agentPollTimeout, when it starts and reads only its copies
// (Ruling T8).
var (
	agentLogsFollowInterval = 3 * time.Second
	logFollowerLinger       = 30 * time.Second
)

const (
	logRingSize = 1000
	logsSinceN  = 500
	noteUpgrade = "snapshot mode: this agent cannot follow logs; re-pair the box to upgrade it"
)

// logEvent is one thing a follower publishes: a line, or the error that the
// last poll ended in (the follower keeps polling).
type logEvent struct {
	hit *logwatch.Hit
	err *api.Error
}

// logSub is one viewer's subscription. dropped is set, under the follower's
// mu, when a line could not be queued because the viewer fell behind.
type logSub struct {
	ch      chan logEvent
	dropped bool
}

// logFollower polls a paired box's journal with logs.since while anyone
// listens, keeps the last logRingSize lines for a viewer's backlog, and stops
// linger after the last viewer leaves. It only ever asks the box's agent: a
// paired box has no other transport.
type logFollower struct {
	interval, linger, pollTimeout time.Duration
	cancel                        context.CancelFunc
	ready                         chan struct{} // closed once the first poll has answered or run out of time
	done                          chan struct{} // closed when the follower has stopped

	mu          sync.Mutex
	ring        []logwatch.Hit
	subs        map[*logSub]struct{}
	cursor      string
	lastErr     *api.Error // the last poll's error; nil after a good poll
	isReady     bool
	unsupported bool // the agent predates logs.since
	stopped     bool
	lingerT     *time.Timer
}

func newLogFollower(cancel context.CancelFunc) *logFollower {
	return &logFollower{
		interval: agentLogsFollowInterval, linger: logFollowerLinger, pollTimeout: agentPollTimeout,
		cancel: cancel, ready: make(chan struct{}), done: make(chan struct{}),
		subs: map[*logSub]struct{}{},
	}
}

func (f *logFollower) append(h logwatch.Hit) {
	f.ring = append(f.ring, h)
	if over := len(f.ring) - logRingSize; over > 0 {
		f.ring = append(f.ring[:0], f.ring[over:]...)
	}
}

// publishLocked queues ev for every viewer. A viewer whose queue is full
// loses it rather than stalling the others, and is marked so it can resync.
func (f *logFollower) publishLocked(ev logEvent) {
	for s := range f.subs {
		select {
		case s.ch <- ev:
		default:
			s.dropped = true
		}
	}
}

// subscribe adds a viewer, cancelling a pending linger. It returns nil when
// the follower has already stopped.
func (f *logFollower) subscribe() *logSub {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.stopped {
		return nil
	}
	if f.lingerT != nil {
		f.lingerT.Stop()
		f.lingerT = nil
	}
	s := &logSub{ch: make(chan logEvent, 256)}
	f.subs[s] = struct{}{}
	return s
}

// unsubscribe removes a viewer; the last one out starts the linger.
func (f *logFollower) unsubscribe(s *logSub) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.subs, s)
	if len(f.subs) == 0 && !f.stopped && f.lingerT == nil {
		f.lingerT = time.AfterFunc(f.linger, f.stopIfIdle)
	}
}

// stopIfIdle ends the linger: the follower stops unless a viewer arrived
// while the timer was firing.
func (f *logFollower) stopIfIdle() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.lingerT = nil
	if len(f.subs) == 0 {
		f.stopLocked()
	}
}

func (f *logFollower) stop() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.stopLocked()
}

func (f *logFollower) stopLocked() {
	f.stopped = true
	f.cancel()
	if f.lingerT != nil {
		f.lingerT.Stop()
		f.lingerT = nil
	}
}

func (f *logFollower) isStopped() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.stopped
}

func (f *logFollower) isUnsupported() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.unsupported
}

func (f *logFollower) markReadyLocked() {
	if !f.isReady {
		f.isReady = true
		close(f.ready)
	}
}

// start is where a viewer begins reading: the newest backlog lines and the
// error the last poll ended in, with everything queued so far dropped, all
// under one lock, so no line is both in the history and queued.
func (f *logFollower) start(s *logSub, backlog int) ([]logwatch.Hit, *api.Error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.drainLocked(s)
	return f.tailLocked(backlog), f.lastErr
}

// resync reports whether s lost lines; if so it empties s's queue and returns
// the newest backlog lines for a fresh reset (as logwatch.Watcher.Resync).
func (f *logFollower) resync(s *logSub, backlog int) ([]logwatch.Hit, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if !s.dropped {
		return nil, false
	}
	f.drainLocked(s)
	return f.tailLocked(backlog), true
}

func (f *logFollower) drainLocked(s *logSub) {
	s.dropped = false
	for {
		select {
		case <-s.ch:
		default:
			return
		}
	}
}

// tailLocked is the newest n lines, never nil, so an empty reset is "[]".
func (f *logFollower) tailLocked(n int) []logwatch.Hit {
	start := max(0, len(f.ring)-n)
	return append([]logwatch.Hit{}, f.ring[start:]...)
}

// logFollowerFor returns t's follower, starting one when there is none or
// the one on record has stopped.
func (s *Server) logFollowerFor(cfg config.Config, t config.Target) *logFollower {
	entry := s.reg.get(t.ID)
	entry.mu.Lock()
	defer entry.mu.Unlock()
	if f := entry.logs; f != nil && !f.isStopped() {
		return f
	}
	ctx, cancel := context.WithCancel(context.Background())
	f := newLogFollower(cancel)
	entry.logs = f
	go func() {
		defer close(f.done)
		defer func() {
			f.stop()
			entry.mu.Lock()
			if entry.logs == f {
				entry.logs = nil
			}
			entry.mu.Unlock()
		}()
		s.followLogs(ctx, cfg, t, f)
	}()
	return f
}

// stopLogFollowers stops every follower and waits for each to finish.
func (s *Server) stopLogFollowers() {
	s.reg.mu.Lock()
	entries := make([]*targetEntry, 0, len(s.reg.entries))
	for _, e := range s.reg.entries {
		entries = append(entries, e)
	}
	s.reg.mu.Unlock()
	for _, e := range entries {
		e.mu.Lock()
		f := e.logs
		e.mu.Unlock()
		if f != nil {
			f.stop()
			<-f.done
		}
	}
}

// followLogs polls until the follower is stopped, the agent turns out to
// predate logs.since, or the target is no longer a paired box. The target is
// read again before each poll, so a re-paired box is asked at its new
// address, and an unpaired or removed one is not asked at all: its viewers'
// streams end and reconnect to whatever transport it has now.
func (s *Server) followLogs(ctx context.Context, cfg config.Config, t config.Target, f *logFollower) {
	tick := time.NewTicker(f.interval)
	defer tick.Stop()
	for first := true; ; first = false {
		if !first {
			if c, err := s.loadConfig(); err == nil {
				nt, ok := findTarget(c, t.ID)
				if !ok || nt.Agent == nil {
					return
				}
				cfg, t = c, nt
			}
		}
		f.mu.Lock()
		cursor := f.cursor
		f.mu.Unlock()
		res, stopping, err := f.poll(ctx, func(pctx context.Context) (json.RawMessage, error) {
			return s.agentResult(pctx, cfg, t, intent.KindLogsSince, intent.LogsSincePayload{Cursor: cursor, N: logsSinceN})
		})
		if stopping {
			return
		}
		if f.apply(res, err) {
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
		}
	}
}

// logsSinceResult is the agent's logs.since answer (agent.LogsSinceResult).
type logsSinceResult struct {
	Hits   []logwatch.Hit `json:"hits"`
	Cursor string         `json:"cursor"`
}

// errPollReported marks a poll whose timeout was already published.
var errPollReported = errors.New("poll timeout already reported")

// poll runs one logs.since under pollTimeout. When the deadline passes first,
// the timeout is published at once (and a viewer waiting for the first poll
// may start), and whatever the poll answers later is dropped: the next poll
// asks again from the same cursor. stopping reports that the follower was
// stopped meanwhile. poll returns only after send has.
func (f *logFollower) poll(ctx context.Context, send func(context.Context) (json.RawMessage, error)) (logsSinceResult, bool, error) {
	pctx, cancel := context.WithTimeout(ctx, f.pollTimeout)
	defer cancel()
	type answer struct {
		res logsSinceResult
		err error
	}
	answers := make(chan answer, 1)
	go func() {
		res, err := decodeResult[logsSinceResult](send(pctx))
		answers <- answer{res, err}
	}()
	timeout := fmt.Errorf("%w: the agent did not answer within %s", agentclient.ErrUnreachable, f.pollTimeout)
	select {
	case a := <-answers:
		if ctx.Err() != nil {
			return a.res, true, a.err
		}
		if a.err != nil && errors.Is(pctx.Err(), context.DeadlineExceeded) {
			a.err = timeout
		}
		return a.res, false, a.err
	case <-pctx.Done():
	}
	if ctx.Err() != nil {
		<-answers
		return logsSinceResult{}, true, nil
	}
	f.mu.Lock()
	f.failLocked(timeout)
	f.markReadyLocked()
	f.mu.Unlock()
	<-answers
	return logsSinceResult{}, ctx.Err() != nil, errPollReported
}

// apply records one poll's outcome and publishes it. It reports true when
// the agent predates logs.since, which ends the follower.
func (f *logFollower) apply(res logsSinceResult, err error) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	defer f.markReadyLocked()
	var rej *agentRejected
	switch {
	case errors.Is(err, errPollReported):
	case errors.As(err, &rej) && rej.Code == intent.ReasonUnknownKind:
		f.unsupported = true
		return true
	case err != nil:
		f.failLocked(err)
	default:
		f.lastErr = nil
		for i := range res.Hits {
			f.append(res.Hits[i])
			f.publishLocked(logEvent{hit: &res.Hits[i]})
		}
		if res.Cursor != "" {
			f.cursor = res.Cursor
		}
	}
	return false
}

func (f *logFollower) failLocked(err error) {
	_, e := apiErrorFor(err)
	f.lastErr = &e
	f.publishLocked(logEvent{err: &e})
}

// streamAgentLogs serves a paired box's logs stream from its follower, or
// from snapshots when the agent predates logs.since. rawBacklog is the
// ?backlog= value, with Task 3's meaning: absent, the stream sends only new
// lines, as default events (the web UI's shape: it fetches the history
// itself); present, it opens on a "reset" of the newest backlog lines (an
// empty one for 0) and sends a fresh reset whenever this viewer lost lines.
func (s *Server) streamAgentLogs(w http.ResponseWriter, r *http.Request, cfg config.Config, t config.Target, rawBacklog string) {
	resets := rawBacklog != ""
	backlog := backlogParam(rawBacklog)
	setVia(w, viaAgent)
	conn, ok := startSSE(w)
	if !ok {
		return
	}
	defer conn.Close()

	var f *logFollower
	var sub *logSub
	for sub == nil {
		f = s.logFollowerFor(cfg, t)
		sub = f.subscribe()
	}
	defer f.unsubscribe(sub)

	// The viewer is counted from here, so leaving while the first poll is
	// still out starts the linger like any other departure.
	for waiting := true; waiting; {
		select {
		case <-r.Context().Done():
			return
		case <-f.ready:
			waiting = false
		case <-f.done:
			waiting = false
		case <-conn.Pings():
			conn.Ping()
		}
	}
	if f.isUnsupported() {
		s.streamAgentLogSnapshots(r, conn, cfg, t, rawBacklog, noteUpgrade)
		return
	}
	hist, lastErr := f.start(sub, backlog)
	if resets {
		conn.SendNamed("reset", hist)
	}
	if lastErr != nil {
		conn.SendNamed("error", lastErr)
	}
	for {
		select {
		case <-r.Context().Done():
			return
		case <-f.done:
			if f.isUnsupported() {
				s.streamAgentLogSnapshots(r, conn, cfg, t, rawBacklog, noteUpgrade)
			}
			return
		case <-conn.Pings():
			if resets {
				if hist, dropped := f.resync(sub, backlog); dropped {
					conn.SendNamed("reset", hist)
				}
			}
			conn.Ping()
		case ev := <-sub.ch:
			if resets {
				if hist, dropped := f.resync(sub, backlog); dropped {
					conn.SendNamed("reset", hist)
					continue
				}
			}
			if ev.err != nil {
				conn.SendNamed("error", ev.err)
			} else {
				conn.Send(ev.hit)
			}
		}
	}
}
