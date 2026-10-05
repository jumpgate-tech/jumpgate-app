// Command jumpgate sets up and monitors an Ethereum / PulseChain /
// PulseChain-v4 node: one binary, guided setup, sync monitoring, and AI log
// explanations, fronted by a local token-gated web UI.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/valve-tech/jumpgate/internal/buildinfo"
	"github.com/valve-tech/jumpgate/internal/daemon"
)

// bindFlagUsage is --bind's help text. Called out here (rather than inline
// in flag.String) so it's independently testable.
const bindFlagUsage = "address to bind the local server to. " +
	"WARNING: binding beyond 127.0.0.1 exposes full control of your servers over plain HTTP"

func runApp() {
	opts, err := addServerFlags(flag.CommandLine, os.Getenv)
	if err != nil {
		log.Fatalf("jumpgate: %v", err)
	}
	noOpen := flag.Bool("no-open", false, "do not open a browser window automatically")
	tray := flag.Bool("tray", false, "open the UI in a native desktop window (tiny-app mode) instead of a browser tab; requires a build made with -tags tray")
	showVersion := flag.Bool("version", false, "print the version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println(buildinfo.Version())
		return
	}

	if warning := bindWarningLine(opts.Bind); warning != "" {
		fmt.Fprintln(os.Stderr, warning)
	}

	// main has already moved ~/.valve-node-app to ~/.jumpgate.

	ctx, stop := shutdownContext(context.Background())
	defer stop()
	windowed := *tray || inAppBundle()

	// One server per user (R25): the app takes the same lock as `jumpgate
	// serve`. If a server is already up — another app launch, or one a CLI
	// command started — open it instead of failing to bind a second one, and
	// say plainly which of this launch's options it was not built with.
	holder, running, err := claimAppInstance(ctx, 10*time.Second)
	if err != nil {
		log.Fatalf("jumpgate: %v", err)
	}
	if running != nil {
		if skew := daemon.SkewWarning(*running); skew != "" {
			fmt.Fprintln(os.Stderr, skew)
		}
		if warning := attachWarning(*opts, *running); warning != "" {
			fmt.Fprintln(os.Stderr, warning)
		}
		openRunningServer(ctx, *running, windowed, *noOpen)
		return
	}
	defer holder.Release()

	// The same composition root as `jumpgate serve`: see buildServer.
	b, err := buildServer(*opts, stop, os.Stderr)
	if err != nil {
		holder.Release()
		log.Fatalf("jumpgate: %v", err)
	}
	s, token, bind := b.srv, b.token, opts.Bind

	url := appURL(daemon.Info{HTTPAddr: bind, Token: token})
	fmt.Println(url)
	b.start(ctx, os.Stdout)

	// Launched by double-clicking the macOS .app bundle, the OS passes no
	// flags — so a bundled build enters tray mode on its own. An explicit
	// --tray still works for running the tray binary straight from a shell.
	if windowed {
		if !trayBuilt {
			log.Fatalf("jumpgate: --tray needs a build made with the tray tag: go build -tags tray ./cmd/jumpgate")
		}
		// The window is the foreground; HTTP runs behind it. runWindow must own
		// the main goroutine (the platform webview owns the UI run loop), so the
		// server goes to a background goroutine. Closing the window returns from
		// runWindow; we then cancel ctx to shut the server down cleanly.
		srvErr := make(chan error, 1)
		go func() { srvErr <- serveAndPublish(ctx, stop, s, holder, bind, token, &b.shape) }()
		if err := waitReady(ctx, bind); err != nil {
			log.Fatalf("jumpgate: server did not come up: %v", err)
		}
		runWindow(ctx, url)
		stop()
		<-srvErr // wait for the shutdown we just asked for; its error is expected
		return
	}

	if !*noOpen {
		openBrowser(url)
	}

	if err := serveAndPublish(ctx, stop, s, holder, bind, token, &b.shape); err != nil {
		holder.Release()
		log.Fatalf("jumpgate: server: %v", err)
	}
}

// openRunningServer shows the already-running server instead of starting a
// second one: in the tray window when this launch is windowed, otherwise in
// the browser. server.json is 0600 in a 0700 directory, so its token is this
// user's own.
func openRunningServer(ctx context.Context, info daemon.Info, windowed, noOpen bool) {
	url := appURL(info)
	fmt.Fprintf(os.Stderr, "jumpgate: already running, pid %d; opening it\n", info.PID)
	fmt.Println(url)
	if windowed {
		if !trayBuilt {
			log.Fatalf("jumpgate: --tray needs a build made with the tray tag: go build -tags tray ./cmd/jumpgate")
		}
		runWindow(ctx, url)
		return
	}
	if !noOpen {
		openBrowser(url)
	}
}

// shutdownContext returns a context canceled by the first SIGINT or SIGTERM.
//
// signal.NotifyContext keeps catching those signals until its stop function
// is called, and main used to call it only on the tray path. So once a
// graceful shutdown was under way, every further Ctrl-C was absorbed by a
// context that was already canceled, and an operator whose shutdown stalled
// had no way to kill the process from the terminal. Here the handler is
// unregistered as soon as the first signal lands, and before the returned
// context reports done, so a second Ctrl-C gets the default behaviour and
// ends the process.
func shutdownContext(parent context.Context) (context.Context, context.CancelFunc) {
	sigCtx, stopSignals := signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM)
	ctx, cancel := context.WithCancel(parent)
	go func() {
		<-sigCtx.Done()
		stopSignals()
		cancel()
	}()
	return ctx, func() {
		stopSignals()
		cancel()
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
// it is. jumpgate's local server is token-gated but plain HTTP: binding
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
