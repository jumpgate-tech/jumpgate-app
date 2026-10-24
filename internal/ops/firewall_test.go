package ops

import (
	"context"
	"strings"
	"testing"

	"github.com/valve-tech/jumpgate/internal/catalog"
	"github.com/valve-tech/jumpgate/internal/executor"
)

func TestFirewallListsPublicModulePorts(t *testing.T) {
	e := newFakeExecutor().
		script("ss -lun", executor.Result{Stdout: sslHeader() + ssLine("0.0.0.0", 30301)}).
		script("ss -ltn", executor.Result{Stdout: sslHeader()})
	items, err := ModuleFirewallItems(context.Background(), e, []catalog.BoxModule{{ID: "el-bootnode", ChainID: 943}})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("items %+v", items)
	}
	it := items[0]
	if it.Title != "UDP 30301 open to the internet (EL bootnode)" || it.Status != "pass" {
		t.Fatalf("item %+v", it)
	}

	e = newFakeExecutor().script("ss -lun", executor.Result{Stdout: sslHeader()}).script("ss -ltn", executor.Result{Stdout: sslHeader()})
	items, err = ModuleFirewallItems(context.Background(), e, []catalog.BoxModule{{ID: "el-bootnode", ChainID: 943, Port: 30400}})
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
