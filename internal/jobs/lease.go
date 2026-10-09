package jobs

import (
	"fmt"
	"time"

	"github.com/valve-tech/jumpgate/internal/eip712"
	"github.com/valve-tech/jumpgate/internal/intent"
)

// LeaseTTL is how long a lease stays fresh without a heartbeat (D-J2): three
// missed 20-second renewals.
const LeaseTTL = 90 * time.Second

// LeaseFresh is judged by the agent's clock. A heartbeat in the future (the
// clock stepped back) is fresh: losing a lease to a clock step would hand a
// running job to whoever asks next.
func (m Manifest) LeaseFresh(now time.Time) bool {
	if m.Lease.Owner == "" {
		return false
	}
	return now.Unix()-m.Lease.HeartbeatAt < int64(LeaseTTL/time.Second)
}

func (m Manifest) ownedBy(who eip712.Address) bool {
	a, err := eip712.ParseAddress(m.Lease.Owner)
	return err == nil && a == who
}

// Holds is nil when who holds the lease. A holder whose lease went stale but
// that nobody took still holds it: it may carry on.
func (m Manifest) Holds(who eip712.Address) error {
	if m.ownedBy(who) {
		return nil
	}
	if m.Lease.Owner == "" {
		return refuse(intent.ReasonLeaseHeld, fmt.Sprintf("nobody holds job %s; take its lease first", m.ID))
	}
	return refuse(intent.ReasonLeaseHeld, fmt.Sprintf("controller %s holds job %s", m.Lease.Owner, m.ID))
}

// TakeLease gives who the lease if it is free, stale or already who's.
// changed is false for a retake by the owner (a renewal, not logged: Q7).
func (m *Manifest) TakeLease(who eip712.Address, now time.Time) (changed bool, prev string, err error) {
	if m.ownedBy(who) {
		m.Lease.HeartbeatAt = now.Unix()
		return false, "", nil
	}
	if m.LeaseFresh(now) {
		age := now.Unix() - m.Lease.HeartbeatAt
		return false, "", refuse(intent.ReasonLeaseHeld, fmt.Sprintf(
			"controller %s holds job %s (heartbeat %ds ago); watch it, or adopt it once it has been silent for %ds",
			m.Lease.Owner, m.ID, age, int(LeaseTTL/time.Second)))
	}
	prev = m.Lease.Owner
	m.Lease = intent.JobLease{Owner: who.Hex(), HeartbeatAt: now.Unix(), TTLSeconds: int(LeaseTTL / time.Second)}
	return true, prev, nil
}

func (m *Manifest) RenewLease(who eip712.Address, now time.Time) error {
	if err := m.Holds(who); err != nil {
		return err
	}
	m.Lease.HeartbeatAt = now.Unix()
	return nil
}

func (m *Manifest) ReleaseLease(who eip712.Address) error {
	if err := m.Holds(who); err != nil {
		return err
	}
	m.Lease = intent.JobLease{}
	return nil
}
