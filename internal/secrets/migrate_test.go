package secrets

import (
	"bytes"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/fsperm"
	"github.com/valve-tech/jumpgate/internal/signer"
	"github.com/valve-tech/jumpgate/internal/testutil"
	"github.com/valve-tech/jumpgate/internal/vault"
	"github.com/valve-tech/jumpgate/internal/vault/vaulttest"
)

const vpnConf = "[Interface]\nPrivateKey = cHJpdmF0ZQ==\n"

// plainFixture is a home with every kind of plain secret.
type plainFixture struct {
	dir, ctlPath, sshPath string
	ctlHex                string
	sshPEM                []byte
}

func newPlainFixture(t *testing.T, store string) plainFixture {
	t.Helper()
	testutil.Home(t)
	dir, err := config.Dir()
	if err != nil {
		t.Fatal(err)
	}
	f := plainFixture{dir: dir, ctlPath: filepath.Join(dir, "keys", "controller.key"), sshPath: filepath.Join(dir, "ssh", "jumpgate_ed25519")}
	ctl := &config.Controller{KeyStore: store, KeyRef: "jumpgate-controller", Address: "0x1"}
	if store == string(signer.StoreFile) {
		k, err := signer.GenerateKeyFile(f.ctlPath)
		if err != nil {
			t.Fatal(err)
		}
		f.ctlHex = hex.EncodeToString(k.Bytes())
		ctl.KeyRef, ctl.Address = f.ctlPath, k.Address().Hex()
	}
	if err := fsperm.MkdirPrivate(filepath.Dir(f.sshPath)); err != nil {
		t.Fatal(err)
	}
	f.sshPEM = []byte("-----BEGIN OPENSSH PRIVATE KEY-----\nplain\n-----END OPENSSH PRIVATE KEY-----\n")
	if err := fsperm.WriteFilePrivate(f.sshPath, f.sshPEM); err != nil {
		t.Fatal(err)
	}
	if _, err := config.Update(func(c *config.Config) error {
		c.Controller = ctl
		c.AIKey = "sk-ai"
		c.ProviderKeys = map[string]string{"INFURA_API_KEY": "inf-1"}
		c.VPNs = []config.VPN{{ID: "proton", Config: vpnConf}}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return f
}

func gone(t *testing.T, p string) {
	t.Helper()
	if _, err := os.Lstat(p); !os.IsNotExist(err) {
		t.Fatalf("%s still exists: %v", p, err)
	}
}

func entry(t *testing.T, v *vault.Vault, name string) vault.Entry {
	t.Helper()
	e, err := v.Entry(name)
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	return e
}

// assertMigrated checks the end state of a full migration.
func assertMigrated(t *testing.T, f plainFixture, v *vault.Vault) {
	t.Helper()
	gone(t, f.ctlPath)
	gone(t, f.sshPath)
	c, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.Controller == nil || c.Controller.KeyStore != "vault" || c.Controller.KeyRef != NameControllerKey {
		t.Fatalf("controller %+v", c.Controller)
	}
	if c.AIKey != "vault:ai-key" || c.ProviderKeys["INFURA_API_KEY"] != "vault:provider/INFURA_API_KEY" || c.VPNs[0].Config != "vault:vpn/proton" {
		t.Fatalf("refs %q %v %q", c.AIKey, c.ProviderKeys, c.VPNs[0].Config)
	}
	if c.SecretStore != config.SecretStoreVault {
		t.Fatalf("secretStore %q", c.SecretStore)
	}
	for name, want := range map[string][2]string{
		NameControllerKey:              {f.ctlHex, f.ctlPath},
		NameTransportKey:               {string(f.sshPEM), f.sshPath},
		NameAIKey:                      {"sk-ai", "aiKey"},
		ProviderName("INFURA_API_KEY"): {"inf-1", "providerKeys.INFURA_API_KEY"},
		VPNName("proton"):              {vpnConf, "vpns.proton.config"},
	} {
		e := entry(t, v, name)
		if string(e.Value) != want[0] || e.From != want[1] {
			t.Fatalf("%s: %q from %q", name, e.Value, e.From)
		}
	}
	n := 0
	for _, m := range c.Notices {
		if m == MigrateNotice {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("notices %q", c.Notices)
	}
	if _, err := signer.OpenVault(v, NameControllerKey); err != nil {
		t.Fatal(err)
	}
}

func TestMigrateMovesEveryPlainSecret(t *testing.T) {
	f := newPlainFixture(t, string(signer.StoreFile))
	v := vaulttest.Unlocked(t)
	rep, err := Migrate(v)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Moved) != 5 || len(rep.Removed) != 2 || len(rep.Conflicts) != 0 {
		t.Fatalf("report %+v", rep)
	}
	assertMigrated(t, f, v)
}

func TestMigrateResumesAfterACrashAtEachStep(t *testing.T) {
	for step := 1; step <= 4; step++ {
		t.Run(string(rune('0'+step)), func(t *testing.T) {
			f := newPlainFixture(t, string(signer.StoreFile))
			v := vaulttest.Unlocked(t)
			boom := errors.New("crash")
			migrateCrash = func(s int) error {
				if s == step {
					return boom
				}
				return nil
			}
			_, err := Migrate(v)
			migrateCrash = nil
			if !errors.Is(err, boom) {
				t.Fatalf("step %d: %v", step, err)
			}
			// While stopped, no reference names a missing entry.
			c, _ := config.Load()
			for _, ref := range []string{c.AIKey, c.ProviderKeys["INFURA_API_KEY"], c.VPNs[0].Config} {
				if name, ok := config.SecretRefName(ref); ok {
					entry(t, v, name)
				}
			}
			if _, err := Migrate(v); err != nil {
				t.Fatalf("resume after step %d: %v", step, err)
			}
			assertMigrated(t, f, v)
		})
	}
}

func TestMigrateKeepsAConflictingPlainCopy(t *testing.T) {
	newPlainFixture(t, "keychain")
	v := vaulttest.Unlocked(t)
	if err := v.Put(NameAIKey, []byte("sk-OLDVALUE"), "test"); err != nil {
		t.Fatal(err)
	}
	if _, err := config.Update(func(c *config.Config) error { c.AIKey = "sk-NEWVALUE"; return nil }); err != nil {
		t.Fatal(err)
	}
	rep, err := Migrate(v)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(rep.Conflicts, []string{"aiKey"}) {
		t.Fatalf("conflicts %v", rep.Conflicts)
	}
	c, _ := config.Load()
	if c.AIKey != "sk-NEWVALUE" || c.SecretStore == config.SecretStoreVault {
		t.Fatalf("aiKey %q secretStore %q", c.AIKey, c.SecretStore)
	}
	if got := entry(t, v, NameAIKey); string(got.Value) != "sk-OLDVALUE" {
		t.Fatalf("vault %q", got.Value)
	}
	var named bool
	for _, m := range c.Notices {
		if strings.Contains(m, "NEWVALUE") || strings.Contains(m, "OLDVALUE") {
			t.Fatalf("a notice carries a value: %q", m)
		}
		if strings.Contains(m, "aiKey") {
			named = true
		}
	}
	if !named {
		t.Fatalf("no notice names aiKey: %q", c.Notices)
	}
	// The rest still moved.
	if c.ProviderKeys["INFURA_API_KEY"] != "vault:provider/INFURA_API_KEY" {
		t.Fatalf("provider %q", c.ProviderKeys["INFURA_API_KEY"])
	}
}

func TestMigrateOverwritesBeforeUnlinking(t *testing.T) {
	f := newPlainFixture(t, string(signer.StoreFile))
	v := vaulttest.Unlocked(t)
	seen := map[string]bool{}
	wipeHook = func(p string) {
		raw, err := os.ReadFile(p)
		if err != nil {
			t.Errorf("%s: %v", p, err)
			return
		}
		want := len(f.sshPEM)
		if p == f.ctlPath {
			want = -1
		}
		if !allZero(raw) || (want >= 0 && len(raw) != want) {
			t.Errorf("%s not zeroed over its length: %d bytes", p, len(raw))
		}
		seen[p] = true
	}
	t.Cleanup(func() { wipeHook = nil })
	if _, err := Migrate(v); err != nil {
		t.Fatal(err)
	}
	if !seen[f.ctlPath] || !seen[f.sshPath] {
		t.Fatalf("hook saw %v", seen)
	}
}

func TestMigrateOverwriteRefusesALink(t *testing.T) {
	testutil.RequireUnix(t)
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	if err := os.WriteFile(target, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if _, err := wipeFile(link); err == nil {
		t.Fatal("a link was wiped")
	}
	if raw, _ := os.ReadFile(target); string(raw) != "keep" {
		t.Fatalf("target %q", raw)
	}
}

func TestMigrateIsANoOpWhenNothingIsPlain(t *testing.T) {
	testutil.Home(t)
	if _, err := config.Update(func(c *config.Config) error { c.AIKey = ""; return nil }); err != nil {
		t.Fatal(err)
	}
	dir, _ := config.Dir()
	before, err := os.ReadFile(filepath.Join(dir, "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	rep, err := Migrate(vaulttest.Unlocked(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Moved)+len(rep.Removed)+len(rep.Conflicts) != 0 {
		t.Fatalf("report %+v", rep)
	}
	after, _ := os.ReadFile(filepath.Join(dir, "config.json"))
	if !bytes.Equal(before, after) {
		t.Fatal("config.json changed")
	}
}

func TestMigrateLeavesOSStoreKeysAlone(t *testing.T) {
	f := newPlainFixture(t, "keychain")
	v := vaulttest.Unlocked(t)
	if _, err := Migrate(v); err != nil {
		t.Fatal(err)
	}
	c, _ := config.Load()
	if c.Controller.KeyStore != "keychain" || c.Controller.KeyRef != "jumpgate-controller" {
		t.Fatalf("controller %+v", c.Controller)
	}
	if _, err := v.Get(NameControllerKey); !errors.Is(err, vault.ErrNotFound) {
		t.Fatalf("controller-key: %v", err)
	}
	gone(t, f.sshPath)
	if c.AIKey != "vault:ai-key" || c.VPNs[0].Config != "vault:vpn/proton" || c.SecretStore != config.SecretStoreVault {
		t.Fatalf("config %q %q %q", c.AIKey, c.VPNs[0].Config, c.SecretStore)
	}
}

func TestMigrateRefusesALockedVault(t *testing.T) {
	testutil.Home(t)
	v := vaulttest.Unlocked(t)
	v.Lock()
	if _, err := Migrate(v); !errors.Is(err, vault.ErrLocked) {
		t.Fatal(err)
	}
	if _, err := Migrate(nil); !errors.Is(err, vault.ErrLocked) {
		t.Fatal(err)
	}
}
