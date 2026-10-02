package relay

import (
	"context"
	"net/http"
	"testing"
	"time"
)

// The store owns pricing, and it is not one credit per call: the seeded book
// charges 20 for a default method and 75 for eth_getLogs. A relay that assumed
// 1 would undercharge by a factor of twenty and nothing would report it.

func TestPriceClientReadsTheStorePrice(t *testing.T) {
	stub := newBillingStub(t)
	stub.body = `{"method":"eth_getLogs","chain_id":1,"credits":75,"known":true}`

	c := NewBillingClient(stub.socket, "relay-token")
	got := c.PriceOf(context.Background(), "eth_getLogs", 1)
	if got != 75 {
		t.Errorf("price = %d, want 75", got)
	}
	if stub.gotPath != "/internal/price" {
		t.Errorf("path = %q, want /internal/price", stub.gotPath)
	}
}

// Prices change on an operator's clock, not a customer's, so the same method is
// asked once and then remembered. Asking per request would put a round trip on
// every call the lease exists to keep off the network.
func TestPriceClientCachesAPrice(t *testing.T) {
	stub := newBillingStub(t)
	stub.body = `{"method":"eth_call","chain_id":1,"credits":20,"known":true}`

	c := NewBillingClient(stub.socket, "relay-token")
	for i := 0; i < 5; i++ {
		if got := c.PriceOf(context.Background(), "eth_call", 1); got != 20 {
			t.Fatalf("call %d: price = %d, want 20", i, got)
		}
	}
	if stub.hits != 1 {
		t.Errorf("store was asked %d times, want 1 — the price is not cached", stub.hits)
	}
}

// An unreachable store must not make calls FREE. Falling back to zero would give
// the product away the moment the ledger hiccupped, so the fallback charges.
func TestPriceClientFallsBackToACharge(t *testing.T) {
	stub := newBillingStub(t)
	stub.status = http.StatusInternalServerError
	stub.body = `{"error":"boom"}`

	c := NewBillingClient(stub.socket, "relay-token")
	if got := c.PriceOf(context.Background(), "eth_call", 1); got <= 0 {
		t.Errorf("price = %d, want a positive fallback — an outage must not make calls free", got)
	}
}

func newClockedClient(t *testing.T, stub *billingStub) (*BillingClient, *fakeClock) {
	t.Helper()
	clock := &fakeClock{now: time.Unix(1_700_000_000, 0)}
	c := NewBillingClient(stub.socket, "relay-token")
	c.now = clock.Now
	return c, clock
}

// The review's finding: one failed lookup became the method's price for the life
// of the process, so a ledger blip at startup sold eth_getLogs at 1 credit
// instead of 75 until the next restart.
func TestPriceClientDoesNotKeepAFallback(t *testing.T) {
	stub := newBillingStub(t)
	stub.status = http.StatusInternalServerError
	c, clock := newClockedClient(t, stub)

	if got := c.PriceOf(context.Background(), "eth_getLogs", 1); got != fallbackPrice {
		t.Fatalf("during the outage: price = %d, want the fallback %d", got, fallbackPrice)
	}

	stub.status = http.StatusOK
	stub.body = `{"method":"eth_getLogs","chain_id":1,"credits":75,"known":true}`
	clock.advance(priceRetryAfter)
	if got := c.PriceOf(context.Background(), "eth_getLogs", 1); got != 75 {
		t.Errorf("after the store recovered: price = %d, want 75", got)
	}
}

// An outage must not turn every request into a 3 s wait on a dead socket, so a
// fallback is held for a short retry window rather than re-asked per call.
func TestPriceClientRetriesAnOutageOnAWindow(t *testing.T) {
	stub := newBillingStub(t)
	stub.status = http.StatusInternalServerError
	c, _ := newClockedClient(t, stub)

	for i := 0; i < 5; i++ {
		c.PriceOf(context.Background(), "eth_call", 1)
	}
	if stub.hits != 1 {
		t.Errorf("store was asked %d times inside one retry window, want 1", stub.hits)
	}
}

// Prices are an operator's to change, so a remembered price expires.
func TestPriceClientPicksUpAPriceChange(t *testing.T) {
	stub := newBillingStub(t)
	stub.body = `{"method":"eth_call","chain_id":1,"credits":20,"known":true}`
	c, clock := newClockedClient(t, stub)
	c.PriceOf(context.Background(), "eth_call", 1)

	stub.body = `{"method":"eth_call","chain_id":1,"credits":30,"known":true}`
	clock.advance(priceTTL - time.Second)
	if got := c.PriceOf(context.Background(), "eth_call", 1); got != 20 {
		t.Errorf("inside the TTL: price = %d, want the remembered 20", got)
	}
	clock.advance(2 * time.Second)
	if got := c.PriceOf(context.Background(), "eth_call", 1); got != 30 {
		t.Errorf("after the TTL: price = %d, want the new 30", got)
	}
}

// A known price beats the fallback: if the refresh fails, keep charging what the
// operator last set rather than dropping to 1.
func TestPriceClientKeepsTheLastGoodPriceThroughAnOutage(t *testing.T) {
	stub := newBillingStub(t)
	stub.body = `{"method":"eth_getLogs","chain_id":1,"credits":75,"known":true}`
	c, clock := newClockedClient(t, stub)
	c.PriceOf(context.Background(), "eth_getLogs", 1)

	stub.status = http.StatusServiceUnavailable
	clock.advance(priceTTL + time.Second)
	if got := c.PriceOf(context.Background(), "eth_getLogs", 1); got != 75 {
		t.Errorf("refresh failed: price = %d, want the last good 75", got)
	}
}
