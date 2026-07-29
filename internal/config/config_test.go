package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/valve-tech/valve-node-app/internal/catalog"
	"github.com/valve-tech/valve-node-app/internal/executor"
)

// catalog cannot import config (config already imports catalog), so
// catalog.ValveKeyPlaceholder duplicates this constant's VALUE rather than
// referencing it. Nothing in the type system keeps the two literals equal —
// this test is that guard.
func TestValveKeyPlaceholderMatchesCatalog(t *testing.T) {
	if catalog.ValveKeyPlaceholder != ValveKeyPlaceholder {
		t.Errorf("catalog.ValveKeyPlaceholder = %q, config.ValveKeyPlaceholder = %q — the two must name the same placeholder",
			catalog.ValveKeyPlaceholder, ValveKeyPlaceholder)
	}
}

func TestLoadMissingReturnsZeroValueWithDefaultRefRPCBase(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	c, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(c.Targets) != 0 {
		t.Errorf("Targets = %+v, want empty", c.Targets)
	}
	if c.AIProvider != "" {
		t.Errorf("AIProvider = %q, want empty", c.AIProvider)
	}
	if c.RefRPCBase != defaultRefRPCBase {
		t.Errorf("RefRPCBase = %q, want default %q", c.RefRPCBase, defaultRefRPCBase)
	}
}

func TestSaveThenLoadRoundTrips(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	want := Config{
		Targets: []Target{
			{ID: "local", Mode: "local"},
			{
				ID:   "box1",
				Mode: "ssh",
				SSH: &executor.SSHConfig{
					Host:        "1.2.3.4",
					User:        "root",
					KeyPath:     "/home/me/.ssh/id_ed25519",
					HostKeyFile: "/home/me/.valve-node-app/known_hosts",
					Port:        2222,
				},
				Wire: &catalog.WireConfig{
					ChainID:  369,
					ExecID:   "reth",
					BeaconID: "lighthouse-pulse",
					DataDir:  "/var/lib/valve-node-app/369",
					JWTPath:  "/var/lib/valve-node-app/369/jwt.hex",
					Archive:  true,
				},
			},
		},
		AIProvider: "gemini",
		AIKey:      "secret-key",
		RefRPCBase: "https://rpc.valve.city/v1/vk_custom",
	}

	if err := want.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(got.Targets) != 2 {
		t.Fatalf("Targets = %+v, want 2 entries", got.Targets)
	}
	if got.Targets[0].ID != "local" || got.Targets[0].Mode != "local" {
		t.Errorf("Targets[0] = %+v, want local/local", got.Targets[0])
	}
	if got.Targets[1].SSH == nil || got.Targets[1].SSH.Host != "1.2.3.4" || got.Targets[1].SSH.Port != 2222 {
		t.Errorf("Targets[1].SSH = %+v, want host 1.2.3.4 port 2222", got.Targets[1].SSH)
	}
	if got.Targets[1].Wire == nil || got.Targets[1].Wire.ChainID != 369 || got.Targets[1].Wire.ExecID != "reth" {
		t.Errorf("Targets[1].Wire = %+v, want chain 369 exec reth", got.Targets[1].Wire)
	}
	if got.AIProvider != "gemini" || got.AIKey != "secret-key" {
		t.Errorf("AIProvider/AIKey = %q/%q, want gemini/secret-key", got.AIProvider, got.AIKey)
	}
	if got.RefRPCBase != "https://rpc.valve.city/v1/vk_custom" {
		t.Errorf("RefRPCBase = %q, want the saved override", got.RefRPCBase)
	}
}

func TestSaveWritesMode0600(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := (Config{}).Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	info, err := os.Stat(filepath.Join(home, ".valve-node-app", "config.json"))
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("config.json mode = %o, want 0600", perm)
	}
}

func TestSaveIsAtomicNoLeftoverTempFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := (Config{AIProvider: "groq"}).Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	entries, err := os.ReadDir(filepath.Join(home, ".valve-node-app"))
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != "config.json" {
		names := make([]string, len(entries))
		for i, e := range entries {
			names[i] = e.Name()
		}
		t.Errorf("dir entries = %v, want exactly [config.json]", names)
	}
}

func TestDirIsHomeDotValveNode(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	d, err := Dir()
	if err != nil {
		t.Fatalf("Dir: %v", err)
	}
	if d != filepath.Join(home, ".valve-node-app") {
		t.Errorf("Dir() = %q, want %q", d, filepath.Join(home, ".valve-node-app"))
	}
}

// ---------------------------------------------------------------------
// migration: Target.gateway → Config.gateways
// ---------------------------------------------------------------------

// writeRawConfig drops a hand-written config.json in place, which is the only
// honest way to test a migration: the point is what happens to a file this
// version of the code can no longer produce.
func writeRawConfig(t *testing.T, home, body string) {
	t.Helper()
	dir := filepath.Join(home, ".valve-node-app")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, configFileName), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

// An operator who configured a gateway before gateways were a fleet-wide
// layer must not open the app to find it gone — while its container is still
// running and still serving. The upgrade is silent and lossless, and the id
// it lands on is the one that maps back to the container that already exists.
func TestLoad_MigratesAPerTargetGatewayToATopLevelOne(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	writeRawConfig(t, home, `{
	  "targets": [
	    {
	      "id": "local",
	      "mode": "local",
	      "devnet": {"HTTPPort": 8600, "WSPort": 8601},
	      "gateway": {
	        "Port": 4100,
	        "Networks": [
	          {"ChainID": 1337, "Upstreams": [{"ID": "devnet", "Endpoint": "http://127.0.0.1:8600", "Local": true}]}
	        ]
	      }
	    }
	  ],
	  "aiProvider": "",
	  "refRpcBase": ""
	}`)

	c, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(c.Gateways) != 1 {
		t.Fatalf("Gateways = %+v, want exactly the migrated one", c.Gateways)
	}
	gw := c.Gateways[0]
	if gw.ID != DefaultGatewayID {
		t.Errorf("migrated id = %q, want %q — that id is what keeps the EXISTING container", gw.ID, DefaultGatewayID)
	}
	if gw.Placement.TargetID != "local" || gw.Placement.Backend != "docker" {
		t.Errorf("placement = %+v, want it on the target that used to own it, docker-backed", gw.Placement)
	}
	// Lossless: the whole config comes across, not a re-derived approximation.
	if gw.Config.Port != 4100 || len(gw.Config.Networks) != 1 || gw.Config.Networks[0].ChainID != 1337 {
		t.Fatalf("migrated config = %+v, want the operator's config", gw.Config)
	}
	// The one upgrade the migration DOES make: a frozen URL that is literally
	// this machine's devnet becomes a reference to it. The two describe the
	// same upstream and only the reference survives the devnet's port moving,
	// so carrying the URL across would carry the staleness bug with it.
	up := gw.Config.Networks[0].Upstreams[0]
	if up.Kind != catalog.UpstreamManagedDevnet || up.TargetID != "local" || up.Endpoint != "" {
		t.Errorf("migrated upstream = %+v, want a managed-devnet reference to \"local\"", up)
	}
	if !up.Local {
		t.Error("the devnet was the preferred upstream and must stay preferred")
	}
	// The devnet stays where it is — it really does belong to the machine.
	if c.Targets[0].Devnet == nil || c.Targets[0].Devnet.HTTPPort != 8600 {
		t.Errorf("the devnet must be left alone: %+v", c.Targets[0].Devnet)
	}
	if c.Targets[0].LegacyGateway != nil {
		t.Error("the legacy field must be cleared, or the next Load would migrate it a second time")
	}

	// And it must not come back: saving writes the new shape, and the old key
	// is gone from the file.
	if err := c.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(home, ".valve-node-app", configFileName))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), `"gateway"`) {
		t.Errorf("the saved file still carries a per-target gateway:\n%s", raw)
	}
	again, err := Load()
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if len(again.Gateways) != 1 {
		t.Errorf("reload produced %d gateways — the migration is not idempotent", len(again.Gateways))
	}
}

// Two targets each carrying a gateway cannot both take the container name
// that only one container can have. The first keeps it; the second gets its
// own id (and therefore its own container) rather than silently colliding.
func TestLoad_MigratesTwoGatewaysWithoutColliding(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	writeRawConfig(t, home, `{
	  "targets": [
	    {"id": "local", "mode": "local", "gateway": {"Port": 4000, "Networks": [{"ChainID": 1, "Upstreams": [{"Endpoint": "https://a.example"}]}]}},
	    {"id": "box1",  "mode": "ssh",   "gateway": {"Port": 4000, "Networks": [{"ChainID": 369, "Upstreams": [{"Endpoint": "https://b.example"}]}]}}
	  ]
	}`)

	c, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(c.Gateways) != 2 {
		t.Fatalf("got %d gateways, want 2", len(c.Gateways))
	}
	if c.Gateways[0].ID == c.Gateways[1].ID {
		t.Fatalf("both migrated gateways took the id %q", c.Gateways[0].ID)
	}
	for _, gw := range c.Gateways {
		if gw.Placement.TargetID == "" {
			t.Errorf("gateway %q lost its placement", gw.ID)
		}
	}
	if _, ok := c.FindGateway(DefaultGatewayID); !ok {
		t.Error("the first migrated gateway must keep the default id, so its existing container is not orphaned")
	}
}

// An external upstream that merely looks local is left exactly as written.
// Guessing at what an endpoint "probably meant" is how a migration silently
// repoints somebody's traffic.
func TestLoad_MigrationDoesNotAdoptAnUnrelatedLocalURL(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	writeRawConfig(t, home, `{
	  "targets": [
	    {
	      "id": "local",
	      "mode": "local",
	      "devnet": {"HTTPPort": 8600, "WSPort": 8601},
	      "gateway": {
	        "Port": 4100,
	        "Networks": [
	          {"ChainID": 1337, "Upstreams": [{"ID": "something-else", "Endpoint": "http://127.0.0.1:9999"}]}
	        ]
	      }
	    }
	  ]
	}`)

	c, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	up := c.Gateways[0].Config.Networks[0].Upstreams[0]
	if up.Kind != "" || up.Endpoint != "http://127.0.0.1:9999" {
		t.Errorf("upstream = %+v, want it untouched", up)
	}
}
