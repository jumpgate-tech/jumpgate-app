package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/valve-tech/valve-node-app/internal/config"
	"github.com/valve-tech/valve-node-app/internal/executor"
	"github.com/valve-tech/valve-node-app/internal/vpn"
)

// defaultVPNInterface is the OS interface name an overlay comes up on when the
// operator did not name one. "jumpgate0" so a `wg show` on the box says whose
// interface it is at a glance.
const defaultVPNInterface = "jumpgate0"

// codeVPNNotFound is the typed code for "no overlay with that id", the VPN
// counterpart to codeGatewayNotFound.
const codeVPNNotFound = "vpn-not-found"

// vpnIDPattern constrains an overlay id: it appears in URLs and (via the
// default) in an interface name, so it stays to the same lower-case,
// url-and-shell-safe alphabet gateway ids use.
var vpnIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,38}$`)

// ifacePattern constrains an explicit interface name to what the kernel will
// accept: at most 15 characters (IFNAMSIZ-1) of a safe alphabet. A name that
// the kernel would reject is refused here, at save time, rather than at the
// first failed bring-up.
var ifacePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]{0,14}$`)

func (s *Server) registerVPNRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/vpns", s.handleVPNList)
	mux.HandleFunc("POST /api/vpns", s.handleVPNSave)
	mux.HandleFunc("DELETE /api/vpns/{id}", s.handleVPNDelete)
	// The live read: is the interface actually up on its host, and has a peer
	// handshaked. GET because it reads — it runs one `wg show dump` and changes
	// nothing. Kept off the list response (which is cheap, stored-only) for the
	// same reason gateway traffic is: a probe per overlay on every poll would
	// turn a list into N round-trips, some of them over SSH.
	mux.HandleFunc("GET /api/vpns/{id}/status", s.handleVPNStatus)
	// Bring up / tear down. The literal segments win over no wildcard here —
	// there is no {action} dispatcher — so they are unambiguous.
	mux.HandleFunc("POST /api/vpns/{id}/up", s.handleVPNUp)
	mux.HandleFunc("POST /api/vpns/{id}/down", s.handleVPNDown)
}

// vpnView is the redacted shape of a config.VPN sent to the UI. It carries
// everything needed to render and act on an overlay EXCEPT the one thing a
// screen must never hold: the `.conf` bytes, because they include the interface
// private key. The endpoints and overlay CIDRs are derived from that config but
// are not secret — an endpoint is a public address and a CIDR is a route — so
// they travel; the key does not, the same rule ProviderKeysSet follows for
// provider keys.
type vpnView struct {
	ID        string `json:"id"`
	Provider  string `json:"provider"`
	Interface string `json:"interface"`
	TargetID  string `json:"targetId"`
	Autostart bool   `json:"autostart"`

	// Configured is always true for a stored overlay; it exists so the UI can
	// treat "has a config" as a first-class fact without ever seeing the bytes,
	// exactly as AIKeySet stands in for the AI key.
	Configured bool `json:"configured"`

	// Valid reports whether the stored `.conf` parses and validates. It is
	// almost always true (save refuses an invalid one), but a hand-edited
	// config.json can carry a broken config, and the UI should say so rather
	// than silently show an overlay that will fail every bring-up.
	Valid bool   `json:"valid"`
	Error string `json:"error,omitempty"`

	// Endpoints are the peer endpoints the tunnel dials (host:port) and Overlay
	// are the interface's own addresses as CIDRs — both derived from the config,
	// neither secret. Never nil, so the UI can iterate without a guard.
	Endpoints []string `json:"endpoints"`
	Overlay   []string `json:"overlay"`
	Peers     int      `json:"peers"`
}

// vpnStatusView is a point-in-time read of a live tunnel — the redacted mirror
// of vpn.State.
type vpnStatusView struct {
	ID            string   `json:"id"`
	Up            bool     `json:"up"`
	Interface     string   `json:"interface"`
	Provider      string   `json:"provider"`
	Addresses     []string `json:"addresses"`
	Peers         int      `json:"peers"`
	Handshaked    bool     `json:"handshaked"`
	LastHandshake int64    `json:"lastHandshake"` // unix seconds; 0 = never
}

// ifaceOf is the interface an overlay actually uses: its own if named, else the
// default. Both bring-up and status must agree on this, so it lives in one
// place rather than being defaulted twice.
func ifaceOf(v config.VPN) string {
	if strings.TrimSpace(v.Interface) != "" {
		return v.Interface
	}
	return defaultVPNInterface
}

// vpnViewFrom builds the redacted view, parsing the stored config for the
// display-only fields. A parse/validate failure is reported (Valid=false),
// never fatal: an overlay that will not come up is still an overlay the
// operator needs to see in order to fix or delete it.
func vpnViewFrom(v config.VPN) vpnView {
	view := vpnView{
		ID:         v.ID,
		Provider:   v.Provider,
		Interface:  ifaceOf(v),
		TargetID:   v.TargetID,
		Autostart:  v.Autostart,
		Configured: true,
		Endpoints:  []string{},
		Overlay:    []string{},
	}
	cfg, err := vpn.ParseConfig(v.Config)
	if err == nil {
		err = cfg.Validate()
	}
	if err != nil {
		view.Error = err.Error()
		return view
	}
	view.Valid = true
	view.Overlay = cfg.OverlayCIDRs()
	if view.Overlay == nil {
		view.Overlay = []string{}
	}
	for _, p := range cfg.Peers {
		if e := strings.TrimSpace(p.Endpoint); e != "" {
			view.Endpoints = append(view.Endpoints, e)
		}
	}
	view.Peers = len(cfg.Peers)
	return view
}

func vpnStatusViewFrom(id string, st vpn.State) vpnStatusView {
	out := vpnStatusView{
		ID:         id,
		Up:         st.Up,
		Interface:  st.Interface,
		Provider:   st.Provider,
		Addresses:  st.Addresses,
		Peers:      st.Peers,
		Handshaked: st.Handshaked(),
	}
	if out.Addresses == nil {
		out.Addresses = []string{}
	}
	if !st.LastHandshake.IsZero() {
		out.LastHandshake = st.LastHandshake.Unix()
	}
	return out
}

// vpnByID resolves {id} to a stored overlay, answering 404 with a typed code
// when there is none — the VPN counterpart to Server.gateway.
func (s *Server) vpnByID(w http.ResponseWriter, r *http.Request) (config.Config, config.VPN, bool) {
	cfg, err := s.loadConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return config.Config{}, config.VPN{}, false
	}
	v, ok := cfg.FindVPN(r.PathValue("id"))
	if !ok {
		writeErrorDetail(w, http.StatusNotFound,
			fmt.Sprintf("no overlay %q", r.PathValue("id")), "", codeVPNNotFound)
		return config.Config{}, config.VPN{}, false
	}
	return cfg, v, true
}

// vpnExecutor resolves the host an overlay runs on and returns an executor for
// it plus the interface name. An empty TargetID is the local host — the desktop
// case; a named one must be a registered machine, else the overlay points at a
// box this app no longer manages and there is nowhere to bring it up.
func (s *Server) vpnExecutor(w http.ResponseWriter, cfg config.Config, v config.VPN) (executor.Executor, string, bool) {
	t := config.Target{ID: "local", Mode: "local"}
	if strings.TrimSpace(v.TargetID) != "" {
		ft, ok := findTarget(cfg, v.TargetID)
		if !ok {
			writeError(w, http.StatusConflict, fmt.Sprintf(
				"overlay %q runs on machine %q, which is not registered — re-add that machine or point the overlay at this host", v.ID, v.TargetID))
			return nil, "", false
		}
		t = ft
	}
	ex, err := s.getExecutor(t)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return nil, "", false
	}
	return ex, ifaceOf(v), true
}

// ---------------------------------------------------------------------
// Autostart (run at process start, not over HTTP)
// ---------------------------------------------------------------------

// AutostartResult reports one overlay's autostart outcome. Err is nil when the
// interface was brought up AND verified present; otherwise it carries why.
type AutostartResult struct {
	ID  string
	Err error
}

// AutostartOverlays brings up every stored BYO overlay marked Autostart, at
// process start. It is best-effort and independent: one overlay's failure (an
// unreachable box, a stale config) never blocks the others, and none of them
// blocks the server — the caller runs this off the serving path and logs the
// results. Overlays without Autostart are left alone; the operator brings those
// up by hand. Provisioned servers are not covered here — a wg-quick server conf
// persists on its host and comes back via the host's own systemd on reboot, so
// this app does not re-run it.
func (s *Server) AutostartOverlays(ctx context.Context) []AutostartResult {
	cfg, err := s.loadConfig()
	if err != nil {
		return []AutostartResult{{Err: fmt.Errorf("load config: %w", err)}}
	}
	var out []AutostartResult
	for _, v := range cfg.VPNs {
		if !v.Autostart {
			continue
		}
		out = append(out, AutostartResult{ID: v.ID, Err: s.bringUpOverlay(ctx, cfg, v)})
	}
	return out
}

// bringUpOverlay resolves an overlay's host and brings its interface up,
// verifying it actually came up (WgQuick.Up checks `wg show`, not just the exit
// code). It is the non-HTTP twin of handleVPNUp + vpnExecutor, so autostart and
// the API start an overlay by the exact same path.
func (s *Server) bringUpOverlay(ctx context.Context, cfg config.Config, v config.VPN) error {
	prov, err := vpn.NewStaticProvider(v.Provider, v.Config)
	if err != nil {
		return fmt.Errorf("stored config is not usable: %w", err)
	}
	t := config.Target{ID: "local", Mode: "local"}
	if strings.TrimSpace(v.TargetID) != "" {
		ft, ok := findTarget(cfg, v.TargetID)
		if !ok {
			return fmt.Errorf("runs on machine %q, which is not registered", v.TargetID)
		}
		t = ft
	}
	ex, err := s.getExecutor(t)
	if err != nil {
		return err
	}
	tun := vpn.WgQuick{Exec: ex, Iface: ifaceOf(v), Provider: prov}
	if _, err := tun.Up(ctx); err != nil {
		return err
	}
	return nil
}

// ---------------------------------------------------------------------
// GET /api/vpns
// ---------------------------------------------------------------------

func (s *Server) handleVPNList(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.loadConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	// Never nil: an empty list must serialise as [] so the UI can render "no
	// overlays yet" rather than choke on a null.
	out := make([]vpnView, 0, len(cfg.VPNs))
	for _, v := range cfg.VPNs {
		out = append(out, vpnViewFrom(v))
	}
	writeJSON(w, http.StatusOK, out)
}

// ---------------------------------------------------------------------
// POST /api/vpns   (create or update)
// ---------------------------------------------------------------------

// vpnSaveRequest uses pointer fields so an update can distinguish "omitted,
// leave unchanged" from "set to empty" — most importantly for config, which
// GET never echoes back (it holds the private key), so a client re-POSTing what
// it last read must not blow away the stored key. Same rule settingsRequest
// follows for aiKey.
type vpnSaveRequest struct {
	ID        *string `json:"id"`
	Provider  *string `json:"provider"`
	Interface *string `json:"interface"`
	TargetID  *string `json:"targetId"`
	Config    *string `json:"config"`
	Autostart *bool   `json:"autostart"`
}

func (s *Server) handleVPNSave(w http.ResponseWriter, r *http.Request) {
	var req vpnSaveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	id := ""
	if req.ID != nil {
		id = strings.TrimSpace(*req.ID)
	}
	if !vpnIDPattern.MatchString(id) {
		writeError(w, http.StatusBadRequest,
			"an overlay id must be lower-case letters, digits, dot, dash or underscore (starting with a letter or digit), at most 39 characters — it appears in the URL and, by default, in the interface name")
		return
	}
	if req.Interface != nil {
		if iface := strings.TrimSpace(*req.Interface); iface != "" && !ifacePattern.MatchString(iface) {
			writeError(w, http.StatusBadRequest,
				"an interface name may hold only letters, digits, dot, dash or underscore and at most 15 characters — that is what the kernel accepts")
			return
		}
	}

	// Validate the config BEFORE anything is written, so a request carrying a
	// bad `.conf` does not half-apply — config itself stores bytes without
	// judging them, so this is the layer that judges. Only when it is being set:
	// an update that omits it keeps the stored (already-valid) one.
	if req.Config != nil {
		if _, err := vpn.NewStaticProvider(strings.TrimSpace(deref(req.Provider)), *req.Config); err != nil {
			writeError(w, http.StatusBadRequest, "the WireGuard config is not usable: "+err.Error())
			return
		}
	}

	created := false
	cfg, err := s.updateConfig(func(c *config.Config) error {
		// A named target must exist — an overlay that points at a machine this
		// app does not manage has nowhere to come up. Checked here, in the write,
		// so the read it depends on and the write cannot race.
		if req.TargetID != nil {
			if tid := strings.TrimSpace(*req.TargetID); tid != "" {
				if _, ok := findTarget(*c, tid); !ok {
					return fmt.Errorf("machine %q is not registered — leave the machine empty to run the overlay on this host, or add that machine first", tid)
				}
			}
		}

		for i := range c.VPNs {
			if c.VPNs[i].ID != id {
				continue
			}
			// Update: apply only the fields the request carried.
			if req.Provider != nil {
				c.VPNs[i].Provider = strings.TrimSpace(*req.Provider)
			}
			if req.Interface != nil {
				c.VPNs[i].Interface = strings.TrimSpace(*req.Interface)
			}
			if req.TargetID != nil {
				c.VPNs[i].TargetID = strings.TrimSpace(*req.TargetID)
			}
			if req.Config != nil {
				c.VPNs[i].Config = *req.Config
			}
			if req.Autostart != nil {
				c.VPNs[i].Autostart = *req.Autostart
			}
			return nil
		}

		// Create: a new overlay is nothing without its config.
		if req.Config == nil || strings.TrimSpace(*req.Config) == "" {
			return fmt.Errorf("a new overlay needs a WireGuard config")
		}
		created = true
		c.VPNs = append(c.VPNs, config.VPN{
			ID:        id,
			Provider:  strings.TrimSpace(deref(req.Provider)),
			Interface: strings.TrimSpace(deref(req.Interface)),
			TargetID:  strings.TrimSpace(deref(req.TargetID)),
			Config:    *req.Config,
			Autostart: req.Autostart != nil && *req.Autostart,
		})
		return nil
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	v, _ := cfg.FindVPN(id)
	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	writeJSON(w, status, vpnViewFrom(v))
}

// ---------------------------------------------------------------------
// DELETE /api/vpns/{id}
// ---------------------------------------------------------------------

// handleVPNDelete forgets the overlay's config. It does NOT tear down a tunnel
// that is up: this app never takes down what a single request cannot both find
// and verify, and reaching a possibly-remote host to run `wg-quick down` inside
// a DELETE is exactly the kind of call that hangs. Bring it down first (POST
// .../down) if it is running; the 204 here means the stored config is gone, not
// that an interface was removed.
func (s *Server) handleVPNDelete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	_, err := s.updateConfig(func(c *config.Config) error {
		kept := c.VPNs[:0]
		found := false
		for _, v := range c.VPNs {
			if v.ID == id {
				found = true
				continue
			}
			kept = append(kept, v)
		}
		if !found {
			return fmt.Errorf("no overlay %q", id)
		}
		c.VPNs = kept
		return nil
	})
	if err != nil {
		writeErrorDetail(w, http.StatusNotFound, err.Error(), "", codeVPNNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---------------------------------------------------------------------
// GET /api/vpns/{id}/status
// ---------------------------------------------------------------------

func (s *Server) handleVPNStatus(w http.ResponseWriter, r *http.Request) {
	cfg, v, ok := s.vpnByID(w, r)
	if !ok {
		return
	}
	ex, iface, ok := s.vpnExecutor(w, cfg, v)
	if !ok {
		return
	}
	// Status needs no provider — it only reads `wg show`.
	tun := vpn.WgQuick{Exec: ex, Iface: iface}
	st, err := tun.Status(r.Context())
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	st.Provider = v.Provider
	writeJSON(w, http.StatusOK, vpnStatusViewFrom(v.ID, st))
}

// ---------------------------------------------------------------------
// POST /api/vpns/{id}/up
// ---------------------------------------------------------------------

func (s *Server) handleVPNUp(w http.ResponseWriter, r *http.Request) {
	cfg, v, ok := s.vpnByID(w, r)
	if !ok {
		return
	}
	prov, err := vpn.NewStaticProvider(v.Provider, v.Config)
	if err != nil {
		// A stored config that no longer parses (hand-edited config.json). 422:
		// the request is well-formed, the stored state is not.
		writeError(w, http.StatusUnprocessableEntity, "the stored WireGuard config is not usable: "+err.Error())
		return
	}
	ex, iface, ok := s.vpnExecutor(w, cfg, v)
	if !ok {
		return
	}
	tun := vpn.WgQuick{Exec: ex, Iface: iface, Provider: prov}
	// WgQuick.Up VERIFIES the interface actually came up rather than trusting
	// wg-quick's exit code, so a 200 here means the tunnel is really present —
	// not merely that a command exited 0. A failure is the host's, hence 502.
	st, err := tun.Up(r.Context())
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, vpnStatusViewFrom(v.ID, st))
}

// ---------------------------------------------------------------------
// POST /api/vpns/{id}/down
// ---------------------------------------------------------------------

func (s *Server) handleVPNDown(w http.ResponseWriter, r *http.Request) {
	cfg, v, ok := s.vpnByID(w, r)
	if !ok {
		return
	}
	ex, iface, ok := s.vpnExecutor(w, cfg, v)
	if !ok {
		return
	}
	tun := vpn.WgQuick{Exec: ex, Iface: iface}
	// Down treats an already-absent interface as success, so this is idempotent:
	// tearing down a tunnel that is not up is a no-op, not an error.
	if err := tun.Down(r.Context()); err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// deref returns the pointed-at string or "" for a nil pointer.
func deref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
