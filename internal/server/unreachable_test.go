package server

// What every route does when the machine behind it cannot be reached.
//
// One table rather than a test per handler, because the contract is uniform
// and worth holding uniform: a route that needs the machine answers 502 with
// the reason, never a 200 describing a machine nobody talked to. That failure
// mode is the one this repo keeps meeting — a swallowed executor error and a
// zero-valued view render as a healthy, empty machine.
//
// The routes that DON'T need the machine are tabled too, and for the opposite
// reason: a config read that starts failing because some unrelated box is
// unreachable is its own bug, so the split is asserted in both directions.

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"strings"
	"testing"

	"github.com/valve-tech/valve-node-app/internal/catalog"
	"github.com/valve-tech/valve-node-app/internal/config"
	"github.com/valve-tech/valve-node-app/internal/executor"
	"github.com/valve-tech/valve-node-app/internal/ops"
)

// errDial is what a target that has moved, or gone, looks like at the moment
// its executor is constructed.
var errDial = errors.New("dial tcp 10.0.0.9:22: connect: no route to host")

// errRun is a link that came up and then dropped mid-command.
var errRun = errors.New("ssh: connection lost")

// brokenExecutor accepts construction and then fails every command — the
// harder case, because the target looks reachable right up until it is used.
type brokenExecutor struct{}

func (brokenExecutor) Run(context.Context, string, *executor.RunOpts) (executor.Result, error) {
	return executor.Result{}, errRun
}
func (brokenExecutor) WriteFile(context.Context, string, []byte, fs.FileMode) error { return errRun }
func (brokenExecutor) ReadFile(context.Context, string) ([]byte, error)             { return nil, errRun }
func (brokenExecutor) Close() error                                                 { return nil }

// seedTarget writes a wired target and a one-chain gateway straight into the
// config, rather than going through POST /api/targets.
//
// That matters: POST builds an executor to validate the target, and
// getExecutor then CACHES it per target. A test that created its target
// through the API would be exercising the cached executor from that call, not
// the failing one it thinks it installed.
func seedTarget(t *testing.T, id string) {
	t.Helper()
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	cfg.Targets = append(cfg.Targets, config.Target{
		ID:   id,
		Mode: "local",
		Wire: &catalog.WireConfig{ChainID: 369, ExecID: "reth", BeaconID: "lighthouse-pulse", DataDir: "/mnt/reth"},
	})
	cfg.Gateways = append(cfg.Gateways, config.Gateway{
		ID:        "default",
		Placement: config.GatewayPlacement{TargetID: id, Backend: "docker"},
		Config:    oneChainGateway(),
	})
	if err := cfg.Save(); err != nil {
		t.Fatalf("save config: %v", err)
	}
}

type route struct {
	method string
	path   string
	body   any
}

// machineRoutes cannot answer without reaching the machine.
func machineRoutes() []route {
	return []route{
		{"GET", "/api/targets/local/du", nil},
		{"GET", "/api/targets/local/disk?path=/mnt/reth", nil},
		{"GET", "/api/targets/local/endpoints", nil},
		{"GET", "/api/targets/local/firewall", nil},
		{"GET", "/api/targets/local/diagnostics", nil},
		{"POST", "/api/targets/local/services/exec/restart", nil},
		{"POST", "/api/targets/local/services/exec/clear", map[string]any{"Confirm": "exec"}},
		{"GET", "/api/targets/local/containers", nil},
		{"GET", "/api/targets/local/containers/devnet", nil},
		{"POST", "/api/targets/local/containers/devnet/wipe", map[string]any{"Confirm": "devnet"}},
		{"POST", "/api/targets/local/containers/devnet/reset", map[string]any{"Confirm": "devnet"}},
		{"POST", "/api/targets/local/containers/devnet/restart", nil},
		{"POST", "/api/gateways/default/restart", nil},
		{"POST", "/api/gateways/default/wipe", map[string]any{"Confirm": "default"}},
		{"GET", "/api/gateways/default/traffic", nil},
		{"GET", "/api/gateways/default/analytics", nil},
	}
}

func TestRoutes_AnUndialableTargetIsAGatewayErrorCarryingTheReason(t *testing.T) {
	for _, rt := range machineRoutes() {
		t.Run(rt.method+" "+rt.path, func(t *testing.T) {
			a := newAPITestServerWithExecutor(t, func(config.Target) (executor.Executor, error) {
				return nil, errDial
			})
			seedTarget(t, "local")

			res := a.do(t, rt.method, rt.path, rt.body)
			defer res.Body.Close()
			body, _ := io.ReadAll(res.Body)

			if res.StatusCode == http.StatusOK {
				t.Fatalf("answered 200 for a machine that was never reached: %s", body)
			}
			if res.StatusCode != http.StatusBadGateway {
				t.Errorf("got %d, want %d: %s", res.StatusCode, http.StatusBadGateway, body)
			}
			// Without the reason, the only actionable content is the status
			// code, and "502" does not tell anyone which box to go look at.
			if !strings.Contains(string(body), "no route to host") {
				t.Errorf("the reason did not reach the operator: %s", body)
			}
		})
	}
}

// reportsReachability are the two routes whose JOB is to report whether the
// machine is answering. A failed command there is the ANSWER, not an error —
// "the exec client is unreachable" is exactly what /endpoints exists to say —
// so they are excluded from the table below and asserted separately.
func reportsReachability(path string) bool {
	return strings.Contains(path, "/endpoints") || strings.HasSuffix(path, "/diagnostics")
}

// A target that dials and then fails every command is the same answer. This
// is the case a construction-time check cannot catch.
func TestRoutes_ATargetWhoseCommandsFailNeverReportsSuccess(t *testing.T) {
	for _, rt := range machineRoutes() {
		if reportsReachability(rt.path) {
			continue
		}
		t.Run(rt.method+" "+rt.path, func(t *testing.T) {
			a := newAPITestServerWithExecutor(t, func(config.Target) (executor.Executor, error) {
				return brokenExecutor{}, nil
			})
			seedTarget(t, "local")

			res := a.do(t, rt.method, rt.path, rt.body)
			defer res.Body.Close()
			body, _ := io.ReadAll(res.Body)

			if res.StatusCode == http.StatusOK && !strings.Contains(string(body), "connection lost") {
				t.Fatalf("a machine whose every command failed was described as healthy: %s", body)
			}
		})
	}
}

// The reachability reporters answer 200 and say the machine is not answering,
// which is the useful result: an operator opening the endpoints panel on a
// dead box wants to be told it is dead, not handed a 502 for the panel itself.
//
// They still 502 when the executor cannot be BUILT (covered by the dial table
// above), and the difference is real: no executor means nothing was probed, so
// there is no reading to report.
func TestRoutes_TheReachabilityReportersReportUnreachabilityAsData(t *testing.T) {
	a := newAPITestServerWithExecutor(t, func(config.Target) (executor.Executor, error) {
		return brokenExecutor{}, nil
	})
	seedTarget(t, "local")

	eps := decode[ops.EndpointInfo](t, a.do(t, "GET", "/api/targets/local/endpoints", nil))
	if eps.ExecReachable || eps.BeaconReachable {
		t.Errorf("a machine whose every command failed reported reachable endpoints: %+v", eps)
	}

	res := a.do(t, "GET", "/api/targets/local/diagnostics", nil)
	if res.StatusCode != http.StatusOK {
		res.Body.Close()
		t.Fatalf("got %d, want 200 — a failing ladder is the diagnosis", res.StatusCode)
	}
	diag := decode[DiagReport](t, res)
	if diag.FailedID == "" {
		t.Error("the ladder found nothing wrong on a machine that answers nothing")
	}
}

// The other half of the contract: these answer from stored configuration, so
// an unreachable machine must not break them. A config form that goes blank
// because some box is down is its own outage.
func TestRoutes_ConfigReadsDoNotDependOnTheMachine(t *testing.T) {
	for _, rt := range []route{
		{"GET", "/api/targets", nil},
		{"GET", "/api/targets/local/containers/devnet/config", nil},
		{"GET", "/api/targets/local/diagnostics/latest", nil},
		{"GET", "/api/gateways", nil},
	} {
		t.Run(rt.method+" "+rt.path, func(t *testing.T) {
			a := newAPITestServerWithExecutor(t, func(config.Target) (executor.Executor, error) {
				return nil, errDial
			})
			seedTarget(t, "local")

			res := a.do(t, rt.method, rt.path, rt.body)
			defer res.Body.Close()
			if res.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(res.Body)
				t.Errorf("a stored-configuration read failed because a machine was down: %d %s", res.StatusCode, body)
			}
		})
	}
}

// GET /api/gateways/{gid} is the deliberate middle case: a gateway is a
// config object this app owns, so it is still described when its machine is
// gone — but it must carry the failure, offer no action, and say why.
func TestGatewayGet_AnUnreachablePlacementIsDescribedAsBrokenNotHealthy(t *testing.T) {
	a := newAPITestServerWithExecutor(t, func(config.Target) (executor.Executor, error) {
		return nil, errDial
	})
	seedTarget(t, "local")

	res := a.do(t, "GET", "/api/gateways/default", nil)
	if res.StatusCode != http.StatusOK {
		res.Body.Close()
		t.Fatalf("got %d, want 200 — the gateway is a config object", res.StatusCode)
	}
	v := decode[gatewayView](t, res)

	if v.Error == "" {
		t.Fatal("an unreachable placement was described with no error at all")
	}
	if !strings.Contains(v.Error, "no route to host") {
		t.Errorf("Error = %q, want the reason", v.Error)
	}
	// The state must not read as running. "running" off a machine nobody
	// reached is the precise lie this guards.
	if v.Status.State == ops.StateRunning {
		t.Error("reported a running container on a machine that could not be reached")
	}
	if len(v.Actions) != 0 {
		t.Errorf("offered %v against an unreachable machine", v.Actions)
	}
	if v.Blocked == "" {
		t.Error("no actions and no explanation is a dead panel with no way forward")
	}
}

// ---------------------------------------------------------------------
// the config-shaped refusals
// ---------------------------------------------------------------------

// A config that cannot be rendered is refused where it was typed, rather than
// stored and failed at provision time — by which point the operator has left
// the form.
func TestGatewayPutConfig_RefusesAConfigERPCWouldNotStartOn(t *testing.T) {
	a := newAPITestServerWithExecutor(t, func(config.Target) (executor.Executor, error) {
		return nil, errDial
	})
	seedTarget(t, "local")

	// Metrics and RPC on one port: two listeners, one port. eRPC binds
	// whichever it starts first and fails the second, so the gateway comes up
	// serving either RPC or counters with nothing saying which.
	clash := oneChainGateway()
	clash.MetricsPort = clash.Port

	res := a.do(t, "PUT", "/api/gateways/default/config", clash)
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)

	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("got %d, want %d — this configuration cannot start: %s", res.StatusCode, http.StatusBadRequest, body)
	}
	if !strings.Contains(string(body), "port") {
		t.Errorf("the refusal does not name the problem: %s", body)
	}

	// And the stored config is untouched by the rejected write.
	got := decode[gatewayView](t, a.do(t, "GET", "/api/gateways/default", nil))
	if got.Config.MetricsPort == got.Config.Port {
		t.Error("the rejected configuration was stored anyway")
	}
}

// An upstream naming a machine that is not there is refused at save time,
// because a managed reference with no target cannot be resolved at render.
func TestGatewayPutConfig_RefusesAManagedUpstreamWithNoMachine(t *testing.T) {
	a := newAPITestServerWithExecutor(t, func(config.Target) (executor.Executor, error) {
		return nil, errDial
	})
	seedTarget(t, "local")

	orphan := oneChainGateway()
	orphan.Networks[0].Upstreams = []catalog.GatewayUpstream{
		{ID: "n1", Kind: catalog.UpstreamManagedNode}, // no TargetID
	}

	res := a.do(t, "PUT", "/api/gateways/default/config", orphan)
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)

	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("got %d, want %d: %s", res.StatusCode, http.StatusBadRequest, body)
	}
	if !strings.Contains(string(body), "machine") {
		t.Errorf("the refusal does not say what is missing: %s", body)
	}
}

func TestGatewayPutConfig_RefusesAnUnknownUpstreamKind(t *testing.T) {
	a := newAPITestServerWithExecutor(t, func(config.Target) (executor.Executor, error) {
		return nil, errDial
	})
	seedTarget(t, "local")

	bad := oneChainGateway()
	bad.Networks[0].Upstreams = []catalog.GatewayUpstream{
		{ID: "n1", Kind: "carrier-pigeon", Endpoint: "https://rpc.example.com"},
	}

	res := a.do(t, "PUT", "/api/gateways/default/config", bad)
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)

	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("got %d, want %d: %s", res.StatusCode, http.StatusBadRequest, body)
	}
	if !strings.Contains(string(body), "carrier-pigeon") {
		t.Errorf("the refusal does not name the offending kind: %s", body)
	}
}

// Malformed JSON is a 400, not a decode panic and not a zero-valued config
// written over a working one.
func TestPutConfig_MalformedJSONLeavesTheStoredConfigAlone(t *testing.T) {
	a := newAPITestServerWithExecutor(t, func(config.Target) (executor.Executor, error) {
		return nil, errDial
	})
	seedTarget(t, "local")

	for _, path := range []string{
		"/api/gateways/default/config",
		"/api/targets/local/containers/devnet/config",
	} {
		t.Run(path, func(t *testing.T) {
			res := a.doRaw(t, "PUT", path, strings.NewReader("{not json"), true)
			defer res.Body.Close()
			if res.StatusCode != http.StatusBadRequest {
				body, _ := io.ReadAll(res.Body)
				t.Errorf("got %d, want %d: %s", res.StatusCode, http.StatusBadRequest, body)
			}
		})
	}

	got := decode[gatewayView](t, a.do(t, "GET", "/api/gateways/default", nil))
	if len(got.Config.Networks) != 1 {
		t.Errorf("a rejected write changed the stored config: %+v", got.Config)
	}
}

// A wipe is destructive and irreversible, so the confirmation must MATCH.
// The mismatch is the typo that saves the operator.
func TestWipe_RequiresAMatchingConfirmation(t *testing.T) {
	a := newAPITestServerWithExecutor(t, func(config.Target) (executor.Executor, error) {
		return brokenExecutor{}, nil
	})
	seedTarget(t, "local")

	for _, tc := range []struct {
		name string
		path string
		body any
	}{
		{"gateway, wrong name", "/api/gateways/default/wipe", map[string]any{"Confirm": "not-the-gateway"}},
		{"gateway, no confirmation", "/api/gateways/default/wipe", map[string]any{}},
		{"container, wrong name", "/api/targets/local/containers/devnet/wipe", map[string]any{"Confirm": "erpc"}},
		{"container, no confirmation", "/api/targets/local/containers/devnet/wipe", map[string]any{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			res := a.do(t, "POST", tc.path, tc.body)
			defer res.Body.Close()
			body, _ := io.ReadAll(res.Body)

			if res.StatusCode != http.StatusBadRequest {
				t.Fatalf("a wipe proceeded on a mismatched confirmation: %d %s", res.StatusCode, body)
			}
			if !strings.Contains(strings.ToLower(string(body)), "confirm") {
				t.Errorf("the refusal does not tell the operator what to type: %s", body)
			}
		})
	}
}

// A route naming a target that does not exist is a 404 — not a 500, and not a
// 200 describing nothing.
func TestRoutes_AnUnknownTargetIsNotFound(t *testing.T) {
	a := newAPITestServer(t)

	for _, rt := range machineRoutes() {
		if !strings.Contains(rt.path, "/targets/local/") {
			continue
		}
		path := strings.Replace(rt.path, "/targets/local/", "/targets/ghost/", 1)
		t.Run(rt.method+" "+path, func(t *testing.T) {
			res := a.do(t, rt.method, path, rt.body)
			defer res.Body.Close()
			if res.StatusCode != http.StatusNotFound {
				body, _ := io.ReadAll(res.Body)
				t.Errorf("got %d, want %d: %s", res.StatusCode, http.StatusNotFound, body)
			}
		})
	}
}
