package apiclient

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/buildinfo"
)

func TestParseSSE(t *testing.T) {
	in := ": ping\n\n" +
		"data: {\"a\":1}\n\n" +
		"event: reset\ndata: [1,\ndata: 2]\n\n" +
		"id: 7\ndata: x\r\n\r\n" +
		"data: unterminated"
	var got []Event
	if err := parseSSE(strings.NewReader(in), func(e Event) { got = append(got, e) }); err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("got %d events: %+v", len(got), got)
	}
	if got[0].Name != "" || string(got[0].Data) != `{"a":1}` {
		t.Errorf("event 0 = %+v", got[0])
	}
	if got[1].Name != "reset" || string(got[1].Data) != "[1,\n2]" {
		t.Errorf("event 1 = %+v", got[1])
	}
	if string(got[2].Data) != "x" {
		t.Errorf("event 2 = %+v (CRLF must be accepted)", got[2])
	}
}

func TestBackoff(t *testing.T) {
	for attempt, want := range []time.Duration{250 * time.Millisecond, 500 * time.Millisecond, time.Second, 2 * time.Second, 4 * time.Second, 8 * time.Second, 8 * time.Second} {
		if got := backoff(attempt, 0.5); got != want {
			t.Errorf("backoff(%d, mid) = %v, want %v", attempt, got, want)
		}
	}
	if lo, hi := backoff(0, 0), backoff(0, 0.999999); lo < 200*time.Millisecond || hi > 300*time.Millisecond {
		t.Errorf("jitter outside ±20%%: %v..%v", lo, hi)
	}
	if backoff(1000, 0.5) != 8*time.Second {
		t.Error("a large attempt must stay capped")
	}
}

func fastStreams(t *testing.T) {
	t.Helper()
	oldIdle, oldBase, oldMax := idleTimeout, baseDelay, maxDelay
	idleTimeout, baseDelay, maxDelay = 300*time.Millisecond, 10*time.Millisecond, 20*time.Millisecond
	t.Cleanup(func() { idleTimeout, baseDelay, maxDelay = oldIdle, oldBase, oldMax })
}

// openStream starts a stream that the test's cleanup ends and drains, so the
// stream's goroutine is gone before fastStreams restores the timings (cleanups
// run last-registered first).
func openStream(t *testing.T, c *Client, path string) <-chan StreamMsg {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	ch := c.Stream(ctx, path)
	t.Cleanup(func() {
		cancel()
		for range ch {
		}
	})
	return ch
}

func next(t *testing.T, ch <-chan StreamMsg, want func(StreamMsg) bool) StreamMsg {
	t.Helper()
	deadline := time.After(3 * time.Second)
	for {
		select {
		case m, ok := <-ch:
			if !ok {
				t.Fatal("stream closed")
			}
			if want(m) {
				return m
			}
		case <-deadline:
			t.Fatal("timed out")
		}
	}
}

// Review Focus 1: the server drops the stream (a restart); the client says
// so and comes back on its own, and the consumer sees the second event.
func TestStreamReconnectsAfterTheServerDrops(t *testing.T) {
	fastStreams(t)
	var n atomic.Int32
	c := serve(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintf(w, "data: %d\n\n", n.Add(1))
		w.(http.Flusher).Flush() // then return: the connection ends
	})
	ch := openStream(t, c, "/s")
	first := next(t, ch, func(m StreamMsg) bool { return m.Event != nil })
	next(t, ch, func(m StreamMsg) bool { return m.State == Retrying && m.Err != nil })
	second := next(t, ch, func(m StreamMsg) bool { return m.Event != nil })
	if string(first.Event.Data) != "1" || string(second.Event.Data) != "2" {
		t.Fatalf("events %q then %q", first.Event.Data, second.Event.Data)
	}
}

// A connection cut in the middle of an event never finished it: the half
// event is dropped, not delivered (and not glued onto the next connection's
// first event).
func TestStreamDropsAnEventCutByADisconnect(t *testing.T) {
	fastStreams(t)
	var n atomic.Int32
	c := serve(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		if n.Add(1) == 1 {
			fmt.Fprint(w, "data: 1\n\nevent: reset\ndata: half")
		} else {
			fmt.Fprint(w, "data: 2\n\n")
		}
		w.(http.Flusher).Flush()
	})
	ch := openStream(t, c, "/s")
	first := next(t, ch, func(m StreamMsg) bool { return m.Event != nil })
	second := next(t, ch, func(m StreamMsg) bool { return m.Event != nil })
	if string(first.Event.Data) != "1" || second.Event.Name != "" || string(second.Event.Data) != "2" {
		t.Fatalf("events %+v then %+v", *first.Event, *second.Event)
	}
}

// A stream that stays open but says nothing (not even a ping) is dead.
func TestStreamTreatsSilenceAsDead(t *testing.T) {
	fastStreams(t)
	c := serve(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.(http.Flusher).Flush()
		<-r.Context().Done()
	})
	m := next(t, openStream(t, c, "/s"), func(m StreamMsg) bool { return m.State == Retrying })
	if m.Err == nil || m.Err.Error() != "the stream went silent" {
		t.Fatalf("retry error %v", m.Err)
	}
}

// A server that accepts the connection but never answers it is as dead as
// one that goes quiet after answering.
func TestStreamTreatsAHungConnectAsSilent(t *testing.T) {
	fastStreams(t)
	c := serve(t, func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done() // no headers, ever
	})
	m := next(t, openStream(t, c, "/s"), func(m StreamMsg) bool { return m.State == Retrying })
	if !errors.Is(m.Err, errSilent) {
		t.Fatalf("retry error %v", m.Err)
	}
}

// Pings are comments: they deliver no event, but they keep a quiet stream
// alive past the idle timeout.
func TestStreamPingsKeepAQuietStreamAlive(t *testing.T) {
	fastStreams(t)
	c := serve(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: hello\n\n")
		w.(http.Flusher).Flush()
		tick := time.NewTicker(idleTimeout / 6)
		defer tick.Stop()
		for {
			select {
			case <-r.Context().Done():
				return
			case <-tick.C:
				fmt.Fprint(w, ": ping\n\n")
				w.(http.Flusher).Flush()
			}
		}
	})
	ch := openStream(t, c, "/s")
	next(t, ch, func(m StreamMsg) bool { return m.Event != nil })
	// Three idle timeouts of pings only: nothing may arrive but the absence
	// of a Retrying (or any further event).
	quiet := time.After(3 * idleTimeout)
	for {
		select {
		case m := <-ch:
			t.Fatalf("a pinged stream produced %+v", m)
		case <-quiet:
			return
		}
	}
}

// A removed target does not retry forever: a 404 ends the stream.
func TestStreamStopsOnAClientError(t *testing.T) {
	fastStreams(t)
	c := serve(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(404)
		fmt.Fprint(w, `{"error":"target not found","code":"target_not_found"}`)
	})
	ch := openStream(t, c, "/s")
	m := next(t, ch, func(m StreamMsg) bool { return m.State == Failed })
	var e *api.Error
	if !errors.As(m.Err, &e) || e.Code != api.CodeTargetNotFound {
		t.Fatalf("failed with %v", m.Err)
	}
	if _, ok := <-ch; ok {
		t.Fatal("the channel stays open after Failed")
	}
}

// 408 and 429 are 4xx a retry can fix: they do not end the stream.
func TestStreamRetriesATimeoutOrRateLimit(t *testing.T) {
	fastStreams(t)
	for _, status := range []int{http.StatusRequestTimeout, http.StatusTooManyRequests, http.StatusServiceUnavailable} {
		c := serve(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(status)
		})
		ch := openStream(t, c, "/s")
		m := next(t, ch, func(m StreamMsg) bool { return m.State == Retrying || m.State == Failed })
		var e *api.Error
		if m.State != Retrying || !errors.As(m.Err, &e) || e.Status != status {
			t.Errorf("%d: %v %v", status, m.State, m.Err)
		}
	}
}

// settled waits for the goroutine count to come back to base. Other tests'
// leftovers can only shrink meanwhile; none of this package's tests run in
// parallel.
func settled(t *testing.T, base int) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for runtime.NumGoroutine() > base {
		if time.Now().After(deadline) {
			buf := make([]byte, 1<<20)
			t.Fatalf("%d goroutines, want at most %d:\n%s", runtime.NumGoroutine(), base, buf[:runtime.Stack(buf, true)])
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// Cancelling a stream ends every goroutine it started, whether it was live,
// waiting out a backoff, or stuck connecting; and a watch on top of it ends
// with it.
func TestStreamLeavesNoGoroutinesBehind(t *testing.T) {
	fastStreams(t)
	http.DefaultTransport.(*http.Transport).CloseIdleConnections()
	base := runtime.NumGoroutine()
	var n atomic.Int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintf(w, "data: %d\n\n", n.Add(1))
		w.(http.Flusher).Flush()
		if r.URL.Path == "/live" {
			<-r.Context().Done()
		}
	}))
	c := newHTTP(ts.URL, "tok", buildinfo.Version())

	ctx, cancel := context.WithCancel(context.Background())
	live := c.Stream(ctx, "/live")
	next(t, live, func(m StreamMsg) bool { return m.Event != nil })
	dropping := c.Stream(ctx, "/drop")
	next(t, dropping, func(m StreamMsg) bool { return m.State == Retrying })
	watched := watch(ctx, c, "/live", decodeJSON[int])
	for u := range watched {
		if u.Has {
			break
		}
	}
	cancel()
	for range live {
	}
	for range dropping {
	}
	for range watched {
	}

	// The server going away while a stream retries ends nothing by itself;
	// the cancel still does.
	ctx2, cancel2 := context.WithCancel(context.Background())
	orphan := c.Stream(ctx2, "/drop")
	next(t, orphan, func(m StreamMsg) bool { return m.Event != nil })
	ts.Close()
	next(t, orphan, func(m StreamMsg) bool { return m.State == Retrying && errors.Is(m.Err, ErrServerUnreachable) })
	cancel2()
	for range orphan {
	}

	http.DefaultTransport.(*http.Transport).CloseIdleConnections()
	settled(t, base)
}

func TestDecodeJSONRefusesANamedEvent(t *testing.T) {
	if v, reset, err := decodeJSON[int](Event{Data: []byte("7")}); v != 7 || reset || err != nil {
		t.Fatalf("decodeJSON = %v %v %v", v, reset, err)
	}
	if _, _, err := decodeJSON[int](Event{Name: "reset", Data: []byte("7")}); err == nil {
		t.Fatal("a named event decoded as a default one")
	}
}

func TestWatchLogsResetsThenAppends(t *testing.T) {
	c := serve(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("backlog") != "50" {
			http.Error(w, "no backlog", 400)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "event: reset\ndata: [{\"unit\":\"u\",\"line\":\"a\"},{\"unit\":\"u\",\"line\":\"b\"}]\n\n")
		fmt.Fprint(w, "data: {\"unit\":\"u\",\"line\":\"c\"}\n\n")
		fmt.Fprint(w, "event: error\ndata: {\"error\":\"agent busy\",\"code\":\"rejected\"}\n\n")
		fmt.Fprint(w, "event: note\ndata: \"snapshot mode\"\n\n")
		w.(http.Flusher).Flush()
		<-r.Context().Done()
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var got []Update[[]api.LogHit]
	for u := range c.WatchLogs(ctx, "box", 50) {
		if u.Has || u.Err != nil || u.Note != "" {
			got = append(got, u)
		}
		if len(got) == 4 {
			break
		}
	}
	if !got[0].Reset || len(got[0].Value) != 2 || got[1].Reset || got[1].Value[0].Line != "c" {
		t.Fatalf("updates %+v", got)
	}
	var e *api.Error
	if got[2].Has || !errors.As(got[2].Err, &e) || e.Message != "agent busy" {
		t.Fatalf("error event became %+v", got[2])
	}
	if got[3].Note != "snapshot mode" || got[3].Has {
		t.Fatalf("note event became %+v", got[3])
	}
}

// rotating is a server whose session token the test can change, the way a
// restarted jumpgate server mints a new one.
func rotating(t *testing.T) (*httptest.Server, *atomic.Value, *atomic.Int32) {
	t.Helper()
	var token atomic.Value
	token.Store("old")
	var n atomic.Int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+token.Load().(string) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprint(w, `{"error":"unauthorized","code":"unauthorized"}`)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintf(w, "data: %d\n\n", n.Add(1))
		w.(http.Flusher).Flush()
	}))
	t.Cleanup(ts.Close)
	return ts, &token, &n
}

// Review fix: a restarted server has a new token. The stream's first 401
// after a drop makes the client re-read server.json and carry on with the
// new token, instead of failing for good.
func TestStreamPicksUpARestartedServersToken(t *testing.T) {
	fastStreams(t)
	ts, token, _ := rotating(t)
	c := newHTTP(ts.URL, "old", buildinfo.Version())
	var looked atomic.Int32
	c.rediscover = func(context.Context) (*Client, error) {
		looked.Add(1)
		return newHTTP(ts.URL, token.Load().(string), buildinfo.Version()), nil
	}
	ch := openStream(t, c, "/s")
	next(t, ch, func(m StreamMsg) bool { return m.Event != nil })
	token.Store("new")
	m := next(t, ch, func(m StreamMsg) bool { return m.Event != nil || m.State == Failed })
	if m.State == Failed || looked.Load() == 0 {
		t.Fatalf("after the token rotated: %v %v (looked %d)", m.State, m.Err, looked.Load())
	}
	if got := c.Info().Token; got != "new" {
		t.Fatalf("client token %q", got)
	}
	// And it keeps working on later reconnects with the new token.
	next(t, ch, func(m StreamMsg) bool { return m.Event != nil })
}

// A 401 that a fresh server.json does not fix (same token, or no way to
// look) is final, as before: the client looks once, not in a loop.
func TestStreamFailsOnA401TheTokenDoesNotFix(t *testing.T) {
	fastStreams(t)
	ts, token, _ := rotating(t)
	token.Store("other")
	c := newHTTP(ts.URL, "old", buildinfo.Version())
	var looked atomic.Int32
	c.rediscover = func(context.Context) (*Client, error) {
		looked.Add(1)
		return newHTTP(ts.URL, "old", buildinfo.Version()), nil
	}
	m := next(t, openStream(t, c, "/s"), func(m StreamMsg) bool { return m.State == Failed })
	var e *api.Error
	if !errors.As(m.Err, &e) || e.Status != http.StatusUnauthorized || looked.Load() != 1 {
		t.Fatalf("failed with %v after %d lookups", m.Err, looked.Load())
	}

	plain := newHTTP(ts.URL, "old", buildinfo.Version()) // no rediscover
	m = next(t, openStream(t, plain, "/s"), func(m StreamMsg) bool { return m.State == Failed })
	if !errors.As(m.Err, &e) || e.Status != http.StatusUnauthorized {
		t.Fatalf("without rediscover: %v", m.Err)
	}
}

// Two streams on one client both get a 401 with the old token: one swaps
// the token in, and the other must see that and retry with it rather than
// fail because a fresh server.json shows nothing new. The server holds the
// first two stale-token answers until both have arrived, so both 401s
// really carry the old token.
func TestConcurrentStreamsBothSurviveATokenRotation(t *testing.T) {
	fastStreams(t)
	var token atomic.Value
	token.Store("old")
	var stale atomic.Int32
	both := make(chan struct{})
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+token.Load().(string) {
			if stale.Add(1) == 2 {
				close(both)
			}
			select {
			case <-both:
			case <-time.After(2 * time.Second):
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprint(w, `{"error":"unauthorized","code":"unauthorized"}`)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintf(w, "data: %s\n\n", token.Load()) // which token let it in
		w.(http.Flusher).Flush()
	}))
	t.Cleanup(ts.Close)
	c := newHTTP(ts.URL, "old", buildinfo.Version())
	c.rediscover = func(context.Context) (*Client, error) {
		return newHTTP(ts.URL, token.Load().(string), buildinfo.Version()), nil
	}
	// Each connection ends after one event, so both streams keep
	// reconnecting and both meet the rotation.
	a, b := openStream(t, c, "/s?a"), openStream(t, c, "/s?b")
	next(t, a, func(m StreamMsg) bool { return m.Event != nil })
	next(t, b, func(m StreamMsg) bool { return m.Event != nil })
	token.Store("new")
	for name, ch := range map[string]<-chan StreamMsg{"a": a, "b": b} {
		m := next(t, ch, func(m StreamMsg) bool { return (m.Event != nil && string(m.Event.Data) == "new") || m.State == Failed })
		if m.State == Failed {
			t.Fatalf("stream %s failed after the rotation: %v", name, m.Err)
		}
	}
}
