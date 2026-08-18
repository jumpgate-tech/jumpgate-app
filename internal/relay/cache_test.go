package relay

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// The relay caches key records because the store lives in another process. The
// cache is not an optimisation detail — it decides how long a revoked key keeps
// working, and it decides whether a burst of unknown keys can flood billing.
// These tests pin both.

// countingAuth is a real Authenticator, not a mock: it answers from a map and
// records how many times it was asked.
type countingAuth struct {
	mu      sync.Mutex
	records map[string]KeyRecord
	errs    map[string]error
	calls   int
}

func newCountingAuth() *countingAuth {
	return &countingAuth{records: map[string]KeyRecord{}, errs: map[string]error{}}
}

func (c *countingAuth) Authenticate(_ context.Context, rawKey string) (KeyRecord, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.calls++
	if err, ok := c.errs[rawKey]; ok {
		return KeyRecord{}, err
	}
	rec, ok := c.records[rawKey]
	if !ok {
		return KeyRecord{}, ErrUnknownKey
	}
	return rec, nil
}

func (c *countingAuth) count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.calls
}

func (c *countingAuth) set(key string, rec KeyRecord) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.records[key] = rec
}

func (c *countingAuth) remove(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.records, key)
}

func (c *countingAuth) fail(key string, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.errs[key] = err
}

// A fake clock keeps TTL behaviour deterministic. A sleeping test is a flaky
// test, and this cache's whole contract is about time.
type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func (f *fakeClock) Now() time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.now
}

func (f *fakeClock) advance(d time.Duration) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.now = f.now.Add(d)
}

func newTestCache(back Authenticator, clock *fakeClock) *KeyCache {
	return NewKeyCache(back, CacheOptions{TTL: 5 * time.Second, NegativeTTL: time.Second, Now: clock.Now})
}

func TestCacheServesFromMemoryWithinTTL(t *testing.T) {
	back := newCountingAuth()
	back.set("jg_good", KeyRecord{ID: "k1", Enabled: true})
	clock := &fakeClock{now: time.Unix(1000, 0)}
	c := newTestCache(back, clock)

	for i := 0; i < 5; i++ {
		rec, err := c.Authenticate(context.Background(), "jg_good")
		if err != nil {
			t.Fatalf("call %d: %v", i, err)
		}
		if rec.ID != "k1" {
			t.Fatalf("call %d: id = %q, want k1", i, rec.ID)
		}
	}
	if got := back.count(); got != 1 {
		t.Errorf("backend calls = %d, want 1 — the cache is not holding the record", got)
	}
}

// Revocation is eventually consistent, and the TTL is the bound. This test is
// the contract the operator is promised: a revoked key stops working within TTL.
func TestCacheSeesRevocationAfterTTL(t *testing.T) {
	back := newCountingAuth()
	back.set("jg_good", KeyRecord{ID: "k1", Enabled: true})
	clock := &fakeClock{now: time.Unix(1000, 0)}
	c := newTestCache(back, clock)

	if _, err := c.Authenticate(context.Background(), "jg_good"); err != nil {
		t.Fatalf("first: %v", err)
	}
	back.remove("jg_good")

	// Still cached, so still valid. This is the honest cost of the fork.
	if _, err := c.Authenticate(context.Background(), "jg_good"); err != nil {
		t.Fatalf("within TTL the record is still served: %v", err)
	}

	clock.advance(6 * time.Second)
	if _, err := c.Authenticate(context.Background(), "jg_good"); !errors.Is(err, ErrUnknownKey) {
		t.Fatalf("after TTL err = %v, want ErrUnknownKey", err)
	}
}

// An attacker who sends random keys must not turn the relay into a load
// generator against billing. Unknown keys cache too, for a shorter time.
func TestCacheRemembersUnknownKeys(t *testing.T) {
	back := newCountingAuth()
	clock := &fakeClock{now: time.Unix(1000, 0)}
	c := newTestCache(back, clock)

	for i := 0; i < 10; i++ {
		if _, err := c.Authenticate(context.Background(), "jg_nope"); !errors.Is(err, ErrUnknownKey) {
			t.Fatalf("call %d: err = %v, want ErrUnknownKey", i, err)
		}
	}
	if got := back.count(); got != 1 {
		t.Errorf("backend calls = %d, want 1 — a negative answer is not cached", got)
	}
}

// A negative answer expires sooner than a positive one, so a newly issued key
// starts working quickly.
func TestCacheNegativeExpiresSooner(t *testing.T) {
	back := newCountingAuth()
	clock := &fakeClock{now: time.Unix(1000, 0)}
	c := newTestCache(back, clock)

	if _, err := c.Authenticate(context.Background(), "jg_new"); !errors.Is(err, ErrUnknownKey) {
		t.Fatalf("err = %v, want ErrUnknownKey", err)
	}
	back.set("jg_new", KeyRecord{ID: "k9", Enabled: true})

	clock.advance(2 * time.Second) // past NegativeTTL, inside TTL
	rec, err := c.Authenticate(context.Background(), "jg_new")
	if err != nil {
		t.Fatalf("after NegativeTTL: %v", err)
	}
	if rec.ID != "k9" {
		t.Errorf("id = %q, want k9", rec.ID)
	}
}

// Fail closed. If billing is unreachable the relay rejects traffic; it never
// falls open. An outage that silently disabled metering would be worse than an
// outage that stops it.
func TestCacheFailsClosedWhenBackendIsDown(t *testing.T) {
	back := newCountingAuth()
	down := errors.New("dial unix: connection refused")
	back.fail("jg_good", down)
	clock := &fakeClock{now: time.Unix(1000, 0)}
	c := newTestCache(back, clock)

	_, err := c.Authenticate(context.Background(), "jg_good")
	if err == nil {
		t.Fatal("err = nil, want a failure — the relay must not fall open")
	}
	if errors.Is(err, ErrUnknownKey) {
		t.Fatal("a backend outage must not be reported as an unknown key")
	}
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("err = %v, want ErrUnavailable", err)
	}
}

// An outage must not be cached, or one blip would lock every customer out for
// the whole TTL.
func TestCacheDoesNotCacheAnOutage(t *testing.T) {
	back := newCountingAuth()
	back.fail("jg_good", errors.New("connection refused"))
	clock := &fakeClock{now: time.Unix(1000, 0)}
	c := newTestCache(back, clock)

	if _, err := c.Authenticate(context.Background(), "jg_good"); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("err = %v, want ErrUnavailable", err)
	}

	back.mu.Lock()
	delete(back.errs, "jg_good")
	back.mu.Unlock()
	back.set("jg_good", KeyRecord{ID: "k1", Enabled: true})

	rec, err := c.Authenticate(context.Background(), "jg_good")
	if err != nil {
		t.Fatalf("after recovery: %v", err)
	}
	if rec.ID != "k1" {
		t.Errorf("id = %q, want k1", rec.ID)
	}
}

// A disabled key is a different fact from an unknown one, and the two map to
// different statuses. The cache must not flatten them together.
func TestCacheKeepsDisabledDistinctFromUnknown(t *testing.T) {
	back := newCountingAuth()
	back.set("jg_off", KeyRecord{ID: "k2", Enabled: false})
	clock := &fakeClock{now: time.Unix(1000, 0)}
	c := newTestCache(back, clock)

	rec, err := c.Authenticate(context.Background(), "jg_off")
	if err != nil {
		t.Fatalf("Authenticate: %v", err)
	}
	if rec.Enabled {
		t.Error("Enabled = true, want false")
	}
}

// The cache must not grow without bound. An attacker who sends a million
// distinct random keys would otherwise buy a million negative entries, and the
// relay would run out of memory rather than reject him.
func TestCacheEvictsWhenOverCapacity(t *testing.T) {
	back := newCountingAuth()
	clock := &fakeClock{now: time.Unix(1000, 0)}
	c := NewKeyCache(back, CacheOptions{
		TTL:         5 * time.Second,
		NegativeTTL: time.Second,
		MaxEntries:  64,
		Now:         clock.Now,
	})

	for i := 0; i < 5000; i++ {
		_, _ = c.Authenticate(context.Background(), "jg_junk"+string(rune('a'+i%26))+time.Duration(i).String())
	}
	if got := c.Len(); got > 64 {
		t.Errorf("cache holds %d entries, want at most 64", got)
	}
}

// The cache must never hold a raw key. A memory dump of the relay would
// otherwise hand over every live credential it has seen.
func TestCacheDoesNotStoreRawKeys(t *testing.T) {
	back := newCountingAuth()
	back.set("jg_verysecret", KeyRecord{ID: "k1", Enabled: true})
	clock := &fakeClock{now: time.Unix(1000, 0)}
	c := newTestCache(back, clock)

	if _, err := c.Authenticate(context.Background(), "jg_verysecret"); err != nil {
		t.Fatalf("Authenticate: %v", err)
	}
	for _, k := range c.entryKeys() {
		if contains(k, "jg_verysecret") {
			t.Fatalf("cache key %q holds the raw key", k)
		}
	}
}

// A revoked key is a verdict, not an outage, so it caches like one. Otherwise a
// customer whose key was just revoked would hammer billing on every retry —
// exactly the caller least entitled to that much of the store's attention.
func TestCacheRemembersDisabledVerdict(t *testing.T) {
	back := newCountingAuth()
	back.fail("jg_revoked", ErrDisabledKey)
	clock := &fakeClock{now: time.Unix(1000, 0)}
	c := newTestCache(back, clock)

	for i := 0; i < 10; i++ {
		if _, err := c.Authenticate(context.Background(), "jg_revoked"); !errors.Is(err, ErrDisabledKey) {
			t.Fatalf("call %d: err = %v, want ErrDisabledKey", i, err)
		}
	}
	if got := back.count(); got != 1 {
		t.Errorf("backend calls = %d, want 1 — a revoked verdict is not cached", got)
	}
}

// A cold burst on one key must collapse into a single backend call. Without
// this, a popular key's first second floods billing with identical questions.
func TestCacheCollapsesConcurrentMisses(t *testing.T) {
	back := newCountingAuth()
	back.set("jg_hot", KeyRecord{ID: "k3", Enabled: true})
	clock := &fakeClock{now: time.Unix(1000, 0)}
	c := newTestCache(back, clock)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := c.Authenticate(context.Background(), "jg_hot"); err != nil {
				t.Errorf("Authenticate: %v", err)
			}
		}()
	}
	wg.Wait()

	if got := back.count(); got != 1 {
		t.Errorf("backend calls = %d, want 1 — concurrent misses are not collapsing", got)
	}
}
