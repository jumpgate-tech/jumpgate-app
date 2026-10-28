package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/fsperm"
	"github.com/valve-tech/jumpgate/internal/peercred"
	"github.com/valve-tech/jumpgate/internal/signer"
	"github.com/valve-tech/jumpgate/internal/vault"
)

// maxInitKeyBody bounds POST /api/system/init-key.
const maxInitKeyBody = 8 << 10

// vaultKeyRef is the vault entry that holds the controller key.
const vaultKeyRef = "controller-key"

// SetRestart sets Config.Restart on a built server, before it serves.
func (s *Server) SetRestart(f func()) { s.cfg.Restart = f }

func (s *Server) registerSystemRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/system/init-key", s.handleSystemInitKey)
	mux.HandleFunc("POST /api/system/seal", s.handleSystemSeal)
	mux.HandleFunc("POST /api/restart", func(w http.ResponseWriter, r *http.Request) {
		if s.cfg.Restart == nil {
			writeError(w, http.StatusNotFound, "only the jumpgate service restarts over the API")
			return
		}
		w.WriteHeader(http.StatusAccepted)
		go s.cfg.Restart()
	})
}

// socketPeer is the admin-socket peer of r. It answers 404 and false for a
// server that is not the jumpgate service, or a request that came over TCP.
func (s *Server) socketPeer(w http.ResponseWriter, r *http.Request) (peercred.Cred, bool) {
	c, ok := peerFrom(r.Context())
	if !s.cfg.System || !ok {
		writeError(w, http.StatusNotFound, "not found")
		return peercred.Cred{}, false
	}
	return c, true
}

// handleSystemInitKey is POST /api/system/init-key: `jumpgate setup server`
// has the service create its own controller key, which the operator never
// holds. The vault store creates the vault with the operator's passphrase
// and leaves it locked, so the operator then unlocks through the normal path.
func (s *Server) handleSystemInitKey(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.socketPeer(w, r); !ok {
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	var req api.SystemInitKey
	// The body holds a passphrase: it is never logged or echoed.
	err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxInitKeyBody)).Decode(&req)
	defer clear(req.Passphrase)
	if err != nil {
		writeError(w, http.StatusBadRequest, `want {"store", "passphrase": base64, …}`)
		return
	}
	if req.Scope != "" && req.Scope != "self" {
		writeError(w, http.StatusBadRequest, `scope must be "" or "self"`)
		return
	}
	if req.Store == "" {
		req.Store = string(signer.StoreVault)
	}
	store := signer.Store(req.Store)
	switch store {
	case signer.StoreVault:
		if req.Ref != "" && req.Ref != vaultKeyRef {
			writeError(w, http.StatusBadRequest, "the vault key's name is always "+vaultKeyRef+"; leave out --ref")
			return
		}
		if len(req.Passphrase) == 0 {
			writeError(w, http.StatusBadRequest, "the passphrase is empty")
			return
		}
		req.Ref = vaultKeyRef
	case signer.StoreOnePassword:
		if !strings.HasPrefix(req.Ref, "op://") {
			writeError(w, http.StatusBadRequest, "--ref op://<vault>/<item>/<field> is required for 1password")
			return
		}
	case signer.StoreFile:
		if !req.InsecurePlain {
			writeError(w, http.StatusBadRequest, "--store file keeps the key unencrypted on disk; add --insecure-plain-key to confirm, or use --store vault")
			return
		}
		// Review Focus 5: the service never writes, or reads, a path a
		// client chose.
		if req.Ref != "" {
			writeError(w, http.StatusBadRequest, "the jumpgate service keeps a file key at its own path; leave out --ref")
			return
		}
		dir, err := config.Dir()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		req.Ref = filepath.Join(dir, "keys", "controller.key")
	default:
		writeError(w, http.StatusBadRequest, fmt.Sprintf("unknown store %q: want vault, 1password or file", req.Store))
		return
	}
	if cfg, err := s.loadConfig(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	} else if cfg.Controller != nil {
		writeControllerExists(w, cfg.Controller.Address)
		return
	}

	var k *signer.Key
	if store == signer.StoreVault {
		k, err = s.createVaultKey(req.Passphrase)
	} else {
		k, err = signer.Create(r.Context(), store, req.Ref)
	}
	if errors.Is(err, signer.ErrKeyExists) || errors.Is(err, fs.ErrExist) {
		writeControllerExists(w, "")
		return
	}
	if err != nil {
		writeVaultError(w, err)
		return
	}
	if _, err := s.updateConfig(func(c *config.Config) error {
		if c.Controller != nil {
			return fmt.Errorf("a controller key was recorded while this one was being created (%s)", c.Controller.Address)
		}
		c.Controller = &config.Controller{KeyStore: string(store), KeyRef: req.Ref, Address: k.Address().Hex(), Scope: req.Scope}
		switch {
		case store == signer.StoreVault && len(c.PlainSecrets()) == 0:
			c.SecretStore = config.SecretStoreVault
		case store == signer.StoreFile:
			c.SecretStore = config.SecretStoreFile
		}
		return nil
	}); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("the key %s was created but could not be recorded: %v", k.Address().Hex(), err))
		return
	}
	if store != signer.StoreVault {
		s.keys.mu.Lock()
		s.keys.sgn, s.keys.fromVault, s.keys.err = k, false, nil
		s.keys.mu.Unlock()
	}
	addr := k.Address().Hex()
	writeJSON(w, http.StatusCreated, api.ControllerView{Recorded: addr, Address: addr, Store: string(store)})
}

func writeControllerExists(w http.ResponseWriter, addr string) {
	msg := "this controller already has a key"
	if addr != "" {
		msg += " (" + addr + ")"
	}
	writeErrorDetail(w, http.StatusConflict, msg+"; jumpgate never replaces one", "", api.CodeControllerExists)
}

// createVaultKey puts a new controller key in the vault. With no vault yet
// it creates one with passphrase and locks it again; an existing vault must
// be unlocked. It clears passphrase.
func (s *Server) createVaultKey(passphrase []byte) (*signer.Key, error) {
	defer clear(passphrase)
	s.vctl.mu.Lock()
	defer s.vctl.mu.Unlock()
	v, err := s.openVaultLocked()
	if err == nil {
		if v.Locked() {
			return nil, vault.ErrLocked
		}
		k, err := signer.CreateVault(v, vaultKeyRef)
		if err != nil {
			return nil, err
		}
		s.keys.mu.Lock()
		s.keys.sgn, s.keys.fromVault, s.keys.ready, s.keys.err = k, true, true, nil
		s.keys.mu.Unlock()
		return k, nil
	}
	if !errors.Is(err, vault.ErrNoVault) {
		return nil, err
	}
	p, err := s.vaultPath()
	if err != nil {
		return nil, err
	}
	if err := disableCoreDumps(); err != nil {
		return nil, fmt.Errorf("vault: refusing to create: cannot disable core dumps: %w", err)
	}
	if err := fsperm.MkdirPrivate(filepath.Dir(p)); err != nil {
		return nil, err
	}
	nv, err := vault.Create(p, passphrase, vault.DefaultParams) // clears passphrase
	if err != nil {
		return nil, err
	}
	// The server finds the file on its next look (S8) and reads it locked.
	defer nv.Lock()
	return signer.CreateVault(nv, vaultKeyRef)
}
