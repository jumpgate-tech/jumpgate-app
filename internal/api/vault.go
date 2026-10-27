package api

// VaultView is GET /api/vault and the answer to unlock and lock. State is
// none (no vault file), locked, unlocked or unusable (the file is there but
// cannot be used; Reason says why). Slots names each key slot's kind. Plain
// lists the secret fields still stored unencrypted in config.json.
type VaultView struct {
	State  string   `json:"state"`
	Slots  []string `json:"slots"`
	Plain  []string `json:"plain,omitempty"`
	Reason string   `json:"reason,omitempty"`
}

// UnlockRequest is POST /api/vault/unlock. Passphrase is a byte slice (base64
// in JSON) so both ends can clear it; it is never logged.
type UnlockRequest struct {
	Passphrase []byte `json:"passphrase"`
}

// SystemSeal is POST /api/system/seal: the vault passphrase, which proves
// the caller could unlock anyway. It is never logged.
type SystemSeal struct {
	Passphrase []byte `json:"passphrase"`
}

// SystemSealed is POST /api/system/seal's answer: the 32-byte key of the
// vault's new systemd-creds slot, for the CLI to seal with systemd-creds.
// It is never logged or stored in the clear.
type SystemSealed struct {
	KEK []byte `json:"kek"`
}

// SecretValue is PUT /api/vault/secrets/{name}. Value is base64 in JSON so
// both ends can clear it; it is never logged.
type SecretValue struct {
	Value []byte `json:"value"`
}
