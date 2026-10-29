package tui

import (
	"encoding/json"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/valve-tech/jumpgate/internal/apiclient"
	"github.com/valve-tech/jumpgate/internal/tui/tuitest"
)

func vpnFake() *tuitest.Fake {
	f := tuitest.NewFake()
	f.Routes["GET /api/vpns"] = []vpnClient{{ID: "office", Provider: "bring-your-own", Interface: "wg0", Configured: true}}
	f.Routes["GET /api/vpn-servers"] = []vpnServer{{ID: "home", TargetID: "box-a", Interface: "wg1", Address: "10.8.0.1/24", ListenPort: 51820,
		Endpoint: "vpn.example.org", Peers: []vpnPeer{{Name: "phone", PublicKey: "PUBKEY=", AllowedIP: "10.8.0.2/32"}}}}
	return f
}

func vpnApp(t *testing.T, f *tuitest.Fake) tea.Model {
	t.Helper()
	return press(t, newTestApp(t, f, 100, 30, "unicode"), "5")
}

func TestVPNListsClientsAndServers(t *testing.T) {
	f := vpnFake()
	m := vpnApp(t, f)
	fr := tuitest.Frame(m)
	for _, want := range []string{"VPN", "office", "bring-your-own", "on this machine"} {
		if !strings.Contains(fr, want) {
			t.Errorf("no %q:\n%s", want, fr)
		}
	}
	fr = tuitest.Frame(press(t, m, "v"))
	for _, want := range []string{"home", "10.8.0.1/24", ":51820", "peers 1", "phone", "10.8.0.2/32"} {
		if !strings.Contains(fr, want) {
			t.Errorf("no %q:\n%s", want, fr)
		}
	}
}

func TestVPNUpDownStatusDelete(t *testing.T) {
	f := vpnFake()
	f.Routes["POST /api/vpns/office/up"] = map[string]any{"up": true}
	f.Routes["POST /api/vpns/office/down"] = map[string]any{}
	f.Routes["GET /api/vpns/office/status"] = map[string]any{"handshake": "12s ago"}
	f.Routes["POST /api/vpn-servers/home/up"] = map[string]any{}
	f.Routes["GET /api/vpn-servers/home/status"] = map[string]any{"peers": 1}
	f.Routes["GET /api/vpn-servers/home"] = map[string]any{"id": "home"}
	f.Routes["DELETE /api/vpn-servers/home"] = nil
	m := press(t, vpnApp(t, f), "u", "d", "s")
	if !strings.Contains(tuitest.Frame(m), "handshake: 12s ago") {
		t.Fatalf("frame:\n%s", tuitest.Frame(m))
	}
	m = press(t, m, "v", "u", "s", "enter", "D")
	if _, ok := m.(*App).modal.(*confirmModal); !ok || f.Called("Do DELETE") {
		t.Fatalf("delete without a typed confirmation: %T", m.(*App).modal)
	}
	press(t, typed(t, m, "home"), "enter")
	for _, c := range []string{"Do POST /api/vpns/office/up", "Do POST /api/vpns/office/down", "Do GET /api/vpns/office/status",
		"Do POST /api/vpn-servers/home/up", "Do GET /api/vpn-servers/home/status", "Do GET /api/vpn-servers/home", "Do DELETE /api/vpn-servers/home"} {
		if !f.Called(c) {
			t.Errorf("no %s: %v", c, f.Calls)
		}
	}
}

// The client config goes through the masked field and is never drawn.
func TestVPNClientSaveMasksTheConfig(t *testing.T) {
	const conf = "[Interface]\nPrivateKey = SECRETKEY=\n[Peer]\nPublicKey = P=\n"
	f := vpnFake()
	f.Routes["POST /api/vpns"] = map[string]any{}
	m := vpnApp(t, f)
	m = press(t, typed(t, press(t, m, "n"), "lab"), "enter")
	m = press(t, m, "enter")                    // provider: bring-your-own
	m = press(t, m, "enter")                    // interface: wg0
	m = press(t, typed(t, m, "box-a"), "enter") // box
	if _, ok := m.(*App).modal.(*secretModal); !ok {
		t.Fatalf("no masked field: %T", m.(*App).modal)
	}
	m, _ = tuitest.Send(m, tea.PasteMsg{Content: conf})
	if fr := tuitest.Frame(m); strings.Contains(fr, "SECRETKEY") || strings.Contains(fr, "PrivateKey") {
		t.Fatalf("the config is drawn:\n%s", fr)
	}
	press(t, m, "enter")
	raw, ok := f.Sent["POST /api/vpns"].(apiclient.RawJSON)
	if !ok {
		t.Fatalf("sent %#v", f.Sent["POST /api/vpns"])
	}
	// The body buffer is cleared after the send.
	if strings.Contains(string(raw), "SECRETKEY") {
		t.Fatalf("the body was not cleared: %q", raw)
	}
}

func TestVPNClientBodyKeepsTheConfigLines(t *testing.T) {
	var got map[string]any
	f := vpnFake()
	f.Routes["POST /api/vpns"] = func(in any) (any, error) {
		_ = json.Unmarshal(in.(apiclient.RawJSON), &got)
		return map[string]any{}, nil
	}
	m := press(t, typed(t, press(t, vpnApp(t, f), "n"), "lab"), "enter", "enter", "enter", "enter")
	m, _ = tuitest.Send(m, tea.PasteMsg{Content: "[Interface]\r\nPrivateKey = K=\r\n"})
	press(t, m, "enter")
	if got["config"] != "[Interface]\nPrivateKey = K=\n" || got["id"] != "lab" || got["interface"] != "wg0" || got["provider"] != "bring-your-own" || got["targetId"] != "" {
		t.Fatalf("sent %v", got)
	}
}

// A new peer's config is shown once and dropped on close.
func TestVPNAddPeerShowsTheConfigOnce(t *testing.T) {
	const conf = "[Interface]\nPrivateKey = DEVICEKEY=\n"
	f := vpnFake()
	f.Routes["POST /api/vpn-servers/home/peers"] = map[string]any{"name": "laptop", "publicKey": "X=", "allowedIp": "10.8.0.3/32", "config": conf}
	m := press(t, vpnApp(t, f), "v", "p")
	m = press(t, typed(t, m, "laptop"), "enter")
	m = press(t, typed(t, press(t, m, "ctrl+u"), "full"), "enter")
	if b, _ := json.Marshal(f.Sent["POST /api/vpn-servers/home/peers"]); string(b) != `{"fullTunnel":true,"name":"laptop"}` {
		t.Fatalf("sent %s", b)
	}
	a := m.(*App)
	once, ok := a.modal.(*onceModal)
	if !ok || !strings.Contains(tuitest.Frame(m), "PrivateKey = DEVICEKEY=") {
		t.Fatalf("no show-once modal: %T\n%s", a.modal, tuitest.Frame(m))
	}
	if strings.Contains(a.flash, "DEVICEKEY") {
		t.Fatalf("flash %q", a.flash)
	}
	secret := once.secret
	m = press(t, m, "esc")
	if strings.Contains(tuitest.Frame(m), "DEVICEKEY") || strings.Contains(string(secret), "DEVICEKEY") {
		t.Fatalf("the config outlived the modal:\n%s", tuitest.Frame(m))
	}
}

func TestVPNEndpointAndRemovePeer(t *testing.T) {
	f := vpnFake()
	f.Routes["POST /api/vpn-servers/home/endpoint"] = map[string]any{}
	f.Routes["POST /api/vpn-servers/home/peers/remove"] = nil
	m := press(t, vpnApp(t, f), "v", "e")
	m = press(t, typed(t, m, "vpn2.example.org"), "enter")
	if b, _ := json.Marshal(f.Sent["POST /api/vpn-servers/home/endpoint"]); string(b) != `{"endpointHost":"vpn2.example.org"}` {
		t.Fatalf("sent %s", b)
	}
	m = press(t, typed(t, press(t, m, "x"), "tablet"), "enter")
	if !strings.Contains(tuitest.Frame(m), "no peer by that name") {
		t.Fatalf("frame:\n%s", tuitest.Frame(m))
	}
	m = press(t, typed(t, press(t, m, "ctrl+u"), "phone"), "enter")
	if f.Called("Do POST /api/vpn-servers/home/peers/remove") {
		t.Fatal("removed without a typed confirmation")
	}
	press(t, typed(t, m, "phone"), "enter")
	if b, _ := json.Marshal(f.Sent["POST /api/vpn-servers/home/peers/remove"]); string(b) != `{"publicKey":"PUBKEY="}` {
		t.Fatalf("sent %s", b)
	}
}

func TestVPNServerCreate(t *testing.T) {
	f := vpnFake()
	f.Routes["POST /api/vpn-servers"] = map[string]any{"server": map[string]any{"id": "s2"}}
	m := press(t, typed(t, press(t, vpnApp(t, f), "v", "n"), "s2"), "enter")
	m = press(t, typed(t, m, "box-a"), "enter")
	press(t, m, "enter")
	if b, _ := json.Marshal(f.Sent["POST /api/vpn-servers"]); string(b) != `{"id":"s2","targetId":"box-a"}` {
		t.Fatalf("sent %s", b)
	}
}
