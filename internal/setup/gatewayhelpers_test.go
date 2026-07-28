package setup

import (
	"strings"
	"testing"

	"github.com/valve-tech/valve-node-app/internal/catalog"
	"github.com/valve-tech/valve-node-app/internal/ops"
)

// The small pure helpers underneath the gateway plan. They are worth pinning
// directly rather than only through a provision, because each one encodes a
// decision that is invisible at the call site and expensive to get wrong: a
// file name two gateways would share, an address that is not a destination, a
// port that must be zero rather than a default.

// A gateway id becomes a FILE NAME here, and two gateways sharing a Caddyfile
// means each provision silently overwrites the other's hostname. The default
// id keeps the bare name so an existing install's file — and the bind-mount
// path baked into its running container — is the one that gets rewritten.
func TestTLSFileName_ScopedPerGatewayExceptTheDefault(t *testing.T) {
	for _, tc := range []struct {
		id   string
		want string
	}{
		{id: "", want: "Caddyfile"},
		{id: ops.DefaultGatewayID, want: "Caddyfile"},
		{id: "  " + ops.DefaultGatewayID + "  ", want: "Caddyfile"},
		{id: "edge", want: "Caddyfile-edge"},
		// The id is already validated at the API, but this is what renders a
		// path: anything that could escape the directory or break the shell
		// becomes a dash rather than reaching the filesystem.
		{id: "../escape", want: "Caddyfile-..-escape"},
		{id: "a b", want: "Caddyfile-a-b"},
	} {
		p := &gatewayPlan{id: tc.id}
		if got := p.tlsFileName("Caddyfile", ""); got != tc.want {
			t.Errorf("id %q: got %q, want %q", tc.id, got, tc.want)
		}
	}
}

func TestSanitizeGatewayID_KeepsOnlyWhatAFileNameTakes(t *testing.T) {
	got := sanitizeGatewayID("Ab9_.-/\\ :étoile")
	if strings.ContainsAny(got, "/\\ :é") {
		t.Errorf("got %q — anything that is not a plain identifier character must be replaced", got)
	}
	if !strings.HasPrefix(got, "Ab9_.-") {
		t.Errorf("got %q — the safe characters must survive unchanged", got)
	}
}

// probeHost turns a BIND address into something connectable. A wildcard names
// every interface but is not itself a destination — macOS refuses a connect to
// 0.0.0.0 outright — and an IPv6 literal has to be bracketed or the port
// separator is ambiguous.
func TestProbeHost_WildcardsBecomeLoopbackAndIPv6IsBracketed(t *testing.T) {
	for in, want := range map[string]string{
		"":            "127.0.0.1",
		"0.0.0.0":     "127.0.0.1",
		"::":          "[::1]",
		"::0":         "[::1]",
		"127.0.0.1":   "127.0.0.1",
		"100.64.0.7":  "100.64.0.7",
		"fd00::1":     "[fd00::1]",
		"[fd00::1]":   "[fd00::1]",
		"example.com": "example.com",
	} {
		if got := probeHost(in); got != want {
			t.Errorf("probeHost(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestParseHexChainID(t *testing.T) {
	for in, want := range map[string]int{
		"0x171": 369,
		"0X171": 369,
		" 0x1 ": 1,
		"539":   1337,
		"0x0":   0,
	} {
		got, err := parseHexChainID(in)
		if err != nil {
			t.Errorf("parseHexChainID(%q): %v", in, err)
			continue
		}
		if got != want {
			t.Errorf("parseHexChainID(%q) = %d, want %d", in, got, want)
		}
	}
	// A node that answers something that is not a quantity is a node whose
	// chain id we do NOT know, which must not collapse to 0 — 0 would compare
	// equal to an unset expectation and read as agreement.
	for _, bad := range []string{"", "nope", "0x", "0xzz"} {
		if _, err := parseHexChainID(bad); err == nil {
			t.Errorf("parseHexChainID(%q) must fail rather than return a number", bad)
		}
	}
}

// The metrics port must be ZERO when the counters are off, not the default.
// It is used to render a -p mapping, so a default here would publish a port
// for a listener that is not running — and, worse, would collide with the
// gateway next door that does have its counters on.
func TestMetricsHostPort_ZeroWhenTheCountersAreOff(t *testing.T) {
	on := &gatewayPlan{gw: catalog.GatewayConfig{MetricsPort: 4455}}
	if got := on.metricsHostPort(); got != 4455 {
		t.Errorf("counters on: got %d, want the configured port", got)
	}

	deflt := &gatewayPlan{gw: catalog.GatewayConfig{}}
	if got := deflt.metricsHostPort(); got == 0 {
		t.Error("counters default to ON, so a zero-value config must still publish a port")
	}

	off := &gatewayPlan{gw: catalog.GatewayConfig{MetricsOff: true, MetricsPort: 4455}}
	if got := off.metricsHostPort(); got != 0 {
		t.Errorf("counters off: got %d, want 0 — publishing a port for a listener that is not running is worse than not publishing", got)
	}
}
