package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"slices"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/catalog"
	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/intent"
	"github.com/valve-tech/jumpgate/internal/monitor"
	"github.com/valve-tech/jumpgate/internal/setup"
)

// ---------------------------------------------------------------------
// GET /api/catalog: modules
// ---------------------------------------------------------------------

// catalogModule is one module as the UI and the TUI read it. Applies is
// decided here, once, so neither client re-derives it.
type catalogModule struct {
	ID        string           `json:"id"`
	Name      string           `json:"name"`
	Summary   string           `json:"summary"`
	SiteURL   string           `json:"siteUrl"`
	DocsStep  string           `json:"docsStep"`
	Placement string           `json:"placement"`
	Resources catalogResources `json:"resources"`
	Applies   []catalogApplies `json:"applies"`
	Methods   []catalogMethod  `json:"methods"`
}

type catalogResources struct {
	RAMBytes  uint64 `json:"ramBytes"`
	DiskBytes uint64 `json:"diskBytes"`
}

type catalogMethod struct {
	Name  string `json:"name"`
	Class string `json:"class"`
}

// catalogApplies is Module.Applies for one (chain, client) pair. OK false
// with an empty Reason hides the toggle; a Reason disables it, verbatim.
type catalogApplies struct {
	ChainID int    `json:"chainId"`
	Client  string `json:"client"`
	OK      bool   `json:"ok"`
	Beta    bool   `json:"beta"`
	Reason  string `json:"reason,omitempty"`
}

// catalogModules lists every registered module with its Applies rows: one per
// (chain, exec client) of networks for a node module, one per chain with
// client "" for a box or gateway module, and chain 0 for a box module that
// runs on any chain.
func catalogModules(networks []catalog.Network) []catalogModule {
	out := []catalogModule{}
	for _, m := range catalog.Modules() {
		cm := catalogModule{
			ID: m.ID, Name: m.Name, Summary: m.Summary, SiteURL: m.SiteURL, DocsStep: m.DocsStep,
			Placement: string(m.Placement),
			Resources: catalogResources{RAMBytes: m.Resources.RAMBytes, DiskBytes: m.Resources.DiskBytes},
			Applies:   []catalogApplies{},
			Methods:   []catalogMethod{},
		}
		for _, mm := range m.Methods {
			cm.Methods = append(cm.Methods, catalogMethod{Name: mm.Name, Class: string(mm.Class)})
		}
		row := func(chain int, client string) {
			ok, reason := m.Applies(chain, client)
			cm.Applies = append(cm.Applies, catalogApplies{ChainID: chain, Client: client, OK: ok, Beta: ok && m.IsBeta(chain), Reason: reason})
		}
		if m.Placement == catalog.PlaceBox && len(m.Chains) == 0 {
			row(0, "")
		}
		for _, n := range networks {
			if m.Placement != catalog.PlaceNode {
				row(n.ChainID, "")
				continue
			}
			for _, c := range n.ExecClients {
				row(n.ChainID, c)
			}
		}
		out = append(out, cm)
	}
	return out
}

// ---------------------------------------------------------------------
// PUT /api/targets/{id}/modules/{module}
// ---------------------------------------------------------------------

// moduleRequest is the body of a module change. A node module reads Enabled
// and Public; a box module reads the rest, which become its BoxModule.
type moduleRequest struct {
	Enabled   bool     `json:"enabled"`
	Public    string   `json:"public"`
	ChainID   int      `json:"chainId"`
	Port      int      `json:"port"`
	BindAddr  string   `json:"bindAddr"`
	ExtIP     string   `json:"extIp"`
	Upstreams []string `json:"upstreams"`
	Webhook   string   `json:"webhook"`
}

// moduleChangeResponse answers a module change: "started" with a setup run
// to follow on .../setup/stream, or "saved" when nothing on the box changes
// ("saved-gateways-failed", a 502, when a gateway could not be
// re-provisioned). Gateways are the gateways whose re-provision started (now)
// or will start (after the run); Failed the ones that could not, with why.
type moduleChangeResponse struct {
	Status   string           `json:"status"`
	Gateways []string         `json:"gateways,omitempty"`
	Failed   []gatewayFailure `json:"failed,omitempty"`
}

type gatewayFailure struct {
	ID    string `json:"id"`
	Error string `json:"error"`
}

// validateBoxModuleFields refuses a box module's fields that would reach a
// unit file or a shell as anything but what they claim to be.
func validateBoxModuleFields(b catalog.BoxModule) error {
	if b.Port < 0 || b.Port > 65535 {
		return fmt.Errorf("port %d is out of range", b.Port)
	}
	if b.BindAddr != "" && net.ParseIP(b.BindAddr) == nil {
		return fmt.Errorf("bindAddr %q is not an IP address", b.BindAddr)
	}
	if b.ExtIP != "" && net.ParseIP(b.ExtIP) == nil {
		return fmt.Errorf("extIp %q is not an IP address", b.ExtIP)
	}
	for _, u := range b.Upstreams {
		if err := plainURL(u, "http", "https", "ws", "wss"); err != nil {
			return fmt.Errorf("upstream %q: %w", u, err)
		}
	}
	if b.Webhook != "" {
		if err := plainURL(b.Webhook, "http", "https"); err != nil {
			return fmt.Errorf("webhook %q: %w", b.Webhook, err)
		}
	}
	return nil
}

// plainURL accepts an absolute URL with one of schemes, a host, and no
// whitespace, quote or shell metacharacter anywhere in it.
func plainURL(raw string, schemes ...string) error {
	for _, r := range raw {
		if r <= ' ' || r == 0x7f || containsRune("\"'`\\$;|&<>(){}", r) {
			return errors.New("has a character a URL here may not hold")
		}
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || !slices.Contains(schemes, u.Scheme) {
		return fmt.Errorf("is not a %v URL", schemes)
	}
	return nil
}

func containsRune(set string, r rune) bool {
	for _, c := range set {
		if c == r {
			return true
		}
	}
	return false
}

// handleSetModule turns one module on or off for a target, or changes its
// settings, by its placement: a node module edits the node's wire and runs the
// wizard's plan (which restarts only what changed), a box module runs its own
// short plan. Gateway modules have their own route.
//
// docs:covers msgboard-enable
// docs:covers box-modules
func (s *Server) handleSetModule(w http.ResponseWriter, r *http.Request) {
	var req moduleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	m, ok := catalog.ModuleByID(r.PathValue("module"))
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Sprintf("%q is not a module", r.PathValue("module")))
		return
	}
	cfg, target, ok := s.nodeTarget(w, r.PathValue("id"))
	if !ok {
		return
	}
	switch m.Placement {
	case catalog.PlaceNode:
		s.setNodeModule(w, r, cfg, target, m, req)
	case catalog.PlaceBox:
		s.setBoxModule(w, r, cfg, target, m, req)
	default:
		writeErrorDetail(w, http.StatusBadRequest,
			fmt.Sprintf("%s is a gateway module: turn it on for a gateway (PUT /api/gateways/{gid}/modules/%s)", m.ID, m.ID),
			"", api.CodeModuleNotApplicable)
	}
}

// writeNotApplicable answers a module that does not apply, with its reason.
func writeNotApplicable(w http.ResponseWriter, m catalog.Module, chainID int, reason string) {
	if reason == "" {
		reason = fmt.Sprintf("%s does not run on chain %d", m.ID, chainID)
	}
	writeErrorDetail(w, http.StatusBadRequest, reason, "", api.CodeModuleNotApplicable)
}

func (s *Server) setNodeModule(w http.ResponseWriter, r *http.Request, cfg config.Config, target config.Target, m catalog.Module, req moduleRequest) {
	if target.Wire == nil {
		writeTargetNotSetUp(w)
		return
	}
	cur := *target.Wire
	want := cur
	want.Modules = slices.DeleteFunc(slices.Clone(cur.Modules), func(nm catalog.NodeModule) bool { return nm.ID == m.ID })
	if req.Enabled {
		if ok, reason := m.Applies(cur.ChainID, cur.ExecID); !ok {
			writeNotApplicable(w, m, cur.ChainID, reason)
			return
		}
		want.Modules = append(want.Modules, catalog.NodeModule{ID: m.ID, Public: req.Public})
	}
	if err := catalog.ValidateModules(want); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Public only decides what the gateways in front route; the node's
	// unit is the same, so nothing on the box runs.
	// It still takes the target's slot and the job guard, as every change
	// to a box's node does, so it never lands mid-run or mid-job.
	if req.Enabled && cur.HasModule(m.ID) {
		claimed, ok := s.claimSetupRun(w, target.ID)
		if !ok {
			return
		}
		if s.refuseDuringJob(w, r.Context(), cfg, target) {
			s.releaseSetupRun(target.ID, claimed)
			return
		}
		_, err := s.updateConfig(func(c *config.Config) error {
			t := findTargetPtr(c, target.ID)
			if t == nil || t.Wire == nil {
				return fmt.Errorf("target %s has no node config any more", target.ID)
			}
			// Edited in place, on the config as it is now, so a change
			// saved since this request read it is kept.
			wc := *t.Wire
			wc.Modules = slices.Clone(wc.Modules)
			for i := range wc.Modules {
				if wc.Modules[i].ID == m.ID {
					wc.Modules[i].Public = req.Public
				}
			}
			t.Wire = &wc
			return nil
		})
		// Released before the gateways provision: one may share this box's slot.
		s.releaseSetupRun(target.ID, claimed)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		ok2, failed := s.reprovisionGatewaysFor(target.ID)
		resp := moduleChangeResponse{Status: "saved", Gateways: ok2, Failed: failed}
		status := http.StatusOK
		if len(failed) > 0 {
			// Saved, but a gateway in front still routes the old way.
			resp.Status = "saved-gateways-failed"
			status = http.StatusBadGateway
		}
		writeJSON(w, status, resp)
		return
	}

	wire, ok := s.reconcileWire(w, r, cfg, target, want)
	if !ok {
		return
	}
	gws := gatewaysReferencing(cfg, target.ID)
	after := func() { s.reprovisionGatewaysFor(target.ID) }
	if !s.startWireSetup(w, r, cfg, target, wire, after) {
		return
	}
	writeJSON(w, http.StatusAccepted, moduleChangeResponse{Status: "started", Gateways: gws})
}

func (s *Server) setBoxModule(w http.ResponseWriter, r *http.Request, cfg config.Config, target config.Target, m catalog.Module, req moduleRequest) {
	b := catalog.BoxModule{ID: m.ID, ChainID: req.ChainID, Port: req.Port, BindAddr: req.BindAddr,
		ExtIP: req.ExtIP, Upstreams: req.Upstreams, Webhook: req.Webhook}
	if b.ChainID == 0 && len(m.Chains) > 0 && target.Wire != nil {
		b.ChainID = target.Wire.ChainID
	}
	if b.ChainID != 0 {
		if _, ok := catalog.NetworkByChainID(b.ChainID); !ok {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("chain %d is not a known network", b.ChainID))
			return
		}
	}
	if err := validateBoxModuleFields(b); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	mods := slices.DeleteFunc(slices.Clone(target.Modules), func(o catalog.BoxModule) bool { return o.Instance() == b.Instance() })
	if req.Enabled {
		if ok, reason := m.Applies(b.ChainID, ""); !ok {
			writeNotApplicable(w, m, b.ChainID, reason)
			return
		}
		mods = append(mods, b)
		// PlanBoxModule checks only b; the box runs all of them at once.
		if err := catalog.ValidateBoxModules(mods, target.Wire); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	steps, err := setup.PlanBoxModule(b, target.Wire, req.Enabled)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	ex, err := s.getShellExecutor(target)
	if err != nil {
		writeDialError(w, err)
		return
	}
	claimed, ok := s.claimSetupRun(w, target.ID)
	if !ok {
		return
	}
	if s.refuseDuringJob(w, r.Context(), cfg, target) {
		s.releaseSetupRun(target.ID, claimed)
		return
	}
	var clash error
	if _, err := s.updateConfig(func(c *config.Config) error {
		t := findTargetPtr(c, target.ID)
		if t == nil {
			return fmt.Errorf("target %s is gone", target.ID)
		}
		// Recomputed on the config as it is now, not the copy read before
		// the claim, so a change saved in between is neither lost nor
		// allowed to clash with this one.
		next := slices.DeleteFunc(slices.Clone(t.Modules), func(o catalog.BoxModule) bool { return o.Instance() == b.Instance() })
		if req.Enabled {
			next = append(next, b)
			if err := catalog.ValidateBoxModules(next, t.Wire); err != nil {
				clash = err
				return err
			}
		}
		if len(next) == 0 {
			next = nil
		}
		t.Modules = next
		return nil
	}); err != nil {
		s.releaseSetupRun(target.ID, claimed)
		if clash != nil {
			writeError(w, http.StatusBadRequest, clash.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	wire := catalog.WireConfig{}
	if target.Wire != nil {
		wire = *target.Wire
	}
	s.launchSetupRun(claimed, ex, steps, wire)
	writeJSON(w, http.StatusAccepted, moduleChangeResponse{Status: "started"})
}

func findTargetPtr(c *config.Config, id string) *config.Target {
	for i := range c.Targets {
		if c.Targets[i].ID == id {
			return &c.Targets[i]
		}
	}
	return nil
}

// gatewaysReferencing lists the gateways with a managed-node upstream on
// target id: the ones whose routes follow that node's modules.
func gatewaysReferencing(cfg config.Config, id string) []string {
	var out []string
	for _, gw := range cfg.Gateways {
		refs := false
		for _, n := range gw.Config.Networks {
			for _, u := range n.Upstreams {
				if u.KindOrDefault() == catalog.UpstreamManagedNode && u.TargetID == id {
					refs = true
				}
			}
		}
		if refs {
			out = append(out, gw.ID)
		}
	}
	return out
}

// reprovisionGateway starts one gateway's provision run; a var so tests can
// count the calls without running one.
var reprovisionGateway = (*Server).provisionGateway

// reprovisionGatewaysFor re-provisions every gateway in front of target id,
// from the config as it is now, and returns their ids. A gateway that cannot
// be provisioned is logged: the module change itself has been made.
func (s *Server) reprovisionGatewaysFor(id string) (started []string, failed []gatewayFailure) {
	cfg, err := s.loadConfig()
	if err != nil {
		log.Printf("jumpgate: modules: reload config to re-provision the gateways for %s: %v", id, err)
		return nil, []gatewayFailure{{ID: "*", Error: err.Error()}}
	}
	for _, gid := range gatewaysReferencing(cfg, id) {
		gw, _ := cfg.FindGateway(gid)
		if _, err := reprovisionGateway(s, cfg, gw); err != nil {
			log.Printf("jumpgate: modules: re-provision gateway %s after a module change on %s: %v", gid, id, err)
			failed = append(failed, gatewayFailure{ID: gid, Error: redactKeys(err.Error(), providerKeys(cfg))})
			continue
		}
		started = append(started, gid)
	}
	return started, failed
}

// ---------------------------------------------------------------------
// GET /api/targets/{id}/modules
// ---------------------------------------------------------------------

// gateStaleMessage is shown when a box's gate predates modules.read.
const gateStaleMessage = "Status needs a newer gate on this box"

type moduleListResponse struct {
	Modules   []monitor.ModuleStatus `json:"modules"`
	GateStale bool                   `json:"gateStale"`
	Message   string                 `json:"message,omitempty"`
}

// readGateModules asks a paired box's gate for its box modules; a var so
// tests can stand in an older gate.
var readGateModules = func(s *Server, ctx context.Context, cfg config.Config, t config.Target) ([]monitor.ModuleStatus, error) {
	return decodeResult[[]monitor.ModuleStatus](s.agentResult(ctx, cfg, t, intent.KindModulesRead, struct{}{}))
}

// handleListModules reports a target's module status: its box modules from
// the box (the gate's modules.read when paired, the same probe over the shell
// otherwise), and its node modules from the latest monitor reading.
func (s *Server) handleListModules(w http.ResponseWriter, r *http.Request) {
	cfg, target, ok := s.nodeTarget(w, r.PathValue("id"))
	if !ok {
		return
	}
	resp := moduleListResponse{Modules: []monitor.ModuleStatus{}}
	resp.Modules = append(resp.Modules, s.latestNodeModules(target.ID)...)

	var box []monitor.ModuleStatus
	if target.Agent != nil {
		got, err := readGateModules(s, r.Context(), cfg, target)
		var rej *agentRejected
		switch {
		case errors.As(err, &rej) && rej.Code == intent.ReasonUnknownKind:
			resp.GateStale, resp.Message = true, gateStaleMessage
			writeJSON(w, http.StatusOK, resp)
			return
		case err != nil:
			writeNodeError(w, err)
			return
		}
		box = got
	} else {
		ex, err := s.getShellExecutor(target)
		if err != nil {
			writeDialError(w, err)
			return
		}
		box = monitor.ProbeBoxModules(r.Context(), ex)
	}
	resp.Modules = append(resp.Modules, box...)
	writeJSON(w, http.StatusOK, resp)
}

// latestNodeModules is the node-module part of the target's latest monitor
// reading, if a monitor runs; it never starts one.
func (s *Server) latestNodeModules(id string) []monitor.ModuleStatus {
	e := s.reg.get(id)
	e.mu.Lock()
	mon := e.mon
	e.mu.Unlock()
	if mon == nil {
		return nil
	}
	return mon.Latest().Modules
}

// ---------------------------------------------------------------------
// PUT /api/gateways/{gid}/modules/{module}
// ---------------------------------------------------------------------

// handleSetGatewayModule turns a gateway module on or off, saves the gateway
// and re-provisions it, so its routes take effect.
func (s *Server) handleSetGatewayModule(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	cfg, gw, ok := s.gateway(w, r)
	if !ok {
		return
	}
	m, ok := catalog.ModuleByID(r.PathValue("module"))
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Sprintf("%q is not a module", r.PathValue("module")))
		return
	}
	if m.Placement != catalog.PlaceGateway {
		writeErrorDetail(w, http.StatusBadRequest,
			fmt.Sprintf("%s is a %s module, not a gateway module: turn it on for a box (PUT /api/targets/{id}/modules/%s)", m.ID, m.Placement, m.ID),
			"", api.CodeModuleNotApplicable)
		return
	}
	g := gw.Config
	g.Modules = slices.DeleteFunc(slices.Clone(g.Modules), func(gm catalog.GatewayModule) bool { return gm.ID == m.ID })
	if req.Enabled {
		g.Modules = append(g.Modules, catalog.GatewayModule{ID: m.ID})
	}
	if err := catalog.ValidateGatewayModules(g); err != nil {
		writeErrorDetail(w, http.StatusBadRequest, err.Error(), "", api.CodeModuleNotApplicable)
		return
	}
	if len(g.Modules) == 0 {
		g.Modules = nil
	}
	cfg, err := s.updateConfig(func(c *config.Config) error {
		for i := range c.Gateways {
			if c.Gateways[i].ID == gw.ID {
				c.Gateways[i].Config.Modules = g.Modules
				return nil
			}
		}
		return fmt.Errorf("no gateway %q", gw.ID)
	})
	if err != nil {
		writeErrorDetail(w, http.StatusNotFound, err.Error(), "", api.CodeGatewayNotFound)
		return
	}
	gw, _ = cfg.FindGateway(gw.ID)
	hostID, err := reprovisionGateway(s, cfg, gw)
	if err != nil {
		writeProvisionError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "started", "targetId": hostID})
}
