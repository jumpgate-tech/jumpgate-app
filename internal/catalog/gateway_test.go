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
func TestRenderGatewayConfig_NoLocalUpstreams(t *testing.T) {
	g := GatewayConfig{Networks: []GatewayNetwork{{ChainID: 1, Upstreams: []GatewayUpstream{
		{Endpoint: "https://ethereum-rpc.publicnode.com"},
		{Endpoint: "wss://ethereum-rpc.publicnode.com"},
	}}}}
	cfg, err := RenderGatewayConfig(g)
	if err != nil {
		t.Fatalf("RenderGatewayConfig: %v", err)
	}
	if got := strings.Count(cfg, "tier:fallback"); got != 2 {
		t.Errorf("every upstream should be a fallback, got %d:\n%s", got, cfg)
	}
	// Ids are generated when the caller supplies none, and must be unique.
	for _, want := range []string{"id: 1-fallback-1", "id: 1-fallback-2"} {
		if !strings.Contains(cfg, want) {
			t.Errorf("missing generated id %q:\n%s", want, cfg)
		}
	}
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

// MEASURED, and the reason this flag exists at all: eRPC's WebSocket upgrade
// returns HTTP 500 ("websocket: response does not implement http.Hijacker")
// whenever the client advertises Accept-Encoding: gzip, and EVERY reverse proxy
// adds that header to what it forwards. A fronted gateway that still offered
// gzip would lose eth_subscribe entirely — silently, since ordinary
// request/response calls keep working.
func TestRenderGatewayConfig_FrontedGatewayDisablesGzip(t *testing.T) {
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
	if !strings.Contains(fronted, "enableGzip: false") {
		t.Errorf("a fronted gateway must disable gzip or it loses eth_subscribe:\n%s", fronted)
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
