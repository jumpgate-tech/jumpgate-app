package server

// The HTTP surface for the DOCKER-BACKED service a target can host: a local
// devnet. Everything here is a thin adapter — the lifecycle itself is
// ops.ServiceStatus / ContainerAction / WipeService, the provisioning is
// setup.PlanDevnet driven through the same setup-run + SSE machinery the node
// wizard uses, and the config is a catalog.DevnetConfig persisted on the
// target.
//
// A devnet genuinely BELONGS to a machine — it is a chain running in a
// container on that box, and it cannot be anywhere else — which is why it is
// addressed under its target. The eRPC gateway used to live here too and no
// longer does: a gateway fronts N chains across M endpoints scattered over
// the whole fleet, so the machine it happens to run on is one field of it,
// not its owner. It has its own top-level surface in gateways.go.
//
// Two things this file does that a pure passthrough would not, both because
// the UI must not have to re-derive them and then drift from what ops will
// actually accept:
//
//   - It computes the ACTIONS each service's current state permits. A stopped
//     container gets no "stop", an absent one gets no "restart", and a target
//     whose docker engine is unreachable gets none at all. An action that can
//     only fail is worse than no action, and the one place that knows which
//     those are is the same place that knows the state.
//
//   - It reports what a WIPE would take down with it — the data it discards
//     and, from the descriptor's FrontedBy, the services it will restart.
//     ops.WipeService cascades that restart for a measured, silent-failure
//     reason (eRPC's monotonic per-network head never notices a chain reset),
//     and a cascade the operator cannot see coming, or confirm afterwards, is
//     the same invisible behaviour in a nicer wrapper.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/valve-tech/valve-node-app/internal/catalog"
	"github.com/valve-tech/valve-node-app/internal/config"
	"github.com/valve-tech/valve-node-app/internal/executor"
	"github.com/valve-tech/valve-node-app/internal/ops"
	"github.com/valve-tech/valve-node-app/internal/setup"
)

// The {svc} path values. They are the ops.DockerService IDs, not display
// names: one identifier across the API, the errors and the reports is what
// makes a wipe report legible next to the status it came from.
const (
	svcDevnet = "devnet"
)

// The actions a container-backed service can be offered, as stable
// identifiers the UI renders verbatim.
const (
	actionStart    = "start"
	actionStop     = "stop"
	actionRestart  = "restart"
	actionCreate   = "create"   // provision from nothing
	actionRecreate = "recreate" // re-provision onto the current config
	actionWipe     = "wipe"
)

// Error codes carried alongside every container failure, so the UI can
// branch on the KIND of failure without matching on message text.
const (
	codeDockerAbsent      = "docker-absent"
	codeDockerUnreachable = "docker-unreachable"
	codeNotCreated        = "service-not-created"
	codeNotConfigured     = "not-configured"
)

// ---------------------------------------------------------------------
// wire shapes
// ---------------------------------------------------------------------

// serviceEndpoint is one URL a caller can actually dial, with the label that
// says what it is ("JSON-RPC", "WebSocket", "chain 1337").
type serviceEndpoint struct {
	Label string `json:"label"`
	URL   string `json:"url"`
}

// containerView is one docker-backed service as the services screen needs
// it: what it is, what state it is in, what it is reachable at, and what may
// be done to it right now.
type containerView struct {
	ID            string `json:"id"`
	Label         string `json:"label"`
	ContainerName string `json:"containerName"`

	// Configured is false when this target carries no config for the
	// service. The service is still reported (its container may exist from a
	// previous run, or from outside this app) — "nothing configured" is a
	// state to show, not a reason to omit the card.
	Configured bool `json:"configured"`

	// Status is ops.ContainerStatus verbatim, an untagged struct, so its
	// fields are PascalCase on the wire. Left untranslated on purpose: the
	// State constants are the same strings ops compares against.
	Status ops.ContainerStatus `json:"status"`

	// Endpoints are empty when the service is not running — a URL shown for
	// a stopped service is an invitation to a connection refused.
	Endpoints []serviceEndpoint `json:"endpoints"`

	// Actions are the ones this state permits, in the order to present them.
	Actions []string `json:"actions"`

	// Blocked explains, in the operator's terms, why Actions is empty or
	// short. It is the "show why, rather than let them hit a wall" half of
	// the same computation.
	Blocked string `json:"blocked,omitempty"`

	// WipeDiscards describes exactly what a wipe of this service deletes,
	// and RestartsOnWipe names the services it would then restart (from the
	// descriptor's FrontedBy — the same list ops.WipeService cascades to).
	WipeDiscards   string   `json:"wipeDiscards"`
	RestartsOnWipe []string `json:"restartsOnWipe"`

	// Warnings are configuration mismatches worth surfacing next to the
	// thing they affect, e.g. a gateway whose upstream does not point at the
	// devnet running beside it.
	Warnings []string `json:"warnings,omitempty"`

	// Devnet carries this service's RESOLVED config (defaults filled in), so
	// the editor opens showing the values that would actually be used rather
	// than a form full of blanks meaning "whatever the server decides".
	Devnet *catalog.DevnetConfig `json:"devnet,omitempty"`

	// Error and Hint report a per-service read failure inside a list
	// response, where one unreadable service must not blank the whole
	// screen. Single-service routes return these as an HTTP error instead.
	Error string `json:"error,omitempty"`
	Hint  string `json:"hint,omitempty"`
	Code  string `json:"code,omitempty"`
}

// dockerView is the engine reading the whole screen depends on. It is
// reported once, at the top level, because "docker is not running" is one
// fact about the machine and repeating it per service would read as two
// separate faults.
type dockerView struct {
	Present       bool   `json:"present"`
	Reachable     bool   `json:"reachable"`
	Flavor        string `json:"flavor"`
	ServerVersion string `json:"serverVersion,omitempty"`
	// Detail is the engine's own complaint, Hint the operator-facing action.
	Detail string `json:"detail,omitempty"`
	Hint   string `json:"hint,omitempty"`
}

type containersResponse struct {
	Docker   dockerView      `json:"docker"`
	Services []containerView `json:"services"`
}

// actionResponse is what start/stop/restart answer with: the state read back
// AFTER the action, never the state the caller asked for.
type actionResponse struct {
	Status ops.ContainerStatus `json:"status"`
}

// wipeResponse carries ops.WipeReport verbatim plus the state afterwards.
//
// Error is populated on a PARTIAL failure, which is a real and important
// outcome rather than a defensive nicety: a cascade that could not restart a
// front means the wipe DID happen and something in front of it is now serving
// a head the chain no longer has. The report has to travel with that error,
// or the operator is told "wipe failed" about a wipe that succeeded.
type wipeResponse struct {
	Report ops.WipeReport      `json:"report"`
	Status ops.ContainerStatus `json:"status"`
	Error  string              `json:"error,omitempty"`
	Hint   string              `json:"hint,omitempty"`
	Code   string              `json:"code,omitempty"`
}

// containerConfigResponse is the read/write shape for a service's stored
// configuration.
type containerConfigResponse struct {
	ID         string                `json:"id"`
	Configured bool                  `json:"configured"`
	Devnet     *catalog.DevnetConfig `json:"devnet,omitempty"`
}

// errorDetail is writeError's body plus the two fields a container failure
// needs: a typed code to branch on and the operator-facing hint the ops
// error already carries. Both are omitempty, so this is wire-compatible with
// every existing {"error": ...} response.
type errorDetail struct {
	Error string `json:"error"`
	Hint  string `json:"hint,omitempty"`
	Code  string `json:"code,omitempty"`
}

func writeErrorDetail(w http.ResponseWriter, status int, msg, hint, code string) {
	writeJSON(w, status, errorDetail{Error: msg, Hint: hint, Code: code})
}

// classifyOpsError maps one of ops' typed lifecycle errors onto the HTTP
// status, hint and code the API promises:
//
//   - ErrServiceNotCreated → 409. The request was well-formed and the target
//     is fine; the service simply has not been provisioned, and the fix is to
//     create it, not to retry.
//   - ErrDockerAbsent / ErrDockerUnreachable → 502. The failure is on the
//     TARGET, not in this request, which is the same reading every other
//     executor-backed route in this package gives a target-side fault. Both
//     carry a Hint written for an operator, and it is passed through verbatim
//     so the UI can show it without paraphrasing.
func classifyOpsError(err error) (status int, hint, code string) {
	var notCreated *ops.ServiceNotCreatedError
	var absent *ops.DockerAbsentError
	var unreachable *ops.DockerUnreachableError
	switch {
	case errors.As(err, &notCreated):
		return http.StatusConflict,
			"this service has not been created on the target yet — create it first",
			codeNotCreated
	case errors.As(err, &absent):
		return http.StatusBadGateway, absent.Hint, codeDockerAbsent
	case errors.As(err, &unreachable):
		return http.StatusBadGateway, unreachable.Hint, codeDockerUnreachable
	default:
		return http.StatusBadGateway, "", ""
	}
}

func writeOpsError(w http.ResponseWriter, err error) {
	status, hint, code := classifyOpsError(err)
	writeErrorDetail(w, status, err.Error(), hint, code)
}

// ---------------------------------------------------------------------
// routes
// ---------------------------------------------------------------------

// registerContainerRoutes mounts the container-service surface. As with
// services/{svc}/clear vs services/{svc}/{action}, the literal "wipe",
// "provision" and "config" segments are more specific than the {action}
// wildcard and win an exact match regardless of registration order.
func (s *Server) registerContainerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/targets/{id}/containers", s.handleContainerList)
	mux.HandleFunc("GET /api/targets/{id}/containers/{svc}", s.handleContainerStatus)
	mux.HandleFunc("GET /api/targets/{id}/containers/{svc}/config", s.handleContainerGetConfig)
	mux.HandleFunc("PUT /api/targets/{id}/containers/{svc}/config", s.handleContainerPutConfig)
	mux.HandleFunc("POST /api/targets/{id}/containers/{svc}/wipe", s.handleContainerWipe)
	mux.HandleFunc("POST /api/targets/{id}/containers/{svc}/reset", s.handleContainerReset)
	mux.HandleFunc("POST /api/targets/{id}/containers/{svc}/provision", s.handleContainerProvision)
	mux.HandleFunc("POST /api/targets/{id}/containers/{svc}/{action}", s.handleContainerAction)
}

// configAndTarget resolves id to a Target, returning the whole config with
// it. The config is needed because a target's devnet no longer knows what
// sits in front of it — the gateways do (config.Gateways), and the wipe
// cascade is computed from them.
//
// Unlike targetWithWire it does NOT require completed node setup: a machine
// can host a devnet and never run a node at all, so demanding a WireConfig
// here would gate the container surface on something unrelated to it.
func (s *Server) configAndTarget(w http.ResponseWriter, id string) (config.Config, config.Target, bool) {
	cfg, err := s.loadConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return config.Config{}, config.Target{}, false
	}
	t, ok := findTarget(cfg, id)
	if !ok {
		writeError(w, http.StatusNotFound, "target not found")
		return config.Config{}, config.Target{}, false
	}
	return cfg, t, true
}

func (s *Server) target(w http.ResponseWriter, id string) (config.Target, bool) {
	_, t, ok := s.configAndTarget(w, id)
	return t, ok
}

// resolveService is the shared preamble: target, service id, executor.
func (s *Server) resolveService(w http.ResponseWriter, r *http.Request) (config.Target, ops.DockerService, executor.Executor, bool) {
	cfg, t, ok := s.configAndTarget(w, r.PathValue("id"))
	if !ok {
		return config.Target{}, ops.DockerService{}, nil, false
	}
	svc := r.PathValue("svc")
	dsvc, err := containerService(cfg, t, svc)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return config.Target{}, ops.DockerService{}, nil, false
	}
	ex, err := s.getExecutor(t)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return config.Target{}, ops.DockerService{}, nil, false
	}
	return t, dsvc, ex, true
}

// ---------------------------------------------------------------------
// GET /api/targets/{id}/containers
// ---------------------------------------------------------------------

// handleContainerList reads this target's container services in one call,
// plus the engine reading they depend on.
//
// A per-service read failure is embedded in that service's view rather than
// failing the whole response. An engine that is missing or down is reported
// once, at the top, and every service then correctly reports no available
// actions.
func (s *Server) handleContainerList(w http.ResponseWriter, r *http.Request) {
	cfg, t, ok := s.configAndTarget(w, r.PathValue("id"))
	if !ok {
		return
	}
	ex, err := s.getExecutor(t)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	docker := probeDockerView(r.Context(), ex)

	views := make([]containerView, 0, 1)
	for _, id := range []string{svcDevnet} {
		dsvc, err := containerService(cfg, t, id)
		if err != nil {
			continue
		}
		views = append(views, s.viewFor(r.Context(), ex, t, id, dsvc, docker))
	}
	writeJSON(w, http.StatusOK, containersResponse{Docker: docker, Services: views})
}

// probeDockerView turns ops.ProbeDocker's reading into the UI's one-line
// verdict. Absence is a reading here, not an error: the screen's job is to
// say "install docker" rather than to fail.
func probeDockerView(ctx context.Context, ex executor.Executor) dockerView {
	info, err := ops.ProbeDocker(ctx, ex)
	if err != nil {
		var absent *ops.DockerAbsentError
		if errors.As(err, &absent) {
			return dockerView{Detail: err.Error(), Hint: absent.Hint}
		}
		return dockerView{Detail: err.Error()}
	}
	v := dockerView{
		Present:       info.Present,
		Reachable:     info.DaemonReachable,
		Flavor:        info.Flavor,
		ServerVersion: info.ServerVersion,
		Detail:        info.DaemonError,
	}
	switch {
	case info.WindowsContainers():
		v.Detail = "this engine is in Windows-container mode"
		v.Hint = "the devnet and gateway images are Linux images — switch Docker to Linux containers and retry"
	case !info.DaemonReachable:
		v.Hint = "start the engine and retry: Docker Desktop / OrbStack / colima on a desktop, or `systemctl start docker` on Linux"
	}
	return v
}

// viewFor reads one service and assembles its card. Errors land in the view
// (see handleContainerList) rather than being returned.
func (s *Server) viewFor(ctx context.Context, ex executor.Executor, t config.Target, id string, dsvc ops.DockerService, docker dockerView) containerView {
	st, err := ops.ServiceStatus(ctx, ex, dsvc)
	v := newContainerView(t, id, dsvc, st, s.livePorts(ctx, ex, dsvc, st))
	if err != nil {
		_, hint, code := classifyOpsError(err)
		v.Error, v.Hint, v.Code = err.Error(), hint, code
	}
	v.Actions, v.Blocked = availableActions(v, docker)
	return v
}

// livePorts reads the host-side ports a RUNNING container is actually
// published on.
//
// It is asked for only when the container is running, and its failure is
// swallowed into an empty map, because it is a refinement rather than a
// reading the screen depends on: without it the endpoints fall back to the
// saved configuration, which is right whenever the container was created from
// that configuration — and wrong in exactly the case this call detects.
func (s *Server) livePorts(ctx context.Context, ex executor.Executor, dsvc ops.DockerService, st ops.ContainerStatus) map[int]ops.PortBinding {
	if st.State != ops.StateRunning {
		return nil
	}
	ports, err := ops.PublishedPorts(ctx, ex, dsvc.ContainerName)
	if err != nil {
		return nil
	}
	return ports
}

// newContainerView assembles everything derivable without further engine
// calls. live is the running container's published ports (nil when it is not
// running, or when they could not be read).
func newContainerView(t config.Target, id string, dsvc ops.DockerService, st ops.ContainerStatus, live map[int]ops.PortBinding) containerView {
	v := containerView{
		ID:            id,
		ContainerName: dsvc.ContainerName,
		Status:        st,
	}
	for _, f := range dsvc.FrontedBy {
		v.RestartsOnWipe = append(v.RestartsOnWipe, f.ID)
	}
	switch id {
	case svcDevnet:
		d := resolvedDevnet(t.Devnet)
		v.Label = "Local devnet"
		v.Configured = t.Devnet != nil
		v.Devnet = &d
		// Named volumes are deliberately absent from the descriptor (reth
		// --dev keeps its database in the container's writable layer), so a
		// wipe's blast radius really is the container and nothing else.
		v.WipeDiscards = "the entire chain — every block, transaction and account state this devnet has produced. It comes back from genesis at block 0."
		if st.State == ops.StateRunning {
			v.Endpoints = []serviceEndpoint{
				{Label: "JSON-RPC", URL: liveURL("http", live, catalog.DevnetContainerHTTPPort, d.HTTPEndpoint())},
				{Label: "WebSocket", URL: liveURL("ws", live, catalog.DevnetContainerWSPort, d.WSEndpoint())},
			}
		}
		v.Warnings = append(v.Warnings,
			portDrift(live, catalog.DevnetContainerHTTPPort, d.HTTP(), "JSON-RPC")...)
		v.Warnings = append(v.Warnings,
			portDrift(live, catalog.DevnetContainerWSPort, d.WS(), "WebSocket")...)
	}
	if !v.Configured && st.Exists() {
		v.Warnings = append(v.Warnings,
			"This container exists but valve-node-app has no saved configuration for it — it was created somewhere else, or its configuration was removed. Saving one below is what makes re-creating it possible.")
	}
	return v
}

// portDrift reports a running container published on a different host port
// than the saved configuration asks for.
//
// This is the one config change that looks applied and is not: a container's
// -p mapping is fixed at creation, so editing a port and saving leaves the old
// container serving on the old port while every URL derived from the config
// says otherwise. Naming both numbers is the difference between "why is my
// wallet not connecting" and "press Re-create".
func portDrift(live map[int]ops.PortBinding, containerPort, configured int, label string) []string {
	b, ok := live[containerPort]
	if !ok || b.HostPort == configured {
		return nil
	}
	return []string{fmt.Sprintf(
		"The running container publishes %s on port %d, but the saved configuration says %d. A container's ports are fixed when it is created — re-create it to apply the change.",
		label, b.HostPort, configured)}
}

// availableActions decides what may be offered, and — just as important —
// says why when the answer is "nothing".
//
// The rules, each one a state that would otherwise produce a button that can
// only fail:
//
//   - No engine, or an unreadable service: nothing. Whatever is wrong is not
//     something a start button fixes.
//   - Not configured: only reading is possible. A create with no config would
//     have nothing to create from.
//   - Absent container: create. Not start — ops would create it via the
//     descriptor's hook, but calling that "start" hides a provisioning run
//     behind a word that means "resume".
//   - Stopped: start, recreate, wipe. No stop (already stopped) and no
//     restart, which for a stopped container is a start under a name that
//     suggests otherwise.
//   - Running: stop, restart, recreate, wipe. No start.
func availableActions(v containerView, docker dockerView) ([]string, string) {
	if v.Error != "" {
		return nil, "This service could not be read, so no action can be offered until that is resolved."
	}
	if !docker.Present {
		return nil, "There is no docker engine on this machine, and both of these services are containers."
	}
	if !docker.Reachable {
		return nil, "The docker CLI is installed but no engine answered, so nothing can be started or read."
	}
	if !v.Configured && !v.Status.Exists() {
		return nil, "Nothing is configured yet. Set it up below — the values you pick are what gets created."
	}

	switch v.Status.State {
	case ops.StateNotCreated:
		return []string{actionCreate}, ""
	case ops.StateStopped:
		return []string{actionStart, actionRecreate, actionWipe}, ""
	case ops.StateRunning:
		return []string{actionStop, actionRestart, actionRecreate, actionWipe}, ""
	default:
		return nil, "The engine did not report a usable state for this container, so no action can be offered."
	}
}

// liveURL builds "<scheme>://<host>:<port>" from the container's ACTUAL
// published binding for containerPort, falling back to the configured URL
// when there is no live reading.
//
// Preferring the live binding is the difference between a URL an operator can
// paste into a wallet and one that merely describes intent: a container's port
// mapping is fixed at creation, so a config edited since then names a port
// nothing is listening on.
func liveURL(scheme string, live map[int]ops.PortBinding, containerPort int, configured string) string {
	b, ok := live[containerPort]
	if !ok {
		return configured
	}
	return fmt.Sprintf("%s://%s:%d", scheme, endpointHost(b.HostIP), b.HostPort)
}

// endpointHost turns a bind address into something dialable, matching
// setup.probeHost: a wildcard names every interface but is not itself a
// destination (macOS refuses a connect to 0.0.0.0), and a wildcard listener
// is on loopback anyway.
func endpointHost(bind string) string {
	host := strings.Trim(bind, "[]")
	switch host {
	case "", "0.0.0.0":
		return "127.0.0.1"
	case "::", "::0":
		return "[::1]"
	}
	if strings.Contains(host, ":") {
		return "[" + host + "]"
	}
	return host
}

// ---------------------------------------------------------------------
// GET /api/targets/{id}/containers/{svc}
// ---------------------------------------------------------------------

func (s *Server) handleContainerStatus(w http.ResponseWriter, r *http.Request) {
	t, dsvc, ex, ok := s.resolveService(w, r)
	if !ok {
		return
	}
	st, err := ops.ServiceStatus(r.Context(), ex, dsvc)
	if err != nil {
		writeOpsError(w, err)
		return
	}
	docker := probeDockerView(r.Context(), ex)
	v := newContainerView(t, r.PathValue("svc"), dsvc, st, s.livePorts(r.Context(), ex, dsvc, st))
	v.Actions, v.Blocked = availableActions(v, docker)
	writeJSON(w, http.StatusOK, v)
}

// ---------------------------------------------------------------------
// POST /api/targets/{id}/containers/{svc}/{action}
// ---------------------------------------------------------------------

func (s *Server) handleContainerAction(w http.ResponseWriter, r *http.Request) {
	action := r.PathValue("action")
	switch action {
	case actionStart, actionStop, actionRestart:
	default:
		writeError(w, http.StatusBadRequest,
			fmt.Sprintf("unknown action %q (want %q, %q or %q — creating or re-creating a service is POST .../provision, and destroying its data is POST .../wipe)",
				action, actionStart, actionStop, actionRestart))
		return
	}

	_, dsvc, ex, ok := s.resolveService(w, r)
	if !ok {
		return
	}

	st, err := ops.ContainerAction(r.Context(), ex, dsvc, action)
	if err != nil {
		writeOpsError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, actionResponse{Status: st})
}

// ---------------------------------------------------------------------
// POST /api/targets/{id}/containers/{svc}/wipe
// ---------------------------------------------------------------------

// wipeRequest mirrors clearRequest's untagged-field convention: the typed
// confirmation gate is the same one handleServiceClear uses, because a wipe
// is the same class of irreversible action.
type wipeRequest struct {
	Confirm string
}

func (s *Server) handleContainerWipe(w http.ResponseWriter, r *http.Request) {
	svc := r.PathValue("svc")

	var req wipeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Confirm != svc {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("confirm must equal service name %q", svc))
		return
	}

	_, dsvc, ex, ok := s.resolveService(w, r)
	if !ok {
		return
	}

	rep, wipeErr := ops.WipeService(r.Context(), ex, dsvc)

	// The state is read back even after a failure. A wipe that failed on the
	// cascade has already removed and re-created the container, and the
	// operator needs to see that rather than infer it from an error string.
	st, stErr := ops.ServiceStatus(r.Context(), ex, dsvc)
	if stErr != nil {
		st = ops.ContainerStatus{ID: dsvc.ID, ContainerName: dsvc.ContainerName, State: ops.StateUnknown, Detail: stErr.Error()}
	}

	res := wipeResponse{Report: rep, Status: st}
	if wipeErr == nil {
		writeJSON(w, http.StatusOK, res)
		return
	}
	status, hint, code := classifyOpsError(wipeErr)
	res.Error, res.Hint, res.Code = wipeErr.Error(), hint, code
	writeJSON(w, status, res)
}

// ---------------------------------------------------------------------
// POST /api/targets/{id}/containers/{svc}/reset
// ---------------------------------------------------------------------

// handleContainerReset is the devnet's routine "give me a fresh chain".
//
// It is the SAME operation as wipe — ops.WipeService, so the container and
// its data go and the gateways in front of it are restarted — with one
// deliberate difference: no typed confirmation. A devnet is a scratch chain
// whose entire value is being cheap to throw away, and making the routine
// case type its own name is friction pretending to be safety. The typed gate
// stays on /wipe, and on the node's clear-and-resync, where what is destroyed
// is hundreds of gigabytes and days of resync.
//
// It is restricted to the devnet on purpose. There is no equivalent for a
// chain node here, and adding one would be putting a one-click button on the
// single most expensive irreversible action in the app.
//
// The cascade is the point, not a side effect: a reset chain restarts at a
// low block while eRPC keeps advertising the old head — its per-network head
// is monotonic, so it never observes the reset — and the restart is what
// clears it. Gateways on OTHER machines cannot be reached by ops' cascade
// (one executor), so they are restarted here and reported in the same list.
func (s *Server) handleContainerReset(w http.ResponseWriter, r *http.Request) {
	svc := r.PathValue("svc")
	if svc != svcDevnet {
		writeError(w, http.StatusNotFound, fmt.Sprintf(
			"only the devnet can be reset (%q is not) — resetting a real chain node means discarding its whole dataset, which is the dashboard's clear-and-resync and is deliberately guarded by a typed confirmation", svc))
		return
	}

	cfg, t, ok := s.configAndTarget(w, r.PathValue("id"))
	if !ok {
		return
	}
	dsvc, err := containerService(cfg, t, svc)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	ex, err := s.getExecutor(t)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	rep, wipeErr := ops.WipeService(r.Context(), ex, dsvc)

	// Gateways elsewhere in the fleet that front this devnet: same stale-head
	// problem, different machine, so they are bounced with their own target's
	// executor and folded into the same report.
	remoteErr := s.restartRemoteFronts(r.Context(), cfg, t, resolvedDevnet(t.Devnet), &rep)

	st, stErr := ops.ServiceStatus(r.Context(), ex, dsvc)
	if stErr != nil {
		st = ops.ContainerStatus{ID: dsvc.ID, ContainerName: dsvc.ContainerName, State: ops.StateUnknown, Detail: stErr.Error()}
	}

	res := wipeResponse{Report: rep, Status: st}
	switch {
	case wipeErr != nil:
		status, hint, code := classifyOpsError(wipeErr)
		res.Error, res.Hint, res.Code = wipeErr.Error(), hint, code
		writeJSON(w, status, res)
	case remoteErr != nil:
		res.Error = remoteErr.Error()
		writeJSON(w, http.StatusBadGateway, res)
	default:
		writeJSON(w, http.StatusOK, res)
	}
}

// restartRemoteFronts restarts gateways on OTHER targets that front this
// devnet, appending them to the report's Cascaded/CascadeSkipped lists so the
// operator sees one account of what was bounced rather than two.
func (s *Server) restartRemoteFronts(ctx context.Context, cfg config.Config, t config.Target, d catalog.DevnetConfig, rep *ops.WipeReport) error {
	var failures []error
	for _, gw := range cfg.Gateways {
		if gw.Placement.TargetID == t.ID || !gatewayFrontsDevnet(gw, t.ID, d) {
			continue
		}
		host, ok := findTarget(cfg, gw.Placement.TargetID)
		if !ok {
			failures = append(failures, fmt.Errorf("gateway %q is placed on target %q, which no longer exists", gw.ID, gw.Placement.TargetID))
			continue
		}
		gex, err := s.getExecutor(host)
		if err != nil {
			failures = append(failures, fmt.Errorf("gateway %q on %q: %w", gw.ID, host.ID, err))
			continue
		}
		dsvc := setup.GatewayService(gw.ID, gw.Config)
		st, err := ops.ServiceStatus(ctx, gex, dsvc)
		if err != nil {
			failures = append(failures, fmt.Errorf("gateway %q on %q: %w", gw.ID, host.ID, err))
			continue
		}
		if st.State != ops.StateRunning {
			rep.CascadeSkipped = append(rep.CascadeSkipped, dsvc.ID)
			continue
		}
		if _, err := ops.ContainerAction(ctx, gex, dsvc, "restart"); err != nil {
			failures = append(failures, fmt.Errorf("gateway %q on %q: %w", gw.ID, host.ID, err))
			continue
		}
		rep.Cascaded = append(rep.Cascaded, dsvc.ID)
	}
	if len(failures) > 0 {
		return fmt.Errorf("the devnet was reset, but a gateway on another machine could not be restarted, so it is now serving a head this chain no longer has — restart it by hand: %w", errors.Join(failures...))
	}
	return nil
}

// ---------------------------------------------------------------------
// POST /api/targets/{id}/containers/{svc}/provision
// ---------------------------------------------------------------------

// handleContainerProvision runs the service's setup plan — PlanDevnet or
// PlanGateway — through the SAME per-target setup run and SSE stream the node
// wizard uses (POST .../setup, GET .../setup/stream). One run slot per target
// is deliberate: these plans all drive the same executor against the same
// machine, and two of them interleaving is how a container gets created
// against a config that is being rewritten underneath it.
//
// It is both "create" and "re-create": each plan's run step removes the
// container before running it, because a container's ports, mounts and
// command are fixed at creation, so applying an edited config is necessarily
// remove + run rather than a restart.
func (s *Server) handleContainerProvision(w http.ResponseWriter, r *http.Request) {
	t, ok := s.target(w, r.PathValue("id"))
	if !ok {
		return
	}
	svc := r.PathValue("svc")

	steps, err := containerPlan(t, svc)
	if err != nil {
		writeErrorDetail(w, http.StatusBadRequest, err.Error(), "", codeNotConfigured)
		return
	}

	ex, err := s.getExecutor(t)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	claimed, ok := s.claimSetupRun(w, t.ID)
	if !ok {
		return
	}
	wire := catalog.WireConfig{}
	if t.Wire != nil {
		wire = *t.Wire
	}
	s.launchSetupRun(claimed, ex, steps, wire)

	writeJSON(w, http.StatusAccepted, map[string]string{"status": "started"})
}

// containerPlan returns the setup plan that provisions svc on t, or the
// reason it cannot be planned. An unconfigured service fails HERE, at plan
// time, with a sentence naming what is missing — the alternative is a run
// that starts, streams two steps and then fails on a rendering error.
func containerPlan(t config.Target, svc string) ([]setup.Step, error) {
	switch svc {
	case svcDevnet:
		if t.Devnet == nil {
			return nil, errors.New("no devnet is configured on this machine — save a devnet configuration first (PUT .../containers/devnet/config)")
		}
		return setup.PlanDevnet(*t.Devnet)
	default:
		return nil, fmt.Errorf("unknown service %q", svc)
	}
}

// ---------------------------------------------------------------------
// GET / PUT /api/targets/{id}/containers/{svc}/config
// ---------------------------------------------------------------------

func (s *Server) handleContainerGetConfig(w http.ResponseWriter, r *http.Request) {
	t, ok := s.target(w, r.PathValue("id"))
	if !ok {
		return
	}
	res, err := configResponseFor(t, r.PathValue("svc"))
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// handleContainerPutConfig validates and stores one service's configuration.
//
// It stores the DESIRED state and says so: nothing here touches a running
// container, because a container's ports and command line are fixed at
// creation. The caller applies it with POST .../provision, which is also why
// this route can safely accept a config for a service that does not exist
// yet.
func (s *Server) handleContainerPutConfig(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	svc := r.PathValue("svc")

	if _, ok := s.target(w, id); !ok {
		return
	}

	var devnet catalog.DevnetConfig
	switch svc {
	case svcDevnet:
		if err := json.NewDecoder(r.Body).Decode(&devnet); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		if err := devnet.Validate(); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
	default:
		writeError(w, http.StatusNotFound, fmt.Sprintf("unknown service %q", svc))
		return
	}

	cfg, err := s.updateConfig(func(c *config.Config) error {
		for i := range c.Targets {
			if c.Targets[i].ID != id {
				continue
			}
			d := devnet
			c.Targets[i].Devnet = &d
			return nil
		}
		return fmt.Errorf("target %q disappeared while saving", id)
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	t, _ := findTarget(cfg, id)
	res, err := configResponseFor(t, svc)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// configResponseFor returns svc's stored config with every default resolved,
// so an editor opens on the values that would actually be used rather than on
// blanks meaning "the server decides".
func configResponseFor(t config.Target, svc string) (containerConfigResponse, error) {
	switch svc {
	case svcDevnet:
		d := resolvedDevnet(t.Devnet)
		return containerConfigResponse{ID: svc, Configured: t.Devnet != nil, Devnet: &d}, nil
	default:
		return containerConfigResponse{}, fmt.Errorf("unknown service %q", svc)
	}
}

// resolvedDevnet fills in every default catalog's accessors would apply. A
// nil config resolves to the zero DevnetConfig's defaults, which are a valid,
// loopback-bound devnet — "just give me a chain" already spelled out.
func resolvedDevnet(d *catalog.DevnetConfig) catalog.DevnetConfig {
	var out catalog.DevnetConfig
	if d != nil {
		out = *d
	}
	out.ChainID = out.ChainIDOrDefault()
	out.BlockTime = out.BlockTimeOrDefault()
	out.ImageRef = out.Image()
	out.ContainerName = out.Name()
	out.BindAddr = out.Bind()
	out.HTTPPort = out.HTTP()
	out.WSPort = out.WS()
	return out
}

// ---------------------------------------------------------------------
// service descriptors
// ---------------------------------------------------------------------

// containerService builds the ops lifecycle descriptor for svc on t.
//
// A service with no stored config still yields a descriptor, built from the
// defaults, and that is the point: the container may well exist (from an
// earlier run, or from a config that was since deleted), and refusing to
// report on it would hide a running container from the only screen that could
// stop it.
//
// The devnet's FrontedBy is the gateways that actually front THIS devnet —
// see devnetFronts. That precision matters in both directions: a gateway
// fronting the devnet MUST be restarted after a wipe (ops.WipeService
// documents the stale-head failure), and one that does not must not be
// bounced for someone else's chain reset.
func containerService(cfg config.Config, t config.Target, svc string) (ops.DockerService, error) {
	switch svc {
	case svcDevnet:
		d := resolvedDevnet(t.Devnet)
		return setup.DevnetService(d, devnetFronts(cfg, t, d)...), nil
	default:
		return ops.DockerService{}, fmt.Errorf("unknown service %q (want %q)", svc, svcDevnet)
	}
}

// devnetFronts returns the gateway lifecycle descriptors that sit in front of
// this target's devnet, for ops.WipeService to cascade a restart to.
//
// Only gateways PLACED ON THE SAME TARGET are returned, and that limit is
// real rather than an oversight: the cascade runs through the one executor
// the wipe was issued on, so a gateway on another machine cannot be restarted
// from here. A cross-machine gateway pointing at this devnet is instead
// reported by handleContainerReset, which can reach the other target's
// executor — hiding it would recreate the exact silent-staleness bug the
// cascade exists to prevent, one machine over.
func devnetFronts(cfg config.Config, t config.Target, d catalog.DevnetConfig) []ops.DockerService {
	var out []ops.DockerService
	for _, gw := range cfg.GatewaysOn(t.ID) {
		if gatewayFrontsDevnet(gw, t.ID, d) {
			out = append(out, setup.GatewayService(gw.ID, gw.Config))
		}
	}
	return out
}

// gatewayFrontsDevnet reports whether gw routes a chain to this devnet.
//
// A managed-devnet upstream naming the target is the unambiguous answer. The
// endpoint comparison after it is for an upstream typed in by hand before
// upstreams had identity: it is the same devnet at the same URL, and treating
// it as unrelated would skip a restart that is required.
func gatewayFrontsDevnet(gw config.Gateway, targetID string, d catalog.DevnetConfig) bool {
	chain := d.ChainIDOrDefault()
	for _, n := range gw.Config.Networks {
		if n.ChainID != chain {
			continue
		}
		for _, u := range n.Upstreams {
			if u.KindOrDefault() == catalog.UpstreamManagedDevnet && u.TargetID == targetID {
				return true
			}
			if strings.EqualFold(strings.TrimSpace(u.Endpoint), d.HTTPEndpoint()) {
				return true
			}
		}
	}
	return false
}
