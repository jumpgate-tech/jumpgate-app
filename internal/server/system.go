package server

import (
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/executor"
)

// The jumpgate service (serve --system) runs as jumpgate-ctl with
// ProtectHome=yes and admits every jumpgate-admin member. It never reads a
// file a client names: an SSH key path would make it a confused deputy for
// its own state, which a member cannot read. It reaches boxes only through
// their agents, over its own transport key; every step that needs an
// operator's SSH login runs in the operator's own CLI process.

// errServiceNoLogin is what the service's executor seam answers for an ssh
// target: it never logs in to a box itself.
var errServiceNoLogin = errors.New("the jumpgate service does not log in to boxes over SSH")

// systemExecutor wraps next so that only local targets get an executor.
func systemExecutor(next func(config.Target) (executor.Executor, error)) func(config.Target) (executor.Executor, error) {
	return func(t config.Target) (executor.Executor, error) {
		if t.Mode == "ssh" {
			return nil, fmt.Errorf("target %q: %w", t.ID, errServiceNoLogin)
		}
		return next(t)
	}
}

// clientKeyPath reports whether any hop of c names a key file.
func clientKeyPath(c *executor.SSHConfig) bool {
	for h := c; h != nil; h = h.Jump {
		if h.KeyPath != "" {
			return true
		}
	}
	return false
}

// handlePairingInfo answers what a CLI needs to pair a box itself: the
// controller address, the public transport key, and in system mode the
// service's uid (the one a self-paired agent enrolls).
func (s *Server) handlePairingInfo(w http.ResponseWriter, r *http.Request) {
	sgn, serr := s.signer()
	if sgn == nil {
		writeNoControllerKey(w, serr)
		return
	}
	cfg, err := s.loadConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	info := api.PairingInfo{Controller: sgn.Address().Hex(), System: s.cfg.System, ServiceUID: -1}
	if cfg.Controller != nil && cfg.Controller.Address != "" {
		info.Controller = cfg.Controller.Address
	}
	if s.cfg.System {
		info.ServiceUID = os.Getuid()
	}
	if id := r.URL.Query().Get("target"); id != "" {
		t, ok := findTarget(cfg, id)
		if !ok {
			writeTargetNotFound(w)
			return
		}
		info.Wire = t.Wire
	}
	if info.TransportKey, err = s.ensureTransportKey(); err != nil {
		writeNodeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, info)
}
