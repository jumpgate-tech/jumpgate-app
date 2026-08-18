package relay

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"
)

// billingTimeout bounds one authenticate call. It sits on the customer request
// path, so a hung store must fail fast rather than pin a goroutine and a socket
// open per request.
const billingTimeout = 3 * time.Second

// authenticatePath is the least-privilege route. The relay's credential opens
// this and nothing else — it cannot mint a key, rotate one, or rewrite a price.
const authenticatePath = "/internal/authenticate"

// BillingClient authenticates a key against the Rust billing service.
//
// The transport is a unix socket rather than a TCP loopback port, and that is a
// security choice rather than a performance one. Binding an unused loopback
// port needs no privilege on Linux or macOS, so a local process that starts
// before billing can squat the port and collect both the relay's credential and
// every raw customer key the relay forwards. A socket file carries filesystem
// permissions, so another user cannot connect at all.
type BillingClient struct {
	hc    *http.Client
	token string
	// base is a dummy authority. A unix-socket transport ignores the host, but
	// net/http still requires a well-formed URL.
	base string
}

// NewBillingClient dials the billing service over a unix socket.
func NewBillingClient(socketPath, token string) *BillingClient {
	dialer := &net.Dialer{Timeout: billingTimeout}
	return &BillingClient{
		hc: &http.Client{
			Timeout: billingTimeout,
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
					return dialer.DialContext(ctx, "unix", socketPath)
				},
				// One store, one socket, short-lived calls. A small warm pool
				// beats reconnecting on every customer request.
				MaxIdleConns:    8,
				IdleConnTimeout: 30 * time.Second,
			},
		},
		token: token,
		base:  "http://billing",
	}
}

// NewBillingClientTCP dials the billing service over TCP. It exists for Windows,
// where AF_UNIX support is patchy in both toolchains. Prefer the unix socket
// everywhere else: a TCP port can be squatted and a socket file cannot.
func NewBillingClientTCP(addr, token string) *BillingClient {
	return &BillingClient{
		hc:    &http.Client{Timeout: billingTimeout},
		token: token,
		base:  "http://" + addr,
	}
}

func (c *BillingClient) httpClient() *http.Client { return c.hc }

// authRequest is the wire body. The raw key travels here rather than in the URL,
// so it never reaches the store's access log — the same leak the relay prevents
// one hop earlier.
type authRequest struct {
	Key string `json:"key"`
}

// Authenticate resolves a raw key to its record.
func (c *BillingClient) Authenticate(ctx context.Context, rawKey string) (KeyRecord, error) {
	body, err := json.Marshal(authRequest{Key: rawKey})
	if err != nil {
		return KeyRecord{}, fmt.Errorf("relay: encode authenticate request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+authenticatePath, bytes.NewReader(body))
	if err != nil {
		return KeyRecord{}, fmt.Errorf("relay: build authenticate request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.hc.Do(req)
	if err != nil {
		// A dial failure, a timeout, or a cancelled context. None of these say
		// anything about the key, so the error stays unclassified and the cache
		// turns it into ErrUnavailable.
		return KeyRecord{}, fmt.Errorf("relay: authenticate: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		var rec KeyRecord
		if err := json.NewDecoder(resp.Body).Decode(&rec); err != nil {
			return KeyRecord{}, fmt.Errorf("relay: decode key record: %w", err)
		}
		return rec, nil
	case http.StatusUnauthorized:
		return KeyRecord{}, ErrUnknownKey
	case http.StatusForbidden:
		return KeyRecord{}, ErrDisabledKey
	default:
		// Anything else is the store misbehaving, not a verdict about the key.
		return KeyRecord{}, fmt.Errorf("relay: authenticate: store returned %s", resp.Status)
	}
}
