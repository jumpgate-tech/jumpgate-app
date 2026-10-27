package server

import (
	"errors"
	"testing"

	"github.com/valve-tech/jumpgate/internal/vault"
	"github.com/valve-tech/jumpgate/internal/vault/vaulttest"
)

// stubCoreDumps replaces disableCoreDumps for the test; it counts calls and
// records whether the vault was still locked at the last call.
func stubCoreDumps(t *testing.T, v *vault.Vault, ret error) (calls *int, lockedAtCall *bool) {
	t.Helper()
	calls, lockedAtCall = new(int), new(bool)
	old := disableCoreDumps
	disableCoreDumps = func() error {
		*calls++
		*lockedAtCall = v.Locked()
		return ret
	}
	t.Cleanup(func() { disableCoreDumps = old })
	return calls, lockedAtCall
}

func TestUnlockDisablesCoreDumpsFirst(t *testing.T) {
	r := newVaultRig(t)
	calls, locked := stubCoreDumps(t, r.v, nil)
	r.unlock(t)
	if *calls != 1 || !*locked {
		t.Fatalf("passphrase unlock: calls=%d lockedAtCall=%v, want 1 call while still locked", *calls, *locked)
	}
	r.lock()
	*calls, *locked = 0, false
	// The KEK path: no such slot, so it fails, but only after the call.
	if _, err := r.srv.UnlockKEK(vault.SlotSystemdCreds, make([]byte, 32)); err == nil {
		t.Fatal("UnlockKEK with no slot succeeded")
	}
	if *calls != 1 || !*locked {
		t.Fatalf("KEK unlock: calls=%d lockedAtCall=%v, want 1 call while still locked", *calls, *locked)
	}
}

func TestUnlockRefusedWhenCoreDumpsStayOn(t *testing.T) {
	r := newVaultRig(t)
	boom := errors.New("prctl failed")
	stubCoreDumps(t, r.v, boom)
	if _, err := r.srv.Unlock([]byte(vaulttest.Passphrase)); !errors.Is(err, boom) {
		t.Fatalf("Unlock err = %v, want %v", err, boom)
	}
	if _, err := r.srv.UnlockKEK(vault.SlotSystemdCreds, make([]byte, 32)); !errors.Is(err, boom) {
		t.Fatalf("UnlockKEK err = %v, want %v", err, boom)
	}
	if !r.v.Locked() || !r.srv.keyLocked() || r.unlocks.Load() != 0 {
		t.Fatalf("after refused unlock: vaultLocked=%v keyLocked=%v unlocks=%d", r.v.Locked(), r.srv.keyLocked(), r.unlocks.Load())
	}
	if st := r.srv.VaultState().State; st != "locked" {
		t.Fatalf("VaultState = %q, want locked", st)
	}
}
