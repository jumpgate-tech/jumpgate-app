package catalog

import (
	"errors"
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
)

// Checkpointz (ethpandaops/checkpointz) is a box module: a checkpoint-sync
// server in front of one or more synced beacon nodes. It serves your own
// boxes only, never the internet: it binds to loopback, or to a private
// address your other boxes can reach, and their beacons can then use it as
// their checkpoint-sync URL instead of a public one.
//
// It runs in "full" mode: "light" serves only blocks, and a beacon cannot
// checkpoint-sync from it. Config keys are checked against v0.32.0 in
// testdata/checkpointz-keys-v0.32.0.txt.
//
// PulseChain is off for now: custom_preset makes checkpointz read the
// beacon's own spec (go-eth2-client custom spec support, dynamic SSZ), which
// should cover PulseChain's preset, but no run against a synced
// lighthouse-pulse beacon has shown it serving a checkpoint another node
// syncs from (plan Task 19). 369/943 stay in Chains and BetaChains, so
// dropping the Unsupported entry once that test passes ships them as beta.
func init() {
	modules = append(modules, Module{
		ID:         "checkpointz",
		Name:       "Checkpointz",
		Summary:    "Serves finalized checkpoints from your own beacon node, so your other nodes can checkpoint-sync from it.",
		SiteURL:    "https://github.com/ethpandaops/checkpointz",
		Placement:  PlaceBox,
		Chains:     []int{1, 560048, 369, 943},
		BetaChains: []int{369, 943},
		Unsupported: map[int]string{
			369: checkpointzPulseReason,
			943: checkpointzPulseReason,
		},
		NeedsSync: true,
		// Full mode caches up to 5 finalized states in memory.
		Resources: Resources{RAMBytes: 3 << 30, DiskBytes: 1 << 20},
		Unit:      checkpointzUnit,
	})
}

const checkpointzPort = 5555

// checkpointzPulseReason disables Checkpointz on PulseChain until the 943
// test passes (plan Task 19's fallback copy).
const checkpointzPulseReason = "Checkpointz has not been shown to work with PulseChain's beacon preset yet."

// checkpointzListen is b's listen host and port: BindAddr (loopback, or a
// private address) or 127.0.0.1, and Port or 5555.
func checkpointzListen(b BoxModule) (string, int, error) {
	port := b.Port
	if port == 0 {
		port = checkpointzPort
	}
	if port < 1 || port > 65535 {
		return "", 0, fmt.Errorf("catalog: checkpointz port %d is out of range", b.Port)
	}
	if b.BindAddr == "" {
		return "127.0.0.1", port, nil
	}
	ip, err := netip.ParseAddr(b.BindAddr)
	if err != nil || ip.Zone() != "" {
		return "", 0, fmt.Errorf("catalog: checkpointz BindAddr %q is not an IP address", b.BindAddr)
	}
	ip = ip.Unmap()
	if !ip.IsLoopback() && !ip.IsPrivate() && !cgnat.Contains(ip) {
		return "", 0, fmt.Errorf("catalog: Checkpointz serves only your own boxes: bind it to loopback or a private address, not %s", ip)
	}
	return ip.String(), port, nil
}

// cgnat is 100.64.0.0/10, the shared range overlay networks (Tailscale)
// give boxes: private in practice, though not in net.IP.IsPrivate.
var cgnat = netip.MustParsePrefix("100.64.0.0/10")

// CheckpointzURL is the checkpoint-sync URL other boxes use for b, or ""
// when b listens on loopback and only its own box can reach it.
//
// docs:covers checkpointz-use
func CheckpointzURL(b BoxModule) string {
	host, port, err := checkpointzListen(b)
	if err != nil {
		return ""
	}
	if ip, err := netip.ParseAddr(host); err == nil && ip.IsLoopback() {
		return ""
	}
	return "http://" + net.JoinHostPort(host, strconv.Itoa(port))
}

// CheckpointzDefaultUpstreams is the box's own beacon, when w is a node on
// chainID: the upstream a Checkpointz gets when none is given. A beacon
// bound to every address is reached on loopback.
func CheckpointzDefaultUpstreams(w *WireConfig, chainID int) []string {
	if w == nil || w.ChainID != chainID {
		return nil
	}
	host := w.RPCBind()
	if ip := net.ParseIP(host); ip == nil || ip.IsUnspecified() {
		host = "127.0.0.1"
	}
	return []string{"http://" + net.JoinHostPort(host, strconv.Itoa(w.BeaconHTTP()))}
}

// checkpointzUpstream checks one beacon URL: http or https with a host, and
// nothing that could break out of a quoted YAML string. Upstream URLs can
// carry keys, so errors name only the entry's position.
func checkpointzUpstream(i int, raw string) error {
	for _, r := range raw {
		if r <= ' ' || r == 0x7f || strings.ContainsRune("\"'`\\$;|&<>(){}", r) {
			return fmt.Errorf("catalog: checkpointz upstream %d has a character a URL here may not hold", i)
		}
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return fmt.Errorf("catalog: checkpointz upstream %d must be an http:// or https:// beacon URL", i)
	}
	return nil
}

// yamlQuote is s as a double-quoted YAML scalar. Every string the config
// holds has been checked first; this keeps it a single scalar regardless.
func yamlQuote(s string) string { return strconv.Quote(s) }

// checkpointzConfig renders config.yaml.
func checkpointzConfig(b BoxModule, listen string) string {
	var sb strings.Builder
	sb.WriteString("global:\n")
	sb.WriteString("  listenAddr: " + yamlQuote(listen) + "\n")
	sb.WriteString("  logging: \"info\"\n")
	sb.WriteString("checkpointz:\n")
	sb.WriteString("  mode: \"full\"\n")
	if b.ChainID == 369 || b.ChainID == 943 {
		sb.WriteString("  custom_preset: true\n")
	}
	sb.WriteString("  frontend:\n")
	sb.WriteString("    enabled: false\n")
	sb.WriteString("beacon:\n")
	sb.WriteString("  upstreams:\n")
	for i, u := range b.Upstreams {
		sb.WriteString("    - name: " + yamlQuote("beacon-"+strconv.Itoa(i+1)) + "\n")
		sb.WriteString("      address: " + yamlQuote(u) + "\n")
		sb.WriteString("      dataProvider: true\n")
	}
	return sb.String()
}

// checkpointzUnit runs checkpointz with its config.yaml from ConfigDir,
// which holds the upstream URLs and is readable by the service's group
// only. The unit names only that file.
//
// docs:covers checkpointz-run
func checkpointzUnit(b BoxModule) (BoxUnit, error) {
	if b.ExtIP != "" || b.Webhook != "" {
		return BoxUnit{}, errors.New("catalog: Checkpointz takes no ExtIP or webhook")
	}
	host, port, err := checkpointzListen(b)
	if err != nil {
		return BoxUnit{}, err
	}
	if len(b.Upstreams) == 0 {
		return BoxUnit{}, fmt.Errorf("catalog: Checkpointz needs at least one synced beacon node on chain %d", b.ChainID)
	}
	seen := map[string]bool{}
	for i, u := range b.Upstreams {
		if err := checkpointzUpstream(i+1, u); err != nil {
			return BoxUnit{}, err
		}
		if seen[u] {
			return BoxUnit{}, fmt.Errorf("catalog: checkpointz upstream %d is listed twice", i+1)
		}
		seen[u] = true
	}
	conf := b.ConfigDir() + "/config.yaml"
	unit, err := renderServiceUnit("module "+b.Instance(), "checkpointz --config "+conf, CurrentNames, b.DataDir(), false)
	if err != nil {
		return BoxUnit{}, err
	}
	listen := net.JoinHostPort(host, strconv.Itoa(port))
	return BoxUnit{
		Unit:   unit,
		Files:  map[string]string{conf: checkpointzConfig(b, listen)},
		Client: "checkpointz",
		Ports:  []PortUse{{Port: port, Proto: "tcp"}},
	}, nil
}
