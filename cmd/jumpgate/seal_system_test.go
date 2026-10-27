package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/apiclient"
	"github.com/valve-tech/jumpgate/internal/apiclient/apiclienttest"
	"github.com/valve-tech/jumpgate/internal/executor"
	"github.com/valve-tech/jumpgate/internal/sysunit"
	"github.com/valve-tech/jumpgate/internal/vault"
)

// sealFixture is a jumpgate service that serves kek from POST
// /api/system/seal, a systemd-creds that fails has-tpm2 when noTPM is set,
// and a recorder for the root commands.
type sealFixture struct {
	rec    *privRecorder
	creds  *[]credsCall
	posted *bool
	pass   *[]byte
	kek    []byte
}

func sealSystemSeams(t *testing.T, noTPM bool) *sealFixture {
	t.Helper()
	f := &sealFixture{rec: &privRecorder{}, kek: bytes.Repeat([]byte{9}, 32)}
	var mu sync.Mutex
	posted := false
	var pass []byte
	f.posted, f.pass = &posted, &pass
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		if r.Method == http.MethodPost && r.URL.Path == "/api/system/seal" {
			posted = true
			var req api.SystemSeal
			_ = json.NewDecoder(r.Body).Decode(&req)
			pass = req.Passphrase
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(api.SystemSealed{KEK: f.kek})
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(ts.Close)
	c := apiclienttest.NewHTTP(t, ts.URL, "tok")
	var calls []credsCall
	f.creds = &calls
	oldF, oldC, oldP, oldE, oldN := findServiceClient, runSystemdCreds, sealPrivileged, geteuid, sudoNonInteractive
	findServiceClient = func(context.Context) (*apiclient.Client, bool) { return c, true }
	runSystemdCreds = func(_ context.Context, stdin []byte, args ...string) error {
		calls = append(calls, credsCall{bytes.Clone(stdin), args})
		if noTPM && args[0] == "has-tpm2" {
			return errors.New("no TPM2")
		}
		return nil
	}
	sealPrivileged = func() executor.Executor { return f.rec }
	geteuid = func() int { return 1000 }
	sudoNonInteractive = func(context.Context) error { return nil }
	t.Cleanup(func() {
		findServiceClient, runSystemdCreds, sealPrivileged, geteuid, sudoNonInteractive = oldF, oldC, oldP, oldE, oldN
	})
	withStdin(t, "pw\n")
	return f
}

func (f *sealFixture) encrypt(t *testing.T) credsCall {
	t.Helper()
	for _, c := range *f.creds {
		if c.args[0] == "encrypt" {
			return c
		}
	}
	t.Fatalf("no encrypt: %+v", *f.creds)
	return credsCall{}
}

func TestSealRefusesWithoutATPM(t *testing.T) {
	f := sealSystemSeams(t, true)
	_, stderr := captureStdio(t)
	if code := cmdVault([]string{"seal", "--tpm2", "--stdin"}); code != exitCode("failed") {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(stderr(), "--allow-host-key") || *f.posted || len(f.rec.cmds) != 0 {
		t.Fatalf("stderr %q posted %v cmds %v", stderr(), *f.posted, f.rec.cmds)
	}
}

func TestSealWritesTheCredentialAndTheDropIn(t *testing.T) {
	f := sealSystemSeams(t, false)
	stdout, stderr := captureStdio(t)
	if code := cmdVault([]string{"seal", "--tpm2", "--stdin"}); code != 0 {
		t.Fatalf("exit %d: %s", code, stderr())
	}
	if string(*f.pass) != "pw" {
		t.Fatalf("seal passphrase %q", *f.pass)
	}
	enc := f.encrypt(t)
	if !slices.Contains(enc.args, "--with-key=tpm2") || !slices.Contains(enc.args, "--name=jumpgate-vault-key") || !bytes.Equal(enc.stdin, f.kek) {
		t.Fatalf("encrypt %q stdin %x", enc.args, enc.stdin)
	}
	staged := enc.args[len(enc.args)-1]
	if !strings.HasPrefix(staged, setupStagingDir+"/") {
		t.Fatalf("encrypted to %s, want the root-only staging dir", staged)
	}
	found := false
	for _, c := range f.rec.cmds {
		if strings.HasPrefix(c, "install ") && strings.Contains(c, "'0600'") && strings.HasSuffix(c, "'"+staged+"' '"+sysunit.SealCredPath+"'") {
			found = true
		}
	}
	if !found {
		t.Fatalf("credential not installed root 0600 at %s: %v", sysunit.SealCredPath, f.rec.cmds)
	}
	_, dropIn := installedAt(t, f.rec, sysunit.SealDropInPath)
	if string(dropIn) != "[Service]\nLoadCredentialEncrypted=jumpgate-vault-key:/etc/jumpgate-ctl/vault-key.cred\n" {
		t.Fatalf("drop-in %q", dropIn)
	}
	if last := f.rec.cmds[len(f.rec.cmds)-1]; last != "systemctl daemon-reload" {
		t.Fatalf("last command %q", last)
	}
	for _, c := range f.rec.cmds {
		if strings.Contains(c, string(f.kek)) {
			t.Fatal("the KEK is on a command line")
		}
	}
	if strings.Contains(stdout()+stderr(), string(f.kek)) {
		t.Fatal("the KEK was printed")
	}
}

func TestSealWithTheHostKeyNeedsTheFlag(t *testing.T) {
	f := sealSystemSeams(t, true)
	_, stderr := captureStdio(t)
	if code := cmdVault([]string{"seal", "--allow-host-key", "--stdin"}); code != 0 {
		t.Fatalf("exit %d: %s", code, stderr())
	}
	if enc := f.encrypt(t); !slices.Contains(enc.args, "--with-key=host") {
		t.Fatalf("encrypt %q", enc.args)
	}
}

func TestStartUnlocksFromTheCredentialsDirectory(t *testing.T) {
	kek := bytes.Repeat([]byte{7}, 32)
	vaultHome(t, func(v *vault.Vault) {
		if err := v.AddKEKSlot(vault.Slot{Kind: vault.SlotSystemdCreds, Cred: vaultCredName}, bytes.Clone(kek)); err != nil {
			t.Fatal(err)
		}
	})
	for _, c := range []struct {
		kek  []byte
		want string
	}{{kek, "unlocked"}, {bytes.Repeat([]byte{8}, 32), "locked"}} {
		credentialDir(t, c.kek)
		var log strings.Builder
		b, err := buildServer(serverOptions{Bind: freeAddr(t), ERPCURL: "http://127.0.0.1:4000", System: true, AdminGID: os.Getgid()}, func() {}, &log)
		if err != nil {
			t.Fatal(err)
		}
		runTask(t, b, "vault-auto-unlock")
		if got := b.srv.VaultState().State; got != c.want {
			t.Fatalf("vault state %q, want %q", got, c.want)
		}
		if c.want == "locked" && !strings.Contains(log.String(), "the passphrase still works") {
			t.Fatalf("log %q", log.String())
		}
		b.srv.Lock()
	}
}
