package catalog

import (
	"fmt"
	"path"
	"strings"
)

// Names ids, as WireConfig.Names stores them. The legacy id is the empty
// string so every wire saved before sub-project 2 reads as legacy unchanged.
const (
	NamesLegacy  = ""
	NamesCurrent = "jumpgate-node"
)

// NodeNames are a node's on-box names: its service account, its two units and
// the default root of its data. A box uses one set or the other, never a mix:
// node.migrate-names moves a box from LegacyNames to CurrentNames in one job.
type NodeNames struct {
	ID       string // NamesLegacy or NamesCurrent
	Prefix   string // unit names and unit descriptions
	User     string
	Group    string
	DataRoot string
}

var (
	LegacyNames  = NodeNames{ID: NamesLegacy, Prefix: "valve-node-app", User: "valve-node-app", Group: "valve-node-app", DataRoot: "/var/lib/valve-node-app"}
	CurrentNames = NodeNames{ID: NamesCurrent, Prefix: "jumpgate-node", User: "jumpgate-node", Group: "jumpgate-node", DataRoot: "/var/lib/jumpgate-node"}
)

func (n NodeNames) ExecUnit() string   { return n.Prefix + "-exec.service" }
func (n NodeNames) BeaconUnit() string { return n.Prefix + "-beacon.service" }
func (n NodeNames) Units() []string    { return []string{n.ExecUnit(), n.BeaconUnit()} }

// UnitPath is where setup writes a node unit.
func UnitPath(unit string) string { return "/etc/systemd/system/" + unit }

// NamesByID resolves a stored id. An unknown id is an error, never a guess:
// acting on the wrong account would stop or chown the wrong things.
func NamesByID(id string) (NodeNames, error) {
	switch id {
	case NamesLegacy:
		return LegacyNames, nil
	case NamesCurrent:
		return CurrentNames, nil
	}
	return NodeNames{}, fmt.Errorf("catalog: unknown node names %q (want %q or %q)", id, NamesLegacy, NamesCurrent)
}

// NodeNames is w's names. An unknown id reads as legacy here so a renderer
// never panics; ValidateNames is where it is refused (setup.Plan, the agent's
// node.json load).
func (w WireConfig) NodeNames() NodeNames {
	n, err := NamesByID(w.Names)
	if err != nil {
		return LegacyNames
	}
	return n
}

// ValidateNames refuses a wire whose names id is unknown.
func ValidateNames(w WireConfig) error {
	_, err := NamesByID(w.Names)
	return err
}

// DefaultDataDirFor is where a chain's data lives by default under a name set.
func DefaultDataDirFor(n NodeNames, chainID int) string {
	return fmt.Sprintf("%s/%d", n.DataRoot, chainID)
}

// UnderRoot reports whether dir is root itself or inside it. Paths on a box
// are POSIX, so this uses package path, never filepath.
func UnderRoot(dir, root string) bool {
	dir = path.Clean(dir)
	return dir == root || strings.HasPrefix(dir, root+"/")
}

// Rebase moves dir from one root to another. ok is false, and dir comes back
// unchanged, when dir is not under from.
func Rebase(dir, from, to string) (string, bool) {
	if !UnderRoot(dir, from) {
		return dir, false
	}
	return to + strings.TrimPrefix(path.Clean(dir), from), true
}

// Completion markers live directly in a node's data dir. Setup writes each one
// only after the work it vouches for finished, so a rerun (and adopting a
// durable job, which is a rerun) can trust it where a looser probe cannot.
const (
	SnapshotMarker = ".jumpgate-snapshot-complete" // the snapshot restore finished
	OwnerMarker    = ".jumpgate-owner"             // chown -R finished; holds the owning account's name
)
