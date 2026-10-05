package apiclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/valve-tech/jumpgate/internal/api"
)

func targetPath(id, rest string) string { return "/api/targets/" + url.PathEscape(id) + rest }

// Intent sends one intent through the server, which signs it.
func (c *Client) Intent(ctx context.Context, target, kind string, payload any) (api.IntentReply, error) {
	if payload == nil {
		payload = struct{}{}
	}
	var r api.IntentReply
	err := c.Do(ctx, http.MethodPost, targetPath(target, "/intent/"+url.PathEscape(kind)), payload, &r)
	return r, err
}

// Pair starts pairing target and returns its events; the channel closes when
// the server ends the stream (after a done or error event) or ctx ends.
func (c *Client) Pair(ctx context.Context, target string, req api.PairRequest) (<-chan api.PairEvent, error) {
	res, err := c.Open(ctx, http.MethodPost, targetPath(target, "/pair"), req, nil)
	if err != nil {
		return nil, err
	}
	ch := make(chan api.PairEvent, 16)
	go func() {
		defer close(ch)
		defer res.Body.Close()
		_ = parseSSE(res.Body, func(ev Event) {
			var pe api.PairEvent
			if json.Unmarshal(ev.Data, &pe) != nil {
				return
			}
			select {
			case ch <- pe:
			case <-ctx.Done():
			}
		})
	}()
	return ch, nil
}
