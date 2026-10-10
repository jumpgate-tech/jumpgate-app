// internal/server/jobs.go
package server

import (
	"context"
	"encoding/hex"
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
	jobsDriveEvery  = 5 * time.Second  // Task 12's step driver
	jobsRenewEvery  = 20 * time.Second // Task 12's lease renewals
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

// verifiedLog is a log prefix this server verified, and the agent key it
// verified it against: a re-paired box (or another box re-added under the
// same id) must never be credited with entries its key did not sign.
type verifiedLog struct {
	agent   eip712.Address
	entries []joblog.Entry
	head    [32]byte
}

// Bounds on one job's log, against a box that holds its agent key and signs
// an endless valid chain (variables so tests shorten them). A job logs its
// creation, each request that changes it, and each step's start and outcome;
// lease renewals are saved, not logged (Q7). A six-hour job of a few steps,
// retried and decided a few times, is a few hundred entries, so 5000 is
// over ten times anything a plan here produces. Entries carry at most an
// intent envelope or a step result, a few KiB each, so 16 MiB (one agent
// reply's limit, agentclient.maxReply) is likewise far above a real log.
// The deadline bounds the whole paged read, not each page.
var (
	jobLogMaxEntries  = 5000
	jobLogMaxBytes    = 16 << 20
	jobLogReadTimeout = 60 * time.Second
)

// jobsHub is this server's view of every paired box's durable jobs: it lists
// them (lazily, at most every jobsListEvery), verifies their logs, serves the
// inbox, and (Task 12) drives the jobs whose lease this controller holds.
type jobsHub struct {
	s      *Server
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// afterList runs after each box is listed, outside mu (Task 12 resumes
	// this controller's own jobs there).
	afterList func(t config.Target, jobs []intent.JobRecord)
	// readJob asks a box for one page of a job (job.read). It is a field so
	// a test can stand in for an agent that lies about its log.
	readJob func(ctx context.Context, cfg config.Config, t config.Target, p intent.JobReadPayload) (intent.JobReadResult, error)

	mu       sync.Mutex
	boxes    map[string]*boxJobs
	listedAt time.Time
	listing  chan struct{}
	subs     map[chan api.JobsInbox]struct{}
	logs     map[jobKey]*verifiedLog
	driving  map[jobKey]context.CancelFunc
	stopped  bool
}

func newJobsHub(s *Server) *jobsHub {
	ctx, cancel := context.WithCancel(context.Background())
	h := &jobsHub{s: s, ctx: ctx, cancel: cancel, boxes: map[string]*boxJobs{}, subs: map[chan api.JobsInbox]struct{}{},
		logs: map[jobKey]*verifiedLog{}, driving: map[jobKey]context.CancelFunc{}}
	h.readJob = func(ctx context.Context, cfg config.Config, t config.Target, p intent.JobReadPayload) (intent.JobReadResult, error) {
		return decodeResult[intent.JobReadResult](s.agentResult(ctx, cfg, t, intent.KindJobRead, p))
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
	for k := range h.logs {
		if !keep[k.target] {
			delete(h.logs, k) // the box was removed or unpaired
		}
	}
	h.mu.Unlock()
}

func (h *jobsHub) listBox(ctx context.Context, cfg config.Config, t config.Target) {
	res, err := decodeResult[intent.JobListResult](h.s.agentResult(ctx, cfg, t, intent.KindJobList, struct{}{}))
	box := &boxJobs{jobs: res.Jobs, agentTime: res.AgentTime}
	var rej *agentRejected
	switch {
	case errors.As(err, &rej) && (rej.Code == intent.ReasonJobsUnsupported || rej.Code == intent.ReasonUnknownKind):
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
		// A job the box no longer lists was pruned there (J6): its cached
		// log goes too, so the cache never outgrows what the boxes keep.
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

// dropLogs forgets target's cached logs, except those of the jobs in keep.
func (h *jobsHub) dropLogs(target string, keep map[string]bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for k := range h.logs {
		if k.target == target && !keep[k.id] {
			delete(h.logs, k)
		}
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
		v.Lease = api.JobLease{Owner: r.Lease.Owner, HeartbeatAt: &hb, Fresh: agentTime-r.Lease.HeartbeatAt < ttl, Mine: h.mine(r)}
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

// readVerified reads a job and its whole log, every page job.read has,
// verifying only the entries this server has not verified before. Each page
// must be signed by the box's agent and chained onto the last (Verify), and
// every request in it must be one its controller signed for this agent and
// this job (VerifyIntents). Verify proves only a valid prefix, so the log
// read in full must then be the one the job's record describes: both its
// length and its head digest, or a box could drop the tail or hand over
// another log. Anything less is errJobLogInvalid, and the cache for the job
// is dropped, so no part of that log is shown.
func (h *jobsHub) readVerified(ctx context.Context, cfg config.Config, t config.Target, id string) (intent.JobRecord, []joblog.Entry, int64, error) {
	at, err := agentTarget(t)
	if err != nil {
		return intent.JobRecord{}, nil, 0, &localError{err}
	}
	ctx, cancel := context.WithTimeout(ctx, jobLogReadTimeout)
	defer cancel()
	k := jobKey{t.ID, id}
	h.mu.Lock()
	vl := h.logs[k]
	if vl == nil || vl.agent != at.Agent {
		vl = &verifiedLog{} // nothing verified against this key: start over
	}
	entries, head := append([]joblog.Entry(nil), vl.entries...), vl.head
	h.mu.Unlock()
	var rec intent.JobRecord
	var agentTime int64
	var claimed uint64 // the first page's LogLen: the most this read takes
	bytes := 0
	for first := true; ; first = false {
		res, err := h.readJob(ctx, cfg, t, intent.JobReadPayload{ID: id, From: uint64(len(entries))})
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
		if uint64(len(entries)+len(res.Entries)) > claimed {
			return h.invalid(k, rec, fmt.Errorf("the box sent more than the %d log entries the job claimed", claimed))
		}
		page := make([]joblog.Entry, 0, len(res.Entries))
		for _, j := range res.Entries {
			if bytes += len(j.Body) + len(j.Kind) + len(j.Prev) + len(j.Sig); bytes > jobLogMaxBytes {
				return h.invalid(k, rec, fmt.Errorf("the job's log is larger than the %d bytes this server reads", jobLogMaxBytes))
			}
			e, err := joblog.FromJSON(j)
			if err != nil {
				return h.invalid(k, rec, err)
			}
			page = append(page, e)
		}
		if head, err = joblog.Verify(at.Agent, id, page, uint64(len(entries)), head); err != nil {
			return h.invalid(k, rec, err)
		}
		if err := joblog.VerifyIntents(at.Agent, id, page); err != nil {
			return h.invalid(k, rec, err)
		}
		entries = append(entries, page...)
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
	// The record must describe the log just verified, or the box is lying
	// about one of them.
	got := "0x" + hex.EncodeToString(head[:])
	if uint64(len(entries)) != rec.LogLen || got != recordedHead(rec) {
		return h.invalid(k, rec, fmt.Errorf("the job says its log has %d entries ending %q; %d verified, ending %s", rec.LogLen, rec.LogHead, len(entries), got))
	}
	// A job's log starts with the request that created it, and only there:
	// VerifyIntents binds that entry to the job id, so a log without it, or
	// with a second one, credits a request the id does not commit to.
	if len(entries) == 0 {
		return h.invalid(k, rec, fmt.Errorf("the job's log is empty; it must start with its %s entry", joblog.KindCreated))
	}
	for i, e := range entries {
		if (i == 0) != (e.Kind == joblog.KindCreated) {
			return h.invalid(k, rec, fmt.Errorf("entry %d is %q; a job's log has its one %s entry first", i, e.Kind, joblog.KindCreated))
		}
	}
	h.mu.Lock()
	h.logs[k] = &verifiedLog{agent: at.Agent, entries: entries, head: head}
	h.mu.Unlock()
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
	delete(h.logs, k)
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
	rec, entries, agentTime, err := s.jobs.readVerified(r.Context(), cfg, t, id)
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
