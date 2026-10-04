// internal/daemon/daemon_test.go
package daemon

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func isolate(t *testing.T) {
	t.Helper()
	// unix socket paths must stay short, so HOME itself is short here.
	home, err := os.MkdirTemp("/tmp", "jgh")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(home) })
	t.Setenv("HOME", home)
}

func TestAcquireIsSingleInstance(t *testing.T) {
	isolate(t)
	h, err := Acquire()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Acquire(); !errors.Is(err, ErrAlreadyRunning) {
		t.Fatalf("second Acquire = %v, want ErrAlreadyRunning", err)
	}
	h.Release()
	h2, err := Acquire()
	if err != nil {
		t.Fatalf("Acquire after Release: %v", err)
	}
	h2.Release()
}

// Review Focus 5: a server.json left by a crashed server, naming a pid that
// now belongs to something else, is never trusted and never signalled.
func TestFindIgnoresAStaleDiscoveryFile(t *testing.T) {
	isolate(t)
	dir, _ := RunDir()
	stale := Info{PID: os.Getpid(), Socket: filepath.Join(dir, "server.sock"), Token: "x"}
	b, _ := json.Marshal(stale)
	if err := os.WriteFile(filepath.Join(dir, "server.json"), b, 0o600); err != nil {
		t.Fatal(err)
	}
	_, running, err := Find(context.Background())
	if err != nil || running {
		t.Fatalf("Find = running %v, err %v; want not running", running, err)
	}
	if _, err := os.Stat(filepath.Join(dir, "server.json")); !os.IsNotExist(err) {
		t.Fatal("stale server.json was left in place")
	}
}

// A live server: lock held, socket answering /api/health with the token.
func TestFindSeesALiveServer(t *testing.T) {
	isolate(t)
	h, err := Acquire()
	if err != nil {
		t.Fatal(err)
	}
	defer h.Release()
	dir, _ := RunDir()
	sock := filepath.Join(dir, "server.sock")
	ln, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatal(err)
	}
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/health" && r.Header.Get("Authorization") == "Bearer tok" {
			w.Write([]byte(`{"ok":true}`))
			return
		}
		http.Error(w, "no", http.StatusUnauthorized)
	})}
	go srv.Serve(ln)
	defer srv.Close()
	if err := h.Publish(Info{PID: os.Getpid(), Socket: sock, Token: "tok", StartedAt: time.Now()}); err != nil {
		t.Fatal(err)
	}
	info, running, err := Find(context.Background())
	if err != nil || !running || info.Token != "tok" {
		t.Fatalf("Find = %+v, %v, %v", info, running, err)
	}
	fi, _ := os.Stat(filepath.Join(dir, "server.json"))
	if fi.Mode().Perm() != 0o600 {
		t.Errorf("server.json mode %o, want 600 (it holds the token)", fi.Mode().Perm())
	}
}
