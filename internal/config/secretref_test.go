package config

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/valve-tech/jumpgate/internal/testutil"
)

func TestSecretRefRoundTrip(t *testing.T) {
	ref := SecretRef("provider/X")
	if ref != "vault:provider/X" {
		t.Fatalf("SecretRef = %q", ref)
	}
	if !IsSecretRef(ref) {
		t.Fatal("IsSecretRef(ref) = false")
	}
	name, ok := SecretRefName(ref)
	if !ok || name != "provider/X" {
		t.Fatalf("SecretRefName = %q, %v", name, ok)
	}
	for _, s := range []string{"vaultx", "vault:", "sk-live", ""} {
		if IsSecretRef(s) {
			t.Errorf("IsSecretRef(%q) = true", s)
		}
		if _, ok := SecretRefName(s); ok {
			t.Errorf("SecretRefName(%q) ok", s)
		}
	}
}

func TestPlainSecretsListsEveryPlainField(t *testing.T) {
	c := Config{
		AIKey: "sk-live",
		ProviderKeys: map[string]string{
			"INFURA_API_KEY": "abc",
			"VALVE_API_KEY":  SecretRef("provider/VALVE_API_KEY"),
		},
		VPNs: []VPN{
			{ID: "proton", Config: "[Interface]\nPrivateKey = x\n"},
			{ID: "ref", Config: SecretRef("vpn/ref")},
		},
	}
	got := c.PlainSecrets()
	want := []string{"aiKey", "providerKeys.INFURA_API_KEY", "vpns.proton.config"}
	if !slices.Equal(got, want) {
		t.Fatalf("PlainSecrets = %v, want %v", got, want)
	}
	if n := (Config{}).PlainSecrets(); len(n) != 0 {
		t.Fatalf("empty config PlainSecrets = %v", n)
	}
}

func TestSaveRefusesPlainSecretsUnderTheVault(t *testing.T) {
	testutil.Home(t)
	err := Config{SecretStore: SecretStoreVault, AIKey: "sk-live"}.Save()
	if err == nil || !strings.Contains(err.Error(), "aiKey") {
		t.Fatalf("Save err = %v, want one naming aiKey", err)
	}
	dir, derr := Dir()
	if derr != nil {
		t.Fatal(derr)
	}
	if _, serr := os.Stat(filepath.Join(dir, "config.json")); !errors.Is(serr, os.ErrNotExist) {
		t.Fatalf("config.json written: %v", serr)
	}
	if err := (Config{SecretStore: SecretStoreVault, AIKey: "vault:ai-key"}).Save(); err != nil {
		t.Fatalf("Save with ref: %v", err)
	}
}

func TestSecretStoreSurvivesLoad(t *testing.T) {
	testutil.Home(t)
	if _, err := Update(func(c *Config) error {
		c.SecretStore = SecretStoreVault
		c.VaultIdleLockMinutes = 15
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.SecretStore != SecretStoreVault || c.VaultIdleLockMinutes != 15 {
		t.Fatalf("loaded %q %d", c.SecretStore, c.VaultIdleLockMinutes)
	}
}

func TestAddNoticeDedupes(t *testing.T) {
	var c Config
	c.AddNotice("a")
	c.AddNotice("a")
	if len(c.Notices) != 1 {
		t.Fatalf("Notices = %v", c.Notices)
	}
}
