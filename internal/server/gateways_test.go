package server

import (
	"net/http"
	"strings"
	"testing"

	"github.com/valve-tech/valve-node-app/internal/catalog"
	"github.com/valve-tech/valve-node-app/internal/config"
	"github.com/valve-tech/valve-node-app/internal/executor"
)

// addGateway creates a top-level gateway placed on a target.
func addGateway(t *testing.T, a *apiTestServer, id, targetID string, cfg catalog.GatewayConfig) gatewayView {
	t.Helper()
	return addGatewayOn(t, a, id, targetID, "docker", cfg)
}

func addGatewayOn(t *testing.T, a *apiTestServer, id, targetID, backend string, cfg catalog.GatewayConfig) gatewayView {
	t.Helper()
	res := a.do(t, "POST", "/api/gateways", map[string]any{
		"id":        id,
		"placement": map[string]string{"targetId": targetID, "backend": backend},
		"config":    cfg,
	})
	if res.StatusCode != http.StatusCreated {
		defer res.Body.Close()
		t.Fatalf("POST /api/gateways: got %d, want 201", res.StatusCode)
	}
	return decode[gatewayView](t, res)
}

func gatewayServer(t *testing.T) *apiTestServer {
	t.Helper()
	a := newAPITestServerWithExecutor(t, func(config.Target) (executor.Executor, error) {
		return dockerExecutor("true|0|valve-node-app/erpc:e909aacb|sha256:abc\n", "4000/tcp=127.0.0.1:4100\n"), nil
	})
	addTarget(t, a)
	return a
}

// ---------------------------------------------------------------------
// placement and identity
// ---------------------------------------------------------------------

// The crux of the whole model: a gateway NAMES its host, the host does not
// own it — and two gateways can therefore coexist, on the same machine or on
// different ones, with distinct containers.
func TestGateways_TwoGatewaysCoexistWithDistinctContainers(t *testing.T) {
	a := gatewayServer(t)

	net := []catalog.GatewayNetwork{{ChainID: 369, Upstreams: []catalog.GatewayUpstream{
		{ID: "public", Endpoint: "https://rpc.pulsechain.com"},
	}}}
	first := addGateway(t, a, "default", "local", catalog.GatewayConfig{Port: 4100, Networks: net})
	second := addGateway(t, a, "edge", "local", catalog.GatewayConfig{Port: 4200, Networks: net})

	if first.ContainerName == second.ContainerName {
		t.Fatalf("two gateways share the container name %q — docker run --name would refuse the second one outright", first.ContainerName)
	}
	if first.ContainerName != "valve-node-app-erpc" {
		t.Errorf("the default gateway must keep the historical container name so an existing install is not orphaned: got %q", first.ContainerName)
	}
	if second.ContainerName != "valve-node-app-erpc-edge" {
		t.Errorf("second container name: got %q", second.ContainerName)
	}

	body := decode[gatewaysResponse](t, a.do(t, "GET", "/api/gateways", nil))
	if len(body.Gateways) != 2 {
		t.Fatalf("got %d gateways, want 2", len(body.Gateways))
	}
}

func TestGateways_RejectAnIDThatCannotBeAContainerName(t *testing.T) {
	a := gatewayServer(t)
	for _, bad := range []string{"Has Spaces", "../escape", "-leading", "UPPER", "way-too-long-" + "x" + "0123456789012345678901234567890"} {
		res := a.do(t, "POST", "/api/gateways", map[string]any{
			"id":        bad,
			"placement": map[string]string{"targetId": "local"},
		})
		res.Body.Close()
		if res.StatusCode != http.StatusBadRequest {
			t.Errorf("id %q: got %d, want 400 — the id becomes a container name, a unit name and a file name", bad, res.StatusCode)
		}
	}
}

func TestGateways_PlacementMustNameARegisteredMachine(t *testing.T) {
	a := gatewayServer(t)
	res := a.do(t, "POST", "/api/gateways", map[string]any{
		"id":        "orphan",
		"placement": map[string]string{"targetId": "nowhere"},
	})
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("got %d, want 400", res.StatusCode)
	}
}

// ---------------------------------------------------------------------
// upstream identity / resolution
// ---------------------------------------------------------------------

// The regression this whole change exists to prevent: a managed upstream
// stores a REFERENCE, so moving the devnet's port moves the gateway with it.
// With a frozen URL the gateway kept pointing at the old port and reported
// itself perfectly healthy while every call went nowhere.
// The gateway here is on the SYSTEMD backend on purpose: a systemd gateway is
// an ordinary process, not a container, so it has no docker network to resolve
// names on and must reach the devnet through its published HOST port — which
// is what makes it the case where "does the reference follow the port" is a
// question with an observable answer. The container-backend answer is a
// container name, and is asserted separately below.
//
// The published port it follows is the WS one, for the reason spelled out on
// resolveUpstream: a devnet upstream is always addressed by ws:// scheme.
func TestGateways_ManagedUpstreamFollowsTheDevnetsPort(t *testing.T) {
	a := gatewayServer(t)
	putConfig(t, a, svcDevnet, catalog.DevnetConfig{HTTPPort: 8600, WSPort: 8601})

	addGatewayOn(t, a, "default", "local", "systemd", catalog.GatewayConfig{
		Port: 4100,
		Networks: []catalog.GatewayNetwork{{ChainID: catalog.DevnetChainID, Upstreams: []catalog.GatewayUpstream{
			{ID: "devnet", Kind: catalog.UpstreamManagedDevnet, TargetID: "local"},
		}}},
	})

	got := decode[gatewayView](t, a.do(t, "GET", "/api/gateways/default", nil))
	if u := got.Networks[0].Upstreams[0]; u.Endpoint != "ws://127.0.0.1:8601" {
		t.Fatalf("endpoint: got %q, want it derived from the devnet's stored WS port", u.Endpoint)
	}

	// Move the devnet. Nothing about the gateway is touched.
	putConfig(t, a, svcDevnet, catalog.DevnetConfig{HTTPPort: 9600, WSPort: 9601})

	got = decode[gatewayView](t, a.do(t, "GET", "/api/gateways/default", nil))
	u := got.Networks[0].Upstreams[0]
	if u.Endpoint != "ws://127.0.0.1:9601" {
		t.Fatalf("endpoint after the devnet moved: got %q, want ws://127.0.0.1:9601 — a stored URL is exactly what goes stale here", u.Endpoint)
	}
	// The stored config still holds the reference, not the derived URL:
	// writing the URL back would freeze it and undo the whole mechanism.
	if stored := got.Config.Networks[0].Upstreams[0]; stored.Endpoint != "" || stored.TargetID != "local" {
		t.Errorf("stored upstream: got %+v, want a bare reference", stored)
	}
	// A managed upstream is one the operator runs, so it is preferred rather
	// than filed in eRPC's 0.2-scored fallback tier.
	if !u.Local {
		t.Error("a managed upstream must be preferred over public fallbacks")
	}
}

// Two containers this app placed on ONE machine talk over the private docker
// network, by CONTAINER NAME. That is what lets the devnet publish nothing at
// all, and it removes the host.docker.internal hop for the one case where both
// ends are ours.
//
// The port asserted is the IN-CONTAINER one (8546), not the published host port
// (8601): a container on the shared network reaches the listener directly, and
// the publish mapping is not in that path.
func TestGateways_SameMachineContainersTalkByContainerName(t *testing.T) {
	a := gatewayServer(t)
	putConfig(t, a, svcDevnet, catalog.DevnetConfig{HTTPPort: 8600, WSPort: 8601})

	addGateway(t, a, "default", "local", catalog.GatewayConfig{
		Port: 4100,
		Networks: []catalog.GatewayNetwork{{ChainID: catalog.DevnetChainID, Upstreams: []catalog.GatewayUpstream{
			{ID: "devnet", Kind: catalog.UpstreamManagedDevnet, TargetID: "local"},
		}}},
	})

	got := decode[gatewayView](t, a.do(t, "GET", "/api/gateways/default", nil))
	u := got.Networks[0].Upstreams[0]
	// ws://, not http://: eRPC infers WebSocket capability from the upstream
	// SCHEME, so an http upstream makes every eth_subscribe fail with
	// ErrNoWsUpstreamAvailable through an otherwise healthy gateway. Measured
	// both ways against a real fronted gateway. A ws upstream serves ordinary
	// request/response calls too, so it costs nothing.
	if want := "ws://valve-node-app-devnet:8546"; u.Endpoint != want {
		t.Fatalf("endpoint: got %q, want %q — a same-host container is addressed by name, and by ws so eth_subscribe works", u.Endpoint, want)
	}
	if u.Problem != "" {
		t.Errorf("a container-name upstream is reachable, not a problem: %s", u.Problem)
	}
}

// A reference to something that has gone must show up as a reason ON the row,
// not as a rendering failure that takes the whole gateway down.
func TestGateways_DeadReferenceIsReportedNotFatal(t *testing.T) {
	a := gatewayServer(t)
	addGateway(t, a, "default", "local", catalog.GatewayConfig{
		Port: 4100,
		Networks: []catalog.GatewayNetwork{
			{ChainID: catalog.DevnetChainID, Upstreams: []catalog.GatewayUpstream{
				// There is no devnet configured on "local".
				{ID: "devnet", Kind: catalog.UpstreamManagedDevnet, TargetID: "local"},
			}},
			{ChainID: 369, Upstreams: []catalog.GatewayUpstream{
				{ID: "public", Endpoint: "https://rpc.pulsechain.com"},
			}},
		},
	})

	got := decode[gatewayView](t, a.do(t, "GET", "/api/gateways/default", nil))
	if len(got.Networks) != 2 {
		t.Fatalf("got %d networks, want both — a broken chain must still be shown", len(got.Networks))
	}
	broken, healthy := got.Networks[0], got.Networks[1]
	if broken.Serviceable {
		t.Error("a chain whose only upstream is unresolvable must not read as serviceable")
	}
	if broken.Upstreams[0].Problem == "" {
		t.Error("the row must say WHY it cannot be used — that is the whole point of putting state on the thing you interact with")
	}
	if !healthy.Serviceable {
		t.Error("one dead reference must not take a healthy chain down with it")
	}
}

// A node on ANOTHER machine bound to loopback is the derivation that looks
// fine and cannot work: from the gateway's box, 127.0.0.1 is the gateway's
// box. Catching it here is the difference between a clear message and a
// gateway that reports healthy while serving errors.
func TestGateways_CrossMachineLoopbackUpstreamIsRefusedWithAReason(t *testing.T) {
	cfg := config.Config{Targets: []config.Target{
		{ID: "here", Mode: "local"},
		{ID: "boxa", Mode: "ssh", Wire: &catalog.WireConfig{ChainID: 369, ExecID: "reth"}},
		{ID: "boxb", Mode: "ssh", Wire: &catalog.WireConfig{ChainID: 369, ExecID: "reth", RPCBindAddr: "100.64.0.7"}},
	}}
	gw := config.Gateway{
		ID:        "default",
		Placement: config.GatewayPlacement{TargetID: "here", Backend: "docker"},
		Config: catalog.GatewayConfig{Networks: []catalog.GatewayNetwork{{ChainID: 369, Upstreams: []catalog.GatewayUpstream{
			{ID: "loopback-elsewhere", Kind: catalog.UpstreamManagedNode, TargetID: "boxa"},
			{ID: "routable-elsewhere", Kind: catalog.UpstreamManagedNode, TargetID: "boxb"},
		}}}},
	}

	resolved, problems := resolveGateway(cfg, gw)
	if len(resolved.Networks) != 1 || len(resolved.Networks[0].Upstreams) != 1 {
		t.Fatalf("resolved upstreams: %+v — the loopback one must be dropped and the routable one kept", resolved.Networks)
	}
	if got := resolved.Networks[0].Upstreams[0].Endpoint; got != "http://100.64.0.7:8545" {
		t.Errorf("kept upstream: got %q, want the routable node", got)
	}
	if len(problems) != 1 {
		t.Fatalf("problems: %v, want exactly the loopback one", problems)
	}
	// The message has to name the machine and say what to do, because the fix
	// is on the NODE's screen, not here.
	for _, want := range []string{"boxa", "loopback", "bind its RPC"} {
		if !strings.Contains(problems[0], want) {
			t.Errorf("problem %q does not mention %q", problems[0], want)
		}
	}
}

// A devnet on ANOTHER machine used to be the one managed-devnet path that
// resolved to http://, which meant eth_subscribe failed through it while the
// identical devnet beside the gateway subscribed fine. The scheme decides the
// capability in eRPC, so that difference was a feature appearing and
// disappearing with placement — the kind of thing an operator hits and cannot
// explain. Every managed devnet is a ws:// upstream now, whatever machine it
// is on; only the ADDRESS changes with placement.
func TestGateways_CrossMachineDevnetIsAWebSocketUpstream(t *testing.T) {
	cfg := config.Config{Targets: []config.Target{
		{ID: "here", Mode: "local"},
		{ID: "boxa", Mode: "ssh", Devnet: &catalog.DevnetConfig{BindAddr: "100.64.0.7", HTTPPort: 8600, WSPort: 8601}},
	}}
	gw := config.Gateway{
		ID:        "default",
		Placement: config.GatewayPlacement{TargetID: "here", Backend: "docker"},
		Config: catalog.GatewayConfig{Networks: []catalog.GatewayNetwork{{ChainID: catalog.DevnetChainID, Upstreams: []catalog.GatewayUpstream{
			{ID: "devnet", Kind: catalog.UpstreamManagedDevnet, TargetID: "boxa"},
		}}}},
	}

	resolved, problems := resolveGateway(cfg, gw)
	if len(problems) != 0 {
		t.Fatalf("problems: %v — a routable devnet on another machine is perfectly usable", problems)
	}
	// The PUBLISHED ws port (8601), not the in-container one: nothing on
	// another machine is on this devnet's docker network.
	if got := resolved.Networks[0].Upstreams[0].Endpoint; got != "ws://100.64.0.7:8601" {
		t.Errorf("endpoint: got %q, want ws://100.64.0.7:8601", got)
	}
}

// Switching that derivation to ws:// must not lose the loopback check with it:
// a devnet on another machine bound to loopback is still unreachable, and the
// URL it is unreachable at is now the ws one.
func TestGateways_CrossMachineLoopbackDevnetIsStillRefused(t *testing.T) {
	cfg := config.Config{Targets: []config.Target{
		{ID: "here", Mode: "local"},
		{ID: "boxa", Mode: "ssh", Devnet: &catalog.DevnetConfig{HTTPPort: 8600, WSPort: 8601}},
	}}
	gw := config.Gateway{
		ID:        "default",
		Placement: config.GatewayPlacement{TargetID: "here", Backend: "docker"},
		Config: catalog.GatewayConfig{Networks: []catalog.GatewayNetwork{{ChainID: catalog.DevnetChainID, Upstreams: []catalog.GatewayUpstream{
			{ID: "devnet", Kind: catalog.UpstreamManagedDevnet, TargetID: "boxa"},
		}}}},
	}

	_, problems := resolveGateway(cfg, gw)
	if len(problems) == 0 {
		t.Fatal("a loopback-bound devnet on another machine must be refused, not dialed into this machine's own loopback")
	}
	for _, want := range []string{"boxa", "ws://127.0.0.1:8601"} {
		if !strings.Contains(problems[0], want) {
			t.Errorf("problem %q does not mention %q", problems[0], want)
		}
	}
}

// The same node on the SAME machine as the gateway is fine on loopback: the
// container reaches it through the host alias, which is exactly what
// ops.GatewayContainerConfig rewrites it to.
func TestGateways_SameMachineLoopbackUpstreamIsFine(t *testing.T) {
	cfg := config.Config{Targets: []config.Target{
		{ID: "here", Mode: "local", Wire: &catalog.WireConfig{ChainID: 369, ExecID: "reth"}},
	}}
	gw := config.Gateway{
		ID:        "default",
		Placement: config.GatewayPlacement{TargetID: "here", Backend: "docker"},
		Config: catalog.GatewayConfig{Networks: []catalog.GatewayNetwork{{ChainID: 369, Upstreams: []catalog.GatewayUpstream{
			{ID: "node", Kind: catalog.UpstreamManagedNode, TargetID: "here"},
		}}}},
	}
	resolved, problems := resolveGateway(cfg, gw)
	if len(problems) != 0 {
		t.Fatalf("problems: %v", problems)
	}
	if got := resolved.Networks[0].Upstreams[0].Endpoint; got != "http://127.0.0.1:8545" {
		t.Errorf("endpoint: got %q", got)
	}
}

// ---------------------------------------------------------------------
// actions
// ---------------------------------------------------------------------

// A gateway with no chain must not be offered "create": eRPC refuses a
// project with no networks, so the button could only fail — and the card has
// to say so instead of leaving a dead button.
func TestGateways_NoChainsOffersNoCreateAndSaysWhy(t *testing.T) {
	a := gatewayServer(t)
	v := addGateway(t, a, "empty", "local", catalog.GatewayConfig{Port: 4300})

	for _, act := range v.Actions {
		if act == actionCreate || act == actionRecreate {
			t.Fatalf("offered %q for a gateway with no chains: eRPC would refuse the config", act)
		}
	}
	if v.Blocked == "" {
		t.Fatal("no actions and no reason — the operator is left guessing")
	}

	res := a.do(t, "POST", "/api/gateways/empty/provision", nil)
	body := decode[errorDetail](t, res)
	if res.StatusCode != http.StatusBadRequest || body.Code != codeNotConfigured {
		t.Fatalf("provision with no chains: got %d/%q, want 400/%s", res.StatusCode, body.Code, codeNotConfigured)
	}
}

func TestGateways_WipeRequiresTypedConfirmation(t *testing.T) {
	a := gatewayServer(t)
	addGateway(t, a, "default", "local", catalog.GatewayConfig{
		Port:     4100,
		Networks: []catalog.GatewayNetwork{{ChainID: 369, Upstreams: []catalog.GatewayUpstream{{ID: "p", Endpoint: "https://rpc.pulsechain.com"}}}},
	})
	for _, confirm := range []string{"", "yes", "erpc"} {
		res := a.do(t, "POST", "/api/gateways/default/wipe", map[string]string{"Confirm": confirm})
		res.Body.Close()
		if res.StatusCode != http.StatusBadRequest {
			t.Fatalf("confirm %q: got %d, want 400", confirm, res.StatusCode)
		}
	}
}

func TestGateways_UnknownGatewayIs404WithATypedCode(t *testing.T) {
	a := gatewayServer(t)
	res := a.do(t, "GET", "/api/gateways/nope", nil)
	body := decode[errorDetail](t, res)
	if res.StatusCode != http.StatusNotFound || body.Code != codeGatewayNotFound {
		t.Fatalf("got %d/%q, want 404/%s", res.StatusCode, body.Code, codeGatewayNotFound)
	}
}

// Deleting a gateway forgets the configuration; it does not silently destroy
// a running container. Saying which happened is the difference between a
// deliberate choice and an unpleasant surprise.
func TestGateways_DeleteForgetsButSaysTheContainerRemains(t *testing.T) {
	a := gatewayServer(t)
	addGateway(t, a, "default", "local", catalog.GatewayConfig{
		Port:     4100,
		Networks: []catalog.GatewayNetwork{{ChainID: 369, Upstreams: []catalog.GatewayUpstream{{ID: "p", Endpoint: "https://rpc.pulsechain.com"}}}},
	})
	body := decode[map[string]string](t, a.do(t, "DELETE", "/api/gateways/default", nil))
	if body["note"] == "" {
		t.Error("delete must state that the container was left running")
	}
	res := a.do(t, "GET", "/api/gateways/default", nil)
	res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("after delete: got %d, want 404", res.StatusCode)
	}
}

// ---------------------------------------------------------------------
// config validation
// ---------------------------------------------------------------------

func TestGateways_ConfigRejectsWhatWouldFailLater(t *testing.T) {
	a := gatewayServer(t)
	addGateway(t, a, "default", "local", catalog.GatewayConfig{Port: 4100})

	for name, body := range map[string]any{
		"an upstream with no scheme": map[string]any{"Networks": []any{map[string]any{"ChainID": 1337, "Upstreams": []any{map[string]any{"Endpoint": "127.0.0.1:8545"}}}}},
		"a chain with no upstream":   map[string]any{"Networks": []any{map[string]any{"ChainID": 1337}}},
		"a duplicate upstream id":    map[string]any{"Networks": []any{map[string]any{"ChainID": 1, "Upstreams": []any{map[string]any{"ID": "x", "Endpoint": "https://a.example"}, map[string]any{"ID": "x", "Endpoint": "https://b.example"}}}}},
		"a managed ref with no machine": map[string]any{"Networks": []any{map[string]any{"ChainID": 1337, "Upstreams": []any{
			map[string]any{"Kind": catalog.UpstreamManagedDevnet},
		}}}},
	} {
		t.Run(name, func(t *testing.T) {
			res := a.do(t, "PUT", "/api/gateways/default/config", body)
			defer res.Body.Close()
			if res.StatusCode != http.StatusBadRequest {
				t.Fatalf("got %d, want 400", res.StatusCode)
			}
		})
	}
}

// A managed upstream carries no endpoint of its own, so validation must not
// reject it for the very thing that makes it a reference.
func TestGateways_ConfigAcceptsAManagedReferenceWithNoURL(t *testing.T) {
	a := gatewayServer(t)
	addGateway(t, a, "default", "local", catalog.GatewayConfig{Port: 4100})

	res := a.do(t, "PUT", "/api/gateways/default/config", catalog.GatewayConfig{
		Port: 4100,
		Networks: []catalog.GatewayNetwork{{ChainID: catalog.DevnetChainID, Upstreams: []catalog.GatewayUpstream{
			{ID: "devnet", Kind: catalog.UpstreamManagedDevnet, TargetID: "local"},
		}}},
	})
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("got %d, want 200", res.StatusCode)
	}
}

// ---------------------------------------------------------------------
// the picker's sources
// ---------------------------------------------------------------------

// The picker's presets come from the catalog, so it cannot drift from what
// the app supports — and the devnet is the one preset that can provision its
// own upstream.
func TestGateways_PresetsComeFromTheCatalog(t *testing.T) {
	a := gatewayServer(t)
	body := decode[gatewaysResponse](t, a.do(t, "GET", "/api/gateways", nil))

	want := map[int]string{}
	for _, n := range catalog.Networks() {
		want[n.ChainID] = n.Name
	}
	seen := map[int]string{}
	devnets := 0
	for _, p := range body.Presets {
		seen[p.ChainID] = p.Name
		if p.Devnet {
			devnets++
			if p.ChainID != catalog.DevnetChainID {
				t.Errorf("the devnet preset must be chain %d, got %d", catalog.DevnetChainID, p.ChainID)
			}
		}
	}
	for id, name := range want {
		if seen[id] != name {
			t.Errorf("preset for chain %d: got %q, want the catalog's %q", id, seen[id], name)
		}
	}
	if devnets != 1 {
		t.Errorf("want exactly one self-provisioning devnet preset, got %d", devnets)
	}
}

// Sources are what "add an endpoint" offers: the real things in the fleet,
// each with the machine it is on, so an operator picks a node rather than
// typing a URL that goes stale.
func TestGateways_SourcesEnumerateTheFleet(t *testing.T) {
	a := gatewayServer(t)
	putConfig(t, a, svcDevnet, catalog.DevnetConfig{HTTPPort: 8600, WSPort: 8601})

	body := decode[gatewaysResponse](t, a.do(t, "GET", "/api/gateways", nil))
	var found bool
	for _, s := range body.Sources {
		if s.Kind == catalog.UpstreamManagedDevnet && s.TargetID == "local" && s.ChainID == catalog.DevnetChainID {
			found = true
			if s.Endpoint != "http://127.0.0.1:8600" {
				t.Errorf("source endpoint: got %q", s.Endpoint)
			}
		}
	}
	if !found {
		t.Fatalf("the local devnet is missing from the sources: %+v", body.Sources)
	}
}

// FOUND BY RUNNING IT: an external endpoint the operator has marked as THEIRS
// was labelled "public endpoint · 192.168.3.22" on a row whose Role column
// said "Yours". Both came from the same upstream, and they contradicted each
// other.
//
// This is not a rare shape. An external upstream carrying Local is the ordinary
// way to front a node valve-node-app does not manage — someone else's box, a
// node installed by hand — and it is the tier that decides the intended share,
// so calling it public where the operator can read it undermines the one
// column that explains why the share bar is amber.
func TestGateways_AnEndpointTheOperatorOwnsIsNotCalledPublic(t *testing.T) {
	cfg := config.Config{Targets: []config.Target{{ID: "here", Mode: "local"}}}
	gw := config.Gateway{
		ID:        "default",
		Placement: config.GatewayPlacement{TargetID: "here", Backend: "docker"},
		Config: catalog.GatewayConfig{Networks: []catalog.GatewayNetwork{{ChainID: 369, Upstreams: []catalog.GatewayUpstream{
			{ID: "mine", Endpoint: "ws://192.168.3.22:8546", Local: true},
			{ID: "theirs", Endpoint: "https://rpc.pulsechain.com", Local: false},
		}}}},
	}

	_, mine, err := resolveUpstream(cfg, gw, gw.Config.Networks[0].Upstreams[0])
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if strings.Contains(mine, "public") {
		t.Errorf("label %q calls the operator's own endpoint public", mine)
	}
	if !strings.Contains(mine, "192.168.3.22") {
		t.Errorf("label %q must still name the host — that is what someone recognises", mine)
	}

	_, theirs, err := resolveUpstream(cfg, gw, gw.Config.Networks[0].Upstreams[1])
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if !strings.Contains(theirs, "public endpoint") {
		t.Errorf("an endpoint filed in the fallback tier IS a public one here: %q", theirs)
	}
}
