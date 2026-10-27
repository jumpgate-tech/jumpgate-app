package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/catalog"
	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/executor"
	"github.com/valve-tech/jumpgate/internal/peercred"
	"github.com/valve-tech/jumpgate/internal/signer"
	"github.com/valve-tech/jumpgate/internal/testutil"
	"github.com/valve-tech/jumpgate/internal/vault"
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

// peerServer is a system-mode server whose every request arrives as if over
// the admin socket from a peer with uid (and in jumpgate-admin, gid 4242,
// when admin is set). It has no controller key.
func peerServer(t *testing.T, uid int, admin bool, adjust func(*Config)) (*httptest.Server, *Server) {
	t.Helper()
	testutil.Home(t)
	cfg := Config{Token: NewSessionToken(), UI: fstest.MapFS{}, System: true, AdminGID: 4242}
	if adjust != nil {
		adjust(&cfg)
	}
	s := New(cfg)
	cred := peercred.Cred{UID: uid}
	if admin {
		cred.GIDs = []int{4242}
	}
	h := s.Handler()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), peerCredKey{}, cred)))
	}))
	t.Cleanup(ts.Close)
	return ts, s
}

// sealPost is POST /api/system/seal with pass, and the KEK it answered.
func sealPost(t *testing.T, ts *httptest.Server, pass string) (int, api.Error, []byte) {
	t.Helper()
	body, _ := json.Marshal(api.SystemSeal{Passphrase: []byte(pass)})
	res, err := http.Post(ts.URL+"/api/system/seal", "application/json", strings.NewReader(string(body)))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var e api.Error
	var ok api.SystemSealed
	if res.StatusCode == http.StatusOK {
		_ = json.NewDecoder(res.Body).Decode(&ok)
	} else {
		_ = json.NewDecoder(res.Body).Decode(&e)
	}
	return res.StatusCode, e, ok.KEK
}

// initVault has the server create its vault key with passphrase pw.
func initVault(t *testing.T, ts *httptest.Server) {
	t.Helper()
	if res, e := do(t, ts, "", "POST", "/api/system/init-key", `{"store":"vault","passphrase":"cHc="}`); res.StatusCode/100 != 2 {
		t.Fatalf("init-key %d %+v", res.StatusCode, e)
	}
}

func TestSealRouteNeedsTheUnlockedVaultAndItsPassphrase(t *testing.T) {
	for _, uid := range []int{0, 1000} {
		ts, s := peerServer(t, uid, true, nil)
		initVault(t, ts)
		if st, e, _ := sealPost(t, ts, "pw"); st != http.StatusServiceUnavailable || e.Code != api.CodeLocked {
			t.Fatalf("uid %d locked: %d %+v, want 503 locked", uid, st, e)
		}
		if _, err := s.Unlock([]byte("pw")); err != nil {
			t.Fatal(err)
		}
		if st, e, kek := sealPost(t, ts, "nope"); st != http.StatusForbidden || e.Code != api.CodeWrongPassphrase || kek != nil {
			t.Fatalf("uid %d wrong passphrase: %d %+v", uid, st, e)
		}
		st, e, kek := sealPost(t, ts, "pw")
		if st != http.StatusOK || len(kek) != 32 {
			t.Fatalf("uid %d: %d %+v, kek %d bytes", uid, st, e, len(kek))
		}
		s.Lock()
		if _, err := s.UnlockKEK(vault.SlotSystemdCreds, kek); err != nil {
			t.Fatalf("uid %d: the served KEK does not open the vault: %v", uid, err)
		}
	}
}

func TestSealRouteIsForSocketPeersOnly(t *testing.T) {
	ts, token, _ := systemServer(t, nil)
	if res, _ := do(t, ts, token, "POST", "/api/system/seal", `{"passphrase":"cHc="}`); res.StatusCode != http.StatusNotFound {
		t.Fatalf("over TCP: %d, want 404", res.StatusCode)
	}
}

func TestRestartRouteOnlyWhenConfigured(t *testing.T) {
	ts, _ := peerServer(t, 0, false, nil)
	if res, _ := do(t, ts, "", "POST", "/api/restart", ""); res.StatusCode != http.StatusNotFound {
		t.Fatalf("without Restart: %d, want 404", res.StatusCode)
	}
	called := make(chan struct{}, 2)
	ts, _ = peerServer(t, 0, false, func(c *Config) { c.Restart = func() { called <- struct{}{} } })
	if res, e := do(t, ts, "", "POST", "/api/restart", ""); res.StatusCode != http.StatusAccepted {
		t.Fatalf("with Restart: %d %+v, want 202", res.StatusCode, e)
	}
	select {
	case <-called:
	case <-time.After(2 * time.Second):
		t.Fatal("Restart was not called")
	}
	select {
	case <-called:
		t.Fatal("Restart called twice")
	case <-time.After(50 * time.Millisecond):
	}
}

func TestSystemInitKeyCreatesAVaultKeyAndLocks(t *testing.T) {
	ts, s := peerServer(t, 0, false, nil)
	res, e := do(t, ts, "", "POST", "/api/system/init-key", `{"store":"vault","passphrase":"cHc="}`)
	if res.StatusCode/100 != 2 {
		t.Fatalf("got %d %+v", res.StatusCode, e)
	}
	c, _ := config.Load()
	if c.Controller == nil || c.Controller.KeyStore != "vault" || c.Controller.KeyRef != "controller-key" || c.Controller.Address == "" {
		t.Fatalf("controller %+v", c.Controller)
	}
	if st := s.VaultState().State; st != "locked" {
		t.Fatalf("vault state %q, want locked", st)
	}
	if _, err := s.Unlock([]byte("pw")); err != nil {
		t.Fatalf("unlock with the setup passphrase: %v", err)
	}
}

func TestSystemInitKeyIsForSocketPeersOnly(t *testing.T) {
	ts, token, _ := systemServer(t, nil)
	if res, _ := do(t, ts, token, "POST", "/api/system/init-key", `{"store":"vault","passphrase":"cHc="}`); res.StatusCode != http.StatusNotFound {
		t.Fatalf("over TCP: %d, want 404", res.StatusCode)
	}
}

func TestSystemInitKeyRefusesASecondKey(t *testing.T) {
	ts, _ := peerServer(t, 0, false, nil)
	k, _ := signer.GenerateKey()
	recordController(t, k)
	res, e := do(t, ts, "", "POST", "/api/system/init-key", `{"store":"vault","passphrase":"cHc="}`)
	if res.StatusCode != http.StatusConflict || e.Code != api.CodeControllerExists || e.Hint == "" {
		t.Fatalf("got %d %+v, want 409 controller_exists", res.StatusCode, e)
	}
}

func TestSystemInitKeyRefusesPlainWithoutTheFlag(t *testing.T) {
	ts, _ := peerServer(t, 0, false, nil)
	res, e := do(t, ts, "", "POST", "/api/system/init-key", `{"store":"file"}`)
	if res.StatusCode != http.StatusBadRequest || !strings.Contains(e.Message, "--insecure-plain-key") {
		t.Fatalf("got %d %+v", res.StatusCode, e)
	}
}

func TestSystemInitKeyRefusesAClientChosenFileRef(t *testing.T) {
	ts, _ := peerServer(t, 0, false, nil)
	res, e := do(t, ts, "", "POST", "/api/system/init-key", `{"store":"file","insecurePlain":true,"ref":"/etc/shadow"}`)
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("got %d %+v, want 400", res.StatusCode, e)
	}
	if c, _ := config.Load(); c.Controller != nil {
		t.Fatalf("recorded %+v", c.Controller)
	}
}

func TestSystemInitKeyRecordsScopeSelf(t *testing.T) {
	ts, _ := peerServer(t, 0, false, nil)
	res, e := do(t, ts, "", "POST", "/api/system/init-key", `{"store":"file","insecurePlain":true,"scope":"self"}`)
	if res.StatusCode/100 != 2 {
		t.Fatalf("got %d %+v", res.StatusCode, e)
	}
	c, _ := config.Load()
	if c.Controller == nil || c.Controller.Scope != "self" || c.Controller.KeyStore != "file" {
		t.Fatalf("controller %+v", c.Controller)
	}
}
