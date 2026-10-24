package catalog

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// upstreamBlocks splits a rendered eRPC config into its upstream entries,
// keyed by id.
func upstreamBlocks(t *testing.T, out string) map[string]string {
	t.Helper()
	_, list, ok := strings.Cut(out, "    upstreams:\n")
	if !ok {
		t.Fatalf("no upstreams in\n%s", out)
	}
	blocks := map[string]string{}
	for _, b := range strings.Split(list, "      - id: ")[1:] {
		id, _, _ := strings.Cut(b, "\n")
		blocks[id] = "      - id: " + b
	}
	return blocks
}

// listUnder is the unquoted entries of the YAML list key in block.
func listUnder(block, key string) []string {
	_, rest, ok := strings.Cut(block, "        "+key+":\n")
	if !ok {
		return nil
	}
	var out []string
	for _, l := range strings.Split(rest, "\n") {
		v, ok := strings.CutPrefix(l, "          - ")
		if !ok {
			break
		}
		out = append(out, strings.Trim(v, `"`))
	}
	return out
}

func privateSendGateway() GatewayConfig {
	return GatewayConfig{Modules: []GatewayModule{{ID: "private-send"}}, Networks: []GatewayNetwork{
		{ChainID: 1, Upstreams: []GatewayUpstream{
			{ID: "1-local", Endpoint: "http://127.0.0.1:8545", Local: true},
			{Endpoint: "https://ethereum-rpc.publicnode.com"},
			{Endpoint: "wss://ethereum-rpc.publicnode.com"},
			{ID: "1-ext", Endpoint: "https://eth.drpc.org", IgnoreMethods: []string{"msgboard_*"}},
		}},
	}}
}

func TestPrivateSendEntry(t *testing.T) {
	m, ok := ModuleByID("private-send")
	if !ok {
		t.Fatal("private-send is not registered")
	}
	if m.Placement != PlaceGateway || !slices.Equal(m.Chains, []int{1}) {
		t.Errorf("placement/chains = %v %v", m.Placement, m.Chains)
	}
	if m.Summary != "Sends your transactions to Flashbots Protect and MEV Blocker instead of the public mempool." {
		t.Errorf("summary = %q", m.Summary)
	}
	want := []GatewayRoute{
		{ID: "flashbots", Endpoint: "https://rpc.flashbots.net/fast", Methods: []string{"eth_sendRawTransaction"}},
		{ID: "mevblocker", Endpoint: "https://rpc.mevblocker.io", Methods: []string{"eth_sendRawTransaction"}},
	}
	got := m.Routes[1]
	if len(got) != len(want) {
		t.Fatalf("routes = %+v", got)
	}
	for i := range want {
		if got[i].ID != want[i].ID || got[i].Endpoint != want[i].Endpoint || !slices.Equal(got[i].Methods, want[i].Methods) {
			t.Errorf("route %d = %+v, want %+v", i, got[i], want[i])
		}
		// No key, token or query string: the URL is written to the gateway
		// config and may show up in its logs.
		if strings.ContainsAny(got[i].Endpoint, "?@#") {
			t.Errorf("route %s endpoint carries a query or credential: %s", got[i].ID, got[i].Endpoint)
		}
	}
	if len(m.Routes) != 1 {
		t.Errorf("routes on chains other than 1: %v", m.Routes)
	}
}

func TestPrivateSendRendersRoutes(t *testing.T) {
	out, err := RenderGatewayConfig(privateSendGateway())
	if err != nil {
		t.Fatal(err)
	}
	blocks := upstreamBlocks(t, out)
	for _, id := range []string{"1-mod-flashbots", "1-mod-mevblocker"} {
		b, ok := blocks[id]
		if !ok {
			t.Fatalf("no upstream %s in\n%s", id, out)
		}
		if got := listUnder(b, "allowMethods"); !slices.Equal(got, []string{"eth_sendRawTransaction"}) {
			t.Errorf("%s allowMethods = %v", id, got)
		}
		if got := listUnder(b, "ignoreMethods"); !slices.Equal(got, []string{"*"}) {
			t.Errorf("%s ignoreMethods = %v", id, got)
		}
		if strings.Contains(b, "tier:fallback") {
			t.Errorf("%s is a fallback:\n%s", id, b)
		}
	}
	if got := listUnder(blocks["1-ext"], "ignoreMethods"); !slices.Equal(got, []string{"msgboard_*", "eth_sendRawTransaction"}) {
		t.Errorf("1-ext keeps its own ignores first: %v", got)
	}
}

// TestPrivateSendNeverFallsBackToPublic: strict mode. Every chain-1 upstream
// that is not a private route, the local node and the tier:fallback public
// ones alike, ignores eth_sendRawTransaction, so when both routes fail the
// send is an error and is never sent publicly. Reads stay on them.
func TestPrivateSendNeverFallsBackToPublic(t *testing.T) {
	g := privateSendGateway()
	out, err := RenderGatewayConfig(g)
	if err != nil {
		t.Fatal(err)
	}
	blocks := upstreamBlocks(t, out)
	if len(blocks) != len(g.Networks[0].Upstreams)+2 {
		t.Fatalf("upstreams = %d:\n%s", len(blocks), out)
	}
	sawFallback := false
	for id, b := range blocks {
		if strings.HasPrefix(id, "1-mod-") {
			continue
		}
		sawFallback = sawFallback || strings.Contains(b, "tier:fallback")
		ign := listUnder(b, "ignoreMethods")
		if !slices.Contains(ign, "eth_sendRawTransaction") {
			t.Errorf("%s can be sent a transaction:\n%s", id, b)
		}
		if slices.Contains(ign, "*") || listUnder(b, "allowMethods") != nil {
			t.Errorf("%s no longer serves reads:\n%s", id, b)
		}
	}
	if !sawFallback {
		t.Error("fixture has no tier:fallback upstream")
	}

	// The same holds after the server's ApplyModuleIgnores pass, which
	// rewrites every upstream's IgnoreMethods before the render.
	g.Networks[0] = ApplyModuleIgnores(TierLocal, g.Networks[0], nil, true)
	out, err = RenderGatewayConfig(g)
	if err != nil {
		t.Fatal(err)
	}
	for id, b := range upstreamBlocks(t, out) {
		if !strings.HasPrefix(id, "1-mod-") && !slices.Contains(listUnder(b, "ignoreMethods"), "eth_sendRawTransaction") {
			t.Errorf("after ApplyModuleIgnores, %s can be sent a transaction:\n%s", id, b)
		}
	}
}

func TestPrivateSendLeavesOtherChains(t *testing.T) {
	g := gatewayGoldenFixtures()["multi-chain"]
	g.Modules = []GatewayModule{{ID: "private-send"}}
	out, err := RenderGatewayConfig(g)
	if err != nil {
		t.Fatal(err)
	}
	golden, err := os.ReadFile(filepath.Join(gatewayGoldenDir, "multi-chain.erpc.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	want, got := upstreamBlocks(t, string(golden)), upstreamBlocks(t, out)
	for id, b := range want {
		if strings.HasPrefix(id, "1-") {
			continue
		}
		if got[id] != b {
			t.Errorf("%s changed:\n--- got\n%s\n--- want\n%s", id, got[id], b)
		}
	}
	for id := range got {
		if strings.Contains(id, "-mod-") && !strings.HasPrefix(id, "1-mod-") {
			t.Errorf("route on another chain: %s", id)
		}
	}
}

func TestPrivateSendAppliesReasons(t *testing.T) {
	m, ok := ModuleByID("private-send")
	if !ok {
		t.Fatal("private-send is not registered")
	}
	for chain, want := range map[int]string{
		560048: "No private send route exists on Hoodi.",
		369:    "No private send route exists on PulseChain.",
		943:    "No private send route exists on PulseChain.",
	} {
		if ok, why := m.Applies(chain, ""); ok || why != want {
			t.Errorf("Applies(%d) = %v %q, want false %q", chain, ok, why, want)
		}
	}
	if ok, why := m.Applies(1, ""); !ok {
		t.Errorf("Applies(1) = false %q", why)
	}
	if err := ValidateGatewayModules(GatewayConfig{Modules: []GatewayModule{{ID: "private-send"}},
		Networks: []GatewayNetwork{{ChainID: 369}}}); err == nil {
		t.Error("a 369-only gateway accepted private-send")
	}
}
