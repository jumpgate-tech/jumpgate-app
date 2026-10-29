package server

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/valve-tech/jumpgate/internal/catalog"
	"github.com/valve-tech/jumpgate/internal/executor"
	"github.com/valve-tech/jumpgate/internal/setup"
)

// runToEnd launches steps as a setup run and returns its events once it ends.
func runToEnd(t *testing.T, a *apiTestServer, steps []setup.Step) []setup.Event {
	t.Helper()
	claimed, busy := a.srv.tryClaimSetupRun("box")
	if busy != "" {
		t.Fatal(busy)
	}
	a.srv.launchSetupRun(claimed, nil, steps, catalog.WireConfig{})
	select {
	case <-claimed.run.done:
	case <-time.After(5 * time.Second):
		t.Fatal("the run never ended")
	}
	claimed.run.mu.Lock()
	defer claimed.run.mu.Unlock()
	return slices.Clone(claimed.run.events)
}

func step(id string, err error) setup.Step {
	return setup.Step{ID: id, Run: func(context.Context, executor.Executor, *setup.State) error { return err }}
}

// Live (tui): the setup stream had no terminal frame, so the TUI and the web
// modules section, which wait for {"stepId":"","done":true}, never saw a
// module change finish. The run now ends with that frame.
func TestSetupRunEndsWithATerminalFrame(t *testing.T) {
	a := newAPITestServer(t)
	evs := runToEnd(t, a, []setup.Step{step("one", nil), step("two", nil)})
	if last := evs[len(evs)-1]; last != (setup.Event{Done: true}) {
		t.Fatalf("last event %+v, want the terminal frame; all %+v", last, evs)
	}
	if evs[len(evs)-2].StepID != "two" {
		t.Fatalf("the terminal frame is not after the last step: %+v", evs)
	}

	// A failed step reports its error; the terminal frame does not repeat it.
	evs = runToEnd(t, a, []setup.Step{step("one", errors.New("boom"))})
	if last := evs[len(evs)-1]; last != (setup.Event{Done: true}) {
		t.Fatalf("after a failed step the last event is %+v; all %+v", last, evs)
	}
}
