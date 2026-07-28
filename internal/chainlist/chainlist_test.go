package chainlist

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/valve-tech/valve-node-app/internal/wsrpc"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// feedServer serves a chains.json body built from chains, so no test ever
// touches chainid.network.
func feedServer(t *testing.T, chains []Chain) *httptest.Server {
	t.Helper()
	body, err := json.Marshal(chains)
	if err != nil {
		t.Fatalf("marshal feed: %v", err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// rpcServer serves eth_chainId over HTTP, answering with chainID (hex-encoded
// as a JSON-RPC quantity). before, if non-nil, runs first and can sleep, fail
// the response, or synchronise with other requests.
func rpcServer(t *testing.T, chainID int, before func(r *http.Request)) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if before != nil {
			before(r)
		}
		var req struct {
			Method string `json:"method"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.Method != "eth_chainId" {
			http.Error(w, "unexpected method "+req.Method, http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"jsonrpc":"2.0","id":1,"result":"0x%x"}`, chainID)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// wsURL rewrites an httptest server's http:// URL to the ws:// scheme.
func wsURL(srv *httptest.Server) string {
	return "ws://" + strings.TrimPrefix(srv.URL, "http://")
}

// wsRPCServer speaks just enough of RFC 6455 (server side) to accept a
// handshake, read one masked client frame, and reply with an eth_chainId
// answer — proving the stdlib-only WebSocket prober works end to end without
// pulling in a WebSocket dependency.
func wsRPCServer(t *testing.T, chainID int, delay time.Duration) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
			http.Error(w, "not a websocket upgrade", http.StatusBadRequest)
			return
		}
		sum := sha1.Sum([]byte(r.Header.Get("Sec-WebSocket-Key") + wsrpc.GUID))
		hj, ok := w.(http.Hijacker)
		if !ok {
			t.Errorf("ResponseWriter is not a Hijacker")
			return
		}
		conn, rw, err := hj.Hijack()
		if err != nil {
			t.Errorf("hijack: %v", err)
			return
		}
		defer conn.Close()
		_, _ = rw.WriteString("HTTP/1.1 101 Switching Protocols\r\n" +
			"Upgrade: websocket\r\nConnection: Upgrade\r\n" +
			"Sec-WebSocket-Accept: " + base64.StdEncoding.EncodeToString(sum[:]) + "\r\n\r\n")
		if err := rw.Flush(); err != nil {
			return
		}
		payload, err := testReadFrame(rw.Reader)
		if err != nil {
			return
		}
		var req struct {
			Method string `json:"method"`
		}
		if err := json.Unmarshal(payload, &req); err != nil || req.Method != "eth_chainId" {
			return
		}
		if delay > 0 {
			time.Sleep(delay)
		}
		resp := fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"result":"0x%x"}`, chainID)
		_ = testWriteFrame(conn, []byte(resp))
	}))
	t.Cleanup(srv.Close)
	return srv
}

// testReadFrame reads one (client-masked) text frame. Test-side mirror of the
// prober's frame reader; deliberately independent so a bug in one does not
// cancel out a bug in the other.
func testReadFrame(br *bufio.Reader) ([]byte, error) {
	var hdr [2]byte
	if _, err := io.ReadFull(br, hdr[:]); err != nil {
		return nil, err
	}
	n := int(hdr[1] & 0x7f)
	switch n {
	case 126:
		var ext [2]byte
		if _, err := io.ReadFull(br, ext[:]); err != nil {
			return nil, err
		}
		n = int(binary.BigEndian.Uint16(ext[:]))
	case 127:
		var ext [8]byte
		if _, err := io.ReadFull(br, ext[:]); err != nil {
			return nil, err
		}
		n = int(binary.BigEndian.Uint64(ext[:]))
	}
	var mask [4]byte
	masked := hdr[1]&0x80 != 0
	if !masked {
		return nil, errors.New("client frame must be masked")
	}
	if _, err := io.ReadFull(br, mask[:]); err != nil {
		return nil, err
	}
	buf := make([]byte, n)
	if _, err := io.ReadFull(br, buf); err != nil {
		return nil, err
	}
	for i := range buf {
		buf[i] ^= mask[i%4]
	}
	return buf, nil
}

// testWriteFrame writes one unmasked text frame (servers must not mask).
func testWriteFrame(w io.Writer, payload []byte) error {
	var buf bytes.Buffer
	buf.WriteByte(0x81) // FIN | text
	switch n := len(payload); {
	case n < 126:
		buf.WriteByte(byte(n))
	default:
		buf.WriteByte(126)
		_ = binary.Write(&buf, binary.BigEndian, uint16(n))
	}
	buf.Write(payload)
	_, err := w.Write(buf.Bytes())
	return err
}

// roundTripFunc adapts a func to http.RoundTripper so a test can serve both
// the feed and the probes entirely in-process.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func jsonResponse(req *http.Request, body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    req,
	}
}

// byURL indexes a result's endpoints for assertions.
func byURL(t *testing.T, res Result) map[string]Endpoint {
	t.Helper()
	m := make(map[string]Endpoint, len(res.Endpoints))
	for _, ep := range res.Endpoints {
		m[ep.URL] = ep
	}
	return m
}

func urlsOf(eps []Endpoint) []string {
	out := make([]string, 0, len(eps))
	for _, ep := range eps {
		out = append(out, ep.URL)
	}
	return out
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// ---------------------------------------------------------------------------
// Candidates: template filtering + http/ws classification
// ---------------------------------------------------------------------------

func TestCandidates(t *testing.T) {
	tests := []struct {
		name       string
		rpcs       []string
		wantKinds  map[string]Kind   // url -> expected kind (pending entries)
		wantReject map[string]string // url -> substring expected in Reason
		wantOrder  []string          // full expected URL order of the output
	}{
		{
			name: "api key templates are filtered",
			rpcs: []string{
				"https://mainnet.infura.io/v3/${INFURA_API_KEY}",
				"wss://mainnet.infura.io/ws/v3/${INFURA_API_KEY}",
				"https://cloudflare-eth.com",
			},
			wantKinds: map[string]Kind{"https://cloudflare-eth.com": KindHTTP},
			wantReject: map[string]string{
				"https://mainnet.infura.io/v3/${INFURA_API_KEY}":  "template",
				"wss://mainnet.infura.io/ws/v3/${INFURA_API_KEY}": "template",
			},
			wantOrder: []string{
				"https://mainnet.infura.io/v3/${INFURA_API_KEY}",
				"wss://mainnet.infura.io/ws/v3/${INFURA_API_KEY}",
				"https://cloudflare-eth.com",
			},
		},
		{
			name: "http and ws schemes are both kept and classified",
			rpcs: []string{
				"https://rpc.pulsechain.com",
				"http://plain.example",
				"wss://pulsechain-rpc.publicnode.com",
				"ws://plain.example",
			},
			wantKinds: map[string]Kind{
				"https://rpc.pulsechain.com":          KindHTTP,
				"http://plain.example":                KindHTTP,
				"wss://pulsechain-rpc.publicnode.com": KindWS,
				"ws://plain.example":                  KindWS,
			},
		},
		{
			name: "non rpc schemes are rejected",
			rpcs: []string{
				"https://ok.example",
				"ipc:///tmp/geth.ipc",
				"not a url at all",
			},
			wantKinds: map[string]Kind{"https://ok.example": KindHTTP},
			wantReject: map[string]string{
				"ipc:///tmp/geth.ipc": "scheme",
				"not a url at all":    "scheme",
			},
		},
		{
			name: "exact duplicates collapse",
			rpcs: []string{
				"https://rpc.pulsechain.com",
				"https://rpc.pulsechain.com",
				"wss://rpc.pulsechain.com",
			},
			wantOrder: []string{"https://rpc.pulsechain.com", "wss://rpc.pulsechain.com"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Candidates(tt.rpcs)

			if tt.wantOrder != nil {
				if !equalStrings(urlsOf(got), tt.wantOrder) {
					t.Fatalf("Candidates(%v) urls = %v, want %v", tt.rpcs, urlsOf(got), tt.wantOrder)
				}
			}

			index := make(map[string]Endpoint, len(got))
			for _, ep := range got {
				index[ep.URL] = ep
			}

			for url, kind := range tt.wantKinds {
				ep, ok := index[url]
				if !ok {
					t.Fatalf("Candidates(%v) dropped %q entirely", tt.rpcs, url)
				}
				if ep.Kind != kind {
					t.Errorf("Candidates: %q kind = %q, want %q", url, ep.Kind, kind)
				}
				if ep.Status != StatusPending {
					t.Errorf("Candidates: %q status = %q, want %q", url, ep.Status, StatusPending)
				}
			}

			for url, reason := range tt.wantReject {
				ep, ok := index[url]
				if !ok {
					t.Fatalf("Candidates(%v) dropped %q entirely; want it reported as rejected", tt.rpcs, url)
				}
				if ep.Status != StatusRejected {
					t.Errorf("Candidates: %q status = %q, want %q", url, ep.Status, StatusRejected)
				}
				if !strings.Contains(strings.ToLower(ep.Reason), reason) {
					t.Errorf("Candidates: %q reason = %q, want it to mention %q", url, ep.Reason, reason)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Discover: probing over a fake feed
// ---------------------------------------------------------------------------

func TestDiscover_ProbesHTTPAndRejectsMismatches(t *testing.T) {
	const chainID = 369

	good := rpcServer(t, chainID, nil)
	wrong := rpcServer(t, 1, nil) // answers, but for the wrong chain
	broken := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	t.Cleanup(broken.Close)

	// A templated URL pointing at a server that records any hit: the prober
	// must never reach it, because filtering happens before probing.
	var templatedHits int32
	var mu sync.Mutex
	templated := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		templatedHits++
		mu.Unlock()
	}))
	t.Cleanup(templated.Close)

	feed := feedServer(t, []Chain{
		{ChainID: 1, Name: "Ethereum Mainnet", RPC: []string{"https://unused.example"}},
		{ChainID: chainID, Name: "PulseChain", RPC: []string{
			good.URL,
			wrong.URL,
			broken.URL,
			templated.URL + "/v3/${API_KEY}",
		}},
	})

	d := New()
	d.FeedURL = feed.URL
	d.ProbeTimeout = 2 * time.Second

	res, err := d.Discover(context.Background(), chainID)
	if err != nil {
		t.Fatalf("Discover: unexpected error: %v", err)
	}
	if res.Source != SourceFeed {
		t.Errorf("Source = %q, want %q", res.Source, SourceFeed)
	}
	if res.ChainID != chainID {
		t.Errorf("ChainID = %d, want %d", res.ChainID, chainID)
	}

	live := res.Live()
	if !equalStrings(urlsOf(live), []string{good.URL}) {
		t.Fatalf("Live() = %v, want [%s]", urlsOf(live), good.URL)
	}
	if live[0].ChainID != chainID {
		t.Errorf("live endpoint ChainID = %d, want %d", live[0].ChainID, chainID)
	}

	index := byURL(t, res)
	if ep := index[wrong.URL]; ep.Status != StatusRejected || ep.ChainID != 1 {
		t.Errorf("mismatching endpoint = %+v, want rejected with ChainID 1", ep)
	} else if !strings.Contains(ep.Reason, "1") || !strings.Contains(strings.ToLower(ep.Reason), "chain") {
		t.Errorf("mismatch reason = %q, want it to name the reported chain id", ep.Reason)
	}
	if ep := index[broken.URL]; ep.Status != StatusRejected {
		t.Errorf("erroring endpoint = %+v, want rejected", ep)
	} else if ep.Reason == "" {
		t.Errorf("erroring endpoint has empty Reason")
	}

	mu.Lock()
	hits := templatedHits
	mu.Unlock()
	if hits != 0 {
		t.Errorf("templated URL was probed %d times, want 0", hits)
	}
	if len(res.Rejected()) != 3 {
		t.Errorf("Rejected() = %v, want 3 entries", urlsOf(res.Rejected()))
	}
	if !equalStrings(res.URLs(), []string{good.URL}) {
		t.Errorf("URLs() = %v, want [%s]", res.URLs(), good.URL)
	}
}

func TestDiscover_WebSocketProbe(t *testing.T) {
	const chainID = 943

	okWS := wsRPCServer(t, chainID, 0)
	wrongWS := wsRPCServer(t, 1, 0)
	feed := feedServer(t, []Chain{{ChainID: chainID, Name: "PulseChain Testnet v4", RPC: []string{
		wsURL(okWS), wsURL(wrongWS),
	}}})

	d := New()
	d.FeedURL = feed.URL
	d.ProbeTimeout = 3 * time.Second

	res, err := d.Discover(context.Background(), chainID)
	if err != nil {
		t.Fatalf("Discover: unexpected error: %v", err)
	}
	index := byURL(t, res)
	if ep := index[wsURL(okWS)]; ep.Status != StatusLive || ep.Kind != KindWS || ep.ChainID != chainID {
		t.Errorf("ws endpoint = %+v, want live ws on chain %d", ep, chainID)
	}
	if ep := index[wsURL(wrongWS)]; ep.Status != StatusRejected {
		t.Errorf("mismatching ws endpoint = %+v, want rejected", ep)
	}
}

func TestDiscover_WebSocketProbingDisabled(t *testing.T) {
	const chainID = 369

	// Point at a closed port: if the prober dialled it despite ProbeWS being
	// off, the endpoint would come back rejected rather than unprobed.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	dead := "ws://" + ln.Addr().String()
	if err := ln.Close(); err != nil {
		t.Fatalf("close listener: %v", err)
	}

	feed := feedServer(t, []Chain{{ChainID: chainID, Name: "PulseChain", RPC: []string{dead}}})

	d := New()
	d.FeedURL = feed.URL
	d.ProbeWS = false

	res, err := d.Discover(context.Background(), chainID)
	if err != nil {
		t.Fatalf("Discover: unexpected error: %v", err)
	}
	if got := urlsOf(res.Unprobed()); !equalStrings(got, []string{dead}) {
		t.Fatalf("Unprobed() = %v, want [%s]", got, dead)
	}
	if got := res.URLs(); !equalStrings(got, []string{dead}) {
		t.Errorf("URLs() = %v, want unprobed ws endpoints included: [%s]", got, dead)
	}
	if len(res.Live()) != 0 {
		t.Errorf("Live() = %v, want empty (ws probing disabled)", urlsOf(res.Live()))
	}
}

func TestDiscover_PerEndpointTimeout(t *testing.T) {
	const chainID = 369

	slow := rpcServer(t, chainID, func(r *http.Request) { time.Sleep(500 * time.Millisecond) })
	fast := rpcServer(t, chainID, nil)
	feed := feedServer(t, []Chain{{ChainID: chainID, Name: "PulseChain", RPC: []string{slow.URL, fast.URL}}})

	d := New()
	d.FeedURL = feed.URL
	d.ProbeTimeout = 50 * time.Millisecond

	start := time.Now()
	res, err := d.Discover(context.Background(), chainID)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("Discover: unexpected error: %v", err)
	}
	if elapsed > 400*time.Millisecond {
		t.Errorf("Discover took %v; the per-endpoint timeout (%v) was not enforced", elapsed, d.ProbeTimeout)
	}
	index := byURL(t, res)
	if ep := index[slow.URL]; ep.Status != StatusRejected {
		t.Errorf("slow endpoint = %+v, want rejected on timeout", ep)
	}
	if ep := index[fast.URL]; ep.Status != StatusLive {
		t.Errorf("fast endpoint = %+v, want live", ep)
	}
}

func TestDiscover_ProbesConcurrently(t *testing.T) {
	const chainID = 369
	const n = 8

	// Every handler blocks on the same barrier: the batch can only complete
	// if all n probes are in flight at once. A serial prober deadlocks here
	// and every endpoint times out.
	var barrier sync.WaitGroup
	barrier.Add(n)
	block := func(r *http.Request) {
		barrier.Done()
		barrier.Wait()
	}

	rpcs := make([]string, 0, n)
	for i := 0; i < n; i++ {
		rpcs = append(rpcs, rpcServer(t, chainID, block).URL)
	}
	feed := feedServer(t, []Chain{{ChainID: chainID, Name: "PulseChain", RPC: rpcs}})

	d := New()
	d.FeedURL = feed.URL
	d.Concurrency = n
	d.ProbeTimeout = 3 * time.Second

	res, err := d.Discover(context.Background(), chainID)
	if err != nil {
		t.Fatalf("Discover: unexpected error: %v", err)
	}
	if got := urlsOf(res.Live()); !equalStrings(got, rpcs) {
		t.Fatalf("Live() = %v, want all %d endpoints live and in feed order (%v)", got, n, rpcs)
	}
}

func TestDiscover_ConcurrencyLimitIsRespected(t *testing.T) {
	const chainID = 369
	const n = 6
	const limit = 2

	var mu sync.Mutex
	var inFlight, peak int
	track := func(r *http.Request) {
		mu.Lock()
		inFlight++
		if inFlight > peak {
			peak = inFlight
		}
		mu.Unlock()
		time.Sleep(20 * time.Millisecond)
		mu.Lock()
		inFlight--
		mu.Unlock()
	}

	rpcs := make([]string, 0, n)
	for i := 0; i < n; i++ {
		rpcs = append(rpcs, rpcServer(t, chainID, track).URL)
	}
	feed := feedServer(t, []Chain{{ChainID: chainID, Name: "PulseChain", RPC: rpcs}})

	d := New()
	d.FeedURL = feed.URL
	d.Concurrency = limit
	d.ProbeTimeout = 3 * time.Second

	if _, err := d.Discover(context.Background(), chainID); err != nil {
		t.Fatalf("Discover: unexpected error: %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if peak > limit {
		t.Errorf("peak concurrent probes = %d, want <= %d", peak, limit)
	}
}

// ---------------------------------------------------------------------------
// Vendored fallback
// ---------------------------------------------------------------------------

func TestDiscover_FetchFailureFallsBackToVendored(t *testing.T) {
	const chainID = 369

	feedErr := errors.New("simulated dns failure")
	var probed []string
	var mu sync.Mutex

	d := New()
	d.FeedURL = "https://chainid.network/chains.json"
	d.ProbeWS = false // vendored lists contain wss:// entries; keep the test off the network
	d.HTTPClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.String() == d.FeedURL {
			return nil, feedErr
		}
		mu.Lock()
		probed = append(probed, r.URL.String())
		mu.Unlock()
		return jsonResponse(r, fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"result":"0x%x"}`, chainID)), nil
	})}

	res, err := d.Discover(context.Background(), chainID)
	if err != nil {
		t.Fatalf("Discover: unexpected error: %v", err)
	}
	if res.Source != SourceVendored {
		t.Fatalf("Source = %q, want %q", res.Source, SourceVendored)
	}
	if !errors.Is(res.FetchErr, feedErr) {
		t.Errorf("FetchErr = %v, want it to wrap %v", res.FetchErr, feedErr)
	}

	want, ok := Vendored(chainID)
	if !ok {
		t.Fatalf("Vendored(%d) missing", chainID)
	}
	if !equalStrings(urlsOf(res.Endpoints), want) {
		t.Errorf("Endpoints = %v, want the vendored list %v", urlsOf(res.Endpoints), want)
	}
	// Every https vendored entry answered with the right chain id, so it is
	// live; the wss ones are unprobed. Both belong in URLs().
	if len(res.URLs()) != len(want) {
		t.Errorf("URLs() = %v, want all %d vendored entries usable", res.URLs(), len(want))
	}
	mu.Lock()
	defer mu.Unlock()
	if len(probed) == 0 {
		t.Errorf("no vendored endpoint was probed")
	}
}

func TestDiscover_FetchFailureWithoutVendoredIsAnError(t *testing.T) {
	feed := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusBadGateway)
	}))
	t.Cleanup(feed.Close)

	d := New()
	d.FeedURL = feed.URL

	if _, err := d.Discover(context.Background(), 12345); err == nil {
		t.Fatal("Discover: want an error for an unknown chain with no vendored fallback, got nil")
	}
}

func TestDiscover_ChainAbsentFromFeedUsesVendored(t *testing.T) {
	const chainID = 943

	feed := feedServer(t, []Chain{{ChainID: 1, Name: "Ethereum Mainnet", RPC: []string{"https://unused.example"}}})

	d := New()
	d.FeedURL = feed.URL
	d.ProbeWS = false
	d.HTTPClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if strings.HasPrefix(r.URL.String(), feed.URL) {
			return http.DefaultTransport.RoundTrip(r)
		}
		return jsonResponse(r, fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"result":"0x%x"}`, chainID)), nil
	})}

	res, err := d.Discover(context.Background(), chainID)
	if err != nil {
		t.Fatalf("Discover: unexpected error: %v", err)
	}
	if res.Source != SourceVendored {
		t.Errorf("Source = %q, want %q", res.Source, SourceVendored)
	}
	if res.FetchErr == nil {
		t.Errorf("FetchErr = nil, want the chain-not-in-feed reason recorded")
	}
}

func TestVendored(t *testing.T) {
	// The known-good sets, verified against the live feed on the date recorded
	// in vendored.go.
	want := map[int][]string{
		1: {
			"https://ethereum-rpc.publicnode.com",
			"wss://ethereum-rpc.publicnode.com",
			"https://cloudflare-eth.com",
			"https://mainnet.gateway.tenderly.co",
		},
		369: {
			"https://rpc.pulsechain.com",
			"https://pulsechain-rpc.publicnode.com",
			"wss://pulsechain-rpc.publicnode.com",
			"https://rpc-pulsechain.g4mm4.io",
		},
		943: {
			"https://rpc.v4.testnet.pulsechain.com",
			"wss://rpc.v4.testnet.pulsechain.com",
			"https://pulsechain-testnet-rpc.publicnode.com",
			"wss://pulsechain-testnet-rpc.publicnode.com",
			"https://rpc-testnet-pulsechain.g4mm4.io",
			"wss://rpc-testnet-pulsechain.g4mm4.io",
		},
	}

	for chainID, urls := range want {
		got, ok := Vendored(chainID)
		if !ok {
			t.Fatalf("Vendored(%d): not present", chainID)
		}
		for _, u := range urls {
			found := false
			for _, g := range got {
				if g == u {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Vendored(%d) = %v, missing known-good %q", chainID, got, u)
			}
		}
		// A vendored snapshot must itself survive the candidate filter: no
		// templated URLs, no unclassifiable schemes.
		for _, ep := range Candidates(got) {
			if ep.Status != StatusPending {
				t.Errorf("Vendored(%d): %q would be rejected: %s", chainID, ep.URL, ep.Reason)
			}
		}
	}

	if _, ok := Vendored(12345); ok {
		t.Errorf("Vendored(12345) = present, want absent")
	}

	// Mutating the returned slice must not corrupt the package-level snapshot.
	got, _ := Vendored(369)
	got[0] = "https://evil.example"
	again, _ := Vendored(369)
	if again[0] == "https://evil.example" {
		t.Errorf("Vendored returns the backing array; callers can corrupt the snapshot")
	}
}

// ---------------------------------------------------------------------------
// Fetch
// ---------------------------------------------------------------------------

func TestFetch(t *testing.T) {
	feed := feedServer(t, []Chain{
		{ChainID: 1, Name: "Ethereum Mainnet", RPC: []string{"https://cloudflare-eth.com"}},
		{ChainID: 369, Name: "PulseChain", RPC: []string{"https://rpc.pulsechain.com"}},
	})

	d := New()
	d.FeedURL = feed.URL

	chains, err := d.Fetch(context.Background())
	if err != nil {
		t.Fatalf("Fetch: unexpected error: %v", err)
	}
	if len(chains) != 2 {
		t.Fatalf("Fetch returned %d chains, want 2", len(chains))
	}
	if chains[1].ChainID != 369 || chains[1].Name != "PulseChain" {
		t.Errorf("chains[1] = %+v, want chain 369 PulseChain", chains[1])
	}
}

func TestFetch_BadStatusAndBadJSON(t *testing.T) {
	tests := []struct {
		name    string
		handler http.HandlerFunc
	}{
		{"http error", func(w http.ResponseWriter, r *http.Request) { http.Error(w, "nope", http.StatusBadGateway) }},
		{"malformed json", func(w http.ResponseWriter, r *http.Request) { _, _ = io.WriteString(w, "{not json") }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(tt.handler)
			t.Cleanup(srv.Close)
			d := New()
			d.FeedURL = srv.URL
			if _, err := d.Fetch(context.Background()); err == nil {
				t.Fatal("Fetch: want an error, got nil")
			}
		})
	}
}

func TestDiscover_ContextCancellation(t *testing.T) {
	feed := feedServer(t, []Chain{{ChainID: 369, Name: "PulseChain", RPC: []string{"https://rpc.pulsechain.com"}}})

	d := New()
	d.FeedURL = feed.URL

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := d.Discover(ctx, 369); err == nil {
		t.Fatal("Discover with a cancelled context: want an error, got nil")
	}
}
