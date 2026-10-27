package secrets

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/hkdf"

	"github.com/valve-tech/jumpgate/internal/signer"
	"github.com/valve-tech/jumpgate/internal/vault"
)

// KeystoreRef is the OS key store item a keystore slot is made from.
const KeystoreRef = "jumpgate-vault"

const keystoreInfo = "jumpgate vault keystore kek v1"

// keystoreKEK derives a slot key from an OS key store item's scalar.
func keystoreKEK(k *signer.Key) ([]byte, error) {
	ikm := k.Bytes()
	defer clear(ikm)
	kek := make([]byte, 32)
	if _, err := io.ReadFull(hkdf.New(sha256.New, ikm, nil, []byte(keystoreInfo)), kek); err != nil {
		clear(kek)
		return nil, fmt.Errorf("secrets: keystore key: %w", err)
	}
	return kek, nil
}

// KeystoreKEK opens the OS key store item ref in store and returns the
// 32-byte key of its vault slot: HKDF-SHA256 over the item's key bytes. The
// caller clears it (UnlockKEK does).
func KeystoreKEK(ctx context.Context, store signer.Store, ref string) ([]byte, error) {
	k, err := signer.Open(ctx, store, ref)
	if err != nil {
		return nil, err
	}
	return keystoreKEK(k)
}

// AddKeystoreSlot creates the OS key store item ref in store (or opens it
// when it already exists) and adds a keystore slot to the unlocked vault v
// whose key comes from it, so the vault unlocks with no prompt for whoever
// can read that item.
func AddKeystoreSlot(ctx context.Context, v *vault.Vault, store signer.Store, ref string) error {
	switch store {
	case signer.StoreKeychain, signer.StoreWinCred, signer.StoreOnePassword:
	default:
		return fmt.Errorf("secrets: a keystore slot needs an OS key store, not %q", store)
	}
	k, err := signer.Create(ctx, store, ref)
	if errors.Is(err, signer.ErrKeyExists) {
		k, err = signer.Open(ctx, store, ref)
	}
	if err != nil {
		return err
	}
	kek, err := keystoreKEK(k)
	if err != nil {
		return err
	}
	return v.AddKEKSlot(vault.Slot{Kind: vault.SlotKeystore, Store: string(store), Ref: ref}, kek) // clears kek
}
