package filelock

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestTryLockIsExclusive(t *testing.T) {
	p := filepath.Join(t.TempDir(), "x.lock")
	h, err := TryLock(p)
	if err != nil {
		t.Fatalf("first TryLock: %v", err)
	}
	if _, err := TryLock(p); !errors.Is(err, ErrLocked) {
		t.Fatalf("second TryLock = %v, want ErrLocked", err)
	}
	if err := h.Unlock(); err != nil {
		t.Fatal(err)
	}
	h2, err := TryLock(p)
	if err != nil {
		t.Fatalf("TryLock after Unlock: %v", err)
	}
	h2.Unlock()
}
