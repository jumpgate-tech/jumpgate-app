package main

import (
	"context"
	"flag"
	"io"
	"net"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/daemon"
)

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
