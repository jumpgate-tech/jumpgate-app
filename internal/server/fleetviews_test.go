package server

import (
	"bufio"
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

	"github.com/valve-tech/jumpgate/internal/agentclient"
	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/monitor"
)

func TestFleetDiskMeasuresWhenThereIsNoReading(t *testing.T) {
	_, fp, ts, token := fleetServer(t, config.Target{ID: "a", Mode: "ssh", Agent: paired})
	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/fleet/a/disk", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var v api.DiskView
	if err := json.NewDecoder(res.Body).Decode(&v); err != nil || len(v.History) != 1 || v.ExpectedLabel != "estimate" || fp.diskCount() != 1 {
		t.Fatalf("view %+v err %v calls %d", v, err, fp.diskCount())
	}
	if res.Header.Get("X-Jumpgate-Via") != "agent" {
		t.Fatalf("via %q", res.Header.Get("X-Jumpgate-Via"))
	}
}

func TestMeasureDiskAddsToTheHistory(t *testing.T) {
	s, fp, ts, token := fleetServer(t, config.Target{ID: "a", Mode: "ssh", Agent: paired})
	s.fleet.recordDisk("a", time.Now().Add(-time.Hour), fakeDU())
	var v api.DiskView
	postJSON(t, ts.URL+"/api/fleet/a/disk/measure", token, nil, &v)
	if len(v.History) != 2 || fp.diskCount() != 1 {
		t.Fatalf("history %d calls %d", len(v.History), fp.diskCount())
	}
}

// Measure-now has no rate limit of its own, so two at once for a box share
// one probe instead of running the command twice.
func TestConcurrentMeasuresJoinTheOneInFlight(t *testing.T) {
	s, fp, ts, token := fleetServer(t, config.Target{ID: "a", Mode: "ssh", Agent: paired})
	fp.diskGate = make(chan struct{})
	fp.diskEntered = make(chan struct{}, 4)
	var wg sync.WaitGroup
	views := make([]api.DiskView, 2)
	for i := range views {
		wg.Add(1)
		go func() {
			defer wg.Done()
			postJSON(t, ts.URL+"/api/fleet/a/disk/measure", token, nil, &views[i])
		}()
	}
	<-fp.diskEntered
	// Wait until the second request has joined: it holds a waiter.
	deadline := time.Now().Add(2 * time.Second)
	for s.fleet.measureWaiters("a") < 2 {
		if time.Now().After(deadline) {
			t.Fatal("the second measure never joined the first")
		}
		time.Sleep(5 * time.Millisecond)
	}
	close(fp.diskGate)
	wg.Wait()
	fp.mu.Lock()
	calls := fp.diskCalls
	fp.mu.Unlock()
	if calls != 1 || len(views[0].History) != 1 || len(views[1].History) != 1 {
		t.Fatalf("calls %d, histories %d and %d, want one probe and one sample", calls, len(views[0].History), len(views[1].History))
	}
}

func TestMeasureFailureIsReportedAndNotRecorded(t *testing.T) {
	s, fp, ts, token := fleetServer(t, config.Target{ID: "a", Mode: "ssh", Agent: paired})
	fp.diskFail = true
	res, e := do(t, ts, token, "POST", "/api/fleet/a/disk/measure", "")
	if res.StatusCode != http.StatusBadGateway || e.Code == "" || e.Message == "" {
		t.Fatalf("status %d %+v", res.StatusCode, e)
	}
	if len(s.fleet.history("a")) != 0 {
		t.Fatal("a failed measure was recorded")
	}
}

func TestFleetRowAndUnknownTarget(t *testing.T) {
	_, _, ts, token := fleetServer(t, config.Target{ID: "a", Mode: "ssh", Agent: paired})
	res, e := do(t, ts, token, "GET", "/api/fleet/nope", "")
	if res.StatusCode != http.StatusNotFound || e.Code != api.CodeTargetNotFound {
		t.Fatalf("unknown: %d %+v", res.StatusCode, e)
	}
	res, _ = do(t, ts, token, "GET", "/api/fleet/a", "")
	if res.StatusCode != http.StatusOK {
		t.Fatalf("known: %d", res.StatusCode)
	}
}

func TestFleetStatusStreamSendsNodeStatus(t *testing.T) {
	old := agentStatusInterval
	agentStatusInterval = 20 * time.Millisecond
	t.Cleanup(func() { agentStatusInterval = old })
	_, _, ts, token := fleetServer(t, config.Target{ID: "a", Mode: "ssh", Agent: paired})
	frames, _ := readFrames(t, ts, token, "/api/fleet/a/status/stream", 2)
	for _, f := range frames {
		if !strings.Contains(f, `"overall":"synced"`) {
			t.Fatalf("frame %q", f)
		}
	}
}

// The log follower is retired with the executor it was started beside, like
// the monitor and watcher, so it restarts fresh; the evicted lease closes.
func TestEvictionRetiresTheLogFollower(t *testing.T) {
	s, f, tg := leaseServer(t)
	tg.Agent = paired
	if _, err := config.Update(func(c *config.Config) error { c.Targets = []config.Target{tg}; return nil }); err != nil {
		t.Fatal(err)
	}
	cfg, _ := s.loadConfig()
	ex, _ := s.getExecutor(tg)
	f1 := s.logFollowerFor(cfg, tg)
	s.reg.evictExecutor("a", ex)
	e := s.reg.get("a")
	e.mu.Lock()
	still := e.logs
	e.mu.Unlock()
	if still != nil || !f1.isStopped() {
		t.Fatalf("the follower survived the eviction: on record %v, stopped %v", still != nil, f1.isStopped())
	}
	eventually(t, "the evicted executor was never closed", func() bool { return f.open() == 0 })
	f2 := s.logFollowerFor(cfg, tg)
	if f2 == f1 {
		t.Fatal("the follower was not rebuilt")
	}
	s.stopLogFollowers()
}

// A measure that fails clears the one in flight, so the next starts fresh.
func TestMeasureErrorClearsTheSingleflight(t *testing.T) {
	s, fp, ts, token := fleetServer(t, config.Target{ID: "a", Mode: "ssh", Agent: paired})
	fp.mu.Lock()
	fp.diskFail = true
	fp.mu.Unlock()
	if res, _ := do(t, ts, token, "POST", "/api/fleet/a/disk/measure", ""); res.StatusCode < 400 {
		t.Fatalf("failing measure answered %d", res.StatusCode)
	}
	if n := s.fleet.measureWaiters("a"); n != 0 {
		t.Fatalf("%d waiters left on a finished measure", n)
	}
	fp.mu.Lock()
	fp.diskFail = false
	fp.mu.Unlock()
	var v api.DiskView
	postJSON(t, ts.URL+"/api/fleet/a/disk/measure", token, nil, &v)
	if fp.diskCount() != 2 || len(v.History) != 1 {
		t.Fatalf("calls %d history %d: the second measure did not start fresh", fp.diskCount(), len(v.History))
	}
}

// A probe that never answers ends at fleetProbeTimeout as unreachable, and
// the measure after it starts fresh.
func TestMeasureTimesOut(t *testing.T) {
	old := fleetProbeTimeout
	fleetProbeTimeout = 50 * time.Millisecond
	t.Cleanup(func() { fleetProbeTimeout = old })
	_, fp, ts, token := fleetServer(t, config.Target{ID: "a", Mode: "ssh", Agent: paired})
	fp.diskGate = make(chan struct{}) // never closed: the probe only ends by its context
	res, e := do(t, ts, token, "POST", "/api/fleet/a/disk/measure", "")
	if res.StatusCode < 400 || e.Code != api.CodeUnreachable {
		t.Fatalf("timeout: %d %+v", res.StatusCode, e)
	}
}

// openStatus opens a status stream and keeps it open until cancel.
func openStatus(t *testing.T, ts *httptest.Server, token, id string) (*http.Response, context.CancelFunc) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL+"/api/fleet/"+id+"/status/stream", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	t.Cleanup(func() { cancel(); res.Body.Close() })
	return res, cancel
}

// dataFrames reads n data frames, false when the stream ends first.
func dataFrames(res *http.Response, n int) bool {
	r := bufio.NewReader(res.Body)
	for n > 0 {
		line, err := r.ReadString('\n')
		if err != nil {
			return false
		}
		if strings.HasPrefix(line, "data:") {
			n--
		}
	}
	return true
}

func waitFor(t *testing.T, what string, ok func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for !ok() {
		if time.Now().After(deadline) {
			t.Fatal(what)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func fastStatus(t *testing.T) {
	t.Helper()
	oldI, oldIdle := agentStatusInterval, statusIdleStop
	agentStatusInterval, statusIdleStop = 40*time.Millisecond, 30*time.Millisecond
	t.Cleanup(func() { agentStatusInterval, statusIdleStop = oldI, oldIdle })
}

// Five windows on one box cost the probes of one.
func TestStatusStreamsShareOneProbe(t *testing.T) {
	fastStatus(t)
	_, fp, ts, token := fleetServer(t, config.Target{ID: "a", Mode: "ssh", Agent: paired})
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		res, _ := openStatus(t, ts, token, "a")
		wg.Add(1)
		go func() {
			defer wg.Done()
			dataFrames(res, 4)
		}()
	}
	wg.Wait()
	// Four frames each took about four ticks of one source; five probers
	// would have made twenty calls.
	if n := fp.callsFor("a"); n > 8 {
		t.Fatalf("%d probes for 5 streams over ~4 ticks: the streams are not sharing", n)
	}
}

// A new subscriber gets the latest result at once, not after the next tick.
func TestStatusStreamStartsWithTheLatestFrame(t *testing.T) {
	fastStatus(t)
	agentStatusInterval = time.Hour
	s, _, ts, token := fleetServer(t, config.Target{ID: "a", Mode: "ssh", Agent: paired})
	first, _ := openStatus(t, ts, token, "a")
	if !dataFrames(first, 1) {
		t.Fatal("no first frame")
	}
	second, _ := openStatus(t, ts, token, "a")
	if !dataFrames(second, 1) {
		t.Fatal("the second window got no frame before the next tick")
	}
	if s.fleet.statusSubscribers("a") != 2 {
		t.Fatalf("subscribers %d", s.fleet.statusSubscribers("a"))
	}
}

func TestNinthStatusStreamIsRefused(t *testing.T) {
	_, _, ts, token := fleetServer(t, config.Target{ID: "a", Mode: "ssh", Agent: paired})
	for i := 0; i < maxStatusStreams; i++ {
		openStatus(t, ts, token, "a")
	}
	res, e := do(t, ts, token, "GET", "/api/fleet/a/status/stream", "")
	if res.StatusCode != http.StatusTooManyRequests || e.Code != api.CodeTooManyStreams || e.Hint == "" {
		t.Fatalf("ninth: %d %+v", res.StatusCode, e)
	}
}

// The source stops after the last window leaves, leaving no goroutine, and
// a hung probe never stops windows from disconnecting.
func TestStatusSourceStopsAfterTheLastSubscriber(t *testing.T) {
	fastStatus(t)
	s, _, ts, token := fleetServer(t, config.Target{ID: "a", Mode: "ssh", Agent: paired})
	res, cancel := openStatus(t, ts, token, "a")
	if !dataFrames(res, 1) {
		t.Fatal("no frame")
	}
	cancel()
	waitFor(t, "the handler kept its subscription after the client left", func() bool { return s.fleet.statusSubscribers("a") == 0 })
	waitFor(t, "the source outlived its last subscriber", func() bool { return s.fleet.statusSources() == 0 })
	requireNoFleetGoroutines(t)
}

func TestHungProbeDoesNotBlockDisconnect(t *testing.T) {
	fastStatus(t)
	s, fp, ts, token := fleetServer(t, config.Target{ID: "a", Mode: "ssh", Agent: paired})
	fp.hang["a"] = true
	t.Cleanup(func() { close(fp.release) }) // runs before the poller's stop waits
	_, cancel := openStatus(t, ts, token, "a")
	waitFor(t, "the probe never started", func() bool { return fp.callsFor("a") == 1 })
	cancel()
	waitFor(t, "a hung probe held the subscriber", func() bool { return s.fleet.statusSubscribers("a") == 0 })
	waitFor(t, "a hung probe held the source", func() bool { return s.fleet.statusSources() == 0 })
}

// A stream ends when its client leaves, when the fleet stops, and when its
// target is removed.
func TestStatusStreamEnds(t *testing.T) {
	fastStatus(t)
	t.Run("fleet stop", func(t *testing.T) {
		s, _, ts, token := fleetServer(t, config.Target{ID: "a", Mode: "ssh", Agent: paired})
		res, _ := openStatus(t, ts, token, "a")
		if !dataFrames(res, 1) {
			t.Fatal("no frame")
		}
		go s.fleet.stop()
		ended := make(chan struct{})
		go func() { io.Copy(io.Discard, res.Body); close(ended) }()
		select {
		case <-ended:
		case <-time.After(3 * time.Second):
			t.Fatal("the stream outlived the fleet")
		}
	})
	t.Run("target removed", func(t *testing.T) {
		_, _, ts, token := fleetServer(t, config.Target{ID: "a", Mode: "ssh", Agent: paired})
		res, _ := openStatus(t, ts, token, "a")
		if !dataFrames(res, 1) {
			t.Fatal("no frame")
		}
		if _, err := config.Update(func(c *config.Config) error { c.Targets = nil; return nil }); err != nil {
			t.Fatal(err)
		}
		ended := make(chan struct{})
		go func() { io.Copy(io.Discard, res.Body); close(ended) }()
		select {
		case <-ended:
		case <-time.After(3 * time.Second):
			t.Fatal("the stream outlived its target")
		}
	})
}

// Evicting a box's executor ends its status source; the next viewer gets a
// new one.
func TestEvictionEndsTheStatusSource(t *testing.T) {
	s, _, tg := leaseServer(t)
	t.Cleanup(s.fleet.stop) // wait for the source's goroutine before the next test changes the timing globals
	tg.Agent = paired
	if _, err := config.Update(func(c *config.Config) error { c.Targets = []config.Target{tg}; return nil }); err != nil {
		t.Fatal(err)
	}
	s.fleet.probe = newFakeProber()
	ex, _ := s.getExecutor(tg)
	ch, unsub, err := s.fleet.subscribeStatus(tg)
	if err != nil {
		t.Fatal(err)
	}
	defer unsub()
	s.reg.evictExecutor("a", ex)
	deadline := time.After(2 * time.Second)
	for open := true; open; {
		select {
		case _, open = <-ch:
		case <-deadline:
			t.Fatal("the status channel stayed open after eviction")
		}
	}
	if s.fleet.statusSources() != 0 {
		t.Fatal("the source is still registered")
	}
}

// evictingProber fails a status probe the way an SSH-only box does when its
// connection check fails: the executor is evicted (which retires the status
// source) before the probe returns its error.
type evictingProber struct {
	*fakeProber
	s *Server
}

func (e evictingProber) status(ctx context.Context, cfg config.Config, t config.Target) (monitor.Snapshot, error) {
	e.s.fleet.retireStatus(t.ID)
	return monitor.Snapshot{}, &dialError{fmt.Errorf("%w: connection lost", agentclient.ErrUnreachable)}
}

// A probe that fails and evicts the executor still publishes its failure
// frame to the subscribers before the source retires and their stream ends.
func TestEvictingProbePublishesItsFailureBeforeRetiring(t *testing.T) {
	fastStatus(t)
	s, _, ts, token := fleetServer(t, config.Target{ID: "a", Mode: "ssh", Agent: paired})
	s.fleet.probe = evictingProber{newFakeProber(), s}
	res, _ := openStatus(t, ts, token, "a")
	body, err := io.ReadAll(res.Body) // the stream must end by itself
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "event: error") || !strings.Contains(string(body), "unreachable") {
		t.Fatalf("the subscriber never saw the failure frame:\n%s", body)
	}
	waitFor(t, "the source is still registered", func() bool { return s.fleet.statusSources() == 0 })
}
