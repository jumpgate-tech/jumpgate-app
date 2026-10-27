package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/catalog"
	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/executor"
	"github.com/valve-tech/jumpgate/internal/signer"
	"github.com/valve-tech/jumpgate/internal/testutil"
)

// systemServer starts a system-mode server (over TCP with a token, which the
// tests use for convenience) with a controller key, and calls adjust first.
func systemServer(t *testing.T, adjust func(*Config)) (*httptest.Server, string, *signer.Key) {
	t.Helper()
	testutil.Home(t)
	ctrl, _ := signer.GenerateKey()
	token := NewSessionToken()
	cfg := Config{Token: token, UI: fstest.MapFS{}, Signer: ctrl, System: true, AdminGID: os.Getgid()}
	if adjust != nil {
		adjust(&cfg)
	}
	ts := httptest.NewServer(New(cfg).Handler())
	t.Cleanup(ts.Close)
	return ts, token, ctrl
}

func TestSystemModeRefusesAClientKeyPath(t *testing.T) {
	ts, token, _ := systemServer(t, func(c *Config) {
		c.NewExecutor = func(config.Target) (executor.Executor, error) { t.Fatal("dialled"); return nil, nil }
	})
	res, e := do(t, ts, token, "POST", "/api/targets",
		`{"id":"b","mode":"ssh","ssh":{"Host":"h","User":"root","KeyPath":"/var/lib/jumpgate-ctl/.jumpgate/ssh/jumpgate_ed25519"}}`)
	if res.StatusCode != http.StatusBadRequest || e.Code != api.CodeKeyPathRefused || e.Hint == "" {
		t.Fatalf("got %d %+v, want 400 key_path_refused", res.StatusCode, e)
	}
	if c, _ := config.Load(); len(c.Targets) != 0 {
		t.Fatalf("saved %v", c.Targets)
	}
}

func TestSystemModeRefusesAJumpHopKeyPath(t *testing.T) {
	ts, token, _ := systemServer(t, nil)
	res, e := do(t, ts, token, "POST", "/api/targets",
		`{"id":"b","mode":"ssh","ssh":{"Host":"h","User":"root","jump":{"Host":"j","User":"u","KeyPath":"/etc/shadow"}}}`)
	if res.StatusCode != http.StatusBadRequest || e.Code != api.CodeKeyPathRefused {
		t.Fatalf("got %d %+v, want 400 key_path_refused", res.StatusCode, e)
	}
}

func TestSystemModeAddTargetDoesNotDial(t *testing.T) {
	ts, token, _ := systemServer(t, func(c *Config) {
		c.NewExecutor = func(config.Target) (executor.Executor, error) { t.Fatal("dialled"); return nil, nil }
	})
	res, e := do(t, ts, token, "POST", "/api/targets", `{"id":"b","mode":"ssh","ssh":{"Host":"h","User":"jumpgate"}}`)
	if res.StatusCode/100 != 2 {
		t.Fatalf("got %d %+v, want 2xx", res.StatusCode, e)
	}
	c, _ := config.Load()
	if len(c.Targets) != 1 || c.Targets[0].ID != "b" {
		t.Fatalf("targets %v", c.Targets)
	}
}

func TestSystemModeProbeRefusesAJumpKey(t *testing.T) {
	ts, token, _ := systemServer(t, nil)
	res, e := do(t, ts, token, "POST", "/api/hostkeys/probe",
		`{"ssh":{"Host":"127.0.0.1","Port":1,"User":"root","jump":{"Host":"127.0.0.1","Port":1,"User":"u","KeyPath":"/k"}}}`)
	if res.StatusCode != http.StatusConflict || e.Code != api.CodeSSHJumpUnsupported {
		t.Fatalf("got %d %+v, want 409 ssh_jump_unsupported", res.StatusCode, e)
	}
}

func TestSystemModeSSHExecutorIsRefused(t *testing.T) {
	ts, token, _ := systemServer(t, func(c *Config) {
		c.NewExecutor = func(config.Target) (executor.Executor, error) { t.Fatal("dialled"); return nil, nil }
	})
	if _, err := config.Update(func(c *config.Config) error {
		c.Targets = append(c.Targets, config.Target{ID: "b", Mode: "ssh",
			SSH:  &executor.SSHConfig{Host: "h", User: "root"},
			Wire: &catalog.WireConfig{ChainID: 369, ExecID: "reth", BeaconID: "lighthouse-pulse", DataDir: "/var/lib/jumpgate-node/369"}})
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	res, e := do(t, ts, token, "GET", "/api/targets/b/du", "")
	if e.Code != api.CodeServiceNoLogin || e.Hint == "" {
		t.Fatalf("got %d %+v, want service_no_login", res.StatusCode, e)
	}
}

func getPairingInfo(t *testing.T, ts *httptest.Server, token string) (int, api.PairingInfo, api.Error) {
	t.Helper()
	req, _ := http.NewRequest("GET", ts.URL+"/api/pairing-info", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var info api.PairingInfo
	var e api.Error
	if res.StatusCode == http.StatusOK {
		_ = json.NewDecoder(res.Body).Decode(&info)
	} else {
		_ = json.NewDecoder(res.Body).Decode(&e)
	}
	return res.StatusCode, info, e
}

func recordController(t *testing.T, k *signer.Key) {
	t.Helper()
	if _, err := config.Update(func(c *config.Config) error {
		c.Controller = &config.Controller{Address: k.Address().Hex(), KeyStore: "file"}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestPairingInfoReportsTheServiceUID(t *testing.T) {
	ts, token, ctrl := systemServer(t, nil)
	recordController(t, ctrl)
	status, info, e := getPairingInfo(t, ts, token)
	if status != http.StatusOK {
		t.Fatalf("status %d %+v", status, e)
	}
	if !info.System || info.ServiceUID != os.Getuid() || !strings.HasPrefix(info.TransportKey, "ssh-ed25519 ") || info.Controller != ctrl.Address().Hex() {
		t.Fatalf("info %+v", info)
	}
}

func TestPairingInfoOutsideSystemMode(t *testing.T) {
	testutil.Home(t)
	ctrl, _ := signer.GenerateKey()
	recordController(t, ctrl)
	token := NewSessionToken()
	ts := httptest.NewServer(New(Config{Token: token, UI: fstest.MapFS{}, Signer: ctrl}).Handler())
	defer ts.Close()
	status, info, e := getPairingInfo(t, ts, token)
	if status != http.StatusOK || info.System || info.ServiceUID != -1 {
		t.Fatalf("got %d %+v %+v", status, info, e)
	}
}

func TestPairingInfoWithoutAKey(t *testing.T) {
	testutil.Home(t)
	token := NewSessionToken()
	ts := httptest.NewServer(New(Config{Token: token, UI: fstest.MapFS{}, System: true}).Handler())
	defer ts.Close()
	status, _, e := getPairingInfo(t, ts, token)
	if status != http.StatusServiceUnavailable || e.Code != api.CodeNoControllerKey {
		t.Fatalf("got %d %+v, want 503 no_controller_key", status, e)
	}
}
