package server

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"net/http"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/vault"
)

// sealCred is the systemd credential, and the vault slot's Cred name, that
// holds the service's unattended-unlock key.
const sealCred = "jumpgate-vault-key"

// sealKEK draws a slot key. The vault has no call that returns a secret for
// a slot it made, so the service draws it and adds the slot itself.
func sealKEK() ([]byte, error) {
	kek := make([]byte, 32)
	if _, err := rand.Read(kek); err != nil {
		return nil, err
	}
	return kek, nil
}

// handleSystemSeal is POST /api/system/seal: `jumpgate vault seal` against
// the jumpgate service. The service owns the vault, so it adds the
// systemd-creds slot and hands the slot key to the CLI, which seals it with
// systemd-creds as root. Any admitted peer (root or jumpgate-admin) may ask,
// but only with the vault's passphrase: the key opens nothing its holder
// could not already open, and a member who can reach an unlocked service
// without knowing the passphrase does not get a lasting key from it.
func (s *Server) handleSystemSeal(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.socketPeer(w, r); !ok {
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	var req api.SystemSeal
	err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxUnlockBody)).Decode(&req)
	defer clear(req.Passphrase)
	if err != nil {
		writeError(w, http.StatusBadRequest, `want {"passphrase": base64}`)
		return
	}
	if len(req.Passphrase) == 0 {
		writeError(w, http.StatusBadRequest, "the passphrase is empty")
		return
	}
	s.vctl.mu.Lock()
	defer s.vctl.mu.Unlock()
	v, err := s.openVaultLocked()
	if err != nil {
		writeVaultError(w, err)
		return
	}
	if v.Locked() {
		writeVaultError(w, vault.ErrLocked)
		return
	}
	// The passphrase is checked on a handle of its own, so the server's
	// unlocked vault is not touched by a wrong one.
	chk, err := vault.Open(v.Path())
	if err != nil {
		writeVaultError(w, err)
		return
	}
	if err := chk.Unlock(req.Passphrase); err != nil {
		writeVaultError(w, err)
		return
	}
	chk.Lock()
	kek, err := sealKEK()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer clear(kek)
	if err := v.AddKEKSlot(vault.Slot{Kind: vault.SlotSystemdCreds, Cred: sealCred}, bytes.Clone(kek)); err != nil { // clears its copy
		writeVaultError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, api.SystemSealed{KEK: kek})
}
