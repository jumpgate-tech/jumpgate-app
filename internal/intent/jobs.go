// internal/intent/jobs.go
package intent

import (
	"encoding/json"
	"time"
)

// Job kinds (sub-project 2). node.migrate-names starts a job; the others act
// on one. All are routine tier by default (agent.DefaultRequirements).
const (
	KindJobList          = "job.list"
	KindJobRead          = "job.read"
	KindJobOutput        = "job.output"
	KindJobLease         = "job.lease"
	KindJobAdvance       = "job.advance"
	KindJobDecide        = "job.decide"
	KindNodeMigrateNames = "node.migrate-names"
)

// Job rejection reasons. Stable, like the others: the server maps each to a hint.
const (
	ReasonJobsUnsupported = "jobs_unsupported" // not Linux, no systemd, or no job manager
	ReasonJobNotFound     = "job_not_found"
	ReasonJobActive       = "job_active"   // another job is not finished; one at a time per box
	ReasonLeaseHeld       = "lease_held"   // another controller holds the job's lease
	ReasonUnknownPlan     = "unknown_plan" // the manifest's format, driver or plan version is unknown here
	ReasonJobState        = "job_state"    // the job is not where this request expects it
)

// Job states.
const (
	JobWaiting       = "waiting"        // at a step boundary, for the lease holder to advance
	JobRunning       = "running"        // a step's unit is running
	JobNeedsDecision = "needs-decision" // stopped at a decision point
	JobSucceeded     = "succeeded"
	JobFailed        = "failed"
	JobAborted       = "aborted"
	JobDamaged       = "damaged" // its log or manifest no longer verifies on the box
)

// JobTerminal reports whether a job in state will never change again.
func JobTerminal(state string) bool {
	switch state {
	case JobSucceeded, JobFailed, JobAborted, JobDamaged:
		return true
	}
	return false
}

// Lease operations, options every failure question offers, question tiers.
const (
	LeaseTake    = "take"
	LeaseRenew   = "renew"
	LeaseRelease = "release"

	OptRetry = "retry"
	OptAbort = "abort"

	TierRoutine  = "routine"
	TierApproval = "approval"
)

type JobStep struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// JobLease is who drives a job. Freshness is the agent's to judge, by its own
// clock: HeartbeatAt is agent unix time.
type JobLease struct {
	Owner       string `json:"owner,omitempty"`
	HeartbeatAt int64  `json:"heartbeatAt,omitempty"`
	TTLSeconds  int    `json:"ttlSeconds,omitempty"`
}

type JobOption struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Detail string `json:"detail,omitempty"`
}

// JobQuestion is a decision point. ID is unique within the job; Key is the
// step's own name for it (what the step reads back from the answer).
type JobQuestion struct {
	ID      string      `json:"id"`
	Step    string      `json:"step"`
	Key     string      `json:"key,omitempty"`
	Prompt  string      `json:"prompt"`
	Options []JobOption `json:"options"`
	Tier    string      `json:"tier"`
	AskedAt int64       `json:"askedAt"`
}

// JobRecord is a job as the agent reports it: the manifest without its inputs.
type JobRecord struct {
	Format       int             `json:"format"`
	ID           string          `json:"id"`
	Kind         string          `json:"kind"`
	Title        string          `json:"title"`
	PlanVersion  string          `json:"planVersion"`
	Driver       string          `json:"driver"`
	Steps        []JobStep       `json:"steps"`
	Step         int             `json:"step"` // index into Steps; len(Steps) once finished
	Attempt      int             `json:"attempt"`
	State        string          `json:"state"`
	Unit         string          `json:"unit,omitempty"`
	Question     *JobQuestion    `json:"question,omitempty"`
	Lease        JobLease        `json:"lease"`
	CreatedBy    string          `json:"createdBy"`
	CreatedAt    int64           `json:"createdAt"`
	UpdatedAt    int64           `json:"updatedAt"`
	Error        string          `json:"error,omitempty"`
	Result       json.RawMessage `json:"result,omitempty"`
	LogLen       uint64          `json:"logLen"`
	LogHead      string          `json:"logHead"`
	AgentVersion string          `json:"agentVersion"`
}

type JobListResult struct {
	Jobs      []JobRecord `json:"jobs"`
	AgentTime int64       `json:"agentTime"`
}

type JobReadPayload struct {
	ID   string `json:"id"`
	From uint64 `json:"from"`
}

// JobReadResult pages the log: at most 500 entries from From.
type JobReadResult struct {
	Job       JobRecord         `json:"job"`
	Entries   []JobLogEntryJSON `json:"entries"`
	More      bool              `json:"more"`
	AgentTime int64             `json:"agentTime"`
}

type JobOutputPayload struct {
	ID     string `json:"id"`
	Cursor string `json:"cursor"`
	N      int    `json:"n"`
}

type JobOutputLine struct {
	Unit string    `json:"unit"`
	Line string    `json:"line"`
	At   time.Time `json:"at"`
}

type JobOutputResult struct {
	Lines  []JobOutputLine `json:"lines"`
	Cursor string          `json:"cursor"`
}

type JobLeasePayload struct {
	ID string `json:"id"`
	Op string `json:"op"`
}

// JobAdvancePayload names the position the controller saw, so an advance sent
// twice, or after another controller moved the job, is refused (job_state).
type JobAdvancePayload struct {
	ID      string `json:"id"`
	Step    int    `json:"step"`
	Attempt int    `json:"attempt"`
}

type JobDecidePayload struct {
	ID       string `json:"id"`
	Decision string `json:"decision"`
	Option   string `json:"option"`
}

// MigrateNamesPayload starts node.migrate-names. It takes no inputs: the box's
// own node.json says what to rename.
type MigrateNamesPayload struct{}

// JobLogEntryJSON is one signed log entry on the wire and on disk.
// Body is base64 in JSON, hashed as stored (joblog).
type JobLogEntryJSON struct {
	Index uint64 `json:"index"`
	Prev  string `json:"prev"`
	Kind  string `json:"kind"`
	Body  []byte `json:"body"`
	At    uint64 `json:"at"`
	Sig   string `json:"sig"`
}
