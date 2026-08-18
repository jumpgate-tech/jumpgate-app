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
	"time"
)

// adminTimeout bounds one key-management call. These run behind an operator's
// click rather than on the request path, so the budget is looser than the
// relay's — but not unbounded, or a hung store freezes the UI.
const adminTimeout = 10 * time.Second

// KeyView is one key as the operator sees it. It carries NO secret and no hash,
// and it has nowhere to put one: the raw key exists only in the reply to the
// call that created it.
type KeyView struct {
	ID           string `json:"id"`
	Label        string `json:"label"`
	Disabled     bool   `json:"disabled"`
	AllowTrace   bool   `json:"allow_trace"`
	CreditExempt bool   `json:"credit_exempt"`
	CreatedAt    int64  `json:"created_at"`
}

// AdminClient manages keys with the operator's admin credential.
//
// It shares a transport with BillingClient and shares nothing else. The admin
// token mints, rotates and revokes keys and rewrites prices; the relay token
// only asks whether a key is valid. Keeping them in separate types makes it
// hard to reach for the wrong one, which is the same reason the store puts them
// on separate route groups.
type AdminClient struct {
	hc    *http.Client
	token string
	base  string
}

// NewAdminClient dials the billing service over a unix socket.
func NewAdminClient(socketPath, token string) *AdminClient {
	dialer := &net.Dialer{Timeout: adminTimeout}
	return &AdminClient{
		hc: &http.Client{
			Timeout: adminTimeout,
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
					return dialer.DialContext(ctx, "unix", socketPath)
				},
			},
		},
		token: token,
		base:  "http://billing",
	}
}

// NewAdminClientTCP dials over TCP, for Windows. See NewBillingClientTCP.
func NewAdminClientTCP(addr, token string) *AdminClient {
	return &AdminClient{
		hc:    &http.Client{Timeout: adminTimeout},
		token: token,
		base:  "http://" + addr,
	}
}

func (c *AdminClient) do(ctx context.Context, method, path string, body any) ([]byte, error) {
	var payload io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("relay: encode %s %s: %w", method, path, err)
		}
		payload = bytes.NewReader(encoded)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.base+path, payload)
	if err != nil {
		return nil, fmt.Errorf("relay: build %s %s: %w", method, path, err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("relay: %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	reply, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
	if err != nil {
		return nil, fmt.Errorf("relay: read %s %s: %w", method, path, err)
	}
	// A refusal is an error rather than a silent success. An operator who
	// clicked revoke must never be told it worked when it did not.
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("relay: %s %s: store returned %s", method, path, resp.Status)
	}
	return reply, nil
}

// CreateKey mints a key and returns its id and its raw secret.
//
// The raw secret is returned HERE AND NOWHERE ELSE. The store keeps only a
// hash, so a caller that drops this value has no way to recover it and the
// operator must issue a new key.
func (c *AdminClient) CreateKey(ctx context.Context, label string) (string, string, error) {
	reply, err := c.do(ctx, http.MethodPost, "/admin/keys", map[string]string{"label": label})
	if err != nil {
		return "", "", err
	}
	var created struct {
		ID  string `json:"id"`
		Key string `json:"key"`
	}
	if err := json.Unmarshal(reply, &created); err != nil {
		return "", "", fmt.Errorf("relay: decode created key: %w", err)
	}
	return created.ID, created.Key, nil
}

// ListKeys returns every key, revoked ones included.
func (c *AdminClient) ListKeys(ctx context.Context) ([]KeyView, error) {
	reply, err := c.do(ctx, http.MethodGet, "/admin/keys", nil)
	if err != nil {
		return nil, err
	}
	// The store reports revocation as a timestamp. The operator only needs the
	// yes or no, so it collapses here rather than in the UI.
	var raw []struct {
		ID           string `json:"id"`
		Label        string `json:"label"`
		AllowTrace   bool   `json:"allow_trace"`
		CreditExempt bool   `json:"credit_exempt"`
		CreatedAt    int64  `json:"created_at"`
		DisabledAt   *int64 `json:"disabled_at"`
	}
	if err := json.Unmarshal(reply, &raw); err != nil {
		return nil, fmt.Errorf("relay: decode key list: %w", err)
	}

	keys := make([]KeyView, 0, len(raw))
	for _, r := range raw {
		keys = append(keys, KeyView{
			ID:           r.ID,
			Label:        r.Label,
			Disabled:     r.DisabledAt != nil,
			AllowTrace:   r.AllowTrace,
			CreditExempt: r.CreditExempt,
			CreatedAt:    r.CreatedAt,
		})
	}
	return keys, nil
}

// RevokeKey clears a key's enabled flag. The relay stops honouring it within
// the cache TTL rather than instantly, which is the cost of the store living in
// another process.
func (c *AdminClient) RevokeKey(ctx context.Context, id string) error {
	_, err := c.do(ctx, http.MethodDelete, "/admin/keys/"+url.PathEscape(id), nil)
	return err
}
