package server

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/valve-tech/jumpgate/internal/catalog"
	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/executor"
	"github.com/valve-tech/jumpgate/internal/logwatch"
)

// openSSE opens path as a stream and returns a reader over its body and a
// stop that ends it. (Ruling T3: stream_test.go already has openStream.)
func openSSE(t *testing.T, a *apiTestServer, path string) (*bufio.Reader, func()) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, a.ts.URL+path, nil)
	req.Header.Set("Authorization", "Bearer "+a.token)
	res, err := http.DefaultClient.Do(req)
	if err != nil || res.StatusCode != http.StatusOK {
		cancel()
		t.Fatalf("open %s: %v %v", path, err, res)
	}
	return bufio.NewReader(res.Body), func() { cancel(); res.Body.Close() }
}

func seedWired(t *testing.T, id string) {
	t.Helper()
	if _, err := config.Update(func(c *config.Config) error {
		c.Targets = append(c.Targets, config.Target{ID: id, Mode: "local",
			Wire: &catalog.WireConfig{ChainID: 369, ExecID: "reth", BeaconID: "lighthouse-pulse", DataDir: "/mnt/reth"}})
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

// readLines reads r line by line until match reports true, the stream ends,
// or the deadline passes. The read runs in its own goroutine so a stream that
// sends nothing at all cannot hang the test; stop ends that goroutine.
func readLines(r *bufio.Reader, d time.Duration, match func(string) bool) bool {
	found := make(chan bool, 1)
	go func() {
		for {
			line, err := r.ReadString('\n')
			if err != nil {
				found <- false
				return
			}
			if match(line) {
				found <- true
				return
			}
		}
	}()
	select {
	case ok := <-found:
		return ok
	case <-time.After(d):
		return false
	}
}

// Streams ping while quiet, so a client can tell a quiet stream from a dead
// one (spec A4).
func TestStreamsPingWhileQuiet(t *testing.T) {
	old := ssePingInterval
	ssePingInterval = 30 * time.Millisecond
	// Registered before the server, so it runs after the server's Close has
	// waited for every handler (which read the interval) to return.
	t.Cleanup(func() { ssePingInterval = old })
	a := newAPITestServer(t)
	seedWired(t, "box")
	// A setup run that is still going, so its stream stays open; closing
	// done lets the server's cleanup stop waiting for it.
	run := newSetupRun(nil)
	t.Cleanup(func() { close(run.done) })
	entry := a.srv.reg.get("box")
	entry.mu.Lock()
	entry.setup = run
	entry.mu.Unlock()
	for _, path := range []string{"/api/targets/box/logs/stream", "/api/targets/box/monitor/stream", "/api/targets/box/setup/stream"} {
		r, stop := openSSE(t, a, path)
		found := readLines(r, 2*time.Second, func(line string) bool { return line == ": ping\n" })
		stop()
		if !found {
			t.Errorf("%s: no ping comment", path)
		}
	}
}

// readFrame reads one SSE frame's first two lines (the event line and the
// data line of a named event), bounded by streamWait.
func readFrame(t *testing.T, r *bufio.Reader) (string, string) {
	t.Helper()
	var lines []string
	if !readLines(r, streamWait, func(line string) bool { lines = append(lines, line); return len(lines) == 2 }) {
		t.Fatalf("the stream sent %q and then nothing", lines)
	}
	return lines[0], lines[1]
}

func backlogOf(t *testing.T, data string) []map[string]any {
	t.Helper()
	var hits []map[string]any
	if err := json.Unmarshal([]byte(strings.TrimPrefix(strings.TrimSpace(data), "data: ")), &hits); err != nil || hits == nil {
		t.Fatalf("backlog %q is not a JSON array: %v", data, err)
	}
	return hits
}

func TestLogsStreamSendsTheBacklogFirst(t *testing.T) {
	a := newAPITestServer(t)
	seedWired(t, "box")
	r, stop := openSSE(t, a, "/api/targets/box/logs/stream?backlog=5")
	defer stop()
	first, data := readFrame(t, r)
	if first != "event: reset\n" || !strings.HasPrefix(data, "data: [") {
		t.Fatalf("first frame %q %q", first, data)
	}
	backlogOf(t, data)
}

// The backlog carries the recent lines, newest last and at most N of them;
// a zero or nonsense N is an empty reset, not the whole ring. (The cap is
// TestBacklogParamClamps: the ring holds fewer lines than maxLogBacklog.)
func TestLogsStreamBacklogCarriesTheRecentLines(t *testing.T) {
	j := newJournalExecutor()
	a := newAPITestServerWithExecutor(t, func(config.Target) (executor.Executor, error) { return j, nil })
	addTarget(t, a)
	completeSetup(t, "local")
	res := a.do(t, "GET", "/api/targets/local/logs", nil)
	res.Body.Close()
	j.emit(t, errorLine)
	j.emit(t, errorLine+" again")
	deadline := time.Now().Add(streamWait)
	for len(decode[[]logwatch.Hit](t, a.do(t, "GET", "/api/targets/local/logs", nil))) < 2 {
		if time.Now().After(deadline) {
			t.Fatal("the two lines never reached the recent buffer")
		}
		time.Sleep(2 * time.Millisecond)
	}

	for backlog, want := range map[string]int{"1": 1, "5": 2, "0": 0, "-3": 0, "abc": 0} {
		r, stop := openSSE(t, a, "/api/targets/local/logs/stream?backlog="+backlog)
		first, data := readFrame(t, r)
		stop()
		hits := backlogOf(t, data)
		if first != "event: reset\n" || len(hits) != want {
			t.Errorf("backlog=%s: %q with %d lines, want %d", backlog, first, len(hits), want)
			continue
		}
		if want > 0 && hits[len(hits)-1]["line"] != errorLine+" again" {
			t.Errorf("backlog=%s: the newest line is not last: %v", backlog, hits)
		}
	}
}

func TestBacklogParamClamps(t *testing.T) {
	for raw, want := range map[string]int{
		"1": 1, "2000": maxLogBacklog, "2001": maxLogBacklog, "999999": maxLogBacklog,
		"0": 0, "-3": 0, "abc": 0, "1e9": 0,
	} {
		if got := backlogParam(raw); got != want {
			t.Errorf("backlogParam(%q) = %d, want %d", raw, got, want)
		}
	}
}
