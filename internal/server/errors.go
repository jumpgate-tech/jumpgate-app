package server

import (
	"context"
	"errors"
	"net"
	"net/http"
	"os"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/executor"
)

// writeAPIError is the one writer of error responses. Every error the API
// sends is {error, hint, code} with a registered code: a handler with nothing
// specific to say gets the status's generic code, and an empty hint is filled
// from the registry.
func writeAPIError(w http.ResponseWriter, status int, e api.Error) {
	if e.Code == "" {
		e.Code = api.CodeForStatus(status)
	}
	if e.Hint == "" {
		e.Hint = api.HintFor(e.Code)
	}
	writeJSON(w, status, e)
}

// writeError is the long tail: a message and a status-derived code.
func writeError(w http.ResponseWriter, status int, msg string) {
	writeAPIError(w, status, api.Error{Message: msg})
}

// writeErrorDetail is an error a client may branch on.
func writeErrorDetail(w http.ResponseWriter, status int, msg, hint string, code api.Code) {
	writeAPIError(w, status, api.Error{Message: msg, Hint: hint, Code: code})
}

// writeTargetNotFound replaces the two spellings ("target not found", "no such
// target") the routes used to send.
func writeTargetNotFound(w http.ResponseWriter) {
	writeErrorDetail(w, http.StatusNotFound, "target not found", "", api.CodeTargetNotFound)
}

func writeTargetNotSetUp(w http.ResponseWriter) {
	writeErrorDetail(w, http.StatusConflict, "target has not completed setup", "", api.CodeTargetNotSetUp)
}

// writeDialError reports a failure to open a target's legacy executor. A
// host-key failure is a security error and never "unreachable"; a refused or
// timed-out connection is "unreachable", the same code the agent routes use
// for the same outage.
func writeDialError(w http.ResponseWriter, err error) {
	var unknown *executor.UnknownHostError
	var op *net.OpError
	switch {
	case errors.As(err, &unknown):
		writeAPIError(w, http.StatusConflict, api.Error{Message: err.Error(), Code: api.CodeUnknownHost, Host: unknown.Host, Fingerprint: unknown.Fingerprint})
	case errors.Is(err, executor.ErrHostKeyMismatch):
		writeErrorDetail(w, http.StatusBadGateway, err.Error(), "", api.CodeHostKey)
	case errors.Is(err, executor.ErrNoPOSIXShell):
		writeErrorDetail(w, http.StatusConflict, err.Error(), "", api.CodeLocalUnsupported)
	case errors.As(err, &op), errors.Is(err, context.DeadlineExceeded), errors.Is(err, os.ErrDeadlineExceeded):
		writeErrorDetail(w, http.StatusGatewayTimeout, err.Error(), "", api.CodeUnreachable)
	default:
		writeError(w, http.StatusBadGateway, err.Error())
	}
}
