package server

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/valve-tech/jumpgate/internal/executor"
	"github.com/valve-tech/jumpgate/internal/intent"
)

// publicRPCBox is a box whose exec HTTP port listens on a public address.
type publicRPCBox struct{ nopExec }

func (publicRPCBox) Run(_ context.Context, cmd string, _ *executor.RunOpts) (executor.Result, error) {
	if strings.HasPrefix(cmd, "ss -ltn") {
		return executor.Result{Stdout: "State Recv-Q Send-Q Local Address:Port Peer Address:Port\nLISTEN 0 4096 203.0.113.5:8545 0.0.0.0:*\n"}, nil
	}
	return executor.Result{}, nil
}
func (publicRPCBox) WriteFile(context.Context, string, []byte, os.FileMode) error { return nil }

// SECURITY: the overlay networks firewall.read treats as trusted are the
// server's own configuration. A request body naming the box's public
// network as an "overlay" must not turn a public RPC bind into a pass.
func TestFirewallReadIgnoresOverlaysFromTheRequest(t *testing.T) {
	requireAgentPeer(t)
	ts, token := pairedBox(t, publicRPCBox{}, true)
	res, out := postIntent(t, ts, token, "/api/targets/box/intent/"+intent.KindFirewallRead, `{"overlayCidrs":["203.0.113.0/24"]}`)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("%d %v", res.StatusCode, out)
	}
	raw, _ := json.Marshal(out["result"])
	var items []struct{ ID, Status string }
	if err := json.Unmarshal(raw, &items); err != nil {
		t.Fatalf("result %s: %v", raw, err)
	}
	for _, it := range items {
		if it.ID == "rpc-not-public" {
			if it.Status != "fail" {
				t.Fatalf("a public RPC bind graded %q under a request-supplied overlay", it.Status)
			}
			return
		}
	}
	t.Fatalf("no rpc-not-public item in %s", raw)
}

// The generic route checks what it can before signing: an unknown box is
// 404, and a payload over 1 MiB is refused whole.
func TestIntentRouteRefusesUnknownBoxesAndOversizedPayloads(t *testing.T) {
	ts, token := pairedLocal(t)
	if res, _ := postIntent(t, ts, token, "/api/targets/nope/intent/status.read", `{}`); res.StatusCode != http.StatusNotFound {
		t.Fatalf("unknown box: %d", res.StatusCode)
	}
	big := `{"pad":"` + strings.Repeat("x", maxIntentBody) + `"}`
	if res, _ := postIntent(t, ts, token, "/api/targets/box/intent/status.read", big); res.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized payload: %d", res.StatusCode)
	}
	if got := nextSeq(t, "box"); got > 1 {
		t.Fatalf("a refused request used sequence numbers (next %d)", got)
	}
}
