package agent

import (
	"context"
	"encoding/json"
	"sync"
	"testing"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"

	"github.com/valve-tech/jumpgate/internal/executor"
	"github.com/valve-tech/jumpgate/internal/intent"
	"github.com/valve-tech/jumpgate/internal/signer"
	"github.com/valve-tech/jumpgate/internal/testutil"
)

// envelope builds and signs one intent like rig.send, but hands back the
// envelope so a test can tamper with its signatures or send it later.
func (r *rig) envelope(t *testing.T, kind string, payload any) intent.Envelope {
	t.Helper()
	body, _ := json.Marshal(payload)
	r.seq++
	n, _ := intent.NewNonce()
	i := intent.Intent{Agent: r.agentKey.Address(), Controller: r.controller.Address(), Seq: r.seq, Nonce: n,
		IssuedAt: uint64(r.now.Unix()), Expiry: uint64(r.now.Unix() + 120), Kind: kind, PayloadHash: intent.Hash(body)}
	sig, err := r.controller.SignTypedData(context.Background(), i.TypedData())
	if err != nil {
		t.Fatal(err)
	}
	return intent.Envelope{Intent: i.JSON(), Payload: body, Sigs: []string{sig.Hex()}}
}

// verified checks the receipt is the agent's and covers its result.
func (r *rig) verified(t *testing.T, out intent.ReceiptEnvelope) (intent.Receipt, []byte) {
	t.Helper()
	rc, err := out.Receipt.Parse()
	if err != nil {
		t.Fatal(err)
	}
	s, _ := signer.ParseSignature(out.Sig)
	if who, err := signer.Recover(rc.TypedData(), s); err != nil || who != r.agentKey.Address() {
		t.Fatalf("receipt not signed by the agent: %v", err)
	}
	if rc.ResultHash != intent.Hash(out.Result) {
		t.Fatal("receipt does not cover its result")
	}
	return rc, out.Result
}

// M8: a signature that does not parse or recover is bad_signature, in a
// signed receipt, and consumes nothing: the same seq then succeeds.
func TestBadSignaturesAreRejectedAtHandle(t *testing.T) {
	testutil.RequireUnix(t) // the agent is Linux-only: its replay and policy persistence fsyncs directories
	highS := func(valid string) string {
		sig, _ := signer.ParseSignature(valid)
		var s secp256k1.ModNScalar
		s.SetByteSlice(sig[32:64])
		s.Negate()
		b := s.Bytes()
		copy(sig[32:64], b[:])
		sig[64] ^= 1 // the twin signature recovers with the other parity
		return sig.Hex()
	}
	badV := func(valid string) string {
		sig, _ := signer.ParseSignature(valid)
		sig[64] = 29
		return sig.Hex()
	}
	cases := []struct {
		name   string
		tamper func(valid string) string
	}{
		{"not hex", func(string) string { return "0xzz" }},
		{"too short", func(v string) string { return v[:len(v)-2] }},
		{"v not 27 or 28", badV},
		{"high s", highS},
	}
	for _, c := range cases {
		r := newRig(t, true)
		env := r.envelope(t, intent.KindAgentInfo, struct{}{})
		good := env.Sigs[0]
		env.Sigs = []string{c.tamper(good)}
		rc, res := r.verified(t, r.a.Handle(context.Background(), env))
		if got := rejection(t, rc, res).Code; got != intent.ReasonBadSignature {
			t.Errorf("%s: code %s, want bad_signature", c.name, got)
			continue
		}
		env.Sigs = []string{good}
		if rc, res := r.verified(t, r.a.Handle(context.Background(), env)); rc.Status != intent.StatusOK {
			t.Errorf("%s: the untampered intent was refused afterwards: %s", c.name, res)
		}
	}
}

// blockingExec holds every command until release is closed, and reports the
// first one on started.
type blockingExec struct {
	fakeExec
	once    sync.Once
	started chan struct{}
	release chan struct{}
}

func (b *blockingExec) Run(ctx context.Context, cmd string, o *executor.RunOpts) (executor.Result, error) {
	b.once.Do(func() { close(b.started) })
	<-b.release
	return b.fakeExec.Run(ctx, cmd, o)
}

// M8: while one intent from a controller runs, a second from the same
// controller is rejected busy, signed, without consuming its seq.
func TestAConcurrentIntentFromTheSameControllerIsBusy(t *testing.T) {
	testutil.RequireUnix(t) // the agent is Linux-only: its replay and policy persistence fsyncs directories
	r := newRig(t, true)
	be := &blockingExec{fakeExec: fakeExec{res: executor.Result{Stdout: "active\nactive\n"}}, started: make(chan struct{}), release: make(chan struct{})}
	r.a.cfg.Exec = be

	first := r.envelope(t, intent.KindStatusRead, struct{}{})
	second := r.envelope(t, intent.KindAgentInfo, struct{}{})
	done := make(chan intent.ReceiptEnvelope, 1)
	go func() { done <- r.a.Handle(context.Background(), first) }()
	<-be.started

	rc, res := r.verified(t, r.a.Handle(context.Background(), second))
	if got := rejection(t, rc, res).Code; got != intent.ReasonBusy {
		t.Fatalf("code %s, want busy", got)
	}
	close(be.release)
	if rc, res := r.verified(t, <-done); rc.Status != intent.StatusOK {
		t.Fatalf("the running intent failed: %s", res)
	}
	// busy is decided before admission, so the refused seq is still usable.
	if rc, res := r.verified(t, r.a.Handle(context.Background(), second)); rc.Status != intent.StatusOK {
		t.Fatalf("the busy intent's seq was consumed: %s", res)
	}
}
