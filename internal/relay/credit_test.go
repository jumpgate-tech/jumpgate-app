package relay

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
)

// The relay must not cache a balance. A stale balance is money: at high QPS a
// customer at zero keeps spending for TTL x QPS requests. So credits are LEASED
// — the relay reserves a block, spends it locally at wire speed, and settles the
// remainder back.
//
// The invariant that matters is simple and absolute: the relay may never spend
// more than the store granted it. Everything below exists to hold that line.

// ledgerStore is a real in-memory ledger, not a mock. It enforces the same rule
// the SQL does — you cannot reserve credits that are not there — so a bug in the
// lease shows up here rather than in production.
type ledgerStore struct {
	mu        sync.Mutex
	remaining map[string]int64
	reserved  map[string]int64
	reserves  atomic.Int64
	settles   atomic.Int64
	failWith  error
}

func newLedger(account string, credits int64) *ledgerStore {
	return &ledgerStore{
		remaining: map[string]int64{account: credits},
		reserved:  map[string]int64{},
	}
}

func (s *ledgerStore) Reserve(_ context.Context, account string, credits int64) (int64, error) {
	s.reserves.Add(1)
	if s.failWith != nil {
		return 0, s.failWith
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	have, ok := s.remaining[account]
	if !ok {
		return 0, ErrNoAccount
	}
	// A partial grant is normal: give what is there rather than failing.
	granted := credits
	if have < granted {
		granted = have
	}
	s.remaining[account] = have - granted
	s.reserved[account] += granted
	return granted, nil
}

func (s *ledgerStore) Settle(_ context.Context, account string, spent, reserved int64) error {
	s.settles.Add(1)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reserved[account] -= reserved
	s.remaining[account] += reserved - spent
	return nil
}

// total is the ledger's conserved quantity. It must never grow.
func (s *ledgerStore) total(account string) int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.remaining[account] + s.reserved[account]
}

const acct = "0xcustomer"

func newLease(store CreditStore, block int64) *CreditLease {
	return NewCreditLease(store, CreditOptions{BlockSize: block})
}

// A spend inside the local block must not touch the store. That is the whole
// point of leasing — the hot path stays off the network.
func TestLeaseSpendsLocallyWithoutCallingTheStore(t *testing.T) {
	store := newLedger(acct, 1000)
	lease := newLease(store, 100)

	for i := 0; i < 50; i++ {
		if err := lease.Spend(context.Background(), acct, 1); err != nil {
			t.Fatalf("spend %d: %v", i, err)
		}
	}
	if got := store.reserves.Load(); got != 1 {
		t.Errorf("store reserves = %d, want 1 — the hot path is hitting the store", got)
	}
}

// Draining the block reserves another.
func TestLeaseRefillsWhenTheBlockRunsOut(t *testing.T) {
	store := newLedger(acct, 1000)
	lease := newLease(store, 10)

	for i := 0; i < 25; i++ {
		if err := lease.Spend(context.Background(), acct, 1); err != nil {
			t.Fatalf("spend %d: %v", i, err)
		}
	}
	if got := store.reserves.Load(); got < 3 {
		t.Errorf("store reserves = %d, want at least 3 for 25 credits in blocks of 10", got)
	}
}

// An account with no credits left is refused. This is the moment the product
// actually stops giving service away.
func TestLeaseRefusesWhenTheAccountIsEmpty(t *testing.T) {
	store := newLedger(acct, 3)
	lease := newLease(store, 10)

	for i := 0; i < 3; i++ {
		if err := lease.Spend(context.Background(), acct, 1); err != nil {
			t.Fatalf("spend %d should succeed: %v", i, err)
		}
	}
	err := lease.Spend(context.Background(), acct, 1)
	if !errors.Is(err, ErrInsufficientCredits) {
		t.Fatalf("err = %v, want ErrInsufficientCredits", err)
	}
}

// A partial grant is spendable. The store gives what it has; the relay uses it.
func TestLeaseHonoursAPartialGrant(t *testing.T) {
	store := newLedger(acct, 4)
	lease := newLease(store, 100)

	spent := 0
	for i := 0; i < 10; i++ {
		if err := lease.Spend(context.Background(), acct, 1); err != nil {
			break
		}
		spent++
	}
	if spent != 4 {
		t.Errorf("spent %d credits, want exactly the 4 the account held", spent)
	}
}

// THE invariant. Under concurrency the relay must never spend more than the
// store granted. Over-spending here is a customer using money that does not
// exist, and it would be invisible until the ledger was reconciled.
func TestLeaseNeverSpendsMoreThanGranted(t *testing.T) {
	const funded = 500
	store := newLedger(acct, funded)
	lease := newLease(store, 32)

	var accepted atomic.Int64
	var wg sync.WaitGroup
	for i := 0; i < 64; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				if err := lease.Spend(context.Background(), acct, 1); err == nil {
					accepted.Add(1)
				}
			}
		}()
	}
	wg.Wait()

	if got := accepted.Load(); got > funded {
		t.Fatalf("accepted %d spends against %d funded credits — the relay spent money that did not exist", got, funded)
	}
	if store.total(acct) > funded {
		t.Fatalf("ledger total grew to %d from %d — credits were created", store.total(acct), funded)
	}
}

// Settling returns what was reserved and not spent, so a quiet customer does not
// have credits stranded in a reservation forever.
func TestLeaseSettleReturnsTheUnspentRemainder(t *testing.T) {
	store := newLedger(acct, 1000)
	lease := newLease(store, 100)

	for i := 0; i < 10; i++ {
		if err := lease.Spend(context.Background(), acct, 1); err != nil {
			t.Fatalf("spend: %v", err)
		}
	}
	if err := lease.SettleAll(context.Background()); err != nil {
		t.Fatalf("SettleAll: %v", err)
	}
	if got := store.total(acct); got != 990 {
		t.Errorf("ledger total = %d, want 990 (1000 funded minus 10 spent)", got)
	}
	if got := store.reserved[acct]; got != 0 {
		t.Errorf("reserved = %d after settle, want 0 — credits are stranded", got)
	}
}

// The conserved quantity survives a full cycle. Reserve, spend, settle, repeat:
// funded minus spent must equal the ledger total, every time.
func TestLeaseArithmeticIsConservedAcrossCycles(t *testing.T) {
	store := newLedger(acct, 1000)
	lease := newLease(store, 16)

	total := 0
	for round := 0; round < 5; round++ {
		for i := 0; i < 20; i++ {
			if err := lease.Spend(context.Background(), acct, 1); err != nil {
				t.Fatalf("round %d spend %d: %v", round, i, err)
			}
			total++
		}
		if err := lease.SettleAll(context.Background()); err != nil {
			t.Fatalf("round %d settle: %v", round, err)
		}
		if got, want := store.total(acct), int64(1000-total); got != want {
			t.Fatalf("round %d: ledger total = %d, want %d", round, got, want)
		}
	}
}

// Two customers must not share a pool. One draining its credits cannot starve
// or subsidise the other.
func TestLeaseKeepsAccountsIndependent(t *testing.T) {
	store := &ledgerStore{
		remaining: map[string]int64{"a": 2, "b": 100},
		reserved:  map[string]int64{},
	}
	lease := newLease(store, 10)

	for i := 0; i < 2; i++ {
		if err := lease.Spend(context.Background(), "a", 1); err != nil {
			t.Fatalf("a spend %d: %v", i, err)
		}
	}
	if err := lease.Spend(context.Background(), "a", 1); !errors.Is(err, ErrInsufficientCredits) {
		t.Fatalf("a: err = %v, want ErrInsufficientCredits", err)
	}
	if err := lease.Spend(context.Background(), "b", 1); err != nil {
		t.Fatalf("b must be unaffected by a running dry: %v", err)
	}
}

// A store outage must not be read as "no credits". Charging nothing and serving
// anyway gives the product away; refusing with the wrong reason tells a paying
// customer to top up an account that is already funded.
func TestLeaseDistinguishesAnOutageFromAnEmptyAccount(t *testing.T) {
	store := newLedger(acct, 1000)
	store.failWith = errors.New("socket is gone")
	lease := newLease(store, 10)

	err := lease.Spend(context.Background(), acct, 1)
	if err == nil {
		t.Fatal("err = nil, want a failure")
	}
	if errors.Is(err, ErrInsufficientCredits) {
		t.Fatal("a store outage must not be reported as an empty account")
	}
}

// A cost larger than one block still works: the lease reserves enough to cover
// it rather than refusing a legitimately expensive call.
func TestLeaseCoversACostBiggerThanOneBlock(t *testing.T) {
	store := newLedger(acct, 1000)
	lease := newLease(store, 10)

	if err := lease.Spend(context.Background(), acct, 250); err != nil {
		t.Fatalf("a 250-credit call must be servable: %v", err)
	}
}
