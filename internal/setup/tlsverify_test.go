package setup

// The live-HTTPS verification, exercised against a REAL TLS server with a real
// private CA — not a fake — because every assertion it makes is about wire
// behaviour (a handshake, a certificate chain, a WebSocket upgrade) that a fake
// would have to reproduce in order to test, at which point the test is testing
// the fake.

import (
	"bufio"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha1"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/valve-tech/valve-node-app/internal/wsrpc"
)

const (
	verifyHost = "gw.verify.test"
	// verifyPath is the gateway path the probe is pointed at. The stub routes
	// on it, so a probe that drops the path is caught rather than tolerated.
	verifyPath = "/main/evm/1337"
)

// gatewayStub is a stand-in for Caddy-in-front-of-eRPC: it answers
// eth_chainId over HTTPS and speaks enough WebSocket to accept or refuse an
// eth_subscribe.
type gatewayStub struct {
	chainID int
	// subscribeErr, when set, is the JSON-RPC error eth_subscribe is answered
	// with — the measured http:// upstream behaviour assertion 5 exists for.
	subscribeErr string
	// noHeads suppresses the newHeads notification, leaving only the
	// subscription id.
	noHeads bool
}

func (g gatewayStub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
		g.serveWS(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	io.WriteString(w, `{"jsonrpc":"2.0","id":1,"result":"`+hexQuantity(g.chainID)+`"}`)
}

func (g gatewayStub) serveWS(w http.ResponseWriter, r *http.Request) {
	// Caddy routes on the Host header and the path, and answers 404 when
	// neither matches a site block. The stub does the same, so that sending the
	// pinned ADDRESS instead of the name — or dropping the path — fails here
	// rather than silently passing against a stub that answers anything.
	// This is the measured production failure mode: a verified certificate, a
	// working gateway, and a 404 because the upgrade asked for the wrong name.
	if host, _, err := net.SplitHostPort(r.Host); err != nil || host != verifyHost {
		http.Error(w, "no site block for Host "+r.Host, http.StatusNotFound)
		return
	}
	if r.URL.Path != verifyPath {
		http.Error(w, "no route for path "+r.URL.Path, http.StatusNotFound)
		return
	}

	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "no hijacker", http.StatusInternalServerError)
		return
	}
	sum := sha1.Sum([]byte(r.Header.Get("Sec-WebSocket-Key") + wsrpc.GUID))
	conn, brw, err := hj.Hijack()
	if err != nil {
		return
	}
	defer conn.Close()
	brw.WriteString("HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: " +
		base64.StdEncoding.EncodeToString(sum[:]) + "\r\n\r\n")
	brw.Flush()

	if _, err := readClientFrame(brw.Reader); err != nil {
		return
	}
	if g.subscribeErr != "" {
		writeServerFrame(brw.Writer, `{"jsonrpc":"2.0","id":1,"error":{"code":-32601,"message":"`+g.subscribeErr+`"}}`)
		brw.Flush()
		return
	}
	writeServerFrame(brw.Writer, `{"jsonrpc":"2.0","id":1,"result":"0xsub1"}`)
	brw.Flush()
	if g.noHeads {
		// Hold the connection open so the client waits for a head that never
		// comes, which is the "subscribed, chain idle" case.
		time.Sleep(200 * time.Millisecond)
		return
	}
	writeServerFrame(brw.Writer, `{"jsonrpc":"2.0","method":"eth_subscription","params":{"subscription":"0xsub1","result":{"number":"0x1"}}}`)
	brw.Flush()
	time.Sleep(50 * time.Millisecond)
}

// writeServerFrame writes one unmasked text frame, which is what a server
// sends (only clients mask).
func writeServerFrame(w io.Writer, payload string) {
	frame := []byte{0x81}
	n := len(payload)
	switch {
	case n < 126:
		frame = append(frame, byte(n))
	default:
		frame = append(frame, 126, byte(n>>8), byte(n))
	}
	frame = append(frame, payload...)
	w.Write(frame)
}

// readClientFrame reads one masked text frame, which is what a client sends.
// It is the server half of the exchange and belongs to this test, not to
// internal/wsrpc: wsrpc is a client, and a stub server written against its
// internals would stop being an independent check of it.
func readClientFrame(br *bufio.Reader) ([]byte, error) {
	var hdr [2]byte
	if _, err := io.ReadFull(br, hdr[:]); err != nil {
		return nil, err
	}
	length := int(hdr[1] & 0x7f)
	switch length {
	case 126:
		var ext [2]byte
		if _, err := io.ReadFull(br, ext[:]); err != nil {
			return nil, err
		}
		length = int(ext[0])<<8 | int(ext[1])
	case 127:
		return nil, errors.New("this stub does not expect a 64-bit length")
	}
	var mask [4]byte
	masked := hdr[1]&0x80 != 0
	if masked {
		if _, err := io.ReadFull(br, mask[:]); err != nil {
			return nil, err
		}
	}
	payload := make([]byte, length)
	if _, err := io.ReadFull(br, payload); err != nil {
		return nil, err
	}
	if !masked {
		return nil, errors.New("the client sent an unmasked frame, which RFC 6455 §5.3 forbids")
	}
	for i := range payload {
		payload[i] ^= mask[i%4]
	}
	return payload, nil
}

func hexQuantity(n int) string { return "0x" + big.NewInt(int64(n)).Text(16) }

// testCA mints a private CA and a leaf for names, mirroring what Caddy's
// internal CA produces: a root the operator is expected to install, and a leaf
// that only verifies against it.
func testCA(t *testing.T, names ...string) (*x509.CertPool, tls.Certificate) {
	t.Helper()

	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("ca key: %v", err)
	}
	caTmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "valve-node-app test CA"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTmpl, caTmpl, &caKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("ca cert: %v", err)
	}
	caCert, err := x509.ParseCertificate(caDER)
	if err != nil {
		t.Fatalf("parse ca: %v", err)
	}

	leafKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("leaf key: %v", err)
	}
	leafTmpl := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: names[0]},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(12 * time.Hour),
		DNSNames:     names,
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	leafDER, err := x509.CreateCertificate(rand.Reader, leafTmpl, caCert, &leafKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("leaf cert: %v", err)
	}

	pool := x509.NewCertPool()
	pool.AddCert(caCert)
	return pool, tls.Certificate{Certificate: [][]byte{leafDER, caDER}, PrivateKey: leafKey}
}

// startStub serves g over TLS with cert, and returns the probe pointed at it.
func startStub(t *testing.T, g gatewayStub, cert tls.Certificate, roots *x509.CertPool) tlsProbe {
	t.Helper()
	srv := httptest.NewUnstartedServer(g)
	srv.TLS = &tls.Config{Certificates: []tls.Certificate{cert}}
	srv.StartTLS()
	t.Cleanup(srv.Close)

	addr := strings.TrimPrefix(srv.URL, "https://")
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatalf("split %q: %v", addr, err)
	}
	p := tlsProbe{
		Hostname:    verifyHost,
		Address:     addr,
		Roots:       roots,
		TrustSource: "the test CA",
		Path:        verifyPath,
		ChainID:     g.chainID,
		CertSource:  "internal",
	}
	p.Port = atoi(t, port)
	return p
}

func atoi(t *testing.T, s string) int {
	t.Helper()
	n, ok := new(big.Int).SetString(s, 10)
	if !ok {
		t.Fatalf("port %q is not a number", s)
	}
	return int(n.Int64())
}

func assertionOf(t *testing.T, v TLSVerification, id string) TLSAssertion {
	t.Helper()
	for _, a := range v.Assertions {
		if a.ID == id {
			return a
		}
	}
	t.Fatalf("no assertion %q in %+v", id, v.Assertions)
	return TLSAssertion{}
}

func wantStatus(t *testing.T, v TLSVerification, id, want string) {
	t.Helper()
	got := assertionOf(t, v, id)
	if got.Status != want {
		t.Errorf("assertion %q: status %q, want %q (%s)", id, got.Status, want, got.Detail)
	}
}

// A front that is genuinely serving passes every assertion, and reports when
// the certificate runs out — which is the fact that is only useful early.
func TestVerifyTLSEndpointAllPass(t *testing.T) {
	roots, cert := testCA(t, verifyHost)
	p := startStub(t, gatewayStub{chainID: 1337}, cert, roots)

	v := verifyTLSEndpoint(context.Background(), p)

	wantStatus(t, v, TLSAssertHandshake, TLSStatusPass)
	wantStatus(t, v, TLSAssertHostname, TLSStatusPass)
	wantStatus(t, v, TLSAssertChain, TLSStatusPass)
	wantStatus(t, v, TLSAssertRPC, TLSStatusPass)
	wantStatus(t, v, TLSAssertSubscribe, TLSStatusPass)
	wantStatus(t, v, TLSAssertPlaintext, TLSStatusPass)
	if !v.OK || !v.SubscriptionsOK {
		t.Errorf("ok=%v subscriptionsOk=%v, want both true: %s", v.OK, v.SubscriptionsOK, v.Summary)
	}
	if v.NotAfter == nil || v.ExpiresIn == "" {
		t.Errorf("expiry not reported: notAfter=%v expiresIn=%q", v.NotAfter, v.ExpiresIn)
	}
	if v.ExpiryWarning != "" {
		t.Errorf("an internal-CA leaf must not raise an expiry warning (it is renewed in process): %q", v.ExpiryWarning)
	}
}

// The failure this whole feature is for: a perfectly valid certificate for the
// WRONG name. Everything below it is skipped rather than passed, because a
// call through an unverified connection proves nothing.
func TestVerifyTLSEndpointHostnameMismatch(t *testing.T) {
	roots, cert := testCA(t, "someone.else.test")
	p := startStub(t, gatewayStub{chainID: 1337}, cert, roots)

	v := verifyTLSEndpoint(context.Background(), p)

	wantStatus(t, v, TLSAssertHandshake, TLSStatusPass)
	wantStatus(t, v, TLSAssertHostname, TLSStatusFail)
	wantStatus(t, v, TLSAssertChain, TLSStatusPass)
	wantStatus(t, v, TLSAssertRPC, TLSStatusSkip)
	wantStatus(t, v, TLSAssertSubscribe, TLSStatusSkip)
	if v.OK {
		t.Error("a certificate for the wrong name must not be reported as OK")
	}
	if d := assertionOf(t, v, TLSAssertHostname).Detail; !strings.Contains(d, "someone.else.test") {
		t.Errorf("the mismatch must name what the certificate DOES cover, got %q", d)
	}
}

// A chain that does not verify against the expected root is a failure, not a
// warning — this is the assertion that would be meaningless with -k.
func TestVerifyTLSEndpointUntrustedChain(t *testing.T) {
	_, cert := testCA(t, verifyHost)
	p := startStub(t, gatewayStub{chainID: 1337}, cert, x509.NewCertPool())

	v := verifyTLSEndpoint(context.Background(), p)

	wantStatus(t, v, TLSAssertHandshake, TLSStatusPass)
	wantStatus(t, v, TLSAssertHostname, TLSStatusPass)
	wantStatus(t, v, TLSAssertChain, TLSStatusFail)
	wantStatus(t, v, TLSAssertRPC, TLSStatusSkip)
	if v.OK {
		t.Error("an unverifiable chain must not be reported as OK")
	}
}

// A gateway serving a DIFFERENT chain on the path is a misroute, and it is
// invisible to every certificate check.
func TestVerifyTLSEndpointWrongChain(t *testing.T) {
	roots, cert := testCA(t, verifyHost)
	p := startStub(t, gatewayStub{chainID: 1337}, cert, roots)
	p.ChainID = 369

	v := verifyTLSEndpoint(context.Background(), p)

	wantStatus(t, v, TLSAssertRPC, TLSStatusFail)
	if d := assertionOf(t, v, TLSAssertRPC).Detail; !strings.Contains(d, "1337") {
		t.Errorf("the detail must name the chain that answered, got %q", d)
	}
}

// THE case assertion 5 exists for: HTTPS is perfect and subscriptions are
// refused. It must not read as a healthy gateway, and it must not read as a
// broken HTTPS front either.
func TestVerifyTLSEndpointSubscriptionsUnavailable(t *testing.T) {
	roots, cert := testCA(t, verifyHost)
	p := startStub(t, gatewayStub{chainID: 1337, subscribeErr: "no upstream with websocket support"}, cert, roots)

	v := verifyTLSEndpoint(context.Background(), p)

	wantStatus(t, v, TLSAssertRPC, TLSStatusPass)
	wantStatus(t, v, TLSAssertSubscribe, TLSStatusUnavailable)
	if !v.OK {
		t.Error("HTTPS itself is fine here and must still be reported OK")
	}
	if v.SubscriptionsOK {
		t.Error("subscriptions are refused, so subscriptionsOk must be false")
	}
	if !strings.Contains(v.Summary, "SUBSCRIPTIONS ARE NOT AVAILABLE") {
		t.Errorf("the summary has to say it, got %q", v.Summary)
	}
}

// An accepted subscription with no block behind it is a pass with a caveat,
// not a failure: the chain being idle is not the gateway's fault.
func TestVerifyTLSEndpointSubscribedButIdle(t *testing.T) {
	roots, cert := testCA(t, verifyHost)
	p := startStub(t, gatewayStub{chainID: 1337, noHeads: true}, cert, roots)

	v := verifyTLSEndpoint(context.Background(), p)

	got := assertionOf(t, v, TLSAssertSubscribe)
	if got.Status != TLSStatusPass || !strings.Contains(got.Detail, "no newHeads") {
		t.Errorf("want a pass that says no head arrived, got %s: %s", got.Status, got.Detail)
	}
}

// A plaintext server on the TLS port is a real outcome and has to be named as
// one, rather than reported as a generic connection failure.
func TestVerifyTLSEndpointPlaintextOnTLSPort(t *testing.T) {
	srv := httptest.NewServer(gatewayStub{chainID: 1337})
	defer srv.Close()

	addr := strings.TrimPrefix(srv.URL, "http://")
	_, port, _ := net.SplitHostPort(addr)
	p := tlsProbe{
		Hostname: verifyHost,
		Port:     atoi(t, port),
		Address:  addr,
		Roots:    x509.NewCertPool(),
		Path:     "/main/evm/1337",
		ChainID:  1337,
	}

	v := verifyTLSEndpoint(context.Background(), p)

	wantStatus(t, v, TLSAssertHandshake, TLSStatusFail)
	if d := assertionOf(t, v, TLSAssertHandshake).Detail; !strings.Contains(d, "plaintext") {
		t.Errorf("the detail must say it looks like plaintext, got %q", d)
	}
	// And the negative assertion catches the same fact from the other side.
	wantStatus(t, v, TLSAssertPlaintext, TLSStatusFail)
}

// Nothing listening is a handshake failure that names the address, so the
// operator can tell it from a certificate problem.
func TestVerifyTLSEndpointNothingListening(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := l.Addr().String()
	l.Close()

	_, port, _ := net.SplitHostPort(addr)
	p := tlsProbe{Hostname: verifyHost, Port: atoi(t, port), Address: addr, Path: "/main/evm/1337", ChainID: 1337}

	v := verifyTLSEndpoint(context.Background(), p)

	wantStatus(t, v, TLSAssertHandshake, TLSStatusFail)
	wantStatus(t, v, TLSAssertHostname, TLSStatusSkip)
	wantStatus(t, v, TLSAssertChain, TLSStatusSkip)
	if v.OK {
		t.Error("a dead endpoint must not be OK")
	}
}

// The eth_chainId reader has to tell "not eRPC" from "eRPC said no", because
// they are different problems with different fixes.
func TestPostChainIDNonJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "<html>not an rpc endpoint</html>")
	}))
	defer srv.Close()

	_, err := postChainID(context.Background(), srv.Client(), srv.URL)
	if err == nil || !strings.Contains(err.Error(), "not JSON") {
		t.Fatalf("want a 'not JSON' error, got %v", err)
	}
}

func TestPostChainIDRPCError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"error": map[string]any{"message": "no upstream"}})
	}))
	defer srv.Close()

	_, err := postChainID(context.Background(), srv.Client(), srv.URL)
	if err == nil || !strings.Contains(err.Error(), "no upstream") {
		t.Fatalf("want the upstream error passed through, got %v", err)
	}
}
