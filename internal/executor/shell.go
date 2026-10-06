package executor

// RequireShell returns nil when e can run Run's `sh -c` commands, and an
// error wrapping ErrNoPOSIXShell when it cannot. Only a local executor on a
// machine with no POSIX shell (Windows) cannot: it still runs Docker through
// RunArgv (spec D30), so it is constructed, and the routes that need a shell
// ask this first.
func RequireShell(e Executor) error {
	if s, ok := e.(interface{ ShellError() error }); ok {
		return s.ShellError()
	}
	return nil
}
