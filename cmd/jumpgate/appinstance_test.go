package main

import (
	"context"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/daemon"
	"github.com/valve-tech/jumpgate/internal/server"
	"github.com/valve-tech/jumpgate/internal/signer"
	"github.com/valve-tech/jumpgate/internal/testutil"
)

// shortHome isolates HOME under /tmp: the server socket lives in
// ~/.jumpgate/run and unix socket paths must stay short.
func shortHome(t *testing.T) {
	t.Helper()
	testutil.Home(t)
}

func freeAddr(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	return ln.Addr().String()
}

// R25: the tray/web entry point takes the same single-instance lock as serve.
func TestClaimAppInstanceTakesTheLock(t *testing.T) {
	shortHome(t)
	holder, running, err := claimAppInstance(context.Background(), 0)
	if err != nil || holder == nil || running != nil {
		t.Fatalf("claimAppInstance = %v, %v, %v; want the lock", holder, running, err)
	}
	defer holder.Release()
	if _, err := daemon.Acquire(); !errors.Is(err, daemon.ErrAlreadyRunning) {
		t.Fatalf("a second server could take the lock: %v", err)
	}
}

// R25: with `jumpgate serve` already up, the app finds it through server.json
// instead of starting a second server on the same port. serveAndPublish is
// what both entry points use to come up and advertise themselves.
func TestClaimAppInstanceFindsTheRunningServer(t *testing.T) {
	shortHome(t)
	h, err := daemon.Acquire()
	if err != nil {
		t.Fatal(err)
	}
	defer h.Release()
	bind, token := freeAddr(t), server.NewSessionToken()
	ctx, stop := context.WithCancel(context.Background())
	s := server.New(server.Config{Bind: bind, Token: token, Shutdown: stop})
	done := make(chan error, 1)
	go func() { done <- serveAndPublish(ctx, stop, s, h, bind, token, nil) }()
	defer func() { stop(); <-done }()

	_, running, err := claimAppInstance(context.Background(), 5*time.Second)
	if err != nil || running == nil {
		t.Fatalf("claimAppInstance = %v, %v; want the running server", running, err)
	}
	if running.HTTPAddr != bind || running.Token != token || running.PID != os.Getpid() {
		t.Fatalf("running = %+v, want addr %s and its token", running, bind)
	}
	if got := appURL(*running); got != "http://"+bind+"/?token="+token {
		t.Errorf("appURL = %q", got)
	}
}

// A lock holder that never answers is not a server to open; the app says so
// instead of waiting forever or starting a second server.
func TestClaimAppInstanceGivesUpOnASilentHolder(t *testing.T) {
	shortHome(t)
	h, err := daemon.Acquire()
	if err != nil {
		t.Fatal(err)
	}
	defer h.Release()
	_, _, err = claimAppInstance(context.Background(), 300*time.Millisecond)
	if err == nil || !strings.Contains(err.Error(), "jumpgate stop") {
		t.Fatalf("err = %v, want a hint naming jumpgate stop", err)
	}
}

// P29: serve and the app both start through loadServerConfig, which tightens
// existing secrets and says so, without failing startup.
func TestServerStartupTightensExistingSecrets(t *testing.T) {
	home := testutil.Home(t)
	keyFile := filepath.Join(home, ".jumpgate", "keys", "controller.key")
	k, err := signer.GenerateKeyFile(keyFile)
	if err != nil {
		t.Fatal(err)
	}
	c := config.Config{Controller: &config.Controller{KeyStore: string(signer.StoreFile), KeyRef: keyFile, Address: k.Address().Hex()}}
	if err := c.Save(); err != nil {
		t.Fatal(err)
	}
	transport := filepath.Join(home, ".jumpgate", "ssh", "jumpgate_ed25519")
	if err := os.MkdirAll(filepath.Dir(transport), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(transport, []byte("old key"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfgFile := filepath.Join(home, ".jumpgate", "config.json")
	for _, p := range []string{keyFile, transport, cfgFile} {
		testutil.Loosen(t, p)
	}

	var stderr strings.Builder
	cfg, err := loadServerConfig(&stderr)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Controller == nil || cfg.Controller.KeyRef != keyFile {
		t.Fatalf("config not loaded: %+v", cfg.Controller)
	}
	for _, p := range []string{keyFile, transport, cfgFile} {
		testutil.AssertPrivate(t, p)
		if !strings.Contains(stderr.String(), p) {
			t.Errorf("startup did not report tightening %s:\n%s", p, stderr.String())
		}
	}
	// The key opens again once it is private.
	if _, err := openControllerKey(cfg); err != nil {
		t.Fatalf("openControllerKey after startup: %v", err)
	}
}

// A controller key file that cannot be made private stops the server
// instead of being used. (A symlink is the stand-in, as in config's test.)
func TestServerStartupRefusesAKeyItCannotRestrict(t *testing.T) {
	testutil.RequireUnix(t)
	home := testutil.Home(t)
	real := filepath.Join(home, "real.key")
	if _, err := signer.GenerateKeyFile(real); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(home, "controller.key")
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	c := config.Config{Controller: &config.Controller{KeyStore: string(signer.StoreFile), KeyRef: link}}
	if err := c.Save(); err != nil {
		t.Fatal(err)
	}
	if _, err := loadServerConfig(io.Discard); err == nil || !strings.Contains(err.Error(), link) {
		t.Fatalf("loadServerConfig with a symlinked key = %v, want an error naming %s", err, link)
	}
}
