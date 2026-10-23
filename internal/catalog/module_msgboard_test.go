package catalog

import (
	"slices"
	"strings"
	"testing"
)

func TestMsgboardEntry(t *testing.T) {
	m, ok := ModuleByID("msgboard")
	if !ok {
		t.Fatal("msgboard is not registered")
	}
	if m.Placement != PlaceNode || len(m.Chains) != 2 || m.Chains[0] != 369 || m.Chains[1] != 943 {
		t.Fatalf("placement/chains %v %v", m.Placement, m.Chains)
	}
	if len(m.Clients) != 1 || m.Clients[0] != "reth" || len(m.Namespaces) != 1 || m.Namespaces[0] != "msgboard" || len(m.Flags) != 0 {
		t.Fatalf("clients %v namespaces %v flags %v", m.Clients, m.Namespaces, m.Flags)
	}
	want := map[string]MethodClass{
		"msgboard_addMessage":  MethodWrite,
		"msgboard_status":      MethodRead,
		"msgboard_categories":  MethodRead,
		"msgboard_contentPage": MethodRead,
		"msgboard_getMessage":  MethodRead,
		"msgboard_content":     MethodUnbounded,
		"msgboard_subscribe":   MethodStream,
		"msgboard_unsubscribe": MethodStream,
	}
	if len(m.Methods) != len(want) {
		t.Fatalf("methods %v", m.Methods)
	}
	for _, mm := range m.Methods {
		if want[mm.Name] != mm.Class {
			t.Errorf("%s: class %q, want %q", mm.Name, mm.Class, want[mm.Name])
		}
	}
	if m.Probe != (ModuleProbe{HelpContains: "--msgboard.block-range", StatusMethod: "msgboard_status"}) {
		t.Fatalf("probe %+v", m.Probe)
	}
	if !m.NeedsSync || m.Resources != (Resources{RAMBytes: 256 << 20, DiskBytes: 128 << 20}) {
		t.Fatalf("needsSync %v resources %+v", m.NeedsSync, m.Resources)
	}
	if m.PollHint != "poll msgboard_contentPage" || m.SiteURL != "https://msgboard.xyz" || m.DocsStep != "msgboard-enable" {
		t.Fatalf("pollHint %q site %q docs %q", m.PollHint, m.SiteURL, m.DocsStep)
	}
	if ok, r := m.Applies(1, "reth"); ok || r != "" {
		t.Fatalf("chain 1: %v %q, want hidden", ok, r)
	}
	if ok, r := m.Applies(369, "go-pulse"); ok || r != "The message board needs reth (valve fork). This node runs go-pulse." {
		t.Fatalf("go-pulse: %v %q", ok, r)
	}
	if ok, r := m.Applies(943, "erigon-pulse"); ok || !strings.Contains(r, "erigon-pulse") {
		t.Fatalf("erigon-pulse: %v %q", ok, r)
	}
	for _, c := range []int{369, 943} {
		if ok, _ := m.Applies(c, "reth"); !ok {
			t.Fatalf("%d reth does not apply", c)
		}
	}
}

func TestMsgboardHTTPAPI(t *testing.T) {
	for _, chain := range []int{369, 943} {
		n, ok := NetworkByChainID(chain)
		if !ok {
			t.Fatalf("no network %d", chain)
		}
		w := WireConfig{ChainID: chain, ExecID: "reth", BeaconID: n.BeaconClients[0], DataDir: "/var/lib/x", Modules: []NodeModule{{ID: "msgboard"}}}
		if err := ValidateModules(w); err != nil {
			t.Fatalf("%d: %v", chain, err)
		}
		execUnit, _, err := RenderUnits(w)
		if err != nil {
			t.Fatalf("%d: %v", chain, err)
		}
		if !strings.Contains(execUnit, " --http.api eth,net,web3,msgboard") {
			t.Fatalf("%d: unit lacks the api list:\n%s", chain, execUnit)
		}
	}
}

// msgboard_content builds the whole board in one call: behind a Public-tier
// gateway no Public choice ever routes it.
func TestMsgboardContentNeverPublic(t *testing.T) {
	u := GatewayUpstream{Kind: UpstreamManagedNode}
	for _, pub := range []string{"", "read", "post"} {
		w := &WireConfig{Modules: []NodeModule{{ID: "msgboard", Public: pub}}}
		ign := moduleIgnoreMethods(TierPublic, u, w)
		if !slices.Contains(ign, "msgboard_content") && !slices.Contains(ign, "msgboard_*") {
			t.Errorf("public %q: msgboard_content is routed behind a Public-tier gateway (ignores %v)", pub, ign)
		}
	}
}
