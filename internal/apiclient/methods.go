package apiclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"

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

// WatchLogs follows target's logs. The first value after every (re)connect is
// a Reset carrying the last backlog lines; later values are single new lines.
func (c *Client) WatchLogs(ctx context.Context, target string, backlog int) <-chan Update[[]api.LogHit] {
	path := targetPath(target, "/logs/stream") + "?backlog=" + strconv.Itoa(backlog)
	return watch(ctx, c, path, func(ev Event) ([]api.LogHit, bool, error) {
		if ev.Name == "reset" {
			var hs []api.LogHit
			err := json.Unmarshal(ev.Data, &hs) // before the return: operands are evaluated left to right
			return hs, true, err
		}
		var h api.LogHit
		if err := json.Unmarshal(ev.Data, &h); err != nil {
			return nil, false, err
		}
		return []api.LogHit{h}, false, nil
	})
}
