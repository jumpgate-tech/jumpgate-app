package vault

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// rotationFixture is a vault with an entry, a recovery slot and two KEK
// slots, plus a copy of the file taken before any rotation.
func rotationFixture(t *testing.T) (v *Vault, p string, code string, kek []byte, old []byte) {
	t.Helper()
	v, p = newVault(t)
	if err := v.Put("ai-key", []byte("k"), ""); err != nil {
		t.Fatal(err)
	}
	code, err := v.AddRecovery()
	if err != nil {
		t.Fatal(err)
	}
	kek = bytes.Repeat([]byte{7}, 32)
	if err := v.AddKEKSlot(Slot{Kind: SlotSystemdCreds, Cred: "jumpgate-vault-key"}, bytes.Clone(kek)); err != nil {
		t.Fatal(err)
	}
	if err := v.AddKEKSlot(Slot{Kind: SlotKeystore, Store: "keychain", Ref: "jumpgate-vault"}, bytes.Clone(kek)); err != nil {
		t.Fatal(err)
	}
	old, _ = os.ReadFile(p)
	return v, p, code, kek, old
}

// oldSlotsCannotOpen checks that no slot of the pre-rotation copy, under its
// credential, unwraps a key that opens the current body, and that the copy
// itself still opens with the old passphrase (the documented limit).
func oldSlotsCannotOpen(t *testing.T, p string, old []byte, code string, kek []byte) {
	t.Helper()
	ho, _, _, err := parse(old)
	if err != nil {
		t.Fatal(err)
	}
	cur, _ := os.ReadFile(p)
	h, line, ct, err := parse(cur)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range ho.Slots {
		var k []byte
		switch s.Label {
		case "primary":
			k = deriveKEK(testPass, *s.KDF)
		case "recovery":
			k = deriveKEK([]byte(code), *s.KDF)
		default:
			k = bytes.Clone(kek)
		}
		d, err := unwrapDEK(k, s)
		if err != nil {
			t.Fatalf("old %s slot does not unwrap the old copy: %v", s.Kind, err)
		}
		if _, err := openBody(line, ct, d, h.Nonce); err == nil {
			t.Fatalf("old %s/%s slot opens the new body", s.Kind, s.Label)
		}
	}
	q := filepath.Join(t.TempDir(), "backup")
	if err := os.WriteFile(q, old, 0o600); err != nil {
		t.Fatal(err)
	}
	b, err := Open(q)
	if err != nil {
		t.Fatal(err)
	}
	if err := b.Unlock(pass()); err != nil {
		t.Fatalf("old backup: %v", err)
	}
}

func droppedSet(d []Slot) map[string]bool {
	m := map[string]bool{}
	for _, s := range d {
		m[string(s.Kind)+"/"+s.Label] = true
	}
	return m
}

func TestPassphraseChangeRotatesTheDEKAndDropsOtherSlots(t *testing.T) {
	v, p, code, kek, old := rotationFixture(t)
	v.mu.Lock()
	before := v.dek
	dek := bytes.Clone(v.dek)
	v.mu.Unlock()
	dropped, err := v.Rekey(pass(), []byte("a new passphrase"), DefaultParams)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, make([]byte, len(before))) {
		t.Fatal("old DEK not cleared")
	}
	v.mu.Lock()
	changed := !bytes.Equal(dek, v.dek)
	v.mu.Unlock()
	if !changed {
		t.Fatal("the DEK did not change")
	}
	got := droppedSet(dropped)
	if len(dropped) != 3 || !got["passphrase/recovery"] || !got["systemd-creds/"] || !got["keystore/"] {
		t.Fatalf("dropped = %+v", dropped)
	}
	if s := v.Slots(); len(s) != 1 || s[0].Label != "primary" {
		t.Fatalf("slots = %+v", s)
	}
	oldSlotsCannotOpen(t, p, old, code, kek)
	w, err := Open(p)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.Unlock([]byte(code)); !errors.Is(err, ErrWrongPassphrase) {
		t.Fatalf("old recovery code: %v", err)
	}
	if err := w.UnlockKEK(SlotSystemdCreds, bytes.Clone(kek)); !errors.Is(err, ErrSlotRefused) {
		t.Fatalf("old sealed key: %v", err)
	}
	if err := w.Unlock([]byte("a new passphrase")); err != nil {
		t.Fatal(err)
	}
	if string(mustGet(t, w, "ai-key")) != "k" {
		t.Fatal("entry lost")
	}
}

func TestRekdfParamsOnlyKeepsTheSlots(t *testing.T) {
	v, _, _, _, _ := rotationFixture(t)
	dropped, err := v.Rekey(pass(), nil, DefaultParams)
	if err != nil || len(dropped) != 0 {
		t.Fatalf("dropped=%v err=%v", dropped, err)
	}
	if n := len(v.Slots()); n != 4 {
		t.Fatalf("%d slots", n)
	}
}

func TestRemoveSlotsRotatesTheDEK(t *testing.T) {
	v, p, code, kek, old := rotationFixture(t)
	if _, err := v.RemoveSlots(SlotKeystore, []byte("wrong")); !errors.Is(err, ErrWrongPassphrase) {
		t.Fatalf("wrong passphrase: %v", err)
	}
	if cur, _ := os.ReadFile(p); !bytes.Equal(cur, old) {
		t.Fatal("a refused removal changed the file")
	}
	dropped, err := v.RemoveSlots(SlotKeystore, pass())
	if err != nil {
		t.Fatal(err)
	}
	if len(dropped) != 3 {
		t.Fatalf("dropped = %+v", dropped)
	}
	oldSlotsCannotOpen(t, p, old, code, kek)
	w := openUnlocked(t, p)
	if string(mustGet(t, w, "ai-key")) != "k" {
		t.Fatal("entry lost")
	}
}
