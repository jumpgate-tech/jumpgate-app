package secrets

import (
	"bytes"
	"context"
	"testing"

	"github.com/valve-tech/jumpgate/internal/signer"
	"github.com/valve-tech/jumpgate/internal/vault"
	"github.com/valve-tech/jumpgate/internal/vault/vaulttest"
)

func TestKeystoreSlotUnlocksWithoutAPrompt(t *testing.T) {
	signer.FakeKeychainForTest(t)
	v := vaulttest.Unlocked(t)
	if err := v.Put("ai-key", []byte("v"), "test"); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := AddKeystoreSlot(ctx, v, signer.StoreKeychain, KeystoreRef); err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, s := range v.Slots() {
		if s.Kind == vault.SlotKeystore && s.Store == "keychain" && s.Ref == KeystoreRef {
			found = true
		}
	}
	if !found {
		t.Fatalf("slots %+v", v.Slots())
	}
	// Adding it again reuses the item.
	if err := AddKeystoreSlot(ctx, v, signer.StoreKeychain, KeystoreRef); err != nil {
		t.Fatal(err)
	}
	if n := len(v.Slots()); n != 2 {
		t.Fatalf("%d slots after a second add, want 2", n)
	}
	v.Lock()
	kek, err := KeystoreKEK(ctx, signer.StoreKeychain, KeystoreRef)
	if err != nil {
		t.Fatal(err)
	}
	if len(kek) != 32 {
		t.Fatalf("kek %d bytes", len(kek))
	}
	again, _ := KeystoreKEK(ctx, signer.StoreKeychain, KeystoreRef)
	if !bytes.Equal(kek, again) {
		t.Fatal("KeystoreKEK is not deterministic")
	}
	if err := v.UnlockKEK(vault.SlotKeystore, kek); err != nil {
		t.Fatal(err)
	}
	if got, err := v.Get("ai-key"); err != nil || string(got) != "v" {
		t.Fatalf("Get = %q, %v", got, err)
	}
}

func TestKeystoreSlotRefusesTheVaultStore(t *testing.T) {
	v := vaulttest.Unlocked(t)
	if err := AddKeystoreSlot(context.Background(), v, signer.StoreVault, KeystoreRef); err == nil {
		t.Fatal("a keystore slot on the vault store")
	}
}
