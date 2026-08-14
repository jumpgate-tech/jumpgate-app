package server

import (
	"context"
	"net/http"
	"sync"
	"testing"

	"github.com/valve-tech/valve-node-app/internal/updatecheck"
)

// fakeUpdater is a test double for updateSource. It returns a canned release
// (or error) and counts calls, so a test can prove the server caches the
// result, skips the network when notices are off, and forces a live check on
// refresh. The truth table for "is an update available" lives in the
// updatecheck package's own tests; these cover only what the SERVER adds.
type fakeUpdater struct {
	mu    sync.Mutex
	rel   updatecheck.Release
	err   error
	calls int
}

func (f *fakeUpdater) Latest(_ context.Context) (updatecheck.Release, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	return f.rel, f.err
}

func (f *fakeUpdater) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

// newUpdateTestServer builds an API test server whose update source is fake, so
// nothing here reaches GitHub.
func newUpdateTestServer(t *testing.T, fake *fakeUpdater) *apiTestServer {
	t.Helper()
	return newAPITestServerCfg(t, nil, func(c *Config) { c.Updater = fake })
}

func TestUpdateGetReportsLatest(t *testing.T) {
	fake := &fakeUpdater{rel: updatecheck.Release{
		Version: "v9.9.9",
		URL:     "https://example.test/releases/v9.9.9",
	}}
	a := newUpdateTestServer(t, fake)

	res := a.do(t, "GET", "/api/update", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/update = %d, want 200", res.StatusCode)
	}
	got := decodeJSON[updateResponse](t, res)

	if !got.NotifyEnabled {
		t.Errorf("NotifyEnabled = false, want true (a fresh config keeps notices on)")
	}
	if got.Latest != "v9.9.9" {
		t.Errorf("Latest = %q, want %q", got.Latest, "v9.9.9")
	}
	if got.ReleaseURL != "https://example.test/releases/v9.9.9" {
		t.Errorf("ReleaseURL = %q, want the fake's URL", got.ReleaseURL)
	}
	// The test binary is built without -ldflags, so buildinfo.Version() is
	// "dev". Evaluate never flags an update for a dev build, so this is false
	// here by design — the "available" path is proven in updatecheck's tests.
	if got.UpdateAvailable {
		t.Errorf("UpdateAvailable = true for a dev build, want false")
	}
	if got.Current != "dev" {
		t.Errorf("Current = %q, want %q for an un-injected test binary", got.Current, "dev")
	}
}

func TestUpdateCachedAcrossCalls(t *testing.T) {
	fake := &fakeUpdater{rel: updatecheck.Release{Version: "v9.9.9"}}
	a := newUpdateTestServer(t, fake)

	a.do(t, "GET", "/api/update", nil).Body.Close()
	a.do(t, "GET", "/api/update", nil).Body.Close()

	// The two calls are milliseconds apart, well within updateCheckInterval,
	// so the second must be served from the cache — GitHub is asked once.
	if n := fake.callCount(); n != 1 {
		t.Errorf("update source called %d times, want 1 (second read should hit the cache)", n)
	}
}

func TestUpdateNotifyDisabledSkipsNetwork(t *testing.T) {
	fake := &fakeUpdater{rel: updatecheck.Release{Version: "v9.9.9"}}
	a := newUpdateTestServer(t, fake)

	// "Don't prompt me" through the real settings path.
	res := a.do(t, "PUT", "/api/settings", map[string]any{"updateNotifyEnabled": false})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("PUT /api/settings = %d, want 200", res.StatusCode)
	}
	res.Body.Close()

	// A plain GET is the banner's background load — it must make no network call.
	got := decodeJSON[updateResponse](t, a.do(t, "GET", "/api/update", nil))
	if got.NotifyEnabled {
		t.Errorf("NotifyEnabled = true after disabling, want false")
	}
	if got.Latest != "" {
		t.Errorf("Latest = %q, want empty on the background load with notices off", got.Latest)
	}
	if n := fake.callCount(); n != 0 {
		t.Errorf("update source called %d times on the background load, want 0 (no network)", n)
	}
}

func TestUpdateRefreshChecksEvenWhenNotifyDisabled(t *testing.T) {
	fake := &fakeUpdater{rel: updatecheck.Release{Version: "v9.9.9"}}
	a := newUpdateTestServer(t, fake)

	a.do(t, "PUT", "/api/settings", map[string]any{"updateNotifyEnabled": false}).Body.Close()

	// The Settings page pulls with ?refresh=1 — a manual check must reach GitHub
	// even when notices are off, and return the latest it finds.
	got := decodeJSON[updateResponse](t, a.do(t, "GET", "/api/update?refresh=1", nil))
	if n := fake.callCount(); n != 1 {
		t.Errorf("update source called %d times on refresh, want 1", n)
	}
	if got.Latest != "v9.9.9" {
		t.Errorf("Latest = %q, want %q on a manual refresh", got.Latest, "v9.9.9")
	}
	if got.NotifyEnabled {
		t.Errorf("NotifyEnabled = true, want false — the banner stays off even after a manual pull")
	}
}

func TestUpdateRefreshBypassesCache(t *testing.T) {
	fake := &fakeUpdater{rel: updatecheck.Release{Version: "v9.9.9"}}
	a := newUpdateTestServer(t, fake)

	a.do(t, "GET", "/api/update", nil).Body.Close()           // fills the cache (call 1)
	a.do(t, "GET", "/api/update?refresh=1", nil).Body.Close() // forces a fresh check (call 2)

	if n := fake.callCount(); n != 2 {
		t.Errorf("update source called %d times, want 2 (refresh must bypass the cache)", n)
	}
}

func TestUpdateCheckErrorSurfaces(t *testing.T) {
	fake := &fakeUpdater{err: context.DeadlineExceeded}
	a := newUpdateTestServer(t, fake)

	got := decodeJSON[updateResponse](t, a.do(t, "GET", "/api/update", nil))
	if !got.NotifyEnabled {
		t.Errorf("NotifyEnabled = false, want true")
	}
	if got.CheckError == "" {
		t.Errorf("CheckError is empty, want the failed check's reason surfaced")
	}
	if got.UpdateAvailable {
		t.Errorf("UpdateAvailable = true after a failed check, want false")
	}
}

func TestSettingsUpdateNotifyToggle(t *testing.T) {
	a := newUpdateTestServer(t, &fakeUpdater{})

	// Default: on.
	got := decodeJSON[settingsResponse](t, a.do(t, "GET", "/api/settings", nil))
	if !got.UpdateNotifyEnabled {
		t.Errorf("default UpdateNotifyEnabled = false, want true")
	}

	// Turn it off and read it back from a fresh GET.
	a.do(t, "PUT", "/api/settings", map[string]any{"updateNotifyEnabled": false}).Body.Close()
	got = decodeJSON[settingsResponse](t, a.do(t, "GET", "/api/settings", nil))
	if got.UpdateNotifyEnabled {
		t.Errorf("UpdateNotifyEnabled = true after disabling, want false")
	}
}
