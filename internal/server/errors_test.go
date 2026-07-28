package server

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/valve-tech/valve-node-app/internal/config"
	"github.com/valve-tech/valve-node-app/internal/executor"
)

// The failure paths. Each of these is a branch an operator only ever meets on
// a bad day, which is exactly why it must not be the branch nobody ran: a
// handler that panics or answers 200 on a corrupt config is a worse day still.

// A config file that will not parse must not take the whole API down with a
// panic or an empty 200. It is a 500 that names the file, because the fix is
// on disk and nothing this app does through the UI will get to it.
func TestUnreadableConfigIs500Everywhere(t *testing.T) {
	a := newAPITestServer(t)

	// Corrupt the config the server reads. The API test server points HOME at
	// a temp dir, so this touches nothing real.
	dir := filepath.Join(a.home, ".valve-node-app")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte("{not json"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	for _, path := range []string{"/api/settings", "/api/targets", "/api/gateways"} {
		res := a.do(t, "GET", path, nil)
		res.Body.Close()
		if res.StatusCode != http.StatusInternalServerError {
			t.Errorf("%s: got %d, want 500 — a config that cannot be read is not an empty config", path, res.StatusCode)
		}
	}

	// And a write must refuse rather than overwrite what it could not read:
	// saving on top of an unparseable file would discard whatever the operator
	// still had in there.
	res := a.do(t, "PUT", "/api/settings", map[string]any{"aiProvider": "groq"})
	res.Body.Close()
	if res.StatusCode != http.StatusInternalServerError {
		t.Errorf("PUT /api/settings over a broken config: got %d, want 500", res.StatusCode)
	}
}

func TestPutSettings_RejectsABodyThatIsNotJSON(t *testing.T) {
	a := newAPITestServer(t)
	res := a.doRaw(t, "PUT", "/api/settings", strings.NewReader("{nope"), true)
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("got %d, want 400", res.StatusCode)
	}
}

// The container action route takes exactly three verbs, and the refusal has to
// say where the other two operations live — "unknown action" alone leaves
// someone guessing which of create/provision/wipe they wanted.
func TestHandleContainerAction_UnknownActionNamesTheRealOnes(t *testing.T) {
	a := newAPITestServerWithExecutor(t, func(config.Target) (executor.Executor, error) {
		return dockerExecutor("true|0|img|sha256:abc\n", ""), nil
	})
	addTarget(t, a)

	res := a.do(t, "POST", "/api/targets/local/containers/devnet/frobnicate", nil)
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("got %d, want 400", res.StatusCode)
	}
	body := decode[errorDetail](t, res)
	for _, want := range []string{"start", "stop", "restart", "provision", "wipe"} {
		if !strings.Contains(body.Error, want) {
			t.Errorf("the message must point at %q: %q", want, body.Error)
		}
	}
}

// An engine that is not installed is a 502 with a typed code and a hint, not a
// bare 500: "docker is not on that machine" is fixable by the operator and the
// UI keys off the code to say how.
func TestHandleContainerAction_DockerAbsentKeepsItsCodeAndHint(t *testing.T) {
	a := newAPITestServerWithExecutor(t, func(config.Target) (executor.Executor, error) {
		return noDockerExecutor(), nil
	})
	addTarget(t, a)

	res := a.do(t, "POST", "/api/targets/local/containers/devnet/start", nil)
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadGateway {
		t.Fatalf("got %d, want 502", res.StatusCode)
	}
	body := decode[errorDetail](t, res)
	if body.Code != codeDockerAbsent {
		t.Errorf("code: got %q, want %q", body.Code, codeDockerAbsent)
	}
	if body.Hint == "" {
		t.Error("an absent engine is fixable, so the response must say how")
	}
}

// A route that names a machine nobody registered is a 404, not a 500 — and
// certainly not a nil-pointer panic on the way to building an executor for it.
func TestUnknownTargetIs404OnTheContainerRoutes(t *testing.T) {
	a := newAPITestServerWithExecutor(t, func(config.Target) (executor.Executor, error) {
		return dockerExecutor("true|0|img|sha256:abc\n", ""), nil
	})
	addTarget(t, a)

	for _, path := range []string{
		"/api/targets/ghost/containers",
		"/api/targets/ghost/containers/devnet/status",
	} {
		res := a.do(t, "GET", path, nil)
		res.Body.Close()
		if res.StatusCode != http.StatusNotFound {
			t.Errorf("%s: got %d, want 404", path, res.StatusCode)
		}
	}
}
