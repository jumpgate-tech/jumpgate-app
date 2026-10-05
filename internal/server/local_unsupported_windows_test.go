package server

import (
	"encoding/json"
	"net/http"
	"testing"
)

// On real Windows nothing is faked: the default executor factory itself
// refuses "this host", and the VPN route turns that into local_unsupported.
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
