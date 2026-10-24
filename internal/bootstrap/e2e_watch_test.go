//go:build e2e

package bootstrap

import (
	"context"
	"encoding/json"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/valve-tech/jumpgate/internal/agentclient"
	"github.com/valve-tech/jumpgate/internal/catalog"
	"github.com/valve-tech/jumpgate/internal/executor"
	"github.com/valve-tech/jumpgate/internal/intent"
	"github.com/valve-tech/jumpgate/internal/monitor"
	"github.com/valve-tech/jumpgate/internal/setup"
	"github.com/valve-tech/jumpgate/internal/signer"
)

// TestE2EWatch adds the health watch to a paired box with one endpoint
// that never answers, and waits for modules.read to report its alert. The
// watch polls every 2s here (JUMPGATE_WATCH_INTERVAL in a unit drop-in).
//
// The webhook is not asserted here: the watch trusts only the system
// roots, so a host httptest certificate is refused (TestDeliver* in
// internal/watch cover delivery). JUMPGATE_E2E_WEBHOOK_URL, when set to an
// https endpoint the box trusts, is configured so it can be checked by hand.
func TestE2EWatch(t *testing.T) {
	port := e2ePort(t)
	root := executor.SSHConfig{Host: "127.0.0.1", Port: port, User: "root", KeyPath: e2eEnv(t, "JUMPGATE_E2E_ROOT_KEY"), HostKey: ssh.InsecureIgnoreHostKey()}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	ex, err := executor.NewSSHContext(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	defer ex.Close()

	controller, _ := signer.GenerateKey()
	agentAddr, err := Run(ctx, Options{
		Exec: ex, Controller: controller.Address(), ControllerLabel: "e2e",
		TransportKey: e2eEnv(t, "JUMPGATE_E2E_TRANSPORT_PUB"),
		AgentBinary:  e2eAgents(t),
	})
	if err != nil {
		t.Fatal(err)
	}
	waitAgentSocket(ctx, t, ex)

	b := catalog.BoxModule{ID: "watch", Upstreams: []string{"dead=http://127.0.0.1:1"}, Webhook: os.Getenv("JUMPGATE_E2E_WEBHOOK_URL")}
	steps, err := setup.PlanBoxModule(b, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	steps = slices.DeleteFunc(steps, func(s setup.Step) bool { return s.ID == "preflight" || s.ID == "toolchain" })
	if err := setup.RunAll(ctx, ex, steps, &setup.State{}); err != nil {
		t.Fatalf("watch plan: %v", err)
	}
	dropIn := "/etc/systemd/system/" + b.UnitName() + ".d"
	cmd := "mkdir -p '" + dropIn + "' && printf '[Service]\\nEnvironment=JUMPGATE_WATCH_INTERVAL=2s\\n' > '" + dropIn + "/interval.conf' && systemctl daemon-reload && systemctl restart '" + b.UnitName() + "'"
	if r, err := ex.Run(ctx, cmd, nil); err != nil || r.ExitCode != 0 {
		t.Fatalf("interval drop-in: %+v %v", r, err)
	}

	tunnel := agentclient.Target{Agent: agentAddr, SSH: executor.SSHConfig{Host: "127.0.0.1", Port: port, User: "jumpgate",
		KeyPath: e2eEnv(t, "JUMPGATE_E2E_TRANSPORT_KEY"), HostKey: ssh.InsecureIgnoreHostKey()}}
	c, err := agentclient.Dial(ctx, tunnel, controller, agentclient.NewMemorySeqStore())
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	deadline := time.Now().Add(90 * time.Second)
	var last []monitor.ModuleStatus
	for time.Now().Before(deadline) {
		res, err := c.Do(ctx, intent.KindModulesRead, struct{}{})
		if err == nil && res.Status == intent.StatusOK && json.Unmarshal(res.Result, &last) == nil {
			for _, m := range last {
				for _, a := range m.Alerts {
					if m.ID == "watch" && a.Key == "endpoint:dead" {
						if strings.Contains(a.Message, "127.0.0.1:1") && strings.HasPrefix(a.Message, "dead (") {
							return
						}
						t.Fatalf("alert message %q", a.Message)
					}
				}
			}
		}
		time.Sleep(2 * time.Second)
	}
	t.Fatalf("no endpoint:dead alert within 90s; modules.read = %+v", last)
}
