package wsrpc

// The server side of the opening handshake. A future relay terminates a
// customer's WebSocket connection with this code, so the proof here is a
// real round trip through httptest.Server and this package's own Dial
// client — not a mock of either end.

import (
	"bufio"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// acceptEcho stands up a real HTTP server whose handler calls Accept and
// echoes back whatever message it reads. It is the httptest.Server side of
// every round-trip test in this file.
func acceptEcho(t *testing.T, opt *Options) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := Accept(w, r, opt)
		if err != nil {
			return
		}
		defer c.Close()
		msg, err := c.ReadMessage()
		if err != nil {
			return
		}
		_ = c.WriteText(msg)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// wsURL turns an httptest.Server's http:// URL into the ws:// URL Dial wants.
func wsURL(srv *httptest.Server) string {
	return "ws" + strings.TrimPrefix(srv.URL, "http")
}

// dialRaw opens a plain TCP connection to srv, for tests that must send
// bytes Dial's well-behaved client cannot produce — a request Accept should
// reject, or frames a compliant client would never send.
func dialRaw(t *testing.T, srv *httptest.Server) net.Conn {
	t.Helper()
	addr := strings.TrimPrefix(srv.URL, "http://")
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	return conn
}

// ---------------------------------------------------------------------
// the happy path
// ---------------------------------------------------------------------

func TestAccept_UpgradesAndEchoes(t *testing.T) {
	srv := acceptEcho(t, nil)

	c, err := Dial(ctx(t), wsURL(srv), nil)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer c.Close()

	if err := c.WriteText([]byte("hello")); err != nil {
		t.Fatalf("WriteText: %v", err)
	}
	got, err := c.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage: %v", err)
	}
	if string(got) != "hello" {
		t.Errorf("got %q, want the echoed message", got)
	}
}

// This repo has a scar: eRPC used to answer HTTP 500 on a WebSocket upgrade
// whenever the client sent Accept-Encoding: gzip, because its gzip
// negotiation ran ahead of the upgrade check — and every reverse proxy adds
// that header automatically, so this hit real traffic. Accept-Encoding has
// nothing to do with upgrading a connection, and Accept must never let it
// decide the outcome.
func TestAccept_IgnoresAcceptEncodingGzip(t *testing.T) {
	srv := acceptEcho(t, nil)
	conn := dialRaw(t, srv)

	req := "GET / HTTP/1.1\r\n" +
		"Host: " + strings.TrimPrefix(srv.URL, "http://") + "\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==\r\n" +
		"Sec-WebSocket-Version: 13\r\n" +
		"Accept-Encoding: gzip\r\n\r\n"
	if _, err := conn.Write([]byte(req)); err != nil {
		t.Fatalf("write request: %v", err)
	}

	resp, err := http.ReadResponse(bufio.NewReader(conn), &http.Request{Method: http.MethodGet})
	if err != nil {
		t.Fatalf("ReadResponse: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusSwitchingProtocols {
		t.Fatalf("status = %s, want 101 — Accept-Encoding: gzip must not change the outcome", resp.Status)
	}
}

// ---------------------------------------------------------------------
// rejections — Accept must not hijack a request it refuses
// ---------------------------------------------------------------------

func TestAccept_RejectsANonUpgradeGET(t *testing.T) {
	var acceptErr error
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, acceptErr = Accept(w, r, nil)
	}))
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL) //nolint:noctx — a plain GET is the point of the test
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusSwitchingProtocols {
		t.Fatal("a plain GET with no upgrade headers was accepted")
	}
	if !errors.Is(acceptErr, ErrBadUpgrade) {
		t.Errorf("Accept error %v is not ErrBadUpgrade", acceptErr)
	}
}

func TestAccept_RejectsWrongSecWebSocketVersion(t *testing.T) {
	var acceptErr error
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, acceptErr = Accept(w, r, nil)
	}))
	t.Cleanup(srv.Close)
	conn := dialRaw(t, srv)

	req := "GET / HTTP/1.1\r\n" +
		"Host: " + strings.TrimPrefix(srv.URL, "http://") + "\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==\r\n" +
		"Sec-WebSocket-Version: 8\r\n\r\n"
	if _, err := conn.Write([]byte(req)); err != nil {
		t.Fatalf("write request: %v", err)
	}

	resp, err := http.ReadResponse(bufio.NewReader(conn), &http.Request{Method: http.MethodGet})
	if err != nil {
		t.Fatalf("ReadResponse: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusSwitchingProtocols {
		t.Fatal("Sec-WebSocket-Version: 8 was accepted")
	}
	if !errors.Is(acceptErr, ErrBadUpgrade) {
		t.Errorf("Accept error %v is not ErrBadUpgrade", acceptErr)
	}
}

// ---------------------------------------------------------------------
// framing on the wire — a server reading a client, byte for byte
// ---------------------------------------------------------------------

// Dial's own client never fragments a message, so proving the server
// reassembles one takes a hand-built client — the same way frame_test.go
// proves readMessage does it, but here through the real Accept handler.
func TestAccept_ReassemblesAFragmentedClientMessage(t *testing.T) {
	srv := acceptEcho(t, nil)
	raw := dialRaw(t, srv)

	addr := strings.TrimPrefix(srv.URL, "http://")
	if _, err := Handshake(ctx(t), raw, addr, "/", nil); err != nil {
		t.Fatalf("Handshake: %v", err)
	}

	mask := []byte{0xDE, 0xAD, 0xBE, 0xEF}
	for _, f := range []wsFrame{
		{fin: false, opcode: opText, payload: []byte("frag"), mask: mask},
		{fin: false, opcode: opCont, payload: []byte("ment"), mask: mask},
		{fin: true, opcode: opCont, payload: []byte("ed"), mask: mask},
	} {
		if _, err := raw.Write(f.bytes()); err != nil {
			t.Fatalf("write fragment: %v", err)
		}
	}

	got, err := readMessage(bufio.NewReader(raw), DefaultMaxMessageBytes, false, nil)
	if err != nil {
		t.Fatalf("reading the echo: %v", err)
	}
	if string(got) != "fragmented" {
		t.Errorf("got %q, want the server to have reassembled and echoed \"fragmented\"", got)
	}
}

// RFC 6455 §5.1 requires every client frame masked, and a server MUST fail
// the connection on one that is not. A proxy or a hostile client can send
// one; readFrame used to unmask liberally and accept it, which would hand
// the relay plausible-looking garbage instead of an error.
func TestAccept_RejectsAnUnmaskedClientFrame(t *testing.T) {
	handlerDone := make(chan error, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := Accept(w, r, nil)
		if err != nil {
			handlerDone <- err
			return
		}
		defer c.Close()
		_, err = c.ReadMessage()
		handlerDone <- err
	}))
	t.Cleanup(srv.Close)
	raw := dialRaw(t, srv)

	addr := strings.TrimPrefix(srv.URL, "http://")
	if _, err := Handshake(ctx(t), raw, addr, "/", nil); err != nil {
		t.Fatalf("Handshake: %v", err)
	}

	unmasked := wsFrame{fin: true, opcode: opText, payload: []byte("no mask")}
	if _, err := raw.Write(unmasked.bytes()); err != nil {
		t.Fatalf("write unmasked frame: %v", err)
	}

	err := <-handlerDone
	if err == nil {
		t.Fatal("the server accepted an unmasked client frame")
	}
}

// ---------------------------------------------------------------------
// control frames
// ---------------------------------------------------------------------

// RFC 6455 §5.5.2 requires a pong carrying the ping's own payload, and the
// exchange must not disturb a data message that follows it.
func TestAccept_RepliesToAPingWithAPong(t *testing.T) {
	srv := acceptEcho(t, nil)
	raw := dialRaw(t, srv)

	addr := strings.TrimPrefix(srv.URL, "http://")
	c, err := Handshake(ctx(t), raw, addr, "/", nil)
	if err != nil {
		t.Fatalf("Handshake: %v", err)
	}

	mask := []byte{1, 2, 3, 4}
	ping := wsFrame{fin: true, opcode: opPing, payload: []byte("keepalive"), mask: mask}
	if _, err := raw.Write(ping.bytes()); err != nil {
		t.Fatalf("write ping: %v", err)
	}

	fin, opcode, payload, err := readFrame(c.br, DefaultMaxMessageBytes, false)
	if err != nil {
		t.Fatalf("readFrame: %v", err)
	}
	if !fin || opcode != opcodePong {
		t.Fatalf("got fin=%v opcode=%#x, want a pong (fin=true opcode=%#x)", fin, opcode, opcodePong)
	}
	if string(payload) != "keepalive" {
		t.Errorf("pong payload = %q, want the ping's payload echoed back", payload)
	}

	text := wsFrame{fin: true, opcode: opText, payload: []byte("hi"), mask: mask}
	if _, err := raw.Write(text.bytes()); err != nil {
		t.Fatalf("write text: %v", err)
	}
	got, err := c.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage after the ping/pong exchange: %v", err)
	}
	if string(got) != "hi" {
		t.Errorf("got %q, want the echoed message", got)
	}
}
