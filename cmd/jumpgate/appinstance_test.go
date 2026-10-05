package main

import (
	"context"
	"errors"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/valve-tech/jumpgate/internal/daemon"
	"github.com/valve-tech/jumpgate/internal/server"
)

// shortHome isolates HOME under /tmp: the server socket lives in
// ~/.jumpgate/run and unix socket paths must stay short.
func shortHome(t *testing.T) {
	t.Helper()
	home, err := os.MkdirTemp("/tmp", "jgh")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(home) })
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
}

func freeAddr(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	return ln.Addr().String()
}

// R25: the tray/web entry point takes the same single-instance lock as serve.
func TestClaimAppInstanceTakesTheLock(t *testing.T) {
	shortHome(t)
	holder, running, err := claimAppInstance(context.Background(), 0)
	if err != nil || holder == nil || running != nil {
		t.Fatalf("claimAppInstance = %v, %v, %v; want the lock", holder, running, err)
	}
	defer holder.Release()
	if _, err := daemon.Acquire(); !errors.Is(err, daemon.ErrAlreadyRunning) {
		t.Fatalf("a second server could take the lock: %v", err)
	}
}

// R25: with `jumpgate serve` already up, the app finds it through server.json
// instead of starting a second server on the same port. serveAndPublish is
// what both entry points use to come up and advertise themselves.
func TestClaimAppInstanceFindsTheRunningServer(t *testing.T) {
	shortHome(t)
	h, err := daemon.Acquire()
	if err != nil {
		t.Fatal(err)
	}
	defer h.Release()
	bind, token := freeAddr(t), server.NewSessionToken()
	ctx, stop := context.WithCancel(context.Background())
	s := server.New(server.Config{Bind: bind, Token: token, Shutdown: stop})
	done := make(chan error, 1)
	go func() { done <- serveAndPublish(ctx, stop, s, h, bind, token, nil) }()
	defer func() { stop(); <-done }()

	_, running, err := claimAppInstance(context.Background(), 5*time.Second)
	if err != nil || running == nil {
		t.Fatalf("claimAppInstance = %v, %v; want the running server", running, err)
	}
	if running.HTTPAddr != bind || running.Token != token || running.PID != os.Getpid() {
		t.Fatalf("running = %+v, want addr %s and its token", running, bind)
	}
	if got := appURL(*running); got != "http://"+bind+"/?token="+token {
		t.Errorf("appURL = %q", got)
	}
}

// A lock holder that never answers is not a server to open; the app says so
// instead of waiting forever or starting a second server.
func TestClaimAppInstanceGivesUpOnASilentHolder(t *testing.T) {
	shortHome(t)
	h, err := daemon.Acquire()
	if err != nil {
		t.Fatal(err)
	}
	defer h.Release()
	_, _, err = claimAppInstance(context.Background(), 300*time.Millisecond)
	if err == nil || !strings.Contains(err.Error(), "jumpgate stop") {
		t.Fatalf("err = %v, want a hint naming jumpgate stop", err)
	}
}
