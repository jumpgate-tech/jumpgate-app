package server

import (
	"context"
	"fmt"
	"io/fs"
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

func lease(ex executor.Executor) *leasedExec { return &leasedExec{ex: ex} }

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
