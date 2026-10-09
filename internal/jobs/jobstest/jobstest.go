// Package jobstest runs durable jobs in-process for the agent's and the
// server's tests: a Runner whose units are goroutines.
package jobstest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/valve-tech/jumpgate/internal/intent"
	"github.com/valve-tech/jumpgate/internal/jobs"
)

// Runner runs each step's argv through Run in a goroutine. A unit is active
// until Run returns, or until Lose says the box rebooted under it.
type Runner struct {
	Run         func(ctx context.Context, unit string, argv []string) error
	Unsupported error

	mu      sync.Mutex
	active  map[string]bool
	started []string
	wg      sync.WaitGroup
}

func (r *Runner) Supported(context.Context) error { return r.Unsupported }

func (r *Runner) Start(_ context.Context, unit string, argv []string) error {
	r.mu.Lock()
	if r.active == nil {
		r.active = map[string]bool{}
	}
	r.active[unit] = true
	r.started = append(r.started, unit)
	r.mu.Unlock()
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		if r.Run != nil {
			_ = r.Run(context.Background(), unit, argv)
		}
		r.mu.Lock()
		delete(r.active, unit)
		r.mu.Unlock()
	}()
	return nil
}

func (r *Runner) Active(_ context.Context, unit string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.active[unit], nil
}

// Wait waits for every started unit's goroutine.
func (r *Runner) Wait() { r.wg.Wait() }

func (r *Runner) Started() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.started...)
}

// Lose marks unit finished without running anything more: a reboot.
func (r *Runner) Lose(unit string) {
	r.mu.Lock()
	delete(r.active, unit)
	r.mu.Unlock()
}

// TestKind is the plan RegisterPlan adds. A policy must list it under kinds
// (it has no default requirement), so tests also prove the policy is read.
const TestKind = "test.steps"

// RegisterPlan registers test.steps/1: steps one and two, failing or asking at
// the step its inputs name, once per job (an answer or a retry passes).
func RegisterPlan(t testing.TB) {
	t.Helper()
	type in struct{ FailAt, AskAt string }
	step := func(id string) jobs.StepDef {
		return jobs.StepDef{ID: id, Title: "Step " + id, AbortNote: "test",
			Run: func(ctx context.Context, env jobs.Env) (jobs.Outcome, error) {
				var i in
				_ = json.Unmarshal(env.Inputs, &i)
				if env.Out != nil {
					fmt.Fprintln(env.Out, "running", id)
				}
				if i.AskAt == id && env.Answer == nil {
					return jobs.Outcome{Ask: &intent.JobQuestion{Key: "pick", Prompt: "pick one", Tier: intent.TierRoutine,
						Options: []intent.JobOption{{ID: "go", Label: "Go on"}, {ID: intent.OptAbort, Label: "Stop"}}}}, nil
				}
				if i.FailAt == id && env.Answer == nil {
					return jobs.Outcome{}, errors.New(id + " broke")
				}
				return jobs.Outcome{Result: json.RawMessage(`{"did":"` + id + `"}`)}, nil
			}}
	}
	jobs.Register(jobs.Plan{Kind: TestKind, Version: TestKind + "/1", Title: "Test steps", Steps: []jobs.StepDef{step("one"), step("two")}})
	t.Cleanup(func() { jobs.Unregister(TestKind + "/1") })
}

// InProcess is a Runner.Run that runs the step the argv names, as the unit
// would, against the jobs directory dir.
func InProcess(dir string, base jobs.Env) func(ctx context.Context, unit string, argv []string) error {
	return func(ctx context.Context, _ string, argv []string) error {
		a, err := jobs.ParseStepArgs(argv)
		if err != nil {
			return err
		}
		return jobs.RunStep(ctx, dir, a, base)
	}
}
