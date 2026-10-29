package tui

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/apiclient"
)

// gatewayKeys act on the selected gateway, or on this machine (chain list,
// Docker, host). Global keys (g, i, q, R, U) and the cursor keys are taken.
var gatewayKeys = struct {
	Reload, New, Delete, Start, Stop, Restart, Config, Provision, Wipe, Trust, TLS, Caps, Known, Analytics, Traffic,
	Keys, NewKey, Revoke, Chains, Chain, Docker, StartDocker, Host key.Binding
}{
	Reload:      key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "reload")),
	New:         key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "new gateway")),
	Delete:      key.NewBinding(key.WithKeys("D"), key.WithHelp("D", "delete, typed confirmation")),
	Start:       key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "start")),
	Stop:        key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "stop, typed confirmation")),
	Restart:     key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "restart")),
	Config:      key.NewBinding(key.WithKeys("G"), key.WithHelp("G", "replace the config (JSON)")),
	Provision:   key.NewBinding(key.WithKeys("P"), key.WithHelp("P", "provision, typed confirmation")),
	Wipe:        key.NewBinding(key.WithKeys("W"), key.WithHelp("W", "wipe its data, typed confirmation")),
	Trust:       key.NewBinding(key.WithKeys("C"), key.WithHelp("C", "trust its certificate here")),
	TLS:         key.NewBinding(key.WithKeys("v"), key.WithHelp("v", "verify TLS")),
	Caps:        key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "capabilities")),
	Known:       key.NewBinding(key.WithKeys("K"), key.WithHelp("K", "known set of one chain")),
	Analytics:   key.NewBinding(key.WithKeys("A"), key.WithHelp("A", "analytics")),
	Traffic:     key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "traffic")),
	Keys:        key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "customer API keys")),
	NewKey:      key.NewBinding(key.WithKeys("+"), key.WithHelp("+", "new customer key (shown once)")),
	Revoke:      key.NewBinding(key.WithKeys("-"), key.WithHelp("-", "revoke a customer key")),
	Chains:      key.NewBinding(key.WithKeys("L"), key.WithHelp("L", "chain list")),
	Chain:       key.NewBinding(key.WithKeys("N"), key.WithHelp("N", "one chain")),
	Docker:      key.NewBinding(key.WithKeys("o"), key.WithHelp("o", "Docker readiness")),
	StartDocker: key.NewBinding(key.WithKeys("O"), key.WithHelp("O", "start Docker here")),
	Host:        key.NewBinding(key.WithKeys("H"), key.WithHelp("H", "this machine")),
}

// plainName is an id the server accepts in a path: letters, digits, '.',
// '_' and '-'.
var plainName = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

const plainNameMiss = "letters, digits, '.', '_' and '-' only"

func gwPath(id, rest string) string { return "/api/gateways/" + url.PathEscape(id) + rest }

// gwKeyMsg is a new customer key: key is the raw value, shown once.
type gwKeyMsg struct {
	gw, id string
	key    []byte
	err    error
}

// gatewaysScreen lists the eRPC gateways (spec D28) and runs every gateway
// call (inventory H): each answer shows under the list as key: value lines.
type gatewaysScreen struct {
	gws    []api.GatewaySummary
	has    bool
	err    error
	cursor int // the selected gateway
	offset int // clamped against the content when drawn
	res    resultPane
}

func (s *gatewaysScreen) capturing() bool { return false }
func (s *gatewaysScreen) keys() []key.Binding {
	return append([]key.Binding{navKeys.Up, navKeys.Down, navKeys.Enter, gwModKey}, keyList(gatewayKeys)...)
}

func (s *gatewaysScreen) selected() (api.GatewaySummary, bool) {
	if s.cursor < len(s.gws) {
		return s.gws[s.cursor], true
	}
	return api.GatewaySummary{}, false
}

func (s *gatewaysScreen) run(a *App, head, method, path string, in any, reload bool) tea.Cmd {
	s.res.pending(a, head)
	return a.call(scrGateways, head, method, path, in, reload)
}

func (s *gatewaysScreen) update(a *App, msg tea.Msg) tea.Cmd {
	if cmd, ok := s.gwModulesUpdate(a, msg); ok {
		return cmd
	}
	switch msg := msg.(type) {
	case enterMsg:
		return loadGateways(a)
	case gatewaysMsg:
		if msg.err != nil {
			s.err = msg.err
			break
		}
		s.gws, s.has, s.err = msg.gws, true, nil
		s.cursor = min(s.cursor, max(len(s.gws)-1, 0))
	case callMsg:
		if msg.to != scrGateways {
			return nil
		}
		s.res.take(a, msg)
		if msg.reload && msg.err == nil {
			return loadGateways(a)
		}
	case gwKeyMsg:
		if msg.err != nil {
			s.res = resultPane{head: "new key on " + sanitizeLine(msg.gw) + ": FAIL", lines: []string{a.errText(msg.err)}, bad: true, has: true}
			a.flash = "new key on " + sanitizeLine(msg.gw) + " failed"
			return nil
		}
		s.res = resultPane{head: "new key " + sanitizeLine(msg.id) + " on " + sanitizeLine(msg.gw), lines: []string{"shown once in the window; y lists the keys"}, has: true}
		a.flash = "created customer key " + sanitizeLine(msg.id) + " on " + sanitizeLine(msg.gw)
		a.modal = newOnce("Customer key "+msg.id, "Give it to the customer. jumpgate keeps only its hash.", msg.key)
	case tea.KeyPressMsg:
		if a.active != scrGateways || a.detail != nil {
			return nil
		}
		return s.key(a, msg)
	}
	return nil
}

func (s *gatewaysScreen) key(a *App, k tea.KeyPressMsg) tea.Cmd {
	gk := gatewayKeys
	// Keys about this machine or a new gateway need no selection.
	switch {
	case key.Matches(k, navKeys.Up):
		s.cursor = max(s.cursor-1, 0)
		return nil
	case key.Matches(k, navKeys.Down):
		s.cursor = min(s.cursor+1, max(len(s.gws)-1, 0))
		return nil
	case key.Matches(k, gk.Reload):
		return loadGateways(a)
	case key.Matches(k, gk.New):
		a.modal = newAsk("New gateway", "Its id: "+plainNameMiss+".", "", func(a *App, id string) (string, tea.Cmd) {
			if !plainName.MatchString(id) {
				return plainNameMiss, nil
			}
			a.modal = newAsk("New gateway "+id, "The box it runs on.", "", func(a *App, box string) (string, tea.Cmd) {
				if box == "" {
					return "type a box name, or esc to cancel", nil
				}
				body := map[string]any{"id": id, "placement": map[string]string{"targetId": box}}
				return "", s.run(a, "create "+id, http.MethodPost, "/api/gateways", body, true)
			})
			return "", nil
		})
		return nil
	case key.Matches(k, gk.Chains):
		return s.run(a, "chain list", http.MethodGet, "/api/chainlist", nil, false)
	case key.Matches(k, gk.Chain):
		a.modal = chainAsk("One chain", func(a *App, n string) tea.Cmd {
			return s.run(a, "chain "+n, http.MethodGet, "/api/chainlist/"+n, nil, false)
		})
		return nil
	case key.Matches(k, gk.Docker):
		return s.run(a, "Docker", http.MethodGet, "/api/docker", nil, false)
	case key.Matches(k, gk.StartDocker):
		a.modal = newConfirm("Start Docker on this machine", "jumpgate starts the Docker service here.", "start", func() tea.Cmd {
			return s.run(a, "start Docker", http.MethodPost, "/api/docker/start", nil, false)
		})
		return nil
	case key.Matches(k, gk.Host):
		return s.run(a, "this machine", http.MethodGet, "/api/host", nil, false)
	case key.Matches(k, gwModKey):
		if g, ok := s.selected(); ok {
			m, cmd := newGwModules(a, g)
			a.modal = m
			return cmd
		}
		return nil
	}
	g, ok := s.selected()
	if !ok {
		return nil
	}
	id := g.ID
	switch {
	case key.Matches(k, navKeys.Enter):
		return s.run(a, id, http.MethodGet, gwPath(id, ""), nil, false)
	case key.Matches(k, gk.Start):
		return s.run(a, "start "+id, http.MethodPost, gwPath(id, "/start"), nil, true)
	case key.Matches(k, gk.Restart):
		return s.run(a, "restart "+id, http.MethodPost, gwPath(id, "/restart"), nil, true)
	case key.Matches(k, gk.Stop):
		a.modal = newConfirm("Stop "+id, "Its RPC stops answering until it is started again.", id, func() tea.Cmd {
			return s.run(a, "stop "+id, http.MethodPost, gwPath(id, "/stop"), nil, true)
		})
	case key.Matches(k, gk.Delete):
		a.modal = newConfirm("Delete the gateway "+id, "Its config is removed; its customers' calls fail.", id, func() tea.Cmd {
			return s.run(a, "delete "+id, http.MethodDelete, gwPath(id, ""), nil, true)
		})
	case key.Matches(k, gk.Provision):
		a.modal = newConfirm("Provision "+id, "jumpgate creates or updates its container on "+g.Placement.TargetID+".", id, func() tea.Cmd {
			return s.run(a, "provision "+id, http.MethodPost, gwPath(id, "/provision"), nil, true)
		})
	case key.Matches(k, gk.Wipe):
		a.modal = newConfirm("Wipe "+id, "Its data is deleted.", id, func() tea.Cmd {
			return s.run(a, "wipe "+id, http.MethodPost, gwPath(id, "/wipe"), map[string]string{"Confirm": id}, true)
		})
	case key.Matches(k, gk.Trust):
		a.modal = newConfirm("Trust the certificate of "+id, "It is added to this machine's trust store; the OS may ask you.", "trust", func() tea.Cmd {
			return s.run(a, "trust cert "+id, http.MethodPost, gwPath(id, "/trust-cert"), nil, false)
		})
	case key.Matches(k, gk.Config):
		a.modal = newAsk("Config of "+id, "The whole config as one line of JSON; it replaces the stored one.", "", func(a *App, v string) (string, tea.Cmd) {
			if !json.Valid([]byte(v)) {
				return "that is not JSON", nil
			}
			return "", s.run(a, "config "+id, http.MethodPut, gwPath(id, "/config"), apiclient.RawJSON(v), true)
		})
	case key.Matches(k, gk.TLS):
		return s.run(a, "TLS "+id, http.MethodGet, gwPath(id, "/tls/verify"), nil, false)
	case key.Matches(k, gk.Caps):
		return s.run(a, "capabilities "+id, http.MethodGet, gwPath(id, "/capabilities"), nil, false)
	case key.Matches(k, gk.Known):
		a.modal = chainAsk("Known set of "+id, func(a *App, n string) tea.Cmd {
			return s.run(a, "known set "+id+" "+n, http.MethodGet, gwPath(id, "/knownset/"+n), nil, false)
		})
	case key.Matches(k, gk.Analytics):
		return s.run(a, "analytics "+id, http.MethodGet, gwPath(id, "/analytics"), nil, false)
	case key.Matches(k, gk.Traffic):
		return s.run(a, "traffic "+id, http.MethodGet, gwPath(id, "/traffic"), nil, false)
	case key.Matches(k, gk.Keys):
		return s.run(a, "customer keys "+id, http.MethodGet, gwPath(id, "/keys"), nil, false)
	case key.Matches(k, gk.NewKey):
		a.modal = newAsk("New customer key on "+id, "A label for it (may be blank).", "", func(a *App, label string) (string, tea.Cmd) {
			return "", newCustomerKey(a, id, label)
		})
	case key.Matches(k, gk.Revoke):
		a.modal = newAsk("Revoke a customer key on "+id, "The key id (y lists them).", "", func(a *App, kid string) (string, tea.Cmd) {
			if !plainName.MatchString(kid) {
				return plainNameMiss, nil
			}
			a.modal = newConfirm("Revoke "+kid+" on "+id, "Its calls start failing at once.", kid, func() tea.Cmd {
				return s.run(a, "revoke "+kid, http.MethodPost, gwPath(id, "/keys/"+url.PathEscape(kid)+"/revoke"), nil, false)
			})
			return "", nil
		})
	}
	return nil
}

// chainAsk asks for a chain id (a number) and runs next with it.
func chainAsk(title string, next func(a *App, n string) tea.Cmd) *askModal {
	return newAsk(title, "The chain id, a number.", "", func(a *App, v string) (string, tea.Cmd) {
		if v == "" || len(v) > 10 || strings.Trim(v, "0123456789") != "" {
			return "a chain id is a number", nil
		}
		return "", next(a, v)
	})
}

// newCustomerKey creates a key; the reply's raw key goes to the show-once
// modal and nowhere else.
func newCustomerKey(a *App, gw, label string) tea.Cmd {
	return a.do("new key", func(ctx context.Context, be Backend) tea.Msg {
		var raw json.RawMessage
		err := be.Do(ctx, http.MethodPost, gwPath(gw, "/keys"), map[string]string{"label": label}, &raw)
		defer clear(raw)
		if err != nil {
			return gwKeyMsg{gw: gw, err: err}
		}
		var head struct {
			ID string `json:"id"`
		}
		_ = json.Unmarshal(raw, &head)
		return gwKeyMsg{gw: gw, id: head.ID, key: secretField(raw, "key")}
	})
}

func (s *gatewaysScreen) view(a *App, w, h int) string {
	head := truncate(a.th.Dim.Render(" n new  s/x/e start/stop/restart  D delete  m modules  ? every key"), w, a.gl.Ellipsis) // the title says Gateways
	res := s.res.view(a)
	switch {
	case s.err != nil && !s.has:
		return head + "\n\n " + a.th.Bad.Render("unavailable: ") + a.errText(s.err)
	case !s.has:
		return head + "\n\n " + a.th.Dim.Render("loading"+a.gl.Ellipsis)
	case len(s.gws) == 0:
		return head + "\n\n " + a.th.Dim.Render("no gateways") + strings.Join(res, "\n")
	}
	var lines []string
	for i, g := range s.gws {
		gl := gatewayLines(a, g)
		if i == s.cursor {
			s.offset = len(lines)
			gl[0] = a.gl.Sel + gl[0][1:]
		}
		lines = append(lines, gl...)
		lines = append(lines, "")
	}
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
