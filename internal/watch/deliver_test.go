package watch

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

type hookServer struct {
	mu     sync.Mutex
	bodies []map[string]any
	ctypes []string
}

func (h *hookServer) handler(status int, delay time.Duration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		var m map[string]any
		_ = json.Unmarshal(b, &m)
		h.mu.Lock()
		h.bodies = append(h.bodies, m)
		h.ctypes = append(h.ctypes, r.Header.Get("Content-Type"))
		h.mu.Unlock()
		time.Sleep(delay)
		w.WriteHeader(status)
	}
}

func (h *hookServer) count() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.bodies)
}

func TestDeliverPostsOpenAndClear(t *testing.T) {
	h := &hookServer{}
	srv := httptest.NewTLSServer(h.handler(200, 0))
	defer srv.Close()
	since := time.Date(2026, 10, 23, 12, 0, 0, 0, time.UTC)
	cleared := since.Add(time.Minute)
	opened := Alert{Key: "exec-down", Message: "The execution client on this box is not running.", Since: since}
	closed := opened
	closed.Cleared = &cleared

	prev := State{}
	next := State{Open: []Alert{opened}}
	last := State{Recent: []Alert{closed}}
	ch := changes(prev, next)
	ch = append(ch, changes(next, last)...)
	if len(ch) != 2 {
		t.Fatalf("changes = %+v, want one open and one clear", ch)
	}
	for _, a := range ch {
		if err := deliver(context.Background(), srv.Client(), srv.URL+"/topic", "box-1", a); err != nil {
			t.Fatal(err)
		}
	}
	if h.count() != 2 {
		t.Fatalf("posts = %d, want 2", h.count())
	}
	for i, b := range h.bodies {
		if h.ctypes[i] != "application/json" {
			t.Errorf("content type %q", h.ctypes[i])
		}
		keys := []string{}
		for k := range b {
			keys = append(keys, k)
		}
		if len(b) != 5 || b["box"] != "box-1" || b["key"] != "exec-down" || b["since"] != "2026-10-23T12:00:00Z" {
			t.Errorf("body %d = %v (keys %v)", i, b, keys)
		}
	}
	if h.bodies[0]["cleared"] != nil || h.bodies[1]["cleared"] != "2026-10-23T12:01:00Z" {
		t.Errorf("cleared = %v / %v", h.bodies[0]["cleared"], h.bodies[1]["cleared"])
	}
}

func TestDeliverRefusesPlainHTTPAndRedirects(t *testing.T) {
	plain := httptest.NewServer(http.NotFoundHandler())
	defer plain.Close()
	if err := deliver(context.Background(), http.DefaultClient, plain.URL, "b", Alert{Key: "k"}); err == nil {
		t.Error("http:// webhook delivered")
	}
	h := &hookServer{}
	target := httptest.NewTLSServer(h.handler(200, 0))
	defer target.Close()
	redir := httptest.NewTLSServer(http.RedirectHandler(target.URL, http.StatusFound))
	defer redir.Close()
	c := redir.Client()
	if err := deliver(context.Background(), c, redir.URL, "b", Alert{Key: "k"}); err == nil {
		t.Error("a redirect counted as delivered")
	}
	if h.count() != 0 {
		t.Error("redirect followed")
	}
}

// The webhook URL is a secret: no log line may carry it, whatever net/http
// puts in its errors.
func TestDeliverNeverLogsURL(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
	h := &hookServer{}
	srv := httptest.NewTLSServer(h.handler(500, 0))
	defer srv.Close()
	secret := "/s3cret-topic-xyz"
	errs := []error{
		deliver(context.Background(), srv.Client(), srv.URL+secret, "b", Alert{Key: "k"}),
		deliver(context.Background(), srv.Client(), "https://127.0.0.1:1"+secret, "b", Alert{Key: "k"}),
		deliver(context.Background(), srv.Client(), "https://bad host"+secret, "b", Alert{Key: "k"}),
	}
	for _, err := range errs {
		if err == nil {
			t.Error("a failing delivery returned nil")
		} else if strings.Contains(err.Error(), secret) {
			t.Errorf("error carries the URL: %v", err)
		}
	}
	if h.count() != 2 {
		t.Errorf("posts to a 500 = %d, want 2 (one retry)", h.count())
	}
	if out := buf.String(); strings.Contains(out, secret) || !strings.Contains(out, "webhook delivery failed") {
		t.Errorf("log = %q", out)
	}
}

func TestDeliverFailureDoesNotBlock(t *testing.T) {
	h := &hookServer{}
	srv := httptest.NewTLSServer(h.handler(500, 2*time.Second))
	defer srv.Close()
	old := newWebhookClient
	newWebhookClient = srv.Client
	t.Cleanup(func() { newWebhookClient = old })
	log.SetOutput(io.Discard)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })

	dir := t.TempDir()
	cfg := Config{
		Endpoints: []Endpoint{{Name: "dead", URL: "http://127.0.0.1:1", Kind: "rpc"}},
		Interval:  50 * time.Millisecond,
		StatePath: filepath.Join(dir, "state.json"),
		Webhook:   srv.URL + "/topic",
	}
	// Five polls must land while one webhook attempt is still hanging (2s).
	// Stop as soon as they do: a fixed window flaked under full-suite load.
	ctx, cancel := context.WithTimeout(context.Background(), 1900*time.Millisecond)
	defer cancel()
	polls := 0
	var last time.Time
	go func() { _ = Run(ctx, cfg) }()
	for ctx.Err() == nil && (polls < 5 || h.count() == 0) {
		if s, err := ReadState(cfg.StatePath); err == nil && s.At.After(last) {
			last = s.At
			polls++
		}
		time.Sleep(10 * time.Millisecond)
	}
	if polls < 5 {
		t.Errorf("polls = %d while the webhook hung, want >= 5", polls)
	}
	if h.count() == 0 {
		t.Error("the open alert was never posted")
	}
}
