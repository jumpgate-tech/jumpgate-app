package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/valve-tech/jumpgate/internal/api"
)

func upgradingServer(t *testing.T, view api.AgentUpgradeView) {
	t.Helper()
	withServerAt(t, "v0.6.1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/targets/box/agent/upgrade" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(view)
	})
}

func TestHostsUpgradePrintsFromAndTo(t *testing.T) {
	upgradingServer(t, api.AgentUpgradeView{Upgraded: true, From: "v0.6.0", To: "v0.6.1"})
	var out strings.Builder
	_, stderr := captureStdio(t)
	if code := hostsUpgrade(context.Background(), &out, "box"); code != 0 {
		t.Fatalf("exit %d, stderr %q", code, stderr())
	}
	if got := strings.TrimSpace(out.String()); got != "box: gate v0.6.0 -> v0.6.1" {
		t.Fatalf("stdout %q", got)
	}
}

func TestHostsUpgradeSaysWhenAlreadyCurrent(t *testing.T) {
	upgradingServer(t, api.AgentUpgradeView{Upgraded: false, From: "v0.6.1", To: "v0.6.1"})
	var out strings.Builder
	_, _ = captureStdio(t)
	if code := hostsUpgrade(context.Background(), &out, "box"); code != 0 {
		t.Fatalf("exit %d", code)
	}
	if got := strings.TrimSpace(out.String()); got != "box: gate already v0.6.1" {
		t.Fatalf("stdout %q", got)
	}
}

// A self-only controller must refuse another box before it confirms a host
// key, saves the target or runs bootstrap on the box: in system mode the push
// runs here, before the server's own self_only check at pair time.
func TestHostsAddSSHRefusedBySelfOnlyBeforeTouchingTheBox(t *testing.T) {
	old := stdinIsTerminal
	t.Cleanup(func() { stdinIsTerminal = old })
	stdinIsTerminal = func() bool { return true }
	var other []string
	withServerAt(t, "v0.6.1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/api/pairing-info" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(api.PairingInfo{Controller: "0x0000000000000000000000000000000000000001", System: true, Scope: "self"})
			return
		}
		other = append(other, r.Method+" "+r.URL.Path)
		http.NotFound(w, r)
	})
	_, stderr := captureStdio(t)
	if code := hostsAdd([]string{"x", "--ssh", "root@192.0.2.1"}); code == 0 {
		t.Fatal("exit 0")
	}
	if !strings.Contains(stderr(), "self-only") {
		t.Fatalf("stderr %q", stderr())
	}
	for _, c := range other {
		if !strings.HasPrefix(c, "GET ") {
			t.Fatalf("a self-only controller still called %s", c)
		}
		if strings.Contains(c, "host-key") {
			t.Fatalf("a self-only controller still probed host keys: %s", c)
		}
	}
}
