package wsrpc

// The server side of the RFC 6455 opening handshake — accepting an inbound
// upgrade, rather than dialling one out. A future relay uses this to
// terminate a customer's WebSocket connection directly instead of only ever
// speaking WebSocket outward to an upstream.

import (
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// ErrBadUpgrade marks a request Accept refused because it was not a valid
// WebSocket upgrade — wrong method, a missing or wrong header, an
// unsupported version. It is a sentinel, the same way ErrRefused is on the
// client side, so a caller can act on the verdict without matching on
// wording.
var ErrBadUpgrade = errors.New("websocket: not a valid upgrade request")

// Accept performs the server side of the RFC 6455 opening handshake: it
// validates r, hijacks the underlying connection, and writes the 101
// response by hand.
//
// On a bad request, Accept writes an ordinary HTTP error response through w
// and returns before hijacking. It must not hijack a connection it is about
// to reject — the client is still expecting a normal HTTP response, and
// hijacking would leave its socket in an undefined state instead.
func Accept(w http.ResponseWriter, r *http.Request, opt *Options) (*Conn, error) {
	if err := validateUpgrade(r); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return nil, err
	}

	sum := sha1.Sum([]byte(r.Header.Get("Sec-WebSocket-Key") + GUID)) //nolint:gosec — the accept token is defined as SHA-1 by RFC 6455
	accept := base64.StdEncoding.EncodeToString(sum[:])

	hj, ok := w.(http.Hijacker)
	if !ok {
		err := fmt.Errorf("%w: the response writer does not support hijacking", ErrBadUpgrade)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return nil, err
	}
	conn, brw, err := hj.Hijack()
	if err != nil {
		return nil, fmt.Errorf("websocket: hijack: %w", err)
	}

	// From here the response goes straight to the wire, not through w — a
	// hijacked ResponseWriter cannot be used again. Nothing here reads or
	// echoes Accept-Encoding, on purpose: eRPC used to answer HTTP 500 on a
	// WebSocket upgrade whenever the client sent Accept-Encoding: gzip,
	// because its gzip negotiation ran ahead of the upgrade check. Every
	// reverse proxy adds that header automatically, so that bug hit real
	// traffic. Content encoding has nothing to do with upgrading a
	// connection, and it must never be given a vote here again.
	resp := "HTTP/1.1 101 Switching Protocols\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Accept: " + accept + "\r\n\r\n"
	if _, err := brw.WriteString(resp); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("websocket: writing the 101 response: %w", err)
	}
	if err := brw.Flush(); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("websocket: flushing the 101 response: %w", err)
	}

	return newServerConn(conn, brw.Reader, opt.maxMessageBytes()), nil
}

// validateUpgrade checks r against RFC 6455 §4.2.1: GET, a Connection header
// naming "upgrade", an Upgrade header naming "websocket", version 13, and a
// present Sec-WebSocket-Key. It deliberately never looks at Accept-Encoding —
// see the comment in Accept for why that header must stay out of this
// decision.
func validateUpgrade(r *http.Request) error {
	if r.Method != http.MethodGet {
		return fmt.Errorf("%w: method is %s, want GET", ErrBadUpgrade, r.Method)
	}
	if !hasToken(r.Header.Get("Connection"), "upgrade") {
		return fmt.Errorf("%w: Connection header %q does not name upgrade", ErrBadUpgrade, r.Header.Get("Connection"))
	}
	if !strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
		return fmt.Errorf("%w: Upgrade header is %q, want websocket", ErrBadUpgrade, r.Header.Get("Upgrade"))
	}
	if r.Header.Get("Sec-WebSocket-Version") != "13" {
		return fmt.Errorf("%w: Sec-WebSocket-Version is %q, want 13", ErrBadUpgrade, r.Header.Get("Sec-WebSocket-Version"))
	}
	if r.Header.Get("Sec-WebSocket-Key") == "" {
		return fmt.Errorf("%w: missing Sec-WebSocket-Key", ErrBadUpgrade)
	}
	return nil
}

// hasToken reports whether list — a comma-separated header value — names
// token, matched case-insensitively per RFC 7230 §3.2.6.
//
// A straight string compare against "Upgrade" would reject a real client:
// "Connection: keep-alive, Upgrade" is legal and is what a browser through a
// proxy actually sends.
func hasToken(list, token string) bool {
	for _, part := range strings.Split(list, ",") {
		if strings.EqualFold(strings.TrimSpace(part), token) {
			return true
		}
	}
	return false
}
