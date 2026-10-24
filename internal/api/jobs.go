package api

import (
	"encoding/json"
	"time"
)

// JobLease is who drives a job, as this server sees it. Fresh is judged by
// the box's clock (the agent reports its time with every list).
type JobLease struct {
	Owner       string     `json:"owner,omitempty"`
	Label       string     `json:"label,omitempty"` // "this controller" when Mine
	HeartbeatAt *time.Time `json:"heartbeatAt,omitempty"`
	Fresh       bool       `json:"fresh"`
	Mine        bool       `json:"mine"`
}

type JobStepView struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	State string `json:"state"` // done, running, asking, waiting, pending, stopped
}

type JobOption struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Detail string `json:"detail,omitempty"`
}

// JobQuestion is a decision point. Signable is false while its tier has no
// signer this server can use (the approval tier, until it ships).
type JobQuestion struct {
	ID       string      `json:"id"`
	Prompt   string      `json:"prompt"`
	Options  []JobOption `json:"options"`
	Tier     string      `json:"tier"`
	AskedAt  time.Time   `json:"askedAt"`
	Signable bool        `json:"signable"`
}

// JobView is one durable job on one box.
type JobView struct {
	Target    string        `json:"target"`
	ID        string        `json:"id"`
	Kind      string        `json:"kind"`
	Title     string        `json:"title"`
	State     string        `json:"state"`
	Step      int           `json:"step"`
	Steps     []JobStepView `json:"steps"`
	Attempt   int           `json:"attempt"`
	Question  *JobQuestion  `json:"question,omitempty"`
	Lease     JobLease      `json:"lease"`
	CreatedAt time.Time     `json:"createdAt"`
	UpdatedAt time.Time     `json:"updatedAt"`
	Error     string        `json:"error,omitempty"`
	Driving   bool          `json:"driving"` // this server is advancing it now
}

type JobsError struct {
	Target string `json:"target"`
	Error  Error  `json:"error"`
}

// JobsInbox is GET /api/jobs and each /api/jobs/stream event: every job on
// every paired box, and the boxes that could not be asked.
type JobsInbox struct {
	At     time.Time   `json:"at"`
	Jobs   []JobView   `json:"jobs"`
	Errors []JobsError `json:"errors"`
}

// NeedsDecision is the fleet-wide ⚑ count.
func (in JobsInbox) NeedsDecision() int {
	n := 0
	for _, j := range in.Jobs {
		if j.State == "needs-decision" {
			n++
		}
	}
	return n
}

// JobLogLine is one verified log entry, described for a person.
type JobLogLine struct {
	Index uint64    `json:"index"`
	Kind  string    `json:"kind"`
	At    time.Time `json:"at"`
	Text  string    `json:"text"`
}

// JobDetail is GET /api/jobs/{target}/{job}. Log holds only entries that
// verified; a log that does not is an error (job_log_invalid), never a partial list.
type JobDetail struct {
	Job JobView      `json:"job"`
	Log []JobLogLine `json:"log"`
}

type JobOutputLine struct {
	Unit string    `json:"unit"`
	Line string    `json:"line"`
	At   time.Time `json:"at"`
}

// JobDecision is POST /api/jobs/{target}/{job}/decide.
type JobDecision struct {
	Decision string `json:"decision"`
	Option   string `json:"option"`
}

// JobStart is POST /api/jobs/{target}.
type JobStart struct {
	Kind   string          `json:"kind"`
	Inputs json.RawMessage `json:"inputs,omitempty"`
}

// RunningJob is one unfinished job's place in its plan: Step of Total steps
// done (Step is 0-based, the one running or asking).
type RunningJob struct {
	Target        string `json:"target"`
	ID            string `json:"id"`
	Kind          string `json:"kind"`
	State         string `json:"state"`
	Step          int    `json:"step"`
	Total         int    `json:"total"`
	NeedsDecision bool   `json:"needsDecision"`
}

// RunningJobs is GET /api/jobs/running: every unfinished job the server
// already knows of, from its own cache. It never asks a box, so it is as
// fresh as the last listing (the fleet poll's, or a driver's read).
type RunningJobs struct {
	At   time.Time    `json:"at"`
	Jobs []RunningJob `json:"jobs"`
}
