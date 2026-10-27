//go:build linux || darwin

package agent

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/valve-tech/jumpgate/internal/intent"
	"github.com/valve-tech/jumpgate/internal/jobs/jobstest"
)

// upgradeRig gives r an agent binary (a temp file holding "old") and counts
// its restarts.
func upgradeRig(t *testing.T, r *rig) (bin string, restarts *atomic.Int32) {
	t.Helper()
	bin = filepath.Join(r.dir, "jumpgate")
	if err := os.WriteFile(bin, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	n := &atomic.Int32{}
	r.a.cfg.BinaryPath = bin
	r.a.cfg.Restart = func() { n.Add(1) }
	return bin, n
}

func shaHex(b []byte) string { s := sha256.Sum256(b); return hex.EncodeToString(s[:]) }

func upgradePayload(body []byte) intent.AgentUpgradePayload {
	return intent.AgentUpgradePayload{Arch: runtime.GOARCH, Version: "v0.6.1", SHA256: shaHex(body), Size: int64(len(body))}
}

// sendUpgrade handles an upgrade envelope with body as its upload, then runs
// the step the route runs once the receipt is written.
func (r *rig) sendUpgrade(t *testing.T, pl intent.AgentUpgradePayload, body io.Reader) (intent.Receipt, []byte) {
	t.Helper()
	env := r.envelope(t, intent.KindAgentUpgrade, pl)
	ctx, up := withUpgradeBody(context.Background(), body)
	out := r.a.Handle(ctx, env)
	up.afterReceipt(r.a)
	return r.verified(t, out)
}

func noNew(t *testing.T, bin string) {
	t.Helper()
	if _, err := os.Stat(bin + ".new"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("%s.new left behind: %v", bin, err)
	}
}

func fileIs(t *testing.T, path, want string) {
	t.Helper()
	b, _ := os.ReadFile(path)
	if string(b) != want {
		t.Fatalf("%s = %q, want %q", path, b, want)
	}
}

func TestUpgradeInstallsAVerifiedBinary(t *testing.T) {
	r := newRig(t, false)
	bin, restarts := upgradeRig(t, r)
	got := ok[intent.AgentUpgradeResult](t)(r.sendUpgrade(t, upgradePayload([]byte("new")), strings.NewReader("new")))
	if !got.Upgraded || got.To != "v0.6.1" || got.From == "" {
		t.Fatalf("result %+v", got)
	}
	fileIs(t, bin, "new")
	fi, _ := os.Stat(bin)
	if fi.Mode().Perm() != 0o755 {
		t.Fatalf("mode %o", fi.Mode().Perm())
	}
	noNew(t, bin)
	if restarts.Load() != 1 {
		t.Fatalf("restarts %d", restarts.Load())
	}
}

func TestUpgradeRefusesAHashMismatch(t *testing.T) {
	r := newRig(t, false)
	bin, restarts := upgradeRig(t, r)
	pl := upgradePayload([]byte("new"))
	pl.SHA256 = shaHex([]byte("other"))
	rc, res := r.sendUpgrade(t, pl, strings.NewReader("new"))
	if rc.Status != intent.StatusFailed || !strings.Contains(string(res), "sha256") {
		t.Fatalf("status %d: %s", rc.Status, res)
	}
	fileIs(t, bin, "old")
	noNew(t, bin)
	if restarts.Load() != 0 {
		t.Fatal("restarted")
	}
}

func TestUpgradeRefusesAShortBody(t *testing.T) {
	r := newRig(t, false)
	bin, restarts := upgradeRig(t, r)
	pl := upgradePayload([]byte("new"))
	pl.Size = 10
	rc, res := r.sendUpgrade(t, pl, strings.NewReader("new"))
	if rc.Status != intent.StatusFailed || !strings.Contains(string(res), "short") {
		t.Fatalf("status %d: %s", rc.Status, res)
	}
	fileIs(t, bin, "old")
	noNew(t, bin)
	if restarts.Load() != 0 {
		t.Fatal("restarted")
	}
}

type failReader struct{ t *testing.T }

func (f failReader) Read([]byte) (int, error) {
	f.t.Error("the upgrade body was read")
	return 0, io.EOF
}

func TestUpgradeRefusesWhileAJobIsActive(t *testing.T) {
	r, _ := jobsRig(t)
	bin, restarts := upgradeRig(t, r)
	_ = ok[intent.JobRecord](t)(r.send(t, jobstest.TestKind, map[string]string{}, nil))
	rc, res := r.sendUpgrade(t, upgradePayload([]byte("new")), failReader{t})
	if got := rejection(t, rc, res).Code; got != intent.ReasonJobActive {
		t.Fatalf("code %s", got)
	}
	fileIs(t, bin, "old")
	noNew(t, bin)
	if restarts.Load() != 0 {
		t.Fatal("restarted")
	}
}

func TestUpgradeRefusesOnTheIntentRoute(t *testing.T) {
	r := newRig(t, false)
	upgradeRig(t, r)
	rc, res := r.send(t, intent.KindAgentUpgrade, upgradePayload([]byte("new")), nil)
	if got := rejection(t, rc, res); got.Code != intent.ReasonInvalidPayload || !strings.Contains(got.Message, "/v1/upgrade") {
		t.Fatalf("%+v", got)
	}
}

func TestUpgradeOfTheSameBinaryDoesNotRestart(t *testing.T) {
	r := newRig(t, false)
	bin, restarts := upgradeRig(t, r)
	if got := ok[intent.AgentUpgradeResult](t)(r.sendUpgrade(t, upgradePayload([]byte("old")), strings.NewReader("old"))); got.Upgraded {
		t.Fatalf("%+v", got)
	}
	fileIs(t, bin, "old")
	noNew(t, bin)
	if restarts.Load() != 0 {
		t.Fatal("restarted")
	}
}

func TestUpgradeRefusesAnotherArch(t *testing.T) {
	r := newRig(t, false)
	upgradeRig(t, r)
	pl := upgradePayload([]byte("new"))
	pl.Arch = "riscv64"
	if runtime.GOARCH == "riscv64" {
		pl.Arch = "amd64"
	}
	rc, res := r.sendUpgrade(t, pl, failReader{t})
	if got := rejection(t, rc, res).Code; got != intent.ReasonValidation {
		t.Fatalf("code %s", got)
	}
}

func TestUpgradePolicyCanRequireApproval(t *testing.T) {
	r := newRig(t, false)
	upgradeRig(t, r)
	path := filepath.Join(r.dir, "policy.json")
	p, _ := LoadPolicy(path)
	p.Kinds = map[string][]Tier{intent.KindAgentUpgrade: {TierRoutine, TierApproval}}
	if err := p.Save(path); err != nil {
		t.Fatal(err)
	}
	rc, res := r.sendUpgrade(t, upgradePayload([]byte("new")), failReader{t})
	if got := rejection(t, rc, res).Code; got != intent.ReasonUnauthorizedKind {
		t.Fatalf("code %s", got)
	}
}

func TestUpgradeEnvelopeHeaderIsCapped(t *testing.T) {
	r := newRig(t, false)
	bin, restarts := upgradeRig(t, r)
	orig := peerGate
	peerGate = func(*Agent, net.Conn, int) bool { return true }
	t.Cleanup(func() { peerGate = orig })
	sock := filepath.Join(shortDir(t), "agent.sock")
	ln, err := Listen(sock, -1, 0o660)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go Serve(ctx, r.a, ln)

	req, _ := http.NewRequest("POST", "http://agent/v1/upgrade", strings.NewReader("new"))
	req.Header.Set(UpgradeEnvelopeHeader, strings.Repeat("A", 65<<10))
	res, err := unixClient(sock).Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("status %d, want 400", res.StatusCode)
	}
	fileIs(t, bin, "old")
	noNew(t, bin)

	// A good one over the same route installs, then restarts.
	env := r.envelope(t, intent.KindAgentUpgrade, upgradePayload([]byte("new")))
	hb, _ := json.Marshal(env)
	req, _ = http.NewRequest("POST", "http://agent/v1/upgrade", bytes.NewReader([]byte("new")))
	req.Header.Set(UpgradeEnvelopeHeader, base64.RawURLEncoding.EncodeToString(hb))
	res, err = unixClient(sock).Do(req)
	if err != nil {
		t.Fatal(err)
	}
	var out intent.ReceiptEnvelope
	_ = json.NewDecoder(res.Body).Decode(&out)
	res.Body.Close()
	if got := ok[intent.AgentUpgradeResult](t)(r.verified(t, out)); !got.Upgraded {
		t.Fatalf("%+v", got)
	}
	fileIs(t, bin, "new")
	waitFor(t, func() bool { return restarts.Load() == 1 })
}
