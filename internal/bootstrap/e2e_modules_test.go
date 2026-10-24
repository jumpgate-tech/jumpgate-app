//go:build e2e

package bootstrap

import (
	"context"
	"encoding/json"
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

// TestE2EBoxModule adds the EL bootnode on 943 to a paired box with no node
// (the container's stub reth prints an enode and sleeps), reads it back
// through the gate's modules.read, then turns it off.
func TestE2EBoxModule(t *testing.T) {
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

	b := catalog.BoxModule{ID: "el-bootnode", ChainID: 943}
	// The stub reth passes install's --version check; preflight's ss and
	// the toolchain need a real box.
	run := func(enable bool) {
		t.Helper()
		steps, err := setup.PlanBoxModule(b, nil, enable)
		if err != nil {
			t.Fatal(err)
		}
		steps = slices.DeleteFunc(steps, func(s setup.Step) bool { return s.ID == "preflight" || s.ID == "toolchain" })
		if err := setup.RunAll(ctx, ex, steps, &setup.State{}); err != nil {
			t.Fatalf("box module plan (enable=%v): %v", enable, err)
		}
	}
	run(true)

	if r, err := ex.Run(ctx, "systemctl is-active jumpgate-mod-el-bootnode-943.service", nil); err != nil || strings.TrimSpace(r.Stdout) != "active" {
		t.Fatalf("unit not active: %+v %v", r, err)
	}

	tunnel := agentclient.Target{Agent: agentAddr, SSH: executor.SSHConfig{Host: "127.0.0.1", Port: port, User: "jumpgate",
		KeyPath: e2eEnv(t, "JUMPGATE_E2E_TRANSPORT_KEY"), HostKey: ssh.InsecureIgnoreHostKey()}}
	c, err := agentclient.Dial(ctx, tunnel, controller, agentclient.NewMemorySeqStore())
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	res, err := c.Do(ctx, intent.KindModulesRead, struct{}{})
	if err != nil || res.Status != intent.StatusOK {
		t.Fatalf("modules.read: %+v %v", res, err)
	}
	var mods []monitor.ModuleStatus
	if err := json.Unmarshal(res.Result, &mods); err != nil {
		t.Fatalf("modules.read result %s: %v", res.Result, err)
	}
	if len(mods) != 1 || mods[0].Instance != "el-bootnode-943" || !mods[0].Active || !strings.HasPrefix(mods[0].Identity, "enode://") {
		t.Fatalf("modules.read = %+v", mods)
	}

	run(false)
	if r, _ := ex.Run(ctx, "test -e /etc/systemd/system/jumpgate-mod-el-bootnode-943.service", nil); r.ExitCode == 0 {
		t.Fatal("the unit is still on the box")
	}
	if r, err := ex.Run(ctx, "test -d "+b.DataDir(), nil); err != nil || r.ExitCode != 0 {
		t.Fatalf("the data dir is gone: %+v %v", r, err)
	}
}
