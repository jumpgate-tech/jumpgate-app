package relay

import (
	"context"
	"errors"
	"net/http"
	"testing"
)

// Where metering actually bites. These tests decide, for every shape of key,
// whether a request costs the customer anything and whether it is served at all.

// countingCredits records spends and can be told to refuse or to break.
type countingCredits struct {
	spends  int
	total   int64
	refuse  bool
	failure error
}

func (c *countingCredits) Reserve(_ context.Context, _ string, credits int64) (int64, error) {
	if c.failure != nil {
		return 0, c.failure
	}
	if c.refuse {
		return 0, nil
	}
	return credits, nil
}

func (c *countingCredits) Settle(context.Context, string, int64, int64) error { return nil }

func creditedHandler(t *testing.T, rec KeyRecord, store CreditStore, got *capturedRequest) *Handler {
	t.Helper()
	up := stubUpstream(t, got)
	cfg := Config{Auth: staticAuth{rec: rec}, ProjectID: "main", ERPC: up}
	if store != nil {
		cfg.Credits = NewCreditLease(store, CreditOptions{BlockSize: 8})
	}
	h, err := NewHandler(cfg)
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}
	return h
}

func fundedKey() KeyRecord {
	return KeyRecord{ID: "k1", Enabled: true, AllowTrace: true, AccountAddress: "0xcustomer"}
}

// With no credit store the relay meters nothing and serves everything. This is
// how a gateway behaves before an operator turns metering on, and turning it on
// must be a deliberate act rather than a side effect.
func TestNoCreditStoreMetersNothing(t *testing.T) {
	var got capturedRequest
	h := creditedHandler(t, fundedKey(), nil, &got)

	if res := post(t, h, "/rpc/jg_k/evm/369", blockNumber, nil); res.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", res.Code)
	}
	if got.hits != 1 {
		t.Errorf("upstream hits = %d, want 1", got.hits)
	}
}

// A funded key is charged and served.
func TestFundedKeyIsChargedAndServed(t *testing.T) {
	var got capturedRequest
	credits := &countingCredits{}
	h := creditedHandler(t, fundedKey(), credits, &got)

	if res := post(t, h, "/rpc/jg_k/evm/369", blockNumber, nil); res.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", res.Code)
	}
	if got.hits != 1 {
		t.Errorf("upstream hits = %d, want 1", got.hits)
	}
}

// An empty account is refused with 402, and the call never reaches an upstream —
// so an unfunded customer costs the operator nothing at all.
func TestEmptyAccountIsRefusedWith402(t *testing.T) {
	var got capturedRequest
	credits := &countingCredits{refuse: true}
	h := creditedHandler(t, fundedKey(), credits, &got)

	res := post(t, h, "/rpc/jg_k/evm/369", blockNumber, nil)
	if res.Code != http.StatusPaymentRequired {
		t.Fatalf("status = %d, want 402", res.Code)
	}
	if got.hits != 0 {
		t.Error("an unfunded call still reached the upstream")
	}
}

// A credit-exempt key is served free. That flag is the ONLY way a key gets free
// service, which is what keeps the next test honest.
func TestCreditExemptKeyIsServedFree(t *testing.T) {
	var got capturedRequest
	credits := &countingCredits{refuse: true}
	rec := fundedKey()
	rec.CreditExempt = true
	h := creditedHandler(t, rec, credits, &got)

	if res := post(t, h, "/rpc/jg_k/evm/369", blockNumber, nil); res.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 for a credit-exempt key", res.Code)
	}
	if got.hits != 1 {
		t.Errorf("upstream hits = %d, want 1", got.hits)
	}
}

// A key bound to no funding account cannot be charged. With metering ON it is
// refused rather than served free — otherwise every unbound key would be a hole
// straight through the billing system, and credit_exempt would mean nothing.
func TestUnboundKeyIsRefusedWhenMeteringIsOn(t *testing.T) {
	var got capturedRequest
	credits := &countingCredits{}
	rec := KeyRecord{ID: "k1", Enabled: true, AllowTrace: true} // no AccountAddress
	h := creditedHandler(t, rec, credits, &got)

	res := post(t, h, "/rpc/jg_k/evm/369", blockNumber, nil)
	if res.Code != http.StatusPaymentRequired {
		t.Fatalf("status = %d, want 402 for a key with no funding account", res.Code)
	}
	if got.hits != 0 {
		t.Error("an unbillable call still reached the upstream")
	}
}

// A ledger outage is a 503, never a 402. Telling a funded customer to top up
// because the store hiccuped sends them to buy credits they already own.
func TestLedgerOutageIs503NotAPaymentProblem(t *testing.T) {
	var got capturedRequest
	credits := &countingCredits{failure: errors.New("socket is gone")}
	h := creditedHandler(t, fundedKey(), credits, &got)

	res := post(t, h, "/rpc/jg_k/evm/369", blockNumber, nil)
	if res.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", res.Code)
	}
	if res.Code == http.StatusPaymentRequired {
		t.Error("a ledger outage was reported as a payment problem")
	}
}

// A batch costs one credit per call in it. Charging a ten-call batch as one
// request would be a discount nobody designed.
func TestBatchIsChargedPerCall(t *testing.T) {
	var got capturedRequest
	credits := &countingCredits{}
	h := creditedHandler(t, fundedKey(), credits, &got)

	batch := `[{"jsonrpc":"2.0","id":1,"method":"eth_call"},` +
		`{"jsonrpc":"2.0","id":2,"method":"eth_call"},` +
		`{"jsonrpc":"2.0","id":3,"method":"eth_call"}]`
	if res := post(t, h, "/rpc/jg_k/evm/369", batch, nil); res.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", res.Code)
	}
	if got.hits != 1 {
		t.Fatalf("upstream hits = %d, want 1", got.hits)
	}
}
