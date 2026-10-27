package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/daemon"
	"github.com/valve-tech/jumpgate/internal/fsperm"
	"github.com/valve-tech/jumpgate/internal/secrets"
	"github.com/valve-tech/jumpgate/internal/signer"
	"github.com/valve-tech/jumpgate/internal/testutil"
	"github.com/valve-tech/jumpgate/internal/vault"
	"github.com/valve-tech/jumpgate/internal/vault/vaulttest"
)

// Every start says so while a secret is stored unencrypted.
func TestServerStartWarnsAboutPlainSecrets(t *testing.T) {
	const warning = "secrets are stored unencrypted; run `jumpgate vault migrate`"
	testutil.Home(t)
	if _, err := config.Update(func(c *config.Config) error { c.AIKey = "sk"; return nil }); err != nil {
		t.Fatal(err)
	}
	var stderr strings.Builder
	if _, err := loadServerConfig(&stderr); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stderr.String(), warning) {
		t.Fatalf("stderr %q, want the warning", stderr.String())
	}
	if _, err := config.Update(func(c *config.Config) error {
		c.SecretStore, c.AIKey = config.SecretStoreVault, config.SecretRefPrefix+"ai-key"
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	stderr.Reset()
	if _, err := loadServerConfig(&stderr); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(stderr.String(), warning) {
		t.Fatalf("warned with only vault references: %q", stderr.String())
	}
	for _, c := range []config.Config{
		{SecretStore: config.SecretStoreFile},
		{Controller: &config.Controller{KeyStore: "file", KeyRef: "/k"}},
	} {
		if plainSecretsWarning(c) == "" {
			t.Errorf("no warning for %+v", c)
		}
	}
}

// serverFlagNames lists the flags that shape the server on a flag set, so the
// two entry points can be compared.
func serverFlagNames(fs *flag.FlagSet) []string {
	var names []string
	fs.VisitAll(func(f *flag.Flag) {
		switch f.Name {
		case "no-open", "tray", "version":
			return // entry-point specific: how to show the UI, not what the server is
		}
		names = append(names, f.Name)
	})
	sort.Strings(names)
	return names
}

func noEnv(string) string { return "" }

// C1/E1: the app and `jumpgate serve` accept the same server flags, so a
// server auto-started by the CLI can be given the same relay and key admin.
func TestServeAndAppAcceptTheSameServerFlags(t *testing.T) {
	app := flag.NewFlagSet("app", flag.ContinueOnError)
	if _, err := addServerFlags(app, noEnv); err != nil {
		t.Fatal(err)
	}
	serve := newServeFlagSet()
	if _, err := addServerFlags(serve, noEnv); err != nil {
		t.Fatal(err)
	}
	a, s := serverFlagNames(app), serverFlagNames(serve)
	if !reflect.DeepEqual(a, s) {
		t.Fatalf("server flags differ:\n app   %v\n serve %v", a, s)
	}
	for _, want := range []string{"bind", "relay-bind", "billing-socket", "erpc-url", "erpc-project", "meter"} {
		if !contains(a, want) {
			t.Errorf("missing flag --%s", want)
		}
	}
}

func contains(xs []string, x string) bool {
	for _, y := range xs {
		if y == x {
			return true
		}
	}
	return false
}

// `serve` parses the relay flags the app takes, instead of rejecting them.
func TestParseServeArgsTakesRelayFlags(t *testing.T) {
	opts, err := parseServeArgs([]string{"--no-open", "--relay-bind", "127.0.0.1:9545", "--billing-socket", "/tmp/b.sock", "--meter"}, noEnv)
	if err != nil {
		t.Fatal(err)
	}
	if opts.RelayBind != "127.0.0.1:9545" || opts.BillingSocket != "/tmp/b.sock" || !opts.Meter {
		t.Fatalf("opts = %+v", opts)
	}
}

// The environment is how a server the CLI starts gets the operator's relay
// settings: EnsureRunning's child inherits it. A flag still wins.
func TestServerOptionsFromEnvironment(t *testing.T) {
	env := map[string]string{
		"JUMPGATE_RELAY_BIND":     "127.0.0.1:9545",
		"JUMPGATE_BILLING_SOCKET": "/tmp/b.sock",
		"JUMPGATE_ERPC_URL":       "http://127.0.0.1:4100",
		"JUMPGATE_ERPC_PROJECT":   "p1",
		"JUMPGATE_METER":          "true",
		"JUMPGATE_RELAY_TOKEN":    "rt",
		"JUMPGATE_ADMIN_TOKEN":    "at",
	}
	getenv := func(k string) string { return env[k] }
	opts, err := parseServeArgs([]string{"--erpc-project", "flagwins"}, getenv)
	if err != nil {
		t.Fatal(err)
	}
	want := serverOptions{
		Bind: "127.0.0.1:8799", RelayBind: "127.0.0.1:9545", BillingSocket: "/tmp/b.sock",
		ERPCURL: "http://127.0.0.1:4100", ERPCProject: "flagwins", Meter: true,
		RelayToken: "rt", AdminToken: "at",
	}
	if opts != want {
		t.Fatalf("opts = %+v\nwant  %+v", opts, want)
	}
}

func TestServerOptionsRejectABadMeterValue(t *testing.T) {
	_, err := parseServeArgs(nil, func(k string) string {
		if k == "JUMPGATE_METER" {
			return "sometimes"
		}
		return ""
	})
	if err == nil || !strings.Contains(err.Error(), "JUMPGATE_METER") {
		t.Fatalf("err = %v, want a JUMPGATE_METER error", err)
	}
}

func taskNames(b *builtServer) []string {
	var names []string
	for _, t := range b.background {
		names = append(names, t.name)
	}
	return names
}

// One composition: with a relay configured, the built server carries the
// relay, the key admin, overlay autostart, the update check and the settle
// loop, whichever entry point built it.
func TestBuildServerWiresRelayKeyAdminAndBackgroundWork(t *testing.T) {
	shortHome(t)
	relayBind := freeAddr(t)
	b, err := buildServer(serverOptions{
		Bind: freeAddr(t), RelayBind: relayBind, BillingSocket: "/tmp/jg-no-such.sock",
		ERPCURL: "http://127.0.0.1:4000", RelayToken: "rt", AdminToken: "at", Meter: true,
	}, func() {}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if b.relay == nil {
		t.Error("no relay handler")
	}
	if b.keyAdmin == nil {
		t.Error("no key admin")
	}
	want := []string{"update-check", "overlay-autostart", "relay-listener", "relay-runtime"}
	if got := taskNames(b); !reflect.DeepEqual(got, want) {
		t.Fatalf("background = %v, want %v", got, want)
	}
	if b.shape.RelayBind != relayBind || !b.shape.Meter || !b.shape.KeyAdmin {
		t.Fatalf("shape = %+v", b.shape)
	}

	// The relay listener task really serves the data plane.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	for _, task := range b.background {
		if task.name == "relay-listener" {
			go task.run(ctx)
		}
	}
	deadline := time.Now().Add(3 * time.Second)
	for {
		c, err := net.DialTimeout("tcp", relayBind, 100*time.Millisecond)
		if err == nil {
			c.Close()
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("relay listener never came up: %v", err)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestBuildServerWithoutRelay(t *testing.T) {
	shortHome(t)
	b, err := buildServer(serverOptions{Bind: freeAddr(t), ERPCURL: "http://127.0.0.1:4000"}, func() {}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if b.relay != nil || b.keyAdmin != nil {
		t.Fatal("relay or key admin wired with no relay configured")
	}
	if got, want := taskNames(b), []string{"update-check", "overlay-autostart"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("background = %v, want %v", got, want)
	}
}

// A half-configured relay is fatal in both entry points.
func TestBuildServerRefusesAHalfConfiguredRelay(t *testing.T) {
	shortHome(t)
	_, err := buildServer(serverOptions{Bind: freeAddr(t), RelayBind: freeAddr(t), ERPCURL: "http://127.0.0.1:4000"}, func() {}, io.Discard)
	if err == nil {
		t.Fatal("a relay with no billing socket was accepted")
	}
}

// Decision: a controller key that will not open is tolerated in BOTH entry
// points. The server comes up, logs why, and the box routes answer 503
// no_controller_key with the reason, instead of `serve` dying and the CLI
// timing out on a server that never appears.
func TestBuildServerToleratesAKeyThatWillNotOpen(t *testing.T) {
	shortHome(t)
	writeControllerConfig(t, "file", "/nonexistent/controller.key", "0x0000000000000000000000000000000000000001")
	var log strings.Builder
	b, err := buildServer(serverOptions{Bind: freeAddr(t), ERPCURL: "http://127.0.0.1:4000"}, func() {}, &log)
	if err != nil {
		t.Fatalf("buildServer failed on a bad key: %v", err)
	}
	if b.signerErr == nil {
		t.Fatal("the key error was dropped")
	}
	if !strings.Contains(log.String(), "controller key not loaded") {
		t.Fatalf("log = %q", log.String())
	}
}

// When the app attaches to a running server built with other options, it says
// exactly which ones are not in effect.
func TestAttachWarningNamesEveryDifference(t *testing.T) {
	running := daemon.Info{PID: 42, HTTPAddr: "127.0.0.1:8799", Shape: &daemon.Shape{}}
	want := serverOptions{Bind: "127.0.0.1:8799", RelayBind: "127.0.0.1:9545", BillingSocket: "/b.sock", Meter: true, AdminToken: "at", RelayToken: "rt", ERPCURL: "http://127.0.0.1:4000"}
	msg := attachWarning(want, running)
	for _, s := range []string{"pid 42", "relay-bind", "billing-socket", "meter", "key admin", "jumpgate stop"} {
		if !strings.Contains(msg, s) {
			t.Errorf("warning lacks %q:\n%s", s, msg)
		}
	}
	same := daemon.Info{PID: 42, HTTPAddr: "127.0.0.1:8799", Shape: &daemon.Shape{RelayBind: "127.0.0.1:9545", BillingSocket: "/b.sock", ERPCURL: "http://127.0.0.1:4000", Meter: true, KeyAdmin: true}}
	if msg := attachWarning(want, same); msg != "" {
		t.Errorf("warned with identical options:\n%s", msg)
	}
	plain := serverOptions{Bind: "127.0.0.1:8799", ERPCURL: "http://127.0.0.1:4000"}
	if msg := attachWarning(plain, daemon.Info{PID: 1, HTTPAddr: "127.0.0.1:8799", Shape: &daemon.Shape{ERPCURL: "http://127.0.0.1:4000"}}); msg != "" {
		t.Errorf("warned with default options:\n%s", msg)
	}
	// An older server publishes no shape: say we cannot tell, if it matters.
	if msg := attachWarning(want, daemon.Info{PID: 7, HTTPAddr: "127.0.0.1:8799"}); !strings.Contains(msg, "jumpgate stop") {
		t.Errorf("no warning for a server with unknown options:\n%s", msg)
	}
}

// writeControllerConfig records a controller in the test HOME's config.json.
func writeControllerConfig(t *testing.T, store, ref, addr string) {
	t.Helper()
	if _, err := config.Update(func(c *config.Config) error {
		c.Controller = &config.Controller{KeyStore: store, KeyRef: ref, Address: addr}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

// D/F: a relay without --meter serves every valid key free. That is allowed,
// but it is said out loud at startup.
func TestBuildServerWarnsOfAnUnmeteredRelay(t *testing.T) {
	shortHome(t)
	opts := serverOptions{Bind: freeAddr(t), RelayBind: freeAddr(t), BillingSocket: "/tmp/jg-no-such.sock",
		ERPCURL: "http://127.0.0.1:4000", RelayToken: "rt", AdminToken: "at"}
	var log strings.Builder
	if _, err := buildServer(opts, func() {}, &log); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(log.String(), "WARNING") || !strings.Contains(log.String(), "--meter") {
		t.Fatalf("no unmetered warning: %q", log.String())
	}
	opts.Meter = true
	log.Reset()
	if _, err := buildServer(opts, func() {}, &log); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(log.String(), "WARNING") {
		t.Fatalf("warned with metering on: %q", log.String())
	}
}

// Fix round 1: the tokens leave the process environment once read, so no
// child (a local-target shell, a keychain or 1Password helper) inherits them.
func TestBuildServerUnsetsTheTokens(t *testing.T) {
	shortHome(t)
	t.Setenv("JUMPGATE_ADMIN_TOKEN", "at")
	t.Setenv("JUMPGATE_RELAY_TOKEN", "rt")
	opts, err := parseServeArgs(nil, os.Getenv)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := buildServer(opts, func() {}, io.Discard); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"JUMPGATE_ADMIN_TOKEN", "JUMPGATE_RELAY_TOKEN"} {
		if _, set := os.LookupEnv(k); set {
			t.Errorf("%s is still in the environment", k)
		}
	}
	if opts.AdminToken != "at" || opts.RelayToken != "rt" {
		t.Fatalf("tokens were not read first: %+v", opts)
	}
}

// A token can come from a file named by JUMPGATE_*_TOKEN_FILE, so it need
// not sit in a shell profile.
func TestTokensFromFiles(t *testing.T) {
	dir := t.TempDir()
	admin := filepath.Join(dir, "admin")
	relayTok := filepath.Join(dir, "relay")
	_ = os.WriteFile(admin, []byte("admin-from-file\n"), 0o600)
	_ = os.WriteFile(relayTok, []byte("relay-from-file\n"), 0o600)
	env := map[string]string{"JUMPGATE_ADMIN_TOKEN_FILE": admin, "JUMPGATE_RELAY_TOKEN_FILE": relayTok}
	opts, err := parseServeArgs(nil, func(k string) string { return env[k] })
	if err != nil {
		t.Fatal(err)
	}
	if opts.AdminToken != "admin-from-file" || opts.RelayToken != "relay-from-file" {
		t.Fatalf("opts = %+v", opts)
	}
	env["JUMPGATE_ADMIN_TOKEN"] = "both"
	if _, err := parseServeArgs(nil, func(k string) string { return env[k] }); err == nil {
		t.Fatal("a token set both directly and by file was accepted")
	}
	delete(env, "JUMPGATE_ADMIN_TOKEN")
	env["JUMPGATE_ADMIN_TOKEN_FILE"] = filepath.Join(dir, "missing")
	if _, err := parseServeArgs(nil, func(k string) string { return env[k] }); err == nil {
		t.Fatal("an unreadable token file was accepted")
	}
}

// vaultHome puts a fixture vault at ~/.jumpgate/vault holding a controller
// key recorded in config.json, runs setup on it unlocked, and leaves it
// locked on disk.
func vaultHome(t *testing.T, setup func(v *vault.Vault)) {
	t.Helper()
	shortHome(t)
	src, pass := vaulttest.Path(t)
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
	v, err := vault.Open(p)
	if err != nil {
		t.Fatal(err)
	}
	if err := v.Unlock(pass); err != nil {
		t.Fatal(err)
	}
	k, err := signer.CreateVault(v, secrets.NameControllerKey)
	if err != nil {
		t.Fatal(err)
	}
	if setup != nil {
		setup(v)
	}
	v.Lock()
	writeControllerConfig(t, string(signer.StoreVault), secrets.NameControllerKey, k.Address().Hex())
}

func controllerState(t *testing.T, b *builtServer) string {
	t.Helper()
	ts := httptest.NewServer(b.srv.Handler())
	defer ts.Close()
	req, _ := http.NewRequest("GET", ts.URL+"/api/controller", nil)
	req.Header.Set("Authorization", "Bearer "+b.token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var cv api.ControllerView
	if err := json.NewDecoder(res.Body).Decode(&cv); err != nil {
		t.Fatal(err)
	}
	return cv.State
}

func runTask(t *testing.T, b *builtServer, name string) {
	t.Helper()
	for _, task := range b.background {
		if task.name == name {
			task.run(context.Background())
			return
		}
	}
	t.Fatalf("no %s task in %v", name, taskNames(b))
}

// A relay whose tokens are in the vault serves nothing until an unlock, then
// comes up with the tokens the vault holds.
func TestRelayWithAVaultTokenStartsAfterUnlock(t *testing.T) {
	vaultHome(t, func(v *vault.Vault) {
		if err := v.Put(secrets.NameRelayToken, []byte("rt"), "test"); err != nil {
			t.Fatal(err)
		}
		if err := v.Put(secrets.NameAdminToken, []byte("at"), "test"); err != nil {
			t.Fatal(err)
		}
	})
	t.Setenv(credentialsDirEnv, "")
	relayBind := freeAddr(t)
	var log syncBuilder
	b, err := buildServer(serverOptions{
		Bind: freeAddr(t), RelayBind: relayBind, BillingSocket: "/tmp/jg-no-such.sock",
		ERPCURL: "http://127.0.0.1:4000", RelayToken: "vault:relay-token", AdminToken: "vault:admin-token", Meter: true,
	}, func() {}, &log)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(log.String(), "relay waits for the vault to be unlocked") {
		t.Fatalf("log = %q", log.String())
	}
	for _, n := range taskNames(b) {
		if strings.HasPrefix(n, "relay-") {
			t.Fatalf("relay task %s before unlock", n)
		}
	}
	if b.srv.RelayHandler() != nil {
		t.Fatal("a relay handler before unlock")
	}
	if !b.shape.KeyAdmin {
		t.Fatalf("shape = %+v", b.shape)
	}
	if _, err := b.srv.Unlock([]byte(vaulttest.Passphrase)); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		c, err := net.DialTimeout("tcp", relayBind, 100*time.Millisecond)
		if err == nil {
			c.Close()
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("relay listener never came up: %v; log %q", err, log.String())
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// syncBuilder is a strings.Builder safe for the log a background task writes.
type syncBuilder struct {
	mu sync.Mutex
	b  strings.Builder
}

func (s *syncBuilder) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *syncBuilder) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

// A vault reference must name the token's own entry.
func TestServerRefusesAForeignTokenReference(t *testing.T) {
	shortHome(t)
	_, err := buildServer(serverOptions{
		Bind: freeAddr(t), RelayBind: freeAddr(t), BillingSocket: "/tmp/jg-no-such.sock",
		ERPCURL: "http://127.0.0.1:4000", RelayToken: "vault:controller-key", AdminToken: "at",
	}, func() {}, io.Discard)
	if err == nil {
		t.Fatal("a relay token naming the controller key was accepted")
	}
}

func TestBuildServerStartsLockedWithAVaultKey(t *testing.T) {
	vaultHome(t, nil)
	t.Setenv(credentialsDirEnv, "")
	var log strings.Builder
	b, err := buildServer(serverOptions{Bind: freeAddr(t), ERPCURL: "http://127.0.0.1:4000"}, func() {}, &log)
	if err != nil {
		t.Fatal(err)
	}
	if !errors.Is(b.signerErr, vault.ErrLocked) {
		t.Fatalf("signerErr %v", b.signerErr)
	}
	if got := controllerState(t, b); got != "locked" {
		t.Fatalf("controller state %q", got)
	}
	if strings.Contains(log.String(), "box commands are unavailable") {
		t.Fatalf("a locked key logged as a missing one: %q", log.String())
	}
}

func TestUnlockRunsMigrationThenOpensTheKey(t *testing.T) {
	vaultHome(t, nil)
	t.Setenv(credentialsDirEnv, "")
	if _, err := config.Update(func(c *config.Config) error { c.AIKey = "sk-plain"; return nil }); err != nil {
		t.Fatal(err)
	}
	var log strings.Builder
	b, err := buildServer(serverOptions{Bind: freeAddr(t), ERPCURL: "http://127.0.0.1:4000"}, func() {}, &log)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := b.srv.Unlock([]byte(vaulttest.Passphrase)); err != nil {
		t.Fatal(err)
	}
	c, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.AIKey != "vault:ai-key" || c.SecretStore != config.SecretStoreVault {
		t.Fatalf("aiKey %q secretStore %q", c.AIKey, c.SecretStore)
	}
	if got := controllerState(t, b); got != "ok" {
		t.Fatalf("controller state %q", got)
	}
	if strings.Contains(log.String(), "sk-plain") {
		t.Fatalf("the log carries a secret: %q", log.String())
	}
}

// credentialDir writes kek as the systemd credential and points the
// environment at it.
func credentialDir(t *testing.T, kek []byte) {
	t.Helper()
	dir := t.TempDir()
	if err := fsperm.WriteFilePrivate(filepath.Join(dir, vaultCredName), kek); err != nil {
		t.Fatal(err)
	}
	t.Setenv(credentialsDirEnv, dir)
}

func TestCredentialUnlocksAtStart(t *testing.T) {
	kek := bytes.Repeat([]byte{7}, 32)
	vaultHome(t, func(v *vault.Vault) {
		if err := v.AddKEKSlot(vault.Slot{Kind: vault.SlotSystemdCreds, Cred: vaultCredName}, bytes.Clone(kek)); err != nil {
			t.Fatal(err)
		}
	})
	credentialDir(t, kek)
	if got, ok := credentialKEK(os.Getenv); !ok || !bytes.Equal(got, kek) {
		t.Fatalf("credentialKEK %x %v", got, ok)
	}
	b, err := buildServer(serverOptions{Bind: freeAddr(t), ERPCURL: "http://127.0.0.1:4000"}, func() {}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	runTask(t, b, "vault-auto-unlock")
	if got := b.srv.VaultState().State; got != "unlocked" {
		t.Fatalf("vault state %q", got)
	}
	if got := controllerState(t, b); got != "ok" {
		t.Fatalf("controller state %q", got)
	}
}

func TestBadCredentialStaysLocked(t *testing.T) {
	vaultHome(t, func(v *vault.Vault) {
		if err := v.AddKEKSlot(vault.Slot{Kind: vault.SlotSystemdCreds, Cred: vaultCredName}, bytes.Repeat([]byte{7}, 32)); err != nil {
			t.Fatal(err)
		}
	})
	for _, kek := range [][]byte{bytes.Repeat([]byte{8}, 32), []byte("short")} {
		credentialDir(t, kek)
		var log strings.Builder
		b, err := buildServer(serverOptions{Bind: freeAddr(t), ERPCURL: "http://127.0.0.1:4000"}, func() {}, &log)
		if err != nil {
			t.Fatal(err)
		}
		runTask(t, b, "vault-auto-unlock")
		if got := b.srv.VaultState().State; got != "locked" {
			t.Fatalf("vault state %q", got)
		}
		if !strings.Contains(log.String(), "the passphrase still works") {
			t.Fatalf("log %q", log.String())
		}
	}
}

func TestCredentialKEKWithoutADirectory(t *testing.T) {
	if kek, ok := credentialKEK(noEnv); ok || kek != nil {
		t.Fatalf("credentialKEK %x %v", kek, ok)
	}
}
