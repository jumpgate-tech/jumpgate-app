package secrets

import (
	"errors"
	"testing"

	"github.com/valve-tech/jumpgate/internal/vault"
	"github.com/valve-tech/jumpgate/internal/vault/vaulttest"
)

func TestResolvePassesLiteralsAndReadsRefs(t *testing.T) {
	if got, err := Resolve(nil, "sk"); got != "sk" || err != nil {
		t.Fatalf("literal: %q %v", got, err)
	}
	if _, err := Resolve(nil, "vault:ai-key"); !errors.Is(err, vault.ErrLocked) {
		t.Fatalf("nil vault: %v", err)
	}
	v := vaulttest.Unlocked(t)
	if err := v.Put(NameAIKey, []byte("sk-1"), "test"); err != nil {
		t.Fatal(err)
	}
	if got, err := Resolve(v, "vault:ai-key"); got != "sk-1" || err != nil {
		t.Fatalf("unlocked: %q %v", got, err)
	}
	if _, err := Resolve(v, "vault:"+ProviderName("X")); !errors.Is(err, vault.ErrNotFound) {
		t.Fatalf("missing entry: %v", err)
	}
	v.Lock()
	if _, err := Resolve(v, "vault:ai-key"); !errors.Is(err, vault.ErrLocked) {
		t.Fatalf("locked: %v", err)
	}
}

func TestNames(t *testing.T) {
	if ProviderName("INFURA_API_KEY") != "provider/INFURA_API_KEY" || VPNName("wg1") != "vpn/wg1" {
		t.Fatal("names")
	}
}
