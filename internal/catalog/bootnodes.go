package catalog

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// errBootnodePrefix is the one error for an entry that is neither kind.
var errBootnodePrefix = errors.New("catalog: a bootnode must start with enode:// or enr:-")

// enodeRe is an enode URL: a 128-hex node id, a host (IPv4, a [IPv6]
// literal or a DNS name) and a port, with an optional discport query.
// enrRe is an ENR: "enr:-" then base64url. Both character sets leave out
// whitespace, commas, quotes, '%' (a systemd specifier), '$', '\' and
// every other shell metacharacter, so an entry is safe in an ExecStart line
// and in any shell string as-is.
var (
	enodeRe = regexp.MustCompile(`^enode://[0-9a-fA-F]{128}@(\[[0-9a-fA-F:.]+\]|[A-Za-z0-9.-]+):[0-9]{1,5}(\?discport=[0-9]{1,5})?$`)
	enrRe   = regexp.MustCompile(`^enr:-[A-Za-z0-9_-]+$`)
)

// ValidateBootnode checks one Bootnodes entry.
func ValidateBootnode(s string) error {
	switch {
	case strings.HasPrefix(s, "enode://"):
		if !enodeRe.MatchString(s) {
			return fmt.Errorf("catalog: bootnode %q is not a valid enode URL (enode://<128 hex>@<host>:<port>)", s)
		}
	case strings.HasPrefix(s, "enr:-"):
		if !enrRe.MatchString(s) {
			return fmt.Errorf("catalog: bootnode %q is not a valid ENR (enr:- then base64url)", s)
		}
	default:
		return errBootnodePrefix
	}
	return nil
}

// ValidateBootnodes checks every entry of w.Bootnodes.
func ValidateBootnodes(w WireConfig) error {
	for _, b := range w.Bootnodes {
		if err := ValidateBootnode(b); err != nil {
			return err
		}
	}
	return nil
}

// elBootnodes are the enode:// entries, for the execution client.
//
// docs:covers bootnode-use
func (w WireConfig) elBootnodes() []string {
	return bootnodesWithPrefix(w.Bootnodes, "enode://")
}

// clBootnodes are the enr:- entries, for the beacon client.
//
// docs:covers bootnode-use
func (w WireConfig) clBootnodes() []string {
	return bootnodesWithPrefix(w.Bootnodes, "enr:-")
}

func bootnodesWithPrefix(list []string, prefix string) []string {
	var out []string
	for _, b := range list {
		if strings.HasPrefix(b, prefix) {
			out = append(out, b)
		}
	}
	return out
}

// execBootnodeFlag is " --bootnodes a,b" (reth, geth, go-pulse,
// erigon-pulse), or "" with no enode entries.
func execBootnodeFlag(w WireConfig) string {
	el := w.elBootnodes()
	if len(el) == 0 {
		return ""
	}
	return " --bootnodes " + strings.Join(el, ",")
}

// beaconBootnodeFlag renders the enr entries for the beacon client: one
// " --boot-nodes a,b" for the lighthouse family, one --bootstrap-node= per
// entry for prysm-pulse, "" with no entries.
func beaconBootnodeFlag(w WireConfig) string {
	cl := w.clBootnodes()
	if len(cl) == 0 {
		return ""
	}
	if w.BeaconID == "prysm-pulse" {
		var out string
		for _, e := range cl {
			out += " --bootstrap-node=" + e
		}
		return out
	}
	return " --boot-nodes " + strings.Join(cl, ",")
}
