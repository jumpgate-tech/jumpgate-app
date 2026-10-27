package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/chainlist"
	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/secrets"
	"github.com/valve-tech/jumpgate/internal/testutil"
	"github.com/valve-tech/jumpgate/internal/vault"
	"github.com/valve-tech/jumpgate/internal/vault/vaulttest"
)

// refsServer is a server with secretStore "vault" (unless plain) and an
// unlocked fixture vault, after edit has shaped config.json.
func refsServer(t *testing.T, plain bool, edit func(c *config.Config)) (*Server, *httptest.Server, string, *vault.Vault) {
	t.Helper()
	testutil.Home(t)
	if _, err := config.Update(func(c *config.Config) error {
		if !plain {
			c.SecretStore = config.SecretStoreVault
		}
		if edit != nil {
			edit(c)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	v := vaulttest.Unlocked(t)
	feed := feedStub(t, []chainlist.Chain{{ChainID: 369, Name: "PulseChain"}})
	d := chainlist.New()
	d.FeedURL = feed.URL
	token := NewSessionToken()
	srv := New(Config{Token: token, UI: fstest.MapFS{}, Vault: v, NewChainlist: func() *chainlist.Discoverer { return d }})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(func() { ts.Close(); srv.fleet.stop(); srv.jobs.stopWithin(time.Second) })
	return srv, ts, token, v
}

func configBytes(t *testing.T) []byte {
	t.Helper()
	dir, err := config.Dir()
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func vaultValue(t *testing.T, v *vault.Vault, name string) string {
	t.Helper()
	raw, err := v.Get(name)
	if err != nil {
		t.Fatalf("vault %s: %v", name, err)
	}
	return string(raw)
}

func TestSettingsWriteGoesIntoTheVault(t *testing.T) {
	_, ts, token, v := refsServer(t, false, nil)
	res, e := do(t, ts, token, "PUT", "/api/settings", `{"aiKey":"sk-1","providerKeys":{"INFURA_API_KEY":"i"}}`)
	if res.StatusCode != 200 {
		t.Fatalf("PUT = %d %+v", res.StatusCode, e)
	}
	c, _ := config.Load()
	if c.AIKey != "vault:ai-key" || c.ProviderKeys["INFURA_API_KEY"] != "vault:provider/INFURA_API_KEY" {
		t.Fatalf("config holds %q %v", c.AIKey, c.ProviderKeys)
	}
	if raw := configBytes(t); bytes.Contains(raw, []byte("sk-1")) {
		t.Fatal("config.json holds the plain AI key")
	}
	if vaultValue(t, v, secrets.NameAIKey) != "sk-1" || vaultValue(t, v, secrets.ProviderName("INFURA_API_KEY")) != "i" {
		t.Fatal("the vault does not hold the values")
	}
	var got settingsResponse
	req, _ := http.NewRequest("GET", ts.URL+"/api/settings", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r2, _ := http.DefaultClient.Do(req)
	_ = json.NewDecoder(r2.Body).Decode(&got)
	r2.Body.Close()
	if !got.AIKeySet || len(got.ProviderKeysSet) != 1 {
		t.Fatalf("settings %+v", got)
	}
	// Clearing a key forgets its vault entry too.
	if res, _ := do(t, ts, token, "PUT", "/api/settings", `{"aiKey":""}`); res.StatusCode != 200 {
		t.Fatalf("clear = %d", res.StatusCode)
	}
	if _, err := v.Get(secrets.NameAIKey); err == nil {
		t.Fatal("the cleared key is still in the vault")
	}
}

// Review Focus 4: a key saved while locked is refused, and nothing lands in
// config.json, plain or otherwise.
func TestSettingsWhileLockedWritesNothing(t *testing.T) {
	srv, ts, token, _ := refsServer(t, false, nil)
	srv.Lock()
	before := configBytes(t)
	res, e := do(t, ts, token, "PUT", "/api/settings", `{"aiKey":"sk-1","providerKeys":{"INFURA_API_KEY":"i"}}`)
	if res.StatusCode != http.StatusServiceUnavailable || e.Code != api.CodeLocked {
		t.Fatalf("PUT = %d %+v", res.StatusCode, e)
	}
	if after := configBytes(t); !bytes.Equal(before, after) {
		t.Fatalf("config.json changed:\n%s", after)
	}
}

func TestExplainWhileLockedIsLocked(t *testing.T) {
	srv, ts, token, _ := refsServer(t, false, func(c *config.Config) {
		c.Targets = append(c.Targets, config.Target{ID: "box", Mode: "local"})
		c.AIProvider, c.AIKey = "groq", "vault:ai-key"
	})
	srv.Lock()
	res, e := do(t, ts, token, "POST", "/api/targets/box/explain", `{"lines":["boom"]}`)
	if res.StatusCode != http.StatusServiceUnavailable || e.Code != api.CodeLocked {
		t.Fatalf("explain = %d %+v", res.StatusCode, e)
	}
}

func TestVPNStoredAsARef(t *testing.T) {
	_, ts, token, v := refsServer(t, false, nil)
	body, _ := json.Marshal(map[string]any{"id": "wg1", "provider": "protonvpn", "config": testWGConf})
	res, e := do(t, ts, token, "POST", "/api/vpns", string(body))
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("save = %d %+v", res.StatusCode, e)
	}
	c, _ := config.Load()
	if got, _ := c.FindVPN("wg1"); got.Config != "vault:vpn/wg1" {
		t.Fatalf("stored %q", got.Config)
	}
	if vaultValue(t, v, secrets.VPNName("wg1")) != testWGConf {
		t.Fatal("the vault does not hold the config")
	}
	var views []vpnView
	req, _ := http.NewRequest("GET", ts.URL+"/api/vpns", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r2, _ := http.DefaultClient.Do(req)
	_ = json.NewDecoder(r2.Body).Decode(&views)
	r2.Body.Close()
	if len(views) != 1 || !views[0].Valid {
		t.Fatalf("views %+v", views)
	}
	if res, _ := do(t, ts, token, "DELETE", "/api/vpns/wg1", ""); res.StatusCode != http.StatusNoContent {
		t.Fatalf("delete = %d", res.StatusCode)
	}
	if _, err := v.Get(secrets.VPNName("wg1")); err == nil {
		t.Fatal("the deleted overlay's config is still in the vault")
	}
}

func TestVPNViewWhileLocked(t *testing.T) {
	srv, ts, token, _ := refsServer(t, false, func(c *config.Config) {
		c.VPNs = append(c.VPNs, config.VPN{ID: "wg1", Provider: "protonvpn", Config: "vault:vpn/wg1"})
	})
	srv.Lock()
	req, _ := http.NewRequest("GET", ts.URL+"/api/vpns", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r2, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	var views []vpnView
	_ = json.NewDecoder(r2.Body).Decode(&views)
	r2.Body.Close()
	if r2.StatusCode != 200 || len(views) != 1 || views[0].Valid || views[0].Error != "locked: unlock to read" {
		t.Fatalf("GET = %d %+v", r2.StatusCode, views)
	}
	res, e := do(t, ts, token, "POST", "/api/vpns/wg1/up", "")
	if res.StatusCode != http.StatusServiceUnavailable || e.Code != api.CodeLocked {
		t.Fatalf("up = %d %+v", res.StatusCode, e)
	}
}

func TestGatewayProvisionWhileLocked(t *testing.T) {
	srv, ts, token, _ := refsServer(t, false, func(c *config.Config) {
		c.ProviderKeys = map[string]string{"INFURA_API_KEY": "vault:provider/INFURA_API_KEY"}
		c.Gateways = append(c.Gateways, config.Gateway{ID: "g1"})
	})
	srv.Lock()
	res, e := do(t, ts, token, "POST", "/api/gateways/g1/provision", "")
	if res.StatusCode != http.StatusServiceUnavailable || e.Code != api.CodeLocked {
		t.Fatalf("provision = %d %+v", res.StatusCode, e)
	}
	res, e = do(t, ts, token, "GET", "/api/gateways/g1/knownset/369", "")
	if res.StatusCode != http.StatusServiceUnavailable || e.Code != api.CodeLocked {
		t.Fatalf("knownset = %d %+v", res.StatusCode, e)
	}
	if res, e := do(t, ts, token, "GET", "/api/chainlist/369", ""); res.StatusCode != 200 {
		t.Fatalf("chainlist = %d %+v", res.StatusCode, e)
	}
}

func TestPlainStoreStillWritesLiterals(t *testing.T) {
	_, ts, token, v := refsServer(t, true, nil)
	if res, e := do(t, ts, token, "PUT", "/api/settings", `{"aiKey":"sk-1"}`); res.StatusCode != 200 {
		t.Fatalf("PUT = %d %+v", res.StatusCode, e)
	}
	if c, _ := config.Load(); c.AIKey != "sk-1" {
		t.Fatalf("stored %q", c.AIKey)
	}
	if names, _ := v.Names(); len(names) != 0 {
		t.Fatalf("the vault got %v", names)
	}
}

func TestVaultViewListsPlainSecrets(t *testing.T) {
	srv, _, _, _ := refsServer(t, true, func(c *config.Config) { c.AIKey = "sk-plain" })
	if v := srv.VaultState(); strings.Join(v.Plain, ",") != "aiKey" {
		t.Fatalf("plain %v", v.Plain)
	}
}
