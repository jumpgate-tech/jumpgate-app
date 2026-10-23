package catalog

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// BoxModule is one box-placed module recorded on a target (config.Target.
// Modules). Setup also writes it, as module.json, into the module's data
// directory so a gate or an SSH probe can list a box's modules on its own.
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

// DataDir is the module's data directory under the current account's root.
func (b BoxModule) DataDir() string { return CurrentNames.DataRoot + "/modules/" + b.Instance() }

// PortUse is one port a service listens on. Proto is "tcp" or "udp".
type PortUse struct {
	Port   int
	Proto  string
	Public bool
}

// BoxUnit is what a box module renders to.
type BoxUnit struct {
	Unit   string            // rendered systemd unit
	Files  map[string]string // absolute path → content, written 0640 owner jumpgate-node
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

// RenderBoxModule validates b, renders its unit and adds its module.json
// sidecar to Files.
func RenderBoxModule(b BoxModule) (BoxUnit, error) {
	m, err := boxModule(b)
	if err != nil {
		return BoxUnit{}, err
	}
	u, err := m.Unit(b)
	if err != nil {
		return BoxUnit{}, err
	}
	side, err := json.Marshal(b)
	if err != nil {
		return BoxUnit{}, err
	}
	if u.Files == nil {
		u.Files = map[string]string{}
	}
	u.Files[b.DataDir()+"/module.json"] = string(side)
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
			if other, ok := taken[k]; ok && other != inst {
				return fmt.Errorf("%s and %s both want %s %d", other, inst, p.Proto, p.Port)
			}
			taken[k] = inst
		}
	}
	return nil
}
