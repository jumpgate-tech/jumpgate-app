package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/valve-tech/jumpgate/internal/agentclient"
	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/bootstrap"
	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/executor"
	"github.com/valve-tech/jumpgate/internal/intent"
)

type pairRequest struct {
	Sudo bool `json:"sudo"`
}

// handlePair installs and pairs the target's agent, streaming each step, then
// proves the pairing with a signed agent.info round trip before recording it.
// The host key must already be confirmed (the CLI does that with the
// operator); an unconfirmed host is refused before anything runs on it.
func (s *Server) handlePair(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Signer == nil {
		s.writeNoControllerKey(w)
		return
	}
	var req pairRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 4096)).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
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
	local := t.Mode == "local"
	if !local && t.SSH == nil {
		writeError(w, http.StatusBadRequest, "this target has no SSH address")
		return
	}

	// Pairing runs root commands on the box, so it takes turns with setup
	// runs, wipes and clears on the same target.
	release, ok := s.claimTargetOp(w, t.ID)
	if !ok {
		return
	}
	defer release()

	// A closed tab must not abort a half-done pairing; shutdown waits for it.
	ctx, done := s.criticalOp(r)
	defer done()

	var priv executor.Executor
	if local {
		priv = executor.Sudo(s.newLocalExecutor())
	} else {
		hostKey, algos, err := config.StrictHostKey()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		login := *t.SSH
		login.HostKey, login.HostKeyAlgorithms = hostKey, algos
		ex, err := executor.NewSSHContext(ctx, login)
		var unknown *executor.UnknownHostError
		if errors.As(err, &unknown) {
			writeAPIError(w, http.StatusConflict, api.Error{Message: err.Error(), Hint: "confirm this fingerprint against the box's console, then pair again", Code: api.CodeUnknownHost, Host: unknown.Host, Fingerprint: unknown.Fingerprint})
			return
		}
		if errors.Is(err, executor.ErrHostKeyMismatch) {
			writeErrorDetail(w, http.StatusBadGateway, err.Error(), "", api.CodeHostKey)
			return
		}
		if err != nil {
			writeErrorDetail(w, http.StatusGatewayTimeout, err.Error(), "check the address, user and key", api.CodeUnreachable)
			return
		}
		priv = ex
		if req.Sudo || t.SSH.User != "root" {
			priv = executor.Sudo(ex)
		}
	}
	defer priv.Close()

	sseHeaders(w)
	flusher, _ := w.(http.Flusher)
	send := func(ev api.PairEvent) {
		writeSSEEvent(w, ev)
		if flusher != nil {
			flusher.Flush()
		}
	}

	transportKey := ""
	if !local {
		if transportKey, err = ensureTransportKey(); err != nil {
			send(api.PairEvent{Step: "transport-key", Error: err.Error(), Code: api.CodeTransportKey})
			return
		}
	}
	addr, err := bootstrap.Run(ctx, bootstrap.Options{
		Exec: priv, Local: local, LocalUID: os.Getuid(), AgentBinary: agentBinary,
		Controller: s.cfg.Signer.Address(), ControllerLabel: controllerLabel(), TransportKey: transportKey, Wire: t.Wire,
		Event: func(step, line string) { send(api.PairEvent{Step: step, Line: line}) },
	})
	if err != nil {
		var se *bootstrap.StepError
		step := ""
		if errors.As(err, &se) {
			step = se.Step
		}
		send(api.PairEvent{Step: step, Error: err.Error(), Code: api.CodeStepFailed})
		return
	}

	transport := "ssh"
	if local {
		transport = "local"
	}
	send(api.PairEvent{Step: "verify", Line: "start"})
	t.Agent = &config.AgentPairing{Address: addr.Hex(), Transport: transport}
	next, ev := s.verifyPairing(ctx, t)
	if ev != nil {
		send(*ev)
		return
	}

	if _, err := s.updateConfig(func(c *config.Config) error {
		for i := range c.Targets {
			if c.Targets[i].ID == t.ID {
				c.Targets[i].Agent = &config.AgentPairing{Address: addr.Hex(), Transport: transport, PairedAt: time.Now().UTC(), NextSeq: next}
				return nil
			}
		}
		return errors.New("target disappeared during pairing")
	}); err != nil {
		send(api.PairEvent{Step: "record", Error: err.Error(), Code: api.CodeRecordFailed})
		return
	}
	writeSSEEvent(w, api.PairEvent{Done: true, Agent: addr.Hex()})
	if flusher != nil {
		flusher.Flush()
	}
}

// verifyPairing makes one signed agent.info round trip to t's freshly
// installed agent and returns the sequence the next intent should use. On a
// re-pair the agent may already hold a higher sequence for this controller;
// the client resynchronises from its signed answer, and the returned value
// follows it.
func (s *Server) verifyPairing(ctx context.Context, t config.Target) (uint64, *api.PairEvent) {
	fail := func(err error, code api.Code, hint string) (uint64, *api.PairEvent) {
		return 0, &api.PairEvent{Step: "verify", Error: err.Error(), Code: code, Hint: hint}
	}
	at, err := agentTarget(t)
	if err != nil {
		return fail(err, api.CodeVerifyFailed, "")
	}
	seqs := agentclient.NewMemorySeqStore()
	client, err := agentclient.Dial(ctx, at, s.cfg.Signer, seqs)
	switch {
	case errors.Is(err, executor.ErrUnknownHost):
		return fail(err, api.CodeUnknownHost, api.HintFor(api.CodeUnknownHost))
	case errors.Is(err, executor.ErrHostKeyMismatch):
		return fail(err, api.CodeHostKey, api.HintFor(api.CodeHostKey))
	case err != nil:
		return fail(err, api.CodeUnreachable, api.HintFor(api.CodeUnreachable))
	}
	defer client.Close()
	res, err := client.Do(ctx, intent.KindAgentInfo, struct{}{})
	switch {
	case errors.Is(err, agentclient.ErrBadReceipt):
		return fail(err, api.CodeBadReceipt, api.HintFor(api.CodeBadReceipt))
	case errors.Is(err, agentclient.ErrAgentHTTP):
		return fail(err, api.CodeAgentHTTP, api.HintFor(api.CodeAgentHTTP))
	case errors.Is(err, agentclient.ErrUnreachable):
		return fail(err, api.CodeUnreachable, api.HintFor(api.CodeUnreachable))
	case err != nil:
		return fail(err, api.CodeVerifyFailed, "")
	case res.Status != intent.StatusOK:
		return fail(fmt.Errorf("agent refused agent.info: %s", res.Result), api.CodeRejected, "")
	}
	next, err := seqs.Next(at.Agent)
	if err != nil {
		return fail(err, api.CodeVerifyFailed, "")
	}
	return next, nil
}

func controllerLabel() string {
	h, err := os.Hostname()
	if err != nil {
		return "controller"
	}
	return h
}
