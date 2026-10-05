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
	prices  map[string]cachedPrice

	// now is the clock. Nil means time.Now.
	now func() time.Time
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

// authenticateReply is the record exactly as billing sends it: the Rust
// AuthenticateView in services/billing/src/admin.rs. It is a wire form only and
// is turned into a KeyRecord at once, so the rest of the relay never sees it.
//
// Its shape is billing's, not ours, because billing is the store and its admin
// API has other readers. The relay once decoded straight into KeyRecord's flat
// fields, which billing never sends, and every constraint and rate limit came
// out empty without a single error. testdata/authenticate_contract.json pins
// the shape on both sides now; change this struct and that fixture together.
type authenticateReply struct {
	ID             string  `json:"id"`
	Label          string  `json:"label"`
	Enabled        bool    `json:"enabled"`
	AccountAddress *string `json:"account_address"`
	CreditExempt   bool    `json:"credit_exempt"`
	AllowTrace     bool    `json:"allow_trace"`
	// Rate is either the string "unlimited" or
	// {"limited": {"per_second": n, "per_day": n}}, serde's form for an enum.
	Rate json.RawMessage `json:"rate"`
	// Constraints is a pointer so that a reply without it is an error rather
	// than a key with no policy. An absent field is the exact failure this
	// type exists to stop: it reads as "allow everything".
	Constraints *authenticateConstraints `json:"constraints"`
}

// authenticateConstraints is billing's ConstraintsView: the key's (kind, value)
// rows grouped into one list per kind.
type authenticateConstraints struct {
	Origins     []string `json:"origins"`
	MethodAllow []string `json:"method_allow"`
	MethodBlock []string `json:"method_block"`
	Networks    []string `json:"networks"`
	IPAllow     []string `json:"ip_allow"`
	IPDeny      []string `json:"ip_deny"`
}

// limitedRate is the body of a {"limited": {...}} rate. The fields are pointers
// so a limit billing forgot to send is an error, not a silent zero, which the
// limiter would read as "this axis is not limited".
type limitedRate struct {
	PerSecond *int `json:"per_second"`
	PerDay    *int `json:"per_day"`
}

// record converts the wire form into the relay's KeyRecord. Anything it does not
// understand is an error. The cache turns that into an outage answer, which
// fails closed: a record the relay cannot read must never be served as a key
// with no limits.
func (a authenticateReply) record() (KeyRecord, error) {
	if a.Constraints == nil {
		return KeyRecord{}, fmt.Errorf("key %q: reply has no constraints", a.ID)
	}
	rec := KeyRecord{
		ID:           a.ID,
		Label:        a.Label,
		Enabled:      a.Enabled,
		CreditExempt: a.CreditExempt,
		AllowTrace:   a.AllowTrace,
		MethodAllow:  a.Constraints.MethodAllow,
		MethodBlock:  a.Constraints.MethodBlock,
		Origins:      a.Constraints.Origins,
		Networks:     a.Constraints.Networks,
		IPAllow:      a.Constraints.IPAllow,
		IPDeny:       a.Constraints.IPDeny,
	}
	if a.AccountAddress != nil {
		rec.AccountAddress = *a.AccountAddress
	}
	if err := decodeRate(a.Rate, &rec); err != nil {
		return KeyRecord{}, fmt.Errorf("key %q: %w", a.ID, err)
	}
	return rec, nil
}

// decodeRate reads billing's rate enum into rec's three rate fields.
func decodeRate(raw json.RawMessage, rec *KeyRecord) error {
	if len(raw) == 0 || string(raw) == "null" {
		return fmt.Errorf("reply has no rate")
	}
	var unit string
	if err := json.Unmarshal(raw, &unit); err == nil {
		if unit != "unlimited" {
			return fmt.Errorf("unknown rate %q", unit)
		}
		rec.RateUnlimited = true
		return nil
	}
	var tagged map[string]json.RawMessage
	if err := json.Unmarshal(raw, &tagged); err != nil {
		return fmt.Errorf("rate is neither a string nor an object: %s", raw)
	}
	body, ok := tagged["limited"]
	if !ok || len(tagged) != 1 {
		return fmt.Errorf("unknown rate %s", raw)
	}
	var lim limitedRate
	if err := json.Unmarshal(body, &lim); err != nil {
		return fmt.Errorf("limited rate: %w", err)
	}
	if lim.PerSecond == nil || lim.PerDay == nil {
		return fmt.Errorf("limited rate is missing a limit: %s", body)
	}
	rec.PerSecondLimit = *lim.PerSecond
	rec.PerDayLimit = *lim.PerDay
	return nil
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
		var wire authenticateReply
		if err := json.NewDecoder(resp.Body).Decode(&wire); err != nil {
			return KeyRecord{}, fmt.Errorf("relay: decode key record: %w", err)
		}
		rec, err := wire.record()
		if err != nil {
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
func (c *BillingClient) Settle(ctx context.Context, account string, spent, reserved int64, settleID string) error {
	body, err := json.Marshal(map[string]any{
		"account":   account,
		"spent":     spent,
		"reserved":  reserved,
		"settle_id": settleID,
	})
	if err != nil {
		return fmt.Errorf("relay: encode settle: %w", err)
	}

	raw, status, err := c.postJSON(ctx, settlePath, body)
	if err != nil {
		return err
	}
	if status == http.StatusConflict {
		// billing's 409 is SettleExceedsReservation or SettleIdReused: neither
		// can ever succeed on a retry. Every other status stays retryable.
		return fmt.Errorf("%w: ledger returned 409: %s", ErrSettleRefused, raw)
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

// priceTTL is how long a price the store gave is trusted. Prices change on an
// operator's clock, so minutes is fresh enough, and it keeps the store off the
// hot path.
const priceTTL = 5 * time.Minute

// priceRetryAfter is how long a failed lookup waits before the store is asked
// again. Without it an outage would put a dead-socket timeout on every call.
const priceRetryAfter = 5 * time.Second

// cachedPrice is one remembered answer. A price that came from the store is
// good; a fallback is not, and expires on the much shorter retry window.
type cachedPrice struct {
	credits int64
	good    bool
	expires time.Time
}

// PriceOf reports what one call costs in credits.
//
// Prices move on an operator's clock rather than a customer's, so a store answer
// is remembered for priceTTL. Asking per request would put a round trip back on
// the hot path that the credit lease exists to remove. A failed lookup is never
// remembered as the price: it charges the last good price if there is one, or
// the fallback, and asks again after priceRetryAfter.
func (c *BillingClient) PriceOf(ctx context.Context, method string, chainID int) int64 {
	key := fmt.Sprintf("%s:%d", method, chainID)
	now := c.clock()

	c.priceMu.RLock()
	cached, ok := c.prices[key]
	c.priceMu.RUnlock()
	if ok && now.Before(cached.expires) {
		return cached.credits
	}

	entry := cachedPrice{good: true, expires: now.Add(priceTTL)}
	price, err := c.fetchPrice(ctx, method, chainID)
	switch {
	case err == nil:
		entry.credits = price
	case ok && cached.good:
		// A price the operator set beats the fallback. Keep charging it and ask
		// again after the retry window.
		entry.credits = cached.credits
		entry.expires = now.Add(priceRetryAfter)
	default:
		entry = cachedPrice{credits: fallbackPrice, expires: now.Add(priceRetryAfter)}
	}

	c.priceMu.Lock()
	if c.prices == nil {
		c.prices = make(map[string]cachedPrice)
	}
	c.prices[key] = entry
	c.priceMu.Unlock()
	return entry.credits
}

func (c *BillingClient) clock() time.Time {
	if c.now != nil {
		return c.now()
	}
	return time.Now()
}

func (c *BillingClient) fetchPrice(ctx context.Context, method string, chainID int) (int64, error) {
	target := fmt.Sprintf("%s%s?method=%s&chain=%d", c.base, pricePath, url.QueryEscape(method), chainID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.hc.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("relay: price lookup: status %d", resp.StatusCode)
	}

	var out struct {
		Credits int64 `json:"credits"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return 0, fmt.Errorf("relay: price lookup: %w", err)
	}
	if out.Credits <= 0 {
		return 0, fmt.Errorf("relay: price lookup: non-positive price %d", out.Credits)
	}
	return out.Credits, nil
}
