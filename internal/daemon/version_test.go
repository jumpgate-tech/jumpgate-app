package daemon

import (
	"context"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/valve-tech/jumpgate/internal/buildinfo"
)

// liveServer publishes a server.json for a healthy fake server of version v.
func liveServer(t *testing.T, v string) {
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
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"ok":true}`))
	})}
	go srv.Serve(ln)
	t.Cleanup(func() { srv.Close() })
	if err := h.Publish(Info{PID: os.Getpid(), Socket: sock, Token: "tok", Version: v, StartedAt: time.Now()}); err != nil {
		t.Fatal(err)
	}
}

// B3/E2c: a CLI upgraded under a running server of another version is told so,
// with the way out, instead of meeting a bare 404 later.
func TestEnsureRunningWarnsOfAnotherVersion(t *testing.T) {
	isolate(t)
	liveServer(t, "v0.0.1-old")
	var warn strings.Builder
	info, err := EnsureRunning(context.Background(), "/nonexistent/jumpgate", &warn)
	if err != nil || info.Version != "v0.0.1-old" {
		t.Fatalf("EnsureRunning = %+v, %v", info, err)
	}
	want := "a different jumpgate version (v0.0.1-old) is running; restart it with `jumpgate stop`"
	if !strings.Contains(warn.String(), want) {
		t.Fatalf("warning = %q, want it to contain %q", warn.String(), want)
	}
}

func TestEnsureRunningIsQuietForTheSameVersion(t *testing.T) {
	isolate(t)
	liveServer(t, buildinfo.Version())
	var warn strings.Builder
	if _, err := EnsureRunning(context.Background(), "/nonexistent/jumpgate", &warn); err != nil {
		t.Fatal(err)
	}
	if warn.Len() != 0 {
		t.Fatalf("warned for the same version: %q", warn.String())
	}
}

func TestSkewWarning(t *testing.T) {
	if s := SkewWarning(Info{Version: buildinfo.Version()}); s != "" {
		t.Fatalf("same version: %q", s)
	}
	if s := SkewWarning(Info{}); !strings.Contains(s, "(unknown)") || !strings.Contains(s, "jumpgate stop") {
		t.Fatalf("no version recorded: %q", s)
	}
}
