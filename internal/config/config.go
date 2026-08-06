// Package config persists valve-node-app's own local state — the targets it
// knows how to manage, and the AI provider it's configured to use for log
// explanations — to a single JSON file under the user's home directory. It
// performs no validation of the domain data it stores (that's the caller's
// job); it only knows how to read and write the file safely.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/valve-tech/valve-node-app/internal/catalog"
	"github.com/valve-tech/valve-node-app/internal/executor"
)

// defaultRefRPCBase is the public demo-key reference RPC base URL, used
// whenever a Config has no explicit override. Callers append "/evm/<chainId>"
// to get the per-chain reference endpoint.
const defaultRefRPCBase = "https://rpc.valve.city/v1/vk_Et-4emAlBIym1PjiCogh5p7IuGtS-Rpj"

// Target is one machine valve-node-app can set up and monitor a node on.
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

// Config is valve-node-app's persisted local state.
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

	// Orphans are containers a merge stopped managing but did NOT stop. They
	// are stored rather than recomputed: migrate() runs in memory and is only
	// written back by the next Save, so a derived notice would vanish on the
	// first save while the container kept serving.
	Orphans []OrphanedContainer `json:"orphanedContainers,omitempty"`

	AIProvider string `json:"aiProvider"` // ""|gemini|groq|ollama
	AIKey      string `json:"aiKey"`
	RefRPCBase string `json:"refRpcBase"` // default: defaultRefRPCBase

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

// Dir returns the directory valve-node-app's local state lives in
// (~/.valve-node-app), without creating it.
func Dir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("config: resolve home directory: %w", err)
	}
	return filepath.Join(home, ".valve-node-app"), nil
}

func filePath() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, configFileName), nil
}

// Load reads Config from ~/.valve-node-app/config.json. A missing file is not an
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

// Save writes c to ~/.valve-node-app/config.json, creating the directory if
// needed. The write is atomic (write to a temp file in the same directory,
// then rename over the target) and the file is mode 0600, since it may
// contain an AI provider API key.
func (c Config) Save() error {
	dir, err := Dir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("config: create %s: %w", dir, err)
	}

	path := filepath.Join(dir, configFileName)

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("config: marshal: %w", err)
	}

	tmp, err := os.CreateTemp(dir, configFileName+".tmp-*")
	if err != nil {
		return fmt.Errorf("config: create temp file: %w", err)
	}
	tmpPath := tmp.Name()
	// If anything below fails before the rename, don't leave the temp file
	// behind.
	success := false
	defer func() {
		if !success {
			os.Remove(tmpPath)
		}
	}()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("config: write temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("config: close temp file: %w", err)
	}
	if err := os.Chmod(tmpPath, 0o600); err != nil {
		return fmt.Errorf("config: chmod temp file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("config: rename into place: %w", err)
	}
	success = true
	return nil
}
