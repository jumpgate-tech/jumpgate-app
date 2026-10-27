package vault

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
)

const bip39EnglishSHA256 = "2f5eed53a4727b4bf8880d8f3f199efc90e58503646d9ff8eff3a2ed3b24dbda"

func TestWordListIsBIP39English(t *testing.T) {
	sum := sha256.Sum256(wordlistEN)
	if hex.EncodeToString(sum[:]) != bip39EnglishSHA256 {
		t.Fatalf("sha256 = %x", sum)
	}
	w := words()
	if len(w) != 2048 {
		t.Fatalf("%d words", len(w))
	}
	seen := map[string]bool{}
	for _, s := range w {
		if seen[s] || s == "" {
			t.Fatalf("duplicate or empty word %q", s)
		}
		seen[s] = true
	}
}

func TestRecoveryCodeUnlocks(t *testing.T) {
	v, _ := newVault(t)
	if err := v.Put("ai-key", []byte("k"), ""); err != nil {
		t.Fatal(err)
	}
	code, err := v.AddRecovery()
	if err != nil {
		t.Fatal(err)
	}
	ws := strings.Split(code, " ")
	if len(ws) != 24 {
		t.Fatalf("%d words", len(ws))
	}
	in := map[string]bool{}
	for _, w := range words() {
		in[w] = true
	}
	for _, w := range ws {
		if !in[w] {
			t.Fatalf("%q is not in the list", w)
		}
	}
	v.Lock()
	if err := v.Unlock([]byte(code)); err != nil {
		t.Fatal(err)
	}
	if string(mustGet(t, v, "ai-key")) != "k" {
		t.Fatal("entry lost")
	}
	// A second recovery code replaces the first.
	code2, err := v.AddRecovery()
	if err != nil {
		t.Fatal(err)
	}
	if n := countSlots(v, SlotPassphrase); n != 2 {
		t.Fatalf("%d passphrase slots, want 2", n)
	}
	v.Lock()
	if err := v.Unlock([]byte(code)); !errors.Is(err, ErrWrongPassphrase) {
		t.Fatalf("old code: %v", err)
	}
	if err := v.Unlock([]byte(code2)); err != nil {
		t.Fatal(err)
	}
}

func countSlots(v *Vault, k SlotKind) int {
	n := 0
	for _, s := range v.Slots() {
		if s.Kind == k {
			n++
		}
	}
	return n
}

func TestRekdfKeepsTheDEKAndDropsTheOldSlot(t *testing.T) {
	v, p := newVault(t)
	if err := v.PutAll([]Item{{Name: "ai-key", Value: []byte("k")}, {Name: "vpn/a", Value: []byte("c")}}); err != nil {
		t.Fatal(err)
	}
	old, _ := os.ReadFile(p)
	v.mu.Lock()
	dek := bytes.Clone(v.dek)
	v.mu.Unlock()
	if err := v.Rekdf(pass(), nil, Params{Memory: 131072, Time: 4, Threads: 4}); err != nil {
		t.Fatal(err)
	}
	v.mu.Lock()
	same := bytes.Equal(dek, v.dek)
	v.mu.Unlock()
	if !same {
		t.Fatal("the DEK changed")
	}
	w := openUnlocked(t, p)
	if string(mustGet(t, w, "ai-key")) != "k" || string(mustGet(t, w, "vpn/a")) != "c" {
		t.Fatal("entries changed")
	}
	if k := w.hdr.Slots[0].KDF; k.M != 131072 || k.T != 4 {
		t.Fatalf("kdf = %+v", k)
	}
	// Splice the old primary slot into the new header: the slot unwraps the
	// same DEK, but the header line no longer matches the body.
	ho, _, _, err := parse(old)
	if err != nil {
		t.Fatal(err)
	}
	cur, _ := os.ReadFile(p)
	spliced := rewriteHeader(t, cur, func(h *header) { h.Slots[0] = ho.Slots[0] })
	h, line, ct, err := parse(spliced)
	if err != nil {
		t.Fatal(err)
	}
	d, err := unwrapDEK(deriveKEK(testPass, *h.Slots[0].KDF), h.Slots[0])
	if err != nil {
		t.Fatal(err)
	}
	if _, err := openBody(line, ct, d, h.Nonce); !errors.Is(err, ErrCorrupt) {
		t.Fatalf("spliced = %v, want ErrCorrupt", err)
	}
}

func TestRekdfRefusesAWrongPassphrase(t *testing.T) {
	v, p := newVault(t)
	before, _ := os.ReadFile(p)
	if err := v.Rekdf([]byte("nope"), []byte("new"), DefaultParams); !errors.Is(err, ErrWrongPassphrase) {
		t.Fatalf("err = %v", err)
	}
	after, _ := os.ReadFile(p)
	if !bytes.Equal(before, after) {
		t.Fatal("the file changed")
	}
}

func TestRekdfCanChangeThePassphrase(t *testing.T) {
	v, p := newVault(t)
	np := []byte("a new passphrase")
	if err := v.Rekdf(pass(), np, DefaultParams); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(np, make([]byte, len(np))) {
		t.Fatal("new passphrase not cleared")
	}
	w, err := Open(p)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.Unlock(pass()); !errors.Is(err, ErrWrongPassphrase) {
		t.Fatalf("old passphrase: %v", err)
	}
	if err := w.Unlock([]byte("a new passphrase")); err != nil {
		t.Fatal(err)
	}
}

func TestKEKSlotUnlocks(t *testing.T) {
	v, p := newVault(t)
	if err := v.Put("ai-key", []byte("k"), ""); err != nil {
		t.Fatal(err)
	}
	kek := bytes.Repeat([]byte{9}, 32)
	in := bytes.Clone(kek)
	s := Slot{Kind: SlotSystemdCreds, Cred: "jumpgate-vault-key"}
	if err := v.AddKEKSlot(s, in); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(in, make([]byte, 32)) {
		t.Fatal("kek not cleared")
	}
	// The same slot again replaces it, never adds one.
	if err := v.AddKEKSlot(s, bytes.Clone(kek)); err != nil {
		t.Fatal(err)
	}
	if n := countSlots(v, SlotSystemdCreds); n != 1 {
		t.Fatalf("%d systemd slots", n)
	}
	w, err := Open(p)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.UnlockKEK(SlotSystemdCreds, bytes.Repeat([]byte{8}, 32)); !errors.Is(err, ErrSlotRefused) {
		t.Fatalf("another kek: %v", err)
	}
	if err := w.UnlockKEK(SlotKeystore, bytes.Clone(kek)); !errors.Is(err, ErrSlotRefused) {
		t.Fatalf("another kind: %v", err)
	}
	if !w.Locked() {
		t.Fatal("unlocked by a refused kek")
	}
	if err := w.UnlockKEK(SlotSystemdCreds, bytes.Clone(kek)); err != nil {
		t.Fatal(err)
	}
	if string(mustGet(t, w, "ai-key")) != "k" {
		t.Fatal("entry lost")
	}
	if err := w.UnlockKEK(SlotPassphrase, bytes.Clone(kek)); err == nil {
		t.Fatal("a KEK unlocked a passphrase slot")
	}
	if err := v.AddKEKSlot(Slot{Kind: SlotPassphrase}, bytes.Clone(kek)); err == nil {
		t.Fatal("a raw KEK became a passphrase slot")
	}
	if err := v.AddKEKSlot(s, []byte("short")); err == nil {
		t.Fatal("a short KEK was accepted")
	}
	if err := v.RemoveSlots(SlotSystemdCreds); err != nil {
		t.Fatal(err)
	}
	if n := countSlots(v, SlotSystemdCreds); n != 0 {
		t.Fatal("slot not removed")
	}
	v.Lock()
	if err := v.UnlockKEK(SlotSystemdCreds, bytes.Clone(kek)); !errors.Is(err, ErrSlotRefused) {
		t.Fatalf("removed slot: %v", err)
	}
}

func TestLastPassphraseSlotCannotBeRemoved(t *testing.T) {
	v, _ := newVault(t)
	if err := v.RemoveSlots(SlotPassphrase); err == nil {
		t.Fatal("removed every passphrase slot")
	}
	if n := countSlots(v, SlotPassphrase); n != 1 {
		t.Fatalf("%d passphrase slots", n)
	}
}

func TestSlotsAreReadableWhileLocked(t *testing.T) {
	v, p := newVault(t)
	if err := v.AddKEKSlot(Slot{Kind: SlotKeystore, Store: "keychain", Ref: "jumpgate-vault"}, bytes.Repeat([]byte{1}, 32)); err != nil {
		t.Fatal(err)
	}
	w, err := Open(p)
	if err != nil {
		t.Fatal(err)
	}
	got := w.Slots()
	want := []Slot{{Kind: SlotPassphrase, Label: "primary"}, {Kind: SlotKeystore, Store: "keychain", Ref: "jumpgate-vault"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Slots = %+v", got)
	}
	for _, f := range reflect.VisibleFields(reflect.TypeFor[Slot]()) {
		if f.Name == "Nonce" || f.Name == "Wrapped" {
			t.Fatalf("Slot exposes %s", f.Name)
		}
	}
	b, _ := json.Marshal(got)
	if strings.Contains(string(b), "wrapped") {
		t.Fatal("key material in Slot")
	}
}

// TestConcurrentSlotChanges runs rotations against entry writes from a
// second Vault on the same file; neither may lose the other's change.
func TestConcurrentSlotChanges(t *testing.T) {
	a, p := newVault(t)
	b := openUnlocked(t, p)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		if err := a.AddKEKSlot(Slot{Kind: SlotSystemdCreds, Cred: "c"}, bytes.Repeat([]byte{3}, 32)); err != nil {
			t.Error(err)
		}
		if _, err := a.AddRecovery(); err != nil {
			t.Error(err)
		}
	}()
	go func() {
		defer wg.Done()
		for i := range 5 {
			if err := b.Put("e"+string(rune('a'+i)), []byte("x"), ""); err != nil {
				t.Error(err)
			}
		}
	}()
	wg.Wait()
	c := openUnlocked(t, p)
	n, _ := c.Names()
	if len(n) != 5 || len(c.Slots()) != 3 {
		t.Fatalf("names %v slots %+v", n, c.Slots())
	}
}
