package secrets

import (
	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/vault"
)

// The fixed vault entry names.
const (
	NameControllerKey = "controller-key"
	NameTransportKey  = "transport-key"
	NameAIKey         = "ai-key"
	NameRelayToken    = "relay-token"
	NameAdminToken    = "admin-token"
)

// ProviderName is the entry for the provider key of a ${placeholder} slot.
func ProviderName(placeholder string) string { return "provider/" + placeholder }

// VPNName is the entry for a WireGuard client config.
func VPNName(id string) string { return "vpn/" + id }

// Resolve returns s itself when it is a literal, or the value the vault
// holds for a vault:NAME reference. A nil or locked vault is vault.ErrLocked.
func Resolve(v *vault.Vault, s string) (string, error) {
	name, ok := config.SecretRefName(s)
	if !ok {
		return s, nil
	}
	if v == nil {
		return "", vault.ErrLocked
	}
	raw, err := v.Get(name)
	if err != nil {
		return "", err
	}
	out := string(raw)
	clear(raw)
	return out, nil
}
