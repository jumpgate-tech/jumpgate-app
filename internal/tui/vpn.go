package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/valve-tech/jumpgate/internal/apiclient"
)

// The VPN view (inventory I): client overlays and VPN servers. A client's
// WireGuard config holds a private key, so it is pasted into the masked
// field and cleared after the save. A new peer's config is shown once and
// never kept.

type (
	vpnClient struct {
		ID         string `json:"id"`
		Provider   string `json:"provider"`
		Interface  string `json:"interface"`
		TargetID   string `json:"targetId"`
		Autostart  bool   `json:"autostart"`
		Configured bool   `json:"configured"`
	}
	vpnPeer struct {
		Name      string `json:"name"`
		PublicKey string `json:"publicKey"`
		AllowedIP string `json:"allowedIp"`
	}
	vpnServer struct {
		ID         string    `json:"id"`
		TargetID   string    `json:"targetId"`
		Interface  string    `json:"interface"`
		Address    string    `json:"address"`
		ListenPort int       `json:"listenPort"`
		PublicKey  string    `json:"publicKey"`
		Endpoint   string    `json:"endpoint"`
		Peers      []vpnPeer `json:"peers"`
	}
	vpnListMsg struct {
		clients []vpnClient
		servers []vpnServer
		cErr    error
		sErr    error
	}
	// vpnPeerMsg is a new peer: config is the device's WireGuard config,
	// private key included, shown once.
	vpnPeerMsg struct {
		server, name, ip string
		config           []byte
		err              error
	}
)

var vpnKeys = struct {
	Switch, Reload, New, Up, Down, Status, Delete, Endpoint, AddPeer, RemovePeer key.Binding
}{
	Switch:     key.NewBinding(key.WithKeys("v"), key.WithHelp("v", "clients / servers")),
	Reload:     key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "reload")),
	New:        key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "new")),
	Up:         key.NewBinding(key.WithKeys("u"), key.WithHelp("u", "up")),
	Down:       key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "down")),
	Status:     key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "status")),
	Delete:     key.NewBinding(key.WithKeys("D"), key.WithHelp("D", "delete, typed confirmation")),
	Endpoint:   key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "server: endpoint")),
	AddPeer:    key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "server: add a peer (config shown once)")),
	RemovePeer: key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "server: remove a peer")),
}

type vpnScreen struct {
	servers bool // the servers list, else the clients
	cs      []vpnClient
	ss      []vpnServer
	has     bool
	cErr    error
	sErr    error
	cursor  int
	offset  int
	res     resultPane
}

func (s *vpnScreen) capturing() bool { return false }
func (s *vpnScreen) keys() []key.Binding {
	return append([]key.Binding{navKeys.Up, navKeys.Down, navKeys.Enter}, keyList(vpnKeys)...)
}

func vpnPath(server bool, id, rest string) string {
	if server {
		return "/api/vpn-servers/" + url.PathEscape(id) + rest
	}
	return "/api/vpns/" + url.PathEscape(id) + rest
}

func loadVPN(a *App) tea.Cmd {
	return a.do("vpn", func(ctx context.Context, be Backend) tea.Msg {
		var m vpnListMsg
		m.cErr = be.Do(ctx, http.MethodGet, "/api/vpns", nil, &m.clients)
		m.sErr = be.Do(ctx, http.MethodGet, "/api/vpn-servers", nil, &m.servers)
		return m
	})
}

func (s *vpnScreen) count() int {
	if s.servers {
		return len(s.ss)
	}
	return len(s.cs)
}

func (s *vpnScreen) run(a *App, head, method, path string, in any, reload bool) tea.Cmd {
	s.res.pending(a, head)
	return a.call(scrVPN, head, method, path, in, reload)
}

func (s *vpnScreen) update(a *App, msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case enterMsg:
		return loadVPN(a)
	case vpnListMsg:
		if msg.cErr == nil {
			s.cs = msg.clients
		}
		if msg.sErr == nil {
			s.ss = msg.servers
		}
		s.cErr, s.sErr, s.has = msg.cErr, msg.sErr, true
		s.cursor = min(s.cursor, max(s.count()-1, 0))
	case callMsg:
		if msg.to != scrVPN {
			return nil
		}
		s.res.take(a, msg)
		if msg.reload {
			return loadVPN(a)
		}
	case vpnPeerMsg:
		if msg.err != nil {
			s.res = resultPane{head: "add peer " + sanitizeLine(msg.name) + ": FAIL", lines: []string{a.errText(msg.err)}, bad: true, has: true}
			a.flash = "add peer " + sanitizeLine(msg.name) + " failed"
			return nil
		}
		s.res = resultPane{head: "peer " + sanitizeLine(msg.name) + " on " + sanitizeLine(msg.server), lines: []string{"address " + sanitizeLine(msg.ip) + "; its config was shown once"}, has: true}
		a.flash = "added peer " + sanitizeLine(msg.name) + " to " + sanitizeLine(msg.server)
		a.modal = newOnce("Config for "+msg.name, "Copy it into the device's WireGuard app. It holds the device's private key.", msg.config)
		return loadVPN(a)
	case tea.KeyPressMsg:
		if a.active != scrVPN || a.detail != nil {
			return nil
		}
		return s.key(a, msg)
	}
	return nil
}

func (s *vpnScreen) key(a *App, k tea.KeyPressMsg) tea.Cmd {
	vk := vpnKeys
	switch {
	case key.Matches(k, navKeys.Up):
		s.cursor = max(s.cursor-1, 0)
		return nil
	case key.Matches(k, navKeys.Down):
		s.cursor = min(s.cursor+1, max(s.count()-1, 0))
		return nil
	case key.Matches(k, vk.Switch):
		s.servers, s.cursor, s.offset = !s.servers, 0, 0
		return nil
	case key.Matches(k, vk.Reload):
		return loadVPN(a)
	case key.Matches(k, vk.New):
		if s.servers {
			a.modal = s.newServer()
		} else {
			a.modal = s.newClient()
		}
		return nil
	}
	if s.cursor >= s.count() {
		return nil
	}
	srv := s.servers
	var id string
	if srv {
		id = s.ss[s.cursor].ID
	} else {
		id = s.cs[s.cursor].ID
	}
	switch {
	case key.Matches(k, navKeys.Enter) && srv:
		return s.run(a, id, http.MethodGet, vpnPath(true, id, ""), nil, false)
	case key.Matches(k, vk.Up):
		return s.run(a, "up "+id, http.MethodPost, vpnPath(srv, id, "/up"), nil, true)
	case key.Matches(k, vk.Down):
		return s.run(a, "down "+id, http.MethodPost, vpnPath(srv, id, "/down"), nil, true)
	case key.Matches(k, vk.Status):
		return s.run(a, "status "+id, http.MethodGet, vpnPath(srv, id, "/status"), nil, false)
	case key.Matches(k, vk.Delete):
		what := "overlay"
		if srv {
			what = "server"
		}
		a.modal = newConfirm("Delete the VPN "+what+" "+id, "Its tunnel goes down and its config is removed.", id, func() tea.Cmd {
			return s.run(a, "delete "+id, http.MethodDelete, vpnPath(srv, id, ""), nil, true)
		})
	case key.Matches(k, vk.Endpoint) && srv:
		a.modal = newAsk("Endpoint of "+id, "The public host or domain devices dial.", "", func(a *App, host string) (string, tea.Cmd) {
			if host == "" {
				return "type a host, or esc to cancel", nil
			}
			return "", s.run(a, "endpoint "+id, http.MethodPost, vpnPath(true, id, "/endpoint"), map[string]string{"endpointHost": host}, true)
		})
	case key.Matches(k, vk.AddPeer) && srv:
		a.modal = newAsk("New peer on "+id, "The device's name: "+plainNameMiss+".", "", func(a *App, name string) (string, tea.Cmd) {
			if !plainName.MatchString(name) {
				return plainNameMiss, nil
			}
			a.modal = newAsk("Route for "+name, "full: all of the device's traffic goes through the tunnel; split: only the VPN's network.", "split", func(a *App, v string) (string, tea.Cmd) {
				if v != "full" && v != "split" {
					return "type full or split", nil
				}
				return "", addPeer(a, id, name, v == "full")
			})
			return "", nil
		})
	case key.Matches(k, vk.RemovePeer) && srv:
		peers := s.ss[s.cursor].Peers
		a.modal = newAsk("Remove a peer from "+id, "The peer's name.", "", func(a *App, name string) (string, tea.Cmd) {
			for _, p := range peers {
				if p.Name == name {
					pk := p.PublicKey
					a.modal = newConfirm("Remove "+name+" from "+id, "The device loses access.", name, func() tea.Cmd {
						return s.run(a, "remove peer "+name, http.MethodPost, vpnPath(true, id, "/peers/remove"), map[string]string{"publicKey": pk}, true)
					})
					return "", nil
				}
			}
			return "no peer by that name", nil
		})
	}
	return nil
}

func addPeer(a *App, server, name string, full bool) tea.Cmd {
	return a.do("add peer", func(ctx context.Context, be Backend) tea.Msg {
		var raw json.RawMessage
		err := be.Do(ctx, http.MethodPost, vpnPath(true, server, "/peers"), map[string]any{"name": name, "fullTunnel": full}, &raw)
		defer clear(raw)
		if err != nil {
			return vpnPeerMsg{server: server, name: name, err: err}
		}
		var p vpnPeer
		_ = json.Unmarshal(raw, &p)
		return vpnPeerMsg{server: server, name: name, ip: p.AllowedIP, config: secretField(raw, "config")}
	})
}

func (s *vpnScreen) newServer() modal {
	return newAsk("New VPN server", "Its id: "+plainNameMiss+".", "", func(a *App, id string) (string, tea.Cmd) {
		if !plainName.MatchString(id) {
			return plainNameMiss, nil
		}
		a.modal = newAsk("New VPN server "+id, "The box it runs on.", "", func(a *App, box string) (string, tea.Cmd) {
			if box == "" {
				return "type a box name, or esc to cancel", nil
			}
			a.modal = newAsk("New VPN server "+id, "The public host devices dial (blank: derive it from the box).", "", func(a *App, host string) (string, tea.Cmd) {
				body := map[string]any{"id": id, "targetId": box}
				if host != "" {
					body["endpointHost"] = host
				}
				return "", s.run(a, "create "+id, http.MethodPost, "/api/vpn-servers", body, true)
			})
			return "", nil
		})
		return "", nil
	})
}

// newClient asks id, provider, interface and box in plain fields, then the
// WireGuard config in the masked field (a paste keeps its lines).
func (s *vpnScreen) newClient() modal {
	return newAsk("New VPN client", "Its id: "+plainNameMiss+".", "", func(a *App, id string) (string, tea.Cmd) {
		if !plainName.MatchString(id) {
			return plainNameMiss, nil
		}
		a.modal = newAsk("Provider of "+id, "A label for who runs the far end.", "bring-your-own", func(a *App, prov string) (string, tea.Cmd) {
			if prov == "" {
				return "type a label, or esc to cancel", nil
			}
			a.modal = newAsk("Interface of "+id, "The WireGuard interface name.", "wg0", func(a *App, iface string) (string, tea.Cmd) {
				if !plainName.MatchString(iface) {
					return plainNameMiss, nil
				}
				a.modal = newAsk("Box for "+id, "The box it runs on (blank: this machine).", "", func(a *App, box string) (string, tea.Cmd) {
					a.modal = newSecret("WireGuard config of "+id, "Paste the whole .conf; it holds a private key and is cleared after the save.", "config", false,
						func(a *App, conf []byte) tea.Cmd { return s.saveClient(a, id, prov, iface, box, conf) }).multiline()
					return "", nil
				})
				return "", nil
			})
			return "", nil
		})
		return "", nil
	})
}

func (s *vpnScreen) saveClient(a *App, id, prov, iface, box string, conf []byte) tea.Cmd {
	pre, _ := json.Marshal(map[string]any{"id": id, "provider": prov, "interface": iface, "targetId": box, "autostart": false})
	lead := string(pre[1:len(pre)-1]) + ","
	s.res.pending(a, "save "+id)
	return a.do("save "+id, func(ctx context.Context, be Backend) tea.Msg {
		defer clear(conf)
		body := jsonStringBody(lead, "config", conf)
		defer clear(body)
		err := be.Do(ctx, http.MethodPost, "/api/vpns", apiclient.RawJSON(body), nil)
		return callMsg{to: scrVPN, head: "save " + id, err: err, reload: true}
	})
}

func (s *vpnScreen) view(a *App, w, h int) string {
	other := "servers"
	if s.servers {
		other = "clients"
	}
	head := truncate(a.th.Dim.Render(" v "+other+"  n new  u/d up/down  s status  D delete  ? every key"), w, a.gl.Ellipsis)
	if !s.has {
		return head + "\n\n " + a.th.Dim.Render("loading"+a.gl.Ellipsis)
	}
	var lines []string
	sel := -1
	if s.servers {
		lines = append(lines, a.th.Dim.Render(" servers"))
		if s.sErr != nil {
			lines = append(lines, " "+a.th.Bad.Render("unavailable: ")+a.errText(s.sErr))
		} else if len(s.ss) == 0 {
			lines = append(lines, " "+a.th.Dim.Render("no VPN servers"))
		}
		for i, v := range s.ss {
			ep := v.Endpoint
			if ep == "" {
				ep = "no endpoint"
			}
			if i == s.cursor {
				sel = len(lines)
			}
			lines = append(lines, fmt.Sprintf("  %s %s %s :%-5d peers %-3d on %s  %s", pad(sanitizeLine(v.ID), 14), pad(sanitizeLine(v.Interface), 8),
				pad(sanitizeLine(v.Address), 16), v.ListenPort, len(v.Peers), sanitizeLine(v.TargetID), maskText(sanitizeLine(ep))))
			if i == s.cursor {
				for _, p := range v.Peers {
					lines = append(lines, "     "+pad(sanitizeLine(p.Name), 14)+" "+pad(sanitizeLine(p.AllowedIP), 16)+" "+sanitizeLine(p.PublicKey))
				}
			}
		}
	} else {
		lines = append(lines, a.th.Dim.Render(" clients"))
		if s.cErr != nil {
			lines = append(lines, " "+a.th.Bad.Render("unavailable: ")+a.errText(s.cErr))
		} else if len(s.cs) == 0 {
			lines = append(lines, " "+a.th.Dim.Render("no client overlays"))
		}
		for i, v := range s.cs {
			box := v.TargetID
			if box == "" {
				box = "this machine"
			}
			extra := ""
			if v.Autostart {
				extra += "  autostart"
			}
			if !v.Configured {
				extra += "  no config"
			}
			if i == s.cursor {
				sel = len(lines)
			}
			lines = append(lines, "  "+pad(sanitizeLine(v.ID), 14)+" "+pad(sanitizeLine(v.Provider), 16)+" "+pad(sanitizeLine(v.Interface), 8)+" on "+sanitizeLine(box)+extra)
		}
	}
	if sel >= 0 {
		s.offset = sel
		lines[sel] = " " + a.gl.Sel + lines[sel][2:]
	}
	res := s.res.view(a)
	listH := h - 2
	if len(res) > 0 {
		listH = max(min(len(lines), h-2-len(res)), 4)
		for i := range res {
			res[i] = truncate(res[i], w, a.gl.Ellipsis)
		}
	}
	out := head + "\n\n" + strings.Join(window(lines, &s.offset, listH, w, a.gl.Ellipsis), "\n")
	if len(res) > 0 {
		out += "\n" + strings.Join(res, "\n")
	}
	return out
}
