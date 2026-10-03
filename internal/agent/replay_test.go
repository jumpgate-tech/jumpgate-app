package agent

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/valve-tech/jumpgate/internal/intent"
)

var t0 = time.Unix(1_800_000_000, 0)

func nonce(b byte) [32]byte { var n [32]byte; n[0] = b; return n }

func newReplay(t *testing.T) (*Replay, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "replay.json")
	if err := InitReplay(path); err != nil {
		t.Fatal(err)
	}
	r, err := OpenReplay(path)
	if err != nil {
		t.Fatal(err)
	}
	return r, path
}

func TestAdmitEnforcesIncreasingSeqAndPersistsIt(t *testing.T) {
	r, path := newReplay(t)
	exp := uint64(t0.Unix() + 120)
	if err := r.Admit(addr(1), 5, nonce(1), exp, t0); err != nil {
		t.Fatal(err)
	}
	var rej *Reject
	if err := r.Admit(addr(1), 5, nonce(2), exp, t0); !errors.As(err, &rej) || rej.Code != intent.ReasonStaleSeq || rej.LastSeq != 5 {
		t.Fatalf("repeat seq: err = %v", err)
	}
	// A fresh process reading the same file must see the persisted seq.
	again, err := OpenReplay(path)
	if err != nil {
		t.Fatal(err)
	}
	if again.LastSeq(addr(1)) != 5 {
		t.Fatalf("persisted LastSeq = %d, want 5", again.LastSeq(addr(1)))
	}
}

func TestAdmitRejectsAReplayedNonceWithinItsWindow(t *testing.T) {
	r, _ := newReplay(t)
	exp := uint64(t0.Unix() + 120)
	_ = r.Admit(addr(1), 1, nonce(7), exp, t0)
	var rej *Reject
	if err := r.Admit(addr(1), 2, nonce(7), exp, t0); !errors.As(err, &rej) || rej.Code != intent.ReasonReplayedNonce {
		t.Fatalf("err = %v, want replayed_nonce", err)
	}
	// Once the window has passed, the nonce is forgotten (seq still protects).
	if err := r.Admit(addr(1), 3, nonce(7), uint64(t0.Unix()+1000), t0.Add(10*time.Minute)); err != nil {
		t.Fatalf("after the window: %v", err)
	}
}

// Review Focus 2: each controller has its own sequence.
func TestControllersAreIndependent(t *testing.T) {
	r, _ := newReplay(t)
	exp := uint64(t0.Unix() + 120)
	if err := r.Admit(addr(1), 50, nonce(1), exp, t0); err != nil {
		t.Fatal(err)
	}
	if err := r.Admit(addr(2), 1, nonce(2), exp, t0); err != nil {
		t.Fatalf("second controller blocked by the first's sequence: %v", err)
	}
}

// Review Focus 1: a corrupt or missing state file fails closed. Starting from
// empty would let any captured intent be replayed.
func TestOpenReplayFailsClosed(t *testing.T) {
	dir := t.TempDir()
	corrupt := filepath.Join(dir, "replay.json")
	if err := os.WriteFile(corrupt, []byte(`{"controllers":`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenReplay(corrupt); !errors.Is(err, ErrReplayState) {
		t.Fatalf("corrupt file: err = %v, want ErrReplayState", err)
	}
	if _, err := OpenReplay(filepath.Join(dir, "missing.json")); !errors.Is(err, ErrReplayState) {
		t.Fatalf("missing file: err = %v, want ErrReplayState", err)
	}
	if err := InitReplay(corrupt); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenReplay(corrupt); !errors.Is(err, ErrReplayState) {
		t.Fatal("InitReplay overwrote an existing (corrupt) file; it must only create")
	}
}
