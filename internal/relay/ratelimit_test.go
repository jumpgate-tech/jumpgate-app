package relay

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"testing"
	"time"
)

// The upstream eRPC is keyless, so the relay is the only thing that can hold a
// key to its plan. Before these tests a 50 req/s key could send 5,000 req/s for
// as long as its credits lasted.

// limitedKey is a key with a plan. Zero on either axis means that axis is not
// limited.
func limitedKey(id string, perSecond, perDay int) KeyRecord {
	return KeyRecord{ID: id, Enabled: true, AllowTrace: true, PerSecondLimit: perSecond, PerDayLimit: perDay}
}

// noon is a fixed moment well away from a UTC day boundary.
func noon() *fakeClock {
	return &fakeClock{now: time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)}
}

func newTestLimiter(clock *fakeClock) *RateLimiter {
	return NewRateLimiter(RateLimitOptions{Now: clock.Now})
}

func TestLimiterHoldsAKeyToItsPerSecondLimit(t *testing.T) {
	clock := noon()
	l := newTestLimiter(clock)
	rec := limitedKey("k1", 3, 0)

	for i := 0; i < 3; i++ {
		if wait, ok := l.Allow(rec, 1); !ok {
			t.Fatalf("call %d refused (retry %v), want allowed", i, wait)
		}
	}
	wait, ok := l.Allow(rec, 1)
	if ok {
		t.Fatal("fourth call in one second allowed, want refused")
	}
	if wait <= 0 || wait > time.Second {
		t.Errorf("retry after = %v, want within (0, 1s]", wait)
	}

	clock.advance(time.Second)
	if _, ok := l.Allow(rec, 1); !ok {
		t.Error("call after a full second refused, want the bucket refilled")
	}
}

// A batch costs N, the same as charge() bills it. Counting it as one call would
// turn every limit into a limit on HTTP requests rather than on calls.
func TestLimiterCountsABatchAsItsCalls(t *testing.T) {
	l := newTestLimiter(noon())
	rec := limitedKey("k1", 5, 0)

	if _, ok := l.Allow(rec, 4); !ok {
		t.Fatal("batch of 4 under a limit of 5 refused")
	}
	if _, ok := l.Allow(rec, 2); ok {
		t.Fatal("batch of 2 with 1 call left allowed, want refused")
	}
	if _, ok := l.Allow(rec, 1); !ok {
		t.Error("single call with 1 left refused: the refused batch must not have consumed anything")
	}
}

func TestLimiterHoldsAKeyToItsDailyLimitUntilUTCMidnight(t *testing.T) {
	clock := &fakeClock{now: time.Date(2026, 10, 2, 23, 0, 0, 0, time.UTC)}
	l := newTestLimiter(clock)
	rec := limitedKey("k1", 0, 2)

	for i := 0; i < 2; i++ {
		if _, ok := l.Allow(rec, 1); !ok {
			t.Fatalf("call %d refused, want allowed", i)
		}
	}
	wait, ok := l.Allow(rec, 1)
	if ok {
		t.Fatal("third call of a 2/day key allowed, want refused")
	}
	if wait != time.Hour {
		t.Errorf("retry after = %v, want 1h (until UTC midnight)", wait)
	}

	clock.advance(59 * time.Minute)
	if _, ok := l.Allow(rec, 1); ok {
		t.Fatal("call before midnight allowed, want refused")
	}
	clock.advance(time.Minute)
	if _, ok := l.Allow(rec, 1); !ok {
		t.Error("call after UTC midnight refused, want the day counter reset")
	}
}

// A call one limit refuses must not count against the other, or a throttled
// caller would be throttled twice.
func TestLimiterRefusalConsumesNothing(t *testing.T) {
	clock := noon()
	l := newTestLimiter(clock)
	rec := limitedKey("k1", 2, 3)

	l.Allow(rec, 2)
	if _, ok := l.Allow(rec, 1); ok {
		t.Fatal("per-second limit not applied")
	}
	// Two of the three daily calls are spent. If the refused call had counted
	// toward the day, none would be left.
	clock.advance(time.Second)
	if _, ok := l.Allow(rec, 1); !ok {
		t.Error("third daily call refused: a per-second refusal was counted against the day")
	}
}

func TestLimiterDailyRefusalSpendsNoTokens(t *testing.T) {
	clock := &fakeClock{now: time.Date(2026, 10, 2, 23, 59, 59, 750_000_000, time.UTC)}
	l := newTestLimiter(clock)
	rec := limitedKey("k1", 4, 2)

	l.Allow(rec, 2)
	if _, ok := l.Allow(rec, 2); ok {
		t.Fatal("daily limit not applied")
	}
	// A quarter second refills one token and rolls the day over. Two calls fit
	// only if the refused call above left its two tokens alone.
	clock.advance(250 * time.Millisecond)
	if _, ok := l.Allow(rec, 2); !ok {
		t.Error("a daily refusal spent per-second tokens")
	}
}

func TestLimiterIgnoresAnUnlimitedKey(t *testing.T) {
	l := newTestLimiter(noon())
	rec := limitedKey("k1", 1, 1)
	rec.RateUnlimited = true

	for i := 0; i < 100; i++ {
		if _, ok := l.Allow(rec, 1); !ok {
			t.Fatalf("call %d refused for a rate-unlimited key", i)
		}
	}
	if n := l.Len(); n != 0 {
		t.Errorf("limiter holds %d entries for an unlimited key, want 0", n)
	}
}

func TestLimiterCountsEachKeySeparately(t *testing.T) {
	l := newTestLimiter(noon())

	if _, ok := l.Allow(limitedKey("a", 1, 0), 1); !ok {
		t.Fatal("key a refused")
	}
	if _, ok := l.Allow(limitedKey("b", 1, 0), 1); !ok {
		t.Error("key b refused because key a spent its limit")
	}
}

// An idle key is forgotten once forgetting it changes nothing: its bucket has
// refilled and its day counter belongs to a day that is over.
func TestLimiterForgetsIdleKeys(t *testing.T) {
	clock := noon()
	l := newTestLimiter(clock)

	l.Allow(limitedKey("burst", 5, 0), 1)
	l.Allow(limitedKey("daily", 0, 10), 1)

	clock.advance(rateSweepInterval)
	l.Allow(limitedKey("other", 5, 0), 1)
	if l.has("burst") {
		t.Error("an idle per-second key is still held after its bucket refilled")
	}
	// Forgetting today's counter would hand the key a fresh day.
	if !l.has("daily") {
		t.Fatal("a key with calls counted today was evicted: that resets its daily limit")
	}

	clock.advance(24 * time.Hour)
	l.Allow(limitedKey("other", 5, 0), 1)
	if l.has("daily") {
		t.Error("a key whose counter belongs to yesterday is still held")
	}
}

// The map has a hard ceiling. Every entry comes from an authenticated key, so
// reaching it takes real keys, but the bound must not depend on that.
func TestLimiterStaysUnderItsCap(t *testing.T) {
	l := NewRateLimiter(RateLimitOptions{Now: noon().Now, MaxKeys: 10})
	for i := 0; i < 50; i++ {
		l.Allow(limitedKey(fmt.Sprintf("k%d", i), 0, 100), 1)
	}
	if n := l.Len(); n > 10 {
		t.Errorf("limiter holds %d keys, want at most 10", n)
	}
}

// --- through the handler -----------------------------------------------------

func limitedHandler(t *testing.T, rec KeyRecord, clock *fakeClock, credits *CreditLease, got *capturedRequest) *Handler {
	t.Helper()
	up := stubUpstream(t, got)
	beacon := stubUpstream(t, got)
	h, err := NewHandler(Config{
		Auth:      staticAuth{rec: rec},
		ProjectID: "main",
		ERPC:      up,
		Beacon:    func(int) (*url.URL, bool) { return beacon, true },
		Credits:   credits,
		Limiter:   newTestLimiter(clock),
		Caller:    &stubCaller{},
		Streams:   newStubStreams(),
	})
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}
	return h
}

func TestHandlerAnswers429WithRetryAfter(t *testing.T) {
	var got capturedRequest
	h := limitedHandler(t, limitedKey("k1", 1, 0), noon(), nil, &got)

	if res := post(t, h, "/rpc/jg_k/evm/369", blockNumber, nil); res.Code != http.StatusOK {
		t.Fatalf("first call = %d, want 200", res.Code)
	}
	res := post(t, h, "/rpc/jg_k/evm/369", blockNumber, nil)
	if res.Code != http.StatusTooManyRequests {
		t.Fatalf("second call = %d, want 429", res.Code)
	}
	secs, err := strconv.Atoi(res.Header().Get("Retry-After"))
	if err != nil || secs < 1 {
		t.Errorf("Retry-After = %q, want a whole number of seconds >= 1", res.Header().Get("Retry-After"))
	}
	if got.hits != 1 {
		t.Errorf("upstream hits = %d, want 1: a throttled call must not be forwarded", got.hits)
	}
}

func TestHandlerCountsABatchAgainstTheLimit(t *testing.T) {
	var got capturedRequest
	h := limitedHandler(t, limitedKey("k1", 2, 0), noon(), nil, &got)

	batch := `[{"jsonrpc":"2.0","id":1,"method":"eth_blockNumber"},{"jsonrpc":"2.0","id":2,"method":"eth_chainId"},{"jsonrpc":"2.0","id":3,"method":"eth_chainId"}]`
	if res := post(t, h, "/rpc/jg_k/evm/369", batch, nil); res.Code != http.StatusTooManyRequests {
		t.Fatalf("3-call batch on a 2/s key = %d, want 429", res.Code)
	}
}

// The limit applies before the charge. Otherwise a throttled call would still
// cost the customer credits for a call nobody served.
func TestThrottledCallCostsNothing(t *testing.T) {
	clock := noon()
	rec := limitedKey("k1", 1, 0)
	rec.AccountAddress = acct
	var got capturedRequest
	// Two credits: enough for exactly two served calls.
	h := limitedHandler(t, rec, clock, newLease(newLedger(acct, 2), 1), &got)

	if res := post(t, h, "/rpc/jg_k/evm/369", blockNumber, nil); res.Code != http.StatusOK {
		t.Fatalf("first call = %d, want 200", res.Code)
	}
	if res := post(t, h, "/rpc/jg_k/evm/369", blockNumber, nil); res.Code != http.StatusTooManyRequests {
		t.Fatalf("second call = %d, want 429", res.Code)
	}
	clock.advance(time.Second)
	if res := post(t, h, "/rpc/jg_k/evm/369", blockNumber, nil); res.Code != http.StatusOK {
		t.Fatalf("third call = %d, want 200: the throttled call must not have been charged", res.Code)
	}
}

func TestBeaconCallsCountAgainstTheSameLimit(t *testing.T) {
	var got capturedRequest
	h := limitedHandler(t, limitedKey("k1", 1, 0), noon(), nil, &got)

	if res := post(t, h, "/rpc/jg_k/evm/369", blockNumber, nil); res.Code != http.StatusOK {
		t.Fatalf("rpc call = %d, want 200", res.Code)
	}
	if res := beaconCall(h, http.MethodGet, "/eth/v1/beacon/genesis"); res.Code != http.StatusTooManyRequests {
		t.Fatalf("beacon call after the limit = %d, want 429", res.Code)
	}
}

// Over WebSocket a throttled frame is answered with an error frame and the
// session survives, the same as an unpaid one.
func TestWSThrottledFrameKeepsTheSession(t *testing.T) {
	clock := noon()
	h := limitedHandler(t, limitedKey("k1", 1, 0), clock, nil, &capturedRequest{})
	ws := dialHandler(t, h, "/rpc/jg_k/evm/369")

	ws.send(t, blockNumber)
	if got := ws.read(t); got["result"] != "0x1" {
		t.Fatalf("first frame = %v, want result 0x1", got)
	}
	ws.send(t, blockNumber)
	if got := ws.read(t); errorCode(got) != codeRateLimited {
		t.Fatalf("second frame = %v, want error %d", got, codeRateLimited)
	}
	clock.advance(time.Second)
	ws.send(t, blockNumber)
	if got := ws.read(t); got["result"] != "0x1" {
		t.Fatalf("frame after the second rolled over = %v, want result 0x1", got)
	}
}
