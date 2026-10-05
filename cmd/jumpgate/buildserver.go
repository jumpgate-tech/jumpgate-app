package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"strconv"
	"strings"

	"github.com/valve-tech/jumpgate/cmd/jumpgate/web"
	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/daemon"
	"github.com/valve-tech/jumpgate/internal/relay"
	"github.com/valve-tech/jumpgate/internal/secretenv"
	"github.com/valve-tech/jumpgate/internal/server"
)

// serverOptions is everything that shapes the controller server. The web app
// (runApp) and `jumpgate serve` fill it from the same flags and environment
// and hand it to buildServer, the one composition root, so the two entry
// points differ only in whether they open a browser or a window.
type serverOptions struct {
	Bind          string
	RelayBind     string
	BillingSocket string
	ERPCURL       string
	ERPCProject   string
	Meter         bool
	// The credentials never arrive as flags: a flag lands in the process
	// listing, where any local user reads it.
	RelayToken string // JUMPGATE_RELAY_TOKEN
	AdminToken string // JUMPGATE_ADMIN_TOKEN
}

// Environment variables that default the server flags. They are how a server
// the CLI starts on its own (daemon.EnsureRunning runs `serve --no-open`, which
// inherits the CLI's environment) gets the same relay as the app: set them
// once in the shell profile or the service unit, and every entry point agrees.
const (
	envRelayBind     = "JUMPGATE_RELAY_BIND"
	envBillingSocket = "JUMPGATE_BILLING_SOCKET"
	envERPCURL       = "JUMPGATE_ERPC_URL"
	envERPCProject   = "JUMPGATE_ERPC_PROJECT"
	envMeter         = "JUMPGATE_METER"
	envRelayToken    = "JUMPGATE_RELAY_TOKEN"
	envAdminToken    = "JUMPGATE_ADMIN_TOKEN"
)

const defaultERPCURL = "http://127.0.0.1:4000"

// readToken reads a credential from the environment variable name, or from the
// file named by name+"_FILE" (one token, surrounding whitespace ignored). The
// file form is preferred: it keeps the token out of shell profiles and out of
// every process that inherits the environment. Setting both is an error, as is
// a named file that is unreadable, empty, or not private to this user (see
// readPrivateFile: a symlink, a non-regular file, group/other permission bits
// or another owner are all refused).
func readToken(getenv func(string) string, name string) (string, error) {
	direct, file := getenv(name), getenv(name+"_FILE")
	if file == "" {
		return direct, nil
	}
	if direct != "" {
		return "", fmt.Errorf("both %s and %s_FILE are set; use one", name, name)
	}
	b, err := readPrivateFile(file)
	if err != nil {
		return "", fmt.Errorf("%s_FILE: %w", name, err)
	}
	tok := strings.TrimSpace(string(b))
	if tok == "" {
		return "", fmt.Errorf("%s_FILE: %s is empty", name, file)
	}
	return tok, nil
}

// secretEnvNames are the credentials the server reads and then removes from
// its own environment, so no child it launches can inherit them.
var secretEnvNames = []string{envRelayToken, envAdminToken, envRelayToken + "_FILE", envAdminToken + "_FILE"}

// addServerFlags registers the server flags on fs, each defaulting from its
// environment variable, and returns the options the parse fills in. A flag
// given on the command line wins over the environment.
func addServerFlags(fs *flag.FlagSet, getenv func(string) string) (*serverOptions, error) {
	admin, err := readToken(getenv, envAdminToken)
	if err != nil {
		return nil, err
	}
	o := &serverOptions{AdminToken: admin}
	fs.StringVar(&o.Bind, "bind", "127.0.0.1:8799", bindFlagUsage)
	if err := addRelayFlags(fs, getenv, o); err != nil {
		return nil, err
	}
	return o, nil
}

// addRelayFlags registers the data-plane flags, shared by the controller
// server and `jumpgate relay`, filling o's relay fields. Each defaults from its
// environment variable; the relay token comes only from the environment.
func addRelayFlags(fs *flag.FlagSet, getenv func(string) string, o *serverOptions) error {
	orDefault := func(key, def string) string {
		if v := getenv(key); v != "" {
			return v
		}
		return def
	}
	meter := false
	if v := getenv(envMeter); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return fmt.Errorf("%s=%q: want true or false", envMeter, v)
		}
		meter = b
	}
	tok, err := readToken(getenv, envRelayToken)
	if err != nil {
		return err
	}
	o.RelayToken = tok
	// The data plane is off unless an operator asks for it. It is a separate
	// listener from --bind on purpose: --bind carries the session token that
	// controls the operator's servers, and this one carries customer traffic
	// authenticated by key. Bind it to the interface Caddy reaches, never to
	// 0.0.0.0 — Caddy is the public door and the TLS terminator, and a
	// plaintext keyed URL would expose the key on the wire.
	fs.StringVar(&o.RelayBind, "relay-bind", getenv(envRelayBind), "address to serve the metered RPC data plane on (empty disables it; env "+envRelayBind+"). "+
		"In the controller this is supported for now; `jumpgate relay` is the recommended deployment, since it keeps the data plane out of the process that holds the controller key")
	fs.StringVar(&o.BillingSocket, "billing-socket", getenv(envBillingSocket), "unix socket of the billing key store (env "+envBillingSocket+")")
	fs.StringVar(&o.ERPCURL, "erpc-url", orDefault(envERPCURL, defaultERPCURL), "base URL of the keyless eRPC the relay forwards to (env "+envERPCURL+")")
	fs.StringVar(&o.ERPCProject, "erpc-project", getenv(envERPCProject), "eRPC project segment, empty means main (env "+envERPCProject+")")
	// Metering is off by default. Serving unmetered is the status quo, so
	// charging customers is a deliberate act rather than a side effect of
	// pointing the relay at a key store.
	fs.BoolVar(&o.Meter, "meter", meter, "charge credits for metered RPC; off means serve without billing (env "+envMeter+")")
	return nil
}

// relayBuildOptions is the relay's startup configuration from o.
func relayBuildOptions(o serverOptions) relay.BuildOptions {
	return relay.BuildOptions{
		RelayBind:      o.RelayBind,
		BillingSocket:  o.BillingSocket,
		RelayToken:     o.RelayToken,
		ERPCURL:        o.ERPCURL,
		ProjectID:      o.ERPCProject,
		EnableMetering: o.Meter,
	}
}

// newServeFlagSet is `jumpgate serve`'s flag set before the server flags.
func newServeFlagSet() *flag.FlagSet {
	fset := flag.NewFlagSet("serve", flag.ContinueOnError)
	_ = fset.Bool("no-open", true, "accepted for symmetry; serve never opens a browser")
	return fset
}

// parseServeArgs reads `jumpgate serve`'s arguments.
func parseServeArgs(args []string, getenv func(string) string) (serverOptions, error) {
	fset := newServeFlagSet()
	opts, err := addServerFlags(fset, getenv)
	if err != nil {
		return serverOptions{}, err
	}
	if err := fset.Parse(args); err != nil {
		return serverOptions{}, err
	}
	return *opts, nil
}

// backgroundTask is work a built server needs running beside its listeners.
type backgroundTask struct {
	name string
	run  func(ctx context.Context)
}

// builtServer is a composed controller server and what it needs started.
type builtServer struct {
	srv        *server.Server
	token      string
	opts       serverOptions
	shape      daemon.Shape
	relay      http.Handler
	keyAdmin   server.KeyAdmin
	signerErr  error
	background []backgroundTask
}

// buildServer is the single composition root for the controller server. Both
// entry points call it, so a server the CLI auto-starts has the same relay, key
// admin, overlay autostart and settle loop the app would have.
//
// A controller key that will not open is tolerated, in both entry points: the
// server comes up, logs why, and the box routes answer 503 no_controller_key
// with the reason. The web UI does not need the key, and a CLI command that
// auto-started the server gets a clear answer instead of a server that never
// appears. Everything else that is wrong (a corrupt config, a half-configured
// relay) is fatal.
func buildServer(opts serverOptions, shutdown func(), logw io.Writer) (*builtServer, error) {
	logf := func(format string, a ...any) { fmt.Fprintf(logw, "jumpgate: "+format+"\n", a...) }
	// opts already holds the tokens; drop them from the environment before
	// anything here can launch a child (a key-store helper, a local target).
	secretenv.Unset(secretEnvNames...)

	// Fail fast on a corrupt config before serving. The server re-reads it per
	// request rather than holding this value.
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	sgn, keyErr := openControllerKey(cfg)
	if keyErr != nil {
		logf("controller key not loaded, box commands are unavailable: %v", keyErr)
		sgn = nil
	}

	uiFS, err := fs.Sub(web.FS, "dist")
	if err != nil {
		return nil, fmt.Errorf("embedded UI: %w", err)
	}

	relayOpts := relayBuildOptions(opts)
	relayHandler, relayRuntime, err := relay.Build(relayOpts)
	if err != nil {
		// A half-configured relay is fatal rather than quietly off. Serving
		// unmetered traffic is worse than serving none: the operator sells
		// access and would be giving it away with nothing to report it.
		return nil, fmt.Errorf("relay: %w", err)
	}
	if w := relay.UnmeteredWarning(relayOpts); w != "" {
		logf("%s", w)
	}

	// Key management is the operator's surface and uses the ADMIN credential,
	// which mints and revokes keys. The relay's credential cannot do either.
	adminClient, err := relay.BuildAdmin(opts.BillingSocket, opts.AdminToken)
	if err != nil {
		return nil, fmt.Errorf("key store: %w", err)
	}
	// Assign only when non-nil. A nil *AdminClient inside a non-nil interface
	// would pass every nil check and then panic on the first click, instead of
	// answering the clean 501 a gateway with no key store should give.
	var keyAdmin server.KeyAdmin
	if adminClient != nil {
		keyAdmin = adminClient
	}

	token := server.NewSessionToken()
	s := server.New(server.Config{
		Bind:      opts.Bind,
		Token:     token,
		UI:        uiFS,
		Relay:     relayHandler,
		Keys:      keyAdmin,
		RelayBind: opts.RelayBind,
		Signer:    sgn,
		SignerErr: keyErr,
		Shutdown:  shutdown,
	})

	b := &builtServer{
		srv: s, token: token, opts: opts, relay: relayHandler, keyAdmin: keyAdmin, signerErr: keyErr,
		shape: daemon.Shape{
			RelayBind: opts.RelayBind, BillingSocket: opts.BillingSocket, ERPCURL: opts.ERPCURL,
			ERPCProject: opts.ERPCProject, Meter: opts.Meter, KeyAdmin: keyAdmin != nil,
		},
	}
	// Warm the update check so the first UI poll is instant. It respects the
	// disabled setting and reports failures through the API.
	b.background = append(b.background, backgroundTask{"update-check", s.PrimeUpdateCheck})
	// Bring up any overlays marked "start with the app". Off the serving path:
	// an overlay on an unreachable box must never delay the server, and one
	// failing must not stop the others.
	b.background = append(b.background, backgroundTask{"overlay-autostart", func(ctx context.Context) {
		for _, r := range s.AutostartOverlays(ctx) {
			if r.Err != nil {
				logf("autostart overlay %q: %v", r.ID, r.Err)
			} else {
				logf("autostart overlay %q is up", r.ID)
			}
		}
	}})
	if relayHandler != nil {
		// The data plane runs beside the control plane on its own listener. A
		// failure here is the relay's and must not be mistaken for the UI
		// failing to come up, so it is logged rather than folded in.
		b.background = append(b.background, backgroundTask{"relay-listener", func(ctx context.Context) {
			if err := s.ListenAndServeRelay(ctx); err != nil {
				logf("relay data plane: %v", err)
			}
		}})
		// Not optional: without it leased credits are never settled back and the
		// beacon pool never re-probes.
		b.background = append(b.background, backgroundTask{"relay-runtime", relayRuntime.Run})
	}
	return b, nil
}

// start runs every background task in its own goroutine.
func (b *builtServer) start(ctx context.Context, out io.Writer) {
	for _, t := range b.background {
		go t.run(ctx)
	}
	if b.relay != nil {
		fmt.Fprintf(out, "metered RPC data plane on %s\n", b.opts.RelayBind)
	}
}

// attachWarning explains, when this launch found a server already running,
// which of its options that server was not built with. Empty when they agree.
func attachWarning(want serverOptions, running daemon.Info) string {
	wantShape := daemon.Shape{
		RelayBind: want.RelayBind, BillingSocket: want.BillingSocket, ERPCURL: want.ERPCURL,
		ERPCProject: want.ERPCProject, Meter: want.Meter, KeyAdmin: want.BillingSocket != "" && want.AdminToken != "",
	}
	var diffs []string
	if want.Bind != running.HTTPAddr {
		diffs = append(diffs, fmt.Sprintf("bind: this launch wants %s, it serves %s", want.Bind, running.HTTPAddr))
	}
	if running.Shape == nil {
		// An older server: its options are unknown. Only worth a word when this
		// launch asks for something beyond the defaults.
		if wantShape.RelayBind != "" || wantShape.BillingSocket != "" || wantShape.Meter || len(diffs) > 0 {
			diffs = append(diffs, "relay and key admin: the running server does not report its options")
		}
	} else {
		have := *running.Shape
		str := func(name, w, h string) {
			if w != h {
				diffs = append(diffs, fmt.Sprintf("%s: this launch wants %q, it has %q", name, w, h))
			}
		}
		str("relay-bind", wantShape.RelayBind, have.RelayBind)
		str("billing-socket", wantShape.BillingSocket, have.BillingSocket)
		if wantShape.RelayBind != "" || have.RelayBind != "" {
			str("erpc-url", wantShape.ERPCURL, have.ERPCURL)
			str("erpc-project", wantShape.ERPCProject, have.ERPCProject)
		}
		if wantShape.Meter != have.Meter {
			diffs = append(diffs, fmt.Sprintf("meter: this launch wants %t, it has %t", wantShape.Meter, have.Meter))
		}
		if wantShape.KeyAdmin != have.KeyAdmin {
			diffs = append(diffs, fmt.Sprintf("key admin: this launch wants %s, it has %s", onOff(wantShape.KeyAdmin), onOff(have.KeyAdmin)))
		}
	}
	if len(diffs) == 0 {
		return ""
	}
	return fmt.Sprintf("jumpgate: WARNING: attaching to the server already running (pid %d), which was built with different options; this launch's options are NOT in effect:\n  %s\n"+
		"  -> stop it with `jumpgate stop` and launch again to apply them. A server a CLI command started takes its relay settings from the JUMPGATE_* environment.",
		running.PID, strings.Join(diffs, "\n  "))
}

func onOff(b bool) string {
	if b {
		return "on"
	}
	return "off"
}
