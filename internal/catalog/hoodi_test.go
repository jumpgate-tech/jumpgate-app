package catalog

import (
	"errors"
	"strings"
	"testing"
)

func TestHoodiNetwork(t *testing.T) {
	n, ok := NetworkByChainID(560048)
	if !ok {
		t.Fatal("NetworkByChainID(560048): not found")
	}
	if n.Name != "Hoodi" {
		t.Errorf("Name = %q, want Hoodi", n.Name)
	}
	if strings.Join(n.ExecClients, ",") != "reth,geth" {
		t.Errorf("ExecClients = %v, want [reth geth]", n.ExecClients)
	}
	if strings.Join(n.BeaconClients, ",") != "lighthouse" {
		t.Errorf("BeaconClients = %v, want [lighthouse]", n.BeaconClients)
	}
	if n.SlotSeconds != 12 {
		t.Errorf("SlotSeconds = %d, want 12", n.SlotSeconds)
	}
	if n.SnapshotSizeTB != 0 {
		t.Errorf("SnapshotSizeTB = %v, want 0 (Valve publishes no Hoodi snapshot)", n.SnapshotSizeTB)
	}
}

func TestSlotSeconds(t *testing.T) {
	for id, want := range map[int]int{1: 12, 369: 10, 943: 10, 560048: 12} {
		n, ok := NetworkByChainID(id)
		if !ok {
			t.Fatalf("chain %d not found", id)
		}
		if n.SlotSeconds != want {
			t.Errorf("chain %d: SlotSeconds = %d, want %d", id, n.SlotSeconds, want)
		}
	}
}

func TestHoodiUnits(t *testing.T) {
	for exec, want := range map[string]string{"reth": "--chain hoodi", "geth": " --hoodi"} {
		execUnit, beaconUnit, err := RenderUnits(WireConfig{ChainID: 560048, ExecID: exec, BeaconID: "lighthouse", DataDir: "/var/lib/jumpgate-node/560048"})
		if err != nil {
			t.Fatalf("%s: %v", exec, err)
		}
		if !strings.Contains(execUnit, want) {
			t.Errorf("%s exec unit lacks %q:\n%s", exec, want, execUnit)
		}
		if !strings.Contains(beaconUnit, "--network hoodi") {
			t.Errorf("beacon unit lacks --network hoodi:\n%s", beaconUnit)
		}
	}
}

func TestExpectedBytesNoGuidance(t *testing.T) {
	for _, archive := range []bool{false, true} {
		got, err := ExpectedBytes(560048, archive)
		if !errors.Is(err, ErrNoSizeGuidance) {
			t.Errorf("archive=%v: err = %v, want ErrNoSizeGuidance", archive, err)
		}
		if got != 0 {
			t.Errorf("archive=%v: got %d, want 0", archive, got)
		}
	}
}

// The byte-for-byte golden check is TestUnitsByteIdenticalWithoutModules
// (modules plan Task 3); this only guards the Hoodi flag from leaking.
func TestMainnetGethUnitUnchanged(t *testing.T) {
	execUnit, _, err := RenderUnits(WireConfig{ChainID: 1, ExecID: "geth", BeaconID: "lighthouse", DataDir: "/var/lib/jumpgate-node/1"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(execUnit, "--hoodi") {
		t.Errorf("mainnet geth unit must not carry --hoodi:\n%s", execUnit)
	}
}
