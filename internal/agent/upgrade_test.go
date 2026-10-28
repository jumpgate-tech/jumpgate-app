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
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/valve-tech/jumpgate/internal/intent"
	"github.com/valve-tech/jumpgate/internal/jobs/jobstest"
	"github.com/valve-tech/jumpgate/internal/signer"
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
	allowRoutineUpgrade(t, r)
	return bin, n
}

// allowRoutineUpgrade opts r's box in to upgrades on the controller's
// signature alone, as policy.json can.
func allowRoutineUpgrade(t *testing.T, r *rig) {
	t.Helper()
	path := filepath.Join(r.dir, "policy.json")
	p, err := LoadPolicy(path)
	if err != nil {
		t.Fatal(err)
	}
	if p.Kinds == nil {
		p.Kinds = map[string][]Tier{}
	}
	p.Kinds[intent.KindAgentUpgrade] = []Tier{TierRoutine}
	if err := p.Save(path); err != nil {
		t.Fatal(err)
	}
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

// noNew checks nothing but the binary itself was left next to it: no .new
// and no staged temp file.
func noNew(t *testing.T, bin string) {
	t.Helper()
	if _, err := os.Stat(bin + ".new"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("%s.new left behind: %v", bin, err)
	}
	ents, _ := os.ReadDir(filepath.Dir(bin))
	for _, e := range ents {
		if strings.HasPrefix(e.Name(), ".jumpgate-upgrade-") {
			t.Fatalf("staged file %s left behind", e.Name())
		}
	}
}

// envelopeFrom signs an intent of kind from k at seq, for a second
// controller enrolled on the rig.
func (r *rig) envelopeFrom(t *testing.T, k *signer.Key, seq uint64, kind string, payload any) intent.Envelope {
	t.Helper()
	body, _ := json.Marshal(payload)
	n, _ := intent.NewNonce()
	i := intent.Intent{Agent: r.agentKey.Address(), Controller: k.Address(), Seq: seq, Nonce: n,
		IssuedAt: uint64(r.now.Unix()), Expiry: uint64(r.now.Unix() + 120), Kind: kind, PayloadHash: intent.Hash(body)}
	sig, err := k.SignTypedData(context.Background(), i.TypedData())
	if err != nil {
		t.Fatal(err)
	}
	return intent.Envelope{Intent: i.JSON(), Payload: body, Sigs: []string{sig.Hex()}}
}

// hookReader serves data, then runs atEnd once when it is read past it.
type hookReader struct {
	r     io.Reader
	atEnd func()
	once  sync.Once
}

func (h *hookReader) Read(p []byte) (int, error) {
	n, err := h.r.Read(p)
	if err == io.EOF {
		h.once.Do(h.atEnd)
	}
	return n, err
}

// Two controllers upgrading at once: the gate takes one upgrade at a time,
// so the second is refused busy while the first streams, and the binary is
// the first one's, whole.
func TestConcurrentUpgradesTakeTurns(t *testing.T) {
	r := newRig(t, false)
	bin, restarts := upgradeRig(t, r)
	other, _ := signer.GenerateKey()
	enroll(t, r, other, TierRoutine)

	var second intent.ReceiptEnvelope
	first := &hookReader{r: strings.NewReader("first"), atEnd: func() {
		env := r.envelopeFrom(t, other, 1, intent.KindAgentUpgrade, upgradePayload([]byte("second")))
		ctx, _ := withUpgradeBody(context.Background(), strings.NewReader("second"))
		done := make(chan struct{})
		go func() { defer close(done); second = r.a.Handle(ctx, env) }()
		<-done
	}}
	rc, res := r.sendUpgrade(t, upgradePayload([]byte("first")), first)
	if got := ok[intent.AgentUpgradeResult](t)(rc, res); !got.Upgraded {
		t.Fatalf("first %+v", got)
	}
	rc2, res2 := r.verified(t, second)
	if got := rejection(t, rc2, res2).Code; got != intent.ReasonBusy {
		t.Fatalf("second: %s", got)
	}
	fileIs(t, bin, "first")
	noNew(t, bin)
	if restarts.Load() != 1 {
		t.Fatalf("restarts %d", restarts.Load())
	}
}

// Many upgrades racing from different controllers never leave a torn or
// staged file: the binary is exactly one of the bodies sent.
func TestRacingUpgradesLeaveOneWholeBinary(t *testing.T) {
	r := newRig(t, false)
	bin, _ := upgradeRig(t, r)
	var keys []*signer.Key
	for range 6 {
		k, _ := signer.GenerateKey()
		enroll(t, r, k, TierRoutine)
		keys = append(keys, k)
	}
	var wg sync.WaitGroup
	bodies := map[string]bool{}
	for i, k := range keys {
		body := strings.Repeat(string(rune('a'+i)), 4096)
		bodies[body] = true
		env := r.envelopeFrom(t, k, 1, intent.KindAgentUpgrade, upgradePayload([]byte(body)))
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx, up := withUpgradeBody(context.Background(), strings.NewReader(body))
			r.a.Handle(ctx, env)
			up.afterReceipt(r.a)
		}()
	}
	wg.Wait()
	got, _ := os.ReadFile(bin)
	if !bodies[string(got)] {
		t.Fatalf("binary is none of the bodies sent (%d bytes)", len(got))
	}
	noNew(t, bin)
}

// A job that starts while the binary streams stops the swap: the old binary
// stays, nothing staged is left and the gate does not restart.
func TestUpgradeRechecksJobsBeforeTheSwap(t *testing.T) {
	r, _ := jobsRig(t)
	bin, restarts := upgradeRig(t, r)
	other, _ := signer.GenerateKey()
	enroll(t, r, other, TierRoutine)
	body := &hookReader{r: strings.NewReader("new"), atEnd: func() {
		env := r.envelopeFrom(t, other, 1, jobstest.TestKind, map[string]string{})
		_ = ok[intent.JobRecord](t)(r.verified(t, r.a.Handle(context.Background(), env)))
	}}
	rc, res := r.sendUpgrade(t, upgradePayload([]byte("new")), body)
	if got := rejection(t, rc, res).Code; got != intent.ReasonJobActive {
		t.Fatalf("code %s: %s", got, res)
	}
	fileIs(t, bin, "old")
	noNew(t, bin)
	if restarts.Load() != 0 {
		t.Fatal("restarted")
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

func TestUpgradeNeedsApprovalByDefault(t *testing.T) {
	r := newRig(t, false)
	upgradeRig(t, r)
	path := filepath.Join(r.dir, "policy.json")
	p, _ := LoadPolicy(path)
	p.Kinds = nil // as a fresh box has it
	if err := p.Save(path); err != nil {
		t.Fatal(err)
	}
	rc, res := r.sendUpgrade(t, upgradePayload([]byte("new")), failReader{t})
	rej := rejection(t, rc, res)
	if rej.Code != intent.ReasonUnauthorizedKind {
		t.Fatalf("code %s", rej.Code)
	}
	if !strings.Contains(rej.Message, "approval signature") || !strings.Contains(rej.Message, `"agent.upgrade": ["routine"]`) {
		t.Fatalf("message %q does not say how to opt in", rej.Message)
	}
	want := []Tier{TierRoutine, TierApproval}
	if got := DefaultRequirements[intent.KindAgentUpgrade]; len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("default tiers %v, want %v", got, want)
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

// The restart may kill the gate the moment it is asked for, so the whole
// receipt must be on the wire by then: a Content-Length response, written in
// full, not a chunked one whose last chunk only the handler's return sends.
func TestUpgradeReceiptIsCompleteBeforeTheRestart(t *testing.T) {
	r := newRig(t, false)
	_, _ = upgradeRig(t, r)
	rec := httptest.NewRecorder()
	var atRestart, length string
	r.a.cfg.Restart = func() {
		atRestart, length = rec.Body.String(), rec.Header().Get("Content-Length")
	}
	env := r.envelope(t, intent.KindAgentUpgrade, upgradePayload([]byte("new")))
	hb, _ := json.Marshal(env)
	req := httptest.NewRequest("POST", "http://agent/v1/upgrade", strings.NewReader("new"))
	req.Header.Set(UpgradeEnvelopeHeader, base64.RawURLEncoding.EncodeToString(hb))
	serveUpgrade(r.a, rec, req)
	if atRestart == "" {
		t.Fatal("no restart, or nothing written before it")
	}
	if length != strconv.Itoa(len(atRestart)) {
		t.Fatalf("Content-Length %q for a %d-byte receipt at restart", length, len(atRestart))
	}
}
