package relay

import (
	"errors"
	"testing"
)

// Policy is what a key is allowed to do. It runs after authentication and
// before the strip, so a denied call never reaches an upstream and never costs
// a credit.

func TestParseMethodsSingleCall(t *testing.T) {
	got, err := ParseMethods([]byte(`{"jsonrpc":"2.0","id":1,"method":"eth_blockNumber","params":[]}`))
	if err != nil {
		t.Fatalf("ParseMethods: %v", err)
	}
	if len(got) != 1 || got[0] != "eth_blockNumber" {
		t.Errorf("methods = %v, want [eth_blockNumber]", got)
	}
}

func TestParseMethodsBatch(t *testing.T) {
	body := `[{"jsonrpc":"2.0","id":1,"method":"eth_blockNumber"},
	          {"jsonrpc":"2.0","id":2,"method":"eth_getBalance"}]`
	got, err := ParseMethods([]byte(body))
	if err != nil {
		t.Fatalf("ParseMethods: %v", err)
	}
	if len(got) != 2 || got[0] != "eth_blockNumber" || got[1] != "eth_getBalance" {
		t.Errorf("methods = %v, want [eth_blockNumber eth_getBalance]", got)
	}
}

// A body the relay cannot read is refused, never forwarded. Forwarding an
// unparsed body would mean an upstream sees a call the policy never checked.
func TestParseMethodsRejectsGarbage(t *testing.T) {
	for _, body := range []string{``, `not json`, `{`, `[{"method":}]`, `null`} {
		if _, err := ParseMethods([]byte(body)); !errors.Is(err, ErrBadRequest) {
			t.Errorf("ParseMethods(%q) err = %v, want ErrBadRequest", body, err)
		}
	}
}

// An empty batch has nothing to check and nothing to answer. Forwarding it
// wastes an upstream call for a caller who asked nothing.
func TestParseMethodsRejectsEmptyBatch(t *testing.T) {
	if _, err := ParseMethods([]byte(`[]`)); !errors.Is(err, ErrBadRequest) {
		t.Fatalf("err = %v, want ErrBadRequest", err)
	}
}

// A call with no method is not a JSON-RPC call.
func TestParseMethodsRejectsMissingMethod(t *testing.T) {
	if _, err := ParseMethods([]byte(`{"jsonrpc":"2.0","id":1}`)); !errors.Is(err, ErrBadRequest) {
		t.Fatalf("err = %v, want ErrBadRequest", err)
	}
}

// No policy at all still works. A key with empty lists is the common case and
// must not be accidentally locked out.
func TestCheckMethodsAllowsWhenNoPolicySet(t *testing.T) {
	rec := KeyRecord{Enabled: true}
	if err := CheckMethods(rec, []string{"eth_call"}); err != nil {
		t.Fatalf("CheckMethods: %v", err)
	}
}

// An empty allow list means "everything except the block list".
func TestCheckMethodsBlockList(t *testing.T) {
	rec := KeyRecord{Enabled: true, MethodBlock: []string{"eth_sendRawTransaction"}}
	if err := CheckMethods(rec, []string{"eth_call"}); err != nil {
		t.Errorf("eth_call: %v", err)
	}
	if err := CheckMethods(rec, []string{"eth_sendRawTransaction"}); !errors.Is(err, ErrMethodDenied) {
		t.Errorf("eth_sendRawTransaction err = %v, want ErrMethodDenied", err)
	}
}

// A non-empty allow list is exclusive: everything not named is denied.
func TestCheckMethodsAllowListIsExclusive(t *testing.T) {
	rec := KeyRecord{Enabled: true, MethodAllow: []string{"eth_call", "eth_blockNumber"}}
	if err := CheckMethods(rec, []string{"eth_call"}); err != nil {
		t.Errorf("eth_call: %v", err)
	}
	if err := CheckMethods(rec, []string{"eth_getBalance"}); !errors.Is(err, ErrMethodDenied) {
		t.Errorf("eth_getBalance err = %v, want ErrMethodDenied", err)
	}
}

// The block list wins. A method on both lists is denied, because a deny is a
// deliberate act and an allow may be a broad default.
func TestCheckMethodsBlockBeatsAllow(t *testing.T) {
	rec := KeyRecord{
		Enabled:     true,
		MethodAllow: []string{"eth_call", "debug_traceCall"},
		MethodBlock: []string{"debug_traceCall"},
	}
	if err := CheckMethods(rec, []string{"debug_traceCall"}); !errors.Is(err, ErrMethodDenied) {
		t.Fatalf("err = %v, want ErrMethodDenied — block must beat allow", err)
	}
}

// trace and debug are the expensive namespaces. allow_trace gates both, so an
// operator does not have to enumerate every method one at a time.
func TestCheckMethodsGatesTraceNamespaces(t *testing.T) {
	locked := KeyRecord{Enabled: true, AllowTrace: false}
	for _, m := range []string{"debug_traceTransaction", "trace_block", "trace_call", "debug_storageRangeAt"} {
		if err := CheckMethods(locked, []string{m}); !errors.Is(err, ErrMethodDenied) {
			t.Errorf("%s err = %v, want ErrMethodDenied when AllowTrace is false", m, err)
		}
	}

	open := KeyRecord{Enabled: true, AllowTrace: true}
	for _, m := range []string{"debug_traceTransaction", "trace_block"} {
		if err := CheckMethods(open, []string{m}); err != nil {
			t.Errorf("%s: %v when AllowTrace is true", m, err)
		}
	}
}

// A method name that merely starts with the letters "trace" is not the trace
// namespace. The gate matches on the namespace separator, not a prefix.
func TestCheckMethodsTraceGateMatchesNamespaceNotPrefix(t *testing.T) {
	rec := KeyRecord{Enabled: true, AllowTrace: false}
	if err := CheckMethods(rec, []string{"tracker_getThing"}); err != nil {
		t.Fatalf("tracker_getThing was denied by the trace gate: %v", err)
	}
}

// One denied entry refuses the whole batch. A partial batch would make
// per-request metering ambiguous for no real gain to the caller.
func TestCheckMethodsRefusesWholeBatchOnOneDenial(t *testing.T) {
	rec := KeyRecord{Enabled: true, MethodBlock: []string{"eth_sendRawTransaction"}}
	err := CheckMethods(rec, []string{"eth_call", "eth_sendRawTransaction", "eth_blockNumber"})
	if !errors.Is(err, ErrMethodDenied) {
		t.Fatalf("err = %v, want ErrMethodDenied", err)
	}
}

// Method names are compared case-sensitively, as JSON-RPC defines them. A
// case-insensitive compare would let eth_SENDRAWTRANSACTION slip past a block
// list that names the real method.
func TestCheckMethodsIsCaseSensitive(t *testing.T) {
	rec := KeyRecord{Enabled: true, MethodAllow: []string{"eth_call"}}
	if err := CheckMethods(rec, []string{"ETH_CALL"}); !errors.Is(err, ErrMethodDenied) {
		t.Fatalf("err = %v, want ErrMethodDenied", err)
	}
}

func TestCheckOrigin(t *testing.T) {
	tests := []struct {
		name    string
		origins []string
		origin  string
		wantErr bool
	}{
		{"no policy allows any", nil, "https://anything.example", false},
		{"no policy allows empty", nil, "", false},
		{"exact match passes", []string{"https://app.example"}, "https://app.example", false},
		{"other origin denied", []string{"https://app.example"}, "https://evil.example", true},
		{"missing origin denied when set", []string{"https://app.example"}, "", true},
		{"one of several passes", []string{"https://a.example", "https://b.example"}, "https://b.example", false},
		{"star allows any", []string{"*"}, "https://whatever.example", false},
	}
	for _, tt := range tests {
		rec := KeyRecord{Enabled: true, Origins: tt.origins}
		err := CheckOrigin(rec, tt.origin)
		if tt.wantErr && !errors.Is(err, ErrOriginDenied) {
			t.Errorf("%s: err = %v, want ErrOriginDenied", tt.name, err)
		}
		if !tt.wantErr && err != nil {
			t.Errorf("%s: err = %v, want nil", tt.name, err)
		}
	}
}

func TestCheckIP(t *testing.T) {
	tests := []struct {
		name    string
		allow   []string
		deny    []string
		ip      string
		wantErr bool
	}{
		{"no policy allows any", nil, nil, "203.0.113.5", false},
		{"allow list exact", []string{"203.0.113.5"}, nil, "203.0.113.5", false},
		{"allow list excludes others", []string{"203.0.113.5"}, nil, "198.51.100.9", true},
		{"allow list cidr", []string{"203.0.113.0/24"}, nil, "203.0.113.77", false},
		{"outside cidr denied", []string{"203.0.113.0/24"}, nil, "198.51.100.9", true},
		{"deny wins over allow", []string{"203.0.113.0/24"}, []string{"203.0.113.5"}, "203.0.113.5", true},
		{"deny cidr", nil, []string{"198.51.100.0/24"}, "198.51.100.9", true},
		{"unparseable ip denied when a policy exists", []string{"203.0.113.0/24"}, nil, "not-an-ip", true},
	}
	for _, tt := range tests {
		rec := KeyRecord{Enabled: true, IPAllow: tt.allow, IPDeny: tt.deny}
		err := CheckIP(rec, tt.ip)
		if tt.wantErr && !errors.Is(err, ErrIPDenied) {
			t.Errorf("%s: err = %v, want ErrIPDenied", tt.name, err)
		}
		if !tt.wantErr && err != nil {
			t.Errorf("%s: err = %v, want nil", tt.name, err)
		}
	}
}

// A key scoped to one chain must not serve another. The network list holds
// chain ids as strings, matching the store's key_constraint rows.
func TestCheckNetwork(t *testing.T) {
	rec := KeyRecord{Enabled: true, Networks: []string{"1", "369"}}
	if err := CheckNetwork(rec, 369); err != nil {
		t.Errorf("chain 369: %v", err)
	}
	if err := CheckNetwork(rec, 137); !errors.Is(err, ErrNetworkDenied) {
		t.Errorf("chain 137 err = %v, want ErrNetworkDenied", err)
	}

	open := KeyRecord{Enabled: true}
	if err := CheckNetwork(open, 137); err != nil {
		t.Errorf("no policy: %v", err)
	}
}
