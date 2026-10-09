package jobs

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/valve-tech/jumpgate/internal/catalog"
	"github.com/valve-tech/jumpgate/internal/fsatomic"
)

// NamesMarker is /etc/jumpgate/names.json, written by node.migrate-names when
// it switches a box's names. The agent applies it over node.json on every
// read (D-J10): pairing rewrites node.json from the controller's copy of the
// wire, and a controller that missed the migration must not point the agent
// back at units and an account that no longer exist.
type NamesMarker struct {
	Names        string `json:"names"`
	DataRootFrom string `json:"dataRootFrom,omitempty"` // empty when the data stayed put
	DataRootTo   string `json:"dataRootTo,omitempty"`
}

// ReadNamesMarker reads the marker at path; ok is false when there is none.
// A marker this agent could not have written is an error, never applied: it
// moves every path the agent and setup act on as root.
func ReadNamesMarker(path string) (NamesMarker, bool, error) {
	var m NamesMarker
	b, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return m, false, nil
	}
	if err != nil {
		return m, false, err
	}
	if err := json.Unmarshal(b, &m); err != nil {
		return m, false, fmt.Errorf("names marker %s: %w", path, err)
	}
	if err := m.check(); err != nil {
		return m, false, fmt.Errorf("names marker %s: %w", path, err)
	}
	return m, true, nil
}

// check accepts exactly what node.migrate-names writes: the current names,
// with the data root either left alone or moved from the legacy root to the
// current one.
func (m NamesMarker) check() error {
	if m.Names != catalog.NamesCurrent {
		return fmt.Errorf("names %q, want %q", m.Names, catalog.NamesCurrent)
	}
	switch {
	case m.DataRootFrom == "" && m.DataRootTo == "":
	case m.DataRootFrom == catalog.LegacyNames.DataRoot && m.DataRootTo == catalog.CurrentNames.DataRoot:
	default:
		return fmt.Errorf("moves the data root from %q to %q, which node.migrate-names never does", m.DataRootFrom, m.DataRootTo)
	}
	return nil
}

func writeNamesMarker(path string, m NamesMarker) error {
	if err := m.check(); err != nil {
		return err
	}
	b, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return fsatomic.WriteFile(path, b)
}

// Apply rewrites a node config the marker post-dates.
func (m NamesMarker) Apply(w *catalog.WireConfig) {
	w.Names = m.Names
	if m.DataRootFrom == "" {
		return
	}
	if d, ok := catalog.Rebase(w.DataDir, m.DataRootFrom, m.DataRootTo); ok {
		w.DataDir = d
	}
	if j, ok := catalog.Rebase(w.JWTPath, m.DataRootFrom, m.DataRootTo); ok {
		w.JWTPath = j
	}
}
