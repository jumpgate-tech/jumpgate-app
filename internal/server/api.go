// This file implements the JSON/SSE API: target CRUD, the setup wizard
// kickoff + progress stream, per-target monitor/log streams, AI log
// explanations, and provider/key settings. Every route here is mounted
// under the authMiddleware in server.go's Handler, so nothing in this file
// needs to re-check the session token.
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/valve-tech/valve-node-app/internal/ai"
	"github.com/valve-tech/valve-node-app/internal/catalog"
	"github.com/valve-tech/valve-node-app/internal/config"
	"github.com/valve-tech/valve-node-app/internal/executor"
	"github.com/valve-tech/valve-node-app/internal/logwatch"
	"github.com/valve-tech/valve-node-app/internal/monitor"
	"github.com/valve-tech/valve-node-app/internal/ops"
	"github.com/valve-tech/valve-node-app/internal/setup"
)

// logUnits is the fixed set of journald units every target's logwatch
// Watcher tails, matching internal/monitor's unit-name constants.
var logUnits = []string{"valve-node-app-exec.service", "valve-node-app-beacon.service"}

// defaultRecentLogs is used when GET .../logs is called without ?n=.
const defaultRecentLogs = 200

// ---------------------------------------------------------------------
// registry: per-target runtime state (executor, monitor, log watcher,
// setup run), created lazily on first use and kept for the life of the
// Server process (or until the target is deleted).
// ---------------------------------------------------------------------

type registry struct {
	mu      sync.Mutex
	entries map[string]*targetEntry
}

func newRegistry() *registry {
	return &registry{entries: map[string]*targetEntry{}}
}

// get returns the entry for id, creating an empty one if this is the first
// time id has been seen.
func (r *registry) get(id string) *targetEntry {
	r.mu.Lock()
	defer r.mu.Unlock()
	e, ok := r.entries[id]
	if !ok {
		e = &targetEntry{}
		r.entries[id] = e
	}
	return e
}

// setupCancelWait bounds how long registry.remove waits for an in-flight
// setup run to observe cancellation and stop touching the target's
// executor before remove closes that executor out from under it.
const setupCancelWait = 5 * time.Second

// remove evicts id's entry, stopping its monitor/watcher goroutines,
// canceling and waiting (bounded) for any in-flight setup run, and only
// then closing its cached executor — closing the executor before an
// in-flight setup.RunAll goroutine has actually stopped using it would be a
// use-after-close race.
func (r *registry) remove(id string) {
	r.mu.Lock()
	e, ok := r.entries[id]
	delete(r.entries, id)
	r.mu.Unlock()
	if !ok {
		return
	}

	e.mu.Lock()
	if e.monStop != nil {
		e.monStop()
	}
	if e.watchStop != nil {
		e.watchStop()
	}
	run := e.setup
	e.mu.Unlock()

	if run != nil {
		run.cancelAndWait(setupCancelWait)
	}

	e.mu.Lock()
	defer e.mu.Unlock()
	if e.exec != nil {
		e.exec.Close()
	}
}

type targetEntry struct {
	mu sync.Mutex

	exec executor.Executor

	mon     *monitor.Monitor
	monStop context.CancelFunc

	watch     *logwatch.Watcher
	watchStop context.CancelFunc

	setup *setupRun

	// Network-diagnostics state, guarded by its own mutex because auto-run
	// goroutines touch it while entry.mu may be held by slow executor
	// dials. See diag.go for the gate semantics.
	diagMu     sync.Mutex
	diagLatest *DiagReport
	diagLast   time.Time
	diagBusy   bool
}

// setExec caches ex as entry's executor under entry.mu. handleAddTarget
// dials an executor before entry.mu is ever taken (it doesn't need the
// entry until this point), so it must go through this locked setter rather
// than writing entry.exec directly — otherwise it races every other path
// (getExecutorLocked, registry.remove) that touches the same field under
// the lock. If a concurrent getExecutorLocked call for the same id already
// cached one first, the redundant dial is closed and the existing one wins.
func (e *targetEntry) setExec(ex executor.Executor) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.exec != nil {
		ex.Close()
		return
	}
	e.exec = ex
}

// setupRun tracks one setup.RunAll invocation for a target: every event it
// has emitted so far (so a new SSE subscriber can replay history) plus the
// live subscriber set.
type setupRun struct {
	mu      sync.Mutex
	events  []setup.Event
	subs    map[chan setup.Event]struct{}
	running bool
	err     error

	// cancel and done let registry.remove interrupt an in-flight run and
	// wait (bounded) for its goroutine to actually stop touching the
	// target's executor before Close()ing it — see cancelAndWait. done is
	// closed by the setup goroutine once setup.RunAll has returned.
	cancel context.CancelFunc
	done   chan struct{}
}

func newSetupRun(cancel context.CancelFunc) *setupRun {
	return &setupRun{
		subs:    map[chan setup.Event]struct{}{},
		running: true,
		cancel:  cancel,
		done:    make(chan struct{}),
	}
}

// cancelAndWait cancels the run's context, then blocks for up to timeout
// waiting for the run's goroutine to finish (signaled by done being
// closed) — so a caller about to Close the run's executor can be sure it's
// no longer in use, without risking an unbounded wait if the run's step
// ignores cancellation somehow.
func (sr *setupRun) cancelAndWait(timeout time.Duration) {
	sr.mu.Lock()
	cancel := sr.cancel
	done := sr.done
	sr.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if done == nil {
		return
	}
	select {
	case <-done:
	case <-time.After(timeout):
	}
}

func (sr *setupRun) append(ev setup.Event) {
	sr.mu.Lock()
	sr.events = append(sr.events, ev)
	for ch := range sr.subs {
		select {
		case ch <- ev:
		default:
			// Slow consumer — drop, matching monitor/logwatch's Subscribe
			// contract. The replay-on-connect behavior means a dropped live
			// tick is never permanently lost to a *new* subscriber anyway.
		}
	}
	sr.mu.Unlock()
}

func (sr *setupRun) finish(err error) {
	sr.mu.Lock()
	sr.running = false
	sr.err = err
	sr.mu.Unlock()
}

// subscribe returns every event emitted so far plus a channel that
// receives every subsequent one, registered atomically so no event can be
// missed or duplicated across the snapshot/live-feed boundary.
func (sr *setupRun) subscribe() ([]setup.Event, chan setup.Event, func()) {
	sr.mu.Lock()
	defer sr.mu.Unlock()
	snapshot := append([]setup.Event(nil), sr.events...)
	ch := make(chan setup.Event, 32)
	sr.subs[ch] = struct{}{}
	unsub := func() {
		sr.mu.Lock()
		delete(sr.subs, ch)
		sr.mu.Unlock()
	}
	return snapshot, ch, unsub
}

// ---------------------------------------------------------------------
// executor construction
// ---------------------------------------------------------------------

// defaultNewExecutor is Server.newExecutor's default: a real local or SSH
// executor depending on Target.Mode.
func defaultNewExecutor(t config.Target) (executor.Executor, error) {
	switch t.Mode {
	case "local":
		// Local mode drives this machine with POSIX shell commands, so a
		// control plane without a POSIX shell (Windows) cannot support it.
		// Refuse here, at construction, so POST /targets answers with an
		// actionable message instead of the target being persisted and then
		// failing on every command it ever runs.
		if err := executor.LocalAvailable(); err != nil {
			return nil, fmt.Errorf("target %q: %w", t.ID, err)
		}
		return executor.NewLocal(), nil
	case "ssh":
		if t.SSH == nil {
			return nil, fmt.Errorf("target %q: mode \"ssh\" requires an ssh config", t.ID)
		}
		return executor.NewSSH(*t.SSH)
	default:
		return nil, fmt.Errorf("target %q: unknown mode %q", t.ID, t.Mode)
	}
}

// getExecutor returns t's cached executor, dialing and caching a new one on
// first use.
func (s *Server) getExecutor(t config.Target) (executor.Executor, error) {
	entry := s.reg.get(t.ID)
	entry.mu.Lock()
	defer entry.mu.Unlock()
	return s.getExecutorLocked(entry, t)
}

// getExecutorLocked is getExecutor's body, for callers that already hold
// entry.mu (avoids re-entrant locking).
func (s *Server) getExecutorLocked(entry *targetEntry, t config.Target) (executor.Executor, error) {
	if entry.exec != nil {
		return entry.exec, nil
	}
	ex, err := s.newExecutor(t)
	if err != nil {
		return nil, err
	}
	entry.exec = ex
	return ex, nil
}

// getMonitor returns t's monitor.Monitor, lazily creating and starting one
// (polling forever, until the target is deleted) on first use.
func (s *Server) getMonitor(t config.Target, refRPCBase string) (*monitor.Monitor, error) {
	entry := s.reg.get(t.ID)
	entry.mu.Lock()
	defer entry.mu.Unlock()
	if entry.mon != nil {
		return entry.mon, nil
	}
	ex, err := s.getExecutorLocked(entry, t)
	if err != nil {
		return nil, err
	}
	refRPC := ""
	if refRPCBase != "" {
		refRPC = fmt.Sprintf("%s/evm/%d", refRPCBase, t.Wire.ChainID)
	}
	mon := monitor.New(monitor.Config{Exec: ex, Wire: *t.Wire, RefRPC: refRPC})
	ctx, cancel := context.WithCancel(context.Background())
	mon.Start(ctx)
	// Auto-diagnostics trigger: failed connections (inactive service, zero
	// peers) in this monitor's snapshots kick off a background diagnostics
	// run, gated by the per-target cooldown (diag.go).
	go s.watchMonitorForDiag(ctx, t, mon)
	entry.mon = mon
	entry.monStop = cancel
	return mon, nil
}

// getWatcher returns t's logwatch.Watcher, lazily creating and starting one
// (tailing forever, until the target is deleted) on first use.
func (s *Server) getWatcher(t config.Target) (*logwatch.Watcher, error) {
	entry := s.reg.get(t.ID)
	entry.mu.Lock()
	defer entry.mu.Unlock()
	if entry.watch != nil {
		return entry.watch, nil
	}
	ex, err := s.getExecutorLocked(entry, t)
	if err != nil {
		return nil, err
	}
	watch := logwatch.New(ex, logUnits)
	ctx, cancel := context.WithCancel(context.Background())
	watch.Start(ctx)
	// Auto-diagnostics trigger: error/critical journal hits kick off a
	// background diagnostics run, gated by the per-target cooldown
	// (diag.go).
	go s.watchLogsForDiag(ctx, t, watch)
	entry.watch = watch
	entry.watchStop = cancel
	return watch, nil
}

// ---------------------------------------------------------------------
// config helpers
// ---------------------------------------------------------------------

func (s *Server) loadConfig() (config.Config, error) {
	s.cfgMu.Lock()
	defer s.cfgMu.Unlock()
	return config.Load()
}

// updateConfig loads the config, applies fn, saves it, and returns the
// saved result — all under cfgMu so concurrent API requests can't clobber
// each other's edits.
func (s *Server) updateConfig(fn func(c *config.Config) error) (config.Config, error) {
	s.cfgMu.Lock()
	defer s.cfgMu.Unlock()
	c, err := config.Load()
	if err != nil {
		return config.Config{}, err
	}
	if err := fn(&c); err != nil {
		return config.Config{}, err
	}
	if err := c.Save(); err != nil {
		return config.Config{}, err
	}
	return c, nil
}

func findTarget(cfg config.Config, id string) (config.Target, bool) {
	for _, t := range cfg.Targets {
		if t.ID == id {
			return t, true
		}
	}
	return config.Target{}, false
}

// ---------------------------------------------------------------------
// JSON helpers
// ---------------------------------------------------------------------

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// writeSSEEvent marshals v and writes it as one `data: <json>\n\n` SSE
// frame. Marshal failures are dropped silently — there is no way to report
// an error mid-stream that wouldn't also break the stream framing.
func writeSSEEvent(w http.ResponseWriter, v any) {
	b, err := json.Marshal(v)
	if err != nil {
		return
	}
	fmt.Fprintf(w, "data: %s\n\n", b)
}

// sseHeaders opens an event stream and FLUSHES, so the client learns it is
// connected now rather than when the first event happens.
//
// The flush is load-bearing, not tidiness. net/http buffers a response until
// something flushes it, so without this a browser's EventSource sits in its
// connecting state — and onerror, not onopen — until the first log line or
// step event arrives, which on a healthy quiet machine can be a long time.
// From the UI's side that is indistinguishable from a stream that is broken.
func sseHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}

// ---------------------------------------------------------------------
// routes
// ---------------------------------------------------------------------

func (s *Server) registerAPIRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/catalog", s.handleCatalog)
	mux.HandleFunc("GET /api/host", s.handleHost)

	mux.HandleFunc("GET /api/targets", s.handleListTargets)
	mux.HandleFunc("POST /api/targets", s.handleAddTarget)
	mux.HandleFunc("DELETE /api/targets/{id}", s.handleDeleteTarget)

	mux.HandleFunc("POST /api/targets/{id}/setup", s.handleStartSetup)
	mux.HandleFunc("GET /api/targets/{id}/setup/stream", s.handleSetupStream)

	mux.HandleFunc("GET /api/targets/{id}/monitor/stream", s.handleTargetMonitorStream)

	mux.HandleFunc("GET /api/targets/{id}/logs", s.handleLogs)
	mux.HandleFunc("GET /api/targets/{id}/logs/stream", s.handleLogsStream)

	mux.HandleFunc("POST /api/targets/{id}/explain", s.handleExplain)

	// The literal "clear" segment is more specific than the {action}
	// wildcard below it and wins for an exact match — Go 1.22+ ServeMux
	// prefers the more specific pattern regardless of registration order —
	// so these two don't collide.
	mux.HandleFunc("POST /api/targets/{id}/services/{svc}/clear", s.handleServiceClear)
	mux.HandleFunc("POST /api/targets/{id}/services/{svc}/{action}", s.handleServiceAction)
	mux.HandleFunc("GET /api/targets/{id}/du", s.handleDiskUsage)
	mux.HandleFunc("GET /api/targets/{id}/disk", s.handleDiskFree)
	mux.HandleFunc("GET /api/targets/{id}/endpoints", s.handleEndpoints)
	mux.HandleFunc("GET /api/targets/{id}/firewall", s.handleFirewall)
	mux.HandleFunc("GET /api/targets/{id}/diagnostics", s.handleDiagnostics)
	mux.HandleFunc("GET /api/targets/{id}/diagnostics/latest", s.handleDiagnosticsLatest)

	// The devnet, which belongs to a machine — see containers.go.
	s.registerContainerRoutes(mux)

	// The eRPC gateways, which do NOT belong to a machine: they are a layer
	// over the whole fleet and are addressed top-level — see gateways.go.
	s.registerGatewayRoutes(mux)

	// The WireGuard overlays (bring-your-own VPN), addressed top-level like
	// gateways: an overlay names the host it runs ON, rather than belonging to
	// one — see vpn.go.
	s.registerVPNRoutes(mux)

	// The provisioned WireGuard servers (the "pick a device" easy button) and
	// the devices enrolled on them — see vpnserver.go.
	s.registerVPNServerRoutes(mux)

	mux.HandleFunc("GET /api/settings", s.handleGetSettings)
	mux.HandleFunc("PUT /api/settings", s.handlePutSettings)
}

// ---------------------------------------------------------------------
// GET /api/catalog
// ---------------------------------------------------------------------

type catalogClient struct {
	ID                string `json:"id"`
	Kind              string `json:"kind"`
	Repo              string `json:"repo"`
	PinVersion        string `json:"pinVersion"`
	Toolchain         string `json:"toolchain"`
	LearnURL          string `json:"learnUrl"`
	SnapshotSupported bool   `json:"snapshotSupported"`
}

type catalogResponse struct {
	Networks []catalog.Network `json:"networks"`
	Clients  []catalogClient   `json:"clients"`
}

// handleHost reports the OS/arch valve-node-app itself is running on. The
// targets UI uses this — not the browser's platform — to decide whether
// local setup (running a node on this same machine) is viable: setup needs
// a Linux host, so on darwin/windows this machine is a controller only.
func (s *Server) handleHost(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, struct {
		OS   string `json:"os"`
		Arch string `json:"arch"`
	}{OS: runtime.GOOS, Arch: runtime.GOARCH})
}

// handleCatalog returns every known network plus every client referenced by
// any of them. catalog.Client isn't itself JSON-safe (ReleaseURL is a
// func), and catalog exposes no "all clients" listing, so the client set is
// derived from the ids each Network's ExecClients/BeaconClients names.
func (s *Server) handleCatalog(w http.ResponseWriter, r *http.Request) {
	networks := catalog.Networks()

	seen := map[string]struct{}{}
	for _, n := range networks {
		for _, id := range n.ExecClients {
			seen[id] = struct{}{}
		}
		for _, id := range n.BeaconClients {
			seen[id] = struct{}{}
		}
	}
	ids := make([]string, 0, len(seen))
	for id := range seen {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	clients := make([]catalogClient, 0, len(ids))
	for _, id := range ids {
		c, ok := catalog.ClientByID(id)
		if !ok {
			continue
		}
		clients = append(clients, catalogClient{
			ID:                c.ID,
			Kind:              c.Kind,
			Repo:              c.Repo,
			PinVersion:        c.PinVersion,
			Toolchain:         c.Toolchain,
			LearnURL:          c.LearnURL,
			SnapshotSupported: c.SnapshotSupported,
		})
	}

	writeJSON(w, http.StatusOK, catalogResponse{Networks: networks, Clients: clients})
}

// ---------------------------------------------------------------------
// targets: list / add / delete
// ---------------------------------------------------------------------

func (s *Server) handleListTargets(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.loadConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	targets := cfg.Targets
	if targets == nil {
		targets = []config.Target{}
	}
	writeJSON(w, http.StatusOK, targets)
}

func (s *Server) handleAddTarget(w http.ResponseWriter, r *http.Request) {
	var t config.Target
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	// Wire is only ever set by the setup wizard (POST .../setup), never
	// accepted directly from a client here.
	t.Wire = nil

	if t.ID == "" {
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}
	if t.Mode != "local" && t.Mode != "ssh" {
		writeError(w, http.StatusBadRequest, `mode must be "local" or "ssh"`)
		return
	}
	if t.Mode == "ssh" {
		if t.SSH == nil || t.SSH.Host == "" || t.SSH.User == "" || t.SSH.KeyPath == "" {
			writeError(w, http.StatusBadRequest, "ssh mode requires ssh.host, ssh.user, and ssh.keyPath")
			return
		}
		if t.SSH.HostKeyFile == "" {
			dir, err := config.Dir()
			if err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			// LOCAL path: known_hosts lives in the operator's own config dir
			// on the control plane and is read/written with os.ReadFile, so
			// filepath (host separator) is correct here — unlike the target
			// paths below, which are always POSIX.
			t.SSH.HostKeyFile = filepath.Join(dir, "known_hosts")
		}
	}

	existing, err := s.loadConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if _, ok := findTarget(existing, t.ID); ok {
		writeError(w, http.StatusConflict, fmt.Sprintf("target %q already exists", t.ID))
		return
	}

	// Dial now (SSH TOFU happens on this call) to validate the target is
	// reachable before persisting it.
	ex, err := s.newExecutor(t)
	if err != nil {
		writeError(w, http.StatusBadGateway, fmt.Sprintf("could not reach target: %v", err))
		return
	}

	cfg, err := s.updateConfig(func(c *config.Config) error {
		if _, ok := findTarget(*c, t.ID); ok {
			return fmt.Errorf("target %q already exists", t.ID)
		}
		c.Targets = append(c.Targets, t)
		return nil
	})
	if err != nil {
		ex.Close()
		writeError(w, http.StatusConflict, err.Error())
		return
	}

	s.reg.get(t.ID).setExec(ex)

	added, _ := findTarget(cfg, t.ID)
	writeJSON(w, http.StatusCreated, added)
}

func (s *Server) handleDeleteTarget(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	found := false
	_, err := s.updateConfig(func(c *config.Config) error {
		out := c.Targets[:0]
		for _, t := range c.Targets {
			if t.ID == id {
				found = true
				continue
			}
			out = append(out, t)
		}
		c.Targets = out
		return nil
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !found {
		writeError(w, http.StatusNotFound, "target not found")
		return
	}

	s.reg.remove(id)
	w.WriteHeader(http.StatusNoContent)
}

// ---------------------------------------------------------------------
// setup kickoff + SSE progress stream
// ---------------------------------------------------------------------

// validateWirePorts rejects any of WireConfig's port fields that fall
// outside 0..65535 — 0 means "use the default" (see catalog.WireConfig),
// so it's the only value below 1 that's allowed. The wizard UI validates
// the same range client-side, but the server can't trust that.
func validateWirePorts(wire catalog.WireConfig) error {
	ports := []struct {
		name string
		port int
	}{
		{"ExecHTTPPort", wire.ExecHTTPPort},
		{"BeaconHTTPPort", wire.BeaconHTTPPort},
		{"ExecP2PPort", wire.ExecP2PPort},
	}
	for _, p := range ports {
		if p.port < 0 || p.port > 65535 {
			return fmt.Errorf("%s: %d is out of range (must be 0 for default, or 1-65535)", p.name, p.port)
		}
	}
	if addr := wire.RPCBindAddr; addr != "" && net.ParseIP(addr) == nil {
		return fmt.Errorf("RPCBindAddr: %q is not a valid IP address (leave empty for loopback, or set a host IP such as your Tailscale address)", addr)
	}
	if cp := wire.CheckpointURL; cp != "" {
		u, err := url.Parse(cp)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return fmt.Errorf("CheckpointURL: %q must be an http(s) URL (leave empty for the network default)", cp)
		}
	}
	if wire.ExecSnapshot {
		client, ok := catalog.ClientByID(wire.ExecID)
		if !ok || !client.SnapshotSupported {
			return fmt.Errorf("ExecSnapshot: execution client %q does not support snapshot restore (reth only)", wire.ExecID)
		}
		if strings.TrimSpace(wire.SnapshotKey) == "" {
			return fmt.Errorf("ExecSnapshot: a free snapshot key (get one at valve.city) is required to restore from Valve's snapshot")
		}
		// The key is interpolated into the reth-download command (and the
		// manifest URL) run on the target, so constrain it to a safe shape
		// (vk_ prefix + url-safe chars) at the boundary — no shell
		// metacharacters can reach command construction.
		if !snapshotKeyPattern.MatchString(wire.SnapshotKey) {
			// The LENGTH is named as well as the character set. A truncated
			// paste is at least as common as a wrong one, and a message that
			// lists only the legal characters sends that operator looking for
			// an illegal character which is not there.
			return fmt.Errorf("SnapshotKey: %q is not a valid key (expected vk_ followed by %d-%d letters, digits, - or _)",
				wire.SnapshotKey, snapshotKeyMinLen, snapshotKeyMaxLen)
		}
	}
	return nil
}

// snapshotKeyPattern is the accepted shape for a Valve snapshot key: a "vk_"
// prefix followed by url-safe characters only (no whitespace or shell
// metacharacters).
//
// The bounds are named constants and the pattern is built from them so the
// rejection message cannot drift from the rule it is describing — which is
// exactly what had happened: the message listed the character set and never
// mentioned the length.
const (
	snapshotKeyMinLen = 8
	snapshotKeyMaxLen = 128
)

var snapshotKeyPattern = regexp.MustCompile(
	fmt.Sprintf(`^vk_[A-Za-z0-9_-]{%d,%d}$`, snapshotKeyMinLen, snapshotKeyMaxLen))

// defaultDataDir and defaultJWTPath name locations on the TARGET, which is
// always a Linux host — never on the control plane, which may be macOS,
// Windows, or Linux. Target paths are therefore POSIX and must be built with
// "path": filepath.Join on a Windows control plane would yield
// `\var\lib\valve-node-app\369\jwt.hex`, which then gets baked into the
// systemd units and every remote `mkdir -p`, breaking setup on every target.
// (Contrast HostKeyFile in handleAddTarget, which IS a control-plane path and
// correctly uses filepath.) They are split out as functions so the rule is
// unit-testable on any host OS.
func defaultDataDir(chainID int) string {
	return fmt.Sprintf("/var/lib/valve-node-app/%d", chainID)
}

func defaultJWTPath(dataDir string) string {
	return path.Join(dataDir, "jwt.hex")
}

func (s *Server) handleStartSetup(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var wire catalog.WireConfig
	if err := json.NewDecoder(r.Body).Decode(&wire); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := validateWirePorts(wire); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if wire.DataDir == "" {
		wire.DataDir = defaultDataDir(wire.ChainID)
	}
	if wire.JWTPath == "" {
		wire.JWTPath = defaultJWTPath(wire.DataDir)
	}

	steps, err := setup.Plan(wire)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	cfg, err := s.loadConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	target, ok := findTarget(cfg, id)
	if !ok {
		writeError(w, http.StatusNotFound, "target not found")
		return
	}

	ex, err := s.getExecutor(target)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	claimed, ok := s.claimSetupRun(w, id)
	if !ok {
		return
	}

	// The wizard "has run" as soon as setup is kicked off, even if it fails
	// partway — the engine is idempotent (each step's Verify is also its
	// "already done" probe), so re-kicking setup against the same target
	// resumes rather than restarts. Persisting immediately also means the
	// UI can show what a target is (or was) being configured with even
	// while the run is still in flight.
	if _, err := s.updateConfig(func(c *config.Config) error {
		for i := range c.Targets {
			if c.Targets[i].ID == id {
				wireCopy := wire
				c.Targets[i].Wire = &wireCopy
			}
		}
		return nil
	}); err != nil {
		// Undo the "running" mark: the run never actually started, so a
		// retry must not be told setup is already in progress.
		s.releaseSetupRun(id, claimed)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.launchSetupRun(claimed, ex, steps, wire)

	writeJSON(w, http.StatusAccepted, map[string]string{"status": "started"})
}

// claimedRun is one reserved setup slot: the run every subscriber of
// .../setup/stream will see, plus the context that cancels it.
type claimedRun struct {
	id     string
	run    *setupRun
	ctx    context.Context
	cancel context.CancelFunc
}

// claimSetupRun reserves the target's single setup slot, answering 409 if one
// is already in flight and returning false when it has already written a
// response.
//
// There is exactly one slot per target, shared by the node wizard and by the
// container services' provisioning (containers.go), because they all drive
// the same executor against the same machine — and because the SSE stream
// that reports progress is likewise per-target. Two runs interleaving would
// produce one event stream describing two different things.
func (s *Server) claimSetupRun(w http.ResponseWriter, id string) (claimedRun, bool) {
	entry := s.reg.get(id)
	entry.mu.Lock()
	defer entry.mu.Unlock()
	if entry.setup != nil && entry.setup.running {
		writeError(w, http.StatusConflict, "setup is already running for this target")
		return claimedRun{}, false
	}
	ctx, cancel := context.WithCancel(context.Background())
	run := newSetupRun(cancel)
	entry.setup = run
	return claimedRun{id: id, run: run, ctx: ctx, cancel: cancel}, true
}

// releaseSetupRun undoes a claim whose run never started, so a retry is not
// told setup is already in progress.
func (s *Server) releaseSetupRun(id string, c claimedRun) {
	entry := s.reg.get(id)
	entry.mu.Lock()
	if entry.setup == c.run {
		entry.setup = nil
	}
	entry.mu.Unlock()
	c.cancel()
}

// launchSetupRun runs steps in the background, feeding every event into the
// claimed run so .../setup/stream can replay and follow it.
func (s *Server) launchSetupRun(c claimedRun, ex executor.Executor, steps []setup.Step, wire catalog.WireConfig) {
	events := make(chan setup.Event, 32)
	go func() {
		for ev := range events {
			c.run.append(ev)
		}
	}()
	go func() {
		defer close(events)
		runErr := setup.RunAll(c.ctx, ex, steps, &setup.State{Wire: wire, Events: events})
		c.run.finish(runErr)
		// Signal that this goroutine is done touching ex — registry.remove
		// waits on this before Close()ing the executor out from under a
		// still-running setup step.
		close(c.run.done)
	}()
}

func (s *Server) handleSetupStream(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	entry := s.reg.get(id)
	entry.mu.Lock()
	run := entry.setup
	entry.mu.Unlock()
	if run == nil {
		writeError(w, http.StatusNotFound, "no setup run has been started for this target")
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming unsupported")
		return
	}

	sseHeaders(w)

	snapshot, ch, unsub := run.subscribe()
	defer unsub()

	for _, ev := range snapshot {
		writeSSEEvent(w, ev)
	}
	flusher.Flush()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-ch:
			if !ok {
				return
			}
			writeSSEEvent(w, ev)
			flusher.Flush()
		}
	}
}

// ---------------------------------------------------------------------
// per-target monitor stream
// ---------------------------------------------------------------------

func (s *Server) handleTargetMonitorStream(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	cfg, err := s.loadConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	target, ok := findTarget(cfg, id)
	if !ok {
		writeError(w, http.StatusNotFound, "target not found")
		return
	}
	if target.Wire == nil {
		writeError(w, http.StatusConflict, "target has not completed setup")
		return
	}

	mon, err := s.getMonitor(target, cfg.RefRPCBase)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming unsupported")
		return
	}

	sseHeaders(w)

	ch, unsub := mon.Subscribe()
	defer unsub()

	writeSSEEvent(w, mon.Latest())
	flusher.Flush()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case snap, ok := <-ch:
			if !ok {
				return
			}
			writeSSEEvent(w, snap)
			flusher.Flush()
		}
	}
}

// ---------------------------------------------------------------------
// logs: recent + SSE tail
// ---------------------------------------------------------------------

func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	cfg, err := s.loadConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	target, ok := findTarget(cfg, id)
	if !ok {
		writeError(w, http.StatusNotFound, "target not found")
		return
	}
	if target.Wire == nil {
		writeError(w, http.StatusConflict, "target has not completed setup")
		return
	}

	watch, err := s.getWatcher(target)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	n := defaultRecentLogs
	if raw := r.URL.Query().Get("n"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			n = parsed
		}
	}

	hits := watch.Recent(n)
	if hits == nil {
		hits = []logwatch.Hit{}
	}
	writeJSON(w, http.StatusOK, hits)
}

func (s *Server) handleLogsStream(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	cfg, err := s.loadConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	target, ok := findTarget(cfg, id)
	if !ok {
		writeError(w, http.StatusNotFound, "target not found")
		return
	}
	if target.Wire == nil {
		writeError(w, http.StatusConflict, "target has not completed setup")
		return
	}

	watch, err := s.getWatcher(target)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming unsupported")
		return
	}

	sseHeaders(w)

	ch, unsub := watch.Subscribe()
	defer unsub()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case hit, ok := <-ch:
			if !ok {
				return
			}
			writeSSEEvent(w, hit)
			flusher.Flush()
		}
	}
}

// ---------------------------------------------------------------------
// explain
// ---------------------------------------------------------------------

// maxDefaultExplainHits caps how many recent error/critical log hits are
// sent to the AI provider when the caller doesn't supply explicit lines.
const maxDefaultExplainHits = 40

type explainRequest struct {
	Lines []string `json:"lines,omitempty"`
}

type explainResponse struct {
	Text string `json:"text"`
	// SentExcerpt is exactly the lines that were sent to the provider, so
	// the UI can show the operator what went out — whether that's the
	// caller-supplied lines or the auto-selected recent error hits.
	SentExcerpt []string `json:"sentExcerpt"`
}

func (s *Server) handleExplain(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	cfg, err := s.loadConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	target, ok := findTarget(cfg, id)
	if !ok {
		writeError(w, http.StatusNotFound, "target not found")
		return
	}
	if cfg.AIProvider == "" {
		writeError(w, http.StatusConflict, "no AI provider is configured; set one in Settings first")
		return
	}

	var req explainRequest
	if r.Body != nil && r.ContentLength != 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
	}

	lines := req.Lines
	if len(lines) == 0 && target.Wire != nil {
		if watch, err := s.getWatcher(target); err == nil {
			for _, hit := range watch.Recent(0) {
				if hit.Severity == "error" || hit.Severity == "critical" {
					lines = append(lines, hit.Line)
				}
			}
			if len(lines) > maxDefaultExplainHits {
				lines = lines[len(lines)-maxDefaultExplainHits:]
			}
		}
	}

	provider, err := s.newAIProvider(cfg.AIProvider, cfg.AIKey, "")
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	var chainName, execID, beaconID string
	var syncing bool
	if target.Wire != nil {
		if net, ok := catalog.NetworkByChainID(target.Wire.ChainID); ok {
			chainName = net.Name
		}
		execID = target.Wire.ExecID
		beaconID = target.Wire.BeaconID
		if mon, err := s.getMonitor(target, cfg.RefRPCBase); err == nil {
			syncing = mon.Latest().ExecSyncing
		}
	}

	text, err := provider.Explain(r.Context(), ai.ExplainRequest{
		ChainName:    chainName,
		ExecClient:   execID,
		BeaconClient: beaconID,
		Syncing:      syncing,
		Lines:        lines,
	})
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, explainResponse{Text: text, SentExcerpt: lines})
}

// ---------------------------------------------------------------------
// service control: start/stop/restart, clear, disk usage, endpoints,
// firewall checklist — all day-2 operator actions from internal/ops,
// gated on the target existing and having completed setup (Wire != nil).
// ---------------------------------------------------------------------

// targetWithWire loads cfg, resolves id to a Target, and checks it has
// completed setup — the shared 404 (unknown target, checked first per the
// v0.1 review's ordering nit) / 409 (Wire == nil) preamble every route in
// this section needs before it can touch ops.
func (s *Server) targetWithWire(w http.ResponseWriter, r *http.Request, id string) (config.Target, bool) {
	cfg, err := s.loadConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return config.Target{}, false
	}
	target, ok := findTarget(cfg, id)
	if !ok {
		writeError(w, http.StatusNotFound, "target not found")
		return config.Target{}, false
	}
	if target.Wire == nil {
		writeError(w, http.StatusConflict, "target has not completed setup")
		return config.Target{}, false
	}
	return target, true
}

// serviceActionResponse deliberately carries no json tag (like the ops
// structs it sits alongside) so it encodes as PascalCase {"Active":...},
// matching the spec's `{Active bool}`.
type serviceActionResponse struct {
	Active bool
}

func (s *Server) handleServiceAction(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	svc := r.PathValue("svc")
	action := r.PathValue("action")

	target, ok := s.targetWithWire(w, r, id)
	if !ok {
		return
	}

	ex, err := s.getExecutor(target)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	active, err := ops.ServiceAction(r.Context(), ex, svc, action)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, serviceActionResponse{Active: active})
}

// clearRequest mirrors serviceActionResponse's untagged-field convention.
type clearRequest struct {
	Confirm string
}

func (s *Server) handleServiceClear(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	svc := r.PathValue("svc")

	target, ok := s.targetWithWire(w, r, id)
	if !ok {
		return
	}

	var req clearRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Confirm != svc {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("confirm must equal service name %q", svc))
		return
	}

	ex, err := s.getExecutor(target)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	if err := ops.ClearService(r.Context(), ex, *target.Wire, svc); err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "cleared"})
}

func (s *Server) handleDiskUsage(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	target, ok := s.targetWithWire(w, r, id)
	if !ok {
		return
	}

	ex, err := s.getExecutor(target)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	du, err := ops.DiskUsage(r.Context(), ex, *target.Wire)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, du)
}

func (s *Server) handleEndpoints(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	target, ok := s.targetWithWire(w, r, id)
	if !ok {
		return
	}

	ex, err := s.getExecutor(target)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	sshMode := target.Mode == "ssh"
	sshHostHint := ""
	if sshMode && target.SSH != nil {
		sshHostHint = fmt.Sprintf("%s@%s", target.SSH.User, target.SSH.Host)
	}

	ep, err := ops.Endpoints(r.Context(), ex, *target.Wire, sshMode, sshHostHint)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, ep)
}

func (s *Server) handleFirewall(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	target, ok := s.targetWithWire(w, r, id)
	if !ok {
		return
	}

	ex, err := s.getExecutor(target)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	// Grade binds to private overlays (WireGuard, Tailscale, etc.) as overlay
	// rather than LAN. This includes both the operator's declared overlays AND
	// the subnet of any WireGuard server this app provisioned — so a gateway
	// bound on a Jumpgate-set-up overlay is recognized as private ingress, not
	// warned about. A config load error here is non-fatal: fall back to no
	// declared overlays (still conservative).
	var overlayCIDRs []string
	if cfg, cerr := s.loadConfig(); cerr == nil {
		overlayCIDRs = cfg.TrustedOverlayCIDRs()
	}
	overlays := ops.ParseOverlayCIDRs(overlayCIDRs)

	items, err := ops.FirewallChecklist(r.Context(), ex, *target.Wire, overlays...)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	if items == nil {
		items = []ops.CheckItem{}
	}

	writeJSON(w, http.StatusOK, items)
}

// handleDiagnostics runs the network-diagnostics ladder manually (trigger
// "manual") and stores the result as the target's latest report. Auto runs
// (journal/monitor triggered — see diag.go) go through the same gate, so a
// manual click during an in-flight run gets a 409 rather than a duplicate
// probe storm.
func (s *Server) handleDiagnostics(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	target, ok := s.targetWithWire(w, r, id)
	if !ok {
		return
	}

	entry := s.reg.get(id)
	if !entry.tryBeginDiag(time.Now(), false) {
		writeError(w, http.StatusConflict, "a diagnostics run is already in progress for this target")
		return
	}

	report, err := s.runDiagnostics(r.Context(), target, "manual")
	if err != nil {
		entry.endDiag(nil)
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	entry.endDiag(report)

	writeJSON(w, http.StatusOK, report)
}

// handleDiskFree probes the free bytes at a candidate data location on the
// target. Unlike the other ops routes it does NOT require completed setup —
// the setup wizard calls it to size up a location before anything is
// created — so it only needs the target to exist and an executor.
func (s *Server) handleDiskFree(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	path := strings.TrimSpace(r.URL.Query().Get("path"))
	if path == "" {
		writeError(w, http.StatusBadRequest, "path query parameter is required")
		return
	}

	cfg, err := s.loadConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	target, ok := findTarget(cfg, id)
	if !ok {
		writeError(w, http.StatusNotFound, "target not found")
		return
	}
	ex, err := s.getExecutor(target)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	free, err := ops.FreeBytesAt(r.Context(), ex, path)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, struct {
		Path      string `json:"path"`
		FreeBytes uint64 `json:"freeBytes"`
	}{Path: path, FreeBytes: free})
}

// handleDiagnosticsLatest returns the target's most recent diagnostics
// report — manual or auto-triggered — or JSON null when none has run yet.
func (s *Server) handleDiagnosticsLatest(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if _, ok := s.targetWithWire(w, r, id); !ok {
		return
	}

	writeJSON(w, http.StatusOK, s.reg.get(id).latestDiag())
}

// ---------------------------------------------------------------------
// settings
// ---------------------------------------------------------------------

type settingsResponse struct {
	AIProvider string `json:"aiProvider"`
	AIKeySet   bool   `json:"aiKeySet"`
	RefRPCBase string `json:"refRpcBase"`

	// ProviderKeysSet names the placeholders that have a key, never the keys.
	// Same rule as AIKeySet.
	ProviderKeysSet []string `json:"providerKeysSet"`
}

// placeholderNamePattern is the name shape chainlist's ${...} slot accepts. It
// is duplicated here rather than exported from chainlist because this is an
// INPUT rule ("what may be stored") and that one is a parsing rule ("what is
// recognised in a feed URL"); they agree today and are allowed to diverge.
var placeholderNamePattern = regexp.MustCompile(`^[A-Za-z0-9_]+$`)

func settingsResponseFrom(c config.Config) settingsResponse {
	out := settingsResponse{
		AIProvider: c.AIProvider,
		AIKeySet:   c.AIKey != "",
		RefRPCBase: c.RefRPCBase,
		// Never nil: a nil slice serialises as JSON null, and the UI should be
		// able to iterate the field without a guard.
		ProviderKeysSet: []string{},
	}
	for name, v := range c.ProviderKeys {
		if strings.TrimSpace(v) != "" {
			out.ProviderKeysSet = append(out.ProviderKeysSet, name)
		}
	}
	// Sorted, because map order is random and a list that reshuffles on every
	// poll is a list the UI cannot render without flicker.
	sort.Strings(out.ProviderKeysSet)
	return out
}

func (s *Server) handleGetSettings(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.loadConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, settingsResponseFrom(cfg))
}

// settingsRequest uses pointer fields so PUT can distinguish "omitted,
// leave unchanged" from "explicitly set to empty" — most importantly for
// aiKey, which GET never echoes back, so a client re-PUTting the response
// of a prior GET must not blow away an already-stored key.
type settingsRequest struct {
	AIProvider *string `json:"aiProvider"`
	AIKey      *string `json:"aiKey"`
	RefRPCBase *string `json:"refRpcBase"`

	// ProviderKeys is a PATCH by placeholder name, not a replacement: a name
	// with a value sets it, a name with an empty value forgets it, and a name
	// that is absent is left alone. It has to work that way for the same reason
	// aiKey is a pointer — GET never echoes the values back, so a client
	// re-PUTting what it last read would otherwise wipe every key it has.
	ProviderKeys map[string]string `json:"providerKeys"`
}

func (s *Server) handlePutSettings(w http.ResponseWriter, r *http.Request) {
	var req settingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	// Names are checked BEFORE anything is written, so a request carrying one
	// bad name does not half-apply. A name that is not a placeholder is not a
	// key that merely fails to work: it would be listed in providerKeysSet as
	// though it were usable, and redactKeys would emit "${bad name}" — which
	// Resolve then cannot parse, so the URL the client posts back is refused on
	// save, blaming a slot the operator never typed.
	for name := range req.ProviderKeys {
		if !placeholderNamePattern.MatchString(strings.TrimSpace(name)) {
			writeError(w, http.StatusBadRequest, fmt.Sprintf(
				"%q is not a provider key name — a name matches the ${...} slot the chain feed uses, so it may hold only letters, digits and underscores (for example INFURA_API_KEY)", name))
			return
		}
	}

	cfg, err := s.updateConfig(func(c *config.Config) error {
		if req.AIProvider != nil {
			c.AIProvider = *req.AIProvider
		}
		if req.AIKey != nil {
			c.AIKey = *req.AIKey
		}
		if req.RefRPCBase != nil {
			c.RefRPCBase = *req.RefRPCBase
		}
		for name, v := range req.ProviderKeys {
			name = strings.TrimSpace(name)
			v = strings.TrimSpace(v)
			if v == "" {
				delete(c.ProviderKeys, name)
				continue
			}
			if c.ProviderKeys == nil {
				c.ProviderKeys = map[string]string{}
			}
			c.ProviderKeys[name] = v
		}
		return nil
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, settingsResponseFrom(cfg))
}
