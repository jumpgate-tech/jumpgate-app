package catalog

import (
	"encoding/json"
	"fmt"
	"path"
	"strconv"
)

// BoxModule is one box-placed module recorded on a target (config.Target.
// Modules). Setup writes it as config.json (root:group 0640) and its public
// BoxSidecar as module.json (0644) into ConfigDir, so a gate or an SSH probe
// can list a box's modules on its own.
type BoxModule struct {
	ID        string   `json:"id"`
	ChainID   int      `json:"chainId,omitempty"`   // 0 for a chain-agnostic module (watch)
	Port      int      `json:"port,omitempty"`      // 0 = the module's default
	BindAddr  string   `json:"bindAddr,omitempty"`  // "" = the module's default
	ExtIP     string   `json:"extIp,omitempty"`     // bootnodes: the public IP advertised
	Upstreams []string `json:"upstreams,omitempty"` // checkpointz: beacon URLs; watch: endpoints
	Webhook   string   `json:"webhook,omitempty"`   // watch only
}

// Instance is "<id>-<chainId>", or "<id>" for a chain-agnostic module.
func (b BoxModule) Instance() string {
	if b.ChainID == 0 {
		return b.ID
	}
	return b.ID + "-" + strconv.Itoa(b.ChainID)
}

// UnitName is the module's own systemd unit.
func (b BoxModule) UnitName() string { return "jumpgate-mod-" + b.Instance() + ".service" }

// BoxModulesConfigRoot holds every box module's root-owned config, outside
// any path the service account can write.
const BoxModulesConfigRoot = "/var/lib/jumpgate/modules"

// ConfigDir is the module's root-owned config dir (root:root 0755): its
// module.json and every file its unit reads. The service cannot rewrite the
// record a prober reads back, nor swap a path in it.
func (b BoxModule) ConfigDir() string { return BoxModulesConfigRoot + "/" + b.Instance() }

// SidecarPath is the module's module.json, in ConfigDir: a world-readable
// BoxSidecar for discovery.
func (b BoxModule) SidecarPath() string { return b.ConfigDir() + "/module.json" }

// ConfigPath is the module's full record, config.json in ConfigDir,
// readable by root and the service's group only: it may hold a webhook URL
// or upstreams.
func (b BoxModule) ConfigPath() string { return b.ConfigDir() + "/config.json" }

// NodePath is the copy of the box's node config (WireConfig, secrets
// stripped) that setup stages for a module that watches the node, in
// ConfigDir: the gate's own node.json sits in a root-only dir.
func (b BoxModule) NodePath() string { return b.ConfigDir() + "/node.json" }

// PublicWire is w without its secrets (the snapshot key, and eRPC
// upstream URLs, which can carry API keys), for a module to read.
func PublicWire(w WireConfig) WireConfig {
	w.SnapshotKey = ""
	w.ERPCUpstreams = nil
	return w
}

// BoxSidecar is the public part of a box module record, all a gate or an
// SSH probe needs to find and read the module. It never holds endpoints,
// webhooks or secrets.
type BoxSidecar struct {
	ID      string `json:"id"`
	ChainID int    `json:"chainId,omitempty"`
	Unit    string `json:"unit"`
	Port    int    `json:"port,omitempty"`
	Proto   string `json:"proto,omitempty"`
	Public  bool   `json:"public,omitempty"` // the port is meant to be reachable from the internet
	Bind    string `json:"bind,omitempty"`   // the address it listens on, when not the module's default
}

// Module is the record the sidecar names, for validation and probing.
func (s BoxSidecar) Module() BoxModule {
	return BoxModule{ID: s.ID, ChainID: s.ChainID, Port: s.Port, BindAddr: s.Bind}
}

// DataDir is the module's service-owned state under the current account's
// root: the unit's working and writable path. Root only creates it.
func (b BoxModule) DataDir() string { return CurrentNames.DataRoot + "/modules/" + b.Instance() }

// ValidateBoxModule checks a record, such as one read back from a box,
// before anything is built from it: a registered box module that applies to
// its chain, and the chain a known network unless it is 0 (chain-agnostic).
func ValidateBoxModule(b BoxModule) error {
	if b.ChainID != 0 {
		if _, ok := NetworkByChainID(b.ChainID); !ok {
			return fmt.Errorf("chain %d is not a known network", b.ChainID)
		}
	}
	_, err := boxModule(b)
	return err
}

// PortUse is one port a service listens on. Proto is "tcp" or "udp".
type PortUse struct {
	Port   int
	Proto  string
	Public bool
}

// BoxUnit is what a box module renders to.
type BoxUnit struct {
	Unit   string            // rendered systemd unit
	Files  map[string]string // path in ConfigDir → content, written root:jumpgate-node 0640; module.json root:root 0644
	Client string            // catalog client id to install, "" = the box's gate binary
	Ports  []PortUse
}

// boxModule resolves b to its registered box module and checks it applies.
func boxModule(b BoxModule) (Module, error) {
	m, ok := ModuleByID(b.ID)
	if !ok {
		return Module{}, fmt.Errorf("%s is not a known module", b.ID)
	}
	if m.Placement != PlaceBox {
		return Module{}, fmt.Errorf("%s is not a box module", b.ID)
	}
	if ok, reason := m.Applies(b.ChainID, ""); !ok {
		if reason == "" {
			reason = fmt.Sprintf("%s does not run on chain %d", b.ID, b.ChainID)
		}
		return Module{}, fmt.Errorf("%s", reason)
	}
	return m, nil
}

// RenderBoxModule validates b, renders its unit and adds its public
// module.json sidecar and its full config.json to Files.
func RenderBoxModule(b BoxModule) (BoxUnit, error) {
	if err := ValidateBoxModule(b); err != nil {
		return BoxUnit{}, err
	}
	m, _ := ModuleByID(b.ID)
	u, err := m.Unit(b)
	if err != nil {
		return BoxUnit{}, err
	}
	full, err := json.Marshal(b)
	if err != nil {
		return BoxUnit{}, err
	}
	pub := BoxSidecar{ID: b.ID, ChainID: b.ChainID, Unit: b.UnitName(), Port: b.Port, Bind: b.BindAddr}
	if len(u.Ports) > 0 {
		pub.Port, pub.Proto, pub.Public = u.Ports[0].Port, u.Ports[0].Proto, u.Ports[0].Public
	}
	side, err := json.Marshal(pub)
	if err != nil {
		return BoxUnit{}, err
	}
	if u.Files == nil {
		u.Files = map[string]string{}
	}
	for f := range u.Files {
		if path.Dir(f) != b.ConfigDir() {
			return BoxUnit{}, fmt.Errorf("%s: file %s is not in %s", b.ID, f, b.ConfigDir())
		}
	}
	u.Files[b.SidecarPath()] = string(side)
	u.Files[b.ConfigPath()] = string(full)
	return u, nil
}

// portKey is one proto/port pair.
type portKey struct {
	proto string
	port  int
}

// nodePorts are the ports w's two clients use, each named by its owner.
func nodePorts(w *WireConfig) map[portKey]string {
	const exec, beacon = "the execution client", "the beacon client"
	out := map[portKey]string{}
	add := func(owner string, port int, protos ...string) {
		for _, p := range protos {
			if _, taken := out[portKey{p, port}]; !taken {
				out[portKey{p, port}] = owner
			}
		}
	}
	add(exec, w.ExecHTTP(), "tcp")
	add(exec, w.ExecP2P(), "tcp", "udp")
	add(exec, 8551, "tcp")
	add(beacon, w.BeaconHTTP(), "tcp")
	switch w.BeaconID {
	case "lighthouse", "lighthouse-pulse":
		add(beacon, 9000, "tcp", "udp")
		add(beacon, 9001, "tcp", "udp")
	case "prysm-pulse":
		// prysm's p2p defaults, which its unit does not override: tcp 13000,
		// udp 12000 (the pair ops checks in beaconP2PPorts).
		add(beacon, 13000, "tcp")
		add(beacon, 12000, "udp")
	}
	if w.ERPCEnabled {
		// The node's own eRPC gateway listens on ERPCHTTP (0 → 4000).
		add("the RPC gateway", w.ERPCHTTP(), "tcp")
	}
	return out
}

// ValidateBoxModules refuses a box module list that cannot run together on
// one box: a non-box or inapplicable module, an instance twice, or two
// services wanting one port. w is the box's node, nil when it has none.
func ValidateBoxModules(mods []BoxModule, w *WireConfig) error {
	var node map[portKey]string
	if w != nil {
		node = nodePorts(w)
	}
	seen := map[string]bool{}
	taken := map[portKey]string{}
	for _, b := range mods {
		inst := b.Instance()
		if seen[inst] {
			return fmt.Errorf("%s is on this box twice", inst)
		}
		seen[inst] = true
		u, err := RenderBoxModule(b)
		if err != nil {
			return err
		}
		for _, p := range u.Ports {
			k := portKey{p.Proto, p.Port}
			if owner, ok := node[k]; ok {
				return fmt.Errorf("%s wants %s %d, which %s already uses", inst, p.Proto, p.Port, owner)
			}
			if other, ok := taken[k]; ok {
				if other == inst {
					return fmt.Errorf("%s wants %s %d twice", inst, p.Proto, p.Port)
				}
				return fmt.Errorf("%s and %s both want %s %d", other, inst, p.Proto, p.Port)
			}
			taken[k] = inst
		}
	}
	return nil
}
