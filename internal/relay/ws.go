package relay

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/valve-tech/jumpgate/internal/wsrpc"
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
	// codeOutOfCredits and codeLedgerUnavailable mirror HTTP's 402 and 503.
	codeOutOfCredits      = -32003
	codeLedgerUnavailable = -32004
	// codeAccountNotProvisioned mirrors HTTP's 403 for a key whose funding
	// account the ledger has never seen.
	codeAccountNotProvisioned = -32006
	// codeRateLimited mirrors HTTP's 429. -32005 is the code EIP-1474 reserves
	// for "limit exceeded", so client libraries already recognise it.
	codeRateLimited = -32005
)

// closePolicyViolation is RFC 6455's 1008: the session ended because the key may
// no longer be served, not because anything broke.
const closePolicyViolation uint16 = 1008

// closeTryAgainLater is 1013: the session ended because the client fell too far
// behind its own stream. Nothing is wrong with the key, and reconnecting is the
// right response.
const closeTryAgainLater uint16 = 1013

// Session timing defaults. A client gets three pings to answer before the idle
// timeout reaps it, so one lost pong on a bad network is not a disconnect. The
// write timeout is long enough for a congested but live client and short enough
// that a client which stopped reading is gone within seconds.
const (
	defaultWSWriteTimeout = 10 * time.Second
	defaultWSIdleTimeout  = 90 * time.Second
	defaultWSPingInterval = 30 * time.Second
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
//
// notify may block: an implementation must not let one subscriber's delivery
// hold up another's. lost, when not nil, is called at most once if the stream
// gives up on a subscriber that fell too far behind. After that, notify is
// never called again, so a caller that ignored lost would hold a subscription
// that had silently stopped.
type Streams interface {
	Subscribe(ctx context.Context, chainID int, kind string, params json.RawMessage, notify func(json.RawMessage), lost func()) (StreamHandle, error)
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
	// Charge meters one call, by method, against the key's account. Nil serves
	// unmetered, matching a gateway with billing off.
	Charge func(ctx context.Context, method string) error
	// Reauth re-checks the key. A non-nil error ends the session. Nil never
	// re-checks.
	Reauth func(ctx context.Context) error
	// Admit applies the key's rate limits to one call. False refuses the frame.
	// Nil admits everything.
	Admit func() bool

	// WriteTimeout bounds one frame write, IdleTimeout ends a session that
	// sends no frame at all for that long, and PingInterval is how often the
	// session pings so a listen-only client has something to answer. Zero
	// means the default for each.
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
	PingInterval time.Duration
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
	if cfg.WriteTimeout <= 0 {
		cfg.WriteTimeout = defaultWSWriteTimeout
	}
	if cfg.IdleTimeout <= 0 {
		cfg.IdleTimeout = defaultWSIdleTimeout
	}
	if cfg.PingInterval <= 0 {
		cfg.PingInterval = defaultWSPingInterval
	}
	return &WSSession{cfg: cfg, subs: make(map[string]StreamHandle)}
}

// Run reads frames until the client goes away.
//
// "Goes away" includes going quiet. A client that vanishes without a close
// frame would otherwise hold its read, and every stream it opened, forever. The
// idle timeout reaps it, and the pings give a live client that only listens
// something to answer.
func (s *WSSession) Run(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	defer s.closeAll()

	s.cfg.Conn.SetWriteTimeout(s.cfg.WriteTimeout)
	s.cfg.Conn.SetIdleTimeout(s.cfg.IdleTimeout)
	go s.ping(ctx)

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

	if !s.stillAuthorised(ctx) {
		return
	}
	if err := CheckMethods(s.cfg.Record, []string{call.Method}); err != nil {
		// The connection survives a refusal, so one denied call does not drop a
		// customer's whole session.
		s.writeError(call.ID, codePolicyDenied, "method not allowed for this key")
		return
	}

	// eth_unsubscribe is exempt from both the throttle and the charge, so a
	// client that is over its limit or out of credit can still stop the stream
	// whose notifications are being charged. It opens no free path: it only
	// cancels a subscription this session owns and never reaches the upstream.
	exempt := call.Method == "eth_unsubscribe"

	// Throttle before charging, as the HTTP path does, so a refused frame costs
	// nothing. The session survives: a client that slows down carries on.
	if !exempt && s.cfg.Admit != nil && !s.cfg.Admit() {
		s.writeError(call.ID, codeRateLimited, "rate limit exceeded for this key")
		return
	}

	// Charge BEFORE serving, as the HTTP path does. eth_subscribe is the one
	// exception: it bills per delivered notification (see deliver), so opening
	// the stream is free.
	if call.Method != "eth_subscribe" && !exempt {
		if err := s.charge(ctx, call.Method); err != nil {
			s.writeChargeError(call.ID, err)
			return
		}
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
		s.deliver(ctx, subID, payload)
	}, func() {
		// The stream gave up on this client for falling behind. Ending the
		// session is the honest answer: a subscription that silently stopped
		// would leave the client waiting for events that will never come.
		s.end(closeTryAgainLater, "subscription fell too far behind")
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

// deliver charges one notification and then writes it.
//
// An empty account ends the session rather than leaving a stream that silently
// stops: the client reconnects into a 402, which says what is wrong. A ledger
// that cannot answer drops this one notification instead, because delivering it
// would be delivering it free, and ending a funded customer's session over an
// outage would be worse than one missed head.
func (s *WSSession) deliver(ctx context.Context, subID string, payload json.RawMessage) {
	if !s.stillAuthorised(ctx) {
		return
	}
	if err := s.charge(ctx, "eth_subscribe"); err != nil {
		switch {
		case errors.Is(err, ErrInsufficientCredits):
			s.end(closePolicyViolation, "account is out of credits")
		case errors.Is(err, ErrNoAccount):
			// Not fixed mid-session either: the operator must provision it.
			logNoAccount(err)
			s.end(closePolicyViolation, msgNotProvisioned)
		}
		return
	}
	s.writeNotification(subID, payload)
}

func (s *WSSession) charge(ctx context.Context, method string) error {
	if s.cfg.Charge == nil {
		return nil
	}
	return s.cfg.Charge(ctx, method)
}

// stillAuthorised re-checks the key and ends the session if it was revoked. The
// handshake alone would let a revoked key keep its socket for as long as the
// customer cared to hold it open.
func (s *WSSession) stillAuthorised(ctx context.Context) bool {
	if s.cfg.Reauth == nil {
		return true
	}
	if err := s.cfg.Reauth(ctx); err != nil {
		s.end(closePolicyViolation, "key is no longer valid")
		return false
	}
	return true
}

// end closes the connection. Run's read then fails, and Run releases every
// stream this session holds.
func (s *WSSession) end(code uint16, reason string) {
	s.writeMu.Lock()
	_ = s.cfg.Conn.WriteClose(code, reason)
	s.writeMu.Unlock()
	_ = s.cfg.Conn.Close()
}

// writeChargeError mirrors the HTTP path's split between "cannot pay" and
// "cannot tell".
func (s *WSSession) writeChargeError(id json.RawMessage, err error) {
	if errors.Is(err, ErrInsufficientCredits) {
		s.writeError(id, codeOutOfCredits, "account is out of credits")
		return
	}
	if errors.Is(err, ErrNoAccount) {
		logNoAccount(err)
		s.writeError(id, codeAccountNotProvisioned, msgNotProvisioned)
		return
	}
	s.writeError(id, codeLedgerUnavailable, "the credit ledger did not answer")
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

// write sends one frame. A write that fails, including one that hit the write
// timeout because the client stopped reading, closes the connection: Run's read
// then fails and releases every stream, instead of the session living on with
// writes that can never land.
func (s *WSSession) write(payload []byte) {
	s.writeMu.Lock()
	err := s.cfg.Conn.WriteText(payload)
	s.writeMu.Unlock()
	if err != nil {
		_ = s.cfg.Conn.Close()
	}
}

// ping keeps a listen-only client inside the idle timeout until Run returns. A
// ping that cannot be written means the client is gone, so it ends the session
// the same way a failed write does.
func (s *WSSession) ping(ctx context.Context) {
	ticker := time.NewTicker(s.cfg.PingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		s.writeMu.Lock()
		err := s.cfg.Conn.WritePing(nil)
		s.writeMu.Unlock()
		if err != nil {
			_ = s.cfg.Conn.Close()
			return
		}
	}
}

// rawOrNull keeps a caller's id shape intact. JSON-RPC allows a string, a
// number, or null, and echoing the wrong type breaks strict clients.
func rawOrNull(id json.RawMessage) any {
	if len(id) == 0 {
		return nil
	}
	return id
}
