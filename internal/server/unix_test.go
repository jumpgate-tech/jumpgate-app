// internal/server/unix_test.go
package server

import (
	"context"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
	"time"

	"github.com/valve-tech/jumpgate/internal/testutil"
)

func TestServeUnixAndShutdown(t *testing.T) {
	dir := testutil.ShortTempDir(t)
	sock := filepath.Join(dir, "s.sock")
	token := NewSessionToken()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stopped := make(chan struct{})
	s := New(Config{Token: token, UI: fstest.MapFS{}, Shutdown: func() { close(stopped) }})
	go s.ServeUnix(ctx, sock)

	client := &http.Client{Transport: &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", sock)
	}}}
	var res *http.Response
	var err error
	for i := 0; i < 50; i++ { // wait for the listener
		req, _ := http.NewRequest(http.MethodGet, "http://x/api/health", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		if res, err = client.Do(req); err == nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if err != nil || res.StatusCode != http.StatusOK {
		t.Fatalf("health over unix socket: %v", err)
	}
	testutil.AssertPrivate(t, sock)

	req, _ := http.NewRequest(http.MethodPost, "http://x/api/shutdown", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	res, err = client.Do(req)
	if err != nil || res.StatusCode != http.StatusAccepted {
		t.Fatalf("shutdown: %v %v", res, err)
	}
	select {
	case <-stopped:
	case <-time.After(2 * time.Second):
		t.Fatal("Shutdown hook not called")
	}
}

func TestServeUnixRefusesToReplaceANonSocket(t *testing.T) {
	dir := testutil.ShortTempDir(t)
	path := filepath.Join(dir, "s.sock")
	if err := os.WriteFile(path, []byte("precious"), 0o600); err != nil {
		t.Fatal(err)
	}
	s := New(Config{Token: "t", UI: fstest.MapFS{}})
	if err := s.ServeUnix(context.Background(), path); err == nil {
		t.Fatal("ServeUnix replaced a regular file")
	}
	if b, _ := os.ReadFile(path); string(b) != "precious" {
		t.Fatalf("file was modified: %q", b)
	}
}

func TestServeUnixReplacesAStaleSocket(t *testing.T) {
	dir := testutil.ShortTempDir(t)
	path := filepath.Join(dir, "s.sock")
	ln, err := net.Listen("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	ln.(*net.UnixListener).SetUnlinkOnClose(false)
	ln.Close() // leaves a dead socket file behind
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	s := New(Config{Token: "t", UI: fstest.MapFS{}})
	go func() { done <- s.ServeUnix(ctx, path) }()
	for i := 0; i < 50; i++ {
		if c, err := net.Dial("unix", path); err == nil {
			c.Close()
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("ServeUnix = %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("ServeUnix did not return after cancel")
	}
}

// I-2: a server killed hard (Task Manager, a crash) leaves its socket file
// behind. The next server must replace it, on every OS, including Windows
// where the file is an AF_UNIX reparse point.
func TestServeUnixServesHealthOverAStaleSocket(t *testing.T) {
	dir := testutil.ShortTempDir(t)
	sock := filepath.Join(dir, "s.sock")
	ln, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatal(err)
	}
	ln.(*net.UnixListener).SetUnlinkOnClose(false)
	ln.Close()
	if _, err := os.Lstat(sock); err != nil {
		t.Fatalf("no stale socket left behind: %v", err)
	}

	token := NewSessionToken()
	ctx, cancel := context.WithCancel(context.Background())
	served := make(chan error, 1)
	go func() { served <- New(Config{Token: token, UI: fstest.MapFS{}}).ServeUnix(ctx, sock) }()
	client := &http.Client{Transport: &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", sock)
	}}}
	var res *http.Response
	for i := 0; i < 50; i++ {
		req, _ := http.NewRequest(http.MethodGet, "http://x/api/health", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		if res, err = client.Do(req); err == nil {
			res.Body.Close()
			break
		}
		select {
		case err := <-served:
			t.Fatalf("ServeUnix over a stale socket: %v", err)
		case <-time.After(20 * time.Millisecond):
		}
	}
	if err != nil || res.StatusCode != http.StatusOK {
		t.Fatalf("health: %v", err)
	}
	cancel()
	<-served
}
