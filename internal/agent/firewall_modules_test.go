package agent

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/valve-tech/jumpgate/internal/executor"
	"github.com/valve-tech/jumpgate/internal/intent"
	"github.com/valve-tech/jumpgate/internal/ops"
	"github.com/valve-tech/jumpgate/internal/testutil"
)

// scriptExec answers by the first matching substring.
type scriptExec struct {
	cmds   []string
	answer map[string]string
}

func (f *scriptExec) Run(_ context.Context, cmd string, _ *executor.RunOpts) (executor.Result, error) {
	f.cmds = append(f.cmds, cmd)
	for k, v := range f.answer {
		if strings.Contains(cmd, k) {
			return executor.Result{Stdout: v}, nil
		}
	}
	return executor.Result{}, nil
}
func (f *scriptExec) WriteFile(context.Context, string, []byte, os.FileMode) error { return nil }
func (f *scriptExec) ReadFile(context.Context, string) ([]byte, error)             { return nil, nil }
func (f *scriptExec) Close() error                                                 { return nil }

func bootnodeBox() *scriptExec {
	return &scriptExec{answer: map[string]string{
		"ls /var/lib/jumpgate/modules/": "/var/lib/jumpgate/modules/el-bootnode-943/module.json\n",
		"cat ":                          `{"id":"el-bootnode","chainId":943,"unit":"jumpgate-mod-el-bootnode-943.service","port":30301,"proto":"udp","public":true}`,
		"ss -lun":                       "State Recv-Q Send-Q Local Address:Port Peer Address:Port\nUNCONN 0 0 0.0.0.0:30301 0.0.0.0:*\n",
	}}
}

func hasItem(items []ops.CheckItem, id string) bool {
	for _, it := range items {
		if it.ID == id {
			return true
		}
	}
	return false
}

// A box with modules and no node still answers firewall.read, with the
// modules' port items.
func TestFirewallReadModuleOnlyBox(t *testing.T) {
	testutil.RequireUnix(t)
	r := newRig(t, false)
	r.a.cfg.Exec = bootnodeBox()
	rc, res := r.send(t, intent.KindFirewallRead, FirewallReadPayload{}, nil)
	if rc.Status != intent.StatusOK {
		t.Fatalf("status %d: %s", rc.Status, res)
	}
	var items []ops.CheckItem
	if err := json.Unmarshal(res, &items); err != nil || len(items) != 1 || !hasItem(items, "module-el-bootnode-943-udp-30301") {
		t.Fatalf("items %s %v", res, err)
	}
}

// A box with a node gets the node checklist plus the module items.
func TestFirewallReadAddsModuleItems(t *testing.T) {
	testutil.RequireUnix(t)
	r := newRig(t, true)
	r.a.cfg.Exec = bootnodeBox()
	rc, res := r.send(t, intent.KindFirewallRead, FirewallReadPayload{}, nil)
	if rc.Status != intent.StatusOK {
		t.Fatalf("status %d: %s", rc.Status, res)
	}
	var items []ops.CheckItem
	if err := json.Unmarshal(res, &items); err != nil || !hasItem(items, "exec-p2p-open") || !hasItem(items, "module-el-bootnode-943-udp-30301") {
		t.Fatalf("items %s %v", res, err)
	}
}

// No node and no modules is still not set up.
func TestFirewallReadBareBoxIsNotSetUp(t *testing.T) {
	testutil.RequireUnix(t)
	r := newRig(t, false)
	rc, res := r.send(t, intent.KindFirewallRead, FirewallReadPayload{}, nil)
	if got := rejection(t, rc, res).Code; got != intent.ReasonNotSetUp {
		t.Fatalf("code %s", got)
	}
}
