package server

import (
	"context"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/executor"
)

// gateExec succeeds at everything, honours its context like a real executor,
// and once armed holds `systemctl stop` until released, so a test can act
// while a clear is half done.
type gateExec struct {
	autoSucceedExecutor
	armed   atomic.Bool
	once    sync.Once
	started chan struct{}
	release chan struct{}
}

func newGateExec() *gateExec {
	return &gateExec{started: make(chan struct{}), release: make(chan struct{})}
}

func (g *gateExec) Run(ctx context.Context, cmd string, o *executor.RunOpts) (executor.Result, error) {
	if g.armed.Load() && strings.Contains(cmd, "systemctl stop") {
		g.once.Do(func() { close(g.started) })
		<-g.release
	}
	if err := ctx.Err(); err != nil {
		return executor.Result{}, err
	}
	return g.autoSucceedExecutor.Run(ctx, cmd, o)
}

func (g *gateExec) ran(sub string) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	for _, c := range g.calls {
		if strings.Contains(c, sub) {
			return true
		}
	}
	return false
}

// A clear stops the unit, deletes data, and starts it again. A browser that
// disconnects between those steps must not leave the node stopped with half
// its data gone.
func TestClearFinishesAfterTheClientGoesAway(t *testing.T) {
	g := newGateExec()
	a := newAPITestServerWithExecutor(t, func(config.Target) (executor.Executor, error) { return g, nil })
	addAndWireLocalTarget(t, a)
	g.armed.Store(true)

	ctx, cancel := context.WithCancel(context.Background())
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, a.ts.URL+"/api/targets/local/services/exec/clear", strings.NewReader(`{"Confirm":"exec"}`))
	req.Header.Set("Authorization", "Bearer "+a.token)
	go func() {
		if res, err := http.DefaultClient.Do(req); err == nil {
			res.Body.Close()
		}
	}()

	<-g.started
	cancel() // the tab closes mid-clear
	time.Sleep(50 * time.Millisecond)
	close(g.release)

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if g.ran("systemctl start") {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("the clear was abandoned when the client went away: the unit was never started again")
}

// Ctrl-C during a clear waits for it to finish rather than killing it half
// done. A second Ctrl-C still forces the exit (signal handling in main).
func TestShutdownWaitsForAnInFlightClear(t *testing.T) {
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	addr := ln.Addr().String()
	ln.Close()

	g := newGateExec()
	a := newAPITestServerCfg(t, func(config.Target) (executor.Executor, error) { return g, nil },
		func(c *Config) { c.Bind = addr })
	addAndWireLocalTarget(t, a)
	g.armed.Store(true)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { _ = a.srv.ListenAndServe(ctx); close(done) }()
	for i := 0; i < 100; i++ {
		if c, err := net.Dial("tcp", addr); err == nil {
			c.Close()
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	req, _ := http.NewRequest(http.MethodPost, "http://"+addr+"/api/targets/local/services/exec/clear", strings.NewReader(`{"Confirm":"exec"}`))
	req.Header.Set("Authorization", "Bearer "+a.token)
	go func() {
		if res, err := http.DefaultClient.Do(req); err == nil {
			res.Body.Close()
		}
	}()
	<-g.started
	cancel() // Ctrl-C

	select {
	case <-done:
		t.Fatal("the server exited while a clear was half done")
	case <-time.After(shutdownGrace + time.Second):
	}
	close(g.release)
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("the server did not exit after the clear finished")
	}
	if !g.ran("systemctl start") {
		t.Fatal("the clear did not run to completion")
	}
}
