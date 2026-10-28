package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/eip712"
)

// invitingServer answers pairing-info as a controller of version would.
func invitingServer(t *testing.T, version, scope string) {
	t.Helper()
	withServerAt(t, version, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/pairing-info" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(api.PairingInfo{
			Controller:   eip712.Address{0x01}.Hex(),
			TransportKey: "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIInviteTestKey jumpgate-controller's",
			System:       true, ServiceUID: 990, Scope: scope,
		})
	})
}

func TestHostsInvitePrintsAPinnedOneLiner(t *testing.T) {
	invitingServer(t, "v0.6.0", "")
	var out strings.Builder
	_, stderr := captureStdio(t)
	if code := hostsInvite(context.Background(), &out, "box1"); code != 0 {
		t.Fatalf("exit %d, stderr %q", code, stderr())
	}
	s := out.String()
	for _, want := range []string{
		"curl -fsSL https://jumpgate.app/install.sh | sh -s -- --agent",
		"--version v0.6.0",
		"--controller " + eip712.Address{0x01}.Hex(),
		`--transport-key 'ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIInviteTestKey jumpgate-controller'"'"'s'`,
		"then: jumpgate hosts add box1 --ssh jumpgate@HOST --installed",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("stdout lacks %q:\n%s", want, s)
		}
	}
}

func TestHostsInviteRefusesADevBuild(t *testing.T) {
	invitingServer(t, "dev", "")
	var out strings.Builder
	_, stderr := captureStdio(t)
	if code := hostsInvite(context.Background(), &out, "box1"); code != exitCode("failed") {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(stderr(), "released version") || out.Len() != 0 {
		t.Fatalf("stderr %q, stdout %q", stderr(), out.String())
	}
}

func TestHostsInviteRefusesSelfOnly(t *testing.T) {
	invitingServer(t, "v0.6.0", "self")
	var out strings.Builder
	_, stderr := captureStdio(t)
	if code := hostsInvite(context.Background(), &out, "box1"); code != api.CodeSelfOnly.Exit() || code == 0 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(stderr(), "self_only") || out.Len() != 0 {
		t.Fatalf("stderr %q, stdout %q", stderr(), out.String())
	}
}
