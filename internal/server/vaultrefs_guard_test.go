package server

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/secrets"
	"github.com/valve-tech/jumpgate/internal/vault"
)

// A client cannot store a value that is itself a reference: it would make
// the field read (or later delete) another entry.
func TestClientValuesThatAreRefsAreRefused(t *testing.T) {
	for _, plain := range []bool{false, true} {
		_, ts, token, _ := refsServer(t, plain, nil)
		for _, body := range []string{
			`{"aiKey":"vault:controller-key"}`,
			`{"providerKeys":{"INFURA_API_KEY":"vault:transport-key"}}`,
		} {
			if res, _ := do(t, ts, token, "PUT", "/api/settings", body); res.StatusCode != http.StatusBadRequest {
				t.Errorf("plain=%v PUT %s = %d", plain, body, res.StatusCode)
			}
		}
		raw, _ := json.Marshal(map[string]any{"id": "wg1", "config": "vault:admin-token"})
		if res, _ := do(t, ts, token, "POST", "/api/vpns", string(raw)); res.StatusCode != http.StatusBadRequest {
			t.Errorf("plain=%v VPN save of a ref = %d", plain, res.StatusCode)
		}
		if c, _ := config.Load(); c.AIKey != "" || len(c.ProviderKeys) != 0 || len(c.VPNs) != 0 {
			t.Errorf("plain=%v config changed: %+v", plain, c)
		}
	}
}

// A field whose stored reference names another entry (an edited
// config.json) never reads the controller key, the transport key or a token.
func TestForeignRefsAreNeverResolved(t *testing.T) {
	const secret = "0xdeadbeefsecret"
	srv, ts, token, v := refsServer(t, false, func(c *config.Config) {
		c.Targets = append(c.Targets, config.Target{ID: "box", Mode: "local"})
		c.AIProvider, c.AIKey = "groq", "vault:"+secrets.NameControllerKey
		c.ProviderKeys = map[string]string{"INFURA_API_KEY": "vault:" + secrets.NameRelayToken}
	})
	for _, n := range []string{secrets.NameControllerKey, secrets.NameRelayToken, secrets.NameTransportKey, secrets.NameAdminToken} {
		if err := v.Put(n, []byte(secret), "test"); err != nil {
			t.Fatal(err)
		}
	}
	for _, n := range []string{secrets.NameControllerKey, secrets.NameTransportKey, secrets.NameAdminToken} {
		if _, err := srv.secret(secrets.NameAIKey, "vault:"+n); err == nil {
			t.Errorf("the AI key field read %s", n)
		}
		if err := srv.dropSecret(secrets.VPNName("wg1"), "vault:"+n); err == nil {
			t.Errorf("a VPN delete reached %s", n)
		}
	}
	keys, err := srv.providerKeys(mustLoad(t))
	if err == nil || strings.Contains(strings.Join(mapValues(keys), ","), secret) {
		t.Fatalf("provider keys %v err %v", keys, err)
	}
	res, _ := do(t, ts, token, "POST", "/api/targets/box/explain", `{"lines":["boom"]}`)
	if res.StatusCode == http.StatusOK {
		t.Fatal("explain ran with a foreign reference")
	}
	for _, n := range []string{secrets.NameControllerKey, secrets.NameRelayToken, secrets.NameTransportKey, secrets.NameAdminToken} {
		if got, err := v.Get(n); err != nil || string(got) != secret {
			t.Fatalf("%s is gone or changed: %v", n, err)
		}
	}
}

// Deleting an overlay whose stored config names a foreign entry leaves that
// entry alone.
func TestVPNDeleteLeavesAForeignRef(t *testing.T) {
	_, ts, token, v := refsServer(t, false, func(c *config.Config) {
		c.VPNs = append(c.VPNs, config.VPN{ID: "wg1", Config: "vault:" + secrets.NameControllerKey})
	})
	if err := v.Put(secrets.NameControllerKey, []byte("k"), "test"); err != nil {
		t.Fatal(err)
	}
	if res, _ := do(t, ts, token, "DELETE", "/api/vpns/wg1", ""); res.StatusCode != http.StatusNoContent {
		t.Fatalf("delete = %d", res.StatusCode)
	}
	if _, err := v.Get(secrets.NameControllerKey); err != nil {
		t.Fatalf("the controller key went with the overlay: %v", err)
	}
}

// With a provider key held in a locked vault, a stored upstream URL that
// carries that key cannot be redacted, so no gateway view goes out.
func TestLockedGatewayViewsNeverLeakAKey(t *testing.T) {
	const key = "infura-secret-123"
	srv, ts, token, v := refsServer(t, false, func(c *config.Config) {
		c.ProviderKeys = map[string]string{"INFURA_API_KEY": "vault:provider/INFURA_API_KEY"}
		c.Gateways = append(c.Gateways, config.Gateway{ID: "g1"})
	})
	if err := v.Put(secrets.ProviderName("INFURA_API_KEY"), []byte(key), "test"); err != nil {
		t.Fatal(err)
	}
	if _, err := config.Update(func(c *config.Config) error {
		raw := `{"networks":[{"chainId":1,"upstreams":[{"endpoint":"https://mainnet.infura.io/v3/` + key + `"}]}]}`
		return json.Unmarshal([]byte(raw), &c.Gateways[len(c.Gateways)-1].Config)
	}); err != nil {
		t.Fatal(err)
	}
	srv.Lock()
	for _, p := range []string{"/api/gateways", "/api/gateways/g1", "/api/gateways/g1/capabilities"} {
		req, _ := http.NewRequest("GET", ts.URL+p, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		raw, _ := io.ReadAll(res.Body)
		res.Body.Close()
		var e api.Error
		_ = json.Unmarshal(raw, &e)
		if strings.Contains(string(raw), key) {
			t.Errorf("GET %s leaked the key: %s", p, raw)
		}
		if res.StatusCode != http.StatusServiceUnavailable || e.Code != api.CodeLocked {
			t.Errorf("GET %s = %d %s", p, res.StatusCode, raw)
		}
	}
	if _, err := srv.providerKeys(mustLoad(t)); err == nil || !strings.Contains(err.Error(), vault.ErrLocked.Error()) {
		t.Fatalf("providerKeys err %v", err)
	}
}

func mustLoad(t *testing.T) config.Config {
	t.Helper()
	c, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func mapValues(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for _, v := range m {
		out = append(out, v)
	}
	return out
}
