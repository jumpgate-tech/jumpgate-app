package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/secrets"
	"github.com/valve-tech/jumpgate/internal/signer"
	"github.com/valve-tech/jumpgate/internal/vault"
)

// maxUnlockBody bounds POST /api/vault/unlock.
const maxUnlockBody = 4 << 10

// keyState is the vault and the controller signer, which may live in it.
// fromVault means the signer is vault-held: while the vault is locked,
// signer() answers vault.ErrLocked and nothing is signed, with no fallback
// to any other key store.
type keyState struct {
	mu        sync.RWMutex
	v         *vault.Vault // nil until found (S8)
	sgn       signer.Signer
	fromVault bool
	// ready: Config.Unlocked has run since the vault was last unlocked.
	// Until then a vault-held key reads locked, even though the vault no
	// longer is.
	ready bool
	err   error // why sgn is nil
}

// vaultCtl serializes the vault's state changes (open, unlock, lock), so an
// unlock's signer is never installed after a lock that ran meanwhile.
type vaultCtl struct {
	mu       sync.Mutex
	unlocked bool // this process has unlocked at least once
}

func (s *Server) initKeys() {
	s.keys.v = s.cfg.Vault
	s.keys.sgn, s.keys.err = s.cfg.Signer, s.cfg.SignerErr
	s.keys.fromVault = s.cfg.Signer == nil && errors.Is(s.cfg.SignerErr, signer.ErrVaultStore)
}

// signer is the controller key to sign with. A nil signer comes with the
// reason (nil when no key is recorded); vault.ErrLocked means the key is
// held in the vault and the vault is locked.
func (s *Server) signer() (signer.Signer, error) {
	s.keys.mu.RLock()
	v, sgn, fromVault, ready, err := s.keys.v, s.keys.sgn, s.keys.fromVault, s.keys.ready, s.keys.err
	s.keys.mu.RUnlock()
	if fromVault {
		if v == nil || !ready || v.Locked() {
			return nil, vault.ErrLocked
		}
		if sgn == nil {
			if err == nil {
				err = errNoController
			}
			return nil, err
		}
		return sgn, nil
	}
	if sgn != nil {
		return sgn, nil
	}
	// A controller recorded in the vault but never opened: locked, never
	// "no key" (which would send the operator to make a second one).
	if c, lerr := config.Load(); lerr == nil && c.Controller != nil && c.Controller.KeyStore == string(signer.StoreVault) {
		return nil, vault.ErrLocked
	}
	return nil, err
}

// keyLocked reports whether signing waits for an unlock.
func (s *Server) keyLocked() bool {
	_, err := s.signer()
	return errors.Is(err, vault.ErrLocked)
}

// vaultPath is Config.VaultPath, or ~/.jumpgate/vault.
func (s *Server) vaultPath() (string, error) {
	if s.cfg.VaultPath != "" {
		return s.cfg.VaultPath, nil
	}
	return secrets.VaultPath()
}

// openVaultLocked returns the vault, opening the file the first time it is
// found. Callers hold s.vctl.mu.
func (s *Server) openVaultLocked() (*vault.Vault, error) {
	s.keys.mu.RLock()
	v := s.keys.v
	s.keys.mu.RUnlock()
	if v != nil {
		return v, nil
	}
	p, err := s.vaultPath()
	if err != nil {
		return nil, err
	}
	v, err = vault.Open(p)
	if err != nil {
		return nil, err
	}
	s.keys.mu.Lock()
	s.keys.v = v
	s.keys.mu.Unlock()
	return v, nil
}

// viewLocked describes the vault. A locked vault's file is checked again,
// so a file made unsafe or replaced reads unusable, never locked. Callers
// hold s.vctl.mu.
func (s *Server) viewLocked() api.VaultView {
	view := api.VaultView{Slots: []string{}, Plain: s.plainSecrets()}
	v, err := s.openVaultLocked()
	if err == nil && v.Locked() {
		_, err = vault.Open(v.Path())
	}
	switch {
	case errors.Is(err, vault.ErrNoVault):
		view.State = "none"
		return view
	case err != nil:
		view.State, view.Reason = "unusable", err.Error()
		return view
	case v.Locked():
		view.State = "locked"
	default:
		view.State = "unlocked"
	}
	for _, sl := range v.Slots() {
		view.Slots = append(view.Slots, string(sl.Kind))
	}
	return view
}

// VaultState is GET /api/vault: none, locked, unlocked or unusable.
func (s *Server) VaultState() api.VaultView {
	s.vctl.mu.Lock()
	defer s.vctl.mu.Unlock()
	return s.viewLocked()
}

// Unlock opens the vault with a passphrase, then loads the controller key
// from it and resumes what waited. It clears passphrase.
func (s *Server) Unlock(passphrase []byte) (api.VaultView, error) {
	defer clear(passphrase)
	return s.unlockWith(func(v *vault.Vault) error { return v.Unlock(passphrase) })
}

// UnlockKEK is Unlock with a slot key (a systemd credential, an OS key
// store item) instead of a passphrase. It clears kek.
func (s *Server) UnlockKEK(kind vault.SlotKind, kek []byte) (api.VaultView, error) {
	defer clear(kek)
	return s.unlockWith(func(v *vault.Vault) error { return v.UnlockKEK(kind, kek) })
}

// disableCoreDumps is vault.DisableCoreDumps; a test seam.
var disableCoreDumps = vault.DisableCoreDumps

func (s *Server) unlockWith(open func(*vault.Vault) error) (api.VaultView, error) {
	s.vctl.mu.Lock()
	v, err := s.openVaultLocked()
	if err == nil {
		// Before any key material is derived: a vault created or first
		// unlocked after startup must not leave this process dumpable.
		// If that cannot be guaranteed, refuse the unlock.
		if derr := disableCoreDumps(); derr != nil {
			err = fmt.Errorf("vault: refusing to unlock: cannot disable core dumps: %w", derr)
		}
	}
	if err == nil {
		err = open(v)
	}
	if err != nil {
		view := s.viewLocked()
		s.vctl.mu.Unlock()
		return view, err
	}
	if s.cfg.Unlocked == nil {
		s.keys.mu.Lock()
		s.keys.ready = true
		s.keys.mu.Unlock()
	} else {
		sgn, err := s.cfg.Unlocked(v)
		s.keys.mu.Lock()
		s.keys.ready = true
		switch {
		case err != nil:
			// The vault opened but the key in it did not: sign nothing.
			s.keys.sgn, s.keys.fromVault, s.keys.err = nil, true, err
		case sgn != nil:
			s.keys.sgn, s.keys.fromVault, s.keys.err = sgn, true, nil
		}
		s.keys.mu.Unlock()
		if err != nil {
			log.Printf("vault unlocked, but the controller key in it did not open: %v", err)
		}
	}
	first := !s.vctl.unlocked
	s.vctl.unlocked = true
	s.markActivity() // an unlock starts the idle clock afresh
	view := s.viewLocked()
	s.vctl.mu.Unlock()
	s.afterUnlock(first)
	return view, nil
}

// afterUnlock resumes this controller's jobs, brings up autostart overlays
// (the first unlock only), runs Config.AfterUnlock, then refreshes the fleet.
func (s *Server) afterUnlock(first bool) {
	ctx := s.jobs.ctx
	s.jobs.wake()
	s.bg.Add(1)
	go func() {
		defer s.bg.Done()
		s.jobs.sweep(ctx)
		if first {
			for _, r := range s.AutostartOverlays(ctx) {
				if r.Err != nil {
					log.Printf("vpn autostart %s: %v", r.ID, r.Err)
				}
			}
		}
		if s.cfg.AfterUnlock != nil {
			s.cfg.AfterUnlock(ctx)
		}
		s.fleet.reload()
	}()
}

// Lock drops the data key and every secret held in memory, and the
// controller key when the vault held it.
func (s *Server) Lock() api.VaultView {
	s.vctl.mu.Lock()
	defer s.vctl.mu.Unlock()
	return s.lockLocked()
}

// lockLocked is Lock; callers hold s.vctl.mu.
func (s *Server) lockLocked() api.VaultView {
	s.keys.mu.Lock()
	v := s.keys.v
	s.keys.ready = false
	if s.keys.fromVault {
		s.keys.sgn = nil
	}
	s.keys.mu.Unlock()
	if v != nil {
		v.Lock()
	}
	s.fleet.reload()
	return s.viewLocked()
}

// vaultError maps a vault failure to its answer; ok is false for any other
// error.
func vaultError(err error) (status int, e api.Error, ok bool) {
	if err == nil {
		return 0, api.Error{}, false
	}
	msg := err.Error()
	switch {
	case errors.Is(err, vault.ErrLocked):
		return http.StatusServiceUnavailable, api.Error{Message: "the vault is locked", Code: api.CodeLocked}, true
	case errors.Is(err, vault.ErrWrongPassphrase), errors.Is(err, vault.ErrSlotRefused):
		return http.StatusForbidden, api.Error{Message: msg, Code: api.CodeWrongPassphrase}, true
	case errors.Is(err, vault.ErrCorrupt):
		return http.StatusInternalServerError, api.Error{Message: msg, Code: api.CodeVaultCorrupt}, true
	case errors.Is(err, vault.ErrUnsafeFile):
		return http.StatusInternalServerError, api.Error{Message: msg, Code: api.CodeVaultUnsafe}, true
	case errors.Is(err, vault.ErrNoVault):
		return http.StatusConflict, api.Error{Message: msg, Code: api.CodeNoVault}, true
	}
	return 0, api.Error{}, false
}

func writeVaultError(w http.ResponseWriter, err error) {
	if errors.Is(err, errRefValue) {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if status, e, ok := vaultError(err); ok {
		writeAPIError(w, status, e)
		return
	}
	writeError(w, http.StatusInternalServerError, err.Error())
}

func (s *Server) registerVaultRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/vault", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, s.VaultState())
	})
	mux.HandleFunc("POST /api/vault/unlock", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		var req api.UnlockRequest
		// The body is a passphrase: it is never logged or echoed.
		err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxUnlockBody)).Decode(&req)
		defer clear(req.Passphrase)
		if err != nil {
			writeError(w, http.StatusBadRequest, "want {\"passphrase\": base64}")
			return
		}
		if len(req.Passphrase) == 0 {
			writeError(w, http.StatusBadRequest, "the passphrase is empty")
			return
		}
		uid := s.requestUID(r)
		if s.passBackedOff(w, "vault unlock", uid) {
			return
		}
		view, err := s.Unlock(req.Passphrase)
		s.passResult("vault unlock", uid, err)
		if err != nil {
			writeVaultError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, view)
	})
	mux.HandleFunc("POST /api/vault/lock", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, s.Lock())
	})
	mux.HandleFunc("PUT /api/vault/secrets/{name}", s.handlePutVaultSecret)
}

// maxSecretBody bounds PUT /api/vault/secrets/{name}.
const maxSecretBody = 4 << 10

// apiSecretNames are the entries PUT /api/vault/secrets may write: the
// in-process relay's tokens, which no settings field owns. Every other entry
// is written only by the field or command that owns it.
var apiSecretNames = map[string]bool{secrets.NameRelayToken: true, secrets.NameAdminToken: true}

// handlePutVaultSecret is PUT /api/vault/secrets/{name} with
// {"value": base64}. The value is never logged or echoed.
func (s *Server) handlePutVaultSecret(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	name := r.PathValue("name")
	if !apiSecretNames[name] {
		writeError(w, http.StatusBadRequest, "only relay-token and admin-token can be set here")
		return
	}
	var req api.SecretValue
	err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxSecretBody)).Decode(&req)
	defer clear(req.Value)
	if err != nil {
		writeError(w, http.StatusBadRequest, `want {"value": base64}`)
		return
	}
	trimmed := bytes.TrimSpace(req.Value)
	if len(trimmed) == 0 {
		writeError(w, http.StatusBadRequest, "the value is empty")
		return
	}
	if bytes.HasPrefix(trimmed, []byte(config.SecretRefPrefix)) {
		writeVaultError(w, errRefValue)
		return
	}
	if err := s.putAPISecret(name, req.Value); err != nil {
		writeVaultError(w, err)
		return
	}
	// A running relay keeps the token it started with; it is built once,
	// after unlock, and not rebuilt here.
	writeJSON(w, http.StatusOK, map[string]string{"note": TokenRestartNote})
}

// TokenRestartNote is what PUT /api/vault/secrets answers, and what `jumpgate
// vault put` prints: the stored token reaches the relay only at its next start.
const TokenRestartNote = "stored; a relay already running keeps its old token: restart jumpgate to apply"

// putAPISecret stores value as name in the unlocked vault: vault.ErrNoVault
// with no vault file, vault.ErrLocked while locked.
func (s *Server) putAPISecret(name string, value []byte) error {
	s.vctl.mu.Lock()
	defer s.vctl.mu.Unlock()
	v, err := s.openVaultLocked()
	if err != nil {
		return err
	}
	if v.Locked() {
		return vault.ErrLocked
	}
	return v.Put(name, value, "api")
}

// errForeignRef is a field whose vault reference names another field's
// entry (a hand-edited or injected config.json): it is never read or deleted.
var errForeignRef = errors.New("this field's vault reference names another entry; jumpgate reads and deletes only the entry the field owns")

// errRefValue refuses a client value that looks like a vault reference.
var errRefValue = errors.New("a secret value may not start with \"" + config.SecretRefPrefix + "\"")

// ownRef checks that a reference names exactly the entry its field owns
// (want), so no field can reach the controller key, the transport key or
// the tokens.
func ownRef(want, stored string) error {
	if n, ok := config.SecretRefName(stored); ok && n != want {
		return errForeignRef
	}
	return nil
}

// secret is stored itself when it is a literal, or the value its vault:NAME
// reference names, which must be name (the entry the field owns). A vault
// not yet found or locked is vault.ErrLocked.
func (s *Server) secret(name, stored string) (string, error) {
	if !config.IsSecretRef(stored) {
		return stored, nil
	}
	if err := ownRef(name, stored); err != nil {
		return "", err
	}
	s.keys.mu.RLock()
	v := s.keys.v
	s.keys.mu.RUnlock()
	return secrets.Resolve(v, stored)
}

// putSecret stores value: under secretStore "vault" in the vault as name,
// returning the reference config.json keeps; otherwise it returns value, the
// literal config.json keeps.
func (s *Server) putSecret(name, value string) (stored string, err error) {
	if strings.HasPrefix(strings.TrimSpace(value), config.SecretRefPrefix) {
		return "", errRefValue
	}
	c, err := config.Load()
	if err != nil {
		return "", err
	}
	if c.SecretStore != config.SecretStoreVault {
		return value, nil
	}
	s.keys.mu.RLock()
	v := s.keys.v
	s.keys.mu.RUnlock()
	if v == nil {
		return "", vault.ErrLocked
	}
	raw := []byte(value)
	defer clear(raw)
	if err := v.Put(name, raw, "settings"); err != nil {
		return "", err
	}
	return config.SecretRef(name), nil
}

// dropSecret deletes the vault entry a reference names, which must be name;
// a literal, or an entry already gone, is nothing to do.
func (s *Server) dropSecret(name, stored string) error {
	if _, ok := config.SecretRefName(stored); !ok {
		return nil
	}
	if err := ownRef(name, stored); err != nil {
		return err
	}
	s.keys.mu.RLock()
	v := s.keys.v
	s.keys.mu.RUnlock()
	if v == nil {
		return vault.ErrLocked
	}
	if err := v.Delete(name); err != nil && !errors.Is(err, vault.ErrNotFound) {
		return err
	}
	return nil
}

// plainSecrets names the secret fields config.json still holds as plain
// values.
func (s *Server) plainSecrets() []string {
	c, err := config.Load()
	if err != nil {
		return nil
	}
	return c.PlainSecrets()
}
