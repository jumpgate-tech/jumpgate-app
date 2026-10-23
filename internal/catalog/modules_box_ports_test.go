package catalog

import "testing"

func TestValidateBoxModulesRefusesPortTwiceInOneModule(t *testing.T) {
	m := fakeBox("checkpointz")
	m.Unit = func(b BoxModule) (BoxUnit, error) {
		u, err := renderServiceUnit("module "+b.Instance(), "/usr/local/bin/fake", CurrentNames, b.DataDir(), false)
		return BoxUnit{Unit: u, Ports: []PortUse{{Port: 5555, Proto: "tcp"}, {Port: 5555, Proto: "tcp"}}}, err
	}
	withModules(t, []Module{m})
	err := ValidateBoxModules([]BoxModule{{ID: "checkpointz", ChainID: 1}}, nil)
	if err == nil || err.Error() != "checkpointz-1 wants tcp 5555 twice" {
		t.Errorf("one module, one port twice: %v", err)
	}
}

func TestValidateBoxModulesKnowsPrysmAndERPCPorts(t *testing.T) {
	withModules(t, []Module{fakePortBox("cl-bootnode", "udp", 9200), fakePortBox("checkpointz", "tcp", 5555)})
	w := &WireConfig{ChainID: 369, ExecID: "geth-pulse", BeaconID: "prysm-pulse", ERPCEnabled: true}
	for _, c := range []struct {
		b    BoxModule
		want string
	}{
		{BoxModule{ID: "checkpointz", ChainID: 369, Port: 13000}, "checkpointz-369 wants tcp 13000, which the beacon client already uses"},
		{BoxModule{ID: "cl-bootnode", ChainID: 369, Port: 12000}, "cl-bootnode-369 wants udp 12000, which the beacon client already uses"},
		{BoxModule{ID: "checkpointz", ChainID: 369, Port: 4000}, "checkpointz-369 wants tcp 4000, which the RPC gateway already uses"},
	} {
		if err := ValidateBoxModules([]BoxModule{c.b}, w); err == nil || err.Error() != c.want {
			t.Errorf("%+v: %v, want %q", c.b, err, c.want)
		}
	}
	w.ERPCEnabled = false
	if err := ValidateBoxModules([]BoxModule{{ID: "checkpointz", ChainID: 369, Port: 4000}}, w); err != nil {
		t.Errorf("4000 without eRPC refused: %v", err)
	}
}
