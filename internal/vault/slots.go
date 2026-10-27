package vault

import (
	"bytes"
	"crypto/rand"
	"crypto/subtle"
	"errors"
	"fmt"
)

// ErrSlotRefused means no slot of the asked kind unwrapped the data key.
var ErrSlotRefused = errors.New("vault: no key slot accepted that key")

// Slot is what a header says about one key slot. It holds no key material.
type Slot struct {
	Kind                    SlotKind
	Label, Cred, Store, Ref string
}

// Slots lists the key slots; it works while locked.
func (v *Vault) Slots() []Slot {
	v.mu.Lock()
	defer v.mu.Unlock()
	out := make([]Slot, 0, len(v.hdr.Slots))
	for _, s := range v.hdr.Slots {
		out = append(out, Slot{Kind: SlotKind(s.Kind), Label: s.Label, Cred: s.Cred, Store: s.Store, Ref: s.Ref})
	}
	return out
}

// UnlockKEK tries kek on every slot of kind, which must not be a passphrase
// slot. It clears kek.
func (v *Vault) UnlockKEK(kind SlotKind, kek []byte) error {
	defer clear(kek)
	if kind == SlotPassphrase {
		return errors.New("vault: a passphrase slot takes a passphrase, not a key")
	}
	if len(kek) != keyLen {
		return ErrSlotRefused
	}
	return v.unlockWith(func(s slotRec) []byte {
		if SlotKind(s.Kind) != kind {
			return nil
		}
		return bytes.Clone(kek)
	}, ErrSlotRefused)
}

// AddKEKSlot wraps the data key under kek (32 bytes) into a slot described
// by s, replacing a slot of the same Kind, Cred and Ref. It clears kek.
func (v *Vault) AddKEKSlot(s Slot, kek []byte) error {
	defer clear(kek)
	switch s.Kind {
	case SlotSystemdCreds, SlotKeystore:
	default:
		return fmt.Errorf("vault: slot kind %q cannot hold a raw key", s.Kind)
	}
	if len(kek) != keyLen {
		return fmt.Errorf("vault: a slot key must be %d bytes", keyLen)
	}
	return v.update(func(h *header, _ map[string]entryRec) error {
		r := slotRec{Kind: string(s.Kind), Label: s.Label, Cred: s.Cred, Store: s.Store, Ref: s.Ref}
		if err := wrapDEK(kek, v.dek, &r); err != nil {
			return err
		}
		replaceSlot(h, r, func(o slotRec) bool { return o.Kind == r.Kind && o.Cred == r.Cred && o.Ref == r.Ref })
		return nil
	})
}

func replaceSlot(h *header, r slotRec, same func(slotRec) bool) {
	for i, o := range h.Slots {
		if same(o) {
			h.Slots[i] = r
			return
		}
	}
	h.Slots = append(h.Slots, r)
}

// AddRecovery adds, or replaces, the "recovery" passphrase slot with a new
// 24-word code, which it returns. The code is a passphrase like any other.
func (v *Vault) AddRecovery() (string, error) {
	code, err := recoveryCode()
	if err != nil {
		return "", err
	}
	defer clear(code)
	err = v.update(func(h *header, _ map[string]entryRec) error {
		s, err := passSlot("recovery", code, v.dek, DefaultParams)
		if err != nil {
			return err
		}
		replaceSlot(h, s, func(o slotRec) bool { return SlotKind(o.Kind) == SlotPassphrase && o.Label == "recovery" })
		return nil
	})
	if err != nil {
		return "", err
	}
	return string(code), nil
}

// Rotating the data key.
//
// Changing the passphrase or removing a slot must not leave an old
// credential able to open the new file, so both draw a fresh data key,
// re-seal the body under it, wrap it into a new primary passphrase slot and
// clear the old key. Every other slot is dropped, because its key is not at
// hand:
//
//   - the recovery slot: its code is shown once and never stored, so it
//     cannot be re-wrapped; `vault add-recovery` makes a new one;
//   - systemd-creds and keystore slots: their keys live outside this
//     process; `vault seal` (or the keystore setup) adds them again.
//
// The dropped slots are returned so the caller can tell the operator.
//
// This protects the current file only. A copy or backup taken before the
// rotation is a separate file under the old data key and still opens with
// the credentials it was made with: old passphrase, old recovery code, old
// sealed key. Destroy those copies if the old credentials may be known.

// Rekdf is Rekey without the list of dropped slots. A caller that changes
// the passphrase should use Rekey and report what it dropped.
func (v *Vault) Rekdf(passphrase, newPassphrase []byte, p Params) error {
	_, err := v.Rekey(passphrase, newPassphrase, p)
	return err
}

// Rekey replaces the primary slot with one derived under p. With
// newPassphrase nil it re-derives from passphrase: only the argon2id
// parameters change, the data key and every other slot stay, and no slot is
// dropped. With a new passphrase it rotates the data key (see above) and
// returns the slots it dropped. It clears both passphrases.
func (v *Vault) Rekey(passphrase, newPassphrase []byte, p Params) ([]Slot, error) {
	defer clear(passphrase)
	defer clear(newPassphrase)
	next := newPassphrase
	if next == nil {
		next = passphrase
	}
	if len(next) == 0 {
		return nil, errors.New("vault: empty passphrase")
	}
	if _, err := p.kdf(); err != nil {
		return nil, err
	}
	if newPassphrase != nil {
		return v.rotate(passphrase, next, p)
	}
	return nil, v.update(func(h *header, _ map[string]entryRec) error {
		i, err := v.checkPrimary(h, passphrase)
		if err != nil {
			return err
		}
		s, err := passSlot("primary", next, v.dek, p)
		if err != nil {
			return err
		}
		h.Slots[i] = s
		return nil
	})
}

// RemoveSlots removes every slot of kind. Removing the passphrase slots is
// refused: the vault would then open only with a machine-held key. A
// removed slot's key would still unwrap the old data key, so it rotates the
// data key (see above), which takes the primary passphrase, and so drops
// every slot but the primary, not only those of kind. It returns what it
// dropped and clears passphrase. The primary keeps its argon2id parameters.
func (v *Vault) RemoveSlots(kind SlotKind, passphrase []byte) ([]Slot, error) {
	defer clear(passphrase)
	if kind == SlotPassphrase {
		return nil, errors.New("vault: refusing to remove the last passphrase slot")
	}
	v.mu.Lock()
	var p Params
	for _, s := range v.hdr.Slots {
		if SlotKind(s.Kind) == SlotPassphrase && s.Label == "primary" && s.KDF != nil {
			p = Params{Memory: s.KDF.M, Time: s.KDF.T, Threads: s.KDF.P}
		}
	}
	v.mu.Unlock()
	if _, err := p.kdf(); err != nil {
		p = DefaultParams
	}
	return v.rotate(passphrase, passphrase, p)
}

// rotate checks passphrase against the primary slot, draws a fresh data
// key, wraps it into a new primary slot from next under p, drops every
// other slot and re-seals the body. updateKey clears the old key.
func (v *Vault) rotate(passphrase, next []byte, p Params) ([]Slot, error) {
	dek := make([]byte, keyLen)
	if _, err := rand.Read(dek); err != nil {
		return nil, fmt.Errorf("vault: %w", err)
	}
	var dropped []Slot
	err := v.updateKey(dek, func(h *header, _ map[string]entryRec) error {
		dropped = nil
		i, err := v.checkPrimary(h, passphrase)
		if err != nil {
			return err
		}
		s, err := passSlot("primary", next, dek, p)
		if err != nil {
			return err
		}
		for j, o := range h.Slots {
			if j != i {
				dropped = append(dropped, Slot{Kind: SlotKind(o.Kind), Label: o.Label, Cred: o.Cred, Store: o.Store, Ref: o.Ref})
			}
		}
		h.Slots = []slotRec{s}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return dropped, nil
}

// checkPrimary returns the index of the primary passphrase slot in h after
// checking that passphrase unwraps the held data key from it.
func (v *Vault) checkPrimary(h *header, passphrase []byte) (int, error) {
	i := -1
	for j, s := range h.Slots {
		if SlotKind(s.Kind) == SlotPassphrase && s.Label == "primary" {
			i = j
		}
	}
	if i < 0 {
		return -1, errors.New("vault: no primary passphrase slot")
	}
	kek := deriveKEK(passphrase, *h.Slots[i].KDF)
	dek, err := unwrapDEK(kek, h.Slots[i])
	clear(kek)
	if err != nil {
		return -1, ErrWrongPassphrase
	}
	same := subtle.ConstantTimeCompare(dek, v.dek) == 1
	clear(dek)
	if !same {
		return -1, ErrWrongPassphrase
	}
	return i, nil
}
