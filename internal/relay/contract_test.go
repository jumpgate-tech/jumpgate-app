package relay

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"reflect"
	"testing"
)

// The key record crosses a language boundary: the Rust billing service builds
// it and this package decodes it, and neither compiler sees the other side. The
// two drifted once. The relay expected flat method_allow, per_second_limit, ...
// fields that billing never sends, so every per-key constraint and rate limit
// decoded empty and nothing was enforced.
//
// testdata/authenticate_contract.json is the contract. Billing's own test
// (services/billing/src/admin.rs, authenticate_view_matches_the_relay_contract_fixture)
// fails unless it emits exactly that file, and the tests here decode the same
// bytes. A change on either side now fails a test rather than a policy.

const contractFixture = "testdata/authenticate_contract.json"

// contractCase is one reply billing is pinned to, kept raw so the client
// decodes exactly the bytes billing would send.
type contractCase struct {
	Case   string          `json:"case"`
	Record json.RawMessage `json:"record"`
}

// loadContract reads the fixture's cases by name.
func loadContract(t *testing.T) map[string]json.RawMessage {
	t.Helper()
	raw, err := os.ReadFile(contractFixture)
	if err != nil {
		t.Fatalf("read %s: %v", contractFixture, err)
	}
	var cases []contractCase
	if err := json.Unmarshal(raw, &cases); err != nil {
		t.Fatalf("parse %s: %v", contractFixture, err)
	}
	out := make(map[string]json.RawMessage, len(cases))
	for _, c := range cases {
		out[c.Case] = c.Record
	}
	return out
}

// contractRecord decodes one fixture case through the real BillingClient over
// its real unix socket, so the test covers the path production takes rather
// than a decoder called on the side.
func contractRecord(t *testing.T, name string) KeyRecord {
	t.Helper()
	body, ok := loadContract(t)[name]
	if !ok {
		t.Fatalf("%s has no case %q", contractFixture, name)
	}
	stub := newBillingStub(t)
	stub.body = string(body)
	rec, err := NewBillingClient(stub.socket, "relay-token").Authenticate(context.Background(), "jg_k")
	if err != nil {
		t.Fatalf("Authenticate(%s): %v", name, err)
	}
	return rec
}

func TestContractDecodesALimitedKeyWithEveryConstraint(t *testing.T) {
	got := contractRecord(t, "limited_with_every_constraint")
	want := KeyRecord{
		ID:             "key_limited",
		Label:          "key_limited-label",
		Enabled:        true,
		CreditExempt:   false,
		AllowTrace:     true,
		MethodAllow:    []string{"eth_call", "eth_getBalance"},
		MethodBlock:    []string{"debug_traceCall", "eth_sendRawTransaction"},
		Origins:        []string{"https://app.example.com", "https://staging.example.com"},
		Networks:       []string{"369", "943"},
		IPAllow:        []string{"203.0.113.0/24", "198.51.100.7"},
		IPDeny:         []string{"203.0.113.66", "2001:db8::/32"},
		RateUnlimited:  false,
		PerSecondLimit: 50,
		PerDayLimit:    100_000,
		AccountAddress: "0x00000000000000000000000000000000000000aa",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("decoded record:\n got %+v\nwant %+v", got, want)
	}
}

func TestContractDecodesAnUnlimitedKeyWithoutConstraints(t *testing.T) {
	got := contractRecord(t, "unlimited_without_constraints")
	if !got.RateUnlimited {
		t.Error("RateUnlimited = false, want true")
	}
	if got.PerSecondLimit != 0 || got.PerDayLimit != 0 {
		t.Errorf("limits = %d/s %d/day, want none", got.PerSecondLimit, got.PerDayLimit)
	}
	if !got.CreditExempt {
		t.Error("CreditExempt = false, want true")
	}
	// A null account decodes to "", which charge() reads as "no account".
	if got.AccountAddress != "" {
		t.Errorf("AccountAddress = %q, want empty", got.AccountAddress)
	}
	for name, list := range map[string][]string{
		"MethodAllow": got.MethodAllow, "MethodBlock": got.MethodBlock,
		"Origins": got.Origins, "Networks": got.Networks,
		"IPAllow": got.IPAllow, "IPDeny": got.IPDeny,
	} {
		if len(list) != 0 {
			t.Errorf("%s = %v, want empty", name, list)
		}
	}
	if got.ID != "key_unlimited" || !got.Enabled {
		t.Errorf("id/enabled = %q/%v, want key_unlimited/true", got.ID, got.Enabled)
	}
}

func TestContractDecodesAPerDayOnlyLimit(t *testing.T) {
	got := contractRecord(t, "per_day_only")
	if got.RateUnlimited {
		t.Error("RateUnlimited = true, want false")
	}
	if got.PerSecondLimit != 0 || got.PerDayLimit != 5000 {
		t.Errorf("limits = %d/s %d/day, want 0/s 5000/day", got.PerSecondLimit, got.PerDayLimit)
	}
	if !reflect.DeepEqual(got.Networks, []string{"1"}) {
		t.Errorf("Networks = %v, want [1]", got.Networks)
	}
}

// A reply the relay cannot fully read must be an error, never a key with no
// policy. The original bug was exactly that: missing fields decoded as zero
// values, and a zero-valued policy allows everything.
func TestContractRefusesARecordItCannotRead(t *testing.T) {
	const constraints = `"constraints":{"origins":[],"method_allow":[],"method_block":[],
	                     "networks":[],"ip_allow":[],"ip_deny":[]}`
	tests := []struct {
		name string
		body string
	}{
		{"the old flat shape", `{"id":"k1","enabled":true,"rate_unlimited":false,"per_second_limit":5,"ip_allow":["10.0.0.1"]}`},
		{"no constraints", `{"id":"k1","enabled":true,"rate":"unlimited"}`},
		{"no rate", `{"id":"k1","enabled":true,` + constraints + `}`},
		{"null rate", `{"id":"k1","enabled":true,"rate":null,` + constraints + `}`},
		{"unknown rate word", `{"id":"k1","enabled":true,"rate":"bounded",` + constraints + `}`},
		{"unknown rate variant", `{"id":"k1","enabled":true,"rate":{"burst":{"per_second":1}},` + constraints + `}`},
		{"limited without per_day", `{"id":"k1","enabled":true,"rate":{"limited":{"per_second":1}},` + constraints + `}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stub := newBillingStub(t)
			stub.body = tt.body
			_, err := NewBillingClient(stub.socket, "relay-token").Authenticate(context.Background(), "jg_k")
			if err == nil {
				t.Fatal("err = nil, want a decode failure")
			}
			if errors.Is(err, ErrUnknownKey) || errors.Is(err, ErrDisabledKey) {
				t.Fatalf("err = %v, must not be a verdict about the key", err)
			}
		})
	}
}

// Decoding is only half of it. A key whose IP allow-list excludes the caller,
// delivered in billing's real shape through the real client and cache, must be
// refused before anything reaches the upstream.
func TestContractIPAllowListIsEnforcedEndToEnd(t *testing.T) {
	stub := newBillingStub(t)
	stub.body = `{"id":"k1","label":"browser","enabled":true,"account_address":null,
	              "credit_exempt":true,"allow_trace":false,"rate":"unlimited",
	              "constraints":{"origins":[],"method_allow":[],"method_block":[],
	                             "networks":[],"ip_allow":["203.0.113.0/24"],"ip_deny":[]}}`
	cache := NewKeyCache(NewBillingClient(stub.socket, "relay-token"), CacheOptions{})

	var got capturedRequest
	h, err := NewHandler(Config{Auth: cache, ProjectID: "main", ERPC: stubUpstream(t, &got)})
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}

	if res := postFrom(h, "198.51.100.9:5555"); res.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403: a caller outside the allow-list was served (body %q)",
			res.Code, res.Body.String())
	}
	if got.hits != 0 {
		t.Errorf("upstream hits = %d, want 0", got.hits)
	}
	if stub.hits == 0 {
		t.Error("billing was never asked: the test did not exercise the real decode")
	}

	// The control: a caller inside the allow-list is served, so the refusal
	// above is the policy and not some other failure.
	if res := postFrom(h, "203.0.113.7:5555"); res.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 for an allowed caller (body %q)", res.Code, res.Body.String())
	}
}
