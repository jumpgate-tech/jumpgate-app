package server

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/valve-tech/jumpgate/internal/agentclient"
	"github.com/valve-tech/jumpgate/internal/executor"
	"github.com/valve-tech/jumpgate/internal/intent"
	"github.com/valve-tech/jumpgate/internal/monitor"
)

// maxIntentBody bounds an intent payload, matching the agent's own limit.
const maxIntentBody = 1 << 20

// refHeadClient fetches the public reference head for status.read. It is
// bounded so a slow public endpoint cannot hold an intent's answer hostage.
var refHeadClient = &http.Client{Timeout: 5 * time.Second}

type intentReply struct {
	Status    uint8             `json:"status"`
	Result    json.RawMessage   `json:"result,omitempty"`
	Rejection *intent.Rejection `json:"rejection,omitempty"`
	Failure   *intent.Failure   `json:"failure,omitempty"`
	RefHead   uint64            `json:"refHead,omitempty"` // status.read only
}

// Hints for the agent errors both agent routes report.
const (
	hintUnreachable = "check that the box is up and reachable over SSH"
	hintBadReceipt  = "the answer was not signed by this box's paired agent; do not trust this box until you re-pair it"
	hintUnknownHost = "nobody has confirmed this box's SSH host key; run `jumpgate hosts add` to compare its fingerprint with the box's console and confirm it"
	hintHostKey     = "the box's SSH host key does not match the one on record: possibly a man-in-the-middle, or the box was rebuilt. Check the key on the box's console; only if it legitimately changed, remove the old line from ~/.jumpgate/confirmed_hosts (and ~/.ssh/known_hosts) and confirm the new one with `jumpgate hosts add`"
	hintAgentHTTP   = "the agent socket refused the request before reading it: this connection is not allowed on the socket (the tunnel user is not in the jumpgate group, or a local uid is not enrolled), or the request was too large"
)

// writeAgentError maps an agentclient error onto the API's status and code,
// and reports whether err was one of them. Host-key failures are checked
// first: they are security errors, never "unreachable".
func writeAgentError(w http.ResponseWriter, err error) bool {
	var unknown *executor.UnknownHostError
	switch {
	case errors.As(err, &unknown):
		writeJSON(w, http.StatusConflict, map[string]string{
			"error": err.Error(), "hint": hintUnknownHost, "code": "unknown_host",
			"host": unknown.Host, "fingerprint": unknown.Fingerprint,
		})
	case errors.Is(err, executor.ErrHostKeyMismatch):
		writeErrorDetail(w, http.StatusBadGateway, err.Error(), hintHostKey, "host_key")
	case errors.Is(err, agentclient.ErrBadReceipt):
		writeErrorDetail(w, http.StatusBadGateway, err.Error(), hintBadReceipt, "bad_receipt")
	case errors.Is(err, agentclient.ErrAgentHTTP):
		writeErrorDetail(w, http.StatusBadGateway, err.Error(), hintAgentHTTP, "agent_http")
	case errors.Is(err, agentclient.ErrUnreachable):
		writeErrorDetail(w, http.StatusGatewayTimeout, err.Error(), hintUnreachable, "unreachable")
	default:
		return false
	}
	return true
}

// writeNoControllerKey answers a box route on a server with no signer. When a
// key is recorded but would not open, the reason is in the message and the
// hint is to fix the key store, not to create a second key.
func (s *Server) writeNoControllerKey(w http.ResponseWriter) {
	if err := s.cfg.SignerErr; err != nil {
		writeErrorDetail(w, http.StatusServiceUnavailable, "this server could not open the controller key: "+err.Error(),
			"fix the key store (unlock the keychain, sign in to 1Password, restore the key file), then restart the server with `jumpgate stop`", "no_controller_key")
		return
	}
	writeErrorDetail(w, http.StatusServiceUnavailable, "this server has no controller key", "run `jumpgate keys init`, then restart the server", "no_controller_key")
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
		writeError(w, http.StatusNotFound, "no such target")
		return
	}
	if t.Agent == nil {
		writeErrorDetail(w, http.StatusConflict, "this target has no paired agent", "run `jumpgate hosts add`", "not_paired")
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
			writeErrorDetail(w, http.StatusGatewayTimeout, err.Error(), hintUnreachable, "unreachable")
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
	reply := intentReply{Status: res.Status, Result: res.Result, Rejection: res.Rejection, Failure: res.Failure}
	if kind == intent.KindStatusRead && res.Status == intent.StatusOK && cfg.RefRPCBase != "" && t.Wire != nil {
		reply.RefHead = monitor.FetchRefHead(r.Context(), refHeadClient, refRPCURL(cfg.RefRPCBase, t.Wire.ChainID))
	}
	writeJSON(w, http.StatusOK, reply)
}
