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

// GatewayUpstream is a single RPC endpoint the gateway can route to.
type GatewayUpstream struct {
	// ID is unique across the whole config. Left empty, a stable id is
	// generated from the chain and position.
	ID string
	// Endpoint is an http(s):// or ws(s):// URL. eRPC infers WebSocket
	// capability from the scheme — there is no separate flag — and a ws
	// upstream also serves ordinary request/response calls.
	Endpoint string
	// Local marks an upstream the operator runs. Local upstreams are
	// preferred; every other upstream is deprioritised so it is used only
	// when the local one cannot serve a request or is down.
	Local bool
	// RecentOnly bounds this upstream to recent history, for a full/pruned
	// node that cannot answer historical state.
	RecentOnly bool
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
}

// gatewayConfigTemplate renders erpc.yaml. Hand-rendered (no YAML dependency,
// matching the systemd unit template): the structure is fixed and only the
// network and upstream lists vary.
//
// Shape verified against eRPC's own config structs on the valve-ws branch
// rather than its erpc.dist.yaml, which is stale. Upstreams are a flat list at
// project level, each tagged with the chainId it serves — they are not nested
// under their network.
const gatewayConfigTemplate = `logLevel: warn
server:
  httpHostV4: {{.Host}}
  httpPortV4: {{.Port}}
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

	vars := gatewayVars{
		Host:              strconv.Quote(g.Bind()),
		Port:              g.HTTP(),
		ProjectID:         g.ProjectIDOrDefault(),
		RecentBlockWindow: recentBlockWindow,
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
