package relay

import (
	"bufio"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/valve-tech/valve-node-app/internal/wsrpc"
)

// The relay must TERMINATE a WebSocket, not proxy it. Proxying would send the
// upgrade to eRPC, which puts the subscription back on the upstream and brings
// the gzip-on-upgrade hazard back with it. These tests drive a real handler
// with a real client and assert on which side answered.

// upgradeCounter records whether an upstream ever saw an upgrade attempt.
type upgradeCounter struct{ upgrades int }

func (u *upgradeCounter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
		u.upgrades++
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":"0x1"}`))
}

func newUpgradeHarness(t *testing.T, rec KeyRecord) (*httptest.Server, *upgradeCounter, *stubStreams) {
	t.Helper()
	counter := &upgradeCounter{}
	upstream := httptest.NewServer(counter)
	t.Cleanup(upstream.Close)

	erpc, err := url.Parse(upstream.URL)
	if err != nil {
		t.Fatalf("parse upstream: %v", err)
	}
	streams := newStubStreams()
	h, err := NewHandler(Config{
		Auth:      staticAuth{rec: rec},
		ProjectID: "main",
		ERPC:      erpc,
		Streams:   streams,
		Caller:    &stubCaller{},
	})
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}
	relaySrv := httptest.NewServer(h)
	t.Cleanup(relaySrv.Close)
	return relaySrv, counter, streams
}

// The relay answers the upgrade itself, and eRPC never sees one.
func TestUpgradeIsTerminatedAtTheRelay(t *testing.T) {
	relaySrv, counter, _ := newUpgradeHarness(t, enabledKey())

	wsURL := "ws" + strings.TrimPrefix(relaySrv.URL, "http") + "/rpc/jg_k/evm/369"
	conn, err := wsrpc.Dial(context.Background(), wsURL, nil)
	if err != nil {
		t.Fatalf("dial through the relay: %v", err)
	}
	defer conn.Close()

	if counter.upgrades != 0 {
		t.Errorf("eRPC saw %d upgrade attempts, want 0 — the relay proxied instead of terminating", counter.upgrades)
	}
}

// gzip on the handshake must not break the upgrade. This is the exact shape of
// the eRPC bug — it answered 500 on an upgrade whenever Accept-Encoding: gzip
// was present, which every proxy adds — and the relay is now a proxy that could
// reintroduce it. The handshake is written by hand because the package's own
// client sends no such header, and the header is the whole point of the test.
func TestUpgradeSurvivesGzipOnTheHandshake(t *testing.T) {
	relaySrv, _, _ := newUpgradeHarness(t, enabledKey())

	addr := strings.TrimPrefix(relaySrv.URL, "http://")
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	req := "GET /rpc/jg_k/evm/369 HTTP/1.1\r\n" +
		"Host: " + addr + "\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==\r\n" +
		"Sec-WebSocket-Version: 13\r\n" +
		"Accept-Encoding: gzip\r\n\r\n"
	if _, err := conn.Write([]byte(req)); err != nil {
		t.Fatalf("write handshake: %v", err)
	}

	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	status, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		t.Fatalf("read status line: %v", err)
	}
	if !strings.Contains(status, "101") {
		t.Fatalf("status = %q, want 101 Switching Protocols with gzip present", strings.TrimSpace(status))
	}
}

// An upgrade is still a keyed request. An unknown key must be refused before
// any connection is established.
func TestUpgradeRefusesAnUnknownKey(t *testing.T) {
	counter := &upgradeCounter{}
	upstream := httptest.NewServer(counter)
	t.Cleanup(upstream.Close)
	erpc, _ := url.Parse(upstream.URL)

	h, err := NewHandler(Config{
		Auth:      staticAuth{err: ErrUnknownKey},
		ProjectID: "main",
		ERPC:      erpc,
		Streams:   newStubStreams(),
		Caller:    &stubCaller{},
	})
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}
	relaySrv := httptest.NewServer(h)
	t.Cleanup(relaySrv.Close)

	wsURL := "ws" + strings.TrimPrefix(relaySrv.URL, "http") + "/rpc/jg_bad/evm/369"
	conn, err := wsrpc.Dial(context.Background(), wsURL, nil)
	if err == nil {
		conn.Close()
		t.Fatal("the relay upgraded a connection for an unknown key")
	}
}

// Policy reaches the stream through the upgrade path too, which is the point of
// terminating: the handshake carries no method, so only per-frame checks work.
func TestUpgradedSessionAppliesMethodPolicy(t *testing.T) {
	rec := KeyRecord{ID: "k1", Enabled: true, MethodBlock: []string{"eth_sendRawTransaction"}}
	relaySrv, _, _ := newUpgradeHarness(t, rec)

	wsURL := "ws" + strings.TrimPrefix(relaySrv.URL, "http") + "/rpc/jg_k/evm/369"
	conn, err := wsrpc.Dial(context.Background(), wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	if err := conn.WriteText([]byte(`{"jsonrpc":"2.0","id":1,"method":"eth_sendRawTransaction","params":["0x0"]}`)); err != nil {
		t.Fatalf("write: %v", err)
	}
	conn.SetDeadline(time.Now().Add(3 * time.Second))
	raw, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("decode %q: %v", raw, err)
	}
	if out["error"] == nil {
		t.Fatalf("got %v, want a policy refusal over the terminated stream", out)
	}
}

// A plain POST on the same path still proxies. Terminating WebSocket must not
// change how an ordinary call is served.
func TestPlainPostStillProxiesAfterUpgradeSupport(t *testing.T) {
	var got capturedRequest
	up := stubUpstream(t, &got)
	h, err := NewHandler(Config{
		Auth:      staticAuth{rec: enabledKey()},
		ProjectID: "main",
		ERPC:      up,
		Streams:   newStubStreams(),
		Caller:    &stubCaller{},
	})
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}
	if res := post(t, h, "/rpc/jg_k/evm/369", blockNumber, nil); res.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", res.Code)
	}
	if got.path != "/main/evm/369" {
		t.Errorf("upstream path = %q, want /main/evm/369", got.path)
	}
}
