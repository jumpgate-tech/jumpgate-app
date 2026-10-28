package agent

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/valve-tech/jumpgate/internal/buildinfo"
	"github.com/valve-tech/jumpgate/internal/fsatomic"
	"github.com/valve-tech/jumpgate/internal/intent"
)

// UpgradeEnvelopeHeader carries an agent.upgrade's signed envelope,
// base64url(json(intent.Envelope)), so the request body can be the binary.
const UpgradeEnvelopeHeader = "Jumpgate-Envelope"

const (
	maxUpgradeHeader = 64 << 10
	maxUpgrade       = 256 << 20
)

// upgradeBody is the upload an agent.upgrade carries through Handle. Only
// /v1/upgrade puts one in the context.
type upgradeBody struct {
	r        io.Reader
	upgraded atomic.Bool
}

type upgradeKey struct{}

// withUpgradeBody hands body to an agent.upgrade handled under ctx.
func withUpgradeBody(ctx context.Context, body io.Reader) (context.Context, *upgradeBody) {
	u := &upgradeBody{r: body}
	return context.WithValue(ctx, upgradeKey{}, u), u
}

// afterReceipt restarts the agent when the upgrade installed a new binary.
// The route calls it only once the receipt is written.
func (u *upgradeBody) afterReceipt(a *Agent) {
	if !u.upgraded.Load() || a.cfg.Restart == nil {
		return
	}
	// A job that started after the swap keeps the gate running; the new
	// binary takes over at the next restart.
	if rej := a.upgradeJobCheck(); rej != nil {
		a.logf("upgrade installed; not restarting: %s", rej.Message)
		return
	}
	a.cfg.Restart()
}

// upgrade runs an admitted agent.upgrade. Everything that can refuse is
// checked before a byte of the body is read; the binary is staged next to
// the old one and renamed over it only once its size and hash match, so a
// failure leaves the old binary in place and no staged file behind.
func (a *Agent) upgrade(ctx context.Context, payload []byte) ([]byte, uint8, *Reject) {
	up, _ := ctx.Value(upgradeKey{}).(*upgradeBody)
	if up == nil {
		return nil, 0, reject(intent.ReasonInvalidPayload, "agent.upgrade is sent to /v1/upgrade with the binary")
	}
	pl, rej := decode[intent.AgentUpgradePayload](payload)
	if rej != nil {
		return nil, 0, rej
	}
	if pl.Arch != runtime.GOARCH {
		return nil, 0, reject(intent.ReasonValidation, fmt.Sprintf("this gate runs on %s; the binary sent is for %q", runtime.GOARCH, pl.Arch))
	}
	if pl.Size < 1 || pl.Size > maxUpgrade {
		return nil, 0, reject(intent.ReasonInvalidPayload, fmt.Sprintf("size %d is outside 1..%d", pl.Size, maxUpgrade))
	}
	want, err := hex.DecodeString(pl.SHA256)
	if err != nil || len(want) != sha256.Size {
		return nil, 0, reject(intent.ReasonInvalidPayload, "sha256 must be 64 hex characters")
	}
	if a.cfg.BinaryPath == "" {
		return nil, 0, reject(intent.ReasonValidation, "this gate has no binary path to upgrade")
	}
	// One upgrade at a time on the whole gate, whichever controller sent
	// it: Handle's per-controller claim does not cover two controllers.
	if !a.upgradeMu.TryLock() {
		return nil, 0, reject(intent.ReasonBusy, "another upgrade is running on this gate")
	}
	defer a.upgradeMu.Unlock()
	if rej := a.upgradeJobCheck(); rej != nil {
		return nil, 0, rej
	}

	installed, rej, err := a.install(up.r, pl.Size, want)
	if rej != nil {
		return nil, 0, rej
	}
	if err != nil {
		return failed(err)
	}
	up.upgraded.Store(installed)
	return mustJSON(intent.AgentUpgradeResult{Upgraded: installed, From: buildinfo.Version(), To: pl.Version}), intent.StatusOK, nil
}

// upgradeJobCheck refuses an upgrade while a job may be running: one that is
// active, or a job store that did not open.
func (a *Agent) upgradeJobCheck() *Reject {
	if a.cfg.JobsUnavailable != nil {
		return reject(intent.ReasonJobActive, fmt.Sprintf("the job store could not be opened (%v), so a job may be running; fix the store and restart the gate", a.cfg.JobsUnavailable))
	}
	if j, busy, err := a.activeJob(); err != nil {
		return reject(intent.ReasonJobActive, err.Error())
	} else if busy {
		return reject(intent.ReasonJobActive, fmt.Sprintf("job %s (%s) is %s; upgrade once it ends", j.ID, j.Kind, j.State))
	}
	return nil
}

// install streams exactly size bytes of body to a temp file of its own in
// the binary's directory, checks them against want, checks again that no job
// started meanwhile, and renames that exact file over BinaryPath. Any
// failure removes the temp file and leaves the old binary. It reports false,
// with nothing changed, when the binary is the one already installed. The
// caller holds upgradeMu.
func (a *Agent) install(body io.Reader, size int64, want []byte) (bool, *Reject, error) {
	bin := a.cfg.BinaryPath
	// A .new from an older gate's upgrade is stale; only this gate's lock
	// holder ever writes beside the binary now.
	if err := os.Remove(bin + ".new"); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return false, nil, err
	}
	f, err := os.CreateTemp(filepath.Dir(bin), ".jumpgate-upgrade-*")
	if err != nil {
		return false, nil, err
	}
	staged := f.Name()
	closed, renamed := false, false
	defer func() {
		if !closed {
			f.Close()
		}
		if !renamed {
			os.Remove(staged)
		}
	}()
	if err := f.Chmod(0o700); err != nil {
		return false, nil, err
	}
	h := sha256.New()
	n, err := io.Copy(io.MultiWriter(f, h), io.LimitReader(body, size))
	if err != nil {
		return false, nil, fmt.Errorf("reading the binary: %w", err)
	}
	if n < size {
		return false, nil, fmt.Errorf("the binary is short: %d of %d bytes arrived", n, size)
	}
	if m, _ := io.ReadFull(body, make([]byte, 1)); m > 0 {
		return false, nil, fmt.Errorf("the binary is longer than its declared %d bytes", size)
	}
	got := h.Sum(nil)
	if hex.EncodeToString(got) != hex.EncodeToString(want) {
		return false, nil, fmt.Errorf("sha256 mismatch: got %x, want %x", got, want)
	}
	if cur, err := fileSHA256(bin); err == nil && hex.EncodeToString(cur) == hex.EncodeToString(got) {
		return false, nil, nil
	}
	if err := f.Sync(); err != nil {
		return false, nil, err
	}
	closed = true
	if err := f.Close(); err != nil {
		return false, nil, err
	}
	// The binary may have streamed for minutes: a job that started meanwhile
	// owns the box until it ends.
	if rej := a.upgradeJobCheck(); rej != nil {
		return false, rej, nil
	}
	if err := os.Chmod(staged, 0o755); err != nil {
		return false, nil, err
	}
	if err := os.Rename(staged, bin); err != nil {
		return false, nil, err
	}
	renamed = true
	// The new binary is in place; a failed directory sync only risks the
	// old one coming back after a power cut.
	_ = fsatomic.SyncDir(filepath.Dir(bin))
	return true, nil, nil
}

func fileSHA256(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return nil, err
	}
	return h.Sum(nil), nil
}

// serveUpgrade is POST /v1/upgrade: the envelope in UpgradeEnvelopeHeader,
// the binary in the body. Handle admits the envelope (signature, policy,
// replay) before the body is read; the restart follows the written receipt.
func serveUpgrade(a *Agent, w http.ResponseWriter, r *http.Request) {
	raw := r.Header.Get(UpgradeEnvelopeHeader)
	if raw == "" || len(raw) > maxUpgradeHeader {
		w.Header().Set("Connection", "close")
		http.Error(w, "the "+UpgradeEnvelopeHeader+" header is missing or over 64 KiB", http.StatusBadRequest)
		return
	}
	var env intent.Envelope
	if b, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(raw, "=")); err != nil || json.Unmarshal(b, &env) != nil {
		// An undecodable envelope still gets a signed rejection.
		env = intent.Envelope{}
	}
	ctx, up := withUpgradeBody(r.Context(), http.MaxBytesReader(w, r.Body, maxUpgrade+1))
	// The restart can stop this process as soon as it is asked for, so the
	// receipt goes out whole, with its length, before it: a chunked answer
	// would end only when this handler returns.
	b, err := json.Marshal(a.Handle(ctx, env))
	if err != nil {
		http.Error(w, "encode the receipt", http.StatusInternalServerError)
		return
	}
	b = append(b, '\n')
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(len(b)))
	_, _ = w.Write(b)
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
	up.afterReceipt(a)
}
