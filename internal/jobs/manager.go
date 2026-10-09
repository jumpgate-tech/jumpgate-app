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
// describes the change. A crash between that entry and the next can cost a
// trailing entry (decision.asked, job.finished), never leave a job running
// for a second fold.
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
	man, _, err := m.load(id)
	var r *Refusal
	if errors.As(err, &r) && r.Code == intent.ReasonUnknownPlan {
		err = nil // a lease on a job this agent cannot advance is still a lease; it lets a controller watch and decide nothing
	}
	if err != nil {
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
		if err := m.store.Append(ctx, &man, joblog.KindIntent, joblog.IntentBody{Envelope: env, Outcome: "lease take"}); err != nil {
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
		if err := m.store.Append(ctx, &man, joblog.KindIntent, joblog.IntentBody{Envelope: env, Outcome: "lease release"}); err != nil {
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
	if man.Decisions == nil {
		man.Decisions = map[string]string{}
	}
	man.Decisions[decision] = option
	man.Question = nil
	answered := joblog.AnswerBody{Question: decision, Option: option, Controller: who.Hex(), Signers: hexes}
	if option == intent.OptAbort {
		man.State = intent.JobAborted
		man.Error = fmt.Sprintf("aborted at step %s by %s", plan.Steps[man.Step].ID, who.Hex())
		if err := m.store.Append(ctx, &man, joblog.KindDecisionAnswered, answered); err != nil {
			return intent.JobRecord{}, err
		}
		if err := m.store.Append(ctx, &man, joblog.KindFinished, joblog.FinishedBody{State: man.State, Error: man.Error}); err != nil {
			return intent.JobRecord{}, err
		}
		return man.JobRecord, nil
	}
	man.Answer = &Answer{Key: q.Key, Option: option, Step: man.Step, Attempt: man.Attempt}
	man.Attempt++
	man.State = intent.JobWaiting
	if err := m.store.Append(ctx, &man, joblog.KindDecisionAnswered, answered); err != nil {
		return intent.JobRecord{}, err
	}
	return man.JobRecord, nil
}

// Reconcile folds every finished step into the log: run on agent start,
// every 5 s, and before each job intent. A unit whose state cannot be read
// is left for the next pass, never guessed at; a damaged job, or one whose
// plan this agent does not know, is left alone.
func (m *Manager) Reconcile(ctx context.Context) error {
	m.lock()
	defer m.unlock()
	ms, err := m.store.List()
	if err != nil {
		return err
	}
	for _, j := range ms {
		if j.State != intent.JobRunning {
			continue // List labels a damaged job damaged, so it never gets here
		}
		man, plan, err := m.load(j.ID)
		if err != nil {
			continue // unknown plan or damage: shown by List, never advanced
		}
		if man.State != intent.JobRunning || man.Step >= len(plan.Steps) {
			continue
		}
		active, err := m.cfg.Runner.Active(ctx, man.Unit)
		if err != nil || active {
			continue
		}
		if err := m.fold(ctx, &man, plan); err != nil {
			return err
		}
	}
	return nil
}

// fold records how the job's ended unit went, from its result file. Each case
// moves the manifest out of running with its first entry, so the fold
// happens once.
func (m *Manager) fold(ctx context.Context, man *Manifest, plan Plan) error {
	s, attempt, unit := plan.Steps[man.Step], man.Attempt, man.Unit
	path, err := ResultPath(m.store.Dir(), man.ID, s.ID, man.Attempt)
	if err != nil {
		return err
	}
	b, err := os.ReadFile(path)
	var res StepResult
	if err == nil {
		err = json.Unmarshal(b, &res)
	}
	switch {
	case err != nil:
		why := "it stopped without reporting a result (the box may have rebooted, or the step was killed)"
		if !errors.Is(err, fs.ErrNotExist) {
			why = "its result file does not parse: " + err.Error()
		}
		q := failureQuestion(s, man.Attempt, why)
		q.ID = fmt.Sprintf("%s-%d-lost", s.ID, man.Attempt)
		return m.askAfter(ctx, man, joblog.KindStepLost, joblog.StepBody{Step: s.ID, Attempt: man.Attempt, Unit: unit, Error: why}, q)
	case !res.OK:
		return m.askAfter(ctx, man, joblog.KindStepFinished, joblog.StepBody{Step: s.ID, Attempt: man.Attempt, Unit: unit, Error: res.Error},
			failureQuestion(s, man.Attempt, res.Error))
	case res.Ask != nil:
		q := *res.Ask
		q.ID, q.Step = fmt.Sprintf("%s-%d-%s", s.ID, man.Attempt, q.Key), s.ID
		return m.askAfter(ctx, man, joblog.KindStepFinished, joblog.StepBody{Step: s.ID, Attempt: man.Attempt, Unit: unit, OK: true}, q)
	}
	if man.Results == nil {
		man.Results = map[string]json.RawMessage{}
	}
	man.Results[s.ID] = res.Result
	man.Step, man.Attempt, man.Answer, man.Unit = man.Step+1, 0, nil, ""
	man.State = intent.JobWaiting
	if man.Step == len(plan.Steps) {
		man.State, man.Result = intent.JobSucceeded, res.Result
	}
	if err := m.store.Append(ctx, man, joblog.KindStepFinished, joblog.StepBody{Step: s.ID, Attempt: attempt, Unit: unit, OK: true}); err != nil {
		return err
	}
	if man.State == intent.JobSucceeded {
		return m.store.Append(ctx, man, joblog.KindFinished, joblog.FinishedBody{State: man.State})
	}
	return nil
}

// askAfter stops the job at q: it appends the entry that explains why (kind,
// body), with the question already in the manifest, then decision.asked.
func (m *Manager) askAfter(ctx context.Context, man *Manifest, kind string, body any, q intent.JobQuestion) error {
	if q.Tier == "" {
		q.Tier = intent.TierRoutine
	}
	q.AskedAt = m.cfg.Now().Unix()
	man.Question, man.State, man.Unit = &q, intent.JobNeedsDecision, ""
	if err := m.store.Append(ctx, man, kind, body); err != nil {
		return err
	}
	return m.store.Append(ctx, man, joblog.KindDecisionAsked, q)
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
