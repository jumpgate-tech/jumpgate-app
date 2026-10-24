package tui

import (
	"regexp"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/apiclient"
	"github.com/valve-tech/jumpgate/internal/tui/tuitest"
)

var agentWord = regexp.MustCompile(`(?i)\bagent\b`)

// User-facing text names the per-box program "gate"; only ssh-agent keeps the
// word.
func TestNoAgentInUserText(t *testing.T) {
	glyphOverride = "ascii"
	t.Cleanup(func() { glyphOverride = "" })

	cases := map[string]func(t *testing.T) tea.Model{
		"fleet": func(t *testing.T) tea.Model {
			f := tuitest.NewFake()
			f.PrefsV.FleetColumns = []string{"host", "link", "jobs", "agent"}
			return fleetApp(t, f, "ascii", apiclient.Live)
		},
		"hosts":         func(t *testing.T) tea.Model { return hostsApp(t, tuitest.NewFake(), "linux") },
		"hosts darwin":  func(t *testing.T) tea.Model { return press(t, hostsApp(t, tuitest.NewFake(), "darwin"), "L") },
		"signers":       func(t *testing.T) tea.Model { return signersApp(t, tuitest.NewFake()) },
		"host overview": func(t *testing.T) tea.Model { return openTestHost(t, tuitest.NewFake(), "ascii") },
		"settings":      func(t *testing.T) tea.Model { return settingsApp(t, tuitest.NewFake()) },
		"hostadd paired": func(t *testing.T) tea.Model {
			f := tuitest.NewFake()
			f.Probes = []api.HostKeyProbe{
				{Hops: []api.HostKeyHop{{HostPort: "bastion:22", State: api.HostKeyUnknown, ProbeID: "p1", Fingerprint: "SHA256:aaa", KeyType: "ssh-ed25519"}}},
				{AllConfirmed: true, Hops: []api.HostKeyHop{{State: api.HostKeyConfirmed}}},
			}
			f.PairV = []api.PairEvent{{Step: "upload", Line: "uploaded"}, {Done: true, Agent: "0xabc"}}
			m := press(t, hostsApp(t, f, "linux"), "a")
			m = typed(t, m, "box-c")
			m = press(t, m, "enter")
			m = typed(t, m, "root@10.0.0.7")
			m = press(t, m, "enter", "enter")
			m = typed(t, m, "ops@bastion")
			m = press(t, m, "enter", "enter")
			m = typed(t, m, "yes")
			return press(t, m, "enter")
		},
	}
	// Frames that must show the renamed text, so the case reached it.
	want := map[string]string{"hostadd paired": "paired: gate 0xabc", "hosts darwin": "a gate runs only on linux", "signers": "gate 0x1234"}
	for name, build := range cases {
		t.Run(name, func(t *testing.T) {
			fr := strings.ReplaceAll(strings.ToLower(tuitest.Frame(build(t))), "ssh-agent", "")
			if w := want[name]; w != "" && !strings.Contains(fr, w) {
				t.Errorf("frame lacks %q:\n%s", w, fr)
			}
			if agentWord.MatchString(fr) {
				t.Errorf("user text says agent:\n%s", fr)
			}
		})
	}
	if w := linkWord(api.FleetRow{Link: api.LinkAgent, Reachable: true}); w != "gate" {
		t.Errorf("linkWord = %q, want gate", w)
	}
}
