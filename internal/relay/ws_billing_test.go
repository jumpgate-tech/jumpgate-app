package relay

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/valve-tech/jumpgate/internal/wsrpc"
)

// Metering must not stop at the upgrade. A WebSocket is the cheapest way to send
// many calls, so a relay that charged only HTTP would sell its most expensive
// traffic for free.

func meteredKey() KeyRecord {
	rec := enabledKey()
	rec.AccountAddress = acct
	return rec
}

// dialHandler serves h on a real listener and dials a WebSocket to path.
func dialHandler(t *testing.T, h http.Handler, path string) *wsHarness {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	client, err := wsrpc.Dial(context.Background(), "ws"+strings.TrimPrefix(srv.URL, "http")+path, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { client.Close() })
	return &wsHarness{conn: client}
}

// readClosed reports whether the server closed the connection within a short
// wait, reading past any frames still in flight.
func readClosed(t *testing.T, c *wsrpc.Conn) bool {
	t.Helper()
	c.SetDeadline(time.Now().Add(2 * time.Second))
	for {
		if _, err := c.ReadMessage(); err != nil {
			return !isTimeout(err)
		}
	}
}

func isTimeout(err error) bool {
	var ne interface{ Timeout() bool }
	return errors.As(err, &ne) && ne.Timeout()
}

func errorCode(msg map[string]any) float64 {
	e, _ := msg["error"].(map[string]any)
	code, _ := e["code"].(float64)
	return code
}

// The handler must wire the same credit lease into the sessions it starts. This
// is the bug the review found: serveWebSocket built a session with no meter.
func TestHandlerChargesWebSocketFrames(t *testing.T) {
	ledger := newLedger(acct, 2)
	caller := &stubCaller{}
	h, err := NewHandler(Config{
		Auth:      staticAuth{rec: meteredKey()},
		ProjectID: "main",
		ERPC:      stubUpstream(t, &capturedRequest{}),
		Caller:    caller,
		Streams:   newStubStreams(),
		Credits:   newLease(ledger, 1),
	})
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}
	ws := dialHandler(t, h, "/rpc/jg_k/evm/369")

	for i := 0; i < 2; i++ {
		ws.send(t, blockNumber)
		if got := ws.read(t); got["result"] != "0x1" {
			t.Fatalf("paid call %d = %v, want result 0x1", i, got)
		}
	}
	ws.send(t, blockNumber)
	if got := ws.read(t); errorCode(got) != codeOutOfCredits {
		t.Fatalf("third call = %v, want error %d", got, codeOutOfCredits)
	}
	if n := len(caller.seen()); n != 2 {
		t.Errorf("upstream served %d calls, want 2: an unpaid frame must cost the operator nothing", n)
	}
}

// A refused frame keeps the connection, the same as a policy refusal, so a
// client can top up without reconnecting.
func TestWSUnpaidCallKeepsTheSession(t *testing.T) {
	h := newWSHarnessWith(t, meteredKey(), func(c *WSConfig) {
		c.Charge = func(context.Context, string) error { return ErrInsufficientCredits }
	})
	h.send(t, blockNumber)
	if got := h.read(t); errorCode(got) != codeOutOfCredits {
		t.Fatalf("got %v, want error %d", got, codeOutOfCredits)
	}
	h.send(t, blockNumber)
	if got := h.read(t); errorCode(got) != codeOutOfCredits {
		t.Fatalf("second frame got %v; the session should still answer", got)
	}
}

// A ledger outage is not a payment problem. Saying "out of credits" would send a
// funded customer to buy credits they already own.
func TestWSLedgerOutageIsNotOutOfCredits(t *testing.T) {
	h := newWSHarnessWith(t, meteredKey(), func(c *WSConfig) {
		c.Charge = func(context.Context, string) error { return ErrUnavailable }
	})
	h.send(t, blockNumber)
	if got := h.read(t); errorCode(got) != codeLedgerUnavailable {
		t.Fatalf("got %v, want error %d", got, codeLedgerUnavailable)
	}
	if n := len(h.caller.seen()); n != 0 {
		t.Errorf("upstream served %d calls while the ledger was down, want 0", n)
	}
}

// eth_subscribe bills per notification (pricing.rs), so opening the stream is
// free and every delivered notification is charged as eth_subscribe.
func TestWSChargesEachNotification(t *testing.T) {
	var (
		mu      sync.Mutex
		charged []string
		credits = 2
	)
	h := newWSHarnessWith(t, meteredKey(), func(c *WSConfig) {
		c.Charge = func(_ context.Context, method string) error {
			mu.Lock()
			defer mu.Unlock()
			charged = append(charged, method)
			if method == "eth_subscribe" {
				if credits == 0 {
					return ErrInsufficientCredits
				}
				credits--
			}
			return nil
		}
	})
	h.send(t, `{"jsonrpc":"2.0","id":1,"method":"eth_subscribe","params":["newHeads"]}`)
	if got := h.read(t); got["result"] == nil {
		t.Fatalf("subscribe = %v, want an id", got)
	}
	mu.Lock()
	n := len(charged)
	mu.Unlock()
	if n != 0 {
		t.Fatal("subscribing was charged; it must be free")
	}

	for i := 0; i < 2; i++ {
		h.streams.push("newHeads", `{"number":"0x1"}`)
		if got := h.read(t); got["method"] != "eth_subscription" {
			t.Fatalf("notification %d = %v", i, got)
		}
	}
	// The account is now empty: the next notification is not delivered and the
	// session ends, so the client reconnects into a 402 rather than idling on a
	// stream that silently stopped.
	h.streams.push("newHeads", `{"number":"0x3"}`)
	if !readClosed(t, h.conn) {
		t.Fatal("session stayed open after the account ran out")
	}
	mu.Lock()
	defer mu.Unlock()
	for _, m := range charged {
		if m != "eth_subscribe" {
			t.Errorf("notification charged as %q, want eth_subscribe", m)
		}
	}
}

// A notification the ledger could not charge is dropped, not delivered free.
func TestWSDropsANotificationTheLedgerCannotCharge(t *testing.T) {
	var failing atomic.Bool
	failing.Store(true)
	h := newWSHarnessWith(t, meteredKey(), func(c *WSConfig) {
		c.Charge = func(_ context.Context, method string) error {
			if method == "eth_subscribe" && failing.Load() {
				return ErrUnavailable
			}
			return nil
		}
	})
	h.send(t, `{"jsonrpc":"2.0","id":1,"method":"eth_subscribe","params":["newHeads"]}`)
	h.read(t)

	h.streams.push("newHeads", `{"number":"0x1"}`)
	failing.Store(false)
	h.streams.push("newHeads", `{"number":"0x2"}`)

	got := h.read(t)
	params, _ := got["params"].(map[string]any)
	result, _ := params["result"].(map[string]any)
	if result["number"] != "0x2" {
		t.Fatalf("first delivered notification = %v, want 0x2 (0x1 was never paid for)", got)
	}
}

// A key revoked mid-session loses its socket. Authentication at the handshake
// alone would let a revoked key stream forever.
func TestWSRevokedKeyLosesItsSocket(t *testing.T) {
	h := newWSHarnessWith(t, meteredKey(), func(c *WSConfig) {
		c.Reauth = func(context.Context) error { return ErrDisabledKey }
	})
	h.send(t, blockNumber)
	if !readClosed(t, h.conn) {
		t.Fatal("a revoked key kept its session")
	}
	if n := len(h.caller.seen()); n != 0 {
		t.Errorf("upstream served %d calls for a revoked key, want 0", n)
	}
}

// settableAuth answers from a settable result and counts lookups.
type settableAuth struct {
	mu    sync.Mutex
	rec   KeyRecord
	err   error
	calls int
}

func (a *settableAuth) Authenticate(context.Context, string) (KeyRecord, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.calls++
	return a.rec, a.err
}

func (a *settableAuth) set(rec KeyRecord, err error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.rec, a.err = rec, err
}

func TestReauthEnforcesRevocationButToleratesAnOutage(t *testing.T) {
	auth := &settableAuth{rec: meteredKey()}
	h, err := NewHandler(Config{Auth: auth, ProjectID: "main", ERPC: stubUpstream(t, &capturedRequest{})})
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}
	ctx := context.Background()

	within := h.reauthEvery("jg_k", time.Hour)
	auth.set(KeyRecord{}, ErrDisabledKey)
	if err := within(ctx); err != nil || auth.calls != 0 {
		t.Fatalf("inside the interval: err=%v lookups=%d, want nil and 0", err, auth.calls)
	}

	check := h.reauthEvery("jg_k", 0)
	auth.set(KeyRecord{}, ErrUnavailable)
	if err := check(ctx); err != nil {
		t.Errorf("store outage ended the session: %v", err)
	}
	auth.set(KeyRecord{ID: "k1", Enabled: false}, nil)
	if err := check(ctx); !errors.Is(err, ErrDisabledKey) {
		t.Errorf("disabled record: err=%v, want ErrDisabledKey", err)
	}
	auth.set(KeyRecord{}, ErrUnknownKey)
	if err := check(ctx); !errors.Is(err, ErrUnknownKey) {
		t.Errorf("deleted key: err=%v, want ErrUnknownKey", err)
	}
}
