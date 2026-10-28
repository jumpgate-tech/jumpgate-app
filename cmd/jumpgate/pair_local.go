package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"

	"github.com/valve-tech/jumpgate/internal/agentbin"
	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/apiclient"
	"github.com/valve-tech/jumpgate/internal/bootstrap"
	"github.com/valve-tech/jumpgate/internal/catalog"
	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/daemon"
	"github.com/valve-tech/jumpgate/internal/eip712"
	"github.com/valve-tech/jumpgate/internal/executor"
)

// Seams for tests; production never reassigns them.
var (
	hostGOOS           = runtime.GOOS
	geteuid            = os.Geteuid
	ensureRunning      = daemon.EnsureRunning
	sudoNonInteractive = func(ctx context.Context) error { return exec.CommandContext(ctx, "sudo", "-n", "true").Run() }
	sudoPrompt         = func(ctx context.Context) error {
		c := exec.CommandContext(ctx, "sudo", "-v")
		c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr
		return c.Run()
	}
	runBootstrap     = bootstrap.Run
	newLocalExecutor = executor.NewLocal
	// serverIsSystem reports whether c is the jumpgate service.
	serverIsSystem = func(c *apiclient.Client) bool { return c.Info().System }
)

// pairingFacts is what a foreground pairing needs to know: which target, the
// controller that enrolls, the uid whose local connections the agent admits,
// and the node configuration to write.
type pairingFacts struct {
	TargetID   string
	Controller eip712.Address
	LocalUID   int
	Wire       *catalog.WireConfig
	Scope      string
}

// factsFromConfig is a per-user server's facts: read from this user's own
// config.json, and the local uid is this user's.
func factsFromConfig(t config.Target) (pairingFacts, int) {
	c, err := config.Load()
	if err != nil {
		return pairingFacts{}, failed("load config: %v", err)
	}
	if c.Controller == nil {
		return pairingFacts{}, reportServerError(os.Stderr, "pair", apiError{Message: "this controller has no signing key yet", Code: api.CodeNoControllerKey})
	}
	controller, err := eip712.ParseAddress(c.Controller.Address)
	if err != nil {
		return pairingFacts{}, failed("controller address in config.json: %v", err)
	}
	return pairingFacts{TargetID: t.ID, Controller: controller, LocalUID: os.Getuid(), Wire: t.Wire, Scope: c.Controller.Scope}, 0
}

// factsFromServer is the jumpgate service's facts: its config is not this
// user's, so the server says them, and the agent admits the service's uid.
func factsFromServer(ctx context.Context, c *apiclient.Client, id string) (pairingFacts, int) {
	info, err := c.PairingInfo(ctx, id)
	var e *api.Error
	if errors.As(err, &e) {
		return pairingFacts{}, reportServerErrorFrom(os.Stderr, "pairing info", c.Info(), *e)
	}
	if err != nil {
		return pairingFacts{}, failed("pairing info: %v", err)
	}
	controller, err := eip712.ParseAddress(info.Controller)
	if err != nil {
		return pairingFacts{}, failed("controller address from the server: %v", err)
	}
	return pairingFacts{TargetID: id, Controller: controller, LocalUID: info.ServiceUID, Wire: info.Wire, Scope: info.Scope}, 0
}

// pairLocalForeground runs the privileged pairing steps for this machine in
// this process, on the operator's terminal, so sudo can ask for a password
// (spec D5). The detached server never could: it has no terminal. sudo's
// cached credential covers the commands below because the local executor
// keeps this terminal session (Setpgid, never Setsid). As root it runs them
// directly: a root shell on stock Debian may have no sudo at all. It returns
// the new agent's address for the server to verify and record; the
// controller key itself never leaves the server.
func pairLocalForeground(ctx context.Context, out io.Writer, f pairingFacts) (string, int) {
	// hosts add saved the target before calling here, so a failure from now
	// on leaves it saved but unpaired; say so, and how to finish.
	unpaired := func(format string, a ...any) (string, int) {
		return "", failed(format+"\nTarget %s is saved but not paired yet; run `jumpgate hosts add %s --local` again to finish.", append(a, f.TargetID, f.TargetID)...)
	}
	ex := newLocalExecutor()
	if geteuid() != 0 {
		if err := sudoNonInteractive(ctx); err != nil {
			fmt.Fprintln(out, "pairing this machine runs commands as root; sudo may ask for your password")
			if err := sudoPrompt(ctx); err != nil {
				return unpaired("sudo: %v. Pairing this machine needs root: ask an administrator for sudo rights, or run jumpgate as root", err)
			}
		}
		ex = executor.Sudo(ex)
	}
	label, _ := os.Hostname()
	if label == "" {
		label = "controller"
	}
	addr, err := runBootstrap(ctx, bootstrap.Options{
		Exec: ex, Local: true, LocalUID: f.LocalUID,
		AgentBinary: agentbin.Reporting(func(line string) { fmt.Fprintf(out, "[upload] %s\n", line) }),
		Controller:  f.Controller, ControllerLabel: label, Wire: f.Wire, Scope: localScope(f.Scope),
		Event: func(step, line string) { fmt.Fprintf(out, "[%s] %s\n", step, line) },
	})
	if err != nil {
		return unpaired("pairing failed: %v", err)
	}
	return addr.Hex(), 0
}

// localScope is the enroll scope for a controller of scope: a self-only
// controller is enrolled on its own box for local use only (D11).
func localScope(scope string) string {
	if scope == "self" {
		return "local"
	}
	return ""
}
