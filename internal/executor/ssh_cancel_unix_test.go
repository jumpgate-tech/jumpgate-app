//go:build unix

package executor

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	gliderssh "github.com/gliderlabs/ssh"
)

// startDetachedSSHD behaves like a real sshd in the one way that matters here:
// closing a session does not kill the command it started.
func startDetachedSSHD(t *testing.T) (testSSHD, string) {
	t.Helper()
	d, keyPath := startTestSSHDWith(t, func(s gliderssh.Session) {
		c := exec.Command("sh", "-c", s.RawCommand())
		c.Stdout, c.Stderr, c.Stdin = s, s.Stderr(), s
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

	raw, _ := os.ReadFile(pidFile)
	pid, _ := strconv.Atoi(strings.TrimSpace(string(raw)))
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if err := syscall.Kill(pid, 0); err != nil {
			return // gone
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("remote process %d survived cancellation", pid)
}
