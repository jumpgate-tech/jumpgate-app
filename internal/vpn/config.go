// Package vpn brings a WireGuard overlay up on a gateway host and treats the
// WireGuard config as the UNIVERSAL, provider-neutral interface.
//
// The design rule (so users can always bring their own): Jumpgate integrates
// the PROTOCOL — WireGuard — never a specific vendor's client. Any source that
// can emit a standard wg-quick `.conf` plugs in identically: ProtonVPN and
// Mullvad both hand you one, a self-hosted peer or Headscale/WireGuard mesh
// emits one, a corporate tunnel exports one. A "provider" (see tunnel.go) is
// therefore just a source of a Config; swapping ProtonVPN for your own is the
// default, not a feature bolted on.
//
// Once a tunnel is up, its interface address feeds the existing overlay
// grading (config.TrustedOverlays -> ops.bindAddrTier), so a gateway bound to
// the overlay IP grades "private" automatically.
package vpn

import (
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"
)

// Config is a parsed WireGuard configuration — the wg-quick `.conf` model.
// It is the lingua franca every provider produces and the wg-quick backend
// consumes.
type Config struct {
	Interface Interface
	Peers     []Peer
}

// Interface is the local end of the tunnel: the [Interface] section.
type Interface struct {
	PrivateKey string
	Address    []string // CIDRs assigned to the tunnel, e.g. ["10.2.0.2/32"]
	DNS        []string
	MTU        int
	ListenPort int
}

// Peer is a remote end: a [Peer] section. Providers ship exactly one for a
// simple client tunnel; a mesh ships several.
type Peer struct {
	PublicKey           string
	PresharedKey        string
	Endpoint            string
	AllowedIPs          []string
	PersistentKeepalive int
}

// ParseConfig reads a standard wg-quick `.conf`. It is intentionally lenient
// about whitespace, comments (# or ;), key casing, and comma-separated lists —
// the point is to accept whatever ProtonVPN, Mullvad, or `wg genkey` tooling
// emits without hand-massaging.
func ParseConfig(text string) (Config, error) {
	var cfg Config
	var section string
	var cur *Peer // the peer currently being filled, when section == "peer"

	for i, raw := range strings.Split(text, "\n") {
		line := strings.TrimSpace(stripComment(raw))
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.ToLower(strings.TrimSpace(line[1 : len(line)-1]))
			if section == "peer" {
				cfg.Peers = append(cfg.Peers, Peer{})
				cur = &cfg.Peers[len(cfg.Peers)-1]
			}
			continue
		}
		key, val, ok := splitKV(line)
		if !ok {
			return Config{}, fmt.Errorf("vpn: line %d: expected KEY = VALUE, got %q", i+1, line)
		}
		switch section {
		case "interface":
			if err := assignInterface(&cfg.Interface, key, val); err != nil {
				return Config{}, fmt.Errorf("vpn: line %d: %w", i+1, err)
			}
		case "peer":
			if cur == nil {
				return Config{}, fmt.Errorf("vpn: line %d: peer key outside a [Peer] section", i+1)
			}
			if err := assignPeer(cur, key, val); err != nil {
				return Config{}, fmt.Errorf("vpn: line %d: %w", i+1, err)
			}
		case "":
			return Config{}, fmt.Errorf("vpn: line %d: %q appears before any section header", i+1, line)
		default:
			// Unknown section (e.g. a vendor extension) — skip its keys rather
			// than fail the whole parse.
		}
	}
	return cfg, nil
}

func assignInterface(in *Interface, key, val string) error {
	switch strings.ToLower(key) {
	case "privatekey":
		in.PrivateKey = val
	case "address":
		in.Address = append(in.Address, splitList(val)...)
	case "dns":
		in.DNS = append(in.DNS, splitList(val)...)
	case "mtu":
		n, err := strconv.Atoi(val)
		if err != nil {
			return fmt.Errorf("bad MTU %q", val)
		}
		in.MTU = n
	case "listenport":
		n, err := strconv.Atoi(val)
		if err != nil {
			return fmt.Errorf("bad ListenPort %q", val)
		}
		in.ListenPort = n
	}
	return nil
}

func assignPeer(p *Peer, key, val string) error {
	switch strings.ToLower(key) {
	case "publickey":
		p.PublicKey = val
	case "presharedkey":
		p.PresharedKey = val
	case "endpoint":
		p.Endpoint = val
	case "allowedips":
		p.AllowedIPs = append(p.AllowedIPs, splitList(val)...)
	case "persistentkeepalive":
		n, err := strconv.Atoi(val)
		if err != nil {
			return fmt.Errorf("bad PersistentKeepalive %q", val)
		}
		p.PersistentKeepalive = n
	}
	return nil
}

// Validate checks the config carries the minimum needed to bring a tunnel up:
// a private key and at least one address on the interface, and at least one
// peer with a public key and somewhere to send (an endpoint or allowed-ips).
func (c Config) Validate() error {
	if c.Interface.PrivateKey == "" {
		return fmt.Errorf("vpn: [Interface] missing PrivateKey")
	}
	if len(c.Interface.Address) == 0 {
		return fmt.Errorf("vpn: [Interface] missing Address")
	}
	if len(c.Peers) == 0 {
		return fmt.Errorf("vpn: no [Peer] section")
	}
	for i, p := range c.Peers {
		if p.PublicKey == "" {
			return fmt.Errorf("vpn: [Peer] %d missing PublicKey", i+1)
		}
		if p.Endpoint == "" && len(p.AllowedIPs) == 0 {
			return fmt.Errorf("vpn: [Peer] %d has neither Endpoint nor AllowedIPs", i+1)
		}
	}
	return nil
}

// OverlayCIDRs returns the tunnel's own interface addresses as CIDRs, to be
// registered as trusted overlays so a gateway bound to the overlay IP grades
// private. A bare address (Proton emits "10.2.0.2/32" but some tools emit
// "10.2.0.2") is normalized to a host route.
func (c Config) OverlayCIDRs() []string {
	seen := map[string]bool{}
	var out []string
	for _, a := range c.Interface.Address {
		cidr := normalizeCIDR(a)
		if cidr != "" && !seen[cidr] {
			seen[cidr] = true
			out = append(out, cidr)
		}
	}
	sort.Strings(out)
	return out
}

// Render emits a canonical wg-quick `.conf`. Round-trips with ParseConfig for
// the fields Jumpgate models.
func (c Config) Render() string {
	var b strings.Builder
	b.WriteString("[Interface]\n")
	writeKV(&b, "PrivateKey", c.Interface.PrivateKey)
	if len(c.Interface.Address) > 0 {
		writeKV(&b, "Address", strings.Join(c.Interface.Address, ", "))
	}
	if len(c.Interface.DNS) > 0 {
		writeKV(&b, "DNS", strings.Join(c.Interface.DNS, ", "))
	}
	if c.Interface.MTU != 0 {
		writeKV(&b, "MTU", strconv.Itoa(c.Interface.MTU))
	}
	if c.Interface.ListenPort != 0 {
		writeKV(&b, "ListenPort", strconv.Itoa(c.Interface.ListenPort))
	}
	for _, p := range c.Peers {
		b.WriteString("\n[Peer]\n")
		writeKV(&b, "PublicKey", p.PublicKey)
		if p.PresharedKey != "" {
			writeKV(&b, "PresharedKey", p.PresharedKey)
		}
		if p.Endpoint != "" {
			writeKV(&b, "Endpoint", p.Endpoint)
		}
		if len(p.AllowedIPs) > 0 {
			writeKV(&b, "AllowedIPs", strings.Join(p.AllowedIPs, ", "))
		}
		if p.PersistentKeepalive != 0 {
			writeKV(&b, "PersistentKeepalive", strconv.Itoa(p.PersistentKeepalive))
		}
	}
	return b.String()
}

// --- small parsing helpers --------------------------------------------------

func stripComment(s string) string {
	for _, c := range []string{"#", ";"} {
		if i := strings.Index(s, c); i >= 0 {
			s = s[:i]
		}
	}
	return s
}

func splitKV(line string) (key, val string, ok bool) {
	i := strings.Index(line, "=")
	if i < 0 {
		return "", "", false
	}
	return strings.TrimSpace(line[:i]), strings.TrimSpace(line[i+1:]), true
}

func splitList(v string) []string {
	var out []string
	for _, p := range strings.Split(v, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func writeKV(b *strings.Builder, k, v string) {
	b.WriteString(k)
	b.WriteString(" = ")
	b.WriteString(v)
	b.WriteString("\n")
}

// normalizeCIDR turns an interface Address into a canonical CIDR. A bare IP
// becomes a /32 (v4) or /128 (v6) host route. Invalid input returns "".
func normalizeCIDR(a string) string {
	a = strings.TrimSpace(a)
	if a == "" {
		return ""
	}
	if strings.Contains(a, "/") {
		ip, ipnet, err := net.ParseCIDR(a)
		if err != nil {
			return ""
		}
		// Prefer the host route (the actual overlay address), not the masked network.
		bits := 32
		if ip.To4() == nil {
			bits = 128
		}
		_ = ipnet
		return fmt.Sprintf("%s/%d", ip.String(), bits)
	}
	ip := net.ParseIP(a)
	if ip == nil {
		return ""
	}
	if ip.To4() != nil {
		return a + "/32"
	}
	return a + "/128"
}
