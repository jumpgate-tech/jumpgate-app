// Package wsrpc speaks just enough RFC 6455 to ask a JSON-RPC endpoint a
// question over a WebSocket and hear the answer.
//
// It is deliberately not a general WebSocket library. It does the opening
// handshake, writes masked text frames, reads data messages, and stops. What
// it is FOR is telling apart three things an operator sees as one:
//
//   - an endpoint that speaks WebSocket,
//   - an endpoint that refuses the upgrade (ErrRefused),
//   - an endpoint that accepts the upgrade and then never answers
//     (ErrNoAnswer).
//
// The third is not hypothetical. eRPC infers WebSocket capability from an
// upstream's URL SCHEME alone, so a gateway pointed at an http:// upstream
// upgrades happily, serves eth_chainId, and refuses every eth_subscribe. It is
// also the measured behaviour of rpc.pulsechain.com and of all four published
// chain-943 endpoints. A probe that reported only "reachable" would call every
// one of those healthy.
//
// This code was written three times before it was written here — once each in
// internal/chainlist, internal/capabilities and internal/setup — and the three
// copies had already drifted apart in their frame caps, their error detail and
// their handshake signatures. That is the drift this package exists to end.
package wsrpc

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/sha1"
	"crypto/tls"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// GUID is the RFC 6455 §1.3 magic value mixed into the Sec-WebSocket-Accept
// digest.
//
// It is exported because the other side of the handshake needs it too: the
// packages that use this client also run hand-rolled WebSocket servers in
// their tests, and a private copy in each of them would be the same
// duplication this package exists to remove. It is a published constant from
// the RFC, not an implementation detail.
const GUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

// ErrRefused marks a definitive refusal: the server answered the upgrade and
// said no. It is a sentinel rather than a string to pattern-match because the
// difference between "this endpoint does not do WebSocket" and "we could not
// tell" is the verdict itself, and a caller matching on wording would be one
// reworded message away from silently reclassifying every refusal.
var ErrRefused = errors.New("websocket upgrade refused")

// ErrNoAnswer marks a handshake that succeeded followed by a socket that
// produced nothing. Load-bearing for the same reason: an upstream that infers
// WebSocket from a URL scheme upgrades and then goes quiet, and calling that a
// refusal sends an operator to fix the wrong end.
var ErrNoAnswer = errors.New("websocket answered nothing")

// Options tune a connection. The zero value is usable; nil is the zero value.
type Options struct {
	// Dialer opens the TCP connection. nil means a default dialer.
	Dialer *net.Dialer

	// TLSConfig is used for wss://. nil means ServerName taken from the URL,
	// with the platform roots. A caller pinning to its own root pool — as
	// gateway TLS verification does — passes one here, or dials itself and
	// calls Handshake.
	TLSConfig *tls.Config

	// MaxMessageBytes caps an assembled message. 0 means
	// DefaultMaxMessageBytes.
	MaxMessageBytes int64

	// HostHeader overrides the Host: sent in the upgrade request. "" means the
	// URL's host. It matters when the connection is pinned to an address but
	// the server routes on a name: send the address and a name-based reverse
	// proxy matches no site and answers 404.
	HostHeader string
}

func (o *Options) dialer() *net.Dialer {
	if o != nil && o.Dialer != nil {
		return o.Dialer
	}
	return &net.Dialer{}
}

func (o *Options) maxMessageBytes() int64 {
	if o != nil && o.MaxMessageBytes > 0 {
		return o.MaxMessageBytes
	}
	return DefaultMaxMessageBytes
}

func (o *Options) tlsConfig(serverName string) *tls.Config {
	if o != nil && o.TLSConfig != nil {
		return o.TLSConfig
	}
	return &tls.Config{ServerName: serverName} //nolint:gosec — MinVersion is the Go default, which is TLS 1.2
}

// Conn is a WebSocket connection past its opening handshake.
//
// It is not safe for concurrent use. Nothing in this tree needs that: every
// caller is one goroutine asking one endpoint one thing.
type Conn struct {
	conn net.Conn
	br   *bufio.Reader
	max  int64

	// isServer marks a Conn Accept built, on the server side of the
	// handshake. It picks the mask direction for every frame this Conn
	// reads and writes: RFC 6455 requires a server to send unmasked frames
	// and demand masked ones back, and a client the exact opposite.
	isServer bool

	// stop ends the context watchdog Dial installs. nil when the caller
	// dialled and therefore owns the connection's lifetime themselves.
	stop chan struct{}
}

// newServerConn builds a Conn for the SERVER side of a handshake Accept just
// completed, over a connection and buffered reader Accept already has —
// typically from http.Hijacker.
//
// It reuses the same frame reader and writer the client path uses, so
// continuation frames, control frames and the message cap behave identically
// on both ends of this one package. A second, hand-rolled server frame
// implementation is exactly the drift this package exists to end.
func newServerConn(conn net.Conn, br *bufio.Reader, max int64) *Conn {
	return &Conn{conn: conn, br: br, max: max, isServer: true}
}

// Dial opens a connection to rawURL (ws:// or wss://) and completes the
// opening handshake.
//
// Dial owns the socket it opened, so it watches ctx for the whole life of the
// connection: a deadline goes onto the socket, and a cancellation closes it so
// a blocked read unblocks. Conn.Close stops the watchdog.
func Dial(ctx context.Context, rawURL string, opt *Options) (*Conn, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("bad URL: %w", err)
	}
	secure := strings.EqualFold(u.Scheme, "wss")
	if !secure && !strings.EqualFold(u.Scheme, "ws") {
		return nil, fmt.Errorf("not a websocket URL: scheme %q is neither ws nor wss", u.Scheme)
	}

	addr := u.Host
	if u.Port() == "" {
		if secure {
			addr = net.JoinHostPort(u.Hostname(), "443")
		} else {
			addr = net.JoinHostPort(u.Hostname(), "80")
		}
	}

	conn, err := opt.dialer().DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, err
	}

	if dl, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(dl)
	}
	stop := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			_ = conn.Close()
		case <-stop:
		}
	}()
	// Any failure from here on must not leak the watchdog or the socket.
	fail := func(err error) (*Conn, error) {
		close(stop)
		_ = conn.Close()
		return nil, err
	}

	if secure {
		tconn := tls.Client(conn, opt.tlsConfig(u.Hostname()))
		if err := tconn.HandshakeContext(ctx); err != nil {
			return fail(err)
		}
		conn = tconn
	}

	host := u.Host
	if opt != nil && opt.HostHeader != "" {
		host = opt.HostHeader
	}
	c, err := handshake(conn, host, u.RequestURI(), opt.maxMessageBytes())
	if err != nil {
		return fail(err)
	}
	c.stop = stop
	return c, nil
}

// Handshake completes the opening handshake over a connection the caller has
// already dialled — including one already wrapped in TLS.
//
// It exists for callers that must control the dial themselves: gateway TLS
// verification pins the connection to a verified certificate pool and a fixed
// address, which no URL can express. Such a caller keeps ownership of the
// socket, so Handshake installs no context watchdog; ctx is honoured only as a
// deadline, and the caller's own deadline discipline governs after that.
func Handshake(ctx context.Context, conn net.Conn, host, path string, opt *Options) (*Conn, error) {
	if dl, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(dl)
	}
	return handshake(conn, host, path, opt.maxMessageBytes())
}

// RoundTrip is the whole exchange for a caller that just has a URL and a
// question: dial, ask, hear one answer, hang up.
//
// Errors past a successful upgrade are wrapped in ErrNoAnswer, so a caller can
// tell a refusal from a silence without reading strings.
func RoundTrip(ctx context.Context, rawURL string, payload []byte, opt *Options) ([]byte, error) {
	c, err := Dial(ctx, rawURL, opt)
	if err != nil {
		return nil, err
	}
	defer c.Close()

	if err := c.WriteText(payload); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrNoAnswer, err)
	}
	msg, err := c.ReadMessage()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrNoAnswer, err)
	}
	return msg, nil
}

// handshake performs the opening HTTP Upgrade and verifies the server's accept
// digest.
//
// The digest check is the point. Anything can echo a 101; only a peer holding
// the RFC 6455 GUID can compute the accept token from our nonce. Without it we
// would happily "talk WebSocket" to a plain HTTP server that mirrored a 101 by
// accident — which is precisely the confusion these probes exist to detect.
func handshake(conn net.Conn, host, path string, max int64) (*Conn, error) {
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return nil, fmt.Errorf("websocket: nonce: %w", err)
	}
	key := base64.StdEncoding.EncodeToString(nonce[:])
	if path == "" {
		path = "/"
	}

	req := "GET " + path + " HTTP/1.1\r\n" +
		"Host: " + host + "\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Key: " + key + "\r\n" +
		"Sec-WebSocket-Version: 13\r\n\r\n"
	if _, err := io.WriteString(conn, req); err != nil {
		return nil, err
	}

	// http.ReadResponse understands the 101 and leaves the frame stream
	// untouched in br, which is exactly the split we need.
	br := bufio.NewReader(conn)
	resp, err := http.ReadResponse(br, &http.Request{Method: http.MethodGet})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusSwitchingProtocols {
		// The BODY, not just the status. eRPC answers 500 to an upgrade
		// whenever the client sends Accept-Encoding: gzip — which every
		// reverse proxy does — and "HTTP 500" on its own gives an operator
		// nothing to act on.
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
		if detail := truncate(strings.TrimSpace(string(body)), 200); detail != "" {
			return nil, fmt.Errorf("%w: HTTP %s: %s", ErrRefused, resp.Status, detail)
		}
		return nil, fmt.Errorf("%w: HTTP %s", ErrRefused, resp.Status)
	}
	if !strings.EqualFold(resp.Header.Get("Upgrade"), "websocket") {
		return nil, fmt.Errorf("%w: 101 without Upgrade: websocket", ErrRefused)
	}
	sum := sha1.Sum([]byte(key + GUID)) //nolint:gosec — the accept token is defined as SHA-1 by RFC 6455
	if want := base64.StdEncoding.EncodeToString(sum[:]); resp.Header.Get("Sec-WebSocket-Accept") != want {
		return nil, fmt.Errorf("%w: bad Sec-WebSocket-Accept digest", ErrRefused)
	}
	return &Conn{conn: conn, br: br, max: max}, nil
}

// WriteText sends payload as one text frame — masked if this Conn is a
// client, unmasked if Accept built it as a server. RFC 6455 §5.1 requires
// exactly one of those for each side and forbids the other.
func (c *Conn) WriteText(payload []byte) error {
	return writeFrame(c.conn, opcodeText, payload, !c.isServer)
}

// ReadMessage returns the next complete data message, reassembling
// continuation frames and skipping control frames — except that on the
// server side, a ping gets a pong back first, as RFC 6455 §5.5.2 requires.
func (c *Conn) ReadMessage() ([]byte, error) {
	var onPing func([]byte) error
	if c.isServer {
		onPing = c.WritePong
	}
	return readMessage(c.br, c.max, c.isServer, onPing)
}

// WritePong answers a ping with a pong carrying the same payload, per RFC
// 6455 §5.5.3. Exported so a caller driving its own read loop — rather than
// ReadMessage's automatic reply — can still answer one.
func (c *Conn) WritePong(payload []byte) error {
	return writeFrame(c.conn, opcodePong, payload, !c.isServer)
}

// WriteClose sends a close frame carrying a status code and a reason, per RFC
// 6455 §5.5.1 and §7.4. A relay uses this to end a customer's connection on
// purpose, instead of just dropping the TCP socket under it.
func (c *Conn) WriteClose(code uint16, reason string) error {
	payload := make([]byte, 2+len(reason))
	binary.BigEndian.PutUint16(payload, code)
	copy(payload[2:], reason)
	return writeFrame(c.conn, opcodeClose, payload, !c.isServer)
}

// SetDeadline bounds the next reads and writes. A caller reading an open-ended
// stream — waiting for a subscription notification that may never come — uses
// this to decide how long "never" is.
func (c *Conn) SetDeadline(t time.Time) error { return c.conn.SetDeadline(t) }

// Close releases the connection and stops the context watchdog, if Dial
// installed one. It is safe to call more than once.
func (c *Conn) Close() error {
	if c.stop != nil {
		close(c.stop)
		c.stop = nil
	}
	return c.conn.Close()
}

// truncate shortens s for an error message, marking that it did so — an
// elided message that looks complete is worse than an obviously clipped one.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
