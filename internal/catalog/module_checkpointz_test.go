package catalog

import (
	"bufio"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"
)

func TestCheckpointzClient(t *testing.T) {
	c, ok := ClientByID("checkpointz")
	if !ok {
		t.Fatal("no checkpointz client")
	}
	if c.Kind != "service" || c.Toolchain != "go" || c.PinVersion != "v0.32.0" {
		t.Errorf("client = %+v", c)
	}
	for _, want := range []string{"--branch v0.32.0 --depth 1", "go build -o /usr/local/bin/checkpointz .", "/usr/local/bin/checkpointz --help"} {
		if !strings.Contains(c.BuildCmd, want) {
			t.Errorf("BuildCmd lacks %q:\n%s", want, c.BuildCmd)
		}
	}
	// checkpointz has no --version flag: install's check runs --help.
	if c.VersionFlag() != "--help" {
		t.Errorf("VersionFlag = %q", c.VersionFlag())
	}
	if r, _ := ClientByID("reth"); r.VersionFlag() != "--version" {
		t.Errorf("reth VersionFlag = %q", r.VersionFlag())
	}
}

const checkpointzGolden1 = `global:
  listenAddr: "127.0.0.1:5555"
  logging: "info"
checkpointz:
  mode: "full"
  frontend:
    enabled: false
beacon:
  upstreams:
    - name: "beacon-1"
      address: "http://127.0.0.1:5052"
      dataProvider: true
`

func TestCheckpointzConfig(t *testing.T) {
	b := BoxModule{ID: "checkpointz", ChainID: 1, Upstreams: []string{"http://127.0.0.1:5052"}}
	u, err := RenderBoxModule(b)
	if err != nil {
		t.Fatal(err)
	}
	conf := b.ConfigDir() + "/config.yaml"
	if got := u.Files[conf]; got != checkpointzGolden1 {
		t.Errorf("config.yaml =\n%s\nwant\n%s", got, checkpointzGolden1)
	}
	if got, want := execStart(t, u.Unit), "checkpointz --config "+conf; got != want {
		t.Errorf("ExecStart = %q, want %q", got, want)
	}
	if u.Client != "checkpointz" || !reflect.DeepEqual(u.Ports, []PortUse{{5555, "tcp", false}}) {
		t.Errorf("unit = %+v", u)
	}
	// The sidecar is public: it never names an upstream.
	if strings.Contains(u.Files[b.SidecarPath()], "5052") {
		t.Errorf("module.json names an upstream: %s", u.Files[b.SidecarPath()])
	}

	b943 := BoxModule{ID: "checkpointz", ChainID: 943, BindAddr: "10.0.0.5", Port: 5600,
		Upstreams: []string{"http://127.0.0.1:5052", "https://10.0.0.6:5052/"}}
	u, err = RenderBoxModule(b943)
	if err != nil {
		t.Fatal(err)
	}
	got := u.Files[b943.ConfigDir()+"/config.yaml"]
	for _, want := range []string{"  custom_preset: true\n", `listenAddr: "10.0.0.5:5600"`, `name: "beacon-2"`, `address: "https://10.0.0.6:5052/"`} {
		if !strings.Contains(got, want) {
			t.Errorf("943 config lacks %q:\n%s", want, got)
		}
	}
	if !strings.Contains(u.Files[b943.SidecarPath()], `"bind":"10.0.0.5"`) {
		t.Errorf("sidecar = %s", u.Files[b943.SidecarPath()])
	}
}

func TestCheckpointzNeedsUpstream(t *testing.T) {
	_, err := RenderBoxModule(BoxModule{ID: "checkpointz", ChainID: 560048})
	if err == nil || err.Error() != "catalog: Checkpointz needs at least one synced beacon node on chain 560048" {
		t.Fatalf("err = %v", err)
	}
}

func TestCheckpointzRefusesBadFields(t *testing.T) {
	up := []string{"http://127.0.0.1:5052"}
	for _, b := range []BoxModule{
		{ID: "checkpointz", ChainID: 1, Upstreams: []string{"ftp://x"}},
		{ID: "checkpointz", ChainID: 1, Upstreams: []string{`http://x/"; rm -rf /`}},
		{ID: "checkpointz", ChainID: 1, Upstreams: []string{"http://x\n  evil: true"}},
		{ID: "checkpointz", ChainID: 1, Upstreams: []string{"http://a:5052", "http://a:5052"}},
		{ID: "checkpointz", ChainID: 1, Upstreams: up, BindAddr: "0.0.0.0"},
		{ID: "checkpointz", ChainID: 1, Upstreams: up, BindAddr: "203.0.113.7"},
		{ID: "checkpointz", ChainID: 1, Upstreams: up, BindAddr: "example.com"},
		{ID: "checkpointz", ChainID: 1, Upstreams: up, Port: 70000},
		{ID: "checkpointz", ChainID: 1, Upstreams: up, ExtIP: "203.0.113.7"},
	} {
		if _, err := RenderBoxModule(b); err == nil {
			t.Errorf("%+v accepted", b)
		}
	}
	// An upstream URL may carry a key: an error never echoes it.
	_, err := RenderBoxModule(BoxModule{ID: "checkpointz", ChainID: 1, Upstreams: []string{"ftp://secret-key@x/"}})
	if err == nil || strings.Contains(err.Error(), "secret-key") {
		t.Errorf("err = %v", err)
	}
	for _, bind := range []string{"127.0.0.1", "10.1.2.3", "192.168.1.4", "100.64.0.9", "fd00::5", "::1"} {
		if _, err := RenderBoxModule(BoxModule{ID: "checkpointz", ChainID: 1, Upstreams: up, BindAddr: bind}); err != nil {
			t.Errorf("bind %s: %v", bind, err)
		}
	}
}

func TestCheckpointzBetaOnPulseChain(t *testing.T) {
	m, ok := ModuleByID("checkpointz")
	if !ok {
		t.Fatal("no checkpointz module")
	}
	if !m.IsBeta(943) || !m.IsBeta(369) || m.IsBeta(1) || m.IsBeta(560048) {
		t.Errorf("BetaChains = %v", m.BetaChains)
	}
	for _, c := range []int{1, 560048, 369, 943} {
		if ok, r := m.Applies(c, ""); !ok {
			t.Errorf("Applies(%d) = %v %q", c, ok, r)
		}
	}
}

func TestCheckpointzDefaultUpstreamIsLocalBeacon(t *testing.T) {
	w := &WireConfig{ChainID: 560048}
	if got := CheckpointzDefaultUpstreams(w, 560048); !reflect.DeepEqual(got, []string{"http://127.0.0.1:5052"}) {
		t.Errorf("got %v", got)
	}
	// A beacon bound to every address still answers on loopback.
	w = &WireConfig{ChainID: 1, RPCBindAddr: "0.0.0.0", BeaconHTTPPort: 5099}
	if got := CheckpointzDefaultUpstreams(w, 1); !reflect.DeepEqual(got, []string{"http://127.0.0.1:5099"}) {
		t.Errorf("got %v", got)
	}
	w = &WireConfig{ChainID: 1, RPCBindAddr: "10.0.0.5"}
	if got := CheckpointzDefaultUpstreams(w, 1); !reflect.DeepEqual(got, []string{"http://10.0.0.5:5052"}) {
		t.Errorf("got %v", got)
	}
	if got := CheckpointzDefaultUpstreams(&WireConfig{ChainID: 1}, 560048); got != nil {
		t.Errorf("other chain: %v", got)
	}
	if got := CheckpointzDefaultUpstreams(nil, 1); got != nil {
		t.Errorf("no node: %v", got)
	}
}

func TestCheckpointzURL(t *testing.T) {
	if got := CheckpointzURL(BoxModule{ID: "checkpointz", ChainID: 1}); got != "" {
		t.Errorf("loopback: %q", got)
	}
	if got := CheckpointzURL(BoxModule{ID: "checkpointz", ChainID: 1, BindAddr: "10.0.0.5"}); got != "http://10.0.0.5:5555" {
		t.Errorf("got %q", got)
	}
	if got := CheckpointzURL(BoxModule{ID: "checkpointz", ChainID: 1, BindAddr: "fd00::5", Port: 5600}); got != "http://[fd00::5]:5600" {
		t.Errorf("got %q", got)
	}
}

// yamlPaths lists the key paths of a block-style YAML document, with list
// items as "[]": enough for checkpointz's example and the rendered config.
func yamlPaths(t *testing.T, doc string) []string {
	t.Helper()
	type level struct {
		indent int
		key    string
	}
	var stack []level
	var out []string
	sc := bufio.NewScanner(strings.NewReader(doc))
	for sc.Scan() {
		line := sc.Text()
		trimmed := strings.TrimLeft(line, " ")
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		indent := len(line) - len(trimmed)
		item := false
		if rest, ok := strings.CutPrefix(trimmed, "- "); ok {
			item, trimmed, indent = true, rest, indent+2
		}
		key, _, ok := strings.Cut(trimmed, ":")
		if !ok {
			continue
		}
		for len(stack) > 0 && stack[len(stack)-1].indent >= indent {
			stack = stack[:len(stack)-1]
		}
		if item && len(stack) > 0 {
			stack[len(stack)-1].key = strings.TrimSuffix(stack[len(stack)-1].key, "[]") + "[]"
		}
		var parts []string
		for _, l := range stack {
			parts = append(parts, l.key)
		}
		p := strings.Join(append(parts, key), ".")
		out = append(out, p)
		stack = append(stack, level{indent, key})
	}
	return out
}

// TestCheckpointzKeysExist checks every key the rendered config uses against
// v0.32.0: testdata/checkpointz-keys-v0.32.0.txt lists the yaml tags of its
// config structs, and every key of its example_config.yaml must be in it.
func TestCheckpointzKeysExist(t *testing.T) {
	raw, err := os.ReadFile("testdata/checkpointz-keys-v0.32.0.txt")
	if err != nil {
		t.Fatal(err)
	}
	var known, unused []string
	for _, l := range strings.Split(string(raw), "\n") {
		if l = strings.TrimSpace(l); l != "" && !strings.HasPrefix(l, "#") {
			if k, ok := strings.CutPrefix(l, "unused: "); ok {
				unused = append(unused, k)
				continue
			}
			known = append(known, l)
		}
	}
	ex, err := os.ReadFile("testdata/checkpointz-example-v0.32.0.yaml")
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range yamlPaths(t, string(ex)) {
		if !slices.Contains(known, p) && !slices.Contains(unused, p) {
			t.Errorf("example key %s is not in the key list", p)
		}
	}
	for _, chain := range []int{1, 943} {
		b := BoxModule{ID: "checkpointz", ChainID: chain, Upstreams: []string{"http://127.0.0.1:5052"}}
		u, err := RenderBoxModule(b)
		if err != nil {
			t.Fatal(err)
		}
		for _, p := range yamlPaths(t, u.Files[b.ConfigDir()+"/config.yaml"]) {
			if !slices.Contains(known, p) {
				t.Errorf("chain %d: rendered key %s is not a checkpointz v0.32.0 key", chain, p)
			}
		}
	}
}
