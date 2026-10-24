package server

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/catalog"
	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/executor"
	"github.com/valve-tech/jumpgate/internal/intent"
	"github.com/valve-tech/jumpgate/internal/jobs/jobstest"
	"github.com/valve-tech/jumpgate/internal/monitor"
)

// The catalog registers no real module until each module's own task lands,
// so these tests stand in a registry shaped like the planned one.
func fakeModules(t *testing.T) {
	t.Helper()
	box := func(id string, proto string, def int, chains ...int) catalog.Module {
		return catalog.Module{ID: id, Name: id, Placement: catalog.PlaceBox, Chains: chains,
			Unit: func(b catalog.BoxModule) (catalog.BoxUnit, error) {
				p := b.Port
				if p == 0 {
					p = def
				}
				return catalog.BoxUnit{Unit: "[Unit]\nDescription=" + b.Instance() + "\n",
					Ports: []catalog.PortUse{{Port: p, Proto: proto, Public: true}}}, nil
			}}
	}
	restore := catalog.SwapModulesForTest([]catalog.Module{
		{
			ID: "msgboard", Name: "Message board", Summary: "Posts gossiped between nodes", Placement: catalog.PlaceNode,
			Chains: []int{369, 943}, BetaChains: []int{943}, Clients: []string{"reth"},
			Namespaces: []string{"msgboard"},
			Methods: []catalog.ModuleMethod{
				{Name: "msgboard_addMessage", Class: catalog.MethodWrite},
				{Name: "msgboard_status", Class: catalog.MethodRead},
			},
			ClientReason: func(client string) string {
				return "The message board needs reth (valve fork). This node runs " + client + "."
			},
			DocsStep: "msgboard-enable",
		},
		box("el-bootnode", "udp", 30301, 369, 943, 1),
		box("cl-bootnode", "udp", 9200, 369, 943, 1),
		{
			ID: "private-send", Name: "Private send", Placement: catalog.PlaceGateway, Chains: []int{1},
			Routes: map[int][]catalog.GatewayRoute{1: {{ID: "flashbots", Endpoint: "https://rpc.flashbots.net", Methods: []string{"eth_sendRawTransaction"}}}},
		},
		box("watch", "tcp", 9090),
	})
	t.Cleanup(restore)
}

// stubProvision records each gateway re-provision instead of running it.
func stubProvision(t *testing.T) *[]string {
	t.Helper()
	var mu sync.Mutex
	got := &[]string{}
	prev := reprovisionGateway
	reprovisionGateway = func(_ *Server, _ config.Config, gw config.Gateway) (string, error) {
		mu.Lock()
		*got = append(*got, gw.ID)
		mu.Unlock()
		return gw.Placement.TargetID, nil
	}
	t.Cleanup(func() { reprovisionGateway = prev })
	return got
}

func resultOK(stdout string) executor.Result { return executor.Result{Stdout: stdout} }

func rethWire(chain int, exec string, mods ...catalog.NodeModule) *catalog.WireConfig {
	return &catalog.WireConfig{ChainID: chain, ExecID: exec, BeaconID: "lighthouse-pulse", Names: catalog.NamesCurrent,
		DataDir: "/var/lib/jumpgate-node/369", JWTPath: "/var/lib/jumpgate-node/369/jwt.hex", Modules: mods}
}

// moduleTarget adds target "box" (local mode) and sets its stored config.
func moduleTarget(t *testing.T, a *apiTestServer, mutate func(c *config.Config)) {
	t.Helper()
	res := a.do(t, "POST", "/api/targets", config.Target{ID: "box", Mode: "local"})
	res.Body.Close()
	if _, err := config.Update(func(c *config.Config) error {
		for i := range c.Targets {
			if c.Targets[i].ID != "box" {
				continue
			}
			c.Targets[i].ID = "box"
		}
		mutate(c)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func targetByID(c *config.Config, id string) *config.Target {
	for i := range c.Targets {
		if c.Targets[i].ID == id {
			return &c.Targets[i]
		}
	}
	return nil
}

func readAPIError(t *testing.T, res *http.Response) api.Error {
	t.Helper()
	return decodeJSON[api.Error](t, res)
}

func storedTarget(t *testing.T, id string) config.Target {
	t.Helper()
	c, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	tg := targetByID(&c, id)
	if tg == nil {
		t.Fatalf("no target %s", id)
	}
	return *tg
}

func waitSetupIdle(t *testing.T, a *apiTestServer, id string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		e := a.srv.reg.get(id)
		e.mu.Lock()
		busy := e.setup != nil && e.setup.isRunning()
		e.mu.Unlock()
		if !busy {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("setup did not finish")
}

func TestSetModuleInapplicable400(t *testing.T) {
	fakeModules(t)
	a := newAPITestServer(t)
	moduleTarget(t, a, func(c *config.Config) { targetByID(c, "box").Wire = rethWire(369, "go-pulse") })

	res := a.do(t, "PUT", "/api/targets/box/modules/msgboard", map[string]any{"enabled": true})
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", res.StatusCode)
	}
	e := readAPIError(t, res)
	if e.Code != api.CodeModuleNotApplicable || e.Message != "The message board needs reth (valve fork). This node runs go-pulse." {
		t.Fatalf("error = %+v", e)
	}
	if e.Hint != "Pick a chain and client the module supports; /api/catalog lists them." {
		t.Fatalf("hint = %q", e.Hint)
	}
	if storedTarget(t, "box").Wire.HasModule("msgboard") {
		t.Fatal("a refused module was stored")
	}
}

func TestSetModuleDuringJob409(t *testing.T) {
	requireAgentPeer(t)
	fakeModules(t)
	b := newBoxRig(t)
	boxWire(t)
	var v api.JobView
	if code := b.do(t, "POST", "/api/jobs/box", api.JobStart{Kind: jobstest.TestKind, Inputs: json.RawMessage(`{"AskAt":"two"}`)}, &v); code != 200 {
		t.Fatalf("start = %d", code)
	}
	waitJob(t, b, v.ID, intent.JobNeedsDecision)

	code, raw := b.req(t, "PUT", "/api/targets/box/modules/msgboard", map[string]any{"enabled": true})
	wantJobActive(t, "module", code, raw, v.ID)
	if b.setupRunning() {
		t.Fatal("a refused module change left a run in flight")
	}
	code, raw = b.req(t, "PUT", "/api/targets/box/modules/el-bootnode", map[string]any{"enabled": true, "chainId": 369})
	wantJobActive(t, "box module", code, raw, v.ID)
}

func TestSetModulePublicOnlyNoUnitWrite(t *testing.T) {
	fakeModules(t)
	ex := &autoSucceedExecutor{}
	a := newAPITestServerWithExecutor(t, func(config.Target) (executor.Executor, error) { return ex, nil })
	provisioned := stubProvision(t)
	moduleTarget(t, a, func(c *config.Config) {
		targetByID(c, "box").Wire = rethWire(369, "reth", catalog.NodeModule{ID: "msgboard"})
		mk := func(id, target string) config.Gateway {
			return config.Gateway{ID: id, Placement: config.GatewayPlacement{TargetID: "host-" + id},
				Config: catalog.GatewayConfig{Networks: []catalog.GatewayNetwork{{ChainID: 369,
					Upstreams: []catalog.GatewayUpstream{{ID: "n", Kind: catalog.UpstreamManagedNode, TargetID: target}}}}}}
		}
		c.Gateways = append(c.Gateways, mk("gw1", "box"), mk("gw2", "box"), mk("gw3", "other"))
	})

	res := a.do(t, "PUT", "/api/targets/box/modules/msgboard", map[string]any{"enabled": true, "public": "read"})
	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		t.Fatalf("status = %d, body %s", res.StatusCode, body)
	}
	res.Body.Close()
	if got := storedTarget(t, "box").Wire.Modules; len(got) != 1 || got[0].Public != "read" {
		t.Fatalf("stored modules = %+v", got)
	}
	ex.mu.Lock()
	calls := len(ex.calls)
	ex.mu.Unlock()
	if calls != 0 || a.srv.reg.get("box").setup != nil {
		t.Fatalf("a public-only change ran setup on the box (%d commands)", calls)
	}
	if strings.Join(*provisioned, ",") != "gw1,gw2" {
		t.Fatalf("provisioned = %v, want gw1,gw2", *provisioned)
	}
}

func TestSetModuleDisableRemovesFlag(t *testing.T) {
	fakeModules(t)
	a := newAPITestServer(t)
	stubProvision(t)
	moduleTarget(t, a, func(c *config.Config) {
		targetByID(c, "box").Wire = rethWire(369, "reth", catalog.NodeModule{ID: "msgboard"})
	})
	res := a.do(t, "PUT", "/api/targets/box/modules/msgboard", map[string]any{"enabled": false})
	if res.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(res.Body)
		t.Fatalf("status = %d, body %s", res.StatusCode, body)
	}
	res.Body.Close()
	waitSetupIdle(t, a, "box")
	w := storedTarget(t, "box").Wire
	if w.HasModule("msgboard") {
		t.Fatal("the stored wire still has the module")
	}
	exec, _, err := catalog.RenderUnits(*w)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(exec, "--http.api") {
		t.Fatalf("the exec unit still sets --http.api:\n%s", exec)
	}
}

func TestSetModuleEnableRendersNamespace(t *testing.T) {
	fakeModules(t)
	a := newAPITestServer(t)
	stubProvision(t)
	moduleTarget(t, a, func(c *config.Config) { targetByID(c, "box").Wire = rethWire(369, "reth") })
	res := a.do(t, "PUT", "/api/targets/box/modules/msgboard", map[string]any{"enabled": true})
	if res.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(res.Body)
		t.Fatalf("status = %d, body %s", res.StatusCode, body)
	}
	res.Body.Close()
	waitSetupIdle(t, a, "box")
	if !storedTarget(t, "box").Wire.HasModule("msgboard") {
		t.Fatal("the module was not stored")
	}
}

func TestSetBoxModuleValidates(t *testing.T) {
	fakeModules(t)
	a := newAPITestServer(t)
	moduleTarget(t, a, func(c *config.Config) {
		targetByID(c, "box").Modules = []catalog.BoxModule{{ID: "cl-bootnode", ChainID: 369, Port: 30301}}
	})
	res := a.do(t, "PUT", "/api/targets/box/modules/el-bootnode", map[string]any{"enabled": true, "chainId": 369})
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", res.StatusCode)
	}
	if e := readAPIError(t, res); e.Message != "cl-bootnode-369 and el-bootnode-369 both want udp 30301" {
		t.Fatalf("message = %q", e.Message)
	}
	if got := storedTarget(t, "box").Modules; len(got) != 1 {
		t.Fatalf("stored modules = %+v", got)
	}

	// A field that would reach a unit or a shell is checked first.
	res = a.do(t, "PUT", "/api/targets/box/modules/el-bootnode", map[string]any{"enabled": true, "chainId": 369, "extIp": "1.2.3.4; rm -rf /"})
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("bad extIp status = %d, want 400", res.StatusCode)
	}
	res.Body.Close()

	// A box module on a chain it does not run on is not applicable.
	res = a.do(t, "PUT", "/api/targets/box/modules/el-bootnode", map[string]any{"enabled": true, "chainId": 560048})
	if e := readAPIError(t, res); res.StatusCode != http.StatusBadRequest || e.Code != api.CodeModuleNotApplicable {
		t.Fatalf("other chain: %d %+v", res.StatusCode, e)
	}

	// A chain-agnostic module takes no chainId: it would only change the
	// instance name.
	res = a.do(t, "PUT", "/api/targets/box/modules/watch", map[string]any{"enabled": true, "chainId": 369})
	if e := readAPIError(t, res); res.StatusCode != http.StatusBadRequest || e.Message != "watch does not take a chain" {
		t.Fatalf("chain on chain-agnostic: %d %+v", res.StatusCode, e)
	}
	if got := storedTarget(t, "box").Modules; len(got) != 1 {
		t.Fatalf("stored modules = %+v", got)
	}
}

func TestSetBoxModuleOnTargetWithoutWire(t *testing.T) {
	fakeModules(t)
	a := newAPITestServer(t)
	moduleTarget(t, a, func(c *config.Config) {})
	res := a.do(t, "PUT", "/api/targets/box/modules/el-bootnode", map[string]any{"enabled": true, "chainId": 369, "extIp": "203.0.113.7"})
	if res.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(res.Body)
		t.Fatalf("status = %d, body %s", res.StatusCode, body)
	}
	res.Body.Close()
	if a.srv.reg.get("box").setup == nil {
		t.Fatal("no setup run started")
	}
	waitSetupIdle(t, a, "box")
	got := storedTarget(t, "box")
	if got.Wire != nil || len(got.Modules) != 1 || !reflect.DeepEqual(got.Modules[0], catalog.BoxModule{ID: "el-bootnode", ChainID: 369, ExtIP: "203.0.113.7"}) {
		t.Fatalf("stored = wire %v, modules %+v", got.Wire, got.Modules)
	}

	// Turning it off drops the record.
	res = a.do(t, "PUT", "/api/targets/box/modules/el-bootnode", map[string]any{"enabled": false, "chainId": 369})
	if res.StatusCode != http.StatusAccepted {
		t.Fatalf("disable status = %d", res.StatusCode)
	}
	res.Body.Close()
	waitSetupIdle(t, a, "box")
	if got := storedTarget(t, "box").Modules; len(got) != 0 {
		t.Fatalf("modules after disable = %+v", got)
	}
}

func TestListModulesGateStale(t *testing.T) {
	fakeModules(t)
	a := newAPITestServer(t)
	moduleTarget(t, a, func(c *config.Config) {
		targetByID(c, "box").Agent = &config.AgentPairing{Address: "0x1", Transport: "local"}
	})
	prev := readGateModules
	readGateModules = func(*Server, context.Context, config.Config, config.Target) ([]monitor.ModuleStatus, error) {
		return nil, &agentRejected{intent.Rejection{Code: intent.ReasonUnknownKind, Message: "unknown kind"}}
	}
	t.Cleanup(func() { readGateModules = prev })

	res := a.do(t, "GET", "/api/targets/box/modules", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", res.StatusCode)
	}
	got := decodeJSON[moduleListResponse](t, res)
	if !got.GateStale || got.Modules == nil || len(got.Modules) != 0 || got.Message != "Status needs a newer gate on this box" {
		t.Fatalf("response = %+v", got)
	}

	// Any other refusal is an error, not a stale gate.
	readGateModules = func(*Server, context.Context, config.Config, config.Target) ([]monitor.ModuleStatus, error) {
		return nil, errors.New("boom")
	}
	res = a.do(t, "GET", "/api/targets/box/modules", nil)
	if res.StatusCode == http.StatusOK {
		t.Fatal("a failed read answered 200")
	}
	res.Body.Close()
}

func TestListModulesOverSSH(t *testing.T) {
	fakeModules(t)
	ex := (&scriptedExecutor{}).
		script("ls "+catalog.BoxModulesConfigRoot, resultOK(catalog.BoxModule{ID: "el-bootnode", ChainID: 369}.SidecarPath()+"\n")).
		script("cat ", resultOK(`{"id":"el-bootnode","chainId":369,"unit":"jumpgate-mod-el-bootnode-369.service"}`)).
		script("systemctl is-active", resultOK("active\n"))
	a := newAPITestServerWithExecutor(t, func(config.Target) (executor.Executor, error) { return ex, nil })
	moduleTarget(t, a, func(c *config.Config) {})
	res := a.do(t, "GET", "/api/targets/box/modules", nil)
	got := decodeJSON[moduleListResponse](t, res)
	if got.GateStale || len(got.Modules) != 1 || got.Modules[0].Instance != "el-bootnode-369" || !got.Modules[0].Active {
		t.Fatalf("response = %+v", got)
	}
}

func TestSetGatewayModule(t *testing.T) {
	fakeModules(t)
	a := newAPITestServer(t)
	provisioned := stubProvision(t)
	moduleTarget(t, a, func(c *config.Config) {
		ext := func(chain int) catalog.GatewayNetwork {
			return catalog.GatewayNetwork{ChainID: chain, Upstreams: []catalog.GatewayUpstream{{ID: "x", Endpoint: "https://example.org"}}}
		}
		c.Gateways = append(c.Gateways,
			config.Gateway{ID: "eth", Placement: config.GatewayPlacement{TargetID: "box"}, Config: catalog.GatewayConfig{Networks: []catalog.GatewayNetwork{ext(1)}}},
			config.Gateway{ID: "pls", Placement: config.GatewayPlacement{TargetID: "other"}, Config: catalog.GatewayConfig{Networks: []catalog.GatewayNetwork{ext(369)}}})
	})

	res := a.do(t, "PUT", "/api/gateways/eth/modules/private-send", map[string]any{"enabled": true})
	if res.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(res.Body)
		t.Fatalf("status = %d, body %s", res.StatusCode, body)
	}
	res.Body.Close()
	c, _ := config.Load()
	gw, _ := c.FindGateway("eth")
	if len(gw.Config.Modules) != 1 || gw.Config.Modules[0].ID != "private-send" {
		t.Fatalf("stored modules = %+v", gw.Config.Modules)
	}
	if strings.Join(*provisioned, ",") != "eth" {
		t.Fatalf("provisioned = %v", *provisioned)
	}

	res = a.do(t, "PUT", "/api/gateways/pls/modules/private-send", map[string]any{"enabled": true})
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("369-only status = %d, want 400", res.StatusCode)
	}
	if e := readAPIError(t, res); e.Message != "private-send applies to none of this gateway's chains" || e.Code != api.CodeModuleNotApplicable {
		t.Fatalf("error = %+v", e)
	}

	res = a.do(t, "PUT", "/api/gateways/eth/modules/private-send", map[string]any{"enabled": false})
	res.Body.Close()
	c, _ = config.Load()
	gw, _ = c.FindGateway("eth")
	if len(gw.Config.Modules) != 0 {
		t.Fatalf("modules after disable = %+v", gw.Config.Modules)
	}
}

func TestCatalogCarriesModules(t *testing.T) {
	fakeModules(t)
	a := newAPITestServer(t)
	res := a.do(t, "GET", "/api/catalog", nil)
	got := decodeJSON[struct {
		Modules []catalogModule `json:"modules"`
	}](t, res)
	ids := []string{}
	for _, m := range got.Modules {
		ids = append(ids, m.ID)
	}
	if strings.Join(ids, ",") != "msgboard,el-bootnode,cl-bootnode,watch,private-send" {
		t.Fatalf("ids = %v", ids)
	}
	mb := got.Modules[0]
	if mb.Placement != "node" || mb.DocsStep != "msgboard-enable" || len(mb.Methods) != 2 || mb.Methods[0].Class != "write" {
		t.Fatalf("msgboard = %+v", mb)
	}
	var sawReason, sawBeta, sawOK bool
	for _, ap := range mb.Applies {
		switch {
		case ap.ChainID == 369 && ap.Client == "go-pulse":
			sawReason = !ap.OK && ap.Reason == "The message board needs reth (valve fork). This node runs go-pulse."
		case ap.ChainID == 943 && ap.Client == "reth":
			sawBeta = ap.OK && ap.Beta
		case ap.ChainID == 369 && ap.Client == "reth":
			sawOK = ap.OK && !ap.Beta
		}
	}
	if !sawReason || !sawBeta || !sawOK {
		t.Fatalf("msgboard applies = %+v", mb.Applies)
	}
	for _, ap := range got.Modules[4].Applies {
		if ap.Client != "" || ap.OK != (ap.ChainID == 1) {
			t.Fatalf("private-send applies row %+v", ap)
		}
	}
}

// TestCatalogPrivateSendReasons: the real entry reaches /api/catalog with
// chain 1 usable and every other chain disabled with its reason, verbatim.
func TestCatalogPrivateSendReasons(t *testing.T) {
	a := newAPITestServer(t)
	res := a.do(t, "GET", "/api/catalog", nil)
	got := decodeJSON[struct {
		Modules []catalogModule `json:"modules"`
	}](t, res)
	want := map[int]string{
		1:      "",
		369:    "No private send route exists on PulseChain.",
		943:    "No private send route exists on PulseChain.",
		560048: "No private send route exists on Hoodi.",
	}
	for _, m := range got.Modules {
		if m.ID != "private-send" {
			continue
		}
		if m.Placement != "gateway" || m.DocsStep != "private-send" {
			t.Fatalf("private-send = %+v", m)
		}
		for _, ap := range m.Applies {
			r, ok := want[ap.ChainID]
			if !ok || ap.OK != (r == "") || ap.Reason != r {
				t.Errorf("applies row %+v", ap)
			}
			delete(want, ap.ChainID)
		}
		if len(want) != 0 {
			t.Errorf("no rows for %v", want)
		}
		return
	}
	t.Fatal("private-send is not in the catalog")
}

func TestSetGatewayModuleRefusesOtherPlacements(t *testing.T) {
	fakeModules(t)
	a := newAPITestServer(t)
	provisioned := stubProvision(t)
	moduleTarget(t, a, func(c *config.Config) {
		c.Gateways = append(c.Gateways, config.Gateway{ID: "eth", Placement: config.GatewayPlacement{TargetID: "box"},
			Config: catalog.GatewayConfig{Networks: []catalog.GatewayNetwork{{ChainID: 369,
				Upstreams: []catalog.GatewayUpstream{{ID: "x", Endpoint: "https://example.org"}}}}}})
	})
	for _, id := range []string{"msgboard", "el-bootnode"} {
		res := a.do(t, "PUT", "/api/gateways/eth/modules/"+id, map[string]any{"enabled": true})
		if e := readAPIError(t, res); res.StatusCode != http.StatusBadRequest || e.Code != api.CodeModuleNotApplicable {
			t.Fatalf("%s: %d %+v", id, res.StatusCode, e)
		}
	}
	c, _ := config.Load()
	if gw, _ := c.FindGateway("eth"); len(gw.Config.Modules) != 0 || len(*provisioned) != 0 {
		t.Fatalf("stored %+v, provisioned %v", gw.Config.Modules, *provisioned)
	}
}

// publicOnlyTarget is a box with msgboard on and gateways gw1, gw2 in front.
func publicOnlyTarget(t *testing.T, a *apiTestServer) {
	t.Helper()
	moduleTarget(t, a, func(c *config.Config) {
		targetByID(c, "box").Wire = rethWire(369, "reth", catalog.NodeModule{ID: "msgboard"})
		for _, id := range []string{"gw1", "gw2"} {
			c.Gateways = append(c.Gateways, config.Gateway{ID: id, Placement: config.GatewayPlacement{TargetID: "host-" + id},
				Config: catalog.GatewayConfig{Networks: []catalog.GatewayNetwork{{ChainID: 369,
					Upstreams: []catalog.GatewayUpstream{{ID: "n", Kind: catalog.UpstreamManagedNode, TargetID: "box"}}}}}})
		}
	})
}

func TestSetModulePublicOnlyReportsFailedGateway(t *testing.T) {
	fakeModules(t)
	a := newAPITestServer(t)
	publicOnlyTarget(t, a)
	prev := reprovisionGateway
	reprovisionGateway = func(_ *Server, _ config.Config, gw config.Gateway) (string, error) {
		if gw.ID == "gw2" {
			return "", &provisionError{status: http.StatusConflict, code: api.CodeConflict, msg: "setup is already running for this target"}
		}
		return gw.Placement.TargetID, nil
	}
	t.Cleanup(func() { reprovisionGateway = prev })

	res := a.do(t, "PUT", "/api/targets/box/modules/msgboard", map[string]any{"enabled": true, "public": "read"})
	if res.StatusCode != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502", res.StatusCode)
	}
	got := decodeJSON[moduleChangeResponse](t, res)
	if strings.Join(got.Gateways, ",") != "gw1" || len(got.Failed) != 1 || got.Failed[0].ID != "gw2" ||
		!strings.Contains(got.Failed[0].Error, "already running") {
		t.Fatalf("response = %+v", got)
	}
	if storedTarget(t, "box").Wire.Modules[0].Public != "read" {
		t.Fatal("the change was not saved")
	}
}

// Two gateways placed on one host cannot both stand: config keeps one
// managed eRPC per device (mergeGatewaysPerTarget), so the re-provision fan-out
// never claims one host's slot twice. This pins that invariant, which the
// one-claim-per-gateway loop in reprovisionGatewaysFor relies on, through
// the real provision path (no stub).
func TestSetModulePublicOnlyTwoGatewaysOneHost(t *testing.T) {
	fakeModules(t)
	a := newAPITestServer(t)
	moduleTarget(t, a, func(c *config.Config) {
		targetByID(c, "box").Wire = rethWire(369, "reth", catalog.NodeModule{ID: "msgboard"})
		for _, id := range []string{"gw1", "gw2"} {
			c.Gateways = append(c.Gateways, config.Gateway{ID: id, Placement: config.GatewayPlacement{TargetID: "box", Backend: "docker"},
				Config: catalog.GatewayConfig{Networks: []catalog.GatewayNetwork{{ChainID: 369,
					Upstreams: []catalog.GatewayUpstream{{ID: "n" + id, Kind: catalog.UpstreamManagedNode, TargetID: "box"}}}}}})
		}
	})
	c, _ := config.Load()
	if n := len(c.GatewaysOn("box")); n != 1 {
		t.Fatalf("gateways on one host after load = %d, want 1 (merged)", n)
	}
	res := a.do(t, "PUT", "/api/targets/box/modules/msgboard", map[string]any{"enabled": true, "public": "read"})
	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		t.Fatalf("status = %d, body %s", res.StatusCode, body)
	}
	got := decodeJSON[moduleChangeResponse](t, res)
	if len(got.Gateways) != 1 || len(got.Failed) != 0 {
		t.Fatalf("response = %+v", got)
	}
	waitSetupIdle(t, a, "box")
}

func TestSetModulePublicOnlyTakesTheSlot(t *testing.T) {
	fakeModules(t)
	a := newAPITestServer(t)
	provisioned := stubProvision(t)
	publicOnlyTarget(t, a)
	claimed, busy := a.srv.tryClaimSetupRun("box")
	if busy != "" {
		t.Fatal(busy)
	}
	res := a.do(t, "PUT", "/api/targets/box/modules/msgboard", map[string]any{"enabled": true, "public": "read"})
	if res.StatusCode != http.StatusConflict {
		t.Fatalf("status = %d, want 409", res.StatusCode)
	}
	res.Body.Close()
	a.srv.releaseSetupRun("box", claimed)
	if storedTarget(t, "box").Wire.Modules[0].Public != "" || len(*provisioned) != 0 {
		t.Fatal("a refused change was saved or provisioned")
	}
	// Once free, the slot is not left held by the change.
	res = a.do(t, "PUT", "/api/targets/box/modules/msgboard", map[string]any{"enabled": true, "public": "read"})
	res.Body.Close()
	if _, busy := a.srv.tryClaimSetupRun("box"); busy != "" {
		t.Fatalf("slot left held: %s", busy)
	}
}

func TestSetModulePublicOnlyDuringJob409(t *testing.T) {
	requireAgentPeer(t)
	fakeModules(t)
	b := newBoxRig(t)
	boxWire(t)
	if _, err := config.Update(func(c *config.Config) error {
		c.Targets[0].Wire.Modules = []catalog.NodeModule{{ID: "msgboard"}}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	var v api.JobView
	if code := b.do(t, "POST", "/api/jobs/box", api.JobStart{Kind: jobstest.TestKind, Inputs: json.RawMessage(`{"AskAt":"two"}`)}, &v); code != 200 {
		t.Fatalf("start = %d", code)
	}
	waitJob(t, b, v.ID, intent.JobNeedsDecision)
	code, raw := b.req(t, "PUT", "/api/targets/box/modules/msgboard", map[string]any{"enabled": true, "public": "read"})
	wantJobActive(t, "public-only", code, raw, v.ID)
	if b.setupRunning() {
		t.Fatal("the refused change left the slot held")
	}
}

// A node module's public options come from its method classes: reads make
// "read" offerable, writes make "post" offerable. Other placements have none.
func TestCatalogModulePublicOptions(t *testing.T) {
	fakeModules(t)
	a := newAPITestServer(t)
	got := decodeJSON[struct {
		Modules []catalogModule `json:"modules"`
	}](t, a.do(t, "GET", "/api/catalog", nil))
	if p := strings.Join(got.Modules[0].Public, ","); p != "read,post" {
		t.Fatalf("msgboard public = %q", p)
	}
	for _, m := range got.Modules[1:] {
		if m.Public == nil || len(m.Public) != 0 {
			t.Fatalf("%s public = %#v, want []", m.ID, m.Public)
		}
	}
}
