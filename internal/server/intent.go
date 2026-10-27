package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/intent"
	"github.com/valve-tech/jumpgate/internal/monitor"
)

// maxIntentBody bounds an intent payload, matching the agent's own limit.
const maxIntentBody = 1 << 20

// refHeadClient fetches the public reference head for status.read. It is
// bounded so a slow public endpoint cannot hold an intent's answer hostage.
var refHeadClient = &http.Client{Timeout: 5 * time.Second}

// writeNoControllerKey answers a box route on a server with no signer, with
// the reason the key did not open (noControllerKeyError), or 503 locked
// while the vault holding it is locked.
func writeNoControllerKey(w http.ResponseWriter, err error) {
	if status, e, ok := vaultError(err); ok {
		writeAPIError(w, status, e)
		return
	}
	writeAPIError(w, http.StatusServiceUnavailable, noControllerKeyError(err))
}

// jobKindHint names the routes and commands that run jobs, for a job kind
// sent to the generic intent route.
const jobKindHint = "jobs run through the /api/jobs/... routes: use `jumpgate jobs` to watch or decide one and `jumpgate migrate-names HOST` to start a migration"

// isJobKind reports whether kind starts or acts on a durable job. Every
// job.* kind counts, including ones added later, so a new job kind never
// becomes reachable through the generic intent route by accident.
func isJobKind(kind string) bool {
	return jobStartKinds[kind] || strings.HasPrefix(kind, "job.")
}

// handleIntent signs one intent with the controller key, sends it to the
// target's agent and returns the verified answer. Keys are only ever loaded
// by the server process; this route is why it must sit behind the Origin
// check, since it makes the server sign.
func (s *Server) handleIntent(w http.ResponseWriter, r *http.Request) {
	if sg, err := s.signer(); sg == nil {
		writeNoControllerKey(w, err)
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
	if _, err := s.agentTarget(t); err != nil {
		if status, e, ok := vaultError(err); ok {
			writeAPIError(w, status, e)
			return
		}
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	kind := r.PathValue("kind")
	if isJobKind(kind) {
		// Refused before anything is read or signed: a job start here would
		// skip the P7 allowlist and the claim that keeps a job from starting
		// while setup, a clear or another op runs on this box, and a job.*
		// kind would act on a job outside the lease its driver holds.
		writeErrorDetail(w, http.StatusBadRequest, fmt.Sprintf("%q is a job intent and is not sent through the generic intent route", kind),
			jobKindHint, api.CodeJobKindNotAllowed)
		return
	}
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
		payload = firewallPayload(cfg)
	case intent.KindEndpointsRead:
		payload = intent.EndpointsReadPayload{SSHLogin: sshLogin(t)}
	}

	res, err := s.sendIntent(r.Context(), cfg, t, kind, payload)
	if err != nil {
		writeNodeError(w, err)
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
