package capabilities

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// helpers
//
// Everything here is served in-process. No test in this package opens a socket
// to anything it did not start itself.
// ---------------------------------------------------------------------------

const testChainID = 943

// reply is one scripted JSON-RPC answer.
type reply struct {
	body   string
	delay  time.Duration
	status int // non-zero to answer with an HTTP error instead
}

// script maps a JSON-RPC method to its scripted answer. The key "batch" covers
// an array request. Methods with no entry get the -32601 an honest node sends.
type script map[string]reply

func okResult(result string) reply {
	return reply{body: fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"result":%s}`, result)}
}

func rpcErrReply(code int, msg string) reply {
	b, _ := json.Marshal(msg)
	return reply{body: fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"error":{"code":%d,"message":%s}}`, code, b)}
}

func defaultReply(method string) reply {
	switch method {
	case "eth_chainId":
		return okResult(fmt.Sprintf("%q", fmt.Sprintf("0x%x", testChainID)))
	case "batch":
		return reply{body: `[{"jsonrpc":"2.0","id":1,"result":"0x3af"},{"jsonrpc":"2.0","id":2,"result":"0x1"}]`}
	}
	return rpcErrReply(-32601, "the method "+method+" does not exist/is not available")
}

// counter records how many scripted requests are in flight at once, so the
// concurrency test can assert on observed parallelism rather than on a clock.
type counter struct {
	mu        sync.Mutex
	cur, peak int
}

func (c *counter) enter() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cur++
	if c.cur > c.peak {
		c.peak = c.cur
	}
}

func (c *counter) leave() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cur--
}

func (c *counter) max() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.peak
}

// rpcServer serves JSON-RPC from a script.
func rpcServer(t *testing.T, s script, c *counter) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if c != nil {
			c.enter()
			defer c.leave()
		}
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "read", http.StatusBadRequest)
			return
		}

		key := "batch"
		if bytes.HasPrefix(bytes.TrimSpace(raw), []byte("{")) {
			var req struct {
				Method string `json:"method"`
			}
			if err := json.Unmarshal(raw, &req); err != nil {
				http.Error(w, "bad json", http.StatusBadRequest)
				return
			}
			key = req.Method
		}

		rep, ok := s[key]
		if !ok {
			rep = defaultReply(key)
		}
		if rep.delay > 0 {
			select {
			case <-time.After(rep.delay):
			case <-r.Context().Done():
				return
			}
		}
		if rep.status != 0 {
			http.Error(w, rep.body, rep.status)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, rep.body)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// testProber is a Prober wired for tests: short timeouts, WebSocket probing off
// by default so a capability test is not also a transport test.
func testProber() *Prober {
	p := NewProber()
	p.ProbeTimeout = 2 * time.Second
	p.ProbeWS = false
	return p
}

func wsScheme(rawURL string) string {
	return "ws://" + strings.TrimPrefix(rawURL, "http://")
}

func boolPtr(b bool) *bool { return &b }

// ---------------------------------------------------------------------------
// WebSocket test servers, server side of RFC 6455 — enough to accept a
// handshake, read one masked client frame and answer, or to refuse in each of
// the ways a real endpoint refuses.
// ---------------------------------------------------------------------------

type wsMode int

const (
	wsAnswer  wsMode = iota // full handshake, answers eth_chainId
	wsRefuse                // refuses the Upgrade outright
	wsBadHash               // 101, but a wrong Sec-WebSocket-Accept digest
	wsSilent                // 101 with a valid digest, then says nothing
)

func wsServer(t *testing.T, mode wsMode, chainID int) *httptest.Server {
	t.Helper()
	stop := make(chan struct{})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if mode == wsRefuse {
			http.Error(w, "websockets not enabled on this endpoint", http.StatusBadRequest)
			return
		}
		sum := sha1.Sum([]byte(r.Header.Get("Sec-WebSocket-Key") + wsGUID))
		accept := base64.StdEncoding.EncodeToString(sum[:])
		if mode == wsBadHash {
			accept = base64.StdEncoding.EncodeToString([]byte("not the right digest"))
		}

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
			"Sec-WebSocket-Accept: " + accept + "\r\n\r\n")
		if err := rw.Flush(); err != nil {
			return
		}
		if mode != wsAnswer {
			<-stop // hold the socket open without answering
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
		_ = testWriteFrame(conn, fmt.Appendf(nil, `{"jsonrpc":"2.0","id":1,"result":"0x%x"}`, chainID))
	}))
	// Registered after the server's own cleanup, so LIFO runs it first and a
	// held-open handler is released before Close waits on anything.
	t.Cleanup(srv.Close)
	t.Cleanup(func() { close(stop) })
	return srv
}

// testReadFrame reads one masked client frame (server side).
func testReadFrame(br *bufio.Reader) ([]byte, error) {
	var hdr [2]byte
	if _, err := io.ReadFull(br, hdr[:]); err != nil {
		return nil, err
	}
	length := uint64(hdr[1] & 0x7F)
	switch length {
	case 126:
		var ext [2]byte
		if _, err := io.ReadFull(br, ext[:]); err != nil {
			return nil, err
		}
		length = uint64(binary.BigEndian.Uint16(ext[:]))
	case 127:
		var ext [8]byte
		if _, err := io.ReadFull(br, ext[:]); err != nil {
			return nil, err
		}
		length = binary.BigEndian.Uint64(ext[:])
	}
	var mask [4]byte
	if hdr[1]&0x80 != 0 {
		if _, err := io.ReadFull(br, mask[:]); err != nil {
			return nil, err
		}
	}
	payload := make([]byte, length)
	if _, err := io.ReadFull(br, payload); err != nil {
		return nil, err
	}
	if hdr[1]&0x80 != 0 {
		for i := range payload {
			payload[i] ^= mask[i%4]
		}
	}
	return payload, nil
}

// testWriteFrame writes one unmasked server text frame.
func testWriteFrame(w io.Writer, payload []byte) error {
	var buf bytes.Buffer
	buf.WriteByte(0x81)
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

// ---------------------------------------------------------------------------
// Reachability
// ---------------------------------------------------------------------------

func TestProbeReachability(t *testing.T) {
	tests := []struct {
		name        string
		script      script
		wantReach   bool
		wantChainOK *bool
		wantChainID int
		wantDetail  string
	}{
		{
			name:        "right chain",
			script:      script{},
			wantReach:   true,
			wantChainOK: boolPtr(true),
			wantChainID: testChainID,
			wantDetail:  "matches",
		},
		{
			name:        "wrong chain is reachable but flagged",
			script:      script{"eth_chainId": okResult(`"0x1"`)},
			wantReach:   true,
			wantChainOK: boolPtr(false),
			wantChainID: 1,
			wantDetail:  "wrong chain",
		},
		{
			name:        "decimal chain id is understood",
			script:      script{"eth_chainId": okResult(`"943"`)},
			wantReach:   true,
			wantChainOK: boolPtr(true),
			wantChainID: testChainID,
			wantDetail:  "matches",
		},
		{
			name:        "error reply is reachable with the chain unclaimed",
			script:      script{"eth_chainId": rpcErrReply(-32000, "rate limited")},
			wantReach:   true,
			wantChainOK: nil,
			wantDetail:  "chain unconfirmed",
		},
		{
			name:        "uninterpretable result leaves the chain unclaimed",
			script:      script{"eth_chainId": okResult(`"banana"`)},
			wantReach:   true,
			wantChainOK: nil,
			wantDetail:  "chain unconfirmed",
		},
		{
			name:        "non-json body is unreachable",
			script:      script{"eth_chainId": reply{body: "<html>gateway</html>"}},
			wantReach:   false,
			wantChainOK: nil,
			wantDetail:  "unreachable",
		},
		{
			name:        "http error is unreachable",
			script:      script{"eth_chainId": reply{status: http.StatusBadGateway, body: "nope"}},
			wantReach:   false,
			wantChainOK: nil,
			wantDetail:  "502",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := rpcServer(t, tt.script, nil)
			ep := testProber().Probe(context.Background(), Target{URL: srv.URL}, testChainID)

			if ep.Reachable != tt.wantReach {
				t.Errorf("Reachable = %v, want %v (detail %q)", ep.Reachable, tt.wantReach, ep.ReachDetail)
			}
			switch {
			case tt.wantChainOK == nil && ep.ChainOK != nil:
				t.Errorf("ChainOK = %v, want nil (tri-state must not be flattened)", *ep.ChainOK)
			case tt.wantChainOK != nil && ep.ChainOK == nil:
				t.Errorf("ChainOK = nil, want %v", *tt.wantChainOK)
			case tt.wantChainOK != nil && *ep.ChainOK != *tt.wantChainOK:
				t.Errorf("ChainOK = %v, want %v", *ep.ChainOK, *tt.wantChainOK)
			}
			if tt.wantChainID != 0 && ep.ChainID != tt.wantChainID {
				t.Errorf("ChainID = %d, want %d", ep.ChainID, tt.wantChainID)
			}
			if !strings.Contains(ep.ReachDetail, tt.wantDetail) {
				t.Errorf("ReachDetail = %q, want it to mention %q", ep.ReachDetail, tt.wantDetail)
			}
			if ep.Origin != OriginLocal {
				t.Errorf("Origin = %q, want %q", ep.Origin, OriginLocal)
			}

			// An unreachable endpoint gets no capability cells at all: eleven
			// inconclusive cells would be noise where one honest verdict fits.
			if !tt.wantReach && len(ep.Capabilities) != 0 {
				t.Errorf("unreachable endpoint carried %d capability results, want 0", len(ep.Capabilities))
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Capability classification
// ---------------------------------------------------------------------------

func TestProbeCapabilityClassification(t *testing.T) {
	tests := []struct {
		name       string
		script     script
		key        string
		wantStatus Status
		wantMethod string
		wantDetail string
	}{
		{
			name:       "trace present",
			script:     script{"trace_transaction": okResult(`[]`)},
			key:        KeyTrace,
			wantStatus: StatusSupported,
			wantMethod: "trace_transaction",
			wantDetail: "returned a result",
		},
		{
			name:       "trace absent via -32601",
			script:     script{"trace_transaction": rpcErrReply(-32601, "Method not found")},
			key:        KeyTrace,
			wantStatus: StatusUnsupported,
			wantMethod: "trace_transaction",
			wantDetail: "-32601",
		},
		{
			name: "trace absent via prose without the standard code",
			// Some gateways answer -32000 with a sentence. The message is the
			// only evidence available and it is unambiguous.
			script:     script{"trace_transaction": rpcErrReply(-32000, "unsupported method: trace_transaction")},
			key:        KeyTrace,
			wantStatus: StatusUnsupported,
			wantMethod: "trace_transaction",
			wantDetail: "unsupported method",
		},
		{
			name: "debug present because the node ran it and complained",
			// "transaction not found" proves the method exists: the node had to
			// route the call and execute it to discover the hash was bogus.
			script:     script{"debug_traceTransaction": rpcErrReply(-32000, "transaction 0x0 not found")},
			key:        KeyDebug,
			wantStatus: StatusSupported,
			wantMethod: "debug_traceTransaction",
			wantDetail: "method exists",
		},
		{
			name:       "debug absent",
			script:     script{},
			key:        KeyDebug,
			wantStatus: StatusUnsupported,
			wantMethod: "debug_traceTransaction",
			wantDetail: "does not exist",
		},
		{
			name:       "logs present",
			script:     script{"eth_getLogs": okResult(`[]`)},
			key:        KeyLogs,
			wantStatus: StatusSupported,
			wantMethod: "eth_getLogs",
		},
		{
			name:       "logs rejected outright",
			script:     script{"eth_getLogs": rpcErrReply(-32601, "eth_getLogs is not available on this tier")},
			key:        KeyLogs,
			wantStatus: StatusUnsupported,
			wantMethod: "eth_getLogs",
		},
		{
			name:       "logs inconclusive when the gateway rate-limits",
			script:     script{"eth_getLogs": reply{status: http.StatusTooManyRequests, body: "slow down"}},
			key:        KeyLogs,
			wantStatus: StatusInconclusive,
			wantMethod: "eth_getLogs",
			wantDetail: "429",
		},
		{
			name:       "filters present",
			script:     script{"eth_newBlockFilter": okResult(`"0x1"`)},
			key:        KeyFilters,
			wantStatus: StatusSupported,
			wantMethod: "eth_newBlockFilter",
		},
		{
			name:       "filters absent",
			script:     script{},
			key:        KeyFilters,
			wantStatus: StatusUnsupported,
			wantMethod: "eth_newBlockFilter",
		},
		{
			name:       "otterscan present",
			script:     script{"ots_getApiLevel": okResult(`8`)},
			key:        KeyOtterscan,
			wantStatus: StatusSupported,
			wantMethod: "ots_getApiLevel",
		},
		{
			name:       "otterscan absent",
			script:     script{},
			key:        KeyOtterscan,
			wantStatus: StatusUnsupported,
			wantMethod: "ots_getApiLevel",
		},
		{
			name:       "txpool present",
			script:     script{"txpool_status": okResult(`{"pending":"0x0","queued":"0x0"}`)},
			key:        KeyTxpool,
			wantStatus: StatusSupported,
			wantMethod: "txpool_status",
		},
		{
			name:       "msgboard present",
			script:     script{"msgboard_status": okResult(`{"ok":true}`)},
			key:        KeyMsgboard,
			wantStatus: StatusSupported,
			wantMethod: "msgboard_status",
		},
		{
			name:       "msgboard absent",
			script:     script{},
			key:        KeyMsgboard,
			wantStatus: StatusUnsupported,
			wantMethod: "msgboard_status",
		},
		{
			name:       "archive present",
			script:     script{"eth_getBalance": okResult(`"0x0"`)},
			key:        KeyArchive,
			wantStatus: StatusSupported,
			wantMethod: "eth_getBalance",
			wantDetail: "historical state",
		},
		{
			name:       "archive absent on a pruned node",
			script:     script{"eth_getBalance": rpcErrReply(-32000, "missing trie node 0xabc (path )")},
			key:        KeyArchive,
			wantStatus: StatusUnsupported,
			wantMethod: "eth_getBalance",
			wantDetail: "pruned",
		},
		{
			name:       "archive absent when the node says it has no state",
			script:     script{"eth_getBalance": rpcErrReply(-32000, "state at block 1 does not have state")},
			key:        KeyArchive,
			wantStatus: StatusUnsupported,
			wantMethod: "eth_getBalance",
		},
		{
			name: "archive inconclusive on an error we cannot read",
			// An unrecognised error is not evidence of pruning. Refusing to
			// guess here is the whole reason the third state exists.
			script:     script{"eth_getBalance": rpcErrReply(-32000, "upstream returned 502")},
			key:        KeyArchive,
			wantStatus: StatusInconclusive,
			wantMethod: "eth_getBalance",
		},
		{
			name:       "archive inconclusive on a null result",
			script:     script{"eth_getBalance": okResult(`null`)},
			key:        KeyArchive,
			wantStatus: StatusInconclusive,
			wantMethod: "eth_getBalance",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := rpcServer(t, tt.script, nil)
			ep := testProber().Probe(context.Background(), Target{URL: srv.URL}, testChainID)

			got, ok := ep.Cap(tt.key)
			if !ok {
				t.Fatalf("no result recorded for %q", tt.key)
			}
			if got.Status != tt.wantStatus {
				t.Errorf("%s status = %q, want %q (detail %q)", tt.key, got.Status, tt.wantStatus, got.Detail)
			}
			if tt.wantMethod != "" && got.Method != tt.wantMethod {
				t.Errorf("%s method = %q, want %q", tt.key, got.Method, tt.wantMethod)
			}
			if tt.wantDetail != "" && !strings.Contains(got.Detail, tt.wantDetail) {
				t.Errorf("%s detail = %q, want it to mention %q", tt.key, got.Detail, tt.wantDetail)
			}
			if got.Detail == "" {
				t.Errorf("%s carries no detail; every cell must be explainable", tt.key)
			}
			if got.Origin != OriginLocal {
				t.Errorf("%s origin = %q, want %q", tt.key, got.Origin, OriginLocal)
			}
		})
	}
}

// TestProbeEveryKeyIsAnswered pins the contract that a reachable endpoint gets a
// verdict for every column — a missing cell renders as "not applicable", which
// is a claim we are not entitled to make.
func TestProbeEveryKeyIsAnswered(t *testing.T) {
	srv := rpcServer(t, script{}, nil)
	ep := testProber().Probe(context.Background(), Target{URL: srv.URL}, testChainID)

	for _, key := range Keys() {
		res, ok := ep.Cap(key)
		if !ok {
			t.Errorf("no result for %q", key)
			continue
		}
		switch res.Status {
		case StatusSupported, StatusUnsupported, StatusInconclusive:
		default:
			t.Errorf("%q status = %q, want one of the three states", key, res.Status)
		}
	}
}

// ---------------------------------------------------------------------------
// Batch
// ---------------------------------------------------------------------------

func TestProbeBatch(t *testing.T) {
	tests := []struct {
		name       string
		reply      reply
		wantStatus Status
		wantDetail string
	}{
		{
			name:       "two element array is batching",
			reply:      defaultReply("batch"),
			wantStatus: StatusSupported,
			wantDetail: "2-element array",
		},
		{
			name:       "single object reply is not batching",
			reply:      okResult(`"0x3af"`),
			wantStatus: StatusUnsupported,
			wantDetail: "not a 2-element array",
		},
		{
			name:       "one element array answered only the first item",
			reply:      reply{body: `[{"jsonrpc":"2.0","id":1,"result":"0x3af"}]`},
			wantStatus: StatusUnsupported,
			wantDetail: "1-element array",
		},
		{
			name:       "http error is inconclusive, not a refusal",
			reply:      reply{status: http.StatusTooManyRequests, body: "slow down"},
			wantStatus: StatusInconclusive,
			wantDetail: "429",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := rpcServer(t, script{"batch": tt.reply}, nil)
			ep := testProber().Probe(context.Background(), Target{URL: srv.URL}, testChainID)

			got, ok := ep.Cap(KeyBatch)
			if !ok {
				t.Fatalf("no batch result")
			}
			if got.Status != tt.wantStatus {
				t.Errorf("batch status = %q, want %q (detail %q)", got.Status, tt.wantStatus, got.Detail)
			}
			if !strings.Contains(got.Detail, tt.wantDetail) {
				t.Errorf("batch detail = %q, want it to mention %q", got.Detail, tt.wantDetail)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// WebSocket
// ---------------------------------------------------------------------------

func TestProbeWS(t *testing.T) {
	tests := []struct {
		name       string
		mode       wsMode
		chainID    int
		wantStatus Status
		wantDetail string
	}{
		{
			name:       "handshake and chain id over the socket",
			mode:       wsAnswer,
			chainID:    testChainID,
			wantStatus: StatusSupported,
			wantDetail: "handshake OK",
		},
		{
			name: "transport works even when the chain is wrong",
			// The ws column asks whether the socket works, not which chain is
			// behind it; wrong-chain belongs in the reachability column.
			mode:       wsAnswer,
			chainID:    1,
			wantStatus: StatusSupported,
			wantDetail: "expected 943",
		},
		{
			name:       "upgrade refused is a definite no",
			mode:       wsRefuse,
			chainID:    testChainID,
			wantStatus: StatusUnsupported,
			wantDetail: "400",
		},
		{
			name: "a 101 with a bad digest is not a websocket",
			// Without the digest check we would happily "talk WebSocket" to
			// anything that echoed a 101 — the exact confusion this probe is
			// here to catch.
			mode:       wsBadHash,
			chainID:    testChainID,
			wantStatus: StatusUnsupported,
			wantDetail: "Sec-WebSocket-Accept",
		},
		{
			name: "handshake then silence is inconclusive, and says so",
			// This is the eRPC scheme-inference failure: the upgrade succeeds
			// because the proxy assumed WebSocket from the URL, and then
			// nothing comes back.
			mode:       wsSilent,
			chainID:    testChainID,
			wantStatus: StatusInconclusive,
			wantDetail: "handshake succeeded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := wsServer(t, tt.mode, tt.chainID)
			p := NewProber()
			p.ProbeTimeout = 500 * time.Millisecond

			got := p.probeWS(context.Background(), wsScheme(srv.URL), testChainID)
			if got.Status != tt.wantStatus {
				t.Errorf("ws status = %q, want %q (detail %q)", got.Status, tt.wantStatus, got.Detail)
			}
			if !strings.Contains(got.Detail, tt.wantDetail) {
				t.Errorf("ws detail = %q, want it to mention %q", got.Detail, tt.wantDetail)
			}
			if got.Method != "ws" {
				t.Errorf("ws method = %q, want %q", got.Method, "ws")
			}
		})
	}
}

func TestProbeWSDisabledIsInconclusiveNotAbsent(t *testing.T) {
	srv := rpcServer(t, script{}, nil)
	ep := testProber().Probe(context.Background(), Target{URL: srv.URL}, testChainID)

	got, ok := ep.Cap(KeyWS)
	if !ok {
		t.Fatalf("ws cell missing; choosing not to probe is not the same as not applicable")
	}
	if got.Status != StatusInconclusive {
		t.Errorf("ws status = %q, want %q", got.Status, StatusInconclusive)
	}
}

func TestDeriveWSURL(t *testing.T) {
	tests := []struct {
		in     string
		want   string
		wantOK bool
	}{
		{in: "https://rpc.example.com", want: "wss://rpc.example.com", wantOK: true},
		{in: "http://127.0.0.1:8600", want: "ws://127.0.0.1:8600", wantOK: true},
		{in: "https://rpc.example.com/v1/key", want: "wss://rpc.example.com/v1/key", wantOK: true},
		{in: "wss://rpc.example.com", want: "wss://rpc.example.com", wantOK: true},
		{in: "ws://127.0.0.1:8601", want: "ws://127.0.0.1:8601", wantOK: true},
		{in: "ipc:///tmp/geth.ipc", wantOK: false},
		{in: "not a url", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, ok := DeriveWSURL(tt.in)
			if ok != tt.wantOK {
				t.Fatalf("DeriveWSURL(%q) ok = %v, want %v", tt.in, ok, tt.wantOK)
			}
			if ok && got != tt.want {
				t.Errorf("DeriveWSURL(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Timeouts and concurrency
// ---------------------------------------------------------------------------

// TestProbePerProbeTimeout pins the property that one black-holed method costs
// one timeout, not the endpoint's whole row.
func TestProbePerProbeTimeout(t *testing.T) {
	srv := rpcServer(t, script{
		"debug_traceTransaction": {delay: 5 * time.Second},
		"trace_transaction":      okResult(`[]`),
	}, nil)

	p := testProber()
	p.ProbeTimeout = 150 * time.Millisecond

	start := time.Now()
	ep := p.Probe(context.Background(), Target{URL: srv.URL}, testChainID)
	elapsed := time.Since(start)

	if !ep.Reachable {
		t.Fatalf("endpoint not reachable: %s", ep.ReachDetail)
	}
	debug, _ := ep.Cap(KeyDebug)
	if debug.Status != StatusInconclusive {
		t.Errorf("debug status = %q, want %q — a timeout is not a refusal", debug.Status, StatusInconclusive)
	}
	if !strings.Contains(debug.Detail, "timed out") {
		t.Errorf("debug detail = %q, want it to say it timed out", debug.Detail)
	}
	if trace, _ := ep.Cap(KeyTrace); trace.Status != StatusSupported {
		t.Errorf("trace status = %q, want %q — one slow method must not poison the row", trace.Status, StatusSupported)
	}
	if elapsed > 2*time.Second {
		t.Errorf("probe took %v; the hanging method should have cost one 150ms timeout", elapsed)
	}
}

// TestProbeAllIsConcurrentAndBounded asserts observed parallelism rather than
// wall-clock, and that the bound is honoured.
func TestProbeAllIsConcurrentAndBounded(t *testing.T) {
	c := &counter{}
	srv := rpcServer(t, script{}, c)

	const limit = 4
	p := testProber()
	p.Concurrency = limit

	targets := make([]Target, 6)
	for i := range targets {
		targets[i] = Target{URL: srv.URL, Label: fmt.Sprintf("ep-%d", i)}
	}

	eps := p.ProbeAll(context.Background(), targets, testChainID)

	if len(eps) != len(targets) {
		t.Fatalf("got %d endpoints, want %d", len(eps), len(targets))
	}
	for i, ep := range eps {
		if want := fmt.Sprintf("ep-%d", i); ep.Label != want {
			t.Errorf("endpoint %d label = %q, want %q — target order must be preserved", i, ep.Label, want)
		}
		if !ep.Reachable {
			t.Errorf("endpoint %d unreachable: %s", i, ep.ReachDetail)
		}
	}
	if peak := c.max(); peak < 2 {
		t.Errorf("peak in-flight requests = %d; probes did not run concurrently", peak)
	} else if peak > limit {
		t.Errorf("peak in-flight requests = %d, want at most Concurrency=%d", peak, limit)
	}
}

// TestProbeOneHangingEndpointDoesNotStallTheSet is the guarantee the wizard
// depends on: a dead endpoint in the list must not hold up the others.
func TestProbeOneHangingEndpointDoesNotStallTheSet(t *testing.T) {
	slow := rpcServer(t, script{"eth_chainId": {delay: 5 * time.Second}}, nil)
	fast := rpcServer(t, script{}, nil)

	p := testProber()
	p.ProbeTimeout = 200 * time.Millisecond

	start := time.Now()
	eps := p.ProbeAll(context.Background(), []Target{
		{URL: slow.URL, Label: "slow"},
		{URL: fast.URL, Label: "fast"},
	}, testChainID)
	elapsed := time.Since(start)

	if eps[0].Reachable {
		t.Errorf("slow endpoint reported reachable")
	}
	if !strings.Contains(eps[0].ReachDetail, "timed out") {
		t.Errorf("slow endpoint detail = %q, want it to say it timed out", eps[0].ReachDetail)
	}
	if !eps[1].Reachable {
		t.Errorf("fast endpoint unreachable: %s", eps[1].ReachDetail)
	}
	if elapsed > 3*time.Second {
		t.Errorf("set took %v; the hanging endpoint stalled it", elapsed)
	}
}

func TestProbeCancelledContext(t *testing.T) {
	srv := rpcServer(t, script{"eth_chainId": {delay: 5 * time.Second}}, nil)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	ep := testProber().Probe(ctx, Target{URL: srv.URL}, testChainID)
	if ep.Reachable {
		t.Errorf("reachable despite cancellation")
	}
	if !strings.Contains(ep.ReachDetail, "cancelled") {
		t.Errorf("detail = %q, want it to say cancelled", ep.ReachDetail)
	}
}

// ---------------------------------------------------------------------------
// valve.city client
// ---------------------------------------------------------------------------

// valveServer serves a canned capability API. handler receives the request path
// so a test can distinguish GET / from POST /check.
func valveServer(t *testing.T, status int, body string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(srv.Close)
	return srv
}

const valveMatrixJSON = `{
  "chainId": 943,
  "capabilityKeys": ["archive","trace","debug","logs","filters","otterscan","txpool","msgboard","batch"],
  "chainlistTotal": 31,
  "chainlistProbed": 12,
  "endpoints": [
    {
      "url": "https://rpc.v4.testnet.pulsechain.com",
      "label": "valve.city",
      "source": "valve",
      "reachable": true,
      "chainOk": true,
      "reachDetail": "eth_chainId → 0x3af (matches)",
      "capabilities": {
        "archive": {"status":"supported","method":"eth_getBalance","detail":"eth_getBalance(…, \"0x1\") → returned historical state"},
        "trace":   {"status":"unsupported","method":"trace_transaction","detail":"trace_transaction → -32601 Method not found"},
        "debug":   {"status":"inconclusive","method":"debug_traceTransaction","detail":"debug_traceTransaction: no reply"},
        "batch":   {"status":"supported","method":"batch","detail":"sent a 2-item batch → got a 2-element array reply"}
      }
    },
    {
      "url": "https://pulsechain-testnet-rpc.publicnode.com/",
      "label": "publicnode.com",
      "source": "chainlist",
      "reachable": true,
      "chainOk": null,
      "reachDetail": "eth_chainId → error reply",
      "capabilities": {
        "trace": {"status":"weird-new-status","method":"trace_transaction","detail":"who knows"}
      }
    },
    {
      "url": "https://dead.example",
      "label": "dead.example",
      "source": "chainlist",
      "reachable": false,
      "chainOk": null,
      "reachDetail": "eth_chainId → no well-formed JSON-RPC reply",
      "capabilities": {}
    }
  ]
}`

func TestClientMatrix(t *testing.T) {
	srv := valveServer(t, http.StatusOK, valveMatrixJSON)
	c := &Client{HTTPClient: srv.Client(), BaseURL: srv.URL}

	m, err := c.Matrix(context.Background(), testChainID)
	if err != nil {
		t.Fatalf("Matrix: %v", err)
	}

	if m.ChainID != testChainID {
		t.Errorf("ChainID = %d, want %d", m.ChainID, testChainID)
	}
	if m.ChainlistTotal != 31 || m.ChainlistProbed != 12 {
		t.Errorf("chainlist sample = %d/%d, want 12/31", m.ChainlistProbed, m.ChainlistTotal)
	}
	if len(m.CapabilityKeys) != 9 {
		t.Errorf("CapabilityKeys = %v, want valve.city's 9", m.CapabilityKeys)
	}
	if len(m.Endpoints) != 3 {
		t.Fatalf("got %d endpoints, want 3", len(m.Endpoints))
	}

	valve := m.Endpoints[0]
	if valve.Source != SourceValve {
		t.Errorf("source = %q, want %q", valve.Source, SourceValve)
	}
	if valve.ChainOK == nil || !*valve.ChainOK {
		t.Errorf("chainOk = %v, want true", valve.ChainOK)
	}
	if valve.Origin != OriginValveCity {
		t.Errorf("origin = %q, want %q", valve.Origin, OriginValveCity)
	}
	for key, want := range map[string]Status{
		KeyArchive: StatusSupported,
		KeyTrace:   StatusUnsupported,
		KeyDebug:   StatusInconclusive,
		KeyBatch:   StatusSupported,
	} {
		got, ok := valve.Cap(key)
		if !ok {
			t.Errorf("no %q result", key)
			continue
		}
		if got.Status != want {
			t.Errorf("%q status = %q, want %q", key, got.Status, want)
		}
		if got.Origin != OriginValveCity {
			t.Errorf("%q origin = %q, want %q", key, got.Origin, OriginValveCity)
		}
		if got.Method == "" || got.Detail == "" {
			t.Errorf("%q lost its method/detail: %+v", key, got)
		}
	}

	// chainOk: null must survive as nil, not become false.
	if m.Endpoints[1].ChainOK != nil {
		t.Errorf("chainOk null decoded as %v, want nil", *m.Endpoints[1].ChainOK)
	}
	// A status string we do not understand must not become a tick or a cross.
	if got, _ := m.Endpoints[1].Cap(KeyTrace); got.Status != StatusInconclusive {
		t.Errorf("unknown status decoded as %q, want %q", got.Status, StatusInconclusive)
	}
}

func TestClientMatrixErrors(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		body       string
		wantErrHas string
	}{
		{
			name:       "api error body is repeated to the operator",
			status:     http.StatusBadRequest,
			body:       `{"error":"chainId must be one of 1, 369, 943"}`,
			wantErrHas: "chainId must be one of",
		},
		{
			name:       "bare failure names the status",
			status:     http.StatusBadGateway,
			body:       "upstream down",
			wantErrHas: "502",
		},
		{
			name:       "malformed json is an error, not an empty matrix",
			status:     http.StatusOK,
			body:       "<html>",
			wantErrHas: "decode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := valveServer(t, tt.status, tt.body)
			c := &Client{HTTPClient: srv.Client(), BaseURL: srv.URL}

			_, err := c.Matrix(context.Background(), testChainID)
			if err == nil {
				t.Fatalf("Matrix succeeded, want an error")
			}
			if !strings.Contains(err.Error(), tt.wantErrHas) {
				t.Errorf("error = %v, want it to mention %q", err, tt.wantErrHas)
			}
		})
	}
}

func TestClientCheck(t *testing.T) {
	var gotPath, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"chainId":943,"capabilityKeys":["archive"],"endpoint":{
			"url":"https://mine.example","label":"mine.example","source":"user",
			"reachable":true,"chainOk":true,"reachDetail":"ok",
			"capabilities":{"archive":{"status":"supported","method":"eth_getBalance","detail":"state"}}}}`)
	}))
	t.Cleanup(srv.Close)

	c := &Client{HTTPClient: srv.Client(), BaseURL: srv.URL}
	ep, err := c.Check(context.Background(), "https://mine.example", testChainID)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}

	if gotPath != "/check" {
		t.Errorf("posted to %q, want %q", gotPath, "/check")
	}
	if !strings.Contains(gotBody, `"url":"https://mine.example"`) || !strings.Contains(gotBody, `"chainId":943`) {
		t.Errorf("request body = %s, want url and chainId", gotBody)
	}
	if ep.Source != SourceUser || ep.Origin != OriginValveCity {
		t.Errorf("source/origin = %q/%q, want %q/%q", ep.Source, ep.Origin, SourceUser, OriginValveCity)
	}
	if got, _ := ep.Cap(KeyArchive); got.Status != StatusSupported {
		t.Errorf("archive = %q, want %q", got.Status, StatusSupported)
	}
}

// TestClientCheckRejected covers the case that motivates local probing at all:
// valve.city's SSRF guard refuses private addresses, so the endpoints an
// operator most wants checked are exactly the ones it will not touch.
func TestClientCheckRejected(t *testing.T) {
	srv := valveServer(t, http.StatusBadRequest, `{"error":"endpoint rejected: private address"}`)
	c := &Client{HTTPClient: srv.Client(), BaseURL: srv.URL}

	_, err := c.Check(context.Background(), "http://10.0.0.5:8545", testChainID)
	if err == nil {
		t.Fatalf("Check succeeded, want an error")
	}
	if !strings.Contains(err.Error(), "private address") {
		t.Errorf("error = %v, want it to repeat the API's explanation", err)
	}
}

// ---------------------------------------------------------------------------
// Merge
// ---------------------------------------------------------------------------

func remoteFixture() Matrix {
	return Matrix{
		ChainID:         testChainID,
		CapabilityKeys:  ValveCityKeys(),
		ChainlistTotal:  31,
		ChainlistProbed: 12,
		Endpoints: []Endpoint{
			{
				URL: "https://public.example", Label: "public.example", Source: SourceChainlist,
				Reachable: true, ChainOK: boolPtr(true), ChainID: testChainID,
				ReachDetail: "remote says reachable", Origin: OriginValveCity,
				Capabilities: map[string]Result{
					KeyArchive: {Status: StatusUnsupported, Method: "eth_getBalance", Detail: "remote: pruned", Origin: OriginValveCity},
					KeyTrace:   {Status: StatusSupported, Method: "trace_transaction", Detail: "remote: trace", Origin: OriginValveCity},
				},
			},
			{
				URL: "https://other.example", Label: "other.example", Source: SourceChainlist,
				Reachable: true, ChainOK: boolPtr(true), ChainID: testChainID,
				ReachDetail: "remote says reachable", Origin: OriginValveCity,
				Capabilities: map[string]Result{
					KeyTrace: {Status: StatusSupported, Method: "trace_transaction", Detail: "remote: trace", Origin: OriginValveCity},
				},
			},
		},
	}
}

func TestMerge(t *testing.T) {
	t.Run("local probe wins for an endpoint we reached", func(t *testing.T) {
		local := []Endpoint{{
			URL: "https://public.example", Label: "public.example", Source: SourceChainlist,
			Reachable: true, ChainOK: boolPtr(true), ChainID: testChainID,
			ReachDetail: "local says reachable", Origin: OriginLocal,
			Capabilities: map[string]Result{
				KeyArchive: {Status: StatusSupported, Method: "eth_getBalance", Detail: "local: state", Origin: OriginLocal},
				KeyWS:      {Status: StatusUnsupported, Method: "ws", Detail: "local: upgrade refused", Origin: OriginLocal},
			},
		}}

		got := Merge(remoteFixture(), local)
		ep, ok := got.Endpoint("https://public.example")
		if !ok {
			t.Fatalf("endpoint missing from merge")
		}

		archive, _ := ep.Cap(KeyArchive)
		if archive.Status != StatusSupported || archive.Origin != OriginLocal {
			t.Errorf("archive = %q/%q, want %q/%q — first-hand and current beats cached", archive.Status, archive.Origin, StatusSupported, OriginLocal)
		}
		// A capability only valve.city probed keeps its own provenance.
		trace, _ := ep.Cap(KeyTrace)
		if trace.Status != StatusSupported || trace.Origin != OriginValveCity {
			t.Errorf("trace = %q/%q, want %q/%q", trace.Status, trace.Origin, StatusSupported, OriginValveCity)
		}
		if ep.Origin != OriginLocal || ep.ReachDetail != "local says reachable" {
			t.Errorf("reachability = %q/%q, want the local one", ep.Origin, ep.ReachDetail)
		}
		if ws, ok := ep.Cap(KeyWS); !ok || ws.Origin != OriginLocal {
			t.Errorf("ws = %+v, want a local verdict — valve.city has no opinion to lose to", ws)
		}
	})

	t.Run("valve.city is kept when we could not reach the endpoint", func(t *testing.T) {
		// Our silence is evidence about our network, not about the endpoint.
		local := []Endpoint{{
			URL: "https://public.example", Reachable: false, Origin: OriginLocal,
			ReachDetail: "local: timed out",
			Capabilities: map[string]Result{
				KeyArchive: {Status: StatusInconclusive, Method: "eth_getBalance", Detail: "local: no reply", Origin: OriginLocal},
			},
		}}

		got := Merge(remoteFixture(), local)
		ep, _ := got.Endpoint("https://public.example")

		archive, _ := ep.Cap(KeyArchive)
		if archive.Status != StatusUnsupported || archive.Origin != OriginValveCity {
			t.Errorf("archive = %q/%q, want valve.city's %q — our ignorance must not overwrite their answer", archive.Status, archive.Origin, StatusUnsupported)
		}
		if !ep.Reachable || ep.Origin != OriginValveCity {
			t.Errorf("reachability = %v/%q, want valve.city's", ep.Reachable, ep.Origin)
		}
	})

	t.Run("an endpoint only we can see is kept and listed first", func(t *testing.T) {
		// The whole reason local probing exists: a Tailscale address is
		// invisible to valve.city.
		local := []Endpoint{{
			URL: "http://100.64.0.7:8545", Label: "managed node", Source: SourceLocal,
			Reachable: true, ChainOK: boolPtr(true), ChainID: testChainID, Origin: OriginLocal,
			Capabilities: map[string]Result{
				KeyArchive: {Status: StatusSupported, Method: "eth_getBalance", Detail: "local", Origin: OriginLocal},
				KeyWS:      {Status: StatusSupported, Method: "ws", Detail: "local", Origin: OriginLocal},
			},
		}}

		got := Merge(remoteFixture(), local)
		if len(got.Endpoints) != 3 {
			t.Fatalf("got %d endpoints, want 3", len(got.Endpoints))
		}
		if got.Endpoints[0].URL != "http://100.64.0.7:8545" {
			t.Errorf("first endpoint = %q, want the local-only one", got.Endpoints[0].URL)
		}
		if got.Endpoints[0].Source != SourceLocal {
			t.Errorf("source = %q, want %q", got.Endpoints[0].Source, SourceLocal)
		}
	})

	t.Run("the ws column is appended after valve.city's own", func(t *testing.T) {
		local := []Endpoint{{
			URL: "https://public.example", Reachable: true, Origin: OriginLocal,
			Capabilities: map[string]Result{KeyWS: {Status: StatusSupported, Method: "ws", Origin: OriginLocal}},
		}}

		got := Merge(remoteFixture(), local)
		want := append(ValveCityKeys(), KeyWS)
		if len(got.CapabilityKeys) != len(want) {
			t.Fatalf("CapabilityKeys = %v, want %v", got.CapabilityKeys, want)
		}
		for i := range want {
			if got.CapabilityKeys[i] != want[i] {
				t.Fatalf("CapabilityKeys = %v, want %v — the published column order must not be reshuffled", got.CapabilityKeys, want)
			}
		}
	})

	t.Run("urls match across trailing slashes and default ports", func(t *testing.T) {
		local := []Endpoint{{
			URL: "https://public.example:443/", Reachable: true, Origin: OriginLocal,
			Capabilities: map[string]Result{
				KeyArchive: {Status: StatusSupported, Method: "eth_getBalance", Origin: OriginLocal},
			},
		}}

		got := Merge(remoteFixture(), local)
		if len(got.Endpoints) != 2 {
			t.Fatalf("got %d endpoints, want 2 — the same endpoint was listed twice", len(got.Endpoints))
		}
		ep, _ := got.Endpoint("https://public.example")
		if archive, _ := ep.Cap(KeyArchive); archive.Origin != OriginLocal {
			t.Errorf("archive origin = %q, want %q", archive.Origin, OriginLocal)
		}
	})

	t.Run("merging does not mutate the caller's matrix", func(t *testing.T) {
		remote := remoteFixture()
		local := []Endpoint{{
			URL: "https://public.example", Reachable: true, Origin: OriginLocal,
			Capabilities: map[string]Result{
				KeyArchive: {Status: StatusSupported, Method: "eth_getBalance", Origin: OriginLocal},
			},
		}}

		_ = Merge(remote, local)
		if got := remote.Endpoints[0].Capabilities[KeyArchive]; got.Status != StatusUnsupported || got.Origin != OriginValveCity {
			t.Errorf("caller's matrix was mutated: archive = %q/%q", got.Status, got.Origin)
		}
	})

	t.Run("an empty remote yields the local endpoints alone", func(t *testing.T) {
		local := []Endpoint{{
			URL: "http://127.0.0.1:8600", Reachable: true, Origin: OriginLocal,
			Capabilities: map[string]Result{
				KeyArchive: {Status: StatusSupported, Method: "eth_getBalance", Origin: OriginLocal},
				KeyWS:      {Status: StatusSupported, Method: "ws", Origin: OriginLocal},
			},
		}}

		got := Merge(Matrix{}, local)
		if len(got.Endpoints) != 1 {
			t.Fatalf("got %d endpoints, want 1", len(got.Endpoints))
		}
		want := []string{KeyArchive, KeyWS}
		if len(got.CapabilityKeys) != len(want) {
			t.Fatalf("CapabilityKeys = %v, want %v (columns derived from what was probed)", got.CapabilityKeys, want)
		}
		for i := range want {
			if got.CapabilityKeys[i] != want[i] {
				t.Fatalf("CapabilityKeys = %v, want %v", got.CapabilityKeys, want)
			}
		}
	})
}

// ---------------------------------------------------------------------------
// Gather — both sources together
// ---------------------------------------------------------------------------

func TestGatherMergesBothSources(t *testing.T) {
	node := rpcServer(t, script{
		"eth_getBalance": okResult(`"0x0"`),
	}, nil)

	body := fmt.Sprintf(`{
	  "chainId": 943,
	  "capabilityKeys": ["archive","trace"],
	  "endpoints": [
	    {"url": %q, "label": "seen by valve.city", "source": "chainlist",
	     "reachable": true, "chainOk": true, "reachDetail": "remote",
	     "capabilities": {
	       "archive": {"status":"unsupported","method":"eth_getBalance","detail":"remote: pruned"},
	       "trace":   {"status":"supported","method":"trace_transaction","detail":"remote: trace"}}}
	  ]
	}`, node.URL)
	valve := valveServer(t, http.StatusOK, body)

	g := &Gatherer{
		Client: &Client{HTTPClient: valve.Client(), BaseURL: valve.URL},
		Prober: testProber(),
	}

	m := g.Gather(context.Background(), testChainID, []Target{{URL: node.URL, Label: "our node"}})

	if m.RemoteErr != nil {
		t.Fatalf("RemoteErr = %v, want nil", m.RemoteErr)
	}
	if len(m.Endpoints) != 1 {
		t.Fatalf("got %d endpoints, want 1 (the same endpoint from both sources)", len(m.Endpoints))
	}
	ep := m.Endpoints[0]

	// Local reached it, so local wins where both spoke...
	archive, _ := ep.Cap(KeyArchive)
	if archive.Status != StatusSupported || archive.Origin != OriginLocal {
		t.Errorf("archive = %q/%q, want %q/%q", archive.Status, archive.Origin, StatusSupported, OriginLocal)
	}
	// ...and locally-probed columns valve.city never had are present too.
	if _, ok := ep.Cap(KeyWS); !ok {
		t.Errorf("ws column missing")
	}
	if _, ok := ep.Cap(KeyBatch); !ok {
		t.Errorf("batch column missing")
	}
}

// TestGatherValveCityUnreachable is the degraded case: one source going quiet
// is recorded, not fatal.
func TestGatherValveCityUnreachable(t *testing.T) {
	node := rpcServer(t, script{}, nil)
	valve := valveServer(t, http.StatusInternalServerError, "down for maintenance")

	g := &Gatherer{
		Client: &Client{HTTPClient: valve.Client(), BaseURL: valve.URL},
		Prober: testProber(),
	}

	m := g.Gather(context.Background(), testChainID, []Target{{URL: node.URL, Label: "our node"}})

	if m.RemoteErr == nil {
		t.Fatalf("RemoteErr = nil, want the valve.city failure recorded")
	}
	if !strings.Contains(m.RemoteErr.Error(), "valve.city unavailable") {
		t.Errorf("RemoteErr = %v, want it to name valve.city", m.RemoteErr)
	}
	if m.ChainID != testChainID {
		t.Errorf("ChainID = %d, want %d", m.ChainID, testChainID)
	}
	if len(m.Endpoints) != 1 {
		t.Fatalf("got %d endpoints, want the locally probed one", len(m.Endpoints))
	}
	if !m.Endpoints[0].Reachable {
		t.Errorf("local probe did not carry the run: %s", m.Endpoints[0].ReachDetail)
	}
	for _, key := range Keys() {
		res, ok := m.Endpoints[0].Cap(key)
		if !ok {
			t.Errorf("column %q missing", key)
			continue
		}
		if res.Origin != OriginLocal {
			t.Errorf("column %q origin = %q, want %q", key, res.Origin, OriginLocal)
		}
	}
}

// TestGatherWithoutAClientProbesLocallyOnly covers the private-only case: an
// operator provisioning a devnet has no use for the published matrix.
func TestGatherWithoutAClientProbesLocallyOnly(t *testing.T) {
	node := rpcServer(t, script{}, nil)
	g := &Gatherer{Prober: testProber()}

	m := g.Gather(context.Background(), testChainID, []Target{{URL: node.URL}})
	if m.RemoteErr != nil {
		t.Errorf("RemoteErr = %v, want nil when no client is configured", m.RemoteErr)
	}
	if len(m.Endpoints) != 1 || !m.Endpoints[0].Reachable {
		t.Fatalf("endpoints = %+v, want one reachable local probe", m.Endpoints)
	}
	if host, _, _ := net.SplitHostPort(strings.TrimPrefix(node.URL, "http://")); !strings.Contains(m.Endpoints[0].Label, host) {
		t.Errorf("label = %q, want it derived from the host", m.Endpoints[0].Label)
	}
}

// ---------------------------------------------------------------------------
// Key set
// ---------------------------------------------------------------------------

func TestKeys(t *testing.T) {
	// valve.city's set must stay byte-identical to CAPABILITY_KEYS in the
	// monorepo router, or the two sources stop being comparable.
	want := []string{"archive", "trace", "debug", "logs", "filters", "otterscan", "txpool", "msgboard", "batch"}
	got := ValveCityKeys()
	if len(got) != len(want) {
		t.Fatalf("ValveCityKeys() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ValveCityKeys() = %v, want %v", got, want)
		}
	}

	all := Keys()
	if len(all) != len(want)+1 || all[len(all)-1] != KeyWS {
		t.Errorf("Keys() = %v, want valve.city's set plus %q", all, KeyWS)
	}

	// Callers must not be able to reach in and reorder the package's own state.
	Keys()[0] = "clobbered"
	if Keys()[0] != KeyArchive {
		t.Errorf("Keys() returned a mutable view of package state")
	}

	for _, k := range all {
		if Label(k) == "" || Help(k) == "" {
			t.Errorf("%q has no label or help text; the UI cannot explain the column", k)
		}
	}
}
