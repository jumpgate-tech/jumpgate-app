// Package jobs is the agent's durable-job engine: the manifests and signed
// logs under /var/lib/jumpgate/jobs, the leases controllers hold, the plans a
// job can run, and the transient systemd units each step runs in. The agent
// process is the only writer of manifests and the only signer; a step
// process reads its manifest and writes one result file (spec D-J4).
package jobs

import (
	"encoding/json"
	"errors"
	"regexp"

	"github.com/valve-tech/jumpgate/internal/intent"
)

// FormatVersion is the manifest format this agent reads and writes. The later
// agent-driven driver needs no new format (spec): it only sets Driver.
const FormatVersion = 1

const (
	DriverController = "controller" // v1: the lease holder advances step by step
	DriverAgent      = "agent"      // later: the agent drives the whole plan
)

// Manifest is /var/lib/jumpgate/jobs/<id>.json.
type Manifest struct {
	intent.JobRecord
	// Inputs are the job's inputs as the intent carried them: decisions, never
	// rendered commands (for setup, the wire config plus the plan version).
	Inputs json.RawMessage `json:"inputs"`
	// Results holds each finished step's result, by step id; later steps read them.
	Results map[string]json.RawMessage `json:"results,omitempty"`
	// Decisions is every answer given, by question id.
	Decisions map[string]string `json:"decisions,omitempty"`
	// Answer is the last answer, which the next attempt of its step reads.
	Answer *Answer `json:"answer,omitempty"`
}

// Answer is what a step asked and what it was told, so the step's next attempt
// can act on it (Env.Answer).
type Answer struct {
	Key     string `json:"key"`
	Option  string `json:"option"`
	Step    int    `json:"step"`
	Attempt int    `json:"attempt"`
}

// Refusal is a request the engine refuses; Code is an intent.Reason*. The
// agent turns it into a signed rejection.
type Refusal struct{ Code, Message string }

func (r *Refusal) Error() string { return r.Code + ": " + r.Message }

func refuse(code, msg string) error { return &Refusal{Code: code, Message: msg} }

var (
	ErrNotFound = errors.New("jobs: no such job")
	ErrDamaged  = errors.New("jobs: the job's log or manifest is damaged")
)

var idRE = regexp.MustCompile(`^[0-9a-f]{16}$`)

// ValidID reports whether id is 16 lowercase hex characters: safe in unit
// names, paths and journal patterns without quoting (D-J5). Ids come from
// joblog.JobIDFor, which derives them from the creating intent.
func ValidID(id string) bool { return idRE.MatchString(id) }
