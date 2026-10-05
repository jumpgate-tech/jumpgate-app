package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"

	"github.com/valve-tech/jumpgate/internal/agentbin"
	"github.com/valve-tech/jumpgate/internal/bootstrap"
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
)

// pairBody is the pair endpoint's request.
type pairBody struct {
	Sudo      bool   `json:"sudo"`
	Installed string `json:"installed,omitempty"`
}

// pairLocalForeground runs the privileged pairing steps for this machine in
// this process, on the operator's terminal, so sudo can ask for a password
// (spec D5). The detached server never could: it has no terminal. sudo's
// cached credential covers the commands below because the local executor
// keeps this terminal session (Setpgid, never Setsid). t is the target as
// saved, so its node configuration reaches the box just as the server's
// pairing would write it. It returns the new agent's address for the server
// to verify and record; the controller key itself never leaves the server.
func pairLocalForeground(ctx context.Context, out io.Writer, t config.Target) (string, int) {
	c, err := config.Load()
	if err != nil {
		return "", failed("load config: %v", err)
	}
	if c.Controller == nil {
		return "", reportServerError(os.Stderr, "pair", apiError{Error: "this controller has no signing key yet", Code: "no_controller_key"})
	}
	controller, err := eip712.ParseAddress(c.Controller.Address)
	if err != nil {
		return "", failed("controller address in config.json: %v", err)
	}
	// hosts add saved the target before calling here, so a failure from now
	// on leaves it saved but unpaired; say so, and how to finish.
	unpaired := func(format string, a ...any) (string, int) {
		return "", failed(format+"\nTarget %s is saved but not paired yet; run `jumpgate hosts add %s --local` again to finish.", append(a, t.ID, t.ID)...)
	}
	if err := sudoNonInteractive(ctx); err != nil {
		fmt.Fprintln(out, "pairing this machine runs commands as root; sudo may ask for your password")
		if err := sudoPrompt(ctx); err != nil {
			return unpaired("sudo: %v. Pairing this machine needs root: ask an administrator for sudo rights, or run jumpgate as root", err)
		}
	}
	label, _ := os.Hostname()
	if label == "" {
		label = "controller"
	}
	addr, err := runBootstrap(ctx, bootstrap.Options{
		Exec: executor.Sudo(newLocalExecutor()), Local: true, LocalUID: os.Getuid(),
		AgentBinary: agentbin.Reporting(func(line string) { fmt.Fprintf(out, "[upload] %s\n", line) }),
		Controller:  controller, ControllerLabel: label, Wire: t.Wire,
		Event: func(step, line string) { fmt.Fprintf(out, "[%s] %s\n", step, line) },
	})
	if err != nil {
		return unpaired("pairing failed: %v", err)
	}
	return addr.Hex(), 0
}
