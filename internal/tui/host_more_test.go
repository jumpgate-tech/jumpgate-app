package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/tui/tuitest"
)

// tab moves the open host n tabs to the right and settles what that loads.
func tab(t *testing.T, a *App, n int) *App {
	t.Helper()
	var m tea.Model = a
	for i := 0; i < n; i++ {
		var cmds []tea.Cmd
		m, cmds = tuitest.Send(m, tuitest.Key("right"))
		m = tuitest.Settle(m, cmds...)
	}
	return m.(*App)
}

func gateway(id, host string) api.GatewaySummary {
	g := api.GatewaySummary{ID: id, Label: id, BaseURL: "https://rpc.example.org/", Networks: []api.GatewayNetwork{{ChainID: 369, Name: "PulseChain", URL: "https://rpc.example.org/369"}}}
	g.Placement.TargetID = host
	g.Status.State = "running"
	return g
}

func TestEndpointsTab(t *testing.T) {
	f := tuitest.NewFake()
	f.EndpointsV["box"] = api.Endpoints{ExecHTTP: "http://127.0.0.1:8545", BeaconHTTP: "http://127.0.0.1:5052", ExecReachable: true, ChainIDMatches: true,
		Access: "ssh", TunnelHint: "ssh -L 8545:127.0.0.1:8545 -L 5052:127.0.0.1:5052 root@10.0.0.5"}
	f.GatewaysV = []api.GatewaySummary{gateway("rpc-main", "box"), gateway("elsewhere", "other")}
	a := tab(t, openTestHost(t, f, "unicode"), 2)
	fr := tuitest.Frame(a)
	for _, want := range []string{"http://127.0.0.1:8545", "reachable", "chain id matches", "not reachable", "ssh -L 8545", "rpc-main", "PulseChain (369)", "read-only"} {
		if !strings.Contains(fr, want) {
			t.Errorf("endpoints lacks %q:\n%s", want, fr)
		}
	}
	if strings.Contains(fr, "elsewhere") {
		t.Errorf("a gateway on another box is listed:\n%s", fr)
	}
	tuitest.Golden(t, "host_endpoints", fr)
}

func TestSecurityTabShowsFixesForFailures(t *testing.T) {
	f := tuitest.NewFake()
	f.FirewallV["box"] = []api.CheckItem{
		{ID: "p2p", Title: "P2P port open", Status: "pass", Why: "peers must reach you"},
		{ID: "rpc", Title: "RPC not public", Status: "fail", Why: "anyone could use your node", Detail: "8545 is bound to 0.0.0.0", Fix: "sudo ufw deny 8545"},
	}
	a := tab(t, openTestHost(t, f, "unicode"), 3)
	fr := tuitest.Frame(a)
	for _, want := range []string{"ok", "P2P port open", "FAIL", "RPC not public", "8545 is bound to 0.0.0.0", "sudo ufw deny 8545", "1 to fix"} {
		if !strings.Contains(fr, want) {
			t.Errorf("security lacks %q:\n%s", want, fr)
		}
	}
}

func TestGatewaysScreenIsReadOnly(t *testing.T) {
	f := tuitest.NewFake()
	g := gateway("rpc-main", "box")
	g.Warnings = []string{"the certificate expires in 6 days"}
	f.GatewaysV = []api.GatewaySummary{g}
	a := newTestApp(t, f, 80, 24, "unicode")
	m, cmds := tuitest.Send(a, tuitest.Key("4"))
	m = tuitest.Settle(m, cmds...)
	fr := tuitest.Frame(m)
	for _, want := range []string{"read-only", "rpc-main", "on box", "running", "https://rpc.example.org/369", "certificate expires"} {
		if !strings.Contains(fr, want) {
			t.Errorf("gateways lacks %q:\n%s", want, fr)
		}
	}
	tuitest.Golden(t, "gateways", fr)
}

func TestMaskURL(t *testing.T) {
	for in, want := range map[string]string{
		"http://127.0.0.1:8545":                                         "http://127.0.0.1:8545",
		"https://rpc.example.org/369":                                   "https://rpc.example.org/369",
		"https://user:hunter2@rpc.example.org:8443/x":                   "https://***/x",
		"https://user:p@ss/w@rpc.example.org/":                          "https://***/***/",
		"https://tok3nvalue@rpc.example.org/":                           "https://***/",
		"https://mainnet.infura.io/v3/0123456789abcdef0123456789abcdef": "https://mainnet.infura.io/v3/***",
		"https://eth-mainnet.g.alchemy.com/v2/Ab3_dEf-Gh1jKlMnOpQrStUv": "https://eth-mainnet.g.alchemy.com/v2/***",
		"https://rpc.example.org/api/key/short":                         "https://rpc.example.org/api/key/***",
		"https://rpc.example.org/369?apikey=abc&chain=1#frag":           "https://rpc.example.org/369?apikey=***&chain=***#***",
		"https://rpc.example.org/execution-layer-endpoint-name":         "https://rpc.example.org/***",
	} {
		if got := maskURL(in); got != want {
			t.Errorf("maskURL(%q) = %q, want %q", in, got, want)
		}
	}
	if got := maskText("see https://u:pw@h/p?k=v now"); got != "*** *** ***" {
		t.Errorf("maskText = %q", got)
	}
}

func TestEndpointsAndGatewaysNeverShowCredentials(t *testing.T) {
	f := tuitest.NewFake()
	f.EndpointsV["box"] = api.Endpoints{ExecHTTP: "https://u:hunter2@rpc.example.org/v3/0123456789abcdef0123456789abcdef?apikey=SECRETQ", BeaconHTTP: "http://127.0.0.1:5052"}
	g := gateway("rpc-main", "box")
	g.BaseURL = "https://bob:swordfish@gw.example.org/"
	g.Networks[0].URL = "https://gw.example.org/369/0123456789abcdef0123456789abcdef"
	f.GatewaysV = []api.GatewaySummary{g}
	a := tab(t, openTestHost(t, f, "unicode"), 2)
	fr := tuitest.Frame(a)
	m, cmds := tuitest.Send(a, tuitest.Key("4"))
	fr += tuitest.Frame(tuitest.Settle(m, cmds...))
	for _, secret := range []string{"hunter2", "0123456789abcdef", "SECRETQ", "swordfish"} {
		if strings.Contains(fr, secret) {
			t.Errorf("%q is on screen:\n%s", secret, fr)
		}
	}
	if !strings.Contains(fr, "gw.example.org") || !strings.Contains(fr, "https://***/") {
		t.Errorf("host or masked userinfo missing:\n%s", fr)
	}
}

func TestHostileDataInEndpointsSecurityAndGateways(t *testing.T) {
	build := func(bad string) *tuitest.Fake {
		f := tuitest.NewFake()
		f.EndpointsV["box"] = api.Endpoints{ExecHTTP: "http://h" + bad + ":8545", BeaconHTTP: "http://b" + bad, Access: "ssh", TunnelHint: "ssh " + bad}
		f.FirewallV["box"] = []api.CheckItem{{ID: "x", Title: "T" + bad, Why: "W" + bad, Status: "fail", Detail: "D" + bad, Fix: "F" + bad + "\nG" + bad}}
		g := gateway("g"+bad, "box")
		g.Label, g.Status.State, g.BaseURL = "L"+bad+strings.Repeat("W", 80), "S"+bad, "https://u:p@h"+bad+"/"+strings.Repeat("z", 120)
		g.Networks = []api.GatewayNetwork{{ChainID: 1, Name: "N" + bad, URL: "https://n" + bad}}
		g.Warnings = []string{"warn" + bad + strings.Repeat("宽", 60)}
		f.GatewaysV = []api.GatewaySummary{g}
		return f
	}
	frames := func(bad string) []string {
		a := openTestHost(t, build(bad), "unicode")
		a.th = plainTheme() // every ESC left in a frame came from data
		var out []string
		for i := 0; i < 3; i++ {
			a = tab(t, a, 1)
			if i > 0 {
				out = append(out, a.View().Content)
			}
		}
		m, cmds := tuitest.Send(a, tuitest.Key("4"))
		return append(out, m.View().Content, tuitest.Settle(m, cmds...).View().Content)
	}
	base, bad := frames(""), frames(hostile)
	for i, fr := range bad {
		if strings.ContainsAny(fr, "\x1b\x07\x9b\u202e\r") {
			t.Errorf("frame %d carries a control from data:\n%q", i, fr)
		}
		if strings.Count(fr, "\n") != strings.Count(base[i], "\n") {
			t.Errorf("frame %d height changed", i)
		}
		for _, l := range strings.Split(fr, "\n") {
			if lipgloss.Width(l) > 80 {
				t.Errorf("frame %d line wider than 80: %q", i, l)
			}
		}
	}
}

func TestEndpointsAndFirewallMessagesFromAnOldOpenAreDropped(t *testing.T) {
	a := newTestApp(t, tuitest.NewFake(), 80, 24, "unicode")
	a.openHost("box")
	old := a.detail.(*hostScreen)
	a.closeDetail()
	a.openHost("box")
	m, cmds := tuitest.Send(a,
		endpointsMsg{id: "box", gen: old.gen, e: api.Endpoints{ExecHTTP: "http://old"}},
		firewallMsg{id: "box", gen: old.gen, items: []api.CheckItem{{Title: "old"}}})
	if h := m.(*App).detail.(*hostScreen); h.epsHas || h.fwHas || len(cmds) != 0 {
		t.Fatalf("the reopened host took the old open's data: %+v", h)
	}
}

func TestScrollClampsWhenDataShrinksAndOnResize(t *testing.T) {
	f := tuitest.NewFake()
	var items []api.CheckItem
	for i := 0; i < 40; i++ {
		items = append(items, api.CheckItem{ID: "c", Title: "check", Status: "fail", Fix: "fix"})
	}
	f.FirewallV["box"] = items
	a := tab(t, openTestHost(t, f, "unicode"), 3)
	h := a.detail.(*hostScreen)
	var m tea.Model = a
	for i := 0; i < 500; i++ {
		m, _ = tuitest.Send(m, tuitest.Key("down"))
		tuitest.Frame(m) // each frame clamps the scroll, as the program does
	}
	if h.scroll > 40*3 {
		t.Fatalf("scroll ran away: %d", h.scroll)
	}
	m, _ = tuitest.Send(m, firewallMsg{id: "box", gen: h.gen, items: items[:2]})
	if fr := tuitest.Frame(m); !strings.Contains(fr, "check") {
		t.Fatalf("shrunk data shows nothing:\n%s", fr)
	}
	m, _ = tuitest.Send(m, tea.WindowSizeMsg{Width: 200, Height: 60})
	if fr := tuitest.Frame(m); !strings.Contains(fr, "check") {
		t.Fatalf("resize lost the data:\n%s", fr)
	}
	// Gateways screen: scroll far, then shrink the list.
	f.GatewaysV = []api.GatewaySummary{gateway("a", "box"), gateway("b", "box"), gateway("c", "box")}
	m, cmds := tuitest.Send(m, tuitest.Key("4"))
	m = tuitest.Settle(m, cmds...)
	for i := 0; i < 50; i++ {
		m, _ = tuitest.Send(m, tuitest.Key("down"))
	}
	m, _ = tuitest.Send(m, gatewaysMsg{gws: f.GatewaysV[:1]})
	if fr := tuitest.Frame(m); !strings.Contains(fr, "rpc.example.org") {
		t.Fatalf("shrunk gateways show nothing:\n%s", fr)
	}
}

func TestEndpointsErrorAfterASuccessIsShownWithTheLastGoodData(t *testing.T) {
	f := tuitest.NewFake()
	f.EndpointsV["box"] = api.Endpoints{ExecHTTP: "http://127.0.0.1:8545", ExecReachable: true}
	f.FirewallV["box"] = []api.CheckItem{{Title: "P2P port open", Status: "pass"}}
	a := tab(t, openTestHost(t, f, "unicode"), 2)
	h := a.detail.(*hostScreen)
	m, _ := tuitest.Send(a, endpointsMsg{id: "box", gen: h.gen, err: &api.Error{Message: "probe timed out"}})
	fr := tuitest.Frame(m)
	if !strings.Contains(fr, "http://127.0.0.1:8545") || !strings.Contains(fr, "probe timed out") {
		t.Fatalf("an endpoints error after a success is hidden or drops the data:\n%s", fr)
	}
	m = tab(t, m.(*App), 1)
	m, _ = tuitest.Send(m, firewallMsg{id: "box", gen: h.gen, err: &api.Error{Message: "ssh dropped"}})
	fr = tuitest.Frame(m)
	if !strings.Contains(fr, "P2P port open") || !strings.Contains(fr, "ssh dropped") {
		t.Fatalf("a firewall error after a success is hidden or drops the data:\n%s", fr)
	}
}
