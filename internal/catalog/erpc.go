package catalog

import (
	"fmt"
	"path"
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

// RenderERPCConfig renders the erpc.yaml for a node's own gateway: the node as
// the preferred upstream, with ERPCUpstreams behind it as fallbacks. Pure
// string rendering; the caller (setup's wire step) writes it. Assumes
// ERPCEnabled — callers gate on that.
//
// This is the single-chain case of RenderGatewayConfig, and delegates to it so
// there is one template rather than two that can drift. Unlike the general
// gateway, the chain must be one this app knows how to run a node on — the
// upstream being configured is that node.
func RenderERPCConfig(w WireConfig) (string, error) {
	if _, ok := NetworkByChainID(w.ChainID); !ok {
		return "", fmt.Errorf("catalog: erpc: unknown chain id %d", w.ChainID)
	}
	return RenderGatewayConfig(GatewayForWire(w))
}

// RenderERPCUnit renders the systemd unit for the eRPC gateway, reusing the
// same hardened/de-rooted template as the exec/beacon units. The unit runs
// against the node's own erpc.yaml (DataDir/erpc.yaml).
func RenderERPCUnit(w WireConfig) (string, error) {
	return RenderERPCUnitAt(w, w.ERPCConfigPath())
}

// RenderERPCUnitAt is RenderERPCUnit with the config path stated explicitly.
//
// It exists because a fleet-wide gateway is identified by its own id, not by
// the machine it sits on, so several gateways can share a target — and each
// needs its OWN erpc.yaml. Deriving the path from the WireConfig (which knows
// only a data dir) would give them all the same file, and each provision
// would rewrite the previous gateway's chains under it.
func RenderERPCUnitAt(w WireConfig, configPath string) (string, error) {
	execStart := fmt.Sprintf("erpc --config %s", configPath)
	return renderUnit("RPC gateway (eRPC)", execStart, w, w.ERPCHTTP() < 1024)
}
