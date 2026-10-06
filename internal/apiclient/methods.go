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

// Targets lists every target with its link.
func (c *Client) Targets(ctx context.Context) ([]api.TargetView, error) {
	var ts []api.TargetView
	err := c.Do(ctx, http.MethodGet, "/api/targets", nil, &ts)
	return ts, err
}

// ServiceAction starts, stops or restarts a node service.
func (c *Client) ServiceAction(ctx context.Context, target, service, action string) (api.ServiceResult, error) {
	var r api.ServiceResult
	err := c.Do(ctx, http.MethodPost, targetPath(target, "/services/"+url.PathEscape(service)+"/"+url.PathEscape(action)), nil, &r)
	return r, err
}

// Endpoints reads the node's RPC endpoints.
func (c *Client) Endpoints(ctx context.Context, target string) (api.Endpoints, error) {
	var e api.Endpoints
	err := c.Do(ctx, http.MethodGet, targetPath(target, "/endpoints"), nil, &e)
	return e, err
}

// Firewall reads the firewall checklist.
func (c *Client) Firewall(ctx context.Context, target string) ([]api.CheckItem, error) {
	var items []api.CheckItem
	err := c.Do(ctx, http.MethodGet, targetPath(target, "/firewall"), nil, &items)
	return items, err
}

// RemoveTarget forgets a target on this controller (spec D19).
func (c *Client) RemoveTarget(ctx context.Context, target string) error {
	return c.Do(ctx, http.MethodDelete, targetPath(target, ""), nil, nil)
}

// ProbeHostKeys asks the server what keys the hosts on ssh's path present,
// jump host first. It trusts nothing: an unconfirmed hop comes back with a
// probe id for ConfirmHostKey.
func (c *Client) ProbeHostKeys(ctx context.Context, ssh api.SSHView) (api.HostKeyProbe, error) {
	var p api.HostKeyProbe
	err := c.Do(ctx, http.MethodPost, "/api/hostkeys/probe", api.HostKeyProbeRequest{SSH: ssh}, &p)
	return p, err
}

// ConfirmHostKey records a probed key after a person compared its
// fingerprint. The server records the key it captured for probeID, and only
// if fingerprint is exactly that key's.
func (c *Client) ConfirmHostKey(ctx context.Context, probeID, fingerprint string) error {
	return c.Do(ctx, http.MethodPost, "/api/hostkeys/confirm", api.HostKeyConfirm{ProbeID: probeID, Fingerprint: fingerprint}, nil)
}

// AddTarget records a new target; it does not pair it.
func (c *Client) AddTarget(ctx context.Context, req api.AddTarget) error {
	return c.Do(ctx, http.MethodPost, "/api/targets", req, nil)
}

// RecordedHostKeys lists every key the server's Strict policy trusts for
// hostPort, by store.
func (c *Client) RecordedHostKeys(ctx context.Context, hostPort string) (api.RecordedHostKeys, error) {
	var r api.RecordedHostKeys
	err := c.Do(ctx, http.MethodGet, "/api/hostkeys?hostPort="+url.QueryEscape(hostPort), nil, &r)
	return r, err
}

// ForgetHostKey removes the one confirmed key for hostPort whose fingerprint
// is fingerprint. Keys in OpenSSH's known_hosts are never removed.
func (c *Client) ForgetHostKey(ctx context.Context, hostPort, fingerprint string) error {
	return c.Do(ctx, http.MethodPost, "/api/hostkeys/forget", api.HostKeyForget{HostPort: hostPort, Fingerprint: fingerprint}, nil)
}

// Fleet is every box's row as the server last polled it.
func (c *Client) Fleet(ctx context.Context) (api.Fleet, error) {
	var f api.Fleet
	err := c.Do(ctx, http.MethodGet, "/api/fleet", nil, &f)
	return f, err
}

// WatchFleet streams the whole fleet after every poll round.
func (c *Client) WatchFleet(ctx context.Context) <-chan Update[api.Fleet] {
	return watch(ctx, c, "/api/fleet/stream", decodeJSON[api.Fleet])
}

func fleetPath(id, rest string) string { return "/api/fleet/" + url.PathEscape(id) + rest }

// Disk is the box's last disk reading with its history (measured now if it
// has none).
func (c *Client) Disk(ctx context.Context, target string) (api.DiskView, error) {
	var d api.DiskView
	err := c.Do(ctx, http.MethodGet, fleetPath(target, "/disk"), nil, &d)
	return d, err
}

// MeasureDisk takes a disk reading now.
func (c *Client) MeasureDisk(ctx context.Context, target string) (api.DiskView, error) {
	var d api.DiskView
	err := c.Do(ctx, http.MethodPost, fleetPath(target, "/disk/measure"), nil, &d)
	return d, err
}

// WatchStatus streams one box's NodeStatus every few seconds.
func (c *Client) WatchStatus(ctx context.Context, target string) <-chan Update[api.NodeStatus] {
	return watch(ctx, c, fleetPath(target, "/status/stream"), decodeJSON[api.NodeStatus])
}

// Explain asks the configured AI provider about lines (or, with none, the
// box's recent error lines).
func (c *Client) Explain(ctx context.Context, target string, lines []string) (api.Explain, error) {
	var e api.Explain
	err := c.Do(ctx, http.MethodPost, targetPath(target, "/explain"), map[string]any{"lines": lines}, &e)
	return e, err
}

// Settings reads the provider settings and the AI disclosure line.
func (c *Client) Settings(ctx context.Context) (api.Settings, error) {
	var s api.Settings
	err := c.Do(ctx, http.MethodGet, "/api/settings", nil, &s)
	return s, err
}

// SaveSettings changes the provider settings and returns the new state.
func (c *Client) SaveSettings(ctx context.Context, u api.SettingsUpdate) (api.Settings, error) {
	var s api.Settings
	err := c.Do(ctx, http.MethodPut, "/api/settings", u, &s)
	return s, err
}
