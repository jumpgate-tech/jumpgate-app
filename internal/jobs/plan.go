package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"sync"

	"github.com/valve-tech/jumpgate/internal/executor"
	"github.com/valve-tech/jumpgate/internal/intent"
)

// Env is what a step runs with, on the box, as root, in its transient unit.
type Env struct {
	Exec      executor.Executor
	Inputs    json.RawMessage
	Results   map[string]json.RawMessage // earlier steps' results, by step id
	Answer    *Answer                    // the answer to this step's last question, if the previous attempt asked one
	NodePath  string                     // /etc/jumpgate/node.json
	NamesPath string                     // /etc/jumpgate/names.json
	Out       io.Writer                  // the unit's stdout: the journal
}

// Outcome is a step's answer. Ask stops the job at a decision point; the step
// runs again (next attempt) with Env.Answer set once someone decides.
type Outcome struct {
	Result json.RawMessage
	Ask    *intent.JobQuestion
}

// StepDef is one step. Run must be idempotent: adoption reruns the plan from
// the step the job stands at, and Run may find its work already done.
// AbortNote says what state the box is left in if the job is aborted at
// this step (Q8); it is shown with the abort option.
type StepDef struct {
	ID, Title, AbortNote string
	Run                  func(ctx context.Context, env Env) (Outcome, error)
}

// Plan is a job kind's steps at one version. A plan's steps never change
// within a version: a manifest records the version, and the agent refuses
// to advance a version it does not know.
type Plan struct {
	Kind, Version, Title string
	Steps                []StepDef
	Validate             func(inputs json.RawMessage) error
}

var (
	plansMu sync.RWMutex
	plans   = map[string]Plan{} // by version
)

// Register adds a plan; plans register in init. A duplicate or a bad step id
// panics: it is a programming error.
func Register(p Plan) {
	plansMu.Lock()
	defer plansMu.Unlock()
	if _, dup := plans[p.Version]; dup {
		panic("jobs: plan " + p.Version + " registered twice")
	}
	for _, s := range p.Steps {
		if !ValidStepID(s.ID) {
			panic(fmt.Sprintf("jobs: plan %s: bad step id %q", p.Version, s.ID))
		}
	}
	plans[p.Version] = p
}

// Unregister is for tests that register their own plans.
func Unregister(version string) {
	plansMu.Lock()
	delete(plans, version)
	plansMu.Unlock()
}

// PlanFor is the plan at exactly version, the one a manifest records.
func PlanFor(version string) (Plan, bool) {
	plansMu.RLock()
	defer plansMu.RUnlock()
	p, ok := plans[version]
	return p, ok
}

// PlanForKind is the newest registered version of kind. v1 has one version
// per kind; when a second appears, new jobs take the newest and running jobs
// keep theirs.
func PlanForKind(kind string) (Plan, bool) {
	plansMu.RLock()
	defer plansMu.RUnlock()
	var best Plan
	found := false
	for _, p := range plans {
		if p.Kind == kind && (!found || p.Version > best.Version) {
			best, found = p, true
		}
	}
	return best, found
}

// Versions are the plan versions this agent runs, for agent.info.
func Versions() []string {
	plansMu.RLock()
	defer plansMu.RUnlock()
	out := make([]string, 0, len(plans))
	for v := range plans {
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}

func (p Plan) index(step string) int {
	for i, s := range p.Steps {
		if s.ID == step {
			return i
		}
	}
	return -1
}

func (p Plan) stepInfos() []intent.JobStep {
	out := make([]intent.JobStep, len(p.Steps))
	for i, s := range p.Steps {
		out[i] = intent.JobStep{ID: s.ID, Title: s.Title}
	}
	return out
}
