package server

// The HTTP surface for eRPC gateways, which are a LAYER OVER THE FLEET
// rather than a service any one machine owns.
//
// WHY they are top-level (/api/gateways/...) and not under a target, which is
// where they used to be: a gateway points at N chains across M endpoints, and
// those endpoints are scattered — a devnet on this laptop, a node on a fleet
// box two racks over, a public mainnet endpoint. Exactly ONE of the things a
// gateway touches is "the machine its process happens to run on", and
// addressing it as that machine's property made the incidental fact
// structural. Worse, it made a single gateway a hard limit: the container
// name was a constant, so a second gateway could not be created at all.
//
// What lives here that a passthrough would not:
//
//   - RESOLUTION. An upstream of a managed kind stores a REFERENCE (kind +
//     target id), not a URL, and the URL is derived here — the only layer
//     that can see targets — immediately before rendering. That is what stops
//     a gateway silently pointing at a node's old RPC address after the
//     operator changed it on the node's own screen.
//
//   - The ACTIONS each gateway's state permits, and the reason when the answer
//     is none, exactly as containers.go does for the devnet. A button that can
//     only fail is worse than no button.
//
//   - The SOURCES a new upstream can be built from (every managed node and
//     devnet in the fleet) and the network PRESETS the add-a-chain picker
//     offers, both derived from the catalog rather than retyped, so the picker
//     cannot drift from what the app actually supports.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/valve-tech/valve-node-app/internal/catalog"
	"github.com/valve-tech/valve-node-app/internal/chainlist"
	"github.com/valve-tech/valve-node-app/internal/config"
	"github.com/valve-tech/valve-node-app/internal/executor"
	"github.com/valve-tech/valve-node-app/internal/ops"
	"github.com/valve-tech/valve-node-app/internal/setup"
)

// gatewayIDPattern is what an id may look like. It is deliberately narrow:
// the id becomes a docker --name, a systemd unit name, a file name and a URL
// path segment, and the intersection of what those four accept is roughly
// this. Validating once, here, is what lets every downstream use it directly.
var gatewayIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,38}$`)

// codeGatewayNotFound is the typed code for "no gateway with that id", so the
// UI can tell it apart from a target-side failure without matching on text.
const codeGatewayNotFound = "gateway-not-found"

// ---------------------------------------------------------------------
// wire shapes
// ---------------------------------------------------------------------

// upstreamView is one endpoint under one chain — the tier-3 row.
//
// It carries BOTH what was stored and what it resolved to, because for a
// managed upstream those are different things and the difference is the whole
// point: "this machine's devnet" is the stored intent, "http://127.0.0.1:8600"
// is where that lands today, and an operator who moved the devnet's port needs
// to see the second value follow the first.
type upstreamView struct {
	ID   string `json:"id"`
	Kind string `json:"kind"`
	// TargetID is the machine a managed upstream refers to, "" for external.
	TargetID string `json:"targetId,omitempty"`
	// Endpoint is the URL eRPC will actually dial, derived for managed kinds.
	Endpoint string `json:"endpoint"`
	// Label says WHAT this is in the operator's terms ("devnet on local",
	// "node on box-a", "public endpoint"), which is the thing the raw URL
	// cannot say.
	Label      string `json:"label"`
	Local      bool   `json:"local"`
	RecentOnly bool   `json:"recentOnly"`

	// Problem is why this upstream cannot be used right now — a referenced
	// target that no longer exists, a devnet that was deleted, a loopback
	// address on a different machine. It is on the row because that is where
	// the operator is looking when they wonder why a chain is failing.
	Problem string `json:"problem,omitempty"`

	// Actions are the ones this KIND permits. Reset appears only on a devnet:
	// a scratch chain is cheap to throw away, a real node's dataset is not,
	// and an external endpoint is not ours to reset at all.
	Actions []string `json:"actions"`
}

// networkView is one chain the gateway serves — the tier-2 chip and, when
// selected, the tier-3 list under it.
type networkView struct {
	ChainID int    `json:"chainId"`
	Name    string `json:"name"`
	// URL is the full address callers dial for this chain, path and all. It
	// is only set while the gateway is actually running.
	URL  string `json:"url,omitempty"`
	Path string `json:"path"`

	Upstreams []upstreamView `json:"upstreams"`

	// KnownSetSize is how many upstreams "Add valve's set" would put on this
	// chain if it had none — the ENTRY count, which is what the redundancy bar
	// has to measure a configured count against. It rides on the view rather
	// than being fetched per chain from the knownset route because the bar is
	// drawn on the first paint of the list: a second request per chain would
	// buy a denominator that pops in late, for a number that is hardcoded and
	// cannot change between the two calls.
	//
	// Zero means valve has measured no set for this chain, and the bar must
	// then show no denominator at all rather than implying a target.
	KnownSetSize int `json:"knownSetSize"`

	// Serviceable is false when this chain has no upstream that can be
	// dialed. eRPC will accept the config and then fail every call on this
	// path, so the chip has to look different from a healthy one rather than
	// waiting for the operator to discover it by making a request.
	Serviceable bool     `json:"serviceable"`
	Warnings    []string `json:"warnings,omitempty"`
}

// tlsView is the HTTPS front as the RPC screen needs it.
//
// It reports the EFFECTIVE certificate source next to the configured one,
// because those two differ exactly when something went wrong with a
// certificate on disk and the app quietly kept serving anyway. A screen that
// showed only the configured value would be showing a setting, not the truth.
type tlsView struct {
	Enabled  bool   `json:"enabled"`
	Hostname string `json:"hostname,omitempty"`
	// URL is the https:// base callers dial. Set whenever TLS is configured,
	// running or not, because it is the thing an operator copies.
	URL string `json:"url,omitempty"`

	// CertSource is what the operator asked for; EffectiveCertSource is what is
	// actually being served. They differ only after an auto-fallback.
	CertSource          string `json:"certSource,omitempty"`
	EffectiveCertSource string `json:"effectiveCertSource,omitempty"`

	// Fallback is the full operator-facing reason the configured source was not
	// used, and FallbackReason its stable identifier ("expired",
	// "hostname-mismatch", ...). Both empty when nothing fell back.
	Fallback       string `json:"fallback,omitempty"`
	FallbackReason string `json:"fallbackReason,omitempty"`

	// ContainerName and Status are the Caddy container's, so the front's state
	// is visible beside the gateway's rather than inferred from it — they can
	// genuinely disagree, and a running gateway behind a dead front is a dead
	// endpoint.
	ContainerName string              `json:"containerName,omitempty"`
	Status        ops.ContainerStatus `json:"status"`

	// RootCAPath is where the internal CA's root was written on the target.
	// It is the file the operator installs to stop the browser warning, and
	// naming it is the difference between a solvable warning and a mystery.
	RootCAPath string `json:"rootCaPath,omitempty"`

	// SuggestedHostname is the name this gateway would get by default: one
	// under a domain whose wildcard already resolves to loopback, so HTTPS can
	// be turned on without first owning a name. It is sent whether or not TLS
	// is on, because the moment it is needed is the moment the operator is
	// looking at an empty hostname field.
	SuggestedHostname string `json:"suggestedHostname,omitempty"`

	// Verification is the LAST live check of this front (GET
	// .../tls/verify), or nil if none has been run since the app started.
	//
	// It is a cached result rather than something re-measured on every read,
	// and deliberately so: the check opens real connections, subscribes, and
	// waits for a block, so running it on every poll of the RPC screen would
	// turn a status read into a load generator. Its At field is on the wire so
	// a stale answer reads as stale.
	Verification *setup.TLSVerification `json:"verification,omitempty"`

	Error string `json:"error,omitempty"`
}

// gatewayView is one gateway as the RPC screen needs it: the full-width bar
// (state, endpoint, actions), the chains it fronts, and each chain's servers.
type gatewayView struct {
	ID            string                  `json:"id"`
	Label         string                  `json:"label"`
	ContainerName string                  `json:"containerName"`
	Placement     config.GatewayPlacement `json:"placement"`

	// Status is ops.ContainerStatus verbatim (untagged, so PascalCase on the
	// wire), matching containerView.
	Status ops.ContainerStatus `json:"status"`

	// Docker is the engine reading for the machine THIS gateway is placed on.
	// It is per-gateway rather than per-response because two gateways can sit
	// on two different machines, and "docker is down" is a fact about one of
	// them.
	Docker dockerView `json:"docker"`

	// BaseURL is the gateway's own front door, without a chain path. It is the
	// https:// one whenever the gateway is fronted — a fronted gateway
	// publishes no plaintext port at all, so the http URL is not merely less
	// good there, it is not listening.
	BaseURL string `json:"baseUrl"`

	// TLS is the HTTPS front, always present so the UI can render the "off"
	// state from the same shape as the "on" one.
	TLS tlsView `json:"tls"`

	Networks []networkView `json:"networks"`

	Actions []string `json:"actions"`
	Blocked string   `json:"blocked,omitempty"`

	WipeDiscards string   `json:"wipeDiscards"`
	Warnings     []string `json:"warnings,omitempty"`

	// Config is the STORED configuration (references intact, not resolved),
	// which is what an editor must round-trip: writing back a resolved config
	// would freeze today's URLs into it and undo the whole point of
	// references.
	//
	// It is REDACTED on the way out — see redactedGatewayConfig. The editor
	// therefore round-trips ${NAME} where a provider key used to be, and
	// resolveUpstreamKeys fills it again on both save paths, so what is stored
	// is still the real dialable URL.
	Config catalog.GatewayConfig `json:"config"`

	Error string `json:"error,omitempty"`
	Hint  string `json:"hint,omitempty"`
	Code  string `json:"code,omitempty"`
}

// upstreamSource is one thing in the fleet a new upstream can be built from.
type upstreamSource struct {
	Kind     string `json:"kind"`
	TargetID string `json:"targetId"`
	ChainID  int    `json:"chainId"`
	Label    string `json:"label"`
	// Endpoint is where it currently resolves to, shown so the operator can
	// see what they are about to point at. It is NOT what gets stored.
	Endpoint string `json:"endpoint"`
}

// networkPreset is one option in the add-a-chain picker. The three real
// networks come from catalog.Networks() and the devnet from
// catalog.DevnetChainID, so the picker cannot drift from the catalog.
type networkPreset struct {
	ChainID int    `json:"chainId"`
	Name    string `json:"name"`
	// Devnet marks the one preset that can provision its own upstream.
	Devnet bool `json:"devnet"`
}

// targetSummary is the placement picker's view of a machine.
type targetSummary struct {
	ID        string `json:"id"`
	Mode      string `json:"mode"`
	HasDevnet bool   `json:"hasDevnet"`
	HasNode   bool   `json:"hasNode"`
}

type gatewaysResponse struct {
	Gateways []gatewayView    `json:"gateways"`
	Targets  []targetSummary  `json:"targets"`
	Sources  []upstreamSource `json:"sources"`
	Presets  []networkPreset  `json:"presets"`

	// Orphans are containers a merge stopped managing but did NOT stop — see
	// config.Config.Orphans. Surfaced here so a merged-away gateway does not
	// keep serving stale config with nothing on any screen pointing at it.
	Orphans []config.OrphanedContainer `json:"orphans,omitempty"`
}

// ---------------------------------------------------------------------
// routes
// ---------------------------------------------------------------------

// registerGatewayRoutes mounts the fleet-wide gateway surface. As elsewhere,
// the literal segments ("wipe", "provision", "config") are more specific than
// the {action} wildcard and win an exact match regardless of order.
func (s *Server) registerGatewayRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/gateways", s.handleGatewayList)
	mux.HandleFunc("POST /api/gateways", s.handleGatewayCreate)
	mux.HandleFunc("GET /api/gateways/{gid}", s.handleGatewayGet)
	mux.HandleFunc("DELETE /api/gateways/{gid}", s.handleGatewayDelete)
	mux.HandleFunc("PUT /api/gateways/{gid}/config", s.handleGatewayPutConfig)
	mux.HandleFunc("POST /api/gateways/{gid}/provision", s.handleGatewayProvision)
	mux.HandleFunc("POST /api/gateways/{gid}/wipe", s.handleGatewayWipe)
	// The live HTTPS check. GET because it reads — it writes nothing, touches
	// no container and creates nothing — even though it is the one read here
	// that opens sockets of its own.
	mux.HandleFunc("GET /api/gateways/{gid}/tls/verify", s.handleGatewayTLSVerify)
	// Who is actually carrying the load. GET because it reads: it runs one
	// curl against a loopback port on the gateway's own machine and changes
	// nothing. Kept off the list response on purpose — see traffic.go.
	mux.HandleFunc("GET /api/gateways/{gid}/traffic", s.handleGatewayTraffic)
	// What each upstream can actually DO (archive, trace, debug, real
	// WebSocket...). GET because it reads, though — like tls/verify — it is
	// the one read here that opens sockets of its own, against every
	// upstream rather than one front. Cached for ten minutes for that reason;
	// ?refresh=1 forces a fresh probe. See capabilities.go.
	mux.HandleFunc("GET /api/gateways/{gid}/capabilities", s.handleGatewayCapabilities)
	// How is it doing, and why — latency, failures, error classes, lag and
	// eRPC's own selection state. One scrape folded twice, uncached like
	// traffic and for the same reason. See analytics.go.
	mux.HandleFunc("GET /api/gateways/{gid}/analytics", s.handleGatewayAnalytics)
	// The hardcoded, measured known set for one chain, with what this gateway
	// already has marked so the count offered is the count that lands. GET
	// because it reads: no key is written here, only reported. See knownset.go.
	mux.HandleFunc("GET /api/gateways/{gid}/knownset/{chainId}", s.handleKnownSet)
	// Install THIS gateway's own internal-CA root into the trust store of the
	// machine it runs on — a literal segment, so it wins the match over the
	// {action} wildcard exactly as wipe/provision do, and never reaches the
	// start/stop/restart dispatcher below. See handleGatewayTrustCert.
	mux.HandleFunc("POST /api/gateways/{gid}/trust-cert", s.handleGatewayTrustCert)
	mux.HandleFunc("POST /api/gateways/{gid}/{action}", s.handleGatewayAction)

	// A leftover container a merge stopped managing but did not stop — see
	// config.Config.Orphans. Dismissing it forgets the RECORD only; the
	// container is untouched, for the same reason handleGatewayDelete leaves
	// one running: this app never stops a container it did not just start.
	mux.HandleFunc("DELETE /api/orphans/{name}", s.handleOrphanDismiss)

	// Public-endpoint discovery. It is NOT under /api/gateways/{gid} because
	// it is a question about a chain, not about any one gateway — the answer
	// is the same whoever asks, and scoping it to a gateway would imply a
	// per-gateway result that does not exist.
	mux.HandleFunc("GET /api/chainlist/{chainId}", s.handleChainlist)
}

// gateway resolves {gid} to a stored gateway, answering 404 with a typed code
// when there is none.
func (s *Server) gateway(w http.ResponseWriter, r *http.Request) (config.Config, config.Gateway, bool) {
	cfg, err := s.loadConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return config.Config{}, config.Gateway{}, false
	}
	gw, ok := cfg.FindGateway(r.PathValue("gid"))
	if !ok {
		writeErrorDetail(w, http.StatusNotFound,
			fmt.Sprintf("no gateway %q", r.PathValue("gid")), "", codeGatewayNotFound)
		return config.Config{}, config.Gateway{}, false
	}
	return cfg, gw, true
}

// ---------------------------------------------------------------------
// GET /api/gateways/{gid}/knownset/{chainId}
// ---------------------------------------------------------------------

// knownSetEndpoint is one entry in the offered set, with AlreadyAdded so the
// UI can grey out what would be a duplicate rather than let the operator add
// it twice.
type knownSetEndpoint struct {
	URL          string `json:"url"`
	Provider     string `json:"provider"`
	WebSocket    bool   `json:"websocket"`
	Archive      bool   `json:"archive"`
	AlreadyAdded bool   `json:"alreadyAdded"`
}

type knownSetResponse struct {
	Endpoints []knownSetEndpoint `json:"endpoints"`
	// UsingDefaultKey says the set was resolved with the shared demo key rather
	// than one of the operator's own, so the UI can point at the reason valve's
	// entry is the least reliable one in the set.
	//
	// The key ITSELF used to travel here too, and does not any more: the UI
	// needs to know WHICH key is in play, never what it is. Whether a key is
	// the operator's own is a boolean; sending the secret to answer a boolean
	// is how a key ends up in a browser's memory, a proxy log and a screenshot.
	UsingDefaultKey bool `json:"usingDefaultKey"`
}

// handleKnownSet offers the hardcoded set for one chain, marking what this
// gateway already has so the count the operator sees before clicking matches
// what actually lands.
func (s *Server) handleKnownSet(w http.ResponseWriter, r *http.Request) {
	gid := r.PathValue("gid")
	chainID, err := strconv.Atoi(r.PathValue("chainId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "chain id must be a number")
		return
	}
	cfg, err := s.loadConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	gw, ok := cfg.FindGateway(gid)
	if !ok {
		writeErrorDetail(w, http.StatusNotFound, "no gateway "+gid, "", codeGatewayNotFound)
		return
	}

	have := map[string]bool{}
	for _, n := range gw.Config.Networks {
		if n.ChainID != chainID {
			continue
		}
		for _, u := range n.Upstreams {
			have[u.Endpoint] = true
		}
	}

	// One key for every chain: valve's key is an account, and migrate() has
	// already folded any per-chain keys an older config held into this one.
	key := strings.TrimSpace(cfg.ProviderKeys[config.ValveKeyPlaceholder])
	out := knownSetResponse{UsingDefaultKey: key == "" || key == catalog.DefaultValveKey}

	// keys is the same provider-key map discovery resolves and redacts
	// against — see providerKeys and redactKeys in chainlist.go — so the set
	// and the feed cannot disagree about which key answers which placeholder.
	keys := providerKeys(cfg)
	for _, e := range catalog.KnownSet(chainID) {
		// AlreadyAdded has to compare against the RESOLVED URL: what is
		// actually stored on the gateway is the dialable address (see
		// resolveUpstreamKeys), never a template.
		url, added := e.URL, have[e.URL]
		if resolved, ok := chainlist.Resolve(e.URL, keys); ok {
			added = have[resolved]
			// Redacted for the same reason handleChainlist redacts: this
			// entry can carry a key as a path segment, and the browser must
			// never see it — only whether one is in play (UsingDefaultKey).
			url = redactKeys(resolved, keys)
		}
		// An entry that cannot resolve (no key stored for its placeholder)
		// comes back with the placeholder still named in its URL rather than
		// being dropped, so the UI can say which key it wants.
		out.Endpoints = append(out.Endpoints, knownSetEndpoint{
			URL: url, Provider: e.Provider, WebSocket: e.WebSocket,
			Archive: e.Archive, AlreadyAdded: added,
		})
	}
	writeJSON(w, http.StatusOK, out)
}

// ---------------------------------------------------------------------
// GET /api/gateways
// ---------------------------------------------------------------------

// handleGatewayList reads every gateway, each against the machine it is
// placed on.
//
// A gateway whose target is unreachable lands as an error INSIDE its own view
// rather than failing the response: one bad SSH host must not blank the whole
// RPC screen, which is precisely the screen an operator opens to find out
// which gateway is broken.
func (s *Server) handleGatewayList(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.loadConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	views := make([]gatewayView, 0, len(cfg.Gateways))
	// One engine probe per TARGET, not per gateway: two gateways on one
	// machine share one docker, and probing twice would be two round trips
	// that can disagree with each other.
	probed := map[string]dockerView{}
	for _, gw := range cfg.Gateways {
		views = append(views, s.gatewayViewFor(r, cfg, gw, probed))
	}

	writeJSON(w, http.StatusOK, gatewaysResponse{
		Gateways: views,
		Targets:  targetSummaries(cfg),
		Sources:  upstreamSources(cfg),
		Presets:  networkPresets(),
		Orphans:  cfg.Orphans,
	})
}

func (s *Server) handleGatewayGet(w http.ResponseWriter, r *http.Request) {
	cfg, gw, ok := s.gateway(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, s.gatewayViewFor(r, cfg, gw, map[string]dockerView{}))
}

// gatewayViewFor assembles one gateway's whole card. Failures land in the
// view, not in an error return, for the reason handleGatewayList documents.
func (s *Server) gatewayViewFor(r *http.Request, cfg config.Config, gw config.Gateway, probed map[string]dockerView) gatewayView {
	resolved, problems := resolveGateway(cfg, gw)
	keys := providerKeys(cfg)

	v := gatewayView{
		ID:            gw.ID,
		Label:         gatewayLabel(gw),
		ContainerName: ops.ERPCContainerNameFor(gw.ID),
		Placement:     gw.Placement,
		Config:        redactedGatewayConfig(gw.Config, keys),
		Status:        ops.ContainerStatus{ID: "erpc:" + gw.ID, ContainerName: ops.ERPCContainerNameFor(gw.ID), State: ops.StateUnknown},
		// The gateway owns no volumes and no host files it may delete: its
		// erpc.yaml is a bind mount the operator owns, and ops.WipeService
		// never touches bind mounts.
		WipeDiscards: "the gateway container only. It is stateless — its erpc.yaml is a file on the host and is left untouched — so this is a rebuild, not a loss of data.",
		// The SAME resolveUpstream errors networkViews redacts into
		// uv.Problem, wrapped one level up ("chain %d, upstream %q: %s"). They
		// ride in the same body on the same poll, so redacting the per-row copy
		// and not this one would apply the argument to one of two strings that
		// carry identical text.
		Warnings: redactEach(problems, keys),
	}

	host, hostOK := findTarget(cfg, gw.Placement.TargetID)
	if !hostOK {
		v.Error = fmt.Sprintf("this gateway is placed on machine %q, which is no longer registered — re-place it or remove it", gw.Placement.TargetID)
		v.Networks = networkViews(cfg, gw, resolved, "")
		v.Blocked = "The machine this gateway runs on is gone, so nothing can be read or started."
		return v
	}

	ex, err := s.getExecutor(host)
	if err != nil {
		v.Error = err.Error()
		v.Networks = networkViews(cfg, gw, resolved, "")
		v.Blocked = "The machine this gateway runs on could not be reached."
		return v
	}

	docker, seen := probed[host.ID]
	if !seen {
		docker = probeDockerView(r.Context(), ex)
		probed[host.ID] = docker
	}
	v.Docker = docker

	dsvc := setup.GatewayService(gw.ID, resolved)
	st, statusErr := ops.ServiceStatus(r.Context(), ex, dsvc)
	v.Status = st
	if statusErr != nil {
		_, hint, code := classifyOpsError(statusErr)
		v.Error, v.Hint, v.Code = statusErr.Error(), hint, code
	}

	if w := st.EmulationWarning(); w != "" {
		v.Warnings = append(v.Warnings, w)
	}

	v.TLS = s.tlsViewFor(r, ex, gw, resolved)
	if w := v.TLS.Status.EmulationWarning(); w != "" {
		v.Warnings = append(v.Warnings, "TLS front: "+w)
	}

	// A fronted gateway's front door is Caddy's, and its readiness is Caddy's
	// too: eRPC publishes no host port there, so an http URL would name a port
	// nothing is bound to.
	base := fmt.Sprintf("http://%s:%d", endpointHost(resolved.Bind()), resolved.HTTP())
	reachable := st.State == ops.StateRunning
	if resolved.Fronted() {
		base = v.TLS.URL
		reachable = reachable && v.TLS.Status.State == ops.StateRunning
	} else if st.State == ops.StateRunning {
		live := s.livePorts(r.Context(), ex, dsvc, st)
		base = liveURL("http", live, ops.ERPCContainerPort, base)
		v.Warnings = append(v.Warnings, portDrift(live, ops.ERPCContainerPort, resolved.HTTP(), "gateway")...)
	}
	v.BaseURL = base

	perChain := ""
	if reachable {
		perChain = base
	}
	v.Networks = networkViews(cfg, gw, resolved, perChain)

	v.Actions, v.Blocked = gatewayActions(v, docker)
	return v
}

// tlsViewFor reads the TLS front: its container's state, and — by re-running
// the same certificate resolution the provisioner uses — whether the
// certificate the operator configured is the one actually being served.
//
// Re-running the resolution on a READ rather than caching what provisioning
// decided is the point. A certificate expires on a wall clock, not on a
// provisioning run, so a gateway that was fine when it was created can be
// serving a fallback certificate weeks later with nothing having happened in
// between. This screen is where that becomes visible.
func (s *Server) tlsViewFor(r *http.Request, ex executor.Executor, gw config.Gateway, resolved catalog.GatewayConfig) tlsView {
	suggested := suggestedTLSHostname(gw.ID)
	if !resolved.Fronted() {
		return tlsView{Status: ops.ContainerStatus{State: ops.StateNotCreated}, SuggestedHostname: suggested}
	}

	tls := resolved.TLS
	v := tlsView{
		Enabled:             true,
		Hostname:            tls.Hostname,
		URL:                 tls.URL(),
		CertSource:          tls.CertSourceOrDefault(),
		EffectiveCertSource: tls.CertSourceOrDefault(),
		ContainerName:       ops.CaddyContainerNameFor(gw.ID),
		SuggestedHostname:   suggested,
		Verification:        s.lastTLSVerification(gw.ID),
	}

	csvc := ops.CaddyServiceKeepingCA(gw.ID)
	st, err := ops.ServiceStatus(r.Context(), ex, csvc)
	v.Status = st
	if err != nil {
		v.Error = err.Error()
		return v
	}

	front, path, err := setup.GatewayTLSState(r.Context(), ex, gw.ID, resolved)
	if err != nil {
		v.Error = err.Error()
		return v
	}
	v.EffectiveCertSource = front.CertSource
	v.Fallback, v.FallbackReason = front.Fallback, front.FallbackReason
	if front.CertSource == catalog.CertInternal {
		v.RootCAPath = path
	}
	return v
}

func gatewayLabel(gw config.Gateway) string {
	if gw.ID == config.DefaultGatewayID {
		return "RPC gateway (eRPC)"
	}
	return "RPC gateway (eRPC) · " + gw.ID
}

// gatewayActions mirrors availableActions: the same states, the same reasons,
// and the same rule that an action which can only fail is never offered.
//
// A gateway with no chain is called out separately because it is the one
// blocked state whose cause is the CONFIG rather than the machine: eRPC
// refuses a project with no networks, so "create" would fail at render time,
// and the fix is to add a chain rather than to look at docker.
func gatewayActions(v gatewayView, docker dockerView) ([]string, string) {
	if v.Error != "" {
		return nil, "This gateway could not be read, so no action can be offered until that is resolved."
	}
	if !docker.Present {
		return nil, "There is no docker engine on the machine this gateway is placed on, and a gateway is a container."
	}
	if !docker.Reachable {
		return nil, "The docker CLI is installed on that machine but no engine answered, so nothing can be started or read."
	}
	if len(v.Config.Networks) == 0 {
		if v.Status.Exists() {
			return []string{actionStop, actionWipe}, "This gateway serves no chains, so it cannot be re-created until you add one."
		}
		return nil, "This gateway serves no chains yet. eRPC refuses a configuration with no networks, so add a chain before creating it."
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

// ---------------------------------------------------------------------
// resolution — the crux of "an endpoint may be local or fleet-accessible"
// ---------------------------------------------------------------------

// resolveGateway returns a COPY of gw's config with every managed upstream's
// Endpoint derived from the target it references, plus the problems found
// doing it.
//
// This is where a reference becomes an address, and it is deliberately the
// only place: catalog cannot do it (it would have to import config, which
// imports catalog) and setup should not (it sees one gateway, not the fleet).
// Deriving on every render is what makes a node's RPC address change
// propagate to every gateway fronting it, instead of leaving each gateway
// pointing at an address that was true when the upstream was added.
//
// Unresolvable upstreams are DROPPED from the returned config and reported.
// Leaving them in would fail the render (an upstream with no endpoint is an
// error, by design) and take the whole gateway down over one dead reference;
// dropping them keeps the other chains serving and puts the reason on the row
// that caused it.
func resolveGateway(cfg config.Config, gw config.Gateway) (catalog.GatewayConfig, []string) {
	out := gw.Config
	out.Networks = make([]catalog.GatewayNetwork, 0, len(gw.Config.Networks))

	var problems []string
	for _, n := range gw.Config.Networks {
		ups := make([]catalog.GatewayUpstream, 0, len(n.Upstreams))
		for _, u := range n.Upstreams {
			endpoint, _, err := resolveUpstream(cfg, gw, u)
			if err != nil {
				problems = append(problems, fmt.Sprintf("chain %d, upstream %q: %s", n.ChainID, upstreamName(u), err))
				continue
			}
			ru := u
			ru.Endpoint = endpoint
			// A managed upstream is one the operator runs, so it is preferred
			// over anything public. eRPC has exactly two tiers (preferred and
			// a 0.2-scored fallback), and filing your own node in the fallback
			// tier would route traffic to a public endpoint while your node
			// sat idle.
			ru.Local = u.Local || u.Managed()
			ups = append(ups, ru)
		}
		if len(ups) == 0 {
			// A network with no usable upstream cannot be rendered (eRPC
			// requires at least one), and including it would fail the whole
			// gateway rather than this one chain.
			problems = append(problems, fmt.Sprintf("chain %d has no usable upstream, so it is not served", n.ChainID))
			continue
		}
		out.Networks = append(out.Networks, catalog.GatewayNetwork{ChainID: n.ChainID, Upstreams: ups})
	}
	return out, problems
}

// resolveUpstream derives one upstream's dialable URL and a human label.
//
// An operator-set Name overrides the derived label on success; it is left
// alone on failure so a dead reference still reads as what kind of thing
// went missing (unresolvedLabel takes over from there, in networkViews).
func resolveUpstream(cfg config.Config, gw config.Gateway, u catalog.GatewayUpstream) (endpoint, label string, err error) {
	defer func() {
		if err == nil {
			if name := strings.TrimSpace(u.Name); name != "" {
				label = name
			}
		}
	}()
	switch u.KindOrDefault() {
	case catalog.UpstreamExternal:
		e := strings.TrimSpace(u.Endpoint)
		if e == "" {
			return "", "", errors.New("no endpoint")
		}
		return e, externalLabel(e, u.Local), nil

	case catalog.UpstreamManagedDevnet:
		t, ok := findTarget(cfg, u.TargetID)
		if !ok {
			return "", "", fmt.Errorf("machine %q is no longer registered", u.TargetID)
		}
		if t.Devnet == nil {
			return "", "", fmt.Errorf("machine %q has no devnet configured any more", u.TargetID)
		}
		d := resolvedDevnet(t.Devnet)
		// A devnet is ALWAYS a ws:// upstream, and it is not a preference — it
		// is the only spelling that makes eth_subscribe work. eRPC infers
		// WebSocket capability from the upstream SCHEME and has no separate
		// flag, so an http:// upstream makes every eth_subscribe fail with
		// ErrNoWsUpstreamAvailable ("requires a WebSocket-capable upstream,
		// none configured"), through a gateway that is otherwise perfectly
		// healthy.
		//
		// MEASURED, both ways, through a real fronted gateway: with an http://
		// upstream eth_chainId succeeded and eth_subscribe was refused; with
		// the ws:// one BOTH succeeded and newHeads arrived. A ws upstream
		// serves ordinary request/response calls too, so this costs nothing.
		//
		// Only the ADDRESS varies with placement, never the scheme. That is
		// deliberate: this used to be ws:// beside the gateway and http://
		// anywhere else, which made subscriptions a feature that appeared and
		// disappeared depending on which machine the devnet happened to be on —
		// with nothing on any screen to explain the difference.
		//
		// A devnet on the SAME machine as a container-hosted gateway is reached
		// by CONTAINER NAME on the shared docker network, not by a published
		// host port. That is what lets the devnet publish no port at all, and it
		// removes the host.docker.internal hop for the one case where both ends
		// are containers we placed ourselves. Note the two forms carry
		// different PORTS — in-container 8546 versus whatever the host mapping
		// publishes — so this is a choice between endpoints, not a choice of
		// hostname.
		if sameHostContainers(gw, u.TargetID) {
			return d.ContainerWSEndpoint(), devnetLabel(gw, u.TargetID), nil
		}
		e := d.WSEndpoint()
		if err := reachableAcrossMachines(gw, u.TargetID, e); err != nil {
			return "", "", err
		}
		return e, devnetLabel(gw, u.TargetID), nil

	case catalog.UpstreamManagedNode:
		t, ok := findTarget(cfg, u.TargetID)
		if !ok {
			return "", "", fmt.Errorf("machine %q is no longer registered", u.TargetID)
		}
		if t.Wire == nil {
			return "", "", fmt.Errorf("machine %q has no node set up any more", u.TargetID)
		}
		e := fmt.Sprintf("http://%s:%d", endpointHost(t.Wire.RPCBind()), t.Wire.ExecHTTP())
		if err := reachableAcrossMachines(gw, u.TargetID, e); err != nil {
			return "", "", err
		}
		return e, nodeLabel(gw, u.TargetID, *t.Wire), nil

	default:
		return "", "", fmt.Errorf("unknown upstream kind %q", u.Kind)
	}
}

// reachableAcrossMachines rejects the one derivation that looks fine and
// cannot work: a managed upstream on a DIFFERENT machine whose RPC is bound
// to loopback.
//
// From the gateway's box, 127.0.0.1 is the gateway's own box. The upstream
// would be dialed, connect to nothing (or, worse, to something else listening
// on that port locally), and the gateway would report a healthy config while
// serving errors. The fix is on the node's own screen — bind its RPC to a
// routable address — so the message says that rather than offering a
// workaround here.
func reachableAcrossMachines(gw config.Gateway, upstreamTarget, endpoint string) error {
	if upstreamTarget == gw.Placement.TargetID {
		return nil
	}
	u, err := url.Parse(endpoint)
	if err != nil {
		return fmt.Errorf("%q is not a URL", endpoint)
	}
	if !isLoopbackHost(u.Hostname()) {
		return nil
	}
	return fmt.Errorf("it is on machine %q but bound to loopback (%s), which from machine %q means that machine's own loopback — bind its RPC to an address the gateway's machine can reach",
		upstreamTarget, endpoint, gw.Placement.TargetID)
}

// sameHostContainers reports whether a gateway and an upstream are BOTH
// containers this app placed on the SAME machine — the one situation where a
// container name is a better address than a URL.
//
// Both halves of the test are load-bearing. Same machine, because docker's
// embedded DNS resolves a name only within one engine's network. Container
// backend, because a systemd-hosted gateway is an ordinary process with no
// network to resolve names on: handing it "valve-node-app-devnet:8545" would
// produce a gateway whose every call fails DNS resolution while its config
// looks perfectly reasonable.
//
// A managed NODE is deliberately not covered even on the same machine: a node
// is a systemd install, not a container, so it has no name on the network and
// host.docker.internal remains the only way to it. That path does not go away
// here, it stops being the default.
func sameHostContainers(gw config.Gateway, upstreamTarget string) bool {
	return upstreamTarget == gw.Placement.TargetID && gw.Placement.Backend == setup.BackendDocker
}

func isLoopbackHost(host string) bool {
	switch strings.Trim(host, "[]") {
	case "127.0.0.1", "localhost", "::1", "0.0.0.0", "::":
		return true
	}
	return false
}

func upstreamName(u catalog.GatewayUpstream) string {
	if u.ID != "" {
		return u.ID
	}
	if u.Managed() {
		return u.KindOrDefault() + "@" + u.TargetID
	}
	return u.Endpoint
}

func devnetLabel(gw config.Gateway, targetID string) string {
	if targetID == gw.Placement.TargetID {
		return "devnet on this machine (" + targetID + ")"
	}
	return "devnet on " + targetID
}

func nodeLabel(gw config.Gateway, targetID string, w catalog.WireConfig) string {
	where := "node on " + targetID
	if targetID == gw.Placement.TargetID {
		where = "node on this machine (" + targetID + ")"
	}
	if n, ok := catalog.NetworkByChainID(w.ChainID); ok {
		return where + " · " + n.Name + " · " + w.ExecID
	}
	return where + " · " + w.ExecID
}

// externalLabel names an external endpoint by its host, which is what an
// operator recognizes ("rpc.pulsechain.com"), rather than repeating the whole
// URL that is already displayed beside it.
//
// Two endpoints are NOT called public, and both exclusions are about not
// telling the operator something untrue on the row they are reading:
//
//   - A LOOPBACK endpoint is something on the gateway's own machine that this
//     app does not manage. Calling that public is both wrong and misleading
//     about where the traffic goes.
//   - An endpoint the operator marked as THEIRS (local, i.e. eRPC's preferred
//     tier) is theirs whether or not this app installed it. FOUND BY RUNNING
//     IT: a node fronted this way was labelled "public endpoint · 192.168.3.22"
//     on a row whose Role column said "Yours", the two contradicting each other
//     in the same table. This is the ordinary way to front a node
//     valve-node-app does not manage — someone else's box, an install done by
//     hand — and the tier is exactly what decides the intended share, so
//     mislabelling it undercuts the column that explains an amber bar.
func externalLabel(endpoint string, mine bool) string {
	u, err := url.Parse(endpoint)
	if err != nil || u.Host == "" {
		return "external endpoint"
	}
	if isLoopbackHost(u.Hostname()) {
		return "unmanaged endpoint on the gateway's own machine"
	}
	if mine {
		return "your endpoint · " + u.Hostname()
	}
	return "public endpoint · " + u.Hostname()
}

// ---------------------------------------------------------------------
// network / upstream views
// ---------------------------------------------------------------------

// redactedGatewayConfig returns g with every upstream endpoint put back into
// its ${PLACEHOLDER} form, for sending to the browser.
//
// The stored endpoint of a templated upstream carries the operator's provider
// key as a path segment (https://mainnet.infura.io/v3/<key>), because
// resolveUpstreamKeys fills the slot on the way IN so that every reader — the
// renderer, the provisioner, the probe — sees a dialable URL. This whole
// config is then serialised to the RPC screen on every poll, which would hand
// that key to any script on the page and to anything that logs a response. The
// placeholder form is just as useful to the editor: it round-trips, and
// resolveUpstreamKeys fills it again on save.
//
// THE COPY IS THE POINT. g arrives as a shallow struct copy of loaded config,
// so g.Networks and each network's Upstreams still share their backing arrays
// with the config this process holds in memory, renders from and saves.
// Redacting through those aliases would replace real keys with ${NAME} in the
// live configuration — a display concern turned into data loss. Deep-copy
// first, exactly as config.cloneNetworks does and for exactly this reason.
//
// LIMIT, not solved here: redaction matches by VALUE against the stored keys,
// so an endpoint whose key is not in ProviderKeys — rotated since it was
// saved, or hand-typed with an embedded secret that was never stored as a
// provider key — does not match and stays visible. Nothing in this process
// can tell that path segment apart from an ordinary one.
func redactedGatewayConfig(g catalog.GatewayConfig, keys map[string]string) catalog.GatewayConfig {
	nets := make([]catalog.GatewayNetwork, len(g.Networks))
	for i, n := range g.Networks {
		nets[i] = n
		nets[i].Upstreams = append([]catalog.GatewayUpstream(nil), n.Upstreams...)
		for j := range nets[i].Upstreams {
			nets[i].Upstreams[j].Endpoint = redactKeys(nets[i].Upstreams[j].Endpoint, keys)
		}
	}
	g.Networks = nets
	return g
}

// redactEach is redactKeys over a slice, returning a new one. nil in, nil out,
// so an empty warnings list stays absent from the wire rather than becoming an
// empty array.
func redactEach(ss []string, keys map[string]string) []string {
	if len(ss) == 0 {
		return ss
	}
	out := make([]string, len(ss))
	for i, s := range ss {
		out[i] = redactKeys(s, keys)
	}
	return out
}

// networkViews renders the chains and their servers. base is "" when the
// gateway is not running, which is what suppresses per-chain URLs.
//
// Every endpoint and every problem string goes out redacted, for the reason
// redactedGatewayConfig gives: this is the same URL, on the same screen, on
// the same poll. Nothing is written back through gw or resolved — the redacted
// string only ever lands in the view being built.
func networkViews(cfg config.Config, gw config.Gateway, resolved catalog.GatewayConfig, base string) []networkView {
	keys := providerKeys(cfg)
	out := make([]networkView, 0, len(gw.Config.Networks))
	for _, n := range gw.Config.Networks {
		nv := networkView{
			ChainID:      n.ChainID,
			Name:         chainName(n.ChainID),
			Path:         resolved.PathFor(n.ChainID),
			KnownSetSize: catalog.KnownSetSize(n.ChainID),
		}
		if base != "" {
			nv.URL = base + nv.Path
		}
		for _, u := range n.Upstreams {
			endpoint, label, err := resolveUpstream(cfg, gw, u)
			uv := upstreamView{
				ID:         upstreamName(u),
				Kind:       u.KindOrDefault(),
				TargetID:   u.TargetID,
				Endpoint:   redactKeys(endpoint, keys),
				Label:      label,
				Local:      u.Local || u.Managed(),
				RecentOnly: u.RecentOnly,
				Actions:    upstreamActions(u),
			}
			if err != nil {
				uv.Problem = redactKeys(err.Error(), keys)
				// An operator-set Name still wins even here: it is the thing
				// they recognise their row by, and that is exactly what they
				// need most when the row has gone unhealthy — reverting to a
				// generic "public endpoint (unusable)" the moment it breaks
				// would hide the one label the operator chose.
				if name := strings.TrimSpace(u.Name); name != "" {
					uv.Label = name
				} else {
					uv.Label = unresolvedLabel(u)
				}
			} else {
				nv.Serviceable = true
			}
			nv.Upstreams = append(nv.Upstreams, uv)
		}
		if len(nv.Upstreams) == 0 {
			nv.Warnings = append(nv.Warnings,
				"This chain has no endpoint, so eRPC has nowhere to send its calls — the gateway cannot even be created until it has one.")
		} else if !nv.Serviceable {
			nv.Warnings = append(nv.Warnings,
				"Every endpoint on this chain is unusable right now, so calls on this path will fail. The reason is on each row below.")
		}
		out = append(out, nv)
	}
	return out
}

func unresolvedLabel(u catalog.GatewayUpstream) string {
	switch u.KindOrDefault() {
	case catalog.UpstreamManagedDevnet:
		return "devnet on " + u.TargetID + " (unavailable)"
	case catalog.UpstreamManagedNode:
		return "node on " + u.TargetID + " (unavailable)"
	default:
		return "public endpoint (unusable)"
	}
}

// upstreamActions is the asymmetry the owner asked for, keyed off the KIND
// rather than off a label, so it cannot be defeated by renaming something:
//
//   - a devnet gets Reset, one click, no typed confirmation. It is a scratch
//     chain and throwing it away is routine;
//   - a real node gets NO reset. The equivalent — discarding hundreds of GB
//     and days of resync — already exists on the dashboard, behind a typed
//     confirmation, and duplicating it here would be softening it;
//   - an external endpoint gets neither. It is not ours to reset.
//
// Remove is always available: it edits this gateway's own configuration and
// touches nothing on the other end.
func upstreamActions(u catalog.GatewayUpstream) []string {
	if u.KindOrDefault() == catalog.UpstreamManagedDevnet {
		return []string{"reset", "remove"}
	}
	return []string{"remove"}
}

func chainName(chainID int) string {
	if n, ok := catalog.NetworkByChainID(chainID); ok {
		return n.Name
	}
	if chainID == catalog.DevnetChainID {
		return "Devnet"
	}
	return fmt.Sprintf("Chain %d", chainID)
}

// ---------------------------------------------------------------------
// fleet inventory: what a new upstream can point at
// ---------------------------------------------------------------------

// upstreamSources enumerates every managed thing in the fleet that can serve
// RPC. The UI offers these directly, so an operator adding an endpoint picks
// a machine's node or devnet by name rather than typing a URL that will be
// wrong the moment the port changes.
func upstreamSources(cfg config.Config) []upstreamSource {
	var out []upstreamSource
	for _, t := range cfg.Targets {
		if t.Devnet != nil {
			d := resolvedDevnet(t.Devnet)
			out = append(out, upstreamSource{
				Kind:     catalog.UpstreamManagedDevnet,
				TargetID: t.ID,
				ChainID:  d.ChainIDOrDefault(),
				Label:    "Devnet on " + t.ID,
				Endpoint: d.HTTPEndpoint(),
			})
		}
		if t.Wire != nil {
			out = append(out, upstreamSource{
				Kind:     catalog.UpstreamManagedNode,
				TargetID: t.ID,
				ChainID:  t.Wire.ChainID,
				Label:    "Node on " + t.ID + " (" + chainName(t.Wire.ChainID) + ")",
				Endpoint: fmt.Sprintf("http://%s:%d", endpointHost(t.Wire.RPCBind()), t.Wire.ExecHTTP()),
			})
		}
	}
	return out
}

// networkPresets is the add-a-chain picker's default list, in the owner's
// order: the catalog's networks first, then the devnet. Names come from the
// catalog rather than being retyped here, so the picker cannot drift from it.
// "Add custom…" is the UI's own trailing option and needs no entry: any chain
// id is valid, because RenderGatewayConfig deliberately does not restrict
// chain ids to the ones this app can run a node for.
func networkPresets() []networkPreset {
	nets := catalog.Networks()
	out := make([]networkPreset, 0, len(nets)+1)
	for _, n := range nets {
		out = append(out, networkPreset{ChainID: n.ChainID, Name: n.Name})
	}
	return append(out, networkPreset{ChainID: catalog.DevnetChainID, Name: "Devnet", Devnet: true})
}

func targetSummaries(cfg config.Config) []targetSummary {
	out := make([]targetSummary, 0, len(cfg.Targets))
	for _, t := range cfg.Targets {
		out = append(out, targetSummary{
			ID:        t.ID,
			Mode:      t.Mode,
			HasDevnet: t.Devnet != nil,
			HasNode:   t.Wire != nil,
		})
	}
	return out
}

// ---------------------------------------------------------------------
// POST /api/gateways  — create
// ---------------------------------------------------------------------

type createGatewayRequest struct {
	ID        string                  `json:"id"`
	Placement config.GatewayPlacement `json:"placement"`
	// Config is optional. A gateway created with no chains is a legitimate
	// starting point — the RPC screen's whole flow is "make the gateway, then
	// add the networks it fronts" — and it simply cannot be provisioned until
	// it has one, which its own Blocked message says.
	Config *catalog.GatewayConfig `json:"config"`
}

func (s *Server) handleGatewayCreate(w http.ResponseWriter, r *http.Request) {
	var req createGatewayRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	id := strings.TrimSpace(req.ID)
	if id == "" {
		id = config.DefaultGatewayID
	}
	if !gatewayIDPattern.MatchString(id) {
		writeError(w, http.StatusBadRequest,
			"a gateway id must be lower-case letters, digits, dot, dash or underscore (starting with a letter or digit), at most 39 characters — it becomes a container name, a unit name and a file name")
		return
	}

	backend := strings.TrimSpace(req.Placement.Backend)
	if backend == "" {
		backend = setup.BackendDocker
	}
	if backend != setup.BackendDocker && backend != setup.BackendSystemd {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("unknown backend %q (want %q or %q)", backend, setup.BackendDocker, setup.BackendSystemd))
		return
	}

	gwCfg := catalog.GatewayConfig{}
	if req.Config != nil {
		gwCfg = *req.Config
	}
	// A new gateway starts with a hostname that actually resolves, rather than
	// with an empty field and a placeholder. Turning HTTPS on then needs no
	// domain, no hosts-file edit and no decision — which is the difference
	// between the feature being used and being skipped. A hostname the caller
	// supplied is never touched, and this does NOT turn HTTPS on: it only means
	// the name is there when it is.
	defaultTLSHostname(&gwCfg, id)
	// TLS is validated unconditionally, unlike the networks: a gateway with no
	// chains yet is a legitimate starting state, but a gateway with a malformed
	// hostname is not one at any stage.
	if err := gwCfg.TLS.ValidateSettings(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	// The same slot-filling the update path does, for the same reason and with
	// one more: validateGatewayConfig does NOT reject a ${...} — url.Parse is
	// perfectly happy with braces in a path — so a gateway created with an
	// unfilled slot would store one, and then every later edit would 400 on the
	// update path's refusal, over an upstream the operator never touched.
	// Creating a config the app will not let you save again is worse than
	// refusing it here.
	keyCfg, err := s.loadConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	keys := providerKeys(keyCfg)
	if err := resolveUpstreamKeys(&gwCfg, keys); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	// Only validate a config that has something in it. An empty one is the
	// intended starting state, and rendering it would fail on "no networks" —
	// a true statement, but not a reason to refuse to create the gateway.
	if len(gwCfg.Networks) > 0 {
		// Redacted, because the slots were just filled in: the validation error
		// for a bad scheme QUOTES the endpoint, and by now that endpoint carries
		// the real key. See handleGatewayPutConfig for the full reasoning.
		if err := validateGatewayConfig(gwCfg); err != nil {
			writeError(w, http.StatusBadRequest, redactKeys(err.Error(), keys))
			return
		}
	}

	cfg, err := s.updateConfig(func(c *config.Config) error {
		if _, exists := c.FindGateway(id); exists {
			return fmt.Errorf("a gateway called %q already exists", id)
		}
		if _, ok := findTarget(*c, req.Placement.TargetID); !ok {
			return fmt.Errorf("machine %q is not registered — a gateway has to be placed on a machine this app manages", req.Placement.TargetID)
		}
		// One managed eRPC per device. The gateway's Placement names its
		// machine, so a second gateway on that machine is a second container
		// fighting over the same chains — and only one of them can be behind
		// the reverse proxy.
		targetID := strings.TrimSpace(req.Placement.TargetID)
		for _, g := range c.Gateways {
			if g.Placement.TargetID == targetID {
				return fmt.Errorf(
					"machine %q already runs gateway %q — a machine hosts one managed eRPC, so add the chains to that gateway instead of creating a second one",
					targetID, g.ID)
			}
		}
		c.Gateways = append(c.Gateways, config.Gateway{
			ID:        id,
			Placement: config.GatewayPlacement{TargetID: req.Placement.TargetID, Backend: backend},
			Config:    gwCfg,
		})
		// A merged-away gateway's id is FREED, so this new gateway may be
		// re-claiming the very container names an old leftover record still
		// tells the operator to `docker rm -f`. That advice was correct while
		// the name belonged to nothing; it is now aimed at a container this
		// app manages, and on a shared or aliased docker host following it
		// destroys the gateway just created. Dropping the record here, inside
		// the same write that appends the gateway, means the banner and the
		// gateway can never both exist.
		//
		// Both derived names go, not just the eRPC one: creating gateway <id>
		// reserves valve-node-app-caddy-<id> as well, and turning HTTPS on
		// later is an edit, not a re-creation, so there would be no second
		// chance to clear it.
		reclaimed := map[string]bool{
			ops.ERPCContainerNameFor(id):  true,
			ops.CaddyContainerNameFor(id): true,
		}
		kept := c.Orphans[:0]
		for _, o := range c.Orphans {
			if !(reclaimed[o.ContainerName] && o.TargetID == targetID) {
				kept = append(kept, o)
			}
		}
		c.Orphans = kept
		return nil
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	gw, _ := cfg.FindGateway(id)
	writeJSON(w, http.StatusCreated, s.gatewayViewFor(r, cfg, gw, map[string]dockerView{}))
}

// ---------------------------------------------------------------------
// PUT /api/gateways/{gid}/config
// ---------------------------------------------------------------------

// handleGatewayPutConfig stores the DESIRED configuration and says so:
// nothing here touches a running container, because a container's published
// port and mounts are fixed at creation. POST .../provision applies it.
func (s *Server) handleGatewayPutConfig(w http.ResponseWriter, r *http.Request) {
	gid := r.PathValue("gid")

	var gwCfg catalog.GatewayConfig
	if err := json.NewDecoder(r.Body).Decode(&gwCfg); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	keyCfg, err := s.loadConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	// Same slot-filling handleGatewayCreate does, and for the same reason: a
	// caller that builds a config from scratch (rather than round-tripping one
	// this app already returned) sends an empty Hostname meaning "you choose",
	// not "turn HTTPS off" — TLS.Enabled already carries that decision. Without
	// this, a config that validated fine on create fails here with "hostname is
	// required" the moment anything else about it changes, which is exactly the
	// gap the one-click setup flow (home.ts/panel.ts) hit: it builds the same
	// internalTLSConfig(networks) twice, once for create and once for this PUT,
	// and only create used to fill the hostname in.
	defaultTLSHostname(&gwCfg, gid)
	// Before validation, because an unfilled ${...} slot is not a URL eRPC
	// could ever dial, and the operator should be told which key is missing
	// rather than whatever the renderer makes of the literal.
	keys := providerKeys(keyCfg)
	if err := resolveUpstreamKeys(&gwCfg, keys); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := gwCfg.TLS.ValidateSettings(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if len(gwCfg.Networks) > 0 {
		// The validation error is REDACTED, because resolveUpstreamKeys has
		// already run: a bad-scheme upstream is reported by quoting the endpoint,
		// and that endpoint now has the operator's real key substituted into it.
		// Unredacted, posting {"Endpoint":"gopher://x/${INFURA_API_KEY}"} would
		// read any stored key straight back out of a 400 body — including one
		// that no upstream uses and no screen shows.
		if err := validateGatewayConfig(gwCfg); err != nil {
			writeError(w, http.StatusBadRequest, redactKeys(err.Error(), keys))
			return
		}
	}

	cfg, err := s.updateConfig(func(c *config.Config) error {
		for i := range c.Gateways {
			if c.Gateways[i].ID != gid {
				continue
			}
			c.Gateways[i].Config = gwCfg
			return nil
		}
		return fmt.Errorf("no gateway %q", gid)
	})
	if err != nil {
		writeErrorDetail(w, http.StatusNotFound, err.Error(), "", codeGatewayNotFound)
		return
	}

	gw, _ := cfg.FindGateway(gid)
	writeJSON(w, http.StatusOK, s.gatewayViewFor(r, cfg, gw, map[string]dockerView{}))
}

// resolveUpstreamKeys fills the ${PLACEHOLDER} slots in external upstream URLs
// from the stored provider keys.
//
// This is the other half of redactKeys (chainlist.go). Discovery hands the
// browser the PLACEHOLDER form of a templated endpoint so the key stays in this
// process; the browser posts that same string back to add it; this is where it
// becomes the URL eRPC will actually dial. Resolving HERE rather than storing
// the placeholder is deliberate: the stored endpoint is read by the renderer,
// the provisioner and the capability probe, and pushing "resolve first" into
// each of them would be three more places for a key to be forgotten and an
// upstream to 401 in a way nothing explains.
//
// An unresolvable slot is refused rather than stored. A URL with a literal
// ${...} in it looks configured and answers nothing, which is exactly the
// failure chainlist.Resolve refuses for the same reason.
func resolveUpstreamKeys(g *catalog.GatewayConfig, keys map[string]string) error {
	for i := range g.Networks {
		for j := range g.Networks[i].Upstreams {
			u := &g.Networks[i].Upstreams[j]
			raw := strings.TrimSpace(u.Endpoint)
			if !strings.Contains(raw, "${") {
				continue
			}
			resolved, ok := chainlist.Resolve(raw, keys)
			if !ok {
				name := chainlist.PlaceholderName(raw)
				if name == "" {
					return fmt.Errorf("chain %d: upstream %q has a malformed ${...} slot in its endpoint", g.Networks[i].ChainID, u.ID)
				}
				return fmt.Errorf("chain %d: upstream %q needs %s — add it in Settings", g.Networks[i].ChainID, u.ID, name)
			}
			u.Endpoint = resolved
		}
	}
	return nil
}

// validateGatewayConfig is the same validation PlanGateway performs, run at
// save time so a bad edit is refused where it was made.
//
// Managed upstreams carry no endpoint of their own, so a stand-in URL is
// substituted before rendering: the render is checking chain ids, duplicate
// upstream ids and endpoint schemes, and a managed reference's real address
// is only knowable at provision time. Everything the render CAN check about
// them is still checked.
func validateGatewayConfig(g catalog.GatewayConfig) error {
	// The TLS settings are checked here even though RenderGatewayConfig checks
	// them too, so that a bad hostname is refused at the moment it is typed
	// rather than at the moment a provisioning run fails. The certificate FILES
	// are not checked here: they live on the target, and a save should not
	// depend on that machine being reachable.
	if err := g.TLS.ValidateSettings(); err != nil {
		return err
	}
	probe := g
	probe.Networks = make([]catalog.GatewayNetwork, len(g.Networks))
	for i, n := range g.Networks {
		ups := make([]catalog.GatewayUpstream, len(n.Upstreams))
		copy(ups, n.Upstreams)
		for j := range ups {
			switch ups[j].KindOrDefault() {
			case catalog.UpstreamManagedNode, catalog.UpstreamManagedDevnet:
				if strings.TrimSpace(ups[j].TargetID) == "" {
					return fmt.Errorf("chain %d: a %s upstream must name the machine it refers to", n.ChainID, ups[j].KindOrDefault())
				}
				ups[j].Endpoint = "http://managed.invalid"
			case catalog.UpstreamExternal:
			default:
				return fmt.Errorf("chain %d: unknown upstream kind %q (want %q, %q or %q)",
					n.ChainID, ups[j].Kind, catalog.UpstreamManagedNode, catalog.UpstreamManagedDevnet, catalog.UpstreamExternal)
			}
		}
		probe.Networks[i] = catalog.GatewayNetwork{ChainID: n.ChainID, Upstreams: ups}
	}
	_, err := catalog.RenderGatewayConfig(probe)
	return err
}

// ---------------------------------------------------------------------
// DELETE /api/gateways/{gid}
// ---------------------------------------------------------------------

// handleGatewayDelete forgets a gateway. It deliberately does NOT remove the
// container: deleting a configuration and destroying a running service are
// different intents, and the wipe route is the one that means the second.
// The response says which happened so the operator is not left guessing.
func (s *Server) handleGatewayDelete(w http.ResponseWriter, r *http.Request) {
	gid := r.PathValue("gid")
	var removed config.Gateway
	if _, err := s.updateConfig(func(c *config.Config) error {
		for i := range c.Gateways {
			if c.Gateways[i].ID != gid {
				continue
			}
			removed = c.Gateways[i]
			c.Gateways = append(c.Gateways[:i], c.Gateways[i+1:]...)
			return nil
		}
		return fmt.Errorf("no gateway %q", gid)
	}); err != nil {
		writeErrorDetail(w, http.StatusNotFound, err.Error(), "", codeGatewayNotFound)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "removed",
		"note": fmt.Sprintf("valve-node-app has forgotten this gateway. Its container %q on machine %q was NOT touched — stop or wipe it before removing it if you wanted it gone.",
			ops.ERPCContainerNameFor(removed.ID), removed.Placement.TargetID),
	})
}

// ---------------------------------------------------------------------
// DELETE /api/orphans/{name}
// ---------------------------------------------------------------------

// handleOrphanDismiss forgets a leftover container record. It does NOT stop the
// container: this app does not stop containers it did not start, and the
// operator dismisses the record once they have wiped it themselves.
func (s *Server) handleOrphanDismiss(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if _, err := s.updateConfig(func(c *config.Config) error {
		for i := range c.Orphans {
			if c.Orphans[i].ContainerName == name {
				c.Orphans = append(c.Orphans[:i], c.Orphans[i+1:]...)
				return nil
			}
		}
		return fmt.Errorf("no leftover container %q", name)
	}); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "dismissed"})
}

// ---------------------------------------------------------------------
// POST /api/gateways/{gid}/{action}
// ---------------------------------------------------------------------

func (s *Server) handleGatewayAction(w http.ResponseWriter, r *http.Request) {
	action := r.PathValue("action")
	switch action {
	case actionStart, actionStop, actionRestart:
	default:
		writeError(w, http.StatusBadRequest, fmt.Sprintf(
			"unknown action %q (want %q, %q or %q — creating or re-creating a gateway is POST .../provision, and destroying its container is POST .../wipe)",
			action, actionStart, actionStop, actionRestart))
		return
	}

	cfg, gw, ok := s.gateway(w, r)
	if !ok {
		return
	}
	ex, _, ok := s.gatewayExecutor(w, cfg, gw)
	if !ok {
		return
	}

	resolved, _ := resolveGateway(cfg, gw)
	st, err := ops.ContainerAction(r.Context(), ex, setup.GatewayService(gw.ID, resolved), action)
	if err != nil {
		writeOpsError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, actionResponse{Status: st})
}

// gatewayExecutor resolves the executor for the machine a gateway is placed
// on, answering the operator-facing failure when that machine is gone.
func (s *Server) gatewayExecutor(w http.ResponseWriter, cfg config.Config, gw config.Gateway) (executor.Executor, config.Target, bool) {
	host, ok := findTarget(cfg, gw.Placement.TargetID)
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Sprintf(
			"this gateway is placed on machine %q, which is no longer registered", gw.Placement.TargetID))
		return nil, config.Target{}, false
	}
	ex, err := s.getExecutor(host)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return nil, config.Target{}, false
	}
	return ex, host, true
}

// ---------------------------------------------------------------------
// POST /api/gateways/{gid}/trust-cert
// ---------------------------------------------------------------------

// trustCertResult is the structured outcome of a trust-store install, so the
// UI can show success or fall back to the exact command for the operator to run
// by hand — the second being the ONLY option when the calling device is not the
// machine the gateway runs on.
type trustCertResult struct {
	// OK is true only when the install actually ran and succeeded on the target.
	OK bool `json:"ok"`
	// RanCommand is the command that was run — or, when OK is false, the one to
	// run by hand. It is echoed so a failed auto-install degrades into a
	// copy-paste rather than a dead end.
	RanCommand string `json:"ranCommand,omitempty"`
	Message    string `json:"message"`
}

// handleGatewayTrustCert installs THIS gateway's own internal-CA root into the
// trust store of the machine the gateway runs on, automating the one manual
// step a fronted gateway otherwise leaves the operator.
//
// SECURITY, and the reason this is safe to expose as a one-click: the only
// certificate it will ever install is the root the app itself exported for this
// gateway (rootCAPath, resolved HERE — never a path taken from the request), and
// only when that gateway is actually served by Caddy's internal CA. A gateway on
// files/ACME is trusted by whoever issued its certificate; there is nothing of
// ours to install, and installing an arbitrary file as a ROOT authority is
// precisely the thing not to make easy. The path is then validated and quoted by
// setup.TrustStoreCommand before it reaches a privileged command.
func (s *Server) handleGatewayTrustCert(w http.ResponseWriter, r *http.Request) {
	cfg, gw, ok := s.gateway(w, r)
	if !ok {
		return
	}
	ex, host, ok := s.gatewayExecutor(w, cfg, gw)
	if !ok {
		return
	}

	resolved, _ := resolveGateway(cfg, gw)
	if !resolved.Fronted() {
		writeErrorDetail(w, http.StatusBadRequest, setup.ErrNoTLSFront.Error(),
			"turn on “Serve HTTPS” in this gateway's settings and re-create it — there is no certificate to trust until it is fronted", codeNotConfigured)
		return
	}

	front, path, err := setup.GatewayTLSState(r.Context(), ex, gw.ID, resolved)
	if err != nil {
		writeOpsError(w, err)
		return
	}
	// front.CertSource is the EFFECTIVE source, so a files certificate that fell
	// back to the internal CA is (correctly) trustable here, while one actually
	// serving a files/ACME certificate is refused: there is no root of ours.
	if front.CertSource != catalog.CertInternal || path == "" {
		writeError(w, http.StatusBadRequest, fmt.Sprintf(
			"this gateway is served with the %q certificate source, not Caddy's own authority, so there is no internal root for valve-node-app to install — a files or ACME certificate is trusted by whoever issued it",
			front.CertSource))
		return
	}

	// Read the root back before installing it. We are about to add a ROOT
	// certificate authority to a trust store, so we install only bytes we can
	// see are a certificate this app exported — never merely whatever happens to
	// sit at that path now.
	pem, err := ex.ReadFile(r.Context(), path)
	if err != nil || !bytes.Contains(pem, []byte("BEGIN CERTIFICATE")) {
		writeError(w, http.StatusBadRequest, fmt.Sprintf(
			"the internal CA root has not been exported to %s on machine %q yet — create or re-create this gateway so its HTTPS front writes it, then trust it",
			path, host.ID))
		return
	}

	goos, err := targetGOOS(r.Context(), ex, host)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	install, err := setup.TrustStoreCommand(goos, path, gw.ID)
	if err != nil {
		// A GOOS we do not automate: hand the reason back with no command, so the
		// UI shows the copy-the-path fallback rather than a broken button.
		writeJSON(w, http.StatusOK, trustCertResult{OK: false, Message: err.Error()})
		return
	}

	// linux (and windows) need root and do not elevate on their own. If the
	// executor is not root, do not prompt for a password we cannot supply — hand
	// back the exact command to run with elevation instead.
	if install.NeedsRoot && !targetIsRoot(r.Context(), ex) {
		writeJSON(w, http.StatusOK, trustCertResult{
			OK:         false,
			RanCommand: install.Command,
			Message: fmt.Sprintf(
				"installing a root certificate needs root on machine %q. Run this on it (e.g. with sudo):", host.ID),
		})
		return
	}

	// The command to hand back when the auto-install can't run: prefer the
	// manual (sudo) form when there is one — darwin's osascript one-liner is
	// useless to paste into a plain shell, its whole point being a GUI prompt.
	manualCmd := install.ManualCommand
	if manualCmd == "" {
		manualCmd = install.Command
	}

	res, err := ex.Run(r.Context(), install.Command, nil)
	if err != nil {
		writeJSON(w, http.StatusOK, trustCertResult{
			OK:         false,
			RanCommand: manualCmd,
			Message:    fmt.Sprintf("the trust-store install could not be run on machine %q (%v). Run it there by hand:", host.ID, err),
		})
		return
	}
	if res.ExitCode != 0 {
		detail := strings.TrimSpace(res.Stderr)
		if detail == "" {
			detail = strings.TrimSpace(res.Stdout)
		}
		// darwin's osascript can only raise its authorization dialog inside a GUI
		// login session; when this app was launched detached it fails with
		// "no user interaction was possible", which reads as a scary internal
		// error rather than the fixable "run it in your own terminal" it is.
		msg := fmt.Sprintf("the trust-store install exited %d on machine %q: %s. Run it there by hand:", res.ExitCode, host.ID, detail)
		if goos == "darwin" {
			msg = fmt.Sprintf("macOS needs a GUI login session to prompt for authorization, which this app didn't have (it was likely launched detached — over SSH, a background service, or nohup). exit %d: %s. Run this in a Terminal you opened — it prompts for your password:", res.ExitCode, detail)
		}
		writeJSON(w, http.StatusOK, trustCertResult{
			OK:         false,
			RanCommand: manualCmd,
			Message:    msg,
		})
		return
	}

	writeJSON(w, http.StatusOK, trustCertResult{
		OK:         true,
		RanCommand: install.Command,
		Message:    fmt.Sprintf("Installed this gateway's root certificate into the trust store on machine %q. Reload your wallet or browser and the warning is gone.", host.ID),
	})
}

// targetGOOS is the operating system of the machine a gateway runs on, which
// decides how its root certificate is installed. A local target IS this
// process's own machine, so runtime.GOOS is exact; an SSH target is asked with
// uname (the same probe preflight uses).
func targetGOOS(ctx context.Context, ex executor.Executor, host config.Target) (string, error) {
	if host.Mode == "local" {
		return runtime.GOOS, nil
	}
	res, err := ex.Run(ctx, "uname", nil)
	if err != nil {
		return "", fmt.Errorf("could not ask machine %q what operating system it runs (uname): %w", host.ID, err)
	}
	switch strings.TrimSpace(res.Stdout) {
	case "Darwin":
		return "darwin", nil
	case "Linux":
		return "linux", nil
	default:
		return strings.ToLower(strings.TrimSpace(res.Stdout)), nil
	}
}

// targetIsRoot reports whether the executor runs as root on the target, so the
// trust-store install can choose between running itself and handing back a
// command to run with sudo.
func targetIsRoot(ctx context.Context, ex executor.Executor) bool {
	res, err := ex.Run(ctx, "id -u", nil)
	return err == nil && strings.TrimSpace(res.Stdout) == "0"
}

// ---------------------------------------------------------------------
// POST /api/gateways/{gid}/wipe
// ---------------------------------------------------------------------

func (s *Server) handleGatewayWipe(w http.ResponseWriter, r *http.Request) {
	gid := r.PathValue("gid")

	var req wipeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	// The typed confirmation is the gateway's own id, matching the devnet's
	// service name and the node's clear-and-resync: the token you type is the
	// name of the thing that goes.
	if req.Confirm != gid {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("confirm must equal the gateway id %q", gid))
		return
	}

	cfg, gw, ok := s.gateway(w, r)
	if !ok {
		return
	}
	ex, _, ok := s.gatewayExecutor(w, cfg, gw)
	if !ok {
		return
	}

	resolved, _ := resolveGateway(cfg, gw)
	dsvc := setup.GatewayService(gw.ID, resolved)
	rep, wipeErr := ops.WipeService(r.Context(), ex, dsvc)

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
// POST /api/gateways/{gid}/provision
// ---------------------------------------------------------------------

// handleGatewayProvision runs setup.PlanGateway through the SAME per-target
// setup run and SSE stream (POST /api/targets/{id}/setup, GET
// .../setup/stream) the node wizard and the devnet use. There is deliberately
// no second event-stream mechanism: the run slot is per MACHINE, and two
// plans driving one executor against one box is exactly how a container gets
// created against a config being rewritten underneath it.
//
// The run is claimed on the gateway's PLACEMENT target, which is the machine
// the work actually happens on — so the UI follows
// /api/targets/<placement>/setup/stream, and a gateway on box-a does not
// block a devnet provision on box-b.
func (s *Server) handleGatewayProvision(w http.ResponseWriter, r *http.Request) {
	cfg, gw, ok := s.gateway(w, r)
	if !ok {
		return
	}

	resolved, problems := resolveGateway(cfg, gw)
	if len(resolved.Networks) == 0 {
		// Redacted for the reason gatewayViewFor's Warnings are: these are the
		// same resolveUpstream strings, reaching the same browser.
		writeErrorDetail(w, http.StatusBadRequest,
			gatewayUnprovisionable(gw, redactEach(problems, providerKeys(cfg))), "", codeNotConfigured)
		return
	}

	steps, err := setup.PlanGateway(gw.ID, resolved, gw.Placement.Backend)
	if err != nil {
		writeErrorDetail(w, http.StatusBadRequest, err.Error(), "", codeNotConfigured)
		return
	}

	host, ok := findTarget(cfg, gw.Placement.TargetID)
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Sprintf(
			"this gateway is placed on machine %q, which is no longer registered", gw.Placement.TargetID))
		return
	}
	ex, err := s.getExecutor(host)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	claimed, ok := s.claimSetupRun(w, host.ID)
	if !ok {
		return
	}
	wire := catalog.WireConfig{}
	if host.Wire != nil {
		wire = *host.Wire
	}
	s.launchSetupRun(claimed, ex, steps, wire)

	writeJSON(w, http.StatusAccepted, map[string]string{
		"status": "started",
		// The stream is per-machine, so the caller is told which one to follow
		// rather than having to know the placement rule.
		"targetId": host.ID,
	})
}

// gatewayUnprovisionable explains, in one sentence, why there is nothing to
// create — naming the dead references when there are any, because "no chains"
// is misleading for a gateway that HAS chains whose every upstream is broken.
func gatewayUnprovisionable(gw config.Gateway, problems []string) string {
	if len(gw.Config.Networks) == 0 {
		return "this gateway serves no chains — eRPC refuses a configuration with no networks, so add a chain and at least one endpoint before creating it"
	}
	return "this gateway has chains but not one usable endpoint between them, so there is nothing to serve: " + strings.Join(problems, "; ")
}

// ---------------------------------------------------------------------
// GET /api/gateways/{gid}/tls/verify
// ---------------------------------------------------------------------

// tlsVerifyTimeout bounds one whole run. It is generous because the run
// deliberately waits for a block on the subscription, and stingy enough that a
// wedged front cannot hold a request open indefinitely.
const tlsVerifyTimeout = 60 * time.Second

// handleGatewayTLSVerify answers "is this gateway ACTUALLY serving HTTPS?"
// with evidence rather than a boolean.
//
// It re-measures every time it is called — a certificate expires on a wall
// clock, an upstream is re-pointed without anything here changing — and caches
// the answer so the RPC screen can show the last result beside the front
// without re-running the whole thing on every poll.
func (s *Server) handleGatewayTLSVerify(w http.ResponseWriter, r *http.Request) {
	cfg, gw, ok := s.gateway(w, r)
	if !ok {
		return
	}
	resolved, _ := resolveGateway(cfg, gw)
	if !resolved.Fronted() {
		writeErrorDetail(w, http.StatusBadRequest, setup.ErrNoTLSFront.Error(),
			"turn on “Serve HTTPS” in this gateway's settings and re-create it, then there will be something to verify", codeNotConfigured)
		return
	}

	ex, host, ok := s.gatewayExecutor(w, cfg, gw)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), tlsVerifyTimeout)
	defer cancel()

	res, err := s.verifyTLS(ctx, ex, gw.ID, resolved, tlsDialHost(host, resolved.TLS))
	if err != nil {
		status, hint, code := classifyOpsError(err)
		writeErrorDetail(w, status, err.Error(), hint, code)
		return
	}
	s.storeTLSVerification(gw.ID, res)
	writeJSON(w, http.StatusOK, res)
}

// tlsDialHost is the address the gateway's hostname is pinned to, which is the
// same choice the provisioner's curl --resolve makes and for the same reason:
// the check is about what the front serves, not about whether the operator has
// pointed DNS at their machine yet.
//
// For a local machine that is the published bind (0.0.0.0 → loopback). For an
// SSH target it is the host this app reaches it on — the one address that is
// known to route there from here.
func tlsDialHost(host config.Target, tls *catalog.GatewayTLS) string {
	if host.Mode == "ssh" && host.SSH != nil && strings.TrimSpace(host.SSH.Host) != "" {
		return strings.TrimSpace(host.SSH.Host)
	}
	return endpointHost(tls.Bind())
}

// storeTLSVerification / lastTLSVerification keep the most recent run per
// gateway so the status view can show it. Nothing expires it: a result with a
// timestamp on it is more use than no result, and the UI says when it ran.
func (s *Server) storeTLSVerification(gid string, v setup.TLSVerification) {
	s.tlsMu.Lock()
	defer s.tlsMu.Unlock()
	if s.tlsChecks == nil {
		s.tlsChecks = map[string]setup.TLSVerification{}
	}
	s.tlsChecks[gid] = v
}

func (s *Server) lastTLSVerification(gid string) *setup.TLSVerification {
	s.tlsMu.Lock()
	defer s.tlsMu.Unlock()
	v, ok := s.tlsChecks[gid]
	if !ok {
		return nil
	}
	return &v
}

// ---------------------------------------------------------------------
// the default TLS hostname
// ---------------------------------------------------------------------

// suggestedTLSHostname is the name a gateway gets by default, under a domain
// whose wildcard resolves to loopback (catalog.DefaultTLSDomain).
//
// It is per-INSTALL rather than a single shared name. The wildcard supports it
// at no cost, and it means two machines that both turn HTTPS on do not end up
// serving different certificates for the same name — which is confusing
// exactly when someone is debugging a trust-store install.
func suggestedTLSHostname(gatewayID string) string {
	return catalog.DefaultTLSHostname(gatewayID, installSeed())
}

// defaultTLSHostname fills in a missing TLS hostname on a config being stored,
// leaving everything else — including whether HTTPS is on at all — alone.
func defaultTLSHostname(g *catalog.GatewayConfig, gatewayID string) {
	if g.TLS == nil {
		g.TLS = &catalog.GatewayTLS{Hostname: suggestedTLSHostname(gatewayID)}
		return
	}
	if strings.TrimSpace(g.TLS.Hostname) == "" {
		g.TLS.Hostname = suggestedTLSHostname(gatewayID)
	}
}

// installSeed identifies THIS install. The machine's own hostname plus the
// state directory is enough: it is stable across restarts, needs nothing
// stored, and two installs on one machine (different $HOME) still differ.
var installSeed = sync.OnceValue(func() string {
	name, _ := os.Hostname()
	dir, _ := config.Dir()
	return name + "\x00" + dir
})

// ---------------------------------------------------------------------
// GET /api/chainlist/{chainId}
// ---------------------------------------------------------------------

// (implemented in chainlist.go)
