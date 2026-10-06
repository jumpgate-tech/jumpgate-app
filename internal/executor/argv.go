package executor

import (
	"context"
	"net"
	"strings"
)

// ArgvRunner is implemented by an executor that can start a program directly
// from an argument vector, with no shell in between. The local executor is
// one, on every OS: that is how a Windows controller runs docker, with no sh
// to hand a command string to (spec D29). The SSH executor is not one,
// because a remote command is a string the far side's shell parses.
type ArgvRunner interface {
	RunArgv(ctx context.Context, argv []string, opts *RunOpts) (Result, error)
}

// LocalHost is implemented by the local executor. Each method answers a
// question a caller would otherwise put to the target's shell (`printf
// "$HOME"`, `uname -m`, `ss`, `curl`), read from this process instead, so the
// answer does not depend on a shell existing.
type LocalHost interface {
	// HostGOOS is runtime.GOOS: whose path rules this target's files follow.
	HostGOOS() string
	// HomeDir is the current user's home directory (os.UserHomeDir).
	HomeDir() (string, error)
	// NativeArch is the machine's CPU as `uname -m` or GOARCH spells it
	// (ops.PlatformForArch reads both), or "" when it cannot be read.
	NativeArch(ctx context.Context) string
	// DialContext connects from this machine, as net.Dialer does.
	DialContext(ctx context.Context, network, addr string) (net.Conn, error)
}

// Command is one program run, in both forms an executor may need.
type Command struct {
	// Argv is the program and its arguments, for an ArgvRunner.
	Argv []string
	// Shell is the `sh -c` form for every other executor. Empty means
	// QuoteArgv(Argv). It is set where an existing probe string must stay
	// byte-identical for SSH targets and the tests that know it.
	Shell string
}

// Exec runs c on e: as argv when e can start programs directly, otherwise
// as a shell string. A caller writes one call, and gets the shell-free path
// on the local machine and the unchanged shell path over SSH.
func Exec(ctx context.Context, e Executor, c Command, opts *RunOpts) (Result, error) {
	if a, ok := e.(ArgvRunner); ok && len(c.Argv) > 0 {
		return a.RunArgv(ctx, c.Argv, opts)
	}
	cmd := c.Shell
	if cmd == "" {
		cmd = QuoteArgv(c.Argv)
	}
	return e.Run(ctx, cmd, opts)
}

// QuoteArgv renders argv as a shell command: the program name as is, then
// every argument single-quoted. It is byte-identical to the string
// ops.DockerRun has always built (an embedded quote is closed, backslashed
// and reopened, unlike ssh.go's double-quoted form), so a shell target sees
// no change.
func QuoteArgv(argv []string) string {
	if len(argv) == 0 {
		return ""
	}
	parts := make([]string, 0, len(argv))
	parts = append(parts, argv[0])
	for _, a := range argv[1:] {
		parts = append(parts, "'"+strings.ReplaceAll(a, "'", `'\''`)+"'")
	}
	return strings.Join(parts, " ")
}
