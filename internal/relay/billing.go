package relay

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sync"
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

	// prices remembers what each method costs. Guarded by priceMu.
	priceMu sync.RWMutex
	prices  map[string]int64
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

// reservePath and settlePath are the ledger's two routes. They ride the same
// least-privilege relay credential as authenticate: the relay may move its own
// customer's credits, and it still cannot mint a key or rewrite a price.
const (
	reservePath = "/internal/reserve"
	settlePath  = "/internal/settle"
)

// Reserve leases credits from an account.
//
// A ZERO grant is a normal answer meaning "out of credits", not an error. The
// lease turns that into a 402; turning it into a failure here would report a
// broke customer as a broken ledger.
func (c *BillingClient) Reserve(ctx context.Context, account string, credits int64) (int64, error) {
	body, err := json.Marshal(map[string]any{"account": account, "credits": credits})
	if err != nil {
		return 0, fmt.Errorf("relay: encode reserve: %w", err)
	}

	raw, status, err := c.postJSON(ctx, reservePath, body)
	if err != nil {
		return 0, err
	}
	switch status {
	case http.StatusOK:
		var out struct {
			Granted int64 `json:"granted"`
		}
		if err := json.Unmarshal(raw, &out); err != nil {
			return 0, fmt.Errorf("relay: decode reserve: %w", err)
		}
		return out.Granted, nil
	case http.StatusNotFound:
		// A distinct fact from an outage. An operator must chase an unbound key
		// rather than a broken socket.
		return 0, ErrNoAccount
	default:
		return 0, fmt.Errorf("relay: reserve: ledger returned %d: %s", status, raw)
	}
}

// Settle reports how much of a reservation was consumed and returns the rest.
//
// A rejection is surfaced rather than swallowed: a silently dropped settle
// strands a customer's own credits inside a reservation.
func (c *BillingClient) Settle(ctx context.Context, account string, spent, reserved int64) error {
	body, err := json.Marshal(map[string]any{"account": account, "spent": spent, "reserved": reserved})
	if err != nil {
		return fmt.Errorf("relay: encode settle: %w", err)
	}

	raw, status, err := c.postJSON(ctx, settlePath, body)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("relay: settle: ledger returned %d: %s", status, raw)
	}
	return nil
}

// postJSON runs one authenticated POST and returns the body and status.
func (c *BillingClient) postJSON(ctx context.Context, path string, body []byte) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+path, bytes.NewReader(body))
	if err != nil {
		return nil, 0, fmt.Errorf("relay: build %s: %w", path, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("relay: %s: %w", path, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
	if err != nil {
		return nil, 0, fmt.Errorf("relay: read %s: %w", path, err)
	}
	return raw, resp.StatusCode, nil
}

// pricePath resolves what one method costs. It is a separate route from
// authenticate because the two answers change on different clocks: a key record
// is worth caching for a few seconds, while a price applies per method and chain
// and one cached key calls many methods.
const pricePath = "/internal/price"

// fallbackPrice is charged when the ledger cannot be asked.
//
// It is deliberately NOT zero. Falling back to free would give the product away
// the moment the store hiccupped, and nothing on the request path would report
// it. Overcharging slightly during an outage is recoverable; undercharging to
// zero is not.
const fallbackPrice = 1

// PriceOf reports what one call costs in credits, remembering each answer.
//
// Prices move on an operator's clock rather than a customer's, so asking once
// per method and chain is enough. Asking per request would put a round trip back
// on the hot path that the credit lease exists to remove.
func (c *BillingClient) PriceOf(ctx context.Context, method string, chainID int) int64 {
	key := fmt.Sprintf("%s:%d", method, chainID)

	c.priceMu.RLock()
	cached, ok := c.prices[key]
	c.priceMu.RUnlock()
	if ok {
		return cached
	}

	price := c.fetchPrice(ctx, method, chainID)

	c.priceMu.Lock()
	if c.prices == nil {
		c.prices = make(map[string]int64)
	}
	c.prices[key] = price
	c.priceMu.Unlock()
	return price
}

func (c *BillingClient) fetchPrice(ctx context.Context, method string, chainID int) int64 {
	target := fmt.Sprintf("%s%s?method=%s&chain=%d", c.base, pricePath, url.QueryEscape(method), chainID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return fallbackPrice
	}
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.hc.Do(req)
	if err != nil {
		return fallbackPrice
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fallbackPrice
	}

	var out struct {
		Credits int64 `json:"credits"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil || out.Credits <= 0 {
		return fallbackPrice
	}
	return out.Credits
}
