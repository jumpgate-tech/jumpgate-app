package relay

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

// The beacon proxy is metered and narrowed to the read API. Before this, any
// enabled key reached any path on the operator's beacon node, with any method,
// for free.

func newBeaconHandler(t *testing.T, credits *CreditLease) (*Handler, *capturedRequest) {
	t.Helper()
	got := &capturedRequest{}
	beacon := stubUpstream(t, got)
	h, err := NewHandler(Config{
		Auth:      staticAuth{rec: meteredKey()},
		ProjectID: "main",
		ERPC:      stubUpstream(t, &capturedRequest{}),
		Beacon:    func(int) (*url.URL, bool) { return beacon, true },
		Credits:   credits,
	})
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}
	return h, got
}

func beaconCall(h http.Handler, method, rest string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, "/beacon/jg_k/evm/369"+rest, nil)
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	return res
}

func TestHandlerChargesBeaconCalls(t *testing.T) {
	h, got := newBeaconHandler(t, newLease(newLedger(acct, 1), 1))

	if res := beaconCall(h, http.MethodGet, "/eth/v1/beacon/genesis"); res.Code != http.StatusOK {
		t.Fatalf("paid call = %d, want 200", res.Code)
	}
	if res := beaconCall(h, http.MethodGet, "/eth/v1/beacon/genesis"); res.Code != http.StatusPaymentRequired {
		t.Fatalf("unpaid call = %d, want 402", res.Code)
	}
	if got.hits != 1 {
		t.Errorf("beacon node served %d calls, want 1", got.hits)
	}
}

func TestHandlerBeaconServesOnlyTheReadAPI(t *testing.T) {
	h, got := newBeaconHandler(t, nil)

	allowed := []struct{ method, rest string }{
		{http.MethodGet, "/eth/v1/beacon/headers/head"},
		{http.MethodGet, "/eth/v2/beacon/blocks/head"},
		{http.MethodHead, "/eth/v1/node/syncing"},
		{http.MethodPost, "/eth/v1/beacon/states/head/validators"},
		{http.MethodPost, "/eth/v1/beacon/states/head/validator_balances"},
	}
	for _, c := range allowed {
		if res := beaconCall(h, c.method, c.rest); res.Code != http.StatusOK {
			t.Errorf("%s %s = %d, want 200", c.method, c.rest, res.Code)
		}
	}
	served := got.hits

	refused := []struct {
		method, rest string
		want         int
	}{
		{http.MethodGet, "", http.StatusNotFound},
		{http.MethodGet, "/lighthouse/peers", http.StatusNotFound},
		{http.MethodGet, "/eth/v1/../lighthouse/peers", http.StatusNotFound},
		{http.MethodGet, "/eth/v1/./node/health", http.StatusNotFound},
		{http.MethodGet, "/eth/v3/validator/blocks/1", http.StatusNotFound},
		{http.MethodGet, "/eth/v1/validator/duties/proposer/1", http.StatusNotFound},
		{http.MethodGet, "/eth/v2/debug/beacon/states/head", http.StatusNotFound},
		{http.MethodGet, "/eth/v1/events?topics=head", http.StatusNotFound},
		{http.MethodPost, "/eth/v1/beacon/blocks", http.StatusMethodNotAllowed},
		{http.MethodPost, "/eth/v1/beacon/pool/voluntary_exits", http.StatusMethodNotAllowed},
		{http.MethodDelete, "/eth/v1/beacon/headers/head", http.StatusMethodNotAllowed},
	}
	for _, c := range refused {
		if res := beaconCall(h, c.method, c.rest); res.Code != c.want {
			t.Errorf("%s %s = %d, want %d", c.method, c.rest, res.Code, c.want)
		}
	}
	if got.hits != served {
		t.Errorf("refused requests reached the beacon node %d times", got.hits-served)
	}
}

// A refused path must not cost the customer anything.
func TestHandlerBeaconRefusalIsFree(t *testing.T) {
	ledger := newLedger(acct, 5)
	h, _ := newBeaconHandler(t, newLease(ledger, 5))
	beaconCall(h, http.MethodGet, "/lighthouse/peers")
	beaconCall(h, http.MethodGet, "/eth/v1/beacon/genesis")
	if spent := 5 - ledger.total(acct); spent != 0 {
		t.Fatalf("ledger total moved by %d before settle; want the lease to hold it", spent)
	}
	// One paid call out of a 5-credit block: four credits left in the lease.
	if err := h.cfg.Credits.Spend(t.Context(), acct, 4); err != nil {
		t.Fatalf("the refused call was charged: %v", err)
	}
}
