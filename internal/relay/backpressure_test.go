package relay

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// One poll loop serves every subscriber on a chain. That is the economic
// argument for terminating WebSocket at the relay, and it is also a hazard: if
// the loop waits on any one subscriber, a single client that stops reading
// silences the chain for everyone. These tests hold the relay to "a slow
// subscriber can only slow itself".

// A subscriber whose delivery never returns must not stall the others. The
// stuck notify stands in for a write to a client that has stopped reading.
func TestStreamsASubscriberThatNeverReturnsDoesNotStallTheOthers(t *testing.T) {
	caller := newScriptedCaller()
	caller.advance()
	streams := NewPollerStreams(caller, 10*time.Millisecond)
	release := make(chan struct{})
	// Cleanups run last-registered first, so the stuck subscriber is released
	// before Stop waits on the loop.
	t.Cleanup(streams.Stop)
	t.Cleanup(func() { close(release) })

	stuck, err := streams.Subscribe(context.Background(), 369, "newHeads", nil,
		func(json.RawMessage) { <-release }, nil)
	if err != nil {
		t.Fatalf("subscribe stuck: %v", err)
	}
	defer stuck.Close()

	var got atomic.Int64
	h, err := streams.Subscribe(context.Background(), 369, "newHeads", nil,
		func(json.RawMessage) { got.Add(1) }, nil)
	if err != nil {
		t.Fatalf("subscribe healthy: %v", err)
	}
	defer h.Close()

	keepAdvancing(t, caller)
	waitFor(t, func() bool { return got.Load() >= 5 })
}

// A subscriber that falls a whole queue behind is dropped, and told so, rather
// than silently losing events. The session it belongs to ends the connection,
// and the client reconnects knowing it may have missed something.
func TestStreamsDropASubscriberThatFallsAQueueBehind(t *testing.T) {
	caller := newScriptedCaller()
	caller.advance()
	streams := NewPollerStreams(caller, 10*time.Millisecond)
	streams.queueLen = 2
	release := make(chan struct{})
	t.Cleanup(streams.Stop)
	t.Cleanup(func() { close(release) })

	lost := make(chan struct{}, 1)
	stuck, err := streams.Subscribe(context.Background(), 369, "newHeads", nil,
		func(json.RawMessage) { <-release },
		func() { lost <- struct{}{} })
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	defer stuck.Close()

	keepAdvancing(t, caller)
	select {
	case <-lost:
	case <-time.After(3 * time.Second):
		t.Fatal("the subscriber overflowed its queue and was never told it was dropped")
	}
}

// Closing a handle must not wait on a delivery that is stuck. detach runs on
// the session's teardown path, and a teardown that hangs behind the very write
// that killed the session would leak the session.
func TestStreamsCloseDoesNotWaitForAStuckDelivery(t *testing.T) {
	caller := newScriptedCaller()
	caller.advance()
	streams := NewPollerStreams(caller, 10*time.Millisecond)
	release := make(chan struct{})
	t.Cleanup(streams.Stop)
	t.Cleanup(func() { close(release) })

	entered := make(chan struct{}, 1)
	h, err := streams.Subscribe(context.Background(), 369, "newHeads", nil, func(json.RawMessage) {
		select {
		case entered <- struct{}{}:
		default:
		}
		<-release
	}, nil)
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}

	keepAdvancing(t, caller)
	select {
	case <-entered:
	case <-time.After(3 * time.Second):
		t.Fatal("no delivery started")
	}

	closed := make(chan struct{})
	go func() { h.Close(); close(closed) }()
	select {
	case <-closed:
	case <-time.After(2 * time.Second):
		t.Fatal("Close hung behind a delivery that never returns")
	}
}

// A client that stops reading must not keep its session forever. The write
// timeout fails the stuck write, and the session ends and releases its streams.
func TestWSSessionEndsWhenTheClientStopsReading(t *testing.T) {
	h := newWSHarnessWith(t, enabledKey(), func(cfg *WSConfig) {
		cfg.WriteTimeout = 50 * time.Millisecond
	})
	h.send(t, `{"jsonrpc":"2.0","id":1,"method":"eth_subscribe","params":["newHeads"]}`)
	h.read(t)

	// The client never reads again. Large payloads fill the socket buffers
	// quickly, after which a write can only wait on the client.
	big := `{"pad":"` + strings.Repeat("x", 64<<10) + `"}`
	stop := make(chan struct{})
	defer close(stop)
	go func() {
		for {
			select {
			case <-stop:
				return
			case <-h.done:
				return
			default:
				h.streams.push("newHeads", big)
			}
		}
	}()

	select {
	case <-h.done:
	case <-time.After(5 * time.Second):
		t.Fatal("the session is still blocked writing to a client that stopped reading")
	}
}

// A client that vanishes without closing must be reaped. Without an idle
// timeout its read never returns and every stream it holds polls on forever.
func TestWSSessionReapsAnIdleClient(t *testing.T) {
	h := newWSHarnessWith(t, enabledKey(), func(cfg *WSConfig) {
		cfg.IdleTimeout = 50 * time.Millisecond
		cfg.PingInterval = time.Hour
	})

	select {
	case <-h.done:
	case <-time.After(3 * time.Second):
		t.Fatal("a silent client was never reaped")
	}
}

// A client that only listens sends no data, so the idle timeout alone would
// reap every quiet subscriber. The session pings, and the client's pong keeps
// it alive for three idle windows here.
func TestWSSessionKeepsAListenOnlyClientThatAnswersPings(t *testing.T) {
	h := newWSHarnessWith(t, enabledKey(), func(cfg *WSConfig) {
		cfg.IdleTimeout = 50 * time.Millisecond
		cfg.PingInterval = 10 * time.Millisecond
	})

	// The client reads, and so answers pings, but sends nothing of its own.
	h.conn.SetDeadline(time.Now().Add(150 * time.Millisecond))
	if _, err := h.conn.ReadMessage(); !isTimeout(err) {
		t.Fatalf("read = %v, want the client's own deadline with nothing to read", err)
	}
	select {
	case <-h.done:
		t.Fatal("a client answering pings was reaped as idle")
	default:
	}

	// The deadline above covered writes too, so lift it before sending.
	h.conn.SetDeadline(time.Now().Add(3 * time.Second))
	h.send(t, `{"jsonrpc":"2.0","id":1,"method":"eth_blockNumber","params":[]}`)
	if got := h.read(t); got["result"] != "0x1" {
		t.Errorf("got %v, want the session still serving", got)
	}
}

// A subscriber must never be attached to a loop that detach has already
// cancelled. Subscribe finds the loop under one lock and attaches under
// another, and the last subscriber leaving in between used to cancel the loop
// under the newcomer, which then got a success reply and an id and never
// received an event. The hook parks Subscribe in exactly that window.
func TestStreamsSubscribeNeverAttachesToACancelledLoop(t *testing.T) {
	caller := newScriptedCaller()
	caller.advance()
	streams := NewPollerStreams(caller, 10*time.Millisecond)
	t.Cleanup(streams.Stop)

	first, err := streams.Subscribe(context.Background(), 369, "newHeads", nil, func(json.RawMessage) {}, nil)
	if err != nil {
		t.Fatalf("subscribe first: %v", err)
	}
	var once sync.Once
	streams.beforeAttach = func() { once.Do(func() { first.Close() }) }

	var got atomic.Int64
	h, err := streams.Subscribe(context.Background(), 369, "newHeads", nil,
		func(json.RawMessage) { got.Add(1) }, nil)
	if err != nil {
		t.Fatalf("subscribe second: %v", err)
	}
	defer h.Close()

	if n := streams.LoopCount(); n != 1 {
		t.Fatalf("poll loops = %d, want 1 serving the subscriber that just attached", n)
	}
	keepAdvancing(t, caller)
	waitFor(t, func() bool { return got.Load() > 0 })
}

// When a stream drops a subscriber for falling behind, the session ends the
// connection. Leaving it open would be a subscription that silently stopped.
func TestWSSessionEndsWhenAStreamDropsIt(t *testing.T) {
	h := newWSHarness(t, enabledKey())
	h.send(t, `{"jsonrpc":"2.0","id":1,"method":"eth_subscribe","params":["newHeads"]}`)
	h.read(t)

	h.streams.drop("newHeads")
	select {
	case <-h.done:
	case <-time.After(3 * time.Second):
		t.Fatal("the session stayed open after its stream dropped it")
	}
}
