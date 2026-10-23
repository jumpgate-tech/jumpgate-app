package catalog

import (
	"slices"
	"strings"
	"testing"
)

// fakeBoardWithUnbounded is fakeMsgboard plus an unbounded read, so every
// method class is present.
func fakeBoardWithUnbounded() Module {
	m := fakeMsgboard()
	m.Methods = append(m.Methods, ModuleMethod{Name: "msgboard_content", Class: MethodUnbounded})
	return m
}

func fakePrivateSend() Module {
	return Module{
		ID: "private-send", Placement: PlaceGateway, Chains: []int{1},
		Routes: map[int][]GatewayRoute{1: {{ID: "a", Endpoint: "https://a", Methods: []string{"eth_sendRawTransaction"}}}},
	}
}

func TestIgnoreMethodsRendered(t *testing.T) {
	g := GatewayConfig{Networks: []GatewayNetwork{{ChainID: 369, Upstreams: []GatewayUpstream{
		{ID: "u1", Endpoint: "https://rpc.pulsechain.com", IgnoreMethods: []string{"msgboard_*"}},
	}}}}
	out, err := RenderGatewayConfig(g)
	if err != nil {
		t.Fatal(err)
	}
	want := "      - id: u1\n        type: evm\n        endpoint: \"https://rpc.pulsechain.com\"\n        evm:\n          chainId: 369\n        ignoreMethods:\n          - \"msgboard_*\"\n"
	if !strings.Contains(out, want) {
		t.Errorf("missing\n%s\nin\n%s", want, out)
	}
}

func TestModuleIgnoreMethods(t *testing.T) {
	withModules(t, []Module{fakeBoardWithUnbounded()})
	node := GatewayUpstream{Kind: UpstreamManagedNode, TargetID: "b"}
	ext := GatewayUpstream{Endpoint: "https://x"}
	without := &WireConfig{ChainID: 369}
	with := func(public string) *WireConfig {
		return &WireConfig{ChainID: 369, Modules: []NodeModule{{ID: "msgboard", Public: public}}}
	}
	all := []string{"msgboard_*"}
	stream := []string{"msgboard_subscribe"}
	for _, tc := range []struct {
		name string
		tier Tier
		u    GatewayUpstream
		w    *WireConfig
		want []string
	}{
		{"external", TierLocal, ext, nil, all},
		{"external public", TierPublic, ext, nil, all},
		{"node without module", TierLocal, node, without, all},
		{"with module, local", TierLocal, node, with(""), stream},
		{"with module, private", TierPrivate, node, with("post"), stream},
		{"public, local only", TierPublic, node, with(""), all},
		{"public, read", TierPublic, node, with("read"), []string{"msgboard_addMessage", "msgboard_subscribe", "msgboard_content"}},
		{"public, post", TierPublic, node, with("post"), []string{"msgboard_subscribe", "msgboard_content"}},
	} {
		if got := moduleIgnoreMethods(tc.tier, tc.u, tc.w); !slices.Equal(got, tc.want) {
			t.Errorf("%s: %v, want %v", tc.name, got, tc.want)
		}
	}
}

// A chain where no upstream serves the module renders no ignore list at all;
// one where a node does fences every other upstream off from it.
func TestApplyModuleIgnores(t *testing.T) {
	withModules(t, []Module{fakeBoardWithUnbounded()})
	n := GatewayNetwork{ChainID: 369, Upstreams: []GatewayUpstream{
		{Kind: UpstreamManagedNode, TargetID: "b"}, {Endpoint: "https://x"},
	}}
	off := ApplyModuleIgnores(TierPublic, n, []*WireConfig{{ChainID: 369}, nil}, false)
	for _, u := range off.Upstreams {
		if len(u.IgnoreMethods) != 0 {
			t.Errorf("no module on, yet %v", u.IgnoreMethods)
		}
	}
	on := ApplyModuleIgnores(TierLocal, n, []*WireConfig{{ChainID: 369, Modules: []NodeModule{{ID: "msgboard"}}}, nil}, true)
	if got := on.Upstreams[0].IgnoreMethods; !slices.Equal(got, []string{"msgboard_subscribe"}) {
		t.Errorf("node: %v", got)
	}
	if got := on.Upstreams[1].IgnoreMethods; !slices.Equal(got, []string{"msgboard_*"}) {
		t.Errorf("external: %v", got)
	}
	if len(n.Upstreams[0].IgnoreMethods) != 0 {
		t.Error("ApplyModuleIgnores changed its input")
	}
	// A module on somewhere else on this gateway, none on this chain: the
	// external upstream still ignores the namespace, so msgboard_addMessage
	// never falls over to a public endpoint.
	elsewhere := ApplyModuleIgnores(TierPublic, n, []*WireConfig{{ChainID: 369}, nil}, true)
	for i, u := range elsewhere.Upstreams {
		if !slices.Equal(u.IgnoreMethods, []string{"msgboard_*"}) {
			t.Errorf("upstream %d: %v, want [msgboard_*]", i, u.IgnoreMethods)
		}
	}
}

func TestGatewayModuleUpstreams(t *testing.T) {
	withModules(t, []Module{fakePrivateSend()})
	g := GatewayConfig{Modules: []GatewayModule{{ID: "private-send"}}, Networks: []GatewayNetwork{
		{ChainID: 1, Upstreams: []GatewayUpstream{{ID: "pub", Endpoint: "https://eth.drpc.org"}}},
		{ChainID: 369, Upstreams: []GatewayUpstream{{ID: "pls", Endpoint: "https://rpc.pulsechain.com"}}},
	}}
	extra, ignore := gatewayModuleUpstreams(g, 1)
	if len(extra) != 1 || extra[0].ID != "1-mod-a" || !slices.Equal(extra[0].AllowMethods, []string{"eth_sendRawTransaction"}) {
		t.Errorf("extra = %+v", extra)
	}
	if !slices.Equal(ignore, []string{"eth_sendRawTransaction"}) {
		t.Errorf("ignoreOnOthers = %v", ignore)
	}
	if extra, ignore := gatewayModuleUpstreams(g, 369); extra != nil || ignore != nil {
		t.Errorf("chain 369 touched: %v %v", extra, ignore)
	}

	out, err := RenderGatewayConfig(g)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"      - id: pub\n        type: evm\n        endpoint: \"https://eth.drpc.org\"\n        evm:\n          chainId: 1\n        ignoreMethods:\n          - \"eth_sendRawTransaction\"\n",
		"      - id: 1-mod-a\n        type: evm\n        endpoint: \"https://a\"\n        evm:\n          chainId: 1\n        ignoreMethods:\n          - \"*\"\n        allowMethods:\n          - \"eth_sendRawTransaction\"\n",
		"      - id: pls\n        type: evm\n        endpoint: \"https://rpc.pulsechain.com\"\n        evm:\n          chainId: 369\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing\n%s\nin\n%s", want, out)
		}
	}
	if strings.Count(out, "eth_sendRawTransaction") != 2 {
		t.Errorf("chain 369 must be untouched:\n%s", out)
	}
	// The route's upstream is not demoted to a fallback, and the public one
	// is not either: neither is a local node.
	if strings.Contains(out, "tier:fallback") {
		t.Errorf("unexpected fallback tier:\n%s", out)
	}
}

func TestValidateGatewayModules(t *testing.T) {
	withModules(t, []Module{fakeMsgboard(), fakePrivateSend()})
	nets := []GatewayNetwork{{ChainID: 369, Upstreams: []GatewayUpstream{{Endpoint: "https://x"}}}}
	for _, tc := range []struct {
		mods []GatewayModule
		want string
	}{
		{[]GatewayModule{{ID: "msgboard"}}, "msgboard is not a gateway module"},
		{[]GatewayModule{{ID: "private-send"}}, "private-send applies to none of this gateway's chains"},
		{[]GatewayModule{{ID: "nope"}}, `"nope" is not a module`},
	} {
		err := ValidateGatewayModules(GatewayConfig{Modules: tc.mods, Networks: nets})
		if err == nil || err.Error() != tc.want {
			t.Errorf("%v: err = %v, want %q", tc.mods, err, tc.want)
		}
		if _, err := RenderGatewayConfig(GatewayConfig{Modules: tc.mods, Networks: nets}); err == nil {
			t.Errorf("%v: render accepted it", tc.mods)
		}
	}
	ok := GatewayConfig{Modules: []GatewayModule{{ID: "private-send"}}, Networks: []GatewayNetwork{{ChainID: 1, Upstreams: []GatewayUpstream{{Endpoint: "https://x"}}}}}
	if err := ValidateGatewayModules(ok); err != nil {
		t.Errorf("good config: %v", err)
	}
	dup := ok
	dup.Modules = []GatewayModule{{ID: "private-send"}, {ID: "private-send"}}
	if err := ValidateGatewayModules(dup); err == nil {
		t.Error("duplicate accepted")
	}
}

func TestGatewayTier(t *testing.T) {
	for _, tc := range []struct {
		tls  *GatewayTLS
		want Tier
	}{
		{nil, TierLocal},
		{&GatewayTLS{Enabled: false, CertSource: CertACME}, TierLocal},
		{&GatewayTLS{Enabled: true}, TierPrivate},
		{&GatewayTLS{Enabled: true, CertSource: CertFiles}, TierPrivate},
		{&GatewayTLS{Enabled: true, CertSource: CertACME}, TierPublic},
	} {
		if got := GatewayTier(GatewayConfig{TLS: tc.tls}); got != tc.want {
			t.Errorf("%+v: %s, want %s", tc.tls, got, tc.want)
		}
	}
}
