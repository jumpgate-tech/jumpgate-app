package server

import (
	"context"
	"errors"
	"io/fs"
	"sync"
	"sync/atomic"
	"testing"
	"testing/fstest"
	"time"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/executor"
	"github.com/valve-tech/jumpgate/internal/executor/argvfake"
)

// execFactory hands out executors and counts how many are open, so a test
// can see an executor that is never closed. A Run on any of them blocks
// while block is set, until unblock is closed.
type execFactory struct {
	opened, closed atomic.Int32
	mu             sync.Mutex
	block          bool
	unblock        chan struct{}
	entered        chan struct{}
}

type factoryExec struct {
	f      *execFactory
	closed atomic.Bool
}

func (f *execFactory) newExecutor(config.Target) (executor.Executor, error) {
	f.opened.Add(1)
	return &factoryExec{f: f}, nil
}

func (f *execFactory) open() int32 { return f.opened.Load() - f.closed.Load() }

func (e *factoryExec) Run(ctx context.Context, cmd string, _ *executor.RunOpts) (executor.Result, error) {
	if e.closed.Load() {
		return executor.Result{}, errors.New("use of a closed executor")
	}
	e.f.mu.Lock()
	block, unblock, entered := e.f.block, e.f.unblock, e.f.entered
	e.f.mu.Unlock()
	if block {
		entered <- struct{}{}
		<-unblock
	}
	return executor.Result{}, nil
}
func (e *factoryExec) WriteFile(context.Context, string, []byte, fs.FileMode) error { return nil }
func (e *factoryExec) ReadFile(context.Context, string) ([]byte, error)             { return nil, nil }
func (e *factoryExec) Close() error {
	if e.closed.CompareAndSwap(false, true) {
		e.f.closed.Add(1)
	}
	return nil
}

// leaseServer is a server whose executors come from f, for target a. The
// auto-diagnostics slot is held so no background diagnostics run opens an
// executor of its own.
func leaseServer(t *testing.T) (*Server, *execFactory, config.Target) {
	t.Helper()
	fleetHome(t)
	f := &execFactory{}
	s := New(Config{Token: NewSessionToken(), UI: fstest.MapFS{}, NewExecutor: f.newExecutor})
	tg := config.Target{ID: "a", Mode: "ssh", SSH: &executor.SSHConfig{Host: "a", User: "u"}, Wire: wired}
	s.reg.get("a").tryBeginDiag(time.Now(), false)
	t.Cleanup(func() { s.reg.remove("a") })
	return s, f, tg
}

func eventually(t *testing.T, what string, ok func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for !ok() {
		if time.Now().After(deadline) {
			t.Fatal(what)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// An evicted executor that a monitor and a log watcher were using is closed,
// and removing the target leaves nothing open.
func TestEvictThenRemoveClosesTheEvictedExecutor(t *testing.T) {
	s, f, tg := leaseServer(t)
	if _, _, err := s.getMonitor(tg, ""); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.getWatcher(tg); err != nil {
		t.Fatal(err)
	}
	ex, _ := s.getExecutor(tg)
	s.reg.evictExecutor("a", ex)
	s.reg.remove("a")
	eventually(t, "the evicted executor was never closed", func() bool { return f.open() == 0 })
}

// Evicting again and again leaves at most the live executor open.
func TestRepeatedEvictionsLeakNothing(t *testing.T) {
	s, f, tg := leaseServer(t)
	for i := 0; i < 5; i++ {
		if _, _, err := s.getMonitor(tg, ""); err != nil {
			t.Fatal(err)
		}
		if _, _, err := s.getWatcher(tg); err != nil {
			t.Fatal(err)
		}
		ex, _ := s.getExecutor(tg)
		s.reg.evictExecutor("a", ex)
		eventually(t, "an evicted executor stayed open", func() bool { return f.open() == 0 })
	}
	if f.opened.Load() != 5 {
		t.Fatalf("opened %d executors, want one per round", f.opened.Load())
	}
}

// After an eviction the monitor and the log watcher are rebuilt on a fresh
// executor at their next use, instead of polling a dead connection.
func TestEvictionRestartsTheMonitorAndWatcherOnAFreshExecutor(t *testing.T) {
	s, f, tg := leaseServer(t)
	mon1, _, _ := s.getMonitor(tg, "")
	w1, _, _ := s.getWatcher(tg)
	ex1, _ := s.getExecutor(tg)
	s.reg.evictExecutor("a", ex1)
	mon2, _, _ := s.getMonitor(tg, "")
	w2, _, _ := s.getWatcher(tg)
	ex2, _ := s.getExecutor(tg)
	if mon2 == mon1 || w2 == w1 || ex2 == ex1 || f.opened.Load() != 2 {
		t.Fatalf("monitor fresh %v, watcher fresh %v, executor fresh %v, opened %d", mon2 != mon1, w2 != w1, ex2 != ex1, f.opened.Load())
	}
	e := s.reg.get("a")
	e.mu.Lock()
	monExec, watchExec := e.monExec, e.watchExec
	e.mu.Unlock()
	if monExec != ex2 || watchExec != ex2 {
		t.Fatal("the rebuilt monitor or watcher is not on the fresh executor")
	}
}

// An executor is never closed under an operation still using it: eviction
// closes it when that operation ends, and a later call on the stale
// reference is an unreachable error, not a use of a closed connection.
func TestEvictionWaitsForTheOperationInFlight(t *testing.T) {
	s, f, tg := leaseServer(t)
	ex, _ := s.getExecutor(tg)
	other := &factoryExec{f: &execFactory{}}
	s.reg.evictExecutor("a", other) // not the cached one: nothing happens
	if again, _ := s.getExecutor(tg); again != ex {
		t.Fatal("evicting a different executor dropped the cached one")
	}
	f.mu.Lock()
	f.block, f.unblock, f.entered = true, make(chan struct{}), make(chan struct{}, 1)
	f.mu.Unlock()
	runErr := make(chan error, 1)
	go func() {
		_, err := ex.Run(context.Background(), "sleep", nil)
		runErr <- err
	}()
	<-f.entered
	s.reg.evictExecutor("a", ex)
	if f.closed.Load() != 0 {
		t.Fatal("the executor was closed while a command was running on it")
	}
	f.mu.Lock()
	f.block = false
	f.mu.Unlock()
	close(f.unblock)
	if err := <-runErr; err != nil {
		t.Fatalf("the running command failed: %v", err)
	}
	eventually(t, "the executor was not closed after its command ended", func() bool { return f.closed.Load() == 1 })
	_, err := ex.Run(context.Background(), "true", nil)
	if err == nil {
		t.Fatal("a call on an evicted executor succeeded")
	}
	if _, e := apiErrorFor(err); e.Code != api.CodeUnreachable {
		t.Fatalf("a call on an evicted executor: %v (%s), want unreachable", err, e.Code)
	}
}

// The lease keeps exactly the optional interfaces of what it wraps: callers
// choose argv or a shell, and this process or a connection, by type assertion.
func TestLeaseKeepsTheExecutorsOptionalInterfaces(t *testing.T) {
	fake := argvfake.New()
	l := lease(fake)
	if _, ok := l.(executor.ArgvRunner); !ok {
		t.Error("a leased ArgvRunner lost RunArgv")
	}
	h, ok := l.(executor.LocalHost)
	if !ok || h.HostGOOS() != "windows" {
		t.Errorf("a leased LocalHost lost its methods (ok=%v)", ok)
	}
	if err := executor.RequireShell(l); err == nil {
		t.Error("a leased shell-less executor passed RequireShell")
	}
	if _, err := executor.Exec(context.Background(), l, executor.Command{Argv: []string{"docker", "--version"}, Shell: "command -v docker"}, nil); err != nil {
		t.Errorf("argv through the lease: %v", err)
	}
	if got := fake.Argvs(); len(got) != 1 || len(got[0]) != 2 || got[0][0] != "docker" {
		t.Errorf("Exec through the lease ran argv %v, want exactly the docker argv", got)
	}
	if calls := fake.ShellCalls(); len(calls) != 0 {
		t.Errorf("Exec through the lease used the shell: %v", calls)
	}

	plain := lease(&autoSucceedExecutor{})
	if _, ok := plain.(executor.ArgvRunner); ok {
		t.Error("a leased shell executor claims RunArgv")
	}
	if _, ok := plain.(executor.LocalHost); ok {
		t.Error("a leased shell executor claims to be this machine")
	}
	if err := executor.RequireShell(plain); err != nil {
		t.Errorf("a leased shell executor fails RequireShell: %v", err)
	}
}

// A retired lease refuses the argv path as it refuses the shell one: a closed
// or evicted connection is never used by RunArgv, and Exec on it does not
// fall back to a shell either.
func TestRetiredLeaseRefusesRunArgv(t *testing.T) {
	fake := argvfake.New()
	l := lease(fake)
	argv := l.(executor.ArgvRunner)
	if _, err := argv.RunArgv(context.Background(), []string{"docker", "ps"}, nil); err != nil {
		t.Fatalf("RunArgv before retirement: %v", err)
	}
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	before := len(fake.Argvs())
	_, err := argv.RunArgv(context.Background(), []string{"docker", "ps"}, nil)
	if err == nil {
		t.Fatal("RunArgv on a closed lease succeeded")
	}
	if _, e := apiErrorFor(err); e.Code != api.CodeUnreachable {
		t.Fatalf("RunArgv on a closed lease: %v (%s), want unreachable", err, e.Code)
	}
	if _, err := executor.Exec(context.Background(), l, executor.Command{Argv: []string{"docker", "ps"}, Shell: "docker ps"}, nil); err == nil {
		t.Error("Exec on a closed lease succeeded")
	}
	if len(fake.Argvs()) != before || len(fake.ShellCalls()) != 0 {
		t.Errorf("a retired lease reached the executor: argv %v, shell %v", fake.Argvs(), fake.ShellCalls())
	}
}

// Eviction retires the cached executor the same way: the stale lease's
// RunArgv is refused.
func TestEvictedLeaseRefusesRunArgv(t *testing.T) {
	fake := argvfake.New()
	fleetHome(t)
	s := New(Config{Token: NewSessionToken(), UI: fstest.MapFS{},
		NewExecutor: func(config.Target) (executor.Executor, error) { return fake, nil }})
	tg := config.Target{ID: "a", Mode: "ssh", SSH: &executor.SSHConfig{Host: "a", User: "u"}, Wire: wired}
	s.reg.get("a").tryBeginDiag(time.Now(), false)
	t.Cleanup(func() { s.reg.remove("a") })
	ex, err := s.getExecutor(tg)
	if err != nil {
		t.Fatal(err)
	}
	argv, ok := ex.(executor.ArgvRunner)
	if !ok {
		t.Fatal("the registry's executor lost RunArgv")
	}
	s.reg.evictExecutor("a", ex)
	if _, err := argv.RunArgv(context.Background(), []string{"docker", "ps"}, nil); err == nil {
		t.Fatal("RunArgv on an evicted executor succeeded")
	}
	if len(fake.Argvs()) != 0 {
		t.Errorf("an evicted executor ran %v", fake.Argvs())
	}
}
