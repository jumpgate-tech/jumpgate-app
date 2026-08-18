package relay

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

var (
	// ErrInsufficientCredits means the account really is out of money. It maps
	// to 402, and it is the moment the product stops giving service away.
	ErrInsufficientCredits = errors.New("relay: account is out of credits")
	// ErrNoAccount is a key bound to an account the store has never seen.
	ErrNoAccount = errors.New("relay: no such account")
)

// defaultCreditBlock is how many credits the relay leases at a time. It trades
// two costs against each other: a small block calls the store often, and a large
// one strands a quiet customer's credits inside a reservation for longer.
const defaultCreditBlock = 256

// CreditStore is the ledger the relay leases from. The Rust billing service
// implements it; the arithmetic there is a single atomic SQL statement, so two
// relays can never reserve the same credits.
type CreditStore interface {
	// Reserve moves up to credits from the account's balance into a
	// reservation. A PARTIAL grant is normal and a zero grant is a valid
	// answer meaning "out of credits" — neither is an error.
	Reserve(ctx context.Context, account string, credits int64) (int64, error)
	// Settle reports how much of a reservation was actually consumed and
	// returns the remainder to the balance.
	Settle(ctx context.Context, account string, spent, reserved int64) error
}

// CreditOptions configures a CreditLease.
type CreditOptions struct {
	// BlockSize is how many credits to lease at a time. Zero takes the default.
	BlockSize int64
}

// creditPool is one account's leased block.
//
// Every field moves under mu, INCLUDING across the store call. Holding the lock
// over the network hop serialises refills for one account, which is the price of
// making the invariant airtight: two goroutines that both saw an empty pool must
// not both spend against a grant only one of them received. Different accounts
// hold different mutexes, so a busy customer never blocks a quiet one, and the
// hot path — a spend that fits in the block — never leaves the mutex.
type creditPool struct {
	mu sync.Mutex
	// available is leased and not yet spent.
	available int64
	// reserved is the total currently held from the store, unsettled.
	reserved int64
	// spent is what has been consumed since the last settle.
	spent int64
}

// CreditLease spends leased credits locally and settles the remainder back.
//
// The relay must NOT cache a balance. A cached balance is money: at high QPS a
// customer at zero keeps spending for the whole cache TTL. A lease has the
// opposite failure mode — the worst case is that a customer's own credits sit
// reserved a little longer than needed, which costs nobody anything.
type CreditLease struct {
	store CreditStore
	block int64

	mu    sync.Mutex
	pools map[string]*creditPool
}

// NewCreditLease builds a lease over store.
func NewCreditLease(store CreditStore, opt CreditOptions) *CreditLease {
	if opt.BlockSize <= 0 {
		opt.BlockSize = defaultCreditBlock
	}
	return &CreditLease{store: store, block: opt.BlockSize, pools: make(map[string]*creditPool)}
}

// poolFor returns an account's pool, creating it on first use.
func (l *CreditLease) poolFor(account string) *creditPool {
	l.mu.Lock()
	defer l.mu.Unlock()
	p, ok := l.pools[account]
	if !ok {
		p = &creditPool{}
		l.pools[account] = p
	}
	return p
}

// Spend consumes credits for one request, leasing more when the block runs out.
//
// It returns ErrInsufficientCredits ONLY when the store confirms the account
// cannot cover the call. A store failure returns that failure instead: charging
// nothing and serving anyway gives the product away, while refusing with the
// wrong reason tells a paying customer to top up an account that is already
// funded.
func (l *CreditLease) Spend(ctx context.Context, account string, credits int64) error {
	if credits <= 0 {
		return nil
	}
	p := l.poolFor(account)

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.available >= credits {
		p.available -= credits
		p.spent += credits
		return nil
	}

	// Lease at least a whole block, and at least enough to cover this call —
	// an expensive method must not be refused merely for costing more than the
	// block size.
	ask := l.block
	if credits > ask {
		ask = credits
	}

	granted, err := l.store.Reserve(ctx, account, ask)
	if err != nil {
		return fmt.Errorf("relay: reserve credits: %w", err)
	}
	p.available += granted
	p.reserved += granted

	if p.available < credits {
		// The store gave what it had and it is not enough. This is the real
		// out-of-credits answer.
		return ErrInsufficientCredits
	}
	p.available -= credits
	p.spent += credits
	return nil
}

// SettleAll reports consumption for every account and returns what was leased
// and not spent. Slice D calls it on a timer; it is safe to call at any time.
func (l *CreditLease) SettleAll(ctx context.Context) error {
	l.mu.Lock()
	accounts := make([]string, 0, len(l.pools))
	for account := range l.pools {
		accounts = append(accounts, account)
	}
	l.mu.Unlock()

	var firstErr error
	for _, account := range accounts {
		if err := l.settle(ctx, account); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// settle flushes one account. The pool is cleared only after the store accepts
// the report, so a failed settle keeps the credits leased rather than losing
// track of them.
func (l *CreditLease) settle(ctx context.Context, account string) error {
	p := l.poolFor(account)

	p.mu.Lock()
	defer p.mu.Unlock()
	if p.reserved == 0 {
		return nil
	}

	spent, reserved := p.spent, p.reserved
	if err := l.store.Settle(ctx, account, spent, reserved); err != nil {
		return fmt.Errorf("relay: settle credits: %w", err)
	}
	p.available = 0
	p.reserved = 0
	p.spent = 0
	return nil
}
