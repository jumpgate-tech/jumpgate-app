package catalog

import (
	"strings"
	"testing"
)

func TestWatchUnit(t *testing.T) {
	b := BoxModule{ID: "watch", Upstreams: []string{"gw-eu=https://gw.example/rpc"}, Webhook: "https://hooks.example/abc"}
	u, err := RenderBoxModule(b)
	if err != nil {
		t.Fatal(err)
	}
	want := "/usr/local/lib/jumpgate/jumpgate agent watch --config /var/lib/jumpgate/modules/watch/config.json"
	if got := execStart(t, u.Unit); got != want {
		t.Errorf("ExecStart = %s, want %s", got, want)
	}
	if !strings.Contains(u.Unit, "\nUser=jumpgate-node\n") {
		t.Errorf("unit lacks User=jumpgate-node:\n%s", u.Unit)
	}
	if b.UnitName() != "jumpgate-mod-watch.service" {
		t.Errorf("UnitName = %s", b.UnitName())
	}
	if u.Client != "" || len(u.Ports) != 0 {
		t.Errorf("Client %q Ports %+v", u.Client, u.Ports)
	}
	if !strings.Contains(u.Files[b.ConfigPath()], "gw-eu=https://gw.example/rpc") || !strings.Contains(u.Files[b.ConfigPath()], "hooks.example") {
		t.Errorf("config.json = %s", u.Files[b.ConfigPath()])
	}
	if strings.Contains(u.Files[b.SidecarPath()], "hooks.example") || strings.Contains(u.Files[b.SidecarPath()], "gw.example") {
		t.Errorf("module.json leaks the config: %s", u.Files[b.SidecarPath()])
	}
}

func TestWatchRefusesBadConfig(t *testing.T) {
	for _, b := range []BoxModule{
		{ID: "watch", Webhook: "http://hooks.example/abc"},
		{ID: "watch", Webhook: "ftp://hooks.example/abc"},
		{ID: "watch", Webhook: "https://"},
		{ID: "watch", Upstreams: []string{"no-equals"}},
		{ID: "watch", Upstreams: []string{"bad name=https://x"}},
		{ID: "watch", Upstreams: []string{"a=file:///etc/passwd"}},
		{ID: "watch", Upstreams: []string{"a=https://"}},
		{ID: "watch", Upstreams: []string{"a=https://x", "a=https://y"}},
	} {
		if _, err := RenderBoxModule(b); err == nil {
			t.Errorf("%+v rendered", b)
		}
	}
}

func TestParseWatchEndpoint(t *testing.T) {
	e, err := ParseWatchEndpoint("cl=beacon+https://b.example:5052")
	if err != nil || e != (WatchEndpoint{"cl", "https://b.example:5052", "beacon"}) {
		t.Errorf("beacon: %+v %v", e, err)
	}
	e, err = ParseWatchEndpoint("gw=http://10.0.0.1:8545")
	if err != nil || e != (WatchEndpoint{"gw", "http://10.0.0.1:8545", "rpc"}) {
		t.Errorf("rpc: %+v %v", e, err)
	}
}
