package agent

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/valve-tech/jumpgate/internal/eip712"
	"github.com/valve-tech/jumpgate/internal/intent"
)

func addr(b byte) eip712.Address { var a eip712.Address; a[19] = b; return a }

func TestAuthoriseNeedsTheControllerToBeAnEnrolledSigner(t *testing.T) {
	p := Policy{Signers: []SignerEntry{{Address: addr(1).Hex(), Tier: TierRoutine, Label: "laptop"}}}

	if err := p.Authorise(intent.KindStatusRead, addr(1), []eip712.Address{addr(1)}); err != nil {
		t.Fatalf("enrolled controller refused: %v", err)
	}
	// Signed by someone else while claiming to be the controller.
	var r *Reject
	if err := p.Authorise(intent.KindStatusRead, addr(1), []eip712.Address{addr(2)}); !errors.As(err, &r) || r.Code != intent.ReasonUnauthorizedKind {
		t.Fatalf("foreign signature: err = %v", err)
	}
	// An unenrolled controller signing for itself.
	if err := p.Authorise(intent.KindStatusRead, addr(2), []eip712.Address{addr(2)}); !errors.As(err, &r) {
		t.Fatalf("unenrolled controller: err = %v", err)
	}
}

func TestAuthoriseUnknownKind(t *testing.T) {
	p := Policy{Signers: []SignerEntry{{Address: addr(1).Hex(), Tier: TierRoutine}}}
	var r *Reject
	if err := p.Authorise("wipe.everything", addr(1), []eip712.Address{addr(1)}); !errors.As(err, &r) || r.Code != intent.ReasonUnknownKind {
		t.Fatalf("err = %v, want unknown_kind", err)
	}
}

// An approval-tier requirement is met only by a distinct signer holding that
// tier; the routine controller's own signature cannot count twice.
func TestAuthoriseApprovalTierNeedsASecondSigner(t *testing.T) {
	p := Policy{
		Signers: []SignerEntry{{Address: addr(1).Hex(), Tier: TierRoutine}, {Address: addr(9).Hex(), Tier: TierApproval}},
		Kinds:   map[string][]Tier{intent.KindServiceAction: {TierRoutine, TierApproval}},
	}
	if err := p.Authorise(intent.KindServiceAction, addr(1), []eip712.Address{addr(1)}); err == nil {
		t.Fatal("approval-tier kind passed with only the routine signature")
	}
	if err := p.Authorise(intent.KindServiceAction, addr(1), []eip712.Address{addr(1), addr(9)}); err != nil {
		t.Fatalf("both tiers present: %v", err)
	}
}

func TestPolicyAddSignerIsIdempotentAndSaveRoundTrips(t *testing.T) {
	var p Policy
	if !p.AddSigner(SignerEntry{Address: addr(1).Hex(), Tier: TierRoutine, Label: "a"}) {
		t.Fatal("first add reported no change")
	}
	if p.AddSigner(SignerEntry{Address: addr(1).Hex(), Tier: TierRoutine, Label: "a"}) {
		t.Fatal("second add of the same signer reported a change")
	}
	p.AddSigner(SignerEntry{Address: addr(2).Hex(), Tier: TierRoutine, Label: "b"})
	path := filepath.Join(t.TempDir(), "policy.json")
	if err := p.Save(path); err != nil {
		t.Fatal(err)
	}
	back, err := LoadPolicy(path)
	if err != nil || len(back.Signers) != 2 {
		t.Fatalf("LoadPolicy = %+v, %v", back, err)
	}
}
