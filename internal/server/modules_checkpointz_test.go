package server

import (
	"io"
	"net/http"
	"reflect"
	"testing"

	"github.com/valve-tech/jumpgate/internal/config"
)

// A Checkpointz turned on with no upstream gets the box's own beacon when
// the box runs a node on its chain, and is refused when it has none.
func TestSetCheckpointzDefaultsToLocalBeacon(t *testing.T) {
	a := newAPITestServer(t)
	moduleTarget(t, a, func(c *config.Config) {
		w := rethWire(560048, "reth")
		w.BeaconID = "lighthouse"
		targetByID(c, "box").Wire = w
	})
	res := a.do(t, "PUT", "/api/targets/box/modules/checkpointz", map[string]any{"enabled": true})
	if res.StatusCode != http.StatusAccepted {
		b, _ := io.ReadAll(res.Body)
		t.Fatalf("status %d: %s", res.StatusCode, b)
	}
	res.Body.Close()
	waitSetupIdle(t, a, "box")
	got := storedTarget(t, "box").Modules
	if len(got) != 1 || got[0].ChainID != 560048 || !reflect.DeepEqual(got[0].Upstreams, []string{"http://127.0.0.1:5052"}) {
		t.Fatalf("stored %+v", got)
	}

	res = a.do(t, "PUT", "/api/targets/box/modules/checkpointz", map[string]any{"enabled": true, "chainId": 1})
	if e := readAPIError(t, res); res.StatusCode != http.StatusBadRequest ||
		e.Message != "catalog: Checkpointz needs at least one synced beacon node on chain 1" {
		t.Fatalf("no upstream: %d %+v", res.StatusCode, e)
	}

	res = a.do(t, "PUT", "/api/targets/box/modules/checkpointz", map[string]any{"enabled": true, "bindAddr": "203.0.113.7"})
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("public bind: %d", res.StatusCode)
	}
	res.Body.Close()
}
