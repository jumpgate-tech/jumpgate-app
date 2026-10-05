// This file backs the panel's Docker readiness gate. GET /api/docker reports
// whether the LOCAL machine — the box the one-click gateway is provisioned on —
// has Docker present and its daemon running. POST /api/docker/start launches
// Docker Desktop, OrbStack or colima on macOS so the daemon comes up, so the power
// button can wait for Docker instead of failing on a raw "docker not found".
package server

import (
	"errors"
	"net/http"
	"runtime"
	"strings"

	"github.com/valve-tech/jumpgate/internal/ops"
)

// dockerStatusResponse is what GET /api/docker returns for the local machine.
type dockerStatusResponse struct {
	// Present is true when a docker CLI is on the local PATH.
	Present bool `json:"present"`
	// Running is true when the daemon answered (`docker info`).
	Running bool `json:"running"`
	// CanStart is true when the app can launch Docker itself — macOS with the
	// CLI present but the daemon down. The UI shows a "Start Docker" path then.
	CanStart bool `json:"canStart"`
	// Hint is operator-facing guidance for the current state (install it, or
	// start it). Empty when Docker is present and running.
	Hint string `json:"hint,omitempty"`
}

const dockerStartHint = "Docker is installed but not running. Start Docker Desktop, OrbStack or colima (`colima start`)."

func (s *Server) handleDockerStatus(w http.ResponseWriter, r *http.Request) {
	info, err := ops.ProbeDocker(r.Context(), s.newLocalExecutor())

	var resp dockerStatusResponse
	var absent *ops.DockerAbsentError
	switch {
	case errors.As(err, &absent):
		resp.Hint = absent.Hint
	case err != nil:
		// A non-absent error means the local executor itself failed — unusual
		// on the control plane, but report it rather than pretend Docker's fine.
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	default:
		resp.Present = info.Present
		resp.Running = info.DaemonReachable
		if resp.Present && !resp.Running {
			resp.Hint = dockerStartHint
		}
	}
	resp.CanStart = runtime.GOOS == "darwin" && resp.Present && !resp.Running
	writeJSON(w, http.StatusOK, resp)
}

// handleDockerStart launches Docker Desktop, OrbStack or colima on macOS so the
// daemon comes up. It returns as soon as the launch is issued — the daemon
// takes a while to be ready, so the caller polls GET /api/docker until running
// flips true. Only macOS can open a desktop app; elsewhere the operator starts
// the engine themselves.
func (s *Server) handleDockerStart(w http.ResponseWriter, r *http.Request) {
	if runtime.GOOS != "darwin" {
		writeError(w, http.StatusBadRequest, "auto-start is only available on macOS; start the Docker engine yourself")
		return
	}
	// Prefer the runtime the active docker context points at, then try the
	// rest. `docker context show` failing just means no preference.
	ctxName := ""
	if cr, cerr := s.newLocalExecutor().Run(r.Context(), "docker context show", nil); cerr == nil && cr.ExitCode == 0 {
		ctxName = strings.TrimSpace(cr.Stdout)
	}
	res, err := s.newLocalExecutor().Run(r.Context(), macStartCommand(ctxName), nil)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, struct {
		Started bool `json:"started"`
	}{Started: res.ExitCode == 0})
}

// macStartCommand builds the shell line that brings a Docker runtime up on
// macOS. Each launcher falls through (`||`) to the next: `open -a` exits
// non-zero when the app is not installed, and colima is only tried when it is
// on PATH (`colima start` is idempotent, so a running VM is harmless). The
// runtime the active docker context names goes first.
func macStartCommand(dockerContext string) string {
	const (
		desktop = "open -a Docker"
		orb     = "open -a OrbStack"
		colima  = "command -v colima >/dev/null 2>&1 && colima start"
	)
	order := []string{desktop, orb, colima}
	switch c := strings.ToLower(dockerContext); {
	case strings.Contains(c, "colima"):
		order = []string{colima, desktop, orb}
	case strings.Contains(c, "orb"):
		order = []string{orb, desktop, colima}
	}
	return strings.Join(order, " || ")
}
