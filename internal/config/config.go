// Package config persists jumpgate's own local state — the targets it
// knows how to manage, and the AI provider it's configured to use for log
// explanations — to a single JSON file under the user's home directory. It
// performs no validation of the domain data it stores (that's the caller's
// job); it only knows how to read and write the file safely.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/valve-tech/jumpgate/internal/catalog"
	"github.com/valve-tech/jumpgate/internal/executor"
	"github.com/valve-tech/jumpgate/internal/filelock"
	"github.com/valve-tech/jumpgate/internal/fsperm"
)

// defaultRefRPCBase is the public demo-key reference RPC base URL, used
// whenever a Config has no explicit override. Callers append "/evm/<chainId>"
// to get the per-chain reference endpoint.
const defaultRefRPCBase = "https://rpc.valve.city/v1/vk_Et-4emAlBIym1PjiCogh5p7IuGtS-Rpj"

// Target is one machine jumpgate can set up and monitor a node on.
//
// Wire and Devnet are two INDEPENDENT things a target may host, not two
// spellings of one. A machine can run a devnet and nothing else, or a full
// node with no devnet — so each is its own optional pointer whose nil means
// "not configured here", rather than fields on a single config that would
// have to be half ignored depending on which one the operator actually asked
// for. See catalog/devnet.go for why the configs cannot be collapsed into
// one.
//
// A GATEWAY is deliberately NOT here; see Gateway below.
type Target struct {
	ID   string              `json:"id"`   // "local" or a slug of the host
	Mode string              `json:"mode"` // "local" | "ssh"
	SSH  *executor.SSHConfig `json:"ssh,omitempty"`
	Wire *catalog.WireConfig `json:"wire,omitempty"` // set once the wizard has run

	// Devnet is the throwaway local chain this target hosts, if any. It is
	// the DESIRED configuration, which is not the same as what is running:
	// a container's ports and command line are fixed at creation, so an
	// edited config only takes effect once the service is re-provisioned.
	Devnet *catalog.DevnetConfig `json:"devnet,omitempty"`

	// LegacyGateway is the gateway this target used to OWN, and exists only
	// so an existing config file can be read and upgraded. Load moves it to
	// Config.Gateways and clears it, so it is never written back out (see
	// migrate). Nothing but the migration may read it.
	LegacyGateway *catalog.GatewayConfig `json:"gateway,omitempty"`

	// Agent is set once a jumpgate agent on this machine has been paired
	// with this controller. Nil means intents cannot be sent to it.
	Agent *AgentPairing `json:"agent,omitempty"`
}

// AgentPairing records that a target runs a paired jumpgate agent.
type AgentPairing struct {
	Address   string    `json:"address"` // the agent's signing address, fixed at pairing
	PairedAt  time.Time `json:"pairedAt"`
	Transport string    `json:"transport"` // "ssh" | "local"
	NextSeq   uint64    `json:"nextSeq"`   // next intent sequence for this controller
	// Socket overrides the agent's socket path, for tests and local
	// development. Empty means the agent's default, /run/jumpgate/agent.sock.
	Socket string `json:"socket,omitempty"`
}

// Controller is this machine's signing identity. The key itself is never here:
// KeyRef names where it lives (a path, a keychain item, an op:// reference).
type Controller struct {
	KeyStore string `json:"keyStore"` // "file" | "keychain" | "1password"
	KeyRef   string `json:"keyRef"`
	Address  string `json:"address"`
}

// GatewayPlacement is WHERE a gateway runs. It is a property OF the gateway,
// not of the machine: naming the host is how a gateway says "this is the box
// that happens to run me", and the machine gets no say in it.
type GatewayPlacement struct {
	// TargetID is the managed machine the gateway container/unit lives on.
	TargetID string `json:"targetId"`
	// Backend is "docker" or "systemd" (setup.BackendDocker / BackendSystemd).
	// Kept as a string so config does not depend on internal/setup.
	Backend string `json:"backend"`
}

// Gateway is one eRPC instance — a LAYER over the fleet, not a service a
// machine owns.
//
// WHY it is top-level rather than a field on Target: an eRPC instance points
// at N chains across M endpoints, and those endpoints can be anywhere — a
// devnet on this laptop, a node on a fleet box in another datacentre, a
// public mainnet endpoint. Exactly one of those M things is "the machine the
// gateway process happens to run on", and modelling the gateway as belonging
// to that machine made the incidental fact (where the process runs) into the
// structural one (what it fronts). Placement keeps the incidental fact where
// it belongs: as one field of the gateway.
//
// The practical consequence is that N gateways can coexist — ID is what
// keeps their containers, units, config files and routes apart.
type Gateway struct {
	// ID is stable and appears in routes, the container name and the unit
	// name. It is chosen once, at creation, and never derived from anything
	// mutable.
	ID string `json:"id"`

	// Placement names the machine that runs this gateway and how.
	Placement GatewayPlacement `json:"placement"`

	// Config is the whole multi-chain eRPC configuration: port, bind,
	// networks and their upstreams. Upstreams of a managed kind carry a
	// reference (kind + target id) rather than a frozen URL — see
	// catalog.GatewayUpstream.
	Config catalog.GatewayConfig `json:"config"`
}

// VPN is one WireGuard overlay the operator has configured — a bring-your-own
// provider config plus the local knobs for how it is applied.
//
// Config is the raw provider-neutral `.conf` text (Proton, Mullvad, a
// self-hosted mesh, …) stored VERBATIM. config does not parse, validate, or
// render it — that is internal/vpn's job (ParseConfig/Validate/Render), the same
// division of labour every other field here follows: config knows how to read
// and write the file safely and nothing about what the bytes mean. Storing the
// text rather than a parsed struct also keeps the exact `.conf` an operator
// pasted, comments and ordering included, so a round-trip never quietly rewrites
// their config.
//
// WHY a list (VPNs) and not one field: the product is provider-neutral by
// design (see internal/vpn's Provider seam), and an operator may hold several
// overlays at once — a Proton exit for one route, a self-hosted mesh for
// another — so each is its own entry keyed by a stable ID, exactly as Gateways
// are.
type VPN struct {
	// ID is stable and names the overlay wherever it is selected — the API,
	// the panel, the interface state. Chosen once at creation, never derived
	// from anything mutable (an endpoint or a key can change; the ID must not).
	ID string `json:"id"`

	// Provider is a display/telemetry label only ("proton", "mullvad",
	// "bring-your-own", …). It selects no behaviour by itself — the Config text
	// is what actually brings the tunnel up — so an unknown label is not an
	// error here, it is just what gets shown.
	Provider string `json:"provider,omitempty"`

	// Interface is the OS interface name to bring up, e.g. "jumpgate0". Empty
	// means the caller chooses a default; config does not invent one, because a
	// name it made up would then be the name teardown has to guess.
	Interface string `json:"interface,omitempty"`

	// TargetID is the machine the overlay is brought up ON. Empty means the
	// host running this app itself — the desktop case, where Jumpgate routes a
	// user's own computer through the tunnel. A named target (one of
	// Config.Targets) is the fleet case: the overlay comes up on a gateway box
	// so its address feeds the same overlay grading a self-hosted node does.
	// It is WHERE the tunnel runs, not what it is, so it lives here as one
	// field rather than splitting VPN into local and remote kinds.
	TargetID string `json:"targetId,omitempty"`

	// Config is the raw WireGuard `.conf` text, INCLUDING the interface private
	// key. It is a secret and lives here for the same reason AIKey and
	// ProviderKeys do: config.json is written mode 0600, and — like those — the
	// API never returns it (a settings response reports that an overlay is
	// configured, never its bytes).
	Config string `json:"config"`

	// Autostart brings this overlay up when the app starts, rather than waiting
	// for an operator to switch it on. Off by default: a tunnel that comes up on
	// its own can silently reroute every upstream, so opting in is deliberate.
	Autostart bool `json:"autostart,omitempty"`
}

// VPNServer is a WireGuard server this app PROVISIONED on one of its machines —
// the "provision on a device" half of the easy button, as opposed to a VPN
// (above), which is a bring-your-own client .conf.
//
// It is a separate type from VPN, not a mode of it, for the same reason
// Target.Wire and Target.Devnet are separate: the two shapes barely overlap and
// collapsing them would mean a struct half of whose fields are always ignored.
// The decisive difference is the secret. A VPN carries a pasted .conf INCLUDING
// its private key; a VPNServer carries NO private key at all — the server's key
// is generated on the host and stays there (see vpn.ProvisionServer), and each
// peer's private key is handed to its device once at enrollment and never
// stored. So everything here is safe to persist and safe to return over the
// API: public keys, addresses, ports, and the overlay IPs handed out.
type VPNServer struct {
	// ID is stable and names the server wherever it is selected.
	ID string `json:"id"`

	// TargetID is the machine the server runs ON. Empty means the host running
	// this app (the desktop case); a named target is a fleet box reached over
	// SSH. Same meaning as VPN.TargetID.
	TargetID string `json:"targetId,omitempty"`

	// Interface is the OS interface the server listens on, e.g. "jumpgate0".
	Interface string `json:"interface"`

	// Address is the server's own overlay address WITH mask, e.g. "10.9.0.1/24".
	// Its subnet is the pool peers are allocated from.
	Address string `json:"address"`

	// ListenPort is the UDP port devices dial.
	ListenPort int `json:"listenPort"`

	// PublicKey is the server's public key — NOT secret; it is exactly what a
	// device needs to talk to the server, and appears in every client config.
	PublicKey string `json:"publicKey"`

	// Endpoint is the host:port devices dial (the machine's reachable address
	// plus ListenPort). Stored because the machine's public address is not
	// derivable from anything else this app holds.
	Endpoint string `json:"endpoint"`

	// Peers are the devices enrolled on this server. Only the public half of
	// each device's identity lives here (public key + assigned overlay IP); the
	// device's private key is never stored — it was delivered to the device once,
	// at enrollment, and re-issuing means re-enrolling.
	Peers []VPNPeer `json:"peers,omitempty"`
}

// VPNPeer is one enrolled device on a VPNServer. It holds nothing secret: a
// name, the device's PUBLIC key, and the overlay address the server routes to
// it. The device's private key is deliberately absent — see VPNServer.
type VPNPeer struct {
	Name      string `json:"name"`
	PublicKey string `json:"publicKey"`
	AllowedIP string `json:"allowedIp"` // the /32 overlay address assigned to this device
}

// Config is jumpgate's persisted local state.
type Config struct {
	Targets []Target `json:"targets"`

	// Gateways are the eRPC instances, each naming the target it runs on.
	Gateways []Gateway `json:"gateways,omitempty"`

	// VPNs are the WireGuard overlays the operator has configured. Each carries
	// a bring-your-own `.conf`; internal/vpn parses and applies them. When one
	// is up, its interface addresses are the kind of private overlay the
	// security checklist grades as a pass — see TrustedOverlays, which is where
	// those CIDRs are declared for grading.
	VPNs []VPN `json:"vpns,omitempty"`

	// VPNServers are the WireGuard servers this app has provisioned on its
	// machines, each with the devices enrolled on it. Distinct from VPNs (which
	// are bring-your-own client configs) because a provisioned server holds no
	// private key — see VPNServer.
	VPNServers []VPNServer `json:"vpnServers,omitempty"`

	// Orphans are containers a merge stopped managing but did NOT stop. They
	// are stored rather than recomputed: migrate() runs in memory and is only
	// written back by the next Save, so a derived notice would vanish on the
	// first save while the container kept serving.
	Orphans []OrphanedContainer `json:"orphanedContainers,omitempty"`

	AIProvider string `json:"aiProvider"` // ""|gemini|groq|ollama
	AIKey      string `json:"aiKey"`
	RefRPCBase string `json:"refRpcBase"` // default: defaultRefRPCBase

	// UpdateNotifyDisabled turns off proactive update notices. The zero value
	// (false) keeps notices on, so a fresh install checks in the background and
	// shows a banner when a newer release exists. When true ("don't prompt
	// me"), the app makes NO background calls to GitHub and shows no banner —
	// the operator checks on demand from the Settings page instead.
	UpdateNotifyDisabled bool `json:"updateNotifyDisabled,omitempty"`

	// TrustedOverlays are CIDR ranges of the operator's private overlay
	// networks (WireGuard, Tailscale, Headscale, Netbird, ZeroTier, …). The
	// security checklist grades a service bound to an address in one of these
	// ranges as a private overlay (pass) — reachable only on that authenticated
	// network — rather than warning as a LAN/public bind. Tailscale's
	// 100.64.0.0/10 is always trusted and need not be listed here.
	TrustedOverlays []string `json:"trustedOverlays,omitempty"`

	// ValveKeys is the OLD per-chain valve API key store.
	//
	// Deprecated: migration input only. A provider key is an account, not a
	// chain, so it collapses into ProviderKeys[ValveKeyPlaceholder] on load and
	// is cleared — see collapseValveKeys. Nothing but the migration may read it.
	ValveKeys map[int]string `json:"valveKeys,omitempty"`

	// ProviderKeys are API keys by PLACEHOLDER NAME — "VALVE_API_KEY",
	// "INFURA_API_KEY" — matching the ${NAME} slots the chain feed uses. Keyed
	// by placeholder rather than by chain because a provider key is an account,
	// not a chain.
	//
	// Secrets: stored here, never returned by the API. See settingsResponse,
	// which reports which placeholders are set and never their values.
	ProviderKeys map[string]string `json:"providerKeys,omitempty"`

	// Notices are one-off messages from a migration that the operator needs to
	// see, e.g. a key discarded when per-chain keys collapsed.
	//
	// NOT YET SHOWN ANYWHERE. This is a write-only record today: migrate appends
	// to it, config.json persists it, and no API field or screen reads it back.
	// The intent was "reported rather than silently dropped", and half of that is
	// built — the record exists and an operator (or a support request) can find it
	// in config.json — but the reporting half is not, so a discarded key is in
	// practice still discarded quietly. Surfacing it means a field on
	// settingsResponse, somewhere to render it, and a way to acknowledge one so it
	// stops reappearing; until that exists this comment says what the field IS
	// rather than what it was meant to be.
	//
	// Entries are deduped and PERSIST — migrate runs on every Load, so nothing
	// here may assume a notice is consumed.
	//
	// They are written for a screen, so nothing secret goes in one: a notice
	// about a discarded key names the chain and a masked fingerprint, never the
	// key itself.
	Notices []string `json:"notices,omitempty"`

	// Controller is this machine's signing identity, set by `jumpgate keys
	// init`. Nil until then.
	Controller *Controller `json:"controller,omitempty"`
}

// ValveKeyPlaceholder is the ${NAME} slot valve's own endpoints carry, and so
// the name ProviderKeys stores valve's key under. It is here rather than in
// catalog because it is the KEY's identity, not the endpoint set's.
const ValveKeyPlaceholder = "VALVE_API_KEY"

// DefaultGatewayID is the id given to a gateway migrated up from the old
// per-target Target.Gateway field, and the id the app offers first when
// creating one. ops maps exactly this id back to the historical container
// name, so an operator who had a gateway before gateways were a layer keeps
// the SAME running container rather than getting a second one alongside it.
const DefaultGatewayID = "default"

// FindGateway returns the gateway with this id.
func (c Config) FindGateway(id string) (Gateway, bool) {
	for _, g := range c.Gateways {
		if g.ID == id {
			return g, true
		}
	}
	return Gateway{}, false
}

// FindVPN returns the VPN overlay with this id.
func (c Config) FindVPN(id string) (VPN, bool) {
	for _, v := range c.VPNs {
		if v.ID == id {
			return v, true
		}
	}
	return VPN{}, false
}

// TrustedOverlayCIDRs is the full set of private-overlay networks the security
// grading should treat as authenticated — the operator's declared
// TrustedOverlays PLUS the subnet of every WireGuard server this app has
// PROVISIONED.
//
// The second part is the ingress link: if Jumpgate stood up a server on
// 10.9.0.1/24, then a gateway on that box bound to its overlay address is
// reachable only over that authenticated tunnel — exactly the "private overlay"
// bindAddrTier already grades as a pass. Deriving it from the provisioned server
// means the operator does not have to also hand-declare the same range in
// TrustedOverlays (and then keep the two in sync) just to stop the checklist
// warning about an ingress the app itself set up.
//
// A server address that does not parse is skipped rather than fatal — grading
// is advisory, and a malformed stored address should not take the checklist
// down. Results are de-duplicated so a range that is both declared and
// provisioned appears once.
func (c Config) TrustedOverlayCIDRs() []string {
	seen := map[string]bool{}
	var out []string
	add := func(cidr string) {
		cidr = strings.TrimSpace(cidr)
		if cidr == "" || seen[cidr] {
			return
		}
		seen[cidr] = true
		out = append(out, cidr)
	}
	for _, o := range c.TrustedOverlays {
		add(o)
	}
	for _, s := range c.VPNServers {
		// The server address is a host CIDR (10.9.0.1/24); grading wants the
		// network it sits on (10.9.0.0/24), so a bind anywhere on that overlay —
		// the server's own address or a peer's — grades as overlay.
		if _, ipnet, err := net.ParseCIDR(strings.TrimSpace(s.Address)); err == nil {
			add(ipnet.String())
		}
	}
	return out
}

// FindVPNServer returns the provisioned server with this id.
func (c Config) FindVPNServer(id string) (VPNServer, bool) {
	for _, s := range c.VPNServers {
		if s.ID == id {
			return s, true
		}
	}
	return VPNServer{}, false
}

// GatewaysOn returns the gateways placed on a target, in config order.
func (c Config) GatewaysOn(targetID string) []Gateway {
	var out []Gateway
	for _, g := range c.Gateways {
		if g.Placement.TargetID == targetID {
			out = append(out, g)
		}
	}
	return out
}

// migrate upgrades an older on-disk config in place.
//
// The one migration so far: a gateway stored on its target (Target.gateway)
// becomes a top-level gateway PLACED on that target. It is silent and
// lossless by construction — the whole catalog.GatewayConfig is carried over
// untouched, only its ownership changes — because the alternative is an
// operator opening the app after an upgrade to find the gateway they
// configured has vanished, while its container is still running and still
// serving.
//
// Ids: the first migrated gateway takes DefaultGatewayID, which is the id
// that maps back to the original container name, so nothing is orphaned. A
// second target that also carried a gateway cannot have that id (it would
// collide) and takes "<targetID>" instead — its container is renamed by the
// next provision, which is unavoidable: two containers cannot share a name,
// and they never could, which is precisely the bug this model fixes.
func (c *Config) migrate() {
	c.repointLegacyPaths()
	taken := make(map[string]bool, len(c.Gateways))
	for _, g := range c.Gateways {
		taken[g.ID] = true
	}

	for i := range c.Targets {
		lg := c.Targets[i].LegacyGateway
		if lg == nil {
			continue
		}
		c.Targets[i].LegacyGateway = nil

		id := DefaultGatewayID
		if taken[id] {
			id = c.Targets[i].ID
			for n := 2; taken[id]; n++ {
				id = fmt.Sprintf("%s-%d", c.Targets[i].ID, n)
			}
		}
		taken[id] = true

		gwCfg := *lg
		adoptDevnetReferences(&gwCfg, c.Targets[i])

		c.Gateways = append(c.Gateways, Gateway{
			ID: id,
			// Docker is the only backend the old per-target gateway surface
			// ever provisioned with (server/containers.go passed
			// setup.BackendDocker unconditionally), so this is what the
			// migrated gateway actually IS, not a guess.
			Placement: GatewayPlacement{TargetID: c.Targets[i].ID, Backend: "docker"},
			Config:    gwCfg,
		})
	}

	// One managed eRPC per device. Two gateways on one target mean two
	// containers, overlapping chains and two pollers against the same node.
	merged, orphans := mergeGatewaysPerTarget(c.Gateways)
	c.Gateways = merged
	// A leftover is identified by its container name AND the machine it is
	// running on, because a container name is only unique within one docker
	// engine. Two machines that each merged away a gateway with the same id
	// have two containers to clear, on two different boxes; keying the dedupe
	// on the name alone would swallow the second and leave it running with
	// nothing on any screen naming it.
	for _, o := range orphans {
		known := false
		for _, have := range c.Orphans {
			if have.ContainerName == o.ContainerName && have.TargetID == o.TargetID {
				known = true
				break
			}
		}
		if !known {
			c.Orphans = append(c.Orphans, o)
		}
	}

	c.collapseValveKeys()
}

// collapseValveKeys folds the old per-chain valve key into the one entry a
// provider key actually is.
//
// valveKeys was keyed by chain because valve's key sits in a URL path and the
// first cut read that as "a key belongs to a chain". It does not: a key is an
// account, and the same account answers every chain it is entitled to. Keyed by
// placeholder, it also lines up with every OTHER provider's key, which is what
// lets one store fill ${VALVE_API_KEY} and ${INFURA_API_KEY} alike.
//
// The lowest chain id wins, because it is the only tie-break that does not
// depend on map order. Anything else stored is REPORTED rather than dropped in
// silence — the same stance the orphan record takes, for the same reason: a
// migration that quietly destroys something an operator typed is indisting-
// uishable from a bug. The notice carries a masked fingerprint, not the key:
// notices are written to be shown, and this whole change exists to stop keys
// reaching a screen.
func (c *Config) collapseValveKeys() {
	if len(c.ValveKeys) == 0 {
		// Nil rather than an empty map, so a re-save does not write back a
		// deprecated field that is merely empty.
		c.ValveKeys = nil
		return
	}

	ids := make([]int, 0, len(c.ValveKeys))
	for id := range c.ValveKeys {
		ids = append(ids, id)
	}
	sort.Ints(ids)

	// An already-stored placeholder key wins over anything per-chain: it is the
	// newer shape, so it is the one the operator most recently meant.
	kept := strings.TrimSpace(c.ProviderKeys[ValveKeyPlaceholder])
	keptFrom := 0
	if kept == "" {
		for _, id := range ids {
			if v := strings.TrimSpace(c.ValveKeys[id]); v != "" {
				kept, keptFrom = v, id
				break
			}
		}
		if kept != "" {
			if c.ProviderKeys == nil {
				c.ProviderKeys = map[string]string{}
			}
			c.ProviderKeys[ValveKeyPlaceholder] = kept
		}
	}

	for _, id := range ids {
		v := strings.TrimSpace(c.ValveKeys[id])
		if v == "" || v == kept {
			continue
		}
		if keptFrom != 0 {
			c.notice(fmt.Sprintf(
				"Chain %d had a different valve API key (%s). Keys are per provider now, not per chain, so chain %d's was kept and this one was discarded — re-enter it under %s in Settings if it was the one you wanted.",
				id, maskSecret(v), keptFrom, ValveKeyPlaceholder))
			continue
		}
		c.notice(fmt.Sprintf(
			"Chain %d had a different valve API key (%s). Keys are per provider now, not per chain, so the %s you already have was kept and this one was discarded.",
			id, maskSecret(v), ValveKeyPlaceholder))
	}

	c.ValveKeys = nil
}

// notice records a migration message once.
//
// The dedupe is the same guard the orphan record carries, for the same reason:
// migrate runs on every Load and the notices are PERSISTED, so a message
// appended unconditionally would stack up one copy per read of a config that
// happens to have kept its old shape. Clearing ValveKeys makes a repeat
// unreachable by today's one caller — but "unreachable because the only caller
// happens to clear its input first" is not a property worth relying on in the
// place a second caller would be added.
func (c *Config) notice(msg string) {
	for _, have := range c.Notices {
		if have == msg {
			return
		}
	}
	c.Notices = append(c.Notices, msg)
}

// maskSecret renders enough of a key to recognise it and not enough to use it.
// Short values are hidden outright: with only a few characters, "enough to
// recognise" and "the whole thing" are the same string.
func maskSecret(s string) string {
	s = strings.TrimSpace(s)
	if len(s) <= 8 {
		return strings.Repeat("•", len(s))
	}
	return s[:3] + "…" + s[len(s)-3:]
}

// adoptDevnetReferences upgrades a frozen URL that IS this target's devnet
// into a managed-devnet reference.
//
// It is part of the migration rather than a separate nicety because the URL
// and the reference describe the same upstream, and only one of them survives
// the operator changing the devnet's port. Leaving the URL would carry the
// exact staleness bug references exist to remove straight across the upgrade
// — and would also label the machine's own devnet as a public endpoint, which
// is simply wrong.
//
// Only an exact match on the devnet's own HTTP endpoint is adopted. Anything
// else — including a URL that merely looks local — is left as the operator
// wrote it, because guessing at what an endpoint "probably meant" is how a
// migration silently repoints traffic.
func adoptDevnetReferences(g *catalog.GatewayConfig, t Target) {
	if t.Devnet == nil {
		return
	}
	want := strings.TrimSpace(t.Devnet.HTTPEndpoint())
	for i := range g.Networks {
		if g.Networks[i].ChainID != t.Devnet.ChainIDOrDefault() {
			continue
		}
		for j := range g.Networks[i].Upstreams {
			u := &g.Networks[i].Upstreams[j]
			if u.KindOrDefault() != catalog.UpstreamExternal {
				continue
			}
			if !strings.EqualFold(strings.TrimSpace(u.Endpoint), want) {
				continue
			}
			u.Kind = catalog.UpstreamManagedDevnet
			u.TargetID = t.ID
			// The stored URL goes: it is derived from here on, and keeping a
			// stale copy beside the reference is an invitation to read the
			// wrong one.
			u.Endpoint = ""
		}
	}
}

// configFileName is the file Load/Save read and write inside Dir().
const configFileName = "config.json"

// dirName is the controller's state directory under $HOME.
const dirName = ".jumpgate"

// legacyDirName is where releases before the rename kept the same state.
const legacyDirName = ".valve-node-app"

// Dir returns the directory jumpgate's local state lives in (~/.jumpgate),
// without creating it.
func Dir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("config: resolve home directory: %w", err)
	}
	return filepath.Join(home, dirName), nil
}

// ConfirmedHostsFile is the host-key store Strict reads: keys a person
// explicitly confirmed. Only executor.RecordHostKey after such a confirmation
// writes it; trust-on-first-use writes Target.SSH.HostKeyFile instead and is
// never consulted for strict checks.
func ConfirmedHostsFile() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "confirmed_hosts"), nil
}

// TightenState restricts ~/.jumpgate and every secret already in it to the
// owner (ruling P29). Writes go through fsperm, but a file that is never
// rewritten keeps whatever permissions it had: a transport key made by an
// older release, files moved over from ~/.valve-node-app, a known_hosts
// another user can append to. The server runs this once at startup. keyFile
// is the controller key's path when it is kept in a file, else "".
//
// Missing files are skipped. It returns the files it had to tighten, so the
// caller can say so (other users may already have read them), and warnings
// for files it could not fix, which do not stop the server. The two signing
// keys are different: if the configured key file or the transport key cannot
// be made private, err names it and the server must not start, since a key
// is never to be left open to other users and used anyway.
func TightenState(keyFile string) (tightened []string, warnings []error, err error) {
	dir, err := Dir()
	if err != nil {
		return nil, nil, err
	}
	if err := fsperm.MkdirPrivate(dir); err != nil {
		warnings = append(warnings, fmt.Errorf("config: restrict %s: %w", dir, err))
	}
	// The transport key's name is internal/server's (transportKeyPath).
	transportKey := filepath.Join(dir, "ssh", "jumpgate_ed25519")
	paths := []string{
		filepath.Join(dir, configFileName),
		filepath.Join(dir, "run", "server.json"),
		filepath.Join(dir, "confirmed_hosts"),
		filepath.Join(dir, "known_hosts"),
	}
	// ssh/ holds the transport key, keys/ the default controller key file.
	for _, sub := range []string{"run", "ssh", "keys"} {
		d := filepath.Join(dir, sub)
		if _, err := os.Lstat(d); errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err := fsperm.MkdirPrivate(d); err != nil {
			warnings = append(warnings, fmt.Errorf("config: restrict %s: %w", d, err))
		}
		if sub == "run" {
			continue // only server.json in it is a secret; the socket is restricted when it is made
		}
		entries, err := os.ReadDir(d)
		if err != nil {
			warnings = append(warnings, fmt.Errorf("config: read %s: %w", d, err))
			continue
		}
		for _, e := range entries {
			if p := filepath.Join(d, e.Name()); !slices.Contains(paths, p) {
				paths = append(paths, p)
			}
		}
	}
	if keyFile != "" && !slices.Contains(paths, keyFile) {
		paths = append(paths, keyFile)
	}
	for _, p := range paths {
		if _, err := os.Lstat(p); errors.Is(err, os.ErrNotExist) {
			continue
		} else if err != nil {
			warnings = append(warnings, fmt.Errorf("config: inspect %s: %w", p, err))
			continue
		}
		if fsperm.CheckPrivate(p) == nil {
			continue
		}
		perr := fsperm.MakePrivate(p)
		if perr == nil {
			perr = fsperm.CheckPrivate(p) // made private, or still not?
		}
		if perr != nil {
			if p == keyFile || p == transportKey {
				return tightened, warnings, fmt.Errorf("config: %s holds a signing key and other users can read or change it, and jumpgate could not restrict it to you (%v); make it a regular file only you can read (chmod 600 on macOS and Linux; Properties > Security on Windows), then start jumpgate again", p, perr)
			}
			warnings = append(warnings, fmt.Errorf("config: restrict %s: %w", p, perr))
			continue
		}
		tightened = append(tightened, p)
	}
	return tightened, warnings, nil
}

// MigrateLegacyDir moves ~/.valve-node-app to ~/.jumpgate once, and leaves a
// MOVED pointer file in the old place. It is a rename, not a copy, so secrets
// (provider keys, VPN private keys) never exist twice on disk. It reports
// whether it moved anything. Both directories holding a config is an error:
// merging two configs silently would lose one of them.
//
// ~/.jumpgate may already exist without a config: any command that takes the
// config lock, the server's run directory, a confirmed host key or a new
// controller key creates it first. Then the legacy entries are moved in one
// by one (R24), descending into directories both sides hold. A jumpgate-made
// file is never overwritten: on a clash the legacy copy stays where it is and
// its path is returned in kept, and repointLegacyPaths leaves a stored path to
// it alone. config.json moves last, so an interrupted merge resumes on the
// next run.
func MigrateLegacyDir() (moved bool, kept []string, err error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return false, nil, fmt.Errorf("config: resolve home directory: %w", err)
	}
	legacy := filepath.Join(home, legacyDirName)
	current := filepath.Join(home, dirName)

	if _, err := os.Stat(filepath.Join(legacy, configFileName)); errors.Is(err, os.ErrNotExist) {
		return false, nil, nil
	} else if err != nil {
		return false, nil, fmt.Errorf("config: inspect %s: %w", legacy, err)
	}
	if _, err := os.Stat(filepath.Join(current, configFileName)); err == nil {
		return false, nil, fmt.Errorf("config: both %s and %s hold a config.json; keep one and remove the other", legacy, current)
	}
	// An empty ~/.jumpgate would make the rename fail; remove it only if it
	// is empty.
	_ = os.Remove(current)
	if _, err := os.Lstat(current); errors.Is(err, os.ErrNotExist) {
		if err := os.Rename(legacy, current); err != nil {
			return false, nil, fmt.Errorf("config: move %s to %s: %w", legacy, current, err)
		}
	} else {
		if kept, err = mergeDir(legacy, current, configFileName); err != nil {
			return false, kept, err
		}
		if err := os.Rename(filepath.Join(legacy, configFileName), filepath.Join(current, configFileName)); err != nil {
			return false, kept, fmt.Errorf("config: move %s: %w", configFileName, err)
		}
	}
	if err := os.MkdirAll(legacy, 0o700); err != nil {
		return true, kept, fmt.Errorf("config: recreate %s for the pointer file: %w", legacy, err)
	}
	note := "jumpgate moved this directory to " + current + "\n"
	for _, k := range kept {
		note += "left here because " + current + " already had one: " + k + "\n"
	}
	if err := os.WriteFile(filepath.Join(legacy, "MOVED"), []byte(note), 0o600); err != nil {
		return true, kept, fmt.Errorf("config: write pointer file: %w", err)
	}
	return true, kept, nil
}

// mergeDir moves every entry of src into dst that dst does not already have,
// recursing into directories both hold, and returns the src paths it left in
// place because dst had an entry of that name. skip names a top-level entry
// the caller moves itself. Emptied source directories are removed.
func mergeDir(src, dst, skip string) ([]string, error) {
	entries, err := os.ReadDir(src)
	if err != nil {
		return nil, fmt.Errorf("config: read %s: %w", src, err)
	}
	var kept []string
	for _, e := range entries {
		if e.Name() == skip {
			continue
		}
		from, to := filepath.Join(src, e.Name()), filepath.Join(dst, e.Name())
		toInfo, err := os.Lstat(to)
		if errors.Is(err, os.ErrNotExist) {
			if err := os.Rename(from, to); err != nil {
				return kept, fmt.Errorf("config: move %s to %s: %w", from, to, err)
			}
			continue
		}
		if err != nil {
			return kept, fmt.Errorf("config: inspect %s: %w", to, err)
		}
		// Lstat-based: a symlink on either side is a clash, never followed.
		if e.Type().IsDir() && toInfo.IsDir() {
			more, err := mergeDir(from, to, "")
			kept = append(kept, more...)
			if err != nil {
				return kept, err
			}
			_ = os.Remove(from) // only succeeds once it is empty
			continue
		}
		kept = append(kept, from)
	}
	return kept, nil
}

func filePath() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, configFileName), nil
}

// Load reads Config from ~/.jumpgate/config.json. A missing file is not an
// error: it returns the zero Config (with RefRPCBase defaulted). RefRPCBase
// is defaulted whenever it's empty, whether that's because the file doesn't
// exist yet or because a stored config happens to have it blank.
//
// Every load runs migrate, so an older file is upgraded in memory before any
// caller sees it and is written back in the new shape by the next Save. The
// upgrade is not conditional on a version field: the migrations are all
// "move this if it is present", which is idempotent, and a version field
// would only add a second thing that can be wrong.
func Load() (Config, error) {
	lp, err := lockPath()
	if err != nil {
		return Config{}, err
	}
	h, err := filelock.Lock(lp, false)
	if err != nil {
		return Config{}, fmt.Errorf("config: lock: %w", err)
	}
	defer h.Unlock()
	return load()
}

// lockFileName sits beside config.json. The lock is on a separate file
// because Save replaces config.json by rename, and a lock on the old inode
// would not exclude a writer that opened the new one.
const lockFileName = "config.json.lock"

func lockPath() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	if err := fsperm.MkdirPrivate(dir); err != nil {
		return "", fmt.Errorf("config: create %s: %w", dir, err)
	}
	return filepath.Join(dir, lockFileName), nil
}

// Update loads the config, applies fn, and saves it, holding an exclusive
// lock across all three so another process's edit can never be lost between
// this one's read and write. If fn fails nothing is saved.
func Update(fn func(*Config) error) (Config, error) {
	lp, err := lockPath()
	if err != nil {
		return Config{}, err
	}
	h, err := filelock.Lock(lp, true)
	if err != nil {
		return Config{}, fmt.Errorf("config: lock: %w", err)
	}
	defer h.Unlock()

	c, err := load()
	if err != nil {
		return Config{}, err
	}
	if err := fn(&c); err != nil {
		return Config{}, err
	}
	if err := c.Save(); err != nil {
		return Config{}, err
	}
	return c, nil
}

// load is Load without the lock; callers must hold it.
func load() (Config, error) {
	path, err := filePath()
	if err != nil {
		return Config{}, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Config{RefRPCBase: defaultRefRPCBase}, nil
		}
		return Config{}, fmt.Errorf("config: read %s: %w", path, err)
	}

	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return Config{}, fmt.Errorf("config: parse %s: %w", path, err)
	}
	if c.RefRPCBase == "" {
		c.RefRPCBase = defaultRefRPCBase
	}
	c.migrate()
	return c, nil
}

// Save writes c to ~/.jumpgate/config.json, creating the directory if
// needed. The write is atomic and owner-only (fsperm.WriteFilePrivate), since
// the file may contain AI provider API keys and VPN private keys.
func (c Config) Save() error {
	dir, err := Dir()
	if err != nil {
		return err
	}
	if err := fsperm.MkdirPrivate(dir); err != nil {
		return fmt.Errorf("config: create %s: %w", dir, err)
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("config: marshal: %w", err)
	}
	if err := fsperm.WriteFilePrivate(filepath.Join(dir, configFileName), data); err != nil {
		return fmt.Errorf("config: write: %w", err)
	}
	return nil
}

// repointLegacyPaths rewrites every stored LOCAL path that lies inside the
// legacy ~/.valve-node-app directory to the same relative path under
// ~/.jumpgate. Paths are stored absolute (handleAddTarget joins them onto
// Dir()), so after MigrateLegacyDir moves the directory they would point at
// nothing; for HostKeyFile that is a security failure, not a nuisance, because
// trust-on-first-use recreates the missing file empty and then trusts whatever
// key the host presents.
//
// The local-path fields are Target.SSH.HostKeyFile and Target.SSH.KeyPath;
// nothing else in Config holds a controller-side path (gateway, devnet, wire
// and VPN configs name on-box paths or carry inline text). On-box paths such
// as /var/lib/valve-node-app are outside the legacy directory and untouched.
// Idempotent: a repointed path no longer lies inside the legacy directory. A
// path whose file still exists in the legacy directory (left there by a merge
// clash, or not yet migrated) is kept as is.
func (c *Config) repointLegacyPaths() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	legacy := filepath.Join(home, legacyDirName)
	current := filepath.Join(home, dirName)
	repoint := func(p string) string {
		if p == legacy {
			return current
		}
		if rest, ok := strings.CutPrefix(p, legacy+string(filepath.Separator)); ok {
			// A file MigrateLegacyDir left in place on a clash is still
			// where the stored path says; pointing it at the other copy
			// would swap a pinned host key for a different file.
			if _, err := os.Lstat(p); err == nil {
				return p
			}
			return filepath.Join(current, rest)
		}
		return p
	}
	for i := range c.Targets {
		if ssh := c.Targets[i].SSH; ssh != nil {
			ssh.HostKeyFile = repoint(ssh.HostKeyFile)
			ssh.KeyPath = repoint(ssh.KeyPath)
		}
	}
}
