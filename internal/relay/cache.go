package relay

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"sync"
	"time"
)

var (
	// ErrUnknownKey is a key the store has never issued. It maps to 401.
	ErrUnknownKey = errors.New("relay: unknown key")
	// ErrDisabledKey is a key the store issued and then revoked, or one whose
	// credits ran out. It maps to 403, because the caller's credential is real
	// and the answer is still no.
	ErrDisabledKey = errors.New("relay: key disabled")
	// ErrUnavailable is the store itself failing to answer. It maps to 503 and
	// it must never be confused with a verdict about the key. A relay that
	// reported an outage as "unknown key" would teach a customer to rotate a
	// perfectly good credential.
	ErrUnavailable = errors.New("relay: key store unavailable")
)

// KeyRecord is what the store knows about one key. It never carries the raw key
// or its hash — the relay holds a secret only for as long as one request needs.
type KeyRecord struct {
	ID             string   `json:"id"`
	Label          string   `json:"label"`
	Enabled        bool     `json:"enabled"`
	CreditExempt   bool     `json:"credit_exempt"`
	AllowTrace     bool     `json:"allow_trace"`
	MethodAllow    []string `json:"method_allow"`
	MethodBlock    []string `json:"method_block"`
	Origins        []string `json:"origins"`
	Networks       []string `json:"networks"`
	IPAllow        []string `json:"ip_allow"`
	IPDeny         []string `json:"ip_deny"`
	RateUnlimited  bool     `json:"rate_unlimited"`
	PerSecondLimit int      `json:"per_second_limit"`
	PerDayLimit    int      `json:"per_day_limit"`
	AccountAddress string   `json:"account_address"`
}

// Authenticator resolves a raw key to its record. The billing client implements
// it over a unix socket; the cache wraps another one.
type Authenticator interface {
	Authenticate(ctx context.Context, rawKey string) (KeyRecord, error)
}

// Cache defaults. The TTL is the promise an operator is given: a revoked key
// stops working within this long. Keep it short.
const (
	defaultCacheTTL      = 5 * time.Second
	defaultNegativeTTL   = 1 * time.Second
	defaultMaxCacheItems = 100_000
)

// CacheOptions configures a KeyCache.
type CacheOptions struct {
	// TTL bounds how long a positive record is served from memory, and so how
	// long a revoked key keeps working.
	TTL time.Duration
	// NegativeTTL bounds an unknown-key answer. It is shorter than TTL so a
	// newly issued key starts working quickly, and non-zero so a flood of
	// random keys cannot turn the relay into a load generator against billing.
	NegativeTTL time.Duration
	// MaxEntries caps the map. Without a cap, distinct random keys are a memory
	// exhaustion vector.
	MaxEntries int
	// Now is injectable so tests do not sleep.
	Now func() time.Time
}

// cacheEntry is one remembered answer. A negative answer stores err and leaves
// rec zero.
type cacheEntry struct {
	rec     KeyRecord
	err     error
	expires time.Time
}

// call is one in-flight backend lookup. Concurrent askers for the same key wait
// on it rather than each starting their own.
type call struct {
	wg  sync.WaitGroup
	rec KeyRecord
	err error
}

// KeyCache holds key records in memory for a short time.
//
// The store lives in another process, so this cache is not a speed tweak. It
// decides how long a revoked key keeps working, and it stands between a hostile
// caller and the billing service.
type KeyCache struct {
	back Authenticator
	opt  CacheOptions

	mu       sync.Mutex
	entries  map[string]cacheEntry
	inflight map[string]*call
}

// NewKeyCache wraps back with an in-memory cache.
func NewKeyCache(back Authenticator, opt CacheOptions) *KeyCache {
	if opt.TTL <= 0 {
		opt.TTL = defaultCacheTTL
	}
	if opt.NegativeTTL <= 0 {
		opt.NegativeTTL = defaultNegativeTTL
	}
	if opt.MaxEntries <= 0 {
		opt.MaxEntries = defaultMaxCacheItems
	}
	if opt.Now == nil {
		opt.Now = time.Now
	}
	return &KeyCache{
		back:     back,
		opt:      opt,
		entries:  make(map[string]cacheEntry),
		inflight: make(map[string]*call),
	}
}

// Authenticate resolves a raw key, from memory when it can.
func (c *KeyCache) Authenticate(ctx context.Context, rawKey string) (KeyRecord, error) {
	id := cacheKey(rawKey)
	now := c.opt.Now()

	c.mu.Lock()
	if e, ok := c.entries[id]; ok && now.Before(e.expires) {
		c.mu.Unlock()
		return e.rec, e.err
	}
	// A lookup for this key may already be running. Wait for it instead of
	// asking the same question again.
	if inflight, ok := c.inflight[id]; ok {
		c.mu.Unlock()
		inflight.wg.Wait()
		return inflight.rec, inflight.err
	}
	inflight := &call{}
	inflight.wg.Add(1)
	c.inflight[id] = inflight
	c.mu.Unlock()

	rec, err := c.back.Authenticate(ctx, rawKey)

	// Classify before caching. A verdict about the key — valid, unknown, or
	// revoked — is cacheable. An outage is not a verdict, so it is neither
	// cached nor reported as one: caching a blip would lock every customer out
	// for the whole TTL, and reporting it as "unknown key" would teach a
	// customer to rotate a perfectly good credential.
	var store bool
	var ttl time.Duration
	switch {
	case err == nil:
		store, ttl = true, c.opt.TTL
	case errors.Is(err, ErrUnknownKey), errors.Is(err, ErrDisabledKey):
		store, ttl = true, c.opt.NegativeTTL
	default:
		err = errors.Join(ErrUnavailable, err)
	}

	inflight.rec, inflight.err = rec, err

	c.mu.Lock()
	if store {
		c.evictIfFullLocked(now)
		c.entries[id] = cacheEntry{rec: rec, err: err, expires: now.Add(ttl)}
	}
	delete(c.inflight, id)
	c.mu.Unlock()

	inflight.wg.Done()
	return rec, err
}

// evictIfFullLocked keeps the map under the cap. It drops expired entries
// first, then arbitrary ones. Go randomises map iteration, which is a good
// enough eviction order here: every entry is cheap to recompute, and the cap
// exists to bound memory rather than to maximise a hit rate.
func (c *KeyCache) evictIfFullLocked(now time.Time) {
	if len(c.entries) < c.opt.MaxEntries {
		return
	}
	for k, e := range c.entries {
		if !now.Before(e.expires) {
			delete(c.entries, k)
		}
	}
	for k := range c.entries {
		if len(c.entries) < c.opt.MaxEntries {
			break
		}
		delete(c.entries, k)
	}
}

// Len reports how many answers the cache holds. Tests and metrics read it.
func (c *KeyCache) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.entries)
}

// entryKeys lists the cache's internal keys. It exists so one test can prove no
// raw key is ever stored.
func (c *KeyCache) entryKeys() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	keys := make([]string, 0, len(c.entries))
	for k := range c.entries {
		keys = append(keys, k)
	}
	return keys
}

// cacheKey hashes the raw key. The cache is a long-lived map in a
// network-facing process, so it holds a digest rather than the credential
// itself: a memory dump then yields nothing a caller could replay.
func cacheKey(rawKey string) string {
	sum := sha256.Sum256([]byte(rawKey))
	return hex.EncodeToString(sum[:])
}
