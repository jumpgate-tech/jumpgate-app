// internal/server/jobs.go
package server

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/eip712"
	"github.com/valve-tech/jumpgate/internal/intent"
	"github.com/valve-tech/jumpgate/internal/joblog"
	"github.com/valve-tech/jumpgate/internal/jobs"
)

// Cadences (variables so tests shorten them). D-J2.
var (
	jobsListEvery   = 10 * time.Second // a refresh within this is reused
	jobsDriveEvery  = 5 * time.Second  // the step driver reads its job (jobsdrive.go)
	jobsRenewEvery  = 20 * time.Second // and renews its lease; three missed renewals is jobs.LeaseTTL
	jobsOutputEvery = 2 * time.Second
)

// errJobLogInvalid is a log that does not verify against the box's agent key,
// or that is not the log the job's own record describes.
var errJobLogInvalid = errors.New("the job's log does not verify")

type jobKey struct{ target, id string }

type boxJobs struct {
	jobs        []intent.JobRecord
	agentTime   int64
	err         *api.Error
	unsupported bool // the agent cannot run jobs (or predates them): no summary
}

// logMark is how far this server last verified a job's log: its length and
// head digest, and the agent key it verified them against. The driver reads
// on from it, verifying only what was added; a re-paired box (or another box
// re-added under the same id) never has a mark another key made, so its
// entries are never credited to a key that did not sign them. No entries are
// kept: the detail view and a decision read and verify the whole log.
type logMark struct {
	agent eip712.Address
	len   uint64
	head  [32]byte
}

// Bounds on one read of a job's log, against a box that holds its agent key
// and signs an endless valid chain (variables so tests shorten them). A job
// logs its creation, each request that changes it, and each step's start and
// outcome; lease renewals are saved, not logged (Q7). A six-hour job of a few
// steps, retried and decided a few times, is a few hundred entries, so 5000
// is over ten times anything a plan here produces. Entries carry at most an
// intent envelope or a step result, a few KiB each, so 16 MiB (one agent
// reply's limit, agentclient.maxReply) is likewise far above a real log; it
// counts the JSON encoding of every entry and job record this read received.
// The deadline bounds the whole paged read, not each page.
var (
	jobLogMaxEntries  = 5000
	jobLogMaxBytes    = 16 << 20
	jobLogReadTimeout = 60 * time.Second
)

// jobsHub is this server's view of every paired box's durable jobs: it lists
// them (lazily, at most every jobsListEvery), verifies their logs, serves the
// inbox, and drives the jobs whose lease this controller holds (jobsdrive.go).
type jobsHub struct {
	s      *Server
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// afterList runs after each box is listed, outside mu (wireDriver
	// resumes this controller's own jobs there).
	afterList func(t config.Target, jobs []intent.JobRecord)
	// readJob asks a box for one page of a job (job.read). It is a field so
	// a test can stand in for an agent that lies about its log.
	readJob func(ctx context.Context, cfg config.Config, t config.Target, p intent.JobReadPayload) (intent.JobReadResult, error)
	// listJobs asks a box for its jobs (job.list). It is a field so a test
	// can stand in for a box whose jobs cannot be listed.
	listJobs func(ctx context.Context, cfg config.Config, t config.Target) (intent.JobListResult, error)

	mu       sync.Mutex
	boxes    map[string]*boxJobs
	listedAt time.Time
	listing  chan struct{}
	subs     map[chan api.JobsInbox]struct{}
	marks    map[jobKey]logMark
	driving  map[jobKey]context.CancelFunc
	// settled are the finished jobs whose result this controller has copied
	// into its record of the box (jobsdrive.go: finished), or is copying.
	settled  map[jobKey]bool
	noDriver bool // Config.noJobsDriver: list and verify, never drive
	stopped  bool
}

func newJobsHub(s *Server) *jobsHub {
	ctx, cancel := context.WithCancel(context.Background())
	h := &jobsHub{s: s, ctx: ctx, cancel: cancel, boxes: map[string]*boxJobs{}, subs: map[chan api.JobsInbox]struct{}{},
		marks: map[jobKey]logMark{}, driving: map[jobKey]context.CancelFunc{},
		settled: map[jobKey]bool{}}
	h.readJob = func(ctx context.Context, cfg config.Config, t config.Target, p intent.JobReadPayload) (intent.JobReadResult, error) {
		return decodeResult[intent.JobReadResult](s.agentResult(ctx, cfg, t, intent.KindJobRead, p))
	}
	h.listJobs = func(ctx context.Context, cfg config.Config, t config.Target) (intent.JobListResult, error) {
		return decodeResult[intent.JobListResult](s.agentResult(ctx, cfg, t, intent.KindJobList, struct{}{}))
	}
	return h
}

func (h *jobsHub) me() (eip712.Address, bool) {
	if h.s.cfg.Signer == nil {
		return eip712.Address{}, false
	}
	return h.s.cfg.Signer.Address(), true
}

// mine reports whether this controller's key holds r's lease.
func (h *jobsHub) mine(r intent.JobRecord) bool {
	me, ok := h.me()
	owner, err := eip712.ParseAddress(r.Lease.Owner)
	return ok && err == nil && owner == me
}

// refresh lists every paired box's jobs. Callers that arrive while one runs
// wait for it; a refresh newer than jobsListEvery is reused unless force.
// The listing itself belongs to the hub, not the caller: it runs under the
// hub's context, so a caller that goes away (a closed stream, the end of a
// fleet round) stops waiting without cutting the listing short and leaving
// every box marked unreachable.
func (h *jobsHub) refresh(ctx context.Context, force bool) {
	h.mu.Lock()
	if h.stopped {
		h.mu.Unlock()
		return
	}
	wait := h.listing
	if wait == nil {
		if !force && time.Since(h.listedAt) < jobsListEvery {
			h.mu.Unlock()
			return
		}
		wait = make(chan struct{})
		h.listing = wait
		// Added under mu, before stopWithin can set stopped and Wait.
		h.wg.Add(1)
		go h.list(wait)
	}
	h.mu.Unlock()
	select {
	case <-wait:
	case <-ctx.Done():
	}
}

// list is one listing of every paired box, at most fleetWorkers at a time,
// each within fleetProbeTimeout. It closes done and publishes when finished.
func (h *jobsHub) list(done chan struct{}) {
	defer h.wg.Done()
	defer func() {
		h.mu.Lock()
		h.listing, h.listedAt = nil, time.Now()
		h.mu.Unlock()
		close(done)
		h.publish()
	}()
	cfg, err := h.s.loadConfig()
	if err != nil {
		return
	}
	sem := make(chan struct{}, fleetWorkers)
	var wg sync.WaitGroup
	keep := map[string]bool{}
	for _, t := range cfg.Targets {
		if t.Agent == nil {
			continue // SSH-only: no agent, no jobs
		}
		keep[t.ID] = true
		wg.Add(1)
		sem <- struct{}{}
		go func(t config.Target) {
			defer wg.Done()
			defer func() { <-sem }()
			pctx, cancel := context.WithTimeout(h.ctx, fleetProbeTimeout)
			defer cancel()
			h.listBox(pctx, cfg, t)
		}(t)
	}
	wg.Wait()
	h.mu.Lock()
	for id := range h.boxes {
		if !keep[id] {
			delete(h.boxes, id)
		}
	}
	for k := range h.marks {
		if !keep[k.target] {
			delete(h.marks, k) // the box was removed or unpaired
		}
	}
	h.mu.Unlock()
}

func (h *jobsHub) listBox(ctx context.Context, cfg config.Config, t config.Target) {
	res, err := h.listJobs(ctx, cfg, t)
	box := &boxJobs{jobs: res.Jobs, agentTime: res.AgentTime}
	switch {
	case runsNoJobs(err):
		box = &boxJobs{unsupported: true} // an old agent, or a box that cannot run jobs
		h.dropLogs(t.ID, nil)
	case err != nil:
		_, e := apiErrorFor(err)
		box = &boxJobs{err: &e}
		h.mu.Lock()
		if old := h.boxes[t.ID]; old != nil {
			box.jobs, box.agentTime = old.jobs, old.agentTime // keep what we knew; the error says it is old
		}
		h.mu.Unlock()
	}
	h.mu.Lock()
	h.boxes[t.ID] = box
	h.mu.Unlock()
	if err == nil {
		// A job the box no longer lists was pruned there (J6): its log
		// mark goes too, so the marks never outgrow what the boxes keep.
		listed := make(map[string]bool, len(res.Jobs))
		for _, j := range res.Jobs {
			listed[j.ID] = true
		}
		h.dropLogs(t.ID, listed)
	}
	if err == nil && h.afterList != nil {
		h.afterList(t, res.Jobs)
	}
}

// runsNoJobs reports whether a listing's error is the box saying it runs no
// jobs: an agent that predates them, or a box that cannot run them.
func runsNoJobs(err error) bool {
	var rej *agentRejected
	return errors.As(err, &rej) && (rej.Code == intent.ReasonJobsUnsupported || rej.Code == intent.ReasonUnknownKind)
}

// activeJob asks box t, fresh, for its jobs, and returns one that has not
// finished (J1). Setup and service clear run over SSH and never reach the
// agent's own J1 check, so they ask here first. A box that runs no jobs has
// none running; a listing that fails is an error, because nothing then
// proves no job runs, and the caller must refuse.
func (h *jobsHub) activeJob(ctx context.Context, cfg config.Config, t config.Target) (intent.JobRecord, bool, error) {
	res, err := h.listJobs(ctx, cfg, t)
	if runsNoJobs(err) {
		return intent.JobRecord{}, false, nil
	}
	if err != nil {
		return intent.JobRecord{}, false, err
	}
	for _, j := range res.Jobs {
		if !intent.JobTerminal(j.State) {
			return j, true, nil
		}
	}
	return intent.JobRecord{}, false, nil
}

// errJobActive is J1's refusal, in the words the agent uses for
// service.action, so every route that defers to a job says the same.
func errJobActive(j intent.JobRecord) error {
	return &agentRejected{intent.Rejection{Code: intent.ReasonJobActive,
		Message: fmt.Sprintf("job %s (%s) is %s and owns the node services until it ends", j.ID, j.Kind, j.State)}}
}

// refuseDuringJob answers for a route that changes a paired box's node and
// returns true when a job on the box has not finished, or when the box
// cannot say. An SSH-only target runs no jobs.
func (s *Server) refuseDuringJob(w http.ResponseWriter, ctx context.Context, cfg config.Config, t config.Target) bool {
	if t.Agent == nil {
		return false
	}
	j, busy, err := s.jobs.activeJob(ctx, cfg, t)
	switch {
	case err != nil:
		// The listing's own status and code, so an unreachable box still
		// reads as one; the message says what was refused.
		status, e := apiErrorFor(err)
		e.Message = fmt.Sprintf("cannot tell whether a job runs on box %s, so its node is left alone: %s", t.ID, e.Message)
		writeAPIError(w, status, e)
		return true
	case busy:
		writeNodeError(w, errJobActive(j))
		return true
	}
	return false
}

// dropLogs forgets target's log marks, except those of the jobs in keep.
func (h *jobsHub) dropLogs(target string, keep map[string]bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for k := range h.marks {
		if k.target == target && !keep[k.id] {
			delete(h.marks, k)
		}
	}
}

// listedLocked reports whether the last listing of k's box has k's job.
// Callers hold h.mu.
func (h *jobsHub) listedLocked(k jobKey) bool {
	box := h.boxes[k.target]
	if box == nil || box.unsupported {
		return false
	}
	for _, j := range box.jobs {
		if j.ID == k.id {
			return true
		}
	}
	return false
}

// mark is k's log mark if it was made against agent.
func (h *jobsHub) mark(k jobKey, agent eip712.Address) (logMark, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	m, ok := h.marks[k]
	return m, ok && m.agent == agent // another key's mark: start over
}

// storeMark keeps m as k's mark while the hub's listing has the job, so the
// marks never outgrow what the boxes list. A job not listed yet (just
// created) has its whole log read until a listing shows it.
func (h *jobsHub) storeMark(k jobKey, m logMark) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.listedLocked(k) {
		h.marks[k] = m
	}
}

// setRecord updates one job in the cache (after a read or a mutation).
func (h *jobsHub) setRecord(target string, r intent.JobRecord, agentTime int64) {
	h.mu.Lock()
	box := h.boxes[target]
	if box == nil {
		box = &boxJobs{}
		h.boxes[target] = box
	}
	box.agentTime = max(box.agentTime, agentTime)
	found := false
	for i := range box.jobs {
		if box.jobs[i].ID == r.ID {
			box.jobs[i], found = r, true
			break
		}
	}
	if !found {
		box.jobs = append([]intent.JobRecord{r}, box.jobs...)
	}
	h.mu.Unlock()
	h.publish()
}

// jobRank orders the inbox: needs-you first, then unfinished, then finished.
// The server owns this order; clients show the list as they receive it.
func jobRank(state string) int {
	switch {
	case state == intent.JobNeedsDecision:
		return 0
	case !intent.JobTerminal(state):
		return 1
	}
	return 2
}

func (h *jobsHub) inbox() api.JobsInbox {
	h.mu.Lock()
	defer h.mu.Unlock()
	in := api.JobsInbox{At: time.Now(), Jobs: []api.JobView{}, Errors: []api.JobsError{}}
	targets := make([]string, 0, len(h.boxes))
	for id := range h.boxes {
		targets = append(targets, id)
	}
	sort.Strings(targets)
	for _, id := range targets {
		box := h.boxes[id]
		if box.err != nil {
			in.Errors = append(in.Errors, api.JobsError{Target: id, Error: *box.err})
		}
		for _, r := range box.jobs {
			in.Jobs = append(in.Jobs, h.viewLocked(id, r, box.agentTime))
		}
	}
	// Newest first within a rank; the stable sort keeps target order for ties.
	sort.SliceStable(in.Jobs, func(i, j int) bool {
		a, b := in.Jobs[i], in.Jobs[j]
		if ra, rb := jobRank(a.State), jobRank(b.State); ra != rb {
			return ra < rb
		}
		return a.CreatedAt.After(b.CreatedAt)
	})
	return in
}

// summary is a fleet row's jobs: nil when the box was never listed or cannot
// run jobs, so the row shows "unavailable", never 0 (TUI spec, honest data).
func (h *jobsHub) summary(target string) *api.JobsSummary {
	h.mu.Lock()
	defer h.mu.Unlock()
	box := h.boxes[target]
	if box == nil || box.unsupported || (box.err != nil && box.jobs == nil) {
		return nil
	}
	s := &api.JobsSummary{}
	for _, r := range box.jobs {
		switch {
		case r.State == intent.JobNeedsDecision:
			s.NeedsDecision++
		case !intent.JobTerminal(r.State):
			s.Running++
		}
	}
	return s
}

func (h *jobsHub) view(target string, r intent.JobRecord, agentTime int64) api.JobView {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.viewLocked(target, r, agentTime)
}

// viewLocked copies a box's record into the API's shape. Every string in it
// came from the box and is passed through as data; clients sanitise it.
func (h *jobsHub) viewLocked(target string, r intent.JobRecord, agentTime int64) api.JobView {
	v := api.JobView{Target: target, ID: r.ID, Kind: r.Kind, Title: r.Title, State: r.State, Step: r.Step, Attempt: r.Attempt,
		Error: r.Error, CreatedAt: time.Unix(r.CreatedAt, 0).UTC(), UpdatedAt: time.Unix(r.UpdatedAt, 0).UTC(),
		Steps: make([]api.JobStepView, len(r.Steps))}
	for i, s := range r.Steps {
		state := "pending"
		switch {
		case i < r.Step || r.State == intent.JobSucceeded:
			state = "done"
		case i == r.Step && r.State == intent.JobRunning:
			state = "running"
		case i == r.Step && r.State == intent.JobNeedsDecision:
			state = "asking"
		case i == r.Step && r.State == intent.JobWaiting:
			state = "waiting"
		case i == r.Step:
			state = "stopped"
		}
		v.Steps[i] = api.JobStepView{ID: s.ID, Title: s.Title, State: state}
	}
	if r.Lease.Owner != "" {
		hb := time.Unix(r.Lease.HeartbeatAt, 0).UTC()
		ttl := int64(r.Lease.TTLSeconds)
		if ttl == 0 {
			ttl = 90
		}
		// The agent's own rule (jobs.Manifest.LeaseFresh): a heartbeat more
		// than a TTL ahead of the agent's clock is stale.
		age := agentTime - r.Lease.HeartbeatAt
		v.Lease = api.JobLease{Owner: r.Lease.Owner, HeartbeatAt: &hb, Fresh: age < ttl && age > -ttl, Mine: h.mine(r)}
		if v.Lease.Mine {
			v.Lease.Label = "this controller"
		}
	}
	if q := r.Question; q != nil {
		v.Question = &api.JobQuestion{ID: q.ID, Prompt: q.Prompt, Tier: q.Tier, AskedAt: time.Unix(q.AskedAt, 0).UTC(),
			Signable: q.Tier != intent.TierApproval} // Q5: no approval signer yet
		for _, o := range q.Options {
			v.Question.Options = append(v.Question.Options, api.JobOption{ID: o.ID, Label: o.Label, Detail: o.Detail})
		}
	}
	_, v.Driving = h.driving[jobKey{target, r.ID}]
	return v
}

// readVerified reads a job and its log, every page job.read has. With full,
// or with no mark made against this box's agent key, it reads and verifies
// the whole log and returns it; otherwise it reads on from the mark and
// verifies, and returns, only what was added. Each page must be signed by
// the box's agent and chained onto the last (Verify), and every request in
// it must be one its controller signed for this agent and this job
// (VerifyIntents). Verify proves only a valid prefix, so the log must then
// be the one the job's record describes: both its length and its head
// digest, or a box could drop the tail or hand over another log. Anything
// less is errJobLogInvalid, and the job's mark is dropped, so no part of
// that log is shown and the next read starts over.
func (h *jobsHub) readVerified(ctx context.Context, cfg config.Config, t config.Target, id string, full bool) (intent.JobRecord, []joblog.Entry, int64, error) {
	at, err := agentTarget(t)
	if err != nil {
		return intent.JobRecord{}, nil, 0, &localError{err}
	}
	ctx, cancel := context.WithTimeout(ctx, jobLogReadTimeout)
	defer cancel()
	k := jobKey{t.ID, id}
	var from logMark // from 0, chained onto the zero digest
	if m, ok := h.mark(k, at.Agent); ok && !full {
		from = m
	}
	head, n := from.head, from.len // the verified log's head and length so far
	var entries []joblog.Entry     // what this read verified
	var rec intent.JobRecord
	var agentTime int64
	var claimed uint64 // the first page's LogLen: the most this read takes
	size := 0          // this read's encoded entries and records
	tooBig := fmt.Errorf("the job's log is larger than the %d bytes this server reads", jobLogMaxBytes)
	for first := true; ; first = false {
		res, err := h.readJob(ctx, cfg, t, intent.JobReadPayload{ID: id, From: n})
		if err != nil {
			return rec, nil, 0, err
		}
		rec, agentTime = res.Job, res.AgentTime
		if first {
			claimed = rec.LogLen
			if claimed > uint64(jobLogMaxEntries) {
				return h.invalid(k, rec, fmt.Errorf("the job claims %d log entries, more than the %d this server reads", claimed, jobLogMaxEntries))
			}
		}
		if n+uint64(len(res.Entries)) > claimed {
			return h.invalid(k, rec, fmt.Errorf("the box sent more than the %d log entries the job claimed", claimed))
		}
		if rb, err := json.Marshal(res.Job); err != nil {
			return h.invalid(k, rec, err)
		} else if size += len(rb); size > jobLogMaxBytes {
			return h.invalid(k, rec, tooBig)
		}
		page := make([]joblog.Entry, 0, len(res.Entries))
		for _, j := range res.Entries {
			// agentclient does not report a reply's size: each entry is
			// measured re-encoded.
			eb, err := json.Marshal(j)
			if err != nil {
				return h.invalid(k, rec, err)
			}
			if size += len(eb); size > jobLogMaxBytes {
				return h.invalid(k, rec, tooBig)
			}
			e, err := joblog.FromJSON(j)
			if err != nil {
				return h.invalid(k, rec, err)
			}
			page = append(page, e)
		}
		if head, err = joblog.Verify(at.Agent, id, page, n, head); err != nil {
			return h.invalid(k, rec, err)
		}
		if err := joblog.VerifyIntents(at.Agent, id, page); err != nil {
			return h.invalid(k, rec, err)
		}
		// A job's log starts with the request that created it, and only
		// there: VerifyIntents binds that entry to the job id, so a log
		// without it, or with a second one, credits a request the id does
		// not commit to.
		for _, e := range page {
			if (e.Index == 0) != (e.Kind == joblog.KindCreated) {
				return h.invalid(k, rec, fmt.Errorf("entry %d is %q; a job's log has its one %s entry first", e.Index, e.Kind, joblog.KindCreated))
			}
		}
		entries = append(entries, page...)
		n += uint64(len(page))
		if !res.More {
			break
		}
		if len(page) == 0 {
			// More with nothing in it would loop for ever.
			return h.invalid(k, rec, errors.New("the box says there is more log but sent none"))
		}
	}
	if rec.ID != id {
		return h.invalid(k, rec, fmt.Errorf("asked for job %s, the box answered with job %q", id, rec.ID))
	}
	if n == 0 {
		return h.invalid(k, rec, fmt.Errorf("the job's log is empty; it must start with its %s entry", joblog.KindCreated))
	}
	// The record must describe the log just verified, or the box is lying
	// about one of them.
	got := "0x" + hex.EncodeToString(head[:])
	if n != rec.LogLen || got != recordedHead(rec) {
		return h.invalid(k, rec, fmt.Errorf("the job says its log has %d entries ending %q; %d verified, ending %s", rec.LogLen, rec.LogHead, n, got))
	}
	h.storeMark(k, logMark{agent: at.Agent, len: n, head: head})
	h.setRecord(t.ID, rec, agentTime)
	return rec, entries, agentTime, nil
}

// recordedHead is the head digest a record claims, lower-case, with a log of
// no entries spelled as the zero digest (as the agent's store does).
func recordedHead(r intent.JobRecord) string {
	if r.LogHead == "" {
		return "0x" + strings.Repeat("00", 32)
	}
	return strings.ToLower(r.LogHead)
}

func (h *jobsHub) invalid(k jobKey, rec intent.JobRecord, err error) (intent.JobRecord, []joblog.Entry, int64, error) {
	h.mu.Lock()
	delete(h.marks, k)
	h.mu.Unlock()
	return rec, nil, 0, fmt.Errorf("%w: %v", errJobLogInvalid, err)
}

func (h *jobsHub) subscribe() (chan api.JobsInbox, func()) {
	ch := make(chan api.JobsInbox, 4)
	h.mu.Lock()
	h.subs[ch] = struct{}{}
	h.mu.Unlock()
	return ch, func() {
		h.mu.Lock()
		delete(h.subs, ch)
		h.mu.Unlock()
	}
}

// publish sends the inbox to every subscriber (dropping a frame for one that
// is behind: the next frame carries everything) and tells the fleet poller,
// whose rows carry the counts. It never holds h.mu while it calls the
// poller, so the lock order stays fleet p.mu, then h.mu.
func (h *jobsHub) publish() {
	in := h.inbox()
	h.mu.Lock()
	for ch := range h.subs {
		select {
		case ch <- in:
		default:
		}
	}
	h.mu.Unlock()
	h.s.fleet.jobsChanged()
}

// stopWithin ends every driver and listing and waits at most d for them.
func (h *jobsHub) stopWithin(d time.Duration) bool {
	h.mu.Lock()
	h.stopped = true
	for _, cancel := range h.driving {
		cancel()
	}
	h.mu.Unlock()
	h.cancel()
	done := make(chan struct{})
	go func() { h.wg.Wait(); close(done) }()
	select {
	case <-done:
		return true
	case <-time.After(d):
		return false
	}
}

// Routes.

func (s *Server) handleJobs(w http.ResponseWriter, r *http.Request) {
	s.jobs.refresh(r.Context(), false)
	writeJSON(w, http.StatusOK, s.jobs.inbox())
}

func (s *Server) handleJobsStream(w http.ResponseWriter, r *http.Request) {
	conn, ok := startSSE(w)
	if !ok {
		return
	}
	defer conn.Close()
	ch, unsub := s.jobs.subscribe()
	defer unsub()
	go s.jobs.refresh(r.Context(), false)
	conn.Send(s.jobs.inbox())
	tick := time.NewTicker(max(jobsListEvery, time.Second))
	defer tick.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-s.jobs.ctx.Done():
			return
		case <-conn.Pings():
			conn.Ping()
		case <-tick.C:
			go s.jobs.refresh(r.Context(), false)
		case in := <-ch:
			conn.Send(in)
		}
	}
}

// jobTarget resolves {target} and {job} for a job route: a paired box and a
// well-formed id.
func (s *Server) jobTarget(w http.ResponseWriter, r *http.Request) (config.Config, config.Target, string, bool) {
	cfg, t, ok := s.nodeTarget(w, r.PathValue("target"))
	if !ok {
		return cfg, t, "", false
	}
	if t.Agent == nil {
		writeErrorDetail(w, http.StatusConflict, "this box has no paired agent, so it runs no jobs", "", api.CodeNotPaired)
		return cfg, t, "", false
	}
	id := r.PathValue("job")
	if id != "" && !jobs.ValidID(id) {
		writeErrorDetail(w, http.StatusNotFound, "no such job", "", api.CodeJobNotFound)
		return cfg, t, "", false
	}
	return cfg, t, id, true
}

func (s *Server) handleJobDetail(w http.ResponseWriter, r *http.Request) {
	cfg, t, id, ok := s.jobTarget(w, r)
	if !ok {
		return
	}
	rec, entries, agentTime, err := s.jobs.readVerified(r.Context(), cfg, t, id, true)
	if err != nil {
		writeNodeError(w, err)
		return
	}
	d := api.JobDetail{Job: s.jobs.view(t.ID, rec, agentTime), Log: make([]api.JobLogLine, 0, len(entries))}
	for _, e := range entries {
		d.Log = append(d.Log, api.JobLogLine{Index: e.Index, Kind: e.Kind, At: time.Unix(int64(e.At), 0).UTC(), Text: joblog.Describe(e)})
	}
	writeJSON(w, http.StatusOK, d)
}

// handleJobOutput follows a job's journal through job.output, a batch per
// event, from the start (or from ?cursor=, which the agent checks).
func (s *Server) handleJobOutput(w http.ResponseWriter, r *http.Request) {
	cfg, t, id, ok := s.jobTarget(w, r)
	if !ok {
		return
	}
	conn, ok := startSSE(w)
	if !ok {
		return
	}
	defer conn.Close()
	cursor := r.URL.Query().Get("cursor")
	poll := func() {
		pctx, cancel := context.WithTimeout(r.Context(), agentPollTimeout)
		res, err := decodeResult[intent.JobOutputResult](s.agentResult(pctx, cfg, t, intent.KindJobOutput, intent.JobOutputPayload{ID: id, Cursor: cursor, N: intent.LogsMaxN}))
		cancel()
		if err != nil {
			_, e := apiErrorFor(err)
			conn.SendNamed("error", e) // apiclient's watch turns it into Update.Err
			return
		}
		if len(res.Lines) > 0 {
			lines := make([]api.JobOutputLine, len(res.Lines))
			for i, l := range res.Lines {
				lines[i] = api.JobOutputLine{Unit: l.Unit, Line: l.Line, At: l.At}
			}
			conn.Send(lines)
		}
		if res.Cursor != "" {
			cursor = res.Cursor
		}
	}
	poll()
	tick := time.NewTicker(jobsOutputEvery)
	defer tick.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-s.jobs.ctx.Done():
			return
		case <-conn.Pings():
			conn.Ping()
		case <-tick.C:
			poll()
		}
	}
}
