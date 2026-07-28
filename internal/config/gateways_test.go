package config

import (
	"testing"

	"github.com/valve-tech/valve-node-app/internal/catalog"
)

func fleet() Config {
	return Config{Gateways: []Gateway{
		{ID: "default", Placement: GatewayPlacement{TargetID: "local", Backend: "docker"}},
		{ID: "edge", Placement: GatewayPlacement{TargetID: "boxa", Backend: "docker"}},
		{ID: "spare", Placement: GatewayPlacement{TargetID: "local", Backend: "systemd"}},
	}}
}

// A gateway names its host; the host does not own it. GatewaysOn is the read
// that turns that around for a machine's own screen, and the one thing it must
// never do is show a machine a gateway that runs somewhere else — every action
// on that card would then be aimed at the wrong box.
func TestGatewaysOn_ReturnsOnlyTheGatewaysPlacedOnThatMachine(t *testing.T) {
	got := fleet().GatewaysOn("local")
	if len(got) != 2 {
		t.Fatalf("got %d gateways, want the 2 placed on local: %+v", len(got), got)
	}
	// Config order, not sorted: the order is what the operator arranged, and
	// re-ordering it on read makes cards move between polls.
	if got[0].ID != "default" || got[1].ID != "spare" {
		t.Errorf("got %q then %q, want config order (default, spare)", got[0].ID, got[1].ID)
	}
	for _, g := range got {
		if g.Placement.TargetID != "local" {
			t.Errorf("gateway %q is placed on %q, not local", g.ID, g.Placement.TargetID)
		}
	}
}

func TestGatewaysOn_UnknownMachineHasNoGateways(t *testing.T) {
	if got := fleet().GatewaysOn("nowhere"); len(got) != 0 {
		t.Fatalf("got %+v, want none", got)
	}
}

// The empty target id is what an unset placement reads as. It must not act as
// a wildcard that scoops up the whole fleet.
func TestGatewaysOn_EmptyTargetIsNotAWildcard(t *testing.T) {
	if got := fleet().GatewaysOn(""); len(got) != 0 {
		t.Fatalf("got %+v, want none — an empty machine id must match nothing, not everything", got)
	}
}

// FindGateway and GatewaysOn are the two reads that reach a stored gateway,
// and they must agree: a gateway GatewaysOn hands back has to be the same
// record, config and all, that FindGateway returns by id.
func TestFindGateway_AgreesWithGatewaysOn(t *testing.T) {
	c := fleet()
	c.Gateways[0].Config = catalog.GatewayConfig{Port: 4100}

	byID, ok := c.FindGateway("default")
	if !ok {
		t.Fatal("FindGateway(default) did not find it")
	}
	onLocal := c.GatewaysOn("local")[0]
	if byID.ID != onLocal.ID || byID.Config.Port != onLocal.Config.Port {
		t.Errorf("FindGateway gave %+v, GatewaysOn gave %+v", byID, onLocal)
	}

	if _, ok := c.FindGateway("nope"); ok {
		t.Error("FindGateway found a gateway that does not exist")
	}
}
