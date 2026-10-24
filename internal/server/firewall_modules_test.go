package server

import (
	"io"
	"net/http"
	"testing"

	"github.com/valve-tech/jumpgate/internal/catalog"
	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/executor"
	"github.com/valve-tech/jumpgate/internal/ops"
)

// A box with only modules (no wire) gets a checklist of its module ports
// over the shell, not "not set up" and not a nil dereference.
func TestFirewallModuleOnlyTarget(t *testing.T) {
	a := newAPITestServerWithExecutor(t, func(config.Target) (executor.Executor, error) {
		e := &scriptedExecutor{}
		e.script("ss -lun", executor.Result{Stdout: "State Recv-Q Send-Q Local Address:Port Peer Address:Port\nUNCONN 0 0 0.0.0.0:30301 0.0.0.0:*\n"})
		return e, nil
	})
	moduleTarget(t, a, func(c *config.Config) {
		targetByID(c, "box").Modules = []catalog.BoxModule{{ID: "el-bootnode", ChainID: 943}}
	})
	res := a.do(t, "GET", "/api/targets/box/firewall", nil)
	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		t.Fatalf("status %d: %s", res.StatusCode, body)
	}
	items := decodeJSON[[]ops.CheckItem](t, res)
	if len(items) != 1 || items[0].ID != "module-el-bootnode-943-udp-30301" || items[0].Status != "pass" {
		t.Fatalf("items %+v", items)
	}
}
