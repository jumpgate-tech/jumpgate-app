package server

import (
	"net/http"
	"strings"
	"testing"

	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/executor"
)

// A box module on a box whose node still runs as the legacy account is
// refused before it is recorded: turning it on would make the current
// account beside the legacy one, which node.migrate-names cannot rename onto.
func TestSetBoxModuleRefusesLegacyAccountBox(t *testing.T) {
	fakeModules(t)
	ex := (&scriptedExecutor{}).script("! id -u jumpgate-node", executor.Result{Stdout: "legacy\n"})
	a := newAPITestServerWithExecutor(t, func(config.Target) (executor.Executor, error) { return ex, nil })
	moduleTarget(t, a, func(c *config.Config) {})
	res := a.do(t, "PUT", "/api/targets/box/modules/watch", map[string]any{"enabled": true})
	if e := readAPIError(t, res); res.StatusCode != http.StatusConflict || !strings.Contains(e.Message, "node.migrate-names") {
		t.Fatalf("legacy box: %d %+v", res.StatusCode, e)
	}
	if got := storedTarget(t, "box").Modules; len(got) != 0 {
		t.Fatalf("a refused module was stored: %+v", got)
	}
	if a.srv.reg.get("box").setup != nil {
		t.Fatal("a refused module started a setup run")
	}
}
