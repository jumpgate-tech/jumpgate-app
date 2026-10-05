package agent

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/valve-tech/jumpgate/internal/catalog"
	"github.com/valve-tech/jumpgate/internal/eip712"
	"github.com/valve-tech/jumpgate/internal/executor"
	"github.com/valve-tech/jumpgate/internal/intent"
	"github.com/valve-tech/jumpgate/internal/ops"
	"github.com/valve-tech/jumpgate/internal/signer"
	"github.com/valve-tech/jumpgate/internal/testutil"
)

// fakeExec answers every command with a fixed result and records them.
type fakeExec struct {
	cmds []string
	res  executor.Result
}

func (f *fakeExec) Run(_ context.Context, cmd string, _ *executor.RunOpts) (executor.Result, error) {
	f.cmds = append(f.cmds, cmd)
	return f.res, nil
}
func (f *fakeExec) WriteFile(context.Context, string, []byte, os.FileMode) error { return nil }
func (f *fakeExec) ReadFile(context.Context, string) ([]byte, error)             { return nil, nil }
func (f *fakeExec) Close() error                                                 { return nil }

type rig struct {
	a          *Agent
	agentKey   *signer.Key
	controller *signer.Key
	exec       *fakeExec
	now        time.Time
	seq        uint64
	dir        string
}

func newRig(t *testing.T, setUp bool) *rig {
	t.Helper()
	dir := t.TempDir()
	r := &rig{dir: dir, exec: &fakeExec{res: executor.Result{Stdout: "active\nactive\n"}}, now: time.Unix(1_800_000_000, 0)}
	r.agentKey, _ = signer.GenerateKey()
	r.controller, _ = signer.GenerateKey()
	p := Policy{Signers: []SignerEntry{{Address: r.controller.Address().Hex(), Tier: TierRoutine, Label: "test"}}}
	if err := p.Save(filepath.Join(dir, "policy.json")); err != nil {
		t.Fatal(err)
	}
	if err := InitReplay(filepath.Join(dir, "replay.json")); err != nil {
		t.Fatal(err)
	}
	if setUp {
		w := catalog.WireConfig{ChainID: 369, ExecID: "reth", BeaconID: "lighthouse", DataDir: "/var/lib/valve-node-app/369", JWTPath: "/var/lib/valve-node-app/369/jwt.hex"}
		b, _ := json.Marshal(w)
		if err := os.WriteFile(filepath.Join(dir, "node.json"), b, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	r.a = New(Config{
		Key: r.agentKey, Exec: r.exec, Now: func() time.Time { return r.now },
		PolicyPath: filepath.Join(dir, "policy.json"), ReplayPath: filepath.Join(dir, "replay.json"), NodePath: filepath.Join(dir, "node.json"),
	})
	return r
}

// send builds, signs and handles one intent; mutate may alter it before signing.
func (r *rig) send(t *testing.T, kind string, payload any, mutate func(*intent.Intent)) (intent.Receipt, []byte) {
	t.Helper()
	body, _ := json.Marshal(payload)
	r.seq++
	n, _ := intent.NewNonce()
	i := intent.Intent{Agent: r.agentKey.Address(), Controller: r.controller.Address(), Seq: r.seq, Nonce: n,
		IssuedAt: uint64(r.now.Unix()), Expiry: uint64(r.now.Unix() + 120), Kind: kind, PayloadHash: intent.Hash(body)}
	if mutate != nil {
		mutate(&i)
	}
	sig, _ := r.controller.SignTypedData(context.Background(), i.TypedData())
	out := r.a.Handle(context.Background(), intent.Envelope{Intent: i.JSON(), Payload: body, Sigs: []string{sig.Hex()}})

	rc, err := out.Receipt.Parse()
	if err != nil {
		t.Fatal(err)
	}
	s, _ := signer.ParseSignature(out.Sig)
	who, err := signer.Recover(rc.TypedData(), s)
	if err != nil || who != r.agentKey.Address() {
		t.Fatalf("receipt not signed by the agent: %v", err)
	}
	if rc.ResultHash != intent.Hash(out.Result) {
		t.Fatal("receipt does not cover its result")
	}
	return rc, out.Result
}

func rejection(t *testing.T, rc intent.Receipt, result []byte) intent.Rejection {
	t.Helper()
	if rc.Status != intent.StatusRejected {
		t.Fatalf("status = %d, want rejected; result %s", rc.Status, result)
	}
	var rej intent.Rejection
	_ = json.Unmarshal(result, &rej)
	return rej
}

func TestAgentInfoAnswersEvenBeforeSetup(t *testing.T) {
	testutil.RequireUnix(t) // the agent is Linux-only: its replay and policy persistence fsyncs directories
	r := newRig(t, false)
	rc, res := r.send(t, intent.KindAgentInfo, struct{}{}, nil)
	if rc.Status != intent.StatusOK {
		t.Fatalf("status %d: %s", rc.Status, res)
	}
	var info intent.AgentInfo
	_ = json.Unmarshal(res, &info)
	if info.Address != r.agentKey.Address().Hex() || info.SetUp || info.LastSeq != 1 {
		t.Fatalf("info = %+v", info)
	}
}

// Review Focus 3.
func TestNodeKindsBeforeSetupAreNotSetUp(t *testing.T) {
	testutil.RequireUnix(t) // the agent is Linux-only: its replay and policy persistence fsyncs directories
	r := newRig(t, false)
	for _, k := range []string{intent.KindStatusRead, intent.KindDiskRead, intent.KindFirewallRead, intent.KindLogsRead} {
		rc, res := r.send(t, k, struct{}{}, nil)
		if got := rejection(t, rc, res).Code; got != intent.ReasonNotSetUp {
			t.Errorf("%s: code %s, want not_set_up", k, got)
		}
	}
	if len(r.exec.cmds) != 0 {
		t.Fatalf("probed an unset-up box: %v", r.exec.cmds)
	}
}

func TestRejections(t *testing.T) {
	testutil.RequireUnix(t) // the agent is Linux-only: its replay and policy persistence fsyncs directories
	cases := []struct {
		name   string
		mutate func(*intent.Intent)
		want   string
	}{
		{"wrong agent", func(i *intent.Intent) { i.Agent = eip712.Address{1} }, intent.ReasonWrongAgent},
		{"expired", func(i *intent.Intent) { i.IssuedAt -= 1000; i.Expiry -= 1000 }, intent.ReasonExpired},
		{"from the future", func(i *intent.Intent) { i.IssuedAt += 600; i.Expiry += 600 }, intent.ReasonClockSkew},
		{"too long-lived", func(i *intent.Intent) { i.Expiry = i.IssuedAt + 3600 }, intent.ReasonInvalidPayload},
		{"payload swapped", func(i *intent.Intent) { i.PayloadHash = intent.Hash([]byte(`{"n":1}`)) }, intent.ReasonInvalidPayload},
		// Times past int64 must not wrap into an accepted window.
		{"times near uint64 max", func(i *intent.Intent) { i.IssuedAt = math.MaxUint64 - 10; i.Expiry = math.MaxUint64 }, intent.ReasonClockSkew},
		{"times straddling int64 max", func(i *intent.Intent) { i.IssuedAt = math.MaxInt64 - 5; i.Expiry = math.MaxInt64 + 5 }, intent.ReasonClockSkew},
	}
	for _, c := range cases {
		r := newRig(t, true)
		rc, res := r.send(t, intent.KindStatusRead, struct{}{}, c.mutate)
		if got := rejection(t, rc, res).Code; got != c.want {
			t.Errorf("%s: code %s, want %s", c.name, got, c.want)
		}
	}
}

func TestAStrangerIsUnauthorised(t *testing.T) {
	testutil.RequireUnix(t) // the agent is Linux-only: its replay and policy persistence fsyncs directories
	r := newRig(t, true)
	r.controller, _ = signer.GenerateKey() // not enrolled
	rc, res := r.send(t, intent.KindStatusRead, struct{}{}, nil)
	if got := rejection(t, rc, res).Code; got != intent.ReasonUnauthorizedKind {
		t.Fatalf("code %s", got)
	}
}

// The spec's acceptance: a captured intent replayed to the same agent is
// rejected. A rejection must not consume the sequence either.
func TestReplayIsRejectedAndRejectionsDoNotConsumeSeq(t *testing.T) {
	testutil.RequireUnix(t) // the agent is Linux-only: its replay and policy persistence fsyncs directories
	r := newRig(t, true)
	body := []byte("{}")
	n, _ := intent.NewNonce()
	i := intent.Intent{Agent: r.agentKey.Address(), Controller: r.controller.Address(), Seq: 1, Nonce: n,
		IssuedAt: uint64(r.now.Unix()), Expiry: uint64(r.now.Unix() + 120), Kind: intent.KindStatusRead, PayloadHash: intent.Hash(body)}
	sig, _ := r.controller.SignTypedData(context.Background(), i.TypedData())
	env := intent.Envelope{Intent: i.JSON(), Payload: body, Sigs: []string{sig.Hex()}}

	first := r.a.Handle(context.Background(), env)
	if first.Receipt.Status != intent.StatusOK {
		t.Fatalf("first: %s", first.Result)
	}
	again := r.a.Handle(context.Background(), env)
	var rej intent.Rejection
	_ = json.Unmarshal(again.Result, &rej)
	if rej.Code != intent.ReasonStaleSeq || rej.LastSeq != 1 {
		t.Fatalf("replay: %+v", rej)
	}
}

func TestRejectionBeforeAdmitDoesNotAdvanceSeq(t *testing.T) {
	testutil.RequireUnix(t) // the agent is Linux-only: its replay and policy persistence fsyncs directories
	r := newRig(t, true)
	r.send(t, intent.KindStatusRead, struct{}{}, func(i *intent.Intent) { i.Agent = eip712.Address{1} }) // seq 1, rejected
	r.seq = 0
	rc, res := r.send(t, intent.KindStatusRead, struct{}{}, nil) // seq 1 again
	if rc.Status != intent.StatusOK {
		t.Fatalf("seq 1 was consumed by a rejected intent: %s", res)
	}
}

func TestServiceActionRunsOpsAndValidatesInput(t *testing.T) {
	testutil.RequireUnix(t) // the agent is Linux-only: its replay and policy persistence fsyncs directories
	r := newRig(t, true)
	rc, res := r.send(t, intent.KindServiceAction, intent.ServiceActionPayload{Service: "beacon", Action: "restart"}, nil)
	if rc.Status != intent.StatusOK {
		t.Fatalf("status %d: %s", rc.Status, res)
	}
	rc, res = r.send(t, intent.KindServiceAction, intent.ServiceActionPayload{Service: "beacon; rm -rf /", Action: "restart"}, nil)
	if got := rejection(t, rc, res).Code; got != intent.ReasonInvalidPayload {
		t.Fatalf("hostile service name: code %s", got)
	}
}

func TestLogsReadClampsN(t *testing.T) {
	testutil.RequireUnix(t) // the agent is Linux-only: its replay and policy persistence fsyncs directories
	r := newRig(t, true)
	rc, res := r.send(t, intent.KindLogsRead, intent.LogsReadPayload{N: 1_000_000}, nil)
	if rc.Status != intent.StatusOK {
		t.Fatalf("status %d: %s", rc.Status, res)
	}
	units := ops.NodeUnits()
	if len(r.exec.cmds) != len(units) {
		t.Fatalf("ran %d commands, want one journalctl per unit (%d): %v", len(r.exec.cmds), len(units), r.exec.cmds)
	}
	for k, unit := range units {
		if c := r.exec.cmds[k]; !containsAll(c, "journalctl", "-u "+unit+" ", "-n 2000 ") {
			t.Fatalf("command %d = %q, want journalctl for %s clamped to -n 2000", k, c, unit)
		}
	}
}

func containsAll(s string, subs ...string) bool {
	for _, x := range subs {
		if !strings.Contains(s, x) {
			return false
		}
	}
	return true
}

// Review Focus 1, end to end: a broken replay record refuses everything.
func TestCorruptReplayStateRefusesEverything(t *testing.T) {
	testutil.RequireUnix(t) // the agent is Linux-only: its replay and policy persistence fsyncs directories
	r := newRig(t, true)
	if err := os.WriteFile(filepath.Join(r.dir, "replay.json"), []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	r.a = New(Config{Key: r.agentKey, Exec: r.exec, Now: func() time.Time { return r.now },
		PolicyPath: filepath.Join(r.dir, "policy.json"), ReplayPath: filepath.Join(r.dir, "replay.json"), NodePath: filepath.Join(r.dir, "node.json")})
	rc, res := r.send(t, intent.KindStatusRead, struct{}{}, nil)
	if got := rejection(t, rc, res).Code; got != intent.ReasonReplayState {
		t.Fatalf("code %s, want replay_state", got)
	}
}

// A failed persist during admission is a signed refusal, never a dispatch.
func TestAdmissionPersistFailureRefusesAndRunsNothing(t *testing.T) {
	testutil.RequireUnix(t) // the agent is Linux-only: its replay and policy persistence fsyncs directories
	r := newRig(t, true)
	orig := syncDir
	syncDir = func(string) error { return errors.New("injected dir sync failure") }
	rc, res := r.send(t, intent.KindServiceAction, intent.ServiceActionPayload{Service: "beacon", Action: "restart"}, nil)
	syncDir = orig
	if got := rejection(t, rc, res).Code; got != intent.ReasonValidation {
		t.Fatalf("code %s, want validation", got)
	}
	if len(r.exec.cmds) != 0 {
		t.Fatalf("ran commands after a failed admission: %v", r.exec.cmds)
	}
}

// A corrupt record refuses, but once an operator repairs it the running agent
// reloads it without a restart. A still-corrupt file keeps refusing.
func TestRunningAgentReloadsRepairedReplayState(t *testing.T) {
	testutil.RequireUnix(t) // the agent is Linux-only: its replay and policy persistence fsyncs directories
	r := newRig(t, true)
	path := filepath.Join(r.dir, "replay.json")
	if err := os.WriteFile(path, []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	r.a = New(Config{Key: r.agentKey, Exec: r.exec, Now: func() time.Time { return r.now },
		PolicyPath: filepath.Join(r.dir, "policy.json"), ReplayPath: path, NodePath: filepath.Join(r.dir, "node.json")})
	rc, res := r.send(t, intent.KindStatusRead, struct{}{}, nil)
	if got := rejection(t, rc, res).Code; got != intent.ReasonReplayState {
		t.Fatalf("code %s, want replay_state", got)
	}
	rc, res = r.send(t, intent.KindStatusRead, struct{}{}, nil)
	if got := rejection(t, rc, res).Code; got != intent.ReasonReplayState {
		t.Fatalf("still corrupt: code %s, want replay_state", got)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := InitReplay(path); err != nil {
		t.Fatal(err)
	}
	rc, res = r.send(t, intent.KindStatusRead, struct{}{}, nil)
	if rc.Status == intent.StatusRejected {
		t.Fatalf("repaired record not reloaded: %s", res)
	}
}
