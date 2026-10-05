package server

import (
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/executor"
)

func TestTargetViewLinks(t *testing.T) {
	paired := &config.AgentPairing{Address: "0xabc", Transport: "ssh", PairedAt: time.Unix(1, 0).UTC(), NextSeq: 9, Socket: "/tmp/x"}
	local := &config.AgentPairing{Address: "0xdef", Transport: "local"}
	for _, c := range []struct {
		t    config.Target
		link api.Link
		this bool
	}{
		{config.Target{ID: "a", Mode: "ssh", SSH: &executor.SSHConfig{Host: "h", User: "u"}, Agent: paired}, api.LinkAgent, false},
		{config.Target{ID: "b", Mode: "local", Agent: local}, api.LinkAgentLocal, true},
		{config.Target{ID: "c", Mode: "ssh", SSH: &executor.SSHConfig{Host: "h", User: "u"}}, api.LinkSSHOnly, false},
		{config.Target{ID: "d", Mode: "local"}, api.LinkLocalOnly, true},
	} {
		v := targetView(c.t)
		if v.Link != c.link || v.ThisMachine != c.this {
			t.Errorf("%s: link %s this %v", c.t.ID, v.Link, v.ThisMachine)
		}
	}
	v := targetView(config.Target{ID: "a", Mode: "ssh", SSH: &executor.SSHConfig{Host: "h", User: "u"}, Agent: paired})
	b, _ := json.Marshal(v)
	var raw map[string]any
	_ = json.Unmarshal(b, &raw)
	agent := raw["agent"].(map[string]any)
	if agent["address"] != "0xabc" || agent["nextSeq"] != nil || agent["socket"] != nil {
		t.Fatalf("agent view leaks internals: %v", agent)
	}
	if raw["ssh"].(map[string]any)["Host"] != "h" {
		t.Fatalf("ssh field names changed: %v", raw["ssh"])
	}
}

func TestListTargetsShowsAgentAndLink(t *testing.T) {
	ts, token := contractServer(t,
		config.Target{ID: "a", Mode: "ssh", SSH: &executor.SSHConfig{Host: "h", User: "u"}, Agent: &config.AgentPairing{Address: "0xabc", Transport: "ssh"}},
		config.Target{ID: "c", Mode: "ssh", SSH: &executor.SSHConfig{Host: "h2", User: "u"}},
	)
	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/targets", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var views []api.TargetView
	if err := json.NewDecoder(res.Body).Decode(&views); err != nil {
		t.Fatal(err)
	}
	if len(views) != 2 || views[0].Link != api.LinkAgent || views[0].Agent == nil || views[1].Link != api.LinkSSHOnly || views[1].Agent != nil {
		t.Fatalf("views %+v", views)
	}
}

// TestSSHConfigRoundTrip checks sshConfig undoes sshView through the whole
// jump chain, so a request that carries an address stores what was shown, and
// that the caller's host-key policy reaches every hop (Ruling T4b).
func TestSSHConfigRoundTrip(t *testing.T) {
	c := &executor.SSHConfig{Host: "10.0.0.5", User: "root", KeyPath: "/k", HostKeyFile: "/kh", Port: 2222,
		Jump: &executor.SSHConfig{Host: "bastion", User: "ops", Port: 22, Jump: &executor.SSHConfig{Host: "edge", User: "e"}}}
	var checked []string
	policy := func(hostname string, _ net.Addr, _ ssh.PublicKey) error {
		checked = append(checked, hostname)
		return nil
	}
	back := sshConfig(sshView(c), policy)
	if back.Host != c.Host || back.User != c.User || back.KeyPath != c.KeyPath || back.HostKeyFile != c.HostKeyFile || back.Port != c.Port ||
		back.Jump == nil || back.Jump.Host != "bastion" || back.Jump.User != "ops" || back.Jump.Port != 22 ||
		back.Jump.Jump == nil || back.Jump.Jump.Host != "edge" || back.Jump.Jump.Jump != nil {
		t.Fatalf("round trip = %+v (jump %+v)", back, back.Jump)
	}
	hops := 0
	for h := back; h != nil; h = h.Jump {
		if h.HostKey == nil {
			t.Fatalf("hop %s has no host-key policy", h.Host)
		}
		_ = h.HostKey(h.Host, nil, nil)
		hops++
	}
	if hops != 3 || strings.Join(checked, ",") != "10.0.0.5,bastion,edge" {
		t.Fatalf("policy reached %v over %d hops, want every hop to carry the caller's policy", checked, hops)
	}
	if sshView(nil) != nil || sshConfig(nil, policy) != nil {
		t.Fatal("nil stays nil")
	}
}
