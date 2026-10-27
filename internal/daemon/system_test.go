package daemon

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/valve-tech/jumpgate/internal/buildinfo"
	"github.com/valve-tech/jumpgate/internal/testutil"
)

func TestPublishSystemIsWorldReadableAndTokenFree(t *testing.T) {
	dir := t.TempDir()
	if err := PublishSystem(dir, SystemInfo{PID: 1, UID: 999, Lock: "locked"}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "system.json")
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && fi.Mode()&0o777 != 0o644 {
		t.Errorf("mode = %o, want 644", fi.Mode()&0o777)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	if _, ok := m["token"]; ok {
		t.Error("system.json has a token field")
	}
	if m["lock"] != "locked" {
		t.Errorf("lock = %v, want locked", m["lock"])
	}
	RemoveSystem(dir)
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("RemoveSystem left system.json: %v", err)
	}
}

// systemServer runs a fake jumpgate service: a socket in a fresh SystemDir
// answering /api/health with no token, and its system.json.
func systemServer(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the jumpgate service runs on Linux")
	}
	old := SystemDir
	SystemDir = testutil.ShortTempDir(t)
	t.Cleanup(func() { SystemDir = old })
	sock := filepath.Join(SystemDir, "server.sock")
	ln, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatal(err)
	}
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" {
			http.Error(w, "a token was sent to the service", http.StatusBadRequest)
			return
		}
		if r.URL.Path == "/api/health" {
			w.Write([]byte(`{"ok":true}`))
			return
		}
		http.NotFound(w, r)
	})}
	go srv.Serve(ln)
	t.Cleanup(func() { srv.Close() })
	if err := PublishSystem(SystemDir, SystemInfo{PID: os.Getpid(), Socket: sock, UID: os.Getuid(), Lock: "locked", StartedAt: time.Now()}); err != nil {
		t.Fatal(err)
	}
	return sock
}

func skipIfRoot(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("needs a non-root unix user")
	}
}

func TestFindPrefersTheSystemServer(t *testing.T) {
	isolate(t)
	liveServer(t, buildinfo.Version())
	sock := systemServer(t)
	info, ok, err := Find(context.Background())
	if err != nil || !ok {
		t.Fatalf("Find = %+v, %v, %v", info, ok, err)
	}
	if !info.System || info.Socket != sock || info.Token != "" || info.Lock != "locked" {
		t.Fatalf("Find = %+v, want the system server at %s", info, sock)
	}
}

func TestFindSystemNotAdminGivesTheUsermodHint(t *testing.T) {
	skipIfRoot(t)
	isolate(t)
	sock := systemServer(t)
	if err := os.Chmod(sock, 0); err != nil {
		t.Fatal(err)
	}
	_, _, err := Find(context.Background())
	if !errors.Is(err, ErrNotAdmin) {
		t.Fatalf("err = %v, want ErrNotAdmin", err)
	}
	if !strings.Contains(err.Error(), "usermod -aG jumpgate-admin") || !strings.Contains(err.Error(), "log in again") {
		t.Fatalf("err = %q lacks the usermod hint", err)
	}
}

func TestFindSystemIgnoresAFileOwnedBySomeoneElse(t *testing.T) {
	skipIfRoot(t)
	isolate(t)
	sock := systemServer(t)
	if err := PublishSystem(SystemDir, SystemInfo{PID: os.Getpid(), Socket: sock, UID: os.Getuid() + 1}); err != nil {
		t.Fatal(err)
	}
	info, ok, err := FindSystem(context.Background())
	if ok || err != nil {
		t.Fatalf("FindSystem = %+v, %v, %v; want a file not owned by its uid ignored", info, ok, err)
	}
}

// noStart fails the test if EnsureRunning tries to start a server.
func noStart(t *testing.T) {
	t.Helper()
	old := startServer
	startServer = func(*exec.Cmd) (*exec.Cmd, error) {
		t.Error("EnsureRunning started a per-user server")
		return nil, errors.New("test: not starting")
	}
	t.Cleanup(func() { startServer = old })
}

func TestEnsureRunningNeverStartsAUserServerBesideTheService(t *testing.T) {
	isolate(t)
	noStart(t)
	old, oldDir := SystemUnitPaths, SystemDir
	SystemDir = t.TempDir()
	unit := filepath.Join(t.TempDir(), "jumpgate.service")
	if err := os.WriteFile(unit, []byte("[Unit]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	SystemUnitPaths = []string{unit}
	t.Cleanup(func() { SystemUnitPaths, SystemDir = old, oldDir })
	_, err := EnsureRunning(context.Background(), "/path/to/jumpgate", nil)
	if err == nil || !strings.Contains(err.Error(), "systemctl start jumpgate") {
		t.Fatalf("err = %v, want the systemctl start hint", err)
	}
}

func TestEnsureRunningNeverStartsAUserServerForANonAdmin(t *testing.T) {
	skipIfRoot(t)
	isolate(t)
	noStart(t)
	sock := systemServer(t)
	if err := os.Chmod(sock, 0); err != nil {
		t.Fatal(err)
	}
	_, err := EnsureRunning(context.Background(), "/path/to/jumpgate", nil)
	if !errors.Is(err, ErrNotAdmin) {
		t.Fatalf("err = %v, want ErrNotAdmin", err)
	}
	dir, _ := RunDir()
	if _, err := os.Stat(filepath.Join(dir, "server.json")); !os.IsNotExist(err) {
		t.Fatalf("server.json exists: %v", err)
	}
}

func TestStopRefusesTheService(t *testing.T) {
	isolate(t)
	systemServer(t)
	if err := Stop(context.Background()); !errors.Is(err, ErrServiceManaged) {
		t.Fatalf("Stop = %v, want ErrServiceManaged", err)
	}
}
