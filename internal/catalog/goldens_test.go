package catalog

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// updateGoldens rewrites testdata/units-1ef27cf. It is only ever run on a
// tree at 1ef27cf: the goldens record what every node rendered before
// modules existed, and later tasks compare against them, never regenerate.
var updateGoldens = flag.Bool("update-goldens", false, "write the unit goldens to testdata/units-1ef27cf")

const unitGoldensDir = "testdata/units-1ef27cf"

// goldenCase is one rendered pair in the golden set.
type goldenCase struct {
	Name string
	W    WireConfig
}

// goldenChains are the chains that shipped at 1ef27cf. Hoodi came later and
// has no golden.
var goldenChains = map[int]bool{1: true, 369: true, 943: true}

// unitGoldenCases is every network × exec × beacon pair valid at 1ef27cf,
// under both name sets, archive and full, at the default and a non-default
// RPC bind.
func unitGoldenCases() []goldenCase {
	var out []goldenCase
	for _, n := range Networks() {
		if !goldenChains[n.ChainID] {
			continue
		}
		for _, exec := range n.ExecClients {
			for _, beacon := range n.BeaconClients {
				for _, names := range []string{NamesLegacy, NamesCurrent} {
					nn, _ := NamesByID(names)
					namesTag := "legacy"
					if names == NamesCurrent {
						namesTag = "current"
					}
					for _, archive := range []bool{false, true} {
						tier := "full"
						if archive {
							tier = "archive"
						}
						for _, bind := range []string{"", "100.64.0.7"} {
							bindTag := "loopback"
							if bind != "" {
								bindTag = "bind"
							}
							out = append(out, goldenCase{
								Name: fmt.Sprintf("%d-%s-%s-%s-%s-%s", n.ChainID, exec, beacon, namesTag, tier, bindTag),
								W: WireConfig{
									ChainID:     n.ChainID,
									ExecID:      exec,
									BeaconID:    beacon,
									DataDir:     fmt.Sprintf("%s/%d", nn.DataRoot, n.ChainID),
									Archive:     archive,
									Names:       names,
									RPCBindAddr: bind,
								},
							})
						}
					}
				}
			}
		}
	}
	return out
}

func TestWriteUnitGoldens(t *testing.T) {
	if !*updateGoldens {
		t.Skip("run with -update-goldens on an unmodified tree at 1ef27cf")
	}
	if err := os.MkdirAll(unitGoldensDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, c := range unitGoldenCases() {
		execUnit, beaconUnit, err := RenderUnits(c.W)
		if err != nil {
			t.Fatalf("%s: %v", c.Name, err)
		}
		for suffix, body := range map[string]string{"exec": execUnit, "beacon": beaconUnit} {
			p := filepath.Join(unitGoldensDir, c.Name+"-"+suffix+".service")
			if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
}
