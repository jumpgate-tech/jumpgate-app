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
