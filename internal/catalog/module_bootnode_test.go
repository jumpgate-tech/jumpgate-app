package catalog

import (
	"os"
	"reflect"
	"strings"
	"testing"
)

func execStart(t *testing.T, unit string) string {
	t.Helper()
	for _, line := range strings.Split(unit, "\n") {
		if s, ok := strings.CutPrefix(line, "ExecStart="); ok {
			return s
		}
	}
	t.Fatalf("no ExecStart in unit:\n%s", unit)
	return ""
}

func TestELBootnodeUnit(t *testing.T) {
	u, err := RenderBoxModule(BoxModule{ID: "el-bootnode", ChainID: 369, ExtIP: "203.0.113.7"})
	if err != nil {
		t.Fatal(err)
	}
	want := "reth p2p bootnode --addr 0.0.0.0:30301 --p2p-secret-key /var/lib/jumpgate-node/modules/el-bootnode-369/bootnode.key --v5 --nat extip:203.0.113.7"
	if got := execStart(t, u.Unit); got != want {
		t.Errorf("ExecStart =\n %s\nwant\n %s", got, want)
	}
	if !reflect.DeepEqual(u.Ports, []PortUse{{30301, "udp", true}}) {
		t.Errorf("Ports = %+v", u.Ports)
	}
	if u.Client != "reth" {
		t.Errorf("Client = %q", u.Client)
	}
	// The node key the service writes itself must come out 0600.
	if !strings.Contains(u.Unit, "\nUMask=0077\n") {
		t.Errorf("unit lacks UMask=0077:\n%s", u.Unit)
	}
	for f := range u.Files {
		if strings.Contains(u.Files[f], "bootnode.key") {
			t.Errorf("%s names the node key", f)
		}
	}
}

func TestELBootnodeOptionsAndValidation(t *testing.T) {
	u, err := RenderBoxModule(BoxModule{ID: "el-bootnode", ChainID: 1, Port: 30400})
	if err != nil {
		t.Fatal(err)
	}
	got := execStart(t, u.Unit)
	if !strings.Contains(got, "--addr 0.0.0.0:30400 ") || strings.Contains(got, "--nat") {
		t.Errorf("ExecStart = %s", got)
	}
	for _, b := range []BoxModule{
		{ID: "el-bootnode", ChainID: 1, ExtIP: "1.2.3.4; touch /tmp/x"},
		{ID: "el-bootnode", ChainID: 1, ExtIP: "example.com"},
		{ID: "el-bootnode", ChainID: 1, Port: 70000},
		{ID: "el-bootnode", ChainID: 1, Port: -1},
		{ID: "el-bootnode", ChainID: 1, BindAddr: "0.0.0.0 --x"},
	} {
		if _, err := RenderBoxModule(b); err == nil {
			t.Errorf("%+v accepted", b)
		}
	}
}

func TestCLBootnodeBinaryPerChain(t *testing.T) {
	for chain, want := range map[int]string{
		1:      "lighthouse boot_node --network mainnet --port 9200 --enr-address 203.0.113.7 --datadir /var/lib/jumpgate-node/modules/cl-bootnode-1",
		560048: "lighthouse boot_node --network hoodi ",
		369:    "lighthouse-pulse boot_node --network pulsechain ",
		943:    "lighthouse-pulse boot_node --network pulsechain_testnet_v4 ",
	} {
		u, err := RenderBoxModule(BoxModule{ID: "cl-bootnode", ChainID: chain, ExtIP: "203.0.113.7"})
		if err != nil {
			t.Fatalf("%d: %v", chain, err)
		}
		got := execStart(t, u.Unit)
		if !strings.HasPrefix(got, want) {
			t.Errorf("%d: ExecStart = %s, want prefix %s", chain, got, want)
		}
		bin := strings.Fields(want)[0]
		if u.Client != bin {
			t.Errorf("%d: Client = %q, want %q", chain, u.Client, bin)
		}
		if !reflect.DeepEqual(u.Ports, []PortUse{{9200, "udp", true}}) {
			t.Errorf("%d: Ports = %+v", chain, u.Ports)
		}
	}
}

func TestCLBootnodeNeedsExtIP(t *testing.T) {
	_, err := RenderBoxModule(BoxModule{ID: "cl-bootnode", ChainID: 1})
	want := "catalog: the CL bootnode needs the box's public IP (ExtIP) for its ENR"
	if err == nil || err.Error() != want {
		t.Fatalf("err = %v, want %q", err, want)
	}
	if _, err := RenderBoxModule(BoxModule{ID: "cl-bootnode", ChainID: 1, ExtIP: "$(id)"}); err == nil {
		t.Fatal("a non-IP ExtIP was accepted")
	}
}

// Every flag the units pass must be one the real binary's --help lists
// (testdata/bootnode-help, recorded from the real binaries).
func TestBootnodeFlagsMatchHelp(t *testing.T) {
	cases := []BoxModule{
		{ID: "el-bootnode", ChainID: 1, ExtIP: "203.0.113.7"},
		{ID: "cl-bootnode", ChainID: 1, ExtIP: "203.0.113.7"},
		{ID: "cl-bootnode", ChainID: 560048, ExtIP: "203.0.113.7"},
		{ID: "cl-bootnode", ChainID: 369, ExtIP: "203.0.113.7"},
		{ID: "cl-bootnode", ChainID: 943, ExtIP: "203.0.113.7"},
	}
	for _, b := range cases {
		u, err := RenderBoxModule(b)
		if err != nil {
			t.Fatal(err)
		}
		help, err := os.ReadFile("testdata/bootnode-help/" + u.Client + ".txt")
		if err != nil {
			t.Fatal(err)
		}
		args := strings.Fields(execStart(t, u.Unit))
		for i, a := range args {
			if strings.HasPrefix(a, "--") && !strings.Contains(string(help), a+" ") && !strings.Contains(string(help), a+"\n") {
				t.Errorf("%s: %s is not in %s's help", b.Instance(), a, u.Client)
			}
			if a == "--network" && !strings.Contains(string(help), args[i+1]) {
				t.Errorf("%s: network %s is not in %s's help", b.Instance(), args[i+1], u.Client)
			}
		}
	}
}

func TestBootnodesApplyToAllFourChains(t *testing.T) {
	for _, id := range []string{"el-bootnode", "cl-bootnode"} {
		m, ok := ModuleByID(id)
		if !ok || m.Placement != PlaceBox {
			t.Fatalf("%s not registered as a box module", id)
		}
		for _, c := range []int{1, 369, 943, 560048} {
			if ok, r := m.Applies(c, ""); !ok {
				t.Errorf("%s on %d: %q", id, c, r)
			}
		}
	}
}
