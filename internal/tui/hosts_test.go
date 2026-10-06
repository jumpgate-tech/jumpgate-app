package tui

import (
	"context"
	"os/exec"
	"slices"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/tui/tuitest"
)

func sampleTargets() []api.TargetView {
	at := testNow.Add(-48 * time.Hour)
	return []api.TargetView{
		{ID: "box-a", Mode: "ssh", SSH: &api.SSHView{User: "root", Host: "10.0.0.5"}, Link: api.LinkAgent, Agent: &api.AgentView{Address: "0x1234567890abcdef1234567890abcdef12345678", Transport: "ssh", PairedAt: at}},
		{ID: "box-b", Mode: "ssh", SSH: &api.SSHView{User: "ops", Host: "10.0.0.6", Port: 2222, Jump: &api.SSHView{User: "ops", Host: "bastion"}}, Link: api.LinkSSHOnly},
		{ID: "laptop", Mode: "local", Link: api.LinkLocalOnly, ThisMachine: true},
	}
}

func hostsApp(t *testing.T, f *tuitest.Fake, goos string) *App {
	t.Helper()
	f.TargetsV = sampleTargets()
	a := newTestApp(t, f, 80, 24, "unicode")
	a.o.GOOS = goos
	return press(t, a, "2").(*App)
}

func TestHostsList(t *testing.T) {
	fr := tuitest.Frame(hostsApp(t, tuitest.NewFake(), "linux"))
	for _, want := range []string{"box-a", "root@10.0.0.5:22", "agent 0x1234…5678", "ops@10.0.0.6:2222 via ops@bastion:22", "not paired", "this machine", "a add", "L pair this machine"} {
		if !strings.Contains(fr, want) {
			t.Errorf("hosts lack %q:\n%s", want, fr)
		}
	}
	tuitest.Golden(t, "hosts", fr)
}

func TestAddFlowConfirmsEachHopThenPairs(t *testing.T) {
	f := tuitest.NewFake()
	f.Probes = []api.HostKeyProbe{
		{Hops: []api.HostKeyHop{{HostPort: "bastion:22", State: api.HostKeyUnknown, ProbeID: "p1", Fingerprint: "SHA256:aaa", KeyType: "ssh-ed25519"}}},
		{Hops: []api.HostKeyHop{{HostPort: "bastion:22", State: api.HostKeyConfirmed}, {HostPort: "10.0.0.7:22", State: api.HostKeyUnknown, ProbeID: "p2", Fingerprint: "SHA256:bbb"}}},
		{AllConfirmed: true, Hops: []api.HostKeyHop{{State: api.HostKeyConfirmed}, {State: api.HostKeyConfirmed}}},
	}
	f.PairV = []api.PairEvent{{Step: "upload", Line: "agent uploaded"}, {Done: true, Agent: "0xabc"}}
	m := press(t, hostsApp(t, f, "linux"), "a")
	m = typed(t, m, "box-c")
	m = press(t, m, "enter")
	m = typed(t, m, "root@10.0.0.7")
	m = press(t, m, "enter", "enter") // no key path: ssh-agent
	m = typed(t, m, "ops@bastion")
	m = press(t, m, "enter", "enter") // sudo: no
	fr := tuitest.Frame(m)
	if !strings.Contains(fr, "bastion:22") || !strings.Contains(fr, "SHA256:aaa") || !strings.Contains(fr, "ssh-keygen -lf") {
		t.Fatalf("first hop not shown:\n%s", fr)
	}
	tuitest.Golden(t, "hostadd_confirm", fr)
	m = typed(t, m, "y")
	m = press(t, m, "enter")
	if f.Called("ConfirmHostKey") {
		t.Fatal(`"y" confirmed a host key`)
	}
	m = typed(t, m, "yes")
	m = press(t, m, "enter")
	if fr := tuitest.Frame(m); !strings.Contains(fr, "SHA256:bbb") {
		t.Fatalf("second hop not shown:\n%s", fr)
	}
	m = typed(t, m, "yes")
	m = press(t, m, "enter")
	fr = tuitest.Frame(m)
	want := []string{"ConfirmHostKey p1 SHA256:aaa", "ConfirmHostKey p2 SHA256:bbb", "AddTarget box-c ssh", "Pair box-c false"}
	for _, w := range want {
		if !f.Called(w) {
			t.Errorf("missing call %q in %v", w, f.Calls)
		}
	}
	if !strings.Contains(fr, "paired: agent 0xabc") || !strings.Contains(fr, "agent uploaded") || !strings.Contains(fr, "PermitRootLogin no") {
		t.Fatalf("frame:\n%s", fr)
	}
}

func TestAddFlowStopsOnAMismatch(t *testing.T) {
	f := tuitest.NewFake()
	f.Probes = []api.HostKeyProbe{{Hops: []api.HostKeyHop{{HostPort: "10.0.0.7:22", State: api.HostKeyMismatch, Error: "SSH host key mismatch"}}}}
	m := press(t, hostsApp(t, f, "linux"), "a")
	m = typed(t, m, "box-c")
	m = press(t, m, "enter")
	m = typed(t, m, "root@10.0.0.7")
	m = press(t, m, "enter", "enter", "enter", "enter")
	fr := tuitest.Frame(m)
	if !strings.Contains(fr, "SECURITY") || f.Called("AddTarget") || f.Called("ConfirmHostKey") {
		t.Fatalf("calls %v frame:\n%s", f.Calls, fr)
	}
}

func TestAddFormRefusesABadAddress(t *testing.T) {
	f := tuitest.NewFake()
	m := press(t, hostsApp(t, f, "linux"), "a")
	m = typed(t, m, "box-c")
	m = press(t, m, "enter")
	m = typed(t, m, "10.0.0.7")
	m = press(t, m, "enter", "enter", "enter", "enter")
	if fr := tuitest.Frame(m); !strings.Contains(fr, "user@host") || f.Called("ProbeHostKeys") {
		t.Fatalf("frame:\n%s", fr)
	}
}

func TestPairThisMachineOffLinuxExplains(t *testing.T) {
	m := press(t, hostsApp(t, tuitest.NewFake(), "darwin"), "L")
	if fr := tuitest.Frame(m); !strings.Contains(fr, "needs Linux") {
		t.Fatalf("frame:\n%s", fr)
	}
}

func TestPairThisMachineRunsTheCLIInTheForeground(t *testing.T) {
	a := hostsApp(t, tuitest.NewFake(), "linux")
	a.o.Self = "/usr/local/bin/jumpgate"
	var got []string
	a.o.Command = func(name string, args ...string) *exec.Cmd {
		got = append([]string{name}, args...)
		return exec.Command("true")
	}
	press(t, a, "L")
	if !slices.Equal(got, []string{"/usr/local/bin/jumpgate", "hosts", "add", "laptop", "--local"}) {
		t.Fatalf("ran %v", got)
	}
}

func TestRemoveNeedsTheHostName(t *testing.T) {
	f := tuitest.NewFake()
	m := press(t, hostsApp(t, f, "linux"), "j", "d")
	m = typed(t, m, "box-a")
	m = press(t, m, "enter")
	if f.Called("RemoveTarget") {
		t.Fatal("the wrong name removed a box")
	}
	m = press(t, m, "esc", "d")
	m = typed(t, m, "box-b")
	press(t, m, "enter")
	if !f.Called("RemoveTarget box-b") {
		t.Fatalf("calls %v", f.Calls)
	}
}

func TestConfirmNeedsTheTypedWord(t *testing.T) {
	f := tuitest.NewFake()
	f.Probes = []api.HostKeyProbe{{Hops: []api.HostKeyHop{{HostPort: "10.0.0.7:22", State: api.HostKeyUnknown, ProbeID: "p1", Fingerprint: "SHA256:aaa", KeyType: "ssh-ed25519"}}}}
	m := press(t, hostsApp(t, f, "linux"), "a")
	m = typed(t, m, "box-c")
	m = press(t, m, "enter")
	m = typed(t, m, "root@10.0.0.7")
	m = press(t, m, "enter", "enter", "enter", "enter")
	m = press(t, m, "enter", "enter") // Enter alone: nothing
	for _, w := range []string{"YES", "yes please", "no", ""} {
		m = typed(t, m, w)
		m = press(t, m, "enter")
	}
	if f.Called("ConfirmHostKey") || f.Called("AddTarget") || f.Called("Pair") {
		t.Fatalf("a wrong answer trusted the key: %v", f.Calls)
	}
	m = press(t, m, "esc")
	if f.Called("ConfirmHostKey") || f.Called("AddTarget") {
		t.Fatalf("esc trusted the key: %v", f.Calls)
	}
	if fr := tuitest.Frame(m); !strings.Contains(fr, "HOSTS") {
		t.Fatalf("esc did not return to the list:\n%s", fr)
	}
}

func TestConfirmSendsTheFingerprintShown(t *testing.T) {
	f := tuitest.NewFake()
	f.Probes = []api.HostKeyProbe{
		{Hops: []api.HostKeyHop{{HostPort: "10.0.0.7:22", State: api.HostKeyUnknown, ProbeID: "probe-9", Fingerprint: "SHA256:Zk+/9x", KeyType: "ssh-rsa"}}},
		{AllConfirmed: true},
	}
	m := press(t, hostsApp(t, f, "linux"), "a")
	m = typed(t, m, "box-c")
	m = press(t, m, "enter")
	m = typed(t, m, "root@10.0.0.7")
	m = press(t, m, "enter", "enter", "enter", "enter")
	fr := tuitest.Frame(m)
	if !strings.Contains(fr, "ssh-rsa") || !strings.Contains(fr, "SHA256:Zk+/9x") {
		t.Fatalf("key not shown:\n%s", fr)
	}
	m = typed(t, m, "yes")
	press(t, m, "enter")
	if !f.Called("ConfirmHostKey probe-9 SHA256:Zk+/9x") {
		t.Fatalf("calls %v", f.Calls)
	}
}

// A key the screen could not show as sent is never confirmed.
func TestConfirmRefusesAKeyItCannotShowFaithfully(t *testing.T) {
	f := tuitest.NewFake()
	f.Probes = []api.HostKeyProbe{{Hops: []api.HostKeyHop{{HostPort: "h:22", State: api.HostKeyUnknown, ProbeID: "p", Fingerprint: "SHA256:aa\x1b[2Jbb"}}}}
	m := press(t, hostsApp(t, f, "linux"), "a")
	m = typed(t, m, "box-c")
	m = press(t, m, "enter")
	m = typed(t, m, "root@10.0.0.7")
	m = press(t, m, "enter", "enter", "enter", "enter")
	m = typed(t, m, "yes")
	m = press(t, m, "enter")
	if f.Called("ConfirmHostKey") || strings.Contains(m.View().Content, "\x1b[2J") {
		t.Fatalf("calls %v", f.Calls)
	}
}

func TestMismatchOffersNoAccept(t *testing.T) {
	f := tuitest.NewFake()
	f.Probes = []api.HostKeyProbe{{Hops: []api.HostKeyHop{{HostPort: "10.0.0.7:22", State: api.HostKeyMismatch, ProbeID: "p1", Fingerprint: "SHA256:evil"}}}}
	m := press(t, hostsApp(t, f, "linux"), "a")
	m = typed(t, m, "box-c")
	m = press(t, m, "enter")
	m = typed(t, m, "root@10.0.0.7")
	m = press(t, m, "enter", "enter", "enter", "enter")
	m = typed(t, m, "yes")
	press(t, m, "enter")
	if f.Called("ConfirmHostKey") || f.Called("AddTarget") {
		t.Fatalf("calls %v", f.Calls)
	}
}

func TestActionKeysTypeTextInTheForm(t *testing.T) {
	f := tuitest.NewFake()
	m := press(t, hostsApp(t, f, "linux"), "a")
	m = typed(t, m, "adprLd")
	if fr := tuitest.Frame(m); !strings.Contains(fr, "adprLd") || !strings.Contains(fr, "ADD A BOX") {
		t.Fatalf("frame:\n%s", fr)
	}
	if f.Called("RemoveTarget") || f.Called("Pair") {
		t.Fatalf("calls %v", f.Calls)
	}
}

func TestFormPasteIsSanitizedAndCtrlVIsOff(t *testing.T) {
	f := tuitest.NewFake()
	m := press(t, hostsApp(t, f, "linux"), "a")
	m, _ = tuitest.Send(m, tea.PasteMsg{Content: "bo\x1b]0;pwn\x07x\n-c"})
	m, _ = tuitest.Send(m, tuitest.Key("ctrl+v"))
	got := m.View().Content
	if strings.Contains(got, "\x1b]0;pwn") || strings.Contains(got, "\x07") {
		t.Fatalf("escape reached the frame: %q", got)
	}
	if fr := tuitest.Frame(m); !strings.Contains(fr, "box -c") {
		t.Fatalf("frame:\n%s", fr)
	}
}

func TestHostsSanitizesTargetText(t *testing.T) {
	f := tuitest.NewFake()
	f.TargetsV = []api.TargetView{{ID: "evil\x1b]52;c;AAAA\x07", Mode: "ssh", SSH: &api.SSHView{User: "r\x1b[2J", Host: "h"}, Link: api.LinkSSHOnly}}
	a := newTestApp(t, f, 80, 24, "unicode")
	m := press(t, a, "2")
	if c := m.View().Content; strings.Contains(c, "\x1b]52") || strings.Contains(c, "\x1b[2J") || strings.Contains(c, "\x07") {
		t.Fatalf("raw escape in frame: %q", c)
	}
}

func TestPairEventsAreSanitizedAndStaleOnesDropped(t *testing.T) {
	f := tuitest.NewFake()
	f.PairV = []api.PairEvent{{Step: "up\x1b[2J", Line: "line\x1b]0;x\x07"}, {Error: "boom\x1b[31m", Code: api.CodeHostKey}}
	m := press(t, hostsApp(t, f, "linux"), "a")
	m = typed(t, m, "box-c")
	m = press(t, m, "enter")
	m = typed(t, m, "root@10.0.0.7")
	m = press(t, m, "enter", "enter", "enter", "enter")
	c := m.View().Content
	if strings.Contains(c, "\x1b[2J") || strings.Contains(c, "\x1b]0") || strings.Contains(c, "\x1b[31m") && strings.Contains(c, "boom\x1b[31m") {
		t.Fatalf("raw escape: %q", c)
	}
	if fr := tuitest.Frame(m); !strings.Contains(fr, "pairing failed") || !strings.Contains(fr, "boom") {
		t.Fatalf("frame:\n%s", fr)
	}
	// Leave, start a new flow, then a late reply from the old one lands.
	old := m.(*App).screens[scrHosts].(*hostsScreen).flow.gen
	m = press(t, m, "esc", "a")
	m, _ = tuitest.Send(m, pairMsg{gen: old, ev: api.PairEvent{Done: true, Agent: "0xdead"}}, probeMsg{gen: old, p: api.HostKeyProbe{AllConfirmed: true}}, hostKeyConfirmedMsg{gen: old}, targetAddedMsg{gen: old})
	if fr := tuitest.Frame(m); strings.Contains(fr, "0xdead") || !strings.Contains(fr, "ADD A BOX") || strings.Count(strings.Join(f.Calls, "\n"), "AddTarget") != 1 {
		t.Fatalf("stale message acted:\n%s", fr)
	}
}

func TestWaitPairStopsWithItsContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan api.PairEvent) // never sends, never closes
	done := make(chan tea.Msg, 1)
	go func() { done <- waitPair(ctx, 1, ch)() }()
	cancel()
	select {
	case m := <-done:
		if !m.(pairMsg).closed {
			t.Fatalf("got %+v", m)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("waitPair leaked: it did not stop on cancel")
	}
}

func TestLeavingTheFlowCancelsItsContext(t *testing.T) {
	m := press(t, hostsApp(t, tuitest.NewFake(), "linux"), "a")
	fl := m.(*App).screens[scrHosts].(*hostsScreen).flow
	press(t, m, "esc")
	if fl.ctx.Err() == nil {
		t.Fatal("the flow's context outlived the flow")
	}
}

func TestRemoveRefusesAnUnshowableName(t *testing.T) {
	f := tuitest.NewFake()
	f.TargetsV = []api.TargetView{{ID: "bad\x1b[2Jname", Mode: "ssh", SSH: &api.SSHView{User: "r", Host: "h"}}}
	a := newTestApp(t, f, 80, 24, "unicode")
	m := press(t, a, "2", "d")
	m = press(t, m, "enter")
	m = typed(t, m, "badname")
	press(t, m, "enter")
	if f.Called("RemoveTarget") {
		t.Fatalf("calls %v", f.Calls)
	}
}

func TestPairThisMachineNameIsNeverRawHostname(t *testing.T) {
	for in, want := range map[string]string{"laptop": "laptop", "My-Box.local": "my-box-local", "-oProxy": "oproxy", "": "local", "\x1b[2J": "2j", "--": "local"} {
		if got := localName(in); got != want {
			t.Errorf("localName(%q) = %q, want %q", in, got, want)
		}
	}
}
