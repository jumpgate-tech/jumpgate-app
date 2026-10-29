package tui

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/apiclient"
	"github.com/valve-tech/jumpgate/internal/tui/tuitest"
)

func gwApp(t *testing.T, f *tuitest.Fake) tea.Model {
	t.Helper()
	f.GatewaysV = []api.GatewaySummary{gateway("rpc-main", "box")}
	return press(t, newTestApp(t, f, 100, 30, "unicode"), "4")
}

func TestGatewayStartShowsTheAnswer(t *testing.T) {
	f := tuitest.NewFake()
	f.Routes["POST /api/gateways/rpc-main/start"] = map[string]any{"status": map[string]any{"State": "running"}}
	m := press(t, gwApp(t, f), "s")
	fr := tuitest.Frame(m)
	if !strings.Contains(fr, "start rpc-main") || !strings.Contains(fr, "status.State: running") {
		t.Fatalf("frame:\n%s", fr)
	}
	if !f.Called("Gateways") {
		t.Fatal("no reload after start")
	}
	f.Routes["POST /api/gateways/rpc-main/restart"] = errors.New("box offline")
	if fr := tuitest.Frame(press(t, m, "e")); !strings.Contains(fr, "restart rpc-main: FAIL") || !strings.Contains(fr, "box offline") {
		t.Fatalf("frame:\n%s", fr)
	}
}

// Stop, delete, provision and wipe run only after the typed confirmation.
func TestGatewayDestructiveActionsAreTyped(t *testing.T) {
	for _, c := range []struct{ k, route string }{
		{"x", "POST /api/gateways/rpc-main/stop"},
		{"D", "DELETE /api/gateways/rpc-main"},
		{"P", "POST /api/gateways/rpc-main/provision"},
		{"W", "POST /api/gateways/rpc-main/wipe"},
	} {
		f := tuitest.NewFake()
		f.Routes[c.route] = map[string]any{}
		m := press(t, gwApp(t, f), c.k)
		if _, ok := m.(*App).modal.(*confirmModal); !ok || f.Called("Do "+c.route) {
			t.Fatalf("%s ran without a confirmation: %T", c.k, m.(*App).modal)
		}
		m = press(t, typed(t, m, "nope"), "enter")
		if f.Called("Do " + c.route) {
			t.Fatalf("%s ran on a wrong word", c.k)
		}
		press(t, typed(t, press(t, m, "ctrl+u"), "rpc-main"), "enter")
		if !f.Called("Do " + c.route) {
			t.Fatalf("%s did not run: %v", c.k, f.Calls)
		}
		if c.k == "W" {
			if b, _ := json.Marshal(f.Sent[c.route]); string(b) != `{"Confirm":"rpc-main"}` {
				t.Fatalf("wipe sent %s", b)
			}
		}
	}
}

func TestGatewayCreate(t *testing.T) {
	f := tuitest.NewFake()
	f.Routes["POST /api/gateways"] = map[string]any{"id": "gw2"}
	m := press(t, gwApp(t, f), "n")
	m = press(t, typed(t, m, "bad id!"), "enter")
	if fr := tuitest.Frame(m); !strings.Contains(fr, plainNameMiss) {
		t.Fatalf("frame:\n%s", fr)
	}
	m = press(t, typed(t, press(t, m, "ctrl+u"), "gw2"), "enter")
	press(t, typed(t, m, "box-a"), "enter")
	if b, _ := json.Marshal(f.Sent["POST /api/gateways"]); string(b) != `{"id":"gw2","placement":{"targetId":"box-a"}}` {
		t.Fatalf("sent %s", b)
	}
}

func TestGatewayReadsAndChainAsk(t *testing.T) {
	f := tuitest.NewFake()
	for _, r := range []string{"/api/gateways/rpc-main/tls/verify", "/api/gateways/rpc-main/capabilities", "/api/gateways/rpc-main/analytics",
		"/api/gateways/rpc-main/traffic", "/api/gateways/rpc-main/keys", "/api/chainlist", "/api/docker", "/api/host",
		"/api/gateways/rpc-main/knownset/369", "/api/chainlist/1", "/api/gateways/rpc-main"} {
		f.Routes["GET "+r] = map[string]any{"requests": 12}
	}
	m := gwApp(t, f)
	for _, k := range []string{"enter", "v", "a", "A", "t", "y", "L", "o", "H"} {
		m = press(t, m, k)
	}
	m = press(t, m, "K")
	m = press(t, typed(t, m, "abc"), "enter")
	if !strings.Contains(tuitest.Frame(m), "a chain id is a number") {
		t.Fatalf("frame:\n%s", tuitest.Frame(m))
	}
	m = press(t, typed(t, press(t, m, "ctrl+u"), "369"), "enter")
	m = press(t, typed(t, press(t, m, "N"), "1"), "enter")
	for _, r := range []string{"tls/verify", "capabilities", "analytics", "traffic", "rpc-main/keys", "chainlist", "docker", "host", "knownset/369", "chainlist/1", "/gateways/rpc-main"} {
		found := false
		for _, c := range f.Calls {
			if strings.HasPrefix(c, "Do GET") && strings.HasSuffix(c, r) {
				found = true
			}
		}
		if !found {
			t.Errorf("no GET …%s: %v", r, f.Calls)
		}
	}
	if fr := tuitest.Frame(m); !strings.Contains(fr, "requests: 12") {
		t.Fatalf("frame:\n%s", fr)
	}
}

func TestGatewayConfigIsJSON(t *testing.T) {
	f := tuitest.NewFake()
	f.Routes["PUT /api/gateways/rpc-main/config"] = map[string]any{}
	m := press(t, typed(t, press(t, gwApp(t, f), "G"), "{nope"), "enter")
	if !strings.Contains(tuitest.Frame(m), "that is not JSON") || f.Called("Do PUT") {
		t.Fatalf("frame:\n%s", tuitest.Frame(m))
	}
	press(t, typed(t, press(t, m, "ctrl+u"), `{"id":"rpc-main"}`), "enter")
	if raw, ok := f.Sent["PUT /api/gateways/rpc-main/config"].(apiclient.RawJSON); !ok || string(raw) != `{"id":"rpc-main"}` {
		t.Fatalf("sent %#v", f.Sent["PUT /api/gateways/rpc-main/config"])
	}
}

// The raw customer key is shown once in a modal: never in the flash, the
// result pane or the log, and gone once the modal closes.
func TestGatewayCustomerKeyShownOnce(t *testing.T) {
	const raw = "jg_live_0123456789abcdef"
	f := tuitest.NewFake()
	f.Routes["POST /api/gateways/rpc-main/keys"] = map[string]string{"id": "k1", "key": raw}
	m := press(t, gwApp(t, f), "+")
	m = press(t, typed(t, m, "acme"), "enter")
	if b, _ := json.Marshal(f.Sent["POST /api/gateways/rpc-main/keys"]); string(b) != `{"label":"acme"}` {
		t.Fatalf("sent %s", b)
	}
	a := m.(*App)
	once, ok := a.modal.(*onceModal)
	if !ok || !strings.Contains(tuitest.Frame(m), raw) {
		t.Fatalf("no show-once modal: %T\n%s", a.modal, tuitest.Frame(m))
	}
	if strings.Contains(a.flash, raw) {
		t.Fatalf("flash %q", a.flash)
	}
	for _, e := range a.log.ev {
		if strings.Contains(e.Text, raw) {
			t.Fatalf("logged: %q", e.Text)
		}
	}
	secret := once.secret
	m = press(t, m, "esc")
	if strings.Contains(tuitest.Frame(m), raw) || strings.Contains(string(secret), "jg_") || once.secret != nil {
		t.Fatalf("the key outlived the modal:\n%s", tuitest.Frame(m))
	}
}

func TestGatewayRevokeIsTyped(t *testing.T) {
	f := tuitest.NewFake()
	f.Routes["POST /api/gateways/rpc-main/keys/k1/revoke"] = map[string]string{"status": "revoked"}
	m := press(t, typed(t, press(t, gwApp(t, f), "-"), "k1"), "enter")
	if _, ok := m.(*App).modal.(*confirmModal); !ok || f.Called("Do POST") {
		t.Fatalf("no confirmation: %T", m.(*App).modal)
	}
	m = press(t, typed(t, m, "k1"), "enter")
	if !strings.Contains(tuitest.Frame(m), "status: revoked") {
		t.Fatalf("frame:\n%s", tuitest.Frame(m))
	}
}

func TestJSONStringBody(t *testing.T) {
	b := jsonStringBody(`"id":"x",`, "config", []byte("a\"b\\c\n\x01"))
	var v map[string]string
	if err := json.Unmarshal(b, &v); err != nil || v["config"] != "a\"b\\c\n\x01" || v["id"] != "x" {
		t.Fatalf("%s: %v %v", b, v, err)
	}
}
