package catalog

import (
	"flag"
	"os"
	"path/filepath"
	"testing"
)

// The gateway and Caddyfile goldens were rendered at 1ef27cf, before modules
// existed. With no module on, every fixture must still render byte for byte
// the same, so upgrading to this jumpgate restarts no gateway. Never
// regenerate them from a later commit.
var updateGatewayGoldens = flag.Bool("update-gateway-goldens", false, "rewrite testdata/gateway-1ef27cf (only ever at 1ef27cf)")

const gatewayGoldenDir = "testdata/gateway-1ef27cf"

func gatewayGoldenFixtures() map[string]GatewayConfig {
	return map[string]GatewayConfig{
		"multi-chain": {Networks: []GatewayNetwork{
			{ChainID: 369, Upstreams: []GatewayUpstream{
				{ID: "369-local", Endpoint: "http://127.0.0.1:8545", Local: true, RecentOnly: true},
				{Endpoint: "https://rpc.pulsechain.com"},
				{Endpoint: "wss://rpc.pulsechain.com"},
			}},
			{ChainID: 1, Upstreams: []GatewayUpstream{
				{Endpoint: "https://ethereum-rpc.publicnode.com"},
				{Endpoint: "https://eth.drpc.org"},
			}},
			{ChainID: 943, Upstreams: []GatewayUpstream{
				{ID: "943-node", Kind: UpstreamManagedNode, TargetID: "box-1", Endpoint: "http://10.0.0.2:8545", Local: true},
			}},
		}},
		"defaults": {Networks: []GatewayNetwork{{ChainID: 1, Upstreams: []GatewayUpstream{{Endpoint: "https://eth.drpc.org"}}}}},
		"custom": {
			ProjectID: "valve", BindAddr: "0.0.0.0", Port: 4100, MetricsPort: 4101, MetricsBindAddr: "0.0.0.0",
			Networks: []GatewayNetwork{{ChainID: 560048, Upstreams: []GatewayUpstream{
				{ID: "hoodi-a", Endpoint: "ws://10.0.0.3:8546", Local: true},
				{ID: "hoodi-b", Endpoint: "https://hoodi.example.org", RecentOnly: true},
			}}},
		},
		"metrics-off": {MetricsOff: true, TLS: &GatewayTLS{Enabled: true, Hostname: "gw.example.com", CertSource: CertACME},
			Networks: []GatewayNetwork{{ChainID: 369, Upstreams: []GatewayUpstream{{Endpoint: "https://rpc.pulsechain.com"}}}}},
		"for-wire": GatewayForWire(WireConfig{ChainID: 369, ERPCUpstreams: []string{"https://rpc.pulsechain.com", "wss://ws.pulsechain.com"}}),
	}
}

func caddyGoldenFixtures() map[string]CaddyConfig {
	return map[string]CaddyConfig{
		"internal": {Hostname: "default-1a2b3c.localhost-valaxy.com", UpstreamHost: "valve-node-app-erpc", UpstreamPort: 4000},
		"files": {Hostname: "box.tail1234.ts.net", CertSource: CertFiles, CertFile: "/var/lib/valve-node-app/tls/cert.pem",
			KeyFile: "/var/lib/valve-node-app/tls/key.pem", UpstreamHost: "erpc", UpstreamPort: 4000},
		"acme":    {Hostname: "gw.example.com", CertSource: CertACME, ACMEEmail: "ops@example.com", UpstreamHost: "erpc", UpstreamPort: 4000, HTTPSPort: 8443},
		"metered": {Hostname: "default-1a2b3c.localhost-valaxy.com", UpstreamHost: "valve-node-app-erpc", UpstreamPort: 4000, Metered: true, RelayHost: "valve-node-app-relay", RelayPort: 8090},
	}
}

func checkGatewayGolden(t *testing.T, name, got string) {
	t.Helper()
	path := filepath.Join(gatewayGoldenDir, name)
	if *updateGatewayGoldens {
		if err := os.MkdirAll(gatewayGoldenDir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("golden %s: %v", name, err)
	}
	if got != string(want) {
		t.Errorf("%s differs from its 1ef27cf golden:\n--- got\n%s\n--- want\n%s", name, got, want)
	}
}

func TestGatewayConfigByteIdenticalWithoutModules(t *testing.T) {
	for name, g := range gatewayGoldenFixtures() {
		for _, mods := range [][]GatewayModule{nil, {}} {
			g := g
			g.Modules = mods
			got, err := RenderGatewayConfig(g)
			if err != nil {
				t.Fatalf("%s: %v", name, err)
			}
			checkGatewayGolden(t, name+".erpc.yaml", got)
		}
	}
	for name, c := range caddyGoldenFixtures() {
		got, err := RenderCaddyfile(c)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		checkGatewayGolden(t, name+".Caddyfile", got)
	}
}
