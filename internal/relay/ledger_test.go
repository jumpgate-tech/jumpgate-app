package relay

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
)

// The Go half of the ledger. These shapes must match the Rust routes byte for
// byte, so the tests assert on the exact wire form rather than on a struct.

func TestBillingClientReserveReturnsTheGrant(t *testing.T) {
	stub := newBillingStub(t)
	stub.body = `{"granted":30}`

	c := NewBillingClient(stub.socket, "relay-token")
	granted, err := c.Reserve(context.Background(), "0xcustomer", 100)
	if err != nil {
		t.Fatalf("Reserve: %v", err)
	}
	if granted != 30 {
		t.Errorf("granted = %d, want 30", granted)
	}
	if stub.gotPath != "/internal/reserve" {
		t.Errorf("path = %q, want /internal/reserve", stub.gotPath)
	}
	if !strings.Contains(stub.gotBody, `"account":"0xcustomer"`) {
		t.Errorf("body = %q, want it to carry the account", stub.gotBody)
	}
	if !strings.Contains(stub.gotBody, `"credits":100`) {
		t.Errorf("body = %q, want it to carry the requested credits", stub.gotBody)
	}
}

// A zero grant is a NORMAL answer meaning "out of credits", not an error. The
// lease turns it into a 402; the client must not turn it into a 503.
func TestBillingClientReserveZeroIsNotAnError(t *testing.T) {
	stub := newBillingStub(t)
	stub.body = `{"granted":0}`

	c := NewBillingClient(stub.socket, "relay-token")
	granted, err := c.Reserve(context.Background(), "0xcustomer", 100)
	if err != nil {
		t.Fatalf("a zero grant must not be an error: %v", err)
	}
	if granted != 0 {
		t.Errorf("granted = %d, want 0", granted)
	}
}

// An account the ledger has never seen is a distinct fact. It must not read as
// an outage, or an operator would chase a broken socket instead of an unbound
// key.
func TestBillingClientReserveUnknownAccount(t *testing.T) {
	stub := newBillingStub(t)
	stub.status = http.StatusNotFound
	stub.body = `{"error":"account not found: 0xnobody"}`

	c := NewBillingClient(stub.socket, "relay-token")
	if _, err := c.Reserve(context.Background(), "0xnobody", 10); !errors.Is(err, ErrNoAccount) {
		t.Fatalf("err = %v, want ErrNoAccount", err)
	}
}

func TestBillingClientSettle(t *testing.T) {
	stub := newBillingStub(t)
	stub.body = `{"credits_remaining":88,"credits_reserved":0}`

	c := NewBillingClient(stub.socket, "relay-token")
	if err := c.Settle(context.Background(), "0xcustomer", 12, 30); err != nil {
		t.Fatalf("Settle: %v", err)
	}
	if stub.gotPath != "/internal/settle" {
		t.Errorf("path = %q, want /internal/settle", stub.gotPath)
	}
	for _, want := range []string{`"account":"0xcustomer"`, `"spent":12`, `"reserved":30`} {
		if !strings.Contains(stub.gotBody, want) {
			t.Errorf("body = %q, want it to contain %s", stub.gotBody, want)
		}
	}
}

// The ledger rejecting a settle is an error the relay must surface, not swallow
// — a silently dropped settle strands a customer's credits in a reservation.
func TestBillingClientSettleSurfacesARejection(t *testing.T) {
	stub := newBillingStub(t)
	stub.status = http.StatusBadRequest
	stub.body = `{"error":"invalid settle: spent 40 must be between 0 and reserved 30"}`

	c := NewBillingClient(stub.socket, "relay-token")
	if err := c.Settle(context.Background(), "0xcustomer", 40, 30); err == nil {
		t.Fatal("err = nil, want the rejection surfaced")
	}
}

// The relay credential is what opens these routes, not the admin one.
func TestBillingClientLedgerUsesTheRelayToken(t *testing.T) {
	stub := newBillingStub(t)
	stub.body = `{"granted":1}`

	c := NewBillingClient(stub.socket, "relay-token")
	if _, err := c.Reserve(context.Background(), "0xcustomer", 1); err != nil {
		t.Fatalf("Reserve: %v", err)
	}
	if want := "Bearer relay-token"; stub.gotAuth != want {
		t.Errorf("Authorization = %q, want %q", stub.gotAuth, want)
	}
}

// A CreditLease driven by the real client must satisfy the CreditStore contract.
// This is a compile-time guarantee, asserted here so a signature drift on either
// side fails the build rather than production.
func TestBillingClientSatisfiesCreditStore(t *testing.T) {
	var _ CreditStore = (*BillingClient)(nil)
}
