package catalog

import (
	"reflect"
	"strings"
	"testing"
)

// withModules swaps the registry for the length of one test.
func withModules(t *testing.T, ms []Module) {
	t.Helper()
	old := modules
	modules = ms
	t.Cleanup(func() { modules = old })
}

func fakeMsgboard() Module {
	return Module{
		ID: "msgboard", Name: "Message board", Placement: PlaceNode,
		Chains: []int{369, 943}, Clients: []string{"reth"},
		Namespaces: []string{"msgboard"},
		Methods: []ModuleMethod{
			{Name: "msgboard_addMessage", Class: MethodWrite},
			{Name: "msgboard_status", Class: MethodRead},
			{Name: "msgboard_subscribe", Class: MethodStream},
		},
		ClientReason: func(client string) string {
			return "The message board needs reth (valve fork). This node runs " + client + "."
		},
	}
}

func fakeBox(id string) Module {
	return Module{ID: id, Placement: PlaceBox, Unit: func(BoxModule) (BoxUnit, error) { return BoxUnit{}, nil }}
}

func wantErr(t *testing.T, ms []Module, sub string) {
	t.Helper()
	err := checkModules(ms)
	if err == nil || !strings.Contains(err.Error(), sub) {
		t.Fatalf("checkModules err = %v, want containing %q", err, sub)
	}
}

func TestCheckModulesRefusesAllAndNone(t *testing.T) {
	for _, ns := range []string{"all", "none"} {
		m := fakeMsgboard()
		m.ID = "x"
		m.Namespaces = []string{ns}
		wantErr(t, []Module{m}, `module "x": namespace "`+ns+`" is refused`)
	}
}

func TestCheckModulesRefusesWrongPlacementFields(t *testing.T) {
	box := fakeBox("watch")
	box.Namespaces = []string{"txpool"}
	wantErr(t, []Module{box}, `module "watch"`)

	node := fakeMsgboard()
	node.Unit = func(BoxModule) (BoxUnit, error) { return BoxUnit{}, nil }
	wantErr(t, []Module{node}, `module "msgboard"`)

	gw := Module{ID: "private-send", Placement: PlaceGateway, Chains: []int{1}}
	wantErr(t, []Module{gw}, `module "private-send"`)

	wantErr(t, []Module{{ID: "watch", Placement: PlaceBox}}, `module "watch"`)

	wantErr(t, []Module{{ID: "watch", Placement: "elsewhere"}}, `placement`)
}

func TestCheckModulesRefusesDuplicateID(t *testing.T) {
	wantErr(t, []Module{fakeBox("watch"), fakeBox("watch")}, `duplicate`)
}

func TestCheckModulesRefusesUnknownID(t *testing.T) {
	wantErr(t, []Module{fakeBox("nope")}, `moduleOrder`)
}

func TestCheckModulesRefusesBetaOutsideChains(t *testing.T) {
	m := fakeMsgboard()
	m.BetaChains = []int{1}
	wantErr(t, []Module{m}, `beta chain 1`)
}

func TestCheckModulesEmptyChainsBoxOnly(t *testing.T) {
	m := fakeMsgboard()
	m.Chains = nil
	wantErr(t, []Module{m}, `module "msgboard"`)

	box := fakeBox("watch")
	if err := checkModules([]Module{box}); err != nil {
		t.Fatalf("box with no chains: %v", err)
	}
	if ok, reason := box.Applies(12345, ""); !ok || reason != "" {
		t.Fatalf("Applies = %v, %q", ok, reason)
	}
}

func TestCheckModulesAcceptsGood(t *testing.T) {
	if err := checkModules([]Module{fakeMsgboard(), fakeBox("watch")}); err != nil {
		t.Fatal(err)
	}
}

func TestAppliesHiddenOnOtherChain(t *testing.T) {
	m := fakeMsgboard()
	m.Chains = []int{369}
	if ok, reason := m.Applies(1, "reth"); ok || reason != "" {
		t.Fatalf("Applies = %v, %q", ok, reason)
	}
	if ok, _ := m.Applies(369, "reth"); !ok {
		t.Fatal("want ok on 369/reth")
	}
}

func TestAppliesReasonForOtherClient(t *testing.T) {
	m := fakeMsgboard()
	ok, reason := m.Applies(369, "go-pulse")
	want := "The message board needs reth (valve fork). This node runs go-pulse."
	if ok || reason != want {
		t.Fatalf("Applies = %v, %q", ok, reason)
	}
}

func TestAppliesUnsupportedReason(t *testing.T) {
	m := Module{ID: "private-send", Placement: PlaceGateway, Chains: []int{1},
		Unsupported: map[int]string{369: "No private send route exists on PulseChain."}}
	if ok, reason := m.Applies(369, ""); ok || reason != "No private send route exists on PulseChain." {
		t.Fatalf("Applies = %v, %q", ok, reason)
	}
}

func TestIsBeta(t *testing.T) {
	m := fakeMsgboard()
	m.BetaChains = []int{943}
	if !m.IsBeta(943) || m.IsBeta(369) {
		t.Fatal("IsBeta wrong")
	}
}

func moduleIDs(ms []Module) (out []string) {
	for _, m := range ms {
		out = append(out, m.ID)
	}
	return out
}

func TestModulesOrderIsStable(t *testing.T) {
	ms := []Module{fakeBox("watch"), fakeMsgboard(), fakeBox("el-bootnode")}
	sortModules(ms)
	withModules(t, ms)
	a, b := Modules(), Modules()
	want := []string{"msgboard", "el-bootnode", "watch"}
	if !reflect.DeepEqual(moduleIDs(a), want) || !reflect.DeepEqual(moduleIDs(b), want) {
		t.Fatalf("order %v / %v", moduleIDs(a), moduleIDs(b))
	}
	a[0].ID = "mutated"
	if Modules()[0].ID != "msgboard" {
		t.Fatal("Modules() must return a copy")
	}
	if m, ok := ModuleByID("watch"); !ok || m.ID != "watch" {
		t.Fatal("ModuleByID")
	}
	if _, ok := ModuleByID("nope"); ok {
		t.Fatal("ModuleByID unknown")
	}
}

func TestModuleWriteMethods(t *testing.T) {
	withModules(t, []Module{fakeMsgboard(), fakeBox("watch")})
	got := ModuleWriteMethods()
	if want := map[string]string{"msgboard_addMessage": "msgboard"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v", got)
	}
}
