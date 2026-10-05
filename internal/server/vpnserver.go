package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/executor"
	"github.com/valve-tech/jumpgate/internal/vpn"
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
	// Disconnect (bring the interface down, keep the conf/key/record so a
	// reconnect brings the same identity back) vs the wipe that DELETE does.
	mux.HandleFunc("POST /api/vpn-servers/{id}/down", s.handleVPNServerDown)
	mux.HandleFunc("POST /api/vpn-servers/{id}/up", s.handleVPNServerUp)
	// Enroll a device. The literal "peers" is more specific than nothing here —
	// there is no {action} wildcard on this tree — so these are unambiguous.
	mux.HandleFunc("POST /api/vpn-servers/{id}/peers", s.handleVPNServerEnroll)
	// Revoke a device. The public key is the stable identifier but carries "/"
	// and "+", so it travels in the body, not the path.
	mux.HandleFunc("POST /api/vpn-servers/{id}/peers/remove", s.handleVPNServerRevoke)
	// Set the public endpoint devices dial. A record-only update, separate from
	// provision, because the endpoint never reaches the host (see the handler).
	mux.HandleFunc("POST /api/vpn-servers/{id}/endpoint", s.handleVPNServerSetEndpoint)
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
	if err == nil {
		err = executor.RequireShell(ex)
	}
	if err != nil {
		writeExecutorError(w, err, http.StatusInternalServerError)
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
	cfg, err := s.loadConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// A field the request omits falls back to the stored record when this id
	// already exists, and to the defaults only for a new server. Defaulting an
	// existing server would silently move it: an omitted targetId means "this
	// host", so a remote server would be re-provisioned on the desktop and its
	// record rewritten to point there, renumbered with the default address and
	// port.
	targetID, iface, address, port := "", defaultServerIface, defaultServerAddress, defaultServerPort
	if existing, ok := cfg.FindVPNServer(id); ok {
		targetID, iface, address, port = existing.TargetID, existing.Interface, existing.Address, existing.ListenPort
	}
	targetID = derefOr(req.TargetID, targetID)
	iface = derefOr(req.Interface, iface)
	address = derefOr(req.Address, address)
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

	// Refuse before touching the host, so the other server is left exactly as
	// it was.
	if other, ok := vpnServerOnInterface(cfg, id, targetID, iface); ok {
		writeError(w, http.StatusConflict, vpnInterfaceTakenMessage(other))
		return
	}
	if existing, ok := cfg.FindVPNServer(id); ok {
		if msg := vpnServerMoveWithPeers(existing, targetID, iface, address); msg != "" {
			writeError(w, http.StatusConflict, msg)
			return
		}
	}

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
	// A stored endpoint is kept over the derived SSH host, since the operator
	// may have set a public name the SSH address is not.
	endpointHost := derefOr(req.EndpointHost, "")
	stored, _ := cfg.FindVPNServer(id)
	if endpointHost == "" && stored.Endpoint == "" {
		if t, ok := findTarget(cfg, targetID); ok && t.Mode == "ssh" && t.SSH != nil {
			endpointHost = t.SSH.Host
		}
	}
	endpoint := ""
	if endpointHost != "" {
		endpoint = net.JoinHostPort(endpointHost, strconv.Itoa(port))
	}

	created := false
	var taken *config.VPNServer
	moved := ""
	cfg, err = s.updateConfig(func(c *config.Config) error {
		// Checked again under the config lock, in case a concurrent provision
		// claimed the interface after the check above.
		if other, ok := vpnServerOnInterface(*c, id, targetID, iface); ok {
			taken = &other
			return errors.New(vpnInterfaceTakenMessage(other))
		}
		for i := range c.VPNServers {
			if c.VPNServers[i].ID != id {
				continue
			}
			// Checked again under the lock: a device enrolled since the check
			// above would be stranded the same way.
			if msg := vpnServerMoveWithPeers(c.VPNServers[i], targetID, iface, address); msg != "" {
				moved = msg
				return errors.New(msg)
			}
			// Re-provision keeps the peers: the server key is idempotent, so
			// every config already handed out stays valid, and ProvisionServer
			// carries the [Peer] sections of the host conf over, so every
			// device recorded here is still admitted there.
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
	if taken != nil || moved != "" {
		// This refusal comes from the re-check under the lock, AFTER
		// ProvisionServer ran: the host already has the new conf even though
		// the record was not changed. Say so. (A per-server mutex held across
		// the host call would close this window; that is a follow-up.)
		writeError(w, http.StatusConflict, err.Error()+" — "+vpnLateRefusalNote)
		return
	}
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
// POST /api/vpn-servers/{id}/endpoint  (set the public endpoint)
// ---------------------------------------------------------------------

type vpnSetEndpointRequest struct {
	EndpointHost string `json:"endpointHost"` // public host or domain devices dial; the port is the server's own
}

// handleVPNServerSetEndpoint records the public host devices dial. It is
// deliberately NOT a re-provision: the endpoint appears only in the Endpoint
// line of the device .conf rendered at enrollment, never in the server's own
// conf on the host, so there is nothing to change there. Going through
// provision instead needed root on the host, bounced the interface (dropping
// every connected device for a metadata edit) and, with the request's other
// fields omitted, re-provisioned with the defaults on the wrong machine.
//
// Configs already handed out keep whatever endpoint they were minted with; this
// changes only the configs issued from now on.
func (s *Server) handleVPNServerSetEndpoint(w http.ResponseWriter, r *http.Request) {
	_, sv, ok := s.vpnServerByID(w, r)
	if !ok {
		return
	}
	var req vpnSetEndpointRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	host := strings.TrimSpace(req.EndpointHost)
	// A bracketed IPv6 literal is accepted the way people paste it; JoinHostPort
	// adds the brackets back.
	host = strings.TrimSuffix(strings.TrimPrefix(host, "["), "]")
	if host == "" || strings.ContainsAny(host, " \t/") {
		writeError(w, http.StatusBadRequest, "endpointHost must be the public host or domain devices dial, like vpn.example.com")
		return
	}
	// The port is always the server's listen port, so a host:port here is a
	// mistake that would render as host:port:port. A bare IPv6 address does not
	// split, so it is not caught by this.
	if _, _, err := net.SplitHostPort(host); err == nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf(
			"endpointHost is the host only; the port is the server's listen port (%d)", sv.ListenPort))
		return
	}
	endpoint := net.JoinHostPort(host, strconv.Itoa(sv.ListenPort))

	cfg, err := s.updateConfig(func(c *config.Config) error {
		for i := range c.VPNServers {
			if c.VPNServers[i].ID == sv.ID {
				c.VPNServers[i].Endpoint = endpoint
				return nil
			}
		}
		return fmt.Errorf("server %q vanished while setting its endpoint", sv.ID)
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	updated, _ := cfg.FindVPNServer(sv.ID)
	writeJSON(w, http.StatusOK, vpnServerViewFrom(updated))
}

// ---------------------------------------------------------------------
// DELETE /api/vpn-servers/{id}
// ---------------------------------------------------------------------

// handleVPNServerDelete WIPES a provisioned server: it tears down the host
// (DeprovisionServer — the exact reverse of provision: interface down, conf and
// key removed, verified gone) and THEN forgets the record. This is the "does the
// opposite of the buildup" delete, unlike the gateway delete's "never stop what
// we didn't start" — here the app DID start it, so wiping should reverse it.
//
// If the host teardown fails (e.g. the box is unreachable), it returns 502 and
// KEEPS the record, so a server that might still be up is not silently forgotten
// — with ?force=true to forget the record anyway when the host is gone for good.
func (s *Server) handleVPNServerDelete(w http.ResponseWriter, r *http.Request) {
	cfg, sv, ok := s.vpnServerByID(w, r)
	if !ok {
		return
	}
	force := r.URL.Query().Get("force") == "true"

	// Tear the host down. Resolve the executor inline (not via hostExecutor,
	// which writes its own error) so the ?force path can forget the record even
	// when the host is unresolvable, without double-writing the response.
	teardownErr := ""
	t := config.Target{ID: "local", Mode: "local"}
	if strings.TrimSpace(sv.TargetID) != "" {
		if ft, ok := findTarget(cfg, sv.TargetID); ok {
			t = ft
		} else {
			teardownErr = fmt.Sprintf("machine %q is not registered", sv.TargetID)
		}
	}
	if teardownErr == "" {
		if ex, err := s.getShellExecutor(t); err != nil {
			teardownErr = err.Error()
		} else if err := vpn.DeprovisionServer(r.Context(), ex, sv.Interface); err != nil {
			teardownErr = err.Error()
		}
	}
	if teardownErr != "" && !force {
		writeErrorDetail(w, http.StatusBadGateway,
			"could not tear the server down on its host: "+teardownErr,
			"the record was kept so it is not lost; retry, or delete with ?force=true to forget it anyway", "")
		return
	}

	if _, err := s.updateConfig(func(c *config.Config) error {
		kept := c.VPNServers[:0]
		for _, x := range c.VPNServers {
			if x.ID != sv.ID {
				kept = append(kept, x)
			}
		}
		c.VPNServers = kept
		return nil
	}); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
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
// POST /api/vpn-servers/{id}/down   (disconnect — reversible)
// ---------------------------------------------------------------------

// handleVPNServerDown brings the interface down but LEAVES the conf, key and
// record in place — a disconnect, not a wipe. Enrolled devices stay enrolled; a
// reconnect (POST .../up) brings the same server identity back.
func (s *Server) handleVPNServerDown(w http.ResponseWriter, r *http.Request) {
	cfg, sv, ok := s.vpnServerByID(w, r)
	if !ok {
		return
	}
	ex, ok := s.hostExecutor(w, cfg, sv.TargetID)
	if !ok {
		return
	}
	if err := (vpn.WgQuick{Exec: ex, Iface: sv.Interface}).Down(r.Context()); err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	st, _ := (vpn.WgQuick{Exec: ex, Iface: sv.Interface}).Status(r.Context())
	writeJSON(w, http.StatusOK, vpnStatusViewFrom(sv.ID, st))
}

// ---------------------------------------------------------------------
// POST /api/vpn-servers/{id}/up   (reconnect)
// ---------------------------------------------------------------------

// handleVPNServerUp brings a disconnected server back up from its EXISTING conf
// via StartServer, not a re-provision: the conf is left untouched, so its
// identity and every enrolled peer come back as they were, and a reconnect
// needs none of provisioning's root-level rewrites.
func (s *Server) handleVPNServerUp(w http.ResponseWriter, r *http.Request) {
	cfg, sv, ok := s.vpnServerByID(w, r)
	if !ok {
		return
	}
	ex, ok := s.hostExecutor(w, cfg, sv.TargetID)
	if !ok {
		return
	}
	if err := vpn.StartServer(r.Context(), ex, sv.Interface); err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	st, _ := (vpn.WgQuick{Exec: ex, Iface: sv.Interface}).Status(r.Context())
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

	// Reserve the device's overlay address under the config lock: allocate it
	// from the peers recorded NOW and record the peer in the same step. Two
	// enrolls allocating from snapshots taken outside the lock got the same
	// address, and WireGuard then silently moved it to the second device. The
	// host call comes after the reservation; a failure rolls it back.
	var ip string
	var full bool
	if _, err := s.updateConfig(func(c *config.Config) error {
		for i := range c.VPNServers {
			if c.VPNServers[i].ID != sv.ID {
				continue
			}
			taken := make([]string, 0, len(c.VPNServers[i].Peers))
			for _, p := range c.VPNServers[i].Peers {
				taken = append(taken, p.AllowedIP)
			}
			next, err := vpn.NextPeerIP(c.VPNServers[i].Address, taken)
			if err != nil {
				full = true
				return err
			}
			ip = next
			c.VPNServers[i].Peers = append(c.VPNServers[i].Peers, config.VPNPeer{
				Name: name, PublicKey: key.PublicKey, AllowedIP: ip,
			})
			return nil
		}
		return fmt.Errorf("server %q vanished mid-enroll", sv.ID)
	}); err != nil {
		status := http.StatusInternalServerError
		if full {
			status = http.StatusConflict
		}
		writeError(w, status, err.Error())
		return
	}
	release := func() { s.forgetVPNPeer(sv.ID, key.PublicKey) }

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
		release()
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Authorize on the host, so the config we hand back is one that actually
	// works. A refusal there frees the reserved address again.
	if err := vpn.AddPeer(r.Context(), ex, vpn.AddPeerParams{
		Iface: sv.Interface, PeerPublicKey: key.PublicKey, AllowedIP: ip,
	}); err != nil {
		release()
		writeError(w, http.StatusBadGateway, err.Error())
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

// forgetVPNPeer drops the peer with pub from server id's records. It undoes an
// enroll's reservation when the enroll fails after it; a failure to undo is
// logged, since the enroll's own error is what the caller must see.
func (s *Server) forgetVPNPeer(id, pub string) {
	if _, err := s.updateConfig(func(c *config.Config) error {
		for i := range c.VPNServers {
			if c.VPNServers[i].ID != id {
				continue
			}
			kept := c.VPNServers[i].Peers[:0]
			for _, p := range c.VPNServers[i].Peers {
				if p.PublicKey != pub {
					kept = append(kept, p)
				}
			}
			c.VPNServers[i].Peers = kept
		}
		return nil
	}); err != nil {
		log.Printf("jumpgate: vpn server %q: could not release a failed enroll's reservation: %v", id, err)
	}
}

// vpnServerMoveWithPeers explains why re-provisioning existing as (targetID,
// iface, address) would strand its peers, or returns "" when it would not. A
// server with devices cannot move to another machine or interface, or onto
// another subnet: the configs already handed out name the old endpoint and
// key, and the carried peers' addresses would sit outside the new subnet,
// while the UI still listed every device as enrolled.
func vpnServerMoveWithPeers(existing config.VPNServer, targetID, iface, address string) string {
	if len(existing.Peers) == 0 {
		return ""
	}
	var moves []string
	if strings.TrimSpace(existing.TargetID) != strings.TrimSpace(targetID) {
		moves = append(moves, fmt.Sprintf("machine %q to %q", existing.TargetID, targetID))
	}
	if existing.Interface != iface {
		moves = append(moves, fmt.Sprintf("interface %s to %s", existing.Interface, iface))
	}
	if !sameSubnet(existing.Address, address) {
		moves = append(moves, fmt.Sprintf("subnet %s to %s", existing.Address, address))
	}
	if len(moves) == 0 {
		return ""
	}
	return fmt.Sprintf("server %q has %d enrolled device(s); re-provisioning would change its %s and strand every config already handed out. "+
		"wipe or migrate first: revoke the devices (or wipe the server), then provision it again", existing.ID, len(existing.Peers), strings.Join(moves, ", "))
}

// vpnLateRefusalNote is appended to a provision refused after the host was
// already provisioned.
const vpnLateRefusalNote = "the host may already have been changed; re-run provision to reconcile"

// sameSubnet reports whether two CIDRs name the same network.
func sameSubnet(a, b string) bool {
	_, na, errA := net.ParseCIDR(a)
	_, nb, errB := net.ParseCIDR(b)
	if errA != nil || errB != nil {
		return a == b
	}
	return na.String() == nb.String()
}

// vpnServerOnInterface returns the server record, other than id, that already
// owns iface on targetID. Two records on one (machine, interface) pair would
// share one conf, key and wg interface, so provisioning the second rewrote the
// first's [Interface] and bounced it, and its devices then appeared under both.
// With the default jumpgate0 every second server on a machine collided.
func vpnServerOnInterface(cfg config.Config, id, targetID, iface string) (config.VPNServer, bool) {
	for _, sv := range cfg.VPNServers {
		if sv.ID != id && strings.TrimSpace(sv.TargetID) == strings.TrimSpace(targetID) && sv.Interface == iface {
			return sv, true
		}
	}
	return config.VPNServer{}, false
}

func vpnInterfaceTakenMessage(other config.VPNServer) string {
	machine := "this host"
	if strings.TrimSpace(other.TargetID) != "" {
		machine = fmt.Sprintf("machine %q", other.TargetID)
	}
	return fmt.Sprintf("server %q already runs on interface %s on %s — pick another interface, or re-provision %q instead",
		other.ID, other.Interface, machine, other.ID)
}

// derefOr returns *p, or def when p is nil.
func derefOr(p *string, def string) string {
	if p == nil {
		return def
	}
	return *p
}
