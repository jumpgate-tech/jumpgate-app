//go:build unix

package main

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"
)

// signalChildEnv marks the re-executed test binary as the child that takes
// the signals, so the parent can watch what they do to a whole process.
const signalChildEnv = "VALVE_SIGNAL_CHILD"

// The first Ctrl-C asks for a graceful shutdown. If that shutdown stalls, a
// second Ctrl-C has to kill the process, which needs the signal handler
// unregistered once the first signal has arrived. Otherwise every later
// Ctrl-C is absorbed by a context that is already canceled.
func TestShutdownContext_SecondInterruptKillsTheProcess(t *testing.T) {
	if os.Getenv(signalChildEnv) == "1" {
		signalChild()
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=^TestShutdownContext_SecondInterruptKillsTheProcess$")
	cmd.Env = append(os.Environ(), signalChildEnv+"=1")
	done := make(chan error, 1)
	if err := cmd.Start(); err != nil {
		t.Fatalf("start child: %v", err)
	}
	go func() { done <- cmd.Wait() }()

	var err error
	select {
	case err = <-done:
	case <-time.After(30 * time.Second):
		cmd.Process.Kill()
		t.Fatal("the child never exited")
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("the child exited cleanly (%v): the second interrupt was swallowed", err)
	}
	ws, ok := exitErr.Sys().(syscall.WaitStatus)
	if !ok || !ws.Signaled() || ws.Signal() != syscall.SIGINT {
		t.Fatalf("the child did not die of the second interrupt: %v", err)
	}
}

// signalChild sends itself one interrupt, waits for the shutdown context to
// report it, then sends a second. The second should kill it outright; if it
// is swallowed instead, the child gives up waiting and exits cleanly, which
// the parent reports as the bug.
func signalChild() {
	ctx, stop := shutdownContext(context.Background())
	defer stop()

	syscall.Kill(os.Getpid(), syscall.SIGINT)
	select {
	case <-ctx.Done():
	case <-time.After(10 * time.Second):
		os.Exit(3) // the first interrupt was never seen
	}

	syscall.Kill(os.Getpid(), syscall.SIGINT)
	time.Sleep(5 * time.Second)
	os.Exit(0)
}
