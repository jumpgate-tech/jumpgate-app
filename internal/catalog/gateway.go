package catalog

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"text/template"
)

// A gateway is one eRPC instance fronting many chains at once. eRPC addresses
// a chain by URL path — http://host:port/<project>/evm/<chainId> — so a single
// process, a single port and a single config serve every chain the operator
// cares about. WebSocket rides the *same* path and port with a ws:// scheme;
// there is no second listener to configure.
//
// That is why a gateway is modelled here rather than as a mode of WireConfig:
// a WireConfig describes one chain's node, and a gateway deliberately spans
// chains. A gateway with no local upstreams is the "I can't host a node, but I
// still want my own RPC" case, and needs no separate representation — it is
// simply a config whose networks happen to carry only fallback upstreams.

// defaultProjectID is the eRPC project id used when none is supplied. It shows
// up in every request path, so it is deliberately short.
const defaultProjectID = "main"

// recentBlockWindow is how far back a non-archive upstream is trusted for
// historical state. Requests below the bound are routed elsewhere instead of
// failing against a pruned node.
const recentBlockWindow = 128

// The kinds of thing an upstream can BE. The distinction exists because two
// of them are not addresses at all — they are references to something this
// app manages, whose address is derived when the config is rendered.
//
// WHY that indirection is not over-engineering: a managed node's RPC address
// is a property of the node, and the operator can change it (RPCBindAddr, a
// devnet's HTTPPort) on the screen that owns the node. Storing a copy of the
// URL in the gateway means the gateway silently keeps pointing at the old
// address — the gateway stays "healthy" (its container runs, its path
// resolves) while every call goes to a port nothing is listening on. Storing
// the REFERENCE means the address is re-derived on every render and cannot
// go stale.
const (
	// UpstreamManagedNode is a chain client valve-node-app set up on one of
	// its targets (config.Target.Wire). TargetID names the machine.
	UpstreamManagedNode = "managed-node"
	// UpstreamManagedDevnet is a devnet container valve-node-app runs on one
	// of its targets (config.Target.Devnet). TargetID names the machine.
	UpstreamManagedDevnet = "managed-devnet"
	// UpstreamExternal is a URL nobody here manages: a public mainnet or
	// testnet endpoint, a chainlist-discovered one, a provider. Endpoint is
	// the whole of it.
	UpstreamExternal = "external"
)

// GatewayUpstream is a single RPC endpoint the gateway can route to.
type GatewayUpstream struct {
	// ID is unique across the whole config. Left empty, a stable id is
	// generated from the chain and position.
	ID string

	// Kind is one of the Upstream* constants ("" → UpstreamExternal, which is
	// what every config written before upstreams had identity meant).
	Kind string

	// TargetID names the managed machine, for the two managed kinds. It is
	// ignored for UpstreamExternal.
	TargetID string

	// Endpoint is an http(s):// or ws(s):// URL. eRPC infers WebSocket
	// capability from the scheme — there is no separate flag — and a ws
	// upstream also serves ordinary request/response calls.
	//
	// For UpstreamExternal this is the operator's own value and is stored as
	// given. For the managed kinds it is DERIVED — the resolver fills it in
	// from the referenced target immediately before rendering, and whatever
	// is stored is overwritten. catalog cannot do that resolution itself (it
	// would have to import config, which imports catalog), so it is the
	// caller's job; RenderGatewayConfig simply refuses an upstream that
	// arrives with no endpoint, which is what makes a missed resolution a
	// loud failure rather than a gateway that renders and serves nothing.
	Endpoint string

	// Local marks an upstream the operator runs. Local upstreams are
	// preferred; every other upstream is deprioritised so it is used only
	// when the local one cannot serve a request or is down.
	Local bool
	// RecentOnly bounds this upstream to recent history, for a full/pruned
	// node that cannot answer historical state.
	RecentOnly bool
}

// KindOrDefault resolves the kind ("" → UpstreamExternal).
func (u GatewayUpstream) KindOrDefault() string {
	if u.Kind == "" {
		return UpstreamExternal
	}
	return u.Kind
}

// Managed reports whether this upstream's endpoint has to be derived from a
// target rather than read from Endpoint.
func (u GatewayUpstream) Managed() bool {
	k := u.KindOrDefault()
	return k == UpstreamManagedNode || k == UpstreamManagedDevnet
}

// GatewayNetwork is one chain served by the gateway.
type GatewayNetwork struct {
	ChainID   int
	Upstreams []GatewayUpstream
}

// GatewayConfig is a whole eRPC instance.
type GatewayConfig struct {
	// ProjectID appears in every request path ("" → "main").
	ProjectID string
	// BindAddr is the listen host ("" → 127.0.0.1). Unlike a node's RPC
	// bind, this is the front door meant to be exposed.
	BindAddr string
	// Port is the listen port (0 → 4000).
	Port int
	// Networks is the set of chains served. Order is preserved so the
	// rendered config stays stable across runs.
	Networks []GatewayNetwork

	// TLS is the HTTPS front for this gateway, nil when there is none.
	//
	// It lives on the gateway config rather than beside it because the rendered
	// erpc.yaml DEPENDS on it: a fronted gateway must not offer gzip (see
	// MustDisableGzipBehindProxy), and a Fronted flag stored somewhere else
	// would be a second source of truth able to disagree with the container
	// that is actually in front.
	TLS *GatewayTLS
}

// Fronted reports whether something terminates TLS in front of this gateway.
// Derived rather than stored, so it cannot drift from the TLS settings.
func (g GatewayConfig) Fronted() bool { return g.TLS.On() }

// gatewayConfigTemplate renders erpc.yaml. Hand-rendered (no YAML dependency,
// matching the systemd unit template): the structure is fixed and only the
// network and upstream lists vary.
//
// Shape verified against eRPC's own config structs on the valve-ws branch
// rather than its erpc.dist.yaml, which is stale. Upstreams are a flat list at
// project level, each tagged with the chainId it serves — they are not nested
// under their network.
// enableGzip is rendered only when the gateway is FRONTED, and only ever as
// false. See MustDisableGzipBehindProxy for the measurement: eRPC's WebSocket
// upgrade returns HTTP 500 ("websocket: response does not implement
// http.Hijacker") whenever the client advertises Accept-Encoding: gzip, and
// every reverse proxy adds that header. An unfronted gateway keeps eRPC's own
// default, because there the compression is free and nothing is inserting the
// header.
const gatewayConfigTemplate = `logLevel: warn
server:
  httpHostV4: {{.Host}}
  httpPortV4: {{.Port}}
{{- if .Fronted}}
  # This gateway sits behind a reverse proxy, and every reverse proxy adds
  # Accept-Encoding: gzip to the requests it forwards. eRPC's gzip response
  # writer does not implement http.Hijacker, so a WebSocket upgrade carrying
  # that header fails with HTTP 500 and eth_subscribe stops working entirely.
  # Losing response compression is the cheaper half of that trade.
  enableGzip: false
{{- end}}
projects:
  - id: {{.ProjectID}}
    networks:
{{- range .Networks}}
      - architecture: evm
        evm:
          chainId: {{.ChainID}}
{{- end}}
    upstreams:
{{- range .Upstreams}}
      - id: {{.ID}}
        type: evm
        endpoint: {{.Endpoint}}
        evm:
          chainId: {{.ChainID}}
{{- if .RecentOnly}}
          blockAvailability:
            lower:
              latestBlockMinus: {{$.RecentBlockWindow}}
{{- end}}
{{- if .Fallback}}
        tags:
          - tier:fallback
        routing:
          scoreMultipliers:
            - overall: 0.2
{{- end}}
{{- end}}
`

var gatewayConfigTmpl = template.Must(template.New("gateway").Parse(gatewayConfigTemplate))

type gatewayNetworkVars struct {
	ChainID int
}

type gatewayUpstreamVars struct {
	ID         string
	Endpoint   string
	ChainID    int
	RecentOnly bool
	Fallback   bool
}

type gatewayVars struct {
	Host              string
	Port              int
	ProjectID         string
	RecentBlockWindow int
	Fronted           bool
	Networks          []gatewayNetworkVars
	Upstreams         []gatewayUpstreamVars
}

// ProjectIDOrDefault resolves the project id ("" → "main").
func (g GatewayConfig) ProjectIDOrDefault() string {
	if g.ProjectID == "" {
		return defaultProjectID
	}
	return g.ProjectID
}

// Bind resolves the listen host ("" → loopback).
func (g GatewayConfig) Bind() string {
	if g.BindAddr == "" {
		return "127.0.0.1"
	}
	return g.BindAddr
}

// HTTP resolves the listen port (0 → 4000).
func (g GatewayConfig) HTTP() int {
	if g.Port == 0 {
		return defaultERPCPort
	}
	return g.Port
}

// PathFor is the request path for a chain — the URL callers actually use.
// The same path serves WebSocket with a ws:// or wss:// scheme.
func (g GatewayConfig) PathFor(chainID int) string {
	return fmt.Sprintf("/%s/evm/%d", g.ProjectIDOrDefault(), chainID)
}

// RenderGatewayConfig renders g to erpc.yaml. Pure string rendering; the
// caller writes it.
//
// Chain ids are not checked against the catalog's own networks: a gateway is
// useful in front of any EVM chain, including ones this app cannot itself set
// up a node for, so restricting it to the three chains with node support would
// be an arbitrary limit.
func RenderGatewayConfig(g GatewayConfig) (string, error) {
	if len(g.Networks) == 0 {
		return "", fmt.Errorf("catalog: gateway: no networks configured")
	}

	if err := g.TLS.ValidateSettings(); err != nil {
		return "", err
	}

	vars := gatewayVars{
		Host:              strconv.Quote(g.Bind()),
		Port:              g.HTTP(),
		ProjectID:         g.ProjectIDOrDefault(),
		RecentBlockWindow: recentBlockWindow,
		Fronted:           g.Fronted(),
	}

	seenChains := make(map[int]bool, len(g.Networks))
	seenIDs := make(map[string]bool)

	for _, n := range g.Networks {
		if n.ChainID <= 0 {
			return "", fmt.Errorf("catalog: gateway: invalid chain id %d", n.ChainID)
		}
		if seenChains[n.ChainID] {
			return "", fmt.Errorf("catalog: gateway: chain %d listed twice", n.ChainID)
		}
		seenChains[n.ChainID] = true

		if len(n.Upstreams) == 0 {
			return "", fmt.Errorf("catalog: gateway: chain %d has no upstreams", n.ChainID)
		}
		vars.Networks = append(vars.Networks, gatewayNetworkVars{ChainID: n.ChainID})

		for i, u := range n.Upstreams {
			endpoint := strings.TrimSpace(u.Endpoint)
			if endpoint == "" {
				return "", fmt.Errorf("catalog: gateway: chain %d upstream %d has no endpoint", n.ChainID, i+1)
			}
			if !validUpstreamScheme(endpoint) {
				return "", fmt.Errorf("catalog: gateway: chain %d upstream %q must be http(s):// or ws(s)://", n.ChainID, endpoint)
			}

			id := u.ID
			if id == "" {
				id = generatedUpstreamID(n.ChainID, u.Local, i+1)
			}
			// eRPC keys upstreams by id; a duplicate would silently
			// shadow rather than error, so catch it here.
			if seenIDs[id] {
				return "", fmt.Errorf("catalog: gateway: duplicate upstream id %q", id)
			}
			seenIDs[id] = true

			vars.Upstreams = append(vars.Upstreams, gatewayUpstreamVars{
				ID:         id,
				Endpoint:   strconv.Quote(endpoint),
				ChainID:    n.ChainID,
				RecentOnly: u.RecentOnly,
				Fallback:   !u.Local,
			})
		}
	}

	var buf bytes.Buffer
	if err := gatewayConfigTmpl.Execute(&buf, vars); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// validUpstreamScheme reports whether endpoint is a scheme eRPC can dial.
func validUpstreamScheme(endpoint string) bool {
	for _, p := range []string{"http://", "https://", "ws://", "wss://"} {
		if strings.HasPrefix(endpoint, p) {
			return true
		}
	}
	return false
}

// generatedUpstreamID builds a stable id for an upstream the caller did not
// name. Position is included because a chain can carry several upstreams of
// the same kind.
func generatedUpstreamID(chainID int, local bool, pos int) string {
	kind := "fallback"
	if local {
		kind = "local"
	}
	return fmt.Sprintf("%d-%s-%d", chainID, kind, pos)
}

// GatewayForWire builds a single-chain gateway from a node's WireConfig: the
// node itself as the preferred upstream, with ERPCUpstreams behind it as
// fallbacks. This is the "gateway in front of my node" case.
func GatewayForWire(w WireConfig) GatewayConfig {
	ups := []GatewayUpstream{{
		ID:         "local-node",
		Endpoint:   fmt.Sprintf("http://%s:%d", w.RPCBind(), w.ExecHTTP()),
		Local:      true,
		RecentOnly: !w.Archive,
	}}
	for i, url := range w.ERPCUpstreams {
		ups = append(ups, GatewayUpstream{
			ID:       fmt.Sprintf("fallback-%d", i+1),
			Endpoint: url,
		})
	}
	return GatewayConfig{
		BindAddr: w.ERPCBind(),
		Port:     w.ERPCHTTP(),
		Networks: []GatewayNetwork{{ChainID: w.ChainID, Upstreams: ups}},
	}
}
