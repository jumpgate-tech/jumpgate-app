package server

// The refusals: what the write routes will not accept, and what they say.
//
// These are worth their own file because a refusal that arrives LATE is the
// expensive kind. A target persisted without a reachable host fails on every
// command it ever runs; a gateway id that duplicates an existing one takes
// over its container name. Both are cheap to catch at the moment they are
// typed and awkward to unpick afterwards.

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/valve-tech/valve-node-app/internal/config"
	"github.com/valve-tech/valve-node-app/internal/executor"
)

// ---------------------------------------------------------------------
// POST /api/targets
// ---------------------------------------------------------------------

func TestAddTarget_RefusesWhatItCannotDrive(t *testing.T) {
	tests := []struct {
		name    string
		body    any
		want    int
		wantSay string
	}{
		{
			name:    "no id",
			body:    map[string]any{"mode": "local"},
			want:    http.StatusBadRequest,
			wantSay: "id is required",
		},
		{
			name:    "a mode that does not exist",
			body:    map[string]any{"id": "box", "mode": "carrier-pigeon"},
			want:    http.StatusBadRequest,
			// The quotes around the modes are JSON-escaped in the body, so
			// the match stops before them.
			wantSay: "mode must be",
		},
		{
			name:    "ssh with no ssh block",
			body:    map[string]any{"id": "box", "mode": "ssh"},
			want:    http.StatusBadRequest,
			wantSay: "ssh.host",
		},
		{
			name: "ssh missing a key path",
			body: map[string]any{"id": "box", "mode": "ssh", "ssh": map[string]any{
				"host": "10.0.0.9", "user": "root",
			}},
			want:    http.StatusBadRequest,
			wantSay: "keyPath",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			a := newAPITestServer(t)

			res := a.do(t, "POST", "/api/targets", tc.body)
			defer res.Body.Close()
			body, _ := io.ReadAll(res.Body)

			if res.StatusCode != tc.want {
				t.Fatalf("got %d, want %d: %s", res.StatusCode, tc.want, body)
			}
			if !strings.Contains(string(body), tc.wantSay) {
				t.Errorf("the refusal does not say what is wrong (want %q): %s", tc.wantSay, body)
			}

			// Nothing was persisted, so the operator can correct and retry.
			list := decode[[]config.Target](t, a.do(t, "GET", "/api/targets", nil))
			if len(list) != 0 {
				t.Errorf("a refused target was stored anyway: %+v", list)
			}
		})
	}
}

func TestAddTarget_RefusesMalformedJSON(t *testing.T) {
	a := newAPITestServer(t)

	res := a.doRaw(t, "POST", "/api/targets", strings.NewReader("{not json"), true)
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		body, _ := io.ReadAll(res.Body)
		t.Fatalf("got %d, want %d: %s", res.StatusCode, http.StatusBadRequest, body)
	}
}

// A target is dialled BEFORE it is persisted. Storing one that cannot be
// reached leaves an entry whose every future command fails, with the error
// arriving far from the mistake that caused it.
func TestAddTarget_RefusesAHostItCannotReach(t *testing.T) {
	a := newAPITestServerWithExecutor(t, func(config.Target) (executor.Executor, error) {
		return nil, errDial
	})

	res := a.do(t, "POST", "/api/targets", map[string]any{"id": "box", "mode": "local"})
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)

	if res.StatusCode != http.StatusBadGateway {
		t.Fatalf("got %d, want %d: %s", res.StatusCode, http.StatusBadGateway, body)
	}
	if !strings.Contains(string(body), "no route to host") {
		t.Errorf("the refusal drops the reason: %s", body)
	}

	list := decode[[]config.Target](t, a.do(t, "GET", "/api/targets", nil))
	if len(list) != 0 {
		t.Errorf("an unreachable target was persisted: %+v", list)
	}
}

// ---------------------------------------------------------------------
// DELETE /api/gateways/{gid}
// ---------------------------------------------------------------------

// Deleting a gateway forgets the RECORD and deliberately leaves the container
// running. Saying so is the whole point: an operator who expected a teardown
// otherwise walks away from a container still serving traffic.
func TestGatewayDelete_ForgetsTheRecordAndSaysTheContainerSurvives(t *testing.T) {
	a := newAPITestServerWithExecutor(t, func(config.Target) (executor.Executor, error) {
		return nil, errDial
	})
	seedTarget(t, "local")

	res := a.do(t, "DELETE", "/api/gateways/default", nil)
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)

	if res.StatusCode != http.StatusOK {
		t.Fatalf("got %d, want 200: %s", res.StatusCode, body)
	}
	if !strings.Contains(string(body), "NOT touched") {
		t.Errorf("the response does not warn that the container is still running: %s", body)
	}
	// It names the container and the machine, so the operator can go do it.
	if !strings.Contains(string(body), "valve-node-app-erpc") {
		t.Errorf("the response does not name the surviving container: %s", body)
	}
	if !strings.Contains(string(body), "local") {
		t.Errorf("the response does not name the machine it is on: %s", body)
	}

	if res := a.do(t, "GET", "/api/gateways/default", nil); res.StatusCode != http.StatusNotFound {
		res.Body.Close()
		t.Errorf("the gateway is still on record: got %d", res.StatusCode)
	}
}

func TestGatewayDelete_AnUnknownGatewayIsNotFound(t *testing.T) {
	a := newAPITestServer(t)

	res := a.do(t, "DELETE", "/api/gateways/ghost", nil)
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)

	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("got %d, want %d: %s", res.StatusCode, http.StatusNotFound, body)
	}
	if !strings.Contains(string(body), "ghost") {
		t.Errorf("the error does not name what was not found: %s", body)
	}
}

// The same 404 shape for the config write and the actions, so a stale UI
// pointing at a removed gateway gets one consistent answer.
func TestGatewayRoutes_AnUnknownGatewayIsNotFound(t *testing.T) {
	a := newAPITestServerWithExecutor(t, func(config.Target) (executor.Executor, error) {
		return nil, errDial
	})
	seedTarget(t, "local")

	for _, rt := range []route{
		{"GET", "/api/gateways/ghost", nil},
		{"PUT", "/api/gateways/ghost/config", oneChainGateway()},
		{"POST", "/api/gateways/ghost/restart", nil},
		{"POST", "/api/gateways/ghost/wipe", map[string]any{"Confirm": "ghost"}},
	} {
		t.Run(rt.method+" "+rt.path, func(t *testing.T) {
			res := a.do(t, rt.method, rt.path, rt.body)
			defer res.Body.Close()
			if res.StatusCode != http.StatusNotFound {
				body, _ := io.ReadAll(res.Body)
				t.Errorf("got %d, want %d: %s", res.StatusCode, http.StatusNotFound, body)
			}
		})
	}
}

// ---------------------------------------------------------------------
// POST /api/targets/{id}/setup
// ---------------------------------------------------------------------

// The wizard's own validation: a setup run is refused before anything is
// installed when the pair it names cannot be wired.
func TestStartSetup_RefusesAWireItCannotPlan(t *testing.T) {
	tests := []struct {
		name string
		body any
	}{
		{"an execution client that does not exist", map[string]any{
			"chainId": 369, "execId": "not-a-client", "beaconId": "lighthouse-pulse", "dataDir": "/mnt/reth",
		}},
		{"a beacon client that does not exist", map[string]any{
			"chainId": 369, "execId": "reth", "beaconId": "not-a-client", "dataDir": "/mnt/reth",
		}},
		{"a chain neither client serves", map[string]any{
			"chainId": 999999, "execId": "reth", "beaconId": "lighthouse-pulse", "dataDir": "/mnt/reth",
		}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			a := newAPITestServer(t)
			addTarget(t, a)

			res := a.do(t, "POST", "/api/targets/local/setup", tc.body)
			defer res.Body.Close()
			body, _ := io.ReadAll(res.Body)

			if res.StatusCode < 400 {
				t.Fatalf("a setup run started on a configuration that cannot be planned: %d %s", res.StatusCode, body)
			}
			if len(body) == 0 {
				t.Error("the refusal carries no explanation")
			}
		})
	}
}

func TestStartSetup_RefusesMalformedJSON(t *testing.T) {
	a := newAPITestServer(t)
	addTarget(t, a)

	res := a.doRaw(t, "POST", "/api/targets/local/setup", strings.NewReader("{not json"), true)
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		body, _ := io.ReadAll(res.Body)
		t.Errorf("got %d, want %d: %s", res.StatusCode, http.StatusBadRequest, body)
	}
}

func TestStartSetup_AnUnknownTargetIsNotFound(t *testing.T) {
	a := newAPITestServer(t)

	res := a.do(t, "POST", "/api/targets/ghost/setup", map[string]any{
		"chainId": 369, "execId": "reth", "beaconId": "lighthouse-pulse", "dataDir": "/mnt/reth",
	})
	defer res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(res.Body)
		t.Errorf("got %d, want %d: %s", res.StatusCode, http.StatusNotFound, body)
	}
}
