package jobs

import (
	"errors"
	"testing"
	"time"

	"github.com/valve-tech/jumpgate/internal/intent"
	"github.com/valve-tech/jumpgate/internal/signer"
)

func TestLeaseTakeRenewRelease(t *testing.T) {
	a, _ := signer.GenerateKey()
	b, _ := signer.GenerateKey()
	now := time.Unix(1_800_000_000, 0)
	var m Manifest

	if changed, prev, err := m.TakeLease(a.Address(), now); err != nil || !changed || prev != "" {
		t.Fatalf("first take: %v %q %v", changed, prev, err)
	}
	// A fresh lease refuses another controller, by the agent's clock.
	var r *Refusal
	if _, _, err := m.TakeLease(b.Address(), now.Add(LeaseTTL-time.Second)); !errors.As(err, &r) || r.Code != intent.ReasonLeaseHeld {
		t.Fatalf("take of a fresh lease = %v", err)
	}
	if err := m.RenewLease(b.Address(), now); err == nil {
		t.Fatal("a controller renewed a lease it does not hold")
	}
	// Taking your own lease again is a renewal: no change to log.
	if changed, _, err := m.TakeLease(a.Address(), now.Add(10*time.Second)); err != nil || changed {
		t.Fatalf("own retake: %v %v", changed, err)
	}
	// Stale: b may take it over, and the previous owner is named.
	if changed, prev, err := m.TakeLease(b.Address(), now.Add(10*time.Second+LeaseTTL+time.Second)); err != nil || !changed || prev != a.Address().Hex() {
		t.Fatalf("takeover: %v %q %v", changed, prev, err)
	}
	if err := m.Holds(a.Address()); err == nil {
		t.Fatal("the old owner still holds the lease")
	}
	if err := m.ReleaseLease(b.Address()); err != nil || m.Lease.Owner != "" {
		t.Fatalf("release: %v %+v", err, m.Lease)
	}
}

// A clock that steps backwards a little must not make a held lease look
// stale.
func TestLeaseIsFreshWhenTheClockStepsBack(t *testing.T) {
	a, _ := signer.GenerateKey()
	now := time.Unix(1_800_000_000, 0)
	var m Manifest
	_, _, _ = m.TakeLease(a.Address(), now)
	if !m.LeaseFresh(now.Add(-LeaseTTL / 2)) {
		t.Fatal("a heartbeat in the near future read as stale")
	}
}

// M2: a clock that jumped forward and back leaves a heartbeat far in the
// future. Taken at face value it would keep the lease fresh for the whole
// jump, and nobody could adopt a job whose controller is gone. A heartbeat
// more than a TTL ahead of the agent's clock is stale.
func TestAFarFutureHeartbeatIsStale(t *testing.T) {
	a, _ := signer.GenerateKey()
	b, _ := signer.GenerateKey()
	now := time.Unix(1_800_000_000, 0)
	var m Manifest
	_, _, _ = m.TakeLease(a.Address(), now.Add(24*time.Hour)) // renewed during the jump
	if m.LeaseFresh(now) {
		t.Fatal("a heartbeat a day ahead read as fresh")
	}
	if changed, prev, err := m.TakeLease(b.Address(), now); err != nil || !changed || prev != a.Address().Hex() {
		t.Fatalf("adoption after the clock came back: %v %q %v", changed, prev, err)
	}
}
