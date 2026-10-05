package server

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"encoding/pem"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"

	"golang.org/x/crypto/ssh"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/catalog"
	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/executor"
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
	dir := filepath.Join(a.home, ".jumpgate")
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
	body := decode[api.Error](t, res)
	for _, want := range []string{"start", "stop", "restart", "provision", "wipe"} {
		if !strings.Contains(body.Message, want) {
			t.Errorf("the message must point at %q: %q", want, body.Message)
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
	body := decode[api.Error](t, res)
	if body.Code != api.CodeDockerAbsent {
		t.Errorf("code: got %q, want %q", body.Code, api.CodeDockerAbsent)
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

// The error contract: every /api error is {error, hint, code} JSON with a
// registered code, whichever route and whichever layer refused.

// contractServer is a server over an isolated HOME holding the given targets.
func contractServer(t *testing.T, targets ...config.Target) (*httptest.Server, string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	if _, err := config.Update(func(c *config.Config) error { c.Targets = append(c.Targets, targets...); return nil }); err != nil {
		t.Fatal(err)
	}
	token := NewSessionToken()
	ts := httptest.NewServer(New(Config{Token: token, UI: fstest.MapFS{}}).Handler())
	t.Cleanup(ts.Close)
	return ts, token
}

func do(t *testing.T, ts *httptest.Server, token, method, path, body string) (*http.Response, api.Error) {
	t.Helper()
	req, _ := http.NewRequest(method, ts.URL+path, strings.NewReader(body))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var e api.Error
	if res.StatusCode >= 300 {
		if ct := res.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
			t.Fatalf("%s %s answered %d with Content-Type %q, want JSON", method, path, res.StatusCode, ct)
		}
		if err := json.NewDecoder(res.Body).Decode(&e); err != nil {
			t.Fatalf("%s %s: error body is not JSON: %v", method, path, err)
		}
	}
	return res, e
}

func TestUnknownTargetIsTargetNotFoundOnEveryRoute(t *testing.T) {
	ts, token := contractServer(t)
	for _, r := range []struct{ method, path string }{
		{"GET", "/api/targets/nope/logs"}, {"GET", "/api/targets/nope/du"}, {"GET", "/api/targets/nope/firewall"},
		{"POST", "/api/targets/nope/explain"},
		{"GET", "/api/targets/nope/monitor/stream"}, {"POST", "/api/targets/nope/services/exec/restart"},
	} {
		res, e := do(t, ts, token, r.method, r.path, "{}")
		if res.StatusCode != http.StatusNotFound || e.Code != api.CodeTargetNotFound || e.Message != "target not found" || e.Hint == "" {
			t.Errorf("%s %s: %d %+v", r.method, r.path, res.StatusCode, e)
		}
	}
}

func TestAuthFailuresAreJSON(t *testing.T) {
	ts, token := contractServer(t)
	res, e := do(t, ts, "", "GET", "/api/targets", "")
	if res.StatusCode != http.StatusUnauthorized || e.Code != api.CodeUnauthorized || e.Message != "unauthorized" {
		t.Fatalf("no token: %d %+v", res.StatusCode, e)
	}
	res, e = do(t, ts, "", "GET", "/api/targets?token=wrong", "")
	if res.StatusCode != http.StatusUnauthorized || e.Code != api.CodeUnauthorized {
		t.Fatalf("wrong ?token=: %d %+v", res.StatusCode, e)
	}

	// A cookie write from another origin is refused with the same body shape.
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/health", nil)
	req.AddCookie(&http.Cookie{Name: cookieName, Value: token})
	req.Header.Set("Origin", "https://evil.example")
	r, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Body.Close()
	var f api.Error
	if err := json.NewDecoder(r.Body).Decode(&f); err != nil || r.StatusCode != http.StatusForbidden || f.Code != api.CodeForbidden || f.Hint == "" {
		t.Fatalf("cross-origin: %d %+v (%v)", r.StatusCode, f, err)
	}
}

// Adding a name already on record answers target_exists: the CLI and the TUI
// re-pair an existing name on exactly this code.
func TestAddingAnExistingTargetIsTargetExists(t *testing.T) {
	ts, token := contractServer(t, config.Target{ID: "bare", Mode: "ssh", SSH: &executor.SSHConfig{Host: "h", User: "u"}})
	res, e := do(t, ts, token, "POST", "/api/targets", `{"id":"bare","mode":"ssh","ssh":{"host":"h","user":"u","keyPath":"/k"}}`)
	if res.StatusCode != http.StatusConflict || e.Code != api.CodeTargetExists {
		t.Fatalf("got %d %+v", res.StatusCode, e)
	}
}

func TestTargetWithoutSetupIsTargetNotSetUp(t *testing.T) {
	ts, token := contractServer(t, config.Target{ID: "bare", Mode: "ssh", SSH: &executor.SSHConfig{Host: "h", User: "u"}})
	res, e := do(t, ts, token, "GET", "/api/targets/bare/firewall", "")
	if res.StatusCode != http.StatusConflict || e.Code != api.CodeTargetNotSetUp {
		t.Fatalf("got %d %+v", res.StatusCode, e)
	}
}

// writeTestKey writes an unencrypted ed25519 private key for dial tests.
func writeTestKey(t *testing.T) string {
	t.Helper()
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	block, err := ssh.MarshalPrivateKey(priv, "")
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(t.TempDir(), "id_ed25519")
	if err := os.WriteFile(p, pem.EncodeToMemory(block), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

// A legacy SSH target whose address refuses connections is "unreachable",
// the same code the agent routes use, not an uncoded 502.
func TestLegacyDialFailureIsUnreachable(t *testing.T) {
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()
	key := writeTestKey(t)
	ts, token := contractServer(t, config.Target{
		ID: "gone", Mode: "ssh",
		SSH:  &executor.SSHConfig{Host: "127.0.0.1", Port: port, User: "root", KeyPath: key},
		Wire: &catalog.WireConfig{ChainID: 1, ExecID: "geth", BeaconID: "lighthouse", DataDir: "/var/lib/jumpgate-node/1"},
	})
	res, e := do(t, ts, token, "GET", "/api/targets/gone/firewall", "")
	if res.StatusCode != http.StatusGatewayTimeout || e.Code != api.CodeUnreachable || !strings.Contains(e.Message, strconv.Itoa(port)) {
		t.Fatalf("got %d %+v", res.StatusCode, e)
	}
}
