package jobs

import (
	"context"
	"errors"
	"fmt"

	"github.com/valve-tech/jumpgate/internal/intent"
	"github.com/valve-tech/jumpgate/internal/joblog"
)

// AbandonedNote is the error an abandoned job ends with, in its manifest and
// in its job.finished entry.
const AbandonedNote = "abandoned by a local operator (jumpgate agent job-abandon)"

// Abandon ends job id as failed, on a local operator's word: the way out for
// a job no signed request can move, such as one whose step result cannot be
// read again (restoreTrailing then fails on every write, and J1 keeps the
// node services and every new job waiting on it for good).
//
// It opens the jobs directory itself, so it fails with ErrStoreInUse while
// the agent runs: the agent must be stopped, or two writers would sign the
// same index. The job.finished it appends is signed by the agent's key, so
// the log still verifies, and says who ended the job. It writes nothing past
// a log that does not verify, and refuses a job that has finished or whose
// unit still runs: ending the job while its step acts on the box would let
// the next job start beside it.
func Abandon(ctx context.Context, cfg ManagerConfig, id string) (intent.JobRecord, error) {
	m, err := NewManager(cfg)
	if err != nil {
		return intent.JobRecord{}, err
	}
	defer m.Close()
	return m.abandon(ctx, id)
}

func (m *Manager) abandon(ctx context.Context, id string) (intent.JobRecord, error) {
	m.lock()
	defer m.unlock()
	// Load, not load: a job whose plan this agent does not know may be
	// abandoned too. Load still verifies the whole log.
	man, err := m.store.Load(id)
	if errors.Is(err, ErrNotFound) {
		return intent.JobRecord{}, refuse(intent.ReasonJobNotFound, "no job "+id+" on this box")
	}
	if err != nil {
		return intent.JobRecord{}, err
	}
	if intent.JobTerminal(man.State) {
		return intent.JobRecord{}, refuse(intent.ReasonJobState, fmt.Sprintf("job %s has already ended (%s)", id, man.State))
	}
	// A box without systemd runs no units, so none can still be running.
	busy := false
	if err := m.cfg.Runner.Supported(ctx); err == nil {
		if busy, err = m.cfg.Runner.AnyActive(ctx, id); err != nil {
			return intent.JobRecord{}, fmt.Errorf("cannot tell whether a unit of job %s still runs: %w", id, err)
		}
	} else if !errors.Is(err, ErrUnsupported) {
		return intent.JobRecord{}, fmt.Errorf("cannot tell whether a unit of job %s still runs: %w", id, err)
	}
	if busy {
		return intent.JobRecord{}, refuse(intent.ReasonJobState, fmt.Sprintf(
			"a unit of job %s is still running; let it end, or stop it (systemctl stop '%s'), then abandon the job", id, UnitPattern(id)))
	}
	at := ""
	if man.Step < len(man.Steps) {
		at = " at step " + man.Steps[man.Step].ID
	}
	man.State, man.Error = intent.JobFailed, AbandonedNote+at
	man.Question, man.Unit, man.Lease = nil, "", intent.JobLease{}
	if err := m.store.Append(ctx, &man, joblog.KindFinished, joblog.FinishedBody{State: man.State, Error: man.Error}); err != nil {
		return intent.JobRecord{}, err
	}
	return man.JobRecord, nil
}
