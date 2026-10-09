// Package jobstest runs durable jobs in-process for the agent's and the
// server's tests: a Runner whose units are goroutines.
package jobstest

import (
	"context"
	"sync"
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
