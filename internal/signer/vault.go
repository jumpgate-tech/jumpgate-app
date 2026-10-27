package signer

import (
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/valve-tech/jumpgate/internal/vault"
)

// ErrVaultStore is what Open and Create return for StoreVault: a vault key
// opens only through an unlocked vault (OpenVault, CreateVault).
var ErrVaultStore = errors.New("signer: a vault key opens only through an unlocked vault; run `jumpgate unlock`")

// OpenVault loads the hex key stored in v under ref.
func OpenVault(v *vault.Vault, ref string) (*Key, error) {
	raw, err := v.Get(ref)
	if err != nil {
		return nil, fmt.Errorf("signer: %w", err)
	}
	defer clear(raw)
	return keyFromHex(string(raw))
}

// CreateVault makes a new key and stores it in v under ref as hex, refusing
// to replace a key already there.
func CreateVault(v *vault.Vault, ref string) (*Key, error) {
	if _, err := v.Get(ref); err == nil {
		return nil, ErrKeyExists
	} else if !errors.Is(err, vault.ErrNotFound) {
		return nil, fmt.Errorf("signer: %w", err)
	}
	k, err := GenerateKey()
	if err != nil {
		return nil, err
	}
	kb := k.Bytes()
	val := []byte(hex.EncodeToString(kb))
	clear(kb)
	err = v.Put(ref, val, "keys init")
	clear(val)
	if err != nil {
		return nil, fmt.Errorf("signer: %w", err)
	}
	return verifyStored(k, func() (*Key, error) { return OpenVault(v, ref) })
}
