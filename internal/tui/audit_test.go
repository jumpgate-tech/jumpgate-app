package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/apiclient"
	"github.com/valve-tech/jumpgate/internal/tui/tuitest"
)

// noEscape fails when a control byte from data reached the frame: none is
// left once the frame's own styling is stripped, and the raw View has no
// sequence the TUI never writes itself.
func noEscape(t *testing.T, what string, m tea.Model) {
	t.Helper()
	raw := m.View().Content
	// A text field styles its prompt and cursor (SGR only); nothing of the
	// TUI's own writes OSC, moves the cursor or clears the screen.
	for _, bad := range []string{"\x1b]", "\x1b[2J", "\x1b[H", "\x1b\\", "\x9b", "\u009b", "\x07", "\r", "\u202e"} {
		if strings.Contains(raw, bad) {
			t.Fatalf("%s: raw frame carries %q:\n%q", what, bad, raw)
		}
	}
	if s := ansi.Strip(raw); strings.ContainsAny(s, "\x1b\x07\r\u009b\u202e") {
		t.Fatalf("%s: stripped frame carries a control:\n%q", what, s)
	}
}

func plain(t *testing.T, m tea.Model) tea.Model {
	t.Helper()
	m.(*App).th = plainTheme()
	return m
}

// Server strings in the gateway, VPN, job and fleet views go through the
// sanitizer however they are drawn: list rows, result panes, confirmations
// and the show-once window.
func TestServerStringsCannotEscape(t *testing.T) {
	t.Run("gateways", func(t *testing.T) {
		f := tuitest.NewFake()
		g := gateway("rpc-main", "box"+hostile)
		g.Label, g.Status.State, g.BaseURL = "lbl"+hostile, "run"+hostile, "https://h.example/"+hostile
		g.Networks[0].Name = "net" + hostile
		g.Warnings = []string{"warn" + hostile}
		f.GatewaysV = []api.GatewaySummary{g}
		f.Routes["GET /api/gateways/rpc-main/analytics"] = map[string]any{"k" + hostile: "v" + hostile, "rows": []any{map[string]any{"m": hostile}}}
		f.Routes["POST /api/gateways/rpc-main/keys"] = map[string]any{"id": "id" + hostile, "key": "key" + hostile}
		f.Routes["POST /api/gateways/rpc-main/start"] = &api.Error{Message: "boom" + hostile, Hint: hostile}
		m := plain(t, press(t, newTestApp(t, f, 100, 30, "unicode"), "4"))
		noEscape(t, "list", m)
		noEscape(t, "analytics", press(t, m, "A"))
		noEscape(t, "error", press(t, m, "s"))
		noEscape(t, "provision confirm", press(t, m, "P"))
		m = press(t, m, "esc", "+")
		m = press(t, typed(t, m, "x"), "enter")
		if _, ok := m.(*App).modal.(*onceModal); !ok {
			t.Fatalf("no show-once modal: %T", m.(*App).modal)
		}
		noEscape(t, "show once", m)
	})
	t.Run("vpn", func(t *testing.T) {
		f := tuitest.NewFake()
		f.Routes["GET /api/vpns"] = []vpnClient{{ID: "c" + hostile, Provider: "p" + hostile, Interface: "i" + hostile, TargetID: "b" + hostile}}
		f.Routes["GET /api/vpn-servers"] = []vpnServer{{ID: "home", TargetID: "b" + hostile, Interface: "i" + hostile, Address: "a" + hostile,
			Endpoint: "https://e.example/" + hostile, Peers: []vpnPeer{{Name: "n" + hostile, PublicKey: "k" + hostile, AllowedIP: "ip" + hostile}}}}
		f.Routes["GET /api/vpn-servers/home/status"] = map[string]any{"s": hostile}
		f.Routes["POST /api/vpn-servers/home/peers"] = map[string]any{"allowedIp": hostile, "config": "[Interface]\n" + hostile}
		m := plain(t, press(t, newTestApp(t, f, 100, 30, "unicode"), "5"))
		noEscape(t, "clients", m)
		m = press(t, m, "v")
		noEscape(t, "servers", m)
		noEscape(t, "status", press(t, m, "s"))
		m = press(t, typed(t, press(t, m, "p"), "laptop"), "enter")
		m = press(t, m, "enter")
		if _, ok := m.(*App).modal.(*onceModal); !ok {
			t.Fatalf("no show-once modal: %T", m.(*App).modal)
		}
		noEscape(t, "peer config", m)
	})
	t.Run("jobs", func(t *testing.T) {
		f := tuitest.NewFake()
		in := sampleInbox()
		j := &in.Jobs[0]
		j.Target, j.Kind, j.Title, j.State = "box"+hostile, "kind"+hostile, "title"+hostile, "needs-decision"
		j.Steps[0].ID = "step" + hostile
		j.Question.Prompt = "prompt" + hostile
		j.Question.Options[0].Label = "opt" + hostile
		j.Lease.Label = "lease" + hostile
		f.JobsCh <- apiclient.Update[api.JobsInbox]{Value: in, Has: true, State: apiclient.Live}
		m := plain(t, press(t, newTestApp(t, f, 120, 40, "unicode"), "2"))
		noEscape(t, "inbox", m)
		noEscape(t, "job detail", press(t, m, "enter"))
	})
	t.Run("fleet", func(t *testing.T) {
		f := tuitest.NewFake()
		fl := sampleFleet()
		r := &fl.Rows[0]
		r.ID, r.Network, r.Agent = "box"+hostile, "net"+hostile, "gate"+hostile
		fl.Rows[1].Error = &api.Error{Message: "err" + hostile, Hint: hostile}
		a := newTestApp(t, f, 120, 40, "unicode")
		m, _ := tuitest.Send(a, fleetMsg{u: apiclient.Update[api.Fleet]{Value: fl, Has: true, State: apiclient.Live}})
		m = plain(t, m)
		noEscape(t, "fleet", m)
		noEscape(t, "hosts", press(t, m, "v"))
		noEscape(t, "box", press(t, m, "enter"))
	})
}
