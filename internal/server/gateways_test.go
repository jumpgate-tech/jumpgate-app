package server

import (
	"encoding/json"
	"io"
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

// Distinct container names still matter — docker run --name refuses a
// duplicate — but two gateways now mean two MACHINES. The same-machine case is
// covered by TestGateways_RefuseASecondGatewayOnOneMachine below.
func TestGateways_TwoGatewaysCoexistWithDistinctContainers(t *testing.T) {
	a := gatewayServer(t)

	res := a.do(t, "POST", "/api/targets", map[string]any{"id": "second", "mode": "local"})
	res.Body.Close()

	net := []catalog.GatewayNetwork{{ChainID: 369, Upstreams: []catalog.GatewayUpstream{
		{ID: "public", Endpoint: "https://rpc.pulsechain.com"},
	}}}
	first := addGateway(t, a, "default", "local", catalog.GatewayConfig{Port: 4100, Networks: net})
	second := addGateway(t, a, "edge", "second", catalog.GatewayConfig{Port: 4200, Networks: net})

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

// A freshly created gateway fronts NO chains — and in particular does not
// auto-seed the local devnet (evm:1337). The RPC screen's devnet opt-in rests
// on this: a newcomer must not land on a gateway already advertising an
// unrecognized chain. "An empty one is the intended starting state" is what
// handleGatewayCreate says in its own comment; this pins it so a future change
// that seeds a default network cannot pass unnoticed.
func TestGateways_FreshGatewayFrontsNoChains(t *testing.T) {
	a := gatewayServer(t)
	// The UI's create flow sends an empty network list.
	addGateway(t, a, "default", "local", catalog.GatewayConfig{Port: 4100})

	got := decode[gatewayView](t, a.do(t, "GET", "/api/gateways/default", nil))
	for _, net := range got.Config.Networks {
		if net.ChainID == catalog.DevnetChainID {
			t.Errorf("a fresh gateway auto-seeded the devnet (evm:%d) — it must be opt-in", catalog.DevnetChainID)
		}
	}
	if n := len(got.Config.Networks); n != 0 {
		t.Errorf("a fresh gateway must front no chains, got %d: %+v", n, got.Config.Networks)
	}
}

// A gateway NAMES the machine it runs on, so a second one on that machine is a
// second managed eRPC container: overlapping chains, two state pollers against
// one node, and only one of them reachable through the reverse proxy.
func TestGateways_RefuseASecondGatewayOnOneMachine(t *testing.T) {
	a := gatewayServer(t)
	addGateway(t, a, "default", "local", catalog.GatewayConfig{Port: 4100})

	res := a.do(t, "POST", "/api/gateways", map[string]any{
		"id":        "edge",
		"placement": map[string]string{"targetId": "local", "backend": "docker"},
	})
	defer res.Body.Close()

	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", res.StatusCode)
	}
	b, _ := io.ReadAll(res.Body)
	if !strings.Contains(string(b), "default") {
		t.Errorf("the error must name the gateway already on that machine: %s", b)
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

// An operator-set Name overrides the derived label; an empty Name leaves the
// derived label exactly as it was.
func TestGateways_UpstreamNameOverridesTheDerivedLabel(t *testing.T) {
	cfg := config.Config{Targets: []config.Target{{ID: "here", Mode: "local"}}}
	gw := config.Gateway{
		ID:        "default",
		Placement: config.GatewayPlacement{TargetID: "here", Backend: "docker"},
		Config: catalog.GatewayConfig{Networks: []catalog.GatewayNetwork{{ChainID: 369, Upstreams: []catalog.GatewayUpstream{
			{ID: "named", Endpoint: "https://rpc.pulsechain.com", Name: "my node"},
			{ID: "unnamed", Endpoint: "https://rpc.pulsechain.com"},
		}}}},
	}

	_, named, err := resolveUpstream(cfg, gw, gw.Config.Networks[0].Upstreams[0])
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if named != "my node" {
		t.Errorf("label = %q, want the operator-set Name %q", named, "my node")
	}

	_, unnamed, err := resolveUpstream(cfg, gw, gw.Config.Networks[0].Upstreams[1])
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if !strings.Contains(unnamed, "public endpoint") {
		t.Errorf("an upstream with no Name should keep its derived label: %q", unnamed)
	}
}

// A set Name must survive resolution FAILING, not just succeeding — that is
// exactly when an operator most needs to recognise their own row (Task 10's
// rename flow can be applied to an already-unhealthy endpoint). This drives
// the real view path (networkViews, via GET), not resolveUpstream directly,
// because the failure-path label is decided there, not in resolveUpstream.
func TestGateways_UpstreamNameSurvivesAnUnresolvedUpstream(t *testing.T) {
	a := gatewayServer(t)
	addGateway(t, a, "default", "local", catalog.GatewayConfig{
		Port: 4100,
		Networks: []catalog.GatewayNetwork{
			{ChainID: catalog.DevnetChainID, Upstreams: []catalog.GatewayUpstream{
				// There is no devnet configured on "local", so this fails to resolve.
				{ID: "devnet", Kind: catalog.UpstreamManagedDevnet, TargetID: "local", Name: "my node"},
			}},
		},
	})

	got := decode[gatewayView](t, a.do(t, "GET", "/api/gateways/default", nil))
	if len(got.Networks) != 1 || len(got.Networks[0].Upstreams) != 1 {
		t.Fatalf("networks/upstreams = %+v, want exactly one of each", got.Networks)
	}
	up := got.Networks[0].Upstreams[0]
	if up.Problem == "" {
		t.Fatal("this upstream must still be unresolvable — that is the case under test")
	}
	if up.Label != "my node" {
		t.Errorf("label = %q, want the operator-set Name %q to survive an unresolved upstream", up.Label, "my node")
	}
}

// ---------------------------------------------------------------------
// orphaned containers — what a merge left running
// ---------------------------------------------------------------------

// The operator has to be told what is still running, or a merged-away gateway
// keeps serving stale config forever with nothing pointing at it.
func TestGatewayListReportsOrphans(t *testing.T) {
	a := gatewayServer(t)
	addGateway(t, a, "default", "local", catalog.GatewayConfig{Port: 4100})

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	cfg.Orphans = []config.OrphanedContainer{{
		ContainerName: "valve-node-app-erpc-edge", TargetID: "local", MergedInto: "default",
	}}
	if err := cfg.Save(); err != nil {
		t.Fatalf("save config: %v", err)
	}

	body := decode[gatewaysResponse](t, a.do(t, "GET", "/api/gateways", nil))
	if len(body.Orphans) != 1 || body.Orphans[0].ContainerName != "valve-node-app-erpc-edge" {
		t.Fatalf("the leftover container must appear in the listing: %+v", body.Orphans)
	}
	if body.Orphans[0].TargetID != "local" || body.Orphans[0].MergedInto != "default" {
		t.Errorf("orphan record: got %+v", body.Orphans[0])
	}
}

// A merged-away gateway's id is freed, so a new gateway can re-claim the very
// container name a stale banner still says to `docker rm -f`. Left standing,
// that banner points the operator at a container this app now manages. But a
// container name is only unique within ONE docker engine, so clearing a
// record must also check it is on the SAME machine the new gateway lands on
// — otherwise creating "edge" on one machine silently erases a legitimate,
// still-running leftover named "edge" on an unrelated one.
func TestGatewayCreateClearsAStaleOrphanRecordForItsOwnContainers(t *testing.T) {
	a := gatewayServer(t)

	res := a.do(t, "POST", "/api/targets", map[string]any{"id": "second", "mode": "local"})
	res.Body.Close()

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	cfg.Orphans = []config.OrphanedContainer{
		{ContainerName: "valve-node-app-erpc-edge", TargetID: "local", MergedInto: "default"},
		{ContainerName: "valve-node-app-caddy-edge", TargetID: "local", MergedInto: "default"},
		{ContainerName: "valve-node-app-erpc-other", TargetID: "local", MergedInto: "default"},
		{ContainerName: "valve-node-app-erpc-edge", TargetID: "second", MergedInto: "default"},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("save config: %v", err)
	}

	addGateway(t, a, "edge", "local", catalog.GatewayConfig{Port: 4100})

	body := decode[gatewaysResponse](t, a.do(t, "GET", "/api/gateways", nil))
	if len(body.Orphans) != 2 {
		t.Fatalf("only same-machine records for names this gateway re-claimed may go: %+v", body.Orphans)
	}
	byTarget := map[string]string{}
	for _, o := range body.Orphans {
		byTarget[o.TargetID] = o.ContainerName
	}
	if byTarget["local"] != "valve-node-app-erpc-other" {
		t.Errorf("an unrelated leftover on the same machine must survive: got %+v", body.Orphans)
	}
	if byTarget["second"] != "valve-node-app-erpc-edge" {
		t.Errorf("a same-named leftover on a DIFFERENT machine must survive — the name is only unique per docker engine: got %+v", body.Orphans)
	}
}

// Dismissing an orphan forgets the RECORD only. It must never touch the
// container: this app never stops a container it did not just start, which is
// the same rule handleGatewayDelete follows for a gateway's own container.
func TestOrphanDismissForgetsTheRecordOnly(t *testing.T) {
	a := gatewayServer(t)
	addGateway(t, a, "default", "local", catalog.GatewayConfig{Port: 4100})

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	cfg.Orphans = []config.OrphanedContainer{{
		ContainerName: "valve-node-app-erpc-edge", TargetID: "local", MergedInto: "default",
	}}
	if err := cfg.Save(); err != nil {
		t.Fatalf("save config: %v", err)
	}

	res := a.do(t, "DELETE", "/api/orphans/valve-node-app-erpc-edge", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("want 200, got %d", res.StatusCode)
	}
	res.Body.Close()

	body := decode[gatewaysResponse](t, a.do(t, "GET", "/api/gateways", nil))
	if len(body.Orphans) != 0 {
		t.Errorf("the record must be gone after dismissal: %+v", body.Orphans)
	}

	// Dismissing again finds nothing left to forget.
	res = a.do(t, "DELETE", "/api/orphans/valve-node-app-erpc-edge", nil)
	defer res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("dismissing twice: got %d, want 404", res.StatusCode)
	}
}

// ---------------------------------------------------------------------
// the known set — GET /api/gateways/{gid}/knownset/{chainId}
// ---------------------------------------------------------------------

// The set is offered with what is already configured marked, so the count the
// operator sees before clicking matches what actually lands.
func TestKnownSetMarksWhatIsAlreadyConfigured(t *testing.T) {
	a := gatewayServer(t)
	addGateway(t, a, "default", "local", catalog.GatewayConfig{
		Port: 4100,
		Networks: []catalog.GatewayNetwork{{ChainID: 1, Upstreams: []catalog.GatewayUpstream{
			{ID: "public-1-1", Kind: catalog.UpstreamExternal, Endpoint: "https://eth.drpc.org"},
		}}},
	})

	body := decode[knownSetResponse](t, a.do(t, "GET", "/api/gateways/default/knownset/1", nil))

	if len(body.Endpoints) == 0 {
		t.Fatal("the set must be offered for chain 1")
	}
	var marked int
	for _, e := range body.Endpoints {
		if e.URL == "https://eth.drpc.org" && e.AlreadyAdded {
			marked++
		}
	}
	if marked != 1 {
		t.Errorf("the configured endpoint must come back marked, got %d marks: %+v", marked, body.Endpoints)
	}
	if !body.UsingDefaultKey {
		t.Error("with no key stored, the set must report it is using the shared demo key")
	}
}

// With a key stored, the set must resolve valve's entry against THAT key rather
// than the shared demo one, and say so — on EVERY chain, because the key is an
// account rather than a chain.
//
// The key is seeded through config.Load()/cfg.Save() — the same accessor the
// server uses under s.loadConfig() — rather than through Settings, because what
// is under test is the read side.
//
// The stored key is SHORT on purpose. A short operator key used to slip past
// redaction's length gate and be serialised into these URLs; it is a secret at
// any length, so the assertion is that the URL comes back in placeholder form —
// which is also proof the operator's key was the one substituted, since the demo
// default would have been left in the URL literally.
func TestKnownSetUsesAStoredKeyOverTheDefault(t *testing.T) {
	a := gatewayServer(t)
	addGateway(t, a, "default", "local", catalog.GatewayConfig{Port: 4100})

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	cfg.ProviderKeys = map[string]string{config.ValveKeyPlaceholder: "vk_mine"}
	if err := cfg.Save(); err != nil {
		t.Fatalf("save config: %v", err)
	}

	body := decode[knownSetResponse](t, a.do(t, "GET", "/api/gateways/default/knownset/1", nil))
	if body.UsingDefaultKey {
		t.Error("a stored key must not be reported as the shared demo key")
	}
	var found bool
	for _, e := range body.Endpoints {
		if e.Provider == "valve" {
			found = true
			if strings.Contains(e.URL, "vk_mine") {
				t.Errorf("the operator's key must not reach the browser, short or not: %q", e.URL)
			}
			if !strings.Contains(e.URL, "${"+config.ValveKeyPlaceholder+"}") {
				t.Errorf("valve's URL must come back in placeholder form once a key of the operator's own is stored: %q", e.URL)
			}
		}
	}
	if !found {
		t.Fatal("chain 1's set must include a valve entry")
	}

	// One key, every chain: the old per-chain store could say "valve on this
	// chain but not that one", and that was a misreading of what a key is.
	body = decode[knownSetResponse](t, a.do(t, "GET", "/api/gateways/default/knownset/369", nil))
	if body.UsingDefaultKey {
		t.Error("the operator's key applies to chain 369 too — a key is an account, not a chain")
	}
}

// A per-chain valve key stored by an older build still resolves the set, having
// been folded into the one provider key on load. This is the migration seen
// from the route that consumes it.
//
// The migrated key must never appear in the response (see
// TestKnownSetNeverReturnsTheKeyItself), so this cannot assert "the URL
// contains vk_legacy" the way it used to. Instead it proves the key was
// actually USED for resolution the same way AlreadyAdded proves anything:
// the gateway already has an upstream stored at the URL vk_legacy resolves
// to, and that only comes back marked if resolution used that exact key.
func TestKnownSetHonoursAMigratedPerChainKey(t *testing.T) {
	a := gatewayServer(t)
	addGateway(t, a, "default", "local", catalog.GatewayConfig{
		Port: 4100,
		Networks: []catalog.GatewayNetwork{{ChainID: 1, Upstreams: []catalog.GatewayUpstream{
			{ID: "valve-1", Kind: catalog.UpstreamExternal, Endpoint: "https://one.valve.city/rpc/vk_legacy/evm/1"},
		}}},
	})

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	cfg.ValveKeys = map[int]string{1: "vk_legacy"}
	if err := cfg.Save(); err != nil {
		t.Fatalf("save config: %v", err)
	}

	body := decode[knownSetResponse](t, a.do(t, "GET", "/api/gateways/default/knownset/1", nil))
	if body.UsingDefaultKey {
		t.Error("a key an older build stored per chain must survive the upgrade")
	}
	var found bool
	for _, e := range body.Endpoints {
		if e.Provider != "valve" || e.WebSocket {
			continue
		}
		found = true
		if strings.Contains(e.URL, "vk_legacy") {
			t.Errorf("the migrated key must not leak into the response: %q", e.URL)
		}
		if !e.AlreadyAdded {
			t.Errorf("resolving with the migrated key must match the already-configured endpoint, got AlreadyAdded=false for %q", e.URL)
		}
	}
	if !found {
		t.Fatal("chain 1's set must include a valve http entry")
	}
}

// The response says WHICH key is in play and never what it is: not as a bare
// "key" field, and not inside a resolved URL either. It used to send the key
// itself both ways; now every templated endpoint goes through the same
// redactKeys seam discovery uses, so this asserts the secret is absent from
// the raw body outright, not merely from one named field.
func TestKnownSetNeverReturnsTheKeyItself(t *testing.T) {
	a := gatewayServer(t)
	addGateway(t, a, "default", "local", catalog.GatewayConfig{Port: 4100})

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	cfg.ProviderKeys = map[string]string{config.ValveKeyPlaceholder: "vk_secret_do_not_send"}
	if err := cfg.Save(); err != nil {
		t.Fatalf("save config: %v", err)
	}

	res := a.do(t, "GET", "/api/gateways/default/knownset/1", nil)
	defer res.Body.Close()
	raw, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	if strings.Contains(string(raw), "vk_secret_do_not_send") {
		t.Errorf("the stored key must not appear anywhere in the response: %s", raw)
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if _, ok := fields["key"]; ok {
		t.Errorf("the response still carries a bare key field: %s", raw)
	}
	if _, ok := fields["usingDefaultKey"]; !ok {
		t.Errorf("usingDefaultKey must survive: it is what the UI actually needs: %s", raw)
	}
}

// ---------------------------------------------------------------------
// the other half of the redaction: filling the slot back in on the way to
// storage
// ---------------------------------------------------------------------

// Discovery hands the browser the ${PLACEHOLDER} form of a templated endpoint
// so the key never leaves this process. That is only honest if the string the
// browser posts back still WORKS, so the save path fills the slot in again —
// the stored endpoint is the URL eRPC will dial, exactly as it always was.
func TestGatewayPutConfig_FillsAProviderSlotFromTheStoredKey(t *testing.T) {
	a := gatewayServer(t)
	addGateway(t, a, "default", "local", catalog.GatewayConfig{Port: 4100})

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	cfg.ProviderKeys = map[string]string{"INFURA_API_KEY": "sk_infura_secret"}
	if err := cfg.Save(); err != nil {
		t.Fatalf("save config: %v", err)
	}

	res := a.do(t, "PUT", "/api/gateways/default/config", catalog.GatewayConfig{
		Port: 4100,
		Networks: []catalog.GatewayNetwork{{ChainID: 1, Upstreams: []catalog.GatewayUpstream{
			{ID: "public-1-1", Kind: catalog.UpstreamExternal, Endpoint: "https://mainnet.infura.io/v3/${INFURA_API_KEY}"},
		}}},
	})
	if res.StatusCode != http.StatusOK {
		defer res.Body.Close()
		raw, _ := io.ReadAll(res.Body)
		t.Fatalf("PUT config: got %d, want 200: %s", res.StatusCode, raw)
	}
	res.Body.Close()

	stored, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	gw, ok := stored.FindGateway("default")
	if !ok {
		t.Fatal("the gateway vanished")
	}
	got := gw.Config.Networks[0].Upstreams[0].Endpoint
	if got != "https://mainnet.infura.io/v3/sk_infura_secret" {
		t.Errorf("stored endpoint = %q, want the slot filled in — eRPC cannot dial a ${...}", got)
	}
}

// TestGatewayPutConfig_FillsAnEmptyTLSHostname pins the bug the one-click
// setup flow (home.ts/panel.ts setupEndpoint) hit live: it builds the same
// internalTLSConfig(networks) — TLS.Enabled true, Hostname "" — twice, once
// for the create (which fills the hostname via defaultTLSHostname) and once
// for this PUT once the networks are known. Before this fix, PUT never
// called defaultTLSHostname, so the second call always 400'd with "hostname
// is required" even though the identical shape had just succeeded on create.
func TestGatewayPutConfig_FillsAnEmptyTLSHostname(t *testing.T) {
	a := gatewayServer(t)
	addGateway(t, a, "default", "local", catalog.GatewayConfig{Port: 4100})

	res := a.do(t, "PUT", "/api/gateways/default/config", catalog.GatewayConfig{
		Port: 4100,
		Networks: []catalog.GatewayNetwork{{ChainID: 1, Upstreams: []catalog.GatewayUpstream{
			{ID: "public-1-1", Kind: catalog.UpstreamExternal, Endpoint: "https://mainnet.example.com"},
		}}},
		TLS: &catalog.GatewayTLS{Enabled: true, Hostname: "", CertSource: catalog.CertInternal},
	})
	if res.StatusCode != http.StatusOK {
		defer res.Body.Close()
		raw, _ := io.ReadAll(res.Body)
		t.Fatalf("PUT config with an empty TLS hostname: got %d, want 200 (the server should fill one in, exactly as create does): %s", res.StatusCode, raw)
	}
	res.Body.Close()

	stored, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	gw, ok := stored.FindGateway("default")
	if !ok {
		t.Fatal("the gateway vanished")
	}
	if gw.Config.TLS == nil || strings.TrimSpace(gw.Config.TLS.Hostname) == "" {
		t.Errorf("stored TLS.Hostname is still empty; want a generated name, as create would produce")
	}
}

// ---------------------------------------------------------------------
// the gateway views never carry a provider key either
// ---------------------------------------------------------------------

// storeProviderKeys puts keys in the stored config, which is what
// providerKeys() reads and therefore what redaction matches against.
func storeProviderKeys(t *testing.T, keys map[string]string) {
	t.Helper()
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	cfg.ProviderKeys = keys
	if err := cfg.Save(); err != nil {
		t.Fatalf("save config: %v", err)
	}
}

// The stored endpoint of a templated upstream carries the real key — that is
// resolveUpstreamKeys doing its job — and the RPC screen polls these two
// routes continuously. Both the per-upstream row and the whole round-trip
// Config must come back with the ${NAME} slot, and this asserts it against the
// RAW body rather than one field, because a key that leaks through some other
// string is leaked just the same.
func TestGatewayViews_NeverReturnAProviderKey(t *testing.T) {
	a := gatewayServer(t)
	storeProviderKeys(t, map[string]string{
		config.ValveKeyPlaceholder: "vk_operator_secret",
		"INFURA_API_KEY":           "sk_infura_operator_secret",
	})
	addGateway(t, a, "default", "local", catalog.GatewayConfig{
		Port: 4100,
		Networks: []catalog.GatewayNetwork{{ChainID: 1, Upstreams: []catalog.GatewayUpstream{
			{ID: "valve-1", Kind: catalog.UpstreamExternal, Endpoint: "https://one.valve.city/rpc/${VALVE_API_KEY}/evm/1"},
			{ID: "infura-1", Kind: catalog.UpstreamExternal, Endpoint: "https://mainnet.infura.io/v3/${INFURA_API_KEY}"},
		}}},
	})

	// The key really was stored resolved — otherwise the rest of this test
	// would be proving redaction of a string that never had a secret in it.
	stored, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	gw, _ := stored.FindGateway("default")
	if got := gw.Config.Networks[0].Upstreams[0].Endpoint; got != "https://one.valve.city/rpc/vk_operator_secret/evm/1" {
		t.Fatalf("stored endpoint = %q, want the resolved URL — nothing here is worth asserting otherwise", got)
	}

	for _, path := range []string{"/api/gateways", "/api/gateways/default"} {
		res := a.do(t, "GET", path, nil)
		raw, err := io.ReadAll(res.Body)
		res.Body.Close()
		if err != nil {
			t.Fatalf("%s: read body: %v", path, err)
		}
		for _, secret := range []string{"vk_operator_secret", "sk_infura_operator_secret"} {
			if strings.Contains(string(raw), secret) {
				t.Errorf("%s: the stored key %q must not appear anywhere in the response: %s", path, secret, raw)
			}
		}
		if !strings.Contains(string(raw), "${VALVE_API_KEY}") || !strings.Contains(string(raw), "${INFURA_API_KEY}") {
			t.Errorf("%s: the placeholder form must come back in its place, so the editor can still round-trip it: %s", path, raw)
		}
	}
}

// The hazard the deep copy exists for. gw.Config arrives as a shallow struct
// copy of loaded configuration: Networks and each network's Upstreams are
// SLICES, so redacting "the copy" in place would write ${NAME} into the very
// configuration this process renders from and saves. That is a display concern
// turned into data loss.
func TestRedactedGatewayConfig_LeavesTheSourceUntouched(t *testing.T) {
	keys := map[string]string{"INFURA_API_KEY": "sk_infura_secret"}
	src := catalog.GatewayConfig{Networks: []catalog.GatewayNetwork{{
		ChainID: 1,
		Upstreams: []catalog.GatewayUpstream{
			{ID: "infura-1", Kind: catalog.UpstreamExternal, Endpoint: "https://mainnet.infura.io/v3/sk_infura_secret"},
		},
	}}}

	out := redactedGatewayConfig(src, keys)

	if got := out.Networks[0].Upstreams[0].Endpoint; got != "https://mainnet.infura.io/v3/${INFURA_API_KEY}" {
		t.Errorf("view endpoint = %q, want the placeholder form", got)
	}
	if got := src.Networks[0].Upstreams[0].Endpoint; got != "https://mainnet.infura.io/v3/sk_infura_secret" {
		t.Fatalf("the SOURCE config was rewritten to %q — the redaction is aliasing the loaded config's backing array", got)
	}

	// The other direction, so a future copy that shares an array is caught by
	// this test whichever side writes first.
	out.Networks[0].Upstreams[0].Endpoint = "https://elsewhere.example/"
	out.Networks[0].ChainID = 999
	if src.Networks[0].Upstreams[0].Endpoint != "https://mainnet.infura.io/v3/sk_infura_secret" || src.Networks[0].ChainID != 1 {
		t.Fatalf("writing through the view changed the source: %+v", src.Networks[0])
	}
}

// The same hazard through the real route: a poll of the RPC screen must leave
// the stored configuration exactly as it found it.
//
// This is the outer guard, not the sharp one — loadConfig re-reads from disk
// per request today, so an aliased write would corrupt only that request's
// copy and never reach the file. The test above is what catches the aliasing
// itself; this one is what catches it the day a config is cached in memory.
func TestGatewayList_DoesNotRedactTheStoredConfig(t *testing.T) {
	a := gatewayServer(t)
	storeProviderKeys(t, map[string]string{"INFURA_API_KEY": "sk_infura_operator_secret"})
	addGateway(t, a, "default", "local", catalog.GatewayConfig{
		Port: 4100,
		Networks: []catalog.GatewayNetwork{{ChainID: 1, Upstreams: []catalog.GatewayUpstream{
			{ID: "infura-1", Kind: catalog.UpstreamExternal, Endpoint: "https://mainnet.infura.io/v3/${INFURA_API_KEY}"},
		}}},
	})

	a.do(t, "GET", "/api/gateways", nil).Body.Close()
	a.do(t, "GET", "/api/gateways/default", nil).Body.Close()

	stored, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	gw, ok := stored.FindGateway("default")
	if !ok {
		t.Fatal("the gateway vanished")
	}
	if got := gw.Config.Networks[0].Upstreams[0].Endpoint; got != "https://mainnet.infura.io/v3/sk_infura_operator_secret" {
		t.Errorf("stored endpoint = %q after two reads — a read must never rewrite the configuration", got)
	}
}

// The round trip the RPC page actually performs: read the view, edit its
// Config, PUT it back. With redaction it now posts ${NAME} where a key used to
// be, and this proves the save path fills it again — the stored endpoint is
// still the URL eRPC dials, and the upstream the operator did NOT touch is not
// quietly degraded into a template.
func TestGatewayView_ConfigRoundTripsBackToRealURLs(t *testing.T) {
	a := gatewayServer(t)
	storeProviderKeys(t, map[string]string{"INFURA_API_KEY": "sk_infura_operator_secret"})
	view := addGateway(t, a, "default", "local", catalog.GatewayConfig{
		Port: 4100,
		Networks: []catalog.GatewayNetwork{{ChainID: 1, Upstreams: []catalog.GatewayUpstream{
			{ID: "infura-1", Kind: catalog.UpstreamExternal, Endpoint: "https://mainnet.infura.io/v3/${INFURA_API_KEY}"},
		}}},
	})

	if got := view.Config.Networks[0].Upstreams[0].Endpoint; got != "https://mainnet.infura.io/v3/${INFURA_API_KEY}" {
		t.Fatalf("view config endpoint = %q, want the placeholder form the editor round-trips", got)
	}

	// Exactly what the page does: take the view's config, add an upstream,
	// post the whole thing back.
	edited := view.Config
	edited.Networks[0].Upstreams = append(edited.Networks[0].Upstreams,
		catalog.GatewayUpstream{ID: "public-1", Kind: catalog.UpstreamExternal, Endpoint: "https://rpc.example.com"})

	res := a.do(t, "PUT", "/api/gateways/default/config", edited)
	if res.StatusCode != http.StatusOK {
		defer res.Body.Close()
		raw, _ := io.ReadAll(res.Body)
		t.Fatalf("PUT config: got %d, want 200: %s", res.StatusCode, raw)
	}
	res.Body.Close()

	stored, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	gw, ok := stored.FindGateway("default")
	if !ok {
		t.Fatal("the gateway vanished")
	}
	ups := gw.Config.Networks[0].Upstreams
	if len(ups) != 2 {
		t.Fatalf("stored %d upstreams, want 2 — the edit did not survive", len(ups))
	}
	if got := ups[0].Endpoint; got != "https://mainnet.infura.io/v3/sk_infura_operator_secret" {
		t.Errorf("stored endpoint = %q, want the real resolved URL — a stored ${...} is an upstream eRPC cannot dial", got)
	}
	if got := ups[1].Endpoint; got != "https://rpc.example.com" {
		t.Errorf("the added upstream = %q, want it stored verbatim", got)
	}
}

// Full stack, not just the resolver: PUT a config with an upstream Name,
// GET the gateway back, and check the JSON label the RPC screen actually
// renders. This is what proves the JSON round-trip (config.Save/Load) keeps
// Name, not just that resolveUpstream honours it in memory.
func TestGatewayView_UpstreamNameRoundTripsToTheLabel(t *testing.T) {
	a := gatewayServer(t)
	addGateway(t, a, "default", "local", catalog.GatewayConfig{
		Port: 4100,
		Networks: []catalog.GatewayNetwork{{ChainID: 1, Upstreams: []catalog.GatewayUpstream{
			{ID: "named-1", Kind: catalog.UpstreamExternal, Endpoint: "https://rpc.example.com", Name: "my node"},
		}}},
	})

	res := a.do(t, "GET", "/api/gateways/default", nil)
	view := decode[gatewayView](t, res)

	if len(view.Networks) != 1 || len(view.Networks[0].Upstreams) != 1 {
		t.Fatalf("view networks/upstreams = %+v, want exactly one of each", view.Networks)
	}
	if got := view.Networks[0].Upstreams[0].Label; got != "my node" {
		t.Errorf("label = %q, want the round-tripped Name %q", got, "my node")
	}

	stored, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	gw, ok := stored.FindGateway("default")
	if !ok {
		t.Fatal("the gateway vanished")
	}
	if got := gw.Config.Networks[0].Upstreams[0].Name; got != "my node" {
		t.Errorf("stored Name = %q, want it to survive the save/load round trip", got)
	}
}

// A REFUSED save must not read the key back out. Validation runs after the slot
// has been filled, and the bad-scheme error quotes the endpoint — so an
// unredacted 400 body would hand back any stored key to anyone who could post a
// deliberately malformed endpoint naming it, including a key that no upstream
// uses and the gateway view therefore never shows. Both doors are checked, and
// the whole raw body is searched rather than one field.
func TestGatewaySave_ARefusedEndpointDoesNotEchoTheKey(t *testing.T) {
	const secret = "sk_infura_never_echo_this"

	seed := func(t *testing.T) {
		t.Helper()
		cfg, err := config.Load()
		if err != nil {
			t.Fatalf("load config: %v", err)
		}
		cfg.ProviderKeys = map[string]string{"INFURA_API_KEY": secret}
		if err := cfg.Save(); err != nil {
			t.Fatalf("save config: %v", err)
		}
	}

	bad := catalog.GatewayConfig{
		Port: 4100,
		Networks: []catalog.GatewayNetwork{{ChainID: 1, Upstreams: []catalog.GatewayUpstream{
			{ID: "public-1-1", Kind: catalog.UpstreamExternal, Endpoint: "gopher://x/${INFURA_API_KEY}"},
		}}},
	}

	check := func(t *testing.T, res *http.Response) {
		t.Helper()
		defer res.Body.Close()
		raw, err := io.ReadAll(res.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if res.StatusCode != http.StatusBadRequest {
			t.Fatalf("got %d, want 400: %s", res.StatusCode, raw)
		}
		if strings.Contains(string(raw), secret) {
			t.Errorf("the stored key came back in the error body: %s", raw)
		}
		if !strings.Contains(string(raw), "${INFURA_API_KEY}") {
			t.Errorf("the error must still name the endpoint it refused, in placeholder form: %s", raw)
		}
	}

	t.Run("update", func(t *testing.T) {
		a := gatewayServer(t)
		addGateway(t, a, "default", "local", catalog.GatewayConfig{Port: 4100})
		seed(t)
		check(t, a.do(t, "PUT", "/api/gateways/default/config", bad))
	})

	t.Run("create", func(t *testing.T) {
		a := gatewayServer(t)
		seed(t)
		check(t, a.do(t, "POST", "/api/gateways", map[string]any{
			"id":        "default",
			"placement": map[string]string{"targetId": "local", "backend": "docker"},
			"config":    bad,
		}))
	})
}

// Create fills the slot too. It has to: validateGatewayConfig does NOT reject a
// ${...} — url.Parse is happy with braces in a path — so a gateway created with
// an unfilled slot would store one, and then EVERY later edit would 400 on the
// update path's refusal, over an upstream the operator never touched and cannot
// reach from the UI. A config the app will not let you save again is worse than
// a refusal at creation.
func TestGatewayCreate_FillsAProviderSlotFromTheStoredKey(t *testing.T) {
	a := gatewayServer(t)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	cfg.ProviderKeys = map[string]string{"INFURA_API_KEY": "sk_infura_secret"}
	if err := cfg.Save(); err != nil {
		t.Fatalf("save config: %v", err)
	}

	addGateway(t, a, "default", "local", catalog.GatewayConfig{
		Port: 4100,
		Networks: []catalog.GatewayNetwork{{ChainID: 1, Upstreams: []catalog.GatewayUpstream{
			{ID: "public-1-1", Kind: catalog.UpstreamExternal, Endpoint: "https://mainnet.infura.io/v3/${INFURA_API_KEY}"},
		}}},
	})

	stored, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	gw, ok := stored.FindGateway("default")
	if !ok {
		t.Fatal("the gateway was not created")
	}
	got := gw.Config.Networks[0].Upstreams[0].Endpoint
	if got != "https://mainnet.infura.io/v3/sk_infura_secret" {
		t.Fatalf("stored endpoint = %q, want the slot filled in at creation", got)
	}

	// The gateway it created must still be editable. This is the whole point:
	// an unfilled slot stored at creation would make every later save 400.
	res := a.do(t, "PUT", "/api/gateways/default/config", gw.Config)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(res.Body)
		t.Fatalf("re-saving a created gateway: got %d, want 200: %s", res.StatusCode, raw)
	}
}

// The same refusal on the create path, so an unfillable slot never reaches
// storage by either door.
func TestGatewayCreate_RefusesASlotItCannotFill(t *testing.T) {
	a := gatewayServer(t)

	res := a.do(t, "POST", "/api/gateways", map[string]any{
		"id":        "default",
		"placement": map[string]string{"targetId": "local", "backend": "docker"},
		"config": catalog.GatewayConfig{
			Port: 4100,
			Networks: []catalog.GatewayNetwork{{ChainID: 1, Upstreams: []catalog.GatewayUpstream{
				{ID: "public-1-1", Kind: catalog.UpstreamExternal, Endpoint: "https://mainnet.infura.io/v3/${INFURA_API_KEY}"},
			}}},
		},
	})
	if res.StatusCode != http.StatusBadRequest {
		defer res.Body.Close()
		raw, _ := io.ReadAll(res.Body)
		t.Fatalf("got %d, want 400: %s", res.StatusCode, raw)
	}
	body := decode[errorDetail](t, res)
	if !strings.Contains(body.Error, "INFURA_API_KEY") {
		t.Errorf("the refusal must name the key to go get: %q", body.Error)
	}

	stored, _ := config.Load()
	if len(stored.Gateways) != 0 {
		t.Errorf("a gateway that could never be saved again was created anyway: %+v", stored.Gateways)
	}
}

// A slot with no key is REFUSED rather than stored. A URL with a literal ${...}
// in it looks configured and answers nothing, which is the same failure
// chainlist.Resolve exists to prevent — and the refusal names the key to go get.
func TestGatewayPutConfig_RefusesASlotItCannotFill(t *testing.T) {
	a := gatewayServer(t)
	addGateway(t, a, "default", "local", catalog.GatewayConfig{Port: 4100})

	res := a.do(t, "PUT", "/api/gateways/default/config", catalog.GatewayConfig{
		Port: 4100,
		Networks: []catalog.GatewayNetwork{{ChainID: 1, Upstreams: []catalog.GatewayUpstream{
			{ID: "public-1-1", Kind: catalog.UpstreamExternal, Endpoint: "https://mainnet.infura.io/v3/${INFURA_API_KEY}"},
		}}},
	})
	if res.StatusCode != http.StatusBadRequest {
		defer res.Body.Close()
		raw, _ := io.ReadAll(res.Body)
		t.Fatalf("got %d, want 400: %s", res.StatusCode, raw)
	}
	body := decode[errorDetail](t, res)
	if !strings.Contains(body.Error, "INFURA_API_KEY") {
		t.Errorf("the refusal must name the key to go get: %q", body.Error)
	}

	stored, _ := config.Load()
	gw, _ := stored.FindGateway("default")
	if len(gw.Config.Networks) != 0 {
		t.Errorf("the unusable upstream was stored anyway: %+v", gw.Config.Networks)
	}
}

// The redundancy bar's denominator rides on the network view, so it has to be
// the count "Add valve's set" actually adds for THAT chain — and zero, not
// four, for a chain valve has never measured. A constant denominator makes the
// page's own primary action overshoot its target on every chain it knows.
func TestGatewayListCarriesThePerChainKnownSetSize(t *testing.T) {
	a := gatewayServer(t)
	addGateway(t, a, "default", "local", catalog.GatewayConfig{
		Port: 4100,
		Networks: []catalog.GatewayNetwork{
			{ChainID: 1, Upstreams: []catalog.GatewayUpstream{{Endpoint: "https://eth.drpc.org"}}},
			{ChainID: 369, Upstreams: []catalog.GatewayUpstream{{Endpoint: "https://rpc.pulsechain.com"}}},
			{ChainID: 31337, Upstreams: []catalog.GatewayUpstream{{Endpoint: "http://127.0.0.1:8545"}}},
		},
	})

	body := decode[gatewaysResponse](t, a.do(t, "GET", "/api/gateways", nil))
	if len(body.Gateways) != 1 {
		t.Fatalf("want one gateway, got %d", len(body.Gateways))
	}
	for _, n := range body.Gateways[0].Networks {
		want := catalog.KnownSetSize(n.ChainID)
		if n.KnownSetSize != want {
			t.Errorf("evm:%d knownSetSize = %d, want %d", n.ChainID, n.KnownSetSize, want)
		}
		if n.ChainID == 31337 && n.KnownSetSize != 0 {
			t.Errorf("a chain with no measured set must report no target, got %d", n.KnownSetSize)
		}
	}
}

func TestKnownSetUnknownGatewayIs404(t *testing.T) {
	a := gatewayServer(t)
	res := a.do(t, "GET", "/api/gateways/nope/knownset/1", nil)
	body := decode[errorDetail](t, res)
	if res.StatusCode != http.StatusNotFound || body.Code != codeGatewayNotFound {
		t.Fatalf("got %d/%q, want 404/%s", res.StatusCode, body.Code, codeGatewayNotFound)
	}
}
