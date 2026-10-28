package server

import (
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/valve-tech/jumpgate/internal/vault/vaulttest"
)

// fakeClock is a settable now.
type fakeClock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *fakeClock) now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *fakeClock) add(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = c.t.Add(d)
}

func TestPassBackoffDoublesPerUIDAndClearsOnSuccess(t *testing.T) {
	c := &fakeClock{t: time.Unix(1e9, 0)}
	b := passBackoff{now: c.now}
	for i := 1; i <= passBackoffFree; i++ {
		if n, d := b.failed(1000); n != i || d != 0 {
			t.Fatalf("failure %d: n %d wait %s, want free", i, n, d)
		}
	}
	for i, want := range []time.Duration{time.Second, 2 * time.Second, 4 * time.Second} {
		if _, d := b.failed(1000); d != want {
			t.Fatalf("failure %d: wait %s, want %s", passBackoffFree+1+i, d, want)
		}
	}
	if d := owed(&b, 1000); d != 4*time.Second {
		t.Fatalf("wait = %s, want 4s", d)
	}
	if d := owed(&b, 1001); d != 0 {
		t.Fatalf("another uid waits %s", d)
	}
	for range 40 {
		b.failed(1000)
	}
	if d := owed(&b, 1000); d != passBackoffMax {
		t.Fatalf("capped wait = %s, want %s", d, passBackoffMax)
	}
	c.add(passBackoffMax)
	if d := owed(&b, 1000); d != 0 {
		t.Fatalf("after the wait: %s", d)
	}
	b.succeeded(1000)
	if n, _ := b.failed(1000); n != 1 {
		t.Fatalf("after success the count is %d, want 1", n)
	}
}

const secretPass = "correct-horse-battery-staple-7f3a"

func TestSealRouteBacksOffAndNeverLogsThePassphrase(t *testing.T) {
	logs := captureLog(t)
	ts, s := peerServer(t, 1000, true, nil)
	c := &fakeClock{t: time.Unix(1e9, 0)}
	s.passBackoff.now = c.now
	initVault(t, ts)
	if _, err := s.Unlock([]byte("pw")); err != nil {
		t.Fatal(err)
	}
	for i := 0; i <= passBackoffFree; i++ {
		if st, _, _ := sealPost(t, ts, secretPass); st != http.StatusForbidden {
			t.Fatalf("wrong passphrase %d: %d, want 403", i+1, st)
		}
	}
	// Now the uid owes a second: even the right passphrase waits.
	if st, _, kek := sealPost(t, ts, "pw"); st != http.StatusTooManyRequests || kek != nil {
		t.Fatalf("during the backoff: %d, want 429", st)
	}
	c.add(time.Second)
	if st, e, kek := sealPost(t, ts, "pw"); st != http.StatusOK || len(kek) != 32 {
		t.Fatalf("after the backoff: %d %+v", st, e)
	}
	out := logs.String()
	if !strings.Contains(out, "wrong passphrase from uid 1000") {
		t.Fatalf("no failure logged with the uid:\n%s", out)
	}
	if strings.Contains(out, secretPass) || strings.Contains(out, "cHc=") {
		t.Fatalf("the log holds a passphrase:\n%s", out)
	}
}

func TestUnlockRouteBacksOffAndNeverLogsThePassphrase(t *testing.T) {
	logs := captureLog(t)
	p, _ := vaulttest.Path(t)
	s, ts, token := vaultServer(t, p)
	c := &fakeClock{t: time.Unix(1e9, 0)}
	s.passBackoff.now = c.now
	for i := 0; i <= passBackoffFree; i++ {
		if res, _ := do(t, ts, token, "POST", "/api/vault/unlock", unlockBody(secretPass)); res.StatusCode != http.StatusForbidden {
			t.Fatalf("wrong passphrase %d: %d, want 403", i+1, res.StatusCode)
		}
	}
	res, _ := do(t, ts, token, "POST", "/api/vault/unlock", unlockBody(vaulttest.Passphrase))
	if res.StatusCode != http.StatusTooManyRequests || res.Header.Get("Retry-After") != "1" {
		t.Fatalf("during the backoff: %d Retry-After %q, want 429 1", res.StatusCode, res.Header.Get("Retry-After"))
	}
	c.add(time.Second)
	if res, e := do(t, ts, token, "POST", "/api/vault/unlock", unlockBody(vaulttest.Passphrase)); res.StatusCode != http.StatusOK {
		t.Fatalf("after the backoff: %d %+v", res.StatusCode, e)
	}
	out := logs.String()
	if !strings.Contains(out, "vault unlock: wrong passphrase from uid ") {
		t.Fatalf("no failure logged:\n%s", out)
	}
	if strings.Contains(out, secretPass) || strings.Contains(out, vaulttest.Passphrase) {
		t.Fatalf("the log holds a passphrase:\n%s", out)
	}
}

// owed is the wait b still holds against uid.
func owed(b *passBackoff, uid int) time.Duration {
	b.mu.Lock()
	defer b.mu.Unlock()
	return max(b.fail[uid].until.Sub(b.clock()), 0)
}

// Concurrent wrong passphrases from one uid cannot each pass the check
// before the first failure is recorded: at most one reaches the KDF.
func TestSealRouteAdmitsOneAttemptPerUIDAtATime(t *testing.T) {
	ts, s := peerServer(t, 1000, true, nil)
	initVault(t, ts)
	if _, err := s.Unlock([]byte("pw")); err != nil {
		t.Fatal(err)
	}
	var checks atomic.Int32
	release := make(chan struct{})
	passCheckHook = func() { checks.Add(1); <-release }
	t.Cleanup(func() { passCheckHook = nil })

	const n = 8
	codes := make(chan int, n)
	for range n {
		go func() { st, _, _ := sealPost(t, ts, "nope"); codes <- st }()
	}
	for range n - 1 {
		if st := <-codes; st != http.StatusTooManyRequests {
			t.Errorf("a concurrent attempt got %d, want 429", st)
		}
	}
	close(release)
	if st := <-codes; st != http.StatusForbidden {
		t.Errorf("the admitted attempt got %d, want 403", st)
	}
	if got := checks.Load(); got != 1 {
		t.Fatalf("%d attempts reached the passphrase check, want 1", got)
	}
	// The reservation is released: the next attempt is checked.
	if st, _, kek := sealPost(t, ts, "pw"); st != http.StatusOK || len(kek) != 32 {
		t.Fatalf("after the race: %d", st)
	}
}
