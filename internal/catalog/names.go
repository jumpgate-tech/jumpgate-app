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

// MarkersDir holds the completion markers. It is root-owned and 0700, outside
// every tree the node's service account can write: root writes markers as it
// works, and a marker in a service-writable directory could be forged to skip
// work, or be a planted symlink that root would then overwrite.
const MarkersDir = "/var/lib/jumpgate/markers"

// SnapshotMarkerPath is the marker that the snapshot restore for w's chain
// finished. The name comes from the chain id alone, never from a path segment
// the service account controls.
func SnapshotMarkerPath(w WireConfig) string {
	return fmt.Sprintf("%s/%d.snapshot", MarkersDir, w.ChainID)
}

// OwnerMarkerPath is the marker that chown -R of w's data dir finished; it
// holds the owning account's name.
func OwnerMarkerPath(w WireConfig) string {
	return fmt.Sprintf("%s/%d.owner", MarkersDir, w.ChainID)
}

// SnapshotMarkerCommand writes the snapshot marker for w: it records which
// data dir the finished restore populated. Setup writes it when a restore
// ends; node.migrate-names rewrites it when the data dir moves.
func SnapshotMarkerCommand(w WireConfig) string {
	return writeMarkerCommand(SnapshotMarkerPath(w), w.DataDir)
}

// OwnerMarkerCommand writes the owner marker for w: it records which account
// chown -R of w's data dir finished for. Setup writes it after the walk;
// node.migrate-names rewrites it when it renames that account in place.
func OwnerMarkerCommand(w WireConfig) string {
	return writeMarkerCommand(OwnerMarkerPath(w), w.NodeNames().User)
}

// writeMarkerCommand writes value to target, a marker in MarkersDir. The
// directory is root-only, so nothing in it can be planted by the service
// account; the value goes to a fresh temp file and mv -f renames it into
// place, which replaces whatever is at the target (a symlink included)
// rather than writing through it. A marker is never chowned: it stays
// root's. It is unexported so every marker is written for a path the two
// commands above derive from the chain id, never for a caller's path.
func writeMarkerCommand(target, value string) string {
	dir := shQuote(MarkersDir)
	return fmt.Sprintf(`install -d -m 0755 %s && umask 077 && mkdir -p %s && chmod 0700 %s && t=$(mktemp %s) && printf '%%s\n' %s > "$t" && mv -f "$t" %s`,
		shQuote(path.Dir(MarkersDir)), dir, dir, shQuote(MarkersDir+"/.tmp.XXXXXX"), shQuote(value), shQuote(target))
}
