package server

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// startServing runs s.ListenAndServe on a free loopback port and waits until
// it accepts connections. It returns the address and the channel
// ListenAndServe's result arrives on.
func startServing(t *testing.T, ctx context.Context, token string, mutate func(*Config)) (*Server, string, <-chan error) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve a port: %v", err)
	}
	addr := ln.Addr().String()
	ln.Close()

	cfg := Config{Bind: addr, Token: token}
	if mutate != nil {
		mutate(&cfg)
	}
	s := New(cfg)
	errCh := make(chan error, 1)
	go func() { errCh <- s.ListenAndServe(ctx) }()

	deadline := time.Now().Add(5 * time.Second)
	for {
		req, _ := http.NewRequest("GET", "http://"+addr+"/api/health", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		res, err := http.DefaultClient.Do(req)
		if err == nil {
			res.Body.Close()
			return s, addr, errCh
		}
		if time.Now().After(deadline) {
			t.Fatalf("the server never accepted a connection: %v", err)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// Canceling request contexts frees every handler that watches its context. One
// that does not (stuck in a call that takes no context) must not hold the
// process open either: shutdown gives up after its grace and closes the
// connection.
func TestServeUntil_ClosesAHandlerThatIgnoresItsContext(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve a port: %v", err)
	}
	addr := ln.Addr().String()
	ln.Close()

	entered := make(chan struct{})
	stuck := make(chan struct{})
	defer close(stuck)
	srv := newHTTPServer(addr, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/stuck" {
			close(entered)
			<-stuck
		}
	}))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	errCh := make(chan error, 1)
	go func() { errCh <- serveUntil(ctx, srv, 50*time.Millisecond) }()

	reqErr := make(chan error, 1)
	go func() {
		deadline := time.Now().Add(5 * time.Second)
		for {
			res, err := http.Get("http://" + addr + "/stuck")
			if err == nil {
				res.Body.Close()
			}
			// Retry only until the listener is up; once the handler has
			// been entered, this request's outcome is the one that counts.
			select {
			case <-entered:
				reqErr <- err
				return
			default:
			}
			if time.Now().After(deadline) {
				reqErr <- err
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
	}()
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("the stuck handler was never reached")
	}

	cancel()
	select {
	case err := <-errCh:
		if err != nil {
			t.Errorf("a forced close after the grace period returned %v, want nil", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("serveUntil did not return: shutdown is still waiting on a handler that ignores its context")
	}
	select {
	case err := <-reqErr:
		if err == nil {
			t.Error("the stuck request completed normally; its connection should have been closed")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the stuck request's connection was never closed")
	}
}

// Ctrl-C with a UI tab open used to hang. http.Server.Shutdown waits for
// every active request to finish, an SSE stream only finishes when its
// request context ends, and Shutdown does not end request contexts. So
// Shutdown waited on the stream and the stream waited on Shutdown.
func TestListenAndServe_ShutsDownWithAnSSEStreamOpen(t *testing.T) {
	token := NewSessionToken()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s, addr, errCh := startServing(t, ctx, token, nil)

	// A claimed slot is all the setup stream needs to stay open.
	c, ok := s.claimSetupRun(httptest.NewRecorder(), "box")
	if !ok {
		t.Fatal("could not claim a setup slot")
	}
	defer s.releaseSetupRun("box", c)

	// Its own transport, not http.DefaultClient's. startServing's readiness
	// poll leaves a connection returning to that pool, and a request sent
	// right behind it can dial a spare connection and then reuse the pooled
	// one. The spare one never sends a request, so the server holds it as
	// StateNew, and Shutdown waits on a StateNew connection until it is 5s
	// old: exactly this test's bound, so it failed about one run in six.
	client := &http.Client{Transport: &http.Transport{}}
	defer client.CloseIdleConnections()
	req, _ := http.NewRequest("GET", "http://"+addr+"/api/targets/box/setup/stream", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	res, err := client.Do(req)
	if err != nil {
		t.Fatalf("open the stream: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("stream status = %d, want 200", res.StatusCode)
	}
	// Headers arrived, so the handler is inside its stream loop.

	cancel()
	select {
	case err := <-errCh:
		if err != nil {
			t.Errorf("shutdown with a stream open returned %v, want a clean nil", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("ListenAndServe did not return: shutdown is waiting on the open SSE stream")
	}
}
