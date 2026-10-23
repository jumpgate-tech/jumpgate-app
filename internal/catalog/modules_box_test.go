package catalog

import (
	"encoding/json"
	"strings"
	"testing"
)

// fakePortBox is a box module whose unit uses one port of proto: def, unless
// the record sets Port.
func fakePortBox(id, proto string, def int) Module {
	m := fakeBox(id)
	m.Unit = func(b BoxModule) (BoxUnit, error) {
		p := b.Port
		if p == 0 {
			p = def
		}
		u, err := renderServiceUnit("module "+b.Instance(), "/usr/local/bin/fake", CurrentNames, b.DataDir(), false)
		return BoxUnit{Unit: u, Ports: []PortUse{{Port: p, Proto: proto, Public: true}}}, err
	}
	return m
}

func TestRenderUnitRefactorIsByteIdentical(t *testing.T) {
	TestUnitsByteIdenticalWithoutModules(t)
}

func TestBoxModuleNames(t *testing.T) {
	b := BoxModule{ID: "el-bootnode", ChainID: 369}
	if got := b.Instance(); got != "el-bootnode-369" {
		t.Errorf("Instance = %q", got)
	}
	if got := b.UnitName(); got != "jumpgate-mod-el-bootnode-369.service" {
		t.Errorf("UnitName = %q", got)
	}
	if got := b.DataDir(); got != "/var/lib/jumpgate-node/modules/el-bootnode-369" {
		t.Errorf("DataDir = %q", got)
	}
	w := BoxModule{ID: "watch"}
	if w.Instance() != "watch" || w.UnitName() != "jumpgate-mod-watch.service" {
		t.Errorf("watch names = %q %q", w.Instance(), w.UnitName())
	}
}

func TestRenderBoxModuleWritesSidecar(t *testing.T) {
	withModules(t, []Module{fakePortBox("watch", "tcp", 0)})
	b := BoxModule{ID: "watch", Upstreams: []string{"http://127.0.0.1:8545"}}
	u, err := RenderBoxModule(b)
	if err != nil {
		t.Fatal(err)
	}
	want, _ := json.Marshal(b)
	if got := u.Files[b.DataDir()+"/module.json"]; got != string(want) {
		t.Errorf("module.json = %q, want %q", got, want)
	}
	for _, s := range []string{"User=jumpgate-node\n", "ReadWritePaths=" + b.DataDir() + "\n"} {
		if !strings.Contains(u.Unit, s) {
			t.Errorf("unit lacks %q:\n%s", s, u.Unit)
		}
	}
}

func TestRenderBoxModuleRefusesUnknownAndNotApplicable(t *testing.T) {
	m := fakePortBox("checkpointz", "tcp", 5555)
	m.Chains = []int{1}
	withModules(t, []Module{m})
	if _, err := RenderBoxModule(BoxModule{ID: "nope"}); err == nil {
		t.Error("unknown module rendered")
	}
	if _, err := RenderBoxModule(BoxModule{ID: "checkpointz", ChainID: 369}); err == nil {
		t.Error("checkpointz on 369 rendered")
	}
}

func TestValidateBoxModulesRefusesPortClash(t *testing.T) {
	cp := fakePortBox("checkpointz", "tcp", 5555)
	withModules(t, []Module{fakePortBox("cl-bootnode", "udp", 9200), cp})
	w := &WireConfig{ChainID: 369, ExecID: "reth", BeaconID: "lighthouse-pulse"}
	err := ValidateBoxModules([]BoxModule{{ID: "cl-bootnode", ChainID: 369, Port: 9000}}, w)
	if err == nil || err.Error() != "cl-bootnode-369 wants udp 9000, which the beacon client already uses" {
		t.Errorf("beacon clash: %v", err)
	}
	if err := ValidateBoxModules([]BoxModule{{ID: "cl-bootnode", ChainID: 369}}, w); err != nil {
		t.Errorf("default 9200 refused: %v", err)
	}
	err = ValidateBoxModules([]BoxModule{{ID: "checkpointz", ChainID: 1}, {ID: "checkpointz", ChainID: 560048}}, nil)
	if err == nil || err.Error() != "checkpointz-1 and checkpointz-560048 both want tcp 5555" {
		t.Errorf("module clash: %v", err)
	}
}

func TestValidateBoxModulesRefusesDuplicateInstance(t *testing.T) {
	withModules(t, []Module{fakePortBox("el-bootnode", "udp", 30301)})
	err := ValidateBoxModules([]BoxModule{{ID: "el-bootnode", ChainID: 369}, {ID: "el-bootnode", ChainID: 369, Port: 30302}}, nil)
	if err == nil || err.Error() != "el-bootnode-369 is on this box twice" {
		t.Errorf("duplicate: %v", err)
	}
}

func TestValidateBoxModulesRefusesNodeModule(t *testing.T) {
	withModules(t, []Module{fakeMsgboard()})
	err := ValidateBoxModules([]BoxModule{{ID: "msgboard"}}, nil)
	if err == nil || err.Error() != "msgboard is not a box module" {
		t.Errorf("node module: %v", err)
	}
}
