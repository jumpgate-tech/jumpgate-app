package jobstest

import (
	"context"
	"encoding/json"
	"io"
	"path/filepath"
	"testing"
	"time"

	"github.com/valve-tech/jumpgate/internal/intent"
	"github.com/valve-tech/jumpgate/internal/jobs"
	"github.com/valve-tech/jumpgate/internal/signer"
	"github.com/valve-tech/jumpgate/internal/testutil"
)

// The helpers drive a real Manager: the test plan's steps run in goroutine
// units, through the same argv a systemd unit would get.
func TestInProcessRunsTheTestPlan(t *testing.T) {
	testutil.RequireUnix(t)
	RegisterPlan(t)
	dir := filepath.Join(t.TempDir(), "jobs")
	box, _ := signer.GenerateKey()
	ctl, _ := signer.GenerateKey()
	r := &Runner{Run: InProcess(dir, jobs.Env{Out: io.Discard})}
	m, err := jobs.NewManager(jobs.ManagerConfig{Dir: dir, Key: box, Runner: r,
		StepArgv: func(a jobs.StepArgs) []string { return jobs.StepArgv("/jumpgate", "/state", "/etc/jumpgate", a) }})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { m.Close() })

	nonce, _ := intent.NewNonce()
	in := intent.Intent{Agent: box.Address(), Controller: ctl.Address(), Nonce: nonce,
		IssuedAt: uint64(time.Now().Unix()), Expiry: uint64(time.Now().Unix() + 600), Kind: TestKind, PayloadHash: intent.Hash([]byte(`{}`))}
	rec, err := m.Start(context.Background(), TestKind, []byte(`{}`), ctl.Address(), intent.Envelope{Intent: in.JSON(), Payload: []byte(`{}`)})
	if err != nil {
		t.Fatal(err)
	}
	for rec.State == intent.JobWaiting {
		if _, err := m.Advance(context.Background(), rec.ID, rec.Step, rec.Attempt, ctl.Address(), intent.Envelope{}); err != nil {
			t.Fatal(err)
		}
		r.Wait()
		if err := m.Reconcile(context.Background()); err != nil {
			t.Fatal(err)
		}
		res, err := m.Read(rec.ID, 0)
		if err != nil {
			t.Fatal(err)
		}
		rec = res.Job
	}
	var got struct{ Did string }
	if rec.State != intent.JobSucceeded || json.Unmarshal(rec.Result, &got) != nil || got.Did != "two" {
		t.Fatalf("job %+v", rec)
	}
	if units := r.Started(); len(units) != 2 || units[1] != jobs.UnitName(rec.ID, "two", 0) {
		t.Fatalf("units %v", units)
	}
}
