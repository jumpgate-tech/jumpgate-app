package wsrpc

// The opening handshake, which none of the three implementations this package
// replaced had a single test for — even though it is the part that decides the
// answer these probes exist to produce. "This endpoint advertises wss:// and
// cannot speak it" is a handshake verdict, and getting it wrong in either
// direction is a wrong answer shown to an operator.

import (
	"bufio"
	"context"
	"crypto/sha1"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// serveRaw hands every accepted connection to fn with the request already
// parsed, so a test can answer with anything at all — including the
// not-quite-WebSocket responses that real endpoints produce and that a
// net/http handler will not let you write.
func serveRaw(t *testing.T, fn func(conn net.Conn, req *http.Request)) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func() {
				defer conn.Close()
				req, err := http.ReadRequest(bufio.NewReader(conn))
				if err != nil {
					return
				}
				fn(conn, req)
			}()
		}
	}()
	return ln.Addr().String()
}

// accept101 answers with a correct RFC 6455 handshake, digest and all.
func accept101(conn net.Conn, req *http.Request) {
	sum := sha1.Sum([]byte(req.Header.Get("Sec-WebSocket-Key") + GUID)) //nolint:gosec — RFC 6455 defines SHA-1 here
	fmt.Fprintf(conn, "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: %s\r\n\r\n",
		base64.StdEncoding.EncodeToString(sum[:]))
}

func ctx(t *testing.T) context.Context {
	t.Helper()
	c, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	return c
}

// ---------------------------------------------------------------------
// the happy path
// ---------------------------------------------------------------------

func TestRoundTrip_AsksAndHearsTheAnswer(t *testing.T) {
	addr := serveRaw(t, func(conn net.Conn, req *http.Request) {
		accept101(conn, req)
		// Read the client's question, then answer it.
		if _, err := readMessage(bufio.NewReader(conn), DefaultMaxMessageBytes, true, nil); err != nil {
			return
		}
		_ = writeUnmasked(conn, []byte(`{"result":"0x171"}`))
	})

	got, err := RoundTrip(ctx(t), "ws://"+addr, []byte(`{"method":"eth_chainId"}`), nil)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	if string(got) != `{"result":"0x171"}` {
		t.Errorf("got %q", got)
	}
}

// The path and Host of the request line are what a reverse proxy routes on.
// An eRPC gateway serves each project on its own path, so a handshake that
// dropped the path would be answered by the wrong site block — or by none.
func TestRoundTrip_SendsThePathAndHostFromTheURL(t *testing.T) {
	seen := make(chan *http.Request, 1)
	addr := serveRaw(t, func(conn net.Conn, req *http.Request) {
		seen <- req
		accept101(conn, req)
		_, _ = readMessage(bufio.NewReader(conn), DefaultMaxMessageBytes, true, nil)
		_ = writeUnmasked(conn, []byte(`{}`))
	})

	if _, err := RoundTrip(ctx(t), "ws://"+addr+"/main/evm/369", []byte(`{}`), nil); err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	req := <-seen
	if req.URL.Path != "/main/evm/369" {
		t.Errorf("request path is %q, want /main/evm/369", req.URL.Path)
	}
	if req.Host != addr {
		t.Errorf("Host header is %q, want %q", req.Host, addr)
	}
}

// A caller that pinned the connection to an address must still send the NAME
// in the Host header. setup/tlsverify dials a verified IP and asks for
// gateway.valve.city; with the address in the Host header instead, Caddy
// matches no site block and answers 404 — a working gateway reported broken.
func TestHandshake_HostHeaderOverrideWins(t *testing.T) {
	seen := make(chan *http.Request, 1)
	addr := serveRaw(t, func(conn net.Conn, req *http.Request) {
		seen <- req
		accept101(conn, req)
		_, _ = readMessage(bufio.NewReader(conn), DefaultMaxMessageBytes, true, nil)
		_ = writeUnmasked(conn, []byte(`{}`))
	})

	raw, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer raw.Close()

	c, err := Handshake(ctx(t), raw, "gateway.valve.city", "/rpc", nil)
	if err != nil {
		t.Fatalf("Handshake: %v", err)
	}
	defer c.Close()

	req := <-seen
	if req.Host != "gateway.valve.city" {
		t.Errorf("Host header is %q, want the name we asked for, not the address we dialled", req.Host)
	}
	if req.URL.Path != "/rpc" {
		t.Errorf("request path is %q, want /rpc", req.URL.Path)
	}
}

// ---------------------------------------------------------------------
// refusals — the verdict these probes exist to produce
// ---------------------------------------------------------------------

// A definitive "no" must be distinguishable from "we could not tell", because
// the two produce different verdicts about the endpoint. A caller pattern
// matching on message text would be one reworded error away from silently
// reclassifying every refusal.
func TestHandshake_NonUpgradeStatusIsARefusal(t *testing.T) {
	addr := serveRaw(t, func(conn net.Conn, req *http.Request) {
		fmt.Fprint(conn, "HTTP/1.1 500 Internal Server Error\r\nContent-Length: 26\r\n\r\nunsupported content coding")
	})

	_, err := RoundTrip(ctx(t), "ws://"+addr, []byte(`{}`), nil)
	if !errors.Is(err, ErrRefused) {
		t.Fatalf("error %v is not ErrRefused", err)
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("error %q does not name the status", err)
	}
	// The BODY is the diagnostic. eRPC answers 500 to a WebSocket upgrade
	// whenever the client sends Accept-Encoding: gzip, and "HTTP 500" alone
	// gives an operator nothing to act on; the body says which 500 it is.
	if !strings.Contains(err.Error(), "unsupported content coding") {
		t.Errorf("error %q does not carry the response body", err)
	}
}

// The digest is the whole reason this is not just a status check. Anything can
// echo a 101; only something holding the RFC 6455 GUID can compute the accept
// token. Without this, a plain HTTP server that mirrored a 101 would be
// reported as a working WebSocket endpoint.
func TestHandshake_WrongAcceptDigestIsARefusal(t *testing.T) {
	addr := serveRaw(t, func(conn net.Conn, req *http.Request) {
		fmt.Fprint(conn, "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: obviously-wrong\r\n\r\n")
	})

	_, err := RoundTrip(ctx(t), "ws://"+addr, []byte(`{}`), nil)
	if !errors.Is(err, ErrRefused) {
		t.Fatalf("error %v is not ErrRefused", err)
	}
	if !strings.Contains(strings.ToLower(err.Error()), "accept") {
		t.Errorf("error %q does not name the accept digest as the reason", err)
	}
}

func TestHandshake_A101WithoutTheUpgradeHeaderIsARefusal(t *testing.T) {
	addr := serveRaw(t, func(conn net.Conn, req *http.Request) {
		sum := sha1.Sum([]byte(req.Header.Get("Sec-WebSocket-Key") + GUID)) //nolint:gosec — RFC 6455 defines SHA-1 here
		fmt.Fprintf(conn, "HTTP/1.1 101 Switching Protocols\r\nSec-WebSocket-Accept: %s\r\n\r\n",
			base64.StdEncoding.EncodeToString(sum[:]))
	})

	_, err := RoundTrip(ctx(t), "ws://"+addr, []byte(`{}`), nil)
	if !errors.Is(err, ErrRefused) {
		t.Fatalf("error %v is not ErrRefused", err)
	}
}

// MEASURED behaviour: an eRPC upstream whose scheme says http:// upgrades
// perfectly and then never answers. That is not a refusal — the endpoint said
// yes — so it gets its own sentinel, and reporting it as a refusal would send
// an operator to fix the wrong thing.
func TestRoundTrip_UpgradeThenSilenceIsNotARefusal(t *testing.T) {
	addr := serveRaw(t, func(conn net.Conn, req *http.Request) {
		accept101(conn, req)
		// and then nothing at all
	})

	_, err := RoundTrip(ctx(t), "ws://"+addr, []byte(`{}`), nil)
	if !errors.Is(err, ErrNoAnswer) {
		t.Fatalf("error %v is not ErrNoAnswer", err)
	}
	if errors.Is(err, ErrRefused) {
		t.Error("a silent-but-upgraded endpoint was classified as a refusal")
	}
}

// A close frame instead of an answer is what the published chain-943 endpoints
// do. Same verdict as silence: upgraded, then no answer.
func TestRoundTrip_CloseInsteadOfAnAnswerIsNoAnswer(t *testing.T) {
	addr := serveRaw(t, func(conn net.Conn, req *http.Request) {
		accept101(conn, req)
		_, _ = readMessage(bufio.NewReader(conn), DefaultMaxMessageBytes, true, nil)
		_, _ = conn.Write([]byte{0x88, 0x00}) // FIN | close, empty payload
	})

	_, err := RoundTrip(ctx(t), "ws://"+addr, []byte(`{}`), nil)
	if !errors.Is(err, ErrNoAnswer) {
		t.Fatalf("error %v is not ErrNoAnswer", err)
	}
}

// ---------------------------------------------------------------------
// dialling
// ---------------------------------------------------------------------

func TestDial_RejectsANonWebSocketScheme(t *testing.T) {
	if _, err := Dial(ctx(t), "https://example.invalid", nil); err == nil {
		t.Fatal("an https:// URL was accepted by a WebSocket dialler")
	}
}

func TestDial_UnreachableEndpointIsNotARefusal(t *testing.T) {
	// Port 1 on loopback: nothing listens, so the connection is refused at TCP
	// level. That is "we could not tell", not "this endpoint says no".
	_, err := Dial(ctx(t), "ws://127.0.0.1:1", nil)
	if err == nil {
		t.Fatal("dialling a closed port succeeded")
	}
	if errors.Is(err, ErrRefused) {
		t.Errorf("a TCP-level failure was reported as a WebSocket refusal: %v", err)
	}
}

// A cancelled context must unblock a read that would otherwise sit until the
// server felt like answering. Dial owns the socket it opened, so it is Dial's
// job to close it on cancellation.
func TestDial_CancellationUnblocksAPendingRead(t *testing.T) {
	addr := serveRaw(t, func(conn net.Conn, req *http.Request) {
		accept101(conn, req)
		select {} // never answers, never closes
	})

	c, cancel := context.WithCancel(context.Background())
	conn, err := Dial(c, "ws://"+addr, nil)
	if err != nil {
		cancel()
		t.Fatalf("Dial: %v", err)
	}
	defer conn.Close()

	done := make(chan error, 1)
	go func() {
		_, err := conn.ReadMessage()
		done <- err
	}()

	cancel()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("a read on a cancelled connection returned success")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("cancelling the context did not unblock the read")
	}
}

// ---------------------------------------------------------------------
// wss:// — the scheme production actually uses
// ---------------------------------------------------------------------

// tlsStub serves one hijacked WebSocket exchange over TLS and returns its
// host:port plus the pool that trusts it.
func tlsStub(t *testing.T, answer string) (string, *x509.CertPool) {
	t.Helper()
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hj, ok := w.(http.Hijacker)
		if !ok {
			return
		}
		conn, brw, err := hj.Hijack()
		if err != nil {
			return
		}
		defer conn.Close()
		sum := sha1.Sum([]byte(r.Header.Get("Sec-WebSocket-Key") + GUID)) //nolint:gosec — RFC 6455 defines SHA-1 here
		brw.WriteString("HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: " +
			base64.StdEncoding.EncodeToString(sum[:]) + "\r\n\r\n")
		brw.Flush()
		if _, err := readMessage(brw.Reader, DefaultMaxMessageBytes, true, nil); err != nil {
			return
		}
		_ = writeUnmasked(brw.Writer, []byte(answer))
		brw.Flush()
	}))
	srv.StartTLS()
	t.Cleanup(srv.Close)

	pool := x509.NewCertPool()
	pool.AddCert(srv.Certificate())
	return strings.TrimPrefix(srv.URL, "https://"), pool
}

// The whole point of a wss:// probe is that it runs over TLS, and a caller
// verifying a gateway supplies its own root pool — the gateway's certificate
// is minted by an internal CA no platform trusts.
func TestDial_SpeaksWSSWithACallerSuppliedRootPool(t *testing.T) {
	addr, pool := tlsStub(t, `{"result":"0x171"}`)

	got, err := RoundTrip(ctx(t), "wss://"+addr, []byte(`{}`), &Options{
		TLSConfig: &tls.Config{RootCAs: pool, ServerName: "example.com"},
	})
	if err != nil {
		t.Fatalf("RoundTrip over wss: %v", err)
	}
	if string(got) != `{"result":"0x171"}` {
		t.Errorf("got %q", got)
	}
}

// And without the pool it must fail. Otherwise the previous test proves
// nothing about verification — only that bytes moved.
func TestDial_WSSWithoutTheRootPoolIsRejected(t *testing.T) {
	addr, _ := tlsStub(t, `{"result":"0x171"}`)

	_, err := RoundTrip(ctx(t), "wss://"+addr, []byte(`{}`), nil)
	if err == nil {
		t.Fatal("a certificate from an untrusted CA was accepted")
	}
	if errors.Is(err, ErrRefused) {
		t.Errorf("a TLS verification failure was reported as a WebSocket refusal: %v", err)
	}
}

// A caller reading an open-ended stream — waiting for a subscription
// notification that may never arrive — decides for itself how long "never" is.
func TestConn_SetDeadlineBoundsAWaitForSomethingThatNeverComes(t *testing.T) {
	addr := serveRaw(t, func(conn net.Conn, req *http.Request) {
		accept101(conn, req)
		select {} // upgraded, then silent forever
	})

	c, err := Dial(ctx(t), "ws://"+addr, nil)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer c.Close()

	if err := c.SetDeadline(time.Now().Add(150 * time.Millisecond)); err != nil {
		t.Fatalf("SetDeadline: %v", err)
	}
	start := time.Now()
	if _, err := c.ReadMessage(); err == nil {
		t.Fatal("a read past its deadline returned success")
	}
	if elapsed := time.Since(start); elapsed > 3*time.Second {
		t.Errorf("the read took %s — the deadline did not bound it", elapsed)
	}
}

// writeUnmasked writes a server-side frame: same framing, no mask, because a
// server must not mask. Used only by these tests, to play the other end.
func writeUnmasked(w io.Writer, payload []byte) error {
	frame := []byte{0x81}
	switch n := len(payload); {
	case n < 126:
		frame = append(frame, byte(n))
	default:
		frame = append(frame, 126, byte(n>>8), byte(n))
	}
	_, err := w.Write(append(frame, payload...))
	return err
}
