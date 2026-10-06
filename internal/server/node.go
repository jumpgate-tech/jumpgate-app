package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/valve-tech/jumpgate/internal/agentclient"
	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/executor"
	"github.com/valve-tech/jumpgate/internal/intent"
	"github.com/valve-tech/jumpgate/internal/logwatch"
	"github.com/valve-tech/jumpgate/internal/monitor"
	"github.com/valve-tech/jumpgate/internal/ops"
	"github.com/valve-tech/jumpgate/internal/signer"
)

// via is the transport a node operation used. Clients see it as the
// X-Jumpgate-Via header; the operation itself is the same either way. "ssh"
// names the legacy executor, which for a local target is this machine's shell.
type via string

const (
	viaAgent via = "agent"
	viaSSH   via = "ssh"
)

func setVia(w http.ResponseWriter, v via) {
	if v != "" {
		w.Header().Set("X-Jumpgate-Via", string(v))
	}
}

// Errors a node operation returns besides the agent client's own.
var (
	errNotSetUp     = errors.New("target has not completed setup")
	errNoController = errors.New("this server has no controller key")
)

// noControllerKey is errNoController with the reason the key did not open
// (Config.SignerErr, nil when no key is recorded), so a node route says what
// the intent route says (Ruling T5a).
type noControllerKey struct{ cause error }

func (e *noControllerKey) Error() string        { return noControllerKeyError(e.cause).Message }
func (e *noControllerKey) Unwrap() error        { return e.cause }
func (e *noControllerKey) Is(target error) bool { return target == errNoController }

// noControllerKeyError is the answer for a server with no signer. When a key
// is recorded but would not open, the reason is in the message and the hint
// is to fix the key store, not to create a second key. Only the plain "no
// key" case takes the registry's hint.
func noControllerKeyError(signerErr error) api.Error {
	switch {
	case errors.Is(signerErr, signer.ErrAddressMismatch):
		return api.Error{Message: "the controller key is not the recorded controller identity: " + signerErr.Error(),
			Hint: "restore the original key in its key store, then restart the server with `jumpgate stop`; `jumpgate keys show` prints both addresses",
			Code: api.CodeControllerKeyMismatch}
	case signerErr != nil:
		return api.Error{Message: "this server could not open the controller key: " + signerErr.Error(),
			Hint: "fix the key store (unlock the keychain, sign in to 1Password, restore the key file), then restart the server with `jumpgate stop`",
			Code: api.CodeNoControllerKey}
	}
	return api.Error{Message: errNoController.Error(), Code: api.CodeNoControllerKey}
}

// agentRejected is a signed rejection; agentFailed a signed failure.
type agentRejected struct{ intent.Rejection }

func (e *agentRejected) Error() string { return "the box refused: " + e.Message }

type agentFailed struct{ msg string }

func (e *agentFailed) Error() string { return e.msg }

// localError is a failure on this machine while sending an intent (signing,
// the sequence store, a pairing record that does not resolve): the server's
// own fault, not the box's, so 500 internal.
type localError struct{ err error }

func (e *localError) Error() string { return e.err.Error() }
func (e *localError) Unwrap() error { return e.err }

// isAgentErr reports whether err is one of agentclient's transport,
// verification or host-key errors, as opposed to a local failure.
func isAgentErr(err error) bool {
	return errors.Is(err, agentclient.ErrUnreachable) || errors.Is(err, agentclient.ErrAgentHTTP) ||
		errors.Is(err, agentclient.ErrBadReceipt) || errors.Is(err, executor.ErrUnknownHost) ||
		errors.Is(err, executor.ErrHostKeyMismatch)
}

// dialError marks a failure to open a legacy executor, so a refused
// connection reads "unreachable" while an operation's own failure does not.
type dialError struct{ err error }

func (e *dialError) Error() string { return e.err.Error() }
func (e *dialError) Unwrap() error { return e.err }

// sendIntent signs one intent for a paired target and returns the verified
// answer. It is the only way the server reaches an agent: one intent per
// target at a time, because the sequence lives in config.
func (s *Server) sendIntent(ctx context.Context, cfg config.Config, t config.Target, kind string, payload any) (agentclient.Response, error) {
	if s.cfg.Signer == nil {
		return agentclient.Response{}, &noControllerKey{s.cfg.SignerErr}
	}
	at, err := agentTarget(t)
	if err != nil {
		return agentclient.Response{}, &localError{err}
	}
	entry := s.reg.get(t.ID)
	entry.intentMu.Lock()
	defer entry.intentMu.Unlock()
	client, err := agentclient.Dial(ctx, at, s.cfg.Signer, configSeqs{targetID: t.ID})
	if err != nil {
		// A host-key failure keeps its type; every other dial failure is an
		// outage of the transport, never a reason to try another one.
		if !isAgentErr(err) {
			err = fmt.Errorf("%w: %v", agentclient.ErrUnreachable, err)
		}
		return agentclient.Response{}, err
	}
	defer client.Close()
	res, err := client.Do(ctx, kind, payload)
	if err != nil && !isAgentErr(err) {
		err = &localError{err}
	}
	return res, err
}

// agentResult is sendIntent's result, with a rejection or failure as an error.
func (s *Server) agentResult(ctx context.Context, cfg config.Config, t config.Target, kind string, payload any) (json.RawMessage, error) {
	res, err := s.sendIntent(ctx, cfg, t, kind, payload)
	switch {
	case err != nil:
		return nil, err
	case res.Rejection != nil:
		return nil, &agentRejected{*res.Rejection}
	case res.Failure != nil:
		return nil, &agentFailed{res.Failure.Message}
	}
	return res.Result, nil
}

func decodeResult[T any](raw json.RawMessage, err error) (T, error) {
	var v T
	if err != nil {
		return v, err
	}
	if err := json.Unmarshal(raw, &v); err != nil {
		return v, fmt.Errorf("the agent's answer does not parse: %w", err)
	}
	return v, nil
}

// legacyExec is the executor for an unpaired target that has been set up. It
// is the server's cached executor, so its host-key policy is legacySSHConfig's.
func (s *Server) legacyExec(t config.Target) (executor.Executor, error) {
	if t.Wire == nil {
		return nil, errNotSetUp
	}
	ex, err := s.getExecutor(t)
	if err != nil {
		return nil, &dialError{err}
	}
	return ex, nil
}

// sshLogin is the user@host an operator would ssh to, for the endpoints
// tunnel hint; "" for a target that is not reached over SSH (Ruling T5b: the
// one login rule, for the node route and /intent alike).
func sshLogin(t config.Target) string {
	if t.Mode == "ssh" && t.SSH != nil {
		return t.SSH.User + "@" + t.SSH.Host
	}
	return ""
}

// The node operations. Each picks the agent for a paired target and the
// legacy executor otherwise; the routes, the fleet poller and explain all
// call these, never ops or agentclient directly. A paired target never falls
// back to the legacy executor: when its agent cannot answer, the operation
// fails with the agent's error.

func (s *Server) nodeStatus(ctx context.Context, cfg config.Config, t config.Target) (monitor.Snapshot, via, error) {
	if t.Agent != nil {
		snap, err := decodeResult[monitor.Snapshot](s.agentResult(ctx, cfg, t, intent.KindStatusRead, struct{}{}))
		if err == nil && cfg.RefRPCBase != "" && t.Wire != nil {
			snap.RefHead = monitor.FetchRefHead(ctx, refHeadClient, refRPCURL(cfg.RefRPCBase, t.Wire.ChainID))
		}
		return snap, viaAgent, err
	}
	if t.Wire == nil {
		return monitor.Snapshot{}, viaSSH, errNotSetUp
	}
	mon, _, err := s.getMonitor(t, cfg.RefRPCBase)
	if err != nil {
		return monitor.Snapshot{}, viaSSH, &dialError{err}
	}
	return mon.Latest(), viaSSH, nil
}

func (s *Server) nodeDisk(ctx context.Context, cfg config.Config, t config.Target) (ops.DU, via, error) {
	if t.Agent != nil {
		// The agent answers {usage, freeBytes}; the route's shape is ops.DU.
		r, err := decodeResult[struct {
			Usage     ops.DU `json:"usage"`
			FreeBytes uint64 `json:"freeBytes"`
		}](s.agentResult(ctx, cfg, t, intent.KindDiskRead, struct{}{}))
		if r.Usage.DiskFreeBytes == 0 {
			r.Usage.DiskFreeBytes = r.FreeBytes
			// An agent that predates DiskFreeKnown still reports freeBytes.
			r.Usage.DiskFreeKnown = r.Usage.DiskFreeKnown || r.FreeBytes > 0
		}
		return r.Usage, viaAgent, err
	}
	ex, err := s.legacyExec(t)
	if err != nil {
		return ops.DU{}, viaSSH, err
	}
	du, err := ops.DiskUsage(ctx, ex, *t.Wire)
	return du, viaSSH, err
}

func (s *Server) nodeEndpoints(ctx context.Context, cfg config.Config, t config.Target) (ops.EndpointInfo, via, error) {
	login := sshLogin(t)
	if t.Agent != nil {
		ep, err := decodeResult[ops.EndpointInfo](s.agentResult(ctx, cfg, t, intent.KindEndpointsRead, intent.EndpointsReadPayload{SSHLogin: login}))
		return ep, viaAgent, err
	}
	ex, err := s.legacyExec(t)
	if err != nil {
		return ops.EndpointInfo{}, viaSSH, err
	}
	ep, err := ops.Endpoints(ctx, ex, *t.Wire, login != "", login)
	return ep, viaSSH, err
}

func (s *Server) nodeFirewall(ctx context.Context, cfg config.Config, t config.Target) ([]ops.CheckItem, via, error) {
	// Grade binds to private overlays (WireGuard, Tailscale, etc.) as overlay
	// rather than LAN: the operator's declared overlays and the subnet of any
	// WireGuard server this app provisioned.
	if t.Agent != nil {
		items, err := decodeResult[[]ops.CheckItem](s.agentResult(ctx, cfg, t, intent.KindFirewallRead, firewallPayload(cfg)))
		return items, viaAgent, err
	}
	ex, err := s.legacyExec(t)
	if err != nil {
		return nil, viaSSH, err
	}
	items, err := ops.FirewallChecklist(ctx, ex, *t.Wire, ops.ParseOverlayCIDRs(cfg.TrustedOverlayCIDRs())...)
	return items, viaSSH, err
}

// firewallPayload is firewall.read's payload: the trusted overlays live in
// the controller's config, not on the box.
func firewallPayload(cfg config.Config) map[string]any {
	return map[string]any{"overlayCidrs": cfg.TrustedOverlayCIDRs()}
}

func (s *Server) nodeLogs(ctx context.Context, cfg config.Config, t config.Target, n int) ([]logwatch.Hit, via, error) {
	if t.Agent != nil {
		hits, err := decodeResult[[]logwatch.Hit](s.agentResult(ctx, cfg, t, intent.KindLogsRead, intent.LogsReadPayload{N: n}))
		return hits, viaAgent, err
	}
	if t.Wire == nil {
		return nil, viaSSH, errNotSetUp
	}
	watch, _, err := s.getWatcher(t)
	if err != nil {
		return nil, viaSSH, &dialError{err}
	}
	return watch.Recent(n), viaSSH, nil
}

func (s *Server) nodeService(ctx context.Context, cfg config.Config, t config.Target, svc, action string) (bool, via, error) {
	if t.Agent != nil {
		r, err := decodeResult[struct {
			Active bool `json:"active"`
		}](s.agentResult(ctx, cfg, t, intent.KindServiceAction, intent.ServiceActionPayload{Service: svc, Action: action}))
		return r.Active, viaAgent, err
	}
	ex, err := s.legacyExec(t)
	if err != nil {
		return false, viaSSH, err
	}
	active, err := ops.ServiceAction(ctx, ex, svc, action)
	return active, viaSSH, err
}

// apiErrorFor maps every error a node operation returns onto the API's
// status and body, with the code's registry hint filled in so a stream's
// "error" event reads like a route's error body. Host-key failures are
// checked before "unreachable": they are security errors, never outages.
func apiErrorFor(err error) (int, api.Error) {
	status, e := classifyNodeError(err)
	if e.Code == "" {
		e.Code = api.CodeForStatus(status)
	}
	if e.Hint == "" {
		e.Hint = api.HintFor(e.Code)
	}
	return status, e
}

func classifyNodeError(err error) (int, api.Error) {
	var (
		rej     *agentRejected
		fail    *agentFailed
		nokey   *noControllerKey
		local   *localError
		dial    *dialError
		unknown *executor.UnknownHostError
		op      *net.OpError
	)
	msg := err.Error()
	switch {
	case errors.As(err, &rej):
		return http.StatusConflict, api.Error{Message: rej.Message, Code: api.CodeRejected, Reason: rej.Code, Hint: api.RejectionHint(rej.Code)}
	case errors.As(err, &fail):
		return http.StatusBadGateway, api.Error{Message: fail.msg, Code: api.CodeAgentFailed}
	case errors.Is(err, errNotSetUp):
		return http.StatusConflict, api.Error{Message: msg, Code: api.CodeTargetNotSetUp}
	case errors.As(err, &nokey):
		return http.StatusServiceUnavailable, noControllerKeyError(nokey.cause)
	case errors.Is(err, errNoController):
		return http.StatusServiceUnavailable, noControllerKeyError(nil)
	case errors.As(err, &unknown):
		return http.StatusConflict, api.Error{Message: msg, Code: api.CodeUnknownHost, Host: unknown.Host, Fingerprint: unknown.Fingerprint}
	case errors.Is(err, executor.ErrHostKeyMismatch):
		return http.StatusBadGateway, api.Error{Message: msg, Code: api.CodeHostKey}
	case errors.Is(err, agentclient.ErrBadReceipt):
		return http.StatusBadGateway, api.Error{Message: msg, Code: api.CodeBadReceipt}
	case errors.Is(err, agentclient.ErrAgentHTTP):
		return http.StatusBadGateway, api.Error{Message: msg, Code: api.CodeAgentHTTP}
	case errors.Is(err, agentclient.ErrUnreachable):
		return http.StatusGatewayTimeout, api.Error{Message: msg, Code: api.CodeUnreachable}
	case errors.As(err, &local):
		return http.StatusInternalServerError, api.Error{Message: msg, Code: api.CodeInternal}
	case errors.Is(err, executor.ErrNoPOSIXShell):
		return http.StatusConflict, api.Error{Message: msg, Code: api.CodeLocalUnsupported}
	case errors.As(err, &dial) && (errors.As(err, &op) || errors.Is(err, context.DeadlineExceeded) || errors.Is(err, os.ErrDeadlineExceeded)):
		return http.StatusGatewayTimeout, api.Error{Message: msg, Code: api.CodeUnreachable}
	}
	return http.StatusBadGateway, api.Error{Message: msg}
}

func writeNodeError(w http.ResponseWriter, err error) {
	status, e := apiErrorFor(err)
	writeAPIError(w, status, e)
}

// nodeTarget resolves a node route's target. Unlike targetWithWire it does
// not require a controller-side wire: a paired box's agent has its own.
func (s *Server) nodeTarget(w http.ResponseWriter, id string) (config.Config, config.Target, bool) {
	cfg, err := s.loadConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return cfg, config.Target{}, false
	}
	t, ok := findTarget(cfg, id)
	if !ok {
		writeTargetNotFound(w)
		return cfg, t, false
	}
	return cfg, t, true
}

// Stream cadences for paired boxes (variables so tests can shorten them).
var (
	agentStatusInterval       = 5 * time.Second
	agentLogsSnapshotInterval = 10 * time.Second
	// agentPollTimeout bounds one poll, well inside apiclient's 45 s idle
	// timeout, so a slow agent is an error event, not a dead stream.
	agentPollTimeout = 25 * time.Second
)

// writeSSENote sends a notice about the stream itself (apiclient: Update.Note).
func writeSSENote(w http.ResponseWriter, text string) { writeSSENamed(w, "note", text) }

// agentPoll is one poll's outcome: the event to send (name "" is a default
// event) or the error to report.
type agentPoll struct {
	name string
	v    any
	err  error
}

// pollAgentStream runs a paired box's stream: poll now, then again every
// interval. The poll runs in its own goroutine under agentPollTimeout, so the
// handler keeps pinging (Task 3's cadence) while an agent is slow, and a poll
// that runs out of time is an "unreachable" error event, well before the
// client's idle timeout. Any error is sent as an "error" event and the stream
// continues, so a box that comes back is picked up again. Only the handler
// goroutine writes to the stream; the poll goroutine only returns a value, and
// the handler waits for it before returning or polling again.
func pollAgentStream(r *http.Request, conn *sseConn, interval time.Duration, poll func(ctx context.Context) agentPoll) {
	tick := time.NewTicker(interval)
	defer tick.Stop()
	for {
		ctx, cancel := context.WithTimeout(r.Context(), agentPollTimeout)
		done := make(chan agentPoll, 1)
		go func() { done <- poll(ctx) }()
		res, timedOut := agentPoll{}, false
		for waiting := true; waiting; {
			select {
			case res = <-done:
				waiting = false
			case <-ctx.Done():
				if r.Context().Err() != nil {
					cancel()
					<-done
					return
				}
				if !timedOut {
					// The poll may still be waiting on something that does
					// not watch ctx (the target's intent lock); say so now
					// and drop whatever it answers later.
					timedOut = true
					_, e := apiErrorFor(fmt.Errorf("%w: the agent did not answer within %s", agentclient.ErrUnreachable, agentPollTimeout))
					conn.SendNamed("error", e)
				}
			case <-conn.Pings():
				conn.Ping()
			}
		}
		cancel()
		switch {
		case timedOut:
		case res.err != nil:
			_, e := apiErrorFor(res.err)
			conn.SendNamed("error", e)
		case res.name == "":
			conn.Send(res.v)
		default:
			conn.SendNamed(res.name, res.v)
		}
		for waiting := true; waiting; {
			select {
			case <-r.Context().Done():
				return
			case <-conn.Pings():
				conn.Ping()
			case <-tick.C:
				waiting = false
			}
		}
	}
}

// streamAgentStatus serves a paired box's monitor stream by sending
// status.read every agentStatusInterval.
func (s *Server) streamAgentStatus(w http.ResponseWriter, r *http.Request, cfg config.Config, t config.Target) {
	setVia(w, viaAgent)
	conn, ok := startSSE(w)
	if !ok {
		return
	}
	defer conn.Close()
	pollAgentStream(r, conn, agentStatusInterval, func(ctx context.Context) agentPoll {
		snap, _, err := s.nodeStatus(ctx, cfg, t)
		return agentPoll{v: snap, err: err}
	})
}

// streamAgentLogSnapshots serves a paired box's logs stream on conn as a
// fresh window of the newest lines every agentLogsSnapshotInterval, each a
// "reset", so a snapshot never duplicates or loses a line the way a diff of
// windows would. It is the stream for an agent that predates logs.since, and
// opens with note, which says so. rawBacklog keeps Task 3's meaning as far as
// snapshots can: absent, each window is defaultRecentLogs lines; n, it is n
// lines; 0, the first reset is empty and later windows are the default size.
func (s *Server) streamAgentLogSnapshots(r *http.Request, conn *sseConn, cfg config.Config, t config.Target, rawBacklog string, note string) {
	writeSSENote(conn.w, note)
	conn.f.Flush()
	n, empty := defaultRecentLogs, false
	if rawBacklog != "" {
		if b := backlogParam(rawBacklog); b > 0 {
			n = b
		} else {
			empty = true
		}
	}
	pollAgentStream(r, conn, agentLogsSnapshotInterval, func(ctx context.Context) agentPoll {
		if empty {
			// Polls run one at a time, so this needs no lock.
			empty = false
			return agentPoll{name: "reset", v: []logwatch.Hit{}}
		}
		hits, _, err := s.nodeLogs(ctx, cfg, t, n)
		if hits == nil {
			hits = []logwatch.Hit{}
		}
		return agentPoll{name: "reset", v: hits, err: err}
	})
}
