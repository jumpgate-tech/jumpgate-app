//go:build e2e

package bootstrap

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/valve-tech/jumpgate/internal/catalog"
	"github.com/valve-tech/jumpgate/internal/executor"
	"github.com/valve-tech/jumpgate/internal/monitor"
	"github.com/valve-tech/jumpgate/internal/setup"
)

// TestE2ECheckpointzUnit turns Checkpointz on for Hoodi on a box with no
// node (the container's stub checkpointz serves a fixed finalized status
// on 5555), probes it, then turns it off.
func TestE2ECheckpointzUnit(t *testing.T) {
	port := e2ePort(t)
	root := executor.SSHConfig{Host: "127.0.0.1", Port: port, User: "root", KeyPath: e2eEnv(t, "JUMPGATE_E2E_ROOT_KEY"), HostKey: ssh.InsecureIgnoreHostKey()}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	ex, err := executor.NewSSHContext(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	defer ex.Close()

	// The tests share one container. Enabling the module creates the
	// jumpgate-node account (and its home), which the migrate-names e2e
	// reads as "both accounts exist", so take back what this test made.
	if r, _ := ex.Run(ctx, "id -u jumpgate-node", nil); r.ExitCode != 0 {
		homeExisted := false
		if r, _ := ex.Run(ctx, "test -e /var/lib/jumpgate-node", nil); r.ExitCode == 0 {
			homeExisted = true
		}
		defer func() {
			ex.Run(context.Background(), "userdel jumpgate-node; getent group jumpgate-node >/dev/null && groupdel jumpgate-node; true", nil)
			if !homeExisted {
				ex.Run(context.Background(), "rm -rf /var/lib/jumpgate-node", nil)
			}
		}()
	}

	b := catalog.BoxModule{ID: "checkpointz", ChainID: 560048, Upstreams: []string{"http://127.0.0.1:5052"}}
	run := func(enable bool) {
		t.Helper()
		steps, err := setup.PlanBoxModule(b, nil, enable)
		if err != nil {
			t.Fatal(err)
		}
		// The stub passes install's --help check; preflight's ss and the
		// toolchain need a real box.
		steps = slices.DeleteFunc(steps, func(s setup.Step) bool { return s.ID == "preflight" || s.ID == "toolchain" })
		if err := setup.RunAll(ctx, ex, steps, &setup.State{}); err != nil {
			t.Fatalf("checkpointz plan (enable=%v): %v", enable, err)
		}
	}
	run(true)

	unit := b.UnitName()
	if r, err := ex.Run(ctx, "systemctl is-active "+unit, nil); err != nil || strings.TrimSpace(r.Stdout) != "active" {
		t.Fatalf("unit not active: %+v %v", r, err)
	}
	// The config holds the upstream; module.json does not.
	if r, _ := ex.Run(ctx, "stat -c %a "+b.ConfigDir()+"/config.yaml", nil); strings.TrimSpace(r.Stdout) != "640" {
		t.Fatalf("config.yaml mode = %q", r.Stdout)
	}
	if r, _ := ex.Run(ctx, "cat "+b.SidecarPath(), nil); strings.Contains(r.Stdout, "5052") {
		t.Fatalf("module.json names an upstream: %s", r.Stdout)
	}

	var st monitor.ModuleStatus
	deadline := time.Now().Add(30 * time.Second)
	for {
		st = monitor.ProbeBoxModule(ctx, ex, b)
		if st.Served || time.Now().After(deadline) {
			break
		}
		time.Sleep(time.Second)
	}
	if !st.Served || st.HeadBlock != 77 || !st.Active {
		t.Fatalf("probe = %+v", st)
	}

	run(false)
	if r, _ := ex.Run(ctx, "test -e /etc/systemd/system/"+unit, nil); r.ExitCode == 0 {
		t.Fatal("the unit is still on the box")
	}
}
