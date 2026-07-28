package catalog

import (
	"strings"
	"testing"
)

// The invocation asserted here is not invented: it is the one that was run by
// hand against ghcr.io/paradigmxyz/reth:latest and observed to seal blocks
// within ~10s on a chain reporting 0x539, with no chain data, no beacon
// client and no JWT. These tests exist to keep it that way.

// argIndex reports where value appears in args, or -1.
func argIndex(args []string, value string) int {
	for i, a := range args {
		if a == value {
			return i
		}
	}
	return -1
}

// flagValue returns the argument following flag, which is how every docker and
// reth option used here is spelled.
func flagValue(t *testing.T, args []string, flag string) string {
	t.Helper()
	i := argIndex(args, flag)
	if i < 0 || i+1 >= len(args) {
		t.Fatalf("no %s in %#v", flag, args)
	}
	return args[i+1]
}

func TestDevnetRunArgs_ZeroValueIsTheVerifiedInvocation(t *testing.T) {
	args := DevnetRunArgs(DevnetConfig{})

	if got := strings.Join(args[:4], " "); got != "run -d --name "+DevnetContainerName {
		t.Fatalf("want a detached, stably named container, got %q", got)
	}
	if v := flagValue(t, args, "--dev.block-time"); v != DefaultDevnetBlockTime {
		t.Fatalf("block time = %q, want %q", v, DefaultDevnetBlockTime)
	}
	// --dev is what removes the consensus client, the JWT and the chain data
	// all at once; without it this is an ordinary reth that syncs nothing.
	if argIndex(args, "--dev") < 0 {
		t.Fatalf("want --dev in %#v", args)
	}
	if v := flagValue(t, args, "--http.api"); v != "eth,net,web3,txpool" {
		// txpool_ is how "why has my transaction not been mined" gets an
		// answer; it is the namespace a devnet is most used through.
		t.Fatalf("http api = %q, want txpool included", v)
	}
	if v := flagValue(t, args, "--ws.api"); v != "eth,net,web3" {
		t.Fatalf("ws api = %q", v)
	}
	// Loopback host binds, wide container binds: a 127.0.0.1 listener inside
	// the container's own namespace is unreachable through a published port.
	if v := flagValue(t, args, "--http.addr"); v != "0.0.0.0" {
		t.Fatalf("in-container http bind = %q, want 0.0.0.0", v)
	}
	if v := flagValue(t, args, "--ws.addr"); v != "0.0.0.0" {
		t.Fatalf("in-container ws bind = %q, want 0.0.0.0", v)
	}
	if got := publishSpecs(args); len(got) != 2 || got[0] != "127.0.0.1:8545:8545" || got[1] != "127.0.0.1:8546:8546" {
		t.Fatalf("port mappings = %#v, want loopback 8545/8546", got)
	}
}

// A devnet is a scratch chain: nothing to mount, and nothing that should come
// back by itself after a reboot.
func TestDevnetRunArgs_NoVolumeAndNoRestartPolicy(t *testing.T) {
	args := DevnetRunArgs(DevnetConfig{})
	for _, unwanted := range []string{"-v", "--volume", "--restart", "--mount"} {
		if argIndex(args, unwanted) >= 0 {
			t.Fatalf("a devnet must not use %s (its state lives in the container and costs no host disk): %#v", unwanted, args)
		}
	}
}

// Everything after the image ref is reth's; everything before it is docker's.
// An option on the wrong side is silently read by the wrong program.
func TestDevnetRunArgs_ImageRefSeparatesDockerFlagsFromRethFlags(t *testing.T) {
	args := DevnetRunArgs(DevnetConfig{Platform: "linux/arm64"})

	image := argIndex(args, DefaultDevnetImage)
	if image < 0 {
		t.Fatalf("no image ref in %#v", args)
	}
	if got := args[image+1]; got != "node" {
		t.Fatalf("first argument to the image = %q, want the reth subcommand", got)
	}
	for _, docker := range []string{"--name", "--platform", "-p", "-d"} {
		if i := argIndex(args, docker); i > image {
			t.Fatalf("%s (%d) must precede the image ref (%d): docker stops reading its own flags there", docker, i, image)
		}
	}
	for _, reth := range []string{"--dev", "--http.port", "--ws.api"} {
		if i := argIndex(args, reth); i < image {
			t.Fatalf("%s (%d) must follow the image ref (%d)", reth, i, image)
		}
	}
	if got := args[image+1:]; strings.Join(got, " ") != strings.Join(DevnetCommand(DevnetConfig{Platform: "linux/arm64"}), " ") {
		// The post-image argv must be exactly DevnetCommand, because that is
		// what a live container's .Config.Cmd is compared against.
		t.Fatalf("post-image argv %#v is not DevnetCommand", got)
	}
}

// --platform is ALWAYS emitted, including when the caller resolved none.
//
// This inverts what this test used to assert, and the reason is a measured
// failure rather than a change of taste: an omitted --platform does not defer
// to the image manifest, it defers to DOCKER_DEFAULT_PLATFORM. With that
// variable exported as linux/amd64 on an arm64 machine, a devnet reset through
// the app created a QEMU-emulated reth that reported State=running and
// answered no RPC at all.
func TestDevnetRunArgs_PlatformIsAlwaysExplicit(t *testing.T) {
	fallback := DevnetRunArgs(DevnetConfig{})
	if v := flagValue(t, fallback, "--platform"); v != DefaultPlatform() {
		t.Fatalf("platform with none resolved = %q, want this app's own %q — omitting the flag hands the choice to DOCKER_DEFAULT_PLATFORM", v, DefaultPlatform())
	}
	args := DevnetRunArgs(DevnetConfig{Platform: "linux/amd64"})
	if v := flagValue(t, args, "--platform"); v != "linux/amd64" {
		t.Fatalf("platform = %q", v)
	}
}

// The devnet joins the private network so a gateway beside it can address it
// by CONTAINER NAME, which is what lets it publish nothing at all.
func TestDevnetRunArgs_JoinsTheNetworkWhenGivenOne(t *testing.T) {
	args := DevnetRunArgs(DevnetConfig{Network: "valve-node-app"})
	if v := flagValue(t, args, "--network"); v != "valve-node-app" {
		t.Fatalf("network = %q", v)
	}
	if i := argIndex(args, "--network"); i > argIndex(args, DefaultDevnetImage) {
		t.Fatal("--network is docker's, so it must fall before the image ref")
	}
	if i := argIndex(DevnetRunArgs(DevnetConfig{}), "--network"); i >= 0 {
		t.Fatal("no --network should be emitted when none was asked for")
	}
}

// A container caller uses the devnet's NAME and its IN-CONTAINER port; the
// published host port is not in that path at all, and using it would produce a
// URL that resolves and then refuses the connection.
func TestDevnetContainerEndpoints_UseTheFixedContainerPorts(t *testing.T) {
	d := DevnetConfig{HTTPPort: 8600, WSPort: 8601}
	if got, want := d.ContainerHTTPEndpoint(), "http://valve-node-app-devnet:8545"; got != want {
		t.Errorf("http: got %q, want %q", got, want)
	}
	if got, want := d.ContainerWSEndpoint(), "ws://valve-node-app-devnet:8546"; got != want {
		t.Errorf("ws: got %q, want %q", got, want)
	}
}

// The operator's choice of port lives on the HOST side of the mapping; the
// container's private namespace keeps the fixed reth ports.
func TestDevnetRunArgs_HostPortsMapOntoTheFixedContainerPorts(t *testing.T) {
	args := DevnetRunArgs(DevnetConfig{HTTPPort: 18545, WSPort: 18546, BindAddr: "0.0.0.0"})

	if got := publishSpecs(args); len(got) != 2 || got[0] != "0.0.0.0:18545:8545" || got[1] != "0.0.0.0:18546:8546" {
		t.Fatalf("port mappings = %#v", got)
	}
	if v := flagValue(t, args, "--http.port"); v != "8545" {
		t.Fatalf("in-container http port = %q, want the fixed 8545", v)
	}
	if v := flagValue(t, args, "--ws.port"); v != "8546" {
		t.Fatalf("in-container ws port = %q, want the fixed 8546", v)
	}
}

// An unbracketed IPv6 bind makes docker read the address's own colons as the
// mapping's field separators and reject the whole -p.
func TestDevnetPublishSpec_BracketsIPv6(t *testing.T) {
	cases := map[string]string{
		"127.0.0.1": "127.0.0.1:8545:8545",
		"::1":       "[::1]:8545:8545",
		"[::1]":     "[::1]:8545:8545",
	}
	for bind, want := range cases {
		if got := devnetPublishSpec(bind, 8545, 8545); got != want {
			t.Errorf("devnetPublishSpec(%q) = %q, want %q", bind, got, want)
		}
	}
}

func TestDevnetConfig_Defaults(t *testing.T) {
	var d DevnetConfig
	if d.ChainIDOrDefault() != DevnetChainID {
		t.Errorf("chain id = %d, want %d", d.ChainIDOrDefault(), DevnetChainID)
	}
	if d.Image() != DefaultDevnetImage {
		t.Errorf("image = %q", d.Image())
	}
	if d.Name() != DevnetContainerName {
		t.Errorf("name = %q", d.Name())
	}
	if d.HTTPEndpoint() != "http://127.0.0.1:8545" {
		t.Errorf("http endpoint = %q", d.HTTPEndpoint())
	}
	if d.WSEndpoint() != "ws://127.0.0.1:8546" {
		t.Errorf("ws endpoint = %q", d.WSEndpoint())
	}
}

// A wildcard bind names every interface but is not a destination — macOS
// refuses a connect to 0.0.0.0 — so the dialable form is loopback.
func TestDevnetConfig_EndpointsAreDialable(t *testing.T) {
	cases := map[string]string{
		"0.0.0.0":  "http://127.0.0.1:8545",
		"::":       "http://[::1]:8545",
		"::1":      "http://[::1]:8545",
		"10.0.0.4": "http://10.0.0.4:8545",
	}
	for bind, want := range cases {
		if got := (DevnetConfig{BindAddr: bind}).HTTPEndpoint(); got != want {
			t.Errorf("bind %q → %q, want %q", bind, got, want)
		}
	}
}

func TestDevnetConfig_ValidateRejectsWhatWouldFailQuietly(t *testing.T) {
	cases := map[string]struct {
		cfg  DevnetConfig
		want string
	}{
		// Serving 1337 under the label "1" would make every downstream chain
		// check a lie, and reth cannot be talked out of its dev genesis.
		"foreign chain id":  {DevnetConfig{ChainID: 1}, "custom genesis"},
		"unparsable block":  {DevnetConfig{BlockTime: "soon"}, "not a duration"},
		"zero block time":   {DevnetConfig{BlockTime: "0s"}, "must be positive"},
		"port out of range": {DevnetConfig{HTTPPort: 70000}, "out of range"},
		"shared host port":  {DevnetConfig{HTTPPort: 9000, WSPort: 9000}, "cannot share"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			err := c.cfg.Validate()
			if err == nil {
				t.Fatal("want an error")
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Fatalf("want an error mentioning %q, got %v", c.want, err)
			}
		})
	}
}

func TestDevnetConfig_ValidateAcceptsTheDefaultAndAnExplicit1337(t *testing.T) {
	if err := (DevnetConfig{}).Validate(); err != nil {
		t.Fatalf("the zero value must be a valid devnet: %v", err)
	}
	if err := (DevnetConfig{ChainID: DevnetChainID, BlockTime: "500ms"}).Validate(); err != nil {
		t.Fatalf("stating the chain id explicitly must be allowed: %v", err)
	}
}

// The point of the whole exercise: a devnet must be frontable by the same
// gateway machinery a real node is, with no special case anywhere.
func TestGatewayForDevnet_RendersAsAnOrdinaryGateway(t *testing.T) {
	d := DevnetConfig{HTTPPort: 18545, WSPort: 18546}

	up := d.Upstream()
	if !up.Local {
		t.Error("a devnet is a node the operator runs; it must be preferred over any fallback")
	}
	if up.RecentOnly {
		t.Error("a devnet has never pruned anything — bounding it to recent blocks routes historical calls away from the only node that can answer them")
	}
	// ws://, and the WS port: an http:// devnet upstream is one eRPC refuses
	// every eth_subscribe through.
	if up.Endpoint != "ws://127.0.0.1:18546" {
		t.Errorf("endpoint = %q", up.Endpoint)
	}

	cfg, err := RenderGatewayConfig(GatewayForDevnet(d))
	if err != nil {
		t.Fatalf("a devnet gateway must render like any other: %v", err)
	}
	for _, want := range []string{"chainId: 1337", `endpoint: "ws://127.0.0.1:18546"`, "architecture: evm"} {
		if !strings.Contains(cfg, want) {
			t.Errorf("missing %q:\n%s", want, cfg)
		}
	}
	// Local upstreams carry no fallback deprioritisation and no block bound.
	if strings.Contains(cfg, "tier:fallback") || strings.Contains(cfg, "latestBlockMinus") {
		t.Errorf("the devnet must be the preferred, unbounded upstream:\n%s", cfg)
	}
}

// publishSpecs collects every -p value, in order.
func publishSpecs(args []string) []string {
	var out []string
	for i, a := range args {
		if a == "-p" && i+1 < len(args) {
			out = append(out, args[i+1])
		}
	}
	return out
}
