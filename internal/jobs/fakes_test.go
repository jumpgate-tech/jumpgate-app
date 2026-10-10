package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/valve-tech/jumpgate/internal/eip712"
	"github.com/valve-tech/jumpgate/internal/intent"
	"github.com/valve-tech/jumpgate/internal/signer"
	"github.com/valve-tech/jumpgate/internal/testutil"
)

// stepRunner is an in-package Runner: jobstest imports jobs, so jobs' own
// tests cannot import it. Start runs the step synchronously unless hold is
// set, so a test decides when a unit "finishes".
type stepRunner struct {
	mu      sync.Mutex
	dir     string
	active  map[string]bool
	argv    map[string][]string
	hold    bool
	started []string
	fail    error
	// acceptThenFail is a Start whose unit systemd accepted, then errors:
	// the unit runs, and Start reports a failure (a cancelled request, a
	// broken pipe). onStart runs first, with Start's own context.
	acceptThenFail error
	onStart        func(ctx context.Context)
}

func (r *stepRunner) Supported(context.Context) error { return nil }
func (r *stepRunner) Start(ctx context.Context, unit string, argv []string) error {
	if r.onStart != nil {
		r.onStart(ctx)
	}
	if r.fail != nil {
		return r.fail
	}
	r.mu.Lock()
	r.started = append(r.started, unit)
	r.active[unit], r.argv[unit] = true, argv
	r.mu.Unlock()
	if r.acceptThenFail != nil {
		return r.acceptThenFail
	}
	if !r.hold {
		r.finish(ctx, unit)
	}
	return nil
}
func (r *stepRunner) Active(_ context.Context, unit string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.active[unit], nil
}

func (r *stepRunner) AnyActive(_ context.Context, id string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for u, on := range r.active {
		if on && strings.HasPrefix(u, "jumpgate-job-"+id+"-") {
			return true, nil
		}
	}
	return false, nil
}

// finish runs the unit's step to completion, as the transient unit would.
func (r *stepRunner) finish(ctx context.Context, unit string) {
	r.mu.Lock()
	argv := r.argv[unit]
	r.mu.Unlock()
	a, _ := ParseStepArgs(argv)
	_ = RunStep(ctx, r.dir, a, Env{Out: io.Discard})
	r.mu.Lock()
	delete(r.active, unit)
	r.mu.Unlock()
}

// lose ends a unit without a result: a reboot.
func (r *stepRunner) lose(unit string) {
	r.mu.Lock()
	delete(r.active, unit)
	r.mu.Unlock()
}

// twoSteps is the engine's test plan. Inputs say which step fails or asks.
func twoSteps(t *testing.T) Plan {
	t.Helper()
	type in struct{ FailAt, AskAt, AskTier string }
	step := func(id string) StepDef {
		return StepDef{ID: id, Title: "Step " + id, AbortNote: "nothing after " + id + " ran",
			Run: func(ctx context.Context, env Env) (Outcome, error) {
				var i in
				_ = json.Unmarshal(env.Inputs, &i)
				if i.AskAt == id && env.Answer == nil {
					tier := intent.TierRoutine
					if i.AskTier != "" {
						tier = i.AskTier
					}
					return Outcome{Ask: &intent.JobQuestion{Key: "pick", Prompt: "pick one", Tier: tier,
						Options: []intent.JobOption{{ID: "go", Label: "Go on"}, {ID: intent.OptAbort, Label: "Stop"}}}}, nil
				}
				if i.FailAt == id && env.Answer == nil {
					return Outcome{}, errors.New(id + " broke")
				}
				return Outcome{Result: json.RawMessage(fmt.Sprintf(`{"did":%q}`, id))}, nil
			}}
	}
	p := Plan{Kind: "test.two", Version: "test.two/1", Title: "Two steps", Steps: []StepDef{step("one"), step("two")}}
	Register(p)
	t.Cleanup(func() { Unregister(p.Version) })
	return p
}

type rig struct {
	m    *Manager
	r    *stepRunner
	box  *signer.Key // the agent's key: intents are addressed to it, and it signs the logs
	a, b *signer.Key // two controllers
	now  time.Time
	dir  string
}

func newRig(t *testing.T) *rig {
	t.Helper()
	testutil.RequireUnix(t)
	twoSteps(t)
	dir := filepath.Join(t.TempDir(), "jobs")
	g := &rig{dir: dir, now: time.Unix(1_800_000_000, 0)}
	g.r = &stepRunner{dir: dir, active: map[string]bool{}, argv: map[string][]string{}}
	g.a, _ = signer.GenerateKey()
	g.b, _ = signer.GenerateKey()
	g.box, _ = signer.GenerateKey()
	var err error
	g.m, err = NewManager(ManagerConfig{Dir: dir, Key: g.box, Runner: g.r, Now: func() time.Time { return g.now },
		StepArgv: func(a StepArgs) []string { return StepArgv("/jumpgate", "/state", "/etc/jumpgate", a) }, Version: "v-test"})
	if err != nil {
		t.Fatal(err)
	}
	// The manager may be swapped for a "restarted agent"; close whichever is current.
	t.Cleanup(func() { g.m.Close() })
	return g
}

// sign is the envelope controller k sends: a fresh intent addressed to this
// box, over payload, signed by k. The intent's random nonce gives every start
// its own job id (joblog.JobIDFor).
func (g *rig) sign(t *testing.T, k *signer.Key, kind string, payload any) intent.Envelope {
	t.Helper()
	b, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	nonce, err := intent.NewNonce()
	if err != nil {
		t.Fatal(err)
	}
	in := intent.Intent{Agent: g.box.Address(), Controller: k.Address(), Nonce: nonce,
		IssuedAt: uint64(g.now.Unix()), Expiry: uint64(g.now.Unix() + 600), Kind: kind, PayloadHash: intent.Hash(b)}
	sig, err := k.SignTypedData(context.Background(), in.TypedData())
	if err != nil {
		t.Fatal(err)
	}
	return intent.Envelope{Intent: in.JSON(), Payload: b, Sigs: []string{sig.Hex()}}
}

func (g *rig) start(t *testing.T, k *signer.Key, inputs string) (intent.JobRecord, error) {
	t.Helper()
	return g.m.Start(context.Background(), "test.two", []byte(inputs), k.Address(),
		g.sign(t, k, "test.two", json.RawMessage(inputs)))
}

func (g *rig) advanceAt(t *testing.T, k *signer.Key, id string, step, attempt int) (intent.JobRecord, error) {
	t.Helper()
	return g.m.Advance(context.Background(), id, step, attempt, k.Address(),
		g.sign(t, k, intent.KindJobAdvance, intent.JobAdvancePayload{ID: id, Step: step, Attempt: attempt}))
}

func (g *rig) lease(t *testing.T, k *signer.Key, id, op string) (intent.JobRecord, error) {
	t.Helper()
	return g.m.Lease(context.Background(), id, op, k.Address(),
		g.sign(t, k, intent.KindJobLease, intent.JobLeasePayload{ID: id, Op: op}))
}

func (g *rig) decide(t *testing.T, k *signer.Key, id, decision, option string) (intent.JobRecord, error) {
	t.Helper()
	return g.decideAs(t, k, id, decision, option, routineOnly)
}

// decideAs is decide with the tier check the agent would pass.
func (g *rig) decideAs(t *testing.T, k *signer.Key, id, decision, option string, check TierCheck) (intent.JobRecord, error) {
	t.Helper()
	return g.m.Decide(context.Background(), id, decision, option, k.Address(), []eip712.Address{k.Address()}, check,
		g.sign(t, k, intent.KindJobDecide, intent.JobDecidePayload{ID: id, Decision: decision, Option: option}))
}

// routineOnly is the tier check of a decision signed by one routine key.
func routineOnly(tier string) error {
	if tier != intent.TierRoutine {
		return fmt.Errorf("only a routine signature")
	}
	return nil
}

// advance moves the job one step as the lease holder would, then reconciles.
func (g *rig) advance(t *testing.T, k *signer.Key, id string) intent.JobRecord {
	t.Helper()
	rec, _ := g.read(t, id)
	if _, err := g.advanceAt(t, k, id, rec.Step, rec.Attempt); err != nil {
		t.Fatalf("advance: %v", err)
	}
	if err := g.m.Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	rec, _ = g.read(t, id)
	return rec
}

func (g *rig) read(t *testing.T, id string) (intent.JobRecord, intent.JobReadResult) {
	t.Helper()
	res, err := g.m.Read(id, 0)
	if err != nil {
		t.Fatal(err)
	}
	return res.Job, res
}
