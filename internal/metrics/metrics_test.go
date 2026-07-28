package metrics

import (
	"errors"
	"math"
	"os"
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------
// ParseText
// ---------------------------------------------------------------------

func TestParseText_SkipsMetadataAndBlankLines(t *testing.T) {
	in := `# HELP foo_total A thing.
# TYPE foo_total counter

foo_total 1
`
	samples, err := ParseText(strings.NewReader(in))
	if err != nil {
		t.Fatalf("ParseText: %v", err)
	}
	if len(samples) != 1 {
		t.Fatalf("got %d samples, want 1: %+v", len(samples), samples)
	}
	if samples[0].Name != "foo_total" || samples[0].Value != 1 {
		t.Fatalf("got %+v", samples[0])
	}
}

func TestParseText_BareAndLabelled(t *testing.T) {
	in := `bare_metric 42
labelled_metric{k="v",k2="v2"} 123.45
`
	samples, err := ParseText(strings.NewReader(in))
	if err != nil {
		t.Fatalf("ParseText: %v", err)
	}
	if len(samples) != 2 {
		t.Fatalf("got %d samples, want 2", len(samples))
	}

	bare := samples[0]
	if bare.Name != "bare_metric" || bare.Value != 42 || bare.Labels != nil {
		t.Fatalf("bare: got %+v", bare)
	}

	labelled := samples[1]
	if labelled.Name != "labelled_metric" || labelled.Value != 123.45 {
		t.Fatalf("labelled: got %+v", labelled)
	}
	if labelled.Labels["k"] != "v" || labelled.Labels["k2"] != "v2" {
		t.Fatalf("labelled labels: got %+v", labelled.Labels)
	}
}

func TestParseText_LabelEscapes(t *testing.T) {
	// A label value containing an escaped quote, an escaped backslash and
	// an escaped newline, per the Prometheus text-exposition grammar.
	in := `m{a="say \"hi\"",b="back\\slash",c="line1\nline2"} 1`
	samples, err := ParseText(strings.NewReader(in))
	if err != nil {
		t.Fatalf("ParseText: %v", err)
	}
	if len(samples) != 1 {
		t.Fatalf("got %d samples, want 1", len(samples))
	}
	labels := samples[0].Labels
	if got, want := labels["a"], `say "hi"`; got != want {
		t.Errorf("a: got %q, want %q", got, want)
	}
	if got, want := labels["b"], `back\slash`; got != want {
		t.Errorf("b: got %q, want %q", got, want)
	}
	if got, want := labels["c"], "line1\nline2"; got != want {
		t.Errorf("c: got %q, want %q", got, want)
	}
}

func TestParseText_LabelValuesWithDelimiterCharacters(t *testing.T) {
	// Commas, spaces, '=' and '}' inside a quoted label value must not be
	// mistaken for the label-block delimiters that surround them — this is
	// exactly what a regexp built from [^,]* would get wrong.
	in := `m{vendor="a, b=c}, and more",network="evm:1"} 7`
	samples, err := ParseText(strings.NewReader(in))
	if err != nil {
		t.Fatalf("ParseText: %v", err)
	}
	if len(samples) != 1 {
		t.Fatalf("got %d samples, want 1", len(samples))
	}
	if got, want := samples[0].Labels["vendor"], "a, b=c}, and more"; got != want {
		t.Errorf("vendor: got %q, want %q", got, want)
	}
	if got, want := samples[0].Labels["network"], "evm:1"; got != want {
		t.Errorf("network: got %q, want %q", got, want)
	}
	if samples[0].Value != 7 {
		t.Errorf("value: got %v, want 7", samples[0].Value)
	}
}

func TestParseText_ValueForms(t *testing.T) {
	tests := []struct {
		name string
		line string
		want float64
	}{
		{"integer", "m 5", 5},
		{"float", "m 12.5", 12.5},
		{"exponent", "m 1.7e+09", 1.7e+09},
		{"negative exponent", "m 1.5e-03", 1.5e-03},
		{"positive infinity", "m +Inf", math.Inf(1)},
		{"negative infinity", "m -Inf", math.Inf(-1)},
		{"nan", "m NaN", math.NaN()},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			samples, err := ParseText(strings.NewReader(tc.line))
			if err != nil {
				t.Fatalf("ParseText: %v", err)
			}
			if len(samples) != 1 {
				t.Fatalf("got %d samples, want 1", len(samples))
			}
			got := samples[0].Value
			if math.IsNaN(tc.want) {
				if !math.IsNaN(got) {
					t.Errorf("got %v, want NaN", got)
				}
				return
			}
			if got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestParseText_MalformedLineNamesLineNumberAndText(t *testing.T) {
	in := "foo_total 1\nnot a valid $$ line\nbar_total 2\n"
	_, err := ParseText(strings.NewReader(in))
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "line 2") {
		t.Errorf("error does not name the line number: %q", msg)
	}
	if !strings.Contains(msg, "not a valid $$ line") {
		t.Errorf("error does not quote the offending text: %q", msg)
	}
}

func TestParseText_MalformedLabelBlock(t *testing.T) {
	tests := []struct {
		name string
		line string
	}{
		{"unterminated label block", `m{a="v" 1`},
		{"missing equals", `m{a} 1`},
		{"unquoted value", `m{a=v} 1`},
		{"unterminated quoted value", `m{a="v} 1`},
		{"dangling escape", `m{a="v\`},
		{"unknown escape", `m{a="v\x"} 1`},
		{"bad delimiter after value", `m{a="v"x} 1`},
		{"no value at all", `m{a="v"}`},
		{"no metric name", `{a="v"} 1`},
		{"nothing after open brace", `m{`},
		{"end of line right after closing quote", `m{a="v"`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseText(strings.NewReader(tc.line))
			if err == nil {
				t.Fatalf("expected an error for %q, got nil", tc.line)
			}
		})
	}
}

func TestParseText_EmptyLabelBlock(t *testing.T) {
	samples, err := ParseText(strings.NewReader(`m{} 1`))
	if err != nil {
		t.Fatalf("ParseText: %v", err)
	}
	if len(samples) != 1 || samples[0].Name != "m" || samples[0].Value != 1 {
		t.Fatalf("got %+v", samples)
	}
	if len(samples[0].Labels) != 0 {
		t.Errorf("got labels %+v, want empty", samples[0].Labels)
	}
}

func TestParseText_LabelBlockLeadingWhitespace(t *testing.T) {
	// Not a shape eRPC's own dump uses (its labels are never
	// space-padded), but tolerating it costs nothing and the leading-space
	// skip is otherwise dead code.
	samples, err := ParseText(strings.NewReader(`m{ a="v"} 1`))
	if err != nil {
		t.Fatalf("ParseText: %v", err)
	}
	if len(samples) != 1 || samples[0].Labels["a"] != "v" {
		t.Fatalf("got %+v", samples)
	}
}

// errReader always fails, to exercise ParseText's handling of a scan
// error distinct from a malformed line — bufio.Scanner surfaces I/O
// failures through Err() after Scan() returns false, not as a line.
type errReader struct{}

var errBoom = errors.New("boom")

func (errReader) Read([]byte) (int, error) { return 0, errBoom }

func TestParseText_ReaderError(t *testing.T) {
	_, err := ParseText(errReader{})
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if !errors.Is(err, errBoom) {
		t.Errorf("got %v, want it to wrap %v", err, errBoom)
	}
}

// ---------------------------------------------------------------------
// FromSamples
// ---------------------------------------------------------------------

func TestFromSamples_RealFixture(t *testing.T) {
	f, err := os.Open("testdata/erpc.txt")
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	defer f.Close()

	samples, err := ParseText(f)
	if err != nil {
		t.Fatalf("ParseText: %v", err)
	}

	traffic := FromSamples(samples, "main")

	// process_start_time_seconds = 1.78519881873e+09 in the fixture: 2026-07-28
	// 00:33:38 UTC and change. float64 has roughly nanosecond precision at
	// this magnitude, so compare with a small tolerance rather than exact
	// equality.
	wantSince := time.Unix(1785198818, 730000000).UTC()
	if diff := traffic.Since.Sub(wantSince); diff < -time.Microsecond || diff > time.Microsecond {
		t.Errorf("Since: got %v, want %v (diff %v)", traffic.Since, wantSince, diff)
	}

	// The fixture's erpc_network_successful_request_total and
	// erpc_network_request_received_total families only ever mention
	// network="evm:369" — the curl-driven eth_blockNumber traffic. Chain 1
	// never appears in either family, even though it carries hundreds of
	// erpc_upstream_request_total / erpc_upstream_attempt_outcome_total
	// hits from the state poller. If FromSamples were reading the wrong
	// metric family, evm:1 would show up here with a huge count; it must
	// not show up at all.
	if len(traffic.Networks) != 1 {
		t.Fatalf("got %d networks, want 1: %+v", len(traffic.Networks), traffic.Networks)
	}

	nt := traffic.Networks[0]
	if nt.Network != "evm:369" {
		t.Errorf("Network: got %q, want evm:369", nt.Network)
	}
	if nt.ChainID != 369 {
		t.Errorf("ChainID: got %d, want 369", nt.ChainID)
	}
	if nt.Received != 5 {
		t.Errorf("Received: got %v, want 5", nt.Received)
	}
	if len(nt.Upstreams) != 1 {
		t.Fatalf("got %d upstreams, want 1: %+v", len(nt.Upstreams), nt.Upstreams)
	}
	up := nt.Upstreams[0]
	if up.Upstream != "public-369-1" {
		t.Errorf("Upstream: got %q, want public-369-1", up.Upstream)
	}
	// The genuine client-facing count (5), not the state poller's
	// eth_getBlockByNumber count against the same upstream (414, all
	// tagged network="n/a" in erpc_upstream_attempt_outcome_total).
	if up.Succeeded != 5 {
		t.Errorf("Succeeded: got %v, want 5 (poller traffic must not leak in)", up.Succeeded)
	}
}

func TestFromSamples_IgnoresUpstreamScopedPollerFamily(t *testing.T) {
	// A synthetic reading that pairs a small, genuine erpc_network_*
	// reading with a much larger erpc_upstream_request_total reading for
	// the same upstream — the shape the state poller produces on a real
	// gateway. FromSamples must report only the small, genuine number.
	samples := []Sample{
		{Name: metricReceived, Labels: map[string]string{"network": "evm:1", "project": "main"}, Value: 5},
		{Name: metricSuccessful, Labels: map[string]string{"network": "evm:1", "project": "main", "upstream": "public-1-1"}, Value: 5},
		{Name: "erpc_upstream_request_total", Labels: map[string]string{"network": "n/a", "project": "main", "upstream": "public-1-1"}, Value: 354},
		{Name: "erpc_upstream_attempt_outcome_total", Labels: map[string]string{"network": "n/a", "project": "main", "upstream": "public-1-1", "outcome": "success"}, Value: 354},
	}

	traffic := FromSamples(samples, "main")
	if len(traffic.Networks) != 1 {
		t.Fatalf("got %d networks, want 1", len(traffic.Networks))
	}
	nt := traffic.Networks[0]
	if len(nt.Upstreams) != 1 || nt.Upstreams[0].Succeeded != 5 {
		t.Fatalf("got %+v, want a single upstream with Succeeded=5", nt.Upstreams)
	}
}

func TestFromSamples_DropsNoNetworkLabel(t *testing.T) {
	samples := []Sample{
		{Name: metricReceived, Labels: map[string]string{"network": "n/a", "project": "main"}, Value: 99},
		{Name: metricSuccessful, Labels: map[string]string{"network": "n/a", "project": "main", "upstream": "devnet"}, Value: 99},
		{Name: metricReceived, Labels: map[string]string{"network": "evm:369", "project": "main"}, Value: 1},
		{Name: metricSuccessful, Labels: map[string]string{"network": "evm:369", "project": "main", "upstream": "public-369-1"}, Value: 1},
	}
	traffic := FromSamples(samples, "main")
	if len(traffic.Networks) != 1 {
		t.Fatalf("got %d networks, want 1 (n/a must be dropped): %+v", len(traffic.Networks), traffic.Networks)
	}
	if traffic.Networks[0].Network != "evm:369" {
		t.Errorf("got network %q, want evm:369", traffic.Networks[0].Network)
	}
}

func TestFromSamples_ProjectFilter(t *testing.T) {
	samples := []Sample{
		{Name: metricReceived, Labels: map[string]string{"network": "evm:1", "project": "main"}, Value: 5},
		{Name: metricSuccessful, Labels: map[string]string{"network": "evm:1", "project": "main", "upstream": "public-1-1"}, Value: 5},
		{Name: metricReceived, Labels: map[string]string{"network": "evm:1", "project": "other"}, Value: 999},
		{Name: metricSuccessful, Labels: map[string]string{"network": "evm:1", "project": "other", "upstream": "public-1-1"}, Value: 999},
	}

	filtered := FromSamples(samples, "main")
	if len(filtered.Networks) != 1 || filtered.Networks[0].Received != 5 {
		t.Fatalf("filtered: got %+v, want Received=5", filtered.Networks)
	}
	if filtered.Networks[0].Upstreams[0].Succeeded != 5 {
		t.Fatalf("filtered: got Succeeded=%v, want 5", filtered.Networks[0].Upstreams[0].Succeeded)
	}

	unfiltered := FromSamples(samples, "")
	if len(unfiltered.Networks) != 1 {
		t.Fatalf("unfiltered: got %d networks, want 1", len(unfiltered.Networks))
	}
	if got, want := unfiltered.Networks[0].Received, 5.0+999.0; got != want {
		t.Errorf("unfiltered Received: got %v, want %v (both projects summed)", got, want)
	}
}

func TestFromSamples_DeterministicOrdering(t *testing.T) {
	// Networks arrive out of chain-id order, with a non-EVM network (chain
	// id 0) mixed in, and upstreams within a network also out of order.
	samples := []Sample{
		{Name: metricReceived, Labels: map[string]string{"network": "evm:943", "project": "main"}, Value: 1},
		{Name: metricSuccessful, Labels: map[string]string{"network": "evm:943", "project": "main", "upstream": "z-upstream"}, Value: 1},
		{Name: metricSuccessful, Labels: map[string]string{"network": "evm:943", "project": "main", "upstream": "a-upstream"}, Value: 1},

		{Name: metricReceived, Labels: map[string]string{"network": "solana", "project": "main"}, Value: 1},
		{Name: metricSuccessful, Labels: map[string]string{"network": "solana", "project": "main", "upstream": "sol-1"}, Value: 1},

		// A second ChainID-0 (non-EVM) network, listed after "solana" in
		// the samples, to exercise the network-name tie-break for equal
		// chain ids rather than relying on chain id alone.
		{Name: metricReceived, Labels: map[string]string{"network": "aptos", "project": "main"}, Value: 1},
		{Name: metricSuccessful, Labels: map[string]string{"network": "aptos", "project": "main", "upstream": "apt-1"}, Value: 1},

		{Name: metricReceived, Labels: map[string]string{"network": "evm:1", "project": "main"}, Value: 1},
		{Name: metricSuccessful, Labels: map[string]string{"network": "evm:1", "project": "main", "upstream": "public-1-1"}, Value: 1},
	}

	traffic := FromSamples(samples, "main")
	if len(traffic.Networks) != 4 {
		t.Fatalf("got %d networks, want 4", len(traffic.Networks))
	}
	// The two ChainID-0 networks sort by name ahead of any positive chain
	// id, and between themselves alphabetically: "aptos" before "solana".
	wantNetworks := []string{"aptos", "solana", "evm:1", "evm:943"}
	for i, want := range wantNetworks {
		if traffic.Networks[i].Network != want {
			t.Errorf("Networks[%d]: got %q, want %q", i, traffic.Networks[i].Network, want)
		}
	}

	chain943 := traffic.Networks[3]
	if len(chain943.Upstreams) != 2 {
		t.Fatalf("got %d upstreams, want 2", len(chain943.Upstreams))
	}
	if chain943.Upstreams[0].Upstream != "a-upstream" || chain943.Upstreams[1].Upstream != "z-upstream" {
		t.Errorf("upstreams not sorted: got %+v", chain943.Upstreams)
	}
}

func TestFromSamples_SkipsSuccessfulSampleWithoutUpstreamLabel(t *testing.T) {
	// Defensive case: erpc_network_successful_request_total always carries
	// an upstream label in practice, but a sample missing one cannot be
	// attributed to anyone and must not panic or invent a placeholder
	// upstream.
	samples := []Sample{
		{Name: metricReceived, Labels: map[string]string{"network": "evm:1", "project": "main"}, Value: 1},
		{Name: metricSuccessful, Labels: map[string]string{"network": "evm:1", "project": "main"}, Value: 1},
	}
	traffic := FromSamples(samples, "main")
	if len(traffic.Networks) != 1 {
		t.Fatalf("got %d networks, want 1", len(traffic.Networks))
	}
	if len(traffic.Networks[0].Upstreams) != 0 {
		t.Errorf("got upstreams %+v, want none", traffic.Networks[0].Upstreams)
	}
}

func TestFromSamples_ChainIDParsing(t *testing.T) {
	tests := []struct {
		network string
		want    int
	}{
		{"evm:1", 1},
		{"evm:369", 369},
		{"evm:abc", 0}, // not a number
		{"evm:0", 0},   // not a valid chain id
		{"evm:-5", 0},  // not a valid chain id
		{"solana", 0},  // not the evm: shape at all
	}
	for _, tc := range tests {
		t.Run(tc.network, func(t *testing.T) {
			samples := []Sample{
				{Name: metricReceived, Labels: map[string]string{"network": tc.network, "project": "main"}, Value: 1},
			}
			traffic := FromSamples(samples, "main")
			if len(traffic.Networks) != 1 {
				t.Fatalf("got %d networks, want 1", len(traffic.Networks))
			}
			if got := traffic.Networks[0].ChainID; got != tc.want {
				t.Errorf("ChainID for %q: got %d, want %d", tc.network, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------
// Shares
// ---------------------------------------------------------------------

func TestShares_SingleLocalUpstreamCarryingEverything(t *testing.T) {
	nt := NetworkTraffic{
		Network: "evm:1",
		Upstreams: []UpstreamTraffic{
			{Upstream: "local-node", Succeeded: 100},
		},
	}
	intents := []Intent{{Upstream: "local-node", Local: true}}

	shares := Shares(nt, intents)
	if len(shares) != 1 {
		t.Fatalf("got %d shares, want 1", len(shares))
	}
	s := shares[0]
	if s.Actual != 1 || s.Intended != 1 || s.Diverged {
		t.Errorf("got %+v, want Actual=1 Intended=1 Diverged=false", s)
	}
}

func TestShares_ZeroTotalIsNotDivergent(t *testing.T) {
	nt := NetworkTraffic{
		Network: "evm:1",
		Upstreams: []UpstreamTraffic{
			{Upstream: "local-node", Succeeded: 0},
			{Upstream: "fallback-1", Succeeded: 0},
		},
	}
	intents := []Intent{
		{Upstream: "local-node", Local: true},
		{Upstream: "fallback-1", Local: false},
	}

	shares := Shares(nt, intents)
	for _, s := range shares {
		if s.Actual != 0 {
			t.Errorf("%s: Actual = %v, want 0", s.Upstream, s.Actual)
		}
		if s.Diverged {
			t.Errorf("%s: Diverged = true, want false for a network with zero total successes", s.Upstream)
		}
	}
}

func TestShares_IntentWithoutCounters(t *testing.T) {
	// local-node is configured but has never appeared in
	// erpc_network_successful_request_total: it should still be reported,
	// with Succeeded and Actual both 0, and — because a healthy config
	// intends it to carry everything — Diverged should be true once
	// something else is carrying all the real traffic.
	nt := NetworkTraffic{
		Network: "evm:1",
		Upstreams: []UpstreamTraffic{
			{Upstream: "fallback-1", Succeeded: 50},
		},
	}
	intents := []Intent{
		{Upstream: "local-node", Local: true},
		{Upstream: "fallback-1", Local: false},
	}

	shares := Shares(nt, intents)
	byName := sharesByName(shares)

	local := byName["local-node"]
	if local.Succeeded != 0 || local.Actual != 0 {
		t.Errorf("local-node: got %+v, want Succeeded=0 Actual=0", local)
	}
	if local.Intended != 1 {
		t.Errorf("local-node: Intended = %v, want 1 (sole local upstream)", local.Intended)
	}
	if !local.Diverged {
		t.Errorf("local-node: Diverged = false, want true (intended 100%%, carrying 0%%): %+v", local)
	}

	fallback := byName["fallback-1"]
	if fallback.Intended != 0 {
		t.Errorf("fallback-1: Intended = %v, want 0", fallback.Intended)
	}
	if !fallback.Diverged {
		t.Errorf("fallback-1: Diverged = false, want true (intended 0%%, carrying 100%%): %+v", fallback)
	}
}

func TestShares_CountersWithoutIntent(t *testing.T) {
	// "ghost-upstream" carries traffic but was removed from the routing
	// config; it must still be reported, with Intended forced to 0, and
	// able to diverge.
	nt := NetworkTraffic{
		Network: "evm:1",
		Upstreams: []UpstreamTraffic{
			{Upstream: "local-node", Succeeded: 0},
			{Upstream: "ghost-upstream", Succeeded: 100},
		},
	}
	intents := []Intent{
		{Upstream: "local-node", Local: true},
	}

	shares := Shares(nt, intents)
	byName := sharesByName(shares)

	ghost, ok := byName["ghost-upstream"]
	if !ok {
		t.Fatal("ghost-upstream missing from Shares output")
	}
	if ghost.Intended != 0 {
		t.Errorf("ghost-upstream: Intended = %v, want 0", ghost.Intended)
	}
	if ghost.Actual != 1 {
		t.Errorf("ghost-upstream: Actual = %v, want 1", ghost.Actual)
	}
	if !ghost.Diverged {
		t.Error("ghost-upstream: Diverged = false, want true")
	}
}

func TestShares_NoLocalUpstreamSpreadsEvenlyAcrossFallbacks(t *testing.T) {
	nt := NetworkTraffic{
		Network: "evm:369",
		Upstreams: []UpstreamTraffic{
			{Upstream: "fallback-a", Succeeded: 50},
			{Upstream: "fallback-b", Succeeded: 50},
		},
	}
	intents := []Intent{
		{Upstream: "fallback-a", Local: false},
		{Upstream: "fallback-b", Local: false},
	}

	shares := Shares(nt, intents)
	for _, s := range shares {
		if s.Intended != 0.5 {
			t.Errorf("%s: Intended = %v, want 0.5 (no local upstream, spread evenly)", s.Upstream, s.Intended)
		}
		if s.Actual != 0.5 {
			t.Errorf("%s: Actual = %v, want 0.5", s.Upstream, s.Actual)
		}
		if s.Diverged {
			t.Errorf("%s: Diverged = true, want false (actual matches the evenly spread intent)", s.Upstream)
		}
	}
}

func TestShares_DivergenceThresholdBoundary(t *testing.T) {
	// One local upstream (Intended=1) carrying exactly 80% of traffic sits
	// exactly at the threshold (|0.8-1.0| == 0.20) and must NOT be flagged
	// ("> 0.20", not ">="); carrying 79% must be.
	makeNT := func(localPct float64) NetworkTraffic {
		return NetworkTraffic{
			Network: "evm:1",
			Upstreams: []UpstreamTraffic{
				{Upstream: "local-node", Succeeded: localPct * 100},
				{Upstream: "fallback-1", Succeeded: (1 - localPct) * 100},
			},
		}
	}
	intents := []Intent{
		{Upstream: "local-node", Local: true},
		{Upstream: "fallback-1", Local: false},
	}

	atBoundary := sharesByName(Shares(makeNT(0.80), intents))["local-node"]
	if atBoundary.Diverged {
		t.Errorf("at exactly the 0.20 boundary, got Diverged=true, want false: %+v", atBoundary)
	}

	pastBoundary := sharesByName(Shares(makeNT(0.79), intents))["local-node"]
	if !pastBoundary.Diverged {
		t.Errorf("past the 0.20 boundary, got Diverged=false, want true: %+v", pastBoundary)
	}
}

func TestShares_SortedByUpstream(t *testing.T) {
	nt := NetworkTraffic{
		Network: "evm:1",
		Upstreams: []UpstreamTraffic{
			{Upstream: "z-upstream", Succeeded: 1},
			{Upstream: "a-upstream", Succeeded: 1},
		},
	}
	intents := []Intent{
		{Upstream: "z-upstream", Local: false},
		{Upstream: "a-upstream", Local: false},
		{Upstream: "m-upstream", Local: false},
	}
	shares := Shares(nt, intents)
	for i := 1; i < len(shares); i++ {
		if shares[i-1].Upstream >= shares[i].Upstream {
			t.Fatalf("not sorted: %+v", shares)
		}
	}
}

func sharesByName(shares []Share) map[string]Share {
	m := make(map[string]Share, len(shares))
	for _, s := range shares {
		m[s.Upstream] = s
	}
	return m
}
