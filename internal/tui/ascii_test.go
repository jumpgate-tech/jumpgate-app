package tui

import (
	"context"
	"errors"
	"reflect"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/apiclient"
	"github.com/valve-tech/jumpgate/internal/tui/tuitest"
)

// Review Focus 5 for every screen and modal: under the ascii glyph set a frame
// holds nothing but printable ASCII. Each case drives a screen through the
// same setup its own tests (and goldens) use, with every newTestApp forced to
// ascii, so a literal that bypasses the glyph set fails here.
func TestEveryScreenIsASCIIUnderASCIIGlyphs(t *testing.T) {
	glyphOverride = "ascii"
	t.Cleanup(func() { glyphOverride = "" })

	addFlowAt := func(t *testing.T, f *tuitest.Fake) tea.Model {
		f.Probes = []api.HostKeyProbe{
			{Hops: []api.HostKeyHop{{HostPort: "bastion:22", State: api.HostKeyUnknown, ProbeID: "p1", Fingerprint: "SHA256:aaa", KeyType: "ssh-ed25519"}}},
		}
		m := press(t, hostsApp(t, f, "linux"), "a")
		m = typed(t, m, "box-c")
		m = press(t, m, "enter")
		m = typed(t, m, "root@10.0.0.7")
		m = press(t, m, "enter", "enter")
		m = typed(t, m, "ops@bastion")
		return press(t, m, "enter", "enter")
	}
	explainFake := func(e api.Explain, provider string) *tuitest.Fake {
		f := tuitest.NewFake()
		f.ExplainV = e
		f.SettingsV = api.Settings{AIProvider: provider, AIDisclosure: "Explain sends the selected log lines to " + provider + "."}
		return f
	}

	cases := map[string]func(t *testing.T) tea.Model{
		"fleet": func(t *testing.T) tea.Model { return fleetApp(t, tuitest.NewFake(), "ascii", apiclient.Live) },
		"fleet narrow cut": func(t *testing.T) tea.Model {
			f := tuitest.NewFake()
			f.PrefsV.FleetColumns = []string{"host", "link", "chain", "sync", "head", "peers", "disk", "firewall", "jobs", "agent"}
			return fleetApp(t, f, "ascii", apiclient.Retrying)
		},
		"jobs": func(t *testing.T) tea.Model { return press(t, newTestApp(t, tuitest.NewFake(), 80, 24, "ascii"), "5") },
		"help": func(t *testing.T) tea.Model {
			return press(t, newTestApp(t, tuitest.NewFake(), 80, 24, "ascii"), "5", "?")
		},
		"palette": func(t *testing.T) tea.Model {
			return typed(t, press(t, newTestApp(t, tuitest.NewFake(), 80, 24, "ascii"), ":"), "logs box-a")
		},
		"too small": func(t *testing.T) tea.Model {
			m, _ := tuitest.Send(newTestApp(t, tuitest.NewFake(), 80, 24, "ascii"), tea.WindowSizeMsg{Width: 60, Height: 20})
			return m
		},
		"skew banner": func(t *testing.T) tea.Model {
			f := tuitest.NewFake()
			f.ServerVersion = "v0.8.0"
			return newTestApp(t, f, 80, 24, "ascii")
		},
		"hosts": func(t *testing.T) tea.Model { return hostsApp(t, tuitest.NewFake(), "linux") },
		"hosts remove confirm": func(t *testing.T) tea.Model {
			return press(t, hostsApp(t, tuitest.NewFake(), "linux"), "d")
		},
		"hostadd form": func(t *testing.T) tea.Model { return press(t, hostsApp(t, tuitest.NewFake(), "linux"), "a") },
		"hostadd confirm": func(t *testing.T) tea.Model {
			return addFlowAt(t, tuitest.NewFake())
		},
		"hostadd mismatch": func(t *testing.T) tea.Model {
			f := tuitest.NewFake()
			m := addFlowAt(t, f)
			fl := m.(*App).screens[scrHosts].(*hostsScreen).flow
			fl.step = stepProbing
			m, _ = tuitest.Send(m, probeMsg{gen: fl.gen, p: api.HostKeyProbe{Hops: []api.HostKeyHop{{HostPort: "10.0.0.7:22", State: api.HostKeyMismatch}}}})
			return m
		},
		"hostadd pairing failed": func(t *testing.T) tea.Model {
			f := tuitest.NewFake()
			m := addFlowAt(t, f)
			fl := m.(*App).screens[scrHosts].(*hostsScreen).flow
			fl.step = stepPairing
			m, _ = tuitest.Send(m, pairMsg{gen: fl.gen, ev: api.PairEvent{Step: "upload", Line: "agent uploaded"}})
			m, _ = tuitest.Send(m, pairMsg{gen: fl.gen, ev: api.PairEvent{Step: "start", Error: "boom", Code: api.CodeUpstream}})
			return m
		},
		"signers": func(t *testing.T) tea.Model {
			a := signersApp(t, tuitest.NewFake())
			a.o.Restart = func(context.Context) (Backend, error) { return nil, nil }
			return a
		},
		"signers error": func(t *testing.T) tea.Model {
			f := tuitest.NewFake()
			f.Err["Controller"] = errors.New("no store")
			return signersApp(t, f)
		},
		"gateways": func(t *testing.T) tea.Model {
			f := tuitest.NewFake()
			g := gateway("rpc-main", "box")
			g.Warnings = []string{"the certificate expires in 6 days"}
			f.GatewaysV = []api.GatewaySummary{g}
			return press(t, newTestApp(t, f, 80, 24, "ascii"), "4")
		},
		"settings": func(t *testing.T) tea.Model { return settingsApp(t, tuitest.NewFake()) },
		"settings key entry": func(t *testing.T) tea.Model {
			return typed(t, press(t, pick(t, settingsApp(t, tuitest.NewFake()), "AI key"), "enter"), "abc")
		},
		"host overview": func(t *testing.T) tea.Model { return openTestHost(t, tuitest.NewFake(), "ascii") },
		"host overview waiting": func(t *testing.T) tea.Model {
			a := newTestApp(t, tuitest.NewFake(), 80, 24, "ascii")
			return tuitest.Settle(a, a.openHost("box"))
		},
		"host storage": func(t *testing.T) tea.Model {
			f := tuitest.NewFake()
			f.DiskV["box"] = sampleDisk()
			return tab(t, openTestHost(t, f, "ascii"), 1)
		},
		"host endpoints": func(t *testing.T) tea.Model {
			f := tuitest.NewFake()
			f.EndpointsV["box"] = api.Endpoints{ExecHTTP: "http://127.0.0.1:8545", BeaconHTTP: "http://127.0.0.1:5052", ExecReachable: true, ChainIDMatches: true,
				Access: "ssh", TunnelHint: "ssh -L 8545:127.0.0.1:8545 root@10.0.0.5"}
			f.GatewaysV = []api.GatewaySummary{gateway("rpc-main", "box")}
			return tab(t, openTestHost(t, f, "ascii"), 2)
		},
		"host security": func(t *testing.T) tea.Model {
			f := tuitest.NewFake()
			f.FirewallV["box"] = []api.CheckItem{
				{ID: "p2p", Title: "P2P port open", Status: "pass", Why: "peers must reach you"},
				{ID: "rpc", Title: "RPC not public", Status: "fail", Why: "anyone could use your node", Detail: "8545 is bound to 0.0.0.0", Fix: "sudo ufw deny 8545"},
				{ID: "ssh", Title: "SSH keys only", Status: "warn", Why: "passwords can be guessed"},
			}
			return tab(t, openTestHost(t, f, "ascii"), 3)
		},
		"host logs":        func(t *testing.T) tea.Model { return logsHost(t, tuitest.NewFake()) },
		"host logs filter": func(t *testing.T) tea.Model { return typed(t, press(t, logsHost(t, tuitest.NewFake()), "/"), "peer") },
		"explain asking": func(t *testing.T) tea.Model {
			return press(t, logsHost(t, explainFake(api.Explain{}, "gemini")), "e")
		},
		"explain done remote": func(t *testing.T) tea.Model {
			return explain(t, logsHost(t, explainFake(api.Explain{Text: "The node lost its peers.", SentExcerpt: []string{"peer <ip-1> disconnected"}, Redacted: true}, "gemini")))
		},
		"explain done local": func(t *testing.T) tea.Model {
			return explain(t, logsHost(t, explainFake(api.Explain{Text: "ok", SentExcerpt: []string{"x"}}, "ollama")))
		},
		"explain failed": func(t *testing.T) tea.Model {
			f := explainFake(api.Explain{}, "groq")
			f.Err["Explain"] = &api.Error{Message: "upstream", Code: api.CodeUpstream, Status: 502}
			return explain(t, logsHost(t, f))
		},
		"host services": func(t *testing.T) tea.Model { return servicesHost(t, tuitest.NewFake()) },
		"host actions":  func(t *testing.T) tea.Model { return press(t, openTestHost(t, tuitest.NewFake(), "ascii"), "x") },
		"host stop confirm": func(t *testing.T) tea.Model {
			return press(t, openTestHost(t, tuitest.NewFake(), "ascii"), "x", "j", "j", "j", "enter")
		},
		"host sidebar": func(t *testing.T) tea.Model {
			f := tuitest.NewFake()
			f.FleetCh <- apiclient.Update[api.Fleet]{Has: true, State: apiclient.Live, Value: api.Fleet{At: testNow, Rows: []api.FleetRow{{ID: "box"}, {ID: "other"}}}}
			f.PrefsV.Glyphs = "ascii" // Init reads the prefs again
			a := openTestHost(t, f, "ascii")
			m := tuitest.Settle(a, a.Init())
			m, _ = tuitest.Send(m, tea.WindowSizeMsg{Width: 120, Height: 30})
			return m
		},
	}
	for name, build := range cases {
		t.Run(name, func(t *testing.T) {
			m := build(t)
			if !m.(*App).gl.ASCII {
				t.Fatal("the app is not on the ascii glyph set")
			}
			requireASCII(t, tuitest.Frame(m))
		})
	}
}

// Every glyph in the ascii set is ASCII, including any added later.
func TestASCIIGlyphSetIsASCII(t *testing.T) {
	v := reflect.ValueOf(asciiGlyphs)
	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		var all []string
		switch f.Kind() {
		case reflect.String:
			all = []string{f.String()}
		case reflect.Slice:
			all = f.Interface().([]string)
		default:
			continue
		}
		for _, s := range all {
			if s == "" {
				t.Errorf("%s is empty", v.Type().Field(i).Name)
			}
			requireASCII(t, s)
		}
	}
}
