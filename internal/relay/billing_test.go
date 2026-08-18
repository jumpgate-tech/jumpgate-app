package relay

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// The billing client talks to the Rust key store over a unix socket. These
// tests drive a REAL http server over a REAL unix socket, because the transport
// is half the point: a TCP loopback port can be squatted by any local process,
// and a socket file cannot.

// billingStub records what the client sent and answers with what the test asks
// for. It is a real HTTP server, not a mock object.
type billingStub struct {
	srv      *httptest.Server
	socket   string
	gotPath  string
	gotAuth  string
	gotBody  string
	gotQuery string
	status   int
	body     string
}

func newBillingStub(t *testing.T) *billingStub {
	t.Helper()
	// A unix socket path is capped near 104 bytes on macOS and 108 on Linux.
	// t.TempDir() embeds the test name under an already long TMPDIR and blows
	// past that, so the socket goes in a short directory of its own.
	dir, err := os.MkdirTemp("/tmp", "jgr")
	if err != nil {
		t.Fatalf("temp dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	sock := filepath.Join(dir, "b.sock")

	st := &billingStub{socket: sock, status: http.StatusOK, body: `{}`}
	ln, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatalf("listen unix: %v", err)
	}
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		st.gotPath = r.URL.Path
		st.gotQuery = r.URL.RawQuery
		st.gotAuth = r.Header.Get("Authorization")
		b, _ := io.ReadAll(r.Body)
		st.gotBody = string(b)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(st.status)
		io.WriteString(w, st.body)
	}))
	srv.Listener.Close()
	srv.Listener = ln
	srv.Start()
	t.Cleanup(srv.Close)
	st.srv = srv
	return st
}

func TestBillingClientAuthenticatesOverUnixSocket(t *testing.T) {
	stub := newBillingStub(t)
	stub.body = `{"id":"k1","label":"prod","enabled":true,"allow_trace":false,
	              "method_block":["debug_traceCall"],"per_second_limit":50}`

	c := NewBillingClient(stub.socket, "relay-token")
	rec, err := c.Authenticate(context.Background(), "jg_secret")
	if err != nil {
		t.Fatalf("Authenticate: %v", err)
	}
	if rec.ID != "k1" {
		t.Errorf("id = %q, want k1", rec.ID)
	}
	if !rec.Enabled {
		t.Error("Enabled = false, want true")
	}
	if len(rec.MethodBlock) != 1 || rec.MethodBlock[0] != "debug_traceCall" {
		t.Errorf("MethodBlock = %v, want [debug_traceCall]", rec.MethodBlock)
	}
	if rec.PerSecondLimit != 50 {
		t.Errorf("PerSecondLimit = %d, want 50", rec.PerSecondLimit)
	}
}

// The relay sends its OWN credential, which authorises exactly one operation.
// The admin token would authorise minting keys and rewriting prices, so it must
// never reach this path.
func TestBillingClientSendsTheRelayToken(t *testing.T) {
	stub := newBillingStub(t)
	stub.body = `{"id":"k1","enabled":true}`

	c := NewBillingClient(stub.socket, "relay-token")
	if _, err := c.Authenticate(context.Background(), "jg_secret"); err != nil {
		t.Fatalf("Authenticate: %v", err)
	}
	if want := "Bearer relay-token"; stub.gotAuth != want {
		t.Errorf("Authorization = %q, want %q", stub.gotAuth, want)
	}
	if stub.gotPath != "/internal/authenticate" {
		t.Errorf("path = %q, want /internal/authenticate", stub.gotPath)
	}
}

// The raw key travels in the request BODY, never in the URL. A key in a URL
// lands in whatever access log the store keeps, which is the exact leak the
// relay exists to prevent one hop earlier.
func TestBillingClientNeverPutsTheKeyInTheURL(t *testing.T) {
	stub := newBillingStub(t)
	stub.body = `{"id":"k1","enabled":true}`

	c := NewBillingClient(stub.socket, "relay-token")
	if _, err := c.Authenticate(context.Background(), "jg_verysecret"); err != nil {
		t.Fatalf("Authenticate: %v", err)
	}
	if strings.Contains(stub.gotPath, "jg_verysecret") {
		t.Errorf("path %q carries the key", stub.gotPath)
	}
	if strings.Contains(stub.gotQuery, "jg_verysecret") {
		t.Errorf("query %q carries the key", stub.gotQuery)
	}
	var sent struct {
		Key string `json:"key"`
	}
	if err := json.Unmarshal([]byte(stub.gotBody), &sent); err != nil {
		t.Fatalf("body is not the expected JSON: %v (%q)", err, stub.gotBody)
	}
	if sent.Key != "jg_verysecret" {
		t.Errorf("body key = %q, want jg_verysecret", sent.Key)
	}
}

func TestBillingClientStatusMapping(t *testing.T) {
	tests := []struct {
		name   string
		status int
		want   error
	}{
		{"unknown key", http.StatusUnauthorized, ErrUnknownKey},
		{"disabled key", http.StatusForbidden, ErrDisabledKey},
	}
	for _, tt := range tests {
		stub := newBillingStub(t)
		stub.status = tt.status
		stub.body = `{"error":"nope"}`

		c := NewBillingClient(stub.socket, "relay-token")
		_, err := c.Authenticate(context.Background(), "jg_x")
		if !errors.Is(err, tt.want) {
			t.Errorf("%s: err = %v, want %v", tt.name, err, tt.want)
		}
	}
}

// A 500 from the store is an outage, not a verdict about the key. It must not
// be reported as an unknown key, or a customer learns to rotate a good
// credential every time billing hiccups.
func TestBillingClientServerErrorIsNotAVerdict(t *testing.T) {
	stub := newBillingStub(t)
	stub.status = http.StatusInternalServerError
	stub.body = `{"error":"boom"}`

	c := NewBillingClient(stub.socket, "relay-token")
	_, err := c.Authenticate(context.Background(), "jg_x")
	if err == nil {
		t.Fatal("err = nil, want a failure")
	}
	if errors.Is(err, ErrUnknownKey) || errors.Is(err, ErrDisabledKey) {
		t.Fatalf("err = %v, must not be a verdict about the key", err)
	}
}

// A missing socket is an outage. The relay then fails closed, which the cache
// test already pins — here we only prove the client reports it as one.
func TestBillingClientMissingSocketIsAnOutage(t *testing.T) {
	c := NewBillingClient(filepath.Join(t.TempDir(), "absent.sock"), "relay-token")
	_, err := c.Authenticate(context.Background(), "jg_x")
	if err == nil {
		t.Fatal("err = nil, want a dial failure")
	}
	if errors.Is(err, ErrUnknownKey) || errors.Is(err, ErrDisabledKey) {
		t.Fatalf("err = %v, must not be a verdict about the key", err)
	}
}

// A slow store must not pin a customer's request open forever. The caller's
// context governs.
func TestBillingClientRespectsContext(t *testing.T) {
	stub := newBillingStub(t)
	stub.body = `{"id":"k1","enabled":true}`

	c := NewBillingClient(stub.socket, "relay-token")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := c.Authenticate(ctx, "jg_x")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
}

// The client must not hold a request open indefinitely even when the caller
// passes a background context. A hung store would otherwise leak a goroutine
// and a socket per customer request.
func TestBillingClientHasADefaultTimeout(t *testing.T) {
	c := NewBillingClient("/nonexistent.sock", "t")
	if c.httpClient().Timeout <= 0 {
		t.Error("client has no default timeout")
	}
	if c.httpClient().Timeout > 10*time.Second {
		t.Errorf("default timeout %v is too long for a request-path call", c.httpClient().Timeout)
	}
}
