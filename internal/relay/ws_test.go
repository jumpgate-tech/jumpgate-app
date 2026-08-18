package relay

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/valve-tech/valve-node-app/internal/wsrpc"
)

// The relay terminates the customer's WebSocket and speaks plain HTTP to every
// upstream. That is a deliberate exception to "delegate, do not rebuild": it
// widens the upstream pool to every HTTP-only node and collapses N subscribers
// into one poll loop. These tests cover the half that is the relay's own —
// frame policy, call translation, and the subscription registry.

// stubCaller answers a JSON-RPC call the way eRPC would, and records what it saw.
type stubCaller struct {
	mu     sync.Mutex
	bodies []string
	reply  string
	err    error
}

func (s *stubCaller) Call(_ context.Context, _ int, body []byte) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.bodies = append(s.bodies, string(body))
	if s.err != nil {
		return nil, s.err
	}
	reply := s.reply
	if reply == "" {
		reply = `{"jsonrpc":"2.0","id":1,"result":"0x1"}`
	}
	return []byte(reply), nil
}

func (s *stubCaller) seen() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.bodies...)
}

// stubStreams stands in for the head pollers. It records what was subscribed and
// can push a notification on demand.
type stubStreams struct {
	mu      sync.Mutex
	kinds   []string
	pushes  map[string]func(json.RawMessage)
	stopped []string
}

func newStubStreams() *stubStreams {
	return &stubStreams{pushes: map[string]func(json.RawMessage){}}
}

func (s *stubStreams) Subscribe(_ context.Context, _ int, kind string, _ json.RawMessage, notify func(json.RawMessage)) (StreamHandle, error) {
	if !SupportedSubscription(kind) {
		return nil, ErrSubscriptionUnsupported
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.kinds = append(s.kinds, kind)
	s.pushes[kind] = notify
	return stubHandle{s: s, kind: kind}, nil
}

func (s *stubStreams) push(kind string, payload string) {
	s.mu.Lock()
	notify := s.pushes[kind]
	s.mu.Unlock()
	if notify != nil {
		notify(json.RawMessage(payload))
	}
}

type stubHandle struct {
	s    *stubStreams
	kind string
}

func (h stubHandle) Close() error {
	h.s.mu.Lock()
	defer h.s.mu.Unlock()
	h.s.stopped = append(h.s.stopped, h.kind)
	return nil
}

// wsHarness stands up a real server that terminates a WebSocket, and dials it
// with the package's own client.
type wsHarness struct {
	conn    *wsrpc.Conn
	caller  *stubCaller
	streams *stubStreams
	done    chan struct{}
}

func newWSHarness(t *testing.T, rec KeyRecord) *wsHarness {
	t.Helper()
	h := &wsHarness{caller: &stubCaller{}, streams: newStubStreams(), done: make(chan struct{})}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := wsrpc.Accept(w, r, nil)
		if err != nil {
			return
		}
		sess := NewWSSession(WSConfig{
			Conn:    conn,
			Record:  rec,
			ChainID: 369,
			Caller:  h.caller,
			Streams: h.streams,
		})
		sess.Run(r.Context())
		close(h.done)
	}))
	t.Cleanup(srv.Close)

	client, err := wsrpc.Dial(context.Background(), "ws"+strings.TrimPrefix(srv.URL, "http"), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { client.Close() })
	h.conn = client
	return h
}

func (h *wsHarness) send(t *testing.T, msg string) {
	t.Helper()
	if err := h.conn.WriteText([]byte(msg)); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func (h *wsHarness) read(t *testing.T) map[string]any {
	t.Helper()
	h.conn.SetDeadline(time.Now().Add(3 * time.Second))
	raw, err := h.conn.ReadMessage()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("decode %q: %v", raw, err)
	}
	return out
}

// An ordinary call arriving as a WebSocket frame becomes an HTTP POST upstream.
// This is the part that needs no caveat, and it is what makes upstream
// WebSocket support irrelevant.
func TestWSForwardsAnOrdinaryCallOverHTTP(t *testing.T) {
	h := newWSHarness(t, enabledKey())
	h.send(t, `{"jsonrpc":"2.0","id":1,"method":"eth_blockNumber","params":[]}`)

	got := h.read(t)
	if got["result"] != "0x1" {
		t.Errorf("result = %v, want 0x1", got["result"])
	}
	seen := h.caller.seen()
	if len(seen) != 1 || !strings.Contains(seen[0], "eth_blockNumber") {
		t.Errorf("upstream saw %v, want one eth_blockNumber", seen)
	}
}

// Policy applies per frame, not just at the handshake. This is the whole reason
// the relay parses the stream rather than proxying it: allow_trace and the
// method lists cannot act on an upgrade, which carries no method.
func TestWSAppliesMethodPolicyPerFrame(t *testing.T) {
	rec := KeyRecord{ID: "k1", Enabled: true, MethodBlock: []string{"eth_sendRawTransaction"}}
	h := newWSHarness(t, rec)

	h.send(t, `{"jsonrpc":"2.0","id":1,"method":"eth_sendRawTransaction","params":["0x0"]}`)
	got := h.read(t)
	if got["error"] == nil {
		t.Fatalf("got %v, want a JSON-RPC error", got)
	}
	if len(h.caller.seen()) != 0 {
		t.Error("a denied frame still reached the upstream")
	}

	// The connection survives a refusal, so one bad call does not drop a
	// customer's whole session.
	h.send(t, `{"jsonrpc":"2.0","id":2,"method":"eth_blockNumber"}`)
	if next := h.read(t); next["result"] != "0x1" {
		t.Errorf("connection did not survive a refusal: %v", next)
	}
}

// The trace gate works on the stream too.
func TestWSGatesTraceOverTheStream(t *testing.T) {
	rec := KeyRecord{ID: "k1", Enabled: true, AllowTrace: false}
	h := newWSHarness(t, rec)

	h.send(t, `{"jsonrpc":"2.0","id":1,"method":"debug_traceTransaction","params":["0x0"]}`)
	if got := h.read(t); got["error"] == nil {
		t.Fatalf("got %v, want a JSON-RPC error", got)
	}
	if len(h.caller.seen()) != 0 {
		t.Error("a trace call reached the upstream with allow_trace false")
	}
}

func TestWSSubscribeReturnsAnID(t *testing.T) {
	h := newWSHarness(t, enabledKey())
	h.send(t, `{"jsonrpc":"2.0","id":1,"method":"eth_subscribe","params":["newHeads"]}`)

	got := h.read(t)
	id, ok := got["result"].(string)
	if !ok || id == "" {
		t.Fatalf("got %v, want a subscription id", got)
	}
	if len(h.streams.kinds) != 1 || h.streams.kinds[0] != "newHeads" {
		t.Errorf("streams saw %v, want [newHeads]", h.streams.kinds)
	}
}

// A notification carries the subscription id the client was given, in the shape
// a native node uses, so an unmodified client library works.
func TestWSDeliversANotification(t *testing.T) {
	h := newWSHarness(t, enabledKey())
	h.send(t, `{"jsonrpc":"2.0","id":1,"method":"eth_subscribe","params":["newHeads"]}`)
	ack := h.read(t)
	id := ack["result"].(string)

	h.streams.push("newHeads", `{"number":"0x10"}`)

	note := h.read(t)
	if note["method"] != "eth_subscription" {
		t.Errorf("method = %v, want eth_subscription", note["method"])
	}
	params, ok := note["params"].(map[string]any)
	if !ok {
		t.Fatalf("params = %v, want an object", note["params"])
	}
	if params["subscription"] != id {
		t.Errorf("subscription = %v, want %v", params["subscription"], id)
	}
}

// newPendingTransactions has no honest HTTP polling equivalent. The relay says
// so plainly rather than opening a stream that never fires.
func TestWSRefusesNewPendingTransactions(t *testing.T) {
	h := newWSHarness(t, enabledKey())
	h.send(t, `{"jsonrpc":"2.0","id":1,"method":"eth_subscribe","params":["newPendingTransactions"]}`)

	got := h.read(t)
	errObj, ok := got["error"].(map[string]any)
	if !ok {
		t.Fatalf("got %v, want a JSON-RPC error", got)
	}
	msg, _ := errObj["message"].(string)
	if !strings.Contains(strings.ToLower(msg), "not supported") {
		t.Errorf("message = %q, want it to say the subscription is not supported", msg)
	}
}

func TestWSUnsubscribeStopsTheStream(t *testing.T) {
	h := newWSHarness(t, enabledKey())
	h.send(t, `{"jsonrpc":"2.0","id":1,"method":"eth_subscribe","params":["newHeads"]}`)
	id := h.read(t)["result"].(string)

	h.send(t, `{"jsonrpc":"2.0","id":2,"method":"eth_unsubscribe","params":["`+id+`"]}`)
	got := h.read(t)
	if got["result"] != true {
		t.Errorf("result = %v, want true", got["result"])
	}
	if len(h.streams.stopped) != 1 {
		t.Errorf("stopped = %v, want the stream closed", h.streams.stopped)
	}
}

// Unsubscribing from an id the caller never held returns false rather than
// tearing down someone else's stream.
func TestWSUnsubscribeUnknownIDIsFalse(t *testing.T) {
	h := newWSHarness(t, enabledKey())
	h.send(t, `{"jsonrpc":"2.0","id":1,"method":"eth_unsubscribe","params":["0xdeadbeef"]}`)

	if got := h.read(t); got["result"] != false {
		t.Errorf("result = %v, want false", got["result"])
	}
}

// A closed connection must release every stream it opened. The relay is
// stateful now, so a leak here is a leak per disconnected customer.
func TestWSClosingReleasesEverySubscription(t *testing.T) {
	h := newWSHarness(t, enabledKey())
	h.send(t, `{"jsonrpc":"2.0","id":1,"method":"eth_subscribe","params":["newHeads"]}`)
	h.read(t)

	h.conn.Close()
	select {
	case <-h.done:
	case <-time.After(3 * time.Second):
		t.Fatal("the session did not finish after the client went away")
	}
	if len(h.streams.stopped) != 1 {
		t.Errorf("stopped = %v, want the stream released on disconnect", h.streams.stopped)
	}
}

// A frame the relay cannot read is answered with an error, not forwarded.
func TestWSRefusesAGarbageFrame(t *testing.T) {
	h := newWSHarness(t, enabledKey())
	h.send(t, `not json`)

	if got := h.read(t); got["error"] == nil {
		t.Fatalf("got %v, want a JSON-RPC error", got)
	}
	if len(h.caller.seen()) != 0 {
		t.Error("a garbage frame reached the upstream")
	}
}
