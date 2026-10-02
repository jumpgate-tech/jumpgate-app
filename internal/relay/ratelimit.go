package relay

import (
	"sync"
	"time"
)

// Rate limiter defaults.
const (
	// rateSweepInterval is how often the limiter looks for keys it can forget.
	// A sweep walks the whole map, so it runs on a timer rather than per call.
	rateSweepInterval = time.Minute
	// defaultMaxRateKeys caps the map, matching the key cache's cap.
	defaultMaxRateKeys = 100_000
)

// RateLimitOptions configures a RateLimiter.
type RateLimitOptions struct {
	// Now is injectable so tests do not sleep. Nil means time.Now.
	Now func() time.Time
	// MaxKeys caps how many keys the limiter tracks. Zero takes the default.
	MaxKeys int
}

// RateLimiter holds each key to its plan's per-second and per-day limits.
//
// The upstream eRPC is keyless, so nothing past the relay knows which key a
// call belongs to. Without this a 50 req/s plan could send 5,000 req/s for as
// long as its credits lasted, and the credits are the only thing that would
// ever stop it.
//
// All state is in memory and per process. Two relays behind one load balancer
// each grant a key its full limit, and a restart hands every key a fresh day.
// That is a known limit of this design rather than an oversight: the credit
// ledger, which is shared, remains the hard bound on spend, and the rate limits
// exist to keep one key from monopolising one relay.
type RateLimiter struct {
	opt RateLimitOptions

	mu        sync.Mutex
	keys      map[string]*keyRate
	lastSweep time.Time
}

// keyRate is one key's counters.
type keyRate struct {
	// tokens is the per-second bucket. It is fractional so a refill between
	// whole seconds is not rounded away.
	tokens float64
	// refilled is when tokens was last brought up to date.
	refilled time.Time
	// day is the UTC midnight that dayUsed counts from.
	day time.Time
	// dayUsed counts calls served since day.
	dayUsed int
}

// NewRateLimiter builds an empty limiter.
func NewRateLimiter(opt RateLimitOptions) *RateLimiter {
	if opt.Now == nil {
		opt.Now = time.Now
	}
	if opt.MaxKeys <= 0 {
		opt.MaxKeys = defaultMaxRateKeys
	}
	now := opt.Now()
	return &RateLimiter{opt: opt, keys: make(map[string]*keyRate), lastSweep: now}
}

// Allow admits n calls for rec's key, or reports how long until it may retry.
//
// n is the number of calls in the request, so a batch counts the same way
// charge() bills it. A refusal consumes nothing on either axis: the caller is
// told to wait, not punished for asking.
//
// A limit of zero or less means that axis is not limited. A key that should be
// served nothing is disabled instead, and treating zero as "deny" would lock
// out every key whose operator set only one of the two limits.
func (l *RateLimiter) Allow(rec KeyRecord, n int) (time.Duration, bool) {
	if rec.RateUnlimited || (rec.PerSecondLimit <= 0 && rec.PerDayLimit <= 0) {
		return 0, true
	}
	if n < 1 {
		n = 1
	}
	now := l.opt.Now()

	l.mu.Lock()
	defer l.mu.Unlock()

	l.sweepLocked(now)

	k, ok := l.keys[rec.ID]
	if !ok {
		l.makeRoomLocked(now)
		k = &keyRate{tokens: float64(rec.PerSecondLimit), refilled: now, day: utcDay(now)}
		l.keys[rec.ID] = k
	}
	k.catchUp(now, rec.PerSecondLimit)

	// Check both axes before consuming either, so a refusal on one does not
	// spend the other.
	if rec.PerDayLimit > 0 && k.dayUsed+n > rec.PerDayLimit {
		return k.day.Add(24 * time.Hour).Sub(now), false
	}
	if rec.PerSecondLimit > 0 && k.tokens < float64(n) {
		// A batch larger than one second's allowance never fits the bucket.
		// It is refused all the same: the plan does not cover it, and letting
		// it through would make a batch the way around every per-second limit.
		deficit := float64(n) - k.tokens
		wait := time.Duration(deficit / float64(rec.PerSecondLimit) * float64(time.Second))
		if wait > time.Second || wait <= 0 {
			wait = time.Second
		}
		return wait, false
	}

	if rec.PerSecondLimit > 0 {
		k.tokens -= float64(n)
	}
	// Only a key with a daily limit counts its day. Counting the rest would
	// keep them from ever looking idle, and nothing would read the count.
	if rec.PerDayLimit > 0 {
		k.dayUsed += n
	}
	return 0, true
}

// catchUp refills the bucket for the time that has passed and rolls the day
// counter over at UTC midnight. The limit is read from the record on every
// call, so a plan change takes effect as soon as the key cache refreshes.
func (k *keyRate) catchUp(now time.Time, perSecond int) {
	if elapsed := now.Sub(k.refilled); elapsed > 0 {
		k.tokens += elapsed.Seconds() * float64(perSecond)
		k.refilled = now
	}
	if k.tokens > float64(perSecond) {
		k.tokens = float64(perSecond)
	}
	if today := utcDay(now); today.After(k.day) {
		k.day, k.dayUsed = today, 0
	}
}

// idle reports whether forgetting this key would change nothing. That is true
// once its bucket has had a full second to refill and its day counter belongs
// to a day that is over. Evicting on a plain idle timer instead would hand a
// key a fresh daily allowance every time it paused.
func (k *keyRate) idle(now time.Time) bool {
	return now.Sub(k.refilled) >= time.Second &&
		(k.dayUsed == 0 || utcDay(now).After(k.day))
}

// sweepLocked forgets idle keys, at most once per rateSweepInterval.
func (l *RateLimiter) sweepLocked(now time.Time) {
	if now.Sub(l.lastSweep) < rateSweepInterval {
		return
	}
	l.lastSweep = now
	for id, k := range l.keys {
		if k.idle(now) {
			delete(l.keys, id)
		}
	}
}

// makeRoomLocked keeps the map under its cap before a new key is added. It
// drops idle keys first, then arbitrary ones. Dropping a busy key resets its
// counters, which is a real cost, but every entry belongs to an authenticated
// key, so reaching the cap takes more live keys than the cap allows, and an
// unbounded map would be the worse failure.
func (l *RateLimiter) makeRoomLocked(now time.Time) {
	if len(l.keys) < l.opt.MaxKeys {
		return
	}
	for id, k := range l.keys {
		if k.idle(now) {
			delete(l.keys, id)
		}
	}
	for id := range l.keys {
		if len(l.keys) < l.opt.MaxKeys {
			break
		}
		delete(l.keys, id)
	}
}

// Len reports how many keys the limiter tracks. Tests and metrics read it.
func (l *RateLimiter) Len() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.keys)
}

// has reports whether the limiter tracks a key. It exists for tests.
func (l *RateLimiter) has(id string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	_, ok := l.keys[id]
	return ok
}

// utcDay truncates t to the start of its UTC day. A plan's day is UTC so that
// every relay agrees on when it ends, whatever zone the host runs in.
func utcDay(t time.Time) time.Time {
	t = t.UTC()
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}
