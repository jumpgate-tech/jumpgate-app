package relay

import (
	"context"
	"net/http"
	"testing"
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
