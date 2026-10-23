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
	if got := b.ConfigDir(); got != "/var/lib/jumpgate/modules/el-bootnode-369" {
		t.Errorf("ConfigDir = %q", got)
	}
	if got := b.DataDir(); got != "/var/lib/jumpgate-node/modules/el-bootnode-369" {
		t.Errorf("DataDir = %q", got)
	}
	if got := b.SidecarPath(); got != "/var/lib/jumpgate/modules/el-bootnode-369/module.json" {
		t.Errorf("SidecarPath = %q", got)
	}
	w := BoxModule{ID: "watch"}
	if w.Instance() != "watch" || w.UnitName() != "jumpgate-mod-watch.service" {
		t.Errorf("watch names = %q %q", w.Instance(), w.UnitName())
	}
}

// A record read back from a box is data: an unknown id, a non-box module or
// a chain jumpgate does not know is refused before any name is built from it.
func TestValidateBoxModuleRefusesHostileRecords(t *testing.T) {
	withModules(t, []Module{fakeBox("watch"), fakeBox("el-bootnode"), fakeMsgboard()})
	for _, b := range []BoxModule{
		{ID: "x; touch /tmp/pwned"},
		{ID: "msgboard", ChainID: 369},
		{ID: "el-bootnode", ChainID: 424242},
		{ID: "el-bootnode", ChainID: -1},
	} {
		if err := ValidateBoxModule(b); err == nil {
			t.Errorf("%+v accepted", b)
		}
	}
	for _, b := range []BoxModule{{ID: "watch"}, {ID: "el-bootnode", ChainID: 369}} {
		if err := ValidateBoxModule(b); err != nil {
			t.Errorf("%+v: %v", b, err)
		}
	}
}

// A unit's config files go in its root-owned ConfigDir, never the
// service-owned DataDir.
func TestRenderBoxModuleRefusesFilesOutsideConfigDir(t *testing.T) {
	m := fakeBox("watch")
	m.Unit = func(b BoxModule) (BoxUnit, error) {
		return BoxUnit{Files: map[string]string{b.DataDir() + "/conf.yaml": "x"}}, nil
	}
	withModules(t, []Module{m})
	if _, err := RenderBoxModule(BoxModule{ID: "watch"}); err == nil {
		t.Fatal("a file in DataDir was accepted")
	}
}

func TestRenderBoxModuleWritesSidecar(t *testing.T) {
	withModules(t, []Module{fakePortBox("watch", "tcp", 7000)})
	b := BoxModule{ID: "watch", Upstreams: []string{"http://127.0.0.1:8545"}, Webhook: "https://hooks.example/s3cret"}
	u, err := RenderBoxModule(b)
	if err != nil {
		t.Fatal(err)
	}
	// The world-readable sidecar is for discovery only.
	side := u.Files[b.SidecarPath()]
	want := `{"id":"watch","unit":"jumpgate-mod-watch.service","port":7000,"proto":"tcp"}`
	if side != want {
		t.Errorf("module.json = %q, want %q", side, want)
	}
	for _, s := range []string{"webhook", "upstream", "http", "s3cret", "127.0.0.1", "secret"} {
		if strings.Contains(strings.ToLower(side), s) {
			t.Errorf("module.json holds %q: %s", s, side)
		}
	}
	// The full record goes to the unit's own config, readable by its group.
	full, _ := json.Marshal(b)
	if got := u.Files[b.ConfigPath()]; got != string(full) {
		t.Errorf("config.json = %q, want %q", got, full)
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
