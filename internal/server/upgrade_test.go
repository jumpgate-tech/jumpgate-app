package server

import (
	"crypto/sha256"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/valve-tech/jumpgate/internal/agent"
	"github.com/valve-tech/jumpgate/internal/buildinfo"
	"github.com/valve-tech/jumpgate/internal/config"
)

// The in-process agent reports this test binary's arch and version, so the
// controller sends the agent for that arch and its wait sees the version it
// sent at once.
func TestAgentUpgradeSendsTheAgentForTheBoxArch(t *testing.T) {
	requireAgentPeer(t)
	var bin string
	var asked []string
	ts, token := pairedBoxWith(t, nopExec{}, pairedOpts{
		agent: func(home string, c *agent.Config) {
			bin = filepath.Join(home, "jumpgate")
			_ = os.WriteFile(bin, []byte("old agent"), 0o755)
			c.BinaryPath = bin
		},
		onServer: func(s *Server) {
			s.loadAgent = func(arch string) ([]byte, error) {
				asked = append(asked, arch)
				return []byte("new agent"), nil
			}
		},
	})
	res, out := postIntent(t, ts, token, "/api/targets/box/agent/upgrade", "")
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d: %+v", res.StatusCode, out)
	}
	if len(asked) != 1 || asked[0] != runtime.GOARCH {
		t.Fatalf("loadAgent asked for %v, want [%s]", asked, runtime.GOARCH)
	}
	if out["upgraded"] != true || out["to"] != buildinfo.Version() || out["from"] != buildinfo.Version() {
		t.Fatalf("response %+v", out)
	}
	got, _ := os.ReadFile(bin)
	if sha256.Sum256(got) != sha256.Sum256([]byte("new agent")) {
		t.Fatalf("box binary %q", got)
	}
}

func TestAgentUpgradeRefusesAnUnpairedTarget(t *testing.T) {
	ts, token := pairedLocal(t)
	_, _ = config.Update(func(c *config.Config) error {
		c.Targets = append(c.Targets, config.Target{ID: "raw", Mode: "local"})
		return nil
	})
	res, out := postIntent(t, ts, token, "/api/targets/raw/agent/upgrade", "")
	if res.StatusCode != http.StatusConflict || out["code"] != "not_paired" {
		t.Fatalf("status %d: %+v", res.StatusCode, out)
	}
}
