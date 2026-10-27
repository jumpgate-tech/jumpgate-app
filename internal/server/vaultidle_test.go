package server

import (
	"bytes"
	"context"
	"log"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/fstest"
	"time"

	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/testutil"
	"github.com/valve-tech/jumpgate/internal/vault/vaulttest"
)

// idleRig is an unlocked vault server with a clock the test moves and a
// fast idle tick.
type idleRig struct {
	srv   *Server
	ts    *httptest.Server
	token string
	clock *atomic.Int64 // unix nanoseconds
}

func newIdleRig(t *testing.T, minutes int) *idleRig {
	t.Helper()
	testutil.Home(t)
	old := vaultIdleTick
	vaultIdleTick = 5 * time.Millisecond
	t.Cleanup(func() { vaultIdleTick = old })
	if _, err := config.Update(func(c *config.Config) error { c.VaultIdleLockMinutes = minutes; return nil }); err != nil {
		t.Fatal(err)
	}
	clock := new(atomic.Int64)
	clock.Store(time.Date(2026, 10, 27, 12, 0, 0, 0, time.UTC).UnixNano())
	p, pass := vaulttest.Path(t)
	token := NewSessionToken()
	srv := New(Config{Token: token, UI: fstest.MapFS{}, VaultPath: p, noJobsDriver: true,
		now: func() time.Time { return time.Unix(0, clock.Load()) }})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(func() { ts.Close(); srv.fleet.stop(); srv.jobs.stopWithin(time.Second) })
	if _, err := srv.Unlock(pass); err != nil {
		t.Fatal(err)
	}
	return &idleRig{srv: srv, ts: ts, token: token, clock: clock}
}

func (r *idleRig) advance(d time.Duration) { r.clock.Add(int64(d)) }

func (r *idleRig) locked() bool { return r.srv.vault().Locked() }

// settle lets the idle loop tick a few times.
func settle() { time.Sleep(20 * vaultIdleTick) }

func (r *idleRig) waitLocked(t *testing.T) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for !r.locked() {
		if time.Now().After(deadline) {
			t.Fatal("the vault never locked")
		}
		time.Sleep(vaultIdleTick)
	}
}

// syncBuf is a log sink safe for the idle loop's goroutine.
type syncBuf struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuf) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *syncBuf) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

func captureLog(t *testing.T) *syncBuf {
	t.Helper()
	var buf syncBuf
	prev := log.Writer()
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(prev) })
	return &buf
}

func TestIdleLockIsOffByDefault(t *testing.T) {
	r := newIdleRig(t, 0)
	r.advance(24 * time.Hour)
	settle()
	if r.locked() {
		t.Fatal("locked with the setting off")
	}
}

func TestIdleLockLocksAfterNMinutes(t *testing.T) {
	logs := captureLog(t)
	r := newIdleRig(t, 5)
	r.advance(4 * time.Minute)
	settle()
	if r.locked() {
		t.Fatal("locked before 5 minutes")
	}
	r.advance(time.Minute)
	r.waitLocked(t)
	if !strings.Contains(logs.String(), "vault locked after 5 minutes idle") {
		t.Fatalf("log %q", logs.String())
	}
}

func TestIdleLockWaitsForRunningPlans(t *testing.T) {
	r := newIdleRig(t, 5)
	k := jobKey{"box", "job"}
	_, cancel := context.WithCancel(context.Background())
	defer cancel()
	r.srv.jobs.mu.Lock()
	r.srv.jobs.driving[k] = cancel
	r.srv.jobs.mu.Unlock()
	r.advance(time.Hour)
	settle()
	if r.locked() {
		t.Fatal("locked while a plan runs")
	}
	r.srv.jobs.mu.Lock()
	delete(r.srv.jobs.driving, k)
	r.srv.jobs.mu.Unlock()
	r.waitLocked(t)
}

func TestReadsDoNotCountAsActivity(t *testing.T) {
	r := newIdleRig(t, 5)
	// One write resets the clock.
	r.advance(4 * time.Minute)
	if res, _ := do(t, r.ts, r.token, "PUT", "/api/settings", `{}`); res.StatusCode != 200 {
		t.Fatalf("PUT /api/settings = %d", res.StatusCode)
	}
	r.advance(4 * time.Minute)
	settle()
	if r.locked() {
		t.Fatal("locked 4 minutes after a write")
	}
	// Reads every minute do not.
	for range 2 {
		r.advance(time.Minute)
		do(t, r.ts, r.token, "GET", "/api/vault", "")
		do(t, r.ts, r.token, "GET", "/api/settings", "")
	}
	r.waitLocked(t)
}

func TestIdleLockMinutesAreBounded(t *testing.T) {
	r := newIdleRig(t, 0)
	for _, body := range []string{`{"vaultIdleLockMinutes":1441}`, `{"vaultIdleLockMinutes":-1}`} {
		res, e := do(t, r.ts, r.token, "PUT", "/api/settings", body)
		if res.StatusCode != 400 || e.Message != "minutes must be between 0 (off) and 1440" {
			t.Fatalf("%s = %d %+v", body, res.StatusCode, e)
		}
	}
	res, _ := do(t, r.ts, r.token, "PUT", "/api/settings", `{"vaultIdleLockMinutes":30}`)
	if res.StatusCode != 200 {
		t.Fatalf("30 = %d", res.StatusCode)
	}
	c, err := config.Load()
	if err != nil || c.VaultIdleLockMinutes != 30 {
		t.Fatalf("stored %d, %v", c.VaultIdleLockMinutes, err)
	}
	if s := settingsResponseFrom(c); s.VaultIdleLockMinutes != 30 {
		t.Fatalf("response %+v", s)
	}
}

// The idle loop, unlocks and locks race in parallel without corrupting
// state (run under -race).
func TestIdleLoopAgainstUnlockAndLock(t *testing.T) {
	r := newIdleRig(t, 1)
	var wg sync.WaitGroup
	for i := range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 3 {
				if i%2 == 0 {
					r.srv.Lock()
				} else {
					_, _ = r.srv.Unlock([]byte(vaulttest.Passphrase))
				}
				r.advance(time.Minute)
			}
		}()
	}
	wg.Wait()
	settle()
}
