package server

import (
	"context"
	"fmt"
	"io/fs"
	"net"
	"sync"

	"github.com/valve-tech/jumpgate/internal/executor"
)

// leasedExec is the registry's cached executor. It counts the calls in
// flight, so retiring it (eviction, target removal) closes the connection
// only once the last call has returned: nothing closes an executor under a
// command still running on it. A call that starts after retirement fails
// as unreachable instead of using a connection that is gone or going.
type leasedExec struct {
	ex executor.Executor

	mu      sync.Mutex
	inUse   int
	retired bool
	closed  bool
}

// lease wraps ex. The wrapper keeps ex's optional interfaces, because
// callers decide by type assertion how to talk to an executor: an
// executor.ArgvRunner runs Docker as argv with no shell, an
// executor.LocalHost answers paths and dials from this process, and
// ShellError (executor.RequireShell) refuses shell routes on a machine with
// no POSIX shell. A wrapper that hid them would send a shell command to
// Windows, or treat this computer like an SSH box. Each is present on the
// wrapper exactly when ex has it.
func lease(ex executor.Executor) executor.Executor {
	l := &leasedExec{ex: ex}
	_, argv := ex.(executor.ArgvRunner)
	_, host := ex.(executor.LocalHost)
	switch {
	case argv && host:
		return leasedArgvHost{leasedHost{l}}
	case argv:
		return leasedArgv{l}
	case host:
		return leasedHost{l}
	}
	return l
}

// ShellError is ex's, or nil when ex has none: what RequireShell would have
// answered for ex itself.
func (l *leasedExec) ShellError() error { return executor.RequireShell(l.ex) }

func (l *leasedExec) runArgv(ctx context.Context, argv []string, opts *executor.RunOpts) (executor.Result, error) {
	if err := l.begin(); err != nil {
		return executor.Result{}, err
	}
	defer l.end()
	return l.ex.(executor.ArgvRunner).RunArgv(ctx, argv, opts)
}

func (l *leasedExec) host() executor.LocalHost { return l.ex.(executor.LocalHost) }

// leasedArgv is a lease of an executor.ArgvRunner.
type leasedArgv struct{ *leasedExec }

func (l leasedArgv) RunArgv(ctx context.Context, argv []string, opts *executor.RunOpts) (executor.Result, error) {
	return l.runArgv(ctx, argv, opts)
}

// leasedHost is a lease of an executor.LocalHost. Its answers come from this
// process, not the connection being leased, so they are passed straight on.
type leasedHost struct{ *leasedExec }

func (l leasedHost) HostGOOS() string                      { return l.host().HostGOOS() }
func (l leasedHost) HomeDir() (string, error)              { return l.host().HomeDir() }
func (l leasedHost) NativeArch(ctx context.Context) string { return l.host().NativeArch(ctx) }
func (l leasedHost) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	return l.host().DialContext(ctx, network, addr)
}

// leasedArgvHost is a lease of an executor that is both (the local one).
type leasedArgvHost struct{ leasedHost }

func (l leasedArgvHost) RunArgv(ctx context.Context, argv []string, opts *executor.RunOpts) (executor.Result, error) {
	return l.runArgv(ctx, argv, opts)
}

func (l *leasedExec) begin() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.retired {
		return &dialError{fmt.Errorf("%w: this connection was retired; try again", errConnectionLost)}
	}
	l.inUse++
	return nil
}

func (l *leasedExec) end() {
	l.mu.Lock()
	l.inUse--
	closeNow := l.retired && l.inUse == 0 && !l.closed
	if closeNow {
		l.closed = true
	}
	l.mu.Unlock()
	if closeNow {
		_ = l.ex.Close()
	}
}

func (l *leasedExec) Run(ctx context.Context, cmd string, opts *executor.RunOpts) (executor.Result, error) {
	if err := l.begin(); err != nil {
		return executor.Result{}, err
	}
	defer l.end()
	return l.ex.Run(ctx, cmd, opts)
}

func (l *leasedExec) WriteFile(ctx context.Context, path string, content []byte, mode fs.FileMode) error {
	if err := l.begin(); err != nil {
		return err
	}
	defer l.end()
	return l.ex.WriteFile(ctx, path, content, mode)
}

func (l *leasedExec) ReadFile(ctx context.Context, path string) ([]byte, error) {
	if err := l.begin(); err != nil {
		return nil, err
	}
	defer l.end()
	return l.ex.ReadFile(ctx, path)
}

// Close retires the executor: it closes now when idle, otherwise when the
// last call in flight returns. It is idempotent.
func (l *leasedExec) Close() error {
	l.mu.Lock()
	l.retired = true
	closeNow := l.inUse == 0 && !l.closed
	if closeNow {
		l.closed = true
	}
	l.mu.Unlock()
	if closeNow {
		return l.ex.Close()
	}
	return nil
}
