package capabilities

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// ProbeRepeat
//
// These tests never touch the network, same discipline as the rest of the
// package: every server is httptest.NewServer, in-process.
// ---------------------------------------------------------------------------

// seqScript maps a JSON-RPC method to a SEQUENCE of replies, one consumed per
// call to that method. A plain script (see capabilities_test.go) only has one
// answer per method, which is enough for a single Probe but not for
// ProbeRepeat: the whole point is that the same method gets asked n times and
// a load-balanced endpoint can answer differently each time. The last entry
// repeats once the sequence is exhausted, so a test only has to script as many
// answers as it cares about.
type seqScript map[string][]reply

func seqServer(t *testing.T, s seqScript) *httptest.Server {
	t.Helper()
	var mu sync.Mutex
	next := make(map[string]int)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

		seq, ok := s[key]
		var rep reply
		switch {
		case !ok || len(seq) == 0:
			rep = defaultReply(key)
		default:
			mu.Lock()
			i := next[key]
			if i >= len(seq) {
				i = len(seq) - 1
			}
			next[key] = i + 1
			mu.Unlock()
			rep = seq[i]
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

// requestCounter counts requests to a JSON-RPC method, so the n<=1
// short-circuit test can assert Probe genuinely ran once rather than
// inferring it from timing.
type requestCounter struct {
	mu     sync.Mutex
	counts map[string]int
}

func newRequestCounter() *requestCounter { return &requestCounter{counts: map[string]int{}} }

func (c *requestCounter) count(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.counts[key]++
}

func (c *requestCounter) get(key string) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.counts[key]
}

func countingServer(t *testing.T, s script, c *requestCounter) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		c.count(key)

		rep, ok := s[key]
		if !ok {
			rep = defaultReply(key)
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

func TestProbeRepeatAllAgreeIsUnchanged(t *testing.T) {
	srv := seqServer(t, seqScript{
		"msgboard_status": {okResult(`{"ok":true}`), okResult(`{"ok":true}`), okResult(`{"ok":true}`)},
	})
	p := testProber()

	ep := p.ProbeRepeat(context.Background(), Target{URL: srv.URL}, testChainID, 3)

	got, ok := ep.Cap(KeyMsgboard)
	if !ok {
		t.Fatalf("no msgboard result")
	}
	if got.Status != StatusSupported {
		t.Errorf("status = %q, want %q (all 3 runs agreed)", got.Status, StatusSupported)
	}
	// Agreement passes the verdict through unchanged — no fold commentary
	// should be grafted onto a Detail that already says everything needed.
	want := "msgboard_status → returned a result"
	if got.Detail != want {
		t.Errorf("detail = %q, want %q unchanged", got.Detail, want)
	}
}

func TestProbeRepeatSplitVerdictIsInconsistentWithCounts(t *testing.T) {
	// MEASURED 2026-07-25: msgboard_status on rpc.pulsechain.com returned 5
	// supported and 10 x -32601 across 15 probes. This test scripts a small
	// version of exactly that shape: a load balancer whose members disagree.
	srv := seqServer(t, seqScript{
		"msgboard_status": {
			okResult(`{"ok":true}`),
			rpcErrReply(-32601, "Method not found"),
			okResult(`{"ok":true}`),
		},
	})
	p := testProber()

	ep := p.ProbeRepeat(context.Background(), Target{URL: srv.URL}, testChainID, 3)

	got, ok := ep.Cap(KeyMsgboard)
	if !ok {
		t.Fatalf("no msgboard result")
	}
	if got.Status != StatusInconsistent {
		t.Fatalf("status = %q, want %q — 2 runs said yes, 1 said no", got.Status, StatusInconsistent)
	}
	for _, want := range []string{"msgboard_status", "2 supported", "1 unsupported", "3 runs"} {
		if !strings.Contains(got.Detail, want) {
			t.Errorf("detail = %q, want it to mention %q — a bare \"inconsistent\" is not actionable", got.Detail, want)
		}
	}
}

func TestProbeRepeatAllInconclusiveStaysInconclusive(t *testing.T) {
	srv := seqServer(t, seqScript{
		"msgboard_status": {{delay: 2 * time.Second}, {delay: 2 * time.Second}, {delay: 2 * time.Second}},
	})
	p := testProber()
	p.ProbeTimeout = 100 * time.Millisecond

	ep := p.ProbeRepeat(context.Background(), Target{URL: srv.URL}, testChainID, 3)

	got, ok := ep.Cap(KeyMsgboard)
	if !ok {
		t.Fatalf("no msgboard result")
	}
	if got.Status != StatusInconclusive {
		t.Errorf("status = %q, want %q — every run timed out, none produced evidence either way", got.Status, StatusInconclusive)
	}
}

func TestProbeRepeatInconclusiveMixedWithSupportedIsSupported(t *testing.T) {
	// A timeout is absence of evidence, not a vote for "unsupported" or
	// against the runs that did get an answer. Only a genuine
	// supported/unsupported split earns StatusInconsistent.
	srv := seqServer(t, seqScript{
		"msgboard_status": {
			okResult(`{"ok":true}`),
			{delay: 2 * time.Second},
			okResult(`{"ok":true}`),
		},
	})
	p := testProber()
	p.ProbeTimeout = 100 * time.Millisecond

	ep := p.ProbeRepeat(context.Background(), Target{URL: srv.URL}, testChainID, 3)

	got, ok := ep.Cap(KeyMsgboard)
	if !ok {
		t.Fatalf("no msgboard result")
	}
	if got.Status != StatusSupported {
		t.Errorf("status = %q, want %q — 2 of 3 runs produced evidence and they agreed", got.Status, StatusSupported)
	}
}

func TestProbeRepeatNLessEqual1ShortCircuits(t *testing.T) {
	for _, n := range []int{0, 1} {
		t.Run(fmt.Sprintf("n=%d", n), func(t *testing.T) {
			c := newRequestCounter()
			srv := countingServer(t, script{}, c)
			p := testProber()

			ep := p.ProbeRepeat(context.Background(), Target{URL: srv.URL}, testChainID, n)

			if got := c.get("eth_chainId"); got != 1 {
				t.Errorf("eth_chainId requests = %d, want 1 — n<=1 must behave exactly like a single Probe", got)
			}
			if !ep.Reachable {
				t.Errorf("Reachable = false, want true")
			}
			// A single run can never disagree with itself.
			for _, key := range Keys() {
				res, ok := ep.Cap(key)
				if !ok {
					continue
				}
				if res.Status == StatusInconsistent {
					t.Errorf("%q status = %q from a single run, which is impossible", key, res.Status)
				}
			}
		})
	}
}

func TestProbeRepeatMergeKeepsLocalInconsistentOverRemote(t *testing.T) {
	// The same "local wins, even reachable-but-disagreeing" rule Merge already
	// applies to StatusSupported/StatusUnsupported must hold for
	// StatusInconsistent too: it is first-hand evidence of a real property of
	// the endpoint (a load balancer whose members disagree), and a cached
	// valve.city verdict that never noticed the split must not override it.
	local := []Endpoint{{
		URL: "https://public.example", Reachable: true, ChainOK: boolPtr(true), ChainID: testChainID,
		ReachDetail: "local says reachable", Origin: OriginLocal,
		Capabilities: map[string]Result{
			KeyMsgboard: {
				Status: StatusInconsistent, Method: "msgboard_status",
				Detail: "msgboard_status → inconsistent across 15 runs: 5 supported, 10 unsupported",
				Origin: OriginLocal,
			},
		},
	}}

	remote := remoteFixture()
	remote.Endpoints[0].Capabilities[KeyMsgboard] = Result{
		Status: StatusSupported, Method: "msgboard_status", Detail: "remote: supported", Origin: OriginValveCity,
	}

	got := Merge(remote, local)
	ep, ok := got.Endpoint("https://public.example")
	if !ok {
		t.Fatalf("endpoint missing from merge")
	}

	msgboard, ok := ep.Cap(KeyMsgboard)
	if !ok {
		t.Fatalf("no msgboard result after merge")
	}
	if msgboard.Status != StatusInconsistent || msgboard.Origin != OriginLocal {
		t.Errorf("msgboard = %q/%q, want %q/%q — a first-hand disagreement finding must not be downgraded to valve.city's stale single verdict",
			msgboard.Status, msgboard.Origin, StatusInconsistent, OriginLocal)
	}
	if !strings.Contains(msgboard.Detail, "5 supported") {
		t.Errorf("detail = %q, lost the per-run counts across the merge", msgboard.Detail)
	}
}

// cancelAfterNChainIDCalls wraps an http.RoundTripper so the caller can cancel
// its own context deterministically after the Nth eth_chainId round trip
// completes, rather than racing a real clock against real goroutines the way
// TestProbeCancelledContext does for a single Probe. ProbeRepeat's runs are
// sequential (see its doc comment), so counting eth_chainId — the first call
// each run makes — pins exactly which run is in flight when cancellation
// lands.
type cancelAfterNChainIDCalls struct {
	base   http.RoundTripper
	n      int
	cancel context.CancelFunc

	mu    sync.Mutex
	calls int
}

func (c *cancelAfterNChainIDCalls) RoundTrip(req *http.Request) (*http.Response, error) {
	raw, err := io.ReadAll(req.Body)
	if err == nil {
		req.Body = io.NopCloser(bytes.NewReader(raw))
	}
	isChainID := bytes.Contains(raw, []byte(`"eth_chainId"`)) && !bytes.HasPrefix(bytes.TrimSpace(raw), []byte("["))

	resp, err := c.base.RoundTrip(req)
	if isChainID {
		c.mu.Lock()
		c.calls++
		got := c.calls
		c.mu.Unlock()
		if got == c.n {
			c.cancel()
		}
	}
	return resp, err
}

// TestProbeRepeatCancellationFoldsWhatItHas is the honesty guarantee from the
// ProbeRepeat doc comment: a cancelled run folds what it gathered rather than
// pretending every requested run happened.
func TestProbeRepeatCancellationFoldsWhatItHas(t *testing.T) {
	srv := rpcServer(t, script{
		"trace_transaction": okResult(`[]`), // gives run 1 unambiguous evidence
	}, nil)

	ctx, cancel := context.WithCancel(context.Background())
	rt := &cancelAfterNChainIDCalls{base: http.DefaultTransport, n: 2, cancel: cancel}

	p := testProber()
	p.HTTPClient = &http.Client{Timeout: p.HTTPClient.Timeout, Transport: rt}

	ep := p.ProbeRepeat(ctx, Target{URL: srv.URL}, testChainID, 5)

	if ctx.Err() == nil {
		t.Fatalf("context was never cancelled; the test's premise did not hold")
	}
	rt.mu.Lock()
	calls := rt.calls
	rt.mu.Unlock()
	if calls != 2 {
		t.Errorf("eth_chainId calls = %d, want exactly 2 — run 3 must not have started once ctx was done", calls)
	}

	// Run 2's own eth_chainId succeeded before cancellation fired (that
	// success is what triggers cancel()), so the endpoint the fold reports is
	// still reachable — cancellation truncates the SEQUENCE of runs, it does
	// not retroactively erase the one that already landed.
	if !ep.Reachable {
		t.Errorf("Reachable = false, want true")
	}

	// Run 1 completed in full before cancellation and is the only run whose
	// capability probes ever reached the server (run 2's were built on an
	// already-cancelled context and never got here — see the cancelled-run
	// case in TestProbeCancelledContext for the same mechanics on a single
	// Probe). That evidence must survive folding a truncated run 2 rather
	// than being diluted into inconclusive or lost.
	trace, ok := ep.Cap(KeyTrace)
	if !ok {
		t.Fatalf("no trace result; run 1's evidence was lost")
	}
	if trace.Status != StatusSupported {
		t.Errorf("trace status = %q, want %q", trace.Status, StatusSupported)
	}
}
