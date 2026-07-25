package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/valve-tech/valve-node-app/internal/catalog"
	"github.com/valve-tech/valve-node-app/internal/config"
	"github.com/valve-tech/valve-node-app/internal/executor"
	"github.com/valve-tech/valve-node-app/internal/ops"
)

// ---------------------------------------------------------------------
// fakes
// ---------------------------------------------------------------------

// dockerExecutor scripts a target that has a reachable engine, one RUNNING
// container and nothing else. Scripts are checked in order, so the two
// inspect formats (state and published ports) must be distinguished by their
// own template text — which is exactly how the real commands differ.
func dockerExecutor(state string, ports string) *scriptedExecutor {
	e := &scriptedExecutor{}
	return e.
		script("{{.State.Running}}", executor.Result{Stdout: state}).
		script("NetworkSettings.Ports", executor.Result{Stdout: ports}).
		script("command -v docker", executor.Result{ExitCode: 0}).
		script("docker --version", executor.Result{Stdout: "Docker version 29.5.2, build abc\n"}).
		script("docker info --format", executor.Result{Stdout: "29.5.2|linux|aarch64|colima|Ubuntu 24.04.4 LTS\n"})
}

// noDockerExecutor is a target with no docker CLI at all: `command -v docker`
// exits non-zero, which is what ProbeDocker reads as absence.
func noDockerExecutor() *scriptedExecutor {
	e := &scriptedExecutor{}
	return e.
		script("command -v docker", executor.Result{ExitCode: 127}).
		script("docker", executor.Result{ExitCode: 127, Stderr: "sh: docker: command not found"})
}

// addTarget registers a bare local target, which is all the container routes
// need — unlike the node routes, they deliberately do not require a completed
// setup (a machine can host a devnet and never run a node).
func addTarget(t *testing.T, a *apiTestServer) {
	t.Helper()
	res := a.do(t, "POST", "/api/targets", map[string]any{"id": "local", "mode": "local"})
	defer res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("POST /api/targets: got %d, want 201", res.StatusCode)
	}
}

func decode[T any](t *testing.T, res *http.Response) T {
	t.Helper()
	defer res.Body.Close()
	var out T
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return out
}

// ---------------------------------------------------------------------
// status + actions
// ---------------------------------------------------------------------

func TestContainers_ListReportsRunningServiceWithItsLivePorts(t *testing.T) {
	a := newAPITestServerWithExecutor(t, func(config.Target) (executor.Executor, error) {
		return dockerExecutor(
			"true|0|ghcr.io/paradigmxyz/reth:latest|sha256:abc\n",
			"8545/tcp=127.0.0.1:8600\n8546/tcp=127.0.0.1:8601\n",
		), nil
	})
	addTarget(t, a)

	body := decode[containersResponse](t, a.do(t, "GET", "/api/targets/local/containers", nil))

	if !body.Docker.Present || !body.Docker.Reachable {
		t.Fatalf("docker reading: got %+v, want present and reachable", body.Docker)
	}
	if len(body.Services) != 2 {
		t.Fatalf("got %d services, want devnet and erpc", len(body.Services))
	}

	devnet := body.Services[0]
	if devnet.ID != svcDevnet || devnet.Status.State != ops.StateRunning {
		t.Fatalf("devnet: got %s/%s, want devnet/running", devnet.ID, devnet.Status.State)
	}
	// The published ports, not the configured ones: a container's mapping is
	// fixed at creation, so this is the only URL a caller can actually dial.
	want := map[string]string{"JSON-RPC": "http://127.0.0.1:8600", "WebSocket": "ws://127.0.0.1:8601"}
	for _, ep := range devnet.Endpoints {
		if want[ep.Label] != ep.URL {
			t.Errorf("endpoint %q: got %q, want %q", ep.Label, ep.URL, want[ep.Label])
		}
	}
	// A running service is never offered "start", and is always offered the
	// three things that can work on it.
	assertActions(t, devnet.Actions, []string{actionStop, actionRestart, actionRecreate, actionWipe})
}

func TestContainers_StoppedServiceIsNotOfferedStopOrRestart(t *testing.T) {
	a := newAPITestServerWithExecutor(t, func(config.Target) (executor.Executor, error) {
		return dockerExecutor("false|137|ghcr.io/paradigmxyz/reth:latest|sha256:abc\n", ""), nil
	})
	addTarget(t, a)

	v := decode[containerView](t, a.do(t, "GET", "/api/targets/local/containers/devnet", nil))
	if v.Status.State != ops.StateStopped {
		t.Fatalf("state: got %q, want %q", v.Status.State, ops.StateStopped)
	}
	assertActions(t, v.Actions, []string{actionStart, actionRecreate, actionWipe})
	// No endpoints for something that is not listening: a URL there is an
	// invitation to a connection refused.
	if len(v.Endpoints) != 0 {
		t.Errorf("stopped service reports endpoints %+v, want none", v.Endpoints)
	}
}

// The regression: a target with no docker engine must not be offered a start
// button that can only fail — and must be told why, once, rather than shown
// an empty card.
func TestContainers_NoDockerOffersNothingAndSaysWhy(t *testing.T) {
	a := newAPITestServerWithExecutor(t, func(config.Target) (executor.Executor, error) {
		return noDockerExecutor(), nil
	})
	addTarget(t, a)

	body := decode[containersResponse](t, a.do(t, "GET", "/api/targets/local/containers", nil))
	if body.Docker.Present {
		t.Fatalf("docker reported present on a target with no docker CLI")
	}
	if body.Docker.Hint == "" {
		t.Errorf("no install hint for an absent docker — the UI has nothing actionable to show")
	}
	for _, v := range body.Services {
		if len(v.Actions) != 0 {
			t.Errorf("%s offers %v with no engine to run them", v.ID, v.Actions)
		}
		if v.Blocked == "" && v.Error == "" {
			t.Errorf("%s offers no actions and gives no reason", v.ID)
		}
	}
}

func TestContainers_UnknownServiceIs404(t *testing.T) {
	a := newAPITestServerWithExecutor(t, func(config.Target) (executor.Executor, error) {
		return dockerExecutor("true|0|img|sha256:abc\n", ""), nil
	})
	addTarget(t, a)

	res := a.do(t, "GET", "/api/targets/local/containers/nope", nil)
	defer res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("got %d, want 404", res.StatusCode)
	}
}

func TestContainers_UnknownActionIs400(t *testing.T) {
	a := newAPITestServerWithExecutor(t, func(config.Target) (executor.Executor, error) {
		return dockerExecutor("true|0|img|sha256:abc\n", ""), nil
	})
	addTarget(t, a)

	res := a.do(t, "POST", "/api/targets/local/containers/devnet/frobnicate", nil)
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("got %d, want 400", res.StatusCode)
	}
}

// ---------------------------------------------------------------------
// wipe
// ---------------------------------------------------------------------

// The gate is handleServiceClear's, for the same reason: a wipe is
// irreversible, so it takes a typed confirmation rather than a bare POST.
func TestContainers_WipeRequiresTypedConfirmation(t *testing.T) {
	a := newAPITestServerWithExecutor(t, func(config.Target) (executor.Executor, error) {
		return dockerExecutor("true|0|img|sha256:abc\n", ""), nil
	})
	addTarget(t, a)

	for _, confirm := range []string{"", "yes", "erpc"} {
		res := a.do(t, "POST", "/api/targets/local/containers/devnet/wipe", map[string]string{"Confirm": confirm})
		if res.StatusCode != http.StatusBadRequest {
			res.Body.Close()
			t.Fatalf("confirm %q: got %d, want 400", confirm, res.StatusCode)
		}
		res.Body.Close()
	}
}

// A wipe's answer has to carry the report, because the cascade — the restart
// of everything fronting the wiped service — is the part an operator cannot
// otherwise see happened.
func TestContainers_WipeReportsWhatItDid(t *testing.T) {
	a := newAPITestServerWithExecutor(t, func(config.Target) (executor.Executor, error) {
		return dockerExecutor("true|0|img|sha256:abc\n", ""), nil
	})
	addTarget(t, a)

	// A gateway serving the devnet's chain is what puts the gateway in the
	// devnet's FrontedBy, and therefore in the cascade.
	putConfig(t, a, svcGateway, catalog.GatewayConfig{
		Port:     4000,
		Networks: []catalog.GatewayNetwork{{ChainID: catalog.DevnetChainID, Upstreams: []catalog.GatewayUpstream{{ID: "devnet", Endpoint: "http://127.0.0.1:8545", Local: true}}}},
	})

	res := decode[wipeResponse](t, a.do(t, "POST", "/api/targets/local/containers/devnet/wipe", map[string]string{"Confirm": "devnet"}))
	if !res.Report.ContainerRemoved {
		t.Errorf("report says no container was removed, but the fake reported one running")
	}
	if len(res.Report.Cascaded) != 1 || res.Report.Cascaded[0] != svcGateway {
		t.Errorf("cascaded: got %v, want [%s] — a wiped chain leaves its gateway advertising a stale head", res.Report.Cascaded, svcGateway)
	}
}

// A devnet with no gateway in front of it must NOT report a cascade: bouncing
// a gateway that serves some other chain would be a restart nobody asked for.
func TestContainers_WipeWithoutAFrontDoesNotCascade(t *testing.T) {
	a := newAPITestServerWithExecutor(t, func(config.Target) (executor.Executor, error) {
		return dockerExecutor("true|0|img|sha256:abc\n", ""), nil
	})
	addTarget(t, a)

	putConfig(t, a, svcGateway, catalog.GatewayConfig{
		Port:     4000,
		Networks: []catalog.GatewayNetwork{{ChainID: 1, Upstreams: []catalog.GatewayUpstream{{ID: "eth", Endpoint: "https://example.invalid"}}}},
	})

	res := decode[wipeResponse](t, a.do(t, "POST", "/api/targets/local/containers/devnet/wipe", map[string]string{"Confirm": "devnet"}))
	if len(res.Report.Cascaded) != 0 {
		t.Errorf("cascaded %v for a gateway that does not serve the devnet's chain", res.Report.Cascaded)
	}
}

// ---------------------------------------------------------------------
// configuration
// ---------------------------------------------------------------------

func putConfig(t *testing.T, a *apiTestServer, svc string, body any) {
	t.Helper()
	res := a.do(t, "PUT", "/api/targets/local/containers/"+svc+"/config", body)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("PUT %s config: got %d, want 200", svc, res.StatusCode)
	}
}

func TestContainers_ConfigRoundTripsAndResolvesDefaults(t *testing.T) {
	a := newAPITestServerWithExecutor(t, func(config.Target) (executor.Executor, error) {
		return dockerExecutor("true|0|img|sha256:abc\n", ""), nil
	})
	addTarget(t, a)

	// Before anything is stored the editor still opens on real values —
	// the defaults that would be used — rather than on blanks.
	got := decode[containerConfigResponse](t, a.do(t, "GET", "/api/targets/local/containers/devnet/config", nil))
	if got.Configured {
		t.Errorf("a fresh target reports a configured devnet")
	}
	if got.Devnet == nil || got.Devnet.BlockTime != catalog.DefaultDevnetBlockTime || got.Devnet.HTTPPort != catalog.DevnetContainerHTTPPort {
		t.Fatalf("unconfigured devnet: got %+v, want resolved defaults", got.Devnet)
	}

	putConfig(t, a, svcDevnet, catalog.DevnetConfig{BlockTime: "5s", HTTPPort: 9545, WSPort: 9546})

	got = decode[containerConfigResponse](t, a.do(t, "GET", "/api/targets/local/containers/devnet/config", nil))
	if !got.Configured || got.Devnet == nil || got.Devnet.BlockTime != "5s" || got.Devnet.HTTPPort != 9545 {
		t.Fatalf("stored devnet: got %+v, want the saved values", got.Devnet)
	}
}

func TestContainers_ConfigRejectsWhatWouldFailLater(t *testing.T) {
	a := newAPITestServerWithExecutor(t, func(config.Target) (executor.Executor, error) {
		return dockerExecutor("true|0|img|sha256:abc\n", ""), nil
	})
	addTarget(t, a)

	for name, tc := range map[string]struct {
		svc  string
		body any
	}{
		"block time that reth cannot parse": {svcDevnet, map[string]any{"BlockTime": "2 seconds"}},
		"one port for both listeners":       {svcDevnet, map[string]any{"HTTPPort": 8545, "WSPort": 8545}},
		"a chain id reth cannot serve":      {svcDevnet, map[string]any{"ChainID": 999}},
		"an upstream with no scheme":        {svcGateway, map[string]any{"Networks": []any{map[string]any{"ChainID": 1337, "Upstreams": []any{map[string]any{"Endpoint": "127.0.0.1:8545"}}}}}},
		"a chain with no upstream":          {svcGateway, map[string]any{"Networks": []any{map[string]any{"ChainID": 1337}}}},
	} {
		t.Run(name, func(t *testing.T) {
			res := a.do(t, "PUT", "/api/targets/local/containers/"+tc.svc+"/config", tc.body)
			defer res.Body.Close()
			if res.StatusCode != http.StatusBadRequest {
				t.Fatalf("got %d, want 400", res.StatusCode)
			}
		})
	}
}

// Provisioning an unconfigured gateway fails at PLAN time, with a sentence
// naming what is missing — not after two streamed steps and a render error.
func TestContainers_ProvisionWithoutAConfigExplainsWhat(t *testing.T) {
	a := newAPITestServerWithExecutor(t, func(config.Target) (executor.Executor, error) {
		return dockerExecutor("true|0|img|sha256:abc\n", ""), nil
	})
	addTarget(t, a)

	res := a.do(t, "POST", "/api/targets/local/containers/erpc/provision", nil)
	body := decode[errorDetail](t, res)
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("got %d, want 400", res.StatusCode)
	}
	if body.Code != codeNotConfigured {
		t.Errorf("code: got %q, want %q", body.Code, codeNotConfigured)
	}
}

// ---------------------------------------------------------------------
// error mapping
// ---------------------------------------------------------------------

// The contract the UI depends on: "not created" is a 409 the operator fixes
// by provisioning, while a missing or unreachable engine is a 502 whose typed
// Hint is written to be shown verbatim.
func TestClassifyOpsError(t *testing.T) {
	for name, tc := range map[string]struct {
		err      error
		status   int
		code     string
		wantHint bool
	}{
		"not created": {
			err:    &ops.ServiceNotCreatedError{ID: "erpc", ContainerName: "x", Action: "start"},
			status: http.StatusConflict, code: codeNotCreated, wantHint: true,
		},
		"docker absent": {
			err:    &ops.DockerAbsentError{Probe: "command -v docker", ExitCode: 127, Hint: "install Docker"},
			status: http.StatusBadGateway, code: codeDockerAbsent, wantHint: true,
		},
		"docker unreachable": {
			err:    &ops.DockerUnreachableError{Probe: "docker inspect", ExitCode: 1, Detail: "cannot connect", Hint: "start the engine"},
			status: http.StatusBadGateway, code: codeDockerUnreachable, wantHint: true,
		},
		"wrapped absent still classifies": {
			err:    errors.New("outer: " + (&ops.DockerAbsentError{Hint: "install Docker"}).Error()),
			status: http.StatusBadGateway, code: "", wantHint: false,
		},
		"anything else": {
			err:    errors.New("boom"),
			status: http.StatusBadGateway, code: "", wantHint: false,
		},
	} {
		t.Run(name, func(t *testing.T) {
			status, hint, code := classifyOpsError(tc.err)
			if status != tc.status {
				t.Errorf("status: got %d, want %d", status, tc.status)
			}
			if code != tc.code {
				t.Errorf("code: got %q, want %q", code, tc.code)
			}
			if (hint != "") != tc.wantHint {
				t.Errorf("hint: got %q, wantHint=%v", hint, tc.wantHint)
			}
		})
	}
}

// A %w-wrapped typed error must still classify — the setup plans wrap ops
// errors with context, and a UI that lost the hint at the first wrap would
// show a generic failure for the one case with a real fix.
func TestClassifyOpsError_UnwrapsThroughContext(t *testing.T) {
	inner := &ops.DockerUnreachableError{Hint: "start the engine and retry"}
	wrapped := errors.Join(errors.New("wipe devnet"), inner)
	status, hint, code := classifyOpsError(wrapped)
	if status != http.StatusBadGateway || code != codeDockerUnreachable || hint != inner.Hint {
		t.Fatalf("got %d/%q/%q, want 502/%q/%q", status, code, hint, codeDockerUnreachable, inner.Hint)
	}
}

func assertActions(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("actions: got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("actions: got %v, want %v", got, want)
		}
	}
}
