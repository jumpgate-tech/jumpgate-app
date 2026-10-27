package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/valve-tech/jumpgate/internal/secrets"
	"github.com/valve-tech/jumpgate/internal/server"
	"github.com/valve-tech/jumpgate/internal/signer"
	"github.com/valve-tech/jumpgate/internal/testutil"
	"github.com/valve-tech/jumpgate/internal/vault"
	"github.com/valve-tech/jumpgate/internal/vault/vaulttest"
)

type credsCall struct {
	stdin []byte
	args  []string
}

// fakeSystemdCreds records each systemd-creds run; fail answers an error for
// the subcommand it names.
func fakeSystemdCreds(t *testing.T, fail string) *[]credsCall {
	t.Helper()
	var calls []credsCall
	old := runSystemdCreds
	runSystemdCreds = func(_ context.Context, stdin []byte, args ...string) error {
		calls = append(calls, credsCall{bytes.Clone(stdin), args})
		if len(args) > 0 && args[0] == "encrypt" {
			// Like systemd-creds, it may leave a partial file on failure.
			if err := os.WriteFile(args[len(args)-1], []byte("cred"), 0o600); err != nil {
				return err
			}
		}
		if len(args) > 0 && args[0] == fail {
			return errors.New("systemd-creds: " + fail + " failed")
		}
		return nil
	}
	t.Cleanup(func() { runSystemdCreds = old })
	fakeCredFile(t, "")
	return &calls
}

// fakeCredFile runs the credential's mv and rm in this process and records
// them; fail answers an error for the command it names.
func fakeCredFile(t *testing.T, fail string) *[][]string {
	t.Helper()
	var calls [][]string
	old := runCredFile
	runCredFile = func(_ context.Context, argv ...string) error {
		calls = append(calls, argv)
		if argv[0] == fail {
			return errors.New(fail + " failed")
		}
		switch argv[0] {
		case "mv":
			return os.Rename(argv[3], argv[4])
		case "rm":
			if err := os.Remove(argv[3]); err != nil && !os.IsNotExist(err) {
				return err
			}
			return nil
		}
		return errors.New("unexpected " + argv[0])
	}
	t.Cleanup(func() { runCredFile = old })
	return &calls
}

// The credential is encrypted to a temporary name beside --out and moved
// into place only after the slot is added; a failed slot leaves the old
// credential and no temporary file.
func TestSealRenamesTheCredentialAfterTheSlot(t *testing.T) {
	testutil.Home(t)
	_, stderr := captureStdio(t)
	placeVault(t)
	calls := fakeSystemdCreds(t, "")
	files := fakeCredFile(t, "")
	useFakeTTY(t, vaulttest.Passphrase)
	dir := t.TempDir()
	out := filepath.Join(dir, "cred")
	if code := cmdVault([]string{"seal", "--allow-host-key", "--out", out}); code != 0 {
		t.Fatalf("exit %d: %s", code, stderr())
	}
	tmp := (*calls)[0].args[len((*calls)[0].args)-1]
	if filepath.Dir(tmp) != dir || tmp == out {
		t.Fatalf("encrypt wrote %s, want a temporary name in %s", tmp, dir)
	}
	if len(*files) != 1 || !slices.Equal((*files)[0], []string{"mv", "-f", "--", tmp, out}) {
		t.Fatalf("file ops %q", *files)
	}
	if b, err := os.ReadFile(out); err != nil || string(b) != "cred" {
		t.Fatalf("credential %q %v", b, err)
	}
	for _, op := range *files {
		for _, a := range op {
			if strings.Contains(a, string((*calls)[0].stdin)) {
				t.Fatal("the KEK is on argv")
			}
		}
	}
}

func TestSealFailedMoveRemovesTheTemp(t *testing.T) {
	testutil.Home(t)
	captureStdio(t)
	placeVault(t)
	fakeSystemdCreds(t, "")
	files := fakeCredFile(t, "mv")
	useFakeTTY(t, vaulttest.Passphrase)
	dir := t.TempDir()
	out := filepath.Join(dir, "cred")
	if err := os.WriteFile(out, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	if code := cmdVault([]string{"seal", "--allow-host-key", "--out", out}); code != 1 {
		t.Fatalf("exit %d", code)
	}
	if len(*files) != 2 || (*files)[1][0] != "rm" {
		t.Fatalf("file ops %q", *files)
	}
	if b, _ := os.ReadFile(out); string(b) != "old" {
		t.Fatalf("credential replaced: %q", b)
	}
	if m, _ := filepath.Glob(filepath.Join(dir, ".cred.tmp-*")); len(m) != 0 {
		t.Fatalf("temp left: %v", m)
	}
}

func slotKinds(t *testing.T, p string) []vault.SlotKind {
	t.Helper()
	v, err := vault.Open(p)
	if err != nil {
		t.Fatal(err)
	}
	var out []vault.SlotKind
	for _, s := range v.Slots() {
		out = append(out, s.Kind)
	}
	return out
}

func TestSealPipesTheKEKOnStdin(t *testing.T) {
	testutil.Home(t)
	_, stderr := captureStdio(t)
	p := placeVault(t)
	calls := fakeSystemdCreds(t, "")
	useFakeTTY(t, vaulttest.Passphrase)
	out := filepath.Join(t.TempDir(), "cred")
	if code := cmdVault([]string{"seal", "--allow-host-key", "--out", out}); code != 0 {
		t.Fatalf("exit %d: %s", code, stderr())
	}
	if !strings.Contains(stderr(), "only protects against copying the vault file alone") {
		t.Fatalf("no host-key warning: %q", stderr())
	}
	if len(*calls) != 1 {
		t.Fatalf("calls %+v", *calls)
	}
	c := (*calls)[0]
	if c.args[0] != "encrypt" || !slices.Contains(c.args, "--with-key=host") || !slices.Contains(c.args, "--name=jumpgate-vault-key") ||
		!slices.Contains(c.args, "-") || filepath.Dir(c.args[len(c.args)-1]) != filepath.Dir(out) {
		t.Fatalf("args %q", c.args)
	}
	if len(c.stdin) != 32 {
		t.Fatalf("stdin %d bytes, want 32", len(c.stdin))
	}
	argv := strings.Join(c.args, " ")
	for _, enc := range []string{hex.EncodeToString(c.stdin), base64.StdEncoding.EncodeToString(c.stdin), base64.RawURLEncoding.EncodeToString(c.stdin), string(c.stdin)} {
		if strings.Contains(argv, enc) {
			t.Fatal("the KEK is on argv")
		}
	}
	v, err := vault.Open(p)
	if err != nil {
		t.Fatal(err)
	}
	var cred bool
	for _, s := range v.Slots() {
		cred = cred || (s.Kind == vault.SlotSystemdCreds && s.Cred == "jumpgate-vault-key")
	}
	if !cred {
		t.Fatalf("slots %+v", v.Slots())
	}
	if err := v.UnlockKEK(vault.SlotSystemdCreds, bytes.Clone(c.stdin)); err != nil {
		t.Fatal(err)
	}
	v.Lock()
}

// Only root reads the host credential secret, so a vault owner who is not
// root encrypts through sudo: the KEK still goes on stdin, the vault is
// written by its owner, and root writes only the credential.
func TestSystemdCredsEncryptRunsAsRoot(t *testing.T) {
	enc := []string{"encrypt", "--with-key=host", "--name=jumpgate-vault-key", "-", "/etc/credstore.encrypted/jumpgate-vault-key"}
	if name, args := systemdCredsCommand(1000, enc); name != "sudo" || !slices.Equal(args, append([]string{"--", "systemd-creds"}, enc...)) {
		t.Fatalf("non-root: %s %q", name, args)
	}
	if name, args := systemdCredsCommand(0, enc); name != "systemd-creds" || !slices.Equal(args, enc) {
		t.Fatalf("root: %s %q", name, args)
	}
	if name, _ := systemdCredsCommand(1000, []string{"has-tpm2"}); name != "systemd-creds" {
		t.Fatalf("has-tpm2 needs no root: %s", name)
	}
}

// With no terminal the passphrase error says to pipe it in with --stdin,
// so seal takes --stdin, as init and unlock do.
func TestSealReadsThePassphraseFromStdin(t *testing.T) {
	testutil.Home(t)
	_, stderr := captureStdio(t)
	p := placeVault(t)
	calls := fakeSystemdCreds(t, "")
	_, opens := useFakeTTY(t)
	withStdin(t, vaulttest.Passphrase+"\n")
	if code := cmdVault([]string{"seal", "--allow-host-key", "--stdin", "--out", filepath.Join(t.TempDir(), "c")}); code != 0 {
		t.Fatalf("exit %d: %s", code, stderr())
	}
	if len(*calls) != 1 || *opens != 0 {
		t.Fatalf("calls %+v, prompts %d", *calls, *opens)
	}
	if k := slotKinds(t, p); len(k) != 2 {
		t.Fatalf("slots %v", k)
	}
}

func TestSealTPM2WithoutATPMRefuses(t *testing.T) {
	testutil.Home(t)
	_, stderr := captureStdio(t)
	p := placeVault(t)
	calls := fakeSystemdCreds(t, "has-tpm2")
	_, opens := useFakeTTY(t, vaulttest.Passphrase)
	if code := cmdVault([]string{"seal", "--tpm2"}); code != 1 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(stderr(), "no usable TPM2 here; --allow-host-key seals with the host key instead, which only protects against copying the vault file alone") {
		t.Fatalf("stderr %q", stderr())
	}
	if len(*calls) != 1 || *opens != 0 {
		t.Fatalf("calls %+v, prompts %d", *calls, *opens)
	}
	if k := slotKinds(t, p); len(k) != 1 {
		t.Fatalf("slots %v", k)
	}
}

func TestSealTPM2UsesTheTPM(t *testing.T) {
	testutil.Home(t)
	_, stderr := captureStdio(t)
	placeVault(t)
	calls := fakeSystemdCreds(t, "")
	useFakeTTY(t, vaulttest.Passphrase)
	if code := cmdVault([]string{"seal", "--tpm2", "--out", filepath.Join(t.TempDir(), "c")}); code != 0 {
		t.Fatalf("exit %d: %s", code, stderr())
	}
	if len(*calls) != 2 || (*calls)[0].args[0] != "has-tpm2" || !slices.Contains((*calls)[1].args, "--with-key=tpm2") {
		t.Fatalf("calls %+v", *calls)
	}
}

func TestSealNeedsAMode(t *testing.T) {
	testutil.Home(t)
	captureStdio(t)
	placeVault(t)
	fakeSystemdCreds(t, "")
	for _, args := range [][]string{{"seal"}, {"seal", "--tpm2", "--allow-host-key"}} {
		if code := cmdVault(args); code != 2 {
			t.Errorf("%v: exit %d, want 2", args, code)
		}
	}
}

func TestSealFailureAddsNoSlot(t *testing.T) {
	testutil.Home(t)
	captureStdio(t)
	p := placeVault(t)
	fakeSystemdCreds(t, "encrypt")
	useFakeTTY(t, vaulttest.Passphrase)
	if code := cmdVault([]string{"seal", "--allow-host-key", "--out", filepath.Join(t.TempDir(), "c")}); code != 1 {
		t.Fatalf("exit %d", code)
	}
	if k := slotKinds(t, p); len(k) != 1 {
		t.Fatalf("slots %v", k)
	}
}

func TestVaultInitAddsAKeystoreSlotOnMacOS(t *testing.T) {
	testutil.Home(t)
	_, stderr := captureStdio(t)
	signer.FakeKeychainForTest(t)
	old := vaultKeystoreStore
	vaultKeystoreStore = func() signer.Store { return signer.StoreKeychain }
	t.Cleanup(func() { vaultKeystoreStore = old })
	useFakeTTY(t, vaulttest.Passphrase, vaulttest.Passphrase)
	if code := cmdVault([]string{"init"}); code != 0 {
		t.Fatalf("exit %d: %s", code, stderr())
	}
	p, _ := secrets.VaultPath()
	v, err := vault.Open(p)
	if err != nil {
		t.Fatal(err)
	}
	var primary, ks bool
	for _, s := range v.Slots() {
		primary = primary || (s.Kind == vault.SlotPassphrase && s.Label == "primary")
		ks = ks || (s.Kind == vault.SlotKeystore && s.Store == "keychain" && s.Ref == secrets.KeystoreRef)
	}
	if !primary || !ks {
		t.Fatalf("slots %+v", v.Slots())
	}

	// The server's auto-unlock opens it with no prompt and no credential.
	s := server.New(server.Config{Bind: freeAddr(t), Token: server.NewSessionToken(), Shutdown: func() {}})
	var logs []string
	autoUnlock(s, v, func(string) string { return "" }, func(f string, a ...any) { logs = append(logs, f) })
	if st := s.VaultState().State; st != "unlocked" {
		t.Fatalf("state %q, logs %q", st, logs)
	}
	s.Lock()
}

func TestVaultInitKeystoreNoneAddsNoSlot(t *testing.T) {
	testutil.Home(t)
	captureStdio(t)
	old := vaultKeystoreStore
	vaultKeystoreStore = func() signer.Store { t.Fatal("asked for the default store"); return "" }
	t.Cleanup(func() { vaultKeystoreStore = old })
	useFakeTTY(t, vaulttest.Passphrase, vaulttest.Passphrase)
	if code := cmdVault([]string{"init", "--keystore", "none"}); code != 0 {
		t.Fatalf("exit %d", code)
	}
	p, _ := secrets.VaultPath()
	if k := slotKinds(t, p); len(k) != 1 {
		t.Fatalf("slots %v", k)
	}
}
