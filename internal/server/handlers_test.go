package server

// The remaining routes: explain, add-target, the container config/provision
// pair, and gateway create. What these have in common is that each one either
// sends something outward (a log excerpt to a third-party AI provider) or
// commits something durable (a machine, a config, a container), so the
// interesting assertions are about what is sent and what is refused.

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/valve-tech/valve-node-app/internal/catalog"
	"github.com/valve-tech/valve-node-app/internal/config"
	"github.com/valve-tech/valve-node-app/internal/executor"
)

func setProvider(t *testing.T, a *apiTestServer, provider string) {
	t.Helper()
	res := a.do(t, "PUT", "/api/settings", map[string]any{"aiProvider": provider, "aiKey": "sk-test"})
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("PUT /api/settings: got %d, want 200", res.StatusCode)
	}
}

// ---------------------------------------------------------------------
// POST /api/targets/{id}/explain
// ---------------------------------------------------------------------

// Explain sends log lines to a third-party provider. What gets sent is
// therefore the assertion that matters most: it must be the operator's ERROR
// lines and nothing else, because the excerpt leaves the machine.
func TestExplain_SendsOnlyTheErrorLinesFromTheJournal(t *testing.T) {
	j := newJournalExecutor()
	a := newAPITestServerWithExecutor(t, func(config.Target) (executor.Executor, error) { return j, nil })
	addTarget(t, a)
	completeSetup(t, "local")
	setProvider(t, a, "anthropic")

	// Start the watcher, then produce one error and one ordinary line.
	res := a.do(t, "GET", "/api/targets/local/logs", nil)
	res.Body.Close()
	j.emit(t, errorLine)
	waitForRecent(t, a)

	res = a.do(t, "POST", "/api/targets/local/explain", nil)
	body := decode[explainResponse](t, res)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("got %d, want 200", res.StatusCode)
	}
	if body.Text == "" {
		t.Error("no explanation came back")
	}

	sent := a.fakeAI.lastReq
	if len(sent.Lines) == 0 {
		t.Fatal("nothing was sent to the provider, so the explanation cannot be about this machine")
	}
	for _, l := range sent.Lines {
		if !strings.Contains(l, "ERROR") {
			t.Errorf("a non-error line was sent to a third party: %q", l)
		}
	}
	// The machine's own shape travels with it, or the answer is generic.
	if sent.ExecClient != "reth" {
		t.Errorf("exec client: got %q, want the target's own", sent.ExecClient)
	}
	if sent.ChainName == "" {
		t.Error("no chain name was sent, so the provider cannot know which network this is")
	}
}

// Explicit lines from the caller are sent verbatim and are NOT topped up from
// the journal: the operator picked those lines, and quietly adding others
// would send more than they chose to.
func TestExplain_ExplicitLinesAreSentInsteadOfTheJournal(t *testing.T) {
	j := newJournalExecutor()
	a := newAPITestServerWithExecutor(t, func(config.Target) (executor.Executor, error) { return j, nil })
	addTarget(t, a)
	completeSetup(t, "local")
	setProvider(t, a, "anthropic")

	res := a.do(t, "GET", "/api/targets/local/logs", nil)
	res.Body.Close()
	j.emit(t, errorLine)
	waitForRecent(t, a)

	chosen := []string{"the one line I care about"}
	res = a.do(t, "POST", "/api/targets/local/explain", map[string]any{"lines": chosen})
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("got %d, want 200", res.StatusCode)
	}

	sent := a.fakeAI.lastReq.Lines
	if len(sent) != 1 || sent[0] != chosen[0] {
		t.Fatalf("sent %q, want exactly the caller's own line", sent)
	}
}

// A provider that fails is a 502 with its reason, not a 500: the failure is
// upstream of this app, and the difference tells the operator whether to check
// their key or file a bug.
func TestExplain_AProviderFailureIsReportedAsUpstream(t *testing.T) {
	j := newJournalExecutor()
	a := newAPITestServerWithExecutor(t, func(config.Target) (executor.Executor, error) { return j, nil })
	addTarget(t, a)
	setProvider(t, a, "anthropic")
	a.fakeAI.err = errors.New("401 invalid x-api-key")

	res := a.do(t, "POST", "/api/targets/local/explain", map[string]any{"lines": []string{"boom"}})
	body := decode[errorDetail](t, res)
	if res.StatusCode != http.StatusBadGateway {
		t.Fatalf("got %d, want 502", res.StatusCode)
	}
	if !strings.Contains(body.Error, "invalid x-api-key") {
		t.Errorf("error %q does not carry the provider's own words", body.Error)
	}
}

func TestExplain_RejectsAMalformedBody(t *testing.T) {
	a := newAPITestServer(t)
	addTarget(t, a)
	setProvider(t, a, "anthropic")

	res := a.doRaw(t, "POST", "/api/targets/local/explain", strings.NewReader("{not json"), true)
	res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("got %d, want 400", res.StatusCode)
	}
}

// ---------------------------------------------------------------------
// POST /api/targets
// ---------------------------------------------------------------------

// A Wire is written only by the setup wizard. Accepting one here would let a
// caller declare a machine "set up" without anything having been installed on
// it, and every screen would then believe it.
func TestAddTarget_RefusesAWireSuppliedByTheCaller(t *testing.T) {
	f := newFleet()
	a := newAPITestServerWithExecutor(t, f.factory)

	res := a.do(t, "POST", "/api/targets", map[string]any{
		"id":   "smuggled",
		"mode": "local",
		"wire": map[string]any{"ChainID": 369, "ExecID": "reth", "BeaconID": "lighthouse-pulse"},
	})
	res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("got %d, want 201", res.StatusCode)
	}

	// The route must have dropped it: the logs route is the observable
	// consequence, since it treats a Wire as "setup finished".
	res = a.do(t, "GET", "/api/targets/smuggled/logs", nil)
	res.Body.Close()
	if res.StatusCode != http.StatusConflict {
		t.Fatalf("logs on the new machine: got %d, want 409 — a caller-supplied wire was accepted", res.StatusCode)
	}
}

func TestAddTarget_RejectsWhatCannotBeATarget(t *testing.T) {
	f := newFleet()
	a := newAPITestServerWithExecutor(t, f.factory)

	for name, body := range map[string]any{
		"no id":                 map[string]any{"mode": "local"},
		"unknown mode":          map[string]any{"id": "x", "mode": "telepathy"},
		"no mode":               map[string]any{"id": "x"},
		"ssh with no host":      map[string]any{"id": "x", "mode": "ssh", "ssh": map[string]string{"user": "ops", "keyPath": "/k"}},
		"ssh with no user":      map[string]any{"id": "x", "mode": "ssh", "ssh": map[string]string{"host": "h", "keyPath": "/k"}},
		"ssh with no key":       map[string]any{"id": "x", "mode": "ssh", "ssh": map[string]string{"host": "h", "user": "ops"}},
		"ssh with no ssh block": map[string]any{"id": "x", "mode": "ssh"},
	} {
		t.Run(name, func(t *testing.T) {
			res := a.do(t, "POST", "/api/targets", body)
			res.Body.Close()
			if res.StatusCode != http.StatusBadRequest {
				t.Fatalf("got %d, want 400", res.StatusCode)
			}
		})
	}
}

// Registering the same machine twice is a conflict, not a silent overwrite:
// the second registration would otherwise replace connection details the
// first one is actively using.
func TestAddTarget_RefusesADuplicateID(t *testing.T) {
	f := newFleet()
	a := newAPITestServerWithExecutor(t, f.factory)
	addTarget(t, a)

	res := a.do(t, "POST", "/api/targets", map[string]any{"id": "local", "mode": "local"})
	res.Body.Close()
	if res.StatusCode != http.StatusConflict {
		t.Fatalf("got %d, want 409", res.StatusCode)
	}
}

// A machine that cannot be reached is not registered at all. Storing it would
// leave a card in the UI whose every action fails, for a machine the operator
// may simply have mistyped.
func TestAddTarget_AnUnreachableMachineIsNotStored(t *testing.T) {
	a := newAPITestServerWithExecutor(t, func(config.Target) (executor.Executor, error) {
		return nil, errors.New("dial tcp 10.0.0.1:22: connect: no route to host")
	})

	res := a.do(t, "POST", "/api/targets", map[string]any{
		"id": "unreachable", "mode": "ssh",
		"ssh": map[string]string{"host": "10.0.0.1", "user": "ops", "keyPath": "/k"},
	})
	body := decode[errorDetail](t, res)
	if res.StatusCode != http.StatusBadGateway {
		t.Fatalf("got %d, want 502", res.StatusCode)
	}
	if !strings.Contains(body.Error, "no route to host") {
		t.Errorf("error %q does not say why it could not be reached", body.Error)
	}

	list := decode[[]config.Target](t, a.do(t, "GET", "/api/targets", nil))
	for _, tg := range list {
		if tg.ID == "unreachable" {
			t.Fatal("an unreachable machine was stored anyway")
		}
	}
}

// ---------------------------------------------------------------------
// PUT /api/targets/{id}/containers/{svc}/config
// ---------------------------------------------------------------------

// A devnet config that cannot work must be refused when it is SAVED, not when
// the container fails to start — by then the operator has left the settings
// screen.
func TestContainerConfig_RejectsWhatCouldNotRun(t *testing.T) {
	f := newFleet()
	a := newAPITestServerWithExecutor(t, f.factory)
	addTarget(t, a)

	for name, body := range map[string]any{
		"http port out of range": catalog.DevnetConfig{HTTPPort: 70000, WSPort: 8601},
		"ws port out of range":   catalog.DevnetConfig{HTTPPort: 8600, WSPort: -1},
		"both ports the same":    catalog.DevnetConfig{HTTPPort: 8600, WSPort: 8600},
	} {
		t.Run(name, func(t *testing.T) {
			res := a.do(t, "PUT", "/api/targets/local/containers/devnet/config", body)
			res.Body.Close()
			if res.StatusCode != http.StatusBadRequest {
				t.Fatalf("got %d, want 400", res.StatusCode)
			}
		})
	}
}

func TestContainerConfig_UnknownServiceIs404(t *testing.T) {
	f := newFleet()
	a := newAPITestServerWithExecutor(t, f.factory)
	addTarget(t, a)

	res := a.do(t, "GET", "/api/targets/local/containers/nonesuch/config", nil)
	res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("got %d, want 404", res.StatusCode)
	}
}

// ---------------------------------------------------------------------
// POST /api/targets/{id}/containers/{svc}/provision
// ---------------------------------------------------------------------

// Provisioning claims the machine's single run slot, for the same reason a
// gateway provision does: two plans driving one executor against one box is
// how a container gets created against a config being rewritten underneath it.
func TestContainerProvision_ClaimsTheMachinesRunSlot(t *testing.T) {
	gate := make(chan struct{})
	t.Cleanup(func() { close(gate) })

	gx := newGatedExecutor(gate)
	a := newAPITestServerWithExecutor(t, func(config.Target) (executor.Executor, error) { return gx, nil })
	addTarget(t, a)
	putConfig(t, a, svcDevnet, catalog.DevnetConfig{HTTPPort: 8600, WSPort: 8601})
	addGateway(t, a, "default", "local", pulsechainOnly(4100))

	res := a.do(t, "POST", "/api/targets/local/containers/devnet/provision", nil)
	res.Body.Close()
	if res.StatusCode != http.StatusAccepted {
		t.Fatalf("devnet provision: got %d, want 202", res.StatusCode)
	}
	gx.awaitInFlight(t)

	// A gateway provision on the same machine now has to wait.
	res = a.do(t, "POST", "/api/gateways/default/provision", nil)
	res.Body.Close()
	if res.StatusCode != http.StatusConflict {
		t.Fatalf("gateway provision during a devnet provision: got %d, want 409", res.StatusCode)
	}
}

// A service this app has no plan for is a 400 with the typed not-configured
// code — not a 404. The machine and the route both exist; what does not exist
// is anything to provision, and the code is what lets the UI say so.
func TestContainerProvision_UnknownServiceIsRefusedWithATypedCode(t *testing.T) {
	f := newFleet()
	a := newAPITestServerWithExecutor(t, f.factory)
	addTarget(t, a)

	res := a.do(t, "POST", "/api/targets/local/containers/nonesuch/provision", nil)
	body := decode[errorDetail](t, res)
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("got %d, want 400", res.StatusCode)
	}
	if body.Code != codeNotConfigured {
		t.Errorf("code: got %q, want %q", body.Code, codeNotConfigured)
	}
}

// ---------------------------------------------------------------------
// POST /api/gateways
// ---------------------------------------------------------------------

// A second gateway with the same id would resolve to the same container name,
// and `docker run --name` would refuse the second one — after the config had
// already been written.
//
// The status is 400, not the 409 that POST /api/targets answers for the same
// class of clash. That inconsistency is asserted here rather than quietly
// tolerated: whichever way it is eventually settled, this test names the
// current behaviour so the change is deliberate.
func TestGatewayCreate_RefusesADuplicateID(t *testing.T) {
	f := newFleet()
	a := newAPITestServerWithExecutor(t, f.factory)
	addTarget(t, a)
	addGateway(t, a, "default", "local", pulsechainOnly(4100))

	res := a.do(t, "POST", "/api/gateways", map[string]any{
		"id":        "default",
		"placement": map[string]string{"targetId": "local"},
	})
	body := decode[errorDetail](t, res)
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("got %d, want 400", res.StatusCode)
	}
	if !strings.Contains(body.Error, "already exists") {
		t.Errorf("error %q does not say the id is taken", body.Error)
	}

	// And the first gateway is untouched — a rejected create must not have
	// half-written over it.
	still := decode[gatewayView](t, a.do(t, "GET", "/api/gateways/default", nil))
	if still.Config.Port != 4100 {
		t.Errorf("the existing gateway was modified by the rejected create: port %d", still.Config.Port)
	}
}

// A new gateway is given a hostname under the loopback wildcard domain even
// when the caller sends no TLS block at all, so turning HTTPS on later never
// opens on a blank field.
func TestGatewayCreate_FillsInADefaultTLSHostname(t *testing.T) {
	f := newFleet()
	a := newAPITestServerWithExecutor(t, f.factory)
	addTarget(t, a)

	got := addGateway(t, a, "fresh", "local", pulsechainOnly(4100))
	if got.Config.TLS == nil {
		t.Fatal("a new gateway was stored with no TLS block, so its hostname has nowhere to live")
	}
	if got.Config.TLS.Enabled {
		t.Error("HTTPS was turned on without being asked for")
	}
	if !strings.HasSuffix(got.Config.TLS.Hostname, catalog.DefaultTLSDomain) {
		t.Errorf("hostname %q is not under the loopback wildcard domain", got.Config.TLS.Hostname)
	}
}

// The backend has to be one this app can actually drive. An unknown one would
// be stored and then fail at provision time with a message about a plan rather
// than about the thing the operator typed.
func TestGatewayCreate_RefusesAnUnknownBackend(t *testing.T) {
	f := newFleet()
	a := newAPITestServerWithExecutor(t, f.factory)
	addTarget(t, a)

	res := a.do(t, "POST", "/api/gateways", map[string]any{
		"id":        "odd",
		"placement": map[string]string{"targetId": "local", "backend": "kubernetes"},
	})
	res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("got %d, want 400", res.StatusCode)
	}
}
