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
