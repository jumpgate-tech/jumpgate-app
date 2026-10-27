package secrets

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"
	"sync"

	"golang.org/x/crypto/ssh"

	"github.com/valve-tech/jumpgate/internal/vault"
)

// transportMu keeps two callers in this process from each generating a key,
// so every caller is handed the same one.
var transportMu sync.Mutex

// EnsureTransportKey returns the controller's tunnel key, held in the vault
// as transport-key, as an authorized_keys line. It creates an ed25519 key
// (OpenSSH PEM, comment jumpgate-controller) when the entry is absent. A nil
// or locked vault is vault.ErrLocked.
func EnsureTransportKey(v *vault.Vault) (authorizedLine string, err error) {
	transportMu.Lock()
	defer transportMu.Unlock()
	s, err := TransportSigner(v)
	if errors.Is(err, vault.ErrNotFound) {
		if err = generateTransportKey(v); err == nil {
			s, err = TransportSigner(v)
		}
	}
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(ssh.MarshalAuthorizedKey(s.PublicKey()))) + " jumpgate-controller", nil
}

func generateTransportKey(v *vault.Vault) error {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return err
	}
	defer clear(priv)
	block, err := ssh.MarshalPrivateKey(priv, "jumpgate-controller")
	if err != nil {
		return err
	}
	defer clear(block.Bytes)
	raw := pem.EncodeToMemory(block)
	defer clear(raw)
	return v.Put(NameTransportKey, raw, "generated")
}

// TransportSigner is the transport key as an in-memory ssh.Signer. A nil or
// locked vault is vault.ErrLocked; no key yet is vault.ErrNotFound.
func TransportSigner(v *vault.Vault) (ssh.Signer, error) {
	if v == nil {
		return nil, vault.ErrLocked
	}
	raw, err := v.Get(NameTransportKey)
	if err != nil {
		return nil, err
	}
	defer clear(raw)
	s, err := ssh.ParsePrivateKey(raw)
	if err != nil {
		return nil, fmt.Errorf("transport key in the vault: %w", err)
	}
	return s, nil
}
