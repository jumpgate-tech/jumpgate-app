package signer

import (
	"encoding/hex"
	"errors"
	"fmt"
)

// credStore is Windows Credential Manager, behind an interface so the store's
// rules (fail closed, never replace, verify) are tested on every OS.
type credStore interface {
	read(target string) ([]byte, error) // errCredNotFound when absent
	write(target string, blob []byte) error
	del(target string) error
}

var errCredNotFound = errors.New("signer: credential not found")

// winCreds is the real Credential Manager on Windows and a store that refuses
// everything elsewhere. Tests replace it.
var winCreds credStore = platformCreds()

// credTarget is the generic credential's target name for a ref.
func credTarget(ref string) string { return "jumpgate/" + ref }

func winCredCreate(ref string) (*Key, error) {
	if !keychainNameRE.MatchString(ref) {
		return nil, fmt.Errorf("signer: credential name %q may only contain letters, digits, '.', '_' and '-'", ref)
	}
	target := credTarget(ref)
	// Fail closed: only a recognised not-found means absent, because treating
	// any other failure as absent could replace a real key.
	switch b, err := winCreds.read(target); {
	case err == nil:
		clear(b)
		return nil, fmt.Errorf("%w: Windows credential %q", ErrKeyExists, target)
	case !errors.Is(err, errCredNotFound):
		return nil, fmt.Errorf("signer: Windows Credential Manager existence check failed: %w", err)
	}
	k, err := GenerateKey()
	if err != nil {
		return nil, err
	}
	raw := k.Bytes()
	blob := make([]byte, hex.EncodedLen(len(raw)))
	hex.Encode(blob, raw)
	clear(raw)
	err = winCreds.write(target, blob)
	clear(blob)
	if err != nil {
		return nil, fmt.Errorf("signer: store in Windows Credential Manager: %w", err)
	}
	k2, err := verifyStored(k, func() (*Key, error) { return winCredRead(ref) })
	if err != nil {
		return nil, fmt.Errorf("signer: a key was written to Windows credential %q but could not be verified; inspect or remove it in Credential Manager: %w", target, err)
	}
	return k2, nil
}

func winCredRead(ref string) (*Key, error) {
	b, err := winCreds.read(credTarget(ref))
	if errors.Is(err, errCredNotFound) {
		return nil, fmt.Errorf("signer: no Windows credential %q; run `jumpgate keys init --store wincred`", credTarget(ref))
	}
	if err != nil {
		return nil, fmt.Errorf("signer: read Windows Credential Manager: %w", err)
	}
	defer clear(b)
	return keyFromHexBytes(b)
}

// keyFromHexBytes is keyFromHex for a secret held in a byte slice, which,
// unlike a string, can be zeroed once the key is built.
func keyFromHexBytes(b []byte) (*Key, error) {
	raw := make([]byte, hex.DecodedLen(len(b)))
	defer clear(raw)
	n, err := hex.Decode(raw, b)
	if err != nil {
		return nil, fmt.Errorf("signer: stored key is not hex")
	}
	return KeyFromBytes(raw[:n])
}
