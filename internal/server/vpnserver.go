package server

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/valve-tech/valve-node-app/internal/config"
	"github.com/valve-tech/valve-node-app/internal/executor"
	"github.com/valve-tech/valve-node-app/internal/vpn"
)

// Defaults for a provisioned server when the request does not specify them: a
// private-range overlay, the standard WireGuard port, and this app's interface
// name.
const (
	defaultServerAddress = "10.9.0.1/24"
	defaultServerPort    = 51820
	defaultServerIface   = defaultVPNInterface // "jumpgate0", shared with vpn.go
)

// codeVPNServerNotFound is the typed code for "no provisioned server with that id".
const codeVPNServerNotFound = "vpn-server-not-found"

func (s *Server) registerVPNServerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/vpn-servers", s.handleVPNServerList)
	mux.HandleFunc("POST /api/vpn-servers", s.handleVPNServerProvision)
	mux.HandleFunc("GET /api/vpn-servers/{id}", s.handleVPNServerGet)
	mux.HandleFunc("DELETE /api/vpn-servers/{id}", s.handleVPNServerDelete)
	mux.HandleFunc("GET /api/vpn-servers/{id}/status", s.handleVPNServerStatus)
	// Enroll a device. The literal "peers" is more specific than nothing here —
	// there is no {action} wildcard on this tree — so these are unambiguous.
	mux.HandleFunc("POST /api/vpn-servers/{id}/peers", s.handleVPNServerEnroll)
	// Revoke a device. The public key is the stable identifier but carries "/"
	// and "+", so it travels in the body, not the path.
	mux.HandleFunc("POST /api/vpn-servers/{id}/peers/remove", s.handleVPNServerRevoke)
}

// ---------------------------------------------------------------------
// views (all fields here are non-secret by construction — a provisioned
// server holds no private key; see config.VPNServer)
// ---------------------------------------------------------------------

type vpnPeerView struct {
	Name      string `json:"name"`
	PublicKey string `json:"publicKey"`
	AllowedIP string `json:"allowedIp"`
}

type vpnServerView struct {
	ID         string        `json:"id"`
	TargetID   string        `json:"targetId"`
	Interface  string        `json:"interface"`
	Address    string        `json:"address"`
	ListenPort int           `json:"listenPort"`
	PublicKey  string        `json:"publicKey"`
	Endpoint   string        `json:"endpoint"`
	Peers      []vpnPeerView `json:"peers"`
}

func vpnServerViewFrom(s config.VPNServer) vpnServerView {
	peers := make([]vpnPeerView, 0, len(s.Peers))
	for _, p := range s.Peers {
		peers = append(peers, vpnPeerView{Name: p.Name, PublicKey: p.PublicKey, AllowedIP: p.AllowedIP})
	}
	return vpnServerView{
		ID: s.ID, TargetID: s.TargetID, Interface: s.Interface, Address: s.Address,
		ListenPort: s.ListenPort, PublicKey: s.PublicKey, Endpoint: s.Endpoint, Peers: peers,
	}
}

// hostExecutor resolves the machine a server runs on to an executor. Empty
// targetID is this host (the desktop case); a named one must be a registered
// machine, else there is nowhere to run.
func (s *Server) hostExecutor(w http.ResponseWriter, cfg config.Config, targetID string) (executor.Executor, bool) {
	t := config.Target{ID: "local", Mode: "local"}
	if strings.TrimSpace(targetID) != "" {
		ft, ok := findTarget(cfg, targetID)
		if !ok {
			writeError(w, http.StatusConflict, fmt.Sprintf(
				"machine %q is not registered — provision on this host (leave the machine empty) or add that machine first", targetID))
			return nil, false
		}
		t = ft
	}
	ex, err := s.getExecutor(t)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return nil, false
	}
	return ex, true
}

func (s *Server) vpnServerByID(w http.ResponseWriter, r *http.Request) (config.Config, config.VPNServer, bool) {
	cfg, err := s.loadConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return config.Config{}, config.VPNServer{}, false
	}
	sv, ok := cfg.FindVPNServer(r.PathValue("id"))
	if !ok {
		writeErrorDetail(w, http.StatusNotFound,
			fmt.Sprintf("no provisioned server %q", r.PathValue("id")), "", codeVPNServerNotFound)
		return config.Config{}, config.VPNServer{}, false
	}
	return cfg, sv, true
}

// ---------------------------------------------------------------------
// GET /api/vpn-servers  and  GET /api/vpn-servers/{id}
// ---------------------------------------------------------------------

func (s *Server) handleVPNServerList(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.loadConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]vpnServerView, 0, len(cfg.VPNServers))
	for _, sv := range cfg.VPNServers {
		out = append(out, vpnServerViewFrom(sv))
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleVPNServerGet(w http.ResponseWriter, r *http.Request) {
	_, sv, ok := s.vpnServerByID(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, vpnServerViewFrom(sv))
}

// ---------------------------------------------------------------------
// POST /api/vpn-servers  (provision — the "pick a device" easy button)
// ---------------------------------------------------------------------

type vpnServerProvisionRequest struct {
	ID           *string `json:"id"`
	TargetID     *string `json:"targetId"`
	Interface    *string `json:"interface"`
	Address      *string `json:"address"`
	ListenPort   *int    `json:"listenPort"`
	EndpointHost *string `json:"endpointHost"` // public host/domain devices dial; derived from an SSH target if omitted
}

type vpnServerProvisionResponse struct {
	Server vpnServerView `json:"server"`
	// FirewallHint is the command to admit peers — this app never opens the
	// firewall itself (see vpn.ServerInfo.FirewallHint). Provisioning-time
	// advice, so it rides the response rather than being stored.
	FirewallHint string `json:"firewallHint"`
	// EndpointConfigured is false when no reachable host could be determined:
	// the server is up, but a device config cannot be issued until an endpoint
	// is known. The UI uses this to prompt for one before enrollment.
	EndpointConfigured bool `json:"endpointConfigured"`
}

func (s *Server) handleVPNServerProvision(w http.ResponseWriter, r *http.Request) {
	var req vpnServerProvisionRequest
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
			"a server id must be lower-case letters, digits, dot, dash or underscore (starting with a letter or digit), at most 39 characters")
		return
	}
	iface := derefOr(req.Interface, defaultServerIface)
	address := derefOr(req.Address, defaultServerAddress)
	port := defaultServerPort
	if req.ListenPort != nil {
		port = *req.ListenPort
	}
	// Validate the request's own inputs here so a client mistake is a 400, not a
	// 502 blamed on the host. The engine validates again, but by the time it
	// runs an error is genuinely host-side.
	if _, _, err := net.ParseCIDR(address); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("address %q must be a CIDR like 10.9.0.1/24", address))
		return
	}
	if port < 1 || port > 65535 {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("listenPort %d is out of range (1-65535)", port))
		return
	}

	cfg, err := s.loadConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	targetID := derefOr(req.TargetID, "")
	ex, ok := s.hostExecutor(w, cfg, targetID)
	if !ok {
		return
	}

	info, err := vpn.ProvisionServer(r.Context(), ex, vpn.ServerParams{Iface: iface, Address: address, ListenPort: port})
	if err != nil {
		// The interface/port failed to come up (verify-by-running caught it),
		// or a prerequisite (root, wireguard-tools) is missing — a host-side
		// failure.
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	// The endpoint host devices dial: an explicit override, else the SSH host
	// of a fleet target, else unknown (local host with no public name).
	endpointHost := derefOr(req.EndpointHost, "")
	if endpointHost == "" {
		if t, ok := findTarget(cfg, targetID); ok && t.Mode == "ssh" && t.SSH != nil {
			endpointHost = t.SSH.Host
		}
	}
	endpoint := ""
	if endpointHost != "" {
		endpoint = net.JoinHostPort(endpointHost, strconv.Itoa(port))
	}

	created := false
	cfg, err = s.updateConfig(func(c *config.Config) error {
		for i := range c.VPNServers {
			if c.VPNServers[i].ID != id {
				continue
			}
			// Re-provision keeps the peers: the server key is idempotent, so
			// every config already handed out stays valid.
			c.VPNServers[i].TargetID = targetID
			c.VPNServers[i].Interface = iface
			c.VPNServers[i].Address = address
			c.VPNServers[i].ListenPort = port
			c.VPNServers[i].PublicKey = info.PublicKey
			if endpoint != "" {
				c.VPNServers[i].Endpoint = endpoint
			}
			return nil
		}
		created = true
		c.VPNServers = append(c.VPNServers, config.VPNServer{
			ID: id, TargetID: targetID, Interface: iface, Address: address,
			ListenPort: port, PublicKey: info.PublicKey, Endpoint: endpoint,
		})
		return nil
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	sv, _ := cfg.FindVPNServer(id)
	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	writeJSON(w, status, vpnServerProvisionResponse{
		Server:             vpnServerViewFrom(sv),
		FirewallHint:       info.FirewallHint,
		EndpointConfigured: sv.Endpoint != "",
	})
}

// ---------------------------------------------------------------------
// DELETE /api/vpn-servers/{id}
// ---------------------------------------------------------------------

// handleVPNServerDelete forgets the server's record. Like every other delete in
// this app, it does not tear down what a single request cannot both find and
// verify: the interface on the host is left running (bring it down first if you
// want it gone). The 204 means the record is gone, not that an interface was
// removed.
func (s *Server) handleVPNServerDelete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	_, err := s.updateConfig(func(c *config.Config) error {
		kept := c.VPNServers[:0]
		found := false
		for _, sv := range c.VPNServers {
			if sv.ID == id {
				found = true
				continue
			}
			kept = append(kept, sv)
		}
		if !found {
			return fmt.Errorf("no provisioned server %q", id)
		}
		c.VPNServers = kept
		return nil
	})
	if err != nil {
		writeErrorDetail(w, http.StatusNotFound, err.Error(), "", codeVPNServerNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---------------------------------------------------------------------
// GET /api/vpn-servers/{id}/status
// ---------------------------------------------------------------------

func (s *Server) handleVPNServerStatus(w http.ResponseWriter, r *http.Request) {
	cfg, sv, ok := s.vpnServerByID(w, r)
	if !ok {
		return
	}
	ex, ok := s.hostExecutor(w, cfg, sv.TargetID)
	if !ok {
		return
	}
	st, err := (vpn.WgQuick{Exec: ex, Iface: sv.Interface}).Status(r.Context())
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, vpnStatusViewFrom(sv.ID, st))
}

// ---------------------------------------------------------------------
// POST /api/vpn-servers/{id}/peers  (enroll a device)
// ---------------------------------------------------------------------

type vpnEnrollRequest struct {
	Name         string   `json:"name"`
	DNS          []string `json:"dns"`
	FullTunnel   bool     `json:"fullTunnel"`   // route ALL the device's traffic through the tunnel
	AllowedIPs   []string `json:"allowedIps"`   // explicit override for what the device routes through the tunnel
	EndpointHost *string  `json:"endpointHost"` // override, if the server has no stored endpoint
}

// vpnEnrollResponse carries the client config ONCE. It is the only place a
// device's private key ever appears in an API response — it is generated at
// enrollment, delivered here, and never stored, so a device that loses it must
// be re-enrolled rather than re-fetched.
type vpnEnrollResponse struct {
	Name      string `json:"name"`
	PublicKey string `json:"publicKey"`
	AllowedIP string `json:"allowedIp"`
	// Config is the WireGuard .conf the device imports — private key included.
	// Show it once (a QR is ideal); it is not retrievable again.
	Config string `json:"config"`
}

func (s *Server) handleVPNServerEnroll(w http.ResponseWriter, r *http.Request) {
	cfg, sv, ok := s.vpnServerByID(w, r)
	if !ok {
		return
	}
	var req vpnEnrollRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		writeError(w, http.StatusBadRequest, "a device needs a name")
		return
	}

	// The endpoint the device will dial. Without one, a config cannot be issued.
	endpoint := sv.Endpoint
	if endpoint == "" && req.EndpointHost != nil && strings.TrimSpace(*req.EndpointHost) != "" {
		endpoint = net.JoinHostPort(strings.TrimSpace(*req.EndpointHost), strconv.Itoa(sv.ListenPort))
	}
	if endpoint == "" {
		writeError(w, http.StatusBadRequest,
			"this server has no reachable endpoint yet — provide endpointHost (the public host or domain devices will dial)")
		return
	}

	// Allocate the device's overlay address from the server's subnet.
	taken := make([]string, 0, len(sv.Peers))
	for _, p := range sv.Peers {
		taken = append(taken, p.AllowedIP)
	}
	ip, err := vpn.NextPeerIP(sv.Address, taken)
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}

	// What the DEVICE routes through the tunnel: an explicit override, else
	// everything (full tunnel), else just the server's subnet — the default,
	// which reaches the services on the box over the overlay and leaves the
	// rest of the device's traffic alone.
	deviceAllowedIPs := req.AllowedIPs
	if len(deviceAllowedIPs) == 0 {
		if req.FullTunnel {
			deviceAllowedIPs = []string{"0.0.0.0/0"}
		} else if _, subnet, err := net.ParseCIDR(sv.Address); err == nil {
			deviceAllowedIPs = []string{subnet.String()}
		}
	}

	key, err := vpn.GenerateKey()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	ex, ok := s.hostExecutor(w, cfg, sv.TargetID)
	if !ok {
		return
	}
	// Authorize on the host first, so the config we hand back is one that
	// actually works; persist to our records second.
	if err := vpn.AddPeer(r.Context(), ex, vpn.AddPeerParams{
		Iface: sv.Interface, PeerPublicKey: key.PublicKey, AllowedIP: ip,
	}); err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	clientConf, err := vpn.RenderClientConfig(vpn.ClientConfigParams{
		PrivateKey:          key.PrivateKey,
		Address:             []string{ip},
		DNS:                 req.DNS,
		ServerPublicKey:     sv.PublicKey,
		Endpoint:            endpoint,
		AllowedIPs:          deviceAllowedIPs,
		PersistentKeepalive: 25,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if _, err := s.updateConfig(func(c *config.Config) error {
		for i := range c.VPNServers {
			if c.VPNServers[i].ID != sv.ID {
				continue
			}
			c.VPNServers[i].Peers = append(c.VPNServers[i].Peers, config.VPNPeer{
				Name: name, PublicKey: key.PublicKey, AllowedIP: ip,
			})
			return nil
		}
		return fmt.Errorf("server %q vanished mid-enroll", sv.ID)
	}); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, vpnEnrollResponse{
		Name: name, PublicKey: key.PublicKey, AllowedIP: ip, Config: clientConf,
	})
}

// ---------------------------------------------------------------------
// POST /api/vpn-servers/{id}/peers/remove  (revoke a device)
// ---------------------------------------------------------------------

type vpnRevokeRequest struct {
	PublicKey string `json:"publicKey"`
}

func (s *Server) handleVPNServerRevoke(w http.ResponseWriter, r *http.Request) {
	cfg, sv, ok := s.vpnServerByID(w, r)
	if !ok {
		return
	}
	var req vpnRevokeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	pub := strings.TrimSpace(req.PublicKey)
	if pub == "" {
		writeError(w, http.StatusBadRequest, "publicKey is required")
		return
	}

	ex, ok := s.hostExecutor(w, cfg, sv.TargetID)
	if !ok {
		return
	}
	// Remove from the running server (RemovePeer verifies it is actually gone)
	// before forgetting the record — so a failure leaves our records honest
	// rather than showing a device revoked while it can still connect.
	if err := vpn.RemovePeer(r.Context(), ex, sv.Interface, pub); err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	if _, err := s.updateConfig(func(c *config.Config) error {
		for i := range c.VPNServers {
			if c.VPNServers[i].ID != sv.ID {
				continue
			}
			kept := c.VPNServers[i].Peers[:0]
			for _, p := range c.VPNServers[i].Peers {
				if p.PublicKey != pub {
					kept = append(kept, p)
				}
			}
			c.VPNServers[i].Peers = kept
			return nil
		}
		return nil
	}); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// derefOr returns *p, or def when p is nil.
func derefOr(p *string, def string) string {
	if p == nil {
		return def
	}
	return *p
}
