// Package api is the wire contract between the local jumpgate server and its
// Go clients, the CLI and the TUI: the error codes and the request and
// response types both sides use. It imports only leaf packages (catalog,
// intent), so a client never links the server and the server never depends on
// a client.
package api

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"

	"github.com/valve-tech/jumpgate/internal/intent"
)

// Code is a machine-readable error code. Every code the server writes is a
// constant below; internal/server has a test that refuses string literals, so
// this file is the registry.
type Code string

const (
	// Generic codes, derived from the HTTP status when nothing more specific
	// applies (see CodeForStatus).
	CodeBadRequest     Code = "bad_request"
	CodeUnauthorized   Code = "unauthorized"
	CodeForbidden      Code = "forbidden"
	CodeNotFound       Code = "not_found"
	CodeConflict       Code = "conflict"
	CodeTooLarge       Code = "too_large"
	CodeInternal       Code = "internal"
	CodeNotImplemented Code = "not_implemented"
	CodeUpstream       Code = "upstream_failed"
	CodeUnavailable    Code = "unavailable"
	CodeTimeout        Code = "timeout"

	// Targets and transports.
	CodeTargetNotFound  Code = "target_not_found"
	CodeTargetExists    Code = "target_exists"
	CodeTargetNotSetUp  Code = "target_not_set_up"
	CodeUnreachable     Code = "unreachable"
	CodeHostKey         Code = "host_key"
	CodeUnknownHost     Code = "unknown_host"
	CodeBadReceipt      Code = "bad_receipt"
	CodeAgentHTTP       Code = "agent_http"
	CodeNotPaired       Code = "not_paired"
	CodeNoControllerKey Code = "no_controller_key"
	// CodeControllerKeyMismatch: the key store opened, but the key in it is
	// not the controller identity the boxes trust.
	CodeControllerKeyMismatch Code = "controller_key_mismatch"
	CodeRejected              Code = "rejected"
	CodeAgentFailed           Code = "agent_failed"
	CodeLocalUnsupported      Code = "local_unsupported"
	CodeLocalNeedsTerminal    Code = "local_needs_terminal"

	// The pairing stream's own failures.
	CodeStepFailed   Code = "step_failed"
	CodeVerifyFailed Code = "verify_failed"
	CodeRecordFailed Code = "record_failed"
	CodeTransportKey Code = "transport_key"

	// Containers, gateways and VPNs (formerly kebab-case).
	CodeDockerAbsent      Code = "docker_absent"
	CodeDockerUnreachable Code = "docker_unreachable"
	CodeServiceNotCreated Code = "service_not_created"
	CodeNotConfigured     Code = "not_configured"
	CodeGatewayNotFound   Code = "gateway_not_found"
	CodeVPNNotFound       Code = "vpn_not_found"
	CodeVPNServerNotFound Code = "vpn_server_not_found"
)

// Exit classes: the CLI's process exit statuses, published in the README.
const (
	ExitOK          = 0
	ExitFailed      = 1
	ExitUsage       = 2
	ExitUnreachable = 3
	ExitSecurity    = 4
)

type codeInfo struct {
	hint string
	exit int
}

// registry is every code with its default hint and exit class. A response
// with its own hint keeps it; HintFor fills in only an empty one.
var registry = map[Code]codeInfo{
	CodeBadRequest:     {"", ExitFailed},
	CodeUnauthorized:   {"sign in again: run `jumpgate open` (browser) or retry the command (CLI); if it persists, `jumpgate stop` and retry", ExitFailed},
	CodeForbidden:      {"a page from another origin cannot change this server", ExitFailed},
	CodeNotFound:       {"", ExitFailed},
	CodeConflict:       {"", ExitFailed},
	CodeTooLarge:       {"", ExitFailed},
	CodeInternal:       {"", ExitFailed},
	CodeNotImplemented: {"", ExitFailed},
	CodeUpstream:       {"", ExitFailed},
	CodeUnavailable:    {"", ExitFailed},
	CodeTimeout:        {"", ExitFailed},

	CodeTargetNotFound:  {"run `jumpgate hosts list` to see the names this controller knows", ExitFailed},
	CodeTargetExists:    {"", ExitFailed},
	CodeTargetNotSetUp:  {"set the node up first (the web app's setup wizard), or pair the box so its own agent answers", ExitFailed},
	CodeUnreachable:     {"check that the box is up and reachable over SSH", ExitUnreachable},
	CodeHostKey:         {"the box's SSH host key does not match the one on record: possibly a man-in-the-middle, or the box was rebuilt. Check the key on the box's console; only if it legitimately changed, remove the old line from ~/.jumpgate/confirmed_hosts (and ~/.ssh/known_hosts) and confirm the new one", ExitSecurity},
	CodeUnknownHost:     {"nobody has confirmed this box's SSH host key; compare its fingerprint with the box's console and confirm it (`jumpgate hosts add`, or Hosts in the TUI)", ExitSecurity},
	CodeBadReceipt:      {"the answer was not signed by this box's paired agent; do not trust this box until you re-pair it", ExitSecurity},
	CodeAgentHTTP:       {"the agent socket refused the request before reading it: this connection is not allowed on the socket (the tunnel user is not in the jumpgate group, or a local uid is not enrolled), or the request was too large", ExitFailed},
	CodeNotPaired:       {"pair this box first with `jumpgate hosts add`", ExitUsage},
	CodeNoControllerKey: {"run `jumpgate keys init`, then `jumpgate stop` so the server restarts with the key", ExitUsage},
	CodeControllerKeyMismatch: {"the key store holds a different key than the controller identity your boxes trust. " +
		"Restore the original key (keychain item, 1Password item or key file), then run `jumpgate stop`; do not re-pair boxes to the new key unless you meant to replace the controller", ExitSecurity},
	CodeRejected:           {"", ExitFailed},
	CodeAgentFailed:        {"the agent ran the request and it failed on the box; the message is the box's", ExitFailed},
	CodeLocalUnsupported:   {"this computer cannot run node commands itself; run this on a Linux machine you added with --ssh", ExitFailed},
	CodeLocalNeedsTerminal: {"run `jumpgate hosts add NAME --local` in a terminal (it asks for your sudo password there), or run jumpgate as root", ExitFailed},

	CodeStepFailed:   {"", ExitFailed},
	CodeVerifyFailed: {"", ExitFailed},
	CodeRecordFailed: {"", ExitFailed},
	CodeTransportKey: {"", ExitFailed},

	CodeDockerAbsent:      {"", ExitFailed},
	CodeDockerUnreachable: {"", ExitFailed},
	CodeServiceNotCreated: {"this service has not been created on the target yet; create it first", ExitFailed},
	CodeNotConfigured:     {"", ExitFailed},
	CodeGatewayNotFound:   {"", ExitFailed},
	CodeVPNNotFound:       {"", ExitFailed},
	CodeVPNServerNotFound: {"", ExitFailed},
}

// Known reports whether c is in the registry.
func Known(c Code) bool { _, ok := registry[c]; return ok }

// Codes lists the registry, sorted, for tests and documentation.
func Codes() []Code {
	out := make([]Code, 0, len(registry))
	for c := range registry {
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// HintFor is c's default hint, or "".
func HintFor(c Code) string { return registry[c].hint }

// Exit is the CLI exit status for an outcome with this code. Unknown codes
// are plain failures.
func (c Code) Exit() int {
	if info, ok := registry[c]; ok {
		return info.exit
	}
	return ExitFailed
}

// CodeForStatus is the generic code for an HTTP status, used when a handler
// has nothing more specific to say.
func CodeForStatus(status int) Code {
	switch status {
	case http.StatusBadRequest:
		return CodeBadRequest
	case http.StatusUnauthorized:
		return CodeUnauthorized
	case http.StatusForbidden:
		return CodeForbidden
	case http.StatusNotFound:
		return CodeNotFound
	case http.StatusConflict:
		return CodeConflict
	case http.StatusRequestEntityTooLarge:
		return CodeTooLarge
	case http.StatusNotImplemented:
		return CodeNotImplemented
	case http.StatusBadGateway:
		return CodeUpstream
	case http.StatusServiceUnavailable:
		return CodeUnavailable
	case http.StatusGatewayTimeout:
		return CodeTimeout
	}
	if status >= 500 {
		return CodeInternal
	}
	return CodeBadRequest
}

// rejectionHints turns each signed rejection reason into one line an operator
// can act on. They used to live in the CLI; the server now sends them.
var rejectionHints = map[string]string{
	intent.ReasonWrongAgent:       "this box's agent identity changed since pairing; re-pair it",
	intent.ReasonExpired:          "the intent expired before the box checked it; check both clocks (enable NTP)",
	intent.ReasonClockSkew:        "the clocks disagree by more than 60s; enable NTP on this machine and the box",
	intent.ReasonBadSignature:     "the box could not verify this controller's signature; re-pair it",
	intent.ReasonUnauthorizedKind: "this controller is not enrolled on the box for that; pair it",
	intent.ReasonUnknownKind:      "the box runs an older agent; re-pair it to upgrade the agent",
	intent.ReasonStaleSeq:         "another process is using this controller key against this box",
	intent.ReasonReplayedNonce:    "the same intent was sent twice; retry",
	intent.ReasonBusy:             "another request from this controller is still running on the box",
	intent.ReasonInvalidPayload:   "the request was malformed; this is a jumpgate bug, please report it",
	intent.ReasonValidation:       "the box's node configuration is invalid; check /etc/jumpgate/node.json",
	intent.ReasonNotSetUp:         "no node is set up on this box yet; run setup first",
	intent.ReasonReplayState:      "the box's replay record is damaged; on the box, inspect /var/lib/jumpgate/replay.json then run `sudo jumpgate agent reset-replay --yes`",
}

// RejectionHint is the remedy for a signed rejection reason, or "".
func RejectionHint(reason string) string { return rejectionHints[reason] }

// Error is every error body the API sends: {"error", "hint", "code"} plus the
// fields a few codes carry. Status is the HTTP status, set by Decode.
type Error struct {
	Status      int    `json:"-"`
	Message     string `json:"error"`
	Hint        string `json:"hint,omitempty"`
	Code        Code   `json:"code"`
	Reason      string `json:"reason,omitempty"`      // with CodeRejected: the agent's signed reason
	Host        string `json:"host,omitempty"`        // with CodeUnknownHost
	Fingerprint string `json:"fingerprint,omitempty"` // with CodeUnknownHost
}

func (e *Error) Error() string {
	if e.Code == "" {
		return e.Message
	}
	return e.Message + " (" + string(e.Code) + ")"
}

// maxMessage bounds a plain-text body taken as a message.
const maxMessage = 1024

// Decode reads an error response. A body that is not the error JSON (an old
// server's plain-text 401, a proxy page) still yields a message, a code from
// the status and that code's hint, so a person is never shown nothing.
func Decode(status int, body []byte) *Error {
	e := &Error{}
	if json.Unmarshal(body, e) != nil || e.Message == "" {
		msg := strings.TrimSpace(string(body))
		if len(msg) > maxMessage {
			msg = msg[:maxMessage]
		}
		if msg == "" {
			msg = http.StatusText(status)
		}
		e.Message = msg
	}
	e.Status = status
	if e.Code == "" {
		e.Code = CodeForStatus(status)
	}
	if e.Hint == "" {
		e.Hint = HintFor(e.Code)
	}
	return e
}
