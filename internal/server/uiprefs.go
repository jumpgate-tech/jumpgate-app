// internal/server/uiprefs.go
package server

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/intent"
)

// loadPrefs is the stored preferences with defaults filled in. A block that
// does not parse or validate is replaced by the defaults rather than failing
// every screen.
func loadPrefs(cfg config.Config) api.UIPrefs {
	var p api.UIPrefs
	if len(cfg.UI) > 0 && json.Unmarshal(cfg.UI, &p) == nil {
		if p = p.WithDefaults(); p.Validate() == nil {
			return p
		}
	}
	return api.DefaultPrefs()
}

func (s *Server) handleGetUIPrefs(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.loadConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, loadPrefs(cfg))
}

func (s *Server) handlePutUIPrefs(w http.ResponseWriter, r *http.Request) {
	var p api.UIPrefs
	if err := json.NewDecoder(io.LimitReader(r.Body, 64<<10)).Decode(&p); err != nil {
		writeErrorDetail(w, http.StatusBadRequest, "invalid JSON body", "", api.CodeInvalidPrefs)
		return
	}
	p = p.WithDefaults()
	if err := p.Validate(); err != nil {
		writeErrorDetail(w, http.StatusBadRequest, err.Error(), "", api.CodeInvalidPrefs)
		return
	}
	raw, _ := json.Marshal(p)
	if _, err := s.updateConfig(func(c *config.Config) error { c.UI = raw; return nil }); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.fleet.reload()
	writeJSON(w, http.StatusOK, p)
}

func (s *Server) handleController(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.loadConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	v := api.ControllerView{}
	if cfg.Controller != nil {
		v.Recorded, v.Store = cfg.Controller.Address, cfg.Controller.KeyStore
	}
	switch {
	case cfg.Controller == nil && s.cfg.Signer == nil:
		v.State = "missing"
	case s.cfg.Signer == nil:
		v.State = "unopened"
		v.Reason = "the server started without loading the key; run `jumpgate stop` and try again"
		// Coordination: with the hotfix's Config.SignerErr, use its text here.
	case cfg.Controller == nil:
		v.Address = s.cfg.Signer.Address().Hex()
		v.State = "unrecorded"
		v.Reason = "a key is loaded but no controller identity was recorded; run `jumpgate keys init` to record it"
	default:
		v.Address = s.cfg.Signer.Address().Hex()
		if strings.EqualFold(v.Address, v.Recorded) {
			v.State = "ok"
		} else {
			v.State = "mismatch"
			v.Reason = "the key store holds a different key than the one this controller recorded; boxes will refuse it"
		}
	}
	writeJSON(w, http.StatusOK, v)
}

// handleFleetCheck sends one signed agent.info: "test signing" for a box.
func (s *Server) handleFleetCheck(w http.ResponseWriter, r *http.Request) {
	cfg, t, ok := s.nodeTarget(w, r.PathValue("id"))
	if !ok {
		return
	}
	if t.Agent == nil {
		writeErrorDetail(w, http.StatusConflict, "this target has no paired agent", "", api.CodeNotPaired)
		return
	}
	start := time.Now()
	raw, err := s.agentResult(r.Context(), cfg, t, intent.KindAgentInfo, struct{}{})
	setVia(w, viaAgent)
	if err != nil {
		writeNodeError(w, err)
		return
	}
	var info intent.AgentInfo
	_ = json.Unmarshal(raw, &info)
	writeJSON(w, http.StatusOK, api.AgentCheck{
		Agent: info.Address, Version: info.Version, ChainID: info.ChainID, SetUp: info.SetUp,
		Signers: info.Signers, ElapsedMs: time.Since(start).Milliseconds(),
	})
}

func (s *Server) handleFleetSSH(w http.ResponseWriter, r *http.Request) {
	_, t, ok := s.nodeTarget(w, r.PathValue("id"))
	if !ok {
		return
	}
	if t.Mode != "ssh" || t.SSH == nil {
		writeErrorDetail(w, http.StatusConflict, "this target has no SSH address", "", api.CodeNoSSH)
		return
	}
	confirmed, openssh, err := strictHostFiles()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	argv, err := sshArgv(t, append([]string{confirmed}, openssh...))
	if errors.Is(err, errSSHJump) {
		writeErrorDetail(w, http.StatusUnprocessableEntity, err.Error(), "", api.CodeSSHJumpUnsupported)
		return
	}
	if err != nil {
		writeErrorDetail(w, http.StatusUnprocessableEntity, err.Error(), "fix this host's SSH address, user or jump host in the config", api.CodeBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, api.SSHCommand{Argv: argv, Display: sshDisplay(argv, runtime.GOOS)})
}
