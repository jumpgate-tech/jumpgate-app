package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/apiclient"
	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/daemon"
	"github.com/valve-tech/jumpgate/internal/intent"
)

// subcommands maps a first argument to its handler. Anything else falls
// through to runApp, so `jumpgate` and `jumpgate --bind …` behave as before.
var subcommands = map[string]func(args []string) int{
	"serve": cmdServe, "relay": cmdRelay, "stop": cmdStop, "keys": cmdKeys, "agent": cmdAgent, "hosts": cmdHosts,
	"status": cmdIntent("status", intent.KindStatusRead), "disk": cmdIntent("disk", intent.KindDiskRead),
	"endpoints": cmdIntent("endpoints", intent.KindEndpointsRead), "firewall": cmdIntent("firewall", intent.KindFirewallRead),
	"logs": cmdLogs, "service": cmdService, "ssh": cmdSSH, "help": cmdHelp, "open": cmdOpen, "home": cmdHome, "tui": cmdTUI,
}

func main() {
	if guiSubcommandRefused(os.Args) {
		showErrorDialog("jumpgate-tray.exe is the desktop app and has no console. Run commands with jumpgate.exe in a terminal, for example:\n\n    jumpgate.exe " + strings.Join(os.Args[1:], " "))
		os.Exit(exitCode("usage"))
	}
	if err := migrateOnStartup(os.Args, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "jumpgate:", err)
		if isGUIExe() {
			// No console for the line above to reach (B-3).
			showErrorDialog("jumpgate: " + err.Error())
		}
		pauseIfStandalone(os.Stdin, os.Stderr)
		os.Exit(1)
	}
	if code, handled := dispatch(os.Args, os.Stderr); handled {
		os.Exit(code)
	}
	if isTerminalLaunch(os.Args) {
		os.Exit(runTerminalHome(context.Background(), os.Stdin, os.Stdout))
	}
	runApp()
}

// migrateOnStartup moves ~/.valve-node-app to ~/.jumpgate before any
// subcommand runs (R24): most of them create ~/.jumpgate first (the config
// lock, the run directory, a key), so a migration left to the web app or
// serve would find it in the way. `agent` runs on the box as root and never
// reads controller state, so a stray legacy directory there must not stop
// the agent. `relay` never touches controller state at all, migration
// included.
func migrateOnStartup(args []string, stderr io.Writer) error {
	if len(args) >= 2 && (args[1] == "agent" || args[1] == "relay") {
		return nil
	}
	moved, kept, err := config.MigrateLegacyDir()
	if moved {
		fmt.Fprintln(stderr, "jumpgate: moved ~/.valve-node-app to ~/.jumpgate")
	}
	for _, k := range kept {
		fmt.Fprintf(stderr, "jumpgate: left %s in place: ~/.jumpgate already has a file of that name; compare the two and remove one\n", k)
	}
	return err
}

// dispatch runs a subcommand. A first argument starting with "-" (or none) is
// left to runApp so `jumpgate --bind x` keeps working; any other word that is
// not a subcommand is a typo and must not silently start the web app.
func dispatch(args []string, stderr io.Writer) (code int, handled bool) {
	if len(args) < 2 || strings.HasPrefix(args[1], "-") {
		return 0, false
	}
	if fn, ok := subcommands[args[1]]; ok {
		return fn(args[2:]), true
	}
	names := make([]string, 0, len(subcommands))
	for n := range subcommands {
		names = append(names, n)
	}
	sort.Strings(names)
	fmt.Fprintf(stderr, "jumpgate: unknown command %q\ncommands: %s\n(run `jumpgate help` for the commands; `jumpgate serve` runs the server in the foreground)\n", args[1], strings.Join(names, " "))
	return exitCode("usage"), true
}

// exitCode maps an outcome to the process exit status: 0 ok, 1 refused or
// failed, 2 usage, 3 unreachable, 4 security. Any other outcome is a server
// error code, whose class the api registry holds.
func exitCode(outcome string) int {
	switch outcome {
	case "ok":
		return api.ExitOK
	case "refused", "failed":
		return api.ExitFailed
	case "usage":
		return api.ExitUsage
	}
	return api.Code(outcome).Exit()
}

// connect returns a client of the running server, starting one if needed.
// A variable so tests can point the CLI at an in-process server: the real one
// would start this binary as a detached `serve`.
var connect = func(ctx context.Context) (*apiclient.Client, error) {
	exe, _ := os.Executable()
	return apiclient.Connect(ctx, apiclient.Options{Start: true, Exe: exe})
}

// parseSSHTarget reads user@host[:port], with IPv6 hosts in brackets.
func parseSSHTarget(s string) (user, host string, port int, err error) {
	v, err := api.ParseLogin(s)
	return v.User, v.Host, v.Port, err
}

// usage reports bad arguments and returns exit status 2, which means usage and
// nothing else.
func usage(format string, a ...any) int {
	fmt.Fprintf(os.Stderr, "jumpgate: "+format+"\n", a...)
	return exitCode("usage")
}

// failed reports a runtime failure or a refusal and returns exit status 1.
func failed(format string, a ...any) int {
	fmt.Fprintf(os.Stderr, "jumpgate: "+format+"\n", a...)
	return exitCode("failed")
}

// exit is os.Exit; a seam for tests.
var exit = os.Exit

// jgFile is a path inside ~/.jumpgate. With no home directory there is no
// safe place for jumpgate's files, so it stops instead of writing keys into
// the working directory (M-12).
func jgFile(parts ...string) string {
	dir, err := config.Dir()
	if err != nil {
		fmt.Fprintln(os.Stderr, "jumpgate:", err)
		exit(1)
	}
	return filepath.Join(append([]string{dir}, parts...)...)
}

// mustRequest builds an authenticated JSON POST to the local server.
func mustRequest(ctx context.Context, info daemon.Info, path string, body []byte) *http.Request {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, daemon.BaseURL+path, bytes.NewReader(body))
	if err != nil {
		panic(err) // the URL is a constant plus a path segment built by this program
	}
	req.Header.Set("Authorization", "Bearer "+info.Token)
	req.Header.Set("Content-Type", "application/json")
	return req
}

// apiError is the server's error JSON, plus the HTTP status it came with: the
// shared type every Go client decodes.
type apiError = api.Error

// readAPIError decodes an error response into the shared error type. A body
// that is not the error JSON still yields the status and its code, so the
// operator is never shown nothing.
func readAPIError(res *http.Response) apiError {
	b, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	return *api.Decode(res.StatusCode, b)
}

// reportServerErrorFrom is reportServerError for an answer from the server
// info describes. A 404 from a server of another version is the wire contract
// changing under the CLI, so it says to restart that server instead. Since the
// error contract a bare 404 decodes to not_found; any more specific code came
// from a server that knew the route.
func reportServerErrorFrom(w io.Writer, what string, info daemon.Info, e apiError) int {
	if e.Status == http.StatusNotFound && (e.Code == "" || e.Code == api.CodeNotFound) {
		if skew := daemon.SkewWarning(info); skew != "" {
			fmt.Fprintf(w, "jumpgate: %s: the running server does not know this request (%s)\n  -> %s\n", what, e.Message, strings.TrimPrefix(skew, "jumpgate: "))
			return exitCode("failed")
		}
	}
	return reportServerError(w, what, e)
}

// reportServerError prints a server error with its hint (or the registry's)
// and returns the exit status for its code, from the api registry: security
// codes 4, unreachable 3, no_controller_key and not_paired 2 (the operator
// must run a command first), any other code 1. An error without a code is a
// plain failure; status 2 is otherwise usage only.
func reportServerError(w io.Writer, what string, e apiError) int {
	hint := e.Hint
	if hint == "" {
		hint = api.HintFor(e.Code)
	}
	switch {
	case e.Code == "":
		fmt.Fprintf(w, "jumpgate: %s: %s\n", what, e.Message)
		return exitCode("failed")
	case e.Code.Exit() == api.ExitSecurity:
		fmt.Fprintf(w, "jumpgate: SECURITY: %s: %s\n", what, e.Message)
	case e.Code == api.CodeUnreachable:
		fmt.Fprintf(w, "jumpgate: %s: could not reach the box: %s\n", what, e.Message)
	case e.Code == api.CodeAgentHTTP:
		fmt.Fprintf(w, "jumpgate: %s: refused: %s\n", what, e.Message)
	default:
		fmt.Fprintf(w, "jumpgate: %s: %s\n", what, e.Message)
	}
	if hint != "" {
		fmt.Fprintf(w, "  -> %s\n", hint)
	}
	return e.Code.Exit()
}
