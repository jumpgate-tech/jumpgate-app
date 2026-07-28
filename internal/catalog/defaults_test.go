package catalog

import "testing"

// The defaults. Every one of these is a decision the app relies on without
// ever stating it at the call site, so a silent change to one is a behaviour
// change with nothing to review: a metrics listener that stops being loopback,
// an upstream that stops counting as the operator's, a devnet that starts
// pulling a different image.

// An upstream with no kind is external. That is not an arbitrary default: it
// is what every configuration written BEFORE upstreams had identity meant, and
// reading those as managed would send the resolver looking for a target that
// was never named.
func TestGatewayUpstream_UnkindedIsExternalAndUnmanaged(t *testing.T) {
	var u GatewayUpstream
	if got := u.KindOrDefault(); got != UpstreamExternal {
		t.Errorf("KindOrDefault: got %q, want %q", got, UpstreamExternal)
	}
	if u.Managed() {
		t.Error("an external upstream reads its Endpoint; nothing has to be derived for it")
	}

	for _, kind := range []string{UpstreamManagedNode, UpstreamManagedDevnet} {
		if !(GatewayUpstream{Kind: kind}).Managed() {
			t.Errorf("%q must be managed — its endpoint is derived from a target at render time", kind)
		}
	}
	if (GatewayUpstream{Kind: UpstreamExternal}).Managed() {
		t.Error("an explicitly external upstream must not be managed either")
	}
}

// The two binds default OPPOSITE ways, and that is the whole distinction: the
// RPC port is a front door meant to be reachable, the metrics port is a
// counter nobody outside the machine has any business reading.
func TestGatewayConfig_BindDefaults(t *testing.T) {
	var g GatewayConfig
	if got := g.Bind(); got != "127.0.0.1" {
		t.Errorf("Bind: got %q", got)
	}
	if got := g.MetricsBind(); got != "127.0.0.1" {
		t.Errorf("MetricsBind: got %q", got)
	}
	set := GatewayConfig{BindAddr: "0.0.0.0", MetricsBindAddr: "10.0.0.5"}
	if set.Bind() != "0.0.0.0" || set.MetricsBind() != "10.0.0.5" {
		t.Errorf("an explicit bind must survive: %q / %q", set.Bind(), set.MetricsBind())
	}
}

// The project id appears in every request path, so its default is part of
// every URL this app hands out.
func TestGatewayConfig_ProjectIDReachesThePath(t *testing.T) {
	var g GatewayConfig
	if got := g.ProjectIDOrDefault(); got != "main" {
		t.Errorf("ProjectIDOrDefault: got %q, want main", got)
	}
	if got := g.PathFor(369); got != "/main/evm/369" {
		t.Errorf("PathFor: got %q", got)
	}
	named := GatewayConfig{ProjectID: "fleet"}
	if got := named.PathFor(1); got != "/fleet/evm/1" {
		t.Errorf("PathFor with a named project: got %q", got)
	}
}

// A generated id has to be STABLE and has to say which tier it came from,
// because it is the id eRPC labels its counters with — and internal/setup
// regenerates it independently when joining those counters back to intent. If
// the two ever disagreed, every request would attribute to nothing.
func TestGeneratedUpstreamID_NamesTheTier(t *testing.T) {
	if got := GeneratedUpstreamID(369, true, 1); got != "369-local-1" {
		t.Errorf("local: got %q", got)
	}
	if got := GeneratedUpstreamID(369, false, 2); got != "369-fallback-2" {
		t.Errorf("fallback: got %q", got)
	}
}

// A devnet's defaults are what "just give me a chain" resolves to. The chain
// id in particular is not a preference — reth's --dev genesis is baked in, and
// Validate refuses any other value rather than serving 1337 under a different
// label.
func TestDevnetConfig_ZeroValueIsAUsableDevnet(t *testing.T) {
	var d DevnetConfig
	if got := d.ChainIDOrDefault(); got != DevnetChainID {
		t.Errorf("ChainIDOrDefault: got %d", got)
	}
	if got := d.Image(); got != DefaultDevnetImage {
		t.Errorf("Image: got %q", got)
	}
	if got := d.Name(); got != DevnetContainerName {
		t.Errorf("Name: got %q", got)
	}
	if got := d.BlockTimeOrDefault(); got != DefaultDevnetBlockTime {
		t.Errorf("BlockTimeOrDefault: got %q", got)
	}
	if err := d.Validate(); err != nil {
		t.Errorf("the zero value must be valid — \"just give me a chain\" should not require filling in a struct: %v", err)
	}

	set := DevnetConfig{ChainID: DevnetChainID, ImageRef: "ghcr.io/x/reth:pinned", ContainerName: "scratch", BlockTime: "5s"}
	if set.Image() != "ghcr.io/x/reth:pinned" || set.Name() != "scratch" || set.BlockTimeOrDefault() != "5s" {
		t.Errorf("explicit values must survive: %+v", set)
	}
}
