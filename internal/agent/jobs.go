package agent

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"time"

	"github.com/valve-tech/jumpgate/internal/eip712"
	"github.com/valve-tech/jumpgate/internal/executor"
	"github.com/valve-tech/jumpgate/internal/intent"
	"github.com/valve-tech/jumpgate/internal/jobs"
)

// ReconcileEvery is how often RunJobs folds finished steps into their logs (D-J2).
var ReconcileEvery = 5 * time.Second

// pruneEvery is how often RunJobs prunes finished jobs (D-J6).
const pruneEvery = 24 * time.Hour

func isJobKind(kind string) bool {
	switch kind {
	case intent.KindJobList, intent.KindJobRead, intent.KindJobOutput, intent.KindJobLease, intent.KindJobAdvance, intent.KindJobDecide:
		return true
	}
	_, ok := jobs.PlanForKind(kind)
	return ok
}

// jobFailure is a manager error as an answer. A refusal is a signed
// rejection with the manager's own reason code. Anything else, ErrStale and
// ErrDamaged included, is a signed failure: the controller sees it and
// decides, and nothing here retries it.
func jobFailure(err error) ([]byte, uint8, *Reject) {
	var r *jobs.Refusal
	if errors.As(err, &r) {
		return nil, 0, reject(r.Code, r.Message)
	}
	return failed(err)
}

// idPayload refuses a job id that is not one: only a checked id reaches a
// path or a command line.
func idPayload(id string) *Reject {
	if !jobs.ValidID(id) {
		return reject(intent.ReasonInvalidPayload, fmt.Sprintf("%q is not a job id", id))
	}
	return nil
}

// decodeJob decodes a job payload and checks the job id it names.
func decodeJob[T any](payload []byte, id func(T) string) (T, *Reject) {
	pl, rej := decode[T](payload)
	if rej == nil {
		rej = idPayload(id(pl))
	}
	return pl, rej
}

// dispatchJob answers the job kinds. Reconcile runs first, so every answer
// reflects units that finished since the last pass.
func (a *Agent) dispatchJob(ctx context.Context, i intent.Intent, env intent.Envelope, signers []eip712.Address, p Policy) ([]byte, uint8, *Reject) {
	m := a.cfg.Jobs
	if m == nil && a.cfg.JobsUnavailable != nil {
		// A controller lists a box's jobs to decide whether setup or a
		// clear may run (C1), and takes jobs_unsupported for "no jobs". A
		// store that may hold a job answers a failure instead, so the
		// controller refuses rather than run under it.
		if i.Kind == intent.KindJobList && a.unopenedJobStore() != nil {
			return failed(fmt.Errorf("the job store did not open (%v), and it may hold a job; fix it and restart the agent", a.cfg.JobsUnavailable))
		}
		return nil, 0, reject(intent.ReasonJobsUnsupported, fmt.Sprintf(
			"this agent runs without durable jobs: its job store did not open (%v); fix that and restart the agent", a.cfg.JobsUnavailable))
	}
	if m == nil {
		return nil, 0, reject(intent.ReasonJobsUnsupported, "this agent runs without durable jobs (no job store)")
	}
	a.reconcile(ctx)
	switch i.Kind {
	case intent.KindJobList:
		res, err := m.List()
		if err != nil {
			return jobFailure(err)
		}
		return mustJSON(res), intent.StatusOK, nil
	case intent.KindJobRead:
		pl, rej := decodeJob(env.Payload, func(p intent.JobReadPayload) string { return p.ID })
		if rej != nil {
			return nil, 0, rej
		}
		res, err := m.Read(pl.ID, pl.From)
		if err != nil {
			return jobFailure(err)
		}
		return mustJSON(res), intent.StatusOK, nil
	case intent.KindJobOutput:
		pl, rej := decodeJob(env.Payload, func(p intent.JobOutputPayload) string { return p.ID })
		if rej != nil {
			return nil, 0, rej
		}
		res, rej, err := jobOutput(ctx, a.cfg.Exec, pl, a.cfg.Now())
		if rej != nil {
			return nil, 0, rej
		}
		if err != nil {
			return failed(err)
		}
		return mustJSON(res), intent.StatusOK, nil
	case intent.KindJobLease:
		pl, rej := decodeJob(env.Payload, func(p intent.JobLeasePayload) string { return p.ID })
		if rej != nil {
			return nil, 0, rej
		}
		rec, err := m.Lease(ctx, pl.ID, pl.Op, i.Controller, env)
		if err != nil {
			return jobFailure(err)
		}
		return mustJSON(rec), intent.StatusOK, nil
	case intent.KindJobAdvance:
		pl, rej := decodeJob(env.Payload, func(p intent.JobAdvancePayload) string { return p.ID })
		if rej != nil {
			return nil, 0, rej
		}
		rec, err := m.Advance(ctx, pl.ID, pl.Step, pl.Attempt, i.Controller, env)
		if err != nil {
			return jobFailure(err)
		}
		return mustJSON(rec), intent.StatusOK, nil
	case intent.KindJobDecide:
		pl, rej := decodeJob(env.Payload, func(p intent.JobDecidePayload) string { return p.ID })
		if rej != nil {
			return nil, 0, rej
		}
		rec, err := m.Decide(ctx, pl.ID, pl.Decision, pl.Option, i.Controller, signers, decisionCheck(p, i.Controller, signers), env)
		if err != nil {
			return jobFailure(err)
		}
		return mustJSON(rec), intent.StatusOK, nil
	}

	// A plan's kind: start a job. i is env's intent, which Handle admitted:
	// signed by its controller, addressed to this box, of this kind, with
	// env.Payload under its payload hash. The envelope is logged as the
	// job's first entry, and its digest is the job's id.
	rec, err := m.Start(ctx, i.Kind, env.Payload, i.Controller, env)
	if err != nil {
		return jobFailure(err)
	}
	return mustJSON(rec), intent.StatusOK, nil
}

// activeJob is the box's unfinished job, if any (J1). Unlike Manager.Active
// it returns a listing error, so service.action fails closed rather than
// running under a job it could not see.
func (a *Agent) activeJob() (intent.JobRecord, bool, error) {
	if a.cfg.Jobs == nil {
		return intent.JobRecord{}, false, nil
	}
	res, err := a.cfg.Jobs.List()
	if err != nil {
		return intent.JobRecord{}, false, fmt.Errorf("cannot tell whether a job owns the node services: %w", err)
	}
	for _, j := range res.Jobs {
		if !intent.JobTerminal(j.State) {
			return j, true, nil
		}
	}
	return intent.JobRecord{}, false, nil
}

// unopenedJobStore is J1's refusal on an agent whose job store did not open:
// a job already in it may own the node services, and nothing can tell. Only
// a jobs directory that is absent, or holds nothing but its lock file,
// proves there is no job; one that cannot be read refuses too.
func (a *Agent) unopenedJobStore() *Reject {
	if a.cfg.Jobs != nil || a.cfg.JobsUnavailable == nil {
		return nil
	}
	ents, err := os.ReadDir(a.cfg.JobsDir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err == nil {
		empty := true
		for _, e := range ents {
			empty = empty && e.Name() == ".lock"
		}
		if empty {
			return nil
		}
	}
	return reject(intent.ReasonJobActive, fmt.Sprintf(
		"the job store could not be opened (%v), so a job in it may own the node services; fix the store and restart the agent", a.cfg.JobsUnavailable))
}

// jobOutput is a job's journal through logsSince, the one bounded read
// (D-J8, ruling P9). The units are matched by the job's own pattern, built
// from a validated id.
func jobOutput(ctx context.Context, ex executor.Executor, pl intent.JobOutputPayload, now time.Time) (intent.JobOutputResult, *Reject, error) {
	lines, cursor, rej, err := logsSince(ctx, ex, []string{jobs.UnitPattern(pl.ID)}, pl.Cursor, pl.N, now,
		func(e journalEntry) intent.JobOutputLine {
			return intent.JobOutputLine{Unit: e.unit, Line: e.line, At: e.at}
		})
	if rej != nil || err != nil {
		return intent.JobOutputResult{}, rej, err
	}
	return intent.JobOutputResult{Lines: lines, Cursor: cursor}, nil, nil
}

// reconcile runs one reconcile pass and logs what it could not do. Reconcile
// collects each job's error and still passes over the others, so one failed
// job never holds up the rest, and the next pass tries it again.
func (a *Agent) reconcile(ctx context.Context) {
	if err := a.cfg.Jobs.Reconcile(ctx); err != nil {
		a.logf("reconcile: %v", err)
	}
}

// RunJobs reconciles at once (an agent restart folds what finished while it
// was down), then every ReconcileEvery, and prunes once a day. A failed pass
// is logged, and the loop goes on until ctx ends.
func (a *Agent) RunJobs(ctx context.Context) {
	if a.cfg.Jobs == nil {
		return
	}
	prune := func() {
		if err := a.cfg.Jobs.Prune(); err != nil {
			a.logf("prune: %v", err)
		}
	}
	a.reconcile(ctx)
	prune()
	tick := time.NewTicker(ReconcileEvery)
	defer tick.Stop()
	lastPrune := time.Now()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			a.reconcile(ctx)
			if time.Since(lastPrune) > pruneEvery {
				prune()
				lastPrune = time.Now()
			}
		}
	}
}
