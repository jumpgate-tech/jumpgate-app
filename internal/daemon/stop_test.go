package daemon

import (
	"context"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// shutdownServer publishes a healthy server whose /api/shutdown answers
// status, and records the Authorization header each shutdown carried.
func shutdownServer(t *testing.T, status int) *[]string {
	t.Helper()
	h, err := Acquire()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(h.Release)
	dir, _ := RunDir()
	sock := filepath.Join(dir, "server.sock")
	ln, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	var auths []string
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/shutdown" && r.Method == http.MethodPost {
			mu.Lock()
			auths = append(auths, r.Header.Get("Authorization"))
			mu.Unlock()
			w.WriteHeader(status)
			return
		}
		w.Write([]byte(`{"ok":true}`))
	})}
	go srv.Serve(ln)
	t.Cleanup(func() { srv.Close() })
	if err := h.Publish(Info{PID: os.Getpid(), Socket: sock, Token: "tok", StartedAt: time.Now()}); err != nil {
		t.Fatal(err)
	}
	return &auths
}

// Stop asks the running server over its authenticated API, with the token
// from its discovery file; it never signals a pid.
func TestStopAsksTheServerWithItsToken(t *testing.T) {
	isolate(t)
	auths := shutdownServer(t, http.StatusAccepted)
	if err := Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(*auths) != 1 || (*auths)[0] != "Bearer tok" {
		t.Fatalf("shutdown requests carried %v", *auths)
	}
}

// A server that does not accept the shutdown is reported, not assumed gone.
func TestStopReportsARefusedShutdown(t *testing.T) {
	isolate(t)
	shutdownServer(t, http.StatusUnauthorized)
	if err := Stop(context.Background()); err == nil || !strings.Contains(err.Error(), "401") {
		t.Fatalf("Stop = %v", err)
	}
}

func TestStopWithNoServerSaysSo(t *testing.T) {
	isolate(t)
	if err := Stop(context.Background()); err == nil || !strings.Contains(err.Error(), "no jumpgate server is running") {
		t.Fatalf("Stop = %v", err)
	}
}
