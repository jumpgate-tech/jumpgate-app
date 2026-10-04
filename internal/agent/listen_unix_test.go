//go:build linux || darwin

package agent

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

// peek returns the first n bytes waiting on c without consuming them.
func peek(c net.Conn, n int) ([]byte, error) {
	raw, err := c.(*net.UnixConn).SyscallConn()
	if err != nil {
		return nil, err
	}
	buf := make([]byte, n)
	deadline := time.Now().Add(5 * time.Second)
	for {
		var got int
		var rerr error
		if err := raw.Control(func(fd uintptr) {
			got, _, rerr = unix.Recvfrom(int(fd), buf, unix.MSG_PEEK|unix.MSG_DONTWAIT)
		}); err != nil {
			return nil, err
		}
		if rerr != nil && !errors.Is(rerr, unix.EAGAIN) {
			return nil, rerr
		}
		if got == n {
			return buf, nil
		}
		if time.Now().After(deadline) {
			return buf[:max(got, 0)], fmt.Errorf("only %d of %d bytes arrived", got, n)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// A refused peer's connection is closed before net/http sees it: no response,
// and none of what it sent is consumed. Serve keeps accepting afterwards.
func TestRefusedPeerIsClosedUnreadAndServeContinues(t *testing.T) {
	r := newRig(t, true)
	sock := filepath.Join(shortDir(t), "agent.sock")
	ln, err := Listen(sock, -1, 0o660)
	if err != nil {
		t.Fatal(err)
	}

	var refuse atomic.Bool
	refuse.Store(true)
	written := make(chan []byte, 1)
	gateErr := make(chan error, 1)
	orig := peerGate
	peerGate = func(a *Agent, c net.Conn, gid int) bool {
		if !refuse.Load() {
			return orig(a, c, gid)
		}
		// Everything the refused client sent must still be unread here.
		want := <-written
		got, err := peek(c, len(want))
		if err == nil && !bytes.Equal(got, want) {
			err = fmt.Errorf("gate saw %q, client sent %q", got, want)
		}
		gateErr <- err
		return false
	}
	t.Cleanup(func() { peerGate = orig })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go Serve(ctx, r.a, ln)

	c, err := net.Dial("unix", sock)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	body := `{"intent":{}}`
	req := []byte("POST /v1/intent HTTP/1.1\r\nHost: agent\r\nContent-Type: application/json\r\nContent-Length: " +
		strconv.Itoa(len(body)) + "\r\n\r\n" + body)
	if _, err := c.Write(req); err != nil {
		t.Fatal(err)
	}
	written <- req
	if err := <-gateErr; err != nil {
		t.Fatalf("the refused request was read before the gate: %v", err)
	}
	_ = c.SetReadDeadline(time.Now().Add(5 * time.Second))
	resp, err := io.ReadAll(c)
	// Linux reports ECONNRESET when a socket closes with input unread; macOS
	// reports EOF. Either means closed with no response; a timeout does not.
	if len(resp) != 0 || (err != nil && !errors.Is(err, unix.ECONNRESET)) {
		t.Fatalf("refused peer got %q (err %v), want the connection closed with no response", resp, err)
	}

	refuse.Store(false)
	res, err := unixClient(sock).Post("http://agent/v1/intent", "application/json", bytes.NewReader([]byte("{}")))
	if err != nil {
		t.Fatalf("Serve stopped accepting after a refusal: %v", err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("allowed peer after a refusal: %d", res.StatusCode)
	}
}
