package signer

import (
	"context"
	"encoding/hex"
	"errors"
	"testing"

	"github.com/valve-tech/jumpgate/internal/vault/vaulttest"
)

func TestCreateVaultStoresAHexKeyThatOpens(t *testing.T) {
	v := vaulttest.Unlocked(t)
	k, err := CreateVault(v, "controller-key")
	if err != nil {
		t.Fatal(err)
	}
	got, err := OpenVault(v, "controller-key")
	if err != nil {
		t.Fatal(err)
	}
	if got.Address() != k.Address() {
		t.Fatalf("OpenVault = %s, want %s", got.Address().Hex(), k.Address().Hex())
	}
	raw, err := v.Get("controller-key")
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) != 64 {
		t.Fatalf("stored value is %d bytes, want 64 hex chars", len(raw))
	}
	if _, err := hex.DecodeString(string(raw)); err != nil {
		t.Fatalf("stored value is not hex: %v", err)
	}
}

func TestCreateVaultNeverReplacesAKey(t *testing.T) {
	v := vaulttest.Unlocked(t)
	if _, err := CreateVault(v, "controller-key"); err != nil {
		t.Fatal(err)
	}
	if _, err := CreateVault(v, "controller-key"); !errors.Is(err, ErrKeyExists) {
		t.Fatalf("second CreateVault: %v, want ErrKeyExists", err)
	}
}

func TestOpenAndCreateRefuseTheVaultStore(t *testing.T) {
	if _, err := Open(context.Background(), StoreVault, "x"); !errors.Is(err, ErrVaultStore) {
		t.Fatalf("Open: %v", err)
	}
	if _, err := Create(context.Background(), StoreVault, "x"); !errors.Is(err, ErrVaultStore) {
		t.Fatalf("Create: %v", err)
	}
}
