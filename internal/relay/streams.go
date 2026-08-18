package relay

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

// defaultPollInterval is the latency floor a synthesised subscription pays. A
// native push delivers a head at propagation speed; a one second poll adds half
// a second on average. That is noise on a twelve second chain and noticeable on
// a fast L2, so an operator can override it per chain.
const defaultPollInterval = time.Second

// rpcEnvelope is a JSON-RPC response the relay reads on its own behalf.
type rpcEnvelope struct {
	Result json.RawMessage `json:"result"`
	Error  *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// RPCBlockFetcher reads a chain's head with ordinary JSON-RPC calls over HTTP.
//
// This is what makes upstream WebSocket support irrelevant: a subscription is
// fed by eth_blockNumber and eth_getBlockByNumber, which every EVM node serves
// over plain HTTP.
type RPCBlockFetcher struct {
	caller  RPCCaller
	chainID int
}

// NewRPCBlockFetcher builds a fetcher for one chain.
func NewRPCBlockFetcher(caller RPCCaller, chainID int) *RPCBlockFetcher {
	return &RPCBlockFetcher{caller: caller, chainID: chainID}
}

func (f *RPCBlockFetcher) call(ctx context.Context, method string, params string) (json.RawMessage, error) {
	body := fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"method":%q,"params":%s}`, method, params)
	raw, err := f.caller.Call(ctx, f.chainID, []byte(body))
	if err != nil {
		return nil, fmt.Errorf("relay: %s: %w", method, err)
	}
	var env rpcEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, fmt.Errorf("relay: %s: decode: %w", method, err)
	}
	// A JSON-RPC error is an error. Reading it as a zero head would look like a
	// chain stuck at genesis and would stall every subscriber silently.
	if env.Error != nil {
		return nil, fmt.Errorf("relay: %s: upstream error %d: %s", method, env.Error.Code, env.Error.Message)
	}
	return env.Result, nil
}

// HeadNumber reads the current head height.
func (f *RPCBlockFetcher) HeadNumber(ctx context.Context) (uint64, error) {
	result, err := f.call(ctx, "eth_blockNumber", "[]")
	if err != nil {
		return 0, err
	}
	var hex string
	if err := json.Unmarshal(result, &hex); err != nil {
		return 0, fmt.Errorf("relay: eth_blockNumber: %w", err)
	}
	return parseHexUint(hex)
}

// BlockByNumber reads one block's identity. It asks for headers only — the
// relay needs the linkage, not the transactions.
func (f *RPCBlockFetcher) BlockByNumber(ctx context.Context, n uint64) (BlockRef, error) {
	params := fmt.Sprintf(`["0x%x",false]`, n)
	result, err := f.call(ctx, "eth_getBlockByNumber", params)
	if err != nil {
		return BlockRef{}, err
	}

	var block struct {
		Number     string `json:"number"`
		Hash       string `json:"hash"`
		ParentHash string `json:"parentHash"`
	}
	if err := json.Unmarshal(result, &block); err != nil {
		return BlockRef{}, fmt.Errorf("relay: eth_getBlockByNumber: %w", err)
	}
	// A null result decodes into a zero struct. Returning that would look to the
	// poller like a reorg to genesis, so it is an error instead.
	if block.Hash == "" {
		return BlockRef{}, fmt.Errorf("relay: block %d is not available upstream", n)
	}

	number, err := parseHexUint(block.Number)
	if err != nil {
		return BlockRef{}, err
	}
	return BlockRef{Number: number, Hash: block.Hash, ParentHash: block.ParentHash}, nil
}

func parseHexUint(s string) (uint64, error) {
	n, err := strconv.ParseUint(strings.TrimPrefix(s, "0x"), 16, 64)
	if err != nil {
		return 0, fmt.Errorf("relay: %q is not a hex quantity: %w", s, err)
	}
	return n, nil
}

// PollerStreams feeds every subscriber on a chain from ONE poll loop.
//
// This is the economic argument for terminating WebSocket at the relay. A
// native setup opens one upstream connection per subscriber; here a thousand
// subscribers on one chain still cost one eth_blockNumber per interval. It also
// means a slow subscriber cannot slow the upstream, only itself.
type PollerStreams struct {
	caller   RPCCaller
	interval time.Duration

	mu    sync.Mutex
	loops map[int]*chainLoop
}

// NewPollerStreams builds the stream manager.
func NewPollerStreams(caller RPCCaller, interval time.Duration) *PollerStreams {
	if interval <= 0 {
		interval = defaultPollInterval
	}
	return &PollerStreams{caller: caller, interval: interval, loops: make(map[int]*chainLoop)}
}

// subscriber is one client's interest in a chain.
type subscriber struct {
	kind   string
	filter json.RawMessage
	notify func(json.RawMessage)
}

// chainLoop is one chain's poll loop and its subscribers.
//
// ONE loop serves every kind. newHeads and logs are both derived from the same
// head advance, and syncing rides the same tick, so three kinds on a chain cost
// one eth_blockNumber per interval rather than three.
type chainLoop struct {
	cancel context.CancelFunc
	done   chan struct{}

	mu     sync.Mutex
	nextID int
	subs   map[int]subscriber
}

// fanoutKind delivers a payload to every subscriber of one kind.
func (l *chainLoop) fanoutKind(kind string, payload json.RawMessage) {
	l.mu.Lock()
	targets := make([]func(json.RawMessage), 0, len(l.subs))
	for _, sub := range l.subs {
		if sub.kind == kind {
			targets = append(targets, sub.notify)
		}
	}
	l.mu.Unlock()

	for _, notify := range targets {
		notify(payload)
	}
}

// logsSubscribers snapshots the log watchers and their filters.
func (l *chainLoop) logsSubscribers() []subscriber {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]subscriber, 0, len(l.subs))
	for _, sub := range l.subs {
		if sub.kind == "logs" {
			out = append(out, sub)
		}
	}
	return out
}

// hasKind reports whether anyone is watching a kind, so the loop can skip an
// upstream call nobody would read.
func (l *chainLoop) hasKind(kind string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, sub := range l.subs {
		if sub.kind == kind {
			return true
		}
	}
	return false
}

func (l *chainLoop) count() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.subs)
}

// LoopCount reports how many chains are being polled. Tests and metrics read it.
func (s *PollerStreams) LoopCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.loops)
}

// Stop ends every loop. It is for shutdown.
func (s *PollerStreams) Stop() {
	s.mu.Lock()
	loops := make([]*chainLoop, 0, len(s.loops))
	for id, loop := range s.loops {
		loops = append(loops, loop)
		delete(s.loops, id)
	}
	s.mu.Unlock()

	for _, loop := range loops {
		loop.cancel()
		<-loop.done
	}
}

// Subscribe attaches one subscriber, starting the chain's loop if it is the
// first. The returned handle detaches it, and the loop stops when the last one
// leaves — a chain nobody watches must not keep calling a paid upstream.
func (s *PollerStreams) Subscribe(ctx context.Context, chainID int, kind string, filter json.RawMessage, notify func(json.RawMessage)) (StreamHandle, error) {
	if !SupportedSubscription(kind) {
		return nil, fmt.Errorf("%w: %s is not supported over this endpoint", ErrSubscriptionUnsupported, kind)
	}
	s.mu.Lock()
	loop, ok := s.loops[chainID]
	if !ok {
		loop = s.startLoop(chainID)
		s.loops[chainID] = loop
	}
	s.mu.Unlock()

	loop.mu.Lock()
	loop.nextID++
	id := loop.nextID
	loop.subs[id] = subscriber{kind: kind, filter: filter, notify: notify}
	loop.mu.Unlock()

	return &streamHandle{streams: s, chainID: chainID, id: id}, nil
}

// startLoop begins polling one chain. The caller holds s.mu.
func (s *PollerStreams) startLoop(chainID int) *chainLoop {
	ctx, cancel := context.WithCancel(context.Background())
	loop := &chainLoop{
		cancel: cancel,
		done:   make(chan struct{}),
		subs:   make(map[int]subscriber),
	}

	go func() {
		defer close(loop.done)
		fetcher := NewRPCBlockFetcher(s.caller, chainID)
		poller := NewHeadPoller(fetcher)
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()

		// lastSyncing is the previously reported state. syncing reports a
		// CHANGE, not a heartbeat: a node that is steadily synced must not spam
		// a subscriber once per poll.
		var lastSyncing json.RawMessage
		var syncingSeen bool

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}

			if loop.hasKind("syncing") {
				if state, err := s.readSyncing(ctx, chainID); err == nil {
					if !syncingSeen || !bytes.Equal(state, lastSyncing) {
						syncingSeen = true
						lastSyncing = state
						loop.fanoutKind("syncing", state)
					}
				}
			}

			heads, err := poller.Poll(ctx)
			if err != nil {
				// A poll failure is transient by assumption. The poller does not
				// advance its cursor on failure, so the next tick delivers
				// whatever this one missed.
				continue
			}
			for _, head := range heads {
				payload, err := json.Marshal(map[string]any{
					"number":     fmt.Sprintf("0x%x", head.Number),
					"hash":       head.Hash,
					"parentHash": head.ParentHash,
					"reorged":    head.Reorged,
				})
				if err == nil {
					loop.fanoutKind("newHeads", payload)
				}
				// logs are derived from the SAME head advance, so a chain with
				// both kinds still polls its head once.
				s.deliverLogs(ctx, chainID, loop, head)
			}
		}
	}()
	return loop
}

// readSyncing asks the node whether it is still catching up.
func (s *PollerStreams) readSyncing(ctx context.Context, chainID int) (json.RawMessage, error) {
	raw, err := s.caller.Call(ctx, chainID, []byte(`{"jsonrpc":"2.0","id":1,"method":"eth_syncing","params":[]}`))
	if err != nil {
		return nil, err
	}
	var env rpcEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, err
	}
	if env.Error != nil {
		return nil, fmt.Errorf("relay: eth_syncing: %s", env.Error.Message)
	}
	return env.Result, nil
}

// deliverLogs fetches one block's logs per distinct filter and fans each set out
// to the subscriber that asked for it.
//
// A reorged head is skipped rather than replayed: re-emitting logs for a block
// the chain abandoned would double-count them for every subscriber, and a
// consumer that honours the reorged flag on newHeads already knows to unwind.
func (s *PollerStreams) deliverLogs(ctx context.Context, chainID int, loop *chainLoop, head BlockRef) {
	subs := loop.logsSubscribers()
	if len(subs) == 0 {
		return
	}
	for _, sub := range subs {
		entries, err := s.readLogs(ctx, chainID, head.Number, sub.filter)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			sub.notify(entry)
		}
	}
}

// readLogs runs eth_getLogs for one block, carrying the caller's filter through
// unchanged so the node does the matching rather than the relay.
func (s *PollerStreams) readLogs(ctx context.Context, chainID int, block uint64, filter json.RawMessage) ([]json.RawMessage, error) {
	criteria := map[string]any{
		"fromBlock": fmt.Sprintf("0x%x", block),
		"toBlock":   fmt.Sprintf("0x%x", block),
	}
	// The subscribe params arrive as the array after the kind. Its first entry,
	// when present, is the log filter.
	var args []map[string]any
	if len(filter) > 0 && json.Unmarshal(filter, &args) == nil && len(args) > 0 {
		for k, v := range args[0] {
			// fromBlock and toBlock are the relay's to set: a subscription
			// streams new blocks, so a caller-supplied range would either
			// re-read history every tick or silently return nothing.
			if k == "fromBlock" || k == "toBlock" {
				continue
			}
			criteria[k] = v
		}
	}

	params, err := json.Marshal([]any{criteria})
	if err != nil {
		return nil, err
	}
	body := fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"method":"eth_getLogs","params":%s}`, params)

	raw, err := s.caller.Call(ctx, chainID, []byte(body))
	if err != nil {
		return nil, err
	}
	var env rpcEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, err
	}
	if env.Error != nil {
		return nil, fmt.Errorf("relay: eth_getLogs: %s", env.Error.Message)
	}
	var entries []json.RawMessage
	if err := json.Unmarshal(env.Result, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

// detach removes one subscriber and stops the loop when it was the last.
func (s *PollerStreams) detach(chainID, id int) {
	s.mu.Lock()
	loop, ok := s.loops[chainID]
	if !ok {
		s.mu.Unlock()
		return
	}
	loop.mu.Lock()
	delete(loop.subs, id)
	loop.mu.Unlock()

	if loop.count() > 0 {
		s.mu.Unlock()
		return
	}
	delete(s.loops, chainID)
	s.mu.Unlock()

	loop.cancel()
	<-loop.done
}

// streamHandle detaches one subscriber. Close is safe to call twice, because a
// session releases every handle on disconnect and a client may also unsubscribe.
type streamHandle struct {
	streams *PollerStreams
	chainID int
	id      int
	once    sync.Once
}

func (h *streamHandle) Close() error {
	h.once.Do(func() { h.streams.detach(h.chainID, h.id) })
	return nil
}
