// internal/server/jobsdrive.go
package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/catalog"
	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/intent"
	"github.com/valve-tech/jumpgate/internal/jobs"
)

// jobStartKinds are the kinds POST /api/jobs/{target} starts (ruling P7).
// The route signs nothing but a kind from this list, so it never becomes a
// way to send an arbitrary intent ("intents, not commands"; a new node
// operation is a new intent kind, not a new route). A new job kind is added
// here on purpose, with its plan on the agent. A variable so tests can add
// jobstest.TestKind.
var jobStartKinds = map[string]bool{intent.KindNodeMigrateNames: true}

// The v1 driver is step-level (spec): the agent runs each step to its end on
// its own, and the lease holder starts the next one. A job whose controller
// is gone finishes its current step and waits at the boundary.

// wireDriver connects the hub's list hook: every list resumes this
// controller's own unfinished jobs (Q4: own leases resume by themselves;
// other controllers' never do, since taking a lease over is an operator's
// adopt), and settles any finished job whose result this controller has not
// yet copied into its record of the box, whoever drove it.
func (h *jobsHub) wireDriver() {
	h.afterList = func(t config.Target, recs []intent.JobRecord) {
		for _, r := range recs {
			switch {
			case !intent.JobTerminal(r.State) && h.mine(r):
				h.drive(t.ID, r.ID)
			case intent.JobTerminal(r.State):
				h.finishLater(t, r)
			}
		}
	}
}

// sweep is run when the server starts: list every paired box now, which
// resumes every job this controller held when it last ran. A server with no
// controller key holds no leases, so its sweep only lists.
func (h *jobsHub) sweep(ctx context.Context) { h.refresh(ctx, true) }

// drive starts the driver for one job, once. A hub built without a driver
// (Config.noJobsDriver) never starts one, and an id that is not a job id
// (a box lists what it likes) is never driven.
func (h *jobsHub) drive(target, id string) {
	if !jobs.ValidID(id) {
		return
	}
	k := jobKey{target, id}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.stopped || h.noDriver || h.driving[k] != nil {
		return
	}
	ctx, cancel := context.WithCancel(h.ctx)
	h.driving[k] = cancel
	// Added under mu, before stopWithin can set stopped and Wait.
	h.wg.Add(1)
	go func() {
		defer h.wg.Done()
		defer func() {
			cancel()
			h.mu.Lock()
			delete(h.driving, k)
			h.mu.Unlock()
			h.publish()
		}()
		h.driveLoop(ctx, k)
	}()
}

// jobsDriveBackoffMax caps how far a driver backs off while its box cannot
// be reached: from jobsDriveEvery, doubling (a variable so tests shorten it).
var jobsDriveBackoffMax = 60 * time.Second

// refused reports whether the box answered err with a refusal. A driver
// ends on any refusal: whether another controller holds the job, this
// controller's key was revoked, or the box will not take the request for any
// other reason, sending it again every few seconds only signs more intents.
// The next listing that finds the job still this controller's starts a
// driver again, which is how a box fixed meanwhile gets its job moving.
func refused(err error) bool {
	var rej *agentRejected
	return errors.As(err, &rej)
}

// driveLoop reads the job every jobsDriveEvery: advances it at a step
// boundary, renews the lease every jobsRenewEvery, and returns when the job
// ends, another controller holds it, the box refuses a request, or its log
// does not verify (a job whose history cannot be checked is never driven).
// Only a box that cannot be reached is tried again, backing off to
// jobsDriveBackoffMax. It never takes a lease: a job whose lease another
// controller took is no longer this driver's.
func (h *jobsHub) driveLoop(ctx context.Context, k jobKey) {
	var lastRenew time.Time
	wait := jobsDriveEvery
	for {
		cfg, err := h.s.loadConfig()
		if err != nil {
			return
		}
		t, ok := findTarget(cfg, k.target)
		if !ok || t.Agent == nil {
			return // the box was removed or unpaired
		}
		rec, _, _, err := h.readVerified(ctx, cfg, t, k.id)
		switch {
		case errors.Is(err, errJobLogInvalid), refused(err):
			return
		case err != nil:
			// Unreachable or slow: try again later; the lease ages on the box.
		case intent.JobTerminal(rec.State):
			h.finished(t, rec)
			return
		case !h.mine(rec):
			return // adopted by another controller, or released
		case rec.State == intent.JobWaiting:
			_, err = h.s.agentResult(ctx, cfg, t, intent.KindJobAdvance, intent.JobAdvancePayload{ID: k.id, Step: rec.Step, Attempt: rec.Attempt})
			if err == nil {
				lastRenew = time.Now() // an advance renews the lease
			}
		case time.Since(lastRenew) >= jobsRenewEvery:
			_, err = h.s.agentResult(ctx, cfg, t, intent.KindJobLease, intent.JobLeasePayload{ID: k.id, Op: intent.LeaseRenew})
			if err == nil {
				lastRenew = time.Now()
			}
		}
		switch {
		case refused(err):
			return
		case err != nil:
			wait = min(wait*2, max(jobsDriveBackoffMax, jobsDriveEvery))
		default:
			wait = jobsDriveEvery
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(wait):
		}
	}
}

// finishLater runs finished for a terminal job the hub has not settled yet,
// in a goroutine the hub tracks, so a listing does not wait on agent.info.
func (h *jobsHub) finishLater(t config.Target, rec intent.JobRecord) {
	if !changesTheBox(rec) {
		return
	}
	k := jobKey{t.ID, rec.ID}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.stopped || h.noDriver || h.settled[k] {
		return
	}
	h.wg.Add(1)
	go func() {
		defer h.wg.Done()
		h.finished(t, rec)
	}()
}

// changesTheBox reports whether rec is a finished job whose result changes
// this controller's record of the box.
func changesTheBox(rec intent.JobRecord) bool {
	return rec.Kind == intent.KindNodeMigrateNames && rec.State == intent.JobSucceeded
}

// finished copies what a finished job changed into this controller's record
// of the box. node.migrate-names moves the box's names, data dir and JWT;
// without this the next setup re-run and diagnostics would go on using
// units and paths that no longer exist (Review Focus 4). The new values come
// from the box's agent.info, which applies the names marker the job wrote
// (D-J10), not from the job's result: the box's node config is what every
// later step and setup reads. A box that cannot answer, or whose node config
// it cannot read just now, leaves the job unsettled, and the next listing
// tries again; an answer that rules the update out settles it unchanged.
func (h *jobsHub) finished(t config.Target, rec intent.JobRecord) {
	if !changesTheBox(rec) {
		return
	}
	k := jobKey{t.ID, rec.ID}
	h.mu.Lock()
	if h.settled[k] {
		h.mu.Unlock()
		return
	}
	h.settled[k] = true
	h.mu.Unlock()
	err := h.applyBoxNames(t.ID)
	var final *wireLeftAlone
	if errors.As(err, &final) {
		// Said once per job and process: retrying would ask the box the
		// same question on every listing for the same answer.
		log.Printf("jumpgate: %s: job %s migrated the box's names, but its stored node config was left as it was: %v", t.ID, rec.ID, final.why)
		return
	}
	if err != nil {
		h.mu.Lock()
		delete(h.settled, k)
		h.mu.Unlock()
	}
}

// wireLeftAlone is an agent.info answer after which the stored wire must not
// take the box's names: retrying would not change it.
type wireLeftAlone struct{ why string }

func (e *wireLeftAlone) Error() string { return e.why }

// applyBoxNames reads the box's names and paths from agent.info and, when
// they differ from the stored wire, saves them and retires the observers
// built from the old wire, so the next viewer builds them from the new one.
// A target with no wire has nothing to correct: a paired box's agent owns
// its node config.
func (h *jobsHub) applyBoxNames(target string) error {
	cfg, err := h.s.loadConfig()
	if err != nil {
		return err
	}
	t, ok := findTarget(cfg, target)
	if !ok || t.Agent == nil || t.Wire == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(h.ctx, fleetProbeTimeout)
	defer cancel()
	info, err := decodeResult[intent.AgentInfo](h.s.agentResult(ctx, cfg, t, intent.KindAgentInfo, struct{}{}))
	if err != nil {
		return err
	}
	box, err := boxNamesFrom(info, *t.Wire)
	if err != nil {
		return err
	}
	changed := false
	if _, err := h.s.updateConfig(func(c *config.Config) error {
		for i := range c.Targets {
			w := c.Targets[i].Wire
			if c.Targets[i].ID != target || w == nil || w.ChainID != box.ChainID {
				continue
			}
			if w.Names != box.Names || w.DataDir != box.DataDir || w.JWTPath != box.JWTPath {
				w.Names, w.DataDir, w.JWTPath = box.Names, box.DataDir, box.JWTPath
				changed = true
			}
		}
		return nil
	}); err != nil {
		return err
	}
	if changed {
		h.s.retireObservers(target)
	}
	return nil
}

// boxNamesFrom is the chain, names, data dir and JWT path agent.info
// reports, each checked as setup would check them: every one reaches a root
// command line through setup and diagnostics. A box set up for another
// chain than the stored wire's, or no longer set up at all, is not the node
// the wire describes, and gets nothing of it (wireLeftAlone). A node config
// the box refuses, or an answer too old to name a data dir (an agent that
// could not have run the job), is an error to retry. An unreported JWT path
// keeps the stored one while the data dir stays put, as setup does
// (setupExistingFrom).
func boxNamesFrom(info intent.AgentInfo, old catalog.WireConfig) (catalog.WireConfig, error) {
	switch {
	case info.SetUpError != nil:
		return catalog.WireConfig{}, &agentRejected{*info.SetUpError}
	case !info.SetUp:
		return catalog.WireConfig{}, &wireLeftAlone{"the box reports no node set up"}
	case info.ChainID != old.ChainID:
		return catalog.WireConfig{}, &wireLeftAlone{fmt.Sprintf("the box runs chain %d, and the stored config is for chain %d", info.ChainID, old.ChainID)}
	case info.DataDir == "":
		return catalog.WireConfig{}, errors.New("the box does not report its node's data dir")
	}
	box := catalog.WireConfig{ChainID: info.ChainID, Names: info.Names, DataDir: info.DataDir, JWTPath: info.JWTPath}
	if err := catalog.ValidateNames(box); err != nil {
		return box, err
	}
	if err := catalog.ValidateDataDir(box.DataDir); err != nil {
		return box, err
	}
	if box.JWTPath == "" && box.DataDir == old.DataDir {
		box.JWTPath = old.JWTPath
	}
	if box.JWTPath != "" {
		if err := catalog.ValidateDataDir(box.JWTPath); err != nil {
			return box, fmt.Errorf("jwt path: %w", err)
		}
	}
	return box, nil
}

// retireObservers stops a target's monitor, log watcher and log follower, if
// it has any, so the next use builds them from the target's current wire.
// It never creates a registry entry for a target that has none.
func (s *Server) retireObservers(id string) {
	s.reg.mu.Lock()
	e, ok := s.reg.entries[id]
	s.reg.mu.Unlock()
	if !ok {
		return
	}
	e.mu.Lock()
	e.retireObserversLocked()
	e.mu.Unlock()
}

// record reads a job fresh from the box and verifies its log, so a decision
// is checked against what the box says now, never against a cached record.
func (h *jobsHub) record(ctx context.Context, cfg config.Config, t config.Target, id string) (intent.JobRecord, int64, error) {
	rec, _, agentTime, err := h.readVerified(ctx, cfg, t, id)
	return rec, agentTime, err
}

func (s *Server) handleJobStart(w http.ResponseWriter, r *http.Request) {
	cfg, t, _, ok := s.jobTarget(w, r)
	if !ok {
		return
	}
	var in api.JobStart
	if err := json.NewDecoder(io.LimitReader(r.Body, maxIntentBody)).Decode(&in); err != nil || in.Kind == "" {
		writeError(w, http.StatusBadRequest, "want {\"kind\": …, \"inputs\": …}")
		return
	}
	if !jobStartKinds[in.Kind] {
		writeErrorDetail(w, http.StatusBadRequest, fmt.Sprintf("%q is not a job this server starts", in.Kind), "", api.CodeJobKindNotAllowed)
		return
	}
	if len(in.Inputs) == 0 {
		in.Inputs = json.RawMessage(`{}`)
	}
	rec, err := decodeResult[intent.JobRecord](s.agentResult(r.Context(), cfg, t, in.Kind, in.Inputs))
	if err != nil {
		writeNodeError(w, err)
		return
	}
	s.jobs.setRecord(t.ID, rec, rec.Lease.HeartbeatAt)
	s.jobs.drive(t.ID, rec.ID)
	writeJSON(w, http.StatusOK, s.jobs.view(t.ID, rec, rec.Lease.HeartbeatAt))
}

func (s *Server) handleJobAdopt(w http.ResponseWriter, r *http.Request) {
	cfg, t, id, ok := s.jobTarget(w, r)
	if !ok {
		return
	}
	rec, err := decodeResult[intent.JobRecord](s.agentResult(r.Context(), cfg, t, intent.KindJobLease, intent.JobLeasePayload{ID: id, Op: intent.LeaseTake}))
	if err != nil {
		writeNodeError(w, err)
		return
	}
	s.jobs.setRecord(t.ID, rec, rec.Lease.HeartbeatAt)
	s.jobs.drive(t.ID, id)
	writeJSON(w, http.StatusOK, s.jobs.view(t.ID, rec, rec.Lease.HeartbeatAt))
}

func (s *Server) handleJobDecide(w http.ResponseWriter, r *http.Request) {
	cfg, t, id, ok := s.jobTarget(w, r)
	if !ok {
		return
	}
	var d api.JobDecision
	if err := json.NewDecoder(io.LimitReader(r.Body, 4096)).Decode(&d); err != nil || d.Decision == "" || d.Option == "" {
		writeError(w, http.StatusBadRequest, "want {\"decision\": …, \"option\": …}")
		return
	}
	rec, _, err := s.jobs.record(r.Context(), cfg, t, id)
	if err != nil {
		writeNodeError(w, err)
		return
	}
	switch {
	case !s.jobs.mine(rec) && rec.Lease.Owner == "":
		writeErrorDetail(w, http.StatusConflict, "nobody holds this job; adopt it to decide", "", api.CodeLeaseHeld)
		return
	case !s.jobs.mine(rec):
		writeErrorDetail(w, http.StatusConflict, "controller "+rec.Lease.Owner+" holds this job; you are watching it", "", api.CodeLeaseHeld)
		return
	case rec.Question != nil && rec.Question.ID == d.Decision && rec.Question.Tier == intent.TierApproval:
		// Q5: this server holds only a routine key, so an approval-tier
		// answer is refused here, before anything is signed. The agent
		// enforces the tier itself in any case (Manager.Decide).
		writeErrorDetail(w, http.StatusConflict, "this decision needs an approval-tier signature", "", api.CodeApprovalUnavailable)
		return
	}
	rec, err = decodeResult[intent.JobRecord](s.agentResult(r.Context(), cfg, t, intent.KindJobDecide, intent.JobDecidePayload{ID: id, Decision: d.Decision, Option: d.Option}))
	if err != nil {
		writeNodeError(w, err)
		return
	}
	s.jobs.setRecord(t.ID, rec, rec.Lease.HeartbeatAt)
	s.jobs.drive(t.ID, id)
	writeJSON(w, http.StatusOK, s.jobs.view(t.ID, rec, rec.Lease.HeartbeatAt))
}
