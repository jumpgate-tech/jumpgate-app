// Package argvfake is a test double for the local executor on a machine with
// no POSIX shell: a Windows controller. Run fails the way the real one does
// there and records the attempt, so a test can assert a whole plan never
// needed a shell. RunArgv answers from scripts, files live in memory, the
// LocalHost facts are fixed, and DialContext reaches only addresses a test
// routed. It is a non-test package so the ops, setup and server tests can
// share it; nothing outside tests imports it.
package argvfake

import (
	"context"
	"fmt"
	"io/fs"
	"net"
	"os"
	"sort"
	"strings"
	"sync"
	"syscall"

	"github.com/valve-tech/jumpgate/internal/executor"
)

type Fake struct {
	GOOS, Home, Arch string
	Files            map[string][]byte

	mu      sync.Mutex
	scripts map[string]executor.Result
	routes  map[string]string
	argvs   [][]string
	shell   []string
}

// New is a Windows controller: GOOS "windows", home C:\Users\dev, amd64.
func New() *Fake {
	return &Fake{GOOS: "windows", Home: `C:\Users\dev`, Arch: "amd64",
		Files: map[string][]byte{}, scripts: map[string]executor.Result{}, routes: map[string]string{}}
}

// Script answers every argv whose space-joined form starts with prefix; the
// longest matching prefix wins. Unscripted argvs succeed with no output.
func (f *Fake) Script(prefix string, res executor.Result) *Fake {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.scripts[prefix] = res
	return f
}

// Route makes DialContext to addr connect to "to" (an httptest listener)
// instead. Every other address refuses, as a free port does.
func (f *Fake) Route(addr, to string) *Fake {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.routes[addr] = to
	return f
}

func (f *Fake) Argvs() [][]string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([][]string(nil), f.argvs...)
}

func (f *Fake) ShellCalls() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.shell...)
}

func (f *Fake) shellErr() error {
	return fmt.Errorf("%w, which %s does not provide", executor.ErrNoPOSIXShell, f.GOOS)
}

func (f *Fake) Run(_ context.Context, cmd string, _ *executor.RunOpts) (executor.Result, error) {
	f.mu.Lock()
	f.shell = append(f.shell, cmd)
	f.mu.Unlock()
	return executor.Result{}, f.shellErr()
}

func (f *Fake) ShellError() error { return f.shellErr() }

func (f *Fake) RunArgv(_ context.Context, argv []string, opts *executor.RunOpts) (executor.Result, error) {
	joined := strings.Join(argv, " ")
	f.mu.Lock()
	f.argvs = append(f.argvs, append([]string(nil), argv...))
	keys := make([]string, 0, len(f.scripts))
	for k := range f.scripts {
		if strings.HasPrefix(joined, k) {
			keys = append(keys, k)
		}
	}
	sort.Slice(keys, func(i, j int) bool { return len(keys[i]) > len(keys[j]) })
	res := executor.Result{}
	if len(keys) > 0 {
		res = f.scripts[keys[0]]
	}
	f.mu.Unlock()
	if opts != nil && opts.Stream != nil {
		for _, line := range strings.Split(strings.TrimRight(res.Stdout, "\n"), "\n") {
			if line != "" {
				opts.Stream(line)
			}
		}
	}
	return res, nil
}

func (f *Fake) WriteFile(_ context.Context, path string, content []byte, _ fs.FileMode) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Files[path] = append([]byte(nil), content...)
	return nil
}

func (f *Fake) ReadFile(_ context.Context, path string) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	b, ok := f.Files[path]
	if !ok {
		return nil, &fs.PathError{Op: "open", Path: path, Err: os.ErrNotExist}
	}
	return b, nil
}

func (f *Fake) Close() error                      { return nil }
func (f *Fake) HostGOOS() string                  { return f.GOOS }
func (f *Fake) HomeDir() (string, error)          { return f.Home, nil }
func (f *Fake) NativeArch(context.Context) string { return f.Arch }

func (f *Fake) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	f.mu.Lock()
	to, ok := f.routes[addr]
	f.mu.Unlock()
	if !ok {
		return nil, &net.OpError{Op: "dial", Net: network, Err: syscall.ECONNREFUSED}
	}
	var d net.Dialer
	return d.DialContext(ctx, network, to)
}

var (
	_ executor.Executor   = (*Fake)(nil)
	_ executor.ArgvRunner = (*Fake)(nil)
	_ executor.LocalHost  = (*Fake)(nil)
)
