package tui

import (
	"context"
	"errors"
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

// probingFlow starts a flow for name and submits its form without running
// the commands, so the flow stays at its first step and the test decides
// which replies arrive.
func probingFlow(t *testing.T, m tea.Model, name string) (tea.Model, *addFlow) {
	t.Helper()
	m = press(t, m, "a")
	m = typed(t, m, name)
	m, _ = tuitest.Send(m, tuitest.Key("enter"))
	m = typed(t, m, "root@10.0.0.7")
	m, _ = tuitest.Send(m, tuitest.Keys("enter", "enter", "enter", "enter")...)
	return m, m.(*App).screens[scrHosts].(*hostsScreen).flow
}

func unknownProbe(gen uint64) probeMsg {
	return probeMsg{gen: gen, p: api.HostKeyProbe{Hops: []api.HostKeyHop{{HostPort: "evil:22", State: api.HostKeyUnknown, ProbeID: "stale", Fingerprint: "SHA256:stale", KeyType: "ssh-ed25519"}}}}
}

// The generation token is what stops a reply for a flow that was left from
// acting on the NEXT flow when both wait at the same step.
func TestALateReplyFromAnEarlierFlowNeverReachesTheNextOne(t *testing.T) {
	newer := func(t *testing.T) (*tuitest.Fake, tea.Model, uint64, *addFlow) {
		f := tuitest.NewFake()
		m, a := probingFlow(t, hostsApp(t, f, "linux"), "box-a2")
		m = press(t, m, "esc") // A is left while it waits
		m, b := probingFlow(t, m, "box-b2")
		if a.gen == b.gen {
			t.Fatal("two flows share a generation")
		}
		return f, m, a.gen, b
	}
	t.Run("probe", func(t *testing.T) {
		f, m, old, b := newer(t)
		m, _ = tuitest.Send(m, unknownProbe(old))
		if b.step != stepProbing || b.cm != nil || strings.Contains(tuitest.Frame(m), "SHA256:stale") {
			t.Fatalf("A's probe reply reached B: step %v\n%s", b.step, tuitest.Frame(m))
		}
		_ = f
	})
	t.Run("confirm", func(t *testing.T) {
		f, m, old, b := newer(t)
		b.asked = true // B has its own confirmation in flight
		m, cmd := tuitest.Send(m, hostKeyConfirmedMsg{gen: old})
		if !b.asked || len(cmd) != 0 || b.step != stepProbing {
			t.Fatalf("A's confirm reply reached B: asked %v step %v", b.asked, b.step)
		}
		_ = f
		_ = m
	})
	t.Run("added", func(t *testing.T) {
		f, m, old, b := newer(t)
		b.step, b.adding = stepPairing, true // B's AddTarget is in flight
		m, cmd := tuitest.Send(m, targetAddedMsg{gen: old, err: errors.New("boom")})
		if !b.adding || b.step != stepPairing || len(cmd) != 0 {
			t.Fatalf("A's add reply reached B: adding %v step %v", b.adding, b.step)
		}
		_ = f
		_ = m
	})
	t.Run("pair", func(t *testing.T) {
		_, m, old, b := newer(t)
		b.step = stepPairing
		tuitest.Send(m, pairMsg{gen: old, ev: api.PairEvent{Done: true, Agent: "0xdead"}})
		if b.step != stepPairing || b.agent != "" {
			t.Fatalf("A's pair event reached B: step %v", b.step)
		}
	})
}

func TestEscFlashSaysOnlyWhatIsTrue(t *testing.T) {
	flashAfterEsc := func(t *testing.T, mutate func(f *addFlow)) string {
		m, fl := probingFlow(t, hostsApp(t, tuitest.NewFake(), "linux"), "box-c")
		mutate(fl)
		m, _ = tuitest.Send(m, tuitest.Key("esc"))
		return m.(*App).flash
	}
	t.Run("probing", func(t *testing.T) {
		if got := flashAfterEsc(t, func(*addFlow) {}); got != "stopped; no pairing was started" {
			t.Fatalf("flash %q", got)
		}
	})
	t.Run("confirming a key", func(t *testing.T) {
		// The typed yes was sent: the key may be on record already.
		got := flashAfterEsc(t, func(f *addFlow) { f.asked = true })
		if got != "stopped; the host key you confirmed may already be recorded, and no pairing was started" {
			t.Fatalf("flash %q", got)
		}
	})
	t.Run("adding", func(t *testing.T) {
		got := flashAfterEsc(t, func(f *addFlow) { f.step, f.adding = stepPairing, true })
		if !strings.Contains(got, "no pairing was started") || !strings.Contains(got, "may have been added unpaired") || !strings.Contains(got, "Hosts list") {
			t.Fatalf("flash %q", got)
		}
	})
	t.Run("pair not yet answered", func(t *testing.T) {
		got := flashAfterEsc(t, func(f *addFlow) { f.step = stepPairing })
		if !strings.Contains(got, "may have been added unpaired") {
			t.Fatalf("flash %q", got)
		}
	})
	t.Run("pairing under way", func(t *testing.T) {
		got := flashAfterEsc(t, func(f *addFlow) { f.step, f.started = stepPairing, true })
		if !strings.Contains(got, "finishes on the server") || strings.Contains(got, "no pairing was started") {
			t.Fatalf("flash %q", got)
		}
	})
}

func TestAddressErrorIsSaidOnce(t *testing.T) {
	m := press(t, hostsApp(t, tuitest.NewFake(), "linux"), "a")
	m = typed(t, m, "box-c")
	m = press(t, m, "enter")
	m = typed(t, m, "10.0.0.7")
	m = press(t, m, "enter", "enter", "enter", "enter")
	if fr := tuitest.Frame(m); strings.Count(fr, "want user@host") != 1 {
		t.Fatalf("frame:\n%s", fr)
	}
}

func TestConfirmReadsAsTheBriefAndNamesTheJump(t *testing.T) {
	f := tuitest.NewFake()
	f.Probes = []api.HostKeyProbe{
		{Hops: []api.HostKeyHop{{HostPort: "bastion:22", State: api.HostKeyUnknown, ProbeID: "p1", Fingerprint: "SHA256:aaa", KeyType: "ssh-ed25519"}}},
		{Hops: []api.HostKeyHop{{HostPort: "bastion:22", State: api.HostKeyConfirmed}, {HostPort: "10.0.0.7:22", State: api.HostKeyUnknown, ProbeID: "p2", Fingerprint: "SHA256:bbb", KeyType: "ssh-ed25519"}}},
	}
	m := press(t, hostsApp(t, f, "linux"), "a")
	m = typed(t, m, "box-c")
	m = press(t, m, "enter")
	m = typed(t, m, "root@10.0.0.7")
	m = press(t, m, "enter", "enter")
	m = typed(t, m, "ops@bastion")
	m = press(t, m, "enter", "enter")
	fr := tuitest.Frame(m)
	for _, want := range []string{"ADD A BOX", "bastion:22 is the jump host for box-c", "Type yes to trust it, esc to stop:", "Compare it with the box's console:"} {
		if !strings.Contains(fr, want) {
			t.Errorf("first hop lacks %q:\n%s", want, fr)
		}
	}
	m = typed(t, m, "yes")
	m = press(t, m, "enter")
	fr = tuitest.Frame(m)
	if !strings.Contains(fr, "10.0.0.7:22 is box-c") || strings.Contains(fr, "jump host") {
		t.Errorf("second hop:\n%s", fr)
	}
}

// Leaving the Hosts screen ends an add or pair flow there and then, and says
// what that left behind, in the same words esc uses; nothing is dropped
// silently on the way back.
func TestLeavingHostsEndsTheFlowAndSaysSo(t *testing.T) {
	leave := func(t *testing.T, m tea.Model, how string) tea.Model {
		t.Helper()
		switch how {
		case "screen":
			a := m.(*App)
			a.flash = ""
			a.openScreen(scrFleet)
			return a
		default: // a restart re-opens the screen it was on
			m, _ = tuitest.Send(m, restartedMsg{be: tuitest.NewFake()})
			return m
		}
	}
	for _, how := range []string{"screen", "restart"} {
		t.Run(how+"/probing", func(t *testing.T) {
			m, fl := probingFlow(t, hostsApp(t, tuitest.NewFake(), "linux"), "box-c")
			m = leave(t, m, how)
			if fl.ctx.Err() == nil || m.(*App).screens[scrHosts].(*hostsScreen).flow != nil {
				t.Fatal("the flow outlived the screen")
			}
			if !strings.Contains(m.(*App).flash, "stopped; no pairing was started") {
				t.Fatalf("flash %q", m.(*App).flash)
			}
		})
		t.Run(how+"/pairing under way", func(t *testing.T) {
			m, fl := probingFlow(t, hostsApp(t, tuitest.NewFake(), "linux"), "box-c")
			fl.step, fl.started = stepPairing, true
			m = leave(t, m, how)
			if fl.ctx.Err() == nil || !strings.Contains(m.(*App).flash, "finishes on the server") {
				t.Fatalf("ctx %v flash %q", fl.ctx.Err(), m.(*App).flash)
			}
		})
		t.Run(how+"/host key on screen", func(t *testing.T) {
			f := tuitest.NewFake()
			f.Probes = []api.HostKeyProbe{{Hops: []api.HostKeyHop{{HostPort: "10.0.0.7:22", State: api.HostKeyUnknown, ProbeID: "p1", Fingerprint: "SHA256:aaa", KeyType: "ssh-ed25519"}}}}
			m := press(t, hostsApp(t, f, "linux"), "a")
			m = typed(t, m, "box-c")
			m = press(t, m, "enter")
			m = typed(t, m, "root@10.0.0.7")
			m = press(t, m, "enter", "enter", "enter", "enter")
			fl := m.(*App).screens[scrHosts].(*hostsScreen).flow
			if fl == nil || fl.step != stepConfirm {
				t.Fatalf("not at the host key: %s", tuitest.Frame(m))
			}
			m = typed(t, m, "yes") // typed, not entered
			m = leave(t, m, how)
			if fl.ctx.Err() == nil || f.Called("ConfirmHostKey") {
				t.Fatalf("ctx %v calls %v", fl.ctx.Err(), f.Calls)
			}
			if got := m.(*App).flash; !strings.Contains(got, "host key was not trusted") {
				t.Fatalf("flash %q", got)
			}
			m = press(t, m, "2")
			if fr := tuitest.Frame(m); strings.Contains(fr, "SHA256:aaa") {
				t.Fatalf("the confirmation came back:\n%s", fr)
			}
		})
		t.Run(how+"/form", func(t *testing.T) {
			m := typed(t, press(t, hostsApp(t, tuitest.NewFake(), "linux"), "a"), "box-c")
			fl := m.(*App).screens[scrHosts].(*hostsScreen).flow
			m = leave(t, m, how)
			if fl.ctx.Err() == nil || !strings.Contains(m.(*App).flash, "nothing was added") {
				t.Fatalf("ctx %v flash %q", fl.ctx.Err(), m.(*App).flash)
			}
		})
	}
}
