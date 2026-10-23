package catalog

import (
	"fmt"
	"net"
	"slices"
	"strings"
)

// GatewayRoute is one method route a gateway module adds for a chain: an
// upstream of its own that serves only Methods.
type GatewayRoute struct {
	ID       string   // upstream id suffix, "flashbots"
	Endpoint string   // http(s):// or ws(s)://
	Methods  []string // the only methods this upstream serves
}

// GatewayModule is a gateway module turned on. Only the id is stored; the
// routes are derived at render time from the catalog, the same way Endpoint
// is derived for managed upstreams.
type GatewayModule struct {
	ID string `json:"id"`
}

// Tier is how exposed a gateway is, and so which module methods it may route.
type Tier string

const (
	TierLocal   Tier = "local"   // no TLS front, bound to loopback
	TierPrivate Tier = "private" // reachable off the box: no TLS, or an internal or operator-supplied cert
	TierPublic  Tier = "public"  // a publicly trusted ACME cert: anyone can reach it
)

// GatewayTier derives a gateway's tier from its TLS front and bind address:
// with no TLS, only a loopback bind is Local; any other bind is reachable
// from the network, so it is Private.
func GatewayTier(g GatewayConfig) Tier {
	switch {
	case !g.TLS.On() && isLoopbackHost(g.Bind()):
		return TierLocal
	case !g.TLS.On():
		return TierPrivate
	case g.TLS.CertSourceOrDefault() == CertACME:
		return TierPublic
	default:
		return TierPrivate
	}
}

// isLoopbackHost is whether host is "localhost" or a loopback IP.
func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(strings.Trim(host, "[]"))
	return ip != nil && ip.IsLoopback()
}

// nsWildcards is "<ns>_*" for each of m's namespaces.
func nsWildcards(m Module) []string {
	out := make([]string, 0, len(m.Namespaces))
	for _, ns := range m.Namespaces {
		out = append(out, ns+"_*")
	}
	return out
}

// methodsOf is m's methods in the given classes, in catalog order.
func methodsOf(m Module, classes ...MethodClass) []string {
	var out []string
	for _, mm := range m.Methods {
		if slices.Contains(classes, mm.Class) {
			out = append(out, mm.Name)
		}
	}
	return out
}

// moduleIgnoreMethods is the ignoreMethods list for one upstream (spec D6).
// w is the managed node's wire, nil for any other upstream.
//
//   - An upstream that does not serve a module (external, devnet, or a node
//     without it) ignores the module's whole namespace, so a write never fails
//     over to a public endpoint and a read never gets "method not found".
//   - A node with the module ignores its stream methods behind every gateway
//     (the node has no WebSocket listener), and behind a Public-tier gateway
//     also whatever its Public setting does not open. Unbounded reads are
//     never public.
//
// docs:covers msgboard-expose
func moduleIgnoreMethods(tier Tier, u GatewayUpstream, w *WireConfig) []string {
	var out []string
	for _, m := range modules {
		if m.Placement != PlaceNode {
			continue
		}
		if u.KindOrDefault() != UpstreamManagedNode || w == nil || !w.HasModule(m.ID) {
			out = append(out, nsWildcards(m)...)
			continue
		}
		if tier != TierPublic {
			out = append(out, methodsOf(m, MethodStream)...)
			continue
		}
		switch nodeModulePublic(*w, m.ID) {
		case "read":
			out = append(out, methodsOf(m, MethodWrite, MethodUnbounded, MethodStream)...)
		case "post":
			out = append(out, methodsOf(m, MethodUnbounded, MethodStream)...)
		default:
			out = append(out, nsWildcards(m)...)
		}
	}
	return out
}

// nodeModulePublic is the Public setting of module id on w.
func nodeModulePublic(w WireConfig, id string) string {
	for _, nm := range w.Modules {
		if nm.ID == id {
			return nm.Public
		}
	}
	return ""
}

// ApplyModuleIgnores sets IgnoreMethods on every upstream of n. wires[i] is
// the managed node behind n.Upstreams[i], nil for any other upstream.
//
// anyModules is whether any module is on anywhere on this gateway: a node
// module on any upstream of any chain, or a gateway module. When it is, every
// list is kept, so a module's writes on a chain no node serves never fall
// over to an external upstream. When it is not, a module's methods are listed
// only on a chain where some upstream serves that module, so a gateway with
// no module at all renders byte for byte what it did before and restarts
// nothing.
func ApplyModuleIgnores(tier Tier, n GatewayNetwork, wires []*WireConfig, anyModules bool) GatewayNetwork {
	served := map[string]bool{}
	for _, w := range wires {
		if w == nil {
			continue
		}
		for _, nm := range w.Modules {
			served[nm.ID] = true
		}
	}
	// The methods of modules no upstream serves, which are dropped again.
	unserved := map[string]bool{}
	for _, m := range modules {
		if !anyModules && m.Placement == PlaceNode && !served[m.ID] {
			for _, x := range nsWildcards(m) {
				unserved[x] = true
			}
		}
	}
	out := n
	out.Upstreams = slices.Clone(n.Upstreams)
	for i := range out.Upstreams {
		var w *WireConfig
		if i < len(wires) {
			w = wires[i]
		}
		var keep []string
		for _, x := range moduleIgnoreMethods(tier, out.Upstreams[i], w) {
			if !unserved[x] {
				keep = append(keep, x)
			}
		}
		out.Upstreams[i].IgnoreMethods = keep
	}
	return out
}

// gatewayModuleUpstreams is what g's gateway modules add on chainID: one extra
// upstream per route, serving only its methods, and the methods every other
// upstream on that chain must then ignore, so those calls never fall over to
// a public endpoint.
func gatewayModuleUpstreams(g GatewayConfig, chainID int) (extra []GatewayUpstream, ignoreOnOthers []string) {
	for _, gm := range g.Modules {
		m, ok := ModuleByID(gm.ID)
		if !ok || m.Placement != PlaceGateway {
			continue
		}
		if ok, _ := m.Applies(chainID, ""); !ok {
			continue
		}
		for _, r := range m.Routes[chainID] {
			extra = append(extra, GatewayUpstream{
				ID:       fmt.Sprintf("%d-mod-%s", chainID, r.ID),
				Endpoint: r.Endpoint,
				// eRPC ignores everything else when allowMethods is set and
				// ignoreMethods is not (erpc common/defaults.go:2046-2051 at
				// a72a9ddb). The "*" is written anyway, so the route stays
				// closed even if that default changes.
				IgnoreMethods: []string{"*"},
				AllowMethods:  slices.Clone(r.Methods),
			})
			for _, meth := range r.Methods {
				if !slices.Contains(ignoreOnOthers, meth) {
					ignoreOnOthers = append(ignoreOnOthers, meth)
				}
			}
		}
	}
	return extra, ignoreOnOthers
}

// ValidateGatewayModules refuses an unknown id, a duplicate, a module that is
// not a gateway module, and one that applies to none of g's chains.
func ValidateGatewayModules(g GatewayConfig) error {
	seen := map[string]bool{}
	for _, gm := range g.Modules {
		if seen[gm.ID] {
			return fmt.Errorf("%s is on this gateway twice", gm.ID)
		}
		seen[gm.ID] = true
		m, ok := ModuleByID(gm.ID)
		if !ok {
			return fmt.Errorf("%q is not a module", gm.ID)
		}
		if m.Placement != PlaceGateway {
			return fmt.Errorf("%s is not a gateway module", gm.ID)
		}
		applies := false
		for _, n := range g.Networks {
			if ok, _ := m.Applies(n.ChainID, ""); ok {
				applies = true
				break
			}
		}
		if !applies {
			return fmt.Errorf("%s applies to none of this gateway's chains", gm.ID)
		}
	}
	return nil
}
