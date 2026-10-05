// Package apiclient is the one Go client of the local jumpgate server, used by
// the CLI and the TUI. It finds (or starts) the server through internal/daemon,
// talks HTTP over the server's owner-only unix socket with the session token,
// decodes every error into *api.Error, and reads SSE streams.
package apiclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/apiclient/internal/testhook"
	"github.com/valve-tech/jumpgate/internal/buildinfo"
	"github.com/valve-tech/jumpgate/internal/daemon"
)

// requestTimeout bounds a request/response call. An intent may take the agent
// client's two minutes, so this is longer; streams have no overall timeout.
const requestTimeout = 3 * time.Minute

// maxBody bounds a JSON answer.
const maxBody = 32 << 20

var (
	// ErrNoServer is Connect's answer when Start is false and none runs.
	ErrNoServer = errors.New("no jumpgate server is running")
	// ErrServerUnreachable wraps a transport failure talking to the local
	// server (it stopped, or its socket is gone), as opposed to an error the
	// server answered with.
	ErrServerUnreachable = errors.New("the jumpgate server did not answer")
)

// Client talks to one local server.
type Client struct {
	base  string
	token string
	info  daemon.Info  // what server.json said; its Version is the server's
	hc    *http.Client // requests
	sc    *http.Client // streams: no overall timeout
	opts  Options
}

// Options says how Connect finds the server.
type Options struct {
	Start bool   // start `Exe serve` when no server runs
	Exe   string // this jumpgate executable, for Start
}

// Connect finds the running server, starting one when o.Start is set. It
// never prints: a caller that wants the version note calls WarnSkew.
func Connect(ctx context.Context, o Options) (*Client, error) {
	var info daemon.Info
	if o.Start {
		var err error
		if info, err = daemon.EnsureRunning(ctx, o.Exe, nil); err != nil {
			return nil, err
		}
	} else {
		var ok bool
		var err error
		if info, ok, err = daemon.Find(ctx); err != nil {
			return nil, err
		} else if !ok {
			return nil, ErrNoServer
		}
	}
	c := New(info)
	c.opts = o
	return c, nil
}

// New is a client for the server described by info (server.json).
func New(info daemon.Info) *Client {
	tr := info.Client().Transport
	return &Client{
		base: daemon.BaseURL, token: info.Token, info: info,
		hc: &http.Client{Transport: tr, Timeout: requestTimeout},
		sc: &http.Client{Transport: tr},
	}
}

// newHTTP is a client for a server at a TCP base URL that reports version:
// tests, and nothing else. It is unexported so no production caller can claim
// a version for a server and silence a real skew; other packages' tests reach
// it through apiclienttest.
func newHTTP(baseURL, token, version string) *Client {
	return &Client{
		base: strings.TrimRight(baseURL, "/"), token: token,
		info: daemon.Info{Token: token, Version: version},
		hc:   &http.Client{Timeout: requestTimeout}, sc: &http.Client{},
	}
}

func init() {
	testhook.NewHTTP = func(baseURL, token, version string) any { return newHTTP(baseURL, token, version) }
}

// Info is the server.json the client was built from. The CLI hands it to
// reportServerErrorFrom, which tells a 404 from another version apart.
func (c *Client) Info() daemon.Info { return c.info }

// ServerVersion is the version the server published in server.json.
func (c *Client) ServerVersion() string { return c.info.Version }

// Skew compares the server's version with this binary's through
// daemon.SkewWarning, the one comparison: an unknown server version (an older
// server.json) counts as a skew.
func (c *Client) Skew() (server, mine string, differs bool) {
	return c.info.Version, buildinfo.Version(), daemon.SkewWarning(c.info) != ""
}

// WarnSkew prints daemon.SkewWarning's line when the server is another
// version: after an upgrade the old detached server keeps answering, and its
// 404s would otherwise read as plain failures.
func WarnSkew(w io.Writer, c *Client) {
	if s := daemon.SkewWarning(c.info); s != "" {
		fmt.Fprintln(w, s)
	}
}

func (c *Client) send(ctx context.Context, hc *http.Client, method, path string, in any, header http.Header) (*http.Response, error) {
	var body io.Reader
	if in != nil {
		b, err := json.Marshal(in)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, body)
	if err != nil {
		return nil, err
	}
	for k, vs := range header {
		req.Header[k] = vs
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := hc.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, fmt.Errorf("%w: %v", ErrServerUnreachable, err)
	}
	if res.StatusCode >= 300 {
		defer res.Body.Close()
		b, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
		return nil, api.Decode(res.StatusCode, b)
	}
	return res, nil
}

// Do sends one request and decodes a 2xx JSON answer into out (unless out is
// nil). Any other status comes back as *api.Error.
func (c *Client) Do(ctx context.Context, method, path string, in, out any) error {
	res, err := c.send(ctx, c.hc, method, path, in, nil)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if out == nil {
		_, _ = io.Copy(io.Discard, io.LimitReader(res.Body, maxBody))
		return nil
	}
	if err := json.NewDecoder(io.LimitReader(res.Body, maxBody)).Decode(out); err != nil {
		return fmt.Errorf("could not read the server's answer to %s %s: %w", method, path, err)
	}
	return nil
}

// Open sends a request whose body the caller reads and closes: a stream.
func (c *Client) Open(ctx context.Context, method, path string, in any, header http.Header) (*http.Response, error) {
	return c.send(ctx, c.sc, method, path, in, header)
}
