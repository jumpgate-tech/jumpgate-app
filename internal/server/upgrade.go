package server

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/valve-tech/jumpgate/internal/agentbin"
	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/buildinfo"
	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/intent"
)

func defaultLoadAgent(arch string) ([]byte, error) {
	b, _, err := agentbin.Load(arch)
	return b, err
}

// Upgrade wait (variables so a test can shorten them).
var (
	upgradeWait     = 30 * time.Second
	upgradeInterval = 500 * time.Millisecond
)

// handleAgentUpgrade is POST /api/targets/{id}/agent/upgrade: it sends this
// controller's gate binary for the box's arch as a signed agent.upgrade, then
// waits for the box to answer agent.info on the new version.
func (s *Server) handleAgentUpgrade(w http.ResponseWriter, r *http.Request) {
	if sg, err := s.signer(); sg == nil {
		writeNoControllerKey(w, err)
		return
	}
	cfg, t, ok := s.nodeTarget(w, r.PathValue("id"))
	if !ok {
		return
	}
	if t.Agent == nil {
		writeErrorDetail(w, http.StatusConflict, "this target has no paired gate", "", api.CodeNotPaired)
		return
	}
	release, ok := s.claimTargetOp(w, t.ID)
	if !ok {
		return
	}
	defer release()
	ctx, done := s.criticalOp(r)
	defer done()

	info, err := decodeResult[intent.AgentInfo](s.agentResult(ctx, cfg, t, intent.KindAgentInfo, struct{}{}))
	if err != nil {
		writeNodeError(w, err)
		return
	}
	bin, err := s.loadAgent(info.Arch)
	if err != nil {
		writeErrorDetail(w, http.StatusUnprocessableEntity, err.Error(), "", api.CodeBadRequest)
		return
	}
	sum := sha256.Sum256(bin)
	pl := intent.AgentUpgradePayload{Arch: info.Arch, Version: buildinfo.Version(), SHA256: hex.EncodeToString(sum[:]), Size: int64(len(bin))}
	res, err := s.sendAgent(ctx, t, intent.KindAgentUpgrade, pl, bin)
	switch {
	case err != nil:
		writeNodeError(w, err)
		return
	case res.Rejection != nil:
		writeNodeError(w, &agentRejected{*res.Rejection})
		return
	case res.Failure != nil:
		writeNodeError(w, &agentFailed{res.Failure.Message})
		return
	}
	var out intent.AgentUpgradeResult
	if err := json.Unmarshal(res.Result, &out); err != nil {
		writeError(w, http.StatusBadGateway, "the gate's answer does not parse: "+err.Error())
		return
	}
	if out.Upgraded {
		if err := s.waitAgentVersion(ctx, cfg, t, out.To); err != nil {
			writeErrorDetail(w, http.StatusGatewayTimeout, err.Error(), "", api.CodeTimeout)
			return
		}
	}
	writeJSON(w, http.StatusOK, api.AgentUpgradeView{Upgraded: out.Upgraded, From: out.From, To: out.To})
}

// waitAgentVersion polls agent.info until the box reports version, while the
// gate restarts onto its new binary.
func (s *Server) waitAgentVersion(ctx context.Context, cfg config.Config, t config.Target, version string) error {
	deadline := time.Now().Add(upgradeWait)
	last := ""
	for {
		pctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		info, err := decodeResult[intent.AgentInfo](s.agentResult(pctx, cfg, t, intent.KindAgentInfo, struct{}{}))
		cancel()
		if err == nil {
			if info.Version == version {
				return nil
			}
			last = info.Version
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("the gate did not come back on %s within %s (last seen %q)", version, upgradeWait, last)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(upgradeInterval):
		}
	}
}
