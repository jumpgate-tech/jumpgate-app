package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"sync"
	"time"

	"github.com/valve-tech/jumpgate/internal/eip712"
	"github.com/valve-tech/jumpgate/internal/intent"
	"github.com/valve-tech/jumpgate/internal/joblog"
	"github.com/valve-tech/jumpgate/internal/signer"
)

const (
	readPageMax = 500
	pruneAge    = 30 * 24 * time.Hour
	pruneKeep   = 100
)

// The outcomes of the lease requests' intent entries, which restoreTrailing
// reads back to find a lease entry that never followed.
const (
	outcomeLeaseTake    = "lease take"
	outcomeLeaseRelease = "lease release"
)

type ManagerConfig struct {
	Dir      string
	Key      signer.Signer
	Runner   Runner
	StepArgv func(StepArgs) []string
	Now      func() time.Time
	Version  string // the agent's build, stamped into each manifest
}

// Manager is every job operation the agent performs. Its mutex makes it the
// single writer of manifests, between concurrent intents and the reconcile
// loop: each operation loads, changes and appends under it, so the store's
// ErrStale (another copy wrote first) can only mean a bug or a hand edit, and
// is returned rather than retried.
//
// Append saves the manifest with each entry. A change that appends several
// entries therefore records the request first (its intent entry), then puts
// the job's whole new state in the manifest with the first entry that
// describes the change. A crash or a failed write can still stop a change
// between an entry and its manifest save, or between its two entries.
// restoreTrailing repairs that from the log's last entry, and Reconcile,
// Lease, Advance and Decide run it before they write anything, so no new
// entry can bury a gap. It repairs:
//
//   - a fold outcome (step.finished or step.lost) logged but not saved: the
//     job leaves running as the entry says, so the step folds once;
//   - a decision.answered logged but not saved: the logged answer applies;
//   - a missing decision.asked after a step's outcome;
//   - a missing job.finished after the last step or an abort;
//   - a missing lease.taken or lease.released after its lease intent.
//
// It deliberately leaves an unsaved step.started or intent entry: the
// request it records changed nothing that was saved, its unit was never
// started, and a retried request logs its own; the extra entry is cosmetic
// and runs nothing twice.
type Manager struct {
	cfg   ManagerConfig
	store *Store
	mu    sync.Mutex
}

// NewManager opens the jobs directory, which locks it until Close: a second
// Manager on the same directory fails with ErrStoreInUse.
func NewManager(cfg ManagerConfig) (*Manager, error) {
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	s, err := OpenStore(cfg.Dir, cfg.Key, cfg.Now)
	if err != nil {
		return nil, err
	}
	return &Manager{cfg: cfg, store: s}, nil
}

// Close releases the jobs directory. The Manager must not be used afterwards.
func (m *Manager) Close() error {
	m.lock()
	defer m.unlock()
	return m.store.Close()
}

func (m *Manager) lock()   { m.mu.Lock() }
func (m *Manager) unlock() { m.mu.Unlock() }

// load reads a job for a change: known format, driver and plan, or refused.
func (m *Manager) load(id string) (Manifest, Plan, error) {
	man, err := m.store.Load(id)
	if errors.Is(err, ErrNotFound) {
		return man, Plan{}, refuse(intent.ReasonJobNotFound, "no job "+id+" on this box")
	}
	if errors.Is(err, ErrDamaged) {
		return man, Plan{}, refuse(intent.ReasonJobState, err.Error())
	}
	if err != nil {
		return man, Plan{}, err
	}
	plan, ok := PlanFor(man.PlanVersion)
	if man.Format != FormatVersion || man.Driver != DriverController || !ok {
		return man, plan, refuse(intent.ReasonUnknownPlan, fmt.Sprintf(
			"job %s is format %d, driver %q, plan %s; this agent runs format %d, driver %q and plans %v",
			id, man.Format, man.Driver, man.PlanVersion, FormatVersion, DriverController, Versions()))
	}
	return man, plan, nil
}

// activeLocked is the unfinished job, if any. A listing error is returned, so
// Start fails closed rather than starting a second job it could not see.
func (m *Manager) activeLocked() (Manifest, bool, error) {
	ms, err := m.store.List()
	if err != nil {
		return Manifest{}, false, err
	}
	for _, j := range ms {
		if !intent.JobTerminal(j.State) {
			return j, true, nil
		}
	}
	return Manifest{}, false, nil
}

// Active is the box's unfinished job, if any (J1).
func (m *Manager) Active() (intent.JobRecord, bool) {
	m.lock()
	defer m.unlock()
	j, ok, _ := m.activeLocked()
	return j.JobRecord, ok
}

// Start creates a job of kind for who. Its id is the one env's intent derives
// (joblog.JobIDFor), so the log's first entry proves which signed request
// started it.
func (m *Manager) Start(ctx context.Context, kind string, inputs []byte, who eip712.Address, env intent.Envelope) (intent.JobRecord, error) {
	plan, ok := PlanForKind(kind)
	if !ok {
		return intent.JobRecord{}, refuse(intent.ReasonUnknownKind, "this agent has no plan for "+kind)
	}
	if err := m.cfg.Runner.Supported(ctx); err != nil {
		return intent.JobRecord{}, refuse(intent.ReasonJobsUnsupported, err.Error())
	}
	if !json.Valid(inputs) {
		return intent.JobRecord{}, refuse(intent.ReasonInvalidPayload, "the job's inputs are not JSON")
	}
	if plan.Validate != nil {
		if err := plan.Validate(inputs); err != nil {
			return intent.JobRecord{}, refuse(intent.ReasonInvalidPayload, err.Error())
		}
	}
	in, err := env.Intent.Parse()
	if err != nil {
		return intent.JobRecord{}, refuse(intent.ReasonValidation, "the creating intent does not parse: "+err.Error())
	}
	id, err := joblog.JobIDFor(in)
	if err != nil {
		return intent.JobRecord{}, refuse(intent.ReasonValidation, "the creating intent has no digest: "+err.Error())
	}
	m.lock()
	defer m.unlock()
	j, busy, err := m.activeLocked()
	if err != nil {
		return intent.JobRecord{}, err
	}
	if busy {
		return intent.JobRecord{}, refuse(intent.ReasonJobActive, fmt.Sprintf("job %s (%s) is %s on this box; one job runs at a time", j.ID, j.Kind, j.State))
	}
	now := m.cfg.Now()
	man := Manifest{JobRecord: intent.JobRecord{Format: FormatVersion, ID: id, Kind: kind, Title: plan.Title,
		PlanVersion: plan.Version, Driver: DriverController, Steps: plan.stepInfos(), State: intent.JobWaiting,
		CreatedBy: who.Hex(), CreatedAt: now.Unix(), UpdatedAt: now.Unix(), AgentVersion: m.cfg.Version},
		Inputs: json.RawMessage(inputs)}
	_, _, _ = man.TakeLease(who, now) // a new job is its creator's to drive
	if err := m.store.Create(ctx, &man, joblog.IntentBody{Envelope: env, Outcome: "created"}); err != nil {
		return intent.JobRecord{}, err
	}
	return man.JobRecord, nil
}

func (m *Manager) List() (intent.JobListResult, error) {
	m.lock()
	defer m.unlock()
	ms, err := m.store.List()
	if err != nil {
		return intent.JobListResult{}, err
	}
	out := intent.JobListResult{Jobs: []intent.JobRecord{}, AgentTime: m.cfg.Now().Unix()}
	for _, j := range ms {
		out.Jobs = append(out.Jobs, j.JobRecord)
	}
	return out, nil
}

// Read is a job and a page of its log. A damaged job is still read, labelled
// damaged with the reason, with as much of its log as parses.
func (m *Manager) Read(id string, from uint64) (intent.JobReadResult, error) {
	m.lock()
	defer m.unlock()
	man, err := m.store.Load(id)
	if errors.Is(err, ErrNotFound) {
		return intent.JobReadResult{}, refuse(intent.ReasonJobNotFound, "no job "+id+" on this box")
	}
	if err != nil && !errors.Is(err, ErrDamaged) {
		return intent.JobReadResult{}, err
	}
	if errors.Is(err, ErrDamaged) {
		man.ID = id // the manifest itself may not have parsed
		man.State, man.Error = intent.JobDamaged, err.Error()
	}
	es, more, rerr := m.store.ReadLog(id, from, readPageMax)
	if rerr != nil && err == nil {
		return intent.JobReadResult{}, rerr
	}
	out := intent.JobReadResult{Job: man.JobRecord, Entries: []intent.JobLogEntryJSON{}, More: more, AgentTime: m.cfg.Now().Unix()}
	for _, e := range es {
		out.Entries = append(out.Entries, joblog.ToJSON(e))
	}
	return out, nil
}

func (m *Manager) Lease(ctx context.Context, id, op string, who eip712.Address, env intent.Envelope) (intent.JobRecord, error) {
	m.lock()
	defer m.unlock()
	man, plan, err := m.load(id)
	var r *Refusal
	if errors.As(err, &r) && r.Code == intent.ReasonUnknownPlan {
		err = nil // a lease on a job this agent cannot advance is still a lease; it lets a controller watch and decide nothing
	}
	if err != nil {
		return intent.JobRecord{}, err
	}
	if err := m.restoreTrailing(ctx, &man, plan); err != nil {
		return intent.JobRecord{}, err
	}
	now := m.cfg.Now()
	switch op {
	case intent.LeaseTake:
		changed, prev, err := man.TakeLease(who, now)
		if err != nil {
			return intent.JobRecord{}, err
		}
		if !changed {
			// The owner taking its own lease again is a renewal (Q7).
			return man.JobRecord, m.store.Save(&man)
		}
		if err := m.store.Append(ctx, &man, joblog.KindIntent, joblog.IntentBody{Envelope: env, Outcome: outcomeLeaseTake}); err != nil {
			return intent.JobRecord{}, err
		}
		if err := m.store.Append(ctx, &man, joblog.KindLeaseTaken, joblog.LeaseBody{Owner: who.Hex(), Previous: prev}); err != nil {
			return intent.JobRecord{}, err
		}
	case intent.LeaseRenew:
		// Renewals every 20 s would swamp the log; they are saved, not logged (Q7).
		if err := man.RenewLease(who, now); err != nil {
			return intent.JobRecord{}, err
		}
		if err := m.store.Save(&man); err != nil {
			return intent.JobRecord{}, err
		}
	case intent.LeaseRelease:
		if err := man.ReleaseLease(who); err != nil {
			return intent.JobRecord{}, err
		}
		if err := m.store.Append(ctx, &man, joblog.KindIntent, joblog.IntentBody{Envelope: env, Outcome: outcomeLeaseRelease}); err != nil {
			return intent.JobRecord{}, err
		}
		if err := m.store.Append(ctx, &man, joblog.KindLeaseReleased, joblog.LeaseBody{Owner: who.Hex()}); err != nil {
			return intent.JobRecord{}, err
		}
	default:
		return intent.JobRecord{}, refuse(intent.ReasonInvalidPayload, fmt.Sprintf("lease op %q", op))
	}
	return man.JobRecord, nil
}

// Advance starts the step the job stands at, as attempt attempt. The job is
// saved running, with its unit, before the unit starts: a crash in between
// leaves a running job whose unit never ran, which Reconcile reports as lost
// and asks about, rather than a waiting job whose next advance would start
// the step a second time.
func (m *Manager) Advance(ctx context.Context, id string, step, attempt int, who eip712.Address, env intent.Envelope) (intent.JobRecord, error) {
	m.lock()
	defer m.unlock()
	man, plan, err := m.load(id)
	if err != nil {
		return intent.JobRecord{}, err
	}
	if err := m.restoreTrailing(ctx, &man, plan); err != nil {
		return intent.JobRecord{}, err
	}
	if err := man.Holds(who); err != nil {
		return intent.JobRecord{}, err
	}
	if man.State != intent.JobWaiting || man.Step != step || man.Attempt != attempt || man.Step >= len(plan.Steps) {
		return intent.JobRecord{}, refuse(intent.ReasonJobState, fmt.Sprintf(
			"job %s is %s at step %d attempt %d; this advance was for step %d attempt %d", id, man.State, man.Step, man.Attempt, step, attempt))
	}
	_ = man.RenewLease(who, m.cfg.Now())
	s := plan.Steps[man.Step]
	args := StepArgs{Job: id, Step: s.ID, Attempt: man.Attempt}
	unit := UnitName(id, s.ID, man.Attempt)
	if err := m.store.Append(ctx, &man, joblog.KindIntent, joblog.IntentBody{Envelope: env, Outcome: "advance to " + s.ID}); err != nil {
		return intent.JobRecord{}, err
	}
	man.State, man.Unit = intent.JobRunning, unit
	if err := m.store.Append(ctx, &man, joblog.KindStepStarted, joblog.StepBody{Step: s.ID, Attempt: man.Attempt, Unit: unit}); err != nil {
		return intent.JobRecord{}, err
	}
	// Only checked ids and an integer reach the unit's command line; an
	// attempt past what a unit name holds cannot start, and asks like any
	// other step that cannot.
	startErr := args.valid()
	if startErr == nil {
		startErr = m.cfg.Runner.Start(ctx, unit, m.cfg.StepArgv(args))
	}
	if startErr != nil {
		why := "it could not start: " + startErr.Error()
		err := m.askAfter(ctx, &man, joblog.KindStepFinished, joblog.StepBody{Step: s.ID, Attempt: man.Attempt, Unit: unit, Error: why},
			failureQuestion(s, man.Attempt, why))
		if err != nil {
			return intent.JobRecord{}, err
		}
	}
	return man.JobRecord, nil
}

// QuestionTier is the tier the job's open question needs; routine when there
// is no such question (Decide then refuses it as job_state).
func (m *Manager) QuestionTier(id, decision string) string {
	m.lock()
	defer m.unlock()
	man, err := m.store.Load(id)
	if err != nil || man.Question == nil || man.Question.ID != decision || man.Question.Tier == "" {
		return intent.TierRoutine
	}
	return man.Question.Tier
}

func (m *Manager) Decide(ctx context.Context, id, decision, option string, who eip712.Address, signers []eip712.Address, env intent.Envelope) (intent.JobRecord, error) {
	m.lock()
	defer m.unlock()
	man, plan, err := m.load(id)
	if err != nil {
		return intent.JobRecord{}, err
	}
	if err := m.restoreTrailing(ctx, &man, plan); err != nil {
		return intent.JobRecord{}, err
	}
	if err := man.Holds(who); err != nil {
		return intent.JobRecord{}, err
	}
	q := man.Question
	if man.State != intent.JobNeedsDecision || q == nil || q.ID != decision || man.Step >= len(plan.Steps) {
		open := "none"
		if q != nil {
			open = q.ID
		}
		return intent.JobRecord{}, refuse(intent.ReasonJobState, fmt.Sprintf("job %s is not waiting on decision %q (open: %s)", id, decision, open))
	}
	known := false
	for _, o := range q.Options {
		known = known || o.ID == option
	}
	if !known {
		return intent.JobRecord{}, refuse(intent.ReasonInvalidPayload, fmt.Sprintf("%q is not an option of %s", option, decision))
	}
	_ = man.RenewLease(who, m.cfg.Now())
	hexes := make([]string, len(signers))
	for i, s := range signers {
		hexes[i] = s.Hex()
	}
	if err := m.store.Append(ctx, &man, joblog.KindIntent, joblog.IntentBody{Envelope: env, Outcome: "decide " + option}); err != nil {
		return intent.JobRecord{}, err
	}
	decideState(&man, plan, option, who.Hex())
	answered := joblog.AnswerBody{Question: decision, Option: option, Controller: who.Hex(), Signers: hexes}
	if err := m.store.Append(ctx, &man, joblog.KindDecisionAnswered, answered); err != nil {
		return intent.JobRecord{}, err
	}
	if man.State == intent.JobAborted {
		if err := m.store.Append(ctx, &man, joblog.KindFinished, joblog.FinishedBody{State: man.State, Error: man.Error}); err != nil {
			return intent.JobRecord{}, err
		}
	}
	return man.JobRecord, nil
}

// Reconcile brings every job's manifest and log back in line: run on agent
// start, every 5 s, and before each job intent. It folds each running job
// whose unit has ended, and restores the trailing entry of any change that
// stopped between its two entries. A unit whose state cannot be read is left
// for the next pass, never guessed at; a damaged job, or one whose plan this
// agent does not know, is left alone. One job's error does not hold up the
// others.
func (m *Manager) Reconcile(ctx context.Context) error {
	m.lock()
	defer m.unlock()
	ms, err := m.store.List()
	if err != nil {
		return err
	}
	var errs []error
	for _, j := range ms {
		if j.State == intent.JobDamaged {
			continue // evidence: never written to
		}
		man, plan, err := m.load(j.ID)
		if err != nil {
			continue // unknown plan or damage: shown by List, never advanced
		}
		if err := m.reconcileJob(ctx, &man, plan); err != nil {
			errs = append(errs, fmt.Errorf("job %s: %w", j.ID, err))
		}
	}
	return errors.Join(errs...)
}

func (m *Manager) reconcileJob(ctx context.Context, man *Manifest, plan Plan) error {
	// First any repair, which may move a running job on (an unsaved fold);
	// then a fold, only for a job still running.
	if err := m.restoreTrailing(ctx, man, plan); err != nil {
		return err
	}
	if man.State != intent.JobRunning || man.Step >= len(plan.Steps) {
		return nil
	}
	active, err := m.cfg.Runner.Active(ctx, man.Unit)
	if err != nil || active {
		return nil
	}
	return m.fold(ctx, man, plan)
}

// fold records how the job's ended unit went and moves the job on. Its
// caller has run restoreTrailing on man, so the log does not already hold
// this attempt's outcome: an unsaved one is rebuilt there, never appended
// twice.
func (m *Manager) fold(ctx context.Context, man *Manifest, plan Plan) error {
	s, attempt, unit := plan.Steps[man.Step], man.Attempt, man.Unit
	path, err := ResultPath(m.store.Dir(), man.ID, s.ID, attempt)
	if err != nil {
		return err
	}
	kind, body, res := readOutcome(path, s.ID, attempt, unit)
	m.settle(man, plan, kind, body, res)
	if err := m.store.Append(ctx, man, kind, body); err != nil {
		return err
	}
	return m.restoreTrailing(ctx, man, plan)
}

// refold is fold for an outcome already in the log whose manifest save
// failed (Load repairs only the log's length and head, so the manifest still
// says running): it rebuilds the state the entry records and saves it.
func (m *Manager) refold(man *Manifest, plan Plan, kind string, body joblog.StepBody) error {
	path, err := ResultPath(m.store.Dir(), man.ID, body.Step, body.Attempt)
	if err != nil {
		return err
	}
	var res StepResult
	if kind == joblog.KindStepFinished && body.OK {
		// The entry says the step succeeded; what it produced is in its
		// result file, which nothing rewrites.
		if res, err = readResult(path); err != nil {
			return fmt.Errorf("step %s attempt %d is logged as finished but its result cannot be read again: %w", body.Step, body.Attempt, err)
		}
	}
	m.settle(man, plan, kind, body, res)
	return m.store.Save(man)
}

// foldedEntry is the step.finished or step.lost the log ends with, if it is
// this attempt's: the sign of a fold whose manifest save failed.
func (m *Manager) foldedEntry(man *Manifest, step string, attempt int, unit string) (string, joblog.StepBody, bool, error) {
	var b joblog.StepBody
	e, ok, err := m.lastEntry(man)
	if err != nil || !ok || (e.Kind != joblog.KindStepFinished && e.Kind != joblog.KindStepLost) {
		return "", b, false, err
	}
	if err := json.Unmarshal(e.Body, &b); err != nil {
		return "", b, false, fmt.Errorf("%w: job %s entry %d: %v", ErrDamaged, man.ID, e.Index, err)
	}
	if b.Step != step || b.Attempt != attempt || b.Unit != unit {
		return "", b, false, nil
	}
	return e.Kind, b, true, nil
}

func readResult(path string) (StepResult, error) {
	var res StepResult
	b, err := os.ReadFile(path)
	if err == nil {
		err = json.Unmarshal(b, &res)
	}
	return res, err
}

// readOutcome is the entry a step's result file calls for: step.lost when
// there is no readable result, else step.finished.
func readOutcome(path, step string, attempt int, unit string) (string, joblog.StepBody, StepResult) {
	body := joblog.StepBody{Step: step, Attempt: attempt, Unit: unit}
	res, err := readResult(path)
	switch {
	case err != nil:
		body.Error = "it stopped without reporting a result (the box may have rebooted, or the step was killed)"
		if !errors.Is(err, fs.ErrNotExist) {
			body.Error = "its result file does not parse: " + err.Error()
		}
		return joblog.KindStepLost, body, res
	case !res.OK:
		body.Error = res.Error
	default:
		body.OK = true
	}
	return joblog.KindStepFinished, body, res
}

// settle moves man out of running as the step's entry (kind, body) and, for
// a step that succeeded, its result res say.
func (m *Manager) settle(man *Manifest, plan Plan, kind string, body joblog.StepBody, res StepResult) {
	s := plan.Steps[man.Step]
	switch {
	case kind == joblog.KindStepLost:
		q := failureQuestion(s, man.Attempt, body.Error)
		q.ID = fmt.Sprintf("%s-%d-lost", s.ID, man.Attempt)
		m.setQuestion(man, q)
	case !body.OK:
		m.setQuestion(man, failureQuestion(s, man.Attempt, body.Error))
	case res.Ask != nil:
		q := *res.Ask
		q.ID, q.Step = fmt.Sprintf("%s-%d-%s", s.ID, man.Attempt, q.Key), s.ID
		m.setQuestion(man, q)
	default:
		if man.Results == nil {
			man.Results = map[string]json.RawMessage{}
		}
		man.Results[s.ID] = res.Result
		man.Step, man.Attempt, man.Answer, man.Unit = man.Step+1, 0, nil, ""
		man.State = intent.JobWaiting
		if man.Step == len(plan.Steps) {
			man.State, man.Result = intent.JobSucceeded, res.Result
		}
	}
}

// setQuestion stops the job at q.
func (m *Manager) setQuestion(man *Manifest, q intent.JobQuestion) {
	if q.Tier == "" {
		q.Tier = intent.TierRoutine
	}
	q.AskedAt = m.cfg.Now().Unix()
	man.Question, man.State, man.Unit = &q, intent.JobNeedsDecision, ""
}

// askAfter stops the job at q: it appends the entry that explains why (kind,
// body), with the question already in the manifest, then decision.asked.
func (m *Manager) askAfter(ctx context.Context, man *Manifest, kind string, body any, q intent.JobQuestion) error {
	m.setQuestion(man, q)
	if err := m.store.Append(ctx, man, kind, body); err != nil {
		return err
	}
	return m.store.Append(ctx, man, joblog.KindDecisionAsked, *man.Question)
}

// decideState applies option, chosen by controller by, to the job's open
// question.
func decideState(man *Manifest, plan Plan, option, by string) {
	q := man.Question
	if man.Decisions == nil {
		man.Decisions = map[string]string{}
	}
	man.Decisions[q.ID] = option
	man.Question = nil
	if option == intent.OptAbort {
		man.State = intent.JobAborted
		man.Error = fmt.Sprintf("aborted at step %s by %s", plan.Steps[man.Step].ID, by)
		return
	}
	man.Answer = &Answer{Key: q.Key, Option: option, Step: man.Step, Attempt: man.Attempt}
	man.Attempt++
	man.State = intent.JobWaiting
}

// lastEntry is the log's last entry; ok is false for an empty log.
func (m *Manager) lastEntry(man *Manifest) (joblog.Entry, bool, error) {
	if man.LogLen == 0 {
		return joblog.Entry{}, false, nil
	}
	es, _, err := m.store.ReadLog(man.ID, man.LogLen-1, 1)
	if err != nil {
		return joblog.Entry{}, false, err
	}
	if len(es) != 1 {
		return joblog.Entry{}, false, fmt.Errorf("%w: job %s's log has no entry %d", ErrDamaged, man.ID, man.LogLen-1)
	}
	return es[0], true, nil
}

// restoreTrailing finishes a change that stopped partway. Append saves the
// manifest with each entry, so a crash or a failed write can leave either of
// two gaps, and both show as the log's last entry:
//
//   - The entry is written but the manifest save after it failed. Load then
//     repairs only the log's length and head, so the manifest still predates
//     the entry. A fold outcome (step.finished or step.lost) is settled
//     with refold; a decision.answered is applied with decideState.
//   - A change's first entry and its manifest are saved, and its second
//     entry (decision.asked, job.finished, lease.taken, lease.released) is
//     missing. It is appended here.
//
// Every operation runs this before it writes, so a gap is closed before a
// new entry could hide it.
func (m *Manager) restoreTrailing(ctx context.Context, man *Manifest, plan Plan) error {
	// A running job whose log ends with this attempt's outcome was folded
	// but not saved. It leaves running here, before anything can append past
	// that outcome and hide it from the check. (A job whose plan this agent
	// does not know cannot be settled; nothing here advances it.)
	if man.State == intent.JobRunning && man.Step < len(plan.Steps) {
		kind, body, written, err := m.foldedEntry(man, plan.Steps[man.Step].ID, man.Attempt, man.Unit)
		if err != nil {
			return err
		}
		if written {
			if err := m.refold(man, plan, kind, body); err != nil {
				return err
			}
		}
	}
	e, ok, err := m.lastEntry(man)
	if err != nil || !ok {
		return err
	}
	bad := func(err error) error {
		return fmt.Errorf("%w: job %s entry %d: %v", ErrDamaged, man.ID, e.Index, err)
	}
	finished := func() error {
		return m.store.Append(ctx, man, joblog.KindFinished, joblog.FinishedBody{State: man.State, Error: man.Error})
	}
	switch e.Kind {
	case joblog.KindStepFinished, joblog.KindStepLost:
		var b joblog.StepBody
		if err := json.Unmarshal(e.Body, &b); err != nil {
			return bad(err)
		}
		switch {
		case man.State == intent.JobNeedsDecision && man.Question != nil && man.Question.Step == b.Step:
			return m.store.Append(ctx, man, joblog.KindDecisionAsked, *man.Question)
		case man.State == intent.JobSucceeded:
			return finished()
		}
	case joblog.KindDecisionAnswered:
		var b joblog.AnswerBody
		if err := json.Unmarshal(e.Body, &b); err != nil {
			return bad(err)
		}
		if man.State == intent.JobNeedsDecision && man.Question != nil && man.Question.ID == b.Question && man.Step < len(plan.Steps) {
			decideState(man, plan, b.Option, b.Controller)
			if err := m.store.Save(man); err != nil {
				return err
			}
		}
		if man.State == intent.JobAborted {
			return finished()
		}
	case joblog.KindIntent:
		var b joblog.IntentBody
		if err := json.Unmarshal(e.Body, &b); err != nil {
			return bad(err)
		}
		if b.Outcome != outcomeLeaseTake && b.Outcome != outcomeLeaseRelease {
			return nil
		}
		// The holder before this request is the one the log records:
		// renewals and retakes by the holder change no owner and log nothing.
		prev, err := m.loggedOwner(man)
		if err != nil {
			return err
		}
		switch {
		case b.Outcome == outcomeLeaseTake && man.Lease.Owner != "" && man.Lease.Owner != prev:
			return m.store.Append(ctx, man, joblog.KindLeaseTaken, joblog.LeaseBody{Owner: man.Lease.Owner, Previous: prev})
		case b.Outcome == outcomeLeaseRelease && man.Lease.Owner == "" && prev != "":
			return m.store.Append(ctx, man, joblog.KindLeaseReleased, joblog.LeaseBody{Owner: prev})
		}
	}
	return nil
}

// loggedOwner is the lease holder the log records: the creator, then each
// lease.taken and lease.released in turn.
func (m *Manager) loggedOwner(man *Manifest) (string, error) {
	es, _, err := m.store.ReadLog(man.ID, 0, 0)
	if err != nil {
		return "", err
	}
	owner := man.CreatedBy
	for _, e := range es {
		if e.Kind != joblog.KindLeaseTaken && e.Kind != joblog.KindLeaseReleased {
			continue
		}
		var b joblog.LeaseBody
		if err := json.Unmarshal(e.Body, &b); err != nil {
			return "", fmt.Errorf("%w: job %s entry %d: %v", ErrDamaged, man.ID, e.Index, err)
		}
		owner = b.Owner
		if e.Kind == joblog.KindLeaseReleased {
			owner = ""
		}
	}
	return owner, nil
}

// failureQuestion is the decision every failed or lost step stops at: nothing
// reruns without a person (Q9).
func failureQuestion(s StepDef, attempt int, why string) intent.JobQuestion {
	return intent.JobQuestion{ID: fmt.Sprintf("%s-%d-failed", s.ID, attempt), Step: s.ID,
		Prompt: fmt.Sprintf("Step %q failed: %s", s.Title, why), Tier: intent.TierRoutine,
		Options: []intent.JobOption{
			{ID: intent.OptRetry, Label: "Retry the step", Detail: "every step checks what is already done before it acts"},
			{ID: intent.OptAbort, Label: "Stop the job here", Detail: s.AbortNote},
		}}
}

// Prune applies D-J6.
func (m *Manager) Prune() error {
	m.lock()
	defer m.unlock()
	return m.store.Prune(pruneAge, pruneKeep)
}
