package config

// Load and Save as a pair, with the properties that matter on disk: a missing
// file is not an error, a corrupt one IS, the write is atomic, the mode is
// 0600 because the file may hold an AI provider key, and no temp file is left
// behind when the write fails partway.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/valve-tech/valve-node-app/internal/catalog"
	"github.com/valve-tech/valve-node-app/internal/executor"
)

// isolate points HOME at a fresh temp dir, so these never touch the real
// ~/.valve-node-app.
func isolate(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", home)
	}
	return home
}

func TestDir_IsUnderHomeAndIsNotCreated(t *testing.T) {
	home := isolate(t)

	dir, err := Dir()
	if err != nil {
		t.Fatalf("Dir: %v", err)
	}
	if dir != filepath.Join(home, ".valve-node-app") {
		t.Errorf("Dir = %q, want it under HOME", dir)
	}
	// Reading where state WOULD live must not create it — Dir is called on
	// paths that only want the name.
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("Dir created the directory as a side effect: %v", err)
	}
}

// A missing file is the first-run case, not an error: returning one would
// make every read fail until something happened to write.
func TestLoad_AMissingFileIsAFreshConfig(t *testing.T) {
	isolate(t)

	c, err := Load()
	if err != nil {
		t.Fatalf("a missing config file was an error: %v", err)
	}
	if len(c.Targets) != 0 || len(c.Gateways) != 0 {
		t.Errorf("a fresh config is not empty: %+v", c)
	}
	// RefRPCBase is defaulted here rather than at every use site.
	if c.RefRPCBase != defaultRefRPCBase {
		t.Errorf("RefRPCBase = %q, want %q", c.RefRPCBase, defaultRefRPCBase)
	}
}

// A corrupt file IS an error. Silently returning a fresh config would look
// exactly like first run, and the next Save would overwrite whatever the
// operator actually had.
func TestLoad_ACorruptFileIsRefusedRatherThanTreatedAsEmpty(t *testing.T) {
	isolate(t)
	dir, err := Dir()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, configFileName)
	if err := os.WriteFile(path, []byte("{ this is not json"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := Load(); err == nil {
		t.Fatal("a corrupt config was read as an empty one, which the next Save would overwrite")
	} else if !strings.Contains(err.Error(), path) {
		t.Errorf("the error does not name the file to go fix: %v", err)
	}

	// And the bad file is still there to be recovered by hand.
	if _, err := os.Stat(path); err != nil {
		t.Errorf("the unreadable config was destroyed: %v", err)
	}
}

// An empty RefRPCBase is defaulted whether it came from a missing file or
// from a stored config that happens to have it blank.
func TestLoad_ABlankRefRPCBaseIsDefaulted(t *testing.T) {
	isolate(t)

	if err := (Config{RefRPCBase: ""}).Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}
	c, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if c.RefRPCBase != defaultRefRPCBase {
		t.Errorf("RefRPCBase = %q, want it defaulted on read", c.RefRPCBase)
	}
}

// The file may hold an AI provider API key, so it is 0600 and its directory
// 0700 — created that way rather than fixed up afterwards.
func TestSave_WritesPrivatelyAndCreatesItsDirectory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix file modes")
	}
	isolate(t)

	if err := (Config{AIKey: "sk-secret"}).Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	dir, err := Dir()
	if err != nil {
		t.Fatal(err)
	}
	di, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("the config directory was not created: %v", err)
	}
	if perm := di.Mode().Perm(); perm != 0o700 {
		t.Errorf("directory mode = %04o, want 0700", perm)
	}

	fi, err := os.Stat(filepath.Join(dir, configFileName))
	if err != nil {
		t.Fatalf("stat config: %v", err)
	}
	if perm := fi.Mode().Perm(); perm != 0o600 {
		t.Errorf("file mode = %04o, want 0600 — this file can hold an API key", perm)
	}
}

// Save leaves no temp file behind. The write goes to a temp file in the same
// directory and is renamed over the target, so a reader never sees a partial
// config — but the debris from that has to be cleaned up too.
func TestSave_LeavesNoTempFilesBehind(t *testing.T) {
	isolate(t)

	for i := 0; i < 3; i++ {
		if err := (Config{RefRPCBase: "https://rpc.example.com"}).Save(); err != nil {
			t.Fatalf("Save: %v", err)
		}
	}

	dir, err := Dir()
	if err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.Contains(e.Name(), ".tmp-") {
			t.Errorf("a temp file survived the write: %s", e.Name())
		}
	}
}

// The round trip is the actual contract: what went in comes back out.
func TestSaveLoad_RoundTrips(t *testing.T) {
	isolate(t)

	want := Config{
		RefRPCBase: "https://rpc.example.com",
		AIProvider: "anthropic",
		AIKey:      "sk-secret",
		Targets: []Target{{
			ID:   "box",
			Mode: "ssh",
			SSH:  &executor.SSHConfig{Host: "10.0.0.9", User: "root", KeyPath: "/root/.ssh/id_ed25519"},
		}},
	}
	if err := want.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.RefRPCBase != want.RefRPCBase || got.AIProvider != want.AIProvider || got.AIKey != want.AIKey {
		t.Errorf("settings did not round trip: %+v", got)
	}
	if len(got.Targets) != 1 || got.Targets[0].ID != "box" {
		t.Fatalf("targets did not round trip: %+v", got.Targets)
	}
	if got.Targets[0].SSH == nil || got.Targets[0].SSH.Host != "10.0.0.9" {
		t.Errorf("the ssh config did not round trip: %+v", got.Targets[0].SSH)
	}
}

// A save over an existing config REPLACES it rather than merging, and the
// file on disk is valid JSON at every point a reader could look.
func TestSave_OverwritesAtomically(t *testing.T) {
	isolate(t)

	if err := (Config{Targets: []Target{{ID: "first", Mode: "local"}}}).Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := (Config{Targets: []Target{{ID: "second", Mode: "local"}}}).Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	dir, _ := Dir()
	data, err := os.ReadFile(filepath.Join(dir, configFileName))
	if err != nil {
		t.Fatal(err)
	}
	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		t.Fatalf("the file on disk is not valid JSON: %v", err)
	}
	if len(c.Targets) != 1 || c.Targets[0].ID != "second" {
		t.Errorf("the second save did not replace the first: %+v", c.Targets)
	}
}

// ---------------------------------------------------------------------
// the failure paths that are actually reachable
// ---------------------------------------------------------------------

// With no home directory there is nowhere for state to live, and every entry
// point has to say so rather than writing to a relative path.
func TestNoHomeDirectoryIsAnErrorEverywhere(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("HOME is not how Windows resolves the user directory")
	}
	t.Setenv("HOME", "")

	if _, err := Dir(); err == nil {
		t.Error("Dir resolved a state directory with no home")
	} else if !strings.Contains(err.Error(), "home") {
		t.Errorf("the error does not say what is missing: %v", err)
	}
	if _, err := Load(); err == nil {
		t.Error("Load read a config with no home directory")
	}
	if err := (Config{}).Save(); err == nil {
		t.Error("Save wrote a config with no home directory")
	}
}

// A read failure that is NOT "missing" must surface. Silently treating an
// unreadable config as absent is the same overwrite risk as a corrupt one.
func TestLoad_AnUnreadableConfigIsNotTreatedAsMissing(t *testing.T) {
	isolate(t)
	dir, err := Dir()
	if err != nil {
		t.Fatal(err)
	}
	// A directory where the file should be: reading it fails with something
	// other than ErrNotExist.
	if err := os.MkdirAll(filepath.Join(dir, configFileName), 0o700); err != nil {
		t.Fatal(err)
	}

	if _, err := Load(); err == nil {
		t.Fatal("an unreadable config was reported as a fresh one")
	}
}

// The state directory cannot be created if something else is already there
// under that name, and Save has to report it instead of losing the write.
func TestSave_ReportsADirectoryItCannotCreate(t *testing.T) {
	home := isolate(t)
	// A regular file where the state directory belongs.
	if err := os.WriteFile(filepath.Join(home, ".valve-node-app"), []byte("in the way"), 0o600); err != nil {
		t.Fatal(err)
	}

	err := (Config{}).Save()
	if err == nil {
		t.Fatal("Save reported success with nowhere to write")
	}
	if !strings.Contains(err.Error(), ".valve-node-app") {
		t.Errorf("the error does not name the path in the way: %v", err)
	}
}

// ---------------------------------------------------------------------
// migrate
// ---------------------------------------------------------------------

// A legacy per-target gateway becomes a top-level one. When the default id is
// already taken, the migrated gateway gets a distinct id rather than
// colliding — two gateways sharing an id would share a container name.
func TestMigrate_LegacyGatewaysGetDistinctIDs(t *testing.T) {
	legacy := func() *catalog.GatewayConfig {
		return &catalog.GatewayConfig{Port: 4100, Networks: []catalog.GatewayNetwork{
			{ChainID: 369, Upstreams: []catalog.GatewayUpstream{{ID: "n1", Endpoint: "https://rpc.example.com"}}},
		}}
	}

	c := Config{
		// The default id is already spoken for, and so is the fallback that
		// would be derived from the first target's own id. Each pre-existing
		// gateway is placed on its own machine, distinct from "box" and
		// "other" below — migrate() now also folds gateways sharing a
		// Placement.TargetID into one (the one-erpc-per-device invariant),
		// and this test is about id collision, not that merge.
		Gateways: []Gateway{
			{ID: DefaultGatewayID, Placement: GatewayPlacement{TargetID: "existing-a"}},
			{ID: "box", Placement: GatewayPlacement{TargetID: "existing-b"}},
		},
		Targets: []Target{
			{ID: "box", Mode: "local", LegacyGateway: legacy()},
			{ID: "other", Mode: "local", LegacyGateway: legacy()},
		},
	}
	c.migrate()

	seen := map[string]bool{}
	for _, g := range c.Gateways {
		if seen[g.ID] {
			t.Fatalf("two gateways share the id %q, so they would share a container", g.ID)
		}
		seen[g.ID] = true
	}
	if len(c.Gateways) != 4 {
		t.Fatalf("got %d gateways, want the 2 existing plus 2 migrated", len(c.Gateways))
	}

	// The legacy field is cleared, so a second migrate is a no-op rather
	// than a second set of gateways.
	for _, tg := range c.Targets {
		if tg.LegacyGateway != nil {
			t.Errorf("target %q still carries its legacy gateway", tg.ID)
		}
	}
	before := len(c.Gateways)
	c.migrate()
	if len(c.Gateways) != before {
		t.Errorf("migrate is not idempotent: %d then %d gateways", before, len(c.Gateways))
	}

	// The migrated gateway is placed on its own target, over docker — the
	// only backend the old per-target surface ever provisioned with.
	for _, g := range c.Gateways[2:] {
		if g.Placement.TargetID == "" {
			t.Errorf("gateway %q was migrated onto no machine", g.ID)
		}
		if g.Placement.Backend != "docker" {
			t.Errorf("gateway %q backend = %q, want docker", g.ID, g.Placement.Backend)
		}
	}
}

// A frozen URL that IS this target's devnet becomes a managed reference, so
// changing the devnet's port cannot leave the gateway pointing at a dead one.
func TestAdoptDevnetReferences_OnlyAnExactMatchIsAdopted(t *testing.T) {
	dev := &catalog.DevnetConfig{HTTPPort: 18545, WSPort: 18546}
	target := Target{ID: "box", Devnet: dev}
	own := dev.HTTPEndpoint()

	g := catalog.GatewayConfig{Networks: []catalog.GatewayNetwork{
		{ChainID: dev.ChainIDOrDefault(), Upstreams: []catalog.GatewayUpstream{
			{ID: "the-devnet", Endpoint: own},
			// Merely local-looking is NOT adopted: guessing at what an
			// endpoint "probably meant" is how a migration silently
			// repoints traffic.
			{ID: "something-else", Endpoint: "http://127.0.0.1:9999"},
			// An already-managed upstream is left alone.
			{ID: "managed", Kind: catalog.UpstreamManagedNode, TargetID: "box"},
		}},
		// A different chain is never touched, even on a matching URL.
		{ChainID: 369, Upstreams: []catalog.GatewayUpstream{{ID: "other-chain", Endpoint: own}}},
	}}

	adoptDevnetReferences(&g, target)

	ups := g.Networks[0].Upstreams
	if ups[0].Kind != catalog.UpstreamManagedDevnet || ups[0].TargetID != "box" {
		t.Errorf("the devnet's own endpoint was not adopted: %+v", ups[0])
	}
	// The stale URL goes, or there are two sources of truth for one address.
	if ups[0].Endpoint != "" {
		t.Errorf("the frozen URL survived beside the reference: %q", ups[0].Endpoint)
	}
	if ups[1].Kind == catalog.UpstreamManagedDevnet {
		t.Error("an unrelated local URL was adopted as the devnet")
	}
	if ups[2].Kind != catalog.UpstreamManagedNode {
		t.Errorf("an already-managed upstream was rewritten: %+v", ups[2])
	}
	if other := g.Networks[1].Upstreams[0]; other.Kind == catalog.UpstreamManagedDevnet {
		t.Error("a matching URL on a DIFFERENT chain was adopted")
	}
}

// valveKeys was per chain because valve's key sits in a URL path. That is wrong
// as a general rule — a provider key is an account, not a chain — so it
// collapses to one entry under the placeholder name.
func TestMigrateCollapsesValveKeysToAProviderKey(t *testing.T) {
	c := Config{ValveKeys: map[int]string{1: "vk_mine", 369: "vk_mine"}}

	c.migrate()

	if got := c.ProviderKeys[ValveKeyPlaceholder]; got != "vk_mine" {
		t.Errorf("VALVE_API_KEY = %q, want vk_mine", got)
	}
	if c.ValveKeys != nil {
		t.Errorf("valveKeys must be cleared once migrated, got %+v", c.ValveKeys)
	}
	// Every chain agreed, so there is nothing to report.
	if len(c.Notices) != 0 {
		t.Errorf("agreeing keys are not news: %+v", c.Notices)
	}

	// Disagreeing keys: the first by chain id wins and the rest are REPORTED,
	// never silently dropped — the same stance the orphan record takes.
	d := Config{ValveKeys: map[int]string{1: "vk_a", 369: "vk_b"}}
	d.migrate()
	if d.ProviderKeys[ValveKeyPlaceholder] != "vk_a" {
		t.Errorf("lowest chain id wins, got %q", d.ProviderKeys[ValveKeyPlaceholder])
	}
	if len(d.Notices) == 0 {
		t.Error("a discarded key must be reported to the operator, not dropped in silence")
	}
}

// A notice is written to be shown, so it names the chain and a fingerprint —
// never the key. Reporting a discarded secret by printing it would put the key
// on exactly the screen this change exists to keep it off.
func TestMigrateNoticesNameTheChainAndNotTheKey(t *testing.T) {
	c := Config{ValveKeys: map[int]string{
		1:   "vk_kept_0123456789",
		369: "vk_discarded_0123456789",
	}}

	c.migrate()

	if len(c.Notices) != 1 {
		t.Fatalf("one discarded key, one notice: %+v", c.Notices)
	}
	n := c.Notices[0]
	if !strings.Contains(n, "369") {
		t.Errorf("the notice does not say which chain lost its key: %q", n)
	}
	if strings.Contains(n, "vk_discarded_0123456789") {
		t.Errorf("the notice quotes the key verbatim: %q", n)
	}
}

// An already-stored placeholder key is the newer shape, so it survives — and
// every per-chain value that disagrees with it is reported rather than
// overwriting it.
func TestMigrateKeepsAnAlreadyStoredProviderKey(t *testing.T) {
	c := Config{
		ProviderKeys: map[string]string{ValveKeyPlaceholder: "vk_current"},
		ValveKeys:    map[int]string{1: "vk_old"},
	}

	c.migrate()

	if got := c.ProviderKeys[ValveKeyPlaceholder]; got != "vk_current" {
		t.Errorf("the stored placeholder key was overwritten by a legacy one: %q", got)
	}
	if len(c.Notices) != 1 {
		t.Errorf("the displaced per-chain key must be reported: %+v", c.Notices)
	}
	if c.ValveKeys != nil {
		t.Errorf("valveKeys must be cleared once migrated, got %+v", c.ValveKeys)
	}
}

// migrate runs on every Load, so a second pass must not re-report what the
// first already collapsed.
func TestMigrateCollapseIsIdempotent(t *testing.T) {
	c := Config{ValveKeys: map[int]string{1: "vk_a", 369: "vk_b"}}
	c.migrate()
	before := len(c.Notices)

	c.migrate()

	if len(c.Notices) != before {
		t.Errorf("a second migrate re-reported the same discard: %d then %d notices", before, len(c.Notices))
	}
	if c.ProviderKeys[ValveKeyPlaceholder] != "vk_a" {
		t.Errorf("the collapsed key did not survive a second migrate: %q", c.ProviderKeys[ValveKeyPlaceholder])
	}
}

// Notices are PERSISTED and migrate runs on every Load, so recording one is
// idempotent by message — the same guard the orphan record carries, for the
// same reason. Clearing ValveKeys makes a repeat unreachable by today's one
// caller, which is not a property worth relying on where a second would be
// added.
func TestNoticeIsRecordedOnce(t *testing.T) {
	c := Config{}

	c.notice("the same thing happened")
	c.notice("the same thing happened")
	c.notice("something else happened")

	if len(c.Notices) != 2 {
		t.Fatalf("a repeated message must be recorded once: %+v", c.Notices)
	}
}

// No devnet on the target means nothing to adopt, and the config is untouched.
func TestAdoptDevnetReferences_NoDevnetLeavesEverythingAlone(t *testing.T) {
	g := catalog.GatewayConfig{Networks: []catalog.GatewayNetwork{
		{ChainID: 1337, Upstreams: []catalog.GatewayUpstream{{ID: "n1", Endpoint: "http://127.0.0.1:18545"}}},
	}}
	before := g.Networks[0].Upstreams[0]

	adoptDevnetReferences(&g, Target{ID: "box"})

	if g.Networks[0].Upstreams[0] != before {
		t.Errorf("a target with no devnet still rewrote an upstream: %+v", g.Networks[0].Upstreams[0])
	}
}
