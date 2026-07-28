package chainlist

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

// A zero-valued Discoverer has to work. It is the shape a caller gets from
// `var d Discoverer` or from a struct literal that sets only the one field it
// cares about, and every accessor below exists so that partial construction
// does not produce a client with no timeout or a concurrency of zero — the
// latter would deadlock rather than fail.
func TestDiscoverer_ZeroValueFallsBackToDefaults(t *testing.T) {
	var d Discoverer

	if got := d.httpClient(); got != http.DefaultClient {
		t.Errorf("httpClient = %v, want http.DefaultClient", got)
	}
	if got := d.feedURL(); got != FeedURL {
		t.Errorf("feedURL = %q, want %q", got, FeedURL)
	}
	if got := d.probeTimeout(); got != defaultProbeTimeout {
		t.Errorf("probeTimeout = %v, want %v", got, defaultProbeTimeout)
	}
	if got := d.concurrency(); got != defaultConcurrency {
		t.Errorf("concurrency = %d, want %d", got, defaultConcurrency)
	}
	// Zero would mean "no worker may run", which deadlocks rather than
	// failing — the reason this accessor exists at all.
	if d.concurrency() < 1 {
		t.Fatal("a concurrency below 1 cannot make progress")
	}
	if got := d.dialer(); got == nil {
		t.Error("dialer = nil, want a usable dialer")
	}
}

// And an explicitly configured one is left alone, or injection does nothing.
func TestDiscoverer_ExplicitSettingsWin(t *testing.T) {
	client := &http.Client{Timeout: time.Second}
	d := Discoverer{
		HTTPClient:   client,
		FeedURL:      "https://chains.example/list.json",
		ProbeTimeout: 3 * time.Second,
		Concurrency:  7,
	}

	if got := d.httpClient(); got != client {
		t.Error("the injected http client was ignored")
	}
	if got := d.feedURL(); got != "https://chains.example/list.json" {
		t.Errorf("feedURL = %q, want the configured feed", got)
	}
	if got := d.probeTimeout(); got != 3*time.Second {
		t.Errorf("probeTimeout = %v, want 3s", got)
	}
	if got := d.concurrency(); got != 7 {
		t.Errorf("concurrency = %d, want 7", got)
	}
}

// New's defaults must be usable as-is.
func TestNew_IsUsableWithoutFurtherSetup(t *testing.T) {
	d := New()
	if d.probeTimeout() <= 0 || d.concurrency() < 1 || d.httpClient() == nil || d.feedURL() == "" {
		t.Fatalf("New returned something unusable: %+v", d)
	}
}

// ---------------------------------------------------------------------
// the vendored snapshot
// ---------------------------------------------------------------------

// The order is fixed rather than a map range: this is user-visible output
// ("these chains still get fallback upstreams") and must not shuffle between
// runs of the same binary.
func TestVendoredChainIDs_IsStableAndMatchesTheSnapshot(t *testing.T) {
	first := VendoredChainIDs()
	if len(first) == 0 {
		t.Fatal("no chains are covered by the offline snapshot")
	}
	for i := 0; i < 20; i++ {
		got := VendoredChainIDs()
		if len(got) != len(first) {
			t.Fatalf("length changed between calls: %v then %v", first, got)
		}
		for j := range got {
			if got[j] != first[j] {
				t.Fatalf("order changed between calls: %v then %v", first, got)
			}
		}
	}

	// Every id it advertises must actually resolve to vendored endpoints, or
	// an offline operator is told a chain is covered and then handed nothing.
	for _, id := range first {
		eps, ok := Vendored(id)
		if !ok || len(eps) == 0 {
			t.Errorf("chain %d is advertised as covered but has no vendored endpoints", id)
		}
	}
}

// ---------------------------------------------------------------------
// parseChainID
// ---------------------------------------------------------------------

// eth_chainId answers in hex, and a node that answers something else must be
// rejected rather than parsed into a chain id nobody asked for.
func TestParseChainID(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want int
		bad  bool
	}{
		{name: "hex with prefix", in: `{"jsonrpc":"2.0","id":1,"result":"0x171"}`, want: 369},
		{name: "mainnet", in: `{"jsonrpc":"2.0","id":1,"result":"0x1"}`, want: 1},
		{name: "an error reply", in: `{"jsonrpc":"2.0","id":1,"error":{"code":-32601,"message":"nope"}}`, bad: true},
		{name: "not json at all", in: `<html>502 Bad Gateway</html>`, bad: true},
		{name: "empty body", in: ``, bad: true},
		{name: "a result that is not hex", in: `{"jsonrpc":"2.0","id":1,"result":"three-six-nine"}`, bad: true},
		{name: "a null result", in: `{"jsonrpc":"2.0","id":1,"result":null}`, bad: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseChainID([]byte(tc.in))
			if tc.bad {
				if err == nil {
					t.Fatalf("parsed %d out of %q, want an error", got, tc.in)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseChainID(%q): %v", tc.in, err)
			}
			if got != tc.want {
				t.Errorf("got %d, want %d", got, tc.want)
			}
		})
	}
}

// probeErr turns transport plumbing into something an operator can read:
// "context deadline exceeded" is a timeout, and url.Error's prefix repeats a
// URL the caller already has.
func TestProbeErr_ReadsAsAReasonNotPlumbing(t *testing.T) {
	live := context.Background()

	if got := probeErr(live, context.DeadlineExceeded); got.Error() != "timed out" {
		t.Errorf("got %q, want %q", got, "timed out")
	}
	if got := probeErr(live, context.Canceled); got.Error() != "cancelled" {
		t.Errorf("got %q, want %q", got, "cancelled")
	}

	// The ctx carries the verdict even when the error itself does not, which
	// is what a client that wraps its own deadline error looks like.
	expired, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	if got := probeErr(expired, errors.New("read: connection reset by peer")); got.Error() != "timed out" {
		t.Errorf("got %q, want the deadline recognised from the context", got)
	}

	stopped, stop := context.WithCancel(context.Background())
	stop()
	if got := probeErr(stopped, errors.New("read: connection reset by peer")); got.Error() != "cancelled" {
		t.Errorf("got %q, want the cancellation recognised from the context", got)
	}

	// One layer of url.Error is unwrapped, so the message is not "Post
	// \"https://…\": dial tcp …" with the URL said twice.
	inner := errors.New("dial tcp 10.0.0.9:443: connect: connection refused")
	wrapped := &url.Error{Op: "Post", URL: "https://rpc.example.com", Err: inner}
	got := probeErr(live, wrapped)
	if got != inner {
		t.Errorf("got %q, want the inner cause unwrapped", got)
	}
	if strings.Contains(got.Error(), "rpc.example.com") {
		t.Errorf("got %q, want the repeated URL dropped", got)
	}

	// Anything else passes through untouched.
	plain := errors.New("http 502")
	if got := probeErr(live, plain); got != plain {
		t.Errorf("got %q, want the error passed through", got)
	}
}
