package catalog

import (
	"fmt"
	"slices"
	"strings"
)

// NodeModule is one node module turned on in a WireConfig.
type NodeModule struct {
	ID     string `json:"id"`
	Public string `json:"public,omitempty"` // "" | "read" | "post"
}

// baseHTTPAPIs is the --http.api list a client serves before any module
// adds to it. Only clients listed here can take node-module namespaces.
var baseHTTPAPIs = map[string][]string{"reth": {"eth", "net", "web3"}}

// HasModule reports whether the node module id is on in w.
func (w WireConfig) HasModule(id string) bool {
	return slices.ContainsFunc(w.Modules, func(m NodeModule) bool { return m.ID == id })
}

// ValidateModules checks every node module on w: known, listed once, a node
// module, applicable to w's chain and exec client, and a valid Public.
func ValidateModules(w WireConfig) error {
	seen := map[string]bool{}
	for _, nm := range w.Modules {
		m, ok := ModuleByID(nm.ID)
		if !ok {
			return fmt.Errorf("catalog: unknown module %q", nm.ID)
		}
		if seen[nm.ID] {
			return fmt.Errorf("catalog: module %q listed twice", nm.ID)
		}
		seen[nm.ID] = true
		if m.Placement != PlaceNode {
			return fmt.Errorf("catalog: %s is not a node module", nm.ID)
		}
		if ok, reason := m.Applies(w.ChainID, w.ExecID); !ok {
			if reason == "" {
				return fmt.Errorf("catalog: the %s module does not apply to chain %d", nm.ID, w.ChainID)
			}
			return fmt.Errorf("%s", reason)
		}
		switch nm.Public {
		case "", "read", "post":
		default:
			return fmt.Errorf("catalog: public must be \"\", \"read\" or \"post\", not %q", nm.Public)
		}
	}
	return nil
}

// enabledNodeModules returns w's node modules in catalog order.
func enabledNodeModules(w WireConfig) []Module {
	var out []Module
	for _, m := range modules {
		if m.Placement == PlaceNode && w.HasModule(m.ID) {
			out = append(out, m)
		}
	}
	return out
}

// execHTTPAPI composes the exec client's --http.api value: its base list,
// then each enabled module's namespaces in catalog order, duplicates
// collapsed. "" means render no flag, which is every wire without modules.
// It never returns a list holding "all" or "none".
func execHTTPAPI(w WireConfig) (string, error) {
	mods := enabledNodeModules(w)
	if len(mods) == 0 {
		return "", nil
	}
	base, ok := baseHTTPAPIs[w.ExecID]
	if !ok {
		return "", fmt.Errorf("catalog: %s takes no node modules", w.ExecID)
	}
	list := slices.Clone(base)
	for _, m := range mods {
		for _, ns := range m.Namespaces {
			if !slices.Contains(list, ns) {
				list = append(list, ns)
			}
		}
	}
	for _, ns := range list {
		if strings.EqualFold(ns, "all") || strings.EqualFold(ns, "none") {
			return "", fmt.Errorf("catalog: --http.api would contain %q", ns)
		}
	}
	return strings.Join(list, ","), nil
}

// execModuleFlags is the exec flag tail for w's node modules: --http.api,
// then each module's Flags in catalog order. "" without modules.
func execModuleFlags(w WireConfig) (string, error) {
	api, err := execHTTPAPI(w)
	if err != nil || api == "" {
		return "", err
	}
	out := " --http.api " + api
	for _, m := range enabledNodeModules(w) {
		for _, f := range m.Flags {
			out += " " + f
		}
	}
	return out, nil
}
