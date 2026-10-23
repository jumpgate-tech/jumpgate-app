package catalog

import (
	"fmt"
	"slices"
)

// Placement says where a module lives, and so where it is recorded and
// what renders it.
type Placement string

const (
	PlaceNode    Placement = "node"    // flags/namespaces on a node's exec client
	PlaceBox     Placement = "box"     // its own systemd unit on a box
	PlaceGateway Placement = "gateway" // method routing on a gateway
)

// Module is an optional capability: an add-on to a node's execution client,
// a service of its own on a box, or routing on a gateway. Fields that only
// one placement uses are zero for the others; checkModules enforces that.
type Module struct {
	ID      string // "msgboard"; also the docs reference name
	Name    string // "Message board"
	Summary string // one line, shown under the toggle
	SiteURL string // "https://msgboard.xyz"; "" for none

	Placement Placement

	Chains      []int          // chains it applies to; empty = any chain (box only)
	BetaChains  []int          // subset of Chains, shown with a "beta" label
	Unsupported map[int]string // chain → reason: disabled-with-reason instead of hidden
	Clients     []string       // node: exec clients it applies to; box/gateway: nil

	Namespaces []string       // node only: added to the exec client's --http.api
	Flags      []string       // node only: extra exec flags
	Methods    []ModuleMethod // node only: every method the namespaces serve

	Probe     ModuleProbe // how to tell a binary and a live node support it
	NeedsSync bool        // serves nothing useful until the node is synced
	Resources Resources   // extra RAM and disk, for display and preflight

	Unit   func(b BoxModule) (BoxUnit, error) // box only
	Routes map[int][]GatewayRoute             // gateway only

	ClientReason func(client string) string // node only: the disabled reason for another client
	PollHint     string                     // appended to the relay's stream refusal: "poll msgboard_contentPage"

	DocsStep string // the docs step that explains it
}

type MethodClass string

const (
	MethodRead      MethodClass = "read"      // bounded reads
	MethodUnbounded MethodClass = "unbounded" // reads with no response bound
	MethodWrite     MethodClass = "write"     // changes node state or gossips
	MethodStream    MethodClass = "stream"    // subscriptions
)

type ModuleMethod struct {
	Name  string
	Class MethodClass
}

type ModuleProbe struct {
	HelpContains string // a flag `<bin> node --help` must list: "--msgboard.block-range"
	StatusMethod string // a cheap read that must answer: "msgboard_status"
}

type Resources struct{ RAMBytes, DiskBytes uint64 }

// moduleOrder is the one registration order. Each module's file appends its
// entry from its own init; the registry is sorted by this list so the order
// never depends on file init order.
var moduleOrder = []string{"msgboard", "el-bootnode", "cl-bootnode", "watch", "private-send", "checkpointz"}

// modules is the registry. Empty until the module tasks add their entries.
var modules []Module

// init runs after the module_*.go inits, which sort before modules.go.
func init() {
	sortModules(modules)
	if err := checkModules(modules); err != nil {
		panic(err)
	}
}

func sortModules(ms []Module) {
	slices.SortStableFunc(ms, func(a, b Module) int {
		return slices.Index(moduleOrder, a.ID) - slices.Index(moduleOrder, b.ID)
	})
}

// Modules returns every module, in moduleOrder. The slice is a copy.
//
// docs:covers modules-what
func Modules() []Module { return slices.Clone(modules) }

// ModuleByID returns the module with that id.
func ModuleByID(id string) (Module, bool) {
	for _, m := range modules {
		if m.ID == id {
			return m, true
		}
	}
	return Module{}, false
}

// Applies reports whether m can run on chainID with client. ok false with an
// empty reason hides the toggle; a non-empty reason disables it and is shown
// verbatim. Unsupported is checked first, then Chains, then Clients.
//
// docs:covers modules-what
func (m Module) Applies(chainID int, client string) (ok bool, reason string) {
	if r, bad := m.Unsupported[chainID]; bad {
		return false, r
	}
	if len(m.Chains) > 0 && !slices.Contains(m.Chains, chainID) {
		return false, ""
	}
	if len(m.Clients) > 0 && !slices.Contains(m.Clients, client) {
		if m.ClientReason == nil {
			return false, ""
		}
		return false, m.ClientReason(client)
	}
	return true, ""
}

// IsBeta reports whether chainID is one of m's beta chains.
func (m Module) IsBeta(chainID int) bool { return slices.Contains(m.BetaChains, chainID) }

// ModuleWriteMethods maps every node module's write method to its module id.
// The relay refuses these unless a key grants them explicitly.
func ModuleWriteMethods() map[string]string {
	out := map[string]string{}
	for _, m := range modules {
		if m.Placement != PlaceNode {
			continue
		}
		for _, mm := range m.Methods {
			if mm.Class == MethodWrite {
				out[mm.Name] = m.ID
			}
		}
	}
	return out
}

// ModuleStreamMethods maps every node module's stream method to the poll
// hint the relay appends to its refusal ("poll msgboard_contentPage").
func ModuleStreamMethods() map[string]string {
	out := map[string]string{}
	for _, m := range modules {
		if m.Placement != PlaceNode {
			continue
		}
		for _, mm := range m.Methods {
			if mm.Class == MethodStream {
				out[mm.Name] = m.PollHint
			}
		}
	}
	return out
}

// checkModules is the init-time check of the registry.
func checkModules(ms []Module) error {
	seen := map[string]bool{}
	for _, m := range ms {
		e := func(format string, a ...any) error {
			return fmt.Errorf("module %q: "+format, append([]any{m.ID}, a...)...)
		}
		if seen[m.ID] {
			return e("duplicate id")
		}
		seen[m.ID] = true
		for _, ns := range m.Namespaces {
			if ns == "all" || ns == "none" {
				return e("namespace %q is refused", ns)
			}
		}
		for _, c := range m.BetaChains {
			if !slices.Contains(m.Chains, c) {
				return e("beta chain %d is not in Chains", c)
			}
		}
		nodeOnly := len(m.Namespaces) > 0 || len(m.Flags) > 0 || len(m.Methods) > 0 || len(m.Clients) > 0 || m.ClientReason != nil
		switch m.Placement {
		case PlaceNode:
			if m.Unit != nil || m.Routes != nil {
				return e("a node module sets Unit or Routes")
			}
			if len(m.Chains) == 0 {
				return e("a node module needs Chains")
			}
		case PlaceBox:
			if nodeOnly || m.Routes != nil {
				return e("a box module sets node or gateway fields")
			}
			if m.Unit == nil {
				return e("a box module needs Unit")
			}
		case PlaceGateway:
			if nodeOnly || m.Unit != nil {
				return e("a gateway module sets node or box fields")
			}
			if len(m.Chains) == 0 {
				return e("a gateway module needs Chains")
			}
			if len(m.Routes) == 0 {
				return e("a gateway module needs Routes")
			}
		default:
			return e("unknown placement %q", m.Placement)
		}
		if !slices.Contains(moduleOrder, m.ID) {
			return e("id is missing from moduleOrder")
		}
	}
	return nil
}
