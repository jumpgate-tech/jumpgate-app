package catalog

import (
	"strings"
	"testing"
)

func TestNamesByIDKnowsBothSetsAndNothingElse(t *testing.T) {
	for id, want := range map[string]NodeNames{NamesLegacy: LegacyNames, NamesCurrent: CurrentNames} {
		got, err := NamesByID(id)
		if err != nil || got != want {
			t.Errorf("NamesByID(%q) = %+v, %v", id, got, err)
		}
	}
	if _, err := NamesByID("valve-node-app"); err == nil {
		t.Error("the legacy prefix is stored as \"\", never by name; anything else must be refused")
	}
}

func TestUnitNamesFollowThePrefix(t *testing.T) {
	if LegacyNames.ExecUnit() != "valve-node-app-exec.service" || LegacyNames.BeaconUnit() != "valve-node-app-beacon.service" {
		t.Fatalf("legacy units %v", LegacyNames.Units())
	}
	if CurrentNames.ExecUnit() != "jumpgate-node-exec.service" || CurrentNames.User != "jumpgate-node" || CurrentNames.DataRoot != "/var/lib/jumpgate-node" {
		t.Fatalf("current names %+v", CurrentNames)
	}
	if UnitPath("x.service") != "/etc/systemd/system/x.service" {
		t.Fatal(UnitPath("x.service"))
	}
	if ServiceUser != LegacyNames.User || ServiceGroup != LegacyNames.Group {
		t.Fatal("the legacy service account constants drifted from LegacyNames")
	}
}

func TestDefaultDataDirIsUnderTheNewRoot(t *testing.T) {
	if got := DefaultDataDir(369); got != "/var/lib/jumpgate-node/369" {
		t.Fatalf("DefaultDataDir = %s", got)
	}
	if got := DefaultDataDirFor(LegacyNames, 369); got != "/var/lib/valve-node-app/369" {
		t.Fatalf("legacy DefaultDataDirFor = %s", got)
	}
}

func TestRebaseOnlyMovesPathsUnderTheRoot(t *testing.T) {
	for _, c := range []struct {
		in, want string
		ok       bool
	}{
		{"/var/lib/valve-node-app/369", "/var/lib/jumpgate-node/369", true},
		{"/var/lib/valve-node-app", "/var/lib/jumpgate-node", true},
		{"/var/lib/valve-node-app-other/1", "/var/lib/valve-node-app-other/1", false}, // a sibling, not a child
		{"/data/node", "/data/node", false},
	} {
		got, ok := Rebase(c.in, LegacyNames.DataRoot, CurrentNames.DataRoot)
		if got != c.want || ok != c.ok {
			t.Errorf("Rebase(%s) = %s, %v", c.in, got, ok)
		}
	}
}

// Legacy boxes must render byte-identical units, or setup's wire Verify would
// see a "changed" config and restart every existing node on its next run.
func TestLegacyUnitsRenderAsBefore(t *testing.T) {
	w := WireConfig{ChainID: 369, ExecID: "reth", BeaconID: "lighthouse-pulse", DataDir: "/var/lib/valve-node-app/369"}
	execUnit, _, err := RenderUnits(w)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Description=valve-node-app ", "User=valve-node-app\n", "Group=valve-node-app\n"} {
		if !strings.Contains(execUnit, want) {
			t.Errorf("legacy unit lacks %q:\n%s", want, execUnit)
		}
	}
	w.Names, w.DataDir = NamesCurrent, "/var/lib/jumpgate-node/369"
	execUnit, _, _ = RenderUnits(w)
	for _, want := range []string{"Description=jumpgate-node ", "User=jumpgate-node\n", "ReadWritePaths=/var/lib/jumpgate-node/369\n"} {
		if !strings.Contains(execUnit, want) {
			t.Errorf("current unit lacks %q:\n%s", want, execUnit)
		}
	}
}
