package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/config"
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
	if err := json.NewDecoder(res.Body).Decode(&v); err != nil || len(v.History) != 1 || v.ExpectedLabel != "estimate" || fp.diskCalls != 1 {
		t.Fatalf("view %+v err %v calls %d", v, err, fp.diskCalls)
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
	if len(v.History) != 2 || fp.diskCalls != 1 {
		t.Fatalf("history %d calls %d", len(v.History), fp.diskCalls)
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
	if res.StatusCode < 400 || e.Message == "" {
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
