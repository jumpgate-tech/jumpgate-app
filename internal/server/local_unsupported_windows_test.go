package server

import (
	"encoding/json"
	"net/http"
	"testing"
)

// On real Windows nothing is faked: the default executor factory builds
// "this host" (it runs Docker through RunArgv), and the VPN route, which
// needs a shell, refuses it through RequireShell as local_unsupported.
func TestVPNUpOnWindowsIsLocalUnsupported(t *testing.T) {
	a := newAPITestServerCfg(t, nil, nil)
	createTestVPN(t, a, "pvpn")
	res := a.do(t, "POST", "/api/vpns/pvpn/up", nil)
	defer res.Body.Close()
	var e struct{ Code, Hint string }
	_ = json.NewDecoder(res.Body).Decode(&e)
	if res.StatusCode != http.StatusConflict || e.Code != "local_unsupported" || e.Hint == "" {
		t.Fatalf("%d %+v, want 409 local_unsupported with a hint", res.StatusCode, e)
	}
}
