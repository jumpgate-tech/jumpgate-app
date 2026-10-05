// This file backs the panel's Docker readiness gate. GET /api/docker reports
// whether the LOCAL machine — the box the one-click gateway is provisioned on —
// has Docker present and its daemon running. POST /api/docker/start launches
// Docker Desktop, OrbStack or colima on macOS so the daemon comes up, so the power
// button can wait for Docker instead of failing on a raw "docker not found".
package server

import (
	"context"
	"errors"
	"log"
	"net/http"
	"runtime"
	"strings"
	"time"

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
	// WindowsContainers is true when an engine answered but runs Windows
	// containers, which cannot run jumpgate's Linux images. Running is then
	// false: the engine is up but unusable, and the UI's not-running path
	// shows Hint without trying to start or provision anything (spec D35).
	WindowsContainers bool `json:"windowsContainers,omitempty"`
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
		resp.WindowsContainers = info.WindowsContainers()
		resp.Running = info.DaemonReachable && !resp.WindowsContainers
		switch {
		case resp.WindowsContainers:
			resp.Hint = info.WindowsContainersHint()
		case resp.Present && !resp.Running:
			resp.Hint = dockerStartHint
		}
	}
	resp.CanStart = runtime.GOOS == "darwin" && resp.Present && !resp.Running && !resp.WindowsContainers
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
	plan := macStartPlan(ctxName)
	started := false
	if plan.openCmd != "" {
		res, err := s.newLocalExecutor().Run(r.Context(), plan.openCmd, nil)
		if err != nil {
			writeError(w, http.StatusBadGateway, err.Error())
			return
		}
		started = res.ExitCode == 0
	}
	if !started && plan.colima {
		s.startColimaDetached()
		started = true
	}
	writeJSON(w, http.StatusOK, struct {
		Started bool `json:"started"`
	}{Started: started})
}

// colimaStartCommand starts colima only when it is installed. `colima start`
// is idempotent, so a running VM is harmless.
const colimaStartCommand = "command -v colima >/dev/null 2>&1 && colima start"

// colimaStartTimeout bounds a detached `colima start`; a first boot downloads
// and boots a VM, which takes minutes.
const colimaStartTimeout = 10 * time.Minute

// macStartPlanT says how to bring Docker up on macOS for a docker context.
type macStartPlanT struct {
	openCmd    string // `open -a` launchers, run synchronously; "" for none
	colima     bool   // colima may be started (context is colima or default)
	colimaOnly bool   // colima is the only runtime to try
}

// macStartPlan prefers the runtime the active docker context names. Colima is
// only ever started when the context is colima (or the unnamed default): with
// the context on desktop-linux, orbstack or anything else, starting a colima
// VM would be a surprise.
func macStartPlan(dockerContext string) macStartPlanT {
	c := strings.ToLower(strings.TrimSpace(dockerContext))
	const desktop, orb = "open -a Docker", "open -a OrbStack"
	switch {
	case strings.Contains(c, "colima"):
		return macStartPlanT{colima: true, colimaOnly: true}
	case strings.Contains(c, "orb"):
		return macStartPlanT{openCmd: orb + " || " + desktop}
	case c == "" || c == "default":
		return macStartPlanT{openCmd: desktop + " || " + orb, colima: true}
	}
	return macStartPlanT{openCmd: desktop + " || " + orb}
}

// startColimaDetached launches `colima start` outside the HTTP request: its
// context is not the request's (a client disconnect must not kill a VM
// mid-boot), the local executor runs it in its own process group, and the
// outcome is logged. The handler returns as soon as this is issued.
func (s *Server) startColimaDetached() {
	e := s.newLocalExecutor()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), colimaStartTimeout)
		defer cancel()
		res, err := e.Run(ctx, colimaStartCommand, nil)
		switch {
		case err != nil:
			log.Printf("jumpgate: colima start: %v", err)
		case res.ExitCode != 0:
			log.Printf("jumpgate: colima start exited %d: %s", res.ExitCode, strings.TrimSpace(res.Stderr))
		default:
			log.Printf("jumpgate: colima start finished")
		}
	}()
}
