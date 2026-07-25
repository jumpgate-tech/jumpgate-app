package capabilities

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// DefaultBaseURL is the valve.city capability API — the same endpoint behind
// the table on learn.valve.city/rpc.
//
// Consuming the published API rather than reimplementing its probes is the
// point: one prober, one method set, one set of classification rules, so what
// this app tells an operator and what the docs tell a reader cannot drift.
const DefaultBaseURL = "https://valve.city/v1/capabilities"

// maxAPIBytes caps a valve.city response. The real one for a busy chain is a
// few tens of kilobytes.
const maxAPIBytes = 8 << 20

// Client reads the valve.city capability API.
//
// It is one of two sources and covers the case the local prober cannot: public
// endpoints as seen from a well-connected host, sampled across everything
// chainlist.org lists, cached server-side so a wizard does not re-probe thirty
// strangers' nodes on every run. What it cannot cover is anything private —
// see the package doc.
type Client struct {
	// HTTPClient is injectable so tests serve the API in-process.
	HTTPClient *http.Client
	// BaseURL defaults to DefaultBaseURL. /check is appended for Check.
	BaseURL string
}

// NewClient returns a Client pointed at valve.city.
func NewClient() *Client {
	return &Client{HTTPClient: &http.Client{}, BaseURL: DefaultBaseURL}
}

func (c *Client) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return http.DefaultClient
}

func (c *Client) baseURL() string {
	if c.BaseURL != "" {
		return strings.TrimSuffix(c.BaseURL, "/")
	}
	return DefaultBaseURL
}

// Matrix fetches the published capability table for a chain.
//
// valve.city serves this for a fixed set of chains (1, 369, 943 at the time of
// writing) and 400s otherwise; a caller provisioning any other chain gets an
// error here and should lean on the local prober alone. Gather already does.
func (c *Client) Matrix(ctx context.Context, chainID int) (Matrix, error) {
	url := fmt.Sprintf("%s?chainId=%d", c.baseURL(), chainID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Matrix{}, fmt.Errorf("capabilities: build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	body, err := c.send(req)
	if err != nil {
		return Matrix{}, err
	}

	var wire wireMatrix
	if err := json.Unmarshal(body, &wire); err != nil {
		return Matrix{}, fmt.Errorf("capabilities: decode %s: %w", url, err)
	}
	return wire.matrix(chainID), nil
}

// Check asks valve.city to probe one URL on the caller's behalf.
//
// This is the ad-hoc path the published table's "add your RPC" box uses. It is
// worth having even though the local prober can check any URL directly, because
// the two answers mean different things: valve.city's says whether the wider
// internet can use the endpoint, ours says whether this box can. An operator
// who has just opened a firewall wants both.
//
// A URL valve.city refuses to touch (its SSRF guard rejects private and
// loopback addresses) comes back as an error, which is the expected outcome for
// exactly the endpoints the local prober exists to cover.
func (c *Client) Check(ctx context.Context, rawURL string, chainID int) (Endpoint, error) {
	payload, err := json.Marshal(struct {
		URL     string `json:"url"`
		ChainID int    `json:"chainId"`
	}{rawURL, chainID})
	if err != nil {
		return Endpoint{}, fmt.Errorf("capabilities: build check request: %w", err)
	}

	url := c.baseURL() + "/check"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return Endpoint{}, fmt.Errorf("capabilities: build check request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	body, err := c.send(req)
	if err != nil {
		return Endpoint{}, err
	}

	var wire struct {
		Endpoint wireEndpoint `json:"endpoint"`
	}
	if err := json.Unmarshal(body, &wire); err != nil {
		return Endpoint{}, fmt.Errorf("capabilities: decode %s: %w", url, err)
	}
	return wire.Endpoint.endpoint(), nil
}

// send performs a request and reads the body, turning a non-200 into an error
// that carries the API's own explanation when it supplied one. valve.city
// answers a rejected URL with {"error": "endpoint rejected: …"}, and repeating
// that sentence is far more use to an operator than "HTTP 400".
func (c *Client) send(req *http.Request) ([]byte, error) {
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("capabilities: fetch %s: %w", req.URL, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxAPIBytes))
	if err != nil {
		return nil, fmt.Errorf("capabilities: read %s: %w", req.URL, err)
	}
	if resp.StatusCode != http.StatusOK {
		var apiErr struct {
			Error string `json:"error"`
		}
		if json.Unmarshal(body, &apiErr) == nil && apiErr.Error != "" {
			return nil, fmt.Errorf("capabilities: %s: %s (HTTP %s)", req.URL, apiErr.Error, resp.Status)
		}
		return nil, fmt.Errorf("capabilities: %s: unexpected status %s", req.URL, resp.Status)
	}
	return body, nil
}

// ---------------------------------------------------------------------------
// Wire types — the shape valve.city actually serves. Decoded into their own
// structs rather than onto the domain types so a field added upstream cannot
// break us, and so the mapping from their vocabulary to ours (status strings,
// chainOk's null, provenance) is written down in one place.
// ---------------------------------------------------------------------------

type wireCapResult struct {
	Status string `json:"status"`
	Method string `json:"method"`
	Detail string `json:"detail"`
}

type wireEndpoint struct {
	URL          string                   `json:"url"`
	Label        string                   `json:"label"`
	Source       string                   `json:"source"`
	Reachable    bool                     `json:"reachable"`
	ChainOK      *bool                    `json:"chainOk"`
	ReachDetail  string                   `json:"reachDetail"`
	Capabilities map[string]wireCapResult `json:"capabilities"`
}

type wireMatrix struct {
	ChainID         int            `json:"chainId"`
	CapabilityKeys  []string       `json:"capabilityKeys"`
	ChainlistTotal  int            `json:"chainlistTotal"`
	ChainlistProbed int            `json:"chainlistProbed"`
	Endpoints       []wireEndpoint `json:"endpoints"`
}

func (w wireMatrix) matrix(fallbackChainID int) Matrix {
	m := Matrix{
		ChainID:         w.ChainID,
		CapabilityKeys:  append([]string(nil), w.CapabilityKeys...),
		ChainlistTotal:  w.ChainlistTotal,
		ChainlistProbed: w.ChainlistProbed,
	}
	if m.ChainID == 0 {
		m.ChainID = fallbackChainID
	}
	if len(m.CapabilityKeys) == 0 {
		m.CapabilityKeys = ValveCityKeys()
	}
	m.Endpoints = make([]Endpoint, 0, len(w.Endpoints))
	for _, we := range w.Endpoints {
		m.Endpoints = append(m.Endpoints, we.endpoint())
	}
	return m
}

func (w wireEndpoint) endpoint() Endpoint {
	ep := Endpoint{
		URL:          w.URL,
		Label:        w.Label,
		Source:       source(w.Source),
		Reachable:    w.Reachable,
		ChainOK:      w.ChainOK,
		ReachDetail:  w.ReachDetail,
		Origin:       OriginValveCity,
		Capabilities: make(map[string]Result, len(w.Capabilities)),
	}
	if ep.Label == "" {
		ep.Label = labelFor(w.URL)
	}
	for key, wc := range w.Capabilities {
		ep.Capabilities[key] = Result{
			Status: status(wc.Status),
			Method: wc.Method,
			Detail: wc.Detail,
			Origin: OriginValveCity,
		}
	}
	return ep
}

// status maps valve.city's status string onto ours. Anything unrecognised —
// a newer status, a truncated response, an empty string — becomes
// inconclusive. Guessing in either other direction would put a tick or a cross
// in front of an operator on the strength of a string we do not understand.
func status(s string) Status {
	switch Status(s) {
	case StatusSupported:
		return StatusSupported
	case StatusUnsupported:
		return StatusUnsupported
	}
	return StatusInconclusive
}

// source maps valve.city's endpoint provenance. An unknown value is reported as
// SourceUser: it came from outside our own catalogue, which is all a caller
// needs the field for.
func source(s string) Source {
	switch Source(s) {
	case SourceValve:
		return SourceValve
	case SourceChainlist:
		return SourceChainlist
	case SourceLocal:
		return SourceLocal
	}
	return SourceUser
}
