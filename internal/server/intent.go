package server

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/valve-tech/jumpgate/internal/agentclient"
	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/executor"
	"github.com/valve-tech/jumpgate/internal/intent"
	"github.com/valve-tech/jumpgate/internal/monitor"
	"github.com/valve-tech/jumpgate/internal/signer"
)

// maxIntentBody bounds an intent payload, matching the agent's own limit.
const maxIntentBody = 1 << 20

// refHeadClient fetches the public reference head for status.read. It is
// bounded so a slow public endpoint cannot hold an intent's answer hostage.
var refHeadClient = &http.Client{Timeout: 5 * time.Second}

// writeAgentError maps an agentclient error onto the API's status and code,
// and reports whether err was one of them. Host-key failures are checked
// first: they are security errors, never "unreachable". The hints come from
// the api registry, the one place a code's remedy is written.
func writeAgentError(w http.ResponseWriter, err error) bool {
	var unknown *executor.UnknownHostError
	switch {
	case errors.As(err, &unknown):
		writeAPIError(w, http.StatusConflict, api.Error{Message: err.Error(), Code: api.CodeUnknownHost, Host: unknown.Host, Fingerprint: unknown.Fingerprint})
	case errors.Is(err, executor.ErrHostKeyMismatch):
		writeErrorDetail(w, http.StatusBadGateway, err.Error(), "", api.CodeHostKey)
	case errors.Is(err, agentclient.ErrBadReceipt):
		writeErrorDetail(w, http.StatusBadGateway, err.Error(), "", api.CodeBadReceipt)
	case errors.Is(err, agentclient.ErrAgentHTTP):
		writeErrorDetail(w, http.StatusBadGateway, err.Error(), "", api.CodeAgentHTTP)
	case errors.Is(err, agentclient.ErrUnreachable):
		writeErrorDetail(w, http.StatusGatewayTimeout, err.Error(), "", api.CodeUnreachable)
	default:
		return false
	}
	return true
}

// writeNoControllerKey answers a box route on a server with no signer. When a
// key is recorded but would not open, the reason is in the message and the
// hint is to fix the key store, not to create a second key. Only the plain
// "no key" case takes the registry's hint.
func (s *Server) writeNoControllerKey(w http.ResponseWriter) {
	if err := s.cfg.SignerErr; errors.Is(err, signer.ErrAddressMismatch) {
		writeErrorDetail(w, http.StatusServiceUnavailable, "the controller key is not the recorded controller identity: "+err.Error(),
			"restore the original key in its key store, then restart the server with `jumpgate stop`; `jumpgate keys show` prints both addresses", api.CodeControllerKeyMismatch)
		return
	}
	if err := s.cfg.SignerErr; err != nil {
		writeErrorDetail(w, http.StatusServiceUnavailable, "this server could not open the controller key: "+err.Error(),
			"fix the key store (unlock the keychain, sign in to 1Password, restore the key file), then restart the server with `jumpgate stop`", api.CodeNoControllerKey)
		return
	}
	writeErrorDetail(w, http.StatusServiceUnavailable, "this server has no controller key", "", api.CodeNoControllerKey)
}

// handleIntent signs one intent with the controller key, sends it to the
// target's agent and returns the verified answer. Keys are only ever loaded
// by the server process; this route is why it must sit behind the Origin
// check, since it makes the server sign.
func (s *Server) handleIntent(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Signer == nil {
		s.writeNoControllerKey(w)
		return
	}
	cfg, err := s.loadConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	t, ok := findTarget(cfg, r.PathValue("id"))
	if !ok {
		writeTargetNotFound(w)
		return
	}
	if t.Agent == nil {
		writeErrorDetail(w, http.StatusConflict, "this target has no paired agent", "", api.CodeNotPaired)
		return
	}
	at, err := agentTarget(t)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	kind := r.PathValue("kind")
	body, err := io.ReadAll(io.LimitReader(r.Body, maxIntentBody+1))
	if err != nil {
		writeError(w, http.StatusBadRequest, "could not read the body")
		return
	}
	if len(body) > maxIntentBody {
		writeError(w, http.StatusRequestEntityTooLarge, "intent payload is larger than 1 MiB")
		return
	}
	if len(body) == 0 {
		body = []byte("{}")
	}
	var payload any = json.RawMessage(body)
	switch kind {
	case intent.KindFirewallRead:
		payload = map[string]any{"overlayCidrs": cfg.TrustedOverlayCIDRs()}
	case intent.KindEndpointsRead:
		login := ""
		if t.SSH != nil {
			login = t.SSH.User + "@" + t.SSH.Host
		}
		payload = intent.EndpointsReadPayload{SSHLogin: login}
	}

	// One intent per target at a time: the sequence lives in config and each
	// request has its own client, so two in flight could sign the same seq.
	entry := s.reg.get(t.ID)
	entry.intentMu.Lock()
	defer entry.intentMu.Unlock()

	client, err := agentclient.Dial(r.Context(), at, s.cfg.Signer, configSeqs{targetID: t.ID})
	if err != nil {
		if !writeAgentError(w, err) {
			writeErrorDetail(w, http.StatusGatewayTimeout, err.Error(), "", api.CodeUnreachable)
		}
		return
	}
	defer client.Close()
	res, err := client.Do(r.Context(), kind, payload)
	if err != nil {
		if !writeAgentError(w, err) {
			writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	reply := api.IntentReply{Status: res.Status, Result: res.Result, Rejection: res.Rejection, Failure: res.Failure}
	if res.Rejection != nil {
		// The remedy travels with the rejection, so no client keeps its own table.
		reply.Hint = api.RejectionHint(res.Rejection.Code)
	}
	if kind == intent.KindStatusRead && res.Status == intent.StatusOK && cfg.RefRPCBase != "" && t.Wire != nil {
		reply.RefHead = monitor.FetchRefHead(r.Context(), refHeadClient, refRPCURL(cfg.RefRPCBase, t.Wire.ChainID))
	}
	writeJSON(w, http.StatusOK, reply)
}
