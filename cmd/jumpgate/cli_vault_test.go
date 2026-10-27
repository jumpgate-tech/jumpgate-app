package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/daemon"
	"github.com/valve-tech/jumpgate/internal/fsperm"
	"github.com/valve-tech/jumpgate/internal/secrets"
	"github.com/valve-tech/jumpgate/internal/testutil"
	"github.com/valve-tech/jumpgate/internal/vault"
	"github.com/valve-tech/jumpgate/internal/vault/vaulttest"
)

// placeVault puts a locked fixture vault at secrets.VaultPath().
func placeVault(t *testing.T) string {
	t.Helper()
	src, _ := vaulttest.Path(t)
	raw, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	p, err := secrets.VaultPath()
	if err != nil {
		t.Fatal(err)
	}
	if err := fsperm.MkdirPrivate(filepath.Dir(p)); err != nil {
		t.Fatal(err)
	}
	if err := fsperm.WriteFilePrivate(p, raw); err != nil {
		t.Fatal(err)
	}
	return p
}

func openAt(t *testing.T, p string, pass []byte) *vault.Vault {
	t.Helper()
	v, err := vault.Open(p)
	if err != nil {
		t.Fatal(err)
	}
	if err := v.Unlock(pass); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(v.Lock)
	return v
}

func noServer(t *testing.T, running bool) {
	t.Helper()
	old := findServer
	findServer = func(context.Context) (daemon.Info, bool, error) { return daemon.Info{}, running, nil }
	t.Cleanup(func() { findServer = old })
}

func TestVaultInitAsksTwiceAndWarns(t *testing.T) {
	testutil.Home(t)
	stdout, stderr := captureStdio(t)
	_, opens := useFakeTTY(t, vaulttest.Passphrase, vaulttest.Passphrase)
	if code := cmdVault([]string{"init"}); code != 0 {
		t.Fatalf("exit %d: %s", code, stderr())
	}
	if *opens != 1 {
		t.Fatalf("terminal opened %d times", *opens)
	}
	if !strings.Contains(stdout(), "a lost passphrase cannot be recovered: jumpgate keeps no copy") {
		t.Fatalf("stdout %q", stdout())
	}
	p, _ := secrets.VaultPath()
	openAt(t, p, []byte(vaulttest.Passphrase))

	useFakeTTY(t, vaulttest.Passphrase, vaulttest.Passphrase)
	if code := cmdVault([]string{"init"}); code != 1 {
		t.Fatalf("second init: exit %d", code)
	}
	if !strings.Contains(stderr(), "a vault already exists at "+p) {
		t.Fatalf("stderr %q", stderr())
	}
}

func TestVaultMigrateMovesSecrets(t *testing.T) {
	testutil.Home(t)
	if _, err := config.Update(func(c *config.Config) error { c.AIKey = "sk-plain-value"; return nil }); err != nil {
		t.Fatal(err)
	}
	p, _ := secrets.VaultPath()
	// A stand-in for the server: its unlock runs secrets.Migrate.
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/vault/unlock" {
			http.NotFound(w, r)
			return
		}
		var req api.UnlockRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		v, err := vault.Open(p)
		if err == nil {
			err = v.Unlock(req.Passphrase)
		}
		if err == nil {
			_, err = secrets.Migrate(v)
			v.Lock()
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(api.VaultView{State: "unlocked", Slots: []string{"passphrase"}})
	})
	withStdin(t, vaulttest.Passphrase+"\n")
	stdout, stderr := captureStdio(t)
	if code := cmdVault([]string{"migrate", "--stdin"}); code != 0 {
		t.Fatalf("exit %d: %s", code, stderr())
	}
	c, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if !config.IsSecretRef(c.AIKey) {
		t.Fatalf("aiKey %q is still plain", c.AIKey)
	}
	v := openAt(t, p, []byte(vaulttest.Passphrase))
	got, err := v.Get(strings.TrimPrefix(c.AIKey, "vault:"))
	if err != nil || string(got) != "sk-plain-value" {
		t.Fatalf("vault holds %q, %v", got, err)
	}
	if !strings.Contains(stdout(), "aiKey") || !strings.Contains(stdout(), "rotate it") {
		t.Fatalf("stdout %q", stdout())
	}
	if strings.Contains(stdout()+stderr(), "sk-plain-value") {
		t.Fatal("a secret value was printed")
	}
}

func TestVaultBackupIsAByteCopy(t *testing.T) {
	testutil.Home(t)
	_, stderr := captureStdio(t)
	p := placeVault(t)
	out := filepath.Join(t.TempDir(), "vault.bak")
	if code := cmdVault([]string{"backup", out}); code != 0 {
		t.Fatalf("exit %d: %s", code, stderr())
	}
	a, _ := os.ReadFile(p)
	b, _ := os.ReadFile(out)
	if !bytes.Equal(a, b) {
		t.Fatal("the backup differs from the vault")
	}
	testutil.AssertPrivate(t, out)
	if code := cmdVault([]string{"backup", out}); code != 1 {
		t.Fatalf("existing FILE: exit %d", code)
	}
}

func TestVaultRestoreRefusesARunningServer(t *testing.T) {
	testutil.Home(t)
	_, stderr := captureStdio(t)
	noServer(t, true)
	src, _ := vaulttest.Path(t)
	if code := cmdVault([]string{"restore", src}); code != 1 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(stderr(), "run `jumpgate stop` first") {
		t.Fatalf("stderr %q", stderr())
	}
	p, _ := secrets.VaultPath()
	if _, err := os.Lstat(p); err == nil {
		t.Fatal("restored under a running server")
	}
}

func TestVaultRestoreChecksTheFileFirst(t *testing.T) {
	testutil.Home(t)
	_, stderr := captureStdio(t)
	noServer(t, false)
	p := placeVault(t)
	before, _ := os.ReadFile(p)
	bad := filepath.Join(t.TempDir(), "bad")
	if err := fsperm.WriteFilePrivate(bad, []byte("JGVAULT1\n{not json\nxx")); err != nil {
		t.Fatal(err)
	}
	if code := cmdVault([]string{"restore", "--replace", bad}); code != 4 {
		t.Fatalf("exit %d: %s", code, stderr())
	}
	after, _ := os.ReadFile(p)
	if !bytes.Equal(before, after) {
		t.Fatal("the current vault changed")
	}
	good, _ := vaulttest.Path(t)
	if code := cmdVault([]string{"restore", good}); code != 1 {
		t.Fatalf("existing vault without --replace: exit %d", code)
	}
	useFakeTTY(t, vaulttest.Passphrase)
	if code := cmdVault([]string{"restore", "--replace", good}); code != 0 {
		t.Fatalf("restore: exit %d: %s", code, stderr())
	}
	g, _ := os.ReadFile(good)
	now, _ := os.ReadFile(p)
	if !bytes.Equal(g, now) {
		t.Fatal("the restored vault is not FILE")
	}
}

// A replacement that does not unlock is refused and changes nothing.
func TestVaultRestoreReplaceRefusesAWrongPassphrase(t *testing.T) {
	testutil.Home(t)
	_, stderr := captureStdio(t)
	noServer(t, false)
	p := placeVault(t)
	before, _ := os.ReadFile(p)
	good, _ := vaulttest.Path(t)
	withStdin(t, "not the passphrase at all\n")
	if code := cmdVault([]string{"restore", "--replace", "--stdin", good}); code == 0 {
		t.Fatal("restored with a wrong passphrase")
	}
	after, _ := os.ReadFile(p)
	if !bytes.Equal(before, after) {
		t.Fatal("the current vault changed")
	}
	if m, _ := filepath.Glob(p + ".bak-*"); len(m) != 0 {
		t.Fatalf("backup written: %v (%s)", m, stderr())
	}
}

// Before replacing, the old vault is kept as a private byte copy, and an
// existing backup name is refused.
func TestVaultRestoreReplaceKeepsTheOldVault(t *testing.T) {
	testutil.Home(t)
	_, stderr := captureStdio(t)
	noServer(t, false)
	p := placeVault(t)
	openAt(t, p, []byte(vaulttest.Passphrase)).Lock()
	old, _ := os.ReadFile(p)
	// A different vault file with the same passphrase.
	v := openAt(t, p, []byte(vaulttest.Passphrase))
	if err := v.Put("x", []byte("y"), "test"); err != nil {
		t.Fatal(err)
	}
	v.Lock()
	cand := filepath.Join(t.TempDir(), "cand")
	nb, _ := os.ReadFile(p)
	if err := fsperm.WriteFilePrivate(cand, nb); err != nil {
		t.Fatal(err)
	}
	if err := fsperm.WriteFilePrivate(p, old); err != nil {
		t.Fatal(err)
	}
	when := time.Date(2026, 10, 27, 12, 0, 0, 0, time.UTC)
	oldNow := restoreNow
	restoreNow = func() time.Time { return when }
	t.Cleanup(func() { restoreNow = oldNow })
	withStdin(t, vaulttest.Passphrase+"\n")
	if code := cmdVault([]string{"restore", "--replace", "--stdin", cand}); code != 0 {
		t.Fatalf("exit %d: %s", code, stderr())
	}
	bak := p + ".bak-20261027T120000Z"
	b, err := os.ReadFile(bak)
	if err != nil || !bytes.Equal(b, old) {
		t.Fatalf("backup %v, equal %v", err, bytes.Equal(b, old))
	}
	testutil.AssertPrivate(t, bak)
	if now, _ := os.ReadFile(p); !bytes.Equal(now, nb) {
		t.Fatal("the restored vault is not FILE")
	}
	withStdin(t, vaulttest.Passphrase+"\n")
	if code := cmdVault([]string{"restore", "--replace", "--stdin", cand}); code == 0 {
		t.Fatal("overwrote an existing backup")
	}
}

func TestVaultAddRecoveryPrintsOnce(t *testing.T) {
	testutil.Home(t)
	stdout, stderr := captureStdio(t)
	p := placeVault(t)
	useFakeTTY(t, vaulttest.Passphrase)
	if code := cmdVault([]string{"add-recovery"}); code != 0 {
		t.Fatalf("exit %d: %s", code, stderr())
	}
	var code string
	for _, l := range strings.Split(stdout(), "\n") {
		if len(strings.Fields(l)) == 24 {
			if code != "" {
				t.Fatal("the code was printed twice")
			}
			code = strings.TrimSpace(l)
		}
	}
	if code == "" {
		t.Fatalf("no 24-word line in %q", stdout())
	}
	if !strings.Contains(stdout(), "offline") {
		t.Fatalf("stdout %q", stdout())
	}
	openAt(t, p, []byte(code))
}

func TestVaultRekdfRefusesBelowTheFloor(t *testing.T) {
	testutil.Home(t)
	captureStdio(t)
	placeVault(t)
	_, opens := useFakeTTY(t, vaulttest.Passphrase)
	if code := cmdVault([]string{"rekdf", "--memory", "1024"}); code != 2 {
		t.Fatalf("exit %d, want 2", code)
	}
	if *opens != 0 {
		t.Fatal("prompted before checking the flags")
	}
}

func TestVaultRekdfKeepsTheEntries(t *testing.T) {
	testutil.Home(t)
	_, stderr := captureStdio(t)
	p := placeVault(t)
	v := openAt(t, p, []byte(vaulttest.Passphrase))
	if err := v.Put("ai-key", []byte("x"), "test"); err != nil {
		t.Fatal(err)
	}
	v.Lock()
	useFakeTTY(t, vaulttest.Passphrase)
	if code := cmdVault([]string{"rekdf", "--time", "4"}); code != 0 {
		t.Fatalf("exit %d: %s", code, stderr())
	}
	v2 := openAt(t, p, []byte(vaulttest.Passphrase))
	if got, err := v2.Get("ai-key"); err != nil || string(got) != "x" {
		t.Fatalf("entry %q, %v", got, err)
	}
}

func TestVaultOfflineUnlockRefusesWithCoreDumpsOn(t *testing.T) {
	testutil.Home(t)
	_, stderr := captureStdio(t)
	placeVault(t)
	failCoreDumps(t)
	useFakeTTY(t, vaulttest.Passphrase)
	if code := cmdVault([]string{"add-recovery"}); code == 0 {
		t.Fatal("exit 0 with core dumps on")
	}
	if !strings.Contains(stderr(), "cannot disable core dumps") {
		t.Fatalf("stderr %q", stderr())
	}
}
