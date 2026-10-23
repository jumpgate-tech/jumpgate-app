package catalog

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestUnitsByteIdenticalWithoutModules(t *testing.T) {
	cases := unitGoldenCases()
	if len(cases) == 0 {
		t.Fatal("no golden cases")
	}
	for _, mods := range [][]NodeModule{nil, {}} {
		for _, c := range cases {
			w := c.W
			w.Modules = mods
			execUnit, beaconUnit, err := RenderUnits(w)
			if err != nil {
				t.Fatalf("%s: %v", c.Name, err)
			}
			for suffix, got := range map[string]string{"exec": execUnit, "beacon": beaconUnit} {
				want, err := os.ReadFile(filepath.Join(unitGoldensDir, c.Name+"-"+suffix+".service"))
				if err != nil {
					t.Fatal(err)
				}
				if got != string(want) {
					t.Errorf("%s-%s (Modules %#v): unit differs from golden", c.Name, suffix, mods)
				}
			}
		}
	}
}

// fakeExtra is a second node module, so composition has two to order.
func fakeExtra() Module {
	m := fakeMsgboard()
	m.ID, m.Name = "watch", "Extra"
	m.Namespaces = []string{"txpool", "msgboard"}
	m.Methods = nil
	m.Flags = []string{"--extra.flag"}
	return m
}

// The all/none guard ignores case: geth reads "ALL" as all.
func TestExecHTTPAPIRefusesAllAnyCase(t *testing.T) {
	for _, ns := range []string{"ALL", "None"} {
		m := fakeMsgboard()
		m.Namespaces = []string{ns}
		withModules(t, []Module{m})
		w := WireConfig{ChainID: 369, ExecID: "reth", Modules: []NodeModule{{ID: m.ID}}}
		if got, err := execHTTPAPI(w); err == nil {
			t.Errorf("%s: got %q, want an error", ns, got)
		}
	}
}

func TestExecHTTPAPIComposition(t *testing.T) {
	withModules(t, []Module{fakeMsgboard(), fakeExtra()})
	w := WireConfig{ChainID: 369, ExecID: "reth"}
	if got, err := execHTTPAPI(w); err != nil || got != "" {
		t.Fatalf("no modules: %q, %v; want empty", got, err)
	}
	w.Modules = []NodeModule{{ID: "msgboard"}}
	if got, err := execHTTPAPI(w); err != nil || got != "eth,net,web3,msgboard" {
		t.Fatalf("msgboard: %q, %v", got, err)
	}
	// Reversed wire order, duplicate namespace: catalog order, collapsed.
	w.Modules = []NodeModule{{ID: "watch"}, {ID: "msgboard"}}
	if got, err := execHTTPAPI(w); err != nil || got != "eth,net,web3,msgboard,txpool" {
		t.Fatalf("both: %q, %v", got, err)
	}
	if !w.HasModule("watch") || w.HasModule("checkpointz") {
		t.Fatal("HasModule")
	}
	execUnit, _, err := RenderUnits(WireConfig{ChainID: 369, ExecID: "reth", BeaconID: "lighthouse-pulse", DataDir: "/var/lib/x/369", Modules: w.Modules})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(execUnit, " --full --http.api eth,net,web3,msgboard,txpool --extra.flag\n") {
		t.Fatalf("exec unit missing composed flags:\n%s", execUnit)
	}
}

func TestExecHTTPAPIRefusesAllNone(t *testing.T) {
	for _, bad := range []string{"all", "none"} {
		m := fakeMsgboard()
		m.Namespaces = []string{bad}
		withModules(t, []Module{m})
		_, err := execHTTPAPI(WireConfig{ChainID: 369, ExecID: "reth", Modules: []NodeModule{{ID: "msgboard"}}})
		if err == nil || !strings.Contains(err.Error(), `--http.api would contain "`+bad+`"`) {
			t.Fatalf("%s: err = %v", bad, err)
		}
	}
}

func TestValidateModules(t *testing.T) {
	withModules(t, []Module{fakeMsgboard(), fakeBox("el-bootnode")})
	base := WireConfig{ChainID: 369, ExecID: "reth"}
	mods := func(ms ...NodeModule) func(WireConfig) WireConfig {
		return func(w WireConfig) WireConfig { w.Modules = ms; return w }
	}
	cases := []struct {
		name string
		w    func(WireConfig) WireConfig
		want string
	}{
		{"none", mods(), ""},
		{"good", mods(NodeModule{ID: "msgboard", Public: "post"}), ""},
		{"read", mods(NodeModule{ID: "msgboard", Public: "read"}), ""},
		{"unknown", mods(NodeModule{ID: "nope"}), `catalog: unknown module "nope"`},
		{"dup", mods(NodeModule{ID: "msgboard"}, NodeModule{ID: "msgboard"}), `catalog: module "msgboard" listed twice`},
		{"chain1", func(w WireConfig) WireConfig { w.ChainID = 1; w.Modules = []NodeModule{{ID: "msgboard"}}; return w }, "catalog: the msgboard module does not apply to chain 1"},
		{"go-pulse", func(w WireConfig) WireConfig {
			w.ExecID = "go-pulse"
			w.Modules = []NodeModule{{ID: "msgboard"}}
			return w
		}, "The message board needs reth (valve fork). This node runs go-pulse."},
		{"erigon-pulse", func(w WireConfig) WireConfig {
			w.ExecID = "erigon-pulse"
			w.Modules = []NodeModule{{ID: "msgboard"}}
			return w
		}, "The message board needs reth (valve fork). This node runs erigon-pulse."},
		{"public", mods(NodeModule{ID: "msgboard", Public: "x"}), `catalog: public must be "", "read" or "post", not "x"`},
		{"box", mods(NodeModule{ID: "el-bootnode"}), "catalog: el-bootnode is not a node module"},
	}
	for _, c := range cases {
		err := ValidateModules(c.w(base))
		if c.want == "" {
			if err != nil {
				t.Errorf("%s: %v", c.name, err)
			}
			continue
		}
		if err == nil || err.Error() != c.want {
			t.Errorf("%s: err = %v, want %q", c.name, err, c.want)
		}
	}
}

// apiValues returns every --http.api / --ws.api value in args.
func apiValues(args []string) []string {
	var out []string
	for i, a := range args {
		for _, f := range []string{"--http.api", "--ws.api"} {
			if a == f && i+1 < len(args) {
				out = append(out, args[i+1])
			} else if v, ok := strings.CutPrefix(a, f+"="); ok {
				out = append(out, v)
			}
		}
	}
	return out
}

func checkAPIValues(t *testing.T, label string, args []string) {
	t.Helper()
	for _, v := range apiValues(args) {
		for _, ns := range strings.Split(v, ",") {
			if strings.EqualFold(ns, "all") || strings.EqualFold(ns, "none") {
				t.Errorf("%s: API list %q contains %q", label, v, ns)
			}
		}
	}
}

// TestNoAPIListIsAll is merge-blocking: no rendered API list may hold
// "all" or "none".
func TestNoAPIListIsAll(t *testing.T) {
	withAPI := 0
	for _, reg := range [][]Module{modules, {fakeMsgboard(), fakeExtra()}} {
		withModules(t, reg)
		for _, n := range Networks() {
			for _, exec := range n.ExecClients {
				var applicable []string
				for _, m := range Modules() {
					if ok, _ := m.Applies(n.ChainID, exec); ok && m.Placement == PlaceNode {
						applicable = append(applicable, m.ID)
					}
				}
				for mask := 0; mask < 1<<len(applicable); mask++ {
					var mods []NodeModule
					for i, id := range applicable {
						if mask&(1<<i) != 0 {
							mods = append(mods, NodeModule{ID: id})
						}
					}
					w := WireConfig{ChainID: n.ChainID, ExecID: exec, BeaconID: n.BeaconClients[0], DataDir: "/var/lib/x", Modules: mods}
					execUnit, _, err := RenderUnits(w)
					if err != nil {
						t.Fatalf("%d %s %v: %v", n.ChainID, exec, mods, err)
					}
					checkAPIValues(t, fmt.Sprintf("%d %s %v", n.ChainID, exec, mods), strings.Fields(execUnit))
					if strings.Contains(execUnit, "--http.api") {
						withAPI++
					}
				}
			}
		}
	}
	if withAPI == 0 {
		t.Fatal("no rendered unit has --http.api; the guard checks nothing")
	}
	devnet := DevnetCommand(DevnetConfig{})
	checkAPIValues(t, "devnet", devnet)
	if !slices.Contains(devnet, "--http.api") {
		t.Fatal("devnet renders no --http.api; the guard checks nothing")
	}
}
