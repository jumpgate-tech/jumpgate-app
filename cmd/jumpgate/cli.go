package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/daemon"
	"github.com/valve-tech/jumpgate/internal/intent"
)

// subcommands maps a first argument to its handler. Anything else falls
// through to runApp, so `jumpgate` and `jumpgate --bind …` behave as before.
var subcommands = map[string]func(args []string) int{
	"serve": cmdServe, "stop": cmdStop, "keys": cmdKeys, "agent": cmdAgent, "hosts": cmdHosts,
	"status": cmdIntent("status", intent.KindStatusRead), "disk": cmdIntent("disk", intent.KindDiskRead),
	"endpoints": cmdIntent("endpoints", intent.KindEndpointsRead), "firewall": cmdIntent("firewall", intent.KindFirewallRead),
	"logs": cmdLogs, "service": cmdService,
}

func main() {
	if code, handled := dispatch(os.Args, os.Stderr); handled {
		os.Exit(code)
	}
	runApp()
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
	fmt.Fprintf(stderr, "jumpgate: unknown command %q\ncommands: %s\n(run `jumpgate` with no arguments, or with flags such as --bind, for the web app)\n", args[1], strings.Join(names, " "))
	return exitCode("usage"), true
}

// remedies turns each rejection code, and each error code the server adds, into
// one line an operator can act on. A server hint, when it sent one, wins.
var remedies = map[string]string{
	intent.ReasonWrongAgent:       "this box's agent identity changed since pairing; re-pair with `jumpgate hosts add`",
	intent.ReasonExpired:          "the intent expired before the box checked it; check both clocks (enable NTP)",
	intent.ReasonClockSkew:        "the clocks disagree by more than 60s; enable NTP on this machine and the box",
	intent.ReasonBadSignature:     "the box could not verify this controller's signature; re-pair with `jumpgate hosts add`",
	intent.ReasonUnauthorizedKind: "this controller is not enrolled on the box for that; pair it with `jumpgate hosts add`",
	intent.ReasonUnknownKind:      "the box runs an older agent; re-run `jumpgate hosts add` to upgrade it",
	intent.ReasonStaleSeq:         "another process is using this controller key against this box",
	intent.ReasonReplayedNonce:    "the same intent was sent twice; retry the command",
	intent.ReasonBusy:             "another command from this controller is still running on the box",
	intent.ReasonInvalidPayload:   "the request was malformed; this is a jumpgate bug, please report it",
	intent.ReasonValidation:       "the box's node configuration is invalid; check /etc/jumpgate/node.json",
	intent.ReasonNotSetUp:         "no node is set up on this box yet; run setup first",
	intent.ReasonReplayState:      "the box's replay record is damaged; on the box, inspect /var/lib/jumpgate/replay.json then run `sudo jumpgate agent reset-replay --yes`",

	"unreachable":       "check that the box is up and reachable over SSH",
	"bad_receipt":       "the answer was not signed by this box's paired agent; do not trust this box until you re-pair it",
	"agent_http":        "the agent socket refused the connection: the tunnel user must be in the jumpgate group, or a local uid must be enrolled (`jumpgate agent enroll --local-uid`)",
	"unknown_host":      "confirm the box's host key with `jumpgate hosts add`, then pair again",
	"no_controller_key": "run `jumpgate keys init`, then `jumpgate stop` so the server restarts with the key",
	"not_paired":        "pair this box first with `jumpgate hosts add`",
}

// exitCode maps an outcome to the process exit status: 0 ok, 1 refused or
// failed, 2 usage, 3 unreachable, 4 security. The server's error codes are
// outcomes too, so the table covers every one the CLI maps.
func exitCode(outcome string) int {
	switch outcome {
	case "ok":
		return 0
	case "refused", "failed", "agent_http":
		return 1
	case "usage", "no_controller_key", "not_paired":
		return 2
	case "unreachable":
		return 3
	case "bad_receipt", "host_key", "unknown_host":
		return 4
	}
	return 1
}

// parseSSHTarget reads user@host[:port], with IPv6 hosts in brackets.
func parseSSHTarget(s string) (user, host string, port int, err error) {
	at := strings.LastIndex(s, "@")
	if at <= 0 || at == len(s)-1 {
		return "", "", 0, fmt.Errorf("want user@host[:port], got %q", s)
	}
	user, rest := s[:at], s[at+1:]
	if h, p, splitErr := net.SplitHostPort(rest); splitErr == nil {
		n, convErr := strconv.Atoi(p)
		if convErr != nil || n <= 0 || n > 65535 {
			return "", "", 0, fmt.Errorf("bad port in %q", s)
		}
		if h == "" {
			return "", "", 0, fmt.Errorf("empty host in %q", s)
		}
		return user, h, n, nil
	}
	switch {
	case strings.HasPrefix(rest, "["):
		if !strings.HasSuffix(rest, "]") || strings.Count(rest, "[") != 1 || strings.Count(rest, "]") != 1 {
			return "", "", 0, fmt.Errorf("unbalanced brackets in %q", s)
		}
	case strings.ContainsAny(rest, "[]"):
		return "", "", 0, fmt.Errorf("unbalanced brackets in %q", s)
	case strings.Contains(rest, ":"):
		return "", "", 0, fmt.Errorf("bad port in %q", s)
	}
	host = strings.Trim(rest, "[]")
	if host == "" {
		return "", "", 0, fmt.Errorf("empty host in %q", s)
	}
	return user, host, 0, nil
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

// jgFile is a path inside ~/.jumpgate.
func jgFile(parts ...string) string {
	dir, err := config.Dir()
	if err != nil {
		dir = "."
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

// apiError is the server's error JSON, plus the HTTP status it came with.
// Host and Fingerprint are only set by pair's unknown_host answer.
type apiError struct {
	Status      int    `json:"-"`
	Error       string `json:"error"`
	Hint        string `json:"hint"`
	Code        string `json:"code"`
	Host        string `json:"host"`
	Fingerprint string `json:"fingerprint"`
}

// readAPIError decodes an error response. A body that is not the error JSON
// still yields the status, so the operator is never shown nothing.
func readAPIError(res *http.Response) apiError {
	e := apiError{Status: res.StatusCode}
	_ = json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&e)
	if e.Error == "" {
		e.Error = res.Status
	}
	return e
}

// reportServerError prints a server error with its hint (or the CLI's remedy)
// and returns the exit status for its code: unreachable 3; bad_receipt and
// unknown_host 4; agent_http 1; no_controller_key and not_paired 2 (the
// operator must run a command first); any other code, or none, 1. Status 2 is
// otherwise usage only.
func reportServerError(w io.Writer, what string, e apiError) int {
	hint := e.Hint
	if hint == "" {
		hint = remedies[e.Code]
	}
	switch e.Code {
	case "":
		fmt.Fprintf(w, "jumpgate: %s: %s\n", what, e.Error)
		return exitCode("failed")
	case "bad_receipt":
		fmt.Fprintf(w, "jumpgate: SECURITY: %s: %s\n", what, e.Error)
	case "unreachable":
		fmt.Fprintf(w, "jumpgate: %s: could not reach the box: %s\n", what, e.Error)
	case "agent_http":
		fmt.Fprintf(w, "jumpgate: %s: refused: %s\n", what, e.Error)
	default:
		fmt.Fprintf(w, "jumpgate: %s: %s\n", what, e.Error)
	}
	if hint != "" {
		fmt.Fprintf(w, "  -> %s\n", hint)
	}
	switch e.Code {
	case "unreachable", "bad_receipt", "agent_http", "unknown_host", "no_controller_key", "not_paired":
		return exitCode(e.Code)
	}
	return exitCode("failed")
}
