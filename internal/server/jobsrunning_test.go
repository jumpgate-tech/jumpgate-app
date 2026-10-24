package server

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/intent"
)

// GET /api/jobs/running answers from the hub's cache alone: the activity
// drawer reads it on every view, so it must never send a box an intent
// (TUI spec: no new box polling). jobsListEvery is 0 in the rig, so a
// handler that refreshed would list the box here and fail the test.
func TestRunningJobsNeverAsksABox(t *testing.T) {
	requireAgentPeer(t)
	b := newBoxRig(t, withoutDriver)
	// The hooks go in before startJob, so no goroutine startJob leaves
	// running reads them mid-write (-race); they only judge once armed,
	// since startJob itself sends and lists.
	var armed atomic.Bool
	prev := onSendIntent
	onSendIntent = func(target, kind string) {
		if armed.Load() {
			t.Errorf("GET /api/jobs/running sent %s to %s", kind, target)
		}
	}
	t.Cleanup(func() { onSendIntent = prev })
	listJobs := b.srv.jobs.listJobs
	b.srv.jobs.listJobs = func(ctx context.Context, cfg config.Config, tg config.Target) (intent.JobListResult, error) {
		if armed.Load() {
			t.Error("GET /api/jobs/running listed a box")
			return intent.JobListResult{}, nil
		}
		return listJobs(ctx, cfg, tg)
	}
	readJob := b.srv.jobs.readJob
	b.srv.jobs.readJob = func(ctx context.Context, cfg config.Config, tg config.Target, p intent.JobReadPayload) (intent.JobReadResult, error) {
		if armed.Load() {
			t.Error("GET /api/jobs/running read a job")
			return intent.JobReadResult{}, nil
		}
		return readJob(ctx, cfg, tg, p)
	}
	rec := b.startJob(t, `{}`)
	armed.Store(true)

	var got api.RunningJobs
	if code := b.do(t, "GET", "/api/jobs/running", nil, &got); code != 200 {
		t.Fatalf("code %d", code)
	}
	if len(got.Jobs) != 1 {
		t.Fatalf("jobs %+v", got)
	}
	j := got.Jobs[0]
	if j.Target != "box" || j.ID != rec.ID || j.Total != len(rec.Steps) || j.Step != rec.Step || j.NeedsDecision || j.Kind != rec.Kind {
		t.Fatalf("job %+v, record %+v", j, rec)
	}
}

// Finished jobs are not running; a job asking a person is, flagged.
func TestRunningJobsShape(t *testing.T) {
	s := &Server{}
	h := newJobsHub(s)
	t.Cleanup(h.cancel)
	steps := make([]intent.JobStep, 3)
	h.boxes["b"] = &boxJobs{jobs: []intent.JobRecord{
		{ID: "a", Kind: "node.x", State: intent.JobSucceeded, Steps: steps, Step: 3},
		{ID: "c", Kind: "node.x", State: intent.JobNeedsDecision, Steps: steps, Step: 1},
	}}
	h.boxes["a"] = &boxJobs{jobs: []intent.JobRecord{{ID: "d", Kind: "node.y", State: intent.JobWaiting, Steps: steps, Step: 2}}}
	got := h.running().Jobs
	if len(got) != 2 || got[0].Target != "a" || got[0].Step != 2 || got[0].Total != 3 ||
		got[1].ID != "c" || !got[1].NeedsDecision || got[1].State != intent.JobNeedsDecision {
		t.Fatalf("%+v", got)
	}
}
