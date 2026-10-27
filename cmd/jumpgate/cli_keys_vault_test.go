package main

import (
	"bytes"
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
