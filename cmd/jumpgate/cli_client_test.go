package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/valve-tech/jumpgate/internal/apiclient"
)

// withServer points connect at an in-process stand-in for the server, so a
// test drives the CLI's server paths without starting a detached one.
func withServer(t *testing.T, h http.HandlerFunc) {
	t.Helper()
	ts := httptest.NewServer(h)
	t.Cleanup(ts.Close)
	old := connect
	connect = func(context.Context) (*apiclient.Client, error) { return apiclient.NewHTTP(ts.URL, "tok"), nil }
	t.Cleanup(func() { connect = old })
}

// captureStdio swaps os.Stdout and os.Stderr for files until the test ends,
// for the commands that print there; the returned funcs read what was written.
func captureStdio(t *testing.T) (stdout, stderr func() string) {
	t.Helper()
	read := func(f *os.File) func() string {
		return func() string { b, _ := os.ReadFile(f.Name()); return string(b) }
	}
	open := func(name string) *os.File {
		f, err := os.Create(filepath.Join(t.TempDir(), name))
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { f.Close() })
		return f
	}
	out, errw := open("stdout"), open("stderr")
	oldOut, oldErr := os.Stdout, os.Stderr
	os.Stdout, os.Stderr = out, errw
	t.Cleanup(func() { os.Stdout, os.Stderr = oldOut, oldErr })
	return read(out), read(errw)
}

// An intent goes through the connect seam and the shared client, and the
// server's rejection hint reaches the operator.
func TestRunIntentGoesThroughConnect(t *testing.T) {
	var path string
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		fmt.Fprint(w, `{"status":1,"rejection":{"code":"busy","message":"later"},"hint":"server says"}`)
	})
	_, stderr := captureStdio(t)
	if code := runIntent("box", "status.read", struct{}{}); code != 1 {
		t.Fatalf("exit %d, stderr %q", code, stderr())
	}
	if path != "/api/targets/box/intent/status.read" || !strings.Contains(stderr(), "server says") {
		t.Fatalf("path %q, stderr %q", path, stderr())
	}
}

// A pairing error event carries "error" (it was "err"); its code decides the
// exit status like any server error.
func TestStreamPairReportsTheErrorEvent(t *testing.T) {
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"step\":\"verify\",\"line\":\"start\"}\n\n")
		fmt.Fprint(w, "data: {\"step\":\"verify\",\"error\":\"not signed\",\"code\":\"bad_receipt\"}\n\n")
	})
	c, _ := connect(context.Background())
	stdout, stderr := captureStdio(t)
	if code := streamPair(c, "box", false); code != 4 {
		t.Fatalf("exit %d, stderr %q", code, stderr())
	}
	if !strings.Contains(stdout(), "[verify] start") || !strings.Contains(stderr(), "SECURITY") || !strings.Contains(stderr(), "not signed") {
		t.Fatalf("stdout %q, stderr %q", stdout(), stderr())
	}
}

// hosts add reaches the server through connect (Ruling T2e), re-pairs a name
// the server answers target_exists for, and prints the paired agent.
func TestHostsAddGoesThroughConnect(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	var paths []string
	withServer(t, func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.Method+" "+r.URL.Path)
		switch r.URL.Path {
		case "/api/targets":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			fmt.Fprint(w, `{"error":"target box already exists","code":"target_exists"}`)
		case "/api/targets/box/pair":
			w.Header().Set("Content-Type", "text/event-stream")
			fmt.Fprint(w, "data: {\"done\":true,\"agent\":\"0xabc\"}\n\n")
		default:
			http.NotFound(w, r)
		}
	})
	stdout, stderr := captureStdio(t)
	if code := hostsAdd([]string{"box", "--local"}); code != 0 {
		t.Fatalf("exit %d, stderr %q", code, stderr())
	}
	if strings.Join(paths, ",") != "POST /api/targets,POST /api/targets/box/pair" {
		t.Fatalf("requests %v", paths)
	}
	if !strings.Contains(stdout(), "pairing it again") || !strings.Contains(stdout(), "paired: agent 0xabc") {
		t.Fatalf("stdout %q", stdout())
	}
}
