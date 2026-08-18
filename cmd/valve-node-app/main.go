// Command valve-node-app sets up and monitors an Ethereum / PulseChain /
// PulseChain-v4 node: one binary, guided setup, sync monitoring, and AI log
// explanations, fronted by a local token-gated web UI.
package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/valve-tech/valve-node-app/internal/buildinfo"
	"github.com/valve-tech/valve-node-app/internal/config"
	"github.com/valve-tech/valve-node-app/internal/relay"
	"github.com/valve-tech/valve-node-app/internal/server"
)

//go:embed all:web/dist
var embeddedUI embed.FS

// bindFlagUsage is --bind's help text. Called out here (rather than inline
// in flag.String) so it's independently testable.
const bindFlagUsage = "address to bind the local server to. " +
	"WARNING: binding beyond 127.0.0.1 exposes full control of your servers over plain HTTP"

func main() {
	bind := flag.String("bind", "127.0.0.1:8799", bindFlagUsage)
	// The data plane is off unless an operator asks for it. It is a separate
	// listener from --bind on purpose: --bind carries the session token that
	// controls the operator's servers, and this one carries customer traffic
	// authenticated by key. Bind it to the interface Caddy reaches, never to
	// 0.0.0.0 — Caddy is the public door and the TLS terminator, and a
	// plaintext keyed URL would expose the key on the wire.
	relayBind := flag.String("relay-bind", "", "address to serve the metered RPC data plane on (empty disables it)")
	billingSocket := flag.String("billing-socket", "", "unix socket of the billing key store")
	erpcURL := flag.String("erpc-url", "http://127.0.0.1:4000", "base URL of the keyless eRPC the relay forwards to")
	projectID := flag.String("erpc-project", "", "eRPC project segment (empty means main)")
	// Metering is off by default. Serving unmetered is the status quo, so
	// charging customers is a deliberate act rather than a side effect of
	// pointing the relay at a key store.
	metering := flag.Bool("meter", false, "charge credits for metered RPC (off means serve without billing)")
	noOpen := flag.Bool("no-open", false, "do not open a browser window automatically")
	tray := flag.Bool("tray", false, "open the UI in a native desktop window (tiny-app mode) instead of a browser tab; requires a build made with -tags tray")
	showVersion := flag.Bool("version", false, "print the version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println(buildinfo.Version())
		return
	}

	if warning := bindWarningLine(*bind); warning != "" {
		fmt.Fprintln(os.Stderr, warning)
	}

	// Load (or lazily create on first Save) valve-node-app's local state —
	// known targets, AI provider settings — from ~/.valve-node-app/config.json.
	// The server re-reads it per-request rather than holding this value, so
	// it's only loaded here to fail fast on a corrupt file before the
	// server starts serving.
	if _, err := config.Load(); err != nil {
		log.Fatalf("valve-node-app: load config: %v", err)
	}

	uiFS, err := fs.Sub(embeddedUI, "web/dist")
	if err != nil {
		log.Fatalf("valve-node-app: embedded UI: %v", err)
	}

	// The relay's credential never arrives as a flag. A flag lands in the
	// process listing, where any local user reads it.
	relayHandler, relayRuntime, err := relay.Build(relay.BuildOptions{
		RelayBind:      *relayBind,
		BillingSocket:  *billingSocket,
		RelayToken:     os.Getenv("JUMPGATE_RELAY_TOKEN"),
		ERPCURL:        *erpcURL,
		ProjectID:      *projectID,
		EnableMetering: *metering,
	})
	if err != nil {
		// A half-configured relay is fatal rather than quietly off. Serving
		// unmetered traffic is worse than serving none: the operator sells
		// access and would be giving it away with nothing to report it.
		log.Fatalf("valve-node-app: relay: %v", err)
	}

	// Key management is the operator's surface and uses the ADMIN credential,
	// which mints and revokes keys. The relay's credential cannot do either.
	adminClient, err := relay.BuildAdmin(*billingSocket, os.Getenv("JUMPGATE_ADMIN_TOKEN"))
	if err != nil {
		log.Fatalf("valve-node-app: key store: %v", err)
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
		Bind:      *bind,
		Token:     token,
		UI:        uiFS,
		Relay:     relayHandler,
		Keys:      keyAdmin,
		RelayBind: *relayBind,
	})

	url := fmt.Sprintf("http://%s/?token=%s", *bind, token)
	fmt.Println(url)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Warm the update check in the background so the first UI poll is instant.
	// It respects the disabled setting and reports failures through the API,
	// so nothing here needs its result.
	go s.PrimeUpdateCheck(ctx)

	// Bring up any overlays the operator marked "start with the app". Runs off
	// the serving path in its own goroutine: an autostart overlay on an
	// unreachable box must never delay the UI coming up, and one overlay failing
	// must not stop the others — AutostartOverlays returns per-overlay results,
	// which we log and otherwise let be.
	go func() {
		for _, r := range s.AutostartOverlays(ctx) {
			if r.Err != nil {
				log.Printf("valve-node-app: autostart overlay %q: %v", r.ID, r.Err)
			} else {
				log.Printf("valve-node-app: autostart overlay %q is up", r.ID)
			}
		}
	}()

	// The data plane runs beside the control plane on its own listener. A
	// failure here is the relay's failure and must not be mistaken for the UI
	// failing to come up, so it is logged rather than folded into the main
	// server's error.
	if relayHandler != nil {
		go func() {
			if err := s.ListenAndServeRelay(ctx); err != nil {
				log.Printf("valve-node-app: relay data plane: %v", err)
			}
		}()
		// The background loops are not optional. Without them leased credits are
		// never settled back and the beacon pool never re-probes, so a customer's
		// balance would sit stranded and a recovered node would stay out of
		// rotation for the life of the process.
		go relayRuntime.Run(ctx)
		fmt.Printf("metered RPC data plane on %s\n", *relayBind)
	}

	// Launched by double-clicking the macOS .app bundle, the OS passes no
	// flags — so a bundled build enters tray mode on its own. An explicit
	// --tray still works for running the tray binary straight from a shell.
	if *tray || inAppBundle() {
		if !trayBuilt {
			log.Fatalf("valve-node-app: --tray needs a build made with the tray tag: go build -tags tray ./cmd/valve-node-app")
		}
		// The window is the foreground; HTTP runs behind it. runWindow must own
		// the main goroutine (the platform webview owns the UI run loop), so the
		// server goes to a background goroutine. Closing the window returns from
		// runWindow; we then cancel ctx to shut the server down cleanly.
		srvErr := make(chan error, 1)
		go func() { srvErr <- s.ListenAndServe(ctx) }()
		if err := waitReady(ctx, *bind); err != nil {
			log.Fatalf("valve-node-app: server did not come up: %v", err)
		}
		runWindow(ctx, url)
		stop()
		<-srvErr // wait for the shutdown we just asked for; its error is expected
		return
	}

	if !*noOpen {
		openBrowser(url)
	}

	if err := s.ListenAndServe(ctx); err != nil {
		log.Fatalf("valve-node-app: server: %v", err)
	}
}

// inAppBundle reports whether this process was launched from inside a macOS
// .app bundle (its executable lives at Foo.app/Contents/MacOS/…). Used to
// default to tray mode when double-clicked, where there are no CLI flags to
// pass --tray. Always false off darwin, where the layout does not occur.
func inAppBundle() bool {
	if runtime.GOOS != "darwin" {
		return false
	}
	exe, err := os.Executable()
	if err != nil {
		return false
	}
	return isAppBundlePath(exe)
}

// isAppBundlePath reports whether exe sits at the canonical macOS bundle
// location Foo.app/Contents/MacOS/binary. Split out from inAppBundle so the
// path rule is testable without a real bundle on disk.
func isAppBundlePath(exe string) bool {
	return strings.Contains(exe, ".app/Contents/MacOS/")
}

// waitReady blocks until the server is accepting connections on bind, so the
// tiny-app window never loads before there is something to serve it. Bounded so
// a server that never binds fails loudly instead of hanging the window.
func waitReady(ctx context.Context, bind string) error {
	d := net.Dialer{Timeout: 200 * time.Millisecond}
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		conn, err := d.DialContext(ctx, "tcp", bind)
		if err == nil {
			_ = conn.Close()
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("timed out waiting for %s to accept connections", bind)
}

// bindWarningLine returns a loud warning line when bind's host is not
// loopback (127.0.0.1 or localhost — the server's safe default), or "" if
// it is. valve-node-app's local server is token-gated but plain HTTP: binding
// it beyond loopback puts full control of every configured target (setup,
// shell-equivalent install/build commands, log access) on the network
// reachable at that address, over an unencrypted channel a network
// observer can read the session token off of.
func bindWarningLine(bind string) string {
	host, _, err := net.SplitHostPort(bind)
	if err != nil {
		// No port (or an unparsable address) — treat the whole string as
		// the host, e.g. a bare "127.0.0.1" or "0.0.0.0".
		host = bind
	}
	if host == "127.0.0.1" || strings.EqualFold(host, "localhost") {
		return ""
	}
	return fmt.Sprintf(
		"WARNING: binding to %s exposes full control of your servers over plain HTTP — "+
			"anyone who can reach %s can drive setup, run install/build commands, and read logs. "+
			"Only bind beyond 127.0.0.1 on a trusted network (e.g. behind an SSH tunnel), never on the open internet.",
		bind, bind,
	)
}

// openBrowser opens url in the user's default browser. Best-effort: errors
// are ignored since this is a convenience, not a requirement.
func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	_ = cmd.Start()
}
