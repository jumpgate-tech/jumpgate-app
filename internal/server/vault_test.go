package server

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/fstest"
	"time"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/intent"
	"github.com/valve-tech/jumpgate/internal/jobs/jobstest"
	"github.com/valve-tech/jumpgate/internal/signer"
	"github.com/valve-tech/jumpgate/internal/testutil"
	"github.com/valve-tech/jumpgate/internal/vault"
	"github.com/valve-tech/jumpgate/internal/vault/vaulttest"
)

// vaultServer is a server with no box and a vault file at path (which need
// not exist yet).
func vaultServer(t *testing.T, path string) (*Server, *httptest.Server, string) {
	t.Helper()
	testutil.Home(t)
	token := NewSessionToken()
	srv := New(Config{Token: token, UI: fstest.MapFS{}, VaultPath: path})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(func() { ts.Close(); srv.fleet.stop(); srv.jobs.stopWithin(time.Second) })
	return srv, ts, token
}

func unlockBody(pass string) string {
	raw, _ := json.Marshal(api.UnlockRequest{Passphrase: []byte(pass)})
	return string(raw)
}

func vaultView(t *testing.T, ts *httptest.Server, token string) api.VaultView {
	t.Helper()
	req, _ := http.NewRequest("GET", ts.URL+"/api/vault", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var v api.VaultView
	if res.StatusCode != 200 || json.NewDecoder(res.Body).Decode(&v) != nil {
		t.Fatalf("GET /api/vault = %d", res.StatusCode)
	}
	return v
}

func TestVaultRoutesUnlockAndLock(t *testing.T) {
	p, pass := vaulttest.Path(t)
	_, ts, token := vaultServer(t, p)
	if v := vaultView(t, ts, token); v.State != "locked" || len(v.Slots) == 0 {
		t.Fatalf("before unlock: %+v", v)
	}
	res, e := do(t, ts, token, "POST", "/api/vault/unlock", unlockBody(string(pass)))
	if res.StatusCode != 200 {
		t.Fatalf("unlock = %d %+v", res.StatusCode, e)
	}
	if v := vaultView(t, ts, token); v.State != "unlocked" {
		t.Fatalf("after unlock: %+v", v)
	}
	if res, _ := do(t, ts, token, "POST", "/api/vault/lock", ""); res.StatusCode != 200 {
		t.Fatalf("lock = %d", res.StatusCode)
	}
	if v := vaultView(t, ts, token); v.State != "locked" {
		t.Fatalf("after lock: %+v", v)
	}
}

func TestUnlockAnswersWrongPassphrase(t *testing.T) {
	p, _ := vaulttest.Path(t)
	_, ts, token := vaultServer(t, p)
	res, e := do(t, ts, token, "POST", "/api/vault/unlock", unlockBody("not it"))
	if res.StatusCode != http.StatusForbidden || e.Code != api.CodeWrongPassphrase {
		t.Fatalf("got %d %+v", res.StatusCode, e)
	}
	if v := vaultView(t, ts, token); v.State != "locked" {
		t.Fatalf("state %+v", v)
	}
}

func TestUnlockRefusesAnEmptyOrHugeBody(t *testing.T) {
	p, _ := vaulttest.Path(t)
	_, ts, token := vaultServer(t, p)
	if res, _ := do(t, ts, token, "POST", "/api/vault/unlock", unlockBody("")); res.StatusCode != 400 {
		t.Fatalf("empty passphrase = %d", res.StatusCode)
	}
	if res, _ := do(t, ts, token, "POST", "/api/vault/unlock", unlockBody(strings.Repeat("x", 8<<10))); res.StatusCode != 400 {
		t.Fatalf("8 KiB passphrase = %d", res.StatusCode)
	}
}

func TestUnlockAnswersVaultCorrupt(t *testing.T) {
	p, pass := vaulttest.Path(t)
	raw, _ := os.ReadFile(p)
	raw[len(raw)-1] ^= 1
	if err := os.WriteFile(p, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	srv, ts, token := vaultServer(t, p)
	res, e := do(t, ts, token, "POST", "/api/vault/unlock", unlockBody(string(pass)))
	if res.StatusCode != http.StatusInternalServerError || e.Code != api.CodeVaultCorrupt {
		t.Fatalf("got %d %+v", res.StatusCode, e)
	}
	if v := vaultView(t, ts, token); v.State != "locked" {
		t.Fatalf("state %+v", v)
	}
	if sg, _ := srv.signer(); sg != nil {
		t.Fatal("a corrupt vault produced a signer")
	}
}

func TestUnlockWithNoVault(t *testing.T) {
	src, _ := vaulttest.Path(t)
	p := filepath.Join(t.TempDir(), "vault")
	_, ts, token := vaultServer(t, p)
	res, e := do(t, ts, token, "POST", "/api/vault/unlock", unlockBody("pw"))
	if res.StatusCode != http.StatusConflict || e.Code != api.CodeNoVault {
		t.Fatalf("got %d %+v", res.StatusCode, e)
	}
	if v := vaultView(t, ts, token); v.State != "none" {
		t.Fatalf("state %+v", v)
	}
	if err := os.Rename(src, p); err != nil {
		t.Fatal(err)
	}
	if v := vaultView(t, ts, token); v.State != "locked" {
		t.Fatalf("after the file appeared: %+v", v)
	}
}

func TestUnsafeVaultIsReportedNotTightened(t *testing.T) {
	testutil.RequireUnix(t)
	p, _ := vaulttest.Path(t)
	if err := os.Chmod(p, 0o640); err != nil {
		t.Fatal(err)
	}
	_, ts, token := vaultServer(t, p)
	if v := vaultView(t, ts, token); v.State != "unusable" || !strings.Contains(v.Reason, "chmod") {
		t.Fatalf("state %+v", v)
	}
	res, e := do(t, ts, token, "POST", "/api/vault/unlock", unlockBody(vaulttest.Passphrase))
	if res.StatusCode != http.StatusInternalServerError || e.Code != api.CodeVaultUnsafe {
		t.Fatalf("unlock got %d %+v", res.StatusCode, e)
	}
	fi, _ := os.Stat(p)
	if fi.Mode().Perm() != 0o640 {
		t.Fatalf("mode is now %o", fi.Mode().Perm())
	}
}

// vaultRig is a paired box whose controller key lives in a locked vault.
// unlocks counts Config.Unlocked calls.
type vaultRig struct {
	*boxRig
	v       *vault.Vault
	unlocks *atomic.Int32
}

func newVaultRig(t *testing.T, opts ...func(*Config)) *vaultRig {
	t.Helper()
	r := &vaultRig{unlocks: new(atomic.Int32)}
	r.boxRig = newBoxRig(t, append([]func(*Config){func(c *Config) {
		k := c.Signer.(*signer.Key)
		p, pass := vaulttest.Path(t)
		v, err := vault.Open(p)
		if err != nil {
			t.Fatal(err)
		}
		if err := v.Unlock(pass); err != nil {
			t.Fatal(err)
		}
		kb := k.Bytes()
		if err := v.Put("controller-key", []byte(hex.EncodeToString(kb)), "test"); err != nil {
			t.Fatal(err)
		}
		v.Lock()
		if _, err := config.Update(func(cc *config.Config) error {
			cc.Controller = &config.Controller{KeyStore: string(signer.StoreVault), KeyRef: "controller-key", Address: k.Address().Hex()}
			return nil
		}); err != nil {
			t.Fatal(err)
		}
		r.v = v
		c.Signer, c.SignerErr, c.Vault = nil, signer.ErrVaultStore, v
		c.Unlocked = func(v *vault.Vault) (signer.Signer, error) {
			r.unlocks.Add(1)
			return signer.OpenVault(v, "controller-key")
		}
	}}, opts...)...)
	return r
}

// unlock unlocks and waits for what the unlock starts (the sweep).
func (r *vaultRig) unlock(t *testing.T) {
	t.Helper()
	if _, err := r.srv.Unlock([]byte(vaulttest.Passphrase)); err != nil {
		t.Fatal(err)
	}
	r.srv.bg.Wait()
}

// lock locks and lets any intent signed just before it land.
func (r *vaultRig) lock() {
	r.srv.Lock()
	time.Sleep(15 * jobsDriveEvery)
}

func TestLockedServerRefusesSignedRoutes(t *testing.T) {
	requireAgentPeer(t)
	r := newVaultRig(t)
	for _, rt := range []struct{ path, body string }{
		{"/api/targets/box/intent/" + intent.KindStatusRead, "{}"},
		{"/api/targets/box/pair", "{}"},
		{"/api/jobs/box", `{"kind":"` + jobstest.TestKind + `","inputs":{}}`},
		{"/api/fleet/box/check", ""},
	} {
		res, e := do(t, r.ts, r.token, "POST", rt.path, rt.body)
		if res.StatusCode != http.StatusServiceUnavailable || e.Code != api.CodeLocked || e.Hint == "" {
			t.Errorf("POST %s = %d %+v", rt.path, res.StatusCode, e)
		}
	}
}

func TestLockedServerServesViews(t *testing.T) {
	requireAgentPeer(t)
	r := newVaultRig(t, func(c *Config) { c.Updater = &fakeUpdater{} })
	for _, p := range []string{"/api/targets", "/api/settings", "/api/fleet", "/api/update", "/api/controller",
		"/api/vault", "/api/gateways", "/api/jobs", "/api/ui-prefs"} {
		if code, body := r.get(t, p); code != 200 {
			t.Errorf("GET %s = %d %s", p, code, body)
		}
	}
	var cv api.ControllerView
	r.do(t, "GET", "/api/controller", nil, &cv)
	if cv.State != "locked" {
		t.Fatalf("controller %+v", cv)
	}
}

func TestFleetRowsSayLocked(t *testing.T) {
	requireAgentPeer(t)
	r := newVaultRig(t)
	deadline := time.Now().Add(10 * time.Second)
	for {
		var f api.Fleet
		r.do(t, "GET", "/api/fleet", nil, &f)
		row, ok := f.Row("box")
		if ok && row.Error != nil {
			if row.Error.Code != api.CodeLocked || row.Reachable || !f.Locked {
				t.Fatalf("fleet %+v row %+v error %+v", f, row, row.Error)
			}
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("no locked row: %+v", f)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func boxSeq(t *testing.T) uint64 {
	t.Helper()
	c, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	return c.Targets[0].Agent.NextSeq
}

// A job this controller holds waits while the vault is locked (nothing is
// signed), and resumes by itself on unlock.
func TestUnlockResumesOwnJobs(t *testing.T) {
	requireAgentPeer(t)
	r := newVaultRig(t)
	r.unlock(t)
	rec := r.startJobUnlisted(t, `{}`)
	r.lock()
	seq := boxSeq(t)
	r.srv.jobs.sweep(context.Background())
	time.Sleep(5 * jobsDriveEvery)
	if n := r.drivers(); n != 0 || boxSeq(t) != seq {
		t.Fatalf("while locked: %d drivers, seq %d -> %d", n, seq, boxSeq(t))
	}
	r.unlock(t)
	deadline := time.Now().Add(10 * time.Second)
	for r.readOnBox(t, rec.ID).State != intent.JobSucceeded {
		if time.Now().After(deadline) {
			t.Fatal("the unlock did not resume the job")
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// A driver running when the vault locks backs off without signing or
// ending, and goes on once unlocked.
func TestADriverWaitsWhileLocked(t *testing.T) {
	requireAgentPeer(t)
	r := newVaultRig(t)
	r.unlock(t)
	var v api.JobView
	r.do(t, "POST", "/api/jobs/box", api.JobStart{Kind: jobstest.TestKind, Inputs: json.RawMessage(`{"AskAt":"two"}`)}, &v)
	asked := waitJob(t, r.boxRig, v.ID, intent.JobNeedsDecision)
	r.lock()
	seq := boxSeq(t)
	time.Sleep(10 * jobsDriveEvery)
	if r.drivers() != 1 || boxSeq(t) != seq {
		t.Fatalf("while locked: %d drivers, seq %d -> %d", r.drivers(), seq, boxSeq(t))
	}
	r.unlock(t)
	if code := r.do(t, "POST", "/api/jobs/box/"+v.ID+"/decide", api.JobDecision{Decision: asked.Question.ID, Option: "go"}, nil); code != 200 {
		t.Fatalf("decide = %d", code)
	}
	waitJob(t, r.boxRig, v.ID, intent.JobSucceeded)
}

func TestLockDropsTheSigner(t *testing.T) {
	requireAgentPeer(t)
	r := newVaultRig(t)
	if _, err := r.srv.signer(); !errors.Is(err, vault.ErrLocked) {
		t.Fatalf("before unlock: %v", err)
	}
	r.unlock(t)
	if sg, err := r.srv.signer(); sg == nil || err != nil {
		t.Fatalf("unlocked: %v %v", sg, err)
	}
	r.srv.Lock()
	if _, err := r.srv.signer(); !errors.Is(err, vault.ErrLocked) {
		t.Fatalf("after lock: %v", err)
	}
	r.unlock(t)
	if n := r.unlocks.Load(); n != 2 {
		t.Fatalf("Config.Unlocked ran %d times for 2 unlocks", n)
	}
	// The vault locking on its own (a reload it cannot decrypt) also stops
	// signing, without a call to Lock.
	r.v.Lock()
	if _, err := r.srv.signer(); !errors.Is(err, vault.ErrLocked) {
		t.Fatalf("vault locked underneath: %v", err)
	}
}

// Unlocks, locks and signed requests in parallel: a signed route answers
// either its result or 503 locked, never anything else, and the final state
// agrees with the signer.
func TestUnlockLockAndSignedRoutesInParallel(t *testing.T) {
	requireAgentPeer(t)
	r := newVaultRig(t)
	var wg sync.WaitGroup
	var bad atomic.Int32
	for i := 0; i < 3; i++ {
		wg.Add(2)
		go func() { defer wg.Done(); _, _ = r.srv.Unlock([]byte(vaulttest.Passphrase)) }()
		go func() { defer wg.Done(); r.srv.Lock() }()
	}
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			code, raw := r.req(t, "POST", "/api/fleet/box/check", nil)
			var e api.Error
			_ = json.Unmarshal(raw, &e)
			if code != 200 && (code != http.StatusServiceUnavailable || e.Code != api.CodeLocked) {
				bad.Add(1)
				t.Errorf("check = %d %s", code, raw)
			}
			r.req(t, "GET", "/api/vault", nil)
		}()
	}
	wg.Wait()
	if bad.Load() != 0 {
		return
	}
	st := r.srv.VaultState().State
	_, err := r.srv.signer()
	if (st == "locked") != errors.Is(err, vault.ErrLocked) {
		t.Fatalf("state %s but signer err %v", st, err)
	}
}
