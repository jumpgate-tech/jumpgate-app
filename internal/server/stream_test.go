package server

// The SSE routes. They are the app's only push path — logs, setup progress and
// the monitor all arrive this way — and none of them could be exercised by the
// existing helpers, which read a whole response body and so would block forever
// on a stream that never ends.
//
// Everything here reads the stream INCREMENTALLY and drives it from the far
// end, because the interesting behaviour is the part that happens after the
// headers: whether an event that occurs while a client is attached actually
// reaches it, and whether a client going away stops the work behind it.

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/valve-tech/valve-node-app/internal/catalog"
	"github.com/valve-tech/valve-node-app/internal/config"
	"github.com/valve-tech/valve-node-app/internal/executor"
	"github.com/valve-tech/valve-node-app/internal/logwatch"
)

// ---------------------------------------------------------------------
// a target that has finished setup
// ---------------------------------------------------------------------

// completeSetup gives the target a Wire, which is what the log routes treat as
// "this machine actually runs a node". It is written to the config directly
// rather than by running the wizard: the wizard is tested elsewhere, and going
// through it here would make every log test depend on the whole plan passing.
func completeSetup(t *testing.T, targetID string) {
	t.Helper()
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	found := false
	for i := range cfg.Targets {
		if cfg.Targets[i].ID == targetID {
			cfg.Targets[i].Wire = &catalog.WireConfig{
				ChainID: 369, ExecID: "reth", BeaconID: "lighthouse-pulse", DataDir: "/mnt/reth",
			}
			found = true
		}
	}
	if !found {
		t.Fatalf("no target %q to complete setup on", targetID)
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("save config: %v", err)
	}
}

// ---------------------------------------------------------------------
// a journal that the test drives
// ---------------------------------------------------------------------

// journalExecutor answers `journalctl -f` by blocking, exactly as the real
// follow does, and streams whatever the test emits into the tail for that
// unit. Driving the journal from the test is what makes the stream assertions
// deterministic: the alternative is emitting up front and racing the
// subscriber, which is a test that passes on a fast machine.
type journalExecutor struct {
	*scriptedExecutor

	mu      sync.Mutex
	streams map[string]func(string) // unit -> the tail's line callback
	ready   chan struct{}
	once    sync.Once
}

func newJournalExecutor() *journalExecutor {
	return &journalExecutor{
		scriptedExecutor: fleetExecutor("true|0|img|sha256:abc\n"),
		streams:          map[string]func(string){},
		ready:            make(chan struct{}),
	}
}

func (j *journalExecutor) Run(ctx context.Context, cmd string, opts *executor.RunOpts) (executor.Result, error) {
	if strings.Contains(cmd, "journalctl") && opts != nil && opts.Stream != nil {
		unit := unitOf(cmd)
		j.mu.Lock()
		j.streams[unit] = opts.Stream
		j.mu.Unlock()
		j.once.Do(func() { close(j.ready) })

		// Block like a real follow. Returning would send the watcher into its
		// backoff-and-retry loop, which is not what any of these tests are about.
		<-ctx.Done()
		return executor.Result{}, ctx.Err()
	}
	return j.scriptedExecutor.Run(ctx, cmd, opts)
}

// unitOf pulls the unit out of `journalctl -u 'name' -f ...`.
func unitOf(cmd string) string {
	parts := strings.SplitN(cmd, "-u ", 2)
	if len(parts) != 2 {
		return ""
	}
	return strings.Trim(strings.Fields(parts[1])[0], "'")
}

// emit pushes one journal line into the exec unit's tail, waiting for the tail
// to exist first.
func (j *journalExecutor) emit(t *testing.T, line string) {
	t.Helper()
	select {
	case <-j.ready:
	case <-time.After(5 * time.Second):
		t.Fatal("no journal tail was ever started, so nothing could be emitted into it")
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		j.mu.Lock()
		fn := j.streams[logUnits[0]]
		j.mu.Unlock()
		if fn != nil {
			fn(line)
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("the tail for %s never started", logUnits[0])
		}
		time.Sleep(time.Millisecond)
	}
}

// errorLine is a journal line logwatch classifies as a hit. Using a real
// error-level line rather than any string is deliberate: an unclassified line
// is dropped, so a test built on one would assert on an empty stream forever.
const errorLine = "ERROR Failed to open database: permission denied"

// ---------------------------------------------------------------------
// reading a stream incrementally
// ---------------------------------------------------------------------

// openStream starts a GET that stays open, returning the response and a
// scanner over its body. The caller must close the response.
//
// The header phase is bounded separately from the stream itself. Go buffers a
// response until something flushes it, so a handler that sets SSE headers and
// then waits for its first event leaves the client blocked in Do() with no way
// to tell "connected, nothing has happened yet" from "still connecting" — and
// a test that simply called Do() would hang rather than report it.
func openStream(t *testing.T, a *apiTestServer, path string) (*http.Response, *bufio.Scanner, context.CancelFunc) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	req, err := http.NewRequestWithContext(ctx, "GET", a.ts.URL+path, nil)
	if err != nil {
		cancel()
		t.Fatalf("build request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+a.token)

	type result struct {
		res *http.Response
		err error
	}
	ch := make(chan result, 1)
	go func() {
		res, err := http.DefaultClient.Do(req)
		ch <- result{res, err}
	}()

	select {
	case r := <-ch:
		if r.err != nil {
			cancel()
			t.Fatalf("GET %s: %v", path, r.err)
		}
		return r.res, bufio.NewScanner(r.res.Body), cancel
	case <-time.After(10 * time.Second):
		cancel()
		t.Fatalf("GET %s: the response headers never arrived — an event stream must flush its headers when it opens, "+
			"not when its first event happens, or a client cannot tell a live-but-quiet stream from one still connecting", path)
	}
	return nil, nil, cancel
}

// nextEvent reads until the next `data:` line and unmarshals it.
func nextEvent[T any](t *testing.T, sc *bufio.Scanner) T {
	t.Helper()
	type result struct {
		v   T
		err error
	}
	ch := make(chan result, 1)
	go func() {
		for sc.Scan() {
			line := sc.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			var v T
			err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &v)
			ch <- result{v, err}
			return
		}
		var zero T
		ch <- result{zero, io.EOF}
	}()

	select {
	case r := <-ch:
		if r.err != nil {
			t.Fatalf("reading the next SSE event: %v", r.err)
		}
		return r.v
	case <-time.After(10 * time.Second):
		t.Fatal("no SSE event arrived")
	}
	var zero T
	return zero
}

// ---------------------------------------------------------------------
// GET /api/targets/{id}/logs
// ---------------------------------------------------------------------

// The recent buffer is what the logs screen opens on, and it must carry the
// classification — severity and the canned explanation — not just the raw
// line, because the whole point of logwatch is that an operator should not
// have to recognize a reth error string.
func TestLogs_RecentCarriesTheClassificationNotJustTheLine(t *testing.T) {
	j := newJournalExecutor()
	a := newAPITestServerWithExecutor(t, func(config.Target) (executor.Executor, error) { return j, nil })
	addTarget(t, a)
	completeSetup(t, "local")

	// Touch the route once to start the watcher, then emit.
	res := a.do(t, "GET", "/api/targets/local/logs", nil)
	res.Body.Close()
	j.emit(t, errorLine)

	var hits []logwatch.Hit
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		hits = decode[[]logwatch.Hit](t, a.do(t, "GET", "/api/targets/local/logs", nil))
		if len(hits) > 0 {
			break
		}
		time.Sleep(2 * time.Millisecond)
	}
	if len(hits) == 0 {
		t.Fatal("an error line was emitted and the recent buffer stayed empty")
	}
	got := hits[len(hits)-1]
	if got.Line != errorLine {
		t.Errorf("line: got %q, want %q", got.Line, errorLine)
	}
	if got.Severity != "error" && got.Severity != "critical" {
		t.Errorf("severity: got %q, want the line classified as a problem", got.Severity)
	}
	if got.Unit != logUnits[0] {
		t.Errorf("unit: got %q, want %q", got.Unit, logUnits[0])
	}
}

// A machine that has not finished setup has no units to read, and saying so is
// different from saying the machine does not exist — one is "finish setup",
// the other is "you are looking at the wrong thing".
func TestLogs_DistinguishesUnknownMachineFromUnfinishedSetup(t *testing.T) {
	j := newJournalExecutor()
	a := newAPITestServerWithExecutor(t, func(config.Target) (executor.Executor, error) { return j, nil })
	addTarget(t, a)

	for path, want := range map[string]int{
		"/api/targets/nope/logs":         http.StatusNotFound,
		"/api/targets/local/logs":        http.StatusConflict,
		"/api/targets/nope/logs/stream":  http.StatusNotFound,
		"/api/targets/local/logs/stream": http.StatusConflict,
	} {
		res := a.do(t, "GET", path, nil)
		res.Body.Close()
		if res.StatusCode != want {
			t.Errorf("GET %s: got %d, want %d", path, res.StatusCode, want)
		}
	}
}

// ?n= is operator input on a route that reads a ring buffer. A garbage or
// negative value must fall back to the default rather than returning nothing,
// which would read as "there are no logs".
//
// Note for anyone tightening this: the handler's `parsed > 0` check is defence
// in depth, not the only guard — logwatch.Recent clamps n <= 0 to the whole
// ring itself. Removing either one alone changes nothing observable here, so a
// mutation of just one will not fail this test, and that is correct rather
// than a gap in it.
func TestLogs_NonsenseCountFallsBackToTheDefault(t *testing.T) {
	j := newJournalExecutor()
	a := newAPITestServerWithExecutor(t, func(config.Target) (executor.Executor, error) { return j, nil })
	addTarget(t, a)
	completeSetup(t, "local")

	res := a.do(t, "GET", "/api/targets/local/logs", nil)
	res.Body.Close()
	j.emit(t, errorLine)
	waitForRecent(t, a)

	for _, n := range []string{"", "0", "-5", "abc", "1e9"} {
		hits := decode[[]logwatch.Hit](t, a.do(t, "GET", "/api/targets/local/logs?n="+n, nil))
		if len(hits) == 0 {
			t.Errorf("n=%q returned nothing; a bad count must not read as no logs", n)
		}
	}
	// A sane count is honoured.
	if hits := decode[[]logwatch.Hit](t, a.do(t, "GET", "/api/targets/local/logs?n=1", nil)); len(hits) != 1 {
		t.Errorf("n=1: got %d hits, want 1", len(hits))
	}
}

func waitForRecent(t *testing.T, a *apiTestServer) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if len(decode[[]logwatch.Hit](t, a.do(t, "GET", "/api/targets/local/logs", nil))) > 0 {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatal("no hit ever reached the recent buffer")
}

// An empty buffer must serialize as [] and not null: the UI iterates it, and
// null is the difference between "no errors yet" and a client-side crash.
func TestLogs_EmptyBufferIsAnArrayNotNull(t *testing.T) {
	j := newJournalExecutor()
	a := newAPITestServerWithExecutor(t, func(config.Target) (executor.Executor, error) { return j, nil })
	addTarget(t, a)
	completeSetup(t, "local")

	res := a.do(t, "GET", "/api/targets/local/logs", nil)
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if strings.TrimSpace(string(body)) == "null" {
		t.Fatal("an empty log buffer serialized as null")
	}
}

// ---------------------------------------------------------------------
// GET /api/targets/{id}/logs/stream
// ---------------------------------------------------------------------

// The point of the stream: a line that happens WHILE a client is attached
// reaches it, without the client polling. This is the one assertion that
// distinguishes a working stream from a route that sets the right headers and
// then never writes anything.
func TestLogsStream_DeliversALineThatHappensWhileAttached(t *testing.T) {
	j := newJournalExecutor()
	a := newAPITestServerWithExecutor(t, func(config.Target) (executor.Executor, error) { return j, nil })
	addTarget(t, a)
	completeSetup(t, "local")

	res, sc, cancel := openStream(t, a, "/api/targets/local/logs/stream")
	defer cancel()
	defer res.Body.Close()

	if got := res.Header.Get("Content-Type"); got != "text/event-stream" {
		t.Fatalf("content type: got %q, want text/event-stream", got)
	}
	if got := res.Header.Get("Cache-Control"); got != "no-cache" {
		t.Errorf("cache control: got %q — a cached event stream is a stream that stops", got)
	}

	// Emitted AFTER the subscription exists, which is the whole test.
	j.emit(t, errorLine)

	hit := nextEvent[logwatch.Hit](t, sc)
	if hit.Line != errorLine {
		t.Errorf("streamed line: got %q, want %q", hit.Line, errorLine)
	}
	if hit.Severity == "" {
		t.Error("the streamed hit carries no severity, so the UI cannot colour it")
	}
}

// Several lines arrive in order. A stream that coalesced or reordered them
// would misrepresent a startup sequence, which is exactly what an operator
// reads these in order to understand.
func TestLogsStream_DeliversLinesInOrder(t *testing.T) {
	j := newJournalExecutor()
	a := newAPITestServerWithExecutor(t, func(config.Target) (executor.Executor, error) { return j, nil })
	addTarget(t, a)
	completeSetup(t, "local")

	res, sc, cancel := openStream(t, a, "/api/targets/local/logs/stream")
	defer cancel()
	defer res.Body.Close()

	want := []string{
		"ERROR first failure",
		"ERROR second failure",
		"ERROR third failure",
	}
	for _, l := range want {
		j.emit(t, l)
	}
	for i, w := range want {
		if got := nextEvent[logwatch.Hit](t, sc).Line; got != w {
			t.Fatalf("event %d: got %q, want %q", i, got, w)
		}
	}
}

// ---------------------------------------------------------------------
// GET /api/targets/{id}/setup/stream
// ---------------------------------------------------------------------

// The setup stream is how the wizard shows progress. Asking for one when no
// run is in flight must not hang the caller waiting for events that will never
// come.
func TestSetupStream_AnswersWhenThereIsNoRun(t *testing.T) {
	j := newJournalExecutor()
	a := newAPITestServerWithExecutor(t, func(config.Target) (executor.Executor, error) { return j, nil })
	addTarget(t, a)

	done := make(chan int, 1)
	go func() {
		res := a.do(t, "GET", "/api/targets/local/setup/stream", nil)
		res.Body.Close()
		done <- res.StatusCode
	}()

	select {
	case code := <-done:
		if code == http.StatusOK {
			t.Errorf("got 200 for a machine with no setup run — the caller is left holding an empty stream")
		}
	case <-time.After(10 * time.Second):
		t.Fatal("the request hung: a stream with no run behind it must answer, not block")
	}
}

// ---------------------------------------------------------------------
// a client going away
// ---------------------------------------------------------------------

// A browser tab closing is the normal way these streams end. The handler must
// notice and return rather than leaving a goroutine writing into a dead
// connection for the life of the process.
func TestLogsStream_HandlerReturnsWhenTheClientDisconnects(t *testing.T) {
	j := newJournalExecutor()
	a := newAPITestServerWithExecutor(t, func(config.Target) (executor.Executor, error) { return j, nil })
	addTarget(t, a)
	completeSetup(t, "local")

	res, sc, cancel := openStream(t, a, "/api/targets/local/logs/stream")
	j.emit(t, errorLine)
	if got := nextEvent[logwatch.Hit](t, sc).Line; got != errorLine {
		t.Fatalf("streamed line: got %q", got)
	}

	// The client goes away.
	cancel()
	res.Body.Close()

	// The watcher is still healthy afterwards: a second client attaches and
	// receives. If the first disconnect had wedged the publisher, this would
	// time out — which is the failure this test exists to catch.
	res2, sc2, cancel2 := openStream(t, a, "/api/targets/local/logs/stream")
	defer cancel2()
	defer res2.Body.Close()

	j.emit(t, "ERROR after the first client left")
	if got := nextEvent[logwatch.Hit](t, sc2).Line; got != "ERROR after the first client left" {
		t.Errorf("second client got %q", got)
	}
}

// ---------------------------------------------------------------------
// GET /api/targets/{id}/monitor/stream
// ---------------------------------------------------------------------

// The monitor stream sends the CURRENT reading immediately on connect rather
// than waiting for the next poll tick. Without that, a dashboard opened
// between ticks shows empty gauges for the whole interval, which reads as a
// node that is not reporting.
func TestMonitorStream_SendsTheLatestReadingOnConnect(t *testing.T) {
	j := newJournalExecutor()
	a := newAPITestServerWithExecutor(t, func(config.Target) (executor.Executor, error) { return j, nil })
	addTarget(t, a)
	completeSetup(t, "local")

	res, sc, cancel := openStream(t, a, "/api/targets/local/monitor/stream")
	defer cancel()
	defer res.Body.Close()

	if got := res.Header.Get("Content-Type"); got != "text/event-stream" {
		t.Fatalf("content type: got %q", got)
	}
	// No poll has been forced and nothing was emitted — an event arriving at
	// all is the assertion.
	got := nextEvent[map[string]any](t, sc)
	if got == nil {
		t.Fatal("the stream opened and sent nothing, so a dashboard would show empty gauges until the first tick")
	}
}

func TestMonitorStream_DistinguishesUnknownMachineFromUnfinishedSetup(t *testing.T) {
	j := newJournalExecutor()
	a := newAPITestServerWithExecutor(t, func(config.Target) (executor.Executor, error) { return j, nil })
	addTarget(t, a)

	for path, want := range map[string]int{
		"/api/targets/nope/monitor/stream":  http.StatusNotFound,
		"/api/targets/local/monitor/stream": http.StatusConflict,
	} {
		res := a.do(t, "GET", path, nil)
		res.Body.Close()
		if res.StatusCode != want {
			t.Errorf("GET %s: got %d, want %d", path, res.StatusCode, want)
		}
	}
}
