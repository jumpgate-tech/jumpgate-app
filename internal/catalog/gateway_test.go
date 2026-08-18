package catalog

import (
	"strings"
	"testing"
)

// The shape asserted here was verified empirically against eRPC on the
// valve-ws branch: a config of this form served chain 369 and 943 from one
// instance over both HTTP and WebSocket. The repo's erpc.dist.yaml is stale
// and is deliberately not used as the reference.

func TestRenderGatewayConfig_MultiChain(t *testing.T) {
	g := GatewayConfig{
		Networks: []GatewayNetwork{
			{ChainID: 369, Upstreams: []GatewayUpstream{
				{ID: "local-369", Endpoint: "http://127.0.0.1:8545", Local: true, RecentOnly: true},
				{ID: "pls-ws", Endpoint: "wss://pulsechain-rpc.publicnode.com"},
			}},
			{ChainID: 943, Upstreams: []GatewayUpstream{
				{ID: "t4", Endpoint: "https://rpc.v4.testnet.pulsechain.com"},
			}},
		},
	}
	cfg, err := RenderGatewayConfig(g)
	if err != nil {
		t.Fatalf("RenderGatewayConfig: %v", err)
	}

	// Both chains must appear as networks — this is what makes eRPC route
	// /main/evm/<id> for each. Upstreams alone do not declare a network.
	for _, want := range []string{"chainId: 369", "chainId: 943", "architecture: evm"} {
		if !strings.Contains(cfg, want) {
			t.Errorf("missing %q:\n%s", want, cfg)
		}
	}
	if got := strings.Count(cfg, "architecture: evm"); got != 2 {
		t.Errorf("expected 2 networks, got %d:\n%s", got, cfg)
	}
	// Upstreams are a flat list at project level, each tagged with its
	// chain — not nested under their network.
	if got := strings.Count(cfg, "type: evm"); got != 3 {
		t.Errorf("expected 3 upstreams, got %d:\n%s", got, cfg)
	}
	// A ws:// endpoint is an ordinary upstream; eRPC infers WebSocket
	// support from the scheme, so no extra key should appear.
	if !strings.Contains(cfg, `endpoint: "wss://pulsechain-rpc.publicnode.com"`) {
		t.Errorf("ws upstream missing:\n%s", cfg)
	}
}

func TestRenderGatewayConfig_LocalPreferredFallbackDeprioritised(t *testing.T) {
	g := GatewayConfig{Networks: []GatewayNetwork{{ChainID: 369, Upstreams: []GatewayUpstream{
		{ID: "mine", Endpoint: "http://127.0.0.1:8545", Local: true},
		{ID: "public", Endpoint: "https://rpc.pulsechain.com"},
	}}}}
	cfg, err := RenderGatewayConfig(g)
	if err != nil {
		t.Fatalf("RenderGatewayConfig: %v", err)
	}
	// Exactly one upstream is deprioritised: the one we don't run.
	if got := strings.Count(cfg, "tier:fallback"); got != 1 {
		t.Errorf("expected exactly 1 fallback tag, got %d:\n%s", got, cfg)
	}
	if got := strings.Count(cfg, "overall: 0.2"); got != 1 {
		t.Errorf("expected exactly 1 score multiplier, got %d:\n%s", got, cfg)
	}
	mine := strings.Index(cfg, "id: mine")
	pub := strings.Index(cfg, "id: public")
	if mine < 0 || pub < 0 || mine > pub {
		t.Errorf("local upstream should be listed first:\n%s", cfg)
	}
}

// A gateway with no local upstreams is the "can't host a node, still want my
// own RPC" case. It needs no special representation and must render fine.
//
// Neither upstream carries tier:fallback: with nothing local on this chain,
// both public endpoints are the only paths there are, and deprioritising the
// only paths that exist would be exactly backwards.
func TestRenderGatewayConfig_NoLocalUpstreams(t *testing.T) {
	g := GatewayConfig{Networks: []GatewayNetwork{{ChainID: 1, Upstreams: []GatewayUpstream{
		{Endpoint: "https://ethereum-rpc.publicnode.com"},
		{Endpoint: "wss://ethereum-rpc.publicnode.com"},
	}}}}
	cfg, err := RenderGatewayConfig(g)
	if err != nil {
		t.Fatalf("RenderGatewayConfig: %v", err)
	}
	if got := strings.Count(cfg, "tier:fallback"); got != 0 {
		t.Errorf("with no local upstream, nothing should be tagged fallback, got %d:\n%s", got, cfg)
	}
	// Ids are generated when the caller supplies none: readable (provider +
	// protocol) and unique (chain + position). Same provider, so the protocol
	// and position are what keep the two apart.
	for _, want := range []string{"id: 1-publicnode-http-1", "id: 1-publicnode-ws-2"} {
		if !strings.Contains(cfg, want) {
			t.Errorf("missing generated id %q:\n%s", want, cfg)
		}
	}
}

// tier:fallback at 0.2 tells eRPC to avoid an upstream. That is right when a
// local node serves the chain and wrong when the "fallback" is the only path
// there is — the gateway would be de-prioritising the one thing that can answer.
func TestRenderGatewayConfig_PublicOnlyChainIsNotAFallback(t *testing.T) {
	// 1337 is listed FIRST on purpose: "pub" is a prefix of "pub2", so this
	// ordering is what makes section's line anchor load-bearing rather than
	// incidentally correct. Rendered the other way round, an unanchored lookup
	// for "pub" finds pub2 and both assertions below inspect the wrong block
	// while still passing.
	g := GatewayConfig{Port: 4000, Networks: []GatewayNetwork{
		{ChainID: 1337, Upstreams: []GatewayUpstream{
			// RenderGatewayConfig requires a resolved endpoint on every
			// upstream, managed kinds included (see the Endpoint field's
			// doc comment) — normally filled in by the caller before
			// rendering. Supplied here directly since this test renders
			// straight from a hand-built GatewayConfig.
			{ID: "devnet", Kind: UpstreamManagedDevnet, TargetID: "local", Local: true, Endpoint: "http://127.0.0.1:8546"},
			{ID: "pub2", Kind: UpstreamExternal, Endpoint: "https://backup.example"},
		}},
		{ChainID: 1, Upstreams: []GatewayUpstream{
			{ID: "pub", Kind: UpstreamExternal, Endpoint: "https://eth.example"},
		}},
	}}

	out, err := RenderGatewayConfig(g)
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	// Chain 1 has nothing local, so its only upstream must go in at full weight.
	pub := section(t, out, "pub")
	if strings.Contains(pub, "tier:fallback") {
		t.Errorf("a chain with no local node has no fallback tier:\n%s", pub)
	}
	// Chain 1337 does have a local node, so the public one stays a fallback.
	pub2 := section(t, out, "pub2")
	if !strings.Contains(pub2, "tier:fallback") {
		t.Errorf("a public upstream beside a local node is still a fallback:\n%s", pub2)
	}
}

// section returns the rendered block for the upstream named id: the
// "- id: <name>" line through to the next "- id:" line or the end of the
// string. Used to make assertions about one upstream without them being
// confused by a sibling upstream's tags.
//
// The marker carries the newline that ENDS the id line, because "- id: pub" is
// also a prefix of "- id: pub2": without it, asking for "pub" returns pub2's
// block whenever pub2 renders first, and the assertion silently inspects the
// wrong upstream instead of failing.
func section(t *testing.T, cfg, id string) string {
	t.Helper()
	marker := "- id: " + id + "\n"
	start := strings.Index(cfg, marker)
	if start < 0 {
		t.Fatalf("no upstream %q in:\n%s", id, cfg)
	}
	rest := cfg[start+len(marker):]
	if next := strings.Index(rest, "- id:"); next >= 0 {
		return cfg[start : start+len(marker)+next]
	}
	return cfg[start:]
}

func TestRenderGatewayConfig_Defaults(t *testing.T) {
	g := GatewayConfig{Networks: []GatewayNetwork{{ChainID: 369, Upstreams: []GatewayUpstream{
		{Endpoint: "https://rpc.pulsechain.com"},
	}}}}
	cfg, err := RenderGatewayConfig(g)
	if err != nil {
		t.Fatalf("RenderGatewayConfig: %v", err)
	}
	for _, want := range []string{`httpHostV4: "127.0.0.1"`, "httpPortV4: 4000", "id: main"} {
		if !strings.Contains(cfg, want) {
			t.Errorf("missing default %q:\n%s", want, cfg)
		}
	}
	if got := g.PathFor(369); got != "/main/evm/369" {
		t.Errorf("PathFor = %q, want /main/evm/369", got)
	}
}

func TestRenderGatewayConfig_Rejects(t *testing.T) {
	tests := []struct {
		name string
		g    GatewayConfig
		want string
	}{
		{
			name: "no networks",
			g:    GatewayConfig{},
			want: "no networks",
		},
		{
			name: "chain with no upstreams",
			g:    GatewayConfig{Networks: []GatewayNetwork{{ChainID: 369}}},
			want: "no upstreams",
		},
		{
			name: "invalid chain id",
			g: GatewayConfig{Networks: []GatewayNetwork{{ChainID: 0, Upstreams: []GatewayUpstream{
				{Endpoint: "https://x.example"},
			}}}},
			want: "invalid chain id",
		},
		{
			name: "duplicate chain",
			g: GatewayConfig{Networks: []GatewayNetwork{
				{ChainID: 369, Upstreams: []GatewayUpstream{{Endpoint: "https://a.example"}}},
				{ChainID: 369, Upstreams: []GatewayUpstream{{Endpoint: "https://b.example"}}},
			}},
			want: "listed twice",
		},
		{
			// eRPC keys upstreams by id, so a duplicate silently shadows
			// rather than failing loudly. Catch it at render time.
			name: "duplicate upstream id",
			g: GatewayConfig{Networks: []GatewayNetwork{{ChainID: 369, Upstreams: []GatewayUpstream{
				{ID: "same", Endpoint: "https://a.example"},
				{ID: "same", Endpoint: "https://b.example"},
			}}}},
			want: "duplicate upstream id",
		},
		{
			name: "empty endpoint",
			g: GatewayConfig{Networks: []GatewayNetwork{{ChainID: 369, Upstreams: []GatewayUpstream{
				{Endpoint: "  "},
			}}}},
			want: "no endpoint",
		},
		{
			name: "unsupported scheme",
			g: GatewayConfig{Networks: []GatewayNetwork{{ChainID: 369, Upstreams: []GatewayUpstream{
				{Endpoint: "ipc:///tmp/geth.ipc"},
			}}}},
			want: "must be http(s):// or ws(s)://",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := RenderGatewayConfig(tc.g)
			if err == nil {
				t.Fatalf("expected an error")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error = %q, want it to contain %q", err, tc.want)
			}
		})
	}
}

func TestGatewayForWire(t *testing.T) {
	w := WireConfig{ChainID: 369, ExecID: "reth", BeaconID: "lighthouse-pulse",
		ERPCEnabled: true, ERPCUpstreams: []string{"https://rpc.pulsechain.com"}}
	g := GatewayForWire(w)
	if len(g.Networks) != 1 || g.Networks[0].ChainID != 369 {
		t.Fatalf("expected a single 369 network, got %+v", g.Networks)
	}
	ups := g.Networks[0].Upstreams
	if len(ups) != 2 {
		t.Fatalf("expected node + 1 fallback, got %d", len(ups))
	}
	if !ups[0].Local {
		t.Errorf("the node itself should be the local upstream: %+v", ups[0])
	}
	if !ups[0].RecentOnly {
		t.Errorf("a non-archive node should be bounded to recent history: %+v", ups[0])
	}
	if ups[1].Local {
		t.Errorf("configured upstreams are fallbacks: %+v", ups[1])
	}
}

// ---------------------------------------------------------------------
// TLS: the gzip constraint
// ---------------------------------------------------------------------

// eRPC a7a53ec2 skips its gzip wrapper on upgrade requests, so a fronted
// gateway no longer has to disable compression to keep eth_subscribe working.
// Both fronted and unfronted gateways now render the same way here.
func TestRenderGatewayConfig_GzipNeverDisabled(t *testing.T) {
	g := GatewayConfig{
		Networks: []GatewayNetwork{{ChainID: 1337, Upstreams: []GatewayUpstream{
			{ID: "devnet", Endpoint: "http://valve-node-app-devnet:8545", Local: true},
		}}},
	}

	plain, err := RenderGatewayConfig(g)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if strings.Contains(plain, "enableGzip") {
		// Nothing is inserting the header, so the compression is free.
		t.Errorf("an unfronted gateway must keep eRPC's own default:\n%s", plain)
	}

	g.TLS = &GatewayTLS{Enabled: true, Hostname: "gw.example"}
	fronted, err := RenderGatewayConfig(g)
	if err != nil {
		t.Fatalf("render fronted: %v", err)
	}
	if strings.Contains(fronted, "enableGzip") {
		// eRPC skips its gzip wrapper on upgrade requests as of a7a53ec2, so a
		// fronted gateway no longer has to trade away response compression to
		// keep eth_subscribe working.
		t.Errorf("a fronted gateway must no longer disable gzip:\n%s", fronted)
	}
	if !g.Fronted() {
		t.Error("Fronted must be derived from the TLS settings, not stored beside them")
	}
}

// A disabled TLS block is kept rather than deleted, so turning HTTPS off and on
// does not lose the hostname — but it must not render as fronted.
func TestGatewayConfig_DisabledTLSIsNotFronted(t *testing.T) {
	g := GatewayConfig{TLS: &GatewayTLS{Hostname: "gw.example"}}
	if g.Fronted() {
		t.Error("Enabled false means not fronted")
	}
	if (GatewayConfig{}).Fronted() {
		t.Error("a nil TLS pointer means not fronted")
	}
}

func TestGatewayTLS_Defaults(t *testing.T) {
	var nilTLS *GatewayTLS
	if nilTLS.On() {
		t.Error("a nil front is off")
	}
	if got := nilTLS.CertSourceOrDefault(); got != CertInternal {
		t.Errorf("cert source default = %q, want %q", got, CertInternal)
	}

	tls := &GatewayTLS{Enabled: true, Hostname: "gw.example"}
	// The TLS front's bind defaults WIDE, unlike every other bind in this app:
	// a front bound to loopback serves only the machine it runs on, which is the
	// one machine that never needed TLS.
	if got := tls.Bind(); got != "0.0.0.0" {
		t.Errorf("bind default = %q, want 0.0.0.0", got)
	}
	if got := tls.HTTPS(); got != 443 {
		t.Errorf("https default = %d, want 443", got)
	}
	if got := tls.URL(); got != "https://gw.example" {
		t.Errorf("url = %q", got)
	}
	tls.HTTPSPort = 8443
	if got := tls.URL(); got != "https://gw.example:8443" {
		t.Errorf("url with a custom port = %q", got)
	}
}

// A hostname a Caddyfile cannot serve must be refused where it was typed, not
// discovered at provision time.
func TestGatewayTLS_ValidateSettings(t *testing.T) {
	if err := (&GatewayTLS{}).ValidateSettings(); err != nil {
		t.Errorf("an off front has nothing to validate: %v", err)
	}
	for _, tc := range []struct {
		name string
		tls  GatewayTLS
		want string
	}{
		{"no hostname", GatewayTLS{Enabled: true}, "hostname is required"},
		{"a pasted URL", GatewayTLS{Enabled: true, Hostname: "https://gw.example/"}, "bare host name"},
		{"files with no files", GatewayTLS{Enabled: true, Hostname: "gw.example", CertSource: CertFiles}, "certFile and keyFile"},
		{"port out of range", GatewayTLS{Enabled: true, Hostname: "gw.example", HTTPSPort: 70000}, "out of range"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.tls.ValidateSettings()
			if err == nil {
				t.Fatal("want an error")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %q, want it to contain %q", err, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------
// metrics
// ---------------------------------------------------------------------

// oneChain is the smallest renderable gateway, so a metrics assertion is not
// buried in unrelated upstream setup.
func oneChain() GatewayConfig {
	return GatewayConfig{Networks: []GatewayNetwork{{ChainID: 369, Upstreams: []GatewayUpstream{
		{Endpoint: "https://rpc.pulsechain.com"},
	}}}}
}

// The block is rendered in BOTH states, and that is the whole point of it.
//
// eRPC's own default is metrics.enabled = true, so a config that says nothing
// still serves counters — measured on this machine's two gateways, both of
// which had been exposing Prometheus on 4001 since they were created while
// this app rendered no metrics block at all. Rendering only the "off" case
// would leave the file unable to say what it is doing, and rendering neither
// would give the operator's off switch nothing to write.
func TestRenderGatewayConfig_MetricsBlockIsAlwaysRendered(t *testing.T) {
	on, err := RenderGatewayConfig(oneChain())
	if err != nil {
		t.Fatalf("RenderGatewayConfig: %v", err)
	}
	for _, want := range []string{"metrics:", "enabled: true", `hostV4: "127.0.0.1"`, "port: 4001"} {
		if !strings.Contains(on, want) {
			t.Errorf("metrics default missing %q:\n%s", want, on)
		}
	}

	g := oneChain()
	g.MetricsOff = true
	off, err := RenderGatewayConfig(g)
	if err != nil {
		t.Fatalf("RenderGatewayConfig(off): %v", err)
	}
	if !strings.Contains(off, "enabled: false") {
		t.Errorf("turning metrics off must WRITE enabled: false, not omit the block:\n%s", off)
	}
}

// The counters bind loopback by default while the RPC listener does not. The
// asymmetry is deliberate — one is a front door, the other is not — so it is
// asserted rather than left to the reader.
func TestGatewayConfig_MetricsBindDefaultsToLoopbackIndependentlyOfRPCBind(t *testing.T) {
	g := oneChain()
	g.BindAddr = "0.0.0.0"
	if got := g.MetricsBind(); got != "127.0.0.1" {
		t.Errorf("MetricsBind = %q, want 127.0.0.1 even with a wide RPC bind", got)
	}
	cfg, err := RenderGatewayConfig(g)
	if err != nil {
		t.Fatalf("RenderGatewayConfig: %v", err)
	}
	if !strings.Contains(cfg, `hostV4: "127.0.0.1"`) || !strings.Contains(cfg, `httpHostV4: "0.0.0.0"`) {
		t.Errorf("a wide RPC bind must not widen the metrics bind:\n%s", cfg)
	}
}

func TestGatewayConfig_MetricsAccessors(t *testing.T) {
	var zero GatewayConfig
	if !zero.MetricsEnabled() {
		t.Error("the zero value must mean metrics ON — anything else silently disables them for every config already on disk")
	}
	if got := zero.MetricsHTTP(); got != 4001 {
		t.Errorf("MetricsHTTP = %d, want 4001", got)
	}
	g := GatewayConfig{MetricsPort: 9101, MetricsOff: true}
	if g.MetricsEnabled() {
		t.Error("MetricsOff must disable")
	}
	if got := g.MetricsHTTP(); got != 9101 {
		t.Errorf("MetricsHTTP = %d, want the explicit 9101", got)
	}
}

// Two listeners cannot share a port. eRPC would bind whichever started first
// and fail the other, leaving a gateway serving either RPC or counters with
// nothing to say which.
func TestRenderGatewayConfig_RejectsMetricsPortCollision(t *testing.T) {
	g := oneChain()
	g.Port = 4001
	if _, err := RenderGatewayConfig(g); err == nil {
		t.Fatal("want an error when the RPC port collides with the default metrics port")
	} else if !strings.Contains(err.Error(), "cannot share one port") {
		t.Errorf("error should name the collision, got: %v", err)
	}

	// Turning metrics off resolves it: there is no second listener to collide.
	g.MetricsOff = true
	if _, err := RenderGatewayConfig(g); err != nil {
		t.Errorf("with metrics off there is no second listener, so 4001 is free: %v", err)
	}
}

// Anti-drift guard. PathFor must always equal PathForArch("evm", ...): a
// second, drifting definition of the path shape would send the relay to a
// path eRPC does not serve.
func TestPathFor_MatchesPathForArchEVM(t *testing.T) {
	g := GatewayConfig{ProjectID: "fleet"}
	got := g.PathFor(369)
	want := g.PathForArch("evm", 369)
	if got != want {
		t.Errorf("PathFor(369) = %q, PathForArch(\"evm\", 369) = %q; they must match", got, want)
	}
	if got != "/fleet/evm/369" {
		t.Errorf("PathFor(369) = %q, want /fleet/evm/369", got)
	}
}
