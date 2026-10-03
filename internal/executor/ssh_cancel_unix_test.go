//go:build unix

package executor

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	gliderssh "github.com/gliderlabs/ssh"
	gossh "golang.org/x/crypto/ssh"
)

// startDetachedSSHD behaves like a real sshd in the one way that matters here:
// closing a session does not kill the command it started.
func startDetachedSSHD(t *testing.T) (testSSHD, string) {
	t.Helper()
	return startDetachedSSHDWith(t, 0)
}

// startDetachedSSHDWith is startDetachedSSHD with each session's first stdout
// write held back by stdoutDelay, as a slow link would hold back the marker.
func startDetachedSSHDWith(t *testing.T, stdoutDelay time.Duration) (testSSHD, string) {
	t.Helper()
	d, keyPath := startTestSSHDWith(t, func(s gliderssh.Session) {
		c := exec.Command("sh", "-c", s.RawCommand())
		c.Stdout, c.Stderr, c.Stdin = &slowStartWriter{w: s, delay: stdoutDelay}, s.Stderr(), s
		_ = c.Start()
		done := make(chan error, 1)
		go func() { done <- c.Wait() }()
		select {
		case err := <-done:
			code := 0
			if ee, ok := err.(*exec.ExitError); ok {
				code = ee.ExitCode()
			}
			_ = s.Exit(code)
		case <-s.Context().Done():
			// Session gone; leave the process running, as sshd would.
		}
	})
	return d, keyPath
}

// slowStartWriter delays its first Write by delay.
type slowStartWriter struct {
	w     io.Writer
	delay time.Duration
	once  sync.Once
}

func (s *slowStartWriter) Write(p []byte) (int, error) {
	s.once.Do(func() { time.Sleep(s.delay) })
	return s.w.Write(p)
}

// waitForPidFile returns the pid a command wrote to path once it appears.
func waitForPidFile(t *testing.T, path string) int {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		raw, _ := os.ReadFile(path)
		if pid, err := strconv.Atoi(strings.TrimSpace(string(raw))); err == nil {
			return pid
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("command never wrote its pid")
	return 0
}

// assertProcessGone fails unless pid exits within ten seconds. It kills the
// process on failure so a failing test leaves nothing behind.
func assertProcessGone(t *testing.T, pid int) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if err := syscall.Kill(pid, 0); err != nil {
			return // gone
		}
		time.Sleep(100 * time.Millisecond)
	}
	_ = syscall.Kill(pid, syscall.SIGKILL)
	t.Fatalf("remote process %d survived cancellation", pid)
}

func TestSSH_CancelKillsTheRemoteProcessGroup(t *testing.T) {
	d, keyPath := startDetachedSSHD(t)
	ex, err := NewSSH(newSSHConfig(t, d, keyPath))
	if err != nil {
		t.Fatal(err)
	}
	defer ex.Close()

	pidFile := filepath.Join(t.TempDir(), "pid")
	ctx, cancel := context.WithCancel(context.Background())
	started := make(chan struct{})
	go func() {
		_, _ = ex.Run(ctx, "sh -c 'echo $$ > "+pidFile+"; echo up; exec sleep 60'", &RunOpts{Stream: func(l string) {
			if l == "up" {
				close(started)
			}
		}})
	}()
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("command never started")
	}
	cancel()

	assertProcessGone(t, waitForPidFile(t, pidFile))
}

// Cancelling before the marker line has reached the executor must still kill
// the command: Run waits briefly for the marker before giving up on the group.
func TestSSH_CancelBeforeMarkerArrivesKillsTheCommand(t *testing.T) {
	d, keyPath := startDetachedSSHDWith(t, 500*time.Millisecond)
	ex, err := NewSSH(newSSHConfig(t, d, keyPath))
	if err != nil {
		t.Fatal(err)
	}
	defer ex.Close()

	pidFile := filepath.Join(t.TempDir(), "pid")
	ctx, cancel := context.WithCancel(context.Background())
	go func() { _, _ = ex.Run(ctx, "sh -c 'echo $$ > "+pidFile+"; exec sleep 60'", nil) }()
	pid := waitForPidFile(t, pidFile) // running, but its marker is still held back
	cancel()
	assertProcessGone(t, pid)
}

// Run must return promptly after cancellation even when the connection can no
// longer open sessions, so the group kill cannot be what Run waits on.
func TestSSH_CancelReturnsPromptlyWhenNewSessionsStall(t *testing.T) {
	release := make(chan struct{})
	var sessions atomic.Int32
	d, keyPath := startTestSSHDWith(t, func(s gliderssh.Session) {
		c := exec.CommandContext(s.Context(), "sh", "-c", s.RawCommand())
		c.Stdout, c.Stderr, c.Stdin = s, s.Stderr(), s
		_ = c.Run()
		_ = s.Exit(0)
	}, func(srv *gliderssh.Server) {
		srv.ChannelHandlers = map[string]gliderssh.ChannelHandler{
			"session": func(srv *gliderssh.Server, conn *gossh.ServerConn, ch gossh.NewChannel, ctx gliderssh.Context) {
				if sessions.Add(1) > 1 {
					<-release // leave the open request unanswered until the test ends
					_ = ch.Reject(gossh.ConnectionFailed, "test over")
					return
				}
				gliderssh.DefaultSessionHandler(srv, conn, ch, ctx)
			},
		}
	})
	ex, err := NewSSH(newSSHConfig(t, d, keyPath))
	if err != nil {
		t.Fatal(err)
	}
	defer ex.Close()
	defer close(release) // before Close, which waits for the stalled kill

	ctx, cancel := context.WithCancel(context.Background())
	started := make(chan struct{})
	returned := make(chan struct{})
	go func() {
		_, _ = ex.Run(ctx, "echo up; sleep 5", &RunOpts{Stream: func(l string) {
			if l == "up" {
				close(started)
			}
		}})
		close(returned)
	}()
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("command never started")
	}
	cancel()
	select {
	case <-returned:
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return within 2s of cancellation")
	}
}

// A caller that closes the executor right after a cancelled Run returns must
// not abort the group kill still in flight.
func TestSSH_CloseRightAfterCancelStillKillsTheCommand(t *testing.T) {
	d, keyPath := startDetachedSSHD(t)
	ex, err := NewSSH(newSSHConfig(t, d, keyPath))
	if err != nil {
		t.Fatal(err)
	}

	pidFile := filepath.Join(t.TempDir(), "pid")
	ctx, cancel := context.WithCancel(context.Background())
	started := make(chan struct{})
	returned := make(chan struct{})
	go func() {
		_, _ = ex.Run(ctx, "sh -c 'echo $$ > "+pidFile+"; echo up; exec sleep 60'", &RunOpts{Stream: func(l string) {
			if l == "up" {
				close(started)
			}
		}})
		close(returned)
	}()
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("command never started")
	}
	pid := waitForPidFile(t, pidFile)
	cancel()
	<-returned
	_ = ex.Close()
	assertProcessGone(t, pid)
}
