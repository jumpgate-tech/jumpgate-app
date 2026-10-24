package ops

import (
	"context"
	"strings"
	"testing"

	"github.com/valve-tech/jumpgate/internal/catalog"
	"github.com/valve-tech/jumpgate/internal/executor"
)

func TestFirewallListsPublicModulePorts(t *testing.T) {
	ports, err := ModulePortsFromConfig([]catalog.BoxModule{{ID: "el-bootnode", ChainID: 943}})
	if err != nil {
		t.Fatal(err)
	}
	e := newFakeExecutor().
		script("ss -lun", executor.Result{Stdout: sslHeader() + ssLine("0.0.0.0", 30301)}).
		script("ss -ltn", executor.Result{Stdout: sslHeader()})
	items, err := ModuleFirewallItems(context.Background(), e, ports)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("items %+v", items)
	}
	it := items[0]
	// Only the bind is checked: the item says so, and asks for the
	// firewall/security group to be opened.
	if it.Title != "UDP 30301 listening publicly (EL bootnode)" || it.Status != "pass" || it.ID != "module-el-bootnode-943-udp-30301" {
		t.Fatalf("item %+v", it)
	}
	if !strings.Contains(it.Detail, "security group") {
		t.Fatalf("detail %q", it.Detail)
	}

	ports, _ = ModulePortsFromConfig([]catalog.BoxModule{{ID: "el-bootnode", ChainID: 943, Port: 30400}})
	e = newFakeExecutor().script("ss -lun", executor.Result{Stdout: sslHeader()}).script("ss -ltn", executor.Result{Stdout: sslHeader()})
	items, err = ModuleFirewallItems(context.Background(), e, ports)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Status == "pass" || !strings.Contains(items[0].Fix, "ufw allow 30400/udp") {
		t.Fatalf("items %+v", items)
	}

	// No modules: no probe at all.
	e = newFakeExecutor()
	if items, err := ModuleFirewallItems(context.Background(), e, nil); err != nil || len(items) != 0 || len(e.calls) != 0 {
		t.Fatalf("items %+v err %v calls %v", items, err, e.calls)
	}
}

// On the box (the gate's path), the ports come from the modules' sidecars;
// a sidecar for a path it does not belong to, or a private port, is skipped.
func TestModulePortsOnBoxReadsSidecars(t *testing.T) {
	side := `{"id":"el-bootnode","chainId":943,"unit":"jumpgate-mod-el-bootnode-943.service","port":30301,"proto":"udp","public":true}`
	e := newFakeExecutor().
		script("ls /var/lib/jumpgate/modules/*/module.json", executor.Result{Stdout: "/var/lib/jumpgate/modules/el-bootnode-943/module.json\n/var/lib/jumpgate/modules/x/module.json\n"}).
		script("cat '/var/lib/jumpgate/modules/el-bootnode-943/module.json'", executor.Result{Stdout: side}).
		script("cat '/var/lib/jumpgate/modules/x/module.json'", executor.Result{Stdout: side})
	ports := ModulePortsOnBox(context.Background(), e)
	if len(ports) != 1 || ports[0] != (ModulePort{Instance: "el-bootnode-943", Name: "EL bootnode", Port: 30301, Proto: "udp"}) {
		t.Fatalf("ports %+v", ports)
	}
}
