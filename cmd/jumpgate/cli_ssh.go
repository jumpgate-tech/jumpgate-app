package main

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"runtime"

	"github.com/valve-tech/jumpgate/internal/api"
)

// sshBinary is the system OpenSSH client's name; jumpgate never falls back to
// anything else.
func sshBinary() string {
	if runtime.GOOS == "windows" {
		return "ssh.exe"
	}
	return "ssh"
}

// runInteractive runs argv attached to this terminal and returns its exit
// status. argv[0] must be the system ssh, found on PATH; the server's command
// is exec'd directly, never through a shell. A variable so tests can see the
// command without running ssh.
var runInteractive = func(argv []string) int {
	if len(argv) == 0 || argv[0] != sshBinary() {
		return failed("refusing to run %q: jumpgate ssh only runs the system %s", argv, sshBinary())
	}
	path, err := exec.LookPath(argv[0])
	if err != nil {
		return failed("%s was not found on PATH: install the OpenSSH client and try again", sshBinary())
	}
	cmd := exec.Command(path, argv[1:]...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			return exit.ExitCode()
		}
		return failed("%s: %v", sshBinary(), err)
	}
	return 0
}

// cmdSSH opens a shell on a box with the system ssh, under the host keys a
// person confirmed (the server builds the command).
func cmdSSH(args []string) int {
	if len(args) != 1 {
		return usage("usage: jumpgate ssh HOST")
	}
	ctx := context.Background()
	c, err := connect(ctx)
	if err != nil {
		return failed("%v", err)
	}
	cmd, err := c.SSHCommand(ctx, args[0])
	var e *api.Error
	if errors.As(err, &e) {
		return reportServerError(os.Stderr, args[0], *e)
	}
	if err != nil {
		return failed("%v", err)
	}
	if len(cmd.Argv) == 0 {
		return failed("the server sent an empty ssh command")
	}
	if cmd.Argv[0] != "ssh" && cmd.Argv[0] != "ssh.exe" {
		return failed("the server sent %q as the program; jumpgate ssh only runs the system ssh", cmd.Argv[0])
	}
	// The server names the program; this machine decides which binary that is.
	return runInteractive(append([]string{sshBinary()}, cmd.Argv[1:]...))
}
