package relay

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// Two things are proved here. First, the relay can read a chain's head with
// nothing but ordinary HTTP calls — that is what makes upstream WebSocket
// support irrelevant. Second, every subscriber on a chain shares ONE poll loop.
// The second claim is the whole economic argument for terminating WebSocket at
// the relay, so it gets a test rather than a comment.

// scriptedCaller answers JSON-RPC calls from a script and counts them.
type scriptedCaller struct {
	mu     sync.Mutex
	calls  atomic.Int64
	head   uint64
	blocks map[uint64]string
}

func newScriptedCaller() *scriptedCaller {
	return &scriptedCaller{blocks: map[uint64]string{0: "0x00"}}
}

func (s *scriptedCaller) advance() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.head++
	s.blocks[s.head] = fmt.Sprintf("0xaa%d", s.head)
}

func (s *scriptedCaller) Call(_ context.Context, _ int, body []byte) ([]byte, error) {
	s.calls.Add(1)
	var req struct {
		Method string            `json:"method"`
		Params []json.RawMessage `json:"params"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	switch req.Method {
	case "eth_blockNumber":
		return []byte(fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"result":"0x%x"}`, s.head)), nil
	case "eth_getBlockByNumber":
		var tag string
		json.Unmarshal(req.Params[0], &tag)
		var n uint64
		fmt.Sscanf(tag, "0x%x", &n)
		hash, ok := s.blocks[n]
		if !ok {
			return []byte(`{"jsonrpc":"2.0","id":1,"result":null}`), nil
		}
		parent := s.blocks[n-1]
		return []byte(fmt.Sprintf(
			`{"jsonrpc":"2.0","id":1,"result":{"number":"0x%x","hash":%q,"parentHash":%q}}`,
			n, hash, parent)), nil
	}
	return nil, fmt.Errorf("unexpected method %q", req.Method)
}

func TestRPCBlockFetcherReadsTheHead(t *testing.T) {
	caller := newScriptedCaller()
	for i := 0; i < 26; i++ {
		caller.advance()
	}
	f := NewRPCBlockFetcher(caller, 369)

	got, err := f.HeadNumber(context.Background())
	if err != nil {
		t.Fatalf("HeadNumber: %v", err)
	}
	if got != 26 {
		t.Errorf("head = %d, want 26 — the hex result was misread", got)
	}
}

func TestRPCBlockFetcherReadsABlock(t *testing.T) {
	caller := newScriptedCaller()
	caller.advance()
	caller.advance()
	f := NewRPCBlockFetcher(caller, 369)

	got, err := f.BlockByNumber(context.Background(), 2)
	if err != nil {
		t.Fatalf("BlockByNumber: %v", err)
	}
	if got.Number != 2 {
		t.Errorf("number = %d, want 2", got.Number)
	}
	if got.Hash != "0xaa2" {
		t.Errorf("hash = %q, want 0xaa2", got.Hash)
	}
	if got.ParentHash != "0xaa1" {
		t.Errorf("parentHash = %q, want 0xaa1", got.ParentHash)
	}
}

// A JSON-RPC error is an error. Reading it as a zero head would look like a
// chain stuck at genesis and stall every subscriber silently.
func TestRPCBlockFetcherSurfacesAJSONRPCError(t *testing.T) {
	f := NewRPCBlockFetcher(callerFunc(func(context.Context, int, []byte) ([]byte, error) {
		return []byte(`{"jsonrpc":"2.0","id":1,"error":{"code":-32000,"message":"nope"}}`), nil
	}), 369)

	if _, err := f.HeadNumber(context.Background()); err == nil {
		t.Fatal("err = nil, want the JSON-RPC error surfaced")
	}
}

// A missing block is an error rather than a zero-valued BlockRef, which would
// look to the poller like a reorg to genesis.
func TestRPCBlockFetcherRejectsANullBlock(t *testing.T) {
	f := NewRPCBlockFetcher(callerFunc(func(context.Context, int, []byte) ([]byte, error) {
		return []byte(`{"jsonrpc":"2.0","id":1,"result":null}`), nil
	}), 369)

	if _, err := f.BlockByNumber(context.Background(), 5); err == nil {
		t.Fatal("err = nil, want an error for a null block")
	}
}

type callerFunc func(context.Context, int, []byte) ([]byte, error)

func (f callerFunc) Call(ctx context.Context, chainID int, body []byte) ([]byte, error) {
	return f(ctx, chainID, body)
}

// THE claim. Two subscribers on one chain must cost one poll loop, not two.
// Without this, terminating WebSocket would multiply upstream load by the
// subscriber count instead of collapsing it.
func TestStreamsShareOnePollLoopPerChain(t *testing.T) {
	caller := newScriptedCaller()
	caller.advance()
	streams := NewPollerStreams(caller, 20*time.Millisecond)
	t.Cleanup(streams.Stop)

	var a, b atomic.Int64
	h1, err := streams.Subscribe(context.Background(), 369, "newHeads", nil, func(json.RawMessage) { a.Add(1) })
	if err != nil {
		t.Fatalf("subscribe 1: %v", err)
	}
	defer h1.Close()
	h2, err := streams.Subscribe(context.Background(), 369, "newHeads", nil, func(json.RawMessage) { b.Add(1) })
	if err != nil {
		t.Fatalf("subscribe 2: %v", err)
	}
	defer h2.Close()

	if got := streams.LoopCount(); got != 1 {
		t.Fatalf("poll loops = %d, want 1 for two subscribers on one chain", got)
	}

	// Both subscribers must receive heads from that one loop.
	keepAdvancing(t, caller)
	waitFor(t, func() bool { return a.Load() > 0 && b.Load() > 0 })
}

// A different chain gets its own loop, because a poller tracks one chain's head.
func TestStreamsRunOneLoopPerChain(t *testing.T) {
	caller := newScriptedCaller()
	caller.advance()
	streams := NewPollerStreams(caller, 50*time.Millisecond)
	t.Cleanup(streams.Stop)

	h1, _ := streams.Subscribe(context.Background(), 1, "newHeads", nil, func(json.RawMessage) {})
	h2, _ := streams.Subscribe(context.Background(), 369, "newHeads", nil, func(json.RawMessage) {})
	defer h1.Close()
	defer h2.Close()

	if got := streams.LoopCount(); got != 2 {
		t.Errorf("poll loops = %d, want 2 for two chains", got)
	}
}

// The last subscriber leaving stops the loop. Otherwise a chain nobody watches
// keeps calling an upstream forever, which on a paid upstream is a bill.
func TestStreamsStopTheLoopWhenTheLastSubscriberLeaves(t *testing.T) {
	caller := newScriptedCaller()
	caller.advance()
	streams := NewPollerStreams(caller, 20*time.Millisecond)
	t.Cleanup(streams.Stop)

	h1, _ := streams.Subscribe(context.Background(), 369, "newHeads", nil, func(json.RawMessage) {})
	h2, _ := streams.Subscribe(context.Background(), 369, "newHeads", nil, func(json.RawMessage) {})

	h1.Close()
	if got := streams.LoopCount(); got != 1 {
		t.Errorf("poll loops = %d after one of two left, want 1", got)
	}

	h2.Close()
	waitFor(t, func() bool { return streams.LoopCount() == 0 })

	before := caller.calls.Load()
	time.Sleep(100 * time.Millisecond)
	if after := caller.calls.Load(); after != before {
		t.Errorf("the upstream was called %d more times after every subscriber left", after-before)
	}
}

// An unsupported kind is refused by the streams too, not only by the session,
// so a caller cannot reach a poller that has nothing to poll.
func TestStreamsRefuseAnUnsupportedKind(t *testing.T) {
	streams := NewPollerStreams(newScriptedCaller(), time.Second)
	t.Cleanup(streams.Stop)

	_, err := streams.Subscribe(context.Background(), 369, "newPendingTransactions", nil, func(json.RawMessage) {})
	if err == nil {
		t.Fatal("err = nil, want the subscription refused")
	}
	if !strings.Contains(err.Error(), "not supported") {
		t.Errorf("err = %v, want it to say the kind is not supported", err)
	}
}

// A notification carries the block, so a client library sees what a node would
// have sent it.
func TestStreamsDeliverTheBlock(t *testing.T) {
	caller := newScriptedCaller()
	caller.advance()
	streams := NewPollerStreams(caller, 20*time.Millisecond)
	t.Cleanup(streams.Stop)

	got := make(chan json.RawMessage, 4)
	h, err := streams.Subscribe(context.Background(), 369, "newHeads", nil, func(m json.RawMessage) {
		select {
		case got <- m:
		default:
		}
	})
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	defer h.Close()

	keepAdvancing(t, caller)
	select {
	case m := <-got:
		var head map[string]any
		if err := json.Unmarshal(m, &head); err != nil {
			t.Fatalf("decode %q: %v", m, err)
		}
		if head["hash"] == nil || head["number"] == nil {
			t.Errorf("notification = %v, want a block with a hash and a number", head)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("no notification arrived")
	}
}

// keepAdvancing produces blocks until the test finishes. A poller primes on its
// first tick and then reports what comes AFTER — which is what a newHeads
// subscriber wants — so a single advance racing the first tick may be swallowed
// by design. A real chain keeps producing, and so does this.
func keepAdvancing(t *testing.T, c *scriptedCaller) {
	t.Helper()
	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			select {
			case <-stop:
				return
			case <-time.After(15 * time.Millisecond):
				c.advance()
			}
		}
	}()
	t.Cleanup(func() { close(stop); <-done })
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("condition never became true")
}

// The eRPC caller must address the same path the proxy path builds, or a
// subscription would poll a chain the gateway does not serve.
func TestERPCCallerPostsToTheChainPath(t *testing.T) {
	var got capturedRequest
	up := stubUpstream(t, &got)

	c := NewERPCCaller(up, "main")
	body := []byte(`{"jsonrpc":"2.0","id":1,"method":"eth_blockNumber","params":[]}`)
	reply, err := c.Call(context.Background(), 369, body)
	if err != nil {
		t.Fatalf("Call: %v", err)
	}
	if got.path != "/main/evm/369" {
		t.Errorf("path = %q, want /main/evm/369", got.path)
	}
	if got.body != string(body) {
		t.Errorf("body = %q, want it forwarded unchanged", got.body)
	}
	if len(reply) == 0 {
		t.Error("reply is empty")
	}
}

// An upstream that is down is an error, not an empty reply that would look like
// a chain with no head.
func TestERPCCallerReportsAnUnreachableUpstream(t *testing.T) {
	dead, _ := url.Parse("http://127.0.0.1:1")
	c := NewERPCCaller(dead, "main")
	if _, err := c.Call(context.Background(), 369, []byte(`{}`)); err == nil {
		t.Fatal("err = nil, want a dial failure")
	}
}
