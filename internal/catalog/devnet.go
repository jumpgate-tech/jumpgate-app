package catalog

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

// A devnet is a throwaway chain whose whole purpose is to make the rest of
// this app testable on a laptop: a node to point a gateway at, an RPC port to
// health-check, a chain that produces blocks — available in seconds and
// costing nothing on disk.
//
// WHY it is modelled here rather than as a mode of WireConfig: a WireConfig
// describes a REAL chain's node, and every field it carries is a consequence
// of that — a checkpoint URL to sync from, a beacon client to shake hands with
// over the engine API, a JWT secret shared between the two, terabytes of
// dataset to size, a de-rooted service account to run it as. A devnet has none
// of those, and not by omission. It self-seals: reth's --dev mode seals a block
// on a timer with NO consensus client at all, from a genesis baked into the
// image, so there is no second process, no shared secret, no checkpoint, no
// snapshot, and nothing on the host's disk to size. Expressing that as a
// WireConfig with two thirds of its fields ignored would be a lie about what a
// devnet is.
//
// Everything below was verified by hand against ghcr.io/paradigmxyz/reth:latest
// rather than read off documentation:
//
//   - the image is multi-arch (linux/amd64 + linux/arm64), so unlike the
//     gateway image (internal/ops/docker.go, built on the target because no
//     arm64 build is published) a devnet is a plain pull on every desktop this
//     app targets;
//   - --dev with no chain data and no beacon client produces blocks roughly ten
//     seconds after `docker run` returns;
//   - the chain it serves reports 0x539 — 1337 — over eth_chainId.
const (
	// DevnetChainID is the chain id reth's --dev genesis serves, 0x539.
	//
	// It is a constant rather than a setting because reth's dev chain spec is
	// baked in: changing the id means supplying a custom genesis.json and
	// mounting it into the container, which is a materially different feature
	// (and one that would reintroduce the host-path bind-mount problem the
	// gateway backend already has to work around). DevnetConfig.ChainID exists
	// so the value is stated where callers read it, and Validate rejects any
	// other value instead of silently serving 1337 under a different label.
	DevnetChainID = 1337

	// DefaultDevnetImage is the upstream reth image. It is PULLED, not built:
	// upstream publishes both linux/amd64 and linux/arm64, and unlike eRPC
	// there is no valve fork whose behaviour we depend on, so building it on
	// the target would cost minutes to produce a byte-identical binary.
	DefaultDevnetImage = "ghcr.io/paradigmxyz/reth:latest"

	// DevnetContainerName is the stable name a devnet container always
	// carries, for the same reason ops.ERPCContainerName is stable: it is how
	// exists/running/remove find the container across app restarts, and what
	// makes re-provisioning idempotent rather than spawning a second chain.
	DevnetContainerName = "valve-node-app-devnet"

	// DefaultDevnetBlockTime is how often --dev seals a block. Two seconds is
	// short enough that a test watching for a new head does not dominate its
	// own runtime, and long enough that an idle devnet left running on a
	// laptop is not a busy loop.
	DefaultDevnetBlockTime = "2s"
)

// The ports the devnet listens on INSIDE the container. Fixed, and
// deliberately not configurable, for exactly the reason ops.erpcContainerPort
// is fixed: the container's port namespace is private, so there is nothing in
// there to collide with, and the operator's choice of port belongs on the HOST
// side of the -p mapping instead. Keeping the in-container side constant also
// keeps the reth argv constant, which is what lets a running container's
// command be compared against a freshly rendered one to detect drift.
const (
	devnetContainerHTTPPort = 8545
	devnetContainerWSPort   = 8546
)

// devnetHTTPAPIs / devnetWSAPIs are the RPC namespaces the devnet exposes.
//
// txpool is included on HTTP because the single most common thing to do with a
// devnet is submit a transaction and then ask why it has not been mined;
// without txpool_ that question has no answer. The admin, debug and trace
// namespaces are left off: they are not needed to exercise this app, and a
// devnet's ports are published on the operator's own machine.
const (
	devnetHTTPAPIs = "eth,net,web3,txpool"
	devnetWSAPIs   = "eth,net,web3"
)

// DevnetConfig describes one local devnet container. The zero value is a
// valid, loopback-bound devnet on the default ports — that is deliberate,
// because "just give me a chain" is the overwhelmingly common request and it
// should not require filling in a struct.
type DevnetConfig struct {
	// ChainID is the chain the devnet is expected to serve (0 → DevnetChainID).
	// It is not passed to reth — see DevnetChainID for why it cannot be — it is
	// the value the readiness probe asserts eth_chainId returns, which is what
	// turns "something is listening on 8545" into "our devnet is listening on
	// 8545".
	ChainID int

	// BlockTime is reth's --dev.block-time ("" → DefaultDevnetBlockTime).
	BlockTime string

	// ImageRef is the container image ("" → DefaultDevnetImage).
	ImageRef string

	// ContainerName is the container's --name ("" → DevnetContainerName).
	ContainerName string

	// BindAddr is the HOST address the published ports bind to ("" →
	// 127.0.0.1). As with the gateway, this is the only thing controlling who
	// can reach the devnet: the in-container listeners are always bound wide,
	// because a loopback listener inside the container's own network namespace
	// is unreachable through a published port.
	BindAddr string

	// HTTPPort and WSPort are the HOST-side ports (0 → 8545 / 8546).
	HTTPPort int
	WSPort   int

	// Platform is the image platform to run as, e.g. "linux/arm64" ("" → emit
	// no --platform and let the engine pick from the manifest).
	//
	// It is NOT defaulted here, which is a departure from ops.ERPCRunSpec. This
	// package performs no I/O and cannot import internal/ops (ops imports
	// catalog), so it has no way to reach ops.EnginePlatform — and a --platform
	// guessed from the wrong reading is strictly worse than none, since it
	// turns the engine's correct multi-arch manifest selection into a hard "no
	// matching manifest" failure. Callers that know the target's architecture
	// pass it; internal/setup resolves it with ops.EnginePlatform, which
	// already yields "" for an architecture nobody recognized.
	Platform string
}

// ChainIDOrDefault resolves the chain id (0 → DevnetChainID).
func (d DevnetConfig) ChainIDOrDefault() int {
	if d.ChainID == 0 {
		return DevnetChainID
	}
	return d.ChainID
}

// BlockTimeOrDefault resolves the block time ("" → DefaultDevnetBlockTime).
func (d DevnetConfig) BlockTimeOrDefault() string {
	if s := strings.TrimSpace(d.BlockTime); s != "" {
		return s
	}
	return DefaultDevnetBlockTime
}

// Image resolves the image ref ("" → DefaultDevnetImage).
func (d DevnetConfig) Image() string {
	if s := strings.TrimSpace(d.ImageRef); s != "" {
		return s
	}
	return DefaultDevnetImage
}

// Name resolves the container name ("" → DevnetContainerName).
func (d DevnetConfig) Name() string {
	if s := strings.TrimSpace(d.ContainerName); s != "" {
		return s
	}
	return DevnetContainerName
}

// Bind resolves the host bind address ("" → loopback).
func (d DevnetConfig) Bind() string {
	if d.BindAddr == "" {
		return "127.0.0.1"
	}
	return d.BindAddr
}

// HTTP resolves the host-side JSON-RPC port (0 → 8545).
func (d DevnetConfig) HTTP() int {
	if d.HTTPPort == 0 {
		return devnetContainerHTTPPort
	}
	return d.HTTPPort
}

// WS resolves the host-side WebSocket port (0 → 8546).
func (d DevnetConfig) WS() int {
	if d.WSPort == 0 {
		return devnetContainerWSPort
	}
	return d.WSPort
}

// HTTPEndpoint and WSEndpoint are the URLs a caller on the host dials. A
// wildcard bind is reported as loopback because a wildcard names every
// interface but is not itself a destination — macOS refuses a connect to
// 0.0.0.0 outright — and a wildcard listener is on loopback anyway.
func (d DevnetConfig) HTTPEndpoint() string {
	return "http://" + net.JoinHostPort(devnetDialHost(d.Bind()), strconv.Itoa(d.HTTP()))
}

func (d DevnetConfig) WSEndpoint() string {
	return "ws://" + net.JoinHostPort(devnetDialHost(d.Bind()), strconv.Itoa(d.WS()))
}

// devnetDialHost turns a bind address into something connectable.
func devnetDialHost(bind string) string {
	host := strings.Trim(bind, "[]")
	switch host {
	case "", "0.0.0.0":
		return "127.0.0.1"
	case "::", "::0":
		return "::1"
	}
	return host
}

// Validate rejects the configurations that would otherwise fail confusingly
// or, worse, quietly.
//
// It exists so PlanDevnet can fail at plan time, exactly as PlanGateway fails
// by rendering the config up front, rather than partway through RunAll with a
// container already removed.
func (d DevnetConfig) Validate() error {
	if d.ChainID != 0 && d.ChainID != DevnetChainID {
		return fmt.Errorf("catalog: devnet: chain id %d is not available — reth's --dev genesis is fixed at %d, and serving another id needs a custom genesis this app does not render", d.ChainID, DevnetChainID)
	}

	// A block time reth cannot parse is a container that exits within a
	// second of starting, whose failure the readiness poll then reports as a
	// timeout — a thoroughly misleading way to learn about a typo. Note this
	// accepts the Go-duration subset of what reth actually takes (it also
	// understands spellings like "2sec"); that is deliberate, since every
	// value this app renders is in the subset and a false rejection here is
	// cheap to work around while a false accept is not.
	bt := d.BlockTimeOrDefault()
	dur, err := time.ParseDuration(bt)
	if err != nil {
		return fmt.Errorf("catalog: devnet: block time %q is not a duration (want e.g. \"2s\", \"500ms\"): %w", bt, err)
	}
	if dur <= 0 {
		return fmt.Errorf("catalog: devnet: block time %q must be positive — a devnet that never seals a block never becomes ready", bt)
	}

	for _, p := range []struct {
		name string
		port int
	}{{"http", d.HTTP()}, {"ws", d.WS()}} {
		if p.port < 1 || p.port > 65535 {
			return fmt.Errorf("catalog: devnet: %s port %d is out of range", p.name, p.port)
		}
	}
	if d.HTTP() == d.WS() {
		// docker accepts two -p flags onto the same host port and then fails
		// at container start with "port is already allocated", naming neither
		// mapping. Saying so here is considerably more useful.
		return fmt.Errorf("catalog: devnet: http and ws cannot share host port %d", d.HTTP())
	}
	return nil
}

// DevnetCommand renders the argv passed to the image — everything AFTER the
// image ref in a `docker run`, i.e. reth's own arguments.
//
// It is split out from DevnetRunArgs for one concrete reason: this is exactly
// what `docker inspect -f '{{json .Config.Cmd}}'` reports for a running
// container, so a caller can compare a live devnet's command against a freshly
// rendered one and tell "the devnet is up" apart from "the devnet is up on the
// settings it was started with". A devnet has no config file, so this argv is
// the only place a settings change (a different block time, say) is visible —
// and without that comparison, changing a setting would leave the old
// container serving while setup reported success.
//
// The listeners bind 0.0.0.0 INSIDE the container, always. The container has
// its own network namespace, so a 127.0.0.1 listener there is reachable only
// from inside the container and docker's published port forwards into a black
// hole. Host-side exposure is enforced by the -p bind address instead, which
// is strictly safer: it is the host kernel refusing the connection rather than
// the application.
func DevnetCommand(d DevnetConfig) []string {
	return []string{
		"node",
		"--dev",
		"--dev.block-time", d.BlockTimeOrDefault(),
		"--http",
		"--http.addr", "0.0.0.0",
		"--http.port", strconv.Itoa(devnetContainerHTTPPort),
		"--http.api", devnetHTTPAPIs,
		"--ws",
		"--ws.addr", "0.0.0.0",
		"--ws.port", strconv.Itoa(devnetContainerWSPort),
		"--ws.api", devnetWSAPIs,
	}
}

// DevnetRunArgs renders the argv for `docker run` — WITHOUT the leading
// "docker" — for a local devnet.
//
// Pure (no Executor, no error) for the same reason ops.ERPCRunArgs is: the
// argument list carries all the fiddly, easy-to-regress detail — flag order,
// IPv6 bracketing in a port mapping, which side of the image ref an option
// falls on — and that is worth asserting on directly without a fake process
// anywhere in the way.
//
// Two deliberate differences from ERPCRunArgs, both of which follow from what
// a devnet IS:
//
//   - No --restart. The gateway is infrastructure and should survive a reboot;
//     a devnet is a scratch chain whose state is inside the container, so
//     resurrecting it on boot would silently hand back a chain the operator had
//     finished with. Setup re-creates it on demand instead.
//   - No -v. There is nothing to mount: reth --dev takes its genesis from the
//     image and keeps its database in the container's own writable layer, which
//     is precisely why a devnet costs zero host disk and why it is immune to the
//     bind-mount-visibility problem that shapes the gateway's config path.
func DevnetRunArgs(d DevnetConfig) []string {
	args := []string{
		"run", "-d",
		"--name", d.Name(),
	}
	// Emitted only when a platform is actually known, matching ERPCRunArgs: a
	// wrong --platform turns correct manifest selection into a hard failure,
	// so no value beats a guessed one.
	if p := strings.TrimSpace(d.Platform); p != "" {
		args = append(args, "--platform", p)
	}
	args = append(args,
		"-p", devnetPublishSpec(d.Bind(), d.HTTP(), devnetContainerHTTPPort),
		"-p", devnetPublishSpec(d.Bind(), d.WS(), devnetContainerWSPort),
		d.Image(),
	)
	// Everything from here on is reth's, not docker's: the image ref is the
	// boundary, and an option that lands on the wrong side of it is read by
	// the wrong program.
	return append(args, DevnetCommand(d)...)
}

// devnetPublishSpec renders one -p value: "<bind>:<hostPort>:<containerPort>".
// An IPv6 bind address must be bracketed or docker parses its colons as the
// field separators and rejects the whole mapping — the one piece of this
// string that is not obvious, and the reason it is a named function with its
// own test. (ops has an identical unexported helper; catalog cannot import ops
// without a cycle, and one four-line function is a better price than the
// dependency.)
func devnetPublishSpec(bind string, hostPort, containerPort int) string {
	if strings.Contains(bind, ":") && !strings.HasPrefix(bind, "[") {
		bind = "[" + bind + "]"
	}
	return bind + ":" + strconv.Itoa(hostPort) + ":" + strconv.Itoa(containerPort)
}

// Upstream expresses the devnet as a gateway upstream, so an eRPC gateway can
// front it exactly as it fronts a real node — which is the point of having a
// devnet at all: it makes the whole node+gateway lifecycle exercisable without
// a chain on disk.
//
// Local is true: this is a node the operator runs, so the gateway prefers it
// and treats every other upstream as fallback. RecentOnly is false — a devnet
// is an archive node by construction. It starts from its own genesis and has
// never pruned anything, so bounding it to the last 128 blocks would route
// historical calls away from the only node that can answer them.
func (d DevnetConfig) Upstream() GatewayUpstream {
	return GatewayUpstream{
		ID:         "devnet",
		Endpoint:   d.HTTPEndpoint(),
		Local:      true,
		RecentOnly: false,
	}
}

// GatewayForDevnet builds a single-chain gateway in front of a devnet, the
// devnet counterpart to GatewayForWire. No fallbacks are added, and there is
// no sensible way to add one: nothing on the public internet serves this
// chain, so a fallback upstream could only answer for a different chain.
func GatewayForDevnet(d DevnetConfig) GatewayConfig {
	return GatewayConfig{
		Networks: []GatewayNetwork{{
			ChainID:   d.ChainIDOrDefault(),
			Upstreams: []GatewayUpstream{d.Upstream()},
		}},
	}
}
