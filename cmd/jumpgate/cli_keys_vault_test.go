package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/secrets"
	"github.com/valve-tech/jumpgate/internal/signer"
	"github.com/valve-tech/jumpgate/internal/testutil"
	"github.com/valve-tech/jumpgate/internal/vault"
	"github.com/valve-tech/jumpgate/internal/vault/vaulttest"
)

func TestKeysInitVaultCreatesTheVaultAndRecordsTheKey(t *testing.T) {
	testutil.Home(t)
	stdout, stderr := captureStdio(t)
	written, _ := useFakeTTY(t, vaulttest.Passphrase, vaulttest.Passphrase)
	if code := keysInit([]string{"--store", "vault"}); code != 0 {
		t.Fatalf("exit %d: %s", code, stderr())
	}
	if !strings.Contains(written.String()+stderr(), "a lost passphrase cannot be recovered: jumpgate keeps no copy") {
		t.Fatal("no warning about a lost passphrase")
	}
	if !strings.Contains(stdout(), "run `jumpgate unlock` so the server uses it") {
		t.Fatalf("stdout %q", stdout())
	}
	c, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.Controller == nil || c.Controller.KeyStore != "vault" || c.Controller.KeyRef != "controller-key" || c.SecretStore != "vault" {
		t.Fatalf("config %+v secretStore %q", c.Controller, c.SecretStore)
	}
	p, err := secrets.VaultPath()
	if err != nil {
		t.Fatal(err)
	}
	v, err := vault.Open(p)
	if err != nil {
		t.Fatal(err)
	}
	if err := v.Unlock([]byte(vaulttest.Passphrase)); err != nil {
		t.Fatal(err)
	}
	defer v.Lock()
	k, err := signer.OpenVault(v, "controller-key")
	if err != nil {
		t.Fatal(err)
	}
	if k.Address().Hex() != c.Controller.Address {
		t.Fatalf("vault key %s, config %s", k.Address().Hex(), c.Controller.Address)
	}
}

func TestKeysShowOpensAVaultKey(t *testing.T) {
	testutil.Home(t)
	captureStdio(t)
	useFakeTTY(t, vaulttest.Passphrase, vaulttest.Passphrase)
	if code := keysInit([]string{"--store", "vault"}); code != 0 {
		t.Fatalf("keys init exit %d", code)
	}
	c, _ := config.Load()
	_, opens := useFakeTTY(t, vaulttest.Passphrase)
	var out, errb bytes.Buffer
	if code := keysShow(nil, &out, &errb); code != 0 {
		t.Fatalf("exit %d: %s", code, errb.String())
	}
	if *opens != 1 {
		t.Fatalf("prompted %d times, want 1", *opens)
	}
	if !strings.Contains(out.String(), c.Controller.Address) {
		t.Fatalf("out %q, want %s", out.String(), c.Controller.Address)
	}
}

// failCoreDumps makes disabling core dumps fail and counts the attempts.
func failCoreDumps(t *testing.T) *int {
	t.Helper()
	n := 0
	old := disableCoreDumps
	disableCoreDumps = func() error { n++; return errors.New("prctl refused") }
	t.Cleanup(func() { disableCoreDumps = old })
	return &n
}

// An offline unlock in the CLI process refuses when core dumps cannot be
// turned off, before any key is derived.
func TestKeysShowRefusesWhenCoreDumpsStayOn(t *testing.T) {
	testutil.Home(t)
	captureStdio(t)
	useFakeTTY(t, vaulttest.Passphrase, vaulttest.Passphrase)
	if code := keysInit([]string{"--store", "vault"}); code != 0 {
		t.Fatalf("keys init exit %d", code)
	}
	n := failCoreDumps(t)
	useFakeTTY(t, vaulttest.Passphrase)
	var out, errb bytes.Buffer
	if code := keysShow(nil, &out, &errb); code == 0 {
		t.Fatalf("exit 0 with core dumps on: %s", out.String())
	}
	if *n == 0 || !strings.Contains(errb.String(), "cannot disable core dumps") {
		t.Fatalf("attempts %d, stderr %q", *n, errb.String())
	}
}

func TestKeysInitVaultRefusesWhenCoreDumpsStayOn(t *testing.T) {
	testutil.Home(t)
	_, stderr := captureStdio(t)
	failCoreDumps(t)
	useFakeTTY(t, vaulttest.Passphrase, vaulttest.Passphrase)
	if code := keysInit([]string{"--store", "vault"}); code == 0 {
		t.Fatal("exit 0 with core dumps on")
	}
	if !strings.Contains(stderr(), "cannot disable core dumps") {
		t.Fatalf("stderr %q", stderr())
	}
	p, _ := secrets.VaultPath()
	if _, err := os.Lstat(p); err == nil {
		t.Fatal("a vault was created with core dumps on")
	}
}

func TestKeysExportRefusesOtherStores(t *testing.T) {
	testutil.Home(t)
	_, stderr := captureStdio(t)
	if code := keysInit([]string{"--store", "file", "--insecure-plain"}); code != 0 {
		t.Fatalf("keys init exit %d", code)
	}
	_, opens := useFakeTTY(t)
	if code := keysExport([]string{"--keystore-v3", filepath.Join(t.TempDir(), "k.json")}); code != 2 {
		t.Fatalf("exit %d", code)
	}
	if *opens != 0 || !strings.Contains(stderr(), "only a vault key is exported; the other stores already hold it where a wallet or tool can reach it") {
		t.Fatalf("prompts %d, stderr %q", *opens, stderr())
	}
}

func TestKeysExportWritesAPrivateFile(t *testing.T) {
	testutil.Home(t)
	_, stderr := captureStdio(t)
	useFakeTTY(t, vaulttest.Passphrase, vaulttest.Passphrase)
	if code := keysInit([]string{"--store", "vault"}); code != 0 {
		t.Fatalf("keys init exit %d", code)
	}
	old := exportScryptN
	exportScryptN = 1 << 12
	t.Cleanup(func() { exportScryptN = old })
	out := filepath.Join(t.TempDir(), "k.json")
	useFakeTTY(t, vaulttest.Passphrase, "wallet pw", "wallet pw")
	if code := keysExport([]string{"--keystore-v3", out}); code != 0 {
		t.Fatalf("exit %d: %s", code, stderr())
	}
	testutil.AssertPrivate(t, out)
	b, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	k, err := signer.DecryptKeystoreV3(b, []byte("wallet pw"))
	if err != nil {
		t.Fatal(err)
	}
	c, _ := config.Load()
	if k.Address().Hex() != c.Controller.Address {
		t.Fatalf("exported %s, controller %s", k.Address().Hex(), c.Controller.Address)
	}
	useFakeTTY(t, vaulttest.Passphrase, "wallet pw", "wallet pw")
	if code := keysExport([]string{"--keystore-v3", out}); code != 1 {
		t.Fatalf("existing FILE: exit %d", code)
	}
}

func TestKeysInitFileNeedsInsecurePlain(t *testing.T) {
	testutil.Home(t)
	_, stderr := captureStdio(t)
	if code := keysInit([]string{"--store", "file"}); code != 2 {
		t.Fatalf("exit %d, want 2", code)
	}
	if !strings.Contains(stderr(), "--store file keeps the key unencrypted on disk; add --insecure-plain to confirm, or use --store vault") {
		t.Fatalf("stderr %q", stderr())
	}
	if c, _ := config.Load(); c.Controller != nil {
		t.Fatal("a key was recorded without --insecure-plain")
	}
	if code := keysInit([]string{"--store", "file", "--insecure-plain"}); code != 0 {
		t.Fatalf("with --insecure-plain: exit %d: %s", code, stderr())
	}
	c, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.Controller == nil || c.SecretStore != "file" {
		t.Fatalf("controller %+v secretStore %q", c.Controller, c.SecretStore)
	}
}
