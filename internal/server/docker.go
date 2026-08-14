// This file backs the panel's Docker readiness gate. GET /api/docker reports
// whether the LOCAL machine — the box the one-click gateway is provisioned on —
// has Docker present and its daemon running. POST /api/docker/start launches
// Docker Desktop (or OrbStack) on macOS so the daemon comes up, so the power
// button can wait for Docker instead of failing on a raw "docker not found".
package server

import (
	"errors"
	"net/http"
	"runtime"

	"github.com/valve-tech/valve-node-app/internal/ops"
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

const dockerStartHint = "Docker is installed but not running. Start Docker Desktop or OrbStack."

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

// handleDockerStart launches Docker Desktop (or OrbStack) on macOS so the
// daemon comes up. It returns as soon as the launch is issued — the daemon
// takes a while to be ready, so the caller polls GET /api/docker until running
// flips true. Only macOS can open a desktop app; elsewhere the operator starts
// the engine themselves.
func (s *Server) handleDockerStart(w http.ResponseWriter, r *http.Request) {
	if runtime.GOOS != "darwin" {
		writeError(w, http.StatusBadRequest, "auto-start is only available on macOS; start the Docker engine yourself")
		return
	}
	// Try Docker Desktop, then OrbStack. `open -a` returns non-zero when the
	// named app is not installed, so the || falls through to the next one.
	res, err := s.newLocalExecutor().Run(r.Context(), "open -a Docker || open -a OrbStack", nil)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, struct {
		Started bool `json:"started"`
	}{Started: res.ExitCode == 0})
}
