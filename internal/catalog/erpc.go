package catalog

import (
	"bytes"
	"fmt"
	"path"
	"text/template"
)

// defaultERPCPort is eRPC's default HTTP listen port (matching eRPC's own
// httpPortV4 default).
const defaultERPCPort = 4000

// ERPCHTTP resolves eRPC's HTTP listen port (0 → 4000).
func (w WireConfig) ERPCHTTP() int {
	if w.ERPCPort == 0 {
		return defaultERPCPort
	}
	return w.ERPCPort
}

// ERPCBind resolves eRPC's listen host (empty → loopback). eRPC's listen
// address is the front door meant to be exposed (e.g. bound to a Tailscale
// IP), distinct from the raw node's RPC bind.
func (w WireConfig) ERPCBind() string {
	if w.ERPCBindAddr == "" {
		return "127.0.0.1"
	}
	return w.ERPCBindAddr
}

// ERPCConfigPath is where the rendered erpc.yaml lives on the target (inside
// the data dir, so the wire step's chown -R covers it).
func (w WireConfig) ERPCConfigPath() string {
	return path.Join(w.DataDir, "erpc.yaml")
}

// erpcConfigTemplate renders erpc.yaml. Hand-rendered (no YAML dep, matching
// the systemd unit template) — the structure is fixed; only the fallback
// upstream list is dynamic. Upstream[0] is always the local node: it serves
// recent state (latestBlockMinus:128) when the node is a full/pruned node,
// or unbounded history (lower: null) when it's an archive. Each fallback is
// deprioritized (tier:fallback + scoreMultipliers overall 0.2) so it's used
// only when the local node can't serve a request (historical state) or is
// down.
const erpcConfigTemplate = `logLevel: warn
server:
  httpHostV4: "{{.Host}}"
  httpPortV4: {{.Port}}
projects:
  - id: main
    upstreams:
      - id: local-node
        endpoint: {{.LocalEndpoint}}
        evm:
          chainId: {{.ChainID}}
          blockAvailability:
            lower:{{if .LocalRecentOnly}}
              latestBlockMinus: 128{{else}} null{{end}}
{{- range .Fallbacks}}
      - id: {{.ID}}
        endpoint: {{.Endpoint}}
        evm:
          chainId: {{$.ChainID}}
        tags:
          - tier:fallback
        routing:
          scoreMultipliers:
            - overall: 0.2
{{- end}}
`

var erpcConfigTmpl = template.Must(template.New("erpc").Parse(erpcConfigTemplate))

type erpcFallback struct {
	ID       string
	Endpoint string
}

type erpcVars struct {
	Host            string
	Port            int
	ChainID         int
	LocalEndpoint   string
	LocalRecentOnly bool
	Fallbacks       []erpcFallback
}

// RenderERPCConfig renders the erpc.yaml for w. Pure string rendering; the
// caller (setup's wire step) writes it. Assumes ERPCEnabled — callers gate
// on that.
func RenderERPCConfig(w WireConfig) (string, error) {
	if _, ok := NetworkByChainID(w.ChainID); !ok {
		return "", fmt.Errorf("catalog: erpc: unknown chain id %d", w.ChainID)
	}
	fallbacks := make([]erpcFallback, 0, len(w.ERPCUpstreams))
	for i, url := range w.ERPCUpstreams {
		fallbacks = append(fallbacks, erpcFallback{ID: fmt.Sprintf("fallback-%d", i+1), Endpoint: url})
	}
	vars := erpcVars{
		Host:            w.ERPCBind(),
		Port:            w.ERPCHTTP(),
		ChainID:         w.ChainID,
		LocalEndpoint:   fmt.Sprintf("http://%s:%d", w.RPCBind(), w.ExecHTTP()),
		LocalRecentOnly: !w.Archive,
		Fallbacks:       fallbacks,
	}
	var buf bytes.Buffer
	if err := erpcConfigTmpl.Execute(&buf, vars); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// RenderERPCUnit renders the systemd unit for the eRPC gateway, reusing the
// same hardened/de-rooted template as the exec/beacon units.
func RenderERPCUnit(w WireConfig) (string, error) {
	execStart := fmt.Sprintf("erpc --config %s", w.ERPCConfigPath())
	return renderUnit("RPC gateway (eRPC)", execStart, w, w.ERPCHTTP() < 1024)
}
