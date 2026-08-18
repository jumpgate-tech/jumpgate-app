package relay

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// callerTimeout bounds one upstream call made on a session's behalf. A poll that
// hangs must not hold a loop past its next tick.
const callerTimeout = 15 * time.Second

// ERPCCaller performs a JSON-RPC call against the keyless eRPC over plain HTTP.
//
// A terminated WebSocket session uses it for every frame, and a head poller uses
// it for eth_blockNumber and eth_getBlockByNumber. Because everything past the
// relay is ordinary HTTP, an upstream needs no WebSocket support at all.
type ERPCCaller struct {
	hc        *http.Client
	base      *url.URL
	projectID string
}

// NewERPCCaller builds a caller against an eRPC base URL.
func NewERPCCaller(base *url.URL, projectID string) *ERPCCaller {
	return &ERPCCaller{
		hc:        &http.Client{Timeout: callerTimeout},
		base:      base,
		projectID: projectID,
	}
}

// Call posts one JSON-RPC body to a chain's path.
//
// The path is built the same way the proxy path builds it, so a subscription
// polls exactly the chain the gateway serves rather than a near miss.
func (c *ERPCCaller) Call(ctx context.Context, chainID int, body []byte) ([]byte, error) {
	target := *c.base
	target.Path = fmt.Sprintf("/%s/evm/%d", c.projectID, chainID)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, target.String(), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("relay: build upstream call: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("relay: upstream call: %w", err)
	}
	defer resp.Body.Close()

	reply, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
	if err != nil {
		return nil, fmt.Errorf("relay: read upstream reply: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("relay: upstream returned %s", resp.Status)
	}
	return reply, nil
}
