package catalog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var (
	testEnode1 = "enode://" + strings.Repeat("ab12", 32) + "@203.0.113.7:30303"
	testEnode2 = "enode://" + strings.Repeat("cd34", 32) + "@198.51.100.2:30303?discport=30301"
	testENR1   = "enr:-Iq4QJk4WqRkjsX5c2CXtOra6HnxN-BMXnWhmhEQO9Bn9iABTJGdjUOurM7Btj1ouKaFkvTRoju5vz2GPmVON2dffQKGAX53x8JigmlkgnY0"
	testENR2   = "enr:-Ly4QFPk-cTMxZ3jWTafiNblEZkQIXGF2aVzCIGW0uHp6KaEAvBMoctE8S7YU0qZtuS7By0AA4YMfKoN9ls_GJRccVpFh2F0dG5ldHOI__________-EZXRoMpCC9KcrAgAQIIS2AQAAAAAAgmlkgnY0"
)

func bootWire(chain int, exec, beacon string) WireConfig {
	return WireConfig{ChainID: chain, ExecID: exec, BeaconID: beacon, DataDir: "/var/lib/x/1",
		Bootnodes: []string{testEnode1, testENR1, testEnode2, testENR2}}
}

func mustNet(t *testing.T, id int) Network {
	t.Helper()
	n, ok := NetworkByChainID(id)
	if !ok {
		t.Fatalf("no chain %d", id)
	}
	return n
}

func TestBootnodesRendered(t *testing.T) {
	enodes := " --bootnodes " + testEnode1 + "," + testEnode2
	enrs := " --boot-nodes " + testENR1 + "," + testENR2
	cases := []struct {
		chain        int
		exec, beacon string
		execWant     string
		beaconWant   []string
	}{
		{1, "reth", "lighthouse", enodes, []string{enrs}},
		{1, "geth", "lighthouse", enodes, []string{enrs}},
		{369, "go-pulse", "lighthouse-pulse", enodes, []string{enrs}},
		{369, "reth", "prysm-pulse", enodes, []string{" --bootstrap-node=" + testENR1, " --bootstrap-node=" + testENR2}},
		{369, "erigon-pulse", "lighthouse-pulse", enodes, []string{enrs}},
	}
	for _, c := range cases {
		n := mustNet(t, c.chain)
		if !contains(n.ExecClients, c.exec) || !contains(n.BeaconClients, c.beacon) {
			t.Fatalf("%s/%s not valid on %d", c.exec, c.beacon, c.chain)
		}
		eu, bu, err := RenderUnits(bootWire(c.chain, c.exec, c.beacon))
		if err != nil {
			t.Fatalf("%s/%s: %v", c.exec, c.beacon, err)
		}
		if !strings.Contains(eu, c.execWant) {
			t.Errorf("%s exec unit lacks %q:\n%s", c.exec, c.execWant, eu)
		}
		for _, bw := range c.beaconWant {
			if !strings.Contains(bu, bw) {
				t.Errorf("%s beacon unit lacks %q:\n%s", c.beacon, bw, bu)
			}
		}
		if strings.Contains(eu, "enr:-") {
			t.Errorf("%s exec unit got an enr entry", c.exec)
		}
		if strings.Contains(bu, "enode://") {
			t.Errorf("%s beacon unit got an enode entry", c.beacon)
		}
	}
	// Only one kind: the other unit gets no flag at all.
	w := bootWire(1, "reth", "lighthouse")
	w.Bootnodes = []string{testEnode1}
	_, bu, err := RenderUnits(w)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(bu, "--boot-nodes") {
		t.Errorf("beacon unit has an empty --boot-nodes:\n%s", bu)
	}
}

func TestBootnodesByteIdenticalWhenEmpty(t *testing.T) {
	for _, c := range unitGoldenCases() {
		c.W.Bootnodes = []string{}
		eu, bu, err := RenderUnits(c.W)
		if err != nil {
			t.Fatalf("%s: %v", c.Name, err)
		}
		for suffix, body := range map[string]string{"exec": eu, "beacon": bu} {
			want, err := os.ReadFile(filepath.Join(unitGoldensDir, c.Name+"-"+suffix+".service"))
			if err != nil {
				t.Fatal(err)
			}
			if string(want) != body {
				t.Errorf("%s-%s differs from its golden", c.Name, suffix)
			}
		}
	}
}

func TestBootnodesValidated(t *testing.T) {
	w := bootWire(1, "reth", "lighthouse")
	w.Bootnodes = []string{"http://x"}
	_, _, err := RenderUnits(w)
	if err == nil || err.Error() != "catalog: a bootnode must start with enode:// or enr:-" {
		t.Fatalf("err = %v", err)
	}
	for _, bad := range []string{
		testEnode1 + " --foo", testEnode1 + ",x", testEnode1 + ";rm", "enr:-abc$(id)", "enr:-a`b`",
		"enr:-a%n", testEnode1 + "\n", "enr:-a'b", "enr:-a\"b", "enode://", "enr:-", "enr:-a\\b",
	} {
		w.Bootnodes = []string{bad}
		if _, _, err := RenderUnits(w); err == nil {
			t.Errorf("RenderUnits accepted %q", bad)
		}
		if err := ValidateBootnode(bad); err == nil {
			t.Errorf("ValidateBootnode accepted %q", bad)
		}
	}
	for _, good := range []string{testEnode1, testEnode2, testENR1, testENR2} {
		if err := ValidateBootnode(good); err != nil {
			t.Errorf("%q: %v", good, err)
		}
	}
}
