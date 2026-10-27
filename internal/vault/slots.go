package vault

import (
	"bytes"
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

// RemoveSlots removes every slot of kind. Removing the passphrase slots is
// refused: the vault would then open only with a machine-held key.
func (v *Vault) RemoveSlots(kind SlotKind) error {
	if kind == SlotPassphrase {
		return errors.New("vault: refusing to remove the last passphrase slot")
	}
	return v.update(func(h *header, _ map[string]entryRec) error {
		kept := h.Slots[:0:0]
		for _, s := range h.Slots {
			if SlotKind(s.Kind) != kind {
				kept = append(kept, s)
			}
		}
		h.Slots = kept
		return nil
	})
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

// Rekdf replaces the primary slot with one derived under p, from
// newPassphrase, or from passphrase again when newPassphrase is nil. The
// data key and the entries stay; the body is re-sealed because the header is
// its associated data. It clears both passphrases.
func (v *Vault) Rekdf(passphrase, newPassphrase []byte, p Params) error {
	defer clear(passphrase)
	defer clear(newPassphrase)
	next := newPassphrase
	if next == nil {
		next = passphrase
	}
	if len(next) == 0 {
		return errors.New("vault: empty passphrase")
	}
	if _, err := p.kdf(); err != nil {
		return err
	}
	return v.update(func(h *header, _ map[string]entryRec) error {
		i := -1
		for j, s := range h.Slots {
			if SlotKind(s.Kind) == SlotPassphrase && s.Label == "primary" {
				i = j
			}
		}
		if i < 0 {
			return errors.New("vault: no primary passphrase slot")
		}
		kek := deriveKEK(passphrase, *h.Slots[i].KDF)
		dek, err := unwrapDEK(kek, h.Slots[i])
		clear(kek)
		if err != nil {
			return ErrWrongPassphrase
		}
		same := subtle.ConstantTimeCompare(dek, v.dek) == 1
		clear(dek)
		if !same {
			return ErrWrongPassphrase
		}
		s, err := passSlot("primary", next, v.dek, p)
		if err != nil {
			return err
		}
		h.Slots[i] = s
		return nil
	})
}
