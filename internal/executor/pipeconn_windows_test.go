//go:build windows

package executor

import (
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"golang.org/x/sys/windows"
)

// testPipe serves one instance of a fresh named pipe and returns its name and
// the server end (a synchronous handle is fine for the test's side).
func testPipe(t *testing.T) (string, windows.Handle) {
	t.Helper()
	name := fmt.Sprintf(`\\.\pipe\jumpgate-test-%d-%d`, os.Getpid(), time.Now().UnixNano())
	p, _ := windows.UTF16PtrFromString(name)
	h, err := windows.CreateNamedPipe(p, windows.PIPE_ACCESS_DUPLEX, windows.PIPE_TYPE_BYTE|windows.PIPE_WAIT, 1, 4096, 4096, 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { windows.CloseHandle(h) })
	return name, h
}

// The deadlock that hung the agent test: x/crypto's agent client parks a
// reader on the pipe and then writes a request. On a synchronous handle that
// write waits behind the read for ever; pipeConn must let both proceed.
func TestPipeConnWriteWhileAReadIsPending(t *testing.T) {
	name, srv := testPipe(t)
	c, err := openPipe(name)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	_ = c.SetDeadline(time.Now().Add(10 * time.Second))

	got := make(chan string, 1)
	go func() {
		b := make([]byte, 4)
		n, err := c.Read(b)
		if err != nil {
			got <- err.Error()
			return
		}
		got <- string(b[:n])
	}()
	time.Sleep(200 * time.Millisecond) // the read is now pending

	if _, err := c.Write([]byte("ping")); err != nil {
		t.Fatalf("write behind a pending read: %v", err)
	}
	buf := make([]byte, 4)
	var n uint32
	if err := windows.ReadFile(srv, buf, &n, nil); err != nil || string(buf[:n]) != "ping" {
		t.Fatalf("server read %q, %v", buf[:n], err)
	}
	if err := windows.WriteFile(srv, []byte("pong"), &n, nil); err != nil {
		t.Fatal(err)
	}
	select {
	case s := <-got:
		if s != "pong" {
			t.Fatalf("read %q", s)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the pending read never completed")
	}
}

func TestPipeConnReadHonoursTheDeadlineAndClose(t *testing.T) {
	name, _ := testPipe(t)
	c, err := openPipe(name)
	if err != nil {
		t.Fatal(err)
	}
	_ = c.SetDeadline(time.Now().Add(300 * time.Millisecond))
	start := time.Now()
	if _, err := c.Read(make([]byte, 1)); !errors.Is(err, os.ErrDeadlineExceeded) {
		t.Fatalf("read on a silent pipe = %v, want a deadline error", err)
	}
	if time.Since(start) > 3*time.Second {
		t.Fatal("the deadline was not honoured promptly")
	}

	_ = c.SetDeadline(time.Time{})
	done := make(chan error, 1)
	go func() { _, err := c.Read(make([]byte, 1)); done <- err }()
	time.Sleep(200 * time.Millisecond)
	c.Close()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("a read cancelled by Close returned no error")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Close did not cancel the pending read")
	}
}
