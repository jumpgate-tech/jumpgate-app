package relay

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/valve-tech/valve-node-app/internal/wsrpc"
)

// ErrSubscriptionUnsupported is a subscription kind v1 cannot synthesise over
// HTTP. The relay says so plainly rather than opening a stream that never fires.
var ErrSubscriptionUnsupported = errors.New("relay: subscription not supported")

// JSON-RPC error codes. -32601 and -32600 are the standard pair; the rest of
// the range is free for the application, and -32001 marks a policy refusal so a
// client can tell it from a malformed call.
const (
	codeInvalidRequest = -32600
	codeMethodNotFound = -32601
	codePolicyDenied   = -32001
	codeUpstream       = -32002
)

// supportedSubscriptions are the kinds a poller can feed over plain HTTP.
//
// newPendingTransactions is deliberately absent. It has no honest HTTP polling
// equivalent: txpool_content is non-standard and heavy, and a mempool firehose
// really does need a push transport. Answering the subscribe request with an
// error is more useful than a stream that stays silent forever.
var supportedSubscriptions = map[string]bool{
	"newHeads": true,
	"logs":     true,
	"syncing":  true,
}

// SupportedSubscription reports whether v1 can synthesise a subscription kind.
func SupportedSubscription(kind string) bool { return supportedSubscriptions[kind] }

// StreamHandle stops one subscription.
type StreamHandle interface{ Close() error }

// Streams starts a synthesised subscription. One implementation serves every
// subscriber on a chain from a single poll loop, which is why terminating here
// costs one upstream connection instead of N.
type Streams interface {
	Subscribe(ctx context.Context, chainID int, kind string, params json.RawMessage, notify func(json.RawMessage)) (StreamHandle, error)
}

// RPCCaller performs one JSON-RPC call over HTTP.
type RPCCaller interface {
	Call(ctx context.Context, chainID int, body []byte) ([]byte, error)
}

// WSConfig wires one terminated WebSocket session.
type WSConfig struct {
	Conn    *wsrpc.Conn
	Record  KeyRecord
	ChainID int
	Caller  RPCCaller
	Streams Streams
}

// WSSession serves one customer WebSocket.
//
// The relay terminates the customer's connection and speaks plain HTTP to every
// upstream. That widens the upstream pool to every HTTP-only node and removes
// the gzip-on-upgrade hazard from the relay-to-eRPC hop, because there is no
// upgrade on that hop at all.
//
// It also makes the relay stateful. This struct holds the subscription registry
// for one connection, and Run releases every entry before it returns — a leak
// here is a leak per disconnected customer.
type WSSession struct {
	cfg WSConfig

	// writeMu serialises writes. Notifications arrive from poller goroutines
	// while the read loop may be answering a call, and two concurrent writes
	// would interleave two frames into nonsense.
	writeMu sync.Mutex

	subsMu sync.Mutex
	subs   map[string]StreamHandle
}

// NewWSSession builds a session.
func NewWSSession(cfg WSConfig) *WSSession {
	return &WSSession{cfg: cfg, subs: make(map[string]StreamHandle)}
}

// Run reads frames until the client goes away.
func (s *WSSession) Run(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	defer s.closeAll()

	for {
		msg, err := s.cfg.Conn.ReadMessage()
		if err != nil {
			// The client went away, or sent something unreadable. Either way
			// this session is over and its streams must be released.
			return
		}
		s.handleFrame(ctx, msg)
	}
}

// handleFrame applies policy to one frame and then serves it.
//
// Policy runs per frame because a WebSocket upgrade carries no method. Without
// parsing the stream, allow_trace and the method lists could only ever allow or
// deny WebSocket wholesale.
func (s *WSSession) handleFrame(ctx context.Context, msg []byte) {
	var call struct {
		ID     json.RawMessage `json:"id"`
		Method string          `json:"method"`
		Params json.RawMessage `json:"params"`
	}
	if err := json.Unmarshal(msg, &call); err != nil || call.Method == "" {
		s.writeError(nil, codeInvalidRequest, "malformed JSON-RPC request")
		return
	}

	if err := CheckMethods(s.cfg.Record, []string{call.Method}); err != nil {
		// The connection survives a refusal, so one denied call does not drop a
		// customer's whole session.
		s.writeError(call.ID, codePolicyDenied, "method not allowed for this key")
		return
	}

	switch call.Method {
	case "eth_subscribe":
		s.handleSubscribe(ctx, call.ID, call.Params)
	case "eth_unsubscribe":
		s.handleUnsubscribe(call.ID, call.Params)
	default:
		s.forwardCall(ctx, call.ID, msg)
	}
}

// forwardCall turns a frame into an HTTP POST. This is the part that needs no
// caveat: an ordinary call translates exactly, and the upstream never learns a
// WebSocket was involved.
func (s *WSSession) forwardCall(ctx context.Context, id json.RawMessage, msg []byte) {
	reply, err := s.cfg.Caller.Call(ctx, s.cfg.ChainID, msg)
	if err != nil {
		s.writeError(id, codeUpstream, "upstream unavailable")
		return
	}
	s.write(reply)
}

// handleSubscribe registers a synthesised subscription.
func (s *WSSession) handleSubscribe(ctx context.Context, id json.RawMessage, params json.RawMessage) {
	kind, rest := splitSubscribeParams(params)
	if kind == "" {
		s.writeError(id, codeInvalidRequest, "eth_subscribe needs a subscription name")
		return
	}
	if !SupportedSubscription(kind) {
		s.writeError(id, codeMethodNotFound,
			fmt.Sprintf("%s is not supported over this endpoint", kind))
		return
	}

	subID, err := newSubscriptionID()
	if err != nil {
		s.writeError(id, codeUpstream, "could not open the subscription")
		return
	}

	handle, err := s.cfg.Streams.Subscribe(ctx, s.cfg.ChainID, kind, rest, func(payload json.RawMessage) {
		s.writeNotification(subID, payload)
	})
	if err != nil {
		if errors.Is(err, ErrSubscriptionUnsupported) {
			s.writeError(id, codeMethodNotFound,
				fmt.Sprintf("%s is not supported over this endpoint", kind))
			return
		}
		s.writeError(id, codeUpstream, "could not open the subscription")
		return
	}

	s.subsMu.Lock()
	s.subs[subID] = handle
	s.subsMu.Unlock()

	s.writeResult(id, subID)
}

// handleUnsubscribe releases one subscription this connection holds. An id the
// caller never held returns false rather than tearing down another session's
// stream.
func (s *WSSession) handleUnsubscribe(id json.RawMessage, params json.RawMessage) {
	var args []string
	if err := json.Unmarshal(params, &args); err != nil || len(args) == 0 {
		s.writeError(id, codeInvalidRequest, "eth_unsubscribe needs a subscription id")
		return
	}

	s.subsMu.Lock()
	handle, ok := s.subs[args[0]]
	if ok {
		delete(s.subs, args[0])
	}
	s.subsMu.Unlock()

	if !ok {
		s.writeResult(id, false)
		return
	}
	_ = handle.Close()
	s.writeResult(id, true)
}

// closeAll releases every stream this connection opened.
func (s *WSSession) closeAll() {
	s.subsMu.Lock()
	handles := make([]StreamHandle, 0, len(s.subs))
	for id, h := range s.subs {
		handles = append(handles, h)
		delete(s.subs, id)
	}
	s.subsMu.Unlock()

	for _, h := range handles {
		_ = h.Close()
	}
}

// splitSubscribeParams reads the subscription name and leaves the rest for the
// stream, which is where a logs filter lives.
func splitSubscribeParams(params json.RawMessage) (string, json.RawMessage) {
	var args []json.RawMessage
	if err := json.Unmarshal(params, &args); err != nil || len(args) == 0 {
		return "", nil
	}
	var kind string
	if err := json.Unmarshal(args[0], &kind); err != nil {
		return "", nil
	}
	if len(args) == 1 {
		return kind, nil
	}
	rest, err := json.Marshal(args[1:])
	if err != nil {
		return kind, nil
	}
	return kind, rest
}

// newSubscriptionID mints an opaque id. It is random rather than sequential so
// one customer cannot guess another's, and it is hex-prefixed to match what
// client libraries expect from a node.
func newSubscriptionID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "0x" + hex.EncodeToString(b), nil
}

func (s *WSSession) writeResult(id json.RawMessage, result any) {
	payload, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      rawOrNull(id),
		"result":  result,
	})
	if err != nil {
		return
	}
	s.write(payload)
}

func (s *WSSession) writeError(id json.RawMessage, code int, message string) {
	payload, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      rawOrNull(id),
		"error":   map[string]any{"code": code, "message": message},
	})
	if err != nil {
		return
	}
	s.write(payload)
}

// writeNotification sends one subscription payload in the shape a native node
// uses, so an unmodified client library works against the relay.
func (s *WSSession) writeNotification(subID string, payload json.RawMessage) {
	msg, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"method":  "eth_subscription",
		"params": map[string]any{
			"subscription": subID,
			"result":       payload,
		},
	})
	if err != nil {
		return
	}
	s.write(msg)
}

func (s *WSSession) write(payload []byte) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	_ = s.cfg.Conn.WriteText(payload)
}

// rawOrNull keeps a caller's id shape intact. JSON-RPC allows a string, a
// number, or null, and echoing the wrong type breaks strict clients.
func rawOrNull(id json.RawMessage) any {
	if len(id) == 0 {
		return nil
	}
	return id
}
