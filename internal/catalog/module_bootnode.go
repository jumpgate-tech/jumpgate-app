package catalog

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
)

// The bootnodes are box modules: a discovery-only service of their own,
// from binaries jumpgate already builds. Flags are checked against the real
// binaries' --help, recorded in testdata/bootnode-help.
//
// Each node key is the service's own secret: the binary generates it on
// first start in the data dir (reth at --p2p-secret-key, lighthouse under
// --datadir), the unit's UMask makes it 0600, and no record names it.
func init() {
	modules = append(modules,
		Module{
			ID:        "el-bootnode",
			Name:      "EL bootnode",
			Summary:   "Runs an execution-layer discovery bootnode (discv4 and discv5) other nodes can seed from.",
			Placement: PlaceBox,
			Chains:    []int{1, 369, 943, 560048},
			Resources: Resources{RAMBytes: 150 << 20, DiskBytes: 1 << 20},
			Unit:      elBootnodeUnit,
		},
		Module{
			ID:        "cl-bootnode",
			Name:      "CL bootnode",
			Summary:   "Runs a consensus-layer discv5 bootnode other beacon nodes can seed from.",
			Placement: PlaceBox,
			Chains:    []int{1, 369, 943, 560048},
			Resources: Resources{RAMBytes: 150 << 20, DiskBytes: 1 << 20},
			Unit:      clBootnodeUnit,
		},
	)
}

const (
	elBootnodePort = 30301
	clBootnodePort = 9200
)

// bootnodePort is b.Port, or def when unset, refused outside 1-65535.
func bootnodePort(b BoxModule, def int) (int, error) {
	if b.Port == 0 {
		return def, nil
	}
	if b.Port < 1 || b.Port > 65535 {
		return 0, fmt.Errorf("catalog: %s port %d is out of range", b.ID, b.Port)
	}
	return b.Port, nil
}

// bootnodeIP parses s as a literal IP; anything else (a name, shell text)
// is refused, so it can go into ExecStart unquoted.
func bootnodeIP(id, field, s string) (net.IP, error) {
	ip := net.ParseIP(s)
	if ip == nil {
		return nil, fmt.Errorf("catalog: %s %s %q is not an IP address", id, field, s)
	}
	return ip, nil
}

// bootnodeUnit renders the service with a 0077 umask, so the key the
// binary writes is 0600.
func bootnodeUnit(b BoxModule, execStart string) (string, error) {
	u, err := renderServiceUnit("module "+b.Instance(), execStart, CurrentNames, b.DataDir(), false)
	if err != nil {
		return "", err
	}
	group := "\nGroup=" + CurrentNames.Group + "\n"
	if !strings.Contains(u, group) {
		return "", errors.New("catalog: unit template has no Group= line")
	}
	return strings.Replace(u, group, group+"UMask=0077\n", 1), nil
}

// elBootnodeUnit runs reth p2p bootnode on the box.
//
// docs:covers bootnode-run
func elBootnodeUnit(b BoxModule) (BoxUnit, error) {
	port, err := bootnodePort(b, elBootnodePort)
	if err != nil {
		return BoxUnit{}, err
	}
	bind := "0.0.0.0"
	if b.BindAddr != "" {
		ip, err := bootnodeIP(b.ID, "BindAddr", b.BindAddr)
		if err != nil {
			return BoxUnit{}, err
		}
		bind = ip.String()
	}
	addr := net.JoinHostPort(bind, strconv.Itoa(port))
	cmd := fmt.Sprintf("reth p2p bootnode --addr %s --p2p-secret-key %s/bootnode.key --v5", addr, b.DataDir())
	if b.ExtIP != "" {
		ip, err := bootnodeIP(b.ID, "ExtIP", b.ExtIP)
		if err != nil {
			return BoxUnit{}, err
		}
		cmd += " --nat extip:" + ip.String()
	}
	u, err := bootnodeUnit(b, cmd)
	if err != nil {
		return BoxUnit{}, err
	}
	return BoxUnit{Unit: u, Client: "reth", Ports: []PortUse{{Port: port, Proto: "udp", Public: true}}}, nil
}

// clBootnodeUnit runs the lighthouse-family boot_node on the box.
//
// docs:covers bootnode-run
func clBootnodeUnit(b BoxModule) (BoxUnit, error) {
	port, err := bootnodePort(b, clBootnodePort)
	if err != nil {
		return BoxUnit{}, err
	}
	if b.ExtIP == "" {
		return BoxUnit{}, errors.New("catalog: the CL bootnode needs the box's public IP (ExtIP) for its ENR")
	}
	ip, err := bootnodeIP(b.ID, "ExtIP", b.ExtIP)
	if err != nil {
		return BoxUnit{}, err
	}
	if b.BindAddr != "" {
		return BoxUnit{}, fmt.Errorf("catalog: %s listens on all addresses; BindAddr is not supported", b.ID)
	}
	network, ok := lighthouseNetworkName[b.ChainID]
	if !ok {
		return BoxUnit{}, fmt.Errorf("catalog: no lighthouse --network for chain %d", b.ChainID)
	}
	bin := "lighthouse"
	if b.ChainID == 369 || b.ChainID == 943 {
		bin = "lighthouse-pulse"
	}
	cmd := fmt.Sprintf("%s boot_node --network %s --port %d --enr-address %s --datadir %s", bin, network, port, ip.String(), b.DataDir())
	u, err := bootnodeUnit(b, cmd)
	if err != nil {
		return BoxUnit{}, err
	}
	return BoxUnit{Unit: u, Client: bin, Ports: []PortUse{{Port: port, Proto: "udp", Public: true}}}, nil
}
