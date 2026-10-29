package tui

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/apiclient"
	"github.com/valve-tech/jumpgate/internal/tui/tuitest"
)

func TestHostsUpgradeTheGate(t *testing.T) {
	f := tuitest.NewFake()
	f.Routes["POST /api/targets/box-a/agent/upgrade"] = api.AgentUpgradeView{Upgraded: true, From: "v0.5.0", To: "v0.6.0"}
	m := press(t, hostsApp(t, f, "linux"), "u")
	if fl := m.(*App).flash; fl != "gate on box-a: v0.5.0 -> v0.6.0" {
		t.Fatalf("flash %q", fl)
	}
	f.Routes["POST /api/targets/box-a/agent/upgrade"] = api.AgentUpgradeView{To: "v0.6.0"}
	if fl := press(t, m, "u").(*App).flash; fl != "the gate on box-a is already v0.6.0" {
		t.Fatalf("flash %q", fl)
	}
	f.Routes["POST /api/targets/box-a/agent/upgrade"] = errors.New("box offline")
	if fl := press(t, m, "u").(*App).flash; !strings.Contains(fl, "upgrade the gate on box-a: box offline") {
		t.Fatalf("flash %q", fl)
	}
}

func TestHostsUpgradeNeedsAGate(t *testing.T) {
	f := tuitest.NewFake()
	m := press(t, hostsApp(t, f, "linux"), "down", "u")
	if f.Called("Do POST") || !strings.Contains(m.(*App).flash, "no gate yet") {
		t.Fatalf("flash %q, calls %v", m.(*App).flash, f.Calls)
	}
}

func TestHostsForgetAHostKey(t *testing.T) {
	f := tuitest.NewFake()
	f.Routes["GET /api/hostkeys?hostPort=10.0.0.5%3A22"] = api.RecordedHostKeys{Keys: []api.RecordedHostKey{
		{Fingerprint: "SHA256:abc", KeyType: "ssh-ed25519", Store: api.HostKeyStoreJumpgate},
		{Fingerprint: "SHA256:def", KeyType: "ssh-rsa", Store: api.HostKeyStoreOpenSSH},
	}}
	f.Routes["POST /api/hostkeys/forget"] = nil
	m := press(t, hostsApp(t, f, "linux"), "H")
	fr := tuitest.Frame(m)
	for _, want := range []string{"Host keys recorded for box-a", "10.0.0.5:22  ssh-ed25519  SHA256:abc  jumpgate", "SHA256:def  openssh", "[f] forget"} {
		if !strings.Contains(fr, want) {
			t.Errorf("no %q:\n%s", want, fr)
		}
	}
	tuitest.Golden(t, "hosts_hostkeys", fr)
	// The OpenSSH key is never forgotten.
	m = press(t, m, "down", "f")
	if fr := tuitest.Frame(m); !strings.Contains(fr, "never edits") {
		t.Fatalf("openssh key offered:\n%s", fr)
	}
	m = press(t, m, "up", "f")
	if _, ok := m.(*App).modal.(*confirmModal); !ok || f.Called("Do POST /api/hostkeys/forget") {
		t.Fatalf("forget without a typed confirmation: %T %v", m.(*App).modal, f.Calls)
	}
	m = press(t, typed(t, m, "forget"), "enter")
	b, _ := json.Marshal(f.Sent["POST /api/hostkeys/forget"])
	if string(b) != `{"hostPort":"10.0.0.5:22","fingerprint":"SHA256:abc"}` {
		t.Fatalf("sent %s", b)
	}
	if !strings.Contains(m.(*App).flash, "forgot SHA256:abc for 10.0.0.5:22") {
		t.Fatalf("flash %q", m.(*App).flash)
	}
}

// A box behind a jump host has a key for each hop.
func TestHostKeysForEachHop(t *testing.T) {
	if got := strings.Join(hostPorts(&api.SSHView{Host: "10.0.0.6", Port: 2222, Jump: &api.SSHView{Host: "bastion"}}), " "); got != "10.0.0.6:2222 bastion:22" {
		t.Fatalf("hops %s", got)
	}
}

func TestHostsLeftoverContainers(t *testing.T) {
	f := tuitest.NewFake()
	f.Routes["GET /api/gateways"] = map[string]any{"orphans": []orphan{{ContainerName: "erpc-gw2", TargetID: "box-a", MergedInto: "gw1"}}}
	f.Routes["DELETE /api/orphans/erpc-gw2"] = map[string]string{"status": "dismissed"}
	m := press(t, hostsApp(t, f, "linux"), "o")
	if fr := tuitest.Frame(m); !strings.Contains(fr, "erpc-gw2 on box-a, merged into gw1") {
		t.Fatalf("no row:\n%s", fr)
	}
	m = press(t, m, "d")
	m = press(t, typed(t, m, "gw1"), "enter")
	if f.Called("Do DELETE") {
		t.Fatalf("dismissed on the wrong name")
	}
	m = press(t, typed(t, m, "erpc-gw2"), "enter")
	if !f.Called("Do DELETE /api/orphans/erpc-gw2") || !strings.Contains(m.(*App).flash, "dismissed erpc-gw2") {
		t.Fatalf("flash %q, calls %v", m.(*App).flash, f.Calls)
	}
	f.Routes["GET /api/gateways"] = map[string]any{}
	m = press(t, press(t, m, "esc"), "o")
	if fr := tuitest.Frame(m); !strings.Contains(fr, "none: no gateway merge") {
		t.Fatalf("empty:\n%s", fr)
	}
}

func TestSignersExportKeystore(t *testing.T) {
	a := signersApp(t, tuitest.NewFake())
	ran := recordSelf(a)
	m := press(t, typed(t, press(t, a, "e"), "-x"), "enter")
	if len(*ran) != 0 {
		t.Fatalf("a flag went through: %v", *ran)
	}
	m = press(t, m, "backspace", "backspace")
	press(t, typed(t, m, "/tmp/k.json"), "enter")
	if len(*ran) != 1 || selfArgs(t, (*ran)[0]) != "keys export --keystore-v3 /tmp/k.json" {
		t.Fatalf("ran %v", *ran)
	}
}

// On the jumpgate service the key is created over the API: the passphrase
// typed twice in a masked field, sent once, then cleared.
func TestSignersServiceCreatesItsKey(t *testing.T) {
	f := tuitest.NewFake()
	f.ControllerV = api.ControllerView{State: "missing"}
	var sent apiclient.RawJSON
	var got api.SystemInitKey
	f.Routes["POST /api/system/init-key"] = func(in any) (any, error) {
		sent = in.(apiclient.RawJSON)
		return nil, json.Unmarshal(sent, &got)
	}
	a := signersApp(t, f)
	ran := recordSelf(a)
	a.o.Server = func() ServerInfo { return ServerInfo{System: true} }
	m := typed(t, press(t, a, "K"), "SECRETpass")
	m = press(t, m, "enter")
	m = typed(t, m, "SECRETpasX")
	m = press(t, m, "enter")
	if fr := tuitest.Frame(m); !strings.Contains(fr, "did not match") || f.Called("Do POST") {
		t.Fatalf("a mismatch went through:\n%s", fr)
	}
	m = press(t, typed(t, m, "SECRETpass"), "enter")
	m = press(t, typed(t, m, "SECRETpass"), "enter")
	if got.Store != "vault" || string(got.Passphrase) != "SECRETpass" || len(*ran) != 0 {
		t.Fatalf("sent %+v, ran %v", got, *ran)
	}
	for _, b := range sent {
		if b != 0 {
			t.Fatalf("the body was not cleared")
		}
	}
	if fr := tuitest.Frame(m); strings.Contains(fr, "SECRET") {
		t.Fatalf("the passphrase is on screen:\n%s", fr)
	}
}
